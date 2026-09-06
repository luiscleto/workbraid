package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"sort"
	"strings"
	"unicode/utf8"

	"workbraid/internal/agentapi"
	"workbraid/internal/architecture"
)

type agentArchitectureProjection struct {
	ArchitectureVersion int                        `json:"architecture_version"`
	Project             agentapi.ProjectContext    `json:"project"`
	Revision            string                     `json:"revision"`
	RootDiagramID       string                     `json:"root_diagram_id"`
	Components          []agentComponentProjection `json:"components"`
	Diagrams            []agentDiagramProjection   `json:"diagrams"`
}

type agentComponentProjection struct {
	componentResponse
	HomeDiagramID       string   `json:"home_diagram_id"`
	ReferenceDiagramIDs []string `json:"reference_diagram_ids"`
}

type agentDiagramProjection struct {
	diagramResponse
	ChildDiagramIDs []string `json:"child_diagram_ids"`
}

type agentValidationProjection struct {
	Code                 string `json:"code,omitempty"`
	ComponentID          string `json:"component_id,omitempty"`
	RelationshipPosition int    `json:"relationship_position,omitempty"`
	RelationshipField    string `json:"relationship_field,omitempty"`
	DiagramID            string `json:"diagram_id,omitempty"`
	DiagramField         string `json:"diagram_field,omitempty"`
}

type agentChangeSetProjection struct {
	ArchitectureVersion int                                      `json:"architecture_version"`
	NodePositions       []architecture.NodePositionChange        `json:"node_positions"`
	StateObject         string                                   `json:"change_set_state"`
	DetailReassignments []architecture.DetailReassignment        `json:"detail_reassignments"`
	ID                  string                                   `json:"id"`
	Name                string                                   `json:"name"`
	Lifecycle           string                                   `json:"lifecycle"`
	BaseRevision        string                                   `json:"base_revision"`
	Generation          uint64                                   `json:"generation"`
	Proposal            string                                   `json:"proposal_markdown"`
	AppliedRevision     string                                   `json:"applied_revision,omitempty"`
	Components          []pendingComponentResponse               `json:"components"`
	NewHomes            []architecture.NewComponentHome          `json:"new_component_homes"`
	DetailDiagrams      []architecture.DetailDiagramChange       `json:"detail_diagrams"`
	DiagramTitles       []architecture.DiagramTitleChange        `json:"diagram_titles"`
	HomeMoves           []architecture.ComponentHomeMove         `json:"home_moves"`
	References          []architecture.ReferenceAppearanceChange `json:"references"`
	Valid               bool                                     `json:"valid"`
	CandidateTree       string                                   `json:"candidate_tree,omitempty"`
	Candidate           *agentArchitectureProjection             `json:"candidate"`
	Validation          *agentValidationProjection               `json:"validation"`
	OutOfDate           bool                                     `json:"out_of_date"`
	Review              *agentReviewIdentity                     `json:"review"`
}

type agentReviewIdentity struct {
	BaseRevision  string `json:"base_revision"`
	CandidateTree string `json:"candidate_tree"`
	Generation    uint64 `json:"generation"`
}

type agentUnavailableChangeSet struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Lifecycle string `json:"lifecycle,omitempty"`
	Reason    string `json:"reason"`
}

func (h *Handler) registerAgentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/agent/v1/", h.agentV1Incompatible)
	mux.HandleFunc("GET /api/agent/v2/status", h.agentStatus)
	mux.HandleFunc("GET /api/agent/v2/projects/list", h.agentProjectsList)
	mux.HandleFunc("GET /api/agent/v2/projects/current", h.agentProjectCurrent)
	mux.HandleFunc("POST /api/agent/v2/projects/create", h.agentProjectCreate)
	mux.HandleFunc("POST /api/agent/v2/projects/open", h.agentProjectOpen)
	mux.HandleFunc("POST /api/agent/v2/projects/close", h.agentProjectClose)
	mux.HandleFunc("GET /api/agent/v2/architecture/inspect", h.agentArchitectureInspect)
	mux.HandleFunc("POST /api/agent/v2/architecture/refresh", h.agentArchitectureRefresh)
	mux.HandleFunc("POST /api/agent/v2/architecture/update", h.agentArchitectureUpdate)
	mux.HandleFunc("POST /api/agent/v2/change-sets/list", h.agentChangeSetsList)
	mux.HandleFunc("POST /api/agent/v2/change-sets/create", h.agentChangeSetCreate)
	mux.HandleFunc("POST /api/agent/v2/change-sets/inspect", h.agentChangeSetInspect)
	mux.HandleFunc("POST /api/agent/v2/change-sets/rename", h.agentChangeSetRename)
	mux.HandleFunc("POST /api/agent/v2/change-sets/edit-proposal", h.agentChangeSetEditProposal)
	mux.HandleFunc("POST /api/agent/v2/change-sets/review", h.agentChangeSetReview)
	mux.HandleFunc("POST /api/agent/v2/change-sets/discard", h.agentChangeSetDiscard)
	mux.HandleFunc("POST /api/agent/v2/review-submissions/list", h.agentReviewSubmissionsList)
	mux.HandleFunc("POST /api/agent/v2/review-submissions/inspect", h.agentReviewSubmissionInspect)
	mux.HandleFunc("POST /api/agent/v2/review-submissions/submit", h.agentReviewSubmissionSubmit)
	mux.HandleFunc("POST /api/agent/v2/components/create", h.agentComponentCreate)
	mux.HandleFunc("POST /api/agent/v2/components/edit", h.agentComponentEdit)
	mux.HandleFunc("POST /api/agent/v2/components/move-home", h.agentComponentMoveHome)
	mux.HandleFunc("POST /api/agent/v2/relationships/add", h.agentRelationshipAdd)
	mux.HandleFunc("POST /api/agent/v2/relationships/edit", h.agentRelationshipEdit)
	mux.HandleFunc("POST /api/agent/v2/relationships/remove", h.agentRelationshipRemove)
	mux.HandleFunc("POST /api/agent/v2/diagrams/create-detail", h.agentDiagramCreateDetail)
	mux.HandleFunc("POST /api/agent/v2/diagrams/parent-options", h.agentDetailParentOptions)
	mux.HandleFunc("POST /api/agent/v2/diagrams/reassign-detail", h.agentReassignDetail)
	mux.HandleFunc("POST /api/agent/v2/diagrams/positions", h.agentPositions)
	mux.HandleFunc("POST /api/agent/v2/diagrams/set-position", h.agentSetPosition)
	mux.HandleFunc("POST /api/agent/v2/diagrams/reset-position", h.agentResetPosition)
	mux.HandleFunc("POST /api/agent/v2/diagrams/reset-layout", h.agentResetLayout)
	mux.HandleFunc("POST /api/agent/v2/change-sets/reconcile-preview", h.agentReconciliationPreview)
	mux.HandleFunc("POST /api/agent/v2/change-sets/reconcile-apply", h.agentReconciliationApply)
	mux.HandleFunc("POST /api/agent/v2/diagrams/edit-title", h.agentDiagramEditTitle)
	mux.HandleFunc("POST /api/agent/v2/diagrams/show-component", h.agentDiagramShowComponent)
	mux.HandleFunc("POST /api/agent/v2/diagrams/stop-showing-component", h.agentDiagramStopShowingComponent)
}

func (h *Handler) agentV1Incompatible(response http.ResponseWriter, request *http.Request) {
	if !h.agentRequestAllowed(response, request, false) {
		return
	}
	h.writeAgentError(response, http.StatusConflict, "incompatible_server", "This server requires the WorkBraid agent v2 change-set protocol.", map[string]any{"protocol": agentapi.Protocol})
}

func (h *Handler) agentRequestAllowed(response http.ResponseWriter, request *http.Request, body bool) bool {
	if origin := request.Header.Get("Origin"); origin != "" && origin != h.expectedOrigin {
		h.writeAgentError(response, http.StatusForbidden, "invalid_request", "This local request has an unexpected browser origin.", nil)
		return false
	}
	if body {
		mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			h.writeAgentError(response, http.StatusUnsupportedMediaType, "invalid_request", "Send this request as application/json.", nil)
			return false
		}
	}
	return true
}

