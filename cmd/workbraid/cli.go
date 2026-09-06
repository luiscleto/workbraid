package main

import (
	"bytes"
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
	operation := strings.ReplaceAll(group, "-", "_") + "_" + strings.ReplaceAll(action, "-", "_")
	if group == "project" && action == "list" {
		return noArgumentCommand("projects_list", actionArgs)
	}
	if group == "project" && action == "current" {
		return noArgumentCommand(operation, actionArgs)
	}
	if group == "architecture" && action == "inspect" {
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
		var storeID, changeSetID, base, tree, generation string
		flags.StringVar(&storeID, "store-id", "", "exact store UUID")
		flags.StringVar(&changeSetID, "change-set-id", "", "exact reviewed change-set UUID")
		flags.StringVar(&base, "base-revision", "", "reviewed base commit")
		flags.StringVar(&tree, "candidate-tree", "", "reviewed candidate tree")
		flags.StringVar(&generation, "generation", "", "reviewed pending generation")
		if flags.Parse(actionArgs) != nil || flags.NArg() != 0 || !requireCLI(storeID, changeSetID, base, tree, generation) {
			return invalid("Architecture update requires the exact --store-id, --change-set-id, --base-revision, --candidate-tree, and --generation review binding.")
		}
		parsed, err := parseRequiredGeneration(generation)
		if err != nil || parsed == nil {
			return invalid("Architecture update generation must be a non-negative integer.")
		}
		return operation, agentapi.ArchitectureUpdateRequest{StoreID: storeID, ChangeSetID: changeSetID, BaseRevision: base, CandidateTree: tree, Generation: *parsed}, nil
	case "change_set_list":
		var storeID string
		flags.StringVar(&storeID, "store-id", "", "exact store UUID")
		if flags.Parse(actionArgs) != nil || flags.NArg() != 0 || storeID == "" {
			return invalid("Change-set list requires --store-id.")
		}
		return "change_sets_list", agentapi.ChangeSetsListRequest{StoreID: storeID}, nil
	case "change_set_create":
		var storeID, revision string
		var name trackedString
		flags.StringVar(&storeID, "store-id", "", "exact store UUID")
		flags.StringVar(&revision, "accepted-revision", "", "exact current Accepted revision")
		flags.Var(&name, "name", "optional change-set name")
		if flags.Parse(actionArgs) != nil || flags.NArg() != 0 || !requireCLI(storeID, revision) {
			return invalid("Change-set create requires --store-id and --accepted-revision.")
		}
		var requested *string
		if name.set {
			requested = &name.value
		}
		return operation, agentapi.ChangeSetCreateRequest{StoreID: storeID, AcceptedRevision: revision, Name: requested}, nil
	case "change_set_inspect":
		var storeID, changeSetID string
		flags.StringVar(&storeID, "store-id", "", "exact store UUID")
		flags.StringVar(&changeSetID, "change-set-id", "", "exact change-set UUID")
		if flags.Parse(actionArgs) != nil || flags.NArg() != 0 || !requireCLI(storeID, changeSetID) {
			return invalid("Change-set inspect requires --store-id and --change-set-id.")
		}
		return operation, agentapi.ChangeSetInspectRequest{StoreID: storeID, ChangeSetID: changeSetID}, nil
	case "change_set_review":
		state, ok := parseStateFlags(flags, actionArgs)
		if !ok {
			return invalid("Change-set review requires exact store, change-set ID, and generation.")
		}
		return operation, agentapi.ChangeSetReviewRequest{StatePreconditions: state}, nil
	case "change_set_discard":
		state, ok := parseStateFlags(flags, actionArgs)
		if !ok {
			return invalid("Change-set discard requires exact store, change-set ID, and generation.")
		}
		return operation, agentapi.ChangeSetDiscardRequest{StatePreconditions: state}, nil
	case "change_set_edit_proposal", "change_set_rename":
		return parseChangeSetText(operation, flags, actionArgs, stdin, invalid)
	case "review_submission_list":
		var storeID, changeSetID string
		flags.StringVar(&storeID, "store-id", "", "exact store UUID")
		flags.StringVar(&changeSetID, "change-set-id", "", "exact Change Set UUID")
		if flags.Parse(actionArgs) != nil || flags.NArg() != 0 || !requireCLI(storeID, changeSetID) {
			return invalid("Review-submission list requires --store-id and --change-set-id.")
		}
		return "review_submissions_list", agentapi.ReviewSubmissionsListRequest{StoreID: storeID, ChangeSetID: changeSetID}, nil
	case "review_submission_inspect":
		var storeID, changeSetID, reviewID string
		flags.StringVar(&storeID, "store-id", "", "exact store UUID")
		flags.StringVar(&changeSetID, "change-set-id", "", "exact Change Set UUID")
		flags.StringVar(&reviewID, "review-id", "", "exact review-submission UUID")
		if flags.Parse(actionArgs) != nil || flags.NArg() != 0 || !requireCLI(storeID, changeSetID, reviewID) {
			return invalid("Review-submission inspect requires --store-id, --change-set-id, and --review-id.")
		}
		return "review_submission_inspect", agentapi.ReviewSubmissionInspectRequest{StoreID: storeID, ChangeSetID: changeSetID, ReviewID: reviewID}, nil
	case "review_submission_submit":
		return parseReviewSubmission(flags, actionArgs, stdin, invalid)
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
	case "diagram_positions", "diagram_set_position", "diagram_reset_position", "diagram_reset_layout":
		return parsePlacementCommand(operation, flags, actionArgs, invalid)
	case "diagram_parent_options", "diagram_reassign_detail":
		return parseDetailParentCommand(operation, flags, actionArgs, invalid)
	case "change_set_reconcile_preview", "change_set_reconcile_apply":
		return parseReconciliationCommand(operation, flags, actionArgs, stdin, invalid)
	case "diagram_create_detail", "diagram_edit_title", "diagram_show_component", "diagram_stop_showing_component":
		return parseDiagramCommand(operation, flags, actionArgs, invalid)
	default:
		return invalid("That WorkBraid command is not available. Use --help to list commands.")
	}
}

