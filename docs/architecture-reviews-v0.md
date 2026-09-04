# Architecture Reviews v0

Status: Approved

Scope: durable review feedback on exact Architecture Change Set revisions

This document adds review submissions to the completed Change Sets 1 and Agent Access 1 product. It does not change Accepted Architecture, Change Set authoring, `Review changes`, reconciliation, or rich Diagram state.

## 1. Product boundary

`Review changes` remains the existing operation which validates one active Change Set and durably binds its exact:

- Change Set UUID;
- base commit;
- candidate tree; and
- generation.

It continues to produce the exact Before/With projections and complete canonical diff used by deliberate acceptance. Agent Access `change-set review` / `change_set_review` retains that meaning.

Reviews 1 adds a separate immutable **review submission**. A submission is descriptive feedback on one already-bound exact Change Set state. It never constructs a candidate, changes a proposal, advances Accepted, or participates in acceptance authority.

Each submission owns:

- one immutable lower-case review UUID;
- the exact Change Set UUID and exact reviewed Change Set state commit;
- the exact base commit, candidate tree, and generation binding stored in that state;
- one verdict: `comment`, `approve`, or `request_changes`;
- one optional exact UTF-8 Markdown overall body;
- zero or more immutable anchored comments; and
- minimal descriptive, untrusted author provenance.

A submitted review is immutable. Further feedback is another submission. There is no draft-review persistence, edit, deletion, thread, reply, resolution, reaction, latest/superseded state, approval count, required reviewer, or review workflow.

Verdicts are informational conclusions only. They do not gate editing, `Review changes`, acceptance, discard, or any other Architecture transition. WorkBraid computes no aggregate approval state. A person may deliberately accept a still-current exactly reviewed proposal regardless of any submitted verdict.

## 2. Identity and descriptive provenance

Review and comment UUIDs are generated exactly once by WorkBraid when submission begins. They are identity; author labels, timestamps, verdicts, bodies, and anchors are not identity.

The author model is deliberately small:

- `author` is a required, trimmed, non-empty, single-line display label supplied by the submitting client; and
- `submitted_at` is the running WorkBraid process's UTC RFC 3339 timestamp at submission.

The normalized author label is stored exactly after trimming. The timestamp and label are descriptive provenance, not authenticated identity, trusted ordering, authorship proof, or authorization. UI and agent documentation must not call a label a verified user or imply that WorkBraid knows who controlled the browser, CLI, or MCP client. Review lists order by `submitted_at` newest first with review UUID as a deterministic tie-breaker; that order has no review-authority meaning.

An anchored comment body must be valid UTF-8 and non-empty after trimming; its submitted bytes are otherwise retained exactly. The overall review body may be empty.

For verdict `comment`, at least one of the overall body being non-empty after trimming or one anchored comment is required. A completely empty `comment` submission is `invalid_request` and creates no review ref. Verdict-only `approve` and `request_changes` submissions remain valid because either verdict itself communicates a conclusion. Markdown bytes which pass these presence checks are retained exactly.

## 3. Durable private-Git representation

### 3.1 Owned ref namespace

Reviews owns exactly:

```text
refs/workbraid/reviews/<change-set-uuid>/<review-uuid>
```

Both path identities are canonical lower-case UUIDs. A review UUID is unique across this Reviews namespace in one store. The ref path, record metadata, and exact reviewed Change Set state must agree on the Change Set identity.

Malformed, duplicated, conflicting, or unsupported records inside `refs/workbraid/reviews/` appear as explicit unavailable review entries. They do not make Accepted, the owning Change Set, or other valid reviews unavailable. Refs in `refs/workbraid/change-sets/`, `refs/workbraid/reconciliation/`, or any other `refs/workbraid/*` subtree are outside Reviews interpretation. No generic namespace registry is introduced.

### 3.2 Review commit and reachability

Each review ref points to one ordinary WorkBraid-authored review commit. That commit has exactly one parent: the exact reviewed Change Set state commit named by the active Change Set ref after `Review changes` durably recorded the matching binding.

The parent link is required. Recording only the state object ID in a blob would not keep a non-chained Change Set generation reachable through ordinary Git garbage collection. The review ref reaches:

```text
review ref
  -> review commit
     -> exact reviewed Change Set state commit
        -> exact base commit
```

