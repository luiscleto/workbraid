package web

import (
	"fmt"
	"path/filepath"
	"sort"

	"workbraid/internal/architecture"
)

type snapshotProjectionResponse struct {
	Revision        string              `json:"revision"`
	FormatVersion   int                 `json:"format_version"`
	ComponentCount  int                 `json:"component_count"`
	ComponentTitles []string            `json:"component_titles"`
	Components      []componentResponse `json:"components"`
	RootDiagramID   string              `json:"root_diagram_id,omitempty"`
	Diagrams        []diagramResponse   `json:"diagrams,omitempty"`
}

type diagramResponse struct {
	ID                      string                        `json:"id"`
	Title                   string                        `json:"title"`
	Filename                string                        `json:"filename"`
	Depth                   int                           `json:"depth"`
	Context                 string                        `json:"context,omitempty"`
	ParentDiagramID         string                        `json:"parent_diagram_id,omitempty"`
	ParentAnchorComponentID string                        `json:"parent_anchor_component_id,omitempty"`
	Breadcrumbs             []diagramBreadcrumbResponse   `json:"breadcrumbs"`
	Appearances             []diagramAppearanceResponse   `json:"appearances"`
	Boundaries              []diagramBoundaryResponse     `json:"boundaries"`
	Relationships           []diagramRelationshipResponse `json:"relationships"`
}

type diagramBreadcrumbResponse struct {
	ID                     string `json:"id"`
	Title                  string `json:"title"`
	FocusAnchorComponentID string `json:"focus_anchor_component_id,omitempty"`
}

type diagramAppearanceResponse struct {
	Size               *architecture.Size     `json:"size"`
	DisplaySize        *architecture.Size     `json:"display_size"`
	SizeSource         string                 `json:"size_source"`
	DisplayPosition    *architecture.Position `json:"display_position"`
	PositionSource     string                 `json:"position_source"`
	Position           *architecture.Position `json:"position"`
	ComponentID        string                 `json:"component_id"`
	Role               string                 `json:"role"`
	DetailDiagramID    string                 `json:"detail_diagram_id,omitempty"`
	DetailDiagramTitle string                 `json:"detail_diagram_title,omitempty"`
}

type diagramBoundaryResponse struct {
	Size             *architecture.Size     `json:"size"`
	DisplaySize      *architecture.Size     `json:"display_size"`
	SizeSource       string                 `json:"size_source"`
	Position         *architecture.Position `json:"position"`
	DisplayPosition  *architecture.Position `json:"display_position"`
	PositionSource   string                 `json:"position_source"`
	Key              string                 `json:"key"`
	ComponentID      string                 `json:"component_id"`
	Title            string                 `json:"title"`
	Context          string                 `json:"context,omitempty"`
	HomeDiagramID    string                 `json:"home_diagram_id"`
	HomeDiagramTitle string                 `json:"home_diagram_title"`
}

type diagramRelationshipResponse struct {
	Routing                 architecture.RouteProjection `json:"routing"`
	Key                     string                       `json:"key"`
	SourceNodeKey           string                       `json:"source_node_key"`
	TargetNodeKey           string                       `json:"target_node_key"`
	SourceComponentID       string                       `json:"source_component_id"`
	TargetComponentID       string                       `json:"target_component_id"`
	Label                   string                       `json:"label"`
	sourceRelationshipIndex int
}

type reviewComparisonResponse struct {
	EdgeRoutes    []reviewEdgeRouteChange            `json:"edge_routes"`
	NodeSizes     []reviewNodeSizeChange             `json:"node_sizes"`
	NodePositions []reviewNodePositionChange         `json:"node_positions"`
	Components    []reviewComponentChangeResponse    `json:"components"`
	Relationships []reviewRelationshipChangeResponse `json:"relationships"`
	Diagrams      []reviewDiagramChangeResponse      `json:"diagrams,omitempty"`
	Appearances   []reviewAppearanceChangeResponse   `json:"appearances,omitempty"`
}

type reviewNodePositionChange struct {
	BeforeSource string                 `json:"before_source"`
	WithSource   string                 `json:"with_source"`
	DiagramID    string                 `json:"diagram_id"`
	ComponentID  string                 `json:"component_id"`
	Before       *architecture.Position `json:"before"`
	With         *architecture.Position `json:"with"`
	Path         string                 `json:"path"`
}

