# Post-Gate Architecture Phase 2 plan

Status: Approved

Architecture baseline: `docs/architecture-v0.md`

UI baseline: `docs/ui-v0.md`

Roadmap: `docs/architecture-post-gate-roadmap.md`

Exact planning baseline: `8fb794137c943942d7761203967722b392f7c0b6`

Target: First-class format-v2 Architecture Diagrams and nested navigation

Architecture or product changes discovered during implementation require explicit human approval rather than silent changes to this plan or its baselines.

## Phase objective

WorkBraid loads, navigates, authors, reviews, accepts, refreshes, and reconstructs the approved first-class Diagram model without replacing Components or Relationships with generic graph objects.

The completed product supports one strict rooted Diagram tree, one home appearance per Component, reusable reference appearances in other Diagrams, home-owned detail links, and derived cross-Diagram boundary references. Valid v1 remains fully supported. The first Diagram action against v1 creates one pending candidate containing both deliberate v1-to-v2 migration and the requested Diagram operation; migration is never a separate accepted revision or standalone user ceremony.

Phase 2 retains the existing private-Git authority, immutable snapshot loader, one backend-held pending change set, complete candidate construction, candidate-aware review binding, exact canonical diff, confirmation, compare-and-swap acceptance, explicit Refresh, and restart reconstruction paths.

## Increment shape

Use three sequential increments:

1. **P2.1 — Load and navigate accepted v2 Diagrams.** Extend the sole accepted-snapshot loader and durable workspace to understand an already-canonical v2 revision. This is independently useful read-only behavior and proves the portable contract before authoring it.
2. **P2.2 — Create nested Diagrams through deliberate migration and home composition.** Add the first complete authoring loop: the first Diagram action against v1 performs migration plus that requested operation in one pending candidate, then uses the existing review/CAS path. Include Diagram titles, detail creation, home movement, and new-Component home assignment.
3. **P2.3 — Author reusable references and complete the Phase 2 gate.** Add structured reference-appearance authoring and prove the complete home/reference/boundary model as one coherent product workflow.

These boundaries are product boundaries, not parser/UI/backend tickets. P2.1 leaves a real accepted-v2 reader and navigator. P2.2 leaves a complete human path from v1 or v2 through nested Diagram acceptance. P2.3 adds the independently useful reusable-appearance behavior and closes the approved Phase 2 model.

Do not split migration from the requested first Diagram operation. Do not expose a conversion-only action, intermediate migrated revision, migration wizard, or setup state which must be accepted before the requested Diagram can exist.

Each increment receives its own proposed execution packet only after the preceding increment is integrated and human-approved. Each packet pins one exact docs-inclusive worker base, uses one implementation worker, receives fresh independent review, and stops for a real human checkpoint before the next increment.

## Shared invariants across Phase 2

- `refs/heads/accepted` remains the sole accepted authority.
- The existing version-aware accepted-snapshot loader is the only canonical v1/v2 interpretation path. Candidate validation, review projections, acceptance publication, Refresh, and restart reuse it.
- Components retain their stable IDs, canonical Markdown/documentation, filenames, modes, and global source-owned Relationships.
- Diagrams are composition over Components and Relationships. No generic canonical node/edge model is introduced.
- Every valid v2 revision has one manifest-identified root, one home appearance per Component, at most one appearance of a Component per Diagram, and a reachable acyclic strict Diagram tree.
- Reference appearances neither copy Components nor own detail links.
- A Diagram-only or membership-only change reuses unchanged Component paths, blobs, and modes exactly.
- Crossing Relationship presentation is derived from global Relationship facts. It never rewrites or duplicates those facts.
- At most one derived boundary reference exists per absent external Component in one active Diagram; every crossing occurrence connects to it with exact direction, label, and multiplicity.
- The normal workspace remains accepted-only. Pending Diagram, membership, hierarchy, Component, and Relationship changes appear only through Changes in progress and Review changes before acceptance.
- Review keeps one exact `(base commit, candidate tree, pending generation)` binding and the existing confirmation/CAS path. Diagram comparison is presentation over the same bound base and candidate snapshots.
- Successful accepted-ref CAS remains the acceptance boundary. No Diagram-specific authority, persistence, or post-commit mechanism is introduced.
- Source repositories remain untouched, and SQLite gains no canonical Architecture or Diagram projection.
- Initialization continues to bootstrap format v1 throughout Phase 2.

