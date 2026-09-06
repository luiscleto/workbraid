package architecture

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type reconciliationCalculation struct {
	result         Reconciliation
	b, a, p, final reconciliationFacts
	resolutions    map[string]ReconciliationResolution
	used           map[string]bool
	err            error
}

// Reconcile computes temporary semantic choices. It never writes a state
// commit or ref. The result tree comes exclusively from ConstructCandidate.
func (manager *Manager) Reconcile(ctx context.Context, base, accepted Snapshot, proposed Candidate, resolutions []ReconciliationResolution) (Reconciliation, error) {
	c := reconciliationCalculation{result: Reconciliation{Status: "ready", AutomaticChanges: []ReconciliationAutomaticChange{}, Conflicts: []ReconciliationConflict{}}, b: snapshotReconciliationFacts(base), a: snapshotReconciliationFacts(accepted), p: snapshotReconciliationFacts(proposed.Snapshot()), final: snapshotReconciliationFacts(accepted), resolutions: map[string]ReconciliationResolution{}, used: map[string]bool{}}
	for _, r := range resolutions {
		key := locatorKey(r.Locator)
		if _, exists := c.resolutions[key]; exists {
			return c.result, &ReconciliationError{"invalid_request", "duplicate locator"}
		}
		c.resolutions[key] = r
	}
	if base.StoreID() != accepted.StoreID() || base.StoreID() != proposed.Snapshot().StoreID() || base.FormatVersion() < 2 || accepted.FormatVersion() < 2 {
		return c.result, &ReconciliationError{"validation_blocked", "invalid_proposal"}
	}
	if base.Revision() == accepted.Revision() {
		if len(resolutions) > 0 {
			return c.result, &ReconciliationError{"invalid_request", "no conflicts require choices"}
		}
		c.result.Status = "not_required"
		return c.result, nil
	}
	c.mergeObjects()
	c.mergeComposition()
	for i, conflict := range c.result.Conflicts {
		if conflict.Locator.Kind == "detail_anchor" {
			options := c.compositionConflict(compositionLocator("", nil, []string{conflict.Locator.DiagramID}), false)
			c.result.Conflicts[i].EligibleAnchors = options.EligibleAnchors
			c.result.Conflicts[i].EligibleParents = options.EligibleParents
		}
	}
	if c.err != nil {
		return c.result, c.err
	}
	c.resolveComposition()
	c.mergePositions()
	c.mergeSizes()
	c.mergeRoutes()
	if c.err != nil {
		return c.result, c.err
	}
	for key := range c.resolutions {
		if !c.used[key] {
			if c.resolutions[key].Locator.Kind == "node_position" || c.resolutions[key].Locator.Kind == "node_size" {
				return c.result, &ReconciliationError{"target_not_eligible", "placement choice is no longer applicable"}
			}
			return c.result, &ReconciliationError{"invalid_request", "unknown or obsolete conflict locator"}
		}
	}
	for _, conflict := range c.result.Conflicts {
		if !conflict.Resolved {
			c.result.Status = "needs_resolution"
			if len(conflict.Choices) == 0 {
				c.result.Status = "blocked"
				break
			}
		}
	}
	if c.result.Status != "ready" {
		return c.result, nil
	}
	changes, composition, err := reconciliationResidual(accepted, proposed.Snapshot(), c.b, c.final)
	if err != nil {
		return c.result, err
	}
	candidate, err := manager.PrepareCandidate(ctx, accepted, changes, &composition)
	if err != nil {
		return c.result, fmt.Errorf("reconciliation constructor: %w", err)
	}
	reproduced := snapshotReconciliationFacts(candidate.Snapshot())
	if !reflect.DeepEqual(c.final.routes, reproduced.routes) {
		return c.result, fmt.Errorf("resolved routes changed during construction")
	}
	for k, s := range c.final.sizes {
		if actual, exists := reproduced.sizes[k]; !exists || actual != s {
			return c.result, fmt.Errorf("resolved size changed during initialization")
		}
	}
	for k, p := range c.final.positions {
		if actual, exists := reproduced.positions[k]; !exists || actual != p {
			return c.result, fmt.Errorf("resolved coordinate changed during initialization")
		}
	}
	replay, err := manager.ConstructCandidate(ctx, accepted, changes, composition)
	if err != nil || replay.Tree() != candidate.Tree() {
		return c.result, fmt.Errorf("reconciliation final facts do not reconstruct exact candidate: %v", err)
	}
	if !sameReconciliationSemantics(c.final, reproduced) || candidate.Snapshot().FormatVersion() != c.final.version {
		return c.result, fmt.Errorf("resolved Architecture is not reproduced by ordinary typed authoring facts")
	}
	c.result.Changes, c.result.Composition, c.result.Candidate = changes, composition, &candidate
	return c.result, nil
}