func decodeAgentRequest[T any](h *Handler, response http.ResponseWriter, request *http.Request) (T, bool) {
	var zero T
	if !h.agentRequestAllowed(response, request, true) {
		return zero, false
	}
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBody)
	contents, err := io.ReadAll(request.Body)
	if err != nil || !utf8.Valid(contents) {
		h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "Correct the request fields and try again.", nil)
		return zero, false
	}
	if strings.HasSuffix(request.URL.Path, "/set-position") || strings.HasSuffix(request.URL.Path, "/reset-position") || strings.HasSuffix(request.URL.Path, "/reset-layout") {
		if !validPlacementFields(contents, request.URL.Path, false) {
			h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "Correct the position request fields and try again.", nil)
			return zero, false
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var value T
	if err := decoder.Decode(&value); err != nil || ensureJSONEnd(decoder) != nil {
		h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "Correct the request fields and try again.", nil)
		return zero, false
	}
	if _, required := any(value).(interface{ RequiresExactGeneration() }); required {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(contents, &fields); err != nil || fields["generation"] == nil || string(fields["generation"]) == "null" {
			h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "The exact change-set generation is required.", map[string]any{"field": "generation"})
			return zero, false
		}
	}
	if _, position := any(value).(agentapi.DiagramSetPositionRequest); position {
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(contents, &fields)
		for _, key := range []string{"x", "y"} {
			if fields[key] == nil || string(fields[key]) == "null" {
				h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "Both position coordinates are required.", nil)
				return zero, false
			}
		}
	}
	return value, true
}

func (h *Handler) agentContextLocked() agentapi.Context {
	context := agentapi.Context{AuthorityState: "none"}
	if h.loadedSnapshot == nil || h.loadedProject == nil {
		return context
	}
	revision := h.loadedSnapshot.Revision()
	context.Project = &agentapi.ProjectContext{StoreID: h.loadedProject.storeID, Name: h.loadedProject.projectName, Slug: h.loadedProject.projectSlug}
	context.AcceptedRevision = &revision
	context.AuthorityState = "current"
	if h.loadedStale {
		context.AuthorityState = "non_current"
	} else if h.acceptedIndeterminate {
		context.AuthorityState = "indeterminate"
	}
	return context
}

func (h *Handler) writeAgentSuccessLocked(response http.ResponseWriter, status int, result any) {
	writeJSON(response, status, h.agentSuccessEnvelopeLocked(result))
}

func (h *Handler) agentSuccessEnvelopeLocked(result any) agentapi.Envelope {
	return agentapi.Envelope{Protocol: agentapi.Protocol, OK: true, Context: h.agentContextLocked(), Result: result}
}

func (h *Handler) writeAgentSuccess(response http.ResponseWriter, status int, result any) {
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	h.writeAgentSuccessLocked(response, status, result)
}

func (h *Handler) writeAgentErrorLocked(response http.ResponseWriter, status int, code, message string, details map[string]any) {
	writeJSON(response, status, h.agentErrorEnvelopeLocked(code, message, details))
}

func (h *Handler) agentErrorEnvelopeLocked(code, message string, details map[string]any) agentapi.Envelope {
	if details == nil {
		details = map[string]any{}
	}
	return agentapi.Envelope{
		Protocol: agentapi.Protocol, OK: false, Context: h.agentContextLocked(),
		Error: &agentapi.Error{Code: code, Message: message, Details: details},
	}
}

func (h *Handler) writeAgentError(response http.ResponseWriter, status int, code, message string, details map[string]any) {
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	h.writeAgentErrorLocked(response, status, code, message, details)
}

func agentMessage(code string) string {
	switch code {
	case "change_set_state_mismatch":
		return "The proposal changed. Inspect it before preparing again."
	case "project_not_found":
		return "That project is not in the WorkBraid catalog. List projects or create it deliberately."
	case "project_conflict":
		return "More than one project claims that slug. Resolve the catalog conflict before opening it."
	case "project_unavailable":
		return "That project cannot be loaded as valid Architecture."
	case "project_not_open":
		return "No project is currently open. Open or create one first."
	case "project_mismatch":
		return "Another project is open. Inspect the current project before continuing."
	case "architecture_non_current":
		return "The loaded Architecture is not current. Refresh before editing."
	case "refresh_failed":
		return "WorkBraid could not determine current accepted Architecture. Retry Refresh explicitly."
	case "change_set_not_found":
		return "That change set was not found. List change sets again."
	case "change_set_unavailable":
		return "That change set cannot be loaded. Other change sets and Accepted remain available."
	case "change_set_name_conflict":
		return "Another active change set already uses that name. Choose another name."
	case "change_set_generation_mismatch":
		return "That change set changed. Inspect it again before editing."
	case "change_set_not_editable":
		return "Applied change sets are read-only. Choose an active change set."
	case "change_set_out_of_date":
		return "This change set is out of date with Accepted. It can still be edited and reviewed. Reconcile with current Accepted before updating Architecture."
	case "target_not_found":
		return "That Component or Diagram is not in the current Architecture, accepted or pending."
	case "target_not_eligible":
		return "That Component or Diagram is not an allowed target for this change."
	case "validation_blocked":
		return "Changes need correction before they can be reviewed."
	case "review_required":
		return "Review the complete current changes before updating Architecture."
	case "review_invalidated":
		return "This Review is no longer valid. Inspect changes, then Review again."
	case "review_submission_not_found":
		return "That submitted review was not found. List reviews for the Change Set again."
	case "review_submission_unavailable":
		return "That submitted review could not be loaded exactly. Do not guess or repair it."
	case "review_anchor_invalid":
		return "A review comment does not point to that exact reviewed source. Correct its anchor and submit again."
	case "review_submission_not_allowed":
		return "New feedback requires an active proposal with an exact prepared Review."
	case "accepted_conflict":
		return "Accepted Architecture changed before update. Refresh and inspect the preserved pending work."
	case "acceptance_uncertain":
		return "WorkBraid could not determine whether acceptance succeeded. Inspect or Refresh before retrying."
	case "accepted_reload_required":
		return "Architecture was accepted, but this process could not reload it. Do not accept again; Refresh or reopen."
	case "unsupported_action":
		return "That action is outside the current Architecture product."
	case "operation_failed":
		return "The operation failed. Inspect current state before retrying."
	default:
		return "Correct the request and try again."
	}
}

