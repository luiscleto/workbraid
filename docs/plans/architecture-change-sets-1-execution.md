# Architecture Change Sets 1 execution packet

Status: Complete

Target: durable named parallel Architecture proposals across browser, CLI, embedded skill, and MCP

Exact completed Agent Access 1 prerequisite and current repository head: `540724919165ea95c9ee3088aca084d91eb8e1c3`

Exact Agent Access 1 final implementation: `0a82c89d2a214896fe6b7020fe312cdd109beaeb`

Approved Change Sets baseline and roadmap: `c606822c360c4ef357449f515a67af6e2c21256b`

Packet-inclusive worker base: the exact clean commit produced by committing this Approved packet alone on that approved baseline; record it before dispatch

This is one cohesive Architecture utility increment. It replaces the single anonymous pending set; it does not create another Architecture authority or implement Reviews 1, Reconciliation 1, or rich-diagram Phase 3.

## Execution discipline

After human approval:

1. Verify approved baseline `c606822c360c4ef357449f515a67af6e2c21256b` is rooted at exact prerequisite `540724919165ea95c9ee3088aca084d91eb8e1c3`.
2. Commit this Approved packet alone on that baseline after `git diff --check`.
3. Record the resulting clean packet-inclusive commit as the one exact worker base.
4. Dispatch exactly one implementation worker from that base.
5. Send the entire worker-base-to-head range to one fresh independent reviewer who did not implement it.
6. Return bounded defects to the same worker; require fresh independent rereview of the corrected complete range.
7. Integrate only the exact approved range, verify its tree identity, and run ordinary checks once.
8. Run two fresh independent weak-agent black-box gates against one fresh project and the same running WorkBraid authority, then have a stronger independent verifier inspect canonical private-Git truth.
9. Run the small real human UI checkpoint and record explicit PASS/FAIL plus exact evidence in a separate completion commit.
10. Stop. Do not plan or implement Architecture Reviews 1, Architecture Reconciliation 1, or rich-diagram Phase 3.

The worker may use conventional commits grouped around durable state/authority, product workspace, and Agent Access v2, but all commits remain one worker, one reviewed range, one integration, and one product gate.

Any need for a different durable schema, proposal lifecycle, review authority, reconciliation rule, or client-specific pending state is a stop for human decision rather than an implementation improvisation.

## Exact worker brief

Before editing, verify the exact clean worker base and read completely:

- `AGENTS.md`;
- `docs/architecture-v0.md`;
- `docs/ui-v0.md`;
- `docs/architecture-post-gate-roadmap.md`;
- `docs/architecture-agent-access-v0.md`;
- `docs/architecture-change-sets-v0.md`;
- this packet;
- completed `docs/plans/architecture-agent-access-1-execution.md` for the running-authority, CLI, skill, MCP, error, and black-box evidence;
- completed `docs/plans/architecture-post-gate-phase-2.4-execution.md` for current project/private-Git and v2 authoring behavior;
- completed `docs/plans/architecture-post-gate-phase-1-execution.md` where immutable review capture and exact comparison remain relevant.

Do not edit approved baselines, historical completion records, or this packet. Executable tests and browser scenarios are living verification and must change where the one-pending behavior is deliberately superseded.

## Worker implementation scope

### 1. Closed durable state

Implement exactly the active/applied ref namespaces, envelope tree, metadata schema, concrete `changes.yaml`, proposal Markdown, parent rules, validation, and unavailable-state behavior approved in `docs/architecture-change-sets-v0.md`.

- Enumerate only the two owned subtrees `refs/workbraid/change-sets/active/` and `refs/workbraid/change-sets/applied/` using fixed direct Git arguments, then validate every owned UUID/ref/object/tree/blob before exposing a record. Ignore unrelated `refs/workbraid/*` namespaces without adding a namespace registry.
- Keep `refs/heads/accepted` the only Accepted Architecture authority. Never treat a change-set commit root as an Architecture tree; only its validated nested `architecture/` tree is a candidate.
- Serialize the current concrete Component and Diagram-composition pending facts without inventing generic operation kinds, fake Component changes, event history, or raw YAML patches.
- Preserve exact invalid Relationship target/label strings and every other structured value necessary for browser/CLI/MCP repair after restart.
- Reconstruct a valid record both from the exact nested candidate tree and by the single `ConstructCandidate` path over `changes.yaml`; require identical tree IDs.
- Derive validation, out-of-date state, projections, unified diff, and base snapshot rather than persisting redundant interpretations.
- Render `proposal.md` through the existing safe Markdown rules and preserve its exact UTF-8 text.
- Surface a malformed individual record as unavailable while retaining truthful access to Accepted and other valid records. Do not repair, skip silently, or infer missing data.
- Add no SQLite, worktrees, filesystem drafts, registry, branch-per-client model, or state-commit generation chain.

