package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"workbraid/internal/architecture"
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
	createdSidecar := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, DiagramID: p21RootID,
		Title: "Sidecar", Description: "Sidecar documentation.\n",
	}))
	var sidecarID string
	for _, component := range createdSidecar.Changes.Components {
		if component.New && component.Title == "Sidecar" {
			sidecarID = component.ID
		}
	}
	if sidecarID == "" {
		t.Fatalf("pending-new sibling anchor = %+v", createdSidecar.Changes)
	}
	decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: p21RecordsID,
		Description: "Changed records documentation.\n", DescriptionChanged: true,
	}))
	mixed := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: p21WorkerID,
		RelationshipsChanged: true, Relationships: []relationshipResponse{{TargetID: queueID, Label: "dispatches"}},
	}))
	if mixed.Changes == nil || !mixed.Changes.Valid || len(mixed.Changes.Components) != 4 {
		t.Fatalf("mixed Component/Relationship/Diagram candidate = %+v", mixed.Changes)
	}
	secondNested := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/diagrams/detail", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: queueID, Title: "Queue internals",
	}))
	if secondNested.Changes == nil || !secondNested.Changes.Valid || len(secondNested.Changes.DetailDiagrams) != 2 {
		t.Fatalf("second candidate-only nested Diagram = %+v", secondNested.Changes)
	}
	queueDetailID := secondNested.Changes.DetailDiagrams[1].ID
	sidecarNested := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/diagrams/detail", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: sidecarID, Title: "Sidecar internals",
	}))
	if sidecarNested.Changes == nil || !sidecarNested.Changes.Valid || len(sidecarNested.Changes.DetailDiagrams) != 3 {
		t.Fatalf("candidate-only sibling Diagram = %+v", sidecarNested.Changes)
	}
	sidecarDetailID := sidecarNested.Changes.DetailDiagrams[2].ID
	renamed := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/diagrams/title", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, DiagramID: p21DetailID, Title: "Shared internals",
	}))
	if renamed.Changes == nil || !renamed.Changes.Valid || diagramResponseByID(t, renamed.Changes.Candidate.Diagrams, p21DetailID).Title != "Shared internals" {
		t.Fatalf("renamed existing Diagram = %+v", renamed.Changes)
	}
	beforeMoveTree := state.pending.candidate.Tree()
	workerDetailPath := renamed.Changes.DetailDiagrams[0].Path
	queueDetailPath := renamed.Changes.DetailDiagrams[1].Path
	sidecarDetailPath := renamed.Changes.DetailDiagrams[2].Path
	workerDetailEntry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", beforeMoveTree, workerDetailPath)
	queueDetailEntry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", beforeMoveTree, queueDetailPath)
	sidecarDetailEntry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", beforeMoveTree, sidecarDetailPath)
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
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", afterMoveTree, sidecarDetailPath); got != sidecarDetailEntry {
		t.Fatalf("moving anchored home rewrote sibling Diagram\nbefore: %s\nafter: %s", sidecarDetailEntry, got)
	}
	if hasHomeDestination(moved.HomeMoveDestinations, p21WorkerID, newDiagram.ID) {
		t.Fatalf("Worker's directly owned detail remained a destination: %+v", moved.HomeMoveDestinations)
	}
	if !hasHomeDestination(moved.HomeMoveDestinations, p21WorkerID, queueDetailID) {
		t.Fatalf("deeper candidate-only descendant was removed as a destination: %+v", moved.HomeMoveDestinations)
	}

	reviewedBeforeRejection := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	if reviewedBeforeRejection.Changes == nil || reviewedBeforeRejection.Changes.Review == nil {
		t.Fatalf("review before direct rejection = %+v", reviewedBeforeRejection.Changes)
	}
	beforeRejectedGeneration := state.pending.generation
	beforeRejectedTree := state.pending.candidate.Tree()
	beforeRejectedMoves := append([]architecture.ComponentHomeMove(nil), state.pending.homeMoves...)
	beforeRejectedReview := *state.pending.review
	directOwned := postDiagramMutation(t, handler, "/api/architecture/components/move-home", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: p21WorkerID, DiagramID: newDiagram.ID,
	})
	if directOwned.Code != http.StatusConflict || !strings.Contains(directOwned.Body.String(), `"code":"diagram_own_detail"`) {
		t.Fatalf("direct-owned move status/body = %d/%s", directOwned.Code, directOwned.Body.String())
	}
	if state.pending.generation != beforeRejectedGeneration || state.pending.candidate.Tree() != beforeRejectedTree || !reflect.DeepEqual(state.pending.homeMoves, beforeRejectedMoves) || !reflect.DeepEqual(*state.pending.review, beforeRejectedReview) {
		t.Fatalf("direct-owned rejection mutated pending: generation=%d moves=%+v review=%+v", state.pending.generation, state.pending.homeMoves, state.pending.review)
	}
	staleDirect := postDiagramMutation(t, handler, "/api/architecture/components/move-home", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: strings.Repeat("f", 40), ComponentID: p21WorkerID, DiagramID: newDiagram.ID,
	})
	if staleDirect.Code != http.StatusConflict || !strings.Contains(staleDirect.Body.String(), `"code":"changes_elsewhere"`) {
		t.Fatalf("stale direct move status/body = %d/%s", staleDirect.Code, staleDirect.Body.String())
	}
	if state.pending.generation != beforeRejectedGeneration || state.pending.candidate.Tree() != beforeRejectedTree || !reflect.DeepEqual(state.pending.homeMoves, beforeRejectedMoves) || !reflect.DeepEqual(*state.pending.review, beforeRejectedReview) {
		t.Fatal("stale direct rejection mutated pending or review binding")
	}

	invalid := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/components/move-home", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: p21WorkerID, DiagramID: queueDetailID,
	}))
	if invalid.Changes == nil || invalid.Changes.Valid || invalid.Changes.ValidationCode != "diagram_cycle" || len(invalid.Changes.DetailDiagrams) != 3 {
		t.Fatalf("invalid retained composition = %+v", invalid.Changes)
	}
	if !hasDiagramAuthoringOption(invalid.Changes.DiagramOptions, sidecarDetailID) {
		t.Fatalf("candidate-only correction destination missing from %+v", invalid.Changes.DiagramOptions)
	}
	if state.pending == nil || state.pending.candidate != nil {
		t.Fatalf("backend pending after invalid move = %+v", state.pending)
	}
	if invalid.Changes.ReviewBlocker != "" || invalid.Changes.ValidationItem != p21WorkerID || invalid.Changes.ValidationDiagramField != "home" {
		t.Fatalf("invalid move was not immediately localized before review: %+v", invalid.Changes)
	}
	if hasHomeDestination(invalid.HomeMoveDestinations, p21WorkerID, newDiagram.ID) || !hasHomeDestination(invalid.HomeMoveDestinations, p21WorkerID, sidecarDetailID) {
		t.Fatalf("invalid-move correction destinations = %+v", invalid.HomeMoveDestinations)
	}
	invalidGeneration := state.pending.generation
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != accepted {
		t.Fatalf("invalid move advanced accepted to %q", got)
	}

	corrected := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/components/move-home", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: p21WorkerID, DiagramID: sidecarDetailID,
	}))
	if corrected.Changes == nil || !corrected.Changes.Valid || corrected.Changes.Candidate == nil || len(corrected.Changes.DetailDiagrams) != 3 {
		t.Fatalf("corrected composition = %+v", corrected.Changes)
	}
	if home, _, ok := state.pending.candidate.Snapshot().ComponentHome(p21WorkerID); !ok || home != sidecarDetailID {
		t.Fatalf("candidate-only correction home = %q, ok=%v", home, ok)
	}
	reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	if reviewed.Changes == nil || reviewed.Changes.Review == nil || reviewed.Changes.Review.Generation != invalidGeneration+1 {
		t.Fatalf("reviewed nested candidate = %+v", reviewed.Changes)
	}
	if len(reviewed.Changes.Review.Comparison.Components) != 3 || len(reviewed.Changes.Review.Comparison.Relationships) == 0 || len(reviewed.Changes.Review.Comparison.Diagrams) != 4 || len(reviewed.Changes.Review.Comparison.Appearances) == 0 {
		t.Fatalf("composition review classification = %+v", reviewed.Changes.Review.Comparison)
	}
}

