package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"workbraid/internal/architecture"
	"workbraid/internal/associations"
	"workbraid/internal/projects"
)

const (
	maxRequestBody                 = 64 << 10
	architectureTransitionTimeout  = 30 * time.Second
	architectureObservationTimeout = 10 * time.Second
)

type Handler struct {
	db             *sql.DB
	expectedOrigin string
	uiDirectory    string
	architecture   *architecture.Manager
	setupMutex     sync.Mutex
	stateMutex     sync.Mutex
	loadedSnapshot *architecture.Snapshot
	loadedProject  *loadedProject
	loadedStale    bool
	acceptedDiff   string
	pending        *pendingChangeSet

	// publicationFailure is a focused test seam at the concrete post-CAS
	// publication boundary. Production never sets it.
	publicationFailure func() error
	// These two focused seams exercise real handler classification at the
	// final CAS boundary. Production never sets them.
	beforeAcceptedCAS           func(successor string)
	acceptedUpdateReportFailure func() error
	// beforeRefreshReobserve is a narrow test/checkpoint seam after an exact
	// replacement snapshot has loaded and before accepted is observed again.
	// Production never sets it.
	beforeRefreshReobserve func(revision string)
}

type loadedProject struct {
	sourceRoot  string
	projectName string
}

type pendingChangeSet struct {
	storeID                        string
	baseRevision                   string
	baseSnapshot                   architecture.Snapshot
	changes                        []architecture.ComponentChange
	diagramSetup                   *architecture.DiagramSetupChange
	newComponentHomes              []architecture.NewComponentHome
	candidate                      *architecture.Candidate
	generation                     uint64
	review                         *reviewBinding
	reviewBlocker                  string
	validationCode                 string
	validationItem                 string
	validationRelationshipPosition int
	validationRelationshipField    string
	stale                          bool
}

type reviewBinding struct {
	baseRevision  string
	candidateTree string
	generation    uint64
	diff          string
	candidate     architecture.Candidate
}

func NewHandler(db *sql.DB, expectedOrigin, uiDirectory, dataDirectory string) http.Handler {
	_, mux := newHandler(db, expectedOrigin, uiDirectory, dataDirectory)
	return mux
}

func newHandler(db *sql.DB, expectedOrigin, uiDirectory, dataDirectory string) (*Handler, http.Handler) {
	handler := &Handler{
		db:             db,
		expectedOrigin: expectedOrigin,
		uiDirectory:    uiDirectory,
		architecture:   architecture.NewManager(dataDirectory),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/projects/open", handler.openProject)
	mux.HandleFunc("POST /api/projects/initialize", handler.initializeProject)
	mux.HandleFunc("POST /api/architecture/components/add", handler.addComponent)
	mux.HandleFunc("POST /api/architecture/components/edit", handler.editComponent)
	mux.HandleFunc("POST /api/architecture/diagrams/setup", handler.setupDiagrams)
	mux.HandleFunc("POST /api/architecture/review", handler.reviewChanges)
	mux.HandleFunc("POST /api/architecture/accept", handler.acceptChanges)
	mux.HandleFunc("POST /api/architecture/discard", handler.discardChanges)
	mux.HandleFunc("POST /api/architecture/refresh", handler.refreshArchitecture)
	mux.HandleFunc("POST /api/projects/leave", handler.leaveProject)
	mux.Handle("/", handler.staticFiles())
	return handler, mux
}

type openProjectRequest struct {
	SourceRoot string `json:"source_root"`
}

type acceptChangesRequest struct {
	SourceRoot    string `json:"source_root"`
	BaseRevision  string `json:"base_revision"`
	CandidateTree string `json:"candidate_tree"`
	Generation    uint64 `json:"generation"`
}

type errorResponse struct {
	Code string `json:"code"`
}

const (
	errorPathRequired            = "path_required"
	errorPathRelative            = "path_relative"
	errorPathMissing             = "path_missing"
	errorPathNotDir              = "path_not_directory"
	errorOriginMismatch          = "origin_mismatch"
	errorLookupFailed            = "lookup_failed"
	errorSetupIncomplete         = "setup_incomplete"
	errorArchitectureUnavailable = "architecture_unavailable"
	errorArchitectureInvalid     = "architecture_invalid"
	errorArchitectureUnsupported = "architecture_unsupported"
	errorArchitectureNotOpen     = "architecture_not_open"
	errorChangesElsewhere        = "changes_elsewhere"
	errorComponentNotFound       = "component_not_found"
	errorChangeFailed            = "change_failed"
	errorReviewFailed            = "review_failed"
	errorReviewChanged           = "review_changed"
	errorArchitectureStale       = "architecture_stale"
	errorUpdateFailed            = "update_failed"
	errorUpdateUncertain         = "update_uncertain"
	errorUpdatedReload           = "updated_reload"
	errorPendingBlocksSwitch     = "pending_blocks_switch"
	errorRefreshFailed           = "refresh_failed"
	errorRefreshChanged          = "refresh_changed"
	errorRefreshUnavailable      = "refresh_unavailable"
	errorRefreshInvalid          = "refresh_invalid"
	errorRefreshUnsupported      = "refresh_unsupported"
	errorChangesUnavailable      = "changes_unavailable"
)

func (h *Handler) openProject(response http.ResponseWriter, request *http.Request) {
	if request.Header.Get("Origin") != h.expectedOrigin {
		writeJSON(response, http.StatusForbidden, errorResponse{Code: errorOriginMismatch})
		return
	}

	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var payload openProjectRequest
	if err := decoder.Decode(&payload); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorLookupFailed})
		return
	}
	if err := ensureJSONEnd(decoder); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorLookupFailed})
		return
	}

	inspection, err := projects.Inspect(request.Context(), h.db, payload.SourceRoot)
	if err != nil {
		writeProjectError(response, err)
		return
	}
	if !inspection.Known {
		h.stateMutex.Lock()
		if h.pending != nil && h.loadedProject != nil && h.loadedProject.sourceRoot != inspection.SourceRoot {
			result := h.currentArchitectureResponseLocked()
			result.ActionError = errorPendingBlocksSwitch
			h.stateMutex.Unlock()
			writeJSON(response, http.StatusConflict, result)
			return
		}
		if h.loadedProject != nil && h.loadedProject.sourceRoot != inspection.SourceRoot {
			h.clearLoadedProjectLocked()
		}
		h.stateMutex.Unlock()
		writeJSON(response, http.StatusOK, inspection)
		return
	}
	if err := h.architecture.ValidateSourceIsolation(inspection.SourceRoot); err != nil {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureInvalid})
		return
	}

	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if h.pending != nil && h.loadedProject != nil && h.loadedProject.sourceRoot != inspection.SourceRoot {
		result := h.currentArchitectureResponseLocked()
		result.ActionError = errorPendingBlocksSwitch
		writeJSON(response, http.StatusConflict, result)
		return
	}
	snapshot, err := h.architecture.LoadAccepted(request.Context(), inspection.StoreID)
	if err != nil {
		writeArchitectureLoadError(response, err)
		return
	}
	stalePending := h.publishSnapshotLocked(inspection.SourceRoot, inspection.ProjectName, snapshot)
	result := h.currentArchitectureResponseLocked()
	if stalePending {
		result.ActionError = errorArchitectureStale
	}
	writeJSON(response, http.StatusOK, result)
}

