package architecture

import "sort"

func compositionLocator(reason string, components, diagrams []string) ReconciliationLocator {
	cs, ds := map[string]bool{}, map[string]bool{}
	for _, id := range components {
		if id != "" {
			cs[id] = true
		}
	}
	for _, id := range diagrams {
		if id != "" {
			ds[id] = true
		}
	}
	return ReconciliationLocator{Kind: "composition", Reason: reason, ComponentIDs: sortedIDs(cs), DiagramIDs: sortedIDs(ds)}
}

// Concrete composition diagnostics for the closed Diagram/home/link model.
// The ordinary loader also uses these checks; constructor output still goes
// through that loader before it can be returned as a candidate.
func compositionProblems(f reconciliationFacts) []ReconciliationLocator {
	problems := map[string]ReconciliationLocator{}
	add := func(reason string, cs, ds []string) {
		l := compositionLocator(reason, cs, ds)
		problems[locatorKey(l)] = l
	}
	for id := range f.components {
		home := f.homes[id]
		if _, ok := f.diagrams[home]; !ok {
			add("missing_home", []string{id}, []string{home})
		}
	}
	for id, home := range f.homes {
		if _, ok := f.components[id]; !ok {
			add("missing_target", []string{id}, []string{home})
		}
	}
	for pair := range f.references {
		_, componentOK := f.components[pair.component]
		_, diagramOK := f.diagrams[pair.diagram]
		if !componentOK || !diagramOK {
			add("missing_target", []string{pair.component}, []string{pair.diagram})
		}
		if f.homes[pair.component] == pair.diagram {
			add("home_reference_overlap", []string{pair.component}, []string{pair.diagram})
		}
	}
	for r, n := range f.relationships {
		if n > 0 {
			_, sourceOK := f.components[r.source]
			_, targetOK := f.components[r.target]
			if !sourceOK || !targetOK {
				add("missing_target", []string{r.source, r.target}, nil)
			}
		}
	}
	children := map[string][]string{}
	for child, anchor := range f.anchors {
		children[anchor] = append(children[anchor], child)
		_, childOK := f.diagrams[child]
		_, anchorOK := f.components[anchor]
		if !childOK || !anchorOK || f.homes[anchor] == "" {
			add("missing_parent", []string{anchor}, []string{child, f.homes[anchor]})
		}
		if child == f.root {
			add("hierarchy_cycle", []string{anchor}, []string{child, f.homes[anchor]})
		}
	}
	for anchor, ds := range children {
		if len(ds) > 1 {
			add("competing_children", []string{anchor}, ds)
		}
	}
	for id := range f.diagrams {
		if id != f.root && f.anchors[id] == "" {
			add("missing_parent", nil, []string{id})
		}
	}
	// Follow each child's unique parent. Collect the exact cycle, rather than
	// labeling every unrelated Diagram unreachable after the first failure.
	for start := range f.diagrams {
		order := []string{}
		positions := map[string]int{}
		current := start
		for current != "" && current != f.root {
			if pos, ok := positions[current]; ok {
				cycle := order[pos:]
				anchors := []string{}
				for _, id := range cycle {
					anchors = append(anchors, f.anchors[id])
				}
				add("hierarchy_cycle", anchors, cycle)
				break
			}
			if _, ok := f.diagrams[current]; !ok {
				break
			}
			positions[current] = len(order)
			order = append(order, current)
			current = f.homes[f.anchors[current]]
		}
	}
	keys := make([]string, 0, len(problems))
	for key := range problems {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]ReconciliationLocator, 0, len(keys))
	for _, key := range keys {
		out = append(out, problems[key])
	}
	return out
}

func (f reconciliationFacts) compositionSide(l ReconciliationLocator) ReconciliationSide {
	value := ReconciliationValue{}
	for _, id := range l.ComponentIDs {
		if home := f.homes[id]; home != "" {
			value.Homes = append(value.Homes, NewComponentHome{ComponentID: id, DiagramID: home})
		}
	}
	for _, id := range l.DiagramIDs {
		if anchor := f.anchors[id]; anchor != "" {
			value.DetailAnchors = append(value.DetailAnchors, DetailReassignment{DiagramID: id, AnchorComponentID: anchor})
		}
	}
	for _, did := range l.DiagramIDs {
		for _, cid := range l.ComponentIDs {
			if _, ok := f.components[cid]; ok {
				if _, ok := f.diagrams[did]; ok {
					value.References = append(value.References, ReferenceAppearanceChange{DiagramID: did, ComponentID: cid, Present: f.references[reconciliationPair{did, cid}]})
				}
			}
		}
	}
	keys := map[reconciliationRelationship]bool{}
	for k := range f.relationships {
		if hasID(l.ComponentIDs, k.source) && hasID(l.ComponentIDs, k.target) {
			keys[k] = true
		}
	}
	for _, k := range sortedRelationships(keys) {
		value.RelationshipCounts = append(value.RelationshipCounts, ReconciliationRelationshipCount{k.source, k.target, k.label, f.relationships[k]})
	}
	return ReconciliationSide{Exists: true, ReconciliationValue: value}
}

