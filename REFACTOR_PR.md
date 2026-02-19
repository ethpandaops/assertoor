# Assertoor Refactoring - Milestones 1-5

## Overview

This PR implements the first five milestones of the Assertoor refactoring effort:

- **Milestone 1**: Event System & Progress Tracking Infrastructure
- **Milestone 2**: Task Lifecycle Refactoring
- **Milestone 3**: React Frontend Core
- **Milestone 4**: Graph Visualization
- **Milestone 5**: Test Builder

These changes fundamentally improve how tasks execute and communicate their state, provide a modern real-time UI, and enable visual test building with drag-and-drop.

---

## Milestone 1: Event System & Progress Tracking

### New Features

#### Event Bus System (`pkg/events/`)
- New pub/sub event bus for task and test state changes
- Event types: `test.started`, `test.completed`, `test.failed`, `task.started`, `task.progress`, `task.completed`, `task.failed`, `task.log`
- Server-Sent Events (SSE) endpoint for real-time streaming

#### API Endpoints
- `GET /api/v1/events/stream` - SSE endpoint for all events
- `GET /api/v1/test_run/{runId}/events` - SSE for specific test run
- `GET /api/v1/task_descriptors` - List all available tasks with schemas
- `GET /api/v1/clients` - List all configured consensus/execution clients with status

#### Task Progress Reporting
- All 40+ tasks now report progress via `t.ctx.ReportProgress(percent, message)`
- Progress stored in task state and emitted as events
- Database schema extended with `progress` and `progress_message` columns

#### Task Output Definitions
- All tasks now declare their outputs via `TaskOutputDefinition`
- Enables UI dropdowns for variable selection in test builder
- Each output has `Name`, `Type`, and `Description`

#### Log Event Emission
- Task logs are now emitted to EventBus via `EventBusHook`
- Enables live log streaming to frontend

#### JWT Authentication (`pkg/web/auth/`)
- New authentication system using JWT tokens
- `/auth/token` endpoint for obtaining tokens (validates upstream proxy header)
- `/auth/login` endpoint for SSO redirect
- Protected API endpoints require `Authorization: Bearer <token>` header
- SSE streams filter out `task.log` events for unauthenticated clients

**Configuration:**
```yaml
web:
  server:
    authHeader: "X-Forwarded-User"  # Header set by auth proxy
    tokenKey: "your-secret-key"     # JWT signing key
  api:
    enabled: true
    disableAuth: false  # Set to true to disable authentication (default: required)
```

**Protected Endpoints:**
- `POST /api/v1/tests/register`
- `POST /api/v1/tests/register_external`
- `POST /api/v1/tests/delete`
- `POST /api/v1/test_runs/schedule`
- `POST /api/v1/test_runs/delete`
- `POST /api/v1/test_run/{runId}/cancel`

#### React Frontend Auth Integration
- Auth store with automatic token refresh
- Auth context provider for React components
- Login status display in header
- Protected actions hidden when not authenticated
- API client adds Authorization header to protected calls

---

## Milestone 2: Task Lifecycle Refactoring

### Core Change: Tasks Self-Complete

**Before:** Check tasks ran indefinitely until externally cancelled by parent glue tasks.

**After:** Tasks return immediately when they reach a terminal state (success/failure).

```go
// Old behavior - kept running after success
for {
    if checkPassed {
        t.ctx.SetResult(types.TaskResultSuccess)  // Sets result but keeps running!
    }
    select {
    case <-event:
    case <-ctx.Done():
        return ctx.Err()  // Only exits when cancelled
    }
}

// New behavior - exits on success
for {
    if checkPassed {
        t.ctx.SetResult(types.TaskResultSuccess)
        t.ctx.ReportProgress(100, "Check passed")
        return nil  // EXIT immediately
    }
    // ... continue polling until success
}
```

### New Config Option: `continueOnPass`

For check tasks where the condition can regress after passing (e.g., finality checks, health checks), a new `continueOnPass` option allows continued monitoring:

```yaml
- name: check_consensus_finality
  config:
    continueOnPass: true  # Keep monitoring even after success
    timeout: 30m
```

**Behavior with `continueOnPass: true`:**
- Task reports `TaskResultSuccess` when condition passes
- Task continues running and rechecking
- If condition fails later, result changes to `TaskResultNone` or `TaskResultFailure`
- Only exits on timeout or context cancellation

### Check Tasks Updated

The following check tasks now self-complete on success and support `continueOnPass`:

| Task | `continueOnPass` Support |
|------|--------------------------|
| `check_consensus_finality` | ✅ Yes |
| `check_clients_are_healthy` | ✅ Yes |
| `check_consensus_sync_status` | ✅ Yes |
| `check_execution_sync_status` | ✅ Yes |
| `check_consensus_reorgs` | ✅ Yes |
| `check_consensus_attestation_stats` | ✅ Yes |
| `check_eth_call` | ✅ Yes |
| `check_consensus_forks` | ✅ Yes |
| `check_consensus_identity` | ✅ Yes |
| `check_consensus_validator_status` | ✅ Yes |
| `check_consensus_proposer_duty` | ❌ No (one-shot check) |
| `check_consensus_slot_range` | ❌ No (time-based, cannot regress) |
| `check_consensus_block_proposals` | ❌ No (already returns on match) |
| `check_eth_config` | ❌ No (one-shot check) |

### Glue Task Simplification

#### `run_tasks`

**Removed options:**
- `stopChildOnResult` - Tasks now self-complete naturally
- `expectFailure` - Use `invertResult` instead

**New options:**
- `invertResult` - Swap success/failure result
- `ignoreResult` - Always succeed regardless of child result

**Kept options:**
- `continueOnFailure` - Continue to next task even if current fails
- `newVariableScope` - Create isolated variable scope for children

#### `run_tasks_concurrent`

**Renamed options:**
- `succeedTaskCount` → `successThreshold`
- `failTaskCount` → `failureThreshold`

**Removed options:**
- `failOnUndecided` - Behavior is now deterministic (fail if any child failed)

**New options:**
- `stopOnThreshold` - Cancel remaining tasks when threshold reached (default: false)
- `invertResult` - Swap success/failure result
- `ignoreResult` - Always succeed regardless of child results

#### `run_task_matrix`

Same changes as `run_tasks_concurrent`:
- Renamed `succeedTaskCount` → `successThreshold`
- Renamed `failTaskCount` → `failureThreshold`
- Removed `failOnUndecided`
- Added `stopOnThreshold`, `invertResult`, `ignoreResult`

#### `run_task_options`

**Removed options:**
- `propagateResult` - Always propagates now
- `exitOnResult` - Tasks self-complete naturally
- `ignoreFailure` - Renamed to `ignoreResult`

**Kept options:**
- `retryOnFailure` - Retry task on failure
- `maxRetryCount` - Maximum retry attempts
- `invertResult` - Swap success/failure result
- `expectFailure` - Alias for `invertResult`
- `ignoreResult` - Always succeed (renamed from `ignoreFailure`)
- `newVariableScope` - Create isolated variable scope

#### `run_task_background`

No breaking changes. Config reorganized for clarity.

---

## Breaking Changes Summary

| Removed Option | Task | Replacement |
|----------------|------|-------------|
| `stopChildOnResult` | `run_tasks` | Not needed - tasks self-complete |
| `expectFailure` | `run_tasks` | Use `invertResult` or wrap with `run_task_options` |
| `succeedTaskCount` | `run_tasks_concurrent` | Renamed to `successThreshold` |
| `failTaskCount` | `run_tasks_concurrent` | Renamed to `failureThreshold` |
| `failOnUndecided` | `run_tasks_concurrent` | Removed - if any child fails, result is failure |
| `succeedTaskCount` | `run_task_matrix` | Renamed to `successThreshold` |
| `failTaskCount` | `run_task_matrix` | Renamed to `failureThreshold` |
| `failOnUndecided` | `run_task_matrix` | Removed - if any child fails, result is failure |
| `propagateResult` | `run_task_options` | Removed - always propagates |
| `exitOnResult` | `run_task_options` | Not needed - tasks self-complete |
| `ignoreFailure` | `run_task_options` | Renamed to `ignoreResult` |

