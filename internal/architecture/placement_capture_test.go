package architecture

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestPlacementHistoricalReconstruction(t *testing.T) {
	raw, err := os.ReadFile("testdata/placement-history.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Base    string
		StoreID string
		Refs    map[string]string
		Objects map[string]struct {
			Type string
			Data []byte
		}
	}
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	data := t.TempDir()
	manager := NewManager(data)
	path, err := manager.StorePath(fixture.StoreID)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
	gitText(t, "init", "--bare", path)
	for id, obj := range fixture.Objects {
		output, err := runGit(ctx, obj.Data, "--git-dir", path, "hash-object", "-w", "-t", obj.Type, "--stdin")
		if err != nil || strings.TrimSpace(string(output)) != id {
			t.Fatalf("restore %s: %s %v", id, output, err)
		}
	}
	for ref, id := range fixture.Refs {
		gitText(t, "--git-dir", path, "update-ref", ref, id)
	}
	records, bad, err := manager.LoadChangeSets(ctx, fixture.StoreID)
	if err != nil || len(bad) > 0 || len(records) != 2 {
		t.Fatalf("records %d bad %+v err %v", len(records), bad, err)
	}
	for _, record := range records {
		// Generic read-only reports must preserve the same supported legacy
		// operational bytes and exact trees before any new Review preparation.
		beforeRefs := gitText(t, "--git-dir", path, "show-ref")
		_, oldSide, newSide, comparisonDiff, comparisonErr := manager.CompareVersions(ctx, fixture.StoreID,
			VersionSelector{Kind: "proposal", ChangeSetID: record.ID, State: record.RefObject, Side: "base"},
			VersionSelector{Kind: "proposal", ChangeSetID: record.ID, State: record.RefObject, Side: "candidate"})
		expectedDiff, diffErr := manager.CandidateDiff(ctx, record.BaseSnapshot, *record.Candidate)
		if comparisonErr != nil || diffErr != nil || oldSide.Info.Revision != record.BaseRevision || newSide.Info.Revision != record.Candidate.Tree() || comparisonDiff != string(expectedDiff) || gitText(t, "--git-dir", path, "show-ref") != beforeRefs {
			t.Fatalf("legacy comparison changed exact history: %v / %v", comparisonErr, diffErr)
		}
		reconstructed, err := manager.ConstructCandidate(ctx, record.BaseSnapshot, record.Changes, record.Composition)
		if err != nil || reconstructed.Tree() != record.Candidate.Tree() {
			t.Fatalf("historical tree %s: %v", record.ID, err)
		}
		if record.RefObject != fixture.Refs[activeChangeSetPrefix+record.ID] {
			t.Fatal("historical ref changed")
		}
		// Removing then preparing the exact same review binding exercises an
		// old operational record's first preparation without converting its facts.
		beforeBlob := gitText(t, "--git-dir", path, "rev-parse", record.RefObject+":changes.yaml")
		unreviewed := record
		unreviewed.Review = nil
		object, err := manager.WriteActiveChangeSet(ctx, fixture.StoreID, unreviewed, record.RefObject)
		if err != nil {
			t.Fatal(err)
		}
		unreviewed.RefObject = object
		unreviewed.Review = record.Review
		object, err = manager.WriteActiveChangeSet(ctx, fixture.StoreID, unreviewed, object)
		if err != nil {
			t.Fatal(err)
		}
		if gitText(t, "--git-dir", path, "rev-parse", object+":changes.yaml") != beforeBlob || gitText(t, "--git-dir", path, "rev-parse", object+":architecture") != record.Candidate.Tree() {
			t.Fatal("review preparation converted historical state or candidate")
		}
		// Discard active reachability; immutable review parents must suffice.
		if err = manager.DeleteActiveChangeSet(ctx, fixture.StoreID, record.ID, object); err != nil {
			t.Fatal(err)
		}
	}
	gitText(t, "--git-dir", path, "gc", "--prune=now")
	restarted := NewManager(data)
	for _, record := range records {
		reviews, bad, err := restarted.LoadReviews(ctx, fixture.StoreID, record.ID)
		if err != nil || len(bad) > 0 || len(reviews) != 1 {
			t.Fatalf("historical review %+v %v", bad, err)
		}
		if reviews[0].ReviewedState != record.RefObject || reviews[0].ReviewedChange.Candidate.Tree() != record.Candidate.Tree() {
			t.Fatal("historical review retargeted")
		}
	}
}
