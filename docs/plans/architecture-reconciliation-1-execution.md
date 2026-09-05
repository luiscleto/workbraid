# Architecture Reconciliation 1 execution packet

Status: Approved

Target: ordinary detail-link reassignment and exact, deliberate reconciliation of a valid out-of-date active proposal onto current Accepted

Exact completed Reviews 1 prerequisite: `94787a8b05fdc502aedd07aae34b5d016b7106a3`

Exact Reviews 1 final implementation: `892f3254ad13b0fdabe02202e0070ac153f07bb8`

Approved Reconciliation baseline/roadmap SHA: `697bdb16cbd34b3dfa40c805d9e1398a72bcbb13`

Original packet-inclusive worker base: `802562170b56cd48eb8dc4bb1cec5436145ad0bc`

Approved ordinary Component serializer baseline amendment: `a332f64b4d04fbe001c21b56672373c49ca99fbd`, committed on the original worker base after the real-Git round-trip stop below

Corrected packet-inclusive worker base: the exact clean commit produced by committing this packet amendment alone on that serializer baseline amendment; record the full SHA before resuming the same worker

The human has approved this one implementation/review/gate cycle. Dispatch only from the exact packet-inclusive worker base. Do not begin rich-diagram Phase 3.

## 1. Execution discipline

One cohesive product increment, not separate backend/frontend tickets: ordinary reassignment makes the reconciliation result representable through the existing authoring model. Implement it first, then reconciliation, within one worker/review/integration/gate cycle. No intermediate human checkpoint is required.

Following the human approval:

1. Cross-read the contract, roadmap and packet. Commit the approved baseline plus roadmap on exact prerequisite `94787a8b05fdc502aedd07aae34b5d016b7106a3` after `git diff --check`; report its full SHA.
2. Replace this packet's pending baseline reference, mark it Approved, commit it alone, and report the exact clean packet-inclusive worker base.
3. Dispatch exactly one implementation worker from that base, using a separate Herdr workspace/worktree, not additional panes. Conventional implementation commits may distinguish ordinary reassignment from reconciliation, but remain one reviewed range.
4. A fresh independent reviewer examines the entire worker-base-to-final-head range, with explicit attention to authority/storage, semantics/fidelity and built-browser UX. Return bounded corrections to the same worker and rereview the complete corrected range.
5. Integrate only the approved exact range, check tree identity, run ordinary checks, and run fresh black-box agent/canonical gates below.
6. Invite the small real human UI checkpoint only after verification is ready. Human UX failure is a failed gate, not a non-blocking green-test result.
7. Record exact implementation/evidence and explicit human PASS in a separate completion record. Stop runtimes/test processes and close task-owned workspaces no longer needed; preserve evidence and unrelated work.
8. Stop after Reconciliation 1. Do not begin rich-diagram Phase 3 or another product stage.

Do not use Grok approval as a substitute for direct built-browser inspection. Use the latest accepted Reviews UX, not obsolete form/dropdown scenarios. If the human requests UI corrections, iterate the focused real UI first; align living tests with the approved final interaction rather than enforcing broken layout.

### Authorized serializer correction and same-worker continuation

The worker stopped before implementation when real candidate construction exposed an existing ordinary Description serializer failure. B has H1 `# API\n` and empty Description; externally accepted A has the semantically identical unterminated H1 `# API`; P supplies exact Description `\nBody\n`. Serializing the residual Description against A produced `# API\nBody\n`, which reparsed as `Body\n` and lost the first Description byte. Original proposal reconstruction succeeded and the reproduction changed no authoritative ref.

The human approved the narrow ordinary-serializer correction now recorded in the living Architecture baseline. Commit that clarification first, then this packet amendment, and fast-forward the same clean worker to the corrected packet-inclusive base. Do not dispatch a second implementation worker or create another increment. Fix and verify this broken ordinary round trip before continuing feature implementation. Historical completed records and the original baseline/dispatch provenance remain unchanged.

