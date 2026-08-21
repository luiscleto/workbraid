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
)

func TestReviewVisualProjectionUsesOneBoundRealGitCandidateAndExactComparison(t *testing.T) {
	db := openWebTestDatabase(t)
	source := createSourceRepository(t)
	sourceBefore := snapshotRepository(t, source)
	dataDirectory := t.TempDir()
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	manifest := []byte(runGit(t, dataDirectory, "--git-dir", storePath, "show", initialized.Revision+":architecture.yaml") + "\n")

	const (
		gatewayID = "32c58784-e3ae-42d3-b32e-13f16f97e304"
		workerID  = "96bd714b-c65a-452d-ae64-f136665d1a6e"
		docsID    = "ad27ed8c-2e96-4b74-aaef-6f18eed80a8d"
		titleID   = "ae253c7e-59e7-4485-a724-efaa470df987"
	)
	externalBase := advanceAcceptedToComponents(t, storePath, initialized.Revision, manifest, []testComponent{
		{path: "gateway.md", mode: "100644", source: []byte("---\nid: \"" + gatewayID + "\"\nrelationships:\n  - target: \"" + workerID + "\"\n    label: calls\n  - target: \"" + workerID + "\"\n    label: parallel\n  - target: \"" + workerID + "\"\n    label: parallel\n  - target: \"" + docsID + "\"\n    label: old target\n---\n# Gateway\nGateway body.\n")},
		{path: "worker.md", mode: "100644", source: []byte("---\nid: \"" + workerID + "\"\nrelationships:\n  - target: \"" + gatewayID + "\"\n    label: returns\n---\n# Worker\nWorker body.\n")},
		{path: "docs.md", mode: "100644", source: []byte("---\nid: \"" + docsID + "\"\n---\n# Docs\nOld documentation.\n")},
		{path: "title.md", mode: "100644", source: []byte("---\nid: \"" + titleID + "\"\n---\n# Internal name\nUnchanged title-only body.\n")},
	})
	base := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	if base.Revision != externalBase || len(base.Components) != 4 {
		t.Fatalf("base fixture did not load: %+v", base)
	}

	added := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: externalBase, Title: "Telemetry", Description: "New component.\n",
	}))
	telemetryID := added.Changes.Components[0].ID
	for _, change := range added.Changes.Components {
		if change.New {
			telemetryID = change.ID
		}
	}
	decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: externalBase, ComponentID: gatewayID, Title: "Public Gateway", TitleChanged: true,
		RelationshipsChanged: true,
		Relationships: []relationshipResponse{
			{TargetID: telemetryID, Label: "calls"},
			{TargetID: workerID, Label: "parallel"},
			{TargetID: workerID, Label: "parallel"},
			{TargetID: workerID, Label: "parallel"},
			{TargetID: docsID, Label: "new target"},
			{TargetID: telemetryID, Label: "  observes λ  "},
		},
	}))
	decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: externalBase, ComponentID: workerID, RelationshipsChanged: true,
		Relationships: []relationshipResponse{{TargetID: gatewayID, Label: "returns differently"}},
	}))
	decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: externalBase, ComponentID: docsID, Description: "New documentation.\n", DescriptionChanged: true,
	}))
	decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: externalBase, ComponentID: titleID, Title: "Public name", TitleChanged: true,
	}))
	decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: externalBase, ComponentID: telemetryID, RelationshipsChanged: true,
		Relationships: []relationshipResponse{{TargetID: gatewayID, Label: "cycles"}},
	}))

	boundTree := state.pending.candidate.Tree()
	refBefore := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted")
	objectsBefore := runGit(t, dataDirectory, "--git-dir", storePath, "cat-file", "--batch-all-objects", "--batch-check=%(objecttype) %(objectname)")
	associationsBefore := snapshotAssociations(t, db)

	reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	if reviewed.Changes == nil || reviewed.Changes.Review == nil {
		t.Fatalf("visual review missing: %+v", reviewed.Changes)
	}
	review := reviewed.Changes.Review
	if review.BaseRevision != base.Revision || review.CandidateTree != boundTree || review.Generation != state.pending.generation ||
		review.Before.Revision != base.Revision || review.WithChanges.Revision != boundTree {
		t.Fatalf("visual projection and confirmation binding diverged: %+v", review)
	}
	if len(review.Before.Components) != len(base.Components) || len(review.WithChanges.Components) != 5 {
		t.Fatalf("bound projections = before %d with %d", len(review.Before.Components), len(review.WithChanges.Components))
	}
	for index, component := range base.Components {
		projected := review.Before.Components[index]
		if projected.ID != component.ID || projected.Title != component.Title || projected.Description != component.Description || projected.Filename != component.Filename {
			t.Fatalf("base projection differs at %d: accepted=%+v review=%+v", index, component, projected)
		}
	}

	componentStatus := make(map[string]string)
	for _, change := range review.Comparison.Components {
		componentStatus[change.ComponentID] = change.Status
	}
	if componentStatus[gatewayID] != "content_changed" || componentStatus[docsID] != "content_changed" ||
		componentStatus[titleID] != "content_changed" || componentStatus[telemetryID] != "added" {
		t.Fatalf("component comparison = %+v", review.Comparison.Components)
	}
	if _, falselyChanged := componentStatus[workerID]; falselyChanged {
		t.Fatalf("relationship-only Worker was content changed: %+v", review.Comparison.Components)
	}

	addedFacts := make(map[string][]reviewRelationshipChangeResponse)
	removedFacts := make(map[string][]reviewRelationshipChangeResponse)
	for _, change := range review.Comparison.Relationships {
		fact := change.SourceID + "\x00" + change.TargetID + "\x00" + change.Label
		if change.Status == "added" {
			addedFacts[fact] = append(addedFacts[fact], change)
		} else if change.Status == "removed" {
			removedFacts[fact] = append(removedFacts[fact], change)
		}
		if change.Path != "components/"+filenameForComponent(review.WithChanges.Components, review.Before.Components, change.SourceID) {
			t.Fatalf("relationship change has wrong source path: %+v", change)
		}
	}
	parallelFact := gatewayID + "\x00" + workerID + "\x00parallel"
	if len(addedFacts[parallelFact]) != 1 || addedFacts[parallelFact][0].Occurrence != 3 {
		t.Fatalf("parallel multiplicity comparison = %+v", addedFacts[parallelFact])
	}
	if len(removedFacts[gatewayID+"\x00"+workerID+"\x00calls"]) != 1 || len(addedFacts[gatewayID+"\x00"+telemetryID+"\x00calls"]) != 1 ||
		len(removedFacts[gatewayID+"\x00"+docsID+"\x00old target"]) != 1 || len(addedFacts[gatewayID+"\x00"+docsID+"\x00new target"]) != 1 ||
		len(addedFacts[gatewayID+"\x00"+telemetryID+"\x00  observes λ  "]) != 1 {
		t.Fatalf("target/label/exact-label comparison missing:\nadded=%+v\nremoved=%+v", addedFacts, removedFacts)
	}
	for _, change := range review.Comparison.Relationships {
		if change.Key == "" || (change.Status == "removed" && change.BeforeKey == "") {
			t.Fatalf("projection occurrence key missing: %+v", change)
		}
	}

	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != refBefore {
		t.Fatalf("visual projection changed accepted from %s to %s", refBefore, got)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "cat-file", "--batch-all-objects", "--batch-check=%(objecttype) %(objectname)"); got != objectsBefore {
		t.Fatalf("visual projection wrote Git objects\nbefore:\n%s\nafter:\n%s", objectsBefore, got)
	}
	if got := snapshotAssociations(t, db); got != associationsBefore {
		t.Fatalf("visual projection changed SQLite\nbefore:\n%s\nafter:\n%s", associationsBefore, got)
	}
	if got := snapshotRepository(t, source); got != sourceBefore {
		t.Fatalf("visual projection changed source repository\nbefore:\n%s\nafter:\n%s", sourceBefore, got)
	}
}

