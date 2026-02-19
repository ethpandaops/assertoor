import { useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useBuilderStore } from '../stores/builderStore';
import { useTestYaml } from '../hooks/useApi';
import { useAuthContext } from '../context/AuthContext';
import BuilderLayout from '../components/builder/BuilderLayout';

function TestBuilder() {
  const [searchParams] = useSearchParams();
  const testId = searchParams.get('testId');

  const { isLoggedIn, loading: authLoading } = useAuthContext();
  const reset = useBuilderStore((state) => state.reset);
  const loadFromYaml = useBuilderStore((state) => state.loadFromYaml);
  const setSourceTestId = useBuilderStore((state) => state.setSourceTestId);
  const setSourceInfo = useBuilderStore((state) => state.setSourceInfo);
  const sourceTestId = useBuilderStore((state) => state.sourceTestId);

  // Fetch test YAML if editing existing test (wait for auth to be ready)
  const { data: testYaml, isLoading: yamlLoading, error: testError } = useTestYaml(testId || '', {
    enabled: !!testId && !authLoading,
  });

  // Load test YAML on initial mount or when testId changes
  useEffect(() => {
    if (testId && testYaml && sourceTestId !== testId) {
      const success = loadFromYaml(testYaml.yaml);
      if (success) {
        setSourceTestId(testId);
        const isExternal = testYaml.source !== 'database' && testYaml.source !== 'api-call';
        setSourceInfo({ source: testYaml.source, isExternal });
      }
    } else if (!testId && sourceTestId) {
      // If no testId in URL but we have a source, reset for new test
      reset();
    }
  }, [testId, testYaml, sourceTestId, loadFromYaml, setSourceTestId, setSourceInfo, reset]);

  const isLoading = yamlLoading;

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