Whenever serialization preserves an unchanged ATX or Setext H1 block without a terminating line break and a non-empty exact Description must follow, append exactly one LF structural heading terminator, then append every Description byte unchanged. Empty Description inserts nothing; whitespace/newline-only Description is non-empty; an already-terminated H1 gains no additional separator. The inserted byte is always LF, without newline-style detection, and never replaces or normalizes Description LF/CRLF/leading whitespace. This is the shared ordinary Component serializer, not a Reconciliation branch, new storage field, raw-source override, second candidate representation, lossless-Markdown subsystem or newline-normalization framework.

## 2. Exact worker brief

Verify the clean exact worker base and read completely:

- `AGENTS.md`;
- `docs/architecture-v0.md`;
- `docs/ui-v0.md`;
- `docs/architecture-post-gate-roadmap.md`;
- `docs/architecture-agent-access-v0.md`;
- `docs/architecture-change-sets-v0.md`;
- `docs/architecture-reviews-v0.md`;
- approved `docs/architecture-reconciliation-v0.md`;
- this packet; and
- completed `docs/plans/architecture-reviews-1-execution.md`, including its final UX and exact-history gate results.

Apply the explicit supersessions in later approved contracts, not obsolete anonymous-pending restrictions in earlier baselines. Do not rewrite completed records or approved baselines during implementation. Executable tests and browser scenarios are living verification.

Inspect the concrete existing paths before editing:

- `internal/architecture/store.go`: snapshot facts, source-aware serialization, Diagram validation and `ConstructCandidate`;
- `internal/architecture/change_sets.go`: closed state encoding, exact reconstruct/tree equality, non-chained envelopes and ref writes;
- `internal/architecture/git.go`: fixed direct Git arguments and atomic ref transactions;
- `internal/architecture/reviews.go`: immutable reviewed-state parent reconstruction;
- `internal/web/handler.go` and `internal/web/agent.go`: the actual loaded-project/proposal mutex and ordinary shared operations;
- shared Agent Access wire types, CLI, embedded skill and MCP adapter; and
- the live proposal/review routes, contextual authoring controls, safe Markdown and leave guard.

One server owns all state. No CLI/MCP private-store access, second Manager, per-client proposal selection, or merge-only authority.

## 3. Implementation scope and acceptance criteria

### A. Ordinary detail-link reassignment

Implement the approved ordinary capability before using it for reconciliation:

- Add `detail_reassignments` as final Diagram-to-anchor values for base-existing non-root Diagrams. For a proposal-new Diagram, update its creation's final anchor instead.
- Write the approved operational `changes.yaml` version 2, with the original six required sequences and the new seventh sequence. Read version 1 into the same constructor with no reassignments, without rewriting historical state. Keep envelope, Review and portable Architecture versions unchanged.
- Preserve exact historical version-1 candidate reconstruction and repeated exact Review-changes state-object no-op. No second constructor or old/new Architecture interpretation is allowed.
- Resolve destinations from the complete candidate. Provide the concrete ordinary parent-options read and structured reassignment through browser, CLI and MCP with exact store/Change Set/generation preconditions.
- Preserve the child UUID/path/blob/mode/subtree and all unrelated Component/Diagram entries. Change only the old/new parent-owned links, plus independently authored changes.
- Enforce root prohibition, current-anchor no-op, unoccupied destination, valid home, no descendant cycle, unique parents and reachability. Reject invalid direct actions without persisting a detached state or losing other pending work.
- Apply final link/home assignments together in the constructor. Do not replay detach/attach commands or reject valid final swaps on transient ordering grounds.
- Display **Change parent component** in the selected Diagram's task context, distinct from moving a Component home and opening a Diagram. Use readable Component/home context and existing controls/spacing.
- Review the same Diagram moving between anchors as composition only. No false entity/content/Relationship delta; existing inline feedback remains historical.

### B. Three-way semantic calculation

Implement the exact units/equality rule in the approved contract, using existing immutable loaded domain facts. No raw file-level merge, YAML property merge, Markdown AST diff, generic graph or operation algebra.

