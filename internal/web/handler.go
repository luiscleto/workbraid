package web

import (
	"context"
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

	"workbraid/internal/architecture"
)

const (
	maxRequestBody                 = 64 << 10
	architectureTransitionTimeout  = 30 * time.Second
	architectureObservationTimeout = 10 * time.Second
)

type Handler struct {
	expectedOrigin string
	uiDirectory    string
	architecture   *architecture.Manager
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
	// beforeRefreshCatalogCheck is a narrow test seam immediately before the
	// other-store slug conflict scan. Production never sets it.
	beforeRefreshCatalogCheck func()
	// beforeAgentResultCapture is a focused test seam between a shared locked
	// operation and its immutable agent result/context capture. Production
	// never sets it.
	beforeAgentResultCapture func()
	// candidateConstructionFailure is a focused test seam for operational
	// ConstructCandidate failure classification. Production never sets it.
	candidateConstructionFailure func() error
}

type loadedProject struct {
	storeID          string
	projectSlug      string
	projectName      string
	validatedCurrent *architecture.Snapshot
}

type pendingChangeSet struct {
	storeID                        string
	baseRevision                   string
	baseSnapshot                   architecture.Snapshot
	changes                        []architecture.ComponentChange
	newComponentHomes              []architecture.NewComponentHome
	detailDiagrams                 []architecture.DetailDiagramChange
	diagramTitles                  []architecture.DiagramTitleChange
	homeMoves                      []architecture.ComponentHomeMove
	references                     []architecture.ReferenceAppearanceChange
	candidate                      *architecture.Candidate
	generation                     uint64
	review                         *reviewBinding
	reviewBlocker                  string
	validationCode                 string
	validationItem                 string
	validationRelationshipPosition int
	validationRelationshipField    string
	validationDiagram              string
	validationDiagramField         string
	stale                          bool
}

type reviewBinding struct {
	baseRevision  string
	candidateTree string
	generation    uint64
	diff          string
	candidate     architecture.Candidate
}

func NewHandler(expectedOrigin, uiDirectory, dataDirectory string) http.Handler {
	_, mux := newHandler(expectedOrigin, uiDirectory, dataDirectory)
	return mux
}

func newHandler(expectedOrigin, uiDirectory, dataDirectory string) (*Handler, http.Handler) {
	handler := &Handler{
		expectedOrigin: expectedOrigin,
		uiDirectory:    uiDirectory,
		architecture:   architecture.NewManager(dataDirectory),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/projects", handler.projectCatalog)
	mux.HandleFunc("POST /api/projects/create", handler.createProject)
	mux.HandleFunc("POST /api/projects/open", handler.openProject)
	mux.HandleFunc("POST /api/architecture/components/add", handler.addComponent)
	mux.HandleFunc("POST /api/architecture/components/edit", handler.editComponent)
	mux.HandleFunc("POST /api/architecture/diagrams/detail", handler.createDetailDiagram)
	mux.HandleFunc("POST /api/architecture/diagrams/title", handler.editDiagramTitle)
	mux.HandleFunc("POST /api/architecture/components/move-home", handler.moveComponentHome)
	mux.HandleFunc("POST /api/architecture/diagrams/show-component", handler.showComponentHere)
	mux.HandleFunc("POST /api/architecture/diagrams/stop-showing-component", handler.stopShowingHere)
	mux.HandleFunc("POST /api/architecture/review", handler.reviewChanges)
	mux.HandleFunc("POST /api/architecture/accept", handler.acceptChanges)
	mux.HandleFunc("POST /api/architecture/discard", handler.discardChanges)
	mux.HandleFunc("POST /api/architecture/refresh", handler.refreshArchitecture)
	mux.HandleFunc("POST /api/projects/leave", handler.leaveProject)
	handler.registerAgentRoutes(mux)
	mux.Handle("/", handler.staticFiles())
	return handler, mux
}

type openProjectRequest struct {
	ProjectSlug string `json:"project_slug"`
}

type architectureActionRequest struct {
	ProjectSlug               string  `json:"project_slug"`
	StoreID                   string  `json:"store_id"`
	ExpectedRevision          string  `json:"expected_revision,omitempty"`
	ExpectedGeneration        *uint64 `json:"expected_pending_generation,omitempty"`
	PendingGenerationObserved bool    `json:"pending_generation_observed,omitempty"`
}

type acceptChangesRequest struct {
	ProjectSlug   string `json:"project_slug"`
	StoreID       string `json:"store_id"`
	BaseRevision  string `json:"base_revision"`
	CandidateTree string `json:"candidate_tree"`
	Generation    uint64 `json:"generation"`
}

type errorResponse struct {
	Code string `json:"code"`
}

const (
	errorNameRequired            = "name_required"
	errorProjectNotFound         = "project_not_found"
	errorCatalogConflict         = "catalog_conflict"
	errorCatalogUnavailable      = "catalog_unavailable"
	errorOriginMismatch          = "origin_mismatch"
	errorLookupFailed            = "lookup_failed"
	errorProjectCreateFailed     = "project_create_failed"
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
	errorHomeMoveUnavailable     = "home_move_unavailable"
)

type catalogProjectResponse struct {
	Name        string `json:"name,omitempty"`
	Slug        string `json:"slug,omitempty"`
	Revision    string `json:"revision,omitempty"`
	StoreID     string `json:"store_id,omitempty"`
	Unavailable bool   `json:"unavailable,omitempty"`
	Conflict    bool   `json:"conflict,omitempty"`
}

type catalogResponse struct {
	Projects []catalogProjectResponse `json:"projects"`
}

func (h *Handler) projectCatalog(response http.ResponseWriter, request *http.Request) {
	projects, err := h.architecture.Catalog(request.Context())
	if err != nil {
		writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorCatalogUnavailable})
		return
	}
	values := make([]catalogProjectResponse, len(projects))
	for index, project := range projects {
		values[index] = catalogProjectResponse{Name: project.Name, Slug: project.Slug, Revision: project.Revision, StoreID: project.StoreID, Unavailable: project.Unavailable, Conflict: project.Conflict}
	}
	writeJSON(response, http.StatusOK, catalogResponse{Projects: values})
}

func (h *Handler) createProject(response http.ResponseWriter, request *http.Request) {
	if request.Header.Get("Origin") != h.expectedOrigin {
		writeJSON(response, http.StatusForbidden, errorResponse{Code: errorOriginMismatch})
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var payload struct {
		Name string `json:"name"`
	}
	if decoder.Decode(&payload) != nil || ensureJSONEnd(decoder) != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorLookupFailed})
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorNameRequired})
		return
	}
	h.stateMutex.Lock()
	result, code := h.createProjectLocked(request.Context(), payload.Name)
	h.stateMutex.Unlock()
	if code == errorPendingBlocksSwitch {
		writeJSON(response, http.StatusConflict, result)
		return
	}
	if code != "" {
		writeJSON(response, http.StatusInternalServerError, errorResponse{Code: code})
		return
	}
	writeJSON(response, http.StatusCreated, result)
}

