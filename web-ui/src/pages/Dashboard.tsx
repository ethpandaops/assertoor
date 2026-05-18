import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
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

// Dashboard owns the edit-mode toggle plus the drag-and-drop context.
// Mutations are local-only — the hook tracks a draft separate from
// the server-loaded config. The user explicitly commits via "Save
// changes"; "Discard" throws the draft away. Importing a JSON file
// behaves the same way: it loads into the draft and is only
// persisted when the user saves.
function Dashboard() {
  const { isLoggedIn } = useAuthContext();
  const {
    config,
    isLoading,
    isDirty,
    save,
    discard,
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

  // Anyone landing on the dashboard while a draft is in flight gets
  // the unsaved warning before navigating away — protects users from
  // losing edits if they accidentally close the tab.
  useEffect(() => {
    if (!isDirty) return;
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      e.returnValue = '';
    };
    window.addEventListener('beforeunload', handler);
    return () => window.removeEventListener('beforeunload', handler);
  }, [isDirty]);

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
        let dstIdx: number | null = toData.beforeIndex;
        if (fromData.rowId === toData.rowId && typeof dstIdx === 'number') {
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
      const lastRow = config.rows[config.rows.length - 1];
      const rowId = lastRow ? lastRow.id : addRow();
      addTile(rowId, type);
    },
    [config.rows, addRow, addTile],
  );

  const handleReset = useCallback(() => {
    if (
      confirm(
        'Reset the dashboard to its defaults? This stages a reset locally — click "Save changes" to make it permanent.',
      )
    ) {
      reset();
    }
  }, [reset]);

  const handleExitEdit = useCallback(() => {
    if (isDirty) {
      if (!confirm('You have unsaved changes. Discard them?')) return;
      discard();
    }
    setEditMode(false);
  }, [isDirty, discard]);

  const handleSave = useCallback(() => {
    save();
    // The hook clears the draft on success; we stay in edit mode so
    // the user can keep working without losing context.
  }, [save]);

  const handleDiscard = useCallback(() => {
    if (!confirm('Discard your unsaved changes?')) return;
    discard();
  }, [discard]);

  // ── Export / Import ────────────────────────────────────────────

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

  const importInputRef = useRef<HTMLInputElement>(null);
  const handleImportClick = useCallback(() => importInputRef.current?.click(), []);
  const handleImportFile = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (!file) return;
      e.target.value = '';
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

  // Editing requires authentication: the PUT endpoint is auth-gated
  // and we don't want to let anonymous users build edits they
  // can't actually save.
  const canEdit = isLoggedIn;

  return (
    <div className="space-y-4">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            Dashboard
            {isDirty && (
              <span className="text-xs px-2 py-0.5 rounded-sm bg-amber-100 dark:bg-amber-900/40 text-amber-800 dark:text-amber-200 font-medium uppercase tracking-wider">
                Unsaved
              </span>
            )}
          </h1>
          <p className="text-sm text-[var(--color-text-secondary)]">
            {editMode
              ? isDirty
                ? 'Changes are local — click "Save changes" to publish them, or "Discard" to throw them away.'
                : 'Edit mode — drag tiles from the palette, rearrange existing ones, and tweak settings.'
              : canEdit
                ? 'Live overview of recent test activity. Customise by clicking "Edit dashboard".'
                : 'Live overview of recent test activity. Log in to customise.'}
          </p>
        </div>

        <div className="flex items-center gap-2">
          {saveError && (
            <span className="text-xs text-error-600" title={saveError.message}>
              Save failed — {saveError.message}
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
                title="Replace the dashboard (locally) with an uploaded JSON file"
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
                title="Stage a reset to the default dashboard (still needs to be saved)"
              >
                Reset
              </button>
              {isDirty && (
                <button
                  type="button"
                  onClick={handleDiscard}
                  className="btn btn-secondary btn-sm"
                  title="Throw away your local edits and revert to the saved dashboard"
                >
                  Discard
                </button>
              )}
              <button
                type="button"
                onClick={handleSave}
                disabled={!isDirty || isSaving}
                className="btn btn-primary btn-sm disabled:opacity-50"
              >
                {isSaving ? 'Saving…' : 'Save changes'}
              </button>
              <button
                type="button"
                onClick={handleExitEdit}
                className="btn btn-secondary btn-sm"
                title="Leave edit mode (you'll be warned if there are unsaved changes)"
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

export type { DashboardTile };
export default Dashboard;
