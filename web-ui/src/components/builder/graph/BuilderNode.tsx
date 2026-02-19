import { memo } from 'react';
import { Handle, Position, type NodeProps } from 'reactflow';
import { useDraggable } from '@dnd-kit/core';
import type { BuilderTask } from '../../../stores/builderStore';
import type { TaskDescriptor } from '../../../types/api';

export interface BuilderNodeData {
  task: BuilderTask;
  descriptor?: TaskDescriptor;
  isSelected: boolean;
  roleLabel?: 'background' | 'foreground';  // For run_task_background children
  isCleanup?: boolean;  // Whether this is a cleanup task
}

function BuilderNode({ data, selected }: NodeProps<BuilderNodeData>) {
  const { task, descriptor, isSelected, roleLabel } = data;

  // Make the node draggable via dnd-kit
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: task.id,
  });

  return (
    <>
      {/* Input handle */}
      <Handle
        type="target"
        position={Position.Top}
        className="!w-2 !h-2 !bg-gray-400 dark:!bg-gray-500 !border-0 !rounded-full"
      />

      {/* Node content */}
      <div
        className={`
          rounded-lg border-2 bg-[var(--color-bg-primary)]
          transition-all duration-200 shadow-sm
          ${isDragging ? 'opacity-50' : 'hover:shadow-md'}
          ${isSelected || selected
            ? 'border-primary-500 ring-2 ring-primary-300 ring-offset-2 ring-offset-[var(--color-bg-primary)]'
            : 'border-[var(--color-border)] hover:border-primary-400'
          }
        `}
        style={{ width: '100%', height: '100%' }}
      >
        {/* Header - Drag handle */}
        <div
          ref={setNodeRef}
          {...attributes}
          {...listeners}
          className="nodrag nopan flex items-center gap-2 px-3 py-1.5 border-b border-[var(--color-border)] bg-[var(--color-bg-secondary)] rounded-t-lg cursor-grab active:cursor-grabbing"
        >
          {/* Drag indicator */}
          <DragIcon className="size-3.5 text-[var(--color-text-tertiary)] shrink-0" />

          {/* Role label for run_task_background children */}
          {roleLabel && (
            <span className={`px-1.5 py-0.5 text-xs font-medium rounded-sm shrink-0 ${
              roleLabel === 'background'
                ? 'bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300'
                : 'bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-300'
            }`}>
              {roleLabel === 'background' ? 'BG' : 'FG'}
            </span>
          )}
          <TaskIcon className="size-3.5 text-[var(--color-text-tertiary)] shrink-0" />
          <span className="text-xs font-medium truncate text-[var(--color-text-secondary)]">
            {task.taskType}
          </span>
          {task.taskId && (
            <span className="text-xs text-[var(--color-text-tertiary)] font-mono ml-auto shrink-0">
              #{task.taskId}
            </span>
          )}
        </div>

        {/* Body */}
        <div className="px-3 py-2">
          <div className="text-sm font-medium truncate" title={task.title || task.taskType}>
            {task.title || descriptor?.name || task.taskType}
          </div>

          {/* Indicators row */}
          <div className="flex items-center gap-2 mt-1.5 flex-wrap">
            {task.timeout && (
              <div className="flex items-center gap-1 text-xs text-[var(--color-text-tertiary)]">
                <ClockIcon className="size-3" />
                <span>{task.timeout}</span>
              </div>
            )}
            {task.ifCondition && (
              <div className="flex items-center gap-1 text-xs text-yellow-600 dark:text-yellow-400">
                <ConditionIcon className="size-3" />
                <span>Conditional</span>
              </div>
            )}
            {Object.keys(task.config).length > 0 && (
              <div className="text-xs text-[var(--color-text-tertiary)]">
                {Object.keys(task.config).length} config
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Output handle */}
      <Handle
        type="source"
        position={Position.Bottom}
        className="!w-2 !h-2 !bg-gray-400 dark:!bg-gray-500 !border-0 !rounded-full"
      />
    </>
  );
}

function DragIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8h16M4 16h16" />
    </svg>
  );
}

function TaskIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"
      />
    </svg>
  );
}

function ClockIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
      />
    </svg>
  );
}

function ConditionIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
      />
    </svg>
  );
}

export default memo(BuilderNode);
