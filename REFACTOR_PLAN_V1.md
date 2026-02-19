# Assertoor Refactoring Plan V1

This document outlines the implementation plan for the major Assertoor refactoring effort.

## Requirements Analysis

### Requirement 1: Task Lifecycle Simplification

**Current State:**
- Tasks are controlled by their parent task (via glue tasks like `run_tasks`, `run_tasks_concurrent`)
- Parent tasks decide when to stop child tasks (e.g., `StopChildOnResult`)
- Complex configuration needed: `ExpectFailure`, `ContinueOnFailure`, `InvertResult`, etc.
- Users must understand glue task semantics to build tests

**Desired State:**
- Each task controls its own runtime autonomously
- By default: stop checks when succeeding
- Allow continuous run via a task-level setting
- More intuitive behavior without complex parent orchestration

**Gap Analysis:**
- Need new task lifecycle model where tasks self-terminate on success
- Need `continuousRun` or similar config option per task
- Glue tasks become simpler orchestrators, not lifecycle controllers
- Result propagation needs rethinking

### Requirement 2: Modern React Web UI

**Current State:**
- Server-rendered Go templates with Bootstrap 5
- jQuery-based JavaScript
- No real-time push (polling only)
- Static task list view

**Desired State:**
- Modern responsive React web app
- API-driven data loading
- Event streams for live task status/progress
- Graph visualization of task execution flow
- Visual test builder with drag-and-drop

**Gap Analysis:**
- Complete frontend rewrite needed
- Backend needs WebSocket/SSE support for real-time updates
- Need new API endpoints for test builder operations
- Task execution model needs event emission

---

## Implementation Plan

### Phase 1: Backend API & Event System

#### 1.1 Event Streaming Infrastructure

**Goal:** Enable real-time updates from backend to frontend

**Tasks:**
1. Create event bus system for task/test state changes
2. Implement Server-Sent Events (SSE) endpoint for live updates
3. Add event types:
   - `test.started`, `test.completed`, `test.failed`
   - `task.started`, `task.progress`, `task.completed`, `task.failed`
   - `task.log` (live log streaming)

**New Files:**
```
pkg/events/
├── bus.go           # Event bus with pub/sub
├── types.go         # Event type definitions
└── sse.go           # SSE connection handling
```

**API Endpoints:**
```
GET /api/v1/events/stream              # SSE endpoint for all events
GET /api/v1/test_run/{runId}/events    # SSE for specific test run
```

#### 1.2 JWT Authentication

**Goal:** Protect admin API endpoints and sensitive data (logs) with JWT authentication

**Authentication Flow:**
1. Frontend requests token from `/auth/token`
2. If authenticated (via upstream proxy header), returns JWT token
3. Token stored in sessionStorage with auto-refresh before expiration
4. Protected API calls include `Authorization: Bearer <token>` header
5. SSE streams filter out `task.log` events for unauthenticated clients

**Auth Endpoints:**
```
GET /auth/token   # Request JWT token (returns token if authenticated via header)
GET /auth/login   # Redirect to login page (for SSO/proxy auth)
```

**Configuration:**
```yaml
web:
  server:
    authHeader: "X-Forwarded-User"  # Header set by auth proxy
    tokenKey: "your-secret-key"     # JWT signing key
  api:
    enabled: true
    disableAuth: false  # Set to true to disable authentication (default: auth required)
```

**Environment Variables:**
```bash
WEB_SERVER_AUTH_HEADER=X-Forwarded-User
WEB_SERVER_TOKEN_KEY=your-secret-key
WEB_API_DISABLE_AUTH=false
```

**Protected Endpoints (require valid JWT):**
- `POST /api/v1/tests/register` - Register new test
- `POST /api/v1/tests/register_external` - Register external test
- `POST /api/v1/tests/delete` - Delete tests
- `POST /api/v1/test_runs/schedule` - Schedule test run
- `POST /api/v1/test_runs/delete` - Delete test runs
- `POST /api/v1/test_run/{runId}/cancel` - Cancel test run

**SSE Log Filtering:**
- `task.log` events are only sent to authenticated clients
- Other events (task.started, task.completed, etc.) are public

**New Files:**
```
pkg/web/auth/
├── handler.go   # Auth handler (GetToken, GetLogin endpoints)
└── check.go     # CheckAuthToken function for JWT validation
```

#### 1.3 Enhanced API for React Frontend

**New/Modified Endpoints:**

```
# Test Builder APIs
GET  /api/v1/task_descriptors          # List all available tasks with schemas
GET  /api/v1/task_descriptor/{name}    # Get task config schema (JSON Schema)
POST /api/v1/tests/validate            # Validate test YAML without running
POST /api/v1/tests/draft               # Save draft test (not registered)
GET  /api/v1/tests/drafts              # List user drafts

# Enhanced Run APIs
GET  /api/v1/test_run/{runId}/tasks    # Paginated task list with status
```

**Note:** Graph visualization is built client-side from existing task data. The `/api/v1/test_run/{runId}/details` endpoint already provides task hierarchy via `ParentIndex`, which the React frontend uses to construct the DAG.

**Task Descriptor Schema Export:**
```go
// Generate JSON Schema from task Config struct
type TaskDescriptorAPI struct {
    Name         string            `json:"name"`
    Description  string            `json:"description"`
    Category     string            `json:"category"`     // "check", "generate", "get", "run", "utility"
    ConfigSchema json.RawMessage   `json:"configSchema"` // JSON Schema for inputs
    Outputs      []TaskOutputField `json:"outputs"`      // Available outputs for UI dropdowns
    Examples     []string          `json:"examples"`
}

// TaskOutputField describes an output that a task produces
type TaskOutputField struct {
    Name        string `json:"name"`        // e.g., "address", "balance"
    Type        string `json:"type"`        // "string", "number", "object", "array"
    Description string `json:"description"` // Human-readable description
}
```

**TaskDescriptor Extension (Go Backend):**
```go
// pkg/types/task.go - Extended TaskDescriptor
type TaskDescriptor struct {
    Name        string
    Aliases     []string
    Description string
    Category    string                    // NEW: Task category for UI grouping
    Config      interface{}               // Config struct template
    Outputs     []TaskOutputDefinition    // NEW: Output definitions for UI
    NewTask     func(...) (Task, error)
}

type TaskOutputDefinition struct {
    Name        string
    Type        string // "string", "number", "bool", "object", "array"
    Description string
}
```

#### 1.4 Task Progress Reporting

**Goal:** Tasks can report intermediate progress

**Changes to Task Interface:**
```go
type TaskContext struct {
    // Existing fields...

    // New: Progress reporting
    ReportProgress func(percent float64, message string)

    // New: Emit custom events
    EmitEvent func(eventType string, data any)
}
```

**Database Schema Addition:**
```sql
ALTER TABLE task_states ADD COLUMN progress REAL DEFAULT 0;
ALTER TABLE task_states ADD COLUMN progress_message TEXT;
```

---

### Phase 2: Task Lifecycle Refactoring

#### 2.1 Design Principles

**Goals:**
1. Tasks self-complete when their work is done (no external cancellation by default)
2. Existing `Timeout` setting is sufficient (no separate `MaxRuntime` needed)
3. Glue tasks orchestrate execution order, not child lifecycle
4. Simple defaults with opt-in complexity for edge cases
5. Developer and power-user friendly for external contributors writing tests

**Key Insight:** The existing `Timeout` field already serves both purposes:
- For check tasks: "max time to wait for condition to be met"
- For action tasks: "max execution time"

