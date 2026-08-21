package web

import (
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestRelationshipAuthoringUsesCompleteCandidateAndExistingAcceptancePath(t *testing.T) {
	source := createSourceRepository(t)
	sourceBefore := snapshotRepository(t, source)
	dataDirectory := t.TempDir()
	databasePath := filepath.Join(dataDirectory, "workbraid.db")
	db := openWebDatabaseAt(t, databasePath)
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	manifest := []byte(runGit(t, dataDirectory, "--git-dir", storePath, "show", initialized.Revision+":architecture.yaml") + "\n")
	sourceID := uuid.NewString()
	workerID := uuid.NewString()
	duplicateAID := uuid.NewString()
	duplicateBID := uuid.NewString()
	sourceBytes := []byte("---\r\nid: \"" + sourceID + "\"\r\nrelationships:\r\n  - target: \"" + workerID + "\"\r\n    label: \"calls\"\r\n---\r\n \r\n# Gateway #\r\n\r\nExact gateway body  \r\n")
	workerBytes := []byte("---\nid: \"" + workerID + "\"\n---\n# Worker\nWorker body\n")
	duplicateABytes := []byte("---\nid: \"" + duplicateAID + "\"\n---\n# Records\nA\n")
	duplicateBBytes := []byte("---\nid: \"" + duplicateBID + "\"\n---\n# Records\nB\n")
	accepted := advanceAcceptedToComponents(t, storePath, initialized.Revision, manifest, []testComponent{
		{path: "gateway.md", mode: "100755", source: sourceBytes},
		{path: "records-a.md", mode: "100644", source: duplicateABytes},
		{path: "records-b.md", mode: "100644", source: duplicateBBytes},
		{path: "worker.md", mode: "100644", source: workerBytes},
	})
	opened := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	if opened.Revision != accepted || len(opened.Components) != 4 {
		t.Fatalf("accepted fixture did not load: %+v", opened)
	}
	acceptedSourceEntry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", accepted, "components/gateway.md")
	acceptedWorkerEntry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", accepted, "components/worker.md")

	textEdited := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: workerID, Description: "Worker body changed\n", DescriptionChanged: true,
	}))
	created := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, Title: "Queue", Description: "Pending queue\n",
	}))
	if textEdited.Changes == nil || created.Changes == nil || len(created.Changes.Components) != 2 {
		t.Fatalf("text/new pending changes missing: %+v", created.Changes)
	}
	var pendingID string
	for _, component := range created.Changes.Components {
		if component.New {
			pendingID = component.ID
		}
	}
	if _, err := uuid.Parse(pendingID); err != nil {
		t.Fatalf("pending component ID = %q", pendingID)
	}

	label := "  publishes: [events] # α\nnext line  "
	relationshipEdited := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: sourceID, RelationshipsChanged: true,
		Relationships: []relationshipResponse{
			{TargetID: workerID, Label: "calls"},
			{TargetID: pendingID, Label: label},
			{TargetID: pendingID, Label: "observes"},
			{TargetID: duplicateBID, Label: "reads from"},
		},
	}))
	cycled := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: pendingID, RelationshipsChanged: true,
		Relationships: []relationshipResponse{{TargetID: sourceID, Label: "reports to"}},
	}))
	if relationshipEdited.Changes == nil || cycled.Changes == nil || !cycled.Changes.Valid || len(cycled.Changes.Components) != 3 {
		t.Fatalf("relationship candidate missing: %+v", cycled.Changes)
	}
	if got := responseComponent(t, cycled.Changes.Components, sourceID).Relationships; len(got) != 4 || got[1].Label != label || got[1].TargetID != pendingID || got[2].TargetID != pendingID {
		t.Fatalf("pending relationship order/fidelity = %+v", got)
	}
	if got := responseComponent(t, cycled.Changes.Components, pendingID).Relationships; len(got) != 1 || got[0].TargetID != sourceID {
		t.Fatalf("pending cycle = %+v", got)
	}
	var pendingTarget relationshipTargetResponse
	duplicateContexts := 0
	for _, target := range cycled.Changes.RelationshipTargets {
		if target.ID == pendingID {
			pendingTarget = target
		}
		if target.Title == "Records" && target.Context != "" {
			duplicateContexts++
		}
		if target.Title != "Records" && target.Context != "" {
			t.Fatalf("non-colliding target exposed filename context: %+v", target)
		}
	}
	if pendingTarget.Title != "Queue" || !pendingTarget.New || pendingTarget.Context != "" || duplicateContexts != 2 {
		t.Fatalf("pending/collision target context = %+v, duplicate contexts=%d", pendingTarget, duplicateContexts)
	}
	if acceptedView := responseComponent(t, cycled.Components, sourceID).Relationships; len(acceptedView) != 1 || acceptedView[0].TargetID != workerID {
		t.Fatalf("pending topology leaked into accepted response: %+v", acceptedView)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != accepted {
		t.Fatalf("relationship authoring advanced accepted to %s", got)
	}

	blank := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: sourceID, RelationshipsChanged: true,
		Relationships: []relationshipResponse{{TargetID: workerID, Label: "calls"}, {TargetID: pendingID, Label: "  \n "}},
	}))
	if blank.Changes == nil || blank.Changes.Valid || blank.Changes.ValidationCode != "relationship_label_required" ||
		blank.Changes.ValidationItem != sourceID || blank.Changes.ValidationRelationshipPosition != 2 || blank.Changes.ValidationRelationshipField != "label" {
		t.Fatalf("blank label was not retained as invalid pending: %+v", blank.Changes)
	}
	blocked := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	if blocked.Changes == nil || blocked.Changes.Review != nil || blocked.Changes.ReviewBlocker != "relationship_label_required" ||
		blocked.Changes.ValidationItem != sourceID || blocked.Changes.ValidationRelationshipPosition != 2 || blocked.Changes.ValidationRelationshipField != "label" {
		t.Fatalf("blank label review outcome = %+v", blocked.Changes)
	}

	missing := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: sourceID, RelationshipsChanged: true,
		Relationships: []relationshipResponse{{TargetID: workerID, Label: "calls"}, {TargetID: "", Label: "reads from"}},
	}))
	if missing.Changes == nil || missing.Changes.Valid || missing.Changes.ValidationCode != "relationship_target_required" ||
		missing.Changes.ValidationItem != sourceID || missing.Changes.ValidationRelationshipPosition != 2 || missing.Changes.ValidationRelationshipField != "target" {
		t.Fatalf("missing target was not retained as invalid pending: %+v", missing.Changes)
	}
	blocked = decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	if blocked.Changes == nil || blocked.Changes.ReviewBlocker != "relationship_target_required" || len(blocked.Changes.Components) != 3 ||
		blocked.Changes.ValidationItem != sourceID || blocked.Changes.ValidationRelationshipPosition != 2 || blocked.Changes.ValidationRelationshipField != "target" {
		t.Fatalf("missing target review discarded pending state: %+v", blocked.Changes)
	}

	corrected := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: sourceID, RelationshipsChanged: true,
		Relationships: []relationshipResponse{
			{TargetID: workerID, Label: "calls"},
			{TargetID: pendingID, Label: label},
			{TargetID: pendingID, Label: "observes"},
			{TargetID: duplicateBID, Label: "reads from"},
		},
	}))
	if corrected.Changes == nil || !corrected.Changes.Valid {
		t.Fatalf("corrected relationship candidate invalid: %+v", corrected.Changes)
	}
	reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	if reviewed.Changes == nil || reviewed.Changes.Review == nil || !strings.Contains(reviewed.Changes.Review.Diff, "relationships:") || !strings.Contains(reviewed.Changes.Review.Diff, pendingID) {
		t.Fatalf("complete relationship diff missing: %+v", reviewed.Changes)
	}
	review := *reviewed.Changes.Review
	acceptedResponse := decodeArchitectureResponse(t, postAcceptChanges(t, handler, testOrigin, source, review))
	if acceptedResponse.Revision == accepted || acceptedResponse.Changes != nil || len(acceptedResponse.Components) != 5 {
		t.Fatalf("relationship successor was not accepted: %+v", acceptedResponse)
	}
	acceptedRelationships := responseComponent(t, acceptedResponse.Components, sourceID).Relationships
	if len(acceptedRelationships) != 4 || acceptedRelationships[1].Label != label || acceptedRelationships[2].TargetID != pendingID {
		t.Fatalf("accepted relationships = %+v", acceptedRelationships)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show", acceptedResponse.Revision+":components/gateway.md"); !strings.Contains(got, "# Gateway #\r\n\r\nExact gateway body") {
		t.Fatalf("relationship-only acceptance changed H1/body bytes: %q", got)
	}
	if entry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", acceptedResponse.Revision, "components/gateway.md"); !strings.HasPrefix(entry, "100755 blob ") || entry == acceptedSourceEntry {
		t.Fatalf("relationship source mode/blob = %q", entry)
	}
	if entry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", accepted, "components/worker.md"); entry != acceptedWorkerEntry {
		t.Fatalf("accepted base worker fixture changed unexpectedly: %q", entry)
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	restartedDB := openWebDatabaseAt(t, databasePath)
	restarted := NewHandler(restartedDB, testOrigin, t.TempDir(), dataDirectory)
	reopened := decodeArchitectureResponse(t, postOpenProject(t, restarted, testOrigin, source))
	if reopened.Revision != acceptedResponse.Revision || len(responseComponent(t, reopened.Components, sourceID).Relationships) != 4 || responseComponent(t, reopened.Components, sourceID).Relationships[1].Label != label {
		t.Fatalf("restart did not reconstruct relationships: %+v", reopened)
	}
	if after := snapshotRepository(t, source); after != sourceBefore {
		t.Fatalf("relationship authoring changed source repository\nbefore:\n%s\nafter:\n%s", sourceBefore, after)
	}
	if state.pending != nil {
		t.Fatalf("accepted relationship change left pending state: %+v", state.pending)
	}
}

