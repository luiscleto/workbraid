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
}

type loadedProject struct {
	storeID     string
	projectSlug string
	projectName string
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
	homeMoveDestinations           []componentHomeDestinationsResponse
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
	mux.Handle("/", handler.staticFiles())
	return handler, mux
}

type openProjectRequest struct {
	ProjectSlug string `json:"project_slug"`
}

type acceptChangesRequest struct {
	ProjectSlug   string `json:"project_slug"`
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
	errorDiagramOwnDetail        = "diagram_own_detail"
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
	defer h.stateMutex.Unlock()
	if h.pending != nil {
		result := h.currentArchitectureResponseLocked()
		result.ActionError = errorPendingBlocksSwitch
		writeJSON(response, http.StatusConflict, result)
		return
	}
	snapshot, err := h.architecture.CreateProject(request.Context(), payload.Name)
	if err != nil {
		writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorProjectCreateFailed})
		return
	}
	h.publishSnapshotLocked(snapshot)
	writeJSON(response, http.StatusCreated, h.currentArchitectureResponseLocked())
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
	defer h.stateMutex.Unlock()
	snapshot, err := h.architecture.OpenProject(request.Context(), payload.ProjectSlug)
	if errors.Is(err, architecture.ErrProjectNotFound) {
		writeJSON(response, http.StatusNotFound, errorResponse{Code: errorProjectNotFound})
		return
	}
	if errors.Is(err, architecture.ErrCatalogConflict) {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorCatalogConflict})
		return
	}
	if err != nil {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorCatalogUnavailable})
		return
	}
	if h.pending != nil && h.loadedProject != nil && h.loadedProject.storeID != snapshot.StoreID() {
		result := h.currentArchitectureResponseLocked()
		result.ActionError = errorPendingBlocksSwitch
		writeJSON(response, http.StatusConflict, result)
		return
	}
	stalePending := h.publishSnapshotLocked(snapshot)
	result := h.currentArchitectureResponseLocked()
	if stalePending {
		result.ActionError = errorArchitectureStale
	}
	writeJSON(response, http.StatusOK, result)
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
	ComponentID string   `json:"component_id"`
	DiagramIDs  []string `json:"diagram_ids"`
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
		} else if len(pending.homeMoveDestinations) > 0 {
			result.HomeMoveDestinations = cloneComponentHomeDestinations(pending.homeMoveDestinations)
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
		diagramIDs := snapshot.ComponentHomeDestinationDiagramIDs(component.ID)
		if diagramIDs == nil {
			continue
		}
		values = append(values, componentHomeDestinationsResponse{ComponentID: component.ID, DiagramIDs: diagramIDs})
	}
	return values
}

