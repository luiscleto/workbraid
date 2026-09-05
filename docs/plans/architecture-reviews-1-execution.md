# Architecture Reviews 1 execution packet

Status: Complete

Target: durable, exact-snapshot review submissions and anchored feedback across browser, CLI, embedded skill, and MCP

Exact completed Change Sets 1 prerequisite and current repository head: `c28f817b8cbe2caa587b4312c1b14c87388b8d52`

Exact Change Sets 1 final implementation: `5870941dc803ba5fd7c9a57d6b278223619e0471`

Approved Reviews baseline and roadmap: `b7eb0e206c3d40b6de8d58742a6314854078573f`

Packet-inclusive worker base: the exact clean commit produced by committing this Approved packet alone on the approved baseline; record it before dispatch

This is one cohesive Architecture utility increment. It adds immutable feedback on exact Change Set reviews; it does not change Change Set candidate/review/acceptance authority, implement reconciliation, or begin rich-diagram Phase 3.

## Execution discipline

Following the human approval recorded in this planning turn:

1. Commit `docs/architecture-reviews-v0.md` and the roadmap update together on exact prerequisite `c28f817b8cbe2caa587b4312c1b14c87388b8d52` after cross-reading and `git diff --check`.
2. Put that exact baseline SHA in this packet, mark it Approved, run `git diff --check`, and commit the packet alone.
3. Record the resulting clean packet-inclusive commit as the one exact worker base.
4. Dispatch exactly one implementation worker from that base.
5. Send the complete worker-base-to-head range to one fresh independent reviewer who did not implement it.
6. Return bounded defects to the same worker and require fresh rereview of the corrected complete range.
7. Integrate only the exact approved range, verify tree identity, and run the ordinary checks once.
8. Run the fresh weak-agent reviewer-to-author iteration gate, then have a stronger independent verifier inspect the exact private-Git records and garbage-collection reachability.
9. Run the small real human UI checkpoint and record explicit PASS/FAIL plus exact evidence in a separate completion commit.
10. Stop. Do not plan or implement Architecture Reconciliation 1 or rich-diagram Phase 3.

The worker may use conventional commits grouped around durable review authority, browser review UX, and Agent Access parity, but all commits remain one worker, one reviewed range, one integration, and one product gate.

Any need for a different ref/tree schema, mutable submissions, trusted identity, comment migration, Change Set lifecycle, review-based acceptance gating, or another candidate/snapshot interpretation is a stop for human decision rather than an implementation improvisation.

## Exact worker brief

Before editing, verify the exact clean worker base and read completely:

- `AGENTS.md`;
- `docs/architecture-v0.md`;
- `docs/ui-v0.md`;
- `docs/architecture-post-gate-roadmap.md`;
- `docs/architecture-agent-access-v0.md`;
- `docs/architecture-change-sets-v0.md`;
- approved `docs/architecture-reviews-v0.md`;
- this packet;
- completed `docs/plans/architecture-change-sets-1-execution.md` for exact proposal storage, generation, review, routes, Agent Access v2, and acceptance behavior;
- completed `docs/plans/architecture-agent-access-1-execution.md` for the shared process, CLI, skill, MCP, error, and black-box evidence; and
- completed Phase 1 review plan/execution record where immutable visual-review capture and snapshot unity remain relevant.

Do not edit approved baselines, historical completion records, or this packet. Executable scenarios are living verification and may be extended proportionately.

## Worker implementation scope

### 1. Closed review storage and exact reachability

Implement the one owned Reviews namespace and exact closed commit/tree/blob schemas from `docs/architecture-reviews-v0.md`.

- Enumerate only `refs/workbraid/reviews/<change-set-uuid>/<review-uuid>` using fixed direct Git arguments. Ignore other `refs/workbraid/*` namespaces without a registry.
- Require one review commit parent equal to metadata `reviewed_state`, and validate that historical parent under the non-applied/active Change Set state-commit schema with the existing parser, candidate constructor, candidate-tree equality check, and version-aware Architecture loader. Do not require a live active ref, current generation, or still-active Change Set to reconstruct it.
- Keep overall and comment Markdown exact, UTF-8, non-semantic, and mode `100644`.
- Validate the exact one-to-one ordered comment-ID/YAML/Markdown pairs and every closed metadata/anchor field.
- Surface a malformed individual review as unavailable without hiding Accepted, its Change Set, or other valid reviews.
- Add no review-state copy, nested Architecture copy, SQLite, worktree, registry, branch chain, or alternate proposal reader.

The review parent must retain the exact reviewed Change Set state, proposal Markdown, base, and candidate across later non-chained proposal generations and application. Verify that ordinary Git garbage collection cannot remove them while the review ref exists.