Keep the Git wrapper concrete. Adding the few fixed ref-list, symbolic object/tree, and reference-transaction commands this representation requires is expected; a generic VCS interface is not.

### 2. One authority and independent records

Replace the handler's singular pending pointer with a project-scoped collection keyed by stable change-set UUID while retaining one Manager, one Accepted snapshot, one state mutex, one candidate builder/validator, and one CAS publication path.

- Browser context is request/local UI selection. The backend has no process-wide current change set.
- CLI/MCP always supply a change-set ID for proposed work.
- Every mutation captures/checks the current store, record ref object, record generation, base, and accepted-authority state under the existing concrete lock.
- Write the new immutable envelope/state commit, CAS only that active ref, then publish the immutable record in memory.
- Mutating, renaming, or editing proposal Markdown in A increments A and invalidates A's review only. B is untouched.
- Review durably updates only A's review binding without incrementing A's generation.
- A rejected or raced mutation changes no ref, in-memory record, Accepted state, or other proposal.
- Opening another project or choosing Accepted/another proposal no longer requires deleting durable work. Existing dirty-browser editor protection remains local and exact.

Do not introduce per-change-set goroutines, locks, Managers, branches as client state, sessions, a transaction framework, command bus, or repository abstraction.

### 3. Creation, naming, proposal Markdown, and delete

Implement explicit **New changes** with optional custom name and the approved collision-safe generated name. Generate the stable UUID once on the server.

- Explicit empty creation persists generation `0`, exact current Accepted base, empty proposal Markdown, and a candidate equal to base.
- Explicit creation requires and atomically checks the caller's exact observed store UUID plus Accepted revision before generating identity/name or writing state.
- Beginning an ordinary structured browser edit from Accepted carries the browser's observed store UUID plus Accepted revision. Under the existing lock, verify both still identify current loaded Accepted, generate the ID/name, construct the requested mutation, create only one durable generation-`1` active ref, and select it in that browser. A stale or wrong-project observation must create nothing.
- Enforce the approved trimmed, non-empty, single-line active-name rule with Go's `strings.EqualFold` and no other Unicode normalization. Rename affects active records only and is generation-bound.
- Applied records retain their accepted names without reserving them. Allow an active record and multiple applied records to share a historical name; group/mark Applied in the UI and add shortened UUID context only when duplicate visible names require it.
- Proposal Markdown replacement is exact, generation-bound, safely rendered, and non-semantic.
- Whole-set discard/delete requires active ID and exact generation, confirms deliberately in the browser, and CAS-deletes only that ref.
- No partial discard, applied deletion, reopen, history, undo, labels, status, owner, or assignment appears.

### 4. Candidate-relative editing and invalid repair

Extend every existing structured Component, Relationship, Diagram, home, and reference operation to explicit change-set ID/generation.

- Resolve identities and eligibility from that record's complete candidate-relative concrete state and exact base.
- Keep all Component/Relationship fidelity, Diagram hierarchy, home/reference normalization, and source-order rules unchanged.
- Invalid work persists without an `architecture/` entry, survives full process restart, and remains repairable through every transport.
- Structured repair uses stored raw pending facts; it never parses or patches candidate YAML as a workaround.
- A valid proposal context exposes one snapshot-unified Diagram tree/map/index/documentation/composition/Relationship projection from its exact candidate.
- An invalid proposal exposes its exact facts, proposal Markdown, localized product-language guidance, and fix routes without manufacturing a partial projection or presenting base as proposed.

### 5. Workspace and product flow

Add one compact, obvious context selector for **Accepted**, named active proposals, and retained applied records.

