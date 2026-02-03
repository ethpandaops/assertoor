import React, { useMemo } from 'react';
import * as yaml from 'yaml';
import type { ValidationResult } from '../../../api/ai';

interface YamlPreviewProps {
  yamlContent: string;
  validation?: ValidationResult | null;
  onApply: () => void;
  onDiscard: () => void;
}

// Validate YAML syntax
function validateYamlSyntax(content: string): { valid: boolean; error?: string } {
  try {
    yaml.parse(content);
    return { valid: true };
  } catch (err) {
    const error = err instanceof Error ? err.message : 'Invalid YAML';
    return { valid: false, error };
  }
}

export const YamlPreview: React.FC<YamlPreviewProps> = ({
  yamlContent,
  validation,
  onApply,
  onDiscard,
}) => {
  const syntaxValidation = useMemo(() => validateYamlSyntax(yamlContent), [yamlContent]);

  // Check if there are any validation errors (not warnings)
  const hasErrors = validation?.issues?.some((i) => i.type === 'error') ?? false;
  const hasWarnings = validation?.issues?.some((i) => i.type === 'warning') ?? false;

  // YAML is valid if syntax is valid and there are no validation errors
  const isValid = syntaxValidation.valid && !hasErrors;

  // Determine the badge to show
  const getBadge = () => {
    if (!syntaxValidation.valid) {
      return (
        <span className="px-2 py-0.5 text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 rounded">
          Invalid Syntax
        </span>
      );
    }

    if (hasErrors) {
      return (
        <span className="px-2 py-0.5 text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 rounded">
          Has Errors
        </span>
      );
    }

    if (hasWarnings) {
      return (
        <span className="px-2 py-0.5 text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 rounded">
          Has Warnings
        </span>
      );
    }

    return (
      <span className="px-2 py-0.5 text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200 rounded">
        Valid
      </span>
    );
  };

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
      <div className="flex items-center justify-between px-3 py-2 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
            Generated YAML
          </span>
          {getBadge()}
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={onDiscard}
            className="px-3 py-1 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100
                       border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-100 dark:hover:bg-gray-700
                       transition-colors"
          >
            Discard
          </button>
          <button
            onClick={onApply}
            disabled={!isValid}
            className="px-3 py-1 text-sm text-white bg-blue-500 rounded hover:bg-blue-600
                       disabled:bg-gray-300 dark:disabled:bg-gray-600 disabled:cursor-not-allowed
                       transition-colors"
            title={!isValid ? 'Fix errors before applying' : undefined}
          >
            Apply to Builder
          </button>
        </div>
      </div>

      <div className="max-h-48 overflow-y-auto">
        <pre className="p-3 text-xs font-mono text-gray-800 dark:text-gray-200 whitespace-pre-wrap">
          {yamlContent}
        </pre>
      </div>

      {/* Syntax error */}
      {!syntaxValidation.valid && syntaxValidation.error && (
        <div className="px-3 py-2 bg-red-50 dark:bg-red-900/20 border-t border-red-200 dark:border-red-800">
          <p className="text-xs text-red-600 dark:text-red-400">
            <span className="font-medium">Syntax Error:</span> {syntaxValidation.error}
          </p>
        </div>
      )}

      {/* Validation issues */}
      {syntaxValidation.valid && validation?.issues && validation.issues.length > 0 && (
        <div className="border-t border-gray-200 dark:border-gray-700">
          <div className="px-3 py-2">
            <p className="text-xs font-medium text-gray-600 dark:text-gray-400 mb-2">
              Validation Issues ({validation.issues.length})
            </p>
            <div className="space-y-1 max-h-32 overflow-y-auto">
              {validation.issues.map((issue, idx) => (
                <div
                  key={idx}
                  className={`text-xs px-2 py-1 rounded ${
                    issue.type === 'error'
                      ? 'bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400'
                      : 'bg-yellow-50 dark:bg-yellow-900/20 text-yellow-600 dark:text-yellow-400'
                  }`}
                >
                  <span className="font-medium">
                    {issue.type === 'error' ? '✗' : '⚠'} {issue.path}:
                  </span>{' '}
                  {issue.message}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
