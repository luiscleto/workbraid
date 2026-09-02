package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
	createdComponent, err := first.CallTool(ctx, &mcp.CallToolParams{Name: "component_create", Arguments: map[string]any{
		"store_id": created.Context.Project.StoreID, "accepted_revision": *created.Context.AcceptedRevision, "pending_generation": nil,
		"title": "Gateway", "description": "Routes requests.\n", "diagram_id": rootID,
	}})
	if err != nil || createdComponent.IsError {
		t.Fatalf("component_create: result=%+v err=%v", createdComponent, err)
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
	if contextValue["pending_generation"] != float64(1) {
		t.Fatalf("second bridge pending context = %#v", contextValue)
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
	for _, command := range []string{"project create", "architecture inspect", "changes review", "architecture update", "relationship edit", "diagram stop-showing-component", "workbraid [--server <loopback-url>] mcp"} {
		if !bytes.Contains(first.Bytes(), []byte(command)) {
			t.Fatalf("skill does not document %q", command)
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
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "workbraid", Version: "test"}, nil)
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
	for index, tool := range listed.Tools {
		gotNames[index] = tool.Name
		input, ok := tool.InputSchema.(map[string]any)
		if !ok || input["type"] != "object" || input["additionalProperties"] != false || tool.OutputSchema == nil || tool.Description == "" || tool.Annotations == nil {
			t.Fatalf("tool %q schema/description/annotations incomplete: input=%#v output=%#v", tool.Name, tool.InputSchema, tool.OutputSchema)
		}
	}
	slices.Sort(gotNames)
	if !slices.Equal(gotNames, wantNames) {
		t.Fatalf("tool names = %v, want %v", gotNames, wantNames)
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