---

## Migration Guide

### 1. Remove `stopChildOnResult` from `run_tasks`

**Before:**
```yaml
- name: run_tasks
  config:
    stopChildOnResult: true  # No longer needed
    tasks:
      - name: check_consensus_finality
```

**After:**
```yaml
- name: run_tasks
  config:
    tasks:
      - name: check_consensus_finality
        # Task automatically returns when finality check passes
```

### 2. Replace `expectFailure` with `invertResult`

**Before:**
```yaml
- name: run_tasks
  config:
    expectFailure: true
    tasks:
      - name: some_task_that_should_fail
```

**After:**
```yaml
- name: run_tasks
  config:
    invertResult: true
    tasks:
      - name: some_task_that_should_fail
```

Or use `run_task_options`:
```yaml
- name: run_task_options
  config:
    expectFailure: true
    task:
      name: some_task_that_should_fail
```

### 3. Rename threshold options in `run_tasks_concurrent`

**Before:**
```yaml
- name: run_tasks_concurrent
  config:
    succeedTaskCount: 3
    failTaskCount: 2
    tasks: [...]
```

**After:**
```yaml
- name: run_tasks_concurrent
  config:
    successThreshold: 3
    failureThreshold: 2
    tasks: [...]
```

### 4. Handle `failOnUndecided` removal

**Before:**
```yaml
- name: run_tasks_concurrent
  config:
    failOnUndecided: true
    tasks: [...]
```

**After:**
The new default behavior is: if any child task fails, the result is failure. No explicit option needed.

If you want to tolerate some failures:
```yaml
- name: run_tasks_concurrent
  config:
    failureThreshold: 3  # Allow up to 2 failures before failing
    tasks: [...]
```

### 5. Rename `ignoreFailure` to `ignoreResult` in `run_task_options`

**Before:**
```yaml
- name: run_task_options
  config:
    ignoreFailure: true
    task: {...}
```

**After:**
```yaml
- name: run_task_options
  config:
    ignoreResult: true
    task: {...}
```

### 6. Use `continueOnPass` for long-running monitoring

If you have check tasks that need to keep monitoring after initial success:

**Before (with glue task control):**
```yaml
- name: run_tasks_concurrent
  config:
    tasks:
      - name: check_consensus_finality
        timeout: 30m
      - name: generate_transactions
```

**After (with explicit continueOnPass):**
```yaml
- name: run_tasks_concurrent
  config:
    tasks:
      - name: check_consensus_finality
        config:
          continueOnPass: true  # Keep monitoring finality
        timeout: 30m
      - name: generate_transactions
```

---

## Testing

All changes have been verified to compile. Recommended test scenarios:

1. **Check task self-completion**: Verify check tasks return immediately on success
2. **`continueOnPass`**: Verify tasks with `continueOnPass: true` keep running after success
3. **Sequential failure handling**: Verify `run_tasks` stops on failure by default
4. **Concurrent thresholds**: Verify `successThreshold` and `failureThreshold` work correctly
5. **Result transformation**: Verify `invertResult` and `ignoreResult` work correctly
6. **Timeout behavior**: Verify existing `timeout` still works for all task types
7. **Authentication**: Verify protected endpoints return 401 without valid token
8. **SSE log filtering**: Verify `task.log` events are not sent to unauthenticated clients
9. **Auth disabled mode**: Verify `disableAuth: true` allows all requests without token

---

## Configuration Changes

### New Configuration Options

