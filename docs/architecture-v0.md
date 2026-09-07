# WorkBraid Architecture

Status: Approved living contract

Phase 3.3 is authorized by delegated root approval of proposal `d119baeb-1b2b-4451-8113-7f4fe05678ec`, generation 8, reviewed state `595bab6de6933c7a047c70520f1830594f166fed`. [Placement §10](architecture-placement-amendment-v0.md#10-deliberate-link-routing-phase-33) defines the closed portable v5 route extension, presentation-slot address, geometry and transitions. Native initialization uses v5; supported v2–v4 remain exactly readable and ordinarily writable without implicit upgrade. This supersedes earlier native-version and routing-exclusion statements. Relationships retain their exact directed multiset semantics and have no stable identity. Routing belongs to Diagram presentation; it changes no catalog, acceptance or runtime authority.

Phase 3.2 extends the portable foundation with complete v4 sizes under [Placement §9](architecture-placement-amendment-v0.md#9-complete-visible-node-sizing-phase-32). Native initialization now writes version 4; v2/v3 remain supported exactly. This supersedes native-v3 statements below, without changing catalog, source fidelity or acceptance authority.

This document owns the shared Architecture semantics, portable v2 foundation, source fidelity and runtime/catalog authority. [Placement](architecture-placement-amendment-v0.md) defines the approved complete v3 extension; [Proposals and Reviews](architecture-proposals-v0.md) owns durable authoring and feedback; [Reconciliation](architecture-reconciliation-v0.md) owns deliberate residual construction; [UI](ui-v0.md) owns presentation. The [roadmap](roadmap.md) records later boundaries; completed execution plans remain in Git history.

## Domain boundary

WorkBraid has three related verticals:

- Architecture owns architecture knowledge, components, relationships, visualization, accepted revisions, and the Architecture proposal lifecycle.
- Agent Control owns Herdr, sessions, workspaces, terminals, messaging, and runtime state.
- Planning owns work items, dependencies, gates, readiness, waves, and work-item review/integration/validation state.

Only Architecture is currently in scope. It must be buildable and usable without Planning, Herdr, or Agent Control.

Agents may act across verticals through authorized operations. Shared agent access does not merge domain models. Planning may originate an Architecture proposal but cannot directly mutate accepted Architecture.

## Canonical authority

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

## Store contract

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

Format and version are compatibility guards. WorkBraid rejects unsupported values rather than interpreting them using current assumptions. Portable v2 remains supported and normally writable for non-placement work. Approved v3 adds complete visible-node positions under the placement contract. Portable v1 remains unsupported; do not confuse it with supported operational change-state v1/v2 over portable v2. The rejected partial-pinning v3 trial is preserved as evidence but has no compatibility requirement. No automatic conversion or deletion of that trial data is authorized.

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

## Initialization and loading

### Bootstrap revision

Architecture-store initialization is a deliberate authorized project-creation action, through the browser or public agent interface.

Initialization succeeds only when:

1. a valid bootstrap commit exists; and
2. `refs/heads/accepted` points to it.

Any earlier failure is incomplete initialization, not provisional canonical state.

The parentless bootstrap revision is native format v3 under the placement contract. It contains:

- the manifest defined above with `version: 3`, written as mode `100644`;
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

There is no filesystem watcher or polling loop. Reads do not implicitly Refresh. Mutations and explicit authority observations recheck real refs under their exact preconditions; loaded snapshots and caches never replace Git authority.

For v2, snapshot construction validates the complete Diagram set, unique home appearances, hierarchy, membership, Component references, and global Relationships before publication. Diagram tree, selected Diagram, map, index, documentation, boundary references, and topology always come from that one snapshot.

If `accepted` advances to invalid or unsupported state, WorkBraid may retain the previous valid snapshot for clearly marked stale, read-only reference. It must not present that snapshot as current accepted state or allow a direct commit from it as though its base remained current.

### Reviewed candidate snapshots

`Review changes` is one deliberate review task over the same complete candidate used by final confirmation. A successful review is bound to:

- the proposal's exact base commit;
- the exact candidate tree;
- the addressed durable Change Set's exact generation.

The review contains one immutable snapshot reconstructed from the exact base commit and one immutable snapshot constructed from the validated candidate tree. Both use the existing version-aware Architecture parser and validation semantics. WorkBraid does not construct another candidate, graph, parser, or review authority for visual review.

Within review, the selected base or candidate snapshot supplies the Diagram tree, selected Diagram, component index, map topology, selected documentation/detail, titles, boundary references, and relationship resolution together. A surface must never combine data from the two snapshots. The base side remains the review's exact bound base; it is not replaced by newly observed accepted Architecture. If external authority moves after review, the review becomes stale under the existing stale-base rules.

If the current review selection is a Diagram that exists only in **With changes**, switching to **Before changes** uses a base-owned fallback rather than leaking candidate state. WorkBraid selects the nearest ancestor of that candidate Diagram which exists in the bound base snapshot; if no such ancestor survives, it selects the bound base root Diagram. The review shows a restrained note that the previously selected Diagram exists only with the changes. Its candidate-only composition, index, documentation context, boundary references, and topology never appear on the base side. Restoring exact focus when returning to **With changes** is disposable UI behavior rather than an Architecture invariant.

Invalid pending state does not produce reviewed snapshots. It remains non-canonical work under Changes in progress with actionable validation guidance.

## Structural validation

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

## Structured authoring

The browser provides structured Architecture authoring rather than raw-frontmatter editing. Every kept change belongs to one named durable proposal; Accepted-origin editing creates its first mutation atomically under exact store/revision preconditions.

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

For an existing component, a Description-only edit preserves the H1 source bytes exactly, with one structural exception: if the preserved H1 block has no terminating line break and the exact Description to serialize has non-zero byte length, WorkBraid appends exactly one LF byte (`\n`) after the H1, then appends the exact Description bytes unchanged. The LF terminates the heading; it is not part of the Description and must not consume, trim, normalize, or replace any Description byte. An exactly empty Description leaves the unterminated H1 unchanged. Whitespace-only or newline-only Description is non-empty and requires the terminator. An already-terminated H1 gains no extra separator. This rule applies to both ATX and Setext H1 blocks; the newly required terminator is always LF, while the Description's own LF/CRLF and leading whitespace remain exact. It does not normalize unrelated Markdown or frontmatter.

If a submitted normalized Title is unchanged, its existing H1 source is preserved under the same rule. If the Title changes, WorkBraid replaces the H1 using the plain-text Title projection and serialization rules above; it does not attempt to preserve inline Markdown formatting that the structured editor does not expose. The existing ATX or Setext heading form is preserved unless doing so would conflict with the Title round-trip invariant.

The structural exception exists solely to preserve the ordinary structured-edit round trip: reparsing serialized source must recover the intended normalized Title and the intended exact Description bytes. For example, preserved H1 `# API` plus Description `\nBody\n` serializes as `# API\n\nBody\n`, not `# API\nBody\n`. No raw-source override, storage field, or separate Markdown interpretation is introduced.

### Diagram authoring

Normal Diagram authoring is structured composition editing rather than raw YAML editing. The first Diagram slice may:

- create a detail Diagram from a Component's home appearance;
- set or change a Diagram title;
- move a Component's home to another Diagram;
- add or remove a reference appearance;
- reassign a detail Diagram to another eligible home Component through the ordinary [reassignment contract](architecture-reconciliation-v0.md#ordinary-detail-link-reassignment); and
- navigate through the root, Diagram tree, breadcrumbs, parent anchor, and detail link.

These operations update only the explicitly addressed backend-owned durable Change Set. Diagram deletion, partial discard, multiple detail Diagrams per anchor, multiple Diagram parents, and general hierarchy lifecycle remain outside ordinary authoring.

## Project catalog and locator

WorkBraid discovers projects directly from its private Architecture-store area. A WorkBraid-created repository lives operationally at `architecture/<store-uuid>.git` under application data. That directory naming is an implementation convention, not portable project identity or a human-facing locator.

For each discovered store, the catalog resolves `refs/heads/accepted` and loads the exact accepted snapshot through the one Architecture loader. Valid accepted manifest state supplies the display name, current slug, and canonical store UUID. The catalog does not use another database, index, registry, fallback ref, working tree, or cached Architecture projection as authority.

Catalog slugs must be unique. If more than one valid discovered store has the same slug, WorkBraid reports an explicit catalog conflict and opens neither by that slug; filesystem enumeration order never selects a winner. A discovered UUID-named store whose accepted state is missing, malformed, or unsupported is surfaced as an unavailable store with bounded technical identity where practical rather than silently treated as a valid project or selected through partial parsing. This is visibility, not repair or recovery machinery.

`/projects/<slug>` resolves the one valid catalog entry with that exact slug. An unknown slug produces a normal not-found state and never initializes, repairs, or selects a project implicitly. After resolution, the backend uses the manifest store UUID as canonical store identity; the route slug does not replace it.

Project creation accepts a human-readable name, generates a new store UUID and locally unique slug, creates the native-v3 bootstrap, and exposes the project in the catalog only as valid accepted state. Project names need not be unique. No source folder, source-root association, reassociation, arbitrary project filesystem root, or second catalog persistence exists.

## Implementation profile

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

Bare repositories remain compatible with ordinary branches and linked worktrees. Product proposals use the exact private ref/envelope representation in the Proposals and Reviews contract. A checkout or worktree is never a product-authoring authority.

### Persistence boundary

The current Architecture product has no demonstrated SQLite need and does not initialize or depend on SQLite. Project discovery comes from private Git stores, and no Architecture, Diagram, catalog, pending, review, navigation, or layout projection is persisted outside them.

Durable proposals, immutable reviews and v3 placement are stored only in their approved Git trees/refs. Any later operational persistence requires a demonstrated need and an approved decision; derived state must identify its exact canonical revision and be rebuildable from Git.


## Candidate fidelity and acceptance boundary

Construct from the exact base tree. Untouched paths reuse their exact entries/blobs; changed existing regular files retain their modes; new files use `100644`. Composition alone does not rewrite Component files. Validate the complete candidate, never a partial graph. `ConstructCandidate(base, concrete facts).Tree` must equal the stored candidate tree for every supported valid active/applied/historical record. No parallel builder, raw blob override or compatibility rewrite may weaken that equality.

Only deliberate Update of a valid current exact review can advance Accepted, using the atomic accepted/active/applied ref transaction defined in the Proposals and Reviews contract. Pre-CAS failure retains the original durable proposal and Accepted. Post-CAS publication/response failure cannot make a successful update uncommitted: recover the actual applied receipt and authoritative Accepted context, never blindly replay. Commit creation without ref publication does not accept anything. The complete canonical diff remains available even if visual rendering fails; this technical review path does not waive the active feature's human visual acceptance gate.
