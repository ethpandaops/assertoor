import { useCallback, useMemo, useState } from 'react';
import { useDashboardConfig } from '../hooks/useDashboardConfig';
import TileGrid from '../components/dashboard/TileGrid';
import AddTileModal from '../components/dashboard/AddTileModal';
import TileEditorModal from '../components/dashboard/TileEditorModal';
import type { DashboardTile, TileType } from '../components/dashboard/types';

// Dashboard is a fully configurable tile grid persisted to
// localStorage. Edit mode flips the page into a hands-on layout
// editor: per-tile resize / move / edit / remove + an "Add tile"
// button. Outside of edit mode the page is pure read-only output.
function Dashboard() {
  const { config, addTile, removeTile, updateTile, moveTile, reset } =
    useDashboardConfig();

  const [editMode, setEditMode] = useState(false);
  const [isAddOpen, setAddOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);

  const editingTile = useMemo(
    () => config.tiles.find((t) => t.id === editingId) ?? null,
    [config.tiles, editingId],
  );

  const handleAddTile = useCallback(
    (type: TileType) => {
      const id = addTile(type);
      // Immediately open the editor so users land in the right place
      // — no "where did it go" moment after picking a tile type.
      setEditingId(id);
    },
    [addTile],
  );

  const handleResize = useCallback(
    (id: string, width: DashboardTile['width']) => {
      updateTile(id, { width });
    },
    [updateTile],
  );

  const handleSaveEdit = useCallback(
    (patch: Partial<DashboardTile>) => {
      if (editingId) updateTile(editingId, patch);
    },
    [editingId, updateTile],
  );

  const handleReset = useCallback(() => {
    if (confirm('Reset the dashboard to its defaults? This clears all your tiles.')) {
      reset();
    }
  }, [reset]);

  return (
    <div className="space-y-4">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">Dashboard</h1>
          <p className="text-sm text-[var(--color-text-secondary)]">
            {editMode
              ? 'Edit mode — add, resize, reorder, or configure tiles. Changes save automatically.'
              : 'Live overview of recent test activity. Customise by clicking "Edit dashboard".'}
          </p>
        </div>

        <div className="flex items-center gap-2">
          {editMode ? (
            <>
              <button
                type="button"
                onClick={() => setAddOpen(true)}
                className="btn btn-primary btn-sm flex items-center gap-1.5"
              >
                <PlusIcon className="size-4" /> Add tile
              </button>
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
                className="btn btn-secondary btn-sm"
              >
                Done
              </button>
            </>
          ) : (
            <button
              type="button"
              onClick={() => setEditMode(true)}
              className="btn btn-secondary btn-sm flex items-center gap-1.5"
            >
              <EditIcon className="size-4" /> Edit dashboard
            </button>
          )}
        </div>
      </header>

      <TileGrid
        tiles={config.tiles}
        editMode={editMode}
        onEditTile={setEditingId}
        onRemoveTile={removeTile}
        onMoveTile={moveTile}
        onResizeTile={handleResize}
      />

      <AddTileModal
        isOpen={isAddOpen}
        onClose={() => setAddOpen(false)}
        onSelect={handleAddTile}
      />

      <TileEditorModal
        isOpen={editingTile !== null}
        tile={editingTile}
        onClose={() => setEditingId(null)}
        onSave={handleSaveEdit}
      />
    </div>
  );
}

function PlusIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
    </svg>
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

export default Dashboard;
