# Post-Gate Architecture Phase 2.2 execution packet

Status: Approved

Architecture baseline: `docs/architecture-v0.md`

UI baseline: `docs/ui-v0.md`

Roadmap: `docs/architecture-post-gate-roadmap.md`

Approved superseding continuation plan: `docs/plans/architecture-post-gate-phase-2-continuation.md`

Historical original Phase 2 plan: `docs/plans/architecture-post-gate-phase-2.md`

Exact planning prerequisite: `14bf828289557230b1287c9d7c7906156b30ffae`

Worker base: `fcee8420c268d450de7f77a32b4375eb784b80a1`

Target: P2.2 — Writable v2 foundation and legacy Diagram setup

Architecture or product changes discovered during implementation require explicit human approval rather than silent changes to this packet, its plan, or its baselines.

## Execution discipline

After human approval:

1. Mark this packet Approved and commit it alone on top of exact planning prerequisite `14bf828289557230b1287c9d7c7906156b30ffae`.
2. Record that clean docs-inclusive commit as the exact worker base.
3. Dispatch exactly one implementation worker from that base.
4. Require one cohesive conventional implementation commit or a small review-correction chain rooted at that exact base; do not split initialization, backend, frontend, or tests into independent product workers.
5. Send the complete committed implementation to one fresh independent reviewer who did not implement it.
6. Return every material finding to the same bounded worker, then require fresh rereview of the corrected complete range.
7. Integrate only the exact independently approved implementation.
8. Run the bounded ordinary checks once from the clean integrated tree.
9. Perform the real human checkpoint, with direct-v2 initialization and authoring as its primary workflow.
10. Record exact provenance and the explicit human result separately in this packet.

Stop after P2.2. Do not prepare, dispatch, or implement P2.3 until P2.2 is independently reviewed, integrated, and explicitly human-approved.

## Exact worker brief

Before editing, verify the exact clean worker base and read completely:

- `AGENTS.md`;
- `docs/architecture-v0.md`;
- `docs/ui-v0.md`;
- `docs/architecture-post-gate-roadmap.md`;
- approved `docs/plans/architecture-post-gate-phase-2-continuation.md`;
- historical `docs/plans/architecture-post-gate-phase-2.md` only to understand what P2.1 executed and what this continuation supersedes;
- this packet;
- completed `docs/plans/architecture-post-gate-phase-2.1-execution.md` for the exact loader/navigation implementation and provenance;
- completed `docs/plans/architecture-post-gate-phase-1.md` and `docs/plans/architecture-post-gate-phase-1-execution.md` for candidate-aware review;
- completed `docs/plans/architecture-i2.2.md`, `docs/plans/architecture-i2.3.md`, `docs/plans/architecture-i3.1.md`, `docs/plans/architecture-i3.2.md`, `docs/plans/architecture-i3.3.md`, and `docs/plans/architecture-increment-4.md` for authoring fidelity, pending state, CAS, workspace, Refresh, restart, source, and SQLite boundaries.

The alpha-stage living baselines and superseding continuation plan govern P2.2. Completed plans and records remain historical and must not be edited. Executable tests and production-browser scenarios are living verification and must be updated when they assert superseded v1 bootstrap or current-v2 **View only** behavior.

Implement only the complete writable-v2 foundation and bounded legacy **Set up diagrams** path described below.

### Direct format-v2 initialization

- Keep initialization an explicit human action using the existing source-root association, private-store location, real-Git bootstrap, and accepted-ref publication path.
- Create one parentless accepted bootstrap containing exactly:
  - normative v2 `architecture.yaml` at mode `100644`;
  - one newly generated stable root Diagram ID named by manifest `root_diagram`;
  - one ordinary `diagrams/root.yaml` blob at mode `100644` with that ID, a title initially derived from the project name, and no appearances;
  - no Components and therefore no Relationships.
- Treat `diagrams/root.yaml` only as the WorkBraid creation-time filename. Do not infer root from the filename, reject a valid externally authored alternative filename, or give the path identity.
- Preserve existing initialization truthfulness and concurrency behavior: an already-present `accepted` revision is loaded, incomplete/invalid state is not silently repaired, ref/object creation is not accepted without successful accepted-ref creation, and retry does not create another canonical bootstrap.
- Publish the initialized snapshot only after loading and completely validating the exact created v2 commit through the P2.1 version-aware loader.
- Do not create an intermediate v1 commit, migration record, SQLite projection, checkout, or source-repository file.

### One pending Architecture authority and writable v2 candidate path

