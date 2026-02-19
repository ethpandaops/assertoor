# Milestone 7: Test Debugger Implementation Plan

This document outlines the implementation plan for the interactive test debugger feature, which integrates with the test builder to allow step-by-step test execution with inspection and modification capabilities.

## Implementation Progress

| Phase | Status | Description |
|-------|--------|-------------|
| 7.1 | ✅ Complete | Debug Session Core - Manager, Session, Frame, Breakpoint, Hook integration |
| 7.1.1 | ✅ Complete | Glue Task Integration - Concurrent/background frame creation |
| 7.2 | ✅ Complete | Debug Events - Event types and SSE publishing |
| 7.3 | ✅ Complete | Debug API Endpoints |
| 7.4 | ⏳ Pending | React UI - Debug Components |
| 7.5 | ⏳ Pending | Builder Integration |

### Files Created/Modified in Phase 7.1

**New Files:**
- `pkg/debugger/errors.go` - Error types (ErrSkipTask, ErrAbortDebug, etc.)
- `pkg/debugger/types.go` - State enums (DebugState, FrameState, PauseReason, StepMode, etc.)
- `pkg/debugger/hook.go` - TaskHookInfo alias, InjectedTask struct
- `pkg/debugger/breakpoint.go` - Breakpoint struct with matching logic
- `pkg/debugger/frame.go` - ExecutionFrame and TaskPauseInfo structs
- `pkg/debugger/session.go` - DebugSession implementing types.DebugHook
- `pkg/debugger/manager.go` - DebugManager for session lifecycle
- `pkg/types/debug.go` - DebugHook interface, DebugTaskInfo, InjectedTaskInfo

**Modified Files:**
- `pkg/types/scheduler.go` - Added debug methods to TaskSchedulerRunner
- `pkg/types/test.go` - Added SetDebugHook to TestRunner interface
- `pkg/scheduler/scheduler.go` - Added debugHook, taskFrameMap, debug methods
- `pkg/scheduler/task_execution.go` - Integrated OnBeforeTask/OnAfterTask hooks
- `pkg/events/event.go` - Added debug event types and data structs
- `pkg/events/sse.go` - Added debug event publishing methods
- `pkg/test/test.go` - Implemented SetDebugHook method
- `pkg/assertoor/testrunner.go` - Added debugManager and ScheduleTestDebug

### Files Created/Modified in Phase 7.1.1

**Modified Files:**
- `pkg/tasks/run_tasks_concurrent/task.go` - Added debug frame creation for each concurrent child task via `OnConcurrentStart`/`OnConcurrentEnd`
- `pkg/tasks/run_task_background/task.go` - Added frame creation for background and foreground tasks with proper frame cleanup
- `pkg/tasks/run_task_matrix/task.go` - Added frame creation when `RunConcurrent` is true

### Files Created/Modified in Phase 7.3

**New Files:**
- `pkg/debugger/eval.go` - Helper type (`evalVars`) for JQ expression evaluation in REPL
- `pkg/web/api/debug_api.go` - All debug API handlers (session management, execution control, breakpoints, variable inspection, expression evaluation)

**Modified Files:**
- `pkg/types/debug.go` - Added `DebugSession` interface for API layer abstraction, `DebugFrameInfo`, `DebugTaskPauseInfo`, `DebugBreakpointConfig`, `DebugBreakpointInfo` types
- `pkg/types/coordinator.go` - Added debug support methods (`ScheduleTestDebug`, `GetDebugSession`, `DeleteDebugSession`)
- `pkg/assertoor/coordinator.go` - Implemented debug methods calling through to runner
- `pkg/debugger/session.go` - Updated method signatures to implement `types.DebugSession` interface, added `GetID`, `GetRunID`, `GetState`, `GetCreatedAt`, `GetLastActivity`, `GetActiveFrameID`, `DeletePendingVarChange`, `EvaluateExpression` methods
- `pkg/debugger/breakpoint.go` - Added `NewBreakpointFromConfig` function and `ToBreakpointInfo` method
- `pkg/debugger/frame.go` - Added `ToFrameInfo` method and `DeletePendingVarChange` method
- `pkg/web/server.go` - Registered all debug API routes

---

## Overview

The test debugger enables developers to:
- Run tests in debug mode with execution pausing before tasks
- Set breakpoints on specific tasks, task types, or conditions
- Step through tasks (into, over, out)
- Inspect variables and resolved task configurations at pause points
- Modify variables before task execution
- Inject additional tasks during execution
- Skip tasks during debugging

## Design Decisions

| Question | Decision |
|----------|----------|
| Attach to running test | **No** - Debug mode must be enabled at test start |
| Multi-user debugging | **No** - Single debug session per test run, but same user can reconnect after page reload |
| Session persistence | **In-memory only** - Sessions don't survive server restart, but persist during test run for reconnection |
| Conditional breakpoints | **Yes** - Use existing JQ infrastructure for condition evaluation |
| Cleanup tasks | **Debuggable** - Debug hooks apply to cleanup tasks, but with separate "cleanup mode" flag |
| Variable modification scope | **Future tasks only** - Changes apply to tasks not yet config-resolved |
| Timeout during pause | **Paused** - Task timeout timer pauses while debug is paused |

---

## Consistency Standards

### Type Usage
- Use `types.TaskIndex` consistently (not raw `uint64`) for task identifiers
- Use `uint64` for frame IDs (new type, not in existing codebase)
- Use existing `types.TaskResult`, `types.Variables`, etc.

---

## Architecture

### Debug Session Model

Due to concurrent execution (`run_tasks_concurrent`, `run_task_background`), multiple tasks can be executing in parallel. The debugger must support **multiple execution frames** that can be paused and stepped independently.

```go
// pkg/debugger/session.go

type DebugSession struct {
    ID              string              // Unique session ID
    RunID           uint64              // Associated test run
    State           DebugState          // running, paused, stepping, completed
    Breakpoints     []*Breakpoint       // Active breakpoints

    // Execution frames (one per parallel execution path)
    Frames          map[uint64]*ExecutionFrame  // frameID -> frame
    nextFrameID     uint64
    ActiveFrameID   uint64              // Currently selected frame in UI

    // Global controls
    PauseAllOnBreakpoint bool           // Pause all frames when any hits breakpoint
    GlobalPauseRequested bool           // User requested pause on all frames

    // Pending modifications (applied to specific frame's scope)
    InjectedTasks   []*InjectedTask     // Tasks to inject (frame-specific)

    // Synchronization
    mu              sync.RWMutex

    // Timestamps
    CreatedAt       time.Time
    LastActivity    time.Time
}

// ExecutionFrame represents a single parallel execution path
// Each concurrent branch or background task gets its own frame
type ExecutionFrame struct {
    ID              uint64              // Unique frame ID
    ParentFrameID   uint64              // Parent frame (0 for root)
    State           FrameState          // running, paused, completed
    Name            string              // Human-readable name (e.g., "concurrent[0]", "background")

    // Current pause state
    PauseReason     PauseReason         // breakpoint, step, user_pause
    CurrentTask     *TaskPauseInfo      // Info about paused task (nil if running)

    // Step execution (per-frame)
    StepMode        StepMode            // none, into, over, out
    StepFromTask    uint64              // Task we're stepping from
    StepFromDepth   int                 // Depth when step started

    // Pending modifications for this frame
    PendingVarChanges map[string]any    // Variables to set before next task
    SkipNextTask      bool              // Skip the current/next task

    // Frame context
    RootTaskIndex   uint64              // The glue task that spawned this frame
    TaskIndices     []uint64            // All tasks in this frame's execution path

    // Synchronization
    resumeChan      chan struct{}       // Receives signal to resume this frame
}

type FrameState int
const (
    FrameStateRunning FrameState = iota
    FrameStatePaused
    FrameStateCompleted
)

type DebugState int
const (
    DebugStateRunning DebugState = iota
    DebugStatePaused
    DebugStateStepping
    DebugStateCompleted
)

type PauseReason string
const (
    PauseReasonBreakpoint PauseReason = "breakpoint"
    PauseReasonStep       PauseReason = "step"
    PauseReasonUserPause  PauseReason = "user_pause"
    PauseReasonInitial    PauseReason = "initial"  // Paused at start
)

type StepMode int
const (
    StepModeNone StepMode = iota
    StepModeInto  // Stop at next task (including children)
    StepModeOver  // Stop at next sibling (after current task + children complete)
    StepModeOut   // Stop when exiting current parent
)

type TaskPauseInfo struct {
    TaskIndex       uint64
    TaskID          string
    TaskName        string
    TaskTitle       string
    ParentIndex     uint64
    Depth           int                 // Nesting depth
    ResolvedConfig  map[string]any      // Config with variables resolved
    ConfigVars      map[string]string   // Original configVars expressions
    AvailableVars   map[string]any      // All variables in scope
}

type InjectedTask struct {
    ID           string              // Unique ID for tracking
    TaskOptions  *types.TaskOptions  // Task definition
    InsertMode   InsertMode          // before_current, after_current
    Executed     bool                // Has this been executed?
}

type InsertMode string
const (
    InsertBeforeCurrent InsertMode = "before_current"
    InsertAfterCurrent  InsertMode = "after_current"
)
```

### Breakpoint Model

```go
// pkg/debugger/breakpoint.go

type Breakpoint struct {
    ID        string          // Unique identifier
    Type      BreakpointType
    Enabled   bool
    HitCount  int             // Number of times hit

    // Type-specific fields
    TaskIndex uint64          // For TaskIndex type
    TaskID    string          // For TaskID type
    TaskName  string          // For TaskName type (task type name)
    Condition string          // For Conditional type (JQ expression)
}

type BreakpointType int
const (
    BreakpointTypeTaskIndex BreakpointType = iota  // Break at specific task index
    BreakpointTypeTaskID                            // Break at task with matching ID
    BreakpointTypeTaskName                          // Break at any task of this type
    BreakpointTypeConditional                       // Break when JQ condition is true
)

// Match checks if breakpoint should trigger for given task
func (b *Breakpoint) Match(taskState *taskState, vars types.Variables) bool {
    if !b.Enabled {
        return false
    }

    switch b.Type {
    case BreakpointTypeTaskIndex:
        return uint64(taskState.taskIndex) == b.TaskIndex
    case BreakpointTypeTaskID:
        return taskState.taskOptions.ID == b.TaskID
    case BreakpointTypeTaskName:
        return taskState.taskOptions.Name == b.TaskName
    case BreakpointTypeConditional:
        return b.evaluateCondition(vars)
    }
    return false
}

func (b *Breakpoint) evaluateCondition(vars types.Variables) bool {
    // Use existing JQ infrastructure from pkg/vars
    result, err := vars.ResolveQuery(b.Condition)
    if err != nil {
        return false
    }
    // Truthy check
    switch v := result.(type) {
    case bool:
        return v
    case nil:
        return false
    default:
        return true
    }
}
```

### Debug Hook Interface

```go
// pkg/debugger/hook.go

// DebugHook is the interface implemented by debug sessions
// and called by the task scheduler during execution
type DebugHook interface {
    // OnBeforeTask is called before each task executes
    // Returns error to abort, ErrSkipTask to skip, nil to continue
    OnBeforeTask(ctx context.Context, taskInfo TaskHookInfo) error

    // OnAfterTask is called after each task completes
    OnAfterTask(ctx context.Context, taskInfo TaskHookInfo, result types.TaskResult, err error)

    // OnTaskCreated is called when a new task is created (for tracking)
    OnTaskCreated(taskInfo TaskHookInfo)

    // OnConcurrentStart is called when a concurrent/background execution begins
    // Returns a frameID that must be passed to subsequent hook calls
    OnConcurrentStart(parentTaskIndex types.TaskIndex, frameName string) uint64

    // OnConcurrentEnd is called when a concurrent/background execution completes
    OnConcurrentEnd(frameID uint64)

    // GetInjectedTasks returns tasks to execute before the given task
    GetInjectedTasks(frameID uint64, beforeTaskIndex types.TaskIndex) []*InjectedTask

    // ShouldPauseTimeout returns true if task timeout should pause during debug pause
    ShouldPauseTimeout(frameID uint64) bool
}

type TaskHookInfo struct {
    TaskIndex     types.TaskIndex
    TaskID        string
    TaskName      string
    TaskTitle     string
    ParentIndex   types.TaskIndex
    Depth         int
    Variables     types.Variables
    TaskOptions   *types.TaskOptions
    FrameID       uint64              // Which execution frame this task belongs to
    IsCleanupTask bool                // True if this is a cleanup task
}

var ErrSkipTask = errors.New("skip task")
var ErrAbortDebug = errors.New("debug session aborted")
```

### Interface Extensions

The `TaskSchedulerRunner` interface needs extensions for debug support:

```go
// pkg/types/scheduler.go - Add to TaskSchedulerRunner interface

type TaskSchedulerRunner interface {
    // ... existing methods ...

    // Debug support
    GetDebugHook() debugger.DebugHook
    SetDebugHook(hook debugger.DebugHook)
    GetTaskFrame(taskIndex TaskIndex) uint64
    SetTaskFrame(taskIndex TaskIndex, frameID uint64)
}
```

This allows glue tasks to access the debug hook via the interface without type assertions.

### Frame Management

Execution frames are created and managed based on task execution patterns:

```
Root Frame (ID: 1)
├── Task: run_tasks_concurrent
│   ├── Frame 2: "concurrent[0]"
│   │   └── Task: check_finality
│   ├── Frame 3: "concurrent[1]"
│   │   └── Task: generate_deposits
│   └── Frame 4: "concurrent[2]"
│       └── Task: check_balance
├── Task: run_task_background
│   ├── Frame 5: "background"
│   │   └── Task: monitor_health
│   └── Frame 6: "foreground" (inherits from root or new frame)
│       └── Task: main_workflow
```

**Frame Creation Rules:**
1. **Root frame**: Created when debug session starts (frame ID 1)
2. **Concurrent frames**: Created by `run_tasks_concurrent` for each child task
3. **Background frames**: Created by `run_task_background` for background and foreground tasks
4. **Sequential tasks**: Inherit parent's frame (no new frame created)

**Frame Lifecycle:**
```
Created → Running → Paused ⟷ Running → Completed
                      ↓
                   Stepping → Running
```

### Frame Propagation in NewTask

Child tasks created via `TaskContext.NewTask()` must inherit the parent's frame:

```go
// pkg/scheduler/task_state.go - Modify newTaskState to accept frame context

func (ts *TaskScheduler) newTaskState(
    parentState *taskState,
    options *types.TaskOptions,
    variables types.Variables,
) (types.TaskIndex, error) {
    // ... existing logic ...

    // Inherit frame from parent
    var frameID uint64
    if parentState != nil {
        frameID = ts.GetTaskFrame(parentState.taskIndex)
    } else {
        frameID = 1 // Root frame
    }
    ts.SetTaskFrame(newTaskIndex, frameID)

    return newTaskIndex, nil
}
```

### Variable Modification Timing

**Important:** Variable changes apply to **future tasks only**, not the currently paused task.

```
Timeline:
1. Task A completes → outputs available
2. Task B pauses before execution (config NOT yet resolved)
3. User modifies variable X
4. User continues
5. Task B's config resolves with modified X ← Changes take effect here
6. Task B executes
```

**Why:** Task config is resolved in `LoadConfig()` which happens *after* the `OnBeforeTask` hook returns. Pending variable changes are applied to the scope before `LoadConfig()` runs.

```go
// In ExecuteTask, after OnBeforeTask returns:
if ts.debugHook != nil {
    // Apply pending changes BEFORE config resolution
    ts.debugHook.ApplyPendingChanges(hookInfo.FrameID, taskState.taskVars)
}

// Then proceed to LoadConfig
err := task.LoadConfig()
```

### Timeout Handling During Pause

Task timeout must pause while debug is paused to prevent spurious timeouts:

```go
// pkg/scheduler/task_execution.go - Modified timeout handling

func (ts *TaskScheduler) ExecuteTask(...) error {
    // ... setup ...

    if timeout > 0 {
        go func() {
            timer := time.NewTimer(timeout)
            defer timer.Stop()

            for {
                select {
                case <-timer.C:
                    // Check if we should pause the timer
                    if ts.debugHook != nil && ts.debugHook.ShouldPauseTimeout(frameID) {
                        // Debug is paused, wait for resume then restart timer
                        <-ts.debugHook.WaitForResume(frameID)
                        timer.Reset(timeout) // Reset full timeout on resume
                        continue
                    }
                    taskState.isTimeout = true
                    taskCancelFn()
                    return
                case <-taskContext.Done():
                    return
                }
            }
        }()
    }
}
```

**Alternative (simpler):** Track elapsed time before pause, subtract from remaining timeout on resume.

### Abort Handling

Abort must cleanly unblock all paused frames:

```go
func (ds *DebugSession) Abort() error {
    ds.mu.Lock()
    defer ds.mu.Unlock()

    // Mark session as aborting
    ds.State = DebugStateCompleted

    // Unblock all paused frames
    for _, frame := range ds.Frames {
        if frame.State == FrameStatePaused && frame.resumeChan != nil {
            // Close channel to unblock waiters
            close(frame.resumeChan)
            frame.resumeChan = nil
        }
        frame.State = FrameStateCompleted
    }

    // Emit abort event
    ds.emitAbortEvent()

    return nil
}

// In OnBeforeTask, check for abort after resume:
select {
case <-resumeChan:
    ds.mu.Lock()
    if ds.State == DebugStateCompleted {
        ds.mu.Unlock()
        return ErrAbortDebug  // Propagate abort
    }
    // ... normal resume logic ...
}
```

### Step-Out Across Frame Boundaries

Step-out needs to track the frame hierarchy, not just depth:

```go
type ExecutionFrame struct {
    // ... existing fields ...

    // For step-out tracking
    StepOutTargetFrame uint64  // Frame to stop in (0 = stop in any ancestor)
}

func (ds *DebugSession) StepOutFrame(frameID uint64) error {
    ds.mu.Lock()
    defer ds.mu.Unlock()

    frame, ok := ds.Frames[frameID]
    if !ok || frame.State != FrameStatePaused {
        return ErrFrameNotPaused
    }

    frame.StepMode = StepModeOut
    frame.StepFromDepth = frame.CurrentTask.Depth
    frame.StepOutTargetFrame = frame.ParentFrameID  // Stop in parent frame

    return ds.resumeFrame(frame)
}

func (ds *DebugSession) shouldPauseAt(taskInfo TaskHookInfo, frame *ExecutionFrame) bool {
    // ... breakpoint checks ...

    switch frame.StepMode {
    case StepModeOut:
        // Stop if:
        // 1. We're in the target frame (or any ancestor if target is 0)
        // 2. AND we've decreased depth (exited a level)
        inTargetFrame := frame.StepOutTargetFrame == 0 ||
                         taskInfo.FrameID == frame.StepOutTargetFrame ||
                         ds.isAncestorFrame(taskInfo.FrameID, frame.StepOutTargetFrame)

        if inTargetFrame && taskInfo.Depth < frame.StepFromDepth {
            frame.StepMode = StepModeNone
            return true
        }

        // Also stop if the current frame completed and we're now in parent
        if taskInfo.FrameID != frame.ID {
            frame.StepMode = StepModeNone
            return true
        }
    }

    return false
}
```

### Concurrent Session Prevention

Prevent multiple debug sessions for the same run:

```go
func (dm *DebugManager) CreateSession(runID uint64, pauseOnStart bool) (*DebugSession, error) {
    dm.mu.Lock()
    defer dm.mu.Unlock()

    // Check for existing session
    if existing, ok := dm.sessions[runID]; ok {
        if existing.State != DebugStateCompleted {
            return nil, ErrSessionAlreadyExists
        }
        // Clean up completed session
        delete(dm.sessions, runID)
    }

    // Create new session
    session := &DebugSession{
        ID:        uuid.New().String(),
        RunID:     runID,
        // ...
    }
    dm.sessions[runID] = session
    return session, nil
}
```

---

## Implementation Plan

### Phase 7.1: Debug Session Core

**Goal:** Implement the core debug session management and pause/resume mechanism.

#### 7.1.1 Debug Session Manager

```go
// pkg/debugger/manager.go

type DebugManager struct {
    sessions    map[uint64]*DebugSession  // runID -> session
    mu          sync.RWMutex
    eventBus    *events.EventBus
}

func NewDebugManager(eventBus *events.EventBus) *DebugManager

// CreateSession creates a new debug session for a test run
func (dm *DebugManager) CreateSession(runID uint64, pauseOnStart bool) (*DebugSession, error)

// GetSession returns the debug session for a test run
func (dm *DebugManager) GetSession(runID uint64) *DebugSession

// DeleteSession removes a debug session
func (dm *DebugManager) DeleteSession(runID uint64)

// CleanupCompletedSessions removes sessions for completed test runs
func (dm *DebugManager) CleanupCompletedSessions()
```

#### 7.1.2 Debug Session Implementation

```go
// pkg/debugger/session.go

// OnConcurrentStart implements DebugHook - creates a new execution frame
func (ds *DebugSession) OnConcurrentStart(parentTaskIndex uint64, frameName string) uint64 {
    ds.mu.Lock()
    defer ds.mu.Unlock()

    // Find parent frame
    var parentFrameID uint64
    for _, frame := range ds.Frames {
        for _, idx := range frame.TaskIndices {
            if idx == parentTaskIndex {
                parentFrameID = frame.ID
                break
            }
        }
    }

    // Create new frame
    ds.nextFrameID++
    frame := &ExecutionFrame{
        ID:            ds.nextFrameID,
        ParentFrameID: parentFrameID,
        State:         FrameStateRunning,
        Name:          frameName,
        RootTaskIndex: parentTaskIndex,
        TaskIndices:   []uint64{},
        resumeChan:    make(chan struct{}),
        PendingVarChanges: make(map[string]any),
    }
    ds.Frames[frame.ID] = frame

    ds.emitFrameCreatedEvent(frame)
    return frame.ID
}

// OnConcurrentEnd implements DebugHook - marks frame as completed
func (ds *DebugSession) OnConcurrentEnd(frameID uint64) {
    ds.mu.Lock()
    defer ds.mu.Unlock()

    if frame, ok := ds.Frames[frameID]; ok {
        frame.State = FrameStateCompleted
        ds.emitFrameCompletedEvent(frame)
    }
}

// OnBeforeTask implements DebugHook
func (ds *DebugSession) OnBeforeTask(ctx context.Context, taskInfo TaskHookInfo) error {
    ds.mu.Lock()

    // Get or create frame for this task
    frame := ds.getOrCreateFrame(taskInfo)
    frame.TaskIndices = append(frame.TaskIndices, taskInfo.TaskIndex)

    // Check if we should skip this task (frame-specific)
    if frame.SkipNextTask {
        frame.SkipNextTask = false
        ds.mu.Unlock()
        return ErrSkipTask
    }

    // Check if we should pause
    shouldPause := ds.shouldPauseAt(taskInfo, frame)

    if !shouldPause {
        ds.mu.Unlock()
        return nil
    }

    // Capture task info for inspection
    frame.CurrentTask = ds.captureTaskInfo(taskInfo)
    frame.PauseReason = ds.determinePauseReason(taskInfo, frame)
    frame.State = FrameStatePaused
    ds.LastActivity = time.Now()

    // Optionally pause all frames when one hits breakpoint
    if ds.PauseAllOnBreakpoint && frame.PauseReason == PauseReasonBreakpoint {
        ds.pauseAllFrames()
    }

    // Create new resume channel for this frame
    frame.resumeChan = make(chan struct{})
    resumeChan := frame.resumeChan

    ds.mu.Unlock()

    // Emit paused event with frame info
    ds.emitFramePausedEvent(frame)

    // Wait for resume signal for THIS frame
    select {
    case <-resumeChan:
        ds.mu.Lock()
        // Apply pending variable changes
        ds.applyPendingChanges(frame, taskInfo.Variables)
        frame.State = FrameStateRunning
        frame.CurrentTask = nil
        ds.mu.Unlock()
        ds.emitFrameResumedEvent(frame)
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (ds *DebugSession) shouldPauseAt(taskInfo TaskHookInfo, frame *ExecutionFrame) bool {
    // Check for global pause request
    if ds.GlobalPauseRequested {
        return true
    }

    // Check for initial pause (first task in root frame)
    if frame.ID == 1 && len(frame.TaskIndices) == 0 && ds.State == DebugStateRunning {
        return true
    }

    // Check breakpoints (global, applies to all frames)
    for _, bp := range ds.Breakpoints {
        if bp.Match(taskInfo) {
            bp.HitCount++
            return true
        }
    }

    // Check frame-specific step mode
    switch frame.StepMode {
    case StepModeInto:
        frame.StepMode = StepModeNone
        return true

    case StepModeOver:
        // Stop if we're at same or lesser depth (sibling or uncle)
        if taskInfo.Depth <= frame.StepFromDepth {
            frame.StepMode = StepModeNone
            return true
        }

    case StepModeOut:
        // Stop if we've exited to a lesser depth
        if taskInfo.Depth < frame.StepFromDepth {
            frame.StepMode = StepModeNone
            return true
        }
    }

    return false
}

func (ds *DebugSession) getOrCreateFrame(taskInfo TaskHookInfo) *ExecutionFrame {
    // If frameID is specified, use it
    if taskInfo.FrameID != 0 {
        if frame, ok := ds.Frames[taskInfo.FrameID]; ok {
            return frame
        }
    }

    // Default to root frame
    if frame, ok := ds.Frames[1]; ok {
        return frame
    }

    // Create root frame if needed
    frame := &ExecutionFrame{
        ID:            1,
        State:         FrameStateRunning,
        Name:          "main",
        resumeChan:    make(chan struct{}),
        PendingVarChanges: make(map[string]any),
    }
    ds.Frames[1] = frame
    ds.nextFrameID = 1
    return frame
}

// Frame-specific control methods
func (ds *DebugSession) ContinueFrame(frameID uint64) error
func (ds *DebugSession) PauseFrame(frameID uint64) error
func (ds *DebugSession) StepIntoFrame(frameID uint64) error
func (ds *DebugSession) StepOverFrame(frameID uint64) error
func (ds *DebugSession) StepOutFrame(frameID uint64) error
func (ds *DebugSession) SkipTaskInFrame(frameID uint64) error

// Global control methods (affect all frames)
func (ds *DebugSession) ContinueAll() error
func (ds *DebugSession) PauseAll() error
func (ds *DebugSession) Abort() error

// Breakpoint methods (global)
func (ds *DebugSession) AddBreakpoint(bp *Breakpoint) error
func (ds *DebugSession) RemoveBreakpoint(id string) error
func (ds *DebugSession) EnableBreakpoint(id string, enabled bool) error
func (ds *DebugSession) GetBreakpoints() []*Breakpoint

// Frame inspection
func (ds *DebugSession) GetFrames() []*ExecutionFrame
func (ds *DebugSession) GetFrame(frameID uint64) *ExecutionFrame
func (ds *DebugSession) SetActiveFrame(frameID uint64) error

// Variable methods (frame-specific)
func (ds *DebugSession) GetVariables(frameID uint64) map[string]any
func (ds *DebugSession) SetVariable(frameID uint64, name string, value any) error

// Task injection (frame-specific)
func (ds *DebugSession) InjectTask(frameID uint64, taskDef *types.TaskOptions, mode InsertMode) (string, error)
func (ds *DebugSession) RemoveInjectedTask(id string) error
func (ds *DebugSession) GetInjectedTasks(frameID uint64) []*InjectedTask
```

#### 7.1.3 Scheduler Integration

Modify task scheduler to support debug hooks:

```go
// pkg/scheduler/scheduler.go

type TaskScheduler struct {
    // ... existing fields ...

    debugHook    DebugHook            // Optional debug hook
    taskFrameMap map[uint64]uint64    // taskIndex -> frameID mapping
}

func (ts *TaskScheduler) SetDebugHook(hook DebugHook) {
    ts.debugHook = hook
    ts.taskFrameMap = make(map[uint64]uint64)
}

// pkg/scheduler/task_execution.go

func (ts *TaskScheduler) ExecuteTask(ctx context.Context, taskIndex types.TaskIndex, taskWatchFn ...) error {
    taskState := ts.taskStateIndex[taskIndex]

    // === DEBUG HOOK: Before Task ===
    if ts.debugHook != nil {
        hookInfo := ts.buildTaskHookInfo(taskState)

        // Check for injected tasks to execute first
        for _, injected := range ts.debugHook.GetInjectedTasks(hookInfo.FrameID, uint64(taskIndex)) {
            if !injected.Executed {
                injectedTask, err := ts.newTaskState(nil, injected.TaskOptions, taskState.taskVars)
                if err != nil {
                    return fmt.Errorf("failed to create injected task: %w", err)
                }
                // Inherit frame from parent
                ts.taskFrameMap[uint64(injectedTask)] = hookInfo.FrameID
                injected.Executed = true
                if err := ts.ExecuteTask(ctx, injectedTask, nil); err != nil {
                    return err
                }
            }
        }

        // Call the before-task hook (may block if paused)
        err := ts.debugHook.OnBeforeTask(ctx, hookInfo)
        if err == ErrSkipTask {
            taskState.setSkipped()
            ts.emitTaskCompleted(taskState)
            return nil
        }
        if err != nil {
            return err
        }
    }
    // === END DEBUG HOOK ===

    // ... existing execution code ...

    // === DEBUG HOOK: After Task ===
    if ts.debugHook != nil {
        hookInfo := ts.buildTaskHookInfo(taskState)
        ts.debugHook.OnAfterTask(ctx, hookInfo, taskState.taskResult, taskState.taskError)
    }
    // === END DEBUG HOOK ===

    return nil
}

func (ts *TaskScheduler) buildTaskHookInfo(taskState *taskState) TaskHookInfo {
    // Get frame ID for this task (inherited from parent or explicitly set)
    frameID := ts.getFrameIDForTask(taskState)

    return TaskHookInfo{
        TaskIndex:   uint64(taskState.taskIndex),
        TaskID:      taskState.taskOptions.ID,
        TaskName:    taskState.taskOptions.Name,
        TaskTitle:   taskState.taskOptions.Title,
        ParentIndex: uint64(taskState.parentIndex),
        Depth:       ts.calculateTaskDepth(taskState),
        Variables:   taskState.taskVars,
        TaskOptions: taskState.taskOptions,
        FrameID:     frameID,
    }
}

func (ts *TaskScheduler) getFrameIDForTask(taskState *taskState) uint64 {
    if ts.taskFrameMap == nil {
        return 1 // Root frame
    }
    if frameID, ok := ts.taskFrameMap[uint64(taskState.taskIndex)]; ok {
        return frameID
    }
    // Inherit from parent
    if taskState.parentIndex != 0 {
        return ts.getFrameIDForTask(ts.taskStateIndex[taskState.parentIndex])
    }
    return 1 // Root frame
}

func (ts *TaskScheduler) calculateTaskDepth(taskState *taskState) int {
    depth := 0
    current := taskState
    for current.parentIndex != 0 {
        depth++
        parent, exists := ts.taskStateIndex[current.parentIndex]
        if !exists {
            break
        }
        current = parent
    }
    return depth
}

// SetTaskFrame explicitly sets the frame for a task (called by glue tasks)
func (ts *TaskScheduler) SetTaskFrame(taskIndex uint64, frameID uint64) {
    if ts.taskFrameMap != nil {
        ts.taskFrameMap[taskIndex] = frameID
    }
}
```

#### 7.1.4 Glue Task Integration for Frame Creation

Glue tasks must create execution frames when spawning concurrent/background tasks:

```go
// pkg/tasks/run_tasks_concurrent/task.go

func (t *Task) Execute(ctx context.Context) error {
    // ... existing setup ...

    for i, childTask := range t.tasks {
        go func(idx int, task types.TaskIndex) {
            defer wg.Done()

            // === DEBUG: Create frame for this concurrent branch ===
            var frameID uint64
            if debugHook := t.ctx.Scheduler.GetDebugHook(); debugHook != nil {
                frameID = debugHook.OnConcurrentStart(
                    uint64(t.ctx.Index),
                    fmt.Sprintf("concurrent[%d]", idx),
                )
                t.ctx.Scheduler.SetTaskFrame(uint64(task), frameID)
                defer debugHook.OnConcurrentEnd(frameID)
            }
            // === END DEBUG ===

            err := t.ctx.Scheduler.ExecuteTask(ctx, task, nil)
            // ... handle result ...
        }(i, childTask)
    }

    wg.Wait()
    // ... existing result handling ...
}

// pkg/tasks/run_task_background/task.go

func (t *Task) Execute(ctx context.Context) error {
    // ... existing setup ...

    // === DEBUG: Create frames for background and foreground ===
    var bgFrameID, fgFrameID uint64
    if debugHook := t.ctx.Scheduler.GetDebugHook(); debugHook != nil {
        bgFrameID = debugHook.OnConcurrentStart(
            uint64(t.ctx.Index),
            "background",
        )
        t.ctx.Scheduler.SetTaskFrame(uint64(t.backgroundTask), bgFrameID)

        fgFrameID = debugHook.OnConcurrentStart(
            uint64(t.ctx.Index),
            "foreground",
        )
        t.ctx.Scheduler.SetTaskFrame(uint64(t.foregroundTask), fgFrameID)
    }
    // === END DEBUG ===

    // Start background task
    go func() {
        defer func() {
            if debugHook := t.ctx.Scheduler.GetDebugHook(); debugHook != nil {
                debugHook.OnConcurrentEnd(bgFrameID)
            }
        }()
        t.ctx.Scheduler.ExecuteTask(bgCtx, t.backgroundTask, nil)
        // ...
    }()

    // Run foreground task
    err := t.ctx.Scheduler.ExecuteTask(ctx, t.foregroundTask, nil)
    if debugHook := t.ctx.Scheduler.GetDebugHook(); debugHook != nil {
        debugHook.OnConcurrentEnd(fgFrameID)
    }

    // ... existing completion handling ...
}
```

#### 7.1.4 Test Runner Integration

```go
// pkg/assertoor/testrunner.go

type TestRunner struct {
    // ... existing fields ...

    debugManager *debugger.DebugManager
}

// ScheduleTestDebug schedules a test to run in debug mode
func (tr *TestRunner) ScheduleTestDebug(testID string, config map[string]any, pauseOnStart bool) (uint64, *debugger.DebugSession, error) {
    // Create test run
    runID, err := tr.ScheduleTest(testID, config, false)
    if err != nil {
        return 0, nil, err
    }

    // Create debug session
    session, err := tr.debugManager.CreateSession(runID, pauseOnStart)
    if err != nil {
        tr.CancelTestRun(runID)
        return 0, nil, err
    }

    return runID, session, nil
}

// In test execution, attach debug hook if session exists
func (tr *TestRunner) executeTest(test types.Test) {
    session := tr.debugManager.GetSession(test.RunID())
    if session != nil {
        test.Scheduler().SetDebugHook(session)
    }

    // ... existing execution ...
}
```

**Files:**
```
pkg/debugger/
├── manager.go      # DebugManager
├── session.go      # DebugSession
├── breakpoint.go   # Breakpoint types and matching
├── hook.go         # DebugHook interface
└── errors.go       # Error types
```

---

### Phase 7.2: Debug Events

**Goal:** Add debug-specific events for real-time UI updates.

#### 7.2.1 Event Types

```go
// pkg/events/event.go

const (
    // ... existing events ...

    EventDebugSessionCreated  EventType = "debug.session.created"
    EventDebugSessionDeleted  EventType = "debug.session.deleted"
    EventDebugFrameCreated    EventType = "debug.frame.created"
    EventDebugFrameCompleted  EventType = "debug.frame.completed"
    EventDebugFramePaused     EventType = "debug.frame.paused"
    EventDebugFrameResumed    EventType = "debug.frame.resumed"
    EventDebugBreakpointHit   EventType = "debug.breakpoint.hit"
    EventDebugTaskSkipped     EventType = "debug.task.skipped"
    EventDebugVariableChanged EventType = "debug.variable.changed"
    EventDebugTaskInjected    EventType = "debug.task.injected"
)

type DebugFrameCreatedEventData struct {
    SessionID     string `json:"sessionId"`
    RunID         uint64 `json:"runId"`
    FrameID       uint64 `json:"frameId"`
    FrameName     string `json:"frameName"`
    ParentFrameID uint64 `json:"parentFrameId"`
    RootTaskIndex uint64 `json:"rootTaskIndex"`
}

type DebugFrameCompletedEventData struct {
    SessionID string `json:"sessionId"`
    RunID     uint64 `json:"runId"`
    FrameID   uint64 `json:"frameId"`
}

type DebugFramePausedEventData struct {
    SessionID       string                 `json:"sessionId"`
    RunID           uint64                 `json:"runId"`
    FrameID         uint64                 `json:"frameId"`
    FrameName       string                 `json:"frameName"`
    Reason          string                 `json:"reason"`
    TaskIndex       uint64                 `json:"taskIndex"`
    TaskID          string                 `json:"taskId"`
    TaskName        string                 `json:"taskName"`
    TaskTitle       string                 `json:"taskTitle"`
    Depth           int                    `json:"depth"`
    BreakpointID    string                 `json:"breakpointId,omitempty"`
    ResolvedConfig  map[string]any         `json:"resolvedConfig"`
    AvailableVars   map[string]any         `json:"availableVars"`
}

type DebugFrameResumedEventData struct {
    SessionID string `json:"sessionId"`
    RunID     uint64 `json:"runId"`
    FrameID   uint64 `json:"frameId"`
    Action    string `json:"action"` // "continue", "step_into", "step_over", "step_out"
}
```

#### 7.2.2 Event Emission

```go
// pkg/debugger/session.go

func (ds *DebugSession) emitPausedEvent() {
    if ds.eventBus == nil {
        return
    }

    ds.eventBus.Publish(&events.Event{
        Type:      events.EventDebugPaused,
        TestRunID: ds.RunID,
        Data: &DebugPausedEventData{
            SessionID:      ds.ID,
            RunID:          ds.RunID,
            Reason:         string(ds.PauseReason),
            TaskIndex:      ds.CurrentTask.TaskIndex,
            TaskID:         ds.CurrentTask.TaskID,
            TaskName:       ds.CurrentTask.TaskName,
            TaskTitle:      ds.CurrentTask.TaskTitle,
            Depth:          ds.CurrentTask.Depth,
            ResolvedConfig: ds.CurrentTask.ResolvedConfig,
            AvailableVars:  ds.CurrentTask.AvailableVars,
        },
    })
}

func (ds *DebugSession) emitResumedEvent(action string) {
    if ds.eventBus == nil {
        return
    }

    ds.eventBus.Publish(&events.Event{
        Type:      events.EventDebugResumed,
        TestRunID: ds.RunID,
        Data: &DebugResumedEventData{
            SessionID: ds.ID,
            RunID:     ds.RunID,
            Action:    action,
        },
    })
}
```

**Files:**
```
pkg/events/event.go           # Add debug event types
pkg/events/sse.go             # Add debug event publishing (if needed)
```

---

### Phase 7.3: Debug API Endpoints

**Goal:** Implement REST API endpoints for debug control.

#### 7.3.1 Session Management

```go
// pkg/web/api/debug_session_api.go

// POST /api/v1/test_runs/schedule/debug
// Start a new test in debug mode
type ScheduleDebugRequest struct {
    TestID               string         `json:"testId"`
    Config               map[string]any `json:"config"`
    PauseOnStart         bool           `json:"pauseOnStart"`
    PauseAllOnBreakpoint bool           `json:"pauseAllOnBreakpoint"` // Pause all frames when any hits breakpoint
    Breakpoints          []Breakpoint   `json:"breakpoints,omitempty"`
}

type ScheduleDebugResponse struct {
    RunID     uint64 `json:"runId"`
    SessionID string `json:"sessionId"`
}

// GET /api/v1/test_run/{runId}/debug
// Get debug session state (for reconnection)
type GetDebugSessionResponse struct {
    SessionID            string             `json:"sessionId"`
    State                string             `json:"state"`
    PauseAllOnBreakpoint bool               `json:"pauseAllOnBreakpoint"`
    Frames               []*FrameInfo       `json:"frames"`
    ActiveFrameID        uint64             `json:"activeFrameId"`
    Breakpoints          []*Breakpoint      `json:"breakpoints"`
    InjectedTasks        []*InjectedTask    `json:"injectedTasks"`
}

type FrameInfo struct {
    ID                uint64            `json:"id"`
    ParentFrameID     uint64            `json:"parentFrameId"`
    Name              string            `json:"name"`
    State             string            `json:"state"` // "running", "paused", "completed"
    RootTaskIndex     uint64            `json:"rootTaskIndex"`
    TaskIndices       []uint64          `json:"taskIndices"`
    PauseReason       string            `json:"pauseReason,omitempty"`
    CurrentTask       *TaskPauseInfo    `json:"currentTask,omitempty"`
    PendingVarChanges map[string]any    `json:"pendingVarChanges,omitempty"`
}

// DELETE /api/v1/test_run/{runId}/debug
// Detach debugger and continue without debugging
```

#### 7.3.2 Execution Control

```go
// pkg/web/api/debug_control_api.go

// POST /api/v1/test_run/{runId}/debug/continue
// Continue execution of a specific frame or all frames
type ContinueRequest struct {
    FrameID uint64 `json:"frameId,omitempty"` // 0 = continue all frames
}

// POST /api/v1/test_run/{runId}/debug/pause
// Pause execution of a specific frame or all frames
type PauseRequest struct {
    FrameID uint64 `json:"frameId,omitempty"` // 0 = pause all frames
}

// POST /api/v1/test_run/{runId}/debug/step
// Step in a specific frame
type StepRequest struct {
    FrameID uint64 `json:"frameId"` // Required - which frame to step
    Mode    string `json:"mode"`    // "into", "over", "out"
}

// POST /api/v1/test_run/{runId}/debug/skip
// Skip the current task in a specific frame
type SkipRequest struct {
    FrameID uint64 `json:"frameId"` // Required - which frame's task to skip
}

// POST /api/v1/test_run/{runId}/debug/abort
// Abort the test run (affects all frames)

// POST /api/v1/test_run/{runId}/debug/active-frame
// Set the active frame for UI display
type SetActiveFrameRequest struct {
    FrameID uint64 `json:"frameId"`
}

// GET /api/v1/test_run/{runId}/debug/frames
// Get list of all frames with their states
type GetFramesResponse struct {
    Frames []*FrameInfo `json:"frames"`
}
```

#### 7.3.3 Breakpoint Management

```go
// pkg/web/api/debug_breakpoint_api.go

// GET /api/v1/test_run/{runId}/debug/breakpoints
// List all breakpoints

// POST /api/v1/test_run/{runId}/debug/breakpoints
type AddBreakpointRequest struct {
    Type      string `json:"type"`      // "task_index", "task_id", "task_name", "conditional"
    TaskIndex uint64 `json:"taskIndex,omitempty"`
    TaskID    string `json:"taskId,omitempty"`
    TaskName  string `json:"taskName,omitempty"`
    Condition string `json:"condition,omitempty"`
    Enabled   bool   `json:"enabled"`
}

type AddBreakpointResponse struct {
    ID string `json:"id"`
}

// PUT /api/v1/test_run/{runId}/debug/breakpoints/{id}
type UpdateBreakpointRequest struct {
    Enabled *bool `json:"enabled,omitempty"`
}

// DELETE /api/v1/test_run/{runId}/debug/breakpoints/{id}
```