- Components match by UUID. Title uses the structured plain-text projection; Description equality is the entire exact Markdown body, including whitespace/source differences. Added distinct identities combine. For both Components and Diagrams, independently added same-new-UUID objects coalesce only when complete supported semantic object/facts, including dependent context, match; divergence is unsupported with reason `replace_identity`, with no side/manual replacement choice.
- Relationships compare exact tuple multiplicities. Preserve label bytes, Unicode/newlines, direction, source and parallel facts. Same-result changes combine once; different count changes conflict. Do not heuristically match label/target edits as one Relationship.
- Merge Diagram titles, homes, reference bits and child anchors. Treat competing children, overlapping home/reference results and cross-change cycles as explicit structural conflicts, not automatically repaired topology.
- Both distinct competing children survive. The side preference identifies who stays; explicit final destinations are required for displaced children. No free anchor means unresolved/ineligible, not deletion or a fabricated parent.
- External absence versus proposed edits is not a bare boolean merge. Offer representable Accepted-side absence and explicitly resolve dependent proposed facts; mark restoration/deletion/source-lifecycle choices individually unsupported. Do not reject all external advancement or silently cascade-drop independent work.
- Adopt A's supported exact manifest/root context; do not add a project/root editor. Source path/mode changes are fidelity, not semantic identity.

Typed conflicts/choices use the contract's closed request-local locators. Manual values have existing domain types. Structural resolution may address only the identified involved facts, expanding context explicitly when a chosen occupied anchor displaces another child. Unknown/duplicate/contradictory choices fail rather than becoming arbitrary edits or last-writer-wins behavior.

### C. Residual authoring facts and fidelity

Build normal concrete facts relative to A, not a merge tree plus an escape hatch:

- existing Components use A source/path/mode and the minimum existing changed flags;
- genuine P-new objects retain IDs and generated source values, with deterministic collision-safe new paths only when required;
- homes/references/titles/new-detail facts and existing-detail reassignments describe net final composition;
- preserve surviving A appearance/Relationship order; append required new occurrences deterministically without turning order into identity;
- unchanged objects reuse exact A tree entries; reassignment never recursively rewrites children;
- construct with the one builder, validate with the same loader, compare semantic output to explicit choices, and enforce exact reconstruct/tree equality at write and restart;
- empty residual state reuses A tree, keeps the proposal/name/Markdown, increments generation once for the basis change and clears its review.

Do not introduce raw-source override fields, a generic inverse-diff engine, synthetic Component changes for composition, reconciliation-specific candidate storage, or a lossless-YAML subsystem.

### D. Prepare/apply authority

Prepare validates/captures S/B/A/P coherently and creates no ref, state commit, generation or review. Re-observe external authority before presenting a known-current usable result. Optional tentative resolutions invoke the same non-mutating calculation for Check choices. A preview cache is not authority and must not become a durable object.

Apply rechecks exact project, state object, generation, base, candidate and Accepted under the existing mutex, recomputes server-side, validates and prepares one ordinary new active state. Its parent is A; its review is absent. Publish only with a fixed transaction verifying Accepted A and CAS-updating that active ref from S. Other proposal/review refs and Accepted are untouched.

Handle late project switches, ordinary edits, rename, proposal text, first review preparation, discard/application and external Accepted movement. Repeated unchanged review preparation is still a durable no-op. Pre-publication failures preserve original state. After a known successful transaction, publish/return the authoritative resulting record; response serialization/delivery failure recovers from the real active ref without recomputing a mutation. A later retry with old S returns `change_set_state_mismatch` and truthful current Change Set context before a generation/base or not-required classification could conceal it. Tell clients to inspect, do not attribute an arbitrary later state to the previous Apply, and never replay blindly or add `reconciliation_uncertain`, a receipt/journal/history. Truthfully report subsequent advancement or indeterminate observation without rolling back a successful reconciliation.

Keep the existing three-ref acceptance boundary unchanged. Reconciliation never prepares/accepts implicitly and never takes review verdicts as policy.

### E. Browser and Agent Access

Extend the existing proposal workbench with one focused reconciliation task, preserving stable proposal/review navigation. Show automatic vs unresolved work; long text and controls must fit/scroll on a normal desktop. Simple side choices and typed editors must be obvious. Competing-child controls show both required final parents; unsupported choices explain what cannot be done.