func TestConcurrentRelationshipAndTextMutationMergeAndInvalidateReview(t *testing.T) {
	db := openWebTestDatabase(t)
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	manifest := []byte(runGit(t, dataDirectory, "--git-dir", storePath, "show", initialized.Revision+":architecture.yaml") + "\n")
	sourceID := uuid.NewString()
	targetID := uuid.NewString()
	accepted := advanceAcceptedToComponents(t, storePath, initialized.Revision, manifest, []testComponent{
		{path: "source.md", mode: "100644", source: []byte("---\nid: \"" + sourceID + "\"\n---\n# Source\nBody\n")},
		{path: "target.md", mode: "100644", source: []byte("---\nid: \"" + targetID + "\"\n---\n# Target\nBody\n")},
	})
	decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: sourceID, Description: "Reviewed body\n", DescriptionChanged: true,
	}))
	reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	if reviewed.Changes == nil || reviewed.Changes.Review == nil {
		t.Fatalf("initial review missing: %+v", reviewed.Changes)
	}
	oldReview := *reviewed.Changes.Review

	requests := []componentMutationRequest{
		{SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: sourceID, Title: "Changed source", TitleChanged: true},
		{SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: sourceID, RelationshipsChanged: true, Relationships: []relationshipResponse{{TargetID: targetID, Label: "calls"}}},
	}
	statuses := make([]int, len(requests))
	var wait sync.WaitGroup
	for index := range requests {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			statuses[index] = postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", requests[index]).Code
		}(index)
	}
	wait.Wait()
	for index, status := range statuses {
		if status != http.StatusOK {
			t.Fatalf("concurrent mutation %d status=%d", index, status)
		}
	}
	current := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	change := responseComponent(t, current.Changes.Components, sourceID)
	if change.Title != "Changed source" || change.Description != "Reviewed body\n" || len(change.Relationships) != 1 || change.Relationships[0].TargetID != targetID {
		t.Fatalf("concurrent field intents were lost: %+v", change)
	}
	if current.Changes.Review != nil || state.pending == nil || state.pending.generation != oldReview.Generation+2 {
		t.Fatalf("relationship mutation retained old review: response=%+v pending=%+v", current.Changes, state.pending)
	}
	oldConfirmation := postAcceptChanges(t, handler, testOrigin, source, oldReview)
	if oldConfirmation.Code != http.StatusConflict || decodeArchitectureResponse(t, oldConfirmation).ActionError != errorReviewChanged {
		t.Fatalf("old review accepted after relationship mutation: status=%d body=%s", oldConfirmation.Code, oldConfirmation.Body.String())
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != accepted {
		t.Fatalf("stale reviewed relationship mutation advanced accepted to %s", got)
	}
}

func responseComponent[T interface {
	componentResponse | pendingComponentResponse
}](t *testing.T, components []T, id string) T {
	t.Helper()
	for _, component := range components {
		var componentID string
		switch value := any(component).(type) {
		case componentResponse:
			componentID = value.ID
		case pendingComponentResponse:
			componentID = value.ID
		}
		if componentID == id {
			return component
		}
	}
	t.Fatalf("component %s not found in %+v", id, components)
	var zero T
	return zero
}
