# Debug Builder Integration Plan

## Overview

Integrate the debugger directly into the Test Builder page instead of having a separate debug page. This allows users to:
- See test execution state directly in the task list/graph they're familiar with
- Modify, add, or remove future tasks while paused
- Set breakpoints by clicking on tasks
- View variables and frame state in context

## Current Architecture

```
BuilderPage
├── BuilderToolbar (save, run, debug buttons)
├── SplitPane
│   ├── Left: TaskListEditor / TaskGraphEditor (switchable)
│   └── Right: TaskConfigEditor (selected task config)
└── TestRunModal (shows test execution logs)
```

## Target Architecture

```
BuilderPage
├── BuilderToolbar (+ debug controls when debugging)
├── DebugBar (frame selector, connection status - only when debugging)
├── SplitPane
│   ├── Left: TaskListEditor / TaskGraphEditor
│   │   └── Enhanced with execution state indicators
│   └── Right: TaskConfigEditor OR DebugInspector (contextual)
└── DebugBottomPanel (variables, expression evaluator - collapsible)
```

## Phase 1: Debug State in Builder

### 1.1 Builder Debug Mode State

Add debug mode state to the builder:

```typescript
// In BuilderPage or a new useBuilderDebug hook
interface BuilderDebugState {
  isDebugging: boolean;
  runId: number | null;
  sessionId: string | null;
}
```

When "Debug" is clicked:
1. Save test (existing)
2. Schedule debug test (existing)
3. **Stay on builder page** (change from current redirect)
4. Connect to SSE events
5. Enter debug mode UI

### 1.2 Debug Controls in Toolbar

When debugging, BuilderToolbar shows:
- **Stop Debug** button (red) - aborts session, exits debug mode
- **Continue** (F8) - resume execution
- **Step Over** (F10) - next task at same depth
- **Step Into** (F11) - step into nested tasks
- **Step Out** (Shift+F11) - step out of current scope
- **Skip** (F9) - skip current task
- **Pause All** - pause all running frames

Hide/disable normal toolbar items that don't apply during debug:
- "Run Test" - hidden (already debugging)
- "Debug" - hidden (already debugging)
- "Save" - could remain enabled for live modifications

### 1.3 Frame Selector Bar

New component below toolbar (only visible when debugging):

```
┌─────────────────────────────────────────────────────────────┐
│ [main ●] [background ○] [matrix[0] ●] [matrix[1] ○]   [SSE ●] │
└─────────────────────────────────────────────────────────────┘
● = paused, ○ = running, ◌ = completed
```

- Shows all execution frames as tabs
- Click to switch active frame
- Visual indicator for paused/running/completed
- Connection status indicator

## Phase 2: Task Execution Visualization

### 2.1 Task List Execution State

Enhance TaskListEditor to show execution state:

```
┌──────────────────────────────────────────────────────────────┐
│ Tasks                                                    [+] │
├──────────────────────────────────────────────────────────────┤
│ ✓ 1. check_clients_are_healthy          [success]           │
│ ▶ 2. generate_transaction        ← RUNNING                  │
│ ◉ 3. wait_for_inclusion          ← BREAKPOINT (click to rm) │
│   4. verify_transaction_receipt                              │
│   5. cleanup_resources                    [cleanup]          │
└──────────────────────────────────────────────────────────────┘
```

State indicators:
- `✓` Green check - completed successfully
- `✗` Red X - failed
- `▶` Yellow arrow - currently running
- `⏸` Blue pause - paused (hit breakpoint or step)
- `◉` Red dot - has breakpoint
- `○` Gray circle - pending/not executed
- `⊘` Gray slash - skipped

### 2.2 Task Graph Execution State

Enhance TaskGraphEditor nodes:

- **Completed nodes**: Green border, slightly faded
- **Running node**: Yellow/amber border, pulsing glow
- **Paused node**: Blue border, highlighted
- **Failed node**: Red border
- **Pending nodes**: Gray/default
- **Breakpoint indicator**: Red dot in corner

