package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"workbraid/internal/agentapi"
	"workbraid/internal/web"
)

//go:embed skill.md
var embeddedSkill string

type globalOptions struct {
	listen        string
	dataDirectory string
	uiDirectory   string
	server        string
	json          bool
}

type trackedString struct {
	value string
	set   bool
}

func (value *trackedString) String() string { return value.value }

func (value *trackedString) Set(input string) error {
	value.value = input
	value.set = true
	return nil
}

func run(args []string, stdout, stderr io.Writer, stdin io.Reader) int {
	if len(args) == 1 && args[0] == "--skill" {
		_, _ = io.WriteString(stdout, embeddedSkill)
		return 0
	}
	jsonRequested := false
	for _, arg := range args {
		if arg == "--json" {
			jsonRequested = true
		}
	}
	options := globalOptions{}
	global := flag.NewFlagSet("workbraid", flag.ContinueOnError)
	global.SetOutput(io.Discard)
	global.StringVar(&options.listen, "listen", "127.0.0.1:8080", "literal loopback listen address")
	global.StringVar(&options.dataDirectory, "data-dir", defaultDataDirectory(), "application-data directory")
	global.StringVar(&options.uiDirectory, "ui-dir", "frontend/dist", "built browser UI directory")
	defaultServer := os.Getenv("WORKBRAID_SERVER")
	if defaultServer == "" {
		defaultServer = "http://127.0.0.1:8080"
	}
	global.StringVar(&options.server, "server", defaultServer, "running WorkBraid loopback URL")
	global.BoolVar(&options.json, "json", false, "emit one machine JSON envelope")
	if err := global.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = io.WriteString(stdout, cliHelp)
			return 0
		}
		return writeCLIEnvelope(stdout, jsonRequested, agentapi.Failure("invalid_request", "Correct the global flags and try again.", nil))
	}
	rest := global.Args()
	if len(rest) == 0 {
		return runServer(options, stderr)
	}
	serverOnlyFlag := ""
	global.Visit(func(value *flag.Flag) {
		if value.Name == "listen" || value.Name == "data-dir" || value.Name == "ui-dir" {
			serverOnlyFlag = value.Name
		}
	})
	if serverOnlyFlag != "" {
		return writeCLIEnvelope(stdout, options.json, agentapi.Failure("invalid_request", "Client and MCP modes do not accept server data or listener flags.", map[string]any{"flag": serverOnlyFlag}))
	}
	if rest[0] == "--skill" {
		return writeCLIEnvelope(stdout, options.json, agentapi.Failure("invalid_request", "Use --skill as the standalone WorkBraid command.", nil))
	}
	if rest[0] == "--help" || rest[0] == "help" {
		_, _ = io.WriteString(stdout, cliHelp)
		return 0
	}
	if rest[0] == "mcp" {
		if len(rest) != 1 || options.json {
			return writeCLIEnvelope(stdout, options.json, agentapi.Failure("invalid_request", "Use `workbraid [--server <loopback-url>] mcp`.", nil))
		}
		return runMCP(options.server, stdin, stdout, stderr)
	}
	client, failure := agentapi.NewClient(options.server)
	if failure != nil {
		return writeCLIEnvelope(stdout, options.json, *failure)
	}
	operation, input, envelope := parseDomainCommand(rest, stdin)
	if envelope != nil {
		return writeCLIEnvelope(stdout, options.json, *envelope)
	}
	result := client.Call(context.Background(), operation, input)
	return writeCLIEnvelope(stdout, options.json, result)
}

func runServer(options globalOptions, stderr io.Writer) int {
	expectedOrigin, err := originForLoopbackAddress(options.listen)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.MkdirAll(options.dataDirectory, 0o700); err != nil {
		_, _ = fmt.Fprintf(stderr, "create application-data directory: %v\n", err)
		return 1
	}
	handler := web.NewHandler(expectedOrigin, options.uiDirectory, options.dataDirectory)
	logger := log.New(stderr, "", log.LstdFlags)
	logger.Printf("WorkBraid is available at %s", expectedOrigin)
	if err := http.ListenAndServe(options.listen, handler); err != nil {
		logger.Print(err)
		return 1
	}
	return 0
}