func hasID(ids []string, id string) bool {
	for _, value := range ids {
		if value == id {
			return true
		}
	}
	return false
}

func (c *reconciliationCalculation) resolveComposition() {
	initial := compositionProblems(c.final)
	assigned := map[string]ReconciliationSide{}
	for _, conflict := range c.result.Conflicts {
		if conflict.Resolved {
			assigned[locatorKey(conflict.Locator)] = c.final.factSide(conflict.Locator)
		}
	}
	for _, l := range initial {
		contestedAnchors := append([]string{}, l.ComponentIDs...)
		involvedDiagrams := append([]string{}, l.DiagramIDs...)
		// A submitted destination may expand the group to a previously occupied
		// child. The expanded locator must be supplied explicitly on Check.
		selected, found := c.resolutions[locatorKey(l)]
		if !found {
			for _, r := range c.resolutions {
				if r.Locator.Kind != "composition" || r.Locator.Reason != l.Reason || r.Value == nil {
					continue
				}
				expanded := c.expandComposition(l, c.compositionAssignments(l, r))
				if locatorKey(expanded) == locatorKey(r.Locator) {
					selected, found = r, true
					l = expanded
					break
				}
			}
		}
		if !found {
			continue
		}
		c.used[locatorKey(selected.Locator)] = true
		if selected.Value == nil {
			c.invalid("composition requires explicit assignments")
			return
		}
		value := c.compositionAssignments(l, selected)
		expanded := c.expandComposition(l, value)
		if locatorKey(expanded) != locatorKey(l) {
			c.result.Conflicts = append(c.result.Conflicts, c.compositionConflict(expanded, false))
			continue
		}
		if selected.Choice != "accepted" && selected.Choice != "proposed" && selected.Choice != "manual" {
			c.invalid("unknown choice")
			return
		}
		side := c.a
		if selected.Choice == "proposed" {
			side = c.p
		}
		assign := func(locator ReconciliationLocator, value, sideValue ReconciliationSide) bool {
			key := locatorKey(locator)
			if prior, ok := assigned[key]; ok && !equalSide(prior, value) {
				c.invalid("contradictory structural assignments")
				return false
			}
			if selected.Choice != "manual" && sideValue.Exists && !equalSide(sideValue, value) {
				c.invalid("assignment contradicts selected side")
				return false
			}
			assigned[key] = value
			return true
		}
		seen := map[string]bool{}
		for _, home := range value.Homes {
			key := "home:" + home.ComponentID
			if seen[key] || !hasID(l.ComponentIDs, home.ComponentID) || !canonicalUUID(home.DiagramID) {
				c.invalid("unrelated or duplicate home assignment")
				return
			}
			seen[key] = true
			if assign(ReconciliationLocator{Kind: "home", ComponentID: home.ComponentID}, homeSide(home.DiagramID), homeSide(side.homes[home.ComponentID])) {
				c.final.homes[home.ComponentID] = home.DiagramID
			}
		}
		for _, anchor := range value.DetailAnchors {
			key := "anchor:" + anchor.DiagramID
			if seen[key] || !hasID(l.DiagramIDs, anchor.DiagramID) || !canonicalUUID(anchor.AnchorComponentID) {
				c.invalid("unrelated or duplicate child assignment")
				return
			}
			seen[key] = true
			if anchor.DiagramID == c.final.root {
				c.err = &ReconciliationError{"reconciliation_unsupported", "reassign_root"}
				return
			}
			sideValue := anchorSide(side.anchors[anchor.DiagramID])
			if l.Reason == "competing_children" && !hasID(contestedAnchors, side.anchors[anchor.DiagramID]) {
				sideValue.Exists = false
			}
			if assign(ReconciliationLocator{Kind: "detail_anchor", DiagramID: anchor.DiagramID}, anchorSide(anchor.AnchorComponentID), sideValue) {
				c.final.anchors[anchor.DiagramID] = anchor.AnchorComponentID
			}
		}
		for _, ref := range value.References {
			locator := ReconciliationLocator{Kind: "reference", DiagramID: ref.DiagramID, ComponentID: ref.ComponentID}
			key := locatorKey(locator)
			if seen[key] || !hasID(l.ComponentIDs, ref.ComponentID) || !hasID(l.DiagramIDs, ref.DiagramID) {
				c.invalid("unrelated or duplicate reference assignment")
				return
			}
			seen[key] = true
			pair := reconciliationPair{ref.DiagramID, ref.ComponentID}
			sideValue := referenceSide(side.references[pair])
			if _, ok := side.components[ref.ComponentID]; !ok {
				sideValue.Exists = false
			}
			if _, ok := side.diagrams[ref.DiagramID]; !ok {
				sideValue.Exists = false
			}
			if assign(locator, referenceSide(ref.Present), sideValue) {
				if ref.Present {
					c.final.references[pair] = true
				} else {
					delete(c.final.references, pair)
				}
			}
		}
		for _, rel := range value.RelationshipCounts {
			locator := ReconciliationLocator{Kind: "relationship_count", SourceID: rel.SourceID, TargetID: rel.TargetID, Label: rel.Label}
			key := locatorKey(locator)
			fact := reconciliationRelationship{rel.SourceID, rel.TargetID, rel.Label}
			if seen[key] || rel.Count < 0 || !hasID(l.ComponentIDs, rel.SourceID) || !hasID(l.ComponentIDs, rel.TargetID) || (c.b.relationships[fact] == 0 && c.a.relationships[fact] == 0 && c.p.relationships[fact] == 0) {
				c.invalid("unrelated or duplicate Relationship assignment")
				return
			}
			seen[key] = true
			if assign(locator, countSide(rel.Count), countSide(side.relationships[fact])) {
				if rel.Count > 0 {
					c.final.relationships[fact] = rel.Count
				} else {
					delete(c.final.relationships, fact)
				}
			}
		}
		complete := true
		if l.Reason == "competing_children" {
			for _, child := range l.DiagramIDs {
				if !seen["anchor:"+child] {
					complete = false
				}
			}
		} else if selected.Choice != "manual" {
			// Facts without an applicable value on the chosen side must be
			// completed explicitly, even if other assignments broke the cycle.
			for _, id := range l.ComponentIDs {
				if c.final.homes[id] != "" && !seen["home:"+id] {
					complete = false
				}
			}
			for _, id := range l.DiagramIDs {
				if c.final.anchors[id] != "" && !seen["anchor:"+id] {
					complete = false
				}
			}
			for ref := range c.final.references {
				if hasID(l.ComponentIDs, ref.component) && hasID(l.DiagramIDs, ref.diagram) && !seen[locatorKey(ReconciliationLocator{Kind: "reference", DiagramID: ref.diagram, ComponentID: ref.component})] {
					complete = false
				}
			}
			for rel := range c.final.relationships {
				if hasID(l.ComponentIDs, rel.source) && hasID(l.ComponentIDs, rel.target) && !seen[locatorKey(ReconciliationLocator{Kind: "relationship_count", SourceID: rel.source, TargetID: rel.target, Label: rel.label})] {
					complete = false
				}
			}
		}
		for _, child := range l.DiagramIDs {
			if hasID(involvedDiagrams, child) {
				continue
			}
			explicit := false
			for _, anchor := range selected.Value.DetailAnchors {
				explicit = explicit || anchor.DiagramID == child
			}
			if !explicit {
				complete = false
			}
		}
		c.result.Conflicts = append(c.result.Conflicts, c.compositionConflict(l, complete && c.err == nil))
	}
	remaining := compositionProblems(c.final)
	for _, l := range remaining {
		found := false
		for i := range c.result.Conflicts {
			if locatorKey(c.result.Conflicts[i].Locator) == locatorKey(l) {
				c.result.Conflicts[i].Resolved = false
				found = true
			}
		}
		if !found {
			c.result.Conflicts = append(c.result.Conflicts, c.compositionConflict(l, false))
		}
	}
}