The reviewed state commit's closed Change Set envelope retains `proposal.md`, `changes.yaml`, its nested candidate Architecture tree, and the exact review binding. Existing Change Set parsing, `ConstructCandidate`, candidate-tree equality, and version-aware Architecture loading validate that parent. Reviews adds no second proposal or Architecture interpretation.

Review commits are independent records, not a chain. One review commit is not the parent of another and review refs are not branches in the product model. Review commit author, timestamp, and message are operational Git data and carry no review semantics.

### 3.3 Closed review tree

The review commit root is a closed operational tree:

```text
review.yaml                         100644
body.md                             100644
comments/                           040000  # omitted when there are no comments
comments/<comment-uuid>.yaml        100644
comments/<comment-uuid>.md          100644
```

Every ID listed by `review.yaml` has exactly one matching YAML/Markdown pair. The comments tree contains no unlisted or unmatched path. Every WorkBraid-created blob uses mode `100644`. `body.md` is always present and contains the exact overall UTF-8 Markdown bytes, including an empty document. Each comment Markdown blob contains that comment's exact body.

Review metadata never enters the Change Set envelope's nested `architecture/` tree or Accepted Architecture. Review bodies and comments are not Architecture semantics.

### 3.4 `review.yaml`

`review.yaml` is UTF-8 YAML with this closed schema and exact key spelling:

```yaml
format: workbraid-review
version: 1
id: 11111111-1111-4111-8111-111111111111
change_set_id: 22222222-2222-4222-8222-222222222222
reviewed_state: 0123456789abcdef0123456789abcdef01234567
binding:
  base_revision: 123456789abcdef0123456789abcdef012345678
  candidate_tree: 23456789abcdef0123456789abcdef0123456789
  generation: 4
verdict: request_changes
author: Architecture reviewer
submitted_at: "2026-09-04T14:30:00Z"
comments:
  - 33333333-3333-4333-8333-333333333333
```

`comments` is required and may be `[]`. Its order preserves the reviewer's submitted presentation order only; it creates no priority, thread, or lifecycle semantics.

`reviewed_state` must equal the review commit's sole parent. That parent must parse under the non-applied/active Change Set state-commit schema. Its stored Change Set identity must equal `change_set_id`, its stored review block must exactly equal `binding`, and its nested candidate tree must exist and validate. Parent validation does not require a live active ref, a current generation, or a still-active Change Set. Unknown or missing keys, invalid identifiers, an unsupported format/version/verdict, duplicate comment IDs, parent mismatch, or binding mismatch makes the review unavailable.

### 3.5 Comment metadata and anchors

Each `comments/<comment-uuid>.yaml` is a closed record:

```yaml
format: workbraid-review-comment
version: 1
id: 33333333-3333-4333-8333-333333333333
anchor:
  kind: component_markdown
  side: with_changes
  component_id: 44444444-4444-4444-8444-444444444444
  start_line: 8
  end_line: 10
```

Exactly one of these anchor shapes is allowed:

```yaml
# Whole proposal and its exact reviewed context. No side.
anchor:
  kind: proposal

# proposal.md source. No Before side exists.
anchor:
  kind: proposal_markdown
  start_line: 2
  end_line: 4

# One Component in an exact Architecture side.
anchor:
  kind: component
  side: with_changes
  component_id: 44444444-4444-4444-8444-444444444444

# The canonical Markdown section of one Component, excluding frontmatter.
anchor:
  kind: component_markdown
  side: before
  component_id: 44444444-4444-4444-8444-444444444444
  start_line: 5
  end_line: 5

# One Diagram.
anchor:
  kind: diagram
  side: with_changes
  diagram_id: 55555555-5555-4555-8555-555555555555

# One canonical composition fact.
anchor:
  kind: composition
  side: with_changes
  diagram_id: 55555555-5555-4555-8555-555555555555
  component_id: 44444444-4444-4444-8444-444444444444
  aspect: reference

# One parent-owned detail link. detail_diagram_id is required only here.
anchor:
  kind: composition
  side: with_changes
  diagram_id: 55555555-5555-4555-8555-555555555555
  component_id: 44444444-4444-4444-8444-444444444444
  aspect: detail
  detail_diagram_id: 66666666-6666-4666-8666-666666666666

# One exact global Relationship fact occurrence.
anchor:
  kind: relationship
  side: before
  source_component_id: 44444444-4444-4444-8444-444444444444
  target_component_id: 77777777-7777-4777-8777-777777777777
  label: calls
  occurrence: 2
```

