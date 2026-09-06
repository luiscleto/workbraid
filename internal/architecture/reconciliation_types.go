package architecture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
)

// These are request-local fact identities and choices, never durable records.
type ReconciliationLocator struct {
	Kind         string   `json:"kind"`
	ComponentID  string   `json:"component_id,omitempty"`
	DiagramID    string   `json:"diagram_id,omitempty"`
	SourceID     string   `json:"source_id,omitempty"`
	TargetID     string   `json:"target_id,omitempty"`
	Label        string   `json:"label,omitempty"`
	Reason       string   `json:"reason,omitempty"`
	ComponentIDs []string `json:"component_ids,omitempty"`
	DiagramIDs   []string `json:"diagram_ids,omitempty"`
}

func (l ReconciliationLocator) MarshalJSON() ([]byte, error) {
	type plain ReconciliationLocator
	if l.Kind == "relationship_count" {
		return json.Marshal(struct {
			Kind     string `json:"kind"`
			SourceID string `json:"source_id"`
			TargetID string `json:"target_id"`
			Label    string `json:"label"`
		}{l.Kind, l.SourceID, l.TargetID, l.Label})
	}
	if l.Kind != "composition" {
		return json.Marshal(plain(l))
	}
	return json.Marshal(struct {
		Kind         string   `json:"kind"`
		Reason       string   `json:"reason"`
		ComponentIDs []string `json:"component_ids"`
		DiagramIDs   []string `json:"diagram_ids"`
	}{l.Kind, l.Reason, append([]string{}, l.ComponentIDs...), append([]string{}, l.DiagramIDs...)})
}

type ReconciliationValue struct {
	Text               *string                           `json:"text,omitempty"`
	Count              *int                              `json:"count,omitempty"`
	Present            *bool                             `json:"present,omitempty"`
	DiagramID          *string                           `json:"diagram_id,omitempty"`
	AnchorComponentID  *string                           `json:"anchor_component_id,omitempty"`
	Homes              []NewComponentHome                `json:"homes,omitempty"`
	References         []ReferenceAppearanceChange       `json:"references,omitempty"`
	DetailAnchors      []DetailReassignment              `json:"detail_anchors,omitempty"`
	RelationshipCounts []ReconciliationRelationshipCount `json:"relationship_counts,omitempty"`
}

type ReconciliationRelationshipCount struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Label    string `json:"label"`
	Count    int    `json:"count"`
}

type ReconciliationResolution struct {
	Locator ReconciliationLocator `json:"locator"`
	Choice  string                `json:"choice"`
	Value   *ReconciliationValue  `json:"value,omitempty"`
}

