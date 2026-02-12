import { useMemo } from 'react';
import type { Node, Edge } from 'reactflow';
import type { BuilderTask } from '../../../stores/builderStore';
import { getNamedChildSlots, hasNamedChildren } from '../../../stores/builderStore';
import type { BuilderNodeData } from './BuilderNode';
import type { GlueNodeData } from './GlueTaskNode';
import type { StartEndNodeData } from './StartEndNode';
import type { DropZoneNodeData } from './DropZoneNode';
import type { LaneLabelNodeData } from './LaneLabelNode';
import type { DividerNodeData } from './DividerNode';
import type { TaskDescriptor } from '../../../types/api';
import { canHaveChildren } from '../../../utils/builder/taskUtils';

// Layout constants
const NODE_WIDTH = 220;
const NODE_HEIGHT = 90;
const DROP_ZONE_HEIGHT = 24;
const VERTICAL_GAP = 12;
const LANE_GAP = 20;
const CONTAINER_PADDING_X = 20;
const CONTAINER_PADDING_TOP = 50;
const CONTAINER_PADDING_BOTTOM = 20;
const START_END_WIDTH = 140;
const START_END_HEIGHT = 40;
const EMPTY_LANE_WIDTH = 50;
const PHASE_DIVIDER_WIDTH = 220;
const PHASE_DIVIDER_HEIGHT = 48;
const BG_TASK_DIVIDER_WIDTH = 2;
const BG_TASK_LABEL_HEIGHT = 20;

// Concurrent glue task types (render children as parallel lanes)
const CONCURRENT_GLUE_TASKS = new Set([
  'run_tasks_concurrent',
  'run_task_matrix',
  'run_task_background',
]);

type AnyNodeData = BuilderNodeData | GlueNodeData | StartEndNodeData | DropZoneNodeData | LaneLabelNodeData | DividerNodeData;

interface LayoutResult {
  nodes: Node<AnyNodeData>[];
  edges: Edge[];
}

interface SizeResult {
  width: number;
  height: number;
}

// Get max children for a task type
function getMaxChildren(taskType: string): number {
  if (hasNamedChildren(taskType)) {
    const slots = getNamedChildSlots(taskType);
    return slots?.length || 0;
  }
  switch (taskType) {
    case 'run_task_options':
    case 'run_task_matrix':
      return 1;
    default:
      return Infinity;
  }
}

