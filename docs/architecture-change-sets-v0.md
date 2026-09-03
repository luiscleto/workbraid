# Architecture Change Sets v0

Status: Proposed

Scope: durable named parallel Architecture proposals over the existing format-v2 product

This document defines the product and authority changes required by Architecture Change Sets 1. It supersedes the single anonymous, process-lifetime pending-set model in `docs/architecture-v0.md` and `docs/architecture-agent-access-v0.md`, plus the Accepted-only/one-pending workspace statements in `docs/ui-v0.md`, only where this document says so. Accepted Architecture, format v2, Diagram semantics, exact review, and accepted-ref CAS remain unchanged.

## 1. Product boundary

Each project has one singular Accepted Architecture and may have multiple independent named change sets.

A change set is a durable proposal for replacing Accepted Architecture. It owns:

- one immutable lower-case UUID;
- one mutable human-readable name;
- one exact Accepted commit captured as its base at creation;
- one independent non-negative generation;
- one concrete set of structured Architecture changes;
- one complete proposed Architecture tree whenever those changes validate;
- one exact Markdown proposal document; and
- at most one current review binding.

The UUID is identity. A name is selection and display text, never identity. Accepted remains the only canonical Architecture authority and continues to be named solely by `refs/heads/accepted`.

Change sets add no owner, assignee, label, priority, status workflow, approval requirement, or proposal branch visible as Architecture authority.

## 2. Names

Names are trimmed, non-empty, single-line human text. Active change-set names are unique among active change sets in one project. Creation, rename, and loading all compare names with Go's Unicode case-fold comparison (`strings.EqualFold`) and apply no additional Unicode normalization. Rename is supported for active change sets only.

Explicit creation may supply a name. Otherwise WorkBraid derives a friendly lower-case adjective-noun name from fixed word lists and UUID entropy, for example `quiet-harbor`. On collision with an active name it appends the first available numeric suffix, such as `quiet-harbor-2`. The exact words and entropy mapping are implementation details; active-name uniqueness within the project is required.

An active-name conflict is explicit. Applied records retain the exact names they had at acceptance but do not reserve those names. An active record may share a name with one or more applied records, and multiple applied records may share a historical name. The UI groups or marks applied records and adds shortened UUID context only where duplicate visible names require disambiguation. WorkBraid never selects or mutates a change set by name.

## 3. Durable private-Git representation

### 3.1 Ref namespaces

The existing private bare Git store gains two WorkBraid-owned ref namespaces:

```text
refs/workbraid/change-sets/active/<change-set-uuid>
refs/workbraid/change-sets/applied/<change-set-uuid>
```

The final ref component is the canonical lower-case UUID. Names do not appear in ref paths. A UUID may occur in exactly one of these namespaces.

Change Sets owns and enumerates exactly `refs/workbraid/change-sets/active/` and `refs/workbraid/change-sets/applied/`. Malformed, duplicate, conflicting, or unsupported state inside that owned subtree appears as an explicit unavailable/conflict entry; enumeration order never resolves it. A different namespace such as `refs/workbraid/reviews/`, `refs/workbraid/reconciliation/`, or `refs/workbraid/other-test/` is outside Change Sets interpretation and is ignored by its enumeration. This is a concrete namespace boundary, not a generic namespace registry. Accepted and unrelated valid records remain available.

These refs are durable proposal metadata. They are not alternate Accepted branches, do not make their trees canonical Architecture, and are never selected as `refs/heads/accepted` by a client.

### 3.2 State commit and envelope tree

Each change-set ref points to an ordinary WorkBraid-authored state commit. Its root tree is a closed operational envelope:

```text
change-set.yaml  100644
proposal.md      100644
changes.yaml     100644
architecture/    040000   # present exactly when the proposal is structurally valid
```

No other path is allowed. The `architecture/` entry points directly to the complete proposed format-v2 Architecture tree. That nested tree obeys the existing accepted Architecture closed-tree contract without proposal metadata. Its tree object ID is the exact `candidate_tree` used by review and acceptance.

An invalid or incomplete proposal has no `architecture/` entry and no review binding. `changes.yaml` remains sufficient to recover and repair its exact structured authored state. This is required because invalid structured relationship or Diagram work cannot truthfully be represented as a valid Architecture tree.

