# WorkBraid Architecture v0

Status: approved design baseline  
Scope: Architecture vertical, first real slice

This document records approved Architecture decisions. It separates portable domain/store invariants from the initial implementation profile. It is not an implementation plan.

## 1. Domain boundary

WorkBraid has three related verticals:

- Architecture owns architecture knowledge, components, relationships, visualization, accepted revisions, and the Architecture proposal lifecycle.
- Agent Control owns Herdr, sessions, workspaces, terminals, messaging, and runtime state.
- Planning owns work items, dependencies, gates, readiness, waves, and work-item review/integration/validation state.

Only Architecture is currently in scope. It must be buildable and usable without Planning, Herdr, or Agent Control.

Agents may act across verticals through authorized operations. Shared agent access does not merge domain models. Planning may originate an Architecture proposal but cannot directly mutate accepted Architecture.

## 2. Canonical authority

Accepted Architecture is canonical Markdown plus minimal structural metadata stored in a private WorkBraid-owned Git repository.

The private repository:

- lives under per-user WorkBraid application data initially;
- is one discoverable WorkBraid project in that private store area;
- must be self-describing;
- must remain independently intelligible without the WorkBraid implementation.

WorkBraid does not require or open an arbitrary source-project folder. Export or synchronization into another repository is optional later functionality requiring explicit configuration.

### Accepted revision

`refs/heads/accepted` is the sole authoritative pointer to accepted Architecture.

- A commit is canonical because `accepted` points to it, not because WorkBraid created it.
- `HEAD`, checkout state, working trees, catalog enumeration, and loaded snapshots have no authority over accepted state.
- An authoritative human may deliberately advance `accepted` using ordinary Git.
- WorkBraid loads and structurally validates whatever `accepted` references.
- Invalid or unsupported state at `accepted` produces a clear load failure.
- WorkBraid never silently falls back to another ref or older revision.

The Git/Markdown contract is independent of Go, Git CLI usage, bare-repository layout, or any particular WorkBraid implementation.

## 3. Store contract

The closed format-v2 accepted tree layout is:

```text
architecture.yaml
components/
  <filename>.md
diagrams/
  <filename>.yaml
```

Format v2 discovers Components non-recursively as `components/*.md` and Diagrams non-recursively as `diagrams/*.yaml`. A valid accepted tree contains only:

- `architecture.yaml`; and
- zero or more Component files directly under `components/` whose filenames end in `.md`; and
- one or more Diagram files directly under `diagrams/` whose filenames end in `.yaml`.

The `components/` directory may be absent when there are zero Components. Nested directories and every other accepted-tree path are invalid.

Each canonical manifest, Component, or Diagram path must be an ordinary Git blob entry. A symlink, submodule/gitlink, or tree at one of those paths is invalid. WorkBraid-created canonical files use mode `100644`. Executable mode has no Architecture semantics; when editing an existing regular-file blob, WorkBraid does not gratuitously change its existing regular-file mode.

### Store manifest

`architecture.yaml` contains:

- a format identifier;
- a format version;
- an immutable opaque WorkBraid store ID;
- a human-readable project name;
- a stable human-readable project slug; and
- the stable identity of the root Diagram.

The literal v2 manifest shape is:

```yaml
format: workbraid-architecture
version: 2
store_id: "6f2f9de7-22c2-4cd5-b7da-91f3454f09e4"
project:
  name: "Example Project"
  slug: "example-project"
root_diagram: "2a863995-88e8-40b5-af0f-34398474dc0a"
```

Its exact fields and types are:

- `format`: required string with the literal value `workbraid-architecture`;
- `version`: required integer with the literal value `2`;
- `store_id`: required string containing a valid UUID;
- `project`: required mapping containing exactly:
  - `name`: required string, non-empty after trimming;
  - `slug`: required string matching `[a-z0-9]+(?:-[a-z0-9]+)*`;
- `root_diagram`: required string containing the stable UUID of one discovered Diagram.

The v2 manifest schema is closed. Unknown keys at the top level or inside `project` are invalid rather than ignored. Future semantic fields require format evolution.

Format and version are compatibility guards. WorkBraid rejects unsupported values rather than interpreting them using current assumptions. This alpha contract supports only the exact format v2 defined here. Old alpha stores are disposable; WorkBraid provides no format-v1 loader, setup action, migration, compatibility adapter, or downgrade.

The WorkBraid store ID is the store's immutable identity. Project name and slug are human-facing project information only:

- neither defines store identity;
- project names need not be unique;
- WorkBraid provides no slug-edit operation and every WorkBraid-authored candidate preserves the exact current slug, making it stable for the project's lifetime under normal WorkBraid use;
- the slug is the current human-facing catalog and route locator, not store identity; and
- slug uniqueness is a local catalog invariant rather than portable store identity.

Project creation trims leading and trailing whitespace from the submitted name, rejects an empty result, and stores that result as `project.name`. It derives a human-readable slug from that stored name. The creation algorithm lowercases ASCII letters, replaces each maximal run outside `[a-z0-9]` with one hyphen, trims leading and trailing hyphens, and uses `project` if nothing remains. If that slug already exists, WorkBraid appends `-2`, then `-3`, and so on until it finds the first available slug. This deterministic suffix choice is creation behavior; the portable invariant is a valid slug.

Because `refs/heads/accepted` remains sole authority and there is deliberately no second durable slug registry, an authoritative external accepted update may replace the manifest with another otherwise-valid slug. If WorkBraid loads or explicitly Refreshes to that revision, the new accepted slug becomes that store UUID's current canonical catalog and route locator. Catalog uniqueness and conflict handling apply normally. The old `/projects/<old-slug>` subsequently resolves as not found unless another valid store currently owns it. When explicit Refresh adopts a new slug for the open project, WorkBraid replaces the browser route with `/projects/<new-slug>` rather than knowingly retaining the stale locator. This is accepted-state observation, not slug editing, migration, history traversal, or another identity authority.

### Components

The initial canonical map unit is an Architecture Component.

Each component:

- has one Markdown body;
- is represented by one Markdown file;
- has one immutable opaque component ID;
- has one required canonical title;
- may declare outgoing relationships.

There is no separate Document domain object and no central component registry.

UUIDs are the initial encoding for store and component IDs. UUID encoding is not a semantic identity scheme.

A component ID survives:

- file rename;
- title change;
- body edits;
- relationship edits.

Deliberately changing the ID creates replacement/new identity rather than an ordinary edit. Filenames carry no identity.

After YAML frontmatter and optional whitespace, the first Markdown block is the required level-one heading and canonical component title. A usable title is non-empty after trimming. Both CommonMark ATX and Setext level-one headings are accepted on load; WorkBraid-generated component files use an ATX H1. The remainder is the component body. The title is not duplicated in frontmatter.

The canonical component H1 remains Markdown source. The structured component Title is its human-readable text projection, not its Markdown source spelling. That projection parses inline Markdown into text: it resolves escapes and entities; discards emphasis, strong, and strikethrough formatting while retaining their text; retains code-span text and visible link text without destinations; and treats raw HTML as literal inert text. Leading and trailing whitespace is trimmed, and the resulting Title must be non-empty.

WorkBraid normalizes a submitted structured Title by trimming leading and trailing whitespace. When serializing that Title into an H1, WorkBraid escapes or encodes the text as needed so parsing the resulting H1 yields the same normalized structured Title. This is a projection and serialization rule, not a separate title encoding or portable metadata field.

The literal v2 Component-frontmatter shape is:

```yaml
---
id: "0f86a8c3-487a-4bc8-9ff0-9d0d7c9dcd34"
relationships:
  - target: "c7f3d6b4-3f4a-42b9-87b8-4d7be325fd79"
    label: "calls"
---
# API
```

Its exact fields and types are:

- `id`: required string containing a valid UUID;
- `relationships`: optional sequence; omission means no outgoing relationships;
- each relationship item: a mapping containing exactly:
  - `target`: required string containing a valid component UUID;
  - `label`: required string, non-empty after trimming.

The Component-frontmatter schema is closed. Unknown Component or Relationship-item keys are invalid rather than ignored. Future semantic fields require format evolution.

### Relationships

Outgoing relationships are declared in the source component's YAML frontmatter.

Each relationship contains:

- a stable target component ID;
- a short source-relative human-readable phrase such as `calls`, `reads from`, or `publishes events to`.

The containing component implies the source; source ID is not repeated.

Relationship rules:

- direction is meaningful;
- labels are free text, not an enumerated taxonomy;
- multiple relationships between the same source and target are allowed;
- relationships do not initially have stable IDs or lifecycle;
- target IDs must resolve within the complete revision being validated: the accepted tree during load and the candidate tree before commit;
- cycles are allowed;
- relationship order has no domain meaning;
- no hierarchy or central relationship registry exists initially.

Relationships are explicit authored Architecture facts. They are never inferred from source code, runtime traffic, Markdown links, Planning, or Agent Control.