func (c *reconciliationCalculation) merge(l ReconciliationLocator, b, a, p ReconciliationSide) ReconciliationSide {
	if equalSide(a, p) {
		if !equalSide(b, a) {
			c.automatic(l, "same_result", b, a, p)
		}
		return a
	}
	if equalSide(b, p) {
		c.automatic(l, "accepted_only", b, a, p)
		return a
	}
	if equalSide(b, a) {
		c.automatic(l, "proposed_only", b, a, p)
		return p
	}
	conflict := ReconciliationConflict{Locator: l, Original: b, Accepted: a, Proposed: p, Choices: []string{"accepted", "proposed", "manual"}}
	if l.Kind == "detail_anchor" && l.DiagramID == c.final.root {
		conflict.Choices = []string{}
		conflict.Unsupported = map[string]string{"manual": "reassign_root"}
		for _, side := range []struct {
			name  string
			value ReconciliationSide
		}{{"accepted", a}, {"proposed", p}} {
			if side.value.Exists {
				conflict.Unsupported[side.name] = "reassign_root"
			} else {
				conflict.Choices = append(conflict.Choices, side.name)
			}
		}
	}
	result := a
	if r, ok := c.resolutions[locatorKey(l)]; ok {
		c.used[locatorKey(l)] = true
		if reason := conflict.Unsupported[r.Choice]; reason != "" {
			c.err = &ReconciliationError{"reconciliation_unsupported", reason}
			c.result.Conflicts = append(c.result.Conflicts, conflict)
			return result
		}
		switch r.Choice {
		case "accepted":
			if r.Value != nil {
				c.invalid("side choice cannot have a scalar value")
			}
			result = a
		case "proposed":
			if r.Value != nil {
				c.invalid("side choice cannot have a scalar value")
			}
			result = p
		case "manual":
			if !validReconciliationManual(l, r.Value) {
				c.invalid("invalid manual value")
				break
			}
			result = ReconciliationSide{Exists: true, ReconciliationValue: *r.Value}
		default:
			c.invalid("unknown choice")
		}
		conflict.Resolved = c.err == nil
	}
	c.result.Conflicts = append(c.result.Conflicts, conflict)
	return result
}

func validReconciliationManual(l ReconciliationLocator, v *ReconciliationValue) bool {
	if v == nil {
		return false
	}
	switch l.Kind {
	case "component_title":
		return v.Text != nil && strings.TrimSpace(*v.Text) == *v.Text && *v.Text != "" && !strings.ContainsAny(*v.Text, "\r\n")
	case "component_description":
		return v.Text != nil
	case "diagram_title":
		return v.Text != nil && strings.TrimSpace(*v.Text) != ""
	case "relationship_count":
		return v.Count != nil && *v.Count >= 0
	case "reference":
		return v.Present != nil
	case "home":
		return v.DiagramID != nil && canonicalUUID(*v.DiagramID)
	case "detail_anchor":
		return v.AnchorComponentID != nil && canonicalUUID(*v.AnchorComponentID)
	}
	return false
}

func (c *reconciliationCalculation) invalid(reason string) {
	c.err = &ReconciliationError{"invalid_request", reason}
}
func (c *reconciliationCalculation) automatic(l ReconciliationLocator, reason string, b, a, p ReconciliationSide) {
	c.result.AutomaticChanges = append(c.result.AutomaticChanges, ReconciliationAutomaticChange{l, reason, b, a, p})
}

