package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	refreshSourceID = "3c8320e7-9c64-46b3-b3e7-0e1204235cb0"
	refreshTargetID = "b1f5dbf5-7fa1-498f-aa9d-b8032570d668"
)

type refreshFixture struct {
	state              *Handler
	handler            http.Handler
	source             string
	sourceBefore       string
	dataDirectory      string
	storePath          string
	manifest           []byte
	loadedRevision     string
	associationsBefore string
}

func newRefreshFixture(t *testing.T) refreshFixture {
	t.Helper()
	source := createSourceRepository(t)
	sourceBefore := snapshotRepository(t, source)
	dataDirectory := t.TempDir()
	db := openWebDatabaseAt(t, filepath.Join(dataDirectory, "workbraid.db"))
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	manifest := []byte(runGit(t, dataDirectory, "--git-dir", storePath, "show", initialized.Revision+":architecture.yaml") + "\n")
	accepted := advanceAcceptedToComponents(t, storePath, initialized.Revision, manifest, refreshComponents("Source A", "Target A"))
	opened := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	if opened.Revision != accepted || opened.ComponentCount != 2 {
		t.Fatalf("refresh fixture did not load exact accepted revision: %+v", opened)
	}
	return refreshFixture{
		state: state, handler: handler, source: source, sourceBefore: sourceBefore,
		dataDirectory: dataDirectory, storePath: storePath, manifest: manifest,
		loadedRevision: accepted, associationsBefore: snapshotAssociations(t, db),
	}
}

func refreshComponents(sourceTitle, targetTitle string) []testComponent {
	return []testComponent{
		{path: "source.md", mode: "100644", source: []byte("---\nid: \"" + refreshSourceID + "\"\nrelationships:\n  - target: \"" + refreshTargetID + "\"\n    label: calls\n---\n# " + sourceTitle + "\nSource body.\n")},
		{path: "target.md", mode: "100644", source: []byte("---\nid: \"" + refreshTargetID + "\"\n---\n# " + targetTitle + "\nTarget body.\n")},
	}
}

func (fixture refreshFixture) advance(t *testing.T, parent, sourceTitle, targetTitle string) string {
	t.Helper()
	return advanceAcceptedToComponents(t, fixture.storePath, parent, fixture.manifest, refreshComponents(sourceTitle, targetTitle))
}

func (fixture refreshFixture) keepAndReview(t *testing.T) architectureResponse {
	t.Helper()
	kept := decodeArchitectureResponse(t, postComponentMutation(t, fixture.handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(fixture.source), ComponentID: refreshSourceID,
		Title: "Pending source", TitleChanged: true,
		Relationships: []relationshipResponse{{TargetID: refreshTargetID, Label: "pending calls"}}, RelationshipsChanged: true,
	}))
	if kept.Changes == nil {
		t.Fatal("pending change was not retained")
	}
	reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, fixture.handler, testOrigin, "/api/architecture/review", fixture.source))
	if reviewed.Changes == nil || reviewed.Changes.Review == nil {
		t.Fatalf("pending change was not reviewed: %+v", reviewed)
	}
	return reviewed
}

func (fixture refreshFixture) assertIsolation(t *testing.T) {
	t.Helper()
	if got := snapshotRepository(t, fixture.source); got != fixture.sourceBefore {
		t.Fatalf("Refresh changed source repository\nbefore:\n%s\nafter:\n%s", fixture.sourceBefore, got)
	}
	if got := snapshotAssociations(t, fixture.state.db); got != fixture.associationsBefore {
		t.Fatalf("Refresh changed logical SQLite state\nbefore:\n%s\nafter:\n%s", fixture.associationsBefore, got)
	}
}

func TestRefreshUnchangedIsQuietAndPreservesPendingReview(t *testing.T) {
	fixture := newRefreshFixture(t)
	reviewed := fixture.keepAndReview(t)
	review := *reviewed.Changes.Review
	gitBefore := snapshotPrivateArchitecture(t, fixture.dataDirectory)

	response := postArchitectureAction(t, fixture.handler, testOrigin, "/api/architecture/refresh", fixture.source)
	refreshed := decodeArchitectureResponse(t, response)
	if response.Code != http.StatusOK || refreshed.Revision != fixture.loadedRevision || refreshed.Stale || refreshed.ActionError != "" {
		t.Fatalf("unchanged Refresh status=%d result=%+v", response.Code, refreshed)
	}
	if refreshed.Changes == nil || refreshed.Changes.Stale || refreshed.Changes.Review == nil || !reflect.DeepEqual(*refreshed.Changes.Review, review) {
		t.Fatalf("unchanged Refresh displaced pending/review state: %+v", refreshed.Changes)
	}
	if got := snapshotPrivateArchitecture(t, fixture.dataDirectory); got != gitBefore {
		t.Fatalf("unchanged Refresh wrote private Git state\nbefore:\n%s\nafter:\n%s", gitBefore, got)
	}
	fixture.assertIsolation(t)
}