func (h *Handler) createProjectLocked(ctx context.Context, name string) (architectureResponse, string) {
	if h.pending != nil {
		result := h.currentArchitectureResponseLocked()
		result.ActionError = errorPendingBlocksSwitch
		return result, errorPendingBlocksSwitch
	}
	snapshot, err := h.architecture.CreateProject(ctx, name)
	if err != nil {
		return architectureResponse{}, errorProjectCreateFailed
	}
	h.publishSnapshotLocked(snapshot)
	return h.currentArchitectureResponseLocked(), ""
}

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

	h.stateMutex.Lock()
	result, code := h.openProjectLocked(request.Context(), payload.ProjectSlug)
	h.stateMutex.Unlock()
	if code != "" {
		status := http.StatusConflict
		if code == errorProjectNotFound {
			status = http.StatusNotFound
		}
		if result.StoreID != "" {
			writeJSON(response, status, result)
		} else {
			writeJSON(response, status, errorResponse{Code: code})
		}
		return
	}
	writeJSON(response, http.StatusOK, result)
}

func (h *Handler) openProjectLocked(ctx context.Context, slug string) (architectureResponse, string) {
	snapshot, err := h.architecture.OpenProject(ctx, slug)
	if errors.Is(err, architecture.ErrProjectNotFound) {
		return architectureResponse{}, errorProjectNotFound
	}
	if errors.Is(err, architecture.ErrCatalogConflict) {
		return architectureResponse{}, errorCatalogConflict
	}
	if err != nil {
		return architectureResponse{}, errorCatalogUnavailable
	}
	if h.pending != nil && h.loadedProject != nil && h.loadedProject.storeID != snapshot.StoreID() {
		result := h.currentArchitectureResponseLocked()
		result.ActionError = errorPendingBlocksSwitch
		return result, errorPendingBlocksSwitch
	}
	stalePending := h.publishSnapshotLocked(snapshot)
	result := h.currentArchitectureResponseLocked()
	if stalePending {
		result.ActionError = errorArchitectureStale
	}
	return result, ""
}

type architectureResponse struct {
	ProjectSlug          string                              `json:"project_slug"`
	StoreID              string                              `json:"store_id"`
	ProjectName          string                              `json:"project_name"`
	State                string                              `json:"state"`
	Revision             string                              `json:"revision"`
	FormatVersion        int                                 `json:"format_version"`
	ComponentCount       int                                 `json:"component_count"`
	ComponentTitles      []string                            `json:"component_titles"`
	Components           []componentResponse                 `json:"components"`
	RootDiagramID        string                              `json:"root_diagram_id,omitempty"`
	Diagrams             []diagramResponse                   `json:"diagrams,omitempty"`
	HomeMoveDestinations []componentHomeDestinationsResponse `json:"home_move_destinations,omitempty"`
	ReferenceChoices     []referenceChoiceResponse           `json:"reference_choices,omitempty"`
	Changes              *changesResponse                    `json:"changes,omitempty"`
	Stale                bool                                `json:"stale,omitempty"`
	ParentDiff           string                              `json:"parent_diff,omitempty"`
	ActionError          string                              `json:"action_error,omitempty"`
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
	BaseRevision                   string                                   `json:"base_revision"`
	Generation                     uint64                                   `json:"generation"`
	Components                     []pendingComponentResponse               `json:"components"`
	RelationshipTargets            []relationshipTargetResponse             `json:"relationship_targets"`
	Valid                          bool                                     `json:"valid"`
	ValidationCode                 string                                   `json:"validation_code,omitempty"`
	ValidationItem                 string                                   `json:"validation_item,omitempty"`
	ValidationRelationshipPosition int                                      `json:"validation_relationship_position,omitempty"`
	ValidationRelationshipField    string                                   `json:"validation_relationship_field,omitempty"`
	ValidationDiagram              string                                   `json:"validation_diagram,omitempty"`
	ValidationDiagramField         string                                   `json:"validation_diagram_field,omitempty"`
	DetailDiagrams                 []architecture.DetailDiagramChange       `json:"detail_diagrams,omitempty"`
	DiagramTitles                  []architecture.DiagramTitleChange        `json:"diagram_titles,omitempty"`
	HomeMoves                      []architecture.ComponentHomeMove         `json:"home_moves,omitempty"`
	References                     []architecture.ReferenceAppearanceChange `json:"references,omitempty"`
	DiagramOptions                 []diagramAuthoringOptionResponse         `json:"diagram_options,omitempty"`
	Candidate                      *snapshotProjectionResponse              `json:"candidate,omitempty"`
	Review                         *reviewResponse                          `json:"review,omitempty"`
	ReviewBlocker                  string                                   `json:"review_blocker,omitempty"`
	Stale                          bool                                     `json:"stale,omitempty"`
}

type diagramAuthoringOptionResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Context string `json:"context,omitempty"`
}

type referenceChoiceResponse struct {
	DiagramID   string `json:"diagram_id"`
	ComponentID string `json:"component_id"`
	Title       string `json:"title"`
	Context     string `json:"context,omitempty"`
	HomeDiagram string `json:"home_diagram"`
}

type componentHomeDestinationsResponse struct {
	ComponentID   string   `json:"component_id"`
	CurrentHomeID string   `json:"current_home_id"`
	DiagramIDs    []string `json:"diagram_ids"`
}

type diagramAuthoringOptionFact struct {
	diagramAuthoringOptionResponse
	anchorComponentID string
	filename          string
}