func parseReviewSubmission(flags *flag.FlagSet, args []string, stdin io.Reader, invalid invalidCommand) (string, any, *agentapi.Envelope) {
	var storeID, changeSetID, reviewedState, base, tree, generation, verdict, author, commentsFile string
	var body, bodyFile trackedString
	flags.StringVar(&storeID, "store-id", "", "exact store UUID")
	flags.StringVar(&changeSetID, "change-set-id", "", "exact Change Set UUID")
	flags.StringVar(&reviewedState, "reviewed-state", "", "exact reviewed Change Set state commit")
	flags.StringVar(&base, "base-revision", "", "exact reviewed base commit")
	flags.StringVar(&tree, "candidate-tree", "", "exact reviewed candidate tree")
	flags.StringVar(&generation, "generation", "", "exact reviewed generation")
	flags.StringVar(&verdict, "verdict", "", "comment, approve, or request_changes")
	flags.StringVar(&author, "author", "", "descriptive reviewer label")
	flags.Var(&body, "body", "optional exact overall Markdown")
	flags.Var(&bodyFile, "body-file", "optional exact overall Markdown file or -")
	flags.StringVar(&commentsFile, "comments-file", "", "optional JSON array of review comments")
	if flags.Parse(args) != nil || flags.NArg() != 0 || !requireCLI(storeID, changeSetID, reviewedState, base, tree, generation, verdict, author) || (body.set && bodyFile.set) {
		return invalid("Review-submission submit requires exact reviewed state/binding, --verdict, and --author; use at most one body input.")
	}
	parsed, err := parseRequiredGeneration(generation)
	if err != nil || parsed == nil {
		return invalid("Review generation must be a non-negative integer.")
	}
	overall := ""
	if body.set || bodyFile.set {
		overall, _, err = exactText(body, bodyFile, stdin)
		if err != nil {
			return invalid("The overall review Markdown could not be read as exact UTF-8.")
		}
	}
	comments := []agentapi.ReviewCommentInput{}
	if commentsFile != "" {
		if commentsFile == "-" && bodyFile.set && bodyFile.value == "-" {
			return invalid("Only one review input may read from standard input.")
		}
		var contents []byte
		if commentsFile == "-" {
			contents, err = io.ReadAll(io.LimitReader(stdin, 4<<20))
		} else {
			contents, err = os.ReadFile(commentsFile)
		}
		decoder := json.NewDecoder(bytes.NewReader(contents))
		decoder.DisallowUnknownFields()
		decodeErr := decoder.Decode(&comments)
		if decodeErr == nil {
			var extra any
			if nextErr := decoder.Decode(&extra); nextErr != io.EOF {
				decodeErr = errors.New("comments JSON contains another value")
			}
		}
		if err != nil || !utf8.Valid(contents) || decodeErr != nil {
			return invalid("--comments-file must contain one valid JSON array of typed review comments.")
		}
	}
	return "review_submission_submit", agentapi.ReviewSubmissionSubmitRequest{StoreID: storeID, ChangeSetID: changeSetID, ReviewedState: reviewedState,
		BaseRevision: base, CandidateTree: tree, Generation: *parsed, Verdict: verdict, Author: author, Body: overall, Comments: comments}, nil
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
	return flags.String("store-id", "", "exact store UUID"), flags.String("change-set-id", "", "exact active change-set UUID"), flags.String("generation", "", "exact inspected change-set generation")
}

