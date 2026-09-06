package architecture

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type reconciliationTestFile struct{ source, mode string }

// Valid external advancement is a real Git authority input. Tests still load
// it with the production loader and derive every proposal with the constructor.
func externalReconciliationSnapshot(t *testing.T, m *Manager, b Snapshot, edit func(map[string]reconciliationTestFile)) Snapshot {
	t.Helper()
	path := mustStorePath(t, m, b.StoreID())
	entries, err := m.git.treeEntries(t.Context(), path, b.Revision())
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]reconciliationTestFile{}
	for _, entry := range entries {
		if entry.Type == "blob" {
			source, err := m.git.readBlob(t.Context(), path, entry.Object)
			if err != nil {
				t.Fatal(err)
			}
			files[entry.Path] = reconciliationTestFile{string(source), entry.Mode}
		}
	}
	edit(files)
	groups := map[string][]string{}
	for name, file := range files {
		dir := filepath.Dir(name)
		groups[dir] = append(groups[dir], file.mode+" blob "+writeTestBlob(t, path, []byte(file.source))+"\t"+filepath.Base(name))
	}
	root := groups["."]
	for _, dir := range []string{"components", "diagrams"} {
		if len(groups[dir]) > 0 {
			tree := mktree(t, path, strings.Join(groups[dir], "\n")+"\n")
			root = append(root, "040000 tree "+tree+"\t"+dir)
		}
	}
	tree := mktree(t, path, strings.Join(root, "\n")+"\n")
	revision := gitText(t, "--git-dir", path, "commit-tree", tree, "-m", "External exact Architecture")
	gitText(t, "--git-dir", path, "update-ref", acceptedRef, revision, b.Revision())
	a, err := m.LoadAccepted(t.Context(), b.StoreID())
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestReconciliationAcceptedSourcePathModeAndPlainTitleFidelity(t *testing.T) {
	m, b, ids := reconciliationFixture(t)
	a := externalReconciliationSnapshot(t, m, b, func(files map[string]reconciliationTestFile) {
		gateway := files["components/gateway.md"]
		gateway.source = strings.Replace(gateway.source, "---\n", "---\n# Accepted metadata formatting\n", 1)
		gateway.source = strings.Replace(gateway.source, "# Gateway\n", "# **Gateway** ###\r\n", 1)
		gateway.mode = "100755"
		delete(files, "components/gateway.md")
		files["components/renamed-gateway.md"] = gateway
		root := files["diagrams/root.yaml"]
		root.source = "# Accepted Diagram formatting\n" + root.source
		root.mode = "100755"
		files["diagrams/root.yaml"] = root
		manifest := files["architecture.yaml"]
		manifest.source = "# Accepted manifest formatting\n" + manifest.source
		files["architecture.yaml"] = manifest
	})
	pChange, _ := b.ChangeForAcceptedComponent(ids.gateway)
	pChange.Description = "\n \r\nExact new body\r\n"
	pChange.DescriptionChanged = true
	p := reconciliationProposed(t, m, b, []ComponentChange{pChange}, CandidateComposition{})
	result, err := m.Reconcile(t.Context(), b, a, p, nil)
	if err != nil || result.Status != "ready" || len(result.Conflicts) > 0 {
		t.Fatalf("source-only changes became conflicts: %+v %v", result, err)
	}
	if len(result.Changes) != 1 || result.Changes[0].TitleChanged || result.Changes[0].RelationshipsChanged || result.Changes[0].Path != "components/renamed-gateway.md" {
		t.Fatalf("source foundation: %+v", result.Changes)
	}
	path := mustStorePath(t, m, b.StoreID())
	for _, name := range []string{"architecture.yaml", "diagrams/root.yaml", "diagrams/detail.yaml", "diagrams/empty.yaml", "components/worker.md", "components/ledger.md", "components/records.md"} {
		if gitText(t, "--git-dir", path, "ls-tree", a.Revision(), "--", name) != gitText(t, "--git-dir", path, "ls-tree", result.Candidate.Tree(), "--", name) {
			t.Fatalf("unchanged A entry changed: %s", name)
		}
	}
	got := gitBytes(t, "--git-dir", path, "show", result.Candidate.Tree()+":components/renamed-gateway.md")
	want := gitBytes(t, "--git-dir", path, "show", a.Revision()+":components/renamed-gateway.md")
	want = []byte(strings.TrimSuffix(string(want), "Gateway docs.\n") + pChange.Description)
	if string(got) != string(want) || !strings.HasPrefix(gitText(t, "--git-dir", path, "ls-tree", result.Candidate.Tree(), "--", "components/renamed-gateway.md"), "100755 ") {
		t.Fatalf("A source/mode or P exact body changed: %q", got)
	}
}