- Preserve one backend-held pending Architecture change set with one exact accepted base, one generation, one `ConstructCandidate` path, one complete candidate tree, and one validation/review/confirmation/CAS path. Do not treat the current literal `ComponentChange` struct as the authority boundary.
- Extend the existing pending object with only the minimum concrete typed state needed to represent non-Component facts honestly. **Set up diagrams** is an Architecture format/composition transition, not a fake Component edit. A new Component's required home insertion is Diagram composition associated with that creation, not a Relationship or synthetic Component mutation.
- Keep the representation concrete to P2.2. Do not add a generic pending-operation algebra, command/event sourcing, pending-kind framework, Diagram draft, second candidate builder, migration object graph, sentinel Component ID, or fake Component change.
- Leave the single pending Architecture authority straightforward for P2.3 to extend with real Diagram-composition operations without redesigning P2.2-specific hacks; do not pre-implement those operations or add future-only types.
- Begin every candidate from the exact accepted base tree. Reuse unchanged root entries, Component blobs/modes/paths, and Diagram blobs/modes/paths exactly; serialize only independently edited/new Components and the Diagram whose home composition must change.
- Preserve the P2.1 loader as the one candidate validation/parser path. The complete resulting v2 tree must validate before an immutable candidate snapshot may be published or reviewed.
- Restore existing structured Component Title, Description, outgoing-Relationship, and Component-creation operations for current valid accepted v2 snapshots.
- Preserve all established Component semantics:
  - Title normalization/projection and ATX/Setext behavior;
  - complete frontmatter preservation for Title/Description-only edits;
  - untouched H1/body byte preservation from explicit changed-field intent;
  - creation-time stable ID/path and `100644` mode;
  - existing-file path/mode preservation;
  - exact unchanged-blob reuse.
- Preserve all established Relationship semantics:
  - target by stable Component identity;
  - resolution against the complete candidate, including pending-new targets;
  - exact valid label fidelity, including whitespace/newlines/Unicode;
  - non-empty-after-trimming validation only;
  - surviving source-order fidelity without order meaning or Relationship IDs.
- A new Component receives exactly one home appearance in the requested active Diagram. Validate that Diagram ID from the server's current accepted/pending state; it is an operation target, not browser authority. Use root only when the request genuinely has no Diagram context.
- A v2 pending candidate may combine existing Component edits, Relationship edits, pending-new Components, and their home insertions. Every relationship target and every home resolves against that one complete candidate.
- P2.2 does not expose detail creation, Diagram rename, arbitrary home movement, or reference add/remove. Existing accepted nested Diagrams, detail links, home/reference appearances, and derived boundaries must remain valid and render unchanged while ordinary authoring proceeds.

### Synchronized authoring eligibility and pending state

- Remove P2.1's temporary valid-current-v2 mutation rejection and **View only** treatment only after the complete v2 candidate path works through review, CAS, and restart.
- Make eligibility decisions while holding the existing concrete backend state synchronization boundary. Use the current loaded project, accepted snapshot format and authority status, pending store/base, and requested stable target identity; never trust a client format, capability, writable flag, base SHA, or stale UI assumption.
- Keep known stale/non-current state and indeterminate Refresh knowledge above ordinary authoring. Neither becomes writable merely because v2 authoring exists.
- Mutations, Discard, Refresh, project switching, review capture, confirmation, WorkBraid CAS publication, and legacy setup must not interleave into mixed loaded/pending state. Keep this on the existing concrete lock; do not build a workspace transaction/event framework.
- Preserve generation invalidation and exact review binding. Any pending mutation after review makes that review unusable until a fresh review.
- Preserve dirty browser-editor navigation guards and project-switch protection. Do not add autosave, persisted drafts, or another browser-owned Architecture model.

### Bounded legacy **Set up diagrams** transition

- Valid current accepted v1 remains loadable and navigable through the existing implicit all-components map but exposes no normal Component/Relationship Add/Edit operation.
- An ordinary mutation request against accepted v1, including a late request from an older browser state, must create no pending change, Git object/ref change, source mutation, SQLite state, or loaded-snapshot mutation.
- When no pending set exists, expose one concise product action: **Set up diagrams** with brief language that it enables editing. Do not surface format, version, migration, schema, YAML, root ID, or step-by-step configuration language.
- Initiating setup atomically creates one backend-held pending operation bound to the exact v1 store/base/generation. Generate its stable root Diagram ID once and retain it across browser reload and repeated candidate construction while the backend process remains alive.
- Construct one complete v2 candidate containing only:
  - the exact normative v2 manifest retaining store/project values and adding the generated root identity;
  - one new root Diagram, using the WorkBraid `diagrams/root.yaml` creation convention and project-derived initial title;
  - one home appearance in root for every accepted v1 Component.
