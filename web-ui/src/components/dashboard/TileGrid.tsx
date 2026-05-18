import { type ReactNode } from 'react';
import SuccessRateTile from './SuccessRateTile';
import LatestResultTile from './LatestResultTile';
import RecentRunsTile from './RecentRunsTile';
import TextTile from './TextTile';
import {
  TILE_WIDTH_COLS,
  tileTypeLabel,
  type DashboardTile,
  type LatestResultConfig,
  type RecentRunsConfig,
  type SuccessRateConfig,
  type TextConfig,
} from './types';

interface TileGridProps {
  tiles: DashboardTile[];
  editMode: boolean;
  // Mutators (only invoked in edit mode)
  onEditTile?: (id: string) => void;
  onRemoveTile?: (id: string) => void;
  onMoveTile?: (id: string, direction: 'up' | 'down') => void;
  onResizeTile?: (id: string, width: DashboardTile['width']) => void;
}

// TileGrid lays the configured tiles out on a 12-column responsive
// grid. In edit mode each tile gains a small action strip (move
// up/down, resize, edit config, remove); the rest of the rendering
// path is identical so users see exactly what they'll see live.
export function TileGrid({
  tiles,
  editMode,
  onEditTile,
  onRemoveTile,
  onMoveTile,
  onResizeTile,
}: TileGridProps) {
  if (tiles.length === 0) {
    return (
      <div className="card p-12 text-center">
        <p className="text-[var(--color-text-secondary)]">
          Your dashboard is empty.
        </p>
        <p className="text-sm mt-2 text-[var(--color-text-tertiary)]">
          {editMode
            ? 'Click "Add tile" above to build out your dashboard.'
            : 'Click "Edit dashboard" to start adding tiles.'}
        </p>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 sm:grid-cols-6 lg:grid-cols-12 gap-4 auto-rows-min">
      {tiles.map((tile, idx) => (
        <TileSlot
          key={tile.id}
          tile={tile}
          editMode={editMode}
          isFirst={idx === 0}
          isLast={idx === tiles.length - 1}
          onEdit={onEditTile}
          onRemove={onRemoveTile}
          onMove={onMoveTile}
          onResize={onResizeTile}
        />
      ))}
    </div>
  );
}

interface TileSlotProps {
  tile: DashboardTile;
  editMode: boolean;
  isFirst: boolean;
  isLast: boolean;
  onEdit?: (id: string) => void;
  onRemove?: (id: string) => void;
  onMove?: (id: string, direction: 'up' | 'down') => void;
  onResize?: (id: string, width: DashboardTile['width']) => void;
}

function TileSlot({
  tile,
  editMode,
  isFirst,
  isLast,
  onEdit,
  onRemove,
  onMove,
  onResize,
}: TileSlotProps) {
  // Tailwind purges class names that aren't statically present in the
  // source. We map width → explicit class strings so every variant
  // survives the build.
  const colSpan = colSpanClass(tile.width);

  return (
    <div className={`${colSpan} ${editMode ? 'ring-1 ring-dashed ring-primary-400/40 rounded-md' : ''}`}>
      {editMode && (
        <div className="mb-1 flex items-center justify-between gap-2 text-xs text-[var(--color-text-tertiary)] px-1">
          <span className="font-medium uppercase tracking-wider">
            {tileTypeLabel(tile.type)}
            {tile.title ? <span className="ml-1">· {tile.title}</span> : null}
          </span>
          <div className="flex items-center gap-0.5">
            <WidthPicker
              value={tile.width}
              onChange={(w) => onResize?.(tile.id, w)}
            />
            <IconBtn
              title="Move up"
              disabled={isFirst}
              onClick={() => onMove?.(tile.id, 'up')}
            >
              ↑
            </IconBtn>
            <IconBtn
              title="Move down"
              disabled={isLast}
              onClick={() => onMove?.(tile.id, 'down')}
            >
              ↓
            </IconBtn>
            <IconBtn title="Edit" onClick={() => onEdit?.(tile.id)}>
              ✎
            </IconBtn>
            <IconBtn title="Remove" onClick={() => onRemove?.(tile.id)} danger>
              ✕
            </IconBtn>
          </div>
        </div>
      )}
      <TileBody tile={tile} />
    </div>
  );
}

// TileBody dispatches to the renderer for the tile's type. Each
// renderer is responsible for its own loading / empty / error states
// so the grid stays dumb.
function TileBody({ tile }: { tile: DashboardTile }) {
  switch (tile.type) {
    case 'success_rate':
      return <SuccessRateTile tile={tile} config={tile.config as SuccessRateConfig} />;
    case 'latest_result':
      return <LatestResultTile tile={tile} config={tile.config as LatestResultConfig} />;
    case 'recent_runs':
      return <RecentRunsTile tile={tile} config={tile.config as RecentRunsConfig} />;
    case 'text':
      return <TextTile tile={tile} config={tile.config as TextConfig} />;
  }
}

function colSpanClass(width: DashboardTile['width']): string {
  // Static strings so Tailwind's JIT can find them.
  switch (width) {
    case 'small':
      return 'sm:col-span-3 lg:col-span-3';
    case 'medium':
      return 'sm:col-span-6 lg:col-span-6';
    case 'large':
      return 'sm:col-span-6 lg:col-span-9';
    case 'full':
      return 'sm:col-span-6 lg:col-span-12';
  }
}

function IconBtn({
  children,
  title,
  disabled,
  danger,
  onClick,
}: {
  children: ReactNode;
  title: string;
  disabled?: boolean;
  danger?: boolean;
  onClick?: () => void;
}) {
  return (
    <button
      type="button"
      title={title}
      disabled={disabled}
      onClick={onClick}
      className={`px-1.5 py-0.5 rounded hover:bg-[var(--color-bg-tertiary)] disabled:opacity-40 disabled:cursor-not-allowed ${
        danger ? 'hover:text-error-600' : ''
      }`}
    >
      {children}
    </button>
  );
}

function WidthPicker({
  value,
  onChange,
}: {
  value: DashboardTile['width'];
  onChange: (w: DashboardTile['width']) => void;
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value as DashboardTile['width'])}
      className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded text-xs px-1 py-0.5 mr-1"
      title="Tile width"
    >
      {(Object.keys(TILE_WIDTH_COLS) as DashboardTile['width'][]).map((w) => (
        <option key={w} value={w}>
          {w} ({TILE_WIDTH_COLS[w]}/12)
        </option>
      ))}
    </select>
  );
}

export default TileGrid;