func TestReconciliationExternalComponentAbsenceRequiresDependentChoice(t *testing.T) {
	m, b, ids := reconciliationFixture(t)
	a := externalReconciliationSnapshot(t, m, b, func(files map[string]reconciliationTestFile) {
		delete(files, "components/ledger.md")
		worker := files["components/worker.md"]
		worker.source = strings.Replace(worker.source, "  - target: \""+ids.ledger+"\"\n    label: uses\n", "", 1)
		files["components/worker.md"] = worker
		for _, name := range []string{"diagrams/root.yaml", "diagrams/detail.yaml"} {
			file := files[name]
			file.source = strings.Replace(file.source, "  - component: \""+ids.ledger+"\"\n    role: home\n", "", 1)
			file.source = strings.Replace(file.source, "  - component: \""+ids.ledger+"\"\n    role: reference\n", "", 1)
			files[name] = file
		}
	})
	ledger, _ := b.ChangeForAcceptedComponent(ids.ledger)
	ledger.Description = "Proposed edited ledger\n"
	ledger.DescriptionChanged = true
	worker, _ := b.ChangeForAcceptedComponent(ids.worker)
	worker.Relationships = append(worker.Relationships, AuthoringRelationship{TargetID: ids.ledger, Label: "new dependency"})
	worker.RelationshipsChanged = true
	p := reconciliationProposed(t, m, b, []ComponentChange{ledger, worker}, CandidateComposition{})
	preview, err := m.Reconcile(t.Context(), b, a, p, nil)
	if err != nil {
		t.Fatal(err)
	}
	var object, dependent ReconciliationLocator
	for _, conflict := range preview.Conflicts {
		if conflict.Locator.Kind == "component_object" {
			object = conflict.Locator
		}
		if conflict.Locator.Reason == "missing_target" {
			dependent = conflict.Locator
		}
	}
	if object.Kind == "" || dependent.Kind == "" {
		t.Fatalf("absence lost context: %+v", preview)
	}
	choice := ReconciliationResolution{Locator: object, Choice: "accepted"}
	partial, err := m.Reconcile(t.Context(), b, a, p, []ReconciliationResolution{choice})
	if err != nil || partial.Status != "needs_resolution" {
		t.Fatalf("dependent work silently dropped: %+v %v", partial, err)
	}
	partial, err = m.Reconcile(t.Context(), b, a, p, []ReconciliationResolution{choice, {Locator: dependent, Choice: "accepted", Value: &ReconciliationValue{}}})
	if err != nil || partial.Status != "needs_resolution" || partial.Candidate != nil {
		t.Fatalf("sparse side choice silently dropped dependent work: %+v %v", partial, err)
	}
	resolved, err := m.Reconcile(t.Context(), b, a, p, []ReconciliationResolution{choice, {Locator: dependent, Choice: "accepted", Value: &ReconciliationValue{RelationshipCounts: []ReconciliationRelationshipCount{{ids.worker, ids.ledger, "new dependency", 0}}}}})
	if err != nil || resolved.Status != "ready" {
		t.Fatalf("supported absence: %+v %v", resolved, err)
	}
	if _, exists := snapshotReconciliationFacts(resolved.Candidate.Snapshot()).components[ids.ledger]; exists {
		t.Fatal("deleted identity restored")
	}
}

func TestReconciliationOccupiedDestinationExpandsBeforeJointSwap(t *testing.T) {
	m, b, ids := reconciliationFixture(t)
	// The new child competes with an existing child that P deliberately moves.
	newChild := b.NewDetailDiagramChange(nil, "New detail", ids.ledger)
	a := reconciliationAccepted(t, m, b, nil, CandidateComposition{DetailDiagrams: []DetailDiagramChange{newChild}})
	p := reconciliationProposed(t, m, b, nil, CandidateComposition{DetailReassignments: []DetailReassignment{{ids.empty, ids.ledger}}})
	preview, err := m.Reconcile(t.Context(), b, a, p, nil)
	if err != nil || len(preview.Conflicts) != 1 {
		t.Fatalf("competing: %+v %v", preview, err)
	}
	choice := ReconciliationResolution{Locator: preview.Conflicts[0].Locator, Choice: "manual", Value: &ReconciliationValue{DetailAnchors: []DetailReassignment{{newChild.ID, ids.ledger}, {ids.empty, ids.gateway}}}}
	expanded, err := m.Reconcile(t.Context(), b, a, p, []ReconciliationResolution{choice})
	if err != nil || expanded.Status != "needs_resolution" || expanded.Candidate != nil {
		t.Fatalf("occupied destination was silently displaced: %+v %v", expanded, err)
	}
	for _, conflict := range expanded.Conflicts {
		if len(conflict.Locator.DiagramIDs) == 3 {
			choice.Locator = conflict.Locator
		}
	}
	if len(choice.Locator.DiagramIDs) != 3 {
		t.Fatalf("expanded context missing: %+v", expanded)
	}
	choice.Value.DetailAnchors = append(choice.Value.DetailAnchors, DetailReassignment{ids.detail, ids.records})
	resolved, err := m.Reconcile(t.Context(), b, a, p, []ReconciliationResolution{choice})
	if err != nil || resolved.Status != "ready" {
		t.Fatalf("explicit expanded swap: %+v %v", resolved, err)
	}
	for _, assignment := range choice.Value.DetailAnchors {
		anchor, _, _ := resolved.Candidate.Snapshot().DiagramParent(assignment.DiagramID)
		if anchor != assignment.AnchorComponentID {
			t.Fatal("joint assignment lost")
		}
	}
	choice.Choice = "accepted"
	sideResolved, err := m.Reconcile(t.Context(), b, a, p, []ReconciliationResolution{choice})
	if err != nil || sideResolved.Status != "ready" {
		t.Fatalf("expanded side preference prevented explicit displacement: %+v %v", sideResolved, err)
	}
}