## P2.1 — Load and navigate accepted v2 Diagrams

### User-visible outcome

WorkBraid can open a valid accepted format-v2 Architecture and present it as one map-centered workspace:

- the manifest-selected root opens initially;
- a compact Diagram tree and breadcrumbs navigate the strict hierarchy;
- the selected Diagram's index and map show its home and reference appearances;
- detail navigation drills through a home Component and returns with its parent anchor identifiable;
- Relationships with both endpoints present use ordinary nodes and edges;
- Relationships with one endpoint absent use one derived **Lives in** boundary reference for that external Component;
- canonical references are identified as **Also shown here** and remain ordinary Component nodes;
- activating a boundary reference opens the external Component in its home Diagram.

Opening valid v1 remains the existing implicit all-components workspace with no fabricated Diagram identity or unnecessary Diagram chrome.

P2.1 is deliberately read-only for accepted v2 Architecture. Until the v2 candidate path arrives in P2.2, the v2 workspace does not expose Component, Relationship, or Diagram mutation controls as usable actions which can only fail. It truthfully supports navigation, documentation, map interaction, explicit Refresh, project switching, and other applicable non-canonical workspace behavior. Existing accepted-v1 authoring remains fully functional and unchanged. The exact temporary read-only presentation is a P2.1 execution-packet decision, not a generic permissions or capability system.

### Implementation scope

- Extend the existing manifest dispatch and accepted-tree loader to parse the exact closed v2 manifest and non-recursive `diagrams/*.yaml` files.
- Add only concrete Diagram, appearance, hierarchy, and per-Diagram projection data required by the immutable accepted snapshot.
- Validate the complete v2 revision before publication: closed tree/schema, UUIDs, unique Diagram IDs, exact root, Diagram titles, appearance resolution, one home per Component, per-Diagram uniqueness, reference/detail restrictions, one parent per non-root, reachability, and acyclicity.
- Preserve the existing Component/Relationship parser byte-for-byte as the v2 Component interpretation path.
- Derive the Diagram tree, breadcrumbs, parent anchors, home lookup, selected-Diagram index, ordinary edges, and boundary references from that one immutable snapshot.
- Extend the existing backend-to-browser snapshot projection. The browser must not parse Diagram YAML, reconstruct hierarchy, resolve membership, or derive Architecture authority.
- Recompose the existing compact navigator only as needed to add the Diagram tree and breadcrumbs without turning it into management/dashboard chrome.
- Gate only accepted-v2 mutation affordances during this staging increment. Do not disable existing v1 authoring, add fake failure paths, or build generalized capability/permission infrastructure.
- Keep selected Diagram/component, map layout, pan, zoom, and breadcrumb focus disposable browser state.
- Preserve explicit Refresh: a valid external v2 advancement replaces all accepted Diagram projections atomically through the existing loader; invalid/unsupported state remains truthful and read-only under the existing semantics.

### Architecture invariants exercised

- v1 and v2 are separate closed portable formats under one version-aware loader.
- Root is manifest authority for Diagram navigation but remains an ordinary Diagram.
- Hierarchy derives only from home-owned `detail_diagram` links.
- Home/reference appearances are composition, not Component copies.
- Relationships remain global facts; boundary nodes are derived presentation without identity.
- One accepted snapshot supplies tree, selected Diagram, index, map, documentation, boundary references, and topology.

### Acceptance criteria

