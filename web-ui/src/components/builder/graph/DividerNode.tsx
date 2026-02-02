import { memo } from 'react';
import type { NodeProps } from 'reactflow';

export interface DividerNodeData {
  orientation: 'vertical' | 'horizontal';
}

function DividerNode({ data }: NodeProps<DividerNodeData>) {
  const { orientation } = data;

  if (orientation === 'vertical') {
    return (
      <div className="h-full w-full flex items-center justify-center">
        <div className="h-full w-0.5 bg-gray-300 dark:bg-gray-600 rounded-full" />
      </div>
    );
  }

  return (
    <div className="h-full w-full flex items-center justify-center">
      <div className="w-full h-0.5 bg-gray-300 dark:bg-gray-600 rounded-full" />
    </div>
  );
}

export default memo(DividerNode);
