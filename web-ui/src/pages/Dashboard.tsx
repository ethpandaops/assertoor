import { useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { useTestRuns, useCancelTestRun, useDeleteTestRuns } from '../hooks/useApi';
import { useAuthContext } from '../context/AuthContext';
import StatusBadge from '../components/common/StatusBadge';
import Dropdown from '../components/common/Dropdown';
import StartTestModal from '../components/test/StartTestModal';
import { formatDuration } from '../utils/time';
import type { TestRun } from '../types/api';

function Dashboard() {
  const [selectedRuns, setSelectedRuns] = useState<Set<number>>(new Set());
  const [expandedRun, setExpandedRun] = useState<number | null>(null);
  const [showStartTestModal, setShowStartTestModal] = useState(false);

  const { isLoggedIn } = useAuthContext();
  const { data: testRuns, isLoading, error } = useTestRuns();
  const cancelMutation = useCancelTestRun();
  const deleteMutation = useDeleteTestRuns();

  const handleSelectAll = useCallback((checked: boolean) => {
    if (checked && testRuns) {
      setSelectedRuns(new Set(testRuns.map(t => t.run_id)));
    } else {
      setSelectedRuns(new Set());
    }
  }, [testRuns]);

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

  const allSelected = testRuns && testRuns.length > 0 &&
    testRuns.every(t => selectedRuns.has(t.run_id));

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">All Test Runs</h1>
        <div className="flex items-center gap-3">
          <span className="text-sm text-[var(--color-text-secondary)]">
            {testRuns?.length ?? 0} test runs
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
                  {testRuns?.map((run) => (
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

          {isLoggedIn && (
            <div className="flex items-center gap-2">
              <button
                onClick={() => handleDelete(Array.from(selectedRuns))}
                disabled={selectedRuns.size === 0 || deleteMutation.isPending}
                className="btn btn-secondary btn-sm disabled:opacity-50"
              >
                Delete Selected ({selectedRuns.size})
              </button>
            </div>
          )}
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

export default Dashboard;