All WorkBraid-created envelope files use mode `100644`. Proposal state never enters an accepted Architecture tree and does not change the format-v2 schema.

### 3.3 `change-set.yaml`

`change-set.yaml` is UTF-8 YAML with this closed schema and exact key spelling:

```yaml
format: workbraid-change-set
version: 1
id: 11111111-1111-4111-8111-111111111111
name: quiet-harbor
base_revision: 0123456789abcdef0123456789abcdef01234567
generation: 3
review:
  base_revision: 0123456789abcdef0123456789abcdef01234567
  candidate_tree: 89abcdef0123456789abcdef0123456789abcdef
  generation: 3
```

`review` is optional. In an active valid record, when present, its base, candidate tree, and generation must exactly match the record's base, nested `architecture/` tree, and generation.

An applied record additionally requires:

```yaml
applied_revision: fedcba9876543210fedcba9876543210fedcba98
```

`applied_revision` is forbidden in an active record. It is required in an applied record. No arbitrary extension keys are accepted.

An active state commit has exactly one parent: `base_revision`. State commits do not chain prior generations and therefore do not create proposal history. An applied state commit has exactly one parent: `applied_revision`. Its metadata retains the original `base_revision`; the applied revision must be a successor whose parent is that base and whose root tree equals the envelope's nested `architecture/` tree.

The parent rules keep exact required commits reachable without turning proposal records into Architecture branches or a history product.

State-commit author, timestamp, and message are operational Git data and carry no change-set semantics. All authoritative proposal fields live in the validated envelope and ref namespace.

### 3.4 `changes.yaml`

`changes.yaml` is a closed, versioned serialization of the existing concrete pending facts. Its exact key spelling is:

```yaml
format: workbraid-change-state
version: 1
components:
  - id: 22222222-2222-4222-8222-222222222222
    path: components/worker.md
    new: false
    title: Worker
    description: |-
      Exact Markdown body.
    title_changed: false
    description_changed: true
    relationships_changed: true
    relationships:
      - target: ""
        label: ""
new_component_homes:
  - component_id: 33333333-3333-4333-8333-333333333333
    diagram_id: 44444444-4444-4444-8444-444444444444
detail_diagrams:
  - id: 55555555-5555-4555-8555-555555555555
    path: diagrams/storage.yaml
    title: Storage
    anchor_component_id: 22222222-2222-4222-8222-222222222222
diagram_titles:
  - diagram_id: 44444444-4444-4444-8444-444444444444
    title: Platform
home_moves:
  - component_id: 22222222-2222-4222-8222-222222222222
    diagram_id: 55555555-5555-4555-8555-555555555555
references:
  - diagram_id: 44444444-4444-4444-8444-444444444444
    component_id: 33333333-3333-4333-8333-333333333333
    present: true
```

All six sequences are required and may be empty. Their item schemas correspond exactly to the current concrete authoring values:

- `components`: Component ID, assigned path, `new`, exact Title and Description, the three changed flags, and ordered outgoing rows containing exact raw `target` and `label` strings;
- `new_component_homes`: Component ID and Diagram ID;
- `detail_diagrams`: generated Diagram ID, assigned path, exact title, and anchor Component ID;
- `diagram_titles`: Diagram ID and exact title;
- `home_moves`: Component ID and destination Diagram ID; and
- `references`: Diagram ID, Component ID, and final `present` boolean.

This is typed operational state, not generic commands, event history, raw YAML patches, or a second Architecture representation. It deliberately preserves incomplete relationship strings and other repairable authored values. The same single `ConstructCandidate` implementation rebuilds the candidate from these facts against the exact base.

For a valid record, restart recovery both loads the exact nested candidate tree through the existing format-v2 loader and reconstructs from `changes.yaml`; the two tree IDs must match. A mismatch makes that change set unavailable without affecting Accepted or other valid change sets. Validation messages, stale flags, rendered projections, unified diffs, and base snapshots are derived rather than persisted.

All scalar and item keys shown above are normative. Empty top-level sequences use `[]`. Unknown keys, missing required keys, duplicate logical change entries, invalid scalar types, and invalid identifiers make the change set unavailable. Repeated `relationships` rows are explicitly allowed because exact multiplicity and source order are existing Relationship semantics. Exact target and label strings in incomplete Relationship rows are deliberately allowed to be empty or invalid; candidate validation, rather than envelope parsing, classifies those authored values. The envelope must not introduce generic operation kinds or sentinel identities.

