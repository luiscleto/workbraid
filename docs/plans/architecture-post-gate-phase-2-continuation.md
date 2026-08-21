# Post-Gate Architecture Phase 2 continuation plan

Status: Approved

Alpha-stage baseline: `456ec4a5655cd896a426b57e4bb1e643b6e52293`

Historical approved Phase 2 plan: `docs/plans/architecture-post-gate-phase-2.md`, committed at `5ca24c23b799323182e9b3164d2cfbf498e19734`

Completed P2.1 implementation: `db2451e2dc46abe3d34d5dd5fb6e5564ca2580a6`

Completed P2.1 record: `4cd0a05f888e93a790a4c11a92fb35d60f5e7a12`

Target: Writable format-v2 Architecture, nested Diagram composition, reusable references, and the Phase 2 gate

Architecture or product decisions discovered during planning or implementation require explicit human approval rather than silent changes to this plan or its baselines.

## Superseding scope

The original approved Phase 2 plan remains historical evidence for the design and execution of P2.1. P2.1 is complete and is not redefined or reopened here.

This continuation plan supersedes only the original plan's remaining P2.2/P2.3 direction and the forward-looking Phase 2 statements which required:

- format-v1 initialization throughout Phase 2;
- ordinary accepted-v1 Component and Relationship authoring;
- combining migration with the first Diagram operation;
- composing migration with arbitrary existing v1 pending work; and
- beginning the final Phase 2 gate from that combined migration workflow.

The original plan's shared authority, identity, Diagram-domain, fidelity, review, CAS, source-isolation, SQLite, and scope boundaries continue unless this plan expressly replaces them. Completed plans and execution records remain unchanged.

Completed planning, execution, and evidence documents are immutable historical records, but executable tests and production-browser scenarios are living verification. When P2.2 replaces v1 new-store bootstrap and P2.1's current-v2 **View only** staging, update existing executable Gate, Phase 1, and P2.1 scenarios proportionately wherever they assert those superseded behaviors. Preserve their still-valid accepted-ref authority, source/pending isolation, exact review/CAS, safe Markdown, Refresh, restart, and SQLite evidence. Do not retain obsolete product behavior merely to keep an old assertion green, create parallel legacy scenarios for unsupported behavior, or turn the scenarios into a generic harness.

## Phase objective from P2.2 onward

WorkBraid becomes a normally writable format-v2 Architecture product. New stores begin as v2 with one empty root Diagram. Existing Component and Relationship authoring works through the same backend-held pending set and candidate/review/CAS path while preserving Diagram composition. A bounded explicit **Set up diagrams** action can transition a readable v1 store to v2 without retaining general v1 authoring compatibility.

On that writable foundation, WorkBraid adds nested detail-Diagram and home-composition authoring, then structured reusable-reference authoring. The completed Phase 2 product loads, navigates, authors, reviews, accepts, refreshes, and reconstructs the approved strict Diagram model without replacing Components or Relationships with generic graph objects.

## Re-evaluated increment shape

Use three sequential product increments:

1. **P2.2 — Writable v2 foundation and legacy Diagram setup.** Change new-store initialization to v2, remove P2.1's temporary current-v2 staging restriction, restore ordinary Component/Relationship authoring on accepted v2, assign new Components a home, and provide the one reviewed v1 **Set up diagrams** transition.
2. **P2.3 — Nested Diagram composition.** Add detail-Diagram creation and titling, Diagram rename, home movement, subtree reparenting, hierarchy validation, and Diagram-aware candidate review on the writable v2 foundation.
3. **P2.4 — Reusable references and the Phase 2 gate.** Add structured **Also shown here** appearance authoring and prove the complete home/reference/boundary product in one cohesive gate.

These are independently useful product boundaries rather than backend/frontend tickets:

- P2.2 leaves every newly initialized Architecture writable in the long-term v2 format and removes the temporary v2 dead end even though nested composition is not yet authorable.
- P2.3 leaves a complete nested decomposition workflow without requiring reusable references.
- P2.4 adds the separate repeated-appearance workflow and closes the complete Phase 2 model.

Merging P2.2 and P2.3 would unnecessarily combine initialization, legacy transition, normal Component parity, Diagram candidate serialization, hierarchy authoring, hierarchy validation, and review presentation into one gate. They are not inseparable: Component creation needs one concrete home insertion into an already selected Diagram, but it does not require detail creation, arbitrary home movement, reparenting, or hierarchy-lifecycle UI. Keep them sequential.

