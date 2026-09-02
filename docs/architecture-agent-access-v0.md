# Architecture Agent Access v0

Status: Approved

Scope: local CLI, embedded agent skill, MCP tools, and black-box agent usability over the completed Architecture product

This document defines an interface layer over Architecture. It does not define another WorkBraid vertical, another canonical store format, or another Architecture authority.

The prerequisite is completed Architecture Phase 2 at completion record `8cd9ce145980fbc57377e729975940b9e2f0b8b6`, with final implementation `39ca3f7035dc93f63dd32078506b6752ee271ec5`.

## Product boundary

Agent Access gives local agents the same semantic reads and deliberate authoring flow already available through the browser.

It preserves exactly one running WorkBraid application authority:

- one loaded/current project for the process;
- one loaded accepted snapshot and current/non-current knowledge state;
- one backend-held pending Architecture change set;
- one exact pending base and generation;
- one candidate constructor and version-aware validator;
- one immutable review binding and exact diff;
- one confirmation, stale-observation, compare-and-swap, and publication path.

The browser, CLI, and MCP are clients of that authority. A CLI invocation or MCP connection never opens a private store through `architecture.Manager`, constructs its own pending state, or writes Git itself.

Agent Access remains local, loopback-only, single-user, and deliberately non-collaborative. Different clients may observe and mutate the same backend-held pending set, but simultaneous unsent browser editing and agent mutation is not a collaborative editing model. Client-visible preconditions and typed conflicts prevent a stale client from silently acting on another state.

The portable format-v2 store contract in `docs/architecture-v0.md` is unchanged.

## Runtime and transport decision

### Current application shape

The current binary starts one Go HTTP process bound to a literal loopback address. Its web handler owns an `architecture.Manager`, the loaded project/snapshot, stale knowledge, the pending set, generation, candidate, review binding, and accepted publication state under one concrete mutex. Browser routes call that handler. This is the authority Agent Access must reuse.

### Alternatives

1. **CLI or MCP opens stores directly.** This is superficially small but creates another loaded snapshot and pending authority in each process. Browser and agent work can diverge, review bindings cease to be singular, and accepting through one process can invalidate hidden work in another. Reject.
2. **Expose MCP directly as another transport inside the long-running process.** This keeps one authority but makes ordinary stdio MCP configuration awkward: an MCP client expects to launch and own a stdio subprocess, while the WorkBraid HTTP/UI process is already running independently. It also couples the application's terminal lifecycle to an agent client. Do not use for Agent Access 1.
3. **Use one same-process local agent HTTP surface, with thin CLI and stdio MCP clients.** The running WorkBraid process still owns all state. One-shot CLI commands and a client-launched MCP stdio bridge call that same loopback surface. The bridge owns no Architecture state. Recommend.
4. **Add a Unix socket, daemon discovery, or registry.** This could avoid an explicit URL but adds lifecycle and platform machinery without improving the authority model. Defer.

### Chosen arrangement

The existing WorkBraid process exposes a small versioned JSON agent surface on its existing literal-loopback listener. Browser handlers and agent handlers enter the same concrete application operations and synchronization boundary; they do not implement parallel domain behavior.

The same `workbraid` binary has three modes:

- with the existing server flags and no agent subcommand, it runs the one authoritative Go/UI process;
- with a CLI command, it is a one-shot HTTP client of an already-running WorkBraid process;
- with `mcp`, it is a stdio MCP server which acts only as a typed transport adapter to that already-running process.

Neither client mode creates an `architecture.Manager`, opens application data, accepts a data-directory argument, or contains a fallback local implementation. If the authoritative process is unavailable, the operation fails as `connection_failed`.

The connection is explicit rather than discovered:

- `--server http://127.0.0.1:8080` selects the running instance;
- `WORKBRAID_SERVER` may supply the same value;
- otherwise the default is `http://127.0.0.1:8080`.

Command-line configuration wins over the environment. Client modes accept only plain HTTP URLs whose host is a literal loopback IP and which contain no path, query, user information, or fragment. There is no daemon management, port scan, PID file, remote URL, TLS, authentication, or service discovery in Agent Access 1.

The local agent HTTP surface has an explicit protocol version returned by a non-mutating status request. A CLI or MCP bridge that cannot use the running server's version fails as `incompatible_server`; it never falls back to opening stores itself.