func TestRefreshAdoptsValidExternalStateAndPreservesOldPendingContextAsStale(t *testing.T) {
	fixture := newRefreshFixture(t)
	fixture.keepAndReview(t)
	pendingGeneration := fixture.state.pending.generation
	pendingCandidateTree := fixture.state.pending.candidate.Tree()
	external := fixture.advance(t, fixture.loadedRevision, "Source B", "Target B")

	before := fixture.state.currentArchitectureResponse()
	if before.Revision != fixture.loadedRevision || before.Components[1].Title != "Target A" {
		t.Fatalf("external state became visible before Refresh: %+v", before)
	}
	response := postArchitectureAction(t, fixture.handler, testOrigin, "/api/architecture/refresh", fixture.source)
	refreshed := decodeArchitectureResponse(t, response)
	if response.Code != http.StatusOK || refreshed.Revision != external || refreshed.Stale || refreshed.ComponentTitles[1] != "Target B" {
		t.Fatalf("valid external state was not atomically adopted: status=%d body=%s", response.Code, response.Body.String())
	}
	if refreshed.Changes == nil || !refreshed.Changes.Stale || refreshed.Changes.Review != nil || fixture.state.pending.review != nil {
		t.Fatalf("old pending work was not stale/read-only: %+v", refreshed.Changes)
	}
	if fixture.state.pending.generation != pendingGeneration || fixture.state.pending.candidate.Tree() != pendingCandidateTree ||
		refreshed.Changes.RelationshipTargets[1].Title != "Target A" {
		t.Fatalf("old pending context was reinterpreted: pending=%+v response=%+v", fixture.state.pending, refreshed.Changes)
	}

	discarded := decodeArchitectureResponse(t, postArchitectureAction(t, fixture.handler, testOrigin, "/api/architecture/discard", fixture.source))
	if discarded.Changes != nil || discarded.Revision != external || discarded.Stale {
		t.Fatalf("discard changed current accepted state: %+v", discarded)
	}
	newPending := decodeArchitectureResponse(t, postComponentMutation(t, fixture.handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(fixture.source), ComponentID: refreshSourceID,
		Description: "New-base work.", DescriptionChanged: true,
	}))
	if newPending.Changes == nil || fixture.state.pending.baseRevision != external || fixture.state.pending.stale {
		t.Fatalf("new pending work did not bind to current accepted state: %+v", fixture.state.pending)
	}
	fixture.assertIsolation(t)
}

func TestRefreshAdoptsAValidExternalRewindWithoutAncestryPolicy(t *testing.T) {
	fixture := newRefreshFixture(t)
	external := fixture.advance(t, fixture.loadedRevision, "Source B", "Target B")
	first := decodeArchitectureResponse(t, postArchitectureAction(t, fixture.handler, testOrigin, "/api/architecture/refresh", fixture.source))
	if first.Revision != external {
		t.Fatalf("initial advancement was not adopted: %+v", first)
	}
	runGit(t, fixture.dataDirectory, "--git-dir", fixture.storePath, "update-ref", "refs/heads/accepted", fixture.loadedRevision, external)
	rewound := decodeArchitectureResponse(t, postArchitectureAction(t, fixture.handler, testOrigin, "/api/architecture/refresh", fixture.source))
	if rewound.Revision != fixture.loadedRevision || rewound.Stale || rewound.ComponentTitles[0] != "Source A" {
		t.Fatalf("valid rewind was ancestry-gated or mixed: %+v", rewound)
	}
}

