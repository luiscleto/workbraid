package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"workbraid/internal/architecture"
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

func TestDiagramAppearanceReviewCarriesExactAnchorSideAndDetail(t *testing.T) {
	root, child := "root", "child"
	before := []diagramResponse{{ID: root, Filename: "root.yaml", Appearances: []diagramAppearanceResponse{
		{ComponentID: "anchor", Role: "home", DetailDiagramID: child},
		{ComponentID: "removed", Role: "reference"},
	}}}
	withChanges := []diagramResponse{{ID: root, Filename: "root.yaml", Appearances: []diagramAppearanceResponse{
		{ComponentID: "anchor", Role: "home"},
		{ComponentID: "added", Role: "reference"},
	}}}
	_, appearances := compareDiagramProjections(before, withChanges)
	byComponent := make(map[string]reviewAppearanceChangeResponse, len(appearances))
	for _, appearance := range appearances {
		byComponent[appearance.ComponentID] = appearance
	}
	if got := byComponent["anchor"]; got.Status != "detail_changed" || got.Side != "before" || got.DetailDiagramID != child {
		t.Fatalf("removed detail link = %+v", got)
	}
	if got := byComponent["removed"]; got.Status != "removed" || got.Side != "before" || got.DetailDiagramID != "" {
		t.Fatalf("removed appearance = %+v", got)
	}
	if got := byComponent["added"]; got.Status != "added" || got.Side != "with_changes" || got.DetailDiagramID != "" {
		t.Fatalf("added appearance = %+v", got)
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

func TestRepeatedReviewKeepsExactStateAndSubmissionNeedsNoAcceptedObservation(t *testing.T) {
	dataDirectory := t.TempDir()
	state, handler := newHandler(testOrigin, testUI(t), dataDirectory)
	base := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Shared review"}))
	pending := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/add", observedComponentMutation(base, componentMutationRequest{
		DiagramID: base.RootDiagramID, Title: "Gateway", Description: "# Detail\n",
	})))
	first := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/review", observedAction(pending)))
	if first.Changes == nil || first.Changes.Review == nil || first.Changes.Review.ReviewedState == "" {
		t.Fatalf("first review = %+v", first.Changes)
	}
	firstState := first.Changes.Review.ReviewedState
	second := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/review", observedAction(first)))
	if second.Changes == nil || second.Changes.Review == nil || second.Changes.Review.ReviewedState != firstState {
		t.Fatalf("second review = %+v", second.Changes)
	}
	state.stateMutex.Lock()
	if state.changeSets[first.Changes.ID].refObject != firstState {
		t.Fatalf("repeated review rewrote active ref")
	}
	state.stateMutex.Unlock()
	_, restarted := newHandler(testOrigin, testUI(t), dataDirectory)
	reopened := decodeArchitectureResponse(t, postJSONRequest(t, restarted, "/api/projects/open", map[string]any{"project_slug": first.ProjectSlug}))
	selectActiveChangeSetForTest(&reopened, first.Changes.ID)
	repeatedAfterRestart := decodeArchitectureResponse(t, postJSONRequest(t, restarted, "/api/architecture/review", observedAction(reopened)))
	if repeatedAfterRestart.Changes == nil || repeatedAfterRestart.Changes.Review == nil || repeatedAfterRestart.Changes.Review.ReviewedState != firstState {
		t.Fatalf("review after restart rewrote active ref: %+v", repeatedAfterRestart.Changes)
	}
	state.stateMutex.Lock()
	// Model an operational Refresh failure after the exact review was already
	// bound. Feedback is immutable and does not need a new Accepted decision.
	state.acceptedIndeterminate = true
	state.stateMutex.Unlock()
	submitted := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/review-submissions/submit", reviewSubmissionRequest{
		ProjectSlug: first.ProjectSlug, StoreID: first.StoreID, ChangeSetID: first.Changes.ID, ReviewedState: firstState,
		BaseRevision: first.Changes.Review.BaseRevision, CandidateTree: first.Changes.Review.CandidateTree, Generation: first.Changes.Review.Generation,
		Verdict: "approve", Author: "Second reviewer", Comments: []architecture.ReviewCommentInput{},
	}))
	if submitted.ActionReviewID == "" || submitted.SubmittedReview == nil {
		t.Fatalf("submitted = %+v", submitted)
	}
	if submitted.ActionError != errorRefreshFailed {
		t.Fatalf("indeterminate submission hid authority context: %+v", submitted)
	}
	state.stateMutex.Lock()
	if state.changeSets[first.Changes.ID].refObject != firstState || state.changeSets[first.Changes.ID].generation != first.Changes.Generation || !state.acceptedIndeterminate {
		t.Fatalf("submission changed authority: %+v", state.changeSets[first.Changes.ID])
	}
	state.acceptedIndeterminate = false
	state.loadedStale = true
	state.stateMutex.Unlock()
	knownNonCurrent := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/review-submissions/submit", reviewSubmissionRequest{
		ProjectSlug: first.ProjectSlug, StoreID: first.StoreID, ChangeSetID: first.Changes.ID, ReviewedState: firstState,
		BaseRevision: first.Changes.Review.BaseRevision, CandidateTree: first.Changes.Review.CandidateTree, Generation: first.Changes.Review.Generation,
		Verdict: "request_changes", Author: "Third reviewer", Comments: []architecture.ReviewCommentInput{},
	}))
	if knownNonCurrent.ActionReviewID == "" || !knownNonCurrent.Stale {
		t.Fatalf("known-non-current submission hid authority context: %+v", knownNonCurrent)
	}
	state.stateMutex.Lock()
	defer state.stateMutex.Unlock()
	if state.changeSets[first.Changes.ID].refObject != firstState || len(state.reviews) != 2 {
		t.Fatalf("multiple submissions changed proposal or replaced feedback: state=%s reviews=%d", state.changeSets[first.Changes.ID].refObject, len(state.reviews))
	}
}

