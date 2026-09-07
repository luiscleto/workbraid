package architecture

import (
	"errors"
	"fmt"
	"sort"
	"unicode/utf8"

	"github.com/google/uuid"
	"go.yaml.in/yaml/v4"
)

type Route struct {
	Bend int `json:"bend" yaml:"bend"`
}

// RouteAddress identifies a Diagram presentation slot, never a declaration.
type RouteAddress struct {
	DiagramID  string `json:"diagram_id" yaml:"diagram_id"`
	SourceID   string `json:"source_id" yaml:"source_id"`
	TargetID   string `json:"target_id" yaml:"target_id"`
	Label      string `json:"label" yaml:"label"`
	Occurrence int    `json:"occurrence" yaml:"occurrence"`
}
type EdgeRouteChange struct {
	RouteAddress `yaml:",inline"`
	Route        *Route `json:"route" yaml:"route"`
}
type diagramRouteYAML struct {
	Source     string `yaml:"source"`
	Target     string `yaml:"target"`
	Label      string `yaml:"label"`
	Occurrence int    `yaml:"occurrence"`
	Bend       int    `yaml:"bend"`
}
type routeSlot struct {
	source, target uuid.UUID
	label          string
	occurrence     int
}
type diagramRoute struct {
	slot  routeSlot
	route Route
}

func ValidRoute(r Route) bool    { return r.Bend >= -PositionLimit && r.Bend <= PositionLimit }
func sameRoute(a, b *Route) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func routeSlotLess(a, b routeSlot) bool {
	if a.source != b.source {
		return a.source.String() < b.source.String()
	}
	if a.target != b.target {
		return a.target.String() < b.target.String()
	}
	if a.label != b.label {
		return a.label < b.label
	}
	return a.occurrence < b.occurrence
}
func (a RouteAddress) slot() (routeSlot, error) {
	s, se := uuid.Parse(a.SourceID)
	t, te := uuid.Parse(a.TargetID)
	if se != nil || te != nil || a.Occurrence < 1 || !utf8.ValidString(a.Label) {
		return routeSlot{}, errors.New("invalid route address")
	}
	return routeSlot{s, t, a.Label, a.Occurrence}, nil
}
func routeAddress(d uuid.UUID, s routeSlot) RouteAddress {
	return RouteAddress{d.String(), s.source.String(), s.target.String(), s.label, s.occurrence}
}
func diagramRouteFor(d diagram, s routeSlot) *Route {
	for _, r := range d.routes {
		if r.slot == s {
			v := r.route
			return &v
		}
	}
	return nil
}
func routeCount(d diagram, cs []component, s routeSlot) int {
	visible := false
	for _, a := range d.appearances {
		if a.component == s.source || a.component == s.target {
			visible = true
		}
	}
	if !visible {
		return 0
	}
	count := 0
	for _, c := range cs {
		if c.id == s.source {
			for _, r := range c.relationships {
				if r.target == s.target && r.label == s.label {
					count++
				}
			}
		}
	}
	return count
}
func validateRoutes(d diagram, cs []component) error {
	seen := map[routeSlot]bool{}
	for _, r := range d.routes {
		if seen[r.slot] || r.slot.source == r.slot.target || r.slot.occurrence < 1 || r.slot.occurrence > routeCount(d, cs, r.slot) || !ValidRoute(r.route) {
			return fmt.Errorf("%w: invalid or ineligible Diagram route", ErrInvalid)
		}
		seen[r.slot] = true
	}
	return nil
}
func validateRouteYAML(item *yaml.Node, operational bool) error {
	keys := map[string]string{"source": "!!str", "target": "!!str", "label": "!!str", "occurrence": "!!int", "bend": "!!int"}
	if operational {
		keys = map[string]string{"diagram_id": "!!str", "source_id": "!!str", "target_id": "!!str", "label": "!!str", "occurrence": "!!int", "route": ""}
	}
	f, err := validateClosedMapping(item, "route", keys, nil)
	if err != nil {
		return err
	}
	var n int
	if err = f["occurrence"].Decode(&n); err != nil || n < 1 {
		return errors.New("route occurrence must be a positive integer")
	}
	b := f["bend"]
	if operational {
		if f["route"].ShortTag() == "!!null" {
			return nil
		}
		r, e := validateClosedMapping(f["route"], "route value", map[string]string{"bend": "!!int"}, nil)
		if e != nil {
			return e
		}
		b = r["bend"]
	}
	if err = b.Decode(&n); err != nil || !ValidRoute(Route{n}) {
		return errors.New("bend must be an integer between -100000 and 100000")
	}
	return nil
}
func setRouteOverride(values []EdgeRouteChange, a RouteAddress, r *Route) []EdgeRouteChange {
	out := append([]EdgeRouteChange(nil), values...)
	for i := range out {
		if out[i].RouteAddress == a {
			out[i].Route = r
			return out
		}
	}
	return append(out, EdgeRouteChange{a, r})
}
func sortRouteFacts(values []EdgeRouteChange) {
	sort.Slice(values, func(i, j int) bool {
		a, b := values[i], values[j]
		if a.DiagramID != b.DiagramID {
			return a.DiagramID < b.DiagramID
		}
		as, _ := a.slot()
		bs, _ := b.slot()
		return routeSlotLess(as, bs)
	})
}

