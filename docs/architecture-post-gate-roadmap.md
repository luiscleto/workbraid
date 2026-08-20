# Post-Gate Architecture roadmap

Status: Approved direction

Scope: Architecture as a standalone product after Gate 1

This roadmap records product sequencing and boundaries. It is not an implementation plan or a portable-store schema.

Architecture remains the only WorkBraid vertical in scope here. Agent Control is explored separately, and Planning remains deferred.

## Product principles

- Architecture Components and Relationships remain semantic Architecture facts with their existing identities and documentation.
- A future first-class Diagram owns composition and presentation rather than replacing those facts with a generic node/edge domain.
- The normal Architecture workspace continues to project exact accepted state.
- Candidate-only topology belongs in the deliberate Review changes task, derived from its exact immutable reviewed candidate.
- Exact canonical diff review and accepted-ref authority remain intact as richer visual review and diagram presentation are added.
- Optional renderers may present approved diagram state differently without creating another canonical Architecture representation.

## Phase 1 — Candidate-aware review workbench — complete

Improve the existing format-v1 review workflow without evolving the portable store:

- one Review changes surface combining candidate-aware visual review and the complete exact unified diff;
- an exact bound base/candidate toggle with snapshot-unified index, map, documentation, and relationships;
- stable-ID and relationship-multiset visual matching;
- visual distinction for added and changed topology and removed relationship facts;
- focused navigation between visual changes, review context, and canonical diff;
- deterministic non-canonical review layout;
- review-specific validation, fix, clear/deselect, and basic diff-readability polish.

The normal workspace map remains accepted-only. Persisted pending state, normal-map pending overlays, URL-backed restoration, semantic/rendered Markdown diff, syntax highlighting, and dedicated themed-scrollbar work remain deferred.

## Phase 2 — First-class Architecture Diagrams and nested navigation

Introduce canonical format-v2 Architecture Diagrams while preserving Components and Relationships as the semantic Architecture facts.

Approved domain direction:

- every Diagram has an immutable stable ID and mutable human-readable title;
- every v2 Architecture has exactly one manifest-identified root Diagram;
- Diagrams form a strict rooted tree: every non-root has one parent anchor, every Diagram is reachable, and cycles are invalid;
- every Component has exactly one home appearance and may additionally have reusable reference appearances in other Diagrams, at most once per Diagram;
- only a parent Diagram's home appearance may own its one optional detail-Diagram link;
- global Relationships crossing the active Diagram boundary use derived external references rather than copied facts or implicit membership;
- automatic layout remains disposable and non-canonical;
- valid v1 remains loadable without rewrite, while the first explicit Diagram operation creates one reviewed v1-to-v2 candidate preserving Component blobs, IDs, modes, paths, and Relationships exactly;
- candidate-aware Diagram review extends the existing exact base/candidate snapshots, unified diff, binding, confirmation, and accepted-ref CAS path.

The smallest useful product slice is root viewing, detail-Diagram creation and titling, home movement, reference appearance authoring, tree/breadcrumb/drill navigation, boundary-reference navigation, exact candidate review, deliberate acceptance, and restart reconstruction. Diagram deletion and general hierarchy lifecycle are not part of this slice.

## Phase 3 — Durable rich diagram editing

After Diagram identity and composition exist, add deliberately authored diagram presentation such as manual layout, sizing, routing and bend points, shapes, and any approved annotations. Decide project-scoped persisted pending-state semantics before substantial manual visual editing so valuable work is not limited to one backend process lifetime.

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
- project-scoped persisted pending state for substantial spatial editing;
- Diagram kinds and kind-specific semantic models;
- any renderer-specific persisted presentation.