### Markdown contract

Component Markdown is UTF-8 CommonMark plus these supported extensions:

- tables;
- task lists;
- strikethrough;
- autolinking.

WorkBraid does not claim general GitHub-Flavored Markdown compatibility.

Rendering rules:

- authored content is never executed;
- fenced code is presentation-only;
- raw HTML is rendered inertly rather than interpreted as active HTML;
- raw HTML does not by itself make a component structurally invalid;
- Mermaid, executable blocks, includes, directives, and similar syntax have no special semantics;
- authored content does not trigger automatic arbitrary external network or file access;
- normal links may be followed only through deliberate user action;
- Markdown links never imply Architecture relationships;
- rendering never rewrites canonical source.

WorkBraid should avoid gratuitously rewriting unrelated Markdown or frontmatter when editing canonical files. The first slice does not require a general lossless-formatting subsystem.

### Format v2 Diagram contract

Format v2 contains first-class Architecture Diagrams without changing Component or Relationship identity or meaning. Components and Relationships remain Architecture semantic facts. A Diagram is canonical composition and presentation over those facts; it is not a generic node/edge replacement for them. Diagram filenames carry no identity.

The literal v2 Diagram-file shape is:

```yaml
id: "2a863995-88e8-40b5-af0f-34398474dc0a"
title: "System"
appearances:
  - component: "0f86a8c3-487a-4bc8-9ff0-9d0d7c9dcd34"
    role: home
    detail_diagram: "e76811e6-6125-48e1-8646-00f464834b27"
  - component: "c7f3d6b4-3f4a-42b9-87b8-4d7be325fd79"
    role: reference
```

Its exact fields and types are:

- `id`: required string containing a valid UUID;
- `title`: required string, non-empty after trimming;
- `appearances`: optional sequence; omission means the Diagram has no Component appearances;
- each appearance item: a mapping containing exactly:
  - `component`: required string containing the UUID of a Component in the complete revision;
  - `role`: required string with the literal value `home` or `reference`;
  - `detail_diagram`: optional string containing a discovered Diagram UUID, permitted only when `role` is `home`.

The v2 Diagram schema is closed. It contains no layout, position, size, route, bend-point, shape, annotation, renderer, or Diagram-kind field. Appearance source order has no domain meaning, though ordinary authoring should not gratuitously reorder surviving entries.

#### Diagram identity, membership, and hierarchy

Each Diagram has one immutable stable ID and one mutable human-readable title. Diagram titles need not be globally unique. Changing a Diagram title or filename does not change its identity.

The complete v2 revision must satisfy all of these invariants:

- Diagram IDs are unique and `root_diagram` resolves to exactly one Diagram;
- the root is otherwise an ordinary Diagram and has no parent anchor;
- any Diagram, including the root, may have no Component appearances;
- every non-root Diagram is the `detail_diagram` of exactly one home appearance;
- every Diagram is reachable from the root and hierarchy cycles are invalid;
- every Component has exactly one `home` appearance across the complete revision;
- a Component may have `reference` appearances in other Diagrams;
- a Component appears at most once in any one Diagram and therefore cannot be both `home` and `reference` there;
- a reference appearance never owns a detail link;
- a home appearance initially owns at most one detail link because `detail_diagram` is singular;
- every Component and Diagram ID named by an appearance resolves within the complete revision.

Home and reference are Diagram-composition roles, not Component semantic facts. A home identifies the Component's primary place in the Diagram tree. A reference means that the same Component is also deliberately shown in another Diagram; it does not copy the Component, its documentation, or its Relationships.

The optional detail link lives only on the parent Diagram's home appearance of the anchoring Component. Child Diagrams do not duplicate parent or anchor fields. The complete composition therefore derives the Diagram tree, breadcrumbs, back navigation, and the anchor to identify when returning to a parent.

Moving a Component's home changes only Diagram composition. It preserves the Component ID, canonical Component file, documentation, and Relationships. If the destination already contains a reference appearance for that Component, the move converts that entry to `home` rather than creating a duplicate. The old home entry is removed and does not silently remain as a reference.

When a moved home appearance owns a detail link, that link moves with the home appearance, reparenting its child Diagram and complete subtree. The resulting complete candidate must still have one root, unique parents, complete reachability, and no hierarchy cycle. Moving an anchor beneath its own descendant is invalid rather than automatically repaired.

In v2 authoring, a new Component created while a Diagram is the active context receives its home appearance in that Diagram. When creation genuinely has no Diagram context, the root is the default. A valid v2 revision has no homeless or reference-only Component.

