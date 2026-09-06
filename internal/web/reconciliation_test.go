package web

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"workbraid/internal/agentapi"
	"workbraid/internal/architecture"
)

func prepareReconciliationFixture(t *testing.T) (nativeRefreshFixture, agentapi.ReconciliationInputs) {
	t.Helper()
	f := newNativeRefreshFixture(t, false)
	p := changeSetState(t, createAgentChangeSet(t, f.handler, f.base.StoreID, f.base.Revision, "Document Gateway"))
	edit := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/components/edit", agentapi.ComponentEditRequest{StatePreconditions: p, ComponentID: f.component, Description: stringPointer("\nProposed exact body.\r\n")}))
	if !edit.OK {
		t.Fatal(edit.Error)
	}
	p = changeSetState(t, edit)
	proposal := "# Exact proposal\r\n\nKeep this feedback historical.\n"
	edited := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/change-sets/edit-proposal", agentapi.ChangeSetEditProposalRequest{StatePreconditions: p, ProposalMarkdown: proposal}))
	if !edited.OK {
		t.Fatal(edited.Error)
	}
	p = changeSetState(t, edited)
	review := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: p}))
	if !review.OK {
		t.Fatal(review.Error)
	}
	// Preserve an actual submitted review parent across reconciliation/restart.
	pending := f.state.changeSets[p.ChangeSetID]
	_, err := f.state.architecture.SubmitReview(t.Context(), p.StoreID, architecture.ReviewSubmissionInput{ChangeSetID: p.ChangeSetID, ReviewedState: pending.refObject, Binding: architecture.ReviewBinding{BaseRevision: pending.baseRevision, CandidateTree: pending.candidate.Tree(), Generation: pending.generation}, Verdict: "comment", Author: "Reviewer", Body: "Historical exact body.\r\n", SubmittedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	a := changeSetState(t, createAgentChangeSet(t, f.handler, f.base.StoreID, f.base.Revision, "Rename Gateway"))
	editA := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/components/edit", agentapi.ComponentEditRequest{StatePreconditions: a, ComponentID: f.component, Title: stringPointer("Accepted Gateway")}))
	if !editA.OK {
		t.Fatal(editA.Error)
	}
	a = changeSetState(t, editA)
	binding := resultMap(t, decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: a})))
	updated := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/architecture/update", agentapi.ArchitectureUpdateRequest{StoreID: a.StoreID, ChangeSetID: a.ChangeSetID, Generation: a.Generation, BaseRevision: binding["base_revision"].(string), CandidateTree: binding["candidate_tree"].(string)}))
	if !updated.OK {
		t.Fatal(updated.Error)
	}
	pending = f.state.changeSets[p.ChangeSetID]
	return f, agentapi.ReconciliationInputs{StatePreconditions: p, ChangeSetState: pending.refObject, BaseRevision: pending.baseRevision, CandidateTree: pending.candidate.Tree(), AcceptedRevision: f.state.loadedSnapshot.Revision()}
}

