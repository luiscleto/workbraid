package architecture

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func reconciliationFixture(t *testing.T) (*Manager, Snapshot, diagramFixtureIDs) {
	t.Helper()
	ctx := t.Context()
	m := NewManager(t.TempDir())
	empty, err := m.CreateProject(ctx, "Semantics")
	if err != nil {
		t.Fatal(err)
	}
	ids := diagramFixtureIDs{root: uuid.NewString(), detail: uuid.NewString(), empty: uuid.NewString(), gateway: uuid.NewString(), worker: uuid.NewString(), records: uuid.NewString(), ledger: uuid.NewString()}
	path := mustStorePath(t, m, empty.StoreID())
	revision := commitV2Fixture(t, path, empty.StoreID(), ids)
	gitText(t, "--git-dir", path, "update-ref", acceptedRef, revision, empty.Revision())
	base, err := m.LoadAccepted(ctx, empty.StoreID())
	if err != nil {
		t.Fatal(err)
	}
	return m, base, ids
}
func reconciliationAccepted(t *testing.T, m *Manager, b Snapshot, changes []ComponentChange, composition CandidateComposition) Snapshot {
	t.Helper()
	ctx := t.Context()
	candidate, err := m.prepareTestCandidate(ctx, b, changes, composition)
	if err != nil {
		t.Fatal(err)
	}
	revision, err := m.CreateSuccessor(ctx, b, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.AdvanceAccepted(ctx, b, revision); err != nil {
		t.Fatal(err)
	}
	a, err := m.LoadAccepted(ctx, b.StoreID())
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func reconciliationProposed(t *testing.T, m *Manager, b Snapshot, changes []ComponentChange, composition CandidateComposition) Candidate {
	t.Helper()
	p, err := m.prepareTestCandidate(t.Context(), b, changes, composition)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestReconciliationAllScalarConflictsExactCountsAndManualValues(t *testing.T) {
	m, b, ids := reconciliationFixture(t)
	aChange, _ := b.ChangeForAcceptedComponent(ids.worker)
	pChange := aChange
	aChange.Title, aChange.TitleChanged = "Accepted *Worker*", true
	pChange.Title, pChange.TitleChanged = "Proposed Worker", true
	aChange.Description, aChange.DescriptionChanged = " \nAccepted\r\n", true
	pChange.Description, pChange.DescriptionChanged = "\nProposed\n", true
	// The base has two exact multiline occurrences. Independent changes to
	// three and one conflict; other differently labelled occurrences survive.
	aChange.Relationships = append(append([]AuthoringRelationship{}, aChange.Relationships...), AuthoringRelationship{TargetID: ids.records, Label: "calls\nnext"})
	aChange.RelationshipsChanged = true
	pChange.Relationships = append([]AuthoringRelationship{}, pChange.Relationships[1:]...)
	pChange.RelationshipsChanged = true
	a := reconciliationAccepted(t, m, b, []ComponentChange{aChange}, CandidateComposition{DiagramTitles: []DiagramTitleChange{{ids.detail, "Accepted detail"}}})
	p := reconciliationProposed(t, m, b, []ComponentChange{pChange}, CandidateComposition{DiagramTitles: []DiagramTitleChange{{ids.detail, "Proposed detail"}}})
	preview, err := m.Reconcile(t.Context(), b, a, p, nil)
	if err != nil || len(preview.Conflicts) != 4 || preview.Status != "needs_resolution" {
		t.Fatalf("all conflicts: %+v %v", preview, err)
	}
	choices := []ReconciliationResolution{}
	for _, conflict := range preview.Conflicts {
		r := ReconciliationResolution{Locator: conflict.Locator, Choice: "manual", Value: &ReconciliationValue{}}
		switch conflict.Locator.Kind {
		case "component_title":
			value := "Exact *Worker* & <API>"
			r.Value.Text = &value
		case "component_description":
			value := " \t\r\n\n"
			r.Value.Text = &value
		case "diagram_title":
			value := "Final\nDiagram"
			r.Value.Text = &value
		case "relationship_count":
			count := 4
			r.Value.Count = &count
		}
		choices = append(choices, r)
	}
	resolved, err := m.Reconcile(t.Context(), b, a, p, choices)
	if err != nil || resolved.Status != "ready" {
		t.Fatalf("manual choices: %+v %v", resolved, err)
	}
	got, _ := resolved.Candidate.Snapshot().ChangeForAcceptedComponent(ids.worker)
	if got.Title != "Exact *Worker* & <API>" || got.Description != " \t\r\n\n" {
		t.Fatalf("exact values lost: %+v", got)
	}
	want := []AuthoringRelationship{{ids.records, "calls\nnext"}, {ids.records, "calls\nnext"}, {ids.ledger, "uses"}, {ids.gateway, "returns"}, {ids.records, "calls\nnext"}, {ids.records, "calls\nnext"}}
	if !reflect.DeepEqual(got.Relationships, want) {
		t.Fatalf("Accepted order/tuple multiplicity: %+v", got.Relationships)
	}
	if _, err := m.Reconcile(t.Context(), b, a, p, append(choices, choices[0])); err == nil {
		t.Fatal("duplicate locator accepted")
	}
}

func TestReconciliationSameNewIdentityRequiresCompleteSemanticContext(t *testing.T) {
	for _, kind := range []string{"component equal", "component body", "component home", "component incoming", "component reference", "component child", "diagram equal", "diagram title", "diagram appearances", "diagram anchor"} {
		t.Run(kind, func(t *testing.T) {
			m, b, ids := reconciliationFixture(t)
			created := m.NewComponentChange(b, nil, "Added", "Exact\n")
			detail := b.NewDetailDiagramChange(nil, "Added detail", ids.ledger)
			ac, pc := []ComponentChange{}, []ComponentChange{}
			av, pv := CandidateComposition{}, CandidateComposition{}
			if strings.HasPrefix(kind, "component") {
				ac, pc = []ComponentChange{created}, []ComponentChange{created}
				av.NewComponentHomes = []NewComponentHome{{created.ID, ids.root}}
				pv.NewComponentHomes = []NewComponentHome{{created.ID, ids.root}}
				switch kind {
				case "component body":
					ac[0].Description = "Divergent\n"
				case "component home":
					av.NewComponentHomes[0].DiagramID = ids.detail
				case "component incoming":
					other, _ := b.ChangeForAcceptedComponent(ids.gateway)
					other.RelationshipsChanged = true
					other.Relationships = append(other.Relationships, AuthoringRelationship{created.ID, "calls"})
					ac = append(ac, other)
				case "component reference":
					av.References = []ReferenceAppearanceChange{{DiagramID: ids.detail, ComponentID: created.ID, Present: true}}
				case "component child":
					child := b.NewDetailDiagramChange(nil, "Dependent child", created.ID)
					av.DetailDiagrams = []DetailDiagramChange{child}
				}
			} else {
				av.DetailDiagrams = []DetailDiagramChange{detail}
				pv.DetailDiagrams = []DetailDiagramChange{detail}
				if kind == "diagram title" {
					av.DetailDiagrams[0].Title = "Divergent"
				}
				if kind == "diagram anchor" {
					av.DetailDiagrams[0].AnchorComponentID = ids.worker
				}
				if kind == "diagram appearances" {
					av.References = []ReferenceAppearanceChange{{DiagramID: detail.ID, ComponentID: ids.gateway, Present: true}}
				}
			}
			a := reconciliationAccepted(t, m, b, ac, av)
			p := reconciliationProposed(t, m, b, pc, pv)
			path := mustStorePath(t, m, b.StoreID())
			refs := gitText(t, "--git-dir", path, "show-ref")
			result, err := m.Reconcile(t.Context(), b, a, p, nil)
			if err != nil {
				t.Fatal(err)
			}
			if strings.HasSuffix(kind, "equal") {
				if result.Status != "ready" || len(result.Changes) != 0 || len(result.Composition.DetailDiagrams) != 0 {
					t.Fatalf("identical object did not coalesce: %+v", result)
				}
				aTree := gitText(t, "--git-dir", path, "rev-parse", a.Revision()+"^{tree}")
				if result.Candidate.Tree() != aTree {
					t.Fatal("empty residual changed A tree")
				}
			} else {
				if result.Status != "blocked" {
					t.Fatalf("divergent identity not blocked: %+v", result)
				}
				var collision *ReconciliationConflict
				for i := range result.Conflicts {
					if result.Conflicts[i].Unsupported["reason"] == "replace_identity" {
						collision = &result.Conflicts[i]
					}
				}
				if collision == nil || len(collision.Choices) != 0 {
					t.Fatal("collision offered replacement choices")
				}
				_, err = m.Reconcile(t.Context(), b, a, p, []ReconciliationResolution{{Locator: collision.Locator, Choice: "accepted"}})
				var domain *ReconciliationError
				if !errors.As(err, &domain) || domain.Code != "reconciliation_unsupported" || domain.Reason != "replace_identity" {
					t.Fatalf("replacement choice: %v", err)
				}
			}
			if gitText(t, "--git-dir", path, "show-ref") != refs {
				t.Fatal("preview mutated refs")
			}
		})
	}
}

func TestReconciliationDistinctPathCollisionsAndExactNewSource(t *testing.T) {
	m, b, ids := reconciliationFixture(t)
	left := m.NewComponentChange(b, nil, "Same title", "\r\nBody\r\n")
	right := m.NewComponentChange(b, nil, "Same title", "\r\nBody\r\n")
	a := reconciliationAccepted(t, m, b, []ComponentChange{left}, CandidateComposition{NewComponentHomes: []NewComponentHome{{left.ID, ids.root}}})
	p := reconciliationProposed(t, m, b, []ComponentChange{right}, CandidateComposition{NewComponentHomes: []NewComponentHome{{right.ID, ids.detail}}})
	result, err := m.Reconcile(t.Context(), b, a, p, nil)
	if err != nil || result.Status != "ready" {
		t.Fatalf("distinct additions: %+v %v", result, err)
	}
	if len(result.Changes) != 1 || result.Changes[0].ID != right.ID || result.Changes[0].Path == left.Path || !strings.Contains(result.Changes[0].Path, right.ID) {
		t.Fatalf("path allocation: %+v", result.Changes)
	}
	path := mustStorePath(t, m, b.StoreID())
	got := gitBytes(t, "--git-dir", path, "show", result.Candidate.Tree()+":"+result.Changes[0].Path)
	want := gitBytes(t, "--git-dir", path, "show", p.Tree()+":"+right.Path)
	if !reflect.DeepEqual(got, want) {
		t.Fatal("path-only allocation changed new source")
	}
	again, err := m.Reconcile(t.Context(), b, a, p, nil)
	if err != nil || again.Candidate.Tree() != result.Candidate.Tree() {
		t.Fatal("allocation not deterministic")
	}
}

func TestReconciliationCompositionOverlapAndCrossChangeCycle(t *testing.T) {
	for _, scenario := range []string{"overlap", "cycle"} {
		t.Run(scenario, func(t *testing.T) {
			m, b, ids := reconciliationFixture(t)
			av, pv := CandidateComposition{}, CandidateComposition{}
			if scenario == "overlap" {
				av.HomeMoves = []ComponentHomeMove{{ids.ledger, ids.empty}}
				pv.References = []ReferenceAppearanceChange{{DiagramID: ids.empty, ComponentID: ids.ledger, Present: true}}
			} else {
				av.HomeMoves = []ComponentHomeMove{{ids.gateway, ids.empty}}
				pv.HomeMoves = []ComponentHomeMove{{ids.records, ids.detail}}
			}
			a := reconciliationAccepted(t, m, b, nil, av)
			p := reconciliationProposed(t, m, b, nil, pv)
			preview, err := m.Reconcile(t.Context(), b, a, p, nil)
			if err != nil || preview.Status != "needs_resolution" || len(preview.Conflicts) != 1 {
				t.Fatalf("structure: %+v %v", preview, err)
			}
			conflict := preview.Conflicts[0]
			wantReason := "home_reference_overlap"
			if scenario == "cycle" {
				wantReason = "hierarchy_cycle"
			}
			if conflict.Locator.Reason != wantReason {
				t.Fatalf("reason=%s", conflict.Locator.Reason)
			}
			value := conflict.Accepted.ReconciliationValue
			result, err := m.Reconcile(t.Context(), b, a, p, []ReconciliationResolution{{Locator: conflict.Locator, Choice: "accepted", Value: &value}})
			if err != nil || result.Status != "ready" {
				t.Fatalf("chosen structure: %+v %v", result, err)
			}
		})
	}
}

func TestReconciliationExternalAbsenceKeepsSupportedAcceptedChoice(t *testing.T) {
	m, b, ids := reconciliationFixture(t)
	path := mustStorePath(t, m, b.StoreID())
	// External A removes a base-existing empty child. Its parent link disappears
	// in the same valid external snapshot, without an authoring deletion fact.
	components, diagrams := map[string]string{}, map[string]string{}
	for _, c := range b.components {
		components[filepath.Base(c.path)] = string(c.source)
	}
	for _, d := range b.diagrams {
		if d.id.String() == ids.empty {
			continue
		}
		d.appearances = append([]diagramAppearance{}, d.appearances...)
		for i := range d.appearances {
			if d.appearances[i].hasDetailLink && d.appearances[i].detailDiagram.String() == ids.empty {
				d.appearances[i].hasDetailLink = false
				d.appearances[i].detailDiagram = uuid.Nil
			}
		}
		source, err := marshalDiagram(d)
		if err != nil {
			t.Fatal(err)
		}
		diagrams[filepath.Base(d.path)] = string(source)
	}
	manifest := string(gitBytes(t, "--git-dir", path, "show", b.Revision()+":architecture.yaml"))
	revision := commitV2Sources(t, path, manifest, components, diagrams, nil)
	gitText(t, "--git-dir", path, "update-ref", acceptedRef, revision, b.Revision())
	a, err := m.LoadAccepted(context.Background(), b.StoreID())
	if err != nil {
		t.Fatal(err)
	}
	p := reconciliationProposed(t, m, b, nil, CandidateComposition{DiagramTitles: []DiagramTitleChange{{ids.empty, "Proposed edited child"}}})
	preview, err := m.Reconcile(t.Context(), b, a, p, nil)
	if err != nil || len(preview.Conflicts) != 1 {
		t.Fatalf("deletion conflict: %+v %v", preview, err)
	}
	conflict := preview.Conflicts[0]
	if conflict.Locator.Kind != "diagram_object" || conflict.Accepted.Exists || conflict.Proposed.Diagram.Title != "Proposed edited child" {
		t.Fatalf("object context: %+v", conflict)
	}
	kept, err := m.Reconcile(t.Context(), b, a, p, []ReconciliationResolution{{Locator: conflict.Locator, Choice: "accepted"}})
	if err != nil || kept.Status != "ready" || kept.Candidate.Snapshot().HasDiagram(ids.empty) {
		t.Fatalf("accepted absence: %+v %v", kept, err)
	}
	_, err = m.Reconcile(t.Context(), b, a, p, []ReconciliationResolution{{Locator: conflict.Locator, Choice: "proposed"}})
	var domain *ReconciliationError
	if !errors.As(err, &domain) || domain.Reason != "restore_diagram" {
		t.Fatalf("restore: %v", err)
	}
}

func TestReconciliationResolutionJSONIsClosed(t *testing.T) {
	id := uuid.NewString()
	valid := `{"locator":{"kind":"component_description","component_id":"` + id + `"},"choice":"manual","value":{"text":"\r\nExact\n"}}`
	var r ReconciliationResolution
	if err := json.Unmarshal([]byte(valid), &r); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []string{strings.Replace(valid, `"choice":"manual"`, `"choice":"manual","choice":"accepted"`, 1), strings.Replace(valid, `"text":"\r\nExact\n"`, `"text":"body","count":1`, 1), strings.Replace(valid, `"component_id":`, `"extra":false,"component_id":`, 1), strings.Replace(valid, `"text":"\r\nExact\n"`, `"text":null`, 1), strings.Replace(valid, `"manual"`, `"accepted"`, 1)} {
		if err := json.Unmarshal([]byte(invalid), &r); err == nil {
			t.Fatalf("accepted invalid union: %s", invalid)
		}
	}
}