#### Relationships in a Diagram

Relationships remain the one global set of source-owned Component-to-Component Architecture facts defined by Component frontmatter. Diagram composition neither duplicates nor owns them.

For one active Diagram:

- if both Relationship endpoints have canonical home or reference appearances there, render the ordinary directed labelled Relationship;
- if exactly one endpoint appears, derive a boundary/external reference for the absent endpoint and render the real Relationship to or from it;
- if neither endpoint appears, omit that Relationship from that Diagram.

Within one active Diagram, WorkBraid derives at most one boundary/external reference for each absent external Component. Every crossing Relationship occurrence to or from that Component connects to that one derived reference while retaining its exact direction, label, and multiplicity. Derived boundary references have no canonical identity, membership, or separate Architecture meaning. Activating one navigates to the external Component's home Diagram. A canonical reference appearance is a real Diagram appearance and therefore uses an ordinary Component node and Relationship edge rather than a boundary reference.

## 4. Initialization and loading

### Bootstrap revision

Architecture-store initialization is an explicit human action.

Initialization succeeds only when:

1. a valid bootstrap commit exists; and
2. `refs/heads/accepted` points to it.

Any earlier failure is incomplete initialization, not provisional canonical state.

The parentless bootstrap revision is valid format v2. It contains:

- the normative v2 `architecture.yaml`, written as mode `100644`;
- one newly generated stable root Diagram ID named by manifest `root_diagram`; and
- one ordinary `diagrams/root.yaml` blob, written as mode `100644`, whose title is initially derived from the project name and whose appearances are empty.

Zero Components and zero Relationships are valid. `diagrams/root.yaml` is only WorkBraid's creation-time filename convention: it carries no identity and does not designate the root. Normal loading continues to accept any conforming non-recursive `diagrams/*.yaml` filename, and manifest `root_diagram` remains the sole root designation. After initialization, the root title is independently mutable and does not track later project-name changes.

Initialization generates one store UUID and one unique slug from the submitted project name. It publishes the project into the catalog only through the valid accepted bootstrap. It does not accept or retain a source-project path.

Opening distinguishes:

- absent store;
- incomplete or invalid store;
- valid store.

It never falls back to catalog state or another Git ref.

### Loaded snapshot

WorkBraid loads one immutable in-memory Architecture snapshot corresponding to one exact accepted commit.

The same snapshot supplies:

- the accepted format version and project slug;
- the Diagram hierarchy and composition;
- the active Diagram's map topology and derived boundary references;
- accepted component titles and documentation;
- relationship resolution;
- bases for newly created pending change sets.

Opening or explicit refresh:

1. resolves `refs/heads/accepted`;
2. reads its committed tree;
3. constructs and minimally validates the complete replacement snapshot;
4. switches to it only after successful construction.

The first slice has no filesystem watcher, polling loop, per-read ref resolution, or required persisted Architecture projection.

For v2, snapshot construction validates the complete Diagram set, unique home appearances, hierarchy, membership, Component references, and global Relationships before publication. Diagram tree, selected Diagram, map, index, documentation, boundary references, and topology always come from that one snapshot.

If `accepted` advances to invalid or unsupported state, WorkBraid may retain the previous valid snapshot for clearly marked stale, read-only reference. It must not present that snapshot as current accepted state or allow a direct commit from it as though its base remained current.

### Reviewed candidate snapshots

`Review changes` is one deliberate review task over the same complete candidate used by final confirmation. A successful review is bound to:

- the exact accepted base commit;
- the exact candidate tree;
- the pending change set's exact in-process generation.

The review contains one immutable snapshot reconstructed from the exact base commit and one immutable snapshot constructed from the validated candidate tree. Both use the existing version-aware Architecture parser and validation semantics. WorkBraid does not construct another candidate, graph, parser, or review authority for visual review.

Within review, the selected base or candidate snapshot supplies the Diagram tree, selected Diagram, component index, map topology, selected documentation/detail, titles, boundary references, and relationship resolution together. A surface must never combine data from the two snapshots. The base side remains the review's exact bound base; it is not replaced by newly observed accepted Architecture. If external authority moves after review, the review becomes stale under the existing stale-base rules.

