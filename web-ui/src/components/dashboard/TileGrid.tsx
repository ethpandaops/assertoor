import { type ReactNode } from 'react';
import {
  useDraggable,
  useDroppable,
} from '@dnd-kit/core';
import SuccessRateTile from './SuccessRateTile';
import LatestResultTile from './LatestResultTile';
import RecentRunsTile from './RecentRunsTile';
import ClientStatusTile from './ClientStatusTile';
import NetworkStatusTile from './NetworkStatusTile';
import TextTile from './TextTile';
import {
  TILE_WIDTH_COLS,
  tileTypeLabel,
  type ClientStatusConfig,
  type DashboardRow,
  type DashboardTile,
  type LatestResultConfig,
  type NetworkStatusConfig,
  type RecentRunsConfig,
  type SuccessRateConfig,
  type TextConfig,
} from './types';

// TileGrid renders the dashboard as a vertical stack of rows. Each
// row hosts a 12-column flex of tiles. In edit mode it gains:
//   - a visible frame around every tile (chip + bordered body)
//   - per-tile action buttons (resize, edit, remove)
//   - drop zones at the start of each row, between every pair of
//     tiles, and at the end of each row — so palette items and
//     dragged tiles always have a clear target
//   - per-row action buttons (rename, move up/down, remove)
//
// The grid does not own state — every mutation is sent up via
// callbacks. That keeps the rendering pure and lets the parent
// component juggle the dnd-kit lifecycle.

interface TileGridProps {
  rows: DashboardRow[];
  editMode: boolean;

  // Tile callbacks
  onEditTile?: (rowId: string, tileId: string) => void;
  onRemoveTile?: (rowId: string, tileId: string) => void;
  onResizeTile?: (rowId: string, tileId: string, width: DashboardTile['width']) => void;

  // Row callbacks
  onEditRow?: (rowId: string) => void;
  onRemoveRow?: (rowId: string) => void;
  onMoveRow?: (rowId: string, direction: 'up' | 'down') => void;
  onAddRow?: (afterRowId?: string) => void;
}

