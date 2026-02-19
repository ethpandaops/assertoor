import { useMemo } from 'react';
import type { Node, Edge } from 'reactflow';
import type { TaskState } from '../../types/api';

// Layout constants
const NODE_WIDTH = 180;
const NODE_HEIGHT = 80;
const JUNCTION_HEIGHT = 8;
const HORIZONTAL_GAP = 60;
const VERTICAL_GAP = 50;

// Glue task names that should not be rendered as nodes
const GLUE_TASKS = new Set([
  'run_tasks',
  'run_tasks_concurrent',
  'run_task_matrix',
  'run_task_options',
  'run_task_background',
]);

// Tasks that always execute children concurrently (parallel lanes)
const CONCURRENT_GLUE_TASKS = new Set([
  'run_tasks_concurrent',
]);

// Background glue: children run concurrently but only the foreground (last child)
// determines execution flow — the background task has no outgoing edges.
const BACKGROUND_GLUE_TASKS = new Set([
  'run_task_background',
]);

function isGlueTask(task: TaskState): boolean {
  return GLUE_TASKS.has(task.name);
}

function isConcurrentGlueTask(task: TaskState): boolean {
  // run_task_matrix is only concurrent when explicitly configured with runConcurrent: true
  if (task.name === 'run_task_matrix') return !!task.run_concurrent;
  return CONCURRENT_GLUE_TASKS.has(task.name);
}

function isBackgroundGlueTask(task: TaskState): boolean {
  return BACKGROUND_GLUE_TASKS.has(task.name);
}

export interface UseTaskGraphResult {
  nodes: Node[];
  edges: Edge[];
}