func TestReconciliationPreviewApplyHistoricalReviewAndRestart(t *testing.T) {
	f, inputs := prepareReconciliationFixture(t)
	refs := git(t, "--git-dir", f.storePath, "show-ref")
	stateBefore := git(t, "--git-dir", f.storePath, "show", inputs.ChangeSetState+":changes.yaml")
	reviewsBefore := git(t, "--git-dir", f.storePath, "for-each-ref", "--format=%(refname) %(objectname)", "refs/workbraid/reviews/")
	preview := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/change-sets/reconcile-preview", agentapi.ReconciliationPreviewRequest{ReconciliationInputs: inputs}))
	if !preview.OK || resultMap(t, preview)["status"] != "ready" {
		t.Fatalf("preview: %+v", preview)
	}
	if git(t, "--git-dir", f.storePath, "show-ref") != refs {
		t.Fatal("preview changed refs")
	}
	result := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/change-sets/reconcile-apply", agentapi.ReconciliationApplyRequest{ReconciliationInputs: inputs, Resolutions: []architecture.ReconciliationResolution{}}))
	if !result.OK {
		t.Fatalf("apply: %+v", result.Error)
	}
	record := f.state.changeSets[inputs.ChangeSetID]
	if record.generation != inputs.Generation+1 || record.baseRevision != inputs.AcceptedRevision || record.refObject == inputs.ChangeSetState || record.review != nil {
		t.Fatalf("bad rebased state: %+v", record)
	}
	if git(t, "--git-dir", f.storePath, "show", "-s", "--format=%P", record.refObject) != inputs.AcceptedRevision {
		t.Fatal("state parent is not exactly A")
	}
	if git(t, "--git-dir", f.storePath, "rev-parse", "refs/heads/accepted") != inputs.AcceptedRevision {
		t.Fatal("reconciliation changed Accepted")
	}
	if git(t, "--git-dir", f.storePath, "show", inputs.ChangeSetState+":changes.yaml") != stateBefore || git(t, "--git-dir", f.storePath, "for-each-ref", "--format=%(refname) %(objectname)", "refs/workbraid/reviews/") != reviewsBefore {
		t.Fatal("historical state/review changed")
	}
	if got := record.candidate.Snapshot().AuthoringComponents()[0]; got.Title != "Accepted Gateway" || got.Description != "\nProposed exact body.\r\n" {
		t.Fatalf("wrong semantics: %+v", got)
	}
	object, tree, proposal := record.refObject, record.candidate.Tree(), record.proposal
	data := filepath.Dir(filepath.Dir(f.storePath))
	restarted, handler := newHandler(testOrigin, t.TempDir(), data)
	postJSONRequest(t, handler, "/api/projects/open", map[string]any{"project_slug": f.base.ProjectSlug})
	loaded := restarted.changeSets[inputs.ChangeSetID]
	if loaded == nil || loaded.refObject != object || loaded.candidate.Tree() != tree || loaded.proposal != proposal || len(restarted.reviews) != 1 {
		t.Fatal("restart lost exact proposal/review")
	}
	for _, review := range restarted.reviews {
		if review.ReviewedState != inputs.ChangeSetState {
			t.Fatal("historical review retargeted")
		}
	}
	requireAgentErrorCode(t, postAgent(t, handler, "/api/agent/v2/change-sets/reconcile-apply", agentapi.ReconciliationApplyRequest{ReconciliationInputs: inputs, Resolutions: []architecture.ReconciliationResolution{}}), "change_set_state_mismatch")
	// The reconciled result goes through ordinary review/update unchanged.
	state := agentapi.StatePreconditions{StoreID: inputs.StoreID, ChangeSetID: inputs.ChangeSetID, Generation: loaded.generation}
	review := resultMap(t, decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: state})))
	accepted := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v2/architecture/update", agentapi.ArchitectureUpdateRequest{StoreID: state.StoreID, ChangeSetID: state.ChangeSetID, Generation: state.Generation, BaseRevision: review["base_revision"].(string), CandidateTree: review["candidate_tree"].(string)}))
	if !accepted.OK {
		t.Fatalf("ordinary acceptance after reconciliation: %+v", accepted.Error)
	}
}