// Strict replay applies explicit final values. It never silently prunes routes.
func applyEdgeRoutes(base Snapshot, composition CandidateComposition, ds map[uuid.UUID]diagram, changed map[uuid.UUID]struct{}, cs []component, version int) error {
	if version < 5 {
		if len(composition.EdgeRoutes) > 0 {
			return ErrInvalid
		}
		return nil
	}
	overrides := map[uuid.UUID]map[routeSlot]*Route{}
	for _, v := range composition.EdgeRoutes {
		d, de := uuid.Parse(v.DiagramID)
		s, se := v.slot()
		_, ok := ds[d]
		if de != nil || se != nil || !ok || v.Route != nil && !ValidRoute(*v.Route) {
			return fmt.Errorf("%w: invalid route fact", ErrInvalid)
		}
		if v.Route == nil && (s.source == s.target || s.occurrence > routeCount(ds[d], cs, s)) && base.routeAt(v.RouteAddress) == nil {
			return fmt.Errorf("%w: route removal does not name a visible or inherited slot", ErrInvalid)
		}
		if overrides[d] == nil {
			overrides[d] = map[routeSlot]*Route{}
		}
		if _, dup := overrides[d][s]; dup {
			return fmt.Errorf("%w: duplicate route fact", ErrInvalid)
		}
		overrides[d][s] = v.Route
	}
	for id, d := range ds {
		original := d.routes
		values := map[routeSlot]Route{}
		for _, r := range original {
			values[r.slot] = r.route
		}
		for s, v := range overrides[id] {
			if v == nil {
				delete(values, s)
			} else {
				values[s] = *v
			}
		}
		routes := []diagramRoute{}
		for _, r := range original {
			if v, ok := values[r.slot]; ok {
				routes = append(routes, diagramRoute{r.slot, v})
				delete(values, r.slot)
			}
		}
		slots := []routeSlot{}
		for s := range values {
			slots = append(slots, s)
		}
		sort.Slice(slots, func(i, j int) bool { return routeSlotLess(slots[i], slots[j]) })
		for _, s := range slots {
			routes = append(routes, diagramRoute{s, values[s]})
		}
		d.routes = routes
		if err := validateRoutes(d, cs); err != nil {
			return err
		}
		equal := len(original) == len(routes)
		if equal {
			for i := range routes {
				if original[i] != routes[i] {
					equal = false
					break
				}
			}
		}
		if !equal {
			changed[id] = struct{}{}
		}
		ds[id] = d
	}
	return nil
}

func SetEdgeRoute(current Snapshot, c CandidateComposition, a RouteAddress, r *Route) CandidateComposition {
	if current.FormatVersion() < 4 {
		for _, d := range current.diagrams {
			for cid := range visibleComponents(d, current.components) {
				v, _ := current.DisplayNodeSize(d.id.String(), cid.String())
				c.NodeSizes = setSizeOverride(c.NodeSizes, d.id.String(), cid.String(), &v)
			}
		}
	}
	c.ArchitectureVersion = 5
	c.EdgeRoutes = setRouteOverride(c.EdgeRoutes, a, r)
	sortRouteFacts(c.EdgeRoutes)
	return c
}

