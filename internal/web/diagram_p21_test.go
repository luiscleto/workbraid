package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

const (
	p21RootID    = "11111111-1111-4111-8111-111111111111"
	p21DetailID  = "22222222-2222-4222-8222-222222222222"
	p21EmptyID   = "33333333-3333-4333-8333-333333333333"
	p21GatewayID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	p21WorkerID  = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	p21RecordsID = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
)

func TestAcceptedV2HandlerProjectionIsReadOnlyAndRestartable(t *testing.T) {
	source := createSourceRepository(t)
	sourceBefore := snapshotRepository(t, source)
	dataDirectory := t.TempDir()
	db := openWebDatabaseAt(t, filepath.Join(dataDirectory, "workbraid.db"))
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	accepted := advanceAcceptedToP21V2(t, storePath, initialized.Revision, storeID, "System")
	associationsBefore := snapshotAssociations(t, db)
	gitBefore := snapshotPrivateArchitecture(t, dataDirectory)

	opened := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	if opened.Revision != accepted || opened.FormatVersion != 2 || opened.RootDiagramID != p21RootID || len(opened.Diagrams) != 3 {
		t.Fatalf("v2 open projection = %+v", opened)
	}
	root := diagramResponseByID(t, opened.Diagrams, p21RootID)
	detail := diagramResponseByID(t, opened.Diagrams, p21DetailID)
	if detail.ParentDiagramID != p21RootID || detail.ParentAnchorComponentID != p21GatewayID || len(detail.Breadcrumbs) != 2 {
		t.Fatalf("detail hierarchy projection = %+v", detail)
	}
	if len(detail.Boundaries) != 2 {
		t.Fatalf("detail boundary projection = %+v", detail.Boundaries)
	}
	var recordsBoundary diagramBoundaryResponse
	for _, boundary := range detail.Boundaries {
		if boundary.ComponentID == p21RecordsID {
			recordsBoundary = boundary
		}
	}
	if recordsBoundary.HomeDiagramID != p21RootID {
		t.Fatalf("Records boundary = %+v", recordsBoundary)
	}
	recordsRelationshipCount := 0
	for _, relationship := range detail.Relationships {
		if relationship.SourceComponentID == p21RecordsID && relationship.SourceNodeKey != recordsBoundary.Key {
			t.Fatalf("crossing source did not use shared boundary: %+v", relationship)
		}
		if relationship.TargetComponentID == p21RecordsID && relationship.TargetNodeKey != recordsBoundary.Key {
			t.Fatalf("crossing target did not use shared boundary: %+v", relationship)
		}
		if relationship.SourceComponentID == p21RecordsID || relationship.TargetComponentID == p21RecordsID {
			recordsRelationshipCount++
		}
	}
	if recordsRelationshipCount != 3 {
		t.Fatalf("Records crossing relationships = %+v", detail.Relationships)
	}
	if root.Appearances[1].Role != "reference" || root.Appearances[1].ComponentID != p21WorkerID {
		t.Fatalf("canonical reference projection = %+v", root.Appearances)
	}

	for _, endpoint := range []string{"/api/architecture/components/add", "/api/architecture/components/edit"} {
		request := componentMutationRequest{SourceRoot: filepath.Clean(source), Title: "Late v1 edit", Description: "Must not persist."}
		if strings.HasSuffix(endpoint, "/edit") {
			request.ComponentID = p21GatewayID
			request.TitleChanged = true
		}
		response := postComponentMutation(t, handler, testOrigin, endpoint, request)
		var failure errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &failure); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusConflict || failure.Code != errorChangesUnavailable || state.pending != nil {
			t.Fatalf("v2 mutation %s status=%d failure=%+v pending=%+v", endpoint, response.Code, failure, state.pending)
		}
	}
	if got := snapshotPrivateArchitecture(t, dataDirectory); got != gitBefore {
		t.Fatal("v2 read-only request changed private Git")
	}
	if got := snapshotRepository(t, source); got != sourceBefore {
		t.Fatal("v2 read-only request changed source repository")
	}
	if got := snapshotAssociations(t, db); got != associationsBefore {
		t.Fatal("v2 read-only request changed SQLite")
	}

	freshDB := openWebDatabaseAt(t, filepath.Join(dataDirectory, "workbraid.db"))
	_, freshHandler := newHandler(freshDB, testOrigin, t.TempDir(), dataDirectory)
	reopened := decodeArchitectureResponse(t, postOpenProject(t, freshHandler, testOrigin, source))
	if reopened.Revision != accepted || reopened.FormatVersion != 2 || reopened.RootDiagramID != p21RootID || len(reopened.Diagrams) != 3 {
		t.Fatalf("fresh handler did not reconstruct v2: %+v", reopened)
	}
}

