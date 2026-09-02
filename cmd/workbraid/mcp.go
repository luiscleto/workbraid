package main

import (
	"context"
	"io"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"workbraid/internal/agentapi"
)

type noToolInput struct{}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func runMCP(serverURL string, stdin io.Reader, stdout, stderr io.Writer) int {
	client, failure := agentapi.NewClient(serverURL)
	if failure != nil {
		_, _ = io.WriteString(stderr, failure.Error.Message+"\n")
		return 1
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "workbraid", Version: "agent-access-1"}, &mcp.ServerOptions{
		Instructions: "WorkBraid tools operate on one process-wide current Architecture project and one server-owned pending set. Inspect exact IDs, accepted revision, and pending generation before mutation. Review the complete candidate, then deliberately update only with the returned exact base/tree/generation binding.",
	})
	registerMCPTools(server, client)
	transport := &mcp.IOTransport{Reader: io.NopCloser(stdin), Writer: nopWriteCloser{stdout}}
	if err := server.Run(context.Background(), transport); err != nil {
		_, _ = io.WriteString(stderr, err.Error()+"\n")
		return 1
	}
	return 0
}

func boolPointer(value bool) *bool { return &value }

func readAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: true, IdempotentHint: true, OpenWorldHint: boolPointer(false), DestructiveHint: boolPointer(false)}
}

func mutationAnnotations(title string, destructive, idempotent bool) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: false, IdempotentHint: idempotent, OpenWorldHint: boolPointer(false), DestructiveHint: boolPointer(destructive)}
}

func addMCPTool[Input any](server *mcp.Server, client *agentapi.Client, name, title, description string, annotations *mcp.ToolAnnotations) {
	mcp.AddTool(server, &mcp.Tool{Name: name, Title: title, Description: description, Annotations: annotations},
		func(ctx context.Context, _ *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, agentapi.Envelope, error) {
			envelope := client.Call(ctx, name, input)
			return &mcp.CallToolResult{IsError: !envelope.OK}, envelope, nil
		})
}

