# Architecture Proposals and Reviews

Status: Approved living contract

## Printable bound proposals

Delegated root approval of proposal `6673c0da-c906-4c5a-bf62-bdd74a6ddfdf`, generation 2, reviewed state `5f72375846631f32fc3661a2be02be087d671dba`, independently approved in review `12235fb9-94d0-41cc-9c06-d9d9aef5a6cb`, authorizes a dedicated derived read-only printable page. It changes no format, persistence, lifecycle, candidate construction, review or acceptance authority. The planning proposal remains active Markdown-only; the combined human visual gate precedes Phase 4.

Under `/projects/:slug/proposals/:id`, active `/print?reviewed_state=:S` requires the current active state and its already prepared exact base/tree/generation binding. Applied `/print?applied_state=:T` requires the current applied state token and prints its retained original base/candidate/Markdown and receipt. Tokens are required and mutually exclusive. An old active S cannot identify applied T. `/reviews/:reviewId/print` follows only that immutable submission's validated parent and survives iteration, application and discard. Resolve the catalog slug to exact store identity; reads never open/select a project, prepare Review, mutate refs or perform arbitrary commit lookup. Invalid, moved and unavailable locators fail truthfully with a nonmutating proposal return link. Current Accepted authority/lifecycle is separate context and never replaces bound Before.

Reuse exact existing Review projections/comparison and source. Render design Markdown first, affected Diagrams second, per-Diagram concrete changes and source-text diffs third. The newer human-requested concise correction requires exactly one candidate-primary image per affected Diagram, retaining removed Relationship review annotations and explicitly base-only missing endpoints; absent candidate Diagrams use one labelled base-only image. No canonical appearance, layout or authority is created. Visible Component source or Relationship multiset changes affect a Diagram even without a Diagram blob delta. Default filtering retains content, title, composition, hierarchy and note text while excluding pure position, size, routing, shape and note geometry changes; explicit presentation opt-in restores those details without image duplication. Empty canonical diff means proposal text only and zero diagrams. A nonempty filtered or format-only diff remains truthfully identified as containing Architecture changes; its complete exact diff is inspectable in normal Review and optionally included through an explicit print control, off by default. Component source diffs appear once at With home (Before home if absent), cross-linked elsewhere. LF boundaries, CRLF, whitespace and absent final newline remain distinguishable; safe source diffs make no rendered-Markdown semantic claim. Notes are Diagram text, not submitted comments. UI owns browser-memory images and browser Print / Save as PDF; existing exact bindings and read-only lifetimes are unchanged.

## Phase 3.4 operational extension

