import type { StringFormat } from '../TaskConfigForm';

interface StringFieldProps {
  value: string | undefined;
  defaultValue?: string;
  placeholder?: string;
  pattern?: string;
  minLength?: number;
  maxLength?: number;
  format?: StringFormat;
  onChange: (value: unknown) => void;
}

function StringField({
  value,
  defaultValue,
  placeholder,
  pattern,
  minLength,
  maxLength,
  format = 'text',
  onChange,
}: StringFieldProps) {
  const displayValue = value ?? '';

  // Multiline/YAML format - render textarea
  if (format === 'multiline' || format === 'yaml') {
    return (
      <div>
        <textarea
          value={displayValue}
          onChange={(e) => onChange(e.target.value || undefined)}
          placeholder={placeholder || defaultValue?.toString()}
          minLength={minLength}
          maxLength={maxLength}
          rows={4}
          className="w-full px-2 py-1.5 text-sm font-mono bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-sm focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent resize-y"
        />
        {defaultValue !== undefined && (
          <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
            Default: {String(defaultValue)}
          </p>
        )}
      </div>
    );
  }

  // Determine input styling based on format
  const getInputClassName = () => {
    const base = 'w-full px-2 py-1.5 text-sm bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-sm focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent';

    switch (format) {
      case 'address':
      case 'hash':
        return `${base} font-mono text-xs`;
      case 'bigint':
        return `${base} font-mono`;
      default:
        return base;
    }
  };

  // Get placeholder based on format
  const getPlaceholder = () => {
    if (placeholder) return placeholder;
    if (defaultValue) return defaultValue.toString();

    switch (format) {
      case 'address':
        return '0x...';
      case 'hash':
        return '0x...';
      case 'bigint':
        return '0';
      default:
        return undefined;
    }
  };

  return (
    <div>
      <input
        type="text"
        value={displayValue}
        onChange={(e) => onChange(e.target.value || undefined)}
        placeholder={getPlaceholder()}
        pattern={pattern}
        minLength={minLength}
        maxLength={maxLength}
        className={getInputClassName()}
      />
      {defaultValue !== undefined && (
        <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
          Default: {String(defaultValue)}
        </p>
      )}
    </div>
  );
}

export default StringField;