func TestReviewSubmissionRaceWithProposalMutationIsExactOrInvalidated(t *testing.T) {
	state, handler := newHandler(testOrigin, testUI(t), t.TempDir())
	base := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Review race"}))
	pending := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/add", observedComponentMutation(base, componentMutationRequest{
		DiagramID: base.RootDiagramID, Title: "Gateway",
	})))
	reviewed := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/review", observedAction(pending)))
	review := reviewed.Changes.Review
	submitBody, err := json.Marshal(reviewSubmissionRequest{
		ProjectSlug: reviewed.ProjectSlug, StoreID: reviewed.StoreID, ChangeSetID: reviewed.Changes.ID, ReviewedState: review.ReviewedState,
		BaseRevision: review.BaseRevision, CandidateTree: review.CandidateTree, Generation: review.Generation,
		Verdict: "approve", Author: "Racing reviewer", Comments: []architecture.ReviewCommentInput{},
	})
	if err != nil {
		t.Fatal(err)
	}
	editBody, err := json.Marshal(changeSetEditRequest{
		ProjectSlug: reviewed.ProjectSlug, StoreID: reviewed.StoreID, ChangeSetID: reviewed.Changes.ID,
		Generation: reviewed.Changes.Generation, Proposal: "# Changed while reviewing\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	submitResponse, editResponse := httptest.NewRecorder(), httptest.NewRecorder()
	start := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		<-start
		handler.ServeHTTP(submitResponse, httptestRequest(http.MethodPost, "/api/architecture/review-submissions/submit", submitBody))
	}()
	go func() {
		defer wait.Done()
		<-start
		handler.ServeHTTP(editResponse, httptestRequest(http.MethodPost, "/api/architecture/change-sets/proposal", editBody))
	}()
	close(start)
	wait.Wait()
	if editResponse.Code != http.StatusOK {
		t.Fatalf("proposal mutation status=%d body=%s", editResponse.Code, editResponse.Body.String())
	}
	if submitResponse.Code != http.StatusCreated && submitResponse.Code != http.StatusConflict {
		t.Fatalf("submission status=%d body=%s", submitResponse.Code, submitResponse.Body.String())
	}
	state.stateMutex.Lock()
	defer state.stateMutex.Unlock()
	current := state.changeSets[reviewed.Changes.ID]
	if current == nil || current.generation != reviewed.Changes.Generation+1 || current.review != nil || current.refObject == review.ReviewedState || state.loadedSnapshot.Revision() != base.Revision {
		t.Fatalf("race changed mixed authority: current=%+v accepted=%s", current, state.loadedSnapshot.Revision())
	}
	if submitResponse.Code == http.StatusCreated {
		if len(state.reviews) != 1 {
			t.Fatalf("successful exact-old submission count=%d", len(state.reviews))
		}
		for _, submission := range state.reviews {
			if submission.ReviewedState != review.ReviewedState || submission.Binding.Generation != review.Generation {
				t.Fatalf("successful submission mixed generations: %+v", submission)
			}
		}
	} else if len(state.reviews) != 0 {
		t.Fatalf("invalidated submission created reviews: %+v", state.reviews)
	}
}