Keep the Git wrapper concrete. One fixed `update-ref --stdin` verify/create transaction and bounded review-ref/tree operations are expected; do not create a generic VCS or metadata persistence layer.

### 2. Immutable submission and one synchronized authority

Extend the one Manager/handler authority and existing state mutex with a project-scoped review collection. The in-memory collection is a cache of private-Git review refs, not a browser/session authority.

`Review changes` remains the only operation which validates and binds a proposal for comparison/acceptance. After its durable Change Set ref write, include the resulting exact state commit as `reviewed_state` in browser and agent review responses.

Make repeated exact review preparation a durable no-op. When the active record already stores the exact binding for its current base, candidate tree, and generation, browser **Review changes**, CLI `change-set review`, and MCP `change_set_review` must return its existing ref object as `reviewed_state` without writing another state commit or CAS-updating the active ref. Only a missing/different binding is durably established. Real Architecture mutation, rename, and proposal-Markdown edit retain their existing generation increment and review invalidation.

A review submission must carry exact store UUID, Change Set UUID, reviewed state commit, base revision, candidate tree, and generation. Under the existing concrete lock:

1. verify the exact loaded project/store;
2. verify the Change Set is active and its current ref equals `reviewed_state`;
3. verify its current durable review block exactly matches base/tree/generation and valid candidate;
4. validate verdict, author, bodies, and every anchor against the immutable Before/With/proposal sources;
5. generate review/comment UUIDs once and write the closed review objects/commit; and
6. atomically verify the same active ref and create the new review ref from zero in one Git transaction.

Only after the transaction succeeds may the server publish the immutable review result in memory. It must not write or increment the Change Set, invalidate its review, reconstruct another candidate, or touch Accepted.

Race submission against proposal mutation, acceptance, and discard. Exactly one coherent outcome may win: either the review ref binds the verified old state, or submission returns typed invalidation/not-allowed with no review ref. A generated but unreferenced object is non-canonical and collectable.

Submissions are active-only. Applied records and no-longer-active reviewed states remain readable but cannot receive new submissions. An out-of-date active proposal may receive feedback on its current exact original-base review while being reported as out of date.

An already-bound exact review may receive immutable feedback even when Accepted authority is known non-current or indeterminate. Return that authority knowledge truthfully and do not claim current/out-of-date when Accepted cannot be determined. This exception applies only to submission. Preparing a new review binding, mutating a Change Set, reconciliation, and acceptance retain their existing Accepted-authority requirements.

Implement the approved discarded-proposal rule exactly. Review refs survive active Change Set discard, remain exact/directly inspectable by IDs, and create neither a retained Change Set record nor a new lifecycle.

### 3. Exact typed anchors

Implement the seven closed anchor kinds without generic AST/location infrastructure:

- whole proposal;
- proposal Markdown lines;
- Component;
- Component Markdown lines;
- Diagram;
- home/reference/detail composition fact; and
- exact Relationship fact occurrence.

Use `before` / `with_changes` exactly where the baseline requires. Resolve all identities and lines from the immutable review parent, never from live Accepted or the current proposal generation.

- Proposal Markdown lines address the exact parent envelope's `proposal.md`.
- Component Markdown lines address the exact post-frontmatter Markdown bytes, including optional whitespace before the H1 and the H1/body through end of file, while excluding the closing delimiter and all frontmatter.
- Composition anchors resolve canonical Diagram facts by stable Diagram/Component identities and exact aspect; derived Lives in nodes are not facts.
- Relationship anchors resolve stable source/target UUID, exact label, and one-based identical-fact occurrence on the selected side.
- Line range validation uses the approved LF-delimited exact-byte rule, including empty source and trailing-newline behavior.
- Reject verdict `comment` with both an empty-after-trimming overall body and zero anchored comments as `invalid_request`; preserve exact accepted Markdown bytes. Permit verdict-only `approve` and `request_changes`.

Return a bounded comment/field location for invalid anchors. Reject the whole submission without creating a ref. Do not add Relationship/appearance/boundary IDs, rendered-DOM/AST anchors, fuzzy mapping, Git blame, or raw YAML/frontmatter review.

### 4. Review reconstruction and lifecycle truth

List/inspect each valid review from its immutable record and historical parent first, without consulting a live Change Set ref to validate that parent. Then derive current context without modifying either:

- whether the reviewed generation/binding still matches the active or applied Change Set's current/final generation;
- active, applied, or no-longer-active context;
- whether an active proposal is out of date with current Accepted; and
- current Accepted revision/authority knowledge separately from the bound Before snapshot.

When the active proposal advances, the old review remains exact and is identified as earlier feedback. It must never be overlaid on, floated to, or described as applying to the new generation. A new exact `Review changes` binding can receive a new independent submission.