If the current review selection is a Diagram that exists only in **With changes**, switching to **Before changes** uses a base-owned fallback rather than leaking candidate state. WorkBraid selects the nearest ancestor of that candidate Diagram which exists in the bound base snapshot; if no such ancestor survives, it selects the bound base root Diagram. The review shows a restrained note that the previously selected Diagram exists only with the changes. Its candidate-only composition, index, documentation context, boundary references, and topology never appear on the base side. Restoring exact focus when returning to **With changes** is disposable UI behavior rather than an Architecture invariant.

Invalid pending state does not produce reviewed snapshots. It remains non-canonical work under Changes in progress with actionable validation guidance.

## 5. Structural validation

Validation remains intentionally small.

A valid accepted Architecture requires:

- canonical manifest, Component, and Diagram paths that are ordinary Git blob entries rather than symlinks, gitlinks, or trees;
- a parseable closed v2 manifest with required store identity, project name, valid slug, and root Diagram identity;
- discovered Component files with parseable YAML frontmatter;
- valid and unique Component IDs;
- valid UTF-8 Markdown;
- a required first-block H1 title whose text is non-empty after trimming;
- parseable relationship declarations;
- resolvable Relationship target IDs;
- discovered Diagram files with the exact closed Diagram schema;
- valid and unique Diagram IDs and non-empty Diagram titles;
- appearances that resolve Component and optional detail-Diagram IDs;
- exactly one home appearance for every Component and no duplicate Component appearance within a Diagram;
- no detail link on a reference appearance;
- exactly one parent anchor for every non-root Diagram;
- complete reachability from root and an acyclic Diagram hierarchy.

No general schema, prose linting, policy engine, or speculative validation framework is introduced.

## 6. Pending change sets and direct commits

A first-slice pending Architecture change set is not a draft owned by one component.

Each pending change set:

- is based on one exact accepted Git revision;
- may eventually change multiple Components, Relationships, Diagrams, membership, hierarchy links, and canonical files coherently;
- remains non-canonical until deliberate compare-and-swap advancement of `refs/heads/accepted` succeeds;
- is owned by the local backend while the application is running;
- survives validation, commit creation, and ref-update failures that occur before the acceptance success boundary during that running application session.

Transient unsent browser edits are allowed, but the browser does not own the authoritative pending change set. Persistence and recovery of pending change sets across backend restart are deferred.

A human may explicitly discard the entire non-canonical pending change set. Discard removes only that pending state; it does not modify accepted Architecture, Git refs or objects, arbitrary user files, or persisted Architecture state. If a current accepted revision is successfully loaded, new pending work may then begin from it. Partial discard, merge, rebase, reconciliation, undo/redo, and a broader draft lifecycle remain deferred.

The canonical Git store remains unchanged until successful compare-and-swap advancement of `refs/heads/accepted`. Validation, commit creation, or ref-update failure before that boundary preserves:

- the previous accepted revision;
- the pending change set for continued editing or retry during the running application session.

Candidate construction starts from the exact base tree. Unchanged paths reuse their exact base-tree entries and blobs; only changed or newly created canonical files are serialized into new blobs. Editing an existing regular file preserves its regular-file mode; newly created canonical files use `100644`. Diagram-only and membership-only changes do not rewrite Component files. Structural, Diagram-hierarchy, membership, and Relationship validation runs against the complete resulting candidate tree before commit.

### Direct human commit flow

A human may directly update accepted Architecture without creating a proposal.

The backend:

1. constructs the complete candidate Architecture from the pending change set and its exact accepted base;
2. performs minimal structural validation and constructs the immutable candidate snapshot from that validated state;
3. generates an exact unified diff between the base and candidate Git trees;
4. gives the user an opportunity to inspect the complete diff;
5. on confirmation, verifies that `refs/heads/accepted` still equals the pending change set's exact base;
6. creates the successor commit;
7. atomically advances `refs/heads/accepted` from the base to the successor;
8. after successful advancement, marks the pending change set as committed/consumed and publishes the already-validated candidate snapshot under the successor commit identity.

The diff includes the entire pending change set, including canonical Component frontmatter, manifest, and Diagram-file changes. It is review evidence, not another canonical artifact. No semantic diff engine is required.

Visual review is assistive evidence derived from the same bound base and candidate snapshots. Candidate structural validation and the complete exact unified diff remain sufficient review evidence if visual rendering fails clearly. The visual review does not create an additional acceptance prerequisite or authority.

Successful atomic advancement of `refs/heads/accepted` is the acceptance success boundary. Once that compare-and-swap succeeds, the successor commit is canonical even if subsequent in-memory publication or the HTTP/UI response fails. WorkBraid must not treat that change as still uncommitted or offer to commit it again. A post-CAS publication failure is recovered by loading the revision named by `accepted`; restart and reopen independently prove reconstruction from canonical state.

