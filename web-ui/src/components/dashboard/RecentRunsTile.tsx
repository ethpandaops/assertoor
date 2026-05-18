import { Link } from 'react-router-dom';
import { useTestRuns, useTests } from '../../hooks/useApi';
import StatusBadge from '../common/StatusBadge';
import { formatDuration } from '../../utils/time';
import type { DashboardTile, RecentRunsConfig } from './types';
import type { TestRun } from '../../types/api';

interface RecentRunsTileProps {
  tile: DashboardTile;
  config: RecentRunsConfig;
}

// RecentRunsTile lists the most recent test runs, optionally filtered
// to a single test. Each row links to the run detail page. The tile
// auto-polls so the home page stays live without manual refresh.
export function RecentRunsTile({ tile, config }: RecentRunsTileProps) {
  const { data: tests } = useTests();
  const test = config.testId
    ? tests?.find((t) => t.id === config.testId)
    : undefined;

  const { data: runs, isLoading, error } = useTestRuns(config.testId, {
    refetchInterval: 5_000,
  });

  const limit = Math.max(1, Math.min(50, config.limit || 5));
  const shown = (runs ?? []).slice(0, limit);

  const title =
    tile.title ||
    (config.testId
      ? `Recent runs · ${test?.name ?? config.testId}`
      : 'Recent runs');

  return (
    <div className="card overflow-hidden h-full flex flex-col">
      <header className="card-header flex items-center justify-between gap-2 flex-shrink-0">
        <span className="font-medium truncate" title={title}>
          {title}
        </span>
        <Link
          to={config.testId ? `/runs?testId=${encodeURIComponent(config.testId)}` : '/runs'}
          className="text-xs text-primary-600 hover:underline shrink-0"
        >
          view all
        </Link>
      </header>

      <div className="flex-1 overflow-auto">
        {isLoading ? (
          <p className="p-3 text-xs text-[var(--color-text-tertiary)]">Loading…</p>
        ) : error ? (
          <p className="p-3 text-xs text-error-600">{error.message}</p>
        ) : shown.length === 0 ? (
          <p className="p-3 text-xs text-[var(--color-text-tertiary)] italic">
            No runs yet.
          </p>
        ) : (
          <ul className="divide-y divide-[var(--color-border)] text-xs">
            {shown.map((run) => (
              <RunRow key={run.run_id} run={run} showTestName={!config.testId} />
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

function RunRow({ run, showTestName }: { run: TestRun; showTestName: boolean }) {
  const duration =
    run.stop_time > 0 && run.start_time > 0
      ? run.stop_time - run.start_time
      : run.start_time > 0
        ? Math.floor(Date.now() / 1000) - run.start_time
        : 0;

  return (
    <li className="flex items-center gap-2 px-3 py-2 hover:bg-[var(--color-bg-tertiary)]">
      <StatusBadge status={run.status} size="sm" />
      <Link
        to={`/run/${run.run_id}`}
        className="font-mono text-primary-600 hover:underline shrink-0"
      >
        #{run.run_id}
      </Link>
      {showTestName && (
        <Link
          to={`/test/${encodeURIComponent(run.test_id)}`}
          className="truncate hover:underline text-[var(--color-text-secondary)] flex-1"
          title={run.name}
        >
          {run.name}
        </Link>
      )}
      {!showTestName && <span className="flex-1" />}
      <span className="text-[var(--color-text-tertiary)] whitespace-nowrap">
        {run.start_time > 0
          ? new Date(run.start_time * 1000).toLocaleString()
          : 'not started'}
      </span>
      {duration > 0 && (
        <span className="font-mono text-[var(--color-text-tertiary)] whitespace-nowrap">
          {formatDuration(duration)}
        </span>
      )}
    </li>
  );
}

export default RecentRunsTile;