func (h *Handler) agentStatus(response http.ResponseWriter, request *http.Request) {
	if !h.agentRequestAllowed(response, request, false) {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	h.writeAgentSuccessLocked(response, http.StatusOK, map[string]any{"protocol": agentapi.Protocol})
}

func (h *Handler) agentProjectsList(response http.ResponseWriter, request *http.Request) {
	if !h.agentRequestAllowed(response, request, false) {
		return
	}
	projects, err := h.architecture.Catalog(request.Context())
	if err != nil {
		h.writeAgentError(response, http.StatusInternalServerError, "operation_failed", agentMessage("operation_failed"), nil)
		return
	}
	values := make([]catalogProjectResponse, len(projects))
	for index, project := range projects {
		values[index] = catalogProjectResponse{Name: project.Name, Slug: project.Slug, Revision: project.Revision, StoreID: project.StoreID, Unavailable: project.Unavailable, Conflict: project.Conflict}
	}
	h.writeAgentSuccess(response, http.StatusOK, map[string]any{"projects": values})
}

func (h *Handler) agentProjectCurrent(response http.ResponseWriter, request *http.Request) {
	if !h.agentRequestAllowed(response, request, false) {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	var project *agentapi.ProjectContext
	if h.loadedProject != nil {
		project = &agentapi.ProjectContext{StoreID: h.loadedProject.storeID, Name: h.loadedProject.projectName, Slug: h.loadedProject.projectSlug}
	}
	h.writeAgentSuccessLocked(response, http.StatusOK, map[string]any{"project": project})
}

func (h *Handler) agentArchitectureInspect(response http.ResponseWriter, request *http.Request) {
	if !h.agentRequestAllowed(response, request, false) {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if h.loadedSnapshot == nil || h.loadedProject == nil {
		h.writeAgentErrorLocked(response, http.StatusConflict, "project_not_open", agentMessage("project_not_open"), nil)
		return
	}
	projection := h.agentArchitectureProjectionLocked(*h.loadedSnapshot)
	h.writeAgentSuccessLocked(response, http.StatusOK, projection)
}

func (h *Handler) agentChangeSetsList(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ChangeSetsListRequest](h, response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if h.loadedProject == nil || payload.StoreID != h.loadedProject.storeID {
		h.writeAgentErrorLocked(response, http.StatusConflict, "project_mismatch", agentMessage("project_mismatch"), nil)
		return
	}
	values := make([]agentChangeSetProjection, 0, len(h.changeSets))
	for _, record := range h.changeSets {
		values = append(values, h.agentChangeSetProjectionLocked(record))
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Lifecycle != values[j].Lifecycle {
			return values[i].Lifecycle < values[j].Lifecycle
		}
		if values[i].Name != values[j].Name {
			return values[i].Name < values[j].Name
		}
		return values[i].ID < values[j].ID
	})
	unavailable := make([]agentUnavailableChangeSet, len(h.unavailableChangeSets))
	for index, record := range h.unavailableChangeSets {
		unavailable[index] = agentUnavailableChangeSet{ID: record.ID, Name: record.Name, Lifecycle: record.Lifecycle, Reason: record.Reason}
	}
	h.writeAgentSuccessLocked(response, http.StatusOK, map[string]any{"change_sets": values, "unavailable": unavailable})
}

func (h *Handler) agentChangeSetInspect(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ChangeSetInspectRequest](h, response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if h.loadedProject == nil || payload.StoreID != h.loadedProject.storeID {
		h.writeAgentErrorLocked(response, http.StatusConflict, "project_mismatch", agentMessage("project_mismatch"), nil)
		return
	}
	if record := h.changeSets[payload.ChangeSetID]; record != nil {
		h.writeAgentSuccessLocked(response, http.StatusOK, h.agentChangeSetProjectionLocked(record))
		return
	}
	for _, record := range h.unavailableChangeSets {
		if record.ID == payload.ChangeSetID {
			h.writeAgentErrorLocked(response, http.StatusConflict, "change_set_unavailable", agentMessage("change_set_unavailable"), map[string]any{"change_set_id": payload.ChangeSetID, "reason": record.Reason})
			return
		}
	}
	h.writeAgentErrorLocked(response, http.StatusNotFound, "change_set_not_found", agentMessage("change_set_not_found"), map[string]any{"change_set_id": payload.ChangeSetID})
}

func (h *Handler) agentArchitectureProjectionLocked(snapshot architecture.Snapshot) agentArchitectureProjection {
	projected := projectSnapshot(snapshot, "")
	components := make([]agentComponentProjection, len(projected.Components))
	for index, component := range projected.Components {
		home, _, _ := snapshot.ComponentHome(component.ID)
		var references []string
		for _, diagram := range projected.Diagrams {
			for _, appearance := range diagram.Appearances {
				if appearance.ComponentID == component.ID && appearance.Role == "reference" {
					references = append(references, diagram.ID)
				}
			}
		}
		components[index] = agentComponentProjection{componentResponse: component, HomeDiagramID: home, ReferenceDiagramIDs: references}
	}
	children := make(map[string][]string, len(projected.Diagrams))
	for _, diagram := range projected.Diagrams {
		if diagram.ParentDiagramID != "" {
			children[diagram.ParentDiagramID] = append(children[diagram.ParentDiagramID], diagram.ID)
		}
	}
	diagrams := make([]agentDiagramProjection, len(projected.Diagrams))
	for index, diagram := range projected.Diagrams {
		diagrams[index] = agentDiagramProjection{diagramResponse: diagram, ChildDiagramIDs: children[diagram.ID]}
	}
	return agentArchitectureProjection{
		ArchitectureVersion: snapshot.FormatVersion(),
		Project:             agentapi.ProjectContext{StoreID: snapshot.StoreID(), Name: snapshot.ProjectName(), Slug: snapshot.ProjectSlug()},
		Revision:            snapshot.Revision(), RootDiagramID: snapshot.RootDiagramID(), Components: components, Diagrams: diagrams,
	}
}

func (h *Handler) agentChangeSetProjectionLocked(pending *pendingChangeSet) agentChangeSetProjection {
	components := make([]pendingComponentResponse, len(pending.changes))
	for index, change := range pending.changes {
		relationships := make([]relationshipResponse, len(change.Relationships))
		for relationshipIndex, relationship := range change.Relationships {
			relationships[relationshipIndex] = relationshipResponse{TargetID: relationship.TargetID, Label: relationship.Label}
		}
		components[index] = pendingComponentResponse{
			ID: change.ID, Title: change.Title, Description: change.Description, Path: change.Path, New: change.New,
			TitleChanged: change.TitleChanged, DescriptionChanged: change.DescriptionChanged,
			Relationships: relationships, RelationshipsChanged: change.RelationshipsChanged,
		}
	}
	value := agentChangeSetProjection{
		ArchitectureVersion: max(pending.architectureVersion, pending.baseSnapshot.FormatVersion()),
		NodePositions:       append([]architecture.NodePositionChange{}, pending.nodePositions...),
		StateObject:         pending.refObject,
		DetailReassignments: append([]architecture.DetailReassignment{}, pending.detailReassignments...),
		ID:                  pending.id, Name: pending.name, Lifecycle: pending.lifecycle, Proposal: pending.proposal, AppliedRevision: pending.appliedRevision,
		BaseRevision: pending.baseRevision, Generation: pending.generation, Components: components,
		NewHomes:       append([]architecture.NewComponentHome{}, pending.newComponentHomes...),
		DetailDiagrams: append([]architecture.DetailDiagramChange{}, pending.detailDiagrams...),
		DiagramTitles:  append([]architecture.DiagramTitleChange{}, pending.diagramTitles...),
		HomeMoves:      append([]architecture.ComponentHomeMove{}, pending.homeMoves...),
		References:     append([]architecture.ReferenceAppearanceChange{}, pending.references...), OutOfDate: pending.lifecycle == "active" && h.loadedSnapshot != nil && pending.baseRevision != h.loadedSnapshot.Revision(),
	}
	if pending.candidate != nil {
		candidate := h.agentArchitectureProjectionLocked(pending.candidate.Snapshot())
		value.Valid = true
		value.CandidateTree = pending.candidate.Tree()
		value.Candidate = &candidate
	}
	if pending.validationCode != "" {
		value.Validation = &agentValidationProjection{
			Code: pending.validationCode, ComponentID: pending.validationItem,
			RelationshipPosition: pending.validationRelationshipPosition, RelationshipField: pending.validationRelationshipField,
			DiagramID: pending.validationDiagram, DiagramField: pending.validationDiagramField,
		}
	}
	if pending.review != nil && pending.review.generation == pending.generation && pending.candidate != nil && pending.review.candidateTree == pending.candidate.Tree() {
		value.Review = &agentReviewIdentity{BaseRevision: pending.review.baseRevision, CandidateTree: pending.review.candidateTree, Generation: pending.review.generation}
	}
	return value
}

func (h *Handler) checkAgentStateLocked(expected agentapi.StatePreconditions) (architecture.Snapshot, *pendingChangeSet, *agentapi.Error) {
	if h.loadedSnapshot == nil || h.loadedProject == nil {
		return architecture.Snapshot{}, nil, &agentapi.Error{Code: "project_not_open", Message: agentMessage("project_not_open"), Details: map[string]any{}}
	}
	if expected.StoreID == "" || expected.StoreID != h.loadedProject.storeID {
		return architecture.Snapshot{}, nil, &agentapi.Error{Code: "project_mismatch", Message: agentMessage("project_mismatch"), Details: map[string]any{"expected_store_id": expected.StoreID, "current_store_id": h.loadedProject.storeID}}
	}
	snapshot := *h.loadedSnapshot
	if h.loadedStale {
		return architecture.Snapshot{}, nil, &agentapi.Error{Code: "architecture_non_current", Message: agentMessage("architecture_non_current"), Details: map[string]any{"loaded_revision": snapshot.Revision()}}
	}
	if h.acceptedIndeterminate {
		return architecture.Snapshot{}, nil, &agentapi.Error{Code: "refresh_failed", Message: agentMessage("refresh_failed"), Details: map[string]any{"loaded_revision": snapshot.Revision()}}
	}
	record, lifecycleErr := h.editableAgentChangeSetLocked(expected.ChangeSetID)
	if lifecycleErr != nil {
		return architecture.Snapshot{}, nil, lifecycleErr
	}
	if record.generation != expected.Generation {
		return architecture.Snapshot{}, nil, &agentapi.Error{Code: "change_set_generation_mismatch", Message: agentMessage("change_set_generation_mismatch"), Details: map[string]any{"change_set_id": expected.ChangeSetID, "expected_generation": expected.Generation, "current_generation": record.generation}}
	}
	return record.baseSnapshot, clonePending(record), nil
}

func (h *Handler) editableAgentChangeSetLocked(id string) (*pendingChangeSet, *agentapi.Error) {
	if record := h.changeSets[id]; record != nil {
		if record.lifecycle != "active" {
			return record, &agentapi.Error{Code: "change_set_not_editable", Message: agentMessage("change_set_not_editable"), Details: map[string]any{"change_set_id": id, "lifecycle": record.lifecycle}}
		}
		return record, nil
	}
	for _, unavailable := range h.unavailableChangeSets {
		if unavailable.ID == id {
			return nil, &agentapi.Error{Code: "change_set_unavailable", Message: agentMessage("change_set_unavailable"), Details: map[string]any{"change_set_id": id, "reason": unavailable.Reason}}
		}
	}
	return nil, &agentapi.Error{Code: "change_set_not_found", Message: agentMessage("change_set_not_found"), Details: map[string]any{"change_set_id": id}}
}

func (h *Handler) writeAgentDomainErrorLocked(response http.ResponseWriter, status int, err *agentapi.Error) {
	h.writeAgentErrorLocked(response, status, err.Code, err.Message, err.Details)
}

func agentDomainErrorStatus(err *agentapi.Error) int {
	switch err.Code {
	case "invalid_request":
		return http.StatusBadRequest
	case "target_not_found", "change_set_not_found":
		return http.StatusNotFound
	default:
		return http.StatusConflict
	}
}

func agentMutationResult(pending *pendingChangeSet, values map[string]any) map[string]any {
	if values == nil {
		values = map[string]any{}
	}
	values["change_set_id"] = pending.id
	values["generation"] = pending.generation
	values["candidate_valid"] = pending.candidate != nil
	if pending.validationCode != "" {
		values["validation_code"] = pending.validationCode
	}
	return values
}

func requireNonEmpty(values ...string) bool {
	for _, value := range values {
		if value == "" {
			return false
		}
	}
	return true
}

func exactOccurrence(values []architecture.AuthoringRelationship, target, label string, occurrence int) int {
	if occurrence < 1 {
		return -1
	}
	seen := 0
	for index, value := range values {
		if value.TargetID == target && value.Label == label {
			seen++
			if seen == occurrence {
				return index
			}
		}
	}
	return -1
}

func (h *Handler) agentChangeSetCreate(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ChangeSetCreateRequest](h, response, request)
	if !ok {
		return
	}
	if payload.Name != nil && (strings.TrimSpace(*payload.Name) == "" || strings.ContainsAny(*payload.Name, "\r\n")) {
		h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "A supplied change-set name must be non-empty and fit on one line.", map[string]any{"field": "name"})
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if h.loadedSnapshot == nil || h.loadedProject == nil || payload.StoreID != h.loadedProject.storeID {
		h.writeAgentErrorLocked(response, http.StatusConflict, "project_mismatch", agentMessage("project_mismatch"), nil)
		return
	}
	if h.loadedStale {
		h.writeAgentErrorLocked(response, http.StatusConflict, "architecture_non_current", agentMessage("architecture_non_current"), map[string]any{"expected_revision": payload.AcceptedRevision, "loaded_revision": h.loadedSnapshot.Revision()})
		return
	}
	if h.acceptedIndeterminate {
		h.writeAgentErrorLocked(response, http.StatusServiceUnavailable, "refresh_failed", agentMessage("refresh_failed"), map[string]any{"loaded_revision": h.loadedSnapshot.Revision()})
		return
	}
	if payload.AcceptedRevision != h.loadedSnapshot.Revision() {
		h.writeAgentErrorLocked(response, http.StatusConflict, "architecture_non_current", agentMessage("architecture_non_current"), map[string]any{"expected_revision": payload.AcceptedRevision, "loaded_revision": h.loadedSnapshot.Revision()})
		return
	}
	name := ""
	if payload.Name != nil {
		name = *payload.Name
	}
	id, selectedName, err := h.architecture.NewChangeSet(h.existingDurableChangeSetsLocked(), name)
	if err != nil {
		h.writeAgentErrorLocked(response, http.StatusConflict, "change_set_name_conflict", agentMessage("change_set_name_conflict"), nil)
		return
	}
	base := *h.loadedSnapshot
	candidate, err := h.architecture.ConstructCandidate(request.Context(), base, nil, architecture.CandidateComposition{})
	if err != nil {
		h.writeAgentErrorLocked(response, http.StatusInternalServerError, "operation_failed", agentMessage("operation_failed"), nil)
		return
	}
	record := &pendingChangeSet{id: id, name: selectedName, lifecycle: "active", storeID: base.StoreID(), baseRevision: base.Revision(), baseSnapshot: base, candidate: &candidate}
	if !h.persistActiveLocked(request.Context(), record, "") {
		h.writeAgentErrorLocked(response, http.StatusConflict, "operation_failed", agentMessage("operation_failed"), nil)
		return
	}
	h.writeAgentSuccessLocked(response, http.StatusCreated, h.agentChangeSetProjectionLocked(record))
}

func (h *Handler) agentChangeSetRename(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ChangeSetRenameRequest](h, response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	_, current, stateErr := h.checkAgentStateLocked(payload.StatePreconditions)
	if stateErr != nil {
		h.writeAgentDomainErrorLocked(response, agentDomainErrorStatus(stateErr), stateErr)
		return
	}
	name := strings.TrimSpace(payload.Name)
	if architecture.ValidateChangeSetName(name) != nil {
		h.writeAgentErrorLocked(response, http.StatusBadRequest, "invalid_request", "The change-set name must be non-empty and fit on one line.", nil)
		return
	}
	for id, record := range h.changeSets {
		if id != current.id && record.lifecycle == "active" && strings.EqualFold(record.name, name) {
			h.writeAgentErrorLocked(response, http.StatusConflict, "change_set_name_conflict", agentMessage("change_set_name_conflict"), nil)
			return
		}
	}
	for _, record := range h.unavailableChangeSets {
		if record.Lifecycle == "active" && record.Name != "" && strings.EqualFold(record.Name, name) {
			h.writeAgentErrorLocked(response, http.StatusConflict, "change_set_name_conflict", agentMessage("change_set_name_conflict"), nil)
			return
		}
	}
	if current.name == name {
		h.writeAgentSuccessLocked(response, http.StatusOK, h.agentChangeSetProjectionLocked(h.changeSets[current.id]))
		return
	}
	current.name, current.review = name, nil
	current.generation++
	if !h.persistAgentMutationLocked(request.Context(), payload.ChangeSetID, current) {
		h.writeAgentErrorLocked(response, http.StatusConflict, "change_set_generation_mismatch", agentMessage("change_set_generation_mismatch"), nil)
		return
	}
	h.writeAgentSuccessLocked(response, http.StatusOK, h.agentChangeSetProjectionLocked(current))
}

func (h *Handler) agentChangeSetEditProposal(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ChangeSetEditProposalRequest](h, response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	_, current, stateErr := h.checkAgentStateLocked(payload.StatePreconditions)
	if stateErr != nil {
		h.writeAgentDomainErrorLocked(response, agentDomainErrorStatus(stateErr), stateErr)
		return
	}
	if current.proposal == payload.ProposalMarkdown {
		h.writeAgentSuccessLocked(response, http.StatusOK, h.agentChangeSetProjectionLocked(h.changeSets[current.id]))
		return
	}
	current.proposal, current.review = payload.ProposalMarkdown, nil
	current.generation++
	if !h.persistAgentMutationLocked(request.Context(), payload.ChangeSetID, current) {
		h.writeAgentErrorLocked(response, http.StatusConflict, "change_set_generation_mismatch", agentMessage("change_set_generation_mismatch"), nil)
		return
	}
	h.writeAgentSuccessLocked(response, http.StatusOK, h.agentChangeSetProjectionLocked(current))
}

func (h *Handler) persistAgentMutationLocked(ctx context.Context, currentID string, proposed *pendingChangeSet) bool {
	current := h.changeSetLocked(currentID)
	if current == nil {
		return false
	}
	return h.persistActiveLocked(ctx, proposed, current.refObject)
}

func (h *Handler) agentProjectCreate(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ProjectCreateRequest](h, response, request)
	if !ok {
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "Project name is required.", map[string]any{"field": "name"})
		return
	}
	h.stateMutex.Lock()
	result, code := h.createProjectLocked(request.Context(), payload.Name)
	h.runBeforeAgentResultCaptureLocked()
	status := http.StatusCreated
	var envelope agentapi.Envelope
	if code != "" {
		status = http.StatusInternalServerError
		envelope = h.mappedAgentErrorEnvelopeLocked(code)
	} else {
		envelope = h.agentSuccessEnvelopeLocked(map[string]any{
			"project":           map[string]any{"store_id": result.StoreID, "name": result.ProjectName, "slug": result.ProjectSlug},
			"accepted_revision": result.Revision, "root_diagram_id": result.RootDiagramID,
		})
	}
	h.stateMutex.Unlock()
	writeJSON(response, status, envelope)
}

