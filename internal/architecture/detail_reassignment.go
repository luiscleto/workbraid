package architecture

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
)

// DetailReassignment records the final anchor of a base-existing child.
// Proposal-new children retain their anchor on the ordinary creation fact.
type DetailReassignment struct {
	DiagramID         string `json:"diagram_id" yaml:"diagram_id"`
	AnchorComponentID string `json:"anchor_component_id" yaml:"anchor_component_id"`
}

func (snapshot Snapshot) DiagramParent(diagramID string) (anchorID, homeID string, ok bool) {
	for _, current := range snapshot.diagrams {
		for _, appearance := range current.appearances {
			if appearance.hasDetailLink && appearance.detailDiagram.String() == diagramID {
				return appearance.component.String(), current.id.String(), true
			}
		}
	}
	return "", "", false
}

// DetailParentComponentIDs reads eligibility from one complete valid snapshot.
func (snapshot Snapshot) DetailParentComponentIDs(diagramID string) []string {
	anchor, _, ok := snapshot.DiagramParent(diagramID)
	if !ok || diagramID == snapshot.RootDiagramID() {
		return nil
	}
	diagrams := make(map[uuid.UUID]diagram, len(snapshot.diagrams))
	for _, current := range snapshot.diagrams {
		diagrams[current.id] = current
	}
	child := uuid.MustParse(diagramID)
	ids := []string{}
	for _, component := range snapshot.components {
		id := component.id.String()
		home, detail, hasHome := snapshot.ComponentHome(id)
		if id != anchor && hasHome && detail == "" && !diagramDescendsFrom(uuid.MustParse(home), child, diagrams) {
			ids = append(ids, id)
		}
	}
	return ids
}

// applyDetailAssignments is part of ConstructCandidate: all homes already
// have their final locations. Compute final child ownership before changing
// links, so explicit swaps never encounter temporary occupancy or detachment.
func applyDetailAssignments(base Snapshot, composition CandidateComposition, diagrams map[uuid.UUID]diagram, changed map[uuid.UUID]struct{}) error {
	anchors := make(map[string]string)
	for _, current := range base.diagrams {
		for _, appearance := range current.appearances {
			if appearance.hasDetailLink {
				anchors[appearance.detailDiagram.String()] = appearance.component.String()
			}
		}
	}
	seen := make(map[string]bool)
	for _, reassignment := range composition.DetailReassignments {
		if _, exists := anchors[reassignment.DiagramID]; !exists || seen[reassignment.DiagramID] || !canonicalUUID(reassignment.AnchorComponentID) {
			return &DiagramValidationError{DiagramID: reassignment.DiagramID, ComponentID: reassignment.AnchorComponentID, Field: "detail", Err: ErrDiagramHomeInvalid}
		}
		seen[reassignment.DiagramID] = true
		anchors[reassignment.DiagramID] = reassignment.AnchorComponentID
	}
	for _, addition := range composition.DetailDiagrams {
		anchors[addition.ID] = addition.AnchorComponentID
	}
	children := make([]string, 0, len(anchors))
	for id := range anchors {
		children = append(children, id)
	}
	sort.Strings(children)
	byAnchor := make(map[string]string)
	for _, child := range children {
		anchor := anchors[child]
		if other := byAnchor[anchor]; other != "" {
			return &DiagramValidationError{DiagramID: child, ComponentID: anchor, Field: "detail", Err: ErrDiagramHomeInvalid}
		}
		byAnchor[anchor] = child
	}
	for id, current := range diagrams {
		for index := range current.appearances {
			appearance := &current.appearances[index]
			if appearance.role != "home" {
				continue
			}
			child := byAnchor[appearance.component.String()]
			delete(byAnchor, appearance.component.String())
			if (child == "" && !appearance.hasDetailLink) || (child != "" && appearance.hasDetailLink && appearance.detailDiagram.String() == child) {
				continue
			}
			appearance.detailDiagram = uuid.Nil
			appearance.hasDetailLink = child != ""
			if child != "" {
				appearance.detailDiagram = uuid.MustParse(child)
			}
			changed[id] = struct{}{}
		}
		diagrams[id] = current
	}
	if len(byAnchor) != 0 {
		return fmt.Errorf("%w: detail anchor has no home", ErrDiagramHomeInvalid)
	}
	return nil
}