type diagramAuthoringComponentFact struct {
	title    string
	filename string
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

func responseForSnapshot(snapshot architecture.Snapshot, pending *pendingChangeSet, stale bool, parentDiff string) architectureResponse {
	projection := projectSnapshot(snapshot, "")
	state := "ready"
	if projection.ComponentCount == 0 {
		state = "empty"
	}
	result := architectureResponse{
		ProjectSlug:          snapshot.ProjectSlug(),
		ProjectName:          snapshot.ProjectName(),
		StoreID:              snapshot.StoreID(),
		State:                state,
		Revision:             projection.Revision,
		FormatVersion:        projection.FormatVersion,
		ComponentCount:       projection.ComponentCount,
		ComponentTitles:      projection.ComponentTitles,
		Components:           projection.Components,
		RootDiagramID:        projection.RootDiagramID,
		Diagrams:             projection.Diagrams,
		HomeMoveDestinations: componentHomeDestinations(snapshot),
		ReferenceChoices:     referenceChoices(snapshot),
		Stale:                stale,
		ParentDiff:           parentDiff,
	}
	if pending != nil && pending.storeID == snapshot.StoreID() {
		result.HomeMoveDestinations = nil
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
			BaseRevision:                   pending.baseRevision,
			Generation:                     pending.generation,
			Components:                     changes,
			RelationshipTargets:            relationshipTargets(pendingAccepted, pending.changes),
			Valid:                          pending.candidate != nil,
			ValidationCode:                 pending.validationCode,
			ValidationItem:                 pending.validationItem,
			ValidationRelationshipPosition: pending.validationRelationshipPosition,
			ValidationRelationshipField:    pending.validationRelationshipField,
			ValidationDiagram:              pending.validationDiagram,
			ValidationDiagramField:         pending.validationDiagramField,
			DetailDiagrams:                 append([]architecture.DetailDiagramChange(nil), pending.detailDiagrams...),
			DiagramTitles:                  append([]architecture.DiagramTitleChange(nil), pending.diagramTitles...),
			HomeMoves:                      append([]architecture.ComponentHomeMove(nil), pending.homeMoves...),
			References:                     append([]architecture.ReferenceAppearanceChange(nil), pending.references...),
			DiagramOptions:                 pendingDiagramAuthoringOptions(pending),
			ReviewBlocker:                  pending.reviewBlocker,
			Stale:                          pending.stale,
		}
		if pending.candidate != nil {
			candidateProjection := projectSnapshot(pending.candidate.Snapshot(), "")
			result.Changes.Candidate = &candidateProjection
			result.HomeMoveDestinations = componentHomeDestinations(pending.candidate.Snapshot())
			result.ReferenceChoices = referenceChoices(pending.candidate.Snapshot())
		}
		if pending.candidate == nil {
			// Reference authoring resolves against the complete candidate. An
			// invalid pending set has no coherent reference-choice authority, so
			// never fall back to the accepted snapshot here.
			result.ReferenceChoices = nil
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

// responseForLoadedProjectLocked keeps the route locator tied to the current
// canonical accepted manifest even when a stale pending set requires the
// projection itself to remain the exact older base snapshot.
func (h *Handler) responseForLoadedProjectLocked(snapshot architecture.Snapshot, pending *pendingChangeSet, stale bool, parentDiff string) architectureResponse {
	result := responseForSnapshot(snapshot, pending, stale, parentDiff)
	if h.loadedProject != nil && h.loadedProject.storeID == snapshot.StoreID() {
		result.ProjectSlug = h.loadedProject.projectSlug
	}
	return result
}

func referenceChoices(snapshot architecture.Snapshot) []referenceChoiceResponse {
	components := snapshot.AuthoringComponents()
	diagrams := snapshot.DiagramProjections()
	titleCounts := make(map[string]int, len(components))
	for _, component := range components {
		titleCounts[component.Title]++
	}
	homeTitles := make(map[string]string, len(diagrams))
	for _, diagram := range diagrams {
		homeTitles[diagram.ID] = diagram.Title
	}
	choices := make([]referenceChoiceResponse, 0)
	for _, diagram := range diagrams {
		for _, component := range components {
			if _, appears := snapshot.ComponentAppearanceRole(diagram.ID, component.ID); appears {
				continue
			}
			homeID, _, hasHome := snapshot.ComponentHome(component.ID)
			if !hasHome || homeID == diagram.ID {
				continue
			}
			choice := referenceChoiceResponse{DiagramID: diagram.ID, ComponentID: component.ID, Title: component.Title, HomeDiagram: homeTitles[homeID]}
			if titleCounts[component.Title] > 1 {
				choice.Context = component.Filename
			}
			choices = append(choices, choice)
		}
	}
	return choices
}

func componentHomeDestinations(snapshot architecture.Snapshot) []componentHomeDestinationsResponse {
	if snapshot.FormatVersion() != 2 {
		return nil
	}
	components := snapshot.AuthoringComponents()
	values := make([]componentHomeDestinationsResponse, 0, len(components))
	for _, component := range components {
		currentHomeID, _, hasHome := snapshot.ComponentHome(component.ID)
		if !hasHome {
			continue
		}
		diagramIDs := snapshot.ComponentHomeDestinationDiagramIDs(component.ID)
		values = append(values, componentHomeDestinationsResponse{ComponentID: component.ID, CurrentHomeID: currentHomeID, DiagramIDs: diagramIDs})
	}
	return values
}

func pendingDiagramAuthoringOptions(pending *pendingChangeSet) []diagramAuthoringOptionResponse {
	projection := projectSnapshot(pending.baseSnapshot, "")
	titles := make(map[string]string, len(pending.diagramTitles))
	for _, change := range pending.diagramTitles {
		titles[change.DiagramID] = change.Title
	}
	facts := make([]diagramAuthoringOptionFact, 0, len(projection.Diagrams)+len(pending.detailDiagrams))
	for _, diagram := range projection.Diagrams {
		title := diagram.Title
		if changed, exists := titles[diagram.ID]; exists {
			title = changed
		}
		facts = append(facts, diagramAuthoringOptionFact{
			diagramAuthoringOptionResponse: diagramAuthoringOptionResponse{ID: diagram.ID, Title: title},
			anchorComponentID:              diagram.ParentAnchorComponentID,
			filename:                       filepath.Base(diagram.Filename),
		})
	}
	components := make(map[string]diagramAuthoringComponentFact, len(projection.Components)+len(pending.changes))
	for _, component := range projection.Components {
		components[component.ID] = diagramAuthoringComponentFact{title: component.Title, filename: filepath.Base(component.Filename)}
	}
	for _, change := range pending.changes {
		components[change.ID] = diagramAuthoringComponentFact{title: change.Title, filename: filepath.Base(change.Path)}
	}
	for _, diagram := range pending.detailDiagrams {
		facts = append(facts, diagramAuthoringOptionFact{
			diagramAuthoringOptionResponse: diagramAuthoringOptionResponse{ID: diagram.ID, Title: diagram.Title},
			anchorComponentID:              diagram.AnchorComponentID,
			filename:                       filepath.Base(diagram.Path),
		})
	}

	titleCounts := make(map[string]int, len(facts))
	for _, fact := range facts {
		titleCounts[fact.Title]++
	}
	for index := range facts {
		if titleCounts[facts[index].Title] < 2 {
			continue
		}
		if facts[index].anchorComponentID == "" {
			facts[index].Context = "Main diagram"
			continue
		}
		anchorTitle := strings.TrimSpace(components[facts[index].anchorComponentID].title)
		if anchorTitle == "" {
			facts[index].Context = "Detail diagram"
		} else {
			facts[index].Context = "Detail for " + anchorTitle
		}
	}
	disambiguateDiagramOptionFacts(facts, components)

	options := make([]diagramAuthoringOptionResponse, len(facts))
	for index, fact := range facts {
		options[index] = fact.diagramAuthoringOptionResponse
	}
	return options
}

func disambiguateDiagramOptionFacts(facts []diagramAuthoringOptionFact, components map[string]diagramAuthoringComponentFact) {
	appendToCollidingContexts(facts, func(fact diagramAuthoringOptionFact) string {
		return components[fact.anchorComponentID].filename
	})
	appendToCollidingContexts(facts, func(fact diagramAuthoringOptionFact) string {
		return fact.filename
	})
	appendToCollidingContexts(facts, func(fact diagramAuthoringOptionFact) string {
		if len(fact.ID) <= 8 {
			return fact.ID
		}
		return fact.ID[:8]
	})
}

func appendToCollidingContexts(facts []diagramAuthoringOptionFact, suffix func(diagramAuthoringOptionFact) string) {
	counts := make(map[string]int, len(facts))
	for _, fact := range facts {
		if fact.Context != "" {
			counts[fact.Title+"\x00"+fact.Context]++
		}
	}
	for index := range facts {
		if facts[index].Context == "" || counts[facts[index].Title+"\x00"+facts[index].Context] < 2 {
			continue
		}
		if value := strings.TrimSpace(suffix(facts[index])); value != "" {
			facts[index].Context += " — " + value
		}
	}
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

func (h *Handler) publishSnapshotLocked(snapshot architecture.Snapshot) bool {
	if h.pending != nil && h.pending.storeID == snapshot.StoreID() && h.pending.baseRevision != snapshot.Revision() &&
		h.pending.baseSnapshot.StoreID() == h.pending.storeID && h.pending.baseSnapshot.Revision() == h.pending.baseRevision {
		h.pending.stale = true
		h.pending.review = nil
		base := h.pending.baseSnapshot
		h.loadedSnapshot = &base
		current := snapshot
		h.loadedProject = &loadedProject{
			storeID: snapshot.StoreID(), projectName: snapshot.ProjectName(), projectSlug: snapshot.ProjectSlug(), validatedCurrent: &current,
		}
		h.loadedStale = true
		h.acceptedDiff = ""
		return true
	}
	keepAcceptedDiff := h.loadedSnapshot != nil && h.loadedSnapshot.StoreID() == snapshot.StoreID() && h.loadedSnapshot.Revision() == snapshot.Revision()
	h.loadedSnapshot = &snapshot
	h.loadedProject = &loadedProject{storeID: snapshot.StoreID(), projectName: snapshot.ProjectName(), projectSlug: snapshot.ProjectSlug()}
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
	return h.responseForLoadedProjectLocked(*h.loadedSnapshot, h.pending, h.loadedStale, h.acceptedDiff)
}

func (h *Handler) clearLoadedProjectLocked() {
	h.loadedSnapshot = nil
	h.loadedProject = nil
	h.loadedStale = false
	h.acceptedDiff = ""
}

func (h *Handler) matchesLoadedProjectLocked(projectSlug, storeID string) bool {
	return h.loadedSnapshot != nil && h.loadedProject != nil && storeID != "" &&
		storeID == h.loadedProject.storeID && projectSlug == h.loadedProject.projectSlug
}

func (h *Handler) matchesExpectedPendingGenerationLocked(expected *uint64, observed bool) bool {
	if !observed {
		return false
	}
	if expected == nil {
		return h.pending == nil
	}
	return h.pending != nil && h.pending.generation == *expected
}

func (h *Handler) refreshArchitecture(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeArchitectureAction(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	result, code, status := h.refreshArchitectureLocked(request.Context(), payload)
	h.stateMutex.Unlock()
	if code != "" && result.StoreID == "" {
		writeJSON(response, status, errorResponse{Code: code})
		return
	}
	writeJSON(response, status, result)
}

func (h *Handler) refreshArchitectureLocked(ctx context.Context, payload architectureActionRequest) (architectureResponse, string, int) {
	if !h.matchesLoadedProjectLocked(payload.ProjectSlug, payload.StoreID) {
		return architectureResponse{}, errorArchitectureNotOpen, http.StatusConflict
	}
	if payload.ExpectedRevision == "" || h.loadedSnapshot == nil {
		return architectureResponse{}, errorChangesElsewhere, http.StatusConflict
	}
	if payload.ExpectedRevision != h.loadedSnapshot.Revision() {
		return architectureResponse{}, errorArchitectureStale, http.StatusConflict
	}

	loaded := *h.loadedSnapshot
	observationContext, cancelObservation := context.WithTimeout(ctx, architectureObservationTimeout)
	observed, present, err := h.architecture.AcceptedRevision(observationContext, loaded)
	cancelObservation()
	if err != nil {
		return h.refreshResultLocked(errorRefreshFailed, http.StatusServiceUnavailable)
	}
	if !present {
		h.markKnownNonCurrentLocked()
		return h.refreshResultLocked(errorRefreshUnavailable, http.StatusConflict)
	}
	if observed == loaded.Revision() && !h.loadedStale {
		return h.currentArchitectureResponseLocked(), "", http.StatusOK
	}

	loadContext, cancelLoad := context.WithTimeout(ctx, architectureTransitionTimeout)
	replacement, loadErr := h.architecture.LoadRevision(loadContext, loaded, observed)
	cancelLoad()
	if loadErr != nil {
		h.markKnownNonCurrentLocked()
		switch {
		case errors.Is(loadErr, architecture.ErrUnsupported):
			return h.refreshResultLocked(errorRefreshUnsupported, http.StatusUnprocessableEntity)
		case errors.Is(loadErr, architecture.ErrUnavailable):
			return h.refreshResultLocked(errorRefreshUnavailable, http.StatusConflict)
		default:
			return h.refreshResultLocked(errorRefreshInvalid, http.StatusConflict)
		}
	}
	if h.beforeRefreshCatalogCheck != nil {
		h.beforeRefreshCatalogCheck()
	}
	candidateSlugAvailable, candidateSlugErr := h.architecture.CatalogSlugAvailable(ctx, replacement.StoreID(), replacement.ProjectSlug())
	retainedSlugAvailable, retainedSlugErr := candidateSlugAvailable, candidateSlugErr
	if loaded.ProjectSlug() != replacement.ProjectSlug() {
		retainedSlugAvailable, retainedSlugErr = h.architecture.CatalogSlugAvailable(ctx, loaded.StoreID(), loaded.ProjectSlug())
	}
	if h.beforeRefreshReobserve != nil {
		h.beforeRefreshReobserve(observed)
	}

	finalContext, cancelFinal := context.WithTimeout(ctx, architectureObservationTimeout)
	finalRevision, finalPresent, finalErr := h.architecture.AcceptedRevision(finalContext, loaded)
	cancelFinal()
	if finalErr != nil {
		return h.refreshResultLocked(errorRefreshFailed, http.StatusServiceUnavailable)
	}
	if !finalPresent {
		h.markKnownNonCurrentLocked()
		return h.refreshResultLocked(errorRefreshUnavailable, http.StatusConflict)
	}
	if finalRevision != observed {
		if finalRevision == loaded.Revision() {
			// Authority returned to the retained, already validated snapshot.
			// A pending set already known stale stays stale until discarded.
			if retainedSlugErr != nil {
				return h.refreshResultLocked(errorRefreshFailed, http.StatusServiceUnavailable)
			}
			if !retainedSlugAvailable {
				h.markKnownNonCurrentLocked()
				return h.refreshResultLocked(errorCatalogConflict, http.StatusConflict)
			}
			h.loadedProject.projectName = loaded.ProjectName()
			h.loadedProject.projectSlug = loaded.ProjectSlug()
			h.loadedProject.validatedCurrent = nil
			h.loadedStale = false
			return h.currentArchitectureResponseLocked(), "", http.StatusOK
		}
		h.markKnownNonCurrentLocked()
		return h.refreshResultLocked(errorRefreshChanged, http.StatusConflict)
	}
	if candidateSlugErr != nil {
		return h.refreshResultLocked(errorRefreshFailed, http.StatusServiceUnavailable)
	}
	if !candidateSlugAvailable {
		h.markKnownNonCurrentLocked()
		return h.refreshResultLocked(errorCatalogConflict, http.StatusConflict)
	}
	h.loadedSnapshot = &replacement
	h.loadedProject.projectName = replacement.ProjectName()
	h.loadedProject.projectSlug = replacement.ProjectSlug()
	h.loadedProject.validatedCurrent = nil
	h.loadedStale = false
	h.acceptedDiff = ""
	if h.pending != nil && (h.pending.storeID != replacement.StoreID() || h.pending.baseRevision != replacement.Revision()) {
		h.markPendingStaleLocked()
	}
	return h.currentArchitectureResponseLocked(), "", http.StatusOK
}

func (h *Handler) refreshResultLocked(actionError string, status int) (architectureResponse, string, int) {
	result := h.currentArchitectureResponseLocked()
	result.ActionError = actionError
	return result, actionError, status
}

func (h *Handler) markKnownNonCurrentLocked() {
	h.loadedStale = true
	if h.loadedProject != nil {
		h.loadedProject.validatedCurrent = nil
	}
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
	result, code := h.discardChangesLocked(payload)
	h.stateMutex.Unlock()
	if code != "" {
		writeJSON(response, http.StatusConflict, errorResponse{Code: code})
		return
	}
	writeJSON(response, http.StatusOK, result)
}

func (h *Handler) discardChangesLocked(payload architectureActionRequest) (architectureResponse, string) {
	if !h.matchesLoadedProjectLocked(payload.ProjectSlug, payload.StoreID) {
		return architectureResponse{}, errorArchitectureNotOpen
	}
	if !h.matchesExpectedPendingGenerationLocked(payload.ExpectedGeneration, payload.PendingGenerationObserved) {
		return architectureResponse{}, errorChangesElsewhere
	}
	h.pending = nil
	if h.loadedStale && h.loadedProject.validatedCurrent != nil {
		current := *h.loadedProject.validatedCurrent
		h.loadedSnapshot = &current
		h.loadedProject.projectName = current.ProjectName()
		h.loadedProject.projectSlug = current.ProjectSlug()
		h.loadedProject.validatedCurrent = nil
		h.loadedStale = false
		h.acceptedDiff = ""
	}
	return h.currentArchitectureResponseLocked(), ""
}

func (h *Handler) leaveProject(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeArchitectureAction(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	result, code := h.leaveProjectLocked(payload.ProjectSlug, payload.StoreID)
	h.stateMutex.Unlock()
	if code != "" {
		if result.StoreID != "" {
			writeJSON(response, http.StatusConflict, result)
		} else {
			writeJSON(response, http.StatusConflict, errorResponse{Code: code})
		}
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (h *Handler) leaveProjectLocked(projectSlug, storeID string) (architectureResponse, string) {
	if !h.matchesLoadedProjectLocked(projectSlug, storeID) {
		return architectureResponse{}, errorArchitectureNotOpen
	}
	if h.pending != nil {
		result := h.currentArchitectureResponseLocked()
		result.ActionError = errorPendingBlocksSwitch
		return result, errorPendingBlocksSwitch
	}
	h.clearLoadedProjectLocked()
	return architectureResponse{}, ""
}

type componentMutationRequest struct {
	ProjectSlug               string                 `json:"project_slug"`
	StoreID                   string                 `json:"store_id"`
	ExpectedRevision          string                 `json:"expected_revision,omitempty"`
	ExpectedGeneration        *uint64                `json:"expected_pending_generation,omitempty"`
	PendingGenerationObserved bool                   `json:"pending_generation_observed,omitempty"`
	ComponentID               string                 `json:"component_id,omitempty"`
	Title                     string                 `json:"title,omitempty"`
	Description               string                 `json:"description,omitempty"`
	TitleChanged              bool                   `json:"title_changed,omitempty"`
	DescriptionChanged        bool                   `json:"description_changed,omitempty"`
	Relationships             []relationshipResponse `json:"relationships,omitempty"`
	RelationshipsChanged      bool                   `json:"relationships_changed,omitempty"`
	DiagramID                 string                 `json:"diagram_id,omitempty"`
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
	if !h.matchesLoadedProjectLocked(payload.ProjectSlug, payload.StoreID) {
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
	if !h.matchesExpectedPendingGenerationLocked(payload.ExpectedGeneration, payload.PendingGenerationObserved) {
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
		if !pendingHasDiagram(snapshot, h.pending, homeDiagramID) {
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
		var found bool
		change, changeIndex, found = componentChangeLocked(snapshot, h.pending, payload.ComponentID)
		if !found {
			writeJSON(response, http.StatusNotFound, errorResponse{Code: errorComponentNotFound})
			return
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
	h.pending = h.keepComponentChangeLocked(request.Context(), snapshot, h.pending, change, changeIndex)
	if h.pending.validationCode == "change_unavailable" {
		writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorChangeFailed})
		return
	}
	writeJSON(response, http.StatusOK, h.responseForLoadedProjectLocked(snapshot, h.pending, h.loadedStale, h.acceptedDiff))
}

func (h *Handler) ensurePendingLocked(snapshot architecture.Snapshot, pending *pendingChangeSet) *pendingChangeSet {
	if pending != nil {
		return pending
	}
	pending = &pendingChangeSet{storeID: snapshot.StoreID(), baseRevision: snapshot.Revision(), baseSnapshot: snapshot}
	h.pending = pending
	return pending
}

func componentChangeLocked(snapshot architecture.Snapshot, pending *pendingChangeSet, componentID string) (architecture.ComponentChange, int, bool) {
	if pending != nil {
		for index, change := range pending.changes {
			if change.ID == componentID {
				return change, index, true
			}
		}
	}
	change, found := snapshot.ChangeForAcceptedComponent(componentID)
	return change, -1, found
}

func applyComponentChangeLocked(pending *pendingChangeSet, change architecture.ComponentChange, index int) {
	if index >= 0 {
		pending.changes[index] = change
		return
	}
	pending.changes = append(pending.changes, change)
}

func (h *Handler) keepComponentChangeLocked(ctx context.Context, snapshot architecture.Snapshot, pending *pendingChangeSet, change architecture.ComponentChange, index int) *pendingChangeSet {
	pending = h.ensurePendingLocked(snapshot, pending)
	applyComponentChangeLocked(pending, change, index)
	h.rebuildPendingLocked(ctx, snapshot, pending)
	if pending.candidate == nil && pending.validationItem == "" {
		pending.validationItem = change.ID
	}
	return pending
}

func (h *Handler) constructCandidate(ctx context.Context, snapshot architecture.Snapshot, pending *pendingChangeSet) (architecture.Candidate, error) {
	if h.candidateConstructionFailure != nil {
		if err := h.candidateConstructionFailure(); err != nil {
			return architecture.Candidate{}, err
		}
	}
	return h.architecture.ConstructCandidate(ctx, snapshot, pending.changes, architecture.CandidateComposition{
		NewComponentHomes: pending.newComponentHomes,
		DetailDiagrams:    pending.detailDiagrams,
		DiagramTitles:     pending.diagramTitles,
		HomeMoves:         pending.homeMoves,
		References:        pending.references,
	})
}

type diagramMutationRequest struct {
	ProjectSlug               string  `json:"project_slug"`
	StoreID                   string  `json:"store_id"`
	ExpectedRevision          string  `json:"expected_revision"`
	ExpectedGeneration        *uint64 `json:"expected_pending_generation,omitempty"`
	PendingGenerationObserved bool    `json:"pending_generation_observed,omitempty"`
	DiagramID                 string  `json:"diagram_id,omitempty"`
	ComponentID               string  `json:"component_id,omitempty"`
	Title                     string  `json:"title,omitempty"`
}

func (h *Handler) decodeDiagramMutation(response http.ResponseWriter, request *http.Request) (diagramMutationRequest, bool) {
	if request.Header.Get("Origin") != h.expectedOrigin {
		writeJSON(response, http.StatusForbidden, errorResponse{Code: errorOriginMismatch})
		return diagramMutationRequest{}, false
	}
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var payload diagramMutationRequest
	if err := decoder.Decode(&payload); err != nil || ensureJSONEnd(decoder) != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorLookupFailed})
		return diagramMutationRequest{}, false
	}
	return payload, true
}

func (h *Handler) writableV2PendingLocked(response http.ResponseWriter, payload diagramMutationRequest) (architecture.Snapshot, *pendingChangeSet, bool) {
	snapshot, pending, ok := h.writableV2StateLocked(response, payload)
	if !ok {
		return architecture.Snapshot{}, nil, false
	}
	if pending == nil {
		pending = &pendingChangeSet{storeID: snapshot.StoreID(), baseRevision: snapshot.Revision(), baseSnapshot: snapshot}
		h.pending = pending
	}
	return snapshot, pending, true
}

func (h *Handler) writableV2StateLocked(response http.ResponseWriter, payload diagramMutationRequest) (architecture.Snapshot, *pendingChangeSet, bool) {
	if !h.matchesLoadedProjectLocked(payload.ProjectSlug, payload.StoreID) {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureNotOpen})
		return architecture.Snapshot{}, nil, false
	}
	if h.loadedStale {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorArchitectureStale})
		return architecture.Snapshot{}, nil, false
	}
	snapshot := *h.loadedSnapshot
	if payload.ExpectedRevision == "" || payload.ExpectedRevision != snapshot.Revision() {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesElsewhere})
		return architecture.Snapshot{}, nil, false
	}
	if snapshot.FormatVersion() != 2 {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesUnavailable})
		return architecture.Snapshot{}, nil, false
	}
	if h.pending != nil && (h.pending.stale || h.pending.storeID != snapshot.StoreID() || h.pending.baseRevision != snapshot.Revision()) {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesElsewhere})
		return architecture.Snapshot{}, nil, false
	}
	if !h.matchesExpectedPendingGenerationLocked(payload.ExpectedGeneration, payload.PendingGenerationObserved) {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesElsewhere})
		return architecture.Snapshot{}, nil, false
	}
	return snapshot, h.pending, true
}

