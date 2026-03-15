# Assertoor - Ethereum Network Testing Framework

## Project Overview

Assertoor is a full-scale testing orchestrator for live Ethereum testnets, built in Go. It manages test playbooks that verify network behavior by executing task-based workflows against consensus and execution layer clients.

## Architecture

```
main.go -> cmd/root.go -> pkg/assertoor/Coordinator
  |- ClientPool (consensus + execution RPC clients)
  |- TestRegistry (loads test definitions from YAML)
  |- TestRunner (executes tests with task scheduler)
  |- Database (SQLite/PostgreSQL persistence)
  |- WebServer (REST API + React frontend)
  |- EventBus (coordinator-wide events)
  |- Spamoor (transaction generation engine)
```

## Key Directories

- `cmd/` - CLI commands (root, tasks, validate)
- `pkg/assertoor/` - Coordinator, config loading, test registry
- `pkg/clients/` - Consensus and execution client pool management
- `pkg/scheduler/` - Task execution engine, state machine, variable scoping
- `pkg/tasks/` - 41+ task implementations (check, generate, flow, utility)
- `pkg/test/` - Test runner, descriptor, lifecycle management
- `pkg/types/` - Core interfaces (Task, Test, Coordinator, Variables)
- `pkg/vars/` - Variable system with jq expression evaluation
- `pkg/txmgr/` - Transaction manager (Spamoor wrapper)
- `pkg/db/` - Database layer with migrations
- `pkg/web/` - REST API with Swagger docs
- `playbooks/` - Pre-built test playbooks organized by network type
- `web-ui/` - React/TypeScript frontend

## Playbook Authoring

See `.ai/PLAYBOOK_AUTHORING.md` for complete playbook writing guide.
See `.ai/TASK_REFERENCE.md` for detailed task documentation with all config parameters and outputs.

## Building

```bash
make build        # Compile binary to bin/assertoor
make test         # Run Go tests
make docs         # Generate Swagger API docs
make ui           # Build web UI
make devnet       # Start local devnet
make devnet-run   # Run assertoor against local devnet
```

## Configuration

Main config file (YAML):
- `coordinator` - max concurrent tests, retention
- `web` - server host/port, API/frontend toggles
- `endpoints` - array of {name, executionUrl, consensusUrl}
- `validatorNames` - inventory for mapping validator indices to names
- `globalVars` - variables available to all tests
- `tests` / `externalTests` - test definitions or file references

## Variable System

- Hierarchical scoping: global -> test -> task -> child task
- jq expression evaluation via `configVars` mappings
- Placeholder syntax: `${varname}` (simple) or `${{.query}}` (jq expression)
- Task outputs accessible via `tasks.<taskId>.outputs.<variable>`
- Task status via `tasks.<taskId>.result` (0=None, 1=Success, 2=Failure)
