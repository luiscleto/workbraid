package architecture

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"reflect"
	"strings"
	"testing"
)

func TestRoutingInvalidWorkRouteProvenanceOnWriteAndLoad(t *testing.T) {
	for _, custom := range []bool{false, true} {
		t.Run(fmt.Sprint(custom), func(t *testing.T) {
			m, b, ids, _ := placementFixture(t)
			address := RouteAddress{ids.root, ids.worker, ids.records, "calls\nnext", 2}
			base := reconciliationAccepted(t, m, b, nil, SetEdgeRoute(b, CandidateComposition{}, address, &Route{80}))
			change, _ := base.ChangeForAcceptedComponent(ids.worker)
			change.TitleChanged = true
			change.Title = ""
			id, name, err := m.NewChangeSet(nil, "Invalid work route provenance")
			if err != nil {
				t.Fatal(err)
			}
			var value *Route
			if custom {
				value = &Route{80}
			}
			record := ChangeSet{ID: id, Name: name, Lifecycle: "active", BaseRevision: base.Revision(), Generation: 1, BaseSnapshot: base, Changes: []ComponentChange{change}, Composition: CandidateComposition{ArchitectureVersion: 5, EdgeRoutes: []EdgeRouteChange{{address, value}}}}
			object, err := m.WriteActiveChangeSet(t.Context(), base.StoreID(), record, "")
			if err != nil {
				t.Fatal(err)
			}
			loaded, unavailable, err := m.LoadChangeSets(t.Context(), base.StoreID())
			if err != nil || len(unavailable) != 0 || len(loaded) != 1 || loaded[0].Candidate != nil || len(loaded[0].Composition.EdgeRoutes) != 1 {
				t.Fatalf("valid invalid-work route fact lost on load: %v %+v", err, unavailable)
			}
			record.Composition.EdgeRoutes[0].SourceID = uuid.NewString()
			record.Composition.EdgeRoutes[0].Occurrence = 999
			if _, err = m.WriteActiveChangeSet(t.Context(), base.StoreID(), record, object); err == nil {
				t.Fatal("unsupported route persisted behind invalid title")
			}
			// Install a deliberately malformed external operational fixture to verify
			// loading rejects it too, without changing the supported historical archive.
			storePath, _ := m.StorePath(base.StoreID())
			entries, err := m.git.directTreeEntries(t.Context(), storePath, object)
			if err != nil {
				t.Fatal(err)
			}
			byPath := map[string]treeEntry{}
			for _, entry := range entries {
				byPath[entry.Path] = entry
			}
			raw, err := marshalChangeState(record.Changes, record.Composition)
			if err != nil {
				t.Fatal(err)
			}
			blob, err := m.git.writeBlob(t.Context(), storePath, raw)
			if err != nil {
				t.Fatal(err)
			}
			entry := byPath["changes.yaml"]
			entry.Object = blob
			byPath["changes.yaml"] = entry
			replacement := replaceChangeSetEnvelope(t, m, t.Context(), storePath, base.Revision(), byPath)
			if err = m.git.updateRef(t.Context(), storePath, activeChangeSetPrefix+id, replacement, object); err != nil {
				t.Fatal(err)
			}
			loaded, unavailable, err = m.LoadChangeSets(t.Context(), base.StoreID())
			if err != nil || len(loaded) != 0 || len(unavailable) != 1 {
				t.Fatalf("unsupported route loaded behind invalid title: %v %+v", err, loaded)
			}
		})
	}
}

