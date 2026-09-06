package architecture

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func placementFixture(t *testing.T) (*Manager, Snapshot, diagramFixtureIDs, string) {
	t.Helper()
	ctx := context.Background()
	m := NewManager(t.TempDir())
	initial, err := m.CreateProject(ctx, "Placement")
	if err != nil {
		t.Fatal(err)
	}
	if initial.FormatVersion() != 5 {
		t.Fatal("native bootstrap is not v5")
	}
	path, _ := m.StorePath(initial.StoreID())
	ids := diagramFixtureIDs{root: uuid.NewString(), detail: uuid.NewString(), empty: uuid.NewString(), gateway: uuid.NewString(), worker: uuid.NewString(), records: uuid.NewString(), ledger: uuid.NewString()}
	commit := commitV2Fixture(t, path, initial.StoreID(), ids)
	gitText(t, "--git-dir", path, "update-ref", acceptedRef, commit)
	base, err := m.LoadAccepted(ctx, initial.StoreID())
	if err != nil {
		t.Fatal(err)
	}
	return m, base, ids, path
}

func TestPlacementClosedPortableSchema(t *testing.T) {
	id, c := uuid.NewString(), uuid.NewString()
	base := fmt.Sprintf("id: %s\ntitle: Diagram\nappearances:\n  - component: %s\n    role: home\n", id, c)
	for _, pair := range []string{"x: 0\n    y: 0", "x: -100000\n    y: 100000", "x: 10\n    y: -20"} {
		source := base + "positions:\n  - component: " + c + "\n    " + pair + "\n"
		if _, err := parseDiagram("diagrams/a.yaml", []byte(source), 3); err != nil {
			t.Fatal(err)
		}
		if _, err := parseDiagram("diagrams/a.yaml", []byte(source), 2); err == nil {
			t.Fatal("v2 admitted positions")
		}
	}
	for _, suffix := range []string{"", "positions: []\n"} {
		if _, err := parseDiagram("diagrams/a.yaml", []byte(base+suffix), 3); err != nil {
			t.Fatal(err)
		}
	}
	bad := []string{"null", "{}", "\n  - component: " + c + "\n    x: 1\n    y: 0\n    extra: 1"}
	for _, coordinate := range []string{"null", "true", "'1'", "1.0", "1.5", "100001", "-100001", "999999999999999999999999999999"} {
		bad = append(bad, "\n  - component: "+c+"\n    x: "+coordinate+"\n    y: 0")
	}
	bad = append(bad, "\n  - component: "+c+"\n    x: 1\n    x: 2\n    y: 0", "\n  - component: "+c+"\n    x: 1\n    y: 0\n  - component: "+strings.ToUpper(c)+"\n    x: 2\n    y: 3")
	for i, suffix := range bad {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			if _, err := parseDiagram("diagrams/a.yaml", []byte(base+"positions: "+suffix+"\n"), 3); err == nil {
				t.Fatalf("accepted invalid positions %s", suffix)
			}
		})
	}
	if _, err := parseDiagram("diagrams/a.yaml", []byte(base+"positions: []\n"), 2); err == nil {
		t.Fatal("v2 empty positions")
	}
}