export function useTaskGraph(
  tasks: TaskState[],
  selectedIndex: number | null,
  onSelect: (index: number) => void
): UseTaskGraphResult {
  return useMemo(() => {
    if (!tasks || tasks.length === 0) {
      return { nodes: [], edges: [] };
    }

    // Build task lookup maps
    const taskMap = new Map<number, TaskState>();
    const childrenMap = new Map<number, TaskState[]>();

    for (const task of tasks) {
      taskMap.set(task.index, task);
    }

    for (const task of tasks) {
      const parentIndex = task.parent_index;
      if (parentIndex >= 0 && parentIndex !== task.index && taskMap.has(parentIndex)) {
        const siblings = childrenMap.get(parentIndex) || [];
        siblings.push(task);
        childrenMap.set(parentIndex, siblings);
      }
    }

    // Sort children by index
    childrenMap.forEach((children) => {
      children.sort((a, b) => a.index - b.index);
    });

    // Find root tasks
    const rootTasks = tasks.filter(task => {
      const parentIndex = task.parent_index;
      return parentIndex < 0 || parentIndex === task.index || !taskMap.has(parentIndex);
    }).sort((a, b) => a.index - b.index);

    // Execution graph edges: from -> to[]
    const graphEdges = new Map<number, Set<number>>();
    const reverseEdges = new Map<number, Set<number>>(); // to -> from[]
    const visibleTasks = new Set<number>();
    const junctionNodes = new Set<number>();
    let junctionCounter = 0;

    function addEdge(from: number, to: number) {
      if (!graphEdges.has(from)) graphEdges.set(from, new Set());
      graphEdges.get(from)!.add(to);
      if (!reverseEdges.has(to)) reverseEdges.set(to, new Set());
      reverseEdges.get(to)!.add(from);
    }

    // Connect two sets of nodes. When both sides are multi-lane,
    // insert a junction node so converging and diverging edges don't overlap.
    function connectNodes(fromList: number[], toList: number[]) {
      if (fromList.length > 1 && toList.length > 1) {
        junctionCounter--;
        const junctionIdx = junctionCounter;
        junctionNodes.add(junctionIdx);
        visibleTasks.add(junctionIdx);
        for (const f of fromList) addEdge(f, junctionIdx);
        for (const t of toList) addEdge(junctionIdx, t);
      } else {
        for (const f of fromList) {
          for (const t of toList) addEdge(f, t);
        }
      }
    }

    // Result of processing: first visible tasks (entry) and last visible tasks (exit)
    interface FlowResult {
      first: number[];
      last: number[];
    }

    // Process task tree and build execution graph
    function process(task: TaskState): FlowResult {
      const children = childrenMap.get(task.index) || [];

      if (isGlueTask(task)) {
        if (children.length === 0) {
          return { first: [], last: [] };
        }

        if (isConcurrentGlueTask(task)) {
          // Concurrent: all children run in parallel
          const allFirst: number[] = [];
          const allLast: number[] = [];

          for (const child of children) {
            const r = process(child);
            allFirst.push(...r.first);
            allLast.push(...r.last);
          }

          return { first: allFirst, last: allLast };
        } else if (isBackgroundGlueTask(task)) {
          // Background: children run concurrently (side by side),
          // but only the foreground (last child) has outgoing flow edges.
          const allFirst: number[] = [];
          const fgResult = process(children[children.length - 1]);
          allFirst.push(...fgResult.first);

          for (let i = 0; i < children.length - 1; i++) {
            const r = process(children[i]);
            allFirst.push(...r.first);
            // Background children have no outgoing edges — intentionally ignored
          }

          return { first: allFirst, last: fgResult.last };
        } else {
          // Sequential: children run one after another
          let prevLast: number[] = [];
          let first: number[] = [];

          for (let i = 0; i < children.length; i++) {
            const r = process(children[i]);

            // First child's entry is our entry
            if (i === 0) {
              first = r.first;
            }

            // Connect previous exit to current entry
            connectNodes(prevLast, r.first);

            // Update exit for next iteration
            prevLast = r.last.length > 0 ? r.last : prevLast;
          }

          return { first, last: prevLast };
        }
      } else {
        // Visible task
        visibleTasks.add(task.index);
        return { first: [task.index], last: [task.index] };
      }
    }

    // Process all roots sequentially
    let prevLast: number[] = [];
    for (const root of rootTasks) {
      const r = process(root);
      connectNodes(prevLast, r.first);
      prevLast = r.last.length > 0 ? r.last : prevLast;
    }

    // Now we have: visibleTasks and edges between them
    // Assign rows using topological sort (BFS from sources)
    const nodeRows = new Map<number, number>();
    const inDegree = new Map<number, number>();

    for (const idx of visibleTasks) {
      inDegree.set(idx, 0);
    }
    for (const idx of visibleTasks) {
      const outs = graphEdges.get(idx);
      if (outs) {
        for (const to of outs) {
          if (visibleTasks.has(to)) {
            inDegree.set(to, (inDegree.get(to) || 0) + 1);
          }
        }
      }
    }

    // Start with nodes that have no incoming edges
    let queue = Array.from(visibleTasks).filter(idx => (inDegree.get(idx) || 0) === 0);
    let row = 0;

    while (queue.length > 0) {
      const nextQueue: number[] = [];

      for (const idx of queue) {
        nodeRows.set(idx, row);
        const outs = graphEdges.get(idx);
        if (outs) {
          for (const to of outs) {
            if (visibleTasks.has(to)) {
              const newDegree = (inDegree.get(to) || 1) - 1;
              inDegree.set(to, newDegree);
              if (newDegree === 0) {
                nextQueue.push(to);
              }
            }
          }
        }
      }

      queue = nextQueue;
      row++;
    }

    // Handle any remaining nodes (cycles or disconnected)
    for (const idx of visibleTasks) {
      if (!nodeRows.has(idx)) {
        nodeRows.set(idx, row);
      }
    }

    // Group nodes by row for lane assignment
    const byRow = new Map<number, number[]>();
    for (const idx of visibleTasks) {
      const r = nodeRows.get(idx) || 0;
      if (!byRow.has(r)) byRow.set(r, []);
      byRow.get(r)!.push(idx);
    }

    // Assign lanes to minimize crossings
    // Strategy: inherit lane from predecessor when possible
    const nodeLanes = new Map<number, number>();
    const sortedRows = Array.from(byRow.keys()).sort((a, b) => a - b);

    for (const r of sortedRows) {
      const nodesInRow = byRow.get(r)!;

      // Sort by predecessor lane (for consistent ordering)
      nodesInRow.sort((a, b) => {
        const getPredLane = (idx: number): number => {
          const preds = reverseEdges.get(idx);
          if (preds) {
            for (const p of preds) {
              const lane = nodeLanes.get(p);
              if (lane !== undefined) return lane;
            }
          }
          return Infinity;
        };
        return getPredLane(a) - getPredLane(b);
      });

      // Assign lanes
      const usedLanes = new Set<number>();

      for (const idx of nodesInRow) {
        // Try to use predecessor's lane
        let targetLane = -1;
        const preds = reverseEdges.get(idx);
        if (preds) {
          for (const p of preds) {
            const lane = nodeLanes.get(p);
            if (lane !== undefined && !usedLanes.has(lane)) {
              targetLane = lane;
              break;
            }
          }
        }

        // If no preference or conflict, find next available
        if (targetLane < 0) {
          targetLane = 0;
          while (usedLanes.has(targetLane)) targetLane++;
        }

        nodeLanes.set(idx, targetLane);
        usedLanes.add(targetLane);
      }
    }

    // Find max lane for centering
    let maxLane = 0;
    for (const lane of nodeLanes.values()) {
      maxLane = Math.max(maxLane, lane);
    }

    // Compute per-row Y positions with variable node heights
    const rowLaneCounts = new Map<number, number>();
    for (const r of sortedRows) {
      rowLaneCounts.set(r, byRow.get(r)!.length);
    }

    const rowYPositions = new Map<number, number>();
    let currentY = 0;
    for (let i = 0; i < sortedRows.length; i++) {
      const r = sortedRows[i];
      rowYPositions.set(r, currentY);

      if (i < sortedRows.length - 1) {
        const isThisJunction = byRow.get(r)!.every(idx => junctionNodes.has(idx));
        const isNextJunction = byRow.get(sortedRows[i + 1])!.every(idx => junctionNodes.has(idx));
        const thisLanes = rowLaneCounts.get(r) || 1;
        const nextLanes = rowLaneCounts.get(sortedRows[i + 1]) || 1;

        // Compact gap around junction rows — custom step edge handles tight spaces.
        // Extra gap only for direct multi-lane transitions without junctions.
        let gap: number;
        if (isThisJunction || isNextJunction) {
          gap = 30;
        } else if (thisLanes > 1 || nextLanes > 1) {
          gap = VERTICAL_GAP + 20;
        } else {
          gap = VERTICAL_GAP;
        }

        const rowHeight = isThisJunction ? JUNCTION_HEIGHT : NODE_HEIGHT;
        currentY += rowHeight + gap;
      }
    }

    // Convert to React Flow format
    const totalWidth = (maxLane + 1) * (NODE_WIDTH + HORIZONTAL_GAP) - HORIZONTAL_GAP;
    const offsetX = -totalWidth / 2;

    const getNodeId = (idx: number): string =>
      junctionNodes.has(idx) ? `junction-${-idx}` : `task-${idx}`;

    const resultNodes: Node[] = [];
    for (const idx of visibleTasks) {
      const nodeRow = nodeRows.get(idx) || 0;
      const y = rowYPositions.get(nodeRow) || 0;

      if (junctionNodes.has(idx)) {
        // Junction node: align with lane 0 (first node position)
        const centerX = offsetX + NODE_WIDTH / 2 - JUNCTION_HEIGHT / 2;

        resultNodes.push({
          id: getNodeId(idx),
          type: 'junctionNode',
          position: { x: centerX, y },
          data: {},
        });
      } else {
        const task = taskMap.get(idx);
        if (!task) continue;

        const lane = nodeLanes.get(idx) || 0;

        resultNodes.push({
          id: getNodeId(idx),
          type: 'taskNode',
          position: {
            x: offsetX + lane * (NODE_WIDTH + HORIZONTAL_GAP),
            y,
          },
          data: {
            task,
            isSelected: idx === selectedIndex,
            onSelect,
          },
        });
      }
    }

    // Create edges
    const resultEdges: Edge[] = [];
    for (const [from, tos] of graphEdges) {
      if (!visibleTasks.has(from)) continue;

      const fromLane = nodeLanes.get(from) || 0;

      for (const to of tos) {
        if (!visibleTasks.has(to)) continue;

        const toLane = nodeLanes.get(to) || 0;

        // Color based on whichever end is an actual task (not a junction)
        const refTask = taskMap.get(to) || taskMap.get(from);
        const isRunning = refTask?.status === 'running';
        const isComplete = refTask?.status === 'complete';
        const isSuccess = refTask?.result === 'success';
        const isFailure = refTask?.result === 'failure';

        // Custom step edges for junctions (handles tight gaps without upward routing),
        // smoothstep for cross-lane task edges, straight for same-lane
        const isJunctionEdge = junctionNodes.has(from) || junctionNodes.has(to);
        const edgeType = isJunctionEdge ? 'customStep' : fromLane !== toLane ? 'smoothstep' : 'straight';

        resultEdges.push({
          id: `edge-${from}-${to}`,
          source: getNodeId(from),
          target: getNodeId(to),
          type: edgeType,
          animated: isRunning,
          style: {
            stroke: isRunning
              ? '#3b82f6'
              : isComplete && isSuccess
                ? '#22c55e'
                : isComplete && isFailure
                  ? '#ef4444'
                  : '#9ca3af',
            strokeWidth: 2,
          },
        });
      }
    }

    return { nodes: resultNodes, edges: resultEdges };
  }, [tasks, selectedIndex, onSelect]);
}

export default useTaskGraph;