func pendingHasDiagram(snapshot architecture.Snapshot, pending *pendingChangeSet, id string) bool {
	if snapshot.HasDiagram(id) {
		return true
	}
	for _, addition := range pending.detailDiagrams {
		if addition.ID == id {
			return true
		}
	}
	return false
}

func (h *Handler) rebuildPendingLocked(ctx context.Context, snapshot architecture.Snapshot, pending *pendingChangeSet) {
	pending.generation++
	pending.review = nil
	pending.candidate = nil
	pending.reviewBlocker = ""
	pending.validationCode = ""
	pending.validationItem = ""
	pending.validationRelationshipPosition = 0
	pending.validationRelationshipField = ""
	pending.validationDiagram = ""
	pending.validationDiagramField = ""
	candidate, err := h.constructCandidate(ctx, snapshot, pending)
	if err != nil {
		h.recordCandidateValidation(pending, err)
		return
	}
	pending.candidate = &candidate
}

const (
	changeTargetNotFound    = "target_not_found"
	changeTargetIneligible  = "target_not_eligible"
	changeValidationBlocked = "validation_blocked"
	changeOperationFailed   = "operation_failed"
)

func pendingOperationFailed(pending *pendingChangeSet) bool {
	return pending != nil && pending.validationCode == "change_unavailable"
}

