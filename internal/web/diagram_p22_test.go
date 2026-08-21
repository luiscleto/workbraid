package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestWritableV2MixedCandidateAssignsNewComponentToServerValidatedActiveDiagram(t *testing.T) {
	source := createSourceRepository(t)
	sourceBefore := snapshotRepository(t, source)
	dataDirectory := t.TempDir()
	db := openWebDatabaseAt(t, filepath.Join(dataDirectory, "workbraid.db"))
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	accepted := advanceAcceptedToP21V2(t, storePath, initialized.Revision, storeID, "System")
	opened := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))

	edited := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: p21WorkerID,
		Description: "Changed worker documentation.\n", DescriptionChanged: true,
	}))
	created := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, DiagramID: p21DetailID,
		Title: "Queue", Description: "Queue documentation.\n",
	}))
	if edited.Changes == nil || created.Changes == nil || len(created.Changes.Components) != 2 || state.pending == nil {
		t.Fatalf("mixed v2 pending state = %+v", created.Changes)
	}
	var queueID string
	for _, change := range created.Changes.Components {
		if change.New {
			queueID = change.ID
		}
	}
	if _, err := uuid.Parse(queueID); err != nil {
		t.Fatalf("new Component ID = %q", queueID)
	}
	related := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: p21GatewayID,
		RelationshipsChanged: true, Relationships: []relationshipResponse{{TargetID: p21WorkerID, Label: "calls"}, {TargetID: queueID, Label: "publishes"}},
	}))
	if related.Changes == nil || !related.Changes.Valid {
		t.Fatalf("mixed v2 candidate invalid: %+v", related.Changes)
	}
	if got := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); got != accepted {
		t.Fatalf("pending work advanced accepted to %q", got)
	}

	reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	if reviewed.Changes == nil || reviewed.Changes.Review == nil {
		t.Fatalf("review missing: %+v", reviewed.Changes)
	}
	detail := diagramResponseByID(t, reviewed.Changes.Review.WithChanges.Diagrams, p21DetailID)
	foundHome := false
	for _, appearance := range detail.Appearances {
		if appearance.ComponentID == queueID && appearance.Role == "home" {
			foundHome = true
		}
	}
	if !foundHome {
		t.Fatalf("new Component home not in selected Diagram: %+v", detail.Appearances)
	}
	rootBefore := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", accepted, "diagrams/root.yaml")
	rootCandidate := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", reviewed.Changes.Review.CandidateTree, "diagrams/root.yaml")
	if rootCandidate != rootBefore {
		t.Fatalf("unrelated root Diagram changed\nbase: %s\ncandidate: %s", rootBefore, rootCandidate)
	}
	acceptedResult := decodeArchitectureResponse(t, postAcceptChanges(t, handler, testOrigin, source, *reviewed.Changes.Review))
	if acceptedResult.Revision == accepted || acceptedResult.FormatVersion != 2 || acceptedResult.Changes != nil {
		t.Fatalf("v2 acceptance result = %+v", acceptedResult)
	}
	if got := diagramResponseByID(t, acceptedResult.Diagrams, p21DetailID); !diagramHasComponent(got, queueID, "home") {
		t.Fatalf("accepted detail lost home: %+v", got)
	}
	freshDB := openWebDatabaseAt(t, filepath.Join(dataDirectory, "workbraid.db"))
	_, freshHandler := newHandler(freshDB, testOrigin, t.TempDir(), dataDirectory)
	reopened := decodeArchitectureResponse(t, postOpenProject(t, freshHandler, testOrigin, source))
	if reopened.Revision != acceptedResult.Revision || !diagramHasComponent(diagramResponseByID(t, reopened.Diagrams, p21DetailID), queueID, "home") {
		t.Fatalf("fresh process reconstruction = %+v", reopened)
	}
	if got := snapshotRepository(t, source); got != sourceBefore {
		t.Fatal("v2 authoring changed source repository")
	}
	_ = opened
}

