# Architecture placement — stable visible nodes

Status: Phase 3.1 delivered; Phase 3.2 sizing approved for implementation

Section 9 extends this contract with the human-authorized Phase 3.2 decision from proposal `bbc915b9-4e06-47c0-ad96-db5f8f166435`, generation 3, reviewed state `f5a44f18557d4070de57dad9fb7e185f2126ce61`. It supersedes the earlier exclusions of sizing and native-version statements below. Historical v2/v3 and operational v1–3 guarantees remain binding. Implementation authorization leaves that Markdown-only proposal active and changes no acceptance lifecycle.

Sections 1–8 record the Phase 3.1 correction. The human rejected the partial-pinning implementation during its visual gate: Queue could not be dragged because it was a boundary node, and arranging one node rearranged other nodes. Technical checks alone did not establish product acceptance. The corrected implementation subsequently received explicit human visual PASS and passed post-acceptance restart verification.

This living contract incorporates the approved stable-canvas correction and replaces the superseded partial-pinning design. It preserves Accepted/ref authority, exact source fidelity, one constructor, supported historical formats and human visual acceptance. [Architecture](architecture-v0.md), [Proposals and Reviews](architecture-proposals-v0.md), [Reconciliation](architecture-reconciliation-v0.md) and [Agent Access](architecture-agent-access-v0.md) supply the shared contracts. The completed execution plan remains in Git history; later scope is in the [roadmap](roadmap.md). Original failed-gate evidence remains intact.

## 1. Product behavior

- Every visible Component node can be dragged, including a **Lives in** node. Its semantic membership does not change when it moves.
- Moving one node leaves every other node at its existing coordinate, during the gesture, after the response, and after navigation/reload/restart. No background relaxation, repacking or automatic fitting follows a drag.
- A newly visible node receives an initial position once as part of the authoring mutation that makes it visible. Existing nodes stay put.
- **Auto-layout** deliberately arranges all visible Component nodes in the selected Diagram and saves those coordinates as one ordinary mutation. Other Diagrams stay unchanged.
- Remove Reset position, Reset layout and the manual/Automatic distinction from the new placement UX. Keep precise X/Y editing and Fit/zoom/pan as separate view actions. Auto-layout replaces reset-to-Automatic, rather than adding another similar control.
- Opening Review changes never initializes or saves positions. Review and historical canvases display their exact immutable snapshot and remain read-only.

No force simulation or replacement graph-editor framework is needed. The existing renderer already consumes explicit coordinates. Manual overlap is allowed; even label/content changes must not silently reposition surrounding nodes.

## 2. Position address and closed portable v3

Retain the existing `positions` sequence with closed entries `{component, x, y}`, integer center coordinates bounded inclusively to −100000..100000. Position identity is still **Diagram UUID + Component UUID**, with no appearance or boundary ID.

For Diagram D, derive its visible Component set using the existing projection rule:

1. Include every canonical home/reference Component in D.
2. For every global Relationship with exactly one endpoint canonically included in D, include its other endpoint as one coalesced boundary node.
3. Do not recursively expand external nodes or add Relationships between two external-only nodes.

A valid v3 Diagram has exactly one position for every Component in that visible set, and no positions for other Components. Missing/empty `positions` is valid only when the visible set is empty. Duplicate entries, portable nulls, invalid coordinates and unknown fields remain invalid. Complete validation uses the same loaded Components/Relationships/composition, not another graph interpretation.

A stored boundary-node coordinate is Diagram presentation over a real Component. It neither creates a canonical reference nor gives the derived boundary an identity. Parallel crossing Relationships continue connecting to the same node with exact labels/direction/multiplicity.

### Visibility transitions

If a pair remains visible, preserve its coordinate through home/reference/boundary presentation changes. In particular, Show component here and Stop showing here do not move a node that remains visible through Relationships. Home movement may leave a visible boundary at the old location; retain that Diagram-local coordinate. A destination already displaying the Component keeps its own coordinate. Never transfer coordinates between Diagrams.