type architectureResponse struct {
	SourceRoot      string              `json:"source_root"`
	ProjectName     string              `json:"project_name"`
	State           string              `json:"state"`
	Revision        string              `json:"revision"`
	FormatVersion   int                 `json:"format_version"`
	ComponentCount  int                 `json:"component_count"`
	ComponentTitles []string            `json:"component_titles"`
	Components      []componentResponse `json:"components"`
	RootDiagramID   string              `json:"root_diagram_id,omitempty"`
	Diagrams        []diagramResponse   `json:"diagrams,omitempty"`
	Changes         *changesResponse    `json:"changes,omitempty"`
	Stale           bool                `json:"stale,omitempty"`
	ParentDiff      string              `json:"parent_diff,omitempty"`
	ActionError     string              `json:"action_error,omitempty"`
}

type componentResponse struct {
	ID            string                 `json:"id"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	Filename      string                 `json:"filename"`
	Relationships []relationshipResponse `json:"relationships"`
}

type relationshipResponse struct {
	TargetID      string `json:"target_id"`
	Label         string `json:"label"`
	ProjectionKey string `json:"projection_key,omitempty"`
}

type pendingComponentResponse struct {
	ID            string                 `json:"id"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	New           bool                   `json:"new"`
	Relationships []relationshipResponse `json:"relationships"`
}

type relationshipTargetResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Context string `json:"context,omitempty"`
	New     bool   `json:"new,omitempty"`
}

type changesResponse struct {
	Components                     []pendingComponentResponse   `json:"components"`
	RelationshipTargets            []relationshipTargetResponse `json:"relationship_targets"`
	Valid                          bool                         `json:"valid"`
	ValidationCode                 string                       `json:"validation_code,omitempty"`
	ValidationItem                 string                       `json:"validation_item,omitempty"`
	ValidationRelationshipPosition int                          `json:"validation_relationship_position,omitempty"`
	ValidationRelationshipField    string                       `json:"validation_relationship_field,omitempty"`
	Review                         *reviewResponse              `json:"review,omitempty"`
	ReviewBlocker                  string                       `json:"review_blocker,omitempty"`
	Stale                          bool                         `json:"stale,omitempty"`
	DiagramSetup                   bool                         `json:"diagram_setup,omitempty"`
	LegacyReadOnly                 bool                         `json:"legacy_read_only,omitempty"`
}

type reviewResponse struct {
	Diff          string                     `json:"diff"`
	BaseRevision  string                     `json:"base_revision"`
	CandidateTree string                     `json:"candidate_tree"`
	Generation    uint64                     `json:"generation"`
	Before        snapshotProjectionResponse `json:"before"`
	WithChanges   snapshotProjectionResponse `json:"with_changes"`
	Comparison    reviewComparisonResponse   `json:"comparison"`
}