func TestFreshApplicationReconstructsTheFinalExternallyAcceptedRevision(t *testing.T) {
	fixture := newRefreshFixture(t)
	external := fixture.advance(t, fixture.loadedRevision, "Source final", "Target final")
	refreshed := decodeArchitectureResponse(t, postArchitectureAction(t, fixture.handler, testOrigin, "/api/architecture/refresh", fixture.source))
	if refreshed.Revision != external || refreshed.Components[0].Relationships[0].TargetID != refreshTargetID {
		t.Fatalf("final external state did not load before restart: %+v", refreshed)
	}

	freshDB := openWebDatabaseAt(t, filepath.Join(fixture.dataDirectory, "workbraid.db"))
	_, freshHandler := newHandler(freshDB, testOrigin, t.TempDir(), fixture.dataDirectory)
	reopened := decodeArchitectureResponse(t, postOpenProject(t, freshHandler, testOrigin, fixture.source))
	if reopened.Revision != external || strings.Join(reopened.ComponentTitles, "|") != "Source final|Target final" ||
		len(reopened.Components[0].Relationships) != 1 || reopened.Components[0].Relationships[0].TargetID != refreshTargetID {
		t.Fatalf("fresh application did not reconstruct exact final accepted state: %+v", reopened)
	}
	fixture.assertIsolation(t)
}

func TestRefreshConclusiveFailuresMakePendingAndRetainedViewReadOnly(t *testing.T) {
	tests := []struct {
		name       string
		wantStatus int
		wantError  string
		arrange    func(t *testing.T, fixture refreshFixture)
	}{
		{
			name: "invalid", wantStatus: http.StatusConflict, wantError: errorRefreshInvalid,
			arrange: func(t *testing.T, fixture refreshFixture) {
				invalid := []byte("format: workbraid-architecture\nversion: 1\nstore_id: not-a-uuid\nproject:\n  name: Project\n  source_hint: /tmp/project\n")
				advanceAcceptedToManifest(t, fixture.storePath, fixture.loadedRevision, invalid, nil)
			},
		},
		{
			name: "unsupported", wantStatus: http.StatusUnprocessableEntity, wantError: errorRefreshUnsupported,
			arrange: func(t *testing.T, fixture refreshFixture) {
				unsupported := []byte(strings.Replace(string(fixture.manifest), "version: 1", "version: 3", 1))
				advanceAcceptedToManifest(t, fixture.storePath, fixture.loadedRevision, unsupported, nil)
			},
		},
		{
			name: "missing", wantStatus: http.StatusConflict, wantError: errorRefreshUnavailable,
			arrange: func(t *testing.T, fixture refreshFixture) {
				runGit(t, fixture.dataDirectory, "--git-dir", fixture.storePath, "update-ref", "-d", "refs/heads/accepted", fixture.loadedRevision)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newRefreshFixture(t)
			fixture.keepAndReview(t)
			generationBefore := fixture.state.pending.generation
			candidateBefore := fixture.state.pending.candidate.Tree()
			test.arrange(t, fixture)
			response := postArchitectureAction(t, fixture.handler, testOrigin, "/api/architecture/refresh", fixture.source)
			result := decodeArchitectureResponse(t, response)
			if response.Code != test.wantStatus || result.ActionError != test.wantError || !result.Stale || result.Revision != fixture.loadedRevision {
				t.Fatalf("conclusive Refresh status=%d result=%+v", response.Code, result)
			}
			if result.Changes == nil || !result.Changes.Stale || result.Changes.Review != nil || fixture.state.pending.review != nil {
				t.Fatalf("conclusive Refresh retained writable/reviewed pending state: %+v", result.Changes)
			}
			if fixture.state.pending.generation != generationBefore || fixture.state.pending.candidate.Tree() != candidateBefore || result.Changes.RelationshipTargets[1].Title != "Target A" {
				t.Fatalf("conclusive Refresh changed old-base pending context: pending=%+v changes=%+v", fixture.state.pending, result.Changes)
			}
			current, present, err := fixture.state.architecture.AcceptedRevision(context.Background(), *fixture.state.loadedSnapshot)
			if err != nil {
				t.Fatal(err)
			}
			arguments := []string{"--git-dir", fixture.storePath, "update-ref", "refs/heads/accepted", fixture.loadedRevision}
			if present {
				arguments = append(arguments, current)
			} else {
				arguments = append(arguments, "0000000000000000000000000000000000000000")
			}
			runGit(t, fixture.dataDirectory, arguments...)
			recoveredResponse := postArchitectureAction(t, fixture.handler, testOrigin, "/api/architecture/refresh", fixture.source)
			recovered := decodeArchitectureResponse(t, recoveredResponse)
			if recoveredResponse.Code != http.StatusOK || recovered.Stale || recovered.Revision != fixture.loadedRevision || recovered.Changes == nil || !recovered.Changes.Stale {
				t.Fatalf("corrected accepted did not recover current view while retaining stale work: status=%d result=%+v", recoveredResponse.Code, recovered)
			}
			fixture.assertIsolation(t)
		})
	}
}

