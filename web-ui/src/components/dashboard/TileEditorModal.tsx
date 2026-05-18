import { useEffect, useState } from 'react';
import Modal from '../common/Modal';
import { useTests } from '../../hooks/useApi';
import {
  tileTypeLabel,
  type DashboardTile,
  type LatestResultConfig,
  type RecentRunsConfig,
  type SuccessRateConfig,
  type TextConfig,
} from './types';

interface TileEditorModalProps {
  isOpen: boolean;
  tile: DashboardTile | null;
  onClose: () => void;
  onSave: (patch: Partial<DashboardTile>) => void;
}

// TileEditorModal exposes only the configuration fields each tile
// actually needs. The form is intentionally minimal — width and order
// are tweaked via inline controls in the grid, not here.
export function TileEditorModal({
  isOpen,
  tile,
  onClose,
  onSave,
}: TileEditorModalProps) {
  // Stage edits in local state so cancelling discards changes.
  const [draft, setDraft] = useState<DashboardTile | null>(tile);

  useEffect(() => {
    setDraft(tile);
  }, [tile]);

  if (!draft) return null;

  const handleSave = () => {
    onSave({ title: draft.title, config: draft.config });
    onClose();
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`Edit ${tileTypeLabel(draft.type)} tile`}
      size="md"
    >
      <div className="space-y-4">
        {/* Title (optional, common to all tile types) */}
        <Field label="Title (optional)">
          <input
            type="text"
            value={draft.title ?? ''}
            onChange={(e) => setDraft({ ...draft, title: e.target.value || undefined })}
            placeholder="leave blank to use the default"
            className="w-full px-3 py-2 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-sm text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
        </Field>

        {/* Type-specific config */}
        <TypeSpecificEditor
          tile={draft}
          onChange={(config) => setDraft({ ...draft, config })}
        />

        <div className="flex justify-end gap-2 pt-2 border-t border-[var(--color-border)]">
          <button type="button" onClick={onClose} className="btn btn-secondary btn-sm">
            Cancel
          </button>
          <button type="button" onClick={handleSave} className="btn btn-primary btn-sm">
            Save
          </button>
        </div>
      </div>
    </Modal>
  );
}

function TypeSpecificEditor({
  tile,
  onChange,
}: {
  tile: DashboardTile;
  onChange: (config: DashboardTile['config']) => void;
}) {
  switch (tile.type) {
    case 'success_rate':
      return (
        <SuccessRateEditor
          config={tile.config as SuccessRateConfig}
          onChange={onChange}
        />
      );
    case 'latest_result':
      return (
        <LatestResultEditor
          config={tile.config as LatestResultConfig}
          onChange={onChange}
        />
      );
    case 'recent_runs':
      return (
        <RecentRunsEditor
          config={tile.config as RecentRunsConfig}
          onChange={onChange}
        />
      );
    case 'text':
      return <TextEditor config={tile.config as TextConfig} onChange={onChange} />;
  }
}

function SuccessRateEditor({
  config,
  onChange,
}: {
  config: SuccessRateConfig;
  onChange: (c: SuccessRateConfig) => void;
}) {
  return (
    <>
      <TestPicker
        required
        value={config.testId}
        onChange={(testId) => onChange({ ...config, testId })}
      />
      <Field label="Window (number of recent runs to consider)">
        <input
          type="number"
          min={1}
          max={50}
          value={config.window}
          onChange={(e) =>
            onChange({ ...config, window: Math.max(1, parseInt(e.target.value || '10', 10)) })
          }
          className="w-32 px-3 py-2 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-sm text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
        />
      </Field>
    </>
  );
}

function LatestResultEditor({
  config,
  onChange,
}: {
  config: LatestResultConfig;
  onChange: (c: LatestResultConfig) => void;
}) {
  return (
    <>
      <TestPicker
        required
        value={config.testId}
        onChange={(testId) => onChange({ ...config, testId })}
      />
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={config.showHeader !== false}
          onChange={(e) => onChange({ ...config, showHeader: e.target.checked })}
        />
        Show header strip (test name, run id, status)
      </label>
    </>
  );
}

function RecentRunsEditor({
  config,
  onChange,
}: {
  config: RecentRunsConfig;
  onChange: (c: RecentRunsConfig) => void;
}) {
  return (
    <>
      <TestPicker
        allowAll
        value={config.testId ?? ''}
        onChange={(testId) =>
          onChange({ ...config, testId: testId === '' ? undefined : testId })
        }
      />
      <Field label="Limit (rows shown)">
        <input
          type="number"
          min={1}
          max={50}
          value={config.limit}
          onChange={(e) =>
            onChange({ ...config, limit: Math.max(1, parseInt(e.target.value || '5', 10)) })
          }
          className="w-32 px-3 py-2 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-sm text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
        />
      </Field>
    </>
  );
}

function TextEditor({
  config,
  onChange,
}: {
  config: TextConfig;
  onChange: (c: TextConfig) => void;
}) {
  return (
    <Field label="Markdown">
      <textarea
        value={config.markdown}
        onChange={(e) => onChange({ ...config, markdown: e.target.value })}
        className="w-full h-48 px-3 py-2 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-sm font-mono text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 resize-y"
        placeholder="# Heading&#10;&#10;Free-form markdown supports GFM (tables, task lists, etc.)."
      />
    </Field>
  );
}

// TestPicker is a select bound to the registered tests list. When
// `allowAll` is true an empty value represents "all tests"; otherwise
// it represents an unconfigured tile.
function TestPicker({
  value,
  onChange,
  required,
  allowAll,
}: {
  value: string;
  onChange: (testId: string) => void;
  required?: boolean;
  allowAll?: boolean;
}) {
  const { data: tests, isLoading } = useTests();

  return (
    <Field label="Test">
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-sm text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
      >
        {!required && allowAll && <option value="">All tests</option>}
        {!required && !allowAll && <option value="">— select a test —</option>}
        {required && <option value="">— select a test —</option>}
        {isLoading && <option disabled>Loading…</option>}
        {tests?.map((t) => (
          <option key={t.id} value={t.id}>
            {t.name || t.id} ({t.id})
          </option>
        ))}
      </select>
      {required && !value && (
        <p className="text-xs text-amber-600 mt-1">A test is required for this tile.</p>
      )}
    </Field>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="block text-sm font-medium mb-1">{label}</label>
      {children}
    </div>
  );
}

export default TileEditorModal;