- Valid v1 opens exactly as before and is not rewritten.
- Valid v2 opens at its exact root and reconstructs the complete strict Diagram hierarchy.
- Selecting a Diagram atomically switches its index, map, documentation context, ordinary Relationships, and boundary references.
- Duplicate Diagram and Component titles remain navigable using only minimal disambiguating context.
- A canonical reference is visibly distinct from a derived boundary reference in approved product language.
- Multiple crossing Relationship occurrences to one absent Component share one derived boundary reference while retaining exact direction, labels, and multiplicity.
- Invalid root, duplicate IDs/appearances, homeless/reference-only Components, unresolved IDs, reference-owned detail links, multiple parents, unreachable Diagrams, and cycles prevent snapshot publication.
- A fresh process using the same application-data directory reconstructs the identical accepted v2 revision and workspace.
- Accepted v2 offers no usable mutation action before its candidate path exists, while navigation, documentation, map interaction, Refresh, and project switching remain coherent.
- Accepted-v1 Component and Relationship authoring behaves exactly as before.
- No pending authoring, migration, Diagram creation control, canonical layout, or new persistence enters P2.1.

### Real validation

- Use real temporary bare Git stores with exact accepted v1 and v2 trees, ordinary blobs/modes, real Git, real SQLite association, and the production open/Refresh paths.
- Cover one representative valid hierarchy with empty/root/detail Diagrams, duplicate titles, home and reference appearances, cycles and parallel Relationships, internal edges, several crossing occurrences to one external Component, and a disconnected Component topology which is still structurally valid.
- Cover a bounded invalid matrix sufficient to prove each distinct v2 authority class without exhaustive corruption testing.
- Through the built browser UI, navigate root → anchored detail → parent, tree, breadcrumbs, canonical reference, and derived boundary reference. Confirm map/index/documentation/topology remain snapshot-unified.
- Advance accepted externally to another valid non-linear v2 revision, use explicit Refresh, and verify atomic replacement; restart and reconstruct its exact SHA.
- Verify source HEAD/status/files/checksums and SQLite operational-only state remain exact.

### Dependencies and deferrals

P2.1 starts from the approved Phase 2 documentation baseline. It depends on the completed v1 loader, workspace, Refresh, safe Markdown renderer, and Phase 1 candidate-aware map infrastructure.

It deliberately defers all Diagram authoring, migration, candidate Diagram review, acceptance of Diagram changes, reference authoring controls, deletion, layout persistence, and Phase 3 behavior.

### Integration and human checkpoint

After one worker and fresh review, integrate the exact reviewed result and run ordinary checks once. The human checkpoint opens one real v1 Architecture unchanged and one real externally authored canonical v2 Architecture, navigates its hierarchy and boundaries, exercises one valid external Refresh, restarts, and confirms exact source/SQLite isolation. Stop before P2.2.

## P2.2 — Create nested Diagrams through deliberate migration and home composition

### User-visible outcome

From a valid v1 workspace, the human chooses a real Diagram task such as creating a detail Diagram. WorkBraid briefly explains that it will set up Diagrams, then places migration and the requested operation into the existing Changes in progress. The human can title the Diagram, move Component homes, create Components in an active Diagram, review the complete v1-to-v2 candidate visually and as an exact diff, deliberately accept it, and reopen the identical v2 hierarchy after restart.

The same tasks work directly against already-accepted v2 without migration language.

P2.2 also restores full existing structured authoring parity on accepted v2: Component Title, Description, outgoing Relationships, and Component creation work alongside Diagram title, detail, and home-composition operations. One v2 pending candidate may contain all of them coherently.

### Implementation scope