#### 7.3.4 Inspection & Modification

```go
// pkg/web/api/debug_inspect_api.go

// GET /api/v1/test_run/{runId}/debug/frame/{frameId}/variables
// Get all variables at current pause point for a specific frame
type GetVariablesResponse struct {
    FrameID        uint64         `json:"frameId"`
    Variables      map[string]any `json:"variables"`
    PendingChanges map[string]any `json:"pendingChanges"`
}

// POST /api/v1/test_run/{runId}/debug/frame/{frameId}/variables
// Set a variable value for a specific frame
type SetVariableRequest struct {
    Name  string `json:"name"`
    Value any    `json:"value"`
}

// DELETE /api/v1/test_run/{runId}/debug/frame/{frameId}/variables/{name}
// Remove a pending variable change for a specific frame

// GET /api/v1/test_run/{runId}/debug/frame/{frameId}/task-config
// Get resolved config for current task in a specific frame
type GetTaskConfigResponse struct {
    FrameID        uint64            `json:"frameId"`
    TaskIndex      uint64            `json:"taskIndex"`
    TaskID         string            `json:"taskId"`
    TaskName       string            `json:"taskName"`
    RawConfig      map[string]any    `json:"rawConfig"`
    ConfigVars     map[string]string `json:"configVars"`
    ResolvedConfig map[string]any    `json:"resolvedConfig"`
}
```

#### 7.3.5 Task Injection

```go
// pkg/web/api/debug_inject_api.go

// POST /api/v1/test_run/{runId}/debug/frame/{frameId}/inject-task
// Inject a task into a specific frame's execution
type InjectTaskRequest struct {
    Task       types.TaskOptions `json:"task"`
    InsertMode string            `json:"insertMode"` // "before_current", "after_current"
}

type InjectTaskResponse struct {
    ID string `json:"id"`
}

// GET /api/v1/test_run/{runId}/debug/frame/{frameId}/injected-tasks
// List pending injected tasks for a frame

// DELETE /api/v1/test_run/{runId}/debug/injected-tasks/{id}
// Remove an injected task before it executes
```

#### 7.3.6 Expression Evaluation (REPL)

```go
// pkg/web/api/debug_eval_api.go

// POST /api/v1/test_run/{runId}/debug/frame/{frameId}/evaluate
// Evaluate a JQ expression against the frame's variable scope
type EvaluateRequest struct {
    Expression string `json:"expression"`
}

type EvaluateResponse struct {
    Result any    `json:"result"`
    Error  string `json:"error,omitempty"`
    Type   string `json:"type"` // "string", "number", "bool", "object", "array", "null"
}

// GET /api/v1/test_run/{runId}/debug/frame/{frameId}/history
// Get execution history for a frame (tasks executed, variable snapshots)
type FrameHistoryResponse struct {
    FrameID      uint64              `json:"frameId"`
    TaskHistory  []TaskHistoryEntry  `json:"taskHistory"`
}

type TaskHistoryEntry struct {
    TaskIndex    types.TaskIndex `json:"taskIndex"`
    TaskName     string          `json:"taskName"`
    Result       string          `json:"result"` // "success", "failure", "skipped"
    StartTime    time.Time       `json:"startTime"`
    EndTime      time.Time       `json:"endTime"`
    WasPaused    bool            `json:"wasPaused"`
}

// GET /api/v1/test_run/{runId}/debug/export
// Export full debug session state for sharing/debugging
type ExportResponse struct {
    Session      DebugSessionExport `json:"session"`
    Frames       []FrameExport      `json:"frames"`
    Breakpoints  []*Breakpoint      `json:"breakpoints"`
    TaskHistory  []TaskHistoryEntry `json:"taskHistory"`
    ExportedAt   time.Time          `json:"exportedAt"`
}
```

**Files:**
```
pkg/web/api/
├── debug_session_api.go     # Session endpoints
├── debug_control_api.go     # Control endpoints
├── debug_breakpoint_api.go  # Breakpoint CRUD
├── debug_inspect_api.go     # Variable/config inspection (frame-specific)
└── debug_inject_api.go      # Task injection (frame-specific)
```

**Routes (pkg/web/server.go):**
```go
// Debug endpoints (all protected)
// Session management
apiRouter.HandleFunc("/test_runs/schedule/debug", s.ScheduleDebugHandler).Methods("POST")
apiRouter.HandleFunc("/test_run/{runId}/debug", s.GetDebugSessionHandler).Methods("GET")
apiRouter.HandleFunc("/test_run/{runId}/debug", s.DeleteDebugSessionHandler).Methods("DELETE")

// Frame management
apiRouter.HandleFunc("/test_run/{runId}/debug/frames", s.GetFramesHandler).Methods("GET")
apiRouter.HandleFunc("/test_run/{runId}/debug/active-frame", s.SetActiveFrameHandler).Methods("POST")

// Global control (all frames)
apiRouter.HandleFunc("/test_run/{runId}/debug/continue", s.DebugContinueHandler).Methods("POST")
apiRouter.HandleFunc("/test_run/{runId}/debug/pause", s.DebugPauseHandler).Methods("POST")
apiRouter.HandleFunc("/test_run/{runId}/debug/abort", s.DebugAbortHandler).Methods("POST")

// Frame-specific control
apiRouter.HandleFunc("/test_run/{runId}/debug/step", s.DebugStepHandler).Methods("POST")
apiRouter.HandleFunc("/test_run/{runId}/debug/skip", s.DebugSkipHandler).Methods("POST")

// Breakpoints (global)
apiRouter.HandleFunc("/test_run/{runId}/debug/breakpoints", s.GetBreakpointsHandler).Methods("GET")
apiRouter.HandleFunc("/test_run/{runId}/debug/breakpoints", s.AddBreakpointHandler).Methods("POST")
apiRouter.HandleFunc("/test_run/{runId}/debug/breakpoints/{id}", s.UpdateBreakpointHandler).Methods("PUT")
apiRouter.HandleFunc("/test_run/{runId}/debug/breakpoints/{id}", s.DeleteBreakpointHandler).Methods("DELETE")

// Frame-specific inspection/modification
apiRouter.HandleFunc("/test_run/{runId}/debug/frame/{frameId}/variables", s.GetFrameVariablesHandler).Methods("GET")
apiRouter.HandleFunc("/test_run/{runId}/debug/frame/{frameId}/variables", s.SetFrameVariableHandler).Methods("POST")
apiRouter.HandleFunc("/test_run/{runId}/debug/frame/{frameId}/variables/{name}", s.DeleteFrameVariableChangeHandler).Methods("DELETE")
apiRouter.HandleFunc("/test_run/{runId}/debug/frame/{frameId}/task-config", s.GetFrameTaskConfigHandler).Methods("GET")
apiRouter.HandleFunc("/test_run/{runId}/debug/frame/{frameId}/inject-task", s.InjectTaskHandler).Methods("POST")
apiRouter.HandleFunc("/test_run/{runId}/debug/frame/{frameId}/injected-tasks", s.GetFrameInjectedTasksHandler).Methods("GET")
apiRouter.HandleFunc("/test_run/{runId}/debug/injected-tasks/{id}", s.DeleteInjectedTaskHandler).Methods("DELETE")

// Expression evaluation and history
apiRouter.HandleFunc("/test_run/{runId}/debug/frame/{frameId}/evaluate", s.EvaluateExpressionHandler).Methods("POST")
apiRouter.HandleFunc("/test_run/{runId}/debug/frame/{frameId}/history", s.GetFrameHistoryHandler).Methods("GET")
apiRouter.HandleFunc("/test_run/{runId}/debug/export", s.ExportDebugSessionHandler).Methods("GET")
```

---

### Phase 7.4: React UI - Debug Components

**Goal:** Implement the debug UI components.

#### 7.4.1 Debug Store

```typescript
// web-ui/src/stores/debugStore.ts

interface DebugState {
    // Session state
    sessionId: string | null;
    runId: number | null;
    pauseAllOnBreakpoint: boolean;

    // Execution frames
    frames: Map<number, ExecutionFrame>;
    activeFrameId: number;

    // Breakpoints (global)
    breakpoints: Breakpoint[];

    // Actions - Session
    setSession: (sessionId: string, runId: number, pauseAllOnBreakpoint: boolean) => void;
    clearSession: () => void;
    setPauseAllOnBreakpoint: (value: boolean) => void;

    // Actions - Frames
    setFrames: (frames: ExecutionFrame[]) => void;
    addFrame: (frame: ExecutionFrame) => void;
    updateFrame: (frameId: number, updates: Partial<ExecutionFrame>) => void;
    removeFrame: (frameId: number) => void;
    setActiveFrame: (frameId: number) => void;

    // Actions - Breakpoints
    setBreakpoints: (breakpoints: Breakpoint[]) => void;
    addBreakpoint: (breakpoint: Breakpoint) => void;
    removeBreakpoint: (id: string) => void;
    toggleBreakpoint: (id: string) => void;

    // Actions - Frame-specific modifications
    setPendingVariable: (frameId: number, name: string, value: any) => void;
    removePendingVariable: (frameId: number, name: string) => void;
    addInjectedTask: (frameId: number, task: InjectedTask) => void;
    removeInjectedTask: (id: string) => void;

    // Computed helpers
    getActiveFrame: () => ExecutionFrame | undefined;
    getPausedFrames: () => ExecutionFrame[];
    getRunningFrames: () => ExecutionFrame[];
    hasAnyPausedFrame: () => boolean;
}

interface ExecutionFrame {
    id: number;
    parentFrameId: number;
    name: string;
    state: 'running' | 'paused' | 'completed';
    rootTaskIndex: number;
    taskIndices: number[];
    pauseReason: string | null;
    currentTask: TaskPauseInfo | null;
    pendingVariables: Record<string, any>;
    injectedTasks: InjectedTask[];
}

interface TaskPauseInfo {
    taskIndex: number;
    taskId: string;
    taskName: string;
    taskTitle: string;
    depth: number;
    resolvedConfig: Record<string, any>;
    configVars: Record<string, string>;
    availableVars: Record<string, any>;
}

interface Breakpoint {
    id: string;
    type: 'task_index' | 'task_id' | 'task_name' | 'conditional';
    enabled: boolean;
    hitCount: number;
    taskIndex?: number;
    taskId?: string;
    taskName?: string;
    condition?: string;
}

interface InjectedTask {
    id: string;
    frameId: number;
    task: TaskOptions;
    insertMode: 'before_current' | 'after_current';
    executed: boolean;
}

// Error and loading states
interface DebugUIState {
    isLoading: boolean;
    isReconnecting: boolean;
    error: string | null;
    lastError: { message: string; timestamp: number } | null;
}

// Watch expressions for REPL-like functionality
interface WatchExpression {
    id: string;
    expression: string;
    result: any;
    error: string | null;
    isPinned: boolean;
}

// Variable history for diff view
interface VariableSnapshot {
    frameId: number;
    taskIndex: number;
    timestamp: number;
    variables: Record<string, any>;
}
```

### Debug Store - Full Implementation

```typescript
// web-ui/src/stores/debugStore.ts

import { create } from 'zustand';

interface DebugStore extends DebugState, DebugUIState {
    // ... state fields from above ...

    // UI state
    isLoading: boolean;
    isReconnecting: boolean;
    error: string | null;
    lastError: { message: string; timestamp: number } | null;

    // Watch expressions
    watchExpressions: WatchExpression[];

    // Variable history (for diff view)
    variableHistory: Map<number, VariableSnapshot[]>;

    // UI preferences
    expandedPaths: Set<string>;
    pinnedVariables: Set<string>;

    // Actions - Error handling
    setLoading: (loading: boolean) => void;
    setError: (error: string | null) => void;
    clearError: () => void;

    // Actions - Watch expressions
    addWatchExpression: (expression: string) => void;
    removeWatchExpression: (id: string) => void;
    updateWatchResult: (id: string, result: any, error: string | null) => void;
    toggleWatchPin: (id: string) => void;

    // Actions - Variable history
    captureVariableSnapshot: (frameId: number, taskIndex: number, variables: Record<string, any>) => void;
    getVariableDiff: (frameId: number) => { added: string[]; changed: string[]; removed: string[] } | null;

    // Actions - UI preferences
    toggleExpandedPath: (path: string) => void;
    togglePinnedVariable: (path: string) => void;
}

export const useDebugStore = create<DebugStore>((set, get) => ({
    // Initial state
    sessionId: null,
    runId: null,
    pauseAllOnBreakpoint: true,
    frames: new Map(),
    activeFrameId: 1,
    breakpoints: [],
    isLoading: false,
    isReconnecting: false,
    error: null,
    lastError: null,
    watchExpressions: [],
    variableHistory: new Map(),
    expandedPaths: new Set(),
    pinnedVariables: new Set(),

    // Error handling
    setLoading: (loading) => set({ isLoading: loading }),
    setError: (error) => set({
        error,
        lastError: error ? { message: error, timestamp: Date.now() } : get().lastError,
    }),
    clearError: () => set({ error: null }),

    // Session management with error handling
    setSession: (sessionId, runId, pauseAllOnBreakpoint) => set({
        sessionId,
        runId,
        pauseAllOnBreakpoint,
        error: null,
        isReconnecting: false,
    }),

    // ... other actions ...
}));
```

#### 7.4.2 Debug Hook for Events