Browser-origin protection remains in place. The agent surface requires JSON for requests with bodies, is reachable only through the literal-loopback listener, does not enable CORS, and rejects an unexpected browser `Origin`. These are concrete local-process safeguards, not an authentication or permission system.

The stdio bridge follows the current stable MCP stdio protocol through the official Go MCP SDK. It writes only MCP messages to stdout and diagnostics only to stderr. MCP connection/session lifetime never owns project selection, pending state, or review state; opening a project changes the one process-wide current project exactly as it does for the browser and CLI.

## Shared operation contract

One small set of concrete application operations sits behind the browser and agent transports. It may use operation-specific request/result types. It is not a generic command bus, event stream, RPC framework, repository layer, or operation algebra.

Every agent mutation is adjudicated under the existing backend synchronization boundary from server-owned loaded project and candidate state. Agent-supplied IDs and expected values are preconditions, never authority.

### State preconditions

Architecture mutations submit:

- the exact current store UUID returned by WorkBraid;
- the exact loaded accepted revision the agent inspected; and
- the exact pending generation it inspected, represented as `null` when no pending set existed.

The server compares all three against its current state atomically before mutation. A mismatch changes nothing and returns a typed error. Successful mutation returns the current store UUID, accepted revision, and resulting pending generation.

Review submits the same current state and generation. Review returns the exact base commit, candidate tree, and generation binding. Acceptance submits that exact binding in addition to store identity. The existing binding checks, explicit accepted-ref observation, successor construction, mandatory CAS, and post-CAS classification remain final authority.

Discard requires the exact pending generation observed by the caller so a stale agent cannot discard work created or changed afterward. Refresh and closing the current project use exact loaded identity and the existing synchronized state transition. Opening or creating another project remains one atomic process-wide selection and retains the existing pending-work guard. These actions preserve the existing pending, stale, known-non-current, and indeterminate-failure semantics.

Read operations return the state tokens needed for the next safe operation. They do not implicitly Refresh, open another project, construct a review, or mutate selection.

### Semantic parity matrix

| Product capability | Browser | CLI | MCP | Agent result requirements |
| --- | --- | --- | --- | --- |
| Check connection/current project | application load | `status` / `project current` | `status`, `project_current` | protocol version, current project or none, authority state |
| List catalog | project catalog | `project list` | `projects_list` | name, slug, store UUID, revision, conflict/unavailable state |
| Create project by name | **New project** | `project create` | `project_create` | selected project, generated slug/store UUID/root Diagram ID, accepted revision |
| Open/select project | catalog/route | `project open` | `project_open` | process-wide selected project and exact loaded revision |
| Leave current project | **Open another project** | `project close` | `project_close` | no current project; pending guard remains exact |
| Inspect accepted Architecture | workbench | `architecture inspect` | `architecture_inspect` | one compact exact accepted projection described below |
| Explicit Refresh | **Refresh** | `architecture refresh` | `architecture_refresh` | adopted/unchanged/known-non-current/indeterminate result and exact current locator |
| Inspect pending and validation | **Changes in progress** | `changes inspect` | `changes_inspect` | base, generation, all pending semantic changes, candidate if valid, validation locations, review state |
| Create Component | **Add component** | `component create` | `component_create` | generated stable Component ID, candidate home, new generation |
| Edit Title/Description | **Edit component** | `component edit` | `component_edit` | exact changed fields and new generation |
| Add outgoing Relationship | structured relationship row | `relationship add` | `relationship_add` | exact source/target/label fact and new generation |
| Edit outgoing Relationship | structured relationship row | `relationship edit` | `relationship_edit` | exact removed/added fact selection and new generation |
| Remove outgoing Relationship | structured relationship row | `relationship remove` | `relationship_remove` | exact removed occurrence and new generation |
| Create detail Diagram | **Create detail diagram** | `diagram create-detail` | `diagram_create_detail` | generated stable Diagram ID, anchor Component ID, title, new generation |
| Rename Diagram | **Edit title** | `diagram edit-title` | `diagram_edit_title` | Diagram identity/title and new generation |
| Move Component home | **Change where … lives** | `component move-home` | `component_move_home` | Component/destination IDs and new generation |
| Add reusable appearance | **Show component here** | `diagram show-component` | `diagram_show_component` | Diagram/Component IDs and new generation |
| Remove reusable appearance | **Stop showing here** | `diagram stop-showing-component` | `diagram_stop_showing_component` | Diagram/Component IDs and new generation |
| Review complete changes | **Review changes** | `changes review` | `changes_review` | exact binding, complete diff, bound projections, structured comparison/context |
| Return from review to pending editing | **Continue editing** | no state-changing command | no state-changing tool | review is not another draft; `changes inspect` and the next structured mutation continue the same pending set and invalidate the old binding |
| Discard whole pending set | **Discard changes** | `changes discard` | `changes_discard` | consumed generation and unchanged accepted revision |
| Deliberately accept | **Update architecture** | `architecture update` | `architecture_update` | submitted binding, accepted successor/revision, publication classification |