`side` accepts only `before` or `with_changes`. It is required for every Architecture anchor and forbidden for proposal/proposal-Markdown anchors. A composition `aspect` is exactly `home`, `reference`, or `detail`. `detail_diagram_id` is required for `detail` and forbidden otherwise. A home anchor identifies the home appearance fact even when it also owns a separately anchorable detail link.

A Relationship occurrence is one-based among identical `(source Component ID, target Component ID, exact label)` facts in source order on the selected snapshot side. It introduces no Relationship identity or ordering semantics. A removed fact can be anchored on `before`; an added fact can be anchored on `with_changes`.

Derived Lives in boundary nodes are not canonical composition facts and cannot be composition anchors. A reviewer may anchor the real crossing Relationship or the external Component instead. No boundary, appearance, or Relationship ID is introduced.

### 3.6 Exact line semantics

Line anchors use one-based inclusive `start_line` and `end_line`, with `start_line <= end_line`. Lines are delimited by the LF byte in the exact reviewed UTF-8 source. A final LF terminates the preceding line and does not create an additional addressable empty line; empty source has zero addressable lines. CR in CRLF remains part of the line bytes.

For `proposal_markdown`, the source is the exact reviewed Change Set state's `proposal.md` blob. For `component_markdown`, it is every exact byte after that side's Component frontmatter closing-delimiter line: any optional whitespace before the H1, the H1, and the body through end of file. The delimiter's line ending belongs to frontmatter and is not included. WorkBraid may render the Markdown safely, but line validation and display must retain those exact source bytes and line boundaries.

Reviews 1 does not anchor rendered DOM nodes, CommonMark AST nodes, unified-diff line numbers, Diagram YAML, Component frontmatter, or inferred semantic text.

### 3.7 Submission-time validation

Every anchor is resolved once against the exact immutable reviewed parent state:

- proposal lines must exist in its exact `proposal.md`;
- a Component or Diagram UUID must resolve on the selected Before/With side;
- Component Markdown lines must exist in that side's exact canonical Markdown section;
- a composition home/reference/detail fact must exist exactly in the selected Diagram and side;
- a detail anchor's child ID must equal the exact parent-owned link;
- a Relationship tuple and occurrence must resolve exactly on that side; and
- fields forbidden for the selected anchor kind must be absent.

Failure rejects the whole review submission and creates no review ref. Submitted anchors never float, follow renames, migrate to later lines, use fuzzy matching, or claim continued applicability after proposal iteration.

## 4. Submission authority and atomicity

Review submission is allowed only for an active Change Set with a valid current `Review changes` binding. Applied records are readable but accept no new submissions in Reviews 1; feedback after application has no demonstrated iteration workflow.

Preparing that binding is idempotent at its durable boundary. If an active Change Set already stores the exact review binding for its current base revision, candidate tree, and generation, repeated browser **Review changes**, CLI `change-set review`, or MCP `change_set_review` returns the same active Change Set state object as `reviewed_state`. It performs no semantically identical state-commit write and no active-ref CAS. If no exact binding exists, `Review changes` may establish it normally. A real Architecture mutation, Change Set rename, or proposal-Markdown edit still increments that Change Set's generation, removes its review binding, and writes a new state normally.

This durable no-op lets multiple reviewers share one exact state safely: preparing the same unchanged review cannot invalidate an earlier reviewer's `reviewed_state`. It introduces no reviewer session, lock, lease, collaboration state, or persistent draft.

The browser, CLI, or MCP submission carries exact:

- `store_id`;
- `change_set_id`;
- `reviewed_state` returned by the successful `Review changes` response;
- `base_revision`;
- `candidate_tree`; and
- `generation`.

Under the existing Manager synchronization boundary the server checks the loaded store, current active record/ref object, valid current review, exact binding, and every anchor. It captures the immutable material needed for the review, writes the review blobs/tree/commit, then uses one fixed Git reference transaction equivalent to:

```text
verify refs/workbraid/change-sets/active/<change-set-uuid> <reviewed-state>
create refs/workbraid/reviews/<change-set-uuid>/<review-uuid> <review-commit>
```

The create is from the zero object. Both commands succeed or neither ref change occurs. This final Git verification prevents another process or late Change Set mutation/acceptance/discard from attaching feedback to a state other than the one inspected. Same-process review submission and Change Set operations serialize through the existing lock; no review transaction framework is added.