func TestRefreshIndeterminateObservationDoesNotInventStaleness(t *testing.T) {
	fixture := newRefreshFixture(t)
	reviewed := fixture.keepAndReview(t)
	away := fixture.storePath + ".away"
	if err := os.Rename(fixture.storePath, away); err != nil {
		t.Fatal(err)
	}
	response := postArchitectureAction(t, fixture.handler, testOrigin, "/api/architecture/refresh", fixture.source)
	if err := os.Rename(away, fixture.storePath); err != nil {
		t.Fatal(err)
	}
	result := decodeArchitectureResponse(t, response)
	if response.Code != http.StatusServiceUnavailable || result.ActionError != errorRefreshFailed || result.Stale || result.Changes == nil || result.Changes.Stale {
		t.Fatalf("indeterminate Refresh invented stale state: status=%d result=%+v", response.Code, result)
	}
	if result.Changes.Review == nil || !reflect.DeepEqual(*result.Changes.Review, *reviewed.Changes.Review) {
		t.Fatalf("indeterminate Refresh invalidated a current review: %+v", result.Changes)
	}

	runGit(t, fixture.dataDirectory, "--git-dir", fixture.storePath, "update-ref", "-d", "refs/heads/accepted", fixture.loadedRevision)
	known := decodeArchitectureResponse(t, postArchitectureAction(t, fixture.handler, testOrigin, "/api/architecture/refresh", fixture.source))
	if !known.Stale || known.Changes == nil || !known.Changes.Stale {
		t.Fatalf("missing accepted did not establish known non-current state: %+v", known)
	}
	if err := os.Rename(fixture.storePath, away); err != nil {
		t.Fatal(err)
	}
	repeated := decodeArchitectureResponse(t, postArchitectureAction(t, fixture.handler, testOrigin, "/api/architecture/refresh", fixture.source))
	if err := os.Rename(away, fixture.storePath); err != nil {
		t.Fatal(err)
	}
	if !repeated.Stale || repeated.Changes == nil || !repeated.Changes.Stale || repeated.ActionError != errorRefreshFailed {
		t.Fatalf("indeterminate retry erased known state: %+v", repeated)
	}
}

func TestRefreshFinalObservationClassifiesRealRefRaces(t *testing.T) {
	tests := []struct {
		name        string
		arrangeRace func(t *testing.T, fixture refreshFixture, observed, third string)
		wantStatus  int
		wantError   string
		wantCurrent bool
		wantStale   bool
	}{
		{
			name: "candidate remains authoritative", wantStatus: http.StatusOK, wantCurrent: false,
			arrangeRace: func(_ *testing.T, _ refreshFixture, _, _ string) {},
		},
		{
			name: "authority returns to retained revision", wantStatus: http.StatusOK, wantCurrent: true,
			arrangeRace: func(t *testing.T, fixture refreshFixture, observed, _ string) {
				runGit(t, fixture.dataDirectory, "--git-dir", fixture.storePath, "update-ref", "refs/heads/accepted", fixture.loadedRevision, observed)
			},
		},
		{
			name: "third revision wins", wantStatus: http.StatusConflict, wantError: errorRefreshChanged, wantCurrent: true, wantStale: true,
			arrangeRace: func(t *testing.T, fixture refreshFixture, observed, third string) {
				runGit(t, fixture.dataDirectory, "--git-dir", fixture.storePath, "update-ref", "refs/heads/accepted", third, observed)
			},
		},
		{
			name: "accepted disappears", wantStatus: http.StatusConflict, wantError: errorRefreshUnavailable, wantCurrent: true, wantStale: true,
			arrangeRace: func(t *testing.T, fixture refreshFixture, observed, _ string) {
				runGit(t, fixture.dataDirectory, "--git-dir", fixture.storePath, "update-ref", "-d", "refs/heads/accepted", observed)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newRefreshFixture(t)
			fixture.keepAndReview(t)
			observed := fixture.advance(t, fixture.loadedRevision, "Source R", "Target R")
			third := fixture.advance(t, observed, "Source S", "Target S")
			runGit(t, fixture.dataDirectory, "--git-dir", fixture.storePath, "update-ref", "refs/heads/accepted", observed, third)
			fixture.state.beforeRefreshReobserve = func(revision string) { test.arrangeRace(t, fixture, revision, third) }
			response := postArchitectureAction(t, fixture.handler, testOrigin, "/api/architecture/refresh", fixture.source)
			result := decodeArchitectureResponse(t, response)
			wantRevision := observed
			if test.wantCurrent {
				wantRevision = fixture.loadedRevision
			}
			if response.Code != test.wantStatus || result.ActionError != test.wantError || result.Revision != wantRevision || result.Stale != test.wantStale {
				t.Fatalf("ref race status=%d result=%+v want revision=%s stale=%t error=%s", response.Code, result, wantRevision, test.wantStale, test.wantError)
			}
			if test.wantStale && (result.Changes == nil || !result.Changes.Stale || result.Changes.Review != nil) {
				t.Fatalf("conclusive ref race did not stale pending context: %+v", result.Changes)
			}
			if test.name == "authority returns to retained revision" && (result.Changes == nil || result.Changes.Stale || result.Changes.Review == nil) {
				t.Fatalf("return-to-retained race disturbed current pending/review: %+v", result.Changes)
			}
		})
	}
}

