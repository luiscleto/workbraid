package web

import (
	"net/http"
	"net/url"

	"workbraid/internal/architecture"
)

func printablePath(slug, id, lifecycle, state string) string {
	key := "reviewed_state"
	if lifecycle == "applied" {
		key = "applied_state"
	}
	return "/projects/" + url.PathEscape(slug) + "/proposals/" + url.PathEscape(id) + "/print?" + key + "=" + url.QueryEscape(state)
}

// printProposal resolves the catalog and existing durable records without
// publishing project selection, refreshing the workbench, or preparing Review.
func (h *Handler) printProposal(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	q := r.URL.Query()
	fail := func(code string, status int) { writeJSON(w, status, errorResponse{Code: code}) }
	for key, values := range q {
		if len(values) != 1 || (key != "project_slug" && key != "store_id" && key != "change_set_id" && key != "reviewed_state" && key != "applied_state" && key != "review_id") {
			fail("invalid_request", 400)
			return
		}
	}
	slug, store, id := q.Get("project_slug"), q.Get("store_id"), q.Get("change_set_id")
	s, t, rid := q.Get("reviewed_state"), q.Get("applied_state"), q.Get("review_id")
	modes := 0
	for _, key := range []string{"reviewed_state", "applied_state", "review_id"} {
		if q.Has(key) {
			modes++
		}
	}
	if slug == "" || store == "" || id == "" || modes != 1 || (s == "" && t == "" && rid == "") {
		fail("invalid_request", 400)
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	accepted, err := h.architecture.OpenProject(r.Context(), slug)
	if err != nil {
		fail("project_unavailable", 409)
		return
	}
	if accepted.StoreID() != store {
		fail("project_mismatch", 409)
		return
	}
	records, unavailable, err := h.architecture.LoadChangeSets(r.Context(), store)
	if err != nil {
		fail("change_set_unavailable", 409)
		return
	}
	var current *architecture.ChangeSet
	for i := range records {
		if records[i].ID == id {
			current = &records[i]
			break
		}
	}
	var change architecture.ChangeSet
	lifecycle, state := "no_longer_active", ""
	for _, value := range unavailable {
		if value.ID == id {
			lifecycle = "unavailable"
		}
	}
	if current != nil {
		lifecycle = current.Lifecycle
	}
	if rid != "" {
		reviews, bad, loadErr := h.architecture.LoadReviews(r.Context(), store, id)
		if loadErr != nil {
			fail("review_submission_unavailable", 409)
			return
		}
		found := false
		for _, review := range reviews {
			if review.ID == rid {
				change = review.ReviewedChange
				state = review.ReviewedState
				found = true
				break
			}
		}
		if !found {
			for _, value := range bad {
				if value.ReviewID == rid {
					fail("review_submission_unavailable", 409)
					return
				}
			}
			fail("review_submission_not_found", 404)
			return
		}
	} else {
		if current == nil {
			for _, value := range unavailable {
				if value.ID == id {
					fail("change_set_unavailable", 409)
					return
				}
			}
			fail("change_set_not_found", 404)
			return
		}
		change = *current
		state = change.RefObject
		if (s != "" && (change.Lifecycle != "active" || s != state)) || (t != "" && (change.Lifecycle != "applied" || t != state)) {
			fail("review_invalidated", 409)
			return
		}
	}
	if change.Review == nil || change.Candidate == nil {
		fail("review_required", 409)
		return
	}
	diff, err := h.architecture.CandidateDiff(r.Context(), change.BaseSnapshot, *change.Candidate)
	if err != nil {
		fail("review_submission_unavailable", 409)
		return
	}
	before, with, comparison := captureReviewPresentation(change.BaseSnapshot, change.Candidate.Snapshot())
	review := reviewResponse{ChangeSetID: id, BaseRevision: change.BaseRevision, CandidateTree: change.Candidate.Tree(), Generation: change.Generation, Diff: string(diff), Before: before, WithChanges: with, Comparison: comparison}
	if t == "" {
		review.ReviewedState = state
	}
	writeJSON(w, 200, map[string]any{"project_name": accepted.ProjectName(), "store_id": store, "name": change.Name, "proposal_markdown": change.Proposal, "lifecycle": lifecycle, "review_id": rid, "state": state, "applied_revision": change.AppliedRevision, "accepted_revision": accepted.Revision(), "review": review})
}