func hasHomeDestination(values []componentHomeDestinationsResponse, componentID, diagramID string) bool {
	for _, value := range values {
		if value.ComponentID == componentID && slices.Contains(value.DiagramIDs, diagramID) {
			return true
		}
	}
	return false
}

func hasDiagramAuthoringOption(options []diagramAuthoringOptionResponse, id string) bool {
	for _, option := range options {
		if option.ID == id {
			return true
		}
	}
	return false
}

func TestDiagramAuthoringOptionsUseFinalCollisionOnlyContext(t *testing.T) {
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	db := openWebDatabaseAt(t, filepath.Join(dataDirectory, "workbraid.db"))
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	accepted := advanceAcceptedToP21V2(t, storePath, initialized.Revision, storeID, "System")
	decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))

	decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/diagrams/title", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, DiagramID: p21DetailID, Title: "Shared internals",
	}))
	renamed := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/diagrams/title", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, DiagramID: p21EmptyID, Title: "Shared internals",
	}))
	firstAccepted := diagramAuthoringOptionByID(t, renamed.Changes.DiagramOptions, p21DetailID)
	secondAccepted := diagramAuthoringOptionByID(t, renamed.Changes.DiagramOptions, p21EmptyID)
	if firstAccepted.Context != "Detail for Shared — gateway.md" || secondAccepted.Context != "Detail for Shared — records.md" {
		t.Fatalf("pending title collision contexts = %+v, %+v", firstAccepted, secondAccepted)
	}

	unique := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/diagrams/detail", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: p21WorkerID, Title: "Worker internals",
	}))
	uniqueDetail := unique.Changes.DetailDiagrams[len(unique.Changes.DetailDiagrams)-1]
	if option := diagramAuthoringOptionByID(t, unique.Changes.DiagramOptions, uniqueDetail.ID); option.Context != "" {
		t.Fatalf("unique pending detail has speculative context: %+v", option)
	}

	firstService := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, DiagramID: p21RootID,
		Title: "Service", Description: "First service.\n",
	}))
	firstServiceID := onlyNewComponentID(t, firstService.Changes.Components, nil)
	secondService := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, DiagramID: p21RootID,
		Title: "Service", Description: "Second service.\n",
	}))
	secondServiceID := onlyNewComponentID(t, secondService.Changes.Components, map[string]bool{firstServiceID: true})

	firstDetail := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/diagrams/detail", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: firstServiceID, Title: "Service internals",
	}))
	firstServiceDetailID := firstDetail.Changes.DetailDiagrams[len(firstDetail.Changes.DetailDiagrams)-1].ID
	secondDetail := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/diagrams/detail", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: secondServiceID, Title: "Service internals",
	}))
	secondServiceDetailID := secondDetail.Changes.DetailDiagrams[len(secondDetail.Changes.DetailDiagrams)-1].ID

	componentFilenames := make(map[string]string)
	for _, change := range secondDetail.Changes.Components {
		componentFilenames[change.ID] = filepath.Base(changePathByID(t, state, change.ID))
	}
	firstCandidate := diagramAuthoringOptionByID(t, secondDetail.Changes.DiagramOptions, firstServiceDetailID)
	secondCandidate := diagramAuthoringOptionByID(t, secondDetail.Changes.DiagramOptions, secondServiceDetailID)
	if firstCandidate.Context != "Detail for Service — "+componentFilenames[firstServiceID] || secondCandidate.Context != "Detail for Service — "+componentFilenames[secondServiceID] || firstCandidate.Context == secondCandidate.Context {
		t.Fatalf("candidate detail collision contexts = %+v, %+v", firstCandidate, secondCandidate)
	}
}