func responseForSnapshot(sourceRoot, projectName string, snapshot architecture.Snapshot, pending *pendingChangeSet, stale bool, parentDiff string) architectureResponse {
	projection := projectSnapshot(snapshot, "")
	state := "ready"
	if projection.ComponentCount == 0 {
		state = "empty"
	}
	result := architectureResponse{
		SourceRoot:      sourceRoot,
		ProjectName:     projectName,
		State:           state,
		Revision:        projection.Revision,
		FormatVersion:   projection.FormatVersion,
		ComponentCount:  projection.ComponentCount,
		ComponentTitles: projection.ComponentTitles,
		Components:      projection.Components,
		RootDiagramID:   projection.RootDiagramID,
		Diagrams:        projection.Diagrams,
		Stale:           stale,
		ParentDiff:      parentDiff,
	}
	if pending != nil && pending.storeID == snapshot.StoreID() {
		pendingAccepted := pending.baseSnapshot.AuthoringComponents()
		changes := make([]pendingComponentResponse, len(pending.changes))
		for index, change := range pending.changes {
			relationships := make([]relationshipResponse, len(change.Relationships))
			for relationshipIndex, relationship := range change.Relationships {
				relationships[relationshipIndex] = relationshipResponse{TargetID: relationship.TargetID, Label: relationship.Label}
			}
			changes[index] = pendingComponentResponse{
				ID: change.ID, Title: change.Title, Description: change.Description, New: change.New, Relationships: relationships,
			}
		}
		result.Changes = &changesResponse{
			Components:                     changes,
			RelationshipTargets:            relationshipTargets(pendingAccepted, pending.changes),
			Valid:                          pending.candidate != nil,
			ValidationCode:                 pending.validationCode,
			ValidationItem:                 pending.validationItem,
			ValidationRelationshipPosition: pending.validationRelationshipPosition,
			ValidationRelationshipField:    pending.validationRelationshipField,
			ReviewBlocker:                  pending.reviewBlocker,
			Stale:                          pending.stale,
			DiagramSetup:                   pending.diagramSetup != nil,
			LegacyReadOnly:                 snapshot.FormatVersion() == 1 && pending.diagramSetup == nil,
		}
		if !pending.stale && pending.review != nil && pending.review.generation == pending.generation && pending.candidate != nil && pending.review.candidateTree == pending.candidate.Tree() {
			before, withChanges, comparison := captureReviewPresentation(pending.baseSnapshot, pending.review.candidate.Snapshot())
			result.Changes.Review = &reviewResponse{
				Diff: pending.review.diff, BaseRevision: pending.review.baseRevision,
				CandidateTree: pending.review.candidateTree, Generation: pending.review.generation,
				Before: before, WithChanges: withChanges, Comparison: comparison,
			}
		}
	}
	return result
}

func relationshipTargets(accepted []architecture.AuthoringComponent, changes []architecture.ComponentChange) []relationshipTargetResponse {
	changesByID := make(map[string]architecture.ComponentChange, len(changes))
	for _, change := range changes {
		changesByID[change.ID] = change
	}
	targets := make([]relationshipTargetResponse, 0, len(accepted)+len(changes))
	for _, component := range accepted {
		title := component.Title
		if change, exists := changesByID[component.ID]; exists {
			title = change.Title
		}
		targets = append(targets, relationshipTargetResponse{ID: component.ID, Title: title, Context: component.Filename})
	}
	for _, change := range changes {
		if change.New {
			targets = append(targets, relationshipTargetResponse{ID: change.ID, Title: change.Title, Context: filepath.Base(change.Path), New: true})
		}
	}
	titleCounts := make(map[string]int, len(targets))
	for _, target := range targets {
		titleCounts[presentedTargetTitle(target.Title)]++
	}
	for index := range targets {
		if titleCounts[presentedTargetTitle(targets[index].Title)] < 2 {
			targets[index].Context = ""
		}
	}
	return targets
}

func presentedTargetTitle(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Untitled component"
	}
	return title
}

func (h *Handler) initializeProject(response http.ResponseWriter, request *http.Request) {
	if request.Header.Get("Origin") != h.expectedOrigin {
		writeJSON(response, http.StatusForbidden, errorResponse{Code: errorOriginMismatch})
		return
	}

	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var payload openProjectRequest
	if err := decoder.Decode(&payload); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorLookupFailed})
		return
	}
	if err := ensureJSONEnd(decoder); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorLookupFailed})
		return
	}

	inspection, err := projects.Inspect(request.Context(), h.db, payload.SourceRoot)
	if err != nil {
		writeProjectError(response, err)
		return
	}
	if err := h.architecture.ValidateSourceIsolation(inspection.SourceRoot); err != nil {
		writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorSetupIncomplete})
		return
	}
	h.setupMutex.Lock()
	defer h.setupMutex.Unlock()
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if h.pending != nil && h.loadedProject != nil && h.loadedProject.sourceRoot != inspection.SourceRoot {
		result := h.currentArchitectureResponseLocked()
		result.ActionError = errorPendingBlocksSwitch
		writeJSON(response, http.StatusConflict, result)
		return
	}

	storeID := inspection.StoreID
	if !inspection.Known {
		proposedStoreID := uuid.NewString()
		storeID, err = associations.GetOrCreate(request.Context(), h.db, inspection.SourceRoot, proposedStoreID)
		if err != nil {
			writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorLookupFailed})
			return
		}
	}

	snapshot, err := h.architecture.InitializeOrLoad(
		request.Context(), storeID, inspection.ProjectName, inspection.SourceRoot,
	)
	if err != nil {
		switch {
		case errors.Is(err, architecture.ErrUnsupported):
			writeJSON(response, http.StatusUnprocessableEntity, errorResponse{Code: errorArchitectureUnsupported})
		case errors.Is(err, architecture.ErrInvalid):
			writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureInvalid})
		default:
			writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorSetupIncomplete})
		}
		return
	}

	h.publishSnapshotLocked(inspection.SourceRoot, inspection.ProjectName, snapshot)
	writeJSON(response, http.StatusOK, h.currentArchitectureResponseLocked())
}

