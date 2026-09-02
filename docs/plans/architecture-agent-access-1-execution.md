# Architecture Agent Access 1 execution packet

Status: Approved

Target: CLI, `--skill`, MCP, and black-box agent usability over the completed Architecture product

Exact Phase 2 completion prerequisite: `8cd9ce145980fbc57377e729975940b9e2f0b8b6`

Exact Phase 2 final implementation: `39ca3f7035dc93f63dd32078506b6752ee271ec5`

Approved Agent Access baseline and roadmap: `4efbe773580dbe1f4ed1778a7211d50ba626892b`

Approved Agent Access product contract: `docs/architecture-agent-access-v0.md`

Roadmap placement: `docs/architecture-post-gate-roadmap.md`

Packet-inclusive worker base: the exact clean commit produced by committing this packet alone on `4efbe773580dbe1f4ed1778a7211d50ba626892b`; record it before dispatch

This is one cohesive interface increment. It creates no Architecture store-format change and no second authority. A genuine product/architecture gap stops implementation for human decision.

## Execution discipline

After human approval:

1. Verify approved Agent Access baseline `4efbe773580dbe1f4ed1778a7211d50ba626892b` is rooted at exact Phase 2 completion prerequisite `8cd9ce145980fbc57377e729975940b9e2f0b8b6`.
2. Commit this Approved packet alone on that exact baseline after `git diff --check`.
3. Record the resulting clean packet-inclusive commit as the one exact worker base.
4. Dispatch exactly one implementation worker from that base.
5. Send the complete worker-base-to-head range to one fresh independent reviewer who did not implement it.
6. Return bounded defects to the same worker and require fresh rereview of the corrected complete range.
7. Integrate only the exact approved range, verify the reviewed tree identity, and run ordinary checks once.
8. Run separate fresh weak-agent CLI and MCP black-box gates, then have the independent technical reviewer verify both canonical results.
9. Run the small real human UI checkpoint and record explicit PASS/FAIL in a separate completion commit.
10. Stop. Do not plan or implement rich-diagram Phase 3.

The implementation worker may make conventional commits that separate the concrete shared operation boundary, CLI/skill, and MCP adapter, but they remain one worker, one reviewed range, one integration, and one product gate.

## Exact worker brief

Before editing, verify the exact clean worker base and read completely:

- `AGENTS.md`;
- `docs/architecture-v0.md`;
- `docs/ui-v0.md`;
- `docs/architecture-post-gate-roadmap.md`;
- `docs/architecture-agent-access-v0.md`;
- this packet;
- completed `docs/plans/architecture-post-gate-phase-2.4-execution.md` for the current slug catalog, v2 Diagram/reference product, single pending authority, and final Phase 2 evidence;
- completed `docs/plans/architecture-post-gate-phase-2.3-execution.md` and `docs/plans/architecture-post-gate-phase-2.2-execution.md` only where the pending composition and candidate writer details remain relevant;
- completed `docs/plans/architecture-post-gate-phase-1-execution.md` for immutable review projection/capture, exact binding, diff, and CAS behavior.

Do not edit approved baselines, roadmap, historical completion records, or this packet. If implementation needs a new domain lifecycle, store field, approval rule, client-specific pending model, remote-security model, or different acceptance path, stop and report it.

## Worker implementation scope

### 1. Preserve one process authority

- Keep one authoritative long-running `workbraid` server process and one `architecture.Manager` instance beneath it.
- Keep the current loaded project/snapshot, current/non-current state, pending set, base, generation, candidate, review binding, and publication state protected by the existing concrete synchronization boundary.
- Add the smallest concrete application-operation methods needed so browser handlers and agent handlers invoke the same state transitions and classifications. Do not leave duplicated browser versus agent implementations which can drift.
- Do not introduce a command bus, generic operation registry, event system, RPC framework, repository interface, service container, second handler instance, or separate domain package merely to name transport calls.
- Capture immutable response data under the existing lock where authority can be invalidated, then serialize after release where safe. Do not hold references to mutable pending/review state in transport responses.
- Every agent mutation checks exact server-owned store UUID, loaded accepted revision, and expected pending generation (`null` for no pending) before changing state. Project switching/creation/close, Refresh, discard, mutation, review, and acceptance remain atomic with each other.
- Refactor the browser requests proportionately to pass/use the same expected state where needed; do not weaken browser authority merely because Agent Access needs typed preconditions.