No separate `MaxRuntime` is needed.

#### 2.2 Task Self-Completion Model

**Current Problem:**
Check tasks like `check_consensus_finality` run indefinitely in a loop:
```go
// Current behavior - DOES NOT exit on success
for {
    if checkPassed {
        t.ctx.SetResult(types.TaskResultSuccess)  // Sets result but keeps running!
    }
    select {
    case <-event:
        // Continue polling
    case <-ctx.Done():
        return ctx.Err()  // Only exits when externally canceled
    }
}
```

**New Behavior: Tasks Exit on Completion**

All 40+ tasks will be updated to **return immediately** when they reach a terminal state:

```go
// New behavior - exits on success
for {
    if checkPassed {
        t.ctx.SetResult(types.TaskResultSuccess)
        t.ctx.ReportProgress(100, "Check passed")
        return nil  // EXIT immediately
    }
    if t.config.FailOnCheckMiss && checkFailed {
        t.ctx.SetResult(types.TaskResultFailure)
        return fmt.Errorf("check failed")  // EXIT immediately
    }
    // Still waiting - continue polling
    select {
    case <-event:
        // Continue
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

**Edge Case: Long-Running Concurrent Checks**

Some check tasks may be used in concurrent execution where you want to keep monitoring even after success (the condition might become false again). This is enabled via an opt-in setting:

```go
type TaskOptions struct {
    // Existing fields...

    // New: Keep running after success (for monitoring tasks in concurrent execution)
    // Default: false - task exits on success
    ContinueOnPass bool `yaml:"continueOnPass" json:"continueOnPass"`
}
```

```yaml
# Example: Monitor finality continuously while other tasks run
- name: run_tasks_concurrent
  config:
    tasks:
      - name: check_consensus_finality
        continueOnPass: true  # Keep checking, might fail again
        timeout: 30m
      - name: generate_deposits
      - name: check_deposits_processed
```

**Behavior with `continueOnPass: true`:**
- Task reports `TaskResultSuccess` when condition passes
- Task continues running and rechecks
- If condition fails later, changes result to `TaskResultNone` (or `TaskResultFailure` if `failOnCheckMiss: true`)
- Only exits on timeout or context cancellation

#### 2.3 Glue Task Simplification

**Design Principle:** Glue tasks orchestrate execution order and aggregate results. They do NOT control when child tasks stop (children self-complete).

##### 2.3.1 `run_tasks` (Sequential Execution)

**New Defaults:**
- Wait for each child to complete naturally (no `StopChildOnResult` - removed)
- Fail immediately when a child fails (default behavior)
- Option to continue on failure

**New Config:**
```go
type Config struct {
    Tasks            []helper.RawMessageMasked `yaml:"tasks" json:"tasks"`
    NewVariableScope bool `yaml:"newVariableScope" json:"newVariableScope"`

    // Failure handling (default: stop on first failure)
    ContinueOnFailure bool `yaml:"continueOnFailure" json:"continueOnFailure"`

    // Result transformation
    InvertResult bool `yaml:"invertResult" json:"invertResult"`  // Swap success/failure
    IgnoreResult bool `yaml:"ignoreResult" json:"ignoreResult"`  // Always succeed
}
```

**Execution Flow:**
```
Child 1 → Runs until self-complete → Success? → Child 2 → ... → Child N
                                   → Failure? → Stop (or continue if continueOnFailure)
```

**Example:**
```yaml
# Default: stop on first failure
- name: run_tasks
  config:
    tasks:
      - name: generate_deposits
      - name: check_deposits_processed  # Runs after generate_deposits completes

# Continue despite failures
- name: run_tasks
  config:
    continueOnFailure: true
    tasks:
      - name: task_that_might_fail
      - name: cleanup_task  # Runs even if previous failed
```

##### 2.3.2 `run_tasks_concurrent` (Parallel Execution)

**New Defaults:**
- Start all child tasks concurrently
- Wait for all children to complete naturally
- Fail if ANY child fails (default)
- Option to set success/failure thresholds for early termination

**New Config:**
```go
type Config struct {
    Tasks            []helper.RawMessageMasked `yaml:"tasks" json:"tasks"`
    NewVariableScope bool `yaml:"newVariableScope" json:"newVariableScope"`

    // Completion behavior (defaults: wait for all, fail if any fails)
    SuccessThreshold uint64 `yaml:"successThreshold" json:"successThreshold"` // 0 = all must succeed
    FailureThreshold uint64 `yaml:"failureThreshold" json:"failureThreshold"` // Default: 1 (any failure fails)

    // Early termination (default: false - wait for all to complete)
    StopOnThreshold bool `yaml:"stopOnThreshold" json:"stopOnThreshold"`

    // Result transformation
    InvertResult bool `yaml:"invertResult" json:"invertResult"`
    IgnoreResult bool `yaml:"ignoreResult" json:"ignoreResult"`
}
```

**Behavior Matrix:**

| Scenario | Default Behavior | With `stopOnThreshold: true` |
|----------|------------------|------------------------------|
| 1 child fails | Wait for all, then FAIL | Cancel remaining, FAIL immediately |
| All succeed | SUCCESS after all complete | SUCCESS after all complete |
| 3/5 succeed (threshold=3) | N/A (no threshold) | SUCCESS, cancel remaining 2 |
| 2/5 fail (threshold=2) | Wait for all, then FAIL | FAIL, cancel remaining 3 |

**Examples:**
```yaml
# Default: wait for all, fail if any fails
- name: run_tasks_concurrent
  config:
    tasks:
      - name: task_a
      - name: task_b
      - name: task_c
    # Result: SUCCESS only if ALL succeed, FAILURE if ANY fail

# Early termination on success threshold
- name: run_tasks_concurrent
  config:
    successThreshold: 2  # Need 2 successes
    stopOnThreshold: true
    tasks:
      - name: redundant_check_1
      - name: redundant_check_2
      - name: redundant_check_3
    # Result: SUCCESS when 2 complete, cancel the 3rd

# Tolerate some failures
- name: run_tasks_concurrent
  config:
    failureThreshold: 3  # Allow up to 2 failures
    tasks:
      - name: flaky_task_1
      - name: flaky_task_2
      - name: flaky_task_3
      - name: flaky_task_4
      - name: flaky_task_5
    # Result: SUCCESS if at least 3 succeed (2 can fail)