func TestRefreshFromV1ToV2MakesOldPendingStaleAndRejectsLateMutation(t *testing.T) {
	db := openWebTestDatabase(t)
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	kept := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), Title: "Pending v1", Description: "Old-base work.",
	}))
	if kept.Changes == nil || state.pending == nil {
		t.Fatal("v1 pending work was not created")
	}
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	accepted := advanceAcceptedToP21V2(t, storePath, initialized.Revision, storeID, "External v2")

	refreshedResponse := postArchitectureAction(t, handler, testOrigin, "/api/architecture/refresh", source)
	refreshed := decodeArchitectureResponse(t, refreshedResponse)
	if refreshedResponse.Code != http.StatusOK || refreshed.Revision != accepted || refreshed.FormatVersion != 2 || refreshed.Stale || refreshed.Changes == nil || !refreshed.Changes.Stale {
		t.Fatalf("v1 to v2 Refresh = status %d, %+v", refreshedResponse.Code, refreshed)
	}
	late := postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{SourceRoot: filepath.Clean(source), Title: "Hidden"})
	var failure errorResponse
	if err := json.Unmarshal(late.Body.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if late.Code != http.StatusConflict || failure.Code != errorChangesUnavailable || len(state.pending.changes) != 1 || state.pending.changes[0].Title != "Pending v1" {
		t.Fatalf("late mutation changed stale pending: status=%d failure=%+v pending=%+v", late.Code, failure, state.pending)
	}
	discarded := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/discard", source))
	if discarded.Changes != nil || discarded.Revision != accepted || discarded.FormatVersion != 2 {
		t.Fatalf("discard after v2 adoption = %+v", discarded)
	}
}

func TestAcceptedV2RefreshIsQuietThenAdoptsNonLinearReplacement(t *testing.T) {
	db := openWebTestDatabase(t)
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	_, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	first := advanceAcceptedToP21V2(t, storePath, initialized.Revision, storeID, "System A")
	opened := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	unchanged := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/refresh", source))
	if opened.Revision != first || unchanged.Revision != first || unchanged.ActionError != "" || unchanged.Stale {
		t.Fatalf("unchanged v2 Refresh = open %+v, Refresh %+v", opened, unchanged)
	}

	second := advanceAcceptedToP21V2From(t, storePath, initialized.Revision, first, storeID, "System B")
	if parents := strings.Fields(runGit(t, storePath, "--git-dir", storePath, "show", "-s", "--format=%P", second)); len(parents) != 1 || parents[0] != initialized.Revision {
		t.Fatalf("replacement is not non-linear: parent=%q base=%q", parents, initialized.Revision)
	}
	replaced := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/refresh", source))
	if replaced.Revision != second || replaced.FormatVersion != 2 || replaced.Stale || diagramResponseByID(t, replaced.Diagrams, p21RootID).Title != "System B" {
		t.Fatalf("non-linear v2 Refresh = %+v", replaced)
	}
}

func TestRefreshToAcceptedV2SerializesLateV1MutationEligibility(t *testing.T) {
	db := openWebTestDatabase(t)
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	accepted := advanceAcceptedToP21V2(t, storePath, initialized.Revision, storeID, "System")

	refreshAtFinalObservation := make(chan struct{})
	releaseRefresh := make(chan struct{})
	state.beforeRefreshReobserve = func(string) {
		close(refreshAtFinalObservation)
		<-releaseRefresh
	}
	refreshDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		refreshDone <- postArchitectureAction(t, handler, testOrigin, "/api/architecture/refresh", source)
	}()
	<-refreshAtFinalObservation

	mutationStarted := make(chan struct{})
	mutationDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		close(mutationStarted)
		mutationDone <- postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
			SourceRoot: filepath.Clean(source), Title: "Late v1 mutation", Description: "Must not persist.",
		})
	}()
	<-mutationStarted
	close(releaseRefresh)
	refreshResponse := <-refreshDone
	mutationResponse := <-mutationDone

	refreshed := decodeArchitectureResponse(t, refreshResponse)
	var failure errorResponse
	if err := json.Unmarshal(mutationResponse.Body.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if refreshResponse.Code != http.StatusOK || refreshed.Revision != accepted || refreshed.FormatVersion != 2 {
		t.Fatalf("raced Refresh = status %d, %+v", refreshResponse.Code, refreshed)
	}
	if mutationResponse.Code != http.StatusConflict || failure.Code != errorChangesUnavailable || state.pending != nil {
		t.Fatalf("raced mutation = status %d, %+v, pending=%+v", mutationResponse.Code, failure, state.pending)
	}
	if state.loadedSnapshot == nil || state.loadedSnapshot.Revision() != accepted || state.loadedSnapshot.FormatVersion() != 2 || state.loadedStale {
		t.Fatalf("mixed loaded state after race: snapshot=%+v stale=%t", state.loadedSnapshot, state.loadedStale)
	}
}

