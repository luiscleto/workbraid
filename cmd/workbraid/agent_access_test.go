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
	"reflect"
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

func runRealSkill(t *testing.T, binary string) []byte {
	t.Helper()
	output, err := exec.Command(binary, "--skill").Output()
	if err != nil {
		t.Fatal(err)
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

func runRealCLIError(t *testing.T, binary, origin string, arguments ...string) agentapi.Envelope {
	t.Helper()
	command := exec.Command(binary, append([]string{"--server", origin, "--json"}, arguments...)...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) || stderr.Len() != 0 {
		t.Fatalf("real CLI %v: err=%v stderr=%s stdout=%s", arguments, err, stderr.String(), stdout.String())
	}
	return decodeCLIEnvelope(t, &stdout)
}

func mcpEnvelope(t *testing.T, result *mcp.CallToolResult) agentapi.Envelope {
	t.Helper()
	if len(result.Content) != 1 {
		t.Fatalf("MCP content count=%d want=1", len(result.Content))
	}
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("MCP content type=%T want text", result.Content[0])
	}
	var textEnvelope agentapi.Envelope
	if err := json.Unmarshal([]byte(textContent.Text), &textEnvelope); err != nil {
		t.Fatalf("decode MCP text envelope: %v\n%s", err, textContent.Text)
	}
	if err := textEnvelope.Validate(); err != nil {
		t.Fatalf("invalid MCP text envelope: %v\n%s", err, textContent.Text)
	}
	if result.StructuredContent == nil {
		return textEnvelope
	}
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
	if !reflect.DeepEqual(textEnvelope, envelope) {
		t.Fatalf("MCP text/structured mismatch:\ntext=%+v\nstructured=%+v", textEnvelope, envelope)
	}
	return envelope
}

func runRealMCP(t *testing.T, ctx context.Context, session *mcp.ClientSession, name string, arguments map[string]any) agentapi.Envelope {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatalf("MCP %s: result=%+v err=%v", name, result, err)
	}
	envelope := mcpEnvelope(t, result)
	if result.IsError {
		t.Fatalf("MCP %s: %+v", name, envelope)
	}
	return envelope
}

func runRealMCPError(t *testing.T, ctx context.Context, session *mcp.ClientSession, name string, arguments map[string]any) agentapi.Envelope {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatalf("MCP %s error: result=%+v err=%v", name, result, err)
	}
	envelope := mcpEnvelope(t, result)
	if !result.IsError {
		t.Fatalf("MCP %s expected error: %+v", name, envelope)
	}
	return envelope
}

func runBareGit(t *testing.T, storePath string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"--git-dir", storePath}, arguments...)...)
	command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
	return strings.TrimSpace(string(output))
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