```

##### 2.3.3 `run_task_matrix` (Parameterized Execution)

Same behavior as `run_tasks_concurrent` but for parameterized task instantiation:

```go
type Config struct {
    Task             *helper.RawMessageMasked `yaml:"task" json:"task"`
    MatrixVar        string                   `yaml:"matrixVar" json:"matrixVar"`
    MatrixValues     []any                    `yaml:"matrixValues" json:"matrixValues"`
    RunConcurrent    bool                     `yaml:"runConcurrent" json:"runConcurrent"`

    // Same threshold options as run_tasks_concurrent
    SuccessThreshold uint64 `yaml:"successThreshold" json:"successThreshold"`
    FailureThreshold uint64 `yaml:"failureThreshold" json:"failureThreshold"`
    StopOnThreshold  bool   `yaml:"stopOnThreshold" json:"stopOnThreshold"`

    // Result transformation
    InvertResult bool `yaml:"invertResult" json:"invertResult"`
    IgnoreResult bool `yaml:"ignoreResult" json:"ignoreResult"`
}
```

##### 2.3.4 `run_task_options` (Single Task Wrapper)

Keep as wrapper for single tasks with result transformation:

```go
type Config struct {
    Task             *helper.RawMessageMasked `yaml:"task" json:"task"`
    NewVariableScope bool                     `yaml:"newVariableScope" json:"newVariableScope"`

    // Retry behavior
    RetryOnFailure bool `yaml:"retryOnFailure" json:"retryOnFailure"`
    MaxRetryCount  uint `yaml:"maxRetryCount" json:"maxRetryCount"`

    // Result transformation
    InvertResult  bool `yaml:"invertResult" json:"invertResult"`
    IgnoreResult  bool `yaml:"ignoreResult" json:"ignoreResult"`
    ExpectFailure bool `yaml:"expectFailure" json:"expectFailure"`  // Alias for invertResult
}
```

##### 2.3.5 `run_task_background` (Foreground + Background)

Keep existing behavior but simplify:

```go
type Config struct {
    ForegroundTask *helper.RawMessageMasked `yaml:"foregroundTask" json:"foregroundTask"`
    BackgroundTask *helper.RawMessageMasked `yaml:"backgroundTask" json:"backgroundTask"`
    NewVariableScope bool                   `yaml:"newVariableScope" json:"newVariableScope"`

    // When to complete (based on foreground task)
    ExitOnForegroundSuccess bool `yaml:"exitOnForegroundSuccess" json:"exitOnForegroundSuccess"`
    ExitOnForegroundFailure bool `yaml:"exitOnForegroundFailure" json:"exitOnForegroundFailure"`

    // What happens if background completes first
    // "ignore" (default), "fail", "success"
    OnBackgroundComplete string `yaml:"onBackgroundComplete" json:"onBackgroundComplete"`
}
```

#### 2.4 Removed/Deprecated Options

The following options are **REMOVED** (no backwards compatibility layer):

| Option | Was In | Replacement |
|--------|--------|-------------|
| `stopChildOnResult` | `run_tasks` | Tasks self-complete naturally |
| `expectFailure` | `run_tasks` | Use `invertResult` or `run_task_options` |
| `succeedTaskCount` | `run_tasks_concurrent` | Renamed to `successThreshold` |
| `failTaskCount` | `run_tasks_concurrent` | Renamed to `failureThreshold` |
| `failOnUndecided` | `run_tasks_concurrent` | Removed - behavior is now deterministic |
| `propagateResult` | `run_task_options` | Always propagates |
| `exitOnResult` | `run_task_options` | Tasks self-complete naturally |

**Migration:** Existing YAML files using removed options will fail validation with clear error messages explaining the replacement.

#### 2.5 Implementation Tasks

**Task 2.5.1: Update All Check Tasks (25+ tasks)**

For each check task, modify the Execute loop to return on success:

```go
// Before
case checkResult:
    t.ctx.SetResult(types.TaskResultSuccess)
    // Falls through to select, keeps looping

// After
case checkResult:
    t.ctx.SetResult(types.TaskResultSuccess)
    t.ctx.ReportProgress(100, "Check passed")
    return nil  // EXIT
```

Add `ContinueOnPass` config option handling:
```go
case checkResult:
    t.ctx.SetResult(types.TaskResultSuccess)
    t.ctx.ReportProgress(100, "Check passed")
    if !t.config.ContinueOnPass {
        return nil  // EXIT
    }
    // Continue monitoring
```

**Task 2.5.2: Simplify `run_tasks`**

- Remove `StopChildOnResult` logic and `WatchTaskPass` call
- Keep `ContinueOnFailure` option
- Add `InvertResult` and `IgnoreResult` options

**Task 2.5.3: Simplify `run_tasks_concurrent`**

- Rename `SucceedTaskCount` → `SuccessThreshold`
- Rename `FailTaskCount` → `FailureThreshold`
- Add `StopOnThreshold` option (default: false)
- Remove `FailOnUndecided` (behavior is now: fail if any child failed)
- Add `InvertResult` and `IgnoreResult` options

**Task 2.5.4: Update `run_task_matrix`**

- Mirror changes from `run_tasks_concurrent`
- Remove `FailOnUndecided`

**Task 2.5.5: Simplify `run_task_options`**

- Remove `PropagateResult` (always true now)
- Remove `ExitOnResult` (tasks self-complete)
- Keep `RetryOnFailure`, `MaxRetryCount`
- Keep `InvertResult`, `ExpectFailure`, `IgnoreResult`

**Task 2.5.6: Update `TaskOptions`**

Add new field to `pkg/types/task.go`:
```go
type TaskOptions struct {
    // ... existing fields ...

    // Keep running after success (for monitoring tasks)
    ContinueOnPass bool `yaml:"continueOnPass" json:"continueOnPass"`
}
```

**Task 2.5.7: Update Task Execution**

Modify `pkg/scheduler/task_execution.go` to no longer use `WatchTaskPass` by default. The scheduler's `RunTasks` method should just call `ExecuteTask` and wait for natural completion.

#### 2.6 Test Cases

1. **Check task self-completion**: Verify check tasks return immediately on success
2. **ContinueOnPass**: Verify tasks with `continueOnPass: true` keep running after success
3. **Sequential failure handling**: Verify `run_tasks` stops on failure by default
4. **Sequential continue on failure**: Verify `continueOnFailure: true` continues
5. **Concurrent default behavior**: Verify all children complete before result
6. **Concurrent early termination**: Verify `stopOnThreshold` cancels remaining tasks
7. **Result transformation**: Verify `invertResult` and `ignoreResult` work correctly
8. **Timeout behavior**: Verify existing `Timeout` still works for all task types

---

### Phase 3: React Frontend Development

#### 3.1 Project Setup

**Technology Stack:**
- React 18+ with TypeScript
- Vite for build tooling
- TailwindCSS for styling
- React Query for API state management
- React Flow for graph visualization
- Zustand for global state

**Directory Structure:**
```
web-ui/
├── src/
│   ├── api/              # API client & types
│   ├── components/       # Reusable UI components
│   │   ├── auth/         # Authentication components (UserDisplay)
│   │   ├── common/       # Buttons, inputs, etc.
│   │   ├── task/         # Task-related components
│   │   ├── test/         # Test-related components
│   │   └── graph/        # Graph visualization
│   ├── context/          # React contexts (AuthContext)
│   ├── hooks/            # Custom React hooks (useAuth, useApi)
│   ├── pages/            # Page components
│   │   ├── Dashboard.tsx
│   │   ├── TestRun.tsx
│   │   ├── TestBuilder.tsx
│   │   └── Registry.tsx
│   ├── stores/           # Stores (authStore for JWT management)
│   ├── types/            # TypeScript types
│   └── utils/            # Utility functions
├── public/
├── package.json
├── vite.config.ts
└── tailwind.config.js
```

**Authentication Integration:**

The React frontend implements JWT authentication:

```typescript
// stores/authStore.ts - Singleton auth store
class AuthStore {
  // Fetches token from /auth/token
  async fetchToken(): Promise<AuthState>
  // Returns "Bearer <token>" header or null
  getAuthHeader(): string | null
  // Auto-refresh before expiration
  scheduleRefresh(expiresAt: number): void
  // Redirect to /auth/login
  login(): void
}

// context/AuthContext.tsx - React context provider
<AuthProvider>
  {children}
</AuthProvider>

// hooks/useAuth.ts - React hook
const { isLoggedIn, user, token, loading, login, getAuthHeader } = useAuth();

// components/auth/UserDisplay.tsx - Header login status
<UserDisplay />  // Shows user name or login button
```

**Protected API Calls:**
```typescript
// api/client.ts - Add auth header to protected calls
async function fetchApiWithAuth<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const authHeader = authStore.getAuthHeader();
  // ... add Authorization header if available
}
```

**UI Auth Gating:**
```tsx
// Pages hide admin actions when not authenticated
const { isLoggedIn } = useAuthContext();