// Strict decoding also catches irrelevant zero-valued fields, nulls, duplicate
// keys and omitted values. A union's closed shape is part of its wire contract.
func (r *ReconciliationResolution) UnmarshalJSON(data []byte) error {
	if err := uniqueJSONKeys(data); err != nil {
		return err
	}
	type plain ReconciliationResolution
	var value plain
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&value); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw) < 2 || raw["locator"] == nil || raw["choice"] == nil {
		return fmt.Errorf("resolution requires locator and choice")
	}
	var loc map[string]json.RawMessage
	if err := json.Unmarshal(raw["locator"], &loc); err != nil {
		return err
	}
	required := []string{"kind"}
	switch value.Locator.Kind {
	case "component_title", "component_description", "home", "component_object":
		required = append(required, "component_id")
	case "diagram_title", "detail_anchor", "diagram_object":
		required = append(required, "diagram_id")
	case "reference":
		required = append(required, "component_id", "diagram_id")
	case "relationship_count":
		required = append(required, "source_id", "target_id", "label")
	case "composition":
		required = append(required, "reason", "component_ids", "diagram_ids")
	default:
		return fmt.Errorf("unknown reconciliation locator")
	}
	if len(loc) != len(required) {
		return fmt.Errorf("irrelevant locator fields")
	}
	for _, key := range required {
		if loc[key] == nil || bytes.Equal(loc[key], []byte("null")) {
			return fmt.Errorf("missing locator %s", key)
		}
		if key == "component_id" || key == "diagram_id" || key == "source_id" || key == "target_id" {
			var id string
			if json.Unmarshal(loc[key], &id) != nil || !canonicalUUID(id) {
				return fmt.Errorf("invalid locator UUID")
			}
		}
	}
	for _, id := range []string{value.Locator.ComponentID, value.Locator.DiagramID, value.Locator.SourceID, value.Locator.TargetID} {
		if id != "" && !canonicalUUID(id) {
			return fmt.Errorf("invalid locator UUID")
		}
	}
	if value.Locator.Kind == "composition" {
		for _, ids := range [][]string{value.Locator.ComponentIDs, value.Locator.DiagramIDs} {
			for i, id := range ids {
				if !canonicalUUID(id) || (i > 0 && ids[i-1] >= id) {
					return fmt.Errorf("composition IDs must be sorted unique UUIDs")
				}
			}
		}
	}
	if value.Choice != "accepted" && value.Choice != "proposed" && value.Choice != "manual" {
		return fmt.Errorf("unknown resolution choice")
	}
	needsValue := value.Choice == "manual" || value.Locator.Kind == "composition"
	if needsValue != (value.Value != nil) || (!needsValue && raw["value"] != nil) {
		return fmt.Errorf("resolution value has wrong presence")
	}
	if needsValue {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw["value"], &fields); err != nil {
			return err
		}
		allowed := map[string]bool{}
		switch value.Locator.Kind {
		case "component_title", "component_description", "diagram_title":
			allowed["text"] = true
		case "relationship_count":
			allowed["count"] = true
		case "reference":
			allowed["present"] = true
		case "home":
			allowed["diagram_id"] = true
		case "detail_anchor":
			allowed["anchor_component_id"] = true
		case "composition":
			for _, key := range []string{"homes", "references", "detail_anchors", "relationship_counts"} {
				allowed[key] = true
			}
		default:
			return fmt.Errorf("object replacement is unsupported")
		}
		if value.Locator.Kind != "composition" && len(fields) != 1 {
			return fmt.Errorf("manual scalar requires one value")
		}
		for key, rawValue := range fields {
			if !allowed[key] || bytes.Equal(rawValue, []byte("null")) {
				return fmt.Errorf("irrelevant/null resolution value %s", key)
			}
			if value.Locator.Kind == "composition" {
				var entries []map[string]json.RawMessage
				if err := json.Unmarshal(rawValue, &entries); err != nil {
					return err
				}
				required := []string{}
				switch key {
				case "homes":
					required = []string{"component_id", "diagram_id"}
				case "references":
					required = []string{"diagram_id", "component_id", "present"}
				case "detail_anchors":
					required = []string{"diagram_id", "anchor_component_id"}
				case "relationship_counts":
					required = []string{"source_id", "target_id", "label", "count"}
				}
				for _, entry := range entries {
					if len(entry) != len(required) {
						return fmt.Errorf("incomplete structural assignment")
					}
					for _, name := range required {
						if entry[name] == nil || bytes.Equal(entry[name], []byte("null")) {
							return fmt.Errorf("missing structural assignment value")
						}
					}
				}
			}
		}
	}
	*r = ReconciliationResolution(value)
	return nil
}

func uniqueJSONKeys(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	var read func() error
	read = func() error {
		token, err := d.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok {
			seen := map[string]bool{}
			for d.More() {
				if delim == '{' {
					key, err := d.Token()
					if err != nil {
						return err
					}
					name := key.(string)
					if seen[name] {
						return fmt.Errorf("duplicate JSON key %s", name)
					}
					seen[name] = true
				}
				if err := read(); err != nil {
					return err
				}
			}
			_, err = d.Token()
			return err
		}
		return nil
	}
	if err := read(); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON")
	}
	return nil
}

type ReconciliationSide struct {
	Exists bool `json:"exists"`
	ReconciliationValue
	Component *AuthoringComponent          `json:"component,omitempty"`
	Diagram   *ReconciliationDiagramObject `json:"diagram,omitempty"`
}

type ReconciliationDiagramObject struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Root  bool   `json:"root"`
}

type ReconciliationConflict struct {
	Locator         ReconciliationLocator        `json:"locator"`
	Original        ReconciliationSide           `json:"original"`
	Accepted        ReconciliationSide           `json:"accepted"`
	Proposed        ReconciliationSide           `json:"proposed"`
	Choices         []string                     `json:"choices"`
	Unsupported     map[string]string            `json:"unsupported,omitempty"`
	EligibleAnchors []string                     `json:"eligible_anchor_component_ids,omitempty"`
	EligibleParents []ReconciliationParentOption `json:"eligible_parents,omitempty"`
	Resolved        bool                         `json:"resolved"`
}

