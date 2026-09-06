package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"workbraid/internal/agentapi"
	"workbraid/internal/architecture"
)

type reconciliationCandidateResponse struct {
	CandidateTree string `json:"candidate_tree"`
	snapshotProjectionResponse
}

type reconciliationPreviewResponse struct {
	architecture.Reconciliation
	Inputs           agentapi.ReconciliationInputs    `json:"inputs"`
	Original         snapshotProjectionResponse       `json:"original"`
	Accepted         snapshotProjectionResponse       `json:"accepted"`
	Proposed         snapshotProjectionResponse       `json:"proposed"`
	ResultCandidate  *reconciliationCandidateResponse `json:"result_candidate,omitempty"`
	RemainingChanges *bool                            `json:"remaining_changes,omitempty"`
}

type reconciliationApplyResponse struct {
	agentChangeSetProjection
	RemainingChanges bool                  `json:"remaining_changes"`
	Publication      string                `json:"publication"`
	Workspace        *architectureResponse `json:"workspace,omitempty"`
}

func reconciliationDomainError(code, reason string) *agentapi.Error {
	details := map[string]any{}
	if reason != "" {
		details["reason"] = reason
	}
	message := agentMessage(code)
	switch code {
	case "reconciliation_unresolved":
		message = "Complete the explicit choices and check them before applying reconciliation."
	case "reconciliation_unsupported":
		message = "This choice requires an Architecture operation that is not supported."
	case "change_set_state_mismatch":
		message = "The active Change Set state changed. Inspect its current state before preparing again; do not replay Apply."
	}
	return &agentapi.Error{Code: code, Message: message, Details: details}
}

// Observe S before generation, basis, Accepted, and not-required checks. A
// response-loss retry can never masquerade as a fresh mutation or a receipt.
func (h *Handler) reconciliationInputsLocked(ctx context.Context, inputs agentapi.ReconciliationInputs) (*pendingChangeSet, *agentapi.Error) {
	if h.loadedProject == nil || h.loadedSnapshot == nil {
		return nil, reconciliationDomainError("project_not_open", "")
	}
	if inputs.StoreID != h.loadedProject.storeID {
		return nil, reconciliationDomainError("project_mismatch", "")
	}
	object, exists, err := h.architecture.ObserveActiveChangeSet(ctx, inputs.StoreID, inputs.ChangeSetID)
	if err != nil {
		return nil, reconciliationDomainError("change_set_unavailable", "state_observation_failed")
	}
	cached := h.changeSets[inputs.ChangeSetID]
	if !exists || cached == nil || cached.refObject != object || object != inputs.ChangeSetState {
		if err := h.loadChangeSetsLocked(ctx, *h.loadedSnapshot); err != nil {
			return nil, reconciliationDomainError("change_set_unavailable", "state_observation_failed")
		}
		current, lifecycleErr := h.editableAgentChangeSetLocked(inputs.ChangeSetID)
		if lifecycleErr != nil {
			return nil, lifecycleErr
		}
		if object != inputs.ChangeSetState || !exists {
			observation := h.observeReconciliationAcceptedLocked(ctx, h.loadedSnapshot.Revision())
			err := reconciliationDomainError("change_set_state_mismatch", "")
			projection := h.agentChangeSetProjectionLocked(current)
			if observation != nil {
				if revision, ok := observation.Details["observed_accepted_revision"].(string); ok {
					projection.OutOfDate = current.baseRevision != revision
					err.Details["observed_accepted_revision"] = revision
				}
			}
			err.Details["change_set"] = projection
			return nil, err
		}
	}
	record, lifecycleErr := h.editableAgentChangeSetLocked(inputs.ChangeSetID)
	if lifecycleErr != nil {
		return nil, lifecycleErr
	}
	if record.generation != inputs.Generation {
		err := reconciliationDomainError("change_set_generation_mismatch", "")
		err.Details["current_generation"] = record.generation
		return nil, err
	}
	if record.baseRevision != inputs.BaseRevision {
		return nil, reconciliationDomainError("invalid_request", "base_revision does not match the exact state")
	}
	if record.candidate == nil {
		return nil, reconciliationDomainError("validation_blocked", "invalid_proposal")
	}
	if record.candidate.Tree() != inputs.CandidateTree {
		return nil, reconciliationDomainError("invalid_request", "candidate_tree does not match the exact state")
	}
	if err := h.observeReconciliationAcceptedLocked(ctx, inputs.AcceptedRevision); err != nil {
		return nil, err
	}
	// Rebuild the exact captured ordinary facts, even for a cached generation.
	reconstructed, err := h.architecture.ConstructCandidate(ctx, record.baseSnapshot, record.changes, h.durableChangeSet(record).Composition)
	if err != nil || reconstructed.Tree() != inputs.CandidateTree {
		return nil, reconciliationDomainError("validation_blocked", "invalid_proposal")
	}
	return clonePending(record), nil
}