type reviewNodeSizeChange struct {
	BeforeSource string             `json:"before_source"`
	WithSource   string             `json:"with_source"`
	DiagramID    string             `json:"diagram_id"`
	ComponentID  string             `json:"component_id"`
	Before       *architecture.Size `json:"before"`
	With         *architecture.Size `json:"with"`
	Path         string             `json:"path"`
}

type reviewDiagramChangeResponse struct {
	DiagramID string `json:"diagram_id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Path      string `json:"path"`
}

type reviewAppearanceChangeResponse struct {
	DiagramID       string `json:"diagram_id"`
	ComponentID     string `json:"component_id"`
	Role            string `json:"role"`
	Status          string `json:"status"`
	Side            string `json:"side"`
	DetailDiagramID string `json:"detail_diagram_id,omitempty"`
	Path            string `json:"path"`
}

type reviewComponentChangeResponse struct {
	ComponentID string `json:"component_id"`
	Status      string `json:"status"`
	Path        string `json:"path"`
}

type reviewRelationshipChangeResponse struct {
	Key                     string                                        `json:"key"`
	BeforeKey               string                                        `json:"before_key,omitempty"`
	SourceID                string                                        `json:"source_id"`
	TargetID                string                                        `json:"target_id"`
	SourceTitle             string                                        `json:"source_title"`
	TargetTitle             string                                        `json:"target_title"`
	Label                   string                                        `json:"label"`
	Status                  string                                        `json:"status"`
	Path                    string                                        `json:"path"`
	Occurrence              int                                           `json:"occurrence"`
	DiagramProjections      []reviewRelationshipDiagramProjectionResponse `json:"diagram_projections,omitempty"`
	sourceRelationshipIndex int
}

type reviewRelationshipDiagramProjectionResponse struct {
	Side          string `json:"side"`
	DiagramID     string `json:"diagram_id"`
	Key           string `json:"key"`
	SourceNodeKey string `json:"source_node_key"`
	TargetNodeKey string `json:"target_node_key"`
}

type reviewRelationshipFact struct {
	sourceID string
	targetID string
	label    string
}

func projectSnapshot(snapshot architecture.Snapshot, relationshipKeyPrefix string) snapshotProjectionResponse {
	authored := snapshot.AuthoringComponents()
	components := make([]componentResponse, len(authored))
	for componentIndex, component := range authored {
		relationships := make([]relationshipResponse, len(component.Relationships))
		for relationshipIndex, relationship := range component.Relationships {
			relationships[relationshipIndex] = relationshipResponse{
				TargetID: relationship.TargetID,
				Label:    relationship.Label,
			}
			if relationshipKeyPrefix != "" {
				relationships[relationshipIndex].ProjectionKey = reviewRelationshipProjectionKey(
					relationshipKeyPrefix, component.ID, relationshipIndex,
				)
			}
		}
		components[componentIndex] = componentResponse{
			ID: component.ID, Title: component.Title, Description: component.Description,
			Filename: component.Filename, Relationships: relationships,
		}
		if source, exists := snapshot.ComponentMarkdownSource(component.ID); exists {
			components[componentIndex].MarkdownSource = string(source)
		}
	}
	return snapshotProjectionResponse{
		Revision:        snapshot.Revision(),
		FormatVersion:   snapshot.FormatVersion(),
		ComponentCount:  snapshot.ComponentCount(),
		ComponentTitles: snapshot.ComponentTitles(),
		Components:      components,
		RootDiagramID:   snapshot.RootDiagramID(),
		Diagrams:        projectDiagrams(snapshot),
	}
}

func projectDiagrams(snapshot architecture.Snapshot) []diagramResponse {
	projected := snapshot.DiagramProjections()
	if len(projected) == 0 {
		return nil
	}
	componentTitles := make(map[string]string, snapshot.ComponentCount())
	componentContexts := make(map[string]string, snapshot.ComponentCount())
	componentTitleCounts := make(map[string]int, snapshot.ComponentCount())
	for _, component := range snapshot.AuthoringComponents() {
		componentTitles[component.ID] = component.Title
		componentContexts[component.ID] = component.Filename
		componentTitleCounts[component.Title]++
	}
	titleCounts := make(map[string]int, len(projected))
	for _, diagram := range projected {
		titleCounts[diagram.Title]++
	}
	result := make([]diagramResponse, len(projected))
	for index, diagram := range projected {
		value := diagramResponse{
			ID: diagram.ID, Title: diagram.Title, Filename: diagram.Filename, Depth: diagram.Depth, ParentDiagramID: diagram.ParentDiagramID,
			ParentAnchorComponentID: diagram.ParentAnchorComponentID,
			Breadcrumbs:             make([]diagramBreadcrumbResponse, len(diagram.Breadcrumbs)),
			Appearances:             make([]diagramAppearanceResponse, len(diagram.Appearances)),
			Boundaries:              make([]diagramBoundaryResponse, len(diagram.Boundaries)),
			Relationships:           make([]diagramRelationshipResponse, len(diagram.Relationships)),
		}
		if titleCounts[diagram.Title] > 1 {
			if diagram.ParentAnchorComponentID == "" {
				value.Context = "Main diagram"
			} else {
				value.Context = "Inside " + componentTitles[diagram.ParentAnchorComponentID]
				if componentTitleCounts[componentTitles[diagram.ParentAnchorComponentID]] > 1 {
					value.Context += " — " + componentContexts[diagram.ParentAnchorComponentID]
				}
			}
		}
		for itemIndex, breadcrumb := range diagram.Breadcrumbs {
			value.Breadcrumbs[itemIndex] = diagramBreadcrumbResponse{ID: breadcrumb.ID, Title: breadcrumb.Title, FocusAnchorComponentID: breadcrumb.FocusAnchorComponentID}
		}
		for itemIndex, appearance := range diagram.Appearances {
			value.Appearances[itemIndex] = diagramAppearanceResponse{
				DisplayPosition: appearance.DisplayPosition, PositionSource: appearance.PositionSource,
				Size: appearance.Size, DisplaySize: appearance.DisplaySize, SizeSource: appearance.SizeSource,
				Position:    appearance.Position,
				ComponentID: appearance.ComponentID, Role: appearance.Role,
				DetailDiagramID: appearance.DetailDiagramID, DetailDiagramTitle: appearance.DetailDiagramTitle,
			}
		}
		for itemIndex, boundary := range diagram.Boundaries {
			value.Boundaries[itemIndex] = diagramBoundaryResponse{
				Position: boundary.Position, DisplayPosition: boundary.DisplayPosition, PositionSource: boundary.PositionSource,
				Size: boundary.Size, DisplaySize: boundary.DisplaySize, SizeSource: boundary.SizeSource,
				Key: boundary.Key, ComponentID: boundary.ComponentID, Title: boundary.Title, Context: boundary.Context,
				HomeDiagramID: boundary.HomeDiagramID, HomeDiagramTitle: boundary.HomeDiagramTitle,
			}
		}
		for itemIndex, relationship := range diagram.Relationships {
			value.Relationships[itemIndex] = diagramRelationshipResponse{
				Routing: relationship.Routing,
				Key:     relationship.Key, SourceNodeKey: relationship.SourceNodeKey, TargetNodeKey: relationship.TargetNodeKey,
				SourceComponentID: relationship.SourceComponentID, TargetComponentID: relationship.TargetComponentID, Label: relationship.Label,
				sourceRelationshipIndex: relationship.SourceRelationshipIndex,
			}
		}
		result[index] = value
	}
	return result
}

// captureReviewPresentation projects and compares one immutable bound pair.
// Callers hold Handler.stateMutex while verifying the review binding and
// invoking this function, so every returned value belongs to one generation.
func captureReviewPresentation(base, candidate architecture.Snapshot) (snapshotProjectionResponse, snapshotProjectionResponse, reviewComparisonResponse) {
	before := projectSnapshot(base, "before")
	withChanges := projectSnapshot(candidate, "with")
	comparison := compareReviewProjections(before.Components, withChanges.Components)
	comparison.Diagrams, comparison.Appearances = compareDiagramProjections(before.Diagrams, withChanges.Diagrams)
	attachDiagramRelationshipProjections(comparison.Relationships, before.Diagrams, withChanges.Diagrams)
	comparison.NodePositions = []reviewNodePositionChange{}
	type pair struct{ diagram, component string }
	old, current := map[pair]*architecture.Position{}, map[pair]*architecture.Position{}
	paths := map[string]string{}
	for _, d := range before.Diagrams {
		paths[d.ID] = "diagrams/" + d.Filename
		for _, a := range d.Appearances {
			old[pair{d.ID, a.ComponentID}] = a.Position
		}
		for _, b := range d.Boundaries {
			old[pair{d.ID, b.ComponentID}] = b.Position
		}
	}
	for _, d := range withChanges.Diagrams {
		paths[d.ID] = "diagrams/" + d.Filename
		for _, a := range d.Appearances {
			current[pair{d.ID, a.ComponentID}] = a.Position
		}
		for _, b := range d.Boundaries {
			current[pair{d.ID, b.ComponentID}] = b.Position
		}
	}
	keys := map[pair]bool{}
	for k := range old {
		keys[k] = true
	}
	for k := range current {
		keys[k] = true
	}
	for k := range keys {
		a, b := old[k], current[k]
		if a == nil && b == nil || a != nil && b != nil && *a == *b {
			continue
		}
		source := func(positions map[pair]*architecture.Position) string {
			p, visible := positions[k]
			if !visible {
				return "not_applicable"
			}
			if p == nil {
				return "derived"
			}
			return "stored"
		}
		comparison.NodePositions = append(comparison.NodePositions, reviewNodePositionChange{DiagramID: k.diagram, ComponentID: k.component, Before: a, With: b, Path: paths[k.diagram], BeforeSource: source(old), WithSource: source(current)})
	}
	sort.Slice(comparison.NodePositions, func(i, j int) bool {
		a, b := comparison.NodePositions[i], comparison.NodePositions[j]
		if a.DiagramID == b.DiagramID {
			return a.ComponentID < b.ComponentID
		}
		return a.DiagramID < b.DiagramID
	})
	comparison.NodeSizes = compareSizes(before, withChanges)
	comparison.EdgeRoutes = compareRoutes(before, withChanges)
	return before, withChanges, comparison
}

type reviewEdgeRouteChange struct {
	architecture.RouteAddress
	Before      *architecture.Route `json:"before"`
	With        *architecture.Route `json:"with"`
	BeforeState string              `json:"before_state"`
	WithState   string              `json:"with_state"`
	Path        string              `json:"path"`
}

func compareRoutes(before, with snapshotProjectionResponse) []reviewEdgeRouteChange {
	old, current := map[architecture.RouteAddress]*architecture.Route{}, map[architecture.RouteAddress]*architecture.Route{}
	paths := map[string]string{}
	for _, side := range []struct {
		s      snapshotProjectionResponse
		values map[architecture.RouteAddress]*architecture.Route
	}{{before, old}, {with, current}} {
		for _, d := range side.s.Diagrams {
			paths[d.ID] = "diagrams/" + d.Filename
			for _, r := range d.Relationships {
				side.values[r.Routing.RouteAddress] = r.Routing.Route
			}
		}
	}
	keys := map[architecture.RouteAddress]bool{}
	for k := range old {
		keys[k] = true
	}
	for k := range current {
		keys[k] = true
	}
	out := []reviewEdgeRouteChange{}
	state := func(v *architecture.Route, exists bool) string {
		if !exists {
			return "not_applicable"
		}
		if v == nil {
			return "default"
		}
		return "custom"
	}
	for k := range keys {
		a, ao := old[k]
		b, bo := current[k]
		if a == nil && b == nil || a != nil && b != nil && *a == *b {
			continue
		}
		out = append(out, reviewEdgeRouteChange{k, a, b, state(a, ao), state(b, bo), paths[k.DiagramID]})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.DiagramID != b.DiagramID {
			return a.DiagramID < b.DiagramID
		}
		if a.SourceID != b.SourceID {
			return a.SourceID < b.SourceID
		}
		if a.TargetID != b.TargetID {
			return a.TargetID < b.TargetID
		}
		if a.Label != b.Label {
			return a.Label < b.Label
		}
		return a.Occurrence < b.Occurrence
	})
	return out
}

func compareSizes(before, with snapshotProjectionResponse) []reviewNodeSizeChange {
	type pair struct{ diagram, component string }
	type side struct {
		size   *architecture.Size
		source string
	}
	old, current := map[pair]side{}, map[pair]side{}
	paths := map[string]string{}
	collect := func(diagrams []diagramResponse, values map[pair]side) {
		for _, d := range diagrams {
			paths[d.ID] = "diagrams/" + d.Filename
			for _, a := range d.Appearances {
				values[pair{d.ID, a.ComponentID}] = side{a.DisplaySize, a.SizeSource}
			}
			for _, b := range d.Boundaries {
				values[pair{d.ID, b.ComponentID}] = side{b.DisplaySize, b.SizeSource}
			}
		}
	}
	collect(before.Diagrams, old)
	collect(with.Diagrams, current)
	keys := map[pair]bool{}
	for k := range old {
		keys[k] = true
	}
	for k := range current {
		keys[k] = true
	}
	result := []reviewNodeSizeChange{}
	for k := range keys {
		a, aok := old[k]
		b, bok := current[k]
		if !aok {
			a.source = "not_applicable"
		}
		if !bok {
			b.source = "not_applicable"
		}
		if a.source == b.source && (a.size == nil && b.size == nil || a.size != nil && b.size != nil && *a.size == *b.size) {
			continue
		}
		result = append(result, reviewNodeSizeChange{BeforeSource: a.source, WithSource: b.source, DiagramID: k.diagram, ComponentID: k.component, Before: a.size, With: b.size, Path: paths[k.diagram]})
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.DiagramID != b.DiagramID {
			return a.DiagramID < b.DiagramID
		}
		return a.ComponentID < b.ComponentID
	})
	return result
}

func attachDiagramRelationshipProjections(changes []reviewRelationshipChangeResponse, before, withChanges []diagramResponse) {
	for index := range changes {
		diagrams := withChanges
		side := "with"
		if changes[index].Status == "removed" {
			diagrams = before
			side = "before"
		}
		for _, diagram := range diagrams {
			for _, relationship := range diagram.Relationships {
				if relationship.SourceComponentID != changes[index].SourceID ||
					relationship.TargetComponentID != changes[index].TargetID ||
					relationship.Label != changes[index].Label ||
					relationship.sourceRelationshipIndex != changes[index].sourceRelationshipIndex {
					continue
				}
				changes[index].DiagramProjections = append(changes[index].DiagramProjections, reviewRelationshipDiagramProjectionResponse{
					Side: side, DiagramID: diagram.ID, Key: relationship.Key,
					SourceNodeKey: relationship.SourceNodeKey, TargetNodeKey: relationship.TargetNodeKey,
				})
			}
		}
	}
}

func compareDiagramProjections(before, withChanges []diagramResponse) ([]reviewDiagramChangeResponse, []reviewAppearanceChangeResponse) {
	beforeByID := make(map[string]diagramResponse, len(before))
	withByID := make(map[string]diagramResponse, len(withChanges))
	for _, current := range before {
		beforeByID[current.ID] = current
	}
	for _, current := range withChanges {
		withByID[current.ID] = current
	}
	var diagrams []reviewDiagramChangeResponse
	var appearances []reviewAppearanceChangeResponse
	for _, current := range withChanges {
		base, exists := beforeByID[current.ID]
		status := ""
		if !exists {
			status = "added"
		} else if base.Title != current.Title {
			status = "title_changed"
		}
		if status != "" {
			diagrams = append(diagrams, reviewDiagramChangeResponse{DiagramID: current.ID, Title: current.Title, Status: status, Path: "diagrams/" + current.Filename})
		}
		beforeAppearances := appearanceSet(base.Appearances)
		for _, appearance := range current.Appearances {
			key := appearance.ComponentID + "\x00" + appearance.Role + "\x00" + appearance.DetailDiagramID
			if _, unchanged := beforeAppearances[key]; !unchanged {
				status := "added"
				side := "with_changes"
				detailDiagramID := ""
				if appearanceIdentityExists(base.Appearances, appearance) {
					status = "detail_changed"
					detailDiagramID = appearance.DetailDiagramID
					if detailDiagramID == "" {
						side = "before"
						for _, previous := range base.Appearances {
							if previous.ComponentID == appearance.ComponentID && previous.Role == appearance.Role {
								detailDiagramID = previous.DetailDiagramID
								break
							}
						}
					}
				}
				appearances = append(appearances, reviewAppearanceChangeResponse{DiagramID: current.ID, ComponentID: appearance.ComponentID, Role: appearance.Role, Status: status, Side: side, DetailDiagramID: detailDiagramID, Path: "diagrams/" + current.Filename})
			}
		}
	}
	for _, current := range before {
		candidate := withByID[current.ID]
		withAppearances := appearanceSet(candidate.Appearances)
		for _, appearance := range current.Appearances {
			key := appearance.ComponentID + "\x00" + appearance.Role + "\x00" + appearance.DetailDiagramID
			if _, unchanged := withAppearances[key]; !unchanged {
				if appearanceIdentityExists(candidate.Appearances, appearance) {
					continue
				}
				appearances = append(appearances, reviewAppearanceChangeResponse{DiagramID: current.ID, ComponentID: appearance.ComponentID, Role: appearance.Role, Status: "removed", Side: "before", Path: "diagrams/" + current.Filename})
			}
		}
	}
	return diagrams, appearances
}

func appearanceIdentityExists(values []diagramAppearanceResponse, target diagramAppearanceResponse) bool {
	for _, appearance := range values {
		if appearance.ComponentID == target.ComponentID && appearance.Role == target.Role {
			return true
		}
	}
	return false
}

func appearanceSet(values []diagramAppearanceResponse) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, appearance := range values {
		result[appearance.ComponentID+"\x00"+appearance.Role+"\x00"+appearance.DetailDiagramID] = struct{}{}
	}
	return result
}

// compareReviewProjections is deliberately concrete to the Review changes
// presentation. It is not a canonical or reusable semantic diff.
func compareReviewProjections(before, withChanges []componentResponse) reviewComparisonResponse {
	beforeByID := make(map[string]componentResponse, len(before))
	for _, component := range before {
		beforeByID[component.ID] = component
	}

	componentChanges := make([]reviewComponentChangeResponse, 0)
	for _, component := range withChanges {
		base, exists := beforeByID[component.ID]
		status := ""
		switch {
		case !exists:
			status = "added"
		case base.Title != component.Title || base.Description != component.Description:
			status = "content_changed"
		}
		if status != "" {
			componentChanges = append(componentChanges, reviewComponentChangeResponse{
				ComponentID: component.ID,
				Status:      status,
				Path:        canonicalComponentPath(component.Filename),
			})
		}
	}

	beforeCounts := relationshipFactCounts(before)
	withCounts := relationshipFactCounts(withChanges)
	relationshipChanges := make([]reviewRelationshipChangeResponse, 0)
	withByID := make(map[string]componentResponse, len(withChanges))
	for _, component := range withChanges {
		withByID[component.ID] = component
	}
	for _, component := range withChanges {
		seen := make(map[reviewRelationshipFact]int)
		for relationshipIndex, relationship := range component.Relationships {
			fact := reviewRelationshipFact{sourceID: component.ID, targetID: relationship.TargetID, label: relationship.Label}
			seen[fact]++
			if seen[fact] <= beforeCounts[fact] {
				continue
			}
			relationshipChanges = append(relationshipChanges, reviewRelationshipChangeResponse{
				Key: relationship.ProjectionKey, SourceID: component.ID, TargetID: relationship.TargetID,
				SourceTitle: component.Title, TargetTitle: withByID[relationship.TargetID].Title,
				Label: relationship.Label, Status: "added", Path: canonicalComponentPath(component.Filename),
				Occurrence: seen[fact], sourceRelationshipIndex: relationshipIndex,
			})
		}
	}
	for _, component := range before {
		seen := make(map[reviewRelationshipFact]int)
		for relationshipIndex, relationship := range component.Relationships {
			fact := reviewRelationshipFact{sourceID: component.ID, targetID: relationship.TargetID, label: relationship.Label}
			seen[fact]++
			if seen[fact] <= withCounts[fact] {
				continue
			}
			relationshipChanges = append(relationshipChanges, reviewRelationshipChangeResponse{
				Key:       reviewRelationshipProjectionKey("removed", component.ID, relationshipIndex),
				BeforeKey: relationship.ProjectionKey, SourceID: component.ID, TargetID: relationship.TargetID,
				SourceTitle: component.Title, TargetTitle: beforeByID[relationship.TargetID].Title,
				Label: relationship.Label, Status: "removed", Path: canonicalComponentPath(component.Filename),
				Occurrence: seen[fact], sourceRelationshipIndex: relationshipIndex,
			})
		}
	}

	return reviewComparisonResponse{Components: componentChanges, Relationships: relationshipChanges}
}

func relationshipFactCounts(components []componentResponse) map[reviewRelationshipFact]int {
	counts := make(map[reviewRelationshipFact]int)
	for _, component := range components {
		for _, relationship := range component.Relationships {
			counts[reviewRelationshipFact{sourceID: component.ID, targetID: relationship.TargetID, label: relationship.Label}]++
		}
	}
	return counts
}

func reviewRelationshipProjectionKey(side, sourceID string, relationshipIndex int) string {
	return fmt.Sprintf("review:%s:%s:%d", side, sourceID, relationshipIndex)
}

func canonicalComponentPath(filename string) string {
	return filepath.ToSlash(filepath.Join("components", filename))
}