### 3.5 `proposal.md`

`proposal.md` stores exact UTF-8 Markdown bytes. It may be empty. WorkBraid neither interprets it as Architecture semantics nor infers Components, Relationships, or Diagram composition from it. Browser rendering follows the existing inert safe-Markdown resource policy.

Editing the proposal document is an ordinary change-set mutation. It survives restart but never appears in the complete canonical Architecture diff because it is not Accepted Architecture content.

### 3.6 Why this representation

The chosen state-commit envelope is smaller and more truthful than the rejected alternatives:

- a ref pointing only to a candidate tree cannot preserve invalid-but-repairable structured work, names, proposal Markdown, generations, or review bindings;
- one branch per proposal would imply history and branch semantics the product does not expose;
- state-commit chains would create a history/revert product unintentionally;
- worktrees would duplicate filesystem state and tempt clients to edit canonical YAML directly; and
- SQLite or a second registry would split durable proposal authority from the existing private Git store.

## 4. Mutation and synchronization

There remains one running Manager, one project store authority, one Accepted Architecture, one synchronization boundary, one candidate constructor/validator, and one Accepted CAS path.

The server replaces its one process-wide pending pointer with a project-scoped map of immutable change-set records keyed by UUID. This is a cache of private-Git authority, not a client-selected current change set.

Every operation concerning proposed work addresses an explicit change-set UUID. Under the existing concrete server lock it:

1. verifies the loaded project/store and current change-set ref object;
2. verifies the caller's exact generation;
3. changes only that record's concrete facts, name, or proposal Markdown;
4. rebuilds and validates through the one candidate path;
5. writes a new envelope tree and non-chained state commit whose sole parent remains the exact base;
6. CAS-updates only that active ref from its previously observed object ID; and
7. publishes the new immutable in-memory record.

A crash before ref CAS leaves the old durable generation. A crash after ref CAS reconstructs the new generation. Different change sets may serialize through the same lock; mutating A increments and invalidates the review of A only. It does not change B's ref, facts, generation, or review.

Successful semantic mutations, rename, and proposal-Markdown edits increment that set's generation and invalidate only its review. A genuine no-op does neither. Review capture durably adds or replaces the review block without incrementing generation. Review still atomically captures the exact base, candidate tree, generation, diff, and immutable snapshot comparison under the existing synchronization boundary.

Creation uses a create-only ref update and starts at generation `0`, with an empty proposal and a valid candidate equal to its base. An explicit creation is allowed to remain empty for later work. When a browser edit begins from Accepted without an explicit change set, the server atomically creates one and applies the requested mutation as its first durable generation; it must not expose an empty intermediate ref as a second operation.

Discard is deliberate whole-change-set deletion: it CAS-deletes the active ref at the exact generation. It changes no Accepted state or other change set. Unreferenced objects may be collected by ordinary Git maintenance. There is no partial discard or undo.

## 5. Accepted and proposal contexts

The browser's selected context is local view state, not backend identity:

- **Accepted** shows the exact currently loaded Accepted snapshot.
- An active named change set shows its exact complete proposed snapshot when valid.
- A retained applied record is read-only evidence and is not ordinary changes in progress.

A compact selector makes the current context unmistakable. A valid proposal switches Diagram tree, map, index, documentation, homes, references, boundaries, Relationships, title, and revision/binding context together. It is visually identified as proposed state but is not overlaid on Accepted. Review remains the deliberate bound **Before changes** / **With changes** comparison.

If the selected proposal is structurally invalid, WorkBraid shows its proposal Markdown, concrete authored facts, localized validation, and repair actions. It does not manufacture a partial Architecture map or substitute its base while calling that the proposal.

Switching contexts changes no Architecture or proposal state. Existing protection for unsent browser-local editor values still applies. Because proposals are durable and independently identified, selecting Accepted, another proposal, or another project no longer requires discarding backend-held work. A browser reload may safely return to Accepted; this stage does not require proposal selection to become URL state.

The former singular **Changes in progress** task becomes the task for the selected named active change set. It contains structured editing, proposal Markdown, validation, Review changes, rename, and deliberate delete. Applied records have no edit, rename, review, or reopen action.

## 6. Starting and editing changes

