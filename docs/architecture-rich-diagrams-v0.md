# Architecture rich Diagrams v0

Status: Approved

Scope: Phase 3.1 — durable manual node placement only

Exact prerequisite: Reconciliation 1 completion `62575d4522d1719ea9f57680d7e4a9afde32a8bd`, final implementation `22e45e977be0de4276b3a40c7ab8683dd058b5bf`.

This is an approved, explicit extension to the living Architecture contract, not approval of the whole rich-Diagram roadmap. It supersedes only the v2-only bootstrap/loading, unpersisted-coordinate exclusions, and corresponding pending/review/reconciliation assumptions in the earlier contracts. The complete portable v2 contract remains unchanged. Change Sets, Reviews, Reconciliation, Agent Access and the current drafting-table workspace continue to govern everything not explicitly extended here. Completed records are not rewritten.

## 1. Recommended boundary and inspected feasibility

One cohesive increment adds placement through ordinary authoring, durable proposals, exact review, reconciliation and acceptance. Shipping stored coordinates without those paths would leave an incomplete product, so these are not separate feature tickets.

Components and Relationships remain Architecture semantic facts. Diagrams own composition and presentation. A canonical appearance is already addressed by **Diagram UUID + Component UUID**; one Component appears at most once per Diagram. No appearance or Relationship identity is added.

The inspected implementation has a practical extension point:

- `internal/architecture/store.go` has one `diagram`/snapshot projection, a closed Diagram serializer, and `ConstructCandidate` which clones composition, rewrites changed Diagram paths and validates the resulting tree. Placement belongs there, not in Component changes or an independent layout store.
- `internal/architecture/change_sets.go` already parses operational versions 1 and 2 and checks exact reconstructed candidate-tree equality. Its current writer emits version 2. New placement facts require explicit version 3 and preservation of old read semantics, not another constructor.
- `internal/architecture/reconciliation_residual.go` already produces ordinary facts relative to Accepted. Placement extends that concrete residual derivation and existing typed conflict values.
- `frontend/src/ArchitectureMap.tsx` supplies explicit positions to Cytoscape's preset layout. Its small `deterministicPositions` function currently assigns a stable-ID circle; dragging is transient. Installed renderer APIs distinguish model coordinates, rendered coordinates and drag completion. Partial pinning does not require a renderer replacement or whole-canvas capture.

The recommended choices are **partial manual placement**, a direct optional Diagram `positions` sequence, bounded integer center coordinates, and one new operational version with explicit target Architecture version. A generic presentation container, appearance IDs, full-layout persistence, and a format-conversion wizard add no demonstrated value to this slice.

## 2. Portable format v3

### 2.1 Tree and manifest

Keep exactly the existing non-recursive closed tree:

```text
architecture.yaml
components/*.md
diagrams/*.yaml
```

All existing ordinary-file, discovery, identity, Markdown, Relationship, membership, hierarchy and closed-schema rules remain. No presentation files or metadata enter another namespace.

The v3 manifest has exactly the existing v2 keys and types, with `version: 3`:

```yaml
format: workbraid-architecture
version: 3
store_id: 11111111-1111-4111-8111-111111111111
project:
  name: Example
  slug: example
root_diagram: 22222222-2222-4222-8222-222222222222
```

`root_diagram` remains the sole root designation. Store UUID, project name/slug rules, file modes and filenames retain their meanings. There is no slug-edit, root-edit or generic version-edit operation.

### 2.2 Closed Diagram schema

V3 adds just one optional top-level key to an otherwise unchanged Diagram:

```yaml
id: 22222222-2222-4222-8222-222222222222
title: Platform
appearances:
  - component: 33333333-3333-4333-8333-333333333333
    role: home
  - component: 44444444-4444-4444-8444-444444444444
    role: reference
positions:
  - component: 33333333-3333-4333-8333-333333333333
    x: 320
    y: -180
```

Exact rules:

- `id` and `title` remain required; `appearances` remains optional with the complete v2 item schema, including home-only `detail_diagram`.
- `positions` is an optional sequence. Missing or `[]` means no manual positions. Null is not a portable sequence or portable position.
- Each position is a closed mapping with exactly required `component`, `x`, `y`.
- `component` uses the existing Component UUID syntax and must resolve to a canonical home/reference appearance in this same Diagram in the complete revision.
- At most one position per Diagram/Component pair. Repeated UUID spellings denoting the same identity cannot evade duplicate detection.
- `x` and `y` must decode as integers, each inclusively within **−100,000 to +100,000**. Reject fractions, floating-point scalars, strings, booleans, missing values, overflow and out-of-range values. WorkBraid writes ordinary decimal integers. `(0,0)` is a real manual position, not an absence sentinel.
- Unknown fields, duplicate keys and malformed YAML are rejected under the existing closed loader rules. A v2 Diagram rejects `positions`, including an empty sequence; only v3 permits it.
- Position sequence order has no domain meaning. Rewrites preserve surviving position order, update existing entries in place, and append newly positioned entries in deterministic Component-ID order. Existing appearance source order is preserved independently.
- No size, route, shape, annotation, kind, renderer, viewport, zoom, scroll, group or future placeholder field is added.

Direct `positions` is smaller than a `presentation.layout.positions` hierarchy with one leaf. The appearance remains composition, while this separate sequence contains only presentation. Nesting coordinates inside an appearance could work, but makes role/home editing unnecessarily responsible for coordinate fields; a separate keyed sequence keeps their independent meanings explicit. Either representation could use the same pair identity; the direct sequence is recommended here.

### 2.3 Coordinate meaning

Coordinates are the **center** of the node in Diagram-local logical presentation units. Positive x is right; positive y is down. `(0,0)` is the logical origin, not a viewport origin. At 100% zoom the initial renderer may map one logical unit to one CSS pixel. Device pixel ratio, pan, zoom, fitting and browser resize never change the canonical numbers.

The bounds give ample ordinary arrangement space and exact safe integer handling in Go/JSON/browser code, without admitting arbitrary magnitudes. They are validation limits, not visible canvas walls or a snapping grid. Manual/manual overlap is allowed: WorkBraid must not move a pinned node to repair an intentional arrangement. No coordinate is clamped or normalized silently.

## 3. Version transition and historical reconstruction

New projects bootstrap directly as v3 with the existing parentless accepted commit, generated store/root UUIDs, slug, project-derived root title and ordinary `diagrams/root.yaml` convention. The root has no appearances or positions. Newly created files remain `100644`.

Valid accepted v2 stays fully navigable **and ordinarily writable**. It has no canonical positions; its map is automatic. Non-layout edits on a v2 proposal remain v2 and reuse its manifest unless a real placement edit explicitly advances that proposal to v3. No v1 portable loading or setup flow returns.

The first successful Set position/drag on a v2 proposal sets that proposal's target Architecture version to 3 in the same mutation. The candidate changes the manifest version and only affected Diagram blobs. Unrelated Diagrams and all independently unedited Components retain exact path/blob/mode. Show one concise notice: **This proposal will save Diagram positions.** The format transition is visible in ordinary exact Review changes; no separate setup commit, wizard or acceptance path exists.

The proposal's target version stays 3 after subsequent Reset position/Reset layout, even if no positions remain. This is a final format value, not a movement-history record. Its now-empty presentation is valid v3. Discard still leaves accepted v2 exact. A no-op reset on an automatic v2 Diagram does not upgrade it or create a proposal. Once Accepted or the current proposal is v3, ordinary authoring never downgrades it. No downgrade action is introduced.

Use the one version-aware loader, passing the exact manifest version into Diagram validation. Shared v2 semantics are shared code, not conversion to an invented canonical v3 snapshot. Historical v2 snapshots have absent positions, not fabricated entries or a rewritten manifest. Unknown portable versions remain unsupported.

This increment explicitly requires preservation of **currently supported exact historical** v2 Change Sets and submitted Reviews, including operational versions 1 and 2. Opening, listing, first/repeated review preparation, or restart must not rewrite their Architecture trees. Preserve existing availability and equality checks; do not revive already-unavailable pre-fix alpha records or reinstate the old Description-loss serializer bug. The earlier alpha waiver is not permission for new placement code to break records that reconstruct exactly at the prerequisite.

