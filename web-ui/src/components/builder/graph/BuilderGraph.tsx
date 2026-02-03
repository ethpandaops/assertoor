import { useCallback, useMemo } from 'react';
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  type Node,
  type NodeTypes,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { useBuilderStore } from '../../../stores/builderStore';
import { useTaskDescriptors } from '../../../hooks/useApi';
import BuilderNode from './BuilderNode';
import GlueTaskNode from './GlueTaskNode';
import StartEndNode from './StartEndNode';
import DropZoneNode from './DropZoneNode';
import LaneLabelNode from './LaneLabelNode';
import DividerNode from './DividerNode';
import { useBuilderGraphLayout } from './useBuilderGraphLayout';
import type { TaskDescriptor } from '../../../types/api';
import type { BuilderDndContext } from '../BuilderLayout';

// Node types
const nodeTypes: NodeTypes = {
  builderNode: BuilderNode,
  glueNode: GlueTaskNode,
  startEnd: StartEndNode,
  dropZone: DropZoneNode,
  laneLabel: LaneLabelNode,
  divider: DividerNode,
};

interface BuilderGraphProps {
  dndContext: BuilderDndContext;
}

function BuilderGraph({ dndContext: _dndContext }: BuilderGraphProps) {
  const tasks = useBuilderStore((state) => state.testConfig.tasks);
  const cleanupTasks = useBuilderStore((state) => state.testConfig.cleanupTasks || []);
  const selection = useBuilderStore((state) => state.selection);
  const setSelection = useBuilderStore((state) => state.setSelection);
  const { data: descriptors } = useTaskDescriptors();

  // Build descriptor map
  const descriptorMap = useMemo(() => {
    const map = new Map<string, TaskDescriptor>();
    if (descriptors) {
      for (const d of descriptors) {
        map.set(d.name, d);
      }
    }
    return map;
  }, [descriptors]);

  // Compute graph layout - using controlled mode (no internal state)
  const { nodes, edges } = useBuilderGraphLayout(
    tasks,
    cleanupTasks,
    descriptorMap,
    selection.primaryTaskId
  );

  // Handle node selection
  const handleNodeClick = useCallback((_event: React.MouseEvent, node: Node) => {
    // Handle clicks on start node for test header config
    if (node.type === 'startEnd' && node.data?.type === 'start') {
      setSelection(['__test_header__'], '__test_header__');
      return;
    }

    // Handle clicks on task nodes
    if (node.type === 'builderNode' || node.type === 'glueNode') {
      const taskId = node.id.replace('node-', '');
      setSelection([taskId], taskId);
    }
  }, [setSelection]);

  // Handle selection change
  const handleSelectionChange = useCallback(({ nodes: selectedNodes }: { nodes: Node[] }) => {
    const taskIds = selectedNodes
      .filter((n) => n.type === 'builderNode' || n.type === 'glueNode')
      .map((n) => n.id.replace('node-', ''));
    if (taskIds.length > 0) {
      setSelection(taskIds, taskIds[0]);
    }
  }, [setSelection]);

  // Handle pane click (deselect)
  const handlePaneClick = useCallback(() => {
    setSelection([]);
  }, [setSelection]);

  return (
    <div className="h-full w-full relative">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodeClick={handleNodeClick}
        onSelectionChange={handleSelectionChange}
        onPaneClick={handlePaneClick}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{
          padding: 0.3,
          maxZoom: 1.2,
        }}
        defaultEdgeOptions={{
          type: 'smoothstep',
          animated: false,
        }}
        minZoom={0.1}
        maxZoom={2}
        proOptions={{ hideAttribution: true }}
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={true}
        selectNodesOnDrag={false}
      >
        <Background color="var(--color-border)" gap={20} size={1} />
        <Controls showInteractive={false} />
        <MiniMap
          nodeColor={(node) => {
            if (node.selected) return 'var(--color-primary-600)';
            if (node.type === 'glueNode') return 'var(--color-primary-400)';
            if (node.type === 'startEnd') {
              if (node.data?.type === 'start') return '#10b981';
              if (node.data?.type === 'cleanup') return '#f59e0b';
              return '#ef4444';
            }
            if (node.type === 'dropZone' || node.type === 'laneLabel' || node.type === 'divider') return 'transparent';
            return 'var(--color-bg-tertiary)';
          }}
          maskColor="rgba(0, 0, 0, 0.1)"
          pannable
          zoomable
        />
      </ReactFlow>
    </div>
  );
}

export default BuilderGraph;
