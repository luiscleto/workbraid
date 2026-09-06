package main

import (
	"flag"
	"io"
	"strconv"
	"workbraid/internal/agentapi"
)

func parseRoutingCommand(operation string, flags *flag.FlagSet, args []string, stdin io.Reader, invalid invalidCommand) (string, any, *agentapi.Envelope) {
	store, proposal, generation := addStateFlags(flags)
	diagram := flags.String("diagram-id", "", "Diagram UUID")
	source := flags.String("source-id", "", "source Component UUID")
	target := flags.String("target-id", "", "target Component UUID")
	var label, file, occurrence, bend trackedString
	flags.Var(&label, "label", "exact Relationship label")
	flags.Var(&file, "label-file", "exact UTF-8 label file or - for stdin")
	flags.Var(&occurrence, "occurrence", "one-based exact tuple occurrence")
	if operation == "diagram_set_route" {
		flags.Var(&bend, "bend", "signed control-point distance, integer -100000..100000")
	}
	if flags.Parse(args) != nil || flags.NArg() != 0 || !requireCLI(*store, *proposal, *generation, *diagram, *source, *target) || label.set == file.set || !occurrence.set {
		return invalid("Routing requires exact state, Diagram/source/target IDs, label or label-file, and occurrence.")
	}
	g, e := parseRequiredGeneration(*generation)
	n, ne := strconv.Atoi(occurrence.value)
	text, _, te := exactText(label, file, stdin)
	if e != nil || g == nil || ne != nil || n < 1 || te != nil {
		return invalid("Use exact UTF-8 label, non-negative generation and positive integer occurrence.")
	}
	v := agentapi.DiagramRestoreDefaultRouteRequest{DiagramAutoLayoutRequest: agentapi.DiagramAutoLayoutRequest{StatePreconditions: agentapi.StatePreconditions{StoreID: *store, ChangeSetID: *proposal, Generation: *g}, DiagramID: *diagram}, SourceID: *source, TargetID: *target, Label: text, Occurrence: n}
	if operation == "diagram_restore_default_route" {
		return operation, v, nil
	}
	b, be := strconv.Atoi(bend.value)
	if !bend.set || be != nil || b < -100000 || b > 100000 {
		return invalid("Bend must be an integer from -100000 through 100000.")
	}
	return operation, agentapi.DiagramSetRouteRequest{DiagramRestoreDefaultRouteRequest: v, Bend: b}, nil
}