func (h *Handler) publishSnapshotLocked(sourceRoot, projectName string, snapshot architecture.Snapshot) bool {
	if h.pending != nil && h.pending.storeID == snapshot.StoreID() && h.pending.baseRevision != snapshot.Revision() &&
		h.pending.baseSnapshot.StoreID() == h.pending.storeID && h.pending.baseSnapshot.Revision() == h.pending.baseRevision {
		h.pending.stale = true
		h.pending.review = nil
		base := h.pending.baseSnapshot
		h.loadedSnapshot = &base
		h.loadedProject = &loadedProject{sourceRoot: sourceRoot, projectName: projectName}
		h.loadedStale = true
		h.acceptedDiff = ""
		return true
	}
	keepAcceptedDiff := h.loadedSnapshot != nil && h.loadedSnapshot.StoreID() == snapshot.StoreID() && h.loadedSnapshot.Revision() == snapshot.Revision()
	h.loadedSnapshot = &snapshot
	h.loadedProject = &loadedProject{sourceRoot: sourceRoot, projectName: projectName}
	h.loadedStale = false
	if !keepAcceptedDiff {
		h.acceptedDiff = ""
	}
	return false
}

func (h *Handler) currentArchitectureResponse() architectureResponse {
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	return h.currentArchitectureResponseLocked()
}

func (h *Handler) currentArchitectureResponseLocked() architectureResponse {
	if h.loadedSnapshot == nil || h.loadedProject == nil {
		return architectureResponse{}
	}
	return responseForSnapshot(h.loadedProject.sourceRoot, h.loadedProject.projectName, *h.loadedSnapshot, h.pending, h.loadedStale, h.acceptedDiff)
}

func (h *Handler) clearLoadedProjectLocked() {
	h.loadedSnapshot = nil
	h.loadedProject = nil
	h.loadedStale = false
	h.acceptedDiff = ""
}

func (h *Handler) refreshArchitecture(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeArchitectureAction(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if h.loadedSnapshot == nil || h.loadedProject == nil || payload.SourceRoot != h.loadedProject.sourceRoot {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureNotOpen})
		return
	}

	loaded := *h.loadedSnapshot
	observationContext, cancelObservation := context.WithTimeout(request.Context(), architectureObservationTimeout)
	observed, present, err := h.architecture.AcceptedRevision(observationContext, loaded)
	cancelObservation()
	if err != nil {
		h.writeRefreshResultLocked(response, http.StatusServiceUnavailable, errorRefreshFailed)
		return
	}
	if !present {
		h.markKnownNonCurrentLocked()
		h.writeRefreshResultLocked(response, http.StatusConflict, errorRefreshUnavailable)
		return
	}
	if observed == loaded.Revision() && !h.loadedStale {
		writeJSON(response, http.StatusOK, h.currentArchitectureResponseLocked())
		return
	}

	loadContext, cancelLoad := context.WithTimeout(request.Context(), architectureTransitionTimeout)
	replacement, loadErr := h.architecture.LoadRevision(loadContext, loaded, observed)
	cancelLoad()
	if loadErr != nil {
		h.markKnownNonCurrentLocked()
		switch {
		case errors.Is(loadErr, architecture.ErrUnsupported):
			h.writeRefreshResultLocked(response, http.StatusUnprocessableEntity, errorRefreshUnsupported)
		case errors.Is(loadErr, architecture.ErrUnavailable):
			h.writeRefreshResultLocked(response, http.StatusConflict, errorRefreshUnavailable)
		default:
			h.writeRefreshResultLocked(response, http.StatusConflict, errorRefreshInvalid)
		}
		return
	}
	if h.beforeRefreshReobserve != nil {
		h.beforeRefreshReobserve(observed)
	}

	finalContext, cancelFinal := context.WithTimeout(request.Context(), architectureObservationTimeout)
	finalRevision, finalPresent, finalErr := h.architecture.AcceptedRevision(finalContext, loaded)
	cancelFinal()
	if finalErr != nil {
		h.writeRefreshResultLocked(response, http.StatusServiceUnavailable, errorRefreshFailed)
		return
	}
	if !finalPresent {
		h.markKnownNonCurrentLocked()
		h.writeRefreshResultLocked(response, http.StatusConflict, errorRefreshUnavailable)
		return
	}
	if finalRevision != observed {
		if finalRevision == loaded.Revision() {
			// Authority returned to the retained, already validated snapshot.
			// A pending set already known stale stays stale until discarded.
			h.loadedStale = false
			writeJSON(response, http.StatusOK, h.currentArchitectureResponseLocked())
			return
		}
		h.markKnownNonCurrentLocked()
		h.writeRefreshResultLocked(response, http.StatusConflict, errorRefreshChanged)
		return
	}

	h.loadedSnapshot = &replacement
	h.loadedStale = false
	h.acceptedDiff = ""
	if h.pending != nil && (h.pending.storeID != replacement.StoreID() || h.pending.baseRevision != replacement.Revision()) {
		h.markPendingStaleLocked()
	}
	writeJSON(response, http.StatusOK, h.currentArchitectureResponseLocked())
}

func (h *Handler) writeRefreshResultLocked(response http.ResponseWriter, status int, actionError string) {
	result := h.currentArchitectureResponseLocked()
	result.ActionError = actionError
	writeJSON(response, status, result)
}

