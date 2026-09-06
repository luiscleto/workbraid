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

	"workbraid/internal/architecture"
)

const Protocol = "workbraid-agent-v2"

type ProjectContext struct {
	StoreID string `json:"store_id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
}

type Context struct {
	Project          *ProjectContext `json:"project"`
	AcceptedRevision *string         `json:"accepted_revision"`
	AuthorityState   string          `json:"authority_state"`
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
	StoreID     string `json:"store_id" jsonschema:"Exact store UUID returned by the current WorkBraid server."`
	ChangeSetID string `json:"change_set_id" jsonschema:"Exact active change-set UUID returned by WorkBraid."`
	Generation  uint64 `json:"generation" jsonschema:"Exact generation inspected for that change set."`
}

type ReconciliationInputs struct {
	StatePreconditions
	ChangeSetState   string `json:"change_set_state" jsonschema:"Exact active state commit S returned by inspect. Any change invalidates these inputs, even without a generation change."`
	BaseRevision     string `json:"base_revision" jsonschema:"Exact proposal base B returned by inspect."`
	CandidateTree    string `json:"candidate_tree" jsonschema:"Exact valid proposal candidate P returned by inspect; no review is required."`
	AcceptedRevision string `json:"accepted_revision" jsonschema:"Exact known-current Accepted A returned by inspect or explicit Refresh."`
}

type ReconciliationPreviewRequest struct {
	ReconciliationInputs
	Resolutions []architecture.ReconciliationResolution `json:"resolutions,omitempty" jsonschema:"Optional complete tentative choices, addressed by the exact typed conflict locators returned by preview. Check is non-mutating."`
}

type ReconciliationApplyRequest struct {
	ReconciliationInputs
	Resolutions []architecture.ReconciliationResolution `json:"resolutions" jsonschema:"Required complete explicit choices; send an empty array for a fully automatic result. Old S after a successful Apply is a state mismatch; inspect, never replay blindly."`
}

type DiagramParentOptionsRequest struct {
	StatePreconditions
	DiagramID string `json:"diagram_id" jsonschema:"Exact non-root child Diagram UUID in the complete proposal."`
}

type DiagramReassignDetailRequest struct {
	StatePreconditions
	DiagramID         string `json:"diagram_id" jsonschema:"Exact non-root Diagram UUID whose parent link will move."`
	AnchorComponentID string `json:"anchor_component_id" jsonschema:"Eligible destination Component UUID from diagram_parent_options. Its home will own the child link."`
}

// RequiresExactGeneration marks requests whose generation field must be
// present even when its valid value is zero.
func (StatePreconditions) RequiresExactGeneration() {}

type ChangeSetsListRequest struct {
	StoreID string `json:"store_id" jsonschema:"Exact store UUID whose active, applied, and unavailable change sets will be listed."`
}

type ChangeSetCreateRequest struct {
	StoreID          string  `json:"store_id" jsonschema:"Exact current store UUID."`
	AcceptedRevision string  `json:"accepted_revision" jsonschema:"Exact current Accepted revision inspected before creation."`
	Name             *string `json:"name,omitempty" jsonschema:"Optional human-readable active name; omit to generate one."`
}

type ChangeSetInspectRequest struct {
	StoreID     string `json:"store_id" jsonschema:"Exact current store UUID."`
	ChangeSetID string `json:"change_set_id" jsonschema:"Exact active or applied change-set UUID."`
}

type ChangeSetRenameRequest struct {
	StatePreconditions
	Name string `json:"name" jsonschema:"New trimmed, non-empty, single-line active name."`
}

type ChangeSetEditProposalRequest struct {
	StatePreconditions
	ProposalMarkdown string `json:"proposal_markdown" jsonschema:"Exact UTF-8 Markdown proposal document, which may be empty."`
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
	ChangeSetID   string `json:"change_set_id" jsonschema:"Exact reviewed active change-set UUID."`
	BaseRevision  string `json:"base_revision" jsonschema:"Exact base commit returned by change_set_review."`
	CandidateTree string `json:"candidate_tree" jsonschema:"Exact candidate tree returned by change_set_review."`
	Generation    uint64 `json:"generation" jsonschema:"Exact change-set generation returned by change_set_review."`
}

func (ArchitectureUpdateRequest) RequiresExactGeneration() {}

type ChangeSetReviewRequest struct {
	StatePreconditions
}

type ChangeSetDiscardRequest struct {
	StatePreconditions
}

type ReviewSubmissionsListRequest struct {
	StoreID     string `json:"store_id" jsonschema:"Exact current store UUID."`
	ChangeSetID string `json:"change_set_id" jsonschema:"Exact Change Set UUID whose submitted reviews will be listed."`
}

type ReviewSubmissionInspectRequest struct {
	StoreID     string `json:"store_id" jsonschema:"Exact current store UUID."`
	ChangeSetID string `json:"change_set_id" jsonschema:"Exact reviewed Change Set UUID."`
	ReviewID    string `json:"review_id" jsonschema:"Exact immutable review-submission UUID."`
}

type ReviewAnchor struct {
	Kind              string `json:"kind" jsonschema:"Anchor kind: proposal, proposal_markdown, component, component_markdown, diagram, composition, or relationship."`
	Side              string `json:"side,omitempty" jsonschema:"Exact Architecture side: before or with_changes, when required by the anchor kind."`
	ComponentID       string `json:"component_id,omitempty" jsonschema:"Stable Component UUID for Component or composition anchors."`
	DiagramID         string `json:"diagram_id,omitempty" jsonschema:"Stable Diagram UUID for Diagram or composition anchors."`
	Aspect            string `json:"aspect,omitempty" jsonschema:"Composition fact: home, reference, or detail."`
	DetailDiagramID   string `json:"detail_diagram_id,omitempty" jsonschema:"Exact child Diagram UUID, required only for a detail composition anchor."`
	SourceComponentID string `json:"source_component_id,omitempty" jsonschema:"Stable source Component UUID for a Relationship anchor."`
	TargetComponentID string `json:"target_component_id,omitempty" jsonschema:"Stable target Component UUID for a Relationship anchor."`
	Label             string `json:"label,omitempty" jsonschema:"Exact Relationship label for a Relationship anchor."`
	Occurrence        int    `json:"occurrence,omitempty" jsonschema:"One-based occurrence among identical Relationship facts."`
	StartLine         int    `json:"start_line,omitempty" jsonschema:"One-based inclusive source start line for Markdown anchors."`
	EndLine           int    `json:"end_line,omitempty" jsonschema:"One-based inclusive source end line for Markdown anchors."`
}

type ReviewCommentInput struct {
	Body   string       `json:"body" jsonschema:"Exact non-empty Markdown comment body."`
	Anchor ReviewAnchor `json:"anchor" jsonschema:"One exact typed reviewed-snapshot anchor."`
}

type ReviewSubmissionSubmitRequest struct {
	StoreID       string               `json:"store_id" jsonschema:"Exact current store UUID."`
	ChangeSetID   string               `json:"change_set_id" jsonschema:"Exact active Change Set UUID."`
	ReviewedState string               `json:"reviewed_state" jsonschema:"Exact reviewed state commit returned by change_set_review."`
	BaseRevision  string               `json:"base_revision" jsonschema:"Exact base commit returned by change_set_review."`
	CandidateTree string               `json:"candidate_tree" jsonschema:"Exact candidate tree returned by change_set_review."`
	Generation    uint64               `json:"generation" jsonschema:"Exact generation returned by change_set_review."`
	Verdict       string               `json:"verdict" jsonschema:"Informational verdict: comment, approve, or request_changes."`
	Author        string               `json:"author" jsonschema:"Descriptive untrusted single-line author label."`
	Body          string               `json:"body" jsonschema:"Optional exact Markdown overall review body."`
	Comments      []ReviewCommentInput `json:"comments" jsonschema:"Immutable comments with exact typed anchors."`
}

func (ReviewSubmissionSubmitRequest) RequiresExactGeneration() {}

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
	"change_set_reconcile_preview":   "/api/agent/v2/change-sets/reconcile-preview",
	"change_set_reconcile_apply":     "/api/agent/v2/change-sets/reconcile-apply",
	"status":                         "/api/agent/v2/status",
	"projects_list":                  "/api/agent/v2/projects/list",
	"project_current":                "/api/agent/v2/projects/current",
	"project_create":                 "/api/agent/v2/projects/create",
	"project_open":                   "/api/agent/v2/projects/open",
	"project_close":                  "/api/agent/v2/projects/close",
	"architecture_inspect":           "/api/agent/v2/architecture/inspect",
	"architecture_refresh":           "/api/agent/v2/architecture/refresh",
	"architecture_update":            "/api/agent/v2/architecture/update",
	"change_sets_list":               "/api/agent/v2/change-sets/list",
	"change_set_create":              "/api/agent/v2/change-sets/create",
	"change_set_inspect":             "/api/agent/v2/change-sets/inspect",
	"change_set_rename":              "/api/agent/v2/change-sets/rename",
	"change_set_edit_proposal":       "/api/agent/v2/change-sets/edit-proposal",
	"change_set_review":              "/api/agent/v2/change-sets/review",
	"change_set_discard":             "/api/agent/v2/change-sets/discard",
	"review_submissions_list":        "/api/agent/v2/review-submissions/list",
	"review_submission_inspect":      "/api/agent/v2/review-submissions/inspect",
	"review_submission_submit":       "/api/agent/v2/review-submissions/submit",
	"component_create":               "/api/agent/v2/components/create",
	"component_edit":                 "/api/agent/v2/components/edit",
	"component_move_home":            "/api/agent/v2/components/move-home",
	"relationship_add":               "/api/agent/v2/relationships/add",
	"relationship_edit":              "/api/agent/v2/relationships/edit",
	"relationship_remove":            "/api/agent/v2/relationships/remove",
	"diagram_parent_options":         "/api/agent/v2/diagrams/parent-options",
	"diagram_reassign_detail":        "/api/agent/v2/diagrams/reassign-detail",
	"diagram_create_detail":          "/api/agent/v2/diagrams/create-detail",
	"diagram_edit_title":             "/api/agent/v2/diagrams/edit-title",
	"diagram_show_component":         "/api/agent/v2/diagrams/show-component",
	"diagram_stop_showing_component": "/api/agent/v2/diagrams/stop-showing-component",
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
	envelope := client.call(ctx, operation, input)
	if operation == "change_set_review" && envelope.OK && envelope.Context.Project != nil {
		request, requestOK := input.(ChangeSetReviewRequest)
		result, resultOK := envelope.Result.(map[string]any)
		if requestOK && resultOK {
			result["review_url"] = client.baseURL + "/projects/" + url.PathEscape(envelope.Context.Project.Slug) + "/proposals/" + url.PathEscape(request.ChangeSetID) + "/review"
		}
	}
	if envelope.OK && envelope.Context.Project != nil {
		result, resultOK := envelope.Result.(map[string]any)
		if resultOK {
			changeSetID, reviewID := "", ""
			switch request := input.(type) {
			case ReviewSubmissionsListRequest:
				if reviews, ok := result["reviews"].([]any); ok {
					for _, value := range reviews {
						if review, ok := value.(map[string]any); ok {
							if id, ok := review["id"].(string); ok {
								review["review_url"] = client.submittedReviewURL(envelope.Context.Project.Slug, request.ChangeSetID, id)
							}
						}
					}
				}
			case ReviewSubmissionInspectRequest:
				changeSetID, reviewID = request.ChangeSetID, request.ReviewID
			case ReviewSubmissionSubmitRequest:
				changeSetID = request.ChangeSetID
				reviewID, _ = result["id"].(string)
			}
			if changeSetID != "" && reviewID != "" {
				result["review_url"] = client.submittedReviewURL(envelope.Context.Project.Slug, changeSetID, reviewID)
			}
		}
	}
	return envelope
}

func (client *Client) submittedReviewURL(slug, changeSetID, reviewID string) string {
	return client.baseURL + "/projects/" + url.PathEscape(slug) + "/proposals/" + url.PathEscape(changeSetID) + "/reviews/" + url.PathEscape(reviewID)
}

func (client *Client) call(ctx context.Context, operation string, input any) Envelope {
	path, exists := operationPaths[operation]
	if !exists {
		return *failure("invalid_request", "That WorkBraid agent operation is not available.", map[string]any{"operation": operation})
	}
	method := http.MethodPost
	var body io.Reader
	if operation == "status" || operation == "projects_list" || operation == "project_current" || operation == "architecture_inspect" {
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
