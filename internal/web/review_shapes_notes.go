package web

import (
	"sort"
	"workbraid/internal/architecture"
)

type reviewShapeChange struct {
	DiagramID   string  `json:"diagram_id"`
	ComponentID string  `json:"component_id"`
	Before      *string `json:"before"`
	With        *string `json:"with"`
	Path        string  `json:"path"`
}
type reviewNoteChange struct {
	DiagramID string                  `json:"diagram_id"`
	NoteID    string                  `json:"note_id"`
	Before    *architecture.NoteValue `json:"before"`
	With      *architecture.NoteValue `json:"with"`
	Path      string                  `json:"path"`
}

func compareShapesNotes(before, with snapshotProjectionResponse) ([]reviewShapeChange, []reviewNoteChange) {
	type address struct{ diagram, id string }
	shapes := map[address]reviewShapeChange{}
	notes := map[address]reviewNoteChange{}
	for side, snapshot := range []snapshotProjectionResponse{before, with} {
		for _, d := range snapshot.Diagrams {
			for _, s := range d.Shapes {
				k := address{d.ID, s.ComponentID}
				v := shapes[k]
				v.DiagramID, v.ComponentID, v.Path = d.ID, s.ComponentID, "diagrams/"+d.Filename
				if side == 0 {
					v.Before = s.Shape
				} else {
					v.With = s.Shape
				}
				shapes[k] = v
			}
			for _, n := range d.Notes {
				k := address{d.ID, n.ID}
				v := notes[k]
				v.DiagramID, v.NoteID, v.Path = d.ID, n.ID, "diagrams/"+d.Filename
				copy := n.NoteValue
				if side == 0 {
					v.Before = &copy
				} else {
					v.With = &copy
				}
				notes[k] = v
			}
		}
	}
	ss := []reviewShapeChange{}
	for _, v := range shapes {
		if v.Before == nil && v.With == nil || v.Before != nil && v.With != nil && *v.Before == *v.With {
			continue
		}
		ss = append(ss, v)
	}
	sort.Slice(ss, func(i, j int) bool {
		if ss[i].DiagramID != ss[j].DiagramID {
			return ss[i].DiagramID < ss[j].DiagramID
		}
		return ss[i].ComponentID < ss[j].ComponentID
	})
	ns := []reviewNoteChange{}
	for _, v := range notes {
		if v.Before != nil && v.With != nil && *v.Before == *v.With {
			continue
		}
		ns = append(ns, v)
	}
	sort.Slice(ns, func(i, j int) bool {
		if ns[i].DiagramID != ns[j].DiagramID {
			return ns[i].DiagramID < ns[j].DiagramID
		}
		return ns[i].NoteID < ns[j].NoteID
	})
	return ss, ns
}
