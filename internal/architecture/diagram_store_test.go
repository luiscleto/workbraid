package architecture

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestLoadAcceptedV2DiagramProjectionFromRealGit(t *testing.T) {
	manager := NewManager(t.TempDir())
	storeID := uuid.NewString()
	base, err := manager.InitializeOrLoad(context.Background(), storeID, "Project", "/tmp/project")
	if err != nil {
		t.Fatal(err)
	}
	storePath, _ := manager.StorePath(storeID)
	ids := diagramFixtureIDs{
		root: uuid.NewString(), detail: uuid.NewString(), empty: uuid.NewString(),
		gateway: uuid.NewString(), worker: uuid.NewString(), records: uuid.NewString(), ledger: uuid.NewString(),
	}
	commit := commitV2Fixture(t, storePath, storeID, ids)
	gitText(t, "--git-dir", storePath, "update-ref", acceptedRef, commit, base.Revision())
	before := acceptedAuthorityState(t, storePath)

	snapshot, err := manager.LoadAccepted(context.Background(), storeID)
	if err != nil {
		t.Fatalf("load v2: %v", err)
	}
	if snapshot.FormatVersion() != 2 || snapshot.Revision() != commit || snapshot.RootDiagramID() != ids.root {
		t.Fatalf("unexpected v2 snapshot: version=%d revision=%q root=%q", snapshot.FormatVersion(), snapshot.Revision(), snapshot.RootDiagramID())
	}
	if after := acceptedAuthorityState(t, storePath); after != before {
		t.Fatalf("read-only v2 load changed repository\nbefore:\n%s\nafter:\n%s", before, after)
	}

	projections := snapshot.DiagramProjections()
	if len(projections) != 3 {
		t.Fatalf("Diagram projections = %d, want 3", len(projections))
	}
	byID := make(map[string]DiagramProjection, len(projections))
	for _, projection := range projections {
		byID[projection.ID] = projection
	}
	detail := byID[ids.detail]
	if detail.ParentDiagramID != ids.root || detail.ParentAnchorComponentID != ids.gateway {
		t.Fatalf("detail parent = (%q, %q)", detail.ParentDiagramID, detail.ParentAnchorComponentID)
	}
	if len(detail.Breadcrumbs) != 2 || detail.Breadcrumbs[0].ID != ids.root || detail.Breadcrumbs[0].FocusAnchorComponentID != ids.gateway || detail.Breadcrumbs[1].ID != ids.detail {
		t.Fatalf("detail breadcrumbs = %+v", detail.Breadcrumbs)
	}
	if len(detail.Boundaries) != 2 {
		t.Fatalf("detail boundaries = %+v, want Records and Gateway", detail.Boundaries)
	}
	boundaryByComponent := make(map[string]DiagramBoundary)
	for _, boundary := range detail.Boundaries {
		boundaryByComponent[boundary.ComponentID] = boundary
	}
	recordsBoundary := boundaryByComponent[ids.records]
	if recordsBoundary.Key != "boundary:"+ids.records || recordsBoundary.HomeDiagramID != ids.root || recordsBoundary.HomeDiagramTitle != "System" {
		t.Fatalf("Records boundary = %+v", recordsBoundary)
	}
	var recordsEdges []DiagramRelationship
	for _, relationship := range detail.Relationships {
		if relationship.SourceComponentID == ids.records || relationship.TargetComponentID == ids.records {
			recordsEdges = append(recordsEdges, relationship)
		}
	}
	if len(recordsEdges) != 3 {
		t.Fatalf("crossing Records edges = %+v, want 3", recordsEdges)
	}
	labels := make(map[string]int)
	for _, relationship := range recordsEdges {
		if relationship.SourceComponentID == ids.records && relationship.SourceNodeKey != recordsBoundary.Key {
			t.Fatalf("crossing source did not use shared boundary: %+v", relationship)
		}
		if relationship.TargetComponentID == ids.records && relationship.TargetNodeKey != recordsBoundary.Key {
			t.Fatalf("crossing target did not use shared boundary: %+v", relationship)
		}
		labels[relationship.Label]++
	}
	if labels["calls\nnext"] != 2 || labels["feeds"] != 1 {
		t.Fatalf("crossing labels/multiplicity changed: %+v", labels)
	}

	root := byID[ids.root]
	var workerAppearance DiagramAppearance
	for _, appearance := range root.Appearances {
		if appearance.ComponentID == ids.worker {
			workerAppearance = appearance
		}
	}
	if workerAppearance.Role != "reference" {
		t.Fatalf("root Worker appearance = %+v, want canonical reference", workerAppearance)
	}
	for _, boundary := range root.Boundaries {
		if boundary.ComponentID == ids.worker {
			t.Fatal("canonical reference was also projected as a boundary")
		}
	}
	if empty := byID[ids.empty]; len(empty.Appearances) != 0 || empty.ParentAnchorComponentID != ids.records {
		t.Fatalf("empty detail projection = %+v", empty)
	}
}

