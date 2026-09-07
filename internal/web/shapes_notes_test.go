package web

import (
	"path/filepath"
	"testing"
	"workbraid/internal/agentapi"
)

func TestFreshProposalRetainsShapeRemovalThroughInvalidRelationshipRestart(t *testing.T) {
	f := newNativeRefreshFixture(t, false)
	token := changeSetState(t, createAgentChangeSet(t, f.handler, f.base.StoreID, f.base.Revision, "Seed boundary"))
	call := func(path string, request any) map[string]any {
		t.Helper()
		response := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/"+path, request))
		if !response.OK {
			t.Fatalf("%s: %+v", path, response.Error)
		}
		token.Generation = f.state.changeSets[token.ChangeSetID].generation
		return resultMap(t, response)
	}
	detail := call("diagrams/create-detail", agentapi.DiagramCreateDetailRequest{StatePreconditions: token, ComponentID: f.component, Title: "Inside"})["diagram_id"].(string)
	source := call("components/create", agentapi.ComponentCreateRequest{StatePreconditions: token, Title: "Source", DiagramID: &detail})["component_id"].(string)
	call("relationships/add", agentapi.RelationshipAddRequest{StatePreconditions: token, SourceID: source, TargetID: f.component, Label: "calls"})
	call("diagrams/set-shape", agentapi.DiagramSetShapeRequest{DiagramRestoreDefaultSizeRequest: agentapi.DiagramRestoreDefaultSizeRequest{DiagramAutoLayoutRequest: agentapi.DiagramAutoLayoutRequest{StatePreconditions: token, DiagramID: detail}, ComponentID: f.component}, Shape: "diamond"})
	call("diagrams/set-route", agentapi.DiagramSetRouteRequest{DiagramRestoreDefaultRouteRequest: agentapi.DiagramRestoreDefaultRouteRequest{DiagramAutoLayoutRequest: agentapi.DiagramAutoLayoutRequest{StatePreconditions: token, DiagramID: detail}, SourceID: source, TargetID: f.component, Label: "calls", Occurrence: 1}, Bend: 73})
	review := call("change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: token})
	call("architecture/update", agentapi.ArchitectureUpdateRequest{StoreID: token.StoreID, ChangeSetID: token.ChangeSetID, Generation: token.Generation, BaseRevision: review["base_revision"].(string), CandidateTree: review["candidate_tree"].(string)})
	token = changeSetState(t, createAgentChangeSet(t, f.handler, token.StoreID, f.state.loadedSnapshot.Revision(), "Invalid target"))
	edit := func(old, target string) {
		call("relationships/edit", agentapi.RelationshipEditRequest{StatePreconditions: token, SourceID: source, OldTargetID: old, OldLabel: "calls", Occurrence: 1, TargetID: target, Label: "calls"})
	}
	edit(f.component, "invalid-target")
	pending := f.state.changeSets[token.ChangeSetID]
	if pending.candidate != nil || len(pending.nodeShapes) != 1 || pending.nodeShapes[0].Shape != nil || len(pending.edgeRoutes) != 1 || pending.edgeRoutes[0].Route != nil {
		t.Fatalf("missing invalid state tombstones: %+v", pending)
	}
	object := pending.refObject
	f.state, f.handler = newHandler(testOrigin, t.TempDir(), filepath.Dir(filepath.Dir(f.storePath)))
	opened := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/projects/open", agentapi.ProjectOpenRequest{Slug: f.base.ProjectSlug}))
	if !opened.OK {
		t.Fatal(opened.Error)
	}
	pending = f.state.changeSets[token.ChangeSetID]
	if pending == nil || pending.refObject != object || pending.candidate != nil {
		t.Fatal("invalid authored state did not survive restart")
	}
	edit("invalid-target", f.component)
	pending = f.state.changeSets[token.ChangeSetID]
	if pending.candidate == nil {
		t.Fatal("repair failed")
	}
	shape, ok := pending.candidate.Snapshot().NodeShape(detail, f.component)
	if !ok || shape != nil {
		t.Fatal("shape resurrected on repair")
	}
	for _, route := range pending.candidate.Snapshot().DiagramRoutes(detail) {
		if route.Route != nil {
			t.Fatal("route resurrected on repair")
		}
	}
}
