# WorkBraid Architecture agent guide

WorkBraid has one Accepted Architecture and durable, named, independent change sets. The running loopback WorkBraid process is the only authority for the current project, Accepted revision, change-set records, review, and update. CLI and MCP are thin clients of that process. Never open the private Git store or edit canonical YAML directly.

Use `workbraid [--server http://127.0.0.1:8080] --json …`. `WORKBRAID_SERVER` supplies the URL when `--server` is absent. Global flags precede the command. Start with `status`; `connection_failed` means the operator must start WorkBraid or correct the URL.

## Identity and exact state

Diagram shapes and notes use portable/operational v6. New projects use v6; old ordinary edits retain their existing version. First actual shape/note change preserves displayed legacy centers/sizes and existing routes. Reads, repeated default/same explicit shape, identical whole-note edit and deletion of an absent valid note UUID are no-ops before upgrade, after exact state/Diagram validation.

Inspect `diagram shapes --store-id <uuid> --diagram-id <uuid> [--change-set-id <uuid>]` and `diagram notes` with the same inputs (MCP `diagram_shapes`, `diagram_notes`). Shapes are tagged explicit enum versus default/null over visible Diagram/Component pairs, including boundaries. `diagram set-shape --store-id <uuid> --change-set-id <uuid> --generation <n> --diagram-id <uuid> --component-id <uuid> --shape rectangle|ellipse|diamond` writes an override; `diagram restore-default-shape` omits shape. MCP names `diagram_set_shape`, `diagram_restore_default_shape` use matching snake_case fields. Explicit Diamond differs from boundary Default even with an equal silhouette. Retained visibility retains shape; disappearance clears it and reappearance gets Default. Shapes do not move/resize nodes or rewrite routes.

Plain notes belong only to their Diagram, with no Component, Relationship, hierarchy or review-comment identity. `diagram add-note` requires exact store/proposal/generation/Diagram and exactly one `--text <plain text>` or `--text-file <path|->`; the backend generates UUID and default 240×120 geometry once, without moving peers. Text is exact valid UTF-8, nonblank, at most 2000 Unicode scalar values. File/stdin preserves whitespace/newlines and is mutually exclusive with literal text. `diagram edit-note` requires those same state fields plus `--note-id`, text/file, and all `--x --y --width --height`: complete replacement, no inherited omitted fields. X/Y are integer −100000..100000, width 120..800, height 48..600. `diagram delete-note` takes state/Diagram/note ID only. MCP `diagram_add_note`, `diagram_edit_note`, `diagram_delete_note` use literal text and matching snake_case fields; file transport is CLI-only. Add accepts no caller UUID; unknown edit is target_not_found. Notes reserve space for new nodes/notes and stay stationary during node Auto-layout. Review lists notes/shapes separately; Before/With retain exact values. Note discussions can use existing Diagram comment anchors.

Reconciliation shape locator is exactly `{"kind":"node_shape","diagram_id":"…","component_id":"…"}`. Side choices contain locator/choice only; manual adds exactly `"value":{"shape":"ellipse"}` or `"value":{"shape":null}` for Default. Legacy retaining formats contribute no shape edit. Notes compare complete values including absence, with locator exactly `{"kind":"diagram_note","diagram_id":"…","note_id":"…"}`. Manual adds exactly `"value":{"note":{"text":"Exact text","x":0,"y":0,"width":240,"height":120}}` or `"value":{"note":null}`. Divergent edit/delete needs explicit whole-note keep/replace/delete. Independently added divergent same UUIDs are unsupported replace_identity. A resolved keep may restore the same old note UUID through ordinary complete final upsert relative to Accepted, only if its owning Diagram survives. This does not authorize parent lifecycle or public creation IDs. Apply retains exact S/B/A/P, changes only the proposal and reconstructs residual facts against Accepted; then use ordinary Review/Update. Never author private Git or reconcile/accept a Markdown-only plan to implement its text.

Route inspection: `diagram routes --store-id <uuid> --diagram-id <uuid> [--change-set-id <uuid>]` (MCP `diagram_routes`) returns every visible link's exact Diagram/source/target/label/one-based-occurrence presentation address, count, custom `route` or default, `display_bend`, eligibility and reason. Omit proposal ID only for Accepted. Slots are not Relationship identities. Self-links keep derived routing; coincident centers reject new routing while retaining any stored scalar for later separation.

