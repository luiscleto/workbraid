package architecture

import (
	"errors"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"go.yaml.in/yaml/v4"
)

const PositionLimit = 100000

func ValidPosition(p Position) bool {
	return p.X >= -PositionLimit && p.X <= PositionLimit && p.Y >= -PositionLimit && p.Y <= PositionLimit
}
func decodePosition(x, y *yaml.Node) (Position, error) {
	var p Position
	if x == nil || y == nil || x.ShortTag() != "!!int" || y.ShortTag() != "!!int" {
		return p, errors.New("position requires integer x and y")
	}
	if err := x.Decode(&p.X); err != nil {
		return p, err
	}
	if err := y.Decode(&p.Y); err != nil {
		return p, err
	}
	if !ValidPosition(p) {
		return p, errors.New("position is outside -100000..100000")
	}
	return p, nil
}
func diagramPositionFor(d diagram, id uuid.UUID) *Position {
	for _, v := range d.positions {
		if v.component == id {
			p := v.position
			return &p
		}
	}
	return nil
}
func samePosition(a, b *Position) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}
func (s Snapshot) NodePosition(diagramID, componentID string) (*Position, bool) {
	for _, d := range s.diagrams {
		if d.id.String() == diagramID {
			cid, err := uuid.Parse(componentID)
			if err == nil && visibleComponents(d, s.components)[cid] {
				return diagramPositionFor(d, cid), true
			}
		}
	}
	return nil, false
}

// visibleComponents is the canonical projection rule, including coalesced
// crossing endpoints, without recursively expanding boundary nodes.
func visibleComponents(d diagram, components []component) map[uuid.UUID]bool {
	canonical, visible := map[uuid.UUID]bool{}, map[uuid.UUID]bool{}
	for _, a := range d.appearances {
		canonical[a.component], visible[a.component] = true, true
	}
	for _, c := range components {
		for _, r := range c.relationships {
			if canonical[c.id] != canonical[r.target] {
				visible[c.id], visible[r.target] = true, true
			}
		}
	}
	return visible
}

func validatePositions(d diagram, components []component) error {
	visible := visibleComponents(d, components)
	for _, p := range d.positions {
		if !visible[p.component] || !ValidPosition(p.position) {
			return fmt.Errorf("%w: invalid visible position in Diagram %s", ErrInvalid, d.id)
		}
		delete(visible, p.component)
	}
	if len(visible) != 0 {
		return fmt.Errorf("%w: missing visible positions in Diagram %s", ErrInvalid, d.id)
	}
	return nil
}

