package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"workbraid/internal/agentapi"
	"workbraid/internal/web"
)

func decodeCLIEnvelope(t *testing.T, output *bytes.Buffer) agentapi.Envelope {
	t.Helper()
	var envelope agentapi.Envelope
	if err := json.Unmarshal(output.Bytes(), &envelope); err != nil {
		t.Fatalf("decode CLI output: %v\n%s", err, output.String())
	}
	if err := envelope.Validate(); err != nil {
		t.Fatalf("invalid CLI envelope: %v", err)
	}
	return envelope
}

func TestRealBinaryCLIAndTwoMCPBridgesShareOneServerAuthority(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot := t.TempDir()
	binary := filepath.Join(runtimeRoot, "workbraid")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "./cmd/workbraid")
	build.Dir = repositoryRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build real binary: %v\n%s", err, output)
	}
	firstSkill := runRealSkill(t, binary)
	secondSkill := runRealSkill(t, binary)
	if !bytes.Equal(firstSkill, secondSkill) || !bytes.HasPrefix(firstSkill, []byte("# WorkBraid Architecture agent guide\n")) {
		t.Fatal("real --skill output is not deterministic canonical Markdown")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	origin := "http://" + address
	serverLog, err := os.Create(filepath.Join(runtimeRoot, "server.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer serverLog.Close()
	server := exec.Command(binary, "--listen", address, "--data-dir", filepath.Join(runtimeRoot, "data"), "--ui-dir", filepath.Join(runtimeRoot, "ui"))
	server.Stdout, server.Stderr = serverLog, serverLog
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	serverDone := make(chan error, 1)
	serverExited := false
	go func() { serverDone <- server.Wait() }()
	defer func() {
		if serverExited {
			return
		}
		if server.Process != nil {
			_ = server.Process.Kill()
		}
		select {
		case <-serverDone:
		case <-time.After(3 * time.Second):
			t.Error("real WorkBraid server did not stop")
		}
	}()
	waitForAgentServer(t, origin)

	created := runRealCLI(t, binary, origin, "project", "create", "--name", "Bridge authority")
	if !created.OK || created.Context.Project == nil || created.Context.AcceptedRevision == nil {
		t.Fatalf("created = %+v", created)
	}
	listedByCLI := runRealCLI(t, binary, origin, "project", "list")
	listedProjects := listedByCLI.Result.(map[string]any)["projects"].([]any)
	if len(listedProjects) != 1 || listedProjects[0].(map[string]any)["slug"] != created.Context.Project.Slug || listedProjects[0].(map[string]any)["store_id"] != created.Context.Project.StoreID {
		t.Fatalf("advertised real project list = %+v", listedByCLI)
	}
	inspected := runRealCLI(t, binary, origin, "architecture", "inspect")
	projection, ok := inspected.Result.(map[string]any)
	if !ok {
		t.Fatalf("projection = %#v", inspected.Result)
	}
	rootID, _ := projection["root_diagram_id"].(string)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	first := connectRealMCP(t, ctx, binary, origin)
	second := connectRealMCP(t, ctx, binary, origin)
	defer second.Close()
	tools, err := first.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 22 {
		t.Fatalf("real MCP discovery: tools=%d err=%v", len(tools.Tools), err)
	}
	var realRelationshipEdit *mcp.Tool
	for _, tool := range tools.Tools {
		if tool.Name == "relationship_edit" {
			realRelationshipEdit = tool
			break
		}
	}
	if realRelationshipEdit == nil || realRelationshipEdit.Description == "" || realRelationshipEdit.OutputSchema == nil {
		t.Fatalf("real MCP relationship_edit discovery = %+v", realRelationshipEdit)
	}
	realEditInput, _ := realRelationshipEdit.InputSchema.(map[string]any)
	if realEditInput["additionalProperties"] != false || !slices.Contains(schemaStrings(realEditInput["required"]), "old_target_id") || !slices.Contains(schemaStrings(realEditInput["required"]), "occurrence") {
		t.Fatalf("real MCP relationship_edit schema = %#v", realRelationshipEdit.InputSchema)
	}
	createdComponent, err := first.CallTool(ctx, &mcp.CallToolParams{Name: "component_create", Arguments: map[string]any{
		"store_id": created.Context.Project.StoreID, "accepted_revision": *created.Context.AcceptedRevision, "pending_generation": nil,
		"title": "Gateway", "description": "Routes requests.\n", "diagram_id": rootID,
	}})
	if err != nil || createdComponent.IsError {
		t.Fatalf("component_create: result=%+v err=%v", createdComponent, err)
	}
	gateway := mcpEnvelope(t, createdComponent)
	gatewayID := gateway.Result.(map[string]any)["component_id"].(string)
	worker := runRealCLI(t, binary, origin, "component", "create",
		"--store-id", created.Context.Project.StoreID, "--accepted-revision", *created.Context.AcceptedRevision, "--generation", "1",
		"--title", "Worker", "--diagram-id", rootID)
	workerID := worker.Result.(map[string]any)["component_id"].(string)
	if worker.Context.PendingGeneration == nil || *worker.Context.PendingGeneration != 2 {
		t.Fatalf("CLI did not continue MCP generation: %+v", worker)
	}
	invalidCLI := runRealCLI(t, binary, origin, "relationship", "add",
		"--store-id", created.Context.Project.StoreID, "--accepted-revision", *created.Context.AcceptedRevision, "--generation", "2",
		"--source-id", gatewayID, "--target-id", "", "--label", "")
	if invalidCLI.Context.PendingGeneration == nil || *invalidCLI.Context.PendingGeneration != 3 || invalidCLI.Result.(map[string]any)["candidate_valid"] != false {
		t.Fatalf("CLI raw invalid relationship = %+v", invalidCLI)
	}
	changes, err := second.CallTool(ctx, &mcp.CallToolParams{Name: "changes_inspect", Arguments: map[string]any{}})
	if err != nil || changes.IsError {
		t.Fatalf("changes_inspect: result=%+v err=%v", changes, err)
	}
	structured, ok := changes.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("structured changes = %#v", changes.StructuredContent)
	}
	contextValue, _ := structured["context"].(map[string]any)
	if contextValue["pending_generation"] != float64(3) {
		t.Fatalf("second bridge pending context = %#v", contextValue)
	}
	changeEnvelope := mcpEnvelope(t, changes)
	components := changeEnvelope.Result.(map[string]any)["components"].([]any)
	rows := components[0].(map[string]any)["relationships"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["target_id"] != "" || rows[0].(map[string]any)["label"] != "" {
		t.Fatalf("CLI raw row lost through MCP inspect: %#v", rows)
	}
	repaired, err := first.CallTool(ctx, &mcp.CallToolParams{Name: "relationship_edit", Arguments: map[string]any{
		"store_id": created.Context.Project.StoreID, "accepted_revision": *created.Context.AcceptedRevision, "pending_generation": 3,
		"source_id": gatewayID, "old_target_id": "", "old_label": "", "occurrence": 1, "target_id": workerID, "label": "calls",
	}})
	if err != nil || repaired.IsError || mcpEnvelope(t, repaired).Context.PendingGeneration == nil || *mcpEnvelope(t, repaired).Context.PendingGeneration != 4 {
		t.Fatalf("MCP exact empty repair: result=%+v err=%v", repaired, err)
	}
	malformed, err := second.CallTool(ctx, &mcp.CallToolParams{Name: "relationship_add", Arguments: map[string]any{
		"store_id": created.Context.Project.StoreID, "accepted_revision": *created.Context.AcceptedRevision, "pending_generation": 4,
		"source_id": gatewayID, "target_id": "not-a-component-id", "label": "   ",
	}})
	if err != nil || malformed.IsError || mcpEnvelope(t, malformed).Context.PendingGeneration == nil || *mcpEnvelope(t, malformed).Context.PendingGeneration != 5 {
		t.Fatalf("MCP malformed raw add: result=%+v err=%v", malformed, err)
	}
	removed := runRealCLI(t, binary, origin, "relationship", "remove",
		"--store-id", created.Context.Project.StoreID, "--accepted-revision", *created.Context.AcceptedRevision, "--generation", "5",
		"--source-id", gatewayID, "--target-id", "not-a-component-id", "--label", "   ", "--occurrence", "1")
	if removed.Context.PendingGeneration == nil || *removed.Context.PendingGeneration != 6 {
		t.Fatalf("CLI exact malformed removal = %+v", removed)
	}
	staleCLI := runRealCLIResult(t, binary, origin, "component", "edit",
		"--store-id", created.Context.Project.StoreID, "--accepted-revision", *created.Context.AcceptedRevision, "--generation", "5",
		"--component-id", gatewayID, "--title", "Stale")
	if staleCLI.OK || staleCLI.Error == nil || staleCLI.Error.Code != "pending_generation_mismatch" {
		t.Fatalf("CLI typed stale failure = %+v", staleCLI)
	}
	missingMCP, err := first.CallTool(ctx, &mcp.CallToolParams{Name: "relationship_remove", Arguments: map[string]any{
		"store_id": created.Context.Project.StoreID, "accepted_revision": *created.Context.AcceptedRevision, "pending_generation": 6,
		"source_id": gatewayID, "target_id": "missing", "label": "missing", "occurrence": 1,
	}})
	if err != nil || !missingMCP.IsError || mcpErrorCode(missingMCP.StructuredContent) != "target_not_found" {
		t.Fatalf("MCP typed target failure: result=%+v err=%v", missingMCP, err)
	}

	// First deliberate acceptance crosses from MCP Review to CLI Update.
	mcpReviewed, err := first.CallTool(ctx, &mcp.CallToolParams{Name: "changes_review", Arguments: map[string]any{
		"store_id": created.Context.Project.StoreID, "accepted_revision": *created.Context.AcceptedRevision,
		"pending_generation": 6, "generation": 6,
	}})
	if err != nil || mcpReviewed.IsError {
		t.Fatalf("MCP Review before CLI Update: result=%+v err=%v", mcpReviewed, err)
	}
	firstReview := mcpEnvelope(t, mcpReviewed).Result.(map[string]any)
	if firstReview["base_revision"] != *created.Context.AcceptedRevision || firstReview["candidate_tree"] == "" || firstReview["generation"] != float64(6) {
		t.Fatalf("MCP review binding = %#v", firstReview)
	}
	firstUpdated := runRealCLI(t, binary, origin, "architecture", "update",
		"--store-id", created.Context.Project.StoreID, "--base-revision", firstReview["base_revision"].(string),
		"--candidate-tree", firstReview["candidate_tree"].(string), "--generation", "6")
	if !firstUpdated.OK || firstUpdated.Context.AcceptedRevision == nil {
		t.Fatalf("CLI Update from MCP Review = %+v", firstUpdated)
	}
	firstAccepted := *firstUpdated.Context.AcceptedRevision
	firstUpdateResult := firstUpdated.Result.(map[string]any)
	if firstUpdated.Context.PendingGeneration != nil || firstAccepted == *created.Context.AcceptedRevision ||
		firstUpdateResult["base_revision"] != firstReview["base_revision"] || firstUpdateResult["candidate_tree"] != firstReview["candidate_tree"] ||
		firstUpdateResult["generation"] != firstReview["generation"] || firstUpdateResult["accepted_revision"] != firstAccepted {
		t.Fatalf("CLI Update did not consume exact MCP binding: %+v", firstUpdated)
	}
	consumed, err := second.CallTool(ctx, &mcp.CallToolParams{Name: "changes_inspect", Arguments: map[string]any{}})
	if err != nil || consumed.IsError {
		t.Fatalf("inspect after first acceptance: result=%+v err=%v", consumed, err)
	}
	consumedEnvelope := mcpEnvelope(t, consumed)
	if consumedEnvelope.Context.PendingGeneration != nil || consumedEnvelope.Context.AcceptedRevision == nil || *consumedEnvelope.Context.AcceptedRevision != firstAccepted || consumedEnvelope.Result.(map[string]any)["changes"] != nil {
		t.Fatalf("first acceptance was not authoritative/consumed: %+v", consumedEnvelope)
	}

	// Second cycle crosses from CLI Review to MCP Update. A newer review makes
	// the old exact binding observably invalid before the final acceptance.
	firstPending := runRealCLI(t, binary, origin, "component", "edit",
		"--store-id", created.Context.Project.StoreID, "--accepted-revision", firstAccepted, "--generation", "none",
		"--component-id", gatewayID, "--title", "Gateway two")
	if firstPending.Context.PendingGeneration == nil || *firstPending.Context.PendingGeneration != 1 {
		t.Fatalf("second-cycle pending edit: %+v", firstPending)
	}
	oldCLIReview := runRealCLI(t, binary, origin, "changes", "review",
		"--store-id", created.Context.Project.StoreID, "--accepted-revision", firstAccepted, "--generation", "1")
	oldBinding := oldCLIReview.Result.(map[string]any)
	if oldBinding["base_revision"] != firstAccepted || oldBinding["candidate_tree"] == "" || oldBinding["generation"] != float64(1) {
		t.Fatalf("first CLI review binding = %#v", oldBinding)
	}
	newerPending := runRealCLI(t, binary, origin, "component", "edit",
		"--store-id", created.Context.Project.StoreID, "--accepted-revision", firstAccepted, "--generation", "1",
		"--component-id", gatewayID, "--title", "Gateway three")
	if newerPending.Context.PendingGeneration == nil || *newerPending.Context.PendingGeneration != 2 {
		t.Fatalf("review-invalidating edit: %+v", newerPending)
	}
	freshCLIReview := runRealCLI(t, binary, origin, "changes", "review",
		"--store-id", created.Context.Project.StoreID, "--accepted-revision", firstAccepted, "--generation", "2")
	freshBinding := freshCLIReview.Result.(map[string]any)
	if freshBinding["base_revision"] != firstAccepted || freshBinding["candidate_tree"] == "" || freshBinding["generation"] != float64(2) {
		t.Fatalf("fresh CLI review binding = %#v", freshBinding)
	}
	invalidated, err := first.CallTool(ctx, &mcp.CallToolParams{Name: "architecture_update", Arguments: map[string]any{
		"store_id": created.Context.Project.StoreID, "base_revision": oldBinding["base_revision"],
		"candidate_tree": oldBinding["candidate_tree"], "generation": oldBinding["generation"],
	}})
	if err != nil || !invalidated.IsError || mcpErrorCode(invalidated.StructuredContent) != "review_invalidated" {
		t.Fatalf("invalidated binding failure: result=%+v err=%v", invalidated, err)
	}
	secondUpdated, err := first.CallTool(ctx, &mcp.CallToolParams{Name: "architecture_update", Arguments: map[string]any{
		"store_id": created.Context.Project.StoreID, "base_revision": freshBinding["base_revision"],
		"candidate_tree": freshBinding["candidate_tree"], "generation": freshBinding["generation"],
	}})
	if err != nil || secondUpdated.IsError {
		t.Fatalf("MCP Update from CLI Review: result=%+v err=%v", secondUpdated, err)
	}
	secondEnvelope := mcpEnvelope(t, secondUpdated)
	secondResult := secondEnvelope.Result.(map[string]any)
	if secondEnvelope.Context.PendingGeneration != nil || secondEnvelope.Context.AcceptedRevision == nil || *secondEnvelope.Context.AcceptedRevision == firstAccepted ||
		secondResult["base_revision"] != freshBinding["base_revision"] || secondResult["candidate_tree"] != freshBinding["candidate_tree"] ||
		secondResult["generation"] != freshBinding["generation"] || secondResult["accepted_revision"] != *secondEnvelope.Context.AcceptedRevision {
		t.Fatalf("MCP Update did not consume exact CLI binding: %+v", secondEnvelope)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close first bridge: %v", err)
	}
	stillThere, err := second.CallTool(ctx, &mcp.CallToolParams{Name: "changes_inspect", Arguments: map[string]any{}})
	if err != nil || stillThere.IsError {
		t.Fatalf("state after first bridge exit: result=%+v err=%v", stillThere, err)
	}

	if err := server.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-serverDone:
		serverExited = true
	case <-time.After(3 * time.Second):
		t.Fatal("real WorkBraid server did not exit")
	}
	withoutAuthority, err := second.CallTool(ctx, &mcp.CallToolParams{Name: "status", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("MCP bridge should return a typed connection result: %v", err)
	}
	if !withoutAuthority.IsError || mcpErrorCode(withoutAuthority.StructuredContent) != "connection_failed" {
		t.Fatalf("bridge without server = %+v", withoutAuthority)
	}
}

func runRealSkill(t *testing.T, binary string) []byte {
	t.Helper()
	command := exec.Command(binary, "--skill")
	command.Env = append(os.Environ(), "WORKBRAID_DATA_DIR=/definitely/unusable/workbraid-agent-access")
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil || stderr.Len() != 0 {
		t.Fatalf("real --skill: err=%v stderr=%q", err, stderr.String())
	}
	return output
}

func runRealCLI(t *testing.T, binary, origin string, arguments ...string) agentapi.Envelope {
	t.Helper()
	command := exec.Command(binary, append([]string{"--server", origin, "--json"}, arguments...)...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		t.Fatalf("real CLI %v: %v, stderr=%s", arguments, err, stderr.String())
	}
	var envelope agentapi.Envelope
	if err := json.Unmarshal(output, &envelope); err != nil {
		t.Fatalf("decode real CLI: %v\n%s", err, output)
	}
	return envelope
}

func runRealCLIResult(t *testing.T, binary, origin string, arguments ...string) agentapi.Envelope {
	t.Helper()
	command := exec.Command(binary, append([]string{"--server", origin, "--json"}, arguments...)...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	if err == nil {
		t.Fatalf("real CLI %v unexpectedly succeeded: %s", arguments, stdout.String())
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) || stderr.Len() != 0 {
		t.Fatalf("real CLI %v: %v, stderr=%s", arguments, err, stderr.String())
	}
	return decodeCLIEnvelope(t, &stdout)
}

func mcpEnvelope(t *testing.T, result *mcp.CallToolResult) agentapi.Envelope {
	t.Helper()
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var envelope agentapi.Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("decode MCP envelope: %v\n%s", err, data)
	}
	if err := envelope.Validate(); err != nil {
		t.Fatalf("invalid MCP envelope: %v\n%s", err, data)
	}
	return envelope
}

func connectRealMCP(t *testing.T, ctx context.Context, binary, origin string) *mcp.ClientSession {
	t.Helper()
	command := exec.Command(binary, "--server", origin, "mcp")
	var stderr bytes.Buffer
	command.Stderr = &stderr
	client := mcp.NewClient(&mcp.Implementation{Name: "workbraid-integration", Version: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: command, TerminateDuration: 2 * time.Second}, nil)
	if err != nil {
		t.Fatalf("connect real MCP bridge: %v, stderr=%s", err, stderr.String())
	}
	return session
}

func mcpErrorCode(value any) string {
	structured, _ := value.(map[string]any)
	errorValue, _ := structured["error"].(map[string]any)
	code, _ := errorValue["code"].(string)
	return code
}

func waitForAgentServer(t *testing.T, origin string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Get(origin + "/api/agent/v1/status")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("real WorkBraid server did not become ready")
}

func TestSkillIsStandaloneDeterministicMarkdown(t *testing.T) {
	var first, second, errors bytes.Buffer
	if code := run([]string{"--skill"}, &first, &errors, bytes.NewReader(nil)); code != 0 {
		t.Fatalf("first --skill exit = %d", code)
	}
	if code := run([]string{"--skill"}, &second, &errors, bytes.NewReader(nil)); code != 0 {
		t.Fatalf("second --skill exit = %d", code)
	}
	if errors.Len() != 0 || first.String() != second.String() || !bytes.HasPrefix(first.Bytes(), []byte("# WorkBraid Architecture agent guide\n")) {
		t.Fatalf("skill output is not clean/deterministic Markdown: stderr=%q", errors.String())
	}
	for _, command := range []string{"project list", "project create", "architecture inspect", "changes review", "architecture update", "relationship edit", "diagram stop-showing-component", "workbraid [--server <loopback-url>] mcp"} {
		if !bytes.Contains(first.Bytes(), []byte(command)) {
			t.Fatalf("skill does not document %q", command)
		}
	}
	for _, guidance := range []string{
		"The CLI does not start WorkBraid.",
		"Copy `store_id`, `accepted_revision` or `revision`, `pending_generation`, and every created Component or Diagram ID",
		"Architecture was already accepted; do not run Update again; Refresh or reopen to load the accepted result.",
	} {
		if !strings.Contains(first.String(), guidance) {
			t.Fatalf("skill missing public guidance %q", guidance)
		}
	}
	for _, internal := range []string{"never retry Update, Refresh", "failure is not a candidate validation result", "--listen", "--data-dir"} {
		if strings.Contains(first.String(), internal) {
			t.Fatalf("skill retains internal or incorrect guidance %q", internal)
		}
	}
}

func TestClientModesRejectServerAuthorityFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--json", "--data-dir", t.TempDir(), "status"}, &stdout, &stderr, bytes.NewReader(nil)); code == 0 {
		t.Fatal("client mode accepted --data-dir")
	}
	envelope := decodeCLIEnvelope(t, &stdout)
	if envelope.Error == nil || envelope.Error.Code != "invalid_request" {
		t.Fatalf("envelope = %+v", envelope)
	}
}

