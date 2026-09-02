package agentapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientRequiresLiteralLoopbackHTTPURL(t *testing.T) {
	for _, address := range []string{
		"https://127.0.0.1:8080", "http://localhost:8080", "http://192.0.2.1:8080",
		"http://127.0.0.1", "http://127.0.0.1:8080/path", "http://user@127.0.0.1:8080",
	} {
		if client, failure := NewClient(address); client != nil || failure == nil || failure.Error == nil || failure.Error.Code != "connection_failed" {
			t.Fatalf("NewClient(%q) = %#v, %#v", address, client, failure)
		}
	}
	if client, failure := NewClient("http://127.0.0.1:8080"); client == nil || failure != nil {
		t.Fatalf("literal loopback rejected: %#v, %#v", client, failure)
	}
}

func TestClientHandshakesBeforeEveryNonStatusOperation(t *testing.T) {
	var statusCalls, listCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/agent/v1/status", func(response http.ResponseWriter, _ *http.Request) {
		statusCalls++
		writeEnvelope(response, Envelope{Protocol: Protocol, OK: true, Context: Context{AuthorityState: "none"}, Result: map[string]any{"protocol": Protocol}})
	})
	mux.HandleFunc("GET /api/agent/v1/projects/list", func(response http.ResponseWriter, _ *http.Request) {
		listCalls++
		writeEnvelope(response, Envelope{Protocol: Protocol, OK: true, Context: Context{AuthorityState: "none"}, Result: map[string]any{"projects": []any{}}})
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	client, failure := NewClient(server.URL)
	if failure != nil {
		t.Fatalf("NewClient: %#v", failure)
	}
	if result := client.Call(context.Background(), "projects_list", nil); !result.OK {
		t.Fatalf("projects_list = %#v", result)
	}
	if statusCalls != 1 || listCalls != 1 {
		t.Fatalf("calls: status=%d list=%d", statusCalls, listCalls)
	}
}

func TestClientStopsAtIncompatibleHandshake(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "text/plain")
		_, _ = response.Write([]byte("not the agent protocol"))
	}))
	defer server.Close()
	client, failure := NewClient(server.URL)
	if failure != nil {
		t.Fatalf("NewClient: %#v", failure)
	}
	result := client.Call(context.Background(), "projects_list", nil)
	if result.OK || result.Error == nil || result.Error.Code != "incompatible_server" {
		t.Fatalf("result = %#v", result)
	}
}

func TestClientRejectsRedirectsBeforeMutationBodyCanLeaveConfiguredLoopback(t *testing.T) {
	targetCalls := 0
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		targetCalls++
	}))
	defer target.Close()

	source := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/api/agent/v1/status" {
			writeEnvelope(response, Envelope{Protocol: Protocol, OK: true})
			return
		}
		http.Redirect(response, request, target.URL+request.URL.Path, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	client, failure := NewClient(source.URL)
	if failure != nil {
		t.Fatalf("NewClient: %#v", failure)
	}
	result := client.Call(context.Background(), "project_create", ProjectCreateRequest{Name: "Must stay local"})
	if result.OK || result.Error == nil || result.Error.Code != "incompatible_server" {
		t.Fatalf("redirect result = %#v", result)
	}
	if targetCalls != 0 {
		t.Fatalf("redirect target received %d requests", targetCalls)
	}
}

func writeEnvelope(response http.ResponseWriter, envelope Envelope) {
	response.Header().Set("Content-Type", "application/json")
	_, _ = response.Write([]byte(`{"protocol":"` + envelope.Protocol + `","ok":true,"context":{"project":null,"accepted_revision":null,"authority_state":"none","pending_generation":null},"result":{}}`))
}