// A structural side choice retains all applicable facts in this group, even
// when its explicit value only completes a subset. Missing identities still
// need explicit dependent assignments; they are not a request to drop facts.
// Competing children have their separate complete, explicit assignment rule.
func (c *reconciliationCalculation) compositionAssignments(l ReconciliationLocator, r ReconciliationResolution) ReconciliationValue {
	if l.Reason == "competing_children" || (r.Choice != "accepted" && r.Choice != "proposed") {
		return *r.Value
	}
	side := c.a
	if r.Choice == "proposed" {
		side = c.p
	}
	applicable := side.compositionSide(l).ReconciliationValue
	// A known tuple with two present endpoints also has an applicable zero
	// count on a side without that Relationship. An absent endpoint instead
	// requires an explicit dependent choice, preserving lifecycle semantics.
	relationships := map[reconciliationRelationship]bool{}
	for _, facts := range []reconciliationFacts{c.b, c.a, c.p} {
		for rel := range facts.relationships {
			_, sourceOK := side.components[rel.source]
			_, targetOK := side.components[rel.target]
			if sourceOK && targetOK && hasID(l.ComponentIDs, rel.source) && hasID(l.ComponentIDs, rel.target) {
				relationships[rel] = true
			}
		}
	}
	applicable.RelationshipCounts = nil
	for _, rel := range sortedRelationships(relationships) {
		applicable.RelationshipCounts = append(applicable.RelationshipCounts, ReconciliationRelationshipCount{rel.source, rel.target, rel.label, side.relationships[rel]})
	}
	value := ReconciliationValue{
		Homes:              append([]NewComponentHome{}, r.Value.Homes...),
		DetailAnchors:      append([]DetailReassignment{}, r.Value.DetailAnchors...),
		References:         append([]ReferenceAppearanceChange{}, r.Value.References...),
		RelationshipCounts: append([]ReconciliationRelationshipCount{}, r.Value.RelationshipCounts...),
	}
	homes, anchors := map[string]bool{}, map[string]bool{}
	refs, counts := map[reconciliationPair]bool{}, map[reconciliationRelationship]bool{}
	for _, home := range value.Homes {
		homes[home.ComponentID] = true
	}
	for _, anchor := range value.DetailAnchors {
		anchors[anchor.DiagramID] = true
	}
	for _, ref := range value.References {
		refs[reconciliationPair{ref.DiagramID, ref.ComponentID}] = true
	}
	for _, count := range value.RelationshipCounts {
		counts[reconciliationRelationship{count.SourceID, count.TargetID, count.Label}] = true
	}
	for _, home := range applicable.Homes {
		if !homes[home.ComponentID] {
			value.Homes = append(value.Homes, home)
		}
	}
	for _, anchor := range applicable.DetailAnchors {
		if !anchors[anchor.DiagramID] {
			value.DetailAnchors = append(value.DetailAnchors, anchor)
		}
	}
	for _, ref := range applicable.References {
		if !refs[reconciliationPair{ref.DiagramID, ref.ComponentID}] {
			value.References = append(value.References, ref)
		}
	}
	for _, count := range applicable.RelationshipCounts {
		if !counts[reconciliationRelationship{count.SourceID, count.TargetID, count.Label}] {
			value.RelationshipCounts = append(value.RelationshipCounts, count)
		}
	}
	return value
}

