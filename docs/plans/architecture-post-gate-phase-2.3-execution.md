# Post-Gate Architecture Phase 2.3 execution packet

Status: Complete

Architecture baseline: `docs/architecture-v0.md`

UI baseline: `docs/ui-v0.md`

Roadmap: `docs/architecture-post-gate-roadmap.md`

Approved superseding continuation plan: `docs/plans/architecture-post-gate-phase-2-continuation.md`

Historical original Phase 2 plan: `docs/plans/architecture-post-gate-phase-2.md`

Completed P2.1 record: `docs/plans/architecture-post-gate-phase-2.1-execution.md`

Completed P2.2 record: `docs/plans/architecture-post-gate-phase-2.2-execution.md`

Exact completed-P2.2 prerequisite: `dd75ceb8db15f1dafef33513d632b25315114c6b`

Worker base: `02aa3615871a03587eea84f25a9056f3d478d8f4`

Target: P2.3 — Nested Diagram composition

Architecture or product changes discovered during implementation require explicit human approval rather than silent changes to this packet, its plan, or its baselines.

## Execution discipline

After human approval:

1. Mark this packet Approved and commit it alone on top of exact prerequisite `dd75ceb8db15f1dafef33513d632b25315114c6b`.
2. Record that clean docs-inclusive commit as the one exact worker base.
3. Dispatch exactly one P2.3 implementation worker from that base.
4. Require one cohesive conventional implementation commit, or a small review-correction chain rooted at the exact worker base. Do not split storage, candidate construction, handlers, review projection, frontend, or tests among product workers.
5. Send the complete committed implementation range to one fresh independent reviewer who did not implement it.
6. Return every material finding to the same bounded worker and require fresh independent rereview of the corrected complete range.
7. Integrate only the exact independently approved implementation.
8. Run the bounded ordinary checks once from the clean integrated tree.
9. Run the real human checkpoint through the built UI served by the real loopback Go process, then record exact Git, source-isolation, SQLite, restart, and human evidence in this packet.
10. Stop after P2.3. Do not prepare, dispatch, or implement P2.4 until P2.3 is independently reviewed, integrated, and explicitly human-approved.

Do not alter approved baseline documents or completed plans/records during implementation. If P2.3 exposes a missing product/domain rule, stop and return the smallest decision to the human.

## Exact worker brief

Before editing, verify the exact clean worker base and read completely:

- `AGENTS.md`;
- `docs/architecture-v0.md`;
- `docs/ui-v0.md`;
- `docs/architecture-post-gate-roadmap.md`;
- approved `docs/plans/architecture-post-gate-phase-2-continuation.md`;
- this packet;
- completed `docs/plans/architecture-post-gate-phase-2.2-execution.md` for the current writable-v2 pending, candidate, review, CAS, UI, and provenance paths;
- completed `docs/plans/architecture-post-gate-phase-2.1-execution.md` for the accepted-v2 loader, hierarchy, navigation, boundary, and closed-schema paths;
- completed `docs/plans/architecture-post-gate-phase-1.md` and `docs/plans/architecture-post-gate-phase-1-execution.md` for the immutable candidate-aware review authority and presentation semantics;
- the completed I2/I3 plans named by the P2.2 record where their authoring fidelity, pending-state, stale, Refresh, CAS, source-isolation, or restart boundaries are directly touched.

Implement only nested Diagram composition on the current normal writable-v2 foundation.

### One pending Architecture authority

- Extend the existing one backend-held pending Architecture change set. Retain one exact accepted base, one generation, one `ConstructCandidate` path, one complete candidate tree, one version-aware loader/validator, one immutable review candidate, one exact unified diff, one `(base commit, candidate tree, pending generation)` review binding, and one confirmation/CAS/publication path.
- Add only the minimum explicit typed pending state required to represent:
  - newly created detail Diagrams and their stable creation facts;
  - explicit Diagram-title edits;
  - explicit Component-home moves, including movement of a home-owned detail link.
- These are Diagram-composition facts, not fake Component edits, Relationships, sentinel IDs, or browser-authored canonical files. Do not force them into `ComponentChange` merely to preserve an old struct shape.
- Keep the representation concrete to these three operations. Do not add a generic operation algebra, command/event system, generic pending-kind framework, Diagram draft store, hierarchy transaction model, graph/membership abstraction, migration object graph, or second candidate builder.
- Resolve every Diagram, Component, appearance, and hierarchy target against the complete current pending candidate under the existing concrete backend synchronization boundary. Do not resolve only against the accepted snapshot: a candidate-only detail Diagram may receive moved/new Component homes and may itself contain an anchoring home for another candidate-only detail Diagram in the same pending set.
- Each successful kept mutation increments the one generation and invalidates the prior review binding. A rejected transport/conflict request creates no hidden pending fact or generation. Review remains possible only for one completely constructed and validated candidate.
- Preserve P2.2 ordinary Component Title, Description, Relationship, and Component-creation changes in the same pending set. One generation may coherently combine all of those with P2.3 Diagram composition.
- Begin every reconstruction from the exact accepted base tree and apply the complete pending state deterministically. Do not incrementally mutate Git or treat an intermediate invalid Diagram tree as canonical.

### Detail-Diagram creation

