# Architecture Phase 3.1 execution packet

Status: Approved

Target: durable manual node placement through the existing Architecture proposal/review/reconciliation product

Exact planning prerequisite: Reconciliation 1 completion `62575d4522d1719ea9f57680d7e4a9afde32a8bd`.

Exact prerequisite implementation: `22e45e977be0de4276b3a40c7ab8683dd058b5bf`.

Approved contract: `docs/architecture-rich-diagrams-v0.md`.

Approved Phase 3.1 baseline/roadmap SHA: `d6697f9689aad2237b4f5ffada585f6a193e7c48`, committed directly on the exact prerequisite above.

Packet-inclusive worker base: the exact clean commit produced by committing this approved packet alone on baseline `d6697f9689aad2237b4f5ffada585f6a193e7c48`. Report that SHA before dispatch and carry it in worker/review/completion provenance; do not amend this commit merely to embed its own hash.

The human approved this one implementation/review/gate cycle, including appearance-aware placement reconciliation, the sticky v3 target, and Reset layout preserving required non-resurrection nulls. Dispatch only from the exact packet-inclusive base. Stop after explicit Phase 3.1 PASS; no later rich-Diagram implementation.

## 1. Decomposition and execution discipline

Use one cohesive product increment with one implementation worker, one fresh independent review, integration, ordinary/production checks, bounded weak-agent discovery and a real human visual gate. Storage-only, canvas-only and reconciliation-later slices would each leave ordinary durable placement incomplete; styling/reset/transport work is not a separate ticket.

Implementation may proceed in ordered concern groups within the same worker: portable/operational state and constructor; shared authoring/reconciliation/projections; client interaction/parity. There is no intermediate human checkpoint or separate layout authority.

Approved execution order:

1. Cross-read the approved contract, roadmap and packet; resolve actual missing decisions before implementation. The baseline is committed at the SHA above after `git diff --check`; commit this approved packet alone and report its exact clean worker base.
2. Dispatch exactly one implementation worker in a separate Herdr workspace/worktree, not extra panes. Preserve unrelated user state and use conventional commits.
3. A fresh independent reviewer checks the whole worker-base→head range, separately covering storage/history, composition/reconciliation, transport authority and actual built-browser interaction. Return bounded corrections to the same worker and rereview the corrected range.
4. Integrate only the exact reviewed range. Record integrated head/tree identity, run checks, and build the real binary/UI. Do not claim visual quality from unit tests or screenshots without trying the interaction.
5. Run bounded weak-agent placement discovery and independent canonical/restart verification. Invite the human only after actual pointer behavior, review toggling and reconciliation controls are ready.
6. A human UX failure is a failed gate. Reproduce and fix it in the same worker/review cycle, preserving the approved semantic boundary; do not preserve bad UI merely to satisfy old tests. No substitute reviewer approval waives human usability.
7. Record exact evidence and explicit Phase 3.1 PASS, stop task runtimes and close unused task-owned workspaces/worktrees while preserving evidence. Do not begin Phase 3.2, routing design or another stage.

## 2. Exact worker brief

Read completely before implementation:

- `AGENTS.md`;
- `docs/architecture-v0.md` and `docs/ui-v0.md`;
- `docs/architecture-agent-access-v0.md`;
- `docs/architecture-change-sets-v0.md`;
- `docs/architecture-reviews-v0.md`;
- `docs/architecture-reconciliation-v0.md`;
- approved `docs/architecture-rich-diagrams-v0.md` and roadmap;
- this approved packet; and
- completed `docs/plans/architecture-reconciliation-1-execution.md`, including final UX/acceptance-landing and serializer results.

Apply later explicit supersessions, not obsolete anonymous-pending or v2-only restrictions. The new historical-reconstruction guarantee protects records supported at this prerequisite; it does not require reviving already-unavailable pre-fix alpha artifacts. Do not rewrite approved/completed documents during implementation.

Inspect the existing concrete paths rather than inventing substitutes:

