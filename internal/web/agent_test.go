package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"workbraid/internal/agentapi"
	"workbraid/internal/architecture"
)

func postAgent(t *testing.T, handler http.Handler, path string, value any) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeAgentEnvelope(t *testing.T, response *httptest.ResponseRecorder) agentapi.Envelope {
	t.Helper()
	var envelope agentapi.Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode agent response: %v\n%s", err, response.Body.String())
	}
	if err := envelope.Validate(); err != nil {
		t.Fatalf("invalid envelope: %v\n%s", err, response.Body.String())
	}
	return envelope
}

func getAgent(t *testing.T, handler http.Handler, path string) agentapi.Envelope {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return decodeAgentEnvelope(t, response)
}

func resultMap(t *testing.T, envelope agentapi.Envelope) map[string]any {
	t.Helper()
	value, ok := envelope.Result.(map[string]any)
	if !ok {
		t.Fatalf("result = %#v, want object", envelope.Result)
	}
	return value
}

func TestAgentAndBrowserSharePendingAuthorityAndRawRelationshipRepair(t *testing.T) {
	_, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), t.TempDir())
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Agent authority"}))

	state := agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision}
	gatewayEnvelope := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/components/create", agentapi.ComponentCreateRequest{
		StatePreconditions: state, Title: "Gateway", DiagramID: &created.RootDiagramID,
	}))
	if !gatewayEnvelope.OK || gatewayEnvelope.Context.PendingGeneration == nil || *gatewayEnvelope.Context.PendingGeneration != 1 {
		t.Fatalf("gateway create = %+v", gatewayEnvelope)
	}
	gatewayID := resultMap(t, gatewayEnvelope)["component_id"].(string)
	state.PendingGeneration = gatewayEnvelope.Context.PendingGeneration

	workerEnvelope := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/components/create", agentapi.ComponentCreateRequest{
		StatePreconditions: state, Title: "Worker", DiagramID: &created.RootDiagramID,
	}))
	if !workerEnvelope.OK || workerEnvelope.Context.PendingGeneration == nil || *workerEnvelope.Context.PendingGeneration != 2 {
		t.Fatalf("worker create = %+v", workerEnvelope)
	}
	workerID := resultMap(t, workerEnvelope)["component_id"].(string)
	state.PendingGeneration = workerEnvelope.Context.PendingGeneration

	invalid := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/relationships/add", agentapi.RelationshipAddRequest{
		StatePreconditions: state, SourceID: gatewayID, TargetID: "", Label: "",
	}))
	if !invalid.OK || resultMap(t, invalid)["candidate_valid"] != false {
		t.Fatalf("invalid raw relationship was not retained: %+v", invalid)
	}
	state.PendingGeneration = invalid.Context.PendingGeneration

	inspectedRequest := httptest.NewRequest(http.MethodGet, "/api/agent/v1/changes/inspect", nil)
	inspectedResponse := httptest.NewRecorder()
	handler.ServeHTTP(inspectedResponse, inspectedRequest)
	inspected := decodeAgentEnvelope(t, inspectedResponse)
	components := resultMap(t, inspected)["components"].([]any)
	firstRelationships := components[0].(map[string]any)["relationships"].([]any)
	row := firstRelationships[0].(map[string]any)
	if row["target_id"] != "" || row["label"] != "" {
		t.Fatalf("raw invalid row lost fidelity: %#v", row)
	}

	repaired := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/relationships/edit", agentapi.RelationshipEditRequest{
		StatePreconditions: state, SourceID: gatewayID, OldTargetID: "", OldLabel: "", Occurrence: 1, TargetID: workerID, Label: "calls",
	}))
	if !repaired.OK || resultMap(t, repaired)["candidate_valid"] != true {
		t.Fatalf("raw relationship repair = %+v", repaired)
	}
	state.PendingGeneration = repaired.Context.PendingGeneration

	staleBrowser := postJSONRequest(t, handler, "/api/architecture/components/edit", componentMutationRequest{
		ProjectSlug: created.ProjectSlug, StoreID: created.StoreID, ExpectedRevision: created.Revision,
		PendingGenerationObserved: true, ExpectedGeneration: nil,
		ComponentID: gatewayID, Title: "Stale overwrite", TitleChanged: true,
	})
	if staleBrowser.Code != http.StatusConflict {
		t.Fatalf("stale browser mutation status = %d, want conflict", staleBrowser.Code)
	}

	reviewed := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/changes/review", agentapi.ChangesReviewRequest{
		StatePreconditions: state, Generation: *state.PendingGeneration,
	}))
	if !reviewed.OK {
		t.Fatalf("review = %+v", reviewed)
	}
	reviewResult := resultMap(t, reviewed)
	accepted := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/architecture/update", agentapi.ArchitectureUpdateRequest{
		StoreID: created.StoreID, BaseRevision: reviewResult["base_revision"].(string), CandidateTree: reviewResult["candidate_tree"].(string), Generation: uint64(reviewResult["generation"].(float64)),
	}))
	if !accepted.OK || accepted.Context.PendingGeneration != nil || accepted.Context.AcceptedRevision == nil || *accepted.Context.AcceptedRevision == created.Revision {
		t.Fatalf("acceptance = %+v", accepted)
	}

	browser := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/open", map[string]any{"project_slug": created.ProjectSlug}))
	if browser.Revision != *accepted.Context.AcceptedRevision || len(browser.Components) != 2 || browser.Components[0].Relationships[0].TargetID != workerID {
		t.Fatalf("browser did not observe agent acceptance: %+v", browser)
	}
}

