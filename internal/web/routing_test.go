package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"workbraid/internal/agentapi"
)

func TestRoutingPublicExactSlotNoopInvalidResetRestart(t *testing.T) {
	data := t.TempDir()
	state, handler := newHandler(testOrigin, t.TempDir(), data)
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Routing authority"}))
	token := changeSetState(t, createAgentChangeSet(t, handler, created.StoreID, created.Revision, "Routes"))
	mutate := func(path string, request any) map[string]any {
		t.Helper()
		response := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/"+path, request))
		if !response.OK {
			t.Fatalf("%s: %+v", path, response.Error)
		}
		result := resultMap(t, response)
		token.Generation = uint64(result["generation"].(float64))
		return result
	}
	source := mutate("components/create", agentapi.ComponentCreateRequest{StatePreconditions: token, Title: "Source", DiagramID: &created.RootDiagramID})["component_id"].(string)
	target := mutate("components/create", agentapi.ComponentCreateRequest{StatePreconditions: token, Title: "Target", DiagramID: &created.RootDiagramID})["component_id"].(string)
	for range 2 {
		mutate("relationships/add", agentapi.RelationshipAddRequest{StatePreconditions: token, SourceID: source, TargetID: target, Label: "calls"})
	}
	request := func() agentapi.DiagramSetRouteRequest {
		return agentapi.DiagramSetRouteRequest{DiagramRestoreDefaultRouteRequest: agentapi.DiagramRestoreDefaultRouteRequest{DiagramAutoLayoutRequest: agentapi.DiagramAutoLayoutRequest{StatePreconditions: token, DiagramID: created.RootDiagramID}, SourceID: source, TargetID: target, Label: "calls", Occurrence: 2}, Bend: 125}
	}
	mutate("diagrams/set-route", request())
	before := state.changeSets[token.ChangeSetID]
	for _, r := range before.candidate.Snapshot().DiagramRoutes(created.RootDiagramID) {
		if r.Occurrence == 1 && r.Route != nil || r.Occurrence == 2 && (r.Route == nil || r.Route.Bend != 125) {
			t.Fatalf("wrong slot %+v", r)
		}
	}
	reviewed := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: token}))
	if !reviewed.OK {
		t.Fatal(reviewed.Error)
	}
	before = state.changeSets[token.ChangeSetID]
	if result := mutate("diagrams/set-route", request()); result["unchanged"] != true {
		t.Fatal("same bend was not no-op")
	}
	if after := state.changeSets[token.ChangeSetID]; after.refObject != before.refObject || after.review == nil {
		t.Fatal("no-op invalidated review")
	}
	stale := request()
	stale.Generation--
	requireAgentErrorCode(t, postAgent(t, handler, "/api/agent/v2/diagrams/set-route", stale), "change_set_generation_mismatch")
	empty := ""
	mutate("components/edit", agentapi.ComponentEditRequest{StatePreconditions: token, ComponentID: target, Title: &empty})
	if len(state.changeSets[token.ChangeSetID].edgeRoutes) != 1 {
		t.Fatal("unrelated invalid edit lost route")
	}
	mutate("relationships/remove", agentapi.RelationshipRemoveRequest{StatePreconditions: token, SourceID: source, TargetID: target, Label: "calls", Occurrence: 2})
	state, handler = newHandler(testOrigin, t.TempDir(), data)
	opened := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/projects/open", agentapi.ProjectOpenRequest{Slug: created.ProjectSlug}))
	if !opened.OK {
		t.Fatal(opened.Error)
	}
	if pending := state.changeSets[token.ChangeSetID]; pending == nil || pending.candidate != nil || len(pending.edgeRoutes) != 0 {
		t.Fatal("restart did not retain invalid state and targeted pending-route reset")
	}
	mutate("relationships/add", agentapi.RelationshipAddRequest{StatePreconditions: token, SourceID: source, TargetID: target, Label: "calls"})
	title := "Target"
	mutate("components/edit", agentapi.ComponentEditRequest{StatePreconditions: token, ComponentID: target, Title: &title})
	for _, r := range state.changeSets[token.ChangeSetID].candidate.Snapshot().DiagramRoutes(created.RootDiagramID) {
		if r.Route != nil {
			t.Fatal("route resurrected")
		}
	}
}

func TestRoutingClosedPublicRequests(t *testing.T) {
	_, handler := newHandler(testOrigin, t.TempDir(), t.TempDir())
	for _, raw := range []string{`{"bend":null}`, `{"bend":0.5}`, `{"bend":100001}`, `{"bend":0,"bend":1}`, `{"occurrence":0}`, `{"occurrence":null}`, `{"label":null}`} {
		for _, path := range []string{"/api/architecture/diagrams/set-route", "/api/agent/v2/diagrams/set-route"} {
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(raw))
			request.Header.Set("Origin", testOrigin)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != 400 {
				t.Fatalf("accepted %s: %s", raw, response.Body.String())
			}
		}
	}
}
