package architecture

import (
	"context"
	"reflect"
	"testing"
)

func TestReconciliationOrdinaryResidualCombinesAndReconstructs(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(t.TempDir())
	empty, err := manager.CreateProject(ctx, "Reconcile")
	if err != nil {
		t.Fatal(err)
	}
	one := manager.NewComponentChange(empty, nil, "Gateway", "Original\n")
	two := manager.NewComponentChange(empty, []ComponentChange{one}, "Worker", "")
	initial, err := manager.ConstructCandidate(ctx, empty, []ComponentChange{one, two}, rootHomes(empty, one, two))
	if err != nil {
		t.Fatal(err)
	}
	revision, err := manager.CreateSuccessor(ctx, empty, initial)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.AdvanceAccepted(ctx, empty, revision); err != nil {
		t.Fatal(err)
	}
	base, err := manager.LoadAccepted(ctx, empty.StoreID())
	if err != nil {
		t.Fatal(err)
	}
	a, _ := base.ChangeForAcceptedComponent(one.ID)
	a.Title = "Accepted Gateway"
	a.TitleChanged = true
	acceptedCandidate, err := manager.ConstructCandidate(ctx, base, []ComponentChange{a}, CandidateComposition{})
	if err != nil {
		t.Fatal(err)
	}
	acceptedRevision, err := manager.CreateSuccessor(ctx, base, acceptedCandidate)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.AdvanceAccepted(ctx, base, acceptedRevision); err != nil {
		t.Fatal(err)
	}
	accepted, err := manager.LoadAccepted(ctx, empty.StoreID())
	if err != nil {
		t.Fatal(err)
	}
	p, _ := base.ChangeForAcceptedComponent(one.ID)
	p.Description = "\nExact proposed body.\r\n"
	p.DescriptionChanged = true
	p.Relationships = []AuthoringRelationship{{TargetID: two.ID, Label: "calls\nλ"}, {TargetID: two.ID, Label: "calls\nλ"}}
	p.RelationshipsChanged = true
	proposed, err := manager.ConstructCandidate(ctx, base, []ComponentChange{p}, CandidateComposition{})
	if err != nil {
		t.Fatal(err)
	}
	path := mustStorePath(t, manager, base.StoreID())
	refs := gitText(t, "--git-dir", path, "show-ref")
	result, err := manager.Reconcile(ctx, base, accepted, proposed, nil)
	if err != nil || result.Status != "ready" || result.Candidate == nil {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if gitText(t, "--git-dir", path, "show-ref") != refs {
		t.Fatal("preview changed authority")
	}
	if len(result.Changes) != 1 || result.Changes[0].TitleChanged || !result.Changes[0].DescriptionChanged || !result.Changes[0].RelationshipsChanged {
		t.Fatalf("not minimal ordinary facts: %+v", result.Changes)
	}
	got, _ := result.Candidate.Snapshot().ChangeForAcceptedComponent(one.ID)
	if got.Title != a.Title || got.Description != p.Description || !reflect.DeepEqual(got.Relationships, p.Relationships) {
		t.Fatalf("semantic result: %+v", got)
	}
	id, name, _ := manager.NewChangeSet(nil, "Residual")
	record := ChangeSet{ID: id, Name: name, Lifecycle: "active", BaseRevision: accepted.Revision(), BaseSnapshot: accepted, Generation: 2, Proposal: "# Exact proposal\r\n", Changes: result.Changes, Composition: result.Composition, Candidate: result.Candidate}
	object, err := manager.WriteActiveChangeSet(ctx, base.StoreID(), record, "")
	if err != nil {
		t.Fatal(err)
	}
	loaded, unavailable, err := manager.LoadChangeSets(ctx, base.StoreID())
	if err != nil || len(unavailable) != 0 || len(loaded) != 1 || loaded[0].RefObject != object || loaded[0].Candidate.Tree() != result.Candidate.Tree() {
		t.Fatalf("residual reconstruction: %v %v", err, unavailable)
	}
}

func TestReconciliationCompetingChildrenRequireBothFinalAnchors(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(t.TempDir())
	empty, err := manager.CreateProject(ctx, "Children")
	if err != nil {
		t.Fatal(err)
	}
	one := manager.NewComponentChange(empty, nil, "Gateway", "")
	two := manager.NewComponentChange(empty, []ComponentChange{one}, "Worker", "")
	initial, err := manager.ConstructCandidate(ctx, empty, []ComponentChange{one, two}, rootHomes(empty, one, two))
	if err != nil {
		t.Fatal(err)
	}
	revision, err := manager.CreateSuccessor(ctx, empty, initial)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.AdvanceAccepted(ctx, empty, revision); err != nil {
		t.Fatal(err)
	}
	base, err := manager.LoadAccepted(ctx, empty.StoreID())
	if err != nil {
		t.Fatal(err)
	}
	left := base.NewDetailDiagramChange(nil, "Operations", one.ID)
	right := base.NewDetailDiagramChange(nil, "Runtime", one.ID)
	ac, err := manager.ConstructCandidate(ctx, base, nil, CandidateComposition{DetailDiagrams: []DetailDiagramChange{left}})
	if err != nil {
		t.Fatal(err)
	}
	ar, err := manager.CreateSuccessor(ctx, base, ac)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.AdvanceAccepted(ctx, base, ar); err != nil {
		t.Fatal(err)
	}
	accepted, err := manager.LoadAccepted(ctx, base.StoreID())
	if err != nil {
		t.Fatal(err)
	}
	proposed, err := manager.ConstructCandidate(ctx, base, nil, CandidateComposition{DetailDiagrams: []DetailDiagramChange{right}})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := manager.Reconcile(ctx, base, accepted, proposed, nil)
	if err != nil || preview.Status != "needs_resolution" || len(preview.Conflicts) != 1 {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	locator := preview.Conflicts[0].Locator
	if locator.Reason != "competing_children" {
		t.Fatalf("wrong conflict: %+v", locator)
	}
	for _, choice := range []string{"accepted", "proposed"} {
		t.Run(choice, func(t *testing.T) {
			anchors := []DetailReassignment{{DiagramID: left.ID, AnchorComponentID: one.ID}, {DiagramID: right.ID, AnchorComponentID: two.ID}}
			if choice == "proposed" {
				anchors[0].AnchorComponentID = two.ID
				anchors[1].AnchorComponentID = one.ID
			}
			result, err := manager.Reconcile(ctx, base, accepted, proposed, []ReconciliationResolution{{Locator: locator, Choice: choice, Value: &ReconciliationValue{DetailAnchors: anchors}}})
			if err != nil || result.Status != "ready" || result.Candidate == nil {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			for _, assignment := range anchors {
				got, _, _ := result.Candidate.Snapshot().DiagramParent(assignment.DiagramID)
				if got != assignment.AnchorComponentID {
					t.Fatal("child lost or wrong anchor")
				}
			}
			if len(result.Composition.DetailDiagrams) != 1 || result.Composition.DetailDiagrams[0].ID != right.ID {
				t.Fatal("new child identity was replaced")
			}
		})
	}
	partial, err := manager.Reconcile(ctx, base, accepted, proposed, []ReconciliationResolution{{Locator: locator, Choice: "accepted", Value: &ReconciliationValue{DetailAnchors: []DetailReassignment{{DiagramID: right.ID, AnchorComponentID: two.ID}}}}})
	if err != nil || partial.Status != "needs_resolution" || partial.Candidate != nil {
		t.Fatalf("partial assignment accepted: %+v %v", partial, err)
	}
}
