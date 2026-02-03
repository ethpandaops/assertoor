import { memo, useCallback } from 'react';
import { useDraggable, useDroppable } from '@dnd-kit/core';
import { useBuilderStore, getNamedChildSlots, hasNamedChildren } from '../../../stores/builderStore';
import { canHaveChildren } from '../../../utils/builder/taskUtils';
import type { BuilderTask, NamedChildSlot } from '../../../stores/builderStore';
import type { TaskDescriptor } from '../../../types/api';

// Color mapping for Tailwind classes (since dynamic class names aren't purged)
const slotColorClasses: Record<string, { bg: string; bgDark: string; text: string; textDark: string }> = {
  amber: {
    bg: 'bg-amber-100',
    bgDark: 'dark:bg-amber-900/30',
    text: 'text-amber-700',
    textDark: 'dark:text-amber-300',
  },
  emerald: {
    bg: 'bg-emerald-100',
    bgDark: 'dark:bg-emerald-900/30',
    text: 'text-emerald-700',
    textDark: 'dark:text-emerald-300',
  },
  blue: {
    bg: 'bg-blue-100',
    bgDark: 'dark:bg-blue-900/30',
    text: 'text-blue-700',
    textDark: 'dark:text-blue-300',
  },
  purple: {
    bg: 'bg-purple-100',
    bgDark: 'dark:bg-purple-900/30',
    text: 'text-purple-700',
    textDark: 'dark:text-purple-300',
  },
};

// Slot label component
function SlotLabel({ slot }: { slot: NamedChildSlot }) {
  const colors = slotColorClasses[slot.colorClass] || slotColorClasses.blue;
  return (
    <span className={`px-1.5 py-0.5 text-xs rounded-sm shrink-0 ${colors.bg} ${colors.bgDark} ${colors.text} ${colors.textDark}`}>
      {slot.label}
    </span>
  );
}

// Drop zone component for named child slots
interface NamedSlotDropZoneProps {
  dropId: string;
  slot: NamedChildSlot;
  childDepth: number;
  childAncestorLines: boolean[];
  isLastSlot: boolean;
  isOverSlot: boolean;
  renderTreeLines: (forDepth: number, lines: boolean[], showConnector: boolean, isLastItem: boolean) => React.ReactNode;
  TREE_INDENT: number;
}

function NamedSlotDropZone({
  dropId,
  slot,
  childDepth,
  childAncestorLines,
  isLastSlot,
  isOverSlot,
  renderTreeLines,
  TREE_INDENT,
}: NamedSlotDropZoneProps) {
  const { setNodeRef, isOver } = useDroppable({ id: dropId });
  const showIndicator = isOver || isOverSlot;
  const colors = slotColorClasses[slot.colorClass] || slotColorClasses.blue;

  return (
    <div
      ref={setNodeRef}
      className="relative"
      style={{ paddingLeft: `${childDepth * TREE_INDENT + 8}px` }}
    >
      {renderTreeLines(childDepth, childAncestorLines, true, isLastSlot)}
      <div
        className={`
          mx-2 my-1 px-3 py-2 rounded-sm border-2 border-dashed
          transition-all duration-150
          ${showIndicator
            ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
            : 'border-gray-300 dark:border-gray-600 hover:border-primary-400 hover:bg-primary-50/50 dark:hover:bg-primary-900/10'
          }
        `}
      >
        <div className="flex items-center gap-2">
          <span className={`px-1.5 py-0.5 text-xs rounded-sm ${colors.bg} ${colors.bgDark} ${colors.text} ${colors.textDark}`}>
            {slot.label}
          </span>
          <span className="text-xs text-[var(--color-text-tertiary)]">
            Drop {slot.name} task here
          </span>
        </div>
      </div>
    </div>
  );
}

interface BuilderListItemProps {
  task: BuilderTask;
  depth: number;
  isSelected: boolean;
  isPrimary: boolean;
  onSelect: (taskId: string, event: React.MouseEvent) => void;
  descriptorMap: Map<string, TaskDescriptor>;
  overId: string | null;
  isLast?: boolean;  // Is this the last child in its parent?
  ancestorLines?: boolean[];  // Which ancestor levels need vertical lines
  slotConfig?: NamedChildSlot;  // Slot configuration for named children (e.g., run_task_background)
  isCleanup?: boolean;  // Whether this is a cleanup task
  hideDropZones?: boolean;  // Hide insert-before drop zones (for fixed-position slots)
}

