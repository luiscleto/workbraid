package architecture

import (
	"errors"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"go.yaml.in/yaml/v4"
)

type NodeShapeChange struct {
	DiagramID   string  `json:"diagram_id" yaml:"diagram_id"`
	ComponentID string  `json:"component_id" yaml:"component_id"`
	Shape       *string `json:"shape" yaml:"shape"`
}
type NoteValue struct {
	Text   string `json:"text" yaml:"text"`
	X      int    `json:"x" yaml:"x"`
	Y      int    `json:"y" yaml:"y"`
	Width  int    `json:"width" yaml:"width"`
	Height int    `json:"height" yaml:"height"`
}
type DiagramNote struct {
	ID        string `json:"id" yaml:"id"`
	NoteValue `yaml:",inline"`
}
type DiagramNoteChange struct {
	DiagramID string     `json:"diagram_id" yaml:"diagram_id"`
	NoteID    string     `json:"note_id" yaml:"note_id"`
	Note      *NoteValue `json:"note" yaml:"note"`
}
type diagramShape struct {
	Component string `yaml:"component"`
	Shape     string `yaml:"shape"`
}

func ValidShape(s string) bool { return s == "rectangle" || s == "ellipse" || s == "diamond" }

func rectanglesOverlap(p Position, s Size, caption int, q Position, t Size, otherCaption int) bool {
	return 2*p.X-s.Width < 2*q.X+t.Width+48 && 2*p.X+s.Width+48 > 2*q.X-t.Width && 2*p.Y-s.Height < 2*q.Y+t.Height+2*otherCaption+48 && 2*p.Y+s.Height+2*caption+48 > 2*q.Y-t.Height
}
func (s Snapshot) NewDiagramNote(did, text string) (DiagramNote, error) {
	v := NoteValue{Text: text, Width: 240, Height: 120}
	if !ValidNote(v) {
		return DiagramNote{}, ErrInvalid
	}
	for _, d := range s.diagrams {
		if d.id.String() != did {
			continue
		}
		for radius := 0; radius <= PositionLimit/24; radius++ {
			for y := -radius; y <= radius; y++ {
				for x := -radius; x <= radius; x++ {
					if radius > 0 && x != -radius && x != radius && y != -radius && y != radius {
						continue
					}
					p := Position{x * 24, y * 24}
					free := true
					for _, n := range d.notes {
						if rectanglesOverlap(p, Size{240, 120}, 0, Position{n.X, n.Y}, Size{n.Width, n.Height}, 0) {
							free = false
							break
						}
					}
					if free {
						for _, projected := range s.DiagramProjections() {
							if projected.ID != did {
								continue
							}
							for _, a := range projected.Appearances {
								if a.DisplayPosition != nil && a.DisplaySize != nil && rectanglesOverlap(p, Size{240, 120}, 0, *a.DisplayPosition, *a.DisplaySize, 0) {
									free = false
								}
							}
							for _, b := range projected.Boundaries {
								if b.DisplayPosition != nil && b.DisplaySize != nil && rectanglesOverlap(p, Size{240, 120}, 0, *b.DisplayPosition, *b.DisplaySize, 24) {
									free = false
								}
							}
						}
					}
					if free {
						v.X, v.Y = p.X, p.Y
						return DiagramNote{uuid.NewString(), v}, nil
					}
				}
			}
		}
		return DiagramNote{}, ErrInvalid
	}
	return DiagramNote{}, ErrInvalid
}
func ValidNote(n NoteValue) bool {
	return utf8.ValidString(n.Text) && strings.TrimSpace(n.Text) != "" && utf8.RuneCountInString(n.Text) <= 2000 && ValidPosition(Position{n.X, n.Y}) && n.Width >= 120 && n.Width <= 800 && n.Height >= 48 && n.Height <= 600
}
func sameShape(a, b *string) bool   { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func sameNote(a, b *NoteValue) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func shapeAt(d diagram, c string) *string {
	for _, s := range d.shapes {
		if s.Component == c {
			v := s.Shape
			return &v
		}
	}
	return nil
}
func (s Snapshot) NodeShape(d, c string) (*string, bool) {
	for _, v := range s.diagrams {
		if v.id.String() == d {
			id, e := uuid.Parse(c)
			if e != nil || !visibleComponents(v, s.components)[id] {
				return nil, false
			}
			return shapeAt(v, id.String()), true
		}
	}
	return nil, false
}
func (s Snapshot) DiagramNotes(d string) ([]DiagramNote, bool) {
	for _, v := range s.diagrams {
		if v.id.String() == d {
			return append([]DiagramNote{}, v.notes...), true
		}
	}
	return nil, false
}
func validateShapeNoteYAML(n *yaml.Node, note, operational bool) error {
	keys := map[string]string{"component": "!!str", "shape": "!!str"}
	if note {
		keys = map[string]string{"id": "!!str", "text": "!!str", "x": "!!int", "y": "!!int", "width": "!!int", "height": "!!int"}
	}
	if operational {
		keys = map[string]string{"diagram_id": "!!str", "component_id": "!!str", "shape": ""}
		if note {
			keys = map[string]string{"diagram_id": "!!str", "note_id": "!!str", "note": ""}
		}
	}
	f, e := validateClosedMapping(n, "shape/note", keys, nil)
	if e != nil {
		return e
	}
	if operational {
		key := "shape"
		if note {
			key = "note"
		}
		if f[key].ShortTag() == "!!null" {
			return nil
		}
		if note {
			_, e = validateClosedMapping(f[key], "note", map[string]string{"text": "!!str", "x": "!!int", "y": "!!int", "width": "!!int", "height": "!!int"}, nil)
			if e != nil {
				return e
			}
			n = f[key]
		}
	}
	if !note {
		if f["shape"].ShortTag() != "!!str" || !ValidShape(f["shape"].Value) {
			return errors.New("invalid shape")
		}
		return nil
	}
	var v NoteValue
	if e = n.Decode(&v); e != nil || !ValidNote(v) {
		return errors.New("invalid note text or geometry")
	}
	return nil
}
func validateShapesNotes(d diagram, cs []component) error {
	visible := visibleComponents(d, cs)
	seen := map[uuid.UUID]bool{}
	for _, s := range d.shapes {
		id, e := uuid.Parse(s.Component)
		if e != nil || seen[id] || !visible[id] || !ValidShape(s.Shape) {
			return ErrInvalid
		}
		seen[id] = true
	}
	seen = map[uuid.UUID]bool{}
	for _, n := range d.notes {
		id, e := uuid.Parse(n.ID)
		if e != nil || seen[id] || !ValidNote(n.NoteValue) {
			return ErrInvalid
		}
		seen[id] = true
	}
	return nil
}
func validateShapeNoteFacts(c CandidateComposition) error {
	seen := map[[2]string]bool{}
	for _, v := range c.NodeShapes {
		k := [2]string{v.DiagramID, v.ComponentID}
		if !canonicalUUID(k[0]) || !canonicalUUID(k[1]) || seen[k] || c.ArchitectureVersion < 6 || v.Shape != nil && !ValidShape(*v.Shape) {
			return ErrInvalid
		}
		seen[k] = true
	}
	seen = map[[2]string]bool{}
	for _, v := range c.DiagramNotes {
		k := [2]string{v.DiagramID, v.NoteID}
		if !canonicalUUID(k[0]) || !canonicalUUID(k[1]) || seen[k] || c.ArchitectureVersion < 6 || v.Note != nil && !ValidNote(*v.Note) {
			return ErrInvalid
		}
		seen[k] = true
	}
	return nil
}

// Observe only authored membership and exact valid relationship rows. This is
// not a partial candidate and does not initialize or validate a canvas.
func shapeVisibility(base Snapshot, changes []ComponentChange, c CandidateComposition) (map[uuid.UUID]diagram, []component, error) {
	ds, e := routeMembership(base, changes, c)
	if e != nil {
		return nil, nil, e
	}
	byID := map[string]component{}
	for _, v := range base.components {
		byID[v.id.String()] = v
	}
	for _, v := range changes {
		current, ok := byID[v.ID]
		if !ok && !v.New {
			continue
		}
		if v.New {
			id, e := uuid.Parse(v.ID)
			if e != nil {
				continue
			}
			current = component{id: id}
		}
		if v.New || v.RelationshipsChanged {
			current.relationships = nil
			for _, r := range v.Relationships {
				target, e := uuid.Parse(r.TargetID)
				if e == nil && utf8.ValidString(r.Label) && strings.TrimSpace(r.Label) != "" {
					current.relationships = append(current.relationships, componentRelationship{target, r.Label})
				}
			}
		}
		byID[v.ID] = current
	}
	cs := []component{}
	for _, v := range byID {
		rows := []componentRelationship{}
		for _, r := range v.relationships {
			if _, ok := byID[r.target.String()]; ok {
				rows = append(rows, r)
			}
		}
		v.relationships = rows
		cs = append(cs, v)
	}
	return ds, cs, nil
}
func validateShapeNoteProvenance(base Snapshot, changes []ComponentChange, c CandidateComposition) error {
	if len(c.NodeShapes)+len(c.DiagramNotes) == 0 {
		return nil
	}
	if e := validateShapeNoteFacts(c); e != nil {
		return e
	}
	ds, cs, e := shapeVisibility(base, changes, c)
	if e != nil {
		return e
	}
	for _, f := range c.DiagramNotes {
		if _, ok := ds[uuid.MustParse(f.DiagramID)]; !ok {
			return errors.New("note owner does not exist")
		}
	}
	for _, f := range c.NodeShapes {
		d, ok := ds[uuid.MustParse(f.DiagramID)]
		if !ok {
			return ErrInvalid
		}
		if visibleComponents(d, cs)[uuid.MustParse(f.ComponentID)] {
			continue
		}
		if f.Shape == nil {
			inherited, _ := base.NodeShape(f.DiagramID, f.ComponentID)
			if inherited != nil {
				continue
			}
		}
		return errors.New("shape must address a visible Component or inherited removal")
	}
	return nil
}
func ResetLostShapes(base Snapshot, beforeChanges, afterChanges []ComponentChange, before, c CandidateComposition) CandidateComposition {
	b, bc, be := shapeVisibility(base, beforeChanges, before)
	a, ac, ae := shapeVisibility(base, afterChanges, c)
	if be != nil || ae != nil {
		return c
	}
	addresses := map[[2]string]bool{}
	for _, d := range base.diagrams {
		for _, s := range d.shapes {
			addresses[[2]string{d.id.String(), s.Component}] = true
		}
	}
	for _, f := range before.NodeShapes {
		if f.Shape != nil {
			addresses[[2]string{f.DiagramID, f.ComponentID}] = true
		}
	}
	for k := range addresses {
		did, cid := uuid.MustParse(k[0]), uuid.MustParse(k[1])
		if visibleComponents(b[did], bc)[cid] && !visibleComponents(a[did], ac)[cid] {
			inherited, _ := base.NodeShape(k[0], k[1])
			if inherited != nil {
				c = SetNodeShape(c, k[0], k[1], nil)
			} else {
				out := []NodeShapeChange{}
				for _, f := range c.NodeShapes {
					if f.DiagramID != k[0] || f.ComponentID != k[1] {
						out = append(out, f)
					}
				}
				c.NodeShapes = out
			}
		}
	}
	return c
}
func SetNodeShape(c CandidateComposition, d, id string, v *string) CandidateComposition {
	c.NodeShapes = append([]NodeShapeChange{}, c.NodeShapes...)
	for i, f := range c.NodeShapes {
		if f.DiagramID == d && f.ComponentID == id {
			c.NodeShapes[i].Shape = v
			return c
		}
	}
	c.NodeShapes = append(c.NodeShapes, NodeShapeChange{d, id, v})
	sort.Slice(c.NodeShapes, func(i, j int) bool {
		a, b := c.NodeShapes[i], c.NodeShapes[j]
		if a.DiagramID != b.DiagramID {
			return a.DiagramID < b.DiagramID
		}
		return a.ComponentID < b.ComponentID
	})
	return c
}
func SetDiagramNote(base Snapshot, c CandidateComposition, d, id string, v *NoteValue) CandidateComposition {
	c.DiagramNotes = append([]DiagramNoteChange{}, c.DiagramNotes...)
	inherited := false
	ns, _ := base.DiagramNotes(d)
	for _, n := range ns {
		if n.ID == id {
			inherited = true
		}
	}
	for i, f := range c.DiagramNotes {
		if f.DiagramID == d && f.NoteID == id {
			if v == nil && !inherited {
				c.DiagramNotes = append(c.DiagramNotes[:i], c.DiagramNotes[i+1:]...)
			} else {
				c.DiagramNotes[i].Note = v
			}
			return c
		}
	}
	if v != nil || inherited {
		c.DiagramNotes = append(c.DiagramNotes, DiagramNoteChange{d, id, v})
	}
	sort.Slice(c.DiagramNotes, func(i, j int) bool {
		a, b := c.DiagramNotes[i], c.DiagramNotes[j]
		if a.DiagramID != b.DiagramID {
			return a.DiagramID < b.DiagramID
		}
		return a.NoteID < b.NoteID
	})
	return c
}
func UpgradePresentation(current Snapshot, c CandidateComposition) CandidateComposition {
	if current.formatVersion < 4 {
		for _, d := range current.diagrams {
			for id := range visibleComponents(d, current.components) {
				v, _ := current.DisplayNodeSize(d.id.String(), id.String())
				c.NodeSizes = setSizeOverride(c.NodeSizes, d.id.String(), id.String(), &v)
			}
		}
	}
	c.ArchitectureVersion = 6
	return c
}

// Only ordinary preparation materializes visibility removals. Strict replay
// requires the same complete facts, including inherited shape tombstones.
func applyShapesNotes(base Snapshot, c *CandidateComposition, ds map[uuid.UUID]diagram, changed map[uuid.UUID]struct{}, cs []component, version int, prepare bool) error {
	if err := validateShapeNoteFacts(*c); err != nil {
		return err
	}
	if version < 6 {
		return nil
	}
	for _, f := range c.NodeShapes {
		if _, ok := ds[uuid.MustParse(f.DiagramID)]; !ok {
			return ErrInvalid
		}
	}
	for _, f := range c.DiagramNotes {
		if _, ok := ds[uuid.MustParse(f.DiagramID)]; !ok {
			return ErrInvalid
		}
	}
	for id, d := range ds {
		oldShapes, oldNotes := d.shapes, d.notes
		shapes := map[string]string{}
		for _, s := range d.shapes {
			shapes[s.Component] = s.Shape
		}
		for _, f := range c.NodeShapes {
			if f.DiagramID == id.String() {
				if f.Shape == nil {
					delete(shapes, f.ComponentID)
				} else {
					shapes[f.ComponentID] = *f.Shape
				}
			}
		}
		visible := visibleComponents(d, cs)
		if prepare {
			for cid := range shapes {
				if !visible[uuid.MustParse(cid)] {
					*c = SetNodeShape(*c, id.String(), cid, nil)
					delete(shapes, cid)
				}
			}
		}
		d.shapes = nil
		for _, s := range oldShapes {
			if v, ok := shapes[s.Component]; ok {
				d.shapes = append(d.shapes, diagramShape{s.Component, v})
				delete(shapes, s.Component)
			}
		}
		ids := []string{}
		for cid := range shapes {
			ids = append(ids, cid)
		}
		sort.Strings(ids)
		for _, cid := range ids {
			d.shapes = append(d.shapes, diagramShape{cid, shapes[cid]})
		}
		notes := map[string]NoteValue{}
		for _, n := range d.notes {
			notes[n.ID] = n.NoteValue
		}
		for _, f := range c.DiagramNotes {
			if f.DiagramID == id.String() {
				if f.Note == nil {
					delete(notes, f.NoteID)
				} else {
					notes[f.NoteID] = *f.Note
				}
			}
		}
		d.notes = nil
		for _, n := range oldNotes {
			if v, ok := notes[n.ID]; ok {
				d.notes = append(d.notes, DiagramNote{n.ID, v})
				delete(notes, n.ID)
			}
		}
		ids = nil
		for nid := range notes {
			ids = append(ids, nid)
		}
		sort.Strings(ids)
		for _, nid := range ids {
			d.notes = append(d.notes, DiagramNote{nid, notes[nid]})
		}
		if err := validateShapesNotes(d, cs); err != nil {
			return err
		}
		if !slices.Equal(oldShapes, d.shapes) || !slices.Equal(oldNotes, d.notes) {
			changed[id] = struct{}{}
		}
		ds[id] = d
	}
	return nil
}