func TestPlacementStickyVersionAndExactFiles(t *testing.T) {
	m, base, ids, path := placementFixture(t)
	ctx := context.Background()
	unchanged, err := m.ConstructCandidate(ctx, base, nil, CandidateComposition{})
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Tree() != gitText(t, "--git-dir", path, "rev-parse", base.Revision()+"^{tree}") {
		t.Fatal("no-op changed tree")
	}
	composition := SetNodePosition(base, CandidateComposition{}, ids.root, ids.worker, &Position{0, -180})
	candidate, err := m.PrepareCandidate(ctx, base, nil, &composition)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"components/worker.md", "components/gateway.md", "diagrams/empty.yaml"} {
		if gitText(t, "--git-dir", path, "rev-parse", base.Revision()+":"+name) != gitText(t, "--git-dir", path, "rev-parse", candidate.Tree()+":"+name) {
			t.Fatalf("unrelated blob changed: %s", name)
		}
	}
	replay, err := m.ConstructCandidate(ctx, base, nil, composition)
	if err != nil {
		t.Fatal(err)
	}
	if replay.Tree() != candidate.Tree() || replay.Snapshot().FormatVersion() != 3 {
		t.Fatal("complete v3 facts did not replay exactly")
	}
	for _, d := range candidate.Snapshot().diagrams {
		if err := validatePositions(d, candidate.Snapshot().components); err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range base.DiagramProjections() {
		for _, a := range d.Appearances {
			if d.ID == ids.root && a.ComponentID == ids.worker {
				continue
			}
			p, _ := candidate.Snapshot().NodePosition(d.ID, a.ComponentID)
			if !samePosition(p, a.DisplayPosition) {
				t.Fatal("first placement moved a peer")
			}
		}
		for _, b := range d.Boundaries {
			p, _ := candidate.Snapshot().NodePosition(d.ID, b.ComponentID)
			if !samePosition(p, b.DisplayPosition) {
				t.Fatal("first placement moved boundary peer")
			}
		}
	}
	id, name, _ := m.NewChangeSet(nil, "Sticky target")
	record := ChangeSet{ID: id, Name: name, Lifecycle: "active", BaseSnapshot: base, BaseRevision: base.Revision(), Composition: composition, Candidate: &candidate, Generation: 2}
	if _, err = m.WriteActiveChangeSet(ctx, base.StoreID(), record, ""); err != nil {
		t.Fatal(err)
	}
	loaded, bad, err := m.LoadChangeSets(ctx, base.StoreID())
	if err != nil || len(bad) > 0 || len(loaded) != 1 || loaded[0].Candidate.Tree() != candidate.Tree() {
		t.Fatalf("sticky reload %+v %v", bad, err)
	}
	raw := gitBytes(t, "--git-dir", path, "show", loaded[0].RefObject+":changes.yaml")
	if !strings.Contains(string(raw), "architecture_version: 3") {
		t.Fatal("target not persisted")
	}
}

func TestPlacementOperationalNullAndClosedValues(t *testing.T) {
	composition := CandidateComposition{ArchitectureVersion: 3, NodePositions: []NodePositionChange{{uuid.NewString(), uuid.NewString(), nil}, {uuid.NewString(), uuid.NewString(), &Position{}}}}
	raw, err := marshalChangeState(nil, composition)
	if err != nil {
		t.Fatal(err)
	}
	_, got, err := parseChangeState(raw)
	if err != nil || !reflect.DeepEqual(got.NodePositions, composition.NodePositions) {
		t.Fatalf("null and zero: %+v %v\n%s", got, err, raw)
	}
	for _, value := range []string{"{}", "{\"position\":{\"x\":0}}", "{\"position\":{\"x\":null,\"y\":0}}", "{\"position\":{\"x\":0.5,\"y\":0}}", "{\"position\":{\"x\":0,\"y\":0,\"extra\":1}}", "{\"position\":{\"x\":100001,\"y\":0}}"} {
		source := fmt.Sprintf(`{"locator":{"kind":"node_position","diagram_id":%q,"component_id":%q},"choice":"manual","value":%s}`, composition.NodePositions[0].DiagramID, composition.NodePositions[0].ComponentID, value)
		var resolution ReconciliationResolution
		if json.Unmarshal([]byte(source), &resolution) == nil {
			t.Fatalf("accepted %s", source)
		}
	}
}

