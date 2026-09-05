package web

import (
	"context"
	"net/http"

	"workbraid/internal/agentapi"
	"workbraid/internal/architecture"
)

type detailParentOption struct {
	ComponentID      string `json:"component_id"`
	Title            string `json:"title"`
	HomeDiagramID    string `json:"home_diagram_id"`
	HomeDiagramTitle string `json:"home_diagram_title"`
	Filename         string `json:"filename"`
}

type detailParentOptions struct {
	DiagramID     string                     `json:"diagram_id"`
	Title         string                     `json:"title"`
	CurrentAnchor detailParentOption         `json:"current_anchor"`
	Eligible      []detailParentOption       `json:"eligible"`
	CandidateTree string                     `json:"candidate_tree"`
	Candidate     snapshotProjectionResponse `json:"candidate"`
	Generation    *uint64                    `json:"generation"`
}

func (h *Handler) detailParentOptionsLocked(ctx context.Context, base architecture.Snapshot, pending *pendingChangeSet, diagramID string) (detailParentOptions, string) {
	current := base
	tree := base.Revision()
	var generation *uint64
	if pending != nil {
		if pending.candidate == nil {
			return detailParentOptions{}, changeValidationBlocked
		}
		object, exists, err := h.architecture.ObserveActiveChangeSet(ctx, pending.storeID, pending.id)
		if err != nil {
			return detailParentOptions{}, changeOperationFailed
		}
		if !exists || object != pending.refObject {
			return detailParentOptions{}, "change_set_state_mismatch"
		}
		current, tree = pending.candidate.Snapshot(), pending.candidate.Tree()
		value := pending.generation
		generation = &value
	} else {
		candidate, err := h.architecture.ConstructCandidate(ctx, base, nil, architecture.CandidateComposition{})
		if err != nil {
			return detailParentOptions{}, changeOperationFailed
		}
		tree = candidate.Tree()
	}
	if !current.HasDiagram(diagramID) {
		return detailParentOptions{}, changeTargetNotFound
	}
	anchor, _, ok := current.DiagramParent(diagramID)
	if !ok {
		return detailParentOptions{}, changeTargetIneligible
	}
	diagrams := map[string]string{}
	for _, diagram := range current.DiagramProjections() {
		diagrams[diagram.ID] = diagram.Title
	}
	option := func(component architecture.AuthoringComponent) detailParentOption {
		home, _, _ := current.ComponentHome(component.ID)
		return detailParentOption{ComponentID: component.ID, Title: component.Title, HomeDiagramID: home, HomeDiagramTitle: diagrams[home], Filename: component.Filename}
	}
	result := detailParentOptions{DiagramID: diagramID, Title: diagrams[diagramID], Eligible: []detailParentOption{}, CandidateTree: tree, Candidate: projectSnapshot(current, ""), Generation: generation}
	eligible := current.DetailParentComponentIDs(diagramID)
	for _, component := range current.AuthoringComponents() {
		if component.ID == anchor {
			result.CurrentAnchor = option(component)
		}
		if containsString(eligible, component.ID) {
			result.Eligible = append(result.Eligible, option(component))
		}
	}
	return result, ""
}

