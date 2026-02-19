import { useState, useMemo } from 'react';
import { useTaskDescriptors } from '../../../hooks/useApi';
import TaskPaletteSearch from './TaskPaletteSearch';
import TaskPaletteCategory from './TaskPaletteCategory';
import type { TaskDescriptor } from '../../../types/api';

// Group tasks by category
function groupByCategory(descriptors: TaskDescriptor[]): Map<string, TaskDescriptor[]> {
  const groups = new Map<string, TaskDescriptor[]>();

  for (const desc of descriptors) {
    const category = desc.category || 'Uncategorized';
    const existing = groups.get(category) || [];
    existing.push(desc);
    groups.set(category, existing);
  }

  // Sort tasks within each category
  for (const [, tasks] of groups) {
    tasks.sort((a, b) => a.name.localeCompare(b.name));
  }

  return groups;
}

// Filter tasks by search query
function filterTasks(
  descriptors: TaskDescriptor[],
  query: string
): TaskDescriptor[] {
  if (!query.trim()) return descriptors;

  const lowerQuery = query.toLowerCase();
  return descriptors.filter((desc) => {
    // Search in name
    if (desc.name.toLowerCase().includes(lowerQuery)) return true;
    // Search in description
    if (desc.description?.toLowerCase().includes(lowerQuery)) return true;
    // Search in category
    if (desc.category?.toLowerCase().includes(lowerQuery)) return true;
    // Search in aliases
    if (desc.aliases?.some((a) => a.toLowerCase().includes(lowerQuery))) return true;
    return false;
  });
}

function TaskPalette() {
  const { data: descriptors, isLoading, error } = useTaskDescriptors();
  const [searchQuery, setSearchQuery] = useState('');
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(new Set());

  // Filter and group tasks
  const filteredAndGrouped = useMemo(() => {
    if (!descriptors) return new Map<string, TaskDescriptor[]>();

    const filtered = filterTasks(descriptors, searchQuery);
    return groupByCategory(filtered);
  }, [descriptors, searchQuery]);

  // Category names sorted
  const categoryNames = useMemo(() => {
    return Array.from(filteredAndGrouped.keys()).sort((a, b) => {
      // Put "Flow Control" and common categories first
      const priority = ['Flow Control', 'Assertions', 'Clients', 'Transactions', 'Beacon'];
      const aIndex = priority.indexOf(a);
      const bIndex = priority.indexOf(b);
      if (aIndex !== -1 && bIndex !== -1) return aIndex - bIndex;
      if (aIndex !== -1) return -1;
      if (bIndex !== -1) return 1;
      return a.localeCompare(b);
    });
  }, [filteredAndGrouped]);

  // Toggle category expansion
  const toggleCategory = (category: string) => {
    setExpandedCategories((prev) => {
      const next = new Set(prev);
      if (next.has(category)) {
        next.delete(category);
      } else {
        next.add(category);
      }
      return next;
    });
  };

  // Expand all when searching
  const effectiveExpanded = useMemo(() => {
    if (searchQuery.trim()) {
      // Expand all when searching
      return new Set(categoryNames);
    }
    return expandedCategories;
  }, [searchQuery, categoryNames, expandedCategories]);

  if (isLoading) {
    return (
      <div className="flex flex-col h-full">
        <div className="p-3 border-b border-[var(--color-border)]">
          <h3 className="text-sm font-semibold">Task Palette</h3>
        </div>
        <div className="flex items-center justify-center flex-1">
          <div className="animate-spin rounded-full size-6 border-b-2 border-primary-600"></div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col h-full">
        <div className="p-3 border-b border-[var(--color-border)]">
          <h3 className="text-sm font-semibold">Task Palette</h3>
        </div>
        <div className="p-3 text-sm text-error-600">
          Failed to load tasks: {error.message}
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="p-3 border-b border-[var(--color-border)]">
        <h3 className="text-sm font-semibold mb-2">Task Palette</h3>
        <TaskPaletteSearch value={searchQuery} onChange={setSearchQuery} />
      </div>

      {/* Task list */}
      <div className="flex-1 overflow-y-auto">
        {categoryNames.length === 0 ? (
          <div className="p-3 text-sm text-[var(--color-text-secondary)] text-center">
            {searchQuery ? 'No tasks match your search' : 'No tasks available'}
          </div>
        ) : (
          <div className="py-1">
            {categoryNames.map((category) => (
              <TaskPaletteCategory
                key={category}
                category={category}
                tasks={filteredAndGrouped.get(category) || []}
                isExpanded={effectiveExpanded.has(category)}
                onToggle={() => toggleCategory(category)}
                searchQuery={searchQuery}
              />
            ))}
          </div>
        )}
      </div>

      {/* Help text */}
      <div className="p-3 border-t border-[var(--color-border)] text-xs text-[var(--color-text-tertiary)]">
        Drag tasks to the canvas or double-click to add
      </div>
    </div>
  );
}

export default TaskPalette;