### 2. Local agent HTTP surface and client

- Add one small versioned JSON agent surface to the existing loopback HTTP mux. It is an internal local transport for the shipped CLI/MCP clients, not a public remote API.
- Add a non-mutating status/handshake returning `workbraid-agent-v1`, current project if any, accepted revision/authority state, and pending generation.
- Keep browser same-origin protection. Require JSON request bodies, no CORS, literal-loopback use, and reject unexpected browser origins on the agent surface without inventing authentication.
- Add one concrete Go client used by both CLI commands and the MCP bridge. It validates an explicit literal-loopback `http` server URL, applies bounded request timeouts, decodes the common envelope, and never accesses application data or private Git.
- Connection/protocol failures use `connection_failed` / `incompatible_server`; no local fallback exists.
- Keep request/response types operation-specific. Do not expose a generic invoke endpoint to CLI or MCP even if internal routing shares helpers.

### 3. Shared reads and result/error contract

- Implement the exact parity matrix and structured reads in `docs/architecture-agent-access-v0.md`.
- Return one exact accepted projection containing project/revision/authority state, Diagram tree and anchors, Components and exact Markdown, homes/references, Relationships, and derived boundary context.
- Return one complete pending projection containing base, generation, all concrete pending facts, candidate projection when valid, localized validation, stale state, and current review identity without implicitly reviewing.
- Use the shared `workbraid-agent-v1` success/error envelope in CLI JSON and MCP structured results.
- Implement the exact stable error vocabulary and map existing concrete browser/domain conditions into it, including the distinction between uncertain acceptance and acceptance which definitely succeeded but requires reload. Do not make agents parse HTTP status, Git stderr, or prose.
- Include truthful current context on failures wherever the server can determine it. Preserve the epistemic distinction between known non-current and Refresh failed to determine current state.
- Do not add map coordinates, rendered Markdown, source inference, raw canonical writes, or generic graph export.

### 4. Structured mutations

- Expose Component creation/editing, outgoing Relationship add/edit/remove, detail Diagram creation/title edit, home movement, Show here, and Stop showing here through the same candidate-relative backend operations.
- Return created Component/Diagram IDs and every resulting pending generation.
- For agent Component creation, use an explicit Diagram ID when supplied and root only when no Diagram context was supplied.
- Preserve the actual pending representation's exact raw target/label strings, including empty, malformed, unresolved, and whitespace-only values. Implement relationship edit/remove selection as exact source ID + raw old target + raw old label + one-based occurrence among identical matching raw pairs, under exact expected generation. CLI parsing must distinguish an explicitly empty selector value from an omitted required flag. This selector is request-local and creates no Relationship identity or ordering semantics.
- Prove every invalid pending Relationship row repairable through the structured browser remains inspectable, editable, and removable through CLI and MCP. Do not force Discard because a transport prematurely validates an old selector as a domain ID.
- Support literal and file/stdin exact Description/label input in CLI without normalization beyond the already-approved domain rules. MCP accepts exact JSON strings.
- Preserve all current complete-candidate validation, source-order fidelity, home/reference normalization, hierarchy constraints, and invalid-pending retention.
- Invalid, stale, mismatched, or ineligible calls change no pending facts, generation, review, accepted Git, or loaded snapshot.

### 5. Exact review, discard, Refresh, and acceptance

- `changes review` / `changes_review` constructs and validates through the existing single `ConstructCandidate`, captures the same immutable review presentation under the existing lock, and returns exact base commit, candidate tree, generation, complete unified diff, bound Before/With projections, and structured comparison/context.
- No visual map render is required in agent output; Diagram/reference/Relationship comparison facts remain available as structured review context.
- `architecture update` / `architecture_update` requires the caller to resubmit the exact inspected base/tree/generation. No convenience acceptance command/tool, `--force`, `--yes`, accept-latest, mutate-and-accept, or automatic rebuild exists.
- Preserve explicit pre-CAS accepted observation, successor construction only after that observation, mandatory CAS, stale classification, response-loss uncertainty, post-CAS publication, and recovery.
- `changes discard` / `changes_discard` requires exact current generation and clears only the whole pending/review state.
- “Continue editing” remains UI navigation only: inspection plus a subsequent structured mutation uses the same pending set and invalidates the old review. Do not add a second pending state transition.
- Explicit Refresh remains a synchronized application mutation and returns exact adopted/unchanged/non-current/indeterminate state, including canonical slug route changes.

