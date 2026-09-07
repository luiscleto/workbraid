package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"workbraid/internal/agentapi"
	"workbraid/internal/architecture"
)

func (h *Handler) versionError(w http.ResponseWriter, err error) {
	code, message, status := "version_unavailable", "This version could not be loaded. Return to Compare versions and choose an available version.", http.StatusConflict
	if errors.Is(err, architecture.ErrInvalid) {
		code, message, status = "invalid_request", "Choose two complete versions from the same project.", http.StatusBadRequest
	}
	if errors.Is(err, architecture.ErrVersionMoved) {
		code, message = "version_moved", "A selected version has changed. Return to Compare versions and choose it again."
	}
	h.writeAgentErrorLocked(w, status, code, message, nil)
}

func (h *Handler) agentArchitectureVersions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	input, ok := decodeAgentRequest[architecture.VersionPageRequest](h, w, r)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	page, err := h.architecture.VersionCatalog(r.Context(), input)
	if err != nil {
		h.versionError(w, err)
		return
	}
	h.writeAgentSuccessLocked(w, 200, page)
}

func (h *Handler) agentArchitectureCompare(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	input, ok := decodeAgentRequest[agentapi.ArchitectureCompareRequest](h, w, r)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	accepted, before, after, diff, err := h.architecture.CompareVersions(r.Context(), input.StoreID, input.Before, input.After)
	if err != nil {
		h.versionError(w, err)
		return
	}
	b, a, changes := captureReviewPresentation(before.Snapshot, after.Snapshot)
	// Comparison supports valid externally removed objects without extending
	// ordinary authoring or borrowing an acceptance binding.
	afterIDs := map[string]bool{}
	for _, component := range a.Components {
		afterIDs[component.ID] = true
	}
	for _, component := range b.Components {
		if !afterIDs[component.ID] {
			changes.Components = append(changes.Components, reviewComponentChangeResponse{ComponentID: component.ID, Status: "removed", Path: canonicalComponentPath(component.Filename)})
		}
	}
	q := url.Values{"store_id": {input.StoreID}}
	for key, value := range map[string]architecture.VersionSelector{"before": input.Before, "after": input.After} {
		data, _ := json.Marshal(value)
		q.Set(key, string(data))
	}
	h.writeAgentSuccessLocked(w, 200, map[string]any{
		"store_id": input.StoreID, "project_name": accepted.ProjectName(), "project_slug": accepted.ProjectSlug(), "accepted_revision": accepted.Revision(),
		"before_version": before.Info, "after_version": after.Info, "before": b, "after": a, "changes": changes, "diff": diff,
		"report_url": "/projects/" + url.PathEscape(accepted.ProjectSlug()) + "/compare/report?" + q.Encode(),
	})
}
