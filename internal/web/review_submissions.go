package web

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"time"

	"workbraid/internal/architecture"
)

type reviewSubmissionSummaryResponse struct {
	ID                string                     `json:"id"`
	ChangeSetID       string                     `json:"change_set_id"`
	ReviewedState     string                     `json:"reviewed_state"`
	Binding           architecture.ReviewBinding `json:"binding"`
	Verdict           string                     `json:"verdict"`
	Author            string                     `json:"author"`
	SubmittedAt       time.Time                  `json:"submitted_at"`
	CommentCount      int                        `json:"comment_count"`
	Lifecycle         string                     `json:"lifecycle"`
	CurrentGeneration bool                       `json:"current_generation"`
	OutOfDate         *bool                      `json:"out_of_date,omitempty"`
}

type reviewCommentResponse struct {
	ID     string                    `json:"id"`
	Body   string                    `json:"body"`
	Anchor architecture.ReviewAnchor `json:"anchor"`
}

type reviewSubmissionResponse struct {
	PrintableURL string `json:"printable_url"`
	reviewSubmissionSummaryResponse
	Body             string                  `json:"body"`
	Comments         []reviewCommentResponse `json:"comments"`
	ProposalMarkdown string                  `json:"proposal_markdown"`
	Review           reviewResponse          `json:"review"`
}

type unavailableReviewResponse struct {
	ChangeSetID string `json:"change_set_id,omitempty"`
	ReviewID    string `json:"review_id,omitempty"`
	Reason      string `json:"reason"`
}

type reviewSubmissionInspectRequest struct {
	ProjectSlug string `json:"project_slug"`
	StoreID     string `json:"store_id"`
	ChangeSetID string `json:"change_set_id"`
	ReviewID    string `json:"review_id"`
}

type reviewSubmissionRequest struct {
	ProjectSlug   string                            `json:"project_slug"`
	StoreID       string                            `json:"store_id"`
	ChangeSetID   string                            `json:"change_set_id"`
	ReviewedState string                            `json:"reviewed_state"`
	BaseRevision  string                            `json:"base_revision"`
	CandidateTree string                            `json:"candidate_tree"`
	Generation    uint64                            `json:"generation"`
	Verdict       string                            `json:"verdict"`
	Author        string                            `json:"author"`
	Body          string                            `json:"body"`
	Comments      []architecture.ReviewCommentInput `json:"comments"`
}

func (h *Handler) reviewSummaryLocked(review architecture.ReviewSubmission) reviewSubmissionSummaryResponse {
	result := reviewSubmissionSummaryResponse{
		ID: review.ID, ChangeSetID: review.ChangeSetID, ReviewedState: review.ReviewedState, Binding: review.Binding,
		Verdict: review.Verdict, Author: review.Author, SubmittedAt: review.SubmittedAt,
		CommentCount: len(review.Comments), Lifecycle: "no_longer_active",
	}
	if current := h.changeSets[review.ChangeSetID]; current != nil {
		result.Lifecycle = current.lifecycle
		bindingMatches := current.generation == review.Binding.Generation && current.review != nil &&
			current.review.baseRevision == review.Binding.BaseRevision && current.review.candidateTree == review.Binding.CandidateTree && current.review.generation == review.Binding.Generation
		result.CurrentGeneration = bindingMatches && (current.lifecycle == "applied" || current.refObject == review.ReviewedState)
		if current.lifecycle == "active" && h.loadedSnapshot != nil && !h.loadedStale && !h.acceptedIndeterminate {
			outOfDate := current.baseRevision != h.loadedSnapshot.Revision()
			result.OutOfDate = &outOfDate
		}
	}
	return result
}

func (h *Handler) reviewSummariesLocked() []reviewSubmissionSummaryResponse {
	values := make([]reviewSubmissionSummaryResponse, 0, len(h.reviews))
	for _, review := range h.reviews {
		values = append(values, h.reviewSummaryLocked(review))
	}
	sort.Slice(values, func(i, j int) bool {
		if !values[i].SubmittedAt.Equal(values[j].SubmittedAt) {
			return values[i].SubmittedAt.After(values[j].SubmittedAt)
		}
		return values[i].ID < values[j].ID
	})
	return values
}

func (h *Handler) fullReviewResponseLocked(ctx context.Context, review architecture.ReviewSubmission) (reviewSubmissionResponse, error) {
	change := review.ReviewedChange
	if change.Candidate == nil {
		return reviewSubmissionResponse{}, errors.New("review candidate is unavailable")
	}
	diff, err := h.architecture.CandidateDiff(ctx, change.BaseSnapshot, *change.Candidate)
	if err != nil {
		return reviewSubmissionResponse{}, err
	}
	before, withChanges, comparison := captureReviewPresentation(change.BaseSnapshot, change.Candidate.Snapshot())
	comments := make([]reviewCommentResponse, len(review.Comments))
	slug := change.BaseSnapshot.ProjectSlug()
	if h.loadedProject != nil {
		slug = h.loadedProject.projectSlug
	}
	for index, comment := range review.Comments {
		comments[index] = reviewCommentResponse{ID: comment.ID, Body: comment.Body, Anchor: comment.Anchor}
	}
	return reviewSubmissionResponse{
		PrintableURL:                    "/projects/" + slug + "/proposals/" + review.ChangeSetID + "/reviews/" + review.ID + "/print",
		reviewSubmissionSummaryResponse: h.reviewSummaryLocked(review), Body: review.Body, Comments: comments,
		ProposalMarkdown: change.Proposal,
		Review: reviewResponse{ChangeSetID: review.ChangeSetID, ReviewedState: review.ReviewedState, Diff: string(diff),
			BaseRevision: review.Binding.BaseRevision, CandidateTree: review.Binding.CandidateTree, Generation: review.Binding.Generation,
			Before: before, WithChanges: withChanges, Comparison: comparison},
	}, nil
}

