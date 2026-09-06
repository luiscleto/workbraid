package architecture

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestChangeSetsPersistValidAndInvalidTypedStateAcrossRestart(t *testing.T) {
	ctx := context.Background()
	data := t.TempDir()
	manager := NewManager(data)
	base, err := manager.CreateProject(ctx, "Change Sets")
	if err != nil {
		t.Fatal(err)
	}

	emptyCandidate, err := manager.ConstructCandidate(ctx, base, nil, CandidateComposition{})
	if err != nil {
		t.Fatal(err)
	}
	emptyID, emptyName, err := manager.NewChangeSet(nil, "Quiet Harbor")
	if err != nil {
		t.Fatal(err)
	}
	empty := ChangeSet{ID: emptyID, Name: emptyName, Lifecycle: "active", BaseRevision: base.Revision(), Proposal: "# Exact proposal\n\nKeep [this](https://example.invalid).\n", BaseSnapshot: base, Candidate: &emptyCandidate}
	emptyObject, err := manager.WriteActiveChangeSet(ctx, base.StoreID(), empty, "")
	if err != nil {
		t.Fatal(err)
	}

	brokenID, brokenName, err := manager.NewChangeSet([]ChangeSet{empty}, "Broken relationship")
	if err != nil {
		t.Fatal(err)
	}
	component := manager.NewComponentChange(base, nil, "Worker", "Body\n")
	component.RelationshipsChanged = true
	component.Relationships = []AuthoringRelationship{{TargetID: "not-a-uuid", Label: ""}}
	composition := CandidateComposition{NewComponentHomes: []NewComponentHome{{ComponentID: component.ID, DiagramID: base.RootDiagramID()}}}
	if _, err := manager.ConstructCandidate(ctx, base, []ComponentChange{component}, composition); !errors.Is(err, ErrRelationshipLabelRequired) {
		t.Fatalf("invalid candidate error = %v", err)
	}
	broken := ChangeSet{ID: brokenID, Name: brokenName, Lifecycle: "active", BaseRevision: base.Revision(), Generation: 1, Proposal: "repair me\n", BaseSnapshot: base, Changes: []ComponentChange{component}, Composition: composition}
	if _, err := manager.WriteActiveChangeSet(ctx, base.StoreID(), broken, ""); err != nil {
		t.Fatal(err)
	}

	restarted := NewManager(data)
	records, unavailable, err := restarted.LoadChangeSets(ctx, base.StoreID())
	if err != nil || len(unavailable) != 0 || len(records) != 2 {
		t.Fatalf("records=%+v unavailable=%+v err=%v", records, unavailable, err)
	}
	byID := map[string]ChangeSet{records[0].ID: records[0], records[1].ID: records[1]}
	if got := byID[emptyID]; got.RefObject != emptyObject || got.Candidate == nil || got.Candidate.Tree() != emptyCandidate.Tree() || got.Proposal != empty.Proposal {
		t.Fatalf("empty record = %+v", got)
	}
	gotBroken := byID[brokenID]
	if gotBroken.Candidate != nil || len(gotBroken.Changes) != 1 || len(gotBroken.Changes[0].Relationships) != 1 {
		t.Fatalf("broken record = %+v", gotBroken)
	}
	row := gotBroken.Changes[0].Relationships[0]
	if row.TargetID != "not-a-uuid" || row.Label != "" {
		t.Fatalf("raw invalid relationship changed: %+v", row)
	}

	storePath, _ := manager.StorePath(base.StoreID())
	paths := gitText(t, "--git-dir", storePath, "ls-tree", "--name-only", emptyObject)
	if paths != "architecture\nchange-set.yaml\nchanges.yaml\nproposal.md" {
		t.Fatalf("valid envelope paths = %q", paths)
	}
	brokenObject := gitText(t, "--git-dir", storePath, "show-ref", "--verify", "--hash", activeChangeSetPrefix+brokenID)
	if paths := gitText(t, "--git-dir", storePath, "ls-tree", "--name-only", brokenObject); paths != "change-set.yaml\nchanges.yaml\nproposal.md" {
		t.Fatalf("invalid envelope paths = %q", paths)
	}
	if parent := gitText(t, "--git-dir", storePath, "rev-list", "--parents", "-n", "1", brokenObject); parent != brokenObject+" "+base.Revision() {
		t.Fatalf("state parent = %q", parent)
	}
	if nested := gitText(t, "--git-dir", storePath, "rev-parse", emptyObject+":architecture"); nested != emptyCandidate.Tree() {
		t.Fatalf("nested Architecture tree=%s want=%s", nested, emptyCandidate.Tree())
	}
	if proposal := string(gitBytes(t, "--git-dir", storePath, "show", emptyObject+":proposal.md")); proposal != empty.Proposal {
		t.Fatalf("proposal bytes=%q want=%q", proposal, empty.Proposal)
	}
	for _, entry := range strings.Split(gitText(t, "--git-dir", storePath, "ls-tree", emptyObject), "\n") {
		if strings.HasSuffix(entry, "\tarchitecture") {
			if !strings.HasPrefix(entry, "040000 tree ") {
				t.Fatalf("Architecture envelope mode=%q", entry)
			}
		} else if !strings.HasPrefix(entry, "100644 blob ") {
			t.Fatalf("envelope file mode=%q", entry)
		}
	}
}

