package architecture

import (
	"sort"

	"github.com/google/uuid"
)

func (f reconciliationFacts) visiblePairs() map[reconciliationPair]bool {
	result := map[reconciliationPair]bool{}
	components := make([]component, 0, len(f.components))
	for id := range f.components {
		c := component{id: uuid.MustParse(id)}
		for r, count := range f.relationships {
			if r.source == id && count > 0 {
				c.relationships = append(c.relationships, componentRelationship{target: uuid.MustParse(r.target), label: r.label})
			}
		}
		components = append(components, c)
	}
	for id := range f.diagrams {
		d := diagram{id: uuid.MustParse(id)}
		for cid, home := range f.homes {
			if home == id {
				d.appearances = append(d.appearances, diagramAppearance{component: uuid.MustParse(cid)})
			}
		}
		for k, present := range f.references {
			if present && k.diagram == id {
				d.appearances = append(d.appearances, diagramAppearance{component: uuid.MustParse(k.component)})
			}
		}
		for cid := range visibleComponents(d, components) {
			if _, ok := f.components[cid.String()]; ok {
				result[reconciliationPair{id, cid.String()}] = true
			}
		}
	}
	return result
}

func (f reconciliationFacts) placementSide(k reconciliationPair) ReconciliationSide {
	if !f.visiblePairs()[k] {
		return ReconciliationSide{State: "not_applicable"}
	}
	if p, ok := f.positions[k]; ok {
		return ReconciliationSide{State: "stored", ReconciliationValue: ReconciliationValue{Position: &p}}
	}
	if f.version == 2 {
		visible := map[uuid.UUID]bool{}
		for pair := range f.visiblePairs() {
			if pair.diagram == k.diagram {
				visible[uuid.MustParse(pair.component)] = true
			}
		}
		positions, _ := allocatePositions(visible, nil)
		for _, p := range positions {
			if p.component.String() == k.component {
				value := p.position
				return ReconciliationSide{State: "derived", ReconciliationValue: ReconciliationValue{Position: &value}}
			}
		}
	}
	return ReconciliationSide{State: "not_applicable"}
}

// Composition is settled before any placement is chosen. Absent branches do
// not contribute resets. Newly introduced stored pairs compare as whole pairs.
func (c *reconciliationCalculation) mergePositions() {
	c.final.version = max(c.a.version, c.p.version)
	c.final.positions = map[reconciliationPair]Position{}
	pairs := map[reconciliationPair]bool{}
	for _, f := range []reconciliationFacts{c.b, c.a, c.p, c.final} {
		for k := range f.visiblePairs() {
			pairs[k] = true
		}
	}
	keys := make([]reconciliationPair, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].diagram == keys[j].diagram {
			return keys[i].component < keys[j].component
		}
		return keys[i].diagram < keys[j].diagram
	})
	for _, k := range keys {
		b, a, p := c.b.placementSide(k), c.a.placementSide(k), c.p.placementSide(k)
		l := ReconciliationLocator{Kind: "node_position", DiagramID: k.diagram, ComponentID: k.component}
		if !c.final.visiblePairs()[k] {
			if b.Position != nil || a.Position != nil || p.Position != nil {
				c.automatic(l, "removed_with_visibility", b, a, p)
			}
			continue
		}
		dependent := false
		for _, conflict := range c.result.Conflicts {
			if conflict.Resolved {
				continue
			}
			loc := conflict.Locator
			if loc.Kind == "component_object" && loc.ComponentID == k.component || loc.Kind == "diagram_object" && loc.DiagramID == k.diagram || loc.Kind == "home" && loc.ComponentID == k.component || loc.Kind == "composition" && (containsID(loc.ComponentIDs, k.component) || containsID(loc.DiagramIDs, k.diagram)) {
				dependent = true
			}
		}
		if dependent {
			continue
		}
		baseline := b
		chosen := ReconciliationSide{State: "not_applicable"}
		reason := ""
		switch {
		case a.State == "not_applicable":
			if p.State != "not_applicable" {
				chosen = p
				reason = "proposed_only"
			}
		case p.State == "not_applicable":
			chosen = a
			reason = "accepted_only"
		case a.State == "derived" && p.State == "stored":
			chosen = p
			reason = "proposed_only"
		case p.State == "derived" && a.State == "stored":
			chosen = a
			reason = "accepted_only"
		case a.State == "derived" && p.State == "derived":
			chosen = a
		case samePosition(a.Position, p.Position):
			chosen = a
			reason = "same_result"
		case samePosition(a.Position, baseline.Position):
			chosen = p
			reason = "proposed_only"
		case samePosition(p.Position, baseline.Position):
			chosen = a
			reason = "accepted_only"
		default:
			conflict := ReconciliationConflict{Locator: l, Original: b, Accepted: a, Proposed: p, Choices: []string{"accepted", "proposed", "manual"}}
			if r, ok := c.resolutions[locatorKey(l)]; ok {
				c.used[locatorKey(l)] = true
				switch r.Choice {
				case "accepted":
					chosen = a
				case "proposed":
					chosen = p
				case "manual":
					if r.Value == nil || r.Value.Position == nil || !ValidPosition(*r.Value.Position) {
						c.invalid("invalid position")
					} else {
						chosen = ReconciliationSide{State: "stored", ReconciliationValue: *r.Value}
					}
				default:
					c.invalid("invalid placement choice")
				}
				conflict.Resolved = c.err == nil
			}
			c.result.Conflicts = append(c.result.Conflicts, conflict)
		}
		if reason != "" && (!samePosition(b.Position, chosen.Position) || b.State == "not_applicable" && chosen.Position != nil) {
			c.automatic(l, reason, b, a, p)
		}
		if chosen.Position != nil && c.final.version == 3 {
			c.final.positions[k] = *chosen.Position
		}
	}
}

func containsID(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