func TestRefreshIndeterminateFinalObservationDoesNotPublishOrInventStaleness(t *testing.T) {
	fixture := newRefreshFixture(t)
	reviewed := fixture.keepAndReview(t)
	fixture.advance(t, fixture.loadedRevision, "Source R", "Target R")
	away := fixture.storePath + ".away"
	fixture.state.beforeRefreshReobserve = func(string) {
		if err := os.Rename(fixture.storePath, away); err != nil {
			t.Fatal(err)
		}
	}
	response := postArchitectureAction(t, fixture.handler, testOrigin, "/api/architecture/refresh", fixture.source)
	if err := os.Rename(away, fixture.storePath); err != nil {
		t.Fatal(err)
	}
	result := decodeArchitectureResponse(t, response)
	if response.Code != http.StatusServiceUnavailable || result.ActionError != errorRefreshFailed || result.Revision != fixture.loadedRevision || result.Stale {
		t.Fatalf("indeterminate final observation published or invented stale state: status=%d result=%+v", response.Code, result)
	}
	if result.Changes == nil || result.Changes.Stale || result.Changes.Review == nil || !reflect.DeepEqual(*result.Changes.Review, *reviewed.Changes.Review) {
		t.Fatalf("indeterminate final observation disturbed current pending/review: %+v", result.Changes)
	}
}

func TestRefreshRequiresExpectedOriginAndLoadedProject(t *testing.T) {
	fixture := newRefreshFixture(t)
	missingOrigin := postArchitectureAction(t, fixture.handler, "", "/api/architecture/refresh", fixture.source)
	if missingOrigin.Code != http.StatusForbidden || !strings.Contains(missingOrigin.Body.String(), errorOriginMismatch) {
		t.Fatalf("missing-origin Refresh status=%d body=%s", missingOrigin.Code, missingOrigin.Body.String())
	}
	wrongProject := postArchitectureAction(t, fixture.handler, testOrigin, "/api/architecture/refresh", filepath.Dir(fixture.source))
	if wrongProject.Code != http.StatusConflict || !strings.Contains(wrongProject.Body.String(), errorArchitectureNotOpen) {
		t.Fatalf("wrong-project Refresh status=%d body=%s", wrongProject.Code, wrongProject.Body.String())
	}
}

func TestRefreshSerializesWithPendingMutation(t *testing.T) {
	fixture := newRefreshFixture(t)
	external := fixture.advance(t, fixture.loadedRevision, "Source B", "Target B")
	loaded := make(chan struct{})
	release := make(chan struct{})
	fixture.state.beforeRefreshReobserve = func(string) {
		close(loaded)
		<-release
	}
	refreshDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		refreshDone <- postArchitectureAction(t, fixture.handler, testOrigin, "/api/architecture/refresh", fixture.source)
	}()
	<-loaded
	mutationDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		mutationDone <- postComponentMutation(t, fixture.handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
			SourceRoot: filepath.Clean(fixture.source), ComponentID: refreshSourceID,
			Description: "Serialized work.", DescriptionChanged: true,
		})
	}()
	select {
	case <-mutationDone:
		t.Fatal("pending mutation interleaved while Refresh owned application state")
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	refreshed := decodeArchitectureResponse(t, <-refreshDone)
	mutated := decodeArchitectureResponse(t, <-mutationDone)
	if refreshed.Revision != external || mutated.Revision != external || mutated.Changes == nil || fixture.state.pending.baseRevision != external {
		t.Fatalf("serialized transitions mixed revisions: refreshed=%+v mutated=%+v pending=%+v", refreshed, mutated, fixture.state.pending)
	}
}