Pan, zoom, Fit map, dock expansion, visual selection, clear focus, and breadcrumb clicks are browser presentation/navigation state and have no CLI/MCP equivalent. `architecture inspect` / `architecture_inspect` returns the complete structured Diagram hierarchy and composition needed for semantic navigation and authoring without adding agent-owned selection state. Agent Access 1 does not need granular Component- or Diagram-inspect commands unless the black-box usability gate demonstrates a concrete missing read.

## Structured Architecture reads

`architecture inspect` / `architecture_inspect` returns one compact projection of the exact loaded accepted snapshot:

- project name, current slug, store UUID, accepted revision, and authority knowledge state;
- root Diagram ID and the strict Diagram tree with Diagram IDs, titles, parent anchor Component IDs, and child IDs;
- every Component's stable ID, title, exact Markdown body/Description, canonical filename, home Diagram ID, reference Diagram IDs, and outgoing Relationship facts;
- Diagram appearances identified by Diagram ID plus Component ID and home/reference role;
- derived boundary context for each Diagram: external Component/home Diagram plus every exact crossing Relationship occurrence;
- enough collision context to distinguish duplicate Component or Diagram titles.

This is a projection of the existing immutable snapshot. It contains no map coordinates, rendered Markdown, browser selection, inferred source model, generic graph export, or independently persisted data.

`changes inspect` / `changes_inspect` returns:

- exact pending base revision and generation;
- the complete pending Component, Relationship, Diagram title/detail, home-move, and reusable-appearance changes;
- the complete candidate projection when construction is valid;
- typed validation code and stable Component/Diagram/relationship location when invalid;
- stale/read-only state;
- whether a still-current review binding exists and, when it does, that exact binding.

It does not silently construct a review. Invalid pending work remains inspectable and correctable through structured mutations; it never gains a review or acceptance path.

## CLI contract

The CLI is implemented in the WorkBraid binary, not as a shell script or browser scraper. Server mode remains the existing default when no CLI/MCP command is selected.

The initial domain commands are exactly:

```text
workbraid status
workbraid project list
workbraid project current
workbraid project create
workbraid project open
workbraid project close
workbraid architecture inspect
workbraid architecture refresh
workbraid architecture update
workbraid changes inspect
workbraid changes review
workbraid changes discard
workbraid component create
workbraid component edit
workbraid component move-home
workbraid relationship add
workbraid relationship edit
workbraid relationship remove
workbraid diagram create-detail
workbraid diagram edit-title
workbraid diagram show-component
workbraid diagram stop-showing-component
workbraid [--server <loopback-url>] mcp
workbraid --skill
```

Client-mode syntax is `workbraid [--server <loopback-url>] [--json] <group> <action> [action flags]`; the global flags precede the command. MCP is `workbraid [--server <loopback-url>] mcp`. `--skill` is the one standalone form and ignores client/server configuration.

The minimum action flags are:

| Command | Required/optional action input |
| --- | --- |
| `project create` | `--name` |
| `project open` | `--slug` |
| `project close` | `--store-id` |
| `architecture refresh` | `--store-id`, `--accepted-revision` |
| `architecture update` | `--store-id`, `--base-revision`, `--candidate-tree`, `--generation` |
| `changes review` | `--store-id`, `--accepted-revision`, `--generation` |
| `changes discard` | `--store-id`, `--generation` |
| `component create` | state preconditions, `--title`, optional `--description` or `--description-file`, optional `--diagram-id` |
| `component edit` | state preconditions, `--component-id`, and at least one of `--title`, `--description`, or `--description-file` |
| `component move-home` | state preconditions, `--component-id`, `--diagram-id` |
| `relationship add` | state preconditions, `--source-id`, `--target-id`, and `--label` or `--label-file` |
| `relationship edit` | state preconditions, `--source-id`, `--old-target-id`, `--old-label` or `--old-label-file`, optional `--occurrence` defaulting to `1`, `--target-id`, and `--label` or `--label-file` |
| `relationship remove` | state preconditions, `--source-id`, `--target-id`, `--label` or `--label-file`, optional `--occurrence` defaulting to `1` |
| `diagram create-detail` | state preconditions, `--component-id`, `--title` |
| `diagram edit-title` | state preconditions, `--diagram-id`, `--title` |
| `diagram show-component` / `stop-showing-component` | state preconditions, `--diagram-id`, `--component-id` |

