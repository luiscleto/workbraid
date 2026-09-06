package architecture

import "sort"

func (f reconciliationFacts) sizingSide(k reconciliationPair) ReconciliationSide {
	if !f.visiblePairs()[k] {
		return ReconciliationSide{State: "not_applicable"}
	}
	if s, ok := f.sizes[k]; ok {
		return ReconciliationSide{State: "stored", ReconciliationValue: ReconciliationValue{Size: &s}}
	}
	if f.version < 4 {
		s := defaultSize(f.homes[k.component] != k.diagram && !f.references[k], true)
		return ReconciliationSide{State: "derived", ReconciliationValue: ReconciliationValue{Size: &s}}
	}
	return ReconciliationSide{State: "not_applicable"}
}

// Composition is settled before any size is chosen. Absent branches do
// not contribute resets. Newly introduced stored pairs compare as whole pairs.
func (c *reconciliationCalculation) mergeSizes() {
	c.final.version = max(c.a.version, c.p.version)
	c.final.sizes = map[reconciliationPair]Size{}
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
		b, a, p := c.b.sizingSide(k), c.a.sizingSide(k), c.p.sizingSide(k)
		l := ReconciliationLocator{Kind: "node_size", DiagramID: k.diagram, ComponentID: k.component}
		if !c.final.visiblePairs()[k] {
			if b.Size != nil || a.Size != nil || p.Size != nil {
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
		case sameSize(a.Size, p.Size):
			chosen = a
			reason = "same_result"
		case sameSize(a.Size, baseline.Size):
			chosen = p
			reason = "proposed_only"
		case sameSize(p.Size, baseline.Size):
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
					if r.Value == nil || r.Value.Size == nil || !ValidSize(*r.Value.Size) {
						c.invalid("invalid size")
					} else {
						chosen = ReconciliationSide{State: "stored", ReconciliationValue: *r.Value}
					}
				default:
					c.invalid("invalid size choice")
				}
				conflict.Resolved = c.err == nil
			}
			c.result.Conflicts = append(c.result.Conflicts, conflict)
		}
		if reason != "" && (!sameSize(b.Size, chosen.Size) || b.State == "not_applicable" && chosen.Size != nil) {
			c.automatic(l, reason, b, a, p)
		}
		if chosen.Size != nil && c.final.version >= 4 {
			c.final.sizes[k] = *chosen.Size
		}
	}
}