- From a Component's home appearance, provide one structured **Create detail diagram** action in the contextual working pane.
- The synchronized backend must establish that the target is currently a home appearance in the complete pending candidate and does not already own a detail Diagram. References, derived boundaries, and homes which already own a detail link cannot create or replace a detail Diagram.
- Generate one stable Diagram UUID exactly once when the kept pending operation begins. Retain that identity through correction, browser reload in the same process, candidate reconstruction, review invalidation, and acceptance. Whole-set discard clears it with the rest of the pending set and never publishes or reuses it. Never accept a browser-selected Diagram ID as authority.
- Generate one collision-safe, human-readable `diagrams/*.yaml` creation path exactly once on the backend. Use the submitted title as the human-readable basis when possible and a neutral human-readable fallback while the title is incomplete; never rename the file when its title later changes. The filename carries no identity, hierarchy, or root authority.
- Create one ordinary mode-`100644` Diagram blob with the generated ID, the authored title, and an initially empty appearance set. Put `detail_diagram` only on the anchoring home appearance in the parent Diagram. The child stores no parent or anchor field.
- The Diagram title is an authored v2 string: it is invalid only when empty after trimming under the approved schema. Retain an otherwise valid submitted value exactly; do not introduce silent trimming, whitespace normalization, case conversion, or another title rule. A temporarily blank title may remain non-canonical pending work, but it blocks Review changes and correction retains the same generated ID and filename.
- Do not introduce Diagram deletion, replacement, detach, or a second-detail lifecycle. Discard remains whole-set only.

### Diagram title editing and source fidelity

- Allow structured title editing for any Diagram, including root, from its contextual Diagram task. Diagram titles need not be globally unique; add only the minimum collision context already approved for Diagram navigation.
- A title edit changes neither Diagram ID nor filename, appearances, detail link, hierarchy, Component membership, or any Component/Relationship fact.
- Reuse the exact existing Git tree entry, blob, and mode for every Diagram path untouched by the pending change.
- When a Diagram is edited, preserve its stable ID, path, regular-file mode, every unchanged semantic field/value, and surviving appearance source order. A title-only edit changes only the semantic title; a composition edit changes only the semantic Diagram fields required by that operation.
- Avoid gratuitous rewriting, but do not implement a general lossless-YAML parser/editor. P2.3 does not require lexical preservation of quoting, whitespace, comments, scalar spelling, or other YAML formatting inside a Diagram file which is itself rewritten.
- Preserve externally authored conforming Diagram filenames and valid titles. Do not impose the WorkBraid creation filename convention on loading or later editing.

### Component-home movement

- Provide one explicit human-facing action for changing where a Component lives. Show target Diagram titles and only collision-needed context; do not expose Diagram IDs, appearance roles, YAML, parent IDs, or root mechanics.
- Resolve the Component's one current home and requested destination from the complete candidate, then change Diagram composition only:
  - remove the source home appearance;
  - add one home appearance in the destination;
  - if the destination already contains that Component as a canonical reference, convert that existing appearance in place to home rather than adding a duplicate;
  - do not leave or synthesize a reference in the source Diagram.
- Preserve surviving appearance source order. A destination reference conversion keeps that appearance's position; a newly inserted home uses the natural authoring insertion position. This is canonical fidelity, not appearance ordering semantics or appearance identity.
- A home move must not change the Component ID, path, blob, regular-file mode, H1/Title, Markdown body/Description, outgoing Relationships, or any Relationship label/order. If the same pending candidate independently edits that Component or its Relationships, only those explicit ordinary edits may change its blob.
- The current Diagram cannot become both home and reference for the Component, and a Component cannot be left homeless or reference-only. The complete candidate validator remains final authority.
- P2.3 may convert an already-canonical destination reference to home only as part of this move. It must not expose general reference creation/removal or make derived **Lives in** boundary references canonical.

### Anchored subtree reparenting and cycle rejection

- If the moved home owns `detail_diagram`, move that exact detail link with the home. This reparents the existing child Diagram and its complete descendant subtree through composition; it does not copy, rewrite, or re-identify child/descendant Diagrams.
- Preserve every child and descendant Diagram ID, path, blob, mode, title, appearance, and descendant link unless it is independently edited by another explicit pending operation. Only the source and destination Diagram composition entries required by the move may change.
- Validate the complete resulting tree through the one existing version-aware candidate loader:
  - exactly one manifest-selected root;
  - exactly one parent link for each non-root Diagram;
  - every Diagram reachable from root;
  - no hierarchy cycle;
  - every detail target exists and is linked only from a home appearance.
- A requested move which would place the anchoring home beneath its own detail descendant is invalid. Do not repair it by detaching the detail link, moving only part of the subtree, relocating another appearance, cloning a Diagram, selecting another parent, or otherwise reconciling it automatically.

### Incomplete state, validation, and correction

- Pending authoring may be temporarily incomplete where the established Changes-in-progress model permits it. Invalid composition never enters Review changes, never changes the accepted projection, and never creates an accepted successor.
- On Review changes, construct and validate the exact complete candidate. Preserve the entire pending set after failure so the person can correct only the responsible title/move while retaining all ordinary and Diagram work.
- Map concrete candidate errors to concise product-language guidance and a useful location. At minimum cover proportionately:
  - blank/incomplete Diagram title;
  - invalid or no-longer-available move destination/home;
  - duplicate or conflicting appearance;
  - missing Component home;
  - a reference which illegally owns a detail Diagram;
  - missing or multiple Diagram parent;
  - unreachable Diagram;
  - descendant-cycle movement.