Here, state preconditions mean `--store-id`, `--accepted-revision`, and `--generation`, where `--generation none` means the caller observed no pending set. Read-only `status`, `project list`, `project current`, `architecture inspect`, and `changes inspect` take no action input. `--help` documents these exact flags and their identity/precondition meaning.

Domain commands accept `--json`. Once JSON mode is recognized, command/flag validation failures also produce the one JSON error envelope. In JSON mode stdout contains exactly one machine result envelope and no prose, progress, or log prefix. Failures still emit the JSON error envelope and use a non-zero exit status. Human-readable non-JSON output is allowed but is not the agent contract. A process-level parse failure before JSON mode can be recognized may use stderr.

Names, slugs, Component titles, and Diagram titles are never accepted as semantic identity where stable IDs exist. Project `open` uses the catalog slug as its locator and returns the store UUID. Component/Diagram mutations use IDs returned by WorkBraid.

Description and Relationship label inputs support exact UTF-8 through literal flags for simple values and file/stdin flags for multiline or shell-sensitive values. A CLI must not require lossy shell quoting to preserve valid Markdown or Relationship labels.

Because Relationships have no identity and order has no domain meaning, selectors operate over the backend's current pending representation:

- add supplies source Component ID, target Component ID, and exact label;
- every backend-held authored row retains its exact raw target and label strings before candidate validation, including empty, malformed, or unresolved target text and blank/whitespace-only labels;
- edit/remove select a row by source ID, exact raw old target, exact raw old label, and a one-based occurrence among identical matching raw pairs;
- edit supplies the new exact target and label;
- the occurrence is a request-local selector over the inspected pending authored source order, not a Relationship ID or lifecycle;
- the expected pending generation prevents that selector from applying after another mutation;
- surviving declarations retain their existing source order as already required.

The CLI distinguishes a required selector flag which is present with an empty value from an omitted flag; MCP schemas likewise allow the exact empty string for old target/label selectors. Edit/remove selection applies to the exact current authored outgoing declarations returned by `changes inspect`, including an incomplete row in an invalid pending set. The returned validation location and raw row values are sufficient to address it. Every invalid pending Relationship state repairable through the structured browser is therefore repairable through CLI and MCP without manufacturing a valid candidate, adding row identity, falling back to raw frontmatter, or forcing whole-set Discard.

Component creation accepts an explicit home Diagram ID; omitting it means there is genuinely no Diagram context and uses the root. Agent examples should pass it explicitly when the intended Diagram is known.

### Machine result envelope

Every JSON-mode command and every MCP tool uses the same logical envelope:

```json
{
  "protocol": "workbraid-agent-v1",
  "ok": true,
  "context": {
    "project": {"store_id": "...", "name": "...", "slug": "..."},
    "accepted_revision": "...",
    "authority_state": "current",
    "pending_generation": 3
  },
  "result": {}
}
```

When no project or pending set exists, the corresponding context value is `null`. A domain failure has `ok: false`, the best truthful current context, and:

```json
{
  "error": {
    "code": "pending_generation_mismatch",
    "message": "Pending work changed. Inspect changes again before editing.",
    "details": {}
  }
}
```

`message` is concise product language for humans. Agents branch on `code`, never text. `details` contains bounded typed identity, expected/current values, or validation location needed to recover; it does not expose arbitrary Git stderr.

## Stable error vocabulary

CLI and MCP use the same codes and meanings:

| Code | Meaning / next safe action |
| --- | --- |
| `invalid_request` | Input is missing, malformed, or violates the command/tool schema; correct it. |
| `connection_failed` | The configured running WorkBraid instance could not be reached; start/check that process and retry. |
| `incompatible_server` | The running instance does not support this agent protocol; use a matching binary/server. |
| `project_not_found` | No catalog project owns that slug; list projects or create deliberately. |
| `project_conflict` | More than one store claims the slug; no project was selected. |
| `project_unavailable` | The catalog knows the store but cannot load a valid accepted Architecture. |
| `project_not_open` | No project is currently selected in the authoritative process. |
| `project_mismatch` | The current project/store is not the one the caller inspected; inspect/open again. |
| `architecture_non_current` | WorkBraid knows the loaded projection is not authoritative; Refresh or discard stale pending work as appropriate. |
| `refresh_failed` | WorkBraid could not determine current accepted authority; preserve prior knowledge and retry explicitly. |
| `pending_conflict` | Pending work blocks project creation/switch/close or the requested pending action. Inspect or discard deliberately. |
| `pending_generation_mismatch` | The one pending set changed since the caller inspected it; inspect again. |
| `target_not_found` | A referenced Component or Diagram ID does not exist in the complete current candidate. |
| `target_not_eligible` | The target exists but the requested semantic action is not allowed in the current candidate. |
| `validation_blocked` | The complete pending candidate is structurally invalid; use the returned locations and correct it. |
| `review_required` | No current successful review is bound to the pending candidate; review first. |
| `review_invalidated` | The supplied or stored base/tree/generation binding is no longer current; inspect and review again. |
| `accepted_conflict` | Accepted authority changed before CAS; pending work is preserved under existing stale rules. |
| `acceptance_uncertain` | The server could not determine the final acceptance result; inspect/Refresh rather than retrying blindly. |
| `accepted_reload_required` | CAS definitely succeeded but in-process publication/reload failed; do not retry acceptance, inspect/Refresh/reopen the canonical result. |
| `unsupported_action` | The action is outside the current product or state; do not simulate it with Git/YAML. |
| `operation_failed` | A bounded internal operation failed without a more specific safe classification; inspect current state before retrying. |

HTTP status, CLI exit status, and MCP `isError` may aid transport handling, but this code is the shared domain classification.

## `workbraid --skill`

`workbraid --skill` is a standalone, non-mutating binary feature:

- it requires no running server or open project;
- it prints one complete Markdown instruction document to stdout;
- stdout begins with the Markdown document and contains no banner, log, fence, or trailing commentary;
- it exits successfully after writing the document;
- the canonical Markdown is embedded into the built binary at build time;
- it documents only commands implemented by that same binary.

The skill is concise but sufficient for an agent with no repository or design-document access. It explains:

- what WorkBraid Architecture does and the one-running-process requirement;
- explicit `--server` / `WORKBRAID_SERVER` connection and `status`;
- catalog discovery, create/open/current project behavior;
- store UUID versus slug versus Component/Diagram IDs versus titles;
- inspecting accepted Architecture and pending work before mutation;
- every structured authoring command and exact text/file inputs;
- one pending base/generation and how typed conflicts require reinspection;
- Review changes, inspection of exact base/tree/generation/diff/context, and exact-bound Update;
- Refresh, known-non-current versus failed-to-determine behavior;
- whole-set Discard and its destructive scope;
- recovery steps for the stable typed error vocabulary;
- one small end-to-end JSON CLI example which creates a project, creates connected Components and a detail Diagram, reviews, and deliberately updates;
- an explicit prohibition on editing private WorkBraid Git stores or using raw Git/YAML as an alternate product path.

There is no skill installer, registry, generator, persona choice, online fetch, or second skill document.

## MCP contract

`workbraid [--server <loopback-url>] mcp` is a client-launched stdio MCP server. Global flags precede the command in help, the embedded skill, tests, examples, and black-box configuration. It uses the official Go MCP SDK and exposes only tools. Resources, prompts, sampling, elicitation, background tasks, subscriptions, and server-initiated Architecture behavior are unnecessary.

Implementation research at approval found official `modelcontextprotocol/go-sdk` v1.7.0 to be the current stable release, supporting MCP protocol 2026-07-28 while retaining compatibility with 2025-11-25 and earlier. Implementation still selects a current stable SDK version compatible with the repository Go version and records the exact dependency plus negotiated protocol evidence. WorkBraid does not hard-code behavior around an older protocol when the selected SDK and client negotiate the current protocol.

Each tool has:

- a precise discoverable description which explains whether it reads or mutates the one process-wide current project;
- a closed JSON Schema input with descriptions for identities and exact preconditions;
- an object output schema matching the shared result envelope;
- `structuredContent` conforming to that schema and the same serialized JSON in text content for client compatibility;
- correct read-only/destructive/idempotence annotations where they are truthful;
- domain execution failures returned as `isError: true` with the structured shared error envelope, not English-only or protocol errors.