func (h *Handler) markKnownNonCurrentLocked() {
	h.loadedStale = true
	h.markPendingStaleLocked()
}

func (h *Handler) markPendingStaleLocked() {
	if h.pending == nil {
		return
	}
	h.pending.stale = true
	h.pending.review = nil
}

func (h *Handler) discardChanges(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeArchitectureAction(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if h.loadedSnapshot == nil || h.loadedProject == nil || payload.SourceRoot != h.loadedProject.sourceRoot {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureNotOpen})
		return
	}
	h.pending = nil
	writeJSON(response, http.StatusOK, h.currentArchitectureResponseLocked())
}

func (h *Handler) leaveProject(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeArchitectureAction(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if h.loadedSnapshot == nil || h.loadedProject == nil || payload.SourceRoot != h.loadedProject.sourceRoot {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureNotOpen})
		return
	}
	if h.pending != nil {
		result := h.currentArchitectureResponseLocked()
		result.ActionError = errorPendingBlocksSwitch
		writeJSON(response, http.StatusConflict, result)
		return
	}
	h.clearLoadedProjectLocked()
	response.WriteHeader(http.StatusNoContent)
}

type componentMutationRequest struct {
	SourceRoot           string                 `json:"source_root"`
	ExpectedRevision     string                 `json:"expected_revision,omitempty"`
	ComponentID          string                 `json:"component_id,omitempty"`
	Title                string                 `json:"title,omitempty"`
	Description          string                 `json:"description,omitempty"`
	TitleChanged         bool                   `json:"title_changed,omitempty"`
	DescriptionChanged   bool                   `json:"description_changed,omitempty"`
	Relationships        []relationshipResponse `json:"relationships,omitempty"`
	RelationshipsChanged bool                   `json:"relationships_changed,omitempty"`
	DiagramID            string                 `json:"diagram_id,omitempty"`
}

func (h *Handler) addComponent(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeComponentMutation(response, request)
	if !ok {
		return
	}
	h.mutateComponent(response, request, payload, true)
}

func (h *Handler) editComponent(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeComponentMutation(response, request)
	if !ok {
		return
	}
	if payload.ComponentID == "" {
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorComponentNotFound})
		return
	}
	h.mutateComponent(response, request, payload, false)
}

func (h *Handler) decodeComponentMutation(response http.ResponseWriter, request *http.Request) (componentMutationRequest, bool) {
	if request.Header.Get("Origin") != h.expectedOrigin {
		writeJSON(response, http.StatusForbidden, errorResponse{Code: errorOriginMismatch})
		return componentMutationRequest{}, false
	}
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var payload componentMutationRequest
	if err := decoder.Decode(&payload); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorLookupFailed})
		return componentMutationRequest{}, false
	}
	if err := ensureJSONEnd(decoder); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorLookupFailed})
		return componentMutationRequest{}, false
	}
	return payload, true
}

func (h *Handler) mutateComponent(response http.ResponseWriter, request *http.Request, payload componentMutationRequest, add bool) {
	payload.Title = strings.TrimSpace(payload.Title)
	if add || payload.DescriptionChanged {
		payload.Description = normalizeAuthoredDescription(payload.Description)
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if h.loadedSnapshot == nil || h.loadedProject == nil || payload.SourceRoot != h.loadedProject.sourceRoot {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureNotOpen})
		return
	}
	if h.loadedStale {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureStale})
		return
	}
	snapshot := *h.loadedSnapshot
	if payload.ExpectedRevision == "" || payload.ExpectedRevision != snapshot.Revision() {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesElsewhere})
		return
	}
	if snapshot.FormatVersion() != 2 {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesUnavailable})
		return
	}
	if h.pending != nil && (h.pending.stale || h.pending.storeID != snapshot.StoreID() || h.pending.baseRevision != snapshot.Revision()) {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesElsewhere})
		return
	}
	if h.pending == nil {
		h.pending = &pendingChangeSet{storeID: snapshot.StoreID(), baseRevision: snapshot.Revision(), baseSnapshot: snapshot}
	}

	var change architecture.ComponentChange
	changeIndex := -1
	if add {
		change = h.architecture.NewComponentChange(snapshot, h.pending.changes, payload.Title, payload.Description)
		change.Relationships = authoringRelationships(payload.Relationships)
		change.RelationshipsChanged = true
		homeDiagramID := payload.DiagramID
		if homeDiagramID == "" {
			homeDiagramID = snapshot.RootDiagramID()
		}
		if !snapshot.HasDiagram(homeDiagramID) {
			if len(h.pending.changes) == 0 {
				h.pending = nil
			}
			writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorChangeFailed})
			return
		}
		h.pending.newComponentHomes = append(h.pending.newComponentHomes, architecture.NewComponentHome{
			ComponentID: change.ID, DiagramID: homeDiagramID,
		})
	} else {
		for index := range h.pending.changes {
			if h.pending.changes[index].ID == payload.ComponentID {
				change = h.pending.changes[index]
				changeIndex = index
				break
			}
		}
		if changeIndex < 0 {
			var found bool
			change, found = snapshot.ChangeForAcceptedComponent(payload.ComponentID)
			if !found {
				writeJSON(response, http.StatusNotFound, errorResponse{Code: errorComponentNotFound})
				return
			}
		}
		if payload.TitleChanged {
			change.Title = payload.Title
			change.TitleChanged = true
		}
		if payload.DescriptionChanged {
			change.Description = payload.Description
			change.DescriptionChanged = true
		}
		if payload.RelationshipsChanged {
			change.Relationships = authoringRelationships(payload.Relationships)
			change.RelationshipsChanged = true
		}
	}
	if changeIndex >= 0 {
		h.pending.changes[changeIndex] = change
	} else {
		h.pending.changes = append(h.pending.changes, change)
	}
	h.pending.generation++
	h.pending.reviewBlocker = ""

	candidate, err := h.constructCandidate(request.Context(), snapshot, h.pending)
	h.pending.candidate = nil
	h.pending.validationCode = ""
	h.pending.validationItem = ""
	h.pending.validationRelationshipPosition = 0
	h.pending.validationRelationshipField = ""
	if err != nil {
		var componentError *architecture.ComponentValidationError
		if errors.As(err, &componentError) {
			h.pending.validationItem = componentError.ComponentID
			h.pending.validationRelationshipPosition = componentError.RelationshipPosition
			h.pending.validationRelationshipField = componentError.RelationshipField
		} else {
			h.pending.validationItem = change.ID
		}
		switch {
		case errors.Is(err, architecture.ErrTitleRequired):
			h.pending.validationCode = "title_required"
		case errors.Is(err, architecture.ErrTitleOneLine):
			h.pending.validationCode = "title_one_line"
		case errors.Is(err, architecture.ErrRelationshipLabelRequired):
			h.pending.validationCode = "relationship_label_required"
		case errors.Is(err, architecture.ErrRelationshipTargetRequired):
			h.pending.validationCode = "relationship_target_required"
		case errors.Is(err, architecture.ErrInvalid):
			h.pending.validationCode = "change_invalid"
		default:
			h.pending.validationCode = "change_unavailable"
			writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorChangeFailed})
			return
		}
	} else {
		h.pending.candidate = &candidate
	}
	writeJSON(response, http.StatusOK, responseForSnapshot(h.loadedProject.sourceRoot, h.loadedProject.projectName, snapshot, h.pending, h.loadedStale, h.acceptedDiff))
}