func (h *Handler) createDetailDiagramLocked(ctx context.Context, snapshot architecture.Snapshot, pending *pendingChangeSet, componentID, title string) (*pendingChangeSet, architecture.DetailDiagramChange, string, string) {
	if pending != nil {
		for _, addition := range pending.detailDiagrams {
			if addition.AnchorComponentID == componentID {
				return pending, architecture.DetailDiagramChange{}, "", changeTargetIneligible
			}
		}
	}
	current := snapshot
	if pending != nil && pending.candidate != nil {
		current = pending.candidate.Snapshot()
	}
	homeDiagramID, detailID, home := current.ComponentHome(componentID)
	if !home {
		return pending, architecture.DetailDiagramChange{}, "", changeTargetNotFound
	}
	if detailID != "" {
		return pending, architecture.DetailDiagramChange{}, "", changeTargetIneligible
	}
	pending = h.ensurePendingLocked(snapshot, pending)
	addition := snapshot.NewDetailDiagramChange(pending.detailDiagrams, title, componentID)
	pending.detailDiagrams = append(pending.detailDiagrams, addition)
	h.rebuildPendingLocked(ctx, snapshot, pending)
	if pendingOperationFailed(pending) {
		return pending, addition, homeDiagramID, changeOperationFailed
	}
	return pending, addition, homeDiagramID, ""
}

