# Post-Gate Architecture Phase 2.4 execution packet

Status: Complete — human Phase 2 gate **PASS**

Target: P2.4 — Slug-native projects, reusable references, and final Phase 2 gate

Exact completed-P2.3 prerequisite: `17a5bdb06e31054371a5fa2aebabeaa1af52888f`

Living Architecture baseline: `docs/architecture-v0.md` at `de3394cee8ec5fe277a71761ae884244a682f3ee`

Living UI baseline: `docs/ui-v0.md` at `de3394cee8ec5fe277a71761ae884244a682f3ee`

Roadmap: `docs/architecture-post-gate-roadmap.md` at `de3394cee8ec5fe277a71761ae884244a682f3ee`

Superseding final continuation: `docs/plans/architecture-post-gate-phase-2-final-continuation.md` at `de3394cee8ec5fe277a71761ae884244a682f3ee`

Historical approved continuation: `docs/plans/architecture-post-gate-phase-2-continuation.md`

Completed P2.1, P2.2, and P2.3 records remain historical evidence. Completed Phase 1 records remain the review-authority reference where needed.

Packet-inclusive worker base: the exact commit produced by committing this packet alone on `de3394cee8ec5fe277a71761ae884244a682f3ee`, to be reported at dispatch

Architecture or product changes discovered during implementation require explicit human approval rather than silent changes to this packet or its baselines.

## Execution discipline

P2.4 contains two ordered implementation parts but is one product increment:

1. **Part A — slug-native project model** replaces the source-folder/association/v1 product with the final private-store catalog, native-v2 creation, and slug route.
2. **Part B — reusable Component references** adds candidate-relative show/stop composition and the final Phase 2 gate on that final project model.

Execution proceeds as follows:

1. Commit this packet alone on exact baseline `de3394cee8ec5fe277a71761ae884244a682f3ee` and record that clean packet-inclusive commit as the one worker base.
2. Dispatch exactly one P2.4 implementation worker from that base. The worker may use two conventional implementation commits, Part A then Part B, but not separate workers, branches, integrations, or product checkpoints.
3. Require one fresh independent reviewer over the entire worker-base-to-final-head range. The review reports Part A and Part B as explicit concern groups before assessing their integration.
4. Return bounded corrections to the same worker. The corrected complete range receives fresh independent rereview.
5. Integrate only the exact approved range, run the ordinary checks and bounded browser evidence once, then run one cohesive real Phase 2 human gate.
6. Stop after explicit human **PASS** and a separate completion record. Do not plan or implement Phase 3.

Do not preserve old alpha behavior merely to keep historical tests green. Historical planning and completion records remain unchanged; production code and executable scenarios are living verification and must be updated or removed when their asserted source-folder, SQLite, v1, or setup behavior no longer exists.

## Exact worker brief

Before editing, verify the exact clean packet-inclusive base and read completely:

- `AGENTS.md`;
- the three living baseline/roadmap documents;
- the superseding final continuation and this packet;
- completed P2.3 for the current one pending Architecture authority, nested composition, home movement, review, UI, and accepted reconstruction;
- completed P2.2 for the native-v2 writer and Component/Relationship fidelity, ignoring its now-superseded v1/source-association behaviors;
- completed P2.1 only for still-live v2 loading/navigation/closed-schema behavior;
- completed Phase 1 plan/execution record for immutable candidate-aware review, exact diff, review binding, and CAS behavior.

Implement Part A completely before building Part B on it. Do not retain a hidden compatibility route or parallel source-folder flow.

## Part A — slug-native project model

### Exact canonical schema and alpha break

- Replace the exact closed v2 manifest `project` mapping with `name` plus `slug`; remove `source_hint` completely.
- Validate `project.name` as non-empty after trimming and `project.slug` against `[a-z0-9]+(?:-[a-z0-9]+)*`.
- Preserve immutable canonical `store_id` as project/store identity. Slug is the stable WorkBraid-non-editable human-facing locator, not identity. WorkBraid authors no slug-change operation.
- Every WorkBraid-authored Architecture candidate preserves the exact current project name and slug; P2.4 adds no project-metadata editing path.
- An authoritative external `refs/heads/accepted` update may deliberately supply another otherwise-valid slug. Loading or explicitly Refreshing to that revision adopts the new accepted slug as the current catalog/route locator for the same store UUID. Apply catalog uniqueness/conflict rules normally; do not consult history, bootstrap metadata, SQLite, or another registry.
- Keep `format: workbraid-architecture`, `version: 2`, `root_diagram`, Component/Relationship/Diagram schemas, and all accepted/ref/candidate semantics otherwise exact.
- Support only this exact v2 schema. Delete format-v1 loading, implicit-map projection, setup candidate construction, **Set up diagrams**, v1 review fallback, v1 mutation gates, and all compatibility tests/UI. Do not add v3, migration, conversion, fallback parsing, old-schema adapter, or a legacy command.
- Old alpha private stores and databases are disposable. A store with the old manifest is unavailable/unsupported under the current loader; it is never silently rewritten.

### Slug creation