func (h *Handler) agentProjectOpen(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ProjectOpenRequest](h, response, request)
	if !ok {
		return
	}
	if payload.Slug == "" {
		h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "Project slug is required.", map[string]any{"field": "slug"})
		return
	}
	h.stateMutex.Lock()
	result, code := h.openProjectLocked(request.Context(), payload.Slug)
	h.runBeforeAgentResultCaptureLocked()
	status := http.StatusOK
	var envelope agentapi.Envelope
	if code != "" {
		status = http.StatusConflict
		if code == errorProjectNotFound {
			status = http.StatusNotFound
		}
		envelope = h.mappedAgentErrorEnvelopeLocked(code)
	} else {
		envelope = h.agentSuccessEnvelopeLocked(map[string]any{
			"project":           map[string]any{"store_id": result.StoreID, "name": result.ProjectName, "slug": result.ProjectSlug},
			"accepted_revision": result.Revision, "root_diagram_id": result.RootDiagramID,
		})
	}
	h.stateMutex.Unlock()
	writeJSON(response, status, envelope)
}

func (h *Handler) agentProjectClose(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ProjectCloseRequest](h, response, request)
	if !ok {
		return
	}
	if payload.StoreID == "" {
		h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "Store ID is required.", map[string]any{"field": "store_id"})
		return
	}
	h.stateMutex.Lock()
	slug := ""
	if h.loadedProject != nil {
		slug = h.loadedProject.projectSlug
	}
	_, code := h.leaveProjectLocked(slug, payload.StoreID)
	h.runBeforeAgentResultCaptureLocked()
	status := http.StatusOK
	var envelope agentapi.Envelope
	if code != "" {
		status = http.StatusConflict
		envelope = h.mappedAgentErrorEnvelopeLocked(code)
	} else {
		envelope = h.agentSuccessEnvelopeLocked(map[string]any{"closed_store_id": payload.StoreID})
	}
	h.stateMutex.Unlock()
	writeJSON(response, status, envelope)
}