Acceptance continues to use only the existing Change Set review binding and three-ref transaction. It neither reads verdicts nor changes review refs. Application must leave prior reviews readable against their exact parent. Subsequent Accepted advancement does not reinterpret them.

Do not add applicability inference, latest review, review status, approval aggregation, proposal history, reconciliation, or feedback-driven Architecture mutation.

### 5. Browser review workspace

Extend the existing proposal Review task rather than adding a dashboard.

- Keep `/projects/<slug>/proposals/<change-set-uuid>/review` as the current exact `Review changes` route.
- Add compact contextual comment actions and one review-composition area with author label, optional overall Markdown, accumulated comments, verdict, and deliberate **Submit review**.
- Keep unsent composition browser-local. Protect it with the established leave-without-keeping guard across proposal/project/route changes; submission is the only durable boundary.
- Provide exact line-addressable proposal and Component Markdown views when authoring or inspecting line comments, without exposing Component frontmatter or Diagram YAML.
- Keep map, Diagram tree, Component documentation, composition, Relationships, Before/With selection, and diff snapshot-unified.
- Reuse selected Component/Diagram/Relationship/composition context for product-semantic comment affordances rather than inventing a generic annotation canvas.
- Render all Markdown through the existing inert safe renderer with no automatic remote/local resource access.

Show submitted reviews contextually with clear verdict, descriptive author, time, exact generation, and current/earlier/applied/out-of-date language. Use `/projects/<slug>/proposals/<change-set-uuid>/reviews/<review-uuid>` for one exact immutable submitted review; reload and history navigation must preserve it. That route reconstructs its parent proposal/review snapshots and never substitutes the latest proposal.

Current-review render failure must not hide exact Markdown bodies, typed anchors, source ranges, or canonical diff. Keep annotations visually subordinate to Architecture review rather than creating a permanent management panel or SaaS review dashboard.

Under the approved discard rule, a known direct submitted-review URL remains readable and says only that its proposal is no longer active. Do not recreate the proposal in the context selector or add an archive page.

### 6. CLI, skill, MCP, and protocol parity

Retain `workbraid-agent-v2`; the operations are additive and do not change its envelope or existing precondition meanings.

Implement exactly:

- CLI `review-submission list`, `review-submission inspect`, and `review-submission submit`;
- MCP `review_submissions_list`, `review_submission_inspect`, and `review_submission_submit`;
- additive `reviewed_state` in `change-set review` / `change_set_review`; and
- the approved structured list/inspect/submit results and typed errors.

Every operation requires exact store and Change Set UUID; inspect also requires review UUID; submit requires the complete exact reviewed-state/base/tree/generation precondition. CLI structured comment input is the closed JSON array described by the baseline, not generic HTTP/API access. MCP uses equivalent typed anchor unions.

Update `--help`, embedded `workbraid --skill`, CLI JSON, MCP schemas/descriptions, examples, and living Agent Access tests together. The skill must teach agents to:

- prepare exact `change-set review` first;
- distinguish that acceptance binding from a durable review submission;
- inspect exact proposal/Architecture sources before anchoring;
- submit the unchanged exact binding/state;
- recognize earlier-generation and out-of-date feedback;
- iterate the Change Set deliberately rather than treating comments as mutations; and
- never inspect or modify private Git.

The CLI remains a thin loopback client and `workbraid [--server <loopback-url>] mcp` a stateless stdio-to-loopback adapter. Neither receives a local fallback, private-store reader, selected review, author identity service, or second Manager.

## Worker acceptance criteria

The worker range is ready for independent review only when:

1. every review ref/commit/tree/blob/parent exactly matches the closed baseline and malformed Reviews-owned records are isolated;
2. the parent link keeps the exact reviewed non-chained Change Set state and candidate reachable through ordinary Git GC;
3. the existing Change Set parser, candidate constructor, version-aware loader, review binding, exact diff, and acceptance path remain singular;
4. repeated exact Review changes across clients returns the same `reviewed_state` and leaves the active ref object unchanged;
5. submit atomically verifies the exact active state ref and create-only review ref, without changing proposal generation/review or Accepted and without requiring Accepted-currentness;
6. every approved anchor validates against only its exact side/source and round-trips without IDs or heuristic migration;
7. old review submissions remain exact after proposal iteration, application, Accepted advancement, restart, and the approved discard behavior;
8. browser current-review composition, immutable-review inspection, URLs, dirty guards, safe Markdown, and revision wording are coherent;
9. CLI, skill, MCP, and protocol v2 expose discoverable semantic parity and stable typed recovery;
10. weak reviewer/author agents can complete the iteration flow without source, private Git, UI scraping, or orchestrator tutoring;
11. stronger verification proves the exact claims and GC reachability from canonical private Git;
12. no review gating, mutable feedback, reconciliation, generic review framework, authentication, or Phase 3 state entered; and
13. all tests use production authorities where relevant, keep asynchronous browser cases runner-owned, terminate processes, and leave a clean tree.