func TestReconciliationActualResponseLossAndOldStateRetry(t *testing.T) {
	f, inputs := prepareReconciliationFixture(t)
	published, resume, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	f.state.afterReconciliationTransaction = func() {
		close(published)
		select {
		case <-resume:
		case <-time.After(3 * time.Second):
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.handler.ServeHTTP(w, r)
		if strings.HasSuffix(r.URL.Path, "reconcile-apply") {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	}))
	defer server.Close()
	payload := agentapi.ReconciliationApplyRequest{ReconciliationInputs: inputs, Resolutions: []architecture.ReconciliationResolution{}}
	data, _ := json.Marshal(payload)
	conn, err := net.DialTimeout("tcp", strings.TrimPrefix(server.URL, "http://"), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	request, _ := http.NewRequest(http.MethodPost, server.URL+"/api/agent/v2/change-sets/reconcile-apply", bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	if err := request.Write(conn); err != nil {
		t.Fatal(err)
	}
	select {
	case <-published:
	case <-time.After(3 * time.Second):
		t.Fatal("real transaction did not publish")
	}
	// Close the actual TCP client without reading any response, after Git has
	// committed and before handler publication/serialization resumes.
	_ = conn.Close()
	close(resume)
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("response-loss request did not finish")
	}
	f.state.afterReconciliationTransaction = nil
	actual := git(t, "--git-dir", f.storePath, "rev-parse", "refs/workbraid/change-sets/active/"+inputs.ChangeSetID)
	response, err := server.Client().Post(server.URL+"/api/agent/v2/change-sets/reconcile-apply", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	var envelope agentapi.Envelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.OK || envelope.Error == nil || envelope.Error.Code != "change_set_state_mismatch" {
		t.Fatalf("old S retry: %s", body)
	}
	current := envelope.Error.Details["change_set"].(map[string]any)
	if current["change_set_state"] != actual || uint64(current["generation"].(float64)) != inputs.Generation+1 {
		t.Fatalf("untruthful recovery: %s", body)
	}
	if git(t, "--git-dir", f.storePath, "rev-parse", "refs/workbraid/change-sets/active/"+inputs.ChangeSetID) != actual {
		t.Fatal("retry applied a second mutation")
	}
}

func TestReconciliationExactInputsAndReviewNoOp(t *testing.T) {
	f, inputs := prepareReconciliationFixture(t)
	before := git(t, "--git-dir", f.storePath, "show-ref")
	// Repeating an identical prepared Review must retain S, even though a first
	// preparation would have changed it without changing generation.
	review := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: inputs.StatePreconditions}))
	if !review.OK || git(t, "--git-dir", f.storePath, "show-ref") != before {
		t.Fatal("unchanged review changed authority")
	}
	for _, field := range []string{"base", "tree", "generation", "accepted"} {
		t.Run(field, func(t *testing.T) {
			wrong := inputs
			code := "invalid_request"
			switch field {
			case "base":
				wrong.BaseRevision = inputs.AcceptedRevision
			case "tree":
				wrong.CandidateTree = inputs.AcceptedRevision
			case "generation":
				wrong.Generation++
				code = "change_set_generation_mismatch"
			case "accepted":
				wrong.AcceptedRevision = inputs.BaseRevision
				code = "accepted_conflict"
			}
			requireAgentErrorCode(t, postAgent(t, f.handler, "/api/agent/v2/change-sets/reconcile-apply", agentapi.ReconciliationApplyRequest{ReconciliationInputs: wrong, Resolutions: []architecture.ReconciliationResolution{}}), code)
			if git(t, "--git-dir", f.storePath, "show-ref") != before {
				t.Fatal("invalid exact inputs wrote a ref")
			}
		})
	}
}

