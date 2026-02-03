interface DurationFieldProps {
  value: string | undefined;
  defaultValue?: string;
  onChange: (value: unknown) => void;
}

function DurationField({
  value,
  defaultValue,
  onChange,
}: DurationFieldProps) {
  const displayValue = value ?? '';

  return (
    <div>
      <div className="flex items-center gap-2">
        <input
          type="text"
          value={displayValue}
          onChange={(e) => onChange(e.target.value || undefined)}
          placeholder={defaultValue || '10s'}
          className="flex-1 px-2 py-1.5 text-sm font-mono bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
        />
        <span className="text-xs text-[var(--color-text-tertiary)] shrink-0">
          e.g., 10s, 5m, 1h
        </span>
      </div>
      {defaultValue && (
        <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
          Default: {defaultValue}
        </p>
      )}
    </div>
  );
}

export default DurationField;