- `internal/architecture/store.go`: native bootstrap, manifest/Diagram closed validation, immutable projections, Diagram serialization and `ConstructCandidate`;
- `internal/architecture/change_sets.go`: operational versions, exact tree equality, active/applied/ref transactions and unchanged-state behavior;
- `internal/architecture/reviews.go`: exact reviewed-parent loading and reachability without a live active ref;
- `internal/architecture/reconciliation*.go`: concrete facts, closed choices, residual generation and exact Apply boundary;
- `internal/web/handler.go`, `agent.go`, `review_projection.go`, `reconciliation.go`: shared locked mutations, proposal copies, state capture and projection;
- `internal/agentapi`, `cmd/workbraid/cli.go`, `mcp.go`, schemas and embedded `skill.md`; and
- `frontend/src/ArchitectureMap.tsx`, App/proposal/review routes, reconciliation task, styles and current production-browser scenarios.

The current map is Cytoscape preset positioning fed by a small deterministic circle. Use its model coordinates and one drag-completion event; do not capture transformed screen pixels or persist its whole node collection. Installed renderer APIs already support this boundary. No new graph-editor dependency is justified by current inspection.

## 3. Worker acceptance criteria

### A. Portable versions, historical state and fidelity

Implement exactly the contract's closed v3 manifest/Diagram schema and integer center range −100,000..100,000. V2 rejects even empty `positions`; v3 positions resolve only to canonical appearances. Native bootstrap becomes v3. Normal non-layout v2 authoring stays v2; a first real placement mutation upgrades one candidate, retaining v3 after resets. No-op reset creates neither upgrade nor proposal.

Support operational v3 with explicit `architecture_version` and final nullable-pair `node_positions`, keeping all old fields unchanged. Null means an explicit automatic override; missing pair means inheritance subject to composition. Persist no drag event, previous position, identity or generic operation. Keep the existing envelope/Review versions and ref namespaces.

`architecture_version` is the final requested value, not derived from nonempty `node_positions`. V2 base → set position → v3 → reset every position remains a real v3 format-only residual. Normal minimization cannot remove that target; v2 no-op reset cannot introduce it. Prove the exact resulting/reconstructed tree, not just a version field in a response.

Before changing serialization, capture bounded currently-valid v2/operational-v1/v2 reconstruction fixtures with real Git and exact tree/ref IDs. They must remain exact afterward. Test edited Diagram output as well as no-change trees: adding an optional field must not reformat all historical candidate blobs. Preserve old schemas and unchanged `changes.yaml` blobs when review preparation does not mutate facts. A real new mutation can write operational v3; loading cannot rewrite anything. Do not weaken candidate equality or branch into an old Architecture parser/constructor.

Placement rewrites only affected Diagram blobs and, for upgrade, manifest. Preserve IDs/paths/modes, unrelated values, surviving appearance/position order and exact untouched tree entries. Existing Component Title/Description/Relationship fidelity, including the approved unterminated-H1 structural LF exception, is unchanged. No lossless-YAML editor, source override or formatting normalization project.

### B. Ordinary placement and composition

Add shared set-one, reset-one and reset-Diagram operations. Resolve the complete candidate under the existing lock, not browser claims. Require exact store/Change Set/generation; browser Accepted-origin action instead verifies store/Accepted revision and publishes only the generated first mutation. One completed drag/reset-Diagram means one generation and one review invalidation; a genuine no-op means none.

Normalize appearance removal and placement together. Drop positive placement when an appearance disappears; use minimal concrete null absence when needed to prevent a base coordinate returning after re-inclusion/home round trips. Same-pair role conversion retains that pair's coordinate. Destination reference→home keeps the destination coordinate, never source coordinates. Candidate-new objects work normally. Subtree reparenting does not rewrite coordinates inside child Diagrams. Existing invalid work stays repairable; no manufactured partial candidate map.

Reset layout makes all currently visible canonical appearances in its Diagram Automatic, including inherited pins, in one mutation. Do not delete every operational entry for that Diagram: retain null overrides for removed appearances where needed for later non-resurrection. Backend normalization owns the distinction; the UI exposes only current manual/Automatic state.

Wrong project, late generation, missing/non-canonical target, invalid integer, cancelled drag and failed CAS must not create hidden proposals or overwrite newer work. Publication/response failure uses real existing state and no blind retry; do not add a drag journal or receipt. Other proposals and immutable review refs remain exact.

### C. Reconciliation and review

