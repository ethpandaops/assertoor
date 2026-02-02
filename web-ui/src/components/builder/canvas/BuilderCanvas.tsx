import { useBuilderStore } from '../../../stores/builderStore';
import BuilderGraph from '../graph/BuilderGraph';
import BuilderList from '../list/BuilderList';
import BuilderYaml from '../yaml/BuilderYaml';
import type { BuilderDndContext } from '../BuilderLayout';

interface BuilderCanvasProps {
  dndContext: BuilderDndContext;
}

function BuilderCanvas({ dndContext }: BuilderCanvasProps) {
  const activeView = useBuilderStore((state) => state.activeView);
  const setActiveView = useBuilderStore((state) => state.setActiveView);
  const testConfig = useBuilderStore((state) => state.testConfig);

  return (
    <div className="flex flex-col h-full">
      {/* View mode toggle */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-[var(--color-border)] bg-[var(--color-bg-primary)]">
        <div className="flex items-center gap-2">
          <span className="text-sm text-[var(--color-text-secondary)]">
            {testConfig.tasks.length} task{testConfig.tasks.length !== 1 ? 's' : ''}
          </span>
        </div>

        {/* View toggle buttons */}
        <div className="flex items-center gap-1 bg-[var(--color-bg-secondary)] rounded p-0.5">
          <ViewToggleButton
            active={activeView === 'graph'}
            onClick={() => setActiveView('graph')}
            icon={<GraphIcon className="size-4" />}
            label="Graph"
          />
          <ViewToggleButton
            active={activeView === 'list'}
            onClick={() => setActiveView('list')}
            icon={<ListIcon className="size-4" />}
            label="List"
          />
          <ViewToggleButton
            active={activeView === 'yaml'}
            onClick={() => setActiveView('yaml')}
            icon={<CodeIcon className="size-4" />}
            label="YAML"
          />
        </div>
      </div>

      {/* Canvas content */}
      <div className="flex-1 overflow-hidden bg-[var(--color-bg-primary)]">
        {activeView === 'graph' && <BuilderGraph dndContext={dndContext} />}
        {activeView === 'list' && <BuilderList dndContext={dndContext} />}
        {activeView === 'yaml' && <BuilderYaml />}
      </div>
    </div>
  );
}

interface ViewToggleButtonProps {
  active: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  label: string;
}

function ViewToggleButton({ active, onClick, icon, label }: ViewToggleButtonProps) {
  return (
    <button
      onClick={onClick}
      className={`
        flex items-center gap-1.5 px-2.5 py-1 rounded text-xs font-medium transition-colors
        ${active
          ? 'bg-primary-600 text-white'
          : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-tertiary)]'
        }
      `}
    >
      {icon}
      {label}
    </button>
  );
}

function GraphIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M4 5a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1V5zm10 0a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1V5zM9 15a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z"
      />
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M7 10v4m10-4v4M12 14v-4"
      />
    </svg>
  );
}

function ListIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M4 6h16M4 12h16M4 18h16"
      />
    </svg>
  );
}

function CodeIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"
      />
    </svg>
  );
}

export default BuilderCanvas;
