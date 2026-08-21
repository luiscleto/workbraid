# WorkBraid UI v0

Status: approved product direction  
This is not a component library or token system.

## Voice

The UI is for a person at a desk, not an API log. Headings are status. Body is one short sentence. Errors say what to do next.

Write as if the reader is tired and non-technical. Prefer what a thing is. Mention what it is not only when a real confusion exists. Trim pasted paths.

Do not pipe backend sentinel strings into the page. Map each failure to one operator sentence that says what to do. Do not claim something is absent when you only failed to look it up.

## Terminology

Say **folder**, **project**, **linked**, **architecture**.

Internal names stay off the screen. Do not show implementation words (source root, association, inspect, canonical, snapshot, payload, origin, and whatever the current internals are called).

## What is on screen

Show only actions and state that exist now. Do not add disabled future controls, placeholder panels, empty queues, or explanatory chrome for features that are not implemented.

Progressively disclose machinery. Paths, IDs, Git revisions, and raw errors are first-class when the current task needs them. Otherwise they belong in details or inspection, not on every surface.

## Visual direction

A **drafting table**: one surface, hairline structure, warm paper, almost no radius or shadow, one accent. Typography does the hierarchy. Not sage SaaS mint, not a green theme, not three stacked tiles.

- Use a deliberately chosen, actually loaded display face and body face with a distinct editorial or technical character. Generic product-SaaS typography or a system-only stack is not acceptable.
- Mono only for paths and IDs, and only where those objects belong.
- Hide empty result chrome until there is something to show.
- Almost no motion: focus and state appearing. No hero animations, no gradient mesh, no purple.
- Screens that share a product must look like one product.
- The architecture map is a 2D drawing unless a later approved visual spec says otherwise.

## Architecture workspace

Project opening and setup are entry states, not permanent workspace chrome. Once a project is open, the opening sheet is gone and WorkBraid shows a map-centered Architecture workbench.

On a normal desktop viewport, the workbench has:

- a compact project/Diagram navigator and Component index appropriate to the accepted format;
- the accepted v1 implicit map or selected accepted v2 Diagram as the primary canvas;
- one contextual working pane for the current task: accepted component documentation or structured authoring.

The Diagram tree, component index, map, and documentation are projections of the same exact accepted Architecture revision. Pending title, Diagram, membership, or hierarchy changes do not alter those normal surfaces before acceptance, and pending new Components or Diagrams do not appear in them. Pending work remains reachable through **Changes in progress**.

The component index is not a management or dashboard surface. It selects components by stable identity and primarily shows their titles, plus only the minimal component-creation affordance needed. When titles collide, show the minimum filename or shortened-ID context needed to disambiguate them. Do not make IDs or paths general index chrome, and do not add status columns, per-component management controls, filters, or speculative controls.

Selecting a component from the map or index focuses the same accepted component and shows its documentation in the working pane. For a writable accepted Architecture, Add/Edit uses that pane for structured component and relationship authoring. The accepted map does not preview pending topology.

**Changes in progress** is a compact visible workspace affordance. It reuses the working area for pending editing, review, and acceptance rather than becoming another permanent region. Exact diff review may temporarily expand into more of the workspace when the task requires it.

### Review changes

**Review changes** remains one task combining visual review and the complete exact unified diff. It is available only after the complete candidate validates. Invalid work remains under Changes in progress with concise, actionable guidance.

The candidate view is the primary review canvas. A compact toggle switches the entire review workspace between **With changes** and **Before changes**:

- **With changes** shows the exact immutable reviewed candidate;
- **Before changes** shows that review's exact bound base, not newly observed accepted Architecture.

The Diagram tree or v1 implicit-map context, selected Diagram, component index, map, selected documentation/detail, titles, canonical appearances, boundary references, and relationship topology always switch together. Never show one snapshot's map beside another snapshot's tree, index, or documentation. If external authority moves after review, mark the review stale through the existing product language; do not relabel its bound base as current.

In the candidate view:

- unchanged topology is subdued;
- added components and relationships are visibly distinct;
- changed existing components remain indicated when only their Title, Description, or documentation changed;
- removed relationship facts are visibly distinct, such as ghosted or dashed;
- selecting a changed component or relationship focuses its review context and the relevant region of the exact unified diff.

For v2 candidates, the same review task also distinguishes added Diagrams, Diagram title changes, home/reference appearance changes, home moves, and detail-link changes. A Component home move is shown as Diagram-composition removal/addition. If composition alone makes a real Relationship change between ordinary and boundary presentation, do not describe that as an Architecture Relationship addition or removal. Selecting a Diagram or membership change focuses its Diagram context and corresponding canonical Diagram-file diff.

During v1-to-v2 review, **Before changes** remains the real implicit v1 map and **With changes** shows the candidate Diagram tree. The UI does not fabricate a v1 Diagram identity.

