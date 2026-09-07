package architecture

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"reflect"
	"strings"
	"testing"
)

func TestSizingClosedPortableAndOperationalSchemas(t *testing.T) {
	id, c := uuid.NewString(), uuid.NewString()
	prefix := fmt.Sprintf("id: %s\ntitle: Diagram\nappearances:\n  - component: %s\n    role: home\npositions:\n  - component: %s\n    x: 0\n    y: 0\nsizes:\n  - component: %s\n    ", id, c, c, c)
	for _, pair := range []string{"width: 80\n    height: 48", "width: 1600\n    height: 1200", "width: 201\n    height: 97"} {
		for _, version := range []int{2, 3, 4} {
			_, err := parseDiagram("diagrams/root.yaml", []byte(prefix+pair+"\n"), version)
			if (err == nil) != (version == 4) {
				t.Fatalf("version %d pair %s: %v", version, pair, err)
			}
		}
	}
	for _, pair := range []string{"width: 79\n    height: 48", "width: 80\n    height: 1201", "width: 80.0\n    height: 48", "width: '80'\n    height: 48", "width: null\n    height: 48", "width: true\n    height: 48", "width: 80", "width: 80\n    height: 48\n    extra: 1", "width: 80\n    width: 90\n    height: 48", "width: 999999999999999999999999\n    height: 48"} {
		if _, err := parseDiagram("diagrams/root.yaml", []byte(prefix+pair+"\n"), 4); err == nil {
			t.Fatalf("invalid size accepted: %s", pair)
		}
	}
	composition := CandidateComposition{ArchitectureVersion: 4, NodeSizes: []NodeSizeChange{{id, c, &Size{200, 96}}}}
	raw, err := marshalChangeState(nil, composition)
	if err != nil {
		t.Fatal(err)
	}
	_, loaded, err := parseChangeState(raw)
	if err != nil || !reflect.DeepEqual(loaded.NodeSizes, composition.NodeSizes) {
		t.Fatalf("size facts: %v %+v", err, loaded)
	}
	for _, bad := range []string{strings.Replace(string(raw), "version: 5", "version: 3", 1), strings.Replace(string(raw), "node_sizes:", "other_sizes:", 1), strings.Replace(string(raw), "width: 200", "width: 200.5", 1)} {
		if _, _, err := parseChangeState([]byte(bad)); err == nil {
			t.Fatal("invalid operational sizes accepted")
		}
	}
}

func TestSizingLegacyInitializationIsOrdinaryAndStrictReplayExact(t *testing.T) {
	for _, version := range []int{2, 3} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			m, base, ids, _ := placementFixture(t)
			if version == 3 {
				base = reconciliationAccepted(t, m, base, nil, CandidateComposition{ArchitectureVersion: 3})
			}
			before := base.DiagramProjections()
			composition := SetNodeSize(base, CandidateComposition{}, ids.root, ids.gateway, Size{321, 159})
			incomplete := composition
			incomplete.NodeSizes = incomplete.NodeSizes[:1]
			if _, err := m.ConstructCandidate(t.Context(), base, nil, incomplete); err == nil {
				t.Fatal("strict constructor initialized missing sizes")
			}
			candidate, err := m.PrepareCandidate(t.Context(), base, nil, &composition)
			if err != nil {
				t.Fatal(err)
			}
			if candidate.Snapshot().FormatVersion() != 4 {
				t.Fatal("not upgraded")
			}
			replay, err := m.ConstructCandidate(t.Context(), base, nil, composition)
			if err != nil || replay.Tree() != candidate.Tree() {
				t.Fatalf("replay %v", err)
			}
			for _, d := range before {
				check := func(cid string, p *Position, s *Size) {
					saved, ok := candidate.Snapshot().NodeSize(d.ID, cid)
					if !ok || saved == nil {
						t.Fatal("missing saved size")
					}
					expected := *s
					if d.ID == ids.root && cid == ids.gateway {
						expected = Size{321, 159}
					}
					if *saved != expected {
						t.Fatalf("peer resized %s: %+v want %+v", cid, saved, expected)
					}
					pos, _ := candidate.Snapshot().NodePosition(d.ID, cid)
					if !samePosition(pos, p) {
						t.Fatal("upgrade moved center")
					}
				}
				for _, a := range d.Appearances {
					check(a.ComponentID, a.DisplayPosition, a.DisplaySize)
				}
				for _, b := range d.Boundaries {
					check(b.ComponentID, b.DisplayPosition, b.DisplaySize)
				}
			}
			composition.NodeSizes = nil
			if _, err := m.ConstructCandidate(t.Context(), base, nil, composition); err == nil {
				t.Fatal("read/review could repair missing size facts")
			}
		})
	}
}