Each increment receives one proposed execution packet only after the preceding increment is integrated and human-approved. Each packet pins one exact docs-inclusive worker base, uses one implementation worker, receives fresh independent review, integrates the exact reviewed result, runs ordinary checks once, and stops for a real human checkpoint before the next increment.

## Shared invariants

- `refs/heads/accepted` remains the sole accepted authority.
- The P2.1 version-aware accepted-snapshot loader remains the only canonical v1/v2 interpretation path. Candidate validation, reviewed projections, accepted publication, Refresh, and restart reuse it.
- New accepted state is created only through the existing complete candidate, exact unified diff, exact `(base commit, candidate tree, pending generation)` review binding, confirmation, stale observation, and accepted-ref compare-and-swap path.
- Successful accepted-ref CAS remains the acceptance boundary. No Diagram-specific ref, acceptance path, publication mechanism, or recovery authority is introduced.
- Components retain stable IDs, canonical Markdown, filenames, ordinary-file modes, and global source-owned Relationships. Diagram composition neither copies nor owns those facts.
- Every valid v2 revision has one manifest-selected root, one home per Component, at most one appearance of a Component per Diagram, and a reachable acyclic strict Diagram tree.
- Diagram-only and membership-only changes reuse unchanged Component tree entries and blobs exactly.
- Relationship direction, exact label, multiplicity, source ownership, and order fidelity remain unchanged. Composition-driven ordinary/boundary presentation changes are not Relationship fact changes.
- The normal workspace remains accepted-only. Pending Component, Relationship, Diagram, membership, and hierarchy changes appear only through Changes in progress and Review changes before acceptance.
- Backend mutation eligibility is based on the synchronized server-owned loaded project, accepted snapshot format and authority state, and pending state—not a browser format, capability, or stale assumption.
- Source repositories remain untouched. SQLite retains only approved operational association state and gains no canonical Architecture, Diagram, graph, document, navigation, layout, migration, or review projection.
- Fresh-process reopen reconstructs the exact accepted revision entirely from the private Git store and operational association.

## P2.2 — Writable v2 foundation and legacy Diagram setup

### User-visible outcome

Initializing a new project creates an empty, immediately writable Architecture with one root Diagram. The human can use existing structured Component Title, Description, and outgoing-Relationship authoring and can create Components in the active Diagram. Changes remain pending, review through the existing visual and exact-diff workspace, accept deliberately, and reconstruct after restart.

A readable older Architecture offers one concise **Set up diagrams** action. That action creates only a reviewed Diagram-setup change. After acceptance, the same Architecture becomes normally writable. There is no migration wizard, combined first-detail task, or format terminology in normal UI.

### Implementation scope

- Change explicit new-store initialization to create the exact approved parentless v2 bootstrap: normative `architecture.yaml`, one newly generated root Diagram ID, one `100644` `diagrams/root.yaml` blob, project-derived initial root title, no appearances, no Components, and no Relationships.
- Preserve `diagrams/root.yaml` as a creation-time filename convention only. Loading continues to accept every conforming `diagrams/*.yaml` filename, and manifest `root_diagram` remains sole root authority.
- Extend the existing single candidate construction path to preserve and, where required, serialize normative v2 manifest/Diagram trees while validating the complete candidate through the P2.1 loader.
- Remove P2.1's temporary **View only** staging and direct v2 mutation rejection only when the v2 candidate path is complete. Do not leave a generic capability layer or duplicated format eligibility checks.
- Route existing Component Title, Description, and outgoing-Relationship operations through the v2 candidate path with all established source-byte, heading/body, frontmatter, label, declaration-order, mode, path, identity, and unchanged-blob guarantees.
- Create a Component through the existing structured operation and add exactly one home appearance in the currently active Diagram. Use root only when there is genuinely no active Diagram context. Preserve every unrelated Diagram and Component entry exactly.
- Allow one v2 pending candidate to combine Component content, Relationship, and Component-creation/home-insertion changes. Keep it one backend-owned pending set and one complete candidate interpretation.
- Add the one v1 **Set up diagrams** operation. It is available only against a current valid v1 snapshot with no pending set and constructs one format-v2 candidate containing the evolved manifest, one stable root Diagram, and one root home for every accepted Component.
- Preserve every v1 Component path, blob, ordinary-file mode, ID, Markdown byte, and Relationship in the setup candidate. Do not perform a detail-Diagram or other authoring operation in that candidate.
- Treat a non-setup v1 pending set, if somehow present, as visible read-only evidence with whole-set Discard only. It cannot be edited, reviewed, accepted into v1, or merged into setup. Setup remains unavailable until discard.
- Reject ordinary v1 mutation and late/stale-client requests atomically under the existing state synchronization boundary without creating hidden pending work.
- Extend the existing candidate-aware review only as needed for complete v2 Component candidates and the setup transition. **Before changes** for setup remains the real v1 implicit map; **With changes** is the exact generated v2 root. The exact diff remains directly inspectable and authoritative.
- During setup review, an unchanged Component receives no content-changed classification merely because the candidate gives it a home, and an unchanged Relationship fact receives no added/removed classification merely because Diagram composition now exists. Present the generated root and homes as Diagram/setup composition change. Actual Component and Relationship semantic deltas continue to use their existing exact comparison rules.
- Preserve known stale/non-current and indeterminate Refresh semantics above ordinary writable/setup presentation.

