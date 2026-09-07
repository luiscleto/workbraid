package architecture

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestComparisonRetentionLifetimesAndRestart(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	m := NewManager(dir)
	base, record, state, path := changeSetFixture(t, m, ctx, "Retained comparison")
	a := VersionSelector{Kind: "accepted", Revision: base.Revision()}
	p := VersionSelector{Kind: "proposal", ChangeSetID: record.ID, State: state, Side: "candidate"}
	refs := func() string { return gitText(t, "--git-dir", path, "show-ref") }
	initial := refs()
	_, before, after, diff, err := m.CompareVersions(ctx, base.StoreID(), a, p)
	if err != nil || diff == "" || before.Info.Document != nil || after.Info.Document == nil || after.Info.Revision != record.Candidate.Tree() {
		t.Fatalf("unprepared comparison: %v %+v %+v", err, before.Info, after.Info)
	}
	if refs() != initial {
		t.Fatal("read changed refs")
	}
	record.Proposal = "# Exact reviewed context\n"
	record.Review = &ChangeSetReview{BaseRevision: base.Revision(), CandidateTree: record.Candidate.Tree(), Generation: record.Generation}
	s, err := m.WriteActiveChangeSet(ctx, base.StoreID(), record, state)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err = m.CompareVersions(ctx, base.StoreID(), a, p); !errors.Is(err, ErrVersionMoved) {
		t.Fatalf("old active state: %v", err)
	}
	review, err := m.SubmitReview(ctx, base.StoreID(), ReviewSubmissionInput{ChangeSetID: record.ID, ReviewedState: s, Binding: ReviewBinding{BaseRevision: base.Revision(), CandidateTree: record.Candidate.Tree(), Generation: record.Generation}, Verdict: "approve", Author: "Reviewer"})
	if err != nil {
		t.Fatal(err)
	}
	r := VersionSelector{Kind: "submitted_review", ChangeSetID: record.ID, State: s, ReviewID: review.ID, Side: "candidate"}
	successor, err := m.CreateSuccessor(ctx, base, *record.Candidate)
	if err != nil {
		t.Fatal(err)
	}
	applied := record
	applied.Lifecycle = "applied"
	applied.AppliedRevision = successor
	token, err := m.PrepareAppliedChangeSet(ctx, base.StoreID(), applied)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.AcceptChangeSet(ctx, base.StoreID(), base.Revision(), successor, record.ID, s, token); err != nil {
		t.Fatal(err)
	}
	receiptSelector := VersionSelector{Kind: "applied", ChangeSetID: record.ID, State: token, Side: "candidate"}
	initial = refs()
	_, historical, receipt, equal, err := NewManager(dir).CompareVersions(ctx, base.StoreID(), r, receiptSelector)
	if err != nil || equal != "" || historical.Info.Revision != receipt.Info.Revision || *historical.Info.Document != record.Proposal {
		t.Fatalf("retained review/receipt: %v %s", err, equal)
	}
	if refs() != initial {
		t.Fatal("restart read wrote refs")
	}
	// External rewind removes ancestry eligibility but cannot invalidate the
	// separately retained applied receipt and submitted-review parent.
	gitText(t, "--git-dir", path, "update-ref", acceptedRef, base.Revision(), successor)
	if _, _, _, _, err = m.CompareVersions(ctx, base.StoreID(), a, VersionSelector{Kind: "accepted", Revision: successor}); !errors.Is(err, ErrVersionUnavailable) {
		t.Fatalf("unretained Accepted selector: %v", err)
	}
	if _, _, _, _, err = m.CompareVersions(ctx, base.StoreID(), r, receiptSelector); err != nil {
		t.Fatal(err)
	}
	other, err := m.CreateProject(ctx, "Other store")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err = m.CompareVersions(ctx, other.StoreID(), a, a); !errors.Is(err, ErrVersionUnavailable) {
		t.Fatalf("cross-store object: %v", err)
	}
}

