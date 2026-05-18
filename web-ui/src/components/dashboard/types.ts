// Dashboard data model — rows of tiles.
//
// The dashboard is an ordered list of **rows**, each holding an
// ordered list of **tiles**. A row is a horizontal flow container:
// its tiles sit side-by-side and share a 12-column grid. Rows stack
// vertically with clear visual separation in edit mode, which gives
// users a way to group related tiles and rearrange them as units.
//
// Each tile carries:
//   - a stable id (React keys + drag targeting)
//   - a type discriminator
//   - a width (1..12 columns within its row)
//   - a free-form config bag specific to its type
//
// Each row carries:
//   - a stable id
//   - an optional title (shown above the row in edit mode and as a
//     subtle heading in read mode)
//   - its list of tiles
//
// Adding a new tile type:
//   1. extend `TileType`
//   2. extend `TileConfig` with a new branch
//   3. register a renderer in `TileGrid.tsx`
//   4. register an editor in `TileEditorModal.tsx`
//   5. add an entry to `TilePalette.tsx`
//
// Config is persisted to localStorage as JSON; the schema is
// versioned (`version: 2` rows-based; `version: 1` was tiles-flat).
// `useDashboardConfig` migrates v1 → v2 transparently.

export type TileType =
  | 'success_rate'
  | 'latest_result'
  | 'recent_runs'
  | 'client_status'
  | 'network_status'
  | 'text';

export type TileWidth = 'small' | 'medium' | 'large' | 'full';

// Maps tile width → column span on the row's 12-column grid. Kept as
// a plain object so it can be imported anywhere without circular
// dependencies.
export const TILE_WIDTH_COLS: Record<TileWidth, number> = {
  small: 3,
  medium: 6,
  large: 9,
  full: 12,
};

export interface DashboardTile {
  id: string;
  type: TileType;
  // Optional explicit title override. Tile renderers fall back to a
  // type-specific default (e.g. "Success rate: <test name>").
  title?: string;
  width: TileWidth;
  config: TileConfig;
}

export type TileConfig =
  | SuccessRateConfig
  | LatestResultConfig
  | RecentRunsConfig
  | ClientStatusConfig
  | NetworkStatusConfig
  | TextConfig;

export interface SuccessRateConfig {
  testId: string;
  // How many recent runs to include in the rate.
  window: number;
}

export interface LatestResultConfig {
  testId: string;
  // When true the tile shows a small header strip with run-id + start
  // time above the markdown body.
  showHeader?: boolean;
}

export interface RecentRunsConfig {
  // When unset the tile shows runs across all tests.
  testId?: string;
  limit: number;
}

export interface ClientStatusConfig {
  // When true the tile shows EL clients in addition to CL clients.
  showExecution?: boolean;
}

export interface NetworkStatusConfig {
  // Reserved for future filters (e.g. fork name). Currently a marker
  // type — the network-status endpoint serves the full payload.
  _unused?: never;
}

export interface TextConfig {
  markdown: string;
}

export interface DashboardRow {
  id: string;
  title?: string;
  tiles: DashboardTile[];
}

export interface DashboardConfig {
  // Schema version of the persisted config.
  version: 2;
  rows: DashboardRow[];
}

// ── Defaults ─────────────────────────────────────────────────────

export const DEFAULT_DASHBOARD: DashboardConfig = {
  version: 2,
  rows: [
    {
      id: 'row-intro',
      tiles: [
        {
          id: 'tile-welcome',
          type: 'text',
          width: 'full',
          config: {
            markdown:
              '# Welcome to Assertoor\n\nThis dashboard is fully configurable. ' +
              'Click **Edit dashboard** to add rows and tiles that surface ' +
              'success rates, latest test results, or recent activity.',
          },
        },
      ],
    },
    {
      id: 'row-runs',
      title: 'Recent activity',
      tiles: [
        {
          id: 'tile-recent',
          type: 'recent_runs',
          width: 'full',
          config: { limit: 8 },
        },
      ],
    },
  ],
};

export function defaultConfigForType(type: TileType): TileConfig {
  switch (type) {
    case 'success_rate':
      return { testId: '', window: 10 };
    case 'latest_result':
      return { testId: '', showHeader: true };
    case 'recent_runs':
      return { limit: 5 };
    case 'client_status':
      return { showExecution: true };
    case 'network_status':
      return {};
    case 'text':
      return { markdown: '' };
  }
}

export function defaultWidthForType(type: TileType): TileWidth {
  switch (type) {
    case 'success_rate':
      return 'small';
    case 'latest_result':
      return 'full';
    case 'recent_runs':
      return 'medium';
    case 'client_status':
      return 'large';
    case 'network_status':
      return 'medium';
    case 'text':
      return 'full';
  }
}

// Pretty-print a tile type for menus and headings.
export function tileTypeLabel(type: TileType): string {
  switch (type) {
    case 'success_rate':
      return 'Success rate';
    case 'latest_result':
      return 'Latest result';
    case 'recent_runs':
      return 'Recent runs';
    case 'client_status':
      return 'Client status';
    case 'network_status':
      return 'Network status';
    case 'text':
      return 'Text / markdown';
  }
}

// Short, single-line description used in the palette and add-tile UI.
export function tileTypeDescription(type: TileType): string {
  switch (type) {
    case 'success_rate':
      return 'Success ring + per-run swatches for one test.';
    case 'latest_result':
      return 'Markdown result of the most recent run that produced one.';
    case 'recent_runs':
      return 'Live-updating list of recent runs (all or one test).';
    case 'client_status':
      return 'EL + CL client liveness, versions, and chain heads.';
    case 'network_status':
      return 'Head, finalized, justified, queue sizes — at a glance.';
    case 'text':
      return 'Free-form markdown (headers, runbook links, notes).';
  }
}

// isValidConfig is a strict shape check; anything that doesn't match
// is replaced with `DEFAULT_DASHBOARD` on load. There is no migration
// from legacy v1 (flat tiles[]) — bumping the schema is intentionally
// a hard reset.
export function isValidConfig(parsed: unknown): parsed is DashboardConfig {
  if (!parsed || typeof parsed !== 'object') return false;
  const p = parsed as Partial<DashboardConfig>;
  return p.version === 2 && Array.isArray(p.rows);
}
