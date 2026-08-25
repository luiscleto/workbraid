package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func TestNestedDiagramCompositionUsesOnePendingCandidateAndRetainsInvalidMove(t *testing.T) {
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	db := openWebDatabaseAt(t, filepath.Join(dataDirectory, "workbraid.db"))
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	accepted := advanceAcceptedToP21V2(t, storePath, initialized.Revision, storeID, "System")
	decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))

	created := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/diagrams/detail", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: p21WorkerID, Title: "   ",
	}))
	if created.Changes == nil || created.Changes.Valid || created.Changes.ValidationCode != "diagram_title_required" || len(created.Changes.DetailDiagrams) != 1 {
		t.Fatalf("detail pending result = %+v", created.Changes)
	}
	newDiagram := created.Changes.DetailDiagrams[0]
	created = decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/diagrams/title", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, DiagramID: newDiagram.ID, Title: "Worker internals",
	}))
	if created.Changes == nil || !created.Changes.Valid || created.Changes.Candidate == nil || created.Changes.DetailDiagrams[0].ID != newDiagram.ID || created.Changes.DetailDiagrams[0].Path != newDiagram.Path {
		t.Fatalf("corrected stable detail = %+v", created.Changes)
	}
	if got := diagramResponseByID(t, created.Changes.Candidate.Diagrams, newDiagram.ID); got.Title != "Worker internals" {
		t.Fatalf("candidate-only Diagram = %+v", got)
	}
	createdComponent := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, DiagramID: newDiagram.ID,
		Title: "Queue", Description: "Queue documentation.\n",
	}))
	var queueID string
	for _, component := range createdComponent.Changes.Components {
		if component.New {
			queueID = component.ID
		}
	}
	if queueID == "" || !diagramHasComponent(diagramResponseByID(t, createdComponent.Changes.Candidate.Diagrams, newDiagram.ID), queueID, "home") {
		t.Fatalf("pending-new Component home in candidate-only Diagram = %+v", createdComponent.Changes)
	}
	decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: p21RecordsID,
		Description: "Changed records documentation.\n", DescriptionChanged: true,
	}))
	mixed := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: p21WorkerID,
		RelationshipsChanged: true, Relationships: []relationshipResponse{{TargetID: queueID, Label: "dispatches"}},
	}))
	if mixed.Changes == nil || !mixed.Changes.Valid || len(mixed.Changes.Components) != 3 {
		t.Fatalf("mixed Component/Relationship/Diagram candidate = %+v", mixed.Changes)
	}
	secondNested := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/diagrams/detail", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: queueID, Title: "Queue internals",
	}))
	if secondNested.Changes == nil || !secondNested.Changes.Valid || len(secondNested.Changes.DetailDiagrams) != 2 {
		t.Fatalf("second candidate-only nested Diagram = %+v", secondNested.Changes)
	}
	queueDetailID := secondNested.Changes.DetailDiagrams[1].ID
	renamed := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/diagrams/title", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, DiagramID: p21DetailID, Title: "Shared internals",
	}))
	if renamed.Changes == nil || !renamed.Changes.Valid || diagramResponseByID(t, renamed.Changes.Candidate.Diagrams, p21DetailID).Title != "Shared internals" {
		t.Fatalf("renamed existing Diagram = %+v", renamed.Changes)
	}
	beforeMoveTree := state.pending.candidate.Tree()
	workerDetailPath := renamed.Changes.DetailDiagrams[0].Path
	queueDetailPath := renamed.Changes.DetailDiagrams[1].Path
	workerDetailEntry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", beforeMoveTree, workerDetailPath)
	queueDetailEntry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", beforeMoveTree, queueDetailPath)
	moved := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/components/move-home", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: p21WorkerID, DiagramID: p21RootID,
	}))
	if moved.Changes == nil || !moved.Changes.Valid || moved.Changes.Candidate == nil {
		t.Fatalf("reference-to-home move = %+v", moved.Changes)
	}
	root := diagramResponseByID(t, moved.Changes.Candidate.Diagrams, p21RootID)
	workerAppearances := 0
	for _, appearance := range root.Appearances {
		if appearance.ComponentID == p21WorkerID {
			workerAppearances++
			if appearance.Role != "home" || appearance.DetailDiagramID != newDiagram.ID {
				t.Fatalf("converted Worker appearance = %+v", appearance)
			}
		}
	}
	if workerAppearances != 1 {
		t.Fatalf("Worker root appearances = %d", workerAppearances)
	}
	if len(root.Appearances) < 2 || root.Appearances[1].ComponentID != p21WorkerID || root.Appearances[1].Role != "home" {
		t.Fatalf("reference conversion did not preserve destination appearance position: %+v", root.Appearances)
	}
	if diagramHasComponent(diagramResponseByID(t, moved.Changes.Candidate.Diagrams, p21DetailID), p21WorkerID, "home") {
		t.Fatal("old home remained after move")
	}
	afterMoveTree := state.pending.candidate.Tree()
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", afterMoveTree, workerDetailPath); got != workerDetailEntry {
		t.Fatalf("moving anchored home rewrote child Diagram\nbefore: %s\nafter: %s", workerDetailEntry, got)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", afterMoveTree, queueDetailPath); got != queueDetailEntry {
		t.Fatalf("moving anchored home rewrote descendant Diagram\nbefore: %s\nafter: %s", queueDetailEntry, got)
	}

	invalid := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/components/move-home", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: p21WorkerID, DiagramID: queueDetailID,
	}))
	if invalid.Changes == nil || invalid.Changes.Valid || invalid.Changes.ValidationCode != "diagram_cycle" || len(invalid.Changes.DetailDiagrams) != 2 {
		t.Fatalf("invalid retained composition = %+v", invalid.Changes)
	}
	if state.pending == nil || state.pending.candidate != nil {
		t.Fatalf("backend pending after invalid move = %+v", state.pending)
	}
	invalidGeneration := state.pending.generation
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != accepted {
		t.Fatalf("invalid move advanced accepted to %q", got)
	}

	corrected := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/components/move-home", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: p21WorkerID, DiagramID: p21RootID,
	}))
	if corrected.Changes == nil || !corrected.Changes.Valid || corrected.Changes.Candidate == nil || len(corrected.Changes.DetailDiagrams) != 2 {
		t.Fatalf("corrected composition = %+v", corrected.Changes)
	}
	reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	if reviewed.Changes == nil || reviewed.Changes.Review == nil || reviewed.Changes.Review.Generation != invalidGeneration+1 {
		t.Fatalf("reviewed nested candidate = %+v", reviewed.Changes)
	}
	if len(reviewed.Changes.Review.Comparison.Components) != 2 || len(reviewed.Changes.Review.Comparison.Relationships) == 0 || len(reviewed.Changes.Review.Comparison.Diagrams) != 3 || len(reviewed.Changes.Review.Comparison.Appearances) == 0 {
		t.Fatalf("composition review classification = %+v", reviewed.Changes.Review.Comparison)
	}
}