func TestLoadAcceptedV2RejectsBoundedInvalidMatrix(t *testing.T) {
	storeID := uuid.NewString()
	rootID := uuid.NewString()
	childID := uuid.NewString()
	componentID := uuid.NewString()
	otherComponentID := uuid.NewString()
	manifest := func(root string) string {
		return fmt.Sprintf("format: workbraid-architecture\nversion: 2\nstore_id: %q\nproject:\n  name: Project\n  source_hint: /tmp/project\nroot_diagram: %q\n", storeID, root)
	}
	component := func(id string) string { return fmt.Sprintf("---\nid: %q\n---\n# Component\n", id) }

	tests := []struct {
		name       string
		manifest   string
		components map[string]string
		diagrams   map[string]string
		extra      []string
	}{
		{name: "unknown manifest key", manifest: manifest(rootID) + "layout: {}\n", diagrams: map[string]string{"root.yaml": fmt.Sprintf("id: %q\ntitle: Root\nappearances: []\n", rootID)}},
		{name: "wrong manifest scalar type", manifest: strings.Replace(manifest(rootID), fmt.Sprintf("root_diagram: %q", rootID), "root_diagram: 42", 1), diagrams: map[string]string{"root.yaml": fmt.Sprintf("id: %q\ntitle: Root\nappearances: []\n", rootID)}},
		{name: "unresolved root", manifest: manifest(uuid.NewString()), diagrams: map[string]string{"root.yaml": fmt.Sprintf("id: %q\ntitle: Root\nappearances: []\n", rootID)}},
		{name: "missing Diagram id", manifest: manifest(rootID), diagrams: map[string]string{"root.yaml": "title: Root\nappearances: []\n"}},
		{name: "invalid Diagram id", manifest: manifest(rootID), diagrams: map[string]string{"root.yaml": "id: not-a-uuid\ntitle: Root\nappearances: []\n"}},
		{name: "duplicate Diagram id", manifest: manifest(rootID), diagrams: map[string]string{
			"root.yaml": fmt.Sprintf("id: %q\ntitle: Root\nappearances: []\n", rootID),
			"also.yaml": fmt.Sprintf("id: %q\ntitle: Also root\nappearances: []\n", rootID),
		}},
		{name: "empty title", manifest: manifest(rootID), diagrams: map[string]string{"root.yaml": fmt.Sprintf("id: %q\ntitle: '   '\nappearances: []\n", rootID)}},
		{name: "unknown Diagram key", manifest: manifest(rootID), diagrams: map[string]string{"root.yaml": fmt.Sprintf("id: %q\ntitle: Root\nlayout: {}\n", rootID)}},
		{name: "wrong appearance type", manifest: manifest(rootID), diagrams: map[string]string{"root.yaml": fmt.Sprintf("id: %q\ntitle: Root\nappearances: home\n", rootID)}},
		{name: "unresolved appearance", manifest: manifest(rootID), diagrams: map[string]string{"root.yaml": fmt.Sprintf("id: %q\ntitle: Root\nappearances:\n  - component: %q\n    role: home\n", rootID, componentID)}},
		{name: "unknown appearance role", manifest: manifest(rootID), components: map[string]string{"one.md": component(componentID)}, diagrams: map[string]string{"root.yaml": fmt.Sprintf("id: %q\ntitle: Root\nappearances:\n  - component: %q\n    role: primary\n", rootID, componentID)}},
		{name: "unresolved detail Diagram", manifest: manifest(rootID), components: map[string]string{"one.md": component(componentID)}, diagrams: map[string]string{"root.yaml": fmt.Sprintf("id: %q\ntitle: Root\nappearances:\n  - component: %q\n    role: home\n    detail_diagram: %q\n", rootID, componentID, childID)}},
		{name: "home with explicitly empty detail Diagram", manifest: manifest(rootID), components: map[string]string{"one.md": component(componentID)}, diagrams: map[string]string{
			"root.yaml": fmt.Sprintf("id: %q\ntitle: Root\nappearances:\n  - component: %q\n    role: home\n    detail_diagram: \"\"\n", rootID, componentID),
		}},
		{name: "reference with explicitly empty detail Diagram", manifest: manifest(rootID), components: map[string]string{"one.md": component(componentID)}, diagrams: map[string]string{
			"root.yaml":  fmt.Sprintf("id: %q\ntitle: Root\nappearances:\n  - component: %q\n    role: home\n    detail_diagram: %q\n", rootID, componentID, childID),
			"child.yaml": fmt.Sprintf("id: %q\ntitle: Child\nappearances:\n  - component: %q\n    role: reference\n    detail_diagram: \"\"\n", childID, componentID),
		}},
		{name: "missing home", manifest: manifest(rootID), components: map[string]string{"one.md": component(componentID)}, diagrams: map[string]string{"root.yaml": fmt.Sprintf("id: %q\ntitle: Root\nappearances:\n  - component: %q\n    role: reference\n", rootID, componentID)}},
		{name: "duplicate same-Diagram appearance", manifest: manifest(rootID), components: map[string]string{"one.md": component(componentID)}, diagrams: map[string]string{
			"root.yaml": fmt.Sprintf("id: %q\ntitle: Root\nappearances:\n  - component: %q\n    role: home\n  - component: %q\n    role: home\n", rootID, componentID, componentID),
		}},
		{name: "duplicate home across Diagrams", manifest: manifest(rootID), components: map[string]string{"one.md": component(componentID), "two.md": component(otherComponentID)}, diagrams: map[string]string{
			"root.yaml":  fmt.Sprintf("id: %q\ntitle: Root\nappearances:\n  - component: %q\n    role: home\n    detail_diagram: %q\n  - component: %q\n    role: home\n", rootID, componentID, childID, otherComponentID),
			"child.yaml": fmt.Sprintf("id: %q\ntitle: Child\nappearances:\n  - component: %q\n    role: home\n", childID, componentID),
		}},
		{name: "reference detail link", manifest: manifest(rootID), components: map[string]string{"one.md": component(componentID)}, diagrams: map[string]string{
			"root.yaml":  fmt.Sprintf("id: %q\ntitle: Root\nappearances:\n  - component: %q\n    role: reference\n    detail_diagram: %q\n", rootID, componentID, childID),
			"child.yaml": fmt.Sprintf("id: %q\ntitle: Child\nappearances: []\n", childID),
		}},
		{name: "unreachable child", manifest: manifest(rootID), diagrams: map[string]string{
			"root.yaml":  fmt.Sprintf("id: %q\ntitle: Root\nappearances: []\n", rootID),
			"child.yaml": fmt.Sprintf("id: %q\ntitle: Child\nappearances: []\n", childID),
		}},
		{name: "multiple parents", manifest: manifest(rootID), components: map[string]string{"one.md": component(componentID), "two.md": component(otherComponentID)}, diagrams: map[string]string{
			"root.yaml":  fmt.Sprintf("id: %q\ntitle: Root\nappearances:\n  - component: %q\n    role: home\n    detail_diagram: %q\n  - component: %q\n    role: home\n    detail_diagram: %q\n", rootID, componentID, childID, otherComponentID, childID),
			"child.yaml": fmt.Sprintf("id: %q\ntitle: Child\nappearances: []\n", childID),
		}},
		{name: "hierarchy cycle", manifest: manifest(rootID), components: map[string]string{"one.md": component(componentID), "two.md": component(otherComponentID)}, diagrams: map[string]string{
			"root.yaml": fmt.Sprintf("id: %q\ntitle: Root\nappearances: []\n", rootID),
			"a.yaml":    fmt.Sprintf("id: %q\ntitle: A\nappearances:\n  - component: %q\n    role: home\n    detail_diagram: %q\n", childID, componentID, otherComponentID),
			"b.yaml":    fmt.Sprintf("id: %q\ntitle: B\nappearances:\n  - component: %q\n    role: home\n    detail_diagram: %q\n", otherComponentID, otherComponentID, childID),
		}},
		{name: "nested Diagram path", manifest: manifest(rootID), diagrams: map[string]string{"root.yaml": fmt.Sprintf("id: %q\ntitle: Root\nappearances: []\n", rootID)}, extra: []string{"nested"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := NewManager(t.TempDir())
			base, err := manager.InitializeOrLoad(context.Background(), storeID, "Project", "/tmp/project")
			if err != nil {
				t.Fatal(err)
			}
			storePath, _ := manager.StorePath(storeID)
			commit := commitV2Sources(t, storePath, test.manifest, test.components, test.diagrams, test.extra)
			gitText(t, "--git-dir", storePath, "update-ref", acceptedRef, commit, base.Revision())
			before := acceptedAuthorityState(t, storePath)
			loaded, err := manager.LoadAccepted(context.Background(), storeID)
			if !errors.Is(err, ErrInvalid) || loaded.Revision() != "" {
				t.Fatalf("invalid v2 load = (%+v, %v)", loaded, err)
			}
			if after := acceptedAuthorityState(t, storePath); after != before {
				t.Fatal("failed v2 load mutated repository")
			}
		})
	}
}

