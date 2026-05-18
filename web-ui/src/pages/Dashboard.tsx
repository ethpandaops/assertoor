import { useCallback, useMemo, useRef, useState } from 'react';
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from '@dnd-kit/core';
import { useDashboardConfig } from '../hooks/useDashboardConfig';
import { useAuthContext } from '../context/AuthContext';
import TileGrid from '../components/dashboard/TileGrid';
import TilePalette from '../components/dashboard/TilePalette';
import TileEditorModal from '../components/dashboard/TileEditorModal';
import RowEditorModal from '../components/dashboard/RowEditorModal';
import {
  tileTypeLabel,
  type DashboardTile,
  type TileType,
} from '../components/dashboard/types';

// Dashboard is the dashboard's only stateful component. It owns:
//   - the edit-mode toggle (only accessible when logged in)
//   - the drag-and-drop context (dispatches palette drops to addTile
//     and tile drops to moveTile)
//   - the editor modals for tiles and rows
//   - the import/export controls
//
// Everything else is dumb: TileGrid renders, useDashboardConfig
// persists, and the palette is drag-source-only.

function Dashboard() {
  const { isLoggedIn } = useAuthContext();
  const {
    config,
    isLoading,
    isSaving,
    saveError,
    addRow,
    removeRow,
    updateRow,
    moveRow,
    addTile,
    removeTile,
    updateTile,
    moveTile,
    importJSON,
    exportJSON,
    reset,
  } = useDashboardConfig();

  const [editMode, setEditMode] = useState(false);

  // ── Editor modals ─────────────────────────────────────────────

  const [editingTile, setEditingTile] = useState<{ rowId: string; tileId: string } | null>(null);
  const [editingRow, setEditingRow] = useState<string | null>(null);

  const editingTileObj = useMemo(() => {
    if (!editingTile) return null;
    const row = config.rows.find((r) => r.id === editingTile.rowId);
    return row?.tiles.find((t) => t.id === editingTile.tileId) ?? null;
  }, [config.rows, editingTile]);

  const editingRowObj = useMemo(() => {
    if (!editingRow) return null;
    return config.rows.find((r) => r.id === editingRow) ?? null;
  }, [config.rows, editingRow]);

  // ── Drag-and-drop wiring ──────────────────────────────────────

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }));
  const [activeDrag, setActiveDrag] = useState<{
    kind: 'palette' | 'tile';
    label: string;
  } | null>(null);

  const handleDragStart = useCallback(
    (event: DragStartEvent) => {
      const data = event.active.data.current as
        | { kind: 'palette'; tileType: TileType }
        | { kind: 'tile'; rowId: string; tileId: string }
        | undefined;
      if (!data) return;

      if (data.kind === 'palette') {
        setActiveDrag({ kind: 'palette', label: tileTypeLabel(data.tileType) });
      } else if (data.kind === 'tile') {
        const row = config.rows.find((r) => r.id === data.rowId);
        const tile = row?.tiles.find((t) => t.id === data.tileId);
        setActiveDrag({
          kind: 'tile',
          label: tile ? (tile.title || tileTypeLabel(tile.type)) : 'Tile',
        });
      }
    },
    [config.rows],
  );

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      setActiveDrag(null);
      const { active, over } = event;
      if (!over) return;

      const fromData = active.data.current as
        | { kind: 'palette'; tileType: TileType }
        | { kind: 'tile'; rowId: string; tileId: string }
        | undefined;
      const toData = over.data.current as
        | { kind: 'tile-drop'; rowId: string; beforeTileId: string | null; beforeIndex: number }
        | undefined;

      if (!fromData || !toData || toData.kind !== 'tile-drop') return;

      if (fromData.kind === 'palette') {
        addTile(toData.rowId, fromData.tileType, toData.beforeIndex);
      } else if (fromData.kind === 'tile') {
        // Reordering: when moving within the same row, dropping
        // *after* the original position needs an index correction
        // because the source slot is going to disappear.
        let dstIdx: number | null = toData.beforeIndex;
        if (
          fromData.rowId === toData.rowId &&
          typeof dstIdx === 'number'
        ) {
          const srcRow = config.rows.find((r) => r.id === fromData.rowId);
          const srcIdx = srcRow?.tiles.findIndex((t) => t.id === fromData.tileId) ?? -1;
          if (srcIdx >= 0 && dstIdx > srcIdx) dstIdx -= 1;
        }
        moveTile(fromData.rowId, fromData.tileId, toData.rowId, dstIdx);
      }
    },
    [config.rows, addTile, moveTile],
  );

  // ── Toolbar handlers ──────────────────────────────────────────

  const handlePalettePick = useCallback(
    (type: TileType) => {
      // Click-to-add: append to the last row, creating one if needed.
      const lastRow = config.rows[config.rows.length - 1];
      const rowId = lastRow ? lastRow.id : addRow();
      addTile(rowId, type);
    },
    [config.rows, addRow, addTile],
  );

  const handleReset = useCallback(() => {
    if (confirm('Reset the dashboard to its defaults? This clears every row and tile.')) {
      reset();
    }
  }, [reset]);

  // ── Export ─────────────────────────────────────────────────────

  const handleExport = useCallback(() => {
    const json = exportJSON();
    const blob = new Blob([json], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `assertoor-dashboard-${new Date().toISOString().slice(0, 10)}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, [exportJSON]);

  // ── Import ─────────────────────────────────────────────────────

  const importInputRef = useRef<HTMLInputElement>(null);
  const handleImportClick = useCallback(() => importInputRef.current?.click(), []);
  const handleImportFile = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (!file) return;
      e.target.value = ''; // allow re-importing the same file
      try {
        const text = await file.text();
        const result = importJSON(text);
        if (!result.ok) alert(`Import failed: ${result.error}`);
      } catch (err) {
        alert(`Import failed: ${err}`);
      }
    },
    [importJSON],
  );

  // ── Render ─────────────────────────────────────────────────────

  // Edit mode is only meaningful when the user is authenticated —
  // the PUT endpoint requires auth, so an anonymous user couldn't
  // save anything anyway. We hide the toggle entirely to keep the
  // UX honest.
  const canEdit = isLoggedIn;

  return (
    <div className="space-y-4">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">Dashboard</h1>
          <p className="text-sm text-[var(--color-text-secondary)]">
            {editMode
              ? 'Edit mode — drag tiles from the palette, rearrange existing ones, and tweak settings. Changes save automatically.'
              : canEdit
                ? 'Live overview of recent test activity. Customise by clicking "Edit dashboard".'
                : 'Live overview of recent test activity. Log in to customise.'}
          </p>
        </div>

        <div className="flex items-center gap-2">
          {isSaving && (
            <span className="text-xs text-[var(--color-text-tertiary)]">Saving…</span>
          )}
          {saveError && (
            <span
              className="text-xs text-error-600"
              title={saveError.message}
            >
              Save failed
            </span>
          )}
          {editMode ? (
            <>
              <button
                type="button"
                onClick={handleExport}
                className="btn btn-secondary btn-sm"
                title="Download the current dashboard as JSON"
              >
                Export
              </button>
              <button
                type="button"
                onClick={handleImportClick}
                className="btn btn-secondary btn-sm"
                title="Replace the dashboard with an uploaded JSON file"
              >
                Import
              </button>
              <input
                ref={importInputRef}
                type="file"
                accept="application/json,.json"
                hidden
                onChange={handleImportFile}
              />
              <button
                type="button"
                onClick={handleReset}
                className="btn btn-secondary btn-sm"
                title="Restore the default dashboard"
              >
                Reset
              </button>
              <button
                type="button"
                onClick={() => setEditMode(false)}
                className="btn btn-primary btn-sm"
              >
                Done
              </button>
            </>
          ) : canEdit ? (
            <button
              type="button"
              onClick={() => setEditMode(true)}
              className="btn btn-secondary btn-sm flex items-center gap-1.5"
            >
              <EditIcon className="size-4" /> Edit dashboard
            </button>
          ) : null}
        </div>
      </header>

      {isLoading ? (
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full size-6 border-b-2 border-primary-600" />
        </div>
      ) : (
        <DndContext sensors={sensors} onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
          <div className={editMode ? 'grid grid-cols-1 lg:grid-cols-[1fr_18rem] gap-4' : ''}>
            <div className="min-w-0">
              <TileGrid
                rows={config.rows}
                editMode={editMode}
                onEditTile={(rowId, tileId) => setEditingTile({ rowId, tileId })}
                onRemoveTile={removeTile}
                onResizeTile={(rowId, tileId, width) =>
                  updateTile(rowId, tileId, { width })
                }
                onEditRow={(rowId) => setEditingRow(rowId)}
                onRemoveRow={removeRow}
                onMoveRow={moveRow}
                onAddRow={addRow}
              />
            </div>
            {editMode && <TilePalette onPick={handlePalettePick} />}
          </div>

          <DragOverlay>
            {activeDrag && (
              <div className="px-3 py-2 rounded border bg-[var(--color-bg-primary)] border-primary-500 shadow-lg text-sm font-medium">
                {activeDrag.kind === 'palette' ? `+ ${activeDrag.label}` : activeDrag.label}
              </div>
            )}
          </DragOverlay>
        </DndContext>
      )}

      <TileEditorModal
        isOpen={editingTileObj !== null}
        tile={editingTileObj}
        onClose={() => setEditingTile(null)}
        onSave={(patch) => {
          if (editingTile) updateTile(editingTile.rowId, editingTile.tileId, patch);
        }}
      />

      <RowEditorModal
        isOpen={editingRowObj !== null}
        row={editingRowObj}
        onClose={() => setEditingRow(null)}
        onSave={(patch) => {
          if (editingRow) updateRow(editingRow, patch);
        }}
      />
    </div>
  );
}

function EditIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"
      />
    </svg>
  );
}

// Re-export so the file still resolves when only the type is imported
// elsewhere (e.g. lazy imports). Not strictly needed but harmless.
export type { DashboardTile };

export default Dashboard;