type ReconciliationParentOption struct {
	ComponentID      string `json:"component_id"`
	Title            string `json:"title"`
	HomeDiagramID    string `json:"home_diagram_id"`
	HomeDiagramTitle string `json:"home_diagram_title"`
}

type ReconciliationAutomaticChange struct {
	Locator  ReconciliationLocator `json:"locator"`
	Reason   string                `json:"reason"`
	Original ReconciliationSide    `json:"original"`
	Accepted ReconciliationSide    `json:"accepted"`
	Proposed ReconciliationSide    `json:"proposed"`
}

type Reconciliation struct {
	Status           string                          `json:"status"`
	AutomaticChanges []ReconciliationAutomaticChange `json:"automatic_changes"`
	Conflicts        []ReconciliationConflict        `json:"conflicts"`
	Changes          []ComponentChange               `json:"-"`
	Composition      CandidateComposition            `json:"-"`
	Candidate        *Candidate                      `json:"-"`
}

type ReconciliationError struct{ Code, Reason string }

func (e *ReconciliationError) Error() string { return e.Code + ": " + e.Reason }

type reconciliationPair struct{ diagram, component string }
type reconciliationRelationship struct{ source, target, label string }
type reconciliationDiagram struct{ title, path string }

type reconciliationFacts struct {
	components    map[string]AuthoringComponent
	diagrams      map[string]reconciliationDiagram
	homes         map[string]string
	references    map[reconciliationPair]bool
	anchors       map[string]string
	relationships map[reconciliationRelationship]int
	root          string
}

func snapshotReconciliationFacts(s Snapshot) reconciliationFacts {
	f := reconciliationFacts{components: map[string]AuthoringComponent{}, diagrams: map[string]reconciliationDiagram{}, homes: map[string]string{}, references: map[reconciliationPair]bool{}, anchors: map[string]string{}, relationships: map[reconciliationRelationship]int{}, root: s.RootDiagramID()}
	for _, c := range s.AuthoringComponents() {
		f.components[c.ID] = c
		for _, rel := range c.Relationships {
			f.relationships[reconciliationRelationship{c.ID, rel.TargetID, rel.Label}]++
		}
	}
	for _, d := range s.diagrams {
		id := d.id.String()
		f.diagrams[id] = reconciliationDiagram{d.title, d.path}
		for _, a := range d.appearances {
			cid := a.component.String()
			if a.role == "home" {
				f.homes[cid] = id
			} else {
				f.references[reconciliationPair{id, cid}] = true
			}
			if a.hasDetailLink {
				f.anchors[a.detailDiagram.String()] = cid
			}
		}
	}
	return f
}

func locatorKey(l ReconciliationLocator) string { data, _ := json.Marshal(l); return string(data) }
func textSide(text string, exists bool) ReconciliationSide {
	if !exists {
		return ReconciliationSide{}
	}
	return ReconciliationSide{Exists: true, ReconciliationValue: ReconciliationValue{Text: &text}}
}
func homeSide(id string) ReconciliationSide {
	if id == "" {
		return ReconciliationSide{}
	}
	return ReconciliationSide{Exists: true, ReconciliationValue: ReconciliationValue{DiagramID: &id}}
}
func anchorSide(id string) ReconciliationSide {
	if id == "" {
		return ReconciliationSide{}
	}
	return ReconciliationSide{Exists: true, ReconciliationValue: ReconciliationValue{AnchorComponentID: &id}}
}
func countSide(count int) ReconciliationSide {
	return ReconciliationSide{Exists: true, ReconciliationValue: ReconciliationValue{Count: &count}}
}
func referenceSide(present bool) ReconciliationSide {
	return ReconciliationSide{Exists: true, ReconciliationValue: ReconciliationValue{Present: &present}}
}
func sortedFactIDs(a, b, c map[string]string) []string {
	ids := map[string]bool{}
	for id := range a {
		ids[id] = true
	}
	for id := range b {
		ids[id] = true
	}
	for id := range c {
		ids[id] = true
	}
	return sortedIDs(ids)
}
func sortedIDs(ids map[string]bool) []string {
	out := make([]string, 0, len(ids))
	for id := range ids {
		if id != "" {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}
func equalSide(a, b ReconciliationSide) bool { return reflect.DeepEqual(a, b) }