func TestCanonicalHelpDocumentsExactActionsAndTopLevelAliasesStayUnavailable(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--help"}, &stdout, &stderr, bytes.NewReader(nil)); code != 0 || stderr.Len() != 0 {
		t.Fatalf("--help exit=%d stderr=%q", code, stderr.String())
	}
	for _, exact := range []string{
		"workbraid [--server <loopback-url>] mcp",
		"Client commands always print one JSON envelope; --json makes it compact.",
		"--server or WORKBRAID_SERVER selects the running WorkBraid URL.",
		"The default is http://127.0.0.1:8080.",
		"Connect, choose a project, and inspect:",
		"project list | project current",
		"--store-id <uuid> --accepted-revision <sha> --generation <n|none>",
		"relationship edit <state> --source-id <uuid> --old-target-id <raw>",
		"architecture update --store-id <uuid> --base-revision <sha> --candidate-tree <tree> --generation <n>",
	} {
		if !strings.Contains(stdout.String(), exact) {
			t.Fatalf("--help missing %q\n%s", exact, stdout.String())
		}
	}
	for _, obsolete := range []string{"architecture_inspect", "changes_inspect"} {
		stdout.Reset()
		if code := run([]string{"--json", obsolete}, &stdout, &stderr, bytes.NewReader(nil)); code == 0 {
			t.Fatalf("obsolete alias %q succeeded", obsolete)
		}
		envelope := decodeCLIEnvelope(t, &stdout)
		if envelope.Error == nil || envelope.Error.Code != "invalid_request" {
			t.Fatalf("obsolete alias %q = %+v", obsolete, envelope)
		}
	}
}

