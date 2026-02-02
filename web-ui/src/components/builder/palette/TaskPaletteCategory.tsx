import TaskPaletteItem from './TaskPaletteItem';
import type { TaskDescriptor } from '../../../types/api';

interface TaskPaletteCategoryProps {
  category: string;
  tasks: TaskDescriptor[];
  isExpanded: boolean;
  onToggle: () => void;
  searchQuery?: string;
}

function TaskPaletteCategory({
  category,
  tasks,
  isExpanded,
  onToggle,
  searchQuery,
}: TaskPaletteCategoryProps) {
  return (
    <div className="border-b border-[var(--color-border)] last:border-b-0">
      {/* Category header */}
      <button
        onClick={onToggle}
        className="w-full flex items-center gap-2 px-3 py-2 hover:bg-[var(--color-bg-tertiary)] transition-colors"
      >
        <ChevronIcon
          className={`size-4 text-[var(--color-text-tertiary)] transition-transform ${
            isExpanded ? 'rotate-90' : ''
          }`}
        />
        <span className="text-sm font-medium flex-1 text-left">{category}</span>
        <span className="text-xs text-[var(--color-text-tertiary)] bg-[var(--color-bg-tertiary)] px-1.5 py-0.5 rounded">
          {tasks.length}
        </span>
      </button>

      {/* Task items */}
      {isExpanded && (
        <div className="pb-1">
          {tasks.map((task) => (
            <TaskPaletteItem
              key={task.name}
              descriptor={task}
              searchQuery={searchQuery}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function ChevronIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
    </svg>
  );
}

export default TaskPaletteCategory;
