package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"workbraid/internal/agentapi"
	"workbraid/internal/architecture"
)

func postAgent(t *testing.T, handler http.Handler, path string, value any) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeAgentEnvelope(t *testing.T, response *httptest.ResponseRecorder) agentapi.Envelope {
	t.Helper()
	var envelope agentapi.Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode agent response: %v\n%s", err, response.Body.String())
	}
	if err := envelope.Validate(); err != nil {
		t.Fatalf("invalid envelope: %v\n%s", err, response.Body.String())
	}
	return envelope
}

func getAgent(t *testing.T, handler http.Handler, path string) agentapi.Envelope {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return decodeAgentEnvelope(t, response)
}

func resultMap(t *testing.T, envelope agentapi.Envelope) map[string]any {
	t.Helper()
	value, ok := envelope.Result.(map[string]any)
	if !ok {
		t.Fatalf("result = %#v, want object", envelope.Result)
	}
	return value
}

func changeSetState(t *testing.T, envelope agentapi.Envelope) agentapi.StatePreconditions {
	t.Helper()
	result := resultMap(t, envelope)
	id, idOK := result["id"].(string)
	if !idOK {
		id, idOK = result["change_set_id"].(string)
	}
	generation, generationOK := result["generation"].(float64)
	if !idOK || !generationOK {
		t.Fatalf("change-set result lacks identity/generation: %#v", result)
	}
	if envelope.Context.Project == nil {
		t.Fatalf("change-set result lacks project context: %+v", envelope)
	}
	return agentapi.StatePreconditions{StoreID: envelope.Context.Project.StoreID, ChangeSetID: id, Generation: uint64(generation)}
}

func createAgentChangeSet(t *testing.T, handler http.Handler, storeID, revision, name string) agentapi.Envelope {
	t.Helper()
	envelope := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/create", agentapi.ChangeSetCreateRequest{
		StoreID: storeID, AcceptedRevision: revision, Name: &name,
	}))
	if !envelope.OK {
		t.Fatalf("create change set %q: %+v", name, envelope)
	}
	return envelope
}

func requireAgentErrorCode(t *testing.T, response *httptest.ResponseRecorder, code string) {
	t.Helper()
	envelope := decodeAgentEnvelope(t, response)
	if envelope.OK || envelope.Error == nil || envelope.Error.Code != code {
		t.Fatalf("error=%+v want=%q", envelope, code)
	}
}

func stringPointer(value string) *string { return &value }

func TestAgentV2ReviewSubmissionParityUsesExactBoundState(t *testing.T) {
	_, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), t.TempDir())
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Agent reviews"}))
	change := createAgentChangeSet(t, handler, created.StoreID, created.Revision, "Review me")
	state := changeSetState(t, change)
	component := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/components/create", agentapi.ComponentCreateRequest{
		StatePreconditions: state, Title: "Gateway", Description: "Routes traffic.\n", DiagramID: &created.RootDiagramID,
	}))
	if !component.OK {
		t.Fatalf("component create: %+v", component)
	}
	state.Generation = uint64(resultMap(t, component)["generation"].(float64))
	componentID := resultMap(t, component)["component_id"].(string)
	unreviewed := agentapi.ReviewSubmissionSubmitRequest{
		StoreID: created.StoreID, ChangeSetID: state.ChangeSetID, ReviewedState: strings.Repeat("c", 40),
		BaseRevision: created.Revision, CandidateTree: strings.Repeat("d", 40), Generation: state.Generation,
		Verdict: "approve", Author: "Early reviewer",
	}
	requireAgentErrorCode(t, postAgent(t, handler, "/api/agent/v2/review-submissions/submit", unreviewed), "review_submission_not_allowed")
	reviewed := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: state}))
	if !reviewed.OK {
		t.Fatalf("prepare review: %+v", reviewed)
	}
	binding := resultMap(t, reviewed)
	input := agentapi.ReviewSubmissionSubmitRequest{
		StoreID: created.StoreID, ChangeSetID: state.ChangeSetID, ReviewedState: binding["reviewed_state"].(string),
		BaseRevision: binding["base_revision"].(string), CandidateTree: binding["candidate_tree"].(string), Generation: state.Generation,
		Verdict: "request_changes", Author: "Reviewer agent", Body: "Please clarify this proposal.\n",
		Comments: []agentapi.ReviewCommentInput{{Body: "Explain this line.", Anchor: agentapi.ReviewAnchor{Kind: "component_markdown", Side: "with_changes", ComponentID: componentID, StartLine: 2, EndLine: 2}}},
	}
	empty := input
	empty.Verdict, empty.Body, empty.Comments = "comment", "", nil
	requireAgentErrorCode(t, postAgent(t, handler, "/api/agent/v2/review-submissions/submit", empty), "invalid_request")
	invalidAnchor := input
	invalidAnchor.Comments = []agentapi.ReviewCommentInput{{Body: "Missing line.", Anchor: agentapi.ReviewAnchor{Kind: "component_markdown", Side: "with_changes", ComponentID: componentID, StartLine: 99, EndLine: 99}}}
	invalidResponse := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/review-submissions/submit", invalidAnchor))
	if invalidResponse.OK || invalidResponse.Error == nil || invalidResponse.Error.Code != "review_anchor_invalid" || invalidResponse.Error.Details["comment_index"] != float64(1) {
		t.Fatalf("invalid anchor response=%+v", invalidResponse)
	}

	submitted := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/review-submissions/submit", input))
	if !submitted.OK {
		t.Fatalf("submit: %+v", submitted)
	}
	submission := resultMap(t, submitted)
	reviewID, ok := submission["id"].(string)
	if !ok || submission["reviewed_state"] != input.ReviewedState || submission["verdict"] != "request_changes" || submission["current_generation"] != true {
		t.Fatalf("submission=%+v", submission)
	}
	listed := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/review-submissions/list", agentapi.ReviewSubmissionsListRequest{StoreID: created.StoreID, ChangeSetID: state.ChangeSetID}))
	if !listed.OK || len(resultMap(t, listed)["reviews"].([]any)) != 1 {
		t.Fatalf("list=%+v", listed)
	}
	inspected := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/review-submissions/inspect", agentapi.ReviewSubmissionInspectRequest{StoreID: created.StoreID, ChangeSetID: state.ChangeSetID, ReviewID: reviewID}))
	if !inspected.OK {
		t.Fatalf("inspect=%+v", inspected)
	}
	inspectResult := resultMap(t, inspected)
	comments := inspectResult["comments"].([]any)
	if inspectResult["body"] != input.Body || inspectResult["proposal_markdown"] != "" || len(comments) != 1 || comments[0].(map[string]any)["body"] != "Explain this line." {
		t.Fatalf("inspect result=%+v", inspectResult)
	}
}

