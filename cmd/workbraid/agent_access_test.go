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
	if err != nil || len(tools.Tools) != 26 {
		t.Fatalf("real MCP discovery: tools=%d err=%v", len(tools.Tools), err)
	}
	createdBResult, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "change_set_create", Arguments: map[string]any{"store_id": storeID, "accepted_revision": revision, "name": "MCP proposal"}})
	if err != nil || createdBResult.IsError {
		t.Fatalf("MCP create: result=%+v err=%v", createdBResult, err)
	}
	changeB := mcpEnvelope(t, createdBResult)
	idB := changeB.Result.(map[string]any)["id"].(string)
	proposalB := "# MCP proposal\n\nIndependent exact body.\n"
	editedBResult, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "change_set_edit_proposal", Arguments: map[string]any{"store_id": storeID, "change_set_id": idB, "generation": 0, "proposal_markdown": proposalB}})
	if err != nil || editedBResult.IsError || mcpEnvelope(t, editedBResult).Result.(map[string]any)["proposal_markdown"] != proposalB {
		t.Fatalf("MCP proposal: result=%+v err=%v", editedBResult, err)
	}

	componentA := runRealCLI(t, binary, origin, "component", "create", "--store-id", storeID, "--change-set-id", idA, "--generation", "1", "--diagram-id", rootID, "--title", "Gateway")
	if componentA.Result.(map[string]any)["generation"] != float64(2) {
		t.Fatalf("CLI mutation: %+v", componentA)
	}
	componentBResult, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "component_create", Arguments: map[string]any{"store_id": storeID, "change_set_id": idB, "generation": 1, "diagram_id": rootID, "title": "Worker", "description": ""}})
	if err != nil || componentBResult.IsError || mcpEnvelope(t, componentBResult).Result.(map[string]any)["generation"] != float64(2) {
		t.Fatalf("MCP mutation: result=%+v err=%v", componentBResult, err)
	}

	reviewA := runRealCLI(t, binary, origin, "change-set", "review", "--store-id", storeID, "--change-set-id", idA, "--generation", "2")
	bindingA := reviewA.Result.(map[string]any)
	reviewBResult, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "change_set_review", Arguments: map[string]any{"store_id": storeID, "change_set_id": idB, "generation": 2}})
	if err != nil || reviewBResult.IsError {
		t.Fatalf("MCP review B: result=%+v err=%v", reviewBResult, err)
	}
	bindingB := mcpEnvelope(t, reviewBResult).Result.(map[string]any)
	updatedA := runRealCLI(t, binary, origin, "architecture", "update", "--store-id", storeID, "--change-set-id", idA, "--base-revision", bindingA["base_revision"].(string), "--candidate-tree", bindingA["candidate_tree"].(string), "--generation", "2")
	if !updatedA.OK || updatedA.Context.AcceptedRevision == nil || *updatedA.Context.AcceptedRevision == revision {
		t.Fatalf("CLI update A: %+v", updatedA)
	}

	inspectBResult, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "change_set_inspect", Arguments: map[string]any{"store_id": storeID, "change_set_id": idB}})
	if err != nil || inspectBResult.IsError || mcpEnvelope(t, inspectBResult).Result.(map[string]any)["out_of_date"] != true {
		t.Fatalf("MCP inspect preserved B: result=%+v err=%v", inspectBResult, err)
	}
	rejected := runRealCLIError(t, binary, origin, "architecture", "update", "--store-id", storeID, "--change-set-id", idB, "--base-revision", bindingB["base_revision"].(string), "--candidate-tree", bindingB["candidate_tree"].(string), "--generation", "2")
	if rejected.Error == nil || rejected.Error.Code != "change_set_out_of_date" {
		t.Fatalf("out-of-date B update: %+v", rejected)
	}
	listed := runRealCLI(t, binary, origin, "change-set", "list", "--store-id", storeID)
	if len(listed.Result.(map[string]any)["change_sets"].([]any)) != 2 {
		t.Fatalf("parallel lifecycle list: %+v", listed)
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
	for _, exact := range []string{"change-set list", "change-set create", "change-set inspect", "change-set rename", "change-set edit-proposal", "change-set review", "change-set discard", "--change-set-id <uuid>", "change_set_out_of_date", "Applied change sets are immutable evidence"} {
		if !strings.Contains(skill.String(), exact) && !strings.Contains(help.String(), exact) {
			t.Fatalf("v2 help/skill missing %q", exact)
		}
	}
	for _, obsolete := range []string{"changes inspect", "changes review", "pending_generation", "--generation <n|none>"} {
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
	for _, exact := range []string{"Accepted Architecture", "durable named change sets", "change_set_id", "generation"} {
		if !strings.Contains(session.InitializeResult().Instructions, exact) {
			t.Fatalf("MCP instructions missing %q: %q", exact, session.InitializeResult().Instructions)
		}
	}
	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantNames := []string{
		"architecture_inspect", "architecture_refresh", "architecture_update", "change_set_create", "change_set_discard", "change_set_edit_proposal", "change_set_inspect", "change_set_rename", "change_set_review", "change_sets_list",
		"component_create", "component_edit", "component_move_home", "diagram_create_detail", "diagram_edit_title", "diagram_show_component", "diagram_stop_showing_component", "project_close", "project_create", "project_current", "project_open", "projects_list", "relationship_add", "relationship_edit", "relationship_remove", "status",
	}
	gotNames := make([]string, len(listed.Tools))
	for index, tool := range listed.Tools {
		gotNames[index] = tool.Name
		input, ok := tool.InputSchema.(map[string]any)
		if !ok || input["type"] != "object" || input["additionalProperties"] != false || tool.OutputSchema == nil || tool.Description == "" || tool.Annotations == nil {
			t.Fatalf("tool %q schema incomplete: %#v", tool.Name, tool.InputSchema)
		}
	}
	slices.Sort(gotNames)
	if !slices.Equal(gotNames, wantNames) {
		t.Fatalf("tool names=%v want=%v", gotNames, wantNames)
	}
	called, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "status", Arguments: map[string]any{}})
	if err != nil || called.IsError || called.StructuredContent == nil {
		t.Fatalf("status result=%+v err=%v", called, err)
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