**New changes** creates a change set from the exact current Accepted revision, using an optional supplied name or generated name. Every creation request carries the exact observed store UUID and Accepted revision. Under the existing synchronization boundary, WorkBraid verifies that both still identify the current loaded Accepted state before generating an ID/name or writing any object/ref.

Browser implicit edit-start likewise carries the browser's observed store UUID and Accepted revision. The backend atomically verifies both, generates the change-set identity and name, constructs the requested first mutation, creates only the durable generation-`1` active ref, and publishes it. A stale or wrong-project request creates no empty record, object reachable from WorkBraid refs, or other state.

Every existing structured Component, Relationship, Diagram, home, and reusable-reference mutation operates against one explicit change-set UUID and expected generation. Candidate-relative identity and validation semantics stay unchanged.

CLI and MCP never rely on browser selection or an implicit current change set. Titles and names may aid display and discovery; UUIDs select mutations.

## 7. Review and acceptance

Review scope becomes:

```text
(change-set ID, exact base commit, exact candidate tree, generation)
```

The review response and visual workbench retain the exact unified Architecture diff and snapshot-unified structured comparison. Proposal Markdown is shown as proposal context, not inserted into the Architecture diff. A later edit to the proposal Markdown invalidates the review because the reviewed proposal context changed.

Invalid work cannot be reviewed. Valid work may be reviewed against its exact original base even when newer Accepted Architecture exists. The UI clearly identifies such a review and proposal as out of date.

Acceptance requires the exact durable current review binding and requires current Accepted to equal the change set's base. There is no force, accept-latest, auto-rebuild, merge, or automatic rebase.

Successful acceptance retains the accepted change set as a minimal applied read-only record:

1. re-observe that `refs/heads/accepted` equals the exact base;
2. create the normal successor commit with the candidate tree and base parent;
3. create the applied envelope commit, parented by that successor;
4. in one Git reference transaction, CAS-update Accepted, CAS-delete the active ref, and create the applied ref; and
5. publish the already-validated Accepted snapshot and applied record.

The atomic reference transaction is the acceptance boundary. If it fails, none of the three refs change; newly written objects remain unreferenced and non-canonical.

The applied ref is the durable acceptance receipt. A valid applied record that exactly matches the submitted change-set identity, original base, final generation, reviewed candidate tree, and `applied_revision` proves that acceptance succeeded. Response-loss recovery validates that receipt and reports the change set as already applied together with truthful current Accepted context. It does not retry or report uncertainty merely because `refs/heads/accepted` has subsequently advanced beyond `applied_revision`.

An applied record preserves the name, proposal Markdown, original base, final generation, exact candidate tree, review binding, and accepted successor. It does not introduce statuses or general proposal history. Accepting one change set never deletes, rewrites, renames, or invalidates another.

The smaller alternative is to delete the active ref and retain only the Accepted successor. That would discard the proposal Markdown immediately and leave no natural object for later exact-snapshot reviews. The chosen applied ref adds one immutable envelope and no editable lifecycle: ref namespace alone distinguishes active from applied. This is the minimum retention that preserves the proposal context without a workflow engine.

A proposal containing no Architecture tree change may be named and documented but cannot advance Accepted; Review explains that there is no Architecture change to accept.

## 8. Accepted advancement and out-of-date proposals

Given:

```text
Accepted R0
Change A base R0
Change B base R0
```

accepting A advances Accepted to R1 and moves A to its applied record. B remains byte-for-byte and identity-for-identity the same durable active proposal against R0. Its candidate, proposal Markdown, generation, and current review remain intact.

B remains selectable, inspectable, editable against its R0 proposal context, and reviewable against its exact R0 base. It is conspicuously **Out of date with Accepted**. It cannot be accepted while Accepted differs from its base. Editing B rebuilds from R0 plus B's own concrete state, not from R1.

Out-of-date is exact commit inequality, not an ancestry judgment. If authoritative Accepted later names B's exact base again, B is current-base again; WorkBraid invents no fast-forward, rewind, or branch-history policy.

This deliberately supersedes the old rule that a single stale anonymous pending set becomes read-only after accepted advancement. No automatic rebase, merge, reinterpretation, freeze, or discard occurs. Architecture Reconciliation 1 will later use the exact original base, current Accepted, and proposal candidate.

