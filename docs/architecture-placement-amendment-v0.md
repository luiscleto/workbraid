# Architecture placement — stable visible nodes

Status: Approved and delivered; Phase 3.1 complete with explicit human visual PASS

This is a correction within Phase 3.1, not Phase 3.2. The human rejected the partial-pinning implementation during its visual gate: Queue could not be dragged because it was a boundary node, and arranging one node rearranged other nodes. Technical checks alone did not establish product acceptance. The corrected implementation subsequently received explicit human visual PASS and passed post-acceptance restart verification.

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
