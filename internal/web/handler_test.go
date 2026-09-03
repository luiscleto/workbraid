package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"workbraid/internal/architecture"
)

const testOrigin = "http://127.0.0.1:8080"

func testActiveChangeSet(handler *Handler) *pendingChangeSet {
	for _, record := range handler.changeSets {
		if record.lifecycle == "active" {
			return record
		}
	}
	return nil
}

func TestSlugNativeCreateCatalogOpenAndReload(t *testing.T) {
	data := t.TempDir()
	_, handler := newHandler(testOrigin, testUI(t), data)
	first := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "  Example Project  "}))
	if first.ProjectName != "Example Project" || first.ProjectSlug != "example-project" || first.StoreID == "" || first.FormatVersion != 2 || first.RootDiagramID == "" {
		t.Fatalf("first = %+v", first)
	}
	second := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Example Project"}))
	if second.ProjectSlug != "example-project-2" || second.StoreID == first.StoreID {
		t.Fatalf("second = %+v", second)
	}
	if _, err := os.Stat(filepath.Join(data, "workbraid.db")); !os.IsNotExist(err) {
		t.Fatalf("obsolete database exists: %v", err)
	}

	catalog := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	handler.ServeHTTP(catalog, request)
	if catalog.Code != http.StatusOK || !strings.Contains(catalog.Body.String(), `"slug":"example-project"`) || !strings.Contains(catalog.Body.String(), `"slug":"example-project-2"`) {
		t.Fatalf("catalog status=%d body=%s", catalog.Code, catalog.Body.String())
	}
	opened := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/open", map[string]any{"project_slug": first.ProjectSlug}))
	if opened.StoreID != first.StoreID || opened.Revision != first.Revision {
		t.Fatalf("opened = %+v, first = %+v", opened, first)
	}
	_, fresh := newHandler(testOrigin, testUI(t), data)
	reloaded := decodeArchitectureResponse(t, postJSONRequest(t, fresh, "/api/projects/open", map[string]any{"project_slug": first.ProjectSlug}))
	if reloaded.StoreID != first.StoreID || reloaded.Revision != first.Revision {
		t.Fatalf("reloaded = %+v", reloaded)
	}

	unknown := postJSONRequest(t, fresh, "/api/projects/open", map[string]any{"project_slug": "missing"})
	if unknown.Code != http.StatusNotFound || !strings.Contains(unknown.Body.String(), errorProjectNotFound) {
		t.Fatalf("unknown status=%d body=%s", unknown.Code, unknown.Body.String())
	}
	if entries, err := os.ReadDir(filepath.Join(data, "architecture")); err != nil || len(entries) != 2 {
		t.Fatalf("unknown open wrote stores: entries=%d err=%v", len(entries), err)
	}
}

func TestCatalogCreationIsAtomicAndProjectSwitchRaceUsesStoreIdentity(t *testing.T) {
	data := t.TempDir()
	state, handler := newHandler(testOrigin, testUI(t), data)
	first := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Same"}))
	second := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Same"}))
	decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/open", map[string]any{"project_slug": first.ProjectSlug}))

	start := make(chan struct{})
	results := make(chan int, 2)
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		<-start
		response := postJSONRequest(t, handler, "/api/architecture/components/add", observedComponentMutation(first, componentMutationRequest{Title: "Worker", DiagramID: first.RootDiagramID}))
		results <- response.Code
	}()
	go func() {
		defer wait.Done()
		<-start
		response := postJSONRequest(t, handler, "/api/projects/open", map[string]any{"project_slug": second.ProjectSlug})
		results <- response.Code
	}()
	close(start)
	wait.Wait()
	close(results)
	statuses := map[int]int{}
	for status := range results {
		statuses[status]++
	}
	if statuses[http.StatusOK]+statuses[http.StatusConflict] != 2 || statuses[http.StatusOK] < 1 {
		t.Fatalf("race statuses = %v", statuses)
	}
	state.stateMutex.Lock()
	defer state.stateMutex.Unlock()
	if state.loadedProject == nil || state.loadedProject.storeID != second.StoreID {
		t.Fatalf("project switch lost race authority: project=%+v", state.loadedProject)
	}
	if pending := testActiveChangeSet(state); pending != nil {
		t.Fatalf("Project A change set leaked into Project B memory: project=%+v change-set=%+v", state.loadedProject, pending)
	}
}

