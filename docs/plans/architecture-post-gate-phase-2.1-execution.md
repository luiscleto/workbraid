# Post-Gate Architecture Phase 2.1 execution packet

Status: Approved

Architecture baseline: `docs/architecture-v0.md`

UI baseline: `docs/ui-v0.md`

Roadmap: `docs/architecture-post-gate-roadmap.md`

Approved Phase 2 plan: `docs/plans/architecture-post-gate-phase-2.md`

Exact planning prerequisite: `5ca24c23b799323182e9b3164d2cfbf498e19734`

Worker base: to be recorded after this packet is approved and committed alone on top of the exact planning prerequisite

Target: P2.1 — Load and navigate accepted v2 Diagrams

Architecture or product changes discovered during implementation require explicit human approval rather than silent changes to this packet, its plan, or its baselines.

## Execution discipline

After human approval:

1. Mark this packet Approved and commit it alone on top of exact planning prerequisite `5ca24c23b799323182e9b3164d2cfbf498e19734`.
2. Record that clean docs-inclusive commit as the exact worker base.
3. Dispatch exactly one implementation worker from that base.
4. Send the complete committed implementation to one fresh independent reviewer who did not implement it.
5. Integrate only after the review has no actionable findings.
6. Run the bounded ordinary checks once from the clean integrated tree.
7. Perform the real human checkpoint.
8. Record provenance and the explicit human result separately in this packet.

Stop after P2.1. Do not prepare, dispatch, or implement P2.2 until P2.1 is independently reviewed, integrated, and explicitly human-approved.

## Exact worker brief

Before editing, verify the exact clean worker base and read completely:

- `AGENTS.md`;
- `docs/architecture-v0.md`;
- `docs/ui-v0.md`;
- `docs/architecture-post-gate-roadmap.md`;
- `docs/plans/architecture-post-gate-phase-2.md`;
- this packet;
- completed `docs/plans/architecture-post-gate-phase-1.md` and `docs/plans/architecture-post-gate-phase-1-execution.md` for the current candidate-aware map/workspace and final provenance;
- completed `docs/plans/architecture-i3.1.md`, `docs/plans/architecture-i3.3.md`, and `docs/plans/architecture-increment-4.md` for accepted workspace, safe rendering, Refresh, restart, and source/SQLite boundaries.

Treat those documents as authoritative. Do not edit approved baselines, plans, historical records, or this packet. If implementation needs an unapproved format key, Diagram lifecycle rule, migration behavior, visual-authoring behavior, or other product/domain decision, stop and report it.

Implement only accepted-v2 loading and read-only navigation through the existing production paths.

### One version-aware accepted loader

- Extend the existing accepted-snapshot loader rather than adding a Diagram loader beside it.
- Dispatch manifest parsing by the exact supported version while retaining the complete closed v1 behavior unchanged.
- Parse the normative closed v2 manifest with exact keys `format`, `version`, `store_id`, `project`, and `root_diagram`.
- Discover v2 Components exactly through the existing non-recursive `components/*.md` path and parser.
- Discover Diagrams non-recursively as `diagrams/*.yaml`. Require one or more ordinary Diagram blobs and reject every other v2 accepted-tree path, nested directory, symlink, gitlink, or tree entry.
- Parse each Diagram with exact keys `id`, `title`, and optional `appearances`; parse each appearance with exact keys `component`, `role`, and optional `detail_diagram`.
- Validate YAML scalar types/tags rather than relying on coercion. Unknown keys, unsupported versions, malformed UTF-8/YAML, or wrong scalar/container types remain invalid or unsupported according to the established loader classifications.
- Accept existing ordinary regular-file modes without giving executable mode domain meaning. WorkBraid creates no v2 blob in P2.1.
- Preserve v1 Component Title, Markdown body, frontmatter, Relationship label/order/fidelity, filename, ID, path, and mode interpretation byte-for-byte. Do not create a second Component parser for v2.

