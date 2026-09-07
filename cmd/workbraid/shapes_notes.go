package main

import (
	"flag"
	"io"
	"strconv"
	"workbraid/internal/agentapi"
)

func parseShapeNoteCommand(op string, f *flag.FlagSet, args []string, stdin io.Reader, invalid invalidCommand) (string, any, *agentapi.Envelope) {
	store := f.String("store-id", "", "exact store UUID")
	proposal := f.String("change-set-id", "", "proposal UUID")
	diagram := f.String("diagram-id", "", "Diagram UUID")
	if op == "diagram_shapes" || op == "diagram_notes" {
		if f.Parse(args) != nil || f.NArg() != 0 || !requireCLI(*store, *diagram) {
			return invalid("Read requires store and Diagram IDs.")
		}
		return op, agentapi.DiagramPositionsRequest{StoreID: *store, ChangeSetID: *proposal, DiagramID: *diagram}, nil
	}
	g := f.String("generation", "", "exact proposal generation")
	var component, id, shape string
	var text, textFile trackedString
	var geometry [4]trackedString
	switch op {
	case "diagram_set_shape", "diagram_restore_default_shape":
		f.StringVar(&component, "component-id", "", "visible Component UUID")
		if op == "diagram_set_shape" {
			f.StringVar(&shape, "shape", "", "rectangle, ellipse or diamond")
		}
	case "diagram_add_note", "diagram_edit_note":
		f.Var(&text, "text", "exact plain text")
		f.Var(&textFile, "text-file", "exact UTF-8 file or - for stdin")
	}
	if op == "diagram_edit_note" || op == "diagram_delete_note" {
		f.StringVar(&id, "note-id", "", "Diagram-local note UUID")
	}
	if op == "diagram_edit_note" {
		for i, k := range []string{"x", "y", "width", "height"} {
			f.Var(&geometry[i], k, "complete integer note geometry")
		}
	}
	if f.Parse(args) != nil || f.NArg() != 0 || !requireCLI(*store, *proposal, *diagram, *g) {
		return invalid("Mutation requires exact store, proposal, generation and Diagram.")
	}
	gen, e := parseRequiredGeneration(*g)
	if e != nil || gen == nil {
		return invalid("Generation must be a non-negative integer.")
	}
	s := agentapi.DiagramAutoLayoutRequest{StatePreconditions: agentapi.StatePreconditions{StoreID: *store, ChangeSetID: *proposal, Generation: *gen}, DiagramID: *diagram}
	if op == "diagram_set_shape" || op == "diagram_restore_default_shape" {
		if component == "" {
			return invalid("Component ID is required.")
		}
		v := agentapi.DiagramRestoreDefaultSizeRequest{DiagramAutoLayoutRequest: s, ComponentID: component}
		if op == "diagram_restore_default_shape" {
			return op, v, nil
		}
		if shape != "rectangle" && shape != "ellipse" && shape != "diamond" {
			return invalid("Shape must be rectangle, ellipse or diamond.")
		}
		return op, agentapi.DiagramSetShapeRequest{DiagramRestoreDefaultSizeRequest: v, Shape: shape}, nil
	}
	n := agentapi.DiagramDeleteNoteRequest{DiagramAutoLayoutRequest: s, NoteID: id}
	if op != "diagram_add_note" && id == "" {
		return invalid("Note ID is required.")
	}
	if op == "diagram_delete_note" {
		return op, n, nil
	}
	value, present, e := exactText(text, textFile, stdin)
	if e != nil || !present {
		return invalid("Use exactly one of --text or --text-file.")
	}
	if op == "diagram_add_note" {
		return op, agentapi.DiagramAddNoteRequest{DiagramAutoLayoutRequest: s, Text: value}, nil
	}
	var ints [4]int
	for i, v := range geometry {
		if !v.set {
			return invalid("Edit requires all note coordinates and dimensions.")
		}
		ints[i], e = strconv.Atoi(v.value)
		if e != nil {
			return invalid("Note geometry must contain integers.")
		}
	}
	return op, agentapi.DiagramEditNoteRequest{DiagramDeleteNoteRequest: n, Text: value, X: ints[0], Y: ints[1], Width: ints[2], Height: ints[3]}, nil
}