- Preserve every v1 Component path, blob, ordinary-file mode, ID, Markdown byte, outgoing Relationship declaration, exact label, and source order. Do not perform a Component/Relationship edit, pending-new Component addition, detail creation, or other Diagram operation as part of setup.
- Setup is an ordinary pending candidate whose complete result is v2. It may be discarded, reviewed, confirmed, accepted through the existing CAS path, published, refreshed, and reconstructed exactly like any other candidate. There is no migration-only authority or v2-to-v1 downgrade.
- If any non-setup v1 pending set somehow exists, retain it visibly as stale/read-only-style evidence without calling it current authoring: no Edit, Fix, Review changes, or Update architecture. Permit whole-set Discard only and keep **Set up diagrams** unavailable until discard.
- Do not reinterpret, merge, accept, persist, recover, or reconcile that defensive pending state. Do not create a general pending-kind or migration framework beyond the minimum concrete distinction required to recognize the setup candidate.

### Candidate-aware review and acceptance

- Reuse the completed Phase 1 review response, immutable base/candidate snapshots, exact unified diff, binding, confirmation payload, stale checks, commit construction, accepted-ref CAS, immediate pending consumption, publication, and post-CAS recovery path.
- Capture the current binding and all immutable response projection/comparison state atomically under the existing backend synchronization boundary, then serialize after releasing it.
- For ordinary v2 changes, switch Diagram tree, selected Diagram, index, map, documentation, appearances, boundaries, Relationships, and revision together between exact bound base and candidate snapshots.
- Component content comparison remains exact structured Title plus exact Markdown body. Relationship facts remain the exact `(source ID, target ID, exact label)` multiset including multiplicity.
- Component creation includes its one home insertion, but ordinary unchanged Components are not content-changed merely because another Component alters Diagram composition.
- During non-empty v1 setup review:
  - **Before changes** is the exact bound v1 implicit all-components map with no fabricated Diagram identity;
  - **With changes** is the exact generated v2 root and home composition;
  - unchanged Components are not classified content-changed merely because they receive homes;
  - unchanged Relationship facts are not classified added/removed merely because the candidate has Diagram composition;
  - generated manifest/root/home state is presented as setup/Diagram composition change;
  - the complete exact unified diff remains directly inspectable and authoritative.
- Candidate-only Diagram state must never leak into the v1 base side. The restrained existing candidate-only context treatment remains sufficient.
- Visual map failure remains non-blocking when the validated candidate, exact diff, review details, and existing acceptance action remain available.
- Preserve explicit stale-base observation before commit creation, final mandatory CAS, CAS-success boundary, failed-CAS classification, response-loss ambiguity handling, and duplicate-accept prevention exactly.

### Workspace and living verification

- Newly initialized and current valid v2 Architecture uses the existing map-centered drafting workbench with its root tree/breadcrumb context and normal Add/Edit/Changes actions. It receives no setup explanation or P2.1 **View only** notice.
- Accepted v1 stays readable with one **Set up diagrams** action when eligible. Setup and defensive pending evidence reuse **Changes in progress** and the contextual working pane rather than adding a migration screen/dashboard.
- Preserve safe Markdown, compact navigation, duplicate-title disambiguation, clear/deselect, dirty-editor protection, whole-set discard, project-switch behavior, Refresh, stale/read-only behavior, and source/SQLite isolation.
- Update executable Gate, Phase 1, and P2.1 tests/scenarios proportionately when they assert v1 bootstrap or current-v2 **View only**. Preserve their still-valid authority, isolation, exact review/CAS, safe-rendering, Refresh, restart, and SQLite evidence.
- Do not edit completed documentation records. Do not create parallel legacy browser scenarios for unsupported v1 authoring or convert existing scenarios into a generic harness.
- Treat the P2.1 breadcrumb hover-contrast note as non-gating; change it only if the exact touched UI naturally requires a bounded correction.

## Worker acceptance criteria

The implementation is ready for independent review only when:

- new-store initialization creates one exact parentless v2 bootstrap with one empty manifest-selected root and no intermediate v1 accepted revision;
- initialization remains explicit, retry-safe, accepted-ref authoritative, source-isolated, and fully reconstructable;
- the P2.1 loader remains the sole accepted/candidate v1/v2 parser and validator;
- current valid v2 supports structured Title, Description, outgoing-Relationship, and Component-creation operations through the one pending/candidate/review/CAS path;
- a new Component receives exactly one home in active Diagram or genuine root fallback while unrelated Diagram/Component entries are reused exactly;
- mixed v2 Component/Relationship/new-home candidates validate, review, accept, publish, Refresh, and restart as one immutable revision;
- accepted v1 is readable but not normally writable, and direct/late mutation requests create no state;
- eligible **Set up diagrams** creates only one stable v2 root/home candidate over exact v1 and preserves all Component/Relationship facts and bytes;
- setup is unavailable with any pending set; defensive non-setup v1 pending work is read-only/discard-only and cannot review or accept;
- setup review classifies only generated Diagram/setup composition while unchanged Component content and Relationship facts remain unchanged;
- binding invalidation, stale protection, CAS boundary, response-loss behavior, source isolation, and SQLite operational-only state do not regress;
- executable scenarios assert the new living baseline without weakening still-valid invariant coverage;
- no P2.3 nested-composition authoring, P2.4 reference authoring, generic framework, approved-doc edit, dependency/build/database artifact, or source-repository mutation enters the diff;
- the isolated worker tree is clean after conventional commit(s).