`diagram set-route --store-id <uuid> --change-set-id <uuid> --generation <n> --diagram-id <uuid> --source-id <uuid> --target-id <uuid> --label <exact-label> --occurrence <n> --bend <integer>` (MCP `diagram_set_route`) writes signed control-point distance −100000..100000. Use `--label-file <path|->` instead of `--label` for exact UTF-8/file/stdin fidelity. Both forms are mutually exclusive. All identity/state/occurrence fields are required. `diagram restore-default-route` (MCP `diagram_restore_default_route`) takes the same fields except bend. It removes custom routing and restores the derived parallel fan. Inspect after mutations; never guess newer generations.

The scalar is the midpoint **control point**, not the curve midpoint, along the source-to-target normal with shared shape-intersection reference. Node centers/sizes/peers stay fixed. An unchanged stored bend, absent equal-displayed-default set, or absent Restore default is an exact no-op before upgrade. Actual routing upgrades legacy to v5, retaining displayed positions and legacy dimensions rather than enlarging peers. New projects use v5; version never downgrades. Stored custom equal to the fan stays custom until Restore default.

Count changes clear all custom routes for that exact tuple; label/target edits clear old and new groups. Actual edge-visibility loss clears only that Diagram. Source reordering and unrelated tuples retain routing. Invalid pending edits clear only affected counts/visibility; unrelated invalid edit/repair preserves routes. Remove/re-add never implicitly resurrects inherited routing, including across restart. Inspect the separate Diagram routing consequences in Review; Review/Update remain deliberate and separate.

Reconciliation first resolves actual count/composition conflicts. Route comparison is tagged default/custom, never fan numbers; legacy default cannot reset a modern custom value. `route_value` locator is exactly `{"kind":"route_value","diagram_id":"…","source_id":"…","target_id":"…","label":"…","occurrence":1}`. Side choices have only locator/choice; manual adds exactly `"value":{"route":{"bend":120}}` or `"value":{"route":null}` for default. Incompatible routing produces symmetric `route_loss`, with locator exactly `{"kind":"route_loss","diagram_id":"…","source_id":"…","target_id":"…","label":"…"}` and affected-side/address/reason context. Its only direct choice is `{"locator":<exact returned locator>,"choice":"clear"}` without value. This acknowledges whole-tuple default. Automatic count/visibility results cannot be overridden through routing; preserving an incompatible group requires ordinary proposal edits then fresh Preview. Apply retains exact S/B/A/P and changes only the proposal, emitting residual nulls against current Accepted. Historical feedback stays exact. No private-Git authoring or automatic acceptance is available.

Every visible node also has a Diagram-local width/height pair. Inspect using `diagram sizes` with store/Diagram and optional change-set ID (MCP `diagram_sizes`). V4 reports `size`, `display_size` and `size_source: stored`; v2/v3 report legacy `display_size` and `size_source: derived`. Reads never save. `diagram set-size --store-id <uuid> --change-set-id <uuid> --generation <n> --diagram-id <uuid> --component-id <uuid> --width 320 --height 160` (MCP `diagram_set_size`) writes one pair. Width is integer 80–1600; height 48–1200. Center, peers and viewport stay fixed. An unchanged displayed pair is an exact no-op BEFORE upgrading: even legacy 116×54 ordinary or 104×62 boundary preserves version, tree, generation and Review. Only actual changes upgrade legacy to v4, initializing peers at their legacy dimensions and v2 centers at their displayed fallback. New v4 nodes default to 200×96 ordinary/reference and 224×112 boundary. `diagram restore-default-size` takes the same state/identity without width/height (MCP `diagram_restore_default_size`), writes the current-role default once and follows the same no-op rule; it never resets positions or enables automatic sizing.

Retained visibility keeps sizes through home/reference/boundary conversion. Disappearance removes them; reappearance gets a fresh default. Explicit Auto-layout uses node/caption bounds with a 24-unit gap and writes positions only. Reconcile size independently of position with locator `{"kind":"node_size","diagram_id":"…","component_id":"…"}`; manual value is exactly `{"size":{"width":320,"height":160}}`, never null. Context is not_applicable, derived legacy dimensions, or stored. Resolve visibility first, inspect/check exact choices, Apply only to the proposal, then ordinary Review/Update. Historical review dimensions remain exact. Empty Architecture diff shows proposal text and informational feedback but has no acceptance action.

Every visible Component has a Diagram-local center addressed by Diagram UUID + Component UUID, including boundary nodes marked **Lives in**. Inspect with `diagram positions --store-id <uuid> --diagram-id <uuid>`; add `--change-set-id <uuid>` for a valid active or Applied proposal. Appearances and boundaries report exact `position` in v3/v4, `display_position`, and `position_source: stored`. Historical v2 has no stored position and reports `position_source: derived` with its display fallback. Reading never saves layout.