func (h *Handler) mappedAgentErrorLocked(code string) *agentapi.Error {
	mapped := "operation_failed"
	details := map[string]any{}
	switch code {
	case errorProjectNotFound:
		mapped = "project_not_found"
	case errorCatalogConflict:
		mapped = "project_conflict"
	case errorCatalogUnavailable, errorArchitectureUnavailable, errorArchitectureInvalid, errorArchitectureUnsupported:
		mapped = "project_unavailable"
	case errorArchitectureNotOpen:
		if h.loadedProject == nil {
			mapped = "project_not_open"
		} else {
			mapped = "project_mismatch"
		}
	case errorChangesUnavailable:
		mapped = "unsupported_action"
	case errorChangesElsewhere:
		mapped = "change_set_generation_mismatch"
	case errorArchitectureStale, errorRefreshChanged, errorRefreshUnavailable, errorRefreshInvalid, errorRefreshUnsupported:
		mapped = "architecture_non_current"
		if code != errorArchitectureStale {
			details["classification"] = "known_non_current"
		}
	case errorRefreshFailed:
		mapped = "refresh_failed"
		details["classification"] = "indeterminate"
	case errorReviewFailed:
		mapped = "validation_blocked"
	case errorReviewChanged:
		mapped = "review_invalidated"
	case errorUpdateUncertain:
		mapped = "acceptance_uncertain"
	case errorUpdatedReload:
		mapped = "accepted_reload_required"
	case errorHomeMoveUnavailable, errorComponentNotFound:
		mapped = "target_not_found"
	case errorChangeFailed:
		mapped = "target_not_eligible"
	}
	return &agentapi.Error{Code: mapped, Message: agentMessage(mapped), Details: details}
}

func (h *Handler) mappedAgentErrorEnvelopeLocked(code string) agentapi.Envelope {
	mapped := h.mappedAgentErrorLocked(code)
	return h.agentErrorEnvelopeLocked(mapped.Code, mapped.Message, mapped.Details)
}

func (h *Handler) agentArchitectureRefresh(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ArchitectureRefreshRequest](h, response, request)
	if !ok {
		return
	}
	if !requireNonEmpty(payload.StoreID, payload.AcceptedRevision) {
		h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "Store ID and accepted revision are required.", nil)
		return
	}
	h.stateMutex.Lock()
	slug := ""
	if h.loadedProject != nil {
		slug = h.loadedProject.projectSlug
	}
	result, code, status := h.refreshArchitectureLocked(request.Context(), architectureActionRequest{
		ProjectSlug: slug, StoreID: payload.StoreID, ExpectedRevision: payload.AcceptedRevision,
	})
	h.runBeforeAgentResultCaptureLocked()
	var envelope agentapi.Envelope
	if code != "" {
		envelope = h.mappedAgentErrorEnvelopeLocked(code)
	} else {
		classification := "unchanged"
		if result.Revision != payload.AcceptedRevision {
			classification = "adopted"
		}
		envelope = h.agentSuccessEnvelopeLocked(map[string]any{"classification": classification, "accepted_revision": result.Revision, "slug": result.ProjectSlug})
	}
	h.stateMutex.Unlock()
	writeJSON(response, status, envelope)
}

