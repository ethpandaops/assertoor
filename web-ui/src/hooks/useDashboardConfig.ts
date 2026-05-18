import { useCallback, useEffect, useState } from 'react';
import {
  DEFAULT_DASHBOARD,
  defaultConfigForType,
  defaultWidthForType,
  type DashboardConfig,
  type DashboardTile,
  type TileType,
  type TileWidth,
} from '../components/dashboard/types';

// Storage key is versioned so we can migrate later without surprising
// existing users. Bumping the suffix is equivalent to a fresh start.
const STORAGE_KEY = 'assertoor:dashboardConfig:v1';

function readFromStorage(): DashboardConfig {
  if (typeof window === 'undefined') return DEFAULT_DASHBOARD;

  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return DEFAULT_DASHBOARD;

    const parsed = JSON.parse(raw) as DashboardConfig;
    if (!parsed || parsed.version !== 1 || !Array.isArray(parsed.tiles)) {
      return DEFAULT_DASHBOARD;
    }
    return parsed;
  } catch {
    return DEFAULT_DASHBOARD;
  }
}

function writeToStorage(cfg: DashboardConfig): void {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(cfg));
  } catch {
    // Quota / serialization failures are non-fatal: the dashboard keeps
    // working in-memory for the rest of the session.
  }
}

function genId(): string {
  return `tile-${Math.random().toString(36).slice(2, 10)}`;
}

// Imperative API exposed by the hook. Each mutator persists immediately
// — there is no separate "save" step — which keeps the UX honest with
// the localStorage model.
export interface DashboardConfigApi {
  config: DashboardConfig;
  addTile: (type: TileType) => string;
  removeTile: (id: string) => void;
  updateTile: (id: string, patch: Partial<DashboardTile>) => void;
  moveTile: (id: string, direction: 'up' | 'down') => void;
  reset: () => void;
  importConfig: (cfg: DashboardConfig) => void;
}

export function useDashboardConfig(): DashboardConfigApi {
  const [config, setConfig] = useState<DashboardConfig>(() => readFromStorage());

  // Keep storage in sync after every mutation. We could write
  // synchronously inside each setter, but using effects also handles
  // setConfig calls from outside the hook (e.g. import).
  useEffect(() => {
    writeToStorage(config);
  }, [config]);

  const addTile = useCallback((type: TileType): string => {
    const newTile: DashboardTile = {
      id: genId(),
      type,
      width: defaultWidthForType(type),
      config: defaultConfigForType(type),
    };
    setConfig((prev) => ({ ...prev, tiles: [...prev.tiles, newTile] }));
    return newTile.id;
  }, []);

  const removeTile = useCallback((id: string) => {
    setConfig((prev) => ({ ...prev, tiles: prev.tiles.filter((t) => t.id !== id) }));
  }, []);

  const updateTile = useCallback((id: string, patch: Partial<DashboardTile>) => {
    setConfig((prev) => ({
      ...prev,
      tiles: prev.tiles.map((t) => (t.id === id ? { ...t, ...patch } : t)),
    }));
  }, []);

  const moveTile = useCallback((id: string, direction: 'up' | 'down') => {
    setConfig((prev) => {
      const idx = prev.tiles.findIndex((t) => t.id === id);
      if (idx < 0) return prev;
      const targetIdx = direction === 'up' ? idx - 1 : idx + 1;
      if (targetIdx < 0 || targetIdx >= prev.tiles.length) return prev;
      const next = [...prev.tiles];
      [next[idx], next[targetIdx]] = [next[targetIdx], next[idx]];
      return { ...prev, tiles: next };
    });
  }, []);

  const reset = useCallback(() => {
    setConfig(DEFAULT_DASHBOARD);
  }, []);

  const importConfig = useCallback((cfg: DashboardConfig) => {
    setConfig(cfg);
  }, []);

  return { config, addTile, removeTile, updateTile, moveTile, reset, importConfig };
}

// Helper exposed for components that want to know how a width maps to
// columns without importing the constants directly.
export { TILE_WIDTH_COLS as TILE_WIDTHS } from '../components/dashboard/types';
export type { TileWidth };
