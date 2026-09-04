# Post-Gate Architecture roadmap

Status: Approved direction

Scope: Architecture as a standalone product after Gate 1

This roadmap records product sequencing and boundaries. It is not an implementation plan or a portable-store schema.

Architecture remains the only WorkBraid vertical in scope here. Agent Control is explored separately, and Planning remains deferred.

## Product principles

- Architecture Components and Relationships remain semantic Architecture facts with their existing identities and documentation.
- A first-class Diagram owns composition and presentation rather than replacing those facts with a generic node/edge domain.
- Each Architecture workspace context projects one exact coherent snapshot: **Accepted** shows exact accepted state, while a selected durable proposal shows only its own complete proposed state. They are never overlaid.
- Comparative change visualization belongs in the deliberate Review changes task and derives from its exact immutable reviewed candidate. A durable proposal workspace may show one complete proposed snapshot, but never an Accepted/proposal overlay.
- Exact canonical diff review and accepted-ref authority remain intact as richer visual review and diagram presentation are added.
- Optional renderers may present approved diagram state differently without creating another canonical Architecture representation.

## Phase 1 — Candidate-aware review workbench — complete

Improve the then-existing Architecture review workflow without introducing Diagram presentation state:

- one Review changes surface combining candidate-aware visual review and the complete exact unified diff;
- an exact bound base/candidate toggle with snapshot-unified index, map, documentation, and relationships;
- stable-ID and relationship-multiset visual matching;
- visual distinction for added and changed topology and removed relationship facts;
- focused navigation between visual changes, review context, and canonical diff;
- deterministic non-canonical review layout;
- review-specific validation, fix, clear/deselect, and basic diff-readability polish.

At Phase 1 completion the normal workspace map remained accepted-only. Change Sets 1 later proposes an explicit whole-proposal workspace context rather than an overlay; semantic/rendered Markdown diff, syntax highlighting, and dedicated themed-scrollbar work remain deferred.

## Phase 2 — First-class Architecture Diagrams and nested navigation — complete

Introduce canonical format-v2 Architecture Diagrams while preserving Components and Relationships as the semantic Architecture facts.

Approved domain direction:

- every Diagram has an immutable stable ID and mutable human-readable title;
- every v2 Architecture has exactly one manifest-identified root Diagram;
- Diagrams form a strict rooted tree: every non-root has one parent anchor, every Diagram is reachable, and cycles are invalid;
- every Component has exactly one home appearance and may additionally have reusable reference appearances in other Diagrams, at most once per Diagram;
- only a parent Diagram's home appearance may own its one optional detail-Diagram link;
- global Relationships crossing the active Diagram boundary use derived external references rather than copied facts or implicit membership;
- automatic layout remains disposable and non-canonical;
- new projects are created by name, receive a WorkBraid-non-editable stable human-facing slug plus immutable store UUID, and initialize directly as writable v2 with one empty manifest-identified root Diagram;
- private accepted Git stores are the project catalog authority, `/projects/<slug>` restores a project, and no source-folder association or SQLite catalog is retained;
- old alpha format-v1 stores and path-association databases are disposable and unsupported; there is no setup transition, migration, compatibility adapter, or v3;
- candidate-aware Diagram review extends the existing exact base/candidate snapshots, unified diff, binding, confirmation, and accepted-ref CAS path.

The smallest useful product slice includes slug-native catalog/create/open/reload, direct-v2 initialization and normal Component/Relationship authoring, root viewing, detail-Diagram creation and titling, home movement, reference appearance authoring, tree/breadcrumb/drill navigation, boundary-reference navigation, exact candidate review, deliberate acceptance, and restart reconstruction. Diagram deletion and general hierarchy lifecycle are not part of this slice.

Phase 2 is complete at completion record `8cd9ce145980fbc57377e729975940b9e2f0b8b6`.

## Architecture Agent Access 1 — complete

Add local agent interfaces over the completed Architecture product before investing in rich diagram presentation:

- a first-class scriptable CLI in the WorkBraid binary;
- one embedded `workbraid --skill` Markdown guide for an agent with no repository context;
- typed MCP tools through a stateless stdio bridge to the running WorkBraid process;
- complete semantic read/authoring parity with the browser's non-presentational Architecture actions;
- the existing exact pending, review binding, diff, stale, confirmation, CAS, and publication authority shared by browser, CLI, and MCP;
- stable machine results and typed error classifications; and
- independent black-box weak-agent CLI and MCP usability gates followed by a small human UI checkpoint.