- Accepted remains exact Accepted state.
- A valid active selection is visibly **Proposed: <name>** and switches all Architecture projections together; it is not an Accepted/proposal overlay.
- An active record whose base differs from valid current Accepted is conspicuously **Out of date with Accepted**, while remaining selectable, editable, and reviewable against its original base.
- Applied records are clearly read-only evidence and do not appear as ordinary changes in progress.
- **Changes in progress** becomes the selected active proposal task with name/rename, exact proposal Markdown, structured authoring, validation, Review, and delete.
- Review stays one **Before changes** / **With changes** surface, bound to that proposal. It never substitutes current Accepted for the exact base.
- Context changes preserve the established unsent-editor leave guard, neutral clear selection, safe Markdown, Diagram navigation, review docks, and responsive workbench structure.
- Reload may return to Accepted; persisting the selected context in a route is not part of this increment.

Do not add proposal overlays, proposal dashboards, status columns, assignments, filters, workflow boards, presence, or collaboration chrome.

### 6. Review and out-of-date behavior

Scope review to exact `(change-set ID, base commit, candidate tree, generation)` and persist only that binding in the record.

- Capture the immutable reviewed candidate/base/comparison under the same existing synchronization boundary and serialize outside it where safe.
- Recompute exact diff and projections after restart from the durable exact binding; reject a binding that does not match current record facts and tree.
- Editing Architecture facts, name, or proposal Markdown invalidates only that proposal's binding.
- An out-of-date proposal may be re-reviewed against its original base; the review remains clearly out of date and cannot be accepted.
- Relationship and Diagram comparison classifications remain the existing exact semantic rules. Proposal persistence does not create content, Relationship, or composition deltas.
- A proposal with no Architecture tree change is inspectable but cannot advance Accepted.

When A advances Accepted, do not touch B's ref or record. Refresh adopts an external valid Accepted change and reclassifies proposal bases without rewriting them. Missing, invalid, unsupported, or indeterminate Accepted authority retains existing truthful read-only/failure semantics; do not label uncertainty as ordinary staleness.

Do not implement merge, rebase, reconciliation, conflict choices, automatic proposal updating, or freeze out-of-date valid proposals.

### 7. Atomic acceptance and applied record

Preserve exact review, explicit pre-CAS observation, successor creation, and post-CAS publication. Extend acceptance with one fixed Git ref transaction:

- update `refs/heads/accepted` from exact base to successor;
- delete the exact active ref from its observed state commit;
- create the exact applied ref from zero to the new applied-record commit.

All three ref changes succeed or fail together. The applied commit/tree obeys the approved schema and records the exact accepted successor. Other active/applied refs do not participate.

The reference transaction remains beneath the existing application lock but Git CAS is the final cross-process authority. A failed transaction changes no ref. A valid applied record matching the submitted ID, original base, final generation/reviewed candidate tree, and applied revision is a durable success receipt. Lost-response recovery reports the change set already applied with truthful current Accepted context even if Accepted later advanced again; it never rebuilds, repeats, or misclassifies that acceptance as uncertain.

After success publish the already-validated successor, remove A from active editing, expose its applied record read-only, and reclassify other bases. Do not create a workflow engine or acceptance status field.

### 8. Agent Access v2 parity

Make an explicit incompatible protocol transition from `workbraid-agent-v1` to `workbraid-agent-v2`.

- Implement the exact change-set CLI/MCP table and explicit identity/generation preconditions in the approved contract.
- `architecture inspect` / `architecture_inspect` remains Accepted-only.
- Remove singular `changes inspect/review/discard`, global `pending_generation`, and old implicit mutation preconditions rather than mapping them to whichever proposal a browser selected.
- Every change-set list/inspect request takes exact store UUID. Every existing mutation takes exact store UUID, change-set UUID, and generation. Create takes exact store UUID plus exact current Accepted revision; update takes exact store/ID/base/tree/generation review binding.
- Return active/applied lifecycle, proposal Markdown, base, generation, validity/candidate, validation, out-of-date state, review, and applied revision as structured data.
- Implement the approved stable change-set error codes in CLI JSON and MCP structured results; retain existing domain/authority codes where still valid.
- Preserve exact raw invalid-Relationship selector repair parity inside the addressed proposal.
- Update the embedded `workbraid --skill`, help, examples, MCP schemas/descriptions, protocol handshake, and ordinary Agent Access scenarios together. Old v1 clients receive `incompatible_server`; no compatibility adapter or parallel legacy API remains.
- Keep `workbraid [--server <loopback-url>] mcp` a stateless stdio-to-loopback adapter and the CLI a thin loopback client. Neither may open private stores or cache selected proposals.