func TestAgentV2ParallelChangeSetsPreserveInvalidRawStateAcrossRestart(t *testing.T) {
	dataDirectory := t.TempDir()
	uiDirectory := t.TempDir()
	_, handler := newHandler("http://127.0.0.1:8080", uiDirectory, dataDirectory)
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Parallel authority"}))

	changeA := createAgentChangeSet(t, handler, created.StoreID, created.Revision, "Gateway work")
	changeB := createAgentChangeSet(t, handler, created.StoreID, created.Revision, "Worker work")
	stateA, stateB := changeSetState(t, changeA), changeSetState(t, changeB)

	gateway := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/components/create", agentapi.ComponentCreateRequest{
		StatePreconditions: stateA, Title: "Gateway", DiagramID: &created.RootDiagramID,
	}))
	if !gateway.OK {
		t.Fatalf("create A Component: %+v", gateway)
	}
	stateA.Generation = uint64(resultMap(t, gateway)["generation"].(float64))
	gatewayID := resultMap(t, gateway)["component_id"].(string)

	worker := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/components/create", agentapi.ComponentCreateRequest{
		StatePreconditions: stateB, Title: "Worker", DiagramID: &created.RootDiagramID,
	}))
	if !worker.OK {
		t.Fatalf("create B Component: %+v", worker)
	}
	stateB.Generation = uint64(resultMap(t, worker)["generation"].(float64))
	workerID := resultMap(t, worker)["component_id"].(string)

	invalid := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/relationships/add", agentapi.RelationshipAddRequest{
		StatePreconditions: stateB, SourceID: workerID, TargetID: "not-a-component-id", Label: "   ",
	}))
	if !invalid.OK || resultMap(t, invalid)["candidate_valid"] != false {
		t.Fatalf("invalid raw row was not retained: %+v", invalid)
	}
	stateB.Generation = uint64(resultMap(t, invalid)["generation"].(float64))

	listed := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/list", agentapi.ChangeSetsListRequest{StoreID: created.StoreID}))
	if !listed.OK || len(resultMap(t, listed)["change_sets"].([]any)) != 2 {
		t.Fatalf("parallel list: %+v", listed)
	}

	_, restarted := newHandler("http://127.0.0.1:8080", uiDirectory, dataDirectory)
	opened := decodeAgentEnvelope(t, postAgent(t, restarted, "/api/agent/v2/projects/open", agentapi.ProjectOpenRequest{Slug: created.ProjectSlug}))
	if !opened.OK {
		t.Fatalf("restart open: %+v", opened)
	}
	inspectedB := decodeAgentEnvelope(t, postAgent(t, restarted, "/api/agent/v2/change-sets/inspect", agentapi.ChangeSetInspectRequest{StoreID: created.StoreID, ChangeSetID: stateB.ChangeSetID}))
	components := resultMap(t, inspectedB)["components"].([]any)
	rows := components[0].(map[string]any)["relationships"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["target_id"] != "not-a-component-id" || rows[0].(map[string]any)["label"] != "   " || resultMap(t, inspectedB)["candidate"] != nil {
		t.Fatalf("restart lost exact invalid state: %#v", resultMap(t, inspectedB))
	}
	inspectedA := decodeAgentEnvelope(t, postAgent(t, restarted, "/api/agent/v2/change-sets/inspect", agentapi.ChangeSetInspectRequest{StoreID: created.StoreID, ChangeSetID: stateA.ChangeSetID}))
	if !inspectedA.OK || uint64(resultMap(t, inspectedA)["generation"].(float64)) != stateA.Generation || resultMap(t, inspectedA)["candidate"] == nil {
		t.Fatalf("A changed while B was edited: %+v", inspectedA)
	}

	repaired := decodeAgentEnvelope(t, postAgent(t, restarted, "/api/agent/v2/relationships/edit", agentapi.RelationshipEditRequest{
		StatePreconditions: stateB, SourceID: workerID, OldTargetID: "not-a-component-id", OldLabel: "   ", Occurrence: 1, TargetID: workerID, Label: "retries",
	}))
	if !repaired.OK || resultMap(t, repaired)["candidate_valid"] != true {
		t.Fatalf("exact raw repair: %+v", repaired)
	}
	if gatewayID == workerID {
		t.Fatal("independent change sets reused a Component identity")
	}
}