## Required automated evidence

Use bounded production-path evidence. Do not replay the complete P2.1 invalid-reader matrix or every completed I2/I3 corruption case.

### Architecture and real Git

Use temporary real bare Git stores and the real executable with fixed controlled arguments.

- Initialize from a genuinely absent store and prove exact parentless v2 manifest/root tree, both WorkBraid-created modes, root UUID consistency, empty appearances, no Components, accepted-ref identity, one bootstrap commit, and no source mutation.
- Retry initialization after success and after one representative pre-ref failure. Prove no second accepted bootstrap, fallback, or repair behavior.
- Rename/rewrite the root Diagram path through controlled valid external Git and prove the loader still follows manifest root rather than `diagrams/root.yaml` naming.
- Starting from a representative accepted v2 hierarchy with existing detail and reference appearances, construct a mixed candidate with Title or Description edit, Relationship edit, pending-new relationship target, and new Component home in a selected non-root Diagram.
- Prove exact Component H1/body/frontmatter/label/order/path/mode fidelity and unchanged blob reuse proportionately; prove only the selected Diagram blob changes for new-home insertion and unrelated Diagram blobs/modes are exact.
- Prove invalid/missing active Diagram target and representative Component/Relationship invalidity retain the complete pending set and publish no snapshot/ref.
- Construct setup over a non-empty connected v1 revision. Prove exact store/project retention, stable generated root, one home per accepted Component, exact Component tree-entry/blob/mode preservation, exact Relationship facts, valid v2 load, complete diff, and no extra authoring change.

### Production handler and synchronization

- Exercise new initialization, open, same-process browser reload, explicit Refresh, and genuinely new backend/SQLite connection using the same application-data directory.
- Send structured v2 Title, Description, Relationship, and new-Component operations through production handlers and prove one pending generation accumulates all changes atomically.
- Validate active-Diagram home assignment from synchronized server-owned state. Race mutation with Refresh/project switch or Discard and prove one coherent loaded/pending outcome.
- Race review capture with mutation/Discard and preserve the exact coherent binding behavior already approved.
- Exercise setup initiation, same-process reload, discard, review, confirmation, accepted CAS, post-CAS response/publication failure, and fresh reopen through the production handler.
- Race setup with ordinary late v1 mutation, project switching, Refresh, Discard, and confirmation proportionately under the existing state lock. Prove setup eligibility and format are server-owned and no hidden v1 pending work appears.
- Produce a defensive non-setup v1 pending state through a real supported authority transition, such as preserving old-base pending work while Refresh adopts external v1. Verify inspection plus whole-set Discard only; no edit/review/accept/setup until discard.
- Verify missing/wrong Origin remains rejected without CORS or state change.

### Review projection and frontend

Use separate runner-owned cases through only the ordinary documented frontend command.

- Fresh v2 initialization opens the writable root workspace with no **View only** or setup explanation and exposes normal Add/Edit/relationship actions.
- Current valid v2 authoring accumulates mixed changes, reloads from the same backend, retains dirty-navigation guards, and leaves normal accepted map/tree/documentation exact before acceptance.
- Component creation passes stable active-Diagram intent but does not supply or control format/capability/authority; backend responses remain the Architecture source.
- Valid v1 shows readable implicit map and **Set up diagrams** only when eligible, with no normal Add/Edit/relationship action or implementation terminology.
- Setup pending state survives browser reload, is discardable, and reaches the existing review workspace. Its Before/With views and exact diff are snapshot-unified.
- A connected non-empty v1 setup fixture yields no false content-changed Component or Relationship added/removed indicators; root/home setup composition remains clear and selectable.
- Defensive non-setup v1 pending evidence exposes View/Discard but no Edit/Fix/Review/Update/Setup action.
- Known stale/non-current and indeterminate Refresh presentation outrank normal writable/setup states.
- Map failure retains complete diff and acceptance; safe Markdown/resource behavior and project-switch/dirty-editor protection do not regress.
- Update existing executable scenarios and assertions rather than preserving superseded bootstrap/View-only behavior or adding parallel legacy scenarios.