func TestBrowserActionsRequireExactSynchronizedRevisionAndPendingObservation(t *testing.T) {
	_, handler := newHandler(testOrigin, testUI(t), t.TempDir())
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Browser preconditions"}))

	omittedInitial := postJSONRequest(t, handler, "/api/architecture/components/add", componentMutationRequest{
		ProjectSlug: created.ProjectSlug, StoreID: created.StoreID, ExpectedRevision: created.Revision,
		DiagramID: created.RootDiagramID, Title: "Omitted",
	})
	if omittedInitial.Code != http.StatusConflict || !strings.Contains(omittedInitial.Body.String(), errorChangesElsewhere) {
		t.Fatalf("omitted initial observation status=%d body=%s", omittedInitial.Code, omittedInitial.Body.String())
	}
	kept := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/add", observedComponentMutation(created, componentMutationRequest{
		DiagramID: created.RootDiagramID, Title: "Observed",
	})))
	if kept.Changes == nil || kept.Changes.Generation != 1 {
		t.Fatalf("initial null observation did not create generation one: %+v", kept.Changes)
	}

	requests := []struct {
		name string
		path string
		body any
	}{
		{name: "component mutation omission", path: "/api/architecture/components/edit", body: componentMutationRequest{
			ProjectSlug: kept.ProjectSlug, StoreID: kept.StoreID, ExpectedRevision: kept.Revision,
			ComponentID: kept.Changes.Components[0].ID, Title: "Late", TitleChanged: true,
		}},
		{name: "diagram mutation omission", path: "/api/architecture/diagrams/title", body: diagramMutationRequest{
			ProjectSlug: kept.ProjectSlug, StoreID: kept.StoreID, ExpectedRevision: kept.Revision,
			DiagramID: kept.RootDiagramID, Title: "Late",
		}},
		{name: "review omission", path: "/api/architecture/review", body: architectureActionRequest{
			ProjectSlug: kept.ProjectSlug, StoreID: kept.StoreID, ExpectedRevision: kept.Revision,
		}},
		{name: "discard omission", path: "/api/architecture/discard", body: architectureActionRequest{
			ProjectSlug: kept.ProjectSlug, StoreID: kept.StoreID, ExpectedRevision: kept.Revision,
		}},
		{name: "refresh revision omission", path: "/api/architecture/refresh", body: architectureActionRequest{
			ProjectSlug: kept.ProjectSlug, StoreID: kept.StoreID, ExpectedGeneration: pendingGenerationFor(kept), PendingGenerationObserved: true,
		}},
		{name: "stale generation", path: "/api/architecture/discard", body: architectureActionRequest{
			ProjectSlug: kept.ProjectSlug, StoreID: kept.StoreID, ExpectedRevision: kept.Revision, PendingGenerationObserved: true,
		}},
	}
	for _, test := range requests {
		t.Run(test.name, func(t *testing.T) {
			response := postJSONRequest(t, handler, test.path, test.body)
			if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), errorChangesElsewhere) {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
	current := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/review", observedAction(kept)))
	if current.Changes == nil || current.Changes.Review == nil || current.Changes.Generation != 1 {
		t.Fatalf("rejected requests changed synchronized state: %+v", current.Changes)
	}
}

func TestReferenceMutationAndDiscardShareOneStateBoundary(t *testing.T) {
	data := t.TempDir()
	state, handler := newHandler(testOrigin, testUI(t), data)
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Reference race"}))
	base := *state.loadedSnapshot
	anchor := state.architecture.NewComponentChange(base, nil, "Anchor", "")
	target := state.architecture.NewComponentChange(base, []architecture.ComponentChange{anchor}, "Target", "")
	detail := base.NewDetailDiagramChange(nil, "Detail", anchor.ID)
	candidate, err := state.architecture.ConstructCandidate(context.Background(), base, []architecture.ComponentChange{anchor, target}, architecture.CandidateComposition{
		NewComponentHomes: []architecture.NewComponentHome{{ComponentID: anchor.ID, DiagramID: base.RootDiagramID()}, {ComponentID: target.ID, DiagramID: base.RootDiagramID()}},
		DetailDiagrams:    []architecture.DetailDiagramChange{detail},
		HomeMoves:         []architecture.ComponentHomeMove{{ComponentID: target.ID, DiagramID: detail.ID}},
	})
	if err != nil {
		t.Fatal(err)
	}
	accepted := acceptArchitectureCandidate(t, state.architecture, base, candidate)
	opened := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/open", map[string]any{"project_slug": created.ProjectSlug}))
	if opened.Revision != accepted.Revision() {
		t.Fatalf("opened revision=%s want=%s", opened.Revision, accepted.Revision())
	}

	start := make(chan struct{})
	results := make(chan *httptest.ResponseRecorder, 2)
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		<-start
		results <- postJSONRequest(t, handler, "/api/architecture/diagrams/show-component", observedDiagramMutation(opened, diagramMutationRequest{DiagramID: opened.RootDiagramID, ComponentID: target.ID}))
	}()
	go func() {
		defer wait.Done()
		<-start
		results <- postJSONRequest(t, handler, "/api/architecture/discard", observedAction(opened))
	}()
	close(start)
	wait.Wait()
	close(results)
	successes := 0
	for response := range results {
		if response.Code == http.StatusOK {
			successes++
		} else if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), errorChangesElsewhere) {
			t.Fatalf("race status=%d body=%s", response.Code, response.Body.String())
		}
	}
	if successes < 1 {
		t.Fatal("neither synchronized action succeeded")
	}
	state.stateMutex.Lock()
	defer state.stateMutex.Unlock()
	if pending := testActiveChangeSet(state); pending != nil {
		if pending.candidate == nil || pending.generation != 1 {
			t.Fatalf("incoherent final change set = %+v", pending)
		}
		projection := projectSnapshot(pending.candidate.Snapshot(), "")
		if role(&projection, opened.RootDiagramID, target.ID) != "reference" {
			t.Fatalf("incoherent final reference projection = %+v", projection)
		}
	}
}