### Architecture invariants exercised

- Initialization publishes one parentless canonical commit only when `accepted` successfully names it.
- The v2 root derives only from the manifest identity, never from `diagrams/root.yaml` naming.
- Ordinary authoring and legacy setup share the one pending/candidate/review/CAS authority path.
- Setup is a format-v2 candidate over an exact v1 base, not a second migration representation or special commit path.
- New-Component identity and home assignment are created once by the backend and validate against the complete v2 candidate.
- Existing accepted nested/reference composition remains intact while ordinary Component/Relationship edits are made.

### Acceptance criteria

- A fresh store's first accepted revision is parentless, exact normative v2, contains one empty root Diagram, and reconstructs as writable after restart.
- New-store setup has no intermediate v1 revision and shows no migration/setup explanation.
- Current valid accepted v2 exposes existing structured Add/Edit/Relationship actions and no P2.1 **View only** treatment.
- Title-only, Description-only, Relationship-only, and combined edits retain every established byte/fidelity invariant on v2.
- A new Component receives one stable ID/path/blob and one home in the selected Diagram; unrelated Component and Diagram entries are reused exactly.
- A candidate may coherently contain ordinary Component edits, Relationship edits, and new Component/home insertion and is reviewed and accepted once.
- Current accepted v1 permits no ordinary mutation and creates no pending state from direct or late requests.
- **Set up diagrams** is available only with no pending set; its one candidate contains only v2 manifest/root/home changes and preserves all Component/Relationship facts exactly.
- Non-setup v1 pending evidence is read-only and discard-only; it cannot reach review or acceptance and never contaminates setup.
- Setup's v1 **Before changes** side is the exact implicit map and its v2 **With changes** side is snapshot-unified without a fabricated base Diagram identity.
- For non-empty v1 setup, unchanged Component content and Relationship facts remain visually unchanged while only the generated manifest/root/home composition is classified as setup change; the complete unified diff remains canonical evidence.
- Validation, stale binding, pre-CAS failure, CAS success, post-CAS recovery, Refresh, project switching, discard, safe Markdown, source isolation, and SQLite boundaries do not regress.

### Real validation

- Use real Git to prove exact direct-v2 bootstrap tree, modes, parentlessness, stable store/root identities, accepted-ref creation, retry behavior, and fresh-process reconstruction.
- Exercise the production handler race between initialization and reopen proportionately; do not build generalized initialization coordination.
- Starting from accepted v2 with an existing nested/reference hierarchy, combine one exact Description edit, one Relationship edit, and one new Component in a non-root active Diagram. Prove unchanged Component and Diagram blobs/modes/paths are reused except the one Diagram receiving the new home.
- Cover Component title/H1/body/frontmatter fidelity and Relationship free-text/order/Unicode behavior through the v2 candidate path without replaying every completed I2/I3 case.
- Construct and accept one real v1 setup candidate. Prove manifest/root/home output, exact preservation of all Component entries and Relationships, v1 implicit-map Before view, v2 root With view, exact diff, binding, CAS, and restart.
- Use a non-empty connected v1 fixture to prove setup adds only expected manifest/root/home composition classification: unchanged Component nodes are not content-changed, unchanged Relationship facts are not added/removed, and the exact diff still exposes every canonical format transition byte.
- Cover setup cancel/discard, stale accepted authority, direct v1 mutation rejection, and a concrete non-setup v1 pending state which remains inspectable/discardable but cannot mutate, review, accept, or enter setup.
- Race a late v1 mutation or setup request with project switch/Refresh under the existing synchronization boundary and prove one coherent server-owned outcome with no hidden pending state.
- Through the built UI, initialize a fresh store, create/edit connected Components in root, review/accept, and restart. Separately open controlled valid v1, use **Set up diagrams**, review/accept the transition, and reopen exact v2.
- Verify source HEAD/status/files/checksums and SQLite operational-only state remain exact.

