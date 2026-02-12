import { useState, useCallback, useMemo } from 'react';
import yaml from 'js-yaml';
import TextEditorModal from './TextEditorModal';

interface ObjectFieldProps {
  value: Record<string, unknown> | undefined;
  schema?: Record<string, unknown>;
  onChange: (value: unknown) => void;
}

function ObjectField({
  value,
  onChange,
}: ObjectFieldProps) {
  const [isEditorOpen, setIsEditorOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const yamlValue = useMemo(() => {
    if (!value || Object.keys(value).length === 0) return '';
    return yaml.dump(value, { indent: 2, lineWidth: -1 });
  }, [value]);

  const [localYaml, setLocalYaml] = useState(yamlValue);

  // Keep local yaml in sync when value changes externally
  useMemo(() => {
    setLocalYaml(yamlValue);
  }, [yamlValue]);

  const handleInlineChange = useCallback((newYaml: string) => {
    setLocalYaml(newYaml);

    if (!newYaml.trim()) {
      setError(null);
      onChange(undefined);
      return;
    }

    try {
      const parsed = yaml.load(newYaml);
      if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
        setError('Value must be an object');
        return;
      }
      setError(null);
      onChange(parsed);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Invalid YAML');
    }
  }, [onChange]);

  const handleModalSave = useCallback((newYaml: string) => {
    handleInlineChange(newYaml);
  }, [handleInlineChange]);

  return (
    <div>
      <div className="flex items-start gap-1">
        <textarea
          value={localYaml}
          onChange={(e) => handleInlineChange(e.target.value)}
          placeholder="key: value"
          rows={3}
          className={`
            flex-1 px-2 py-1.5 text-sm font-mono bg-[var(--color-bg-primary)] border rounded-sm resize-y
            focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent
            ${error ? 'border-red-500' : 'border-[var(--color-border)]'}
          `}
        />
        <button
          type="button"
          onClick={() => setIsEditorOpen(true)}
          className="shrink-0 p-1.5 text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] bg-[var(--color-bg-tertiary)] hover:bg-[var(--color-bg-secondary)] rounded-sm transition-colors"
          title="Open in editor"
        >
          <ExpandIcon className="size-4" />
        </button>
      </div>
      {error && (
        <p className="text-xs text-red-500 mt-1">{error}</p>
      )}

      <TextEditorModal
        isOpen={isEditorOpen}
        onClose={() => setIsEditorOpen(false)}
        value={localYaml}
        onSave={handleModalSave}
        title="Edit Object (YAML)"
        language="yaml"
        placeholder="key: value&#10;another: value"
      />
    </div>
  );
}

function ExpandIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5v-4m0 4h-4m4 0l-5-5" />
    </svg>
  );
}

export default ObjectField;
