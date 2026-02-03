import { memo } from 'react';
import type { NodeProps } from 'reactflow';

export interface LaneLabelNodeData {
  label: string;
  variant: 'background' | 'foreground';
}

function LaneLabelNode({ data }: NodeProps<LaneLabelNodeData>) {
  const { label, variant } = data;

  return (
    <div className={`
      flex items-center justify-center h-full
      text-xs font-semibold uppercase tracking-wide
      ${variant === 'background'
        ? 'text-amber-600 dark:text-amber-400'
        : 'text-emerald-600 dark:text-emerald-400'
      }
    `}>
      {label}
    </div>
  );
}

export default memo(LaneLabelNode);