- Extend the one backend-held pending change set with concrete Diagram composition operations; do not add a second Diagram draft/candidate representation.
- Extend the existing Component Title, Description, outgoing-Relationship, and Component-creation operations through the same v2 candidate construction path. Preserve all established Component byte-fidelity and Relationship label/order/identity semantics.
- When the first Diagram action targets v1, construct migration inside that same pending set and complete candidate before applying the requested operation.
- If Component/Relationship changes already exist in the v1 pending set, retain them exactly. Give accepted and pending-new Components homes in the generated root before applying the requested detail/membership operation.
- Generate stable Diagram UUIDs and creation-time filenames. Serialize only the exact normative v2 manifest and Diagram schema.
- Add structured contextual actions for creating/titling one detail Diagram from a home appearance, renaming a Diagram, and moving a Component home.
- New Components created in an active Diagram receive their home there; root is used only when no Diagram context genuinely exists.
- Moving home converts an existing destination reference to home, removes the old home without leaving an implicit reference, and moves an owned detail link/subtree with the home.
- Reject a move which would place an anchor beneath its own descendant. Do not repair, clone, detach, or reconcile hierarchy automatically.
- Reuse the existing complete candidate builder and accepted loader for migration, Diagram serialization, full structural validation, immutable candidate snapshot construction, exact diff, reviewed binding, confirmation, CAS, publication, Refresh, and restart.
- Permit one v2 pending set and candidate to combine ordinary Component/Relationship edits with Diagram title, detail-link, and home-composition edits. Diagram-only and membership-only operations still reuse unchanged Component entries and blobs exactly.
- Extend the completed Phase 1 review workspace to compare Diagrams by stable Diagram ID and present Diagram additions, title changes, home movement, and detail-link changes as composition deltas.
- For a candidate-only selected Diagram, implement the approved **Before changes** fallback: nearest surviving bound-base ancestor, otherwise bound-base root; for a v1 base, the real implicit map. Show the restrained candidate-only note and leak no candidate composition or documentation into the base side.
- Keep the first migration confirmation concise and product-facing. It is not a metadata form or separate review/accept step.

### Architecture invariants exercised

- Migration is one non-canonical candidate transition, never silent accepted mutation.
- Existing pending Component/Relationship work and pending-new Component identity survive migration in the same pending set.
- Every candidate Component receives one home, and all Diagram/hierarchy validation applies to the complete candidate.
- Home movement changes Diagram composition only and preserves Component blobs and global Relationships.
- Review's v1 base remains the real implicit v1 map; candidate v2 uses its real Diagram tree.
- Review, confirmation, and accepted CAS retain one authority path across a format-version transition.
- Accepted-v2 authoring preserves the complete existing structured Component/Relationship contract while composing it with Diagram operations in the same pending/candidate authority path.

### Acceptance criteria

- Starting the first Diagram action against clean v1 creates no canonical Git change until deliberate acceptance.
- Starting it with existing pending text, Relationship, and pending-new Component changes preserves them and includes them in the same complete migration candidate.
- Cancel or whole-set discard leaves accepted v1 exact and clears the entire pending candidate through existing discard semantics.
- A valid migration produces version 2, one stable root ID, one root Diagram, exact homes for all candidate Components, and the requested detail/membership operation.
- Existing Component paths, blobs, modes, IDs, documentation, and Relationships remain exact unless independently edited in the same pending set.
- Diagram title edit, detail creation, Component creation in context, and home movement survive review, acceptance, Refresh, and restart.
- Starting from accepted v2, Title, Description, outgoing-Relationship, and Component-creation edits work normally and can coexist with Diagram-composition changes in one reviewed candidate.
- Component byte preservation, Relationship label fidelity, declaration-order fidelity without order semantics, and unchanged-blob reuse remain exact on v2.
- Destination reference-to-home conversion, old-home removal, anchored-subtree movement, and descendant-cycle rejection follow the baseline exactly.
- Normal accepted v1/v2 workspace remains unchanged before CAS.
- Review remains snapshot-unified and its candidate-only Diagram fallback never leaks candidate state into Before changes.
- Successful CAS advances manifest, Diagram tree, map/index/documentation, and revision together; stale/ref-update protection remains unchanged.

### Real validation