This is an interface layer over Architecture, not another vertical or authority. It adds no raw Git/YAML agent path, per-agent pending state, remote exposure, multi-user collaboration, autonomous agent runtime, or generic command/tool framework. `docs/architecture-agent-access-v0.md` contains the approved contract.

Architecture Agent Access 1 is complete at completion record `540724919165ea95c9ee3088aca084d91eb8e1c3`, with final implementation `0a82c89d2a214896fe6b7020fe312cdd109beaeb`.

## Architecture Change Sets 1 — complete

Replace the one anonymous process-lifetime pending set with durable named parallel Architecture proposals:

- multiple independently identified change sets per project over one singular Accepted Architecture;
- private-Git persistence for exact base, structured proposed state, proposal Markdown, generation, candidate, and review binding;
- Accepted/proposal workspace selection without mixing snapshots;
- independent structured editing and exact review/acceptance through the existing candidate, validation, diff, and CAS authorities;
- proposals which remain exact and editable against their original base when another proposal advances Accepted; and
- browser, CLI, embedded skill, and MCP parity through explicit change-set IDs and generations.

Change Sets 1 adds neither review comments nor reconciliation. `docs/architecture-change-sets-v0.md` contains its product contract.

Architecture Change Sets 1 is complete at completion record `c28f817b8cbe2caa587b4312c1b14c87388b8d52`, with final implementation `5870941dc803ba5fd7c9a57d6b278223619e0471`.

## Architecture Reviews 1 — next stage

Add durable review submissions against one exact change-set revision. A review may contain comments anchored to:

- the whole proposal;
- a proposal-Markdown line or range;
- a Component;
- a Component Markdown line or range;
- a Diagram;
- one home/reference composition fact; or
- one exact Relationship fact occurrence.

A submission may carry the lightweight verdict **Comment**, **Approve**, or **Request changes**. These verdicts do not gate editing or acceptance in this stage. CLI and MCP can read reviews so agents can iterate the same change set. Comments remain bound to the exact reviewed base, candidate tree, and generation; later edits must not silently present an old comment as feedback on new content.

Reviews use their own private-Git ref namespace and retain the exact reviewed Change Set state through a Git parent link so later proposal mutation, application, discard, restart, and ordinary garbage collection cannot retarget or lose the feedback. Review submissions remain separate from `Review changes`: the latter still prepares the exact candidate/binding used for acceptance, while a submission records informational feedback on that already-bound state. `docs/architecture-reviews-v0.md` contains the approved exact contract, including permanent direct access to submitted reviews after proposal discard without inventing a discarded Change Set lifecycle.

## Architecture Reconciliation 1 — later utility stage

Add deliberate reconciliation for an out-of-date change set using three exact inputs retained by Change Sets 1:

- its original base;
- current Accepted; and
- its exact proposal candidate.

Reconciliation may automatically combine non-conflicting work and must surface real conflicts for explicit **Use Accepted**, **Use proposed**, or structured/manual resolution. It never silently overwrites Accepted or accepts the result. Conflict taxonomy, candidate lifecycle, review interaction, and durable representation require their own product decision.

## Phase 3 — Durable rich diagram editing — deferred until after the utility stages

After Change Sets 1, Reviews 1, and Reconciliation 1, add deliberately authored diagram presentation such as manual layout, sizing, routing and bend points, shapes, and any approved annotations. Durable named proposals will already keep valuable proposed work across process restarts; the separate product decisions for spatial editing, review, and reconciliation must still be made before adding canonical presentation fields.

This phase must preserve the distinction between Architecture semantic identity and diagram-presentation identity. It does not imply relationship domain IDs or a generic diagram-node model.

## Phase 4A — Additional semantic diagram kinds

Add other diagram kinds, such as a UML-style class diagram, only where a distinct semantic model is justified. Rendering and canvas infrastructure may be shared, but Architecture Components and UML Classes are not assumed to be the same domain object. A full UML schema is not approved by this roadmap.

## Phase 4B — Alternate renderers

Explore optional alternate presentations, including an isometric view, as renderers of approved rich diagram state rather than separate canonical Architecture representations.

Phases 4A and 4B are not ordered relative to each other. Either may follow Phase 3 according to demonstrated product value.

## Decisions deliberately deferred beyond the first Diagram slice

- Diagram deletion and general hierarchy lifecycle;
- multiple parents or reusable Diagram DAGs;
- multiple appearances of one Component inside one Diagram;
- multiple detail Diagrams per anchor;
- canonical layout, routing, shape, and annotation fields;
- Diagram-local presentation identity needed by later routing or repeated visual facts;
- interaction between durable change sets and substantial spatial editing;
- Diagram kinds and kind-specific semantic models;
- any renderer-specific persisted presentation.