func TestVisualReviewCaptureIsCoherentAcrossConcurrentInvalidation(t *testing.T) {
	for _, action := range []string{"mutation", "discard"} {
		t.Run(action, func(t *testing.T) {
			db := openWebTestDatabase(t)
			source := createSourceRepository(t)
			dataDirectory := t.TempDir()
			state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
			base := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
			pending := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
				SourceRoot: filepath.Clean(source), ExpectedRevision: base.Revision, Title: "Gateway", Description: "First generation.\n",
			}))
			componentID := pending.Changes.Components[0].ID

			body, err := json.Marshal(map[string]string{"source_root": filepath.Clean(source)})
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/api/architecture/review", bytes.NewReader(body))
			request.Header.Set("Origin", testOrigin)
			request.Header.Set("Content-Type", "application/json")
			writer := newBlockingResponseWriter()
			done := make(chan struct{})
			go func() {
				handler.ServeHTTP(writer, request)
				close(done)
			}()
			<-writer.writeStarted

			var invalidated architectureResponse
			if action == "mutation" {
				invalidated = decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
					SourceRoot: filepath.Clean(source), ExpectedRevision: base.Revision, ComponentID: componentID, Description: "Second generation.\n", DescriptionChanged: true,
				}))
			} else {
				invalidated = decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/discard", source))
			}
			if invalidated.Changes != nil && invalidated.Changes.Review != nil {
				t.Fatalf("%s exposed invalidated review: %+v", action, invalidated.Changes.Review)
			}

			close(writer.release)
			<-done
			var captured architectureResponse
			if err := json.Unmarshal(writer.body.Bytes(), &captured); err != nil {
				t.Fatalf("decode captured review: %v\n%s", err, writer.body.String())
			}
			if captured.Changes == nil || captured.Changes.Review == nil {
				t.Fatalf("captured response lost its coherent review: %+v", captured)
			}
			review := captured.Changes.Review
			if review.Generation != 1 || review.BaseRevision != base.Revision || review.CandidateTree != review.WithChanges.Revision ||
				len(review.Before.Components) != 0 || len(review.WithChanges.Components) != 1 || review.WithChanges.Components[0].Description != "First generation.\n" {
				t.Fatalf("captured response mixed generations: %+v", review)
			}
			if state.pending != nil && state.pending.review != nil && state.pending.review.generation == state.pending.generation &&
				state.pending.candidate != nil && state.pending.review.candidateTree == state.pending.candidate.Tree() {
				t.Fatalf("%s retained a current invalidated binding: %+v", action, state.pending.review)
			}
			confirmation := postAcceptChanges(t, handler, testOrigin, source, *review)
			if confirmation.Code != http.StatusConflict {
				t.Fatalf("invalidated captured binding remained confirmable: status=%d body=%s", confirmation.Code, confirmation.Body.String())
			}
		})
	}
}

func filenameForComponent(withChanges, before []componentResponse, componentID string) string {
	for _, components := range [][]componentResponse{withChanges, before} {
		for _, component := range components {
			if component.ID == componentID {
				return component.Filename
			}
		}
	}
	return ""
}

type blockingResponseWriter struct {
	header       http.Header
	status       int
	body         bytes.Buffer
	writeStarted chan struct{}
	release      chan struct{}
	once         sync.Once
}

func newBlockingResponseWriter() *blockingResponseWriter {
	return &blockingResponseWriter{
		header: make(http.Header), writeStarted: make(chan struct{}), release: make(chan struct{}),
	}
}

func (writer *blockingResponseWriter) Header() http.Header { return writer.header }

func (writer *blockingResponseWriter) WriteHeader(status int) { writer.status = status }

func (writer *blockingResponseWriter) Write(value []byte) (int, error) {
	writer.once.Do(func() { close(writer.writeStarted) })
	<-writer.release
	return writer.body.Write(value)
}

func (writer *blockingResponseWriter) String() string { return strings.TrimSpace(writer.body.String()) }