func writeCLIEnvelope(writer io.Writer, jsonMode bool, envelope agentapi.Envelope) int {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	if !jsonMode {
		encoder.SetIndent("", "  ")
	}
	_ = encoder.Encode(envelope)
	if envelope.OK {
		return 0
	}
	return 1
}

func parseDomainCommand(args []string, stdin io.Reader) (string, any, *agentapi.Envelope) {
	if len(args) == 1 {
		switch args[0] {
		case "status":
			return "status", nil, nil
		}
	}
	if len(args) < 2 {
		failure := agentapi.Failure("invalid_request", "Choose a complete WorkBraid command. Use --help to list commands.", nil)
		return "", nil, &failure
	}
	group, action, actionArgs := args[0], args[1], args[2:]
	operation := group + "_" + strings.ReplaceAll(action, "-", "_")
	if group == "project" && action == "list" {
		return noArgumentCommand(operation, actionArgs)
	}
	if group == "project" && action == "current" {
		return noArgumentCommand(operation, actionArgs)
	}
	if group == "architecture" && action == "inspect" {
		return noArgumentCommand(operation, actionArgs)
	}
	if group == "changes" && action == "inspect" {
		return noArgumentCommand(operation, actionArgs)
	}
	flags := flag.NewFlagSet(group+" "+action, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	invalid := func(message string) (string, any, *agentapi.Envelope) {
		failure := agentapi.Failure("invalid_request", message, map[string]any{"command": group + " " + action})
		return "", nil, &failure
	}

	switch operation {
	case "project_create":
		var name trackedString
		flags.Var(&name, "name", "project name")
		if flags.Parse(actionArgs) != nil || flags.NArg() != 0 || !name.set {
			return invalid("Project create requires --name.")
		}
		return operation, agentapi.ProjectCreateRequest{Name: name.value}, nil
	case "project_open":
		var slug trackedString
		flags.Var(&slug, "slug", "project slug")
		if flags.Parse(actionArgs) != nil || flags.NArg() != 0 || !slug.set || slug.value == "" {
			return invalid("Project open requires --slug.")
		}
		return operation, agentapi.ProjectOpenRequest{Slug: slug.value}, nil
	case "project_close":
		var storeID string
		flags.StringVar(&storeID, "store-id", "", "exact store UUID")
		if flags.Parse(actionArgs) != nil || flags.NArg() != 0 || storeID == "" {
			return invalid("Project close requires --store-id.")
		}
		return operation, agentapi.ProjectCloseRequest{StoreID: storeID}, nil
	case "architecture_refresh":
		var storeID, revision string
		flags.StringVar(&storeID, "store-id", "", "exact store UUID")
		flags.StringVar(&revision, "accepted-revision", "", "inspected accepted revision")
		if flags.Parse(actionArgs) != nil || flags.NArg() != 0 || !requireCLI(storeID, revision) {
			return invalid("Architecture refresh requires --store-id and --accepted-revision.")
		}
		return operation, agentapi.ArchitectureRefreshRequest{StoreID: storeID, AcceptedRevision: revision}, nil
	case "architecture_update":
		var storeID, base, tree, generation string
		flags.StringVar(&storeID, "store-id", "", "exact store UUID")
		flags.StringVar(&base, "base-revision", "", "reviewed base commit")
		flags.StringVar(&tree, "candidate-tree", "", "reviewed candidate tree")
		flags.StringVar(&generation, "generation", "", "reviewed pending generation")
		if flags.Parse(actionArgs) != nil || flags.NArg() != 0 || !requireCLI(storeID, base, tree, generation) {
			return invalid("Architecture update requires the exact --store-id, --base-revision, --candidate-tree, and --generation review binding.")
		}
		parsed, err := parseRequiredGeneration(generation)
		if err != nil || parsed == nil {
			return invalid("Architecture update generation must be a non-negative integer.")
		}
		return operation, agentapi.ArchitectureUpdateRequest{StoreID: storeID, BaseRevision: base, CandidateTree: tree, Generation: *parsed}, nil
	case "changes_review":
		state, generation, ok := parseStateFlags(flags, actionArgs)
		if !ok || generation == nil {
			return invalid("Changes review requires exact state and a numeric --generation.")
		}
		return operation, agentapi.ChangesReviewRequest{StatePreconditions: state, Generation: *generation}, nil
	case "changes_discard":
		var storeID, generation string
		flags.StringVar(&storeID, "store-id", "", "exact store UUID")
		flags.StringVar(&generation, "generation", "", "inspected pending generation")
		if flags.Parse(actionArgs) != nil || flags.NArg() != 0 || !requireCLI(storeID, generation) {
			return invalid("Changes discard requires --store-id and numeric --generation.")
		}
		parsed, err := parseRequiredGeneration(generation)
		if err != nil || parsed == nil {
			return invalid("Changes discard generation must be a non-negative integer.")
		}
		return operation, agentapi.ChangesDiscardRequest{StoreID: storeID, Generation: *parsed}, nil
	case "component_create":
		return parseComponentCreate(flags, actionArgs, stdin, invalid)
	case "component_edit":
		return parseComponentEdit(flags, actionArgs, stdin, invalid)
	case "component_move_home":
		return parseComponentMove(flags, actionArgs, invalid)
	case "relationship_add":
		return parseRelationshipAdd(flags, actionArgs, stdin, invalid)
	case "relationship_edit":
		return parseRelationshipEdit(flags, actionArgs, stdin, invalid)
	case "relationship_remove":
		return parseRelationshipRemove(flags, actionArgs, stdin, invalid)
	case "diagram_create_detail", "diagram_edit_title", "diagram_show_component", "diagram_stop_showing_component":
		return parseDiagramCommand(operation, flags, actionArgs, invalid)
	default:
		return invalid("That WorkBraid command is not available. Use --help to list commands.")
	}
}

type invalidCommand func(string) (string, any, *agentapi.Envelope)

func noArgumentCommand(operation string, args []string) (string, any, *agentapi.Envelope) {
	if len(args) != 0 {
		failure := agentapi.Failure("invalid_request", "This read command takes no action flags.", nil)
		return "", nil, &failure
	}
	return operation, nil, nil
}

func addStateFlags(flags *flag.FlagSet) (*string, *string, *string) {
	return flags.String("store-id", "", "exact store UUID"), flags.String("accepted-revision", "", "inspected accepted revision"), flags.String("generation", "", "inspected pending generation or none")
}

func parseStateFlags(flags *flag.FlagSet, args []string) (agentapi.StatePreconditions, *uint64, bool) {
	storeID, revision, generation := addStateFlags(flags)
	if flags.Parse(args) != nil || flags.NArg() != 0 || !requireCLI(*storeID, *revision, *generation) {
		return agentapi.StatePreconditions{}, nil, false
	}
	parsed, err := parseRequiredGeneration(*generation)
	if err != nil {
		return agentapi.StatePreconditions{}, nil, false
	}
	return agentapi.StatePreconditions{StoreID: *storeID, AcceptedRevision: *revision, PendingGeneration: parsed}, parsed, true
}

func parseRequiredGeneration(value string) (*uint64, error) {
	if value == "none" {
		return nil, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func requireCLI(values ...string) bool {
	for _, value := range values {
		if value == "" {
			return false
		}
	}
	return true
}

func exactText(literal, file trackedString, stdin io.Reader) (string, bool, error) {
	if literal.set == file.set {
		return "", false, nil
	}
	if literal.set {
		return literal.value, true, nil
	}
	var data []byte
	var err error
	if file.value == "-" {
		data, err = io.ReadAll(io.LimitReader(stdin, 4<<20))
	} else {
		data, err = os.ReadFile(file.value)
	}
	if err != nil {
		return "", true, err
	}
	if !utf8.Valid(data) {
		return "", true, fmt.Errorf("input is not UTF-8")
	}
	return string(data), true, nil
}

func parseComponentCreate(flags *flag.FlagSet, args []string, stdin io.Reader, invalid invalidCommand) (string, any, *agentapi.Envelope) {
	storeID, revision, generation := addStateFlags(flags)
	var title, description, descriptionFile trackedString
	var diagramID trackedString
	flags.Var(&title, "title", "Component title")
	flags.Var(&description, "description", "Markdown description")
	flags.Var(&descriptionFile, "description-file", "Markdown file or - for stdin")
	flags.Var(&diagramID, "diagram-id", "home Diagram ID")
	if flags.Parse(args) != nil || flags.NArg() != 0 || !requireCLI(*storeID, *revision, *generation) || !title.set || (description.set && descriptionFile.set) {
		return invalid("Component create requires exact state and --title; use at most one description input.")
	}
	parsed, err := parseRequiredGeneration(*generation)
	if err != nil {
		return invalid("Generation must be a non-negative integer or none.")
	}
	text := ""
	if description.set || descriptionFile.set {
		text, _, err = exactText(description, descriptionFile, stdin)
		if err != nil {
			return invalid("The Description input could not be read as exact UTF-8.")
		}
	}
	var diagram *string
	if diagramID.set {
		diagram = &diagramID.value
	}
	return "component_create", agentapi.ComponentCreateRequest{StatePreconditions: agentapi.StatePreconditions{StoreID: *storeID, AcceptedRevision: *revision, PendingGeneration: parsed}, Title: title.value, Description: text, DiagramID: diagram}, nil
}

func parseComponentEdit(flags *flag.FlagSet, args []string, stdin io.Reader, invalid invalidCommand) (string, any, *agentapi.Envelope) {
	storeID, revision, generation := addStateFlags(flags)
	var componentID string
	var title, description, descriptionFile trackedString
	flags.StringVar(&componentID, "component-id", "", "stable Component ID")
	flags.Var(&title, "title", "new Component title")
	flags.Var(&description, "description", "new Markdown description")
	flags.Var(&descriptionFile, "description-file", "new Markdown file or - for stdin")
	if flags.Parse(args) != nil || flags.NArg() != 0 || !requireCLI(*storeID, *revision, *generation, componentID) || (description.set && descriptionFile.set) || (!title.set && !description.set && !descriptionFile.set) {
		return invalid("Component edit requires exact state, --component-id, and at least one changed field.")
	}
	parsed, err := parseRequiredGeneration(*generation)
	if err != nil {
		return invalid("Generation must be a non-negative integer or none.")
	}
	var titleValue, descriptionValue *string
	if title.set {
		titleValue = &title.value
	}
	if description.set || descriptionFile.set {
		text, _, readErr := exactText(description, descriptionFile, stdin)
		if readErr != nil {
			return invalid("The Description input could not be read as exact UTF-8.")
		}
		descriptionValue = &text
	}
	return "component_edit", agentapi.ComponentEditRequest{StatePreconditions: agentapi.StatePreconditions{StoreID: *storeID, AcceptedRevision: *revision, PendingGeneration: parsed}, ComponentID: componentID, Title: titleValue, Description: descriptionValue}, nil
}

func parseComponentMove(flags *flag.FlagSet, args []string, invalid invalidCommand) (string, any, *agentapi.Envelope) {
	storeID, revision, generation := addStateFlags(flags)
	var componentID, diagramID string
	flags.StringVar(&componentID, "component-id", "", "stable Component ID")
	flags.StringVar(&diagramID, "diagram-id", "", "destination Diagram ID")
	if flags.Parse(args) != nil || flags.NArg() != 0 || !requireCLI(*storeID, *revision, *generation, componentID, diagramID) {
		return invalid("Component move-home requires exact state, --component-id, and --diagram-id.")
	}
	parsed, err := parseRequiredGeneration(*generation)
	if err != nil {
		return invalid("Generation must be a non-negative integer or none.")
	}
	return "component_move_home", agentapi.ComponentMoveHomeRequest{StatePreconditions: agentapi.StatePreconditions{StoreID: *storeID, AcceptedRevision: *revision, PendingGeneration: parsed}, ComponentID: componentID, DiagramID: diagramID}, nil
}

func relationshipState(flags *flag.FlagSet) (*string, *string, *string, *string) {
	storeID, revision, generation := addStateFlags(flags)
	sourceID := flags.String("source-id", "", "stable source Component ID")
	_ = storeID
	_ = revision
	_ = generation
	return storeID, revision, generation, sourceID
}

func parseRelationshipAdd(flags *flag.FlagSet, args []string, stdin io.Reader, invalid invalidCommand) (string, any, *agentapi.Envelope) {
	storeID, revision, generation, sourceID := relationshipState(flags)
	var targetID trackedString
	var label, labelFile trackedString
	flags.Var(&targetID, "target-id", "raw target Component ID")
	flags.Var(&label, "label", "exact Relationship label")
	flags.Var(&labelFile, "label-file", "exact label file or - for stdin")
	if flags.Parse(args) != nil || flags.NArg() != 0 || !requireCLI(*storeID, *revision, *generation, *sourceID) || !targetID.set {
		return invalid("Relationship add requires exact state, --source-id, --target-id, and one label input.")
	}
	parsed, err := parseRequiredGeneration(*generation)
	if err != nil {
		return invalid("Generation must be a non-negative integer or none.")
	}
	text, present, err := exactText(label, labelFile, stdin)
	if err != nil || !present {
		return invalid("Use exactly one of --label or --label-file.")
	}
	return "relationship_add", agentapi.RelationshipAddRequest{StatePreconditions: agentapi.StatePreconditions{StoreID: *storeID, AcceptedRevision: *revision, PendingGeneration: parsed}, SourceID: *sourceID, TargetID: targetID.value, Label: text}, nil
}

func parseRelationshipEdit(flags *flag.FlagSet, args []string, stdin io.Reader, invalid invalidCommand) (string, any, *agentapi.Envelope) {
	storeID, revision, generation, sourceID := relationshipState(flags)
	var oldTarget, target trackedString
	var oldLabel, oldLabelFile, label, labelFile trackedString
	occurrence := flags.Int("occurrence", 1, "one-based matching raw occurrence")
	flags.Var(&oldTarget, "old-target-id", "exact raw old target")
	flags.Var(&oldLabel, "old-label", "exact raw old label")
	flags.Var(&oldLabelFile, "old-label-file", "exact raw old label file")
	flags.Var(&target, "target-id", "exact new target")
	flags.Var(&label, "label", "exact new label")
	flags.Var(&labelFile, "label-file", "exact new label file")
	if flags.Parse(args) != nil || flags.NArg() != 0 || !requireCLI(*storeID, *revision, *generation, *sourceID) || !oldTarget.set || !target.set || *occurrence < 1 {
		return invalid("Relationship edit requires exact state, raw old selector, occurrence, and new target/label.")
	}
	parsed, err := parseRequiredGeneration(*generation)
	if err != nil {
		return invalid("Generation must be a non-negative integer or none.")
	}
	oldText, oldPresent, oldErr := exactText(oldLabel, oldLabelFile, stdin)
	newText, newPresent, newErr := exactText(label, labelFile, stdin)
	if oldErr != nil || newErr != nil || !oldPresent || !newPresent {
		return invalid("Use exactly one old-label input and one new label input.")
	}
	return "relationship_edit", agentapi.RelationshipEditRequest{StatePreconditions: agentapi.StatePreconditions{StoreID: *storeID, AcceptedRevision: *revision, PendingGeneration: parsed}, SourceID: *sourceID, OldTargetID: oldTarget.value, OldLabel: oldText, Occurrence: *occurrence, TargetID: target.value, Label: newText}, nil
}

func parseRelationshipRemove(flags *flag.FlagSet, args []string, stdin io.Reader, invalid invalidCommand) (string, any, *agentapi.Envelope) {
	storeID, revision, generation, sourceID := relationshipState(flags)
	var target, label, labelFile trackedString
	occurrence := flags.Int("occurrence", 1, "one-based matching raw occurrence")
	flags.Var(&target, "target-id", "exact raw target")
	flags.Var(&label, "label", "exact raw label")
	flags.Var(&labelFile, "label-file", "exact raw label file")
	if flags.Parse(args) != nil || flags.NArg() != 0 || !requireCLI(*storeID, *revision, *generation, *sourceID) || !target.set || *occurrence < 1 {
		return invalid("Relationship remove requires exact state, raw target/label, and occurrence.")
	}
	parsed, err := parseRequiredGeneration(*generation)
	if err != nil {
		return invalid("Generation must be a non-negative integer or none.")
	}
	text, present, err := exactText(label, labelFile, stdin)
	if err != nil || !present {
		return invalid("Use exactly one of --label or --label-file.")
	}
	return "relationship_remove", agentapi.RelationshipRemoveRequest{StatePreconditions: agentapi.StatePreconditions{StoreID: *storeID, AcceptedRevision: *revision, PendingGeneration: parsed}, SourceID: *sourceID, TargetID: target.value, Label: text, Occurrence: *occurrence}, nil
}

func parseDiagramCommand(operation string, flags *flag.FlagSet, args []string, invalid invalidCommand) (string, any, *agentapi.Envelope) {
	storeID, revision, generation := addStateFlags(flags)
	var componentID, diagramID string
	var title trackedString
	flags.StringVar(&componentID, "component-id", "", "stable Component ID")
	flags.StringVar(&diagramID, "diagram-id", "", "stable Diagram ID")
	flags.Var(&title, "title", "Diagram title")
	if flags.Parse(args) != nil || flags.NArg() != 0 || !requireCLI(*storeID, *revision, *generation) {
		return invalid("Diagram command requires exact state and its target fields.")
	}
	parsed, err := parseRequiredGeneration(*generation)
	if err != nil {
		return invalid("Generation must be a non-negative integer or none.")
	}
	state := agentapi.StatePreconditions{StoreID: *storeID, AcceptedRevision: *revision, PendingGeneration: parsed}
	switch operation {
	case "diagram_create_detail":
		if componentID == "" || !title.set {
			return invalid("Diagram create-detail requires --component-id and --title.")
		}
		return operation, agentapi.DiagramCreateDetailRequest{StatePreconditions: state, ComponentID: componentID, Title: title.value}, nil
	case "diagram_edit_title":
		if diagramID == "" || !title.set {
			return invalid("Diagram edit-title requires --diagram-id and --title.")
		}
		return operation, agentapi.DiagramEditTitleRequest{StatePreconditions: state, DiagramID: diagramID, Title: title.value}, nil
	default:
		if diagramID == "" || componentID == "" || title.set {
			return invalid("Show/stop-showing requires --diagram-id and --component-id.")
		}
		return operation, agentapi.DiagramComponentRequest{StatePreconditions: state, DiagramID: diagramID, ComponentID: componentID}, nil
	}
}

const cliHelp = `WorkBraid Architecture

Server mode: workbraid [--listen <loopback>] [--data-dir <dir>] [--ui-dir <dir>]
Client mode: workbraid [--server <loopback-url>] [--json] <command> [action flags]
MCP mode:    workbraid [--server <loopback-url>] mcp
Skill:       workbraid --skill

Read and project commands:
  status
  project list | project current
  project create --name <name>
  project open --slug <slug>
  project close --store-id <uuid>
  architecture inspect
  architecture refresh --store-id <uuid> --accepted-revision <sha>
  changes inspect

Every authoring command requires this exact inspected state:
  --store-id <uuid> --accepted-revision <sha> --generation <n|none>

Authoring commands:
  component create <state> --title <text> [--description <text>|--description-file <path|->] [--diagram-id <uuid>]
  component edit <state> --component-id <uuid> [--title <text>] [--description <text>|--description-file <path|->]
  component move-home <state> --component-id <uuid> --diagram-id <uuid>
  relationship add <state> --source-id <uuid> --target-id <raw> (--label <raw>|--label-file <path|->)
  relationship edit <state> --source-id <uuid> --old-target-id <raw> (--old-label <raw>|--old-label-file <path|->) [--occurrence <n>] --target-id <raw> (--label <raw>|--label-file <path|->)
  relationship remove <state> --source-id <uuid> --target-id <raw> (--label <raw>|--label-file <path|->) [--occurrence <n>]
  diagram create-detail <state> --component-id <uuid> --title <text>
  diagram edit-title <state> --diagram-id <uuid> --title <text>
  diagram show-component <state> --diagram-id <uuid> --component-id <uuid>
  diagram stop-showing-component <state> --diagram-id <uuid> --component-id <uuid>

Review and deliberate update:
  changes review <state with numeric generation>
  changes discard --store-id <uuid> --generation <n>
  architecture update --store-id <uuid> --base-revision <sha> --candidate-tree <tree> --generation <n>

The slug locates a project; store, Component, and Diagram UUIDs are stable identity.
Inspect after conflicts. Review returns the only base/tree/generation binding accepted
by Update. Run workbraid --skill for typed recovery and a complete JSON workflow.
`
