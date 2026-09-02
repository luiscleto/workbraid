# WorkBraid Architecture agent guide

WorkBraid keeps accepted Architecture as documented Components, Relationships, and nested Diagrams. One running WorkBraid server is the only authority for the current project, accepted revision, pending changes, review, and update. CLI and MCP commands are loopback clients of that process. Never edit WorkBraid's private Git stores or canonical YAML directly.

Use `workbraid [--server http://127.0.0.1:8080] --json …`. `WORKBRAID_SERVER` supplies the server URL when `--server` is absent. Global flags always precede the command. Start with `status`; a connection error means the authoritative server must be started or checked, never that the CLI should open a store itself.

## Identity and safe state

- A project slug locates a catalog entry. Its store UUID is the exact project identity.
- Component and Diagram titles are labels; use their stable IDs for authoring.
- `architecture inspect` returns the accepted revision, exact Markdown, Diagram tree, homes, reusable references, Relationships, and boundary context.
- `changes inspect` returns the one pending base, generation, raw authored rows, validation, candidate when valid, and current review identity.
- Every authoring command requires `--store-id`, `--accepted-revision`, and `--generation`. Use `--generation none` only when inspection reported no pending set. After every mutation, use the returned generation or inspect again.

## Commands

Connection and projects:

```text
workbraid status
workbraid project list
workbraid project current
workbraid project create --name <name>
workbraid project open --slug <slug>
workbraid project close --store-id <uuid>
```

Reads, Refresh, review, discard, and deliberate update:

```text
workbraid architecture inspect
workbraid architecture refresh --store-id <uuid> --accepted-revision <sha>
workbraid changes inspect
workbraid changes review --store-id <uuid> --accepted-revision <sha> --generation <n>
workbraid changes discard --store-id <uuid> --generation <n>
workbraid architecture update --store-id <uuid> --base-revision <sha> --candidate-tree <tree> --generation <n>
```

`changes review` validates and returns the complete unified diff, immutable Before/With projections, comparison context, and exact base/tree/generation binding. Inspect it. `architecture update` accepts only that exact binding; there is no force, accept-latest, automatic review, or combined mutate-and-accept command. Continue editing by inspecting and using another structured mutation; this invalidates the old review. Discard removes the entire pending set only and requires its exact generation.

Structured authoring:

```text
workbraid component create <state> --title <text> [--description <markdown>|--description-file <path|->] [--diagram-id <uuid>]
workbraid component edit <state> --component-id <uuid> [--title <text>] [--description <markdown>|--description-file <path|->]
workbraid component move-home <state> --component-id <uuid> --diagram-id <uuid>
workbraid relationship add <state> --source-id <uuid> --target-id <raw> (--label <raw>|--label-file <path|->)
workbraid relationship edit <state> --source-id <uuid> --old-target-id <raw> (--old-label <raw>|--old-label-file <path|->) [--occurrence <n>] --target-id <raw> (--label <raw>|--label-file <path|->)
workbraid relationship remove <state> --source-id <uuid> --target-id <raw> (--label <raw>|--label-file <path|->) [--occurrence <n>]
workbraid diagram create-detail <state> --component-id <uuid> --title <text>
workbraid diagram edit-title <state> --diagram-id <uuid> --title <text>
workbraid diagram show-component <state> --diagram-id <uuid> --component-id <uuid>
workbraid diagram stop-showing-component <state> --diagram-id <uuid> --component-id <uuid>
```

Here `<state>` means `--store-id <uuid> --accepted-revision <sha> --generation <n|none>`. Pass `--diagram-id` when Component creation has a known Diagram context; omission deliberately uses the root. File values are read as exact UTF-8; `-` reads stdin. Literal and file forms are mutually exclusive. Relationship edit/remove selects source ID plus exact raw old target, exact raw old label, and the one-based occurrence among identical raw pairs. Empty selector values are valid when the flag is explicitly present, so malformed or incomplete pending rows can be repaired without Discard.

## Recovery by error code

- `connection_failed`: start/check the configured server; do not access application data.
- `incompatible_server`: use matching WorkBraid client/server binaries.
- `project_not_open`, `project_mismatch`: run `project current`, list/open deliberately, then inspect.
- `pending_generation_mismatch`: run `changes inspect` and retry with its exact generation.
- `architecture_non_current`, `accepted_conflict`: Refresh, inspect accepted and preserved pending state, then decide whether to discard. WorkBraid does not reconcile.
- `refresh_failed`: authority could not be determined; preserve prior knowledge and retry Refresh explicitly.
- `validation_blocked`: use the returned stable location and raw pending values to correct the structured row.
- `review_required`, `review_invalidated`: inspect and run a fresh Review before Update.
- `acceptance_uncertain`: inspect/Refresh before any retry.
- `accepted_reload_required`: acceptance definitely succeeded; never retry Update, Refresh or reopen the canonical result.
- `pending_conflict`: inspect pending work and discard the whole set only when deliberate.

## Small JSON workflow

```text
workbraid --json project create --name "Agent example"
workbraid --json architecture inspect
workbraid --json component create --store-id <store> --accepted-revision <revision> --generation none --diagram-id <root> --title Gateway --description "Entry point"
workbraid --json component create --store-id <store> --accepted-revision <revision> --generation 1 --diagram-id <root> --title Worker
workbraid --json relationship add --store-id <store> --accepted-revision <revision> --generation 2 --source-id <gateway> --target-id <worker> --label calls
workbraid --json diagram create-detail --store-id <store> --accepted-revision <revision> --generation 3 --component-id <gateway> --title Runtime
workbraid --json component move-home --store-id <store> --accepted-revision <revision> --generation 4 --component-id <worker> --diagram-id <runtime>
workbraid --json diagram show-component --store-id <store> --accepted-revision <revision> --generation 5 --diagram-id <runtime> --component-id <gateway>
workbraid --json changes inspect
workbraid --json changes review --store-id <store> --accepted-revision <revision> --generation 6
workbraid --json architecture update --store-id <store> --base-revision <review-base> --candidate-tree <review-tree> --generation 6
workbraid --json architecture inspect
```

For MCP clients, launch `workbraid [--server <loopback-url>] mcp`. Its typed tools implement these same reads, preconditions, review, and update against the same process-wide current project.