### Complete v2 snapshot validation

Construct and publish an immutable v2 snapshot only after the complete accepted revision validates:

- Diagram UUIDs are valid and unique;
- `root_diagram` resolves to exactly one discovered Diagram;
- Diagram titles are strings and non-empty after trimming; titles need not be unique;
- appearance Component IDs resolve against the complete Component set;
- `role` is exactly `home` or `reference`;
- `detail_diagram`, when present, resolves to a discovered Diagram and occurs only on a home appearance;
- every Component appears exactly once as home across all Diagrams;
- a Component appears at most once in one Diagram and is never both home and reference there;
- reference appearances own no detail link;
- every non-root Diagram is targeted by exactly one home appearance, while root has no parent anchor;
- every Diagram is reachable from root and the hierarchy is acyclic;
- any Diagram, including root or a detail Diagram, may have zero appearances when the complete home and hierarchy constraints still hold;
- the existing complete Component and global Relationship validation remains exact.

Keep validation concrete to the closed v2 contract. Do not introduce a schema framework, generic graph validator, ORM, repository layer, or future Diagram-kind/layout types.

### Immutable Diagram projection

- Add only concrete immutable snapshot data needed for Diagram identity, title, appearances, parent anchors, child links, home lookup, and navigation.
- Root selection comes from the manifest. Root remains otherwise an ordinary Diagram.
- Derive the strict Diagram tree, breadcrumbs, and return anchor from validated home-owned detail links. Child Diagram files contain no duplicated parent state.
- For each selected Diagram, project its exact home/reference appearances, their Component titles/documentation context, and the global Relationships relevant to that Diagram.
- If both Relationship endpoints have canonical appearances in the selected Diagram, project the ordinary directed labelled edge.
- If exactly one endpoint appears, derive at most one boundary/external reference for the absent external Component within that Diagram. Every crossing Relationship occurrence to or from it connects to that one boundary reference with exact direction, label, and multiplicity.
- If neither endpoint appears, omit the Relationship from that Diagram.
- A canonical reference appearance counts as a real appearance and uses an ordinary Component node/edge. It is not rendered as a boundary reference.
- Boundary references have no canonical or Architecture identity. Any response/render key is projection-only, deterministic, scoped to the selected snapshot/Diagram, and never persisted or accepted.
- Activating a boundary reference resolves to the external Component's one home Diagram. Activating a home detail link resolves to its child Diagram.
- Supply the browser with backend-resolved Diagram tree, appearances, navigation targets, ordinary edges, and boundary projections. The browser may render those supplied facts but must not parse YAML, infer home ownership, resolve hierarchy, or independently interpret Architecture semantics.

### Durable workspace navigation

- Extend the existing map-centered drafting workbench rather than appending a Diagram screen beneath it.
- For v1, retain the existing implicit all-components map, compact Component index, structured authoring, Changes in progress, and review workflow unchanged. Do not fabricate Diagram identity or show unnecessary Diagram navigation.
- For v2, open at the exact root and place a compact project-style Diagram tree and breadcrumbs in the existing navigator/workbench composition.
- Selecting a Diagram switches its tree selection, breadcrumb, Component index, map topology, selected documentation context, ordinary edges, and boundary references together from the same immutable accepted snapshot.
- Selecting a home or canonical reference appearance opens the one canonical Component document. Selecting a boundary reference navigates to that Component's home Diagram and focuses it. Drilling into a detail Diagram and returning to its parent identifies the anchor Component.
- Present canonical reference appearances with concise **Also shown here** language and derived boundary references with concise **Lives in** language. Do not surface YAML roles, UUIDs, filenames, root keys, or hierarchy internals as normal chrome.
- Disambiguate duplicate Diagram or Component titles only with the minimum ancestor, anchor, filename, or shortened-ID context needed at the collision.
- Preserve current safe Markdown rendering, resource-request prevention, clear/deselect, deterministic automatic layout, map failure degradation, project identity, explicit Refresh, project switching, and narrow structural fallback.
- Keep selection, expansion, pan, zoom, fit, layout, and breadcrumb focus disposable browser state. Add no URL, local-storage, SQLite, Git, or backend layout persistence.

