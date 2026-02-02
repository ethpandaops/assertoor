import { useCallback, useState } from 'react';
import { useDraggable } from '@dnd-kit/core';
import { useBuilderStore } from '../../../stores/builderStore';
import { createTask, canHaveChildren } from '../../../utils/builder/taskUtils';
import type { TaskDescriptor } from '../../../types/api';

interface TaskPaletteItemProps {
  descriptor: TaskDescriptor;
  searchQuery?: string;
}

// Highlight matching text
function highlightMatch(text: string, query: string): React.ReactNode {
  if (!query.trim()) return text;

  const lowerText = text.toLowerCase();
  const lowerQuery = query.toLowerCase();
  const index = lowerText.indexOf(lowerQuery);

  if (index === -1) return text;

  return (
    <>
      {text.substring(0, index)}
      <mark className="bg-yellow-200 dark:bg-yellow-800 rounded px-0.5">
        {text.substring(index, index + query.length)}
      </mark>
      {text.substring(index + query.length)}
    </>
  );
}

function TaskPaletteItem({ descriptor, searchQuery }: TaskPaletteItemProps) {
  const addTask = useBuilderStore((state) => state.addTask);
  const setSelection = useBuilderStore((state) => state.setSelection);
  const [showTooltip, setShowTooltip] = useState(false);

  // Draggable setup
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `palette-${descriptor.name}`,
    data: {
      type: 'palette-item',
      descriptor,
    },
  });

  // Handle double-click to add task
  const handleDoubleClick = useCallback(() => {
    const task = createTask(descriptor.name);
    addTask(task);
    setSelection([task.id], task.id);
  }, [addTask, setSelection, descriptor.name]);

  const isGlue = canHaveChildren(descriptor.name);

  return (
    <div
      ref={setNodeRef}
      {...listeners}
      {...attributes}
      onDoubleClick={handleDoubleClick}
      onMouseEnter={() => setShowTooltip(true)}
      onMouseLeave={() => setShowTooltip(false)}
      className={`
        relative mx-2 mb-1 px-2.5 py-1.5 rounded cursor-grab
        bg-[var(--color-bg-primary)] border border-[var(--color-border)]
        hover:border-primary-400 hover:shadow-sm
        transition-all duration-150
        ${isDragging ? 'opacity-50 cursor-grabbing' : ''}
      `}
    >
      {/* Task name */}
      <div className="flex items-center gap-1.5">
        {isGlue && (
          <ContainerIcon className="size-3.5 text-[var(--color-text-tertiary)] shrink-0" />
        )}
        <span className="text-sm font-medium truncate">
          {searchQuery ? highlightMatch(descriptor.name, searchQuery) : descriptor.name}
        </span>
      </div>

      {/* Description preview */}
      {descriptor.description && (
        <p className="text-xs text-[var(--color-text-tertiary)] truncate mt-0.5">
          {searchQuery
            ? highlightMatch(descriptor.description, searchQuery)
            : descriptor.description}
        </p>
      )}

      {/* Tooltip */}
      {showTooltip && !isDragging && (
        <div className="absolute left-full top-0 ml-2 z-50 w-64 p-3 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded shadow-lg">
          <h4 className="font-semibold text-sm mb-1">{descriptor.name}</h4>
          {descriptor.description && (
            <p className="text-xs text-[var(--color-text-secondary)] mb-2">
              {descriptor.description}
            </p>
          )}
          {descriptor.category && (
            <div className="text-xs">
              <span className="text-[var(--color-text-tertiary)]">Category: </span>
              <span className="text-[var(--color-text-secondary)]">{descriptor.category}</span>
            </div>
          )}
          {descriptor.outputs && descriptor.outputs.length > 0 && (
            <div className="mt-2">
              <span className="text-xs text-[var(--color-text-tertiary)]">Outputs:</span>
              <ul className="text-xs mt-1 space-y-0.5">
                {descriptor.outputs.slice(0, 3).map((output) => (
                  <li key={output.name} className="text-[var(--color-text-secondary)]">
                    <span className="font-mono text-primary-600">{output.name}</span>
                    <span className="text-[var(--color-text-tertiary)]"> ({output.type})</span>
                  </li>
                ))}
                {descriptor.outputs.length > 3 && (
                  <li className="text-[var(--color-text-tertiary)]">
                    +{descriptor.outputs.length - 3} more...
                  </li>
                )}
              </ul>
            </div>
          )}
          <div className="mt-2 pt-2 border-t border-[var(--color-border)] text-xs text-[var(--color-text-tertiary)]">
            Double-click or drag to add
          </div>
        </div>
      )}
    </div>
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

export default TaskPaletteItem;