- Project creation accepts one human-readable name. The backend trims leading/trailing whitespace, rejects an empty result, stores that result as the display name, and derives the slug server-side.
- Derivation lowercases ASCII letters, replaces each maximal run outside `[a-z0-9]` with one hyphen, trims leading/trailing hyphens, and uses `project` when no slug characters remain.
- Resolve a local collision by testing the base slug, then `-2`, `-3`, and so on, selecting the first available valid slug while the catalog/create state is synchronized.
- Project names need not be unique. Catalog slugs must be unique. The browser never supplies an authoritative store UUID, prevalidated slug, or collision result.
- Generate the store UUID and root Diagram UUID exactly once for the successful creation attempt. Initialize one parentless accepted native-v2 bootstrap with `architecture.yaml` and `diagrams/root.yaml` at `100644`, no Components, no Relationships, and an empty root appearance set.
- The root title is initially derived from the project name and remains independently mutable. `diagrams/root.yaml` remains only a creation-time filename convention.

### Private-store catalog authority

- Discover projects directly from the private Architecture area under application data. WorkBraid-created stores remain operationally named `architecture/<store-uuid>.git`; this path convention is not portable identity or normal UI chrome.
- Use the one accepted loader against `refs/heads/accepted` to obtain each valid store's canonical UUID, display name, slug, revision, and root. Do not add a catalog database, index file, registry, cache authority, alternate parser, fallback ref, or working-tree projection.
- Validate that a UUID-named store's canonical manifest UUID matches its operational directory identity as already required by the manager.
- Build one deterministic catalog response independent of filesystem enumeration order. Project display ordering is a UI concern, but slug resolution and conflicts are not.
- When two valid accepted stores expose the same slug, return an explicit catalog-conflict state for that slug and open neither. Do not choose first/last or mutate either store.
- Surface a discovered UUID-named store with missing, malformed, or unsupported accepted state as an unavailable catalog entry with bounded technical store identity where practical. Do not partially parse it for project authority, silently omit it as though it never existed, repair it, or add recovery machinery.
- An unknown slug returns a normal project-not-found result. It never creates a project, initializes a store, guesses by name, or falls back to another slug/ref.
- Resolve project-open, reload, mutation, Refresh, review, confirmation, discard, and switching from synchronized server-owned catalog/loaded state. The route slug locates a store; the canonical manifest UUID remains authority after resolution.
- After explicit Refresh adopts a different accepted slug for the currently open store, replace the browser route with `/projects/<new-slug>`. The old route becomes not found unless another valid store owns it. This is accepted-state publication, not a slug-edit lifecycle.

### URL and project workspace

- Replace the source-folder opening sheet with one project-catalog entry state containing discoverable projects and one **New project** action asking only for a name.
- Opening a catalog entry navigates to `/projects/<slug>`. Direct navigation and browser reload at that route open the same exact store/revision through the backend catalog.
- Implement only the bounded browser-history/path handling needed for the catalog and project route. Do not add a generic navigation framework or routing dependency.
- Unknown routes show a normal not-found state with a route back to the catalog. Catalog conflict and unavailable-store states are distinct, concise, and non-technical by default, with bounded technical details when useful.
- The workspace keeps the project display name primary. Show slug only where route/disambiguation context matters. Store UUIDs and private paths remain technical details rather than list chrome.
- **Open another project** returns to the catalog while preserving the existing backend pending-change/project-switch guard and browser-local dirty-editor guard. Do not invent persisted pending work to allow leaving.
- Project names may collide; only then use minimal slug context to disambiguate catalog rows.
- Keep the loaded project, pending set, review binding, Refresh, switch, discard, and acceptance transitions under the existing concrete synchronization boundary. A late request for a formerly loaded slug cannot mutate whichever project is current now.

### Delete the obsolete source/SQLite product

- Delete source-root path normalization/inspection, source-folder open/initialize request fields and UI, folder-to-store association behavior, source isolation checks, source hints, reassociation language, and source-project gate fixtures/checksums.
- Delete the association package/table and database initialization. SQLite has no remaining demonstrated production use, so remove it from process startup, handler dependencies, Go module dependencies, tests, and runtime artifacts entirely.
- Do not replace SQLite with another catalog persistence layer. Do not retain an empty database interface or placeholder for possible future pending persistence.
- Do not replace the source folder with another arbitrary filesystem root. The application-data directory and its private UUID-named Git stores are the only current storage locations.
- Preserve direct fixed-argument Git execution, bare-store layout, accepted-ref authority, safe candidate construction, stale/CAS behavior, and no durable project content outside WorkBraid application data.

## Part B — reusable Component references

### One pending Architecture authority