type RouteProjection struct {
	RouteAddress
	Count       int    `json:"count"`
	Route       *Route `json:"route"`
	DisplayBend int    `json:"display_bend"`
	Eligible    bool   `json:"eligible"`
	Reason      string `json:"reason,omitempty"`
}

func (s Snapshot) DiagramRoutes(did string) []RouteProjection {
	var d diagram
	for _, v := range s.diagrams {
		if v.id.String() == did {
			d = v
			break
		}
	}
	if d.id == uuid.Nil {
		return nil
	}
	positions := d.positions
	if s.formatVersion == 2 {
		positions, _ = allocatePositions(visibleComponents(d, s.components), nil)
	}
	p := map[uuid.UUID]Position{}
	for _, v := range positions {
		p[v.component] = v.position
	}
	present := map[uuid.UUID]bool{}
	for _, a := range d.appearances {
		present[a.component] = true
	}
	result := []RouteProjection{}
	for _, source := range s.components {
		counts := map[uuid.UUID]int{}
		for _, r := range source.relationships {
			counts[r.target]++
		}
		indices := map[uuid.UUID]int{}
		occurrences := map[routeSlot]int{}
		for _, r := range source.relationships {
			index := indices[r.target]
			indices[r.target]++
			if !present[source.id] && !present[r.target] {
				continue
			}
			slot := routeSlot{source.id, r.target, r.label, 0}
			occurrences[slot]++
			slot.occurrence = occurrences[slot]
			v := RouteProjection{RouteAddress: routeAddress(d.id, slot), Count: routeCount(d, s.components, slot), Route: diagramRouteFor(d, slot), DisplayBend: (2*index - (counts[r.target] - 1)) * 26, Eligible: true}
			if source.id == r.target {
				v.Eligible = false
				v.Reason = "self_link"
			} else if p[source.id] == p[r.target] {
				v.Eligible = false
				v.Reason = "coincident_centers"
			}
			if v.Route != nil && v.Eligible {
				v.DisplayBend = v.Route.Bend
			}
			result = append(result, v)
		}
	}
	return result
}

// Count comparisons use exact authored rows even when the proposal is invalid.
// Only affected tuple groups acquire tombstones; unrelated invalid fields do
// not clear presentation. No partial candidate is created here.
func ResetChangedRouteCounts(base Snapshot, before, after []ComponentChange, c CandidateComposition) CandidateComposition {
	counts := func(changes []ComponentChange) map[routeSlot]int {
		rows := map[string][]AuthoringRelationship{}
		for _, v := range base.AuthoringComponents() {
			rows[v.ID] = v.Relationships
		}
		for _, v := range changes {
			if v.New || v.RelationshipsChanged {
				rows[v.ID] = v.Relationships
			}
		}
		out := map[routeSlot]int{}
		for source, rs := range rows {
			sid, e := uuid.Parse(source)
			if e != nil {
				continue
			}
			for _, r := range rs {
				tid, e := uuid.Parse(r.TargetID)
				if e == nil {
					out[routeSlot{sid, tid, r.Label, 0}]++
				}
			}
		}
		return out
	}
	b, a := counts(before), counts(after)
	addresses := map[RouteAddress]bool{}
	for _, d := range base.diagrams {
		for _, r := range d.routes {
			addresses[routeAddress(d.id, r.slot)] = true
		}
	}
	for _, r := range c.EdgeRoutes {
		addresses[r.RouteAddress] = true
	}
	for address := range addresses {
		s, _ := address.slot()
		s.occurrence = 0
		if b[s] != a[s] {
			c.EdgeRoutes = removeRouteOverride(base, c.EdgeRoutes, address)
		}
	}
	sortRouteFacts(c.EdgeRoutes)
	return c
}

func (s Snapshot) routeAt(a RouteAddress) *Route {
	slot, e := a.slot()
	if e != nil {
		return nil
	}
	for _, d := range s.diagrams {
		if d.id.String() == a.DiagramID {
			return diagramRouteFor(d, slot)
		}
	}
	return nil
}
func removeRouteOverride(base Snapshot, values []EdgeRouteChange, a RouteAddress) []EdgeRouteChange {
	if base.routeAt(a) != nil {
		return setRouteOverride(values, a, nil)
	}
	out := []EdgeRouteChange{}
	for _, v := range values {
		if v.RouteAddress != a {
			out = append(out, v)
		}
	}
	return out
}

