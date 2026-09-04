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

	"workbraid/internal/architecture"
)

type nativeRefreshFixture struct {
	state     *Handler
	handler   http.Handler
	base      architectureResponse
	component string
	storePath string
}

func newNativeRefreshFixture(t *testing.T, reviewed bool) nativeRefreshFixture {
	t.Helper()
	data := t.TempDir()
	state, handler := newHandler(testOrigin, testUI(t), data)
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Refresh fixture"}))
	base := *state.loadedSnapshot
	component := state.architecture.NewComponentChange(base, nil, "Gateway", "Accepted body.\n")
	candidate, err := state.architecture.ConstructCandidate(context.Background(), base, []architecture.ComponentChange{component}, architecture.CandidateComposition{
		NewComponentHomes: []architecture.NewComponentHome{{ComponentID: component.ID, DiagramID: base.RootDiagramID()}},
	})
	if err != nil {
		t.Fatal(err)
	}
	acceptArchitectureCandidate(t, state.architecture, base, candidate)
	opened := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/open", map[string]any{"project_slug": created.ProjectSlug}))
	fixture := nativeRefreshFixture{
		state: state, handler: handler, base: opened, component: component.ID,
		storePath: filepath.Join(data, "architecture", created.StoreID+".git"),
	}
	if reviewed {
		fixture.keepAndReview(t)
	}
	return fixture
}

func (fixture nativeRefreshFixture) action() architectureActionRequest {
	fixture.state.stateMutex.Lock()
	response := fixture.state.currentArchitectureResponseLocked()
	fixture.state.stateMutex.Unlock()
	return observedAction(response)
}

func (fixture nativeRefreshFixture) keepAndReview(t *testing.T) architectureResponse {
	t.Helper()
	kept := decodeArchitectureResponse(t, postJSONRequest(t, fixture.handler, "/api/architecture/components/edit", observedComponentMutation(fixture.base, componentMutationRequest{
		ComponentID: fixture.component, Description: "Pending body.\n", DescriptionChanged: true,
	})))
	if kept.Changes == nil {
		t.Fatal("pending change missing")
	}
	reviewed := decodeArchitectureResponse(t, postJSONRequest(t, fixture.handler, "/api/architecture/review", observedAction(kept)))
	if reviewed.Changes == nil || reviewed.Changes.Review == nil {
		t.Fatalf("review missing: %+v", reviewed.Changes)
	}
	return reviewed
}

func (fixture nativeRefreshFixture) advanceTitle(t *testing.T, base architecture.Snapshot, title string) string {
	t.Helper()
	candidate, err := fixture.state.architecture.ConstructCandidate(context.Background(), base, nil, architecture.CandidateComposition{
		DiagramTitles: []architecture.DiagramTitleChange{{DiagramID: base.RootDiagramID(), Title: title}},
	})
	if err != nil {
		t.Fatal(err)
	}
	revision, err := fixture.state.architecture.CreateSuccessor(context.Background(), base, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.state.architecture.AdvanceAccepted(context.Background(), base, revision); err != nil {
		t.Fatal(err)
	}
	return revision
}

func replaceAcceptedManifest(t *testing.T, storePath, parent string, transform func(string) string) string {
	t.Helper()
	manifest := transform(git(t, "--git-dir", storePath, "show", parent+":architecture.yaml"))
	blob := gitInput(t, []byte(manifest+"\n"), "--git-dir", storePath, "hash-object", "-w", "--stdin")
	entries := strings.Split(git(t, "--git-dir", storePath, "ls-tree", parent), "\n")
	for index, entry := range entries {
		if strings.HasSuffix(entry, "\tarchitecture.yaml") {
			entries[index] = "100644 blob " + blob + "\tarchitecture.yaml"
		}
	}
	tree := gitInput(t, []byte(strings.Join(entries, "\n")+"\n"), "--git-dir", storePath, "mktree")
	commit := gitInput(t, []byte("external manifest\n"), "-c", "user.name=Test", "-c", "user.email=test@workbraid.invalid", "--git-dir", storePath, "commit-tree", tree, "-p", parent)
	git(t, "--git-dir", storePath, "update-ref", "refs/heads/accepted", commit, parent)
	return commit
}

func TestEveryLoadedProjectActionRequiresStoreIdentityAsWellAsSlug(t *testing.T) {
	state, handler := newHandler(testOrigin, testUI(t), t.TempDir())
	first := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "First"}))
	second := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Second"}))
	decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/architecture/components/add", observedComponentMutation(second, componentMutationRequest{DiagramID: second.RootDiagramID, Title: "Kept"})))
	stale := architectureActionRequest{ProjectSlug: second.ProjectSlug, StoreID: first.StoreID}
	requests := []struct {
		path string
		body any
	}{
		{"/api/architecture/refresh", stale},
		{"/api/architecture/discard", stale},
		{"/api/projects/leave", stale},
		{"/api/architecture/review", stale},
		{"/api/architecture/accept", acceptChangesRequest{ProjectSlug: second.ProjectSlug, StoreID: first.StoreID}},
		{"/api/architecture/components/add", componentMutationRequest{ProjectSlug: second.ProjectSlug, StoreID: first.StoreID, ExpectedRevision: second.Revision, DiagramID: second.RootDiagramID, Title: "Late"}},
		{"/api/architecture/diagrams/title", diagramMutationRequest{ProjectSlug: second.ProjectSlug, StoreID: first.StoreID, ExpectedRevision: second.Revision, DiagramID: second.RootDiagramID, Title: "Late"}},
	}
	for _, request := range requests {
		response := postJSONRequest(t, handler, request.path, request.body)
		if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), errorArchitectureNotOpen) {
			t.Fatalf("%s accepted stale store binding: status=%d body=%s", request.path, response.Code, response.Body.String())
		}
	}
	state.stateMutex.Lock()
	defer state.stateMutex.Unlock()
	active := testActiveChangeSet(state)
	if state.loadedProject.storeID != second.StoreID || active == nil || len(active.changes) != 1 {
		t.Fatalf("stale action changed current state: project=%+v active=%+v", state.loadedProject, active)
	}
}

