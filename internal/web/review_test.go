package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewAcceptAndFreshApplicationReconstructsExactSuccessor(t *testing.T) {
	source := createSourceRepository(t)
	sourceBefore := snapshotRepository(t, source)
	dataDirectory := t.TempDir()
	databasePath := filepath.Join(dataDirectory, "workbraid.db")
	dbA := openWebDatabaseAt(t, databasePath)
	state, handler := newHandler(dbA, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, dbA, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	manifest := []byte(runGit(t, dataDirectory, "--git-dir", storePath, "show", initialized.Revision+":architecture.yaml") + "\n")
	gatewayID := "32c58784-e3ae-42d3-b32e-13f16f97e304"
	recordsID := "96bd714b-c65a-452d-ae64-f136665d1a6e"
	externalBase := advanceAcceptedToComponents(t, storePath, initialized.Revision, manifest, []testComponent{
		{path: "gateway-internal.md", mode: "100755", source: []byte("---\nid: \"" + gatewayID + "\"\nrelationships:\n  - target: \"" + recordsID + "\"\n    label: reads from\n---\n# Gateway\nAccepted gateway.\n")},
		{path: "records.md", mode: "100644", source: []byte("---\nid: \"" + recordsID + "\"\n---\n# Records\nStores records.\n")},
	})
	base := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	if base.Revision != externalBase || base.ComponentCount != 2 {
		t.Fatalf("accepted fixture did not load: %+v", base)
	}
	associationsBefore := snapshotAssociations(t, dbA)
	manifestEntryBefore := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", base.Revision, "architecture.yaml")
	recordsEntryBefore := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", base.Revision, "components/records.md")

	first := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: externalBase, ComponentID: gatewayID, Title: "Public Gateway", Description: "Receives requests.", TitleChanged: true, DescriptionChanged: true,
	}))
	second := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: externalBase, Title: "Worker", Description: "Processes\x00jobs and literal \\x00.",
	}))
	if first.Changes == nil || second.Changes == nil || len(second.Changes.Components) != 2 {
		t.Fatalf("multi-file pending state missing: %+v", second.Changes)
	}
	i22CandidateTree := state.pending.candidate.Tree()

	reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	if reviewed.Changes == nil || reviewed.Changes.Review == nil {
		t.Fatalf("review response missing exact review: %+v", reviewed.Changes)
	}
	review := *reviewed.Changes.Review
	if review.BaseRevision != base.Revision || review.CandidateTree != i22CandidateTree || review.Generation != 2 ||
		!strings.Contains(review.Diff, "components/gateway-internal.md") || !strings.Contains(review.Diff, "components/worker.md") ||
		!strings.Contains(review.Diff, `id: "`) || !strings.Contains(review.Diff, `+Processes\x00jobs and literal \\x00.`) ||
		strings.Contains(review.Diff, "Binary files differ") || strings.Contains(review.Diff, "No newline at end of file") {
		t.Fatalf("review did not cover complete canonical candidate: %+v\n%s", review, review.Diff)
	}
	if state.pending == nil || state.pending.review == nil || state.pending.candidate == nil ||
		state.pending.review.candidateTree != state.pending.candidate.Tree() || state.pending.review.generation != state.pending.generation {
		t.Fatalf("review is not bound to the exact candidate/generation: %+v", state.pending)
	}

	// A mutation after review advances the pending generation. Browser B reviews
	// that newer generation before browser A submits its older confirmation.
	firstID := second.Changes.Components[0].ID
	mutated := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: externalBase, ComponentID: firstID, Description: "Receives public requests.", DescriptionChanged: true,
	}))
	if mutated.Changes == nil || mutated.Changes.Review != nil || state.pending.generation != 3 {
		t.Fatalf("mutation did not invalidate review: response=%+v pending=%+v", mutated.Changes, state.pending)
	}
	newerReviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	newerReview := *newerReviewed.Changes.Review
	oldConfirmation := postAcceptChanges(t, handler, testOrigin, source, review)
	oldConfirmationBody := decodeArchitectureResponse(t, oldConfirmation)
	if oldConfirmation.Code != http.StatusConflict || oldConfirmationBody.ActionError != errorReviewChanged || oldConfirmationBody.Changes == nil || oldConfirmationBody.Changes.Review != nil {
		t.Fatalf("old review confirmation status=%d body=%s", oldConfirmation.Code, oldConfirmation.Body.String())
	}
	if state.pending == nil || state.pending.review == nil || state.pending.review.candidateTree != newerReview.CandidateTree || state.pending.review.generation != newerReview.Generation {
		t.Fatalf("old browser confirmation displaced newer backend review: %+v", state.pending)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != base.Revision {
		t.Fatalf("old browser confirmation advanced accepted to %s", got)
	}
	review = newerReview

	acceptedResponse := postAcceptChanges(t, handler, testOrigin, source, review)
	accepted := decodeArchitectureResponse(t, acceptedResponse)
	if acceptedResponse.Code != http.StatusOK || accepted.Changes != nil || accepted.Revision == base.Revision || accepted.ComponentCount != 3 || accepted.ParentDiff != review.Diff {
		t.Fatalf("accepted response status=%d body=%s", acceptedResponse.Code, acceptedResponse.Body.String())
	}
	if state.pending != nil || state.loadedSnapshot == nil || state.loadedSnapshot.Revision() != accepted.Revision {
		t.Fatalf("CAS success did not consume/publish exactly once: pending=%+v snapshot=%+v", state.pending, state.loadedSnapshot)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != accepted.Revision {
		t.Fatalf("accepted ref=%s response=%s", got, accepted.Revision)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show", "-s", "--format=%T", accepted.Revision); got != review.CandidateTree {
		t.Fatalf("successor tree=%s reviewed tree=%s", got, review.CandidateTree)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show", "-s", "--format=%P", accepted.Revision); got != base.Revision {
		t.Fatalf("successor parent=%s base=%s", got, base.Revision)
	}
	if entry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", accepted.Revision, "components/gateway-internal.md"); !strings.HasPrefix(entry, "100755 blob ") {
		t.Fatalf("edited component mode was not preserved: %s", entry)
	}
	if source := runGit(t, dataDirectory, "--git-dir", storePath, "show", accepted.Revision+":components/gateway-internal.md"); !strings.Contains(source, `id: "`+gatewayID+`"`) || !strings.Contains(source, `target: "`+recordsID+`"`) || !strings.Contains(source, "# Public Gateway") || !strings.Contains(source, "Receives public requests.") {
		t.Fatalf("accepted edited component lost identity/relationship/authored content: %s", source)
	}
	if entry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", accepted.Revision, "components/worker.md"); !strings.HasPrefix(entry, "100644 blob ") {
		t.Fatalf("new component mode is not 100644: %s", entry)
	}
	if entry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", accepted.Revision, "architecture.yaml"); entry != manifestEntryBefore {
		t.Fatalf("accepted successor rewrote unchanged manifest\nbefore: %s\nafter:  %s", manifestEntryBefore, entry)
	}
	if entry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", accepted.Revision, "components/records.md"); entry != recordsEntryBefore {
		t.Fatalf("accepted successor rewrote unchanged component\nbefore: %s\nafter:  %s", recordsEntryBefore, entry)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show", "-s", "--format=%an <%ae>%n%s", accepted.Revision); got != "WorkBraid <architecture@workbraid.invalid>\nUpdate Architecture" {
		t.Fatalf("uncontrolled commit identity/message: %q", got)
	}
	duplicate := postAcceptChanges(t, handler, testOrigin, source, review)
	if duplicate.Code != http.StatusConflict || !strings.Contains(duplicate.Body.String(), errorReviewFailed) {
		t.Fatalf("duplicate confirmation status=%d body=%s", duplicate.Code, duplicate.Body.String())
	}
	sameProcessReload := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	if sameProcessReload.Revision != accepted.Revision || sameProcessReload.ParentDiff != review.Diff {
		t.Fatalf("same-process reload lost accepted revision/diff inspection: %+v", sameProcessReload)
	}

	if after := snapshotAssociations(t, dbA); after != associationsBefore {
		t.Fatalf("acceptance changed association state\nbefore:\n%s\nafter:\n%s", associationsBefore, after)
	}
	if err := dbA.Close(); err != nil {
		t.Fatal(err)
	}
	dbB := openWebDatabaseAt(t, databasePath)
	newState, newHandler := newHandler(dbB, testOrigin, t.TempDir(), dataDirectory)
	reopened := decodeArchitectureResponse(t, postOpenProject(t, newHandler, testOrigin, source))
	if reopened.Revision != accepted.Revision || reopened.ComponentCount != 3 || reopened.Changes != nil || newState.loadedSnapshot == nil || newState.loadedSnapshot.Revision() != accepted.Revision {
		t.Fatalf("fresh application did not reconstruct successor: %+v", reopened)
	}
	if after := snapshotRepository(t, source); after != sourceBefore {
		t.Fatalf("review/accept/restart changed source repository\nbefore:\n%s\nafter:\n%s", sourceBefore, after)
	}
}