func (f reconciliationFacts) componentContext(id string) any {
	c, exists := f.components[id]
	if !exists {
		return nil
	}
	rels := map[reconciliationRelationship]int{}
	refs := map[reconciliationPair]bool{}
	children := map[string]string{}
	positions := map[reconciliationPair]Position{}
	sizes := map[reconciliationPair]Size{}
	routes := map[RouteAddress]Route{}
	for k, v := range f.routes {
		if k.SourceID == id || k.TargetID == id {
			routes[k] = v
		}
	}
	for k, v := range f.positions {
		if k.component == id {
			positions[k] = v
		}
	}
	for k, v := range f.sizes {
		if k.component == id {
			sizes[k] = v
		}
	}
	for k, v := range f.relationships {
		if k.source == id || k.target == id {
			rels[k] = v
		}
	}
	for k, v := range f.references {
		if k.component == id {
			refs[k] = v
		}
	}
	for k, v := range f.anchors {
		if v == id {
			children[k] = v
		}
	}
	return struct {
		Title, Description, Home string
		Relationships            map[reconciliationRelationship]int
		References               map[reconciliationPair]bool
		Children                 map[string]string
		Positions                map[reconciliationPair]Position
		Sizes                    map[reconciliationPair]Size
		Routes                   map[RouteAddress]Route
	}{c.Title, c.Description, f.homes[id], rels, refs, children, positions, sizes, routes}
}

func (f reconciliationFacts) diagramContext(id string) any {
	d, exists := f.diagrams[id]
	if !exists {
		return nil
	}
	homes := map[string]string{}
	refs := map[reconciliationPair]bool{}
	children := map[string]string{}
	positions := map[reconciliationPair]Position{}
	sizes := map[reconciliationPair]Size{}
	routes := map[RouteAddress]Route{}
	for k, v := range f.routes {
		if k.DiagramID == id {
			routes[k] = v
		}
	}
	for k, v := range f.positions {
		if k.diagram == id {
			positions[k] = v
		}
	}
	for k, v := range f.sizes {
		if k.diagram == id {
			sizes[k] = v
		}
	}
	for k, v := range f.homes {
		if v == id {
			homes[k] = v
		}
	}
	for k, v := range f.references {
		if k.diagram == id {
			refs[k] = v
		}
	}
	for k, v := range f.anchors {
		if f.homes[v] == id {
			children[k] = v
		}
	}
	return struct {
		Title, Anchor string
		Root          bool
		Homes         map[string]string
		References    map[reconciliationPair]bool
		Children      map[string]string
		Positions     map[reconciliationPair]Position
		Sizes         map[reconciliationPair]Size
		Routes        map[RouteAddress]Route
	}{d.title, f.anchors[id], f.root == id, homes, refs, children, positions, sizes, routes}
}

func (c *reconciliationCalculation) object(l ReconciliationLocator, b, a, p any) bool {
	if b == nil && a != nil && p != nil && !reflect.DeepEqual(a, p) {
		c.result.Conflicts = append(c.result.Conflicts, ReconciliationConflict{Locator: l, Original: c.b.objectSide(l), Accepted: c.a.objectSide(l), Proposed: c.p.objectSide(l), Choices: []string{}, Unsupported: map[string]string{"reason": "replace_identity"}})
		if _, ok := c.resolutions[locatorKey(l)]; ok {
			c.err = &ReconciliationError{"reconciliation_unsupported", "replace_identity"}
		}
		return true
	}
	if b != nil && a == nil && p != nil && !reflect.DeepEqual(b, p) {
		reason := "restore_component"
		if l.Kind == "diagram_object" {
			reason = "restore_diagram"
		}
		conflict := ReconciliationConflict{Locator: l, Original: c.b.objectSide(l), Accepted: c.a.objectSide(l), Proposed: c.p.objectSide(l), Choices: []string{"accepted"}, Unsupported: map[string]string{"proposed": reason}}
		if r, ok := c.resolutions[locatorKey(l)]; ok {
			c.used[locatorKey(l)] = true
			if r.Choice != "accepted" {
				c.err = &ReconciliationError{"reconciliation_unsupported", reason}
			} else if r.Value != nil {
				c.invalid("object side choice has no value")
			} else {
				conflict.Resolved = true
			}
		}
		c.result.Conflicts = append(c.result.Conflicts, conflict)
	}
	if b != nil && a == nil && reflect.DeepEqual(b, p) {
		c.automatic(l, "accepted_only", c.b.objectSide(l), c.a.objectSide(l), c.p.objectSide(l))
	}
	return a != nil || (b == nil && p != nil)
}