func TestLoadAcceptedV2RejectsNonOrdinaryDiagramEntry(t *testing.T) {
	tests := []struct {
		name  string
		entry func(t *testing.T, storePath, diagramSource string) string
	}{
		{name: "symlink mode", entry: func(t *testing.T, storePath, diagramSource string) string {
			return "120000 blob " + writeTestBlob(t, storePath, []byte(diagramSource)) + "\troot.yaml\n"
		}},
		{name: "tree object at yaml path", entry: func(t *testing.T, storePath, _ string) string {
			nestedBlob := writeTestBlob(t, storePath, []byte("not a Diagram"))
			nestedTree := mktree(t, storePath, "100644 blob "+nestedBlob+"\tvalue\n")
			return "040000 tree " + nestedTree + "\troot.yaml\n"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := NewManager(t.TempDir())
			storeID := uuid.NewString()
			base, err := manager.InitializeOrLoad(context.Background(), storeID, "Project", "/tmp/project")
			if err != nil {
				t.Fatal(err)
			}
			storePath, _ := manager.StorePath(storeID)
			rootID := uuid.NewString()
			diagramSource := fmt.Sprintf("id: %q\ntitle: Root\nappearances: []\n", rootID)
			diagramTree := mktree(t, storePath, test.entry(t, storePath, diagramSource))
			manifest := fmt.Sprintf("format: workbraid-architecture\nversion: 2\nstore_id: %q\nproject:\n  name: Project\n  source_hint: /tmp/project\nroot_diagram: %q\n", storeID, rootID)
			commit := commitManifestTree(t, storePath, []byte(manifest), "100644", []string{"040000 tree " + diagramTree + "\tdiagrams"})
			gitText(t, "--git-dir", storePath, "update-ref", acceptedRef, commit, base.Revision())
			before := acceptedAuthorityState(t, storePath)

			loaded, err := manager.LoadAccepted(context.Background(), storeID)
			if !errors.Is(err, ErrInvalid) || loaded.Revision() != "" {
				t.Fatalf("non-ordinary Diagram load = (%+v, %v)", loaded, err)
			}
			if after := acceptedAuthorityState(t, storePath); after != before {
				t.Fatal("rejected non-ordinary Diagram entry mutated repository")
			}
		})
	}
}

