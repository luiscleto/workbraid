package architecture

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDetailReassignmentUsesFinalAnchorsAndHomes(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(t.TempDir())
	bootstrap, err := manager.CreateProject(ctx, "Composition")
	if err != nil {
		t.Fatal(err)
	}
	path := mustStorePath(t, manager, bootstrap.StoreID())
	ids := diagramFixtureIDs{root: uuid.NewString(), detail: uuid.NewString(), empty: uuid.NewString(), gateway: uuid.NewString(), worker: uuid.NewString(), records: uuid.NewString(), ledger: uuid.NewString()}
	revision := commitV2Fixture(t, path, bootstrap.StoreID(), ids)
	gitText(t, "--git-dir", path, "update-ref", acceptedRef, revision, bootstrap.Revision())
	base, err := manager.LoadAccepted(ctx, bootstrap.StoreID())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name        string
		composition CandidateComposition
		valid       bool
	}{
		{"root home destination", CandidateComposition{DetailReassignments: []DetailReassignment{{ids.detail, ids.ledger}}}, true},
		{"joint swap", CandidateComposition{DetailReassignments: []DetailReassignment{{ids.detail, ids.records}, {ids.empty, ids.gateway}}}, true},
		{"joint homes and links", CandidateComposition{HomeMoves: []ComponentHomeMove{{ids.worker, ids.root}, {ids.gateway, ids.detail}}, DetailReassignments: []DetailReassignment{{ids.detail, ids.worker}}}, true},
		{"root forbidden", CandidateComposition{DetailReassignments: []DetailReassignment{{ids.root, ids.ledger}}}, false},
		{"occupied", CandidateComposition{DetailReassignments: []DetailReassignment{{ids.detail, ids.records}}}, false},
		{"descendant", CandidateComposition{DetailReassignments: []DetailReassignment{{ids.detail, ids.worker}}}, false},
		{"missing target", CandidateComposition{DetailReassignments: []DetailReassignment{{ids.detail, uuid.NewString()}}}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			refs := gitText(t, "--git-dir", path, "show-ref")
			candidate, err := manager.ConstructCandidate(ctx, base, nil, test.composition)
			if (err == nil) != test.valid {
				t.Fatalf("candidate error=%v valid=%v", err, test.valid)
			}
			if gitText(t, "--git-dir", path, "show-ref") != refs {
				t.Fatal("constructor changed refs")
			}
			if !test.valid {
				return
			}
			for _, link := range test.composition.DetailReassignments {
				anchor, _, _ := candidate.Snapshot().DiagramParent(link.DiagramID)
				if anchor != link.AnchorComponentID {
					t.Fatalf("anchor=%s, wanted %s", anchor, link.AnchorComponentID)
				}
			}
			for _, component := range base.components {
				if gitText(t, "--git-dir", path, "ls-tree", candidate.Tree(), component.path) != gitText(t, "--git-dir", path, "ls-tree", revision, component.path) {
					t.Fatal("composition rewrote a Component")
				}
			}
		})
	}
	// The exact subtree entries (both blobs and modes) survive a root-level move.
	nested, err := manager.ConstructCandidate(ctx, base, nil, CandidateComposition{DetailReassignments: []DetailReassignment{{ids.empty, ids.worker}}})
	if err != nil {
		t.Fatal(err)
	}
	nestedRevision, err := manager.CreateSuccessor(ctx, base, nested)
	if err != nil {
		t.Fatal(err)
	}
	gitText(t, "--git-dir", path, "update-ref", acceptedRef, nestedRevision, revision)
	nestedBase, err := manager.LoadAccepted(ctx, bootstrap.StoreID())
	if err != nil {
		t.Fatal(err)
	}
	moved, err := manager.ConstructCandidate(ctx, nestedBase, nil, CandidateComposition{DetailReassignments: []DetailReassignment{{ids.detail, ids.ledger}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, filename := range []string{"diagrams/detail.yaml", "diagrams/empty.yaml"} {
		if gitText(t, "--git-dir", path, "ls-tree", moved.Tree(), filename) != gitText(t, "--git-dir", path, "ls-tree", nestedRevision, filename) {
			t.Fatalf("subtree blob/mode changed: %s", filename)
		}
	}
}

func TestChangeStateVersion2ClosedReassignmentsAndVersion1Read(t *testing.T) {
	composition := CandidateComposition{ArchitectureVersion: 2, DetailReassignments: []DetailReassignment{{uuid.NewString(), uuid.NewString()}}}
	encoded, err := marshalChangeState(nil, composition)
	if err != nil {
		t.Fatal(err)
	}
	_, parsed, err := parseChangeState(encoded)
	if err != nil || len(parsed.DetailReassignments) != 1 || parsed.DetailReassignments[0] != composition.DetailReassignments[0] {
		t.Fatalf("round trip: %+v %v", parsed, err)
	}
	empty := []byte("format: workbraid-change-state\nversion: 2\ncomponents: []\nnew_component_homes: []\ndetail_diagrams: []\ndiagram_titles: []\nhome_moves: []\nreferences: []\ndetail_reassignments: []\n")
	v1 := strings.Replace(strings.Replace(string(empty), "version: 2", "version: 1", 1), "detail_reassignments: []\n", "", 1)
	if _, old, err := parseChangeState([]byte(v1)); err != nil || len(old.DetailReassignments) != 0 {
		t.Fatalf("v1: %+v %v", old, err)
	}
	for _, invalid := range []string{
		strings.Replace(string(empty), "detail_reassignments: []\n", "", 1),
		strings.Replace(string(empty), "version: 2", "version: 3", 1),
		v1 + "detail_reassignments: []\n",
		string(empty) + "unknown: true\n",
		strings.Replace(string(encoded), "anchor_component_id:", "unknown:", 1),
	} {
		if _, _, err := parseChangeState([]byte(invalid)); err == nil {
			t.Fatalf("accepted invalid encoding: %s", invalid)
		}
	}
	newID := uuid.NewString()
	if err := validateChangeState(nil, CandidateComposition{DetailDiagrams: []DetailDiagramChange{{ID: newID, Path: "diagrams/new.yaml", Title: "New", AnchorComponentID: uuid.NewString()}}, DetailReassignments: []DetailReassignment{{newID, uuid.NewString()}}}); err == nil {
		t.Fatal("accepted reassignment for a pending-new child")
	}
	if err := validateChangeState(nil, CandidateComposition{DetailReassignments: append(composition.DetailReassignments, composition.DetailReassignments...)}); err == nil {
		t.Fatal("accepted duplicate child entries")
	}
}
