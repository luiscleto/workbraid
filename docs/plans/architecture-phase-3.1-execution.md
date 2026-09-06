# Phase 3.1 — Stable placement: active plan and roadmap

Status: Approved; in progress. The partial-pinning implementation failed the human visual gate. Its technical PASS did not establish product acceptance.

The [approved placement contract](../architecture-placement-amendment-v0.md) is the current correction. Every visible Component node, including a derived **Lives in** node, keeps its saved coordinate until deliberately moved or auto-laid out. Moving one node must not repack peers. Preserve prior failed-gate evidence; use fresh isolated data for the corrected visual gate.

## Current progress

Reviewed implementation `78cce8d2525bd0cd8f5983ddf8f2fc8ae19b4689` and approved consolidated docs were integrated at `406e18ea26a01fc0ed82e2c1c5efe66bb72de4cf`. Code and documentation tree comparisons were exact. Independent full-range review, Go tests/race/vet/module verification, frontend 94/94, built-browser 24/24 without retries, and canonical/Luna/history/restart verification passed. Integration builds and the live browser smoke passed; no console errors were observed.

The durable self-project at `/home/luisc/.config/workbraid` has one unaccepted proposal with seven Components and two Diagrams, including Canvas as a root boundary through Browser workspace. [Review WorkBraid](http://127.0.0.1:8080/projects/workbraid/proposals/44f2c963-40e3-476f-9ef1-985bf29e7006/review): generation 16, exact draft/catalog/review reconstructed after a complete process restart; the legacy database stayed untouched. The corrected human visual gate and deliberate Architecture acceptance remain pending. Phase 3.1 is not complete.

## Ownership and next work

Root coordinates the human and assigned agents through Herdr. Continue the existing implementation worker for code/tests and the documentation owner for contracts/cleanup. Keep implementation isolated; root integrates the reviewed result. Do not create extra feature workers or repeat baseline/packet commit ceremonies.

1. Complete closed v3 coverage and ordinary final position facts. Keep v2 non-placement and supported operational v1/v2 history exact. The one constructor reconstructs stored facts; it never reruns initial placement or Auto-layout.
2. Complete stable drag/precise X/Y and selected-Diagram Auto-layout for ordinary and boundary nodes. Update CLI/MCP/help/embedded skill and remove unaccepted reset controls together.
3. Try the built application early: drag an ordinary node and Queue/**Lives in**, track every peer and viewport through grab/drop/response/navigation, and verify no accidental navigation. Fix interaction before expensive downstream checks.
4. Independently review the corrected implementation as a whole, emphasizing authority, storage/history, visibility transitions, residual equality and actual interaction. Apply bounded corrections through the same worker.
5. Run proportionate ordinary checks and focused real-browser/CLI/MCP/private-Git/restart evidence. Record the actual integrated build and exact candidate/ref/binding facts needed to reproduce results. Repeat only affected checks after a bounded correction unless risk warrants more.
6. Run fresh bounded weak-agent discovery through public CLI or MCP without source/private-store hints. Independently verify its actual Git state and restart reconstruction.
7. Prepare the corrected human gate through product authoring, obtain explicit visual PASS/FAIL, stop task-owned runtimes and preserve evidence. Do not declare Phase 3.1 complete without human PASS.

## Required evidence

- V3 has one bounded integer pair for every visible Component in each Diagram, no hidden/orphan positions, and no duplicate/missing coverage. V2 still rejects positions and reads never rewrite supported history.
- One drag/drop means one exact-precondition mutation/generation, with no movement-frame writes. Ordinary and boundary nodes work; peers and viewport stay exact through response, navigation and restart. Cancellation/stale/wrong-project failure cannot create an empty proposal or overwrite newer work.
- New visibility gets one initial coordinate through ordinary authoring. Retained visibility keeps coordinates through home/reference/boundary conversion. Full disappearance/reappearance cannot resurrect an inherited pin. No cross-Diagram transfer or subtree repacking.
- Auto-layout changes only the selected Diagram's saved pairs in one mutation, with no membership/Relationship change; unchanged results are a no-op. X/Y, Fit, pan and zoom have distinct purposes.
- First placement against v2 completes v3 coverage in every Diagram through normal facts and exposes the transition in Review. Supported v2/op-v1/v2 candidates and immutable old reviews retain exact blobs, modes, trees and parent reachability after bounded GC in dedicated copies.
- Before/With use their own immutable coordinates in a common frame. Review/history never initialize positions. Map failure leaves the exact diff and feedback inspectable.
- Independent node edits merge; divergent whole pairs require exact side/manual choice. Visibility and mixed v2/v3 comparisons remain truthful. Preview/Check do not mutate refs; Apply verifies S/B/A/P, changes only the active proposal, clears Review, and rejects old-S retry. The residual reconstructs exactly relative to A; normal Review/Update remains separate.
- CLI/MCP/help/skill use the same authority and closed operations. No layout registry, SQLite, session/ref, second constructor, private-Git repair or automatic acceptance.

Run ordinary Go/frontend checks and focused production Playwright cases appropriate to these changes: diff check, Go tests/race/vet/mod verify, frontend tests and production build. Repeat or expand checks for new changes, failures or remaining risks. Tests support real-product evidence; they do not waive human usability.

## Human checkpoint

Use meaningful root/detail Diagrams, connected Components, a canonical reference and a **Lives in** node with readable documentation. The person drags both node kinds, checks peer stability, uses precise position and Auto-layout, pans/zooms/fits, and compares exact Before/With. Preserve a submitted comment on an older binding while editing further. Deliberately Update the final binding, stop/start the process and inspect the reconstructed result.

Then prepare parallel proposals through normal clients. Distinct-node moves combine; one shared node has divergent coordinates. The person reviews/accepts one proposal, understands Original/Accepted/Proposed, selects a whole-pair result, applies reconciliation without changing Accepted, and uses normal Review/Update for the residual. Finish with a fresh-process check and independent exact tree/coordinate proof. Human PASS covers usability and clarity, not just returned coordinates.

## WorkBraid for WorkBraid

Inspect existing configuration/catalog first and use a deliberately selected persistent app-data directory and supported running authority for a real self-project. Do not create it in disposable gate data, import a failed trial store, write private Git, or treat the development source folder as the catalog. Do not initialize it with the rejected partial-placement v3 binary. Author through public operations and preserve deliberate human acceptance of its Architecture. Start with one root and concrete code boundaries; domain content remains subject to the human's Architecture review. This is Architecture dogfooding, not implementation of Planning or Agent Control.

## Later roadmap — not implementation approval

Completed foundations: Architecture/Diagrams and slug-native projects, candidate review, Agent Access, named proposals, immutable Reviews and semantic Reconciliation. Completed plans are archived in Git history.

- **3.2 — Richer node presentation:** consider sizing and separately approved visual treatment. No size fields are predeclared.
- **3.3 — Routing:** first decide presentation addressability for Relationship occurrences, identical parallel facts and derived boundary edges. Node placement does not decide edge identity.
- **3.4 — Shapes/annotations and richer authoring:** only with demonstrated need and a separate canonical model.
- **4A — Semantic Diagram kinds:** distinct models such as UML require their own domain decision; Components are not assumed to be UML Classes.
- **4B — Alternate renderers:** optional presentations, possibly isometric, over approved state. 4A and 4B have no fixed relative order.

Later numbering is indicative; reprioritization requires a human decision. Still deferred: general Component/Diagram deletion or identity replacement, reusable Diagram DAGs/multiple parents, multiple appearances of one Component per Diagram, multiple children per anchor, grouping, sizing/routing/shapes/annotations, multi-select/snapping, undo/history, viewport persistence, source inference, export/synchronization, project lifecycle, remote/multi-user operation, Planning and Agent Control product scope. Stop after the approved Phase 3.1 gate.