func diagramResponseByID(t *testing.T, values []diagramResponse, id string) diagramResponse {
	t.Helper()
	for _, value := range values {
		if value.ID == id {
			return value
		}
	}
	t.Fatalf("Diagram %s not found in %+v", id, values)
	return diagramResponse{}
}

func advanceAcceptedToP21V2(t *testing.T, storePath, oldRevision, storeID, rootTitle string) string {
	return advanceAcceptedToP21V2From(t, storePath, oldRevision, oldRevision, storeID, rootTitle)
}

func advanceAcceptedToP21V2From(t *testing.T, storePath, parentRevision, expectedRevision, storeID, rootTitle string) string {
	t.Helper()
	manifest := "format: workbraid-architecture\nversion: 2\nstore_id: \"" + storeID + "\"\nproject:\n  name: Project\n  source_hint: /tmp/project\nroot_diagram: \"" + p21RootID + "\"\n"
	components := []testComponent{
		{path: "gateway.md", mode: "100644", source: []byte("---\nid: \"" + p21GatewayID + "\"\nrelationships:\n  - target: \"" + p21WorkerID + "\"\n    label: calls\n---\n# Shared\nGateway docs.\n")},
		{path: "worker.md", mode: "100755", source: []byte("---\nid: \"" + p21WorkerID + "\"\nrelationships:\n  - target: \"" + p21RecordsID + "\"\n    label: writes\n  - target: \"" + p21RecordsID + "\"\n    label: writes\n---\n# Worker\nWorker docs.\n")},
		{path: "records.md", mode: "100644", source: []byte("---\nid: \"" + p21RecordsID + "\"\nrelationships:\n  - target: \"" + p21WorkerID + "\"\n    label: feeds\n---\n# Shared\nRecords docs.\n")},
	}
	manifestBlob := runGitWithInput(t, storePath, []byte(manifest), "--git-dir", storePath, "hash-object", "-w", "--stdin")
	componentEntries := make([]string, len(components))
	for index, component := range components {
		blob := runGitWithInput(t, storePath, component.source, "--git-dir", storePath, "hash-object", "-w", "--stdin")
		componentEntries[index] = component.mode + " blob " + blob + "\t" + component.path
	}
	componentTree := runGitWithInput(t, storePath, []byte(strings.Join(componentEntries, "\n")+"\n"), "--git-dir", storePath, "mktree")
	diagrams := []struct{ name, source string }{
		{"root.yaml", "id: \"" + p21RootID + "\"\ntitle: \"" + rootTitle + "\"\nappearances:\n  - component: \"" + p21GatewayID + "\"\n    role: home\n    detail_diagram: \"" + p21DetailID + "\"\n  - component: \"" + p21WorkerID + "\"\n    role: reference\n  - component: \"" + p21RecordsID + "\"\n    role: home\n    detail_diagram: \"" + p21EmptyID + "\"\n"},
		{"detail.yaml", "id: \"" + p21DetailID + "\"\ntitle: Detail\nappearances:\n  - component: \"" + p21WorkerID + "\"\n    role: home\n"},
		{"empty.yaml", "id: \"" + p21EmptyID + "\"\ntitle: Detail\nappearances: []\n"},
	}
	diagramEntries := make([]string, len(diagrams))
	for index, diagram := range diagrams {
		blob := runGitWithInput(t, storePath, []byte(diagram.source), "--git-dir", storePath, "hash-object", "-w", "--stdin")
		diagramEntries[index] = "100644 blob " + blob + "\t" + diagram.name
	}
	diagramTree := runGitWithInput(t, storePath, []byte(strings.Join(diagramEntries, "\n")+"\n"), "--git-dir", storePath, "mktree")
	rootTree := runGitWithInput(t, storePath, []byte("100644 blob "+manifestBlob+"\tarchitecture.yaml\n040000 tree "+componentTree+"\tcomponents\n040000 tree "+diagramTree+"\tdiagrams\n"), "--git-dir", storePath, "mktree")
	commit := runGitWithInput(t, storePath, []byte("accepted v2 fixture\n"), "-c", "user.name=Test", "-c", "user.email=test@workbraid.invalid", "--git-dir", storePath, "commit-tree", rootTree, "-p", parentRevision)
	runGit(t, storePath, "--git-dir", storePath, "update-ref", "refs/heads/accepted", commit, expectedRevision)
	return commit
}