## 4. Partial layout and ordinary authoring

### 4.1 Partial rather than whole-Diagram capture

A position pins only its addressed canonical node. All other canonical nodes and all derived boundary nodes remain automatic. Adding a node or dragging one node never captures other nodes' current screen coordinates.

Retain stable-ID-aware automatic layout as a disposable starting point. Place canonical pins at their exact coordinates first, then deterministically place automatic nodes around occupied node/label bounds with a modest visual gap. A simple ordered free-space search around the existing automatic seeds is sufficient; no force simulation, optimizer, spatial database or new graph framework is required. The search must find space rather than silently giving up into an obvious overlap on a small ordinary Diagram. Account for boundary labels as well as ordinary node bounds.

Automatically placed nodes may move when pins or composition change. That movement is not a canonical position change. Never shift pins to satisfy non-overlap. Deliberately overlapping manual nodes remain exact. Fit must include the actual placed nodes, including negative/extreme valid coordinates; the current minimum zoom must not prevent Fit from revealing a valid layout. Auto placement has no persisted correctness dependency on a previous browser session.

Transient reuse of automatic positions is allowed only for the exact Diagram/snapshot context and only if it does not override pins or cause obvious new overlap. It is an optimization, not an authority. Whole-Diagram saved coordinates were rejected because one drag would otherwise create unrelated canonical changes and impede later additions.

### 4.2 Gesture and transaction

In an editable active proposal, dragging a canonical node is ordinary placement editing. From Accepted it uses the existing implicit Change Set behavior. Review changes, submitted historical reviews, Applied records and invalid/non-current read-only contexts are not editable canvases. Derived boundary nodes and comment markers never submit placement mutations.

1. On grab capture exact displayed store, Diagram, Component and either active Change Set/generation or observed Accepted revision. Respect existing unsent-editor/review-composition guards before starting another task.
2. Pointer movement changes local rendering only. Show a grabbed/moving state, distinct from panning. Preserve any open comment notes as view-only context; they do not acquire persisted positions.
3. Crossing a small drag threshold suppresses click selection/drill-down. A click without a drag remains a click. Background drag pans. Escape, pointer cancellation/loss or context departure cancels the local move and restores the snapshot position; it writes nothing.
4. On one completed drag, read **model center** coordinates, not rendered pixels. Round each once to the nearest integer, with exact half values away from zero. Submit one final mutation. Pointer moves, pan, resize, fit and animation callbacks never write Git.
5. Under the existing server synchronization boundary verify exact authority, generation and candidate-relative appearance, apply on a private copy, construct/validate, and CAS-publish one durable generation with the current review cleared. Reject out-of-bounds or stale/ineligible requests without changing the proposal. Only this proposal's review/generation changes.
6. From Accepted, verify exact store + Accepted revision before creating the generated proposal and first generation-1 placement together. No empty ref is published on a failed, cancelled or stale drag. CLI/MCP continue to require explicit Change Set IDs.
7. Render the returned authoritative snapshot. A failed request rolls back the local preview with an actionable explanation. Response loss is not proof of failure: inspect the real proposal/context, never automatically replay a placement against a newer generation. Do not route to a proposal merely because its creation was attempted.

A completed move of an automatic node creates a manual position even if it rounds to the same displayed automatic coordinates; manual versus automatic is a real distinction. A set equal to an already stored pair is a no-op. Reset of absent placement is a no-op. Genuine no-ops create no implicit proposal, generation or review invalidation. While a drop is awaiting its authoritative result, prevent another drag from accidentally reusing its old generation; no durable queue or retry system is added.

### 4.3 Reset and keyboard access

The selected positioned node offers **Reset position**; the selected Diagram offers **Reset layout** when it has manual positions. Reset layout makes every currently visible canonical appearance in that Diagram Automatic in one mutation/generation, not one request per node. This includes inherited base pins, cleared through necessary null overrides. It does not blindly delete the Diagram's internal position facts: required null overrides for removed appearances survive so later re-inclusion cannot resurrect base pins. Backend normalization owns that distinction; the UI shows only current manual/Automatic state, not internal clears. Neither action captures automatic output. Do not add a second **Arrange automatically** action with a different meaning; Reset layout is the automatic-arrangement action for this slice.

