package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestReviewComparisonRetainsContentAndRelationshipMultisetSemantics(t *testing.T) {
	before := []componentResponse{
		{ID: "source", Title: "Source", Description: "Before\n", Filename: "source.md", Relationships: []relationshipResponse{
			{TargetID: "target", Label: "calls"},
			{TargetID: "target", Label: "parallel"},
			{TargetID: "target", Label: "parallel"},
		}},
		{ID: "target", Title: "Target", Description: "Stable\n", Filename: "target.md"},
		{ID: "relationship-only", Title: "Relationship only", Description: "Stable\n", Filename: "relationship-only.md", Relationships: []relationshipResponse{{TargetID: "target", Label: "old"}}},
	}
	withChanges := []componentResponse{
		{ID: "source", Title: "Source", Description: "After\n", Filename: "source.md", Relationships: []relationshipResponse{
			{TargetID: "target", Label: "calls"},
			{TargetID: "target", Label: "parallel"},
			{TargetID: "target", Label: "parallel"},
			{TargetID: "target", Label: "parallel"},
			{TargetID: "target", Label: "  exact λ  "},
		}},
		{ID: "target", Title: "Target", Description: "Stable\n", Filename: "target.md"},
		{ID: "relationship-only", Title: "Relationship only", Description: "Stable\n", Filename: "relationship-only.md", Relationships: []relationshipResponse{{TargetID: "target", Label: "new"}}},
		{ID: "added", Title: "Added", Description: "New\n", Filename: "added.md"},
	}
	comparison := compareReviewProjections(before, withChanges)
	statuses := map[string]string{}
	for _, change := range comparison.Components {
		statuses[change.ComponentID] = change.Status
	}
	if statuses["source"] != "content_changed" || statuses["added"] != "added" || statuses["relationship-only"] != "" {
		t.Fatalf("component comparison = %+v", comparison.Components)
	}
	facts := map[string][]reviewRelationshipChangeResponse{}
	for _, change := range comparison.Relationships {
		facts[change.Status+"\x00"+change.SourceID+"\x00"+change.TargetID+"\x00"+change.Label] = append(
			facts[change.Status+"\x00"+change.SourceID+"\x00"+change.TargetID+"\x00"+change.Label], change,
		)
	}
	parallel := facts["added\x00source\x00target\x00parallel"]
	if len(parallel) != 1 || parallel[0].Occurrence != 3 ||
		len(facts["added\x00source\x00target\x00  exact λ  "]) != 1 ||
		len(facts["removed\x00relationship-only\x00target\x00old"]) != 1 ||
		len(facts["added\x00relationship-only\x00target\x00new"]) != 1 {
		t.Fatalf("relationship comparison = %+v", comparison.Relationships)
	}
}

func TestVisualReviewCaptureRemainsCoherentAcrossConcurrentInvalidation(t *testing.T) {
	for _, action := range []string{"mutation", "discard"} {
		t.Run(action, func(t *testing.T) {
			state, handler := newHandler(testOrigin, testUI(t), t.TempDir())
			base := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Capture"}))
			pending := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/add", observedComponentMutation(base, componentMutationRequest{
				DiagramID: base.RootDiagramID, Title: "Gateway", Description: "First generation.\n",
			})))
			componentID := pending.Changes.Components[0].ID

			body, err := json.Marshal(observedAction(pending))
			if err != nil {
				t.Fatal(err)
			}
			request := httptestRequest(http.MethodPost, "/api/architecture/review", body)
			writer := newBlockingResponseWriter()
			done := make(chan struct{})
			go func() {
				handler.ServeHTTP(writer, request)
				close(done)
			}()
			<-writer.writeStarted

			if action == "mutation" {
				invalidated := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/edit", observedComponentMutation(pending, componentMutationRequest{
					ComponentID: componentID, Description: "Second generation.\n", DescriptionChanged: true,
				})))
				if invalidated.Changes.Review != nil {
					t.Fatalf("mutation exposed invalidated review: %+v", invalidated.Changes.Review)
				}
			} else {
				invalidated := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/discard", observedAction(pending)))
				if invalidated.Changes != nil {
					t.Fatalf("discard retained changes: %+v", invalidated.Changes)
				}
			}

			close(writer.release)
			<-done
			var captured architectureResponse
			if err := json.Unmarshal(writer.body.Bytes(), &captured); err != nil {
				t.Fatalf("decode captured review: %v\n%s", err, writer.body.String())
			}
			if captured.Changes == nil || captured.Changes.Review == nil {
				t.Fatalf("captured response lost coherent review: %+v", captured)
			}
			review := captured.Changes.Review
			if review.Generation != 1 || review.BaseRevision != base.Revision || review.CandidateTree != review.WithChanges.Revision ||
				len(review.Before.Components) != 0 || len(review.WithChanges.Components) != 1 || review.WithChanges.Components[0].Description != "First generation.\n" {
				t.Fatalf("captured response mixed generations: %+v", review)
			}
			state.stateMutex.Lock()
			active := testActiveChangeSet(state)
			current := active != nil && active.review != nil && active.review.generation == active.generation
			state.stateMutex.Unlock()
			if current {
				t.Fatalf("%s retained the invalidated binding", action)
			}
			confirmation := postJSONRequest(t, handler, "/api/architecture/accept", acceptChangesRequest{
				ProjectSlug: base.ProjectSlug, StoreID: base.StoreID, ChangeSetID: review.ChangeSetID, BaseRevision: review.BaseRevision,
				CandidateTree: review.CandidateTree, Generation: review.Generation,
			})
			if confirmation.Code != http.StatusConflict {
				t.Fatalf("invalidated binding remained confirmable: status=%d body=%s", confirmation.Code, confirmation.Body.String())
			}
		})
	}
}

func httptestRequest(method, path string, body []byte) *http.Request {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Origin", testOrigin)
	request.Header.Set("Content-Type", "application/json")
	return request
}

type blockingResponseWriter struct {
	header       http.Header
	status       int
	body         bytes.Buffer
	writeStarted chan struct{}
	release      chan struct{}
	once         sync.Once
}

func newBlockingResponseWriter() *blockingResponseWriter {
	return &blockingResponseWriter{header: make(http.Header), writeStarted: make(chan struct{}), release: make(chan struct{})}
}

func (writer *blockingResponseWriter) Header() http.Header    { return writer.header }
func (writer *blockingResponseWriter) WriteHeader(status int) { writer.status = status }
func (writer *blockingResponseWriter) Write(value []byte) (int, error) {
	writer.once.Do(func() { close(writer.writeStarted) })
	<-writer.release
	return writer.body.Write(value)
}
func (writer *blockingResponseWriter) String() string { return strings.TrimSpace(writer.body.String()) }
