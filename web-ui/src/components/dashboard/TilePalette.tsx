import { useDraggable } from '@dnd-kit/core';
import {
  tileTypeLabel,
  tileTypeDescription,
  type TileType,
} from './types';

// TilePalette is the edit-mode sidebar that lists every tile type
// available to add. Each entry is draggable onto a row's drop zone;
// `id` carries a `palette:<type>` discriminator so the grid's drop
// handler can distinguish "create-tile" drops from "move-tile" drops.

const TILE_TYPES: TileType[] = [
  'success_rate',
  'latest_result',
  'recent_runs',
  'client_status',
  'network_status',
  'text',
];

// Glyph per tile type. Plain characters so we don't add an icon
// library just for this palette.
const TILE_GLYPH: Record<TileType, string> = {
  success_rate: '◐',
  latest_result: '📄',
  recent_runs: '⟳',
  client_status: '⚙',
  network_status: '🛰',
  text: '✎',
};

interface TilePaletteProps {
  // When dropped on a target the grid emits an `addTile(type)` —
  // this callback exists so palette items can be added by click
  // too (a row's "+" affordance opens the palette and clicks pick).
  onPick?: (type: TileType) => void;
}

export function TilePalette({ onPick }: TilePaletteProps) {
  return (
    <aside className="card sticky top-4 flex flex-col h-fit max-h-[calc(100vh-8rem)]">
      <header className="card-header flex-shrink-0">
        <div className="font-medium">Tile palette</div>
        <div className="text-[11px] text-[var(--color-text-tertiary)] mt-0.5">
          Drag a tile into a row, or click to add to the last row.
        </div>
      </header>

      <div className="p-2 overflow-y-auto flex flex-col gap-1.5">
        {TILE_TYPES.map((type) => (
          <PaletteItem key={type} type={type} onPick={onPick} />
        ))}
      </div>
    </aside>
  );
}

function PaletteItem({
  type,
  onPick,
}: {
  type: TileType;
  onPick?: (type: TileType) => void;
}) {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `palette:${type}`,
    data: { kind: 'palette', tileType: type },
  });

  return (
    <button
      ref={setNodeRef}
      {...listeners}
      {...attributes}
      onClick={() => onPick?.(type)}
      type="button"
      className={`
        text-left px-3 py-2 rounded border border-[var(--color-border)]
        bg-[var(--color-bg-primary)] hover:border-primary-400 hover:shadow-sm
        cursor-grab active:cursor-grabbing transition-all
        ${isDragging ? 'opacity-50' : ''}
      `}
    >
      <div className="flex items-center gap-2">
        <span className="text-base leading-none">{TILE_GLYPH[type]}</span>
        <span className="font-medium text-sm">{tileTypeLabel(type)}</span>
      </div>
      <p className="text-[11px] text-[var(--color-text-tertiary)] mt-0.5 leading-snug">
        {tileTypeDescription(type)}
      </p>
    </button>
  );
}

export default TilePalette;