func TestChangeSetAcceptanceUsesOneThreeRefTransactionAndLeavesOtherActive(t *testing.T) {
	ctx := context.Background()
	data := t.TempDir()
	manager := NewManager(data)
	base, err := manager.CreateProject(ctx, "Atomic")
	if err != nil {
		t.Fatal(err)
	}
	makeRecord := func(name string) ChangeSet {
		change := manager.NewComponentChange(base, nil, name, name+" body\n")
		change.RelationshipsChanged = true
		composition := CandidateComposition{NewComponentHomes: []NewComponentHome{{ComponentID: change.ID, DiagramID: base.RootDiagramID()}}}
		candidate, candidateErr := manager.PrepareCandidate(ctx, base, []ComponentChange{change}, &composition)
		if candidateErr != nil {
			t.Fatal(candidateErr)
		}
		id, generatedName, nameErr := manager.NewChangeSet(nil, name)
		if nameErr != nil {
			t.Fatal(nameErr)
		}
		return ChangeSet{ID: id, Name: generatedName, Lifecycle: "active", BaseRevision: base.Revision(), Generation: 1, BaseSnapshot: base, Changes: []ComponentChange{change}, Composition: composition, Candidate: &candidate, Review: &ChangeSetReview{BaseRevision: base.Revision(), CandidateTree: candidate.Tree(), Generation: 1}}
	}
	a := makeRecord("Change A")
	b := makeRecord("Change B")
	aObject, err := manager.WriteActiveChangeSet(ctx, base.StoreID(), a, "")
	if err != nil {
		t.Fatal(err)
	}
	bObject, err := manager.WriteActiveChangeSet(ctx, base.StoreID(), b, "")
	if err != nil {
		t.Fatal(err)
	}
	successor, err := manager.CreateSuccessor(ctx, base, *a.Candidate)
	if err != nil {
		t.Fatal(err)
	}
	applied := a
	applied.Lifecycle = "applied"
	applied.AppliedRevision = successor
	appliedObject, err := manager.PrepareAppliedChangeSet(ctx, base.StoreID(), applied)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.AcceptChangeSet(ctx, base.StoreID(), base.Revision(), successor, a.ID, aObject, appliedObject); err != nil {
		t.Fatal(err)
	}

	storePath, _ := manager.StorePath(base.StoreID())
	refs := gitText(t, "--git-dir", storePath, "for-each-ref", "--format=%(refname) %(objectname)")
	for _, exact := range []string{
		"refs/heads/accepted " + successor,
		appliedChangeSetPrefix + a.ID + " " + appliedObject,
		activeChangeSetPrefix + b.ID + " " + bObject,
	} {
		if !strings.Contains(refs, exact) {
			t.Fatalf("refs missing %q:\n%s", exact, refs)
		}
	}
	if strings.Contains(refs, activeChangeSetPrefix+a.ID) {
		t.Fatalf("accepted active ref remains:\n%s", refs)
	}
	records, unavailable, err := NewManager(data).LoadChangeSets(ctx, base.StoreID())
	if err != nil || len(unavailable) != 0 || len(records) != 2 {
		t.Fatalf("records=%+v unavailable=%+v err=%v", records, unavailable, err)
	}
}