### 6. First-class CLI and embedded skill

- Extend `cmd/workbraid` with the exact command groups/spellings from the Agent Access contract while preserving existing no-command server mode and loopback validation.
- Support global `--server`, `WORKBRAID_SERVER`, and `--json` exactly as approved. Keep help concise and complete.
- In JSON mode emit exactly one envelope on stdout. Use non-zero status on error without replacing the envelope with prose. Keep logs/diagnostics off stdout.
- Implement `workbraid --skill` before any connection/server initialization. Embed one canonical Markdown document into the binary and print it byte-for-byte with no surrounding chatter.
- The skill must be sufficient to drive the black-box workflow from no source/docs knowledge and must accurately describe every command, identity/precondition, review/update, Refresh/stale, discard, typed recovery, exact-text input, and private-Git prohibition.
- Add no skill installer, registry, template generator, dynamic download, or multiple personas.

### 7. Thin MCP stdio bridge

- Use the official Go MCP SDK at a current stable release compatible with the repository's Go version. Approval-time research identifies `modelcontextprotocol/go-sdk` v1.7.0 as current stable, with MCP 2026-07-28 support and backward compatibility with 2025-11-25 and earlier. Record the selected exact dependency and negotiated protocol evidence; do not hard-code the bridge around the older protocol when the SDK/client negotiates the current one.
- `workbraid [--server <loopback-url>] mcp` creates only the stdio protocol adapter and shared loopback client. Global flags precede the command everywhere. It must not create a web handler, `architecture.Manager`, data directory, store, project selection cache, pending copy, or review copy.
- Emit only valid MCP protocol messages on stdout; diagnostics go to stderr.
- Register exactly the approved typed tools with closed input schemas, descriptive fields, object output schemas, structured content plus serialized JSON text compatibility, and truthful annotations.
- Domain/tool execution failures use `isError: true` with the common typed envelope; malformed MCP requests remain protocol errors as the MCP specification requires.
- Tool descriptions must teach process-wide current-project behavior, exact IDs/preconditions, and deliberate review/update sequencing without requiring source knowledge.
- Do not expose generic call/execute/HTTP/Git/YAML/shell/file tools, prompts/resources, background tasks, sampling, elicitation, subscriptions, or agent session state.

### 8. Browser continuity

- Preserve the completed drafting-table workspace, catalog/routes, accepted-only normal projections, Changes/Review dock, safe Markdown, Diagram/reference behavior, explicit Refresh, project switching/discard, and all Phase 2 acceptance semantics.
- An agent mutation need not live-update an unsent browser editor or create collaboration behavior. After a normal reload/reopen or explicit state fetch, the browser must truthfully show the same backend pending/review/accepted state.
- If the browser submits an old mutation after an agent has changed pending work or switched the project, the shared server preconditions reject it without overwriting agent work.
- Do not add agent activity dashboards, identities, cursors, presence, notifications, polling, or disabled future controls.

## Worker acceptance criteria

The implementation is ready for independent review only when:

1. server mode remains the sole Architecture authority and neither CLI nor MCP can instantiate/open domain state;
2. browser, CLI, and MCP reach the same synchronized concrete operations and one pending/base/generation/candidate/review/CAS path;
3. exact state preconditions prevent stale cross-client mutation/discard/acceptance without creating client-specific pending state;
4. every parity-matrix read/action exists through CLI JSON and MCP with equivalent results and typed codes;
5. accepted and pending inspection is sufficient to author safely without UI scraping or private Git;
6. structured Relationship selection handles exact labels and multiplicity without IDs or lost fidelity;
7. review returns the existing exact binding, complete diff, immutable projections/comparison, and acceptance requires that exact binding;
8. Refresh, stale, validation, discard, CAS, uncertain-success, and publication classifications retain their approved meaning;
9. `workbraid --skill` is embedded, deterministic, server-independent, stdout-clean, complete, and matches the built commands;
10. `workbraid [--server <loopback-url>] mcp` is a stateless stdio proxy with discoverable closed tool schemas and structured results;
11. browser workflow and accepted-only projections remain coherent after agent work and after restart;
12. no private-store writer, alternate parser/candidate/review authority, generic framework, remote access, or Phase 3 behavior appears;
13. tests use real Git/server/process paths where authority matters, obey runner-owned async boundaries, terminate all children, and leave a clean tree.