func TestV2ReviewRelationshipDeltasMapToExactInternalAndBoundaryEdges(t *testing.T) {
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	db := openWebDatabaseAt(t, filepath.Join(dataDirectory, "workbraid.db"))
	_, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	accepted := advanceAcceptedToP21V2(t, storePath, initialized.Revision, storeID, "System")
	decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))

	// Retaining one parallel writes fact removes exactly its second occurrence;
	// reports is one new fact. Both render internally in root and across a
	// derived boundary in detail, without changing Component content.
	changed := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: accepted, ComponentID: p21WorkerID, RelationshipsChanged: true,
		Relationships: []relationshipResponse{
			{TargetID: p21RecordsID, Label: "writes"},
			{TargetID: p21GatewayID, Label: "reports"},
		},
	}))
	if changed.Changes == nil || !changed.Changes.Valid {
		t.Fatalf("relationship-only v2 candidate = %+v", changed.Changes)
	}
	reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	if reviewed.Changes == nil || reviewed.Changes.Review == nil {
		t.Fatalf("relationship-only v2 review = %+v", reviewed.Changes)
	}
	review := reviewed.Changes.Review
	if len(review.Comparison.Components) != 0 || len(review.Comparison.Relationships) != 2 {
		t.Fatalf("relationship-only review classification = %+v", review.Comparison)
	}

	var added, removed reviewRelationshipChangeResponse
	for _, relationship := range review.Comparison.Relationships {
		switch relationship.Status {
		case "added":
			added = relationship
		case "removed":
			removed = relationship
		}
	}
	if added.SourceID != p21WorkerID || added.TargetID != p21GatewayID || added.SourceTitle != "Worker" || added.TargetTitle != "Shared" || added.Label != "reports" || added.Occurrence != 1 {
		t.Fatalf("added fact = %+v", added)
	}
	if removed.SourceID != p21WorkerID || removed.TargetID != p21RecordsID || removed.SourceTitle != "Worker" || removed.TargetTitle != "Shared" || removed.Label != "writes" || removed.Occurrence != 2 {
		t.Fatalf("removed fact = %+v", removed)
	}

	assertReviewDiagramRelationshipProjection(t, added, "with", p21RootID, p21WorkerID, p21GatewayID, review.WithChanges.Diagrams)
	assertReviewDiagramRelationshipProjection(t, added, "with", p21DetailID, p21WorkerID, "boundary:"+p21GatewayID, review.WithChanges.Diagrams)
	assertReviewDiagramRelationshipProjection(t, removed, "before", p21RootID, p21WorkerID, p21RecordsID, review.Before.Diagrams)
	assertReviewDiagramRelationshipProjection(t, removed, "before", p21DetailID, p21WorkerID, "boundary:"+p21RecordsID, review.Before.Diagrams)

	// Diagram presentation does not participate in the global Relationship
	// fact comparison. The same facts remain unchanged even if an endpoint's
	// selected-Diagram node changes between ordinary and boundary presentation.
	rootWrites := diagramRelationshipByFact(t, diagramResponseByID(t, review.Before.Diagrams, p21RootID), p21WorkerID, p21RecordsID, "writes")
	detailWrites := diagramRelationshipByFact(t, diagramResponseByID(t, review.Before.Diagrams, p21DetailID), p21WorkerID, p21RecordsID, "writes")
	if rootWrites.SourceNodeKey != p21WorkerID || rootWrites.TargetNodeKey != p21RecordsID || detailWrites.SourceNodeKey != p21WorkerID || detailWrites.TargetNodeKey != "boundary:"+p21RecordsID {
		t.Fatalf("same Relationship fact did not project internal/boundary as expected: root=%+v detail=%+v", rootWrites, detailWrites)
	}
	compositionOnlyBefore := review.Before.Components
	compositionOnlyWith := append([]componentResponse(nil), compositionOnlyBefore...)
	compositionOnly := compareReviewProjections(compositionOnlyBefore, compositionOnlyWith)
	compositionOnly.Diagrams, compositionOnly.Appearances = compareDiagramProjections(
		[]diagramResponse{{ID: "composition", Title: "Composition", Appearances: []diagramAppearanceResponse{{ComponentID: p21WorkerID, Role: "home"}, {ComponentID: p21RecordsID, Role: "reference"}}, Relationships: []diagramRelationshipResponse{rootWrites}}},
		[]diagramResponse{{ID: "composition", Title: "Composition", Appearances: []diagramAppearanceResponse{{ComponentID: p21WorkerID, Role: "home"}}, Boundaries: []diagramBoundaryResponse{{Key: "boundary:" + p21RecordsID, ComponentID: p21RecordsID, Title: "Shared"}}, Relationships: []diagramRelationshipResponse{detailWrites}}},
	)
	if len(compositionOnly.Relationships) != 0 || len(compositionOnly.Appearances) == 0 {
		t.Fatalf("composition-only presentation classification = %+v", compositionOnly)
	}
}