- Use real Git candidates to prove exact v1-to-v2 tree construction, stable IDs, normative keys, closed schema, unchanged Component entries/blobs/modes, and one successor parent/CAS.
- Cover migration with no pending work and with one pending set containing an accepted Component edit, Relationship edit, and pending-new Component.
- Cover detail creation, Diagram title mutation, active-context Component creation, home movement into a destination reference, subtree reparenting, and a rejected descendant cycle.
- Begin separately from an already-accepted v2 revision and combine at least one ordinary Component/Relationship edit with a Diagram-composition change through candidate validation, review, CAS acceptance, and fresh-process reconstruction.
- Prove validation failure retains the complete pending candidate and accepted v1/v2 remains exact.
- Prove review's base/candidate projections, v1 implicit Before side, v2 candidate tree, composition deltas, exact diff focus, candidate-only fallback, binding invalidation, stale authority, post-CAS boundary, and fresh-process reconstruction.
- Through the built real application, begin from accepted v1, retain pre-existing pending Component/Relationship work, request a detail Diagram, review one combined migration candidate, accept once, navigate the resulting v2 tree, and restart at the exact accepted SHA.
- Verify source and SQLite isolation exactly.

### Dependencies and deferrals

P2.2 starts only after P2.1 is integrated and human-approved. It reuses P2.1's sole v2 loader/projection and all existing I2/I3/Phase 1 pending, candidate, review, and authority paths.

It deliberately defers structured add/remove reference actions to P2.3. It does not defer loading or preserving externally authored reference appearances, converting a destination reference during home movement, or rendering boundary references already required by P2.1.

### Integration and human checkpoint

After one worker and fresh review, integrate and run ordinary checks once. The human checkpoint starts from real accepted v1 with one existing multi-file pending set, initiates one actual detail-Diagram task, verifies the concise setup explanation, completes title/home authoring, reviews the exact combined migration candidate, accepts it, navigates the accepted result, restarts, and confirms exact v2 reconstruction. It then creates one accepted-v2 candidate combining ordinary Component/Relationship authoring with a Diagram-composition change, accepts it, and reconstructs it after another fresh start. Confirm source/SQLite isolation throughout. Stop before P2.3.

## P2.3 — Author reusable references and complete the Phase 2 gate

### User-visible outcome

Within an active v2 Diagram, the human can deliberately show a Component which lives elsewhere and later stop showing that reference. Canonical **Also shown here** appearances behave as ordinary nodes, while non-members reached only by crossing Relationships remain derived **Lives in** boundary references. The complete home/reference/hierarchy model is reviewable, accepted, refreshable, and reconstructable.

### Implementation scope

- Add structured contextual actions to add and remove a reference appearance using stable Component identity with human-readable titles and minimal duplicate-title context.
- Preserve the existing pending set and candidate builder. Do not add appearance IDs, browser-owned YAML, generic membership infrastructure, or a second Diagram review representation.
- Prevent duplicate appearances, home+reference in one Diagram, and detail links on references through complete-candidate validation.
- Preserve surviving appearance source order where practical without assigning it domain meaning or row identity.
- When adding a reference for a Component currently represented by a boundary reference, replace only the presentation in the candidate Diagram: the canonical reference becomes an ordinary node and the same global Relationship facts become ordinary edges.
- Removing a reference restores derived boundary presentation only when crossing Relationships require it. It never deletes or edits the Component or Relationships.
- Extend the existing Diagram composition review treatment to reference additions/removals. Do not classify ordinary↔boundary presentation changes caused solely by membership as Relationship fact additions/removals.
- Keep exact canonical Diagram-file diff directly inspectable and retain candidate-only Diagram fallback and all Phase 1 review behavior.

### Architecture invariants exercised

- One home remains mandatory; references are additional canonical appearances only.
- A Component appears no more than once per Diagram and a reference owns no hierarchy.
- Relationship identity, exact labels, direction, multiplicity, and source ownership remain unchanged by appearance authoring.
- Boundary references remain identity-free derived presentation and collapse by absent Component within a Diagram.
- Reference acceptance uses the same candidate snapshot, exact diff, binding, CAS, publication, Refresh, and restart path.

### Acceptance criteria