- Mark the relevant Changes-in-progress Diagram/Component row, make the **Fix** action visibly actionable, open the responsible task, and focus/highlight the exact Diagram-title or home-destination control when there is one. The summary may state that one or more Diagram changes need correction; it must not expose YAML, UUID, parser, candidate, tree, ref, or schema terminology.
- Keep raw structural loader errors internal. Reuse the current concrete validation-location fields or make the smallest Diagram-specific extension; do not build a generic validation engine or promise a UI editor for arbitrary externally malformed canonical YAML.
- Known stale/non-current state, an indeterminate Refresh failure, legacy-v1 restrictions, and whole-set discard retain their approved meanings. A late request from stale browser state must be rejected from synchronized backend-owned project/snapshot/pending authority.

### Candidate-aware Diagram review

- Extend the existing one **Review changes** workspace and its exact immutable bound base/candidate pair. Do not add a Diagram review object, alternate comparison authority, another candidate interpretation, or another acceptance path.
- Capture the still-current binding and every immutable snapshot/projection/comparison value needed for the response atomically under the existing concrete backend lock. Serialize after releasing the lock. Pending mutation, discard, Refresh/stale transition, project switch, or acceptance invalidation must yield one coherent generation or reject the capture.
- Compare Diagrams by stable Diagram ID and add only concrete review-presentation classifications needed for:
  - added Diagram;
  - Diagram title change;
  - home removal from a source Diagram;
  - home addition in a destination Diagram;
  - one home move represented coherently rather than as Component replacement;
  - detail-link addition or movement;
  - the resulting child/subtree parent-context change where relevant.
- A home move is Diagram composition. It is not Component content change and must not create a Component added/replaced classification.
- Retain Relationship comparison exclusively as the exact `(source Component ID, target Component ID, exact label)` multiset including multiplicity. A composition-only change which causes an unchanged Relationship to switch between ordinary internal-edge and derived **Lives in** boundary presentation must not create a Relationship addition/removal. If the candidate also contains a real Relationship edit, show that independent real fact delta normally.
- Preserve existing changed-Diagram indicators in the Diagram tree and retain the current Diagram when review opens where possible. Selecting a Diagram/home/detail-link change must focus its review context and the relevant region(s) of the complete exact unified diff. Exact raw diff remains complete and authoritative.
- Keep the review canvas assistive. A clearly reported Diagram/map-render failure must not remove the exact diff, review binding/details, **Continue editing**, **Discard changes**, or eligible **Update architecture** path.

### Snapshot unity and candidate-only Diagrams

- **With changes** and **Before changes** must switch the Diagram tree, breadcrumbs, selected Diagram, index, map, Component documentation, canonical appearances, derived boundaries, Relationships, titles, and revision together from the same exact bound side.
- If the selected Diagram exists only in the candidate, switching to **Before changes** must select the nearest ancestor in that candidate Diagram's parent chain which exists in the bound v2 base; if none survives, select the bound base root.
- Show the restrained approved note that the previously selected Diagram exists only with the changes. Never leak its appearances, index, documentation context, boundary references, topology, titles, or descendant state into the base side.
- Exact focus restoration when returning to **With changes** remains disposable browser behavior. Do not turn selection into canonical, persisted, or review-authority state.
- Preserve the P2.2 legacy setup comparison unchanged; P2.3 Diagram-composition mutation is available only against current writable v2.

### Contextual authoring workspace

- Keep the normal map, Diagram tree, index, and documentation exact accepted-only projections. Pending new Diagrams, titles, home moves, and hierarchy changes must not appear there before successful CAS.
- Reuse **Changes in progress** and the contextual working pane for pending Diagram composition. Once a Diagram operation exists, make candidate-only Diagrams and candidate-relative destinations reachable for continued composition editing from that task without overlaying them on the normal accepted map or pretending they are accepted.
- The structured flow must support two nested candidate-only Diagram creations and moves in one pending set. It must not require accepting an intermediate hierarchy or editing canonical YAML.
- Preserve dirty-editor guards when navigation would replace unsent local Component, Relationship, Diagram-title, or move controls. **Leave without keeping** drops only browser-local values; it does not alter backend pending work.
- Preserve the approved P2.2 distinctions and layout:
  - **Edit component** edits semantic Component content;
  - **Open _diagram_** is a separately styled navigation action beneath it;
  - changing where a Component lives is an explicit composition task;
  - canonical-reference context and separate external **Lives in _diagram_** captions remain truthful;
  - review and external-reference information share the collapsible bottom dock;
  - the Changes dock remains available in review even when the selected Diagram has no visual deltas;
  - **Continue editing** remains near the review Update/Discard actions;
  - clear selection remains neutral and does not reopen an arbitrary Component.
- Keep the P2.1 breadcrumb hover-contrast and P2.2 italic-visibility observations non-gating unless the exact touched UI naturally admits a bounded correction. Do not create a polish subtask.

### Authority, acceptance, Refresh, and isolation

