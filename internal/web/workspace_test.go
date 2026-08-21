package web

import (
	"net/http"
	"path/filepath"
	"sync"
	"testing"
)

func TestDiscardChangesClearsOnlyInProcessPendingAndReviewState(t *testing.T) {
	db := openWebTestDatabase(t)
	source := createSourceRepository(t)
	sourceBefore := snapshotRepository(t, source)
	dataDirectory := t.TempDir()
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")

	changed := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: initialized.Revision, Title: "Worker", Description: "Does work.",
	}))
	if changed.Changes == nil || !changed.Changes.Valid {
		t.Fatalf("pending change not created: %+v", changed)
	}
	reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	if reviewed.Changes == nil || reviewed.Changes.Review == nil || state.pending == nil || state.pending.review == nil {
		t.Fatalf("review not created: %+v", reviewed.Changes)
	}
	review := *reviewed.Changes.Review
	acceptedBefore := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted")
	objectsBefore := runGit(t, dataDirectory, "--git-dir", storePath, "cat-file", "--batch-all-objects", "--batch-check=%(objectname)")
	associationsBefore := snapshotAssociations(t, db)
	snapshotBefore := *state.loadedSnapshot

	discarded := postArchitectureAction(t, handler, testOrigin, "/api/architecture/discard", source)
	discardedBody := decodeArchitectureResponse(t, discarded)
	if discarded.Code != http.StatusOK || discardedBody.Changes != nil || state.pending != nil {
		t.Fatalf("discard response status=%d body=%s pending=%+v", discarded.Code, discarded.Body.String(), state.pending)
	}
	if state.loadedSnapshot == nil || state.loadedSnapshot.Revision() != snapshotBefore.Revision() || state.loadedSnapshot.StoreID() != snapshotBefore.StoreID() {
		t.Fatalf("discard changed loaded accepted snapshot: before=%s after=%+v", snapshotBefore.Revision(), state.loadedSnapshot)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != acceptedBefore {
		t.Fatalf("discard changed accepted ref from %s to %s", acceptedBefore, got)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "cat-file", "--batch-all-objects", "--batch-check=%(objectname)"); got != objectsBefore {
		t.Fatalf("discard changed Git objects\nbefore:\n%s\nafter:\n%s", objectsBefore, got)
	}
	if got := snapshotAssociations(t, db); got != associationsBefore {
		t.Fatalf("discard changed SQLite state\nbefore:\n%s\nafter:\n%s", associationsBefore, got)
	}
	if got := snapshotRepository(t, source); got != sourceBefore {
		t.Fatalf("discard changed source repository\nbefore:\n%s\nafter:\n%s", sourceBefore, got)
	}
	oldConfirmation := postAcceptChanges(t, handler, testOrigin, source, review)
	if oldConfirmation.Code != http.StatusConflict {
		t.Fatalf("discarded review remained confirmable: status=%d body=%s", oldConfirmation.Code, oldConfirmation.Body.String())
	}
	newPending := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: initialized.Revision, Title: "Gateway", Description: "Routes calls.",
	}))
	if newPending.Changes == nil || state.pending == nil || state.pending.baseRevision != initialized.Revision {
		t.Fatalf("new pending did not bind to loaded accepted revision: %+v", newPending.Changes)
	}
}

func TestDiscardChangesEnforcesOrigin(t *testing.T) {
	db := openWebTestDatabase(t)
	source := createSourceRepository(t)
	handler := NewHandler(db, testOrigin, t.TempDir(), t.TempDir())
	base := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: base.Revision, Title: "Worker",
	}))

	for _, origin := range []string{"", "http://127.0.0.1:9999"} {
		response := postArchitectureAction(t, handler, origin, "/api/architecture/discard", source)
		if response.Code != http.StatusForbidden || response.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("origin %q status=%d headers=%v body=%s", origin, response.Code, response.Header(), response.Body.String())
		}
	}
	reopened := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	if reopened.Changes == nil {
		t.Fatal("rejected discard removed pending changes")
	}
}