func TestAgentProjectCatalogSelectionCloseAndRefreshUseRunningAuthority(t *testing.T) {
	_, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), t.TempDir())
	status := getAgent(t, handler, "/api/agent/v1/status")
	if !status.OK || status.Context.Project != nil || resultMap(t, status)["protocol"] != agentapi.Protocol {
		t.Fatalf("initial status = %+v", status)
	}
	created := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/projects/create", agentapi.ProjectCreateRequest{Name: "Selection authority"}))
	if !created.OK || created.Context.Project == nil || created.Context.AcceptedRevision == nil {
		t.Fatalf("create = %+v", created)
	}
	storeID := created.Context.Project.StoreID
	revision := *created.Context.AcceptedRevision
	if current := getAgent(t, handler, "/api/agent/v1/projects/current"); !current.OK || current.Context.Project == nil || current.Context.Project.StoreID != storeID {
		t.Fatalf("current = %+v", current)
	}
	listed := getAgent(t, handler, "/api/agent/v1/projects/list")
	if !listed.OK || len(resultMap(t, listed)["projects"].([]any)) != 1 {
		t.Fatalf("list = %+v", listed)
	}
	refreshed := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/architecture/refresh", agentapi.ArchitectureRefreshRequest{StoreID: storeID, AcceptedRevision: revision}))
	if !refreshed.OK || resultMap(t, refreshed)["classification"] != "unchanged" {
		t.Fatalf("refresh = %+v", refreshed)
	}
	closed := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/projects/close", agentapi.ProjectCloseRequest{StoreID: storeID}))
	if !closed.OK || closed.Context.Project != nil {
		t.Fatalf("close = %+v", closed)
	}
	closedAgain := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/projects/close", agentapi.ProjectCloseRequest{StoreID: storeID}))
	if closedAgain.OK || closedAgain.Error == nil || closedAgain.Error.Code != "project_not_open" {
		t.Fatalf("close without current project = %+v", closedAgain)
	}
	notOpen := getAgent(t, handler, "/api/agent/v1/architecture/inspect")
	if notOpen.OK || notOpen.Error == nil || notOpen.Error.Code != "project_not_open" {
		t.Fatalf("inspect without project = %+v", notOpen)
	}
	opened := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/projects/open", agentapi.ProjectOpenRequest{Slug: "selection-authority"}))
	if !opened.OK || opened.Context.Project == nil || opened.Context.Project.StoreID != storeID || opened.Context.AcceptedRevision == nil || *opened.Context.AcceptedRevision != revision {
		t.Fatalf("reopen = %+v", opened)
	}
}