func TestAgentV2AcceptanceRetainsAppliedReceiptAndOtherOutOfDateProposal(t *testing.T) {
	state, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), t.TempDir())
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Acceptance isolation"}))
	changeA := createAgentChangeSet(t, handler, created.StoreID, created.Revision, "Change A")
	changeB := createAgentChangeSet(t, handler, created.StoreID, created.Revision, "Change B")
	stateA, stateB := changeSetState(t, changeA), changeSetState(t, changeB)

	mutate := func(state *agentapi.StatePreconditions, title string) string {
		t.Helper()
		result := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/components/create", agentapi.ComponentCreateRequest{StatePreconditions: *state, Title: title, DiagramID: &created.RootDiagramID}))
		if !result.OK {
			t.Fatalf("mutate %s: %+v", title, result)
		}
		state.Generation = uint64(resultMap(t, result)["generation"].(float64))
		return resultMap(t, result)["component_id"].(string)
	}
	mutate(&stateA, "Gateway")
	componentB := mutate(&stateB, "Worker")
	reviewA := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: stateA}))
	reviewB := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: stateB}))
	if !reviewA.OK || !reviewB.OK {
		t.Fatalf("reviews A=%+v B=%+v", reviewA, reviewB)
	}
	bindingA := resultMap(t, reviewA)
	originalBindingB := resultMap(t, reviewB)
	state.stateMutex.Lock()
	bObjectBefore := state.changeSets[stateB.ChangeSetID].refObject
	state.stateMutex.Unlock()
	updatedA := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/architecture/update", agentapi.ArchitectureUpdateRequest{
		StoreID: created.StoreID, ChangeSetID: stateA.ChangeSetID, BaseRevision: bindingA["base_revision"].(string), CandidateTree: bindingA["candidate_tree"].(string), Generation: stateA.Generation,
	}))
	if !updatedA.OK || updatedA.Context.AcceptedRevision == nil || *updatedA.Context.AcceptedRevision == created.Revision {
		t.Fatalf("accept A: %+v", updatedA)
	}
	if resultMap(t, updatedA)["publication"] != "published" {
		t.Fatalf("first acceptance classification: %+v", updatedA)
	}

	appliedA := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/inspect", agentapi.ChangeSetInspectRequest{StoreID: created.StoreID, ChangeSetID: stateA.ChangeSetID}))
	activeB := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/inspect", agentapi.ChangeSetInspectRequest{StoreID: created.StoreID, ChangeSetID: stateB.ChangeSetID}))
	if resultMap(t, appliedA)["lifecycle"] != "applied" || resultMap(t, appliedA)["applied_revision"] != *updatedA.Context.AcceptedRevision {
		t.Fatalf("A receipt: %+v", appliedA)
	}
	if resultMap(t, appliedA)["out_of_date"] != false {
		t.Fatalf("applied receipt was mislabeled out of date: %+v", appliedA)
	}
	activeBResult := resultMap(t, activeB)
	activeBReview, _ := activeBResult["review"].(map[string]any)
	if activeBResult["lifecycle"] != "active" || activeBResult["out_of_date"] != true || uint64(activeBResult["generation"].(float64)) != stateB.Generation ||
		activeBResult["candidate_tree"] != originalBindingB["candidate_tree"] || activeBReview["base_revision"] != originalBindingB["base_revision"] ||
		activeBReview["candidate_tree"] != originalBindingB["candidate_tree"] || activeBReview["generation"] != originalBindingB["generation"] {
		t.Fatalf("B was not preserved: %+v", activeB)
	}
	state.stateMutex.Lock()
	bObjectAfter := state.changeSets[stateB.ChangeSetID].refObject
	state.stateMutex.Unlock()
	if bObjectAfter != bObjectBefore {
		t.Fatalf("accepting A rewrote B: before=%s after=%s", bObjectBefore, bObjectAfter)
	}

	editedB := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/components/edit", agentapi.ComponentEditRequest{
		StatePreconditions: stateB, ComponentID: componentB, Title: pointerTo("Worker v2"),
	}))
	if !editedB.OK {
		t.Fatalf("out-of-date B must remain editable: %+v", editedB)
	}
	stateB.Generation = uint64(resultMap(t, editedB)["generation"].(float64))
	reviewedB := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: stateB}))
	if !reviewedB.OK {
		t.Fatalf("out-of-date B must remain reviewable: %+v", reviewedB)
	}
	bindingB := resultMap(t, reviewedB)
	rejectedB := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/architecture/update", agentapi.ArchitectureUpdateRequest{
		StoreID: created.StoreID, ChangeSetID: stateB.ChangeSetID, BaseRevision: bindingB["base_revision"].(string), CandidateTree: bindingB["candidate_tree"].(string), Generation: stateB.Generation,
	}))
	if rejectedB.OK || rejectedB.Error == nil || rejectedB.Error.Code != "change_set_out_of_date" || rejectedB.Context.AcceptedRevision == nil || *rejectedB.Context.AcceptedRevision != *updatedA.Context.AcceptedRevision {
		t.Fatalf("out-of-date update classification: %+v", rejectedB)
	}

	receiptRetry := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/architecture/update", agentapi.ArchitectureUpdateRequest{
		StoreID: created.StoreID, ChangeSetID: stateA.ChangeSetID, BaseRevision: bindingA["base_revision"].(string), CandidateTree: bindingA["candidate_tree"].(string), Generation: stateA.Generation,
	}))
	if !receiptRetry.OK || receiptRetry.Context.AcceptedRevision == nil || *receiptRetry.Context.AcceptedRevision != *updatedA.Context.AcceptedRevision {
		t.Fatalf("durable receipt retry: %+v", receiptRetry)
	}
	if resultMap(t, receiptRetry)["publication"] != "already_applied" {
		t.Fatalf("receipt retry classification: %+v", receiptRetry)
	}

	changeC := createAgentChangeSet(t, handler, created.StoreID, *updatedA.Context.AcceptedRevision, "Change C")
	stateC := changeSetState(t, changeC)
	mutate(&stateC, "Scheduler")
	reviewC := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: stateC}))
	bindingC := resultMap(t, reviewC)
	updatedC := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/architecture/update", agentapi.ArchitectureUpdateRequest{
		StoreID: created.StoreID, ChangeSetID: stateC.ChangeSetID, BaseRevision: bindingC["base_revision"].(string), CandidateTree: bindingC["candidate_tree"].(string), Generation: stateC.Generation,
	}))
	if !updatedC.OK || updatedC.Context.AcceptedRevision == nil || *updatedC.Context.AcceptedRevision == *updatedA.Context.AcceptedRevision {
		t.Fatalf("advance Accepted after A receipt: %+v", updatedC)
	}
	receiptAfterAdvance := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/architecture/update", agentapi.ArchitectureUpdateRequest{
		StoreID: created.StoreID, ChangeSetID: stateA.ChangeSetID, BaseRevision: bindingA["base_revision"].(string), CandidateTree: bindingA["candidate_tree"].(string), Generation: stateA.Generation,
	}))
	if !receiptAfterAdvance.OK || receiptAfterAdvance.Context.AcceptedRevision == nil || *receiptAfterAdvance.Context.AcceptedRevision != *updatedC.Context.AcceptedRevision || resultMap(t, receiptAfterAdvance)["publication"] != "already_applied" {
		t.Fatalf("receipt after later Accepted advancement: %+v", receiptAfterAdvance)
	}
}

