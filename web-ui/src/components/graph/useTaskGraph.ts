import { useMemo } from 'react';
import type { Node, Edge } from 'reactflow';
import type { TaskState } from '../../types/api';
import type { TaskNodeData } from './TaskGraphNode';

// Layout constants
const NODE_WIDTH = 180;
const NODE_HEIGHT = 80;
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

// Tasks that execute children concurrently (parallel lanes)
const CONCURRENT_GLUE_TASKS = new Set([
  'run_tasks_concurrent',
  'run_task_matrix',
]);

function isGlueTask(task: TaskState): boolean {
  return GLUE_TASKS.has(task.name);
}

function isConcurrentGlueTask(task: TaskState): boolean {
  return CONCURRENT_GLUE_TASKS.has(task.name);
}

export interface UseTaskGraphResult {
  nodes: Node<TaskNodeData>[];
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
    const edges = new Map<number, Set<number>>();
    const reverseEdges = new Map<number, Set<number>>(); // to -> from[]
    const visibleTasks = new Set<number>();

    function addEdge(from: number, to: number) {
      if (!edges.has(from)) edges.set(from, new Set());
      edges.get(from)!.add(to);
      if (!reverseEdges.has(to)) reverseEdges.set(to, new Set());
      reverseEdges.get(to)!.add(from);
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
            for (const f of prevLast) {
              for (const t of r.first) {
                addEdge(f, t);
              }
            }

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
      for (const f of prevLast) {
        for (const t of r.first) {
          addEdge(f, t);
        }
      }
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
      const outs = edges.get(idx);
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
        const outs = edges.get(idx);
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

    // Compute per-row Y positions with extra gap at convergence/divergence transitions
    // When closing edges from a multi-lane row converge to a node that also fans out
    // to the next multi-lane row, the smoothstep curves overlap. Add extra gap there.
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
        let gap = VERTICAL_GAP;
        const thisLanes = rowLaneCounts.get(r) || 1;
        const nextLanes = rowLaneCounts.get(sortedRows[i + 1]) || 1;

        // Add extra gap when multi-lane rows are close together
        // (converging edges from this row + diverging edges to next row would overlap)
        if (thisLanes > 1 || nextLanes > 1) {
          gap += 30;
        }

        currentY += NODE_HEIGHT + gap;
      }
    }

    // Convert to React Flow format
    const totalWidth = (maxLane + 1) * (NODE_WIDTH + HORIZONTAL_GAP) - HORIZONTAL_GAP;
    const offsetX = -totalWidth / 2;

    const resultNodes: Node<TaskNodeData>[] = [];
    for (const idx of visibleTasks) {
      const task = taskMap.get(idx);
      if (!task) continue;

      const lane = nodeLanes.get(idx) || 0;
      const nodeRow = nodeRows.get(idx) || 0;

      resultNodes.push({
        id: `task-${idx}`,
        type: 'taskNode',
        position: {
          x: offsetX + lane * (NODE_WIDTH + HORIZONTAL_GAP),
          y: rowYPositions.get(nodeRow) || 0,
        },
        data: {
          task,
          isSelected: idx === selectedIndex,
          onSelect,
        },
      });
    }

    // Create edges
    const resultEdges: Edge[] = [];
    for (const [from, tos] of edges) {
      if (!visibleTasks.has(from)) continue;

      const fromLane = nodeLanes.get(from) || 0;

      for (const to of tos) {
        if (!visibleTasks.has(to)) continue;

        const toLane = nodeLanes.get(to) || 0;
        const toTask = taskMap.get(to);
        const isRunning = toTask?.status === 'running';
        const isComplete = toTask?.status === 'complete';
        const isSuccess = toTask?.result === 'success';
        const isFailure = toTask?.result === 'failure';

        // Use straight edge for same lane (vertical), smoothstep for different lanes
        const edgeType = fromLane === toLane ? 'straight' : 'smoothstep';

        resultEdges.push({
          id: `edge-${from}-${to}`,
          source: `task-${from}`,
          target: `task-${to}`,
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
