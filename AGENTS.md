# WorkBraid agent guide

**Authority:** the human, then the approved living contracts linked from [README](README.md), then the active implementation plan. Explicit human decisions supersede older instructions. Git history is the archive for completed plans; scratch notes are not authority.

Agents may plan and execute implementation within approved scope. Keep plans short: outcome, ownership, meaningful risks, checks, and remaining work. Do not require separate baseline/packet commits or repeated SHA approval ceremonies. Record the actual reviewed/integrated commit when it helps establish what was tested.

Meaningful product/domain decisions remain human: identity, portable schemas, lifecycle, authority, acceptance semantics, or a new user-facing capability require an approved decision. When one is missing, explain the concrete question, smallest options, trade-offs, and recommendation. Routine implementation choices and proportionate verification belong to the assigned agent. Do not change an approved contract's meaning without human authorization; keep documentation coherent when an approved change authorizes it.

## Coordination and execution

- The root is the human's contact and orchestrator. Use Herdr to assign, coordinate, review, and integrate work; root does not implement alongside workers. Follow the installed `herdr --skill` and use explicit agent/pane targets.
- Assign clear file/scope ownership. Use separate worktrees when code work needs isolation; avoid duplicate workers on one concern. A documentation owner may work on main while implementation stays isolated. Keep the human informed of outcomes and decisions, not terminal ceremony.
- Review proportionately. Use independent technical review for authority, storage, reconstruction, reconciliation, or substantial interaction changes. Inspect the complete corrected change before integration; a small documentation edit does not need a full product-gate replay.
- A previously accepted real workflow failing freezes feature expansion. Reproduce and fix the broken invariant through the product. Do not bypass it with manual Git, terminal, or provider actions and call the workflow complete.
- Green unit tests are not product acceptance. Relevant real application paths must work against their real authorities and survive restart. Human visual acceptance remains required for interaction changes covered by the active gate; technical review cannot substitute for it.
- Draft and maintain future plans as named proposals in the real WorkBraid self-project, not repository plan files. Keep only discovery links and the concise roadmap in the repository; do not duplicate the plan. Use the public CLI/UI/MCP after inspecting current store/proposal state, and read back exact Markdown and generation. A Markdown-only plan must not manufacture Architecture changes to obtain a review route.
- Human plan approval is an explicit orchestration decision identifying the proposal UUID and generation/text. Architecture Update accepts Architecture; submitted review verdicts are informational. Neither grants implementation authorization. Keep canonical contracts unchanged until the human resolves material domain/schema choices. Existing completed repository plans remain archived in Git history.
- Planning discovery: store `7329b076-50c4-4ac2-b63d-cb5cdb2a87fa`, [current Phase 3.2 design proposal](http://127.0.0.1:8080/projects/workbraid/proposals/bbc915b9-4e06-47c0-ad96-db5f8f166435) (`bbc915b9-4e06-47c0-ad96-db5f8f166435`, draft generation 2); inspect the catalog and proposal list for current state. Root continues to orchestrate through Herdr.
- Use WorkBraid for WorkBraid's real Architecture work through supported CLI/UI/MCP and one deliberately configured durable app-data authority. Inspect before changing, use named proposals and exact Review/Update, and preserve the human's acceptance boundary. Never use disposable failed-gate data or write private Git to bypass authoring. Do not initialize real data with the rejected partial-placement v3 binary. Dogfooding does not implement Planning or Agent Control inside Architecture.

## Invariants and checks

- Owning domains keep their own state and approval rules. A shared shell or agent does not merge them.
- One loopback Go process serves the built UI from the same origin. Bind a literal loopback IP. Browser, CLI and MCP are clients; the backend owns durable transitions.
- Preserve exact store/proposal/generation and S/B/A/P preconditions, ref CAS, one candidate constructor, immutable review parents, and exact historical tree reconstruction. Loading or review must not repair or rewrite supported history.
- Git uses the real executable with fixed args, no shell, hooks, pagers, signing or external diff execution. Tests use temporary real repositories; fakes cannot grant production capabilities or count as real-authority evidence.
- Parameterized asynchronous browser scenarios are separate runner-owned cases (for example `it.each`), not manual render/unmount/mock-restoration loops inside one test.
- Run checks appropriate to the change, escalating to real backend/browser/restart evidence for changed product paths. Repeat or broaden only for new changes, failures or unresolved risks.
- Architecture has no SQLite authority or event bus. Operational writes stay purposeful. Do not modify an external user project without explicit configuration.
- Show only current actions/state. Follow [UI direction](docs/ui-v0.md); no disabled future controls or implementation machinery in normal product flows.
- Keep packages concrete and small. No generic framework, ORM, repository layer, VCS interface, validation engine or speculative types. Follow the approved contract rather than inventing a parallel one.
- Use conventional scoped commits. Do not commit dependencies, build output, `*.db`, runtime fixtures or private stores. Inventory exact tracked cleanup targets; report uncertain untracked files instead of deleting them.
