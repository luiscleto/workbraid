package architecture

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestDescriptionPreservesExactHeadingBoundary(t *testing.T) {
	for _, heading := range []struct{ name, source string }{
		{"ATX", "# **API**"}, {"Setext", "**API**\n======"},
		{"terminated ATX", "# **API**\n"}, {"terminated Setext", "**API**\n======\n"},
		{"CRLF ATX", "# **API**\r\n"}, {"CRLF Setext", "**API**\r\n======\r\n"},
		{"unterminated CRLF Setext", "**API**\r\n======"},
	} {
		for _, body := range []struct{ name, source string }{
			{"leading newline", "\nBody\n"}, {"plain", "Body\n"},
			{"whitespace", " \t"}, {"newlines", "\n\n"}, {"empty", ""},
			{"CRLF body", "\r\n  Body\r\n"},
		} {
			t.Run(heading.name+"/"+body.name, func(t *testing.T) {
				prefix := "---\nid: \"" + uuid.NewString() + "\"\n---\n\n"
				source := []byte(prefix + heading.source)
				base, err := parseComponent("components/api.md", source)
				if err != nil {
					t.Fatal(err)
				}
				base.source = source
				// Even an unchanged submitted Title with a changed flag must
				// preserve its existing Markdown H1 spelling.
				change := ComponentChange{ID: base.id.String(), Title: "API", TitleChanged: true, Description: body.source, DescriptionChanged: true}
				got, err := editedComponentSource(base, change)
				if err != nil {
					t.Fatal(err)
				}
				separator := ""
				if body.source != "" && heading.source[len(heading.source)-1] != '\n' {
					separator = "\n"
				}
				want := prefix + heading.source + separator + body.source
				if string(got) != want {
					t.Fatalf("source=%q, want %q", got, want)
				}
				parsed, err := parseComponent("components/api.md", got)
				if err != nil {
					t.Fatal(err)
				}
				if parsed.title != "API" || string(parsed.body) != body.source {
					t.Fatalf("Title=%q Description=%q, want API and %q", parsed.title, parsed.body, body.source)
				}
			})
		}
	}
}