func TestInvalidReviewBlocksConfirmationAndRetainsPendingAcrossBrowserReload(t *testing.T) {
	db := openWebTestDatabase(t)
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	base := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	invalid := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: base.Revision, Title: "   ", Description: "Useful description\n",
	}))
	if invalid.Changes == nil || invalid.Changes.Valid {
		t.Fatalf("invalid pending state missing: %+v", invalid.Changes)
	}
	response := postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source)
	blocked := decodeArchitectureResponse(t, response)
	if response.Code != http.StatusUnprocessableEntity || blocked.Changes == nil || blocked.Changes.Review != nil || blocked.Changes.ReviewBlocker != "title_required" || blocked.ActionError != errorReviewFailed {
		t.Fatalf("invalid review status=%d body=%s", response.Code, response.Body.String())
	}
	if state.pending == nil || len(state.pending.changes) != 1 || state.pending.changes[0].Title != "" {
		t.Fatalf("invalid review discarded pending values: %+v", state.pending)
	}
	reloaded := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	if reloaded.Changes == nil || reloaded.Changes.ReviewBlocker != "title_required" || len(reloaded.Changes.Components) != 1 {
		t.Fatalf("same-process browser reload lost blocked change: %+v", reloaded.Changes)
	}
	if got := runGit(t, dataDirectory, "--git-dir", filepath.Join(dataDirectory, "architecture", associatedStoreID(t, db, filepath.Clean(source))+".git"), "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != base.Revision {
		t.Fatalf("invalid review advanced accepted from %s to %s", base.Revision, got)
	}
	for _, endpoint := range []string{"/api/architecture/review", "/api/architecture/accept"} {
		for _, origin := range []string{"", "http://127.0.0.1:9999"} {
			wrong := postArchitectureAction(t, handler, origin, endpoint, source)
			if wrong.Code != http.StatusForbidden || wrong.Header().Get("Access-Control-Allow-Origin") != "" || !strings.Contains(wrong.Body.String(), errorOriginMismatch) {
				t.Fatalf("origin %q %s status=%d headers=%v body=%s", origin, endpoint, wrong.Code, wrong.Header(), wrong.Body.String())
			}
		}
	}
}

