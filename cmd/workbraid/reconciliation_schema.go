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
	} {
		fields := map[string]*jsonschema.Schema{"kind": constant(unit.kind)}
		for _, id := range unit.ids {
			fields[id] = text()
		}
		locator := closed(fields, append([]string{"kind"}, unit.ids...)...)
		variants = append(variants, closed(map[string]*jsonschema.Schema{"locator": locator, "choice": {Type: "string", Enum: []any{"accepted", "proposed"}}}, "locator", "choice"))
		if unit.value != "" {
			value := text()
			if unit.value == "present" {
				value = &jsonschema.Schema{Type: "boolean"}
			}
			if unit.value == "count" {
				minimum := float64(0)
				value = &jsonschema.Schema{Type: "integer", Minimum: &minimum}
			}
			variants = append(variants, closed(map[string]*jsonschema.Schema{"locator": locator, "choice": constant("manual"), "value": closed(map[string]*jsonschema.Schema{unit.value: value}, unit.value)}, "locator", "choice", "value"))
		}
	}
	ids := &jsonschema.Schema{Type: "array", Items: text(), UniqueItems: true}
	composition := closed(map[string]*jsonschema.Schema{"kind": constant("composition"), "reason": {Type: "string", Enum: []any{"competing_children", "home_reference_overlap", "missing_home", "missing_target", "missing_parent", "unreachable_diagram", "hierarchy_cycle"}}, "component_ids": ids, "diagram_ids": ids}, "kind", "reason", "component_ids", "diagram_ids")
	value, err := jsonschema.For[architecture.ReconciliationValue](nil)
	if err != nil {
		panic(err)
	}
	for _, field := range []string{"text", "count", "present", "diagram_id", "anchor_component_id"} {
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
