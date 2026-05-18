import { useCallback, useMemo, useRef } from 'react';
import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query';
import {
  DEFAULT_DASHBOARD,
  defaultConfigForType,
  defaultWidthForType,
  isValidConfig,
  type DashboardConfig,
  type DashboardRow,
  type DashboardTile,
  type TileType,
  type TileWidth,
} from '../components/dashboard/types';
import * as api from '../api/client';

const QUERY_KEY = ['dashboardConfig'] as const;

function genId(prefix: string): string {
  return `${prefix}-${Math.random().toString(36).slice(2, 10)}`;
}

// Imperative API exposed by the hook. Every mutator updates the
// react-query cache immediately for snappy UI then fires an async
// PUT to persist. The PUT is auth-required server-side; the hook
// surfaces a `saveError` if persistence fails so callers can warn
// the user (typically "log in to save changes").
export interface DashboardConfigApi {
  config: DashboardConfig;
  isLoading: boolean;
  saveError: Error | null;
  isSaving: boolean;

  // Row mutators
  addRow: (afterRowId?: string) => string;
  removeRow: (rowId: string) => void;
  updateRow: (rowId: string, patch: Partial<DashboardRow>) => void;
  moveRow: (rowId: string, direction: 'up' | 'down') => void;
  reorderRows: (rowIds: string[]) => void;

  // Tile mutators
  addTile: (rowId: string, type: TileType, dstIdx?: number | null) => string;
  removeTile: (rowId: string, tileId: string) => void;
  updateTile: (rowId: string, tileId: string, patch: Partial<DashboardTile>) => void;
  moveTile: (
    srcRowId: string,
    tileId: string,
    dstRowId: string,
    dstIdx: number | null,
  ) => void;

  // Import / export
  exportJSON: () => string;
  importJSON: (json: string) => { ok: true } | { ok: false; error: string };

  reset: () => void;
}