- Extend P2.3's one backend-held pending Architecture change set. Retain one exact accepted base, one generation, one `ConstructCandidate`, one complete candidate tree, one v2 loader/validator, one immutable review candidate, one exact diff, one `(base commit, candidate tree, pending generation)` binding, and one confirmation/stale/CAS/publication path.
- Add one small concrete typed net reference state for a `(Diagram ID, Component ID)` pair: deliberate reference presence or deliberate absence. Do not use fake Component/Relationship changes, sentinel IDs, appearance IDs, Diagram drafts, a generic operation algebra, event log, membership framework, or second candidate builder.
- Resolve reference choices and mutations against the complete current pending candidate under the existing backend synchronization boundary. The browser may submit selected stable IDs from backend choices, but never decides home, externality, absence, target existence, or eligibility.
- Successful mutation increments the one generation and invalidates its review. Rejected/stale/ineligible mutation changes no pending fact, generation, review binding, accepted Git, or loaded snapshot.

### Show component here

- Provide a structured **Show component here** action for an active Diagram, including candidate-relative Diagram composition under **Changes in progress**.
- The action works in accepted and candidate-only Diagrams and may target accepted Components or pending-new Components whose one candidate home is elsewhere.
- Present Component title first with only collision/pending/home context needed for understanding. Do not expose UUID, YAML, `role`, appearance mechanics, or private paths as normal controls.
- The Component must resolve in the complete candidate, have exactly one home in another Diagram, and have no home/reference appearance in the target Diagram.
- Add exactly one canonical `role: reference` entry at the natural authoring position. Copy no Component data, change no Component identity/documentation/Relationship, and add no detail link.
- Reject duplicate or home-plus-reference attempts from current server-owned candidate state. The UI omits known-ineligible choices, while backend rejection protects direct/stale requests.
- If the Component was visible only as one derived **Lives in** boundary, the canonical reference replaces that presentation with one ordinary node. Every exact crossing Relationship occurrence retains direction, label, multiplicity, and source ownership; no Relationship fact changes and no boundary identity is persisted.

### Stop showing here

- Offer **Stop showing here** only for a canonical reference appearance, never for a home or derived boundary.
- Remove only that one reference entry. Never move/remove the home, change Component documentation or Relationships, detach hierarchy, or operate on the child/detail link.
- If crossing Relationships still require the absent Component, derive one identity-free **Lives in** boundary and attach all exact occurrences. With no crossing Relationship, the Component simply disappears from that Diagram.
- Persist no hidden boundary or removed-reference object. Keep the contextual action distinct from **Edit component**, **Open _diagram_**, and **Change where it lives**.

### Net home/reference normalization

Candidate reconstruction starts from the exact accepted base and deterministically produces final composition, not request history.

- Moving home into a Diagram makes that pair `home` and suppresses/subsumes any reference state there.
- Moving home out of a Diagram establishes that former-home pair as absent unless a deliberate surviving **Show component here** intent says otherwise.
- An accepted/base reference must not resurrect after temporarily becoming home and then ceasing to be home in the same pending set.
- Required bounded sequence:

  ```text
  accepted: A = home, B = reference
  pending:  move A -> B; move B -> C
  result:   A = absent, B = absent, C = home
  ```

- A later explicit **Show component here** in B replaces the net absence and yields A absent, B reference, C home.
- A concrete net reference-absence value for the Diagram/Component pair is permitted. Do not add timestamps, command ordering, dependency sorting, operation replay, generic conflict resolution, or appearance identity.

### Fidelity and validation

- Untouched Diagram and Component paths reuse exact Git tree entries/blobs/modes.
- A rewritten Diagram preserves stable ID, path, regular-file mode, title, unchanged semantic fields, detail links, and relative source order of surviving appearances. New references use natural insertion position. Appearance order remains non-semantic.
- Reference-only changes preserve every Component blob and every Relationship declaration/fact exactly. Do not add lossless-YAML editing; lexical YAML formatting within a rewritten Diagram is not guaranteed.
- Review remains unavailable until the complete candidate validates. Cover concise localized product guidance for duplicate appearance, home in target, disappeared Component/Diagram, stopping a non-reference, illegal reference detail link, and conflicting/missing-home composition.
- Failed validation retains the whole pending set. Reuse the concrete Changes/Fix/location patterns; do not build a generic validator or show schema/parser terminology.

### Candidate-aware review and projections

- Extend the one existing Review changes workspace and its immutable bound snapshots. Match reference composition by stable Diagram ID plus Component ID; do not add appearance IDs or another review object.
- Show a Component becoming shown/no longer shown in a Diagram and the corresponding ordinary-node/boundary/absent presentation.
- Reference add/remove is Diagram composition only. It is not Component addition/removal, Component content change, or Relationship fact addition/removal.
- Preserve Relationship comparison exclusively as the exact `(source Component ID, target Component ID, exact label)` multiset including multiplicity. Real Relationship edits in the same candidate appear independently.
- Boundary-to-ordinary or ordinary-to-boundary solely from composition is presentation change, not semantic Relationship delta.
- Preserve snapshot unity across Diagram tree, selected Diagram, breadcrumbs, index, map, documentation, appearances, boundaries, Relationships, titles, and revision. Candidate-only Diagram/reference state never leaks into **Before changes**; keep the nearest-base-ancestor/root fallback.
- Selecting a reference composition change focuses its Diagram context and exact Diagram-file diff. Complete unified diff remains authoritative. A clear visual-map failure must retain review and acceptance through the validated candidate/diff.
- Normal workspace remains accepted-only before CAS. Pending references are reachable through **Changes in progress** and review, never a pending overlay on the normal map.