func registerMCPTools(server *mcp.Server, client *agentapi.Client) {
	addMCPTool[noToolInput](server, client, "status", "Check WorkBraid", "Check the local agent protocol and process-wide current project, accepted authority state, and pending generation. This never opens or refreshes a project.", readAnnotations("Check WorkBraid"))
	addMCPTool[noToolInput](server, client, "projects_list", "List projects", "List the private-store-derived project catalog with names, slugs, immutable store UUIDs, revisions, conflicts, and unavailable entries. This does not change the current project.", readAnnotations("List projects"))
	addMCPTool[noToolInput](server, client, "project_current", "Inspect current project", "Return the one process-wide current project or null. Project selection belongs to the running WorkBraid process, not this MCP connection.", readAnnotations("Inspect current project"))
	addMCPTool[agentapi.ProjectCreateRequest](server, client, "project_create", "Create project", "Create a native Architecture project by human-readable name and select it process-wide. Pending work blocks creation; inspect or deliberately discard it first.", mutationAnnotations("Create project", false, false))
	addMCPTool[agentapi.ProjectOpenRequest](server, client, "project_open", "Open project", "Select one catalog project process-wide by slug. The result returns its immutable store UUID and exact loaded revision. Pending work in another project blocks switching.", mutationAnnotations("Open project", false, false))
	addMCPTool[agentapi.ProjectCloseRequest](server, client, "project_close", "Close current project", "Close the exact process-wide current store. This never discards pending work; pending work blocks closing.", mutationAnnotations("Close current project", false, false))
	addMCPTool[noToolInput](server, client, "architecture_inspect", "Inspect accepted Architecture", "Return the complete exact accepted projection for the current project: revision, Diagram hierarchy and anchors, Component IDs and Markdown, homes/references, Relationships, appearances, and boundary context. Use these IDs and state tokens for safe authoring.", readAnnotations("Inspect accepted Architecture"))
	addMCPTool[agentapi.ArchitectureRefreshRequest](server, client, "architecture_refresh", "Refresh Architecture", "Explicitly re-observe accepted authority for the exact store and inspected loaded revision. It reports adopted or unchanged state; known non-current and failed-to-determine results require different recovery.", mutationAnnotations("Refresh Architecture", false, false))
	addMCPTool[agentapi.ArchitectureUpdateRequest](server, client, "architecture_update", "Update Architecture", "Deliberately accept only the exact base commit, candidate tree, and pending generation returned by changes_review for the current store. There is no force, accept-latest, automatic rebuild, or retry-on-uncertainty behavior.", mutationAnnotations("Update Architecture", true, false))
	addMCPTool[noToolInput](server, client, "changes_inspect", "Inspect changes", "Return the complete server-owned pending set: exact base and generation, raw authored Relationship rows, all Component/Diagram/composition facts, candidate when valid, localized validation, stale state, and current review identity. This does not construct a review.", readAnnotations("Inspect changes"))
	addMCPTool[agentapi.ChangesReviewRequest](server, client, "changes_review", "Review changes", "Validate the exact current pending generation and return its immutable base/tree/generation binding, complete unified diff, bound Before/With projections, and structured comparison. Review before any architecture_update.", mutationAnnotations("Review changes", false, true))
	addMCPTool[agentapi.ChangesDiscardRequest](server, client, "changes_discard", "Discard all changes", "Destructively clear the whole exact pending generation and its review without changing accepted Git. Inspect first; partial discard and undo do not exist.", mutationAnnotations("Discard all changes", true, false))
	addMCPTool[agentapi.ComponentCreateRequest](server, client, "component_create", "Create Component", "Create a pending Component under exact store/revision/generation preconditions. Pass an explicit home Diagram ID when context is known; null deliberately uses root. Returns the stable generated ID and new generation.", mutationAnnotations("Create Component", false, false))
	addMCPTool[agentapi.ComponentEditRequest](server, client, "component_edit", "Edit Component", "Edit structured Title and/or exact Markdown Description for a stable Component ID under exact state preconditions. Omit unchanged fields; the returned generation replaces the inspected one.", mutationAnnotations("Edit Component", false, false))
	addMCPTool[agentapi.ComponentMoveHomeRequest](server, client, "component_move_home", "Move Component home", "Move one stable Component home to an eligible stable Diagram in the complete current candidate. It preserves identity, documentation, Relationships, and any anchored detail subtree.", mutationAnnotations("Move Component home", false, false))
	addMCPTool[agentapi.RelationshipAddRequest](server, client, "relationship_add", "Add Relationship", "Append one exact raw outgoing Relationship row to the stable source Component under exact state preconditions. Raw invalid target/label text remains inspectable and repairable while candidate validation is blocked.", mutationAnnotations("Add Relationship", false, false))
	addMCPTool[agentapi.RelationshipEditRequest](server, client, "relationship_edit", "Edit Relationship", "Replace one pending authored Relationship selected by source ID, exact raw old target, exact raw old label, and one-based identical-pair occurrence. Empty/malformed old selectors are valid; expected generation prevents applying after another edit.", mutationAnnotations("Edit Relationship", false, false))
	addMCPTool[agentapi.RelationshipRemoveRequest](server, client, "relationship_remove", "Remove Relationship", "Remove one pending authored Relationship selected by source ID, exact raw target, exact raw label, and one-based identical-pair occurrence. This selector is request-local and creates no Relationship identity.", mutationAnnotations("Remove Relationship", true, false))
	addMCPTool[agentapi.DiagramCreateDetailRequest](server, client, "diagram_create_detail", "Create detail Diagram", "Create one pending detail Diagram anchored by an eligible home Component in the complete current candidate. Returns the stable generated Diagram ID and new generation.", mutationAnnotations("Create detail Diagram", false, false))
	addMCPTool[agentapi.DiagramEditTitleRequest](server, client, "diagram_edit_title", "Edit Diagram title", "Edit the authored title of a stable accepted or pending Diagram under exact state preconditions. Identity, filename, hierarchy, and composition remain unchanged.", mutationAnnotations("Edit Diagram title", false, false))
	addMCPTool[agentapi.DiagramComponentRequest](server, client, "diagram_show_component", "Show Component here", "Add one reusable reference appearance for a Component whose home is another Diagram, resolved against the complete current candidate. No Component or Relationship fact is copied.", mutationAnnotations("Show Component here", false, false))
	addMCPTool[agentapi.DiagramComponentRequest](server, client, "diagram_stop_showing_component", "Stop showing Component here", "Remove only an eligible canonical reference appearance from the target Diagram. The Component home, documentation, and Relationships remain exact; a derived boundary may reappear.", mutationAnnotations("Stop showing Component here", true, false))
}
