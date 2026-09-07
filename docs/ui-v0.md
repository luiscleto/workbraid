# WorkBraid UI v0

Status: approved product direction  
This is not a component library or token system.

## Printable proposal page

The approved generation-2 printable proposal in [Proposals and Reviews](architecture-proposals-v0.md#printable-bound-proposals) authorizes this dedicated read-only surface, superseding earlier print exclusions only within this scope. The newer human-requested concise correction supersedes paired Before/With print drawings. Keep a compact identity/binding header and nonmutating return link; present Printable proposal as a styled button. Required reading order is design Markdown FIRST, affected Diagrams with highlighted changes, then per-Diagram human-readable concrete facts and inline added/deleted source-text diffs. Omit the optional overview. Use deterministic hierarchy order and exactly one image per affected Diagram: the candidate's exact topology, shapes, notes, routes and geometry, with removed Relationship annotations from existing exact Review deltas. Reuse surviving candidate endpoints; missing endpoints are explicitly base-only annotations at base geometry, never canonical appearances. A Diagram absent from the candidate uses its labelled Before-only drawing. Never auto-layout for printing. Use the actual shared renderer and safe Markdown renderer, with no automatic external resources.

Default to content: Component H1/body source, Diagram title, composition/parentage, Relationship endpoints/labels/multiplicity and note text changes. Exclude position/size/routing/shape-only Diagrams and presentation detail noise, including note-only geometry. A small explicit Include presentation changes option restores these changes and details while retaining one image per Diagram. Full Component H1/body source diff appears once under With home (Before home if absent), with cross-links from other affected Diagrams. Preserve source whitespace/CRLF visibly and escape hostile text. Complete exact canonical diff remains directly inspectable in normal Review; an explicit Include complete raw diff print option adds the appendix, off by default. Empty canonical diff says proposal text only. A nonempty diff with zero Diagrams after filtering must truthfully explain excluded presentation or other changes, never claim no Architecture changes. Highlight changes strongly and keep unchanged topology legible.

Generate self-contained diagram images in browser memory only after fonts and actual drawing finish. Show explicit rendering failure rather than an apparently successful blank drawing. Print is explicit once ready; the browser owns Print / Save as PDF. Use A4 landscape, 12mm margins, one drawing per available sheet fitted to its rendered bounds, aspect ratio preserved, readable legends and freely paginating long text without clipped scroll boxes. Screen controls disappear in print. No PDF service, server file writer, cache, schema or new authority. The final combined human visual gate remains required before Phase 4.

Phase 3.4 is authorized by delegated root approval of proposal `3cf11d4b-c980-4022-92d2-f98d75bf36a4` generation 2, reviewed state `d9d35b11134533016248e5e64294462f76b3aeda`. Placement §11 owns shapes/notes and supersedes earlier exclusions. Offer Default, Rectangle, Ellipse and Diamond in the existing collapsed Position and size section. Preserve role stroke/caption treatment, center/size/routes/frame and safe title omission. Review uses stroke/color/small indicators instead of an added-node hexagon overriding shapes. Independent plain Diagram notes select on canvas or a small contextual Notes list, with Add note, complete text/geometry editing and deliberate Delete note in the existing pane. Notes remain distinct from Components and immutable review comments; text is inert and full source stays available in the pane. Pointer/numeric paths share dirty/stale guards and one-write/cancel behavior. Fit includes notes, which remain stationary during node Auto-layout. Review lists note/shape facts separately, uses exact Before/With values and existing Diagram comment anchors. Preserve review dock/navigation, safe Markdown and neutral clear behavior. No print or Phase 4 work is included; the final combined human visual gate remains required.

Phase 3.3 routing is authorized by delegated root approval of proposal `d119baeb-1b2b-4451-8113-7f4fe05678ec` generation 8, exact reviewed state `595bab6de6933c7a047c70520f1830594f166fed`. This supersedes routing exclusions below. [Placement §10](architecture-placement-amendment-v0.md#10-deliberate-link-routing-phase-33) owns the exact shared intersection/control-point geometry, eligibility, no-op and reset rules. Select an ordinary or crossing link by its exact Diagram tuple/occurrence presentation slot. Show a control-point handle with a restrained off-curve guide and **Bend**, **Keep route**, **Restore default** in the existing collapsed geometry language. Never hide unsent fields. Provide occurrence N of count without implying declaration identity. Pointer/numeric paths share exact authority and dirty-state guards; one release writes once, cancellation/click writes nothing, and peers/frame remain fixed. Self-links have no manual controls. Coincident routed endpoints retain their scalar with a clear derived-fallback indication. Fit and Before/With include curve extents and captions; Review/history remain read-only. Routing changes, including automatic resets, are separate from Relationship multiset changes. Reconciliation exposes the closed route-value choices and acknowledged whole-tuple clear for route loss; revising semantics appears only for actual returned semantic conflicts. The final combined human visual gate before Phase 4 remains required.

Phase 3.2 adds the sizing interaction specified normatively in [Placement §9](architecture-placement-amendment-v0.md#9-complete-visible-node-sizing-phase-32), superseding the earlier sizing exclusion below. The selected editable visible node has a discoverable corner resize handle and precise width/height fields, plus Restore default size. Preview preserves center, peers and viewport; cancellation writes nothing and one release submits one mutation. Titles wrap inside the safe shape region with explicit ellipsis and full title/home context in the existing pointer/keyboard-accessible pane. Caption geometry, zoom and Fit use the exact shared envelope. Review distinguishes size changes and uses each side's dimensions in a common frame. Empty canonical diff hides its empty box and says “No Architecture changes; this proposal contains only proposal text.” No lifecycle or broader review redesign is added. Human visual PASS remains required.

## Voice

Routing rendering follows the version/source-checked Cytoscape 3.34.1 patch in Placement §10. Undefined/nonfinite shape intersections make only canvas bend dragging unavailable, with an explicit browser explanation; numeric authoring remains available for canonically eligible endpoints. Preserve stored bends and show derived rendering only where possible, without promising a visible fallback curve. Resume the saved bend without a write when representable. Selection and deselection must not change intersection geometry. Before/With Fit uses actual-renderer union bounds of both exact sides, including curves and captions, while ordinary routing preserves the viewport.

The UI is for a person at a desk, not an API log. Headings are status. Body is one short sentence. Errors say what to do next.

Write as if the reader is tired and non-technical. Prefer what a thing is. Mention what it is not only when a real confusion exists.

Do not pipe backend sentinel strings into the page. Map each failure to one operator sentence that says what to do. Do not claim something is absent when you only failed to look it up.

## Terminology

Say **project**, **architecture**, **Diagram**, and **Component**.

Internal names stay off the screen. Do not show implementation words (store UUID, canonical, snapshot, payload, origin, and whatever the current internals are called).

## What is on screen

Show only actions and state that exist now. Do not add disabled future controls, placeholder panels, empty queues, or explanatory chrome for features that are not implemented.

Progressively disclose machinery. Canonical file paths, IDs, Git revisions, and raw errors are first-class when the current task needs them. Otherwise they belong in details or inspection, not on every surface.

## Visual direction

A **drafting table**: one surface, hairline structure, warm paper, almost no radius or shadow, one accent. Typography does the hierarchy. Not sage SaaS mint, not a green theme, not three stacked tiles.

- Use a deliberately chosen, actually loaded display face and body face with a distinct editorial or technical character. Generic product-SaaS typography or a system-only stack is not acceptable.
- Mono only for paths and IDs, and only where those objects belong.
- Hide empty result chrome until there is something to show.
- Almost no motion: focus and state appearing. No hero animations, no gradient mesh, no purple.
- Screens that share a product must look like one product.
- The architecture map is a 2D drawing unless a later approved visual spec says otherwise.

## Architecture workspace

### Project catalog and routes

The entry state is a WorkBraid project catalog, not a source-folder picker. It lists discoverable private Architecture projects by human-readable name and provides one deliberate **New project** action. Creating a project asks only for its name; WorkBraid generates its stable route slug and opens the native writable Architecture after successful initialization.

Selecting an existing project opens it by catalog identity. `/projects/<slug>` restores that same project on reload and after a fresh process start. The slug may appear in the browser URL and bounded technical details, but the display name remains primary. Store UUIDs and private Git paths are not normal selection chrome.

WorkBraid offers no slug-edit action. If explicit **Refresh** adopts an authoritative external accepted revision with another valid slug, replace the current browser route with `/projects/<new-slug>`. The old route then behaves like any other unknown slug unless another project currently owns it. Do not present this accepted-state observation as a rename or migration workflow.

Project names need not be unique. When names collide, show the minimum slug context needed to distinguish them. An unknown route shows an ordinary not-found state with an action back to the project catalog and never creates a project. Duplicate discovered slugs show an explicit catalog conflict and never select a winner. A discovered malformed store appears as unavailable with bounded technical context where practical rather than silently disappearing; no repair or recovery workflow is implied.

Project opening and creation are entry states, not permanent workspace chrome. Once a project is open, the catalog is gone and WorkBraid shows a map-centered Architecture workbench. The application frame keeps the current project visible and provides an unobtrusive way to open another project through the catalog. Unsent browser-local editor/review-composer guards apply before leaving. Durable proposals survive project/context switches; they do not have to be discarded.

On a normal desktop viewport, the workbench has:

- a compact Diagram navigator and Component index;
- the selected Diagram from the explicit Accepted or named-proposal context as the primary canvas;
- one contextual working pane for the current task: accepted component documentation or structured authoring.

The Diagram tree, index, map and documentation project one exact selected context: Accepted, a complete valid named proposal, or a read-only applied record. A proposal is never overlaid on Accepted. Invalid proposals show their exact authored values, validation and repair actions without a partial map. The compact proposal selector and Changes in progress task follow [Proposals and Reviews](architecture-proposals-v0.md).

The component index is not a management or dashboard surface. It selects components by stable identity and primarily shows their titles, plus only the minimal component-creation affordance needed. When titles collide, show the minimum filename or shortened-ID context needed to disambiguate them. Do not make IDs or paths general index chrome, and do not add status columns, per-component management controls, filters, or speculative controls.

Selecting a component from the map or index focuses the same context-owned Component and shows its documentation in the working pane. For a writable accepted Architecture, Add/Edit uses that pane for structured component and relationship authoring. The Accepted context does not preview proposed topology.

**Changes in progress** is a compact visible workspace affordance. It reuses the working area for pending editing, review, and acceptance rather than becoming another permanent region. Exact diff review may temporarily expand into more of the workspace when the task requires it.

### Review changes

**Review changes** remains one task combining visual review and the complete exact unified diff. It is available only after the complete candidate validates. Invalid work remains under Changes in progress with concise, actionable guidance.

The candidate view is the primary review canvas. A compact toggle switches the entire review workspace between **With changes** and **Before changes**:

- **With changes** shows the exact immutable reviewed candidate;
- **Before changes** shows that review's exact bound base, not newly observed accepted Architecture.

The Diagram tree, selected Diagram, component index, map, selected documentation/detail, titles, canonical appearances, boundary references, and relationship topology always switch together. Never show one snapshot's map beside another snapshot's tree, index, or documentation. If external authority moves after review, mark the review stale through the existing product language; do not relabel its bound base as current.

In the candidate view:

- unchanged topology is subdued;
- added components and relationships are visibly distinct;
- changed existing components remain indicated when only their Title, Description, or documentation changed;
- removed relationship facts are visibly distinct, such as ghosted or dashed;
- selecting a changed component or relationship focuses its review context and the relevant region of the exact unified diff.

Components and Diagrams match by stable UUID. Relationships compare as a multiset of exact `(source Component ID, target Component ID, label)` facts, preserving direction and multiplicity; a target/label edit is a removed fact plus an added fact. Render-only edge keys never give identical parallel facts domain identity. Ordinary Component deletion is not an authoring capability and does not introduce removed-Component ghosts into this review.

For Diagram candidates, the same review task also distinguishes added Diagrams, Diagram title changes, home/reference appearance changes, home moves, and detail-link changes. A Component home move is shown as Diagram-composition removal/addition. If composition alone makes a real Relationship change between ordinary and boundary presentation, do not describe that as an Architecture Relationship addition or removal. Selecting a Diagram or membership change focuses its Diagram context and corresponding canonical Diagram-file diff.

If the selected Diagram exists only in **With changes**, switching to **Before changes** selects the nearest ancestor which exists in the bound base, or the bound base root when no ancestor survives. Show a restrained note that the previously selected Diagram exists only with the changes. Never retain that candidate-only Diagram's composition, index, documentation context, boundary references, or topology on the base side. Restoring its exact focus when returning to **With changes** is optional UI behavior.

The raw unified diff remains directly inspectable in the same Review changes surface. Basic added/removed line coloring may improve readability, but it does not become a semantic or rendered-Markdown diff. If the visual map fails to render, say so clearly and retain the validated candidate and complete unified diff review path.

Before/With use their own snapshot coordinates in a common logical frame; toggling must not independently fit away the movement. V2 uses deterministic read-time fallback; v3 uses exact complete saved positions. Opening review/history never initializes or saves layout. Position changed is distinct from content, membership and Relationship deltas, including for boundary nodes.

Validation-bearing rows in Changes in progress visibly indicate which component needs attention. Fix affordances must look actionable, and opening one highlights and focuses the exact affected relationship control. The contextual pane also provides a clear/deselect action where selection would otherwise trap the current document or task.

An out-of-date valid proposal remains editable and reviewable against its exact original base, with conspicuous **Out of date with Accepted** context. It cannot Update until its base equals current Accepted. [Reconciliation](architecture-reconciliation-v0.md) is the explicit task for combining it with Accepted; Apply changes the proposal, then ordinary Review/Update follows. Unknown/non-current Accepted authority is distinct and pauses mutation/new review/acceptance while preserving inspection.

The application frame keeps the current project and Architecture context visible, with compact actions for explicit refresh and returning to the project catalog. Do not permanently display a positive current/accepted status merely because it exists. Make stale or non-current state conspicuous when relevant; otherwise let the workspace stay quiet.

The same surfaces may collapse into one-at-a-time views on narrower layouts. Mobile-specific interaction remains deferred.

### Diagram navigation and composition

Architecture opens at its root Diagram and provides a compact project-style Diagram tree plus breadcrumbs. The tree is navigation, not a Diagram-management dashboard.

For the selected Diagram:

- the index lists its canonical home and reference Component appearances using titles as the primary label;
- the map shows those appearances, ordinary Relationships whose endpoints both appear, and derived boundary references for Relationships crossing the Diagram boundary;
- selecting a Component from the tree/index/map opens the same canonical documentation in the contextual pane;
- activating a home Component's detail affordance drills into its child Diagram; this navigation action is visually distinct from Component editing and appears below the edit action in the contextual pane;
- back/breadcrumb navigation returns to the parent with the anchor Component identifiable;
- selecting a boundary node exposes local positioning without forcing navigation; **Open home** remains an explicit action to the external Component's home Diagram.

For each absent external Component, the active Diagram shows at most one derived boundary reference. Every crossing Relationship occurrence connects to that one reference, retaining its own direction, label, and multiplicity.

Normal product language distinguishes the two forms without requiring domain jargon:

- a canonical reference appearance is **Included here · Lives in _home Diagram_**;
- a derived boundary reference says the related Component **Lives in** its home Diagram.

The map canvas keeps the Component title primary. Canonical reference nodes add no secondary canvas wording; derived boundary nodes add a subdued **Lives in _home Diagram_** subtitle so their external location remains understandable. Reference and home-location wording in the index or contextual dock is visually subordinate, small, italic, and muted.

IDs, Diagram filenames, YAML roles, and hierarchy keys remain out of normal navigation chrome. Duplicate Diagram or Component titles receive only the minimum ancestor, anchor, filename, or shortened-ID context required to distinguish them.

Diagram authoring reuses the contextual working pane. The first Diagram slice provides structured tasks to create and title a detail Diagram, move a Component's home, and also show or stop showing a Component by reference. Creating a Component inside an active Diagram places it there; root is the fallback only when no Diagram context exists. Diagram deletion and general hierarchy management are not shown.

Newly initialized Architecture is already ready for Diagram and Component authoring and receives no setup, migration, source-folder, or linking explanation.

Every visible Component node, including a **Lives in** node, has a stable saved v3 center. Dragging one node keeps all peers and the viewport fixed through response/navigation/restart. The contextual pane offers precise X/Y and **Keep position**; selected-Diagram **Auto-layout** deliberately saves all visible positions in one mutation. Remove trial Reset position/Reset layout and ongoing manual/Automatic modes. Fit/zoom/pan are view actions. Review/history are read-only. The [placement contract](architecture-placement-amendment-v0.md) defines visibility, v2 fallback and complete coverage. No sizing, routing, shapes or viewport persistence is implied.


The drafting-table text specification above is the living visual direction. Superseded inspiration images are removed; a real product interaction and explicit human visual gate determine acceptance.