For people who cannot drag, the existing Component contextual pane offers a compact **Position** action with labelled X/Y integer inputs and **Keep position**, plus Reset when applicable. It edits the selected Component **in this Diagram**, not every appearance. This is also useful for precise adjustment and gives keyboard access without an editor toolbar, arrow-key event persistence or a new interaction mode. Controls use normal drafting-table spacing and clearly underlined/button actions. No future disabled tools are shown.

Do not auto-fit or recenter after each drop; keep the user's viewport stable. Placement survives Diagram/proposal navigation, reload and restart because it comes from the snapshot, not the component lifecycle. Fit/zoom, neutral selection, routing, proposal design document and comments dock retain their existing roles.

## 5. Concrete durable proposal representation

### 5.1 Operational version 3

Keep the existing envelope, ref ownership, `change-set.yaml` version, non-chained state commits, proposal Markdown and Review metadata unchanged. `changes.yaml` gains **workbraid-change-state version 3**, with the seven version-2 sequences unchanged and these additional required fields:

```yaml
format: workbraid-change-state
version: 3
architecture_version: 3
components: []
new_component_homes: []
detail_diagrams: []
diagram_titles: []
home_moves: []
references: []
detail_reassignments: []
node_positions:
  - diagram_id: 22222222-2222-4222-8222-222222222222
    component_id: 33333333-3333-4333-8333-333333333333
    position:
      x: 320
      y: -180
  - diagram_id: 22222222-2222-4222-8222-222222222222
    component_id: 44444444-4444-4444-8444-444444444444
    position: null
```

`architecture_version` is required integer 2 or 3: the final requested portable version, never below the proposal base's version. It is server-owned, not a public version selector. Target 2 requires empty `node_positions`. Target 3 may have none. This one scalar preserves a deliberate v3 transition after reset and makes a format-only residual exactly representable; deriving version solely from nonempty coordinates would lose that state.

`architecture_version` is a final requested Change Set value, not derived from whether `node_positions` is empty. The sequence v2 base → Set position → v3 → reset every position still produces v3: a valid real format-only residual. State minimization may remove redundant position overrides but must retain the deliberate v3 target. A v2 no-op reset remains exactly a no-op and never upgrades.

Each `node_positions` item contains exactly `diagram_id`, `component_id`, `position`; at most one per pair. `position` is either a mapping containing exactly required bounded integer `x`/`y`, or explicit YAML null meaning final automatic placement. Omission of an item means inherit base placement subject to final composition, **not** clear. No `old`, delta, timestamp, event, appearance ID, movement sequence or generic operation kind exists.

A positive position must resolve against the complete final candidate's canonical appearance. A clear is a concrete absence override, not a position for an absent Component: it may address an appearance present in the base or current composition even when that appearance is now removed. Unknown Diagram/Component identities are rejected. Remove redundant overrides equal to the base where safe, but retain necessary explicit absence to prevent base positions reappearing after subsequent composition edits. New entries can be sorted by the identity pair for deterministic operational serialization; that creates no authoring-order semantics.

Use minimum typed state alongside `CandidateComposition`/the existing pending object; do not force coordinates into `ComponentChange`. The one constructor receives both composition and presentation facts. No parallel node-layout object store, operation algebra or raw source override is introduced.

New proposals and actual mutations write operational version 3. Versions 1 and 2 retain their exact six/seven-sequence schemas over portable-v2 bases/candidates, with no placement overrides or format transition. They do not gain default serialized fields on read; operational v3 is required to represent a portable-v3 candidate. An unchanged old `changes.yaml` tree entry can be reused when only establishing a review binding; a real mutation may write new operational version 3 without rewriting an unchanged candidate. Preserve old closed-schema rejection and old constructor outputs when no placement/version transition is requested.

### 5.2 Composition normalization

Derive final membership and links through the existing concrete composition logic before applying placement. Use the same logic in mutation, construction, reconstruction and reconciliation; no browser membership algorithm.