## Required automated evidence

### Private-Git representation and recovery

With real temporary private stores and the production Manager/handler, prove:

- exact review ref path, UUID/metadata agreement, one-parent commit, closed tree paths/modes, ordered comment pairing, exact Markdown bytes, and exact reviewed binding;
- review parent equals the durable review-bound active-form state object and validates through the existing Change Set reconstruction path without requiring a live/current Change Set ref;
- multiple independent submissions do not chain or mutate one another;
- malformed path UUID, duplicate review UUID, wrong parent, unknown tree/blob path, metadata mismatch, unsupported schema/verdict/anchor, missing pair, and candidate/binding mismatch yield explicit unavailable reviews without harming Change Sets or Accepted;
- unrelated refs under Change Sets, reconciliation-test, and other WorkBraid namespaces are ignored by Reviews enumeration;
- after proposal generation advances and old Change Set objects would otherwise be unreachable, `git gc --prune=now` retains and reloads the exact reviewed state through the review parent;
- application leaves review refs and parents exact; and
- under the approved discard rule, deleting the active ref preserves the review ref/parent and direct list/inspect while creating no discarded Change Set record.

Keep the corruption matrix bounded to Reviews-specific contracts; do not replay the entire Change Set/Architecture format matrix.

### Submission synchronization and failure boundaries

Prove:

- repeated exact Review changes from two browser/CLI/MCP clients leaves the active ref object unchanged, returns the same `reviewed_state`, and does not invalidate the first caller's later submission;
- a successful submit requires exact store, active Change Set, reviewed state object, base, candidate tree, and generation, but not Accepted-currentness;
- the review ref transaction verifies the active ref and creates the review ref atomically;
- submit changes no Change Set ref/object/generation/review, candidate, Accepted, or other submission;
- submit racing a proposal/proposal-Markdown mutation returns either one review of the exact old state or typed invalidation, never mixed content;
- submit racing acceptance or discard cannot attach to a state after it ceases to be active;
- a stale/wrong-project request creates no review object reachable from WorkBraid refs;
- invalid anchors or malformed bodies create no ref;
- an out-of-date active proposal may be reviewed/submitted against its exact original base and reports out-of-date truthfully;
- an already-bound exact reviewed state accepts feedback while Accepted observation is indeterminate, changes neither Change Set nor Accepted, reports indeterminate context, and makes no current/out-of-date claim;
- known-non-current Accepted likewise does not block an otherwise exact submission and is reported truthfully;
- applied or missing active Change Sets reject new submission while retained review reads remain available; and
- completely empty `comment` returns `invalid_request` with no ref, while verdict-only `approve` and verdict-only `request_changes` each create a valid review.

### Anchor fidelity and revision truth

Use one nontrivial review with duplicate Relationship facts and different Before/With content to prove:

- whole-proposal and proposal-Markdown anchors;
- Component and Component-Markdown anchors on both sides, including candidate-only and old Markdown ranges;
- Diagram anchors on both sides;
- home, reference, and detail composition anchors;
- exact added and removed Relationship occurrences, including multiplicity and exact Unicode/whitespace labels;
- LF/CRLF, trailing newline, empty source, one-line, and multi-line range rules;
- rejection of nonexistent entity, wrong side, out-of-range line, nonexistent composition aspect/detail target, and excess Relationship occurrence;
- title/Diagram/membership/Relationship edits in a later generation do not alter or retarget old anchors; and
- review inspection reconstructs exact old proposal Markdown, snapshots, comparison, and diff after restart.

### Browser product behavior

Use bounded frontend cases plus one real built-UI scenario to prove:

- current Review route has compact overall/anchored/verdict composition without obscuring the established review workbench;
- contextual anchors target the visible exact snapshot side and exact source line selections;
- local comment editing/removal before submission and dirty-navigation protection change no backend review state;
- successful submission clears local composition and exposes the immutable result;
- submitted-review route reload/Back/Forward preserve exact IDs and parent revision;
- earlier-generation feedback is visibly historical and appears only on its exact reviewed snapshot, not current proposal content;
- applied and out-of-date wording is truthful and verdicts do not disable editing/acceptance;
- the direct retained-review route remains truthful after proposal discard under the approved rule; and
- safe Markdown, canonical diff fallback, proposal review URLs, Diagram navigation, and narrow workspace structure do not regress.

Do not build a generic comment-editor, route, or visual-review harness.

### CLI, skill, and MCP

Build the real binary and use one real loopback server to prove:

- unchanged `workbraid-agent-v2` negotiation;
- exact command/tool names, JSON/MCP schemas, result parity, state preconditions, and all new typed errors;
- additive `reviewed_state` is the exact current review-bound Change Set ref object;
- CLI exact Markdown literal/file and structured JSON-comment input round-trip;
- MCP typed anchor unions reject kind-incompatible fields;
- list/inspect return exact bodies/anchors/snapshots/currentness/lifecycle/out-of-date context and stable review URL;
- two clients cannot submit to a silently changed generation;
- `--skill` is embedded, stdout-clean, server-independent, concise, and sufficient for the black-box workflow; and
- neither transport reads private Git, selects a process-wide review, or converts feedback into mutation/acceptance.

### Ordinary checks

After integration run once, from the clean integrated tree:

- `git diff --check` and `git status --short`;
- `go test ./... -count=1`;
- `go test -race ./... -count=1`;
- `go vet ./...`;
- `go mod verify`;
- `npm test` from `frontend/` using the repository command only;
- `npm run build` from `frontend/`; and
- the bounded built-browser Reviews scenario plus still-relevant Change Sets/Agent Access regression scenarios.

Do not invoke raw Vitest or put repeated render/unmount/mock-restoration lifecycles inside one manual loop. Stop and diagnose abnormal memory/process behavior rather than retrying it.

## Fresh independent reviewer brief

Review the entire worker-base-to-head range and issue one explicit verdict. Inspect these concerns separately:

### Durable authority and reachability

- Review refs are confined to their owned namespace and every commit/tree/blob/parent follows the closed schema.
- The sole parent is the exact review-bound Change Set state and actually prevents GC loss.
- Submission uses one atomic active-ref verify plus review-ref create and cannot race onto another generation/lifecycle.
- Loading the parent reuses the one Change Set/Architecture reconstruction authority.
- Reviews do not alter Accepted, Change Set generation/review, acceptance, or unrelated refs.
- Historical review loading validates its parent independently of any live active/applied Change Set ref, and immutable feedback remains exact across edit, application, discard, and restart.
- Repeated exact Review changes is a durable no-op which cannot invalidate another reviewer's `reviewed_state`.
- Exact submission remains available under known-non-current/indeterminate Accepted context without relaxing review preparation, mutation, reconciliation, or acceptance.
- Malformed review isolation cannot hide or corrupt Accepted or a valid proposal.

### Anchors and revision truth

- Every field/kind/side/range resolves against only the immutable parent source.
- Proposal and Component Markdown bytes/line rules are exact and frontmatter/YAML stays out of normal review.
- Composition and Relationship anchors add no identity and derived boundaries are not persisted facts.
- Old comments never float or render against current proposal content.
- Currentness, applied, out-of-date, no-longer-active, and Accepted context are derived and worded honestly.

### Product and Agent Access

- Review submission is visibly separate from `Review changes` and never appears to approve/update Architecture.
- Composer/durable-review routes fit the drafting workbench, protect unsent values, and keep exact evidence reachable.
- CLI/MCP/skill operations are discoverable, equivalent, and retain protocol v2 without semantic ambiguity.
- Public language avoids Git/ref/candidate jargon outside bounded technical details.
- No auth, workflow, aggregation, reconciliation, generic framework, or Phase 3 work entered.

The reviewer must reproduce critical ref/GC/race claims with bounded real-Git evidence rather than infer them only from unit mocks. Confusing comment affordances, source-line behavior, or agent instructions are product findings.

## Integration procedure

1. Require a clean worker tree and conventional commits based exactly on the packet-inclusive worker base.
2. Record worker head and `git diff --check` result.
3. Obtain the fresh independent review of the complete range.
4. Return bounded corrections to the same worker and rereview the corrected complete range.
5. Integrate only the approved range and verify integrated tree identity equals the reviewed worker tree.
6. Run ordinary checks and the bounded production-browser Reviews scenario once.
7. Stop every test/server/browser process before agent gates.
8. Run the weak-agent iteration gate, then stronger canonical verification.
9. Prepare the small human checkpoint. Do not begin Reconciliation or Phase 3.

## Weak-agent reviewer-to-author iteration gate

Start one real built WorkBraid process with fresh application data and create one fresh native-v2 project. Give agents no repository source, design docs, packet, private-store path, endpoint names, or step-by-step orchestrator hints.