func waitForAgentServer(t *testing.T, origin string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Get(origin + "/api/agent/v2/status")
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

func TestRealBinaryCLIAndMCPShareParallelDurableChangeSets(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot := t.TempDir()
	binary := filepath.Join(runtimeRoot, "workbraid")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "./cmd/workbraid")
	build.Dir = repositoryRoot
	if output, buildErr := build.CombinedOutput(); buildErr != nil {
		t.Fatalf("build real binary: %v\n%s", buildErr, output)
	}
	if skill := runRealSkill(t, binary); !bytes.Contains(skill, []byte("durable, named, independent change sets")) {
		t.Fatalf("real skill is not v2: %s", skill)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	origin := "http://" + address
	dataDirectory := filepath.Join(runtimeRoot, "data")
	serverLog, err := os.Create(filepath.Join(runtimeRoot, "server.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer serverLog.Close()
	server := exec.Command(binary, "--listen", address, "--data-dir", dataDirectory, "--ui-dir", filepath.Join(runtimeRoot, "ui"))
	server.Stdout, server.Stderr = serverLog, serverLog
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.Wait() }()
	defer func() {
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
	storeID, revision := created.Context.Project.StoreID, *created.Context.AcceptedRevision
	accepted := runRealCLI(t, binary, origin, "architecture", "inspect")
	rootID := accepted.Result.(map[string]any)["root_diagram_id"].(string)

	changeA := runRealCLI(t, binary, origin, "change-set", "create", "--store-id", storeID, "--accepted-revision", revision, "--name", "CLI proposal")
	changeAMap := changeA.Result.(map[string]any)
	idA := changeAMap["id"].(string)
	proposalA := "# CLI proposal\n\nExact λ markdown.\n"
	proposalFile := filepath.Join(runtimeRoot, "proposal.md")
	if err := os.WriteFile(proposalFile, []byte(proposalA), 0o600); err != nil {
		t.Fatal(err)
	}
	editedA := runRealCLI(t, binary, origin, "change-set", "edit-proposal", "--store-id", storeID, "--change-set-id", idA, "--generation", "0", "--proposal-file", proposalFile)
	if editedA.Result.(map[string]any)["proposal_markdown"] != proposalA {
		t.Fatalf("CLI proposal lost bytes: %#v", editedA.Result)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	session := connectRealMCP(t, ctx, binary, origin)
	defer session.Close()
	tools, err := session.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 51 {
		t.Fatalf("real MCP discovery: tools=%d err=%v", len(tools.Tools), err)
	}
	if status := runRealMCP(t, ctx, session, "status", map[string]any{}); status.Result.(map[string]any)["protocol"] != agentapi.Protocol {
		t.Fatalf("MCP protocol: %+v", status)
	}
	runRealMCP(t, ctx, session, "projects_list", map[string]any{})
	runRealMCP(t, ctx, session, "project_current", map[string]any{})
	runRealMCP(t, ctx, session, "architecture_inspect", map[string]any{})
	versions := runRealCLI(t, binary, origin, "architecture", "versions", "--store-id", storeID, "--source", "accepted", "--limit", "1")
	if len(versions.Result.(map[string]any)["versions"].([]any)) != 1 {
		t.Fatal("missing CLI version discovery")
	}
	cliComparison := runRealCLI(t, binary, origin, "architecture", "compare", "--store-id", storeID, "--before-kind", "accepted", "--before-revision", revision, "--after-kind", "accepted", "--after-revision", revision)
	side := map[string]any{"kind": "accepted", "revision": revision}
	mcpComparison := runRealMCP(t, ctx, session, "architecture_compare", map[string]any{"store_id": storeID, "before": side, "after": side})
	if !reflect.DeepEqual(cliComparison.Result, mcpComparison.Result) || cliComparison.Result.(map[string]any)["diff"] != "" {
		t.Fatal("CLI/MCP comparison differ")
	}
	refreshed := runRealCLI(t, binary, origin, "architecture", "refresh", "--store-id", storeID, "--accepted-revision", revision)
	if refreshed.Result.(map[string]any)["classification"] != "unchanged" {
		t.Fatalf("CLI Refresh: %+v", refreshed)
	}

	changeB := runRealMCP(t, ctx, session, "change_set_create", map[string]any{"store_id": storeID, "accepted_revision": revision, "name": "MCP proposal"})
	idB := changeB.Result.(map[string]any)["id"].(string)
	proposalB := "# MCP proposal\n\nIndependent exact body.\n"
	editedB := runRealMCP(t, ctx, session, "change_set_edit_proposal", map[string]any{"store_id": storeID, "change_set_id": idB, "generation": 0, "proposal_markdown": proposalB})
	if editedB.Result.(map[string]any)["proposal_markdown"] != proposalB {
		t.Fatalf("MCP proposal: %+v", editedB)
	}

	componentA := runRealCLI(t, binary, origin, "component", "create", "--store-id", storeID, "--change-set-id", idA, "--generation", "1", "--diagram-id", rootID, "--title", "Gateway")
	if componentA.Result.(map[string]any)["generation"] != float64(2) {
		t.Fatalf("CLI mutation: %+v", componentA)
	}
	gatewayID := componentA.Result.(map[string]any)["component_id"].(string)
	workerA := runRealMCP(t, ctx, session, "component_create", map[string]any{"store_id": storeID, "change_set_id": idA, "generation": 2, "diagram_id": rootID, "title": "Worker", "description": "Does work.\n"})
	workerAID := workerA.Result.(map[string]any)["component_id"].(string)
	invalid := runRealMCP(t, ctx, session, "relationship_add", map[string]any{
		"store_id": storeID, "change_set_id": idA, "generation": 3, "source_id": gatewayID, "target_id": "not-a-component-id", "label": "   ",
	})
	if invalid.Result.(map[string]any)["candidate_valid"] != false || invalid.Result.(map[string]any)["generation"] != float64(4) {
		t.Fatalf("invalid raw Relationship was not retained: %+v", invalid)
	}
	invalidInspect := runRealCLI(t, binary, origin, "change-set", "inspect", "--store-id", storeID, "--change-set-id", idA)
	invalidRows := invalidInspect.Result.(map[string]any)["components"].([]any)[0].(map[string]any)["relationships"].([]any)
	if len(invalidRows) != 1 || invalidRows[0].(map[string]any)["target_id"] != "not-a-component-id" || invalidRows[0].(map[string]any)["label"] != "   " {
		t.Fatalf("invalid raw selector changed: %+v", invalidInspect)
	}
	repaired := runRealCLI(t, binary, origin, "relationship", "edit", "--store-id", storeID, "--change-set-id", idA, "--generation", "4", "--source-id", gatewayID, "--old-target-id", "not-a-component-id", "--old-label", "   ", "--occurrence", "1", "--target-id", workerAID, "--label", "calls")
	if repaired.Result.(map[string]any)["candidate_valid"] != true {
		t.Fatalf("raw Relationship repair failed: %+v", repaired)
	}
	runRealMCP(t, ctx, session, "relationship_add", map[string]any{
		"store_id": storeID, "change_set_id": idA, "generation": 5, "source_id": gatewayID, "target_id": workerAID, "label": "calls",
	})
	duplicateEdited := runRealCLI(t, binary, origin, "relationship", "edit", "--store-id", storeID, "--change-set-id", idA, "--generation", "6", "--source-id", gatewayID, "--old-target-id", workerAID, "--old-label", "calls", "--occurrence", "2", "--target-id", workerAID, "--label", "calls async")
	if duplicateEdited.Result.(map[string]any)["generation"] != float64(7) {
		t.Fatalf("duplicate occurrence edit: %+v", duplicateEdited)
	}
	runRealMCP(t, ctx, session, "relationship_remove", map[string]any{
		"store_id": storeID, "change_set_id": idA, "generation": 7, "source_id": gatewayID, "target_id": workerAID, "label": "calls", "occurrence": 1,
	})
	runRealMCP(t, ctx, session, "component_edit", map[string]any{
		"store_id": storeID, "change_set_id": idA, "generation": 8, "component_id": gatewayID, "title": "Gateway", "description": "Routes requests.\n",
	})
	detailCreated := runRealCLI(t, binary, origin, "diagram", "create-detail", "--store-id", storeID, "--change-set-id", idA, "--generation", "9", "--component-id", gatewayID, "--title", "Gateway internals")
	detailID := detailCreated.Result.(map[string]any)["diagram_id"].(string)
	runRealMCP(t, ctx, session, "component_move_home", map[string]any{
		"store_id": storeID, "change_set_id": idA, "generation": 10, "component_id": workerAID, "diagram_id": detailID,
	})
	shown := runRealCLI(t, binary, origin, "diagram", "show-component", "--store-id", storeID, "--change-set-id", idA, "--generation", "11", "--diagram-id", detailID, "--component-id", gatewayID)
	if shown.Result.(map[string]any)["present"] != true {
		t.Fatalf("show Component: %+v", shown)
	}
	stopped := runRealMCP(t, ctx, session, "diagram_stop_showing_component", map[string]any{
		"store_id": storeID, "change_set_id": idA, "generation": 12, "diagram_id": detailID, "component_id": gatewayID,
	})
	if stopped.Result.(map[string]any)["present"] != false {
		t.Fatalf("stop showing Component: %+v", stopped)
	}
	runRealCLI(t, binary, origin, "diagram", "edit-title", "--store-id", storeID, "--change-set-id", idA, "--generation", "13", "--diagram-id", detailID, "--title", "Gateway runtime")
	renamedA := runRealMCP(t, ctx, session, "change_set_rename", map[string]any{
		"store_id": storeID, "change_set_id": idA, "generation": 14, "name": "CLI and MCP proposal",
	})
	if renamedA.Result.(map[string]any)["generation"] != float64(15) {
		t.Fatalf("rename generation: %+v", renamedA)
	}
	finalA := runRealCLI(t, binary, origin, "change-set", "inspect", "--store-id", storeID, "--change-set-id", idA).Result.(map[string]any)
	if finalA["generation"] != float64(15) || len(finalA["detail_diagrams"].([]any)) != 1 || len(finalA["home_moves"].([]any)) != 1 {
		t.Fatalf("structured parity projection: %+v", finalA)
	}
	stoppedReferenceSeen := false
	for _, value := range finalA["references"].([]any) {
		reference := value.(map[string]any)
		stoppedReferenceSeen = stoppedReferenceSeen || reference["diagram_id"] == detailID && reference["component_id"] == gatewayID && reference["present"] == false
	}
	if !stoppedReferenceSeen {
		t.Fatalf("stopped reference fact missing: %+v", finalA["references"])
	}
	finalRelationships := finalA["components"].([]any)[0].(map[string]any)["relationships"].([]any)
	if len(finalRelationships) != 1 || finalRelationships[0].(map[string]any)["label"] != "calls async" {
		t.Fatalf("Relationship occurrence parity: %+v", finalRelationships)
	}

	componentB := runRealMCP(t, ctx, session, "component_create", map[string]any{"store_id": storeID, "change_set_id": idB, "generation": 1, "diagram_id": rootID, "title": "Independent", "description": ""})
	if componentB.Result.(map[string]any)["generation"] != float64(2) {
		t.Fatalf("MCP independent mutation: %+v", componentB)
	}

	reviewA := runRealCLI(t, binary, origin, "change-set", "review", "--store-id", storeID, "--change-set-id", idA, "--generation", "15")
	bindingA := reviewA.Result.(map[string]any)
	expectedReviewAURL := origin + "/projects/" + created.Context.Project.Slug + "/proposals/" + idA + "/review"
	if bindingA["review_url"] != expectedReviewAURL {
		t.Fatalf("CLI review URL = %#v, want %q", bindingA["review_url"], expectedReviewAURL)
	}
	repeatedReviewA := runRealMCP(t, ctx, session, "change_set_review", map[string]any{"store_id": storeID, "change_set_id": idA, "generation": 15})
	if repeatedReviewA.Result.(map[string]any)["reviewed_state"] != bindingA["reviewed_state"] {
		t.Fatalf("repeated cross-client review changed state: CLI=%+v MCP=%+v", bindingA, repeatedReviewA.Result)
	}
	commentFile := filepath.Join(runtimeRoot, "review-comments.json")
	commentJSON := `[{"body":"Keep this exact proposal context.","anchor":{"kind":"proposal"}}]`
	if err := os.WriteFile(commentFile, []byte(commentJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	submittedA := runRealCLI(t, binary, origin, "review-submission", "submit", "--store-id", storeID, "--change-set-id", idA,
		"--reviewed-state", bindingA["reviewed_state"].(string), "--base-revision", bindingA["base_revision"].(string), "--candidate-tree", bindingA["candidate_tree"].(string),
		"--generation", "15", "--verdict", "request_changes", "--author", "CLI reviewer", "--body", "Review body.", "--comments-file", commentFile)
	submittedAMap := submittedA.Result.(map[string]any)
	reviewAID := submittedAMap["id"].(string)
	expectedSubmittedAURL := expectedReviewAURL[:len(expectedReviewAURL)-len("review")] + "reviews/" + reviewAID
	if submittedAMap["review_url"] != expectedSubmittedAURL {
		t.Fatalf("submitted review URL=%#v want=%q", submittedAMap["review_url"], expectedSubmittedAURL)
	}
	listedAReviews := runRealMCP(t, ctx, session, "review_submissions_list", map[string]any{"store_id": storeID, "change_set_id": idA})
	listedReviews := listedAReviews.Result.(map[string]any)["reviews"].([]any)
	if len(listedReviews) != 1 || listedReviews[0].(map[string]any)["review_url"] != expectedSubmittedAURL {
		t.Fatalf("MCP review list=%+v", listedAReviews)
	}
	inspectedAReview := runRealMCP(t, ctx, session, "review_submission_inspect", map[string]any{"store_id": storeID, "change_set_id": idA, "review_id": reviewAID})
	if inspectedAReview.Result.(map[string]any)["review_url"] != expectedSubmittedAURL {
		t.Fatalf("MCP review inspect=%+v", inspectedAReview)
	}
	reviewB := runRealMCP(t, ctx, session, "change_set_review", map[string]any{"store_id": storeID, "change_set_id": idB, "generation": 2})
	expectedReviewBURL := origin + "/projects/" + created.Context.Project.Slug + "/proposals/" + idB + "/review"
	if reviewB.Result.(map[string]any)["review_url"] != expectedReviewBURL {
		t.Fatalf("MCP review URL = %#v, want %q", reviewB.Result, expectedReviewBURL)
	}
	bindingBInitial := reviewB.Result.(map[string]any)
	submittedB := runRealMCP(t, ctx, session, "review_submission_submit", map[string]any{
		"store_id": storeID, "change_set_id": idB, "reviewed_state": bindingBInitial["reviewed_state"], "base_revision": bindingBInitial["base_revision"],
		"candidate_tree": bindingBInitial["candidate_tree"], "generation": 2, "verdict": "approve", "author": "MCP reviewer", "body": "", "comments": []any{},
	})
	reviewBID := submittedB.Result.(map[string]any)["id"].(string)
	if inspectedBReview := runRealCLI(t, binary, origin, "review-submission", "inspect", "--store-id", storeID, "--change-set-id", idB, "--review-id", reviewBID); inspectedBReview.Result.(map[string]any)["verdict"] != "approve" {
		t.Fatalf("CLI review inspect=%+v", inspectedBReview)
	}
	updatedA := runRealCLI(t, binary, origin, "architecture", "update", "--store-id", storeID, "--change-set-id", idA, "--base-revision", bindingA["base_revision"].(string), "--candidate-tree", bindingA["candidate_tree"].(string), "--generation", "15")
	if !updatedA.OK || updatedA.Context.AcceptedRevision == nil || *updatedA.Context.AcceptedRevision == revision {
		t.Fatalf("CLI update A: %+v", updatedA)
	}
	appliedReviewA := runRealCLI(t, binary, origin, "review-submission", "inspect", "--store-id", storeID, "--change-set-id", idA, "--review-id", reviewAID)
	if appliedReviewA.Result.(map[string]any)["lifecycle"] != "applied" || appliedReviewA.Result.(map[string]any)["current_generation"] != true {
		t.Fatalf("applied review context=%+v", appliedReviewA)
	}
	acceptedRevision := *updatedA.Context.AcceptedRevision

	inspectB := runRealMCP(t, ctx, session, "change_set_inspect", map[string]any{"store_id": storeID, "change_set_id": idB})
	if inspectB.Result.(map[string]any)["out_of_date"] != true {
		t.Fatalf("MCP inspect preserved B: %+v", inspectB)
	}
	editedOutOfDate := runRealMCP(t, ctx, session, "change_set_edit_proposal", map[string]any{"store_id": storeID, "change_set_id": idB, "generation": 2, "proposal_markdown": proposalB + "Still editable.\n"})
	if editedOutOfDate.Result.(map[string]any)["out_of_date"] != true || editedOutOfDate.Result.(map[string]any)["generation"] != float64(3) {
		t.Fatalf("out-of-date edit: %+v", editedOutOfDate)
	}
	bindingB := runRealMCP(t, ctx, session, "change_set_review", map[string]any{"store_id": storeID, "change_set_id": idB, "generation": 3}).Result.(map[string]any)
	if bindingB["review_url"] != expectedReviewBURL {
		t.Fatalf("out-of-date MCP review URL = %#v, want %q", bindingB["review_url"], expectedReviewBURL)
	}
	rejected := runRealCLIError(t, binary, origin, "architecture", "update", "--store-id", storeID, "--change-set-id", idB, "--base-revision", bindingB["base_revision"].(string), "--candidate-tree", bindingB["candidate_tree"].(string), "--generation", "3")
	if rejected.Error == nil || rejected.Error.Code != "change_set_out_of_date" {
		t.Fatalf("out-of-date B update: %+v", rejected)
	}
	// Both transports proxy the same inspected S/B/A/P and ordinary residual.
	beforeReconcile := runRealCLI(t, binary, origin, "change-set", "inspect", "--store-id", storeID, "--change-set-id", idB).Result.(map[string]any)
	inputs := map[string]any{"store_id": storeID, "change_set_id": idB, "generation": beforeReconcile["generation"], "change_set_state": beforeReconcile["change_set_state"], "base_revision": beforeReconcile["base_revision"], "candidate_tree": beforeReconcile["candidate_tree"], "accepted_revision": acceptedRevision}
	preview := runRealMCP(t, ctx, session, "change_set_reconcile_preview", inputs).Result.(map[string]any)
	if preview["status"] != "ready" {
		t.Fatalf("MCP reconciliation: %+v", preview)
	}
	cliInputs := []string{"--store-id", storeID, "--change-set-id", idB, "--generation", "3", "--change-set-state", beforeReconcile["change_set_state"].(string), "--base-revision", beforeReconcile["base_revision"].(string), "--candidate-tree", beforeReconcile["candidate_tree"].(string), "--accepted-revision", acceptedRevision}
	cliPreview := runRealCLI(t, binary, origin, append([]string{"change-set", "reconcile-preview"}, cliInputs...)...).Result.(map[string]any)
	if !reflect.DeepEqual(preview, cliPreview) {
		t.Fatal("CLI/MCP previews differ")
	}
	command := exec.Command(binary, append(append([]string{"--server", origin, "--json", "change-set", "reconcile-apply"}, cliInputs...), "--resolutions-file", "-")...)
	command.Stdin = strings.NewReader("[]")
	output, err := command.Output()
	if err != nil {
		t.Fatalf("CLI stdin Apply: %s %v", output, err)
	}
	var applied agentapi.Envelope
	if err := json.Unmarshal(output, &applied); err != nil || !applied.OK {
		t.Fatalf("CLI stdin result: %s %v", output, err)
	}
	reconciled := applied.Result.(map[string]any)
	if reconciled["generation"] != float64(4) || reconciled["base_revision"] != acceptedRevision || reconciled["review"] != nil {
		t.Fatalf("CLI residual: %+v", reconciled)
	}
	inputs["resolutions"] = []any{}
	retry := runRealMCPError(t, ctx, session, "change_set_reconcile_apply", inputs)
	if retry.Error == nil || retry.Error.Code != "change_set_state_mismatch" {
		t.Fatalf("MCP old-S retry: %+v", retry)
	}
	parentInputs := map[string]any{"store_id": storeID, "change_set_id": idB, "generation": 4, "diagram_id": detailID}
	options := runRealMCP(t, ctx, session, "diagram_parent_options", parentInputs)
	if !options.OK {
		t.Fatal(options.Error)
	}
	runRealCLI(t, binary, origin, "diagram", "parent-options", "--store-id", storeID, "--change-set-id", idB, "--generation", "4", "--diagram-id", detailID)
	parentInputs["anchor_component_id"] = componentB.Result.(map[string]any)["component_id"]
	moved := runRealMCP(t, ctx, session, "diagram_reassign_detail", parentInputs).Result.(map[string]any)
	if moved["generation"] != float64(5) {
		t.Fatalf("MCP ordinary parent: %+v", moved)
	}
	runRealCLI(t, binary, origin, "diagram", "reassign-detail", "--store-id", storeID, "--change-set-id", idB, "--generation", "5", "--diagram-id", detailID, "--anchor-component-id", gatewayID)
	appliedReview := runRealCLIError(t, binary, origin, "change-set", "review", "--store-id", storeID, "--change-set-id", idA, "--generation", "999")
	if appliedReview.Error == nil || appliedReview.Error.Code != "change_set_not_editable" {
		t.Fatalf("applied CLI review: %+v", appliedReview)
	}
	appliedDiscard := runRealMCPError(t, ctx, session, "change_set_discard", map[string]any{"store_id": storeID, "change_set_id": idA, "generation": 999})
	if appliedDiscard.Error == nil || appliedDiscard.Error.Code != "change_set_not_editable" {
		t.Fatalf("applied MCP discard: %+v", appliedDiscard)
	}

	discardable := runRealMCP(t, ctx, session, "change_set_create", map[string]any{"store_id": storeID, "accepted_revision": acceptedRevision, "name": "Discard me"})
	discardID := discardable.Result.(map[string]any)["id"].(string)
	runRealCLI(t, binary, origin, "change-set", "discard", "--store-id", storeID, "--change-set-id", discardID, "--generation", "0")
	discardedInspect := runRealCLIError(t, binary, origin, "change-set", "inspect", "--store-id", storeID, "--change-set-id", discardID)
	if discardedInspect.Error == nil || discardedInspect.Error.Code != "change_set_not_found" {
		t.Fatalf("discarded inspect: %+v", discardedInspect)
	}

	malformed := runRealCLI(t, binary, origin, "change-set", "create", "--store-id", storeID, "--accepted-revision", acceptedRevision, "--name", "Malformed record")
	malformedID := malformed.Result.(map[string]any)["id"].(string)
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	malformedRef := "refs/workbraid/change-sets/active/" + malformedID
	malformedObject := runBareGit(t, storePath, "rev-parse", malformedRef)
	runBareGit(t, storePath, "update-ref", malformedRef, acceptedRevision, malformedObject)
	runRealMCP(t, ctx, session, "project_open", map[string]any{"slug": "bridge-authority"})
	unavailableInspect := runRealCLIError(t, binary, origin, "change-set", "inspect", "--store-id", storeID, "--change-set-id", malformedID)
	if unavailableInspect.Error == nil || unavailableInspect.Error.Code != "change_set_unavailable" {
		t.Fatalf("unavailable CLI inspect: %+v", unavailableInspect)
	}
	unavailableMutation := runRealMCPError(t, ctx, session, "diagram_edit_title", map[string]any{
		"store_id": storeID, "change_set_id": malformedID, "generation": 999, "diagram_id": rootID, "title": "No",
	})
	if unavailableMutation.Error == nil || unavailableMutation.Error.Code != "change_set_unavailable" {
		t.Fatalf("unavailable MCP mutation: %+v", unavailableMutation)
	}
	listed := runRealCLI(t, binary, origin, "change-set", "list", "--store-id", storeID)
	if len(listed.Result.(map[string]any)["change_sets"].([]any)) != 2 {
		t.Fatalf("parallel lifecycle list: %+v", listed)
	}
	if len(listed.Result.(map[string]any)["unavailable"].([]any)) != 1 {
		t.Fatalf("unavailable lifecycle list: %+v", listed)
	}
	runRealMCP(t, ctx, session, "project_close", map[string]any{"store_id": storeID})
	reopened := runRealCLI(t, binary, origin, "project", "open", "--slug", "bridge-authority")
	if reopened.Context.Project == nil || reopened.Context.Project.StoreID != storeID {
		t.Fatalf("reopened project: %+v", reopened)
	}
	reloadedAppliedReview := runRealMCP(t, ctx, session, "review_submission_inspect", map[string]any{"store_id": storeID, "change_set_id": idA, "review_id": reviewAID})
	if reloadedAppliedReview.Result.(map[string]any)["lifecycle"] != "applied" || reloadedAppliedReview.Result.(map[string]any)["current_generation"] != true {
		t.Fatalf("reloaded applied review: %+v", reloadedAppliedReview)
	}
	reloadedEarlierReview := runRealCLI(t, binary, origin, "review-submission", "inspect", "--store-id", storeID, "--change-set-id", idB, "--review-id", reviewBID)
	if reloadedEarlierReview.Result.(map[string]any)["lifecycle"] != "active" || reloadedEarlierReview.Result.(map[string]any)["current_generation"] != false || reloadedEarlierReview.Result.(map[string]any)["binding"].(map[string]any)["generation"] != float64(2) {
		t.Fatalf("reloaded earlier review: %+v", reloadedEarlierReview)
	}
	placement := runRealCLI(t, binary, origin, "change-set", "create", "--store-id", storeID, "--accepted-revision", acceptedRevision, "--name", "Transport placement")
	placementID := placement.Result.(map[string]any)["id"].(string)
	args := map[string]any{"store_id": storeID, "change_set_id": placementID, "generation": 0, "diagram_id": rootID, "component_id": gatewayID, "x": -180, "y": 420}
	keptPosition := runRealMCP(t, ctx, session, "diagram_set_position", args)
	if keptPosition.Result.(map[string]any)["generation"] != float64(1) {
		t.Fatal("MCP placement generation")
	}
	readPosition := runRealCLI(t, binary, origin, "diagram", "positions", "--store-id", storeID, "--change-set-id", placementID, "--diagram-id", rootID)
	readBytes, _ := json.Marshal(readPosition.Result)
	if !bytes.Contains(readBytes, []byte(`"x":-180`)) {
		t.Fatal("CLI did not read MCP placement")
	}
	args["generation"] = 1
	delete(args, "x")
	delete(args, "y")
	delete(args, "component_id")
	runRealMCP(t, ctx, session, "diagram_auto_layout", args)
	runRealCLI(t, binary, origin, "diagram", "set-position", "--store-id", storeID, "--change-set-id", placementID, "--generation", "2", "--diagram-id", rootID, "--component-id", gatewayID, "--x=-100000", "--y=100000")
	args["generation"] = 3
	delete(args, "component_id")
	runRealMCP(t, ctx, session, "diagram_auto_layout", args)
	readBack := runRealMCP(t, ctx, session, "diagram_positions", map[string]any{"store_id": storeID, "change_set_id": placementID, "diagram_id": rootID})
	for _, appearance := range readBack.Result.(map[string]any)["appearances"].([]any) {
		if appearance.(map[string]any)["position"] == nil || appearance.(map[string]any)["position_source"] != "stored" {
			t.Fatal("auto-layout did not retain stored coordinates")
		}
	}
	invalidPosition := runRealCLIError(t, binary, origin, "diagram", "set-position", "--store-id", storeID, "--change-set-id", placementID, "--generation", "4", "--diagram-id", rootID, "--component-id", gatewayID, "--x=100001", "--y=0")
	if invalidPosition.Error == nil || invalidPosition.Error.Code != "invalid_request" {
		t.Fatalf("bounds: %+v", invalidPosition)
	}
	runRealCLI(t, binary, origin, "diagram", "set-position", "--store-id", storeID, "--change-set-id", placementID, "--generation", "4", "--diagram-id", rootID, "--component-id", gatewayID, "--x=-180", "--y=420")
	other := runRealMCP(t, ctx, session, "change_set_create", map[string]any{"store_id": storeID, "accepted_revision": acceptedRevision, "name": "Other placement"}).Result.(map[string]any)["id"].(string)
	runRealMCP(t, ctx, session, "diagram_set_position", map[string]any{"store_id": storeID, "change_set_id": other, "generation": 0, "diagram_id": rootID, "component_id": gatewayID, "x": 20, "y": 30})
	otherReview := runRealMCP(t, ctx, session, "change_set_review", map[string]any{"store_id": storeID, "change_set_id": other, "generation": 1}).Result.(map[string]any)
	otherAccepted := runRealMCP(t, ctx, session, "architecture_update", map[string]any{"store_id": storeID, "change_set_id": other, "generation": 1, "base_revision": otherReview["base_revision"], "candidate_tree": otherReview["candidate_tree"]})
	currentPlacement := runRealMCP(t, ctx, session, "change_set_inspect", map[string]any{"store_id": storeID, "change_set_id": placementID}).Result.(map[string]any)
	placementInputs := map[string]any{"store_id": storeID, "change_set_id": placementID, "generation": 5, "change_set_state": currentPlacement["change_set_state"], "base_revision": currentPlacement["base_revision"], "candidate_tree": currentPlacement["candidate_tree"], "accepted_revision": *otherAccepted.Context.AcceptedRevision}
	placementPreview := runRealMCP(t, ctx, session, "change_set_reconcile_preview", placementInputs).Result.(map[string]any)
	conflicts := placementPreview["conflicts"].([]any)
	if placementPreview["status"] != "needs_resolution" || len(conflicts) != 1 || conflicts[0].(map[string]any)["locator"].(map[string]any)["kind"] != "node_position" {
		t.Fatalf("typed placement conflict: %+v", placementPreview)
	}
	placementInputs["resolutions"] = []any{map[string]any{"locator": conflicts[0].(map[string]any)["locator"], "choice": "manual", "value": map[string]any{"position": map[string]any{"x": 320, "y": -180}}}}
	resolvedPlacement := runRealMCP(t, ctx, session, "change_set_reconcile_apply", placementInputs).Result.(map[string]any)
	if resolvedPlacement["generation"] != float64(6) || resolvedPlacement["review"] != nil {
		t.Fatalf("placement residual: %+v", resolvedPlacement)
	}
	if retry := runRealMCPError(t, ctx, session, "change_set_reconcile_apply", placementInputs); retry.Error.Code != "change_set_state_mismatch" {
		t.Fatalf("placement old-S retry: %+v", retry)
	}
	// The production clients share one sizing authority, including exact no-ops.
	sizeArgs := map[string]any{"store_id": storeID, "change_set_id": placementID, "generation": 6, "diagram_id": rootID, "component_id": gatewayID, "width": 321, "height": 159}
	sized := runRealMCP(t, ctx, session, "diagram_set_size", sizeArgs).Result.(map[string]any)
	if sized["generation"] != float64(7) {
		t.Fatalf("MCP sizing: %+v", sized)
	}
	readSize := runRealCLI(t, binary, origin, "diagram", "sizes", "--store-id", storeID, "--change-set-id", placementID, "--diagram-id", rootID)
	mcpSize := runRealMCP(t, ctx, session, "diagram_sizes", map[string]any{"store_id": storeID, "change_set_id": placementID, "diagram_id": rootID})
	if !reflect.DeepEqual(readSize.Result, mcpSize.Result) {
		t.Fatal("CLI/MCP size projections differ")
	}
	for _, appearance := range readSize.Result.(map[string]any)["appearances"].([]any) {
		a := appearance.(map[string]any)
		if a["size"] == nil || a["size_source"] != "stored" || a["position"] == nil {
			t.Fatal("native v4 incomplete geometry")
		}
	}
	beforeNoop := runRealCLI(t, binary, origin, "change-set", "review", "--store-id", storeID, "--change-set-id", placementID, "--generation", "7")
	noopSize := runRealCLI(t, binary, origin, "diagram", "set-size", "--store-id", storeID, "--change-set-id", placementID, "--generation", "7", "--diagram-id", rootID, "--component-id", gatewayID, "--width", "321", "--height", "159").Result.(map[string]any)
	if noopSize["generation"] != float64(7) || !reflect.DeepEqual(noopSize["review"], beforeNoop.Result.(map[string]any)["review"]) {
		t.Fatalf("CLI same-size invalidated review: %+v", noopSize)
	}
	invalidSize, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "diagram_set_size", Arguments: map[string]any{"store_id": storeID, "change_set_id": placementID, "generation": 7, "diagram_id": rootID, "component_id": gatewayID, "width": 79, "height": 48}})
	if err != nil || !invalidSize.IsError {
		t.Fatalf("MCP size bounds: %+v", invalidSize)
	}
	delete(sizeArgs, "width")
	delete(sizeArgs, "height")
	sizeArgs["generation"] = 7
	restoredSize := runRealMCP(t, ctx, session, "diagram_restore_default_size", sizeArgs).Result.(map[string]any)
	if restoredSize["generation"] != float64(8) {
		t.Fatalf("MCP restore default: %+v", restoredSize)
	}
}

func TestSkillHelpAndCLIExposeOnlyV2ChangeSetWorkflow(t *testing.T) {
	var skill, help, stderr bytes.Buffer
	if code := run([]string{"--skill"}, &skill, &stderr, bytes.NewReader(nil)); code != 0 {
		t.Fatalf("skill exit=%d", code)
	}
	if code := run([]string{"--help"}, &help, &stderr, bytes.NewReader(nil)); code != 0 {
		t.Fatalf("help exit=%d", code)
	}
	for _, exact := range []string{"change-set list", "change-set create", "change-set inspect", "change-set rename", "change-set edit-proposal", "change-set review", "change-set discard", "review-submission list", "review-submission inspect", "review-submission submit", "--change-set-id <uuid>", "change_set_out_of_date", "review_anchor_invalid", "component_markdown", "reviewed_state", "Applied change sets are immutable evidence", "review_url", "diagram parent-options", "diagram reassign-detail", "change-set reconcile-preview", "change-set reconcile-apply", "change_set_state_mismatch", "component_description", "relationship_count", "competing_children", "replace_identity", "--resolutions-file"} {
		if !strings.Contains(skill.String(), exact) && !strings.Contains(help.String(), exact) {
			t.Fatalf("v2 help/skill missing %q", exact)
		}
	}
	for _, obsolete := range []string{"changes inspect", "changes review", "pending_generation", "--generation <n|none>", "Reconciliation is a future product", "WorkBraid does not reconcile"} {
		if strings.Contains(skill.String(), obsolete) || strings.Contains(help.String(), obsolete) {
			t.Fatalf("v1 wording remains: %q", obsolete)
		}
	}
}

func TestCLIUsesRunningLoopbackAuthorityAndRejectsServerFlags(t *testing.T) {
	server := httptest.NewServer(web.NewHandler("http://127.0.0.1", t.TempDir(), t.TempDir()))
	defer server.Close()
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--server", server.URL, "--json", "project", "create", "--name", "CLI project"}, &stdout, &stderr, bytes.NewReader(nil)); code != 0 {
		t.Fatalf("create exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	created := decodeCLIEnvelope(t, &stdout)
	stdout.Reset()
	if code := run([]string{"--server", server.URL, "--json", "change-set", "create", "--store-id", created.Context.Project.StoreID, "--accepted-revision", *created.Context.AcceptedRevision}, &stdout, &stderr, bytes.NewReader(nil)); code != 0 {
		t.Fatalf("change-set create exit=%d stdout=%s", code, stdout.String())
	}
	if result := decodeCLIEnvelope(t, &stdout).Result.(map[string]any); result["generation"] != float64(0) || result["id"] == "" || result["name"] == "" {
		t.Fatalf("generated change set = %#v", result)
	}
	stdout.Reset()
	if code := run([]string{"--json", "--data-dir", t.TempDir(), "status"}, &stdout, &stderr, bytes.NewReader(nil)); code == 0 {
		t.Fatal("client mode accepted --data-dir")
	}
	if envelope := decodeCLIEnvelope(t, &stdout); envelope.Error == nil || envelope.Error.Code != "invalid_request" {
		t.Fatalf("server flag envelope = %+v", envelope)
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
	client := mcp.NewClient(&mcp.Implementation{Name: "workbraid-test", Version: "test"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for _, exact := range []string{"Accepted Architecture", "durable named change sets", "change_set_id", "generation", "review_url"} {
		if !strings.Contains(session.InitializeResult().Instructions, exact) {
			t.Fatalf("MCP instructions missing %q: %q", exact, session.InitializeResult().Instructions)
		}
	}
	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantNames := []string{
		"architecture_versions", "architecture_compare",
		"diagram_shapes", "diagram_notes", "diagram_set_shape", "diagram_restore_default_shape", "diagram_add_note", "diagram_edit_note", "diagram_delete_note",
		"architecture_inspect", "architecture_refresh", "architecture_update", "change_set_create", "change_set_discard", "change_set_edit_proposal", "change_set_inspect", "change_set_reconcile_apply", "change_set_reconcile_preview", "change_set_rename", "change_set_review", "change_sets_list",
		"component_create", "component_edit", "component_move_home", "diagram_create_detail", "diagram_edit_title", "diagram_parent_options", "diagram_positions", "diagram_set_position", "diagram_auto_layout", "diagram_sizes", "diagram_set_size", "diagram_restore_default_size", "diagram_routes", "diagram_set_route", "diagram_restore_default_route", "diagram_reassign_detail", "diagram_show_component", "diagram_stop_showing_component", "project_close", "project_create", "project_current", "project_open", "projects_list", "relationship_add", "relationship_edit", "relationship_remove",
		"review_submission_inspect", "review_submission_submit", "review_submissions_list", "status",
	}
	gotNames := make([]string, len(listed.Tools))
	for index, tool := range listed.Tools {
		gotNames[index] = tool.Name
		input, ok := tool.InputSchema.(map[string]any)
		if !ok || input["type"] != "object" || input["additionalProperties"] != false || tool.OutputSchema == nil || tool.Description == "" || tool.Annotations == nil {
			t.Fatalf("tool %q schema incomplete: %#v", tool.Name, tool.InputSchema)
		}
		if tool.Name == "change_set_review" && !strings.Contains(tool.Description, "review_url") {
			t.Fatalf("review tool does not tell agents about review_url: %q", tool.Description)
		}
		if tool.Name == "review_submission_submit" {
			properties, ok := input["properties"].(map[string]any)
			comments, commentsOK := properties["comments"].(map[string]any)
			items, itemsOK := comments["items"].(map[string]any)
			commentProperties, propertiesOK := items["properties"].(map[string]any)
			anchor, anchorOK := commentProperties["anchor"].(map[string]any)
			variants, variantsOK := anchor["oneOf"].([]any)
			if !ok || !commentsOK || !itemsOK || !propertiesOK || !anchorOK || !variantsOK || len(variants) != 9 {
				t.Fatalf("review submission does not expose closed typed anchor variants: %#v", input)
			}
		}
	}
	slices.Sort(gotNames)
	slices.Sort(wantNames)
	if !slices.Equal(gotNames, wantNames) {
		t.Fatalf("tool names=%v want=%v", gotNames, wantNames)
	}
	called, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "status", Arguments: map[string]any{}})
	if err != nil || called.IsError || called.StructuredContent == nil {
		t.Fatalf("status result=%+v err=%v", called, err)
	}
	// Exercise the discovered closed manual union through the MCP SDK and the
	// real HTTP/Git authority, alongside CLI stdin Check choices.
	project := runRealMCP(t, ctx, session, "project_create", map[string]any{"name": "Typed reconciliation"})
	store, accepted := project.Context.Project.StoreID, *project.Context.AcceptedRevision
	root := runRealMCP(t, ctx, session, "architecture_inspect", map[string]any{}).Result.(map[string]any)["root_diagram_id"]
	create := func(name string) string {
		return runRealMCP(t, ctx, session, "change_set_create", map[string]any{"store_id": store, "accepted_revision": accepted, "name": name}).Result.(map[string]any)["id"].(string)
	}
	accept := func(id string, generation int) {
		binding := runRealMCP(t, ctx, session, "change_set_review", map[string]any{"store_id": store, "change_set_id": id, "generation": generation}).Result.(map[string]any)
		result := runRealMCP(t, ctx, session, "architecture_update", map[string]any{"store_id": store, "change_set_id": id, "generation": generation, "base_revision": binding["base_revision"], "candidate_tree": binding["candidate_tree"]})
		accepted = *result.Context.AcceptedRevision
	}
	setup := create("Initial Gateway")
	component := runRealMCP(t, ctx, session, "component_create", map[string]any{"store_id": store, "change_set_id": setup, "generation": 0, "diagram_id": root, "title": "Gateway", "description": ""}).Result.(map[string]any)["component_id"]
	accept(setup, 1)
	p := create("Proposed Description")
	a := create("Accepted Description")
	runRealMCP(t, ctx, session, "component_edit", map[string]any{"store_id": store, "change_set_id": p, "generation": 0, "component_id": component, "title": "Gateway", "description": "Proposed\n"})
	runRealMCP(t, ctx, session, "component_edit", map[string]any{"store_id": store, "change_set_id": a, "generation": 0, "component_id": component, "title": "Gateway", "description": "Accepted\n"})
	accept(a, 1)
	inspected := runRealMCP(t, ctx, session, "change_set_inspect", map[string]any{"store_id": store, "change_set_id": p}).Result.(map[string]any)
	inputs := map[string]any{"store_id": store, "change_set_id": p, "generation": 1, "change_set_state": inspected["change_set_state"], "base_revision": inspected["base_revision"], "candidate_tree": inspected["candidate_tree"], "accepted_revision": accepted}
	preview := runRealMCP(t, ctx, session, "change_set_reconcile_preview", inputs).Result.(map[string]any)
	conflicts := preview["conflicts"].([]any)
	if len(conflicts) != 1 {
		t.Fatalf("typed conflict: %+v", preview)
	}
	body := " \r\nExact **manual** Description\n"
	resolutions := []any{map[string]any{"locator": conflicts[0].(map[string]any)["locator"], "choice": "manual", "value": map[string]any{"text": body}}}
	encoded, _ := json.Marshal(resolutions)
	args := []string{"--server", httpServer.URL, "--json", "change-set", "reconcile-preview", "--store-id", store, "--change-set-id", p, "--generation", "1", "--change-set-state", inspected["change_set_state"].(string), "--base-revision", inspected["base_revision"].(string), "--candidate-tree", inspected["candidate_tree"].(string), "--accepted-revision", accepted, "--resolutions-file", "-"}
	var stdout, stderr bytes.Buffer
	if code := run(args, &stdout, &stderr, bytes.NewReader(encoded)); code != 0 {
		t.Fatalf("CLI typed stdin: %s %s", stdout.String(), stderr.String())
	}
	inputs["resolutions"] = resolutions
	checked := runRealMCP(t, ctx, session, "change_set_reconcile_preview", inputs)
	if !reflect.DeepEqual(decodeCLIEnvelope(t, &stdout).Result, checked.Result) {
		t.Fatal("typed CLI/MCP Check choices differed")
	}
	applied := runRealMCP(t, ctx, session, "change_set_reconcile_apply", inputs).Result.(map[string]any)
	if applied["generation"] != float64(2) || applied["base_revision"] != accepted || applied["review"] != nil {
		t.Fatalf("MCP manual apply: %+v", applied)
	}
	candidate := applied["candidate"].(map[string]any)
	if candidate["components"].([]any)[0].(map[string]any)["description"] != body {
		t.Fatal("MCP manual value changed exact body")
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