For an explicit active proposal, use `diagram set-position --store-id <uuid> --change-set-id <uuid> --generation <n> --diagram-id <uuid> --component-id <uuid> --x=-180 --y=320`. X increases right, Y down, in Diagram-local logical units independent of zoom/pan. Both integers must be between -100000 and 100000. One set changes one pair and one generation. Reinspect after any mutation or mismatch; never replay against a guessed newer generation.

`diagram auto-layout` takes store/proposal/generation/Diagram flags and deliberately arranges all visible nodes there once using their size and caption bounds. Existing nodes otherwise keep their coordinates; newly visible nodes receive an initial coordinate during authoring. Auto-layout does not alter sizes, membership or Relationships. An unchanged set or arrangement preserves generation and Review. First placement on v2 completes stored positions in every Diagram of that proposal using its displayed fallback; inspect the exact normal Review changes, then deliberately Update its exact binding. New projects use v4; authoring never downgrades the portable version. Restore default size is an explicit size write and does not reset positions.

MCP equivalents are `diagram_positions`, `diagram_set_position`, and `diagram_auto_layout`. Placement reconciliation uses existing preview/check/apply with locator `{"kind":"node_position","diagram_id":"…","component_id":"…"}`. Original/Accepted/Proposed context is `not_applicable` (not visible here), `derived` (v2 comparison fallback), or `stored`, with exact coordinates for the latter two. Composition and Relationships resolve first; absent sides do not mean resets and retaining v2 contributes no placement edit. A manual resolution value is exactly `{"position":{"x":320,"y":-180}}`, never null; side choices use `accepted`/`proposed` without a value. X and Y are one pair. Apply only changes the proposal; ordinary Review and Update follow.

- A project slug locates a catalog entry; its store UUID is project identity.
- Accepted Architecture is singular. `architecture inspect` returns its exact revision, Components, Relationships, Diagram hierarchy, homes, references, and IDs.
- A change-set UUID is proposal identity. Its mutable name is display text only. Never select by name.
- `change-set list --store-id <uuid>` returns active, applied, and unavailable records. `change-set inspect` returns one exact record: lifecycle, name, base, generation, `change_set_state`, proposal Markdown, concrete facts, validity, `candidate_tree`, candidate, review, out-of-date state, and applied revision. Inspect supplies reconciliation inputs without preparing a Review.
- Every structured proposal mutation supplies the exact `store_id`, `change_set_id`, and that record's `generation`. After a successful mutation, use its returned generation or inspect that exact ID again.
- Copy stable IDs and review fields from results. Never invent IDs or infer one proposal from browser selection.

## Commands

Connection and projects:

```text
workbraid status
workbraid project list
workbraid project current
workbraid project create --name <name>
workbraid project open --slug <slug>
workbraid project close --store-id <uuid>
```

Accepted and change-set reads:

```text
workbraid architecture inspect
workbraid architecture refresh --store-id <uuid> --accepted-revision <sha>
workbraid change-set list --store-id <uuid>
workbraid change-set create --store-id <uuid> --accepted-revision <sha> [--name <text>]
workbraid change-set inspect --store-id <uuid> --change-set-id <uuid>
```

Change-set context:

```text
workbraid change-set rename <state> --name <text>
workbraid change-set edit-proposal <state> (--proposal <markdown>|--proposal-file <path|->)
workbraid change-set review <state>
workbraid change-set discard <state>
workbraid architecture update --store-id <uuid> --change-set-id <uuid> --base-revision <sha> --candidate-tree <tree> --generation <n>
```

Review feedback (separate from preparing `Review changes`):

```text
workbraid review-submission list --store-id <uuid> --change-set-id <uuid>
workbraid review-submission inspect --store-id <uuid> --change-set-id <uuid> --review-id <uuid>
workbraid review-submission submit --store-id <uuid> --change-set-id <uuid> --reviewed-state <commit> --base-revision <sha> --candidate-tree <tree> --generation <n> --verdict <comment|approve|request_changes> --author <label> [--body <markdown>|--body-file <path|->] [--comments-file <path|->]
```

Structured authoring:

```text
workbraid component create <state> --title <text> [--description <markdown>|--description-file <path|->] [--diagram-id <uuid>]
workbraid component edit <state> --component-id <uuid> [--title <text>] [--description <markdown>|--description-file <path|->]
workbraid component move-home <state> --component-id <uuid> --diagram-id <uuid>
workbraid relationship add <state> --source-id <uuid> --target-id <raw> (--label <raw>|--label-file <path|->)
workbraid relationship edit <state> --source-id <uuid> --old-target-id <raw> (--old-label <raw>|--old-label-file <path|->) [--occurrence <n>] --target-id <raw> (--label <raw>|--label-file <path|->)
workbraid relationship remove <state> --source-id <uuid> --target-id <raw> (--label <raw>|--label-file <path|->) [--occurrence <n>]
workbraid diagram parent-options <state> --diagram-id <uuid>
workbraid diagram reassign-detail <state> --diagram-id <uuid> --anchor-component-id <uuid>
workbraid diagram create-detail <state> --component-id <uuid> --title <text>
workbraid diagram edit-title <state> --diagram-id <uuid> --title <text>
workbraid diagram show-component <state> --diagram-id <uuid> --component-id <uuid>
workbraid diagram stop-showing-component <state> --diagram-id <uuid> --component-id <uuid>
```

Here `<state>` is `--store-id <uuid> --change-set-id <uuid> --generation <n>`. Component creation may omit `--diagram-id` only for deliberate root fallback. File values are exact UTF-8; `-` reads stdin; literal and file forms are mutually exclusive. Relationship edit/remove uses the exact raw source/target/label plus one-based occurrence. Empty or malformed raw selectors remain valid inputs so invalid rows can be repaired without deleting the change set.

To change a Diagram’s parent Component, read `diagram parent-options` for its exact proposal generation and choose a returned eligible Component UUID. `diagram reassign-detail` keeps the Diagram and its full subtree; it moves only its parent link. This differs from `component move-home`, which moves where a Component lives. Root cannot change parent, and an occupied or descendant anchor is unavailable. With no eligible parent, keep the original proposal and add a suitable Component through ordinary authoring if intended.

Creation starts at generation 0 with a valid candidate equal to its exact Accepted base. Rename, proposal edit, semantic edits, and reconciliation increment only that change set and invalidate only its Review. `change-set review` returns the complete unified Architecture diff, Before/With projections, exact ID/base/tree/generation binding, `reviewed_state`, and a `review_url`; give that URL to the reviewer. Repeating it on the unchanged generation returns the same `reviewed_state`. `architecture update` accepts only that binding. Reconciliation updates the proposal separately; it never prepares a Review or accepts Architecture implicitly. There is no force, accept-latest, automatic review, or combined mutate-and-accept action.

A review submission is immutable informational feedback; it does not accept Architecture or gate later edits. Submit only against the exact `reviewed_state`, base, tree, and generation returned by `change-set review`. The author is a self-described display label, not authenticated identity. A `comment` verdict needs an overall note or at least one anchored comment; `approve` and `request_changes` may stand alone. A comments file is a closed JSON array such as:

```json
[
  {
    "body": "Clarify this responsibility.",
    "anchor": {
      "kind": "component_markdown",
      "side": "with_changes",
      "component_id": "<component-uuid>",
      "start_line": 3,
      "end_line": 4
    }
  }
]
```

Anchor kinds are `proposal`, `proposal_markdown`, `component`, `component_markdown`, `diagram`, `composition`, and `relationship`. Markdown ranges are one-based inclusive lines. Architecture anchors require `side: before` or `side: with_changes`. Inspect the exact review source first and copy IDs, exact Relationship label plus one-based identical-fact occurrence, or composition aspect (`home`, `reference`, or `detail`) from it. Submitted feedback stays attached to that exact version after iteration, acceptance, or discard; use `review-submission inspect` and its `review_url` rather than treating old comments as current.

An out-of-date active change set keeps its base, remains editable and reviewable, and cannot update Accepted until deliberate reconciliation brings it onto current Accepted. Applied change sets are immutable evidence. Discard deletes one whole active change set at its exact generation and changes neither Accepted nor any other record.

## Reconcile with Accepted

Inspect the proposal and Accepted. Both reconciliation operations require exact `--store-id`, `--change-set-id`, `--generation`, `--change-set-state` S, `--base-revision` B, `--candidate-tree` P, and known-current `--accepted-revision` A. Copy S/B/P from Change Set inspect and A from Architecture inspect or explicit Refresh. A first Review preparation can change S without changing generation; an unchanged repeated Review does not.

