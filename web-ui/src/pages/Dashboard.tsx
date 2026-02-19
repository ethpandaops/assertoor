import { useState, useCallback, useMemo, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { useTestRuns, useCancelTestRun, useDeleteTestRuns } from '../hooks/useApi';
import { useAuthContext } from '../context/AuthContext';
import StatusBadge from '../components/common/StatusBadge';
import Dropdown from '../components/common/Dropdown';
import StartTestModal from '../components/test/StartTestModal';
import { formatDuration } from '../utils/time';
import type { TestRun } from '../types/api';

const PAGE_SIZE_KEY = 'dashboard-page-size';
const PAGE_SIZES = [25, 50, 100, 200] as const;

function Dashboard() {
  const [selectedRuns, setSelectedRuns] = useState<Set<number>>(new Set());
  const [expandedRun, setExpandedRun] = useState<number | null>(null);
  const [showStartTestModal, setShowStartTestModal] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState<number>(() => {
    const stored = localStorage.getItem(PAGE_SIZE_KEY);
    return stored ? parseInt(stored, 10) : 25;
  });

  const { isLoggedIn } = useAuthContext();
  const { data: testRuns, isLoading, error } = useTestRuns();
  const cancelMutation = useCancelTestRun();
  const deleteMutation = useDeleteTestRuns();

  // Store page size preference
  useEffect(() => {
    localStorage.setItem(PAGE_SIZE_KEY, String(pageSize));
  }, [pageSize]);

  // Calculate pagination
  const totalItems = testRuns?.length ?? 0;
  const totalPages = Math.ceil(totalItems / pageSize);

  // Ensure current page is valid when data changes
  useEffect(() => {
    if (currentPage > totalPages && totalPages > 0) {
      setCurrentPage(totalPages);
    }
  }, [currentPage, totalPages]);

  // Get current page items
  const paginatedRuns = useMemo(() => {
    if (!testRuns) return [];
    const startIndex = (currentPage - 1) * pageSize;
    return testRuns.slice(startIndex, startIndex + pageSize);
  }, [testRuns, currentPage, pageSize]);

  // Handle page size change
  const handlePageSizeChange = useCallback((newSize: number) => {
    setPageSize(newSize);
    setCurrentPage(1);
    setSelectedRuns(new Set());
  }, []);

  // Handle select all - only select items on current page
  const handleSelectAll = useCallback((checked: boolean) => {
    if (checked) {
      setSelectedRuns(new Set(paginatedRuns.map(t => t.run_id)));
    } else {
      setSelectedRuns(new Set());
    }
  }, [paginatedRuns]);

  const handleSelectRun = useCallback((runId: number, checked: boolean) => {
    setSelectedRuns(prev => {
      const next = new Set(prev);
      if (checked) {
        next.add(runId);
      } else {
        next.delete(runId);
      }
      return next;
    });
  }, []);

  const handleCancel = useCallback(async (run: TestRun) => {
    if (!confirm(`Cancel test run ${run.run_id}?`)) return;
    try {
      await cancelMutation.mutateAsync(run.run_id);
    } catch (err) {
      alert(`Failed to cancel: ${err}`);
    }
  }, [cancelMutation]);

  const handleDelete = useCallback(async (runIds: number[]) => {
    if (!confirm(`Delete ${runIds.length} test run(s)?`)) return;
    try {
      await deleteMutation.mutateAsync(runIds);
      setSelectedRuns(new Set());
    } catch (err) {
      alert(`Failed to delete: ${err}`);
    }
  }, [deleteMutation]);

  const handleRowClick = useCallback((runId: number, e: React.MouseEvent) => {
    const target = e.target as HTMLElement;
    if (target.closest('button') || target.closest('a') || target.closest('input')) {
      return;
    }
    setExpandedRun(prev => prev === runId ? null : runId);
  }, []);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="card p-6 text-center">
        <p className="text-error-600">Failed to load test runs: {error.message}</p>
      </div>
    );
  }

  const allSelected = paginatedRuns.length > 0 &&
    paginatedRuns.every(t => selectedRuns.has(t.run_id));
  const someSelected = paginatedRuns.some(t => selectedRuns.has(t.run_id));

  const startItem = totalItems === 0 ? 0 : (currentPage - 1) * pageSize + 1;
  const endItem = Math.min(currentPage * pageSize, totalItems);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">All Test Runs</h1>
        <div className="flex items-center gap-3">
          <span className="text-sm text-[var(--color-text-secondary)]">
            {totalItems} test runs
          </span>
          {isLoggedIn && (
            <button
              onClick={() => setShowStartTestModal(true)}
              className="btn btn-primary btn-sm flex items-center gap-1.5"
            >
              <PlayIcon className="size-4" />
              Start Test
            </button>
          )}
        </div>
      </div>

      {testRuns && testRuns.length === 0 ? (
        <div className="card p-12 text-center">
          <p className="text-[var(--color-text-secondary)]">No test runs yet.</p>
          <p className="text-sm mt-2">
            <Link to="/registry" className="text-primary-600 hover:underline">
              Go to Registry
            </Link>{' '}
            to schedule a test run.
          </p>
        </div>
      ) : (
        <>
          <div className="card">
            <div className="overflow-x-auto">
              <table className="table">
                <thead>
                  <tr>
                    <th className="w-10">
                      <input
                        type="checkbox"
                        checked={allSelected}
                        ref={(el) => {
                          if (el) el.indeterminate = someSelected && !allSelected;
                        }}
                        onChange={(e) => handleSelectAll(e.target.checked)}
                        className="rounded border-gray-300"
                      />
                    </th>
                    <th className="w-24">Run ID</th>
                    <th>Test Name</th>
                    <th className="w-48">Start Time</th>
                    <th className="w-40">Duration</th>
                    <th className="w-28">Status</th>
                    <th className="w-24">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {paginatedRuns.map((run) => (
                    <TestRunRow
                      key={run.run_id}
                      run={run}
                      isSelected={selectedRuns.has(run.run_id)}
                      isExpanded={expandedRun === run.run_id}
                      canCancel={isLoggedIn}
                      canDelete={isLoggedIn}
                      onSelect={handleSelectRun}
                      onCancel={handleCancel}
                      onDelete={(id) => handleDelete([id])}
                      onRowClick={handleRowClick}
                    />
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Pagination Controls */}
          <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              {isLoggedIn && (
                <button
                  onClick={() => handleDelete(Array.from(selectedRuns))}
                  disabled={selectedRuns.size === 0 || deleteMutation.isPending}
                  className="btn btn-secondary btn-sm disabled:opacity-50"
                >
                  Delete Selected ({selectedRuns.size})
                </button>
              )}
            </div>

            <div className="flex items-center gap-4">
              {/* Page size selector */}
              <div className="flex items-center gap-2">
                <span className="text-sm text-[var(--color-text-secondary)]">Show</span>
                <select
                  value={pageSize}
                  onChange={(e) => handlePageSizeChange(Number(e.target.value))}
                  className="rounded border border-[var(--color-border)] bg-[var(--color-bg-secondary)] px-2 py-1 text-sm"
                >
                  {PAGE_SIZES.map(size => (
                    <option key={size} value={size}>{size}</option>
                  ))}
                </select>
                <span className="text-sm text-[var(--color-text-secondary)]">per page</span>
              </div>

              {/* Page info */}
              <span className="text-sm text-[var(--color-text-secondary)]">
                {startItem}-{endItem} of {totalItems}
              </span>

              {/* Page navigation */}
              <div className="flex items-center gap-1">
                <button
                  onClick={() => setCurrentPage(1)}
                  disabled={currentPage === 1}
                  className="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] disabled:opacity-50 disabled:cursor-not-allowed"
                  title="First page"
                >
                  <FirstPageIcon className="size-4" />
                </button>
                <button
                  onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                  disabled={currentPage === 1}
                  className="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] disabled:opacity-50 disabled:cursor-not-allowed"
                  title="Previous page"
                >
                  <ChevronLeftIcon className="size-4" />
                </button>
                <span className="px-2 text-sm">
                  Page {currentPage} of {totalPages || 1}
                </span>
                <button
                  onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                  disabled={currentPage >= totalPages}
                  className="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] disabled:opacity-50 disabled:cursor-not-allowed"
                  title="Next page"
                >
                  <ChevronRightIcon className="size-4" />
                </button>
                <button
                  onClick={() => setCurrentPage(totalPages)}
                  disabled={currentPage >= totalPages}
                  className="p-1.5 rounded hover:bg-[var(--color-bg-tertiary)] disabled:opacity-50 disabled:cursor-not-allowed"
                  title="Last page"
                >
                  <LastPageIcon className="size-4" />
                </button>
              </div>
            </div>
          </div>
        </>
      )}

      {/* Start Test Modal */}
      <StartTestModal
        isOpen={showStartTestModal}
        onClose={() => setShowStartTestModal(false)}
      />
    </div>
  );
}