func (h *Handler) constructCandidate(ctx context.Context, snapshot architecture.Snapshot, pending *pendingChangeSet) (architecture.Candidate, error) {
	return h.architecture.ConstructCandidate(ctx, snapshot, pending.changes, architecture.CandidateComposition{
		DiagramSetup: pending.diagramSetup, NewComponentHomes: pending.newComponentHomes,
	})
}

func (h *Handler) setupDiagrams(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeArchitectureAction(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if h.loadedSnapshot == nil || h.loadedProject == nil || payload.SourceRoot != h.loadedProject.sourceRoot {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureNotOpen})
		return
	}
	if h.loadedStale {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureStale})
		return
	}
	snapshot := *h.loadedSnapshot
	if snapshot.FormatVersion() != 1 || h.pending != nil {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesUnavailable})
		return
	}
	setup := snapshot.NewDiagramSetupChange()
	h.pending = &pendingChangeSet{
		storeID: snapshot.StoreID(), baseRevision: snapshot.Revision(), baseSnapshot: snapshot,
		diagramSetup: &setup, generation: 1,
	}
	candidate, err := h.constructCandidate(request.Context(), snapshot, h.pending)
	if err != nil {
		h.pending = nil
		writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorChangeFailed})
		return
	}
	h.pending.candidate = &candidate
	writeJSON(response, http.StatusOK, h.currentArchitectureResponseLocked())
}

func authoringRelationships(values []relationshipResponse) []architecture.AuthoringRelationship {
	relationships := make([]architecture.AuthoringRelationship, len(values))
	for index, value := range values {
		relationships[index] = architecture.AuthoringRelationship{TargetID: value.TargetID, Label: value.Label}
	}
	return relationships
}

func normalizeAuthoredDescription(description string) string {
	if description == "" || strings.HasSuffix(description, "\n") {
		return description
	}
	return description + "\n"
}

func (h *Handler) reviewChanges(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeArchitectureAction(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	if h.loadedSnapshot == nil || h.loadedProject == nil || payload.SourceRoot != h.loadedProject.sourceRoot {
		h.stateMutex.Unlock()
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureNotOpen})
		return
	}
	if h.loadedStale {
		h.stateMutex.Unlock()
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureStale})
		return
	}
	snapshot := *h.loadedSnapshot
	if h.pending == nil || h.pending.stale || h.pending.storeID != snapshot.StoreID() || h.pending.baseRevision != snapshot.Revision() {
		h.stateMutex.Unlock()
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorReviewFailed})
		return
	}

	if snapshot.FormatVersion() == 1 && h.pending.diagramSetup == nil {
		h.stateMutex.Unlock()
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesUnavailable})
		return
	}
	candidate, err := h.constructCandidate(request.Context(), snapshot, h.pending)
	h.pending.review = nil
	h.pending.reviewBlocker = ""
	if err != nil {
		h.recordCandidateValidation(h.pending, err)
		status := http.StatusUnprocessableEntity
		if h.pending.validationCode == "change_unavailable" {
			status = http.StatusInternalServerError
		} else {
			h.pending.reviewBlocker = h.pending.validationCode
		}
		result := responseForSnapshot(h.loadedProject.sourceRoot, h.loadedProject.projectName, snapshot, h.pending, false, h.acceptedDiff)
		result.ActionError = errorReviewFailed
		h.stateMutex.Unlock()
		writeJSON(response, status, result)
		return
	}
	diff, err := h.architecture.CandidateDiff(request.Context(), snapshot, candidate)
	if err != nil {
		result := responseForSnapshot(h.loadedProject.sourceRoot, h.loadedProject.projectName, snapshot, h.pending, false, h.acceptedDiff)
		result.ActionError = errorReviewFailed
		h.stateMutex.Unlock()
		writeJSON(response, http.StatusInternalServerError, result)
		return
	}
	h.pending.candidate = &candidate
	h.pending.validationCode = ""
	h.pending.validationItem = ""
	h.pending.validationRelationshipPosition = 0
	h.pending.validationRelationshipField = ""
	h.pending.review = &reviewBinding{
		baseRevision: snapshot.Revision(), candidateTree: candidate.Tree(), generation: h.pending.generation,
		diff: string(diff), candidate: candidate,
	}
	// Build the complete immutable visual presentation while the binding and
	// pending generation are protected by the same concrete state lock. JSON
	// serialization can then proceed without blocking invalidating mutations.
	result := responseForSnapshot(h.loadedProject.sourceRoot, h.loadedProject.projectName, snapshot, h.pending, false, h.acceptedDiff)
	h.stateMutex.Unlock()
	writeJSON(response, http.StatusOK, result)
}