func TestCLIUsesRunningLoopbackAuthority(t *testing.T) {
	server := httptest.NewServer(web.NewHandler("http://127.0.0.1", t.TempDir(), t.TempDir()))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--server", server.URL, "--json", "project", "create", "--name", "CLI project"}, &stdout, &stderr, bytes.NewReader(nil)); code != 0 {
		t.Fatalf("project create exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	created := decodeCLIEnvelope(t, &stdout)
	if !created.OK || created.Context.Project == nil || created.Context.Project.Name != "CLI project" {
		t.Fatalf("created = %+v", created)
	}
	stdout.Reset()
	if code := run([]string{"--server", server.URL, "--json", "project", "list"}, &stdout, &stderr, bytes.NewReader(nil)); code != 0 {
		t.Fatalf("project list exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	listed := decodeCLIEnvelope(t, &stdout)
	projects := listed.Result.(map[string]any)["projects"].([]any)
	if len(projects) != 1 || projects[0].(map[string]any)["slug"] != created.Context.Project.Slug {
		t.Fatalf("project list = %+v", listed)
	}
	stdout.Reset()
	if code := run([]string{"--server", server.URL, "--json", "architecture", "inspect"}, &stdout, &stderr, bytes.NewReader(nil)); code != 0 {
		t.Fatalf("inspect exit = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	inspected := decodeCLIEnvelope(t, &stdout)
	if !inspected.OK || inspected.Context.AcceptedRevision == nil || inspected.Context.PendingGeneration != nil {
		t.Fatalf("inspect = %+v", inspected)
	}
}

func TestMCPDiscoverySchemasAndStructuredStatus(t *testing.T) {
	httpServer := httptest.NewServer(web.NewHandler("http://127.0.0.1", t.TempDir(), t.TempDir()))
	defer httpServer.Close()
	loopbackClient, failure := agentapi.NewClient(httpServer.URL)
	if failure != nil {
		t.Fatalf("new loopback client: %+v", failure)
	}
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "workbraid", Version: "test"}, &mcp.ServerOptions{Instructions: mcpInstructions})
	registerMCPTools(mcpServer, loopbackClient)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	serverDone := make(chan error, 1)
	go func() { serverDone <- mcpServer.Run(ctx, serverTransport) }()
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "workbraid-test", Version: "test"}, nil)
	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if got := session.InitializeResult().ProtocolVersion; got != "2026-07-28" {
		t.Fatalf("negotiated protocol = %q", got)
	}
	instructions := session.InitializeResult().Instructions
	for _, exact := range []string{"one current Architecture project", "one pending change set", "Inspect exact IDs", "base_revision", "candidate_tree", "generation"} {
		if !strings.Contains(instructions, exact) {
			t.Fatalf("MCP instructions missing %q: %q", exact, instructions)
		}
	}
	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantNames := []string{
		"architecture_inspect", "architecture_refresh", "architecture_update", "changes_discard", "changes_inspect", "changes_review",
		"component_create", "component_edit", "component_move_home", "diagram_create_detail", "diagram_edit_title", "diagram_show_component",
		"diagram_stop_showing_component", "project_close", "project_create", "project_current", "project_open", "projects_list",
		"relationship_add", "relationship_edit", "relationship_remove", "status",
	}
	gotNames := make([]string, len(listed.Tools))
	var relationshipEdit *mcp.Tool
	for index, tool := range listed.Tools {
		gotNames[index] = tool.Name
		input, ok := tool.InputSchema.(map[string]any)
		if !ok || input["type"] != "object" || input["additionalProperties"] != false || tool.OutputSchema == nil || tool.Description == "" || tool.Annotations == nil {
			t.Fatalf("tool %q schema/description/annotations incomplete: input=%#v output=%#v", tool.Name, tool.InputSchema, tool.OutputSchema)
		}
		if tool.Name == "relationship_edit" {
			relationshipEdit = tool
		}
		for _, internal := range []string{"private-store-derived", "accepted authority state", "complete current candidate", "candidate validation", "derived boundary", "automatic rebuild", "binding"} {
			if strings.Contains(strings.ToLower(tool.Description), internal) {
				t.Fatalf("tool %q retains internal wording %q: %q", tool.Name, internal, tool.Description)
			}
		}
	}
	slices.Sort(gotNames)
	if !slices.Equal(gotNames, wantNames) {
		t.Fatalf("tool names = %v, want %v", gotNames, wantNames)
	}
	if relationshipEdit == nil {
		t.Fatal("relationship_edit schema missing")
	}
	var updateDescription, reviewDescription string
	for _, tool := range listed.Tools {
		switch tool.Name {
		case "architecture_update":
			updateDescription = tool.Description
		case "changes_review":
			reviewDescription = tool.Description
		}
	}
	for _, description := range []string{updateDescription, reviewDescription} {
		for _, field := range []string{"base_revision", "candidate_tree", "generation"} {
			if !strings.Contains(description, field) {
				t.Fatalf("Review/Update description missing %s: %q", field, description)
			}
		}
	}
	input := relationshipEdit.InputSchema.(map[string]any)
	properties, _ := input["properties"].(map[string]any)
	for _, field := range []string{"store_id", "accepted_revision", "pending_generation", "source_id", "old_target_id", "old_label", "occurrence", "target_id", "label"} {
		property, _ := properties[field].(map[string]any)
		if property["description"] == "" {
			t.Fatalf("relationship_edit property %q lacks exact contract: %#v", field, property)
		}
	}
	required := schemaStrings(input["required"])
	for _, field := range []string{"store_id", "accepted_revision", "pending_generation", "source_id", "old_target_id", "old_label", "occurrence", "target_id", "label"} {
		if !slices.Contains(required, field) {
			t.Fatalf("relationship_edit required fields = %v; missing %q", required, field)
		}
	}
	output, ok := relationshipEdit.OutputSchema.(map[string]any)
	if !ok || output["type"] != "object" || output["additionalProperties"] != false {
		t.Fatalf("relationship_edit output schema is not a closed envelope: %#v", relationshipEdit.OutputSchema)
	}
	called, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "status", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	if called.IsError || called.StructuredContent == nil || len(called.Content) != 1 {
		t.Fatalf("status result = %+v", called)
	}
	text, ok := called.Content[0].(*mcp.TextContent)
	if !ok || !json.Valid([]byte(text.Text)) {
		t.Fatalf("status compatibility content = %#v", called.Content)
	}
	_ = session.Close()
	cancel()
	select {
	case err := <-serverDone:
		if err != nil && err != context.Canceled && err != io.EOF {
			t.Fatalf("MCP server shutdown: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("MCP server did not stop")
	}
}

func schemaStrings(value any) []string {
	switch values := value.(type) {
	case []string:
		return values
	case []any:
		result := make([]string, 0, len(values))
		for _, value := range values {
			if text, ok := value.(string); ok {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}