{isLoggedIn && (
  <button onClick={handleDelete}>Delete</button>
)}
```

#### 3.2 Core Components

**Dashboard Page:**
- Test runs list with real-time status updates
- Filtering by status, test ID
- Pagination
- Quick actions (run, cancel, delete)

**Test Run Page:**
- Real-time task execution graph
- Task details panel (logs, config, outputs)
- Progress indicators
- Timeline view

**Task Graph Visualization (Execution Flow Graph):**

The graph is built **client-side** from the task list returned by `/api/v1/test_run/{runId}/details`:
- Each task has a `ParentIndex` indicating its parent task
- The frontend constructs a DAG by analyzing parent-child relationships
- Sequential tasks: vertical chain (tasks with same parent, executed in order)
- Concurrent tasks: horizontal branches (multiple children of a `run_tasks_concurrent` parent)
- Tasks rejoin after concurrent sections complete

```tsx
// Using React Flow
interface TaskNode {
  id: string;
  type: 'task';
  data: {
    name: string;
    title: string;
    taskType: string;           // e.g., "check_consensus_finality"
    status: 'pending' | 'running' | 'success' | 'failure' | 'skipped';
    progress: number;
    duration: number;
    hasCondition: boolean;      // Show skip indicator badge
    isCollapsed: boolean;       // For collapsible groups
  };
  position: { x: number; y: number };
}

interface TaskEdge {
  id: string;
  source: string;
  target: string;
  animated: boolean;            // Animate when source is running
  style: { stroke: string };    // Color based on status
}
```

**Graph Layout Algorithm (Client-Side):**

The layout algorithm processes the flat task list into a visual graph:
1. **Build tree**: Group tasks by `ParentIndex` to create hierarchy
2. **Detect concurrency**: Tasks sharing a parent from `run_tasks_concurrent` are siblings
3. **Calculate positions**:
   - **Vertical flow**: Tasks arranged top-to-bottom in execution order
   - **Horizontal branching**: Concurrent siblings spread horizontally
   - **Lane merging**: Parallel tasks rejoin at synchronization points
4. **Interactive features**:
   - Collapse/expand nested task groups
   - Status coloring (edges/nodes colored by execution state)
   - Zoom/pan (standard React Flow controls)
   - Click to select (opens task detail panel)

#### 3.3 Test Builder

**Core Features:**

1. **Task Palette**
   - Searchable list of available tasks
   - Grouped by category (check, generate, get, run, utility)
   - Drag-and-drop to canvas

2. **Visual Canvas**
   - Drop zones for task placement
   - Visual connections between tasks
   - Reorder via drag-and-drop

3. **Task Configuration Panel**
   - Form generated from JSON Schema
   - **Dropdown-based variable selection:**
     - Global variables (test inputs) dropdown
     - Task outputs dropdown (shows outputs from preceding tasks)
     - Available outputs fetched from task descriptor API
   - Config validation feedback
   - Preview of resolved values
   - Badge/icon for tasks with skip conditions

4. **Test Settings**
   - Test name, timeout, scheduling
   - Global variables
   - Import/export YAML

**Component Structure:**
```tsx
// TestBuilder page layout
<TestBuilder>
  <Sidebar>
    <TaskPalette />        {/* Available tasks */}
    <VariablesPanel />     {/* Test variables */}
  </Sidebar>
  <Canvas>
    <TaskGraph />          {/* Visual task arrangement */}
    <DropZone />           {/* For new tasks */}
  </Canvas>
  <ConfigPanel>
    <TaskConfig />         {/* Selected task config */}
    <TestConfig />         {/* Global test settings */}
  </ConfigPanel>
  <Toolbar>
    <ValidateButton />
    <SaveDraftButton />
    <RegisterButton />
    <RunButton />
  </Toolbar>
</TestBuilder>
```

**Drag-and-Drop Implementation:**
```tsx
// Using @dnd-kit/core
import { DndContext, useDraggable, useDroppable } from '@dnd-kit/core';

function TaskPaletteItem({ task }) {
  const { attributes, listeners, setNodeRef } = useDraggable({
    id: task.name,
    data: { type: 'task', task }
  });
  // ...
}

function TaskDropZone({ onDrop }) {
  const { setNodeRef, isOver } = useDroppable({ id: 'canvas' });
  // ...
}
```

#### 3.4 Real-Time Updates

**SSE Integration:**
```tsx
// useEventStream hook
function useEventStream(runId: string) {
  const queryClient = useQueryClient();

  useEffect(() => {
    const eventSource = new EventSource(`/api/v1/test_run/${runId}/events`);

    eventSource.onmessage = (event) => {
      const data = JSON.parse(event.data);

      switch (data.type) {
        case 'task.progress':
          queryClient.setQueryData(['task', data.taskId], (old) => ({
            ...old,
            progress: data.progress
          }));
          break;
        case 'task.completed':
          queryClient.invalidateQueries(['tasks', runId]);
          break;
        // ...
      }
    };

    return () => eventSource.close();
  }, [runId]);
}
```

---

### Phase 4: Integration & Migration

#### 4.2 Router Configuration

**React-Only Frontend (Legacy Removed):**
```go
func (s *Server) setupRoutes() {
    // React app serves all frontend routes
    s.router.PathPrefix("/").Handler(
        http.FileServer(http.FS(frontend.GetFrontendFS())))

    // API routes
    s.router.PathPrefix("/api/").Handler(s.apiRouter)

    // SPA fallback - serve index.html for client-side routing
    s.router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if strings.HasPrefix(r.URL.Path, "/api/") {
            http.NotFound(w, r)
            return
        }
        http.ServeFile(w, r, "index.html")
    })
}
```

#### 4.3 Configuration Migration

**Simplified Config (Legacy Options Removed):**
```yaml
web:
  frontend:
    enabled: true
    siteName: "Assertoor"
  api:
    enabled: true
    cors:
      enabled: true
      origins: ["*"]  # For development
```

**YAML Migration Warnings:**
Old task configuration options will trigger deprecation warnings at startup:
```
WARN: Deprecated config option 'stopChildOnResult' in task 'run_tasks'.
      Use task-level 'runMode' instead. See migration guide.