If the selected Diagram exists only in **With changes**, switching to **Before changes** selects the nearest ancestor which exists in the bound v2 base, or the bound base root when no ancestor survives. With a v1 base, it switches to the real implicit all-components map. Show a restrained note that the previously selected Diagram exists only with the changes. Never retain that candidate-only Diagram's composition, index, documentation context, boundary references, or topology on the base side. Restoring its exact focus when returning to **With changes** is optional UI behavior.

The raw unified diff remains directly inspectable in the same Review changes surface. Basic added/removed line coloring may improve readability, but it does not become a semantic or rendered-Markdown diff. If the visual map fails to render, say so clearly and retain the validated candidate and complete unified diff review path.

Automatic review layout should be deterministic and stable-ID-aware, keeping unchanged components as stable as practical between Before changes and With changes. No review coordinates are canonical or persisted.

Validation-bearing rows in Changes in progress visibly indicate which component needs attention. Fix affordances must look actionable, and opening one highlights and focuses the exact affected relationship control. The contextual pane also provides a clear/deselect action where selection would otherwise trap the current document or task.

Pending work whose accepted base is stale remains visible and read-only through **Changes in progress**. It cannot be reviewed or accepted. The human may discard that whole non-canonical change set so new work can begin from current accepted Architecture; discard is not partial editing, reconciliation, or undo.

The application frame keeps the current project and Architecture context visible, with compact actions for explicit refresh and opening another project. Do not permanently display a positive current/accepted status merely because it exists. Make stale or non-current state conspicuous when relevant; otherwise let the workspace stay quiet.

The same surfaces may collapse into one-at-a-time views on narrower layouts. Mobile-specific interaction remains deferred.

### Diagram navigation and composition

Format v1 retains its existing implicit all-components map and does not pretend to have a canonical Diagram identity. Format v2 opens at its root Diagram and adds a compact project-style Diagram tree plus breadcrumbs. The tree is navigation, not a Diagram-management dashboard.

For the selected Diagram:

- the index lists its canonical home and reference Component appearances using titles as the primary label;
- the map shows those appearances, ordinary Relationships whose endpoints both appear, and derived boundary references for Relationships crossing the Diagram boundary;
- selecting a Component from the tree/index/map opens the same canonical documentation in the contextual pane;
- activating a home Component's detail affordance drills into its child Diagram;
- back/breadcrumb navigation returns to the parent with the anchor Component identifiable;
- activating a boundary reference opens the external Component in its home Diagram.

For each absent external Component, the active Diagram shows at most one derived boundary reference. Every crossing Relationship occurrence connects to that one reference, retaining its own direction, label, and multiplicity.

Normal product language distinguishes the two forms without requiring domain jargon:

- a canonical reference appearance is **Also shown here**;
- a derived boundary reference says the related Component **Lives in** its home Diagram.

IDs, Diagram filenames, YAML roles, and hierarchy keys remain out of normal navigation chrome. Duplicate Diagram or Component titles receive only the minimum ancestor, anchor, filename, or shortened-ID context required to distinguish them.

Diagram authoring reuses the contextual working pane. The first Diagram slice provides structured tasks to create and title a detail Diagram, move a Component's home, and also show or stop showing a Component by reference. Creating a Component inside an active Diagram places it there; root is the fallback only when no Diagram context exists. Diagram deletion and general hierarchy management are not shown.

Newly initialized Architecture is already ready for Diagram and Component authoring and receives no setup or migration explanation.

A valid accepted v1 Architecture remains readable and navigable through its implicit all-components map, but normal Add/Edit and relationship authoring are not shown. When no Changes in progress exist, it offers one concise **Set up diagrams** action with product language such as **Set up diagrams to start editing this architecture.** Do not mention format versions, migration, schemas, YAML, root identity, or upgrade steps.

**Set up diagrams** creates one reviewable pending change containing only the Diagram setup. It does not also begin detail-Diagram or Component authoring. Because its complete candidate is the writable Architecture shape, the human reviews and deliberately updates Architecture through the normal Changes-in-progress and Review-changes workspace. Cancel or whole-set discard leaves the accepted Architecture untouched.

If non-setup pending work somehow already exists for this readable older Architecture, keep it visibly read-only under **Changes in progress**. Show no Edit, Fix, Review changes, or Update architecture action for that work. Whole-set **Discard changes** remains available; **Set up diagrams** stays unavailable until discard clears it. Do not merge, reinterpret, accept, recover, or persist this defensive transitional state.

Automatic Diagram layout remains disposable presentation. No drag position, route, bend point, size, shape, or view state is implied or persisted.

## Map references

These images are tone and information-design references for a later map, not the first-slice widget set and not a 3D assignment:

- [System map](ui-v0-ref-system-map.jpg)
- [Loop map](ui-v0-ref-loop-map.jpg)

An approved screenshot of the live product can be added here once a screen matches this direction.