func TestRoutingLegacyUpgradeAndStrictReplay(t *testing.T) {
	for _, version := range []int{2, 3, 4} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			m, base, ids, _ := placementFixture(t)
			if version > 2 {
				base = reconciliationAccepted(t, m, base, nil, CandidateComposition{ArchitectureVersion: version})
			}
			address := RouteAddress{ids.root, ids.worker, ids.records, "calls\nnext", 2}
			before := base.DiagramProjections()
			facts := SetEdgeRoute(base, CandidateComposition{}, address, &Route{125})
			candidate, e := m.PrepareCandidate(t.Context(), base, nil, &facts)
			if e != nil {
				t.Fatal(e)
			}
			if candidate.Snapshot().FormatVersion() != 5 || candidate.Snapshot().routeAt(address).Bend != 125 {
				t.Fatal("missing v5 route")
			}
			replay, e := m.ConstructCandidate(t.Context(), base, nil, facts)
			if e != nil || replay.Tree() != candidate.Tree() {
				t.Fatalf("replay %v", e)
			}
			for _, d := range before {
				for _, v := range append(append([]DiagramAppearance{}, d.Appearances...), func() []DiagramAppearance {
					out := []DiagramAppearance{}
					for _, b := range d.Boundaries {
						out = append(out, DiagramAppearance{ComponentID: b.ComponentID, DisplayPosition: b.DisplayPosition, DisplaySize: b.DisplaySize})
					}
					return out
				}()...) {
					p, _ := candidate.Snapshot().NodePosition(d.ID, v.ComponentID)
					s, _ := candidate.Snapshot().NodeSize(d.ID, v.ComponentID)
					if !samePosition(p, v.DisplayPosition) || !sameSize(s, v.DisplaySize) {
						t.Fatal("routing upgrade moved or resized peer")
					}
				}
			}
			raw, e := marshalChangeState(nil, facts)
			if e != nil {
				t.Fatal(e)
			}
			_, loaded, e := parseChangeState(raw)
			if e != nil || !reflect.DeepEqual(loaded.EdgeRoutes, facts.EdgeRoutes) {
				t.Fatalf("facts %v", e)
			}
			for _, bad := range []string{strings.Replace(string(raw), "version: 6", "version: 4", 1), strings.Replace(string(raw), "bend: 125", "bend: null", 1), strings.Replace(string(raw), "occurrence: 2", "occurrence: 0", 1)} {
				if _, _, e := parseChangeState([]byte(bad)); e == nil {
					t.Fatal("bad operational route accepted")
				}
			}
		})
	}
}

func TestRoutingNullsAndPortableClosedSchema(t *testing.T) {
	m, b, ids, _ := placementFixture(t)
	a := RouteAddress{ids.root, ids.worker, ids.records, "calls\nnext", 2}
	base := reconciliationAccepted(t, m, b, nil, SetEdgeRoute(b, CandidateComposition{}, a, &Route{100}))
	for _, edit := range []func(*RouteAddress){func(a *RouteAddress) { a.SourceID = uuid.NewString() }, func(a *RouteAddress) { a.TargetID = a.SourceID }, func(a *RouteAddress) { a.Occurrence = 999 }, func(a *RouteAddress) { a.Label = "unknown" }} {
		address := a
		edit(&address)
		if _, e := m.ConstructCandidate(t.Context(), base, nil, CandidateComposition{ArchitectureVersion: 5, EdgeRoutes: []EdgeRouteChange{{address, nil}}}); e == nil {
			t.Fatalf("arbitrary null accepted %+v", address)
		}
	}
	change, _ := base.ChangeForAcceptedComponent(ids.worker)
	change.RelationshipsChanged = true
	change.Relationships = change.Relationships[2:]
	if _, e := m.ConstructCandidate(t.Context(), base, []ComponentChange{change}, CandidateComposition{ArchitectureVersion: 5}); e == nil {
		t.Fatal("strict replay pruned inherited route")
	}
	facts := ResetChangedRouteCounts(base, nil, []ComponentChange{change}, CandidateComposition{ArchitectureVersion: 5})
	candidate, e := m.PrepareCandidate(t.Context(), base, []ComponentChange{change}, &facts)
	if e != nil || candidate.Snapshot().routeAt(a) != nil {
		t.Fatalf("real inherited null removal: %v", e)
	}
	restored, e := m.ConstructCandidate(t.Context(), base, nil, facts)
	if e != nil || restored.Snapshot().routeAt(a) != nil {
		t.Fatalf("count reappearance resurrected: %v", e)
	}
	source := fmt.Sprintf("id: %s\ntitle: D\nroutes:\n  - source: %s\n    target: %s\n    label: calls\n    occurrence: 1\n    bend: 0\n", ids.root, ids.worker, ids.records)
	if _, e := parseDiagram("diagrams/d.yaml", []byte(source), 5); e != nil {
		t.Fatal(e)
	}
	for _, v := range []int{2, 3, 4} {
		if _, e := parseDiagram("diagrams/d.yaml", []byte(source), v); e == nil {
			t.Fatal("legacy accepted routes")
		}
	}
	for _, bad := range []string{strings.Replace(source, "bend: 0", "bend: 0.0", 1), strings.Replace(source, "bend: 0", "bend: 100001", 1), strings.Replace(source, "bend: 0", "bend: null", 1), strings.Replace(source, "bend: 0", "bend: 1\n    bend: 2", 1), strings.Replace(source, "occurrence: 1", "occurrence: null", 1), strings.Replace(source, "    bend: 0\n", "", 1), strings.Replace(source, "    bend: 0", "    bend: 0\n    extra: false", 1)} {
		if _, e := parseDiagram("diagrams/d.yaml", []byte(bad), 5); e == nil {
			t.Fatal("bad portable route accepted")
		}
	}
}