### Existing workspace and authority continuity

- Preserve P2.3's grouped Diagram composition, action hierarchy, retained review context, review/Changes bottom dock, captions, non-overlapping layout, neutral clear selection, candidate invalidity visibility, dirty-editor navigation guard, and deterministic stable-ID layout.
- Preserve the existing accepted-ref pre-CAS observation, successor creation, mandatory CAS, CAS-success boundary, response-loss classification, publication/recovery, explicit Refresh, stale pending semantics, whole-set discard, and restart reconstruction.
- Project creation/open/reload/switch, pending mutation, reference mutation, home movement, review capture, discard, Refresh, confirmation, CAS, and publication cannot interleave into mixed project/snapshot/pending state.

## Worker acceptance criteria

The implementation is ready for review only when:

1. The exact manifest is closed v2 with `project.name` and valid stable `project.slug`; WorkBraid authors no slug changes, while an authoritative external accepted replacement is adopted without changing store UUID. `source_hint` and v1 are unsupported with no adapter or migration.
2. Fresh creation by name generates one UUID, first-free valid slug, parentless native-v2 root, and valid catalog entry; duplicate names remain allowed.
3. Catalog/open/reload derive from exact accepted private stores; `/projects/<slug>` restores the same store/revision, unknown slugs do not create, duplicate slugs conflict deterministically, and malformed stores are not silently mistaken for valid/missing projects.
4. Source-folder APIs/UI/state, associations, SQLite startup/dependency, source isolation, and v1/setup production behavior are gone rather than hidden.
5. One concrete net reference state joins the existing pending Architecture authority without a second builder/review/acceptance path or generic framework.
6. Show/stop resolve accepted, pending-new, accepted-Diagram, and candidate-only-Diagram state from the complete candidate and preserve one home/one appearance-per-Diagram/no-reference-detail invariants.
7. `A home + B accepted reference -> home B -> home C` leaves B absent; explicit later Show B creates exactly one B reference while C remains sole home.
8. Reference add/remove preserves Component and Relationship facts, surviving appearance order, untouched blobs/modes/paths, and exact crossing edge direction/label/multiplicity.
9. Review presents reference composition and boundary changes without false Component/Relationship deltas, retains snapshot unity/fallback/exact diff, and invalidates with later mutation.
10. Normal accepted surfaces remain unchanged before CAS and publish together after CAS; Refresh, stale handling, project switching, discard, safe Markdown, restart, and private-Git authority pass.
11. No obsolete database/artifact, Phase 3 state, migration compatibility, source-project model, generic catalog/membership framework, or another vertical enters the result.

## Required automated evidence

Use real Git, real private bare stores, real filesystem state, the production Go handlers, and the built browser application. Do not use fake Git/catalog authority or preserve removed v1/source fixtures as alternate scenarios.

### Part A evidence

- Exact manifest parse/write rejection and acceptance for `name + slug`, including grammar boundaries and closed-schema rejection of `source_hint`/v1.
- Deterministic slug derivation for punctuation/spacing/case, empty-derived fallback, duplicate names, and `-2`/`-3` collision selection.
- Real concurrent project creation/catalog refresh bounded under the existing concrete synchronization so duplicate slugs cannot both publish or select by enumeration order.
- Fresh name-based creation: parentless commit, exact tree/modes, store/root UUIDs, accepted ref, catalog listing, direct slug route, browser reload, and fresh-process reopen.
- Multiple same-name projects disambiguated by slug; unknown slug not found without writes; duplicate-slug fixtures produce conflict; malformed/missing-accepted store is surfaced unavailable without fallback/repair.
- A bounded real accepted-ref case changes one valid store's slug externally without changing its store UUID. Explicit Refresh adopts it, catalog resolution moves to the new slug, the old slug becomes not found unless reused, and uniqueness/conflict behavior remains deterministic.
- Pending/dirty-editor guards while returning to catalog/opening another project; late old-project mutations rejected from server-owned current state.
- Removal of association schema, SQLite process initialization/dependency, source-root payloads/state, source isolation, v1 loading/setup, and their obsolete executable scenarios. Assert no runtime `.db` artifact is created.
- Update still-valid Gate/Phase 1/P2.1/P2.2/P2.3 executable scenarios proportionately to slug-native native-v2 setup while preserving accepted-ref, pending/review/CAS, safe Markdown, Refresh, and restart coverage.

### Part B architecture/handler evidence

