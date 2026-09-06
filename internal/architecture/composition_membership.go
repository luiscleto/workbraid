package architecture

import (
	"fmt"
	"github.com/google/uuid"
)

// Shared composition steps used by the one constructor and the ordinary
// mutation's targeted route-loss observation, including invalid row edits.
func applyComponentHomes(changes []ComponentChange, composition CandidateComposition, diagrams map[uuid.UUID]diagram, changedDiagrams map[uuid.UUID]struct{}) error {
	seenHomes := make(map[string]struct{}, len(composition.NewComponentHomes))
	for _, home := range composition.NewComponentHomes {
		if _, duplicate := seenHomes[home.ComponentID]; duplicate {
			return fmt.Errorf("%w: new Component has more than one home", ErrInvalid)
		}
		seenHomes[home.ComponentID] = struct{}{}
		componentID, componentErr := uuid.Parse(home.ComponentID)
		diagramID, diagramErr := uuid.Parse(home.DiagramID)
		if componentErr != nil || diagramErr != nil {
			return fmt.Errorf("%w: new Component home is invalid", ErrInvalid)
		}
		changeFound := false
		for _, change := range changes {
			if change.New && change.ID == home.ComponentID {
				changeFound = true
				break
			}
		}
		if !changeFound {
			return fmt.Errorf("%w: home does not belong to a new Component", ErrInvalid)
		}
		current, exists := diagrams[diagramID]
		if !exists {
			return fmt.Errorf("%w: home Diagram does not exist", ErrInvalid)
		}
		for _, appearance := range current.appearances {
			if appearance.component == componentID {
				return fmt.Errorf("%w: new Component already appears in its home Diagram", ErrInvalid)
			}
		}
		current.appearances = append(current.appearances, diagramAppearance{component: componentID, role: "home"})
		diagrams[diagramID] = current
		changedDiagrams[diagramID] = struct{}{}
	}
	for _, change := range changes {
		if !change.New {
			continue
		}
		if _, exists := seenHomes[change.ID]; !exists {
			return fmt.Errorf("%w: new Component is missing its home Diagram", ErrInvalid)
		}
	}
	for _, move := range composition.HomeMoves {
		componentID, componentErr := uuid.Parse(move.ComponentID)
		destinationID, diagramErr := uuid.Parse(move.DiagramID)
		if componentErr != nil || diagramErr != nil {
			return &DiagramValidationError{ComponentID: move.ComponentID, DiagramID: move.DiagramID, Field: "home", Err: ErrDiagramHomeInvalid}
		}
		destination, destinationExists := diagrams[destinationID]
		if !destinationExists {
			return &DiagramValidationError{ComponentID: move.ComponentID, DiagramID: move.DiagramID, Field: "home", Err: ErrDiagramHomeInvalid}
		}
		var sourceID uuid.UUID
		var home diagramAppearance
		found := false
		for id, current := range diagrams {
			for _, appearance := range current.appearances {
				if appearance.component == componentID && appearance.role == "home" {
					sourceID, home, found = id, appearance, true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return &DiagramValidationError{ComponentID: move.ComponentID, DiagramID: move.DiagramID, Field: "home", Err: ErrDiagramHomeInvalid}
		}
		if sourceID == destinationID {
			continue
		}
		source := diagrams[sourceID]
		filtered := source.appearances[:0:0]
		for _, appearance := range source.appearances {
			if appearance.component != componentID || appearance.role != "home" {
				filtered = append(filtered, appearance)
			}
		}
		source.appearances = filtered
		diagrams[sourceID] = source
		changedDiagrams[sourceID] = struct{}{}

		converted := false
		for index := range destination.appearances {
			if destination.appearances[index].component == componentID {
				if destination.appearances[index].role != "reference" || destination.appearances[index].hasDetailLink {
					return &DiagramValidationError{ComponentID: move.ComponentID, DiagramID: move.DiagramID, Field: "home", Err: ErrDiagramHomeInvalid}
				}
				destination.appearances[index] = home
				converted = true
				break
			}
		}
		if !converted {
			destination.appearances = append(destination.appearances, home)
		}
		diagrams[destinationID] = destination
		changedDiagrams[destinationID] = struct{}{}
	}
	return nil
}
func applyComponentReferences(composition CandidateComposition, diagrams map[uuid.UUID]diagram, changedDiagrams map[uuid.UUID]struct{}, candidateIDs map[string]struct{}) error {
	seenReferences := make(map[string]struct{}, len(composition.References))
	for _, change := range composition.References {
		componentID, componentErr := uuid.Parse(change.ComponentID)
		diagramID, diagramErr := uuid.Parse(change.DiagramID)
		key := change.DiagramID + "\x00" + change.ComponentID
		if componentErr != nil || diagramErr != nil {
			return &DiagramValidationError{ComponentID: change.ComponentID, DiagramID: change.DiagramID, Field: "reference", Err: ErrDiagramHomeInvalid}
		}
		if _, duplicate := seenReferences[key]; duplicate {
			return &DiagramValidationError{ComponentID: change.ComponentID, DiagramID: change.DiagramID, Field: "reference", Err: ErrDiagramHomeInvalid}
		}
		seenReferences[key] = struct{}{}
		current, exists := diagrams[diagramID]
		if !exists {
			return &DiagramValidationError{ComponentID: change.ComponentID, DiagramID: change.DiagramID, Field: "reference", Err: ErrDiagramHomeInvalid}
		}
		if _, exists := candidateIDs[change.ComponentID]; !exists {
			return &DiagramValidationError{ComponentID: change.ComponentID, DiagramID: change.DiagramID, Field: "reference", Err: ErrDiagramHomeInvalid}
		}
		appearanceIndex := -1
		for index, appearance := range current.appearances {
			if appearance.component == componentID {
				appearanceIndex = index
				break
			}
		}
		if change.Present {
			if appearanceIndex >= 0 {
				// A home subsumes any requested reference state. A reference is
				// already the requested final state.
				if current.appearances[appearanceIndex].role == "home" || current.appearances[appearanceIndex].role == "reference" {
					continue
				}
				return &DiagramValidationError{ComponentID: change.ComponentID, DiagramID: change.DiagramID, Field: "reference", Err: ErrDiagramHomeInvalid}
			}
			homeDiagram := uuid.Nil
			for candidateDiagramID, candidateDiagram := range diagrams {
				for _, appearance := range candidateDiagram.appearances {
					if appearance.component == componentID && appearance.role == "home" {
						homeDiagram = candidateDiagramID
					}
				}
			}
			if homeDiagram == uuid.Nil || homeDiagram == diagramID {
				return &DiagramValidationError{ComponentID: change.ComponentID, DiagramID: change.DiagramID, Field: "reference", Err: ErrDiagramHomeInvalid}
			}
			current.appearances = append(current.appearances, diagramAppearance{component: componentID, role: "reference"})
		} else if appearanceIndex >= 0 && current.appearances[appearanceIndex].role == "reference" {
			current.appearances = append(current.appearances[:appearanceIndex:appearanceIndex], current.appearances[appearanceIndex+1:]...)
		}
		diagrams[diagramID] = current
		changedDiagrams[diagramID] = struct{}{}
	}
	return nil
}
