import { memo } from 'react';
import { Handle, Position, type NodeProps } from 'reactflow';

export interface StartEndNodeData {
  type: 'start' | 'end' | 'cleanup';
  isSelected?: boolean;
}

function StartEndNode({ data, selected }: NodeProps<StartEndNodeData>) {
  const isStart = data.type === 'start';
  const isEnd = data.type === 'end';
  const isCleanup = data.type === 'cleanup';
  const isSelected = data.isSelected || selected;

  if (isCleanup) {
    // Cleanup phase divider - horizontal pill design
    return (
      <div className="relative w-full h-full flex items-center justify-center">
        <Handle
          type="target"
          position={Position.Top}
          className="!w-2 !h-2 !bg-amber-400 !border-0 !rounded-full"
        />

        {/* Divider line with label */}
        <div className="flex items-center gap-3">
          <div className="w-6 h-0.5 bg-amber-400/60" />
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-amber-50 dark:bg-amber-950/50 border border-amber-300 dark:border-amber-700">
            <CleanupIcon className="size-3.5 text-amber-600 dark:text-amber-400" />
            <span className="text-xs font-medium text-amber-700 dark:text-amber-300 uppercase tracking-wide">
              Cleanup
            </span>
          </div>
          <div className="w-6 h-0.5 bg-amber-400/60" />
        </div>

        <Handle
          type="source"
          position={Position.Bottom}
          className="!w-2 !h-2 !bg-amber-400 !border-0 !rounded-full"
        />
      </div>
    );
  }

  if (isStart) {
    // Start node - clickable config badge
    return (
      <div className="relative w-full h-full flex items-center justify-center">
        <div
          className={`
            nodrag nopan cursor-pointer
            flex items-center gap-2 px-3 py-1.5 rounded-full
            border-2 transition-all duration-200
            ${isSelected
              ? 'bg-emerald-100 dark:bg-emerald-900/40 border-emerald-500 ring-2 ring-emerald-300/50 dark:ring-emerald-700/50'
              : 'bg-emerald-50 dark:bg-emerald-950/30 border-emerald-400 dark:border-emerald-600 hover:bg-emerald-100 dark:hover:bg-emerald-900/40 hover:border-emerald-500'
            }
          `}
          title="Click to configure test settings"
        >
          <div className={`
            p-1 rounded-full
            ${isSelected
              ? 'bg-emerald-500 text-white'
              : 'bg-emerald-400 dark:bg-emerald-600 text-white'
            }
          `}>
            <PlayIcon className="size-2.5" />
          </div>
          <span className="text-xs font-semibold text-emerald-700 dark:text-emerald-300 uppercase tracking-wide">
            Start
          </span>
          <SettingsIcon className={`size-3 ${isSelected ? 'text-emerald-600' : 'text-emerald-500 dark:text-emerald-400'}`} />
        </div>

        <Handle
          type="source"
          position={Position.Bottom}
          className="!w-2 !h-2 !bg-emerald-500 !border-0 !rounded-full"
        />
      </div>
    );
  }

  if (isEnd) {
    // End node - simple badge
    return (
      <div className="relative w-full h-full flex items-center justify-center">
        <Handle
          type="target"
          position={Position.Top}
          className="!w-2 !h-2 !bg-red-500 !border-0 !rounded-full"
        />

        <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-red-50 dark:bg-red-950/30 border-2 border-red-400 dark:border-red-600">
          <div className="p-1 rounded-full bg-red-500 text-white">
            <StopIcon className="size-2.5" />
          </div>
          <span className="text-xs font-semibold text-red-700 dark:text-red-300 uppercase tracking-wide">
            End
          </span>
        </div>

        <Handle
          type="source"
          position={Position.Bottom}
          className="!w-2 !h-2 !bg-amber-400 !border-0 !rounded-full"
        />
      </div>
    );
  }

  return null;
}

function PlayIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="currentColor" viewBox="0 0 24 24">
      <path d="M8 5v14l11-7z" />
    </svg>
  );
}

function StopIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="currentColor" viewBox="0 0 24 24">
      <path d="M6 6h12v12H6z" />
    </svg>
  );
}

function SettingsIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
      />
      <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
    </svg>
  );
}

function CleanupIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
      />
    </svg>
  );
}

export default memo(StartEndNode);