func TestAgentDistinguishesDefiniteAcceptanceWithReloadRequired(t *testing.T) {
	state, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), t.TempDir())
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Reload classification"}))
	kept := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/components/create", agentapi.ComponentCreateRequest{
		StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision},
		Title:              "Gateway", DiagramID: &created.RootDiagramID,
	}))
	generation := *kept.Context.PendingGeneration
	reviewed := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/changes/review", agentapi.ChangesReviewRequest{
		StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision, PendingGeneration: &generation},
		Generation:         generation,
	}))
	review := resultMap(t, reviewed)
	state.publicationFailure = func() error { return errors.New("publication unavailable") }
	updated := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/architecture/update", agentapi.ArchitectureUpdateRequest{
		StoreID: created.StoreID, BaseRevision: review["base_revision"].(string), CandidateTree: review["candidate_tree"].(string), Generation: generation,
	}))
	if updated.OK || updated.Error == nil || updated.Error.Code != "accepted_reload_required" || updated.Context.PendingGeneration != nil ||
		updated.Context.AcceptedRevision == nil || *updated.Context.AcceptedRevision == created.Revision {
		t.Fatalf("reload-required classification = %+v", updated)
	}
}

func TestAgentClassifiesOperationalCandidateFailureAndAcceptanceConflicts(t *testing.T) {
	t.Run("candidate construction is operational failure", func(t *testing.T) {
		state, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), t.TempDir())
		created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Operational failure"}))
		state.candidateConstructionFailure = func() error { return errors.New("git unavailable") }
		mutation := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/components/create", agentapi.ComponentCreateRequest{
			StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision},
			Title:              "Gateway", DiagramID: &created.RootDiagramID,
		}))
		if mutation.OK || mutation.Error == nil || mutation.Error.Code != "operation_failed" || mutation.Context.PendingGeneration == nil {
			t.Fatalf("mutation operational classification = %+v", mutation)
		}
		generation := *mutation.Context.PendingGeneration
		state.stateMutex.Lock()
		componentID := state.pending.changes[0].ID
		state.stateMutex.Unlock()
		for _, next := range []struct {
			name string
			path string
			body any
		}{
			{name: "move home", path: "/api/agent/v1/components/move-home", body: agentapi.ComponentMoveHomeRequest{
				StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision, PendingGeneration: &generation},
				ComponentID:        componentID, DiagramID: created.RootDiagramID,
			}},
			{name: "show component", path: "/api/agent/v1/diagrams/show-component", body: agentapi.DiagramComponentRequest{
				StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision, PendingGeneration: &generation},
				ComponentID:        componentID, DiagramID: created.RootDiagramID,
			}},
			{name: "stop showing component", path: "/api/agent/v1/diagrams/stop-showing-component", body: agentapi.DiagramComponentRequest{
				StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision, PendingGeneration: &generation},
				ComponentID:        componentID, DiagramID: created.RootDiagramID,
			}},
		} {
			t.Run(next.name, func(t *testing.T) {
				result := decodeAgentEnvelope(t, postAgent(t, handler, next.path, next.body))
				if result.OK || result.Error == nil || result.Error.Code != "operation_failed" {
					t.Fatalf("existing operational pending classification = %+v", result)
				}
			})
		}
		review := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/changes/review", agentapi.ChangesReviewRequest{
			StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision, PendingGeneration: &generation},
			Generation:         generation,
		}))
		if review.OK || review.Error == nil || review.Error.Code != "operation_failed" {
			t.Fatalf("review operational classification = %+v", review)
		}
	})

	for _, scenario := range []struct {
		name string
		want string
		hook bool
	}{
		{name: "known non-current before update", want: "architecture_non_current"},
		{name: "true final CAS conflict", want: "accepted_conflict", hook: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			state, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), t.TempDir())
			created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Acceptance classification"}))
			kept := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/components/create", agentapi.ComponentCreateRequest{
				StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision},
				Title:              "Gateway", DiagramID: &created.RootDiagramID,
			}))
			generation := *kept.Context.PendingGeneration
			reviewed := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/changes/review", agentapi.ChangesReviewRequest{
				StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision, PendingGeneration: &generation},
				Generation:         generation,
			}))
			review := resultMap(t, reviewed)
			base := *state.loadedSnapshot
			externalCandidate, err := state.architecture.ConstructCandidate(context.Background(), base, nil, architecture.CandidateComposition{
				DiagramTitles: []architecture.DiagramTitleChange{{DiagramID: base.RootDiagramID(), Title: "External"}},
			})
			if err != nil {
				t.Fatal(err)
			}
			advanceExternal := func() {
				revision, createErr := state.architecture.CreateSuccessor(context.Background(), base, externalCandidate)
				if createErr != nil {
					t.Fatal(createErr)
				}
				if advanceErr := state.architecture.AdvanceAccepted(context.Background(), base, revision); advanceErr != nil {
					t.Fatal(advanceErr)
				}
			}
			if scenario.hook {
				state.beforeAcceptedCAS = func(string) { advanceExternal() }
			} else {
				advanceExternal()
			}
			updated := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/architecture/update", agentapi.ArchitectureUpdateRequest{
				StoreID: created.StoreID, BaseRevision: review["base_revision"].(string), CandidateTree: review["candidate_tree"].(string), Generation: generation,
			}))
			if updated.OK || updated.Error == nil || updated.Error.Code != scenario.want {
				t.Fatalf("update classification = %+v, want %q", updated, scenario.want)
			}
		})
	}
}

