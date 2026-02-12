import { useCallback, useEffect, useMemo, useRef } from 'react';
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  type NodeTypes,
  useReactFlow,
  ReactFlowProvider,
} from 'reactflow';
import 'reactflow/dist/style.css';
import type { TaskState } from '../../types/api';
import TaskGraphNode from './TaskGraphNode';
import { useTaskGraph } from './useTaskGraph';

interface TaskGraphProps {
  tasks: TaskState[];
  selectedIndex: number | null;
  onSelect: (index: number) => void;
}

// Define custom node types
const nodeTypes: NodeTypes = {
  taskNode: TaskGraphNode,
};

function TaskGraphInner({ tasks, selectedIndex, onSelect }: TaskGraphProps) {
  const { nodes, edges } = useTaskGraph(tasks, selectedIndex, onSelect);
  const { fitView, getNode } = useReactFlow();
  const prevTasksLengthRef = useRef(tasks.length);
  const initialFitDone = useRef(false);

  // Fit view on initial load
  useEffect(() => {
    if (nodes.length > 0 && !initialFitDone.current) {
      // Small delay to ensure nodes are rendered
      const timer = setTimeout(() => {
        fitView({ padding: 0.2, duration: 300 });
        initialFitDone.current = true;
      }, 100);
      return () => clearTimeout(timer);
    }
  }, [nodes.length, fitView]);

  // Fit view when new tasks are added
  useEffect(() => {
    if (tasks.length !== prevTasksLengthRef.current) {
      prevTasksLengthRef.current = tasks.length;
      if (initialFitDone.current) {
        // Don't animate when adding tasks during execution
        fitView({ padding: 0.2, duration: 0 });
      }
    }
  }, [tasks.length, fitView]);

  // Center on selected node when selection changes
  useEffect(() => {
    if (selectedIndex === null) return;

    const node = getNode(`task-${selectedIndex}`);
    if (!node) return;

    // Don't center if node is already visible in viewport
    // Just highlight it instead
  }, [selectedIndex, getNode]);

  // MiniMap node color based on status
  const minimapNodeColor = useCallback((node: { data: { task: TaskState } }) => {
    const task = node.data?.task;
    if (!task) return '#9ca3af';

    if (task.status === 'running') return '#3b82f6';
    if (task.status === 'complete') {
      if (task.result === 'success') return '#22c55e';
      if (task.result === 'failure') return '#ef4444';
    }
    return '#9ca3af';
  }, []);

  // Empty state
  if (tasks.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-[var(--color-text-secondary)]">
        No tasks to display
      </div>
    );
  }

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      nodeTypes={nodeTypes}
      fitView
      fitViewOptions={{ padding: 0.2 }}
      minZoom={0.1}
      maxZoom={2}
      defaultEdgeOptions={{
        type: 'smoothstep',
      }}
      proOptions={{ hideAttribution: true }}
      className="bg-[var(--color-bg-primary)]"
    >
      <Background
        color="var(--color-border)"
        gap={20}
        size={1}
      />
      <Controls
        className="!bg-[var(--color-bg-secondary)] !border-[var(--color-border)] !shadow-sm [&>button]:!bg-[var(--color-bg-secondary)] [&>button]:!border-[var(--color-border)] [&>button]:!text-[var(--color-text-primary)] [&>button:hover]:!bg-[var(--color-bg-tertiary)]"
      />
      <MiniMap
        nodeColor={minimapNodeColor}
        maskColor="rgba(0, 0, 0, 0.1)"
        className="!bg-[var(--color-bg-secondary)] !border-[var(--color-border)]"
        pannable
        zoomable
      />
    </ReactFlow>
  );
}

// Wrapper component that provides ReactFlow context
function TaskGraph(props: TaskGraphProps) {
  // Memoize to prevent unnecessary re-renders of the entire graph
  const memoizedProps = useMemo(() => props, [props]);

  return (
    <ReactFlowProvider>
      <TaskGraphInner {...memoizedProps} />
    </ReactFlowProvider>
  );
}

export default TaskGraph;