func TestSizingReconciliationIndependentPairsAndMixedFormats(t *testing.T) {
	for _, mode := range []string{"size_position", "different_nodes", "same_result", "conflict", "legacy_accepted", "legacy_proposed"} {
		t.Run(mode, func(t *testing.T) {
			m, legacy, ids, _ := placementFixture(t)
			base := reconciliationAccepted(t, m, legacy, nil, CandidateComposition{ArchitectureVersion: 3})
			ac, pc := CandidateComposition{}, CandidateComposition{}
			ac = SetNodeSize(base, ac, ids.root, ids.gateway, Size{360, 180})
			switch mode {
			case "size_position":
				pc = SetNodePosition(base, pc, ids.root, ids.gateway, &Position{321, -123})
			case "different_nodes":
				pc = SetNodeSize(base, pc, ids.root, ids.worker, Size{400, 200})
			case "same_result":
				pc = SetNodeSize(base, pc, ids.root, ids.gateway, Size{360, 180})
			case "conflict":
				pc = SetNodeSize(base, pc, ids.root, ids.gateway, Size{420, 220})
			case "legacy_accepted":
				ac = CandidateComposition{DiagramTitles: []DiagramTitleChange{{DiagramID: ids.root, Title: "Accepted title"}}}
				pc = SetNodeSize(base, pc, ids.root, ids.gateway, Size{360, 180})
			case "legacy_proposed":
				pc = CandidateComposition{DiagramTitles: []DiagramTitleChange{{DiagramID: ids.root, Title: "Proposed title"}}}
			}
			a := reconciliationAccepted(t, m, base, nil, ac)
			p := reconciliationProposed(t, m, base, nil, pc)
			result, err := m.Reconcile(t.Context(), base, a, p, nil)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "conflict" {
				if result.Status != "needs_resolution" || len(result.Conflicts) != 1 || result.Conflicts[0].Locator.Kind != "node_size" {
					t.Fatalf("conflict %+v", result)
				}
				value := Size{500, 250}
				result, err = m.Reconcile(t.Context(), base, a, p, []ReconciliationResolution{{Locator: result.Conflicts[0].Locator, Choice: "manual", Value: &ReconciliationValue{Size: &value}}})
				if err != nil {
					t.Fatal(err)
				}
			}
			if result.Status != "ready" || result.Candidate == nil {
				t.Fatalf("not ready %+v", result)
			}
			expected := Size{360, 180}
			if mode == "conflict" {
				expected = Size{500, 250}
			}
			got, _ := result.Candidate.Snapshot().NodeSize(ids.root, ids.gateway)
			if got == nil || *got != expected {
				t.Fatalf("lost stored size %+v", got)
			}
			if mode == "size_position" {
				pos, _ := result.Candidate.Snapshot().NodePosition(ids.root, ids.gateway)
				if pos == nil || *pos != (Position{321, -123}) {
					t.Fatal("size swallowed move")
				}
			}
			replay, err := m.ConstructCandidate(t.Context(), a, result.Changes, result.Composition)
			if err != nil || replay.Tree() != result.Candidate.Tree() {
				t.Fatalf("residual %v", err)
			}
		})
	}
}
func TestSizingManualResolutionClosedPair(t *testing.T) {
	id, c := uuid.NewString(), uuid.NewString()
	prefix := fmt.Sprintf("{\"locator\":{\"kind\":\"node_size\",\"diagram_id\":%q,\"component_id\":%q},\"choice\":\"manual\",\"value\":", id, c)
	for _, value := range []string{`{"size":{"width":80,"height":48}}`, `{"size":{"width":1600,"height":1200}}`} {
		var r ReconciliationResolution
		if err := json.Unmarshal([]byte(prefix+value+"}"), &r); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range []string{`{"size":null}`, `{"size":{"width":null,"height":48}}`, `{"size":{"width":80.5,"height":48}}`, `{"size":{"width":80,"height":48,"x":0}}`, `{"size":{"width":80,"width":90,"height":48}}`, `{"size":{"width":80,"height":47}}`, `{"position":{"x":0,"y":0}}`} {
		var r ReconciliationResolution
		if err := json.Unmarshal([]byte(prefix+value+"}"), &r); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
}