Run only `npm test` from `frontend/`. Do not invoke raw `vitest`, `npx vitest`, manual parameterized render/unmount loops, or unattended retries.

### Integrated production-browser evidence

Add one bounded runner-owned P2.2 production-browser scenario or equivalently focused case. It must use the production frontend build served by the real loopback Go process, real Git, real bare private stores, real filesystem state, real SQLite associations, and completely stopped/restarted Go processes.

The primary path must:

1. start with a fresh application-data directory and real throwaway source repository;
2. initialize through the UI and verify exact direct-v2 root bootstrap through bounded technical inspection;
3. author connected Components through structured UI, including a pending-new target and mixed content/Relationship work;
4. verify pending state does not alter the accepted normal workspace;
5. review exact binding, candidate visual state, and complete diff; accept once;
6. stop the process, start a genuinely new process with the same application data, reopen exact accepted SHA, and verify writable reconstructed root/components/relationships;
7. prove source HEAD/status/files/checksums and SQLite operational-only rows remain exact.

Add a bounded secondary legacy path in the same runner-owned scenario only if it remains clear and fail-fast; otherwise use one separate runner-owned P2.2 test case, not a manual lifecycle loop. It should open controlled non-empty v1, perform **Set up diagrams**, verify preservation/classification and exact diff, accept, restart, and reconstruct exact v2. Do not make legacy setup the main scenario or exercise an arbitrary compatibility matrix.

Existing Gate, Phase 1, and P2.1 scenarios are living code. Update their setup/assertions where needed and run them only when their shared production paths changed, retaining useful coverage without creating a generic harness.

Every scenario must own cleanup in `finally`, use bounded action/test timeouts, stop process groups on success/failure, and leave no WorkBraid, Playwright, Chromium, npm, Node, or Vitest child. If a runner grows abnormally or fails to terminate, stop immediately and report the test defect; do not raise limits, add permanent machine throttling, or retry blindly.

### Ordinary checks

Run once from the clean worker tree:

- `git diff --check`;
- `go test ./... -count=1`;
- `go test -race ./... -count=1`;
- `go vet ./...`;
- `go mod verify`;
- `npm test` from `frontend/`;
- `npm run build` from `frontend/`;
- the documented bounded P2.2 production-browser scenario(s);
- living Gate, Phase 1, and P2.1 scenarios when changed or when the shared production path requires regression evidence.

Run each ordinary command only once after focused development checks are green. Report every initial failure and correction truthfully. Do not commit generated build output, dependencies, databases, traces, screenshots, or runtime fixtures.

## Fresh independent reviewer brief

The reviewer receives the exact packet-inclusive worker base, committed worker head, complete diff, this packet, worker evidence, alpha baseline, approved continuation plan, historical original plan, and completed P2.1 record. Review in a fresh isolated worktree:

1. **Provenance and supersession:** exact base/head, approved docs untouched, P2.1 history preserved, living executable scenarios updated only for superseded behavior.
2. **Initialization authority:** exact direct-v2 bootstrap, one generated root, manifest-only root authority, parentless commit, accepted-ref success boundary, retry/failure truthfulness, no v1 intermediate.
3. **Single pending authority and interpretation:** one pending Architecture set/base/generation, one P2.1 version-aware loader, and one extended `ConstructCandidate` path. Confirm format transition and home insertion use minimal honest typed state rather than fake/sentinel Component changes, while avoiding a generic operation/pending-kind framework, Diagram draft, migration graph, second parser, review object, ref, or acceptance path. Verify the shape can accept real P2.3 Diagram-composition state without requiring removal of P2.2 hacks, but contains no future-only operations or types.
4. **v2 fidelity:** complete Component/Relationship semantics, active-Diagram home assignment, unchanged Component/Diagram blob/mode/path reuse, existing nested/reference composition preservation.
5. **Synchronized state:** server-owned eligibility; coherent mutation/setup/discard/Refresh/switch/review/confirm/CAS behavior; no client capability authority or hidden pending work.
6. **Legacy boundary:** v1 readable but not ordinarily writable; setup only with no pending; stable transition candidate; defensive non-setup pending is inspect/discard only; no merge, review, v1 acceptance, or framework.
7. **Review:** exact binding and snapshot unity; setup composition classification without false Component/Relationship deltas; complete exact diff; stale and CAS behavior unchanged; map failure remains non-blocking.
8. **Workspace:** writable current-v2 UI replaces temporary staging; v1 setup language is concise; stale/indeterminate priority, dirty guards, project switch, safe rendering, and accepted-only normal surfaces hold.
9. **Regression evidence:** executable Gate/Phase 1/P2.1 scenarios reflect the living baseline while retaining valid authority/isolation/restart evidence; no parallel unsupported legacy product.
10. **Scope and QA:** no nested Diagram composition authoring, reference authoring, layout, generic framework, committed artifact, source mutation, test loop, or resource anomaly.

