package architecture

import (
	"context"
	"github.com/google/uuid"
	"reflect"
	"testing"
)

func TestPlacementValidatorRequiresExactVisibleCoverage(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	components := []component{{id: a, relationships: []componentRelationship{{target: b, label: "one"}, {target: b, label: "two"}}}, {id: b, relationships: []componentRelationship{{target: c, label: "external only"}}}, {id: c}}
	d := diagram{id: uuid.New(), appearances: []diagramAppearance{{component: a, role: "home"}}, positions: []diagramPosition{{a, Position{}}, {b, Position{320, 0}}}}
	if err := validatePositions(d, components); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name      string
		positions []diagramPosition
	}{
		{"missing boundary", d.positions[:1]},
		{"duplicate", append(append([]diagramPosition{}, d.positions...), d.positions[0])},
		{"external-only extra", append(append([]diagramPosition{}, d.positions...), diagramPosition{c, Position{640, 0}})},
		{"missing all", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			copy := d
			copy.positions = tc.positions
			if validatePositions(copy, components) == nil {
				t.Fatal("invalid complete coverage admitted")
			}
		})
	}
}

func TestPlacementVisibilityTransitionsAndNonResurrection(t *testing.T) {
	m, v2, ids, _ := placementFixture(t)
	ctx := context.Background()
	seed := CandidateComposition{ArchitectureVersion: 3, NodePositions: []NodePositionChange{{ids.root, ids.worker, &Position{7654, -100000}}, {ids.detail, ids.worker, &Position{-900, 700}}}}
	initial, err := m.PrepareCandidate(ctx, v2, nil, &seed)
	if err != nil {
		t.Fatal(err)
	}
	base := initial.Snapshot()
	comp := CandidateComposition{References: []ReferenceAppearanceChange{{ids.root, ids.worker, false}}}
	boundary, err := m.PrepareCandidate(ctx, base, nil, &comp)
	if err != nil {
		t.Fatal(err)
	}
	requirePosition := func(s Snapshot, d, c string, want Position) {
		t.Helper()
		p, visible := s.NodePosition(d, c)
		if !visible || p == nil || *p != want {
			t.Fatalf("position %s/%s: %v visible=%v want=%v", d, c, p, visible, want)
		}
	}
	requirePosition(boundary.Snapshot(), ids.root, ids.worker, Position{7654, -100000})
	comp.References[0].Present = true
	ordinary, err := m.PrepareCandidate(ctx, base, nil, &comp)
	if err != nil {
		t.Fatal(err)
	}
	if ordinary.Tree() != initial.Tree() {
		t.Fatal("boundary/reference round trip changed stored tree")
	}
	comp.HomeMoves = []ComponentHomeMove{{ids.worker, ids.root}}
	comp.References = nil
	moved, err := m.PrepareCandidate(ctx, base, nil, &comp)
	if err != nil {
		t.Fatal(err)
	}
	requirePosition(moved.Snapshot(), ids.root, ids.worker, Position{7654, -100000})
	requirePosition(moved.Snapshot(), ids.detail, ids.worker, Position{-900, 700}) // retained as boundary through Ledger
	comp = CandidateComposition{References: []ReferenceAppearanceChange{{ids.root, ids.worker, false}}}
	changes := []ComponentChange{}
	for _, c := range base.AuthoringComponents() {
		v, _ := base.ChangeForAcceptedComponent(c.ID)
		v.Relationships = nil
		v.RelationshipsChanged = true
		changes = append(changes, v)
	}
	absent, err := m.PrepareCandidate(ctx, base, changes, &comp)
	if err != nil {
		t.Fatal(err)
	}
	if _, visible := absent.Snapshot().NodePosition(ids.root, ids.worker); visible {
		t.Fatal("removed pair remains visible")
	}
	retainedNull := false
	for _, v := range comp.NodePositions {
		if v.DiagramID == ids.root && v.ComponentID == ids.worker && v.Position == nil {
			retainedNull = true
		}
	}
	if !retainedNull {
		t.Fatal("absence override lost")
	}
	comp.References[0].Present = true
	returned, err := m.PrepareCandidate(ctx, base, changes, &comp)
	if err != nil {
		t.Fatal(err)
	}
	p, visible := returned.Snapshot().NodePosition(ids.root, ids.worker)
	if !visible || p == nil || *p == (Position{7654, -100000}) {
		t.Fatal("old coordinate resurrected")
	}
	replay, err := m.ConstructCandidate(ctx, base, changes, comp)
	if err != nil || replay.Tree() != returned.Tree() {
		t.Fatalf("reappearance final facts not exact: %v", err)
	}
}