```text
workbraid --json change-set reconcile-preview <state> --change-set-state <S> --base-revision <B> --candidate-tree <P> --accepted-revision <A>
workbraid --json change-set reconcile-preview <state> --change-set-state <S> --base-revision <B> --candidate-tree <P> --accepted-revision <A> --resolutions-file choices.json
workbraid --json change-set reconcile-apply <state> --change-set-state <S> --base-revision <B> --candidate-tree <P> --accepted-revision <A> --resolutions-file choices.json
```

Preview optionally accepts choices for Check choices. Apply requires `--resolutions-file`; use an exact JSON `[]` for a fully automatic result. `-` reads JSON from stdin. Preview/Check change no refs, state, generation, Review binding or submitted feedback. They return `ready`, `needs_resolution`, `blocked`, or `not_required`, exact inputs and complete Original/Accepted/Proposed context. `automatic_changes` reports `accepted_only`, `proposed_only`, or `same_result`. Only a ready result contains a valid complete `result_candidate`.

Resolutions form a complete JSON array. Copy each returned locator exactly; unknown/duplicate/contradictory locators are rejected. Side choices have only `locator` and `choice`. Manual scalar choices add one typed `value`. Description is the full exact Markdown body, including whitespace and source differences. Title is the existing plain-text projection. Relationship identity is the exact source/target/label tuple, and the merged value is its occurrence count. For example (replace placeholders with inspected UUIDs):

```json
[
  {"locator":{"kind":"component_title","component_id":"<component>"},"choice":"accepted"},
  {"locator":{"kind":"component_description","component_id":"<component>"},"choice":"manual","value":{"text":"\nExact Markdown body.\r\n"}},
  {"locator":{"kind":"relationship_count","source_id":"<source>","target_id":"<target>","label":"calls\nλ"},"choice":"manual","value":{"count":2}}
]
```

Other scalar locators are `diagram_title(diagram_id)` with `{text}`, `home(component_id)` with `{diagram_id}`, `reference(diagram_id,component_id)` with `{present}`, and `detail_anchor(diagram_id)` with `{anchor_component_id}`. Only manual choice carries these values. `component_object(component_id)` and `diagram_object(diagram_id)` offer supported side choices for external absence; no manual object creation or replacement is allowed. A divergent independently added same-new-UUID object is blocked with `replace_identity`, including divergence in dependent facts. Do not remap IDs or migrate dependencies.

Structural locators have `kind: composition`, a returned `reason`, and sorted `component_ids`/`diagram_ids`. Their required `value` contains explicit final `homes`, `references`, `detail_anchors`, and/or `relationship_counts` for the affected facts. Omitted arrays add no choices; they do not mean deletion. An Accepted/Proposed side preference must agree with that side's applicable values. A manual choice can set a valid complete combination.

For two competing children, both must survive. Choosing Accepted keeps its child on the contested anchor, but must also explicitly place the other child. Both final assignments are required:

```json
[
  {
    "locator":{"kind":"composition","reason":"competing_children","component_ids":["<contested-anchor>"],"diagram_ids":["<child-1>","<child-2>"]},
    "choice":"accepted",
    "value":{"detail_anchors":[
      {"diagram_id":"<accepted-child>","anchor_component_id":"<contested-anchor>"},
      {"diagram_id":"<proposed-child>","anchor_component_id":"<explicit-free-anchor>"}
    ]}
  }
]
```

Use the returned eligible targets and complete context; root cannot be a child. Choosing an occupied anchor expands the group to include the other retained child. Check returns that exact expanded locator; retain choices by fact IDs and resubmit the expanded group with every displaced child's explicit destination. Valid joint swaps are supported. With no free valid assignment, leave reconciliation and edit the original proposal through ordinary authoring, such as adding a suitable Component. Never invent a parent or delete/orphan a displaced child.

Apply changes only the active proposal. Its base becomes A, generation increments once, name/proposal Markdown remain exact, and its current Review binding is absent. Accepted, other proposals and submitted Reviews remain unchanged. Use ordinary Review changes, inspect its new diff, then Update architecture. If no Architecture changes remain, the proposal stays active on A; it is neither Applied nor discarded.

After any response loss, inspect the real current proposal. A retry carrying old S returns `change_set_state_mismatch` and current Change Set context before generation/base/not-required checks. Never blindly replay Apply, attribute an arbitrary later state to the earlier request, or invent a reconciliation receipt. If the proposal was meanwhile Applied or discarded, its lifecycle/not-found response remains authoritative.

## Recovery by error code