func TestReferenceHandlersUseOneCandidateAndNormalizeRepeatedHomeMoves(t *testing.T) {
	data := t.TempDir()
	state, handler := newHandler(testOrigin, testUI(t), data)
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "References"}))
	base := *state.loadedSnapshot
	manager := state.architecture
	anchorB := manager.NewComponentChange(base, nil, "Anchor B", "")
	anchorC := manager.NewComponentChange(base, []architecture.ComponentChange{anchorB}, "Anchor C", "")
	moving := manager.NewComponentChange(base, []architecture.ComponentChange{anchorB, anchorC}, "Moving", "")
	b := base.NewDetailDiagramChange(nil, "B", anchorB.ID)
	c := base.NewDetailDiagramChange([]architecture.DetailDiagramChange{b}, "C", anchorC.ID)
	candidate, err := manager.ConstructCandidate(context.Background(), base, []architecture.ComponentChange{anchorB, anchorC, moving}, architecture.CandidateComposition{
		NewComponentHomes: []architecture.NewComponentHome{{ComponentID: anchorB.ID, DiagramID: base.RootDiagramID()}, {ComponentID: anchorC.ID, DiagramID: base.RootDiagramID()}, {ComponentID: moving.ID, DiagramID: base.RootDiagramID()}},
		DetailDiagrams:    []architecture.DetailDiagramChange{b, c},
		References:        []architecture.ReferenceAppearanceChange{{DiagramID: b.ID, ComponentID: moving.ID, Present: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	commit, err := manager.CreateSuccessor(context.Background(), base, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.AdvanceAccepted(context.Background(), base, commit); err != nil {
		t.Fatal(err)
	}
	opened := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/open", map[string]any{"project_slug": created.ProjectSlug}))

	moveB := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/move-home", observedDiagramMutation(opened, diagramMutationRequest{DiagramID: b.ID, ComponentID: moving.ID})))
	moveC := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/move-home", observedDiagramMutation(moveB, diagramMutationRequest{DiagramID: c.ID, ComponentID: moving.ID})))
	if moveB.Changes == nil || moveC.Changes == nil || moveC.Changes.Candidate == nil {
		t.Fatalf("moves did not remain one candidate: B=%+v C=%+v", moveB.Changes, moveC.Changes)
	}
	if role(moveC.Changes.Candidate, b.ID, moving.ID) != "" || role(moveC.Changes.Candidate, c.ID, moving.ID) != "home" {
		t.Fatalf("reference resurrected: B=%q C=%q", role(moveC.Changes.Candidate, b.ID, moving.ID), role(moveC.Changes.Candidate, c.ID, moving.ID))
	}
	shown := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/diagrams/show-component", observedDiagramMutation(moveC, diagramMutationRequest{DiagramID: b.ID, ComponentID: moving.ID})))
	if shown.Changes == nil || shown.Changes.Candidate == nil || role(shown.Changes.Candidate, b.ID, moving.ID) != "reference" || role(shown.Changes.Candidate, c.ID, moving.ID) != "home" {
		t.Fatalf("show-back failed: %+v", shown.Changes)
	}
	firstReview := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/review", observedAction(shown)))
	if firstReview.Changes == nil || firstReview.Changes.Review == nil {
		t.Fatalf("first review missing: %+v", firstReview.Changes)
	}
	stopped := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/diagrams/stop-showing-component", observedDiagramMutation(firstReview, diagramMutationRequest{DiagramID: b.ID, ComponentID: moving.ID})))
	if stopped.Changes == nil || stopped.Changes.Candidate == nil || stopped.Changes.Review != nil || role(stopped.Changes.Candidate, b.ID, moving.ID) != "" {
		t.Fatalf("stop failed: %+v", stopped.Changes)
	}
	invalidated := postJSONRequest(t, handler, "/api/architecture/accept", acceptChangesRequest{
		ProjectSlug: opened.ProjectSlug, StoreID: opened.StoreID, ChangeSetID: firstReview.Changes.ID, BaseRevision: firstReview.Changes.Review.BaseRevision,
		CandidateTree: firstReview.Changes.Review.CandidateTree, Generation: firstReview.Changes.Review.Generation,
	})
	if invalidated.Code != http.StatusConflict || !strings.Contains(invalidated.Body.String(), errorReviewFailed) {
		t.Fatalf("invalidated review status=%d body=%s", invalidated.Code, invalidated.Body.String())
	}

	reviewed := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/review", observedAction(stopped)))
	if reviewed.Changes == nil || reviewed.Changes.Review == nil {
		t.Fatalf("review missing: %+v", reviewed.Changes)
	}
	accepted := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/accept", acceptChangesRequest{ProjectSlug: opened.ProjectSlug, StoreID: opened.StoreID, ChangeSetID: reviewed.Changes.ID, BaseRevision: reviewed.Changes.Review.BaseRevision, CandidateTree: reviewed.Changes.Review.CandidateTree, Generation: reviewed.Changes.Review.Generation}))
	if accepted.Revision == opened.Revision || roleSnapshot(accepted, c.ID, moving.ID) != "home" || roleSnapshot(accepted, b.ID, moving.ID) != "" {
		t.Fatalf("accepted = %+v", accepted)
	}
}

func TestHomeMoveUsesCandidateEligibilityAndRejectsWithoutMutatingPending(t *testing.T) {
	state, handler := newHandler(testOrigin, testUI(t), t.TempDir())
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Move eligibility"}))
	manager := state.architecture
	base := *state.loadedSnapshot
	gateway := manager.NewComponentChange(base, nil, "Gateway", "Gateway body.\n")
	worker := manager.NewComponentChange(base, []architecture.ComponentChange{gateway}, "Worker", "Worker body.\n")
	otherAnchor := manager.NewComponentChange(base, []architecture.ComponentChange{gateway, worker}, "Other", "Other body.\n")
	runtime := base.NewDetailDiagramChange(nil, "Runtime", gateway.ID)
	other := base.NewDetailDiagramChange([]architecture.DetailDiagramChange{runtime}, "Other diagram", otherAnchor.ID)
	initial, err := manager.ConstructCandidate(context.Background(), base, []architecture.ComponentChange{gateway, worker, otherAnchor}, architecture.CandidateComposition{
		NewComponentHomes: []architecture.NewComponentHome{
			{ComponentID: gateway.ID, DiagramID: base.RootDiagramID()},
			{ComponentID: worker.ID, DiagramID: base.RootDiagramID()},
			{ComponentID: otherAnchor.ID, DiagramID: base.RootDiagramID()},
		},
		DetailDiagrams: []architecture.DetailDiagramChange{runtime, other},
		HomeMoves:      []architecture.ComponentHomeMove{{ComponentID: worker.ID, DiagramID: runtime.ID}},
	})
	if err != nil {
		t.Fatal(err)
	}
	base = acceptArchitectureCandidate(t, manager, base, initial)
	storage := base.NewDetailDiagramChange(nil, "Storage", worker.ID)
	nested, err := manager.ConstructCandidate(context.Background(), base, nil, architecture.CandidateComposition{DetailDiagrams: []architecture.DetailDiagramChange{storage}})
	if err != nil {
		t.Fatal(err)
	}
	base = acceptArchitectureCandidate(t, manager, base, nested)
	opened := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/open", map[string]any{"project_slug": created.ProjectSlug}))

	var gatewayOptions componentHomeDestinationsResponse
	for _, options := range opened.HomeMoveDestinations {
		if options.ComponentID == gateway.ID {
			gatewayOptions = options
			break
		}
	}
	if gatewayOptions.CurrentHomeID != opened.RootDiagramID || slices.Contains(gatewayOptions.DiagramIDs, opened.RootDiagramID) || slices.Contains(gatewayOptions.DiagramIDs, runtime.ID) || slices.Contains(gatewayOptions.DiagramIDs, storage.ID) || !slices.Contains(gatewayOptions.DiagramIDs, other.ID) {
		t.Fatalf("gateway move options = %+v", gatewayOptions)
	}
	initialRejected := postJSONRequest(t, handler, "/api/architecture/components/move-home", observedDiagramMutation(opened, diagramMutationRequest{ComponentID: gateway.ID, DiagramID: storage.ID}))
	if initialRejected.Code != http.StatusConflict || !strings.Contains(initialRejected.Body.String(), errorHomeMoveUnavailable) {
		t.Fatalf("initial ineligible move status=%d body=%s", initialRejected.Code, initialRejected.Body.String())
	}
	state.stateMutex.Lock()
	if pending := testActiveChangeSet(state); pending != nil {
		t.Fatalf("rejected move created hidden change-set state: %+v", pending)
	}
	state.stateMutex.Unlock()

	kept := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/edit", observedComponentMutation(opened, componentMutationRequest{
		ComponentID: gateway.ID, Description: "Pending gateway.\n", DescriptionChanged: true,
	})))
	reviewed := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/review", observedAction(kept)))
	if reviewed.Changes == nil || reviewed.Changes.Review == nil {
		t.Fatalf("review missing: %+v", reviewed.Changes)
	}
	state.stateMutex.Lock()
	pendingBefore := testActiveChangeSet(state)
	generationBefore := pendingBefore.generation
	reviewBefore := pendingBefore.review
	treeBefore := pendingBefore.candidate.Tree()
	movesBefore := append([]architecture.ComponentHomeMove(nil), pendingBefore.homeMoves...)
	referencesBefore := append([]architecture.ReferenceAppearanceChange(nil), pendingBefore.references...)
	state.stateMutex.Unlock()

	rejected := postJSONRequest(t, handler, "/api/architecture/components/move-home", observedDiagramMutation(reviewed, diagramMutationRequest{ComponentID: gateway.ID, DiagramID: storage.ID}))
	if rejected.Code != http.StatusConflict || !strings.Contains(rejected.Body.String(), errorHomeMoveUnavailable) {
		t.Fatalf("ineligible move status=%d body=%s", rejected.Code, rejected.Body.String())
	}
	state.stateMutex.Lock()
	pendingAfter := testActiveChangeSet(state)
	if pendingAfter != pendingBefore || pendingAfter.generation != generationBefore || pendingAfter.review != reviewBefore || pendingAfter.candidate.Tree() != treeBefore || !slices.Equal(pendingAfter.homeMoves, movesBefore) || !slices.Equal(pendingAfter.references, referencesBefore) {
		t.Fatalf("rejected move mutated change set: before=%+v after=%+v", pendingBefore, pendingAfter)
	}
	state.stateMutex.Unlock()

	moved := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/move-home", observedDiagramMutation(reviewed, diagramMutationRequest{ComponentID: worker.ID, DiagramID: opened.RootDiagramID})))
	if moved.Changes == nil || moved.Changes.Candidate == nil || moved.Changes.Review != nil || role(moved.Changes.Candidate, opened.RootDiagramID, worker.ID) != "home" {
		t.Fatalf("valid move after rejection failed: %+v", moved.Changes)
	}
	state.stateMutex.Lock()
	pendingAfter = testActiveChangeSet(state)
	if pendingAfter.generation != generationBefore+1 || pendingAfter.review != nil {
		t.Fatalf("successful move generation/review = %d/%+v", pendingAfter.generation, pendingAfter.review)
	}
	state.stateMutex.Unlock()
	var gatewayAfterReparent componentHomeDestinationsResponse
	for _, options := range moved.HomeMoveDestinations {
		if options.ComponentID == gateway.ID {
			gatewayAfterReparent = options
			break
		}
	}
	if !slices.Contains(gatewayAfterReparent.DiagramIDs, storage.ID) {
		t.Fatalf("Storage did not become eligible after its anchor moved out first: %+v", gatewayAfterReparent)
	}
	reparented := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/move-home", observedDiagramMutation(moved, diagramMutationRequest{ComponentID: gateway.ID, DiagramID: storage.ID})))
	if reparented.Changes == nil || reparented.Changes.Candidate == nil || role(reparented.Changes.Candidate, storage.ID, gateway.ID) != "home" {
		t.Fatalf("ordered subtree reparent failed: %+v", reparented.Changes)
	}
	if _, err := manager.ConstructCandidate(context.Background(), base, nil, architecture.CandidateComposition{
		HomeMoves: []architecture.ComponentHomeMove{{ComponentID: gateway.ID, DiagramID: storage.ID}},
	}); !errors.Is(err, architecture.ErrDiagramCycle) {
		t.Fatalf("remove-only intermediate error = %v, want Diagram cycle", err)
	}
	replaced := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/move-home", observedDiagramMutation(reparented, diagramMutationRequest{ComponentID: worker.ID, DiagramID: other.ID})))
	if replaced.Changes == nil || replaced.Changes.Candidate == nil || role(replaced.Changes.Candidate, other.ID, worker.ID) != "home" || role(replaced.Changes.Candidate, storage.ID, gateway.ID) != "home" {
		t.Fatalf("complete replacement move failed: %+v", replaced.Changes)
	}
	reviewedReplacement := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/review", observedAction(replaced)))
	if reviewedReplacement.Changes == nil || reviewedReplacement.Changes.Review == nil {
		t.Fatalf("replacement review missing: %+v", reviewedReplacement.Changes)
	}
	acceptedReplacement := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/accept", acceptChangesRequest{
		ProjectSlug: opened.ProjectSlug, StoreID: opened.StoreID, ChangeSetID: reviewedReplacement.Changes.ID,
		BaseRevision: reviewedReplacement.Changes.Review.BaseRevision, CandidateTree: reviewedReplacement.Changes.Review.CandidateTree, Generation: reviewedReplacement.Changes.Review.Generation,
	}))
	if roleSnapshot(acceptedReplacement, other.ID, worker.ID) != "home" || roleSnapshot(acceptedReplacement, storage.ID, gateway.ID) != "home" {
		t.Fatalf("accepted replacement = %+v", acceptedReplacement)
	}
}