```typescript
// web-ui/src/hooks/useDebugEvents.ts

export function useDebugEvents(runId: number) {
    const debugStore = useDebugStore();
    const reconnectTimeoutRef = useRef<number>();

    useEffect(() => {
        let eventSource: EventSource | null = null;
        let reconnectAttempts = 0;
        const maxReconnectAttempts = 10;
        const baseReconnectDelay = 1000;

        const connect = () => {
            const token = authStore.getToken();
            const url = `/api/v1/test_run/${runId}/events${token ? `?token=${token}` : ''}`;

            eventSource = new EventSource(url);

            eventSource.onopen = () => {
                reconnectAttempts = 0;
                debugStore.setError(null);
                // Sync full state on reconnect
                fetchDebugSession(runId).then(session => {
                    if (session) {
                        debugStore.setSession(session.sessionId, runId, session.pauseAllOnBreakpoint);
                        debugStore.setFrames(session.frames);
                        debugStore.setBreakpoints(session.breakpoints);
                    }
                });
            };

            eventSource.onerror = () => {
                eventSource?.close();
                eventSource = null;

                if (reconnectAttempts < maxReconnectAttempts) {
                    reconnectAttempts++;
                    debugStore.setError(`Connection lost. Reconnecting (${reconnectAttempts}/${maxReconnectAttempts})...`);
                    const delay = Math.min(baseReconnectDelay * Math.pow(2, reconnectAttempts - 1), 30000);
                    reconnectTimeoutRef.current = window.setTimeout(connect, delay);
                } else {
                    debugStore.setError('Connection lost. Please refresh the page.');
                }
            };

            eventSource.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    handleDebugEvent(data, debugStore);
                } catch (e) {
                    console.error('Failed to parse debug event:', e);
                }
            };
        };

        connect();

        return () => {
            if (reconnectTimeoutRef.current) {
                clearTimeout(reconnectTimeoutRef.current);
            }
            eventSource?.close();
        };
    }, [runId]);
}

function handleDebugEvent(event: DebugEvent, store: DebugStore) {
    switch (event.type) {
        case 'debug.frame.created':
            store.addFrame({
                id: event.data.frameId,
                parentFrameId: event.data.parentFrameId,
                name: event.data.frameName,
                state: 'running',
                rootTaskIndex: event.data.rootTaskIndex,
                taskIndices: [],
                pauseReason: null,
                currentTask: null,
                pendingVariables: {},
                injectedTasks: [],
            });
            break;

        case 'debug.frame.paused':
            store.updateFrame(event.data.frameId, {
                state: 'paused',
                pauseReason: event.data.reason,
                currentTask: {
                    taskIndex: event.data.taskIndex,
                    taskId: event.data.taskId,
                    taskName: event.data.taskName,
                    taskTitle: event.data.taskTitle,
                    depth: event.data.depth,
                    resolvedConfig: event.data.resolvedConfig,
                    configVars: event.data.configVars || {},
                    availableVars: event.data.availableVars,
                },
            });
            // Capture variable snapshot for diff view
            store.captureVariableSnapshot(
                event.data.frameId,
                event.data.taskIndex,
                event.data.availableVars
            );
            // Auto-switch to paused frame
            store.setActiveFrame(event.data.frameId);
            // Browser notification if tab not focused
            if (document.hidden && Notification.permission === 'granted') {
                new Notification('Debug: Frame Paused', {
                    body: `${event.data.frameName} paused at ${event.data.taskName}`,
                });
            }
            break;

        case 'debug.frame.resumed':
            store.updateFrame(event.data.frameId, {
                state: 'running',
                pauseReason: null,
                currentTask: null,
            });
            break;

        case 'debug.frame.completed':
            store.updateFrame(event.data.frameId, {
                state: 'completed',
            });
            break;

        case 'debug.session.deleted':
            store.clearSession();
            break;
    }
}
```

#### 7.4.2.1 API Hook with Error Handling

```typescript
// web-ui/src/hooks/useDebugApi.ts

export function useDebugApi(runId: number) {
    const debugStore = useDebugStore();

    const withErrorHandling = async <T>(
        operation: () => Promise<T>,
        loadingMessage?: string
    ): Promise<T | null> => {
        try {
            debugStore.setLoading(true);
            debugStore.clearError();
            const result = await operation();
            return result;
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Unknown error';
            debugStore.setError(message);
            return null;
        } finally {
            debugStore.setLoading(false);
        }
    };

    return {
        continueFrame: (frameId: number) =>
            withErrorHandling(() => api.debugContinue(runId, { frameId })),

        continueAll: () =>
            withErrorHandling(() => api.debugContinue(runId, { frameId: 0 })),

        pauseAll: () =>
            withErrorHandling(() => api.debugPause(runId, { frameId: 0 })),

        stepInto: (frameId: number) =>
            withErrorHandling(() => api.debugStep(runId, { frameId, mode: 'into' })),

        stepOver: (frameId: number) =>
            withErrorHandling(() => api.debugStep(runId, { frameId, mode: 'over' })),

        stepOut: (frameId: number) =>
            withErrorHandling(() => api.debugStep(runId, { frameId, mode: 'out' })),

        skipTask: (frameId: number) =>
            withErrorHandling(() => api.debugSkip(runId, { frameId })),

        abort: () =>
            withErrorHandling(() => api.debugAbort(runId)),

        addBreakpoint: (bp: Omit<Breakpoint, 'id' | 'hitCount'>) =>
            withErrorHandling(() => api.addBreakpoint(runId, bp)),

        removeBreakpoint: (id: string) =>
            withErrorHandling(() => api.removeBreakpoint(runId, id)),

        setVariable: (frameId: number, name: string, value: any) =>
            withErrorHandling(() => api.setVariable(runId, frameId, name, value)),

        injectTask: (frameId: number, task: InjectTaskRequest) =>
            withErrorHandling(() => api.injectTask(runId, frameId, task)),

        evaluateExpression: (frameId: number, expression: string) =>
            withErrorHandling(() => api.evaluateExpression(runId, frameId, expression)),
    };
}
```

#### 7.4.3 Debug Toolbar Component

```tsx
// web-ui/src/components/debug/DebugToolbar.tsx

export function DebugToolbar({ runId }: { runId: number }) {
    const {
        frames,
        activeFrameId,
        setActiveFrame,
        getActiveFrame,
        getPausedFrames,
        hasAnyPausedFrame
    } = useDebugStore();
    const {
        continueAll, continueFrame,
        pauseAll,
        stepInto, stepOver, stepOut,
        skipTask, abort
    } = useDebugApi(runId);

    const activeFrame = getActiveFrame();
    const pausedFrames = getPausedFrames();
    const isActiveFramePaused = activeFrame?.state === 'paused';
    const hasRunningFrames = Array.from(frames.values()).some(f => f.state === 'running');

    return (
        <div className="flex flex-col bg-gray-800 border-b border-gray-700">
            {/* Frame selector row (shown when multiple frames exist) */}
            {frames.size > 1 && (
                <div className="flex items-center gap-2 px-2 py-1 border-b border-gray-700 bg-gray-850">
                    <span className="text-xs text-gray-400">Frames:</span>
                    <FrameSelector
                        frames={Array.from(frames.values())}
                        activeFrameId={activeFrameId}
                        onSelect={setActiveFrame}
                    />
                    {pausedFrames.length > 1 && (
                        <span className="text-xs text-yellow-500 ml-2">
                            {pausedFrames.length} frames paused
                        </span>
                    )}
                </div>
            )}

            {/* Main toolbar row */}
            <div className="flex items-center gap-2 p-2">
                {/* Active frame status */}
                <div className="flex items-center gap-2 px-3 py-1 rounded bg-gray-700 min-w-[200px]">
                    {activeFrame?.state === 'paused' ? (
                        <>
                            <PauseIcon className="w-4 h-4 text-yellow-500" />
                            <div className="flex flex-col">
                                <span className="text-yellow-500 text-sm">
                                    {activeFrame.name} - Paused
                                </span>
                                {activeFrame.currentTask && (
                                    <span className="text-gray-400 text-xs">
                                        #{activeFrame.currentTask.taskIndex} {activeFrame.currentTask.taskTitle || activeFrame.currentTask.taskName}
                                    </span>
                                )}
                            </div>
                        </>
                    ) : activeFrame?.state === 'running' ? (
                        <>
                            <PlayIcon className="w-4 h-4 text-green-500 animate-pulse" />
                            <span className="text-green-500">{activeFrame.name} - Running</span>
                        </>
                    ) : activeFrame?.state === 'completed' ? (
                        <>
                            <CheckIcon className="w-4 h-4 text-blue-500" />
                            <span className="text-blue-500">{activeFrame.name} - Completed</span>
                        </>
                    ) : (
                        <>
                            <StopIcon className="w-4 h-4 text-gray-500" />
                            <span className="text-gray-500">No active frame</span>
                        </>
                    )}
                </div>

                {/* Continue/Pause controls */}
                <div className="flex items-center gap-1 border-l border-gray-600 pl-2">
                    {/* Continue active frame */}
                    <Button
                        onClick={() => continueFrame(activeFrameId)}
                        disabled={!isActiveFramePaused}
                        title="Continue Frame (F8)"
                    >
                        <PlayIcon className="w-4 h-4" />
                    </Button>

                    {/* Continue all (when multiple paused) */}
                    {pausedFrames.length > 1 && (
                        <Button
                            onClick={continueAll}
                            disabled={pausedFrames.length === 0}
                            title="Continue All Frames"
                            variant="secondary"
                        >
                            <PlayIcon className="w-4 h-4" />
                            <span className="text-xs ml-1">All</span>
                        </Button>
                    )}

                    <Button
                        onClick={pauseAll}
                        disabled={!hasRunningFrames}
                        title="Pause All"
                    >
                        <PauseIcon className="w-4 h-4" />
                    </Button>
                </div>

                {/* Step controls (frame-specific) */}
                <div className="flex items-center gap-1 border-l border-gray-600 pl-2">
                    <Button
                        onClick={() => stepInto(activeFrameId)}
                        disabled={!isActiveFramePaused}
                        title="Step Into (F11)"
                    >
                        <StepIntoIcon className="w-4 h-4" />
                    </Button>

                    <Button
                        onClick={() => stepOver(activeFrameId)}
                        disabled={!isActiveFramePaused}
                        title="Step Over (F10)"
                    >
                        <StepOverIcon className="w-4 h-4" />
                    </Button>

                    <Button
                        onClick={() => stepOut(activeFrameId)}
                        disabled={!isActiveFramePaused}
                        title="Step Out (Shift+F11)"
                    >
                        <StepOutIcon className="w-4 h-4" />
                    </Button>
                </div>

                {/* Skip and Abort */}
                <div className="flex items-center gap-1 border-l border-gray-600 pl-2">
                    <Button
                        onClick={() => skipTask(activeFrameId)}
                        disabled={!isActiveFramePaused}
                        title="Skip Task in Frame"
                        variant="warning"
                    >
                        <SkipIcon className="w-4 h-4" />
                        Skip
                    </Button>

                    <Button
                        onClick={abort}
                        disabled={frames.size === 0}
                        title="Abort Test"
                        variant="danger"
                    >
                        <StopIcon className="w-4 h-4" />
                        Abort
                    </Button>
                </div>
            </div>
        </div>
    );
}

// Frame selector component
function FrameSelector({ frames, activeFrameId, onSelect }: FrameSelectorProps) {
    return (
        <div className="flex items-center gap-1 flex-wrap">
            {frames.map(frame => (
                <button
                    key={frame.id}
                    onClick={() => onSelect(frame.id)}
                    className={`
                        px-2 py-0.5 rounded text-xs font-medium transition-colors
                        ${frame.id === activeFrameId
                            ? 'bg-blue-600 text-white'
                            : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                        }
                        ${frame.state === 'paused' ? 'ring-1 ring-yellow-500' : ''}
                        ${frame.state === 'completed' ? 'opacity-50' : ''}
                    `}
                >
                    <span className="mr-1">
                        {frame.state === 'paused' && '⏸'}
                        {frame.state === 'running' && '▶'}
                        {frame.state === 'completed' && '✓'}
                    </span>
                    {frame.name}
                </button>
            ))}
        </div>
    );
}
```

#### 7.4.4 Breakpoint Panel

```tsx
// web-ui/src/components/debug/BreakpointPanel.tsx

export function BreakpointPanel({ runId }: { runId: number }) {
    const { breakpoints, addBreakpoint, removeBreakpoint, toggleBreakpoint } = useDebugStore();
    const [showAddDialog, setShowAddDialog] = useState(false);

    return (
        <div className="flex flex-col h-full">
            <div className="flex items-center justify-between p-2 border-b border-gray-700">
                <h3 className="font-medium">Breakpoints</h3>
                <Button size="sm" onClick={() => setShowAddDialog(true)}>
                    <PlusIcon className="w-4 h-4" />
                </Button>
            </div>

            <div className="flex-1 overflow-auto">
                {breakpoints.length === 0 ? (
                    <div className="p-4 text-gray-500 text-center">
                        No breakpoints set
                    </div>
                ) : (
                    <ul className="divide-y divide-gray-700">
                        {breakpoints.map((bp) => (
                            <li key={bp.id} className="flex items-center gap-2 p-2 hover:bg-gray-800">
                                <input
                                    type="checkbox"
                                    checked={bp.enabled}
                                    onChange={() => toggleBreakpoint(bp.id)}
                                    className="rounded"
                                />
                                <BreakpointIcon type={bp.type} />
                                <span className="flex-1 truncate">
                                    {formatBreakpoint(bp)}
                                </span>
                                <span className="text-gray-500 text-xs">
                                    {bp.hitCount > 0 && `×${bp.hitCount}`}
                                </span>
                                <Button
                                    size="xs"
                                    variant="ghost"
                                    onClick={() => removeBreakpoint(bp.id)}
                                >
                                    <TrashIcon className="w-3 h-3" />
                                </Button>
                            </li>
                        ))}
                    </ul>
                )}
            </div>

            {showAddDialog && (
                <AddBreakpointDialog
                    onAdd={(bp) => {
                        addBreakpoint(bp);
                        setShowAddDialog(false);
                    }}
                    onClose={() => setShowAddDialog(false)}
                />
            )}
        </div>
    );
}
```

#### 7.4.5 Variable Inspector