If the atomic ref update fails:

- accepted Architecture has not changed;
- the pending change set is stale;
- WorkBraid does not silently overwrite newer accepted state.

Commit or object creation without successful compare-and-swap advancement is not accepted state. The previous accepted revision remains authoritative and the pending change set remains uncommitted.

After success, WorkBraid exposes the exact accepted commit identity and parent diff without requiring the SHA to dominate the normal UI.

Proposal approval is a separate future workflow.

## 7. First-slice authoring

The browser provides structured format-v2 Architecture authoring rather than raw-frontmatter editing as the normal flow.

Initial controls include:

- a structured Title field projecting the canonical H1;
- a Markdown body editor;
- read-only inspection/copying of the component ID;
- structured outgoing-relationship controls;
- relationship target selection by stable component identity.

Targets are presented primarily by human-readable title. Titles need not be globally unique, so the UI provides enough context to disambiguate duplicate titles.

Creating a component generates:

- its immutable UUID;
- minimal YAML frontmatter;
- its required H1;
- a human-readable filename.

In v2, creation also adds exactly one home appearance. The active Diagram is the home when creation occurs in a Diagram context; root is the fallback only when creation genuinely has no Diagram context.

Filename generation is creation-time behavior only. Loading accepts any filename that matches the non-recursive Component discovery rule. Changing the title does not automatically rename the file.

Initial UI does not require:

- component deletion;
- file renaming;
- deliberate identity replacement;
- raw-frontmatter editing.

All controls edit the pending change set, never canonical Git directly.

For an existing component, a Description-only edit preserves the H1 bytes exactly. If a submitted normalized Title is unchanged, its existing H1 bytes are also preserved exactly. If the Title changes, WorkBraid replaces the H1 using the plain-text Title projection and serialization rules above; it does not attempt to preserve inline Markdown formatting that the structured editor does not expose. The existing ATX or Setext heading form is preserved unless doing so would conflict with the Title round-trip invariant.

### Initial Diagram authoring

Normal Diagram authoring is structured composition editing rather than raw YAML editing. The first Diagram slice may:

- create a detail Diagram from a Component's home appearance;
- set or change a Diagram title;
- move a Component's home to another Diagram;
- add or remove a reference appearance;
- navigate through the root, Diagram tree, breadcrumbs, parent anchor, and detail link.

These operations update the one backend-owned pending Architecture change set. Diagram deletion, partial pending discard, multiple detail Diagrams per anchor, multiple Diagram parents, and general hierarchy lifecycle are not part of the first Diagram slice.

## 8. Accepted Diagram map and candidate review map

The normal workspace map is an interactive projection of one exact accepted Architecture revision: the selected accepted Diagram's exact composition plus derived cross-Diagram boundary references.

A v2 workspace begins at the root Diagram and provides the complete accepted Diagram tree. Selecting a Diagram switches its Component index, map, documentation context, ordinary Relationships, and derived boundary references together. Selecting a Component focuses its accepted documentation. Activating a home Component's detail link drills into its child Diagram; breadcrumbs and back navigation return to the parent and identify the anchor. Activating a derived boundary reference navigates to the external Component's home Diagram.

Product presentation distinguishes a canonical reference appearance, meaning the Component is **also shown here**, from a derived boundary reference, meaning the related Component **lives elsewhere**. Neither wording changes Component or Relationship semantics.

Multiple Relationships between the same Components remain representable and inspectable. A component inspector may simultaneously show pending edits from a change set based on that accepted revision, but the normal accepted map does not preview pending topology or pending Diagram composition.

The map and Diagram tree rebuild only when accepted state advances successfully or an accepted revision is explicitly reloaded.

`Review changes` may instead display the exact immutable reviewed candidate snapshot and allow a compact switch to the review's exact immutable base snapshot. The candidate map is the primary visual review canvas. Switching snapshots switches the Diagram tree, selected Diagram, index, map, selected documentation/detail, titles, canonical appearances, boundary references, and relationship topology as one revision-pinned unit. The normal workspace map remains accepted-only.

Visual change matching follows these rules:

- components match by stable component ID;
- relationship facts compare as a multiset of `(source component ID, target component ID, exact relationship label)`, including multiplicity;
- a relationship target or label edit appears as one removed fact and one added fact;
- identical parallel facts remain semantically indistinguishable;
- projection-only edge keys may distinguish rendered edges but have no Architecture meaning.

