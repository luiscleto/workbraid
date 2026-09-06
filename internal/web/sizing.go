package web

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"
	"workbraid/internal/agentapi"
	"workbraid/internal/architecture"
)

func (h *Handler) sizingLocked(ctx context.Context, base architecture.Snapshot, pending *pendingChangeSet, d, c string, size *architecture.Size) (*pendingChangeSet, bool, string) {
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
	before, exists := current.DisplayNodeSize(d, c)
	if !exists {
		return pending, false, changeTargetIneligible
	}
	if size == nil {
		v, ok := current.DefaultNodeSize(d, c)
		if !ok {
			return pending, false, changeTargetIneligible
		}
		size = &v
	}
	if !architecture.ValidSize(*size) {
		return pending, false, "invalid_request"
	}
	// Equality deliberately precedes legacy format initialization.
	if before == *size {
		return pending, true, ""
	}
	proposed := h.ensurePendingLocked(base, clonePending(pending))
	composition := architecture.SetNodeSize(current, h.durableChangeSet(proposed).Composition, d, c, *size)
	proposed.nodeSizes = composition.NodeSizes
	proposed.architectureVersion = composition.ArchitectureVersion
	h.rebuildPendingLocked(ctx, base, proposed)
	if pendingOperationFailed(proposed) {
		return pending, false, changeOperationFailed
	}
	if proposed.candidate == nil {
		return pending, false, changeValidationBlocked
	}
	return proposed, false, ""
}
func (h *Handler) browserSizing(w http.ResponseWriter, r *http.Request) {
	contents, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBody))
	if err != nil || !utf8.Valid(contents) || !validPlacementFields(contents, r.URL.Path, true) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Code: "invalid_request"})
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(contents))
	v, ok := h.decodeDiagramMutation(w, r)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	base, pending, ok := h.writableV2StateLocked(w, v)
	if !ok {
		return
	}
	var size *architecture.Size
	if strings.HasSuffix(r.URL.Path, "/set-size") {
		if v.Width == nil || v.Height == nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Code: "invalid_request"})
			return
		}
		size = &architecture.Size{Width: *v.Width, Height: *v.Height}
	}
	proposed, unchanged, code := h.sizingLocked(r.Context(), base, pending, v.DiagramID, v.ComponentID, size)
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
func (h *Handler) agentSizing(w http.ResponseWriter, r *http.Request, v agentapi.DiagramRestoreDefaultSizeRequest, size *architecture.Size) {
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	base, pending, err := h.checkAgentStateLocked(v.StatePreconditions)
	if err != nil {
		h.writeAgentDomainErrorLocked(w, http.StatusConflict, err)
		return
	}
	proposed, unchanged, code := h.sizingLocked(r.Context(), base, pending, v.DiagramID, v.ComponentID, size)
	if code != "" {
		h.writeAgentErrorLocked(w, http.StatusConflict, code, agentMessage(code), nil)
		return
	}
	if !unchanged && !h.persistAgentMutationLocked(r.Context(), v.ChangeSetID, proposed) {
		h.writeAgentErrorLocked(w, http.StatusConflict, "change_set_state_mismatch", agentMessage("change_set_state_mismatch"), nil)
		return
	}
	result := map[string]any{"change_set": h.agentChangeSetProjectionLocked(proposed), "diagram_id": v.DiagramID, "component_id": v.ComponentID, "unchanged": unchanged}
	if proposed != nil {
		result = agentMutationResult(proposed, result)
	}
	h.writeAgentSuccessLocked(w, http.StatusOK, result)
}
func (h *Handler) agentSetSize(w http.ResponseWriter, r *http.Request) {
	v, ok := decodeAgentRequest[agentapi.DiagramSetSizeRequest](h, w, r)
	if !ok {
		return
	}
	h.agentSizing(w, r, v.DiagramRestoreDefaultSizeRequest, &architecture.Size{Width: v.Width, Height: v.Height})
}
func (h *Handler) agentRestoreDefaultSize(w http.ResponseWriter, r *http.Request) {
	v, ok := decodeAgentRequest[agentapi.DiagramRestoreDefaultSizeRequest](h, w, r)
	if !ok {
		return
	}
	h.agentSizing(w, r, v, nil)
}