If a pair disappears completely, remove its coordinate. A later reappearance receives a fresh initial placement, not an implicitly resurrected base position. No hidden portable positions are retained. Reparenting a detail Diagram leaves all coordinates inside that Diagram/subtree unchanged.

Relationship or composition edits can now legitimately add/remove positions in affected Diagrams without changing their canonical membership. The exact diff shows those presentation changes; they are not extra Relationship facts. Untouched Diagram entries and all independently unedited Component blobs/paths/modes remain exact.

## 3. Initialization and version boundary

The approved correction amends portable/operational v3 within the not-yet-accepted Phase 3.1 increment. Preserve failed-gate artifacts as evidence, but do not build compatibility for this disposable partial-pinning v3 trial or introduce v4 solely for it. Do not delete or convert the current human fixture automatically. Use fresh data for the corrected visual gate.

Completed portable v2 and operational v1/v2 historical reconstruction remain exact, including submitted Reviews. V2 remains normally writable for non-placement work. Reads never rewrite its trees.

The first Set position or Auto-layout against a v2 proposal advances it to v3 and assigns positions to every visible node in every Diagram in that candidate. This deliberately replaces the earlier promise to touch only the selected Diagram during first placement: full v3 coverage requires completing all Diagrams. Reuse the same deterministic initial layout used to display the v2 snapshot, then apply the requested position/selected-Diagram arrangement so an initial drag does not reshuffle its peers. Make the format/position initialization visible in ordinary review. No wizard or separate acceptance is added.

Native new projects remain v3; new nodes get positions immediately. Once the proposal or Accepted is v3 it does not downgrade. V2 has only a read-time layout fallback, not invented canonical coordinates. New placement UI does not expose an ongoing Automatic mode.

## 4. One ordinary authoring model

Retain operational v3 `architecture_version` and final `node_positions` facts. A non-null pair specifies its exact final coordinate. A null is only an internal removal/non-resurrection override for a no-longer-visible pair; it must not leave a visible v3 node unpositioned. Missing overrides inherit base coordinates subject to final visibility. A newly visible pair receives a concrete set fact, replacing any required absence override.

The synchronized ordinary mutation path resolves final composition/Relationships, retains existing coordinates, allocates only missing newly visible nodes, and materializes those coordinates into the normal typed facts before publishing a valid candidate. Use a small deterministic bounded free-space search, stable-ID tie-breaking and sensible proximity where practical. Never capture arbitrary browser node positions or move existing nodes to make space. A coordinate assigned by the initial-placement algorithm is subsequently just a coordinate, not an ongoing algorithm-controlled state.

Candidate reconstruction must replay final facts, not rerun initial placement or Auto-layout. A future algorithm improvement cannot alter an existing candidate tree. Preserve exactly:

`ConstructCandidate(base, changes).Tree == stored candidate_tree`

Invalid pending authoring remains retainable/correctable without a fabricated partial canvas. When correction yields a valid complete proposal, materialize its missing coordinates within that correcting mutation. Review preparation must not repair or initialize it. Reuse the one constructor's composition logic; an inability to produce ordinary final facts is a stop, not permission for a second builder.

One pointer-up remains one exact-precondition mutation/generation; pointer movement is local. A stale/failed Accepted-origin drag creates no empty proposal. An unchanged set is a no-op. Auto-layout uses the same state checks, writes final pairs once, and is a no-op when all results already match. There is no event log, layout session, position registry or new acceptance path.

## 5. Review and reconciliation

Position changed applies to any visible Diagram/Component pair, including boundaries. It never implies Component content, membership or Relationship change. Before/With use their own coordinates in a common logical frame. Existing historical comments remain exact; no new comment identity is required.

Resolve semantic existence/composition/Relationships first, then derive final visibility and reconcile placement. An absent visible pair is not a reset. If the final pair is absent, prune its placement/conflict. If only one branch retains the pair, retain that branch's coordinate. If both retain it, apply the normal whole-pair three-way rule against B. Same-result moves combine; different moves of the same node conflict; different nodes merge independently. Boundary-to-ordinary conversion alone is not a position change.