func TestReviewClassifiesDetailLinkAsCompositionInsteadOfPlacement(t *testing.T) {
	before := []diagramResponse{{
		ID: "root", Filename: "root.yaml", Title: "System",
		Appearances: []diagramAppearanceResponse{{ComponentID: p21WorkerID, Role: "home"}},
	}}
	withChanges := []diagramResponse{{
		ID: "root", Filename: "root.yaml", Title: "System",
		Appearances: []diagramAppearanceResponse{{ComponentID: p21WorkerID, Role: "home", DetailDiagramID: p21DetailID}},
	}}

	diagrams, appearances := compareDiagramProjections(before, withChanges)
	if len(diagrams) != 0 || len(appearances) != 1 || appearances[0].Status != "detail_changed" {
		t.Fatalf("detail-link review classification: diagrams=%+v appearances=%+v", diagrams, appearances)
	}
}

func TestDiagramTitleOnlyPreservesCompositionPathsModesAndUnrelatedEntries(t *testing.T) {
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	db := openWebDatabaseAt(t, filepath.Join(dataDirectory, "workbraid.db"))
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	accepted := advanceAcceptedToP21V2(t, storePath, initialized.Revision, storeID, "System")
	base := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))

	renamed := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/diagrams/title", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, DiagramID: p21DetailID, Title: "Renamed detail",
	}))
	if renamed.Changes == nil || renamed.Changes.Candidate == nil || state.pending == nil || state.pending.candidate == nil {
		t.Fatalf("title-only pending = %+v", renamed.Changes)
	}
	before := diagramResponseByID(t, base.Diagrams, p21DetailID)
	after := diagramResponseByID(t, renamed.Changes.Candidate.Diagrams, p21DetailID)
	if after.ID != before.ID || after.Filename != before.Filename || after.Title != "Renamed detail" || !reflect.DeepEqual(after.Appearances, before.Appearances) || !reflect.DeepEqual(after.Boundaries, before.Boundaries) || !reflect.DeepEqual(after.Relationships, before.Relationships) {
		t.Fatalf("title edit changed composition\nbefore=%+v\nafter=%+v", before, after)
	}
	candidateTree := state.pending.candidate.Tree()
	for _, path := range []string{"architecture.yaml", "components", "diagrams/root.yaml", "diagrams/empty.yaml"} {
		if beforeEntry, afterEntry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", accepted, path), runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", candidateTree, path); beforeEntry != afterEntry {
			t.Fatalf("title edit rewrote unrelated %s\nbefore: %s\nafter: %s", path, beforeEntry, afterEntry)
		}
	}
	if entry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", candidateTree, "diagrams/detail.yaml"); len(entry) < len("100644") || entry[:len("100644")] != "100644" {
		t.Fatalf("title edit changed Diagram mode/path: %q", entry)
	}
}