func TestRefreshSlugConflictDoesNotPublishAmbiguousLocator(t *testing.T) {
	data := t.TempDir()
	state, handler := newHandler(testOrigin, testUI(t), data)
	first := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Alpha"}))
	second := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Beta"}))
	decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/open", map[string]any{"project_slug": first.ProjectSlug}))
	storePath := filepath.Join(data, "architecture", first.StoreID+".git")
	replaceAcceptedManifest(t, storePath, first.Revision, func(value string) string {
		return strings.Replace(value, "slug: alpha", "slug: "+second.ProjectSlug, 1)
	})
	response := postJSONRequest(t, handler, "/api/architecture/refresh", observedAction(first))
	result := decodeArchitectureBody(t, response)
	if response.Code != http.StatusConflict || result.ActionError != errorCatalogConflict || !result.Stale || result.ProjectSlug != first.ProjectSlug || result.Revision != first.Revision {
		t.Fatalf("conflicting Refresh status=%d result=%+v", response.Code, result)
	}
	state.stateMutex.Lock()
	defer state.stateMutex.Unlock()
	if state.loadedProject.projectSlug != first.ProjectSlug || state.loadedSnapshot.Revision() != first.Revision {
		t.Fatalf("conflicting locator was published: project=%+v snapshot=%s", state.loadedProject, state.loadedSnapshot.Revision())
	}
}

func TestRefreshCandidateSlugConflictDoesNotOverrideReturnToRetainedRevision(t *testing.T) {
	fixture := newNativeRefreshFixture(t, false)
	other, err := fixture.state.architecture.CreateProject(context.Background(), "Moved locator")
	if err != nil {
		t.Fatal(err)
	}
	if other.ProjectSlug() != "moved-locator" {
		t.Fatalf("other slug=%q want moved-locator", other.ProjectSlug())
	}
	changedSlugRevision := replaceAcceptedManifest(t, fixture.storePath, fixture.base.Revision, func(value string) string {
		return strings.Replace(value, "slug: refresh-fixture", "slug: moved-locator", 1)
	})
	fixture.state.beforeRefreshReobserve = func(string) {
		git(t, "--git-dir", fixture.storePath, "update-ref", "refs/heads/accepted", fixture.base.Revision, changedSlugRevision)
	}
	response := postJSONRequest(t, fixture.handler, "/api/architecture/refresh", fixture.action())
	result := decodeArchitectureBody(t, response)
	if response.Code != http.StatusOK || result.ActionError != "" || result.Stale || result.ProjectSlug != fixture.base.ProjectSlug || result.Revision != fixture.base.Revision {
		t.Fatalf("candidate locator conflict overrode retained authority: status=%d result=%+v", response.Code, result)
	}
	if fixture.state.loadedProject.projectSlug != fixture.base.ProjectSlug || fixture.state.loadedSnapshot.Revision() != fixture.base.Revision {
		t.Fatalf("retained authority was not kept current: project=%+v snapshot=%s", fixture.state.loadedProject, fixture.state.loadedSnapshot.Revision())
	}
}