- Add references for accepted and pending-new Components in accepted and candidate-only Diagrams; reject duplicate/home/missing/stale targets atomically.
- Show boundary-to-canonical conversion with several parallel/differently labelled crossing Relationships and prove exact multiset preservation; stop to restore one boundary; stop an unconnected reference to produce absence.
- Prove exact untouched Diagram/Component entry reuse, rewritten Diagram semantics/mode/path, natural insertion, and surviving appearance order.
- Exercise reference-to-home conversion, move-away plus explicit show-back, and the mandatory accepted-reference non-resurrection sequence through deterministic reconstruction from the accepted base.
- Construct one mixed candidate containing Component content, real Relationship edit, pending-new Component/home, Diagram title/detail/home changes, and reference add/remove facts.
- Cover invalid candidate retention/correction, generation/review invalidation, stale authority, pre-CAS observation, CAS success/race, response loss/publication recovery, explicit Refresh, and fresh-process reconstruction.
- Race reference mutation proportionately with home movement, discard, project switch, review capture, and confirmation. Each outcome is one coherent project/base/generation/binding.

### Frontend and production-browser evidence

- Separate runner-owned frontend cases prove catalog/create/not-found/conflict/unavailable states, slug routes/reload, name-collision disambiguation, pending/dirty guards, and absence of folder/setup UI. If inexpensive, one bounded route case proves that Refresh adopting an external accepted slug replaces the open browser URL rather than retaining the stale slug.
- Separate cases prove title-first candidate-relative Show choices, Stop only on canonical references, direct error mapping, grouped Changes editing, boundary/canonical/absent rendering, review markers/focus/fallback, no false deltas, and stable layout.
- Use the ordinary repository frontend command. Never put repeated asynchronous render/unmount/mock-restoration lifecycles in a manual loop.
- Add one bounded combined P2.4/Phase 2 production-browser scenario beginning with fresh app data and project creation by name, then performing the final gate's two accepted candidates, slug reload, Refresh, full stop, and fresh-process reconstruction.
- Every scenario owns cleanup in `finally`, uses bounded timeouts, terminates process groups, and leaves no WorkBraid, Playwright, Chromium, npm, Node, or Vitest process.

## Ordinary checks and resource safety

After focused development checks are green, run once from the clean worker tree:

- `git diff --check`;
- `go test ./... -count=1`;
- `go test -race ./... -count=1`;
- `go vet ./...`;
- `go mod verify`;
- the repository's ordinary frontend `npm test` command;
- `npm run build`;
- the bounded combined P2.4/Phase 2 browser scenario;
- only living earlier scenarios whose still-supported production paths were materially changed.

Never invoke raw `vitest`, `npx vitest`, alternate unbounded frontend commands, or hand-written repeated render/unmount loops. Run browser scenarios with one runner-owned worker. Stop immediately on abnormal memory growth or orphaned test/browser processes; diagnose before retrying. Do not weaken product defaults or add permanent machine limits to mask a test defect.

Record commands, results, tested SHA, initial failures/corrections, and non-blocking warnings. Do not commit dependencies, build output, databases, logs, screenshots, traces, temporary stores, or binaries.

## Fresh independent reviewer brief

The reviewer receives the exact packet-inclusive base, complete worker range, living baselines, superseding continuation, this packet, and relevant completed records. Review the code and evidence, not only summaries.

### Concern group A — slug-native projects

Review specifically:

1. exact closed schema, slug grammar, WorkBraid-authored stability, UUID identity, authoritative external accepted-slug replacement semantics, native-v2 bootstrap, and no hidden v1/source-hint compatibility;
2. server-owned slug derivation/collision handling and atomic creation without name uniqueness;
3. catalog authority exclusively from exact accepted private stores, UUID directory validation, deterministic duplicate-slug conflict, unavailable versus missing behavior, and no alternate parser/ref/index/database authority;
4. `/projects/<slug>` direct/reload/fresh-process behavior, not-found no-write behavior, and backend ownership after route resolution;
5. complete removal of folder/source-root/association/reassociation/source-isolation/v1/setup/SQLite code, dependencies, UI, and obsolete living tests;
6. preservation of pending/dirty switch guards, Refresh, accepted-ref/CAS authority, and no durable project content outside application data;
7. absence of a generic catalog/repository/database framework or migration machinery.

### Concern group B — reusable references

Review specifically:

1. one pending/base/generation/candidate/loader/review/CAS authority and a minimal net reference presence/absence representation;
2. candidate-relative accepted/pending-new and accepted/candidate-only Diagram eligibility under synchronized backend state;
3. exact home/reference normalization, especially accepted B reference non-resurrection across A-to-B-to-C moves and explicit later show-back;
4. one home, at most one appearance per Diagram, no reference detail link, no appearance ID, and unchanged strict hierarchy;
5. untouched blob/mode/path fidelity, surviving appearance order, and no lossless-YAML subsystem;
6. exact boundary/canonical/absent projection and crossing Relationship direction/label/multiplicity with no false semantic delta;
7. localized invalidity, atomic rejected requests, binding invalidation, snapshot-unified review, candidate-only fallback, and exact diff;
8. accepted-only normal workspace, CAS publication, Refresh/stale behavior, and fresh-process Git reconstruction;
9. no Phase 3 state, Diagram lifecycle, generic graph/membership/reference framework, or other vertical.

### Combined scrutiny

Confirm Part B genuinely runs on the slug-native Part A product and no source-root/store association remains hidden in reference handlers, test fixtures, browser state, project switching, Refresh, or final reconstruction. Confirm the worker range is rooted at the exact packet base and any two commits remain one reviewed/integrated product increment.