- All new mutation handlers use the existing expected-origin checks and concrete `stateMutex` boundary. Eligibility and stable targets come from the current server-owned loaded project, accepted snapshot, and one pending candidate, never browser format/capability/base claims.
- Preserve exact review-binding confirmation, explicit stale-base observation before successor creation, mandatory accepted-ref CAS, immediate pending consumption only on CAS success, publication of the already-validated successor snapshot, response-loss classification, and failed-publication recovery.
- Normal accepted workspace advances only after the existing successful CAS boundary. Diagram tree, root/detail navigation, index, map, documentation, appearances, boundaries, Relationships, and revision publish together from the successor snapshot.
- Preserve explicit Refresh as accepted-authority re-observation. One valid external v2 advancement replaces all projections atomically; old-base pending Diagram composition becomes stale/read-only and remains inspectable/discardable in its exact old context.
- Reconstruct accepted Diagram composition only from canonical Git after a complete process restart. Do not project Architecture, Diagrams, pending work, review state, hierarchy, or layout into SQLite.
- Do not modify the source repository. Keep all candidate/commit/ref work in the private Architecture store using the real Git executable with fixed controlled arguments, no shell, hooks, signing, or pager.

## Worker acceptance criteria

The implementation is ready for fresh independent review only when:

- detail creation from a complete-candidate home generates one stable Diagram ID/path exactly once, creates one empty titled child, and stores its only parent link on the anchoring home;
- blank title blocks review with localized guidance while correction preserves the generated identity/path and the entire mixed pending set;
- Diagram rename preserves ID/path/mode, every unchanged semantic field/value, and surviving appearance source order without requiring lexical YAML preservation inside the rewritten Diagram;
- ordinary home move removes the old home, adds one destination home, leaves no source reference, and rewrites no Component or Relationship fact;
- an existing destination reference converts in place to the one home without duplicate appearance;
- an anchoring home carries its exact detail link to the destination and reparents the existing full subtree without changing child/descendant identities or blobs;
- a descendant-cycle move and representative invalid composition cannot reach review/acceptance, retain all pending work, and can be corrected through the structured UI;
- P2.2 Component content, Relationship, and new-Component authoring coexist with detail creation/title/home movement in one base/generation/candidate/diff/binding/CAS path;
- composition review shows added/title/home/detail/hierarchy changes without false Component-content or Relationship-fact deltas and keeps exact diff focus available;
- candidate-only Diagram fallback is snapshot-unified and leaks no candidate state into **Before changes**;
- pending Diagram composition does not alter the normal accepted workspace before CAS; successful CAS advances every accepted projection together;
- review invalidation, stale authority, Refresh, project switching, discard, dirty-editor protection, post-CAS publication/recovery, safe Markdown, source isolation, SQLite boundary, and restart reconstruction do not regress;
- no P2.4 reference authoring, Diagram deletion, persisted layout/pending state, graphical editing, generic framework, approved-doc edit, dependency/build/database artifact, or source-project mutation enters the implementation;
- the worker commits conventionally and leaves its isolated tree clean.

## Required automated evidence

Use bounded production-path evidence. Reuse the real loader/candidate/handler/browser authorities and do not replay P2.1's closed-schema corruption matrix or P2.2's initialization/legacy-setup matrix.

### Architecture and real Git

Use temporary real bare Architecture stores and the real Git executable with fixed controlled arguments.

- Starting from accepted writable v2, construct one mixed pending candidate containing an ordinary Component body/Description edit, a real Relationship edit, a pending-new Component homed in a candidate-only Diagram, a newly created detail Diagram/title, and at least one home move. Load the complete candidate through the existing version-aware loader.
- Prove the new Diagram UUID and collision-safe path are generated once and remain stable across failed validation, correction, repeated candidate construction, review invalidation, acceptance, Refresh, and fresh-process reconstruction.
- Prove detail creation changes only the new Diagram plus the parent Diagram composition required for its link. Prove title-only rename changes only the semantic title while retaining Diagram ID/path/mode, every unchanged semantic field/value, and surviving appearance source order. Require exact tree-entry/blob/mode reuse for every untouched Diagram path, but do not assert lexical YAML preservation inside an edited Diagram blob.
- Prove ordinary home movement reuses the exact unchanged Component blob/path/mode and every unchanged Relationship fact. Verify source-home removal, destination-home addition, and no implicit source reference.
- Prove destination reference-to-home conversion leaves one appearance in its original position and no duplicate home/reference pair.
- Build a two-level detail subtree, move its anchoring home, and prove the child and descendants retain exact IDs, paths, blobs, modes, titles, appearances, and internal hierarchy while the parent context changes once.
- Attempt a descendant-cycle move and prove the candidate cannot validate or become reviewable, accepted Git remains exact, and the complete mixed pending set survives correction.
- Cover missing home, duplicate/conflicting appearance, illegal reference/detail combination, missing/multiple parent, unreachable Diagram, and blank title proportionately through the complete candidate/loader path. Do not create a second validator or broad corruption matrix.
- Prove composition-driven internal-edge/boundary projection changes preserve exact global Relationship facts and yield no Relationship multiset delta; separately prove a real Relationship edit in the same candidate still yields its exact delta.
- Prove parent diff, tree identity, accepted-ref CAS, successor load, and exact fresh-process reconstruction.

### Production handler and synchronization