func TestStalePreObservationPreservesPendingAndCreatesNoSuccessor(t *testing.T) {
	db := openWebTestDatabase(t)
	source := createSourceRepository(t)
	sourceBefore := snapshotRepository(t, source)
	dataDirectory := t.TempDir()
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	base := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: base.Revision, Title: "Gateway", Description: "Body\n",
	}))
	reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	storePath := filepath.Join(dataDirectory, "architecture", associatedStoreID(t, db, filepath.Clean(source))+".git")
	baseTree := runGit(t, dataDirectory, "--git-dir", storePath, "show", "-s", "--format=%T", base.Revision)
	external := runGitWithInput(t, dataDirectory, []byte("External Architecture update\n"),
		"-c", "user.name=External Human", "-c", "user.email=human@example.invalid",
		"--git-dir", storePath, "commit-tree", baseTree, "-p", base.Revision)
	runGit(t, dataDirectory, "--git-dir", storePath, "update-ref", "refs/heads/accepted", external, base.Revision)
	objectsBefore := runGit(t, dataDirectory, "--git-dir", storePath, "cat-file", "--batch-all-objects", "--batch-check=%(objecttype) %(objectname)")

	response := postAcceptChanges(t, handler, testOrigin, source, *reviewed.Changes.Review)
	stale := decodeArchitectureResponse(t, response)
	if response.Code != http.StatusConflict || stale.ActionError != errorArchitectureStale || !stale.Stale || stale.Changes == nil {
		t.Fatalf("stale confirmation status=%d body=%s", response.Code, response.Body.String())
	}
	if state.pending == nil || state.pending.review != nil || !state.loadedStale || state.loadedSnapshot.Revision() != base.Revision {
		t.Fatalf("stale state did not preserve pending/read-only base: pending=%+v stale=%v snapshot=%s", state.pending, state.loadedStale, state.loadedSnapshot.Revision())
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != external {
		t.Fatalf("stale confirmation overwrote external accepted: %s", got)
	}
	if objectsAfter := runGit(t, dataDirectory, "--git-dir", storePath, "cat-file", "--batch-all-objects", "--batch-check=%(objecttype) %(objectname)"); objectsAfter != objectsBefore {
		t.Fatalf("pre-observed stale confirmation created an object\nbefore:\n%s\nafter:\n%s", objectsBefore, objectsAfter)
	}
	if after := snapshotRepository(t, source); after != sourceBefore {
		t.Fatalf("stale path changed source repository\nbefore:\n%s\nafter:\n%s", sourceBefore, after)
	}
}

