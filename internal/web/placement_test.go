package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"workbraid/internal/agentapi"
	"workbraid/internal/architecture"
)

func TestPlacementSharedAuthorityAbsenceCompositionAndRestart(t *testing.T) {
	f := newNativeRefreshFixture(t, false)
	state := changeSetState(t, createAgentChangeSet(t, f.handler, f.base.StoreID, f.base.Revision, "Seed pins"))
	call := func(path string, v any) map[string]any {
		t.Helper()
		e := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/"+path, v))
		if !e.OK {
			t.Fatalf("%s: %+v", path, e.Error)
		}
		result := resultMap(t, e)
		state.Generation = f.state.changeSets[state.ChangeSetID].generation
		return result
	}
	root := f.base.RootDiagramID
	worker := call("components/create", agentapi.ComponentCreateRequest{StatePreconditions: state, Title: "Worker", DiagramID: &root})["component_id"].(string)
	detail := call("diagrams/create-detail", agentapi.DiagramCreateDetailRequest{StatePreconditions: state, ComponentID: f.component, Title: "Runtime"})["diagram_id"].(string)
	call("components/move-home", agentapi.ComponentMoveHomeRequest{StatePreconditions: state, ComponentID: worker, DiagramID: detail})
	call("diagrams/show-component", agentapi.DiagramComponentRequest{StatePreconditions: state, DiagramID: root, ComponentID: worker})
	set := func(d, c string, x, y int) {
		t.Helper()
		call("diagrams/set-position", agentapi.DiagramSetPositionRequest{DiagramAutoLayoutRequest: agentapi.DiagramAutoLayoutRequest{StatePreconditions: state, DiagramID: d}, ComponentID: c, X: x, Y: y})
	}
	set(root, worker, 100, 100)
	set(root, f.component, -200, 0)
	set(detail, worker, 33, 44)
	accept := func() {
		t.Helper()
		review := call("change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: state})
		call("architecture/update", agentapi.ArchitectureUpdateRequest{StoreID: state.StoreID, ChangeSetID: state.ChangeSetID, Generation: state.Generation, BaseRevision: review["base_revision"].(string), CandidateTree: review["candidate_tree"].(string)})
	}
	accept()
	accepted := f.state.loadedSnapshot.Revision()
	state = changeSetState(t, createAgentChangeSet(t, f.handler, state.StoreID, accepted, "Reset semantics"))
	call("diagrams/stop-showing-component", agentapi.DiagramComponentRequest{StatePreconditions: state, DiagramID: root, ComponentID: worker})
	call("diagrams/auto-layout", agentapi.DiagramAutoLayoutRequest{StatePreconditions: state, DiagramID: root})
	pending := f.state.changeSets[state.ChangeSetID]
	nulls := 0
	for _, v := range pending.nodePositions {
		if v.DiagramID == root && v.Position == nil {
			nulls++
		}
	}
	if nulls != 1 {
		t.Fatalf("reset erased required null: %+v", pending.nodePositions)
	}
	call("diagrams/show-component", agentapi.DiagramComponentRequest{StatePreconditions: state, DiagramID: root, ComponentID: worker})
	if p, ok := f.state.changeSets[state.ChangeSetID].candidate.Snapshot().NodePosition(root, worker); !ok || p == nil || *p == (architecture.Position{X: 100, Y: 100}) {
		t.Fatal("removed pin resurrected")
	}
	set(root, worker, 0, 0)
	call("components/move-home", agentapi.ComponentMoveHomeRequest{StatePreconditions: state, ComponentID: worker, DiagramID: root})
	pending = f.state.changeSets[state.ChangeSetID]
	if p, ok := pending.candidate.Snapshot().NodePosition(root, worker); !ok || p == nil || *p != (architecture.Position{}) {
		t.Fatal("reference to home lost destination zero pin")
	}
	call("components/move-home", agentapi.ComponentMoveHomeRequest{StatePreconditions: state, ComponentID: worker, DiagramID: detail})
	if p, ok := f.state.changeSets[state.ChangeSetID].candidate.Snapshot().NodePosition(detail, worker); !ok || p == nil || *p == (architecture.Position{X: 33, Y: 44}) {
		t.Fatal("home round trip resurrected source pin")
	}
	set(detail, worker, 70, -90)
	call("change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: state})
	pending = f.state.changeSets[state.ChangeSetID]
	object, tree, generation := pending.refObject, pending.candidate.Tree(), pending.generation
	set(detail, worker, 70, -90)
	if f.state.changeSets[state.ChangeSetID].refObject != object {
		t.Fatal("no-op set invalidated review")
	}
	stale := state
	stale.Generation--
	requireAgentErrorCode(t, postAgent(t, f.handler, "/api/agent/v2/diagrams/auto-layout", agentapi.DiagramAutoLayoutRequest{StatePreconditions: stale, DiagramID: detail}), "change_set_generation_mismatch")
	data := filepath.Dir(filepath.Dir(f.storePath))
	restarted, handler := newHandler(testOrigin, t.TempDir(), data)
	postJSONRequest(t, handler, "/api/projects/open", map[string]any{"project_slug": f.base.ProjectSlug})
	loaded := restarted.changeSets[state.ChangeSetID]
	if loaded == nil || loaded.refObject != object || loaded.candidate.Tree() != tree || loaded.generation != generation || restarted.loadedSnapshot.Revision() != accepted {
		t.Fatal("restart changed exact state")
	}
	_, _, comparison := captureReviewPresentation(loaded.baseSnapshot, loaded.candidate.Snapshot())
	if len(comparison.Components) != 0 || len(comparison.Relationships) != 0 || len(comparison.NodePositions) == 0 {
		t.Fatalf("placement false semantic delta %+v", comparison)
	}
}

func TestPlacementAcceptedPreconditionsAndClosedRequests(t *testing.T) {
	f := newNativeRefreshFixture(t, false)
	base := diagramMutationRequest{ProjectSlug: f.base.ProjectSlug, StoreID: f.base.StoreID, ExpectedRevision: f.base.Revision, PendingGenerationObserved: true, DiagramID: f.base.RootDiagramID, ComponentID: f.component}
	// An already arranged native singleton creates no implicit proposal.
	layout := base
	layout.ComponentID = ""
	if response := postJSONRequest(t, f.handler, "/api/architecture/diagrams/auto-layout", layout); response.Code != 200 || len(f.state.changeSets) != 0 {
		t.Fatalf("auto-layout: %s", response.Body.String())
	}
	x, y := 1, -2
	base.X, base.Y = &x, &y
	wrong := base
	wrong.ExpectedRevision = strings.Repeat("f", 40)
	if response := postJSONRequest(t, f.handler, "/api/architecture/diagrams/set-position", wrong); response.Code == 200 || len(f.state.changeSets) != 0 {
		t.Fatal("stale implicit drag created a proposal")
	}
	created := decodeArchitectureResponse(t, postJSONRequest(t, f.handler, "/api/architecture/diagrams/set-position", base))
	if created.Changes == nil || created.Changes.Generation != 1 || len(f.state.changeSets) != 1 {
		t.Fatal("drop not one implicit generation")
	}
	state := agentapi.StatePreconditions{StoreID: base.StoreID, ChangeSetID: created.Changes.ID, Generation: 1}
	for _, pair := range []string{`"x":null,"y":0`, `"x":0`, `"x":0.5,"y":0`, `"x":100001,"y":0`, `"x":"0","y":0`} {
		raw := `{"store_id":"` + state.StoreID + `","change_set_id":"` + state.ChangeSetID + `","generation":1,"diagram_id":"` + base.DiagramID + `","component_id":"` + base.ComponentID + `",` + pair + `}`
		var fields map[string]any
		if err := json.Unmarshal([]byte(raw), &fields); err != nil {
			t.Fatal(err)
		}
		response := postAgent(t, f.handler, "/api/agent/v2/diagrams/set-position", fields)
		requireAgentErrorCode(t, response, "invalid_request")
	}
	for _, path := range []string{"/api/agent/v2/diagrams/set-position", "/api/architecture/diagrams/set-position"} {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"x":0,"x":1,"y":0}`))
		request.Header.Set("Origin", testOrigin)
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		f.handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("duplicate placement fields: %s", response.Body.String())
		}
	}
	if f.state.changeSets[state.ChangeSetID].generation != 1 {
		t.Fatal("rejected request changed state")
	}
	response := postAgent(t, f.handler, "/api/agent/v2/diagrams/positions", agentapi.DiagramPositionsRequest{StoreID: state.StoreID, ChangeSetID: state.ChangeSetID, DiagramID: base.DiagramID})
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
}