export function useDashboardConfig(): DashboardConfigApi {
  const queryClient = useQueryClient();

  // ── Server-backed config ──────────────────────────────────────
  //
  // The cache holds whatever the server has (or `null` if no
  // config has been saved yet). We normalize that to a working
  // `config` object so renderers don't have to deal with `null`.
  const { data: serverConfig, isLoading } = useQuery<DashboardConfig | null>({
    queryKey: QUERY_KEY,
    queryFn: async () => {
      const raw = await api.getDashboardConfig();
      if (raw === null) return null;
      return isValidConfig(raw) ? raw : null;
    },
    staleTime: 10_000,
  });

  const config = serverConfig ?? DEFAULT_DASHBOARD;

  // ── Persistence ────────────────────────────────────────────────
  //
  // A single mutation is shared by every mutator; we coalesce
  // rapid edits (e.g. drag-and-drop) with a 250ms debounce.
  const persistMutation = useMutation({
    mutationFn: (cfg: DashboardConfig) => api.putDashboardConfig(cfg),
    onError: () => {
      // Roll the cache back to the server's last-known state so the
      // UI doesn't drift away from reality after a failed save.
      queryClient.invalidateQueries({ queryKey: QUERY_KEY });
    },
  });

  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const persist = useCallback(
    (cfg: DashboardConfig) => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
      debounceRef.current = setTimeout(() => {
        persistMutation.mutate(cfg);
      }, 250);
    },
    [persistMutation],
  );

  // setLocal updates the react-query cache + queues a persistence
  // round-trip. All mutators below funnel through it so the
  // optimistic-update / debounce policy is in one place.
  const setLocal = useCallback(
    (updater: (prev: DashboardConfig) => DashboardConfig) => {
      const prev = queryClient.getQueryData<DashboardConfig | null>(QUERY_KEY) ?? null;
      const base = prev ?? DEFAULT_DASHBOARD;
      const next = updater(base);
      queryClient.setQueryData<DashboardConfig | null>(QUERY_KEY, next);
      persist(next);
    },
    [queryClient, persist],
  );

  // ── Row mutators ──────────────────────────────────────────────

  const addRow = useCallback(
    (afterRowId?: string): string => {
      const newRow: DashboardRow = { id: genId('row'), tiles: [] };
      setLocal((prev) => {
        if (!afterRowId) return { ...prev, rows: [...prev.rows, newRow] };
        const idx = prev.rows.findIndex((r) => r.id === afterRowId);
        if (idx < 0) return { ...prev, rows: [...prev.rows, newRow] };
        const rows = [...prev.rows];
        rows.splice(idx + 1, 0, newRow);
        return { ...prev, rows };
      });
      return newRow.id;
    },
    [setLocal],
  );

  const removeRow = useCallback(
    (rowId: string) => {
      setLocal((prev) => ({ ...prev, rows: prev.rows.filter((r) => r.id !== rowId) }));
    },
    [setLocal],
  );

  const updateRow = useCallback(
    (rowId: string, patch: Partial<DashboardRow>) => {
      setLocal((prev) => ({
        ...prev,
        rows: prev.rows.map((r) => (r.id === rowId ? { ...r, ...patch } : r)),
      }));
    },
    [setLocal],
  );

  const moveRow = useCallback(
    (rowId: string, direction: 'up' | 'down') => {
      setLocal((prev) => {
        const idx = prev.rows.findIndex((r) => r.id === rowId);
        if (idx < 0) return prev;
        const targetIdx = direction === 'up' ? idx - 1 : idx + 1;
        if (targetIdx < 0 || targetIdx >= prev.rows.length) return prev;
        const rows = [...prev.rows];
        [rows[idx], rows[targetIdx]] = [rows[targetIdx], rows[idx]];
        return { ...prev, rows };
      });
    },
    [setLocal],
  );

  const reorderRows = useCallback(
    (rowIds: string[]) => {
      setLocal((prev) => {
        const byId = new Map(prev.rows.map((r) => [r.id, r] as const));
        const next = rowIds
          .map((id) => byId.get(id))
          .filter((r): r is DashboardRow => !!r);
        for (const r of prev.rows) if (!rowIds.includes(r.id)) next.push(r);
        return { ...prev, rows: next };
      });
    },
    [setLocal],
  );

  // ── Tile mutators ─────────────────────────────────────────────

  const addTile = useCallback(
    (rowId: string, type: TileType, dstIdx: number | null = null): string => {
      const newTile: DashboardTile = {
        id: genId('tile'),
        type,
        width: defaultWidthForType(type),
        config: defaultConfigForType(type),
      };
      setLocal((prev) => ({
        ...prev,
        rows: prev.rows.map((r) => {
          if (r.id !== rowId) return r;
          const tiles = [...r.tiles];
          const insertAt = dstIdx ?? tiles.length;
          tiles.splice(Math.max(0, Math.min(insertAt, tiles.length)), 0, newTile);
          return { ...r, tiles };
        }),
      }));
      return newTile.id;
    },
    [setLocal],
  );

  const removeTile = useCallback(
    (rowId: string, tileId: string) => {
      setLocal((prev) => ({
        ...prev,
        rows: prev.rows.map((r) =>
          r.id === rowId ? { ...r, tiles: r.tiles.filter((t) => t.id !== tileId) } : r,
        ),
      }));
    },
    [setLocal],
  );

  const updateTile = useCallback(
    (rowId: string, tileId: string, patch: Partial<DashboardTile>) => {
      setLocal((prev) => ({
        ...prev,
        rows: prev.rows.map((r) =>
          r.id !== rowId
            ? r
            : {
                ...r,
                tiles: r.tiles.map((t) => (t.id === tileId ? { ...t, ...patch } : t)),
              },
        ),
      }));
    },
    [setLocal],
  );

  const moveTile = useCallback(
    (
      srcRowId: string,
      tileId: string,
      dstRowId: string,
      dstIdx: number | null,
    ) => {
      setLocal((prev) => {
        const srcRow = prev.rows.find((r) => r.id === srcRowId);
        if (!srcRow) return prev;
        const tile = srcRow.tiles.find((t) => t.id === tileId);
        if (!tile) return prev;

        if (srcRowId === dstRowId) {
          const tiles = srcRow.tiles.filter((t) => t.id !== tileId);
          const insertAt = dstIdx ?? tiles.length;
          tiles.splice(Math.max(0, Math.min(insertAt, tiles.length)), 0, tile);
          return {
            ...prev,
            rows: prev.rows.map((r) => (r.id === srcRowId ? { ...r, tiles } : r)),
          };
        }

        return {
          ...prev,
          rows: prev.rows.map((r) => {
            if (r.id === srcRowId) {
              return { ...r, tiles: r.tiles.filter((t) => t.id !== tileId) };
            }
            if (r.id === dstRowId) {
              const tiles = [...r.tiles];
              const insertAt = dstIdx ?? tiles.length;
              tiles.splice(Math.max(0, Math.min(insertAt, tiles.length)), 0, tile);
              return { ...r, tiles };
            }
            return r;
          }),
        };
      });
    },
    [setLocal],
  );

  const reset = useCallback(() => {
    setLocal(() => DEFAULT_DASHBOARD);
  }, [setLocal]);

  const exportJSON = useCallback((): string => {
    return JSON.stringify(config, null, 2);
  }, [config]);

  const importJSON = useCallback(
    (json: string): { ok: true } | { ok: false; error: string } => {
      try {
        const parsed = JSON.parse(json);
        if (!isValidConfig(parsed)) {
          return {
            ok: false,
            error:
              'JSON does not match the dashboard schema (expected version: 2 with a rows array).',
          };
        }
        setLocal(() => parsed);
        return { ok: true };
      } catch (err) {
        return {
          ok: false,
          error: err instanceof Error ? err.message : String(err),
        };
      }
    },
    [setLocal],
  );

  return useMemo(
    () => ({
      config,
      isLoading,
      saveError: persistMutation.error,
      isSaving: persistMutation.isPending,

      addRow,
      removeRow,
      updateRow,
      moveRow,
      reorderRows,

      addTile,
      removeTile,
      updateTile,
      moveTile,

      exportJSON,
      importJSON,

      reset,
    }),
    [
      config,
      isLoading,
      persistMutation.error,
      persistMutation.isPending,
      addRow,
      removeRow,
      updateRow,
      moveRow,
      reorderRows,
      addTile,
      removeTile,
      updateTile,
      moveTile,
      exportJSON,
      importJSON,
      reset,
    ],
  );
}

// Re-exports for components that want width helpers without
// importing from `components/dashboard/types` directly.
export { TILE_WIDTH_COLS as TILE_WIDTHS } from '../components/dashboard/types';
export type { TileWidth };