## Required automated evidence

### Shared authority and concurrency

Use real temporary private Git stores and the production handler/application path to prove:

- browser and agent calls observe the same loaded store, accepted revision, pending generation, candidate, validation, and review binding;
- a CLI/MCP-style mutation is visible through the browser response and a browser mutation is visible through agent inspection;
- stale store/revision/generation preconditions reject without mutation;
- races between agent mutation and browser mutation, discard, Refresh, project switch, review capture, or acceptance produce one coherent serialized outcome under the existing lock;
- a mutation/discard racing exact review capture cannot make a mixed response or leave an invalidated binding confirmable;
- a late request for a previously current project cannot mutate the newly selected project;
- no agent client/bridge opens a store or creates a second pending state;
- real accepted-ref stale and CAS success/uncertainty boundaries remain exact.

Keep this a bounded operation matrix, not a generic concurrency harness.

### Agent API and CLI

Build the real binary and run it against one real loopback WorkBraid server with fresh application data. Prove:

- connection URL validation permits literal loopback and rejects remote/ambiguous URLs;
- status/version negotiation and incompatible-server failure;
- stable one-envelope JSON success/error output and exit codes;
- catalog/create/open/current/close and exact project/store identity;
- accepted/pending structured inspection completeness;
- every structured mutation, including pending-new identities, nested Diagram/home/reference composition, and exact Relationship add/edit/remove multiplicity;
- structured edit/remove of empty-label, empty-target, malformed-target, and unresolved-target pending Relationship rows through exact raw selectors, without Discard;
- generation mismatch, validation blocked, wrong project, non-current, review invalidation, CAS conflict, and uncertain acceptance classifications;
- exact review binding/diff and deliberate acceptance;
- complete stop/fresh-process reconstruction solely from accepted Git;
- CLI never reads the application-data directory directly.

### `--skill`

Prove from the built binary that:

- `workbraid --skill` succeeds with no server and an unusable/missing application-data location;
- stdout is only one complete Markdown document and stderr is empty on success;
- output is byte-identical across invocations;
- every documented command exists and every implemented agent command is represented;
- the example and error-recovery instructions use JSON, stable IDs, expected state, exact review binding, and no direct Git/YAML.

Do not test usability by teaching the test a second hidden command table; the later black-box gate is authority for discoverability.

### MCP

Launch `workbraid [--server <loopback-url>] mcp` as a real stdio subprocess connected to the same real server. Through an MCP client using the official SDK's normal negotiated discovery/call lifecycle, prove:

- stdout protocol cleanliness and graceful shutdown;
- exact tool list, unique names, closed/descriptive input schemas, object output schemas, and annotations;
- success structured content conforms to the common envelope and includes compatible JSON text;
- domain failures are typed tool execution errors while invalid protocol/input remains correctly classified;
- all parity actions produce the same semantic results/codes as CLI over the same state;
- two separate MCP bridge processes still observe one server-owned current project and pending generation rather than per-session copies;
- killing/restarting a bridge loses no Architecture state and creates none;
- the bridge cannot function by accessing private stores when the WorkBraid server is stopped.

### Frontend and integrated browser evidence

Add only proportionate frontend/backend coverage to prove shared preconditions and truthful UI after agent-originated pending/review/accepted changes. Use one bounded production-browser scenario which:

- creates/opens a project in the UI;
- performs representative pending mutations and review through the real agent client;
- reloads/reopens and shows the same pending/review state in the browser;
- deliberately accepts through one interface and shows the exact accepted result through the other;
- proves an old browser mutation cannot overwrite newer agent pending state;
- restarts and reconstructs the exact accepted result.