Do not add a generic RPC/operation framework, remote exposure, agent identity, sessions, auth, automatic acceptance, or raw Git/YAML tools.

## Worker acceptance criteria

The worker range is ready for independent review only when:

1. private Git durably reconstructs multiple active and applied change-set records with exact identity, name, base, generation, proposal text, concrete facts, candidate validity/tree, and review binding;
2. the closed envelope/ref/parent contracts reject malformed or conflicting records within the owned Change Sets subtree without harming Accepted, interpreting unrelated `refs/workbraid/*`, or choosing by enumeration order;
3. invalid structured proposals survive restart and remain repairable without a second parser or candidate builder;
4. one synchronized Manager authority serves browser, CLI, and MCP while every proposed operation addresses an explicit record;
5. mutation/review/discard/accept races yield one coherent ref and generation outcome, with no cross-change-set invalidation;
6. Accepted and valid/invalid/out-of-date/applied workspace contexts are truthful, snapshot-unified, and never overlaid;
7. successful acceptance atomically advances Accepted, consumes exactly A into one applied record, and leaves B exact;
8. B remains editable/reviewable from its original base and is explicitly blocked from acceptance until future reconciliation;
9. Agent Access v2 has full semantic parity, typed results/errors, embedded-skill accuracy, and no implicit current proposal;
10. process restart reconstructs Accepted and every retained proposal solely from approved private-Git refs/objects;
11. the implementation contains no SQLite/worktree/registry, proposal history/workflow, reconciliation/reviews, alternate authority, or Phase 3 work; and
12. tests use real Git and production paths where authority matters, keep asynchronous cases runner-owned, terminate every process, and leave a clean tree.

## Required automated evidence

### Private-Git representation and recovery

With real temporary bare stores and the production Manager/handler, prove:

- exact create/list/load/rename/proposal-edit/delete behavior, active-name `strings.EqualFold` collision suffixing, name reuse after acceptance, and duplicate applied-name disambiguation;
- exact active/applied ref names, state-commit parent rules, closed envelope paths/modes, metadata values, proposal bytes, and nested candidate tree IDs;
- an unrelated well-formed `refs/workbraid/other-test/...` ref is ignored, while malformed/unsupported state inside `refs/workbraid/change-sets/...` is reported;
- valid candidate reconstruction equality between persisted concrete facts and nested Architecture tree;
- invalid empty/malformed/unresolved Relationship rows persist without a candidate tree, survive restart, and remain editable/removable;
- malformed state, duplicate UUID across namespaces, conflicting active name, unknown envelope path, review mismatch, and candidate reconstruction mismatch are explicit and never selected by enumeration order;
- state commits do not chain generations and discarded old state is unreachable through WorkBraid refs;
- Accepted remains loadable and unchanged when one proposal record is unavailable.

### Independent change sets and concurrency

Prove with at least A and B over the same R0:

- mutations to A increment/invalidate only A and B's exact object/generation/review stay unchanged;
- simultaneous different-record operations serialize safely without a global selected proposal;
- same-record stale generation and ref-CAS races reject without mutation;
- review capture racing mutation or discard returns one coherent binding or invalidation;
- browser context switching changes no backend proposal state;
- an Accepted-start edit atomically creates and mutates one proposal while an explicitly created empty proposal begins at generation 0;
- explicit CLI/MCP creation atomically requires exact store UUID and exact Accepted revision;
- a stale Accepted-start request creates no object/ref/state;
- a late browser or agent create prepared against Project A cannot create a proposal in currently loaded Project B, even if its Accepted revision happens to match;
- project switching is no longer blocked by durable proposals, while unsent browser-local values still require the established choice.

### Review, acceptance, and out-of-date behavior

Using real Git objects and refs, prove:

- A and B review independently against R0 with exact distinct IDs/generations/candidate trees/diffs;
- proposal Markdown/name mutation invalidates only its own review;
- accepting A uses one reference transaction to advance Accepted, remove active A, and create applied A;
- injected transaction failure leaves all three refs unchanged and generated objects non-canonical;
- response-loss recovery identifies the applied result from its exact applied receipt without a second acceptance, including after Accepted has advanced beyond that receipt's `applied_revision`;
- B retains exact ref/object/base/facts/proposal/generation/candidate/review after A succeeds;
- B is marked out of date, may be edited and re-reviewed against R0, and exact update is rejected with `change_set_out_of_date` while Accepted remains R1;
- explicit external valid advancement plus Refresh reclassifies proposals without rewriting them;
- current-authority invalid/missing/indeterminate paths do not manufacture ordinary staleness or mutate proposal records.

### Browser product behavior

Use bounded frontend cases plus one built-UI production-browser scenario to prove:

- compact Accepted/active/applied context selection and unmistakable proposed/out-of-date/read-only treatment;
- snapshot-unified valid proposal projection and truthful invalid-proposal correction surface;
- New changes custom/default names, active-only collision handling, applied-name reuse/disambiguation, rename, exact proposal Markdown edit/render, implicit edit-start, and deliberate delete;
- every existing structured mutation operates in the selected proposal and preserves other proposals;
- review is exact base/with for the selected ID, including stale/out-of-date wording;
- accepting A moves it out of ordinary editing while B remains selectable/editable;
- switching proposals/projects does not delete durable work and still protects unsent local edits;
- safe Markdown resource blocking and existing Diagram/review workspace behavior do not regress.

Do not build a generic multi-proposal browser harness.

### CLI, skill, and MCP

Build the real binary and run it against one real loopback server to prove:

- `workbraid-agent-v2` negotiation and clean rejection of v1;
- every approved change-set command/tool, closed schema, ID/generation requirement, result, and typed error;
- `architecture inspect` remains Accepted-only and no client selection state exists;
- two independent clients can create/mutate/inspect/review distinct proposals without changing each other's context;
- exact proposal Markdown survives CLI literal/file input, MCP JSON, and restart;
- change-set creation requires both exact store UUID and Accepted revision, and a cross-project late create returns a typed error without state;
- all existing Component/Relationship/Diagram/home/reference operations retain parity inside an explicit proposal;
- exact review binding and update require the same proposal ID;
- out-of-date proposals remain editable/readable but unacceptable with a typed code;
- `workbraid --skill` is embedded, server-independent, stdout-clean, concise, and sufficient for the black-box workflow;
- MCP remains a stateless stdio proxy and neither client can access the private store locally.

Update living executable Agent Access tests which assert superseded one-pending behavior. Do not preserve an obsolete v1 path solely for historical records.

### Ordinary checks

After integration run once, from the clean integrated tree:

- `git diff --check` and `git status --short`;
- `go test ./... -count=1`;
- `go test -race ./... -count=1`;
- `go vet ./...`;
- `go mod verify`;
- `npm test` from `frontend/` using the repository command only;
- `npm run build` from `frontend/`; and
- the bounded built-browser Change Sets scenario.

Do not invoke raw Vitest or place repeated render/unmount/mock-restoration lifecycles inside manual loops. Stop and diagnose abnormal memory/process behavior rather than retrying it.

## Fresh independent reviewer brief

Review the exact worker-base-to-head range in two explicit concern groups, then issue one verdict for the whole range.

### Concern group A — durable authority and lifecycle

Verify:

- the exact closed ref/commit/tree/blob/parent schema and base reachability;
- typed invalid-state persistence rather than candidate-only loss or a second Architecture parser;
- one candidate constructor/validator and one Manager lock;
- per-record generations/reviews without process-wide selection;
- exact ref CAS and atomic three-ref acceptance transaction;
- applied lifecycle and durable-receipt response-loss classification even after later Accepted advancement;
- out-of-date B remains exact/editable/reviewable but cannot accept;
- malformed/conflicting record isolation; and
- absence of hidden history, migration, SQLite, worktrees, proposal framework, Reviews 1, or reconciliation.

### Concern group B — product and Agent Access

Verify:

- Accepted/proposal/invalid/out-of-date/applied workspace truthfulness and snapshot unity;
- implicit browser creation is one atomic mutation;
- proposal Markdown is exact, safe, durable, and non-semantic;
- every structured action addresses one explicit proposal in backend authority;
- protocol v2, CLI, skill, and MCP are consistent and discoverable;
- old v1 commands cannot silently target a proposal;
- black-box tasks require no undocumented names, endpoints, UI scraping, or private-Git edits; and
- no rich-diagram, review-comment, reconciliation, PM, or collaboration scope entered.

