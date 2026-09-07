package architecture

import "sort"

type reconciliationNoteKey struct{ diagram, note string }

func (f reconciliationFacts) componentShapes(id string) map[reconciliationPair]string {
	out := map[reconciliationPair]string{}
	for k, v := range f.shapes {
		if k.component == id {
			out[k] = v
		}
	}
	return out
}
func (f reconciliationFacts) diagramShapes(id string) map[reconciliationPair]string {
	out := map[reconciliationPair]string{}
	for k, v := range f.shapes {
		if k.diagram == id {
			out[k] = v
		}
	}
	return out
}
func (f reconciliationFacts) diagramNotes(id string) map[reconciliationNoteKey]NoteValue {
	out := map[reconciliationNoteKey]NoteValue{}
	for k, v := range f.notes {
		if k.diagram == id {
			out[k] = v
		}
	}
	return out
}
func (f reconciliationFacts) shapeSide(k reconciliationPair) ReconciliationSide {
	if !f.visiblePairs()[k] {
		return ReconciliationSide{State: "not_applicable"}
	}
	if v, ok := f.shapes[k]; ok {
		return ReconciliationSide{Exists: true, State: "explicit", ReconciliationValue: ReconciliationValue{Shape: &v}}
	}
	return ReconciliationSide{Exists: true, State: "default"}
}
func (f reconciliationFacts) noteSide(k reconciliationNoteKey) ReconciliationSide {
	if v, ok := f.notes[k]; ok {
		return ReconciliationSide{Exists: true, ReconciliationValue: ReconciliationValue{Note: &v}}
	}
	return ReconciliationSide{}
}
func (c *reconciliationCalculation) mergeShapesNotes() {
	c.final.shapes = map[reconciliationPair]string{}
	c.final.notes = map[reconciliationNoteKey]NoteValue{}
	pairs := map[reconciliationPair]bool{}
	notes := map[reconciliationNoteKey]bool{}
	for _, f := range []reconciliationFacts{c.b, c.a, c.p} {
		for k := range f.shapes {
			pairs[k] = true
		}
		for k := range f.notes {
			notes[k] = true
		}
	}
	keys := []reconciliationPair{}
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].diagram != keys[j].diagram {
			return keys[i].diagram < keys[j].diagram
		}
		return keys[i].component < keys[j].component
	})
	for _, k := range keys {
		if !c.final.visiblePairs()[k] {
			continue
		}
		b, a, p := c.b.shapeSide(k), c.a.shapeSide(k), c.p.shapeSide(k)
		l := ReconciliationLocator{Kind: "node_shape", DiagramID: k.diagram, ComponentID: k.component}
		chosen := a
		reason := ""
		switch {
		case !a.Exists:
			chosen, reason = p, "proposed_only"
		case !p.Exists:
			reason = "accepted_only"
		case c.a.version < 6 && p.Shape != nil:
			chosen, reason = p, "proposed_only"
		case c.p.version < 6 && a.Shape != nil:
			reason = "accepted_only"
		case sameShape(a.Shape, p.Shape):
			reason = "same_result"
		case sameShape(a.Shape, b.Shape):
			chosen, reason = p, "proposed_only"
		case sameShape(p.Shape, b.Shape):
			reason = "accepted_only"
		default:
			chosen = c.shapeNoteConflict(l, b, a, p)
		}
		if reason != "" && !sameShape(b.Shape, chosen.Shape) {
			c.automatic(l, reason, b, a, p)
		}
		if chosen.Shape != nil {
			c.final.shapes[k] = *chosen.Shape
		}
	}
	nks := []reconciliationNoteKey{}
	for k := range notes {
		nks = append(nks, k)
	}
	sort.Slice(nks, func(i, j int) bool {
		if nks[i].diagram != nks[j].diagram {
			return nks[i].diagram < nks[j].diagram
		}
		return nks[i].note < nks[j].note
	})
	for _, k := range nks {
		if _, ok := c.final.diagrams[k.diagram]; !ok {
			continue
		}
		b, a, p := c.b.noteSide(k), c.a.noteSide(k), c.p.noteSide(k)
		l := ReconciliationLocator{Kind: "diagram_note", DiagramID: k.diagram, NoteID: k.note}
		if b.Note == nil && a.Note != nil && p.Note != nil && !sameNote(a.Note, p.Note) {
			c.result.Conflicts = append(c.result.Conflicts, ReconciliationConflict{Locator: l, Original: b, Accepted: a, Proposed: p, Choices: []string{}, Unsupported: map[string]string{"reason": "replace_identity"}})
			if _, ok := c.resolutions[locatorKey(l)]; ok {
				c.err = &ReconciliationError{"reconciliation_unsupported", "replace_identity"}
			}
			continue
		}
		chosen := a
		reason := ""
		switch {
		case sameNote(a.Note, p.Note):
			reason = "same_result"
		case sameNote(a.Note, b.Note):
			chosen, reason = p, "proposed_only"
		case sameNote(p.Note, b.Note):
			reason = "accepted_only"
		default:
			chosen = c.shapeNoteConflict(l, b, a, p)
		}
		if reason != "" && !sameNote(b.Note, chosen.Note) {
			c.automatic(l, reason, b, a, p)
		}
		if chosen.Note != nil {
			c.final.notes[k] = *chosen.Note
		}
	}
}
func (c *reconciliationCalculation) shapeNoteConflict(l ReconciliationLocator, b, a, p ReconciliationSide) ReconciliationSide {
	conflict := ReconciliationConflict{Locator: l, Original: b, Accepted: a, Proposed: p, Choices: []string{"accepted", "proposed", "manual"}}
	chosen := ReconciliationSide{}
	if r, ok := c.resolutions[locatorKey(l)]; ok {
		c.used[locatorKey(l)] = true
		switch r.Choice {
		case "accepted":
			chosen = a
		case "proposed":
			chosen = p
		case "manual":
			if r.Value == nil || l.Kind == "node_shape" && r.Value.Shape != nil && !ValidShape(*r.Value.Shape) || l.Kind == "diagram_note" && r.Value.Note != nil && !ValidNote(*r.Value.Note) {
				c.invalid("invalid shape/note value")
			} else {
				chosen = ReconciliationSide{Exists: true, ReconciliationValue: *r.Value}
			}
		default:
			c.invalid("invalid shape/note choice")
		}
		conflict.Resolved = c.err == nil
	}
	c.result.Conflicts = append(c.result.Conflicts, conflict)
	return chosen
}
func shapeNoteResidual(base Snapshot, a, final reconciliationFacts, c CandidateComposition) CandidateComposition {
	keys := map[reconciliationPair]bool{}
	for k := range a.shapes {
		keys[k] = true
	}
	for k := range final.shapes {
		keys[k] = true
	}
	for k := range keys {
		b, bok := a.shapes[k]
		v, ok := final.shapes[k]
		if bok == ok && b == v {
			continue
		}
		var ptr *string
		if ok {
			ptr = &v
		}
		c = SetNodeShape(c, k.diagram, k.component, ptr)
	}
	notes := map[reconciliationNoteKey]bool{}
	for k := range a.notes {
		notes[k] = true
	}
	for k := range final.notes {
		notes[k] = true
	}
	for k := range notes {
		b, bok := a.notes[k]
		v, ok := final.notes[k]
		if bok == ok && b == v {
			continue
		}
		var ptr *NoteValue
		if ok {
			ptr = &v
		}
		c = SetDiagramNote(base, c, k.diagram, k.note, ptr)
	}
	return c
}