func TestVersionCatalogBoundsMalformedCursorAndDiscardDiscovery(t *testing.T) {
	ctx := context.Background()
	m := NewManager(t.TempDir())
	base, record, state, path := changeSetFixture(t, m, ctx, "Discarded review context")
	record.Review = &ChangeSetReview{BaseRevision: base.Revision(), CandidateTree: record.Candidate.Tree(), Generation: record.Generation}
	s, err := m.WriteActiveChangeSet(ctx, base.StoreID(), record, state)
	if err != nil {
		t.Fatal(err)
	}
	review, err := m.SubmitReview(ctx, base.StoreID(), ReviewSubmissionInput{ChangeSetID: record.ID, ReviewedState: s, Binding: ReviewBinding{BaseRevision: base.Revision(), CandidateTree: record.Candidate.Tree(), Generation: record.Generation}, Verdict: "approve", Author: "Review owner"})
	if err != nil {
		t.Fatal(err)
	}
	if err = m.DeleteActiveChangeSet(ctx, base.StoreID(), record.ID, s); err != nil {
		t.Fatal(err)
	}
	gitText(t, "--git-dir", path, "gc", "--prune=now")
	groups, err := m.VersionCatalog(ctx, VersionPageRequest{StoreID: base.StoreID(), Source: "review_proposals", Limit: 1})
	if err != nil || len(groups.ReviewProposals) != 1 || groups.ReviewProposals[0].Label != record.Name {
		t.Fatalf("discarded discovery: %+v %v", groups, err)
	}
	if _, err = m.VersionCatalog(ctx, VersionPageRequest{StoreID: base.StoreID(), Source: "submitted_review"}); !errors.Is(err, ErrInvalid) {
		t.Fatal("unscoped review page accepted")
	}
	page, err := m.VersionCatalog(ctx, VersionPageRequest{StoreID: base.StoreID(), Source: "submitted_review", ChangeSetID: record.ID, Limit: 1})
	if err != nil || len(page.Versions) != 2 || page.Versions[1].Selector.ReviewID != review.ID {
		t.Fatalf("scoped page %+v %v", page, err)
	}
	// A long history is paged before loading. A malformed ancestor remains an
	// unavailable row; it is never silently replaced by current Architecture.
	parent := base.Revision()
	for i := 0; i < 12; i++ {
		parent = gitText(t, "--git-dir", path, "commit-tree", record.Candidate.Tree(), "-p", parent)
	}
	gitText(t, "--git-dir", path, "update-ref", acceptedRef, parent, base.Revision())
	request := VersionPageRequest{StoreID: base.StoreID(), Source: "accepted", Limit: 3}
	page, err = m.VersionCatalog(ctx, request)
	if err != nil || len(page.Versions) != 3 || page.NextCursor == "" {
		t.Fatalf("page %+v %v", page, err)
	}
	seen := map[string]bool{}
	for _, v := range page.Versions {
		seen[v.Selector.Revision] = true
	}
	request.Cursor = page.NextCursor
	next, err := m.VersionCatalog(ctx, request)
	if err != nil || len(next.Versions) != 3 {
		t.Fatal(err)
	}
	for _, v := range next.Versions {
		if seen[v.Selector.Revision] {
			t.Fatal("repeated history entry")
		}
	}
	data, _ := base64.RawURLEncoding.DecodeString(request.Cursor)
	for _, bad := range [][]byte{append(append([]byte{}, data...), []byte(` {}`)...), []byte(strings.Replace(string(data), `"source":"accepted"`, `"source":"accepted","source":"accepted"`, 1)), []byte(`{"source":"accepted"}`)} {
		request.Cursor = base64.RawURLEncoding.EncodeToString(bad)
		if _, err = m.VersionCatalog(ctx, request); !errors.Is(err, ErrInvalid) {
			t.Fatalf("malformed cursor accepted: %s / %v", bad, err)
		}
	}
	request.Cursor = page.NextCursor
	gitText(t, "--git-dir", path, "update-ref", acceptedRef, base.Revision(), parent)
	if _, err = m.VersionCatalog(ctx, request); !errors.Is(err, ErrVersionMoved) {
		t.Fatalf("moved page: %v", err)
	}
	// Namespace paging bounds reconstruction even with unavailable records.
	for i := 0; i < 8; i++ {
		gitText(t, "--git-dir", path, "update-ref", activeChangeSetPrefix+uuid.NewString(), base.Revision())
	}
	badPage, err := m.VersionCatalog(ctx, VersionPageRequest{StoreID: base.StoreID(), Source: "proposal", Limit: 2})
	if err != nil || len(badPage.Versions) != 2 || badPage.NextCursor == "" || !badPage.Versions[0].Unavailable {
		t.Fatalf("unavailable page %+v %v", badPage, err)
	}
}

