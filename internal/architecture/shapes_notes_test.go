package architecture

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestShapesNotesLegacyUpgradeAndExactReplay(t *testing.T) {
	for _, version := range []int{2, 3, 4, 5} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			m, base, ids, _ := placementFixture(t)
			if version > 2 {
				base = reconciliationAccepted(t, m, base, nil, CandidateComposition{ArchitectureVersion: version})
			}
			before := base.DiagramProjections()
			text := " \r\n<script src='https://invalid.test'>x</script> **plain** λ🙂\n "
			note, e := base.NewDiagramNote(ids.root, text)
			if e != nil {
				t.Fatal(e)
			}
			shape := "diamond"
			c := SetNodeShape(UpgradePresentation(base, CandidateComposition{}), ids.root, ids.worker, &shape)
			c = SetDiagramNote(base, c, ids.root, note.ID, &note.NoteValue)
			candidate, e := m.PrepareCandidate(t.Context(), base, nil, &c)
			if e != nil {
				t.Fatal(e)
			}
			if candidate.Snapshot().FormatVersion() != 6 {
				t.Fatal("missing upgrade")
			}
			replay, e := m.ConstructCandidate(t.Context(), base, nil, c)
			if e != nil || replay.Tree() != candidate.Tree() {
				t.Fatalf("replay: %v", e)
			}
			for _, d := range before {
				for _, a := range d.Appearances {
					pos, _ := candidate.Snapshot().NodePosition(d.ID, a.ComponentID)
					size, _ := candidate.Snapshot().NodeSize(d.ID, a.ComponentID)
					if !samePosition(pos, a.DisplayPosition) || !sameSize(size, a.DisplaySize) {
						t.Fatal("upgrade changed peer geometry")
					}
				}
			}
			ns, _ := candidate.Snapshot().DiagramNotes(ids.root)
			if len(ns) != 1 || ns[0].NoteValue != note.NoteValue {
				t.Fatal("note source changed")
			}
			raw, e := marshalChangeState(nil, c)
			if e != nil {
				t.Fatal(e)
			}
			changes, loaded, e := parseChangeState(raw)
			if e != nil {
				t.Fatal(e)
			}
			again, e := m.ConstructCandidate(t.Context(), base, changes, loaded)
			if e != nil || again.Tree() != candidate.Tree() {
				t.Fatal("operational roundtrip mismatch", e)
			}
			c = SetDiagramNote(base, c, ids.root, note.ID, nil)
			if len(c.DiagramNotes) != 0 {
				t.Fatal("new-note deletion retained fact")
			}
		})
	}
}