- `invalid_request`: correct missing or malformed fields. All mutation state fields are required.
- `connection_failed`: start/check the configured server; do not access application data.
- `incompatible_server`: use matching WorkBraid v2 client/server binaries.
- `project_not_found`: list projects, then use the exact slug or create deliberately.
- `project_conflict`, `project_unavailable`: do not bypass the catalog or store authority.
- `project_not_open`, `project_mismatch`: inspect/open the intended project, then inspect again.
- `change_set_not_found`: list change sets for the exact store and copy the UUID.
- `change_set_unavailable`: the record is malformed; Accepted and other records remain usable.
- `change_set_name_conflict`: choose a different active name.
- `change_set_generation_mismatch`: inspect that exact change-set ID and retry from its current generation.
- `change_set_state_mismatch`: inspect the returned current Change Set context and prepare again with new exact inputs. An old-S Apply retry does not perform another mutation; never replay it blindly.
- `change_set_not_editable`: the record is applied and read-only.
- `change_set_out_of_date`: keep editing/reviewing on its base or prepare deliberate reconciliation with current Accepted; do not retry Update on the old basis.
- `architecture_non_current`, `accepted_conflict`: explicitly Refresh and inspect Accepted plus the preserved proposal, then prepare fresh exact reconciliation inputs.
- `reconciliation_unresolved`: complete every returned conflict, including explicit displaced-child assignments, and Check choices again. The original proposal is unchanged.
- `reconciliation_unsupported`: use the returned reason and supported choices. Restoration, deletion, source restoration, root reassignment, and divergent identity replacement are not authoring capabilities. Choose a representable result or return to ordinary proposal editing; never patch the store.
- `refresh_failed`: authority is indeterminate; preserve prior knowledge and retry Refresh explicitly.
- `target_not_found`: inspect Accepted and the addressed change set for stable IDs or the exact raw selector.
- `target_not_eligible`: choose an allowed target from the inspected proposal context.
- `validation_blocked`: `invalid_proposal` requires ordinary proposal repair first; `reconciled_architecture_invalid` requires correcting the explicit choices. Use returned stable locations and exact raw values; no failed preview or Apply changes the proposal.
- `review_required`: inspect and Review the exact current generation.
- `review_invalidated`: inspect, then Review again.
- `review_submission_not_found`: list reviews for the exact Change Set and copy both UUIDs.
- `review_submission_unavailable`: the immutable review record cannot be validated; do not guess or edit private Git.
- `review_anchor_invalid`: inspect the exact reviewed Before/With source and correct the reported comment location.
- `review_submission_not_allowed`: feedback requires an active Change Set with an exact durable Review binding.
- `acceptance_uncertain`: inspect/list/Refresh before any retry; an applied record is the durable receipt.
- `accepted_reload_required`: acceptance succeeded; do not Update again. Refresh or reopen.
- `unsupported_action`: do not substitute raw Git/YAML operations.
- `operation_failed`: inspect current authoritative state before retrying.

## Small JSON workflow

Every value below is copied from the preceding JSON result:

```text
workbraid --json project create --name "Agent example"
workbraid --json architecture inspect
workbraid --json change-set create --store-id <store> --accepted-revision <revision> --name "Gateway proposal"
workbraid --json change-set edit-proposal --store-id <store> --change-set-id <change> --generation 0 --proposal "# Gateway proposal"
workbraid --json component create --store-id <store> --change-set-id <change> --generation 1 --diagram-id <root> --title Gateway --description "Entry point"
workbraid --json component create --store-id <store> --change-set-id <change> --generation 2 --diagram-id <root> --title Worker
workbraid --json relationship add --store-id <store> --change-set-id <change> --generation 3 --source-id <gateway> --target-id <worker> --label calls
workbraid --json change-set inspect --store-id <store> --change-set-id <change>
workbraid --json change-set review --store-id <store> --change-set-id <change> --generation 4
workbraid --json review-submission submit --store-id <store> --change-set-id <change> --reviewed-state <reviewed-state> --base-revision <review-base> --candidate-tree <review-tree> --generation 4 --verdict comment --author "Reviewer agent" --body "Ready for a human decision."
workbraid --json review-submission list --store-id <store> --change-set-id <change>
workbraid --json architecture update --store-id <store> --change-set-id <change> --base-revision <review-base> --candidate-tree <review-tree> --generation 4
workbraid --json architecture inspect
```

For MCP, launch `workbraid [--server <loopback-url>] mcp`. Its typed tools implement the same explicit records and exact preconditions against the same running process.