func TestVersionSelectorsAreClosedAndRefsAreRechecked(t *testing.T) {
	valid := `{"kind":"accepted","revision":"` + strings.Repeat("a", 40) + `"}`
	for _, bad := range []string{strings.Replace(valid, `"kind":"accepted"`, `"kind":"accepted","kind":"accepted"`, 1), strings.Replace(valid, `"kind":"accepted"`, `"kind":"accepted","state":""`, 1), strings.Replace(valid, `"kind":"accepted"`, `"kind":"accepted","side":"candidate"`, 1), strings.Replace(valid, `"kind":"accepted"`, `"kind":"accepted","extra":1`, 1), `null`} {
		var s VersionSelector
		if json.Unmarshal([]byte(bad), &s) == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	ctx := context.Background()
	m := NewManager(t.TempDir())
	base, record, state, path := changeSetFixture(t, m, ctx, "Recheck")
	selector := VersionSelector{Kind: "proposal", ChangeSetID: record.ID, State: state, Side: "candidate"}
	resolved, err := m.retainedVersion(ctx, base, selector)
	if err != nil {
		t.Fatal(err)
	}
	record.Proposal = "Changed"
	if _, err = m.WriteActiveChangeSet(ctx, base.StoreID(), record, state); err != nil {
		t.Fatal(err)
	}
	if err = m.verifyVersionRefs(ctx, path, resolved.refs); !errors.Is(err, ErrVersionMoved) {
		t.Fatal("contributing ref change not detected")
	}
	if reflect.DeepEqual(resolved.Info.Selector, VersionSelector{}) {
		t.Fatal("missing provenance")
	}
}

func TestAcceptedMergeAncestryLoadsLegacyAndExternalAbsence(t *testing.T) {
	ctx := context.Background()
	m := NewManager(t.TempDir())
	base, err := m.CreateProject(ctx, "External history")
	if err != nil {
		t.Fatal(err)
	}
	path, _ := m.StorePath(base.StoreID())
	ids := diagramFixtureIDs{root: uuid.NewString(), detail: uuid.NewString(), empty: uuid.NewString(), gateway: uuid.NewString(), worker: uuid.NewString(), records: uuid.NewString(), ledger: uuid.NewString()}
	legacy := commitV2Fixture(t, path, base.StoreID(), ids)
	tree := gitText(t, "--git-dir", path, "rev-parse", base.Revision()+"^{tree}")
	// A valid externally authored merge retains the old graph but its result
	// removes those objects. Both exact sides remain legitimate report inputs.
	merge := gitText(t, "--git-dir", path, "commit-tree", tree, "-p", base.Revision(), "-p", legacy)
	gitText(t, "--git-dir", path, "update-ref", acceptedRef, merge, base.Revision())
	refs := gitText(t, "--git-dir", path, "show-ref")
	_, a, b, diff, err := m.CompareVersions(ctx, base.StoreID(), VersionSelector{Kind: "accepted", Revision: legacy}, VersionSelector{Kind: "accepted", Revision: merge})
	if err != nil || a.Snapshot.FormatVersion() != 2 || a.Snapshot.ComponentCount() == 0 || b.Snapshot.ComponentCount() != 0 || diff == "" || a.Info.Document != nil || b.Info.Document != nil {
		t.Fatalf("external absence comparison: %v", err)
	}
	page, err := m.VersionCatalog(ctx, VersionPageRequest{StoreID: base.StoreID(), Source: "accepted"})
	if err != nil || len(page.Versions) != 3 {
		t.Fatalf("merge discovery %+v %v", page, err)
	}
	if gitText(t, "--git-dir", path, "show-ref") != refs {
		t.Fatal("historical read changed refs")
	}
}