Every material correction returns to the same worker and the corrected full range receives fresh independent rereview.

## Integration procedure

1. Verify the main worktree is clean and at the exact packet-inclusive worker base.
2. Verify the worker range is rooted there and contains only the reviewed Part A/Part B implementation and bounded corrections.
3. Integrate the exact reviewed range without rebuilding it by hand.
4. Confirm approved living docs and completed historical records are untouched by implementation.
5. Run the ordinary checks and bounded browser evidence once from the clean integrated tree.
6. Stop all test/runtime/browser processes and verify no WorkBraid, npm, Node, Vitest, Playwright, or Chromium process remains.
7. Prepare fresh application data with no projects for the human gate; do not pre-seed the primary Architecture workflow.
8. A gate defect within approved semantics returns to the same worker plus fresh rereview. A missing product/domain rule stops for human decision.

## Final Phase 2 human gate

Use the built UI served by the real loopback Go process, real private Git stores, and a fresh application-data directory containing no projects. No source repository or SQLite fixture participates.

1. Open WorkBraid and verify the empty project catalog offers deliberate name-based creation with no folder/path/link/setup language.
2. Create a project using a representative human-readable name. Record its generated valid slug, `/projects/<slug>` URL, immutable store UUID in bounded technical inspection, exact parentless accepted commit, root Diagram UUID, root title, empty appearances, canonical file modes, and sole `refs/heads/accepted` authority.
3. Reload `/projects/<slug>` and verify the same store/revision opens. Return to the catalog, verify the project is listed, and reopen it. Create another project with the same display name if proportionate and verify unique suffixed slug plus unambiguous catalog selection.
4. Through structured UI create meaningfully connected Components and Relationships, including a pending-new target and parallel/differently labelled crossing facts.
5. Create/title nested detail Diagrams, move homes into the hierarchy, and create one Component while a detail Diagram is active.
6. Establish a Component visible only as a derived **Lives in** boundary, choose **Show component here**, and verify one ordinary canonical node replaces the boundary while all exact crossing Relationship occurrences remain.
7. Show another external Component in a Diagram where no crossing Relationship required it, proving deliberate contextual reuse independently of Relationships.
8. In the same pending set combine Component content, a real Relationship edit, Diagram title/home work, a pending-new Component/home, and reference composition involving candidate-only state. Verify normal accepted tree/map/index/docs/composition remains exact and pending work stays under **Changes in progress**.
9. Review and record exact base commit, candidate tree, and generation. Verify **With changes**/**Before changes** snapshot unity, candidate-only fallback, reference additions, boundary presentation changes, independent real Relationship deltas, contextual diff focus, and complete exact canonical diff.
10. Deliberately accept once. Record successor/parent/tree and verify Diagram tree, map, index, documentation, hierarchy, homes, references, boundaries, Relationships, and revision publish together.
11. Navigate accepted root/details/references/boundaries. Verify Component edit, detail navigation, home movement, Show, and Stop actions remain distinct.
12. Begin a second candidate. Exercise the approved normalization from accepted A home/B reference through home B then home C; verify B is absent. Explicitly Show B and verify exactly one B reference with C sole home.
13. Remove one reference whose crossing Relationships require a boundary and another unconnected reference. Verify boundary return with exact occurrences versus complete disappearance.
14. Review the second candidate and verify home/reference composition without false Component/Relationship deltas plus exact Diagram diff. Deliberately accept and record successor/parent/tree.
15. Exercise one controlled valid external accepted-v2 advancement which preserves store UUID and changes the manifest to another valid non-conflicting slug. Explicitly Refresh, verify all accepted projections switch together, verify the browser route becomes `/projects/<new-slug>`, and verify the old route is now not found. Confirm the catalog resolves the new slug to the same project/store identity.
16. Stop WorkBraid completely. Start a genuinely fresh process with the same application data and open `/projects/<slug>`. Reconstruct exact final accepted SHA/tree, project name/slug/store UUID, hierarchy, Diagram identities/titles, homes, references, Component docs, Relationships, boundaries, and navigation solely from private Git.
17. Inspect application data and process dependencies: no source-root association, SQLite database/catalog, v1/setup state, or non-Git Architecture projection exists.
18. Record explicit human **PASS** or **FAIL**. Phase 2 completes only on **PASS**.

The gate is stop-the-line. Do not bypass a broken product path with manual canonical edits, weaken evidence, or treat green tests as completion.

## Explicit exclusions

P2.4 must not introduce:

- source folders, project filesystem roots, associations, reassociation, source hints, or source-repository inference/export;
- SQLite, another catalog/index/registry database, generic catalog framework, migration, v1 support, setup transition, v3, compatibility adapter, or downgrade;
- project deletion/import/recovery, project-name editing, slug editing, or unavailable-store repair;
- Diagram deletion/lifecycle, multiple parents/DAGs, reusable child Diagrams, multiple appearances of one Component in a Diagram, or multiple detail Diagrams per home;
- Component deletion, Relationship IDs/lifecycle, appearance IDs/lifecycle, or ordering semantics;
- persisted/multiple pending sets, partial discard, undo/redo, merge/rebase/reconciliation;
- persisted/manual coordinates, layout, sizes, routes, bend points, shapes, annotations, renderer state, or graphical authoring;
- a generic Diagram/graph/membership/reference/hierarchy/validation/workflow/command/event/transaction/capability framework;
- another parser, candidate builder, review object, acceptance path, accepted ref, or browser-owned Architecture interpretation;
- pending overlays on the normal accepted map, semantic/rendered Markdown diff, syntax highlighting, dedicated scrollbar work, or unrelated polish;
- Diagram kinds, UML, isometric/3D, Planning, Agent Control, another vertical, authentication, remote access, multi-user behavior, or mobile-specific UX.

P2.4 stops after fresh independent review, exact integration, ordinary checks, the restart-backed Phase 2 gate, a separate completion record, and explicit human **PASS**. Phase 3 remains unplanned and unstarted.

## Execution result

Status: Complete — human Phase 2 gate **PASS** on 2026-09-02

- Exact completed-P2.3 prerequisite: `17a5bdb06e31054371a5fa2aebabeaa1af52888f`.
- Alpha-stage Phase 2 baseline/final-continuation commit: `de3394cee8ec5fe277a71761ae884244a682f3ee`.
- Approved packet-inclusive worker base: `22940454f163338b9b17bd173f588b50904df47e`.
- Initial combined Part A/Part B implementation and authority corrections reached `e34ab0a9b55c5aa7fb21058d5ac7dc49dce8bec6`.
- Final integrated implementation after human-gate catalog, review-label, home-move, and living-scenario corrections: `39ca3f7035dc93f63dd32078506b6752ee271ec5`.
- The implementation replaced source-folder association with the private-store-derived slug catalog, direct native-v2 bootstrap, and `/projects/<slug>` restoration. It removed source-root selection/reassociation, `source_hint`, v1/setup behavior, SQLite, and their running dependencies rather than retaining an alpha compatibility path.
- Reusable Component appearances extend the same one pending Architecture base/generation, one complete-candidate builder/validator, one immutable review binding, exact diff, confirmation, stale/CAS, and publication authority. No appearance identity, Diagram draft, second candidate/review path, command log, generic catalog, or graph/membership framework was introduced.
- The final home-move correction consciously supersedes the P2.3 interaction choice that allowed an anchor to target deeper descendants and rely on later correction. Server-owned eligibility now excludes the current home and the complete subtree owned by the Component's detail link. The backend rechecks that exact candidate-relative choice under the existing lock, constructs one complete final proposal on isolated pending state, and publishes it only when valid. Rejected choices preserve the live pending candidate, generation, review binding, and net reference state exactly.
- The approved A-home/B-reference → home B → home C normalization remains exact: B does not resurrect, while a later explicit **Show component here** deliberately restores one B reference. Valid coupled multi-move/reparenting candidates remain supported because only the complete final proposal is validated.

### Independent review and automated evidence

- Exactly one implementation worker produced the full range `22940454f163338b9b17bd173f588b50904df47e..39ca3f7035dc93f63dd32078506b6752ee271ec5`. Every material correction returned to that same worker.
- One fresh independent reviewer assessed Part A and Part B separately and the combined range. Review caught and rejected catalog/Refresh authority races and an intermediate-candidate home-move implementation; bounded corrections received fresh rereview. Final verdict: **PASS** with no findings for the complete range through `39ca3f7035dc93f63dd32078506b6752ee271ec5`.
- Reviewer scrutiny confirmed store-UUID action binding, deterministic catalog conflicts, authoritative external-slug Refresh behavior, exact route replacement, one reference/candidate/CAS authority, candidate-relative references, non-resurrection, Diagram/source fidelity, no false Relationship deltas, snapshot-unified review, valid final multi-move composition, subtree-pruned move eligibility, rejected-request isolation, and collision-only authoring context.
- Final clean-tree checks: **PASS** for `git diff --check`, `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go mod verify`, 34 ordinary frontend tests, and the production frontend build. The existing production chunk-size warning remains non-blocking.
- The bounded `npm run test:p2.4` production-browser scenario passed 1/1 in 4.6 seconds. Its one stale accessible-name locator was updated to the new exact **Change where _Component_ lives** action without changing topology, order, timeout, harness, or assertions; that one-line living-scenario correction received separate fresh review.
- Tests ran through ordinary documented commands with one browser worker. No raw Vitest invocation, manual render/unmount loop, abnormal memory growth, or swap use occurred. All gate/test WorkBraid, npm, Node, Vitest, Playwright, and Chromium processes were stopped before completion.

### Final Phase 2 human-gate evidence

- Fresh runtime root: `/tmp/workbraid-p24-human.dV7DBE`; application-data directory: `/tmp/workbraid-p24-human.dV7DBE/app-data`. The built browser UI was served by the real loopback Go process over real private Git stores. No source repository or SQLite fixture participated.
- From an empty catalog, the human created `Phase Two Gate` through the name-only UI. Canonical store UUID: `143d1aac-44c9-4663-94e6-ebb938dac66b`; initial slug/route: `phase-two-gate` / `/projects/phase-two-gate`; parentless bootstrap: `9463f7d8f5da0e0ff542e227ddcf3c34618fbfef`; bootstrap tree: `66ac88cd0f2664eef0d6289a8b2cdae64f8a7bd8`; root Diagram UUID: `962738f1-4285-47b7-b317-81dbe9d34278`. The root title was independently initialized to `Phase Two Gate`, appearances were empty, and the only canonical files were mode `100644`.
- Route reload and catalog reopening resolved the same exact store/revision. Creating the same display name again produced distinct slug `phase-two-gate-2`, store UUID `3f8b0452-61a2-42f4-919f-5633ca3353fc`, and parentless bootstrap `2bfc9b4df1b33c604d9be16af75286051c3e8e73`; catalog selection remained unambiguous.
- One coherent first candidate created Gateway, Worker, Records, Notes, and Archive; two differently labelled parallel Gateway→Worker facts plus Worker→Records; nested `Runtime` and `Storage` Diagrams; candidate-relative homes; a boundary→canonical Worker reference; and an unconnected Notes reference. The accepted workspace remained the empty bootstrap until review.
- First human review binding: base `9463f7d8f5da0e0ff542e227ddcf3c34618fbfef`, candidate tree `c3fdf802ece90005e28f73b7deeeae7e11662805`, generation `13`. **With changes** / **Before changes**, candidate-only fallback, Diagram/reference composition, real Relationship deltas, exact Diagram-labelled review entries, contextual focus, and the complete unified diff passed. Deliberate acceptance produced revision `8592b4f7e8555b2980186a39bc69ebb8952d2251`, parent `9463f7d8f5da0e0ff542e227ddcf3c34618fbfef`, and tree `c3fdf802ece90005e28f73b7deeeae7e11662805`.
- The gate exposed a real home-move defect: a deeper descendant was offered, stored as an invalid move, produced an empty Fix destination list, and then blocked an unrelated move. Feature work stopped. The descendant-permitted executable expectation and invalid-home repair cache/UI were removed; candidate-relative subtree pruning and validate-before-publication replaced them. The corrected UI identifies the Component and current Diagram, hides the action when no destination exists, and shows only backend-approved choices.
- On the corrected build, Worker moved Runtime → root → Storage in one pending set. The accepted root reference did not resurrect after the second move. A later explicit Show restored exactly one root reference; stopping it returned the exact two Gateway→Worker occurrences to one **Lives in Storage** boundary. Removing the unconnected Notes reference made Notes disappear from Storage while its root home remained exact.
- Second human review binding: base `8592b4f7e8555b2980186a39bc69ebb8952d2251`, candidate tree `a29facced0ee7c2af1f5aee6ac40bf071567cc8c`, generation `5`. Composition changes used exact Diagram names and produced no false Component-content or Relationship-fact deltas. Deliberate acceptance produced revision `30a449b81dfb7474ecde424d08842fa080d8693f`, parent `8592b4f7e8555b2980186a39bc69ebb8952d2251`, and tree `a29facced0ee7c2af1f5aee6ac40bf071567cc8c`.
- A controlled external authoritative successor changed only the manifest slug to `phase-two-gate-refreshed`: revision `e8829e2dd7e302888c34009891db77107f3090b7`, parent `30a449b81dfb7474ecde424d08842fa080d8693f`, tree `7c651855ae1ac30f99dbfd1896b8a3549a53d93b`. It remained invisible until explicit **Refresh**; Refresh atomically adopted all projections and replaced the browser route with `/projects/phase-two-gate-refreshed`. The old route became normal not-found, while the catalog resolved the new slug to the same store UUID.
- After a complete process stop, a genuinely fresh process reopened `/projects/phase-two-gate-refreshed` and reconstructed exact revision `e8829e2dd7e302888c34009891db77107f3090b7`, root → Runtime → Storage hierarchy, Diagram/Component identities, documentation, homes, Relationships, boundaries, navigation, and catalog solely from canonical Git. Worker remained solely home in Storage; Notes remained solely home in root; root derived one Worker boundary carrying both exact Gateway relationships.
- Final private-store inspection found sole authority `refs/heads/accepted`, only UUID-named private Git stores, canonical file modes `100644`, and no corrupt Git objects. Unreachable candidate trees remained ordinary non-canonical Git objects. No SQLite/database file, source-root association, v1/setup state, external catalog registry, or non-Git Architecture projection existed.
- Human-requested catalog alignment, creation-first ordering, exact Diagram names in composition review, and Component/current-home move wording were bounded implementation corrections, covered proportionately, and independently rereviewed. Search/filter scaling and the conceptual wording/integration of explicit reusable appearances remain non-gating future product candidates.
- Human gate result: **PASS**.

Phase 2 is complete. Stop here; Phase 3 remains unplanned and unstarted.