func TestReferenceAuthoringUsesPendingNewComponentsAndCandidateOnlyDiagrams(t *testing.T) {
	state, handler := newHandler(testOrigin, testUI(t), t.TempDir())
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Candidate references"}))
	anchorResult := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/add", observedComponentMutation(created, componentMutationRequest{DiagramID: created.RootDiagramID, Title: "Anchor"})))
	anchorID := anchorResult.Changes.Components[0].ID
	newResult := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/add", observedComponentMutation(anchorResult, componentMutationRequest{DiagramID: created.RootDiagramID, Title: "Pending target"})))
	newID := newResult.Changes.Components[1].ID
	detailResult := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/diagrams/detail", observedDiagramMutation(newResult, diagramMutationRequest{ComponentID: anchorID, Title: "Candidate detail"})))
	detailID := detailResult.Changes.DetailDiagrams[0].ID
	moved := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/move-home", observedDiagramMutation(detailResult, diagramMutationRequest{DiagramID: detailID, ComponentID: newID})))
	shownNew := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/diagrams/show-component", observedDiagramMutation(moved, diagramMutationRequest{DiagramID: created.RootDiagramID, ComponentID: newID})))
	shownAnchor := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/diagrams/show-component", observedDiagramMutation(shownNew, diagramMutationRequest{DiagramID: detailID, ComponentID: anchorID})))
	if shownAnchor.Changes == nil || shownAnchor.Changes.Candidate == nil ||
		role(shownAnchor.Changes.Candidate, created.RootDiagramID, newID) != "reference" ||
		role(shownAnchor.Changes.Candidate, detailID, newID) != "home" ||
		role(shownAnchor.Changes.Candidate, detailID, anchorID) != "reference" {
		t.Fatalf("candidate-relative references failed: new=%+v anchor=%+v", shownNew.Changes, shownAnchor.Changes)
	}
	state.stateMutex.Lock()
	generation := testActiveChangeSet(state).generation
	state.stateMutex.Unlock()
	duplicate := postJSONRequest(t, handler, "/api/architecture/diagrams/show-component", observedDiagramMutation(shownAnchor, diagramMutationRequest{DiagramID: detailID, ComponentID: anchorID}))
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d body=%s", duplicate.Code, duplicate.Body.String())
	}
	home := postJSONRequest(t, handler, "/api/architecture/diagrams/show-component", observedDiagramMutation(shownAnchor, diagramMutationRequest{DiagramID: created.RootDiagramID, ComponentID: anchorID}))
	if home.Code != http.StatusConflict {
		t.Fatalf("home status=%d body=%s", home.Code, home.Body.String())
	}
	state.stateMutex.Lock()
	if current := testActiveChangeSet(state); current.generation != generation {
		t.Fatalf("rejected reference mutation changed generation: got %d want %d", current.generation, generation)
	}
	state.stateMutex.Unlock()

	invalid := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/edit", observedComponentMutation(shownAnchor, componentMutationRequest{ComponentID: anchorID, Title: "   ", TitleChanged: true})))
	if invalid.Changes == nil || invalid.Changes.Candidate != nil || len(invalid.ReferenceChoices) != 0 {
		t.Fatalf("invalid candidate still offered reference authority: %+v", invalid)
	}
	blocked := postJSONRequest(t, handler, "/api/architecture/diagrams/stop-showing-component", observedDiagramMutation(invalid, diagramMutationRequest{DiagramID: detailID, ComponentID: anchorID}))
	if blocked.Code != http.StatusConflict || !strings.Contains(blocked.Body.String(), errorChangesUnavailable) {
		t.Fatalf("invalid-candidate mutation status=%d body=%s", blocked.Code, blocked.Body.String())
	}
}