The reviewer must reproduce important authority claims with bounded real-Git tests, not infer them solely from unit mocks. Findings include confusing public wording or tool discovery gaps, not only code defects.

## Integration procedure

1. Require a clean worker tree and conventional commits based exactly on the recorded packet-inclusive worker base.
2. Record worker head and `git diff --check` result.
3. Obtain the fresh independent review of the complete range.
4. If corrections are needed, return them to the same worker and rereview the corrected complete range.
5. Integrate only the approved range; verify integrated tree identity equals reviewed worker tree.
6. Run the ordinary checks and bounded production-browser scenario once.
7. Stop every test/server/browser process before the black-box gates.
8. Run the two weak-agent gates, then stronger canonical verification.
9. Prepare the small human UI checkpoint. Do not begin Reviews 1.

## Two-agent black-box gate

Start one real built WorkBraid process with fresh application data. Through the product create one fresh project at Accepted R0. Provide no repository source, internal docs, packet, private-store path, endpoint names, or orchestrator command hints.

Use two fresh independent weak agents, preferably Luna, with the same project but separate working directories and tasks:

- **Agent A (CLI)** receives the business task, the running binary/server, and the instruction to begin with `workbraid --skill`. It creates named Change A, writes meaningful proposal Markdown, authors a coherent multi-Component/Relationship/Diagram proposal, and reviews it exactly.
- **Agent B (MCP)** receives a different coherent business task and a configured WorkBraid MCP connection. It discovers tools and creates named Change B from the same R0, writes its own proposal Markdown, authors its proposal, and reviews it exactly.

Launch both without step-by-step sequencing hints. Their operations may interleave through the one server. They pass only if each discovers the workflow, always uses its explicit change-set ID/generation, and neither corrupts, invalidates, or selects the other's proposal.

After both reviews, direct Agent A only to inspect its exact review and deliberately accept Change A. Then prove through product interfaces:

- Accepted advances exactly to R1;
- A is a read-only applied record, not ordinary editable changes;
- B retains exact ID/name/proposal Markdown/base R0/generation/candidate/review;
- B is visibly and structurally out of date;
- an attempted exact B acceptance returns the typed out-of-date classification and changes nothing;
- Agent B can still edit and re-review B against R0 without reconciliation.

Completely stop WorkBraid. Start a genuinely fresh process with the same application-data directory and prove Accepted R1, applied A, and active B reconstruct exactly. Record both agent transcripts/tool sequences, exact bindings, failures, final revisions, and the absence of human hints/intervention.

Confusing skill/tool language, missing reads, ambiguous typed recovery, reliance on source/private Git, or a need to tutor normal sequencing is a product FAIL. Do not patch verifier prompts to compensate.

## Strong independent canonical verification

After the weak-agent gate, an independent stronger verifier receives only the claimed project/store/change-set identities and the approved storage contract. It inspects bounded private Git facts and confirms:

- exact Accepted R0/R1 commits, parent, tree, and CAS result;
- exact active/applied refs and UUID uniqueness;
- state-commit parents, closed envelope trees/modes, metadata, proposal bytes, and candidate tree equality;
- A's applied successor and B's unchanged original base/generation/review before its later explicit edit;
- B's later generation still reconstructs from R0 and cannot be mistaken for R1;
- complete fresh-process projections match canonical Git; and
- no SQLite, worktree, source-project, hidden proposal registry, or alternate Architecture authority was created.

The verifier also checks CLI/MCP claims against the actual objects; black-box self-report alone is insufficient.

## Small real human checkpoint

Using the restarted real application and black-box project, the human only needs to:

1. open the project through its normal slug/catalog route;
2. select Accepted and confirm R1/the accepted Agent A Architecture;
3. select applied Change A and confirm its name/proposal Markdown/read-only treatment is understandable;
4. select active Change B and confirm its exact proposal projection, proposal Markdown, original base, and **Out of date with Accepted** treatment are clear;
5. make or observe one small structured edit to B and confirm A/Accepted do not change;
6. review B and confirm Before remains R0, With remains B, and acceptance is clearly unavailable until future reconciliation; and
7. give explicit PASS/FAIL on context selection, naming, proposal text, out-of-date meaning, and overall UI truthfulness.