// Membership observation reuses the constructor steps and never validates or
// publishes a partial Architecture canvas.
// Route provenance is operational validity, so check it before candidate-wide
// authored validation can stop reconstruction on an unrelated invalid field.
func validateRouteFactProvenance(base Snapshot, changes []ComponentChange, c CandidateComposition) error {
	rows := map[string][]AuthoringRelationship{}
	for _, component := range base.AuthoringComponents() {
		rows[component.ID] = component.Relationships
	}
	for _, change := range changes {
		if change.New {
			rows[change.ID] = change.Relationships
		} else if _, exists := rows[change.ID]; exists && change.RelationshipsChanged {
			rows[change.ID] = change.Relationships
		}
	}
	for _, fact := range c.EdgeRoutes {
		if fact.Route == nil && base.routeAt(fact.RouteAddress) != nil {
			continue
		}
		fail := func() error {
			return fmt.Errorf("route fact does not name an eligible visible slot or inherited removal")
		}
		if fact.SourceID == fact.TargetID {
			return fail()
		}
		sourceRows, sourceExists := rows[fact.SourceID]
		_, targetExists := rows[fact.TargetID]
		if !sourceExists || !targetExists {
			return fail()
		}
		count := 0
		for _, row := range sourceRows {
			if row.TargetID == fact.TargetID && row.Label == fact.Label {
				count++
			}
		}
		if fact.Occurrence < 1 || fact.Occurrence > count {
			return fail()
		}
		diagrams, err := routeMembership(base, changes, c)
		if err != nil {
			return fail()
		}
		visible := false
		for _, appearance := range diagrams[uuid.MustParse(fact.DiagramID)].appearances {
			if appearance.component.String() == fact.SourceID || appearance.component.String() == fact.TargetID {
				visible = true
				break
			}
		}
		if !visible {
			return fail()
		}
	}
	return nil
}

func routeMembership(base Snapshot, changes []ComponentChange, c CandidateComposition) (map[uuid.UUID]diagram, error) {
	ds := map[uuid.UUID]diagram{}
	for _, d := range base.diagrams {
		d.appearances = append([]diagramAppearance(nil), d.appearances...)
		ds[d.id] = d
	}
	for _, d := range c.DetailDiagrams {
		id, e := uuid.Parse(d.ID)
		if e != nil {
			return nil, e
		}
		ds[id] = diagram{id: id}
	}
	changed := map[uuid.UUID]struct{}{}
	ids := map[string]struct{}{}
	for _, v := range base.components {
		ids[v.id.String()] = struct{}{}
	}
	for _, v := range changes {
		ids[v.ID] = struct{}{}
	}
	if e := applyComponentHomes(changes, c, ds, changed); e != nil {
		return nil, e
	}
	if e := applyComponentReferences(c, ds, changed, ids); e != nil {
		return nil, e
	}
	return ds, nil
}
func ResetLostRouteVisibility(base Snapshot, beforeChanges, afterChanges []ComponentChange, before, c CandidateComposition) CandidateComposition {
	b, be := routeMembership(base, beforeChanges, before)
	a, ae := routeMembership(base, afterChanges, c)
	if be != nil || ae != nil {
		return c
	}
	addresses := map[RouteAddress]bool{}
	for _, d := range base.diagrams {
		for _, r := range d.routes {
			addresses[routeAddress(d.id, r.slot)] = true
		}
	}
	for _, r := range c.EdgeRoutes {
		addresses[r.RouteAddress] = true
	}
	visible := func(ds map[uuid.UUID]diagram, r RouteAddress) bool {
		d := ds[uuid.MustParse(r.DiagramID)]
		for _, p := range d.appearances {
			if p.component.String() == r.SourceID || p.component.String() == r.TargetID {
				return true
			}
		}
		return false
	}
	for r := range addresses {
		if visible(b, r) && !visible(a, r) {
			c.EdgeRoutes = removeRouteOverride(base, c.EdgeRoutes, r)
		}
	}
	sortRouteFacts(c.EdgeRoutes)
	return c
}