### Explicit P2.1 read-only staging

- Accepted v2 is navigable but not authorable in P2.1. Do not expose **Add component**, **Edit**, relationship mutation, a fresh editable **Changes in progress** task, Diagram mutation, or another accepted-v2 mutation control as usable when no v2 candidate path exists.
- Use one compact product-language treatment in the v2 workspace rather than disabled controls or repeated warnings: heading **View only** with body **You can explore this architecture, but changes are not available here yet.** Keep it subordinate to the actual Diagram workspace rather than making it a blocking page.
- Show that staging treatment only while WorkBraid holds a valid current accepted-v2 snapshot. Known stale/non-current state takes precedence and retains the existing conspicuous non-current/read-only language and actions; never make it look like ordinary P2.1 View only. An indeterminate Refresh failure likewise retains the existing **WorkBraid couldn't check for architecture changes** current-knowledge semantics and is not replaced by or hidden behind the staging notice. Preserve already-known stale state through an indeterminate failure as before.
- Existing accepted-v1 mutation controls and the complete v1 pending/review/acceptance path remain visible and functional exactly as before.
- If explicit Refresh conclusively adopts v2 while an older-base v1 pending set exists, preserve that already-approved stale state exactly: it remains inspectable and read-only through Changes in progress, any review stays invalid, and whole-set Discard remains available. It does not make accepted v2 authorable and must not be hidden or stranded.
- Do not build a generic capability, permission, role, feature-flag, or version-negotiation UI system. This is one concrete format check for the P2.1 staging boundary.
- Backend mutation handlers must decide eligibility from the server's current loaded project, immutable snapshot format, and authority state while holding the existing concrete application synchronization boundary. Never trust or accept a browser-supplied format, version, capability, editable flag, or stale client assumption as authority. Reject an attempted accepted-v2 Component/Relationship mutation safely and atomically, without creating or changing pending state, Git objects/refs, source files, SQLite, or the loaded snapshot. In particular, a late old-v1 request arriving after Refresh or project switching has moved backend state to accepted v2 cannot create hidden pending work. Map that exceptional direct/stale-client response to product language such as **Changes are not available for this architecture yet.** Do not expose parser/version/schema terminology.
- Navigation, documentation, map interaction, Refresh, opening another project, and existing non-canonical workspace actions remain available where their current invariants permit them.

### Refresh, restart, and authority

- Preserve `refs/heads/accepted` as sole authority. Do not inspect `HEAD`, a working tree, SQLite, or another ref for current Architecture.
- Reuse the existing explicit protected Refresh mutation and application-state synchronization boundary.
- Resolve accepted, load and completely validate the exact v1/v2 revision through the same loader, perform the existing mandatory final ref observation, and atomically publish one replacement snapshot only under the approved Refresh semantics.
- A valid v2 fast-forward, rewind, or non-linear external replacement is adoptable without ancestry policy.
- Invalid/unsupported/missing and indeterminate Refresh outcomes retain their established truthful read-only behavior. Do not add repair, fallback, initialization, retry loops, watchers, or polling.
- A genuine new backend/application instance using the same application-data directory must reopen and reconstruct the exact accepted v2 revision, tree, navigation, Component documents, appearances, Relationships, and boundaries.
- P2.1 does not construct, review, or accept v2 candidates. It does not change the existing direct WorkBraid CAS publication path for v1.

### Scope and repository integrity

