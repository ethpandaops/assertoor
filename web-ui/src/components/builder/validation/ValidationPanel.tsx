import { useCallback } from 'react';
import { useBuilderStore } from '../../../stores/builderStore';
import type { ValidationError } from '../../../stores/builderStore';

interface ValidationPanelProps {
  errors: ValidationError[];
}

function ValidationPanel({ errors }: ValidationPanelProps) {
  const setSelection = useBuilderStore((state) => state.setSelection);
  const clearValidationErrors = useBuilderStore((state) => state.clearValidationErrors);

  // Group errors by severity
  const errorCount = errors.filter((e) => e.severity === 'error').length;
  const warningCount = errors.filter((e) => e.severity === 'warning').length;

  // Handle click on error with taskId
  const handleErrorClick = useCallback((error: ValidationError) => {
    if (error.taskId) {
      setSelection([error.taskId], error.taskId);
    }
  }, [setSelection]);

  if (errors.length === 0) {
    return null;
  }

  return (
    <div className="border-t border-[var(--color-border)] bg-[var(--color-bg-secondary)]">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-[var(--color-border)]">
        <div className="flex items-center gap-3">
          <span className="text-sm font-medium">Validation</span>
          {errorCount > 0 && (
            <span className="px-1.5 py-0.5 text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 rounded">
              {errorCount} error{errorCount !== 1 ? 's' : ''}
            </span>
          )}
          {warningCount > 0 && (
            <span className="px-1.5 py-0.5 text-xs bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300 rounded">
              {warningCount} warning{warningCount !== 1 ? 's' : ''}
            </span>
          )}
        </div>
        <button
          onClick={clearValidationErrors}
          className="p-1 hover:bg-[var(--color-bg-tertiary)] rounded"
          title="Dismiss"
        >
          <CloseIcon className="size-4 text-[var(--color-text-tertiary)]" />
        </button>
      </div>

      {/* Error list */}
      <div className="max-h-32 overflow-y-auto">
        {errors.map((error, index) => (
          <div
            key={index}
            onClick={() => handleErrorClick(error)}
            className={`
              flex items-start gap-2 px-3 py-2 border-b border-[var(--color-border)] last:border-b-0
              ${error.taskId ? 'cursor-pointer hover:bg-[var(--color-bg-tertiary)]' : ''}
            `}
          >
            {error.severity === 'error' ? (
              <ErrorIcon className="size-4 text-red-500 shrink-0 mt-0.5" />
            ) : (
              <WarningIcon className="size-4 text-yellow-500 shrink-0 mt-0.5" />
            )}
            <div className="flex-1 min-w-0">
              <p className={`text-sm ${error.severity === 'error' ? 'text-red-700 dark:text-red-300' : 'text-yellow-700 dark:text-yellow-300'}`}>
                {error.message}
              </p>
              {error.field && (
                <p className="text-xs text-[var(--color-text-tertiary)] mt-0.5">
                  Field: {error.field}
                </p>
              )}
              {error.taskId && (
                <p className="text-xs text-[var(--color-text-tertiary)] mt-0.5">
                  Click to select task
                </p>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function ErrorIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
      />
    </svg>
  );
}

function WarningIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
      />
    </svg>
  );
}

function CloseIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
    </svg>
  );
}

export default ValidationPanel;