func TestExternalAcceptedSlugRefreshAdoptsLocator(t *testing.T) {
	data := t.TempDir()
	_, handler := newHandler(testOrigin, testUI(t), data)
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Old Name"}))
	storePath := filepath.Join(data, "architecture", created.StoreID+".git")
	manifest := git(t, "--git-dir", storePath, "show", created.Revision+":architecture.yaml")
	manifest = strings.Replace(manifest, "slug: old-name", "slug: new-locator", 1)
	manifestBlob := gitInput(t, []byte(manifest), "--git-dir", storePath, "hash-object", "-w", "--stdin")
	root := strings.Split(git(t, "--git-dir", storePath, "ls-tree", created.Revision), "\n")
	for index, line := range root {
		if strings.HasSuffix(line, "\tarchitecture.yaml") {
			root[index] = "100644 blob " + manifestBlob + "\tarchitecture.yaml"
		}
	}
	tree := gitInput(t, []byte(strings.Join(root, "\n")+"\n"), "--git-dir", storePath, "mktree")
	commit := gitInput(t, []byte("external slug\n"), "-c", "user.name=Test", "-c", "user.email=test@workbraid.invalid", "--git-dir", storePath, "commit-tree", tree, "-p", created.Revision)
	git(t, "--git-dir", storePath, "update-ref", "refs/heads/accepted", commit, created.Revision)

	refreshed := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/refresh", observedAction(created)))
	if refreshed.ProjectSlug != "new-locator" || refreshed.StoreID != created.StoreID || refreshed.Revision != commit {
		t.Fatalf("refresh = %+v", refreshed)
	}
	old := postJSONRequest(t, handler, "/api/projects/open", map[string]any{"project_slug": created.ProjectSlug})
	if old.Code != http.StatusNotFound {
		t.Fatalf("old slug status=%d body=%s", old.Code, old.Body.String())
	}
	newOpen := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/open", map[string]any{"project_slug": "new-locator"}))
	if newOpen.StoreID != created.StoreID {
		t.Fatalf("new slug opened %+v", newOpen)
	}
}