func diagramRelationshipByFact(t *testing.T, diagram diagramResponse, sourceID, targetID, label string) diagramRelationshipResponse {
	t.Helper()
	for _, relationship := range diagram.Relationships {
		if relationship.SourceComponentID == sourceID && relationship.TargetComponentID == targetID && relationship.Label == label {
			return relationship
		}
	}
	t.Fatalf("relationship %s — %s — %s missing from Diagram %+v", sourceID, label, targetID, diagram)
	return diagramRelationshipResponse{}
}

func assertReviewDiagramRelationshipProjection(t *testing.T, change reviewRelationshipChangeResponse, side, diagramID, sourceNodeKey, targetNodeKey string, diagrams []diagramResponse) {
	t.Helper()
	for _, projection := range change.DiagramProjections {
		if projection.Side != side || projection.DiagramID != diagramID {
			continue
		}
		if projection.SourceNodeKey != sourceNodeKey || projection.TargetNodeKey != targetNodeKey {
			t.Fatalf("%s %s projection endpoints = %+v", side, diagramID, projection)
		}
		diagram := diagramResponseByID(t, diagrams, diagramID)
		for _, relationship := range diagram.Relationships {
			if relationship.Key == projection.Key && relationship.SourceNodeKey == sourceNodeKey && relationship.TargetNodeKey == targetNodeKey && relationship.SourceComponentID == change.SourceID && relationship.TargetComponentID == change.TargetID && relationship.Label == change.Label {
				return
			}
		}
		t.Fatalf("%s %s projection does not identify an actual rendered edge: %+v diagram=%+v", side, diagramID, projection, diagram.Relationships)
	}
	t.Fatalf("%s %s projection missing from %+v", side, diagramID, change.DiagramProjections)
}