func (h *Handler) agentChangeSetReview(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ChangeSetReviewRequest](h, response, request)
	if !ok {
		return
	}
	generation := payload.Generation
	h.stateMutex.Lock()
	if _, _, stateErr := h.checkAgentStateLocked(payload.StatePreconditions); stateErr != nil {
		status := agentDomainErrorStatus(stateErr)
		envelope := h.agentErrorEnvelopeLocked(stateErr.Code, stateErr.Message, stateErr.Details)
		h.stateMutex.Unlock()
		writeJSON(response, status, envelope)
		return
	}
	slug := ""
	if h.loadedProject != nil {
		slug = h.loadedProject.projectSlug
	}
	result, code, status := h.reviewChangesLocked(request.Context(), architectureActionRequest{
		ProjectSlug: slug, StoreID: payload.StoreID, ChangeSetID: payload.ChangeSetID, ExpectedGeneration: &generation, PendingGenerationObserved: true,
	})
	h.runBeforeAgentResultCaptureLocked()
	var envelope agentapi.Envelope
	if code != "" || result.Changes == nil || result.Changes.Review == nil {
		if result.Changes != nil && result.Changes.ValidationCode != "" && result.Changes.ValidationCode != "change_unavailable" {
			envelope = h.agentErrorEnvelopeLocked("validation_blocked", agentMessage("validation_blocked"), map[string]any{
				"code": result.Changes.ValidationCode, "component_id": result.Changes.ValidationItem,
				"relationship_position": result.Changes.ValidationRelationshipPosition, "relationship_field": result.Changes.ValidationRelationshipField,
				"diagram_id": result.Changes.ValidationDiagram, "diagram_field": result.Changes.ValidationDiagramField,
			})
			status = http.StatusUnprocessableEntity
		} else if status == http.StatusInternalServerError {
			envelope = h.agentErrorEnvelopeLocked("operation_failed", agentMessage("operation_failed"), nil)
		} else if code == errorReviewFailed {
			envelope = h.agentErrorEnvelopeLocked("review_required", agentMessage("review_required"), nil)
			status = http.StatusConflict
		} else {
			envelope = h.mappedAgentErrorEnvelopeLocked(code)
		}
	} else {
		envelope = h.agentSuccessEnvelopeLocked(result.Changes.Review)
	}
	h.stateMutex.Unlock()
	writeJSON(response, status, envelope)
}

func (h *Handler) agentChangeSetDiscard(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ChangeSetDiscardRequest](h, response, request)
	if !ok {
		return
	}
	generation := payload.Generation
	h.stateMutex.Lock()
	if _, _, stateErr := h.checkAgentStateLocked(payload.StatePreconditions); stateErr != nil {
		status := agentDomainErrorStatus(stateErr)
		envelope := h.agentErrorEnvelopeLocked(stateErr.Code, stateErr.Message, stateErr.Details)
		h.stateMutex.Unlock()
		writeJSON(response, status, envelope)
		return
	}
	slug := ""
	if h.loadedProject != nil {
		slug = h.loadedProject.projectSlug
	}
	_, code := h.discardChangesLocked(request.Context(), architectureActionRequest{
		ProjectSlug: slug, StoreID: payload.StoreID, ChangeSetID: payload.ChangeSetID, ExpectedGeneration: &generation, PendingGenerationObserved: true,
	})
	h.runBeforeAgentResultCaptureLocked()
	status := http.StatusOK
	var envelope agentapi.Envelope
	if code != "" {
		status = http.StatusConflict
		envelope = h.mappedAgentErrorEnvelopeLocked(code)
	} else {
		envelope = h.agentSuccessEnvelopeLocked(map[string]any{"change_set_id": payload.ChangeSetID, "discarded_generation": payload.Generation})
	}
	h.stateMutex.Unlock()
	writeJSON(response, status, envelope)
}

func (h *Handler) agentArchitectureUpdate(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ArchitectureUpdateRequest](h, response, request)
	if !ok {
		return
	}
	if !requireNonEmpty(payload.StoreID, payload.ChangeSetID, payload.BaseRevision, payload.CandidateTree) {
		h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "Store ID and exact review binding are required.", nil)
		return
	}
	h.stateMutex.Lock()
	slug := ""
	if h.loadedProject != nil {
		slug = h.loadedProject.projectSlug
	}
	if h.loadedSnapshot != nil && h.loadedProject != nil && payload.StoreID == h.loadedProject.storeID && !h.loadedStale {
		record, lifecycleErr := h.editableAgentChangeSetLocked(payload.ChangeSetID)
		if lifecycleErr != nil && !(record != nil && record.lifecycle == "applied" && appliedReceiptMatches(record, acceptChangesRequest{
			ChangeSetID: payload.ChangeSetID, BaseRevision: payload.BaseRevision, CandidateTree: payload.CandidateTree, Generation: payload.Generation,
		})) {
			status := agentDomainErrorStatus(lifecycleErr)
			envelope := h.agentErrorEnvelopeLocked(lifecycleErr.Code, lifecycleErr.Message, lifecycleErr.Details)
			h.stateMutex.Unlock()
			writeJSON(response, status, envelope)
			return
		}
	}
	wasOutOfDate := false
	if record := h.changeSets[payload.ChangeSetID]; record != nil && h.loadedSnapshot != nil {
		wasOutOfDate = record.lifecycle == "active" && record.baseRevision != h.loadedSnapshot.Revision()
	}
	result, code, status, casConflict := h.acceptChangesLocked(acceptChangesRequest{
		ProjectSlug: slug, StoreID: payload.StoreID, ChangeSetID: payload.ChangeSetID, BaseRevision: payload.BaseRevision, CandidateTree: payload.CandidateTree, Generation: payload.Generation,
	})
	h.runBeforeAgentResultCaptureLocked()
	var envelope agentapi.Envelope
	if code != "" {
		if code == errorArchitectureStale && casConflict {
			classification := "accepted_conflict"
			if wasOutOfDate {
				classification = "change_set_out_of_date"
			}
			envelope = h.agentErrorEnvelopeLocked(classification, agentMessage(classification), map[string]any{"change_set_id": payload.ChangeSetID})
		} else if code == errorReviewFailed {
			envelope = h.agentErrorEnvelopeLocked("review_required", agentMessage("review_required"), nil)
		} else {
			envelope = h.mappedAgentErrorEnvelopeLocked(code)
		}
	} else {
		publication := "published"
		if result.AlreadyApplied {
			publication = "already_applied"
		}
		envelope = h.agentSuccessEnvelopeLocked(map[string]any{
			"change_set_id": payload.ChangeSetID, "base_revision": payload.BaseRevision, "candidate_tree": payload.CandidateTree, "generation": payload.Generation,
			"accepted_revision": result.Revision, "publication": publication, "parent_diff": result.ParentDiff,
		})
	}
	h.stateMutex.Unlock()
	writeJSON(response, status, envelope)
}

func (h *Handler) runBeforeAgentResultCaptureLocked() {
	if h.beforeAgentResultCapture != nil {
		h.beforeAgentResultCapture()
	}
}

func (h *Handler) agentComponentCreate(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ComponentCreateRequest](h, response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	snapshot, pending, stateErr := h.checkAgentStateLocked(payload.StatePreconditions)
	if stateErr != nil {
		h.writeAgentDomainErrorLocked(response, http.StatusConflict, stateErr)
		return
	}
	if snapshot.FormatVersion() < 2 {
		h.writeAgentErrorLocked(response, http.StatusConflict, "unsupported_action", agentMessage("unsupported_action"), nil)
		return
	}
	homeDiagramID := snapshot.RootDiagramID()
	if payload.DiagramID != nil {
		homeDiagramID = *payload.DiagramID
	}
	if homeDiagramID == "" || !pendingHasDiagram(snapshot, pending, homeDiagramID) {
		h.writeAgentErrorLocked(response, http.StatusNotFound, "target_not_found", agentMessage("target_not_found"), map[string]any{"diagram_id": homeDiagramID})
		return
	}
	pending = h.ensurePendingLocked(snapshot, pending)
	change := h.architecture.NewComponentChange(snapshot, pending.changes, strings.TrimSpace(payload.Title), normalizeAuthoredDescription(payload.Description))
	change.Relationships = []architecture.AuthoringRelationship{}
	change.RelationshipsChanged = true
	pending.newComponentHomes = append(pending.newComponentHomes, architecture.NewComponentHome{ComponentID: change.ID, DiagramID: homeDiagramID})
	pending = h.keepComponentChangeLocked(request.Context(), snapshot, pending, change, -1)
	if pendingOperationFailed(pending) {
		h.writeAgentErrorLocked(response, http.StatusInternalServerError, "operation_failed", agentMessage("operation_failed"), nil)
		return
	}
	if !h.persistAgentMutationLocked(request.Context(), payload.ChangeSetID, pending) {
		h.writeAgentErrorLocked(response, http.StatusConflict, "change_set_generation_mismatch", agentMessage("change_set_generation_mismatch"), nil)
		return
	}
	h.writeAgentSuccessLocked(response, http.StatusOK, agentMutationResult(pending, map[string]any{"component_id": change.ID, "home_diagram_id": homeDiagramID}))
}