### Dependencies and deliberate deferrals

P2.2 starts from alpha baseline `456ec4a5655cd896a426b57e4bb1e643b6e52293` and completed P2.1. It reuses P2.1 loading/navigation plus the completed I2/I3/Phase 1 pending, candidate, review, Refresh, CAS, and workspace paths.

It deliberately defers detail-Diagram creation, Diagram rename, arbitrary home movement, subtree reparenting, structured reference authoring, and the final Phase 2 gate. It introduces no layout state or graphical editing.

### Integration and human checkpoint

After one worker and fresh independent review, integrate the exact reviewed result and run ordinary checks once. The human checkpoint proves one new direct-v2 project can be initialized, authored through a mixed Component/Relationship/new-home candidate, accepted, stopped, and reconstructed. A separate valid v1 project proves read-only navigation, standalone **Set up diagrams**, exact base/candidate review, deliberate acceptance, and reconstruction. Exercise one defensive non-setup v1 pending/discard case without turning it into a compatibility matrix. Confirm source and SQLite isolation. Stop before P2.3.

## P2.3 — Nested Diagram composition

### User-visible outcome

From a writable accepted-v2 workspace, the human can create and title a detail Diagram from a Component's home appearance, rename Diagrams, and move Component homes. The Diagram tree, breadcrumbs, drill-down, boundary presentation, Changes in progress, candidate-aware review, acceptance, Refresh, and restart remain one coherent product workflow.

### Implementation scope

- Extend the one backend-held pending set with concrete detail creation, Diagram-title change, and home-move operations. Do not create a Diagram draft store or second candidate representation.
- Generate stable Diagram UUIDs and creation-time filenames for new detail Diagrams; filenames carry no identity.
- Create at most one detail Diagram from a home appearance and place the link only on that home entry.
- Rename a Diagram without changing its ID, filename, appearances, hierarchy, or unrelated blob bytes.
- Move a Component home by removing the old home and adding the destination home. Convert an existing destination reference to home rather than duplicating it, and do not silently retain a reference at the old home.
- Move an owned detail link with its home, reparenting that child and its subtree. Validate the complete resulting tree and reject a move beneath the anchor's own descendant without repair, detach, cloning, or reconciliation.
- Permit Component/Relationship operations from P2.2 and Diagram composition operations to coexist in one pending candidate.
- Keep incomplete pending UI state backend-owned and inspectable where appropriate, but allow Review changes only after the complete candidate passes the existing version-aware structural validation.
- Extend the existing candidate-aware review to match Diagrams by stable ID and show added Diagrams, title changes, home removals/additions, home moves, and detail-link changes as composition deltas.
- Keep Relationship multiset comparison exclusively for actual Relationship fact changes. An internal edge becoming a boundary edge, or the reverse, solely because of composition is not a Relationship addition/removal.
- Implement the approved candidate-only Diagram fallback without leaking candidate composition: nearest surviving base ancestor, otherwise base root. The setup-transition v1 fallback from P2.2 remains unchanged.

### Architecture invariants exercised

- The strict tree derives only from root identity and home-owned detail links.
- Home movement changes composition, not Component or Relationship identity.
- Destination reference conversion produces one appearance, not home-plus-reference duplication.
- Reparenting moves one existing child subtree through the anchor link and never creates another parent.
- Candidate Diagram tree, selected Diagram, index, map, documentation, appearances, boundaries, and topology remain snapshot-unified in review.

### Acceptance criteria

