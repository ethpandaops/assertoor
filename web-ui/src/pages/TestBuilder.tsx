import { useEffect, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useBuilderStore } from '../stores/builderStore';
import { useTestDetails, useTaskDescriptors } from '../hooks/useApi';
import { useAuthContext } from '../context/AuthContext';
import BuilderLayout from '../components/builder/BuilderLayout';
import type { TaskDescriptor } from '../types/api';

function TestBuilder() {
  const [searchParams] = useSearchParams();
  const testId = searchParams.get('testId');

  const { isLoggedIn } = useAuthContext();
  const reset = useBuilderStore((state) => state.reset);
  const loadTest = useBuilderStore((state) => state.loadTest);
  const sourceTestId = useBuilderStore((state) => state.sourceTestId);

  // Fetch test details if editing existing test
  const { data: testDetails, isLoading: testLoading, error: testError } = useTestDetails(testId || '', {
    enabled: !!testId,
  });

  // Fetch task descriptors for loading
  const { data: descriptors, isLoading: descriptorsLoading } = useTaskDescriptors();

  // Build descriptor map
  const descriptorMap = useMemo(() => {
    const map = new Map<string, TaskDescriptor>();
    if (descriptors) {
      for (const d of descriptors) {
        map.set(d.name, d);
      }
    }
    return map;
  }, [descriptors]);

  // Load test on initial mount or when testId changes
  useEffect(() => {
    if (testId && testDetails && !descriptorsLoading && sourceTestId !== testId) {
      loadTest(testDetails, descriptorMap);
    } else if (!testId && sourceTestId) {
      // If no testId in URL but we have a source, reset for new test
      reset();
    }
  }, [testId, testDetails, descriptorsLoading, descriptorMap, sourceTestId, loadTest, reset]);

  const isLoading = testLoading || (testId && descriptorsLoading);

  // Handle error loading test
  if (testError) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold">Test Builder</h1>
        </div>
        <div className="card p-6 text-center">
          <p className="text-error-600 mb-4">
            Failed to load test: {testError.message}
          </p>
          <button
            onClick={() => window.location.href = '/builder'}
            className="btn btn-primary btn-sm"
          >
            Create New Test
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <h1 className="text-2xl font-bold">Test Builder</h1>
          {testId && (
            <span className="text-sm text-[var(--color-text-secondary)]">
              Editing: <span className="font-mono">{testId}</span>
            </span>
          )}
        </div>

        {!isLoggedIn && (
          <div className="px-3 py-1.5 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300 text-sm rounded">
            Log in to save tests
          </div>
        )}
      </div>

      {/* Builder layout */}
      <BuilderLayout isLoading={!!isLoading} />
    </div>
  );
}

export default TestBuilder;
