import { Link } from 'react-router-dom';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { useLatestTestResult, useTests } from '../../hooks/useApi';
import StatusBadge from '../common/StatusBadge';
import type { DashboardTile, LatestResultConfig } from './types';

interface LatestResultTileProps {
  tile: DashboardTile;
  config: LatestResultConfig;
}

// LatestResultTile renders the most-recent $ASSERTOOR_TEST_RESULT
// markdown produced by any run of the configured test. Server-side we
// already walk the newest N runs to find one that has a result, so
// this tile only does one round-trip even when the latest few runs
// failed without writing anything.
export function LatestResultTile({ tile, config }: LatestResultTileProps) {
  const { data: tests } = useTests();
  const test = tests?.find((t) => t.id === config.testId);

  const { data, isLoading, error } = useLatestTestResult(config.testId, {
    enabled: !!config.testId,
  });

  const title = tile.title || test?.name || config.testId || 'Unconfigured';
  const hasResult = data && data.run_id > 0 && data.markdown;

  return (
    <div className="card overflow-hidden h-full flex flex-col">
      {config.showHeader !== false && (
        <header className="card-header flex items-center justify-between gap-2 flex-shrink-0">
          <div className="flex items-center gap-2 min-w-0">
            <span className="font-medium truncate" title={title}>
              {title}
            </span>
            {data && data.run_id > 0 && (
              <>
                <Link
                  to={`/run/${data.run_id}`}
                  className="font-mono text-xs text-primary-600 hover:underline shrink-0"
                >
                  #{data.run_id}
                </Link>
                {data.status && <StatusBadge status={data.status} size="sm" />}
              </>
            )}
          </div>
          {data && data.run_id > 0 && (
            <a
              href={`/api/v1/test_run/${data.run_id}/result`}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-primary-600 hover:underline shrink-0"
            >
              view raw
            </a>
          )}
        </header>
      )}

      <div className="p-4 flex-1 overflow-auto markdown-body text-sm">
        {!config.testId ? (
          <p className="text-xs text-[var(--color-text-tertiary)] italic">
            Edit this tile and select a test whose latest result you want to surface.
          </p>
        ) : isLoading ? (
          <p className="text-xs text-[var(--color-text-tertiary)]">Loading…</p>
        ) : error ? (
          <p className="text-xs text-error-600">{error.message}</p>
        ) : !hasResult ? (
          <p className="text-xs text-[var(--color-text-tertiary)] italic">
            No run of <span className="font-mono">{config.testId}</span> has produced a
            result yet. Tasks can write to <code>$ASSERTOOR_TEST_RESULT</code> to
            surface markdown here.
          </p>
        ) : (
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{data!.markdown}</ReactMarkdown>
        )}
      </div>
    </div>
  );
}

export default LatestResultTile;