Do not turn the Phase 2 browser gate into a generic cross-transport harness or replay its topology matrix.

### Ordinary checks

From the clean worker tree run each once:

- `git diff --check`;
- `go test ./... -count=1`;
- `go test -race ./... -count=1`;
- `go vet ./...`;
- `go mod verify`;
- `npm test` from `frontend/`;
- `npm run build` from `frontend/`;
- the one bounded Agent Access production-browser scenario;
- the bounded real-binary CLI and MCP integration checks.

Use only the documented frontend test command. Do not invoke raw `vitest`, `npx vitest`, manual render/unmount loops, or unattended retries. If Node, browser, agent, or server processes grow abnormally or fail to terminate, stop them immediately, diagnose, and report rather than increasing limits or piling on workers.

Record exact commands, results, peak-process observations where relevant, tested SHA, dependencies added, and non-blocking warnings. Do not commit binaries, frontend build output, dependencies, logs, databases, temporary stores, transcripts containing environment secrets, or browser artifacts.

## Fresh independent reviewer brief

The reviewer receives the exact packet-inclusive base, full worker range, approved Agent Access contract, current Architecture/UI baselines, relevant completed records, and worker evidence. Review code and tests, not only summaries.

Review specifically:

1. **One authority:** one running handler/application state, one manager, one pending/base/generation/candidate/review/CAS path; CLI/MCP are incapable of local fallback or private-store access.
2. **Concrete sharing:** browser and agent handlers call the same concrete operations without a generic command/event/RPC framework or behavior duplication.
3. **Atomicity:** store/revision/generation checks and state capture/mutation occur under the existing synchronization boundary; cross-client races cannot mix projects, generations, snapshots, or bindings.
4. **Read sufficiency:** accepted/pending/review projections expose all approved semantic facts, exact Markdown/labels/multiplicity, authority knowledge, identities, and recovery state without UI scraping.
5. **Mutation fidelity:** Component, Relationship, Diagram, home, and reference operations preserve the completed product semantics; Relationship occurrence selection creates no identity/order semantics.
6. **Review/acceptance:** exact existing candidate/binding/diff/stale/CAS path only; no accept-latest, force, auto-review, convenience commit, or agent-specific authority.
7. **Errors:** CLI and MCP share stable codes/details and truthful current context; prose/HTTP/Git stderr is never classification authority.
8. **CLI/skill:** exact command surface, JSON/stdout discipline, loopback connection, embedded skill accuracy/completeness, no server requirement for `--skill`.
9. **MCP:** official SDK, standard stdio behavior, schema-complete discoverable tools, structured output/tool errors, no session-owned Architecture state or generic tools.
10. **Browser continuity:** current UI remains coherent, stale browser requests fail safely, and no collaboration/polling/activity chrome entered scope.
11. **Restart/canonical truth:** agent-created accepted state reconstructs from the same private Git authority with no alternate persistence.
12. **Scope/quality:** no remote/auth/multi-user/daemon management, Phase 3 work, broad framework, unsafe test loop, generated artifact, or approved-doc edit.

Every material finding returns to the same worker. The corrected full range receives fresh independent rereview before integration.

## Integration procedure

1. Verify the main worktree is clean at the exact packet-inclusive worker base.
2. Verify the complete worker range is rooted there and includes only Agent Access implementation, tests, embedded skill source, and bounded reviewed corrections.
3. Integrate the exact reviewed range without reconstructing it by hand.
4. Confirm approved baselines, roadmap, packet, and historical completion records are unchanged by implementation.
5. Verify integrated tree identity against the reviewed head.
6. Run the ordinary checks and bounded real-binary/browser integrations once.
7. Stop all test/runtime/browser/MCP/agent processes and verify none remain.
8. Start one fresh authoritative WorkBraid process with fresh application data for the two sequential black-box gates.
9. If a gate exposes an implementation/usability defect within approved semantics, return it to the same worker and require fresh rereview plus rerun. If it exposes a missing product/domain decision, stop for the human.

## Black-box agent usability gates

### Isolation and evidence rules

