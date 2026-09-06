# WorkBraid UI v0

Status: approved product direction  
This is not a component library or token system.

## Voice

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
