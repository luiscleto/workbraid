# Post-Gate Architecture Phase 2 final continuation

Status: Approved

Completed P2.3 prerequisite: `17a5bdb06e31054371a5fa2aebabeaa1af52888f`

Historical continuation plan: `docs/plans/architecture-post-gate-phase-2-continuation.md`

Living baselines: `docs/architecture-v0.md`, `docs/ui-v0.md`, and `docs/architecture-post-gate-roadmap.md`

Target: one combined P2.4 implementation and final Phase 2 gate

## Superseding scope

The original Phase 2 plan, the earlier continuation plan, and completed P2.1/P2.2/P2.3 execution records remain unchanged historical evidence. P2.1 through P2.3 remain complete and are not reopened.

This note supersedes only the remaining P2.4 direction and every forward-looking continuation statement which still requires a source-project path, source-root association, SQLite catalog state, format-v1 loading, or **Set up diagrams** compatibility. Executable tests and browser scenarios are living verification and must follow the current baselines rather than preserve those removed alpha behaviors.

## Combined P2.4 boundary

P2.4 is **Slug-native projects, reusable references, and final Phase 2 gate**. It has two ordered implementation parts but one worker, review, integration, ordinary-check run, and human-gate cycle:

1. **Slug-native project model.** Replace folder-driven lookup with a private-store-derived project catalog; replace canonical `source_hint` with a stable WorkBraid-non-editable `slug`; create projects by name directly as native writable v2; support `/projects/<slug>` reload and authoritative external accepted-slug replacement on Refresh; remove source-root association, SQLite runtime/dependency, format-v1 loading, and **Set up diagrams**.
2. **Reusable Component references.** Add candidate-relative **Show component here** and **Stop showing here** through the existing one pending/candidate/review/CAS authority, including exact net normalization with repeated home moves.

Part A is established before Part B inside the worker range so reference authoring is built and tested against the final project model. Two conventional implementation commits are permitted for concern clarity, but there is no intermediate product checkpoint or independently shippable catalog increment.

The final Phase 2 gate starts from fresh application data, creates and reloads a slug-native project, authors and accepts the complete nested home/reference/boundary Architecture workflow, restarts, and reconstructs solely from private accepted Git. It includes no source-repository or SQLite-isolation ceremony because neither remains a product authority or runtime dependency.

## Preserved authority and scope

The breaking alpha simplification does not change accepted-ref authority, immutable store/Component/Diagram identities, exact candidate validation and diff, review binding, stale protection, compare-and-swap acceptance, Component/Relationship fidelity, safe rendering, accepted-only normal projections, or restart reconstruction.

P2.4 introduces no migration, compatibility layer, catalog database, generic catalog/graph/membership framework, persisted pending work, Diagram lifecycle, layout/routing/presentation state, Diagram kinds, another vertical, or Phase 3 implementation.

This note authorizes no implementation by itself. P2.4 begins only after the living baseline changes, this continuation, and the separate execution packet are human-approved and committed in the recorded order.