func (h *Handler) editDiagramTitleLocked(ctx context.Context, snapshot architecture.Snapshot, pending *pendingChangeSet, diagramID, title string) (*pendingChangeSet, string) {
	if !pendingHasDiagram(snapshot, pending, diagramID) {
		return pending, changeTargetNotFound
	}
	pending = h.ensurePendingLocked(snapshot, pending)
	for index := range pending.detailDiagrams {
		if pending.detailDiagrams[index].ID == diagramID {
			pending.detailDiagrams[index].Title = title
			h.rebuildPendingLocked(ctx, snapshot, pending)
			if pendingOperationFailed(pending) {
				return pending, changeOperationFailed
			}
			return pending, ""
		}
	}
	for index := range pending.diagramTitles {
		if pending.diagramTitles[index].DiagramID == diagramID {
			pending.diagramTitles[index].Title = title
			h.rebuildPendingLocked(ctx, snapshot, pending)
			if pendingOperationFailed(pending) {
				return pending, changeOperationFailed
			}
			return pending, ""
		}
	}
	pending.diagramTitles = append(pending.diagramTitles, architecture.DiagramTitleChange{DiagramID: diagramID, Title: title})
	h.rebuildPendingLocked(ctx, snapshot, pending)
	if pendingOperationFailed(pending) {
		return pending, changeOperationFailed
	}
	return pending, ""
}

func (h *Handler) moveComponentHomeLocked(ctx context.Context, snapshot architecture.Snapshot, pending *pendingChangeSet, componentID, diagramID string) (*pendingChangeSet, bool, string) {
	authority := snapshot
	if pending != nil {
		if pending.candidate == nil {
			if pendingOperationFailed(pending) {
				return pending, false, changeOperationFailed
			}
			return pending, false, changeValidationBlocked
		}
		authority = pending.candidate.Snapshot()
	}
	if !authority.HasComponent(componentID) || !authority.HasDiagram(diagramID) {
		return pending, false, changeTargetNotFound
	}
	currentHome, _, hasHome := authority.ComponentHome(componentID)
	if !hasHome {
		return pending, false, changeTargetIneligible
	}
	if currentHome == diagramID {
		return pending, true, ""
	}
	if !containsString(authority.ComponentHomeDestinationDiagramIDs(componentID), diagramID) {
		return pending, false, changeTargetIneligible
	}
	proposed := pendingChangeSet{storeID: snapshot.StoreID(), baseRevision: snapshot.Revision(), baseSnapshot: snapshot}
	if pending != nil {
		proposed = *pending
	}
	proposed.references = referenceChangesWithoutPair(proposed.references, diagramID, componentID)
	setReferenceChange(&proposed, currentHome, componentID, false)
	proposed.homeMoves = homeMovesWithoutComponent(proposed.homeMoves, componentID)
	if initialComponentHome(snapshot, &proposed, componentID) != diagramID {
		proposed.homeMoves = append(proposed.homeMoves, architecture.ComponentHomeMove{ComponentID: componentID, DiagramID: diagramID})
	}
	if pendingChangeSetEmpty(&proposed) {
		h.pending = nil
		return nil, false, ""
	}
	h.rebuildPendingLocked(ctx, snapshot, &proposed)
	if pendingOperationFailed(&proposed) {
		return pending, false, changeOperationFailed
	}
	if proposed.candidate == nil {
		return pending, false, changeTargetIneligible
	}
	h.pending = &proposed
	return h.pending, false, ""
}