- Do not modify the user's source repository.
- Do not add SQLite Architecture, Diagram, relationship, navigation, graph, layout, or document projection.
- Do not write Diagram files, migration objects, v2 candidate blobs, or accepted refs from WorkBraid in P2.1.
- Do not edit approved docs, historical records, or existing Gate/Phase 1 browser scenarios merely to describe implementation.
- Do not commit dependencies directories, build output, databases, screenshots, traces, or temporary Git/browser/runtime artifacts.
- Follow the repository's runner-owned asynchronous test rule. Do not use manual parameterized render/unmount/mock-reset loops or raw/alternate Vitest invocations.

## Worker acceptance criteria

The implementation is ready for independent review only when:

- one version-aware accepted loader parses both exact v1 and normative v2 without changing valid v1 behavior;
- complete v2 validation enforces the closed tree/schema, IDs, one root, one home per Component, appearance uniqueness, detail-link restrictions, strict reachability, and acyclicity before snapshot publication;
- one immutable v2 snapshot supplies Diagram tree, breadcrumbs, selected-Diagram index/map/documentation, appearances, ordinary Relationships, boundary references, and navigation targets;
- crossing Relationship occurrences collapse onto one derived boundary reference per absent external Component without losing exact direction, label, or multiplicity;
- canonical reference appearances use ordinary nodes/edges and **Also shown here**, while derived references use **Lives in** and navigate home;
- v2 opens at root and supports tree, breadcrumb, drill-down, back/anchor, Component, canonical-reference, and boundary-reference navigation;
- a valid current accepted-v2 snapshot truthfully appears View only with no usable mutation controls, while known-non-current and indeterminate Refresh states retain their existing higher-priority product semantics;
- direct/stale mutation requests are judged from synchronized server-owned loaded state and rejected without any state change, while any pre-existing stale old-base pending set remains inspectable/discardable;
- accepted-v1 authoring, pending changes, candidate review, CAS acceptance, Refresh, discard/project switching, safe Markdown, and Phase 1 workspace behavior do not regress;
- valid external v2 Refresh and a genuine fresh-process reopen reconstruct the exact accepted state;
- no v2 candidate construction, migration, Diagram authoring, reference authoring, persisted layout, or P2.2 behavior exists;
- source and SQLite isolation remain exact and the worktree is clean after one conventional implementation commit.

## Required automated evidence

### Architecture loader with real Git

Use temporary real bare Git stores, controlled ordinary Git objects/refs, and the production accepted loader.

- Load one representative valid v2 tree with a manifest-selected root, empty Diagram, nested detail hierarchy, duplicate Diagram/Component titles, home and canonical reference appearances, cycles/parallel global Relationships, internal edges, and several differently labelled/parallel crossing occurrences to one absent Component.
- Prove one derived boundary reference is produced for that absent Component and every crossing edge retains direction, exact label, and multiplicity.
- Prove canonical reference appearance changes the same facts to ordinary edges without copying or altering the global Relationships.
- Prove boundary activation resolves the external Component's home and detail/back navigation derives the exact parent anchor.
- Cover a bounded table of distinct invalid classes: wrong/unknown v2 manifest or Diagram keys/types; invalid/non-blob/nested paths; missing/duplicate/unresolved root or Diagram ID; empty Diagram title; unresolved appearance; missing/duplicate home; duplicate same-Diagram appearance; reference detail link; missing/multiple non-root parent; unreachable Diagram; hierarchy cycle.
- Keep this a concrete contract table, not exhaustive arbitrary-corruption testing or a reusable schema engine.
- Re-run focused v1 loader fixtures proving empty/component-bearing v1 behavior, exact Title/body/Relationship fidelity, unsupported-version handling, and accepted-only authority remain unchanged.
- Verify loads create no ref/object/source/SQLite mutation and ignore unrelated `HEAD` or objects.

### Production handler and concurrency boundary

