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
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"workbraid/internal/architecture"
)

const (
	maxRequestBody                 = 64 << 10
	architectureTransitionTimeout  = 30 * time.Second
	architectureObservationTimeout = 10 * time.Second
)

type Handler struct {
	expectedOrigin        string
	uiDirectory           string
	architecture          *architecture.Manager
	stateMutex            sync.Mutex
	loadedSnapshot        *architecture.Snapshot
	loadedProject         *loadedProject
	loadedStale           bool
	acceptedIndeterminate bool
	acceptedDiff          string
	changeSets            map[string]*pendingChangeSet
	unavailableChangeSets []architecture.UnavailableChangeSet
	reviews               map[string]architecture.ReviewSubmission
	unavailableReviews    []architecture.UnavailableReview

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
	// changeSetLoadFailure is a focused test seam for transient private-Git
	// change-set recovery failure. Production never sets it.
	changeSetLoadFailure func() error
	// Narrow checkpoints around the real reconciliation transaction. Tests
	// change real refs or drop a real response; production leaves these nil.
	beforeReconciliationReobserve   func()
	beforeReconciliationTransaction func()
	afterReconciliationTransaction  func()
}

type loadedProject struct {
	storeID          string
	projectSlug      string
	projectName      string
	validatedCurrent *architecture.Snapshot
}

type pendingChangeSet struct {
	architectureVersion            int
	nodePositions                  []architecture.NodePositionChange
	edgeRoutes                     []architecture.EdgeRouteChange
	nodeSizes                      []architecture.NodeSizeChange
	detailReassignments            []architecture.DetailReassignment
	id                             string
	name                           string
	lifecycle                      string
	proposal                       string
	appliedRevision                string
	refObject                      string
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
		changeSets:     make(map[string]*pendingChangeSet),
		reviews:        make(map[string]architecture.ReviewSubmission),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/projects", handler.projectCatalog)
	mux.HandleFunc("POST /api/projects/create", handler.createProject)
	mux.HandleFunc("POST /api/projects/open", handler.openProject)
	mux.HandleFunc("POST /api/architecture/components/add", handler.addComponent)
	mux.HandleFunc("POST /api/architecture/components/edit", handler.editComponent)
	mux.HandleFunc("POST /api/architecture/diagrams/detail", handler.createDetailDiagram)
	mux.HandleFunc("POST /api/architecture/diagrams/parent-options", handler.browserDetailParentOptions)
	mux.HandleFunc("POST /api/architecture/diagrams/reassign-detail", handler.browserReassignDetail)
	mux.HandleFunc("POST /api/architecture/diagrams/set-position", handler.browserPlacement)
	mux.HandleFunc("POST /api/architecture/diagrams/auto-layout", handler.browserPlacement)
	mux.HandleFunc("POST /api/architecture/diagrams/set-size", handler.browserSizing)
	mux.HandleFunc("POST /api/architecture/diagrams/set-route", handler.browserRouting)
	mux.HandleFunc("POST /api/architecture/diagrams/restore-default-route", handler.browserRouting)
	mux.HandleFunc("POST /api/architecture/diagrams/restore-default-size", handler.browserSizing)
	mux.HandleFunc("POST /api/architecture/diagrams/title", handler.editDiagramTitle)
	mux.HandleFunc("POST /api/architecture/components/move-home", handler.moveComponentHome)
	mux.HandleFunc("POST /api/architecture/diagrams/show-component", handler.showComponentHere)
	mux.HandleFunc("POST /api/architecture/diagrams/stop-showing-component", handler.stopShowingHere)
	mux.HandleFunc("POST /api/architecture/change-sets/create", handler.createChangeSet)
	mux.HandleFunc("POST /api/architecture/change-sets/rename", handler.renameChangeSet)
	mux.HandleFunc("POST /api/architecture/change-sets/proposal", handler.editChangeSetProposal)
	mux.HandleFunc("POST /api/architecture/review", handler.reviewChanges)
	mux.HandleFunc("POST /api/architecture/review-submissions/inspect", handler.inspectReviewSubmission)
	mux.HandleFunc("POST /api/architecture/review-submissions/submit", handler.submitReviewSubmission)
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
	ChangeSetID               string  `json:"change_set_id,omitempty"`
}

type acceptChangesRequest struct {
	ProjectSlug   string `json:"project_slug"`
	StoreID       string `json:"store_id"`
	BaseRevision  string `json:"base_revision"`
	CandidateTree string `json:"candidate_tree"`
	Generation    uint64 `json:"generation"`
	ChangeSetID   string `json:"change_set_id"`
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
	if code != "" {
		writeJSON(response, http.StatusInternalServerError, errorResponse{Code: code})
		return
	}
	writeJSON(response, http.StatusCreated, result)
}

func (h *Handler) createProjectLocked(ctx context.Context, name string) (architectureResponse, string) {
	snapshot, err := h.architecture.CreateProject(ctx, name)
	if err != nil {
		return architectureResponse{}, errorProjectCreateFailed
	}
	h.publishSnapshotLocked(ctx, snapshot)
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
	h.publishSnapshotLocked(ctx, snapshot)
	result := h.currentArchitectureResponseLocked()
	return result, ""
}

type architectureResponse struct {
	ProjectSlug           string                              `json:"project_slug"`
	StoreID               string                              `json:"store_id"`
	ProjectName           string                              `json:"project_name"`
	State                 string                              `json:"state"`
	Revision              string                              `json:"revision"`
	FormatVersion         int                                 `json:"format_version"`
	ComponentCount        int                                 `json:"component_count"`
	ComponentTitles       []string                            `json:"component_titles"`
	Components            []componentResponse                 `json:"components"`
	RootDiagramID         string                              `json:"root_diagram_id,omitempty"`
	Diagrams              []diagramResponse                   `json:"diagrams,omitempty"`
	HomeMoveDestinations  []componentHomeDestinationsResponse `json:"home_move_destinations,omitempty"`
	ReferenceChoices      []referenceChoiceResponse           `json:"reference_choices,omitempty"`
	Changes               *changesResponse                    `json:"changes,omitempty"`
	ChangeSets            []*changesResponse                  `json:"change_sets"`
	UnavailableChangeSets []unavailableChangeSetResponse      `json:"unavailable_change_sets,omitempty"`
	ReviewSubmissions     []reviewSubmissionSummaryResponse   `json:"review_submissions,omitempty"`
	UnavailableReviews    []unavailableReviewResponse         `json:"unavailable_reviews,omitempty"`
	SubmittedReview       *reviewSubmissionResponse           `json:"submitted_review,omitempty"`
	ActionChangeSetID     string                              `json:"action_change_set_id,omitempty"`
	ActionReviewID        string                              `json:"action_review_id,omitempty"`
	Stale                 bool                                `json:"stale,omitempty"`
	ParentDiff            string                              `json:"parent_diff,omitempty"`
	ActionError           string                              `json:"action_error,omitempty"`
	AlreadyApplied        bool                                `json:"-"`
}