Run each documented ordinary check once and the bounded production-browser evidence once. Report actionable findings and residual human-only risks. Any material finding returns to the implementation worker and the corrected complete range receives fresh independent rereview.

## Integration procedure

After a clear independent review:

1. verify the main worktree is clean at the exact packet-inclusive worker base;
2. integrate the exact reviewed implementation commit(s) without rewriting them;
3. verify the integrated tree matches the independently reviewed head;
4. verify approved baselines/plans/records are unchanged by implementation;
5. rerun `git diff --check` and the ordinary checks once;
6. start one fresh bounded real human-checkpoint runtime with fresh application data and source repositories;
7. stop before P2.3.

If integration differs from the reviewed tree, a still-approved workflow regresses, a test asserts obsolete behavior instead of being updated, or a required check fails, stop and return the exact issue through bounded correction and fresh review.

## Real human checkpoint

Use the built UI served by the real Go process, real Git executable, real private bare repositories, real filesystem state, and real SQLite association state.

### Primary direct-v2 workflow

1. Start with a fresh application-data directory and a real throwaway source repository containing tracked and untracked files. Record source HEAD, status, modes, and checksums.
2. Open the project, verify no link is known, and explicitly initialize Architecture.
3. Record the exact parentless accepted bootstrap SHA. Through bounded technical inspection verify normative v2 manifest, generated root identity, `100644` `diagrams/root.yaml`, empty appearances, no Components, and sole `accepted` authority.
4. Verify the root workspace is immediately writable: no **View only**, no setup explanation, and normal Add/Edit/relationship actions are available.
5. Through structured UI, create several connected Components including a pending-new relationship target. Combine one existing Component Description edit and one outgoing-Relationship edit in the same pending set.
6. Verify every new Component has one root home, browser reload retrieves the backend-held work, and accepted tree/map/index/documentation remain exact before review.
7. Deliberately make one existing validation error, verify actionable product guidance and retained pending state, then correct it.
8. Review the exact complete candidate. Verify root/index/map/documentation/Relationships are candidate-consistent, the complete diff includes Component and root-Diagram home changes, and review details bind exact base/tree/generation.
9. Deliberately Update architecture once. Verify accepted revision, root composition, Components, documentation, and Relationships advance together.
10. Using a separately controlled accepted-v2 fixture with an existing non-root Diagram and reference appearances, create a new Component while that detail Diagram is active. Verify its home is there, unrelated composition is preserved, review/accept works, and the workspace remains writable.

### Bounded legacy setup workflow

11. Open a separate controlled non-empty accepted-v1 Architecture containing connected Components. Verify readable implicit map/documentation, no ordinary Add/Edit/relationship action, and the concise **Set up diagrams** action.
12. Begin setup and record its exact review binding. Verify **Before changes** is the real implicit v1 map and **With changes** is one generated root containing homes for every Component.
13. Verify unchanged Component content is not marked changed and unchanged Relationship facts are not marked added/removed; generated manifest/root/home composition is visible and the complete exact diff preserves every Component blob/mode/path.
14. Discard once and verify accepted v1 remains exact; begin setup again, review, deliberately accept, and verify one transition-only v2 successor becomes writable.
15. Exercise one bounded defensive non-setup v1 pending case prepared through a real supported authority transition. Verify read-only inspection plus whole-set Discard only and no Setup/Edit/Review/Update until discard. Do not expand this into a compatibility matrix.

### Restart and isolation

16. Completely stop WorkBraid. Start a genuinely new process using the same application-data directory. Reopen each final v2 project and verify exact accepted SHAs, roots, existing nested navigation, Components, homes, documentation, Relationships, and writable actions reconstruct from canonical Git.
17. Verify all source repositories retain exact HEAD, status, files, modes, and checksums.
18. Verify SQLite contains only approved source-root/store-ID associations and no Architecture, Component, Relationship, Diagram, pending, migration, review, navigation, graph, or layout projection.

Human checkpoint completion requires explicit **PASS**. Record exact bootstrap, reviewed candidate, and final accepted SHAs.

## Explicit stop and exclusions

Stop after the human explicitly accepts or rejects P2.2. Do not prepare or begin P2.3 from this packet.

Do not introduce:

- ordinary accepted-v1 Component or Relationship authoring;
- composition of **Set up diagrams** with non-setup pending work;
- v2-to-v1 downgrade, migration state machine/table, generic migration framework, or conversion wizard;
- detail-Diagram creation, Diagram rename, arbitrary home movement, subtree reparenting, or hierarchy authoring;
- structured reference-appearance add/remove;
- Diagram deletion, multiple parents, Diagram DAG/reuse, repeated same-Diagram appearances, or multiple detail Diagrams per anchor;
- canonical/persisted layout, coordinates, sizes, routes, bend points, shapes, annotations, Diagram kinds, renderer state, or appearance identities;
- manual/graphical editing or pending/draft overlays on the normal accepted map;
- persisted/project-scoped pending work, review state, navigation, selection, or layout;
- Component deletion, Relationship IDs/lifecycle/taxonomy, raw YAML/frontmatter editing, file rename, or identity replacement;
- another parser, candidate builder, Diagram draft, review binding, acceptance path, accepted ref, or SQLite projection;
- a generic schema, graph, validation, navigation, membership, capability, transaction, event, repository, VCS, or workflow framework;
- watcher, polling, automatic Refresh, retry loop, fallback, repair/reset, merge/rebase/reconciliation, history, revert, proposal, export, or sync;
- URL restoration, syntax highlighting, rendered/semantic Markdown diff, themed-scrollbar work, or unrelated P2.1 polish;
- UML/class semantics, additional Diagram kinds, isometric rendering, Planning, Agent Control, or another vertical.

## Execution result

Status: Complete — human checkpoint **PASS** on 2026-08-25

- Alpha-stage Phase 2 baseline amendment: `456ec4a5655cd896a426b57e4bb1e643b6e52293`.
- Approved superseding continuation plan: `14bf828289557230b1287c9d7c7906156b30ffae`.
- Approved docs-inclusive worker base and P2.2 execution-packet commit: `fcee8420c268d450de7f77a32b4375eb784b80a1`.
- Initial implementation: `1182b64f1f00783a65ace482e535970d1a7945fe`.
- Final integrated implementation after independently reviewed correctness and human-checkpoint UX corrections: `5dd1f9a3e712dffc7e620c168a751ef227b3b681`.
- Human-approved UI-baseline clarifications made during the checkpoint are `a0cd41ca79d1413e9e714f6767c652095975b570` and `eb57bbcf4c0089a8f8a7134cc037a453787b210e`. They clarify canonical-reference language, external-location subtitles, and the distinction between edit and detail-Diagram navigation without changing the Diagram domain or portable format.
- Independent review: the initial complete implementation and every material correction returned through the same bounded worker and fresh independent rereview. Corrections covered synchronized writable-v2 transition boundaries, inert defensive legacy pending state, Diagram-scoped candidate review and relationship focus, accepted-map clear focus, retained Diagram review context, non-overlapping review docks, return from review to editing, stable changes-dock presence, canonical-reference language, distinct detail navigation, and presentation-only boundary captions. The final review of `cff9a105dd4a331641655ec53917a4326efd9de4..5dd1f9a3e712dffc7e620c168a751ef227b3b681` reported no findings and verified that captions remain snapshot-derived, non-interactive presentation rather than graph/domain state.
- Automated validation: PASS for `git diff --check`, full uncached Go tests, full race-enabled Go tests, Go vet, module verification, 85 ordinary frontend tests, the production frontend build, and all three bounded P2.2 production-browser scenarios. Living Gate, Phase 1, and P2.1 executable evidence was updated only where the alpha-stage baseline superseded bootstrap or writable-format behavior. Frontend checkpoint corrections reran the ordinary frontend suite, production build, and all P2.2 browser scenarios after each final correction. No abnormal resource use or lingering WorkBraid, Playwright, Vitest, npm, Node, or Chromium process was observed. The existing production-chunk-size warning remains non-blocking.

### Human checkpoint evidence