func TestPlacementAuthoringMaterializesOnceAndStrictReplayCannotRepair(t *testing.T) {
	m := NewManager(t.TempDir())
	ctx := context.Background()
	base, err := m.CreateProject(ctx, "Stable authoring")
	if err != nil {
		t.Fatal(err)
	}
	a := m.NewComponentChange(base, nil, "A", "")
	comp := rootHomes(base, a)
	if _, err := m.ConstructCandidate(ctx, base, []ComponentChange{a}, comp); err == nil {
		t.Fatal("strict replay initialized missing coordinates")
	}
	first, err := m.PrepareCandidate(ctx, base, []ComponentChange{a}, &comp)
	if err != nil {
		t.Fatal(err)
	}
	saved := append([]NodePositionChange(nil), comp.NodePositions...)
	a.Description = "Longer documentation does not move nodes."
	a.DescriptionChanged = true
	second, err := m.PrepareCandidate(ctx, base, []ComponentChange{a}, &comp)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, comp.NodePositions) {
		t.Fatal("content edit changed position facts")
	}
	p, _ := second.Snapshot().NodePosition(base.RootDiagramID(), a.ID)
	before, _ := first.Snapshot().NodePosition(base.RootDiagramID(), a.ID)
	if !samePosition(p, before) {
		t.Fatal("content edit moved node")
	}
	comp.NodePositions[0].Position = nil
	if _, err := m.ConstructCandidate(ctx, base, []ComponentChange{a}, comp); err == nil {
		t.Fatal("strict replay allowed visible null")
	}
}

func TestReconciliationInitializesVisibilityIntroducedOnlyByCombinedSemantics(t *testing.T) {
	m, v2, ids, path := placementFixture(t)
	ctx := context.Background()
	initial := reconciliationAccepted(t, m, v2, nil, CandidateComposition{ArchitectureVersion: 3})
	// A adds a reference to the empty Diagram; P adds a Relationship from that
	// Component. The crossing endpoint exists in neither branch's empty Diagram.
	a := reconciliationAccepted(t, m, initial, nil, CandidateComposition{References: []ReferenceAppearanceChange{{ids.empty, ids.ledger, true}}})
	change, _ := initial.ChangeForAcceptedComponent(ids.ledger)
	change.RelationshipsChanged = true
	change.Relationships = []AuthoringRelationship{{ids.gateway, "new crossing"}}
	p := reconciliationProposed(t, m, initial, []ComponentChange{change}, CandidateComposition{})
	refs := gitText(t, "--git-dir", path, "show-ref")
	result, err := m.Reconcile(ctx, initial, a, p, nil)
	if err != nil || result.Status != "ready" {
		t.Fatalf("preview %+v %v", result, err)
	}
	position, visible := result.Candidate.Snapshot().NodePosition(ids.empty, ids.gateway)
	if !visible || position == nil {
		t.Fatal("combined boundary not initialized")
	}
	again, err := m.Reconcile(ctx, initial, a, p, nil)
	if err != nil || again.Candidate.Tree() != result.Candidate.Tree() {
		t.Fatal("preview allocation not deterministic")
	}
	replay, err := m.ConstructCandidate(ctx, a, result.Changes, result.Composition)
	if err != nil || replay.Tree() != result.Candidate.Tree() {
		t.Fatalf("residual is not exact: %v", err)
	}
	if refs != gitText(t, "--git-dir", path, "show-ref") {
		t.Fatal("preview changed refs")
	}
}