type componentResponse struct {
	ID             string                 `json:"id"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	MarkdownSource string                 `json:"markdown_source,omitempty"`
	Filename       string                 `json:"filename"`
	Relationships  []relationshipResponse `json:"relationships"`
}

type relationshipResponse struct {
	TargetID      string `json:"target_id"`
	Label         string `json:"label"`
	ProjectionKey string `json:"projection_key,omitempty"`
}

type pendingComponentResponse struct {
	ID                   string                 `json:"id"`
	Title                string                 `json:"title"`
	Description          string                 `json:"description"`
	Path                 string                 `json:"path"`
	New                  bool                   `json:"new"`
	TitleChanged         bool                   `json:"title_changed"`
	DescriptionChanged   bool                   `json:"description_changed"`
	Relationships        []relationshipResponse `json:"relationships"`
	RelationshipsChanged bool                   `json:"relationships_changed"`
}

type relationshipTargetResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Context string `json:"context,omitempty"`
	New     bool   `json:"new,omitempty"`
}

type changesResponse struct {
	StateObject                    string                                   `json:"change_set_state"`
	CandidateTree                  string                                   `json:"candidate_tree,omitempty"`
	DetailReassignments            []architecture.DetailReassignment        `json:"detail_reassignments,omitempty"`
	ID                             string                                   `json:"id"`
	Name                           string                                   `json:"name"`
	Lifecycle                      string                                   `json:"lifecycle"`
	Proposal                       string                                   `json:"proposal_markdown"`
	AppliedRevision                string                                   `json:"applied_revision,omitempty"`
	OutOfDate                      bool                                     `json:"out_of_date,omitempty"`
	ReadOnly                       bool                                     `json:"read_only,omitempty"`
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

type unavailableChangeSetResponse struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Lifecycle string `json:"lifecycle,omitempty"`
	Reason    string `json:"reason"`
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
	ChangeSetID   string                     `json:"change_set_id"`
	ReviewedState string                     `json:"reviewed_state"`
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
		ChangeSets:           []*changesResponse{},
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
				ID: change.ID, Title: change.Title, Description: change.Description, Path: change.Path, New: change.New,
				TitleChanged: change.TitleChanged, DescriptionChanged: change.DescriptionChanged,
				Relationships: relationships, RelationshipsChanged: change.RelationshipsChanged,
			}
		}
		result.Changes = &changesResponse{
			StateObject:                    pending.refObject,
			DetailReassignments:            append([]architecture.DetailReassignment(nil), pending.detailReassignments...),
			ID:                             pending.id,
			Name:                           pending.name,
			Lifecycle:                      pending.lifecycle,
			Proposal:                       pending.proposal,
			AppliedRevision:                pending.appliedRevision,
			OutOfDate:                      pending.lifecycle == "active" && pending.baseRevision != snapshot.Revision(),
			ReadOnly:                       pending.lifecycle != "active",
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
			Stale:                          pending.stale || stale,
		}
		if pending.candidate != nil {
			candidateProjection := projectSnapshot(pending.candidate.Snapshot(), "")
			result.Changes.Candidate = &candidateProjection
			result.Changes.CandidateTree = pending.candidate.Tree()
			result.HomeMoveDestinations = componentHomeDestinations(pending.candidate.Snapshot())
			result.ReferenceChoices = referenceChoices(pending.candidate.Snapshot())
		}
		if pending.candidate == nil {
			// Reference authoring resolves against the complete candidate. An
			// invalid pending set has no coherent reference-choice authority, so
			// never fall back to the accepted snapshot here.
			result.ReferenceChoices = nil
		}
		if pending.review != nil && pending.review.generation == pending.generation && pending.candidate != nil && pending.review.candidateTree == pending.candidate.Tree() {
			before, withChanges, comparison := captureReviewPresentation(pending.baseSnapshot, pending.review.candidate.Snapshot())
			reviewedState := ""
			if pending.lifecycle == "active" {
				reviewedState = pending.refObject
			}
			result.Changes.Review = &reviewResponse{
				ChangeSetID:   pending.id,
				ReviewedState: reviewedState,
				Diff:          pending.review.diff, BaseRevision: pending.review.baseRevision,
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
	ids := make([]string, 0, len(h.changeSets))
	for id := range h.changeSets {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		left, right := h.changeSets[ids[i]], h.changeSets[ids[j]]
		if left.lifecycle != right.lifecycle {
			return left.lifecycle == "active"
		}
		if left.name != right.name {
			return left.name < right.name
		}
		return left.id < right.id
	})
	result.ChangeSets = make([]*changesResponse, 0, len(ids))
	for _, id := range ids {
		entry := responseForSnapshot(snapshot, h.changeSets[id], stale, "").Changes
		result.ChangeSets = append(result.ChangeSets, entry)
	}
	result.UnavailableChangeSets = make([]unavailableChangeSetResponse, len(h.unavailableChangeSets))
	for index, unavailable := range h.unavailableChangeSets {
		result.UnavailableChangeSets[index] = unavailableChangeSetResponse{ID: unavailable.ID, Name: unavailable.Name, Lifecycle: unavailable.Lifecycle, Reason: unavailable.Reason}
	}
	result.ReviewSubmissions = h.reviewSummariesLocked()
	result.UnavailableReviews = make([]unavailableReviewResponse, len(h.unavailableReviews))
	for index, unavailable := range h.unavailableReviews {
		result.UnavailableReviews[index] = unavailableReviewResponse{ChangeSetID: unavailable.ChangeSetID, ReviewID: unavailable.ReviewID, Reason: unavailable.Reason}
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
	if snapshot.FormatVersion() < 2 {
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

func (h *Handler) publishSnapshotLocked(ctx context.Context, snapshot architecture.Snapshot) {
	keepAcceptedDiff := h.loadedSnapshot != nil && h.loadedSnapshot.StoreID() == snapshot.StoreID() && h.loadedSnapshot.Revision() == snapshot.Revision()
	h.loadedSnapshot = &snapshot
	h.loadedProject = &loadedProject{storeID: snapshot.StoreID(), projectName: snapshot.ProjectName(), projectSlug: snapshot.ProjectSlug()}
	h.loadedStale = false
	h.acceptedIndeterminate = false
	if !keepAcceptedDiff {
		h.acceptedDiff = ""
	}
	if err := h.loadChangeSetsLocked(ctx, snapshot); err != nil {
		h.changeSets = make(map[string]*pendingChangeSet)
		h.unavailableChangeSets = []architecture.UnavailableChangeSet{{Reason: "Change sets could not be read."}}
	}
}

func (h *Handler) loadChangeSetsLocked(ctx context.Context, snapshot architecture.Snapshot) error {
	if h.changeSetLoadFailure != nil {
		if err := h.changeSetLoadFailure(); err != nil {
			return err
		}
	}
	records, unavailable, err := h.architecture.LoadChangeSets(ctx, snapshot.StoreID())
	if err != nil {
		return err
	}
	nextRecords := make(map[string]*pendingChangeSet, len(records))
	nextUnavailable := append([]architecture.UnavailableChangeSet(nil), unavailable...)
	for _, durable := range records {
		record := pendingFromDurableChangeSet(durable)
		if durable.ValidationError != nil {
			h.recordCandidateValidation(record, durable.ValidationError)
		}
		if durable.Review != nil && durable.Candidate != nil {
			diff, diffErr := h.architecture.CandidateDiff(ctx, durable.BaseSnapshot, *durable.Candidate)
			if diffErr != nil {
				nextUnavailable = append(nextUnavailable, architecture.UnavailableChangeSet{ID: durable.ID, Name: durable.Name, Lifecycle: durable.Lifecycle, Reason: "Review comparison could not be reconstructed."})
				continue
			}
			record.review = &reviewBinding{baseRevision: durable.Review.BaseRevision, candidateTree: durable.Review.CandidateTree, generation: durable.Review.Generation, diff: string(diff), candidate: *durable.Candidate}
		}
		nextRecords[record.id] = record
	}
	h.changeSets = nextRecords
	h.unavailableChangeSets = nextUnavailable
	reviews, unavailableReviews, reviewErr := h.architecture.LoadReviews(ctx, snapshot.StoreID(), "")
	if reviewErr != nil {
		return reviewErr
	}
	h.reviews = make(map[string]architecture.ReviewSubmission, len(reviews))
	for _, review := range reviews {
		h.reviews[review.ID] = review
	}
	h.unavailableReviews = unavailableReviews
	return nil
}

func pendingFromDurableChangeSet(durable architecture.ChangeSet) *pendingChangeSet {
	record := &pendingChangeSet{
		architectureVersion: durable.Composition.ArchitectureVersion,
		nodePositions:       append([]architecture.NodePositionChange(nil), durable.Composition.NodePositions...),
		nodeSizes:           append([]architecture.NodeSizeChange(nil), durable.Composition.NodeSizes...),
		edgeRoutes:          append([]architecture.EdgeRouteChange(nil), durable.Composition.EdgeRoutes...),
		detailReassignments: append([]architecture.DetailReassignment(nil), durable.Composition.DetailReassignments...),
		id:                  durable.ID, name: durable.Name, lifecycle: durable.Lifecycle, proposal: durable.Proposal,
		appliedRevision: durable.AppliedRevision, refObject: durable.RefObject,
		storeID: durable.BaseSnapshot.StoreID(), baseRevision: durable.BaseRevision, baseSnapshot: durable.BaseSnapshot,
		changes:           append([]architecture.ComponentChange(nil), durable.Changes...),
		newComponentHomes: append([]architecture.NewComponentHome(nil), durable.Composition.NewComponentHomes...),
		detailDiagrams:    append([]architecture.DetailDiagramChange(nil), durable.Composition.DetailDiagrams...),
		diagramTitles:     append([]architecture.DiagramTitleChange(nil), durable.Composition.DiagramTitles...),
		homeMoves:         append([]architecture.ComponentHomeMove(nil), durable.Composition.HomeMoves...),
		references:        append([]architecture.ReferenceAppearanceChange(nil), durable.Composition.References...),
		candidate:         durable.Candidate, generation: durable.Generation,
	}
	if durable.Review != nil && durable.Candidate != nil {
		record.review = &reviewBinding{baseRevision: durable.Review.BaseRevision, candidateTree: durable.Review.CandidateTree, generation: durable.Review.Generation, candidate: *durable.Candidate}
	}
	return record
}

func (h *Handler) durableChangeSet(record *pendingChangeSet) architecture.ChangeSet {
	value := architecture.ChangeSet{
		ID: record.id, Name: record.name, Lifecycle: record.lifecycle, Proposal: record.proposal,
		AppliedRevision: record.appliedRevision, RefObject: record.refObject,
		BaseRevision: record.baseRevision, Generation: record.generation, BaseSnapshot: record.baseSnapshot,
		Changes:     record.changes,
		Composition: architecture.CandidateComposition{ArchitectureVersion: record.architectureVersion, NodePositions: record.nodePositions, NodeSizes: record.nodeSizes, EdgeRoutes: record.edgeRoutes, DetailReassignments: record.detailReassignments, NewComponentHomes: record.newComponentHomes, DetailDiagrams: record.detailDiagrams, DiagramTitles: record.diagramTitles, HomeMoves: record.homeMoves, References: record.references},
		Candidate:   record.candidate,
	}
	if record.review != nil {
		value.Review = &architecture.ChangeSetReview{BaseRevision: record.review.baseRevision, CandidateTree: record.review.candidateTree, Generation: record.review.generation}
	}
	return value
}

func clonePending(record *pendingChangeSet) *pendingChangeSet {
	if record == nil {
		return nil
	}
	clone := *record
	clone.nodePositions = append([]architecture.NodePositionChange(nil), record.nodePositions...)
	clone.nodeSizes = append([]architecture.NodeSizeChange(nil), record.nodeSizes...)
	clone.edgeRoutes = append([]architecture.EdgeRouteChange(nil), record.edgeRoutes...)
	clone.detailReassignments = append([]architecture.DetailReassignment(nil), record.detailReassignments...)
	clone.changes = append([]architecture.ComponentChange(nil), record.changes...)
	for index := range clone.changes {
		clone.changes[index].Relationships = append([]architecture.AuthoringRelationship(nil), record.changes[index].Relationships...)
	}
	clone.newComponentHomes = append([]architecture.NewComponentHome(nil), record.newComponentHomes...)
	clone.detailDiagrams = append([]architecture.DetailDiagramChange(nil), record.detailDiagrams...)
	clone.diagramTitles = append([]architecture.DiagramTitleChange(nil), record.diagramTitles...)
	clone.homeMoves = append([]architecture.ComponentHomeMove(nil), record.homeMoves...)
	clone.references = append([]architecture.ReferenceAppearanceChange(nil), record.references...)
	return &clone
}

func (h *Handler) changeSetLocked(id string) *pendingChangeSet {
	record := h.changeSets[id]
	if record == nil || record.lifecycle != "active" {
		return nil
	}
	return record
}

func (h *Handler) persistActiveLocked(ctx context.Context, proposed *pendingChangeSet, expectedObject string) bool {
	object, err := h.architecture.WriteActiveChangeSet(ctx, proposed.storeID, h.durableChangeSet(proposed), expectedObject)
	if err != nil {
		return false
	}
	proposed.refObject = object
	h.changeSets[proposed.id] = proposed
	return true
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
	result := h.responseForLoadedProjectLocked(*h.loadedSnapshot, nil, h.loadedStale, h.acceptedDiff)
	if h.acceptedIndeterminate {
		result.ActionError = errorRefreshFailed
	}
	return result
}

func (h *Handler) clearLoadedProjectLocked() {
	h.loadedSnapshot = nil
	h.loadedProject = nil
	h.loadedStale = false
	h.acceptedIndeterminate = false
	h.acceptedDiff = ""
	h.changeSets = make(map[string]*pendingChangeSet)
	h.unavailableChangeSets = nil
	h.reviews = make(map[string]architecture.ReviewSubmission)
	h.unavailableReviews = nil
}

func (h *Handler) matchesLoadedProjectLocked(projectSlug, storeID string) bool {
	return h.loadedSnapshot != nil && h.loadedProject != nil && storeID != "" &&
		storeID == h.loadedProject.storeID && projectSlug == h.loadedProject.projectSlug
}

func (h *Handler) matchesExpectedPendingGenerationLocked(id string, expected *uint64, observed bool) bool {
	if !observed {
		return false
	}
	record := h.changeSetLocked(id)
	if expected == nil {
		return id == ""
	}
	return record != nil && record.generation == *expected
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
		h.acceptedIndeterminate = false
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
			h.acceptedIndeterminate = false
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
	h.acceptedIndeterminate = false
	h.acceptedDiff = ""
	if err := h.loadChangeSetsLocked(ctx, replacement); err != nil {
		h.loadedStale = true
		return h.refreshResultLocked(errorRefreshFailed, http.StatusServiceUnavailable)
	}
	return h.currentArchitectureResponseLocked(), "", http.StatusOK
}

func (h *Handler) refreshResultLocked(actionError string, status int) (architectureResponse, string, int) {
	if actionError == errorRefreshFailed {
		h.acceptedIndeterminate = true
	}
	result := h.currentArchitectureResponseLocked()
	result.ActionError = actionError
	return result, actionError, status
}

func (h *Handler) markKnownNonCurrentLocked() {
	h.loadedStale = true
	h.acceptedIndeterminate = false
	if h.loadedProject != nil {
		h.loadedProject.validatedCurrent = nil
	}
}

func (h *Handler) discardChanges(response http.ResponseWriter, request *http.Request) {
	payload, ok := h.decodeArchitectureAction(response, request)
	if !ok {
		return
	}
	h.stateMutex.Lock()
	result, code := h.discardChangesLocked(request.Context(), payload)
	h.stateMutex.Unlock()
	if code != "" {
		writeJSON(response, http.StatusConflict, errorResponse{Code: code})
		return
	}
	writeJSON(response, http.StatusOK, result)
}

func (h *Handler) discardChangesLocked(ctx context.Context, payload architectureActionRequest) (architectureResponse, string) {
	if !h.matchesLoadedProjectLocked(payload.ProjectSlug, payload.StoreID) {
		return architectureResponse{}, errorArchitectureNotOpen
	}
	if h.loadedStale {
		return architectureResponse{}, errorArchitectureStale
	}
	if h.acceptedIndeterminate {
		return architectureResponse{}, errorRefreshFailed
	}
	if !h.matchesExpectedPendingGenerationLocked(payload.ChangeSetID, payload.ExpectedGeneration, payload.PendingGenerationObserved) {
		return architectureResponse{}, errorChangesElsewhere
	}
	record := h.changeSetLocked(payload.ChangeSetID)
	if record == nil || h.architecture.DeleteActiveChangeSet(ctx, record.storeID, record.id, record.refObject) != nil {
		return architectureResponse{}, errorChangesElsewhere
	}
	delete(h.changeSets, record.id)
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
	h.clearLoadedProjectLocked()
	return architectureResponse{}, ""
}

type changeSetCreateRequest struct {
	ProjectSlug      string `json:"project_slug"`
	StoreID          string `json:"store_id"`
	AcceptedRevision string `json:"accepted_revision"`
	Name             string `json:"name,omitempty"`
}

type changeSetEditRequest struct {
	ProjectSlug string `json:"project_slug"`
	StoreID     string `json:"store_id"`
	ChangeSetID string `json:"change_set_id"`
	Generation  uint64 `json:"generation"`
	Name        string `json:"name,omitempty"`
	Proposal    string `json:"proposal_markdown,omitempty"`
}

func (h *Handler) decodeBrowserJSON(response http.ResponseWriter, request *http.Request, value any) bool {
	if request.Header.Get("Origin") != h.expectedOrigin {
		writeJSON(response, http.StatusForbidden, errorResponse{Code: errorOriginMismatch})
		return false
	}
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBody)
	contents, err := io.ReadAll(request.Body)
	if err != nil || !utf8.Valid(contents) {
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorLookupFailed})
		return false
	}
	decoder := json.NewDecoder(strings.NewReader(string(contents)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(value) != nil || ensureJSONEnd(decoder) != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorLookupFailed})
		return false
	}
	return true
}

func (h *Handler) existingDurableChangeSetsLocked() []architecture.ChangeSet {
	values := make([]architecture.ChangeSet, 0, len(h.changeSets)+len(h.unavailableChangeSets))
	for _, record := range h.changeSets {
		values = append(values, h.durableChangeSet(record))
	}
	for _, record := range h.unavailableChangeSets {
		if record.Lifecycle == "active" && record.Name != "" {
			values = append(values, architecture.ChangeSet{Name: record.Name, Lifecycle: "active"})
		}
	}
	return values
}

func (h *Handler) createChangeSet(response http.ResponseWriter, request *http.Request) {
	var payload changeSetCreateRequest
	if !h.decodeBrowserJSON(response, request, &payload) {
		return
	}
	h.stateMutex.Lock()
	defer h.stateMutex.Unlock()
	if !h.matchesLoadedProjectLocked(payload.ProjectSlug, payload.StoreID) || h.loadedSnapshot == nil {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesElsewhere})
		return
	}
	if h.acceptedIndeterminate {
		writeJSON(response, http.StatusServiceUnavailable, errorResponse{Code: errorRefreshFailed})
		return
	}
	if h.loadedStale || payload.AcceptedRevision != h.loadedSnapshot.Revision() {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesElsewhere})
		return
	}
	id, name, err := h.architecture.NewChangeSet(h.existingDurableChangeSetsLocked(), payload.Name)
	if err != nil {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangeFailed})
		return
	}
	base := *h.loadedSnapshot
	candidate, err := h.architecture.ConstructCandidate(request.Context(), base, nil, architecture.CandidateComposition{})
	if err != nil {
		writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorChangeFailed})
		return
	}
	record := &pendingChangeSet{id: id, name: name, lifecycle: "active", storeID: base.StoreID(), baseRevision: base.Revision(), baseSnapshot: base, candidate: &candidate}
	if !h.persistActiveLocked(request.Context(), record, "") {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesElsewhere})
		return
	}
	result := h.responseForLoadedProjectLocked(base, record, false, h.acceptedDiff)
	result.ActionChangeSetID = id
	writeJSON(response, http.StatusCreated, result)
}

func (h *Handler) renameChangeSet(response http.ResponseWriter, request *http.Request) {
	h.editChangeSetText(response, request, true)
}

func (h *Handler) editChangeSetProposal(response http.ResponseWriter, request *http.Request) {
	h.editChangeSetText(response, request, false)
}

func (h *Handler) editChangeSetText(response http.ResponseWriter, request *http.Request, rename bool) {
	var payload changeSetEditRequest
	if !h.decodeBrowserJSON(response, request, &payload) {
		return
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
	if h.acceptedIndeterminate {
		writeJSON(response, http.StatusServiceUnavailable, errorResponse{Code: errorRefreshFailed})
		return
	}
	current := h.changeSetLocked(payload.ChangeSetID)
	if current == nil || current.generation != payload.Generation {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesElsewhere})
		return
	}
	proposed := clonePending(current)
	if rename {
		name := strings.TrimSpace(payload.Name)
		if architecture.ValidateChangeSetName(name) != nil {
			writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorChangeFailed})
			return
		}
		for id, record := range h.changeSets {
			if id != proposed.id && record.lifecycle == "active" && strings.EqualFold(record.name, name) {
				writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangeFailed})
				return
			}
		}
		for _, record := range h.unavailableChangeSets {
			if record.Lifecycle == "active" && record.Name != "" && strings.EqualFold(record.Name, name) {
				writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangeFailed})
				return
			}
		}
		if proposed.name == name {
			writeJSON(response, http.StatusOK, h.responseForLoadedProjectLocked(*h.loadedSnapshot, current, h.loadedStale, h.acceptedDiff))
			return
		}
		proposed.name = name
	} else {
		if !utf8.ValidString(payload.Proposal) {
			writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorChangeFailed})
			return
		}
		if proposed.proposal == payload.Proposal {
			writeJSON(response, http.StatusOK, h.responseForLoadedProjectLocked(*h.loadedSnapshot, current, h.loadedStale, h.acceptedDiff))
			return
		}
		proposed.proposal = payload.Proposal
	}
	proposed.generation++
	proposed.review = nil
	if !h.persistActiveLocked(request.Context(), proposed, current.refObject) {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesElsewhere})
		return
	}
	result := h.responseForLoadedProjectLocked(*h.loadedSnapshot, proposed, h.loadedStale, h.acceptedDiff)
	result.ActionChangeSetID = proposed.id
	writeJSON(response, http.StatusOK, result)
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
	ChangeSetID               string                 `json:"change_set_id,omitempty"`
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
	if h.acceptedIndeterminate {
		writeJSON(response, http.StatusServiceUnavailable, errorResponse{Code: errorRefreshFailed})
		return
	}
	accepted := *h.loadedSnapshot
	if payload.ChangeSetID == "" && (payload.ExpectedRevision == "" || payload.ExpectedRevision != accepted.Revision()) {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesElsewhere})
		return
	}
	if accepted.FormatVersion() < 2 {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesUnavailable})
		return
	}
	if !h.matchesExpectedPendingGenerationLocked(payload.ChangeSetID, payload.ExpectedGeneration, payload.PendingGenerationObserved) {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesElsewhere})
		return
	}
	current := h.changeSetLocked(payload.ChangeSetID)
	pending := clonePending(current)
	if pending == nil {
		pending = h.ensurePendingLocked(accepted, nil)
	}
	snapshot := pending.baseSnapshot

	var change architecture.ComponentChange
	changeIndex := -1
	if add {
		change = h.architecture.NewComponentChange(snapshot, pending.changes, payload.Title, payload.Description)
		change.Relationships = authoringRelationships(payload.Relationships)
		change.RelationshipsChanged = true
		homeDiagramID := payload.DiagramID
		if homeDiagramID == "" {
			homeDiagramID = snapshot.RootDiagramID()
		}
		if !pendingHasDiagram(snapshot, pending, homeDiagramID) {
			writeJSON(response, http.StatusBadRequest, errorResponse{Code: errorChangeFailed})
			return
		}
		pending.newComponentHomes = append(pending.newComponentHomes, architecture.NewComponentHome{
			ComponentID: change.ID, DiagramID: homeDiagramID,
		})
	} else {
		var found bool
		change, changeIndex, found = componentChangeLocked(snapshot, pending, payload.ComponentID)
		if !found {
			writeJSON(response, http.StatusNotFound, errorResponse{Code: errorComponentNotFound})
			return
		}
		changed := false
		if payload.TitleChanged && change.Title != payload.Title {
			change.Title = payload.Title
			change.TitleChanged = true
			changed = true
		}
		if payload.DescriptionChanged && change.Description != payload.Description {
			change.Description = payload.Description
			change.DescriptionChanged = true
			changed = true
		}
		relationships := authoringRelationships(payload.Relationships)
		if payload.RelationshipsChanged && !sameAuthoringRelationships(change.Relationships, relationships) {
			change.Relationships = relationships
			change.RelationshipsChanged = true
			changed = true
		}
		if !changed {
			writeJSON(response, http.StatusOK, h.responseForLoadedProjectLocked(accepted, current, h.loadedStale, h.acceptedDiff))
			return
		}
	}
	pending = h.keepComponentChangeLocked(request.Context(), snapshot, pending, change, changeIndex)
	if pending.validationCode == "change_unavailable" {
		writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorChangeFailed})
		return
	}
	expectedObject := ""
	if current != nil {
		expectedObject = current.refObject
	}
	if !h.persistActiveLocked(request.Context(), pending, expectedObject) {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesElsewhere})
		return
	}
	result := h.responseForLoadedProjectLocked(accepted, pending, h.loadedStale, h.acceptedDiff)
	result.ActionChangeSetID = pending.id
	writeJSON(response, http.StatusOK, result)
}

func (h *Handler) ensurePendingLocked(snapshot architecture.Snapshot, pending *pendingChangeSet) *pendingChangeSet {
	if pending != nil {
		return pending
	}
	id, name, _ := h.architecture.NewChangeSet(h.existingDurableChangeSetsLocked(), "")
	pending = &pendingChangeSet{id: id, name: name, lifecycle: "active", storeID: snapshot.StoreID(), baseRevision: snapshot.Revision(), baseSnapshot: snapshot}
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

func (h *Handler) constructCandidate(ctx context.Context, snapshot architecture.Snapshot, pending *pendingChangeSet, initialize bool) (architecture.Candidate, error) {
	if h.candidateConstructionFailure != nil {
		if err := h.candidateConstructionFailure(); err != nil {
			return architecture.Candidate{}, err
		}
	}
	composition := architecture.CandidateComposition{
		ArchitectureVersion: pending.architectureVersion,
		NodePositions:       pending.nodePositions,
		NodeSizes:           pending.nodeSizes,
		EdgeRoutes:          pending.edgeRoutes,
		DetailReassignments: pending.detailReassignments,
		NewComponentHomes:   pending.newComponentHomes,
		DetailDiagrams:      pending.detailDiagrams,
		DiagramTitles:       pending.diagramTitles,
		HomeMoves:           pending.homeMoves,
		References:          pending.references,
	}
	if !initialize {
		return h.architecture.ConstructCandidate(ctx, snapshot, pending.changes, composition)
	}
	candidate, err := h.architecture.PrepareCandidate(ctx, snapshot, pending.changes, &composition)
	if err == nil {
		pending.architectureVersion, pending.nodePositions = composition.ArchitectureVersion, composition.NodePositions
		pending.nodeSizes = composition.NodeSizes
		pending.edgeRoutes = composition.EdgeRoutes
	}
	return candidate, err
}

type diagramMutationRequest struct {
	Width                     *int    `json:"width,omitempty"`
	Height                    *int    `json:"height,omitempty"`
	X                         *int    `json:"x,omitempty"`
	Y                         *int    `json:"y,omitempty"`
	AnchorComponentID         string  `json:"anchor_component_id,omitempty"`
	ProjectSlug               string  `json:"project_slug"`
	StoreID                   string  `json:"store_id"`
	ExpectedRevision          string  `json:"expected_revision"`
	ExpectedGeneration        *uint64 `json:"expected_pending_generation,omitempty"`
	PendingGenerationObserved bool    `json:"pending_generation_observed,omitempty"`
	DiagramID                 string  `json:"diagram_id,omitempty"`
	ComponentID               string  `json:"component_id,omitempty"`
	Title                     string  `json:"title,omitempty"`
	ChangeSetID               string  `json:"change_set_id,omitempty"`
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
		pending = h.ensurePendingLocked(snapshot, nil)
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
	if h.acceptedIndeterminate {
		writeJSON(response, http.StatusServiceUnavailable, errorResponse{Code: errorRefreshFailed})
		return architecture.Snapshot{}, nil, false
	}
	accepted := *h.loadedSnapshot
	if payload.ChangeSetID == "" && (payload.ExpectedRevision == "" || payload.ExpectedRevision != accepted.Revision()) {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesElsewhere})
		return architecture.Snapshot{}, nil, false
	}
	if accepted.FormatVersion() < 2 {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesUnavailable})
		return architecture.Snapshot{}, nil, false
	}
	if !h.matchesExpectedPendingGenerationLocked(payload.ChangeSetID, payload.ExpectedGeneration, payload.PendingGenerationObserved) {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesElsewhere})
		return architecture.Snapshot{}, nil, false
	}
	current := h.changeSetLocked(payload.ChangeSetID)
	if current != nil {
		return current.baseSnapshot, clonePending(current), true
	}
	return accepted, nil, true
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
	var previous []architecture.ComponentChange
	var previousComposition architecture.CandidateComposition
	if old := h.changeSets[pending.id]; old != nil {
		previous = old.changes
		previousComposition = h.durableChangeSet(old).Composition
	}
	reset := architecture.ResetChangedRouteCounts(snapshot, previous, pending.changes, h.durableChangeSet(pending).Composition)
	reset = architecture.ResetLostRouteVisibility(snapshot, previous, pending.changes, previousComposition, reset)
	pending.edgeRoutes = reset.EdgeRoutes
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
	candidate, err := h.constructCandidate(ctx, snapshot, pending, true)
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

func (h *Handler) editDiagramTitleLocked(ctx context.Context, snapshot architecture.Snapshot, pending *pendingChangeSet, diagramID, title string) (*pendingChangeSet, bool, string) {
	if !pendingHasDiagram(snapshot, pending, diagramID) {
		return pending, false, changeTargetNotFound
	}
	pending = h.ensurePendingLocked(snapshot, pending)
	for index := range pending.detailDiagrams {
		if pending.detailDiagrams[index].ID == diagramID {
			if pending.detailDiagrams[index].Title == title {
				return pending, true, ""
			}
			pending.detailDiagrams[index].Title = title
			h.rebuildPendingLocked(ctx, snapshot, pending)
			if pendingOperationFailed(pending) {
				return pending, false, changeOperationFailed
			}
			return pending, false, ""
		}
	}
	baseTitle := ""
	for _, diagram := range snapshot.DiagramProjections() {
		if diagram.ID == diagramID {
			baseTitle = diagram.Title
			break
		}
	}
	for index := range pending.diagramTitles {
		if pending.diagramTitles[index].DiagramID == diagramID {
			if pending.diagramTitles[index].Title == title {
				return pending, true, ""
			}
			if title == baseTitle {
				pending.diagramTitles = append(pending.diagramTitles[:index], pending.diagramTitles[index+1:]...)
			} else {
				pending.diagramTitles[index].Title = title
			}
			h.rebuildPendingLocked(ctx, snapshot, pending)
			if pendingOperationFailed(pending) {
				return pending, false, changeOperationFailed
			}
			return pending, false, ""
		}
	}
	if baseTitle == title {
		return pending, true, ""
	}
	pending.diagramTitles = append(pending.diagramTitles, architecture.DiagramTitleChange{DiagramID: diagramID, Title: title})
	h.rebuildPendingLocked(ctx, snapshot, pending)
	if pendingOperationFailed(pending) {
		return pending, false, changeOperationFailed
	}
	return pending, false, ""
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
	proposedPointer := clonePending(pending)
	if proposedPointer == nil {
		proposedPointer = h.ensurePendingLocked(snapshot, nil)
	}
	proposed := *proposedPointer
	proposed.references = referenceChangesWithoutPair(proposed.references, diagramID, componentID)
	setReferenceChange(&proposed, currentHome, componentID, false)
	proposed.homeMoves = homeMovesWithoutComponent(proposed.homeMoves, componentID)
	if initialComponentHome(snapshot, &proposed, componentID) != diagramID {
		proposed.homeMoves = append(proposed.homeMoves, architecture.ComponentHomeMove{ComponentID: componentID, DiagramID: diagramID})
	}
	h.rebuildPendingLocked(ctx, snapshot, &proposed)
	if pendingOperationFailed(&proposed) {
		return pending, false, changeOperationFailed
	}
	if proposed.candidate == nil {
		return pending, false, changeTargetIneligible
	}
	return &proposed, false, ""
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
	pending, _, _, operationError := h.createDetailDiagramLocked(request.Context(), snapshot, pending, payload.ComponentID, payload.Title)
	if operationError != "" {
		if operationError == changeOperationFailed {
			writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorChangeFailed})
			return
		}
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangeFailed})
		return
	}
	h.persistBrowserChangeSetMutationLocked(response, request.Context(), payload.ChangeSetID, pending)
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
	pending, unchanged, operationError := h.editDiagramTitleLocked(request.Context(), snapshot, pending, payload.DiagramID, payload.Title)
	if operationError != "" {
		if operationError == changeOperationFailed {
			writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorChangeFailed})
			return
		}
		writeJSON(response, http.StatusNotFound, errorResponse{Code: errorChangeFailed})
		return
	}
	if unchanged {
		writeJSON(response, http.StatusOK, h.responseForLoadedProjectLocked(*h.loadedSnapshot, h.changeSetLocked(payload.ChangeSetID), h.loadedStale, h.acceptedDiff))
		return
	}
	h.persistBrowserChangeSetMutationLocked(response, request.Context(), payload.ChangeSetID, pending)
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
	pending, unchanged, operationError := h.moveComponentHomeLocked(request.Context(), snapshot, pending, payload.ComponentID, payload.DiagramID)
	if operationError == "" {
		if unchanged {
			selected := h.changeSetLocked(payload.ChangeSetID)
			writeJSON(response, http.StatusOK, h.responseForLoadedProjectLocked(*h.loadedSnapshot, selected, h.loadedStale, h.acceptedDiff))
			return
		}
		h.persistBrowserChangeSetMutationLocked(response, request.Context(), payload.ChangeSetID, pending)
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

func (h *Handler) persistBrowserChangeSetMutationLocked(response http.ResponseWriter, ctx context.Context, currentID string, pending *pendingChangeSet) {
	expectedObject := ""
	if current := h.changeSetLocked(currentID); current != nil {
		expectedObject = current.refObject
	}
	if pending == nil || pendingOperationFailed(pending) {
		writeJSON(response, http.StatusInternalServerError, errorResponse{Code: errorChangeFailed})
		return
	}
	if !h.persistActiveLocked(ctx, pending, expectedObject) {
		writeJSON(response, http.StatusConflict, errorResponse{Code: errorChangesElsewhere})
		return
	}
	result := h.responseForLoadedProjectLocked(*h.loadedSnapshot, pending, h.loadedStale, h.acceptedDiff)
	result.ActionChangeSetID = pending.id
	writeJSON(response, http.StatusOK, result)
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
	return len(pending.edgeRoutes) == 0 && len(pending.nodeSizes) == 0 && len(pending.nodePositions) == 0 && (pending.architectureVersion == 0 || pending.architectureVersion == pending.baseSnapshot.FormatVersion()) && len(pending.changes) == 0 && len(pending.newComponentHomes) == 0 &&
		len(pending.detailDiagrams) == 0 && len(pending.diagramTitles) == 0 && len(pending.homeMoves) == 0 && len(pending.references) == 0 && len(pending.detailReassignments) == 0
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
	pending, operationError := h.changeReferenceLocked(request.Context(), snapshot, pending, payload.DiagramID, payload.ComponentID, present)
	if operationError == "" {
		h.persistBrowserChangeSetMutationLocked(response, request.Context(), payload.ChangeSetID, pending)
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

func sameAuthoringRelationships(left, right []architecture.AuthoringRelationship) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
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
	if h.acceptedIndeterminate {
		return architectureResponse{}, errorRefreshFailed, http.StatusConflict
	}
	if !h.matchesExpectedPendingGenerationLocked(payload.ChangeSetID, payload.ExpectedGeneration, payload.PendingGenerationObserved) {
		return architectureResponse{}, errorChangesElsewhere, http.StatusConflict
	}
	current := h.changeSetLocked(payload.ChangeSetID)
	if current == nil {
		return architectureResponse{}, errorReviewFailed, http.StatusConflict
	}
	proposed := clonePending(current)
	base := proposed.baseSnapshot
	candidate, err := h.constructCandidate(ctx, base, proposed, false)
	proposed.review = nil
	proposed.reviewBlocker = ""
	if err != nil {
		h.recordCandidateValidation(proposed, err)
		status := http.StatusUnprocessableEntity
		if proposed.validationCode == "change_unavailable" {
			status = http.StatusInternalServerError
		} else {
			proposed.reviewBlocker = proposed.validationCode
		}
		result := h.responseForLoadedProjectLocked(*h.loadedSnapshot, proposed, false, h.acceptedDiff)
		result.ActionError = errorReviewFailed
		return result, errorReviewFailed, status
	}
	diff, err := h.architecture.CandidateDiff(ctx, base, candidate)
	if err != nil {
		result := h.responseForLoadedProjectLocked(*h.loadedSnapshot, current, false, h.acceptedDiff)
		result.ActionError = errorReviewFailed
		return result, errorReviewFailed, http.StatusInternalServerError
	}
	proposed.candidate = &candidate
	proposed.validationCode = ""
	proposed.validationItem = ""
	proposed.validationRelationshipPosition = 0
	proposed.validationRelationshipField = ""
	proposed.validationDiagram = ""
	proposed.validationDiagramField = ""
	proposed.review = &reviewBinding{
		baseRevision: base.Revision(), candidateTree: candidate.Tree(), generation: proposed.generation,
		diff: string(diff), candidate: candidate,
	}
	if current.review != nil && current.review.baseRevision == proposed.review.baseRevision &&
		current.review.candidateTree == proposed.review.candidateTree && current.review.generation == proposed.review.generation {
		// The durable binding already identifies this exact review. Refresh only
		// the derived in-memory presentation after a process restart; do not
		// manufacture another semantically identical state commit.
		current.review.diff = proposed.review.diff
		current.review.candidate = proposed.review.candidate
		result := h.responseForLoadedProjectLocked(*h.loadedSnapshot, current, false, h.acceptedDiff)
		result.ActionChangeSetID = current.id
		return result, "", http.StatusOK
	}
	object, persistErr := h.architecture.WriteActiveChangeSet(ctx, proposed.storeID, h.durableChangeSet(proposed), current.refObject)
	if persistErr != nil {
		return architectureResponse{}, errorChangesElsewhere, http.StatusConflict
	}
	proposed.refObject = object
	h.changeSets[proposed.id] = proposed
	// Build the complete immutable visual presentation while the binding and
	// pending generation are protected by the same concrete state lock. JSON
	// serialization can then proceed without blocking invalidating mutations.
	result := h.responseForLoadedProjectLocked(*h.loadedSnapshot, proposed, false, h.acceptedDiff)
	result.ActionChangeSetID = proposed.id
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
	if h.acceptedIndeterminate {
		return architectureResponse{}, errorRefreshFailed, http.StatusServiceUnavailable, false
	}
	snapshot := *h.loadedSnapshot
	if receipt := h.changeSets[payload.ChangeSetID]; receipt != nil && receipt.lifecycle == "applied" && appliedReceiptMatches(receipt, payload) {
		result := h.responseForLoadedProjectLocked(snapshot, receipt, false, h.acceptedDiff)
		result.ActionChangeSetID = receipt.id
		result.AlreadyApplied = true
		return result, "", http.StatusOK, false
	}
	pending := h.changeSetLocked(payload.ChangeSetID)
	if pending == nil || pending.review == nil {
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
	if pending.storeID != snapshot.StoreID() ||
		review.baseRevision != pending.baseRevision || review.generation != pending.generation ||
		pending.candidate == nil || review.candidateTree != pending.candidate.Tree() || review.candidateTree != review.candidate.Tree() {
		result := h.responseForLoadedProjectLocked(snapshot, pending, false, h.acceptedDiff)
		result.ActionError = errorReviewChanged
		return result, errorReviewChanged, http.StatusConflict, false
	}
	// Once the human confirms, the local authority transition must reach a
	// classified boundary even if the browser disconnects before the response.
	transitionContext, cancelTransition := context.WithTimeout(context.Background(), architectureTransitionTimeout)
	defer cancelTransition()

	base := pending.baseSnapshot
	observed, present, err := h.architecture.AcceptedRevision(transitionContext, base)
	if err != nil || !present {
		h.loadedStale = true
		result := h.responseForLoadedProjectLocked(snapshot, pending, true, h.acceptedDiff)
		result.ActionError = errorUpdateUncertain
		return result, errorUpdateUncertain, http.StatusConflict, false
	}
	if observed != review.baseRevision {
		if snapshot.Revision() != observed {
			h.markKnownNonCurrentLocked()
		}
		result := h.responseForLoadedProjectLocked(snapshot, pending, h.loadedStale, h.acceptedDiff)
		result.ActionError = errorArchitectureStale
		return result, errorArchitectureStale, http.StatusConflict, true
	}
	if review.diff == "" {
		result := h.responseForLoadedProjectLocked(snapshot, pending, false, h.acceptedDiff)
		result.ActionError = errorReviewFailed
		return result, errorReviewFailed, http.StatusConflict, false
	}

	successor, err := h.architecture.CreateSuccessor(transitionContext, base, review.candidate)
	if err != nil {
		result := h.responseForLoadedProjectLocked(snapshot, pending, false, h.acceptedDiff)
		result.ActionError = errorUpdateFailed
		return result, errorUpdateFailed, http.StatusInternalServerError, false
	}
	if h.beforeAcceptedCAS != nil {
		h.beforeAcceptedCAS(successor)
	}
	applied := clonePending(pending)
	applied.lifecycle = "applied"
	applied.appliedRevision = successor
	appliedObject, err := h.architecture.PrepareAppliedChangeSet(transitionContext, pending.storeID, h.durableChangeSet(applied))
	if err != nil {
		result := h.responseForLoadedProjectLocked(snapshot, pending, false, h.acceptedDiff)
		result.ActionError = errorUpdateFailed
		return result, errorUpdateFailed, http.StatusInternalServerError, false
	}
	updateErr := h.architecture.AcceptChangeSet(transitionContext, pending.storeID, base.Revision(), successor, pending.id, pending.refObject, appliedObject)
	if updateErr == nil && h.acceptedUpdateReportFailure != nil {
		updateErr = h.acceptedUpdateReportFailure()
	}
	if updateErr != nil {
		observationContext, cancelObservation := context.WithTimeout(context.Background(), architectureObservationTimeout)
		recoveredRecords, _, recoveryErr := h.architecture.LoadChangeSets(observationContext, pending.storeID)
		if recoveryErr == nil {
			for _, recovered := range recoveredRecords {
				candidateTree := ""
				if recovered.Candidate != nil {
					candidateTree = recovered.Candidate.Tree()
				}
				if recovered.ID == payload.ChangeSetID && recovered.Lifecycle == "applied" && recovered.BaseRevision == payload.BaseRevision && recovered.Generation == payload.Generation && candidateTree == payload.CandidateTree {
					recoveredAccepted, loadErr := h.architecture.LoadAccepted(observationContext, pending.storeID)
					if loadErr == nil {
						h.loadedSnapshot = &recoveredAccepted
						h.loadedStale = false
						if h.loadChangeSetsLocked(observationContext, recoveredAccepted) == nil {
							receipt := h.changeSets[payload.ChangeSetID]
							result := h.responseForLoadedProjectLocked(recoveredAccepted, receipt, false, h.acceptedDiff)
							result.ActionChangeSetID = payload.ChangeSetID
							result.AlreadyApplied = true
							cancelObservation()
							return result, "", http.StatusOK, false
						}
					}
				}
			}
		}
		observed, present, observeErr := h.architecture.AcceptedRevision(observationContext, base)
		cancelObservation()
		if observeErr != nil || !present {
			h.loadedStale = true
			result := h.responseForLoadedProjectLocked(snapshot, pending, true, h.acceptedDiff)
			result.ActionError = errorUpdateUncertain
			return result, errorUpdateUncertain, http.StatusInternalServerError, false
		}
		if observed != review.baseRevision {
			if snapshot.Revision() != observed {
				h.markKnownNonCurrentLocked()
			}
			result := h.responseForLoadedProjectLocked(snapshot, pending, h.loadedStale, h.acceptedDiff)
			result.ActionError = errorArchitectureStale
			return result, errorArchitectureStale, http.StatusConflict, true
		}
		if observed == review.baseRevision {
			result := h.responseForLoadedProjectLocked(snapshot, pending, false, h.acceptedDiff)
			result.ActionError = errorUpdateFailed
			return result, errorUpdateFailed, http.StatusInternalServerError, false
		}
	}
	// The atomic transaction made the successor and applied receipt canonical.
	applied.refObject = appliedObject
	h.changeSets[pending.id] = applied
	h.acceptedDiff = review.diff
	acceptedSnapshot := review.candidate.SnapshotAt(successor)
	if h.publicationFailure != nil {
		if err := h.publicationFailure(); err != nil {
			recoveryContext, cancelRecovery := context.WithTimeout(context.Background(), architectureTransitionTimeout)
			recovered, loadErr := h.architecture.LoadAccepted(recoveryContext, snapshot.StoreID())
			if loadErr == nil {
				h.loadedSnapshot = &recovered
				h.loadedProject.projectName = recovered.ProjectName()
				h.loadedProject.projectSlug = recovered.ProjectSlug()
				h.loadedProject.validatedCurrent = nil
				h.loadedStale = h.loadChangeSetsLocked(recoveryContext, recovered) != nil
			} else {
				h.loadedStale = true
			}
			cancelRecovery()
			result := h.responseForLoadedProjectLocked(*h.loadedSnapshot, h.changeSets[payload.ChangeSetID], h.loadedStale, h.acceptedDiff)
			result.ActionError = errorUpdatedReload
			return result, errorUpdatedReload, http.StatusInternalServerError, false
		}
	}
	h.loadedSnapshot = &acceptedSnapshot
	h.loadedProject.projectName = acceptedSnapshot.ProjectName()
	h.loadedProject.projectSlug = acceptedSnapshot.ProjectSlug()
	h.loadedProject.validatedCurrent = nil
	h.loadedStale = false
	h.acceptedIndeterminate = false
	result := h.responseForLoadedProjectLocked(acceptedSnapshot, applied, false, h.acceptedDiff)
	result.ActionChangeSetID = applied.id
	return result, "", http.StatusOK, false
}

func appliedReceiptMatches(record *pendingChangeSet, payload acceptChangesRequest) bool {
	return record.baseRevision == payload.BaseRevision && record.generation == payload.Generation && record.candidate != nil && record.candidate.Tree() == payload.CandidateTree && record.review != nil
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
	if pending.validationCode != "change_unavailable" {
		pending.reviewBlocker = pending.validationCode
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
