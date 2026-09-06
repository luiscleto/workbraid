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
func diagramManualPosition(d diagram, id uuid.UUID) *Position {
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
			for _, a := range d.appearances {
				if a.component.String() == componentID {
					return diagramManualPosition(d, a.component), true
				}
			}
		}
	}
	return nil, false
}
func applyNodePositions(base Snapshot, composition CandidateComposition, diagrams map[uuid.UUID]diagram, changed map[uuid.UUID]struct{}, components map[string]struct{}) error {
	overrides := map[uuid.UUID]map[uuid.UUID]*Position{}
	for _, v := range composition.NodePositions {
		did, de := uuid.Parse(v.DiagramID)
		cid, ce := uuid.Parse(v.ComponentID)
		d, ok := diagrams[did]
		_, known := components[v.ComponentID]
		if de != nil || ce != nil || !ok || !known {
			return fmt.Errorf("%w: unknown placement target", ErrInvalid)
		}
		if overrides[did] == nil {
			overrides[did] = map[uuid.UUID]*Position{}
		}
		if _, dup := overrides[did][cid]; dup {
			return fmt.Errorf("%w: duplicate placement", ErrInvalid)
		}
		present := false
		for _, a := range d.appearances {
			if a.component == cid {
				present = true
			}
		}
		_, basePresent := base.NodePosition(v.DiagramID, v.ComponentID)
		if v.Position != nil && (!present || !ValidPosition(*v.Position)) || v.Position == nil && !present && !basePresent {
			return fmt.Errorf("%w: ineligible placement target", ErrInvalid)
		}
		overrides[did][cid] = v.Position
	}
	for id, d := range diagrams {
		present := map[uuid.UUID]bool{}
		for _, a := range d.appearances {
			present[a.component] = true
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
			if p != nil && !seen[cid] {
				additions = append(additions, cid.String())
			}
		}
		sort.Strings(additions)
		for _, cid := range additions {
			key := uuid.MustParse(cid)
			positions = append(positions, diagramPosition{key, *overrides[id][key]})
		}
		equal := len(positions) == len(d.positions)
		if equal {
			for i := range positions {
				if positions[i] != d.positions[i] {
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
	if p != nil {
		composition.ArchitectureVersion = 3
	}
	if composition.ArchitectureVersion == 0 {
		composition.ArchitectureVersion = base.FormatVersion()
	}
	return composition
}
