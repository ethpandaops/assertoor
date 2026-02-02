import { memo } from 'react';
import { Handle, Position, type NodeProps } from 'reactflow';
import { useDraggable } from '@dnd-kit/core';
import type { BuilderTask } from '../../../stores/builderStore';
import type { TaskDescriptor } from '../../../types/api';

export interface GlueNodeData {
  task: BuilderTask;
  descriptor?: TaskDescriptor;
  isSelected: boolean;
  childCount: number;
  isConcurrent: boolean;
  containerWidth?: number;
  containerHeight?: number;
  isCleanup?: boolean;  // Whether this is a cleanup task
  roleLabel?: 'background' | 'foreground';  // For run_task_background children
}

function GlueTaskNode({ data, selected }: NodeProps<GlueNodeData>) {
  const { task, isSelected, isConcurrent, containerWidth, containerHeight, roleLabel } = data;

  // Make the node draggable via dnd-kit
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: task.id,
  });

  // Get display name for glue task type
  const getGlueTypeName = () => {
    switch (task.taskType) {
      case 'run_tasks':
        return 'Sequential';
      case 'run_tasks_concurrent':
        return 'Parallel';
      case 'run_task_matrix':
        return 'Matrix';
      case 'run_task_options':
        return 'Options';
      case 'run_task_background':
        return 'Background';
      default:
        return task.taskType;
    }
  };

  const getGlueTypeIcon = () => {
    if (isConcurrent) {
      return <ParallelIcon className="size-4" />;
    }
    return <SequentialIcon className="size-4" />;
  };

  return (
    <>
      {/* Input handle */}
      <Handle
        type="target"
        position={Position.Top}
        className="!w-2 !h-2 !bg-primary-500 !border-0 !rounded-full"
      />

      {/* Container node - transparent background to show children */}
      <div
        className={`
          rounded-lg border-2
          transition-all duration-200 overflow-visible
          ${isDragging ? 'opacity-50' : ''}
          ${isSelected || selected
            ? 'border-primary-500 ring-2 ring-primary-300 ring-offset-2 ring-offset-[var(--color-bg-primary)]'
            : isConcurrent
              ? 'border-purple-400 dark:border-purple-600'
              : 'border-blue-400 dark:border-blue-600'
          }
        `}
        style={{
          width: containerWidth || 260,
          height: containerHeight || 100,
          background: isConcurrent
            ? 'rgba(147, 51, 234, 0.05)'
            : 'rgba(59, 130, 246, 0.05)',
        }}
      >
        {/* Header - Drag handle */}
        <div
          ref={setNodeRef}
          {...attributes}
          {...listeners}
          className={`
            nodrag nopan flex items-center gap-2 px-3 py-2 rounded-t-md border-b cursor-grab active:cursor-grabbing
            ${isConcurrent
              ? 'bg-purple-100/80 dark:bg-purple-900/40 border-purple-200 dark:border-purple-800'
              : 'bg-blue-100/80 dark:bg-blue-900/40 border-blue-200 dark:border-blue-800'
            }
          `}
        >
          {/* Drag indicator */}
          <DragIcon className={`size-4 shrink-0 ${isConcurrent ? 'text-purple-500' : 'text-blue-500'}`} />

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

          <span className={isConcurrent ? 'text-purple-600' : 'text-blue-600'}>
            {getGlueTypeIcon()}
          </span>
          <span className={`text-xs font-semibold ${isConcurrent ? 'text-purple-700 dark:text-purple-300' : 'text-blue-700 dark:text-blue-300'}`}>
            {getGlueTypeName()}
          </span>
          {task.title && (
            <span className="text-xs text-[var(--color-text-secondary)] truncate ml-1">
              - {task.title}
            </span>
          )}
          {task.taskId && (
            <span className="text-xs text-[var(--color-text-tertiary)] font-mono ml-auto shrink-0">
              #{task.taskId}
            </span>
          )}
        </div>

        {/* Content area is empty - children are rendered as separate nodes */}

        {/* Indicators in bottom left */}
        <div className="absolute bottom-2 left-2 flex items-center gap-2">
          {task.timeout && (
            <div className="flex items-center gap-1 text-xs text-[var(--color-text-tertiary)] bg-[var(--color-bg-primary)]/80 px-1.5 py-0.5 rounded">
              <ClockIcon className="size-3" />
              <span>{task.timeout}</span>
            </div>
          )}
          {task.ifCondition && (
            <div className="flex items-center gap-1 text-xs text-yellow-600 dark:text-yellow-400 bg-[var(--color-bg-primary)]/80 px-1.5 py-0.5 rounded">
              <ConditionIcon className="size-3" />
              <span>Conditional</span>
            </div>
          )}
        </div>
      </div>

      {/* Output handle */}
      <Handle
        type="source"
        position={Position.Bottom}
        className="!w-2 !h-2 !bg-primary-500 !border-0 !rounded-full"
      />
    </>
  );
}

// Icons
function DragIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8h16M4 16h16" />
    </svg>
  );
}

function SequentialIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 14l-7 7m0 0l-7-7m7 7V3" />
    </svg>
  );
}

function ParallelIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7h12M8 12h12M8 17h12M4 7h.01M4 12h.01M4 17h.01" />
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

export default memo(GlueTaskNode);
