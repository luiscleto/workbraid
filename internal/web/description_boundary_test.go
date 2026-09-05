package web

import (
	"context"
	"strings"
	"testing"

	"workbraid/internal/agentapi"
)

func TestOrdinaryDescriptionEditPreservesUnterminatedHeading(t *testing.T) {
	for _, transport := range []string{"browser", "agent"} {
		t.Run(transport, func(t *testing.T) {
			fixture := newNativeRefreshFixture(t, false)
			source := "---\nid: \"" + fixture.component + "\"\n---\n# **Gateway**"
			blob := gitInput(t, []byte(source), "--git-dir", fixture.storePath, "hash-object", "-w", "--stdin")
			componentTree := gitInput(t, []byte("100644 blob "+blob+"\tgateway.md\n"), "--git-dir", fixture.storePath, "mktree")
			entries := strings.Split(git(t, "--git-dir", fixture.storePath, "ls-tree", fixture.base.Revision), "\n")
			for i, entry := range entries {
				if strings.HasSuffix(entry, "\tcomponents") {
					entries[i] = "040000 tree " + componentTree + "\tcomponents"
				}
			}
			tree := gitInput(t, []byte(strings.Join(entries, "\n")+"\n"), "--git-dir", fixture.storePath, "mktree")
			revision := gitInput(t, []byte("External heading spelling\n"), "--git-dir", fixture.storePath, "commit-tree", tree, "-p", fixture.base.Revision)
			git(t, "--git-dir", fixture.storePath, "update-ref", "refs/heads/accepted", revision, fixture.base.Revision)
			opened := decodeArchitectureResponse(t, postJSONRequest(t, fixture.handler, "/api/projects/open", map[string]any{"project_slug": fixture.base.ProjectSlug}))
			description := "\n  Exact body\r\n"
			id := ""
			if transport == "browser" {
				kept := decodeArchitectureResponse(t, postJSONRequest(t, fixture.handler, "/api/architecture/components/edit", observedComponentMutation(opened, componentMutationRequest{
					ComponentID: fixture.component, Title: "Gateway", TitleChanged: true, Description: description, DescriptionChanged: true,
				})))
				id = kept.ActionChangeSetID
			} else {
				state := changeSetState(t, createAgentChangeSet(t, fixture.handler, opened.StoreID, opened.Revision, "Document Gateway"))
				result := decodeAgentEnvelope(t, postAgent(t, fixture.handler, "/api/agent/v2/components/edit", agentapi.ComponentEditRequest{
					StatePreconditions: state, ComponentID: fixture.component, Title: stringPointer(" Gateway "), Description: &description,
				}))
				if !result.OK {
					t.Fatalf("ordinary edit: %+v", result)
				}
				id = state.ChangeSetID
			}
			pending := fixture.state.changeSets[id]
			if pending == nil || pending.candidate == nil {
				t.Fatal("ordinary edit has no valid proposal")
			}
			candidate := pending.candidate.Snapshot()
			component := candidate.AuthoringComponents()[0]
			markdown, _ := candidate.ComponentMarkdownSource(fixture.component)
			if component.Title != "Gateway" || component.Description != description || string(markdown) != "# **Gateway**\n"+description {
				t.Fatalf("ordinary edit lost source or values: %+v source=%q", component, markdown)
			}
			loaded, unavailable, err := fixture.state.architecture.LoadChangeSets(context.Background(), opened.StoreID)
			if err != nil || len(unavailable) != 0 || len(loaded) != 1 || loaded[0].Candidate.Tree() != pending.candidate.Tree() {
				t.Fatalf("durable edit: %v %v", err, unavailable)
			}
			if git(t, "--git-dir", fixture.storePath, "rev-parse", "refs/heads/accepted") != revision {
				t.Fatal("ordinary edit changed Accepted")
			}
		})
	}
}