func TestReopenAfterExternalAdvanceKeepsPendingVisibleAsStale(t *testing.T) {
	db := openWebTestDatabase(t)
	source := createSourceRepository(t)
	sourceBefore := snapshotRepository(t, source)
	dataDirectory := t.TempDir()
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	base := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: base.Revision, Title: "Gateway", Description: "Body",
	}))
	reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	if reviewed.Changes == nil || reviewed.Changes.Review == nil {
		t.Fatalf("review missing before external advancement: %+v", reviewed.Changes)
	}
	storePath := filepath.Join(dataDirectory, "architecture", associatedStoreID(t, db, filepath.Clean(source))+".git")
	baseTree := runGit(t, dataDirectory, "--git-dir", storePath, "show", "-s", "--format=%T", base.Revision)
	external := runGitWithInput(t, dataDirectory, []byte("External accepted update\n"),
		"-c", "user.name=External Human", "-c", "user.email=human@example.invalid",
		"--git-dir", storePath, "commit-tree", baseTree, "-p", base.Revision)
	runGit(t, dataDirectory, "--git-dir", storePath, "update-ref", "refs/heads/accepted", external, base.Revision)
	objectsBefore := runGit(t, dataDirectory, "--git-dir", storePath, "cat-file", "--batch-all-objects", "--batch-check=%(objecttype) %(objectname)")

	response := postOpenProject(t, handler, testOrigin, source)
	reopened := decodeArchitectureResponse(t, response)
	if response.Code != http.StatusOK || reopened.Revision != base.Revision || !reopened.Stale || reopened.ActionError != errorArchitectureStale ||
		reopened.Changes == nil || len(reopened.Changes.Components) != 1 || reopened.Changes.Review != nil {
		t.Fatalf("stale reopen status=%d body=%s", response.Code, response.Body.String())
	}
	if state.pending == nil || state.pending.review != nil || !state.loadedStale || state.loadedSnapshot.Revision() != base.Revision {
		t.Fatalf("stale reopen lost pending/base: pending=%+v stale=%v loaded=%s", state.pending, state.loadedStale, state.loadedSnapshot.Revision())
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != external {
		t.Fatalf("stale reopen changed external accepted to %s", got)
	}
	if objectsAfter := runGit(t, dataDirectory, "--git-dir", storePath, "cat-file", "--batch-all-objects", "--batch-check=%(objecttype) %(objectname)"); objectsAfter != objectsBefore {
		t.Fatalf("stale reopen created an object\nbefore:\n%s\nafter:\n%s", objectsBefore, objectsAfter)
	}
	if after := snapshotRepository(t, source); after != sourceBefore {
		t.Fatalf("stale reopen changed source repository\nbefore:\n%s\nafter:\n%s", sourceBefore, after)
	}
}