func (h *Handler) changeReferenceLocked(ctx context.Context, snapshot architecture.Snapshot, pending *pendingChangeSet, diagramID, componentID string, present bool) (*pendingChangeSet, string) {
	authority := snapshot
	if pending != nil {
		if pending.candidate == nil && !pendingChangeSetEmpty(pending) {
			if pendingOperationFailed(pending) {
				return pending, changeOperationFailed
			}
			return pending, changeValidationBlocked
		}
		if pending.candidate != nil {
			authority = pending.candidate.Snapshot()
		}
	}
	role, appears := authority.ComponentAppearanceRole(diagramID, componentID)
	homeID, _, hasHome := authority.ComponentHome(componentID)
	if !authority.HasDiagram(diagramID) || !authority.HasComponent(componentID) || !hasHome {
		return pending, changeTargetNotFound
	}
	eligible := !appears && homeID != diagramID
	if !present {
		eligible = appears && role == "reference"
	}
	if !eligible {
		return pending, changeTargetIneligible
	}
	pending = h.ensurePendingLocked(snapshot, pending)
	setReferenceChange(pending, diagramID, componentID, present)
	h.rebuildPendingLocked(ctx, snapshot, pending)
	if pendingOperationFailed(pending) {
		return pending, changeOperationFailed
	}
	return pending, ""
}

func (h *Handler) createDetailDiagram(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeDiagramMutation(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	snapshot, pending, ok := h.writableV2StateLocked(response, payload)
	if !ok {
		return
	}
	if _, _, _, operationError := h.createDetailDiagramLocked(request.Context(), snapshot, pending, payload.ComponentID, payload.Title); operationError != "" {
		if operationError == changeOperationFailed {
			writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorChangeFailed})
			return
		}
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangeFailed})
		return
	}
	writeJSON(response, http.StatusOK, h.currentArchitectureResponseLocked())
}

func (h *Handler) editDiagramTitle(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeDiagramMutation(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	snapshot, pending, ok := h.writableV2StateLocked(response, payload)
	if !ok {
		return
	}
	if _, operationError := h.editDiagramTitleLocked(request.Context(), snapshot, pending, payload.DiagramID, payload.Title); operationError != "" {
		if operationError == changeOperationFailed {
			writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorChangeFailed})
			return
		}
		writeJSON(response, http.StatusNotFound, errorResponse{Code: errorChangeFailed})
		return
	}
	writeJSON(response, http.StatusOK, h.currentArchitectureResponseLocked())
}

func (h *Handler) moveComponentHome(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeDiagramMutation(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	snapshot, pending, ok := h.writableV2StateLocked(response, payload)
	if !ok {
		return
	}
	_, _, operationError := h.moveComponentHomeLocked(request.Context(), snapshot, pending, payload.ComponentID, payload.DiagramID)
	if operationError == "" {
		writeJSON(response, http.StatusOK, h.currentArchitectureResponseLocked())
		return
	}
	if operationError == changeOperationFailed {
		writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorChangeFailed})
		return
	}
	code := errorHomeMoveUnavailable
	if operationError == changeValidationBlocked {
		code = errorChangesUnavailable
	}
	writeJSON(response, http.StatusConflict, errorResponse{Code: code})
}

func initialComponentHome(snapshot architecture.Snapshot, pending *pendingChangeSet, componentID string) string {
	if homeID, _, ok := snapshot.ComponentHome(componentID); ok {
		return homeID
	}
	for _, home := range pending.newComponentHomes {
		if home.ComponentID == componentID {
			return home.DiagramID
		}
	}
	return ""
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func homeMovesWithoutComponent(moves []architecture.ComponentHomeMove, componentID string) []architecture.ComponentHomeMove {
	remaining := make([]architecture.ComponentHomeMove, 0, len(moves))
	for _, move := range moves {
		if move.ComponentID != componentID {
			remaining = append(remaining, move)
		}
	}
	return remaining
}

func pendingChangeSetEmpty(pending *pendingChangeSet) bool {
	return len(pending.changes) == 0 && len(pending.newComponentHomes) == 0 &&
		len(pending.detailDiagrams) == 0 && len(pending.diagramTitles) == 0 && len(pending.homeMoves) == 0 && len(pending.references) == 0
}

func referenceChangesWithoutPair(changes []architecture.ReferenceAppearanceChange, diagramID, componentID string) []architecture.ReferenceAppearanceChange {
	result := make([]architecture.ReferenceAppearanceChange, 0, len(changes))
	for _, change := range changes {
		if change.DiagramID != diagramID || change.ComponentID != componentID {
			result = append(result, change)
		}
	}
	return result
}

func setReferenceChange(pending *pendingChangeSet, diagramID, componentID string, present bool) {
	if diagramID == "" {
		return
	}
	pending.references = referenceChangesWithoutPair(pending.references, diagramID, componentID)
	pending.references = append(pending.references, architecture.ReferenceAppearanceChange{DiagramID: diagramID, ComponentID: componentID, Present: present})
}

func (h *Handler) showComponentHere(response http.ResponseWriter, request *http.Request) {
	h.changeReferenceAppearance(response, request, true)
}

func (h *Handler) stopShowingHere(response http.ResponseWriter, request *http.Request) {
	h.changeReferenceAppearance(response, request, false)
}

func (h *Handler) changeReferenceAppearance(response http.ResponseWriter, request *http.Request, present bool) {
	payload, ok := h.decodeDiagramMutation(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	snapshot, pending, ok := h.writableV2StateLocked(response, payload)
	if !ok {
		return
	}
	if _, operationError := h.changeReferenceLocked(request.Context(), snapshot, pending, payload.DiagramID, payload.ComponentID, present); operationError == "" {
		writeJSON(response, http.StatusOK, h.currentArchitectureResponseLocked())
		return
	} else if operationError == changeValidationBlocked {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesUnavailable})
		return
	} else if operationError == changeOperationFailed {
		writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorChangeFailed})
		return
	}
	writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangeFailed})
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
	result, code, status := h.reviewChangesLocked(request.Context(), payload)
	h.stateMutex.Unlock()
	if code != "" && result.StoreID == "" {
		writeJSON(response, status, errorResponse{Code: code})
		return
	}
	writeJSON(response, status, result)
}

func (h *Handler) reviewChangesLocked(ctx context.Context, payload architectureActionRequest) (architectureResponse, string, int) {
	if !h.matchesLoadedProjectLocked(payload.ProjectSlug, payload.StoreID) {
		return architectureResponse{}, errorArchitectureNotOpen, http.StatusConflict
	}
	if h.loadedStale {
		return architectureResponse{}, errorArchitectureStale, http.StatusConflict
	}
	snapshot := *h.loadedSnapshot
	if payload.ExpectedRevision == "" {
		return architectureResponse{}, errorChangesElsewhere, http.StatusConflict
	}
	if payload.ExpectedRevision != snapshot.Revision() {
		return architectureResponse{}, errorArchitectureStale, http.StatusConflict
	}
	if !h.matchesExpectedPendingGenerationLocked(payload.ExpectedGeneration, payload.PendingGenerationObserved) {
		return architectureResponse{}, errorChangesElsewhere, http.StatusConflict
	}
	if h.pending == nil || h.pending.stale || h.pending.storeID != snapshot.StoreID() || h.pending.baseRevision != snapshot.Revision() {
		return architectureResponse{}, errorReviewFailed, http.StatusConflict
	}

	candidate, err := h.constructCandidate(ctx, snapshot, h.pending)
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
		result := h.responseForLoadedProjectLocked(snapshot, h.pending, false, h.acceptedDiff)
		result.ActionError = errorReviewFailed
		return result, errorReviewFailed, status
	}
	diff, err := h.architecture.CandidateDiff(ctx, snapshot, candidate)
	if err != nil {
		result := h.responseForLoadedProjectLocked(snapshot, h.pending, false, h.acceptedDiff)
		result.ActionError = errorReviewFailed
		return result, errorReviewFailed, http.StatusInternalServerError
	}
	h.pending.candidate = &candidate
	h.pending.validationCode = ""
	h.pending.validationItem = ""
	h.pending.validationRelationshipPosition = 0
	h.pending.validationRelationshipField = ""
	h.pending.validationDiagram = ""
	h.pending.validationDiagramField = ""
	h.pending.review = &reviewBinding{
		baseRevision: snapshot.Revision(), candidateTree: candidate.Tree(), generation: h.pending.generation,
		diff: string(diff), candidate: candidate,
	}
	// Build the complete immutable visual presentation while the binding and
	// pending generation are protected by the same concrete state lock. JSON
	// serialization can then proceed without blocking invalidating mutations.
	result := h.responseForLoadedProjectLocked(snapshot, h.pending, false, h.acceptedDiff)
	return result, "", http.StatusOK
}