Use the established unsent leave guard for local conflict values, no session persistence. A partial combination is not displayed as a valid map. A ready preview is one coherent projection. After apply show the new proposal and ordinary Review changes, with historical feedback clearly old and never retargeted.

Implement all four additive operations and exact schemas from the baseline: parent options, ordinary reassignment, reconciliation preview/check and apply. Preserve `workbraid-agent-v2`, global CLI flag ordering, JSON result/errors, exact file/stdin Markdown values, stateless MCP, server-owned eligibility and server-independent embedded skill. Stable IDs, never names, address mutations.

Return the complete conflict set and exact inputs, not prose-only summaries. Add no raw YAML/patch endpoint, force/accept-latest action, generic call API or hidden retry loop.

## 4. Bounded automated evidence

Use real temporary private Git stores, the real executable and production Manager/HTTP paths. No fake Git authority, fabricated capability or browser-owned candidate. Keep tests runner-owned and focused; do not replay all prior bootstrap/schema/Reviews matrices.

| Concern | Required bounded evidence |
| --- | --- |
| Ordinary reassignment | Base-existing child; pending-new child updates creation fact; candidate-only destination Component; destination home in root allowed; repeated moves normalize to one final fact; return-to-base residual removal; current-anchor no-op; root child/occupied/descendant/missing target rejection. |
| Atomic composition | Full subtree identities and exact child/unrelated blobs/modes retained; home plus anchor changes together; valid explicit joint anchor swap; invalid cycle/occupancy leaves original durable proposal exact. |
| Operational encoding | New seven-sequence version-2 round trip; closed-key/version validation; existing version-1 active/applied and review-retained historical states reconstruct exactly; repeated reviewed-state no-op still exact. |
| Scalar rules | A-only, P-only, equal-result and divergent Title/Description/Diagram title. Body whitespace/Markdown-source changes are exact Description differences; source-only H1 formatting with the same structured Title is not a Title difference. |
| Ordinary H1/Description round trip | Unterminated ATX H1 with `\nBody\n`, `Body\n`, whitespace/newline-only non-empty Description and exact empty Description; already-terminated H1 without an extra separator; Setext equivalents; Description LF/CRLF/leading whitespace preserved. Assert exact source bytes and reparsed intended Title/Description, including unchanged normalized Title. Prove ordinary Component Description editing uses this same serializer. Retain the original real-Git B/A/P failure as a regression: residual construction against unterminated A, reparse, durable write/load and exact reconstructed candidate-tree equality must pass without changing Accepted. No Reconciliation-only escape hatch. |
| Identity collisions | Distinct additions combine; identical same-new-UUID complete Component/Diagram facts coalesce using A fidelity; divergent content or dependent Relationship/home/reference/detail context returns `reconciliation_unsupported` / `replace_identity`, offers no side/manual resolution, and mutates no ref. No UUID remapping or dependency migration. |
| Relationships | Several differently labelled and identical parallel facts; base count 1/A count 2/P count 3 conflict; independent tuple deltas combine; same-result count not doubled; exact YAML-sensitive/Unicode/multiline labels and surviving order. |
| Composition interactions | Different home destinations; reference/home overlap; different anchors for same child; two independently created children on one anchor with explicit displaced-child assignment; no free anchor remains unresolved; independent changes creating a cycle yield localized structural context. |
| Unsupported branches | Externally deleted edited Component/Diagram: representable Accepted choice works where dependencies resolved, restoration choice rejected without mutation; no blanket external/ref-history restriction; no silent child discard. |
| Source/candidate fidelity | A-only YAML/frontmatter spelling/order, Diagram lexical formatting, path spelling and regular-file mode remain the serialization foundation, unless a semantic edit requires rewrite. These are not body-byte equivalence. Unmodified blobs reused; generated filename collision with distinct UUID deterministic; constructor output equals chosen semantics and stored tree; empty residual equals A tree. |
| Preview isolation | Ready, conflicting, unsupported and not-required previews leave all authoritative refs/generations/reviews/Accepted exact; repeated/check-choices calculation has no state/session refs. |
| Apply races | Pending mutation/rename/proposal edit; state-only first review binding write; discard/application; project switch; Accepted movement during calculation and immediately before Git transaction. Wrong input performs no active/Accepted update. |
| Apply response loss/retry | Let the real ref transaction succeed, lose the response, then retry with identical old S/A inputs. Exactly one generation/ref transition occurred; retry returns `change_set_state_mismatch` plus current context without another write. Known-success response recovery reads the real ref; later state is not guessed to be a receipt. |
| Review preservation | Repeated exact review preparation leaves S unchanged; successful reconciliation clears only current binding; old submitted-review refs/parents/anchors exact; new generation review/feedback works independently; informational verdicts do not affect reconciliation. |
| Restart and no history | Fresh Manager reconstructs residual state and exact tree; active parent is new A, not old S; old reviewed S remains reachable through Review parent; no reconciliation namespace/database/history chain. |
| Transport/UI parity | Shared typed outcomes; exact state/generation/Accepted guards; parent-options and reassignment; CLI JSON/stdin resolutions; MCP discovery/schema; embedded skill; leave guard, long text/choice ergonomics, historical comment isolation and post-apply ordinary review. |