func (h *Handler) reassignDetailLocked(ctx context.Context, base architecture.Snapshot, pending *pendingChangeSet, diagramID, anchorID string) (*pendingChangeSet, bool, string) {
	options, code := h.detailParentOptionsLocked(ctx, base, pending, diagramID)
	if code != "" {
		return pending, false, code
	}
	if options.CurrentAnchor.ComponentID == anchorID {
		return pending, true, ""
	}
	eligible := false
	for _, option := range options.Eligible {
		if option.ComponentID == anchorID {
			eligible = true
		}
	}
	if !eligible {
		current := base
		if pending != nil {
			current = pending.candidate.Snapshot()
		}
		if !current.HasComponent(anchorID) {
			return pending, false, changeTargetNotFound
		}
		return pending, false, changeTargetIneligible
	}
	proposed := h.ensurePendingLocked(base, clonePending(pending))
	isNew := false
	for index := range proposed.detailDiagrams {
		if proposed.detailDiagrams[index].ID == diagramID {
			proposed.detailDiagrams[index].AnchorComponentID = anchorID
			isNew = true
		}
	}
	if !isNew {
		final := []architecture.DetailReassignment{}
		for _, value := range proposed.detailReassignments {
			if value.DiagramID != diagramID {
				final = append(final, value)
			}
		}
		baseAnchor, _, _ := base.DiagramParent(diagramID)
		if baseAnchor != anchorID {
			final = append(final, architecture.DetailReassignment{DiagramID: diagramID, AnchorComponentID: anchorID})
		}
		proposed.detailReassignments = final
	}
	h.rebuildPendingLocked(ctx, base, proposed)
	if pendingOperationFailed(proposed) {
		return pending, false, changeOperationFailed
	}
	if proposed.candidate == nil {
		return pending, false, changeTargetIneligible
	}
	return proposed, false, ""
}

func (h *Handler) browserDetailParentOptions(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeDiagramMutation(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	base, pending, ok := h.writableV2StateLocked(response, payload)
	if !ok {
		return
	}
	result, code := h.detailParentOptionsLocked(request.Context(), base, pending, payload.DiagramID)
	if code != "" {
		writeJSON(response, http.StatusConflict, errorResponse{Code: code})
		return
	}
	writeJSON(response, http.StatusOK, result)
}

func (h *Handler) browserReassignDetail(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeDiagramMutation(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	base, pending, ok := h.writableV2StateLocked(response, payload)
	if !ok {
		return
	}
	proposed, unchanged, code := h.reassignDetailLocked(request.Context(), base, pending, payload.DiagramID, payload.AnchorComponentID)
	if code != "" {
		writeJSON(response, http.StatusConflict, errorResponse{Code: code})
		return
	}
	if unchanged {
		writeJSON(response, http.StatusOK, h.responseForLoadedProjectLocked(*h.loadedSnapshot, pending, h.loadedStale, h.acceptedDiff))
		return
	}
	h.persistBrowserChangeSetMutationLocked(response, request.Context(), payload.ChangeSetID, proposed)
}

func (h *Handler) agentDetailParentOptions(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.DiagramParentOptionsRequest](h, response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	base, pending, err := h.checkAgentStateLocked(payload.StatePreconditions)
	if err != nil {
		h.writeAgentDomainErrorLocked(response, http.StatusConflict, err)
		return
	}
	result, code := h.detailParentOptionsLocked(request.Context(), base, pending, payload.DiagramID)
	if code != "" {
		h.writeAgentErrorLocked(response, http.StatusConflict, code, agentMessage(code), nil)
		return
	}
	h.writeAgentSuccessLocked(response, http.StatusOK, result)
}

func (h *Handler) agentReassignDetail(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.DiagramReassignDetailRequest](h, response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	base, pending, err := h.checkAgentStateLocked(payload.StatePreconditions)
	if err != nil {
		h.writeAgentDomainErrorLocked(response, http.StatusConflict, err)
		return
	}
	proposed, unchanged, code := h.reassignDetailLocked(request.Context(), base, pending, payload.DiagramID, payload.AnchorComponentID)
	if code != "" {
		h.writeAgentErrorLocked(response, http.StatusConflict, code, agentMessage(code), nil)
		return
	}
	if !unchanged && !h.persistAgentMutationLocked(request.Context(), payload.ChangeSetID, proposed) {
		h.writeAgentErrorLocked(response, http.StatusConflict, "change_set_state_mismatch", agentMessage("change_set_state_mismatch"), nil)
		return
	}
	h.writeAgentSuccessLocked(response, http.StatusOK, agentMutationResult(proposed, map[string]any{"diagram_id": payload.DiagramID, "anchor_component_id": payload.AnchorComponentID, "unchanged": unchanged}))
}