function BuilderListItem({
  task,
  depth,
  isSelected,
  isPrimary,
  onSelect,
  descriptorMap,
  overId,
  isLast = true,
  ancestorLines = [],
  slotConfig,
  isCleanup = false,
  hideDropZones = false,
}: BuilderListItemProps) {
  const removeTask = useBuilderStore((state) => state.removeTask);
  const duplicateTask = useBuilderStore((state) => state.duplicateTask);
  const selection = useBuilderStore((state) => state.selection);

  const {
    attributes,
    listeners,
    setNodeRef,
    isDragging,
  } = useDraggable({
    id: task.id,
  });

  const isGlue = canHaveChildren(task.taskType);
  const hasChildren = task.children && task.children.length > 0;
  const childCount = task.children?.length || 0;
  const taskNamedChildSlots = getNamedChildSlots(task.taskType);

  // Determine max children allowed for this task type
  const getMaxChildren = (taskType: string): number => {
    if (hasNamedChildren(taskType)) {
      const slots = getNamedChildSlots(taskType);
      return slots?.length || 0;
    }
    switch (taskType) {
      case 'run_task_options':
      case 'run_task_matrix':
        return 1; // single child task
      default:
        return Infinity; // unlimited for run_tasks, run_tasks_concurrent
    }
  };

  const maxChildren = getMaxChildren(task.taskType);
  const canAddMoreChildren = childCount < maxChildren;

  // Drop zone prefix based on whether this is a cleanup task
  const dropPrefix = isCleanup ? 'cleanup-' : '';

  // Drop zone for inserting BEFORE this task (disabled for fixed-position slots)
  const { setNodeRef: setBeforeRef, isOver: isOverBefore } = useDroppable({
    id: `${dropPrefix}insert-before-${task.id}`,
    disabled: hideDropZones,
  });

  // Determine glue task type category
  const isSingleChildTask = task.taskType === 'run_task_options' || task.taskType === 'run_task_matrix';
  const hasNamedChildSlots = !!taskNamedChildSlots;
  const isMultiChildTask = task.taskType === 'run_tasks' || task.taskType === 'run_tasks_concurrent';

  // For single-child tasks (run_task_options, run_task_matrix)
  const singleChild = isSingleChildTask ? task.children?.[0] : null;
  const { setNodeRef: setSingleChildRef, isOver: isOverSingleChild } = useDroppable({
    id: `${dropPrefix}insert-single-child-${task.id}`,
    disabled: !isSingleChildTask || !!singleChild,
  });

  // For multi-child tasks (run_tasks, run_tasks_concurrent) - "add more" placeholder at end
  const { setNodeRef: setAddMoreRef, isOver: isOverAddMore } = useDroppable({
    id: `${dropPrefix}insert-after-children-${task.id}`,
    disabled: !isMultiChildTask,
  });

  // Handle delete
  const handleDelete = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    removeTask(task.id);
  }, [removeTask, task.id]);

  // Handle duplicate
  const handleDuplicate = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    duplicateTask(task.id);
  }, [duplicateTask, task.id]);

  // Handle click
  const handleClick = useCallback((e: React.MouseEvent) => {
    onSelect(task.id, e);
  }, [onSelect, task.id]);

  // Check if any drop indicator should show
  const showBeforeIndicator = isOverBefore || overId === `${dropPrefix}insert-before-${task.id}`;
  const showSingleChildIndicator = isOverSingleChild || overId === `${dropPrefix}insert-single-child-${task.id}`;
  const showAddMoreIndicator = isOverAddMore || overId === `${dropPrefix}insert-after-children-${task.id}`;

  // Tree line width for each level
  const TREE_INDENT = 20;

  // Helper to render tree lines for a given depth
  const renderTreeLines = (forDepth: number, lines: boolean[], showConnector: boolean, isLastItem: boolean) => {
    if (forDepth === 0) return null;

    return (
      <div
        className="absolute top-0 bottom-0 left-0 flex pointer-events-none"
        style={{ marginLeft: 8 }}
      >
        {/* Ancestor continuation lines */}
        {lines.map((showLine, i) => (
          <div key={i} className="relative" style={{ width: TREE_INDENT }}>
            {showLine && (
              <div className="absolute left-1/2 top-0 bottom-0 w-0.5 bg-[var(--color-border)] -translate-x-1/2" />
            )}
          </div>
        ))}
        {/* Current level */}
        <div className="relative" style={{ width: TREE_INDENT }}>
          {showConnector ? (
            <>
              {/* Vertical line - half height if last, full height if not */}
              <div
                className="absolute left-1/2 w-0.5 bg-[var(--color-border)] -translate-x-1/2 top-0"
                style={{ height: isLastItem ? '50%' : '100%' }}
              />
              {/* Horizontal connector */}
              <div
                className="absolute left-1/2 top-1/2 h-0.5 bg-[var(--color-border)] -translate-y-1/2"
                style={{ width: '50%' }}
              />
            </>
          ) : (
            /* Just continuation line if not last */
            !isLastItem && (
              <div className="absolute left-1/2 top-0 bottom-0 w-0.5 bg-[var(--color-border)] -translate-x-1/2" />
            )
          )}
        </div>
      </div>
    );
  };

  return (
    <div className={isDragging ? 'opacity-50' : ''}>
      {/* Drop indicator BEFORE this task (hidden for fixed-position slots) */}
      {!hideDropZones && (
        <div
          ref={setBeforeRef}
          className="relative"
          style={{ paddingLeft: `${depth * TREE_INDENT + 8}px` }}
        >
          {/* Continuation lines for drop indicator - always show line since task is below */}
          {renderTreeLines(depth, ancestorLines, false, false)}
          <div
            className={`
              h-1 mx-2 rounded-sm transition-all duration-150
              ${showBeforeIndicator
                ? 'bg-primary-500 my-1'
                : 'bg-transparent hover:bg-primary-200 dark:hover:bg-primary-800'
              }
            `}
          />
        </div>
      )}

      {/* Main task row with tree lines */}
      <div
        className="relative flex"
        style={{ paddingLeft: `${depth * TREE_INDENT + 8}px` }}
      >
        {/* Tree lines - absolutely positioned */}
        {renderTreeLines(depth, ancestorLines, true, isLast)}

        {/* Clickable task content */}
        <div
          ref={setNodeRef}
          onClick={handleClick}
          className={`
            group flex items-center gap-2 flex-1 mr-2 px-2 py-1.5 rounded-sm cursor-pointer
            transition-all duration-150
            ${isSelected
              ? 'bg-primary-100 dark:bg-primary-900/30 border border-primary-300 dark:border-primary-700'
              : 'hover:bg-[var(--color-bg-tertiary)] border border-transparent'
            }
            ${isPrimary ? 'ring-2 ring-primary-500 ring-offset-1 ring-offset-[var(--color-bg-primary)]' : ''}
          `}
        >
          {/* Drag handle */}
          <div
            {...attributes}
            {...listeners}
            className="p-1 cursor-grab hover:bg-[var(--color-bg-tertiary)] rounded-sm shrink-0"
          >
            <DragIcon className="size-4 text-[var(--color-text-tertiary)]" />
          </div>

          {/* Task icon */}
          <div className={`shrink-0 ${isGlue ? 'text-primary-600' : 'text-[var(--color-text-tertiary)]'}`}>
            {isGlue ? (
              <ContainerIcon className="size-4" />
            ) : (
              <TaskIcon className="size-4" />
            )}
          </div>

          {/* Slot label for named children */}
          {slotConfig && (
            <SlotLabel slot={slotConfig} />
          )}

          {/* Task info */}
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium truncate">
                {task.title || task.taskType}
              </span>
              {task.taskId && (
                <span className="text-xs text-[var(--color-text-tertiary)] font-mono shrink-0">
                  #{task.taskId}
                </span>
              )}
            </div>
            {task.title && task.title !== task.taskType && (
              <span className="text-xs text-[var(--color-text-tertiary)] truncate block">
                {task.taskType}
              </span>
            )}
          </div>

          {/* Quick actions */}
          <div className="flex items-center gap-1 shrink-0 opacity-0 group-hover:opacity-100">
            <button
              onClick={handleDuplicate}
              className="p-1 hover:bg-[var(--color-bg-tertiary)] rounded-sm"
              title="Duplicate"
            >
              <DuplicateIcon className="size-3.5 text-[var(--color-text-tertiary)]" />
            </button>
            <button
              onClick={handleDelete}
              className="p-1 hover:bg-red-100 dark:hover:bg-red-900/30 rounded-sm"
              title="Delete"
            >
              <DeleteIcon className="size-3.5 text-red-500" />
            </button>
          </div>
        </div>
      </div>

      {/* Children section for glue tasks */}
      {isGlue && (
        <>
          {/* Calculate ancestor lines for children:
              - Children at depth 1 (of root tasks) should have NO ancestor lines
              - Children at depth 2+ inherit parent's lines + parent's continuation status */}
          {(() => {
            const childDepth = depth + 1;
            // Only add parent's continuation if parent has tree lines (depth > 0)
            const childAncestorLines = depth === 0 ? [] : [...ancestorLines, !isLast];

            // === Named child slots (e.g., run_task_background): render slots dynamically ===
            if (hasNamedChildSlots && taskNamedChildSlots) {
              return (
                <>
                  {taskNamedChildSlots.map((slot, slotIndex) => {
                    const slotChild = task.namedChildren?.[slot.name];
                    const isLastSlot = slotIndex === taskNamedChildSlots.length - 1;
                    const dropId = `${dropPrefix}insert-named-${slot.name}-${task.id}`;
                    const isOverSlot = overId === dropId;

                    if (slotChild) {
                      return (
                        <BuilderListItem
                          key={`${slot.name}-child-${slotChild.id}`}
                          task={slotChild}
                          depth={childDepth}
                          isSelected={selection.taskIds.has(slotChild.id)}
                          isPrimary={selection.primaryTaskId === slotChild.id}
                          onSelect={onSelect}
                          descriptorMap={descriptorMap}
                          overId={overId}
                          isLast={isLastSlot}
                          ancestorLines={childAncestorLines}
                          slotConfig={slot}
                          isCleanup={isCleanup}
                          hideDropZones={true}
                        />
                      );
                    }

                    return (
                      <NamedSlotDropZone
                        key={`${slot.name}-placeholder-${task.id}`}
                        dropId={dropId}
                        slot={slot}
                        childDepth={childDepth}
                        childAncestorLines={childAncestorLines}
                        isLastSlot={isLastSlot}
                        isOverSlot={isOverSlot}
                        renderTreeLines={renderTreeLines}
                        TREE_INDENT={TREE_INDENT}
                      />
                    );
                  })}
                </>
              );
            }

            // === Single-child tasks (run_task_options, run_task_matrix): show 1 slot ===
            if (isSingleChildTask) {
              return (
                <>
                  {singleChild ? (
                    <BuilderListItem
                      task={singleChild}
                      depth={childDepth}
                      isSelected={selection.taskIds.has(singleChild.id)}
                      isPrimary={selection.primaryTaskId === singleChild.id}
                      onSelect={onSelect}
                      descriptorMap={descriptorMap}
                      overId={overId}
                      isLast={true}
                      ancestorLines={childAncestorLines}
                      isCleanup={isCleanup}
                      hideDropZones={true}
                    />
                  ) : (
                    <div
                      ref={setSingleChildRef}
                      className="relative"
                      style={{ paddingLeft: `${childDepth * TREE_INDENT + 8}px` }}
                    >
                      {renderTreeLines(childDepth, childAncestorLines, true, true)}
                      <div
                        className={`
                          mx-2 my-1 px-3 py-2 rounded-sm border-2 border-dashed
                          transition-all duration-150
                          ${showSingleChildIndicator
                            ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
                            : 'border-gray-300 dark:border-gray-600 hover:border-primary-400 hover:bg-primary-50/50 dark:hover:bg-primary-900/10'
                          }
                        `}
                      >
                        <div className="flex items-center gap-2">
                          <PlusIcon className="size-4 text-gray-400" />
                          <span className="text-xs text-[var(--color-text-tertiary)]">
                            Drop task here
                          </span>
                        </div>
                      </div>
                    </div>
                  )}
                </>
              );
            }

            // === Multi-child tasks (run_tasks, run_tasks_concurrent): show children + add placeholder ===
            return (
              <>
                {/* Render existing children */}
                {task.children?.map((child, index) => {
                  const isLastChild = index === task.children!.length - 1;

                  return (
                    <BuilderListItem
                      key={child.id}
                      task={child}
                      depth={childDepth}
                      isSelected={selection.taskIds.has(child.id)}
                      isPrimary={selection.primaryTaskId === child.id}
                      onSelect={onSelect}
                      descriptorMap={descriptorMap}
                      overId={overId}
                      isLast={isLastChild && !canAddMoreChildren}
                      ancestorLines={childAncestorLines}
                      isCleanup={isCleanup}
                    />
                  );
                })}

                {/* "Add more" placeholder at the end */}
                <div
                  ref={setAddMoreRef}
                  className="relative"
                  style={{ paddingLeft: `${childDepth * TREE_INDENT + 8}px` }}
                >
                  {renderTreeLines(childDepth, childAncestorLines, true, true)}
                  <div
                    className={`
                      mx-2 my-1 px-3 py-2 rounded-sm border-2 border-dashed
                      transition-all duration-150
                      ${showAddMoreIndicator
                        ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
                        : 'border-gray-300 dark:border-gray-600 hover:border-primary-400 hover:bg-primary-50/50 dark:hover:bg-primary-900/10'
                      }
                    `}
                  >
                    <div className="flex items-center gap-2">
                      <PlusIcon className="size-4 text-gray-400" />
                      <span className="text-xs text-[var(--color-text-tertiary)]">
                        {hasChildren ? 'Drop to add more tasks' : 'Drop task here'}
                      </span>
                    </div>
                  </div>
                </div>
              </>
            );
          })()}
        </>
      )}
    </div>
  );
}

// Icons
function DragIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M4 8h16M4 16h16"
      />
    </svg>
  );
}

function ContainerIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
      />
    </svg>
  );
}

function TaskIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"
      />
    </svg>
  );
}

function DuplicateIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
      />
    </svg>
  );
}

function DeleteIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
      />
    </svg>
  );
}

function PlusIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M12 4v16m8-8H4"
      />
    </svg>
  );
}

export default memo(BuilderListItem);
