package architecture

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestReviewSubmissionPersistsExactParentAndSurvivesDiscardAndGC(t *testing.T) {
	ctx := context.Background()
	dataDirectory := t.TempDir()
	manager := NewManager(dataDirectory)
	base, record, object, storePath := changeSetFixture(t, manager, ctx, "Review persistence")
	record.Proposal = "# Proposal\n"
	record.Review = &ChangeSetReview{BaseRevision: base.Revision(), CandidateTree: record.Candidate.Tree(), Generation: record.Generation}
	reviewedState, err := manager.WriteActiveChangeSet(ctx, base.StoreID(), record, object)
	if err != nil {
		t.Fatal(err)
	}
	componentID := record.Changes[0].ID
	review, err := manager.SubmitReview(ctx, base.StoreID(), ReviewSubmissionInput{
		ChangeSetID: record.ID, ReviewedState: reviewedState,
		Binding: ReviewBinding{BaseRevision: base.Revision(), CandidateTree: record.Candidate.Tree(), Generation: 1},
		Verdict: "request_changes", Author: "Reviewer agent", Body: "Overall exact λ\n", SubmittedAt: time.Date(2026, 9, 4, 14, 30, 0, 0, time.UTC),
		Comments: []ReviewCommentInput{
			{Body: "Whole proposal", Anchor: ReviewAnchor{Kind: "proposal"}},
			{Body: "Proposal line", Anchor: ReviewAnchor{Kind: "proposal_markdown", StartLine: 1, EndLine: 1}},
			{Body: "Component", Anchor: ReviewAnchor{Kind: "component", Side: "with_changes", ComponentID: componentID}},
			{Body: "Markdown", Anchor: ReviewAnchor{Kind: "component_markdown", Side: "with_changes", ComponentID: componentID, StartLine: 1, EndLine: 2}},
			{Body: "Diagram", Anchor: ReviewAnchor{Kind: "diagram", Side: "with_changes", DiagramID: base.RootDiagramID()}},
			{Body: "Home", Anchor: ReviewAnchor{Kind: "composition", Side: "with_changes", DiagramID: base.RootDiagramID(), ComponentID: componentID, Aspect: "home"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if review.ReviewedState != reviewedState || review.RefObject == "" {
		t.Fatalf("review = %+v", review)
	}
	if parent := gitText(t, "--git-dir", storePath, "rev-parse", review.RefObject+"^"); parent != reviewedState {
		t.Fatalf("parent = %s", parent)
	}
	entries := gitText(t, "--git-dir", storePath, "ls-tree", review.RefObject)
	if !strings.Contains(entries, "100644 blob ") || !strings.Contains(entries, "\tbody.md") || !strings.Contains(entries, "040000 tree ") || !strings.Contains(entries, "\tcomments") || !strings.Contains(entries, "\treview.yaml") {
		t.Fatalf("review entries = %q", entries)
	}
	if err := manager.DeleteActiveChangeSet(ctx, base.StoreID(), record.ID, reviewedState); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("git", "--git-dir", storePath, "gc", "--prune=now")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git gc: %v: %s", err, output)
	}
	loaded, unavailable, err := NewManager(dataDirectory).LoadReviews(ctx, base.StoreID(), record.ID)
	if err != nil || len(unavailable) != 0 || len(loaded) != 1 {
		t.Fatalf("loaded=%+v unavailable=%+v err=%v", loaded, unavailable, err)
	}
	if loaded[0].ReviewedChange.RefObject != reviewedState || loaded[0].Body != "Overall exact λ\n" || len(loaded[0].Comments) != 6 {
		t.Fatalf("reloaded = %+v", loaded[0])
	}
}

func TestReviewSubmissionPresenceRulesAndAtomicStateVerification(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(t.TempDir())
	base, record, object, storePath := changeSetFixture(t, manager, ctx, "Review rules")
	record.Review = &ChangeSetReview{BaseRevision: base.Revision(), CandidateTree: record.Candidate.Tree(), Generation: record.Generation}
	reviewedState, err := manager.WriteActiveChangeSet(ctx, base.StoreID(), record, object)
	if err != nil {
		t.Fatal(err)
	}
	input := ReviewSubmissionInput{ChangeSetID: record.ID, ReviewedState: reviewedState, Binding: ReviewBinding{BaseRevision: base.Revision(), CandidateTree: record.Candidate.Tree(), Generation: 1}, Author: "Reviewer"}
	input.Verdict = "comment"
	if _, err := manager.SubmitReview(ctx, base.StoreID(), input); !errors.Is(err, ErrReviewRequestInvalid) {
		t.Fatalf("empty comment error = %v", err)
	}
	if refs := gitText(t, "--git-dir", storePath, "for-each-ref", "--format=%(refname)", reviewsRefPrefix); refs != "" {
		t.Fatalf("empty comment wrote refs: %s", refs)
	}
	input.Verdict = "approve"
	approve, err := manager.SubmitReview(ctx, base.StoreID(), input)
	if err != nil {
		t.Fatalf("verdict-only approve: %v", err)
	}
	input.Verdict = "request_changes"
	requestChanges, err := manager.SubmitReview(ctx, base.StoreID(), input)
	if err != nil {
		t.Fatalf("verdict-only request changes: %v", err)
	}
	if approve.RefObject == requestChanges.RefObject || gitText(t, "--git-dir", storePath, "rev-parse", approve.RefObject+"^") != reviewedState || gitText(t, "--git-dir", storePath, "rev-parse", requestChanges.RefObject+"^") != reviewedState {
		t.Fatalf("independent submissions did not retain the same exact reviewed parent: approve=%+v request=%+v", approve, requestChanges)
	}

	// Advancing the exact active ref makes the earlier reviewed_state fail the
	// final transaction verification even though all of its objects still exist.
	record.Generation++
	record.Review = nil
	next, err := manager.WriteActiveChangeSet(ctx, base.StoreID(), record, reviewedState)
	if err != nil {
		t.Fatal(err)
	}
	input.Verdict, input.Body = "comment", "late"
	if _, err := manager.SubmitReview(ctx, base.StoreID(), input); !errors.Is(err, ErrReviewInvalidated) {
		t.Fatalf("late submit error = %v", err)
	}
	refs := gitText(t, "--git-dir", storePath, "for-each-ref", "--format=%(refname)", reviewsRefPrefix)
	if len(strings.Fields(refs)) != 2 || next == reviewedState {
		t.Fatalf("refs=%q next=%s", refs, next)
	}
}

func TestReviewSubmissionValidatesEveryExactAnchorKind(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(t.TempDir())
	storeID := uuid.NewString()
	bootstrap, err := manager.InitializeOrLoad(ctx, storeID, "Project", "project")
	if err != nil {
		t.Fatal(err)
	}
	storePath, _ := manager.StorePath(storeID)
	ids := diagramFixtureIDs{root: uuid.NewString(), detail: uuid.NewString(), empty: uuid.NewString(), gateway: uuid.NewString(), worker: uuid.NewString(), records: uuid.NewString(), ledger: uuid.NewString()}
	accepted := commitV2Fixture(t, storePath, storeID, ids)
	gitText(t, "--git-dir", storePath, "update-ref", acceptedRef, accepted, bootstrap.Revision())
	base, err := manager.LoadAccepted(ctx, storeID)
	if err != nil {
		t.Fatal(err)
	}
	workerChange, ok := base.ChangeForAcceptedComponent(ids.worker)
	if !ok {
		t.Fatal("worker fixture is missing")
	}
	workerChange.Description = "Revised worker docs.\n"
	workerChange.DescriptionChanged = true
	workerChange.Relationships = append([]AuthoringRelationship(nil), workerChange.Relationships[1:]...)
	workerChange.RelationshipsChanged = true
	newComponent := manager.NewComponentChange(base, []ComponentChange{workerChange}, "Candidate only", "Candidate docs.\n")
	newComponent.Relationships = []AuthoringRelationship{{TargetID: ids.gateway, Label: "observes"}}
	newComponent.RelationshipsChanged = true
	changes := []ComponentChange{workerChange, newComponent}
	composition := CandidateComposition{NewComponentHomes: []NewComponentHome{{ComponentID: newComponent.ID, DiagramID: ids.root}}}
	candidate, err := manager.ConstructCandidate(ctx, base, changes, composition)
	if err != nil {
		t.Fatal(err)
	}
	changeSetID, name, _ := manager.NewChangeSet(nil, "Exact anchors")
	record := ChangeSet{ID: changeSetID, Name: name, Lifecycle: "active", BaseRevision: base.Revision(), Proposal: "first\r\nsecond\n", BaseSnapshot: base, Changes: changes, Composition: composition, Candidate: &candidate,
		Review: &ChangeSetReview{BaseRevision: base.Revision(), CandidateTree: candidate.Tree()}}
	reviewedState, err := manager.WriteActiveChangeSet(ctx, storeID, record, "")
	if err != nil {
		t.Fatal(err)
	}
	binding := ReviewBinding{BaseRevision: base.Revision(), CandidateTree: candidate.Tree()}
	comments := []ReviewCommentInput{
		{Body: "proposal", Anchor: ReviewAnchor{Kind: "proposal"}},
		{Body: "proposal range", Anchor: ReviewAnchor{Kind: "proposal_markdown", StartLine: 1, EndLine: 2}},
		{Body: "component", Anchor: ReviewAnchor{Kind: "component", Side: "before", ComponentID: ids.worker}},
		{Body: "component source", Anchor: ReviewAnchor{Kind: "component_markdown", Side: "with_changes", ComponentID: ids.worker, StartLine: 1, EndLine: 2}},
		{Body: "candidate component", Anchor: ReviewAnchor{Kind: "component", Side: "with_changes", ComponentID: newComponent.ID}},
		{Body: "candidate source", Anchor: ReviewAnchor{Kind: "component_markdown", Side: "with_changes", ComponentID: newComponent.ID, StartLine: 1, EndLine: 2}},
		{Body: "diagram", Anchor: ReviewAnchor{Kind: "diagram", Side: "before", DiagramID: ids.root}},
		{Body: "home", Anchor: ReviewAnchor{Kind: "composition", Side: "with_changes", DiagramID: ids.root, ComponentID: ids.gateway, Aspect: "home"}},
		{Body: "reference", Anchor: ReviewAnchor{Kind: "composition", Side: "before", DiagramID: ids.root, ComponentID: ids.worker, Aspect: "reference"}},
		{Body: "detail", Anchor: ReviewAnchor{Kind: "composition", Side: "with_changes", DiagramID: ids.root, ComponentID: ids.gateway, Aspect: "detail", DetailDiagramID: ids.detail}},
		{Body: "relationship", Anchor: ReviewAnchor{Kind: "relationship", Side: "before", SourceComponentID: ids.worker, TargetComponentID: ids.records, Label: "calls\nnext", Occurrence: 2}},
	}
	created, err := manager.SubmitReview(ctx, storeID, ReviewSubmissionInput{ChangeSetID: changeSetID, ReviewedState: reviewedState, Binding: binding, Verdict: "comment", Author: "Reviewer", Comments: comments})
	if err != nil || len(created.Comments) != len(comments) {
		t.Fatalf("created=%+v err=%v", created, err)
	}
	loaded, unavailable, err := manager.LoadReviews(ctx, storeID, changeSetID)
	if err != nil || len(unavailable) != 0 || len(loaded) != 1 || len(loaded[0].Comments) != len(comments) {
		t.Fatalf("loaded=%+v unavailable=%+v err=%v", loaded, unavailable, err)
	}
	for index := range comments {
		if loaded[0].Comments[index].Body != comments[index].Body || loaded[0].Comments[index].Anchor != comments[index].Anchor {
			t.Fatalf("comment %d changed: got=%+v want=%+v", index, loaded[0].Comments[index], comments[index])
		}
	}

	invalid := []ReviewAnchor{
		{Kind: "proposal_markdown", StartLine: 1, EndLine: 3},
		{Kind: "component", Side: "before", ComponentID: ids.worker, StartLine: 1},
		{Kind: "component_markdown", Side: "with_changes", ComponentID: ids.worker, StartLine: 0, EndLine: 1},
		{Kind: "component", Side: "before", ComponentID: newComponent.ID},
		{Kind: "diagram", Side: "before", DiagramID: uuid.NewString()},
		{Kind: "composition", Side: "before", DiagramID: ids.root, ComponentID: ids.worker, Aspect: "home"},
		{Kind: "composition", Side: "before", DiagramID: ids.root, ComponentID: ids.gateway, Aspect: "detail", DetailDiagramID: ids.empty},
		{Kind: "relationship", Side: "before", SourceComponentID: ids.worker, TargetComponentID: ids.records, Label: "calls\nnext", Occurrence: 3},
		{Kind: "relationship", Side: "with_changes", SourceComponentID: ids.worker, TargetComponentID: ids.records, Label: "calls\nnext", Occurrence: 2},
	}
	for index, anchor := range invalid {
		_, err := manager.SubmitReview(ctx, storeID, ReviewSubmissionInput{ChangeSetID: changeSetID, ReviewedState: reviewedState, Binding: binding, Verdict: "comment", Author: "Reviewer", Comments: []ReviewCommentInput{{Body: "invalid", Anchor: anchor}}})
		if !errors.Is(err, ErrReviewAnchorInvalid) {
			t.Fatalf("invalid anchor %d error=%v", index, err)
		}
		var location *ReviewAnchorValidationError
		if !errors.As(err, &location) || location.CommentIndex != 1 || location.Reason == "" {
			t.Fatalf("invalid anchor %d location=%+v", index, location)
		}
	}
	if refs := strings.Fields(gitText(t, "--git-dir", storePath, "for-each-ref", "--format=%(refname)", reviewsRefPrefix)); len(refs) != 1 {
		t.Fatalf("invalid anchors wrote review refs: %v", refs)
	}
}

func TestReviewEnumerationOwnsOnlyItsNamespaceAndIsolatesMalformedRecords(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(t.TempDir())
	base, record, _, storePath := changeSetFixture(t, manager, ctx, "Review namespace")
	otherRef := "refs/workbraid/other-test/" + uuid.NewString()
	gitText(t, "--git-dir", storePath, "update-ref", otherRef, base.Revision())
	badReviewID := uuid.NewString()
	badRef := reviewRef(record.ID, badReviewID)
	gitText(t, "--git-dir", storePath, "update-ref", badRef, base.Revision())

	valid, unavailable, err := manager.LoadReviews(ctx, base.StoreID(), "")
	if err != nil || len(valid) != 0 || len(unavailable) != 1 || unavailable[0].ReviewID != badReviewID {
		t.Fatalf("valid=%+v unavailable=%+v err=%v", valid, unavailable, err)
	}
	for _, item := range unavailable {
		if strings.Contains(item.Ref, "other-test") {
			t.Fatalf("Reviews interpreted unrelated namespace: %+v", item)
		}
	}
}

func TestReviewLoadingIsolatesClosedSchemaParentAndTreeFailures(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(t.TempDir())
	base, record, object, storePath := changeSetFixture(t, manager, ctx, "Review isolation")
	record.Review = &ChangeSetReview{BaseRevision: base.Revision(), CandidateTree: record.Candidate.Tree(), Generation: record.Generation}
	reviewedState, err := manager.WriteActiveChangeSet(ctx, base.StoreID(), record, object)
	if err != nil {
		t.Fatal(err)
	}
	valid, err := manager.SubmitReview(ctx, base.StoreID(), ReviewSubmissionInput{
		ChangeSetID: record.ID, ReviewedState: reviewedState,
		Binding: ReviewBinding{BaseRevision: base.Revision(), CandidateTree: record.Candidate.Tree(), Generation: record.Generation},
		Verdict: "approve", Author: "Reviewer",
	})
	if err != nil {
		t.Fatal(err)
	}
	bodyBlob, err := manager.git.writeBlob(ctx, storePath, nil)
	if err != nil {
		t.Fatal(err)
	}
	metadata := func(id string, version int, reviewed string) string {
		return fmt.Sprintf("format: workbraid-review\nversion: %d\nid: %s\nchange_set_id: %s\nreviewed_state: %s\nbinding:\n  base_revision: %s\n  candidate_tree: %s\n  generation: %d\nverdict: approve\nauthor: Reviewer\nsubmitted_at: \"2026-09-04T14:30:00Z\"\ncomments: []\n", version, id, record.ID, reviewed, base.Revision(), record.Candidate.Tree(), record.Generation)
	}
	writeRecord := func(id string, version int, parent string, unknownPath bool) {
		metadataBlob, writeErr := manager.git.writeBlob(ctx, storePath, []byte(metadata(id, version, parent)))
		if writeErr != nil {
			t.Fatal(writeErr)
		}
		treeInput := fmt.Sprintf("100644 blob %s\tbody.md\n100644 blob %s\treview.yaml\n", bodyBlob, metadataBlob)
		if unknownPath {
			treeInput += fmt.Sprintf("100644 blob %s\tunexpected.txt\n", bodyBlob)
		}
		tree, treeErr := manager.git.makeTree(ctx, storePath, []byte(treeInput))
		if treeErr != nil {
			t.Fatal(treeErr)
		}
		commit, commitErr := manager.git.makeReviewCommit(ctx, storePath, tree, parent)
		if commitErr != nil {
			t.Fatal(commitErr)
		}
		gitText(t, "--git-dir", storePath, "update-ref", reviewRef(record.ID, id), commit)
	}
	writeRecord(uuid.NewString(), 1, reviewedState, true)
	writeRecord(uuid.NewString(), 2, reviewedState, false)
	writeRecord(uuid.NewString(), 1, base.Revision(), false)

	loaded, unavailable, err := manager.LoadReviews(ctx, base.StoreID(), record.ID)
	if err != nil || len(loaded) != 1 || loaded[0].ID != valid.ID || len(unavailable) != 3 {
		t.Fatalf("loaded=%+v unavailable=%+v err=%v", loaded, unavailable, err)
	}
}

func TestReviewLineRangesUseExactLFBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		start, end int
		valid      bool
	}{
		{name: "empty has no lines", source: "", start: 1, end: 1},
		{name: "one line", source: "one", start: 1, end: 1, valid: true},
		{name: "final LF adds no line", source: "one\n", start: 2, end: 2},
		{name: "CR remains in first line", source: "one\r\ntwo\n", start: 1, end: 2, valid: true},
		{name: "inclusive multiline range", source: "one\ntwo\nthree", start: 2, end: 3, valid: true},
		{name: "start is one based", source: "one", start: 0, end: 1},
		{name: "end precedes start", source: "one\ntwo", start: 2, end: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validLineRange([]byte(test.source), test.start, test.end); got != test.valid {
				t.Fatalf("validLineRange(%q, %d, %d) = %t, want %t", test.source, test.start, test.end, got, test.valid)
			}
		})
	}
}