func TestRoutingVisibilityAndUnrelatedInvalidRetention(t *testing.T) {
	m, b, ids, _ := placementFixture(t)
	ref := ReferenceAppearanceChange{DiagramID: ids.empty, ComponentID: ids.worker, Present: true}
	visible := reconciliationAccepted(t, m, b, nil, CandidateComposition{ArchitectureVersion: 4, References: []ReferenceAppearanceChange{ref}})
	address := RouteAddress{ids.empty, ids.worker, ids.records, "calls\nnext", 1}
	base := reconciliationAccepted(t, m, visible, nil, SetEdgeRoute(visible, CandidateComposition{}, address, &Route{-70}))
	invalid, _ := base.ChangeForAcceptedComponent(ids.gateway)
	invalid.Title = ""
	invalid.TitleChanged = true
	removed := CandidateComposition{ArchitectureVersion: 5, References: []ReferenceAppearanceChange{{DiagramID: ids.empty, ComponentID: ids.worker, Present: false}}}
	removed = ResetLostRouteVisibility(base, nil, []ComponentChange{invalid}, CandidateComposition{}, removed)
	if len(removed.EdgeRoutes) != 1 || removed.EdgeRoutes[0].Route != nil {
		t.Fatal("invalid pending visibility loss not materialized")
	}
	raw, _ := marshalChangeState([]ComponentChange{invalid}, removed)
	_, loaded, e := parseChangeState(raw)
	if e != nil {
		t.Fatal(e)
	}
	loaded.References = nil
	candidate, e := m.PrepareCandidate(t.Context(), base, nil, &loaded)
	if e != nil || candidate.Snapshot().routeAt(address) != nil {
		t.Fatalf("visibility resurrection: %v", e)
	}
	retained := ResetChangedRouteCounts(base, nil, []ComponentChange{invalid}, CandidateComposition{ArchitectureVersion: 5})
	retained = ResetLostRouteVisibility(base, nil, []ComponentChange{invalid}, CandidateComposition{}, retained)
	candidate, e = m.PrepareCandidate(t.Context(), base, nil, &retained)
	if e != nil || candidate.Snapshot().routeAt(address) == nil {
		t.Fatalf("unrelated invalid edit cleared route: %v", e)
	}
}