func TestProductionHandlerFinalCASRacePreservesPendingAndMarksSnapshotStale(t *testing.T) {
	db := openWebTestDatabase(t)
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	base := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: base.Revision, Title: "Gateway", Description: "Body\n",
	}))
	reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	storePath := filepath.Join(dataDirectory, "architecture", associatedStoreID(t, db, filepath.Clean(source))+".git")
	baseTree := runGit(t, dataDirectory, "--git-dir", storePath, "show", "-s", "--format=%T", base.Revision)
	external := runGitWithInput(t, dataDirectory, []byte("Racing accepted update\n"),
		"-c", "user.name=External Human", "-c", "user.email=human@example.invalid",
		"--git-dir", storePath, "commit-tree", baseTree, "-p", base.Revision)
	var workbraidSuccessor string
	state.beforeAcceptedCAS = func(successor string) {
		workbraidSuccessor = successor
		runGit(t, dataDirectory, "--git-dir", storePath, "update-ref", "refs/heads/accepted", external, base.Revision)
	}

	response := postAcceptChanges(t, handler, testOrigin, source, *reviewed.Changes.Review)
	stale := decodeArchitectureResponse(t, response)
	if response.Code != http.StatusConflict || stale.ActionError != errorArchitectureStale || !stale.Stale || stale.Changes == nil {
		t.Fatalf("final-CAS race status=%d body=%s", response.Code, response.Body.String())
	}
	if workbraidSuccessor == "" || state.pending == nil || state.pending.review != nil || !state.loadedStale || state.loadedSnapshot.Revision() != base.Revision {
		t.Fatalf("final-CAS race state successor=%q pending=%+v stale=%v loaded=%s", workbraidSuccessor, state.pending, state.loadedStale, state.loadedSnapshot.Revision())
	}
	if refs := runGit(t, dataDirectory, "--git-dir", storePath, "for-each-ref", "--contains", workbraidSuccessor, "--format=%(refname)"); refs != "" {
		t.Fatalf("unaccepted WorkBraid successor is referenced: %s", refs)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != external {
		t.Fatalf("raced CAS moved accepted to %s", got)
	}
}

