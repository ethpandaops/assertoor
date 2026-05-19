import { useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import {
  useTests,
  useDeleteTest,
  useRegisterTest,
  useRegisterExternalTest,
  useTestDetails,
  useTestRuns,
} from '../../hooks/useApi';
import { useAuthContext } from '../../context/AuthContext';
import Modal from '../common/Modal';
import StartTestModal from '../test/StartTestModal';
import StatusBadge from '../common/StatusBadge';
import ScheduleCard from '../schedule/ScheduleCard';
import { formatDuration } from '../../utils/time';
import type { Test } from '../../types/api';

// MyTestsTab is the previous Registry view, extracted unchanged so the
// new tabbed Test Library page can host it. Behaviour is identical to
// the old Registry page.
function MyTestsTab() {
  const { isLoggedIn } = useAuthContext();
  const { data: tests, isLoading, error } = useTests();
  const deleteMutation = useDeleteTest();
  const [showRegisterModal, setShowRegisterModal] = useState(false);
  const [runTestId, setRunTestId] = useState<string | null>(null);

  const handleSchedule = useCallback((testId: string) => {
    setRunTestId(testId);
  }, []);

  const handleDelete = useCallback(
    async (testId: string) => {
      if (!confirm(`Delete test ${testId}? This action cannot be undone.`)) return;
      try {
        await deleteMutation.mutateAsync(testId);
      } catch (err) {
        alert(`Failed to delete test: ${err}`);
      }
    },
    [deleteMutation],
  );

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full size-8 border-b-2 border-primary-600"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="card p-6 text-center">
        <p className="text-error-600">Failed to load registry: {error.message}</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="text-sm text-[var(--color-text-secondary)]">
          {tests?.length ?? 0} tests registered
        </div>
        {isLoggedIn && (
          <div className="flex items-center gap-3">
            <Link
              to="/builder"
              className="btn btn-secondary btn-sm flex items-center gap-1.5"
            >
              <BuilderIcon className="size-4" />
              Build Test
            </Link>
            <button
              onClick={() => setShowRegisterModal(true)}
              className="btn btn-primary btn-sm flex items-center gap-1.5"
            >
              <PlusIcon className="size-4" />
              Register Test
            </button>
          </div>
        )}
      </div>

      {tests && tests.length === 0 ? (
        <div className="card p-12 text-center">
          <p className="text-[var(--color-text-secondary)]">No tests registered.</p>
          <p className="text-sm mt-2">
            {isLoggedIn ? (
              <button
                onClick={() => setShowRegisterModal(true)}
                className="text-primary-600 hover:underline"
              >
                Register a test
              </button>
            ) : (
              'Log in to register tests.'
            )}
          </p>
        </div>
      ) : (
        <div className="card overflow-hidden">
          <div className="overflow-x-auto">
            <table className="table">
              <thead>
                <tr>
                  <th>Test Name</th>
                  <th className="w-32">Source</th>
                  <th className="w-32">Actions</th>
                </tr>
              </thead>
              <tbody>
                {tests?.map((test) => (
                  <TestRow
                    key={test.id}
                    test={test}
                    canStart={isLoggedIn}
                    canEdit={isLoggedIn}
                    canDelete={isLoggedIn}
                    onSchedule={handleSchedule}
                    onDelete={handleDelete}
                  />
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <RegisterTestModal
        isOpen={showRegisterModal}
        onClose={() => setShowRegisterModal(false)}
      />

      <StartTestModal
        isOpen={runTestId !== null}
        onClose={() => setRunTestId(null)}
        initialTestId={runTestId}
      />
    </div>
  );
}

interface TestRowProps {
  test: Test;
  canStart: boolean;
  canEdit: boolean;
  canDelete: boolean;
  onSchedule: (testId: string) => void;
  onDelete: (testId: string) => void;
}

function TestRow({ test, canStart, canEdit, canDelete, onSchedule, onDelete }: TestRowProps) {
  const [expanded, setExpanded] = useState(false);
  const isDev = (test.version ?? '').startsWith('0.');

  return (
    <>
      <tr
        className="cursor-pointer hover:bg-[var(--color-bg-tertiary)]"
        onClick={() => setExpanded(!expanded)}
      >
        <td>
          <div className="flex items-center gap-2">
            <ChevronIcon className={`size-4 transition-transform ${expanded ? 'rotate-90' : ''}`} />
            <div className="min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <span className="font-medium">{test.name || test.id}</span>
                {test.version && (
                  <span
                    className={`px-1.5 py-0.5 rounded-xs text-xs font-medium ${
                      isDev
                        ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/40 dark:text-yellow-200'
                        : 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200'
                    }`}
                  >
                    v{test.version}
                  </span>
                )}
                {test.tags && test.tags.length > 0 && (
                  <span className="text-xs text-[var(--color-text-tertiary)]">
                    {test.tags.slice(0, 3).join(' · ')}
                    {test.tags.length > 3 && ` · +${test.tags.length - 3}`}
                  </span>
                )}
              </div>
              <div className="text-xs text-[var(--color-text-tertiary)]">{test.id}</div>
            </div>
          </div>
        </td>
        <td className="text-sm">
          <span className="px-2 py-0.5 bg-[var(--color-bg-tertiary)] rounded-xs text-xs">
            {test.source}
          </span>
        </td>
        <td onClick={(e) => e.stopPropagation()}>
          <div className="flex items-center gap-1">
            {canEdit && (
              <Link
                to={`/builder?testId=${encodeURIComponent(test.id)}`}
                className="p-1.5 hover:bg-[var(--color-bg-tertiary)] rounded-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
                title="Edit in builder"
              >
                <EditIcon className="size-4" />
              </Link>
            )}
            {canStart && (
              <button
                onClick={() => onSchedule(test.id)}
                className="p-1.5 hover:bg-[var(--color-bg-tertiary)] rounded-xs text-primary-600"
                title="Run test"
              >
                <PlayIcon className="size-4" />
              </button>
            )}
            {canDelete && (
              <button
                onClick={() => onDelete(test.id)}
                className="p-1.5 hover:bg-[var(--color-bg-tertiary)] rounded-xs text-error-600"
                title="Delete test"
              >
                <TrashIcon className="size-4" />
              </button>
            )}
          </div>
        </td>
      </tr>
      {expanded && (
        <tr>
          <td
            colSpan={3}
            className="bg-[var(--color-bg-tertiary)] dark:bg-[var(--color-bg-primary)] border-l-2 border-primary-500 p-0"
          >
            <ExpandedTestDetails test={test} />
          </td>
        </tr>
      )}
    </>
  );
}

interface ExpandedTestDetailsProps {
  test: Test;
}

// ExpandedTestDetails lazy-loads the test description + recent runs.
// React Query keeps a 30s cooldown on test runs (no auto-polling) so
// rapid expand/collapse doesn't hammer the API; details are
// effectively immutable for the test's lifetime and cached forever.
function ExpandedTestDetails({ test }: ExpandedTestDetailsProps) {
  const detailsQuery = useTestDetails(test.id, { enabled: true });
  const runsQuery = useTestRuns(test.id, {
    enabled: true,
    refetchInterval: false,
    staleTime: 30_000,
  });

  const description = detailsQuery.data?.description ?? test.description;
  const tags = detailsQuery.data?.tags ?? test.tags ?? [];
  const version = detailsQuery.data?.version ?? test.version;
  const timeout = detailsQuery.data?.timeout;
  const runs = runsQuery.data ?? [];
  const recentRuns = runs.slice(0, 5);

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 lg:divide-x divide-[var(--color-border)]">
      <div className="p-4 space-y-3">
        <SectionHeader>Details</SectionHeader>

        {description ? (
          <div className="markdown-body">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{description}</ReactMarkdown>
          </div>
        ) : (
          <p className="text-xs text-[var(--color-text-tertiary)] italic">
            No description provided.
          </p>
        )}

        {tags.length > 0 && (
          <div className="flex flex-wrap items-center gap-1.5 pt-1">
            <span className="text-[10px] font-semibold text-[var(--color-text-tertiary)] uppercase tracking-wider mr-1">
              Tags
            </span>
            {tags.map((tag) => (
              <span
                key={tag}
                className="px-2 py-0.5 bg-[var(--color-bg-secondary)] dark:bg-[var(--color-bg-tertiary)] border border-[var(--color-border)] text-[var(--color-text-secondary)] rounded-xs text-xs"
              >
                {tag}
              </span>
            ))}
          </div>
        )}

        <dl className="grid grid-cols-[max-content_1fr] gap-x-3 gap-y-1 text-xs pt-1">
          <dt className="text-[var(--color-text-tertiary)]">Test ID</dt>
          <dd className="font-mono break-all">{test.id}</dd>
          {version && (
            <>
              <dt className="text-[var(--color-text-tertiary)]">Version</dt>
              <dd className="font-mono">{version}</dd>
            </>
          )}
          {typeof timeout === 'number' && timeout > 0 && (
            <>
              <dt className="text-[var(--color-text-tertiary)]">Timeout</dt>
              <dd className="font-mono">{formatDuration(timeout)}</dd>
            </>
          )}
          {test.basePath && (
            <>
              <dt className="text-[var(--color-text-tertiary)]">Base path</dt>
              <dd className="font-mono break-all">{test.basePath}</dd>
            </>
          )}
        </dl>

        <div className="pt-2">
          <SectionHeader>Schedule</SectionHeader>
          <div className="mt-1">
            <ScheduleCard testId={test.id} />
          </div>
        </div>
      </div>

      <div className="p-4 space-y-2">
        <SectionHeader
          action={
            runs.length > 0 ? (
              <Link
                to={`/test/${encodeURIComponent(test.id)}`}
                className="text-xs text-primary-600 hover:underline"
              >
                View all ({runs.length})
              </Link>
            ) : null
          }
        >
          Recent runs
        </SectionHeader>

        {runsQuery.isLoading ? (
          <p className="text-xs text-[var(--color-text-tertiary)]">Loading…</p>
        ) : runsQuery.error ? (
          <p className="text-xs text-error-600">Failed to load runs: {runsQuery.error.message}</p>
        ) : runs.length === 0 ? (
          <p className="text-xs text-[var(--color-text-tertiary)] italic">
            This test has not run yet.
          </p>
        ) : (
          <ul className="divide-y divide-[var(--color-border)] rounded border border-[var(--color-border)] bg-[var(--color-bg-secondary)] dark:bg-[var(--color-bg-secondary)] overflow-hidden">
            {recentRuns.map((run) => {
              const duration =
                run.stop_time > 0 && run.start_time > 0
                  ? run.stop_time - run.start_time
                  : run.start_time > 0
                    ? Math.floor(Date.now() / 1000) - run.start_time
                    : 0;
              return (
                <li
                  key={run.run_id}
                  className="flex items-center gap-2 px-2 py-1.5 hover:bg-[var(--color-bg-tertiary)] text-xs"
                >
                  <StatusBadge status={run.status} />
                  <Link
                    to={`/run/${run.run_id}`}
                    className="font-mono text-primary-600 hover:underline"
                  >
                    #{run.run_id}
                  </Link>
                  <span className="text-[var(--color-text-secondary)] flex-1 truncate">
                    {run.start_time > 0
                      ? new Date(run.start_time * 1000).toLocaleString()
                      : 'not started'}
                  </span>
                  {duration > 0 && (
                    <span className="font-mono text-[var(--color-text-tertiary)]">
                      {formatDuration(duration)}
                    </span>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </div>
  );
}

// SectionHeader is a small uppercase title with an optional trailing
// action (e.g. a "view all" link). Used to demarcate sections inside
// the flat expanded drawer without adding more boxes.
function SectionHeader({
  children,
  action,
}: {
  children: React.ReactNode;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between pb-1.5 mb-1 border-b border-[var(--color-border)]">
      <h4 className="text-[11px] font-semibold text-[var(--color-text-tertiary)] uppercase tracking-wider">
        {children}
      </h4>
      {action}
    </div>
  );
}

interface RegisterTestModalProps {
  isOpen: boolean;
  onClose: () => void;
}

function RegisterTestModal({ isOpen, onClose }: RegisterTestModalProps) {
  const [activeTab, setActiveTab] = useState<'yaml' | 'url'>('yaml');
  const [yamlContent, setYamlContent] = useState('');
  const [externalUrl, setExternalUrl] = useState('');
  const [error, setError] = useState<string | null>(null);

  const registerTestMutation = useRegisterTest();
  const registerExternalMutation = useRegisterExternalTest();

  const isSubmitting = registerTestMutation.isPending || registerExternalMutation.isPending;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    try {
      if (activeTab === 'yaml') {
        if (!yamlContent.trim()) {
          setError('Please enter YAML content');
          return;
        }
        await registerTestMutation.mutateAsync(yamlContent);
      } else {
        if (!externalUrl.trim()) {
          setError('Please enter a URL');
          return;
        }
        await registerExternalMutation.mutateAsync({ url: externalUrl });
      }

      setYamlContent('');
      setExternalUrl('');
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to register test');
    }
  };

  const handleClose = () => {
    setError(null);
    onClose();
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Register Test" size="lg">
      <div className="space-y-4">
        <div className="flex border-b border-[var(--color-border)]">
          <button
            onClick={() => setActiveTab('yaml')}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === 'yaml'
                ? 'border-primary-600 text-primary-600'
                : 'border-transparent text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]'
            }`}
          >
            YAML Content
          </button>
          <button
            onClick={() => setActiveTab('url')}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === 'url'
                ? 'border-primary-600 text-primary-600'
                : 'border-transparent text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]'
            }`}
          >
            External URL
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {activeTab === 'yaml' ? (
            <div>
              <label className="block text-sm font-medium mb-1">Test Configuration (YAML)</label>
              <textarea
                value={yamlContent}
                onChange={(e) => setYamlContent(e.target.value)}
                placeholder={`id: my-test
name: My Test
tasks:
  - name: sleep
    title: Wait 10 seconds
    config:
      duration: 10s`}
                className="w-full h-64 px-3 py-2 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-xs font-mono text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent resize-none"
                disabled={isSubmitting}
              />
              <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
                Paste the full test configuration in YAML format
              </p>
            </div>
          ) : (
            <div>
              <label className="block text-sm font-medium mb-1">External Test URL</label>
              <input
                type="url"
                value={externalUrl}
                onChange={(e) => setExternalUrl(e.target.value)}
                placeholder="https://raw.githubusercontent.com/.../test.yaml"
                className="w-full px-3 py-2 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-xs text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                disabled={isSubmitting}
              />
              <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
                URL to a YAML test configuration file (e.g., from GitHub raw content)
              </p>
            </div>
          )}

          {error && (
            <div className="p-3 bg-error-50 dark:bg-error-900/20 border border-error-200 dark:border-error-800 rounded-xs">
              <p className="text-sm text-error-600">{error}</p>
            </div>
          )}

          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={handleClose}
              className="btn btn-secondary btn-sm"
              disabled={isSubmitting}
            >
              Cancel
            </button>
            <button type="submit" className="btn btn-primary btn-sm" disabled={isSubmitting}>
              {isSubmitting ? 'Registering...' : 'Register Test'}
            </button>
          </div>
        </form>
      </div>
    </Modal>
  );
}

function ChevronIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
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

function TrashIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
      />
    </svg>
  );
}

function PlusIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
    </svg>
  );
}

function EditIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"
      />
    </svg>
  );
}

function BuilderIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M11 4a2 2 0 114 0v1a1 1 0 001 1h3a1 1 0 011 1v3a1 1 0 01-1 1h-1a2 2 0 100 4h1a1 1 0 011 1v3a1 1 0 01-1 1h-3a1 1 0 01-1-1v-1a2 2 0 10-4 0v1a1 1 0 01-1 1H7a1 1 0 01-1-1v-3a1 1 0 00-1-1H4a2 2 0 110-4h1a1 1 0 001-1V7a1 1 0 011-1h3a1 1 0 001-1V4z"
      />
    </svg>
  );
}

export default MyTestsTab;