The `change_set_review` result gains the exact `reviewed_state` commit ID after its review binding has been durably written. Adding that machine field does not change the meaning of review or its base/tree/generation acceptance binding.

Submitting feedback does not mutate the Change Set ref, generation, current review binding, candidate, or Accepted. A failed/raced submission leaves the proposal and every existing submission unchanged. Objects written before a failed ref transaction are non-canonical and may be collected.

If the Change Set changes between review inspection and submission, the request returns `review_invalidated`; the caller inspects and reviews the new generation rather than silently retargeting feedback. A valid out-of-date active proposal may still receive a submission against its exact original-base review, while the result truthfully reports that the proposal is out of date with current Accepted.

Submitting feedback against an already durable exact review binding does not require Accepted to be known-current. Known-non-current or indeterminate Accepted authority is returned truthfully as context, but does not prevent an otherwise exact immutable submission. When current Accepted cannot be determined, WorkBraid does not claim that the proposal is current or out of date. This relaxation applies only to review submission: Change Set mutation, preparation of a new `Review changes` binding, reconciliation, and acceptance retain their existing Accepted-authority rules.

## 5. Later proposal and lifecycle changes

When a reviewed Change Set advances, every prior review ref and parent remains unchanged. List/inspect derives and reports:

- exact reviewed base, candidate tree, generation, and state commit;
- whether that generation still equals the current Change Set generation and binding;
- whether the Change Set is active or applied;
- whether an active Change Set is currently out of date with Accepted; and
- current Accepted revision and authority knowledge as context, never as a replacement for the review's Before side.

List/inspect always reconstructs an immutable review from its review ref and exact parent first. It then separately derives current lifecycle and Accepted context if an active/applied Change Set and authoritative Accepted observation exist. An earlier review's proposal Markdown, Before/With Architecture, comments, and anchors are never overlaid onto a newer proposal generation. Reviews 1 does not decide whether old feedback still applies.

Acceptance retains its existing three-ref Change Set transaction and does not modify review refs. Reviews remain readable when the Change Set becomes Applied. The applied receipt and review records are separate durable facts; neither becomes an approval gate or aggregate state.

### Discarded Change Sets

Change Sets 1 permits deliberate deletion of an active proposal, while Reviews 1 requires submitted records to be immutable and durable. The approved rule is:

- Change Set discard keeps its existing meaning and deletes only the exact active Change Set ref;
- it does not delete refs owned by Reviews;
- existing review refs continue to retain their exact reviewed state commits;
- no new review can be submitted because no active current review exists;
- exact review list/inspect and a direct submitted-review browser URL remain usable when supplied the Change Set/review UUIDs;
- the normal proposal selector does not invent a discarded proposal record or lifecycle; and
- the review view says only that the proposal is no longer active.

This preserves both namespace ownership and permanent submitted feedback without reopening Change Sets into an archive/workflow product. WorkBraid does not cascade-delete submitted reviews, block Change Set discard because reviews exist, create an archived/discarded Change Set lifecycle, or reopen discarded proposals.

## 6. Browser review workspace

The stable current-review route remains:

```text
/projects/<slug>/proposals/<change-set-uuid>/review
```

It continues to show the exact current bound Before/With review and canonical diff. Reviews 1 adds contextual annotation affordances and one compact review-composition area in that task, not a separate dashboard or permanent management panel.

The composer supports:

- a self-described author label;
- optional overall Markdown;
- accumulated local anchored comments;
- one verdict; and
- deliberate **Submit review**.

Before submission, comments may be added, adjusted, or removed as transient browser-local composition. Existing dirty-navigation protection covers non-empty unsent review composition when leaving the review, switching proposal/project context, or opening an earlier submitted review. Leaving without submission drops only those local values. There is no server-side review draft or autosave.

Contextual **Add comment** actions operate on product concepts:

- the proposal as a whole;
- selected proposal-Markdown source lines;
- the selected Component or exact Component-Markdown source lines;
- the selected Diagram;
- a selected home/reference/detail composition fact; and
- a selected exact Relationship occurrence.

Line-addressable views expose only exact proposal Markdown or the Component's canonical Markdown section. Raw Diagram YAML and Component frontmatter never become ordinary review UI. All Markdown uses the approved inert safe renderer and resource-request protections.

Submitted reviews appear in the proposal/review context with verdict, author label, submitted time, binding/generation, and current/earlier/applied/out-of-date context. Inspecting one uses a stable exact route:

