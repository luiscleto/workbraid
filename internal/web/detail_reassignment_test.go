package web

import (
	"context"
	"testing"

	"workbraid/internal/agentapi"
)

// This workflow creates every object through the ordinary HTTP authoring path,
// applies the fixture through ordinary review/update, and reopens the store.
func TestOrdinaryDetailReassignmentAcrossRestart(t *testing.T) {
	data, ui := t.TempDir(), t.TempDir()
	h, handler := newHandler(testOrigin, ui, data)
	project := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Parent workflow"}))
	state := changeSetState(t, createAgentChangeSet(t, handler, project.StoreID, project.Revision, "Compose"))
	call := func(path string, payload any) map[string]any {
		t.Helper()
		result := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/"+path, payload))
		if !result.OK {
			t.Fatalf("%s: %+v", path, result)
		}
		value := resultMap(t, result)
		if generation, ok := value["generation"].(float64); ok {
			state.Generation = uint64(generation)
		}
		return value
	}
	create := func(title string) string {
		return call("components/create", agentapi.ComponentCreateRequest{StatePreconditions: state, Title: title, DiagramID: &project.RootDiagramID})["component_id"].(string)
	}
	a, b, c := create("Gateway"), create("Worker"), create("Records")
	child := call("diagrams/create-detail", agentapi.DiagramCreateDetailRequest{StatePreconditions: state, ComponentID: a, Title: "Inside Gateway"})["diagram_id"].(string)
	call("diagrams/reassign-detail", agentapi.DiagramReassignDetailRequest{StatePreconditions: state, DiagramID: child, AnchorComponentID: b})
	if pending := h.changeSets[state.ChangeSetID]; len(pending.detailReassignments) != 0 || pending.detailDiagrams[0].AnchorComponentID != b {
		t.Fatal("pending-new reanchor did not update its existing creation fact")
	}
	review := call("change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: state})
	call("architecture/update", agentapi.ArchitectureUpdateRequest{StoreID: state.StoreID, ChangeSetID: state.ChangeSetID, Generation: state.Generation, BaseRevision: review["base_revision"].(string), CandidateTree: review["candidate_tree"].(string)})
	accepted := h.loadedSnapshot.Revision()
	// Merely opening the browser chooser on Accepted creates no Change Set.
	opened := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/open", map[string]any{"project_slug": project.ProjectSlug}))
	options := postJSONRequest(t, handler, "/api/architecture/diagrams/parent-options", diagramMutationRequest{ProjectSlug: opened.ProjectSlug, StoreID: opened.StoreID, ExpectedRevision: opened.Revision, PendingGenerationObserved: true, DiagramID: child})
	if options.Code != 200 || len(h.changeSets) != 1 {
		t.Fatalf("opening options: %s", options.Body.String())
	}
	state = changeSetState(t, createAgentChangeSet(t, handler, project.StoreID, accepted, "Reanchor"))
	before := h.changeSets[state.ChangeSetID].refObject
	call("diagrams/parent-options", agentapi.DiagramParentOptionsRequest{StatePreconditions: state, DiagramID: child})
	call("diagrams/reassign-detail", agentapi.DiagramReassignDetailRequest{StatePreconditions: state, DiagramID: child, AnchorComponentID: b})
	if h.changeSets[state.ChangeSetID].refObject != before {
		t.Fatal("read/current-anchor no-op wrote state")
	}
	call("diagrams/reassign-detail", agentapi.DiagramReassignDetailRequest{StatePreconditions: state, DiagramID: child, AnchorComponentID: c})
	pending := h.changeSets[state.ChangeSetID]
	if len(pending.detailReassignments) != 1 || len(pending.changes) != 0 || len(pending.detailDiagrams) != 0 {
		t.Fatal("move not an ordinary composition-only fact")
	}
	_, _, comparison := captureReviewPresentation(pending.baseSnapshot, pending.candidate.Snapshot())
	if len(comparison.Components) != 0 || len(comparison.Relationships) != 0 || len(comparison.Diagrams) != 0 || len(comparison.Appearances) == 0 {
		t.Fatalf("false content delta: %+v", comparison)
	}
	object, tree := pending.refObject, pending.candidate.Tree()
	requireAgentErrorCode(t, postAgent(t, handler, "/api/agent/v2/diagrams/reassign-detail", agentapi.DiagramReassignDetailRequest{StatePreconditions: state, DiagramID: project.RootDiagramID, AnchorComponentID: a}), changeTargetIneligible)
	if h.changeSets[state.ChangeSetID].refObject != object {
		t.Fatal("rejected root edit changed state")
	}
	loaded, unavailable, err := h.architecture.LoadChangeSets(context.Background(), project.StoreID)
	if err != nil || len(unavailable) != 0 || len(loaded) != 2 {
		t.Fatalf("load: %v %v", err, unavailable)
	}
	h, handler = newHandler(testOrigin, ui, data)
	postJSONRequest(t, handler, "/api/projects/open", map[string]any{"project_slug": project.ProjectSlug})
	pending = h.changeSets[state.ChangeSetID]
	if pending == nil || pending.refObject != object || pending.candidate.Tree() != tree || h.loadedSnapshot.Revision() != accepted {
		t.Fatal("restart changed exact active/Accepted state")
	}
	// Returning to the base anchor removes the final reassignment fact.
	call("diagrams/reassign-detail", agentapi.DiagramReassignDetailRequest{StatePreconditions: state, DiagramID: child, AnchorComponentID: b})
	if len(h.changeSets[state.ChangeSetID].detailReassignments) != 0 {
		t.Fatal("return to base retained a redundant fact")
	}
	// The destination comes from the complete candidate, including Components
	// authored after the child already existed in Accepted.
	newParent := create("Candidate-only parent")
	call("diagrams/parent-options", agentapi.DiagramParentOptionsRequest{StatePreconditions: state, DiagramID: child})
	call("diagrams/reassign-detail", agentapi.DiagramReassignDetailRequest{StatePreconditions: state, DiagramID: child, AnchorComponentID: newParent})
	anchor, _, _ := h.changeSets[state.ChangeSetID].candidate.Snapshot().DiagramParent(child)
	if anchor != newParent {
		t.Fatal("candidate-only destination was unavailable")
	}
}