```

---

## Implementation Timeline

### Milestone 1: Event System & API (Foundation) ✅ COMPLETE
- [x] Event bus implementation (`pkg/events/bus.go`)
- [x] SSE endpoint (`pkg/events/sse.go`, routes in `server.go`)
- [x] Task descriptor schema export (`pkg/tasks/schema.go`, `pkg/web/api/get_task_descriptors_api.go`)
- [x] Add output definitions to all 40+ existing tasks
- [x] Task progress reporting (`TaskContext.ReportProgress`, `taskState.SetProgress`)
- [x] Event emission from task execution (`emitTaskStarted/Completed/Failed/Progress`)
- [x] EventBus integration in coordinator and TaskServices
- [x] Log event emission via EventBus hook (`pkg/logger/eventbushook.go`)
- [x] Progress reporting implemented in all 40+ tasks

### Milestone 2: Task Lifecycle Refactoring ✅ COMPLETE
- [x] Add `ContinueOnPass` to check task configs (added to individual task configs instead of TaskOptions)
- [x] Update all 12 check tasks to return on success (exit loop on pass)
- [x] Add `ContinueOnPass` handling to check tasks
- [x] Simplify `run_tasks`: remove `StopChildOnResult`, add `InvertResult`/`IgnoreResult`
- [x] Simplify `run_tasks_concurrent`: rename thresholds, add `StopOnThreshold`, remove `FailOnUndecided`
- [x] Update `run_task_matrix` to match `run_tasks_concurrent` changes
- [x] Simplify `run_task_options`: remove `PropagateResult`/`ExitOnResult`
- [x] Update `run_task_background` config cleanup
- [x] Remove `WatchTaskPass` usage from glue task child execution (now pass nil)
- [x] Update documentation (READMEs updated for all modified tasks)
- [ ] Write test cases for new behavior (8 test scenarios) - deferred to later

### Milestone 3: React Frontend Core ✅ COMPLETE
- [x] Project setup with React + TypeScript (Note: Using Webpack instead of Vite)
  - React 18.3.1 with TypeScript
  - Webpack 5 with HMR dev server (port 3000)
  - TailwindCSS 3.4.12 for styling
  - Zustand 4.5.5 for state management
  - React Router v6.26.2 for client-side routing
- [x] API client with React Query v5
  - Complete TypeScript API client (`src/api/client.ts`)
  - Auth-aware fetch functions (`fetchApi`, `fetchApiWithAuth`)
  - React Query hooks for all endpoints (`src/hooks/useApi.ts`)
  - Optimized refetch intervals per endpoint type
  - Mutation hooks for admin operations
- [x] Dashboard page (`src/pages/Dashboard.tsx`)
  - Test runs list with table display
  - Multi-select checkboxes for bulk operations
  - Expandable rows with detailed information
  - Status badges, action buttons (view, cancel, delete)
  - "Start Test" modal for authenticated users
- [x] Test run page with task list (`src/pages/TestRun.tsx`)
  - Summary cards (tasks, passed, failed, duration)
  - Resizable split pane layout (40%/60% default)
  - Hierarchical task tree with collapse/expand (`src/components/task/TaskList.tsx`)
  - Task details panel with logs, config YAML, result YAML (`src/components/task/TaskDetails.tsx`)
  - Progress bars for running tasks
- [x] Real-time updates via SSE (`src/hooks/useEventStream.ts`)
  - Per-run event stream (`/api/v1/test_run/{runId}/events`)
  - Global event stream (`/api/v1/events/stream`)
  - Client events stream (`/api/v1/events/clients`)
  - Token-based auth via query parameter
  - Automatic reconnect with 5s timeout
  - React Query cache invalidation and optimistic updates
  - Smart task refresh scheduling with debouncing
- [x] Authentication integration
  - JWT token management (`src/stores/authStore.ts`)
  - Auto-refresh before expiry, session storage persistence
  - AuthContext and useAuth hook (`src/context/AuthContext.tsx`, `src/hooks/useAuth.ts`)
  - UserDisplay component for header (`src/components/auth/UserDisplay.tsx`)
  - UI auth gating for admin actions
- [x] Additional pages implemented
  - Registry page (`src/pages/Registry.tsx`) - test management with register modal
  - TestPage (`src/pages/TestPage.tsx`) - individual test details with run history
  - Clients page (`src/pages/Clients.tsx`) - client status monitoring

### Milestone 4: Graph Visualization (Client-Side) ✅ COMPLETE
- [x] React Flow integration (`reactflow: ^11.11.4`)
- [x] Client-side graph builder from task list (using `ParentIndex` hierarchy)
- [x] Auto-layout algorithm (detect sequential vs concurrent from parent relationships)
  - Topological sort for row assignment
  - Lane assignment to minimize edge crossings
  - Glue tasks hidden (children promoted to visible layer)
- [x] Task node components (`TaskGraphNode.tsx`)
  - Status indicators (pending, running, success, failure, skipped)
  - Progress bars with percentage for running tasks
  - Runtime display with live updates
  - Error messages for failed tasks
- [x] Interactive features (zoom, pan, select)
  - Controls component for zoom/pan
  - MiniMap for navigation
  - Click to select task
  - fitView on initial load and when tasks change
  - Note: Collapse/expand not implemented - glue tasks are hidden automatically
- [x] Live status updates in graph via SSE events
  - TestRun.tsx handles task.created, task.started, task.completed, task.failed, task.progress
  - React Query cache updates trigger graph re-render
- [x] View mode toggle (List/Graph) with localStorage persistence

### Milestone 4.5: Spamoor Library Integration ✅ COMPLETE

**Goal:** Replace custom wallet and transaction management code with the spamoor library for improved transaction submission, tracking, and wallet pool management.

- [x] Remove old `pkg/wallet` package (blobtx, wallet, walletpool, manager)
- [x] Create `pkg/txmgr/spamoor.go` wrapper for spamoor wallet manager
  - WalletManager interface wrapping spamoor functionality
  - GetWalletByPrivkey, GetWalletPoolByPrivkey methods
  - GetClient, GetTxPool for transaction submission
- [x] Update transaction-generating tasks to use spamoor:
  - `generate_transaction` - uses spamoor for tx building and submission
  - `generate_deposits` - uses spamoor for deposit transactions
  - `generate_consolidations` - uses spamoor for consolidation transactions
  - `generate_withdrawal_requests` - uses spamoor for withdrawal transactions
  - `generate_eoa_transactions` - uses spamoor for EOA transactions
  - `generate_blob_transactions` - uses spamoor for blob transactions
  - `generate_child_wallet` - uses spamoor wallet pools for child wallet creation
- [x] Update wallet info tasks:
  - `get_wallet_details` - uses spamoor wallet for balance/nonce queries
- [x] Add new task: `run_spamoor_scenario` for running spamoor scenarios
- [x] Fix WaitGroup double-decrement panic in tasks using `onComplete` callback
  - Spamoor's `onComplete` callback is always called, even on `SendTransaction` error
  - Tasks now only handle cleanup in `onComplete`, not in error path
- [x] Fix wallet pool funding calculation in spamoor
  - Added `calculateFundingAmount` helper: `max(refillAmount, refillBalance - currentBalance)`
  - Ensures wallets reach target balance even when refillAmount is small
- [x] Fix deposit amount overflow using big.Int arithmetic
- [x] Use test run context instead of task context for wallet operations
  - Prevents funding transactions from being cancelled when task completes

**Files Changed:**
```
pkg/txmgr/
├── spamoor.go           # NEW: Spamoor wrapper with WalletManager interface
└── spamoor_scenario.go  # NEW: Scenario execution support

pkg/wallet/              # REMOVED
├── blobtx/              # REMOVED
├── wallet.go            # REMOVED
├── walletpool.go        # REMOVED
└── manager.go           # REMOVED