func TestOpenAnotherProjectIsAtomicWithPendingMutation(t *testing.T) {
	db := openWebTestDatabase(t)
	first := createSourceRepository(t)
	second := createSourceRepository(t)
	dataDirectory := t.TempDir()
	setupFirst := NewHandler(db, testOrigin, t.TempDir(), dataDirectory)
	firstBase := decodeArchitectureResponse(t, postInitializeProject(t, setupFirst, testOrigin, first))
	setupSecond := NewHandler(db, testOrigin, t.TempDir(), dataDirectory)
	decodeArchitectureResponse(t, postInitializeProject(t, setupSecond, testOrigin, second))

	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, first))
	start := make(chan struct{})
	var responses [2]responseResult
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		<-start
		response := postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
			SourceRoot: filepath.Clean(first), ExpectedRevision: firstBase.Revision, Title: "Worker",
		})
		responses[0] = responseResult{status: response.Code, body: response.Body.String()}
	}()
	go func() {
		defer wait.Done()
		<-start
		response := postOpenProject(t, handler, testOrigin, second)
		responses[1] = responseResult{status: response.Code, body: response.Body.String()}
	}()
	close(start)
	wait.Wait()

	state.stateMutex.Lock()
	defer state.stateMutex.Unlock()
	mutationWon := responses[0].status == http.StatusOK && responses[1].status == http.StatusConflict
	switchWon := responses[0].status == http.StatusConflict && responses[1].status == http.StatusOK
	if !mutationWon && !switchWon {
		t.Fatalf("atomic race had invalid outcomes: mutation=%+v switch=%+v", responses[0], responses[1])
	}
	if mutationWon {
		if state.loadedProject == nil || state.loadedProject.sourceRoot != filepath.Clean(first) || state.pending == nil || len(state.pending.changes) != 1 {
			t.Fatalf("mutation-first left inconsistent state: project=%+v pending=%+v", state.loadedProject, state.pending)
		}
		return
	}
	if state.loadedProject == nil || state.loadedProject.sourceRoot != filepath.Clean(second) || state.pending != nil {
		t.Fatalf("switch-first left hidden old pending state: project=%+v pending=%+v", state.loadedProject, state.pending)
	}
}

type responseResult struct {
	status int
	body   string
}

func TestLeaveProjectRequiresDiscardWhenChangesExist(t *testing.T) {
	db := openWebTestDatabase(t)
	source := createSourceRepository(t)
	state, handler := newHandler(db, testOrigin, t.TempDir(), t.TempDir())
	base := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: base.Revision, Title: "Worker",
	}))

	blocked := postArchitectureAction(t, handler, testOrigin, "/api/projects/leave", source)
	if blocked.Code != http.StatusConflict || decodeArchitectureResponse(t, blocked).ActionError != errorPendingBlocksSwitch || state.loadedProject == nil || state.pending == nil {
		t.Fatalf("leave did not preserve workspace and pending: status=%d body=%s", blocked.Code, blocked.Body.String())
	}
	postArchitectureAction(t, handler, testOrigin, "/api/architecture/discard", source)
	left := postArchitectureAction(t, handler, testOrigin, "/api/projects/leave", source)
	if left.Code != http.StatusNoContent || state.loadedProject != nil || state.loadedSnapshot != nil || state.pending != nil {
		t.Fatalf("leave after discard status=%d project=%+v snapshot=%+v pending=%+v", left.Code, state.loadedProject, state.loadedSnapshot, state.pending)
	}
}

func TestLeaveProjectIsAtomicWithPendingMutation(t *testing.T) {
	db := openWebTestDatabase(t)
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	setup := NewHandler(db, testOrigin, t.TempDir(), dataDirectory)
	decodeArchitectureResponse(t, postInitializeProject(t, setup, testOrigin, source))
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	base := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))

	start := make(chan struct{})
	var responses [2]responseResult
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		<-start
		response := postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
			SourceRoot: filepath.Clean(source), ExpectedRevision: base.Revision, Title: "Worker",
		})
		responses[0] = responseResult{status: response.Code, body: response.Body.String()}
	}()
	go func() {
		defer wait.Done()
		<-start
		response := postArchitectureAction(t, handler, testOrigin, "/api/projects/leave", source)
		responses[1] = responseResult{status: response.Code, body: response.Body.String()}
	}()
	close(start)
	wait.Wait()

	state.stateMutex.Lock()
	defer state.stateMutex.Unlock()
	mutationWon := responses[0].status == http.StatusOK && responses[1].status == http.StatusConflict
	leaveWon := responses[0].status == http.StatusConflict && responses[1].status == http.StatusNoContent
	if !mutationWon && !leaveWon {
		t.Fatalf("atomic leave race had invalid outcomes: mutation=%+v leave=%+v", responses[0], responses[1])
	}
	if mutationWon {
		if state.loadedProject == nil || state.pending == nil || len(state.pending.changes) != 1 {
			t.Fatalf("mutation-first leave race lost workspace: project=%+v pending=%+v", state.loadedProject, state.pending)
		}
		return
	}
	if state.loadedProject != nil || state.loadedSnapshot != nil || state.pending != nil {
		t.Fatalf("leave-first race retained hidden state: project=%+v snapshot=%+v pending=%+v", state.loadedProject, state.loadedSnapshot, state.pending)
	}
}