func (h *Handler) observeReconciliationAcceptedLocked(ctx context.Context, expected string) *agentapi.Error {
	actual, exists, err := h.architecture.AcceptedRevision(ctx, *h.loadedSnapshot)
	if err != nil {
		h.acceptedIndeterminate = true
		return reconciliationDomainError("refresh_failed", "")
	}
	if !exists || actual != h.loadedSnapshot.Revision() {
		h.markKnownNonCurrentLocked()
		failure := reconciliationDomainError("architecture_non_current", "")
		if exists {
			failure.Details["observed_accepted_revision"] = actual
		}
		return failure
	}
	if h.loadedStale {
		return reconciliationDomainError("architecture_non_current", "")
	}
	if h.acceptedIndeterminate {
		return reconciliationDomainError("refresh_failed", "")
	}
	if actual != expected {
		return reconciliationDomainError("accepted_conflict", "")
	}
	return nil
}

func (h *Handler) calculateReconciliationLocked(ctx context.Context, inputs agentapi.ReconciliationInputs, resolutions []architecture.ReconciliationResolution) (reconciliationPreviewResponse, *pendingChangeSet, *agentapi.Error) {
	record, err := h.reconciliationInputsLocked(ctx, inputs)
	if err != nil {
		return reconciliationPreviewResponse{}, nil, err
	}
	result, calculationErr := h.architecture.Reconcile(ctx, record.baseSnapshot, *h.loadedSnapshot, *record.candidate, resolutions)
	if calculationErr != nil {
		var domain *architecture.ReconciliationError
		if errors.As(calculationErr, &domain) {
			return reconciliationPreviewResponse{}, nil, reconciliationDomainError(domain.Code, domain.Reason)
		}
		return reconciliationPreviewResponse{}, nil, reconciliationDomainError("validation_blocked", "reconciled_architecture_invalid")
	}
	response := reconciliationPreviewResponse{Reconciliation: result, Inputs: inputs, Original: projectSnapshot(record.baseSnapshot, ""), Accepted: projectSnapshot(*h.loadedSnapshot, ""), Proposed: projectSnapshot(record.candidate.Snapshot(), "")}
	if result.Candidate != nil {
		response.ResultCandidate = &reconciliationCandidateResponse{CandidateTree: result.Candidate.Tree(), snapshotProjectionResponse: projectSnapshot(result.Candidate.Snapshot(), "")}
		unchanged, compareErr := h.architecture.ConstructCandidate(ctx, *h.loadedSnapshot, nil, architecture.CandidateComposition{})
		if compareErr != nil {
			return reconciliationPreviewResponse{}, nil, reconciliationDomainError("operation_failed", "")
		}
		remaining := result.Candidate.Tree() != unchanged.Tree()
		response.RemainingChanges = &remaining
	}
	if h.beforeReconciliationReobserve != nil {
		h.beforeReconciliationReobserve()
	}
	if _, err := h.reconciliationInputsLocked(ctx, inputs); err != nil {
		return reconciliationPreviewResponse{}, nil, err
	}
	return response, record, nil
}

func completeReconciliationInputs(inputs agentapi.ReconciliationInputs) bool {
	return requireNonEmpty(inputs.StoreID, inputs.ChangeSetID, inputs.ChangeSetState, inputs.BaseRevision, inputs.CandidateTree, inputs.AcceptedRevision)
}

func (h *Handler) agentReconciliationPreview(w http.ResponseWriter, r *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ReconciliationPreviewRequest](h, w, r)
	if !ok {
		return
	}
	if !completeReconciliationInputs(payload.ReconciliationInputs) {
		h.writeAgentError(w, http.StatusBadRequest, "invalid_request", "Exact S/B/A/P inputs are required.", nil)
		return
	}
	h.stateMutex.Lock()
	result, _, err := h.calculateReconciliationLocked(r.Context(), payload.ReconciliationInputs, payload.Resolutions)
	var envelope agentapi.Envelope
	status := http.StatusOK
	if err != nil {
		status = agentDomainErrorStatus(err)
		envelope = h.agentErrorEnvelopeLocked(err.Code, err.Message, err.Details)
	} else {
		envelope = h.agentSuccessEnvelopeLocked(result)
	}
	h.stateMutex.Unlock()
	writeJSON(w, status, envelope)
}

func (h *Handler) agentReconciliationApply(w http.ResponseWriter, r *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ReconciliationApplyRequest](h, w, r)
	if !ok {
		return
	}
	if !completeReconciliationInputs(payload.ReconciliationInputs) || payload.Resolutions == nil {
		h.writeAgentError(w, http.StatusBadRequest, "invalid_request", "Exact S/B/A/P and resolutions are required; use [] for automatic reconciliation.", nil)
		return
	}
	h.stateMutex.Lock()
	result, err, published := h.applyReconciliationLocked(r.Context(), payload)
	if err == nil && r.Header.Get("Origin") != "" {
		workspace := h.responseForLoadedProjectLocked(*h.loadedSnapshot, h.changeSets[payload.ChangeSetID], h.loadedStale, h.acceptedDiff)
		result.Workspace = &workspace
	}
	var envelope agentapi.Envelope
	status := http.StatusOK
	if err != nil {
		status = agentDomainErrorStatus(err)
		envelope = h.agentErrorEnvelopeLocked(err.Code, err.Message, err.Details)
	} else {
		envelope = h.agentSuccessEnvelopeLocked(result)
	}
	h.stateMutex.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(envelope); err != nil && published {
		// No mutation is retried when encoding/delivery fails. Refresh only the
		// real active record, provided the process still has this project open.
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 10*time.Second)
		defer cancel()
		h.stateMutex.Lock()
		defer h.stateMutex.Unlock()
		if h.loadedSnapshot != nil && h.loadedSnapshot.StoreID() == payload.StoreID {
			_ = h.loadChangeSetsLocked(ctx, *h.loadedSnapshot)
		}
	}
}