Extend existing semantic reconciliation with request-local `not_applicable`, `automatic`, `manual(x,y)` context per Diagram/Component. Never treat an absent appearance as Automatic. Resolve composition first. Final absent appearance prunes placement/conflicts. If B contained a surviving pair, an A/P branch removing it contributes no placement value; use the sole retained branch's value without a false conflict, or normal whole-pair three-way comparison if both retain it. If B lacked a newly introduced final pair, use Automatic as comparison baseline: Automatic plus manual combines, different manual pairs conflict. Keep actual three-side context truthful. `not_applicable` is neither portable nor operational persisted state and is not a manual resolution choice. No coordinate transfer, false Relationship delta, identity-collision reinterpretation or presentation-driven restore.

Use target format max(A,P), including all requested mixed v2/v3 combinations and format-only/empty residuals. Extend normal residual typed state relative to A, then verify the one constructor produces the exact stored candidate. Keep S/B/A/P preconditions, non-mutating Preview/Check, exact active/Accepted transaction, one new generation/cleared binding, no Accepted mutation and old-S retry protection. No layout-specific reconciliation storage or tree.

Add `Position changed` to the existing review comparison and changed-Diagram/context affordances. Both sides use only their own pins in one logical viewing frame; per-side fitting must not hide movement. Component/Diagram comments and historical review routes retain exact old snapshots. Layout-only work cannot become content/membership/Relationship change. Diff remains complete and authoritative if rendering fails.

### D. Browser interaction

Implement real canonical-node drag in Accepted/valid editable proposal contexts, local until drop; grabbed state, threshold/click suppression, pan separation, Escape/cancellation and stale rollback. Use model centers with nearest-integer half-away-from-zero rounding. Do not issue one request per frame or accidentally make boundary/comment nodes writable. Avoid accepting another gesture with an outstanding old generation.

Pins remain exact while automatic canonical and boundary nodes avoid obvious node/label overlap. Use a small deterministic extension to existing automatic layout, no force-framework replacement or persisted cache. Pin/pin overlap is not automatically repaired. Keep viewport stable after drop and during same-Diagram Before/With; Fit must still reveal all bounded coordinates. Neither viewport nor automatic changes become canonical diffs.

Provide compact contextual Position X/Y + Keep position for keyboard/precise access, Reset position and selected-Diagram Reset layout. No editor palette, extra top bar or future chrome. Preserve drafting-table layout, visible action styling, readable/non-cropped fields, diagram/component action distinctions, comment notes/dock, neutral selection, proposal design document and Accepted landing after successful Update. Immutable review/history canvases do not persist drags.

### E. Agent Access

Implement the four contract commands/tools/API operations and additive projection fields through the same backend operations. Keep agent-v2, exact identity/generation, no client-local state/store access, and common errors. Inspect returns manual/Automatic, never fake persisted automatic coordinates. Set/reset uses stable Component/Diagram IDs. Update embedded `--skill`, help and MCP schemas/descriptions including placement reconciliation; no generic JSON/YAML editor.

## 4. Bounded automated evidence

Real Git executable, temporary private stores/filesystem, real Go handlers and built browser remain authorities. Pure helper/UI tests supplement them. Use separate runner-owned asynchronous test cases, not a manual render/unmount loop. Keep tests concrete and proportionate; do not replay whole previous bootstrap/invalid-hierarchy/review-anchor matrices.

