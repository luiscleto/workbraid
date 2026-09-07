package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"workbraid/internal/agentapi"
	"workbraid/internal/architecture"
)

// Author a normal project first; install an explicit supported external legacy
// snapshot to exercise the loader and public mutation boundary, not a capability.
func sizingLegacyFixture(t *testing.T, version int) nativeRefreshFixture {
	t.Helper()
	f := newNativeRefreshFixture(t, false)
	parent := f.base.Revision
	manifest := strings.Replace(git(t, "--git-dir", f.storePath, "show", parent+":architecture.yaml"), "version: 6", fmt.Sprintf("version: %d", version), 1) + "\n"
	diagram := fmt.Sprintf("id: %s\ntitle: Legacy\nappearances:\n  - component: %s\n    role: home\n", f.base.RootDiagramID, f.component)
	if version >= 3 {
		diagram += fmt.Sprintf("positions:\n  - component: %s\n    x: 123\n    y: -456\n", f.component)
	}
	if version == 4 {
		diagram += fmt.Sprintf("sizes:\n  - component: %s\n    width: 200\n    height: 96\n", f.component)
	}
	blob := gitInput(t, []byte(diagram), "--git-dir", f.storePath, "hash-object", "-w", "--stdin")
	diagrams := gitInput(t, []byte("100644 blob "+blob+"\troot.yaml\n"), "--git-dir", f.storePath, "mktree")
	manifestBlob := gitInput(t, []byte(manifest), "--git-dir", f.storePath, "hash-object", "-w", "--stdin")
	components := git(t, "--git-dir", f.storePath, "rev-parse", parent+":components")
	tree := gitInput(t, []byte("100644 blob "+manifestBlob+"\tarchitecture.yaml\n040000 tree "+components+"\tcomponents\n040000 tree "+diagrams+"\tdiagrams\n"), "--git-dir", f.storePath, "mktree")
	commit := gitInput(t, []byte("Explicit legacy fixture\n"), "-c", "user.name=Test", "-c", "user.email=test@workbraid.invalid", "--git-dir", f.storePath, "commit-tree", tree, "-p", parent)
	git(t, "--git-dir", f.storePath, "update-ref", "refs/heads/accepted", commit, parent)
	f.base = decodeArchitectureResponse(t, postJSONRequest(t, f.handler, "/api/projects/open", map[string]any{"project_slug": f.base.ProjectSlug}))
	return f
}
func TestSizingPublicNoopBeforeLegacyUpgradeAndRestart(t *testing.T) {
	for _, version := range []int{2, 3, 4} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			f := sizingLegacyFixture(t, version)
			state := changeSetState(t, createAgentChangeSet(t, f.handler, f.base.StoreID, f.base.Revision, "Size"))
			request := agentapi.DiagramSetSizeRequest{DiagramRestoreDefaultSizeRequest: agentapi.DiagramRestoreDefaultSizeRequest{DiagramAutoLayoutRequest: agentapi.DiagramAutoLayoutRequest{StatePreconditions: state, DiagramID: f.base.RootDiagramID}, ComponentID: f.component}, Width: 116, Height: 54}
			if version == 4 {
				request.Width, request.Height = 200, 96
			}
			review := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: state}))
			if !review.OK {
				t.Fatal(review.Error)
			}
			before := f.state.changeSets[state.ChangeSetID]
			object, tree := before.refObject, before.candidate.Tree()
			unchanged := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/diagrams/set-size", request))
			if !unchanged.OK || resultMap(t, unchanged)["unchanged"] != true {
				t.Fatalf("no-op %+v", unchanged)
			}
			if after := f.state.changeSets[state.ChangeSetID]; after.refObject != object || after.candidate.Tree() != tree || after.generation != 0 || after.review == nil || after.candidate.Snapshot().FormatVersion() != version {
				t.Fatal("same-size request upgraded or invalidated review")
			}
			wrong := request
			wrong.Generation = 1
			requireAgentErrorCode(t, postAgent(t, f.handler, "/api/agent/v2/diagrams/set-size", wrong), "change_set_generation_mismatch")
			request.Width, request.Height = 333, 177
			kept := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/diagrams/set-size", request))
			if !kept.OK {
				t.Fatal(kept.Error)
			}
			after := f.state.changeSets[state.ChangeSetID]
			if after.generation != 1 || after.review != nil || after.candidate.Snapshot().FormatVersion() != 4 {
				t.Fatal("not one actual sizing upgrade")
			}
			posBefore := f.base.Diagrams[0].Appearances[0].DisplayPosition
			posAfter, _ := after.candidate.Snapshot().NodePosition(f.base.RootDiagramID, f.component)
			if !reflect.DeepEqual(posBefore, posAfter) {
				t.Fatal("sizing moved center")
			}
			_, _, comparison := captureReviewPresentation(after.baseSnapshot, after.candidate.Snapshot())
			if len(comparison.NodeSizes) != 1 || len(comparison.Components) != 0 || len(comparison.Relationships) != 0 {
				t.Fatalf("size review %+v", comparison)
			}
			data := filepath.Dir(filepath.Dir(f.storePath))
			restarted, handler := newHandler(testOrigin, t.TempDir(), data)
			postJSONRequest(t, handler, "/api/projects/open", map[string]any{"project_slug": f.base.ProjectSlug})
			loaded := restarted.changeSets[state.ChangeSetID]
			if loaded == nil || loaded.refObject != after.refObject || loaded.candidate.Tree() != after.candidate.Tree() {
				t.Fatal("restart lost exact sizing")
			}
		})
	}
}
func TestSizingAcceptedNoopAndClosedRequests(t *testing.T) {
	f := sizingLegacyFixture(t, 3)
	w, h := 116, 54
	payload := diagramMutationRequest{ProjectSlug: f.base.ProjectSlug, StoreID: f.base.StoreID, ExpectedRevision: f.base.Revision, PendingGenerationObserved: true, DiagramID: f.base.RootDiagramID, ComponentID: f.component, Width: &w, Height: &h}
	response := postJSONRequest(t, f.handler, "/api/architecture/diagrams/set-size", payload)
	if response.Code != 200 || len(f.state.changeSets) != 0 {
		t.Fatal("legacy accepted no-op created proposal")
	}
	for _, raw := range []string{`{"width":80,"width":90,"height":48}`, `{"width":null,"height":48}`, `{"width":80.5,"height":48}`, `{"width":80,"height":47}`, `{"width":80}`, `{"width":80,"height":48,"x":0}`} {
		for _, path := range []string{"/api/architecture/diagrams/set-size", "/api/agent/v2/diagrams/set-size"} {
			r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(raw))
			r.Header.Set("Origin", testOrigin)
			r.Header.Set("Content-Type", "application/json")
			out := httptest.NewRecorder()
			f.handler.ServeHTTP(out, r)
			if out.Code != 400 {
				t.Fatalf("accepted invalid %s %s: %s", path, raw, out.Body.String())
			}
		}
	}
	// Restore default is a concrete actual write on a legacy node.
	payload.Width, payload.Height = nil, nil
	restored := decodeArchitectureResponse(t, postJSONRequest(t, f.handler, "/api/architecture/diagrams/restore-default-size", payload))
	if restored.Changes == nil || restored.Changes.Generation != 1 {
		t.Fatal("restore not one write")
	}
	size, _ := f.state.changeSets[restored.Changes.ID].candidate.Snapshot().NodeSize(f.base.RootDiagramID, f.component)
	if size == nil || *size != (architecture.Size{Width: 200, Height: 96}) {
		t.Fatal("restore not native default")
	}
}