func TestPlacementAppearanceAwareFiveCases(t *testing.T) {
	cases := []struct {
		name, b, a, p string
		keep          bool
		want          *Position
		conflict      bool
	}{
		{"accepted removes", "pin", "absent", "move", true, &Position{300, 300}, false},
		{"proposed removes", "pin", "move", "absent", true, &Position{300, 300}, false},
		{"new identical", "absent", "move", "move", true, &Position{300, 300}, false},
		{"new divergent manual", "absent", "pin", "move", true, nil, true},
		{"final absent", "pin", "pin", "move", false, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, base, ids, _ := placementFixture(t)
			ctx := context.Background()
			changes := []ComponentChange{}
			for _, c := range base.AuthoringComponents() {
				v, _ := base.ChangeForAcceptedComponent(c.ID)
				v.Relationships = nil
				v.RelationshipsChanged = true
				changes = append(changes, v)
			}
			isolated, err := m.ConstructCandidate(ctx, base, changes, CandidateComposition{})
			if err != nil {
				t.Fatal(err)
			}
			base = isolated.Snapshot()
			k := reconciliationPair{ids.root, ids.worker}
			makeSide := func(state string) Snapshot {
				composition := CandidateComposition{ArchitectureVersion: 3}
				if state == "absent" {
					composition.References = []ReferenceAppearanceChange{{ids.root, ids.worker, false}}
				}
				if state == "pin" || state == "move" {
					p := Position{100, 100}
					if state == "move" {
						p = Position{300, 300}
					}
					composition.NodePositions = []NodePositionChange{{ids.root, ids.worker, &p}}
				}
				candidate, err := m.prepareTestCandidate(ctx, base, nil, composition)
				if err != nil {
					t.Fatal(err)
				}
				return candidate.Snapshot()
			}
			b, a, p := makeSide(tc.b), makeSide(tc.a), makeSide(tc.p)
			final := makeSide("automatic")
			if !tc.keep {
				final = makeSide("absent")
			}
			calc := reconciliationCalculation{b: snapshotReconciliationFacts(b), a: snapshotReconciliationFacts(a), p: snapshotReconciliationFacts(p), final: snapshotReconciliationFacts(final), resolutions: map[string]ReconciliationResolution{}, used: map[string]bool{}}
			calc.mergePositions()
			if calc.err != nil {
				t.Fatal(calc.err)
			}
			if (len(calc.result.Conflicts) > 0) != tc.conflict {
				t.Fatalf("conflicts %+v", calc.result.Conflicts)
			}
			got, ok := calc.final.positions[k]
			if tc.want != nil && (!ok || got != *tc.want) || tc.want == nil && ok {
				t.Fatalf("position %v %v", got, ok)
			}
			if tc.conflict {
				if calc.result.Conflicts[0].Original.State != "not_applicable" {
					t.Fatal("invented original Automatic")
				}
				return
			}
			changes, composition, err := reconciliationResidual(a, p, calc.b, calc.final)
			if err != nil {
				t.Fatal(err)
			}
			candidate, err := m.ConstructCandidate(ctx, a, changes, composition)
			if err != nil {
				t.Fatal(err)
			}
			if pos, present := candidate.Snapshot().NodePosition(ids.root, ids.worker); present != tc.keep || !samePosition(pos, tc.want) {
				t.Fatalf("residual %v %v", pos, present)
			}
		})
	}
}

func TestPlacementMixedVersionResiduals(t *testing.T) {
	for _, versions := range [][3]int{{2, 2, 2}, {2, 2, 3}, {2, 3, 2}, {2, 3, 3}, {3, 3, 3}} {
		t.Run(fmt.Sprint(versions), func(t *testing.T) {
			m, base, ids, path := placementFixture(t)
			ctx := context.Background()
			construct := func(b Snapshot, version int, pins []NodePositionChange) Candidate {
				t.Helper()
				candidate, err := m.prepareTestCandidate(ctx, b, nil, CandidateComposition{ArchitectureVersion: version, NodePositions: pins})
				if err != nil {
					t.Fatal(err)
				}
				return candidate
			}
			commit := func(b Snapshot, candidate Candidate) Snapshot {
				t.Helper()
				revision, err := m.CreateSuccessor(ctx, b, candidate)
				if err != nil {
					t.Fatal(err)
				}
				snapshot, err := m.LoadRevision(ctx, b, revision)
				if err != nil {
					t.Fatal(err)
				}
				return snapshot
			}
			base = commit(base, construct(base, versions[0], nil))
			var aPins, pPins []NodePositionChange
			if versions[1] == 3 {
				aPins = []NodePositionChange{{ids.root, ids.gateway, &Position{-50, 0}}}
			}
			if versions[2] == 3 {
				pPins = []NodePositionChange{{ids.root, ids.worker, &Position{0, 80}}}
			}
			accepted := commit(base, construct(base, versions[1], aPins))
			proposed := construct(base, versions[2], pPins)
			refs := gitText(t, "--git-dir", path, "show-ref")
			result, err := m.Reconcile(ctx, base, accepted, proposed, nil)
			if err != nil || result.Status != "ready" {
				t.Fatalf("%+v %v", result, err)
			}
			if result.Composition.ArchitectureVersion != max(versions[1], versions[2]) {
				t.Fatal("target downgrade")
			}
			for _, pin := range append(aPins, pPins...) {
				p, present := result.Candidate.Snapshot().NodePosition(pin.DiagramID, pin.ComponentID)
				if !present || !samePosition(p, pin.Position) {
					t.Fatal("lost independent pin")
				}
			}
			if gitText(t, "--git-dir", path, "show-ref") != refs {
				t.Fatal("preview mutated refs")
			}
			id, name, _ := m.NewChangeSet(nil, "Mixed version residual")
			record := ChangeSet{ID: id, Name: name, Lifecycle: "active", BaseRevision: accepted.Revision(), BaseSnapshot: accepted, Generation: 2, Composition: result.Composition, Candidate: result.Candidate}
			object, err := m.WriteActiveChangeSet(ctx, base.StoreID(), record, "")
			if err != nil {
				t.Fatal(err)
			}
			loaded, bad, err := NewManager(filepath.Dir(m.storeRoot)).LoadChangeSets(ctx, base.StoreID())
			if err != nil || len(bad) != 0 || len(loaded) != 1 || loaded[0].RefObject != object || loaded[0].Candidate.Tree() != result.Candidate.Tree() {
				t.Fatalf("restart %v %v", bad, err)
			}
			// A sticky but cleared v3 P is a format-only residual on v2 A,
			// and exactly empty when A is already v3.
			cleared := construct(base, max(versions[0], 3), nil)
			clearResult, err := m.Reconcile(ctx, base, accepted, cleared, nil)
			if err != nil || clearResult.Status != "ready" {
				t.Fatalf("clear %v %v", clearResult, err)
			}
			if clearResult.Candidate.Snapshot().FormatVersion() != 3 || versions[1] == 3 && len(clearResult.Composition.NodePositions) != 0 {
				t.Fatal("cleared target lost sticky version or A pins")
			}
			aTree := construct(accepted, versions[1], nil).Tree()
			if (clearResult.Candidate.Tree() == aTree) != (versions[1] == 3) {
				t.Fatal("format-only versus empty residual")
			}
		})
	}
}

