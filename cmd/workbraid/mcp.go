package main

import (
	"context"
	"io"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"workbraid/internal/agentapi"
)

type noToolInput struct{}

const mcpInstructions = "WorkBraid has one current Architecture project and one pending change set. Inspect exact IDs, accepted revision, and pending generation before editing. Review all changes, then Update only with the exact base_revision, candidate_tree, and generation returned by Review."

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func runMCP(serverURL string, stdin io.Reader, stdout, stderr io.Writer) int {
	client, failure := agentapi.NewClient(serverURL)
	if failure != nil {
		_, _ = io.WriteString(stderr, failure.Error.Message+"\n")
		return 1
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "workbraid", Version: "agent-access-1"}, &mcp.ServerOptions{
		Instructions: mcpInstructions,
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
	addMCPTool[noToolInput](server, client, "status", "Check WorkBraid", "Check the local agent protocol, current project, accepted revision, and pending generation. This never opens or refreshes a project.", readAnnotations("Check WorkBraid"))
	addMCPTool[noToolInput](server, client, "projects_list", "List projects", "List the project catalog with names, slugs, stable store UUIDs, revisions, conflicts, and unavailable entries. This does not change the current project.", readAnnotations("List projects"))
	addMCPTool[noToolInput](server, client, "project_current", "Inspect current project", "Return the current project or null. All MCP connections use the same current WorkBraid project.", readAnnotations("Inspect current project"))
	addMCPTool[agentapi.ProjectCreateRequest](server, client, "project_create", "Create project", "Create an Architecture project by human-readable name and make it current. Pending changes block creation; inspect or deliberately discard them first.", mutationAnnotations("Create project", false, false))
	addMCPTool[agentapi.ProjectOpenRequest](server, client, "project_open", "Open project", "Make one catalog project current by slug. The result returns its stable store UUID and exact loaded revision. Pending changes in another project block switching.", mutationAnnotations("Open project", false, false))
	addMCPTool[agentapi.ProjectCloseRequest](server, client, "project_close", "Close current project", "Close the current project identified by its exact store UUID. This never discards pending changes; pending changes block closing.", mutationAnnotations("Close current project", false, false))
	addMCPTool[noToolInput](server, client, "architecture_inspect", "Inspect accepted Architecture", "Return the accepted Architecture for the current project: revision, Diagram hierarchy and anchors, Component IDs and Markdown, homes and references, Relationships, appearances, and Relationship context. Use these IDs and state values for safe editing.", readAnnotations("Inspect accepted Architecture"))
	addMCPTool[agentapi.ArchitectureRefreshRequest](server, client, "architecture_refresh", "Refresh Architecture", "Check accepted Architecture again for the exact store and inspected revision. The result says whether WorkBraid adopted a revision, found no change, knows the loaded revision is no longer current, or could not determine the current revision.", mutationAnnotations("Refresh Architecture", false, false))
	addMCPTool[agentapi.ArchitectureUpdateRequest](server, client, "architecture_update", "Update Architecture", "Accept only the exact base_revision, candidate_tree, and generation returned by changes_review for the current store. There is no force, accept-latest, or retry when the outcome is uncertain.", mutationAnnotations("Update Architecture", true, false))
	addMCPTool[noToolInput](server, client, "changes_inspect", "Inspect changes", "Return all pending changes: exact base and generation, raw Relationship rows, Component and Diagram edits, the current preview when valid, localized problems, stale state, and current Review identity. This does not run Review.", readAnnotations("Inspect changes"))
	addMCPTool[agentapi.ChangesReviewRequest](server, client, "changes_review", "Review changes", "Review the exact pending generation and return the exact base_revision, candidate_tree, and generation required by architecture_update, plus the complete diff, Before/With projections, and structured comparison.", mutationAnnotations("Review changes", false, true))
	addMCPTool[agentapi.ChangesDiscardRequest](server, client, "changes_discard", "Discard all changes", "Clear all pending changes at the exact generation, including its Review, without changing accepted Architecture. Inspect first; partial discard and undo do not exist.", mutationAnnotations("Discard all changes", true, false))
	addMCPTool[agentapi.ComponentCreateRequest](server, client, "component_create", "Create Component", "Create a pending Component under exact store/revision/generation preconditions. Pass an explicit home Diagram ID when context is known; null deliberately uses root. Returns the stable generated ID and new generation.", mutationAnnotations("Create Component", false, false))
	addMCPTool[agentapi.ComponentEditRequest](server, client, "component_edit", "Edit Component", "Edit structured Title and/or exact Markdown Description for a stable Component ID under exact state preconditions. Omit unchanged fields; the returned generation replaces the inspected one.", mutationAnnotations("Edit Component", false, false))
	addMCPTool[agentapi.ComponentMoveHomeRequest](server, client, "component_move_home", "Move Component home", "Move one stable Component home to an allowed stable Diagram in the accepted and pending Architecture. It preserves identity, documentation, Relationships, and any anchored detail subtree.", mutationAnnotations("Move Component home", false, false))
	addMCPTool[agentapi.RelationshipAddRequest](server, client, "relationship_add", "Add Relationship", "Append one exact raw outgoing Relationship row to the stable source Component under exact state preconditions. Invalid raw target or label text remains inspectable and repairable, but Review is unavailable until it is corrected.", mutationAnnotations("Add Relationship", false, false))
	addMCPTool[agentapi.RelationshipEditRequest](server, client, "relationship_edit", "Edit Relationship", "Replace one pending authored Relationship selected by source ID, exact raw old target, exact raw old label, and one-based identical-pair occurrence. Empty/malformed old selectors are valid; expected generation prevents applying after another edit.", mutationAnnotations("Edit Relationship", false, false))
	addMCPTool[agentapi.RelationshipRemoveRequest](server, client, "relationship_remove", "Remove Relationship", "Remove one pending authored Relationship selected by source ID, exact raw target, exact raw label, and one-based identical-pair occurrence. This selector is request-local and creates no Relationship identity.", mutationAnnotations("Remove Relationship", true, false))
	addMCPTool[agentapi.DiagramCreateDetailRequest](server, client, "diagram_create_detail", "Create detail Diagram", "Create one pending detail Diagram anchored by an allowed home Component in the accepted and pending Architecture. Returns the stable generated Diagram ID and new generation.", mutationAnnotations("Create detail Diagram", false, false))
	addMCPTool[agentapi.DiagramEditTitleRequest](server, client, "diagram_edit_title", "Edit Diagram title", "Edit the authored title of a stable accepted or pending Diagram under exact state preconditions. Identity, filename, hierarchy, and composition remain unchanged.", mutationAnnotations("Edit Diagram title", false, false))
	addMCPTool[agentapi.DiagramComponentRequest](server, client, "diagram_show_component", "Show Component here", "Add one reusable reference appearance for a Component whose home is another Diagram, using the accepted and pending Architecture. No Component or Relationship is copied.", mutationAnnotations("Show Component here", false, false))
	addMCPTool[agentapi.DiagramComponentRequest](server, client, "diagram_stop_showing_component", "Stop showing Component here", "Remove an allowed reference appearance from the target Diagram. The Component home, documentation, and Relationships stay unchanged; Relationships may still make the Component appear automatically.", mutationAnnotations("Stop showing Component here", true, false))
}
