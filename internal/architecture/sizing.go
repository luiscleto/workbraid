package architecture

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"go.yaml.in/yaml/v4"
	"sort"
)

type Size struct {
	Width  int `json:"width" yaml:"width"`
	Height int `json:"height" yaml:"height"`
}
type NodeSizeChange struct {
	DiagramID   string `json:"diagram_id" yaml:"diagram_id"`
	ComponentID string `json:"component_id" yaml:"component_id"`
	Size        *Size  `json:"size" yaml:"size"`
}
type diagramSize struct {
	component uuid.UUID
	size      Size
}
type diagramSizeYAML struct {
	Component string `yaml:"component"`
	Width     int    `yaml:"width"`
	Height    int    `yaml:"height"`
}

func allocateSizedPositions(d diagram, components []component, existing []diagramPosition, version int) ([]diagramPosition, error) {
	visible := visibleComponents(d, components)
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
	sizeFor := func(id uuid.UUID) Size {
		if s := diagramSizeFor(d, id); s != nil {
			return *s
		}
		return defaultSize(boundaryNode(d, id), version < 4)
	}
	for _, id := range ids {
		cid := uuid.MustParse(id)
		size := sizeFor(cid)
		caption := 0
		if boundaryNode(d, cid) {
			caption = 24
		}
		found := false
		// Integer grid and stable IDs make allocation independent of browser/font metrics.
		for radius := 0; radius <= PositionLimit/24 && !found; radius++ {
			for y := -radius; y <= radius && !found; y++ {
				for x := -radius; x <= radius; x++ {
					if radius > 0 && x != -radius && x != radius && y != -radius && y != radius {
						continue
					}
					p := Position{x * 24, y * 24}
					free := true
					for _, n := range d.notes {
						if rectanglesOverlap(p, size, caption, Position{n.X, n.Y}, Size{n.Width, n.Height}, 0) {
							free = false
							break
						}
					}
					for _, occupied := range result {
						other := sizeFor(occupied.component)
						otherCaption := 0
						if boundaryNode(d, occupied.component) {
							otherCaption = 24
						}
						// Doubled arithmetic preserves half-unit outer bounds for odd sizes.
						if 2*p.X-size.Width < 2*occupied.position.X+other.Width+48 &&
							2*p.X+size.Width+48 > 2*occupied.position.X-other.Width &&
							2*p.Y-size.Height < 2*occupied.position.Y+other.Height+2*otherCaption+48 &&
							2*p.Y+size.Height+2*caption+48 > 2*occupied.position.Y-other.Height {
							free = false
							break
						}
					}
					if free {
						result = append(result, diagramPosition{cid, p})
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

func ValidSize(s Size) bool {
	return s.Width >= 80 && s.Width <= 1600 && s.Height >= 48 && s.Height <= 1200
}
func decodeSize(w, h *yaml.Node) (Size, error) {
	var s Size
	if w == nil || h == nil || w.ShortTag() != "!!int" || h.ShortTag() != "!!int" {
		return s, errors.New("size requires integer width and height")
	}
	if err := w.Decode(&s.Width); err != nil {
		return s, err
	}
	if err := h.Decode(&s.Height); err != nil {
		return s, err
	}
	if !ValidSize(s) {
		return s, errors.New("width must be 80–1600 and height 48–1200")
	}
	return s, nil
}
func boundaryNode(d diagram, id uuid.UUID) bool {
	for _, a := range d.appearances {
		if a.component == id {
			return false
		}
	}
	return true
}
func defaultSize(boundary, legacy bool) Size {
	if legacy {
		if boundary {
			return Size{104, 62}
		}
		return Size{116, 54}
	}
	if boundary {
		return Size{224, 112}
	}
	return Size{200, 96}
}
func diagramSizeFor(d diagram, id uuid.UUID) *Size {
	for _, s := range d.sizes {
		if s.component == id {
			v := s.size
			return &v
		}
	}
	return nil
}
func sameSize(a, b *Size) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func sizeProjection(d diagram, id uuid.UUID, version int) (*Size, *Size, string) {
	s := diagramSizeFor(d, id)
	if version >= 4 {
		return s, s, "stored"
	}
	v := defaultSize(boundaryNode(d, id), true)
	return nil, &v, "derived"
}
func (s Snapshot) NodeSize(did, cid string) (*Size, bool) {
	for _, d := range s.diagrams {
		if d.id.String() == did {
			id, err := uuid.Parse(cid)
			if err == nil && visibleComponents(d, s.components)[id] {
				return diagramSizeFor(d, id), true
			}
		}
	}
	return nil, false
}
func (s Snapshot) DisplayNodeSize(did, cid string) (Size, bool) {
	for _, d := range s.diagrams {
		if d.id.String() == did {
			id, err := uuid.Parse(cid)
			if err == nil && visibleComponents(d, s.components)[id] {
				_, v, _ := sizeProjection(d, id, s.formatVersion)
				if v != nil {
					return *v, true
				}
			}
		}
	}
	return Size{}, false
}
func (s Snapshot) DefaultNodeSize(did, cid string) (Size, bool) {
	for _, d := range s.diagrams {
		if d.id.String() == did {
			id, err := uuid.Parse(cid)
			if err == nil && visibleComponents(d, s.components)[id] {
				return defaultSize(boundaryNode(d, id), false), true
			}
		}
	}
	return Size{}, false
}
func validateSizes(d diagram, components []component) error {
	visible := visibleComponents(d, components)
	for _, s := range d.sizes {
		if !visible[s.component] || !ValidSize(s.size) {
			return fmt.Errorf("%w: invalid visible size in Diagram %s", ErrInvalid, d.id)
		}
		delete(visible, s.component)
	}
	if len(visible) != 0 {
		return fmt.Errorf("%w: missing visible sizes in Diagram %s", ErrInvalid, d.id)
	}
	return nil
}
func applyNodeSizes(base Snapshot, composition *CandidateComposition, diagrams map[uuid.UUID]diagram, changed map[uuid.UUID]struct{}, components []component, version int, initialize bool) error {
	if version < 4 {
		return nil
	}
	overrides := map[uuid.UUID]map[uuid.UUID]*Size{}
	known := map[uuid.UUID]bool{}
	for _, c := range components {
		known[c.id] = true
	}
	for _, v := range composition.NodeSizes {
		did, de := uuid.Parse(v.DiagramID)
		cid, ce := uuid.Parse(v.ComponentID)
		d, ok := diagrams[did]
		if de != nil || ce != nil || !ok || !known[cid] {
			return fmt.Errorf("%w: unknown size target", ErrInvalid)
		}
		if overrides[did] == nil {
			overrides[did] = map[uuid.UUID]*Size{}
		}
		if _, dup := overrides[did][cid]; dup {
			return fmt.Errorf("%w: duplicate size", ErrInvalid)
		}
		visible := visibleComponents(d, components)[cid]
		if v.Size != nil && (!ValidSize(*v.Size) || !visible && !initialize) || v.Size == nil && visible && !initialize {
			return fmt.Errorf("%w: ineligible size target", ErrInvalid)
		}
		overrides[did][cid] = v.Size
	}
	for did, d := range diagrams {
		visible := visibleComponents(d, components)
		original := d.sizes
		values := map[uuid.UUID]Size{}
		for _, s := range original {
			if visible[s.component] {
				values[s.component] = s.size
			}
		}
		for cid, v := range overrides[did] {
			if v == nil {
				delete(values, cid)
			} else if visible[cid] {
				values[cid] = *v
			}
		}
		if initialize {
			for cid := range visible {
				if _, ok := values[cid]; !ok {
					values[cid] = defaultSize(boundaryNode(d, cid), false)
				}
			}
		}
		sizes := []diagramSize{}
		for _, s := range original {
			if v, ok := values[s.component]; ok {
				sizes = append(sizes, diagramSize{s.component, v})
				delete(values, s.component)
			}
		}
		ids := []string{}
		for cid := range values {
			ids = append(ids, cid.String())
		}
		sort.Strings(ids)
		for _, id := range ids {
			cid := uuid.MustParse(id)
			sizes = append(sizes, diagramSize{cid, values[cid]})
		}
		d.sizes = sizes
		if err := validateSizes(d, components); err != nil {
			return err
		}
		if initialize {
			for _, s := range sizes {
				bp, _ := base.NodeSize(did.String(), s.component.String())
				override, overridden := overrides[did][s.component]
				if !sameSize(bp, &s.size) || overridden && override == nil {
					v := s.size
					composition.NodeSizes = setSizeOverride(composition.NodeSizes, did.String(), s.component.String(), &v)
				}
			}
			for i, v := range composition.NodeSizes {
				if v.DiagramID == did.String() && !visible[uuid.MustParse(v.ComponentID)] {
					composition.NodeSizes[i].Size = nil
				}
			}
			for _, bd := range base.diagrams {
				if bd.id == did {
					for _, s := range bd.sizes {
						if !visible[s.component] {
							composition.NodeSizes = setSizeOverride(composition.NodeSizes, did.String(), s.component.String(), nil)
						}
					}
				}
			}
		}
		equal := len(original) == len(sizes)
		if equal {
			for i := range sizes {
				if original[i] != sizes[i] {
					equal = false
					break
				}
			}
		}
		if !equal {
			changed[did] = struct{}{}
		}
		diagrams[did] = d
	}
	if initialize {
		composition.ArchitectureVersion = version
		sort.Slice(composition.NodeSizes, func(i, j int) bool {
			a, b := composition.NodeSizes[i], composition.NodeSizes[j]
			if a.DiagramID != b.DiagramID {
				return a.DiagramID < b.DiagramID
			}
			return a.ComponentID < b.ComponentID
		})
	}
	return nil
}
func setSizeOverride(values []NodeSizeChange, d, c string, s *Size) []NodeSizeChange {
	out := append([]NodeSizeChange(nil), values...)
	for i := range out {
		if out[i].DiagramID == d && out[i].ComponentID == c {
			out[i].Size = s
			return out
		}
	}
	return append(out, NodeSizeChange{d, c, s})
}
func SetNodeSize(current Snapshot, composition CandidateComposition, d, c string, s Size) CandidateComposition {
	// Capture the exact currently displayed side before its first actual resize,
	// including nodes added or converted by earlier legacy proposal edits.
	if current.FormatVersion() < 4 {
		for _, diagram := range current.diagrams {
			for cid := range visibleComponents(diagram, current.components) {
				v, _ := current.DisplayNodeSize(diagram.id.String(), cid.String())
				composition.NodeSizes = setSizeOverride(composition.NodeSizes, diagram.id.String(), cid.String(), &v)
			}
		}
	}
	composition.NodeSizes = setSizeOverride(composition.NodeSizes, d, c, &s)
	composition.ArchitectureVersion = max(4, current.FormatVersion())
	return composition
}
