import { useState } from 'react';
import type { StringFormat } from '../TaskConfigForm';
import TextEditorModal from './TextEditorModal';
import type { EditorLanguage } from './TextEditorModal';

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

// Convert a value to display string, avoiding exponential notation for numbers.
function toDisplayString(val: unknown): string {
  if (val === undefined || val === null) return '';
  if (typeof val === 'number') {
    const s = String(val);
    // Avoid exponential notation (e.g. 1e+22 → full decimal)
    if (s.includes('e') || s.includes('E')) {
      return val.toFixed(0);
    }
    return s;
  }
  return String(val);
}

// Shared preview + modal editor for large text fields
function LargeTextPreview({
  value,
  onOpen,
  emptyLabel,
}: {
  value: string;
  onOpen: () => void;
  emptyLabel: string;
}) {
  const lines = value.split('\n');
  const previewLines = lines.slice(0, 3);
  const remainingCount = lines.length - 3;
  const hasContent = value.length > 0;

  return (
    <div
      onClick={onOpen}
      className="w-full px-2 py-1.5 text-sm font-mono bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-sm cursor-pointer hover:border-primary-500 transition-colors min-h-[3.5rem]"
    >
      {hasContent ? (
        <div className="text-xs text-[var(--color-text-primary)]">
          {previewLines.map((line, i) => (
            <div key={i} className="truncate">{line || '\u00A0'}</div>
          ))}
          {remainingCount > 0 && (
            <div className="text-[var(--color-text-tertiary)] mt-0.5">
              ... {remainingCount} more line{remainingCount !== 1 ? 's' : ''}
            </div>
          )}
        </div>
      ) : (
        <span className="text-[var(--color-text-tertiary)]">{emptyLabel}</span>
      )}
    </div>
  );
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
  const displayValue = toDisplayString(value);
  const [isEditorOpen, setIsEditorOpen] = useState(false);

  // Shell script format - preview + modal editor
  if (format === 'shell') {
    return (
      <div>
        <LargeTextPreview
          value={displayValue}
          onOpen={() => setIsEditorOpen(true)}
          emptyLabel="Click to add shell script..."
        />
        <button
          type="button"
          onClick={() => setIsEditorOpen(true)}
          className="mt-1 px-2 py-1 text-xs bg-[var(--color-bg-tertiary)] hover:bg-[var(--color-bg-secondary)] rounded-sm transition-colors"
        >
          Edit Script
        </button>
        <TextEditorModal
          isOpen={isEditorOpen}
          onClose={() => setIsEditorOpen(false)}
          value={displayValue}
          onSave={(v) => onChange(v || undefined)}
          title="Edit Shell Script"
          placeholder="#!/bin/bash&#10;# Enter your shell script here..."
        />
      </div>
    );
  }

  // Multiline/YAML format - preview + modal editor
  if (format === 'multiline' || format === 'yaml') {
    const language: EditorLanguage = format === 'yaml' ? 'yaml' : 'plain';
    const label = format === 'yaml' ? 'Edit YAML' : 'Edit Text';

    return (
      <div>
        <LargeTextPreview
          value={displayValue}
          onOpen={() => setIsEditorOpen(true)}
          emptyLabel={`Click to add ${format === 'yaml' ? 'YAML' : 'text'}...`}
        />
        <button
          type="button"
          onClick={() => setIsEditorOpen(true)}
          className="mt-1 px-2 py-1 text-xs bg-[var(--color-bg-tertiary)] hover:bg-[var(--color-bg-secondary)] rounded-sm transition-colors"
        >
          {label}
        </button>
        <TextEditorModal
          isOpen={isEditorOpen}
          onClose={() => setIsEditorOpen(false)}
          value={displayValue}
          onSave={(v) => onChange(v || undefined)}
          title={label}
          language={language}
          placeholder={placeholder || (defaultValue ? toDisplayString(defaultValue) : undefined)}
        />
        {defaultValue !== undefined && (
          <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
            Default: {toDisplayString(defaultValue)}
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
    if (defaultValue) return toDisplayString(defaultValue);

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
          Default: {toDisplayString(defaultValue)}
        </p>
      )}
    </div>
  );
}

export default StringField;
