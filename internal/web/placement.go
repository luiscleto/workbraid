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

// Placement requests are flat, closed facts: duplicate coordinates must never
// silently select one of two authored values.
func validPlacementFields(contents []byte, path string, browser bool) bool {
	decoder := json.NewDecoder(bytes.NewReader(contents))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return false
	}
	seen := map[string]bool{}
	for decoder.More() {
		token, err = decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok || seen[key] {
			return false
		}
		seen[key] = true
		if browser && (key == "title" || key == "anchor_component_id") {
			return false
		}
		if strings.HasSuffix(path, "/auto-layout") && key == "component_id" {
			return false
		}
		var raw json.RawMessage
		if decoder.Decode(&raw) != nil {
			return false
		}
		if key == "x" || key == "y" {
			if !strings.HasSuffix(path, "/set-position") || string(raw) == "null" {
				return false
			}
			var coordinate int
			if json.Unmarshal(raw, &coordinate) != nil || coordinate < -architecture.PositionLimit || coordinate > architecture.PositionLimit {
				return false
			}
		}
		if key == "width" || key == "height" {
			if !strings.HasSuffix(path, "/set-size") || string(raw) == "null" {
				return false
			}
			var n int
			if json.Unmarshal(raw, &n) != nil || key == "width" && (n < 80 || n > 1600) || key == "height" && (n < 48 || n > 1200) {
				return false
			}
		}
	}
	if _, err = decoder.Token(); err != nil || ensureJSONEnd(decoder) != nil {
		return false
	}
	return (!strings.HasSuffix(path, "/set-position") || seen["x"] && seen["y"]) && (!strings.HasSuffix(path, "/set-size") || seen["width"] && seen["height"])
}

func (h *Handler) placementLocked(ctx context.Context, base architecture.Snapshot, pending *pendingChangeSet, d, c string, p *architecture.Position, autoLayout bool) (*pendingChangeSet, bool, string) {
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
	if !current.HasDiagram(d) {
		return pending, false, changeTargetIneligible
	}
	if p != nil && !architecture.ValidPosition(*p) {
		return pending, false, "invalid_request"
	}
	targets := []architecture.NodePositionChange{}
	if autoLayout {
		layout, err := current.AutoLayout(d)
		if err != nil {
			return pending, false, changeOperationFailed
		}
		for _, v := range layout {
			before, _ := current.NodePosition(d, v.ComponentID)
			if before == nil || *before != *v.Position {
				targets = append(targets, v)
			}
		}
	} else {
		if p == nil {
			return pending, false, "invalid_request"
		}
		before, exists := current.NodePosition(d, c)
		if !exists {
			return pending, false, changeTargetIneligible
		}
		if !(before == nil && p == nil || before != nil && p != nil && *before == *p) {
			targets = append(targets, architecture.NodePositionChange{DiagramID: d, ComponentID: c, Position: p})
		}
	}
	if len(targets) == 0 && current.FormatVersion() >= 3 {
		return pending, true, ""
	}
	proposed := h.ensurePendingLocked(base, clonePending(pending))
	proposed.architectureVersion = max(3, current.FormatVersion())
	for _, v := range targets {
		composition := architecture.SetNodePosition(base, h.durableChangeSet(proposed).Composition, d, v.ComponentID, v.Position)
		proposed.nodePositions = composition.NodePositions
		proposed.architectureVersion = composition.ArchitectureVersion
	}
	h.rebuildPendingLocked(ctx, base, proposed)
	if pendingOperationFailed(proposed) {
		return pending, false, changeOperationFailed
	}
	if proposed.candidate == nil {
		return pending, false, changeValidationBlocked
	}
	proposed.nodePositions = architecture.NormalizeNodePositions(base, proposed.candidate.Snapshot(), proposed.nodePositions)
	candidate, err := h.constructCandidate(ctx, base, proposed, false)
	if err != nil {
		return pending, false, changeOperationFailed
	}
	proposed.candidate = &candidate
	return proposed, false, ""
}

func (h *Handler) browserPlacement(w http.ResponseWriter, r *http.Request) {
	contents, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBody))
	if err != nil || !utf8.Valid(contents) || !validPlacementFields(contents, r.URL.Path, true) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Code: "invalid_request"})
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(contents))
	payload, ok := h.decodeDiagramMutation(w, r)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	base, pending, ok := h.writableV2StateLocked(w, payload)
	if !ok {
		return
	}
	var p *architecture.Position
	if strings.HasSuffix(r.URL.Path, "/set-position") {
		if payload.X == nil || payload.Y == nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Code: "invalid_request"})
			return
		}
		p = &architecture.Position{X: *payload.X, Y: *payload.Y}
	} else if payload.X != nil || payload.Y != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Code: "invalid_request"})
		return
	}
	proposed, unchanged, code := h.placementLocked(r.Context(), base, pending, payload.DiagramID, payload.ComponentID, p, strings.HasSuffix(r.URL.Path, "/auto-layout"))
	if code != "" {
		writeJSON(w, http.StatusConflict, errorResponse{Code: code})
		return
	}
	if unchanged {
		writeJSON(w, http.StatusOK, h.responseForLoadedProjectLocked(*h.loadedSnapshot, pending, h.loadedStale, h.acceptedDiff))
		return
	}
	h.persistBrowserChangeSetMutationLocked(w, r.Context(), payload.ChangeSetID, proposed)
}