func (f reconciliationFacts) factSide(l ReconciliationLocator) ReconciliationSide {
	switch l.Kind {
	case "home":
		return homeSide(f.homes[l.ComponentID])
	case "detail_anchor":
		return anchorSide(f.anchors[l.DiagramID])
	case "reference":
		return referenceSide(f.references[reconciliationPair{l.DiagramID, l.ComponentID}])
	case "relationship_count":
		return countSide(f.relationships[reconciliationRelationship{l.SourceID, l.TargetID, l.Label}])
	}
	return ReconciliationSide{}
}

func (c *reconciliationCalculation) expandComposition(l ReconciliationLocator, v ReconciliationValue) ReconciliationLocator {
	cs, ds := append([]string{}, l.ComponentIDs...), append([]string{}, l.DiagramIDs...)
	for _, assignment := range v.DetailAnchors {
		if !hasID(l.DiagramIDs, assignment.DiagramID) {
			continue
		}
		for child, anchor := range c.final.anchors {
			if anchor == assignment.AnchorComponentID && !hasID(ds, child) {
				ds = append(ds, child)
				cs = append(cs, anchor)
			}
		}
	}
	return compositionLocator(l.Reason, cs, ds)
}

func (c *reconciliationCalculation) compositionConflict(l ReconciliationLocator, resolved bool) ReconciliationConflict {
	conflict := ReconciliationConflict{Locator: l, Original: c.b.compositionSide(l), Accepted: c.a.compositionSide(l), Proposed: c.p.compositionSide(l), Choices: []string{"accepted", "proposed", "manual"}, Resolved: resolved}
	// Candidates are only suggestions for a direct displacement. Explicit joint
	// swaps may additionally name occupied anchors and expand their context.
	for id := range c.final.components {
		occupied := false
		for child, anchor := range c.final.anchors {
			if anchor == id && !hasID(l.DiagramIDs, child) {
				occupied = true
			}
		}
		if occupied || c.final.homes[id] == "" {
			continue
		}
		valid := true
		for _, child := range l.DiagramIDs {
			current := c.final.homes[id]
			seen := map[string]bool{}
			for current != "" && !seen[current] {
				if current == child {
					valid = false
					break
				}
				seen[current] = true
				current = c.final.homes[c.final.anchors[current]]
			}
		}
		if valid {
			conflict.EligibleAnchors = append(conflict.EligibleAnchors, id)
		}
	}
	sort.Strings(conflict.EligibleAnchors)
	for _, id := range conflict.EligibleAnchors {
		home := c.final.homes[id]
		conflict.EligibleParents = append(conflict.EligibleParents, ReconciliationParentOption{ComponentID: id, Title: c.final.components[id].Title, HomeDiagramID: home, HomeDiagramTitle: c.final.diagrams[home].title})
	}
	return conflict
}