func (h *Handler) agentComponentEdit(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ComponentEditRequest](h, response, request)
	if !ok {
		return
	}
	if payload.ComponentID == "" || (payload.Title == nil && payload.Description == nil) {
		h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "Component ID and at least one changed field are required.", nil)
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	snapshot, pending, stateErr := h.checkAgentStateLocked(payload.StatePreconditions)
	if stateErr != nil {
		h.writeAgentDomainErrorLocked(response, http.StatusConflict, stateErr)
		return
	}
	change, index, found := componentChangeLocked(snapshot, pending, payload.ComponentID)
	if !found {
		h.writeAgentErrorLocked(response, http.StatusNotFound, "target_not_found", agentMessage("target_not_found"), map[string]any{"component_id": payload.ComponentID})
		return
	}
	changedFields := make([]string, 0, 2)
	if payload.Title != nil && change.Title != strings.TrimSpace(*payload.Title) {
		change.Title = strings.TrimSpace(*payload.Title)
		change.TitleChanged = true
		changedFields = append(changedFields, "title")
	}
	if payload.Description != nil && change.Description != normalizeAuthoredDescription(*payload.Description) {
		change.Description = normalizeAuthoredDescription(*payload.Description)
		change.DescriptionChanged = true
		changedFields = append(changedFields, "description")
	}
	if len(changedFields) == 0 {
		h.writeAgentSuccessLocked(response, http.StatusOK, agentMutationResult(pending, map[string]any{"component_id": payload.ComponentID, "changed_fields": changedFields, "unchanged": true}))
		return
	}
	pending = h.keepComponentChangeLocked(request.Context(), snapshot, pending, change, index)
	if pendingOperationFailed(pending) {
		h.writeAgentErrorLocked(response, http.StatusInternalServerError, "operation_failed", agentMessage("operation_failed"), nil)
		return
	}
	if !h.persistAgentMutationLocked(request.Context(), payload.ChangeSetID, pending) {
		h.writeAgentErrorLocked(response, http.StatusConflict, "change_set_generation_mismatch", agentMessage("change_set_generation_mismatch"), nil)
		return
	}
	h.writeAgentSuccessLocked(response, http.StatusOK, agentMutationResult(pending, map[string]any{"component_id": payload.ComponentID, "changed_fields": changedFields}))
}

func (h *Handler) relationshipChangeStateLocked(expected agentapi.StatePreconditions, sourceID string) (architecture.Snapshot, *pendingChangeSet, architecture.ComponentChange, int, *agentapi.Error) {
	snapshot, pending, stateErr := h.checkAgentStateLocked(expected)
	if stateErr != nil {
		return architecture.Snapshot{}, nil, architecture.ComponentChange{}, -1, stateErr
	}
	if sourceID == "" {
		return architecture.Snapshot{}, nil, architecture.ComponentChange{}, -1, &agentapi.Error{Code: "invalid_request", Message: "Source Component ID is required.", Details: map[string]any{"field": "source_id"}}
	}
	change, index, found := componentChangeLocked(snapshot, pending, sourceID)
	if !found {
		return architecture.Snapshot{}, nil, architecture.ComponentChange{}, -1, &agentapi.Error{Code: "target_not_found", Message: agentMessage("target_not_found"), Details: map[string]any{"component_id": sourceID}}
	}
	return snapshot, pending, change, index, nil
}

func (h *Handler) commitRelationshipChangeLocked(ctx context.Context, snapshot architecture.Snapshot, pending *pendingChangeSet, change architecture.ComponentChange, index int) *pendingChangeSet {
	change.RelationshipsChanged = true
	return h.keepComponentChangeLocked(ctx, snapshot, pending, change, index)
}

func (h *Handler) agentRelationshipAdd(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.RelationshipAddRequest](h, response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	snapshot, pending, change, index, stateErr := h.relationshipChangeStateLocked(payload.StatePreconditions, payload.SourceID)
	if stateErr != nil {
		h.writeAgentDomainErrorLocked(response, agentDomainErrorStatus(stateErr), stateErr)
		return
	}
	change.Relationships = append(change.Relationships, architecture.AuthoringRelationship{TargetID: payload.TargetID, Label: payload.Label})
	pending = h.commitRelationshipChangeLocked(request.Context(), snapshot, pending, change, index)
	if pendingOperationFailed(pending) {
		h.writeAgentErrorLocked(response, http.StatusInternalServerError, "operation_failed", agentMessage("operation_failed"), nil)
		return
	}
	if !h.persistAgentMutationLocked(request.Context(), payload.ChangeSetID, pending) {
		h.writeAgentErrorLocked(response, http.StatusConflict, "change_set_generation_mismatch", agentMessage("change_set_generation_mismatch"), nil)
		return
	}
	h.writeAgentSuccessLocked(response, http.StatusOK, agentMutationResult(pending, map[string]any{
		"source_id": payload.SourceID, "target_id": payload.TargetID, "label": payload.Label,
	}))
}

func (h *Handler) agentRelationshipEdit(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.RelationshipEditRequest](h, response, request)
	if !ok {
		return
	}
	if payload.Occurrence < 1 {
		h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "Relationship occurrence must be a one-based integer.", map[string]any{"field": "occurrence"})
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	snapshot, pending, change, index, stateErr := h.relationshipChangeStateLocked(payload.StatePreconditions, payload.SourceID)
	if stateErr != nil {
		h.writeAgentDomainErrorLocked(response, agentDomainErrorStatus(stateErr), stateErr)
		return
	}
	selected := exactOccurrence(change.Relationships, payload.OldTargetID, payload.OldLabel, payload.Occurrence)
	if selected < 0 {
		h.writeAgentErrorLocked(response, http.StatusNotFound, "target_not_found", "That exact pending Relationship occurrence was not found. Inspect changes again.", map[string]any{
			"source_id": payload.SourceID, "old_target_id": payload.OldTargetID, "old_label": payload.OldLabel, "occurrence": payload.Occurrence,
		})
		return
	}
	old := change.Relationships[selected]
	if old.TargetID == payload.TargetID && old.Label == payload.Label {
		h.writeAgentSuccessLocked(response, http.StatusOK, agentMutationResult(pending, map[string]any{
			"source_id": payload.SourceID, "unchanged": true,
		}))
		return
	}
	change.Relationships[selected] = architecture.AuthoringRelationship{TargetID: payload.TargetID, Label: payload.Label}
	pending = h.commitRelationshipChangeLocked(request.Context(), snapshot, pending, change, index)
	if pendingOperationFailed(pending) {
		h.writeAgentErrorLocked(response, http.StatusInternalServerError, "operation_failed", agentMessage("operation_failed"), nil)
		return
	}
	if !h.persistAgentMutationLocked(request.Context(), payload.ChangeSetID, pending) {
		h.writeAgentErrorLocked(response, http.StatusConflict, "change_set_generation_mismatch", agentMessage("change_set_generation_mismatch"), nil)
		return
	}
	h.writeAgentSuccessLocked(response, http.StatusOK, agentMutationResult(pending, map[string]any{
		"source_id": payload.SourceID,
		"removed":   map[string]any{"target_id": old.TargetID, "label": old.Label, "occurrence": payload.Occurrence},
		"added":     map[string]any{"target_id": payload.TargetID, "label": payload.Label},
	}))
}