One or a few built-browser runner-owned cases should exercise ordinary reassignment and the focused conflict task against real Go/Git, including correction, exact apply and restart. Prefer semantic assertions and real visible interactions over giant selector fixtures. Screenshots/inspection must cover long descriptions, two displaced-child selectors, empty/no-anchor feedback and a normal desktop viewport. Do not treat clipped/overlapping UI as a passed scenario.

Run focused tests during development. Ordinary integrated checks, from the correct worktree, are the repository's current `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go mod verify`, frontend `npm test`, `npm run build`, the bounded real-browser scenario(s), and `git diff --check`. Run memory-heavy/browser checks serially, inspect failures before repeating, and stop the exact task processes immediately if asked. No manual loop of repeated async render/unmount/mock restoration, runaway watcher, arbitrary retry until green, or new permanent performance policy.

## 5. Fresh independent reviewer brief

Review the complete exact implementation range, not only test summaries. Confirm:

1. There is one candidate/parser/validator interpretation. Residual state relative to A reconstructs the exact candidate, not a separately built merged tree. Version-1 retained Reviews still round-trip through this same path.
2. Reassignment is ordinary authoring through browser/CLI/MCP, with minimal final-state facts. Only parent-owned links change; no recursive child rewrites, detach history, root reassignment, hidden deletion or synthetic Component edits.
3. Three-way units honor structured Title versus byte-exact Description (including body whitespace), exact relationship counts, homes, references and anchors. Divergent same-new-UUID Component/Diagram collisions are unsupported even when differing dependent facts alone reveal the collision; no whole-object replacement choice or dependency migration. Identical complete facts coalesce once. Both competing children survive with explicit destinations; side labels do not hide incomplete choices.
4. Unsupported lifecycle handling is choice-scoped. Accepted-side absence is not denied solely because Proposed-side restoration is unavailable; neither missing identities nor dangling dependencies are silently reconstructed/dropped.
5. A source/path/mode fidelity and deterministic new-file collision allocation are exact. Independently inspect the ordinary H1 structural-terminator exception and rerun the original real-Git B/A/P reproduction: exact Description bytes survive ordinary editing and residual reconstruction, empty/already-terminated cases gain no separator, and ATX/Setext are covered. No Reconciliation-specific serializer, raw-source override, generic YAML/property merge, lossless-Markdown subsystem or serializer-driven semantic loss.
6. Preview is read-only at authoritative refs. Apply reobserves and atomically verifies A plus updates exact S; all race branches preserve truthful knowledge and other proposals/Reviews. Successful response-loss recovery reads real authority; a retry with old S is state-mismatch and cannot apply twice. No guessed receipt, reconciliation-uncertain classification, extra acceptance path, preview/session registry, merge commit or history parent.
7. The closed Agent Access values/errors and skill genuinely support discovery and recovery, including incomplete displaced-child decisions, without prose parsing/private Git.
8. Inspect the built browser at real desktop dimensions: focused task placement, choice clarity, no clipped controls, readable long text, useful no-anchor/unsupported feedback, unsent guard, proposal/Accepted navigation and exact historical comments. Do not approve behavior while deferring an unusable UI to the human.

