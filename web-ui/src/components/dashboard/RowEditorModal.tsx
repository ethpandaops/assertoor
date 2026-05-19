import { useEffect, useState } from 'react';
import Modal from '../common/Modal';
import type { DashboardRow } from './types';

interface RowEditorModalProps {
  isOpen: boolean;
  row: DashboardRow | null;
  onClose: () => void;
  onSave: (patch: Partial<DashboardRow>) => void;
}

// RowEditorModal exposes the (tiny) row-level configuration: title.
// Tile arrangement happens via drag-and-drop in the grid, not here.
export function RowEditorModal({ isOpen, row, onClose, onSave }: RowEditorModalProps) {
  const [title, setTitle] = useState(row?.title ?? '');

  useEffect(() => {
    setTitle(row?.title ?? '');
  }, [row]);

  if (!row) return null;

  const handleSave = () => {
    onSave({ title: title.trim() || undefined });
    onClose();
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Edit row" size="sm">
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium mb-1">Title (optional)</label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="e.g. Devnet health"
            className="w-full px-3 py-2 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-sm text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
          <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
            Titles appear above the row in read mode and help group related tiles.
          </p>
        </div>
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

export default RowEditorModal;