func (h *Handler) agentRelationshipRemove(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.RelationshipRemoveRequest](h, response, request)
	if !ok {
		return
	}
	if payload.Occurrence < 1 {
		h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "Relationship occurrence must be a one-based integer.", map[string]any{"field": "occurrence"})
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	snapshot, pending, change, index, stateErr := h.relationshipChangeStateLocked(payload.StatePreconditions, payload.SourceID)
	if stateErr != nil {
		h.writeAgentDomainErrorLocked(response, agentDomainErrorStatus(stateErr), stateErr)
		return
	}
	selected := exactOccurrence(change.Relationships, payload.TargetID, payload.Label, payload.Occurrence)
	if selected < 0 {
		h.writeAgentErrorLocked(response, http.StatusNotFound, "target_not_found", "That exact pending Relationship occurrence was not found. Inspect changes again.", map[string]any{
			"source_id": payload.SourceID, "target_id": payload.TargetID, "label": payload.Label, "occurrence": payload.Occurrence,
		})
		return
	}
	change.Relationships = append(change.Relationships[:selected], change.Relationships[selected+1:]...)
	pending = h.commitRelationshipChangeLocked(request.Context(), snapshot, pending, change, index)
	if pendingOperationFailed(pending) {
		h.writeAgentErrorLocked(response, http.StatusInternalServerError, "operation_failed", agentMessage("operation_failed"), nil)
		return
	}
	if !h.persistAgentMutationLocked(request.Context(), payload.ChangeSetID, pending) {
		h.writeAgentErrorLocked(response, http.StatusConflict, "change_set_generation_mismatch", agentMessage("change_set_generation_mismatch"), nil)
		return
	}
	h.writeAgentSuccessLocked(response, http.StatusOK, agentMutationResult(pending, map[string]any{
		"source_id": payload.SourceID, "removed": map[string]any{"target_id": payload.TargetID, "label": payload.Label, "occurrence": payload.Occurrence},
	}))
}

func (h *Handler) agentDiagramCreateDetail(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.DiagramCreateDetailRequest](h, response, request)
	if !ok {
		return
	}
	if payload.ComponentID == "" {
		h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "Anchor Component ID is required.", nil)
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	snapshot, pending, stateErr := h.checkAgentStateLocked(payload.StatePreconditions)
	if stateErr != nil {
		h.writeAgentDomainErrorLocked(response, http.StatusConflict, stateErr)
		return
	}
	pending, addition, homeDiagramID, operationError := h.createDetailDiagramLocked(request.Context(), snapshot, pending, payload.ComponentID, payload.Title)
	if operationError != "" {
		status := http.StatusConflict
		if operationError == changeTargetNotFound {
			status = http.StatusNotFound
		} else if operationError == changeOperationFailed {
			status = http.StatusInternalServerError
		}
		h.writeAgentErrorLocked(response, status, operationError, agentMessage(operationError), map[string]any{"component_id": payload.ComponentID})
		return
	}
	if !h.persistAgentMutationLocked(request.Context(), payload.ChangeSetID, pending) {
		h.writeAgentErrorLocked(response, http.StatusConflict, "change_set_generation_mismatch", agentMessage("change_set_generation_mismatch"), nil)
		return
	}
	h.writeAgentSuccessLocked(response, http.StatusOK, agentMutationResult(pending, map[string]any{
		"diagram_id": addition.ID, "anchor_component_id": payload.ComponentID, "parent_diagram_id": homeDiagramID, "title": payload.Title,
	}))
}

func (h *Handler) agentDiagramEditTitle(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.DiagramEditTitleRequest](h, response, request)
	if !ok {
		return
	}
	if payload.DiagramID == "" {
		h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "Diagram ID is required.", nil)
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	snapshot, pending, stateErr := h.checkAgentStateLocked(payload.StatePreconditions)
	if stateErr != nil {
		h.writeAgentDomainErrorLocked(response, http.StatusConflict, stateErr)
		return
	}
	pending, unchanged, operationError := h.editDiagramTitleLocked(request.Context(), snapshot, pending, payload.DiagramID, payload.Title)
	if operationError != "" {
		status := http.StatusNotFound
		if operationError == changeOperationFailed {
			status = http.StatusInternalServerError
		}
		h.writeAgentErrorLocked(response, status, operationError, agentMessage(operationError), map[string]any{"diagram_id": payload.DiagramID})
		return
	}
	if unchanged {
		h.writeAgentSuccessLocked(response, http.StatusOK, agentMutationResult(pending, map[string]any{"diagram_id": payload.DiagramID, "title": payload.Title, "unchanged": true}))
		return
	}
	if !h.persistAgentMutationLocked(request.Context(), payload.ChangeSetID, pending) {
		h.writeAgentErrorLocked(response, http.StatusConflict, "change_set_generation_mismatch", agentMessage("change_set_generation_mismatch"), nil)
		return
	}
	h.writeAgentSuccessLocked(response, http.StatusOK, agentMutationResult(pending, map[string]any{"diagram_id": payload.DiagramID, "title": payload.Title}))
}

func (h *Handler) agentComponentMoveHome(response http.ResponseWriter, request *http.Request) {
	payload, ok := decodeAgentRequest[agentapi.ComponentMoveHomeRequest](h, response, request)
	if !ok {
		return
	}
	if !requireNonEmpty(payload.ComponentID, payload.DiagramID) {
		h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "Component and destination Diagram IDs are required.", nil)
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	snapshot, pending, stateErr := h.checkAgentStateLocked(payload.StatePreconditions)
	if stateErr != nil {
		h.writeAgentDomainErrorLocked(response, http.StatusConflict, stateErr)
		return
	}
	pending, unchanged, operationError := h.moveComponentHomeLocked(request.Context(), snapshot, pending, payload.ComponentID, payload.DiagramID)
	if operationError != "" {
		status := http.StatusConflict
		if operationError == changeTargetNotFound {
			status = http.StatusNotFound
		} else if operationError == changeOperationFailed {
			status = http.StatusInternalServerError
		}
		h.writeAgentErrorLocked(response, status, operationError, agentMessage(operationError), nil)
		return
	}
	if unchanged {
		h.writeAgentSuccessLocked(response, http.StatusOK, map[string]any{"change_set_id": payload.ChangeSetID, "component_id": payload.ComponentID, "diagram_id": payload.DiagramID, "generation": payload.Generation, "unchanged": true})
		return
	}
	if !h.persistAgentMutationLocked(request.Context(), payload.ChangeSetID, pending) {
		h.writeAgentErrorLocked(response, http.StatusConflict, "change_set_generation_mismatch", agentMessage("change_set_generation_mismatch"), nil)
		return
	}
	h.writeAgentSuccessLocked(response, http.StatusOK, agentMutationResult(pending, map[string]any{"component_id": payload.ComponentID, "diagram_id": payload.DiagramID}))
}

func (h *Handler) agentDiagramShowComponent(response http.ResponseWriter, request *http.Request) {
	h.agentChangeReference(response, request, true)
}

func (h *Handler) agentDiagramStopShowingComponent(response http.ResponseWriter, request *http.Request) {
	h.agentChangeReference(response, request, false)
}

func (h *Handler) agentChangeReference(response http.ResponseWriter, request *http.Request, present bool) {
	payload, ok := decodeAgentRequest[agentapi.DiagramComponentRequest](h, response, request)
	if !ok {
		return
	}
	if !requireNonEmpty(payload.DiagramID, payload.ComponentID) {
		h.writeAgentError(response, http.StatusBadRequest, "invalid_request", "Diagram and Component IDs are required.", nil)
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	snapshot, pending, stateErr := h.checkAgentStateLocked(payload.StatePreconditions)
	if stateErr != nil {
		h.writeAgentDomainErrorLocked(response, http.StatusConflict, stateErr)
		return
	}
	pending, operationError := h.changeReferenceLocked(request.Context(), snapshot, pending, payload.DiagramID, payload.ComponentID, present)
	if operationError != "" {
		status := http.StatusConflict
		if operationError == changeTargetNotFound {
			status = http.StatusNotFound
		} else if operationError == changeOperationFailed {
			status = http.StatusInternalServerError
		}
		h.writeAgentErrorLocked(response, status, operationError, agentMessage(operationError), nil)
		return
	}
	if !h.persistAgentMutationLocked(request.Context(), payload.ChangeSetID, pending) {
		h.writeAgentErrorLocked(response, http.StatusConflict, "change_set_generation_mismatch", agentMessage("change_set_generation_mismatch"), nil)
		return
	}
	h.writeAgentSuccessLocked(response, http.StatusOK, agentMutationResult(pending, map[string]any{
		"diagram_id": payload.DiagramID, "component_id": payload.ComponentID, "present": present,
	}))
}