- Detail creation, Diagram title, stable ID/path, and home-owned link survive review, acceptance, Refresh, and restart.
- Diagram rename changes only required Diagram source and preserves identity and unrelated entries.
- Home move preserves the Component blob/path/mode, documentation, and all global Relationships.
- Destination reference becomes home in place, the old home is removed, and no duplicate appearance remains.
- Moving an anchoring home reparents its exact child/subtree while preserving Diagram identities and content.
- A descendant-cycle move and representative incomplete/invalid Diagram states block review with concise actionable product guidance and retain the pending set for correction.
- One pending candidate may combine a Component content edit, Relationship edit, detail creation or title change, and home movement.
- Review distinguishes composition changes without false Component identity or Relationship fact changes; candidate-only Diagram fallback is truthful and exact diff focus remains available.
- Accepted workspace remains unchanged before CAS and advances atomically after CAS. Restart reconstructs the exact hierarchy.

### Real validation

- Use real Git candidates for detail creation, Diagram rename, home movement, destination reference conversion, old-home removal, subtree reparenting, unchanged-blob/mode reuse, and rejected descendant cycle.
- Cover missing home, duplicate appearance, illegal reference/detail combination encountered from existing state, missing/multiple parent, unreachable Diagram, cycle, and malformed/incomplete Diagram title proportionately through the complete candidate path. Do not replay P2.1's reader corruption matrix or build a generic validation engine.
- Prove validation failure retains the entire mixed pending set and accepted Git/workspace remains exact.
- Prove Diagram comparison, actual Relationship multiset comparison, composition-driven ordinary/boundary presentation, candidate-only fallback, exact diff focus, review-binding invalidation, stale protection, CAS boundary, and post-CAS reconstruction.
- Through the built UI, start from writable accepted v2, create nested detail structure, move homes, deliberately trigger and correct one hierarchy-invalid move, combine an ordinary Component/Relationship edit, review exact base/candidate state, accept, navigate, and restart.
- Verify source and SQLite isolation exactly.

### Dependencies and deliberate deferrals

P2.3 depends on completed P2.2's normal v2 candidate writer, Component/Relationship parity, home insertion, setup transition, and removal of temporary staging.

It deliberately defers structured add/remove reference actions to P2.4. Existing canonical references remain loadable/renderable and may be converted by home movement. Diagram deletion, general hierarchy lifecycle, manual layout, and graphical editing remain out of scope.

### Integration and human checkpoint

After one worker and fresh independent review, integrate and run ordinary checks once. The human checkpoint builds a nested hierarchy through the real UI, exercises Diagram title and home composition, corrects one invalid descendant move, combines ordinary Component/Relationship work, reviews composition and exact diff, deliberately accepts, navigates the result, and reconstructs the exact accepted SHA after a complete restart. Confirm source and SQLite isolation. Stop before P2.4.

## P2.4 — Reusable references and the Phase 2 gate

### User-visible outcome

Within an active v2 Diagram, the human can deliberately show a Component which lives elsewhere and later stop showing it. Canonical **Also shown here** appearances behave as ordinary nodes, while Components represented only because of crossing Relationships remain derived **Lives in** boundaries. The complete home/reference/hierarchy model is reviewable, accepted, refreshable, and reconstructable.

### Implementation scope

- Add structured contextual actions to add and remove a reference appearance using stable Component identity, human-readable title, and collision-only context.
- Preserve the one pending set and candidate builder. Do not add appearance IDs, browser-owned YAML, generic membership infrastructure, or another Diagram review model.
- Reject duplicate appearance, home-plus-reference in one Diagram, and any detail link on a reference through complete-candidate validation.
- Preserve surviving appearance source order where practical without assigning domain ordering or row identity.
- Adding a reference for a boundary Component changes only candidate composition/presentation: one canonical ordinary node replaces the derived boundary and all real Relationship facts remain exact.
- Removing a reference restores one derived boundary only when crossing Relationships require it; it never removes or edits the Component, home, documentation, or Relationships.
- Extend existing composition review to reference additions/removals without false Relationship additions/removals.

### Architecture invariants exercised

- Every Component retains one home; references are additional canonical appearances only.
- A Component appears at most once per Diagram and references own no hierarchy.
- Boundary references remain derived, identity-free, and collapsed by absent external Component while exact edge direction, label, and multiplicity survive.
- Reference changes use the same candidate snapshot, exact diff, binding, CAS, publication, Refresh, and restart path.

### Acceptance criteria