func TestReconciliationRealAuthorityRaces(t *testing.T) {
	for _, scenario := range []string{"preview_accepted", "transaction_accepted", "transaction_state", "successful_publication_loss_and_later_advancement"} {
		t.Run(scenario, func(t *testing.T) {
			f, inputs := prepareReconciliationFixture(t)
			active := "refs/workbraid/change-sets/active/" + inputs.ChangeSetID
			expectedState, expectedAccepted := inputs.ChangeSetState, inputs.AcceptedRevision
			advanceAccepted := func() {
				tree := git(t, "--git-dir", f.storePath, "rev-parse", inputs.AcceptedRevision+"^{tree}")
				expectedAccepted = git(t, "--git-dir", f.storePath, "commit-tree", tree, "-p", inputs.AcceptedRevision, "-m", "External authority advance")
				git(t, "--git-dir", f.storePath, "update-ref", "refs/heads/accepted", expectedAccepted, inputs.AcceptedRevision)
			}
			renameActual := func() {
				records, unavailable, err := f.state.architecture.LoadChangeSets(t.Context(), inputs.StoreID)
				if err != nil || len(unavailable) > 0 {
					t.Fatalf("load actual: %v %+v", err, unavailable)
				}
				for _, record := range records {
					if record.ID == inputs.ChangeSetID {
						record.Name = "Renamed after capture"
						record.Generation++
						record.Review = nil
						expectedState, err = f.state.architecture.WriteActiveChangeSet(t.Context(), inputs.StoreID, record, record.RefObject)
						if err != nil {
							t.Fatal(err)
						}
						return
					}
				}
				t.Fatal("actual record disappeared")
			}
			code := "accepted_conflict"
			switch scenario {
			case "preview_accepted":
				f.state.beforeReconciliationReobserve = advanceAccepted
				code = "architecture_non_current"
			case "transaction_accepted":
				f.state.beforeReconciliationTransaction = advanceAccepted
			case "transaction_state":
				f.state.beforeReconciliationTransaction = renameActual
				code = "change_set_state_mismatch"
			case "successful_publication_loss_and_later_advancement":
				f.state.afterReconciliationTransaction = func() {
					renameActual()
					advanceAccepted()
					// Lose the cached publication entirely after the known successful
					// Git transaction. Only the actual ordinary record can recover it.
					delete(f.state.changeSets, inputs.ChangeSetID)
				}
			}
			response := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/change-sets/reconcile-apply", agentapi.ReconciliationApplyRequest{ReconciliationInputs: inputs, Resolutions: []architecture.ReconciliationResolution{}}))
			if scenario == "successful_publication_loss_and_later_advancement" {
				if !response.OK {
					t.Fatal(response.Error)
				}
				result := resultMap(t, response)
				if result["publication"] != "changed_after_reconciliation" || result["change_set_state"] != expectedState || result["out_of_date"] != true || result["name"] != "Renamed after capture" {
					t.Fatalf("guessed or stale publication: %+v", result)
				}
				f.state.afterReconciliationTransaction = nil
				retry := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/change-sets/reconcile-apply", agentapi.ReconciliationApplyRequest{ReconciliationInputs: inputs, Resolutions: []architecture.ReconciliationResolution{}}))
				if retry.OK || retry.Error.Code != "change_set_state_mismatch" {
					t.Fatalf("retry: %+v", retry)
				}
				current := retry.Error.Details["change_set"].(map[string]any)
				if current["change_set_state"] != expectedState || current["out_of_date"] != true || current["name"] != "Renamed after capture" {
					t.Fatalf("retry guessed a receipt: %+v", current)
				}
			} else if response.OK || response.Error.Code != code {
				t.Fatalf("race: %+v", response)
			}
			if git(t, "--git-dir", f.storePath, "rev-parse", active) != expectedState || git(t, "--git-dir", f.storePath, "rev-parse", "refs/heads/accepted") != expectedAccepted {
				t.Fatal("race performed an extra mutation")
			}
		})
	}
}

