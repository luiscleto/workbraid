package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestPendingComponentAuthoringUsesOneBackendCandidateAndLeavesAcceptedUnchanged(t *testing.T) {
	db := openWebTestDatabase(t)
	source := createSourceRepository(t)
	sourceBefore := snapshotRepository(t, source)
	dataDirectory := t.TempDir()
	handler := NewHandler(db, testOrigin, t.TempDir(), dataDirectory)

	initialized := postInitializeProject(t, handler, testOrigin, source)
	var empty architectureResponse
	if initialized.Code != http.StatusOK || json.Unmarshal(initialized.Body.Bytes(), &empty) != nil {
		t.Fatalf("initialize status=%d body=%s", initialized.Code, initialized.Body.String())
	}
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	manifest := []byte(runGit(t, dataDirectory, "--git-dir", storePath, "show", empty.Revision+":architecture.yaml") + "\n")
	apiID := uuid.NewString()
	workerID := uuid.NewString()
	apiSource := []byte("---\nid: \"" + apiID + "\"\nrelationships:\n  - target: \"" + workerID + "\"\n    label: calls\n---\n# API #\n\nAccepted API body\n")
	workerSource := []byte("---\nid: \"" + workerID + "\"\n---\nWorker\n======\nWorker body\n")
	accepted := advanceAcceptedToComponents(t, storePath, empty.Revision, manifest, []testComponent{
		{path: "api.md", mode: "100755", source: apiSource},
		{path: "worker.md", mode: "100644", source: workerSource},
	})
	opened := postOpenProject(t, handler, testOrigin, source)
	if opened.Code != http.StatusOK {
		t.Fatalf("open status=%d body=%s", opened.Code, opened.Body.String())
	}

	acceptedTree := runGit(t, dataDirectory, "--git-dir", storePath, "show", "-s", "--format=%T", accepted)
	acceptedAPIEntry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", accepted, "components/api.md")
	acceptedWorkerEntry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", accepted, "components/worker.md")
	associationBefore := snapshotAssociations(t, db)
	reachableBefore := runGit(t, dataDirectory, "--git-dir", storePath, "rev-list", "--objects", "--all")

	edited := postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: apiID, Title: "  Gateway  ", Description: "\nChanged API body\n", TitleChanged: true, DescriptionChanged: true,
	})
	editedBody := decodeArchitectureResponse(t, edited)
	if edited.Code != http.StatusOK || editedBody.Changes == nil || !editedBody.Changes.Valid || len(editedBody.Changes.Components) != 1 || editedBody.Changes.Components[0].Title != "Gateway" {
		t.Fatalf("edit response status=%d body=%s", edited.Code, edited.Body.String())
	}
	added := postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, Title: "  API  ", Description: "\nNew API body",
	})
	addedBody := decodeArchitectureResponse(t, added)
	if added.Code != http.StatusOK || addedBody.Changes == nil || !addedBody.Changes.Valid || len(addedBody.Changes.Components) != 2 {
		t.Fatalf("add response status=%d body=%s", added.Code, added.Body.String())
	}
	var newID string
	for _, change := range addedBody.Changes.Components {
		if change.New {
			newID = change.ID
			if change.Title != "API" {
				t.Fatalf("new pending title = %q, want normalized API", change.Title)
			}
			if change.Description != "\nNew API body\n" {
				t.Fatalf("new pending description = %q, want trailing newline", change.Description)
			}
		}
	}
	if _, err := uuid.Parse(newID); err != nil {
		t.Fatalf("generated component ID = %q", newID)
	}

	invalid := postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: newID, Title: "   ", TitleChanged: true,
	})
	invalidBody := decodeArchitectureResponse(t, invalid)
	if invalid.Code != http.StatusOK || invalidBody.Changes == nil || invalidBody.Changes.Valid || invalidBody.Changes.ValidationCode != "title_required" || invalidBody.Changes.ValidationItem != newID || len(invalidBody.Changes.Components) != 2 {
		t.Fatalf("invalid response status=%d body=%s", invalid.Code, invalid.Body.String())
	}
	for _, change := range invalidBody.Changes.Components {
		if change.ID == newID && change.Title != "" {
			t.Fatalf("invalid pending title = %q, want normalized empty title", change.Title)
		}
	}

	// Reopening through a fresh HTTP request against the same handler models a
	// browser reload without granting cross-process persistence.
	reloaded := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	if reloaded.Changes == nil || reloaded.Changes.Valid || len(reloaded.Changes.Components) != 2 || reloaded.Changes.ValidationItem != newID {
		t.Fatalf("same-process reload lost pending state: %+v", reloaded.Changes)
	}
	corrected := postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: newID, Title: "  API helper  ", TitleChanged: true,
	})
	correctedBody := decodeArchitectureResponse(t, corrected)
	if correctedBody.Changes == nil || !correctedBody.Changes.Valid || len(correctedBody.Changes.Components) != 2 {
		t.Fatalf("corrected response = %s", corrected.Body.String())
	}
	for _, change := range correctedBody.Changes.Components {
		if change.ID == newID && change.Title != "API helper" {
			t.Fatalf("corrected pending title = %q, want normalized API helper", change.Title)
		}
	}
	if correctedBody.Revision != accepted || strings.Join(correctedBody.ComponentTitles, "|") != "API|Worker" {
		t.Fatalf("accepted response advanced during authoring: %+v", correctedBody)
	}
	newProcessHandler := NewHandler(db, testOrigin, t.TempDir(), dataDirectory)
	newProcessView := decodeArchitectureResponse(t, postOpenProject(t, newProcessHandler, testOrigin, source))
	if newProcessView.Changes != nil || newProcessView.Revision != accepted {
		t.Fatalf("new backend unexpectedly recovered in-process changes: %+v", newProcessView)
	}

	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != accepted {
		t.Fatalf("accepted ref changed from %q to %q", accepted, got)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show", "-s", "--format=%T", accepted); got != acceptedTree {
		t.Fatalf("accepted tree changed from %q to %q", acceptedTree, got)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", accepted, "components/api.md"); got != acceptedAPIEntry {
		t.Fatalf("accepted API entry changed from %q to %q", acceptedAPIEntry, got)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", accepted, "components/worker.md"); got != acceptedWorkerEntry {
		t.Fatalf("accepted Worker entry changed from %q to %q", acceptedWorkerEntry, got)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "rev-list", "--objects", "--all"); got != reachableBefore {
		t.Fatalf("authoring created a reachable successor\nbefore:\n%s\nafter:\n%s", reachableBefore, got)
	}
	if after := snapshotAssociations(t, db); after != associationBefore {
		t.Fatalf("authoring changed associations\nbefore:\n%s\nafter:\n%s", associationBefore, after)
	}
	var tables string
	if err := db.QueryRow(`SELECT group_concat(name, ',') FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != "source_architecture_associations" {
		t.Fatalf("authoring persisted Architecture tables: %q", tables)
	}
	if after := snapshotRepository(t, source); after != sourceBefore {
		t.Fatalf("authoring changed source repository\nbefore:\n%s\nafter:\n%s", sourceBefore, after)
	}
}

func TestPendingFieldIntentPreservesUntouchedCanonicalSections(t *testing.T) {
	db := openWebTestDatabase(t)
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	manifest := []byte(runGit(t, dataDirectory, "--git-dir", storePath, "show", initialized.Revision+":architecture.yaml") + "\n")
	titleOnlyID := uuid.NewString()
	descriptionOnlyID := uuid.NewString()
	titleOnlySource := []byte("---\r\nid: \"" + titleOnlyID + "\"\r\n---\r\n# Original\r\n\r\nExact body  \r\nSecond exact\r\n")
	descriptionOnlySource := []byte("---\nid: \"" + descriptionOnlyID + "\"\n---\nFirst *line*\nsecond line\n===========\nOriginal body\n")
	accepted := advanceAcceptedToComponents(t, storePath, initialized.Revision, manifest, []testComponent{
		{path: "title-only.md", mode: "100644", source: titleOnlySource},
		{path: "description-only.md", mode: "100644", source: descriptionOnlySource},
	})
	opened := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	if opened.Revision != accepted || strings.Join(opened.ComponentTitles, "|") != "First line second line|Original" {
		t.Fatalf("opened titles = %+v", opened)
	}

	titleEdited := postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: titleOnlyID, Title: "Renamed", Description: "\nExact body  \nSecond exact\n", TitleChanged: true,
	})
	if body := decodeArchitectureResponse(t, titleEdited); titleEdited.Code != http.StatusOK || body.Changes == nil || !body.Changes.Valid {
		t.Fatalf("title-only edit status=%d body=%s", titleEdited.Code, titleEdited.Body.String())
	}
	titleCandidate := runGit(t, dataDirectory, "--git-dir", storePath, "show", state.pending.candidate.Tree()+":components/title-only.md")
	wantTitleCandidate := "---\r\nid: \"" + titleOnlyID + "\"\r\n---\r\n# Renamed\r\n\r\nExact body  \r\nSecond exact"
	if titleCandidate != wantTitleCandidate {
		t.Fatalf("title-only edit changed canonical body bytes\ngot:  %q\nwant: %q", titleCandidate, wantTitleCandidate)
	}

	descriptionEdited := postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: descriptionOnlyID, Title: "First line second line", Description: "Changed body\n", DescriptionChanged: true,
	})
	if body := decodeArchitectureResponse(t, descriptionEdited); descriptionEdited.Code != http.StatusOK || body.Changes == nil || !body.Changes.Valid {
		t.Fatalf("description-only edit status=%d body=%s", descriptionEdited.Code, descriptionEdited.Body.String())
	}
	descriptionCandidate := runGit(t, dataDirectory, "--git-dir", storePath, "show", state.pending.candidate.Tree()+":components/description-only.md")
	wantDescriptionCandidate := "---\nid: \"" + descriptionOnlyID + "\"\n---\nFirst *line*\nsecond line\n===========\nChanged body"
	if descriptionCandidate != wantDescriptionCandidate {
		t.Fatalf("description-only edit changed canonical H1 bytes\ngot:  %q\nwant: %q", descriptionCandidate, wantDescriptionCandidate)
	}
}

func TestConcurrentComponentMutationsAccumulateWithoutLostChanges(t *testing.T) {
	db := openWebTestDatabase(t)
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	handler := NewHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	manifest := []byte(runGit(t, dataDirectory, "--git-dir", storePath, "show", initialized.Revision+":architecture.yaml") + "\n")
	firstID := uuid.NewString()
	secondID := uuid.NewString()
	accepted := advanceAcceptedToComponents(t, storePath, initialized.Revision, manifest, []testComponent{
		{path: "first.md", mode: "100644", source: []byte("---\nid: \"" + firstID + "\"\n---\n# First\nFirst body\n")},
		{path: "second.md", mode: "100644", source: []byte("---\nid: \"" + secondID + "\"\n---\n# Second\nSecond body\n")},
	})
	if response := postOpenProject(t, handler, testOrigin, source); response.Code != http.StatusOK {
		t.Fatalf("open status=%d body=%s", response.Code, response.Body.String())
	}

	requests := []componentMutationRequest{
		{SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: firstID, Title: "First changed", TitleChanged: true},
		{SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: secondID, Title: "Second changed", TitleChanged: true},
	}
	responses := make([]*httptest.ResponseRecorder, len(requests))
	var wait sync.WaitGroup
	for index := range requests {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			responses[index] = postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", requests[index])
		}(index)
	}
	wait.Wait()
	for index, response := range responses {
		if response.Code != http.StatusOK {
			t.Fatalf("concurrent response %d status=%d body=%s", index, response.Code, response.Body.String())
		}
	}
	final := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	if final.Changes == nil || !final.Changes.Valid || len(final.Changes.Components) != 2 {
		t.Fatalf("concurrent changes were lost or partial: %+v", final.Changes)
	}
	titles := map[string]string{}
	for _, change := range final.Changes.Components {
		titles[change.ID] = change.Title
	}
	if titles[firstID] != "First changed" || titles[secondID] != "Second changed" {
		t.Fatalf("concurrent titles = %#v", titles)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != accepted {
		t.Fatalf("concurrent authoring changed accepted from %q to %q", accepted, got)
	}
}

func TestComponentMutationsEnforceOriginAndLoadedProject(t *testing.T) {
	db := openWebTestDatabase(t)
	source := createSourceRepository(t)
	handler := NewHandler(db, testOrigin, t.TempDir(), t.TempDir())
	base := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	payload := componentMutationRequest{SourceRoot: filepath.Clean(source), ExpectedRevision: base.Revision, Title: "Worker", Description: "Body"}

	wrongOrigin := postComponentMutation(t, handler, "http://127.0.0.1:9999", "/api/architecture/components/add", payload)
	if wrongOrigin.Code != http.StatusForbidden || wrongOrigin.Header().Get("Access-Control-Allow-Origin") != "" || !strings.Contains(wrongOrigin.Body.String(), errorOriginMismatch) {
		t.Fatalf("wrong-origin response status=%d headers=%v body=%s", wrongOrigin.Code, wrongOrigin.Header(), wrongOrigin.Body.String())
	}
	payload.SourceRoot = filepath.Join(filepath.Clean(source), "elsewhere")
	wrongProject := postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", payload)
	if wrongProject.Code != http.StatusConflict || !strings.Contains(wrongProject.Body.String(), errorArchitectureNotOpen) {
		t.Fatalf("wrong-project response status=%d body=%s", wrongProject.Code, wrongProject.Body.String())
	}

	payload.SourceRoot = filepath.Clean(source)
	created := postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", payload)
	if created.Code != http.StatusOK || decodeArchitectureResponse(t, created).Changes == nil {
		t.Fatalf("create pending response status=%d body=%s", created.Code, created.Body.String())
	}
	otherSource := createSourceRepository(t)
	payload.SourceRoot = filepath.Clean(otherSource)
	otherMutation := postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", payload)
	if otherMutation.Code != http.StatusConflict || !strings.Contains(otherMutation.Body.String(), errorArchitectureNotOpen) {
		t.Fatalf("other-store mutation status=%d body=%s", otherMutation.Code, otherMutation.Body.String())
	}
	reopened := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	if reopened.Changes == nil || len(reopened.Changes.Components) != 1 {
		t.Fatalf("returning to exact store/base lost pending changes: %+v", reopened.Changes)
	}
}

func postComponentMutation(t *testing.T, handler http.Handler, origin, path string, payload componentMutationRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	request.Header.Set("Origin", origin)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeArchitectureResponse(t *testing.T, response *httptest.ResponseRecorder) architectureResponse {
	t.Helper()
	var value architectureResponse
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode architecture response status=%d body=%s: %v", response.Code, response.Body.String(), err)
	}
	return value
}
