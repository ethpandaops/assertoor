import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import {
  useTestRuns,
  useTests,
  useCancelTestRun,
  useDeleteTestRuns,
} from '../hooks/useApi';
import { useAuthContext } from '../context/AuthContext';
import SplitPane from '../components/common/SplitPane';
import StatusBadge from '../components/common/StatusBadge';
import Dropdown from '../components/common/Dropdown';
import StartTestModal from '../components/test/StartTestModal';
import { formatDuration } from '../utils/time';
import type { Test, TestRun } from '../types/api';

// Runs is a split-view page: registered tests on the left, runs for
// the selected test (or all runs) on the right. It is the new home
// for the test-runs table that used to live on `/`, plus a list of
// the registered tests to scope the runs by. The URL carries the
// selected test as `?testId=…` so reloads + deep links work.
function Runs() {
  const [searchParams, setSearchParams] = useSearchParams();
  const selectedTestId = searchParams.get('testId') ?? '';

  const { data: tests, isLoading: testsLoading } = useTests();
  const [search, setSearch] = useState('');

  const handleSelectTest = useCallback(
    (testId: string) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (testId) next.set('testId', testId);
          else next.delete('testId');
          return next;
        },
        { replace: true },
      );
    },
    [setSearchParams],
  );

  // Filtered + sorted test list driven by the search box.
  const filteredTests = useMemo(() => {
    if (!tests) return [];
    const q = search.trim().toLowerCase();
    if (!q) return tests;
    return tests.filter((t) => {
      return (
        t.id.toLowerCase().includes(q) ||
        (t.name ?? '').toLowerCase().includes(q) ||
        (t.tags ?? []).some((tag) => tag.toLowerCase().includes(q))
      );
    });
  }, [tests, search]);

  return (
    <div className="space-y-4">
      <header className="flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">Tests &amp; Runs</h1>
          <p className="text-sm text-[var(--color-text-secondary)]">
            Pick a test on the left to scope the runs view.
          </p>
        </div>
      </header>

      <SplitPane
        storageKey="runs-page"
        defaultLeftWidth={32}
        minLeftWidth={22}
        maxLeftWidth={55}
        maxHeight="calc(100vh - 12rem)"
        left={
          <TestList
            tests={filteredTests}
            isLoading={testsLoading}
            selectedId={selectedTestId}
            onSelect={handleSelectTest}
            search={search}
            onSearchChange={setSearch}
            totalCount={tests?.length ?? 0}
          />
        }
        right={<RunsPane testId={selectedTestId || undefined} tests={tests ?? []} />}
      />
    </div>
  );
}

// ── Test list (left pane) ─────────────────────────────────────────

interface TestListProps {
  tests: Test[];
  isLoading: boolean;
  selectedId: string;
  onSelect: (id: string) => void;
  search: string;
  onSearchChange: (s: string) => void;
  totalCount: number;
}

function TestList({
  tests,
  isLoading,
  selectedId,
  onSelect,
  search,
  onSearchChange,
  totalCount,
}: TestListProps) {
  return (
    <div className="h-full flex flex-col pr-2">
      <div className="card-header flex items-center justify-between flex-shrink-0">
        <span className="font-medium">
          Tests <span className="text-xs text-[var(--color-text-tertiary)] font-normal">({totalCount})</span>
        </span>
        <Link to="/registry" className="text-xs text-primary-600 hover:underline">
          manage in library
        </Link>
      </div>

      <div className="px-2 py-2 border-b border-[var(--color-border)] flex-shrink-0">
        <input
          type="search"
          placeholder="Search tests…"
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          className="w-full px-2 py-1.5 text-sm bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
        />
      </div>

      <div className="flex-1 overflow-auto">
        <TestListItem
          label="All tests"
          subtitle={`${totalCount} registered`}
          active={selectedId === ''}
          onClick={() => onSelect('')}
        />
        {isLoading ? (
          <p className="p-3 text-xs text-[var(--color-text-tertiary)]">Loading…</p>
        ) : tests.length === 0 ? (
          <p className="p-3 text-xs text-[var(--color-text-tertiary)] italic">
            No matching tests.
          </p>
        ) : (
          tests.map((t) => (
            <TestListItem
              key={t.id}
              label={t.name || t.id}
              subtitle={t.id}
              tags={t.tags}
              active={selectedId === t.id}
              onClick={() => onSelect(t.id)}
            />
          ))
        )}
      </div>
    </div>
  );
}

function TestListItem({
  label,
  subtitle,
  tags,
  active,
  onClick,
}: {
  label: string;
  subtitle: string;
  tags?: string[];
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`w-full text-left px-3 py-2 border-b border-[var(--color-border)] hover:bg-[var(--color-bg-tertiary)] ${
        active ? 'bg-primary-50 dark:bg-primary-900/20 border-l-2 border-l-primary-500' : ''
      }`}
    >
      <div className="text-sm font-medium truncate" title={label}>
        {label}
      </div>
      <div className="text-xs text-[var(--color-text-tertiary)] truncate font-mono" title={subtitle}>
        {subtitle}
      </div>
      {tags && tags.length > 0 && (
        <div className="text-[10px] text-[var(--color-text-tertiary)] mt-0.5 truncate">
          {tags.slice(0, 3).join(' · ')}
        </div>
      )}
    </button>
  );
}