```yaml
web:
  server:
    # Authentication settings (for JWT token generation)
    authHeader: "X-Forwarded-User"  # Header containing authenticated username
    tokenKey: "your-secret-key"     # Secret key for signing JWT tokens
  api:
    enabled: true
    disableAuth: false  # Set to true to disable API authentication
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `WEB_SERVER_AUTH_HEADER` | Header containing authenticated username | (empty) |
| `WEB_SERVER_TOKEN_KEY` | Secret key for JWT signing | (empty) |
| `WEB_API_DISABLE_AUTH` | Disable API authentication | `false` |

### Authentication Behavior

- **Default (auth enabled)**: Protected endpoints require valid JWT token
- **Auth disabled**: All endpoints accessible without authentication
- **No token key**: If `tokenKey` is empty, auth is effectively disabled

---

## Milestone 3: React Frontend Core

### Technology Stack

- **React 18.3.1** with TypeScript
- **Webpack 5** with HMR dev server (port 3000)
- **TailwindCSS 3.4.12** for styling
- **React Query v5** for API state management
- **Zustand 4.5.5** for global state
- **React Router v6.26.2** for client-side routing

### New Frontend Structure

```
web-ui/
├── src/
│   ├── api/client.ts          # API client with auth support
│   ├── types/api.ts           # TypeScript interfaces
│   ├── stores/authStore.ts    # JWT token management
│   ├── context/AuthContext.tsx # Auth context provider
│   ├── hooks/
│   │   ├── useApi.ts          # React Query hooks
│   │   ├── useAuth.ts         # Auth state hook
│   │   ├── useEventStream.ts  # SSE hook for real-time updates
│   │   └── useClientEvents.ts # Client-specific SSE stream
│   ├── pages/
│   │   ├── Dashboard.tsx      # Test runs list with bulk operations
│   │   ├── TestRun.tsx        # Test run details with task list/graph
│   │   ├── TestPage.tsx       # Individual test with run history
│   │   ├── Registry.tsx       # Test registry with register modal
│   │   └── Clients.tsx        # Client status monitoring
│   └── components/
│       ├── common/            # Layout, StatusBadge, Modal, SplitPane, etc.
│       ├── task/              # TaskList, TaskDetails
│       ├── test/              # StartTestModal
│       ├── auth/              # UserDisplay
│       └── graph/             # TaskGraph, TaskGraphNode (Milestone 4)
```

### Key Features

#### Dashboard Page
- Test runs list with table display
- Multi-select checkboxes for bulk operations (delete)
- Expandable rows with detailed information
- Status badges, action buttons (view, cancel, delete)
- "Start Test" modal for authenticated users

#### Test Run Page
- Summary cards (tasks, passed, failed, duration)
- Resizable split pane layout (40%/60% default)
- Hierarchical task tree with collapse/expand
- Task details panel with logs, config YAML, result YAML
- Progress bars for running tasks
- View mode toggle (List/Graph)

#### Real-time Updates via SSE
- Per-run event stream (`/api/v1/test_run/{runId}/events`)
- Global event stream (`/api/v1/events/stream`)
- Client events stream (`/api/v1/events/clients`)
- Token-based auth via query parameter
- Automatic reconnect with 5s timeout
- React Query cache invalidation and optimistic updates

#### Authentication Integration
- JWT token management with auto-refresh
- Session storage persistence
- AuthContext and useAuth hook
- UserDisplay component for header
- UI auth gating for admin actions

#### Additional Pages
- **Registry**: Test management with register modal, run history
- **TestPage**: Individual test details with run history
- **Clients**: Client status monitoring with real-time updates

---

## Milestone 4: Graph Visualization

### React Flow Integration

The task graph visualization uses **React Flow v11.11.4** to render an interactive DAG (Directed Acyclic Graph) of task execution.

### Graph Layout Algorithm

The graph is built **client-side** from the task list returned by the API:

1. **Build tree**: Group tasks by `parent_index` to create hierarchy
2. **Detect glue tasks**: Identify orchestration tasks (`run_tasks`, `run_tasks_concurrent`, etc.)
3. **Hide glue tasks**: Glue tasks are not rendered; their children are promoted to the visible layer
4. **Detect concurrency**: Tasks under `run_tasks_concurrent` or `run_task_matrix` are siblings (parallel lanes)
5. **Topological sort**: Assign rows using BFS from sources (respects execution order)
6. **Lane assignment**: Assign columns to minimize edge crossings, inheriting lanes from predecessors
7. **Center graph**: Offset positions to center the graph horizontally

### Task Node Components

Each visible task is rendered as a `TaskGraphNode` with:

- **Status indicator**: Color-coded dot (pending=gray, running=blue, success=green, failure=red, skipped=yellow)
- **Pulsing animation**: Running tasks have animated status dot
- **Task index**: Displayed in header
- **Runtime**: Live-updating duration for running tasks
- **Title/Name**: Task title with name subtitle if different
- **Progress bar**: For running tasks with progress > 0
- **Progress message**: Displayed below progress bar
- **Error message**: Displayed for failed tasks (truncated with tooltip)

### Edge Styling

Edges between nodes are styled based on the target task's status:
- **Gray**: Pending
- **Blue (animated)**: Running
- **Green**: Success
- **Red**: Failure

Edge type:
- **Straight**: Same lane (vertical connection)
- **Smoothstep**: Different lanes (curved connection)

### Interactive Features

| Feature | Implementation |
|---------|----------------|
| Zoom | Controls component (+/- buttons) |
| Pan | Drag on background |
| Fit View | Control button, auto on load/task add |
| MiniMap | Bottom-right overview with status colors |
| Select | Click node to select, opens detail panel |
| Keyboard | Arrow keys via React Flow |

### Live Status Updates

The graph receives live updates via SSE:

1. **task.created**: New task added to task list, graph re-renders
2. **task.started**: Task details fetched, status updated
3. **task.progress**: Progress bar updated
4. **task.completed/failed**: Final status reflected, edge color changes

Updates flow through React Query cache, triggering automatic re-renders of the graph with new node states.

### View Mode Toggle

Users can switch between List and Graph views:
- Toggle button in task panel header
- Preference persisted in localStorage
- Both views share selection state

---

## Milestone 4.5: Spamoor Library Integration

### Overview

This milestone integrates the **spamoor** library (`github.com/ethpandaops/spamoor`) as the foundation for wallet management and transaction submission. This replaces the custom `pkg/wallet` package with a battle-tested library that provides:

- Robust transaction pool management with rebroadcast support
- Wallet pools with automatic refunding
- Transaction building utilities (legacy, dynamic fee, blob)
- Completion callbacks for tracking transaction confirmations

### Changes

#### Removed: `pkg/wallet` Package

The following files have been removed:
- `pkg/wallet/blobtx/blob_encode.go` - Blob encoding utilities
- `pkg/wallet/blobtx/blobtx.go` - Blob transaction building
- `pkg/wallet/wallet.go` - Wallet implementation
- `pkg/wallet/walletpool.go` - Wallet pool management
- `pkg/wallet/manager.go` - Wallet manager

#### New: `pkg/txmgr` Package

A new package wraps spamoor functionality for assertoor:

```go
// pkg/txmgr/spamoor.go
type WalletManager interface {
    GetWalletByPrivkey(ctx context.Context, privKey *ecdsa.PrivateKey) (*spamoor.Wallet, error)
    GetWalletPoolByPrivkey(ctx context.Context, logger logrus.FieldLogger, privKey *ecdsa.PrivateKey, config *WalletPoolConfig) (*spamoor.WalletPool, error)
    GetClient(client *execution.Client) *spamoor.Client
    GetTxPool() *spamoor.TxPool
}

