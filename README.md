# WorkBraid

WorkBraid is a local Architecture workbench for documenting Components, their Relationships and nested Diagrams. Named proposals keep changes separate from Accepted Architecture. Review compares exact snapshots and their complete diff; Update deliberately accepts the reviewed result. Submitted feedback stays attached to the version it reviewed, and reconciliation combines parallel proposals before a new Review/Update.

One Go process serves the UI and owns private Git stores. Browser, CLI and MCP use that same process. WorkBraid does not scan or modify your source repository.

## Run locally

Build from a checkout with Go, Node/npm and Git installed:

```sh
npm ci --prefix frontend
npm run build --prefix frontend
go build -o /tmp/workbraid ./cmd/workbraid
/tmp/workbraid --listen 127.0.0.1:8080 --ui-dir frontend/dist
```

Open `http://127.0.0.1:8080`. Use **New project** to create an empty Architecture or select an existing project. Project names generate stable route slugs; the source checkout is not the project catalog.

Application data defaults to `workbraid` under the operating system's user configuration directory. `WORKBRAID_DATA_DIR` or the server's `--data-dir` selects another location. Use a durable per-user location for real work, separate from test/gate data, and run only one authoritative process against it. A temporary binary path does not make app data temporary. Private stores live under `architecture/<store-uuid>.git`; preserve the whole app-data directory when backing up work.

## CLI and MCP

```sh
/tmp/workbraid --skill
/tmp/workbraid --server http://127.0.0.1:8080 --json status
/tmp/workbraid --server http://127.0.0.1:8080 --json project list
/tmp/workbraid --server http://127.0.0.1:8080 mcp
```

CLI global flags precede the command. `--help` and the embedded `--skill` describe the commands supported by that binary. Use returned IDs and generations, inspect the exact Review binding, and Update only when acceptance is intended. Clients do not open private Git directly. MCP is a stateless stdio bridge to the running loopback process.

## Current work and contracts

Phase 3.1 stable placement is **complete**, with explicit human visual PASS on implementation `406e18ea26a01fc0ed82e2c1c5efe66bb72de4cf`. The human accepted WorkBraid’s self-Architecture at `06110ef95cec9e385d38c17831c38c5c51c6cfc3`; its seven Components, two Diagrams, complete visible coordinates and applied receipt reconstructed exactly after a full process restart. The built Accepted route passed without console errors. [Open WorkBraid locally](http://127.0.0.1:8080/projects/workbraid). Later scope is not started.

| Document | Owns |
| --- | --- |
| [Architecture](docs/architecture-v0.md) | Domain, identity, portable v2 base, catalog, source fidelity and runtime authority |
| [Proposals and Reviews](docs/architecture-proposals-v0.md) | Durable state, operational versions, exact Review/Update, immutable feedback and anchors |
| [Reconciliation](docs/architecture-reconciliation-v0.md) | Detail reassignment, semantic choices, S/B/A/P, residual construction and exact Apply |
| [Placement](docs/architecture-placement-amendment-v0.md) | Approved complete v3 positions, stable visible nodes, v2 transition and placement reconciliation |
| [UI](docs/ui-v0.md) | Language, drafting-table direction, navigation and review interaction |
| [Agent Access](docs/architecture-agent-access-v0.md) | Local CLI/MCP protocol, preconditions, discovery and recovery |
| [Roadmap](docs/roadmap.md) | Future boundaries; no next increment is authorized |

[AGENTS.md](AGENTS.md) describes development coordination. Completed plans and superseded designs live in Git history, not a second documentation archive. Runtime evidence and historical test fixtures remain separate from this cleanup.