- A same-pair reference→home conversion retains that pair's position, whether inherited or pending. Any complete joint composition result that keeps the appearance in place but changes home/reference role likewise retains it. A role alone is not placement identity.
- Removing a home/reference appearance removes its positive pending position and its resulting canonical position in the same mutation. When its base had a position, keep a null override as needed so later explicit re-inclusion does not resurrect obsolete coordinates. A candidate-only position with no base entry can simply be removed.
- Moving home retains a destination reference's existing position when converting it to home, but copies nothing from the source. A destination with no previous appearance starts automatic. Other reference appearances and their positions remain independent.
- Removing then deliberately showing a Component again in the former Diagram starts automatic unless a new Set position is made. This removal/re-inclusion rule concerns an appearance that actually became absent in the kept proposal, not an in-place home/reference conversion. Home round trips must not resurrect a base coordinate via reconstruction. This is net final absence state, analogous to the existing reference non-resurrection rule, not a request log.
- Detail-link reassignment and subtree reparenting leave positions **inside** unchanged Diagrams exact: coordinate spaces are Diagram-local. Moving an anchor's home removes only that anchor's old-Diagram position; it does not relocate every descendant's coordinates.
- Reset layout makes currently visible canonical appearances Automatic and folds necessary clears into the same state, preserving existing non-resurrection null overrides for absent appearances. For example, base D/C is pinned, C is removed with a retained null, and other visible nodes in D are reset: the retained null must survive so re-showing C stays Automatic. There is no durable reset command or future-node reset policy.

The constructor independently filters inherited base positions for appearances absent from final composition and rejects orphan positive overrides rather than using them to create membership. Normal authoring performs the concrete normalization before persistence. Invalid other authored work remains recoverable with its exact facts; it does not produce a partial layout canvas. Repair it normally before issuing placement against a complete projection. Existing invalid-state loading must not be weakened to hide orphan positions.

### 5.3 Fidelity and tree equality

Construct from the exact base tree. A placement-only edit rewrites only the relevant Diagram(s), plus the manifest only for v2→v3. Preserve untouched entries/blobs/modes exactly, including Components, child Diagrams and unrelated layout. A rewritten Diagram preserves ID, path, regular-file mode, every unrelated semantic value, appearance order and surviving position order. Lexical YAML quoting/comments/whitespace need not be preserved inside that rewritten blob. Omit empty `positions` when writing a changed Diagram; reuse untouched empty-field blobs exactly. Manifest upgrade changes only its semantic version value and preserves every other field value and the existing regular-file mode; it need not preserve lexical YAML spelling inside the rewritten manifest.

The serialized result is loaded by the one v2/v3 validator. For every valid active/applied/historical record:

```text
ConstructCandidate(exact base, concrete changes).Tree == stored candidate_tree
```

No candidate tree is accepted merely because a different merge/layout tree looks equivalent. Historical v2 serialization with no new fields must retain exact existing tree equality, not only semantic equality. No frontend coordinate cache can satisfy this invariant.

## 6. Exact review and historical feedback

Extend the existing immutable review comparison with `node_positions`, keyed by Diagram ID + Component ID and carrying exact optional Before/With pairs. Show **Position changed**, with **Automatic** versus the exact pair where useful. New pin, changed pair and reset are presentation changes, not Component content, Diagram membership/title, home movement or Relationship fact deltas. A pair changing role with unchanged position has no position delta. Appearance removal may show that its position was removed in that composition context, but never creates a removed-Component ghost.

Mark affected Diagrams and canonical nodes so a layout-only proposal is discoverable. Selecting the change focuses that Diagram/Component and the relevant Diagram-file region in the complete raw diff. Position comparison is a field of the existing bound review response, not a new layout-diff authority or persisted review object. Actual content/composition/Relationship changes in a mixed proposal keep their independent existing classifications.

Before/With use their own exact snapshot coordinates; a v2 side is automatic. Neither side borrows the other's pins or a live Accepted position. Tree, index, docs, titles, topology, boundaries and placement switch together. Preserve a common logical viewport while toggling the same Diagram so automatic per-side fitting does not visually cancel the very movement being reviewed. An initial fit may include both bound layouts; explicit Fit remains a view action. Candidate-only Diagram fallback stays unchanged.