pkg/tasks/generate_transaction/task.go      # Modified: use spamoor
pkg/tasks/generate_deposits/task.go         # Modified: use spamoor, fix overflow
pkg/tasks/generate_consolidations/task.go   # Modified: use spamoor, fix WaitGroup
pkg/tasks/generate_withdrawal_requests/task.go  # Modified: use spamoor, fix WaitGroup
pkg/tasks/generate_eoa_transactions/task.go # Modified: use spamoor, fix WaitGroup
pkg/tasks/generate_blob_transactions/task.go    # Modified: use spamoor
pkg/tasks/generate_child_wallet/task.go     # Modified: use spamoor, use testRunCtx
pkg/tasks/get_wallet_details/task.go        # Modified: use spamoor
pkg/tasks/run_spamoor_scenario/             # NEW: Run spamoor scenarios
```

**Spamoor Dependency:**
```
github.com/ethpandaops/spamoor v0.x.x
```

### Milestone 5: Test Builder ✅ COMPLETE
- [x] Task palette with search (`src/components/builder/palette/`)
  - Searchable list of available tasks
  - Grouped by category (check, generate, run, utility)
  - Drag-and-drop to canvas
  - `TaskPalette.tsx`, `TaskPaletteSearch.tsx`, `TaskPaletteCategory.tsx`, `TaskPaletteItem.tsx`
- [x] Drag-and-drop canvas (`src/components/builder/`)
  - Uses @dnd-kit/core for drag-and-drop
  - Drop zones for task placement (insert before, insert after, named slots)
  - Visual connections between tasks
  - Reorder via drag-and-drop with multi-select support
  - `BuilderLayout.tsx` - Main DnD context and layout
  - `BuilderCanvas.tsx` - View mode toggle (list/graph/YAML)
- [x] Builder views
  - **List view** (`src/components/builder/list/`)
    - Hierarchical task tree with indentation
    - Tree lines connecting parent/child tasks
    - `BuilderList.tsx`, `BuilderListItem.tsx`
  - **Graph view** (`src/components/builder/graph/`)
    - Interactive React Flow graph visualization
    - Custom node types: `BuilderNode`, `GlueTaskNode`, `StartEndNode`, `DropZoneNode`
    - Auto-layout algorithm with proper sizing
    - `BuilderGraph.tsx`, `useBuilderGraphLayout.ts`
  - **YAML view** (`src/components/builder/yaml/`)
    - CodeMirror YAML editor with syntax highlighting
    - Live sync with builder state
    - `BuilderYaml.tsx`
- [x] JSON Schema form generation (`src/components/builder/config/`)
  - Forms generated from task descriptor schemas
  - Field components: `StringField`, `NumberField`, `BooleanField`, `ArrayField`, `ObjectField`, `DurationField`
  - Expression input with variable references
  - `TaskConfigForm.tsx`, `ConfigPanel.tsx`
- [x] Variable autocomplete (`src/components/builder/config/`)
  - Global variables (test inputs) dropdown
  - Task outputs dropdown (outputs from preceding tasks)
  - JQ expression support via `configVars`
  - `VariableSelector.tsx`, `ExpressionInput.tsx`
- [x] YAML import/export (`src/utils/builder/`)
  - Serialize builder state to YAML
  - Deserialize YAML to builder state
  - Validation with error messages
  - `yamlSerializer.ts`
- [x] Validation & preview (`src/components/builder/validation/`)
  - Real-time validation errors
  - Task type validation against descriptors
  - `ValidationPanel.tsx`
- [x] Builder store (`src/stores/builderStore.ts`)
  - Zustand-based state management
  - Task CRUD operations (add, update, remove, move, duplicate)
  - Selection state (single and multi-select)
  - Test config management (name, timeout, testVars)
  - Cleanup tasks support
- [x] Glue task support
  - **Named children slots** (generalized system for tasks with fixed named inputs)
    - Configuration-based slot definitions (`NAMED_CHILD_TASK_TYPES`)
    - Helper functions: `getNamedChildSlots()`, `hasNamedChildren()`, `getSlotIndex()`, `getSlotName()`
    - Drop zone IDs: `insert-named-{slotName}-{parentId}`
    - Supports `run_task_background` (background/foreground slots)
    - Future-proof for new glue tasks with named inputs
  - **Multi-child tasks** (`run_tasks`, `run_tasks_concurrent`)
    - Unlimited children with add-more placeholder
  - **Single-child tasks** (`run_task_options`, `run_task_matrix`)
    - Single child slot
- [x] Task utilities (`src/utils/builder/taskUtils.ts`)
  - Tree traversal: `findTaskById`, `findParentTask`, `findTaskPath`
  - Task operations: `removeTaskById`, `insertTaskAt`, `moveTaskTo`
  - Validation: `wouldCreateCircular`, `isDescendantOf`
  - Utilities: `getAllTaskIds`, `getAllTasks`, `countTasks`, `getMaxDepth`
  - Preceding tasks for variable context: `findPrecedingTasks`
- [x] Builder toolbar (`src/components/builder/toolbar/`)
  - View mode toggle (list/graph/YAML)
  - Panel visibility controls
  - `BuilderToolbar.tsx`

**Files Created:**
```
web-ui/src/
├── pages/
│   └── TestBuilder.tsx              # Test builder page
├── stores/
│   └── builderStore.ts              # Zustand store for builder state
├── utils/builder/
│   ├── taskUtils.ts                 # Task tree utilities
│   └── yamlSerializer.ts            # YAML serialization
└── components/builder/
    ├── BuilderLayout.tsx            # Main layout with DnD context
    ├── canvas/
    │   └── BuilderCanvas.tsx        # View mode container
    ├── list/
    │   ├── BuilderList.tsx          # List view container
    │   └── BuilderListItem.tsx      # List item with tree lines
    ├── graph/
    │   ├── BuilderGraph.tsx         # React Flow graph
    │   ├── BuilderNode.tsx          # Regular task node
    │   ├── GlueTaskNode.tsx         # Glue task container node
    │   ├── StartEndNode.tsx         # Start/End/Cleanup divider nodes
    │   ├── DropZoneNode.tsx         # Drop zone node
    │   └── useBuilderGraphLayout.ts # Layout algorithm
    ├── yaml/
    │   └── BuilderYaml.tsx          # YAML editor view
    ├── palette/
    │   ├── TaskPalette.tsx          # Palette container
    │   ├── TaskPaletteSearch.tsx    # Search input
    │   ├── TaskPaletteCategory.tsx  # Category accordion
    │   └── TaskPaletteItem.tsx      # Draggable task item
    ├── config/
    │   ├── ConfigPanel.tsx          # Config panel container
    │   ├── TaskConfigForm.tsx       # Form generator
    │   ├── ExpressionInput.tsx      # Expression/variable input
    │   ├── VariableSelector.tsx     # Variable dropdown
    │   └── fields/
    │       ├── StringField.tsx      # String input
    │       ├── NumberField.tsx      # Number input
    │       ├── BooleanField.tsx     # Checkbox input
    │       ├── ArrayField.tsx       # Array editor
    │       ├── ObjectField.tsx      # Object editor
    │       └── DurationField.tsx    # Duration input
    ├── toolbar/
    │   └── BuilderToolbar.tsx       # Toolbar with actions
    └── validation/
        └── ValidationPanel.tsx      # Validation errors display
```

### Milestone 5.2: AI Assistant Features ✅ COMPLETE

**Goal:** AI-assisted test generation with validation, persistence, and seamless integration with the test builder.

- [x] AI YAML validation (`pkg/ai/validator.go`)
  - Validates AI-generated YAML against task registry
  - JSON schema validation for task configs
  - ConfigVars validation (ensures all values are JQ expression strings, not literals)
  - Nested child task validation (supports `tasks`, `task`, `foregroundTask`, `backgroundTask`)
  - Returns structured validation errors with line numbers
- [x] API endpoints for AI assistant
  - `POST /api/v1/ai/generate` - Generate test YAML from natural language
  - `POST /api/v1/ai/validate` - Validate test YAML without registering
  - `GET /api/v1/ai/models` - List available AI models
- [x] Test persistence with full YAML source
  - Database schema: `yaml_source` column in `test_configs` table
  - API-registered tests store full YAML for later editing
  - Tests loaded from DB with `YamlSource` don't apply config as overrides
  - Migrations for SQLite and PostgreSQL
- [x] Global variables resolution fix
  - Test config values set as defaults (`SetDefaultVar`) not overrides
  - Parent scope (global vars) checked before defaults
  - ConfigVars JQ expressions resolved against combined scope
  - Variable query parser handles expressions starting with `.`
- [x] Test builder integration
  - Load existing tests from registry for editing
  - API sends raw YAML with `Content-Type: application/yaml`
  - Frontend YAML editor with syntax highlighting
  - Save/register tests back to registry

**Files Changed:**
```
# Backend
pkg/ai/
├── validator.go           # YAML validation with JSON schema
├── generator.go           # AI test generation
└── handler.go             # AI API endpoints

