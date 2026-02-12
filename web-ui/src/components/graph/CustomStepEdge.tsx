import { memo } from 'react';
import { BaseEdge, type EdgeProps } from 'reactflow';

const BORDER_RADIUS = 8;

/**
 * Custom step edge that always routes down→horizontal→down.
 * Unlike ReactFlow's smoothstep, this never routes upward,
 * even with very small vertical gaps.
 */
function CustomStepEdge({
  sourceX,
  sourceY,
  targetX,
  targetY,
  style,
  markerEnd,
  markerStart,
}: EdgeProps) {
  const dx = targetX - sourceX;
  const midY = (sourceY + targetY) / 2;

  let path: string;

  if (Math.abs(dx) < 1) {
    // Same X — straight vertical line
    path = `M ${sourceX},${sourceY} L ${targetX},${targetY}`;
  } else {
    const halfVert = (targetY - sourceY) / 2;
    const r = Math.min(BORDER_RADIUS, Math.abs(halfVert) - 1, Math.abs(dx) / 2);
    const sign = dx > 0 ? 1 : -1;

    if (r > 0.5) {
      path = [
        `M ${sourceX},${sourceY}`,
        `L ${sourceX},${midY - r}`,
        `Q ${sourceX},${midY} ${sourceX + r * sign},${midY}`,
        `L ${targetX - r * sign},${midY}`,
        `Q ${targetX},${midY} ${targetX},${midY + r}`,
        `L ${targetX},${targetY}`,
      ].join(' ');
    } else {
      path = [
        `M ${sourceX},${sourceY}`,
        `L ${sourceX},${midY}`,
        `L ${targetX},${midY}`,
        `L ${targetX},${targetY}`,
      ].join(' ');
    }
  }

  return <BaseEdge path={path} style={style} markerEnd={markerEnd} markerStart={markerStart} />;
}

export default memo(CustomStepEdge);