- Use two fresh projects with distinct names/slugs; the CLI and MCP agents never reuse each other's project.
- Run each agent in a fresh working directory outside the WorkBraid repository with no repository context fork, source/design-doc prompt, private-store path, or Git authority.
- Prefer `gpt-5.6-luna` or another available model meaningfully weaker than the implementation and review agents. Record the exact model.
- The orchestrator may provide the user task, runtime URL/binary for CLI, or configured MCP connection for MCP. It may not answer normal discoverability questions, name commands/tools, repair requests manually, or modify the task after failure.
- Capture the full CLI transcript or MCP tool call/result sequence and retain it as gate evidence, with secrets/environment values redacted but semantic arguments/results intact.

### CLI verifier

Give the agent only this task outcome: using the running WorkBraid instance, create a new project; inspect it; create connected Components; create a nested Diagram; move a Component home; add a reusable Component appearance; inspect all pending work; Review changes; inspect the exact binding and diff; deliberately Update using that binding; then report the project/store identity, accepted revision, hierarchy, and Relationships.

Tell it only to begin with `workbraid --skill`. Do not give any other command name or sequence. The gate passes only if the embedded skill and CLI results/errors are sufficient for independent completion.

### MCP verifier

Give a separate fresh agent the equivalent outcome and a configured `workbraid [--server <loopback-url>] mcp` connection. Give no CLI skill, tool names, call sequence, HTTP URL, source, or docs. The gate passes only if ordinary MCP discovery/descriptions/results/errors are sufficient for independent completion through typed tools and exact-bound acceptance.

### Independent technical verification

After both weak-agent runs, the fresh independent reviewer (or another equally strong independent verifier who did not implement) receives only their transcripts, claimed identities/revisions, runtime application data, and approved contract. Through bounded real Git/object inspection plus a completely fresh WorkBraid process, verify:

- each claimed accepted ref, commit, parent, and tree;
- exact Components, Markdown, Relationships, Diagram hierarchy, homes, references, and boundaries;
- review binding corresponds to the accepted candidate tree;
- no private-Git edit or undocumented HTTP path appears in transcripts;
- no second pending/accepted authority or non-Git canonical state exists;
- restart and `/projects/<slug>` reconstruction match each claim.

Record a separate PASS/FAIL for CLI usability, MCP usability, and technical truth. A prompt patch, human rescue, private-store fix, or acceptance bypass is FAIL.

## Small final human UI checkpoint

Using the same authoritative runtime or a fresh process over the same application data:

1. Open the CLI-created project from the catalog/route. Inspect its exact accepted revision, nested Diagram tree, Components/documentation, home/reference composition, boundaries, and Relationships.
2. Open the MCP-created project and inspect the equivalent accepted result.
3. Verify both workspaces remain coherent and ordinary browser structured editing can begin against the accepted result.
4. Optionally observe one short agent mutation → pending inspection → exact review → update cycle while reopening/reloading the browser to confirm truthful shared state. Do not repeat the complete black-box workflows.
5. Record explicit human **PASS** or **FAIL**.

Agent Access 1 completes only after both black-box gates, independent technical verification, restart reconstruction, and this human checkpoint pass.

## Explicit exclusions and stop

Do not introduce:

- rich-diagram Phase 3 coordinates, manual layout, sizing, routing, bend points, shapes, annotations, graphical authoring, UML, isometric rendering, or agent visual-state tools;
- Planning, Agent Control, Herdr integration, another vertical, autonomous/background agents, or scheduling;
- another daemon, daemon manager, service discovery, registry, public/remote transport, CORS architecture, authentication, permissions, or multi-user collaboration;
- per-client current projects, per-agent branches/pending sets, collaborative unsent-editor merging, presence, cursors, activity feeds, polling, or notifications;
- raw Git/YAML/frontmatter/filesystem/shell/HTTP tools, generic execute/call/patch tools, private-store agent access, or alternate acceptance;
- generic command/event/RPC/plugin/tool/repository/validation/workflow frameworks;
- persisted pending work, proposal branches, history/revert, merge/rebase/reconciliation, or store schema changes;
- a convenience accept-latest/force/yes flow or combined mutate-review-accept operation;
- unrelated Phase 2 polish or recorded future UX candidates.

Stop after explicit Agent Access 1 human **PASS** and its separate completion record. Do not prepare or begin rich-diagram Phase 3.