pkg/db/
├── test_configs.go        # Added YamlSource field
└── schema/
    ├── sqlite/20250203000000_test_yaml_source.sql
    └── pgsql/20250203000000_test_yaml_source.sql

pkg/assertoor/testregistry.go
  - AddLocalTestWithYaml() stores YAML source
  - localTestCfgToDB() skips config when yamlSource present
  - LoadTests() skips config loading when YamlSource present

pkg/test/descriptor.go
  - LoadExternalTestConfig() handles YamlSource
  - LoadTestDescriptors() uses SetDefaultVar for config

pkg/vars/variables.go
  - ResolveQuery() handles expressions starting with "."
  - ConsumeVars() handles expressions starting with "."

pkg/types/test.go
  - ExternalTestConfig.YamlSource field

pkg/types/coordinator.go
  - TestRegistry.AddLocalTestWithYaml() interface method

pkg/web/api/post_tests_register_api.go
  - Accepts raw YAML with Content-Type: application/yaml
  - Stores YAML source for persistence

# Frontend
web-ui/src/api/client.ts
  - registerTest() sends raw YAML

web-ui/src/components/builder/toolbar/BuilderToolbar.tsx
  - Load from Registry feature
  - LoadModal component
```

### Milestone 6: Polish & Migration ✅ COMPLETE

**Goal:** Final polish, security hardening, legacy cleanup, and documentation.

- [x] API Documentation page with Swagger UI
  - Added `swagger-ui-react` dependency
  - Created `ApiDocs.tsx` page with dark mode support
  - Comprehensive CSS overrides for dark mode styling
  - Added "API" nav item to Layout
- [x] Dashboard pagination
  - Page size selector (25, 50, 100, 200) with localStorage persistence
  - First/Prev/Next/Last navigation buttons
  - Page info display ("1-25 of 100")
  - Indeterminate checkbox state for partial selection
- [x] Remove legacy frontend code
  - Removed `pkg/web/handlers/` (handler.go, index.go, test.go, test_run.go, clients.go, registry.go, sidebar.go, logs.go)
  - Removed `pkg/web/templates/` (entire directory)
  - Removed `pkg/web/utils/` (entire directory)
  - Simplified `pkg/web/server.go` (no fallback to legacy)
- [x] System logs API with legacy alias
  - Created `pkg/web/api/get_logs_api.go` with auth protection
  - Route: `GET /api/v1/logs/{since}` (new)
  - Route: `GET /logs/{since}` (legacy alias for external tools)
- [x] Test YAML API endpoint for builder
  - Created `pkg/web/api/get_test_yaml_api.go`
  - Route: `GET /api/v1/test/{testId}/yaml` (protected)
  - Returns stored YAML from DB or loads from external URL
  - Updated builder to use this API for loading existing tests
- [x] UI branding updates
  - Replaced [A] logo with Assertoor mascot image
  - Added mascot image to `web-ui/public/img/assertoor.png`
  - Updated Layout.tsx with proper gap and sizing
- [x] Security hardening & API review
  - **Protected endpoints added:**
    - `GET /api/v1/test/{testId}/yaml` - Full YAML may contain secrets
    - `GET /api/v1/test_run/{runId}/task/{taskId}/result/{resultType}/{fileId}` - Task results may be sensitive
  - **Endpoint modifications:**
    - `GET /api/v1/test/{testId}` - Config/ConfigVars now hidden for guests (only `Vars` was protected before)
  - **Already protected (confirmed):**
    - `GET /api/v1/logs/{since}` - System logs
    - `GET /api/v1/test_run/{runId}/details` - Falls back to basic info without auth
    - `GET /api/v1/test_run/{runId}/task/{taskIndex}/details` - Task details with logs
    - All POST endpoints
- [x] Guest mode UI updates
  - Dashboard: Start/Delete/Cancel buttons already hidden for guests
  - Registry: Register/Delete buttons already hidden for guests
  - TestRun: Cancel button now checks `isLoggedIn`
  - TaskDetails: Shows login message with 🔒 indicator for protected tabs (Logs, Config, Result)
  - TestBuilder: Shows "Log in to save tests" warning, YAML loading gracefully fails for guests
- [x] Swagger docs regenerated for all new/modified endpoints
- [ ] YAML migration guide (deprecated options) - deferred, existing tests continue to work

**Files Changed:**
```
# New files
pkg/web/api/get_logs_api.go           # System logs API
pkg/web/api/get_test_yaml_api.go      # Test YAML API for builder
web-ui/src/pages/ApiDocs.tsx          # Swagger UI page
web-ui/public/img/assertoor.png       # Assertoor mascot logo

# Modified files
pkg/web/server.go                     # Added routes, removed legacy fallback
pkg/web/api/get_test_api.go           # Config/ConfigVars hidden for guests
pkg/web/api/get_task_result_api.go    # Added auth check
pkg/db/test_configs.go                # Added GetTestConfig method
web-ui/src/components/common/Layout.tsx  # Updated logo, added API nav item
web-ui/src/pages/Dashboard.tsx        # Added pagination
web-ui/src/pages/TestRun.tsx          # Cancel button checks auth
web-ui/src/pages/TestBuilder.tsx      # Uses new YAML API
web-ui/src/components/task/TaskDetails.tsx  # Login message for protected tabs
web-ui/src/stores/builderStore.ts     # Added setSourceTestId action
web-ui/src/hooks/useApi.ts            # Added useTestYaml hook
web-ui/src/api/client.ts              # Added getTestYaml function
web-ui/src/types/api.ts               # Added TestYamlResponse type
web-ui/src/App.tsx                    # Added ApiDocs route