```tsx
// web-ui/src/components/debug/VariableInspector.tsx

export function VariableInspector({ runId }: { runId: number }) {
    const { activeFrameId, getActiveFrame, setPendingVariable, removePendingVariable } = useDebugStore();
    const [filter, setFilter] = useState('');
    const [editingVar, setEditingVar] = useState<string | null>(null);

    const activeFrame = getActiveFrame();

    if (!activeFrame || activeFrame.state !== 'paused' || !activeFrame.currentTask) {
        return (
            <div className="p-4 text-gray-500 text-center">
                {activeFrame?.state === 'running'
                    ? `Frame "${activeFrame.name}" is running...`
                    : activeFrame?.state === 'completed'
                    ? `Frame "${activeFrame.name}" has completed`
                    : 'Select a paused frame to inspect variables'}
            </div>
        );
    }

    const { currentTask, pendingVariables } = activeFrame;
    const filteredVars = filterVariables(currentTask.availableVars, filter);

    return (
        <div className="flex flex-col h-full">
            <div className="p-2 border-b border-gray-700 flex items-center gap-2">
                <input
                    type="text"
                    placeholder="Filter variables..."
                    value={filter}
                    onChange={(e) => setFilter(e.target.value)}
                    className="flex-1 px-2 py-1 bg-gray-800 border border-gray-600 rounded"
                />
                <span className="text-xs text-gray-500">
                    Frame: {activeFrame.name}
                </span>
            </div>

            <div className="flex-1 overflow-auto">
                {/* Pending Changes */}
                {Object.keys(pendingVariables).length > 0 && (
                    <div className="border-b border-gray-700">
                        <div className="px-2 py-1 bg-yellow-900/30 text-yellow-500 text-sm font-medium">
                            Pending Changes
                        </div>
                        {Object.entries(pendingVariables).map(([name, value]) => (
                            <div key={name} className="flex items-center gap-2 px-2 py-1 bg-yellow-900/10">
                                <span className="text-yellow-400">{name}</span>
                                <span className="text-gray-500">→</span>
                                <span className="flex-1 text-white truncate">
                                    {JSON.stringify(value)}
                                </span>
                                <Button
                                    size="xs"
                                    variant="ghost"
                                    onClick={() => removePendingVariable(activeFrameId, name)}
                                >
                                    <XIcon className="w-3 h-3" />
                                </Button>
                            </div>
                        ))}
                    </div>
                )}

                {/* Variable Tree */}
                <VariableTree
                    variables={filteredVars}
                    onEdit={(name) => setEditingVar(name)}
                    pendingChanges={pendingVariables}
                />
            </div>

            {editingVar && (
                <EditVariableDialog
                    name={editingVar}
                    currentValue={getNestedValue(currentTask.availableVars, editingVar)}
                    onSave={(value) => {
                        setPendingVariable(activeFrameId, editingVar, value);
                        setEditingVar(null);
                    }}
                    onClose={() => setEditingVar(null)}
                />
            )}
        </div>
    );
}

function VariableTree({ variables, onEdit, pendingChanges, path = '' }: VariableTreeProps) {
    return (
        <ul className="pl-4">
            {Object.entries(variables).map(([key, value]) => {
                const fullPath = path ? `${path}.${key}` : key;
                const isPending = fullPath in pendingChanges;

                if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
                    return (
                        <li key={key}>
                            <details>
                                <summary className="cursor-pointer hover:bg-gray-800 px-1">
                                    <span className="text-purple-400">{key}</span>
                                </summary>
                                <VariableTree
                                    variables={value}
                                    onEdit={onEdit}
                                    pendingChanges={pendingChanges}
                                    path={fullPath}
                                />
                            </details>
                        </li>
                    );
                }

                return (
                    <li
                        key={key}
                        className={`flex items-center gap-2 px-1 hover:bg-gray-800 ${isPending ? 'bg-yellow-900/10' : ''}`}
                    >
                        <span className="text-blue-400">{key}</span>
                        <span className="text-gray-500">:</span>
                        <span className="flex-1 text-white truncate">
                            {formatValue(value)}
                        </span>
                        <Button
                            size="xs"
                            variant="ghost"
                            onClick={() => onEdit(fullPath)}
                            title="Edit"
                        >
                            <PencilIcon className="w-3 h-3" />
                        </Button>
                    </li>
                );
            })}
        </ul>
    );
}
```

#### 7.4.6 Task Inspector

```tsx
// web-ui/src/components/debug/TaskInspector.tsx

export function TaskInspector({ runId }: { runId: number }) {
    const { getActiveFrame } = useDebugStore();
    const [viewMode, setViewMode] = useState<'resolved' | 'raw' | 'configVars'>('resolved');

    const activeFrame = getActiveFrame();

    if (!activeFrame || activeFrame.state !== 'paused' || !activeFrame.currentTask) {
        return (
            <div className="p-4 text-gray-500 text-center">
                {activeFrame?.state === 'running'
                    ? `Frame "${activeFrame.name}" is running...`
                    : activeFrame?.state === 'completed'
                    ? `Frame "${activeFrame.name}" has completed`
                    : 'Select a paused frame to inspect task config'}
            </div>
        );
    }

    const { currentTask } = activeFrame;

    return (
        <div className="flex flex-col h-full">
            <div className="flex items-center gap-2 p-2 border-b border-gray-700">
                <h3 className="font-medium flex-1">
                    #{currentTask.taskIndex} {currentTask.taskTitle || currentTask.taskName}
                </h3>
                <span className="text-xs text-gray-500">
                    Frame: {activeFrame.name}
                </span>
                <select
                    value={viewMode}
                    onChange={(e) => setViewMode(e.target.value as any)}
                    className="px-2 py-1 bg-gray-800 border border-gray-600 rounded text-sm"
                >
                    <option value="resolved">Resolved Config</option>
                    <option value="raw">Raw Config</option>
                    <option value="configVars">ConfigVars</option>
                </select>
            </div>

            <div className="flex-1 overflow-auto p-2">
                <pre className="text-sm font-mono whitespace-pre-wrap">
                    {viewMode === 'resolved' && (
                        <code>{yaml.stringify(currentTask.resolvedConfig)}</code>
                    )}
                    {viewMode === 'raw' && (
                        <code>{yaml.stringify(currentTask.rawConfig)}</code>
                    )}
                    {viewMode === 'configVars' && (
                        <ConfigVarsView
                            configVars={currentTask.configVars}
                            resolvedValues={currentTask.resolvedConfig}
                        />
                    )}
                </pre>
            </div>

            <div className="flex gap-2 p-2 border-t border-gray-700">
                <Button variant="secondary" className="flex-1">
                    Edit Config
                </Button>
            </div>
        </div>
    );
}

function ConfigVarsView({ configVars, resolvedValues }: ConfigVarsViewProps) {
    return (
        <div className="space-y-2">
            {Object.entries(configVars).map(([field, expression]) => (
                <div key={field} className="border border-gray-700 rounded p-2">
                    <div className="text-blue-400 font-medium">{field}</div>
                    <div className="text-gray-400 text-xs mt-1">
                        Expression: <code className="text-purple-400">{expression}</code>
                    </div>
                    <div className="text-green-400 mt-1">
                        → {JSON.stringify(resolvedValues[field])}
                    </div>
                </div>
            ))}
        </div>
    );
}
```

#### 7.4.7 Task Injector

```tsx
// web-ui/src/components/debug/TaskInjector.tsx

export function TaskInjector({ runId, onClose }: { runId: number; onClose: () => void }) {
    const { activeFrameId, getActiveFrame } = useDebugStore();
    const { injectTask } = useDebugApi(runId);
    const { data: taskDescriptors } = useTaskDescriptors();
    const [selectedTask, setSelectedTask] = useState<string>('');
    const [insertMode, setInsertMode] = useState<'before_current' | 'after_current'>('after_current');
    const [config, setConfig] = useState<Record<string, any>>({});

    const activeFrame = getActiveFrame();
    const selectedDescriptor = taskDescriptors?.find(t => t.name === selectedTask);

    const handleInject = async () => {
        await injectTask(activeFrameId, {
            task: {
                name: selectedTask,
                config,
            },
            insertMode,
        });
        onClose();
    };

    if (!activeFrame || activeFrame.state !== 'paused') {
        return (
            <div className="p-4 text-gray-500 text-center">
                Select a paused frame to inject tasks
            </div>
        );
    }

    return (
        <div className="flex flex-col h-full">
            <div className="flex items-center justify-between p-2 border-b border-gray-700">
                <div>
                    <h3 className="font-medium">Inject Task</h3>
                    <span className="text-xs text-gray-500">Into frame: {activeFrame.name}</span>
                </div>
                <Button size="sm" variant="ghost" onClick={onClose}>
                    <XIcon className="w-4 h-4" />
                </Button>
            </div>

            <div className="flex-1 overflow-auto p-4 space-y-4">
                {/* Insert Mode */}
                <div>
                    <label className="block text-sm font-medium mb-1">Insert Position</label>
                    <div className="flex gap-4">
                        <label className="flex items-center gap-2">
                            <input
                                type="radio"
                                value="before_current"
                                checked={insertMode === 'before_current'}
                                onChange={() => setInsertMode('before_current')}
                            />
                            Before current task
                        </label>
                        <label className="flex items-center gap-2">
                            <input
                                type="radio"
                                value="after_current"
                                checked={insertMode === 'after_current'}
                                onChange={() => setInsertMode('after_current')}
                            />
                            After current task
                        </label>
                    </div>
                </div>

                {/* Task Type Selection */}
                <div>
                    <label className="block text-sm font-medium mb-1">Task Type</label>
                    <select
                        value={selectedTask}
                        onChange={(e) => {
                            setSelectedTask(e.target.value);
                            setConfig({});
                        }}
                        className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded"
                    >
                        <option value="">Select a task...</option>
                        {taskDescriptors?.map((task) => (
                            <option key={task.name} value={task.name}>
                                {task.name} - {task.description}
                            </option>
                        ))}
                    </select>
                </div>

                {/* Task Config Form */}
                {selectedDescriptor && (
                    <div>
                        <label className="block text-sm font-medium mb-1">Configuration</label>
                        <TaskConfigForm
                            descriptor={selectedDescriptor}
                            config={config}
                            onChange={setConfig}
                        />
                    </div>
                )}
            </div>

            <div className="flex gap-2 p-2 border-t border-gray-700">
                <Button variant="secondary" onClick={onClose} className="flex-1">
                    Cancel
                </Button>
                <Button
                    variant="primary"
                    onClick={handleInject}
                    disabled={!selectedTask}
                    className="flex-1"
                >
                    Inject Task
                </Button>
            </div>
        </div>
    );
}
```

**Files:**
```
web-ui/src/
├── stores/
│   └── debugStore.ts           # Debug state management
├── hooks/
│   ├── useDebugApi.ts          # API hooks for debug operations
│   └── useDebugEvents.ts       # SSE event handling
├── components/debug/
│   ├── DebugToolbar.tsx        # Control buttons
│   ├── BreakpointPanel.tsx     # Breakpoint management
│   ├── VariableInspector.tsx   # Variable viewing/editing
│   ├── TaskInspector.tsx       # Current task inspection
│   ├── TaskInjector.tsx        # On-the-fly task creation
│   └── AddBreakpointDialog.tsx # Breakpoint creation dialog
├── pages/
│   └── TestDebugger.tsx        # Debug view page
└── api/
    └── debug.ts                # Debug API client functions
```

---

### Phase 7.5: Builder Integration

**Goal:** Integrate debugging with the test builder for seamless debug workflow.

#### 7.5.1 Debug Run from Builder

```tsx
// web-ui/src/components/builder/toolbar/BuilderToolbar.tsx

export function BuilderToolbar() {
    const { tasks, testConfig } = useBuilderStore();
    const navigate = useNavigate();

    const handleDebugRun = async () => {
        // 1. Register test (or use draft)
        const yaml = serializeToYaml(tasks, testConfig);
        const testId = await registerTest(yaml);

        // 2. Start debug session
        const { runId, sessionId } = await scheduleDebug({
            testId,
            pauseOnStart: true,
        });

        // 3. Navigate to debug view
        navigate(`/debug/${runId}`);
    };

    return (
        <div className="flex items-center gap-2">
            {/* ... existing buttons ... */}

            <Button
                onClick={handleDebugRun}
                variant="secondary"
                title="Start in Debug Mode"
            >
                <BugIcon className="w-4 h-4 mr-1" />
                Debug
            </Button>
        </div>
    );
}
```

#### 7.5.2 Breakpoint Markers in Graph/List View

```tsx
// Extend existing task components to show breakpoint markers

// In TaskGraphNode.tsx or TaskList.tsx
function TaskNode({ task, breakpoints }: TaskNodeProps) {
    const hasBreakpoint = breakpoints.some(bp =>
        (bp.type === 'task_index' && bp.taskIndex === task.index) ||
        (bp.type === 'task_id' && bp.taskId === task.id) ||
        (bp.type === 'task_name' && bp.taskName === task.name)
    );

    return (
        <div className="relative">
            {hasBreakpoint && (
                <div className="absolute -left-2 top-1/2 -translate-y-1/2">
                    <div className="w-3 h-3 rounded-full bg-red-500" />
                </div>
            )}
            {/* ... existing task rendering ... */}
        </div>
    );
}
```

#### 7.5.3 Debug View Page

