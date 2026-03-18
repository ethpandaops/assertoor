# Assertoor System Overview

## What is Assertoor?

Assertoor is a comprehensive testing framework for live Ethereum testnets. It orchestrates test scenarios (called "playbooks") that verify the behavior of Ethereum consensus and execution layer clients. Tests are defined in YAML and composed of reusable tasks that check network state, generate transactions, manage validators, and control execution flow.

## Core Concepts

### Coordinator
The central orchestrator (`pkg/assertoor/`) that:
- Loads configuration and connects to Ethereum endpoints
- Manages the test registry and schedules test runs
- Maintains the client pool for consensus and execution layers
- Runs the web server (API + frontend)
- Manages the database for test history

### Tests (Playbooks)
A test is a YAML file that defines:
- **Identity**: id, name, timeout
- **Configuration**: default variables and variable bindings
- **Tasks**: ordered list of tasks to execute
- **Cleanup tasks**: tasks that always run after the test completes

Tests can be loaded from:
- Inline YAML in the assertoor config
- External YAML files (local or remote URLs)
- The REST API

### Tasks
Tasks are the atomic units of work. Each task:
- Has a registered type name (e.g., `check_consensus_finality`)
- Accepts configuration parameters
- Has access to a variable scope for reading/writing state
- Can produce output variables accessible by subsequent tasks
- Reports a result: Success, Failure, or None (skipped)

### Task Categories

| Category | Purpose | Examples |
|----------|---------|---------|
| **Check** | Verify network state against expected conditions | `check_consensus_finality`, `check_clients_are_healthy` |
| **Generate** | Perform network operations (transactions, validator ops) | `generate_transaction`, `generate_deposits` |
| **Get** | Retrieve data from the network | `get_consensus_specs`, `get_wallet_details` |
| **Flow** | Control task execution order and logic | `run_tasks`, `run_tasks_concurrent`, `run_task_matrix` |
| **Utility** | Shell commands, sleep, mnemonic generation | `run_shell`, `sleep`, `get_random_mnemonic` |

### Variable System

The variable system provides hierarchical scoping with jq expression evaluation:

```
Global Scope (coordinator config)
  -> Test Scope (test config defaults + configVars from parent)
    -> Task Scope (inherits test scope)
      -> Child Task Scope (inherits parent task scope)
```

**Key features:**
- Variables flow down through scoping hierarchy
- Child scopes can read parent variables but modifications stay local
- jq expressions can query and transform variables
- Task outputs are isolated in `tasks.<taskId>.outputs` namespace
- `configVars` map config fields to jq queries against the variable scope

### Client Pool

Manages connections to Ethereum nodes:
- **Consensus clients**: Beacon API (REST) for chain state, validators, blocks
- **Execution clients**: JSON-RPC for transactions, balances, calls
- Configurable via `endpoints` array with name, URLs, and optional headers
- Client selection via regex patterns (`clientPattern`)
- Built-in health monitoring and sync status tracking

### Scheduler

The task scheduler (`pkg/scheduler/`) manages per-test execution:
- Sequential execution of root tasks
- Task state machine: pending -> running -> success/failure/skipped
- Timeout enforcement with context cancellation
- Conditional execution via `if` expressions
- Nested task support (flow tasks create child tasks)
- Result notification channels for inter-task coordination

## Architecture Diagram

```
                    +------------------+
                    |   YAML Config    |
                    +--------+---------+
                             |
                    +--------v---------+
                    |   Coordinator    |
                    +--------+---------+
                             |
          +------------------+------------------+
          |                  |                  |
  +-------v------+  +-------v------+  +-------v------+
  |  ClientPool  |  | TestRegistry |  |  WebServer   |
  +-------+------+  +-------+------+  +-------+------+
          |                 |                 |
  +-------v------+  +-------v------+  +-------v------+
  | CL + EL RPCs |  |  TestRunner  |  |  REST API    |
  +--------------+  +-------+------+  |  + React UI  |
                            |         +--------------+
                    +-------v------+
                    |  Scheduler   |
                    +-------+------+
                            |
              +-------------+-------------+
              |             |             |
       +------v---+  +-----v----+  +-----v----+
       | Check    |  | Generate |  | Flow     |
       | Tasks    |  | Tasks    |  | Tasks    |
       +----------+  +----------+  +----------+
```

## Data Flow

1. **Configuration loaded** -> Coordinator initializes client pool and test registry
2. **Tests scheduled** -> Test runner creates test instances with variable scopes
3. **Task execution** -> Scheduler processes each task:
   a. Evaluate `if` condition (skip if false)
   b. Load config (static YAML + dynamic configVars resolution)
   c. Execute task logic
   d. Record outputs and result
4. **Inter-task communication** -> Via variable scope and task output references
5. **Results persisted** -> Database stores test runs, task states, logs
6. **Results reported** -> Web UI / API / exit code

## Database Schema

- `test_runs` - Test execution history with status, timing, config
- `task_states` - Per-task state snapshots
- `task_results` - Task execution results
- `task_logs` - Task log entries
- `test_configs` - Stored test configurations

Supported engines: SQLite (default, in-memory), PostgreSQL

## REST API

Base endpoint configurable via `web.server.host` and `web.server.port`.

Key endpoints:
- `GET /api/tests` - List all registered tests
- `GET /api/test-runs` - List test run history
- `POST /api/test-runs` - Schedule a new test run
- `GET /api/test-runs/{id}` - Get test run details
- `GET /api/task-descriptors` - List available task types
- `GET /api/logs` - Streaming log output

Swagger UI available at `/api/docs` when API is enabled.

## Configuration Reference

```yaml
coordinator:
  maxConcurrentTests: 1         # Max tests running simultaneously
  testRetentionTime: 336h       # How long to keep test history

web:
  server:
    host: "0.0.0.0"
    port: 8080
  api:
    enabled: true
  frontend:
    enabled: true

database:
  engine: sqlite                # sqlite or pgsql
  sqlite:
    file: ":memory:?cache=shared"

endpoints:
  - name: "client-1"
    executionUrl: "http://localhost:8545"
    consensusUrl: "http://localhost:5052"
    headers:                    # Optional custom headers
      Authorization: "Bearer token"

validatorNames:
  inventory:                    # Maps validator indices to names
    "0-63": "lighthouse-geth"
    "64-127": "prysm-geth"

globalVars:
  walletPrivkey: "0x..."
  depositContract: "0x..."

tests:
  - id: my-test
    file: "./playbooks/my-test.yaml"
    name: "My Test"
    timeout: 30m
    config:
      key: value
    schedule:
      startup: true             # Run on startup
      cron:
        - "0 */6 * * *"        # Run every 6 hours

externalTests:
  - file: "https://example.com/test.yaml"
    name: "Remote Test"
```
