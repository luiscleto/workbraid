package architecture

import (
	"context"
	"strings"
	"sync"
	"testing"
)

func TestCreateProjectCatalogUsesStableUUIDAndUniqueSlugs(t *testing.T) {
	manager := NewManager(t.TempDir())
	first, err := manager.CreateProject(context.Background(), "  Example Project  ")
	if err != nil {
		t.Fatal(err)
	}
	second, err := manager.CreateProject(context.Background(), "Example Project")
	if err != nil {
		t.Fatal(err)
	}
	fallback, err := manager.CreateProject(context.Background(), "日本語")
	if err != nil {
		t.Fatal(err)
	}
	if first.ProjectName() != "Example Project" || first.ProjectSlug() != "example-project" || second.ProjectSlug() != "example-project-2" || fallback.ProjectSlug() != "project" {
		t.Fatalf("unexpected projects: first=%q/%q second=%q fallback=%q", first.ProjectName(), first.ProjectSlug(), second.ProjectSlug(), fallback.ProjectSlug())
	}
	if first.StoreID() == second.StoreID() || first.RootDiagramID() == second.RootDiagramID() {
		t.Fatal("creation reused stable identity")
	}
	projects, err := manager.Catalog(context.Background())
	if err != nil || len(projects) != 3 {
		t.Fatalf("catalog = %+v, %v", projects, err)
	}
	reopened, err := NewManager(strings.TrimSuffix(manager.storeRoot, "/architecture")).OpenProject(context.Background(), first.ProjectSlug())
	if err != nil || reopened.StoreID() != first.StoreID() || reopened.Revision() != first.Revision() {
		t.Fatalf("reopen = %+v, %v", reopened, err)
	}
}

