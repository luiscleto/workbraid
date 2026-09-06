package architecture

import "sort"

func (f reconciliationFacts) appearance(k reconciliationPair) bool {
	_, component := f.components[k.component]
	_, diagram := f.diagrams[k.diagram]
	return component && diagram && (f.homes[k.component] == k.diagram || f.references[k])
}
func (f reconciliationFacts) placementSide(k reconciliationPair) ReconciliationSide {
	if !f.appearance(k) {
		return ReconciliationSide{State: "not_applicable"}
	}
	if p, ok := f.positions[k]; ok {
		return ReconciliationSide{State: "manual", ReconciliationValue: ReconciliationValue{Position: &p}}
	}
	return ReconciliationSide{State: "automatic"}
}

// Composition is settled before any placement is chosen. Absent branches do
// not contribute resets; a newly introduced pair compares against Automatic.
func (c *reconciliationCalculation) mergePositions() {
	c.final.version = max(c.a.version, c.p.version)
	c.final.positions = map[reconciliationPair]Position{}
	pairs := map[reconciliationPair]bool{}
	for _, f := range []reconciliationFacts{c.b, c.a, c.p, c.final} {
		for id, d := range f.homes {
			pairs[reconciliationPair{d, id}] = true
		}
		for k := range f.references {
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
		if !c.final.appearance(k) {
			if b.Position != nil || a.Position != nil || p.Position != nil {
				c.automatic(l, "removed_with_appearance", b, a, p)
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
		if baseline.State == "not_applicable" {
			baseline = ReconciliationSide{State: "automatic"}
		}
		chosen := ReconciliationSide{State: "automatic"}
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
					if r.Value == nil || r.Value.Position != nil && !ValidPosition(*r.Value.Position) {
						c.invalid("invalid position")
					} else {
						chosen = ReconciliationSide{State: "automatic", ReconciliationValue: *r.Value}
						if chosen.Position != nil {
							chosen.State = "manual"
						}
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
		if chosen.Position != nil {
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