func TestWholeNoteReconciliationRestorationAndCollision(t *testing.T) {
	for _, direction := range []string{"accepted_deleted", "proposed_deleted", "collision", "independent"} {
		t.Run(direction, func(t *testing.T) {
			m, b, ids, _ := placementFixture(t)
			n, e := b.NewDiagramNote(ids.root, "Original\r\nλ")
			if e != nil {
				t.Fatal(e)
			}
			if direction != "collision" && direction != "independent" {
				b = reconciliationAccepted(t, m, b, nil, SetDiagramNote(b, UpgradePresentation(b, CandidateComposition{}), ids.root, n.ID, &n.NoteValue))
			}
			edited := n.NoteValue
			edited.Text = " Kept edit\n🙂 "
			edited.Width = 321
			ac := UpgradePresentation(b, CandidateComposition{})
			pc := UpgradePresentation(b, CandidateComposition{})
			switch direction {
			case "accepted_deleted":
				ac = SetDiagramNote(b, ac, ids.root, n.ID, nil)
				pc = SetDiagramNote(b, pc, ids.root, n.ID, &edited)
			case "proposed_deleted":
				ac = SetDiagramNote(b, ac, ids.root, n.ID, &edited)
				pc = SetDiagramNote(b, pc, ids.root, n.ID, nil)
			case "collision":
				ac = SetDiagramNote(b, ac, ids.root, n.ID, &n.NoteValue)
				pc = SetDiagramNote(b, pc, ids.root, n.ID, &edited)
			case "independent":
				other, _ := b.NewDiagramNote(ids.root, "Independent")
				ac = SetDiagramNote(b, ac, ids.root, n.ID, &n.NoteValue)
				pc = SetDiagramNote(b, pc, ids.root, other.ID, &other.NoteValue)
			}
			a := reconciliationAccepted(t, m, b, nil, ac)
			p := reconciliationProposed(t, m, b, nil, pc)
			r, e := m.Reconcile(t.Context(), b, a, p, nil)
			if e != nil {
				t.Fatal(e)
			}
			if direction == "collision" {
				if r.Status != "blocked" || len(r.Conflicts) != 1 || r.Conflicts[0].Unsupported["reason"] != "replace_identity" {
					t.Fatalf("collision %+v", r)
				}
				return
			}
			if direction == "independent" {
				if r.Status != "ready" || len(r.Candidate.Snapshot().diagrams[0].notes) > 2 {
					t.Fatalf("independent %+v", r)
				}
				return
			}
			if r.Status != "needs_resolution" || len(r.Conflicts) != 1 || r.Conflicts[0].Locator.Kind != "diagram_note" {
				t.Fatalf("whole-note conflict %+v", r)
			}
			for _, choice := range []string{"accepted", "proposed", "manual"} {
				resolution := ReconciliationResolution{Locator: r.Conflicts[0].Locator, Choice: choice}
				if choice == "manual" {
					resolution.Value = &ReconciliationValue{Note: &edited}
				}
				raw, _ := json.Marshal(resolution)
				var decoded ReconciliationResolution
				if e = json.Unmarshal(raw, &decoded); e != nil {
					t.Fatal(e)
				}
				resolved, e := m.Reconcile(t.Context(), b, a, p, []ReconciliationResolution{decoded})
				if e != nil || resolved.Status != "ready" {
					t.Fatalf("%s %v %+v", choice, e, resolved)
				}
				replay, e := m.ConstructCandidate(t.Context(), a, resolved.Changes, resolved.Composition)
				if e != nil || replay.Tree() != resolved.Candidate.Tree() {
					t.Fatal("residual replay", e)
				}
				notes, _ := replay.Snapshot().DiagramNotes(ids.root)
				keep := choice == "manual" || choice == "proposed" && direction == "accepted_deleted" || choice == "accepted" && direction == "proposed_deleted"
				if keep && (len(notes) != 1 || notes[0].ID != n.ID || notes[0].NoteValue != edited) || !keep && len(notes) != 0 {
					t.Fatalf("wrong chosen note %+v", notes)
				}
			}
		})
	}
}