func (f reconciliationFacts) objectSide(l ReconciliationLocator) ReconciliationSide {
	cs, ds := map[string]bool{}, map[string]bool{}
	if l.ComponentID != "" {
		if _, exists := f.components[l.ComponentID]; !exists {
			return ReconciliationSide{}
		}
		cs[l.ComponentID] = true
		for k := range f.relationships {
			if k.source == l.ComponentID || k.target == l.ComponentID {
				cs[k.source] = true
				cs[k.target] = true
			}
		}
		for k := range f.references {
			if k.component == l.ComponentID {
				ds[k.diagram] = true
			}
		}
		for child, anchor := range f.anchors {
			if anchor == l.ComponentID {
				ds[child] = true
			}
		}
		ds[f.homes[l.ComponentID]] = true
	} else {
		if _, exists := f.diagrams[l.DiagramID]; !exists {
			return ReconciliationSide{}
		}
		ds[l.DiagramID] = true
		cs[f.anchors[l.DiagramID]] = true
		for id, home := range f.homes {
			if home == l.DiagramID {
				cs[id] = true
			}
		}
		for k := range f.references {
			if k.diagram == l.DiagramID {
				cs[k.component] = true
			}
		}
		for child, anchor := range f.anchors {
			if f.homes[anchor] == l.DiagramID {
				ds[child] = true
			}
		}
	}
	side := f.compositionSide(compositionLocator("", sortedIDs(cs), sortedIDs(ds)))
	if l.ComponentID != "" {
		v := f.components[l.ComponentID]
		side.Component = &v
	} else {
		side.Diagram = &ReconciliationDiagramObject{ID: l.DiagramID, Title: f.diagrams[l.DiagramID].title, Root: l.DiagramID == f.root}
	}
	return side
}

func (c *reconciliationCalculation) mergeObjects() {
	ids := map[string]bool{}
	for id := range c.b.components {
		ids[id] = true
	}
	for id := range c.a.components {
		ids[id] = true
	}
	for id := range c.p.components {
		ids[id] = true
	}
	for _, id := range sortedIDs(ids) {
		b, bok := c.b.components[id]
		a, aok := c.a.components[id]
		p, pok := c.p.components[id]
		if !c.object(ReconciliationLocator{Kind: "component_object", ComponentID: id}, c.b.componentContext(id), c.a.componentContext(id), c.p.componentContext(id)) {
			delete(c.final.components, id)
			continue
		}
		if !aok {
			a = p
		}
		if aok && pok && bok {
			title := c.merge(ReconciliationLocator{Kind: "component_title", ComponentID: id}, textSide(b.Title, bok), textSide(a.Title, aok), textSide(p.Title, pok))
			body := c.merge(ReconciliationLocator{Kind: "component_description", ComponentID: id}, textSide(b.Description, bok), textSide(a.Description, aok), textSide(p.Description, pok))
			a.Title, a.Description = *title.Text, *body.Text
		} else if !bok && (!aok || !pok || reflect.DeepEqual(c.a.componentContext(id), c.p.componentContext(id))) {
			c.automatic(ReconciliationLocator{Kind: "component_object", ComponentID: id}, additionReason(aok, pok), ReconciliationSide{}, ReconciliationSide{Exists: aok}, ReconciliationSide{Exists: pok})
		}
		c.final.components[id] = a
	}
	ids = map[string]bool{}
	for id := range c.b.diagrams {
		ids[id] = true
	}
	for id := range c.a.diagrams {
		ids[id] = true
	}
	for id := range c.p.diagrams {
		ids[id] = true
	}
	for _, id := range sortedIDs(ids) {
		b, bok := c.b.diagrams[id]
		a, aok := c.a.diagrams[id]
		p, pok := c.p.diagrams[id]
		if !c.object(ReconciliationLocator{Kind: "diagram_object", DiagramID: id}, c.b.diagramContext(id), c.a.diagramContext(id), c.p.diagramContext(id)) {
			delete(c.final.diagrams, id)
			continue
		}
		if !aok {
			a = p
		}
		if aok && pok && bok {
			title := c.merge(ReconciliationLocator{Kind: "diagram_title", DiagramID: id}, textSide(b.title, bok), textSide(a.title, aok), textSide(p.title, pok))
			a.title = *title.Text
		} else if !bok && (!aok || !pok || reflect.DeepEqual(c.a.diagramContext(id), c.p.diagramContext(id))) {
			c.automatic(ReconciliationLocator{Kind: "diagram_object", DiagramID: id}, additionReason(aok, pok), ReconciliationSide{}, ReconciliationSide{Exists: aok}, ReconciliationSide{Exists: pok})
		}
		c.final.diagrams[id] = a
	}
}

