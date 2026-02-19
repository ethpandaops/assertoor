interface BooleanFieldProps {
  value: boolean | undefined;
  defaultValue?: boolean;
  onChange: (value: unknown) => void;
}

function BooleanField({
  value,
  defaultValue,
  onChange,
}: BooleanFieldProps) {
  const displayValue = value ?? defaultValue ?? false;

  return (
    <div className="flex items-center justify-between">
      <label className="flex items-center gap-2 cursor-pointer">
        <input
          type="checkbox"
          checked={displayValue}
          onChange={(e) => onChange(e.target.checked)}
          className="rounded border-[var(--color-border)]"
        />
        <span className="text-sm text-[var(--color-text-secondary)]">
          {displayValue ? 'Enabled' : 'Disabled'}
        </span>
      </label>
      {defaultValue !== undefined && (
        <span className="text-xs text-[var(--color-text-tertiary)]">
          Default: {defaultValue ? 'true' : 'false'}
        </span>
      )}
    </div>
  );
}

export default BooleanField;