func TestRefreshPreservesOldBaseChangeSetAsEditableOutOfDateWork(t *testing.T) {
	data := t.TempDir()
	state, handler := newHandler(testOrigin, testUI(t), data)
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Refresh"}))
	pending := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/add", observedComponentMutation(created, componentMutationRequest{DiagramID: created.RootDiagramID, Title: "Pending"})))
	if pending.Changes == nil || pending.Changes.Candidate == nil {
		t.Fatalf("pending = %+v", pending.Changes)
	}

	base := *state.loadedSnapshot
	external, err := state.architecture.ConstructCandidate(context.Background(), base, nil, architecture.CandidateComposition{
		DiagramTitles: []architecture.DiagramTitleChange{{DiagramID: base.RootDiagramID(), Title: "Externally renamed"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	externalRevision, err := state.architecture.CreateSuccessor(context.Background(), base, external)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.architecture.AdvanceAccepted(context.Background(), base, externalRevision); err != nil {
		t.Fatal(err)
	}

	refreshed := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/refresh", observedAction(pending)))
	selectActiveChangeSetForTest(&refreshed, pending.Changes.ID)
	if refreshed.Revision != externalRevision || refreshed.Stale || refreshed.Changes == nil || !refreshed.Changes.OutOfDate || refreshed.Changes.Stale {
		t.Fatalf("refreshed = %+v changes=%+v", refreshed, refreshed.Changes)
	}
	edited := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/add", observedComponentMutation(refreshed, componentMutationRequest{DiagramID: created.RootDiagramID, Title: "Still editable"})))
	if edited.Changes == nil || !edited.Changes.OutOfDate || edited.Changes.Generation != pending.Changes.Generation+1 {
		t.Fatalf("out-of-date edit = %+v", edited.Changes)
	}
	reviewed := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/review", observedAction(edited)))
	if reviewed.Changes == nil || reviewed.Changes.Review == nil || !reviewed.Changes.OutOfDate {
		t.Fatalf("out-of-date review = %+v", reviewed.Changes)
	}
	discarded := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/discard", observedAction(reviewed)))
	if discarded.Changes != nil || discarded.Revision != externalRevision {
		t.Fatalf("discarded = %+v", discarded)
	}
	newPending := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/add", observedComponentMutation(discarded, componentMutationRequest{DiagramID: created.RootDiagramID, Title: "Current"})))
	if newPending.Changes == nil || newPending.Changes.Stale || newPending.Changes.Candidate == nil {
		t.Fatalf("new pending = %+v", newPending.Changes)
	}
}

func TestAcceptedCASResponseLossAndStaleRaceRemainAuthoritative(t *testing.T) {
	t.Run("reported CAS failure is classified from accepted", func(t *testing.T) {
		state, handler := newHandler(testOrigin, testUI(t), t.TempDir())
		created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "CAS"}))
		kept := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/add", observedComponentMutation(created, componentMutationRequest{DiagramID: created.RootDiagramID, Title: "Worker"})))
		reviewed := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/review", observedAction(kept)))
		state.acceptedUpdateReportFailure = func() error { return errors.New("response lost") }
		accepted := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/accept", acceptChangesRequest{
			ProjectSlug: created.ProjectSlug, StoreID: created.StoreID, ChangeSetID: reviewed.Changes.ID, BaseRevision: reviewed.Changes.Review.BaseRevision,
			CandidateTree: reviewed.Changes.Review.CandidateTree, Generation: reviewed.Changes.Review.Generation,
		}))
		if accepted.Revision == created.Revision || testActiveChangeSet(state) != nil {
			t.Fatalf("accepted = %+v active=%+v", accepted, testActiveChangeSet(state))
		}
	})

	t.Run("post-CAS load failure retains receipts and Refresh recovers every record", func(t *testing.T) {
		state, handler := newHandler(testOrigin, testUI(t), t.TempDir())
		created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Publication"}))
		changeB := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/change-sets/create", changeSetCreateRequest{
			ProjectSlug: created.ProjectSlug, StoreID: created.StoreID, AcceptedRevision: created.Revision, Name: "Independent B",
		}))
		changeBID := changeB.ActionChangeSetID
		kept := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/add", observedComponentMutation(created, componentMutationRequest{DiagramID: created.RootDiagramID, Title: "Worker"})))
		reviewed := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/review", observedAction(kept)))
		state.publicationFailure = func() error { return errors.New("publication lost") }
		failLoad := true
		state.changeSetLoadFailure = func() error {
			if failLoad {
				failLoad = false
				return errors.New("change-set load lost")
			}
			return nil
		}
		response := postJSONRequest(t, handler, "/api/architecture/accept", acceptChangesRequest{
			ProjectSlug: created.ProjectSlug, StoreID: created.StoreID, ChangeSetID: reviewed.Changes.ID, BaseRevision: reviewed.Changes.Review.BaseRevision,
			CandidateTree: reviewed.Changes.Review.CandidateTree, Generation: reviewed.Changes.Review.Generation,
		})
		value := decodeArchitectureBody(t, response)
		if response.Code != http.StatusInternalServerError || value.ActionError != errorUpdatedReload || value.Revision == created.Revision || !value.Stale {
			t.Fatalf("response=%d value=%+v", response.Code, value)
		}
		appliedSeen, activeSeen := false, false
		for _, record := range value.ChangeSets {
			appliedSeen = appliedSeen || record.ID == reviewed.Changes.ID && record.Lifecycle == "applied"
			activeSeen = activeSeen || record.ID == changeBID && record.Lifecycle == "active"
		}
		if !appliedSeen || !activeSeen || len(state.changeSets) != 2 {
			t.Fatalf("post-CAS records disappeared: applied=%t active=%t state=%+v", appliedSeen, activeSeen, state.changeSets)
		}
		accepted, present, err := state.architecture.AcceptedRevision(context.Background(), *state.loadedSnapshot)
		if err != nil || !present || accepted != value.Revision {
			t.Fatalf("accepted=%q present=%t err=%v response=%q", accepted, present, err, value.Revision)
		}
		refreshedResponse := postJSONRequest(t, handler, "/api/architecture/refresh", architectureActionRequest{
			ProjectSlug: value.ProjectSlug, StoreID: value.StoreID, ExpectedRevision: value.Revision,
		})
		refreshed := decodeArchitectureBody(t, refreshedResponse)
		if refreshedResponse.Code != http.StatusOK || refreshed.Stale || len(refreshed.ChangeSets) != 2 {
			t.Fatalf("Refresh did not complete recovery: status=%d result=%+v", refreshedResponse.Code, refreshed)
		}
		records, unavailable, err := state.architecture.LoadChangeSets(context.Background(), created.StoreID)
		if err != nil || len(unavailable) != 0 || len(records) != 2 {
			t.Fatalf("durable records after recovery: records=%+v unavailable=%+v err=%v", records, unavailable, err)
		}
	})

	t.Run("final CAS race preserves external authority and stales pending", func(t *testing.T) {
		state, handler := newHandler(testOrigin, testUI(t), t.TempDir())
		created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Race"}))
		kept := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/add", observedComponentMutation(created, componentMutationRequest{DiagramID: created.RootDiagramID, Title: "Worker"})))
		reviewed := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/review", observedAction(kept)))
		base := *state.loadedSnapshot
		externalCandidate, err := state.architecture.ConstructCandidate(context.Background(), base, nil, architecture.CandidateComposition{
			DiagramTitles: []architecture.DiagramTitleChange{{DiagramID: base.RootDiagramID(), Title: "External"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		externalRevision := ""
		state.beforeAcceptedCAS = func(string) {
			var createErr error
			externalRevision, createErr = state.architecture.CreateSuccessor(context.Background(), base, externalCandidate)
			if createErr != nil {
				t.Fatal(createErr)
			}
			if createErr = state.architecture.AdvanceAccepted(context.Background(), base, externalRevision); createErr != nil {
				t.Fatal(createErr)
			}
		}
		response := postJSONRequest(t, handler, "/api/architecture/accept", acceptChangesRequest{
			ProjectSlug: created.ProjectSlug, StoreID: created.StoreID, ChangeSetID: reviewed.Changes.ID, BaseRevision: reviewed.Changes.Review.BaseRevision,
			CandidateTree: reviewed.Changes.Review.CandidateTree, Generation: reviewed.Changes.Review.Generation,
		})
		value := decodeArchitectureBody(t, response)
		if response.Code != http.StatusConflict || value.ActionError != errorArchitectureStale || !value.Stale || value.Changes == nil || !value.Changes.Stale {
			t.Fatalf("response=%d value=%+v", response.Code, value)
		}
		accepted, present, err := state.architecture.AcceptedRevision(context.Background(), base)
		if err != nil || !present || accepted != externalRevision {
			t.Fatalf("accepted=%q external=%q present=%t err=%v", accepted, externalRevision, present, err)
		}
	})
}

func TestUnavailableActiveReadableNamesBlockCreateAndRename(t *testing.T) {
	cases := []struct {
		name    string
		corrupt func(*testing.T, string, string, string)
	}{
		{
			name: "invalid parent",
			corrupt: func(t *testing.T, storePath, activeRef, object string) {
				tree := git(t, "--git-dir", storePath, "rev-parse", object+"^{tree}")
				invalidParent := gitInput(t, []byte("Invalid parent\n"), "-c", "user.name=Test", "-c", "user.email=test@workbraid.invalid", "--git-dir", storePath, "commit-tree", tree)
				git(t, "--git-dir", storePath, "update-ref", activeRef, invalidParent, object)
			},
		},
		{
			name: "duplicate lifecycle",
			corrupt: func(t *testing.T, storePath, activeRef, object string) {
				id := strings.TrimPrefix(activeRef, "refs/workbraid/change-sets/active/")
				git(t, "--git-dir", storePath, "update-ref", "refs/workbraid/change-sets/applied/"+id, object, strings.Repeat("0", 40))
			},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			data := t.TempDir()
			state, handler := newHandler(testOrigin, testUI(t), data)
			created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Reserved names"}))
			reserved := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/change-sets/create", changeSetCreateRequest{
				ProjectSlug: created.ProjectSlug, StoreID: created.StoreID, AcceptedRevision: created.Revision, Name: "Reserved Early",
			}))
			renameSource := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/change-sets/create", changeSetCreateRequest{
				ProjectSlug: created.ProjectSlug, StoreID: created.StoreID, AcceptedRevision: created.Revision, Name: "Rename Source",
			}))
			activeRef := "refs/workbraid/change-sets/active/" + reserved.ActionChangeSetID
			storePath := storePathFor(t, state.architecture, created.StoreID)
			object := git(t, "--git-dir", storePath, "show-ref", "--verify", "--hash", activeRef)
			test.corrupt(t, storePath, activeRef, object)

			restarted, restartedHandler := newHandler(testOrigin, testUI(t), data)
			opened := decodeArchitectureResponse(t, postJSONRequest(t, restartedHandler, "/api/projects/open", map[string]any{"project_slug": created.ProjectSlug}))
			foundUnavailable := false
			for _, record := range opened.UnavailableChangeSets {
				if record.ID == reserved.ActionChangeSetID && record.Lifecycle == "active" && record.Name == "Reserved Early" && record.Reason != "" {
					foundUnavailable = true
				}
			}
			if !foundUnavailable {
				t.Fatalf("readable active record was not typed unavailable: %+v", opened.UnavailableChangeSets)
			}

			createConflict := postJSONRequest(t, restartedHandler, "/api/architecture/change-sets/create", changeSetCreateRequest{
				ProjectSlug: opened.ProjectSlug, StoreID: opened.StoreID, AcceptedRevision: opened.Revision, Name: "reserved early",
			})
			if createConflict.Code != http.StatusConflict || !strings.Contains(createConflict.Body.String(), `"code":"`+errorChangeFailed+`"`) {
				t.Fatalf("create reused unavailable name: status=%d body=%s", createConflict.Code, createConflict.Body.String())
			}

			var source *changesResponse
			for _, record := range opened.ChangeSets {
				if record.ID == renameSource.ActionChangeSetID {
					source = record
					break
				}
			}
			if source == nil || source.Lifecycle != "active" {
				t.Fatalf("rename source missing after reload: %+v", opened.ChangeSets)
			}
			renameConflict := postJSONRequest(t, restartedHandler, "/api/architecture/change-sets/rename", changeSetEditRequest{
				ProjectSlug: opened.ProjectSlug, StoreID: opened.StoreID, ChangeSetID: source.ID, Generation: source.Generation, Name: "RESERVED EARLY",
			})
			if renameConflict.Code != http.StatusConflict || !strings.Contains(renameConflict.Body.String(), `"code":"`+errorChangeFailed+`"`) {
				t.Fatalf("rename reused unavailable name: status=%d body=%s", renameConflict.Code, renameConflict.Body.String())
			}

			restarted.stateMutex.Lock()
			defer restarted.stateMutex.Unlock()
			if restarted.changeSets[source.ID] == nil || restarted.changeSets[source.ID].name != "Rename Source" {
				t.Fatalf("failed conflicts mutated rename source: %+v", restarted.changeSets[source.ID])
			}
			stillUnavailable := false
			for _, record := range restarted.unavailableChangeSets {
				stillUnavailable = stillUnavailable || record.ID == reserved.ActionChangeSetID && record.Lifecycle == "active" && record.Name == "Reserved Early"
			}
			if !stillUnavailable {
				t.Fatalf("conflict attempts lost typed unavailable state: %+v", restarted.unavailableChangeSets)
			}
		})
	}
}