func TestChangeSetLoaderIgnoresOtherNamespaceAndReportsOwnedMalformedRecord(t *testing.T) {
	ctx := context.Background()
	data := t.TempDir()
	manager := NewManager(data)
	base, err := manager.CreateProject(ctx, "Isolation")
	if err != nil {
		t.Fatal(err)
	}
	storePath, _ := manager.StorePath(base.StoreID())
	gitText(t, "--git-dir", storePath, "update-ref", "refs/workbraid/other-test/kept", base.Revision())
	gitText(t, "--git-dir", storePath, "update-ref", activeChangeSetPrefix+"not-a-uuid", base.Revision())
	records, unavailable, err := manager.LoadChangeSets(ctx, base.StoreID())
	if err != nil || len(records) != 0 || len(unavailable) != 1 || unavailable[0].ID != "not-a-uuid" {
		t.Fatalf("records=%+v unavailable=%+v err=%v", records, unavailable, err)
	}
}

func TestChangeSetInvalidRelationshipVariantsRoundTripExactFacts(t *testing.T) {
	cases := []struct {
		name   string
		target string
		label  string
	}{
		{name: "empty target", target: "", label: "calls"},
		{name: "malformed target", target: "not-a-uuid", label: "calls"},
		{name: "unresolved target", target: "99999999-9999-4999-8999-999999999999", label: "calls"},
		{name: "blank label", target: "self", label: " \t "},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			data := t.TempDir()
			manager := NewManager(data)
			base, err := manager.CreateProject(ctx, "Invalid relationship")
			if err != nil {
				t.Fatal(err)
			}
			component := manager.NewComponentChange(base, nil, "Worker", "Body\n")
			target := test.target
			if target == "self" {
				target = component.ID
			}
			component.RelationshipsChanged = true
			component.Relationships = []AuthoringRelationship{{TargetID: target, Label: test.label}}
			composition := CandidateComposition{NewComponentHomes: []NewComponentHome{{ComponentID: component.ID, DiagramID: base.RootDiagramID()}}}
			if _, err := manager.ConstructCandidate(ctx, base, []ComponentChange{component}, composition); err == nil {
				t.Fatal("invalid relationship unexpectedly constructed")
			}
			id, name, err := manager.NewChangeSet(nil, test.name)
			if err != nil {
				t.Fatal(err)
			}
			record := ChangeSet{ID: id, Name: name, Lifecycle: "active", BaseRevision: base.Revision(), Generation: 1, BaseSnapshot: base, Changes: []ComponentChange{component}, Composition: composition}
			if _, err := manager.WriteActiveChangeSet(ctx, base.StoreID(), record, ""); err != nil {
				t.Fatal(err)
			}
			loaded, unavailable, err := NewManager(data).LoadChangeSets(ctx, base.StoreID())
			if err != nil || len(unavailable) != 0 || len(loaded) != 1 || loaded[0].Candidate != nil || loaded[0].ValidationError == nil {
				t.Fatalf("loaded=%+v unavailable=%+v err=%v", loaded, unavailable, err)
			}
			row := loaded[0].Changes[0].Relationships[0]
			if row.TargetID != target || row.Label != test.label {
				t.Fatalf("raw row changed: got=%+v want target=%q label=%q", row, target, test.label)
			}
		})
	}
}