For a pair absent in B and introduced on both branches, identical coordinates coalesce and different coordinates conflict. There is no persisted automatically-assigned-versus-manually-assigned provenance from which to infer priority. Manual resolution supplies exact x/y, never Automatic/null. If final visibility introduces a pair on neither branch, allocate its initial coordinate deterministically during Preview/Check and materialize it as ordinary facts on exact Apply.

Target portable version stays `max(A,P)`: 2/2/2 remains 2; 2/2/3, 2/3/2, 2/3/3 and 3/3/3 produce complete v3 placement. V2 sides have no stored placement and cannot contribute a reset of a v3 coordinate. For a pair visible in a v2 B, use B's deterministic displayed fallback as the comparison baseline; a retaining v2 branch contributes no placement edit. Distinguish this derived comparison baseline from stored coordinates in response context. If B lacked the pair and only one branch supplies a stored coordinate, retain it. Fill any remaining v3 coordinates without moving selected existing ones.

Preview/Check remain non-mutating. Exact Apply recomputes under current S/B/A/P authority, emits ordinary residual facts relative to A, verifies exact constructor-tree equality, changes only that proposal's generation/base and clears its review binding. Accepted and submitted Reviews remain untouched. No placement session, merge-only tree or receipt is introduced.

## 6. Browser and agent surface

Keep drag threshold/click suppression, Escape/cancellation, distinct pan, integer model-coordinate rounding, stable viewport, pending-response generation safety and rollback behavior. Boundary selection must offer local positioning without forcing navigation home; opening its home remains an explicit navigation action. Review/history do not persist drags. Preserve the existing pane/dock visual direction.

Agent Access retains `diagram positions` / `diagram_positions` and `diagram set-position` / `diagram_set_position`, now covering every visible Component node. Replace trial reset commands/tools with `diagram auto-layout` / `diagram_auto_layout` and `/api/agent/v2/diagrams/auto-layout`. Use exact existing store/Change Set/generation preconditions. No compatibility aliases for unaccepted reset behavior. Update help, embedded skill, schemas and tests together; the established agent-v2 authority/envelope remains unchanged.

Inspect distinguishes persisted v3 coordinates from derived v2 fallback and reports ordinary/boundary context honestly. No generic presentation JSON, browser-owned layout authority, automatic acceptance or membership mutation is added.

## 7. Corrected verification and continuation

The corrected increment completed independent review and the human gate. Preserve failed-gate and technical evidence. The following verification requirements remain regression guidance; completion does not authorize Phase 3.2.

Prioritize a real built-browser interaction check before repeating expensive downstream verification: drag Queue and ordinary nodes; observe every other coordinate and viewport through grab/drop/server response; navigate away/back. Do not ask the human to accept another partial-pinning variant.

Required bounded production evidence:

- All visible nodes have persisted v3 coordinates, including boundary nodes; absent/duplicate/missing coverage rejected by the one validator.
- Dragging either kind changes only that pair and one generation; no pointer-move writes, peer movement, recentering or accidental navigation.
- New Component/reference/crossing Relationship adds only required initial coordinates; retained visible pairs stay exact across boundary/ordinary transitions. Remove/reappear cannot resurrect an inherited position implicitly.
- Auto-layout saves all selected-Diagram results once, changes no membership/Relationships and leaves other Diagrams exact; restart reproduces those coordinates.
- V2 non-placement/history remains exact; first placement completes v3 coverage without changing Component bytes, and never initializes during Review/read.
- Before/With placement, historical feedback, parallel same-node conflict, independent-node merge, mixed v2/v3 results and exact residual reconstruction remain truthful.
- Updated CLI/MCP/skill parity and bounded fresh weak-agent discovery; ordinary checks and independent private-Git/history/restart verification.

The final human gate still covers actual dragging, Auto-layout, exact review/acceptance, restart and parallel-placement reconciliation. Technical green does not waive that gate. Stop after explicit Phase 3.1 PASS.

Still excluded: sizing, routing/bend points, shapes, annotations, appearance/Relationship IDs, graphical membership editing, multi-select, snapping, undo/history, viewport persistence, another graph framework and Phase 3.2.

## 8. Exact schema and fidelity details

