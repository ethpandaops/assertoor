interface NumberFieldProps {
  value: number | undefined;
  defaultValue?: number;
  min?: number;
  max?: number;
  onChange: (value: unknown) => void;
}

function NumberField({
  value,
  defaultValue,
  min,
  max,
  onChange,
}: NumberFieldProps) {
  const displayValue = value ?? '';

  return (
    <div>
      <input
        type="number"
        value={displayValue}
        onChange={(e) => {
          const val = e.target.value;
          if (val === '') {
            onChange(undefined);
          } else {
            const num = parseFloat(val);
            onChange(isNaN(num) ? undefined : num);
          }
        }}
        placeholder={defaultValue?.toString()}
        min={min}
        max={max}
        className="w-full px-2 py-1.5 text-sm bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
      />
      {defaultValue !== undefined && (
        <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
          Default: {defaultValue}
        </p>
      )}
    </div>
  );
}

export default NumberField;