| Concern | Required evidence |
| --- | --- |
| Closed portable schema | Valid positive/negative/zero/boundary integer pairs; absent/empty positions; reject duplicate pair/keys, absent or boundary-only Component, fractional/string/boolean/null coordinates, overflow/out-of-range and unknown fields. V2 rejects positions. Bounds are consistent in Go, JSON and UI. |
| Version transition | Fresh native v3 bootstrap; ordinary mixed Component/Relationship/Diagram v2 work remains v2; first set upgrades only manifest + affected Diagram; reset-all after upgrade stays v3; v2 no-op reset does nothing. |
| Exact historical equality | Real currently-supported operational v1/v2 records over v2, including rewritten Component/Diagram candidates, survive loading/review preparation/restart with the same trees; a Review retains an old candidate after later mutation/application or active-ref discard and bounded real GC. No compatibility rewrite. |
| Typed durable state | Set/reset/null versus `(0,0)`, target version, candidate-new Diagram/Component, positive orphan rejection, invalid-other-work retention, exact envelope reload and `ConstructCandidate` tree equality. |
| Source fidelity | Placement-only Component blobs/paths/modes exact; untouched Diagram entries exact; edited Diagram semantic/order/path/mode fidelity; unaffected child/subtree blobs exact; resetting last entry omits only the changed file's positions. |
| Composition interaction | Reference→home keeps destination pin; source pin removed without transfer; remove/re-include and home round trip do not resurrect base coordinates; unrelated references keep their own pins; child reparenting leaves internal coordinates exact. |
| Shared authority | Late cross-project implicit drag creates no proposal; two clients on one generation cannot both mutate; separate proposals stay independent; response loss does not trigger replay. No-op set/reset preserves reviewed state/ref. |
| Drag production path | Actual pointer movement at non-default zoom/pan persists the rounded model center once; pointer moves create no refs/generations; click is not drag; pan/fit/reset/cancel and negative coordinates; failed/stale drop restores truthful rendering. |
| Partial layout | Dragging one of several automatic nodes persists only one pair; adding a Component does not capture other nodes; automatic/boundary nodes avoid obvious pinned-node/label overlap; deliberately pinned overlap remains exact. |
| Exact review | Position-only marking and canonical diff region; no content/composition/Relationship false deltas; own-side v2/v3 positions and stable logical viewport; candidate-only Diagram fallback; clear/continue-editing/read-only review behavior; map failure leaves exact review path. |
| Reconciliation values | Different nodes, same pair, divergent pair, reset-versus-move, Automatic versus zero; side/manual/Automatic choice; stale irrelevant resolution after composition choice rejected; final absent appearance prunes pin; same-pair role change retains/merges. |
| Appearance-aware comparison | B pinned / A removes / P moves / final keeps P → P coordinate, no false conflict; symmetric P removes/final keeps A → A coordinate; B absent / A adds Automatic / P adds manual → manual survives; B absent / both add different manual pairs → conflict; final absent → placement pruned. `not_applicable` never persists. |
| Sticky version / reset normalization | Exact tree for v2 → set → v3 → reset all remains v3 format-only, and v2 no-op reset is exact. Base D/C pin → remove C retaining null → Reset layout for other visible D nodes → re-show C remains Automatic; reset clears visible inherited pins but preserves the required null. |
| Mixed versions/residual | 2/2/3, 2/3/2, 2/3/3, 3/3/3 and unchanged 2/2/2; A-only pins preserved; target max(A,P); cleared v3 stays v3; format-only residual and truly empty residual distinguished; exact residual tree equality/restart. |
| Reconciliation authority | Preview/Check unchanged refs; stale S or A rejects Apply; success changes only one active ref/base/generation and clears review; old-S retry cannot apply twice; normal review + three-ref acceptance remains separate. |
| CLI/MCP parity | Shared read/set/reset-one/reset-all and typed negative/error cases, bounds, IDs/generations, skill/help/server independence, one typed placement conflict through existing preview/apply. |
| Publication/restart | Deliberate exact-bound acceptance, truthful response recovery, explicit external valid placement Refresh, route/catalog identity unchanged; full stopped-process restart yields exact final coordinates, candidate/Accepted trees and historical feedback. No new SQLite/registry/layout refs. |

Browser production evidence should combine these into a few purposeful cases: actual drag/reset/review/acceptance; failed gesture/preconditions; placement reconciliation and historical viewing. Frontend helper tests can cover geometry/rounding/obstacle cases separately. Preserve existing ordinary suites; update executable v2-bootstrap assertions to the new living baseline, not immutable historical documents. Old-format fixtures remain only for supported v2/reconstruction evidence, not a parallel legacy product workflow.

## 5. Fresh independent reviewer brief

Review the full exact range from the packet-inclusive base, not only the newest patch or worker claims. Verify four concern groups independently before approving the whole increment:

1. **State/version fidelity:** v3 is explicit and closed; historical v2/operational-v1/v2 reconstruct exactly; target version/null clear are concrete facts; unchanged blobs/modes/orders survive. No automatic serializer output change can hide behind semantic equality.
2. **Composition/reconciliation:** absent appearance is not Automatic; verify all five appearance-aware cases, including the Automatic baseline for a new pair. No transfer/resurrection; Reset layout retains required nulls; role/subtree behavior exact; sticky target/max-version does not strip A-only pins; whole-pair conflict/Automatic resolution; normal residual equality, no second builder/session/ref.
3. **Authority/transports:** every gesture and client uses the same locked state/preconditions; one drop/reset means one mutation; cancelled/stale Accepted gestures leave no empty proposal; exact review/Apply/Update boundaries remain distinct; historical Reviews not retargeted.
4. **Real interaction:** operate the built browser. Check drag versus click/drill-down/pan, automatic gaps, retained pins, zoom/fit, side toggling, keyboard Position controls, cancelled/stale gestures, comments and pane sizing. A green mocked test is insufficient.

Include one independently reproduced real-Git candidate round trip from v2 and one v3 reconciliation residual. Review the actual historical fixture IDs and GC proof. Findings return to the same worker; review corrections over the entire final range. A new product/identity decision is a stop for the human, not reviewer license to invent it.

## 6. Integration and ordinary checks

Preserve unrelated edits. Verify the exact clean worker base, reviewed head and tree, then integrate non-destructively and compare the resulting implementation tree with the reviewed tree. Record final implementation SHA, build identity, commands and results. No unreviewed production fixes are smuggled into integration.

Run the normal Go and frontend checks, with bounded additions:

```text
git diff --check
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
go mod verify
npm test --prefix frontend
npm run build --prefix frontend
```

Run the relevant existing production Playwright suite plus the new focused placement cases against the real built Go/UI using the repository's concrete runner. Record exact command, cases and retry count; do not add a generic test framework. Tests that intentionally assert old bootstrap version change proportionately. Previously accepted authoring/review/reconciliation failures freeze the increment until reproduced and corrected.

## 7. Bounded agent and canonical verification

After implementation review/checks, one fresh weaker verifier (prefer Luna) receives a separate product-created project and only a business task plus ordinary CLI `workbraid --skill` or configured MCP discovery. No source, packet, private Git or operation-name tutoring. Ask it to discover placement, inspect an active proposal, set one Component in one Diagram to stated logical coordinates, reset another positioned node, and clear a Diagram layout in a separate ordinary proposal if needed. It inspects/reviews the exact result and reports IDs/state/generation/coordinates. Tasks specify desired values, not aesthetically judged arrangements or command sequences.

Passing requires independent discovery without hidden API/Git edits or human unblocking. Missing reads, unclear Automatic semantics or ambiguous tool instructions are product findings; do not compensate in the prompt. Automated tests cover the other transport. Capture the transcript/tool sequence and exact proposal state/binding; do not let this agent accept the human gate's proposals or modify its fixture.

A stronger independent technical verifier checks real private-Git truth for both technical and human workflows:

- manifests, Diagram entries and exact position pairs; unchanged Component/Diagram blobs/modes;
- proposal envelope/state/generation/target format and constructor equality, not merely rendered coordinates;
- preview/ref immutability, reconciliation active-only transaction, exact new base/residual and old-S rejection;
- ordinary applied receipts and accepted successor parents/trees;
- immutable old Review parent/comment refs including v2 historical candidates after bounded GC in a dedicated temporary store; and
- restart from the same app-data directory with a genuinely fresh process, catalog/route opening, exact accepted/proposal/history results and no layout/session/cache authority outside Git.

Retain evidence before cleanup. No extra metadata registry, reconciliation ref or SQLite state is permitted. Verification may inspect private Git; the black-box agent may not. Do not substitute source-folder fixtures now that WorkBraid projects are slug-native.

## 8. Real human visual checkpoint

Prepare the meaningful project **through current production authoring** using fresh app data and the integrated built Go/UI. It should have a root and detail Diagram, several connected Components, a canonical reference, at least one Lives in boundary and readable documentation. Technical setup may create/accept that content so the human does not repeat earlier gates. Record its native-v3 bootstrap and seed Accepted R0. Historical v2 compatibility is technical evidence, not a conversion wizard for the human.