func (h *Handler) acceptChanges(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeAcceptChanges(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	result, code, status, _ := h.acceptChangesLocked(payload)
	h.stateMutex.Unlock()
	if code != "" && result.StoreID == "" {
		writeJSON(response, status, errorResponse{Code: code})
		return
	}
	writeJSON(response, status, result)
}

func (h *Handler) acceptChangesLocked(payload acceptChangesRequest) (architectureResponse, string, int, bool) {
	if !h.matchesLoadedProjectLocked(payload.ProjectSlug, payload.StoreID) {
		return architectureResponse{}, errorArchitectureNotOpen, http.StatusConflict, false
	}
	if h.loadedStale {
		return architectureResponse{}, errorArchitectureStale, http.StatusConflict, false
	}
	snapshot := *h.loadedSnapshot
	pending := h.pending
	if pending == nil || pending.stale || pending.review == nil {
		return architectureResponse{}, errorReviewFailed, http.StatusConflict, false
	}
	review := pending.review
	if payload.BaseRevision != review.baseRevision || payload.CandidateTree != review.candidateTree || payload.Generation != review.generation {
		result := h.responseForLoadedProjectLocked(snapshot, pending, false, h.acceptedDiff)
		result.ActionError = errorReviewChanged
		if result.Changes != nil {
			// The backend may hold a newer review for another browser. Do not
			// expose it as though this client had inspected it.
			result.Changes.Review = nil
		}
		return result, errorReviewChanged, http.StatusConflict, false
	}
	if pending.storeID != snapshot.StoreID() || pending.baseRevision != snapshot.Revision() ||
		review.baseRevision != pending.baseRevision || review.generation != pending.generation ||
		pending.candidate == nil || review.candidateTree != pending.candidate.Tree() || review.candidateTree != review.candidate.Tree() {
		pending.review = nil
		result := h.responseForLoadedProjectLocked(snapshot, pending, false, h.acceptedDiff)
		result.ActionError = errorReviewChanged
		return result, errorReviewChanged, http.StatusConflict, false
	}
	// Once the human confirms, the local authority transition must reach a
	// classified boundary even if the browser disconnects before the response.
	transitionContext, cancelTransition := context.WithTimeout(context.Background(), architectureTransitionTimeout)
	defer cancelTransition()

	observed, present, err := h.architecture.AcceptedRevision(transitionContext, snapshot)
	if err != nil || !present {
		h.loadedStale = true
		pending.review = nil
		result := h.responseForLoadedProjectLocked(snapshot, pending, true, h.acceptedDiff)
		result.ActionError = errorUpdateUncertain
		return result, errorUpdateUncertain, http.StatusConflict, false
	}
	if observed != review.baseRevision {
		h.markStale(pending)
		result := h.responseForLoadedProjectLocked(snapshot, pending, true, h.acceptedDiff)
		result.ActionError = errorArchitectureStale
		return result, errorArchitectureStale, http.StatusConflict, false
	}

	successor, err := h.architecture.CreateSuccessor(transitionContext, snapshot, review.candidate)
	if err != nil {
		result := h.responseForLoadedProjectLocked(snapshot, pending, false, h.acceptedDiff)
		result.ActionError = errorUpdateFailed
		return result, errorUpdateFailed, http.StatusInternalServerError, false
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
			result := h.responseForLoadedProjectLocked(snapshot, pending, true, h.acceptedDiff)
			result.ActionError = errorUpdateUncertain
			return result, errorUpdateUncertain, http.StatusInternalServerError, false
		}
		if observed != successor && observed != review.baseRevision {
			h.markStale(pending)
			result := h.responseForLoadedProjectLocked(snapshot, pending, true, h.acceptedDiff)
			result.ActionError = errorArchitectureStale
			return result, errorArchitectureStale, http.StatusConflict, true
		}
		if observed == review.baseRevision {
			result := h.responseForLoadedProjectLocked(snapshot, pending, false, h.acceptedDiff)
			result.ActionError = errorUpdateFailed
			return result, errorUpdateFailed, http.StatusInternalServerError, false
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
			result := h.responseForLoadedProjectLocked(*h.loadedSnapshot, nil, h.loadedStale, h.acceptedDiff)
			result.ActionError = errorUpdatedReload
			return result, errorUpdatedReload, http.StatusInternalServerError, false
		}
	}
	h.loadedSnapshot = &acceptedSnapshot
	h.loadedProject.projectName = acceptedSnapshot.ProjectName()
	h.loadedProject.projectSlug = acceptedSnapshot.ProjectSlug()
	h.loadedProject.validatedCurrent = nil
	h.loadedStale = false
	return responseForSnapshot(acceptedSnapshot, nil, false, h.acceptedDiff), "", http.StatusOK, false
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
	pending.validationDiagram = ""
	pending.validationDiagramField = ""
	var componentError *architecture.ComponentValidationError
	if errors.As(err, &componentError) {
		pending.validationItem = componentError.ComponentID
		pending.validationRelationshipPosition = componentError.RelationshipPosition
		pending.validationRelationshipField = componentError.RelationshipField
	}
	var diagramError *architecture.DiagramValidationError
	if errors.As(err, &diagramError) {
		pending.validationItem = diagramError.ComponentID
		pending.validationDiagram = diagramError.DiagramID
		pending.validationDiagramField = diagramError.Field
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
	case errors.Is(err, architecture.ErrDiagramTitleRequired):
		pending.validationCode = "diagram_title_required"
	case errors.Is(err, architecture.ErrDiagramCycle):
		pending.validationCode = "diagram_cycle"
	case errors.Is(err, architecture.ErrDiagramHomeInvalid):
		pending.validationCode = "diagram_home_invalid"
	case !errors.Is(err, architecture.ErrInvalid):
		pending.validationCode = "change_unavailable"
	}
}

func (h *Handler) decodeArchitectureAction(response http.ResponseWriter, request *http.Request) (architectureActionRequest, bool) {
	if request.Header.Get("Origin") != h.expectedOrigin {
		writeJSON(response, http.StatusForbidden, errorResponse{Code: errorOriginMismatch})
		return architectureActionRequest{}, false
	}
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var payload architectureActionRequest
	if err := decoder.Decode(&payload); err != nil || ensureJSONEnd(decoder) != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorLookupFailed})
		return architectureActionRequest{}, false
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

func (h *Handler) staticFiles() http.Handler {
	files := http.FileServer(http.Dir(h.uiDirectory))
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/" || strings.HasPrefix(request.URL.Path, "/projects/") {
			indexPath := filepath.Join(h.uiDirectory, "index.html")
			if _, err := os.Stat(indexPath); err != nil {
				http.Error(response, "WorkBraid browser UI is not built", http.StatusServiceUnavailable)
				return
			}
			if request.URL.Path != "/" {
				http.ServeFile(response, request, indexPath)
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