func TestAgentV2LifecycleErrorsAreTruthfulBeforeGenerationChecks(t *testing.T) {
	state, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), t.TempDir())
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Lifecycle errors"}))
	change := createAgentChangeSet(t, handler, created.StoreID, created.Revision, "Applied record")
	exact := changeSetState(t, change)
	component := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/components/create", agentapi.ComponentCreateRequest{
		StatePreconditions: exact, Title: "Worker", DiagramID: &created.RootDiagramID,
	}))
	componentID := resultMap(t, component)["component_id"].(string)
	exact.Generation = uint64(resultMap(t, component)["generation"].(float64))
	reviewed := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: exact}))
	binding := resultMap(t, reviewed)
	accepted := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/architecture/update", agentapi.ArchitectureUpdateRequest{
		StoreID: created.StoreID, ChangeSetID: exact.ChangeSetID, BaseRevision: binding["base_revision"].(string), CandidateTree: binding["candidate_tree"].(string), Generation: exact.Generation,
	}))
	if !accepted.OK || accepted.Context.AcceptedRevision == nil {
		t.Fatalf("accept: %+v", accepted)
	}
	wrongApplied := exact
	wrongApplied.Generation += 100
	requireAgentErrorCode(t, postAgent(t, handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: wrongApplied}), "change_set_not_editable")
	requireAgentErrorCode(t, postAgent(t, handler, "/api/agent/v2/change-sets/discard", agentapi.ChangeSetDiscardRequest{StatePreconditions: wrongApplied}), "change_set_not_editable")
	requireAgentErrorCode(t, postAgent(t, handler, "/api/agent/v2/components/edit", agentapi.ComponentEditRequest{StatePreconditions: wrongApplied, ComponentID: componentID, Title: stringPointer("No")}), "change_set_not_editable")

	unavailable := createAgentChangeSet(t, handler, created.StoreID, *accepted.Context.AcceptedRevision, "Unavailable record")
	unavailableState := changeSetState(t, unavailable)
	storePath, err := state.architecture.StorePath(created.StoreID)
	if err != nil {
		t.Fatal(err)
	}
	activeRef := "refs/workbraid/change-sets/active/" + unavailableState.ChangeSetID
	activeObject := git(t, "--git-dir", storePath, "rev-parse", activeRef)
	git(t, "--git-dir", storePath, "update-ref", activeRef, *accepted.Context.AcceptedRevision, activeObject)
	opened := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/projects/open", agentapi.ProjectOpenRequest{Slug: created.ProjectSlug}))
	if !opened.OK {
		t.Fatalf("reload malformed record: %+v", opened)
	}
	listed := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/list", agentapi.ChangeSetsListRequest{StoreID: created.StoreID}))
	unavailableRows := resultMap(t, listed)["unavailable"].([]any)
	if len(unavailableRows) != 1 || unavailableRows[0].(map[string]any)["id"] != unavailableState.ChangeSetID {
		t.Fatalf("listed unavailable=%+v", unavailableRows)
	}
	wrongUnavailable := unavailableState
	wrongUnavailable.Generation += 100
	requireAgentErrorCode(t, postAgent(t, handler, "/api/agent/v2/change-sets/rename", agentapi.ChangeSetRenameRequest{StatePreconditions: wrongUnavailable, Name: "No"}), "change_set_unavailable")
	requireAgentErrorCode(t, postAgent(t, handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: wrongUnavailable}), "change_set_unavailable")
	requireAgentErrorCode(t, postAgent(t, handler, "/api/agent/v2/change-sets/discard", agentapi.ChangeSetDiscardRequest{StatePreconditions: wrongUnavailable}), "change_set_unavailable")
	requireAgentErrorCode(t, postAgent(t, handler, "/api/agent/v2/components/create", agentapi.ComponentCreateRequest{StatePreconditions: wrongUnavailable, Title: "No", DiagramID: &created.RootDiagramID}), "change_set_unavailable")
	requireAgentErrorCode(t, postAgent(t, handler, "/api/agent/v2/architecture/update", agentapi.ArchitectureUpdateRequest{
		StoreID: created.StoreID, ChangeSetID: wrongUnavailable.ChangeSetID, BaseRevision: *accepted.Context.AcceptedRevision, CandidateTree: binding["candidate_tree"].(string), Generation: wrongUnavailable.Generation,
	}), "change_set_unavailable")
}