The initial tool names are those in the parity matrix:

```text
status
projects_list
project_current
project_create
project_open
project_close
architecture_inspect
architecture_refresh
architecture_update
changes_inspect
changes_review
changes_discard
component_create
component_edit
component_move_home
relationship_add
relationship_edit
relationship_remove
diagram_create_detail
diagram_edit_title
diagram_show_component
diagram_stop_showing_component
```

Tool implementations only translate typed MCP input to the same local agent HTTP client used by CLI and translate its result back to MCP. They do not retain a current project, snapshot, pending generation, selected Diagram, reviewed candidate, retry policy, or acceptance convenience. There is no generic `call_api`, `execute`, `patch_yaml`, `run_git`, or arbitrary command tool.

## Agent usability gate

Agent Access 1 is incomplete until two fresh black-box agents and a stronger independent technical verifier pass.

### CLI black-box gate

Use a fresh project and an agent meaningfully weaker than the implementation/review agents, preferably Luna. Give it only:

- a representative Architecture task;
- the built `workbraid` binary and configured running loopback instance;
- an isolated working directory outside the repository; and
- the instruction to begin by running `workbraid --skill`.

Do not give it repository source, design docs, an execution packet, private-store access, endpoint names, command names after the initial `--skill` instruction, or orchestration hints.

It must independently discover and complete: create/open project, inspect, create connected Components, create/navigate nested Diagram semantics, move a home, show a reusable Component, inspect pending state, review the exact candidate and diff, deliberately accept the exact binding, and report the accepted revision/state.

### MCP black-box gate

Use a separate fresh project and a separate fresh weak agent. Give it only:

- an equivalent representative Architecture task; and
- a configured WorkBraid MCP connection with normal MCP tool discovery.

Do not teach tool names, call order, repository concepts, HTTP endpoints, or private Git. It must independently complete the equivalent structured workflow and deliberate exact-bound acceptance through MCP.

### Pass criteria and evidence

Each verifier must complete without source inspection, private-Git writes, undocumented HTTP calls, orchestrator hints, human intervention, or review bypass. Confusing help/skill/tool descriptions, missing reads, ambiguous typed recovery, or inability to discover the next safe action is an Agent Access product failure; do not patch the verifier prompt.

Record for each gate:

- exact implementation and binary SHA;
- model/version and isolated starting conditions;
- complete CLI transcript or MCP tool-call/result sequence;
- exact project slug/store UUID;
- review binding and accepted revision claimed by the agent;
- any failed calls and whether the documented/tool-described recovery was sufficient.

After both weak-agent runs, a stronger independent technical verifier inspects the resulting private Git stores through bounded real Git/object reads and uses a fresh WorkBraid process to confirm the exact accepted revisions, Components, documentation, Relationships, Diagram hierarchy, homes, references, and restart reconstruction claimed by each agent. The verifier confirms no second pending/accepted authority or private-store mutation path was used.

The final human checkpoint is intentionally small: open both agent-created projects in the browser, inspect their accepted results and coherent workspace, optionally observe one short agent pending/review/update cycle, and explicitly PASS or FAIL. The human does not repeat either workflow manually.

## Deliberate exclusions

Agent Access 1 does not introduce:

- another WorkBraid vertical, Planning, Agent Control, or Herdr integration;
- another backend daemon, service discovery, PID/port registry, distributed session, or public API;
- remote network exposure, authentication, authorization, permissions, or multi-user coordination;
- collaborative unsent-editor merging, per-client current projects, per-agent branches, or per-agent pending sets;
- autonomous/background agents, scheduled work, notifications, polling, or watchers;
- a generic command/event bus, RPC framework, plugin system, tool framework, repository layer, or operation algebra;
- raw Git, YAML/frontmatter, filesystem, shell, arbitrary HTTP, or generic patch tools;
- a convenience accept-latest/force/yes path, automatic review, or combined mutate-review-accept tool;
- persisted pending work, proposal branches, history/revert, merge/rebase/reconciliation, or new Architecture store fields;
- Phase 3 coordinates, manual layout, routes, bend points, shapes, graphical editing, UML, isometric rendering, or renderer-specific agent state.

Stop after Agent Access 1 passes. Rich-diagram Phase 3 remains deferred.