1. A fresh weak author agent, preferably Luna, receives a coherent Architecture proposal task through one Agent Access transport. If CLI, it begins only with `workbraid --skill`. It creates/names a Change Set, writes meaningful proposal Markdown, authors a nontrivial Component/Relationship/Diagram change, and prepares exact `Review changes` without accepting.
2. A separate fresh weak reviewer agent uses the other transport and ordinary tool/skill discovery. It receives only a business review request. It discovers the Change Set/current review, inspects exact Before/With/proposal/diff, and submits `request_changes` with an overall Markdown note, one Component-Markdown anchored comment, and one Diagram-composition or Relationship anchored comment.
3. A fresh weak author agent discovers and reads that submission, understands its exact source/semantic anchors, and makes an appropriate structured proposal iteration. It must not use source, private Git, raw YAML, undocumented endpoints, or human command tutoring.
4. Prove the first submission still reconstructs exact prior proposal generation and anchors; it is clearly earlier feedback, no comment moved, the current proposal has a new generation/candidate, and Accepted remains untouched.
5. Prepare a new exact `Review changes` binding. A weak reviewer inspects it and submits a separate `approve` review against that exact generation. Both submissions remain independently durable; approval still does not accept Architecture.
6. Completely stop WorkBraid. Start a genuinely fresh process with the same application data and prove the Change Set current proposal/review plus both submitted review records, bodies, anchors, parent states, and revision truth reconstruct exactly.

Run the author and reviewer roles in genuinely separate fresh contexts. Record their prompts, transcript/tool sequence, exact IDs/bindings/state commits, typed failures, and final currentness results. A need to teach command/tool names, explain anchor encoding, scrape the UI, edit private Git, or weaken exact submission is a product FAIL; fix product discoverability rather than verifier prompts.

## Strong independent canonical verification

After the weak-agent gate, an independent stronger verifier receives only the claimed project/store/change-set/review identities and approved storage contract. It confirms:

- exact Accepted ref stayed unchanged;
- current active Change Set ref/object/generation/candidate/review matches the author's final claim;
- both exact Reviews-owned refs, review commit parents, closed trees/modes, metadata, Markdown bytes, comment pairs, anchors, and bindings;
- the request-changes parent is the exact earlier Change Set state and the approve parent is the exact later state;
- neither review commit/state appears inside Accepted Architecture or mutates the Change Set ref;
- application-lifecycle behavior with a bounded separate fixture leaves reviews exact;
- `git gc --prune=now` preserves and reloads both reviewed parent states through their review refs; and
- a full process restart returns the same structured browser/CLI/MCP facts.

The verifier also confirms there is no SQLite, review database, second candidate builder, hidden review branch chain, Relationship/appearance ID, or review acceptance gate. Black-box agent claims alone are insufficient.

## Small real human checkpoint

Using the restarted black-box project, the human only needs to:

1. open the exact proposal Review route and confirm the current proposal/binding is understandable;
2. open the earlier `request_changes` submission and inspect its overall note plus Component-Markdown and composition/Relationship comments on the exact old Before/With sources;
3. return to the current generation and confirm old annotations are visibly earlier feedback and are not overlaid as current;
4. inspect the later `approve` submission and confirm its independent exact binding;
5. optionally compose and submit one small review/comment in the real UI, then confirm an agent can read its exact anchor;
6. confirm verdict wording does not imply an acceptance gate and Accepted remains unchanged; and
7. give explicit PASS/FAIL on anchor clarity, revision truthfulness, authoring ergonomics, routes, and public language.

The human does not reproduce the agent workflow. Record exact project/store/Change Set/review/comment IDs, both reviewed state commits and bindings, current generation, Accepted revision, and explicit result.

## Explicit exclusions and stop

This packet does not authorize:

- changes to the meaning of `Review changes`, Change Set lifecycle, Accepted authority, exact update, or the three-ref acceptance transaction;
- review-based acceptance gates, required approvals/reviewers, counts, aggregate status, permissions, or trusted identity;
- mutable/deletable submissions, persisted review drafts, threads, replies, resolution, reactions, notifications, latest/superseded state, or applicability inference;
- fuzzy/semantic anchor migration, AST/DOM/Git-blame anchors, Relationship/appearance/boundary IDs, or raw Git/YAML review tools;
- reconciliation, merge, rebase, conflict handling, proposal base updates, or proposal history beyond reviewed parent reachability;
- SQLite, worktrees, another database/registry, review branches, generic metadata/review/workflow/comment/command/event framework;
- remote exposure, multi-user collaboration, autonomous agents, Herdr/Agent Control integration, Planning, or another vertical; or
- persisted/manual Diagram layout, routing, bend points, shapes, annotations, graphical editing, Diagram kinds, UML, isometric rendering, or rich-diagram Phase 3 work.

Reviews 1 completes only after ordinary checks, the weak-agent iteration gate, independent canonical/GC verification, and explicit human PASS. Record completion separately, verify the tree is clean and every runtime/test process is stopped, then stop before Architecture Reconciliation 1.

## Execution result

Status: Complete — final human checkpoint **PASS** on 2026-09-05.