An external accepted-ref change remains invisible until explicit Refresh. Refresh adopts valid canonical reality as before but does not mutate change-set refs. If a valid current Accepted differs from a proposal base, the proposal becomes out of date but remains editable. If Accepted is missing, unsupported, invalid, or cannot be determined, existing proposal records remain inspectable but Architecture mutation/review/acceptance pauses under the existing non-current/indeterminate authority rules; WorkBraid does not claim that an unknown authority is merely a normal out-of-date base.

## 9. Agent Access v2

The one-pending assumptions in `workbraid-agent-v1` are intentionally incompatible with this model. Change Sets 1 introduces `workbraid-agent-v2`; the server rejects v1 protocol clients rather than guessing a proposal or silently applying old preconditions.

`architecture inspect` / `architecture_inspect` continues to mean Accepted Architecture. Change-set reads are explicit:

| Product action | CLI | MCP |
|---|---|---|
| List active and applied change sets | `change-set list` | `change_sets_list` |
| Create, optional custom name | `change-set create` | `change_set_create` |
| Inspect one proposal | `change-set inspect` | `change_set_inspect` |
| Rename active proposal | `change-set rename` | `change_set_rename` |
| Replace exact proposal Markdown | `change-set edit-proposal` | `change_set_edit_proposal` |
| Review one proposal | `change-set review` | `change_set_review` |
| Delete whole active proposal | `change-set discard` | `change_set_discard` |
| Accept exact reviewed proposal | `architecture update` | `architecture_update` |

Every change-set list, inspect, or read-modify request identifies the exact `store_id`; the server checks it against synchronized loaded-project state rather than relying on process-wide selection alone. All existing Architecture mutations retain their domain-oriented command/tool names and require exact `store_id`, `change_set_id`, and change-set `generation`. Change-set creation requires exact `store_id` and exact current `accepted_revision`. Acceptance requires `store_id`, `change_set_id`, exact base, candidate tree, and generation. CLI/MCP carry no select-current-change command or session state.

The v1 `changes inspect`, `changes review`, `changes discard`, global `pending_generation`, and implicit one-pending mutation preconditions are removed rather than retained as ambiguous aliases.

Machine results expose store/project identity, change-set ID/name/lifecycle, base, generation, proposal Markdown, exact concrete facts, validity, candidate tree/projection when valid, out-of-date state, review binding, and applied revision where relevant. Both transports share typed errors including:

- `change_set_not_found`;
- `change_set_unavailable`;
- `change_set_name_conflict`;
- `change_set_generation_mismatch`;
- `change_set_not_editable`;
- `change_set_out_of_date`;
- the existing target, validation, review-required, review-invalidated, Accepted-conflict, uncertain-acceptance, connection, and protocol classifications.

Agents never classify English strings. Invalid relationship rows retain the same exact raw-selector repair parity within the addressed change set.

The embedded `workbraid --skill`, CLI help/examples, MCP schemas/descriptions, and local protocol all move together to v2. They explain independent IDs and generations, durable proposal Markdown, explicit addressing, out-of-date editability, exact review/update, and the future reconciliation boundary. CLI and MCP remain thin clients of the one running WorkBraid authority and never open private Git directly.

## 10. Later-stage compatibility

Change Sets 1 deliberately preserves the exact inputs later stages need without designing those stages:

- Reviews can bind submissions and anchored comments to a change-set ID plus exact base, candidate tree, generation, and proposal/Architecture locations.
- Reconciliation can read original base, current Accepted, and exact proposal candidate without inferring old state from a mutable workspace.

No review comment, verdict, reconciliation result, conflict model, or new lifecycle is added here.

## 11. Deliberate exclusions

Change Sets 1 does not add:

- review comments, anchors, submissions, reviewers, or verdicts;
- approval gates or workflow statuses;
- merge, rebase, reconciliation, conflict resolution, or automatic Accepted updates;
- proposal history, undo/redo, partial discard, reopen, or applied-record deletion;
- labels, owners, assignees, priorities, or project-management fields;
- per-agent branches, Managers, pending copies, locks, or selected contexts;
- SQLite, worktrees, a proposal registry, generic draft database, generic operation algebra, event sourcing, or command bus;
- raw Git/YAML mutation tools;
- multi-user collaboration, remote access, permissions, or background agents;
- manual/persisted Diagram layout, routing, shapes, richer Diagram kinds, or any rich-diagram Phase 3 work.
