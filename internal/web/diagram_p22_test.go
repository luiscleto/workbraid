package web

import (
	"net/http"
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

func diagramHasComponent(diagram diagramResponse, componentID, role string) bool {
	for _, appearance := range diagram.Appearances {
		if appearance.ComponentID == componentID && appearance.Role == role {
			return true
		}
	}
	return false
}
