# Architecture Agent Access

Status: Approved living contract

## Phase 3.3 routing surface

Delegated root approval of routing proposal `d119baeb-1b2b-4451-8113-7f4fe05678ec` generation 8, reviewed state `595bab6de6933c7a047c70520f1830594f166fed`, authorizes these additive agent-v2 operations under the unchanged authority/envelope contract:

| CLI | MCP | API suffix under `/api/agent/v2` |
| --- | --- | --- |
| `diagram routes` | `diagram_routes` | `/diagrams/routes` |
| `diagram set-route` | `diagram_set_route` | `/diagrams/set-route` |
| `diagram restore-default-route` | `diagram_restore_default_route` | `/diagrams/restore-default-route` |

Inspect requires `--store-id --diagram-id`, with optional `--change-set-id` for an exact valid active/Applied proposal; omission selects Accepted. Return complete Diagram/source/target/exact-label/occurrence addresses, tuple counts, stored custom versus default, displayed bend, eligibility and reason. Reads write nothing. Set requires `--store-id --change-set-id --generation --diagram-id --source-id --target-id (--label|--label-file <path|->) --occurrence --bend`. Restore takes the same fields except bend. Every listed value is required; literal/file are mutually exclusive, `-` reads stdin, and valid UTF-8 label bytes remain exact. MCP uses matching snake_case identity/state fields and literal `label`; file transport is CLI-only. Reject missing/unknown/duplicate fields, nulls, wrong types, non-positive occurrences and bends outside integer −100000..100000.