func TestAgentV2ConcurrentRecordsAndProjectBoundCreationRemainIndependent(t *testing.T) {
	dataDirectory := t.TempDir()
	_, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), dataDirectory)
	projectA := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Concurrent A"}))
	changeA := createAgentChangeSet(t, handler, projectA.StoreID, projectA.Revision, "Parallel A")
	changeB := createAgentChangeSet(t, handler, projectA.StoreID, projectA.Revision, "Parallel B")
	stateA, stateB := changeSetState(t, changeA), changeSetState(t, changeB)

	type callResult struct {
		name     string
		response *httptest.ResponseRecorder
	}
	start := make(chan struct{})
	results := make(chan callResult, 2)
	var wait sync.WaitGroup
	for _, operation := range []struct {
		name  string
		state agentapi.StatePreconditions
	}{
		{name: "A", state: stateA},
		{name: "B", state: stateB},
	} {
		body, err := json.Marshal(agentapi.ComponentCreateRequest{StatePreconditions: operation.state, Title: "Component " + operation.name, DiagramID: &projectA.RootDiagramID})
		if err != nil {
			t.Fatal(err)
		}
		wait.Add(1)
		go func(name string, body []byte) {
			defer wait.Done()
			<-start
			request := httptest.NewRequest(http.MethodPost, "/api/agent/v2/components/create", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			results <- callResult{name: name, response: response}
		}(operation.name, body)
	}
	close(start)
	wait.Wait()
	close(results)
	for result := range results {
		envelope := decodeAgentEnvelope(t, result.response)
		if !envelope.OK || resultMap(t, envelope)["generation"] != float64(1) {
			t.Fatalf("concurrent %s mutation: %+v", result.name, envelope)
		}
	}
	stateA.Generation, stateB.Generation = 1, 1
	reviewA := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: stateA}))
	reviewB := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: stateB}))
	if !reviewA.OK || !reviewB.OK {
		t.Fatalf("independent reviews A=%+v B=%+v", reviewA, reviewB)
	}
	bindingB := resultMap(t, reviewB)
	proposal := "# Exact proposal\n\nOnly A changes.\n"
	editedA := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/edit-proposal", agentapi.ChangeSetEditProposalRequest{StatePreconditions: stateA, ProposalMarkdown: proposal}))
	if !editedA.OK || resultMap(t, editedA)["proposal_markdown"] != proposal || resultMap(t, editedA)["review"] != nil {
		t.Fatalf("A proposal edit: %+v", editedA)
	}
	stateA.Generation = 2
	unchangedB := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/inspect", agentapi.ChangeSetInspectRequest{StoreID: projectA.StoreID, ChangeSetID: stateB.ChangeSetID}))
	unchangedBReview, _ := resultMap(t, unchangedB)["review"].(map[string]any)
	if !unchangedB.OK || resultMap(t, unchangedB)["generation"] != float64(1) || unchangedBReview["base_revision"] != bindingB["base_revision"] ||
		unchangedBReview["candidate_tree"] != bindingB["candidate_tree"] || unchangedBReview["generation"] != bindingB["generation"] {
		t.Fatalf("A invalidated B: %+v", unchangedB)
	}

	conflict := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/rename", agentapi.ChangeSetRenameRequest{StatePreconditions: stateA, Name: "parallel b"}))
	if conflict.OK || conflict.Error == nil || conflict.Error.Code != "change_set_name_conflict" {
		t.Fatalf("active-name conflict: %+v", conflict)
	}
	renamedA := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/rename", agentapi.ChangeSetRenameRequest{StatePreconditions: stateA, Name: "Renamed A"}))
	if !renamedA.OK || resultMap(t, renamedA)["generation"] != float64(3) {
		t.Fatalf("rename A: %+v", renamedA)
	}
	staleDiscard := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/discard", agentapi.ChangeSetDiscardRequest{StatePreconditions: stateA}))
	if staleDiscard.OK || staleDiscard.Error == nil || staleDiscard.Error.Code != "change_set_generation_mismatch" {
		t.Fatalf("stale discard: %+v", staleDiscard)
	}
	stateA.Generation = 3
	discardedA := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/discard", agentapi.ChangeSetDiscardRequest{StatePreconditions: stateA}))
	if !discardedA.OK {
		t.Fatalf("discard A: %+v", discardedA)
	}
	if keptB := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/inspect", agentapi.ChangeSetInspectRequest{StoreID: projectA.StoreID, ChangeSetID: stateB.ChangeSetID})); !keptB.OK {
		t.Fatalf("discard A removed B: %+v", keptB)
	}

	projectB := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Concurrent B"}))
	late := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/create", agentapi.ChangeSetCreateRequest{StoreID: projectA.StoreID, AcceptedRevision: projectA.Revision}))
	if late.OK || late.Error == nil || late.Error.Code != "project_mismatch" {
		t.Fatalf("late Project A create while B current: %+v", late)
	}
	if projectB.StoreID == projectA.StoreID {
		t.Fatal("projects reused a store identity")
	}
	openedA := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/projects/open", agentapi.ProjectOpenRequest{Slug: projectA.ProjectSlug}))
	if !openedA.OK {
		t.Fatalf("reopen A: %+v", openedA)
	}
	listedA := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/list", agentapi.ChangeSetsListRequest{StoreID: projectA.StoreID}))
	if !listedA.OK || len(resultMap(t, listedA)["change_sets"].([]any)) != 1 {
		t.Fatalf("late create changed A: %+v", listedA)
	}
}