func TestReconciliationAdoptsExternalRootWithoutReassigningIt(t *testing.T) {
	m, b, ids := reconciliationFixture(t)
	a := externalReconciliationSnapshot(t, m, b, func(files map[string]reconciliationTestFile) {
		manifest := files["architecture.yaml"]
		manifest.source = strings.Replace(manifest.source, "root_diagram: \""+ids.root+"\"", "root_diagram: \""+ids.detail+"\"", 1)
		files["architecture.yaml"] = manifest
		root := files["diagrams/root.yaml"]
		root.source = strings.Replace(root.source, "    detail_diagram: \""+ids.detail+"\"\n", "", 1)
		files["diagrams/root.yaml"] = root
		detail := files["diagrams/detail.yaml"]
		detail.source = strings.Replace(detail.source, "    role: home\n", "    role: home\n    detail_diagram: \""+ids.root+"\"\n", 1)
		files["diagrams/detail.yaml"] = detail
	})
	p := reconciliationProposed(t, m, b, nil, CandidateComposition{DetailReassignments: []DetailReassignment{{ids.detail, ids.ledger}}})
	preview, err := m.Reconcile(t.Context(), b, a, p, nil)
	if err != nil || len(preview.Conflicts) != 1 {
		t.Fatalf("root context: %+v %v", preview, err)
	}
	conflict := preview.Conflicts[0]
	if conflict.Unsupported["proposed"] != "reassign_root" {
		t.Fatalf("root choice offered: %+v", conflict)
	}
	resolved, err := m.Reconcile(t.Context(), b, a, p, []ReconciliationResolution{{Locator: conflict.Locator, Choice: "accepted"}})
	if err != nil || resolved.Status != "ready" {
		t.Fatalf("accepted root: %+v %v", resolved, err)
	}
	_, err = m.Reconcile(t.Context(), b, a, p, []ReconciliationResolution{{Locator: conflict.Locator, Choice: "proposed"}})
	var domain *ReconciliationError
	if !errors.As(err, &domain) || domain.Reason != "reassign_root" {
		t.Fatalf("root reassignment: %v", err)
	}
}

func TestReconciliationEmptyRelationshipLabelLocatorRoundTrips(t *testing.T) {
	l := ReconciliationLocator{Kind: "relationship_count", SourceID: uuid.NewString(), TargetID: uuid.NewString(), Label: ""}
	data, err := json.Marshal(ReconciliationResolution{Locator: l, Choice: "accepted"})
	if err != nil {
		t.Fatal(err)
	}
	var decoded ReconciliationResolution
	if err := json.Unmarshal(data, &decoded); err != nil || locatorKey(decoded.Locator) != locatorKey(l) {
		t.Fatalf("empty exact label: %s %v", data, err)
	}
}

func TestReconciliationHomeAndExistingChildAnchorChoices(t *testing.T) {
	m, b, ids := reconciliationFixture(t)
	a := reconciliationAccepted(t, m, b, nil, CandidateComposition{HomeMoves: []ComponentHomeMove{{ids.worker, ids.root}}, DetailReassignments: []DetailReassignment{{ids.detail, ids.records}, {ids.empty, ids.gateway}}})
	p := reconciliationProposed(t, m, b, nil, CandidateComposition{HomeMoves: []ComponentHomeMove{{ids.worker, ids.empty}}, DetailReassignments: []DetailReassignment{{ids.detail, ids.ledger}}})
	preview, err := m.Reconcile(t.Context(), b, a, p, nil)
	if err != nil || len(preview.Conflicts) != 2 {
		t.Fatalf("home/anchor conflicts: %+v %v", preview, err)
	}
	choices := []ReconciliationResolution{}
	for _, conflict := range preview.Conflicts {
		if conflict.Locator.Kind != "home" && conflict.Locator.Kind != "detail_anchor" {
			t.Fatalf("unexpected unit: %+v", conflict)
		}
		choices = append(choices, ReconciliationResolution{Locator: conflict.Locator, Choice: "proposed"})
	}
	resolved, err := m.Reconcile(t.Context(), b, a, p, choices)
	if err != nil || resolved.Status != "ready" {
		t.Fatalf("chosen home/anchor: %+v %v", resolved, err)
	}
	facts := snapshotReconciliationFacts(resolved.Candidate.Snapshot())
	if facts.homes[ids.worker] != ids.empty || facts.anchors[ids.detail] != ids.ledger {
		t.Fatal("chosen scalar composition lost")
	}
	_, err = m.Reconcile(t.Context(), b, a, p, append(choices, choices[0]))
	var domain *ReconciliationError
	if !errors.As(err, &domain) || domain.Code != "invalid_request" {
		t.Fatalf("duplicate choices: %v", err)
	}
}