func (h *Handler) agentPlacement(w http.ResponseWriter, r *http.Request, state agentapi.StatePreconditions, d, c string, p *architecture.Position, all bool) {
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	base, pending, err := h.checkAgentStateLocked(state)
	if err != nil {
		h.writeAgentDomainErrorLocked(w, http.StatusConflict, err)
		return
	}
	proposed, unchanged, code := h.placementLocked(r.Context(), base, pending, d, c, p, all)
	if code != "" {
		h.writeAgentErrorLocked(w, http.StatusConflict, code, agentMessage(code), nil)
		return
	}
	if !unchanged && !h.persistAgentMutationLocked(r.Context(), state.ChangeSetID, proposed) {
		h.writeAgentErrorLocked(w, http.StatusConflict, "change_set_state_mismatch", agentMessage("change_set_state_mismatch"), nil)
		return
	}
	result := map[string]any{"change_set": h.agentChangeSetProjectionLocked(proposed), "diagram_id": d, "component_id": c, "unchanged": unchanged}
	if proposed != nil {
		result = agentMutationResult(proposed, result)
	}
	h.writeAgentSuccessLocked(w, http.StatusOK, result)
}
func (h *Handler) agentSetPosition(w http.ResponseWriter, r *http.Request) {
	v, ok := decodeAgentRequest[agentapi.DiagramSetPositionRequest](h, w, r)
	if !ok {
		return
	}
	h.agentPlacement(w, r, v.StatePreconditions, v.DiagramID, v.ComponentID, &architecture.Position{X: v.X, Y: v.Y}, false)
}
func (h *Handler) agentAutoLayout(w http.ResponseWriter, r *http.Request) {
	v, ok := decodeAgentRequest[agentapi.DiagramAutoLayoutRequest](h, w, r)
	if !ok {
		return
	}
	h.agentPlacement(w, r, v.StatePreconditions, v.DiagramID, "", nil, true)
}
func (h *Handler) agentPositions(w http.ResponseWriter, r *http.Request) {
	v, ok := decodeAgentRequest[agentapi.DiagramPositionsRequest](h, w, r)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if h.loadedSnapshot == nil {
		h.writeAgentErrorLocked(w, http.StatusConflict, "project_not_open", agentMessage("project_not_open"), nil)
		return
	}
	if v.StoreID != h.loadedSnapshot.StoreID() {
		h.writeAgentErrorLocked(w, http.StatusConflict, "project_mismatch", agentMessage("project_mismatch"), nil)
		return
	}
	snapshot := *h.loadedSnapshot
	result := map[string]any{"revision": snapshot.Revision()}
	if v.ChangeSetID != "" {
		pending := h.changeSets[v.ChangeSetID]
		if pending == nil {
			for _, unavailable := range h.unavailableChangeSets {
				if unavailable.ID == v.ChangeSetID {
					h.writeAgentErrorLocked(w, http.StatusConflict, "change_set_unavailable", agentMessage("change_set_unavailable"), map[string]any{"change_set_id": v.ChangeSetID, "reason": unavailable.Reason})
					return
				}
			}
			h.writeAgentErrorLocked(w, http.StatusNotFound, "change_set_not_found", agentMessage("change_set_not_found"), nil)
			return
		}
		if pending.candidate == nil {
			h.writeAgentErrorLocked(w, http.StatusConflict, changeValidationBlocked, agentMessage(changeValidationBlocked), nil)
			return
		}
		snapshot = pending.candidate.Snapshot()
		result = map[string]any{"change_set": h.agentChangeSetProjectionLocked(pending)}
	}
	for _, diagram := range snapshot.DiagramProjections() {
		if diagram.ID == v.DiagramID {
			if strings.HasSuffix(r.URL.Path, "/shapes") || strings.HasSuffix(r.URL.Path, "/notes") {
				result["diagram_id"], result["architecture_version"] = diagram.ID, snapshot.FormatVersion()
				if strings.HasSuffix(r.URL.Path, "/notes") {
					result["notes"] = diagram.Notes
				} else {
					result["shapes"] = diagram.Shapes
				}
				h.writeAgentSuccessLocked(w, http.StatusOK, result)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/routes") {
				result["diagram_id"] = diagram.ID
				result["architecture_version"] = snapshot.FormatVersion()
				result["routes"] = snapshot.DiagramRoutes(diagram.ID)
				h.writeAgentSuccessLocked(w, http.StatusOK, result)
				return
			}
			titles := map[string]string{}
			for _, c := range snapshot.AuthoringComponents() {
				titles[c.ID] = c.Title
			}
			appearances := []map[string]any{}
			for _, a := range diagram.Appearances {
				appearances = append(appearances, map[string]any{"component_id": a.ComponentID, "title": titles[a.ComponentID], "role": a.Role, "size": a.Size, "display_size": a.DisplaySize, "size_source": a.SizeSource, "position": a.Position, "display_position": a.DisplayPosition, "position_source": a.PositionSource})
			}
			boundaries := []map[string]any{}
			for _, b := range diagram.Boundaries {
				boundaries = append(boundaries, map[string]any{"component_id": b.ComponentID, "title": b.Title, "role": "boundary", "home_diagram_id": b.HomeDiagramID, "home_diagram_title": b.HomeDiagramTitle, "size": b.Size, "display_size": b.DisplaySize, "size_source": b.SizeSource, "position": b.Position, "display_position": b.DisplayPosition, "position_source": b.PositionSource})
			}
			result["boundaries"] = boundaries
			result["diagram_id"] = diagram.ID
			result["architecture_version"] = snapshot.FormatVersion()
			result["appearances"] = appearances
			h.writeAgentSuccessLocked(w, http.StatusOK, result)
			return
		}
	}
	h.writeAgentErrorLocked(w, http.StatusNotFound, changeTargetNotFound, agentMessage(changeTargetNotFound), nil)
}
