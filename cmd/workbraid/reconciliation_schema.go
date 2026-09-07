package main

import (
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"

	"workbraid/internal/agentapi"
	"workbraid/internal/architecture"
)

func reconciliationInputSchema(apply bool) *jsonschema.Schema {
	text := func() *jsonschema.Schema { return &jsonschema.Schema{Type: "string"} }
	constant := func(value string) *jsonschema.Schema {
		var v any = value
		return &jsonschema.Schema{Type: "string", Const: &v}
	}
	closed := func(fields map[string]*jsonschema.Schema, required ...string) *jsonschema.Schema {
		return &jsonschema.Schema{Type: "object", Properties: fields, Required: required, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
	}
	variants := []*jsonschema.Schema{}
	for _, unit := range []struct {
		kind  string
		ids   []string
		value string
	}{
		{"component_title", []string{"component_id"}, "text"}, {"component_description", []string{"component_id"}, "text"},
		{"relationship_count", []string{"source_id", "target_id", "label"}, "count"}, {"diagram_title", []string{"diagram_id"}, "text"},
		{"home", []string{"component_id"}, "diagram_id"}, {"reference", []string{"diagram_id", "component_id"}, "present"}, {"detail_anchor", []string{"diagram_id"}, "anchor_component_id"},
		{"component_object", []string{"component_id"}, ""}, {"diagram_object", []string{"diagram_id"}, ""},
		{"node_position", []string{"diagram_id", "component_id"}, "position"},
		{"node_size", []string{"diagram_id", "component_id"}, "size"},
		{"node_shape", []string{"diagram_id", "component_id"}, "shape"},
		{"diagram_note", []string{"diagram_id", "note_id"}, "note"},
		{"route_value", []string{"diagram_id", "source_id", "target_id", "label", "occurrence"}, "route"},
	} {
		fields := map[string]*jsonschema.Schema{"kind": constant(unit.kind)}
		for _, id := range unit.ids {
			fields[id] = text()
			if id == "occurrence" {
				minimum := float64(1)
				fields[id] = &jsonschema.Schema{Type: "integer", Minimum: &minimum}
			}
		}
		locator := closed(fields, append([]string{"kind"}, unit.ids...)...)
		variants = append(variants, closed(map[string]*jsonschema.Schema{"locator": locator, "choice": {Type: "string", Enum: []any{"accepted", "proposed"}}}, "locator", "choice"))
		if unit.value != "" {
			value := text()
			if unit.value == "shape" {
				value = &jsonschema.Schema{AnyOf: []*jsonschema.Schema{{Type: "string", Enum: []any{"rectangle", "ellipse", "diamond"}}, {Type: "null"}}}
			}
			if unit.value == "note" {
				n, _ := jsonschema.For[architecture.NoteValue](nil)
				boundNoteSchema(n)
				value = &jsonschema.Schema{AnyOf: []*jsonschema.Schema{n, {Type: "null"}}}
			}
			if unit.value == "present" {
				value = &jsonschema.Schema{Type: "boolean"}
			}
			if unit.value == "count" {
				minimum := float64(0)
				value = &jsonschema.Schema{Type: "integer", Minimum: &minimum}
			}
			if unit.value == "position" {
				minimum, maximum := float64(-100000), float64(100000)
				coordinate := &jsonschema.Schema{Type: "integer", Minimum: &minimum, Maximum: &maximum}
				value = closed(map[string]*jsonschema.Schema{"x": coordinate, "y": coordinate}, "x", "y")
			}
			if unit.value == "route" {
				minimum, maximum := float64(-100000), float64(100000)
				value = &jsonschema.Schema{AnyOf: []*jsonschema.Schema{closed(map[string]*jsonschema.Schema{"bend": {Type: "integer", Minimum: &minimum, Maximum: &maximum}}, "bend"), {Type: "null"}}}
			}
			if unit.value == "size" {
				wmin, wmax, hmin, hmax := float64(80), float64(1600), float64(48), float64(1200)
				value = closed(map[string]*jsonschema.Schema{"width": {Type: "integer", Minimum: &wmin, Maximum: &wmax}, "height": {Type: "integer", Minimum: &hmin, Maximum: &hmax}}, "width", "height")
			}
			variants = append(variants, closed(map[string]*jsonschema.Schema{"locator": locator, "choice": constant("manual"), "value": closed(map[string]*jsonschema.Schema{unit.value: value}, unit.value)}, "locator", "choice", "value"))
		}
	}
	loss := closed(map[string]*jsonschema.Schema{"kind": constant("route_loss"), "diagram_id": text(), "source_id": text(), "target_id": text(), "label": text()}, "kind", "diagram_id", "source_id", "target_id", "label")
	variants = append(variants, closed(map[string]*jsonschema.Schema{"locator": loss, "choice": constant("clear")}, "locator", "choice"))
	ids := &jsonschema.Schema{Type: "array", Items: text(), UniqueItems: true}
	composition := closed(map[string]*jsonschema.Schema{"kind": constant("composition"), "reason": {Type: "string", Enum: []any{"competing_children", "home_reference_overlap", "missing_home", "missing_target", "missing_parent", "unreachable_diagram", "hierarchy_cycle"}}, "component_ids": ids, "diagram_ids": ids}, "kind", "reason", "component_ids", "diagram_ids")
	value, err := jsonschema.For[architecture.ReconciliationValue](nil)
	if err != nil {
		panic(err)
	}
	for _, field := range []string{"text", "count", "present", "diagram_id", "anchor_component_id", "position", "size", "route", "shape", "note"} {
		delete(value.Properties, field)
	}
	variants = append(variants, closed(map[string]*jsonschema.Schema{"locator": composition, "choice": {Type: "string", Enum: []any{"accepted", "proposed", "manual"}}, "value": value}, "locator", "choice", "value"))
	options := &jsonschema.ForOptions{TypeSchemas: map[reflect.Type]*jsonschema.Schema{reflect.TypeFor[architecture.ReconciliationResolution](): {OneOf: variants}}}
	var schema *jsonschema.Schema
	if apply {
		schema, err = jsonschema.For[agentapi.ReconciliationApplyRequest](options)
	} else {
		schema, err = jsonschema.For[agentapi.ReconciliationPreviewRequest](options)
	}
	if err != nil {
		panic(err)
	}
	return schema
}
