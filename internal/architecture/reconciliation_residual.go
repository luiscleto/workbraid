package architecture

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// Derive the existing concrete authoring facts, relative to Accepted. This
// function writes no files and never supplies source blobs to the constructor.
func reconciliationResidual(accepted, proposed Snapshot, original, final reconciliationFacts) ([]ComponentChange, CandidateComposition, error) {
	a, p := snapshotReconciliationFacts(accepted), snapshotReconciliationFacts(proposed)
	changes := []ComponentChange{}
	composition := CandidateComposition{ArchitectureVersion: final.version}
	composition = shapeNoteResidual(accepted, a, final, composition)
	routeAddresses := map[RouteAddress]bool{}
	for k := range a.routes {
		routeAddresses[k] = true
	}
	for k := range final.routes {
		routeAddresses[k] = true
	}
	for k := range routeAddresses {
		before, bok := a.routes[k]
		after, aok := final.routes[k]
		if bok == aok && before == after {
			continue
		}
		var r *Route
		if aok {
			r = &after
		}
		composition.EdgeRoutes = append(composition.EdgeRoutes, EdgeRouteChange{k, r})
	}
	sortRouteFacts(composition.EdgeRoutes)
	pairs := map[reconciliationPair]bool{}
	for k := range a.positions {
		pairs[k] = true
	}
	for k := range final.positions {
		pairs[k] = true
	}
	positionKeys := make([]reconciliationPair, 0, len(pairs))
	for k := range pairs {
		positionKeys = append(positionKeys, k)
	}
	sort.Slice(positionKeys, func(i, j int) bool {
		if positionKeys[i].diagram == positionKeys[j].diagram {
			return positionKeys[i].component < positionKeys[j].component
		}
		return positionKeys[i].diagram < positionKeys[j].diagram
	})
	for _, k := range positionKeys {
		before, bok := a.positions[k]
		after, aok := final.positions[k]
		if bok == aok && before == after {
			continue
		}
		var p *Position
		if aok {
			p = &after
		}
		composition.NodePositions = append(composition.NodePositions, NodePositionChange{k.diagram, k.component, p})
	}
	sizePairs := map[reconciliationPair]bool{}
	for k := range a.sizes {
		sizePairs[k] = true
	}
	for k := range final.sizes {
		sizePairs[k] = true
	}
	sizeKeys := make([]reconciliationPair, 0, len(sizePairs))
	for k := range sizePairs {
		sizeKeys = append(sizeKeys, k)
	}
	sort.Slice(sizeKeys, func(i, j int) bool {
		if sizeKeys[i].diagram == sizeKeys[j].diagram {
			return sizeKeys[i].component < sizeKeys[j].component
		}
		return sizeKeys[i].diagram < sizeKeys[j].diagram
	})
	for _, k := range sizeKeys {
		before, bok := a.sizes[k]
		after, aok := final.sizes[k]
		if bok == aok && before == after {
			continue
		}
		var p *Size
		if aok {
			p = &after
		}
		composition.NodeSizes = append(composition.NodeSizes, NodeSizeChange{k.diagram, k.component, p})
	}
	used := map[string]bool{}
	acceptedPaths, proposedPaths := map[string]string{}, map[string]string{}
	for _, c := range accepted.components {
		acceptedPaths[c.id.String()] = c.path
		used[c.path] = true
	}
	for _, c := range proposed.components {
		proposedPaths[c.id.String()] = c.path
	}
	for _, d := range a.diagrams {
		used[d.path] = true
	}
	ids := map[string]bool{}
	for id := range final.components {
		ids[id] = true
	}
	for _, id := range sortedIDs(ids) {
		chosen := final.components[id]
		existing, exists := a.components[id]
		change := ComponentChange{ID: id, Title: chosen.Title, Description: chosen.Description, Path: acceptedPaths[id]}
		if !exists {
			if _, old := original.components[id]; old {
				return nil, composition, &ReconciliationError{"reconciliation_unsupported", "restore_component"}
			}
			_, ok := p.components[id]
			if !ok {
				return nil, composition, &ReconciliationError{"reconciliation_unsupported", "restore_source"}
			}
			allocated, err := reconciliationNewPath(proposedPaths[id], id, used)
			if err != nil {
				return nil, composition, err
			}
			change.New, change.Path = true, allocated
			composition.NewComponentHomes = append(composition.NewComponentHomes, NewComponentHome{ComponentID: id, DiagramID: final.homes[id]})
		}
		change.TitleChanged = chosen.Title != existing.Title
		change.DescriptionChanged = chosen.Description != existing.Description
		change.RelationshipsChanged = !sameOutgoingRelationships(id, a.relationships, final.relationships)
		change.Relationships = orderedReconciliationRelationships(id, existing.Relationships, p.components[id].Relationships, final.relationships)
		if change.New || change.TitleChanged || change.DescriptionChanged || change.RelationshipsChanged {
			changes = append(changes, change)
		}
		if exists && a.homes[id] != final.homes[id] {
			composition.HomeMoves = append(composition.HomeMoves, ComponentHomeMove{ComponentID: id, DiagramID: final.homes[id]})
		}
	}
	ids = map[string]bool{}
	for id := range final.diagrams {
		ids[id] = true
	}
	for _, id := range sortedIDs(ids) {
		d := final.diagrams[id]
		existing, exists := a.diagrams[id]
		if !exists {
			if _, old := original.diagrams[id]; old {
				return nil, composition, &ReconciliationError{"reconciliation_unsupported", "restore_diagram"}
			}
			created, ok := p.diagrams[id]
			if !ok {
				return nil, composition, &ReconciliationError{"reconciliation_unsupported", "restore_source"}
			}
			allocated, err := reconciliationNewPath(created.path, id, used)
			if err != nil {
				return nil, composition, err
			}
			composition.DetailDiagrams = append(composition.DetailDiagrams, DetailDiagramChange{ID: id, Path: allocated, Title: d.title, AnchorComponentID: final.anchors[id]})
		} else {
			if d.title != existing.title {
				composition.DiagramTitles = append(composition.DiagramTitles, DiagramTitleChange{DiagramID: id, Title: d.title})
			}
			if final.anchors[id] != a.anchors[id] {
				if id == a.root {
					return nil, composition, &ReconciliationError{"reconciliation_unsupported", "reassign_root"}
				}
				composition.DetailReassignments = append(composition.DetailReassignments, DetailReassignment{DiagramID: id, AnchorComponentID: final.anchors[id]})
			}
		}
	}
	refs := map[reconciliationPair]bool{}
	for k := range a.references {
		refs[k] = true
	}
	for k := range final.references {
		refs[k] = true
	}
	keys := make([]reconciliationPair, 0, len(refs))
	for k := range refs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].diagram == keys[j].diagram {
			return keys[i].component < keys[j].component
		}
		return keys[i].diagram < keys[j].diagram
	})
	for _, k := range keys {
		if a.references[k] != final.references[k] {
			composition.References = append(composition.References, ReferenceAppearanceChange{DiagramID: k.diagram, ComponentID: k.component, Present: final.references[k]})
		}
	}
	return changes, composition, nil
}