func TestRefreshConflictScanPrecedesMandatoryFinalAcceptedObservation(t *testing.T) {
	fixture := newNativeRefreshFixture(t, false)
	observed := replaceAcceptedManifest(t, fixture.storePath, fixture.base.Revision, func(value string) string {
		return strings.Replace(value, "slug: refresh-fixture", "slug: observed-locator", 1)
	})
	base := *fixture.state.loadedSnapshot
	observedSnapshot, err := fixture.state.architecture.LoadRevision(context.Background(), base, observed)
	if err != nil {
		t.Fatal(err)
	}
	thirdCandidate, err := fixture.state.architecture.ConstructCandidate(context.Background(), observedSnapshot, nil, architecture.CandidateComposition{
		DiagramTitles: []architecture.DiagramTitleChange{{DiagramID: observedSnapshot.RootDiagramID(), Title: "Third revision"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	third, err := fixture.state.architecture.CreateSuccessor(context.Background(), observedSnapshot, thirdCandidate)
	if err != nil {
		t.Fatal(err)
	}
	fixture.state.beforeRefreshCatalogCheck = func() {
		git(t, "--git-dir", fixture.storePath, "update-ref", "refs/heads/accepted", third, observed)
	}
	response := postJSONRequest(t, fixture.handler, "/api/architecture/refresh", fixture.action())
	result := decodeArchitectureBody(t, response)
	if response.Code != http.StatusConflict || result.ActionError != errorRefreshChanged || result.Revision != fixture.base.Revision || !result.Stale || result.ProjectSlug != fixture.base.ProjectSlug {
		t.Fatalf("conflict-scan race status=%d result=%+v", response.Code, result)
	}
	if accepted := git(t, "--git-dir", fixture.storePath, "rev-parse", "refs/heads/accepted"); accepted != third {
		t.Fatalf("accepted=%s want third=%s", accepted, third)
	}
}

func TestSlugChangingReopenKeepsCurrentRouteAndOldPendingUntilDiscard(t *testing.T) {
	fixture := newNativeRefreshFixture(t, false)
	decodeArchitectureResponse(t, postJSONRequest(t, fixture.handler, "/api/architecture/components/edit", observedComponentMutation(fixture.base, componentMutationRequest{
		ComponentID: fixture.component, Description: "Old-base pending.\n", DescriptionChanged: true,
	})))
	newRevision := replaceAcceptedManifest(t, fixture.storePath, fixture.base.Revision, func(value string) string {
		return strings.Replace(value, "slug: refresh-fixture", "slug: moved-project", 1)
	})
	reopened := decodeArchitectureResponse(t, postJSONRequest(t, fixture.handler, "/api/projects/open", map[string]any{"project_slug": "moved-project"}))
	if reopened.ProjectSlug != "moved-project" || reopened.Revision != newRevision || reopened.Stale || reopened.Changes != nil || len(reopened.ChangeSets) != 1 || !reopened.ChangeSets[0].OutOfDate {
		t.Fatalf("open did not adopt the current route and preserve proposal context: %+v", reopened)
	}
	selectActiveChangeSetForTest(&reopened, reopened.ChangeSets[0].ID)
	discarded := decodeArchitectureResponse(t, postJSONRequest(t, fixture.handler, "/api/architecture/discard", observedAction(reopened)))
	if discarded.ProjectSlug != "moved-project" || discarded.Revision != newRevision || discarded.Stale || discarded.Changes != nil {
		t.Fatalf("discard did not recover validated current snapshot: %+v", discarded)
	}
}

func TestSameHomeMoveIsAnExactNoOp(t *testing.T) {
	fixture := newNativeRefreshFixture(t, false)
	before := git(t, "--git-dir", fixture.storePath, "ls-tree", "-r", fixture.base.Revision)
	response := decodeArchitectureResponse(t, postJSONRequest(t, fixture.handler, "/api/architecture/components/move-home", observedDiagramMutation(fixture.base, diagramMutationRequest{ComponentID: fixture.component, DiagramID: fixture.base.RootDiagramID})))
	if response.Changes != nil {
		t.Fatalf("same-home move created pending work: %+v", response.Changes)
	}
	if got := git(t, "--git-dir", fixture.storePath, "ls-tree", "-r", fixture.base.Revision); got != before {
		t.Fatalf("same-home move changed canonical tree\nbefore=%s\nafter=%s", before, got)
	}
	found := false
	for _, choice := range response.HomeMoveDestinations {
		if choice.ComponentID == fixture.component {
			found = true
			if choice.CurrentHomeID != fixture.base.RootDiagramID {
				t.Fatalf("current home=%q want %q", choice.CurrentHomeID, fixture.base.RootDiagramID)
			}
			for _, diagramID := range choice.DiagramIDs {
				if diagramID == fixture.base.RootDiagramID {
					t.Fatalf("current home remained a destination: %+v", choice)
				}
			}
		}
	}
	if !found {
		t.Fatal("component move options missing")
	}
}

func TestRefreshUnchangedPreservesExactReviewBinding(t *testing.T) {
	fixture := newNativeRefreshFixture(t, true)
	active := testActiveChangeSet(fixture.state)
	before := *active.review
	result := decodeArchitectureResponse(t, postJSONRequest(t, fixture.handler, "/api/architecture/refresh", fixture.action()))
	selectActiveChangeSetForTest(&result, active.id)
	expected := reviewResponseForBinding(active.id, before, active.baseSnapshot)
	expected.ReviewedState = active.refObject
	if result.ActionError != "" || result.Stale || result.Changes == nil || result.Changes.Stale || result.Changes.Review == nil ||
		!reflect.DeepEqual(*result.Changes.Review, expected) {
		t.Fatalf("unchanged Refresh displaced review presentation: result=%+v", result.Changes)
	}
	current := testActiveChangeSet(fixture.state)
	if current.review == nil || !reflect.DeepEqual(*current.review, before) {
		t.Fatalf("unchanged Refresh displaced binding: review=%+v", current.review)
	}
}

func reviewResponseForBinding(changeSetID string, binding reviewBinding, base architecture.Snapshot) reviewResponse {
	before, withChanges, comparison := captureReviewPresentation(base, binding.candidate.Snapshot())
	return reviewResponse{
		ChangeSetID: changeSetID, Diff: binding.diff, BaseRevision: binding.baseRevision, CandidateTree: binding.candidateTree, Generation: binding.generation,
		Before: before, WithChanges: withChanges, Comparison: comparison,
	}
}

func TestRefreshConclusiveAndIndeterminateFailuresRemainDistinct(t *testing.T) {
	tests := []struct {
		name       string
		wantStatus int
		wantError  string
		arrange    func(*testing.T, nativeRefreshFixture)
		restore    func(*testing.T, nativeRefreshFixture)
		stale      bool
	}{
		{name: "invalid", wantStatus: http.StatusConflict, wantError: errorRefreshInvalid, stale: true, arrange: func(t *testing.T, f nativeRefreshFixture) {
			replaceAcceptedManifest(t, f.storePath, f.base.Revision, func(value string) string { return strings.Replace(value, "slug: refresh-fixture", "slug: Invalid", 1) })
		}},
		{name: "unsupported", wantStatus: http.StatusUnprocessableEntity, wantError: errorRefreshUnsupported, stale: true, arrange: func(t *testing.T, f nativeRefreshFixture) {
			replaceAcceptedManifest(t, f.storePath, f.base.Revision, func(value string) string { return strings.Replace(value, "version: 2", "version: 3", 1) })
		}},
		{name: "missing", wantStatus: http.StatusConflict, wantError: errorRefreshUnavailable, stale: true, arrange: func(t *testing.T, f nativeRefreshFixture) {
			git(t, "--git-dir", f.storePath, "update-ref", "-d", "refs/heads/accepted", f.base.Revision)
		}},
		{name: "indeterminate", wantStatus: http.StatusServiceUnavailable, wantError: errorRefreshFailed, arrange: func(t *testing.T, f nativeRefreshFixture) {
			if err := os.Rename(f.storePath, f.storePath+".away"); err != nil {
				t.Fatal(err)
			}
		}, restore: func(t *testing.T, f nativeRefreshFixture) {
			if err := os.Rename(f.storePath+".away", f.storePath); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newNativeRefreshFixture(t, true)
			active := testActiveChangeSet(fixture.state)
			binding := *active.review
			test.arrange(t, fixture)
			response := postJSONRequest(t, fixture.handler, "/api/architecture/refresh", fixture.action())
			if test.restore != nil {
				test.restore(t, fixture)
			}
			result := decodeArchitectureBody(t, response)
			selectActiveChangeSetForTest(&result, active.id)
			if response.Code != test.wantStatus || result.ActionError != test.wantError || result.Stale != test.stale || result.Changes == nil || result.Changes.Stale != test.stale {
				t.Fatalf("status=%d result=%+v", response.Code, result)
			}
			current := testActiveChangeSet(fixture.state)
			if current.review == nil || !reflect.DeepEqual(*current.review, binding) || result.Changes.Review == nil {
				t.Fatal("Refresh failure changed the durable review binding")
			}
		})
	}
}

func TestRefreshFinalObservationRaceClassifications(t *testing.T) {
	for _, scenario := range []string{"retained", "third", "missing", "indeterminate"} {
		t.Run(scenario, func(t *testing.T) {
			fixture := newNativeRefreshFixture(t, true)
			active := testActiveChangeSet(fixture.state)
			baseSnapshot := *fixture.state.loadedSnapshot
			observed := fixture.advanceTitle(t, baseSnapshot, "Observed")
			observedSnapshot, err := fixture.state.architecture.LoadRevision(context.Background(), baseSnapshot, observed)
			if err != nil {
				t.Fatal(err)
			}
			third := fixture.advanceTitle(t, observedSnapshot, "Third")
			git(t, "--git-dir", fixture.storePath, "update-ref", "refs/heads/accepted", observed, third)
			away := fixture.storePath + ".away"
			fixture.state.beforeRefreshReobserve = func(string) {
				switch scenario {
				case "retained":
					git(t, "--git-dir", fixture.storePath, "update-ref", "refs/heads/accepted", fixture.base.Revision, observed)
				case "third":
					git(t, "--git-dir", fixture.storePath, "update-ref", "refs/heads/accepted", third, observed)
				case "missing":
					git(t, "--git-dir", fixture.storePath, "update-ref", "-d", "refs/heads/accepted", observed)
				case "indeterminate":
					if err := os.Rename(fixture.storePath, away); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(fixture.storePath, []byte("not a repository"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			}
			response := postJSONRequest(t, fixture.handler, "/api/architecture/refresh", fixture.action())
			if scenario == "indeterminate" {
				if err := os.Remove(fixture.storePath); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(away, fixture.storePath); err != nil {
					t.Fatal(err)
				}
			}
			result := decodeArchitectureBody(t, response)
			selectActiveChangeSetForTest(&result, active.id)
			switch scenario {
			case "retained":
				if response.Code != http.StatusOK || result.Revision != fixture.base.Revision || result.Stale || result.Changes == nil || result.Changes.Review == nil {
					t.Fatalf("retained=%+v", result)
				}
			case "third":
				if response.Code != http.StatusConflict || result.ActionError != errorRefreshChanged || !result.Stale || result.Changes == nil || !result.Changes.Stale {
					t.Fatalf("third=%+v", result)
				}
			case "missing":
				if response.Code != http.StatusConflict || result.ActionError != errorRefreshUnavailable || !result.Stale {
					t.Fatalf("missing=%+v", result)
				}
			case "indeterminate":
				if response.Code != http.StatusServiceUnavailable || result.ActionError != errorRefreshFailed || result.Stale || result.Changes == nil || result.Changes.Stale || result.Changes.Review == nil {
					t.Fatalf("indeterminate=%+v", result)
				}
			}
		})
	}
}

func TestRefreshAdoptsNonLinearRevisionAndSerializesMutation(t *testing.T) {
	fixture := newNativeRefreshFixture(t, false)
	baseSnapshot := *fixture.state.loadedSnapshot
	advanced := fixture.advanceTitle(t, baseSnapshot, "Advanced")
	first := decodeArchitectureResponse(t, postJSONRequest(t, fixture.handler, "/api/architecture/refresh", fixture.action()))
	if first.Revision != advanced {
		t.Fatalf("advance=%+v", first)
	}
	git(t, "--git-dir", fixture.storePath, "update-ref", "refs/heads/accepted", fixture.base.Revision, advanced)
	rewound := decodeArchitectureResponse(t, postJSONRequest(t, fixture.handler, "/api/architecture/refresh", observedAction(first)))
	if rewound.Revision != fixture.base.Revision || rewound.Stale {
		t.Fatalf("rewind=%+v", rewound)
	}

	secondSnapshot := *fixture.state.loadedSnapshot
	external := fixture.advanceTitle(t, secondSnapshot, "Serialized")
	loaded := make(chan struct{})
	release := make(chan struct{})
	fixture.state.beforeRefreshReobserve = func(string) { close(loaded); <-release }
	refreshDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		refreshDone <- postJSONRequest(t, fixture.handler, "/api/architecture/refresh", observedAction(rewound))
	}()
	<-loaded
	mutationDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		mutationDone <- postJSONRequest(t, fixture.handler, "/api/architecture/components/edit", observedComponentMutation(rewound, componentMutationRequest{ComponentID: fixture.component,
			Description: "Late.\n", DescriptionChanged: true,
		}))
	}()
	select {
	case <-mutationDone:
		t.Fatal("mutation interleaved with Refresh")
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	refreshed := decodeArchitectureResponse(t, <-refreshDone)
	mutation := <-mutationDone
	if refreshed.Revision != external || mutation.Code != http.StatusConflict || !strings.Contains(mutation.Body.String(), errorChangesElsewhere) {
		t.Fatalf("refresh=%+v mutation=%d %s", refreshed, mutation.Code, mutation.Body.String())
	}
}

func TestStalePreObservationCreatesNoSuccessorAndInvalidReviewSurvivesReload(t *testing.T) {
	t.Run("stale confirmation", func(t *testing.T) {
		fixture := newNativeRefreshFixture(t, true)
		active := testActiveChangeSet(fixture.state)
		review := *active.review
		base := *fixture.state.loadedSnapshot
		fixture.advanceTitle(t, base, "External")
		createdSuccessor := false
		fixture.state.beforeAcceptedCAS = func(string) { createdSuccessor = true }
		response := postJSONRequest(t, fixture.handler, "/api/architecture/accept", acceptChangesRequest{
			ProjectSlug: fixture.base.ProjectSlug, StoreID: fixture.base.StoreID, ChangeSetID: active.id, BaseRevision: review.baseRevision,
			CandidateTree: review.candidateTree, Generation: review.generation,
		})
		result := decodeArchitectureBody(t, response)
		if response.Code != http.StatusConflict || result.ActionError != errorArchitectureStale || createdSuccessor || result.Changes == nil || !result.Changes.Stale {
			t.Fatalf("stale confirmation status=%d result=%+v successor=%t", response.Code, result, createdSuccessor)
		}
	})
	t.Run("invalid review reload", func(t *testing.T) {
		fixture := newNativeRefreshFixture(t, false)
		kept := decodeArchitectureResponse(t, postJSONRequest(t, fixture.handler, "/api/architecture/components/edit", observedComponentMutation(fixture.base, componentMutationRequest{
			ComponentID: fixture.component, Title: "   ", TitleChanged: true,
		})))
		if kept.Changes == nil || kept.Changes.Valid {
			t.Fatalf("invalid=%+v", kept.Changes)
		}
		blockedResponse := postJSONRequest(t, fixture.handler, "/api/architecture/review", observedAction(kept))
		blocked := decodeArchitectureBody(t, blockedResponse)
		if blockedResponse.Code != http.StatusUnprocessableEntity || blocked.Changes == nil || blocked.Changes.ReviewBlocker != "title_required" {
			t.Fatalf("blocked=%+v", blocked)
		}
		reloaded := decodeArchitectureResponse(t, postJSONRequest(t, fixture.handler, "/api/projects/open", map[string]any{"project_slug": fixture.base.ProjectSlug}))
		selectActiveChangeSetForTest(&reloaded, kept.Changes.ID)
		if reloaded.Changes == nil || reloaded.Changes.ReviewBlocker != "title_required" || reloaded.Revision != fixture.base.Revision {
			t.Fatalf("reload=%+v", reloaded)
		}
	})
}