// Calculate size needed for a task and its children (recursive)
function calculateTaskSize(task: BuilderTask): SizeResult {
  const isGlue = canHaveChildren(task.taskType);

  if (!isGlue) {
    // Regular task - fixed size
    return { width: NODE_WIDTH, height: NODE_HEIGHT };
  }

  const children = task.children || [];
  const isConcurrent = CONCURRENT_GLUE_TASKS.has(task.taskType);

  // Special handling for tasks with named children (e.g., run_task_background)
  const namedSlots = getNamedChildSlots(task.taskType);
  if (namedSlots) {
    // Calculate size for each slot
    const slotSizes = namedSlots.map((slot) => {
      const slotChild = task.namedChildren?.[slot.name];
      return slotChild ? calculateTaskSize(slotChild) : { width: NODE_WIDTH, height: NODE_HEIGHT };
    });

    const maxChildHeight = Math.max(...slotSizes.map((s) => s.height));

    // Content height = label + drop zone + child + drop zone
    const laneContentHeight = BG_TASK_LABEL_HEIGHT + DROP_ZONE_HEIGHT + VERTICAL_GAP + maxChildHeight + VERTICAL_GAP + DROP_ZONE_HEIGHT;

    // Width = padding + (slot widths + gaps + dividers)
    const totalSlotWidth = slotSizes.reduce((sum, s) => sum + s.width, 0);
    const dividerCount = namedSlots.length - 1;
    const totalWidth = CONTAINER_PADDING_X * 2 + totalSlotWidth + (namedSlots.length - 1) * LANE_GAP + dividerCount * (BG_TASK_DIVIDER_WIDTH + LANE_GAP);

    return {
      width: totalWidth,
      height: CONTAINER_PADDING_TOP + laneContentHeight + CONTAINER_PADDING_BOTTOM,
    };
  }

  const maxChildren = getMaxChildren(task.taskType);
  const canAddMore = children.length < maxChildren;

  if (children.length === 0) {
    // Empty glue task - show minimum size with one drop zone
    const minWidth = NODE_WIDTH + CONTAINER_PADDING_X * 2;
    const contentHeight = DROP_ZONE_HEIGHT + VERTICAL_GAP;
    return {
      width: minWidth,
      height: CONTAINER_PADDING_TOP + contentHeight + CONTAINER_PADDING_BOTTOM,
    };
  }

  // Calculate sizes for all children first
  const childSizes = children.map((child) => calculateTaskSize(child));

  if (isConcurrent) {
    // Concurrent: children in parallel lanes side by side
    const maxChildHeight = Math.max(...childSizes.map((s) => s.height));
    const totalChildWidth = childSizes.reduce((sum, s) => sum + s.width, 0);

    // Lane content height = just child height (inter-lane gaps handle reordering)
    const laneContentHeight = maxChildHeight;

    // Total width = inter-lane gaps (one before each lane) + children + empty lane
    const lanesWidth = children.length * LANE_GAP + totalChildWidth + (canAddMore ? LANE_GAP + EMPTY_LANE_WIDTH : 0);

    return {
      width: CONTAINER_PADDING_X * 2 + Math.max(NODE_WIDTH, lanesWidth),
      height: CONTAINER_PADDING_TOP + laneContentHeight + CONTAINER_PADDING_BOTTOM,
    };
  } else {
    // Sequential: children stacked vertically
    const maxChildWidth = Math.max(...childSizes.map((s) => s.width));

    // Content height = initial drop zone + (child + gap + drop zone) for each
    let contentHeight = DROP_ZONE_HEIGHT + VERTICAL_GAP;
    for (const size of childSizes) {
      contentHeight += size.height + VERTICAL_GAP + DROP_ZONE_HEIGHT + VERTICAL_GAP;
    }

    return {
      width: CONTAINER_PADDING_X * 2 + Math.max(NODE_WIDTH, maxChildWidth),
      height: CONTAINER_PADDING_TOP + contentHeight + CONTAINER_PADDING_BOTTOM,
    };
  }
}