func additionReason(a, p bool) string {
	if a && p {
		return "same_result"
	}
	if a {
		return "accepted_only"
	}
	return "proposed_only"
}

func (c *reconciliationCalculation) mergeComposition() {
	c.final.homes = map[string]string{}
	for _, id := range sortedFactIDs(c.b.homes, c.a.homes, c.p.homes) {
		v := c.merge(ReconciliationLocator{Kind: "home", ComponentID: id}, homeSide(c.b.homes[id]), homeSide(c.a.homes[id]), homeSide(c.p.homes[id]))
		if v.Exists {
			c.final.homes[id] = *v.DiagramID
		}
	}
	c.final.anchors = map[string]string{}
	for _, id := range sortedFactIDs(c.b.anchors, c.a.anchors, c.p.anchors) {
		v := c.merge(ReconciliationLocator{Kind: "detail_anchor", DiagramID: id}, anchorSide(c.b.anchors[id]), anchorSide(c.a.anchors[id]), anchorSide(c.p.anchors[id]))
		if v.Exists {
			c.final.anchors[id] = *v.AnchorComponentID
		}
	}
	refs := map[reconciliationPair]bool{}
	for k := range c.b.references {
		refs[k] = true
	}
	for k := range c.a.references {
		refs[k] = true
	}
	for k := range c.p.references {
		refs[k] = true
	}
	keys := make([]reconciliationPair, 0, len(refs))
	for k := range refs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].diagram == keys[j].diagram {
			return keys[i].component < keys[j].component
		}
		return keys[i].diagram < keys[j].diagram
	})
	c.final.references = map[reconciliationPair]bool{}
	for _, k := range keys {
		v := c.merge(ReconciliationLocator{Kind: "reference", DiagramID: k.diagram, ComponentID: k.component}, referenceSide(c.b.references[k]), referenceSide(c.a.references[k]), referenceSide(c.p.references[k]))
		if *v.Present {
			c.final.references[k] = true
		}
	}
	rels := map[reconciliationRelationship]bool{}
	for k := range c.b.relationships {
		rels[k] = true
	}
	for k := range c.a.relationships {
		rels[k] = true
	}
	for k := range c.p.relationships {
		rels[k] = true
	}
	c.final.relationships = map[reconciliationRelationship]int{}
	for _, k := range sortedRelationships(rels) {
		v := c.merge(ReconciliationLocator{Kind: "relationship_count", SourceID: k.source, TargetID: k.target, Label: k.label}, countSide(c.b.relationships[k]), countSide(c.a.relationships[k]), countSide(c.p.relationships[k]))
		if *v.Count > 0 {
			c.final.relationships[k] = *v.Count
		}
	}
}

func sortedRelationships(values map[reconciliationRelationship]bool) []reconciliationRelationship {
	keys := make([]reconciliationRelationship, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.source != b.source {
			return a.source < b.source
		}
		if a.target != b.target {
			return a.target < b.target
		}
		return a.label < b.label
	})
	return keys
}

func sameReconciliationSemantics(a, b reconciliationFacts) bool {
	if a.root != b.root || len(a.components) != len(b.components) || len(a.diagrams) != len(b.diagrams) {
		return false
	}
	for id, c := range a.components {
		other, ok := b.components[id]
		if !ok || c.Title != other.Title || c.Description != other.Description {
			return false
		}
	}
	for id, d := range a.diagrams {
		other, ok := b.diagrams[id]
		if !ok || d.title != other.title {
			return false
		}
	}
	return reflect.DeepEqual(a.homes, b.homes) && reflect.DeepEqual(a.references, b.references) && reflect.DeepEqual(a.anchors, b.anchors) && reflect.DeepEqual(a.relationships, b.relationships)
}