# Removed files
pkg/web/handlers/*.go                 # All legacy page handlers
pkg/web/templates/*                   # All Go templates
pkg/web/utils/*                       # Legacy utilities
```

**API Security Summary:**
| Endpoint | Auth Required | Notes |
|----------|---------------|-------|
| `GET /api/v1/tests` | No | Public - test list only |
| `GET /api/v1/test/{id}` | Partial | Config/ConfigVars require auth |
| `GET /api/v1/test/{id}/yaml` | **Yes** | Full YAML with secrets |
| `GET /api/v1/test_runs` | No | Public - run list only |
| `GET /api/v1/test_run/{id}` | No | Public - basic run info |
| `GET /api/v1/test_run/{id}/details` | Partial | Falls back to basic without auth |
| `GET /api/v1/test_run/{id}/task/{idx}/details` | **Yes** | Task logs/config/result |
| `GET /api/v1/test_run/{id}/task/{id}/result/...` | **Yes** | Task result files |
| `GET /api/v1/logs/{since}` | **Yes** | System logs |
| `GET /api/v1/clients` | No | Public - client list |
| `GET /api/v1/task_descriptors` | No | Public - task schemas |
| `POST /api/v1/*` | **Yes** | All mutations protected |

---

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking existing tests | High | Clear migration guide, descriptive validation errors for removed options |
| Complex graph layouts | Medium | Use proven library (React Flow), iterative improvement |
| Real-time performance | Medium | Throttle events, efficient React updates |
| Build complexity | Low | Clear separation, documented build process |
| Task self-completion edge cases | Medium | Thorough testing, `continueOnPass` option for monitoring scenarios |

---

## Design Decisions (Resolved)

### 1. Task Lifecycle

| Question | Decision |
|----------|----------|
| Timeout vs MaxRuntime | **Existing Timeout is sufficient** - No separate `maxRuntime` needed. Timeout serves both "max wait for condition" and "max execution time" |
| Task self-completion | **Tasks return on success** - All tasks exit their Execute loop when they reach terminal state (success/failure) |
| Long-running check edge case | **Optional `continueOnPass`** - Tasks that need to keep monitoring after success use `continueOnPass: true` |
| Glue task child control | **No external control by default** - Glue tasks wait for children to self-complete, don't cancel them |
| Backwards compatibility | **Clean break** - Removed options cause validation errors with migration guidance (no silent mapping) |

### 2. Graph Visualization

| Question | Decision |
|----------|----------|
| Graph data source | **Client-side from existing API** - Build graph from task list using `ParentIndex` hierarchy. No dedicated graph API needed. |
| Nested task hierarchies | **Execution flow graph** - Render as actual DAG with parallel branches for concurrent tasks, not nested containers |
| Collapse/expand support | **Yes, interactive** - Users can collapse/expand task groups to manage complexity |

### 3. Test Builder

| Question | Decision |
|----------|----------|
| Visual conditional branching | **Skip indicators only** - Conditions skip tasks, not branch. Show badge/icon on tasks with conditions, edit in config panel |
| ConfigVars expressions | **Dropdown-based selection** - Dropdowns for global variables and task outputs. Requires task output schema in API |

### 4. Migration

| Question | Decision |
|----------|----------|
| Legacy frontend timeline | **Remove immediately** - No dual frontend support, remove legacy Go templates when React is functional |
| Existing YAML files | **Warn and migrate** - Auto-migrate old configs but log deprecation warnings for old options |

---

## Appendix: File Changes Summary

### New Files
```
# Backend (Milestones 1-2)
pkg/events/bus.go              # ✅ Implemented
pkg/events/event.go            # ✅ Implemented (was types.go)
pkg/events/sse.go              # ✅ Implemented (with auth support for log filtering)
pkg/tasks/schema.go            # ✅ Implemented
pkg/web/api/get_task_descriptors_api.go  # ✅ Implemented
pkg/web/auth/handler.go        # ✅ Implemented - JWT token endpoints
pkg/web/auth/check.go          # ✅ Implemented - Token validation
pkg/logger/eventbushook.go     # ✅ Implemented - Log events via EventBus
pkg/web/frontend/embed.go      # Phase 4

# React Frontend (Milestone 3) ✅ COMPLETE
web-ui/
├── package.json               # ✅ Dependencies (React 18, React Query v5, Webpack, Tailwind)
├── webpack.config.js          # ✅ Webpack 5 config with HMR
├── tsconfig.json              # ✅ TypeScript config (strict mode)
├── tailwind.config.js         # ✅ TailwindCSS configuration
├── src/
│   ├── index.tsx              # ✅ Entry point with React 18 createRoot
│   ├── App.tsx                # ✅ Router setup with AuthProvider
│   ├── api/
│   │   └── client.ts          # ✅ API client with auth support
│   ├── types/
│   │   └── api.ts             # ✅ TypeScript interfaces for all API types
│   ├── stores/
│   │   └── authStore.ts       # ✅ JWT token management singleton
│   ├── context/
│   │   └── AuthContext.tsx    # ✅ Auth context provider
│   ├── hooks/
│   │   ├── useApi.ts          # ✅ React Query hooks for all endpoints
│   │   ├── useAuth.ts         # ✅ Auth state hook
│   │   ├── useEventStream.ts  # ✅ SSE hook for real-time updates
│   │   └── useClientEvents.ts # ✅ Client-specific SSE stream
│   ├── pages/
│   │   ├── Dashboard.tsx      # ✅ Test runs list with bulk operations
│   │   ├── TestRun.tsx        # ✅ Test run details with task list
│   │   ├── TestPage.tsx       # ✅ Individual test with run history
│   │   ├── Registry.tsx       # ✅ Test registry with register modal
│   │   └── Clients.tsx        # ✅ Client status monitoring
│   ├── components/
│   │   ├── common/
│   │   │   ├── Layout.tsx     # ✅ Main layout wrapper
│   │   │   ├── StatusBadge.tsx # ✅ Status indicator
│   │   │   ├── Modal.tsx      # ✅ Modal dialog
│   │   │   ├── Dropdown.tsx   # ✅ Dropdown menu
│   │   │   ├── SplitPane.tsx  # ✅ Resizable split pane
│   │   │   └── Pagination.tsx # ✅ Pagination component
│   │   ├── task/
│   │   │   ├── TaskList.tsx   # ✅ Hierarchical task tree
│   │   │   └── TaskDetails.tsx # ✅ Task detail view with logs
│   │   ├── test/
│   │   │   └── StartTestModal.tsx # ✅ Test scheduling wizard
│   │   ├── auth/
│   │   │   └── UserDisplay.tsx # ✅ Login status component
│   │   └── graph/             # ✅ Milestone 4: Graph visualization
│   │       ├── index.ts       # ✅ Module exports
│   │       ├── TaskGraph.tsx  # ✅ Main graph component with ReactFlow
│   │       ├── TaskGraphNode.tsx # ✅ Custom node component
│   │       └── useTaskGraph.ts # ✅ Graph layout algorithm hook
│   └── utils/
│       └── time.ts            # ✅ Time formatting utilities
```

### Modified Files
```
pkg/types/task.go              # ✅ TaskOutputDefinition, Category, ReportProgress, EmitEvent
                               # Phase 2: Add ContinueOnPass field to TaskOptions
pkg/types/scheduler.go         # ✅ EventBus in TaskServices interface
pkg/scheduler/task_execution.go  # ✅ Event emission, progress reporting
                                 # Phase 2: Remove WatchTaskPass from default execution
pkg/scheduler/task_state.go      # ✅ Progress tracking (SetProgress)
pkg/scheduler/services.go        # ✅ EventBus in servicesProvider
pkg/assertoor/coordinator.go     # ✅ EventBus initialization
pkg/logger/logscope.go           # ✅ EventBus integration for log events
pkg/tasks/*/task.go              # ✅ Output definitions + progress reporting added to all 40+ tasks
                                 # Phase 2: Check tasks return on success, add ContinueOnPass handling
pkg/tasks/run_tasks/             # Phase 2: Remove StopChildOnResult, add InvertResult/IgnoreResult
pkg/tasks/run_tasks_concurrent/  # Phase 2: Rename thresholds, add StopOnThreshold, remove FailOnUndecided
pkg/tasks/run_task_matrix/       # Phase 2: Match run_tasks_concurrent changes
pkg/tasks/run_task_options/      # Phase 2: Remove PropagateResult/ExitOnResult
pkg/tasks/run_task_background/   # Phase 2: Config cleanup
pkg/web/server.go                # ✅ SSE routes, task descriptor routes
Makefile                         # Phase 4 - Build process updates
```

### Removed Files (Legacy Frontend)
```
pkg/web/templates/         # Go templates - REMOVED
pkg/web/handlers/          # Page handlers - REMOVED
pkg/web/static/            # Static assets - REMOVED (replaced by React build)
```