func TestAgentRejectsUnexpectedOriginAndWrongGenerationWithoutMutation(t *testing.T) {
	state, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), t.TempDir())
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Preconditions"}))

	request := httptest.NewRequest(http.MethodPost, "/api/agent/v1/components/create", bytes.NewReader([]byte(`{}`)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://example.com")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if envelope := decodeAgentEnvelope(t, response); envelope.OK || envelope.Error.Code != "invalid_request" {
		t.Fatalf("unexpected origin = %+v", envelope)
	}

	wrong := uint64(9)
	response = postAgent(t, handler, "/api/agent/v1/components/create", agentapi.ComponentCreateRequest{
		StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision, PendingGeneration: &wrong},
		Title:              "No mutation", DiagramID: &created.RootDiagramID,
	})
	envelope := decodeAgentEnvelope(t, response)
	if envelope.OK || envelope.Error.Code != "pending_generation_mismatch" {
		t.Fatalf("wrong generation = %+v", envelope)
	}
	state.stateMutex.Lock()
	defer state.stateMutex.Unlock()
	if state.pending != nil {
		t.Fatalf("wrong generation created pending state: %+v", state.pending)
	}
}

func TestAgentRefreshAndReviewDistinguishWrongRevisionFromWrongGeneration(t *testing.T) {
	_, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), t.TempDir())
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Distinct preconditions"}))
	kept := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/components/create", agentapi.ComponentCreateRequest{
		StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision},
		Title:              "Gateway", DiagramID: &created.RootDiagramID,
	}))
	generation := *kept.Context.PendingGeneration
	wrongRevision := "0000000000000000000000000000000000000000"
	refreshed := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/architecture/refresh", agentapi.ArchitectureRefreshRequest{
		StoreID: created.StoreID, AcceptedRevision: wrongRevision,
	}))
	if refreshed.OK || refreshed.Error == nil || refreshed.Error.Code != "architecture_non_current" {
		t.Fatalf("wrong-revision Refresh = %+v", refreshed)
	}
	reviewed := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/changes/review", agentapi.ChangesReviewRequest{
		StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: wrongRevision, PendingGeneration: &generation},
		Generation:         generation,
	}))
	if reviewed.OK || reviewed.Error == nil || reviewed.Error.Code != "architecture_non_current" {
		t.Fatalf("wrong-revision Review = %+v", reviewed)
	}
	wrongGeneration := generation + 1
	generationMismatch := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/changes/review", agentapi.ChangesReviewRequest{
		StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision, PendingGeneration: &wrongGeneration},
		Generation:         wrongGeneration,
	}))
	if generationMismatch.OK || generationMismatch.Error == nil || generationMismatch.Error.Code != "pending_generation_mismatch" {
		t.Fatalf("wrong-generation Review = %+v", generationMismatch)
	}
}