func TestReferenceReviewChangesCompositionWithoutSemanticDeltas(t *testing.T) {
	ctx := context.Background()
	manager := architecture.NewManager(t.TempDir())
	base, err := manager.CreateProject(ctx, "Projection")
	if err != nil {
		t.Fatal(err)
	}
	anchor := manager.NewComponentChange(base, nil, "Anchor", "")
	target := manager.NewComponentChange(base, []architecture.ComponentChange{anchor}, "Target", "")
	lonely := manager.NewComponentChange(base, []architecture.ComponentChange{anchor, target}, "Lonely", "")
	source := manager.NewComponentChange(base, []architecture.ComponentChange{anchor, target, lonely}, "Source", "")
	source.Relationships = []architecture.AuthoringRelationship{{TargetID: target.ID, Label: "calls"}, {TargetID: target.ID, Label: "calls async"}}
	source.RelationshipsChanged = true
	detail := base.NewDetailDiagramChange(nil, "Detail", anchor.ID)
	initial, err := manager.ConstructCandidate(ctx, base, []architecture.ComponentChange{anchor, target, lonely, source}, architecture.CandidateComposition{
		NewComponentHomes: []architecture.NewComponentHome{
			{ComponentID: anchor.ID, DiagramID: base.RootDiagramID()},
			{ComponentID: target.ID, DiagramID: base.RootDiagramID()},
			{ComponentID: lonely.ID, DiagramID: base.RootDiagramID()},
			{ComponentID: source.ID, DiagramID: base.RootDiagramID()},
		},
		DetailDiagrams: []architecture.DetailDiagramChange{detail},
		HomeMoves: []architecture.ComponentHomeMove{
			{ComponentID: target.ID, DiagramID: detail.ID},
			{ComponentID: lonely.ID, DiagramID: detail.ID},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	base = acceptArchitectureCandidate(t, manager, base, initial)
	rootBefore := diagramProjection(t, base, base.RootDiagramID())
	if len(rootBefore.Boundaries) != 1 || len(rootBefore.Relationships) != 2 {
		t.Fatalf("base root boundaries=%d relationships=%d", len(rootBefore.Boundaries), len(rootBefore.Relationships))
	}

	withReferences, err := manager.ConstructCandidate(ctx, base, nil, architecture.CandidateComposition{References: []architecture.ReferenceAppearanceChange{
		{DiagramID: base.RootDiagramID(), ComponentID: target.ID, Present: true},
		{DiagramID: base.RootDiagramID(), ComponentID: lonely.ID, Present: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	_, withProjection, comparison := captureReviewPresentation(base, withReferences.Snapshot())
	if len(comparison.Components) != 0 || len(comparison.Relationships) != 0 || len(comparison.Appearances) != 2 {
		t.Fatalf("reference comparison = %+v", comparison)
	}
	rootWith := diagramProjectionResponse(t, withProjection.Diagrams, base.RootDiagramID())
	if len(rootWith.Boundaries) != 0 || len(rootWith.Relationships) != 2 {
		t.Fatalf("reference root boundaries=%d relationships=%d", len(rootWith.Boundaries), len(rootWith.Relationships))
	}
	for _, relationship := range rootWith.Relationships {
		if relationship.TargetNodeKey != target.ID {
			t.Fatalf("relationship did not connect to canonical target: %+v", relationship)
		}
	}

	componentsBefore := git(t, "--git-dir", storePathFor(t, manager, base.StoreID()), "ls-tree", base.Revision(), "components")
	componentsAfter := git(t, "--git-dir", storePathFor(t, manager, base.StoreID()), "ls-tree", withReferences.Tree(), "components")
	if componentsBefore != componentsAfter {
		t.Fatalf("reference edit rewrote Components:\nbefore %s\nafter  %s", componentsBefore, componentsAfter)
	}

	acceptedWithReferences := acceptArchitectureCandidate(t, manager, base, withReferences)
	removed, err := manager.ConstructCandidate(ctx, acceptedWithReferences, nil, architecture.CandidateComposition{References: []architecture.ReferenceAppearanceChange{
		{DiagramID: base.RootDiagramID(), ComponentID: target.ID, Present: false},
		{DiagramID: base.RootDiagramID(), ComponentID: lonely.ID, Present: false},
	}})
	if err != nil {
		t.Fatal(err)
	}
	_, removedProjection, removedComparison := captureReviewPresentation(acceptedWithReferences, removed.Snapshot())
	if len(removedComparison.Components) != 0 || len(removedComparison.Relationships) != 0 || len(removedComparison.Appearances) != 2 {
		t.Fatalf("removal comparison = %+v", removedComparison)
	}
	rootRemoved := diagramProjectionResponse(t, removedProjection.Diagrams, base.RootDiagramID())
	if len(rootRemoved.Boundaries) != 1 || len(rootRemoved.Relationships) != 2 {
		t.Fatalf("removed root boundaries=%d relationships=%d", len(rootRemoved.Boundaries), len(rootRemoved.Relationships))
	}
	for _, appearance := range rootRemoved.Appearances {
		if appearance.ComponentID == lonely.ID {
			t.Fatal("unconnected removed reference remained")
		}
	}
}

func acceptArchitectureCandidate(t *testing.T, manager *architecture.Manager, base architecture.Snapshot, candidate architecture.Candidate) architecture.Snapshot {
	t.Helper()
	commit, err := manager.CreateSuccessor(context.Background(), base, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.AdvanceAccepted(context.Background(), base, commit); err != nil {
		t.Fatal(err)
	}
	loaded, err := manager.LoadAccepted(context.Background(), base.StoreID())
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func diagramProjection(t *testing.T, snapshot architecture.Snapshot, diagramID string) architecture.DiagramProjection {
	t.Helper()
	for _, diagram := range snapshot.DiagramProjections() {
		if diagram.ID == diagramID {
			return diagram
		}
	}
	t.Fatalf("diagram %s missing", diagramID)
	return architecture.DiagramProjection{}
}

func diagramProjectionResponse(t *testing.T, diagrams []diagramResponse, diagramID string) diagramResponse {
	t.Helper()
	for _, diagram := range diagrams {
		if diagram.ID == diagramID {
			return diagram
		}
	}
	t.Fatalf("diagram %s missing", diagramID)
	return diagramResponse{}
}

func storePathFor(t *testing.T, manager *architecture.Manager, storeID string) string {
	t.Helper()
	path, err := manager.StorePath(storeID)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestProjectRouteServesBuiltApplication(t *testing.T) {
	ui := testUI(t)
	handler := NewHandler(testOrigin, ui, t.TempDir())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/projects/example", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "WorkBraid test UI") {
		t.Fatalf("route status=%d body=%s", response.Code, response.Body.String())
	}
}

func postJSONRequest(t *testing.T, handler http.Handler, path string, value any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	request.Header.Set("Origin", testOrigin)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func pendingGenerationFor(response architectureResponse) *uint64 {
	if response.Changes == nil {
		return nil
	}
	generation := response.Changes.Generation
	return &generation
}

func observedAction(response architectureResponse) architectureActionRequest {
	return architectureActionRequest{
		ProjectSlug: response.ProjectSlug, StoreID: response.StoreID, ExpectedRevision: response.Revision,
		ChangeSetID:        changeSetIDFor(response),
		ExpectedGeneration: pendingGenerationFor(response), PendingGenerationObserved: true,
	}
}

func observedComponentMutation(response architectureResponse, payload componentMutationRequest) componentMutationRequest {
	payload.ProjectSlug = response.ProjectSlug
	payload.StoreID = response.StoreID
	payload.ExpectedRevision = response.Revision
	payload.ChangeSetID = changeSetIDFor(response)
	payload.ExpectedGeneration = pendingGenerationFor(response)
	payload.PendingGenerationObserved = true
	return payload
}

func observedDiagramMutation(response architectureResponse, payload diagramMutationRequest) diagramMutationRequest {
	payload.ProjectSlug = response.ProjectSlug
	payload.StoreID = response.StoreID
	payload.ExpectedRevision = response.Revision
	payload.ChangeSetID = changeSetIDFor(response)
	payload.ExpectedGeneration = pendingGenerationFor(response)
	payload.PendingGenerationObserved = true
	return payload
}

func changeSetIDFor(response architectureResponse) string {
	if response.Changes == nil {
		return ""
	}
	return response.Changes.ID
}

func selectActiveChangeSetForTest(response *architectureResponse, id string) {
	if response.Changes != nil && response.Changes.ID == id {
		return
	}
	for _, record := range response.ChangeSets {
		if record.ID == id {
			response.Changes = record
			return
		}
	}
}

func decodeArchitectureResponse(t *testing.T, response *httptest.ResponseRecorder) architectureResponse {
	t.Helper()
	if response.Code < 200 || response.Code >= 300 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	return decodeArchitectureBody(t, response)
}

func decodeArchitectureBody(t *testing.T, response *httptest.ResponseRecorder) architectureResponse {
	t.Helper()
	var value architectureResponse
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func role(snapshot *snapshotProjectionResponse, diagramID, componentID string) string {
	if snapshot == nil {
		return ""
	}
	for _, diagram := range snapshot.Diagrams {
		if diagram.ID != diagramID {
			continue
		}
		for _, appearance := range diagram.Appearances {
			if appearance.ComponentID == componentID {
				return appearance.Role
			}
		}
	}
	return ""
}

func roleSnapshot(snapshot architectureResponse, diagramID, componentID string) string {
	for _, diagram := range snapshot.Diagrams {
		if diagram.ID != diagramID {
			continue
		}
		for _, appearance := range diagram.Appearances {
			if appearance.ComponentID == componentID {
				return appearance.Role
			}
		}
	}
	return ""
}

func testUI(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "index.html"), []byte("<!doctype html><title>WorkBraid test UI</title>"), 0o600); err != nil {
		t.Fatal(err)
	}
	return directory
}

func git(t *testing.T, arguments ...string) string { return gitInput(t, nil, arguments...) }

func gitInput(t *testing.T, input []byte, arguments ...string) string {
	t.Helper()
	command := exec.CommandContext(context.Background(), "git", arguments...)
	command.Stdin = bytes.NewReader(input)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
	return strings.TrimSpace(string(output))
}