Immutable review canvases do not issue placement edits. Ordinary **Continue editing** returns to the proposal. Existing Component/Diagram/composition comments provide enough anchoring; no coordinate/comment identity is added. Historical submitted feedback reconstructs its old v2/v3 snapshots and coordinates after later placement edits, reconciliation, application, discard and GC. No annotation follows a Component onto a newer review generation. A clearly failed map still leaves exact diff review/acceptance usable; persisted coordinates remain inspectable through structured reads and the diff.

## 7. Reconciliation extension

### 7.1 Placement unit and composition precedence

Keep exact S/B/A/P, known-current Accepted, non-mutating Preview/Check, exact Apply ref verification, one new generation, retained proposal identity/Markdown, cleared current binding and no Accepted write. Placement is one **whole coordinate pair** for `(Diagram ID, Component ID)`, never x from one branch and y from the other. Its comparison context distinguishes three states on each exact side:

- `not_applicable`: the canonical appearance does not exist on that side;
- `automatic`: the appearance exists but has no persisted position;
- `manual(x,y)`: the appearance exists with that exact pair.

`not_applicable` is request-local reconciliation context only. It is never persisted in portable Architecture or `changes.yaml`, and must not be treated as Automatic. Automatic is distinct from `(0,0)`. Placement is not involved in the Relationship tuple/multiplicity comparison.

Resolve semantic existence/composition first. For a pair absent in the chosen final composition, omit its position and any now-irrelevant placement conflict, explicitly reporting that placement was removed with the appearance. Do not request an anchor/restore/reference solely to preserve coordinates. If composition is still unresolved, defer only the placement choices dependent on it while returning independent conflicts normally. Check choices recomputes these dependencies; an obsolete/inapplicable supplied choice is rejected with target-not-eligible context, not silently applied to another node.

If B contains the pair and final composition retains it, use B's actual Automatic/manual value as the placement base. A branch which removed the appearance contributes no placement value when final composition keeps the other branch's appearance. If only A retains it, use A's placement; if only P retains it, use P's. Do not manufacture a reset or a conflict from the absent branch. If both retain it, apply the ordinary three-way rule below.

If B has no appearance and final composition introduces the pair, use Automatic as the placement baseline for this new appearance. A missing branch contributes no value; use the sole present branch if only one introduces it. If both introduce it, compare their actual Automatic/manual values against that Automatic baseline. Thus Automatic plus a new manual pin combines, while two different manual pins conflict. If joint composition introduces a pair present in neither A nor P, it starts Automatic; do not import another Diagram's coordinate.

With both branch appearances present, using the base value just defined:

```text
if A_position == B_position: choose P_position
else if P_position == B_position: choose A_position
else if A_position == P_position: choose that value
else: require an explicit placement resolution
```

| B | A | P | Chosen final composition | Placement result |
| --- | --- | --- | --- | --- |
| Manual (100,100) | Not present | Manual (300,300) | Keep P appearance | (300,300), no placement conflict |
| Manual (100,100) | Manual (300,300) | Not present | Keep A appearance | (300,300), no placement conflict |
| Not present | New Automatic | New manual (300,300) | Pair present | (300,300), proposed-only placement |
| Not present | New manual (200,200) | New manual (300,300) | Pair present | Placement conflict against Automatic baseline |
| Any | Any | Any | Pair absent | No position; placement conflict pruned |

Independent nodes and same-result moves combine; divergent moves or reset-versus-move on retained appearances require a choice. If the appearance is in a different Diagram, do not transfer coordinates; merge only that destination pair's own values. Position-only differences do not create object identity collisions: the existing complete **semantic** identity-collision checks remain unchanged, followed by placement comparison. Divergent semantic same-new-UUID objects remain unsupported with no replacement choice. External object lifecycle remains bounded by Reconciliation 1; position alone does not authorize restoration or obstruct a chosen valid removal of an appearance.

### 7.2 Closed placement resolution

Add one request-local locator to the existing reconciliation vocabulary:

```json
{"kind":"node_position","diagram_id":"<uuid>","component_id":"<uuid>"}
```

