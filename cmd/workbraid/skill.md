# WorkBraid Architecture agent guide

WorkBraid has one Accepted Architecture and durable, named, independent change sets. The running loopback WorkBraid process is the only authority for the current project, Accepted revision, change-set records, review, and update. CLI and MCP are thin clients of that process. Never open the private Git store or edit canonical YAML directly.

Use `workbraid [--server http://127.0.0.1:8080] --json …`. `WORKBRAID_SERVER` supplies the URL when `--server` is absent. Global flags precede the command. Start with `status`; `connection_failed` means the operator must start WorkBraid or correct the URL.

## Identity and exact state

- A project slug locates a catalog entry; its store UUID is project identity.
- Accepted Architecture is singular. `architecture inspect` returns its exact revision, Components, Relationships, Diagram hierarchy, homes, references, and IDs.
- A change-set UUID is proposal identity. Its mutable name is display text only. Never select by name.
- `change-set list --store-id <uuid>` returns active, applied, and unavailable records. `change-set inspect` returns one exact record: lifecycle, name, original base, generation, proposal Markdown, concrete facts, validity, candidate, review, out-of-date state, and applied revision.
- Every structured proposal mutation supplies the exact `store_id`, `change_set_id`, and that record's `generation`. After a successful mutation, use its returned generation or inspect that exact ID again.
- Copy stable IDs and review fields from results. Never invent IDs or infer one proposal from browser selection.

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

Accepted and change-set reads:

```text
workbraid architecture inspect
workbraid architecture refresh --store-id <uuid> --accepted-revision <sha>
workbraid change-set list --store-id <uuid>
workbraid change-set create --store-id <uuid> --accepted-revision <sha> [--name <text>]
workbraid change-set inspect --store-id <uuid> --change-set-id <uuid>
```

Change-set context:

```text
workbraid change-set rename <state> --name <text>
workbraid change-set edit-proposal <state> (--proposal <markdown>|--proposal-file <path|->)
workbraid change-set review <state>
workbraid change-set discard <state>
workbraid architecture update --store-id <uuid> --change-set-id <uuid> --base-revision <sha> --candidate-tree <tree> --generation <n>
```

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

Here `<state>` is `--store-id <uuid> --change-set-id <uuid> --generation <n>`. Component creation may omit `--diagram-id` only for deliberate root fallback. File values are exact UTF-8; `-` reads stdin; literal and file forms are mutually exclusive. Relationship edit/remove uses the exact raw source/target/label plus one-based occurrence. Empty or malformed raw selectors remain valid inputs so invalid rows can be repaired without deleting the change set.

Creation starts at generation 0 with a valid candidate equal to its exact Accepted base. Rename, proposal edit, and semantic edits increment only that change set and invalidate only its Review. `change-set review` returns the complete unified Architecture diff, Before/With projections, exact ID/base/tree/generation binding, and a `review_url`; give that URL to the reviewer. `architecture update` accepts only that binding. There is no force, accept-latest, automatic review, rebase, merge, or combined mutate-and-accept action.

An out-of-date active change set keeps its original base, remains editable and reviewable, and cannot update Accepted. Reconciliation is a future product. Applied change sets are immutable evidence. Discard deletes one whole active change set at its exact generation and changes neither Accepted nor any other record.

## Recovery by error code

- `invalid_request`: correct missing or malformed fields. All mutation state fields are required.
- `connection_failed`: start/check the configured server; do not access application data.
- `incompatible_server`: use matching WorkBraid v2 client/server binaries.
- `project_not_found`: list projects, then use the exact slug or create deliberately.
- `project_conflict`, `project_unavailable`: do not bypass the catalog or store authority.
- `project_not_open`, `project_mismatch`: inspect/open the intended project, then inspect again.
- `change_set_not_found`: list change sets for the exact store and copy the UUID.
- `change_set_unavailable`: the record is malformed; Accepted and other records remain usable.
- `change_set_name_conflict`: choose a different active name.
- `change_set_generation_mismatch`: inspect that exact change-set ID and retry from its current generation.
- `change_set_not_editable`: the record is applied and read-only.
- `change_set_out_of_date`: keep editing/reviewing against its original base or wait for a future reconciliation capability; do not retry Update.
- `architecture_non_current`, `accepted_conflict`: Refresh and inspect Accepted plus the preserved change sets. WorkBraid does not reconcile.
- `refresh_failed`: authority is indeterminate; preserve prior knowledge and retry Refresh explicitly.
- `target_not_found`: inspect Accepted and the addressed change set for stable IDs or the exact raw selector.
- `target_not_eligible`: choose an allowed target from the inspected proposal context.
- `validation_blocked`: use the returned stable location and raw values to repair the addressed change set.
- `review_required`: inspect and Review the exact current generation.
- `review_invalidated`: inspect, then Review again.
- `acceptance_uncertain`: inspect/list/Refresh before any retry; an applied record is the durable receipt.
- `accepted_reload_required`: acceptance succeeded; do not Update again. Refresh or reopen.
- `unsupported_action`: do not substitute raw Git/YAML operations.
- `operation_failed`: inspect current authoritative state before retrying.

## Small JSON workflow

Every value below is copied from the preceding JSON result:

```text
workbraid --json project create --name "Agent example"
workbraid --json architecture inspect
workbraid --json change-set create --store-id <store> --accepted-revision <revision> --name "Gateway proposal"
workbraid --json change-set edit-proposal --store-id <store> --change-set-id <change> --generation 0 --proposal "# Gateway proposal"
workbraid --json component create --store-id <store> --change-set-id <change> --generation 1 --diagram-id <root> --title Gateway --description "Entry point"
workbraid --json component create --store-id <store> --change-set-id <change> --generation 2 --diagram-id <root> --title Worker
workbraid --json relationship add --store-id <store> --change-set-id <change> --generation 3 --source-id <gateway> --target-id <worker> --label calls
workbraid --json change-set inspect --store-id <store> --change-set-id <change>
workbraid --json change-set review --store-id <store> --change-set-id <change> --generation 4
workbraid --json architecture update --store-id <store> --change-set-id <change> --base-revision <review-base> --candidate-tree <review-tree> --generation 4
workbraid --json architecture inspect
```

For MCP, launch `workbraid [--server <loopback-url>] mcp`. Its typed tools implement the same explicit records and exact preconditions against the same running process.