### Exact provenance

- Completed Change Sets 1 prerequisite: `c28f817b8cbe2caa587b4312c1b14c87388b8d52`.
- Approved Reviews baseline and roadmap: `b7eb0e206c3d40b6de8d58742a6314854078573f`.
- Approved packet-inclusive worker base: `cc7b93596d3f6e3de724ae702b61105eb8e2d5bd`.
- Initial implementation: `9a471c308f91a7f62e2e8793dd73376d91f9890d`.
- Exact-context correction: `ee17a9555908196359c9f48aade00215a3081021`.
- Contextual UI/navigation corrections: `42068dfa0d79655c1c06ab4c75be4e8aab5d4004` and `bed66837d90bae4c8bcdce4ca32f12e0442a1231`.
- Final human-approved UI correction: `7e1bf045066ed33da5dfff2a8c7016253faaa89e`.
- Final integrated implementation, including the living browser-scenario update: `892f3254ad13b0fdabe02202e0070ac153f07bb8`; tree `a6a88f5a90e8ea0b9100bffb955c5a1f1852ab65`.
- Complete implementation range: `cc7b93596d3f6e3de724ae702b61105eb8e2d5bd..892f3254ad13b0fdabe02202e0070ac153f07bb8`.

Review submissions use the approved Reviews-owned refs and one-parent records. They preserve the existing Change Set constructor, validator, exact review preparation, and acceptance authority. Informational verdicts do not gate acceptance. No reconciliation, additional Architecture interpretation, SQLite, or rich-diagram state was added.

### Independent review and corrections

- The implementation and bounded authority/context corrections received independent technical review before the agent and human gates. Review covered exact parent reconstruction without live active-ref dependence, GC reachability, exact submission preconditions, repeated-review no-op behavior, anchor fidelity, and CLI/MCP parity.
- Initial human inspection passed behavior but rejected review-authoring ergonomics. That was a failed human UX gate, not completion. At the human's direction, the correction cycle addressed review navigation, contextual comment creation, source-line selection, and map annotations. Earlier Grok approval did not override those findings; Grok was not used for the final correction review.
- The paused worker's saved frontend correction was continued by the orchestrator after the worker session was no longer available. No new Architecture authority or implementation increment was dispatched. A separate Herdr workspace housed the fresh technical reviewer.
- Final independent frontend reviewer `wb_reviews_inline_check` (Herdr session `01a06ecf-11db-7e83-823c-bb2a7db7be6b`) returned **PASS** after bounded correction/rereview. Findings fixed shared comment-edit coordination, range-only dirty detection, explicit discard/reopen resetting, live pinned-note membership, cross-Diagram reopening, and exact Before/With note isolation. Map and summary editing now share one browser-local active comment editor and save/remove path.
- The final UI presents independently openable map notes, selected-item add/comment markers, Diagram comment affordances even with no existing feedback, wrapped exact logical source lines, clear actions, and guarded unsent edits. Component details do not contain unrelated Diagram/placement authoring controls. Long map notes scroll internally. Historical annotations remain on their exact reviewed side.
- The final closeout commit updates only the executable browser scenario to those approved controls and wording. It changes no product code, storage contract, or historical record.

### Automated and production-browser evidence

