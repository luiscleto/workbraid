package main

import (
	"context"
	"io"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"workbraid/internal/agentapi"
)

type noToolInput struct{}

const mcpInstructions = "WorkBraid has one Accepted Architecture and durable named change sets. List or create change sets, then address every edit by exact change_set_id and generation. Review that same change set, give its returned review_url to the reviewer, and Update only with its exact returned binding."

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func runMCP(serverURL string, stdin io.Reader, stdout, stderr io.Writer) int {
	client, failure := agentapi.NewClient(serverURL)
	if failure != nil {
		_, _ = io.WriteString(stderr, failure.Error.Message+"\n")
		return 1
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "workbraid", Version: "agent-access-2"}, &mcp.ServerOptions{
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
	addMCPTool[noToolInput](server, client, "status", "Check WorkBraid", "Check the local agent protocol, current project, Accepted revision, and authority state. This never opens or refreshes a project.", readAnnotations("Check WorkBraid"))
	addMCPTool[noToolInput](server, client, "projects_list", "List projects", "List the project catalog with names, slugs, stable store UUIDs, revisions, conflicts, and unavailable entries. This does not change the current project.", readAnnotations("List projects"))
	addMCPTool[noToolInput](server, client, "project_current", "Inspect current project", "Return the current project or null. All MCP connections use the same current WorkBraid project.", readAnnotations("Inspect current project"))
	addMCPTool[agentapi.ProjectCreateRequest](server, client, "project_create", "Create project", "Create an Architecture project by human-readable name and make it current. Durable change sets in another project are retained.", mutationAnnotations("Create project", false, false))
	addMCPTool[agentapi.ProjectOpenRequest](server, client, "project_open", "Open project", "Make one catalog project current by slug. Durable change sets remain in their owning projects.", mutationAnnotations("Open project", false, false))
	addMCPTool[agentapi.ProjectCloseRequest](server, client, "project_close", "Close current project", "Close the current project identified by its exact store UUID without deleting its durable change sets.", mutationAnnotations("Close current project", false, false))
	addMCPTool[noToolInput](server, client, "architecture_inspect", "Inspect accepted Architecture", "Return the accepted Architecture for the current project: revision, Diagram hierarchy and anchors, Component IDs and Markdown, homes and references, Relationships, appearances, and Relationship context. Use these IDs and state values for safe editing.", readAnnotations("Inspect accepted Architecture"))
	addMCPTool[agentapi.ArchitectureRefreshRequest](server, client, "architecture_refresh", "Refresh Architecture", "Check accepted Architecture again for the exact store and inspected revision. The result says whether WorkBraid adopted a revision, found no change, knows the loaded revision is no longer current, or could not determine the current revision.", mutationAnnotations("Refresh Architecture", false, false))
	addMCPTool[agentapi.ArchitectureUpdateRequest](server, client, "architecture_update", "Update Architecture", "Accept only the exact change_set_id, base_revision, candidate_tree, and generation returned by change_set_review. Out-of-date change sets cannot be accepted.", mutationAnnotations("Update Architecture", true, false))
	addMCPTool[agentapi.ChangeSetsListRequest](server, client, "change_sets_list", "List change sets", "List active, applied, and unavailable change sets for the exact current store UUID. No browser selection state is used.", readAnnotations("List change sets"))
	addMCPTool[agentapi.ChangeSetCreateRequest](server, client, "change_set_create", "Create change set", "Create one durable active change set from the exact current Accepted revision, optionally with a custom name. It begins at generation 0.", mutationAnnotations("Create change set", false, false))
	addMCPTool[agentapi.ChangeSetInspectRequest](server, client, "change_set_inspect", "Inspect change set", "Inspect one exact active or applied change set, including proposal Markdown, base, generation, concrete facts, validity, candidate, out-of-date state, Review, and applied revision.", readAnnotations("Inspect change set"))
	addMCPTool[agentapi.ChangeSetRenameRequest](server, client, "change_set_rename", "Rename change set", "Rename one active change set at its exact generation. Names select visually; the stable UUID remains identity.", mutationAnnotations("Rename change set", false, false))
	addMCPTool[agentapi.ChangeSetEditProposalRequest](server, client, "change_set_edit_proposal", "Edit proposal", "Replace the exact Markdown proposal document for one active change set at its exact generation. This invalidates only that change set's Review.", mutationAnnotations("Edit proposal", false, false))
	addMCPTool[agentapi.ChangeSetReviewRequest](server, client, "change_set_review", "Review change set", "Review one exact active generation against its original base and return the exact binding required by architecture_update plus a review_url to give the reviewer. Out-of-date work remains reviewable.", mutationAnnotations("Review change set", false, true))
	addMCPTool[agentapi.ChangeSetDiscardRequest](server, client, "change_set_discard", "Delete change set", "Delete one whole active change set at its exact generation without changing Accepted or any other record. Partial discard and applied deletion do not exist.", mutationAnnotations("Delete change set", true, false))
	addMCPTool[agentapi.ComponentCreateRequest](server, client, "component_create", "Create Component", "Create a Component in one explicitly addressed active change set. Pass its exact store UUID, change-set UUID, and generation, plus an explicit home Diagram when known.", mutationAnnotations("Create Component", false, false))
	addMCPTool[agentapi.ComponentEditRequest](server, client, "component_edit", "Edit Component", "Edit structured Title and/or exact Markdown Description for a stable Component ID under exact state preconditions. Omit unchanged fields; the returned generation replaces the inspected one.", mutationAnnotations("Edit Component", false, false))
	addMCPTool[agentapi.ComponentMoveHomeRequest](server, client, "component_move_home", "Move Component home", "Move one stable Component home to an allowed stable Diagram in the explicitly addressed proposal Architecture. It preserves identity, documentation, Relationships, and any anchored detail subtree.", mutationAnnotations("Move Component home", false, false))
	addMCPTool[agentapi.RelationshipAddRequest](server, client, "relationship_add", "Add Relationship", "Append one exact raw outgoing Relationship row to the stable source Component under exact state preconditions. Invalid raw target or label text remains inspectable and repairable, but Review is unavailable until it is corrected.", mutationAnnotations("Add Relationship", false, false))
	addMCPTool[agentapi.RelationshipEditRequest](server, client, "relationship_edit", "Edit Relationship", "Replace one authored Relationship in the addressed change set, selected by source ID, exact raw old target, exact raw old label, and one-based identical-pair occurrence. Empty or malformed old selectors remain repairable.", mutationAnnotations("Edit Relationship", false, false))
	addMCPTool[agentapi.RelationshipRemoveRequest](server, client, "relationship_remove", "Remove Relationship", "Remove one authored Relationship in the addressed change set, selected by source ID, exact raw target, exact raw label, and one-based identical-pair occurrence. This request-local selector creates no Relationship identity.", mutationAnnotations("Remove Relationship", true, false))
	addMCPTool[agentapi.DiagramCreateDetailRequest](server, client, "diagram_create_detail", "Create detail Diagram", "Create one detail Diagram in the addressed change set, anchored by an allowed home Component. Returns the stable generated Diagram ID and new generation.", mutationAnnotations("Create detail Diagram", false, false))
	addMCPTool[agentapi.DiagramEditTitleRequest](server, client, "diagram_edit_title", "Edit Diagram title", "Edit the authored title of a stable Diagram in the addressed change set under exact generation preconditions. Identity, filename, hierarchy, and composition remain unchanged.", mutationAnnotations("Edit Diagram title", false, false))
	addMCPTool[agentapi.DiagramComponentRequest](server, client, "diagram_show_component", "Show Component here", "Add one reusable reference appearance in the addressed change set for a Component whose home is another Diagram. No Component or Relationship is copied.", mutationAnnotations("Show Component here", false, false))
	addMCPTool[agentapi.DiagramComponentRequest](server, client, "diagram_stop_showing_component", "Stop showing Component here", "Remove an allowed reference appearance from the target Diagram. The Component home, documentation, and Relationships stay unchanged; Relationships may still make the Component appear automatically.", mutationAnnotations("Stop showing Component here", true, false))
}