export function TileGrid({
  rows,
  editMode,
  onEditTile,
  onRemoveTile,
  onResizeTile,
  onEditRow,
  onRemoveRow,
  onMoveRow,
  onAddRow,
}: TileGridProps) {
  if (rows.length === 0) {
    return (
      <div className="card p-12 text-center">
        <p className="text-[var(--color-text-secondary)]">Your dashboard is empty.</p>
        <p className="text-sm mt-2 text-[var(--color-text-tertiary)]">
          {editMode
            ? 'Add a row to get started, then drag tiles from the palette.'
            : 'Click "Edit dashboard" to start adding rows and tiles.'}
        </p>
        {editMode && (
          <button
            type="button"
            onClick={() => onAddRow?.()}
            className="btn btn-primary btn-sm mt-4"
          >
            + Add row
          </button>
        )}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {rows.map((row, idx) => (
        <Row
          key={row.id}
          row={row}
          editMode={editMode}
          isFirst={idx === 0}
          isLast={idx === rows.length - 1}
          onEditTile={onEditTile}
          onRemoveTile={onRemoveTile}
          onResizeTile={onResizeTile}
          onEditRow={onEditRow}
          onRemoveRow={onRemoveRow}
          onMoveRow={onMoveRow}
        />
      ))}

      {editMode && (
        <button
          type="button"
          onClick={() => onAddRow?.()}
          className="w-full py-3 border-2 border-dashed border-[var(--color-border)] rounded-md text-sm text-[var(--color-text-tertiary)] hover:border-primary-400 hover:text-primary-600 transition-colors"
        >
          + Add row
        </button>
      )}
    </div>
  );
}

// ── Row ───────────────────────────────────────────────────────────

interface RowProps extends Pick<
  TileGridProps,
  'onEditTile' | 'onRemoveTile' | 'onResizeTile' | 'onEditRow' | 'onRemoveRow' | 'onMoveRow'
> {
  row: DashboardRow;
  editMode: boolean;
  isFirst: boolean;
  isLast: boolean;
}

function Row({
  row,
  editMode,
  isFirst,
  isLast,
  onEditTile,
  onRemoveTile,
  onResizeTile,
  onEditRow,
  onRemoveRow,
  onMoveRow,
}: RowProps) {
  return (
    <section
      className={
        editMode
          ? 'border-2 border-dashed border-[var(--color-border)] rounded-md bg-[var(--color-bg-secondary)]/30'
          : ''
      }
    >
      {/* Row header — only in edit mode (read mode shows the title
          inside the row's content area for cleanliness). */}
      {editMode && (
        <header className="flex items-center justify-between gap-2 px-3 py-1.5 border-b border-dashed border-[var(--color-border)] text-xs">
          <div className="flex items-center gap-2 min-w-0">
            <RowIcon className="size-3.5 text-[var(--color-text-tertiary)] shrink-0" />
            <span className="uppercase tracking-wider font-semibold text-[var(--color-text-tertiary)]">
              Row
            </span>
            {row.title && (
              <span className="truncate text-[var(--color-text-secondary)] font-medium">
                · {row.title}
              </span>
            )}
            <span className="text-[var(--color-text-tertiary)]">
              ({row.tiles.length} tile{row.tiles.length === 1 ? '' : 's'})
            </span>
          </div>
          <div className="flex items-center gap-0.5 shrink-0">
            <IconBtn title="Edit row" onClick={() => onEditRow?.(row.id)}>✎</IconBtn>
            <IconBtn
              title="Move row up"
              disabled={isFirst}
              onClick={() => onMoveRow?.(row.id, 'up')}
            >↑</IconBtn>
            <IconBtn
              title="Move row down"
              disabled={isLast}
              onClick={() => onMoveRow?.(row.id, 'down')}
            >↓</IconBtn>
            <IconBtn title="Remove row" danger onClick={() => onRemoveRow?.(row.id)}>✕</IconBtn>
          </div>
        </header>
      )}

      {/* Show the row's title (if any) above the tiles in read mode
          so it acts as a section heading. */}
      {!editMode && row.title && (
        <h2 className="text-sm font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] mb-2 px-1">
          {row.title}
        </h2>
      )}

      {/* The row's 12-column flow. Drop zones live between tiles in
          edit mode. */}
      <div
        className={`
          grid grid-cols-1 sm:grid-cols-6 lg:grid-cols-12 gap-3 auto-rows-min
          ${editMode ? 'p-3' : ''}
        `}
      >
        {row.tiles.length === 0 && editMode && (
          <RowDropZone rowId={row.id} index={0} empty />
        )}

        {row.tiles.map((tile, idx) => (
          <Tile
            key={tile.id}
            rowId={row.id}
            tile={tile}
            tileIndex={idx}
            editMode={editMode}
            onEdit={onEditTile}
            onRemove={onRemoveTile}
            onResize={onResizeTile}
          />
        ))}

        {row.tiles.length > 0 && editMode && (
          <RowDropZone rowId={row.id} index={row.tiles.length} />
        )}
      </div>
    </section>
  );
}

// ── Tile ──────────────────────────────────────────────────────────

interface TileProps {
  rowId: string;
  tile: DashboardTile;
  tileIndex: number;
  editMode: boolean;
  onEdit?: (rowId: string, tileId: string) => void;
  onRemove?: (rowId: string, tileId: string) => void;
  onResize?: (rowId: string, tileId: string, width: DashboardTile['width']) => void;
}

function Tile({ rowId, tile, tileIndex, editMode, onEdit, onRemove, onResize }: TileProps) {
  // Each tile is both:
  //   - a draggable source (the chip handle in edit mode)
  //   - a droppable target so palette items / other tiles can be
  //     inserted *before* this tile
  const { attributes, listeners, setNodeRef: setDragRef, isDragging } = useDraggable({
    id: `tile:${tile.id}`,
    data: { kind: 'tile', rowId, tileId: tile.id, tileIndex },
    disabled: !editMode,
  });

  const { setNodeRef: setDropRef, isOver } = useDroppable({
    id: `tile-drop:${tile.id}`,
    data: { kind: 'tile-drop', rowId, beforeTileId: tile.id, beforeIndex: tileIndex },
    disabled: !editMode,
  });

  const colSpan = colSpanClass(tile.width);

  // Combine drag + drop refs onto the same node so the tile is
  // both a handle and a target. The drop handler at the parent
  // level dispatches based on `kind`.
  const setRefs = (el: HTMLElement | null) => {
    setDragRef(el);
    setDropRef(el);
  };

  return (
    <div
      ref={setRefs}
      className={`
        ${colSpan}
        ${editMode ? 'rounded-md border border-[var(--color-border)] bg-[var(--color-bg-primary)] shadow-sm overflow-hidden' : ''}
        ${isDragging ? 'opacity-40' : ''}
        ${isOver ? 'ring-2 ring-primary-500' : ''}
        relative
      `}
    >
      {editMode && (
        <div
          className="flex items-center justify-between gap-2 px-2 py-1 bg-[var(--color-bg-secondary)] border-b border-[var(--color-border)] text-xs min-w-0 cursor-grab active:cursor-grabbing"
          {...listeners}
          {...attributes}
        >
          <div className="flex items-center gap-1.5 min-w-0">
            <DragIcon className="size-3 text-[var(--color-text-tertiary)] shrink-0" />
            <span
              className="font-semibold uppercase tracking-wider text-[var(--color-text-tertiary)] truncate"
              title={tile.title ? `${tileTypeLabel(tile.type)} · ${tile.title}` : tileTypeLabel(tile.type)}
            >
              {tileTypeLabel(tile.type)}
            </span>
          </div>
          <div className="flex items-center gap-0.5 shrink-0">
            <WidthPicker
              value={tile.width}
              onChange={(w) => onResize?.(rowId, tile.id, w)}
            />
            <IconBtn title="Edit tile" onClick={() => onEdit?.(rowId, tile.id)}>
              ✎
            </IconBtn>
            <IconBtn
              title="Remove tile"
              danger
              onClick={() => onRemove?.(rowId, tile.id)}
            >
              ✕
            </IconBtn>
          </div>
        </div>
      )}

      <div className={editMode ? '' : ''}>
        <TileBody tile={tile} />
      </div>
    </div>
  );
}

// TileBody dispatches to the renderer for the tile's type.
function TileBody({ tile }: { tile: DashboardTile }) {
  switch (tile.type) {
    case 'success_rate':
      return <SuccessRateTile tile={tile} config={tile.config as SuccessRateConfig} />;
    case 'latest_result':
      return <LatestResultTile tile={tile} config={tile.config as LatestResultConfig} />;
    case 'recent_runs':
      return <RecentRunsTile tile={tile} config={tile.config as RecentRunsConfig} />;
    case 'client_status':
      return <ClientStatusTile tile={tile} config={tile.config as ClientStatusConfig} />;
    case 'network_status':
      return <NetworkStatusTile tile={tile} config={tile.config as NetworkStatusConfig} />;
    case 'text':
      return <TextTile tile={tile} config={tile.config as TextConfig} />;
  }
}

// ── Drop zones ────────────────────────────────────────────────────

function RowDropZone({
  rowId,
  index,
  empty,
}: {
  rowId: string;
  index: number;
  empty?: boolean;
}) {
  const { setNodeRef, isOver } = useDroppable({
    id: `row-end:${rowId}:${index}`,
    data: { kind: 'tile-drop', rowId, beforeTileId: null, beforeIndex: index },
  });

  return (
    <div
      ref={setNodeRef}
      className={`
        ${empty ? 'col-span-full' : 'col-span-3'}
        flex items-center justify-center
        min-h-[3rem] rounded-md border-2 border-dashed
        ${isOver
          ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20 text-primary-600'
          : 'border-[var(--color-border)] text-[var(--color-text-tertiary)] hover:border-primary-400'}
        text-xs italic transition-colors
      `}
    >
      {empty ? 'Drop tiles here' : '+'}
    </div>
  );
}

// ── Misc helpers ──────────────────────────────────────────────────

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
      onClick={(e) => {
        e.stopPropagation();
        onClick?.();
      }}
      onPointerDown={(e) => e.stopPropagation()}
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
  const symbols: Record<DashboardTile['width'], string> = {
    small: 'S',
    medium: 'M',
    large: 'L',
    full: 'XL',
  };
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value as DashboardTile['width'])}
      onPointerDown={(e) => e.stopPropagation()}
      onClick={(e) => e.stopPropagation()}
      className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded text-xs px-1 py-0.5 mr-1"
      title={`Width: ${value} (${TILE_WIDTH_COLS[value]}/12 cols)`}
    >
      {(Object.keys(TILE_WIDTH_COLS) as DashboardTile['width'][]).map((w) => (
        <option key={w} value={w}>
          {symbols[w]}
        </option>
      ))}
    </select>
  );
}

function DragIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor">
      <circle cx="9" cy="6" r="1.5" />
      <circle cx="15" cy="6" r="1.5" />
      <circle cx="9" cy="12" r="1.5" />
      <circle cx="15" cy="12" r="1.5" />
      <circle cx="9" cy="18" r="1.5" />
      <circle cx="15" cy="18" r="1.5" />
    </svg>
  );
}

function RowIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <rect x="3" y="6" width="18" height="4" rx="1" strokeWidth="2" />
      <rect x="3" y="14" width="18" height="4" rx="1" strokeWidth="2" />
    </svg>
  );
}

export default TileGrid;
