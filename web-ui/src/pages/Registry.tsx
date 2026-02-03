import { useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { useTests, useDeleteTest, useRegisterTest, useRegisterExternalTest } from '../hooks/useApi';
import { useAuthContext } from '../context/AuthContext';
import Modal from '../components/common/Modal';
import StartTestModal from '../components/test/StartTestModal';
import type { Test } from '../types/api';

function Registry() {
  const { isLoggedIn } = useAuthContext();
  const { data: tests, isLoading, error } = useTests();
  const deleteMutation = useDeleteTest();
  const [showRegisterModal, setShowRegisterModal] = useState(false);
  const [runTestId, setRunTestId] = useState<string | null>(null);

  const handleSchedule = useCallback((testId: string) => {
    setRunTestId(testId);
  }, []);

  const handleDelete = useCallback(async (testId: string) => {
    if (!confirm(`Delete test ${testId}? This action cannot be undone.`)) return;
    try {
      await deleteMutation.mutateAsync(testId);
    } catch (err) {
      alert(`Failed to delete test: ${err}`);
    }
  }, [deleteMutation]);

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
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Test Registry</h1>
        <div className="flex items-center gap-3">
          <div className="text-sm text-[var(--color-text-secondary)]">
            {tests?.length ?? 0} tests registered
          </div>
          {isLoggedIn && (
            <>
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
            </>
          )}
        </div>
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

      {/* Register Test Modal */}
      <RegisterTestModal
        isOpen={showRegisterModal}
        onClose={() => setShowRegisterModal(false)}
      />

      {/* Run Test Modal */}
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
  canDelete: boolean;
  onSchedule: (testId: string) => void;
  onDelete: (testId: string) => void;
}

function TestRow({ test, canStart, canDelete, onSchedule, onDelete }: TestRowProps) {
  const [expanded, setExpanded] = useState(false);

  return (
    <>
      <tr
        className="cursor-pointer hover:bg-[var(--color-bg-tertiary)]"
        onClick={() => setExpanded(!expanded)}
      >
        <td>
          <div className="flex items-center gap-2">
            <ChevronIcon className={`size-4 transition-transform ${expanded ? 'rotate-90' : ''}`} />
            <div>
              <div className="font-medium">{test.name || test.id}</div>
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
          <td colSpan={3} className="bg-[var(--color-bg-secondary)] p-4">
            <div className="grid grid-cols-2 gap-4 text-sm">
              {test.basePath && (
                <div>
                  <span className="text-[var(--color-text-secondary)]">Base Path:</span>{' '}
                  <span className="font-mono text-xs">{test.basePath}</span>
                </div>
              )}
              <div>
                <span className="text-[var(--color-text-secondary)]">Test ID:</span>{' '}
                <span className="font-mono text-xs">{test.id}</span>
              </div>
            </div>
          </td>
        </tr>
      )}
    </>
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
        await registerExternalMutation.mutateAsync(externalUrl);
      }

      // Success - close modal and reset form
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
        {/* Tabs */}
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
              <label className="block text-sm font-medium mb-1">
                Test Configuration (YAML)
              </label>
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
              <label className="block text-sm font-medium mb-1">
                External Test URL
              </label>
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
            <button
              type="submit"
              className="btn btn-primary btn-sm"
              disabled={isSubmitting}
            >
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

export default Registry;