Return blocking findings with precise paths and reproductions. Corrections return to the same worker; rereview the complete final range. A newly required lifecycle/schema/identity/authority decision is a human stop, not reviewer permission to improvise.

## 6. Integration and ordinary checks

Verify the worker started from the recorded docs-inclusive base and the reviewed head/tree are exact. Preserve unrelated work. Integrate non-destructively, compare the integrated tree to the reviewed tree, record the final implementation SHA and build identity, then run the ordinary checks and bounded browser evidence.

No feature work continues around a failing previously accepted workflow. If a correction changes the reviewed range, rereview it and rerun proportionate checks. Update living test selectors only to real approved controls, not to manufacture passing paths. Do not edit historical completion evidence.

Prepare the gate runtime with fresh task-owned application data and the built binary/UI; one literal-loopback process remains authority. No other task runtime is restarted/restored. Do not restore quarantined Herdr sessions. Record exact task-owned process/workspace identifiers for cleanup; preserve fixtures/evidence needed for the human.

## 7. Black-box parallel-proposal and resolution gate

### 7.1 Product-created starting Architecture

Use one fresh project. Build and deliberately accept a small R0 using public product operations, never component-bearing private-Git pre-seeding. Include connected Components, enough existing Diagrams for a home conflict, one unoccupied detail anchor, and a free alternate anchor. Record store/slug, R0 commit/tree and initial identities.

Prepare two independent business tasks for fresh weak agents, preferably Luna, in separate Herdr workspaces. Both receive the same project and product business constraints but no repository/source/private Git, design/packet, endpoint names or operation-order tutoring. CLI starts only with `workbraid --skill`; MCP gets ordinary configured discovery. Ensure both proposals are created from R0 before either is accepted, but do not sequence their edits to hide cross-proposal interference.

Agent A and B each create/name/document a proposal and review it. Tasks deliberately yield:

- independent additional Component/Diagram/Relationship work;
- an identical final change to a common value;
- different Title or Description on one Component;
- different multiplicities of the same exact Relationship fact;
- different home destinations or another composition conflict;
- preferably distinct detail children at the same free anchor, with an available alternate Component for explicit displacement resolution.

The orchestrator may state these business outcomes, not command/tool names or selectors. Record independent full transcripts, exact bases/generations/candidates/reviewed states, and any discovery failures. Confusing tools/skill are product findings; do not patch prompts to teach missing usage.

The black-box workflow uses normally generated distinct identities, not a whole-object replacement exercise. Divergent same-new-UUID collisions are bounded negative backend cases above; no gate prompt or expected result may teach or require resolving such a collision through a side choice, remapping or dependency rewriting.

### 7.2 Historical feedback and advancement

A separate fresh reviewer submits useful feedback on exact reviewed B through the other transport, including at least one affected Component/Markdown or composition anchor. Record the review UUID/ref/parent/binding. Deliberately accept reviewed A through the existing public flow to R1. B stays exact and out of date; its review and submitted feedback remain readable.

### 7.3 Fresh weak-agent reconciliation

Give a fresh weak agent the business task to update B onto Accepted and the intended final outcomes. Require different conflicts to retain Accepted and Proposed values respectively, plus at least one manual typed result (for example exact Description or an explicit multiplicity). If competing children are included, specify that both remain and give a meaningful intended alternate parent, without teaching commands/locators.

The agent must independently discover out-of-date state, preview, inspect automatic/conflicting results, make complete choices, check/correct when necessary, apply, inspect, and report the result. No repository/private Git or hints are allowed.

Record before ordinary acceptance:

- Accepted remains exactly R1;
- B's UUID/name/proposal Markdown are unchanged;
- B's base is R1, generation increased exactly once by apply, candidate is valid, and current review binding is absent;
- exact independently combined facts and explicit conflict/manual assignments;
- equality of normal reconstruction from new `changes.yaml` with its candidate tree;
- no out-of-date claim while Accepted still equals R1;
- old feedback remains attached to exact pre-reconciliation B.

