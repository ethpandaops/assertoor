import { useMemo, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useTests, useTestRuns } from '../hooks/useApi';
import { useAuthContext } from '../context/AuthContext';
import StatusBadge from '../components/common/StatusBadge';
import StartTestModal from '../components/test/StartTestModal';
import { formatDuration } from '../utils/time';

function TestPage() {
  const { testId } = useParams<{ testId: string }>();
  const { isLoggedIn } = useAuthContext();
  const [showRunModal, setShowRunModal] = useState(false);

  const { data: tests, isLoading: testsLoading } = useTests();
  const { data: allRuns, isLoading: runsLoading, error: runsError } = useTestRuns(testId);

  // Find the test in the registry
  const test = useMemo(() => {
    return tests?.find(t => t.id === testId);
  }, [tests, testId]);

  const isLoading = testsLoading || runsLoading;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full size-8 border-b-2 border-primary-600"></div>
      </div>
    );
  }

  if (!test) {
    return (
      <div className="card p-6 text-center">
        <p className="text-error-600">Test not found: {testId}</p>
        <Link to="/registry" className="text-primary-600 hover:underline mt-4 inline-block">
          Back to Registry
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3">
            <Link
              to="/registry"
              className="text-[var(--color-text-secondary)] hover:text-primary-600"
            >
              <ChevronLeftIcon className="size-5" />
            </Link>
            <h1 className="text-2xl font-bold">{test.name || test.id}</h1>
          </div>
          <p className="mt-1 text-[var(--color-text-secondary)] font-mono text-sm">{test.id}</p>
        </div>

        {isLoggedIn && (
          <button
            onClick={() => setShowRunModal(true)}
            className="btn btn-primary"
          >
            Run Test
          </button>
        )}
      </div>

      {/* Test details */}
      <div className="card p-4">
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="text-[var(--color-text-secondary)]">Source:</span>{' '}
            <span className="px-2 py-0.5 bg-[var(--color-bg-tertiary)] rounded-sm text-xs">
              {test.source}
            </span>
          </div>
          <div>
            <span className="text-[var(--color-text-secondary)]">Total Runs:</span>{' '}
            <span className="font-mono">{allRuns?.length ?? 0}</span>
          </div>
          {test.basePath && (
            <div className="col-span-2">
              <span className="text-[var(--color-text-secondary)]">Base Path:</span>{' '}
              <span className="font-mono text-xs">{test.basePath}</span>
            </div>
          )}
        </div>
      </div>

      {/* Run history */}
      <div className="card overflow-hidden">
        <div className="card-header">Recent Runs</div>
        {runsError ? (
          <div className="p-6 text-center text-error-600">
            Failed to load runs: {runsError.message}
          </div>
        ) : !allRuns || allRuns.length === 0 ? (
          <div className="p-6 text-center text-[var(--color-text-secondary)]">
            No recent runs for this test.
          </div>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th className="w-24">Run ID</th>
                <th className="w-28">Status</th>
                <th>Started</th>
                <th className="w-32">Duration</th>
              </tr>
            </thead>
            <tbody>
              {allRuns.map((run) => {
                const duration = run.stop_time > 0 && run.start_time > 0
                  ? run.stop_time - run.start_time
                  : run.start_time > 0
                  ? Math.floor(Date.now() / 1000) - run.start_time
                  : 0;

                return (
                  <tr key={run.run_id}>
                    <td>
                      <Link
                        to={`/run/${run.run_id}`}
                        className="text-primary-600 hover:underline font-mono"
                      >
                        #{run.run_id}
                      </Link>
                    </td>
                    <td>
                      <StatusBadge status={run.status} />
                    </td>
                    <td className="text-sm">
                      {run.start_time > 0
                        ? new Date(run.start_time * 1000).toLocaleString()
                        : '-'}
                    </td>
                    <td className="font-mono text-sm">
                      {duration > 0 ? formatDuration(duration) : '-'}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {/* Run Test Modal */}
      <StartTestModal
        isOpen={showRunModal}
        onClose={() => setShowRunModal(false)}
        initialTestId={testId}
      />
    </div>
  );
}

function ChevronLeftIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
    </svg>
  );
}

export default TestPage;