func reconciliationNewPath(original, id string, used map[string]bool) (string, error) {
	if !used[original] {
		used[original] = true
		return original, nil
	}
	ext := path.Ext(original)
	stem := strings.TrimSuffix(original, ext) + "-" + id
	// At most len(used) existing paths can occupy these distinct names.
	for suffix := 0; suffix <= len(used); suffix++ {
		candidate := stem + ext
		if suffix > 0 {
			candidate = fmt.Sprintf("%s-%d%s", stem, suffix, ext)
		}
		if !used[candidate] {
			used[candidate] = true
			return candidate, nil
		}
	}
	return "", &ReconciliationError{"reconciliation_unsupported", "restore_source"}
}

func sameOutgoingRelationships(id string, a, b map[reconciliationRelationship]int) bool {
	for k, v := range a {
		if k.source == id && b[k] != v {
			return false
		}
	}
	for k, v := range b {
		if k.source == id && a[k] != v {
			return false
		}
	}
	return true
}

func orderedReconciliationRelationships(id string, accepted, proposed []AuthoringRelationship, counts map[reconciliationRelationship]int) []AuthoringRelationship {
	remaining := map[reconciliationRelationship]int{}
	for k, v := range counts {
		if k.source == id {
			remaining[k] = v
		}
	}
	result := []AuthoringRelationship{}
	appendSurvivors := func(values []AuthoringRelationship) {
		for _, r := range values {
			k := reconciliationRelationship{id, r.TargetID, r.Label}
			if remaining[k] > 0 {
				result = append(result, r)
				remaining[k]--
			}
		}
	}
	appendSurvivors(accepted)
	appendSurvivors(proposed)
	keys := map[reconciliationRelationship]bool{}
	for k, n := range remaining {
		if n > 0 {
			keys[k] = true
		}
	}
	for _, k := range sortedRelationships(keys) {
		for i := 0; i < remaining[k]; i++ {
			result = append(result, AuthoringRelationship{TargetID: k.target, Label: k.label})
		}
	}
	return result
}