func TestReconciliationOrdinaryChangesInvalidateExactState(t *testing.T) {
	for _, scenario := range []string{"component_edit", "rename", "proposal", "first_review", "discard", "project_switch", "applied"} {
		t.Run(scenario, func(t *testing.T) {
			f, inputs := prepareReconciliationFixture(t)
			if scenario == "first_review" {
				edited := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/change-sets/edit-proposal", agentapi.ChangeSetEditProposalRequest{StatePreconditions: inputs.StatePreconditions, ProposalMarkdown: "Before preparing the first review again\n"}))
				if !edited.OK {
					t.Fatal(edited.Error)
				}
				p := f.state.changeSets[inputs.ChangeSetID]
				inputs.Generation = p.generation
				inputs.ChangeSetState = p.refObject
			}
			preview := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/change-sets/reconcile-preview", agentapi.ReconciliationPreviewRequest{ReconciliationInputs: inputs}))
			if !preview.OK {
				t.Fatal(preview.Error)
			}
			var response *httptest.ResponseRecorder
			code := "change_set_state_mismatch"
			switch scenario {
			case "component_edit":
				response = postAgent(t, f.handler, "/api/agent/v2/components/edit", agentapi.ComponentEditRequest{StatePreconditions: inputs.StatePreconditions, ComponentID: f.component, Description: stringPointer("Later edit\n")})
			case "rename":
				response = postAgent(t, f.handler, "/api/agent/v2/change-sets/rename", agentapi.ChangeSetRenameRequest{StatePreconditions: inputs.StatePreconditions, Name: "Later name"})
			case "proposal":
				response = postAgent(t, f.handler, "/api/agent/v2/change-sets/edit-proposal", agentapi.ChangeSetEditProposalRequest{StatePreconditions: inputs.StatePreconditions, ProposalMarkdown: "Later proposal\n"})
			case "first_review":
				response = postAgent(t, f.handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: inputs.StatePreconditions})
			case "discard":
				response = postAgent(t, f.handler, "/api/agent/v2/change-sets/discard", agentapi.ChangeSetDiscardRequest{StatePreconditions: inputs.StatePreconditions})
				code = "change_set_not_found"
			case "project_switch":
				response = postAgent(t, f.handler, "/api/agent/v2/projects/create", agentapi.ProjectCreateRequest{Name: "Other project"})
				code = "project_mismatch"
			case "applied":
				applied := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/change-sets/reconcile-apply", agentapi.ReconciliationApplyRequest{ReconciliationInputs: inputs, Resolutions: []architecture.ReconciliationResolution{}}))
				if !applied.OK {
					t.Fatal(applied.Error)
				}
				current := inputs.StatePreconditions
				current.Generation++
				binding := resultMap(t, decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/change-sets/review", agentapi.ChangeSetReviewRequest{StatePreconditions: current})))
				response = postAgent(t, f.handler, "/api/agent/v2/architecture/update", agentapi.ArchitectureUpdateRequest{StoreID: inputs.StoreID, ChangeSetID: inputs.ChangeSetID, Generation: current.Generation, BaseRevision: binding["base_revision"].(string), CandidateTree: binding["candidate_tree"].(string)})
				code = "change_set_not_editable"
			}
			if result := decodeAgentEnvelope(t, response); !result.OK {
				t.Fatalf("ordinary change: %+v", result.Error)
			}
			refs := git(t, "--git-dir", f.storePath, "show-ref")
			requireAgentErrorCode(t, postAgent(t, f.handler, "/api/agent/v2/change-sets/reconcile-apply", agentapi.ReconciliationApplyRequest{ReconciliationInputs: inputs, Resolutions: []architecture.ReconciliationResolution{}}), code)
			if git(t, "--git-dir", f.storePath, "show-ref") != refs {
				t.Fatal("invalidated Apply mutated refs")
			}
		})
	}
}

func TestReconciliationNotRequiredPreservesExactActiveState(t *testing.T) {
	f := newNativeRefreshFixture(t, false)
	state := changeSetState(t, createAgentChangeSet(t, f.handler, f.base.StoreID, f.base.Revision, "Already current"))
	record := f.state.changeSets[state.ChangeSetID]
	inputs := agentapi.ReconciliationInputs{StatePreconditions: state, ChangeSetState: record.refObject, BaseRevision: record.baseRevision, CandidateTree: record.candidate.Tree(), AcceptedRevision: f.state.loadedSnapshot.Revision()}
	refs := git(t, "--git-dir", f.storePath, "show-ref")
	preview := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/change-sets/reconcile-preview", agentapi.ReconciliationPreviewRequest{ReconciliationInputs: inputs}))
	if !preview.OK || resultMap(t, preview)["status"] != "not_required" {
		t.Fatalf("current preview: %+v", preview)
	}
	apply := decodeAgentEnvelope(t, postAgent(t, f.handler, "/api/agent/v2/change-sets/reconcile-apply", agentapi.ReconciliationApplyRequest{ReconciliationInputs: inputs, Resolutions: []architecture.ReconciliationResolution{}}))
	if !apply.OK || resultMap(t, apply)["publication"] != "not_required" || resultMap(t, apply)["change_set_state"] != inputs.ChangeSetState {
		t.Fatalf("current apply: %+v", apply)
	}
	if git(t, "--git-dir", f.storePath, "show-ref") != refs {
		t.Fatal("not-required changed authority")
	}
}