func TestDiagramMutationRacingDiscardLeavesOneCoherentPendingAuthority(t *testing.T) {
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	db := openWebDatabaseAt(t, filepath.Join(dataDirectory, "workbraid.db"))
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	rootID := initialized.RootDiagramID
	decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/diagrams/title", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: initialized.Revision, DiagramID: rootID, Title: "First pending title",
	}))
	reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	if reviewed.Changes == nil || reviewed.Changes.Review == nil {
		t.Fatal("pre-race review missing")
	}

	mutationBody, err := json.Marshal(diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: initialized.Revision, DiagramID: rootID, Title: "Second pending title",
	})
	if err != nil {
		t.Fatal(err)
	}
	discardBody, err := json.Marshal(openProjectRequest{SourceRoot: filepath.Clean(source)})
	if err != nil {
		t.Fatal(err)
	}
	requests := []*http.Request{
		httptest.NewRequest(http.MethodPost, "/api/architecture/diagrams/title", bytes.NewReader(mutationBody)),
		httptest.NewRequest(http.MethodPost, "/api/architecture/discard", bytes.NewReader(discardBody)),
	}
	for _, request := range requests {
		request.Header.Set("Origin", testOrigin)
		request.Header.Set("Content-Type", "application/json")
	}
	responses := []*httptest.ResponseRecorder{httptest.NewRecorder(), httptest.NewRecorder()}
	start := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(2)
	for index := range requests {
		go func(index int) {
			defer wait.Done()
			<-start
			handler.ServeHTTP(responses[index], requests[index])
		}(index)
	}
	close(start)
	wait.Wait()
	if responses[0].Code != http.StatusOK || responses[1].Code != http.StatusOK {
		t.Fatalf("race responses = %d, %d", responses[0].Code, responses[1].Code)
	}
	if state.pending != nil {
		if state.pending.review != nil || state.pending.candidate == nil || state.pending.generation != 1 || len(state.pending.diagramTitles) != 1 || state.pending.diagramTitles[0].Title != "Second pending title" {
			t.Fatalf("post-race pending authority = %+v", state.pending)
		}
	}
	if state.loadedSnapshot == nil || state.loadedSnapshot.Revision() != initialized.Revision {
		t.Fatalf("race changed accepted projection: %+v", state.loadedSnapshot)
	}
}

func postDiagramMutation(t *testing.T, handler http.Handler, path string, payload diagramMutationRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", testOrigin)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
