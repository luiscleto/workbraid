# Architecture Reconciliation v0

Status: Approved

## Phase 3.3 routing reconciliation

Delegated root approval of routing proposal `d119baeb-1b2b-4451-8113-7f4fe05678ec`, generation 8, reviewed state `595bab6de6933c7a047c70520f1830594f166fed`, extends comparison with the exact Diagram presentation slots and portable/operational v5 rules in [Placement §10](architecture-placement-amendment-v0.md#10-deliberate-link-routing-phase-33). It supersedes routing exclusions and new operational-version statements below, without changing S/B/A/P or acceptance authority.

Resolve semantic counts/composition first. A side's route group is eligible only when final canonical edge visibility and tuple count match that side. With stable visibility/count, compare the whole **tagged default versus custom bend** per slot using the ordinary B-based three-way rule. Never compare displayed fan numbers. Independent slots/Diagrams combine, equal results coalesce, and divergent values conflict as `route_value`. Legacy no-custom-route is default, not a reset instruction: legacy A default plus modern P custom and the reverse retain custom without conflict. Modern custom versus explicit default uses normal B-based comparison. Stored custom equal to today's displayed fan stays custom until ordinary Restore default. Complete supported same-new-UUID comparison includes relevant routing facts.

The closed `route_value` locator is exactly `{kind, diagram_id, source_id, target_id, label, occurrence}` with `kind: "route_value"`. A side resolution is exactly `{locator, choice: "accepted"|"proposed"}`. Manual is exactly `{locator, choice: "manual", value: {route: {bend}|null}}`; null means default and bend uses the approved integer bounds. Side applicability is validated, not inferred from a displayed default number.

If either A or P edited routes whose count or visibility is incompatible with the final projection, emit `route_loss` for that Diagram/tuple, symmetrically. Return the affected side, addresses and reason as conflict context. Its closed locator is exactly `{kind: "route_loss", diagram_id, source_id, target_id, label}`. Its only direct resolution is exactly `{locator, choice: "clear"}`, without value. Clear acknowledges loss and makes the final whole tuple default, stripping both inherited and pending overrides. Final invisibility permits only acknowledged clear. Reject unknown/missing fields, other choices, duplicate/stale locators, irrelevant values and ineligible addresses.

Existing returned count/composition conflicts may be revised with their own existing resolution shapes, followed by recomputation. Route loss never grants authority to override an automatic semantic result, restore identity/visibility or copy an ineligible group. For B count 2, A count 1, P count 2 plus a bend edit, final count 1 is automatic: clear is the only resolution within that preview. The same holds for automatic visibility loss. Preserving incompatible routing instead requires ordinary proposal authoring and a fresh attempt. Browser and CLI must not offer an unusable revise action when no actual semantic conflict exists.

Result format is max(A,P). Emit ordinary final residual facts relative to A, including necessary nulls against A; never copy B tombstones. The single strict constructor must reproduce the exact selected result tree. Preview/Check remain non-mutating, Apply changes only the proposal under exact S/B/A/P, and old-S retries retain existing classification. Accepted, other proposals and submitted Reviews remain exact. Verify both count-loss directions, both visibility-loss directions, both mixed-version custom/default directions and exact whole-tuple clear residual replay. Historical review anchors and the final human gate remain separate from technical approval.

Phase 3.2 adds independent whole size pairs after final visibility, under [Placement §9](architecture-placement-amendment-v0.md#9-complete-visible-node-sizing-phase-32). The closed locator is `{kind: "node_size", diagram_id, component_id}`. Side choices have only locator/choice; manual requires exactly `value: {size: {width, height}}` with approved bounded integers. Null and absent-side values are not manual sizes. Context distinguishes `not_applicable`, legacy `derived` displayed dimensions and v4 `stored` dimensions. Older retained formats contribute no sizing edit; max(A,P) governs result format and selected stored choices survive. Size and position merge independently. Complete supported same-new-UUID comparison includes both presentation facts. Preview/Check/Apply and exact residual reconstruction remain unchanged. New operational writes use v4; historical version descriptions below remain exact.

Scope: ordinary detail-link reassignment and deliberate reconciliation of one valid out-of-date active Architecture proposal

This living contract owns ordinary detail-link reassignment and semantic reconciliation. [Proposals and Reviews](architecture-proposals-v0.md) owns durable envelopes and feedback; [Architecture](architecture-v0.md) owns source fidelity. The [placement contract](architecture-placement-amendment-v0.md) extends comparison to complete visible-node coordinates and mixed v2/v3 inputs.

## Authority and product boundary

There is one running WorkBraid application/Manager authority, one concrete application synchronization boundary, and one Accepted Architecture per project. Browser, CLI, and MCP enter the same operations. Every proposal operation addresses its store UUID and Change Set UUID; there is no process-wide selected proposal or reconciliation session.

Reconciliation has exactly these immutable inputs:

- `S`: the active Change Set state commit, identifying its exact metadata, proposed facts, and optional current review binding;
- `B`: that state's exact base Accepted commit;
- `P`: its valid candidate tree, reconstructed by the existing constructor and validated by the existing Architecture loader;
- `A`: the exact known-current Accepted commit observed for this attempt.

The request additionally carries the exact store UUID, Change Set UUID, and generation. These are preconditions, not client authority. A valid active proposal is required. An invalid proposal must first be repaired against B. An Applied proposal cannot reconcile. When A equals B, return `not_required` without changing anything.

Reconciliation updates only that active Change Set. It never updates Accepted, prepares an acceptance review implicitly, submits reviewer feedback, or interprets a verdict. The later ordinary Review changes and deliberate Update retain the existing exact binding and three-ref acceptance transaction.

## Ordinary detail-link reassignment

### Domain semantics

A non-root detail Diagram may be reassigned to another Component's home appearance. Its UUID, canonical path, regular-file mode, contents, and full subtree remain exact except for independently authored edits. Only the two parent-owned `detail_diagram` links change. No Component blob or Relationship fact changes. Children gain no parent fields, copies, or new identities.

Resolve eligibility from the complete proposal context, including pending-new Components, pending-created Diagrams, earlier home changes, and earlier net anchor changes. The destination must have a home, be different from the current anchor, have no other child in the resulting assignment, and not lie in the reassigned Diagram's subtree. Root cannot be reassigned; a Component living in root is a valid destination when otherwise eligible. The complete result must have exactly one parent per non-root, root reachability, and no cycle.

A direct repeated request naming the already-current anchor returns a no-op without a generation or review change; it is not offered as a destination. An ineligible direct action retains the original proposal, rather than durably detaching a child while waiting for a second action. Unrelated invalid proposal state continues to use existing repair controls; this action does not pretend a complete candidate exists when it does not.

One reassignment is an atomic ordinary composition mutation. The backend calculates the requested final links on a private copy, constructs/validates once through the shared path, and publishes one new proposal generation only on success. No intermediate detached or multiply parented durable state is permitted.

### Minimal concrete pending representation

Add one sequence of final values to the existing concrete composition:

```yaml
detail_reassignments:
  - diagram_id: 55555555-5555-4555-8555-555555555555
    anchor_component_id: 22222222-2222-4222-8222-222222222222
```

Each entry identifies an existing non-root Diagram in the proposal base and its desired final anchor Component. There is at most one entry per Diagram. Both values are stable UUIDs. There is no old-anchor field, detach value, timestamp, request order, or history. Returning to the base anchor removes the residual entry.

For a Diagram created in this same proposal, update its existing `detail_diagrams[].anchor_component_id`; do not add a reassignment entry for that new Diagram. Its generated UUID/path remain unchanged. A home move still carries the Component's final child link; it does not change which Component anchors the child.

All desired child assignments are evaluated together with final homes. This allows a complete set of explicit choices to exchange occupied anchors without an intermediate canonical detachment. It does not permit an ordinary single reassignment request to displace an unspecified child. No intermediate per-entry cycle/occupancy check may reject an otherwise valid final batch merely because of serialization order.

This is a concrete extension of `CandidateComposition` and `ConstructCandidate`, not an operation algebra or a second builder. Existing creation, home, and reference facts retain their meanings and source-order behavior.

### Closed operational schema evolution

The supported historical `changes.yaml` **version 2** retains `format: workbraid-change-state` and every version-1 item spelling, with seven required sequences:

```yaml
format: workbraid-change-state
version: 2
components: []
new_component_homes: []
detail_diagrams: []
diagram_titles: []
home_moves: []
references: []
detail_reassignments: []
```

The six original sequences retain their exact version-1 schemas. The new item has exactly the two keys above. Unknown/missing keys, duplicate Diagram entries, malformed UUIDs, a reassignment entry for a newly-created Diagram, and unsupported versions are rejected through the existing closed operational-state validation. Domain eligibility remains complete-candidate validation, not YAML interpretation in clients.

Operational v1/v2 remain supported with exact closed schemas and reconstructed trees over portable v2. New proposals and actual mutations now use operational v3 as defined in [Proposals and Reviews](architecture-proposals-v0.md#operational-versions-and-supported-history). Loading/review must not convert old facts or lose candidate availability. Preserve supported historical reviews across GC and restart; already-unavailable pre-fix alpha artifacts need not be revived.

The Change Set envelope version, active/applied refs, one-parent state-commit rule, proposal Markdown and Review storage do not change. Reassignment itself needs no portable-schema change; placement uses its approved v3 extension. No metadata file enters accepted Architecture. Every loaded record must still pass exact candidate-tree equality through the one current constructor; never bypass that check for legacy data. New valid records and their submitted Reviews retain all exact reconstruction, immutable feedback, reachability and lifecycle guarantees across later iteration, reconciliation, application and restart. Repeated exact Review changes on a supported unchanged record remains a durable state-object no-op. Completed execution records remain archived in Git history; runtime evidence is preserved separately.

### Ordinary UX and agent parity

The selected non-root Diagram's contextual actions offer **Change parent component**. The pane says **Change parent component for “Runtime”**, shows the present parent Component and its containing Diagram, and offers valid destination Components grouped by where they live. Titles are primary; collision context appears only when needed. This is neither **Change where Component lives** nor **Open Diagram**. Keep actions visibly actionable and within the existing pane, not another top bar or hierarchy-management screen.

In a proposal it edits that proposal; starting from Accepted uses the established exact store/revision implicit Change Set creation plus first mutation. Ineligible requests create no empty proposal. Opening the browser form may read destinations against exact store/Accepted revision without creating a proposal; only keeping a real change starts one. Backend eligibility reads and mutation share the existing authority boundary. Expose a concrete candidate-relative destination read to CLI/MCP as well as the browser; clients must not derive eligibility from a partial tree.

Review shows **Runtime moved from Gateway to Operations** as composition on the affected parent Diagrams, identifying the two Components while retaining the existing Diagram identity. It is not Diagram deletion/addition, Component content change, or a Relationship delta. Exact Diagram-file diff remains available. Submitted Reviews of the old links stay on their old generation and side.

## Three-way semantic comparison

For each supported value, compare exact domain values:

```text
if A_value == B_value: choose P_value
else if P_value == B_value: choose A_value
else if A_value == P_value: choose that value
else: require explicit resolution
```

Absence is explicit, not an empty string, zero UUID, or null Markdown. Comparisons use the existing loaded Architecture facts, not YAML nodes, filenames, rendered Markdown, or a new graph model. Temporary comparison records are request data and cannot construct or persist Architecture independently.

### Exact units

| Fact | Unit and equality | Resolution boundary |
| --- | --- | --- |
| Component existence/identity | Stable Component UUID; additions are distinct by UUID. Presence is assessed before field comparison, with deletion/edit protection below. | Distinct additions combine. Same-new-UUID additions coalesce only if their complete supported semantic object/facts are identical; otherwise `reconciliation_unsupported`, reason `replace_identity`, with no side/manual resolution. |
| Component Title | Existing structured plain-text Title value | Use either exact value or an ordinary valid structured Title. Source-only H1 formatting differences do not become Title conflicts. |
| Component Description | Entire exact Markdown body bytes returned by the loader | Whitespace and Markdown-source differences are real Description differences. Use either exact body or supply exact UTF-8 Markdown. No line merge, AST merge, whitespace normalization, or rendered-text comparison. |
| Relationship | Multiplicity of `(source UUID, target UUID, exact label)` | Non-negative integer count; zero removes that fact. Direction/source ownership and label bytes are exact. |
| Diagram existence/identity | Stable Diagram UUID; title and composition form its object context, not its filename | Independent additions combine. Same-new-UUID additions coalesce only if their complete supported semantic object/facts are identical; divergence is `reconciliation_unsupported`, reason `replace_identity`, not a selectable replacement. |
| Diagram Title | Exact loaded title string | Either side or a non-blank valid Diagram title, retaining valid submitted text under existing authoring rules. |
| Component home | One Diagram UUID for each retained Component | Different destinations conflict. Manual choice names one existing result Diagram. |
| Reusable reference | Boolean presence for `(Diagram UUID, Component UUID)` | Pure boolean three-way changes do not themselves produce divergent scalar conflicts. They can interact with homes or entity absence and require composition resolution. |
| Detail parentage | Anchor Component UUID for each retained non-root Diagram | Different anchors for the same Diagram conflict. Anchor occupancy across different Diagrams is a structured competing-children conflict. The link remains stored only on the parent home. |
| Root | Manifest's root Diagram UUID | P inherits B's manifest. Adopt A's root; no manual root replacement or reassignment operation. Validate all resulting topology against it. |
| Project name and slug | Exact manifest values | P inherits B. Reuse A's manifest values and blob, including a legitimate externally changed slug. Reconciliation is not project rename. |
| Store ID, format, version | Required identity and supported format invariants | All inputs must belong to the same store and load as supported Architecture. They are not mergeable fields. |

No other current canonical semantic field is silently dropped: appearances consist only of home/reference membership and optional detail links; boundary references are derived; filenames, modes and YAML spelling are fidelity concerns below. Proposal name/Markdown and submitted Reviews are not Architecture merge inputs.

For a UUID absent from B and independently added in both A and P, compare the complete supported semantic object and facts involving that identity. For a Component this includes content, incoming/outgoing exact Relationship multisets, home/reference placement and owned detail link; for a Diagram it includes title, root/parent-anchor context and complete home/reference/detail composition. Do not compare only the file-local title/body while ignoring side-specific dependent facts. Equality uses the existing concrete domain facts, not a generic dependency graph.

If those facts are identical, coalesce once using A's source/path/mode fidelity. Otherwise report `reconciliation_unsupported` with reason `replace_identity` and the exact colliding identity/context. Offer no Accepted, Proposed or manual replacement choice for that collision: independently merging dependencies after choosing one object's content could attach them to the wrong conceptual object. No UUID remapping, dependency migration, cloning, copying or per-collision Relationship rewriting is allowed. This collision blocks the attempt, unlike an external deletion conflict which may have a supported Accepted-side result. For retained base identities, compare the separate units above rather than classifying an ordinary edit as an identity collision.

### Relationship consequences and order

Count each exact tuple independently, including identical parallel occurrences. Same-result count changes combine once, not additively. Differently changed counts conflict. Label/target edits remain removed fact plus added fact; two different new labels may therefore both survive automatically. Do not infer a shared Relationship identity or a conflict between unrelated new facts.

Serialization keeps surviving A occurrences in A source order up to each chosen final count, then fills deficits from P occurrence order, then any additional manually requested multiplicity in deterministic tuple order. Never trim valid labels or collapse whitespace/newlines. If the final multiset equals A's, retain A's outgoing source untouched. Order has no domain meaning and no occurrence acquires identity.

### Structural conflicts and competing children

Scalar choices are not proof of a valid Architecture. The complete result must satisfy the one existing validator. Interactions include home/reference overlap, missing homes/targets, occupied anchors, and a cycle produced by individually legal home/anchor choices. A boundary changing into an ordinary node is derived presentation, not a Relationship conflict.

Report concrete involved Component/Diagram facts and the validator's location, not a raw YAML conflict. A structural conflict exposes only relevant typed homes, reference-presence bits, child-anchor assignments, and dependent Relationship counts/entity choices. It does not offer arbitrary properties or a generic patch editor. Ordinary validity rules remain unchanged.

For competing children:

```text
B: Gateway has no child
A: Gateway -> Operations (D1)
P: Gateway -> Runtime (D2)
```

Keep both distinct added Diagrams. The conflict is one occupied anchor with two required child assignments. The UI offers **Keep Operations on Gateway** or **Keep Runtime on Gateway**, followed by **Parent component for the other Diagram**. Neither first choice is complete until the displaced child has an explicit valid free anchor. Show both children and their final anchors together. API side choices likewise require complete explicit child assignments; `choice: proposed` alone cannot drop D1, nor can `choice: accepted` drop D2.

Root cannot be a reassigned child. An occupied, missing, or descendant Component anchor is unavailable for a single displacement; homes in root remain eligible. With no valid assignment, leave the conflict unresolved and explain that another free parent Component is needed. The human/agent may leave reconciliation, edit the original proposal through ordinary authoring (for example add a suitable Component), and prepare a new preview. Never create an anchor, collapse the Diagrams, select a parent automatically, or delete a displaced subtree.

Resolve assignments jointly: one explicit conflict result may relocate several involved children and homes, with each stable identity retained. If a chosen anchor is occupied by another retained child, that child's destination must also be explicit and included in the affected context. A fully specified swap can be valid; a partially specified displacement cannot. Root/child contents and identities are not rewritten to implement it.

Home/reference collisions are also explicit. The resolver must select a valid final home and reference presence; it cannot silently claim a selected reference survived when the same pair became home. The existing net absence rule is preserved when deriving residual facts, so references from A cannot resurrect after a chosen home departure. There is no command-history reconstruction.

### External lifecycle and unsupported choices

Valid externally selected Accepted remains authoritative and loadable. Reconciliation does not reject it as corrupt merely because normal authoring cannot reproduce a reverse transition.

Detect deletion versus modification before merging fields. For a base object missing in A, do not apply a bare boolean presence rule and silently discard a proposed content or composition edit. Return the absent Accepted side and exact proposed object/dependencies as a conflict. If P has not changed that object or dependent facts, adopt its absence normally. Newly dangling proposed facts still require explicit correction.

For `B: X exists; A: X absent; P: X edited`:

- **Use Accepted** may keep X absent. Omit obsolete edits to X from the new residual state; explicitly resolve any surviving proposed homes, references, links, or Relationship facts requiring X. Do not cascade-drop independent proposals silently.
- **Use proposed** is individually unavailable with `reconciliation_unsupported`, reason `restore_component` (or `restore_diagram`). It would restore an identity/path/source which this slice cannot author.

Similarly, choices requiring deletion of an object present in A, root reassignment, identity replacement/restoration, or arbitrary canonical-path/source restoration are individually unsupported. Keep other representable choices available. Selecting A's path/mode for the same retained identity is not restoration. A changed external root can be adopted when the remaining facts are representable and valid; do not blanket-reject all non-linear/external updates.

If no complete valid representable selection exists, apply is blocked with typed reasons and no mutation. No undelete, clone, source-copy escape hatch, UUID replacement, raw patch, or hidden fallback is introduced. Distinct supported independently created identities are not discarded merely to resolve anchor occupancy.

## Fidelity and reconstruction relative to A

A is the serialization foundation, not B. Preserve every untouched A tree entry, exact blob and regular-file mode. YAML/frontmatter spelling/order, Diagram YAML lexical formatting, canonical path spelling and regular-file mode are fidelity concerns, not semantic conflict units. A-only changes to those properties remain the foundation unless an approved semantic edit requires rewriting the file. This does not make Component body whitespace ignorable: the entire exact Markdown body remains its Description value. H1 equality alone uses the existing structured Title projection.

For edited Components, use the existing source-aware serializer: Title-only/Description-only/Relationship-only rules remain exact. Unchanged H1/body bytes stay intact as required; a Relationship edit rewrites only required frontmatter metadata. For changed Diagrams preserve IDs, paths, modes, unchanged semantic values and surviving A appearance order. No lossless-YAML subsystem is required for an edited Diagram. Reassignment does not rewrite the child/subtree files unless independently edited.

Use P's assigned path for a genuinely proposal-new object when free in A. If it collides with a different identity's path, keep the A path and allocate the new object's path deterministically using the original stem and its stable UUID, with a bounded deterministic suffix search for a further collision. Do not derive identity from paths, rename existing objects, or silently generate new UUIDs. Retain new-object source bytes when the existing constructor can reproduce them; changed path alone must not change its content. Existing new-file mode remains `100644`.

Generate concrete residual authoring facts, not a merged-tree patch:

1. Existing Components in A: begin with the existing authoring projection, set only Title/Description/Relationship changed flags whose chosen semantic values differ from A, and retain A path. Emit no synthetic Component edits for composition.
2. Proposal-new Components absent from B and A: retain their generated IDs, supported source values and allocated paths, emit `new: true`, and one final `new_component_homes` entry. An externally deleted base identity is not treated as new.
3. Existing Diagrams: emit only changed `diagram_titles`, `home_moves`, final reference presence/absence, and `detail_reassignments` relative to A.
4. Proposal-new detail Diagrams: emit their existing creation facts with selected final title/path/anchor. Do not emit historical moves or a reassignment for the same new child.
5. For every final home/ref pair, account for home moves' subsumption and removal behavior, using explicit net reference absence or presence where required. Preserve surviving appearance order; insert genuinely new appearances in deterministic natural authoring order. Do not sort all existing appearances.
6. Submit those facts to the one `ConstructCandidate(A, facts)`, load/validate the resulting complete tree through the one version-aware path, and check that its semantic facts equal the explicit resolved result. Constructor normalization must not silently swallow a conflict choice.
7. Reconstruct again at the ordinary durable write/load equality boundary: `ConstructCandidate(A, reconciled_changes).Tree == reconciled_candidate_tree` exactly. The constructor's output is the candidate; there is no independently constructed merge tree to install.

A result outside this concrete authoring model is a typed unsupported resolution, not permission to extend it silently. No generic inverse-diff program, source importer, raw blob override, merge-only pending kind, or alternate parser is permitted.

If all chosen facts equal A, emit empty residual sequences and reuse A's exact tree, including its lexical formatting. Keep the same active proposal and Markdown, set base A, increment generation once, clear the review binding, and say **No Architecture changes remain**. Do not mark it Applied or discard it. A no-tree-change proposal still cannot advance Accepted.

## Preview, resolutions, and atomic apply

### Prepare reconciliation

Under the existing lock check exact loaded store, active S/generation/B/P, valid reconstructed candidate, and known-current loaded A. Observe the real Accepted ref and active ref, not just cached generation. If external Accepted differs from loaded A, require explicit Refresh rather than adopting it inside reconciliation. Operational observation failure retains truthful indeterminate knowledge.

Capture immutable inputs coherently. Compare, derive conflicts and any resolved candidate; expensive response serialization may happen after capture. Before returning a usable preview, re-observe S and A so a known superseded input is not advertised as ready. A later race is still guarded at apply. Preparing creates no ref, state commit, generation, review binding, or submitted review. Constructing preview trees may write ordinary unreachable content-addressed blobs/trees through the existing constructor, but never a preview ref or durable session.

Return all exact input tokens; complete existing B/A/P projections; changed units combined automatically with reason `accepted_only`, `proposed_only`, or `same_result`; every known conflict with three-side context and supported choices; and a complete result projection/tree only when fully resolved and validated. Report all independent scalar conflicts together, not one per request. Structural conflicts arising only after a particular choice are reported when that choice is checked. Unresolved values are not a partial valid Architecture map.

Prepare also accepts an optional complete set of tentative typed resolutions. This is the same non-mutating operation for **Check choices**, not a third workflow stage. It permits joint structural validation and discovery of conflicts caused by choices before apply. It retains the same B/A/P identity; a different S or A requires a fresh attempt. Do not retry or persist automatically.

### Closed request-local resolution vocabulary

Conflict locators are typed identity tuples, not UUID records. Each is a closed object with `kind` plus the indicated fields: `component_title(component_id)`, `component_description(component_id)`, `relationship_count(source_id,target_id,label)`, `diagram_title(diagram_id)`, `home(component_id)`, `reference(diagram_id,component_id)`, `detail_anchor(diagram_id)`, and `component_object(component_id)` / `diagram_object(diagram_id)` for existence/deletion context. These object locators may identify an unsupported collision diagnostic but never make a divergent same-new-UUID collision resolvable. Labels are exact strings, never concatenated into ambiguous identity keys. Three-side context uses explicit `exists` plus the applicable typed value when present, so absent and empty remain distinct.

A scalar resolution contains exactly `locator`, `choice` (`accepted`, `proposed`, or `manual`), and, for manual choice only, one `value` object. Text units accept `{text}`; relationship counts `{count}`; references `{present}`; homes `{diagram_id}`; child anchors `{anchor_component_id}`. Existence/deletion conflicts offer only their supported side choices, not arbitrary object/identity creation. Divergent same-new-UUID collision diagnostics accept no resolution choice. Unknown keys, duplicated locators, invalid value types and irrelevant fields are rejected.

Structural conflicts return a deterministic locator with exactly `kind: composition`, `reason`, sorted `component_ids`, and sorted `diagram_ids`. Initial reasons are `competing_children`, `home_reference_overlap`, `missing_home`, `missing_target`, `missing_parent`, `unreachable_diagram`, and `hierarchy_cycle`; occupied/multiple parent facts identify every involved child/anchor. The resolution has `locator`, `choice`, and a required `value` object containing explicit final `homes` (`component_id`, `diagram_id`), `references` (`diagram_id`, `component_id`, `present`), `detail_anchors` (`diagram_id`, `anchor_component_id`), and affected `relationship_counts` (`source_id`, `target_id`, `label`, `count`) as needed. Omitted arrays mean no additional choices, not deletion. This is temporary resolution data, never `changes.yaml`. Entries may name only identified affected facts; selecting an occupied anchor expands the involved context before that group can be resolved. No topology property bag is accepted.

For competing children the required `detail_anchors` array lists both children and their final anchors. An Accepted/Proposed choice fixes that side's child at the contested anchor; its explicit value must also place every displaced child consistently. A manual choice may specify another complete valid assignment. For other structural groups, a side choice retains that side's applicable involved values and must explicitly complete missing/dependent assignments; values contradicting that side choice are rejected rather than silently overriding it. Cross-group contradictory choices are rejected, not applied last-writer-wins. Manual values can resolve affected automatic facts when needed to repair a structural incompatibility; unrelated facts cannot be edited through the resolution request.

If choices expose another structural conflict, return its exact affected context with the unchanged inputs and retain browser choices locally. Match those local values only by their exact typed fact identities, never by old display position; an expanded group must be resubmitted with its returned locator. The human/agent can check revised complete choices again. Neither prepare nor failed apply mutates the original valid proposal. The existing validator remains final authority; factor its concrete composition checks for reuse if necessary, rather than building a second validator or a generic validation engine.

### Apply reconciliation

Apply carries the exact preview inputs and complete explicit resolutions. Under the existing lock:

1. Verify store/project, active ref exactly S, generation, B, P, and known-current A. For a later Apply retry, check the active state object before classifying a generation/base change or returning not-required: old S against a moved active ref returns `change_set_state_mismatch` with truthful current Change Set context. A new review binding can change S without changing generation; it still invalidates this preview. Repeated exact Review changes remains the existing durable no-op and does not invalidate it.
2. Re-observe Accepted, reconstruct/validate the exact inputs, and recompute the semantic comparison and resolutions server-side. Do not trust a supplied result tree or a remembered preview cache.
3. Reject unresolved, unsupported, contradictory, ineligible or invalid results without a state change. Construct and verify the residual candidate as above.
4. Prepare one ordinary active state commit preserving UUID/name/proposal bytes, with base A, generation plus one, chosen facts/candidate and no review block. Its sole parent is A, not S. Use the ordinary envelope writer and equality checks.
5. Publish with one concrete Git ref transaction: `verify refs/heads/accepted A` plus `update refs/workbraid/change-sets/active/<id> <new-state> S`. This verifies Accepted without updating it and closes the external-ref race after computation. No reconciliation namespace is created.
6. After a known successful transaction, publish/return the authoritative resulting active Change Set state. If response serialization/delivery or cached publication fails after that known success, recover by reading the real active ref, not by recomputing another mutation. There is no reconciliation journal/receipt/history.

Successful publication changes the active ref away from S. A later client retry carrying the old exact preview inputs therefore returns `change_set_state_mismatch` and truthful current Change Set context; it must not apply again or silently return not-required as though the old request were current. The caller inspects the current proposal. Do not guess whether an arbitrary later state resulted from the previous reconciliation. Do not add `reconciliation_uncertain`, an idempotency receipt, journal, hidden history, or blind Apply replay solely to classify retries. Existing not-found/not-editable handling remains truthful if the proposal has since been discarded or Applied.

Failed preconditions or transaction change neither proposal nor Accepted. Unreachable prepared objects are not durable proposals. Observe truthful post-transaction context: Accepted may advance immediately afterward, making the successfully reconciled proposal out of date again. Report that rather than rolling back or claiming it is definitely current. Indeterminate observation is not proof of staleness or transaction failure.

Other proposals and all submitted-review refs remain exact. No old active-state parent chain is added. Existing review commits retain old states where feedback exists; otherwise an old active state may become unreachable under normal Git GC.

## Browser task

An out-of-date valid active proposal offers **Reconcile with Accepted** on its existing proposal route. Invalid work instead gives actionable repair guidance. Accepted, Applied, and known-non-current/indeterminate states do not offer a misleading ready-to-apply reconciliation action.

Use a focused task in the existing workbench, with project/proposal identity and navigation back to the proposal/Accepted preserved. No permanent merge dashboard, extra process-wide selection, or large form in the application header. No extra route is required for an unpersisted attempt; reopening the proposal never implies saved conflict choices.

Separate **Combined automatically** from **Needs a decision**. Group conflicts by Component/Diagram with counts and a scrollable compact navigator; open one context at a time in the pane. Show **Original**, **Accepted**, **Proposed**, and the chosen result with IDs/revisions in bounded technical details. Show complete long Markdown with wrapping and internal scrolling, not cropped boxes. No Git ours/theirs, merge-base jargon, raw YAML, or generic editable JSON.

Simple conflicts have explicit side choices and an appropriate existing text/structured editor. Relationship conflicts explain the exact label, endpoints and count; home choices show where the Component would live. Child displacement uses the two-child interaction in section 3.3, not falsely complete side buttons. Unsupported choices explain their specific missing operation while leaving supported choices actionable.

**Check choices** recomputes a non-mutating preview when needed. **Apply reconciliation** is deliberate and changes only this proposal. After success return to its coherent proposed workspace, with a clear next **Review changes** action. Ordinary Review provides the complete new-base diff; no separate authoritative reconciliation-diff product is introduced.

Protect unsent manual values and selections with the existing leave guard, including project/proposal navigation and Refresh. Local choices are not durable and do not change the original proposal. Large conflict sets require navigable, non-cropped controls and complete agent output, not hidden truncation or an unapproved session store. If actual usage demonstrates that restart-safe resolutions are needed, stop for that separate decision.

Preserve safe Markdown, retained review context, proposal Markdown visibility, normal proposed/Accepted snapshot separation, map/comment ergonomics, and direct historical-review navigation. Historical feedback stays visibly attached to its exact old state and is never overlaid as current after reconciliation.

## CLI, MCP, skill and local protocol

Keep `workbraid-agent-v2`; these are additive operations with existing envelope/error conventions. CLI global flags precede commands. MCP remains `workbraid [--server <loopback-url>] mcp`, a stateless proxy to the same running authority.

| Capability | CLI | MCP | Local API suffix under `/api/agent/v2` |
| --- | --- | --- | --- |
| Eligible parent Components | `diagram parent-options` | `diagram_parent_options` | `/diagrams/parent-options` |
| Ordinary reassignment | `diagram reassign-detail` | `diagram_reassign_detail` | `/diagrams/reassign-detail` |
| Prepare/check reconciliation | `change-set reconcile-preview` | `change_set_reconcile_preview` | `/change-sets/reconcile-preview` |
| Apply reconciliation | `change-set reconcile-apply` | `change_set_reconcile_apply` | `/change-sets/reconcile-apply` |

Parent options and reassignment require ordinary exact `store_id`, `change_set_id`, `generation`, and `diagram_id`; reassignment adds `anchor_component_id`. CLI flags use their kebab-case spellings. Options return current anchor/home, candidate context/tree, and eligible Component IDs/titles/home Diagram context, not a browser capability claim. Applied/invalid/unavailable state is classified truthfully. A late request is revalidated under the backend lock.

Both reconciliation operations require `store_id`, `change_set_id`, `change_set_state`, `generation`, `base_revision`, `candidate_tree`, and `accepted_revision`. Existing Change Set inspect supplies the state commit and candidate information; Architecture inspect/Refresh supplies observed Accepted. Do not require Review changes just to obtain these inputs. CLI flags use kebab-case. Preview optionally accepts `--resolutions-file`; apply requires it, with `[]` for a fully automatic result. `-` means exact JSON from stdin. MCP supplies the same closed `resolutions` array directly. These are typed JSON values, not YAML or file patches.

Successful preview returns `status: ready | needs_resolution | blocked | not_required`, exact `inputs`, `automatic_changes`, `conflicts`, and `result_candidate` only when valid. Each conflict includes exact three-side values, typed locator, supported choices and unsupported reasons, and concrete eligible targets where relevant. Apply returns exact new Change Set state/base/generation/candidate, remaining-change indication, truthful Accepted context, and absent current review binding. No acceptance binding is fabricated.

Common errors remain stable; agents never classify prose:

| Code / result | Meaning |
| --- | --- |
| existing project/change-set lookup, mismatch, unavailable, not-editable codes | Wrong authority or lifecycle; nothing changed. |
| `architecture_non_current` / `refresh_failed` | Accepted is known non-current or cannot currently be determined; explicit Refresh is required as appropriate. |
| `change_set_generation_mismatch` | The inspected generation changed. |
| `change_set_state_mismatch` | Exact active state object changed, including a retry with old S after successful Apply. Return truthful current Change Set context; inspect it before preparing again. Never replay blindly or infer the cause of an arbitrary later state. |
| `accepted_conflict` | Accepted changed from preview A; Refresh/inspect and prepare again. |
| `validation_blocked` | Invalid starting proposal (`reason: invalid_proposal`) or invalid resolved Architecture (`reason: reconciled_architecture_invalid`), with localized context. |
| `invalid_request` | Malformed/unknown/duplicate locator, value or resolution fields. |
| `reconciliation_unresolved` | Required scalar choices or displaced-child/structural assignments remain incomplete. |
| `target_not_found` / `target_not_eligible` | Selected target disappeared or is not a valid result destination. |
| `reconciliation_unsupported` | A divergent independently added UUID collision or a chosen result requiring unsupported lifecycle/source behavior; details name the affected locator, reason and any supported alternatives. A divergent identity collision has no replacement alternatives. |
| `not_required` success status | A equals B; no generation/ref/review changes. |
| `operation_failed` | Other operational failure; inspect authoritative state before retrying. |

Unsupported reason values are a closed initial set: `restore_component`, `restore_diagram`, `delete_component`, `delete_diagram`, `replace_identity`, `restore_source`, `reassign_root`. A lack of free valid anchor is an unresolved/ineligible composition result, not permission to orphan a child. Add no broad lifecycle framework.

Update CLI help, the embedded server-independent `--skill`, MCP descriptions and schemas together. Teach ordinary reassignment and the discoverable flow: inspect out-of-date proposal, preview, inspect automatic/conflicting values, explicitly resolve/check, apply, inspect, then ordinary Review changes. Explain that apply does not accept, feedback stays historical, and private Git must never be edited by an agent. No hidden retry, `--force`, accept-latest shortcut, client-local candidate, or per-session proposal state.


## Placement and completion

Resolve semantic objects, composition and Relationships before deriving visible nodes and resolving their whole coordinate pairs. The [placement contract](architecture-placement-amendment-v0.md#5-review-and-reconciliation) defines absent pairs, v2 derived comparison baselines, complete v3 coverage, exact manual choices and target max(A,P). There is no Automatic/null manual resolution in corrected v3. Emit normal final facts relative to A and preserve exact constructor equality.

Use proportionate independent review and real-product tests for changed paths, including exact non-mutating preview, ref races, old-S retry and retained history after restart/GC. Human visual acceptance remains mandatory for the active placement gate. Component/Diagram restoration/deletion, root replacement, identity remapping, raw source restoration, merge-only storage, a resolution registry/session, automatic acceptance and new workflow semantics remain outside this contract.