func TestAgentV2NonCurrentAuthorityCannotMutateProposalRecords(t *testing.T) {
	dataDirectory := t.TempDir()
	state, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), dataDirectory)
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Non-current"}))
	change := createAgentChangeSet(t, handler, created.StoreID, created.Revision, "Retained")
	exact := changeSetState(t, change)
	storePath, err := state.architecture.StorePath(created.StoreID)
	if err != nil {
		t.Fatal(err)
	}
	git(t, "--git-dir", storePath, "update-ref", "-d", "refs/heads/accepted", created.Revision)
	refresh := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/architecture/refresh", agentapi.ArchitectureRefreshRequest{StoreID: created.StoreID, AcceptedRevision: created.Revision}))
	if refresh.OK || refresh.Error == nil || refresh.Error.Code != "architecture_non_current" {
		t.Fatalf("missing Accepted refresh: %+v", refresh)
	}
	state.stateMutex.Lock()
	beforeObject := state.changeSets[exact.ChangeSetID].refObject
	state.stateMutex.Unlock()
	requests := []*httptest.ResponseRecorder{
		postAgent(t, handler, "/api/agent/v2/change-sets/edit-proposal", agentapi.ChangeSetEditProposalRequest{StatePreconditions: exact, ProposalMarkdown: "must not persist"}),
		postAgent(t, handler, "/api/agent/v2/change-sets/rename", agentapi.ChangeSetRenameRequest{StatePreconditions: exact, Name: "Must not persist"}),
		postAgent(t, handler, "/api/agent/v2/change-sets/discard", agentapi.ChangeSetDiscardRequest{StatePreconditions: exact}),
	}
	for _, response := range requests {
		envelope := decodeAgentEnvelope(t, response)
		if envelope.OK || envelope.Error == nil || envelope.Error.Code != "architecture_non_current" {
			t.Fatalf("non-current mutation: %+v", envelope)
		}
	}
	state.stateMutex.Lock()
	after := state.changeSets[exact.ChangeSetID]
	state.stateMutex.Unlock()
	if after == nil || after.refObject != beforeObject || after.name != "Retained" || after.proposal != "" || after.generation != 0 {
		t.Fatalf("non-current authority mutated record: %+v", after)
	}
	if object := git(t, "--git-dir", filepath.Clean(storePath), "rev-parse", "refs/workbraid/change-sets/active/"+exact.ChangeSetID); object != beforeObject {
		t.Fatalf("non-current authority changed ref: %s", object)
	}
	if inspected := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/inspect", agentapi.ChangeSetInspectRequest{StoreID: created.StoreID, ChangeSetID: exact.ChangeSetID})); !inspected.OK || resultMap(t, inspected)["out_of_date"] != false {
		t.Fatalf("read-only inspection mislabeled unknown authority: %+v", inspected)
	}
}