The human does not repeat either agent workflow. Record exact project slug/store UUID, Accepted revision, A/B IDs, bases, generations, trees, review bindings, and explicit result.

## Explicit exclusions and stop

This packet does not authorize:

- durable review comments, anchors, submissions, reviewers, or verdicts;
- approval gates or workflow statuses;
- merge, rebase, reconciliation, conflict resolution, or automatic updating;
- history/revert/undo, partial discard, applied reopen/deletion, or multiple candidate revisions per change set;
- labels, owners, assignees, priorities, queues, filters, or dashboards;
- per-agent Managers, branches, pending copies, or selected proposal state;
- SQLite, worktrees, another registry/database, generic proposal/draft/operation/command/event framework;
- raw Git/YAML authoring, remote exposure, multi-user collaboration, permissions, or background agents;
- persisted/manual Diagram positions, layout, sizing, routes, shapes, annotations, graphical editing, UML, isometric rendering, or other rich-diagram Phase 3 work;
- Planning, Agent Control, or another vertical.

Change Sets 1 completes only after ordinary checks, both weak-agent gates, independent canonical verification, and explicit human PASS. Record completion separately, verify the tree is clean and all processes are stopped, then stop before Architecture Reviews 1.

## Execution result

Status: Complete — human checkpoint **PASS** on 2026-09-04

- Exact completed Agent Access 1 prerequisite: `540724919165ea95c9ee3088aca084d91eb8e1c3`.
- Approved Change Sets baseline and roadmap: `c606822c360c4ef357449f515a67af6e2c21256b`.
- Approved packet-inclusive worker base: `90866df1438f245da2c77a12d7093249a0fbc10d`.
- Final integrated implementation: `5870941dc803ba5fd7c9a57d6b278223619e0471`.
- The complete implementation range is `90866df1438f245da2c77a12d7093249a0fbc10d..5870941dc803ba5fd7c9a57d6b278223619e0471`.
- Change Sets 1 replaces the process-wide anonymous pending set with project-scoped durable active/applied UUID records in the approved private-Git namespaces. Each active proposal retains its own name, exact base, generation, proposal Markdown, concrete pending facts, candidate validity/tree, and review binding while sharing the one Manager, candidate constructor, validator, accepted ref, and confirmation/CAS/publication authority.
- Browser selection remains local view context. CLI and MCP use explicit store/change-set identities under `workbraid-agent-v2`; no process-wide agent selection, SQLite, second candidate interpretation, proposal database, worktree authority, review-comment model, or reconciliation behavior was introduced.
- Acceptance atomically advances `refs/heads/accepted`, consumes only the accepted active ref, and creates the applied receipt. Other proposals remain exact, durable, editable/reviewable against their original bases, and unacceptably out of date until the deferred Reconciliation stage.

### Independent review and automated evidence

- The implementation worker first produced `f50579ba99344fce595c008d6df0f86b8d477365`. Fresh independent review found bounded authority, validation, and unavailable-name handling defects; corrections at `62a24e7fee8df8660cadcf8fd3fa31c07b93aa57` and `bee9571c062824526dbef5f2f2ce62e88202a8b5` received fresh rereview. The durable authority/Agent Access implementation verdict was **PASS**.
- Review confirmed enumeration is restricted to `refs/workbraid/change-sets/active/*` and `refs/workbraid/change-sets/applied/*`; unrelated `refs/workbraid/*` namespaces are ignored; malformed owned records remain explicit; active names use the approved case-insensitive uniqueness rule; and an applied ref is durable acceptance evidence even after Accepted advances again.
- Review also confirmed exact store plus Accepted-revision creation preconditions, independent per-record generations/review bindings, invalid Relationship repair after restart, candidate reconstruction equality, out-of-date-but-editable behavior, three-ref atomic acceptance, and no second Architecture authority.
- Ordinary integrated checks passed: `git diff --check`; clean `git status --short`; `go test ./... -count=1`; `go test -race ./... -count=1`; `go vet ./...`; `go mod verify`; the repository `npm test`; `npm run build`; the bounded production-browser Change Sets scenario; and the Phase 2 production-browser regression. Tests used the repository-owned runner and one Playwright worker; no raw Vitest invocation or abnormal memory growth occurred.
- The final review-handoff and route corrections increased the ordinary frontend result to 58/58 passing tests. On exact final head, `go test ./... -count=1`, `npm test`, `npm run build`, `npm run test:change-sets`, and `npm run test:p2.4` all passed.