func diagramAuthoringOptionByID(t *testing.T, options []diagramAuthoringOptionResponse, id string) diagramAuthoringOptionResponse {
	t.Helper()
	for _, option := range options {
		if option.ID == id {
			return option
		}
	}
	t.Fatalf("Diagram option %q missing from %+v", id, options)
	return diagramAuthoringOptionResponse{}
}

func onlyNewComponentID(t *testing.T, changes []pendingComponentResponse, excluded map[string]bool) string {
	t.Helper()
	for _, change := range changes {
		if change.New && !excluded[change.ID] {
			return change.ID
		}
	}
	t.Fatalf("new Component missing from %+v", changes)
	return ""
}

func changePathByID(t *testing.T, handler *Handler, id string) string {
	t.Helper()
	handler.stateMutex.Lock()
	defer handler.stateMutex.Unlock()
	for _, change := range handler.pending.changes {
		if change.ID == id {
			return change.Path
		}
	}
	t.Fatalf("pending Component %q missing", id)
	return ""
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
	accepted := advanceAcceptedToP21V2(t, storePath, initialized.Revision, storeID, "External\\nArchitecture")
	base := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	if got := diagramResponseByID(t, base.Diagrams, p21RootID).Title; got != "External\nArchitecture" {
		t.Fatalf("externally authored multiline title = %q", got)
	}

	renamed := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/diagrams/title", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, DiagramID: p21DetailID, Title: "Renamed\ndetail",
	}))
	if renamed.Changes == nil || renamed.Changes.Candidate == nil || state.pending == nil || state.pending.candidate == nil {
		t.Fatalf("title-only pending = %+v", renamed.Changes)
	}
	before := diagramResponseByID(t, base.Diagrams, p21DetailID)
	after := diagramResponseByID(t, renamed.Changes.Candidate.Diagrams, p21DetailID)
	if after.ID != before.ID || after.Filename != before.Filename || after.Title != "Renamed\ndetail" || !reflect.DeepEqual(after.Appearances, before.Appearances) || !reflect.DeepEqual(after.Boundaries, before.Boundaries) || !reflect.DeepEqual(after.Relationships, before.Relationships) {
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

func TestReviewResponseCaptureRemainsOneGenerationDuringPostUnlockMutation(t *testing.T) {
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	db := openWebDatabaseAt(t, filepath.Join(dataDirectory, "workbraid.db"))
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	rootID := initialized.RootDiagramID
	decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/diagrams/title", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: initialized.Revision, DiagramID: rootID, Title: "First pending title",
	}))

	body, err := json.Marshal(openProjectRequest{SourceRoot: filepath.Clean(source)})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/architecture/review", bytes.NewReader(body))
	request.Header.Set("Origin", testOrigin)
	request.Header.Set("Content-Type", "application/json")
	writer := newGatedResponseWriter()
	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.ServeHTTP(writer, request)
	}()
	<-writer.reachedSerialization

	mutated := decodeArchitectureResponse(t, postDiagramMutation(t, handler, "/api/architecture/diagrams/title", diagramMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: initialized.Revision, DiagramID: rootID, Title: "Second pending title",
	}))
	if mutated.Changes == nil || mutated.Changes.Review != nil || state.pending == nil || state.pending.generation != 2 {
		t.Fatalf("mutation did not invalidate captured binding: %+v", mutated.Changes)
	}
	close(writer.releaseSerialization)
	<-done

	var captured architectureResponse
	if err := json.Unmarshal(writer.body.Bytes(), &captured); err != nil {
		t.Fatalf("decode captured response: %v\n%s", err, writer.body.String())
	}
	if writer.status != http.StatusOK || captured.Changes == nil || captured.Changes.Review == nil || captured.Changes.Review.Generation != 1 || len(captured.Changes.DiagramTitles) != 1 || captured.Changes.DiagramTitles[0].Title != "First pending title" {
		t.Fatalf("mixed review response capture: status=%d changes=%+v", writer.status, captured.Changes)
	}
	if got := diagramResponseByID(t, captured.Changes.Review.WithChanges.Diagrams, rootID).Title; got != "First pending title" {
		t.Fatalf("captured reviewed title = %q", got)
	}
	if confirmation := postAcceptChanges(t, handler, testOrigin, source, *captured.Changes.Review); confirmation.Code != http.StatusConflict {
		t.Fatalf("invalidated captured review confirmation status = %d body=%s", confirmation.Code, confirmation.Body.String())
	}
}

type gatedResponseWriter struct {
	header               http.Header
	body                 bytes.Buffer
	status               int
	reachedSerialization chan struct{}
	releaseSerialization chan struct{}
	once                 sync.Once
}

func newGatedResponseWriter() *gatedResponseWriter {
	return &gatedResponseWriter{
		header: make(http.Header), reachedSerialization: make(chan struct{}), releaseSerialization: make(chan struct{}),
	}
}

func (writer *gatedResponseWriter) Header() http.Header {
	writer.once.Do(func() {
		close(writer.reachedSerialization)
		<-writer.releaseSerialization
	})
	return writer.header
}

func (writer *gatedResponseWriter) WriteHeader(status int) { writer.status = status }

func (writer *gatedResponseWriter) Write(contents []byte) (int, error) {
	return writer.body.Write(contents)
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