func (h *Handler) inspectReviewSubmission(response http.ResponseWriter, request *http.Request) {
	var payload reviewSubmissionInspectRequest
	if !h.decodeBrowserJSON(response, request, &payload) {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if !h.matchesLoadedProjectLocked(payload.ProjectSlug, payload.StoreID) {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureNotOpen})
		return
	}
	review := h.reviews[payload.ReviewID]
	if review.ID == "" || review.ChangeSetID != payload.ChangeSetID {
		for _, unavailable := range h.unavailableReviews {
			if unavailable.ChangeSetID == payload.ChangeSetID && unavailable.ReviewID == payload.ReviewID {
				writeJSON(response, http.StatusConflict, errorResponse{Code: "review_submission_unavailable"})
				return
			}
		}
		writeJSON(response, http.StatusNotFound, errorResponse{Code: "review_submission_not_found"})
		return
	}
	value, err := h.fullReviewResponseLocked(request.Context(), review)
	if err != nil {
		writeJSON(response, http.StatusConflict, errorResponse{Code: "review_submission_unavailable"})
		return
	}
	result := h.currentArchitectureResponseLocked()
	historical := pendingFromDurableChangeSet(review.ReviewedChange)
	historical.lifecycle = value.Lifecycle
	historical.review = &reviewBinding{baseRevision: review.Binding.BaseRevision, candidateTree: review.Binding.CandidateTree, generation: review.Binding.Generation, diff: value.Review.Diff, candidate: *review.ReviewedChange.Candidate}
	result.Changes = responseForSnapshot(review.ReviewedChange.BaseSnapshot, historical, false, "").Changes
	if result.Changes != nil {
		result.Changes.Review = &value.Review
		result.Changes.ReadOnly = true
		result.Changes.OutOfDate = value.OutOfDate != nil && *value.OutOfDate
	}
	result.SubmittedReview = &value
	writeJSON(response, http.StatusOK, result)
}

func (h *Handler) submitReviewSubmission(response http.ResponseWriter, request *http.Request) {
	var payload reviewSubmissionRequest
	if !h.decodeBrowserJSON(response, request, &payload) {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if !h.matchesLoadedProjectLocked(payload.ProjectSlug, payload.StoreID) {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureNotOpen})
		return
	}
	current := h.changeSetLocked(payload.ChangeSetID)
	if current == nil {
		writeJSON(response, http.StatusConflict, errorResponse{Code: "review_submission_not_allowed"})
		return
	}
	if current.review == nil {
		writeJSON(response, http.StatusConflict, errorResponse{Code: "review_submission_not_allowed"})
		return
	}
	if current.refObject != payload.ReviewedState || current.review.baseRevision != payload.BaseRevision || current.review.candidateTree != payload.CandidateTree || current.review.generation != payload.Generation {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorReviewChanged})
		return
	}
	created, err := h.architecture.SubmitReview(request.Context(), payload.StoreID, architecture.ReviewSubmissionInput{
		ChangeSetID: payload.ChangeSetID, ReviewedState: payload.ReviewedState,
		Binding: architecture.ReviewBinding{BaseRevision: payload.BaseRevision, CandidateTree: payload.CandidateTree, Generation: payload.Generation},
		Verdict: payload.Verdict, Author: payload.Author, Body: payload.Body, Comments: payload.Comments,
	})
	if err != nil {
		code, status := "review_submission_unavailable", http.StatusConflict
		switch {
		case errors.Is(err, architecture.ErrReviewRequestInvalid):
			code, status = errorLookupFailed, http.StatusBadRequest
		case errors.Is(err, architecture.ErrReviewAnchorInvalid):
			code, status = "review_anchor_invalid", http.StatusUnprocessableEntity
		case errors.Is(err, architecture.ErrReviewInvalidated):
			code = errorReviewChanged
		case errors.Is(err, architecture.ErrReviewNotAllowed):
			code = "review_submission_not_allowed"
		}
		writeJSON(response, status, errorResponse{Code: code})
		return
	}
	h.reviews[created.ID] = created
	result := h.currentArchitectureResponseLocked()
	result.ActionReviewID = created.ID
	value, fullErr := h.fullReviewResponseLocked(request.Context(), created)
	if fullErr == nil {
		result.SubmittedReview = &value
	}
	writeJSON(response, http.StatusCreated, result)
}