interface TestRunRowProps {
  run: TestRun;
  isSelected: boolean;
  isExpanded: boolean;
  canCancel: boolean;
  canDelete: boolean;
  onSelect: (runId: number, checked: boolean) => void;
  onCancel: (run: TestRun) => void;
  onDelete: (runId: number) => void;
  onRowClick: (runId: number, e: React.MouseEvent) => void;
}

function TestRunRow({
  run,
  isSelected,
  isExpanded,
  canCancel,
  canDelete,
  onSelect,
  onCancel,
  onDelete,
  onRowClick,
}: TestRunRowProps) {
  const canCancelRun = canCancel && (run.status === 'pending' || run.status === 'running');
  const duration = run.stop_time > 0 && run.start_time > 0
    ? run.stop_time - run.start_time
    : run.start_time > 0
    ? Math.floor(Date.now() / 1000) - run.start_time
    : 0;

  return (
    <>
      <tr
        className="cursor-pointer hover:bg-[var(--color-bg-tertiary)]"
        onClick={(e) => onRowClick(run.run_id, e)}
      >
        <td onClick={(e) => e.stopPropagation()}>
          <input
            type="checkbox"
            checked={isSelected}
            onChange={(e) => onSelect(run.run_id, e.target.checked)}
            className="rounded border-gray-300"
          />
        </td>
        <td className="font-mono">{run.run_id}</td>
        <td>
          <Link
            to={`/run/${run.run_id}`}
            className="text-primary-600 hover:underline font-medium"
            onClick={(e) => e.stopPropagation()}
          >
            {run.name}
          </Link>
        </td>
        <td className="text-sm">
          {run.start_time > 0 ? new Date(run.start_time * 1000).toLocaleString() : '-'}
        </td>
        <td className="text-sm font-mono">
          {duration > 0 ? formatDuration(duration) : '-'}
        </td>
        <td>
          <StatusBadge status={run.status} />
        </td>
        <td onClick={(e) => e.stopPropagation()}>
          <div className="flex items-center gap-1">
            <Link
              to={`/run/${run.run_id}`}
              className="p-1.5 hover:bg-[var(--color-bg-tertiary)] rounded"
              title="View details"
            >
              <EyeIcon className="w-4 h-4" />
            </Link>
            <Dropdown
              trigger={
                <button className="p-1.5 hover:bg-[var(--color-bg-tertiary)] rounded">
                  <DotsIcon className="w-4 h-4" />
                </button>
              }
            >
              <Link
                to={`/run/${run.run_id}`}
                className="block px-4 py-2 text-sm hover:bg-[var(--color-bg-tertiary)]"
              >
                View Test Details
              </Link>
              <hr className="border-[var(--color-border)]" />
              {canCancelRun && (
                <button
                  onClick={() => onCancel(run)}
                  className="w-full text-left px-4 py-2 text-sm hover:bg-[var(--color-bg-tertiary)] text-error-600"
                >
                  Cancel Test Run
                </button>
              )}
              {canDelete && (
                <button
                  onClick={() => onDelete(run.run_id)}
                  className="w-full text-left px-4 py-2 text-sm hover:bg-[var(--color-bg-tertiary)] text-error-600"
                >
                  Delete Test Run
                </button>
              )}
            </Dropdown>
          </div>
        </td>
      </tr>
      {isExpanded && (
        <tr>
          <td colSpan={7} className="bg-[var(--color-bg-secondary)] p-4">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-[var(--color-text-secondary)]">Test ID:</span>{' '}
                {run.test_id}
              </div>
              <div>
                <span className="text-[var(--color-text-secondary)]">Status:</span>{' '}
                <StatusBadge status={run.status} />
                {canCancelRun && (
                  <button
                    onClick={() => onCancel(run)}
                    className="ml-2 btn btn-danger btn-sm"
                  >
                    Cancel
                  </button>
                )}
              </div>
              {run.start_time > 0 && (
                <div>
                  <span className="text-[var(--color-text-secondary)]">Start Time:</span>{' '}
                  {new Date(run.start_time * 1000).toLocaleString()}
                </div>
              )}
              {run.stop_time > 0 && (
                <div>
                  <span className="text-[var(--color-text-secondary)]">Stop Time:</span>{' '}
                  {new Date(run.stop_time * 1000).toLocaleString()}
                </div>
              )}
              <div>
                <Link to={`/run/${run.run_id}`} className="text-primary-600 hover:underline">
                  View Test Details
                </Link>
              </div>
            </div>
          </td>
        </tr>
      )}
    </>
  );
}

function EyeIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
    </svg>
  );
}

function DotsIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
    </svg>
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

function ChevronLeftIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
    </svg>
  );
}

function ChevronRightIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
    </svg>
  );
}

function FirstPageIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 19l-7-7 7-7M18 19l-7-7 7-7" />
    </svg>
  );
}

function LastPageIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 5l7 7-7 7M6 5l7 7-7 7" />
    </svg>
  );
}

export default Dashboard;