func TestAgentMutationRacingBrowserDiscardHasOneSerializedOutcome(t *testing.T) {
	_, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), t.TempDir())
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Serialized clients"}))
	initial := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/components/create", agentapi.ComponentCreateRequest{
		StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision},
		Title:              "Gateway", DiagramID: &created.RootDiagramID,
	}))
	componentID := resultMap(t, initial)["component_id"].(string)
	generation := *initial.Context.PendingGeneration

	editData, err := json.Marshal(agentapi.ComponentEditRequest{
		StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision, PendingGeneration: &generation},
		ComponentID:        componentID, Title: pointerTo("Edited by agent"),
	})
	if err != nil {
		t.Fatal(err)
	}
	editRequest := httptest.NewRequest(http.MethodPost, "/api/agent/v1/components/edit", bytes.NewReader(editData))
	editRequest.Header.Set("Content-Type", "application/json")
	editResponse := httptest.NewRecorder()
	discardData, err := json.Marshal(architectureActionRequest{
		ProjectSlug: created.ProjectSlug, StoreID: created.StoreID,
		ExpectedGeneration: &generation, PendingGenerationObserved: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	discardRequest := httptest.NewRequest(http.MethodPost, "/api/architecture/discard", bytes.NewReader(discardData))
	discardRequest.Header.Set("Content-Type", "application/json")
	discardRequest.Header.Set("Origin", "http://127.0.0.1:8080")
	discardResponse := httptest.NewRecorder()

	start := make(chan struct{})
	done := make(chan struct{}, 2)
	go func() { <-start; handler.ServeHTTP(editResponse, editRequest); done <- struct{}{} }()
	go func() { <-start; handler.ServeHTTP(discardResponse, discardRequest); done <- struct{}{} }()
	close(start)
	<-done
	<-done

	edit := decodeAgentEnvelope(t, editResponse)
	inspectRequest := httptest.NewRequest(http.MethodGet, "/api/agent/v1/changes/inspect", nil)
	inspectResponse := httptest.NewRecorder()
	handler.ServeHTTP(inspectResponse, inspectRequest)
	inspected := decodeAgentEnvelope(t, inspectResponse)
	if edit.OK {
		if discardResponse.Code != http.StatusConflict || inspected.Context.PendingGeneration == nil || *inspected.Context.PendingGeneration != generation+1 {
			t.Fatalf("edit-first outcome mixed: edit=%+v discard=%d inspect=%+v", edit, discardResponse.Code, inspected)
		}
		components := resultMap(t, inspected)["components"].([]any)
		if components[0].(map[string]any)["title"] != "Edited by agent" {
			t.Fatalf("agent edit missing after race: %#v", components)
		}
		return
	}
	if edit.Error == nil || edit.Error.Code != "pending_generation_mismatch" || discardResponse.Code != http.StatusOK || inspected.Context.PendingGeneration != nil || resultMap(t, inspected)["changes"] != nil {
		t.Fatalf("discard-first outcome mixed: edit=%+v discard=%d inspect=%+v", edit, discardResponse.Code, inspected)
	}
}

func TestAgentResultAndContextStayAtomicAcrossReviewAndAcceptanceRaces(t *testing.T) {
	t.Run("review and mutation", func(t *testing.T) {
		state, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), t.TempDir())
		created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Atomic review"}))
		kept := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/components/create", agentapi.ComponentCreateRequest{
			StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision},
			Title:              "Gateway", DiagramID: &created.RootDiagramID,
		}))
		generation := *kept.Context.PendingGeneration
		componentID := resultMap(t, kept)["component_id"].(string)
		reached, release := make(chan struct{}), make(chan struct{})
		state.beforeAgentResultCapture = func() {
			state.beforeAgentResultCapture = nil
			close(reached)
			<-release
		}
		reviewDone := make(chan *httptest.ResponseRecorder, 1)
		go func() {
			reviewDone <- postAgent(t, handler, "/api/agent/v1/changes/review", agentapi.ChangesReviewRequest{
				StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision, PendingGeneration: &generation},
				Generation:         generation,
			})
		}()
		<-reached
		mutationDone := make(chan *httptest.ResponseRecorder, 1)
		go func() {
			mutationDone <- postAgent(t, handler, "/api/agent/v1/components/edit", agentapi.ComponentEditRequest{
				StatePreconditions: agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision, PendingGeneration: &generation},
				ComponentID:        componentID, Title: pointerTo("After review"),
			})
		}()
		select {
		case <-mutationDone:
			t.Fatal("mutation entered between review result and context capture")
		case <-time.After(25 * time.Millisecond):
		}
		close(release)
		reviewed := decodeAgentEnvelope(t, <-reviewDone)
		mutated := decodeAgentEnvelope(t, <-mutationDone)
		review := resultMap(t, reviewed)
		if !reviewed.OK || review["generation"] != float64(generation) || reviewed.Context.PendingGeneration == nil || *reviewed.Context.PendingGeneration != generation {
			t.Fatalf("review result/context mixed: %+v", reviewed)
		}
		if !mutated.OK || mutated.Context.PendingGeneration == nil || *mutated.Context.PendingGeneration != generation+1 {
			t.Fatalf("serialized mutation = %+v", mutated)
		}
	})

	t.Run("acceptance and project switch", func(t *testing.T) {
		state, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), t.TempDir())
		first := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "First atomic"}))
		second := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Second atomic"}))
		decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/open", map[string]any{"project_slug": first.ProjectSlug}))
		kept := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/components/create", agentapi.ComponentCreateRequest{
			StatePreconditions: agentapi.StatePreconditions{StoreID: first.StoreID, AcceptedRevision: first.Revision},
			Title:              "Gateway", DiagramID: &first.RootDiagramID,
		}))
		generation := *kept.Context.PendingGeneration
		reviewed := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/changes/review", agentapi.ChangesReviewRequest{
			StatePreconditions: agentapi.StatePreconditions{StoreID: first.StoreID, AcceptedRevision: first.Revision, PendingGeneration: &generation},
			Generation:         generation,
		}))
		review := resultMap(t, reviewed)
		reached, release := make(chan struct{}), make(chan struct{})
		state.beforeAgentResultCapture = func() {
			state.beforeAgentResultCapture = nil
			close(reached)
			<-release
		}
		updateDone := make(chan *httptest.ResponseRecorder, 1)
		go func() {
			updateDone <- postAgent(t, handler, "/api/agent/v1/architecture/update", agentapi.ArchitectureUpdateRequest{
				StoreID: first.StoreID, BaseRevision: review["base_revision"].(string), CandidateTree: review["candidate_tree"].(string), Generation: generation,
			})
		}()
		<-reached
		openDone := make(chan *httptest.ResponseRecorder, 1)
		go func() {
			openDone <- postAgent(t, handler, "/api/agent/v1/projects/open", agentapi.ProjectOpenRequest{Slug: second.ProjectSlug})
		}()
		select {
		case <-openDone:
			t.Fatal("project switch entered between acceptance result and context capture")
		case <-time.After(25 * time.Millisecond):
		}
		close(release)
		updated := decodeAgentEnvelope(t, <-updateDone)
		opened := decodeAgentEnvelope(t, <-openDone)
		result := resultMap(t, updated)
		if !updated.OK || updated.Context.Project == nil || updated.Context.Project.StoreID != first.StoreID ||
			updated.Context.AcceptedRevision == nil || result["accepted_revision"] != *updated.Context.AcceptedRevision || updated.Context.PendingGeneration != nil {
			t.Fatalf("acceptance result/context mixed: %+v", updated)
		}
		if !opened.OK || opened.Context.Project == nil || opened.Context.Project.StoreID != second.StoreID {
			t.Fatalf("serialized switch = %+v", opened)
		}
	})
}