func TestAgentV2AcceptanceCASRaceIsAnAcceptedConflict(t *testing.T) {
	state, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), t.TempDir())
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Accepted race"}))
	change := createAgentChangeSet(t, handler, created.StoreID, created.Revision, "Racing proposal")
	exact := changeSetState(t, change)
	mutated := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/components/create", agentapi.ComponentCreateRequest{
		StatePreconditions: exact, Title: "Proposed", DiagramID: &created.RootDiagramID,
	}))
	exact.Generation = uint64(resultMap(t, mutated)["generation"].(float64))
	reviewed := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: exact}))
	binding := resultMap(t, reviewed)

	state.stateMutex.Lock()
	base := *state.loadedSnapshot
	state.stateMutex.Unlock()
	externalChange := state.architecture.NewComponentChange(base, nil, "External", "")
	externalCandidate, err := state.architecture.ConstructCandidate(context.Background(), base, []architecture.ComponentChange{externalChange}, architecture.CandidateComposition{
		NewComponentHomes: []architecture.NewComponentHome{{ComponentID: externalChange.ID, DiagramID: base.RootDiagramID()}},
	})
	if err != nil {
		t.Fatal(err)
	}
	externalRevision, err := state.architecture.CreateSuccessor(context.Background(), base, externalCandidate)
	if err != nil {
		t.Fatal(err)
	}
	state.beforeAcceptedCAS = func(string) {
		if advanceErr := state.architecture.AdvanceAccepted(context.Background(), base, externalRevision); advanceErr != nil {
			t.Error(advanceErr)
		}
	}
	result := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/architecture/update", agentapi.ArchitectureUpdateRequest{
		StoreID: created.StoreID, ChangeSetID: exact.ChangeSetID, BaseRevision: binding["base_revision"].(string), CandidateTree: binding["candidate_tree"].(string), Generation: exact.Generation,
	}))
	if result.OK || result.Error == nil || result.Error.Code != "accepted_conflict" {
		t.Fatalf("Accepted CAS race: %+v", result)
	}
	state.stateMutex.Lock()
	retained := state.changeSets[exact.ChangeSetID]
	state.stateMutex.Unlock()
	if retained == nil || retained.lifecycle != "active" || retained.generation != exact.Generation || retained.review == nil {
		t.Fatalf("CAS race did not retain exact proposal: %+v", retained)
	}
}