Component deletion is not part of this slice, so review does not introduce removed-component semantics or visualization. Review may distinguish added components and relationships, changed existing components, and removed relationship facts. Selecting a changed component or relationship may focus its review context and the relevant region of the exact unified diff.

For v2 review:

- Diagrams match by stable Diagram ID;
- added Diagrams, title changes, home/reference appearance additions or removals, home moves, and detail-link changes are Diagram-composition changes;
- a home move is a removal from one Diagram and addition to another, not a Component identity or Relationship change;
- an ordinary internal edge becoming a boundary edge, or the reverse, solely because composition changed is presentation change and is not reported as an Architecture Relationship addition or removal;
- the exact canonical diff remains authoritative evidence for manifest and Diagram-file changes.

When a selected Diagram exists only in the candidate, **Before changes** follows the base-owned fallback defined for reviewed candidate snapshots: nearest surviving ancestor, otherwise the bound base root. Candidate-only Diagram state never leaks into the base projection.

Selection, viewport, and automatic-layout details are UI state, not canonical Architecture. Pan, zoom, and fit are desirable initial UX rather than domain invariants.

Review layout is deterministic and stable-ID-aware so unchanged components do not move gratuitously between the bound base and candidate views. Reusing transient positions from an already rendered base map within the current browser session is allowed, but coordinates remain unpersisted UI state and have no effect on review correctness.

Deferred map behavior includes:

- graphical creation or editing;
- draft-topology preview;
- manual or persisted layout;
- grouping;
- relationship editing on the map;
- Diagram deletion or general hierarchy lifecycle;
- multiple Diagram parents or Diagram reuse;
- multiple appearances of one Component within one Diagram;
- runtime or Planning overlays;
- source-code inference.

## 9. Project catalog and locator

WorkBraid discovers projects directly from its private Architecture-store area. A WorkBraid-created repository lives operationally at `architecture/<store-uuid>.git` under application data. That directory naming is an implementation convention, not portable project identity or a human-facing locator.

For each discovered store, the catalog resolves `refs/heads/accepted` and loads the exact accepted snapshot through the one Architecture loader. Valid accepted manifest state supplies the display name, current slug, and canonical store UUID. The catalog does not use another database, index, registry, fallback ref, working tree, or cached Architecture projection as authority.

Catalog slugs must be unique. If more than one valid discovered store has the same slug, WorkBraid reports an explicit catalog conflict and opens neither by that slug; filesystem enumeration order never selects a winner. A discovered UUID-named store whose accepted state is missing, malformed, or unsupported is surfaced as an unavailable store with bounded technical identity where practical rather than silently treated as a valid project or selected through partial parsing. This is visibility, not repair or recovery machinery.

`/projects/<slug>` resolves the one valid catalog entry with that exact slug. An unknown slug produces a normal not-found state and never initializes, repairs, or selects a project implicitly. After resolution, the backend uses the manifest store UUID as canonical store identity; the route slug does not replace it.

Project creation accepts a human-readable name, generates a new store UUID and locally unique slug, creates the native-v2 bootstrap, and exposes the project in the catalog only as valid accepted state. Project names need not be unique. No source folder, source-root association, reassociation, arbitrary project filesystem root, or second catalog persistence exists.

## 10. Initial implementation profile

These are v0 implementation choices, not portable store semantics.

### Application shape

- One local Go backend process.
- One loopback-only, single-user browser application.
- Prefer one local origin, with the backend serving the browser UI.
- Modular monolith.
- No authentication, tailnet exposure, mobile-specific behavior, or multi-user behavior in the first slice.
- No separately deployed frontend/backend services, CORS architecture, or public API requirement.

The backend owns:

- Git and filesystem authority;
- accepted snapshot loading;
- pending change-set bases;
- candidate construction;
- structural, Diagram-composition, hierarchy, and Relationship validation;
- exact diff generation;
- stale-base checking;
- commit creation;
- atomic `accepted` advancement.

The browser is an authoring, rendering, diff-review, and map client. It does not independently implement canonical Architecture transitions.

Go is not part of the canonical store contract. No backend framework, generic persistence architecture, or generic VCS interface is implied.

### Git access

The Go backend uses a compatible real Git executable through fixed direct process invocations.

Rules:

- never invoke Git through a shell;
- run Git in a controlled, non-interactive environment;
- do not unexpectedly execute hooks, editors, pagers, signing, external diff behavior, or user presentation configuration;
- provide operation-relevant Git identity/configuration explicitly;
- treat stderr and localized Git text as diagnostics only;
- derive domain classifications from explicit ref/object observation;
- never allow browser/API input to become arbitrary Git arguments or subcommands.