Then the agent prepares a new exact Review changes, inspects the complete diff, and deliberately accepts its exact binding to R2 through the existing acceptance path. Reconciliation itself must not have performed any of these actions implicitly.

### 7.4 Strong independent canonical/restart verification

A stronger verifier, separate from the implementation and weak agents, checks the actual private Git:

- exact Accepted R0/R1/R2 parent/tree identities and Applied A/B receipts;
- the reconciled active state captured before acceptance: sole parent R1, not old B state; same metadata/Markdown, new generation, no review block;
- persisted operational facts reconstruct the exact candidate using the normal product loader/constructor;
- both retained child identities/paths/subtrees where displaced-child resolution was exercised;
- relationship multiplicities, home/reference/boundary derivation, independently merged content and source fidelity;
- old Review ref/commit parent, exact old candidate/base/generation/body/anchors remain unchanged and reachable without a live old active ref;
- no merge commit, reconciliation ref/session, registry, SQLite, worktree-based product state, hidden history or second Architecture representation.

Stop WorkBraid completely. Start a genuinely fresh process with the same application data, open `/projects/<slug>`, and reconstruct R2, Applied proposals/Markdown and historical feedback. Confirm exact public inspect results agree with private objects. If GC is exercised, restrict it to the explicitly task-owned throwaway store and record unchanged authoritative refs/reachable reviewed objects; never rely on reflogs to satisfy historical review retention.

## 8. Small real human UI checkpoint

Do not ask the human to reproduce the two-agent workflow. After the completed agent/canonical checks, prepare a fresh bounded proposal pair through the product for live pre-apply inspection, alongside the completed agent result. Do not interrupt the weak-agent workflow for human resolution tutoring or claim screenshots alone exercise the controls.

Ask the human to:

1. Open the out-of-date proposal and find **Reconcile with Accepted** without internal instruction jargon.
2. Inspect automatic vs conflicting work, a long text conflict, meaningful side/manual controls and one two-child displacement choice. Confirm it does not appear complete until the second parent is chosen, and no clipped/overlapping controls obscure the decision.
3. Try ordinary **Change parent component** in a small proposal context, confirming it is distinct from moving where a Component lives, and inspect its composition-only review.
4. Inspect the reconciled proposal, original proposal Markdown and exact old submitted Review. Verify earlier comments are not retargeted; navigation back to proposal/Accepted/review is available.
5. Confirm ordinary Review changes and deliberate Update remain the next acceptance flow, with no claim that reconciliation itself updated Accepted.
6. Give explicit PASS/FAIL for behavior, wording and ergonomics.

Use the established unsent guard; do not mutate the strong verifier's exact evidence accidentally while preparing inspection. Preserve and label fixture vs agent identities/revisions separately.

## 9. Completion and explicit stop

Completion requires ordinary checks, production-browser evidence, weak-agent discovery/resolution, strong canonical/restart verification and explicit human PASS. Record exact code/base/tree/build SHAs, commands/results, agent identities/transcripts, project/store/proposal/review IDs, R0/R1/R2, pre/post-apply S/generation/candidate, ref/tree/mode evidence, and any corrections. Do not claim completion based only on tests or a preview.

If any supported resolved result cannot round-trip through the one normal concrete authoring model, stop with that exact case. Do not introduce a new domain/lifecycle field or alternate candidate to finish the gate. Unsupported choices approved by the baseline remain truthful typed limitations, not implementation failures or permission to restore/delete.

Exclude deletion/restore, free-floating/multiply parented Diagrams, automatic anchors, arbitrary hierarchy editing, Relationship/appearance IDs, generic graph/merge/operation frameworks, raw YAML/file patches, reconciliation-only trees, persistent resolution sessions, proposal/conflict history, extra refs/registries, merge commits, background rebase, merge-and-accept, comment migration/resolution/verdict policy, SQLite/worktrees as product storage, remote collaboration/permissions, other verticals and rich-diagram Phase 3.

After explicit Reconciliation 1 PASS and completion recording, stop. No Phase 3 planning or implementation.