```tsx
// web-ui/src/pages/TestDebugger.tsx

export function TestDebugger() {
    const { runId } = useParams<{ runId: string }>();
    const { data: debugSession, isLoading } = useDebugSession(Number(runId));
    const { data: testRun } = useTestRun(Number(runId));
    const { frames, activeFrameId, getActiveFrame, getPausedFrames } = useDebugStore();

    useDebugEvents(Number(runId));

    if (isLoading) {
        return <LoadingSpinner />;
    }

    if (!debugSession) {
        return <Navigate to={`/run/${runId}`} />;
    }

    const activeFrame = getActiveFrame();
    const pausedFrames = getPausedFrames();

    return (
        <div className="flex flex-col h-screen">
            {/* Debug Toolbar with frame selector */}
            <DebugToolbar runId={Number(runId)} />

            {/* Multi-frame status bar (when multiple frames are paused) */}
            {pausedFrames.length > 1 && (
                <div className="px-4 py-2 bg-yellow-900/30 border-b border-yellow-700 flex items-center gap-4">
                    <WarningIcon className="w-4 h-4 text-yellow-500" />
                    <span className="text-yellow-500 text-sm">
                        Multiple frames paused: {pausedFrames.map(f => f.name).join(', ')}
                    </span>
                    <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => /* continueAll */}
                        className="ml-auto"
                    >
                        Continue All
                    </Button>
                </div>
            )}

            {/* Main Content */}
            <div className="flex-1 flex overflow-hidden">
                {/* Left Panel: Task Graph with frame visualization */}
                <div className="w-1/2 border-r border-gray-700 flex flex-col">
                    <DebugTaskGraph
                        tasks={testRun?.tasks || []}
                        frames={Array.from(frames.values())}
                        activeFrameId={activeFrameId}
                        breakpoints={debugSession.breakpoints}
                        onTaskClick={(task) => {/* toggle breakpoint */}}
                        onFrameClick={(frameId) => {/* set active frame */}}
                    />
                </div>

                {/* Right Panel: Frame-specific Inspection */}
                <div className="w-1/2 flex flex-col">
                    {/* Active frame header */}
                    {activeFrame && (
                        <div className="px-4 py-2 bg-gray-800 border-b border-gray-700 flex items-center gap-2">
                            <span className={`w-2 h-2 rounded-full ${
                                activeFrame.state === 'paused' ? 'bg-yellow-500' :
                                activeFrame.state === 'running' ? 'bg-green-500 animate-pulse' :
                                'bg-gray-500'
                            }`} />
                            <span className="font-medium">{activeFrame.name}</span>
                            {activeFrame.currentTask && (
                                <span className="text-gray-400 text-sm">
                                    → #{activeFrame.currentTask.taskIndex} {activeFrame.currentTask.taskName}
                                </span>
                            )}
                        </div>
                    )}

                    <Tabs defaultValue="variables" className="flex-1 flex flex-col">
                        <TabsList>
                            <TabsTrigger value="variables">
                                Variables
                                {activeFrame?.pendingVariables && Object.keys(activeFrame.pendingVariables).length > 0 && (
                                    <span className="ml-1 px-1.5 py-0.5 text-xs bg-yellow-600 rounded-full">
                                        {Object.keys(activeFrame.pendingVariables).length}
                                    </span>
                                )}
                            </TabsTrigger>
                            <TabsTrigger value="task">Task Config</TabsTrigger>
                            <TabsTrigger value="breakpoints">Breakpoints</TabsTrigger>
                            <TabsTrigger value="inject">Inject</TabsTrigger>
                            <TabsTrigger value="frames">Frames</TabsTrigger>
                        </TabsList>

                        <TabsContent value="variables" className="flex-1 overflow-hidden">
                            <VariableInspector runId={Number(runId)} />
                        </TabsContent>

                        <TabsContent value="task" className="flex-1 overflow-hidden">
                            <TaskInspector runId={Number(runId)} />
                        </TabsContent>

                        <TabsContent value="breakpoints" className="flex-1 overflow-hidden">
                            <BreakpointPanel runId={Number(runId)} />
                        </TabsContent>

                        <TabsContent value="inject" className="flex-1 overflow-hidden">
                            <TaskInjector runId={Number(runId)} onClose={() => {}} />
                        </TabsContent>

                        <TabsContent value="frames" className="flex-1 overflow-hidden">
                            <FrameListPanel runId={Number(runId)} />
                        </TabsContent>
                    </Tabs>
                </div>
            </div>
        </div>
    );
}

// Frame list panel showing all execution frames
function FrameListPanel({ runId }: { runId: number }) {
    const { frames, activeFrameId, setActiveFrame } = useDebugStore();
    const { continueFrame, pauseFrame } = useDebugApi(runId);

    return (
        <div className="flex flex-col h-full">
            <div className="p-2 border-b border-gray-700">
                <h3 className="font-medium">Execution Frames</h3>
                <p className="text-xs text-gray-500">
                    Each concurrent/background branch has its own frame
                </p>
            </div>

            <div className="flex-1 overflow-auto">
                {Array.from(frames.values()).map(frame => (
                    <div
                        key={frame.id}
                        onClick={() => setActiveFrame(frame.id)}
                        className={`
                            p-3 border-b border-gray-700 cursor-pointer
                            ${frame.id === activeFrameId ? 'bg-blue-900/30' : 'hover:bg-gray-800'}
                        `}
                    >
                        <div className="flex items-center gap-2">
                            <span className={`w-2 h-2 rounded-full ${
                                frame.state === 'paused' ? 'bg-yellow-500' :
                                frame.state === 'running' ? 'bg-green-500 animate-pulse' :
                                'bg-gray-500'
                            }`} />
                            <span className="font-medium">{frame.name}</span>
                            <span className="text-xs text-gray-500">#{frame.id}</span>
                            {frame.parentFrameId > 0 && (
                                <span className="text-xs text-gray-500">
                                    (child of #{frame.parentFrameId})
                                </span>
                            )}
                        </div>

                        {frame.currentTask && (
                            <div className="mt-1 text-sm text-gray-400">
                                At: #{frame.currentTask.taskIndex} {frame.currentTask.taskName}
                            </div>
                        )}

                        <div className="mt-2 flex gap-2">
                            {frame.state === 'paused' && (
                                <Button
                                    size="xs"
                                    onClick={(e) => {
                                        e.stopPropagation();
                                        continueFrame(frame.id);
                                    }}
                                >
                                    Continue
                                </Button>
                            )}
                            {frame.state === 'running' && (
                                <Button
                                    size="xs"
                                    variant="secondary"
                                    onClick={(e) => {
                                        e.stopPropagation();
                                        pauseFrame(frame.id);
                                    }}
                                >
                                    Pause
                                </Button>
                            )}
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
}
```

#### 7.5.4 Debug Task Graph with Frame Visualization

The debug graph shows execution frames as lanes/swimlanes:

```tsx
// web-ui/src/components/debug/DebugTaskGraph.tsx

export function DebugTaskGraph({
    tasks,
    frames,
    activeFrameId,
    breakpoints,
    onTaskClick,
    onFrameClick,
}: DebugTaskGraphProps) {
    // Group tasks by frame
    const tasksByFrame = useMemo(() => {
        const map = new Map<number, TaskInfo[]>();
        for (const frame of frames) {
            map.set(frame.id, tasks.filter(t => frame.taskIndices.includes(t.index)));
        }
        return map;
    }, [tasks, frames]);

    return (
        <div className="h-full flex flex-col">
            {/* Frame lanes */}
            <div className="flex-1 flex overflow-auto">
                {frames.map(frame => (
                    <div
                        key={frame.id}
                        className={`
                            flex-1 min-w-[300px] border-r border-gray-700
                            ${frame.id === activeFrameId ? 'bg-blue-900/10' : ''}
                        `}
                    >
                        {/* Frame header */}
                        <div
                            onClick={() => onFrameClick(frame.id)}
                            className={`
                                sticky top-0 p-2 bg-gray-800 border-b border-gray-700
                                cursor-pointer hover:bg-gray-750
                                ${frame.state === 'paused' ? 'border-l-2 border-l-yellow-500' : ''}
                            `}
                        >
                            <div className="flex items-center gap-2">
                                <span className={`w-2 h-2 rounded-full ${
                                    frame.state === 'paused' ? 'bg-yellow-500' :
                                    frame.state === 'running' ? 'bg-green-500 animate-pulse' :
                                    'bg-gray-500'
                                }`} />
                                <span className="font-medium text-sm">{frame.name}</span>
                            </div>
                        </div>

                        {/* Tasks in this frame */}
                        <div className="p-2 space-y-2">
                            {tasksByFrame.get(frame.id)?.map(task => (
                                <DebugTaskNode
                                    key={task.index}
                                    task={task}
                                    isCurrentTask={frame.currentTask?.taskIndex === task.index}
                                    hasBreakpoint={breakpoints.some(bp =>
                                        (bp.type === 'task_index' && bp.taskIndex === task.index) ||
                                        (bp.type === 'task_id' && bp.taskId === task.id)
                                    )}
                                    onClick={() => onTaskClick(task)}
                                />
                            ))}
                        </div>
                    </div>
                ))}
            </div>

            {/* Legend */}
            <div className="p-2 bg-gray-800 border-t border-gray-700 flex gap-4 text-xs text-gray-400">
                <span><span className="inline-block w-2 h-2 rounded-full bg-yellow-500 mr-1" /> Paused</span>
                <span><span className="inline-block w-2 h-2 rounded-full bg-green-500 mr-1" /> Running</span>
                <span><span className="inline-block w-2 h-2 rounded-full bg-red-500 mr-1" /> Breakpoint</span>
                <span><span className="inline-block w-2 h-2 rounded-full bg-blue-500 mr-1" /> Current task</span>
            </div>
        </div>
    );
}

function DebugTaskNode({ task, isCurrentTask, hasBreakpoint, onClick }: DebugTaskNodeProps) {
    return (
        <div
            onClick={onClick}
            className={`
                relative p-2 rounded border cursor-pointer transition-colors
                ${isCurrentTask
                    ? 'bg-blue-900/50 border-blue-500'
                    : 'bg-gray-800 border-gray-600 hover:border-gray-500'
                }
            `}
        >
            {/* Breakpoint indicator */}
            {hasBreakpoint && (
                <div className="absolute -left-1 top-1/2 -translate-y-1/2 w-2 h-2 rounded-full bg-red-500" />
            )}

            {/* Current task indicator */}
            {isCurrentTask && (
                <div className="absolute -right-1 top-1/2 -translate-y-1/2">
                    <ArrowRightIcon className="w-4 h-4 text-blue-500" />
                </div>
            )}

            <div className="text-xs text-gray-500">#{task.index}</div>
            <div className="font-medium text-sm truncate">
                {task.title || task.name}
            </div>
            <div className="text-xs text-gray-400 truncate">{task.name}</div>
        </div>
    );
}
```

**Files:**
```
web-ui/src/
├── pages/
│   └── TestDebugger.tsx        # Main debug view
├── components/builder/toolbar/
│   └── BuilderToolbar.tsx      # Add debug button
└── App.tsx                     # Add /debug/:runId route
```

---

## Implementation Checklist

### Phase 7.1: Debug Session Core ✅ COMPLETED
- [x] Create `pkg/debugger/` package structure
- [x] Implement `DebugManager` for session management
- [x] Implement `DebugSession` with pause/resume mechanics
- [x] Implement `ExecutionFrame` for parallel execution tracking
- [x] Implement frame creation/completion lifecycle
- [x] Implement `Breakpoint` types and matching logic
- [x] Define `DebugHook` interface with frame support
- [x] Integrate hook into `TaskScheduler.ExecuteTask()`
- [x] Add `SetDebugHook()` and `SetTaskFrame()` to scheduler
- [x] Add `taskFrameMap` for task-to-frame mapping
- [x] Integrate with `TestRunner` for debug mode scheduling
- [x] Add task depth calculation to scheduler (uses existing `taskDepth` field)

**Implementation Notes (Phase 7.1):**
- Created `pkg/debugger/` package with: `errors.go`, `types.go`, `hook.go`, `breakpoint.go`, `frame.go`, `session.go`, `manager.go`
- Added `types.DebugHook` interface in `pkg/types/debug.go` to avoid circular imports
- Extended `types.TaskSchedulerRunner` interface with debug methods
- Extended `types.TestRunner` interface with `SetDebugHook()` method
- Added `ScheduleTestDebug()` method to `TestRunner` for debug mode scheduling
- Debug events implemented in Phase 7.2 (below)

### Phase 7.1.1: Glue Task Integration ✅ COMPLETED
- [x] Update `run_tasks_concurrent` to call `OnConcurrentStart`/`OnConcurrentEnd`
- [x] Update `run_task_background` to create background/foreground frames
- [x] Update `run_task_matrix` for concurrent mode frame creation
- [x] Ensure proper frame cleanup on task completion/cancellation

**Implementation Notes (Phase 7.1.1):**
- `run_tasks_concurrent/task.go`: Each concurrent child task gets its own frame via `OnConcurrentStart(t.ctx.Index, fmt.Sprintf("concurrent[%d]", i))` with deferred `OnConcurrentEnd(frameID)` cleanup
- `run_task_background/task.go`: Both `execBackgroundTask` and `execForegroundTask` create frames with names like `"background"` and `"foreground"` respectively, with proper frame cleanup in deferred functions
- `run_task_matrix/task.go`: When `RunConcurrent` is true, each matrix combination gets a frame via `OnConcurrentStart` with cleanup in deferred goroutine
- All glue tasks use `t.ctx.Scheduler.SetTaskFrame(task, frameID)` to associate child tasks with frames
- Frame cleanup is handled via deferred `OnConcurrentEnd(frameID)` calls to ensure cleanup even on errors/cancellation

### Phase 7.2: Debug Events ✅ COMPLETED
- [x] Add debug event types to `pkg/events/event.go`
- [x] Add frame-specific events (created, paused, resumed, completed)
- [x] Implement event emission in `DebugSession`
- [x] Add SSE publishing for debug events

**Implementation Notes (Phase 7.2):**
- Added event types: `debug.session_created`, `debug.session_aborted`, `debug.frame_created`, `debug.frame_paused`, `debug.frame_resumed`, `debug.frame_completed`, `debug.breakpoint_hit`
- Added corresponding data structs and `Publish*` methods to `EventBus`
- `DebugSession` emits events via `eventBus` field

### Phase 7.3: Debug API Endpoints ✅ COMPLETED
- [x] `POST /api/v1/test_runs/schedule_debug` - Start debug session
- [x] `GET /api/v1/debug/{runId}` - Get session state with all frames
- [x] `DELETE /api/v1/debug/{runId}` - Detach debugger
- [x] `POST /api/v1/debug/{runId}/frame` - Set active frame
- [x] `POST /api/v1/debug/{runId}/continue` - Continue (frame or all)
- [x] `POST /api/v1/debug/{runId}/pause` - Pause (frame or all)
- [x] `POST /api/v1/debug/{runId}/step` - Step in frame (into/over/out)
- [x] `POST /api/v1/debug/{runId}/skip` - Skip task in frame
- [x] `POST /api/v1/debug/{runId}/abort` - Abort (all frames)
- [x] Breakpoint CRUD endpoints (`GET/POST /api/v1/debug/{runId}/breakpoints`, `DELETE /api/v1/debug/{runId}/breakpoints/{bpId}`)
- [x] Frame-specific variable inspection/modification endpoints (`GET/POST /api/v1/debug/{runId}/frames/{frameId}/variables`)
- [x] Frame-specific expression evaluation endpoint (`POST /api/v1/debug/{runId}/frames/{frameId}/evaluate`)
- [x] Add routes to `server.go`
- [ ] Regenerate Swagger docs
- [ ] Task injection endpoints (deferred - not needed for MVP)

**Implementation Notes (Phase 7.3):**
- Created `pkg/web/api/debug_api.go` with all debug API handlers in a single file
- Added `types.DebugSession` interface in `pkg/types/debug.go` to avoid exposing internal debugger types to API layer
- Added API info types: `DebugFrameInfo`, `DebugTaskPauseInfo`, `DebugBreakpointConfig`, `DebugBreakpointInfo`
- Added conversion methods: `Breakpoint.ToBreakpointInfo()`, `ExecutionFrame.ToFrameInfo()`, `NewBreakpointFromConfig()`
- Added `evalVars` helper in `pkg/debugger/eval.go` for JQ expression evaluation in REPL
- Routes registered under `/api/v1/debug/{runId}/...` pattern (simplified from original `/api/v1/test_run/{runId}/debug/...`)
- All debug endpoints require authentication (protected APIs)
- `GetDebugSession` returns frames, breakpoints, and state in a single response
- `ContinueRequest` and `PauseRequest` accept optional `frameId` - if 0, operates on all frames