func TestRefLockFailurePreservesPendingAndPostCASPublicationFailureConsumesIt(t *testing.T) {
	t.Run("ref lock before successful CAS", func(t *testing.T) {
		db := openWebTestDatabase(t)
		source := createSourceRepository(t)
		dataDirectory := t.TempDir()
		state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
		base := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
		decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{SourceRoot: filepath.Clean(source), ExpectedRevision: base.Revision, Title: "Gateway"}))
		reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
		storePath := filepath.Join(dataDirectory, "architecture", associatedStoreID(t, db, filepath.Clean(source))+".git")
		lock := filepath.Join(storePath, "refs", "heads", "accepted.lock")
		if err := os.WriteFile(lock, []byte("held"), 0o600); err != nil {
			t.Fatal(err)
		}
		response := postAcceptChanges(t, handler, testOrigin, source, *reviewed.Changes.Review)
		if response.Code != http.StatusInternalServerError || decodeArchitectureResponse(t, response).ActionError != errorUpdateFailed || state.pending == nil {
			t.Fatalf("ref-lock failure status=%d body=%s pending=%+v", response.Code, response.Body.String(), state.pending)
		}
		if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != base.Revision {
			t.Fatalf("ref-lock failure changed accepted: %s", got)
		}
	})

	t.Run("request cancellation does not interrupt a confirmed authority transition", func(t *testing.T) {
		db := openWebTestDatabase(t)
		source := createSourceRepository(t)
		dataDirectory := t.TempDir()
		state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
		base := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
		decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{SourceRoot: filepath.Clean(source), ExpectedRevision: base.Revision, Title: "Gateway"}))
		reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
		payload := acceptChangesRequest{
			SourceRoot: filepath.Clean(source), BaseRevision: reviewed.Changes.Review.BaseRevision,
			CandidateTree: reviewed.Changes.Review.CandidateTree, Generation: reviewed.Changes.Review.Generation,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, "/api/architecture/accept", bytes.NewReader(body))
		request.Header.Set("Origin", testOrigin)
		request.Header.Set("Content-Type", "application/json")
		ctx, cancel := context.WithCancel(request.Context())
		cancel()
		request = request.WithContext(ctx)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		accepted := decodeArchitectureResponse(t, response)
		if response.Code != http.StatusOK || accepted.Revision == base.Revision || state.pending != nil || state.loadedSnapshot.Revision() != accepted.Revision {
			t.Fatalf("canceled-request transition status=%d body=%s pending=%+v", response.Code, response.Body.String(), state.pending)
		}
		storePath := filepath.Join(dataDirectory, "architecture", associatedStoreID(t, db, filepath.Clean(source))+".git")
		if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != accepted.Revision {
			t.Fatalf("canceled request did not complete authority transition: %s", got)
		}
	})

	t.Run("reported failure after successful CAS is consumed once", func(t *testing.T) {
		db := openWebTestDatabase(t)
		source := createSourceRepository(t)
		dataDirectory := t.TempDir()
		state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
		base := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
		decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{SourceRoot: filepath.Clean(source), ExpectedRevision: base.Revision, Title: "Gateway"}))
		reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
		reported := 0
		state.acceptedUpdateReportFailure = func() error {
			reported++
			return errors.New("reported failure after real accepted update")
		}

		response := postAcceptChanges(t, handler, testOrigin, source, *reviewed.Changes.Review)
		accepted := decodeArchitectureResponse(t, response)
		if response.Code != http.StatusOK || accepted.Revision == base.Revision || accepted.Changes != nil || state.pending != nil || reported != 1 {
			t.Fatalf("reported-error-after-success status=%d body=%s pending=%+v reports=%d", response.Code, response.Body.String(), state.pending, reported)
		}
		storePath := filepath.Join(dataDirectory, "architecture", associatedStoreID(t, db, filepath.Clean(source))+".git")
		if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != accepted.Revision {
			t.Fatalf("observed attempted successor=%s response=%s", got, accepted.Revision)
		}
		duplicate := postAcceptChanges(t, handler, testOrigin, source, *reviewed.Changes.Review)
		if duplicate.Code != http.StatusConflict || !strings.Contains(duplicate.Body.String(), errorReviewFailed) {
			t.Fatalf("reported-error path offered duplicate status=%d body=%s", duplicate.Code, duplicate.Body.String())
		}
	})

	t.Run("publication fails after CAS", func(t *testing.T) {
		source := createSourceRepository(t)
		sourceBefore := snapshotRepository(t, source)
		dataDirectory := t.TempDir()
		databasePath := filepath.Join(dataDirectory, "workbraid.db")
		db := openWebDatabaseAt(t, databasePath)
		state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
		base := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
		decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{SourceRoot: filepath.Clean(source), ExpectedRevision: base.Revision, Title: "Gateway"}))
		reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
		state.publicationFailure = func() error { return errors.New("focused publication failure") }
		response := postAcceptChanges(t, handler, testOrigin, source, *reviewed.Changes.Review)
		updated := decodeArchitectureResponse(t, response)
		if response.Code != http.StatusInternalServerError || updated.ActionError != errorUpdatedReload || updated.Revision == base.Revision || updated.Changes != nil || state.pending != nil {
			t.Fatalf("post-CAS response status=%d body=%s pending=%+v", response.Code, response.Body.String(), state.pending)
		}
		storePath := filepath.Join(dataDirectory, "architecture", associatedStoreID(t, db, filepath.Clean(source))+".git")
		if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != updated.Revision {
			t.Fatalf("post-CAS failure lost successor: %s", got)
		}
		duplicate := postAcceptChanges(t, handler, testOrigin, source, *reviewed.Changes.Review)
		if duplicate.Code != http.StatusConflict || !strings.Contains(duplicate.Body.String(), errorReviewFailed) {
			t.Fatalf("post-CAS duplicate status=%d body=%s", duplicate.Code, duplicate.Body.String())
		}
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		dbFresh := openWebDatabaseAt(t, databasePath)
		fresh := NewHandler(dbFresh, testOrigin, t.TempDir(), dataDirectory)
		reopened := decodeArchitectureResponse(t, postOpenProject(t, fresh, testOrigin, source))
		if reopened.Revision != updated.Revision || reopened.ComponentCount != 1 || reopened.Changes != nil {
			t.Fatalf("fresh load after post-CAS failure = %+v", reopened)
		}
		if after := snapshotRepository(t, source); after != sourceBefore {
			t.Fatalf("post-CAS recovery changed source repository\nbefore:\n%s\nafter:\n%s", sourceBefore, after)
		}
	})
}

func postArchitectureAction(t *testing.T, handler http.Handler, origin, path, sourceRoot string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"source_root": filepath.Clean(sourceRoot)})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func postAcceptChanges(t *testing.T, handler http.Handler, origin, sourceRoot string, review reviewResponse) *httptest.ResponseRecorder {
	t.Helper()
	payload := acceptChangesRequest{
		SourceRoot: filepath.Clean(sourceRoot), BaseRevision: review.BaseRevision,
		CandidateTree: review.CandidateTree, Generation: review.Generation,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/architecture/accept", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