func TestPlacementWholePairConflictChoices(t *testing.T) {
	for _, choice := range []string{"accepted", "proposed", "manual", "invalid null"} {
		t.Run(choice, func(t *testing.T) {
			m, base, ids, _ := placementFixture(t)
			ctx := context.Background()
			b, err := m.prepareTestCandidate(ctx, base, nil, SetNodePosition(base, CandidateComposition{}, ids.root, ids.worker, &Position{10, 20}))
			if err != nil {
				t.Fatal(err)
			}
			rev, _ := m.CreateSuccessor(ctx, base, b)
			base, err = m.LoadRevision(ctx, base, rev)
			if err != nil {
				t.Fatal(err)
			}
			ac, _ := m.ConstructCandidate(ctx, base, nil, SetNodePosition(base, CandidateComposition{}, ids.root, ids.worker, &Position{80, 90}))
			ar, _ := m.CreateSuccessor(ctx, base, ac)
			a, err := m.LoadRevision(ctx, base, ar)
			if err != nil {
				t.Fatal(err)
			}
			p, _ := m.ConstructCandidate(ctx, base, nil, SetNodePosition(base, CandidateComposition{}, ids.root, ids.worker, &Position{}))
			preview, err := m.Reconcile(ctx, base, a, p, nil)
			if err != nil || preview.Status != "needs_resolution" || len(preview.Conflicts) != 1 {
				t.Fatalf("%+v %v", preview, err)
			}
			conflict := preview.Conflicts[0]
			if conflict.Original.State != "stored" || conflict.Accepted.State != "stored" || conflict.Proposed.State != "stored" {
				t.Fatal("stored position context lost")
			}
			resolution := ReconciliationResolution{Locator: conflict.Locator, Choice: choice}
			want := &Position{80, 90}
			if choice == "proposed" {
				want = &Position{}
			}
			if choice == "manual" {
				want = &Position{-180, 420}
				resolution.Value = &ReconciliationValue{Position: want}
			}
			if choice == "invalid null" {
				resolution.Choice = "manual"
				resolution.Value = &ReconciliationValue{}
			}
			result, err := m.Reconcile(ctx, base, a, p, []ReconciliationResolution{resolution})
			if choice == "invalid null" {
				if err == nil {
					t.Fatal("manual null accepted")
				}
				return
			}
			if err != nil || result.Status != "ready" {
				t.Fatalf("%+v %v", result, err)
			}
			got, _ := result.Candidate.Snapshot().NodePosition(ids.root, ids.worker)
			if !samePosition(got, want) {
				t.Fatalf("whole pair %+v, want %+v", got, want)
			}
		})
	}
}