The manifest remains closed with exactly `format: workbraid-architecture`, integer `version: 3`, `store_id`, `project: {name, slug}`, and `root_diagram`; all values other than version retain the Architecture contract's types/identity rules. The accepted tree remains only the manifest, non-recursive Component Markdown and Diagram YAML. Root designation comes only from the manifest.

V3 Diagram keys are exactly required `id`, `title`, optional `appearances` with unchanged home/reference/detail schema, and `positions`. Position coverage is required exactly for the derived visible set. An empty Diagram may omit positions or use `[]`; portable null is invalid. V2 rejects positions even if empty. Each position requires exactly Component UUID, x and y; repeated UUID spellings cannot evade pair-duplicate validation. Reject unknown/duplicate keys, null/missing/fractional/floating/string/boolean coordinates, overflow and values outside −100000..100000. `(0,0)` is a real coordinate. No size, route, viewport, algorithm provenance or boundary identity is added.

Coordinates are Diagram-local node centers, positive x right and positive y down. Pan, zoom, device scale and fitting never change them. One completed pointer drop rounds model values once to nearest integer, exact halves away from zero; do not clamp. Fit must reveal all valid bounded positions. Manual/manual overlap is allowed and is not repaired by moving peers.

Surviving position order is preserved; updates occur in place and new pairs append in deterministic Component-ID order. Preserve appearance order independently. Placement rewrites only affected Diagram blobs, except first v2→v3 placement must initialize all Diagrams and update the manifest. Preserve every untouched path/blob/mode and independently unedited Component. Rewritten Diagrams retain IDs, paths, regular-file modes and unrelated semantic values; no lexical YAML preservation framework is required. Manifest upgrade preserves every other field value and regular-file mode.

Operational `changes.yaml` version 3 is closed: required `format: workbraid-change-state`, integer `version: 3`, integer `architecture_version: 2|3`, and required sequences `components`, `new_component_homes`, `detail_diagrams`, `diagram_titles`, `home_moves`, `references`, `detail_reassignments`, `node_positions`. The first seven item schemas remain exact. Each position fact has exactly `diagram_id`, `component_id`, `position`; at most one per pair, with exact bounded `{x,y}` or the internal null described above. Target cannot be below base; target 2 has no position facts. The unchanged envelope and Review versions/ref namespaces remain authoritative. Supported operational v1/v2 do not acquire these serialized fields on load or unchanged review preparation.

Placement conflict locator remains closed `{kind: "node_position", diagram_id, component_id}`. Side/manual resolution uses the existing `{locator, choice}` union; `choice: "manual"` requires exactly `value: {position: {x, y}}`. Null/Automatic/not-applicable is not a manual choice. Reject duplicate/unknown/fractional/irrelevant fields as `invalid_request`, absent-side or non-visible targets as `target_not_eligible`, incomplete choices as `reconciliation_unresolved`, and invalid complete results as `validation_blocked`. Response context must distinguish absent visibility, stored v3 coordinates and derived v2 comparison fallback without persisting provenance or inventing a coordinate reset. These are extensions of the existing reconciliation response, not a new durable representation.

## 9. Complete visible-node sizing (Phase 3.2)

Every visible Diagram UUID / Component UUID pair has one complete size in portable v4, including home, reference and boundary presentation. Width is an integer in 80–1600 inclusive; height is an integer in 48–1200 inclusive. These are logical outer shape bounds excluding stroke and external captions. Native new projects use v4. New ordinary/reference nodes use 200×96; new boundary nodes use 224×112. Legacy v2/v3 display remains 116×54 ordinary/reference and 104×62 boundary, with exact existing coordinates or the unchanged v2 coordinate fallback. Reads never upgrade.

The closed v4 manifest differs from v3 only by integer `version: 4`. A v4 Diagram adds `sizes`, a sequence of closed `{component, width, height}` entries. Exactly one entry covers each visible Component and none covers an invisible Component. Empty visibility permits omission or `[]`; null, duplicate IDs/keys, missing dimensions, unknown fields, non-integers, overflow and out-of-bounds values are invalid. V2/v3 reject sizes. Position coverage and all unchanged semantic schemas remain exact. Surviving size entries keep their order; changes update in place and new pairs append in stable Component-ID order. Preserve untouched entries/blobs/modes; an affected Diagram retains path, mode, identity and unrelated semantic values.