- Open v1 and v2 through the real handler with real association/SQLite/store paths and verify responses project only the loaded snapshot.
- For v2, verify root/tree/appearance/boundary/navigation data is backend-resolved and snapshot-unified.
- Exercise protected explicit Refresh to an unchanged v2 revision and a different valid non-linear v2 revision, including the existing final-observation race classifications proportionately.
- Create a genuinely new handler/backend/SQLite connection using the same application-data directory and reconstruct the identical v2 accepted SHA and projection.
- Refresh from v1 to valid v2 with an existing v1 pending set and prove the old-base work becomes stale/read-only and discardable without being reinterpreted against v2 or enabling new v2 authoring.
- Send representative direct/stale-client Component and Relationship mutation requests against accepted v2. Verify one product-level read-only failure and zero pending-state, Git, source, SQLite, or loaded-snapshot mutation.
- Race a late old-v1 mutation request with Refresh or project switching to accepted v2 under the existing concrete state lock. Prove eligibility is decided from one synchronized server-owned loaded project/snapshot, the v2 outcome creates no hidden pending work, and no mixed loaded/pending project state is exposed. Do not add concurrency infrastructure or accept a client capability/version field.
- Verify v1 mutation/review/accept behavior remains on its existing path.

### Frontend

Use separate runner-owned cases through the ordinary documented frontend test command.

- v1 retains the implicit map and current Add/Edit/relationship/Changes/review controls without Diagram chrome or a read-only notice.
- v2 starts at root, shows the compact tree/breadcrumbs and single View-only treatment, and exposes no usable mutation controls.
- Known stale/non-current v2 shows the established authority warning instead of the staging notice. An indeterminate Refresh failure shows the established couldn't-check/current-knowledge message without being replaced by View only; already-known stale state remains stale.
- Tree, breadcrumb, detail, back/anchor, Component index/map, canonical-reference, and boundary-reference navigation switch all contextual projections coherently.
- Duplicate Diagram and Component titles receive collision-only context.
- Canonical **Also shown here** and derived **Lives in** presentation are visually and accessibly distinguishable without relying only on color.
- Several crossing edges share one boundary node and remain separately visible/focusable with exact labels and direction.
- Safe Markdown rendering and zero automatic resource access remain exact inside Diagram navigation.
- Clear/deselect, map failure degradation, Refresh state replacement, project switching, and narrow structural layout do not regress.
- A direct v2 mutation error payload is mapped to product language without exposing format/YAML/parser terminology.

Run only `npm test` from `frontend/`. Do not run raw `vitest`, `npx vitest`, a manual test loop, or unattended retries.

### Integrated production-browser evidence

Add one bounded P2.1 runner-owned Playwright scenario or equivalently focused production-browser case. Do not rewrite the historical Gate or Phase 1 scenarios into a generic harness.

The scenario must use the production frontend build served by the real loopback Go process, a real Git executable, real bare Architecture stores, real filesystem state, and real SQLite associations. Because P2.1 intentionally cannot author v2, preparing a canonical accepted-v2 fixture through controlled real Git is valid staging evidence; it must not be presented as a WorkBraid authoring capability.

The scenario should:

- open one valid accepted v2 revision at root;
- verify view-only truthfulness and absence of v2 mutation controls;
- navigate tree, breadcrumbs, detail/back anchor, duplicate titles, canonical reference, boundary reference, documentation, internal/parallel/crossing edges, and map selection;
- verify one absent external Component yields one boundary node with every crossing occurrence;
- externally move accepted to one different valid non-linear v2 revision, prove it remains invisible before explicit Refresh, then verify all projections switch together after Refresh;
- completely stop the first Go process, start a genuinely new process with the same application-data directory, reopen, and verify the identical final accepted SHA and navigation;
- prove source Git/files/checksums and SQLite operational-only state remain exact;
- terminate every child process and remove runtime artifacts on success and failure.

Keep the scenario bounded and fail-fast. Do not add a browser lifecycle loop, permanent machine limit, or production fault-injection control.

### Ordinary checks

