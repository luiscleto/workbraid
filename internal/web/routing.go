package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"
	"workbraid/internal/agentapi"
	"workbraid/internal/architecture"
)

func validRoutingFields(data []byte, set, browser bool) bool {
	d := json.NewDecoder(bytes.NewReader(data))
	t, e := d.Token()
	if e != nil || t != json.Delim('{') {
		return false
	}
	required := map[string]bool{"store_id": false, "diagram_id": false, "source_id": false, "target_id": false, "label": false, "occurrence": false}
	if set {
		required["bend"] = false
	}
	seen := map[string]bool{}
	for d.More() {
		t, e = d.Token()
		k, ok := t.(string)
		if e != nil || !ok || seen[k] {
			return false
		}
		seen[k] = true
		_, known := required[k]
		if !known && k != "change_set_id" && !(k == "generation" && !browser) && !(browser && (k == "project_slug" || k == "expected_revision" || k == "expected_pending_generation" || k == "pending_generation_observed")) {
			return false
		}
		var raw json.RawMessage
		if d.Decode(&raw) != nil {
			return false
		}
		if string(raw) == "null" {
			if browser && k == "expected_pending_generation" {
				continue
			}
			return false
		}
		if k == "pending_generation_observed" {
			var b bool
			if json.Unmarshal(raw, &b) != nil {
				return false
			}
		} else if k == "bend" || k == "occurrence" || k == "generation" || k == "expected_pending_generation" {
			var n int
			if json.Unmarshal(raw, &n) != nil || k == "bend" && !architecture.ValidRoute(architecture.Route{Bend: n}) || k == "occurrence" && n < 1 || k == "generation" && n < 0 {
				return false
			}
		} else {
			var s string
			if json.Unmarshal(raw, &s) != nil {
				return false
			}
		}
	}
	for k := range required {
		if !seen[k] {
			return false
		}
	}
	_, e = d.Token()
	return e == nil && ensureJSONEnd(d) == nil
}
func (h *Handler) routingLocked(ctx context.Context, base architecture.Snapshot, pending *pendingChangeSet, a architecture.RouteAddress, route *architecture.Route) (*pendingChangeSet, bool, string) {
	current := base
	if pending != nil {
		if pending.candidate == nil {
			return pending, false, changeValidationBlocked
		}
		object, exists, err := h.architecture.ObserveActiveChangeSet(ctx, pending.storeID, pending.id)
		if err != nil {
			return pending, false, changeOperationFailed
		}
		if !exists || object != pending.refObject {
			return pending, false, "change_set_state_mismatch"
		}
		current = pending.candidate.Snapshot()
	}
	if !current.HasDiagram(a.DiagramID) {
		return pending, false, changeTargetNotFound
	}
	var selected *architecture.RouteProjection
	for _, v := range current.DiagramRoutes(a.DiagramID) {
		if v.RouteAddress == a {
			copy := v
			selected = &copy
			break
		}
	}
	if selected == nil || !selected.Eligible {
		return pending, false, changeTargetIneligible
	}
	if route != nil && !architecture.ValidRoute(*route) {
		return pending, false, "invalid_request"
	}
	if route == nil && selected.Route == nil || route != nil && (selected.Route != nil && *route == *selected.Route || selected.Route == nil && route.Bend == selected.DisplayBend) {
		return pending, true, ""
	}
	proposed := h.ensurePendingLocked(base, clonePending(pending))
	c := architecture.SetEdgeRoute(current, h.durableChangeSet(proposed).Composition, a, route)
	proposed.edgeRoutes = c.EdgeRoutes
	proposed.nodeSizes = c.NodeSizes
	proposed.architectureVersion = c.ArchitectureVersion
	h.rebuildPendingLocked(ctx, base, proposed)
	if pendingOperationFailed(proposed) {
		return pending, false, changeOperationFailed
	}
	if proposed.candidate == nil {
		return pending, false, changeValidationBlocked
	}
	return proposed, false, ""
}
func (h *Handler) browserRouting(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") != h.expectedOrigin {
		writeJSON(w, http.StatusForbidden, errorResponse{Code: errorOriginMismatch})
		return
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBody))
	set := strings.HasSuffix(r.URL.Path, "/set-route")
	if err != nil || !utf8.Valid(data) || !validRoutingFields(data, set, true) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Code: "invalid_request"})
		return
	}
	var v struct {
		ProjectSlug      string  `json:"project_slug"`
		StoreID          string  `json:"store_id"`
		ChangeSetID      string  `json:"change_set_id"`
		Generation       *uint64 `json:"expected_pending_generation"`
		ExpectedRevision string  `json:"expected_revision"`
		Observed         bool    `json:"pending_generation_observed"`
		architecture.RouteAddress
		Bend int `json:"bend"`
	}
	if json.Unmarshal(data, &v) != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Code: "invalid_request"})
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	base, pending, ok := h.writableV2StateLocked(w, diagramMutationRequest{ProjectSlug: v.ProjectSlug, StoreID: v.StoreID, ChangeSetID: v.ChangeSetID, ExpectedGeneration: v.Generation, ExpectedRevision: v.ExpectedRevision, PendingGenerationObserved: v.Observed, DiagramID: v.DiagramID})
	if !ok {
		return
	}
	var route *architecture.Route
	if set {
		route = &architecture.Route{Bend: v.Bend}
	}
	proposed, unchanged, code := h.routingLocked(r.Context(), base, pending, v.RouteAddress, route)
	if code != "" {
		writeJSON(w, http.StatusConflict, errorResponse{Code: code})
		return
	}
	if unchanged {
		writeJSON(w, http.StatusOK, h.responseForLoadedProjectLocked(*h.loadedSnapshot, pending, h.loadedStale, h.acceptedDiff))
		return
	}
	h.persistBrowserChangeSetMutationLocked(w, r.Context(), v.ChangeSetID, proposed)
}
func (h *Handler) agentRouting(w http.ResponseWriter, r *http.Request, v agentapi.DiagramRestoreDefaultRouteRequest, route *architecture.Route) {
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	base, pending, err := h.checkAgentStateLocked(v.StatePreconditions)
	if err != nil {
		h.writeAgentDomainErrorLocked(w, http.StatusConflict, err)
		return
	}
	a := architecture.RouteAddress{DiagramID: v.DiagramID, SourceID: v.SourceID, TargetID: v.TargetID, Label: v.Label, Occurrence: v.Occurrence}
	proposed, unchanged, code := h.routingLocked(r.Context(), base, pending, a, route)
	if code != "" {
		h.writeAgentErrorLocked(w, http.StatusConflict, code, agentMessage(code), nil)
		return
	}
	if !unchanged && !h.persistAgentMutationLocked(r.Context(), v.ChangeSetID, proposed) {
		h.writeAgentErrorLocked(w, http.StatusConflict, "change_set_state_mismatch", agentMessage("change_set_state_mismatch"), nil)
		return
	}
	h.writeAgentSuccessLocked(w, http.StatusOK, agentMutationResult(proposed, map[string]any{"change_set": h.agentChangeSetProjectionLocked(proposed), "unchanged": unchanged}))
}
func (h *Handler) agentSetRoute(w http.ResponseWriter, r *http.Request) {
	v, ok := decodeAgentRequest[agentapi.DiagramSetRouteRequest](h, w, r)
	if ok {
		h.agentRouting(w, r, v.DiagramRestoreDefaultRouteRequest, &architecture.Route{Bend: v.Bend})
	}
}
func (h *Handler) agentRestoreDefaultRoute(w http.ResponseWriter, r *http.Request) {
	v, ok := decodeAgentRequest[agentapi.DiagramRestoreDefaultRouteRequest](h, w, r)
	if ok {
		h.agentRouting(w, r, v, nil)
	}
}