func TestSubmittedReviewRouteSurvivesProposalDiscardAndRestart(t *testing.T) {
	dataDirectory := t.TempDir()
	_, handler := newHandler(testOrigin, testUI(t), dataDirectory)
	base := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Retained feedback"}))
	pending := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/add", observedComponentMutation(base, componentMutationRequest{
		DiagramID: base.RootDiagramID, Title: "Gateway", Description: "Gateway docs.\n",
	})))
	reviewed := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/review", observedAction(pending)))
	review := reviewed.Changes.Review
	submitted := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/review-submissions/submit", reviewSubmissionRequest{
		ProjectSlug: reviewed.ProjectSlug, StoreID: reviewed.StoreID, ChangeSetID: reviewed.Changes.ID, ReviewedState: review.ReviewedState,
		BaseRevision: review.BaseRevision, CandidateTree: review.CandidateTree, Generation: review.Generation,
		Verdict: "comment", Author: "Reviewer", Body: "Keep this exact context.\n",
	}))
	if submitted.ActionReviewID == "" {
		t.Fatalf("submission=%+v", submitted)
	}
	discarded := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/discard", observedAction(reviewed)))
	if discarded.Changes != nil {
		t.Fatalf("discard retained active proposal: %+v", discarded.Changes)
	}

	_, restarted := newHandler(testOrigin, testUI(t), dataDirectory)
	opened := decodeArchitectureResponse(t, postJSONRequest(t, restarted, "/api/projects/open", map[string]any{"project_slug": base.ProjectSlug}))
	if len(opened.ChangeSets) != 0 || len(opened.ReviewSubmissions) != 1 {
		t.Fatalf("restart context=%+v reviews=%+v", opened.ChangeSets, opened.ReviewSubmissions)
	}
	inspected := decodeArchitectureResponse(t, postJSONRequest(t, restarted, "/api/architecture/review-submissions/inspect", reviewSubmissionInspectRequest{
		ProjectSlug: base.ProjectSlug, StoreID: base.StoreID, ChangeSetID: reviewed.Changes.ID, ReviewID: submitted.ActionReviewID,
	}))
	if inspected.SubmittedReview == nil || inspected.SubmittedReview.Lifecycle != "no_longer_active" || inspected.SubmittedReview.Body != "Keep this exact context.\n" || inspected.Changes == nil || !inspected.Changes.ReadOnly {
		t.Fatalf("retained review=%+v changes=%+v", inspected.SubmittedReview, inspected.Changes)
	}
	for _, value := range inspected.ChangeSets {
		if value.ID == reviewed.Changes.ID {
			t.Fatalf("discarded proposal was recreated in selector: %+v", value)
		}
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