Run once from the clean worker tree:

- `git diff --check`;
- `go test ./... -count=1`;
- `go test -race ./... -count=1`;
- `go vet ./...`;
- `go mod verify`;
- `npm test` from `frontend/`;
- `npm run build` from `frontend/`;
- the one documented bounded P2.1 production-browser scenario;
- existing bounded Gate/Phase 1 scenarios only if the implementation changes their shared production path or repository instructions require them.

If a test or child process grows abnormally or fails to terminate, stop it immediately and report the test defect. Do not raise limits, add permanent throttling, start raw Vitest, or rerun blindly.

## Fresh independent reviewer brief

The reviewer receives the exact packet-inclusive worker base, committed worker head, complete diff, this packet, and worker evidence. Review in a fresh isolated worktree:

1. **Authority:** one accepted ref, one version-aware loader, one immutable accepted snapshot, no fallback or alternate v2 authority.
2. **Closed format:** exact normative v2 keys/tree/types and unchanged complete v1 contract; no Phase 3 or speculative keys.
3. **Validation:** root, homes, appearances, hierarchy, reachability, cycles, IDs, blob/path modes, and Relationships validate completely before publication.
4. **Projection:** tree, breadcrumb, index, map, documentation, appearances, boundary references, and topology come from one snapshot; browser does not reconstruct domain semantics.
5. **Boundary fidelity:** one derived boundary per absent external Component, exact directed labelled multiplicity, no identity/persistence, canonical references use ordinary presentation.
6. **Workspace:** root-first tree/breadcrumb/drill/back/home navigation is compact and drafting-table consistent; duplicate-title context is collision-only.
7. **Read-only truthfulness:** only valid current accepted v2 uses the staging treatment; stale/non-current and indeterminate states retain their authority semantics; accepted v2 has no usable mutation chrome; server-synchronized loaded state—not client claims—decides direct mutation eligibility; late requests cannot create hidden work; v1 authoring remains complete and unchanged; no generic capability system exists.
8. **Refresh/restart:** explicit valid non-linear Refresh atomically replaces all projections; new process reconstruction uses the same loader and exact accepted SHA.
9. **Regression:** safe Markdown/resource behavior, Phase 1 map/review, v1 authoring, discard/project switch, origin checks, source isolation, and SQLite boundaries hold.
10. **Scope/QA:** no v2 candidate, migration, Diagram/reference authoring, layout persistence, generic framework, approved-doc edit, committed artifact, manual async loop, or test-resource anomaly.

Run each ordinary check once through its documented command and the bounded P2.1 production-browser scenario once. Report actionable findings and residual human-only risks. Any material finding returns to the same bounded worker and then receives fresh rereview.

## Integration procedure

After a clear independent review:

1. verify the main worktree is clean at the exact packet-inclusive worker base;
2. integrate the exact reviewed implementation commit without rewriting it;
3. verify the integrated head matches the reviewed tree;
4. rerun `git diff --check` and the ordinary checks once;
5. start one fresh real human-checkpoint runtime;
6. stop before P2.2.

If integration differs from the reviewed tree, an approved v1 workflow regresses, or a required check fails, stop and return the exact issue through bounded correction and fresh review.

## Real human checkpoint

Use the built UI served by the real Go process, real Git executable, real bare Architecture repositories, real filesystem, and real SQLite association state.