func TestConcurrentSameNameCreationSelectsDistinctSlugs(t *testing.T) {
	manager := NewManager(t.TempDir())
	start := make(chan struct{})
	results := make(chan Snapshot, 2)
	errors := make(chan error, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			value, err := manager.CreateProject(context.Background(), "Concurrent Project")
			results <- value
			errors <- err
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	slugs := map[string]bool{}
	for result := range results {
		slugs[result.ProjectSlug()] = true
	}
	if !slugs["concurrent-project"] || !slugs["concurrent-project-2"] || len(slugs) != 2 {
		t.Fatalf("concurrent slugs = %v", slugs)
	}
}

func TestCatalogConflictsAndUnavailableStoresAreExplicit(t *testing.T) {
	data := t.TempDir()
	manager := NewManager(data)
	first, err := manager.CreateProject(context.Background(), "First")
	if err != nil {
		t.Fatal(err)
	}
	second, err := manager.CreateProject(context.Background(), "Second")
	if err != nil {
		t.Fatal(err)
	}
	storePath, _ := manager.StorePath(second.StoreID())
	manifest := gitBytes(t, "--git-dir", storePath, "show", second.Revision()+":architecture.yaml")
	manifest = []byte(strings.Replace(string(manifest), "slug: second", "slug: first", 1))
	manifestBlob := writeTestBlob(t, storePath, manifest)
	root := gitText(t, "--git-dir", storePath, "ls-tree", second.Revision())
	lines := strings.Split(root, "\n")
	for index, line := range lines {
		if strings.HasSuffix(line, "\tarchitecture.yaml") {
			lines[index] = "100644 blob " + manifestBlob + "\tarchitecture.yaml"
		}
	}
	tree := mktree(t, storePath, strings.Join(lines, "\n")+"\n")
	commit := gitTextWithInput(t, []byte("external slug\n"), "--git-dir", storePath, "commit-tree", tree, "-p", second.Revision())
	gitText(t, "--git-dir", storePath, "update-ref", acceptedRef, commit, second.Revision())

	missingID := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	if err := manager.git.initBare(context.Background(), manager.storeRoot+"/"+missingID+".git"); err != nil {
		t.Fatal(err)
	}
	projects, err := manager.Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	conflicts, unavailable := 0, 0
	for _, project := range projects {
		if project.Conflict {
			conflicts++
		}
		if project.Unavailable {
			unavailable++
		}
	}
	if conflicts != 2 || unavailable != 1 {
		t.Fatalf("catalog states: %+v", projects)
	}
	if _, err := manager.OpenProject(context.Background(), first.ProjectSlug()); err != ErrCatalogConflict {
		t.Fatalf("conflict open error = %v", err)
	}
	if _, err := manager.OpenProject(context.Background(), "unknown"); err != ErrProjectNotFound {
		t.Fatalf("unknown open error = %v", err)
	}
}

func TestReferenceCompositionAndHomeMoveNonResurrection(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(t.TempDir())
	base, err := manager.CreateProject(ctx, "References")
	if err != nil {
		t.Fatal(err)
	}
	root := base.RootDiagramID()
	anchorB := manager.NewComponentChange(base, nil, "Anchor B", "")
	anchorC := manager.NewComponentChange(base, []ComponentChange{anchorB}, "Anchor C", "")
	moving := manager.NewComponentChange(base, []ComponentChange{anchorB, anchorC}, "Moving", "")
	b := base.NewDetailDiagramChange(nil, "B", anchorB.ID)
	c := base.NewDetailDiagramChange([]DetailDiagramChange{b}, "C", anchorC.ID)
	initial, err := manager.prepareTestCandidate(ctx, base, []ComponentChange{anchorB, anchorC, moving}, CandidateComposition{
		NewComponentHomes: []NewComponentHome{{ComponentID: anchorB.ID, DiagramID: root}, {ComponentID: anchorC.ID, DiagramID: root}, {ComponentID: moving.ID, DiagramID: root}},
		DetailDiagrams:    []DetailDiagramChange{b, c},
	})
	if err != nil {
		t.Fatal(err)
	}
	base = acceptCandidate(t, manager, base, initial)
	withReference, err := manager.prepareTestCandidate(ctx, base, nil, CandidateComposition{References: []ReferenceAppearanceChange{{DiagramID: b.ID, ComponentID: moving.ID, Present: true}}})
	if err != nil {
		t.Fatal(err)
	}
	base = acceptCandidate(t, manager, base, withReference)

	moved, err := manager.prepareTestCandidate(ctx, base, nil, CandidateComposition{
		HomeMoves:  []ComponentHomeMove{{ComponentID: moving.ID, DiagramID: c.ID}},
		References: []ReferenceAppearanceChange{{DiagramID: root, ComponentID: moving.ID, Present: false}, {DiagramID: b.ID, ComponentID: moving.ID, Present: false}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := moved.Snapshot().ComponentAppearanceRole(root, moving.ID); ok {
		t.Fatal("old home remained")
	}
	if _, ok := moved.Snapshot().ComponentAppearanceRole(b.ID, moving.ID); ok {
		t.Fatal("accepted reference resurrected")
	}
	if role, ok := moved.Snapshot().ComponentAppearanceRole(c.ID, moving.ID); !ok || role != "home" {
		t.Fatalf("destination = %q %t", role, ok)
	}

	shown, err := manager.prepareTestCandidate(ctx, base, nil, CandidateComposition{
		HomeMoves:  []ComponentHomeMove{{ComponentID: moving.ID, DiagramID: c.ID}},
		References: []ReferenceAppearanceChange{{DiagramID: root, ComponentID: moving.ID, Present: false}, {DiagramID: b.ID, ComponentID: moving.ID, Present: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if role, ok := shown.Snapshot().ComponentAppearanceRole(b.ID, moving.ID); !ok || role != "reference" {
		t.Fatalf("show-back = %q %t", role, ok)
	}
	if role, _ := shown.Snapshot().ComponentAppearanceRole(c.ID, moving.ID); role != "home" {
		t.Fatalf("home = %q", role)
	}
}

func acceptCandidate(t *testing.T, manager *Manager, base Snapshot, candidate Candidate) Snapshot {
	t.Helper()
	commit, err := manager.CreateSuccessor(context.Background(), base, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.AdvanceAccepted(context.Background(), base, commit); err != nil {
		t.Fatal(err)
	}
	loaded, err := manager.LoadAccepted(context.Background(), base.StoreID())
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Revision() != commit {
		t.Fatalf("accepted %s, want %s", loaded.Revision(), commit)
	}
	return loaded
}

func TestProjectSlugGrammar(t *testing.T) {
	valid := []string{"a", "a1", "example-project", "project-2"}
	invalid := []string{"", "-a", "a-", "a--b", "A", "a_b", "é"}
	for _, value := range valid {
		if !validProjectSlug(value) {
			t.Errorf("%q invalid", value)
		}
	}
	for _, value := range invalid {
		if validProjectSlug(value) {
			t.Errorf("%q valid", value)
		}
	}
	if got := projectSlug("  Hello, WORLD!  "); got != "hello-world" {
		t.Fatalf("slug = %q", got)
	}
	if got := projectSlug("💥"); got != "project" {
		t.Fatalf("fallback = %q", got)
	}
}
