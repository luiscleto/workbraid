package architecture

import (
	"encoding/json"
	"errors"
	"testing"
)

// The independent review reproduction used three individually valid home
// moves. Explicitly restoring only Z breaks their combined cycle, but a side
// choice must also retain the selected side's other involved facts.
func TestReconciliationStructuralSideChoiceRetainsInvolvedFacts(t *testing.T) {
	for _, structure := range []string{"homes", "homes and anchor"} {
		for _, choice := range []string{"accepted", "proposed"} {
			t.Run(structure+"/"+choice, func(t *testing.T) {
				data := t.TempDir()
				m := NewManager(data)
				empty, err := m.CreateProject(t.Context(), "Structural side choice")
				if err != nil {
					t.Fatal(err)
				}
				var components []ComponentChange
				for _, title := range []string{"X", "Y", "Z", "Alternate", "Outside left", "Outside right"} {
					components = append(components, m.NewComponentChange(empty, components, title, ""))
				}
				x, y, z, alternate, outsideLeft, outsideRight := components[0], components[1], components[2], components[3], components[4], components[5]
				dx := empty.NewDetailDiagramChange(nil, "X detail", x.ID)
				dy := empty.NewDetailDiagramChange([]DetailDiagramChange{dx}, "Y detail", y.ID)
				dz := empty.NewDetailDiagramChange([]DetailDiagramChange{dx, dy}, "Z detail", z.ID)
				initial := rootHomes(empty, components...)
				initial.DetailDiagrams = []DetailDiagramChange{dx, dy, dz}
				b := reconciliationAccepted(t, m, empty, components, initial)
				left := CandidateComposition{
					HomeMoves:  []ComponentHomeMove{{x.ID, dy.ID}, {outsideLeft.ID, dx.ID}},
					References: []ReferenceAppearanceChange{{DiagramID: dy.ID, ComponentID: z.ID, Present: true}},
				}
				right := CandidateComposition{
					HomeMoves:  []ComponentHomeMove{{y.ID, dz.ID}, {z.ID, dx.ID}, {outsideRight.ID, dz.ID}},
					References: []ReferenceAppearanceChange{{DiagramID: dz.ID, ComponentID: x.ID, Present: true}},
				}
				if structure == "homes and anchor" {
					right.HomeMoves[0].ComponentID = alternate.ID
					right.DetailReassignments = []DetailReassignment{{dy.ID, alternate.ID}}
				}
				leftText, _ := b.ChangeForAcceptedComponent(x.ID)
				leftText.Title, leftText.TitleChanged = "Left title", true
				leftText.Relationships, leftText.RelationshipsChanged = []AuthoringRelationship{{z.ID, "left involved"}}, true
				rightText, _ := b.ChangeForAcceptedComponent(x.ID)
				rightText.Description, rightText.DescriptionChanged = "\nExact right body\r\n", true
				rightText.Relationships, rightText.RelationshipsChanged = []AuthoringRelationship{{z.ID, "right involved"}, {outsideRight.ID, "outside group"}}, true
				av, pv, ac, pc := left, right, leftText, rightText
				if choice == "proposed" {
					av, pv, ac, pc = right, left, rightText, leftText
				}
				a := reconciliationAccepted(t, m, b, []ComponentChange{ac}, av)
				p := reconciliationProposed(t, m, b, []ComponentChange{pc}, pv)
				selected := a
				if choice == "proposed" {
					selected = p.Snapshot()
				}
				path := mustStorePath(t, m, b.StoreID())
				refs := gitText(t, "--git-dir", path, "show-ref")
				preview, err := m.Reconcile(t.Context(), b, a, p, nil)
				if err != nil || preview.Status != "needs_resolution" || len(preview.Conflicts) != 1 || preview.Conflicts[0].Locator.Reason != "hierarchy_cycle" {
					t.Fatalf("cycle preview: %+v %v", preview, err)
				}
				locator := preview.Conflicts[0].Locator
				request := ReconciliationResolution{Locator: locator, Choice: choice, Value: &ReconciliationValue{Homes: []NewComponentHome{{z.ID, b.RootDiagramID()}}}}
				wire, err := json.Marshal(request)
				if err != nil {
					t.Fatal(err)
				}
				var decoded ReconciliationResolution
				if err := json.Unmarshal(wire, &decoded); err != nil {
					t.Fatal(err)
				}
				result, err := m.Reconcile(t.Context(), b, a, p, []ReconciliationResolution{decoded})
				if err != nil || result.Status != "ready" || result.Candidate == nil {
					t.Fatalf("sparse %s choice: %+v %v", choice, result, err)
				}
				actual, want := snapshotReconciliationFacts(result.Candidate.Snapshot()), snapshotReconciliationFacts(selected)
				for _, id := range locator.ComponentIDs {
					if actual.homes[id] != want.homes[id] {
						t.Fatalf("%s side retained opposing home for %s: got %s want %s", choice, id, actual.homes[id], want.homes[id])
					}
					for _, diagram := range locator.DiagramIDs {
						pair := reconciliationPair{diagram, id}
						if actual.references[pair] != want.references[pair] {
							t.Fatalf("%s side lost reference value for %+v", choice, pair)
						}
					}
				}
				for _, id := range locator.DiagramIDs {
					if actual.anchors[id] != want.anchors[id] {
						t.Fatalf("%s side lost anchor for %s", choice, id)
					}
				}
				if actual.relationships[reconciliationRelationship{x.ID, z.ID, "left involved"}] != 1 || actual.relationships[reconciliationRelationship{x.ID, z.ID, "right involved"}] != 0 {
					t.Fatal("selected side did not retain involved Relationship counts, including zero")
				}
				if actual.homes[outsideLeft.ID] != dx.ID || actual.homes[outsideRight.ID] != dz.ID || actual.relationships[reconciliationRelationship{x.ID, outsideRight.ID, "outside group"}] != 1 || actual.components[x.ID].Title != leftText.Title || actual.components[x.ID].Description != rightText.Description {
					t.Fatal("structural choice replaced unrelated automatically combined facts")
				}
				if gitText(t, "--git-dir", path, "show-ref") != refs {
					t.Fatal("preview changed authoritative refs")
				}
				id, name, err := m.NewChangeSet(nil, "Resolved cycle")
				if err != nil {
					t.Fatal(err)
				}
				record := ChangeSet{ID: id, Name: name, Lifecycle: "active", BaseRevision: a.Revision(), BaseSnapshot: a, Generation: 2, Proposal: "Exact proposal\r\n", Changes: result.Changes, Composition: result.Composition, Candidate: result.Candidate}
				state, err := m.WriteActiveChangeSet(t.Context(), b.StoreID(), record, "")
				if err != nil {
					t.Fatal(err)
				}
				loaded, unavailable, err := NewManager(data).LoadChangeSets(t.Context(), b.StoreID())
				if err != nil || len(unavailable) != 0 || len(loaded) != 1 || loaded[0].RefObject != state || loaded[0].Candidate.Tree() != result.Candidate.Tree() || loaded[0].Proposal != record.Proposal {
					t.Fatalf("ordinary residual restart: %+v %v %v", loaded, unavailable, err)
				}
				if gitText(t, "--git-dir", path, "rev-parse", acceptedRef) != a.Revision() {
					t.Fatal("residual write changed Accepted")
				}
			})
		}
	}
}