Keep the checkpoint in two short sessions within one gate, with exact bindings/coordinates recorded by the orchestrator before asking for acceptance. Do not ask the person to inspect Git or copy inaccessible diffs after Update.

### A. Arrange, compare and reconstruct

1. From Accepted, drag one canonical node. Confirm this starts one named proposal and leaves Accepted exact. Arrange several home/reference nodes, leave at least one automatic, and pan/zoom/fit. Check that only intended moves persist, labels remain usable and dragging does not drill into another Diagram.
2. Use Reset position and Reset layout in a bounded proposal context, confirming Automatic and no semantic membership/Relationship change. Use keyboard Position controls for one node. End with a useful partly manual layout. Switch Diagram/proposal and return; no placed node should jump to automatic coordinates.
3. Review the proposal. Inspect Position changed, exact Before/With placement in a stable logical frame, and complete diff. A small submitted Component/Diagram comment can be prepared through the product at this binding. Continue editing and move a node once more; inspect that old feedback on its exact older snapshot and return to current Review. No new comment anchor model is required.
4. Deliberately Update the final exact binding. Land on Accepted R1. Fully stop the Go process, prove it is stopped, start the exact built binary afresh with the same app data, and reopen the project route. Confirm exact coordinates and sensible layout, including automatic nodes and preserved feedback.

### B. Resolve parallel placement

5. Prepare two explicit proposals A and B from R1 through normal clients. The human can make the few moves: A moves X and Shared; B moves Y and Shared to a different location. Keep at least one same-result or reset case in automated evidence rather than enlarging this manual matrix. Record names, IDs, base, generations and pairs. Review both independently.
6. Deliberately accept A to R2. B remains exact/editable and out of date. Open its existing Reconcile with Accepted task. Confirm X/Y combine automatically, Shared needs a decision, and Original/Accepted/Proposed positions are understandable. Choose a side or enter a deliberate manual pair; Apply reconciliation changes B, not Accepted R2.
7. Inspect B's new proposed arrangement and historical old feedback where present. Ordinary Review shows the exact residual relative to R2. Deliberately Update to R3; confirm Accepted landing and sensible rendering. The stronger verifier proves exact reconstructed residual and accepted tree independently.
8. Perform a final complete stop/start and reopen. Confirm R3/positions and durable proposal/review navigation. One bounded external valid position change + explicit Refresh is exercised in production automated evidence unless a live discrepancy warrants human inspection; do not prolong the gate by replaying the prior Refresh matrix.

The human PASS/FAIL covers dragging, partial placement, reset/keyboard controls, review movement, reconciliation clarity and restart rendering—not merely returned JSON. Model-created coordinates cannot substitute for this judgment. Retain the human's explicit Phase 3.1 PASS with exact R0/R1/R2/R3, bindings, proposals/reviews, screenshot and canonical evidence. If the workflow needs new domain semantics or hidden manual Git repair, stop.

## 9. Completion, stop conditions and exclusions

Complete only after full-range independent review, checks, bounded agent discovery, canonical/restart proof and explicit human visual PASS. Record baseline/packet/worker/reviewer/integration/build SHAs, exact refs/tree IDs/generations, transport transcript and measured coordinates, UI findings/corrections and stopped-process evidence. Stop/clean only task-owned runtimes/workspaces no longer needed; preserve user data and retained evidence.

Stop and return a concrete case if partial placement cannot remain useful without whole-layout persistence, a supported residual cannot be encoded by normal typed state, historical supported v2 trees cannot reconstruct without another interpretation, or an identity question is required. Do not mask any of these with a cache, raw tree/blob override, generic migration/operation subsystem or weakened equality test.

No appearance IDs, Relationship IDs, canonical boundary coordinates, sizing, routes/bend points, shapes/annotations, grouping, graphical composition/Relationship authoring, UML/kinds/isometric renderer, whole-automatic-layout capture, persisted view state, snapping/multiselect/layers/minimaps/rulers/guides, undo/redo/history, new review anchors/lifecycle, layout session/ref/registry/SQLite, remote collaboration/permissions, Planning/Agent Control or other verticals. Do not begin Phase 3.2 or routing identity design after this gate.
