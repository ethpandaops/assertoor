import { memo, useEffect, useState } from 'react';
import { Handle, Position, type NodeProps } from 'reactflow';
import type { TaskState } from '../../types/api';

export interface TaskNodeData {
  task: TaskState;
  isSelected: boolean;
  onSelect: (index: number) => void;
}

function TaskGraphNode({ data, selected }: NodeProps<TaskNodeData>) {
  const { task, isSelected, onSelect } = data;
  const [now, setNow] = useState(Date.now());

  // Update timer for running tasks
  useEffect(() => {
    if (task.status !== 'running') return;

    const interval = setInterval(() => {
      setNow(Date.now());
    }, 1000);

    return () => clearInterval(interval);
  }, [task.status]);

  // Determine display status
  const getDisplayStatus = (): 'pending' | 'running' | 'success' | 'failure' | 'skipped' => {
    if (task.status === 'running') return 'running';
    if (task.status === 'complete') {
      if (task.result === 'success') return 'success';
      if (task.result === 'failure') return 'failure';
      // result === 'none' typically means skipped
      if (!task.started) return 'skipped';
    }
    return 'pending';
  };

  const displayStatus = getDisplayStatus();
  const isRunning = task.status === 'running';
  const hasProgress = task.progress > 0 && task.progress < 100;

  // Calculate runtime
  const getRuntimeMs = (): number => {
    if (task.status === 'running' && task.start_time > 0) {
      return now - task.start_time;
    }
    return task.runtime || 0;
  };

  const runtimeMs = getRuntimeMs();

  // Status colors
  const statusColors = {
    pending: {
      border: 'border-gray-300 dark:border-gray-600',
      bg: 'bg-gray-50 dark:bg-gray-800',
      dot: 'bg-gray-400',
    },
    running: {
      border: 'border-blue-400 dark:border-blue-500',
      bg: 'bg-blue-50 dark:bg-blue-900/20',
      dot: 'bg-blue-500',
    },
    success: {
      border: 'border-green-400 dark:border-green-500',
      bg: 'bg-green-50 dark:bg-green-900/20',
      dot: 'bg-green-500',
    },
    failure: {
      border: 'border-red-400 dark:border-red-500',
      bg: 'bg-red-50 dark:bg-red-900/20',
      dot: 'bg-red-500',
    },
    skipped: {
      border: 'border-yellow-400 dark:border-yellow-500',
      bg: 'bg-yellow-50 dark:bg-yellow-900/20',
      dot: 'bg-yellow-500',
    },
  };

  const colors = statusColors[displayStatus];

  return (
    <>
      {/* Input handle (top) */}
      <Handle
        type="target"
        position={Position.Top}
        className="!w-2 !h-2 !bg-gray-400 dark:!bg-gray-500 !border-2 !border-[var(--color-bg-primary)]"
      />

      {/* Node content */}
      <div
        onClick={() => onSelect(task.index)}
        className={`
          w-[180px] rounded-lg border-2 cursor-pointer
          transition-all duration-200 shadow-sm hover:shadow-md
          ${colors.border} ${colors.bg}
          ${isSelected || selected ? 'ring-2 ring-primary-500 ring-offset-2 ring-offset-[var(--color-bg-primary)]' : ''}
        `}
      >
        {/* Header with status indicator */}
        <div className="flex items-center gap-2 px-2.5 py-1.5 border-b border-[var(--color-border)]">
          <span
            className={`shrink-0 size-2 rounded-full ${colors.dot} ${
              isRunning ? 'animate-pulse' : ''
            }`}
          />
          <span className="text-[10px] text-[var(--color-text-tertiary)] font-mono">
            #{task.index}
          </span>
          {runtimeMs > 0 && (
            <span className="text-[10px] text-[var(--color-text-tertiary)] font-mono ml-auto">
              {formatRuntimeCompact(runtimeMs)}
            </span>
          )}
        </div>

        {/* Task info */}
        <div className="px-2.5 py-2">
          <div className="text-xs font-medium truncate" title={task.title || task.name}>
            {task.title || task.name}
          </div>
          {task.title && task.title !== task.name && (
            <div
              className="text-[10px] text-[var(--color-text-secondary)] truncate mt-0.5"
              title={task.name}
            >
              {task.name}
            </div>
          )}

          {/* Progress bar for running tasks */}
          {isRunning && hasProgress && (
            <div className="mt-2">
              <div className="flex items-center gap-2">
                <div className="flex-1 h-1 bg-[var(--color-bg-tertiary)] rounded-full overflow-hidden">
                  <div
                    className="h-full bg-blue-500 transition-all duration-300"
                    style={{ width: `${task.progress}%` }}
                  />
                </div>
                <span className="text-[10px] text-[var(--color-text-tertiary)] font-mono shrink-0">
                  {Math.round(task.progress)}%
                </span>
              </div>
              {task.progress_message && (
                <div className="text-[10px] text-[var(--color-text-tertiary)] truncate mt-1">
                  {task.progress_message}
                </div>
              )}
            </div>
          )}

          {/* Error message for failed tasks */}
          {displayStatus === 'failure' && task.result_error && (
            <div className="mt-2 text-[10px] text-red-600 dark:text-red-400 truncate" title={task.result_error}>
              {task.result_error}
            </div>
          )}
        </div>
      </div>

      {/* Output handle (bottom) */}
      <Handle
        type="source"
        position={Position.Bottom}
        className="!w-2 !h-2 !bg-gray-400 dark:!bg-gray-500 !border-2 !border-[var(--color-bg-primary)]"
      />
    </>
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
    return `${minutes}m${remainingSeconds > 0 ? ` ${remainingSeconds}s` : ''}`;
  }

  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return `${hours}h${remainingMinutes > 0 ? ` ${remainingMinutes}m` : ''}`;
}

export default memo(TaskGraphNode);