1. Open a real accepted-v1 project and verify its implicit map plus current Component/Relationship authoring remain available. Make and discard one harmless pending edit through the normal UI to prove the existing path still works without changing accepted.
2. Open a separately associated, controlled real-Git accepted-v2 project. Record its exact accepted SHA. The fixture contains root plus nested/empty detail Diagrams, duplicate titles, home/reference appearances, Component documentation, cycles/parallel Relationships, internal edges, and several crossing occurrences to one external Component.
3. Verify v2 opens at root with one restrained **View only** treatment and no usable Add/Edit/relationship/Diagram/Changes mutation actions.
4. Navigate through the Diagram tree and breadcrumbs into a detail and back. Verify the return anchor is identifiable and map/index/documentation/topology switch together.
5. Select home and **Also shown here** appearances and confirm they open the same canonical documentation. Activate a **Lives in** boundary reference and confirm navigation to the external Component in its home Diagram.
6. Verify one absent external Component is represented by one boundary reference while every directed labelled crossing occurrence and parallel multiplicity remains visible.
7. Verify duplicate Diagram and Component titles remain understandable without general ID/path chrome. Exercise map selection, clear/deselect, Fit, and safe Markdown documentation; confirm no authored image/embed/resource triggers automatic access.
8. Externally advance `refs/heads/accepted` to a different valid non-linear v2 revision. Verify the current workspace does not change before explicit Refresh, then choose Refresh and confirm tree, selected Diagram context, index, map, documentation, appearances, boundaries, relationships, and revision switch together.
9. Exercise opening another project and returning to the v2 project. Confirm no pending work was invented or stranded.
10. Completely stop WorkBraid. Start a genuinely new process with the same application-data directory, reopen the v2 project, and verify the exact final accepted SHA and hierarchy/navigation reconstruct.
11. Verify both source repositories retain exact HEAD, status, files, modes, and checksums. Verify SQLite contains only approved operational source-root/store-ID associations and no Architecture, Diagram, graph, document, boundary, navigation, or layout projection.

The v2 fixture is prepared through bounded technical real-Git inspection because P2.1 is intentionally read-only; do not imply the UI authored it. Human checkpoint completion requires explicit **PASS**.

## Explicit stop and exclusions

Stop after the human explicitly accepts or rejects P2.1. Do not prepare or begin P2.2 from this packet.

Do not introduce:

- v1-to-v2 migration or any conversion-only flow;
- v2 pending changes, candidate construction, review, commit, or accepted-ref advancement;
- Component, Relationship, Diagram, home, detail-link, or reference mutation for accepted v2;
- a generic permission/capability/version-negotiation system;
- Diagram deletion, multiple parents, Diagram DAG/reuse, repeated same-Diagram appearances, or multiple detail Diagrams per anchor;
- canonical/persisted layout, coordinates, sizes, routing, bend points, shapes, annotations, Diagram kinds, renderer state, or appearance identities;
- manual/graphical editing or pending/draft overlays on the normal accepted map;
- persisted/project-scoped pending work, migration state, selection, navigation, or layout;
- another parser, graph authority, candidate builder, review binding, acceptance path, ref, or SQLite projection;
- automatic Refresh, watcher, polling, retry, fallback, repair, initialization, merge/rebase/reconciliation, history, revert, proposal, export, or sync behavior;
- syntax highlighting, rendered/semantic Markdown diff, URL restoration, themed-scrollbar project, or other non-gating polish;
- UML/class semantics, additional Diagram kinds, isometric rendering, Planning, Agent Control, or another vertical.

## Execution result

Status: Complete — human checkpoint **PASS** on 2026-08-21