These addresses are presentation slots, never Relationship IDs or source-row/projection/boundary keys. The backend checks exact tokens and eligibility before no-op comparison. Self/coincident routing is `target_not_eligible`; changed node geometry may retain stored bends while displaying the specified fallback. [Placement §10](architecture-placement-amendment-v0.md#10-deliberate-link-routing-phase-33) governs signed control-point distance, default/custom continuity, conservative tuple resets, invalid-row retention, no resurrection and v5 upgrade/no-op behavior. Clients must expose these loss consequences without constructing candidates or changing acceptance authority.

Reconciliation exposes the exact `route_value` and `route_loss` locators/choices in [Reconciliation](architecture-reconciliation-v0.md#phase-33-routing-reconciliation): manual bend/default, or acknowledged whole-tuple clear. Automatic count/visibility loss has no route-owned semantic override. Preserve exact S/B/A/P, residual replay and separate Review/Update. Existing invalid_request, target_not_found, target_not_eligible, validation_blocked and state/lifecycle errors retain their meaning. Help, embedded skill, MCP descriptions/closed schemas and HTTP behavior must agree, including exact label file/stdin fidelity and bounded fresh discovery evidence. No generic presentation endpoint or client-local authority is introduced.

Browser, CLI and MCP are clients of one loopback WorkBraid authority. [Proposals and Reviews](architecture-proposals-v0.md) defines durable work and feedback; [Reconciliation](architecture-reconciliation-v0.md) defines exact preview/apply; [Placement](architecture-placement-amendment-v0.md) defines the approved corrected placement surface. This interface adds no separate domain, private-store client, daemon or acceptance authority.

## Runtime and connection

Renderer-private intersection availability is browser-only state. Undefined/nonfinite intersections do not add backend eligibility restrictions or error codes: otherwise eligible noncoincident endpoints still allow numeric/CLI/MCP bend authoring. The browser retains scalars and reports canvas dragging/fallback limitations under Placement §10; it resumes rendering without a write when representable.

One Go/UI process owns the selected project, loaded Accepted snapshot and authority knowledge, independently identified durable proposals, generation/review bindings, one candidate constructor and one synchronization/ref-CAS boundary. Different clients share that process; unsent editor collaboration is not implied. Project open changes process-wide selection; proposals are always addressed explicitly.

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

The stdio bridge uses the official Go MCP SDK and negotiates supported protocol versions. It writes only MCP messages to stdout and diagnostics only to stderr. MCP connection/session lifetime never owns project selection, pending state, or review state; opening a project changes the one process-wide current project exactly as it does for the browser and CLI.

## Exact state preconditions

Every authoring mutation carries exact `store_id`, `change_set_id`, and that proposal's `generation`. No client-selected current proposal or global pending generation exists. The backend checks these against its immutable loaded record and actual ref under the lock before mutating only that proposal.

Creation carries exact `store_id` and observed `accepted_revision`, and starts at generation 0. Browser Accepted-origin edits verify those tokens and publish first mutation/generation 1 atomically without an empty intermediate proposal. Agents explicitly create proposals before editing. Real mutations increment only that proposal's generation and clear its Review; genuine no-ops do neither.

Review requires store/proposal/generation and returns exact `reviewed_state`, `base_revision`, `candidate_tree`, generation, complete diff and bound projections. Update requires store/proposal plus that exact base/tree/generation binding; it verifies current Accepted equals base and uses the one three-ref acceptance transaction. No force, accept-latest, implicit review or combined mutate/review/accept tool exists.

Discard verifies store/proposal/generation and deletes only the exact active ref. Submitted Reviews retain their parents. Refresh uses exact store and observed Accepted revision; project close requires exact store. Switching projects does not discard durable work. Unknown or non-current Accepted pauses mutation/new review/acceptance without confusing it with a merely out-of-date proposal, which remains editable/reviewable against its own original base. Existing exact review feedback submission has the separately defined authority exception.

Reads return necessary tokens and do not implicitly Refresh, open projects, prepare Review or mutate selection. Reconciliation additionally checks exact S/B/A/P as specified in its contract. Immutable feedback submission checks exact reviewed_state and binding. Failed/raced calls retain truthful context rather than retrying blindly.

## Command and tool parity

| Capability | CLI | MCP |
| --- | --- | --- |
| Connection | `status` | `status` |
| Catalog | `project list` | `projects_list` |
| Current project | `project current` | `project_current` |
| Create/open/close project | `project create`, `project open`, `project close` | `project_create`, `project_open`, `project_close` |
| Accepted inspect/Refresh/Update | `architecture inspect`, `architecture refresh`, `architecture update` | `architecture_inspect`, `architecture_refresh`, `architecture_update` |
| Proposals | `change-set list`, `create`, `inspect`, `rename`, `edit-proposal`, `review`, `discard` | `change_sets_list`, `change_set_create`, `change_set_inspect`, `change_set_rename`, `change_set_edit_proposal`, `change_set_review`, `change_set_discard` |
| Component authoring | `component create`, `edit`, `move-home` | `component_create`, `component_edit`, `component_move_home` |
| Relationship rows | `relationship add`, `edit`, `remove` | `relationship_add`, `relationship_edit`, `relationship_remove` |
| Diagram authoring | `diagram create-detail`, `edit-title`, `show-component`, `stop-showing-component` | `diagram_create_detail`, `diagram_edit_title`, `diagram_show_component`, `diagram_stop_showing_component` |
| Detail reassignment | `diagram parent-options`, `reassign-detail` | `diagram_parent_options`, `diagram_reassign_detail` |
| Submitted feedback | `review-submission list`, `inspect`, `submit` | `review_submissions_list`, `review_submission_inspect`, `review_submission_submit` |
| Reconciliation | `change-set reconcile-preview`, `reconcile-apply` | `change_set_reconcile_preview`, `change_set_reconcile_apply` |
| Placement | `diagram positions`, `set-position`, `auto-layout` | `diagram_positions`, `diagram_set_position`, `diagram_auto_layout` |
| Sizing | `diagram sizes`, `set-size`, `restore-default-size` | `diagram_sizes`, `diagram_set_size`, `diagram_restore_default_size` |

Sizing uses API suffixes `/diagrams/sizes`, `/diagrams/set-size`, `/diagrams/restore-default-size` under agent-v2. Inspection takes store/Diagram IDs and optional Change Set ID and reports all visible pairs with stored size when v4, displayed size and `size_source: stored|derived`. Set-size takes exact store/proposal/generation/Diagram/Component plus integer `width`/`height` (CLI `--width`, `--height`). Restore takes the same state/identity without dimensions. Bounds, current-role defaults, no-op-before-upgrade and results follow [Placement §9](architecture-placement-amendment-v0.md#9-complete-visible-node-sizing-phase-32). No-op preserves version/tree/generation/Review on legacy formats too. Clients share one server operation and never capture browser geometry as authority.

Placement Auto-layout and all-visible-node semantics were delivered in Phase 3.1 with human visual acceptance. The built binary's help/skill must advertise only its implemented surface. Trial reset commands/tools have no compatibility aliases in the corrected product. The old agent-v1 `changes` commands and `pending_generation` are also removed, not ambiguous aliases.

Pan, zoom, Fit, selection, dock expansion and breadcrumbs are browser view state. Agents inspect complete hierarchy/composition and exact coordinate context instead of owning navigation state. No generic call_api/execute/patch_yaml/run_git/shell/filesystem tool exists.

## Structured reads

Accepted inspection returns exact project name/slug/store UUID, revision/authority knowledge, root/tree with parent anchors and child IDs, Component IDs/titles/exact Description/canonical Markdown source/filename/home/references/outgoing facts, Diagram appearances, and coalesced boundary/crossing-Relationship context. Duplicate titles carry enough identity context for selection. Rendering and inferred source models are not exported as authority.

Proposal inspection requires exact store/proposal IDs and returns ID/name/lifecycle, state commit, base, generation, proposal Markdown, concrete facts, validity and localized repairable validation, exact candidate tree/projection when valid, out-of-date context, current review and applied revision where applicable. Invalid work remains addressable without a partial map or invented valid candidate. Inspect does not prepare a review.

Placement reads cover every visible Component, including boundary nodes. They distinguish exact persisted v3 coordinates from derived v2 fallback; no ongoing Automatic mode is inferred. Historical review inspection returns its exact parent/binding, comments, proposal Markdown, Before/With, comparison and diff separately from current lifecycle/authority context. Exact response/anchor rules remain in the owning contracts.

## CLI syntax and text fidelity

Client syntax is `workbraid [--server <loopback-url>] [--json] <group> <action> [action flags]`; global flags precede the command. `workbraid [--server <loopback-url>] mcp` launches the stateless stdio bridge. Standalone `workbraid --skill` requires no server. Help documents exact IDs and preconditions.

The minimum action flags are:

| Command | Required/optional action input |
| --- | --- |
| `project create` | `--name` |
| `project open` | `--slug` |
| `project close` | `--store-id` |
| `architecture refresh` | `--store-id`, `--accepted-revision` |
| `architecture update` | `--store-id`, `--change-set-id`, `--base-revision`, `--candidate-tree`, `--generation` |
| `change-set review` / `discard` | `--store-id`, `--change-set-id`, `--generation` |
| `change-set list` | `--store-id` |
| `change-set create` | `--store-id`, `--accepted-revision`, optional `--name` |
| `change-set inspect` | `--store-id`, `--change-set-id` |
| `change-set rename` | state preconditions, `--name` |
| `change-set edit-proposal` | state preconditions, `--proposal` or `--proposal-file` |
| `component create` | state preconditions, `--title`, optional `--description` or `--description-file`, optional `--diagram-id` |
| `component edit` | state preconditions, `--component-id`, and at least one of `--title`, `--description`, or `--description-file` |
| `component move-home` | state preconditions, `--component-id`, `--diagram-id` |
| `relationship add` | state preconditions, `--source-id`, `--target-id`, and `--label` or `--label-file` |
| `relationship edit` | state preconditions, `--source-id`, `--old-target-id`, `--old-label` or `--old-label-file`, optional `--occurrence` defaulting to `1`, `--target-id`, and `--label` or `--label-file` |
| `relationship remove` | state preconditions, `--source-id`, `--target-id`, `--label` or `--label-file`, optional `--occurrence` defaulting to `1` |
| `diagram create-detail` | state preconditions, `--component-id`, `--title` |
| `diagram edit-title` | state preconditions, `--diagram-id`, `--title` |
| `diagram show-component` / `stop-showing-component` | state preconditions, `--diagram-id`, `--component-id` |

Here, state preconditions mean exact `--store-id`, `--change-set-id`, `--generation`. Read-only status/catalog/current/Accepted inspection take no action input. No `--generation none` exists. Parent-options/reassign require state and Diagram ID; reassignment adds anchor Component ID. Placement positions requires store/Diagram and optional Change Set ID; set-position requires state/Diagram/Component IDs and bounded integer `--x`/`--y` (negative example `--x=-180`); Auto-layout requires state/Diagram ID. Feedback and reconciliation exact flags and closed input arrays are specified in their owning contracts.

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

The CLI distinguishes a required selector flag which is present with an empty value from an omitted flag; MCP schemas likewise allow the exact empty string for old target/label selectors. Edit/remove selection applies to the exact current authored outgoing declarations returned by `change-set inspect`, including an incomplete row in an invalid pending set. The returned validation location and raw row values are sufficient to address it. Every invalid pending Relationship state repairable through the structured browser is therefore repairable through CLI and MCP without manufacturing a valid candidate, adding row identity, falling back to raw frontmatter, or forcing whole-set Discard.

Component creation accepts an explicit home Diagram ID; omitting it means there is genuinely no Diagram context and uses the root. Agent examples should pass it explicitly when the intended Diagram is known.

### Machine result envelope

Every JSON-mode command and every MCP tool uses the same logical envelope:

```json
{
  "protocol": "workbraid-agent-v2",
  "ok": true,
  "context": {
    "project": {"store_id": "...", "name": "...", "slug": "..."},
    "accepted_revision": "...",
    "authority_state": "current"
  },
  "result": {}
}
```

Project and accepted-revision context may be null when absent/unknown. Proposal state/generation belongs to the addressed result, never a process-wide pending context. Authorities report current, known non-current, or indeterminate knowledge truthfully. A domain failure has `ok: false`, the best truthful current context, and:

```json
{
  "error": {
    "code": "change_set_generation_mismatch",
    "message": "This proposal changed. Inspect it again before editing.",
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
| `architecture_non_current` | WorkBraid knows the loaded projection is not authoritative; Inspect authority and explicitly Refresh as appropriate; durable work remains retained. |
| `refresh_failed` | WorkBraid could not determine current accepted authority; preserve prior knowledge and retry explicitly. |
| `target_not_found` | A referenced Component or Diagram ID does not exist in the complete current candidate. |
| `target_not_eligible` | The target exists but the requested semantic action is not allowed in the current candidate. |
| `validation_blocked` | The complete proposal candidate is structurally invalid; use the returned locations and correct it. |
| `review_required` | No current successful review is bound to the proposal candidate; review first. |
| `review_invalidated` | The supplied or stored base/tree/generation binding is no longer current; inspect and review again. |
| `accepted_conflict` | Accepted authority changed before CAS; the addressed proposal remains preserved. |
| `acceptance_uncertain` | The server could not determine the final acceptance result; inspect/Refresh rather than retrying blindly. |
| `accepted_reload_required` | CAS definitely succeeded but in-process publication/reload failed; do not retry acceptance, inspect/Refresh/reopen the canonical result. |
| `unsupported_action` | The action is outside the current product or state; do not simulate it with Git/YAML. |
| `operation_failed` | A bounded internal operation failed without a more specific safe classification; inspect current state before retrying. |

HTTP status, CLI exit status, and MCP `isError` may aid transport handling, but this code is the shared domain classification.

Proposal codes are `change_set_not_found`, `change_set_unavailable`, `change_set_name_conflict`, `change_set_generation_mismatch`, `change_set_not_editable`, and `change_set_out_of_date`. Inspect returned identity/lifecycle/generation before correcting or retrying. Reconciliation also uses `change_set_state_mismatch`, `reconciliation_unresolved`, and `reconciliation_unsupported`; the owning contract closes its supported reason vocabulary. Feedback uses `review_submission_not_found`, `review_submission_unavailable`, `review_anchor_invalid`, `review_submission_not_allowed`, and `review_invalidated`. Do not classify English prose or repair private records.

## Embedded skill and MCP

`workbraid --skill` is standalone, non-mutating, server-independent embedded Markdown. It prints the complete document without banner/fence/logs and exits successfully. Its commands describe that binary, not future actions. Keep CLI help, skill, MCP descriptions/schemas and HTTP handlers consistent. Teach explicit IDs/generations, named proposals, exact text/file inputs, invalid-row repair, truthful authority, Review versus informational feedback, exact Update, reconciliation, placement and recovery. No online skill registry/installer or second instruction authority is needed.

MCP uses the official Go SDK, negotiates supported protocol versions and writes only protocol messages to stdout; diagnostics go to stderr. It exposes tools, not unnecessary resources/prompts/sampling/elicitation/tasks/subscriptions. Each tool has a precise discoverable description, closed input schema with exact identity/preconditions, shared-envelope object output schema, conforming structuredContent and equivalent serialized JSON text, and truthful read-only/destructive/idempotence annotations. Domain errors return `isError: true` with the structured envelope. The bridge uses the same local HTTP client as CLI, with no local Manager/store, snapshot/generation/review cache, selection, retry policy or acceptance convenience.

## Usability and scope

For changed agent-facing behavior, use bounded fresh discovery through CLI skill or MCP tools without source, private-store access, operation-name tutoring or a patched prompt to hide product confusion. Verify actual resulting refs/trees/bindings and fresh-process reconstruction independently. Preserve explicit human visual acceptance where the active feature changes interaction. Completed Agent Access gate matrices are historical, not mandatory full replays for every edit.

No extra vertical, daemon discovery/PID registry, remote exposure/authentication/multi-user model, per-client current project, autonomous background agents, polling/watchers, generic command bus/plugin framework, raw Git/YAML path, auto-review/accept convenience or renderer-owned authority is introduced.