func (h *Handler) acceptChanges(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeAcceptChanges(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if h.loadedSnapshot == nil || h.loadedProject == nil || payload.SourceRoot != h.loadedProject.sourceRoot {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureNotOpen})
		return
	}
	if h.loadedStale {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureStale})
		return
	}
	snapshot := *h.loadedSnapshot
	pending := h.pending
	if pending == nil || pending.stale || pending.review == nil {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorReviewFailed})
		return
	}
	review := pending.review
	if payload.BaseRevision != review.baseRevision || payload.CandidateTree != review.candidateTree || payload.Generation != review.generation {
		result := responseForSnapshot(h.loadedProject.sourceRoot, h.loadedProject.projectName, snapshot, pending, false, h.acceptedDiff)
		result.ActionError = errorReviewChanged
		if result.Changes != nil {
			// The backend may hold a newer review for another browser. Do not
			// expose it as though this client had inspected it.
			result.Changes.Review = nil
		}
		writeJSON(response, http.StatusConflict, result)
		return
	}
	if pending.storeID != snapshot.StoreID() || pending.baseRevision != snapshot.Revision() ||
		review.baseRevision != pending.baseRevision || review.generation != pending.generation ||
		pending.candidate == nil || review.candidateTree != pending.candidate.Tree() || review.candidateTree != review.candidate.Tree() {
		pending.review = nil
		result := responseForSnapshot(h.loadedProject.sourceRoot, h.loadedProject.projectName, snapshot, pending, false, h.acceptedDiff)
		result.ActionError = errorReviewChanged
		writeJSON(response, http.StatusConflict, result)
		return
	}
	// Once the human confirms, the local authority transition must reach a
	// classified boundary even if the browser disconnects before the response.
	transitionContext, cancelTransition := context.WithTimeout(context.Background(), architectureTransitionTimeout)
	defer cancelTransition()

	observed, present, err := h.architecture.AcceptedRevision(transitionContext, snapshot)
	if err != nil || !present {
		h.loadedStale = true
		pending.review = nil
		result := responseForSnapshot(h.loadedProject.sourceRoot, h.loadedProject.projectName, snapshot, pending, true, h.acceptedDiff)
		result.ActionError = errorUpdateUncertain
		writeJSON(response, http.StatusConflict, result)
		return
	}
	if observed != review.baseRevision {
		h.markStale(pending)
		result := responseForSnapshot(h.loadedProject.sourceRoot, h.loadedProject.projectName, snapshot, pending, true, h.acceptedDiff)
		result.ActionError = errorArchitectureStale
		writeJSON(response, http.StatusConflict, result)
		return
	}

	successor, err := h.architecture.CreateSuccessor(transitionContext, snapshot, review.candidate)
	if err != nil {
		result := responseForSnapshot(h.loadedProject.sourceRoot, h.loadedProject.projectName, snapshot, pending, false, h.acceptedDiff)
		result.ActionError = errorUpdateFailed
		writeJSON(response, http.StatusInternalServerError, result)
		return
	}
	if h.beforeAcceptedCAS != nil {
		h.beforeAcceptedCAS(successor)
	}
	updateErr := h.architecture.AdvanceAccepted(transitionContext, snapshot, successor)
	if updateErr == nil && h.acceptedUpdateReportFailure != nil {
		updateErr = h.acceptedUpdateReportFailure()
	}
	if updateErr != nil {
		observationContext, cancelObservation := context.WithTimeout(context.Background(), architectureObservationTimeout)
		observed, present, observeErr := h.architecture.AcceptedRevision(observationContext, snapshot)
		cancelObservation()
		if observeErr != nil || !present {
			h.loadedStale = true
			pending.review = nil
			result := responseForSnapshot(h.loadedProject.sourceRoot, h.loadedProject.projectName, snapshot, pending, true, h.acceptedDiff)
			result.ActionError = errorUpdateUncertain
			writeJSON(response, http.StatusInternalServerError, result)
			return
		}
		if observed != successor && observed != review.baseRevision {
			h.markStale(pending)
			result := responseForSnapshot(h.loadedProject.sourceRoot, h.loadedProject.projectName, snapshot, pending, true, h.acceptedDiff)
			result.ActionError = errorArchitectureStale
			writeJSON(response, http.StatusConflict, result)
			return
		}
		if observed == review.baseRevision {
			result := responseForSnapshot(h.loadedProject.sourceRoot, h.loadedProject.projectName, snapshot, pending, false, h.acceptedDiff)
			result.ActionError = errorUpdateFailed
			writeJSON(response, http.StatusInternalServerError, result)
			return
		}
	}
	// The authoritative ref names our successor. Consume the pending change
	// before any fallible publication or response work.
	h.pending = nil
	h.acceptedDiff = review.diff
	acceptedSnapshot := review.candidate.SnapshotAt(successor)
	if h.publicationFailure != nil {
		if err := h.publicationFailure(); err != nil {
			recoveryContext, cancelRecovery := context.WithTimeout(context.Background(), architectureTransitionTimeout)
			recovered, loadErr := h.architecture.LoadAccepted(recoveryContext, snapshot.StoreID())
			cancelRecovery()
			if loadErr == nil {
				h.loadedSnapshot = &recovered
				h.loadedStale = false
			} else {
				h.loadedStale = true
			}
			result := responseForSnapshot(h.loadedProject.sourceRoot, h.loadedProject.projectName, *h.loadedSnapshot, nil, h.loadedStale, h.acceptedDiff)
			result.ActionError = errorUpdatedReload
			writeJSON(response, http.StatusInternalServerError, result)
			return
		}
	}
	h.loadedSnapshot = &acceptedSnapshot
	h.loadedStale = false
	writeJSON(response, http.StatusOK, responseForSnapshot(h.loadedProject.sourceRoot, h.loadedProject.projectName, acceptedSnapshot, nil, false, h.acceptedDiff))
}