func TestChangeSetLoaderIsolatesClosedSchemaAndIdentityConflicts(t *testing.T) {
	t.Run("malformed envelopes remain unavailable while Accepted loads", func(t *testing.T) {
		cases := []struct {
			name   string
			mutate func(*testing.T, *Manager, string, Snapshot, map[string]treeEntry)
		}{
			{name: "unknown envelope path", mutate: func(t *testing.T, manager *Manager, storePath string, _ Snapshot, entries map[string]treeEntry) {
				blob, err := manager.git.writeBlob(context.Background(), storePath, []byte("unexpected"))
				if err != nil {
					t.Fatal(err)
				}
				entries["extra.txt"] = treeEntry{Mode: "100644", Type: "blob", Object: blob, Path: "extra.txt"}
			}},
			{name: "unknown changes field", mutate: func(t *testing.T, manager *Manager, storePath string, _ Snapshot, entries map[string]treeEntry) {
				contents, err := manager.git.readBlob(context.Background(), storePath, entries["changes.yaml"].Object)
				if err != nil {
					t.Fatal(err)
				}
				blob, err := manager.git.writeBlob(context.Background(), storePath, append(contents, []byte("unknown: true\n")...))
				if err != nil {
					t.Fatal(err)
				}
				entry := entries["changes.yaml"]
				entry.Object = blob
				entries["changes.yaml"] = entry
			}},
			{name: "missing required changes field", mutate: func(t *testing.T, manager *Manager, storePath string, _ Snapshot, entries map[string]treeEntry) {
				contents, err := manager.git.readBlob(context.Background(), storePath, entries["changes.yaml"].Object)
				if err != nil {
					t.Fatal(err)
				}
				blob, err := manager.git.writeBlob(context.Background(), storePath, []byte(strings.Replace(string(contents), "references: []\n", "", 1)))
				if err != nil {
					t.Fatal(err)
				}
				entry := entries["changes.yaml"]
				entry.Object = blob
				entries["changes.yaml"] = entry
			}},
			{name: "wrong changes scalar type", mutate: func(t *testing.T, manager *Manager, storePath string, _ Snapshot, entries map[string]treeEntry) {
				contents, err := manager.git.readBlob(context.Background(), storePath, entries["changes.yaml"].Object)
				if err != nil {
					t.Fatal(err)
				}
				blob, err := manager.git.writeBlob(context.Background(), storePath, []byte(strings.Replace(string(contents), "\nversion: 3\n", "\nversion: wrong\n", 1)))
				if err != nil {
					t.Fatal(err)
				}
				entry := entries["changes.yaml"]
				entry.Object = blob
				entries["changes.yaml"] = entry
			}},
			{name: "duplicate logical changes", mutate: func(t *testing.T, manager *Manager, storePath string, base Snapshot, entries map[string]treeEntry) {
				contents, err := manager.git.readBlob(context.Background(), storePath, entries["changes.yaml"].Object)
				if err != nil {
					t.Fatal(err)
				}
				replacement := fmt.Sprintf("diagram_titles:\n  - diagram_id: %s\n    title: First\n  - diagram_id: %s\n    title: Second", base.RootDiagramID(), base.RootDiagramID())
				blob, err := manager.git.writeBlob(context.Background(), storePath, []byte(strings.Replace(string(contents), "diagram_titles: []", replacement, 1)))
				if err != nil {
					t.Fatal(err)
				}
				entry := entries["changes.yaml"]
				entry.Object = blob
				entries["changes.yaml"] = entry
			}},
			{name: "non-repairable candidate mismatch", mutate: func(t *testing.T, manager *Manager, storePath string, _ Snapshot, entries map[string]treeEntry) {
				contents, err := manager.git.readBlob(context.Background(), storePath, entries["changes.yaml"].Object)
				if err != nil {
					t.Fatal(err)
				}
				blob, err := manager.git.writeBlob(context.Background(), storePath, []byte(strings.Replace(string(contents), "path: components/worker.md", "path: components/other.md", 1)))
				if err != nil {
					t.Fatal(err)
				}
				entry := entries["changes.yaml"]
				entry.Object = blob
				entries["changes.yaml"] = entry
				delete(entries, "architecture")
			}},
			{name: "review mismatch", mutate: func(t *testing.T, manager *Manager, storePath string, base Snapshot, entries map[string]treeEntry) {
				contents, err := manager.git.readBlob(context.Background(), storePath, entries["change-set.yaml"].Object)
				if err != nil {
					t.Fatal(err)
				}
				metadata, err := parseChangeSetMetadata(contents)
				if err != nil {
					t.Fatal(err)
				}
				metadata.Review = &changeSetReview{BaseRevision: base.Revision(), CandidateTree: base.Revision(), Generation: metadata.Generation}
				encoded, err := yaml.Marshal(metadata)
				if err != nil {
					t.Fatal(err)
				}
				blob, err := manager.git.writeBlob(context.Background(), storePath, encoded)
				if err != nil {
					t.Fatal(err)
				}
				entry := entries["change-set.yaml"]
				entry.Object = blob
				entries["change-set.yaml"] = entry
			}},
			{name: "candidate reconstruction mismatch", mutate: func(t *testing.T, manager *Manager, storePath string, base Snapshot, entries map[string]treeEntry) {
				baseTree, err := manager.git.commitTree(context.Background(), storePath, base.Revision())
				if err != nil {
					t.Fatal(err)
				}
				entry := entries["architecture"]
				entry.Object = baseTree
				entries["architecture"] = entry
			}},
		}
		for _, test := range cases {
			t.Run(test.name, func(t *testing.T) {
				ctx := context.Background()
				data := t.TempDir()
				manager := NewManager(data)
				base, record, object, storePath := changeSetFixture(t, manager, ctx, "Closed schema")
				entries, err := manager.git.directTreeEntries(ctx, storePath, object)
				if err != nil {
					t.Fatal(err)
				}
				byPath := make(map[string]treeEntry, len(entries))
				for _, entry := range entries {
					byPath[entry.Path] = entry
				}
				test.mutate(t, manager, storePath, base, byPath)
				replacement := replaceChangeSetEnvelope(t, manager, ctx, storePath, base.Revision(), byPath)
				if err := manager.git.updateRef(ctx, storePath, activeChangeSetPrefix+record.ID, replacement, object); err != nil {
					t.Fatal(err)
				}
				loaded, unavailable, err := manager.LoadChangeSets(ctx, base.StoreID())
				if err != nil || len(loaded) != 0 || len(unavailable) != 1 {
					t.Fatalf("loaded=%+v unavailable=%+v err=%v", loaded, unavailable, err)
				}
				if unavailable[0].Name != record.Name {
					t.Fatalf("unavailable readable name=%q want=%q", unavailable[0].Name, record.Name)
				}
				if accepted, err := manager.LoadAccepted(ctx, base.StoreID()); err != nil || accepted.Revision() != base.Revision() {
					t.Fatalf("Accepted harmed by malformed proposal: revision=%q err=%v", accepted.Revision(), err)
				}
			})
		}
	})

	t.Run("malformed metadata retains its readable active name for collision detection", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManager(t.TempDir())
		base, first, object, storePath := changeSetFixture(t, manager, ctx, "Reserved Name")
		entries, err := manager.git.directTreeEntries(ctx, storePath, object)
		if err != nil {
			t.Fatal(err)
		}
		byPath := make(map[string]treeEntry, len(entries))
		for _, entry := range entries {
			byPath[entry.Path] = entry
		}
		metadata, err := manager.git.readBlob(ctx, storePath, byPath["change-set.yaml"].Object)
		if err != nil {
			t.Fatal(err)
		}
		metadataBlob, err := manager.git.writeBlob(ctx, storePath, []byte(strings.Replace(string(metadata), "version: 1", "version: invalid", 1)))
		if err != nil {
			t.Fatal(err)
		}
		metadataEntry := byPath["change-set.yaml"]
		metadataEntry.Object = metadataBlob
		byPath["change-set.yaml"] = metadataEntry
		malformed := replaceChangeSetEnvelope(t, manager, ctx, storePath, base.Revision(), byPath)
		if err := manager.git.updateRef(ctx, storePath, activeChangeSetPrefix+first.ID, malformed, object); err != nil {
			t.Fatal(err)
		}

		secondID, secondName, err := manager.NewChangeSet(nil, "reserved name")
		if err != nil {
			t.Fatal(err)
		}
		candidate, err := manager.ConstructCandidate(ctx, base, nil, CandidateComposition{})
		if err != nil {
			t.Fatal(err)
		}
		second := ChangeSet{ID: secondID, Name: secondName, Lifecycle: "active", BaseRevision: base.Revision(), BaseSnapshot: base, Candidate: &candidate}
		if _, err := manager.WriteActiveChangeSet(ctx, base.StoreID(), second, ""); err != nil {
			t.Fatal(err)
		}

		loaded, unavailable, err := manager.LoadChangeSets(ctx, base.StoreID())
		if err != nil || len(loaded) != 0 || len(unavailable) != 2 {
			t.Fatalf("loaded=%+v unavailable=%+v err=%v", loaded, unavailable, err)
		}
		for _, record := range unavailable {
			if !strings.EqualFold(record.Name, "Reserved Name") {
				t.Fatalf("unavailable record lost reserved name: %+v", record)
			}
		}
	})

	t.Run("invalid parent retains its readable active name", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManager(t.TempDir())
		base, record, object, storePath := changeSetFixture(t, manager, ctx, "Invalid Parent")
		tree, err := manager.git.commitTree(ctx, storePath, object)
		if err != nil {
			t.Fatal(err)
		}
		invalidParent, err := manager.git.makeBootstrapCommit(ctx, storePath, tree)
		if err != nil {
			t.Fatal(err)
		}
		if err := manager.git.updateRef(ctx, storePath, activeChangeSetPrefix+record.ID, invalidParent, object); err != nil {
			t.Fatal(err)
		}

		loaded, unavailable, err := manager.LoadChangeSets(ctx, base.StoreID())
		if err != nil || len(loaded) != 0 || len(unavailable) != 1 {
			t.Fatalf("loaded=%+v unavailable=%+v err=%v", loaded, unavailable, err)
		}
		if unavailable[0].Name != record.Name || unavailable[0].Lifecycle != "active" || unavailable[0].Reason != "state commit must have exactly one parent" {
			t.Fatalf("invalid-parent record = %+v", unavailable[0])
		}
		if _, _, err := manager.NewChangeSet([]ChangeSet{{Name: unavailable[0].Name, Lifecycle: unavailable[0].Lifecycle}}, "invalid parent"); err == nil {
			t.Fatal("readable unavailable active name did not reserve creation")
		}
	})

	t.Run("duplicate lifecycle UUID is never selected", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManager(t.TempDir())
		base, record, object, storePath := changeSetFixture(t, manager, ctx, "Duplicate lifecycle")
		if err := manager.git.createRef(ctx, storePath, appliedChangeSetPrefix+record.ID, object); err != nil {
			t.Fatal(err)
		}
		loaded, unavailable, err := manager.LoadChangeSets(ctx, base.StoreID())
		if err != nil || len(loaded) != 0 || len(unavailable) != 2 {
			t.Fatalf("loaded=%+v unavailable=%+v err=%v", loaded, unavailable, err)
		}
		var reserved []ChangeSet
		for _, unavailableRecord := range unavailable {
			if unavailableRecord.Reason != "change-set identity occurs in more than one lifecycle" {
				t.Fatalf("duplicate-lifecycle record = %+v", unavailableRecord)
			}
			if unavailableRecord.Lifecycle == "active" {
				if unavailableRecord.Name != record.Name {
					t.Fatalf("active duplicate lost readable name: %+v", unavailableRecord)
				}
				reserved = append(reserved, ChangeSet{Name: unavailableRecord.Name, Lifecycle: unavailableRecord.Lifecycle})
			}
		}
		if _, _, err := manager.NewChangeSet(reserved, "duplicate LIFECYCLE"); err == nil {
			t.Fatal("readable duplicate-lifecycle active name did not reserve creation")
		}
	})

	t.Run("case-folding active name conflict marks every winner unavailable", func(t *testing.T) {
		ctx := context.Background()
		manager := NewManager(t.TempDir())
		base, first, _, _ := changeSetFixture(t, manager, ctx, "Folded Name")
		candidate, err := manager.ConstructCandidate(ctx, base, nil, CandidateComposition{})
		if err != nil {
			t.Fatal(err)
		}
		secondID, _, err := manager.NewChangeSet(nil, "folded name")
		if err != nil {
			t.Fatal(err)
		}
		second := ChangeSet{ID: secondID, Name: "folded name", Lifecycle: "active", BaseRevision: base.Revision(), BaseSnapshot: base, Candidate: &candidate}
		if _, err := manager.WriteActiveChangeSet(ctx, base.StoreID(), second, ""); err != nil {
			t.Fatal(err)
		}
		loaded, unavailable, err := manager.LoadChangeSets(ctx, base.StoreID())
		if err != nil || len(loaded) != 0 || len(unavailable) != 2 || first.Name != "Folded Name" {
			t.Fatalf("loaded=%+v unavailable=%+v err=%v", loaded, unavailable, err)
		}
	})
}