type diagramFixtureIDs struct {
	root, detail, empty, gateway, worker, records, ledger string
}

func commitV2Fixture(t *testing.T, storePath, storeID string, ids diagramFixtureIDs) string {
	components := map[string]string{
		"gateway.md": fmt.Sprintf("---\nid: %q\nrelationships:\n  - target: %q\n    label: calls\n---\n# Gateway\nGateway docs.\n", ids.gateway, ids.worker),
		"worker.md":  fmt.Sprintf("---\nid: %q\nrelationships:\n  - target: %q\n    label: |-\n      calls\n      next\n  - target: %q\n    label: |-\n      calls\n      next\n  - target: %q\n    label: uses\n  - target: %q\n    label: returns\n---\n# Worker\nWorker docs.\n", ids.worker, ids.records, ids.records, ids.ledger, ids.gateway),
		"records.md": fmt.Sprintf("---\nid: %q\nrelationships:\n  - target: %q\n    label: feeds\n---\n# Records\nRecords docs.\n", ids.records, ids.worker),
		"ledger.md":  fmt.Sprintf("---\nid: %q\nrelationships: []\n---\n# Worker\nLedger docs.\n", ids.ledger),
	}
	diagrams := map[string]string{
		"root.yaml":   fmt.Sprintf("id: %q\ntitle: System\nappearances:\n  - component: %q\n    role: home\n    detail_diagram: %q\n  - component: %q\n    role: reference\n  - component: %q\n    role: home\n    detail_diagram: %q\n  - component: %q\n    role: home\n", ids.root, ids.gateway, ids.detail, ids.worker, ids.records, ids.empty, ids.ledger),
		"detail.yaml": fmt.Sprintf("id: %q\ntitle: Detail\nappearances:\n  - component: %q\n    role: home\n  - component: %q\n    role: reference\n", ids.detail, ids.worker, ids.ledger),
		"empty.yaml":  fmt.Sprintf("id: %q\ntitle: Detail\nappearances: []\n", ids.empty),
	}
	manifest := fmt.Sprintf("format: workbraid-architecture\nversion: 2\nstore_id: %q\nproject:\n  name: Project\n  source_hint: /tmp/project\nroot_diagram: %q\n", storeID, ids.root)
	return commitV2Sources(t, storePath, manifest, components, diagrams, nil)
}