Delegated root approval of proposal `3cf11d4b-c980-4022-92d2-f98d75bf36a4` generation 2, reviewed state `d9d35b11134533016248e5e64294462f76b3aeda`, authorizes operational v6. It retains v5 and requires `node_shapes` entries exactly `{diagram_id,component_id,shape}` and `diagram_notes` entries exactly `{diagram_id,note_id,note}`. All IDs are UUIDs; each address occurs at most once. Shape is `rectangle|ellipse|diamond` or internal null removal. Note is null deletion or exactly `{text,x,y,width,height}`, with [Placement §11](architecture-placement-amendment-v0.md#11-simple-shapes-and-diagram-notes-phase-34) validation, no repeated ID. Sort arrays by Diagram UUID then Component/note UUID. Reject unknown/missing/duplicate keys, invalid types and values. `architecture_version` permits 2..6, never below base; target<6 requires both arrays empty and target<5 also empty edge_routes. Earlier schemas stay closed.

Omission inherits; null prevents inherited resurrection. Shape null may name a visible pair or absent inherited pair. Non-null note facts require an existing owning Diagram and are ordinary complete final upserts, including explicitly resolved same-ID note restoration during reconciliation. No parent lifecycle or public creation ID is authorized. Proposal-new note deletion may normalize away creation. Only production authoring generates new note UUID/default geometry, once. One strict constructor replays final facts without allocation, pruning or repair. New proposals and actual mutations write v6, superseding earlier new-version statements; supported portable v2–v5/op v1–v5 candidate trees, bytes/modes and immutable review parents remain exact through GC/restart. Unchanged reads/Review preserve historical operational bytes.

Review reports shapes and notes separately from Component content, Relationships and submitted feedback. Removed notes appear on their exact Before side and as removed facts, without fabricated Component ghosts. Existing review anchors remain unchanged; note discussion may use its Diagram anchor. Ref/CAS, envelopes, S/B/A/P and deliberate Review/Update authority are unchanged.

## Phase 3.3 operational extension

Under delegated root approval of routing proposal `d119baeb-1b2b-4451-8113-7f4fe05678ec` generation 8, reviewed state `595bab6de6933c7a047c70520f1830594f166fed`, new proposals and real mutations write closed operational **version 5**. All v4 fields retain their exact schema and a required `edge_routes` sequence is added. Each entry requires exactly `diagram_id`, `source_id`, `target_id`, `label`, `occurrence`, `route`. IDs are Component/Diagram UUIDs in their stated roles; label is exact valid UTF-8; occurrence is an integer >=1. Route is null or exactly `{bend}` with an integer in −100000..100000. Duplicate addresses/keys, unknown/missing fields, invalid types and unsupported targets are rejected. There is at most one fact per complete address. Sort final facts by Diagram UUID, source UUID, target UUID, exact UTF-8 label bytes, then numeric occurrence.

`architecture_version` accepts 2, 3, 4 or 5, cannot be below base, and a target below 5 requires empty `edge_routes`. Non-null routes must resolve to eligible final slots. Null is explicit no-custom-route and may target visible slots or absent inherited slots; omission inherits. The full reset/non-resurrection and invalid-retention rules are normative in [Placement §10](architecture-placement-amendment-v0.md#10-deliberate-link-routing-phase-33). The ordinary mutation path materializes final resets/upgrades; read/Review/replay never initializes, prunes or infers events.

This supersedes new-v4-write statements below. Portable v2–v4 and operational v1–v4 remain closed, unchanged and exactly reconstructible. Loading and unchanged Review reuse original changes.yaml; repeated Review remains an exact state/ref no-op. Preserve exact candidates, entries/modes and immutable review parents through later authoring, reconciliation, application, discard, GC and restart. Every valid supported record still satisfies `ConstructCandidate(base, facts).Tree == candidate_tree`. Envelope/Review versions, refs, one-parent rules, informational verdicts and separate exact Review/Update authority do not change. Historical Relationship comments retain their original side-owned tuple/occurrence anchors. Routing changes are separate Diagram presentation review facts, never inferred Relationship survivor identities. Markdown-only plan approval does not make a proposal Applied.

## Phase 3.2 operational extension

New proposals and real mutations now write closed operational **version 4**. It retains every v3 field and requires `node_sizes: []`, with closed `{diagram_id, component_id, size}` final facts, at most one per pair. A non-null size is exactly `{width, height}` with the bounds in [Placement §9](architecture-placement-amendment-v0.md#9-complete-visible-node-sizing-phase-32). Null means internal disappearance/non-resurrection only; no visible v4 pair may remain unsized. Missing overrides inherit base sizes subject to final visibility. `architecture_version` accepts 2, 3 or 4, cannot be below base, and targets below 4 require empty node_sizes. Envelope, review schemas and ref namespaces do not change.

This supersedes statements below about new operational-v3 writes. All supported portable v2/v3 and operational v1–3 history must reconstruct exact trees, entries/modes and immutable review parents through mutation, reconciliation, application, discard, GC and restart. Reads and first unchanged Review preserve original changes.yaml; repeated Review is an exact state/ref no-op. Same-displayed-size requests preserve even legacy operational versions before any upgrade. Strict reconstruction never runs initialization or sizing algorithms. No compatibility waiver applies to supported history.

When the canonical diff is empty, hide the empty Complete change / Raw unified diff box and say exactly “No Architecture changes; this proposal contains only proposal text.” Keep proposal Markdown and informational feedback available. This adds no plan-only acceptance or Applied transition.

Named Change Sets and immutable submitted Reviews share one project authority but own separate namespaces and rules. [Architecture](architecture-v0.md) defines portable state/source fidelity; [Placement](architecture-placement-amendment-v0.md) extends v3 facts; [Reconciliation](architecture-reconciliation-v0.md) changes proposal bases deliberately; [Agent Access](architecture-agent-access-v0.md) defines the shared clients.

## Change Sets

### Product boundary

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

### Names

Names are trimmed, non-empty, single-line human text. Active change-set names are unique among active change sets in one project. Creation, rename, and loading all compare names with Go's Unicode case-fold comparison (`strings.EqualFold`) and apply no additional Unicode normalization. Rename is supported for active change sets only.

Explicit creation may supply a name. Otherwise WorkBraid derives a friendly lower-case adjective-noun name from fixed word lists and UUID entropy, for example `quiet-harbor`. On collision with an active name it appends the first available numeric suffix, such as `quiet-harbor-2`. The exact words and entropy mapping are implementation details; active-name uniqueness within the project is required.

An active-name conflict is explicit. Applied records retain the exact names they had at acceptance but do not reserve those names. An active record may share a name with one or more applied records, and multiple applied records may share a historical name. The UI groups or marks applied records and adds shortened UUID context only where duplicate visible names require disambiguation. WorkBraid never selects or mutates a change set by name.

### Durable private-Git representation

#### Ref namespaces

The existing private bare Git store gains two WorkBraid-owned ref namespaces:

```text
refs/workbraid/change-sets/active/<change-set-uuid>
refs/workbraid/change-sets/applied/<change-set-uuid>
```

The final ref component is the canonical lower-case UUID. Names do not appear in ref paths. A UUID may occur in exactly one of these namespaces.

Change Sets owns and enumerates exactly `refs/workbraid/change-sets/active/` and `refs/workbraid/change-sets/applied/`. Malformed, duplicate, conflicting, or unsupported state inside that owned subtree appears as an explicit unavailable/conflict entry; enumeration order never resolves it. A different namespace such as `refs/workbraid/reviews/`, `refs/workbraid/reconciliation/`, or `refs/workbraid/other-test/` is outside Change Sets interpretation and is ignored by its enumeration. This is a concrete namespace boundary, not a generic namespace registry. Accepted and unrelated valid records remain available.

These refs are durable proposal metadata. They are not alternate Accepted branches, do not make their trees canonical Architecture, and are never selected as `refs/heads/accepted` by a client.

#### State commit and envelope tree

Each change-set ref points to an ordinary WorkBraid-authored state commit. Its root tree is a closed operational envelope:

```text
change-set.yaml  100644
proposal.md      100644
changes.yaml     100644
architecture/    040000   # present exactly when the proposal is structurally valid
```

No other path is allowed. The `architecture/` entry points directly to the complete proposed supported Architecture tree. That nested tree obeys the existing accepted Architecture closed-tree contract without proposal metadata. Its tree object ID is the exact `candidate_tree` used by review and acceptance.

An invalid or incomplete proposal has no `architecture/` entry and no review binding. `changes.yaml` remains sufficient to recover and repair its exact structured authored state. This is required because invalid structured relationship or Diagram work cannot truthfully be represented as a valid Architecture tree.

All WorkBraid-created envelope files use mode `100644`. Proposal state never enters an accepted Architecture tree and does not enter the portable Architecture schema.

#### `change-set.yaml`

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

#### `changes.yaml`

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

For a valid record, restart recovery both loads the exact nested candidate tree through the one version-aware Architecture loader and reconstructs from `changes.yaml`; the two tree IDs must match. A mismatch makes that change set unavailable without affecting Accepted or other valid change sets. Validation messages, stale flags, rendered projections, unified diffs, and base snapshots are derived rather than persisted.

All scalar and item keys shown above are normative. Empty top-level sequences use `[]`. Unknown keys, missing required keys, duplicate logical change entries, invalid scalar types, and invalid identifiers make the change set unavailable. Repeated `relationships` rows are explicitly allowed because exact multiplicity and source order are existing Relationship semantics. Exact target and label strings in incomplete Relationship rows are deliberately allowed to be empty or invalid; candidate validation, rather than envelope parsing, classifies those authored values. The envelope must not introduce generic operation kinds or sentinel identities.

#### Operational versions and supported history

The six-sequence example above is the supported operational **version 1** schema over portable v2. Operational **version 2** adds required `detail_reassignments: []`, with closed `{diagram_id, anchor_component_id}` entries as defined by [Reconciliation](architecture-reconciliation-v0.md#closed-operational-schema-evolution). All other fields retain their exact spelling, types and source semantics.

Operational **version 3** retains the seven sequences and requires `architecture_version` (integer 2 or 3) and `node_positions` (sequence of closed `{diagram_id, component_id, position}` values). Target 2 requires empty node_positions; target 3 follows the [complete-placement contract](architecture-placement-amendment-v0.md). Target version cannot be below the base. A pair is exact bounded integer `{x,y}`; null is only the permitted internal absence/non-resurrection fact. Missing a pair means inheritance subject to final visibility, not a reset.

New proposals and real mutations write operational v3. Loading supported v1/v2 records must preserve exact schemas and candidate trees; do not serialize new default fields on read. Repeated exact Review is a state/ref no-op. First review preparation of an old unchanged record may write its binding but reuses the unchanged `changes.yaml` entry. Preserve currently supported v2/op-v1/v2 historical candidates and submitted reviews through later mutation, application, discard, GC and restart. Already-unavailable pre-fix alpha records need not be revived; that does not waive preservation of supported history. The failed partial-placement v3 trial is separately excluded by the approved correction.

#### `proposal.md`

`proposal.md` stores exact UTF-8 Markdown bytes. It may be empty. WorkBraid neither interprets it as Architecture semantics nor infers Components, Relationships, or Diagram composition from it. Browser rendering follows the existing inert safe-Markdown resource policy.

Editing the proposal document is an ordinary change-set mutation. It survives restart but never appears in the complete canonical Architecture diff because it is not Accepted Architecture content.

### Mutation and synchronization

There remains one running Manager, one project store authority, one Accepted Architecture, one synchronization boundary, one candidate constructor/validator, and one Accepted CAS path.

The server caches project-scoped immutable change-set records keyed by UUID. This is a cache of private-Git authority, not a client-selected current change set.

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

### Accepted and proposal contexts

The browser's selected context is local view state, not backend identity:

- **Accepted** shows the exact currently loaded Accepted snapshot.
- An active named change set shows its exact complete proposed snapshot when valid.
- A retained applied record is read-only evidence and is not ordinary changes in progress.

A compact selector makes the current context unmistakable. A valid proposal switches Diagram tree, map, index, documentation, homes, references, boundaries, Relationships, title, and revision/binding context together. It is visually identified as proposed state but is not overlaid on Accepted. Review remains the deliberate bound **Before changes** / **With changes** comparison.

If the selected proposal is structurally invalid, WorkBraid shows its proposal Markdown, concrete authored facts, localized validation, and repair actions. It does not manufacture a partial Architecture map or substitute its base while calling that the proposal.

Switching contexts changes no Architecture or proposal state. Existing protection for unsent browser-local editor values still applies. Because proposals are durable and independently identified, selecting Accepted, another proposal, or another project no longer requires discarding backend-held work. A browser reload may safely return to Accepted; this stage does not require proposal selection to become URL state.

The former singular **Changes in progress** task becomes the task for the selected named active change set. It contains structured editing, proposal Markdown, validation, Review changes, rename, and deliberate delete. Applied records have no edit, rename, review, or reopen action.

### Starting and editing changes

**New changes** creates a change set from the exact current Accepted revision, using an optional supplied name or generated name. Every creation request carries the exact observed store UUID and Accepted revision. Under the existing synchronization boundary, WorkBraid verifies that both still identify the current loaded Accepted state before generating an ID/name or writing any object/ref.

Browser implicit edit-start likewise carries the browser's observed store UUID and Accepted revision. The backend atomically verifies both, generates the change-set identity and name, constructs the requested first mutation, creates only the durable generation-`1` active ref, and publishes it. A stale or wrong-project request creates no empty record, object reachable from WorkBraid refs, or other state.

Every existing structured Component, Relationship, Diagram, home, and reusable-reference mutation operates against one explicit change-set UUID and expected generation. Candidate-relative identity and validation semantics stay unchanged.

CLI and MCP never rely on browser selection or an implicit current change set. Titles and names may aid display and discovery; UUIDs select mutations.

### Review and acceptance

Review scope becomes:

```text
(change-set ID, exact base commit, exact candidate tree, generation)
```

The review response and visual workbench retain the exact unified Architecture diff and snapshot-unified structured comparison. Proposal Markdown is shown as proposal context, not inserted into the Architecture diff. A later edit to the proposal Markdown invalidates the review because the reviewed proposal context changed.

Invalid work cannot be reviewed. Valid work may be reviewed against its exact original base even when newer Accepted Architecture exists. The UI clearly identifies such a review and proposal as out of date.

Acceptance requires the exact durable current review binding and requires current Accepted to equal the change set's base. There is no force, accept-latest, auto-rebuild or automatic rebase. Deliberate reconciliation is separate and never accepts.

Successful acceptance retains the accepted change set as a minimal applied read-only record:

1. re-observe that `refs/heads/accepted` equals the exact base;
2. create the normal successor commit with the candidate tree and base parent;
3. create the applied envelope commit, parented by that successor;
4. in one Git reference transaction, CAS-update Accepted, CAS-delete the active ref, and create the applied ref; and
5. publish the already-validated Accepted snapshot and applied record.

The atomic reference transaction is the acceptance boundary. If it fails, none of the three refs change; newly written objects remain unreferenced and non-canonical.

The applied ref is the durable acceptance receipt. A valid applied record that exactly matches the submitted change-set identity, original base, final generation, reviewed candidate tree, and `applied_revision` proves that acceptance succeeded. Response-loss recovery validates that receipt and reports the change set as already applied together with truthful current Accepted context. It does not retry or report uncertainty merely because `refs/heads/accepted` has subsequently advanced beyond `applied_revision`.

An applied record preserves the name, proposal Markdown, original base, final generation, exact candidate tree, review binding, and accepted successor. It does not introduce statuses or general proposal history. Accepting one change set never deletes, rewrites, renames, or invalidates another.

Ref namespace alone distinguishes active from applied. The applied receipt preserves proposal context without an editable lifecycle.

A proposal containing no Architecture tree change may be named and documented but cannot advance Accepted; Review explains that there is no Architecture change to accept.

### Accepted advancement and out-of-date proposals

Given:

```text
Accepted R0
Change A base R0
Change B base R0
```

accepting A advances Accepted to R1 and moves A to its applied record. B remains byte-for-byte and identity-for-identity the same durable active proposal against R0. Its candidate, proposal Markdown, generation, and current review remain intact.

B remains selectable, inspectable, editable against its R0 proposal context, and reviewable against its exact R0 base. It is conspicuously **Out of date with Accepted**. It cannot be accepted while Accepted differs from its base. Editing B rebuilds from R0 plus B's own concrete state, not from R1.

Out-of-date is exact commit inequality, not an ancestry judgment. If authoritative Accepted later names B's exact base again, B is current-base again; WorkBraid invents no fast-forward, rewind, or branch-history policy.

No automatic rebase, merge, reinterpretation, freeze, or discard occurs. Deliberate [Reconciliation](architecture-reconciliation-v0.md) uses exact original base, current Accepted and proposal candidate.

An external accepted-ref change remains invisible until explicit Refresh. Refresh adopts valid canonical reality as before but does not mutate change-set refs. If a valid current Accepted differs from a proposal base, the proposal becomes out of date but remains editable. If Accepted is missing, unsupported, invalid, or cannot be determined, existing proposal records remain inspectable but Architecture mutation/review/acceptance pauses under the existing non-current/indeterminate authority rules; WorkBraid does not claim that an unknown authority is merely a normal out-of-date base.


## Submitted Reviews

### Product boundary

`Review changes` remains the existing operation which validates one active Change Set and durably binds its exact:

- Change Set UUID;
- base commit;
- candidate tree; and
- generation.

It continues to produce the exact Before/With projections and complete canonical diff used by deliberate acceptance. Agent Access `change-set review` / `change_set_review` retains that meaning.

An immutable **review submission** records feedback separately on one already-bound exact Change Set state. It never constructs a candidate, changes a proposal, advances Accepted, or participates in acceptance authority.

Each submission owns:

- one immutable lower-case review UUID;
- the exact Change Set UUID and exact reviewed Change Set state commit;
- the exact base commit, candidate tree, and generation binding stored in that state;
- one verdict: `comment`, `approve`, or `request_changes`;
- one optional exact UTF-8 Markdown overall body;
- zero or more immutable anchored comments; and
- minimal descriptive, untrusted author provenance.

A submitted review is immutable. Further feedback is another submission. There is no draft-review persistence, edit, deletion, thread, reply, resolution, reaction, latest/superseded state, approval count, required reviewer, or review workflow.

Verdicts are informational conclusions only. They do not gate editing, `Review changes`, acceptance, discard, or any other Architecture transition. WorkBraid computes no aggregate approval state. A person may deliberately accept a still-current exactly reviewed proposal regardless of any submitted verdict.

### Identity and descriptive provenance

Review and comment UUIDs are generated exactly once by WorkBraid when submission begins. They are identity; author labels, timestamps, verdicts, bodies, and anchors are not identity.

The author model is deliberately small:

- `author` is a required, trimmed, non-empty, single-line display label supplied by the submitting client; and
- `submitted_at` is the running WorkBraid process's UTC RFC 3339 timestamp at submission.

The normalized author label is stored exactly after trimming. The timestamp and label are descriptive provenance, not authenticated identity, trusted ordering, authorship proof, or authorization. UI and agent documentation must not call a label a verified user or imply that WorkBraid knows who controlled the browser, CLI, or MCP client. Review lists order by `submitted_at` newest first with review UUID as a deterministic tie-breaker; that order has no review-authority meaning.

An anchored comment body must be valid UTF-8 and non-empty after trimming; its submitted bytes are otherwise retained exactly. The overall review body may be empty.

For verdict `comment`, at least one of the overall body being non-empty after trimming or one anchored comment is required. A completely empty `comment` submission is `invalid_request` and creates no review ref. Verdict-only `approve` and `request_changes` submissions remain valid because either verdict itself communicates a conclusion. Markdown bytes which pass these presence checks are retained exactly.

### Durable private-Git representation

#### Owned ref namespace

Reviews owns exactly:

```text
refs/workbraid/reviews/<change-set-uuid>/<review-uuid>
```

Both path identities are canonical lower-case UUIDs. A review UUID is unique across this Reviews namespace in one store. The ref path, record metadata, and exact reviewed Change Set state must agree on the Change Set identity.

Malformed, duplicated, conflicting, or unsupported records inside `refs/workbraid/reviews/` appear as explicit unavailable review entries. They do not make Accepted, the owning Change Set, or other valid reviews unavailable. Refs in `refs/workbraid/change-sets/`, `refs/workbraid/reconciliation/`, or any other `refs/workbraid/*` subtree are outside Reviews interpretation. No generic namespace registry is introduced.

#### Review commit and reachability

Each review ref points to one ordinary WorkBraid-authored review commit. That commit has exactly one parent: the exact reviewed Change Set state commit named by the active Change Set ref after `Review changes` durably recorded the matching binding.

The parent link is required. Recording only the state object ID in a blob would not keep a non-chained Change Set generation reachable through ordinary Git garbage collection. The review ref reaches:

```text
review ref
  -> review commit
     -> exact reviewed Change Set state commit
        -> exact base commit
```

The reviewed state commit's closed Change Set envelope retains `proposal.md`, `changes.yaml`, its nested candidate Architecture tree, and the exact review binding. Existing Change Set parsing, `ConstructCandidate`, candidate-tree equality, and version-aware Architecture loading validate that parent. Reviews adds no second proposal or Architecture interpretation.

Review commits are independent records, not a chain. One review commit is not the parent of another and review refs are not branches in the product model. Review commit author, timestamp, and message are operational Git data and carry no review semantics.

#### Closed review tree

The review commit root is a closed operational tree:

```text
review.yaml                         100644
body.md                             100644
comments/                           040000  # omitted when there are no comments
comments/<comment-uuid>.yaml        100644
comments/<comment-uuid>.md          100644
```

Every ID listed by `review.yaml` has exactly one matching YAML/Markdown pair. The comments tree contains no unlisted or unmatched path. Every WorkBraid-created blob uses mode `100644`. `body.md` is always present and contains the exact overall UTF-8 Markdown bytes, including an empty document. Each comment Markdown blob contains that comment's exact body.

Review metadata never enters the Change Set envelope's nested `architecture/` tree or Accepted Architecture. Review bodies and comments are not Architecture semantics.

#### `review.yaml`

`review.yaml` is UTF-8 YAML with this closed schema and exact key spelling:

```yaml
format: workbraid-review
version: 1
id: 11111111-1111-4111-8111-111111111111
change_set_id: 22222222-2222-4222-8222-222222222222
reviewed_state: 0123456789abcdef0123456789abcdef01234567
binding:
  base_revision: 123456789abcdef0123456789abcdef012345678
  candidate_tree: 23456789abcdef0123456789abcdef0123456789
  generation: 4
verdict: request_changes
author: Architecture reviewer
submitted_at: "2026-09-04T14:30:00Z"
comments:
  - 33333333-3333-4333-8333-333333333333
```

`comments` is required and may be `[]`. Its order preserves the reviewer's submitted presentation order only; it creates no priority, thread, or lifecycle semantics.

`reviewed_state` must equal the review commit's sole parent. That parent must parse under the non-applied/active Change Set state-commit schema. Its stored Change Set identity must equal `change_set_id`, its stored review block must exactly equal `binding`, and its nested candidate tree must exist and validate. Parent validation does not require a live active ref, a current generation, or a still-active Change Set. Unknown or missing keys, invalid identifiers, an unsupported format/version/verdict, duplicate comment IDs, parent mismatch, or binding mismatch makes the review unavailable.

#### Comment metadata and anchors

Each `comments/<comment-uuid>.yaml` is a closed record:

```yaml
format: workbraid-review-comment
version: 1
id: 33333333-3333-4333-8333-333333333333
anchor:
  kind: component_markdown
  side: with_changes
  component_id: 44444444-4444-4444-8444-444444444444
  start_line: 8
  end_line: 10
```

Exactly one of these anchor shapes is allowed:

```yaml
# Whole proposal and its exact reviewed context. No side.
anchor:
  kind: proposal

# proposal.md source. No Before side exists.
anchor:
  kind: proposal_markdown
  start_line: 2
  end_line: 4

# One Component in an exact Architecture side.
anchor:
  kind: component
  side: with_changes
  component_id: 44444444-4444-4444-8444-444444444444

# The canonical Markdown section of one Component, excluding frontmatter.
anchor:
  kind: component_markdown
  side: before
  component_id: 44444444-4444-4444-8444-444444444444
  start_line: 5
  end_line: 5

# One Diagram.
anchor:
  kind: diagram
  side: with_changes
  diagram_id: 55555555-5555-4555-8555-555555555555

# One canonical composition fact.
anchor:
  kind: composition
  side: with_changes
  diagram_id: 55555555-5555-4555-8555-555555555555
  component_id: 44444444-4444-4444-8444-444444444444
  aspect: reference

# One parent-owned detail link. detail_diagram_id is required only here.
anchor:
  kind: composition
  side: with_changes
  diagram_id: 55555555-5555-4555-8555-555555555555
  component_id: 44444444-4444-4444-8444-444444444444
  aspect: detail
  detail_diagram_id: 66666666-6666-4666-8666-666666666666

# One exact global Relationship fact occurrence.
anchor:
  kind: relationship
  side: before
  source_component_id: 44444444-4444-4444-8444-444444444444
  target_component_id: 77777777-7777-4777-8777-777777777777
  label: calls
  occurrence: 2
```

`side` accepts only `before` or `with_changes`. It is required for every Architecture anchor and forbidden for proposal/proposal-Markdown anchors. A composition `aspect` is exactly `home`, `reference`, or `detail`. `detail_diagram_id` is required for `detail` and forbidden otherwise. A home anchor identifies the home appearance fact even when it also owns a separately anchorable detail link.

A Relationship occurrence is one-based among identical `(source Component ID, target Component ID, exact label)` facts in source order on the selected snapshot side. It introduces no Relationship identity or ordering semantics. A removed fact can be anchored on `before`; an added fact can be anchored on `with_changes`.

Derived Lives in boundary nodes are not canonical composition facts and cannot be composition anchors. A reviewer may anchor the real crossing Relationship or the external Component instead. No boundary, appearance, or Relationship ID is introduced.

#### Exact line semantics

Line anchors use one-based inclusive `start_line` and `end_line`, with `start_line <= end_line`. Lines are delimited by the LF byte in the exact reviewed UTF-8 source. A final LF terminates the preceding line and does not create an additional addressable empty line; empty source has zero addressable lines. CR in CRLF remains part of the line bytes.

For `proposal_markdown`, the source is the exact reviewed Change Set state's `proposal.md` blob. For `component_markdown`, it is every exact byte after that side's Component frontmatter closing-delimiter line: any optional whitespace before the H1, the H1, and the body through end of file. The delimiter's line ending belongs to frontmatter and is not included. WorkBraid may render the Markdown safely, but line validation and display must retain those exact source bytes and line boundaries.

Reviews 1 does not anchor rendered DOM nodes, CommonMark AST nodes, unified-diff line numbers, Diagram YAML, Component frontmatter, or inferred semantic text.

#### Submission-time validation

Every anchor is resolved once against the exact immutable reviewed parent state:

- proposal lines must exist in its exact `proposal.md`;
- a Component or Diagram UUID must resolve on the selected Before/With side;
- Component Markdown lines must exist in that side's exact canonical Markdown section;
- a composition home/reference/detail fact must exist exactly in the selected Diagram and side;
- a detail anchor's child ID must equal the exact parent-owned link;
- a Relationship tuple and occurrence must resolve exactly on that side; and
- fields forbidden for the selected anchor kind must be absent.

Failure rejects the whole review submission and creates no review ref. Submitted anchors never float, follow renames, migrate to later lines, use fuzzy matching, or claim continued applicability after proposal iteration.

### Submission authority and atomicity

Review submission is allowed only for an active Change Set with a valid current `Review changes` binding. Applied records are readable but accept no new submissions in Reviews 1; feedback after application has no demonstrated iteration workflow.

Preparing that binding is idempotent at its durable boundary. If an active Change Set already stores the exact review binding for its current base revision, candidate tree, and generation, repeated browser **Review changes**, CLI `change-set review`, or MCP `change_set_review` returns the same active Change Set state object as `reviewed_state`. It performs no semantically identical state-commit write and no active-ref CAS. If no exact binding exists, `Review changes` may establish it normally. A real Architecture mutation, Change Set rename, or proposal-Markdown edit still increments that Change Set's generation, removes its review binding, and writes a new state normally.

This durable no-op lets multiple reviewers share one exact state safely: preparing the same unchanged review cannot invalidate an earlier reviewer's `reviewed_state`. It introduces no reviewer session, lock, lease, collaboration state, or persistent draft.

The browser, CLI, or MCP submission carries exact:

- `store_id`;
- `change_set_id`;
- `reviewed_state` returned by the successful `Review changes` response;
- `base_revision`;
- `candidate_tree`; and
- `generation`.

Under the existing Manager synchronization boundary the server checks the loaded store, current active record/ref object, valid current review, exact binding, and every anchor. It captures the immutable material needed for the review, writes the review blobs/tree/commit, then uses one fixed Git reference transaction equivalent to:

```text
verify refs/workbraid/change-sets/active/<change-set-uuid> <reviewed-state>
create refs/workbraid/reviews/<change-set-uuid>/<review-uuid> <review-commit>
```

The create is from the zero object. Both commands succeed or neither ref change occurs. This final Git verification prevents another process or late Change Set mutation/acceptance/discard from attaching feedback to a state other than the one inspected. Same-process review submission and Change Set operations serialize through the existing lock; no review transaction framework is added.

The `change_set_review` result gains the exact `reviewed_state` commit ID after its review binding has been durably written. Adding that machine field does not change the meaning of review or its base/tree/generation acceptance binding.

Submitting feedback does not mutate the Change Set ref, generation, current review binding, candidate, or Accepted. A failed/raced submission leaves the proposal and every existing submission unchanged. Objects written before a failed ref transaction are non-canonical and may be collected.

If the Change Set changes between review inspection and submission, the request returns `review_invalidated`; the caller inspects and reviews the new generation rather than silently retargeting feedback. A valid out-of-date active proposal may still receive a submission against its exact original-base review, while the result truthfully reports that the proposal is out of date with current Accepted.

Submitting feedback against an already durable exact review binding does not require Accepted to be known-current. Known-non-current or indeterminate Accepted authority is returned truthfully as context, but does not prevent an otherwise exact immutable submission. When current Accepted cannot be determined, WorkBraid does not claim that the proposal is current or out of date. This relaxation applies only to review submission: Change Set mutation, preparation of a new `Review changes` binding, reconciliation, and acceptance retain their existing Accepted-authority rules.

### Later proposal and lifecycle changes

When a reviewed Change Set advances, every prior review ref and parent remains unchanged. List/inspect derives and reports:

- exact reviewed base, candidate tree, generation, and state commit;
- whether that generation still equals the current Change Set generation and binding;
- whether the Change Set is active or applied;
- whether an active Change Set is currently out of date with Accepted; and
- current Accepted revision and authority knowledge as context, never as a replacement for the review's Before side.

List/inspect always reconstructs an immutable review from its review ref and exact parent first. It then separately derives current lifecycle and Accepted context if an active/applied Change Set and authoritative Accepted observation exist. An earlier review's proposal Markdown, Before/With Architecture, comments, and anchors are never overlaid onto a newer proposal generation. Reviews 1 does not decide whether old feedback still applies.

Acceptance retains its existing three-ref Change Set transaction and does not modify review refs. Reviews remain readable when the Change Set becomes Applied. The applied receipt and review records are separate durable facts; neither becomes an approval gate or aggregate state.

#### Discarded Change Sets

Change Sets 1 permits deliberate deletion of an active proposal, while Reviews 1 requires submitted records to be immutable and durable. The approved rule is:

- Change Set discard keeps its existing meaning and deletes only the exact active Change Set ref;
- it does not delete refs owned by Reviews;
- existing review refs continue to retain their exact reviewed state commits;
- no new review can be submitted because no active current review exists;
- exact review list/inspect and a direct submitted-review browser URL remain usable when supplied the Change Set/review UUIDs;
- the normal proposal selector does not invent a discarded proposal record or lifecycle; and
- the review view says only that the proposal is no longer active.

This preserves both namespace ownership and permanent submitted feedback without reopening Change Sets into an archive/workflow product. WorkBraid does not cascade-delete submitted reviews, block Change Set discard because reviews exist, create an archived/discarded Change Set lifecycle, or reopen discarded proposals.

### Browser review workspace

The stable current-review route remains:

```text
/projects/<slug>/proposals/<change-set-uuid>/review
```

It continues to show the exact current bound Before/With review and canonical diff. Reviews 1 adds contextual annotation affordances and one compact review-composition area in that task, not a separate dashboard or permanent management panel.

The composer supports:

- a self-described author label;
- optional overall Markdown;
- accumulated local anchored comments;
- one verdict; and
- deliberate **Submit review**.

Before submission, comments may be added, adjusted, or removed as transient browser-local composition. Existing dirty-navigation protection covers non-empty unsent review composition when leaving the review, switching proposal/project context, or opening an earlier submitted review. Leaving without submission drops only those local values. There is no server-side review draft or autosave.

Contextual **Add comment** actions operate on product concepts:

- the proposal as a whole;
- selected proposal-Markdown source lines;
- the selected Component or exact Component-Markdown source lines;
- the selected Diagram;
- a selected home/reference/detail composition fact; and
- a selected exact Relationship occurrence.

Line-addressable views expose only exact proposal Markdown or the Component's canonical Markdown section. Raw Diagram YAML and Component frontmatter never become ordinary review UI. All Markdown uses the approved inert safe renderer and resource-request protections.

Submitted reviews appear in the proposal/review context with verdict, author label, submitted time, binding/generation, and current/earlier/applied/out-of-date context. Inspecting one uses a stable exact route:

```text
/projects/<slug>/proposals/<change-set-uuid>/reviews/<review-uuid>
```

That read-only route reconstructs the exact parent snapshot and shows its own proposal Markdown, Before/With toggle, structured review map/detail, exact diff, overall body, and anchored comments. It remains directly readable after application and under the approved discard rule. Back/Forward and reload preserve the exact review identity.

Current-generation anchors may be highlighted contextually. Earlier-generation comments are shown only on their own exact review route; they are never rendered inline on the current proposal as if current. Review render failure must leave the exact bodies, anchor data, source ranges, and canonical diff inspectable rather than making durable feedback unreachable.

### Agent Access parity

Reviews 1 adds domain-oriented operations without changing the meaning of `change-set review`:

| Product action | CLI | MCP |
|---|---|---|
| List submitted reviews for one Change Set | `review-submission list` | `review_submissions_list` |
| Inspect one submission and exact reviewed snapshot | `review-submission inspect` | `review_submission_inspect` |
| Submit feedback on an exact current review | `review-submission submit` | `review_submission_submit` |

CLI syntax follows the existing global-flag contract, for example:

```text
workbraid --json review-submission list --store-id <uuid> --change-set-id <uuid>
workbraid --json review-submission inspect --store-id <uuid> --change-set-id <uuid> --review-id <uuid>
```

`review-submission submit` takes the exact identity/binding flags, verdict, author, optional overall Markdown literal/file, and an optional file containing the closed JSON array of `{body, anchor}` comments. This is structured review input, not a generic API call or raw Architecture/YAML patch. WorkBraid generates and returns the review/comment UUIDs.

MCP uses the same closed request fields and a typed comments array. Both transports ultimately call the same server operation and return the same semantic result. There is no process-wide selected review, local private-store fallback, or client-owned snapshot.

List results are compact summaries. Inspect returns:

- review/change-set/comment identities;
- author, timestamp, verdict, overall Markdown, and every comment body/typed anchor;
- `reviewed_state` and exact base/tree/generation;
- exact reviewed proposal Markdown, Before/With projections, structured comparison, and canonical diff;
- whether the reviewed generation is still current;
- active/applied/no-longer-active context;
- truthful out-of-date/current Accepted context; and
- stable browser review URL.

The existing local protocol remains `workbraid-agent-v2`. These operations and the additive `reviewed_state` result field do not invalidate v2 envelopes or existing precondition semantics. The CLI help, embedded `workbraid --skill`, MCP tool schemas/descriptions, and black-box examples change together. The skill clearly separates **prepare Review changes** from **submit review feedback** and explains exact anchors, earlier-generation truthfulness, and informational verdicts.

New shared typed errors are:

| Code | Meaning / next safe action |
|---|---|
| `review_submission_not_found` | That exact Change Set/review UUID pair has no durable review ref. |
| `review_submission_unavailable` | The owned review ref exists but its commit, schema, parent state, or anchors cannot be validated; do not guess or repair it. |
| `review_anchor_invalid` | At least one submitted anchor does not resolve exactly in the bound snapshot; correct the returned comment/field location. |
| `review_submission_not_allowed` | The Change Set is not an active proposal with a current bound review; inspect its lifecycle/state. |

`review_invalidated` remains the classification for a changed state object or base/tree/generation between review inspection and submission. Existing project/store, non-current authority, malformed request, connection, protocol, and operation errors retain their meanings. Agents branch on codes, never prose.


## Boundaries

No approval counts/gates, authenticated reviewers, owner/assignee/priority fields, threads/replies/resolution, editable/deletable submitted feedback, persistent review drafts, proposal history/undo/reopen, applied deletion, raw Git/YAML mutation tool, per-agent authority or generic workflow framework. Rendering and placement do not alter these lifecycle/acceptance rules. Review submissions never advance Accepted.