func TestRoutingReconciliationValuesAndLossBothDirections(t *testing.T) {
	for _, mode := range []string{"legacy_accepted", "legacy_proposed", "size_peer", "independent", "same", "conflict", "reset_conflict", "count_loss_a", "count_loss_p", "visibility_loss_a", "visibility_loss_p"} {
		t.Run(mode, func(t *testing.T) {
			m, b, ids, _ := placementFixture(t)
			b = reconciliationAccepted(t, m, b, nil, CandidateComposition{ArchitectureVersion: 4, References: []ReferenceAppearanceChange{{DiagramID: ids.empty, ComponentID: ids.worker, Present: true}}})
			address := RouteAddress{ids.root, ids.worker, ids.records, "calls\nnext", 2}
			if strings.HasPrefix(mode, "visibility") {
				address.DiagramID = ids.empty
			}
			if mode == "reset_conflict" {
				b = reconciliationAccepted(t, m, b, nil, SetEdgeRoute(b, CandidateComposition{}, address, &Route{70}))
			}
			ac := CandidateComposition{DiagramTitles: []DiagramTitleChange{{DiagramID: ids.root, Title: "A title"}}}
			pc := SetEdgeRoute(b, CandidateComposition{}, address, &Route{140})
			var ach, pch []ComponentChange
			switch mode {
			case "legacy_proposed":
				ac, pc = pc, ac
			case "size_peer":
				ac = SetNodeSize(b, ac, ids.root, ids.gateway, Size{444, 222})
			case "independent":
				other := address
				other.Occurrence = 1
				ac = SetEdgeRoute(b, ac, other, &Route{-90})
			case "same":
				ac = SetEdgeRoute(b, ac, address, &Route{140})
			case "conflict":
				ac = SetEdgeRoute(b, ac, address, &Route{180})
			case "reset_conflict":
				ac = SetEdgeRoute(b, ac, address, nil)
			case "count_loss_a", "count_loss_p":
				change, _ := b.ChangeForAcceptedComponent(ids.worker)
				change.RelationshipsChanged = true
				change.Relationships = change.Relationships[1:]
				ach = []ComponentChange{change}
				if mode == "count_loss_p" {
					ac, pc = pc, ac
					ach, pch = pch, ach
				}
			case "visibility_loss_a", "visibility_loss_p":
				ac.References = []ReferenceAppearanceChange{{DiagramID: ids.empty, ComponentID: ids.worker, Present: false}}
				if mode == "visibility_loss_p" {
					ac, pc = pc, ac
				}
			}
			a := reconciliationAccepted(t, m, b, ach, ac)
			p := reconciliationProposed(t, m, b, pch, pc)
			r, e := m.Reconcile(t.Context(), b, a, p, nil)
			if e != nil {
				t.Fatal(e)
			}
			loss := strings.Contains(mode, "loss")
			conflict := strings.Contains(mode, "conflict")
			if loss || conflict {
				kind := "route_value"
				choice := "manual"
				value := &ReconciliationValue{Route: nil}
				if loss {
					kind = "route_loss"
					choice = "clear"
					value = nil
				}
				if r.Status != "needs_resolution" || len(r.Conflicts) != 1 || r.Conflicts[0].Locator.Kind != kind {
					t.Fatalf("expected %s: %+v", kind, r)
				}
				resolution := ReconciliationResolution{Locator: r.Conflicts[0].Locator, Choice: choice, Value: value}
				raw, _ := json.Marshal(resolution)
				var decoded ReconciliationResolution
				if e := json.Unmarshal(raw, &decoded); e != nil {
					t.Fatal(e)
				}
				r, e = m.Reconcile(t.Context(), b, a, p, []ReconciliationResolution{decoded})
				if e != nil {
					t.Fatal(e)
				}
			}
			if r.Status != "ready" || r.Candidate == nil {
				t.Fatalf("not ready %+v", r)
			}
			route := r.Candidate.Snapshot().routeAt(address)
			if loss || conflict {
				if route != nil {
					t.Fatal("clear/default lost")
				}
			} else if route == nil || route.Bend != 140 {
				t.Fatal("custom lost")
			}
			if mode == "size_peer" {
				s, _ := r.Candidate.Snapshot().NodeSize(ids.root, ids.gateway)
				if s == nil || *s != (Size{444, 222}) {
					t.Fatal("v5 lost sized peer")
				}
			}
			replay, e := m.ConstructCandidate(t.Context(), a, r.Changes, r.Composition)
			if e != nil || replay.Tree() != r.Candidate.Tree() {
				t.Fatalf("residual replay %v", e)
			}
		})
	}
}