func TestChangeSetNamesGenerationsDeletionAndFailedAcceptance(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(t.TempDir())
	base, record, firstObject, storePath := changeSetFixture(t, manager, ctx, "Lifecycle")

	record.Generation = 2
	record.Proposal = "second generation\n"
	secondObject, err := manager.WriteActiveChangeSet(ctx, base.StoreID(), record, firstObject)
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range []string{firstObject, secondObject} {
		parent := gitText(t, "--git-dir", storePath, "rev-list", "--parents", "-n", "1", object)
		if parent != object+" "+base.Revision() {
			t.Fatalf("state generation chained: %q", parent)
		}
	}

	successor, err := manager.CreateSuccessor(ctx, base, *record.Candidate)
	if err != nil {
		t.Fatal(err)
	}
	applied := record
	applied.Lifecycle = "applied"
	applied.AppliedRevision = successor
	applied.Review = &ChangeSetReview{BaseRevision: base.Revision(), CandidateTree: record.Candidate.Tree(), Generation: record.Generation}
	appliedObject, err := manager.PrepareAppliedChangeSet(ctx, base.StoreID(), applied)
	if err != nil {
		t.Fatal(err)
	}
	acceptedBefore := gitText(t, "--git-dir", storePath, "rev-parse", acceptedRef)
	if err := manager.AcceptChangeSet(ctx, base.StoreID(), base.Revision(), successor, record.ID, strings.Repeat("0", 40), appliedObject); err == nil {
		t.Fatal("acceptance with wrong active CAS unexpectedly succeeded")
	}
	if got := gitText(t, "--git-dir", storePath, "rev-parse", acceptedRef); got != acceptedBefore {
		t.Fatalf("failed transaction advanced Accepted: %s", got)
	}
	if got := gitText(t, "--git-dir", storePath, "rev-parse", activeChangeSetPrefix+record.ID); got != secondObject {
		t.Fatalf("failed transaction changed active ref: %s", got)
	}
	if output := gitText(t, "--git-dir", storePath, "for-each-ref", "--format=%(refname)", appliedChangeSetPrefix+record.ID); output != "" {
		t.Fatalf("failed transaction published applied ref: %s", output)
	}

	if err := manager.DeleteActiveChangeSet(ctx, base.StoreID(), record.ID, secondObject); err != nil {
		t.Fatal(err)
	}
	if output := gitText(t, "--git-dir", storePath, "for-each-ref", "--format=%(refname)", activeChangeSetPrefix); output != "" {
		t.Fatalf("deleted change set remains referenced: %s", output)
	}
	if output := gitText(t, "--git-dir", storePath, "for-each-ref", "--contains="+firstObject, "--format=%(refname)", "refs/workbraid/change-sets/"); output != "" {
		t.Fatalf("old state generation remains reachable through WorkBraid refs: %s", output)
	}

	allGenerated := make([]ChangeSet, 0, 64)
	for _, adjective := range []string{"calm", "clear", "gentle", "quiet", "steady", "bright", "open", "swift"} {
		for _, noun := range []string{"harbor", "bridge", "meadow", "signal", "river", "lantern", "grove", "compass"} {
			allGenerated = append(allGenerated, ChangeSet{Name: adjective + "-" + noun, Lifecycle: "active"})
		}
	}
	_, generated, err := manager.NewChangeSet(allGenerated, "")
	if err != nil || !strings.HasSuffix(generated, "-2") {
		t.Fatalf("generated collision suffix: name=%q err=%v", generated, err)
	}
	_, reused, err := manager.NewChangeSet([]ChangeSet{{Name: "Reusable", Lifecycle: "applied"}}, "Reusable")
	if err != nil || reused != "Reusable" {
		t.Fatalf("applied name was not reusable: name=%q err=%v", reused, err)
	}
}