func TestLegacySetUpDiagramsIsOneReviewedV2CandidateAndPreservesFacts(t *testing.T) {
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	db := openWebDatabaseAt(t, filepath.Join(dataDirectory, "workbraid.db"))
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	sourceID := uuid.NewString()
	targetID := uuid.NewString()
	manifest := []byte("format: workbraid-architecture\nversion: 1\nstore_id: \"" + storeID + "\"\nproject:\n  name: Legacy Project\n  source_hint: " + filepath.Clean(source) + "\n")
	sourceBytes := []byte("---\nid: \"" + sourceID + "\"\nrelationships:\n  - target: \"" + targetID + "\"\n    label: \"  calls α  \"\n---\n# Source\nExact source body.\n")
	targetBytes := []byte("---\nid: \"" + targetID + "\"\n---\n# Target\nExact target body.\n")
	v1 := advanceAcceptedToComponents(t, storePath, initialized.Revision, manifest, []testComponent{
		{path: "source.md", mode: "100755", source: sourceBytes},
		{path: "target.md", mode: "100644", source: targetBytes},
	})
	opened := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	if opened.FormatVersion != 1 || opened.Revision != v1 || len(opened.Components) != 2 {
		t.Fatalf("legacy open = %+v", opened)
	}
	rejected := postComponentMutation(t, handler, testOrigin, "/api/architecture/components/edit", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: v1, ComponentID: sourceID, Description: "No", DescriptionChanged: true,
	})
	if rejected.Code != http.StatusConflict || state.pending != nil {
		t.Fatalf("ordinary v1 mutation status=%d pending=%+v", rejected.Code, state.pending)
	}

	setup := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/diagrams/setup", source))
	if setup.Changes == nil || !setup.Changes.DiagramSetup || !setup.Changes.Valid || state.pending == nil || state.pending.diagramSetup == nil {
		t.Fatalf("setup pending = %+v", setup.Changes)
	}
	stableRoot := state.pending.diagramSetup.RootDiagramID
	reloaded := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	if reloaded.Changes == nil || state.pending.diagramSetup.RootDiagramID != stableRoot {
		t.Fatal("same-process reload changed setup identity")
	}
	reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	review := reviewed.Changes.Review
	if review == nil || review.Before.FormatVersion != 1 || review.WithChanges.FormatVersion != 2 || review.WithChanges.RootDiagramID != stableRoot {
		t.Fatalf("setup review = %+v", review)
	}
	if len(review.Comparison.Components) != 0 || len(review.Comparison.Relationships) != 0 || len(review.Comparison.Diagrams) != 1 || len(review.Comparison.Appearances) != 2 {
		t.Fatalf("setup comparison misclassified facts: %+v", review.Comparison)
	}
	for _, name := range []string{"source.md", "target.md"} {
		baseEntry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", v1, "components/"+name)
		candidateEntry := runGit(t, dataDirectory, "--git-dir", storePath, "ls-tree", review.CandidateTree, "components/"+name)
		if candidateEntry != baseEntry {
			t.Fatalf("setup rewrote %s\nbase: %s\ncandidate: %s", name, baseEntry, candidateEntry)
		}
	}
	if !strings.Contains(review.Diff, "version: 2") || !strings.Contains(review.Diff, "diagrams/root.yaml") {
		t.Fatalf("setup exact diff missing transition: %s", review.Diff)
	}
	accepted := decodeArchitectureResponse(t, postAcceptChanges(t, handler, testOrigin, source, *review))
	if accepted.FormatVersion != 2 || accepted.RootDiagramID != stableRoot || accepted.ComponentCount != 2 || accepted.Changes != nil {
		t.Fatalf("accepted setup = %+v", accepted)
	}
	root := diagramResponseByID(t, accepted.Diagrams, stableRoot)
	if !diagramHasComponent(root, sourceID, "home") || !diagramHasComponent(root, targetID, "home") || len(root.Relationships) != 1 || root.Relationships[0].Label != "  calls α  " {
		t.Fatalf("accepted setup lost composition/facts: %+v", root)
	}
}

func TestLegacyNonSetupPendingEvidenceIsReadOnlyDiscardOnly(t *testing.T) {
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	db := openWebDatabaseAt(t, filepath.Join(dataDirectory, "workbraid.db"))
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	pending := decodeArchitectureResponse(t, postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
		SourceRoot: filepath.Clean(source), ExpectedRevision: initialized.Revision, DiagramID: initialized.RootDiagramID,
		Title: "Pending evidence", Description: "Not canonical.\n",
	}))
	if pending.Changes == nil || state.pending == nil {
		t.Fatal("v2 pending evidence was not created")
	}
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	v1Manifest := []byte("format: workbraid-architecture\nversion: 1\nstore_id: \"" + storeID + "\"\nproject:\n  name: Legacy\n  source_hint: " + filepath.Clean(source) + "\n")
	v1 := advanceAcceptedToManifest(t, storePath, initialized.Revision, v1Manifest, nil)
	refreshedResponse := postArchitectureAction(t, handler, testOrigin, "/api/architecture/refresh", source)
	refreshed := decodeArchitectureResponse(t, refreshedResponse)
	if refreshedResponse.Code != http.StatusOK || refreshed.Revision != v1 || refreshed.FormatVersion != 1 || refreshed.Changes == nil || !refreshed.Changes.Stale || !refreshed.Changes.LegacyReadOnly {
		t.Fatalf("defensive legacy pending = status %d, %+v", refreshedResponse.Code, refreshed)
	}
	if response := postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source); response.Code != http.StatusConflict {
		t.Fatalf("legacy pending review status=%d body=%s", response.Code, response.Body.String())
	}
	if response := postArchitectureAction(t, handler, testOrigin, "/api/architecture/diagrams/setup", source); response.Code != http.StatusConflict {
		t.Fatalf("setup with legacy pending status=%d body=%s", response.Code, response.Body.String())
	}
	discarded := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/discard", source))
	if discarded.Changes != nil || discarded.Revision != v1 {
		t.Fatalf("discard legacy pending = %+v", discarded)
	}
	setup := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/diagrams/setup", source))
	if setup.Changes == nil || !setup.Changes.DiagramSetup {
		t.Fatalf("setup unavailable after discard: %+v", setup)
	}
}

