import { useMemo } from 'react';
import { Link } from 'react-router-dom';
import { useTestRuns, useTests } from '../../hooks/useApi';
import type { SuccessRateConfig, DashboardTile } from './types';
import type { TestRun } from '../../types/api';

interface SuccessRateTileProps {
  tile: DashboardTile;
  config: SuccessRateConfig;
}

// Map of test status → swatch colour. Kept in sync with `StatusBadge`
// vocabulary but stylistically simpler (we render small squares, not
// labelled pills).
const SWATCH: Record<string, string> = {
  success: 'bg-green-500',
  failure: 'bg-red-500',
  aborted: 'bg-orange-500',
  skipped: 'bg-gray-400',
  running: 'bg-blue-500 animate-pulse',
  pending: 'bg-gray-300 dark:bg-gray-600',
};

// SuccessRateTile aggregates the most recent `window` runs of a test
// into a success-rate ring + a strip of per-run swatches. Clicking a
// swatch opens the corresponding run.
export function SuccessRateTile({ tile, config }: SuccessRateTileProps) {
  const { data: tests } = useTests();
  const test = useMemo(
    () => tests?.find((t) => t.id === config.testId),
    [tests, config.testId],
  );
  const { data: runs, isLoading, error } = useTestRuns(config.testId, {
    enabled: !!config.testId,
    refetchInterval: 15_000,
    staleTime: 5_000,
  });

  const window = Math.max(1, Math.min(50, config.window || 10));
  const recent = useMemo<TestRun[]>(
    () => (runs ? runs.slice(0, window) : []),
    [runs, window],
  );

  const counts = useMemo(() => {
    const out = {
      total: recent.length,
      success: 0,
      failure: 0,
      aborted: 0,
      skipped: 0,
      running: 0,
      pending: 0,
    };
    for (const r of recent) {
      switch (r.status) {
        case 'success':
          out.success++; break;
        case 'failure':
          out.failure++; break;
        case 'aborted':
          out.aborted++; break;
        case 'skipped':
          out.skipped++; break;
        case 'running':
          out.running++; break;
        case 'pending':
          out.pending++; break;
      }
    }
    return out;
  }, [recent]);

  // Denominator counts only completed runs — pending / running don't
  // belong in a success-rate yet.
  const denominator = counts.success + counts.failure + counts.aborted + counts.skipped;
  const rate = denominator > 0 ? (counts.success / denominator) * 100 : null;

  const title = tile.title || (test?.name ?? config.testId) || 'Unconfigured';

  return (
    <div className="card p-4 h-full flex flex-col gap-3 overflow-hidden">
      <header className="flex items-start justify-between gap-2 min-w-0">
        <div className="min-w-0">
          {config.testId ? (
            <Link
              to={`/test/${encodeURIComponent(config.testId)}`}
              className="font-medium truncate block hover:underline text-[var(--color-text-primary)]"
              title={title}
            >
              {title}
            </Link>
          ) : (
            <span className="font-medium truncate block text-[var(--color-text-tertiary)]">
              No test selected
            </span>
          )}
          <span className="text-xs text-[var(--color-text-tertiary)]">
            Last {window} runs
          </span>
        </div>
      </header>

      {!config.testId ? (
        <p className="text-xs text-[var(--color-text-tertiary)] italic">
          Edit this tile and select a test to track.
        </p>
      ) : isLoading ? (
        <p className="text-xs text-[var(--color-text-tertiary)]">Loading…</p>
      ) : error ? (
        <p className="text-xs text-error-600">{error.message}</p>
      ) : counts.total === 0 ? (
        <p className="text-xs text-[var(--color-text-tertiary)] italic">
          No runs yet.
        </p>
      ) : (
        <>
          <div className="flex items-center gap-4">
            <RateRing percentage={rate} />
            <div className="text-xs space-y-0.5 text-[var(--color-text-secondary)]">
              <Counter label="Success" value={counts.success} dot="bg-green-500" />
              <Counter label="Failure" value={counts.failure} dot="bg-red-500" />
              {counts.aborted > 0 && (
                <Counter label="Aborted" value={counts.aborted} dot="bg-orange-500" />
              )}
              {(counts.running + counts.pending) > 0 && (
                <Counter
                  label="In flight"
                  value={counts.running + counts.pending}
                  dot="bg-blue-500"
                />
              )}
            </div>
          </div>

          {/* Per-run swatch strip — newest on the left */}
          <div className="flex flex-wrap gap-1 pt-1">
            {recent.map((r) => (
              <Link
                key={r.run_id}
                to={`/run/${r.run_id}`}
                title={`#${r.run_id} ${r.status} — ${
                  r.start_time > 0 ? new Date(r.start_time * 1000).toLocaleString() : 'not started'
                }`}
                className={`size-4 rounded-sm ${SWATCH[r.status] ?? 'bg-gray-300'} hover:ring-2 hover:ring-primary-400`}
              />
            ))}
          </div>
        </>
      )}
    </div>
  );
}

function Counter({ label, value, dot }: { label: string; value: number; dot: string }) {
  return (
    <div className="flex items-center gap-1.5">
      <span className={`size-2 rounded-full ${dot}`} />
      <span>
        {label}: <span className="font-mono">{value}</span>
      </span>
    </div>
  );
}

// RateRing draws a circular progress arc representing the success
// percentage. The colour shifts from red→amber→green as the rate
// improves to make at-a-glance reading easy.
function RateRing({ percentage }: { percentage: number | null }) {
  const size = 64;
  const strokeWidth = 6;
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const pct = percentage ?? 0;
  const offset = circumference - (pct / 100) * circumference;
  const center = size / 2;

  const colour =
    percentage === null
      ? 'text-gray-400'
      : pct >= 90
        ? 'text-green-500'
        : pct >= 60
          ? 'text-amber-500'
          : 'text-red-500';

  return (
    <div className={`relative ${colour}`} style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90">
        <circle
          cx={center}
          cy={center}
          r={radius}
          fill="none"
          stroke="currentColor"
          strokeWidth={strokeWidth}
          opacity={0.15}
        />
        <circle
          cx={center}
          cy={center}
          r={radius}
          fill="none"
          stroke="currentColor"
          strokeWidth={strokeWidth}
          strokeDasharray={circumference}
          strokeDashoffset={percentage === null ? circumference : offset}
          strokeLinecap="round"
          className="transition-all duration-500"
        />
      </svg>
      <span className="absolute inset-0 flex items-center justify-center text-sm font-mono font-bold">
        {percentage === null ? '–' : `${Math.round(pct)}%`}
      </span>
    </div>
  );
}

export default SuccessRateTile;