- Add/remove reference is structured and never requires UUID or YAML entry.
- Duplicate titles receive only necessary context; IDs and paths do not become general chrome.
- Adding/removing a reference rewrites no Component blob or Relationship fact.
- Ordinary-to-boundary and boundary-to-ordinary presentation preserves every crossing Relationship occurrence.
- Invalid reference composition remains pending and cannot enter Review changes.
- Review presents reference composition changes without false semantic deltas and retains snapshot unity, exact diff, and acceptance authority.
- Full Phase 2 workspace, authoring, Refresh, stale handling, discard/project switching, safe Markdown, restart, source isolation, and SQLite boundaries pass together.

### Real validation

- Use real Git and the production candidate path for add/remove reference, exact Diagram rewrite, unchanged Component/Relationship blobs, duplicate rejection, source-order fidelity, and ordinary/boundary changes.
- Cover several differently labelled and parallel crossing Relationships sharing one absent Component and prove one derived boundary plus exact occurrence multiplicity.
- Cover review comparison, binding invalidation, acceptance, valid external Refresh, stale behavior, and restart reconstruction proportionately.
- Through the built UI, add and accept a reference, navigate canonical documentation and ordinary edges, then remove and accept it and verify boundary presentation returns without a semantic Relationship delta.

### Integration and final human checkpoint

After one worker and fresh independent review, integrate and run ordinary checks once. Run one cohesive real Phase 2 gate beginning from a freshly initialized v2 store:

1. verify exact parentless v2 bootstrap and root identity;
2. create connected Components in root through structured authoring;
3. create and title nested detail Diagrams, move homes, and create a Component in active context;
4. add a reusable reference and verify ordinary versus boundary presentation;
5. combine Component, Relationship, Diagram-title, home, and reference changes in one pending candidate while the normal accepted workspace remains exact;
6. review **With changes** and **Before changes**, Diagram composition, candidate-only fallback, real Relationship deltas, presentation-only boundary changes, and the complete exact canonical diff;
7. deliberately accept and verify tree, breadcrumbs, map, index, documentation, appearances, boundaries, Relationships, and revision advance together;
8. remove a reference in a second candidate and deliberately accept;
9. externally select another valid v2 revision and adopt it through explicit Refresh;
10. completely restart and reconstruct the exact final accepted hierarchy, identities, documentation, Relationships, appearances, boundary behavior, and revision;
11. verify source repository and SQLite isolation remain exact.

Include one bounded valid-v1 **Set up diagrams** regression check during the Phase 2 gate or immediately alongside it, but do not make legacy conversion the primary cohesive workflow and do not replay defensive pending-state matrices already proven by P2.2.

Phase 2 completes only after this gate receives explicit human **PASS**. Stop before Phase 3 planning or implementation.

## Phase-wide explicit exclusions

Do not introduce:

- ordinary accepted-v1 Component or Relationship authoring;
- composition of v1 setup with non-setup pending work;
- v2-to-v1 downgrade or a generic migration/version-negotiation framework;
- Diagram deletion or general hierarchy lifecycle;
- multiple Diagram parents, Diagram DAGs, or reusable child Diagrams;
- multiple appearances of one Component inside one Diagram;
- multiple detail Diagrams per home anchor;
- Component deletion or Relationship identity/lifecycle;
- layout, position, size, route, bend point, shape, annotation, Diagram-kind, renderer, or appearance-identity fields;
- manual or persisted layout, graphical Diagram editing, or graphical Relationship authoring;
- persisted/project-scoped pending changes, migration state, review state, selection, navigation, or layout;
- pending/draft overlays on the normal accepted map;
- a generic schema, graph, validation, navigation, membership, migration, review, or workflow framework;
- another parser, candidate builder, review binding, acceptance path, accepted ref, or SQLite projection;
- merge, rebase, reconciliation, automatic repair, history, revert, proposals, export, or sync;
- URL-backed restoration, syntax highlighting, rendered/semantic Markdown diff, dedicated themed-scrollbar work, or other recorded non-gating polish;
- UML/class semantics, additional Diagram kinds, isometric rendering, Planning, Agent Control, or another vertical.

## Planning completion boundary

This proposed continuation plan authorizes no implementation. After human approval and a separate conventional planning commit, prepare P2.2 only as a proposed execution packet with one exact docs-inclusive worker base, one worker, fresh independent review, integration procedure, ordinary checks, real human checkpoint, and explicit stop before P2.3. Do not dispatch or implement until that packet is separately approved.