func TestLegacyNonSetupPendingWithRetainedReviewCannotAdvertiseOrAccept(t *testing.T) {
	state, handler, source, dataDirectory, v1 := newLegacySetupFixture(t)
	decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/diagrams/setup", source))
	reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	if reviewed.Changes == nil || reviewed.Changes.Review == nil || state.pending == nil || state.pending.review == nil {
		t.Fatalf("setup review missing: response=%+v pending=%+v", reviewed.Changes, state.pending)
	}
	oldReview := *reviewed.Changes.Review

	// Reproduce the defensive alpha-transition boundary: an old non-setup v1
	// pending set survives in memory while its previously built review binding
	// also remains. Neither response projection nor confirmation may revive it.
	state.stateMutex.Lock()
	legacyChange := state.architecture.NewComponentChange(*state.loadedSnapshot, nil, "Legacy pending evidence", "Not canonical.\n")
	state.pending.diagramSetup = nil
	state.pending.newComponentHomes = nil
	state.pending.changes = append(state.pending.changes[:0], legacyChange)
	retainedPending := state.pending
	retainedReview := state.pending.review
	state.stateMutex.Unlock()

	current := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	if current.Revision != v1 || current.Stale || current.Changes == nil || !current.Changes.LegacyReadOnly || current.Changes.Review != nil || len(current.Changes.Components) != 1 {
		t.Fatalf("legacy retained-review projection = %+v", current)
	}
	if state.pending != retainedPending || state.pending.review != retainedReview {
		t.Fatalf("response projection mutated retained evidence: pending=%+v", state.pending)
	}

	storeID := associatedStoreID(t, state.db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	acceptedBefore := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted")
	objectsBefore := runGit(t, dataDirectory, "--git-dir", storePath, "count-objects", "-v")
	acceptedCASReached := false
	state.beforeAcceptedCAS = func(string) { acceptedCASReached = true }
	rejected := postAcceptChanges(t, handler, testOrigin, source, oldReview)
	var failure errorResponse
	if err := json.Unmarshal(rejected.Body.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if rejected.Code != http.StatusConflict || failure.Code != errorChangesUnavailable || acceptedCASReached {
		t.Fatalf("legacy retained review confirmation status=%d failure=%+v CAS=%t", rejected.Code, failure, acceptedCASReached)
	}
	acceptedAfter := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted")
	objectsAfter := runGit(t, dataDirectory, "--git-dir", storePath, "count-objects", "-v")
	if acceptedAfter != acceptedBefore || objectsAfter != objectsBefore || state.pending != retainedPending || state.pending.review != retainedReview {
		t.Fatalf("legacy rejection changed Git authority/objects or evidence: accepted=%q before=%q objects changed=%t pending=%+v", acceptedAfter, acceptedBefore, objectsAfter != objectsBefore, state.pending)
	}

	discarded := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/discard", source))
	if discarded.Revision != v1 || discarded.Changes != nil || state.pending != nil {
		t.Fatalf("legacy retained-review discard = %+v pending=%+v", discarded, state.pending)
	}
}

func TestSetUpDiagramsSerializesWithExistingWorkspaceTransitions(t *testing.T) {
	t.Run("late legacy mutation", func(t *testing.T) {
		state, handler, source, _, v1 := newLegacySetupFixture(t)
		setup, other := raceSetupAction(t, handler, source, func() *httptest.ResponseRecorder {
			return postComponentMutation(t, handler, testOrigin, "/api/architecture/components/add", componentMutationRequest{
				SourceRoot: filepath.Clean(source), ExpectedRevision: v1, Title: "Legacy write must not exist",
			})
		})
		if setup.Code != http.StatusOK || other.Code != http.StatusConflict || state.pending == nil || state.pending.diagramSetup == nil || len(state.pending.changes) != 0 {
			t.Fatalf("setup/mutation race mixed state: setup=%d mutation=%d pending=%+v", setup.Code, other.Code, state.pending)
		}
	})

	t.Run("project switch", func(t *testing.T) {
		state, handler, source, dataDirectory, _ := newLegacySetupFixture(t)
		otherSource := createSourceRepository(t)
		setupOther := NewHandler(state.db, testOrigin, t.TempDir(), dataDirectory)
		decodeArchitectureResponse(t, postInitializeProject(t, setupOther, testOrigin, otherSource))
		setup, other := raceSetupAction(t, handler, source, func() *httptest.ResponseRecorder {
			return postOpenProject(t, handler, testOrigin, otherSource)
		})
		setupWon := setup.Code == http.StatusOK && other.Code == http.StatusConflict
		switchWon := setup.Code == http.StatusConflict && other.Code == http.StatusOK
		if !setupWon && !switchWon {
			t.Fatalf("setup/project race outcomes: setup=%d project=%d", setup.Code, other.Code)
		}
		if setupWon && (state.loadedProject == nil || state.loadedProject.sourceRoot != filepath.Clean(source) || state.pending == nil || state.pending.diagramSetup == nil) {
			t.Fatalf("setup-first project race mixed state: project=%+v pending=%+v", state.loadedProject, state.pending)
		}
		if switchWon && (state.loadedProject == nil || state.loadedProject.sourceRoot != filepath.Clean(otherSource) || state.pending != nil) {
			t.Fatalf("switch-first setup race mixed state: project=%+v pending=%+v", state.loadedProject, state.pending)
		}
	})

	t.Run("Refresh", func(t *testing.T) {
		state, handler, source, dataDirectory, v1 := newLegacySetupFixture(t)
		storeID := associatedStoreID(t, state.db, filepath.Clean(source))
		storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
		externalV2 := advanceAcceptedToP21V2(t, storePath, v1, storeID, "Externally current")
		setup, refresh := raceSetupAction(t, handler, source, func() *httptest.ResponseRecorder {
			return postArchitectureAction(t, handler, testOrigin, "/api/architecture/refresh", source)
		})
		if refresh.Code != http.StatusOK || (setup.Code != http.StatusOK && setup.Code != http.StatusConflict) || state.loadedSnapshot == nil || state.loadedSnapshot.Revision() != externalV2 || state.loadedSnapshot.FormatVersion() != 2 {
			t.Fatalf("setup/Refresh race outcomes: setup=%d refresh=%d snapshot=%+v", setup.Code, refresh.Code, state.loadedSnapshot)
		}
		if state.pending != nil && (!state.pending.stale || state.pending.diagramSetup == nil || state.pending.baseRevision != v1) {
			t.Fatalf("setup/Refresh produced mixed pending state: %+v", state.pending)
		}
	})

	t.Run("Discard", func(t *testing.T) {
		state, handler, source, _, _ := newLegacySetupFixture(t)
		decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/diagrams/setup", source))
		setup, discard := raceSetupAction(t, handler, source, func() *httptest.ResponseRecorder {
			return postArchitectureAction(t, handler, testOrigin, "/api/architecture/discard", source)
		})
		if discard.Code != http.StatusOK || (setup.Code != http.StatusOK && setup.Code != http.StatusConflict) {
			t.Fatalf("setup/Discard race outcomes: setup=%d discard=%d", setup.Code, discard.Code)
		}
		if state.pending != nil && (state.pending.diagramSetup == nil || state.pending.stale || len(state.pending.changes) != 0) {
			t.Fatalf("setup/Discard produced mixed pending state: %+v", state.pending)
		}
	})

	t.Run("confirmation", func(t *testing.T) {
		state, handler, source, _, _ := newLegacySetupFixture(t)
		decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/diagrams/setup", source))
		reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
		setup, confirmation := raceSetupAction(t, handler, source, func() *httptest.ResponseRecorder {
			return postAcceptChanges(t, handler, testOrigin, source, *reviewed.Changes.Review)
		})
		if setup.Code != http.StatusConflict || confirmation.Code != http.StatusOK || state.pending != nil || state.loadedSnapshot == nil || state.loadedSnapshot.FormatVersion() != 2 {
			t.Fatalf("setup/confirmation race mixed state: setup=%d confirmation=%d pending=%+v snapshot=%+v", setup.Code, confirmation.Code, state.pending, state.loadedSnapshot)
		}
	})
}

func TestSetUpDiagramsPostCASPublicationFailureConsumesAndReloadsCanonicalSuccessor(t *testing.T) {
	state, handler, source, dataDirectory, v1 := newLegacySetupFixture(t)
	setup := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/diagrams/setup", source))
	if setup.Changes == nil || !setup.Changes.DiagramSetup {
		t.Fatalf("setup pending missing: %+v", setup)
	}
	reviewed := decodeArchitectureResponse(t, postArchitectureAction(t, handler, testOrigin, "/api/architecture/review", source))
	review := *reviewed.Changes.Review
	state.publicationFailure = func() error { return errors.New("focused setup publication failure") }
	response := postAcceptChanges(t, handler, testOrigin, source, review)
	updated := decodeArchitectureResponse(t, response)
	if response.Code != http.StatusInternalServerError || updated.ActionError != errorUpdatedReload || updated.Revision == v1 || updated.FormatVersion != 2 || updated.Changes != nil || state.pending != nil {
		t.Fatalf("setup post-CAS response status=%d body=%s pending=%+v", response.Code, response.Body.String(), state.pending)
	}
	storeID := associatedStoreID(t, state.db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	if accepted := runGit(t, dataDirectory, "--git-dir", storePath, "show-ref", "--verify", "--hash", "refs/heads/accepted"); accepted != updated.Revision {
		t.Fatalf("setup post-CAS accepted=%q response=%q", accepted, updated.Revision)
	}
	if duplicate := postAcceptChanges(t, handler, testOrigin, source, review); duplicate.Code != http.StatusConflict {
		t.Fatalf("setup post-CAS retry status=%d body=%s", duplicate.Code, duplicate.Body.String())
	}
	freshDB := openWebDatabaseAt(t, filepath.Join(dataDirectory, "workbraid.db"))
	fresh := NewHandler(freshDB, testOrigin, t.TempDir(), dataDirectory)
	reopened := decodeArchitectureResponse(t, postOpenProject(t, fresh, testOrigin, source))
	if reopened.Revision != updated.Revision || reopened.FormatVersion != 2 || reopened.RootDiagramID == "" || reopened.Changes != nil {
		t.Fatalf("fresh setup post-CAS reconstruction = %+v", reopened)
	}
}

func newLegacySetupFixture(t *testing.T) (*Handler, http.Handler, string, string, string) {
	t.Helper()
	source := createSourceRepository(t)
	dataDirectory := t.TempDir()
	db := openWebDatabaseAt(t, filepath.Join(dataDirectory, "workbraid.db"))
	state, handler := newHandler(db, testOrigin, t.TempDir(), dataDirectory)
	initialized := decodeArchitectureResponse(t, postInitializeProject(t, handler, testOrigin, source))
	storeID := associatedStoreID(t, db, filepath.Clean(source))
	storePath := filepath.Join(dataDirectory, "architecture", storeID+".git")
	manifest := []byte("format: workbraid-architecture\nversion: 1\nstore_id: \"" + storeID + "\"\nproject:\n  name: Legacy\n  source_hint: " + filepath.Clean(source) + "\n")
	v1 := advanceAcceptedToManifest(t, storePath, initialized.Revision, manifest, nil)
	opened := decodeArchitectureResponse(t, postOpenProject(t, handler, testOrigin, source))
	if opened.Revision != v1 || opened.FormatVersion != 1 {
		t.Fatalf("legacy setup fixture = %+v", opened)
	}
	return state, handler, source, dataDirectory, v1
}

func raceSetupAction(t *testing.T, handler http.Handler, source string, other func() *httptest.ResponseRecorder) (*httptest.ResponseRecorder, *httptest.ResponseRecorder) {
	t.Helper()
	start := make(chan struct{})
	setupDone := make(chan *httptest.ResponseRecorder, 1)
	otherDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		<-start
		setupDone <- postArchitectureAction(t, handler, testOrigin, "/api/architecture/diagrams/setup", source)
	}()
	go func() {
		<-start
		otherDone <- other()
	}()
	close(start)
	return <-setupDone, <-otherDone
}

func diagramHasComponent(diagram diagramResponse, componentID, role string) bool {
	for _, appearance := range diagram.Appearances {
		if appearance.ComponentID == componentID && appearance.Role == role {
			return true
		}
	}
	return false
}