func (h *Handler) markStale(pending *pendingChangeSet) {
	h.loadedStale = true
	if pending != nil {
		pending.stale = true
		pending.review = nil
	}
}

func (h *Handler) recordCandidateValidation(pending *pendingChangeSet, err error) {
	pending.candidate = nil
	pending.validationCode = "change_invalid"
	pending.validationItem = ""
	pending.validationRelationshipPosition = 0
	pending.validationRelationshipField = ""
	var componentError *architecture.ComponentValidationError
	if errors.As(err, &componentError) {
		pending.validationItem = componentError.ComponentID
		pending.validationRelationshipPosition = componentError.RelationshipPosition
		pending.validationRelationshipField = componentError.RelationshipField
	}
	switch {
	case errors.Is(err, architecture.ErrTitleRequired):
		pending.validationCode = "title_required"
	case errors.Is(err, architecture.ErrTitleOneLine):
		pending.validationCode = "title_one_line"
	case errors.Is(err, architecture.ErrRelationshipLabelRequired):
		pending.validationCode = "relationship_label_required"
	case errors.Is(err, architecture.ErrRelationshipTargetRequired):
		pending.validationCode = "relationship_target_required"
	case !errors.Is(err, architecture.ErrInvalid):
		pending.validationCode = "change_unavailable"
	}
}

func (h *Handler) decodeArchitectureAction(response http.ResponseWriter, request *http.Request) (openProjectRequest, bool) {
	if request.Header.Get("Origin") != h.expectedOrigin {
		writeJSON(response, http.StatusForbidden, errorResponse{Code: errorOriginMismatch})
		return openProjectRequest{}, false
	}
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var payload openProjectRequest
	if err := decoder.Decode(&payload); err != nil || ensureJSONEnd(decoder) != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorLookupFailed})
		return openProjectRequest{}, false
	}
	return payload, true
}

func (h *Handler) decodeAcceptChanges(response http.ResponseWriter, request *http.Request) (acceptChangesRequest, bool) {
	if request.Header.Get("Origin") != h.expectedOrigin {
		writeJSON(response, http.StatusForbidden, errorResponse{Code: errorOriginMismatch})
		return acceptChangesRequest{}, false
	}
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var payload acceptChangesRequest
	if err := decoder.Decode(&payload); err != nil || ensureJSONEnd(decoder) != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorLookupFailed})
		return acceptChangesRequest{}, false
	}
	return payload, true
}

func writeArchitectureLoadError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, architecture.ErrUnavailable):
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureUnavailable})
	case errors.Is(err, architecture.ErrUnsupported):
		writeJSON(response, http.StatusUnprocessableEntity, errorResponse{Code: errorArchitectureUnsupported})
	default:
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureInvalid})
	}
}

func writeProjectError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, projects.ErrPathRequired):
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorPathRequired})
	case errors.Is(err, projects.ErrPathRelative):
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorPathRelative})
	case errors.Is(err, projects.ErrPathMissing):
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorPathMissing})
	case errors.Is(err, projects.ErrPathNotDir):
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorPathNotDir})
	default:
		writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorLookupFailed})
	}
}

func (h *Handler) staticFiles() http.Handler {
	files := http.FileServer(http.Dir(h.uiDirectory))
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/" {
			indexPath := filepath.Join(h.uiDirectory, "index.html")
			if _, err := os.Stat(indexPath); err != nil {
				http.Error(response, "WorkBraid browser UI is not built", http.StatusServiceUnavailable)
				return
			}
		}
		files.ServeHTTP(response, request)
	})
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("unexpected extra JSON value")
		}
		return err
	}
	return nil
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}
