import { useState, useCallback } from 'react';
import yaml from 'js-yaml';

interface ObjectFieldProps {
  value: Record<string, unknown> | undefined;
  schema?: Record<string, unknown>;
  onChange: (value: unknown) => void;
}

function ObjectField({
  value,
  onChange,
}: ObjectFieldProps) {
  // For complex objects, use YAML editing
  const [yamlValue, setYamlValue] = useState(() => {
    if (!value || Object.keys(value).length === 0) return '';
    return yaml.dump(value, { indent: 2, lineWidth: -1 });
  });
  const [error, setError] = useState<string | null>(null);

  const handleChange = useCallback((newYaml: string) => {
    setYamlValue(newYaml);

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

  return (
    <div>
      <textarea
        value={yamlValue}
        onChange={(e) => handleChange(e.target.value)}
        placeholder="key: value&#10;another: value"
        rows={4}
        className={`
          w-full px-2 py-1.5 text-sm font-mono bg-[var(--color-bg-primary)] border rounded resize-y
          focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent
          ${error ? 'border-red-500' : 'border-[var(--color-border)]'}
        `}
      />
      {error && (
        <p className="text-xs text-red-500 mt-1">{error}</p>
      )}
      <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
        Enter object in YAML format
      </p>
    </div>
  );
}

export default ObjectField;