func commitV2Sources(t *testing.T, storePath, manifest string, components, diagrams map[string]string, extra []string) string {
	t.Helper()
	rootEntries := make([]string, 0, 3)
	if len(components) > 0 {
		var entries []string
		for name, source := range components {
			entries = append(entries, "100644 blob "+writeTestBlob(t, storePath, []byte(source))+"\t"+name)
		}
		componentTree := mktree(t, storePath, strings.Join(entries, "\n")+"\n")
		rootEntries = append(rootEntries, "040000 tree "+componentTree+"\tcomponents")
	}
	var diagramEntries []string
	for name, source := range diagrams {
		diagramEntries = append(diagramEntries, "100644 blob "+writeTestBlob(t, storePath, []byte(source))+"\t"+name)
	}
	for _, nested := range extra {
		blob := writeTestBlob(t, storePath, []byte("id: nested"))
		nestedTree := mktree(t, storePath, "100644 blob "+blob+"\tvalue.yaml\n")
		diagramEntries = append(diagramEntries, "040000 tree "+nestedTree+"\t"+nested)
	}
	diagramTree := mktree(t, storePath, strings.Join(diagramEntries, "\n")+"\n")
	rootEntries = append(rootEntries, "040000 tree "+diagramTree+"\tdiagrams")
	return commitManifestTree(t, storePath, []byte(manifest), "100644", rootEntries)
}
