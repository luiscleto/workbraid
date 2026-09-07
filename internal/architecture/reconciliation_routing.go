package architecture

import (
	"reflect"
	"sort"
)

type RouteLossContext struct {
	Side      string         `json:"side"`
	Reason    string         `json:"reason"`
	Addresses []RouteAddress `json:"addresses"`
}
type routeGroup struct{ diagram, source, target, label string }

func routeGroupFor(a RouteAddress) routeGroup {
	return routeGroup{a.DiagramID, a.SourceID, a.TargetID, a.Label}
}
func (g routeGroup) locator(kind string) ReconciliationLocator {
	return ReconciliationLocator{Kind: kind, DiagramID: g.diagram, SourceID: g.source, TargetID: g.target, Label: g.label}
}
func (f reconciliationFacts) routeCount(g routeGroup) int {
	return f.relationships[reconciliationRelationship{g.source, g.target, g.label}]
}
func (f reconciliationFacts) routeVisible(g routeGroup) bool {
	_, d := f.diagrams[g.diagram]
	_, s := f.components[g.source]
	_, t := f.components[g.target]
	return d && s && t && g.source != g.target && f.routeCount(g) > 0 && (f.homes[g.source] == g.diagram || f.homes[g.target] == g.diagram || f.references[reconciliationPair{g.diagram, g.source}] || f.references[reconciliationPair{g.diagram, g.target}])
}
func (f reconciliationFacts) routeGroupValues(g routeGroup) map[RouteAddress]Route {
	out := map[RouteAddress]Route{}
	for a, r := range f.routes {
		if routeGroupFor(a) == g {
			out[a] = r
		}
	}
	return out
}
func (f reconciliationFacts) routeSide(a RouteAddress) ReconciliationSide {
	if !f.routeVisible(routeGroupFor(a)) || a.Occurrence > f.routeCount(routeGroupFor(a)) {
		return ReconciliationSide{State: "not_applicable"}
	}
	if r, ok := f.routes[a]; ok {
		return ReconciliationSide{State: "custom", ReconciliationValue: ReconciliationValue{Route: &r}}
	}
	return ReconciliationSide{State: "default"}
}
func (c *reconciliationCalculation) mergeRoutes() {
	c.final.routes = map[RouteAddress]Route{}
	groups := map[routeGroup]bool{}
	for _, f := range []reconciliationFacts{c.b, c.a, c.p} {
		for a := range f.routes {
			groups[routeGroupFor(a)] = true
		}
	}
	ordered := []routeGroup{}
	for g := range groups {
		ordered = append(ordered, g)
	}
	sort.Slice(ordered, func(i, j int) bool {
		return locatorKey(ordered[i].locator("route_loss")) < locatorKey(ordered[j].locator("route_loss"))
	})
	for _, g := range ordered {
		dependent := false
		for _, v := range c.result.Conflicts {
			l := v.Locator
			if v.Resolved {
				continue
			}
			if l.Kind == "relationship_count" && l.SourceID == g.source && l.TargetID == g.target && l.Label == g.label || l.Kind == "home" && (l.ComponentID == g.source || l.ComponentID == g.target) || l.Kind == "component_object" && (l.ComponentID == g.source || l.ComponentID == g.target) || l.Kind == "diagram_object" && l.DiagramID == g.diagram || l.Kind == "composition" && (containsID(l.ComponentIDs, g.source) || containsID(l.ComponentIDs, g.target) || containsID(l.DiagramIDs, g.diagram)) {
				dependent = true
			}
		}
		if dependent {
			continue
		}
		losses := []RouteLossContext{}
		finalVisible := c.final.routeVisible(g)
		count := c.final.routeCount(g)
		compatible := func(f reconciliationFacts) bool { return finalVisible && f.routeVisible(g) && f.routeCount(g) == count }
		for _, side := range []struct {
			name string
			f    reconciliationFacts
		}{{"accepted", c.a}, {"proposed", c.p}} {
			values := side.f.routeGroupValues(g)
			before := c.b.routeGroupValues(g)
			if !reflect.DeepEqual(values, before) && !compatible(side.f) {
				addresses := map[RouteAddress]bool{}
				for a := range values {
					addresses[a] = true
				}
				for a := range before {
					addresses[a] = true
				}
				changed := []EdgeRouteChange{}
				for a := range addresses {
					br, bok := before[a]
					sr, sok := values[a]
					if bok != sok || br != sr {
						changed = append(changed, EdgeRouteChange{RouteAddress: a})
					}
				}
				sortRouteFacts(changed)
				context := RouteLossContext{Side: side.name, Reason: "tuple_count_changed", Addresses: []RouteAddress{}}
				if !finalVisible || !side.f.routeVisible(g) {
					context.Reason = "visibility_changed"
				}
				for _, r := range changed {
					context.Addresses = append(context.Addresses, r.RouteAddress)
				}
				losses = append(losses, context)
			}
		}
		if len(losses) > 0 {
			l := g.locator("route_loss")
			conflict := ReconciliationConflict{Locator: l, Choices: []string{"clear"}, RouteLoss: losses, Original: countSide(c.b.routeCount(g)), Accepted: countSide(c.a.routeCount(g)), Proposed: countSide(c.p.routeCount(g))}
			if r, ok := c.resolutions[locatorKey(l)]; ok {
				c.used[locatorKey(l)] = true
				if r.Choice != "clear" || r.Value != nil {
					c.invalid("route loss requires clear without value")
				} else {
					conflict.Resolved = true
				}
			}
			c.result.Conflicts = append(c.result.Conflicts, conflict)
			continue
		}
		if !finalVisible {
			continue
		}
		addresses := map[RouteAddress]bool{}
		for _, f := range []reconciliationFacts{c.b, c.a, c.p} {
			for a := range f.routeGroupValues(g) {
				if a.Occurrence <= count {
					addresses[a] = true
				}
			}
		}
		slots := []EdgeRouteChange{}
		for a := range addresses {
			slots = append(slots, EdgeRouteChange{RouteAddress: a})
		}
		sortRouteFacts(slots)
		for _, slot := range slots {
			address := slot.RouteAddress
			b, a, p := c.b.routeSide(address), c.a.routeSide(address), c.p.routeSide(address)
			l := g.locator("route_value")
			l.Occurrence = address.Occurrence
			chosen := ReconciliationSide{State: "default"}
			reason := ""
			switch {
			case !compatible(c.a) && !compatible(c.p):
			case !compatible(c.a):
				chosen = p
				reason = "proposed_only"
			case !compatible(c.p):
				chosen = a
				reason = "accepted_only"
			case c.a.version < 5 && p.Route != nil:
				chosen = p
				reason = "proposed_only"
			case c.p.version < 5 && a.Route != nil:
				chosen = a
				reason = "accepted_only"
			case sameRoute(a.Route, p.Route):
				chosen = a
				reason = "same_result"
			case sameRoute(a.Route, b.Route):
				chosen = p
				reason = "proposed_only"
			case sameRoute(p.Route, b.Route):
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
						if r.Value == nil || r.Value.Route != nil && !ValidRoute(*r.Value.Route) {
							c.invalid("invalid route value")
						} else {
							chosen = ReconciliationSide{State: "default", ReconciliationValue: *r.Value}
							if chosen.Route != nil {
								chosen.State = "custom"
							}
						}
					default:
						c.invalid("invalid route choice")
					}
					conflict.Resolved = c.err == nil
				}
				c.result.Conflicts = append(c.result.Conflicts, conflict)
			}
			if chosen.Route != nil {
				c.final.routes[address] = *chosen.Route
			}
			if reason != "" && !sameRoute(b.Route, chosen.Route) {
				c.automatic(l, reason, b, a, p)
			}
		}
	}
}
