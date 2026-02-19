import { useCallback, useMemo } from 'react';
import { useDroppable } from '@dnd-kit/core';
import { useBuilderStore } from '../../../stores/builderStore';
import { useTaskDescriptors } from '../../../hooks/useApi';
import BuilderListItem from './BuilderListItem';
import type { TaskDescriptor } from '../../../types/api';
import type { BuilderDndContext } from '../BuilderLayout';

interface BuilderListProps {
  dndContext: BuilderDndContext;
}

function BuilderList({ dndContext }: BuilderListProps) {
  const testConfig = useBuilderStore((state) => state.testConfig);
  const tasks = testConfig.tasks;
  const cleanupTasks = testConfig.cleanupTasks || [];
  const selection = useBuilderStore((state) => state.selection);
  const setSelection = useBuilderStore((state) => state.setSelection);
  const { data: descriptors } = useTaskDescriptors();

  const { overId } = dndContext;

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

  // Handle click to select
  const handleSelect = useCallback((taskId: string, event: React.MouseEvent) => {
    if (event.ctrlKey || event.metaKey) {
      // Multi-select
      if (selection.taskIds.has(taskId)) {
        const newIds = Array.from(selection.taskIds).filter((id) => id !== taskId);
        setSelection(newIds, newIds[0] || undefined);
      } else {
        setSelection([...Array.from(selection.taskIds), taskId], taskId);
      }
    } else if (event.shiftKey && selection.primaryTaskId) {
      // Range select (simplified - just add to selection)
      const newIds = new Set(selection.taskIds);
      newIds.add(taskId);
      setSelection(Array.from(newIds), taskId);
    } else {
      // Single select
      setSelection([taskId], taskId);
    }
  }, [selection, setSelection]);

  // Handle test header click
  const handleHeaderClick = useCallback(() => {
    setSelection(['__test_header__'], '__test_header__');
  }, [setSelection]);

  // Drop zone for empty main tasks section
  const { setNodeRef: setEmptyMainRef, isOver: isOverEmptyMain } = useDroppable({
    id: 'empty-main-tasks',
    disabled: tasks.length > 0,
  });

  // Drop zone for appending to main tasks (separate from main area)
  const { setNodeRef: setAppendMainRef, isOver: isOverAppendMain } = useDroppable({
    id: 'insert-at-end',
    disabled: tasks.length === 0,
  });

  // Drop zone for empty cleanup tasks section
  const { setNodeRef: setEmptyCleanupRef, isOver: isOverEmptyCleanup } = useDroppable({
    id: 'empty-cleanup-tasks',
    disabled: cleanupTasks.length > 0,
  });

  // Drop zone for appending to cleanup tasks (separate from main area)
  const { setNodeRef: setAppendCleanupRef, isOver: isOverAppendCleanup } = useDroppable({
    id: 'cleanup-insert-at-end',
    disabled: cleanupTasks.length === 0,
  });

  const showMainDropIndicator = isOverEmptyMain || isOverAppendMain || overId === 'insert-at-end' || overId === 'empty-main-tasks';
  const showCleanupDropIndicator = isOverEmptyCleanup || isOverAppendCleanup || overId === 'cleanup-insert-at-end' || overId === 'empty-cleanup-tasks';
  const isHeaderSelected = selection.primaryTaskId === '__test_header__';

  return (
    <div className="h-full overflow-y-auto">
      {/* Test Header Section */}
      <div
        className={`
          mx-3 mt-3 p-3 rounded-lg border-2 cursor-pointer transition-all
          ${isHeaderSelected
            ? 'bg-emerald-50 dark:bg-emerald-900/20 border-emerald-500 ring-2 ring-emerald-300 ring-offset-2 ring-offset-[var(--color-bg-primary)]'
            : 'bg-[var(--color-bg-secondary)] border-[var(--color-border)] hover:border-emerald-400'
          }
        `}
        onClick={handleHeaderClick}
      >
        <div className="flex items-center gap-2">
          <div className={`p-1.5 rounded ${isHeaderSelected ? 'bg-emerald-200 dark:bg-emerald-800' : 'bg-emerald-100 dark:bg-emerald-900/50'}`}>
            <SettingsIcon className="size-4 text-emerald-600" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium truncate">{testConfig.name}</div>
            <div className="text-xs text-[var(--color-text-tertiary)]">
              Test Configuration
              {testConfig.timeout && ` • ${testConfig.timeout}`}
              {testConfig.testVars && Object.keys(testConfig.testVars).length > 0 && (
                ` • ${Object.keys(testConfig.testVars).length} variable${Object.keys(testConfig.testVars).length === 1 ? '' : 's'}`
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Main Tasks Section */}
      <div className="mt-3">
        <div className="px-4 py-1.5 flex items-center gap-2">
          <div className="h-px flex-1 bg-[var(--color-border)]" />
          <span className="text-xs font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider">
            Main Tasks
          </span>
          <span className="text-xs text-[var(--color-text-tertiary)]">
            ({tasks.length})
          </span>
          <div className="h-px flex-1 bg-[var(--color-border)]" />
        </div>

        {/* Main tasks list area */}
        <div className="mx-3 my-2">
          {tasks.length === 0 ? (
            <div
              ref={setEmptyMainRef}
              className={`
                min-h-24 rounded-lg transition-colors flex flex-col items-center justify-center text-center
                ${showMainDropIndicator
                  ? 'bg-primary-50 dark:bg-primary-900/20 ring-2 ring-primary-400'
                  : 'bg-[var(--color-bg-tertiary)]/50'
                }
              `}
            >
              <p className="text-sm text-[var(--color-text-tertiary)]">No tasks yet</p>
              <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
                Drag tasks here from the palette
              </p>
            </div>
          ) : (
            <div className="py-2">
              {tasks.map((task, index) => (
                <BuilderListItem
                  key={task.id}
                  task={task}
                  depth={0}
                  isSelected={selection.taskIds.has(task.id)}
                  isPrimary={selection.primaryTaskId === task.id}
                  onSelect={handleSelect}
                  descriptorMap={descriptorMap}
                  overId={overId}
                  isLast={index === tasks.length - 1}
                  ancestorLines={[]}
                  isCleanup={false}
                />
              ))}

              {/* Append placeholder at the end */}
              <div
                ref={setAppendMainRef}
                className={`
                  mx-2 mt-2 px-3 py-2 rounded-sm border-2 border-dashed
                  transition-all duration-150
                  ${showMainDropIndicator
                    ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
                    : 'border-gray-300 dark:border-gray-600 hover:border-primary-400 hover:bg-primary-50/50 dark:hover:bg-primary-900/10'
                  }
                `}
              >
                <div className="flex items-center gap-2">
                  <PlusIcon className="size-4 text-gray-400" />
                  <span className="text-xs text-[var(--color-text-tertiary)]">
                    Drop to add task at end
                  </span>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Cleanup Tasks Section */}
      <div className="mt-3 mb-3">
        <div className="px-4 py-1.5 flex items-center gap-2">
          <div className="h-px flex-1 bg-amber-300 dark:bg-amber-700" />
          <CleanupIcon className="size-4 text-amber-600" />
          <span className="text-xs font-semibold text-amber-700 dark:text-amber-300 uppercase tracking-wider">
            Cleanup Tasks
          </span>
          <span className="text-xs text-[var(--color-text-tertiary)]">
            ({cleanupTasks.length})
          </span>
          <div className="h-px flex-1 bg-amber-300 dark:bg-amber-700" />
        </div>

        {/* Cleanup tasks list area */}
        <div className="mx-3 my-2">
          {cleanupTasks.length === 0 ? (
            <div
              ref={setEmptyCleanupRef}
              className={`
                min-h-24 rounded-lg transition-colors flex flex-col items-center justify-center text-center
                ${showCleanupDropIndicator
                  ? 'bg-amber-50 dark:bg-amber-900/20 ring-2 ring-amber-400'
                  : 'bg-[var(--color-bg-tertiary)]/50'
                }
              `}
            >
              <p className="text-sm text-[var(--color-text-tertiary)]">No cleanup tasks</p>
              <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
                Drag tasks here from the palette
              </p>
            </div>
          ) : (
            <div className="py-2">
              {cleanupTasks.map((task, index) => (
                <BuilderListItem
                  key={task.id}
                  task={task}
                  depth={0}
                  isSelected={selection.taskIds.has(task.id)}
                  isPrimary={selection.primaryTaskId === task.id}
                  onSelect={handleSelect}
                  descriptorMap={descriptorMap}
                  overId={overId}
                  isLast={index === cleanupTasks.length - 1}
                  ancestorLines={[]}
                  isCleanup={true}
                />
              ))}

              {/* Append placeholder at the end of cleanup */}
              <div
                ref={setAppendCleanupRef}
                className={`
                  mx-2 mt-2 px-3 py-2 rounded-sm border-2 border-dashed
                  transition-all duration-150
                  ${showCleanupDropIndicator
                    ? 'border-amber-500 bg-amber-50 dark:bg-amber-900/20'
                    : 'border-gray-300 dark:border-gray-600 hover:border-amber-400 hover:bg-amber-50/50 dark:hover:bg-amber-900/10'
                  }
                `}
              >
                <div className="flex items-center gap-2">
                  <PlusIcon className="size-4 text-gray-400" />
                  <span className="text-xs text-[var(--color-text-tertiary)]">
                    Drop to add cleanup task at end
                  </span>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function SettingsIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
      />
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
      />
    </svg>
  );
}

function CleanupIcon({ className }: { className?: string }) {
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

export default BuilderList;