export function useBuilderGraphLayout(
  tasks: BuilderTask[],
  cleanupTasks: BuilderTask[],
  descriptorMap: Map<string, TaskDescriptor>,
  selectedTaskId: string | null
): LayoutResult {
  return useMemo(() => {
    const nodes: Node<AnyNodeData>[] = [];
    const edges: Edge[] = [];
    let nodeIdCounter = 0;

    const generateId = (prefix: string) => `${prefix}-${nodeIdCounter++}`;

    // Create edge between two nodes
    const addEdge = (sourceId: string, targetId: string, color?: string, dashed?: boolean) => {
      edges.push({
        id: `edge-${sourceId}-${targetId}`,
        source: sourceId,
        target: targetId,
        type: 'smoothstep',
        style: {
          stroke: color || 'var(--color-border)',
          strokeWidth: 2,
          strokeDasharray: dashed ? '5 5' : undefined,
        },
      });
    };

    // Process a single task and return its node ID
    function processTask(
      task: BuilderTask,
      x: number,
      y: number,
      parentNodeId: string | undefined,
      roleLabel?: 'background' | 'foreground',
      isCleanup?: boolean
    ): { nodeId: string; size: SizeResult } {
      const isGlue = canHaveChildren(task.taskType);
      const isConcurrent = CONCURRENT_GLUE_TASKS.has(task.taskType);
      const nodeId = `node-${task.id}`;
      const descriptor = descriptorMap.get(task.taskType);
      const size = calculateTaskSize(task);
      const children = task.children || [];
      const childCount = children.length;
      const maxChildren = getMaxChildren(task.taskType);
      const canAddMore = childCount < maxChildren;

      if (isGlue) {
        // Create glue container node
        nodes.push({
          id: nodeId,
          type: 'glueNode',
          position: { x, y },
          parentNode: parentNodeId,
          extent: parentNodeId ? 'parent' : undefined,
          draggable: false,
          data: {
            task,
            descriptor,
            isSelected: task.id === selectedTaskId,
            childCount,
            isConcurrent,
            containerWidth: size.width,
            containerHeight: size.height,
            isCleanup,
            roleLabel,
          } as GlueNodeData,
          selected: task.id === selectedTaskId,
          style: { width: size.width, height: size.height },
        });

        const edgeColor = isConcurrent ? 'var(--color-purple-400)' : 'var(--color-blue-400)';

        // Special handling for tasks with named children (e.g., run_task_background)
        const taskNamedSlots = getNamedChildSlots(task.taskType);
        if (taskNamedSlots) {
          // Calculate sizes for each slot
          const slotSizes = taskNamedSlots.map((slot) => {
            const slotChild = task.namedChildren?.[slot.name];
            return slotChild ? calculateTaskSize(slotChild) : { width: NODE_WIDTH, height: NODE_HEIGHT };
          });
          const maxChildHeight = Math.max(...slotSizes.map((s) => s.height));

          const laneTopY = CONTAINER_PADDING_TOP;
          let currentX = CONTAINER_PADDING_X;

          taskNamedSlots.forEach((slot, slotIndex) => {
            const slotChild = task.namedChildren?.[slot.name];
            const slotSize = slotSizes[slotIndex];

            // Slot label
            const labelId = generateId('label');
            nodes.push({
              id: labelId,
              type: 'laneLabel',
              position: { x: currentX, y: laneTopY },
              parentNode: nodeId,
              extent: 'parent',
              draggable: false,
              data: {
                label: slot.name.charAt(0).toUpperCase() + slot.name.slice(1),
                variant: slot.name as 'background' | 'foreground',
              } as LaneLabelNodeData,
              style: { width: slotSize.width, height: BG_TASK_LABEL_HEIGHT },
            });

            // Slot child or empty placeholder
            const slotChildY = laneTopY + BG_TASK_LABEL_HEIGHT + DROP_ZONE_HEIGHT + VERTICAL_GAP;
            if (slotChild) {
              processTask(slotChild, currentX, slotChildY, nodeId, undefined, isCleanup);
            } else {
              const emptyDropId = generateId('drop');
              const emptyHeight = DROP_ZONE_HEIGHT + VERTICAL_GAP + maxChildHeight + VERTICAL_GAP + DROP_ZONE_HEIGHT;
              nodes.push({
                id: emptyDropId,
                type: 'dropZone',
                position: { x: currentX, y: laneTopY + BG_TASK_LABEL_HEIGHT },
                parentNode: nodeId,
                extent: 'parent',
                draggable: false,
                data: {
                  dropId: isCleanup ? `cleanup-insert-named-${slot.name}-${task.id}` : `insert-named-${slot.name}-${task.id}`,
                  isHorizontal: true,
                  parentTaskId: task.id,
                  insertIndex: slotIndex,
                  isCleanup,
                } as DropZoneNodeData,
                style: { width: slotSize.width, height: emptyHeight },
              });
            }

            currentX += slotSize.width + LANE_GAP;

            // Add divider between slots (not after the last one)
            if (slotIndex < taskNamedSlots.length - 1) {
              const dividerHeight = BG_TASK_LABEL_HEIGHT + DROP_ZONE_HEIGHT + VERTICAL_GAP + maxChildHeight + VERTICAL_GAP + DROP_ZONE_HEIGHT;
              const dividerId = generateId('divider');
              nodes.push({
                id: dividerId,
                type: 'divider',
                position: { x: currentX, y: laneTopY },
                parentNode: nodeId,
                extent: 'parent',
                draggable: false,
                data: {
                  orientation: 'vertical',
                } as DividerNodeData,
                style: { width: BG_TASK_DIVIDER_WIDTH, height: dividerHeight },
              });
              currentX += BG_TASK_DIVIDER_WIDTH + LANE_GAP;
            }
          });
        } else if (isConcurrent && childCount > 0) {
          // Other concurrent tasks: render parallel lanes with inter-lane drop zones
          const childSizes = children.map((child) => calculateTaskSize(child));
          const maxChildHeight = Math.max(...childSizes.map((s) => s.height));

          let laneX = CONTAINER_PADDING_X;
          const laneTopY = CONTAINER_PADDING_TOP;
          const gapHeight = maxChildHeight;

          for (let i = 0; i < childCount; i++) {
            const child = children[i];
            const childSize = childSizes[i];

            // Inter-lane drop zone before this lane (for reordering and inserting)
            const gapDropId = generateId('drop');
            nodes.push({
              id: gapDropId,
              type: 'dropZone',
              position: { x: laneX, y: laneTopY },
              parentNode: nodeId,
              extent: 'parent',
              draggable: false,
              data: {
                dropId: isCleanup ? `cleanup-insert-before-${child.id}` : `insert-before-${child.id}`,
                isInterLane: true,
                parentTaskId: task.id,
                insertIndex: i,
                isCleanup,
              } as DropZoneNodeData,
              style: { width: LANE_GAP, height: gapHeight },
            });

            laneX += LANE_GAP;

            // Process child task directly in the lane
            processTask(child, laneX, laneTopY, nodeId, undefined, isCleanup);

            laneX += childSize.width;
          }

          // Add empty lane for adding new parallel task (if allowed)
          if (canAddMore) {
            laneX += LANE_GAP;
            const emptyLaneDropId = generateId('drop');
            nodes.push({
              id: emptyLaneDropId,
              type: 'dropZone',
              position: { x: laneX, y: laneTopY },
              parentNode: nodeId,
              extent: 'parent',
              draggable: false,
              data: {
                dropId: isCleanup ? `cleanup-insert-after-children-${task.id}` : `insert-after-children-${task.id}`,
                isHorizontal: true,
                parentTaskId: task.id,
                insertIndex: childCount,
                isCleanup,
              } as DropZoneNodeData,
              style: { width: EMPTY_LANE_WIDTH, height: gapHeight },
            });
          }
        } else {
          // Sequential or empty: stack children vertically with execution line
          const childSizes = children.map((child) => calculateTaskSize(child));
          const maxChildWidth = childCount > 0 ? Math.max(...childSizes.map((s) => s.width)) : NODE_WIDTH;
          const isSingleChildTask = task.taskType === 'run_task_options' || task.taskType === 'run_task_matrix';

          let currentY = CONTAINER_PADDING_TOP;
          const centerX = CONTAINER_PADDING_X + (size.width - CONTAINER_PADDING_X * 2 - maxChildWidth) / 2;

          // For single-child tasks with a filled slot, render child directly without drop zones
          if (isSingleChildTask && childCount > 0) {
            const child = children[0];
            const childSize = childSizes[0];
            const childX = centerX + (maxChildWidth - childSize.width) / 2;
            // Add some padding at top
            currentY += DROP_ZONE_HEIGHT + VERTICAL_GAP;
            processTask(child, childX, currentY, nodeId, undefined, isCleanup);
          } else if (isSingleChildTask && childCount === 0) {
            // Single-child task with empty slot - show drop zone
            const emptyDropId = generateId('drop');
            nodes.push({
              id: emptyDropId,
              type: 'dropZone',
              position: { x: centerX, y: currentY },
              parentNode: nodeId,
              extent: 'parent',
              draggable: false,
              data: {
                dropId: isCleanup ? `cleanup-insert-single-child-${task.id}` : `insert-single-child-${task.id}`,
                parentTaskId: task.id,
                insertIndex: 0,
                isCleanup,
              } as DropZoneNodeData,
              style: { width: maxChildWidth, height: DROP_ZONE_HEIGHT },
            });
          } else {
            // Multi-child tasks (run_tasks, run_tasks_concurrent): normal flow with drop zones

            // Initial drop zone
            const firstDropId = generateId('drop');
            nodes.push({
              id: firstDropId,
              type: 'dropZone',
              position: { x: centerX, y: currentY },
              parentNode: nodeId,
              extent: 'parent',
              draggable: false,
              data: {
                dropId: childCount > 0
                  ? (isCleanup ? `cleanup-insert-before-${children[0].id}` : `insert-before-${children[0].id}`)
                  : (isCleanup ? `cleanup-insert-first-child-${task.id}` : `insert-first-child-${task.id}`),
                parentTaskId: task.id,
                insertIndex: 0,
                isCleanup,
              } as DropZoneNodeData,
              style: { width: maxChildWidth, height: DROP_ZONE_HEIGHT },
            });

            let prevNodeId = firstDropId;
            currentY += DROP_ZONE_HEIGHT + VERTICAL_GAP;

            // Process each child
            for (let i = 0; i < childCount; i++) {
              const child = children[i];
              const childSize = childSizes[i];

              // Center child within container
              const childX = centerX + (maxChildWidth - childSize.width) / 2;
              const childResult = processTask(child, childX, currentY, nodeId, undefined, isCleanup);

              // Edge from previous node to child
              addEdge(prevNodeId, childResult.nodeId, edgeColor);

              currentY += childSize.height + VERTICAL_GAP;

              // Drop zone after this child
              const dropId = generateId('drop');
              nodes.push({
                id: dropId,
                type: 'dropZone',
                position: { x: centerX, y: currentY },
                parentNode: nodeId,
                extent: 'parent',
                draggable: false,
                data: {
                  dropId: i < childCount - 1
                    ? (isCleanup ? `cleanup-insert-before-${children[i + 1].id}` : `insert-before-${children[i + 1].id}`)
                    : (isCleanup ? `cleanup-insert-after-children-${task.id}` : `insert-after-children-${task.id}`),
                  parentTaskId: task.id,
                  insertIndex: i + 1,
                  isCleanup,
                } as DropZoneNodeData,
                style: { width: maxChildWidth, height: DROP_ZONE_HEIGHT },
              });

              // Edge from child to drop zone
              addEdge(childResult.nodeId, dropId, edgeColor);

              prevNodeId = dropId;
              currentY += DROP_ZONE_HEIGHT + VERTICAL_GAP;
            }
          }
        }
      } else {
        // Regular task node
        nodes.push({
          id: nodeId,
          type: 'builderNode',
          position: { x, y },
          parentNode: parentNodeId,
          extent: parentNodeId ? 'parent' : undefined,
          draggable: false,
          data: {
            task,
            descriptor,
            isSelected: task.id === selectedTaskId,
            roleLabel,
            isCleanup,
          } as BuilderNodeData,
          selected: task.id === selectedTaskId,
          style: { width: size.width, height: size.height },
        });
      }

      return { nodeId, size };
    }

    // Calculate sizes for all tasks
    const taskSizes = tasks.map((task) => calculateTaskSize(task));
    const cleanupTaskSizes = cleanupTasks.map((task) => calculateTaskSize(task));

    // Calculate max width across all tasks
    const allWidths = [...taskSizes.map((s) => s.width), ...cleanupTaskSizes.map((s) => s.width)];
    const maxRootWidth = allWidths.length > 0 ? Math.max(...allWidths, NODE_WIDTH) : NODE_WIDTH;

    // Add start node (clickable for test config)
    const startNodeId = 'start-node';
    nodes.push({
      id: startNodeId,
      type: 'startEnd',
      position: { x: (maxRootWidth - START_END_WIDTH) / 2, y: 0 },
      draggable: false,
      data: {
        type: 'start',
        isSelected: selectedTaskId === '__test_header__',
      } as StartEndNodeData,
      selected: selectedTaskId === '__test_header__',
      style: { width: START_END_WIDTH, height: START_END_HEIGHT },
    });

    let currentY = START_END_HEIGHT + VERTICAL_GAP;

    // First drop zone (after start)
    const firstDropId = generateId('drop');
    nodes.push({
      id: firstDropId,
      type: 'dropZone',
      position: { x: 0, y: currentY },
      draggable: false,
      data: {
        dropId: tasks.length > 0 ? `insert-before-${tasks[0].id}` : 'insert-at-end',
        insertIndex: 0,
      } as DropZoneNodeData,
      style: { width: maxRootWidth, height: DROP_ZONE_HEIGHT },
    });

    addEdge(startNodeId, firstDropId);

    let prevNodeId = firstDropId;
    currentY += DROP_ZONE_HEIGHT + VERTICAL_GAP;

    // Process root tasks
    for (let i = 0; i < tasks.length; i++) {
      const task = tasks[i];
      const size = taskSizes[i];

      // Center task horizontally
      const taskX = (maxRootWidth - size.width) / 2;
      const result = processTask(task, taskX, currentY, undefined);

      // Edge from previous node
      addEdge(prevNodeId, result.nodeId);

      currentY += size.height + VERTICAL_GAP;

      // Drop zone after this task
      const dropId = generateId('drop');
      nodes.push({
        id: dropId,
        type: 'dropZone',
        position: { x: 0, y: currentY },
        draggable: false,
        data: {
          dropId: i < tasks.length - 1 ? `insert-before-${tasks[i + 1].id}` : 'insert-at-end',
          insertIndex: i + 1,
        } as DropZoneNodeData,
        style: { width: maxRootWidth, height: DROP_ZONE_HEIGHT },
      });

      // Edge to drop zone
      addEdge(result.nodeId, dropId);

      prevNodeId = dropId;
      currentY += DROP_ZONE_HEIGHT + VERTICAL_GAP;
    }

    // Add end node
    const endNodeId = 'end-node';
    nodes.push({
      id: endNodeId,
      type: 'startEnd',
      position: { x: (maxRootWidth - START_END_WIDTH) / 2, y: currentY },
      draggable: false,
      data: { type: 'end' } as StartEndNodeData,
      style: { width: START_END_WIDTH, height: START_END_HEIGHT },
    });

    addEdge(prevNodeId, endNodeId);

    currentY += START_END_HEIGHT + VERTICAL_GAP;

    // Add cleanup phase divider and tasks
    const cleanupDividerId = 'cleanup-divider';
    nodes.push({
      id: cleanupDividerId,
      type: 'startEnd',
      position: { x: (maxRootWidth - PHASE_DIVIDER_WIDTH) / 2, y: currentY },
      draggable: false,
      data: { type: 'cleanup' } as StartEndNodeData,
      style: { width: PHASE_DIVIDER_WIDTH, height: PHASE_DIVIDER_HEIGHT },
    });

    addEdge(endNodeId, cleanupDividerId, 'var(--color-amber-500)', true);

    currentY += PHASE_DIVIDER_HEIGHT + VERTICAL_GAP;

    // First cleanup drop zone
    const firstCleanupDropId = generateId('drop');
    nodes.push({
      id: firstCleanupDropId,
      type: 'dropZone',
      position: { x: 0, y: currentY },
      draggable: false,
      data: {
        dropId: cleanupTasks.length > 0 ? `cleanup-insert-before-${cleanupTasks[0].id}` : 'cleanup-insert-at-end',
        insertIndex: 0,
        isCleanup: true,
      } as DropZoneNodeData,
      style: { width: maxRootWidth, height: DROP_ZONE_HEIGHT },
    });

    addEdge(cleanupDividerId, firstCleanupDropId, 'var(--color-amber-500)');

    let prevCleanupNodeId = firstCleanupDropId;
    currentY += DROP_ZONE_HEIGHT + VERTICAL_GAP;

    // Process cleanup tasks
    for (let i = 0; i < cleanupTasks.length; i++) {
      const task = cleanupTasks[i];
      const size = cleanupTaskSizes[i];

      // Center task horizontally
      const taskX = (maxRootWidth - size.width) / 2;
      const result = processTask(task, taskX, currentY, undefined, undefined, true);

      // Edge from previous node
      addEdge(prevCleanupNodeId, result.nodeId, 'var(--color-amber-500)');

      currentY += size.height + VERTICAL_GAP;

      // Drop zone after this task
      const dropId = generateId('drop');
      nodes.push({
        id: dropId,
        type: 'dropZone',
        position: { x: 0, y: currentY },
        draggable: false,
        data: {
          dropId: i < cleanupTasks.length - 1 ? `cleanup-insert-before-${cleanupTasks[i + 1].id}` : 'cleanup-insert-at-end',
          insertIndex: i + 1,
          isCleanup: true,
        } as DropZoneNodeData,
        style: { width: maxRootWidth, height: DROP_ZONE_HEIGHT },
      });

      // Edge to drop zone
      addEdge(result.nodeId, dropId, 'var(--color-amber-500)');

      prevCleanupNodeId = dropId;
      currentY += DROP_ZONE_HEIGHT + VERTICAL_GAP;
    }

    // Center the entire graph horizontally around x=0
    const offsetX = -maxRootWidth / 2;
    for (const node of nodes) {
      if (!node.parentNode) {
        node.position.x += offsetX;
      }
    }

    return { nodes, edges };
  }, [tasks, cleanupTasks, descriptorMap, selectedTaskId]);
}