- Add/remove reference is structured and never requires UUID or YAML entry.
- Duplicate titles receive only necessary context; IDs and filenames do not become general authoring chrome.
- Adding a reference does not rewrite the Component blob or any Relationship fact.
- Removing a reference does not remove the Component, its home, its documentation, or Relationships.
- The active Diagram shows one canonical reference node after addition and, when applicable, one derived boundary node after removal, with every edge occurrence preserved.
- Invalid duplicate/home+reference/reference-detail candidates remain pending and cannot enter Review changes.
- Review presents reference composition changes without false Relationship additions/removals and keeps all selected tree/map/index/detail surfaces snapshot-unified.
- Acceptance, explicit external Refresh, stale protection, discard/project switching, safe Markdown, restart reconstruction, source isolation, and SQLite boundaries do not regress.

### Real validation

- Use real Git and the production candidate path for add/remove reference, duplicate rejection, exact Diagram blob rewrite, unchanged Component/Relationship blobs, appearance order fidelity, and ordinary↔boundary projection changes.
- Cover several differently labelled and parallel crossing Relationships sharing one external Component, proving one derived boundary reference and exact edge multiplicity.
- Cover review comparison, exact diff, binding invalidation, acceptance, a valid external Refresh, stale handling, and restart reconstruction.
- Through the built UI, add a reference, navigate its canonical Component documentation, inspect ordinary edges, review and accept; then remove it in another candidate and verify boundary presentation returns without semantic Relationship delta.

### Integration and final human checkpoint

After one worker and fresh review, integrate and run ordinary checks once. Perform one cohesive Phase 2 checkpoint from accepted v1:

1. create pre-existing pending Component/Relationship work including a pending-new Component;
2. initiate a real detail-Diagram action and verify migration plus the request form one pending candidate;
3. title the detail, move Component homes, create a Component in active context, and add a reusable reference;
4. verify the normal accepted workspace remains exact before review;
5. review **With changes** and the real v1 implicit **Before changes**, including candidate-only Diagram fallback and exact canonical diff;
6. deliberately accept and verify root/tree/breadcrumb/drill/index/map/documentation/boundaries advance together;
7. author and accept one further v2 home/reference composition change;
8. externally advance to another valid v2 revision and use explicit Refresh;
9. completely restart and reconstruct the exact final accepted hierarchy, identities, documentation, Relationships, appearances, boundary behavior, and revision;
10. verify source repository and SQLite isolation remain exact.

Phase 2 completes only after this final checkpoint receives explicit human **PASS**. Stop before Phase 3 planning or implementation.

## Phase-wide explicit exclusions

Do not introduce:

- Diagram deletion or general hierarchy lifecycle;
- multiple Diagram parents, Diagram DAGs, or reusable child Diagrams;
- multiple appearances of one Component inside one Diagram;
- multiple detail Diagrams per home anchor;
- component deletion or relationship identity/lifecycle;
- layout, position, size, route, bend-point, shape, annotation, Diagram-kind, renderer, or appearance-identity fields;
- manual or persisted layout, graphical Diagram editing, or graphical Relationship authoring;
- persisted/project-scoped pending changes, migration state, or review state;
- pending/draft overlays on the normal accepted map;
- a generic migration framework, schema framework, graph model, navigation framework, semantic diff engine, or second candidate/review/acceptance path;
- merge, rebase, reconciliation, automatic repair, history, revert, proposals, export, or sync;
- URL-backed restoration, syntax highlighting, rendered/semantic Markdown diff, dedicated themed-scrollbar work, or other recorded non-gating polish;
- UML/class semantics, additional Diagram kinds, isometric rendering, Planning, Agent Control, or another vertical.

## Planning completion boundary

This approved plan authorizes preparation of P2.1 only. Prepare it as a proposed execution packet on top of this plan's exact docs-inclusive commit, with one worker, fresh independent review, integration procedure, real human checkpoint, and explicit stop before P2.2. Do not implement or dispatch until that packet receives human approval.