func TestShapeNoteInvalidFactAddressesAndVisibility(t *testing.T) {
	m, b, ids, _ := placementFixture(t)
	shape := "ellipse"
	seed := UpgradePresentation(b, CandidateComposition{References: []ReferenceAppearanceChange{{DiagramID: ids.empty, ComponentID: ids.worker, Present: true}}})
	seed = SetNodeShape(seed, ids.empty, ids.worker, &shape)
	b = reconciliationAccepted(t, m, b, nil, seed)
	invalid, _ := b.ChangeForAcceptedComponent(ids.worker)
	invalid.TitleChanged = true
	invalid.Title = ""
	for _, c := range []CandidateComposition{
		{ArchitectureVersion: 6, NodeShapes: []NodeShapeChange{{ids.root, uuid.NewString(), nil}}},
		{ArchitectureVersion: 6, References: []ReferenceAppearanceChange{{DiagramID: ids.empty, ComponentID: ids.worker, Present: false}}, NodeShapes: []NodeShapeChange{{ids.empty, ids.ledger, &shape}}},
		{ArchitectureVersion: 6, DiagramNotes: []DiagramNoteChange{{uuid.NewString(), uuid.NewString(), nil}}},
		{ArchitectureVersion: 6, DiagramNotes: []DiagramNoteChange{{uuid.NewString(), uuid.NewString(), &NoteValue{Text: "note", Width: 240, Height: 120}}}},
	} {
		id, name, _ := m.NewChangeSet(nil, "Invalid addresses")
		record := ChangeSet{ID: id, Name: name, Lifecycle: "active", BaseRevision: b.Revision(), BaseSnapshot: b, Generation: 1, Changes: []ComponentChange{invalid}, Composition: c}
		if _, e := m.WriteActiveChangeSet(t.Context(), b.StoreID(), record, ""); e == nil {
			t.Fatal("unsupported fact hidden by invalid title")
		}
		if _, e := m.PrepareCandidate(t.Context(), b, nil, &c); e == nil {
			t.Fatal("prepare normalized unsupported fact")
		}
	}
	c := CandidateComposition{ArchitectureVersion: 6, References: []ReferenceAppearanceChange{{DiagramID: ids.empty, ComponentID: ids.worker, Present: false}}}
	c = ResetLostShapes(b, nil, []ComponentChange{invalid}, CandidateComposition{}, c)
	if len(c.NodeShapes) != 1 || c.NodeShapes[0].Shape != nil {
		t.Fatal("invalid visibility loss not retained")
	}
	c.References[0].Present = true
	c = ResetLostShapes(b, []ComponentChange{invalid}, nil, c, c)
	p, e := m.PrepareCandidate(t.Context(), b, nil, &c)
	if e != nil {
		t.Fatal(e)
	}
	v, _ := p.Snapshot().NodeShape(ids.empty, ids.worker)
	if v != nil {
		t.Fatal("shape resurrected")
	}
	unrelated := ResetLostShapes(b, nil, []ComponentChange{invalid}, CandidateComposition{}, CandidateComposition{ArchitectureVersion: 6})
	if len(unrelated.NodeShapes) != 0 {
		t.Fatal("unrelated invalid title cleared shape")
	}
}

func TestShapeNoteClosedFieldsAndOrdering(t *testing.T) {
	m, b, ids, _ := placementFixture(t)
	shape := "diamond"
	n, _ := b.NewDiagramNote(ids.root, "exact\ntext")
	c := SetDiagramNote(b, SetNodeShape(UpgradePresentation(b, CandidateComposition{}), ids.root, ids.worker, &shape), ids.root, n.ID, &n.NoteValue)
	p, e := m.PrepareCandidate(t.Context(), b, nil, &c)
	if e != nil {
		t.Fatal(e)
	}
	b = acceptCandidate(t, m, b, p)
	n2, _ := b.NewDiagramNote(ids.root, "second")
	edited := n.NoteValue
	edited.Text = " replacement \r\n"
	next := SetDiagramNote(b, CandidateComposition{ArchitectureVersion: 6}, ids.root, n.ID, &edited)
	next = SetDiagramNote(b, next, ids.root, n2.ID, &n2.NoteValue)
	p, e = m.PrepareCandidate(t.Context(), b, nil, &next)
	if e != nil {
		t.Fatal(e)
	}
	notes, _ := p.Snapshot().DiagramNotes(ids.root)
	if len(notes) != 2 || notes[0].ID != n.ID || notes[1].ID != n2.ID || notes[0].Text != edited.Text {
		t.Fatal("surviving source order changed")
	}
	raw, _ := marshalChangeState(nil, next)
	for _, bad := range []string{strings.Replace(string(raw), "diagram_notes:", "unexpected:", 1), strings.Replace(string(raw), "width: 240", "width: null", 1), strings.Replace(string(raw), "height: 120", "height: 601", 1), strings.Replace(string(raw), "note_id:", "id:", 1)} {
		if _, _, e := parseChangeState([]byte(bad)); e == nil {
			t.Fatal("invalid operational note admitted")
		}
	}
	before := p.Snapshot().DiagramProjections()
	layout, e := p.Snapshot().AutoLayout(ids.root)
	if e != nil {
		t.Fatal(e)
	}
	facts := next
	facts.NodePositions = layout
	laid, e := m.PrepareCandidate(t.Context(), b, nil, &facts)
	if e != nil {
		t.Fatal(e)
	}
	after, _ := laid.Snapshot().DiagramNotes(ids.root)
	if !reflect.DeepEqual(notes, after) {
		t.Fatal("layout moved notes")
	}
	_ = before
}