func TestReconciliationStructuralSideChoiceRejectsImplicitContradiction(t *testing.T) {
	m, b, ids := reconciliationFixture(t)
	a := reconciliationAccepted(t, m, b, nil, CandidateComposition{HomeMoves: []ComponentHomeMove{{ids.gateway, ids.empty}}})
	p := reconciliationProposed(t, m, b, nil, CandidateComposition{
		HomeMoves:  []ComponentHomeMove{{ids.records, ids.detail}},
		References: []ReferenceAppearanceChange{{DiagramID: ids.empty, ComponentID: ids.gateway, Present: true}},
	})
	preview, err := m.Reconcile(t.Context(), b, a, p, nil)
	if err != nil || len(preview.Conflicts) != 2 {
		t.Fatalf("overlapping structural groups: %+v %v", preview, err)
	}
	var choices []ReconciliationResolution
	for _, conflict := range preview.Conflicts {
		choice := "accepted"
		if conflict.Locator.Reason == "home_reference_overlap" {
			choice = "proposed"
		}
		choices = append(choices, ReconciliationResolution{Locator: conflict.Locator, Choice: choice, Value: &ReconciliationValue{}})
	}
	path := mustStorePath(t, m, b.StoreID())
	refs := gitText(t, "--git-dir", path, "show-ref")
	_, err = m.Reconcile(t.Context(), b, a, p, choices)
	var domain *ReconciliationError
	if !errors.As(err, &domain) || domain.Code != "invalid_request" || domain.Reason != "contradictory structural assignments" {
		t.Fatalf("contradictory implicit side values: %v", err)
	}
	if gitText(t, "--git-dir", path, "show-ref") != refs {
		t.Fatal("contradictory preview changed authoritative refs")
	}
}