Add execution flow visualization:
- Animate edge when task transitions
- Show execution path taken so far

### 2.3 Breakpoint Management

Click on task to toggle breakpoint:
- In list: Click on row left margin or dedicated column
- In graph: Click on node corner or right-click menu

Breakpoint types from UI:
- **Task breakpoint** (click on task) - pause before this task
- **Conditional breakpoint** (right-click > Add conditional) - pause if condition met

Show breakpoints inline rather than in separate panel.

## Phase 3: Debug Inspector Panel

### 3.1 Contextual Right Panel

When debugging and a task is paused, the right panel switches from TaskConfigEditor to DebugInspector:

```
┌─────────────────────────────────────────────┐
│ Task: generate_transaction                  │
│ Status: Paused (breakpoint)                 │
├─────────────────────────────────────────────┤
│ [Variables] [Config] [Output]               │
├─────────────────────────────────────────────┤
│ Variables:                                  │
│ ├─ walletPrivKey: "0x1234..."              │
│ ├─ targetAddr: "0xabcd..."                 │
│ └─ clients: [{...}, {...}]                 │
│                                             │
│ Config (resolved):                          │
│ ├─ privateKey: $.walletPrivKey             │
│ │   → "0x1234..."                          │
│ └─ toAddress: $.targetAddr                 │
│     → "0xabcd..."                          │
└─────────────────────────────────────────────┘
```

Tabs:
- **Variables**: Current scope variables (editable)
- **Config**: Task config with resolved values shown
- **Output**: Task logs/output (if running or completed)

### 3.2 Bottom Panel (Collapsible)

New collapsible panel at bottom for advanced debug features:

```
┌─────────────────────────────────────────────────────────────┐
│ Debug Console                                    [▼ Collapse]│
├─────────────────────────────────────────────────────────────┤
│ > $.clients | length                                        │
│ 6                                                           │
│ > $.walletPrivKey                                           │
│ "0x1234..."                                                 │
│ > _                                                         │
└─────────────────────────────────────────────────────────────┘
```

Features:
- JQ expression evaluator (REPL)
- Watch expressions
- Variable diff view (what changed since last pause)

Default state: collapsed (just a thin bar showing "Debug Console")

## Phase 4: Live Task Modification

### 4.1 Edit Future Tasks

While paused, allow editing tasks that haven't executed yet:
- Click on pending task to edit its config
- Changes are staged and applied when execution reaches that task
- Visual indicator for modified tasks

### 4.2 Add/Remove Tasks

While paused:
- **Add task**: Insert new task at any position after current
- **Remove task**: Remove pending task from execution queue
- **Reorder**: Drag pending tasks to reorder

Constraints:
- Cannot modify completed tasks
- Cannot remove currently paused task
- Changes tracked as "pending modifications"

### 4.3 Task Injection

Quick inject common debug tasks:
- "Log Variables" - inject a task that logs current variables
- "Wait" - inject a sleep task
- "Checkpoint" - inject a manual pause point

## Phase 5: Multi-Frame Support

### 5.1 Frame-Aware Task Display

When test has multiple frames (concurrent, background, matrix):

Option A: **Merged view with frame labels**
```
│ [main] 1. setup_task                    ✓    │
│ [main] 2. run_background                ✓    │
│   [bg] 2.1. background_monitor          ▶    │
│ [main] 3. do_work                       ⏸    │
│   [bg] 2.2. check_status                ○    │
```

Option B: **Frame tabs with separate task lists**
```
[main ⏸] [background ▶]

│ 1. setup_task                           ✓    │
│ 2. run_background                       ✓    │
│ 3. do_work                              ⏸    │  ← current
│ 4. cleanup                              ○    │
```

Recommendation: Option B for clarity, with visual connection indicators in graph view.