// allocatePositions runs only while authoring or deriving a v2 display. Stored
// candidates replay its concrete results, never this algorithm.
func allocatePositions(visible map[uuid.UUID]bool, existing []diagramPosition) ([]diagramPosition, error) {
	result := append([]diagramPosition(nil), existing...)
	present := map[uuid.UUID]bool{}
	for _, p := range existing {
		present[p.component] = true
	}
	ids := []string{}
	for id := range visible {
		if !present[id] {
			ids = append(ids, id.String())
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		found := false
		for radius := 0; radius <= PositionLimit/320 && !found; radius++ {
			for y := -radius; y <= radius && !found; y++ {
				for x := -radius; x <= radius; x++ {
					if radius > 0 && x != -radius && x != radius && y != -radius && y != radius {
						continue
					}
					p := Position{x * 320, y * 200}
					free := true
					for _, occupied := range result {
						dx, dy := p.X-occupied.position.X, p.Y-occupied.position.Y
						if dx > -300 && dx < 300 && dy > -180 && dy < 180 {
							free = false
							break
						}
					}
					if free {
						result = append(result, diagramPosition{uuid.MustParse(id), p})
						found = true
						break
					}
				}
			}
		}
		if !found {
			return nil, fmt.Errorf("%w: no initial position available", ErrInvalid)
		}
	}
	return result, nil
}

func (s Snapshot) AutoLayout(diagramID string) ([]NodePositionChange, error) {
	for _, d := range s.diagrams {
		if d.id.String() != diagramID {
			continue
		}
		positions, err := allocateSizedPositions(d, s.components, nil, s.formatVersion)
		if err != nil {
			return nil, err
		}
		result := make([]NodePositionChange, 0, len(positions))
		for _, p := range positions {
			value := p.position
			result = append(result, NodePositionChange{diagramID, p.component.String(), &value})
		}
		return result, nil
	}
	return nil, fmt.Errorf("%w: unknown Diagram", ErrInvalid)
}

func applyNodePositions(base Snapshot, composition *CandidateComposition, diagrams map[uuid.UUID]diagram, changed map[uuid.UUID]struct{}, components []component, version int, initialize bool) error {
	if version == 2 {
		return nil
	}
	knownComponents := map[string]bool{}
	for _, c := range components {
		knownComponents[c.id.String()] = true
	}
	overrides := map[uuid.UUID]map[uuid.UUID]*Position{}
	for _, v := range composition.NodePositions {
		did, de := uuid.Parse(v.DiagramID)
		cid, ce := uuid.Parse(v.ComponentID)
		d, ok := diagrams[did]
		known := knownComponents[v.ComponentID]
		if de != nil || ce != nil || !ok || !known {
			return fmt.Errorf("%w: unknown placement target", ErrInvalid)
		}
		if overrides[did] == nil {
			overrides[did] = map[uuid.UUID]*Position{}
		}
		if _, dup := overrides[did][cid]; dup {
			return fmt.Errorf("%w: duplicate placement", ErrInvalid)
		}
		present := visibleComponents(d, components)[cid]
		if v.Position != nil && (!ValidPosition(*v.Position) || !present && !initialize) || v.Position == nil && present && !initialize {
			return fmt.Errorf("%w: ineligible placement target", ErrInvalid)
		}
		overrides[did][cid] = v.Position
	}
	for id, d := range diagrams {
		present := visibleComponents(d, components)
		original := d.positions
		// On first v2 placement, use exactly the layout shown by the v2 projection.
		if initialize && base.formatVersion == 2 {
			var err error
			d.positions, err = allocatePositions(present, nil)
			if err != nil {
				return err
			}
		}
		positions := []diagramPosition{}
		seen := map[uuid.UUID]bool{}
		for _, p := range d.positions {
			value := &p.position
			if override, ok := overrides[id][p.component]; ok {
				value = override
			}
			if present[p.component] && value != nil {
				positions = append(positions, diagramPosition{p.component, *value})
				seen[p.component] = true
			}
		}
		additions := []string{}
		for cid, p := range overrides[id] {
			if p != nil && present[cid] && !seen[cid] {
				additions = append(additions, cid.String())
			}
		}
		sort.Strings(additions)
		for _, cid := range additions {
			key := uuid.MustParse(cid)
			positions = append(positions, diagramPosition{key, *overrides[id][key]})
		}
		if initialize {
			var err error
			positions, err = allocateSizedPositions(d, components, positions, version)
			if err != nil {
				return err
			}
			// Match strict replay: retain base sequence order, then append all
			// newly positioned pairs in stable ID order, regardless of which
			// authoring mutation first introduced each pair.
			order := map[uuid.UUID]int{}
			for i, p := range original {
				order[p.component] = i + 1
			}
			sort.SliceStable(positions, func(i, j int) bool {
				a, b := order[positions[i].component], order[positions[j].component]
				if a != 0 && b != 0 {
					return a < b
				}
				if a != 0 {
					return true
				}
				if b != 0 {
					return false
				}
				return positions[i].component.String() < positions[j].component.String()
			})
			for _, p := range positions {
				bp, _ := base.NodePosition(id.String(), p.component.String())
				override, overridden := overrides[id][p.component]
				if !samePosition(bp, &p.position) || overridden && override == nil {
					value := p.position
					composition.NodePositions = setPositionOverride(composition.NodePositions, id.String(), p.component.String(), &value)
				}
			}
			for i, v := range composition.NodePositions {
				if v.DiagramID == id.String() && !present[uuid.MustParse(v.ComponentID)] {
					composition.NodePositions[i].Position = nil
				}
			}
			for _, p := range original {
				if !present[p.component] {
					composition.NodePositions = setPositionOverride(composition.NodePositions, id.String(), p.component.String(), nil)
				}
			}
		}
		equal := len(positions) == len(original)
		if equal {
			for i := range positions {
				if positions[i] != original[i] {
					equal = false
					break
				}
			}
		}
		if !equal {
			d.positions = positions
			diagrams[id] = d
			changed[id] = struct{}{}
		}
		if err := validatePositions(diagrams[id], components); err != nil {
			return err
		}
	}
	if initialize {
		composition.ArchitectureVersion = version
		sort.Slice(composition.NodePositions, func(i, j int) bool {
			a, b := composition.NodePositions[i], composition.NodePositions[j]
			if a.DiagramID != b.DiagramID {
				return a.DiagramID < b.DiagramID
			}
			return a.ComponentID < b.ComponentID
		})
	}
	return nil
}

// NormalizeNodePositions keeps final absence facts when a kept composition edit
// removes an appearance. It never captures automatic renderer output.
func NormalizeNodePositions(base, final Snapshot, values []NodePositionChange) []NodePositionChange {
	result := append([]NodePositionChange(nil), values...)
	for _, d := range base.diagrams {
		for _, p := range d.positions {
			if _, present := final.NodePosition(d.id.String(), p.component.String()); !present {
				result = setPositionOverride(result, d.id.String(), p.component.String(), nil)
			}
		}
	}
	kept := result[:0:0]
	for _, v := range result {
		_, present := final.NodePosition(v.DiagramID, v.ComponentID)
		basePosition, _ := base.NodePosition(v.DiagramID, v.ComponentID)
		if !present {
			v.Position = nil
		}
		if !samePosition(v.Position, basePosition) {
			kept = append(kept, v)
		}
	}
	return kept
}
func setPositionOverride(values []NodePositionChange, d, c string, p *Position) []NodePositionChange {
	out := append([]NodePositionChange(nil), values...)
	for i := range out {
		if out[i].DiagramID == d && out[i].ComponentID == c {
			out[i].Position = p
			return out
		}
	}
	return append(out, NodePositionChange{d, c, p})
}
func SetNodePosition(base Snapshot, composition CandidateComposition, d, c string, p *Position) CandidateComposition {
	composition.NodePositions = setPositionOverride(composition.NodePositions, d, c, p)
	if p != nil && composition.ArchitectureVersion < 3 && base.FormatVersion() < 4 {
		composition.ArchitectureVersion = 3
	}
	if composition.ArchitectureVersion == 0 {
		composition.ArchitectureVersion = base.FormatVersion()
	}
	return composition
}