func TestAgentV2RejectsLegacyPathAndUsesExplicitGeneration(t *testing.T) {
	state, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), t.TempDir())
	legacy := getAgent(t, handler, "/api/agent/v1/status")
	if legacy.OK || legacy.Error == nil || legacy.Error.Code != "incompatible_server" {
		t.Fatalf("legacy protocol response: %+v", legacy)
	}
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Exact preconditions"}))
	change := createAgentChangeSet(t, handler, created.StoreID, created.Revision, "Exact")
	exact := changeSetState(t, change)
	wrong := exact
	wrong.Generation++
	result := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/components/create", agentapi.ComponentCreateRequest{StatePreconditions: wrong, Title: "No write", DiagramID: &created.RootDiagramID}))
	if result.OK || result.Error == nil || result.Error.Code != "change_set_generation_mismatch" {
		t.Fatalf("wrong generation: %+v", result)
	}
	state.stateMutex.Lock()
	record := state.changeSets[exact.ChangeSetID]
	state.stateMutex.Unlock()
	if record == nil || record.generation != 0 || len(record.changes) != 0 {
		t.Fatalf("wrong generation mutated record: %+v", record)
	}

	missing := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/components/create", map[string]any{
		"store_id": exact.StoreID, "change_set_id": exact.ChangeSetID, "title": "No implicit zero", "diagram_id": created.RootDiagramID,
	}))
	if missing.OK || missing.Error == nil || missing.Error.Code != "invalid_request" {
		t.Fatalf("missing generation: %+v", missing)
	}

	createdComponent := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/components/create", agentapi.ComponentCreateRequest{
		StatePreconditions: exact, Title: "Stable", DiagramID: &created.RootDiagramID,
	}))
	componentID := resultMap(t, createdComponent)["component_id"].(string)
	exact.Generation = uint64(resultMap(t, createdComponent)["generation"].(float64))
	state.stateMutex.Lock()
	beforeObject := state.changeSets[exact.ChangeSetID].refObject
	state.stateMutex.Unlock()
	noOp := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/components/edit", agentapi.ComponentEditRequest{
		StatePreconditions: exact, ComponentID: componentID, Title: pointerTo("Stable"),
	}))
	if !noOp.OK || resultMap(t, noOp)["unchanged"] != true || uint64(resultMap(t, noOp)["generation"].(float64)) != exact.Generation {
		t.Fatalf("component no-op: %+v", noOp)
	}
	noOpDiagram := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/diagrams/edit-title", agentapi.DiagramEditTitleRequest{
		StatePreconditions: exact, DiagramID: created.RootDiagramID, Title: "Exact preconditions",
	}))
	if !noOpDiagram.OK || resultMap(t, noOpDiagram)["unchanged"] != true || uint64(resultMap(t, noOpDiagram)["generation"].(float64)) != exact.Generation {
		t.Fatalf("Diagram no-op: %+v", noOpDiagram)
	}
	state.stateMutex.Lock()
	afterObject := state.changeSets[exact.ChangeSetID].refObject
	state.stateMutex.Unlock()
	if afterObject != beforeObject {
		t.Fatalf("no-op rewrote durable state: before=%s after=%s", beforeObject, afterObject)
	}
	changedDiagram := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/diagrams/edit-title", agentapi.DiagramEditTitleRequest{
		StatePreconditions: exact, DiagramID: created.RootDiagramID, Title: "Changed title",
	}))
	if !changedDiagram.OK || resultMap(t, changedDiagram)["generation"] != float64(exact.Generation+1) {
		t.Fatalf("Diagram title change: %+v", changedDiagram)
	}
	exact.Generation++
	revertedDiagram := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/diagrams/edit-title", agentapi.DiagramEditTitleRequest{
		StatePreconditions: exact, DiagramID: created.RootDiagramID, Title: "Exact preconditions",
	}))
	if !revertedDiagram.OK || resultMap(t, revertedDiagram)["generation"] != float64(exact.Generation+1) || resultMap(t, revertedDiagram)["unchanged"] != nil {
		t.Fatalf("Diagram title revert: %+v", revertedDiagram)
	}
	exact.Generation++
	inspected := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/inspect", agentapi.ChangeSetInspectRequest{StoreID: exact.StoreID, ChangeSetID: exact.ChangeSetID}))
	if titles := resultMap(t, inspected)["diagram_titles"].([]any); len(titles) != 0 {
		t.Fatalf("reverted Diagram title left redundant facts: %+v", titles)
	}
	state.stateMutex.Lock()
	revertedObject := state.changeSets[exact.ChangeSetID].refObject
	state.stateMutex.Unlock()
	repeatedRevert := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/diagrams/edit-title", agentapi.DiagramEditTitleRequest{
		StatePreconditions: exact, DiagramID: created.RootDiagramID, Title: "Exact preconditions",
	}))
	if !repeatedRevert.OK || resultMap(t, repeatedRevert)["unchanged"] != true || resultMap(t, repeatedRevert)["generation"] != float64(exact.Generation) {
		t.Fatalf("repeated Diagram revert: %+v", repeatedRevert)
	}
	state.stateMutex.Lock()
	finalObject := state.changeSets[exact.ChangeSetID].refObject
	state.stateMutex.Unlock()
	if finalObject != revertedObject {
		t.Fatalf("repeated Diagram revert rewrote state: before=%s after=%s", revertedObject, finalObject)
	}
}

func pointerTo(value string) *string { return &value }