Three-side request-local context uses `{ "state": "not_applicable" }`, `{ "state": "automatic" }`, or `{ "state": "manual", "position": { "x": 320, "y": -180 } }`. Only manual context includes a position. The actual B context remains Not present where applicable; the separate Automatic comparison baseline for a newly introduced appearance must not falsify what existed in B.

A resolution retains the existing `locator` and `choice: accepted | proposed | manual`. Only manual adds `value`, whose exact shape is `{ "position": null }` for Automatic or `{ "position": { "x": 320, "y": -180 } }`. It never accepts `not_applicable` as a placement choice. Missing, duplicate, fractional, extra or conflicting fields are `invalid_request`; an absent/non-canonical target or absent-side choice after semantic choices is `target_not_eligible`. Unresolved choices reuse `reconciliation_unresolved`, invalid complete candidates `validation_blocked`; no layout error subsystem is needed.

The browser shows **Position of Worker in Runtime**, **Original / Accepted / Proposed**, Not present here, Automatic or exact X/Y as appropriate, and meaningful side choices or the compact coordinate editor/Automatic choice. It does not label an absent appearance Automatic. Optional small side previews are assistive; no spatial merge dashboard, draggable conflict solver, conflict history or persistent session. Existing unsent resolution guards apply.

### 7.3 Portable version reconciliation

The output target is **max(A's portable version, P's portable version)** across supported 2/3 inputs. Ordinary P never downgrades its base. B supplies comparison facts, not a conversion instruction. A legitimate external version replacement remains loadable; using the same rule with ordinary P still preserves the proposal's v3 floor. Store identity and format identifier remain invariants; project metadata/root continue to follow existing Reconciliation rules. Version difference alone is not a conflict.

| B / A / P | Result version | Placement consequence |
| --- | --- | --- |
| 2 / 2 / 3 | 3 | P's placement combines with A's semantic work; no old coordinates are fabricated. |
| 2 / 3 / 2 | 3 | Retained v2 appearances are Automatic, not absent. A-only pins survive under the composition-aware rules; newly introduced pairs use the Automatic baseline. |
| 2 / 3 / 3 | 3 | Compare independent/same/divergent optional pairs normally. |
| 3 / 3 / 3 | 3 | Moves and explicit resets compare against the exact original pins. Empty result stays v3. |

Also retain 2/2/2 as 2 for non-layout reconciliation. A/P may both be v3 with empty positions; max-version remains 3. If P brought only a v3 format transition and A is already v3 with identical Architecture, the residual may be empty. If A is v2 and P's target is v3 with empty positions, the manifest transition is a real residual tree change, not an empty proposal. No separate migration conflict or setup action is introduced.

### 7.4 Exact residual construction

After resolving final semantic composition, compute final pair values, prune absent pairs, and emit target version plus minimal set/clear overrides **relative to A** alongside the ordinary content/composition facts. Use A's paths, modes, untouched blobs and surviving orders. P-new identities/paths use the existing collision-safe allocation. No raw source or tree override is added.

Call the same `ConstructCandidate(A, changes)` and verify its loaded composition/positions equal the resolved facts, then enforce exact stored-tree reconstruction again. The version floor and explicit Automatic overrides are normal authoring facts, including when positions were all cleared. If no tree difference remains, retain the active proposal and Markdown exactly as the existing empty-residual rule requires. Historical submitted Reviews remain on old parents and positions. Old-S Apply retries remain `change_set_state_mismatch`; placement adds no receipt/session/ref/history or second acceptance path.

## 8. Agent Access parity

Keep additive `workbraid-agent-v2`, one loopback process, shared typed application operations, thin CLI and stateless `workbraid [--server <loopback-url>] mcp`. No client reads/writes private Git.

| Action | CLI | MCP | Local API under `/api/agent/v2` |
| --- | --- | --- | --- |
| Inspect Diagram placement | `diagram positions` | `diagram_positions` | `/diagrams/positions` |
| Set final pair | `diagram set-position` | `diagram_set_position` | `/diagrams/set-position` |
| Reset one | `diagram reset-position` | `diagram_reset_position` | `/diagrams/reset-position` |
| Reset all in Diagram | `diagram reset-layout` | `diagram_reset_layout` | `/diagrams/reset-layout` |