// ── Runs pane (right side) ────────────────────────────────────────

interface RunsPaneProps {
  testId?: string;
  tests: Test[];
}

const PAGE_SIZE = 25;

function RunsPane({ testId, tests }: RunsPaneProps) {
  const { isLoggedIn } = useAuthContext();
  const { data: runs, isLoading, error } = useTestRuns(testId, {
    refetchInterval: 5_000,
  });

  const cancelMutation = useCancelTestRun();
  const deleteMutation = useDeleteTestRuns();

  const [page, setPage] = useState(1);
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const [showStartModal, setShowStartModal] = useState(false);

  // Reset selection + paging when the scope changes.
  useEffect(() => {
    setSelected(new Set());
    setPage(1);
  }, [testId]);

  const selectedTest = useMemo(
    () => tests.find((t) => t.id === testId),
    [tests, testId],
  );

  const total = runs?.length ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  useEffect(() => {
    if (page > totalPages) setPage(totalPages);
  }, [page, totalPages]);

  const paginated = useMemo(() => {
    if (!runs) return [];
    const start = (page - 1) * PAGE_SIZE;
    return runs.slice(start, start + PAGE_SIZE);
  }, [runs, page]);

  const handleSelectAll = useCallback(
    (checked: boolean) => {
      setSelected(checked ? new Set(paginated.map((r) => r.run_id)) : new Set());
    },
    [paginated],
  );

  const handleSelectOne = useCallback((runId: number, checked: boolean) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (checked) next.add(runId);
      else next.delete(runId);
      return next;
    });
  }, []);

  const handleCancel = useCallback(
    async (run: TestRun) => {
      if (!confirm(`Cancel test run #${run.run_id}?`)) return;
      try {
        await cancelMutation.mutateAsync(run.run_id);
      } catch (err) {
        alert(`Failed to cancel: ${err}`);
      }
    },
    [cancelMutation],
  );

  const handleDelete = useCallback(
    async (runIds: number[]) => {
      if (runIds.length === 0) return;
      if (!confirm(`Delete ${runIds.length} test run(s)?`)) return;
      try {
        await deleteMutation.mutateAsync(runIds);
        setSelected(new Set());
      } catch (err) {
        alert(`Failed to delete: ${err}`);
      }
    },
    [deleteMutation],
  );

  const allSelected = paginated.length > 0 && paginated.every((r) => selected.has(r.run_id));
  const someSelected = paginated.some((r) => selected.has(r.run_id));

  return (
    <div className="h-full flex flex-col pl-2">
      {/* Pane header */}
      <div className="card-header flex items-center justify-between gap-2 flex-shrink-0">
        <div className="min-w-0">
          <span className="font-medium truncate block">
            {testId ? `Runs · ${selectedTest?.name ?? testId}` : 'All test runs'}
          </span>
          <span className="text-xs text-[var(--color-text-tertiary)]">
            {total} run{total === 1 ? '' : 's'}
          </span>
        </div>
        {isLoggedIn && (
          <button
            type="button"
            onClick={() => setShowStartModal(true)}
            className="btn btn-primary btn-sm flex items-center gap-1.5 shrink-0"
          >
            <PlayIcon className="size-4" />
            Start test
          </button>
        )}
      </div>

      <div className="flex-1 overflow-auto">
        {isLoading ? (
          <div className="flex items-center justify-center h-32">
            <div className="animate-spin rounded-full size-6 border-b-2 border-primary-600" />
          </div>
        ) : error ? (
          <p className="p-4 text-error-600 text-sm">{error.message}</p>
        ) : total === 0 ? (
          <p className="p-6 text-center text-sm text-[var(--color-text-secondary)]">
            {testId ? 'No runs for this test yet.' : 'No test runs yet.'}
          </p>
        ) : (
          <table className="table">
            <thead className="sticky top-0 bg-[var(--color-bg-primary)] z-10">
              <tr>
                <th className="w-8">
                  <input
                    type="checkbox"
                    checked={allSelected}
                    ref={(el) => {
                      if (el) el.indeterminate = someSelected && !allSelected;
                    }}
                    onChange={(e) => handleSelectAll(e.target.checked)}
                  />
                </th>
                <th className="w-20">Run</th>
                {!testId && <th>Test</th>}
                <th className="w-44">Started</th>
                <th className="w-28">Duration</th>
                <th className="w-24">Status</th>
                <th className="w-20" />
              </tr>
            </thead>
            <tbody>
              {paginated.map((run) => (
                <RunRow
                  key={run.run_id}
                  run={run}
                  showTestName={!testId}
                  isSelected={selected.has(run.run_id)}
                  canMutate={isLoggedIn}
                  onSelect={handleSelectOne}
                  onCancel={handleCancel}
                  onDelete={(id) => handleDelete([id])}
                />
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Footer: bulk actions + pagination */}
      <div className="flex items-center justify-between gap-2 p-2 border-t border-[var(--color-border)] text-sm flex-shrink-0">
        {isLoggedIn ? (
          <button
            type="button"
            onClick={() => handleDelete(Array.from(selected))}
            disabled={selected.size === 0 || deleteMutation.isPending}
            className="btn btn-secondary btn-sm disabled:opacity-50"
          >
            Delete selected ({selected.size})
          </button>
        ) : (
          <span />
        )}

        <Pagination page={page} totalPages={totalPages} onChange={setPage} />
      </div>

      <StartTestModal
        isOpen={showStartModal}
        onClose={() => setShowStartModal(false)}
        initialTestId={testId ?? undefined}
      />
    </div>
  );
}

interface RunRowProps {
  run: TestRun;
  showTestName: boolean;
  isSelected: boolean;
  canMutate: boolean;
  onSelect: (id: number, checked: boolean) => void;
  onCancel: (run: TestRun) => void;
  onDelete: (id: number) => void;
}

function RunRow({
  run,
  showTestName,
  isSelected,
  canMutate,
  onSelect,
  onCancel,
  onDelete,
}: RunRowProps) {
  const duration =
    run.stop_time > 0 && run.start_time > 0
      ? run.stop_time - run.start_time
      : run.start_time > 0
        ? Math.floor(Date.now() / 1000) - run.start_time
        : 0;

  const canCancel = canMutate && (run.status === 'pending' || run.status === 'running');

  return (
    <tr className="hover:bg-[var(--color-bg-tertiary)]">
      <td>
        <input
          type="checkbox"
          checked={isSelected}
          onChange={(e) => onSelect(run.run_id, e.target.checked)}
        />
      </td>
      <td>
        <Link to={`/run/${run.run_id}`} className="font-mono text-primary-600 hover:underline">
          #{run.run_id}
        </Link>
      </td>
      {showTestName && (
        <td className="truncate max-w-xs" title={run.name}>
          <Link
            to={`/test/${encodeURIComponent(run.test_id)}`}
            className="text-primary-600 hover:underline"
          >
            {run.name}
          </Link>
        </td>
      )}
      <td className="text-sm whitespace-nowrap">
        {run.start_time > 0 ? new Date(run.start_time * 1000).toLocaleString() : '–'}
      </td>
      <td className="text-sm font-mono whitespace-nowrap">
        {duration > 0 ? formatDuration(duration) : '–'}
      </td>
      <td>
        <StatusBadge status={run.status} size="sm" />
      </td>
      <td>
        <Dropdown
          trigger={
            <button className="p-1.5 hover:bg-[var(--color-bg-tertiary)] rounded">
              <DotsIcon className="size-4" />
            </button>
          }
        >
          <Link
            to={`/run/${run.run_id}`}
            className="block px-4 py-2 text-sm hover:bg-[var(--color-bg-tertiary)]"
          >
            View details
          </Link>
          {canCancel && (
            <button
              onClick={() => onCancel(run)}
              className="w-full text-left px-4 py-2 text-sm hover:bg-[var(--color-bg-tertiary)] text-error-600"
            >
              Cancel
            </button>
          )}
          {canMutate && (
            <button
              onClick={() => onDelete(run.run_id)}
              className="w-full text-left px-4 py-2 text-sm hover:bg-[var(--color-bg-tertiary)] text-error-600"
            >
              Delete
            </button>
          )}
        </Dropdown>
      </td>
    </tr>
  );
}

function Pagination({
  page,
  totalPages,
  onChange,
}: {
  page: number;
  totalPages: number;
  onChange: (page: number) => void;
}) {
  if (totalPages <= 1) return <span />;
  return (
    <div className="flex items-center gap-1">
      <button
        onClick={() => onChange(Math.max(1, page - 1))}
        disabled={page <= 1}
        className="p-1 rounded hover:bg-[var(--color-bg-tertiary)] disabled:opacity-40"
      >
        ‹
      </button>
      <span className="text-xs px-2">
        {page} / {totalPages}
      </span>
      <button
        onClick={() => onChange(Math.min(totalPages, page + 1))}
        disabled={page >= totalPages}
        className="p-1 rounded hover:bg-[var(--color-bg-tertiary)] disabled:opacity-40"
      >
        ›
      </button>
    </div>
  );
}

function PlayIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
      />
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
      />
    </svg>
  );
}

function DotsIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M12 5v.01M12 12v.01M12 19v.01"
      />
    </svg>
  );
}

export default Runs;