- Exact Phase 2 documentation baseline: `8fb794137c943942d7761203967722b392f7c0b6`.
- Approved Phase 2 plan: `5ca24c23b799323182e9b3164d2cfbf498e19734`.
- Approved docs-inclusive worker base and P2.1 execution-packet commit: `ea0daec7906f2b2239538f7306fd30556847a48f`.
- Initial implementation: `8fad0a3029881e0995d94b17aa7e270d6ceb57aa`.
- Final integrated implementation after independently reviewed corrections: `db2451e2dc46abe3d34d5dd5fb6e5564ca2580a6`.
- Independent review: the first review removed unused future Diagram serialization fields and required real non-blob Diagram-entry evidence. Rereview found the first non-ordinary fixture was still a blob, so `6a429af16c8e7ca6e66f27021ed4dfee173b4d1d` added an actual tree object at `diagrams/root.yaml`. A subsequent audit found that explicitly empty `detail_diagram` values were not distinguished from omission and that an empty selected Diagram used whole-Architecture empty-state copy; `db2451e2dc46abe3d34d5dd5fb6e5564ca2580a6` corrected both. A fresh acceptance review of the complete worker-base-to-head range reported no actionable findings.
- Automated validation: PASS for `git diff --check`, full uncached Go tests, full race-enabled Go tests, Go vet, module verification, 75 ordinary frontend tests, the production frontend build, the bounded P2.1 production-browser scenario, and the existing Phase 1 and Gate 1 production-browser scenarios. No abnormal resource use or lingering WorkBraid, Playwright, Vitest, npm, Node, or Chromium process was observed. The existing approximately 833 kB production-chunk warning remains non-blocking.

### Human checkpoint evidence

- Persistent checkpoint runtime: `/home/luisc/workbraid/frontend/dist/p21-human-runtime`; accepted-v1 source: `/home/luisc/workbraid/frontend/dist/p21-human-runtime/source-v1`; accepted-v2 source: `/home/luisc/workbraid/frontend/dist/p21-human-runtime/source-v2`. The fixture was technically prepared in the ignored build area because P2.1 intentionally has no v2 authoring path.
- Accepted v1 remained fully editable. The human made and discarded a harmless pending change through the normal UI; no **View only** staging appeared and accepted stayed at exact revision `35219560d9d62ea05f92a5f8f1a6de3911991fff`.
- Accepted v2 opened at its root Diagram with the restrained **View only** treatment and no usable Component, Relationship, Diagram, or Changes mutation controls. The human navigated both detail Diagrams, including the empty Records detail, through the Diagram tree and breadcrumbs.
- Diagram projections: PASS. Home and **Also shown here** appearances opened the same canonical documentation. **Lives in** references navigated to external Components in their home Diagram. Gateway and Records were correctly derived as external to System A's detail. One boundary reference represented each absent external Component while all directed labels and parallel occurrences remained visible.
- Duplicate Diagram and Component titles remained understandable without general ID/path chrome. Map selection, clear/deselect, Fit, and safe inert Markdown behavior passed. The selected empty Diagram correctly said `This diagram has no components.`
- Explicit Refresh: the externally selected valid non-linear v2 revision remained invisible until Refresh. Refresh atomically advanced the tree, selected Diagram context, index, map, documentation, appearances, boundaries, relationships, and revision from `c9c7e334ad9bafb6be64f0548ac47b4c4cd3903a` to `5f16f5998b5739c3427a16ed4bca57175e03647a`; valid-current **View only** staging remained truthful.
- Project switching and return: PASS. No pending work was invented or stranded.
- Restart reconstruction: PASS. WorkBraid stopped completely. A genuinely new process using the same application-data directory reopened exact accepted revision `5f16f5998b5739c3427a16ed4bca57175e03647a` with the identical root/detail hierarchy, appearances, boundaries, relationships, documentation, and navigation reconstructed from canonical Git.
- Source isolation: PASS for both source repositories. Their recorded HEAD, status, files, modes, and checksums remained exact.
- SQLite isolation: PASS. The only table is `source_architecture_associations`, containing the two expected operational association rows and no Architecture, Diagram, graph, document, boundary, navigation, layout, or pending projection.
- Non-gating observation: the green hover treatment can reduce the contrast of the `System A` breadcrumb text. The human explicitly treated this as minor polish that does not block P2.1 or require another review.
- Human checkpoint result: **PASS**.
- Scope: no v1-to-v2 migration, v2 candidate or mutation path, Diagram authoring, canonical presentation state, persisted layout, graphical editing, P2.2 behavior, or another vertical entered this increment.

P2.1 is complete. Stop here; P2.2 remains unstarted.