The read requires `store_id`, `diagram_id`, and optionally `change_set_id`. Omission selects Accepted, never a process-wide current proposal. Explicit ID can inspect a valid active/Applied proposal. Return exact revision or state/base/generation/candidate context and each canonical appearance's Component ID, title, home/reference role and `position: null | {x,y}`. Do not return transient automatic coordinates as if persisted. Missing/invalid projections use existing target/validation/unavailable results. `architecture inspect`, Change Set inspect, exact review and historical review projections also expose snapshot placements, so the granular read is convenience, not another interpretation.

Mutations require exact `store_id`, `change_set_id`, `generation`, `diagram_id`; set/reset-one additionally require `component_id`, and set requires integer `x`,`y`. CLI uses matching kebab-case flags including `--x=-180` for negative values. Reset-all is one mutation. MCP uses closed typed fields with bounds, not arbitrary presentation JSON. Browser Accepted-origin mutation uses the existing store/accepted-revision implicit-create path; it is not an agent convenience bypass.

Return exact project/store, proposal ID/state/base/generation/candidate, portable version, addressed IDs, resulting placement or the Diagram's reset result, `unchanged` where applicable, validation and authority context. Reuse existing wrong-store/generation, non-current, out-of-date editability, target, validation, operation-failure and review-invalidated rules. Reject invalid integer requests as `invalid_request`; a boundary-only/absent appearance is `target_not_eligible`. Nothing mutates on rejected requests.

Update help, embedded server-independent `--skill`, MCP descriptions/schemas and reconciliation resolution discovery together. Explain center units, Automatic versus `(0,0)`, per-Diagram identity, exact generations, resets, v2 first-placement review, and ordinary Review → exact binding → Update. Agents author final coordinates and never emulate a drag or edit YAML. Placement reconciliation uses the existing preview/apply commands and one new closed value, not separate layout tools.

## 9. Verification and later boundaries

The packet combines real Git/transport/reconciliation verification with a **human-first visual gate**. Exact coordinates and refs can be proven technically; pleasing or usable manual placement cannot be delegated to a model-generated coordinate arrangement. Require real pointer interaction, mixed automatic/manual nodes, same logical Before/With frame, reset, normal acceptance, restart and one real parallel-placement reconciliation with a human-chosen result. Weak-agent discovery is bounded to placement reads/set/reset through one transport; parity for both transports is automated.

Protect v2 historical trees/review parents with real object/ref checks across restart and bounded GC, not a no-op mock or an adapter. Test direct-v3 initialization and both-format ordinary authoring while leaving completed Gate records untouched. Living executable assertions about v2 bootstrap must change proportionately; still-valid authority, source-byte, review, catalog and restart invariants remain.

No case inspected requires new appearance/Relationship identity, a second candidate representation or reinterpretation of historical v2. Exact schema keys, bounds, sticky proposal version, net placement absence, keyboard alternative and command names above are approved for the one Phase 3.1 increment. If implementation exposes an unrepresentable residual or exact historical-tree regression, stop rather than bypass equality or add a merge-only format.

Deferred, not designed here: Phase 3.2 richer node presentation/possible sizing; Phase 3.3 routing after an explicit presentation-identity decision for exact parallel Relationship facts and derived boundary edges; Phase 3.4 shapes/annotations/richer authoring after demonstrated need. This is an indicative order, not approval of those features. Sizing is not a prerequisite for placement because centers are stable anchors; routing could be reprioritized only after its identity decision. UML/other semantic Diagram kinds and alternate/isometric renderers remain later separate decisions.

Exclude sizing, routes/bend points, Relationship/appearance IDs, canonical boundary positions, grouping, shapes/annotations, generic property graphs, graph editing frameworks, generic operation/event systems, whole-layout capture, persisted automatic coordinates, viewport state, undo/redo/history, multi-select, snapping/guides/rulers/layers/minimaps, graphical membership/Relationship authoring, layout-specific comment identities, proposal/review lifecycle changes, new refs/registries/SQLite/worktrees, migration wizards, v2 rewrites on load, remote collaboration, permissions, other verticals and later Phase 3 implementation.
