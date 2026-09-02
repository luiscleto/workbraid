// Package agentapi contains the versioned local wire contract and the thin
// loopback client shared by WorkBraid's CLI and MCP adapter. It owns no
// Architecture state and never opens application data.
package agentapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const Protocol = "workbraid-agent-v1"

type ProjectContext struct {
	StoreID string `json:"store_id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
}

type Context struct {
	Project           *ProjectContext `json:"project"`
	AcceptedRevision  *string         `json:"accepted_revision"`
	AuthorityState    string          `json:"authority_state"`
	PendingGeneration *uint64         `json:"pending_generation"`
}

type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

type Envelope struct {
	Protocol string  `json:"protocol"`
	OK       bool    `json:"ok"`
	Context  Context `json:"context"`
	Result   any     `json:"result,omitempty"`
	Error    *Error  `json:"error,omitempty"`
}

type StatePreconditions struct {
	StoreID           string  `json:"store_id" jsonschema:"Exact store UUID returned by the current WorkBraid server."`
	AcceptedRevision  string  `json:"accepted_revision" jsonschema:"Exact loaded accepted revision the caller inspected."`
	PendingGeneration *uint64 `json:"pending_generation" jsonschema:"Exact inspected pending generation, or null when no pending set existed."`
}

type ProjectCreateRequest struct {
	Name string `json:"name" jsonschema:"Human-readable project name; WorkBraid generates the slug and identities."`
}

type ProjectOpenRequest struct {
	Slug string `json:"slug" jsonschema:"Exact catalog slug used only to locate and select a project."`
}

type ProjectCloseRequest struct {
	StoreID string `json:"store_id" jsonschema:"Exact store UUID of the process-wide current project to close."`
}

type ArchitectureRefreshRequest struct {
	StoreID          string `json:"store_id" jsonschema:"Exact store UUID of the current project."`
	AcceptedRevision string `json:"accepted_revision" jsonschema:"Exact loaded revision inspected before explicit Refresh."`
}

type ArchitectureUpdateRequest struct {
	StoreID       string `json:"store_id" jsonschema:"Exact current store UUID."`
	BaseRevision  string `json:"base_revision" jsonschema:"Exact base commit returned by changes_review."`
	CandidateTree string `json:"candidate_tree" jsonschema:"Exact candidate tree returned by changes_review."`
	Generation    uint64 `json:"generation" jsonschema:"Exact pending generation returned by changes_review."`
}

type ChangesReviewRequest struct {
	StatePreconditions
	Generation uint64 `json:"generation" jsonschema:"Exact numeric generation being reviewed; must match pending_generation."`
}

type ChangesDiscardRequest struct {
	StoreID    string `json:"store_id" jsonschema:"Exact current store UUID."`
	Generation uint64 `json:"generation" jsonschema:"Exact pending generation whose complete change set will be discarded."`
}

type ComponentCreateRequest struct {
	StatePreconditions
	Title       string  `json:"title" jsonschema:"Structured Component title."`
	Description string  `json:"description" jsonschema:"Exact Markdown Description text."`
	DiagramID   *string `json:"diagram_id" jsonschema:"Explicit home Diagram ID, or null only for deliberate root fallback."`
}

type ComponentEditRequest struct {
	StatePreconditions
	ComponentID string  `json:"component_id" jsonschema:"Stable Component ID to edit."`
	Title       *string `json:"title" jsonschema:"New structured title when changed; omit otherwise."`
	Description *string `json:"description" jsonschema:"New exact Markdown Description when changed; omit otherwise."`
}

type ComponentMoveHomeRequest struct {
	StatePreconditions
	ComponentID string `json:"component_id" jsonschema:"Stable Component ID whose home will move."`
	DiagramID   string `json:"diagram_id" jsonschema:"Stable eligible destination Diagram ID."`
}

type RelationshipAddRequest struct {
	StatePreconditions
	SourceID string `json:"source_id" jsonschema:"Stable source Component ID owning the outgoing row."`
	TargetID string `json:"target_id" jsonschema:"Exact raw target value to append; validation is candidate-wide."`
	Label    string `json:"label" jsonschema:"Exact raw label text to append; validation is candidate-wide."`
}

type RelationshipEditRequest struct {
	StatePreconditions
	SourceID    string `json:"source_id" jsonschema:"Stable source Component ID owning the row."`
	OldTargetID string `json:"old_target_id" jsonschema:"Exact raw old target, including an empty or malformed value."`
	OldLabel    string `json:"old_label" jsonschema:"Exact raw old label, including empty or whitespace-only text."`
	Occurrence  int    `json:"occurrence" jsonschema:"One-based occurrence among identical raw old target and label pairs."`
	TargetID    string `json:"target_id" jsonschema:"Exact replacement target text."`
	Label       string `json:"label" jsonschema:"Exact replacement label text."`
}

type RelationshipRemoveRequest struct {
	StatePreconditions
	SourceID   string `json:"source_id" jsonschema:"Stable source Component ID owning the row."`
	TargetID   string `json:"target_id" jsonschema:"Exact raw target selector, including an empty or malformed value."`
	Label      string `json:"label" jsonschema:"Exact raw label selector, including empty or whitespace-only text."`
	Occurrence int    `json:"occurrence" jsonschema:"One-based occurrence among identical raw target and label pairs."`
}

type DiagramCreateDetailRequest struct {
	StatePreconditions
	ComponentID string `json:"component_id" jsonschema:"Stable home Component ID that will anchor the detail Diagram."`
	Title       string `json:"title" jsonschema:"Authored Diagram title."`
}

type DiagramEditTitleRequest struct {
	StatePreconditions
	DiagramID string `json:"diagram_id" jsonschema:"Stable Diagram ID to retitle."`
	Title     string `json:"title" jsonschema:"New authored Diagram title."`
}

type DiagramComponentRequest struct {
	StatePreconditions
	DiagramID   string `json:"diagram_id" jsonschema:"Stable target Diagram ID."`
	ComponentID string `json:"component_id" jsonschema:"Stable Component ID to show by reference or stop showing."`
}

var operationPaths = map[string]string{
	"status":                         "/api/agent/v1/status",
	"projects_list":                  "/api/agent/v1/projects/list",
	"project_current":                "/api/agent/v1/projects/current",
	"project_create":                 "/api/agent/v1/projects/create",
	"project_open":                   "/api/agent/v1/projects/open",
	"project_close":                  "/api/agent/v1/projects/close",
	"architecture_inspect":           "/api/agent/v1/architecture/inspect",
	"architecture_refresh":           "/api/agent/v1/architecture/refresh",
	"architecture_update":            "/api/agent/v1/architecture/update",
	"changes_inspect":                "/api/agent/v1/changes/inspect",
	"changes_review":                 "/api/agent/v1/changes/review",
	"changes_discard":                "/api/agent/v1/changes/discard",
	"component_create":               "/api/agent/v1/components/create",
	"component_edit":                 "/api/agent/v1/components/edit",
	"component_move_home":            "/api/agent/v1/components/move-home",
	"relationship_add":               "/api/agent/v1/relationships/add",
	"relationship_edit":              "/api/agent/v1/relationships/edit",
	"relationship_remove":            "/api/agent/v1/relationships/remove",
	"diagram_create_detail":          "/api/agent/v1/diagrams/create-detail",
	"diagram_edit_title":             "/api/agent/v1/diagrams/edit-title",
	"diagram_show_component":         "/api/agent/v1/diagrams/show-component",
	"diagram_stop_showing_component": "/api/agent/v1/diagrams/stop-showing-component",
}

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(serverURL string) (*Client, *Envelope) {
	parsed, err := url.Parse(serverURL)
	if err != nil || parsed.Scheme != "http" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Host == "" {
		return nil, failure("connection_failed", "The WorkBraid server address must be a plain HTTP literal-loopback URL.", nil)
	}
	host := parsed.Hostname()
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() || parsed.Port() == "" {
		return nil, failure("connection_failed", "The WorkBraid server address must use a literal loopback IP and port.", nil)
	}
	return &Client{
		baseURL: strings.TrimSuffix(parsed.String(), "/"),
		http: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (client *Client) Call(ctx context.Context, operation string, input any) Envelope {
	if operation != "status" {
		handshake := client.call(ctx, "status", nil)
		if !handshake.OK {
			return handshake
		}
	}
	return client.call(ctx, operation, input)
}

func (client *Client) call(ctx context.Context, operation string, input any) Envelope {
	path, exists := operationPaths[operation]
	if !exists {
		return *failure("invalid_request", "That WorkBraid agent operation is not available.", map[string]any{"operation": operation})
	}
	method := http.MethodPost
	var body io.Reader
	if operation == "status" || operation == "projects_list" || operation == "project_current" || operation == "architecture_inspect" || operation == "changes_inspect" {
		method = http.MethodGet
	} else {
		encoded, err := json.Marshal(input)
		if err != nil {
			return *failure("invalid_request", "The request could not be encoded.", nil)
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, body)
	if err != nil {
		return *failure("connection_failed", "The WorkBraid server request could not be created.", nil)
	}
	if method == http.MethodPost {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.http.Do(request)
	if err != nil {
		details := map[string]any{}
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			details["timeout"] = true
		}
		return *failure("connection_failed", "The running WorkBraid server could not be reached.", details)
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, 8<<20)
	decoder := json.NewDecoder(limited)
	decoder.UseNumber()
	var envelope Envelope
	if err := decoder.Decode(&envelope); err != nil {
		return *failure("incompatible_server", "The running server did not return the WorkBraid agent protocol.", nil)
	}
	if envelope.Protocol != Protocol {
		return *failure("incompatible_server", "The running server uses an incompatible WorkBraid agent protocol.", map[string]any{"received_protocol": envelope.Protocol})
	}
	if envelope.Error != nil && envelope.Error.Details == nil {
		envelope.Error.Details = map[string]any{}
	}
	return envelope
}

func failure(code, message string, details map[string]any) *Envelope {
	if details == nil {
		details = map[string]any{}
	}
	return &Envelope{
		Protocol: Protocol,
		OK:       false,
		Context:  Context{AuthorityState: "none"},
		Error:    &Error{Code: code, Message: message, Details: details},
	}
}

func Failure(code, message string, details map[string]any) Envelope {
	return *failure(code, message, details)
}

func OperationNames() []string {
	names := make([]string, 0, len(operationPaths))
	for name := range operationPaths {
		names = append(names, name)
	}
	return names
}

func (envelope Envelope) Validate() error {
	if envelope.Protocol != Protocol {
		return fmt.Errorf("unexpected protocol %q", envelope.Protocol)
	}
	if envelope.OK && envelope.Error != nil {
		return errors.New("successful envelope contains an error")
	}
	if !envelope.OK && envelope.Error == nil {
		return errors.New("failed envelope has no error")
	}
	return nil
}
