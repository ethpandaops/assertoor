import { useMemo, useState, useEffect, useCallback } from 'react';
import type { TaskState } from '../../types/api';
import StatusBadge from '../common/StatusBadge';

interface TaskListProps {
  tasks: TaskState[];
  selectedIndex: number | null;
  onSelect: (index: number) => void;
}

interface ProcessedTask extends TaskState {
  depth: number;
  hasChildren: boolean;
  childCount: number;
}

function TaskList({ tasks, selectedIndex, onSelect }: TaskListProps) {
  const [collapsedTasks, setCollapsedTasks] = useState<Set<number>>(new Set());

  // Build tree structure with hierarchical ordering
  const { processedTasks, childrenMap } = useMemo(() => {
    // Build a map of parent -> children
    const childrenMap = new Map<number, TaskState[]>();
    const taskMap = new Map<number, TaskState>();
    const rootTasks: TaskState[] = [];

    // First pass: index all tasks and find roots
    for (const task of tasks) {
      taskMap.set(task.index, task);
      const parentIndex = task.parent_index;

      if (parentIndex < 0 || parentIndex === task.index || !tasks.some((t) => t.index === parentIndex)) {
        // Root task (no parent, or parent doesn't exist)
        rootTasks.push(task);
      } else {
        // Child task
        const siblings = childrenMap.get(parentIndex) || [];
        siblings.push(task);
        childrenMap.set(parentIndex, siblings);
      }
    }

    // Sort root tasks by index
    rootTasks.sort((a, b) => a.index - b.index);

    // Sort all children arrays by index
    childrenMap.forEach((children) => {
      children.sort((a, b) => a.index - b.index);
    });

    // Count all descendants for each task
    const countDescendants = (taskIndex: number): number => {
      const children = childrenMap.get(taskIndex) || [];
      let count = children.length;
      for (const child of children) {
        count += countDescendants(child.index);
      }
      return count;
    };

    // Flatten the tree with DFS traversal to get hierarchical order
    const result: ProcessedTask[] = [];

    const traverse = (task: TaskState, depth: number) => {
      const children = childrenMap.get(task.index) || [];
      const childCount = countDescendants(task.index);
      result.push({
        ...task,
        depth,
        hasChildren: children.length > 0,
        childCount,
      });
      for (const child of children) {
        traverse(child, depth + 1);
      }
    };

    for (const rootTask of rootTasks) {
      traverse(rootTask, 0);
    }

    return { processedTasks: result, childrenMap };
  }, [tasks]);

  // Filter out tasks whose ancestors are collapsed
  const visibleTasks = useMemo(() => {
    if (collapsedTasks.size === 0) return processedTasks;

    // Build a set of all hidden task indices (children of collapsed tasks)
    const hiddenTasks = new Set<number>();

    const hideDescendants = (taskIndex: number) => {
      const children = childrenMap.get(taskIndex) || [];
      for (const child of children) {
        hiddenTasks.add(child.index);
        hideDescendants(child.index);
      }
    };

    for (const collapsedIndex of collapsedTasks) {
      hideDescendants(collapsedIndex);
    }

    return processedTasks.filter((task) => !hiddenTasks.has(task.index));
  }, [processedTasks, collapsedTasks, childrenMap]);

  const toggleCollapse = useCallback((taskIndex: number, e: React.MouseEvent) => {
    e.stopPropagation();
    setCollapsedTasks((prev) => {
      const next = new Set(prev);
      if (next.has(taskIndex)) {
        next.delete(taskIndex);
      } else {
        next.add(taskIndex);
      }
      return next;
    });
  }, []);

  const collapseAll = useCallback(() => {
    const tasksWithChildren = processedTasks
      .filter((t) => t.hasChildren)
      .map((t) => t.index);
    setCollapsedTasks(new Set(tasksWithChildren));
  }, [processedTasks]);

  const expandAll = useCallback(() => {
    setCollapsedTasks(new Set());
  }, []);

  const hasCollapsibleTasks = processedTasks.some((t) => t.hasChildren);

  return (
    <div className="flex flex-col h-full">
      {/* Collapse/Expand controls */}
      {hasCollapsibleTasks && (
        <div className="flex items-center gap-2 px-2 py-1.5 border-b border-[var(--color-border)] bg-[var(--color-bg-secondary)]">
          <button
            onClick={expandAll}
            className="text-[10px] text-[var(--color-text-secondary)] hover:text-primary-600 transition-colors"
          >
            Expand All
          </button>
          <span className="text-[var(--color-text-tertiary)]">|</span>
          <button
            onClick={collapseAll}
            className="text-[10px] text-[var(--color-text-secondary)] hover:text-primary-600 transition-colors"
          >
            Collapse All
          </button>
        </div>
      )}

      {/* Task list */}
      <div className="divide-y divide-[var(--color-border)] flex-1 overflow-y-auto">
        {visibleTasks.map((task) => (
          <TaskListItem
            key={task.index}
            task={task}
            depth={task.depth}
            isSelected={task.index === selectedIndex}
            isCollapsed={collapsedTasks.has(task.index)}
            onClick={() => onSelect(task.index)}
            onToggleCollapse={(e) => toggleCollapse(task.index, e)}
          />
        ))}
      </div>
    </div>
  );
}