type WalletPoolConfig struct {
    WalletCount   uint64
    WalletSeed    string
    RefillAmount  *uint256.Int  // Amount to send per refill tx
    RefillBalance *uint256.Int  // Target balance threshold
}
```

#### Updated Tasks

The following tasks now use spamoor for transaction operations:

| Task | Changes |
|------|---------|
| `generate_transaction` | Uses spamoor for tx building and submission |
| `generate_deposits` | Uses spamoor; fixed amount overflow with big.Int |
| `generate_consolidations` | Uses spamoor; fixed WaitGroup double-decrement |
| `generate_withdrawal_requests` | Uses spamoor; fixed WaitGroup double-decrement |
| `generate_eoa_transactions` | Uses spamoor wallet pools; fixed WaitGroup |
| `generate_blob_transactions` | Uses spamoor for blob tx building |
| `generate_child_wallet` | Uses spamoor wallet pools; uses testRunCtx |
| `get_wallet_details` | Uses spamoor wallet for queries |

#### New Task: `run_spamoor_scenario`

A new task allows running spamoor scenarios directly within assertoor tests:

```yaml
- name: run_spamoor_scenario
  config:
    scenarioName: "blob-spammer"
    scenarioConfig:
      # Scenario-specific configuration
```

### Bug Fixes

#### WaitGroup Double-Decrement Panic

**Problem:** Tasks using spamoor's `SendTransaction` with an `onComplete` callback experienced `sync: negative WaitGroup counter` panics.

**Cause:** Spamoor's `onComplete` callback is **always** called, even when `SendTransaction` returns an error. Tasks were draining `pendingChan` and calling `pendingWg.Done()` in both the error path and the callback.

**Fix:** Removed cleanup from error paths; `onComplete` now handles all cleanup:

```go
// Before (PANIC on error)
if err != nil {
    <-pendingChan       // WRONG: onComplete also drains
    pendingWg.Done()    // WRONG: onComplete also calls Done
    return err
}

