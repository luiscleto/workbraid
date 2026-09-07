package web

import (
	"path/filepath"
	"strings"
	"testing"

	"workbraid/internal/agentapi"
	"workbraid/internal/architecture"
)

func TestVersionAgentReadsWithoutCurrentProjectOrAcceptanceBinding(t *testing.T) {
	f := newNativeRefreshFixture(t, true)
	active := testActiveChangeSet(f.state)
	input := agentapi.ArchitectureCompareRequest{StoreID: f.base.StoreID, Before: architecture.VersionSelector{Kind: "accepted", Revision: f.base.Revision}, After: architecture.VersionSelector{Kind: "proposal", ChangeSetID: active.id, State: active.refObject, Side: "candidate"}}
	refs := git(t, "--git-dir", f.storePath, "show-ref")
	state, handler := newHandler(testOrigin, testUI(t), filepath.Dir(filepath.Dir(f.storePath)))
	result := resultMap(t, decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/architecture/compare", input)))
	if _, ok := result["review"]; ok {
		t.Fatal("comparison masquerades as Review")
	}
	for _, field := range []string{"reviewed_state", "candidate_tree", "generation", "base_revision"} {
		if _, ok := result[field]; ok {
			t.Fatalf("comparison exposes acceptance binding field %s", field)
		}
	}
	if result["diff"] == "" || result["before"] == nil || result["after"] == nil || !strings.Contains(result["report_url"].(string), "/compare/report?") {
		t.Fatal(result)
	}
	page := resultMap(t, decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/architecture/versions", architecture.VersionPageRequest{StoreID: f.base.StoreID, Source: "proposal", Limit: 1})))
	if len(page["versions"].([]any)) != 2 {
		t.Fatal(page)
	}
	if state.loadedProject != nil || state.loadedSnapshot != nil || git(t, "--git-dir", f.storePath, "show-ref") != refs {
		t.Fatal("comparison selected project or wrote refs")
	}
	input.After.State = strings.Repeat("1", 40)
	requireAgentErrorCode(t, postAgent(t, handler, "/api/agent/v2/architecture/compare", input), "version_moved")
}