interface TaskListItemProps {
  task: ProcessedTask;
  depth: number;
  isSelected: boolean;
  isCollapsed: boolean;
  onClick: () => void;
  onToggleCollapse: (e: React.MouseEvent) => void;
}

function TaskListItem({ task, depth, isSelected, isCollapsed, onClick, onToggleCollapse }: TaskListItemProps) {
  // Determine display status based on task state
  const getDisplayStatus = (): 'pending' | 'running' | 'success' | 'failure' => {
    if (task.status === 'running') return 'running';
    if (task.status === 'complete') {
      if (task.result === 'success') return 'success';
      if (task.result === 'failure') return 'failure';
    }
    return 'pending';
  };

  const displayStatus = getDisplayStatus();
  const isRunning = task.status === 'running';
  const hasProgress = task.progress > 0 && task.progress < 100;

  return (
    <button
      onClick={onClick}
      className={`w-full text-left px-2 py-1.5 hover:bg-[var(--color-bg-tertiary)] transition-colors ${
        isSelected ? 'bg-primary-50 dark:bg-primary-900/20' : ''
      }`}
      style={{ paddingLeft: `${0.5 + depth * 1}rem` }}
    >
      <div className="flex items-center justify-between gap-2">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-1">
            {/* Collapse/expand toggle */}
            {task.hasChildren ? (
              <span
                onClick={onToggleCollapse}
                className="shrink-0 p-0.5 -ml-0.5 hover:bg-[var(--color-bg-tertiary)] rounded-xs cursor-pointer"
                title={isCollapsed ? `Expand (${task.childCount} children)` : 'Collapse'}
              >
                <ChevronIcon
                  className={`size-3 text-[var(--color-text-tertiary)] transition-transform ${
                    isCollapsed ? '' : 'rotate-90'
                  }`}
                />
              </span>
            ) : (
              <span className="shrink-0 w-4" /> // Spacer for alignment
            )}
            <span className="text-[10px] text-[var(--color-text-tertiary)] shrink-0">#{task.index}</span>
            <span className="text-xs font-medium truncate">{task.title || task.name}</span>
            {/* Show child count when collapsed */}
            {isCollapsed && task.childCount > 0 && (
              <span className="text-[10px] text-[var(--color-text-tertiary)] shrink-0">
                (+{task.childCount})
              </span>
            )}
          </div>
          {task.title && task.title !== task.name && (
            <div className="text-[10px] text-[var(--color-text-secondary)] truncate ml-6">{task.name}</div>
          )}
          {/* Progress bar for running tasks with progress */}
          {isRunning && hasProgress && (
            <div className="mt-1 ml-6">
              <div className="flex items-center gap-2">
                <div className="flex-1 h-1 bg-[var(--color-bg-tertiary)] rounded-full overflow-hidden">
                  <div
                    className="h-full bg-primary-500 transition-all duration-300"
                    style={{ width: `${task.progress}%` }}
                  />
                </div>
                <span className="text-[10px] text-[var(--color-text-tertiary)] shrink-0">
                  {Math.round(task.progress)}%
                </span>
              </div>
              {task.progress_message && (
                <div className="text-[10px] text-[var(--color-text-tertiary)] truncate mt-0.5">
                  {task.progress_message}
                </div>
              )}
            </div>
          )}
        </div>
        <div className="shrink-0 flex flex-col items-end gap-0.5">
          <StatusBadge status={displayStatus} size="sm" />
          <TaskRuntime task={task} />
        </div>
      </div>
    </button>
  );
}

function ChevronIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
    </svg>
  );
}

interface TaskRuntimeProps {
  task: TaskState;
}

function TaskRuntime({ task }: TaskRuntimeProps) {
  const [now, setNow] = useState(Date.now());

  // Update every second for running tasks
  useEffect(() => {
    if (task.status !== 'running') return;

    const interval = setInterval(() => {
      setNow(Date.now());
    }, 1000);

    return () => clearInterval(interval);
  }, [task.status]);

  // Calculate runtime
  const getRuntimeMs = (): number => {
    if (task.status === 'running' && task.start_time > 0) {
      // For running tasks, calculate from start_time (in ms) to now
      return now - task.start_time;
    }
    // For completed tasks, use the pre-calculated runtime (already in ms)
    return task.runtime || 0;
  };

  const runtimeMs = getRuntimeMs();

  // Don't show if no runtime
  if (runtimeMs <= 0 && task.status === 'pending') {
    return null;
  }

  return (
    <span className="text-[10px] text-[var(--color-text-tertiary)] font-mono">
      {formatRuntimeCompact(runtimeMs)}
    </span>
  );
}

function formatRuntimeCompact(ms: number): string {
  if (ms < 1000) {
    return `${ms}ms`;
  }

  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) {
    return `${seconds}s`;
  }

  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  if (minutes < 60) {
    return `${minutes}m ${remainingSeconds}s`;
  }

  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return `${hours}h ${remainingMinutes}m`;
}

export default TaskList;
