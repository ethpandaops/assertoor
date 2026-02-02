import { useCallback, useState, useRef } from 'react';
import {
  DndContext,
  DragOverlay,
  closestCenter,
  PointerSensor,
  useSensor,
  useSensors,
  type DragStartEvent,
  type DragEndEvent,
  type DragOverEvent,
} from '@dnd-kit/core';
import { useBuilderStore } from '../../stores/builderStore';
import { createTask, findTaskById, wouldCreateCircular, getTaskIndex } from '../../utils/builder/taskUtils';
import TaskPalette from './palette/TaskPalette';
import BuilderCanvas from './canvas/BuilderCanvas';
import ConfigPanel from './config/ConfigPanel';
import BuilderToolbar from './toolbar/BuilderToolbar';
import ValidationPanel from './validation/ValidationPanel';
import type { TaskDescriptor } from '../../types/api';

// Resize handle component
interface ResizeHandleProps {
  onResize: (delta: number) => void;
  direction: 'left' | 'right';
}

function ResizeHandle({ onResize, direction }: ResizeHandleProps) {
  const handleRef = useRef<HTMLDivElement>(null);
  const isDragging = useRef(false);
  const startX = useRef(0);

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    isDragging.current = true;
    startX.current = e.clientX;

    const handleMouseMove = (e: MouseEvent) => {
      if (!isDragging.current) return;
      const delta = e.clientX - startX.current;
      startX.current = e.clientX;
      onResize(direction === 'left' ? delta : -delta);
    };

    const handleMouseUp = () => {
      isDragging.current = false;
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
  }, [onResize, direction]);

  return (
    <div
      ref={handleRef}
      onMouseDown={handleMouseDown}
      className="w-1 cursor-col-resize hover:bg-primary-400 active:bg-primary-500 transition-colors shrink-0"
      title="Drag to resize"
    />
  );
}

interface BuilderLayoutProps {
  isLoading?: boolean;
}

// DnD context value to pass to children
export interface BuilderDndContext {
  activeId: string | null;
  overId: string | null;
}

// Valid drop zone prefixes/IDs for main tasks
const VALID_DROP_ZONES = [
  'insert-before-',
  'insert-first-child-',
  'insert-after-children-',
  'insert-at-end',
];

// Valid drop zone prefixes/IDs for cleanup tasks
const VALID_CLEANUP_DROP_ZONES = [
  'cleanup-insert-before-',
  'cleanup-insert-first-child-',
  'cleanup-insert-after-children-',
  'cleanup-insert-at-end',
];

function isValidDropZone(id: string): boolean {
  return VALID_DROP_ZONES.some((zone) => id === zone || id.startsWith(zone));
}

function isCleanupDropZone(id: string): boolean {
  return VALID_CLEANUP_DROP_ZONES.some((zone) => id === zone || id.startsWith(zone));
}