- Runtime root: `/tmp/workbraid-p22-human.5z8D6I`; application-data directory: `/tmp/workbraid-p22-human.5z8D6I/app-data`. The built UI was served by the real loopback Go process using real Git, real bare Architecture stores, real filesystem repositories, and real SQLite.
- Direct-v2 initialization: PASS. The primary source initialized to parentless accepted v2 bootstrap `910c744685c12798a4b7632f5222b7025c699af0`, with one generated root Diagram, an empty `100644` `diagrams/root.yaml`, no Components, and no intermediate v1 revision. The root workspace was immediately writable without **View only** or setup language.
- Primary writable-v2 candidate: base `910c744685c12798a4b7632f5222b7025c699af0`, candidate tree `2b19b194dc0f49398c590e756a8d1566dd1ab416`, generation `8`, accepted revision `17fcaba353602cf6d69c1c000c525f66662911d1`. Structured authoring combined connected Components, a pending-new target, Description and Relationship work, root-home insertions, validation failure/correction, exact review, one deliberate CAS acceptance, and accepted map/index/documentation/Relationship advancement.
- Existing nested-v2 preservation: PASS. A controlled accepted fixture at `3765243ddfe37e1026548fa5d39c6c954caf01d3` contained root/detail/empty Diagrams plus home/reference appearances and boundary Relationships. Creating Queue while System A was active produced candidate tree `a78f1d04d01c783901f56b0dde51cb682838fdad`, generation `2`, and accepted revision `91a6a37795749b168d4cb8bb7884af6c85a67e92`; Queue's home is System A and unrelated hierarchy/composition remained exact.
- Candidate-aware review and workspace composition: PASS. Diagram selection remained stable when review opened; changed Diagrams were marked; Diagram/index/map/documentation/relationship context switched together; exact diff remained inspectable; **Continue editing** returned from a held review without rebuilding it; Changes and External references share one collapsible bottom dock; a zero-change Diagram truthfully keeps the Changes dock with no visual changes. Clear selection remains neutral rather than reopening an arbitrary Component.
- Reference/boundary language and navigation: PASS. Canonical reference context reads **Included here · Lives in _home Diagram_** in subdued index text. Derived boundary nodes retain normal Component-title prominence plus a separate small muted **Lives in _home Diagram_** caption that follows the real node across Fit/pan/zoom/resize without becoming a graph node or intercepting clicks. **Edit component** remains primary; **Open _detail Diagram_** is a separately styled navigation action beneath it. The human accepted the final presentation; italic visibility remains a non-gating visual observation.
- Legacy setup: a controlled non-empty v1 accepted revision `235afe72e55f0d1018b735657e764d49cc1603ff` remained readable but not ordinarily writable. Its `100755` Gateway blob `2ee498153c8fad65196a3c3db1b633bff9c2b665`, `100644` Worker blob `6e2cc4ae84e60fccf66840d3fa8942d8f720ea8c`, exact Unicode/whitespace Relationship label, paths, modes, IDs, and Markdown bytes were preserved.
- The first setup review bound base `235afe72e55f0d1018b735657e764d49cc1603ff`, candidate tree `20a8538e196b4609f4e537755070ac1f242abb03`, generation `1`. Before changes was the real implicit v1 map; With changes added only generated root/home composition; no Component content or Relationship fact was falsely changed. Whole-set discard left accepted v1 exact.
- A second setup accepted transition-only revision `76ecd510b194cf10c45cda41d43957acbd485e51`, with parent `235afe72e55f0d1018b735657e764d49cc1603ff`, candidate tree `b811dc9f1cfe89e844d84425c91ff67ac3bd4843`, and generation `1`. The resulting v2 workspace was normally writable.
- Defensive legacy pending: PASS. A real v2 Description change was kept, external authority rewound to the valid v1 revision, and explicit Refresh preserved the old-base pending set as stale/read-only evidence. Edit/Fix/Review/Update and **Set up diagrams** were unavailable; whole-set Discard alone cleared it without altering Git or source state.
- Final legacy setup bound base `235afe72e55f0d1018b735657e764d49cc1603ff`, candidate tree `23a53f9db05a42f5853dd3f4772a78ff03545357`, generation `1`, and accepted exact v2 revision `ab321f0273cc08bec678756ffc99a92d3a947511`.
- Restart reconstruction: PASS. WorkBraid stopped completely and a genuinely new process using the same application-data directory reopened exact accepted revisions `17fcaba353602cf6d69c1c000c525f66662911d1`, `91a6a37795749b168d4cb8bb7884af6c85a67e92`, and `ab321f0273cc08bec678756ffc99a92d3a947511`. Roots, nested navigation, Components, homes, canonical references, boundary references, documentation, Relationships, and writable actions reconstructed from canonical Git.
- Source isolation: PASS. The three source repositories retained exact recorded HEADs, tracked entries/modes, untracked files, and SHA-256 values. There were no tracked or staged changes. WorkBraid did not modify a source repository.
- SQLite isolation: PASS. The only table is `source_architecture_associations`, with exactly the three expected source-root/store-ID rows and no Architecture, Component, Relationship, Diagram, pending, migration, review, navigation, graph, or layout projection.
- Human checkpoint result: **PASS**.
- Scope: no nested Diagram composition authoring, structured reusable-reference authoring, Diagram deletion, persisted pending state, canonical layout/routing/presentation, graphical editing, Diagram kinds, UML, isometric renderer, P2.3 behavior, another vertical, or generic migration/graph/workflow framework entered P2.2.

P2.2 is complete. Stop here; P2.3 remains unstarted.