func parseStateFlags(flags *flag.FlagSet, args []string) (agentapi.StatePreconditions, bool) {
	storeID, changeSetID, generation := addStateFlags(flags)
	if flags.Parse(args) != nil || flags.NArg() != 0 || !requireCLI(*storeID, *changeSetID, *generation) {
		return agentapi.StatePreconditions{}, false
	}
	parsed, err := parseRequiredGeneration(*generation)
	if err != nil || parsed == nil {
		return agentapi.StatePreconditions{}, false
	}
	return agentapi.StatePreconditions{StoreID: *storeID, ChangeSetID: *changeSetID, Generation: *parsed}, true
}

func parseRequiredGeneration(value string) (*uint64, error) {
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
		if !utf8.ValidString(literal.value) {
			return "", true, fmt.Errorf("input is not UTF-8")
		}
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

func parseChangeSetText(operation string, flags *flag.FlagSet, args []string, stdin io.Reader, invalid invalidCommand) (string, any, *agentapi.Envelope) {
	storeID, changeSetID, generation := addStateFlags(flags)
	var name, proposal, proposalFile trackedString
	if operation == "change_set_rename" {
		flags.Var(&name, "name", "new active change-set name")
	} else {
		flags.Var(&proposal, "proposal", "exact proposal Markdown")
		flags.Var(&proposalFile, "proposal-file", "exact proposal Markdown file or - for stdin")
	}
	if flags.Parse(args) != nil || flags.NArg() != 0 || !requireCLI(*storeID, *changeSetID, *generation) {
		return invalid("Change-set edit requires exact store, change-set ID, and generation.")
	}
	parsed, err := parseRequiredGeneration(*generation)
	if err != nil || parsed == nil {
		return invalid("Generation must be a non-negative integer.")
	}
	state := agentapi.StatePreconditions{StoreID: *storeID, ChangeSetID: *changeSetID, Generation: *parsed}
	if operation == "change_set_rename" {
		if !name.set {
			return invalid("Change-set rename requires --name.")
		}
		return operation, agentapi.ChangeSetRenameRequest{StatePreconditions: state, Name: name.value}, nil
	}
	text, present, readErr := exactText(proposal, proposalFile, stdin)
	if readErr != nil || !present {
		return invalid("Use exactly one of --proposal or --proposal-file.")
	}
	return operation, agentapi.ChangeSetEditProposalRequest{StatePreconditions: state, ProposalMarkdown: text}, nil
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
	if err != nil || parsed == nil {
		return invalid("Generation must be a non-negative integer.")
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
	return "component_create", agentapi.ComponentCreateRequest{StatePreconditions: agentapi.StatePreconditions{StoreID: *storeID, ChangeSetID: *revision, Generation: *parsed}, Title: title.value, Description: text, DiagramID: diagram}, nil
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
	if err != nil || parsed == nil {
		return invalid("Generation must be a non-negative integer.")
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
	return "component_edit", agentapi.ComponentEditRequest{StatePreconditions: agentapi.StatePreconditions{StoreID: *storeID, ChangeSetID: *revision, Generation: *parsed}, ComponentID: componentID, Title: titleValue, Description: descriptionValue}, nil
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
	if err != nil || parsed == nil {
		return invalid("Generation must be a non-negative integer.")
	}
	return "component_move_home", agentapi.ComponentMoveHomeRequest{StatePreconditions: agentapi.StatePreconditions{StoreID: *storeID, ChangeSetID: *revision, Generation: *parsed}, ComponentID: componentID, DiagramID: diagramID}, nil
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
	if err != nil || parsed == nil {
		return invalid("Generation must be a non-negative integer.")
	}
	text, present, err := exactText(label, labelFile, stdin)
	if err != nil || !present {
		return invalid("Use exactly one of --label or --label-file.")
	}
	return "relationship_add", agentapi.RelationshipAddRequest{StatePreconditions: agentapi.StatePreconditions{StoreID: *storeID, ChangeSetID: *revision, Generation: *parsed}, SourceID: *sourceID, TargetID: targetID.value, Label: text}, nil
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
	if err != nil || parsed == nil {
		return invalid("Generation must be a non-negative integer.")
	}
	oldText, oldPresent, oldErr := exactText(oldLabel, oldLabelFile, stdin)
	newText, newPresent, newErr := exactText(label, labelFile, stdin)
	if oldErr != nil || newErr != nil || !oldPresent || !newPresent {
		return invalid("Use exactly one old-label input and one new label input.")
	}
	return "relationship_edit", agentapi.RelationshipEditRequest{StatePreconditions: agentapi.StatePreconditions{StoreID: *storeID, ChangeSetID: *revision, Generation: *parsed}, SourceID: *sourceID, OldTargetID: oldTarget.value, OldLabel: oldText, Occurrence: *occurrence, TargetID: target.value, Label: newText}, nil
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
	if err != nil || parsed == nil {
		return invalid("Generation must be a non-negative integer.")
	}
	text, present, err := exactText(label, labelFile, stdin)
	if err != nil || !present {
		return invalid("Use exactly one of --label or --label-file.")
	}
	return "relationship_remove", agentapi.RelationshipRemoveRequest{StatePreconditions: agentapi.StatePreconditions{StoreID: *storeID, ChangeSetID: *revision, Generation: *parsed}, SourceID: *sourceID, TargetID: target.value, Label: text, Occurrence: *occurrence}, nil
}

func parseDetailParentCommand(operation string, flags *flag.FlagSet, args []string, invalid invalidCommand) (string, any, *agentapi.Envelope) {
	storeID, changeSetID, generation := addStateFlags(flags)
	var diagramID, anchorID string
	flags.StringVar(&diagramID, "diagram-id", "", "non-root child Diagram UUID")
	if operation == "diagram_reassign_detail" {
		flags.StringVar(&anchorID, "anchor-component-id", "", "eligible destination Component UUID")
	}
	if flags.Parse(args) != nil || flags.NArg() != 0 || !requireCLI(*storeID, *changeSetID, *generation, diagramID) {
		return invalid("Parent commands require exact state and --diagram-id.")
	}
	parsed, err := parseRequiredGeneration(*generation)
	if err != nil || parsed == nil {
		return invalid("Generation must be a non-negative integer.")
	}
	state := agentapi.StatePreconditions{StoreID: *storeID, ChangeSetID: *changeSetID, Generation: *parsed}
	if operation == "diagram_parent_options" {
		return operation, agentapi.DiagramParentOptionsRequest{StatePreconditions: state, DiagramID: diagramID}, nil
	}
	if anchorID == "" {
		return invalid("Reassign-detail requires --anchor-component-id from parent-options.")
	}
	return operation, agentapi.DiagramReassignDetailRequest{StatePreconditions: state, DiagramID: diagramID, AnchorComponentID: anchorID}, nil
}

func parsePlacementCommand(operation string, flags *flag.FlagSet, args []string, invalid invalidCommand) (string, any, *agentapi.Envelope) {
	store := flags.String("store-id", "", "exact store UUID")
	proposal := flags.String("change-set-id", "", "proposal UUID; omit only for Accepted positions read")
	diagram := flags.String("diagram-id", "", "Diagram UUID")
	if operation == "diagram_positions" {
		if flags.Parse(args) != nil || flags.NArg() != 0 || !requireCLI(*store, *diagram) {
			return invalid("Positions requires --store-id and --diagram-id.")
		}
		return operation, agentapi.DiagramPositionsRequest{StoreID: *store, ChangeSetID: *proposal, DiagramID: *diagram}, nil
	}
	generation := flags.String("generation", "", "exact proposal generation")
	component := ""
	if operation != "diagram_reset_layout" {
		flags.StringVar(&component, "component-id", "", "canonical Component UUID in this Diagram")
	}
	var x, y trackedString
	if operation == "diagram_set_position" {
		flags.Var(&x, "x", "integer center X, -100000..100000")
		flags.Var(&y, "y", "integer center Y, -100000..100000")
	}
	if flags.Parse(args) != nil || flags.NArg() != 0 || !requireCLI(*store, *proposal, *diagram, *generation) {
		return invalid("Placement requires exact store, proposal, generation and Diagram.")
	}
	g, err := parseRequiredGeneration(*generation)
	if err != nil || g == nil {
		return invalid("Generation must be a non-negative integer.")
	}
	reset := agentapi.DiagramResetLayoutRequest{StatePreconditions: agentapi.StatePreconditions{StoreID: *store, ChangeSetID: *proposal, Generation: *g}, DiagramID: *diagram}
	if operation == "diagram_reset_layout" {
		return operation, reset, nil
	}
	if component == "" {
		return invalid("Placement requires --component-id.")
	}
	one := agentapi.DiagramResetPositionRequest{DiagramResetLayoutRequest: reset, ComponentID: component}
	if operation == "diagram_reset_position" {
		return operation, one, nil
	}
	xv, xe := strconv.Atoi(x.value)
	yv, ye := strconv.Atoi(y.value)
	if !x.set || !y.set || xe != nil || ye != nil || xv < -100000 || xv > 100000 || yv < -100000 || yv > 100000 {
		return invalid("X and Y must be integers from -100000 to 100000. Use --x=-180 for negative values.")
	}
	return operation, agentapi.DiagramSetPositionRequest{DiagramResetPositionRequest: one, X: xv, Y: yv}, nil
}

func parseReconciliationCommand(operation string, flags *flag.FlagSet, args []string, stdin io.Reader, invalid invalidCommand) (string, any, *agentapi.Envelope) {
	storeID, changeSetID, generation := addStateFlags(flags)
	var inputs agentapi.ReconciliationInputs
	var filename string
	flags.StringVar(&inputs.ChangeSetState, "change-set-state", "", "exact active state S from inspect")
	flags.StringVar(&inputs.BaseRevision, "base-revision", "", "exact proposal base B")
	flags.StringVar(&inputs.CandidateTree, "candidate-tree", "", "exact valid proposal tree P")
	flags.StringVar(&inputs.AcceptedRevision, "accepted-revision", "", "exact known-current Accepted A")
	flags.StringVar(&filename, "resolutions-file", "", "exact typed resolutions JSON file, or - for stdin; Apply requires [] for automatic work")
	if flags.Parse(args) != nil || flags.NArg() != 0 || !requireCLI(*storeID, *changeSetID, *generation, inputs.ChangeSetState, inputs.BaseRevision, inputs.CandidateTree, inputs.AcceptedRevision) {
		return invalid("Reconciliation requires exact store, Change Set, generation and S/B/A/P inputs from inspect.")
	}
	parsed, err := parseRequiredGeneration(*generation)
	if err != nil || parsed == nil {
		return invalid("Generation must be a non-negative integer.")
	}
	inputs.StatePreconditions = agentapi.StatePreconditions{StoreID: *storeID, ChangeSetID: *changeSetID, Generation: *parsed}
	request := agentapi.ReconciliationApplyRequest{ReconciliationInputs: inputs}
	if filename != "" {
		var data []byte
		if filename == "-" {
			data, err = io.ReadAll(stdin)
		} else {
			data, err = os.ReadFile(filename)
		}
		if err != nil {
			return invalid("The resolutions file could not be read.")
		}
		if err = json.Unmarshal(data, &request.Resolutions); err != nil || request.Resolutions == nil {
			return invalid("Resolutions must be a closed typed JSON array, with [] for automatic work.")
		}
	}
	if operation == "change_set_reconcile_preview" {
		return operation, agentapi.ReconciliationPreviewRequest{ReconciliationInputs: inputs, Resolutions: request.Resolutions}, nil
	}
	if filename == "" {
		return invalid("Apply requires --resolutions-file, with [] for a fully automatic result.")
	}
	return operation, request, nil
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
	if err != nil || parsed == nil {
		return invalid("Generation must be a non-negative integer.")
	}
	state := agentapi.StatePreconditions{StoreID: *storeID, ChangeSetID: *revision, Generation: *parsed}
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

Client commands always print one JSON envelope; --json makes it compact.
--server or WORKBRAID_SERVER selects the running WorkBraid URL.
The default is http://127.0.0.1:8080.

Connect, choose a project, and inspect:
  status
  project list | project current
  project create --name <name>
  project open --slug <slug>
  project close --store-id <uuid>
  architecture inspect
  architecture refresh --store-id <uuid> --accepted-revision <sha>

Durable change sets:
  change-set list --store-id <uuid>
  change-set create --store-id <uuid> --accepted-revision <sha> [--name <text>]
  change-set inspect --store-id <uuid> --change-set-id <uuid>
  change-set rename <state> --name <text>
  change-set edit-proposal <state> (--proposal <markdown>|--proposal-file <path|->)

Durable review feedback:
  review-submission list --store-id <uuid> --change-set-id <uuid>
  review-submission inspect --store-id <uuid> --change-set-id <uuid> --review-id <uuid>
  review-submission submit --store-id <uuid> --change-set-id <uuid> --reviewed-state <commit> --base-revision <sha> --candidate-tree <tree> --generation <n> --verdict <comment|approve|request_changes> --author <label> [--body <markdown>|--body-file <path|->] [--comments-file <json>]

Every authoring command requires this exact inspected state:
  --store-id <uuid> --change-set-id <uuid> --generation <n>

Authoring commands:
  component create <state> --title <text> [--description <text>|--description-file <path|->] [--diagram-id <uuid>]
  component edit <state> --component-id <uuid> [--title <text>] [--description <text>|--description-file <path|->]
  component move-home <state> --component-id <uuid> --diagram-id <uuid>
  relationship add <state> --source-id <uuid> --target-id <raw> (--label <raw>|--label-file <path|->)
  relationship edit <state> --source-id <uuid> --old-target-id <raw> (--old-label <raw>|--old-label-file <path|->) [--occurrence <n>] --target-id <raw> (--label <raw>|--label-file <path|->)
  relationship remove <state> --source-id <uuid> --target-id <raw> (--label <raw>|--label-file <path|->) [--occurrence <n>]
  diagram parent-options <state> --diagram-id <uuid>
  diagram reassign-detail <state> --diagram-id <uuid> --anchor-component-id <uuid>
  diagram positions --store-id <uuid> --diagram-id <uuid> [--change-set-id <uuid>]
  diagram set-position <state> --diagram-id <uuid> --component-id <uuid> --x=-180 --y=320
  diagram reset-position <state> --diagram-id <uuid> --component-id <uuid>
  diagram reset-layout <state> --diagram-id <uuid>
  change-set reconcile-preview <state> --change-set-state <S> --base-revision <B> --candidate-tree <P> --accepted-revision <A> [--resolutions-file <path|->]
  change-set reconcile-apply <state> --change-set-state <S> --base-revision <B> --candidate-tree <P> --accepted-revision <A> --resolutions-file <path|->
  diagram create-detail <state> --component-id <uuid> --title <text>
  diagram edit-title <state> --diagram-id <uuid> --title <text>
  diagram show-component <state> --diagram-id <uuid> --component-id <uuid>
  diagram stop-showing-component <state> --diagram-id <uuid> --component-id <uuid>

Review and deliberate update:
  change-set review <state>
  change-set discard <state>
  architecture update --store-id <uuid> --change-set-id <uuid> --base-revision <sha> --candidate-tree <tree> --generation <n>

The slug locates a project. Store, change-set, Component, and Diagram UUIDs are stable
identity. Each change set has its own generation. Inspect after conflicts. Review returns
the only ID/base/tree/generation binding accepted by Update and a review_url to give the
reviewer. Review submissions are immutable informational feedback and never accept
Architecture. Run workbraid --skill for typed recovery, out-of-date rules, and a complete
JSON workflow.
`