- Exercise structured detail creation, title edit, and home move through production handlers. Prove targets are checked against synchronized server-owned complete pending state, including a candidate-only Diagram and a pending-new/moved anchoring home.
- Prove one pending base/generation accumulates P2.2 ordinary edits and P2.3 composition; every kept mutation invalidates the old review binding and a late confirmation cannot accept it.
- Race one Diagram mutation with Discard, Refresh/project switch, or review capture. The result must be one coherent loaded/pending/binding generation, with no partial Diagram fact or hidden pending work.
- Prove a late request against another project, known-stale snapshot, discarded generation, accepted v1, or superseded target is rejected without Git/source/SQLite/loaded-state mutation.
- Prove failed complete-candidate validation records localized product guidance, retains all pending facts, leaves accepted projections exact, and correction produces one new coherent candidate.
- Preserve the explicit pre-commit stale observation, final CAS race behavior, response-loss handling, immediate pending consumption only after CAS success, and post-CAS publication/recovery.
- Exercise one valid external accepted-v2 advancement and explicit Refresh through the shared production path; old-base pending composition must become exact stale/read-only evidence rather than being reinterpreted.

### Frontend and built-browser behavior

- Add runner-owned frontend cases for contextual **Create detail diagram**, Diagram title/rename, structured home destination selection, destination-reference conversion presentation, and candidate-relative continued editing. Do not put repeated render/unmount/mock-restoration lifecycles inside a manual loop.
- Prove blank-title and invalid-move errors mark the relevant Changes row, expose an obvious **Fix** action, and focus/highlight the exact affected control without parser/schema language.
- Prove dirty local Diagram controls receive the established **Keep editing** / **Leave without keeping** guard before navigation replaces them, and leaving does not clear backend pending work.
- Prove the normal accepted tree/map/index remain unchanged while pending composition is edited, while **Changes in progress** can reach candidate-only Diagrams for further work.
- Prove Diagram review markers, selection, diff focus, stable Diagram context, zero-change Changes dock, candidate/base snapshot unity, and candidate-only fallback.
- Prove an internal↔boundary presentation change from composition produces no false Relationship delta and retains the approved canonical-reference/**Lives in** captions and navigation.
- Prove **Edit component**, **Open _diagram_**, and home-movement controls remain visually and semantically distinct; preserve **Continue editing**, Update/Discard placement, bottom-dock behavior, and neutral clear selection.
- Through a bounded production-browser scenario served by the real Go process and built UI, perform nested detail creation, a corrected invalid move, mixed ordinary/composition review, exact acceptance, navigation, and reopen. Do not substitute frontend mocks for this evidence.

### Ordinary checks and resource safety

After focused tests pass, run from the clean integrated tree:

- `git diff --check`;
- `go test ./... -count=1`;
- `go test -race ./... -count=1`;
- `go vet ./...`;
- `go mod verify`;
- the repository's ordinary frontend `npm test` command;
- `npm run build`;
- the bounded P2.3 production-browser scenario;
- only the existing P2.2/Phase 1/P2.1 production scenarios whose still-supported shared path was materially changed.

Never invoke raw `vitest`, `npx vitest`, or a hand-written repeated render/unmount loop. Let the repository test command and test-runner case boundaries own resource limits and cleanup. Stop immediately on abnormal memory growth or orphaned npm/Node/Vitest/Chromium processes; diagnose before retrying. Do not weaken ordinary product defaults merely to mask a runaway test.

Record commands, results, exact tested SHA, and any non-blocking warnings. Do not commit browser output, dependencies, temporary stores, binaries, logs, screenshots, or databases.

## Fresh independent reviewer brief

The reviewer receives the exact worker base, complete implementation commit/range, approved baselines, continuation plan, this packet, and relevant completed P2.1/P2.2/Phase 1 records. The reviewer must read the implementation rather than relying on test summaries and report findings by severity with exact evidence.

Review specifically:

1. **Single authority:** one pending set/base/generation/candidate builder/loader/review binding/CAS path remains. Diagram composition is represented by small honest typed state rather than fake Component changes, generic commands, Diagram drafts, or another graph authority.
2. **Candidate-relative composition:** candidate-only Diagrams and moved/pending homes resolve inside the same complete candidate under the existing synchronization boundary; browser IDs/state never decide eligibility.
3. **Creation identity:** detail UUID/path are generated once, retained across correction, and never inferred from title/path. Link ownership is parent-home-only and no deletion/replacement lifecycle slipped in.
4. **Fidelity:** every untouched Diagram path reuses its exact tree entry/blob/mode; an edited Diagram preserves ID/path/mode, unchanged semantic values, and surviving appearance order while changing only operation-required semantics. No lossless-YAML subsystem or lexical-preservation claim appears. Component and Relationship blobs/facts remain exact unless independently edited, and unchanged subtree blobs are preserved.
5. **Home semantics:** source removal, destination addition, in-place reference conversion, no source reference, one home, no duplicates, and no general reference authoring.
6. **Hierarchy:** moving an anchor moves its existing detail link and reparents the exact subtree; root/one-parent/reachability/acyclic validation is complete; descendant cycles fail without automatic repair.
7. **Validation UX:** invalid pending work is retained, localized, correctable, and cannot review/accept. Product UI exposes no YAML, IDs, roles, parser, schema, candidate, tree, or ref language.
8. **Review truth:** stable-ID Diagram comparison, composition classification, hierarchy context, exact diff focus, snapshot unity, candidate-only fallback, and no false Component-content or Relationship-fact deltas.
9. **Workspace continuity:** accepted-only normal workspace, candidate-relative Changes task, safe Markdown, dirty guards, discard/project switching, retained Diagram context, dock behavior, captions, clear selection, and action distinction do not regress.
10. **Concurrency and authority:** pending mutation, discard, Refresh, switch, review capture, confirmation, CAS, and publication cannot interleave into mixed state; stale/late requests and failed CAS cannot publish or hide work.
11. **Real evidence:** tests use real Git/filesystem/SQLite/Go/browser paths proportionately, cover restart and isolation, do not preserve superseded behavior, and introduce no resource-risk manual async loops or fake production capabilities.
12. **Scope:** no P2.4 reference action, Diagram deletion, DAG/reuse lifecycle, persisted layout/pending state, graphical authoring, generic framework, approved-doc mutation, source-project write, dependency artifact, or unrelated polish.

Any material correction returns to the one P2.3 worker. The corrected full range receives a fresh independent rereview before integration.

## Integration procedure

1. Verify the main worktree is clean and still at the packet-inclusive worker base.
2. Verify the worker range is rooted at that exact base and contains only the reviewed P2.3 implementation/correction commits.
3. Integrate the exact reviewed range without rebuilding it by hand.
4. Confirm the integrated tree/commit matches what the reviewer approved.
5. Run the ordinary checks and bounded browser scenarios above once from the integrated tree.
6. Stop every test/runtime/browser process and verify no WorkBraid, npm, Node, Vitest, Playwright, or Chromium process remains.
7. Prepare one fresh real human-checkpoint environment. Do not pre-seed the primary workflow with canonical Diagram changes that the checkpoint is meant to author through WorkBraid.
8. If the human checkpoint exposes a defect within approved semantics, stop the gate, record exact evidence, return the bounded fix to the same worker, obtain fresh rereview, reintegrate, rerun affected checks, and restart the human checkpoint from truthful state. If it exposes a missing product/domain decision, stop and ask the human.

## Real human checkpoint

Use the built frontend served by the real loopback Go process, real SQLite, the real filesystem, and real Git against a throwaway source repository and private application-data directory. Start from a normally writable accepted-v2 Architecture created through the current product. Record the source repository's HEAD, tracked/untracked status, file modes, and checksums before authoring.

1. Open and, if needed, initialize the real project through WorkBraid. Record the exact accepted v2 bootstrap/current revision and verify the root workspace is normally writable.
2. Through structured UI, create several meaningfully connected Components in root if needed. Establish enough global Relationships that later home moves visibly exercise ordinary versus **Lives in** boundary presentation.
3. Select a root-home Component and choose **Create detail diagram**. Enter a human-readable title and keep the change. Verify the normal accepted tree/map/index still shows the unchanged accepted Architecture, while the new pending Diagram is reachable through **Changes in progress**.
4. Move at least one Component home into the candidate-only detail Diagram. Verify the Component identity/documentation/Relationships remain present in pending context and no reference is silently left in root.
5. From a home inside that candidate Diagram, create and title a second-level detail Diagram and move a Component into it. Verify the pending hierarchy, candidate-relative target choices, and structured controls remain understandable without IDs/YAML/role language.
6. Rename one Diagram. Verify its identity/path and existing composition remain stable in the pending review details/diff.
7. Move an anchoring Component whose home owns the second-level detail Diagram to another valid destination. Verify the existing child/subtree follows the anchor and its Diagram identities/content do not change.
8. Deliberately attempt one move which would place an anchoring home beneath its own descendant. Choose **Review changes** and verify review is blocked with concise localized guidance, accepted Git/workspace remains exact, and every ordinary/Diagram pending change remains available.
9. Use the obvious **Fix** route, correct only the invalid move, and retain the rest of the pending set.
10. In the same pending set, make at least one ordinary Component Title or Description edit and one real outgoing-Relationship edit. Where a destination already has a canonical reference, also exercise reference-to-home conversion through a home move if the starting fixture makes this proportionate.
11. Recheck the normal accepted workspace: no pending Diagram, title, home, hierarchy, Component, or Relationship change has leaked into its tree/map/index/documentation.
12. Choose **Review changes** and record exact base commit, candidate tree, and pending generation. Verify **With changes** presents the complete nested candidate and marks added Diagram, Diagram title, home movement, detail link, and hierarchy context. Verify the real Relationship edit is distinct and composition-only internal↔boundary changes are not false Relationship deltas.
13. Select the candidate-only second-level Diagram, switch to **Before changes**, and verify fallback to the nearest surviving base ancestor or base root with the restrained note and no candidate-only composition/documentation/topology leakage. Toggle back and verify the whole selected side remains snapshot-unified.
14. Select representative Diagram/home/Relationship changes and verify they focus the relevant exact canonical diff region(s). Inspect the complete unified diff and exact expected Diagram/Component tree changes. Verify unrelated Component blobs/modes/paths and child/subtree identities remain exact through bounded technical inspection.
15. Deliberately choose **Update architecture** once. Record the exact successor revision and parent diff. Verify Diagram tree, breadcrumbs, selected navigation, map/index/documentation, homes, boundaries, Relationships, and revision advance together.
16. Navigate root, first detail, and second-level detail through tree, breadcrumbs, Component drill-down, and back navigation. Verify **Edit component**, **Open _diagram_**, home context, canonical-reference context, and external **Lives in** captions remain distinct and truthful.
17. If proportionate to the integrated changes, create one controlled valid external accepted-v2 advancement and verify it remains invisible until explicit **Refresh**, then all accepted projections switch together without fallback or source mutation.
18. Stop WorkBraid completely. Start a genuinely fresh process using the same application-data directory, reopen the project, and verify the identical accepted SHA/tree, root/detail hierarchy, Diagram IDs/titles/paths, parent anchors, homes, Component IDs/docs, Relationships, boundaries, and navigation reconstruct solely from canonical Git.
19. Compare the source repository's HEAD, tracked/untracked status, file modes, and checksums with the starting record. They must be exact. Inspect SQLite and verify it contains only approved source/Architecture association state and no Architecture, Diagram, Component, Relationship, hierarchy, pending, review, or layout projection.
20. Record an explicit human **PASS** or **FAIL**. P2.3 completes only on **PASS**.

Keep the checkpoint cohesive and proportional. Do not replay the P2.1 invalid-v2 matrix, the P2.2 initialization/legacy-setup matrix, or every earlier Refresh/CAS edge case.

## Explicit exclusions

P2.3 does not introduce:

- structured reusable-reference add/remove; that remains P2.4;
- Diagram deletion, replacement, detach, or general lifecycle;
- multiple Diagram parents, DAG hierarchy, reusable child Diagrams, or hierarchy reconciliation;
- more than one appearance of a Component in a Diagram;
- more than one detail Diagram per home anchor;
- Component deletion;
- Relationship or appearance IDs/lifecycle/taxonomy;
- persisted/project-scoped pending changes, multiple pending sets, partial discard, undo/redo, merge, or rebase;
- persisted/manual coordinates, layout, sizes, routes, bend points, shapes, annotations, or renderer state;
- graphical Component, Relationship, appearance, or hierarchy editing;
- Diagram kinds, UML/class semantics, isometric/3D rendering, or another WorkBraid vertical;
- pending/draft overlay on the normal accepted workspace map;
- URL-backed restoration, semantic/rendered Markdown diff, syntax highlighting, or unrelated recorded polish;
- a generic Diagram, graph, membership, hierarchy, validation, migration, workflow, command, transaction, event, permission, or capability framework;
- a second parser, candidate builder, review object, acceptance path, browser-owned Architecture interpretation, SQLite Architecture projection, or source-repository write.

P2.3 stops after independent review, integration, ordinary checks, restart-backed real human checkpoint, completion record, and explicit human approval. P2.4 remains unstarted.

## Execution result

Status: Complete — human checkpoint **PASS** on 2026-08-27

- Exact completed-P2.2 prerequisite: `dd75ceb8db15f1dafef33513d632b25315114c6b`.
- Approved docs-inclusive worker base and P2.3 execution-packet commit: `02aa3615871a03587eea84f25a9056f3d478d8f4`.
- Initial cohesive implementation: `a0635f8529d3ef55bd2c9e4e6fe64a06c39a765f`.
- Final integrated implementation after independently reviewed correctness, human-checkpoint UX, review-layout, and living-scenario corrections: `61dceb9e97d752285bfb67a22b96a8a85775d54b`.
- The implementation stayed on one pending Architecture base/generation, one complete-candidate constructor and version-aware validator, one immutable review binding, and the existing confirmation/stale/CAS/publication path. Detail creation, Diagram-title edits, and home moves use concrete Diagram-composition state; no Diagram draft, second candidate/review authority, child parent field, recursive subtree rewrite, or generic hierarchy/operation framework was introduced.
- Independent review: the initial implementation and every material correction returned to the same bounded worker and received fresh independent rereview. The complete implementation through `89ae04343583d4f27eafe487196d6c7f491d3b63` passed with no findings. The subsequent review-layout delta through `f0bcef68184f9a2b3a84ea766f17245c76a0f3ae` and living-scenario delta through `61dceb9e97d752285bfb67a22b96a8a85775d54b` each passed separate fresh rereview with no findings.
- Reviewer scrutiny confirmed candidate-relative nested composition, stable creation identity, exact reuse of untouched Component/Diagram blobs and modes, semantic/source-order fidelity for rewritten Diagrams without lossless-YAML machinery, source-home removal, destination reference-to-home conversion, parent-owned detail-link movement, exact subtree identity preservation, complete hierarchy validation, localized retained invalid work, Diagram-aware snapshot-unified review, synchronized mutation eligibility, and the absence of P2.4 behavior.
- Automated validation: **PASS** for `git diff --check`, `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go mod verify`, 94 ordinary frontend tests, the production frontend build, and both bounded P2.3 production-browser scenarios. Because Diagram-scoped Component creation made the old global **Add component** selector ambiguous, the living Phase 1 and Gate scenarios were proportionately scoped to the accessible **Diagrams and components** navigator; Phase 1 passed and the complete four-scenario Gate run passed at the final SHA. The production behavior and invariant assertions were unchanged. The existing production-chunk-size warning remains non-blocking.
- Resource safety: checks ran sequentially with one browser worker. No abnormal memory growth or swap use occurred. All WorkBraid, npm, Node, Vitest, Playwright, and Chromium checkpoint/test processes were stopped at completion.

### Human checkpoint evidence

- Runtime root: `/tmp/workbraid-p23-human.wr1MI2`; application-data directory: `/tmp/workbraid-p23-human.wr1MI2/app-data`; source project: `/tmp/workbraid-p23-human.wr1MI2/source-project`; private Architecture store: `/tmp/workbraid-p23-human.wr1MI2/app-data/architecture/7afc8c07-4461-4b73-9063-f342ef2a54e6.git`.
- The real built browser UI, loopback Go process, Git executable, filesystem, and SQLite were used. The normally writable parentless v2 bootstrap was `febd53d4c964da5c743519b2299a0e7117df4bbe`; the connected accepted starting Architecture before P2.3 composition was `f8557b0a854406acd5bfa82eb44c0019e8153f07`.
- Structured authoring created and titled nested detail Diagrams, moved Component homes, moved an anchoring home with its existing child subtree, retained exact Component identities/documentation/Relationships, combined Component and real Relationship edits, and kept the normal accepted workspace unchanged before review. Candidate-relative Diagram creation and Component creation in an active pending Diagram remained in the same backend-held pending set.
- A deliberately invalid descendant move was retained and blocked Review with localized Diagram-composition guidance. The direct self-detail destination is now omitted and independently rejected under synchronized backend authority; deeper descendants remain selectable because other pending moves can make them valid. An invalid deeper move immediately presents one coherent attention state rather than a hollow or flattened candidate projection. Correction retained the rest of the pending work.
- Human-reviewed binding: base `f8557b0a854406acd5bfa82eb44c0019e8153f07`, candidate tree `d944cd822e40ccc5f349a66b04b39554dbc4c2a6`, generation `11`. **Before changes** / **With changes**, candidate-only fallback, Diagram tree and composition markers, exact diff focus, real Relationship delta, internal/boundary presentation, and accepted-only normal workspace all passed.
- One deliberate acceptance produced exact revision `24512a013bad4a371f1c205ef7db9dac3a1718f3`, parent `f8557b0a854406acd5bfa82eb44c0019e8153f07`, and tree `d944cd822e40ccc5f349a66b04b39554dbc4c2a6`. The accepted tree contains the expected root, Runtime detail, Gateway Internals detail, five Components, exact Relationships, and only `100644` canonical files. Untouched Component blobs and modes were reused exactly.
- Accepted navigation through root, first-level, and second-level detail Diagrams passed. Breadcrumbs, Diagram tree, **Edit component**, **Open _diagram_**, home context, canonical-reference context, external **Lives in** captions, retained review context, Changes/external-reference dock behavior, and neutral clear selection remained coherent. Human-requested contextual form alignment, action hierarchy, underlined actions, grouped pending Diagram presentation, and immediate invalid-composition attention were corrected and rereviewed without expanding the Diagram domain.
- After a genuine process stop and rebuild from final implementation `61dceb9e97d752285bfb67a22b96a8a85775d54b`, reopening the same application-data directory reconstructed exact accepted revision `24512a013bad4a371f1c205ef7db9dac3a1718f3` and its hierarchy, identities, homes, documentation, Relationships, boundaries, and navigation solely from canonical Git.
- A final review-only regression check on the rebuilt process bound base `24512a013bad4a371f1c205ef7db9dac3a1718f3`, candidate tree `4e48170c19ac8ac2870397c7f4eb1327ae2e32cc`, generation `1`. Multiple derived boundary nodes received distinct stable automatic positions and no longer overlapped. The temporary title candidate was then whole-set discarded; accepted revision `24512a013bad4a371f1c205ef7db9dac3a1718f3` remained exact with no pending changes.
- Explicit external Refresh was not replayed manually because P2.3 did not change Refresh semantics and the final correction was review-presentation-only. The required real-Git handler/browser evidence for valid external advancement, synchronized stale pending behavior, and atomic snapshot replacement passed in automated validation.
- Source isolation: **PASS**. Source HEAD remained `74b20a88be61c983a5cf8092f88c92cf83aad974`; status remained exactly `?? local-notes.txt`; tracked modes/blobs remained `100644 ff1e184894e55a516886a274e1e7778ed519cb7f README.md` and `100644 5b6ef892cebb4fe8427a246f6af265a04a82423b src/main.go`. SHA-256 values remained `eb0832cf3bea68738e71fd4bdecc5040b55d6a0bcc412b835526917bc7bb4845` for `README.md`, `b16292eca78e7c1f040ff51bd3966bbb3b350a15b4a7474b8e6d73922005c591` for `src/main.go`, and `20d7989a8b60e781c23f1f778f52d33657fdf87a397abc5d50516885e44a22e2` for `local-notes.txt`.
- SQLite isolation: **PASS**. The only table remained `source_architecture_associations`, containing exactly the source-root/store-ID association above and no Architecture, Diagram, Component, Relationship, hierarchy, pending, review, navigation, graph, or layout projection.
- Human checkpoint result: **PASS**.
- Scope: no structured reusable-reference authoring, Diagram deletion, multiple parents/DAG, persisted pending or layout state, coordinates/routing/shapes, graphical authoring, Diagram kinds, UML, isometric renderer, generic hierarchy/graph/workflow framework, or P2.4 implementation entered P2.3.

P2.3 is complete. Stop here; P2.4 remains unstarted.