func TestAgentRepairsAndRemovesEveryInvalidRawRelationshipSelector(t *testing.T) {
	_, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), t.TempDir())
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Raw selectors"}))
	state := agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision}
	source := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/components/create", agentapi.ComponentCreateRequest{
		StatePreconditions: state, Title: "Source", DiagramID: &created.RootDiagramID,
	}))
	sourceID := resultMap(t, source)["component_id"].(string)
	state.PendingGeneration = source.Context.PendingGeneration
	target := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/components/create", agentapi.ComponentCreateRequest{
		StatePreconditions: state, Title: "Target", DiagramID: &created.RootDiagramID,
	}))
	targetID := resultMap(t, target)["component_id"].(string)
	state.PendingGeneration = target.Context.PendingGeneration

	add := func(targetValue, label string) agentapi.Envelope {
		t.Helper()
		result := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/relationships/add", agentapi.RelationshipAddRequest{
			StatePreconditions: state, SourceID: sourceID, TargetID: targetValue, Label: label,
		}))
		if !result.OK {
			t.Fatalf("add raw relationship: %+v", result)
		}
		state.PendingGeneration = result.Context.PendingGeneration
		return result
	}
	add("", "")
	add("not-a-component-id", "   ")
	unresolved := "00000000-0000-4000-8000-000000000099"
	add(unresolved, "calls")

	inspectedRequest := httptest.NewRequest(http.MethodGet, "/api/agent/v1/changes/inspect", nil)
	inspectedResponse := httptest.NewRecorder()
	handler.ServeHTTP(inspectedResponse, inspectedRequest)
	rows := resultMap(t, decodeAgentEnvelope(t, inspectedResponse))["components"].([]any)[0].(map[string]any)["relationships"].([]any)
	if len(rows) != 3 || rows[0].(map[string]any)["target_id"] != "" || rows[0].(map[string]any)["label"] != "" ||
		rows[1].(map[string]any)["target_id"] != "not-a-component-id" || rows[1].(map[string]any)["label"] != "   " ||
		rows[2].(map[string]any)["target_id"] != unresolved {
		t.Fatalf("raw pending rows lost fidelity: %#v", rows)
	}

	removedMalformed := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/relationships/remove", agentapi.RelationshipRemoveRequest{
		StatePreconditions: state, SourceID: sourceID, TargetID: "not-a-component-id", Label: "   ", Occurrence: 1,
	}))
	if !removedMalformed.OK {
		t.Fatalf("remove malformed row: %+v", removedMalformed)
	}
	state.PendingGeneration = removedMalformed.Context.PendingGeneration
	repairedEmpty := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/relationships/edit", agentapi.RelationshipEditRequest{
		StatePreconditions: state, SourceID: sourceID, OldTargetID: "", OldLabel: "", Occurrence: 1, TargetID: targetID, Label: "recovers",
	}))
	if !repairedEmpty.OK {
		t.Fatalf("repair empty row: %+v", repairedEmpty)
	}
	state.PendingGeneration = repairedEmpty.Context.PendingGeneration
	repairedUnresolved := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/relationships/edit", agentapi.RelationshipEditRequest{
		StatePreconditions: state, SourceID: sourceID, OldTargetID: unresolved, OldLabel: "calls", Occurrence: 1, TargetID: targetID, Label: "calls",
	}))
	if !repairedUnresolved.OK || resultMap(t, repairedUnresolved)["candidate_valid"] != true {
		t.Fatalf("repair unresolved row: %+v", repairedUnresolved)
	}
	state.PendingGeneration = repairedUnresolved.Context.PendingGeneration
	add(targetID, "duplicate")
	add(targetID, "duplicate")
	removedSecond := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/relationships/remove", agentapi.RelationshipRemoveRequest{
		StatePreconditions: state, SourceID: sourceID, TargetID: targetID, Label: "duplicate", Occurrence: 2,
	}))
	if !removedSecond.OK {
		t.Fatalf("remove duplicate occurrence: %+v", removedSecond)
	}
	state.PendingGeneration = removedSecond.Context.PendingGeneration

	inspectedRequest = httptest.NewRequest(http.MethodGet, "/api/agent/v1/changes/inspect", nil)
	inspectedResponse = httptest.NewRecorder()
	handler.ServeHTTP(inspectedResponse, inspectedRequest)
	rows = resultMap(t, decodeAgentEnvelope(t, inspectedResponse))["components"].([]any)[0].(map[string]any)["relationships"].([]any)
	duplicateCount := 0
	for _, rowValue := range rows {
		row := rowValue.(map[string]any)
		if row["target_id"] == targetID && row["label"] == "duplicate" {
			duplicateCount++
		}
	}
	if duplicateCount != 1 {
		t.Fatalf("occurrence selection removed wrong rows: %#v", rows)
	}
}