Review is based on exact base and candidate trees using a predictable unified diff. Exact rendered diff bytes are not canonical Architecture.

Tests use temporary real Git repositories and the real Git executable. No fake Git API is introduced.

### Repository layout

The initial private Architecture repository is bare and has no permanent working tree.

WorkBraid:

- reads accepted state from committed objects;
- constructs temporary candidate state separately;
- does not synchronize a permanent checkout after commits;
- ignores arbitrary checkout state when determining authority.

WorkBraid does not configure or require the bare repository's `HEAD` to point to `accepted`.

Bare layout is an implementation choice, not part of the portable store contract.

Bare repositories remain compatible with ordinary branches and linked worktrees. Future proposal branches may coexist with `accepted`, and agents may later receive linked worktrees. Exact proposal representation and acceptance semantics remain undecided.

### Persistence boundary

The current Architecture product has no demonstrated SQLite need and does not initialize or depend on SQLite. Project discovery comes from private Git stores, and no Architecture, Diagram, catalog, pending, review, navigation, or layout projection is persisted outside them.

Future recoverable draft persistence or another demonstrated operational need requires a separate approved decision. Any persisted derived state introduced later must identify its exact canonical revision and be rebuildable from Git.

## 11. First real gate

Using the real WorkBraid application through production code paths:

1. Start with fresh WorkBraid application data and no projects.
2. Create a project by human-readable name.
3. Verify its generated valid unique slug, `/projects/<slug>` route, immutable store UUID, and catalog entry.
4. Verify the parentless format-v2 bootstrap commit, its manifest-identified empty root Diagram, and the `accepted` ref.
5. Verify that `architecture.yaml` contains stable store identity, project name, current slug, and root Diagram identity while `diagrams/root.yaml` carries no authority beyond its canonical contents.
6. Create a tiny Architecture through WorkBraid.
7. Review and deliberately commit its exact candidate diff.
8. See the accepted map and navigate Component documentation.
9. Edit accepted Architecture through a pending change set, verify that validation, commit creation, or ref-update failure before successful compare-and-swap preserves it while the application remains running, and then commit a valid accepted revision.
10. Verify the exact resulting commit identity and parent diff.
11. Reload `/projects/<slug>` and verify it resolves the same store and revision.
12. Restart WorkBraid after the accepted commit.
13. Reopen the route and reconstruct the same Architecture, documentation, Relationships, Diagrams, and map solely from private Git. Recovery of an uncommitted pending change set across backend restart is not part of this gate.
14. Verify that no source-root association, SQLite catalog, or other Architecture projection exists.

Use real private Git repositories, real filesystem state, and the real backend-to-Git path. Focused tests may support the gate, but fake Git APIs and large synchronization simulators do not satisfy it.

## 12. Deferred and open decisions

Deferred beyond the first slice:

- proposal representation, proposal branches/refs, review, conflict handling, and acceptance semantics;
- automatic merge or reconciliation behavior;
- persistence and recovery of pending change sets across backend restart;
- stale-change-set reconciliation UX;
- non-component Architecture documents;
- component deletion and inbound-relationship handling;
- file-renaming UI;
- deliberate identity replacement;
- raw-frontmatter editing;
- stable relationship identity or relationship lifecycle;
- relationship taxonomy, hierarchy, and grouping;
- draft-aware or graph-based editing;
- persisted/manual map layout;
- node sizing, edge routing and bend points, shapes, Diagram annotations, and renderer state;
- project-scoped persistence of pending spatial edits;
- Diagram deletion and general hierarchy lifecycle;
- multiple Diagram parents or reusable Diagram DAGs;
- multiple appearances of one Component inside one Diagram;
- multiple detail Diagrams per anchor;
- Diagram-local presentation identity for parallel rendered edges;
- additional Diagram kinds and kind-specific semantics;
- alternate renderers such as isometric presentation;
- full revision-history browsing;
- arbitrary revision comparison;
- revert UI;
- semantic diffing;
- project rename and slug-change lifecycle;
- project deletion, import, unavailable-store recovery, and custom private-store locations;
- repository export/synchronization and external divergence handling;
- Jira, Linear, or other external surfaces;
- remote/embedded Markdown resource behavior;
- authentication, tailnet exposure, mobile UX, and multi-user behavior;
- any persisted Architecture projection not justified by a demonstrated need.