- Ordinary Go checks passed: `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, and `go mod verify`. All four packages passed normal/race tests and all modules verified.
- Final repository `npm test`: **80/80 tests**, four files. `npm run build` and `git diff --check`: **PASS**. The existing bundle-size advisory remains non-gating. Browser tests use runner-owned cases; no manual repeated render/unmount lifecycle or unbounded test retry was introduced.
- Real built-browser checks against the existing Go/Git gate fixture exercised empty-Diagram commenting, browser-local Component and Relationship notes, multiple open notes while panning, unchanged map instance/pan/zoom during commenting, wrapped source lines, and historical-review annotations. Those visual checks submitted no extra feedback and changed neither proposal nor Accepted.
- At closeout, `npm run test:change-sets` exposed obsolete selectors for the replaced review form and an ambiguous author assertion shared with collapsed history. The test was updated to use the actual selected-Component line picker and scoped submitted-review presentation, retaining its exact-anchor and restart assertions. Final result: **one production-browser scenario PASS**, one worker, including browser feedback submission, exact source excerpt, direct review URL, history navigation, full Go-process restart, applied/out-of-date proposal behavior, and unchanged Accepted.
- Local evidence: `/tmp/workbraid-review-full.log`, `/tmp/workbraid-review-build.log`, `/tmp/workbraid-reviews-close-go-{test,race,vet,mod}.log`, `/tmp/workbraid-reviews-close-browser.log`, and `/tmp/wb-ux-review/` screenshots/browser log. These are temporary verification artifacts; exact durable identities and outcomes are recorded below.

### Weak-agent iteration and canonical truth

- Separate fresh weak-agent author/reviewer contexts used the documented CLI/embedded skill and configured MCP discovery against one running WorkBraid authority, without repository/private-Git access or command/anchor tutoring. The reviewer submitted request-changes feedback, a fresh author discovered and addressed its exact comments, and a separate reviewer submitted approval of the later exact generation. An interrupted server made an initial author attempt report typed `connection_failed` without mutation; restoring the runtime and retrying the same business task required no product workaround.
- Project: **Reviews Agent Gate**; slug `reviews-agent-gate`; store UUID `df588ba2-cf38-4f98-a6fc-80d3f97e1b72`.
- Proposal: **Reliable ingestion path**; Change Set UUID `e10f85ff-92f4-4edf-a46b-94b13ddd5d77`.
- Accepted stayed exactly `845fbfeadd34113d4e748cc2d1906df51bb3e5cf` throughout authoring, both feedback submissions, restart/GC verification, the UI gate, and the final ref observation. Review approval did not accept the proposal.

| Feedback | Review UUID | Review commit | Exact reviewed state | Generation | Candidate tree |
| --- | --- | --- | --- | --- | --- |
| Request changes | `28825862-5d7d-45ba-b469-181bd8249d08` | `72b2a9c15232dec2bb72718a16671313ea5c5bf1` | `97a20097c580806ca054e5c2baee77ad59268d0c` | 8 | `3a66bc4138163a5136724d914c4d93a4f828b5d1` |
| Approve | `9b426e60-c963-44b5-a646-f215babb3636` | `b67502e5bf54fcf0e80f693ae102787c4db6476d` | `0a5cfcc5f80f4390d1639d4e464ec2cf737ae18c` | 11 | `29bad342713a868eb627e52318213d27054782be` |

Both bindings use base `845fbfeadd34113d4e748cc2d1906df51bb3e5cf`. The final active ref remains the generation-11 state above.

- Request-changes comment `e6535267-da40-4487-8c17-8aa3bb5c2f64` anchors `with_changes` Component Markdown lines 2–2 on Queue Worker `3be835bf-e3c3-4c96-b9ea-b22e836dc0ef`, whose exact old text is `Processes queued jobs.`
- Comment `5c632580-422d-413e-839c-6680039c09e0` anchors the exact `with_changes` fact from that Worker to Event Store `b3d4caa8-9e3a-4232-9e50-4cc27715342d`, label `writes event`, occurrence 1.
- The author expanded retry/backoff, acknowledgement, dead-letter/replay and idempotency documentation; edited the relationship label to `durably commits event before ack; deduplicates by ingestion idempotency key`; and updated proposal Markdown. Generation 11 addressed both comments while generation 8 and its original anchors remained exact and visibly historical. The later approval has an overall Markdown note and zero anchored comments.
- The stronger independent canonical verifier returned **PASS**. Each review commit has exactly one parent: its own reviewed active-form state, whose parent is the original base. Closed trees, modes, comment ordering/pairs, bodies, proposal bytes, candidate-tree equality and refs matched the public claims. Neither historical loading nor GC depended on an active ref still pointing to generation 8.
- Authorized `git gc --prune=now` on only the throwaway gate store changed 107 loose objects to 39 packed reachable objects. All four authoritative refs, exact object identities, and sorted reachability manifests were unchanged; both historical candidates remained reachable and inspectable. `git fsck --full` passed; the unused unborn bare `HEAD` notice was not an integrity failure.
- Full process restart reconstructed current proposal/review and both immutable submissions. Public CLI inspection after GC reported generation 8 as earlier feedback and generation 11 as current. No database/registry or non-Git application data appeared. Final read-only ref inspection after the human gate confirmed the same Accepted, active proposal, and two review refs.
- Detailed agent and independent verifier results are retained locally under `/tmp/workbraid-reviews1-gate.QPJkf7/{reviewer,author-b,reviewer-approve,verifier}/`.

### Human checkpoint and stop

The human inspected the agent-created current and historical review behavior, then requested the bounded UX corrections above. After reviewing the final built UI, the human gave explicit **“Pass. GOod work”**. This is the final human acceptance of Reviews 1; earlier behavior-only/partial passes did not close it.

The human gate runtime on `127.0.0.1:18428` was stopped after PASS. The final browser regression terminated its own temporary Go/browser processes; the independent-review Herdr workspace was closed. Gate data and verification artifacts were retained. Implementation was committed separately from this completion record, with a clean implementation worktree verified before recording completion.

Architecture Reviews 1 is complete. Stop here. Architecture Reconciliation 1 and rich-diagram Phase 3 remain unstarted.
