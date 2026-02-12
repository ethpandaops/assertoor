# Assertoor Current State Analysis

This document provides a comprehensive analysis of the current Assertoor architecture as a foundation for the planned refactoring effort.

## Table of Contents
1. [Overall Architecture](#1-overall-architecture)
2. [Task System](#2-task-system)
3. [Variable System](#3-variable-system)
4. [Web UI & API](#4-web-ui--api)

---

## 1. Overall Architecture

### 1.1 Project Structure

The project has recently undergone a refactoring (commit bdf21aa) that moved packages from `pkg/coordinator/` to top-level `pkg/` directories:

```
assertoor/
├── cmd/                          # CLI entry point
├── pkg/
│   ├── assertoor/               # Core coordinator logic
│   ├── test/                    # Test execution & descriptors
│   ├── scheduler/               # Task scheduling engine
│   ├── tasks/                   # 40+ built-in task implementations
│   ├── clients/                 # Ethereum client pools
│   │   ├── consensus/           # Consensus layer clients
│   │   └── execution/           # Execution layer clients
│   ├── db/                      # Database abstraction
│   ├── web/                     # Web UI & API
│   ├── vars/                    # Variable/scope system
│   ├── wallet/                  # Wallet management
│   └── [other packages]         # Logger, helper, names, etc.
```

### 1.2 Entry Point & Main Application Flow

```
main.go
  └── cmd.Execute(ctx)
      └── Coordinator.Run(ctx)
          1. Initialize database
          2. Initialize client pools (consensus + execution)
          3. Initialize web servers
          4. Load test registry
          5. Spawn goroutines:
             - RunTestScheduler() - cron/startup scheduling
             - RunTestCleanup()   - historical data cleanup
             - runEpochGC()       - garbage collection
             - RunOffQueueTestExecutionLoop() - skip-queue tests
             - RunTestExecutionLoop(concurrency) - main execution [BLOCKS]
```

### 1.3 Coordinator Architecture

The `Coordinator` is the heart of the system:

```go
// pkg/assertoor/coordinator.go
type Coordinator struct {
    Config          *Config
    log             *logger.LogScope
    database        *db.Database
    clientPool      *clients.ClientPool
    walletManager   *wallet.Manager
    webserver       *web.Server
    validatorNames  *names.ValidatorNames
    globalVars      types.Variables
    registry        *TestRegistry
    runner          *TestRunner
}
```

It provides:
- Access to database, clients, wallet, validator names
- Test scheduling & execution control
- Global variable management
- History queries

### 1.4 Test Lifecycle & Execution

**TestRegistry** (`pkg/assertoor/testregistry.go`):
- Loads test definitions from local configs + external YAML files
- Maps test IDs to `TestDescriptor` objects
- Supports both local and external (HTTP/file) test sources

**TestRunner** (`pkg/assertoor/testrunner.go`):
- Maintains queue of pending tests
- Manages test run lifecycle
- Implements scheduling:
  - **Startup tests**: Run immediately on startup
  - **Cron tests**: Triggered by cron expressions (minute-based)
  - **Queued tests**: Run sequentially with concurrency limit

```go
type TestRunner struct {
    testRunMap               map[uint64]types.Test    // All test runs
    testQueue                []types.TestRunner       // Pending tests
    queueNotificationChan    chan bool                // Queue updates
    offQueueNotificationChan chan types.TestRunner    // Skip-queue tests
}
```

**Test** (`pkg/test/test.go`):
- Represents a single test execution with:
  - Run ID, test ID, timeout
  - Task scheduler instance
  - Variables scope
  - Status tracking (pending → running → success/failure)
- Flow: `Validate()` → `Run(ctx)` → task scheduler execution

### 1.5 Execution Model Summary

```
Coordinator.Run()
├── RunTestScheduler (goroutine)
│   ├── Schedule startup tests immediately
│   └── Schedule cron tests every minute
├── RunTestExecutionLoop (main, with semaphore)
│   └── Dequeue test
│       └── Test.Run()
│           └── TaskScheduler.RunTasks()
│               ├── ExecuteTask (recursive)
│               │   ├── Check condition
│               │   ├── Load config
│               │   └── task.Execute()
│               ├── Run cleanup tasks
│               └── Return result
├── RunOffQueueTestExecutionLoop (goroutine)
│   └── Handle skip-queue tests
└── RunTestCleanup (goroutine)
    └── Periodically clean old test history
```

---

## 2. Task System

### 2.1 Task Interface & Descriptor

**File:** `pkg/types/task.go`

```go
type Task interface {
    Config() interface{}
    Timeout() time.Duration
    LoadConfig() error          // Parse & validate config
    Execute(ctx context.Context) error
}

type TaskDescriptor struct {
    Name        string
    Aliases     []string
    Description string
    Config      interface{}           // Config struct template
    NewTask     func(...) (Task, error) // Factory
}
```

### 2.2 Task Lifecycle

#### Creation Flow:
1. **Validation**: Task descriptor is looked up by name (`tasks.GetTaskDescriptor()`)
2. **State Initialization**: A `taskState` struct is created with:
   - Unique index (TaskIndex - incrementing uint64)
   - Logger scope for structured logging
   - Variables scope (inherited from parent or root)
   - Outputs scope for child task results
   - Status variables for task result tracking
3. **Database Registration**: Task is inserted into database with metadata
4. **Parent-Child Link**: If created from a parent task, parent index is stored

#### Execution Flow (`pkg/scheduler/task_execution.go`):

1. **Starting Phase**:
   - Mark task as started, set start time
   - Update database with started flag
   - Set task status variables: `started=true, running=true`

2. **Condition Evaluation** (if task has `If` field):
   - Evaluate condition using variable query resolution (gojq)
   - If condition is false, skip task (result = TaskResultNone)
   - If condition errors, fail task (result = TaskResultFailure)

3. **Task Instantiation**:
   - Call descriptor's NewTask function to create task instance
   - Pass TaskContext containing:
     - Scheduler reference (for creating child tasks)
     - Task index and variables scope
     - Output variables object
     - Logger scope
     - NewTask callback for creating child tasks
     - SetResult callback to explicitly set task result

4. **Config Loading**:
   - Call `task.LoadConfig()` to deserialize and validate configuration
   - Variables from ConfigVars are injected using ConsumeVars
   - If validation fails, task fails

5. **Execution**:
   - Call `task.Execute(taskContext)` with cancellable context
   - Timeout implemented via goroutine that cancels context
   - Panic recovery implemented to catch panics

6. **Result Determination** (if task didn't explicitly set result):
   - If timeout or error: TaskResultFailure
   - If no error: TaskResultSuccess

7. **Stopping Phase**:
   - Mark task as not running
   - Set stop time
   - Flush logs
   - Update database with final state

### 2.3 Run States and Result States

#### Run States (tracked in `taskState`):
- `isStarted bool` - Task has begun execution
- `isRunning bool` - Task is currently executing
- `isSkipped bool` - Task condition was false
- `isTimeout bool` - Task exceeded timeout duration

State transitions:
```
(Initial) → isStarted=true, isRunning=true
    → [Task Execution]
    → isRunning=false
    → [Final State: started, skipped, timeout]
```

#### Result States (`pkg/types/task.go`):

```go
type TaskResult uint8

const (
    TaskResultNone    TaskResult = 0  // Task hasn't completed or was skipped
    TaskResultSuccess TaskResult = 1  // Task succeeded
    TaskResultFailure TaskResult = 2  // Task failed
)
```

Result Determination:
- `TaskResultNone`: Task skipped (condition false) OR still running
- `TaskResultSuccess`: Task returned no error AND no timeout AND result not explicitly set to failure
- `TaskResultFailure`: Task returned error OR timeout occurred OR explicitly set to failure

### 2.4 Parent Task Control Over Child Tasks

Parent tasks create and execute child tasks during their `Execute` phase via the `TaskContext.NewTask` callback.

**Creating Child Tasks:**
```go
// From run_tasks/task.go LoadConfig
for i := range config.Tasks {
    taskOpts, err := t.ctx.Scheduler.ParseTaskOptions(&config.Tasks[i])
    task, err := t.ctx.NewTask(taskOpts, taskVars)  // Creates child
    childTasks = append(childTasks, task)
}
```

**Executing Child Tasks:**
```go
func (t *Task) Execute(ctx context.Context) error {
    for i, task := range t.tasks {
        err := t.ctx.Scheduler.ExecuteTask(ctx, task, func(ctx context.Context,
            cancelFn context.CancelFunc, task types.TaskIndex) {
            if t.config.StopChildOnResult {
                t.ctx.Scheduler.WatchTaskPass(ctx, cancelFn, task)
            }
        })
        // Handle result based on config...
    }
    return nil
}
```

**Result Notification System:**
Tasks can watch for result changes using `GetTaskResultUpdateChan()`:
```go
func (ts *taskState) GetTaskResultUpdateChan(oldResult types.TaskResult) <-chan bool {
    // Returns channel that closes when result changes
}
```

### 2.5 Available Task Categories (40+ implementations)

| Category | Examples |
|----------|----------|
| **Check tasks** | check_consensus_finality, check_clients_are_healthy, check_consensus_validator_status |
| **Generate tasks** | generate_deposits, generate_exits, generate_blob_transactions |
| **Get tasks** | get_consensus_specs, get_wallet_details, get_consensus_validator |
| **Run tasks** | run_tasks (sequential), run_tasks_concurrent, run_task_matrix, run_task_background |
| **Utility tasks** | sleep, run_shell |

### 2.6 Control Flow Tasks (Glue Tasks)

**run_tasks** - Sequential execution:
```go
type Config struct {
    Tasks             []helper.RawMessageMasked
    StopChildOnResult bool  // Cancel siblings when one finishes
    ExpectFailure     bool  // Task should fail
    ContinueOnFailure bool  // Continue even if child fails
    NewVariableScope  bool  // Create new scope for children
}
```

**run_tasks_concurrent** - Parallel execution with limits:
```go
type Config struct {
    SucceedTaskCount uint64  // Number of successful children needed
    FailTaskCount    uint64  // Number of failed children that cause failure
    FailOnUndecided  bool    // Fail if limits not met
    NewVariableScope bool
    Tasks            []helper.RawMessageMasked
}
```

**run_task_matrix** - Iteration over values:
```go
type Config struct {
    Task            *helper.RawMessage
    MatrixVar       string        // Variable name to set in each iteration
    MatrixValues    []interface{} // Values to iterate over
    RunConcurrent   bool          // Run all iterations concurrently
    SucceedTaskCount uint64
    FailTaskCount   uint64
}
```

**run_task_background** - Foreground/background coordination:
```go
type Config struct {
    BackgroundTask         *helper.RawMessage
    ForegroundTask         *helper.RawMessage
    ExitOnForegroundSuccess bool
    ExitOnForegroundFailure bool
    OnBackgroundComplete    string  // "fail" / "success" / "failOrIgnore"
}
```

**run_task_options** - Retry and result manipulation:
```go
type Config struct {
    Task            *helper.RawMessage
    RetryOnFailure  bool   // Retry on failure
    MaxRetryCount   uint   // Max retries
    ExpectFailure   bool   // Task should fail
    IgnoreFailure   bool   // Don't fail if child fails
    InvertResult    bool   // Invert success/failure
    PropagateResult bool   // Propagate child result
    ExitOnResult    bool   // Exit when result received
}
```

---

## 3. Variable System

### 3.1 Variable Definition Sources

- **Global Variables**: Defined in coordinator configuration (`globalVars` section)
- **Test Variables**: Defined in test definition `config` section
- **Task Configuration**: Static (`config`) and dynamic (`configVars`)
- **Task Outputs**: Set by tasks during execution

### 3.2 Scope Hierarchy

The variable system uses a hierarchical parent-child scope model:

```
Global Scope (coordinator)
  └── Test Scope (per test run)
        └── Task Scope (per task)
              └── Child Task Scopes (nested tasks)
```

**Scope Creation:**
```go
// Test creates child scope from global
testVars := globalVars.NewScope()

// Task inherits from parent or root
if parentState != nil {
    variables = parentState.taskVars
} else {
    variables = ts.rootVars
}
```

**Scope Lookup Order:**
1. Current scope variables
2. Parent scope (recursive)
3. Default values (fallback)

### 3.3 Variable Resolution and Interpolation

**Two Resolution Mechanisms:**

**A. Simple Placeholder - `${varName}`**
```go
// Pattern: \${([^}]+)}
str = "${walletPrivkey}"  // → resolves to value of walletPrivkey
```

**B. Query-Based Resolution - `${{jq.expression}}`**
```go
// Pattern: \${{(.*?)}}
str = "${{tasks.get_specs.outputs.specs.ELECTRA_FORK_EPOCH}}"
// Uses gojq library for JQ-style queries
```

### 3.4 ConfigVars Mechanism

The `configVars` field maps YAML config fields to variable expressions:

```go
type TaskOptions struct {
    Config     *helper.RawMessage       // Static YAML
    ConfigVars map[string]string        // "configField" → "query.path"
}
```

**Example:**
```yaml
- name: check_consensus_slot_range
  configVars:
    minEpochNumber: "tasks.get_specs.outputs.specs.ELECTRA_FORK_EPOCH"
    publicKey: "tasks.get_validator.outputs.pubkey"
```

**Processing via `ConsumeVars()`:**
```go
func (v *Variables) ConsumeVars(config interface{}, consumeMap map[string]string) error {
    // For each mapping, execute JQ query and apply result to config struct
    for cfgName, varQuery := range consumeMap {
        query, _ := gojq.Parse(fmt.Sprintf(".%v", varQuery))
        iter := query.RunWithContext(ctx, varsMap)
        val, ok := iter.Next()
        applyMap[cfgName] = val
    }
    // Marshal/unmarshal to apply to config
}
```

### 3.5 JQ Expression Examples

```yaml
# Simple field access
configVars:
  walletPrivkey: "walletPrivkey"

# Nested object access
configVars:
  minEpochNumber: "tasks.get_specs.outputs.specs.ELECTRA_FORK_EPOCH"

# Complex transformations
configVars:
  # String concatenation
  withdrawalCredentials: '| "0x020000000000000000000000" + (.tasks.depositor_wallet.outputs.childWallet.address | capture("(0x)?(?<addr>.+)").addr)'

  # Arithmetic
  minEpochNumber: "|(.lastValidatorInfo.validator.activation_epoch | tonumber) + 256"

  # Multiplication
  prefundMinBalance: '| (.worker.depositCount + 1) * 1000000000000000000'
```

### 3.6 Task Outputs

Each task has a separate `taskOutputs` variable scope:

**Setting Outputs:**
```go
func (t *Task) Execute(ctx context.Context) error {
    // Task logic...
    t.ctx.Outputs.SetVar("address", wal.GetAddress())
    t.ctx.Outputs.SetVar("balance", wal.GetBalance().String())
    return nil
}
```

**Accessing Outputs:**
```yaml
configVars:
  # Pattern: tasks.<task-id>.outputs.<field>
  walletPrivkey: "tasks.depositor_wallet.outputs.childWallet.privkey"
```

**Task Status Variables:**
```go
// Auto-populated for each task
tasks.<id> = {
    outputs: { <task-outputs> },
    result: TaskResult (0=none, 1=success, 2=failure),
    running: bool,
    started: bool,
    timeout: bool
}
```

### 3.7 Sub-Scopes

Variables supports hierarchical sub-scopes (not just flat variables):

```go
func (v *Variables) GetSubScope(name string) types.Variables {
    // Creates/returns named sub-scope
    // Parent sub-scopes are inherited
}
```

The "tasks" sub-scope is automatic and contains all task results/outputs:
```go
tasksScope := variables.GetSubScope("tasks")
tasksScope.SetSubScope(options.ID, taskState.taskStatusVars)
```

---

## 4. Web UI & API

### 4.1 Server Structure

**Location:** `pkg/web/`

**Architecture:**
- Go's `net/http` with **Gorilla mux** router
- **Negroni** middleware framework
- Configuration via `types.ServerConfig`

**Key Files:**
- `server.go` - Core server setup
- `api/*.go` - REST API handlers (16 files)
- `handlers/*.go` - Frontend handlers (8 files)
- `templates/**/*.html` - Go templates (13 files)
- `static/` - CSS, JS, fonts (embedded)

### 4.2 API Endpoints (16 total)

#### Public Read-Only:
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/tests` | GET | List all test definitions |
| `/api/v1/test/{testId}` | GET | Get single test definition |
| `/api/v1/test_runs` | GET | List all test runs |
| `/api/v1/test_run/{runId}` | GET | Get test run summary |
| `/api/v1/test_run/{runId}/status` | GET | Get current status |

#### Admin Endpoints:
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/tests/register` | POST | Register new test |
| `/api/v1/tests/register_external` | POST | Register external test |
| `/api/v1/tests/delete` | POST | Delete test definition |
| `/api/v1/test_runs/schedule` | POST | Schedule new test run |
| `/api/v1/test_runs/delete` | POST | Delete test runs |
| `/api/v1/test_run/{runId}/cancel` | POST | Cancel running test |
| `/api/v1/test_run/{runId}/details` | GET | Full details with logs |
| `/api/v1/test_run/{runId}/task/{taskIndex}/details` | GET | Task details |
| `/api/v1/test_run/{runId}/task/{taskId}/result/{resultType}/{fileId}` | GET | Task result files |

**Response Format:**
```json
{
  "status": "OK" or "ERROR: <message>",
  "data": <response_data>
}
```

### 4.3 Frontend Structure

#### Routes:
| Route | Purpose |
|-------|---------|
| `/` | Dashboard - all test runs (paginated) |
| `/registry` | Test registry - available tests |
| `/test/{testId}` | Test page with run history |
| `/run/{runId}` | Test run details with tasks |
| `/clients` | Client status (CL & EL) |
| `/logs/{since}` | System logs (admin only) |

#### Template System:
- Go `html/template` with embedded static files
- Template caching with sync.RWMutex
- HTML minification support
- Custom template functions (math, time formatting, utilities)

**Templates:**
```
_layout/
  ├── layout.html, header.html, footer.html, blank.html, 404.html, 500.html
index/
  └── index.html
test/
  ├── test.html, test_runs.html
test_run/
  └── test_run.html
registry/
  └── registry.html
sidebar/
  └── sidebar.html
clients/
  └── clients.html
```

### 4.4 Static Assets

**Embedded Files:**
```
css/
  ├── bootstrap.min.css, fontawesome.min.css, layout.css
js/
  ├── jquery.min.js, bootstrap.bundle.min.js
  ├── color-modes.js, assertoor.js, clipboard.min.js
  ├── yaml-0.3.0.min.js
  └── ace-1.5.0/ (code editor)
webfonts/
  └── fa-*.* (Font Awesome)
```

### 4.5 Real-Time Capabilities

**Current Status: NO WebSocket or real-time push support**

**How Updates Work:**
1. Frontend timers update every 1 second via `setInterval`
2. Relative times calculated client-side
3. JSON API endpoints for live data polling:
   - `/run/{runId}?json=true`
   - `/api/v1/test_run/{runId}/status`

### 4.6 Security

**Security Trimming (`securityTrimmed=true`):**
- Hides admin APIs (register, delete, cancel)
- Disables task result downloads
- Hides sensitive config variables
- Blocks `/logs/{since}` endpoint

### 4.7 Configuration

```go
type WebConfig struct {
    Server       *ServerConfig   // HTTP server settings
    PublicServer *ServerConfig   // Optional public endpoint
    Frontend     *FrontendConfig // UI settings
    API          *APIConfig      // API settings
}

type FrontendConfig struct {
    Enabled  bool
    Debug    bool
    Pprof    bool
    Minify   bool
    SiteName string
}
```

---

## Summary of Key Architectural Patterns

1. **Interface-Based Design**: Extensive use of interfaces (Coordinator, Test, TestRegistry, Task, TaskScheduler)
2. **Hierarchical Scoping**: Variable scopes mirror test/task hierarchy
3. **Task Graph Execution**: Tasks can spawn child tasks dynamically
4. **Condition-Based Execution**: Tasks have optional `if` conditions
5. **Async Execution**: Tests run concurrently; tasks within test are serial/parallel based on glue tasks
6. **State Persistence**: All test/task state written to database
7. **Graceful Shutdown**: Context cancellation propagates through execution tree
8. **Embedded Assets**: Zero runtime dependencies for web UI

---

## Key Files Reference

| Component | Path |
|-----------|------|
| Coordinator | `pkg/assertoor/coordinator.go` |
| Test Runner | `pkg/assertoor/testrunner.go` |
| Test Registry | `pkg/assertoor/testregistry.go` |
| Test Execution | `pkg/test/test.go` |
| Task Scheduler | `pkg/scheduler/scheduler.go` |
| Task Execution | `pkg/scheduler/task_execution.go` |
| Task State | `pkg/scheduler/task_state.go` |
| Task Types | `pkg/types/task.go` |
| Task Registry | `pkg/tasks/tasks.go` |
| Variables | `pkg/vars/variables.go` |
| Web Server | `pkg/web/server.go` |
| API Handlers | `pkg/web/api/*.go` |
| Frontend Handlers | `pkg/web/handlers/*.go` |