### Phase 7.4: React UI - Debug Components
- [ ] Create `debugStore.ts` with Zustand (frame-aware)
- [ ] Implement frame state management (Map<frameId, ExecutionFrame>)
- [ ] Add error/loading states to store
- [ ] Add watch expressions and variable history to store
- [ ] Create `useDebugApi.ts` hook with frame-specific methods and error handling
- [ ] Create `useDebugEvents.ts` hook with reconnection logic
- [ ] Create `useDebugKeyboard.ts` hook for keyboard shortcuts
- [ ] Implement `DebugToolbar.tsx` with frame selector
- [ ] Implement `FrameSelector.tsx` component
- [ ] Implement `BreakpointPanel.tsx` with add dialog
- [ ] Implement `VariableInspector.tsx` (frame-aware, with diff view)
- [ ] Implement `TaskInspector.tsx` (frame-aware)
- [ ] Implement `TaskInjector.tsx` (frame-aware)
- [ ] Implement `FrameListPanel.tsx` for frame management
- [ ] Implement `ExpressionEvaluator.tsx` (REPL panel)
- [ ] Implement `WatchPanel.tsx` for pinned expressions
- [ ] Add keyboard shortcuts (F8, F10, F11, Shift+F11, ?)
- [ ] Add error toast notifications
- [ ] Add loading spinners to control buttons
- [ ] Add SSE connection status indicator

### Phase 7.5: Builder Integration
- [ ] Add "Debug" button to `BuilderToolbar.tsx`
- [ ] Create `TestDebugger.tsx` page with multi-frame layout
- [ ] Implement `DebugTaskGraph.tsx` with frame lanes/swimlanes
- [ ] Add breakpoint markers to task nodes
- [ ] Add current task indicator per frame
- [ ] Add `/debug/:runId` route to `App.tsx`
- [ ] Handle multi-paused-frame notification bar

---

## File Summary

### New Backend Files
```
pkg/debugger/
├── manager.go         # DebugManager - session lifecycle management
├── session.go         # DebugSession - pause/resume, frames, control methods
├── frame.go           # ExecutionFrame and TaskPauseInfo structs with ToFrameInfo()
├── breakpoint.go      # Breakpoint types, matching logic, ToBreakpointInfo(), NewBreakpointFromConfig()
├── hook.go            # DebugHook interface definition, TaskHookInfo alias
├── types.go           # State enums (DebugState, FrameState, PauseReason, StepMode, BreakpointType)
├── errors.go          # Error types (ErrSkipTask, ErrAbortDebug)
└── eval.go            # evalVars helper for JQ expression evaluation (REPL)

pkg/web/api/
└── debug_api.go       # All debug API handlers (session, control, breakpoints, variables, evaluate)
```

### Modified Backend Files
```
# Core infrastructure
pkg/types/debug.go               # DebugHook interface, DebugSession interface, DebugTaskInfo, API info types
pkg/types/scheduler.go           # Added debug methods to TaskSchedulerRunner
pkg/types/test.go                # Added SetDebugHook to TestRunner interface
pkg/types/coordinator.go         # Added ScheduleTestDebug, GetDebugSession, DeleteDebugSession

# Scheduler integration
pkg/scheduler/scheduler.go       # Add SetDebugHook, SetTaskFrame, taskFrameMap, GetDebugHook
pkg/scheduler/task_execution.go  # Add debug hook calls (OnBeforeTask, OnAfterTask)

# Coordinator/Runner
pkg/assertoor/testrunner.go      # Add ScheduleTestDebug, debugManager field
pkg/assertoor/coordinator.go     # Implement coordinator debug methods

# Events
pkg/events/event.go              # Add debug event types (frame.created, frame.paused, etc.)
pkg/events/sse.go                # Add debug event publishing methods

# Web/API
pkg/web/server.go                # Add debug API routes
pkg/test/test.go                 # Implement SetDebugHook method

# Glue task modifications for frame creation
pkg/tasks/run_tasks_concurrent/task.go  # Call OnConcurrentStart/End per child
pkg/tasks/run_task_background/task.go   # Create bg/fg frames
pkg/tasks/run_task_matrix/task.go       # Create frames in concurrent mode
```

### New Frontend Files
```
web-ui/src/
├── pages/
│   └── TestDebugger.tsx              # Main debug view with multi-frame layout
├── stores/
│   └── debugStore.ts                 # Zustand store with frames, errors, watch expressions
├── hooks/
│   ├── useDebugApi.ts                # API hooks with error handling
│   ├── useDebugEvents.ts             # SSE with reconnection logic
│   └── useDebugKeyboard.ts           # Keyboard shortcuts (F8, F10, F11)
├── components/debug/
│   ├── DebugToolbar.tsx              # Toolbar with frame selector, connection status
│   ├── FrameSelector.tsx             # Frame tabs/buttons
│   ├── FrameListPanel.tsx            # Frame list with status and controls
│   ├── BreakpointPanel.tsx           # Breakpoint management (global)
│   ├── VariableInspector.tsx         # Frame-specific variable inspection with diff
│   ├── VariableDiff.tsx              # Show changed variables since last pause
│   ├── TaskInspector.tsx             # Frame-specific task config view
│   ├── TaskInjector.tsx              # Frame-specific task injection
│   ├── ExpressionEvaluator.tsx       # REPL for JQ expressions
│   ├── WatchPanel.tsx                # Pinned watch expressions
│   ├── DebugTaskGraph.tsx            # Multi-frame swimlane visualization
│   ├── DebugTaskNode.tsx             # Task node with breakpoint/current markers
│   ├── AddBreakpointDialog.tsx       # Breakpoint creation dialog
│   ├── ShortcutsHelp.tsx             # Keyboard shortcuts legend modal
│   └── ErrorToast.tsx                # Error notification component
└── api/
    └── debug.ts                      # Debug API client functions
```

### Modified Frontend Files
```
web-ui/src/App.tsx                              # Add /debug/:runId route
web-ui/src/components/builder/toolbar/BuilderToolbar.tsx  # Add "Debug" button
web-ui/src/components/graph/TaskGraphNode.tsx   # Add breakpoint markers
web-ui/src/components/task/TaskList.tsx         # Add breakpoint markers
```

---

---

## UX Improvements

### Priority 1: Essential Features

| Feature | Description | Component |
|---------|-------------|-----------|
| Error Toast | Show API errors as dismissible toast | `DebugToolbar.tsx` |
| Loading Indicators | Spinner on buttons during API calls | All control buttons |
| Keyboard Shortcuts | F8=Continue, F10=StepOver, F11=StepInto | `useDebugKeyboard.ts` |
| Connection Status | Show SSE connection state | `DebugToolbar.tsx` |

### Priority 2: High-Value Features

| Feature | Description | Effort |
|---------|-------------|--------|
| Expression Evaluator | REPL panel to test JQ expressions | Medium |
| Watch Expressions | Pin JQ expressions that update on each pause | Medium |
| Variable Diff View | Show what changed since last pause | Medium |
| Call Stack View | Visual hierarchy: main → concurrent[0] → task | Low |

### Priority 3: Quick Wins

| Feature | Description | Effort |
|---------|-------------|--------|
| Copy Variable Path | Click to copy JQ path (e.g., `.tasks.myTask.outputs.txHash`) | Low |
| Shortcuts Help | `?` to show keyboard shortcuts legend | Low |
| Collapse/Expand All | For variable tree | Low |
| Search Variables | Ctrl+F filter within variable panel | Low |
| Pin Variables | Star frequently accessed variables | Low |
| Task Docs Link | Link to README.md in task inspector | Low |
| Frame Status Badge | Show paused frame count in page title | Low |

### Priority 4: Nice-to-Have

| Feature | Description | Effort |
|---------|-------------|--------|
| Browser Notifications | Notify when frame pauses (if tab not focused) | Low |
| Export Debug Session | Export state as JSON for sharing | Low |
| Task Replay | Re-run single task with modified config | High |
| Conditional Auto-Continue | Set expression, auto-continue when true | Medium |
| Breakpoint Hit Highlight | Flash task node when breakpoint hits | Low |
| Frame Timeline | Visual timeline of frame execution/pauses | High |

### Keyboard Shortcuts

```typescript
// web-ui/src/hooks/useDebugKeyboard.ts

export function useDebugKeyboard(runId: number) {
    const { stepInto, stepOver, stepOut, continueFrame, abort } = useDebugApi(runId);
    const { activeFrameId, getActiveFrame } = useDebugStore();

    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            // Don't trigger if typing in an input
            if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
                return;
            }

            const activeFrame = getActiveFrame();
            if (!activeFrame || activeFrame.state !== 'paused') return;

            switch (e.key) {
                case 'F8':
                    e.preventDefault();
                    continueFrame(activeFrameId);
                    break;
                case 'F10':
                    e.preventDefault();
                    stepOver(activeFrameId);
                    break;
                case 'F11':
                    e.preventDefault();
                    if (e.shiftKey) {
                        stepOut(activeFrameId);
                    } else {
                        stepInto(activeFrameId);
                    }
                    break;
                case 'Escape':
                    // Show abort confirmation
                    break;
                case '?':
                    // Show shortcuts help
                    break;
            }
        };

        window.addEventListener('keydown', handleKeyDown);
        return () => window.removeEventListener('keydown', handleKeyDown);
    }, [activeFrameId]);
}
```

### Expression Evaluator Component

```tsx
// web-ui/src/components/debug/ExpressionEvaluator.tsx

export function ExpressionEvaluator({ runId }: { runId: number }) {
    const { activeFrameId, getActiveFrame, watchExpressions, addWatchExpression } = useDebugStore();
    const { evaluateExpression } = useDebugApi(runId);
    const [expression, setExpression] = useState('');
    const [history, setHistory] = useState<{ expr: string; result: any; error?: string }[]>([]);

    const activeFrame = getActiveFrame();

    const handleEvaluate = async () => {
        if (!expression.trim() || !activeFrame || activeFrame.state !== 'paused') return;

        const result = await evaluateExpression(activeFrameId, expression);
        if (result) {
            setHistory(prev => [...prev, {
                expr: expression,
                result: result.result,
                error: result.error,
            }]);
            setExpression('');
        }
    };

    return (
        <div className="flex flex-col h-full">
            <div className="p-2 border-b border-gray-700">
                <h3 className="font-medium">Expression Evaluator</h3>
                <p className="text-xs text-gray-500">Test JQ expressions against current scope</p>
            </div>

            {/* History */}
            <div className="flex-1 overflow-auto p-2 space-y-2 font-mono text-sm">
                {history.map((entry, i) => (
                    <div key={i} className="border-b border-gray-800 pb-2">
                        <div className="text-blue-400">{'>'} {entry.expr}</div>
                        {entry.error ? (
                            <div className="text-red-400 ml-2">Error: {entry.error}</div>
                        ) : (
                            <div className="text-green-400 ml-2 whitespace-pre-wrap">
                                {JSON.stringify(entry.result, null, 2)}
                            </div>
                        )}
                        <button
                            className="text-xs text-gray-500 hover:text-gray-300"
                            onClick={() => addWatchExpression(entry.expr)}
                        >
                            + Add to Watch
                        </button>
                    </div>
                ))}
            </div>

            {/* Input */}
            <div className="p-2 border-t border-gray-700">
                <div className="flex gap-2">
                    <input
                        type="text"
                        value={expression}
                        onChange={(e) => setExpression(e.target.value)}
                        onKeyDown={(e) => e.key === 'Enter' && handleEvaluate()}
                        placeholder=".tasks.myTask.outputs.value"
                        className="flex-1 px-2 py-1 bg-gray-800 border border-gray-600 rounded font-mono"
                        disabled={!activeFrame || activeFrame.state !== 'paused'}
                    />
                    <Button
                        onClick={handleEvaluate}
                        disabled={!activeFrame || activeFrame.state !== 'paused'}
                    >
                        Eval
                    </Button>
                </div>
            </div>
        </div>
    );
}
```

---

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Pause mechanism deadlocks | High | Use buffered channels, timeouts, careful lock ordering |
| Race conditions in session/frame state | High | Use RWMutex per session, frame-level locks for fine-grained access |
| Memory leaks from abandoned sessions | Medium | Cleanup on test completion, session timeout |
| Performance impact when not debugging | Low | Only activate hooks when debug session exists |
| Complex step logic edge cases | Medium | Thorough testing of step modes with nested tasks |
| UI state sync issues on reconnect | Medium | Full state in GET session endpoint, SSE for updates |
| Frame lifecycle edge cases | High | Careful handling of frame completion during pause, cleanup on abort |
| Concurrent frame pause coordination | Medium | `PauseAllOnBreakpoint` option, clear frame state machine |
| Frame orphaning on parent cancellation | Medium | Propagate cancellation to child frames, mark as completed |
| UI complexity with many frames | Medium | Frame list panel, collapsible frame lanes, clear visual hierarchy |
| Step-out across frame boundaries | Medium | Track frame hierarchy via `StepOutTargetFrame`, test thoroughly |
| Timeout during pause | Medium | Pause timeout timer when debug paused, track elapsed time |
| Variable modification timing | Medium | Document that changes apply to future tasks only, clear UI indication |
| Concurrent session creation | Low | Check for existing session in `CreateSession`, return error |
| SSE disconnection | Medium | Exponential backoff reconnection, full state sync on reconnect |
| Cleanup task interaction | Low | Add `IsCleanupTask` flag to `TaskHookInfo`, respect breakpoints |

---

## Open Questions (Resolved)

| Question | Resolution |
|----------|------------|
| How to access DebugHook from glue tasks? | Add `GetDebugHook()` to `TaskSchedulerRunner` interface |
| How do child tasks inherit frame ID? | Modify `newTaskState()` to inherit from parent via `taskFrameMap` |
| When do variable modifications take effect? | After `OnBeforeTask` returns, before `LoadConfig()` |
| How to handle abort during pause? | Close all resume channels, check for abort after resume |
| How to step-out across frame boundaries? | Track `StepOutTargetFrame`, stop in ancestor frame |
| How to prevent spurious timeouts during pause? | Add `ShouldPauseTimeout()` to hook, pause timer goroutine |