```text
/projects/<slug>/proposals/<change-set-uuid>/reviews/<review-uuid>
```

That read-only route reconstructs the exact parent snapshot and shows its own proposal Markdown, Before/With toggle, structured review map/detail, exact diff, overall body, and anchored comments. It remains directly readable after application and under the approved discard rule. Back/Forward and reload preserve the exact review identity.

Current-generation anchors may be highlighted contextually. Earlier-generation comments are shown only on their own exact review route; they are never rendered inline on the current proposal as if current. Review render failure must leave the exact bodies, anchor data, source ranges, and canonical diff inspectable rather than making durable feedback unreachable.

## 7. Agent Access parity

Reviews 1 adds domain-oriented operations without changing the meaning of `change-set review`:

| Product action | CLI | MCP |
|---|---|---|
| List submitted reviews for one Change Set | `review-submission list` | `review_submissions_list` |
| Inspect one submission and exact reviewed snapshot | `review-submission inspect` | `review_submission_inspect` |
| Submit feedback on an exact current review | `review-submission submit` | `review_submission_submit` |

CLI syntax follows the existing global-flag contract, for example:

```text
workbraid --json review-submission list --store-id <uuid> --change-set-id <uuid>
workbraid --json review-submission inspect --store-id <uuid> --change-set-id <uuid> --review-id <uuid>
```

`review-submission submit` takes the exact identity/binding flags, verdict, author, optional overall Markdown literal/file, and an optional file containing the closed JSON array of `{body, anchor}` comments. This is structured review input, not a generic API call or raw Architecture/YAML patch. WorkBraid generates and returns the review/comment UUIDs.

MCP uses the same closed request fields and a typed comments array. Both transports ultimately call the same server operation and return the same semantic result. There is no process-wide selected review, local private-store fallback, or client-owned snapshot.

List results are compact summaries. Inspect returns:

- review/change-set/comment identities;
- author, timestamp, verdict, overall Markdown, and every comment body/typed anchor;
- `reviewed_state` and exact base/tree/generation;
- exact reviewed proposal Markdown, Before/With projections, structured comparison, and canonical diff;
- whether the reviewed generation is still current;
- active/applied/no-longer-active context;
- truthful out-of-date/current Accepted context; and
- stable browser review URL.

The existing local protocol remains `workbraid-agent-v2`. These operations and the additive `reviewed_state` result field do not invalidate v2 envelopes or existing precondition semantics. The CLI help, embedded `workbraid --skill`, MCP tool schemas/descriptions, and black-box examples change together. The skill clearly separates **prepare Review changes** from **submit review feedback** and explains exact anchors, earlier-generation truthfulness, and informational verdicts.

New shared typed errors are:

| Code | Meaning / next safe action |
|---|---|
| `review_submission_not_found` | That exact Change Set/review UUID pair has no durable review ref. |
| `review_submission_unavailable` | The owned review ref exists but its commit, schema, parent state, or anchors cannot be validated; do not guess or repair it. |
| `review_anchor_invalid` | At least one submitted anchor does not resolve exactly in the bound snapshot; correct the returned comment/field location. |
| `review_submission_not_allowed` | The Change Set is not an active proposal with a current bound review; inspect its lifecycle/state. |

`review_invalidated` remains the classification for a changed state object or base/tree/generation between review inspection and submission. Existing project/store, non-current authority, malformed request, connection, protocol, and operation errors retain their meanings. Agents branch on codes, never prose.

## 8. Deliberate exclusions

Reviews 1 does not add:

- any second candidate, parser, review binding, acceptance authority, or Architecture projection;
- review editing/deletion, persistent drafts, threads, replies, resolution, reactions, notifications, or applicability inference;
- authenticated identity, permissions, trusted authorship, required reviewers, counts, aggregate status, or acceptance gates;
- fuzzy line/rename mapping, Git blame, AST anchors, Relationship IDs, appearance IDs, or raw Git/YAML review tools;
- proposal history beyond immutable states retained as review parents;
- reconciliation, merge, rebase, conflict handling, or proposal-base changes;
- SQLite, worktrees, another registry/database, review branches, generic metadata/review/workflow frameworks;
- remote exposure, multi-user collaboration, autonomous agents, Agent Control, or Planning; or
- manual/persisted Diagram layout, routes, shapes, annotations, Diagram kinds, UML, isometric rendering, or rich-diagram Phase 3 state.