function BuilderLayout({ isLoading }: BuilderLayoutProps) {
  const validationErrors = useBuilderStore((state) => state.validationErrors);
  const selection = useBuilderStore((state) => state.selection);
  const tasks = useBuilderStore((state) => state.testConfig.tasks);
  const cleanupTasks = useBuilderStore((state) => state.testConfig.cleanupTasks || []);
  const addTask = useBuilderStore((state) => state.addTask);
  const moveTask = useBuilderStore((state) => state.moveTask);
  const addCleanupTask = useBuilderStore((state) => state.addCleanupTask);
  const moveCleanupTask = useBuilderStore((state) => state.moveCleanupTask);
  const moveTaskToCleanup = useBuilderStore((state) => state.moveTaskToCleanup);
  const moveCleanupTaskToMain = useBuilderStore((state) => state.moveCleanupTaskToMain);
  const setSelection = useBuilderStore((state) => state.setSelection);

  // Panel visibility and sizes
  const [showPalette, setShowPalette] = useState(true);
  const [showConfig, setShowConfig] = useState(true);
  const [paletteWidth, setPaletteWidth] = useState(256); // 16rem = 256px
  const [configWidth, setConfigWidth] = useState(320); // 20rem = 320px

  // Resize handlers
  const handlePaletteResize = useCallback((delta: number) => {
    setPaletteWidth((w) => Math.max(200, Math.min(400, w + delta)));
  }, []);

  const handleConfigResize = useCallback((delta: number) => {
    setConfigWidth((w) => Math.max(280, Math.min(800, w + delta)));
  }, []);

  // DnD state
  const [activeId, setActiveId] = useState<string | null>(null);
  const [overId, setOverId] = useState<string | null>(null);

  // Configure sensors
  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8,
      },
    })
  );

  // Handle drag start
  const handleDragStart = useCallback((event: DragStartEvent) => {
    const { active } = event;
    const activeIdStr = String(active.id);
    setActiveId(activeIdStr);

    // If dragging a task that's not in the selection, select only that task
    if (!activeIdStr.startsWith('palette-') && !selection.taskIds.has(activeIdStr)) {
      setSelection([activeIdStr], activeIdStr);
    }
  }, [selection.taskIds, setSelection]);

  // Handle drag over
  const handleDragOver = useCallback((event: DragOverEvent) => {
    const { over } = event;
    setOverId(over ? String(over.id) : null);
  }, []);

  // Parse drop target for main tasks
  const parseDropTarget = useCallback((overIdStr: string): { parentId: string | undefined; index: number } | null => {
    // Insert at end of root tasks
    if (overIdStr === 'insert-at-end') {
      return { parentId: undefined, index: tasks.length };
    }

    // Insert before a specific task
    if (overIdStr.startsWith('insert-before-')) {
      const targetTaskId = overIdStr.replace('insert-before-', '');
      const location = getTaskIndex(tasks, targetTaskId);
      if (location) {
        return { parentId: location.parentId || undefined, index: location.index };
      }
      return null;
    }

    // Insert as first child of a glue task
    if (overIdStr.startsWith('insert-first-child-')) {
      const parentTaskId = overIdStr.replace('insert-first-child-', '');
      return { parentId: parentTaskId, index: 0 };
    }

    // Insert at end of glue task children
    if (overIdStr.startsWith('insert-after-children-')) {
      const parentTaskId = overIdStr.replace('insert-after-children-', '');
      const parent = findTaskById(tasks, parentTaskId);
      if (parent) {
        return { parentId: parentTaskId, index: parent.children?.length || 0 };
      }
      return null;
    }

    return null;
  }, [tasks]);

  // Parse drop target for cleanup tasks
  const parseCleanupDropTarget = useCallback((overIdStr: string): { parentId: string | undefined; index: number } | null => {
    // Insert at end of cleanup tasks
    if (overIdStr === 'cleanup-insert-at-end') {
      return { parentId: undefined, index: cleanupTasks.length };
    }

    // Insert before a specific cleanup task
    if (overIdStr.startsWith('cleanup-insert-before-')) {
      const targetTaskId = overIdStr.replace('cleanup-insert-before-', '');
      const location = getTaskIndex(cleanupTasks, targetTaskId);
      if (location) {
        return { parentId: location.parentId || undefined, index: location.index };
      }
      return null;
    }

    // Insert as first child of a glue task
    if (overIdStr.startsWith('cleanup-insert-first-child-')) {
      const parentTaskId = overIdStr.replace('cleanup-insert-first-child-', '');
      return { parentId: parentTaskId, index: 0 };
    }

    // Insert at end of glue task children
    if (overIdStr.startsWith('cleanup-insert-after-children-')) {
      const parentTaskId = overIdStr.replace('cleanup-insert-after-children-', '');
      const parent = findTaskById(cleanupTasks, parentTaskId);
      if (parent) {
        return { parentId: parentTaskId, index: parent.children?.length || 0 };
      }
      return null;
    }

    return null;
  }, [cleanupTasks]);

  // Handle drag end
  const handleDragEnd = useCallback((event: DragEndEvent) => {
    const { active, over } = event;

    setActiveId(null);
    setOverId(null);

    if (!over) return;

    const activeIdStr = String(active.id);
    const overIdStr = String(over.id);

    const isCleanup = isCleanupDropZone(overIdStr);
    const isMain = isValidDropZone(overIdStr);

    // Only process drops on valid drop zones
    if (!isMain && !isCleanup) return;

    // Handle palette drop (new task)
    if (activeIdStr.startsWith('palette-')) {
      const descriptor = active.data.current?.descriptor as TaskDescriptor | undefined;
      if (!descriptor) return;

      if (isCleanup) {
        const target = parseCleanupDropTarget(overIdStr);
        if (!target) return;

        const newTask = createTask(descriptor.name);
        addCleanupTask(newTask, target.parentId, target.index);
        setSelection([newTask.id], newTask.id);
      } else {
        const target = parseDropTarget(overIdStr);
        if (!target) return;

        const newTask = createTask(descriptor.name);
        addTask(newTask, target.parentId, target.index);
        setSelection([newTask.id], newTask.id);
      }
      return;
    }

    // Handle reordering existing tasks
    if (activeIdStr !== overIdStr && !activeIdStr.startsWith('palette-')) {
      // Determine which task array the dragged task is from
      const isFromMain = findTaskById(tasks, activeIdStr) !== null;
      const isFromCleanup = findTaskById(cleanupTasks, activeIdStr) !== null;

      // Handle moving within main tasks
      if (isMain && isFromMain) {
        const target = parseDropTarget(overIdStr);
        if (!target) return;

        // Get all task IDs to move (all selected tasks if the active one is selected)
        const taskIdsToMove = selection.taskIds.has(activeIdStr)
          ? Array.from(selection.taskIds).filter((id) => findTaskById(tasks, id) !== null)
          : [activeIdStr];

        // Check for circular reference for ALL tasks being moved
        if (target.parentId) {
          for (const taskIdToMove of taskIdsToMove) {
            if (wouldCreateCircular(tasks, taskIdToMove, target.parentId)) {
              return; // Can't drop any of the selected tasks into their own children
            }
          }
        }

        // Move each selected task to the target location
        const tasksWithLocations = taskIdsToMove
          .map((id) => ({ id, location: getTaskIndex(tasks, id) }))
          .filter((t) => t.location !== null)
          .sort((a, b) => {
            if (a.location!.parentId !== b.location!.parentId) {
              return (a.location!.parentId || '').localeCompare(b.location!.parentId || '');
            }
            return a.location!.index - b.location!.index;
          });

        let insertIndex = target.index;
        for (let i = 0; i < tasksWithLocations.length; i++) {
          const { id, location } = tasksWithLocations[i];

          let adjustedIndex = insertIndex;
          if (location!.parentId === (target.parentId || null)) {
            if (location!.index < insertIndex) {
              adjustedIndex = insertIndex - 1;
            }
          }

          moveTask(id, target.parentId, adjustedIndex);
          insertIndex++;
        }
      } else if (isCleanup && isFromCleanup) {
        const target = parseCleanupDropTarget(overIdStr);
        if (!target) return;

        // Get all task IDs to move
        const taskIdsToMove = selection.taskIds.has(activeIdStr)
          ? Array.from(selection.taskIds).filter((id) => findTaskById(cleanupTasks, id) !== null)
          : [activeIdStr];

        // Check for circular reference
        if (target.parentId) {
          for (const taskIdToMove of taskIdsToMove) {
            if (wouldCreateCircular(cleanupTasks, taskIdToMove, target.parentId)) {
              return;
            }
          }
        }

        // Move each selected task
        const tasksWithLocations = taskIdsToMove
          .map((id) => ({ id, location: getTaskIndex(cleanupTasks, id) }))
          .filter((t) => t.location !== null)
          .sort((a, b) => {
            if (a.location!.parentId !== b.location!.parentId) {
              return (a.location!.parentId || '').localeCompare(b.location!.parentId || '');
            }
            return a.location!.index - b.location!.index;
          });

        let insertIndex = target.index;
        for (let i = 0; i < tasksWithLocations.length; i++) {
          const { id, location } = tasksWithLocations[i];

          let adjustedIndex = insertIndex;
          if (location!.parentId === (target.parentId || null)) {
            if (location!.index < insertIndex) {
              adjustedIndex = insertIndex - 1;
            }
          }

          moveCleanupTask(id, target.parentId, adjustedIndex);
          insertIndex++;
        }
      } else if (isCleanup && isFromMain) {
        // Moving from main to cleanup
        const target = parseCleanupDropTarget(overIdStr);
        if (!target) return;

        // Only move the active task (not multi-select for cross-area moves)
        moveTaskToCleanup(activeIdStr, target.parentId, target.index);
      } else if (isMain && isFromCleanup) {
        // Moving from cleanup to main
        const target = parseDropTarget(overIdStr);
        if (!target) return;

        // Only move the active task (not multi-select for cross-area moves)
        moveCleanupTaskToMain(activeIdStr, target.parentId, target.index);
      }
    }
  }, [tasks, cleanupTasks, addTask, addCleanupTask, moveTask, moveCleanupTask, moveTaskToCleanup, moveCleanupTaskToMain, setSelection, parseDropTarget, parseCleanupDropTarget, selection.taskIds]);

  // Get active item for drag overlay
  const activeTask = activeId && !activeId.startsWith('palette-')
    ? (findTaskById(tasks, activeId) || findTaskById(cleanupTasks, activeId))
    : null;

  const activePaletteItem = activeId?.startsWith('palette-')
    ? activeId.replace('palette-', '')
    : null;

  // DnD context value
  const dndContext: BuilderDndContext = { activeId, overId };

  // Check if we should show config panel
  const showConfigPanel = showConfig && selection.primaryTaskId && selection.taskIds.size === 1;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-[calc(100vh-12rem)]">
        <div className="animate-spin rounded-full size-8 border-b-2 border-primary-600"></div>
      </div>
    );
  }

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      onDragStart={handleDragStart}
      onDragOver={handleDragOver}
      onDragEnd={handleDragEnd}
    >
      <div className="flex flex-col h-[calc(100vh-12rem)] min-h-[500px]">
        {/* Toolbar */}
        <BuilderToolbar
          showPalette={showPalette}
          showConfig={showConfig}
          onTogglePalette={() => setShowPalette(!showPalette)}
          onToggleConfig={() => setShowConfig(!showConfig)}
        />

        {/* Main content area */}
        <div className="flex flex-1 overflow-hidden border border-[var(--color-border)] rounded-b-lg">
          {/* Left sidebar - Task Palette */}
          {showPalette && (
            <>
              <div
                className="flex flex-col bg-[var(--color-bg-secondary)] overflow-hidden"
                style={{ width: paletteWidth }}
              >
                <TaskPalette />
              </div>
              <ResizeHandle direction="left" onResize={handlePaletteResize} />
            </>
          )}

          {/* Center - Canvas */}
          <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
            <BuilderCanvas dndContext={dndContext} />

            {/* Validation errors */}
            {validationErrors.length > 0 && (
              <ValidationPanel errors={validationErrors} />
            )}
          </div>

          {/* Right sidebar - Config Panel */}
          {showConfigPanel && selection.primaryTaskId && (
            <>
              <ResizeHandle direction="right" onResize={handleConfigResize} />
              <div
                className="flex flex-col bg-[var(--color-bg-secondary)] overflow-hidden"
                style={{ width: configWidth }}
              >
                <ConfigPanel taskId={selection.primaryTaskId} />
              </div>
            </>
          )}

          {/* Multi-selection info panel */}
          {showConfig && selection.taskIds.size > 1 && (
            <>
              <ResizeHandle direction="right" onResize={handleConfigResize} />
              <div
                className="flex flex-col bg-[var(--color-bg-secondary)] overflow-hidden"
                style={{ width: configWidth }}
              >
                <div className="p-4">
                  <h3 className="text-sm font-semibold mb-2">Multiple Tasks Selected</h3>
                  <p className="text-xs text-[var(--color-text-secondary)] mb-3">
                    {selection.taskIds.size} tasks selected
                  </p>
                  <p className="text-xs text-[var(--color-text-tertiary)]">
                    Drag to move all selected tasks together.
                    Click a single task to edit its configuration.
                  </p>
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      {/* Drag overlay */}
      <DragOverlay>
        {activeTask && (
          <div className="px-3 py-2 bg-[var(--color-bg-primary)] border border-primary-500 rounded shadow-lg">
            <span className="text-sm font-medium">{activeTask.title || activeTask.taskType}</span>
          </div>
        )}
        {activePaletteItem && (
          <div className="px-3 py-2 bg-[var(--color-bg-primary)] border border-primary-500 rounded shadow-lg">
            <span className="text-sm font-medium">{activePaletteItem}</span>
          </div>
        )}
      </DragOverlay>
    </DndContext>
  );
}

export default BuilderLayout;