### Two-agent and canonical verification evidence

- Two fresh independent weak-agent verifiers used the same running WorkBraid authority and one fresh project without repository source, private-store access, undocumented endpoints, or orchestrator command tutoring. The CLI verifier began with the embedded `workbraid --skill`; the MCP verifier used ordinary tool discovery. They independently created, documented, authored, and reviewed two proposals from the same Accepted base without changing one another's identities, generations, facts, or reviews.
- Gate project: `Change Sets Agent Gate`; slug `change-sets-agent-gate`; store UUID `43f3c488-5a08-44b4-8bbc-ea469cfd4d05`; initial Accepted `26d363b21fd1a39ee642ddaf209ab1afb60f253a`; Accepted after Change A `d17790b00ec15e26928b561829f6968c1bcdc33f`.
- Applied Change A: UUID `f236b936-05c7-49ab-b2d9-e77655563235`; name `Request processing backbone`; original base `26d363b21fd1a39ee642ddaf209ab1afb60f253a`; final generation `9`; reviewed candidate tree `c913cbf5cd027b9174311fcfaa82e1c11157ba09`; applied revision `d17790b00ec15e26928b561829f6968c1bcdc33f`.
- Surviving Change B: UUID `52194895-e4d9-4b5e-8fb2-9d1f5cb8d271`; name `Observability model`; original base `26d363b21fd1a39ee642ddaf209ab1afb60f253a`; final generation `10`; reviewed candidate tree `82d9d01dc9523e20a6f559502065aed96d0de387`. After A's acceptance, B remained exact, durable, editable/reviewable and visibly out of date, while exact acceptance was rejected without changing Accepted.
- A complete process stop/restart reconstructed Accepted, applied A, active B, proposal Markdown, exact bases/generations/candidates/reviews, Components, Relationships, Diagrams, homes, references, and boundaries from the private Git authority.
- A stronger independent verifier checked the claimed accepted and Change Set refs/objects, closed envelope trees and modes, proposal bytes, candidate reconstruction, applied receipt, surviving out-of-date record, and fresh-process projections. Canonical verification result: **PASS**; no SQLite, hidden registry, source project, alternate Architecture projection, or second acceptance authority was found.

### UX correction cycle and human checkpoint

- The first human UI inspection stopped the gate because the initial context selector, proposal creation task, and active/applied proposal panes did not meet the approved drafting-workbench direction. At the human's direction, one dedicated UI correction worker owned the bounded presentation work in a separate Herdr workspace; this explicit correction cycle superseded the packet's ordinary single-worker handoff without changing domain/storage semantics.
- Grok reviewed the running production browser at 1440×900 and 1280×800 for layout, terminology, proposal context, and public-facing readability. The corrected selector, right-pane creation task, proposal-first presentation, applied/read-only treatment, and Markdown containment received **UX PASS**.
- Fresh technical review found bounded dirty-navigation, selector-keyboard, narrow-layout, and living-scenario issues. Corrections through `0241a34a26deabaa26b019cc1d6b25f0318b4e80` received fresh rereview and **PASS**.
- The final human request added the proposal Markdown directly to Review and introduced stable identity routes: `/projects/<slug>/proposals/<change-set-uuid>` for proposal context and `/projects/<slug>/proposals/<change-set-uuid>/review` for its exact bound review. CLI/MCP review results also return the review URL without adding domain authority.
- Independent review caught and corrected guarded-history, stale-route, accessibility, newly-created-proposal routing, and duplicate-history defects through `5870941dc803ba5fd7c9a57d6b278223619e0471`. Grok's final real-browser review and the fresh technical rereview both returned **PASS**.
- The human inspected Accepted, applied A, and active out-of-date B; confirmed proposal Markdown and exact review context were understandable; confirmed the selector, direct proposal/review URLs, reload, Back/Forward, and read-only/update treatment; and gave explicit final **PASS**.

Architecture Change Sets 1 is complete. Stop here; Architecture Reviews 1, Architecture Reconciliation 1, and rich-diagram Phase 3 remain unstarted.