Retained visibility preserves both dimensions through home/reference/boundary conversion. Disappearance removes size; reappearance receives a fresh current-role default. No hidden sizes or cross-Diagram transfer exist. Restore default size writes the current-role concrete default once and never changes coordinates or enables automatic sizing.

After ordinary identity/generation and integer/bounds validation, compare requested dimensions to the current displayed pair. Equality preserves portable/operational version, tree, generation and Review, including first v2/v3 requests and Restore default. Only an actual size change upgrades a legacy candidate: initialize every visible pair to that side's displayed legacy dimensions, then change the requested pair. V2 also materializes its exact displayed coordinates. Ordinary non-sizing work retains its portable version; v4 never downgrades. Invalid pending work is corrected through ordinary authoring before a complete canvas exists; read/review cannot initialize it.

The one constructor strictly replays final facts. Ordinary authoring alone materializes missing sizes and initial positions before publishing, including correction of invalid pending work. No default/allocation algorithm runs during strict reconstruction. Exact supported v2/v3 and operational v1–3 candidates, source entries/modes and immutable review parents must survive later work, restart and GC without a compatibility waiver.

Resize preview keeps the selected center, every peer and viewport fixed. One visible corner handle on the selected editable node changes width/height symmetrically; its screen hit area remains usable under zoom and never triggers node drag, pan or navigation. One release rounds logical dimensions once (nearest integer, halves away from zero) and submits one exact-generation typed mutation. Invalid results are rejected, not clamped. Escape, pointer cancellation, blur or context departure restores the snapshot without a write. Numeric width/height fields use the same operation. Review/history are read-only. No aspect lock, background reflow, automatic Fit, whole-Diagram reset or title-driven resizing exists; deliberate overlap is permitted.

Titles use 14-unit text and 12-unit usable padding inside the actual shape's safe text region. Wrap to that region and explicitly ellipsize overflow; never shrink fonts or alter source. Full title and home context are immediately available in the existing pane through pointer/keyboard selection. The diamond's safe region is smaller than its outer rectangle.

Each boundary reserves a caption rectangle centered below it: width equals outer node width, height 18, top 6 below shape bottom. Render “Lives in <home title>” at 12-unit font, 18-unit line height, 4-unit horizontal inset, one line with clipped end ellipsis. Scale the rectangle, font and gap uniformly with zoom; no minimum screen-font or pixel gap. Renderer, Fit and allocator use the union of shape and this complete reserved rectangle, independent of font measurement or title length. Before/With use their own dimensions with the same caption rule, including legacy snapshots.

Explicit Auto-layout and initial placement reserve these size/caption envelopes with a 24-unit gap. Initial allocation searches deterministically with stable-ID ordering, retaining every existing center and size. Auto-layout writes only selected-Diagram final positions once and leaves all sizes unchanged. Legacy v2 display fallback stays unchanged. Neither path captures browser geometry. Fit includes caption envelopes; review toggling uses one common logical frame.

Size changes are separate review facts from position, content, membership and Relationships. Reconciliation resolves visibility first, then whole width/height pairs independently of x/y pairs. Same results coalesce; different nodes merge; divergent sizes on one node require a side or manual pair. Older retained sides contribute no sizing edit: compare against truthful legacy displayed dimensions, preserve a selected stored v4 size, and initialize only remaining final visibility. Result version is max(A,P). Absent final pairs have no size conflict. Residual facts relative to A must reproduce the exact candidate through the strict constructor.

No shapes, routing, PDF, richer content, per-node fonts, presentation property bag, new identities, Planning/Agent Control or plan-only acceptance is introduced. Real browser/CLI authoring, restart/history/GC, cancellation/no-op/stale guards, size-position merge/conflicts and caption geometry at 0.5×/1×/2× require proportionate verification. Independent technical review and explicit human visual PASS remain distinct requirements.
