package web

import (
	"errors"
	"net/http"

	"workbraid/internal/agentapi"
	"workbraid/internal/architecture"
)

func (h *Handler) agentReviewSubmissionsList(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ReviewSubmissionsListRequest](h, response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if h.loadedProject == nil || h.loadedProject.storeID != payload.StoreID {
		h.writeAgentErrorLocked(response, http.StatusConflict, "project_mismatch", agentMessage("project_mismatch"), nil)
		return
	}
	values := make([]reviewSubmissionSummaryResponse, 0)
	for _, value := range h.reviewSummariesLocked() {
		if value.ChangeSetID == payload.ChangeSetID {
			values = append(values, value)
		}
	}
	unavailable := make([]unavailableReviewResponse, 0)
	for _, value := range h.unavailableReviews {
		if value.ChangeSetID == payload.ChangeSetID {
			unavailable = append(unavailable, unavailableReviewResponse{ChangeSetID: value.ChangeSetID, ReviewID: value.ReviewID, Reason: value.Reason})
		}
	}
	h.writeAgentSuccessLocked(response, http.StatusOK, map[string]any{"reviews": values, "unavailable": unavailable})
}

func (h *Handler) agentReviewSubmissionInspect(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ReviewSubmissionInspectRequest](h, response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if h.loadedProject == nil || h.loadedProject.storeID != payload.StoreID {
		h.writeAgentErrorLocked(response, http.StatusConflict, "project_mismatch", agentMessage("project_mismatch"), nil)
		return
	}
	review := h.reviews[payload.ReviewID]
	if review.ID == "" || review.ChangeSetID != payload.ChangeSetID {
		for _, value := range h.unavailableReviews {
			if value.ChangeSetID == payload.ChangeSetID && value.ReviewID == payload.ReviewID {
				h.writeAgentErrorLocked(response, http.StatusConflict, "review_submission_unavailable", agentMessage("review_submission_unavailable"), map[string]any{"reason": value.Reason})
				return
			}
		}
		h.writeAgentErrorLocked(response, http.StatusNotFound, "review_submission_not_found", agentMessage("review_submission_not_found"), nil)
		return
	}
	value, err := h.fullReviewResponseLocked(request.Context(), review)
	if err != nil {
		h.writeAgentErrorLocked(response, http.StatusConflict, "review_submission_unavailable", agentMessage("review_submission_unavailable"), nil)
		return
	}
	h.writeAgentSuccessLocked(response, http.StatusOK, value)
}

func (h *Handler) agentReviewSubmissionSubmit(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ReviewSubmissionSubmitRequest](h, response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if h.loadedProject == nil || h.loadedProject.storeID != payload.StoreID {
		h.writeAgentErrorLocked(response, http.StatusConflict, "project_mismatch", agentMessage("project_mismatch"), nil)
		return
	}
	current := h.changeSetLocked(payload.ChangeSetID)
	if current == nil {
		h.writeAgentErrorLocked(response, http.StatusConflict, "review_submission_not_allowed", agentMessage("review_submission_not_allowed"), nil)
		return
	}
	if current.refObject != payload.ReviewedState || current.review == nil || current.review.baseRevision != payload.BaseRevision || current.review.candidateTree != payload.CandidateTree || current.review.generation != payload.Generation {
		h.writeAgentErrorLocked(response, http.StatusConflict, "review_invalidated", agentMessage("review_invalidated"), nil)
		return
	}
	comments := make([]architecture.ReviewCommentInput, len(payload.Comments))
	for index, value := range payload.Comments {
		comments[index] = architecture.ReviewCommentInput{Body: value.Body, Anchor: architecture.ReviewAnchor{
			Kind: value.Anchor.Kind, Side: value.Anchor.Side, ComponentID: value.Anchor.ComponentID, DiagramID: value.Anchor.DiagramID,
			Aspect: value.Anchor.Aspect, DetailDiagramID: value.Anchor.DetailDiagramID, SourceComponentID: value.Anchor.SourceComponentID,
			TargetComponentID: value.Anchor.TargetComponentID, Label: value.Anchor.Label, Occurrence: value.Anchor.Occurrence,
			StartLine: value.Anchor.StartLine, EndLine: value.Anchor.EndLine,
		}}
	}
	created, err := h.architecture.SubmitReview(request.Context(), payload.StoreID, architecture.ReviewSubmissionInput{
		ChangeSetID: payload.ChangeSetID, ReviewedState: payload.ReviewedState,
		Binding: architecture.ReviewBinding{BaseRevision: payload.BaseRevision, CandidateTree: payload.CandidateTree, Generation: payload.Generation},
		Verdict: payload.Verdict, Author: payload.Author, Body: payload.Body, Comments: comments,
	})
	if err != nil {
		code, status, details := "review_submission_unavailable", http.StatusConflict, map[string]any{}
		switch {
		case errors.Is(err, architecture.ErrReviewRequestInvalid):
			code, status = "invalid_request", http.StatusBadRequest
		case errors.Is(err, architecture.ErrReviewAnchorInvalid):
			code, status = "review_anchor_invalid", http.StatusUnprocessableEntity
			var anchorErr *architecture.ReviewAnchorValidationError
			if errors.As(err, &anchorErr) {
				details["comment_index"] = anchorErr.CommentIndex
				details["reason"] = anchorErr.Reason
			}
		case errors.Is(err, architecture.ErrReviewInvalidated):
			code = "review_invalidated"
		case errors.Is(err, architecture.ErrReviewNotAllowed):
			code = "review_submission_not_allowed"
		}
		h.writeAgentErrorLocked(response, status, code, agentMessage(code), details)
		return
	}
	h.reviews[created.ID] = created
	value, err := h.fullReviewResponseLocked(request.Context(), created)
	if err != nil {
		h.writeAgentErrorLocked(response, http.StatusInternalServerError, "operation_failed", agentMessage("operation_failed"), nil)
		return
	}
	h.writeAgentSuccessLocked(response, http.StatusCreated, value)
}