func changeSetFixture(t *testing.T, manager *Manager, ctx context.Context, name string) (Snapshot, ChangeSet, string, string) {
	t.Helper()
	base, err := manager.CreateProject(ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	component := manager.NewComponentChange(base, nil, "Worker", "Body\n")
	component.RelationshipsChanged = true
	composition := CandidateComposition{NewComponentHomes: []NewComponentHome{{ComponentID: component.ID, DiagramID: base.RootDiagramID()}}}
	candidate, err := manager.PrepareCandidate(ctx, base, []ComponentChange{component}, &composition)
	if err != nil {
		t.Fatal(err)
	}
	id, selectedName, err := manager.NewChangeSet(nil, name)
	if err != nil {
		t.Fatal(err)
	}
	record := ChangeSet{ID: id, Name: selectedName, Lifecycle: "active", BaseRevision: base.Revision(), Generation: 1, BaseSnapshot: base, Changes: []ComponentChange{component}, Composition: composition, Candidate: &candidate}
	object, err := manager.WriteActiveChangeSet(ctx, base.StoreID(), record, "")
	if err != nil {
		t.Fatal(err)
	}
	storePath, err := manager.StorePath(base.StoreID())
	if err != nil {
		t.Fatal(err)
	}
	return base, record, object, storePath
}

func replaceChangeSetEnvelope(t *testing.T, manager *Manager, ctx context.Context, storePath, parent string, entries map[string]treeEntry) string {
	t.Helper()
	paths := make([]string, 0, len(entries))
	for path := range entries {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var source strings.Builder
	for _, path := range paths {
		entry := entries[path]
		fmt.Fprintf(&source, "%s %s %s\t%s\n", entry.Mode, entry.Type, entry.Object, path)
	}
	tree, err := manager.git.makeTree(ctx, storePath, []byte(source.String()))
	if err != nil {
		t.Fatal(err)
	}
	commit, err := manager.git.makeStateCommit(ctx, storePath, tree, parent)
	if err != nil {
		t.Fatal(err)
	}
	return commit
}
