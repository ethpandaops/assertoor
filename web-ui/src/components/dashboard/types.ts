// Dashboard tile model — the source of truth for what a tile is.
//
// The dashboard is a flat ordered list of tiles. Each tile carries:
//   - a stable id (used for React keys + reorder targeting)
//   - a type discriminator
//   - a width (1..12 columns on the 12-col responsive grid)
//   - a free-form config bag specific to its type
//
// Adding a new tile type:
//   1. extend `TileType`
//   2. extend `TileConfig` with a new branch
//   3. register a renderer in `TileGrid.tsx`
//   4. register an editor in `TileEditor.tsx` (only the bits the user
//      should tweak; defaults belong in `defaultConfigForType`)
//   5. add a card to `AddTileModal.tsx`
//
// Config is persisted to localStorage as JSON; treat the shape as
// versioned (the storage key embeds `:v1` so we can migrate cleanly
// later).

export type TileType =
  | 'success_rate'
  | 'latest_result'
  | 'recent_runs'
  | 'text';

export type TileWidth = 'small' | 'medium' | 'large' | 'full';

// Maps tile width → column span on the 12-column grid used by TileGrid.
// Kept as a plain object so it can be imported anywhere without
// circular deps.
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

export interface TextConfig {
  markdown: string;
}

export interface DashboardConfig {
  // Schema version of the persisted config; bump when making
  // breaking changes to the structure.
  version: 1;
  tiles: DashboardTile[];
}

// Default starter dashboard shown when a user opens the page for the
// first time (or after a reset). Intentionally lightweight so it
// renders gracefully even when no tests are registered yet.
export const DEFAULT_DASHBOARD: DashboardConfig = {
  version: 1,
  tiles: [
    {
      id: 'welcome',
      type: 'text',
      width: 'full',
      config: {
        markdown:
          '# Welcome to Assertoor\n\nThis dashboard is fully configurable. ' +
          'Click **Edit dashboard** to add tiles that show success rates ' +
          'across recent runs, render the latest test result, or surface ' +
          'recent activity.',
      },
    },
    {
      id: 'recent-runs',
      type: 'recent_runs',
      width: 'full',
      config: { limit: 8 },
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
    case 'text':
      return 'Text / markdown';
  }
}