// After (correct)
if err != nil {
    t.logger.Errorf("error: %v", err)
    // Note: onComplete callback is still called by spamoor even on error,
    // so we don't drain pendingChan or call pendingWg.Done() here
}
```

#### Wallet Pool Funding Calculation

**Problem:** Child wallets weren't reaching target balance when `refillAmount` was smaller than the difference between target and current balance.

**Cause:** Spamoor was sending exactly `refillAmount` per transaction, regardless of how much was needed to reach `refillBalance`.

**Fix:** Added `calculateFundingAmount` in spamoor:

```go
func calculateFundingAmount(currentBalance, refillAmount, refillBalance *uint256.Int) *uint256.Int {
    fundingAmount := new(uint256.Int).Set(refillAmount)
    if refillBalance.Cmp(currentBalance) > 0 {
        neededAmount := new(uint256.Int).Sub(refillBalance, currentBalance)
        if neededAmount.Cmp(fundingAmount) > 0 {
            fundingAmount = neededAmount
        }
    }
    return fundingAmount
}
```

#### Deposit Amount Overflow

**Problem:** Large deposit amounts (e.g., 32 ETH) overflowed when converted to Gwei.

**Fix:** Use big.Int arithmetic with overflow check:

```go
depositAmountGwei := new(big.Int).SetUint64(t.config.DepositAmount)
depositAmountGwei.Mul(depositAmountGwei, big.NewInt(1000000000))
if !depositAmountGwei.IsUint64() {
    return nil, nil, fmt.Errorf("deposit amount too large: %v ETH", t.config.DepositAmount)
}
```

#### Task Context vs Test Run Context

**Problem:** `generate_child_wallet` funding transactions were cancelled when the task completed.

**Fix:** Use `testRunCtx` for wallet operations that should outlive the task:

```go
// Use test run context for wallet operations
testRunCtx := t.ctx.Scheduler.GetTestRunCtx()
rootWallet, err := walletMgr.GetWalletByPrivkey(testRunCtx, privKey)
```

---

## Milestone 5: Test Builder

### Overview

A complete visual test builder that allows users to create and edit Assertoor tests through a drag-and-drop interface.

### Features

#### Task Palette
- Searchable list of all available tasks
- Grouped by category (check, generate, run, utility)
- Drag tasks from palette to canvas

#### Three View Modes

**List View:**
- Hierarchical task tree with indentation and tree lines
- Drag-and-drop reordering
- Multi-select support for bulk operations

**Graph View:**
- Interactive React Flow visualization
- Custom node types for different task kinds
- Visual connections showing task flow
- Auto-layout algorithm

**YAML View:**
- CodeMirror editor with YAML syntax highlighting
- Live sync with builder state
- Direct editing for power users

#### Task Configuration Panel
- Auto-generated forms from JSON Schema
- Field types: string, number, boolean, duration, array, object
- Variable selector dropdown for referencing:
  - Test variables (global inputs)
  - Task outputs (from preceding tasks)
- JQ expression support via `configVars`

#### Glue Task Support

**Named Children System:**

A generalized approach for glue tasks with fixed named input slots (e.g., `run_task_background` with background/foreground slots):

```typescript
// Configuration-based slot definitions
export const NAMED_CHILD_TASK_TYPES: Record<string, NamedChildSlot[]> = {
  'run_task_background': [
    { name: 'background', label: 'BG', yamlKey: 'backgroundTask', colorClass: 'amber' },
    { name: 'foreground', label: 'FG', yamlKey: 'foregroundTask', colorClass: 'emerald' },
  ],
  // Future glue tasks with named slots can be added here
};
```

- Drop zones for each named slot with visual indicators
- Drag between slots (e.g., move task from background to foreground)
- Color-coded labels for slot identification
- Future-proof for new glue task types

**Other Glue Tasks:**
- `run_tasks` / `run_tasks_concurrent`: Unlimited children with add-more placeholder
- `run_task_options` / `run_task_matrix`: Single child slot

#### Validation
- Real-time validation errors display
- Task type validation against descriptors
- YAML syntax validation

#### YAML Import/Export
- Load existing test YAML
- Export builder state to YAML
- Round-trip preservation of structure

### Component Structure

```
web-ui/src/components/builder/
├── BuilderLayout.tsx            # Main layout with DnD context
├── canvas/BuilderCanvas.tsx     # View mode container
├── list/
│   ├── BuilderList.tsx          # List view
│   └── BuilderListItem.tsx      # Tree item with drop zones
├── graph/
│   ├── BuilderGraph.tsx         # React Flow graph
│   ├── BuilderNode.tsx          # Task node
│   ├── GlueTaskNode.tsx         # Container node
│   ├── StartEndNode.tsx         # Start/End markers
│   ├── DropZoneNode.tsx         # Drop targets
│   └── useBuilderGraphLayout.ts # Layout algorithm
├── yaml/BuilderYaml.tsx         # YAML editor
├── palette/
│   ├── TaskPalette.tsx          # Palette container
│   ├── TaskPaletteSearch.tsx    # Search
│   ├── TaskPaletteCategory.tsx  # Category accordion
│   └── TaskPaletteItem.tsx      # Draggable item
├── config/
│   ├── ConfigPanel.tsx          # Config panel
│   ├── TaskConfigForm.tsx       # Form generator
│   ├── ExpressionInput.tsx      # Expression input
│   ├── VariableSelector.tsx     # Variable dropdown
│   └── fields/                  # Field components
├── toolbar/BuilderToolbar.tsx   # Toolbar
└── validation/ValidationPanel.tsx
```

### State Management

**Builder Store (`src/stores/builderStore.ts`):**
- Zustand-based state management
- Test configuration (name, timeout, testVars)
- Task tree with children and namedChildren
- Selection state (multi-select support)
- CRUD operations with immutable updates

**Task Utilities (`src/utils/builder/taskUtils.ts`):**
- Tree traversal functions
- Task manipulation (move, insert, remove)
- Circular reference detection
- Preceding task calculation for variable context

### Technologies Used
- **@dnd-kit/core**: Drag-and-drop framework
- **React Flow**: Graph visualization
- **CodeMirror**: YAML editor
- **Zustand**: State management
- **js-yaml**: YAML parsing/serialization
