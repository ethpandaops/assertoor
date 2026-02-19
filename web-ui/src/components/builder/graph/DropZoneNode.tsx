import { memo } from 'react';
import { Handle, Position, type NodeProps } from 'reactflow';
import { useDroppable } from '@dnd-kit/core';

export interface DropZoneNodeData {
  dropId: string;  // The drop zone ID for dnd-kit
  isHorizontal?: boolean;  // For parallel lanes (add new lane)
  isInterLane?: boolean;  // For vertical separators between parallel lanes (reorder)
  laneIndex?: number;  // Which lane this is in (for parallel containers)
  parentTaskId?: string;  // Parent glue task ID
  insertIndex: number;  // Where to insert when dropped
  disabled?: boolean;  // Whether dropping is disabled
  isCleanup?: boolean;  // Whether this is in the cleanup phase
}

function DropZoneNode({ data }: NodeProps<DropZoneNodeData>) {
  const { dropId, isHorizontal, isInterLane, disabled } = data;

  const { setNodeRef, isOver } = useDroppable({
    id: dropId,
    disabled,
  });

  const isActive = isOver && !disabled;

  if (isInterLane) {
    // Vertical separator between parallel lanes (for reordering)
    return (
      <div
        ref={setNodeRef}
        className={`
          relative w-full h-full
          flex items-center justify-center
          transition-all duration-150
          ${isActive
            ? 'bg-primary-500/10'
            : ''
          }
        `}
      >
        <Handle
          type="target"
          position={Position.Left}
          className="!w-1 !h-1 !bg-transparent !border-0"
        />
        {isActive ? (
          <div className="w-0.5 h-4/5 bg-primary-500 rounded-full" />
        ) : (
          <div className="w-px h-1/3 bg-[var(--color-border)] opacity-30" />
        )}
        <Handle
          type="source"
          position={Position.Right}
          className="!w-1 !h-1 !bg-transparent !border-0"
        />
      </div>
    );
  }

  if (isHorizontal) {
    // Horizontal drop zone (for adding parallel lanes)
    return (
      <div
        ref={setNodeRef}
        className={`
          relative w-full h-full
          flex items-center justify-center
          border-2 border-dashed rounded-lg
          transition-all duration-150
          ${isActive
            ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
            : disabled
              ? 'border-transparent'
              : 'border-gray-300/50 dark:border-gray-600/50 hover:border-primary-400 hover:bg-primary-50/50 dark:hover:bg-primary-900/10'
          }
        `}
      >
        <Handle
          type="target"
          position={Position.Left}
          className="!w-1 !h-1 !bg-transparent !border-0"
        />
        <PlusIcon className={`size-4 ${isActive ? 'text-primary-500' : 'text-gray-400/30'}`} />
        <Handle
          type="source"
          position={Position.Right}
          className="!w-1 !h-1 !bg-transparent !border-0"
        />
      </div>
    );
  }

  // Vertical drop zone (default - between sequential tasks)
  // Minimal visual footprint, shows highlight only when dragging over
  return (
    <div
      ref={setNodeRef}
      className="relative w-full h-full flex items-center justify-center"
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!w-1 !h-1 !bg-transparent !border-0"
      />

      {/* Only show visual feedback when actively dragging over */}
      {isActive && (
        <div className="absolute inset-x-4 top-1/2 -translate-y-1/2 h-1 bg-primary-500 rounded-full" />
      )}

      <Handle
        type="source"
        position={Position.Bottom}
        className="!w-1 !h-1 !bg-transparent !border-0"
      />
    </div>
  );
}

function PlusIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
    </svg>
  );
}

export default memo(DropZoneNode);