func TestAgentUsesBrowserCompositionOperationsForNestedHomeAndReferenceChanges(t *testing.T) {
	_, handler := newHandler("http://127.0.0.1:8080", t.TempDir(), t.TempDir())
	created := decodeArchitectureResponse(t, postJSONRequest(t, handler, "/api/projects/create", map[string]any{"name": "Agent composition"}))
	state := agentapi.StatePreconditions{StoreID: created.StoreID, AcceptedRevision: created.Revision}
	create := func(title string) agentapi.Envelope {
		t.Helper()
		result := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/components/create", agentapi.ComponentCreateRequest{
			StatePreconditions: state, Title: title, DiagramID: &created.RootDiagramID,
		}))
		state.PendingGeneration = result.Context.PendingGeneration
		return result
	}
	gateway := create("Gateway")
	worker := create("Worker")
	gatewayID := resultMap(t, gateway)["component_id"].(string)
	workerID := resultMap(t, worker)["component_id"].(string)
	detail := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/diagrams/create-detail", agentapi.DiagramCreateDetailRequest{
		StatePreconditions: state, ComponentID: gatewayID, Title: "Gateway internals",
	}))
	if !detail.OK {
		t.Fatalf("create detail: %+v", detail)
	}
	detailID := resultMap(t, detail)["diagram_id"].(string)
	state.PendingGeneration = detail.Context.PendingGeneration
	moved := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/components/move-home", agentapi.ComponentMoveHomeRequest{
		StatePreconditions: state, ComponentID: workerID, DiagramID: detailID,
	}))
	if !moved.OK {
		t.Fatalf("move home: %+v", moved)
	}
	state.PendingGeneration = moved.Context.PendingGeneration
	shown := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/diagrams/show-component", agentapi.DiagramComponentRequest{
		StatePreconditions: state, DiagramID: detailID, ComponentID: gatewayID,
	}))
	if !shown.OK {
		t.Fatalf("show reference: %+v", shown)
	}
	state.PendingGeneration = shown.Context.PendingGeneration
	renamed := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/diagrams/edit-title", agentapi.DiagramEditTitleRequest{
		StatePreconditions: state, DiagramID: detailID, Title: "Runtime internals",
	}))
	if !renamed.OK {
		t.Fatalf("edit title: %+v", renamed)
	}
	state.PendingGeneration = renamed.Context.PendingGeneration
	stopped := decodeAgentEnvelope(t, postAgent(t, handler, "/api/agent/v1/diagrams/stop-showing-component", agentapi.DiagramComponentRequest{
		StatePreconditions: state, DiagramID: detailID, ComponentID: gatewayID,
	}))
	if !stopped.OK || resultMap(t, stopped)["candidate_valid"] != true {
		t.Fatalf("stop reference: %+v", stopped)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/agent/v1/changes/inspect", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	changes := resultMap(t, decodeAgentEnvelope(t, response))
	if len(changes["detail_diagrams"].([]any)) != 1 || len(changes["home_moves"].([]any)) != 1 || len(changes["references"].([]any)) != 2 {
		t.Fatalf("composition projection incomplete: %#v", changes)
	}
	diagrams := changes["candidate"].(map[string]any)["diagrams"].([]any)
	foundRuntime := false
	for _, value := range diagrams {
		diagram := value.(map[string]any)
		if diagram["id"] == detailID && diagram["title"] == "Runtime internals" {
			foundRuntime = true
		}
	}
	if !foundRuntime {
		t.Fatalf("candidate did not retain nested title: %#v", diagrams)
	}
}

func pointerTo(value string) *string { return &value }