func TestDescriptionResidualRoundTripAgainstUnterminatedAcceptedH1(t *testing.T) {
	ctx := context.Background()
	data := t.TempDir()
	manager := NewManager(data)
	storeID := uuid.NewString()
	bootstrap, err := manager.InitializeOrLoad(ctx, storeID, "Description boundary", "description-boundary")
	if err != nil {
		t.Fatal(err)
	}
	created := manager.NewComponentChange(bootstrap, nil, "API", "")
	initial, err := manager.prepareTestCandidate(ctx, bootstrap, []ComponentChange{created}, rootHomes(bootstrap, created))
	if err != nil {
		t.Fatal(err)
	}
	baseRevision, err := manager.CreateSuccessor(ctx, bootstrap, initial)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.AdvanceAccepted(ctx, bootstrap, baseRevision); err != nil {
		t.Fatal(err)
	}
	base, err := manager.LoadAccepted(ctx, storeID)
	if err != nil {
		t.Fatal(err)
	}
	change, _ := base.ChangeForAcceptedComponent(created.ID)
	change.Description, change.DescriptionChanged = "\nBody\n", true
	proposed, err := manager.ConstructCandidate(ctx, base, []ComponentChange{change}, CandidateComposition{})
	if err != nil {
		t.Fatal(err)
	}
	proposalID, name, err := manager.NewChangeSet(nil, "Document API")
	if err != nil {
		t.Fatal(err)
	}
	record := ChangeSet{ID: proposalID, Name: name, Lifecycle: "active", BaseRevision: baseRevision, BaseSnapshot: base,
		Generation: 1, Proposal: "Document the API.\n", Changes: []ComponentChange{change}, Candidate: &proposed}
	state, err := manager.WriteActiveChangeSet(ctx, storeID, record, "")
	if err != nil {
		t.Fatal(err)
	}
	storePath := mustStorePath(t, manager, storeID)
	baseSource := gitBytes(t, "--git-dir", storePath, "show", baseRevision+":"+created.Path)
	if !bytes.HasSuffix(baseSource, []byte("# API\n")) {
		t.Fatalf("B source=%q", baseSource)
	}
	// Only this external Accepted edit uses Git source writes. B and P were
	// created with ordinary typed facts and the production constructor.
	acceptedSource := bytes.TrimSuffix(baseSource, []byte("\n"))
	componentTree := mktree(t, storePath, "100644 blob "+writeTestBlob(t, storePath, acceptedSource)+"\t"+filepath.Base(created.Path)+"\n")
	manifest := gitText(t, "--git-dir", storePath, "rev-parse", baseRevision+":architecture.yaml")
	diagrams := gitText(t, "--git-dir", storePath, "rev-parse", baseRevision+":diagrams")
	acceptedTree := mktree(t, storePath, "100644 blob "+manifest+"\tarchitecture.yaml\n040000 tree "+componentTree+"\tcomponents\n040000 tree "+diagrams+"\tdiagrams\n")
	acceptedRevision := gitText(t, "--git-dir", storePath, "commit-tree", acceptedTree, "-p", baseRevision, "-m", "External heading terminator")
	gitText(t, "--git-dir", storePath, "update-ref", acceptedRef, acceptedRevision, baseRevision)
	accepted, err := NewManager(data).LoadAccepted(ctx, storeID)
	if err != nil {
		t.Fatal(err)
	}
	if got := accepted.AuthoringComponents()[0]; got.Title != "API" || got.Description != "" {
		t.Fatalf("A=%+v", got)
	}
	loaded, unavailable, err := NewManager(data).LoadChangeSets(ctx, storeID)
	if err != nil || len(unavailable) != 0 || len(loaded) != 1 || loaded[0].RefObject != state || loaded[0].Candidate.Tree() != proposed.Tree() {
		t.Fatalf("original P/S reconstruction: %v %v", err, unavailable)
	}
	refs := gitText(t, "--git-dir", storePath, "show-ref")
	residual, _ := accepted.ChangeForAcceptedComponent(created.ID)
	residual.Description, residual.DescriptionChanged = change.Description, true
	result, err := manager.ConstructCandidate(ctx, accepted, []ComponentChange{residual}, CandidateComposition{})
	if err != nil {
		t.Fatal(err)
	}
	actual := result.Snapshot().AuthoringComponents()[0]
	if actual.Title != residual.Title || actual.Description != residual.Description {
		t.Fatalf("residual round trip: %+v", actual)
	}
	source := gitBytes(t, "--git-dir", storePath, "show", result.Tree()+":"+created.Path)
	if want := string(acceptedSource) + "\n" + residual.Description; string(source) != want {
		t.Fatalf("source=%q want=%q", source, want)
	}
	if result.Tree() != proposed.Tree() {
		t.Fatal("same exact final Architecture did not yield the P tree")
	}
	reconciled, err := manager.Reconcile(ctx, base, accepted, proposed, nil)
	if err != nil || reconciled.Status != "ready" || reconciled.Candidate == nil || reconciled.Candidate.Tree() != result.Tree() {
		t.Fatalf("ordinary exact Description residual through reconciliation: %+v %v", reconciled, err)
	}
	if gitText(t, "--git-dir", storePath, "show-ref") != refs {
		t.Fatal("construction changed refs")
	}
	// Prove these ordinary residual facts survive the actual envelope writer
	// and a fresh Manager, without exercising a second constructor.
	record.BaseRevision, record.BaseSnapshot = acceptedRevision, accepted
	record.Generation++
	record.Changes, record.Candidate = []ComponentChange{residual}, &result
	newState, err := manager.WriteActiveChangeSet(ctx, storeID, record, state)
	if err != nil {
		t.Fatal(err)
	}
	loaded, unavailable, err = NewManager(data).LoadChangeSets(ctx, storeID)
	if err != nil || len(unavailable) != 0 || len(loaded) != 1 {
		t.Fatalf("residual restart: %v %v", err, unavailable)
	}
	if loaded[0].RefObject != newState || loaded[0].Candidate.Tree() != result.Tree() || loaded[0].Changes[0].Description != change.Description {
		t.Fatal("residual state/tree/Description changed on restart")
	}
	if gitText(t, "--git-dir", storePath, "rev-parse", acceptedRef) != acceptedRevision {
		t.Fatal("durable write changed Accepted")
	}
}