func (h *Handler) applyReconciliationLocked(ctx context.Context, payload agentapi.ReconciliationApplyRequest) (reconciliationApplyResponse, *agentapi.Error, bool) {
	preview, record, err := h.calculateReconciliationLocked(ctx, payload.ReconciliationInputs, payload.Resolutions)
	if err != nil {
		return reconciliationApplyResponse{}, err, false
	}
	if preview.Status == "not_required" {
		return reconciliationApplyResponse{agentChangeSetProjection: h.agentChangeSetProjectionLocked(record), RemainingChanges: !pendingChangeSetEmpty(record), Publication: "not_required"}, nil, false
	}
	if preview.Status != "ready" {
		code := "reconciliation_unresolved"
		if preview.Status == "blocked" {
			code = "reconciliation_unsupported"
		}
		err := reconciliationDomainError(code, "")
		if preview.Status == "blocked" {
			for _, conflict := range preview.Conflicts {
				if reason := conflict.Unsupported["reason"]; reason != "" {
					err.Details["reason"] = reason
					break
				}
			}
		}
		err.Details["conflicts"] = preview.Conflicts
		return reconciliationApplyResponse{}, err, false
	}
	if record.generation == ^uint64(0) {
		return reconciliationApplyResponse{}, reconciliationDomainError("operation_failed", "generation_overflow"), false
	}
	durable := h.durableChangeSet(record)
	durable.BaseSnapshot = *h.loadedSnapshot
	durable.BaseRevision = payload.AcceptedRevision
	durable.Generation++
	durable.Review = nil
	durable.Changes = preview.Changes
	durable.Composition = preview.Composition
	durable.Candidate = preview.Candidate
	object, prepareErr := h.architecture.PrepareActiveChangeSet(ctx, payload.StoreID, durable)
	if prepareErr != nil {
		return reconciliationApplyResponse{}, reconciliationDomainError("operation_failed", "prepare_failed"), false
	}
	if h.beforeReconciliationTransaction != nil {
		h.beforeReconciliationTransaction()
	}
	if publishErr := h.architecture.PublishReconciliation(ctx, payload.StoreID, payload.ChangeSetID, payload.AcceptedRevision, payload.ChangeSetState, object); publishErr != nil {
		if _, err := h.reconciliationInputsLocked(ctx, payload.ReconciliationInputs); err != nil {
			if err.Code == "architecture_non_current" {
				err.Code = "accepted_conflict"
				err.Message = agentMessage(err.Code)
			}
			return reconciliationApplyResponse{}, err, false
		}
		return reconciliationApplyResponse{}, reconciliationDomainError("operation_failed", "transaction_failed"), false
	}
	// A successful Git transaction is the only mutation boundary. Recover the
	// actual ref rather than publishing a guess from the prepared candidate.
	if h.afterReconciliationTransaction != nil {
		h.afterReconciliationTransaction()
	}
	recovery, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := h.loadChangeSetsLocked(recovery, *h.loadedSnapshot); err != nil {
		return reconciliationApplyResponse{}, reconciliationDomainError("change_set_unavailable", "published_state_could_not_be_read"), true
	}
	actual, exists, observeErr := h.architecture.ObserveActiveChangeSet(recovery, payload.StoreID, payload.ChangeSetID)
	if observeErr != nil {
		return reconciliationApplyResponse{}, reconciliationDomainError("change_set_unavailable", "state_observation_failed"), true
	}
	current, lifecycleErr := h.editableAgentChangeSetLocked(payload.ChangeSetID)
	if lifecycleErr != nil {
		return reconciliationApplyResponse{}, lifecycleErr, true
	}
	if !exists || current.refObject != actual {
		return reconciliationApplyResponse{}, reconciliationDomainError("change_set_unavailable", "state_changed_during_observation"), true
	}
	publication := "published"
	if actual != object {
		publication = "changed_after_reconciliation"
	}
	observation := h.observeReconciliationAcceptedLocked(recovery, payload.AcceptedRevision)
	projection := h.agentChangeSetProjectionLocked(current)
	if observation != nil {
		if actual, ok := observation.Details["observed_accepted_revision"].(string); ok {
			projection.OutOfDate = current.baseRevision != actual
		}
	}
	return reconciliationApplyResponse{agentChangeSetProjection: projection, RemainingChanges: !pendingChangeSetEmpty(current), Publication: publication}, nil, true
}
