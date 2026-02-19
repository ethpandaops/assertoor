import { useState, useCallback } from 'react';
import ExpressionInput from '../ExpressionInput';

interface ExpressionMapFieldProps {
  value: Record<string, string> | undefined;
  taskId: string;
  onChange: (value: unknown) => void;
}

function ExpressionMapField({ value, taskId, onChange }: ExpressionMapFieldProps) {
  const [newKey, setNewKey] = useState('');
  const entries = Object.entries(value || {});

  const handleValueChange = useCallback((key: string, newVal: string) => {
    const updated = { ...(value || {}), [key]: newVal };
    onChange(updated);
  }, [value, onChange]);

  const handleRemove = useCallback((key: string) => {
    const updated = { ...(value || {}) };
    delete updated[key];
    onChange(Object.keys(updated).length > 0 ? updated : undefined);
  }, [value, onChange]);

  const handleAdd = useCallback(() => {
    const trimmed = newKey.trim();
    if (!trimmed) return;
    if (value && trimmed in value) return;

    const updated = { ...(value || {}), [trimmed]: '' };
    onChange(updated);
    setNewKey('');
  }, [newKey, value, onChange]);

  const handleAddKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleAdd();
    }
  }, [handleAdd]);

  return (
    <div className="space-y-2">
      {entries.map(([key, val]) => (
        <div key={key} className="flex items-start gap-2">
          <div className="shrink-0 w-28">
            <span className="inline-block px-2 py-1.5 text-xs font-mono bg-[var(--color-bg-tertiary)] border border-[var(--color-border)] rounded-sm truncate w-full" title={key}>
              {key}
            </span>
          </div>
          <div className="flex-1 min-w-0">
            <ExpressionInput
              taskId={taskId}
              value={val}
              onChange={(v) => handleValueChange(key, v)}
              placeholder="Expression for value"
            />
          </div>
          <button
            type="button"
            onClick={() => handleRemove(key)}
            className="shrink-0 p-1.5 text-[var(--color-text-tertiary)] hover:text-red-500 transition-colors"
            title="Remove entry"
          >
            <RemoveIcon className="size-4" />
          </button>
        </div>
      ))}

      {/* Add new entry */}
      <div className="flex items-center gap-2">
        <input
          type="text"
          value={newKey}
          onChange={(e) => setNewKey(e.target.value)}
          onKeyDown={handleAddKeyDown}
          placeholder="Variable name"
          className="w-28 px-2 py-1.5 text-xs font-mono bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-sm focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent"
        />
        <button
          type="button"
          onClick={handleAdd}
          disabled={!newKey.trim()}
          className="px-2 py-1.5 text-xs bg-[var(--color-bg-tertiary)] hover:bg-[var(--color-bg-secondary)] rounded-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          + Add
        </button>
      </div>
    </div>
  );
}

function RemoveIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
    </svg>
  );
}

export default ExpressionMapField;
