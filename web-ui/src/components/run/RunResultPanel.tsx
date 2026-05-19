import { useState } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { useTestRunResult } from '../../hooks/useApi';

interface RunResultPanelProps {
  runId: number;
  // Poll while the test is still going, stop once it ends.
  isRunning: boolean;
}

// RunResultPanel renders the run-level $ASSERTOOR_TEST_RESULT markdown as a
// prominent panel on the test run page. It is collapsed by default but
// auto-expands once content is available.
export function RunResultPanel({ runId, isRunning }: RunResultPanelProps) {
  const { data, error, isLoading } = useTestRunResult(runId, {
    refetchInterval: isRunning ? 5000 : false,
  });

  const [collapsed, setCollapsed] = useState(false);

  if (isLoading && data === undefined) {
    return null;
  }

  if (error || !data) {
    // No result set yet — don't render the panel at all.
    return null;
  }

  return (
    <div className="card overflow-hidden">
      <button
        type="button"
        onClick={() => setCollapsed((v) => !v)}
        className="card-header w-full flex items-center justify-between text-left hover:bg-[var(--color-bg-tertiary)] transition-colors"
        aria-expanded={!collapsed}
      >
        <span className="font-medium">Result</span>
        <div className="flex items-center gap-3 text-xs text-[var(--color-text-tertiary)]">
          <a
            href={`/api/v1/test_run/${runId}/result`}
            target="_blank"
            rel="noopener noreferrer"
            onClick={(e) => e.stopPropagation()}
            className="text-primary-600 hover:underline"
          >
            view raw
          </a>
          <span>{collapsed ? '▸' : '▾'}</span>
        </div>
      </button>

      {!collapsed && (
        <div className="markdown-body p-4 text-sm overflow-auto">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{data}</ReactMarkdown>
        </div>
      )}
    </div>
  );
}

export default RunResultPanel;