func TestReconciliationStructuralSideChoiceRequiresMissingSideAssignment(t *testing.T) {
	m := NewManager(t.TempDir())
	empty, err := m.CreateProject(t.Context(), "New child in cycle")
	if err != nil {
		t.Fatal(err)
	}
	x := m.NewComponentChange(empty, nil, "X", "")
	y := m.NewComponentChange(empty, []ComponentChange{x}, "Y", "")
	dy := empty.NewDetailDiagramChange(nil, "Y detail", y.ID)
	initial := rootHomes(empty, x, y)
	initial.DetailDiagrams = []DetailDiagramChange{dy}
	b := reconciliationAccepted(t, m, empty, []ComponentChange{x, y}, initial)
	a := reconciliationAccepted(t, m, b, nil, CandidateComposition{HomeMoves: []ComponentHomeMove{{x.ID, dy.ID}}})
	dx := b.NewDetailDiagramChange(nil, "X detail", x.ID)
	p := reconciliationProposed(t, m, b, nil, CandidateComposition{DetailDiagrams: []DetailDiagramChange{dx}, HomeMoves: []ComponentHomeMove{{y.ID, dx.ID}}})
	path := mustStorePath(t, m, b.StoreID())
	refs := gitText(t, "--git-dir", path, "show-ref")
	preview, err := m.Reconcile(t.Context(), b, a, p, nil)
	if err != nil || len(preview.Conflicts) != 1 || preview.Conflicts[0].Locator.Reason != "hierarchy_cycle" {
		t.Fatalf("new child cycle: %+v %v", preview, err)
	}
	choice := ReconciliationResolution{Locator: preview.Conflicts[0].Locator, Choice: "accepted", Value: &ReconciliationValue{}}
	partial, err := m.Reconcile(t.Context(), b, a, p, []ReconciliationResolution{choice})
	if err != nil || partial.Status != "needs_resolution" || partial.Candidate != nil {
		t.Fatalf("side absent child silently assigned: %+v %v", partial, err)
	}
	choice.Value.DetailAnchors = []DetailReassignment{{dx.ID, x.ID}}
	resolved, err := m.Reconcile(t.Context(), b, a, p, []ReconciliationResolution{choice})
	if err != nil || resolved.Status != "ready" || resolved.Candidate == nil {
		t.Fatalf("explicit new child assignment: %+v %v", resolved, err)
	}
	if !resolved.Candidate.Snapshot().HasDiagram(dx.ID) || !resolved.Candidate.Snapshot().HasDiagram(dy.ID) || len(resolved.Composition.DetailDiagrams) != 1 || resolved.Composition.DetailDiagrams[0].ID != dx.ID || len(resolved.Composition.DetailReassignments) != 0 {
		t.Fatal("child identity or ordinary creation fact lost")
	}
	if gitText(t, "--git-dir", path, "show-ref") != refs {
		t.Fatal("preview changed authoritative refs")
	}
}