### 5.2 Frame-Specific Actions

Debug controls apply to active frame:
- Continue/Step applies to selected frame
- "Continue All" button for resuming all paused frames
- Frame selector shows which frames are paused

## Implementation Order

### Sprint 1: Core Debug Mode (1-2 days)
1. Add debug mode state to BuilderPage
2. Keep user on builder page after starting debug
3. Add debug controls to BuilderToolbar
4. Add basic frame selector bar
5. Connect to SSE debug events

### Sprint 2: Execution Visualization (1-2 days)
1. Add execution state to TaskListEditor
2. Add execution state to TaskGraphEditor (basic)
3. Breakpoint toggle on task click
4. Breakpoint indicators in list/graph

### Sprint 3: Debug Inspector (1-2 days)
1. Create DebugInspector component
2. Variables tab with tree view
3. Config tab with resolved values
4. Switch right panel contextually

### Sprint 4: Bottom Panel & Polish (1 day)
1. Collapsible debug console
2. JQ expression evaluator
3. Keyboard shortcuts
4. Connection status handling

### Sprint 5: Live Modification (2-3 days)
1. Edit future task configs
2. Add/remove pending tasks
3. Task injection shortcuts
4. Pending modifications tracking

### Sprint 6: Multi-Frame (1-2 days)
1. Frame tabs in task list
2. Frame visualization in graph
3. Frame-specific controls

### Sprint 7: Remove Standalone Debug Page (0.5 day)
1. Remove `/debug/:runId` route from App.tsx
2. Delete `pages/TestDebugger.tsx`
3. Remove standalone debug components no longer needed:
   - `components/debug/ShortcutsHelp.tsx` (integrate into builder help)
   - Any other components only used by TestDebugger
4. Update BuilderToolbar to no longer redirect
5. Remove lazy import for TestDebugger
6. Clean up unused exports from `components/debug/index.ts`
7. Update any documentation/links referencing `/debug/`

## Component Changes Summary

| Component | Changes |
|-----------|---------|
| `BuilderPage` | Add debug mode state, bottom panel |
| `BuilderToolbar` | Add debug controls, hide/show based on mode |
| `TaskListEditor` | Add execution state, breakpoint indicators |
| `TaskGraphEditor` | Add execution state, visual enhancements |
| `TaskConfigEditor` | Add "pending change" mode |
| `DebugBar` (new) | Frame selector, connection status |
| `DebugInspector` (new) | Variables, config, output tabs |
| `DebugConsole` (new) | Bottom panel with REPL |

## Hooks/Stores

Reuse existing:
- `useDebugStore` - debug session state
- `useDebugEvents` - SSE connection
- `useDebugApi` - API calls
- `useDebugKeyboard` - keyboard shortcuts

New:
- `useBuilderDebug` - builder-specific debug state and actions

## Migration from Separate Debug Page

1. During development (Sprints 1-6): Keep `/debug/:runId` route for testing
2. Default flow changes: debug from builder stays in builder
3. After Sprint 6 verification: Remove standalone debug page entirely (Sprint 7)
4. No "Open in Debug Page" option - all functionality in builder

## Open Questions

1. **Graph complexity**: How to show execution in complex nested/concurrent graphs?
2. **Performance**: Real-time updates for many tasks - throttle/batch?
3. **State persistence**: Save breakpoints with test? Session storage?
4. **Mobile/small screens**: How to handle debug UI on narrow viewports?

## Success Criteria

- [ ] User can start debug session and stay on builder page
- [ ] User can see which task is running/paused in task list
- [ ] User can set breakpoints by clicking on tasks
- [ ] User can see variables when paused
- [ ] User can step through execution with keyboard shortcuts
- [ ] User can modify future tasks while paused
- [ ] Multi-frame tests show correct execution state
- [ ] Standalone `/debug/` page removed - all debug features in builder
- [ ] No dead code or unused debug components remain