func cloneComponentHomeDestinations(values []componentHomeDestinationsResponse) []componentHomeDestinationsResponse {
	cloned := make([]componentHomeDestinationsResponse, len(values))
	for index, value := range values {
		cloned[index] = componentHomeDestinationsResponse{
			ComponentID: value.ComponentID,
			DiagramIDs:  append([]string(nil), value.DiagramIDs...),
		}
	}
	return cloned
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
		h.loadedProject = &loadedProject{storeID: snapshot.StoreID(), projectName: snapshot.ProjectName(), projectSlug: snapshot.ProjectSlug()}
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
	return responseForSnapshot(*h.loadedSnapshot, h.pending, h.loadedStale, h.acceptedDiff)
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
	if h.loadedSnapshot == nil || h.loadedProject == nil || payload.ProjectSlug != h.loadedProject.projectSlug {
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
	h.loadedProject.projectName = replacement.ProjectName()
	h.loadedProject.projectSlug = replacement.ProjectSlug()
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
	if h.loadedSnapshot == nil || h.loadedProject == nil || payload.ProjectSlug != h.loadedProject.projectSlug {
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
	if h.loadedSnapshot == nil || h.loadedProject == nil || payload.ProjectSlug != h.loadedProject.projectSlug {
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
	ProjectSlug          string                 `json:"project_slug"`
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
	if h.loadedSnapshot == nil || h.loadedProject == nil || payload.ProjectSlug != h.loadedProject.projectSlug {
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
	h.pending.validationDiagram = ""
	h.pending.validationDiagramField = ""
	if err != nil {
		h.recordCandidateValidation(h.pending, err)
		if h.pending.validationItem == "" {
			h.pending.validationItem = change.ID
		}
		if h.pending.validationCode == "change_unavailable" {
			writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorChangeFailed})
			return
		}
	} else {
		h.pending.candidate = &candidate
	}
	writeJSON(response, http.StatusOK, responseForSnapshot(snapshot, h.pending, h.loadedStale, h.acceptedDiff))
}

func (h *Handler) constructCandidate(ctx context.Context, snapshot architecture.Snapshot, pending *pendingChangeSet) (architecture.Candidate, error) {
	return h.architecture.ConstructCandidate(ctx, snapshot, pending.changes, architecture.CandidateComposition{
		NewComponentHomes: pending.newComponentHomes,
		DetailDiagrams:    pending.detailDiagrams,
		DiagramTitles:     pending.diagramTitles,
		HomeMoves:         pending.homeMoves,
		References:        pending.references,
	})
}

type diagramMutationRequest struct {
	ProjectSlug      string `json:"project_slug"`
	ExpectedRevision string `json:"expected_revision"`
	DiagramID        string `json:"diagram_id,omitempty"`
	ComponentID      string `json:"component_id,omitempty"`
	Title            string `json:"title,omitempty"`
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
	if h.loadedSnapshot == nil || h.loadedProject == nil || payload.ProjectSlug != h.loadedProject.projectSlug {
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
	if h.pending == nil {
		h.pending = &pendingChangeSet{storeID: snapshot.StoreID(), baseRevision: snapshot.Revision(), baseSnapshot: snapshot}
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

func pendingHasComponent(snapshot architecture.Snapshot, pending *pendingChangeSet, id string) bool {
	if snapshot.HasComponent(id) {
		return true
	}
	for _, change := range pending.changes {
		if change.New && change.ID == id {
			return true
		}
	}
	return false
}

func (h *Handler) rebuildPendingLocked(ctx context.Context, snapshot architecture.Snapshot, pending *pendingChangeSet) {
	pending.generation++
	pending.review = nil
	pending.candidate = nil
	pending.homeMoveDestinations = nil
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

func (h *Handler) createDetailDiagram(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeDiagramMutation(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	snapshot, pending, ok := h.writableV2PendingLocked(response, payload)
	if !ok {
		return
	}
	for _, addition := range pending.detailDiagrams {
		if addition.AnchorComponentID == payload.ComponentID {
			writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangeFailed})
			return
		}
	}
	current := snapshot
	if pending.candidate != nil {
		current = pending.candidate.Snapshot()
	}
	_, detailID, home := current.ComponentHome(payload.ComponentID)
	if !home || detailID != "" {
		if len(pending.changes) == 0 && len(pending.detailDiagrams) == 0 && len(pending.diagramTitles) == 0 && len(pending.homeMoves) == 0 {
			h.pending = nil
		}
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangeFailed})
		return
	}
	addition := snapshot.NewDetailDiagramChange(pending.detailDiagrams, payload.Title, payload.ComponentID)
	pending.detailDiagrams = append(pending.detailDiagrams, addition)
	h.rebuildPendingLocked(request.Context(), snapshot, pending)
	writeJSON(response, http.StatusOK, h.currentArchitectureResponseLocked())
}

func (h *Handler) editDiagramTitle(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeDiagramMutation(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	snapshot, pending, ok := h.writableV2PendingLocked(response, payload)
	if !ok {
		return
	}
	if !pendingHasDiagram(snapshot, pending, payload.DiagramID) {
		if pendingChangeSetEmpty(pending) {
			h.pending = nil
		}
		writeJSON(response, http.StatusNotFound, errorResponse{Code: errorChangeFailed})
		return
	}
	for index := range pending.detailDiagrams {
		if pending.detailDiagrams[index].ID == payload.DiagramID {
			pending.detailDiagrams[index].Title = payload.Title
			h.rebuildPendingLocked(request.Context(), snapshot, pending)
			writeJSON(response, http.StatusOK, h.currentArchitectureResponseLocked())
			return
		}
	}
	updated := false
	for index := range pending.diagramTitles {
		if pending.diagramTitles[index].DiagramID == payload.DiagramID {
			pending.diagramTitles[index].Title = payload.Title
			updated = true
			break
		}
	}
	if !updated {
		pending.diagramTitles = append(pending.diagramTitles, architecture.DiagramTitleChange{DiagramID: payload.DiagramID, Title: payload.Title})
	}
	h.rebuildPendingLocked(request.Context(), snapshot, pending)
	writeJSON(response, http.StatusOK, h.currentArchitectureResponseLocked())
}

func (h *Handler) moveComponentHome(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeDiagramMutation(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	hadPending := h.pending != nil
	snapshot, pending, ok := h.writableV2PendingLocked(response, payload)
	if !ok {
		return
	}
	if !pendingHasComponent(snapshot, pending, payload.ComponentID) || !pendingHasDiagram(snapshot, pending, payload.DiagramID) {
		if pendingChangeSetEmpty(pending) {
			h.pending = nil
		}
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorChangeFailed})
		return
	}
	authority := snapshot
	if pending.candidate != nil {
		authority = pending.candidate.Snapshot()
	} else if hadPending && !pendingChangeSetEmpty(pending) {
		withoutCurrentMove := *pending
		withoutCurrentMove.homeMoves = homeMovesWithoutComponent(pending.homeMoves, payload.ComponentID)
		candidate, err := h.constructCandidate(request.Context(), snapshot, &withoutCurrentMove)
		if err != nil {
			writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesUnavailable})
			return
		}
		authority = candidate.Snapshot()
	}
	if !authority.HasComponent(payload.ComponentID) || !authority.HasDiagram(payload.DiagramID) {
		if !hadPending && pendingChangeSetEmpty(pending) {
			h.pending = nil
		}
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorChangeFailed})
		return
	}
	currentAuthorityHome, ownedDetailID, hasHome := authority.ComponentHome(payload.ComponentID)
	if !hasHome || payload.DiagramID == ownedDetailID {
		if !hadPending && pendingChangeSetEmpty(pending) {
			h.pending = nil
		}
		code := errorChangeFailed
		if payload.DiagramID == ownedDetailID && ownedDetailID != "" {
			code = errorDiagramOwnDetail
		}
		writeJSON(response, http.StatusConflict, errorResponse{Code: code})
		return
	}
	// A home consumes any reference intent at its destination. Leaving the
	// current home records an explicit absence so a base reference that was
	// temporarily converted to home cannot reappear during reconstruction.
	pending.references = referenceChangesWithoutPair(pending.references, payload.DiagramID, payload.ComponentID)
	setReferenceChange(pending, currentAuthorityHome, payload.ComponentID, false)
	remaining := homeMovesWithoutComponent(pending.homeMoves, payload.ComponentID)
	pending.homeMoves = remaining
	withoutMove, err := h.constructCandidate(request.Context(), snapshot, pending)
	currentHome := ""
	if err == nil {
		currentHome, _, _ = withoutMove.Snapshot().ComponentHome(payload.ComponentID)
	}
	if currentHome != payload.DiagramID {
		pending.homeMoves = append(pending.homeMoves, architecture.ComponentHomeMove{ComponentID: payload.ComponentID, DiagramID: payload.DiagramID})
	}
	if pendingChangeSetEmpty(pending) {
		h.pending = nil
		writeJSON(response, http.StatusOK, h.currentArchitectureResponseLocked())
		return
	}
	h.rebuildPendingLocked(request.Context(), snapshot, pending)
	if pending.candidate == nil && pending.validationDiagramField == "home" && pending.validationItem == payload.ComponentID && err == nil {
		pending.homeMoveDestinations = componentHomeDestinations(withoutMove.Snapshot())
	}
	writeJSON(response, http.StatusOK, h.currentArchitectureResponseLocked())
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
	hadPending := h.pending != nil
	snapshot, pending, ok := h.writableV2PendingLocked(response, payload)
	if !ok {
		return
	}
	authority := snapshot
	if pending.candidate != nil {
		authority = pending.candidate.Snapshot()
	} else if !pendingChangeSetEmpty(pending) {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesUnavailable})
		return
	}
	role, appears := authority.ComponentAppearanceRole(payload.DiagramID, payload.ComponentID)
	homeID, _, hasHome := authority.ComponentHome(payload.ComponentID)
	valid := authority.HasDiagram(payload.DiagramID) && authority.HasComponent(payload.ComponentID) && hasHome
	if present {
		valid = valid && !appears && homeID != payload.DiagramID
	} else {
		valid = valid && appears && role == "reference"
	}
	if !valid {
		if !hadPending && pendingChangeSetEmpty(pending) {
			h.pending = nil
		}
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangeFailed})
		return
	}
	setReferenceChange(pending, payload.DiagramID, payload.ComponentID, present)
	h.rebuildPendingLocked(request.Context(), snapshot, pending)
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
	if h.loadedSnapshot == nil || h.loadedProject == nil || payload.ProjectSlug != h.loadedProject.projectSlug {
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
		result := responseForSnapshot(snapshot, h.pending, false, h.acceptedDiff)
		result.ActionError = errorReviewFailed
		h.stateMutex.Unlock()
		writeJSON(response, status, result)
		return
	}
	diff, err := h.architecture.CandidateDiff(request.Context(), snapshot, candidate)
	if err != nil {
		result := responseForSnapshot(snapshot, h.pending, false, h.acceptedDiff)
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
	h.pending.validationDiagram = ""
	h.pending.validationDiagramField = ""
	h.pending.review = &reviewBinding{
		baseRevision: snapshot.Revision(), candidateTree: candidate.Tree(), generation: h.pending.generation,
		diff: string(diff), candidate: candidate,
	}
	// Build the complete immutable visual presentation while the binding and
	// pending generation are protected by the same concrete state lock. JSON
	// serialization can then proceed without blocking invalidating mutations.
	result := responseForSnapshot(snapshot, h.pending, false, h.acceptedDiff)
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
	if h.loadedSnapshot == nil || h.loadedProject == nil || payload.ProjectSlug != h.loadedProject.projectSlug {
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
		result := responseForSnapshot(snapshot, pending, false, h.acceptedDiff)
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
		result := responseForSnapshot(snapshot, pending, false, h.acceptedDiff)
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
		result := responseForSnapshot(snapshot, pending, true, h.acceptedDiff)
		result.ActionError = errorUpdateUncertain
		writeJSON(response, http.StatusConflict, result)
		return
	}
	if observed != review.baseRevision {
		h.markStale(pending)
		result := responseForSnapshot(snapshot, pending, true, h.acceptedDiff)
		result.ActionError = errorArchitectureStale
		writeJSON(response, http.StatusConflict, result)
		return
	}

	successor, err := h.architecture.CreateSuccessor(transitionContext, snapshot, review.candidate)
	if err != nil {
		result := responseForSnapshot(snapshot, pending, false, h.acceptedDiff)
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
			result := responseForSnapshot(snapshot, pending, true, h.acceptedDiff)
			result.ActionError = errorUpdateUncertain
			writeJSON(response, http.StatusInternalServerError, result)
			return
		}
		if observed != successor && observed != review.baseRevision {
			h.markStale(pending)
			result := responseForSnapshot(snapshot, pending, true, h.acceptedDiff)
			result.ActionError = errorArchitectureStale
			writeJSON(response, http.StatusConflict, result)
			return
		}
		if observed == review.baseRevision {
			result := responseForSnapshot(snapshot, pending, false, h.acceptedDiff)
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
			result := responseForSnapshot(*h.loadedSnapshot, nil, h.loadedStale, h.acceptedDiff)
			result.ActionError = errorUpdatedReload
			writeJSON(response, http.StatusInternalServerError, result)
			return
		}
	}
	h.loadedSnapshot = &acceptedSnapshot
	h.loadedProject.projectName = acceptedSnapshot.ProjectName()
	h.loadedProject.projectSlug = acceptedSnapshot.ProjectSlug()
	h.loadedStale = false
	writeJSON(response, http.StatusOK, responseForSnapshot(acceptedSnapshot, nil, false, h.acceptedDiff))
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
