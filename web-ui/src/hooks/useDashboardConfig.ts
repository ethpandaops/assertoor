import { useCallback, useMemo, useState } from 'react';
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

// Imperative API exposed by the hook.
//
// The hook keeps two parallel pieces of state:
//
// - `serverConfig` (react-query cache): the last known config from
//   the server. This is what anonymous viewers see.
// - `draft` (local React state): an optional in-progress edit. As
//   soon as any mutator runs we seed `draft` from `serverConfig`
//   and apply edits there. The UI renders `draft ?? serverConfig`,
//   so changes appear instantly without touching the network.
//
// Saving is **explicit**: the user must hit "Save changes" to PUT
// the draft to the server. Until then, navigating away or hitting
// "Discard" simply throws the draft away — there is no implicit
// live-save. This makes the edit flow feel like a small import:
// you build the dashboard, then commit it as one atomic write.
export interface DashboardConfigApi {
  config: DashboardConfig;
  isLoading: boolean;

  // True iff there are unsaved local edits.
  isDirty: boolean;

  // Save / discard / progress reporting for the explicit-save flow.
  save: () => void;
  discard: () => void;
  isSaving: boolean;
  saveError: Error | null;

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

  // Import / export — both are local-only; they mutate the draft.
  exportJSON: () => string;
  importJSON: (json: string) => { ok: true } | { ok: false; error: string };

  reset: () => void;
}

export function useDashboardConfig(): DashboardConfigApi {
  const queryClient = useQueryClient();

  // ── Server-fetched config (read-only baseline) ─────────────────
  const { data: serverConfig, isLoading } = useQuery<DashboardConfig | null>({
    queryKey: QUERY_KEY,
    queryFn: async () => {
      const raw = await api.getDashboardConfig();
      if (raw === null) return null;
      return isValidConfig(raw) ? raw : null;
    },
    staleTime: 30_000,
  });

  // ── Local edit draft ────────────────────────────────────────────
  //
  // `null` means "no edits in progress; render the server config".
  // Any mutator below seeds the draft from `serverConfig` (or the
  // default dashboard) if it's still null.
  const [draft, setDraft] = useState<DashboardConfig | null>(null);

  const baseline = serverConfig ?? DEFAULT_DASHBOARD;
  const config = draft ?? baseline;
  const isDirty = draft !== null;

  // updateDraft is the single funnel every mutator goes through.
  // It seeds the draft on first edit and applies the user's
  // updater to whatever the current draft (or baseline) is.
  const updateDraft = useCallback(
    (updater: (prev: DashboardConfig) => DashboardConfig) => {
      setDraft((prev) => updater(prev ?? baseline));
    },
    [baseline],
  );

  // ── Save ───────────────────────────────────────────────────────
  const saveMutation = useMutation({
    mutationFn: (cfg: DashboardConfig) => api.putDashboardConfig(cfg),
    onSuccess: (_, cfg) => {
      // Commit the draft to the react-query cache so other consumers
      // pick up the new baseline immediately, then clear the draft.
      queryClient.setQueryData<DashboardConfig | null>(QUERY_KEY, cfg);
      setDraft(null);
    },
  });

  const save = useCallback(() => {
    if (draft) saveMutation.mutate(draft);
  }, [draft, saveMutation]);

  const discard = useCallback(() => {
    setDraft(null);
  }, []);

  // ── Row mutators ──────────────────────────────────────────────

  const addRow = useCallback(
    (afterRowId?: string): string => {
      const newRow: DashboardRow = { id: genId('row'), tiles: [] };
      updateDraft((prev) => {
        if (!afterRowId) return { ...prev, rows: [...prev.rows, newRow] };
        const idx = prev.rows.findIndex((r) => r.id === afterRowId);
        if (idx < 0) return { ...prev, rows: [...prev.rows, newRow] };
        const rows = [...prev.rows];
        rows.splice(idx + 1, 0, newRow);
        return { ...prev, rows };
      });
      return newRow.id;
    },
    [updateDraft],
  );

  const removeRow = useCallback(
    (rowId: string) => {
      updateDraft((prev) => ({ ...prev, rows: prev.rows.filter((r) => r.id !== rowId) }));
    },
    [updateDraft],
  );

  const updateRow = useCallback(
    (rowId: string, patch: Partial<DashboardRow>) => {
      updateDraft((prev) => ({
        ...prev,
        rows: prev.rows.map((r) => (r.id === rowId ? { ...r, ...patch } : r)),
      }));
    },
    [updateDraft],
  );

  const moveRow = useCallback(
    (rowId: string, direction: 'up' | 'down') => {
      updateDraft((prev) => {
        const idx = prev.rows.findIndex((r) => r.id === rowId);
        if (idx < 0) return prev;
        const targetIdx = direction === 'up' ? idx - 1 : idx + 1;
        if (targetIdx < 0 || targetIdx >= prev.rows.length) return prev;
        const rows = [...prev.rows];
        [rows[idx], rows[targetIdx]] = [rows[targetIdx], rows[idx]];
        return { ...prev, rows };
      });
    },
    [updateDraft],
  );

  const reorderRows = useCallback(
    (rowIds: string[]) => {
      updateDraft((prev) => {
        const byId = new Map(prev.rows.map((r) => [r.id, r] as const));
        const next = rowIds
          .map((id) => byId.get(id))
          .filter((r): r is DashboardRow => !!r);
        for (const r of prev.rows) if (!rowIds.includes(r.id)) next.push(r);
        return { ...prev, rows: next };
      });
    },
    [updateDraft],
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
      updateDraft((prev) => ({
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
    [updateDraft],
  );

  const removeTile = useCallback(
    (rowId: string, tileId: string) => {
      updateDraft((prev) => ({
        ...prev,
        rows: prev.rows.map((r) =>
          r.id === rowId ? { ...r, tiles: r.tiles.filter((t) => t.id !== tileId) } : r,
        ),
      }));
    },
    [updateDraft],
  );

  const updateTile = useCallback(
    (rowId: string, tileId: string, patch: Partial<DashboardTile>) => {
      updateDraft((prev) => ({
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
    [updateDraft],
  );

  const moveTile = useCallback(
    (
      srcRowId: string,
      tileId: string,
      dstRowId: string,
      dstIdx: number | null,
    ) => {
      updateDraft((prev) => {
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
    [updateDraft],
  );

  const reset = useCallback(() => {
    updateDraft(() => DEFAULT_DASHBOARD);
  }, [updateDraft]);

  const exportJSON = useCallback((): string => JSON.stringify(config, null, 2), [config]);

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
        updateDraft(() => parsed);
        return { ok: true };
      } catch (err) {
        return {
          ok: false,
          error: err instanceof Error ? err.message : String(err),
        };
      }
    },
    [updateDraft],
  );

  return useMemo(
    () => ({
      config,
      isLoading,
      isDirty,

      save,
      discard,
      isSaving: saveMutation.isPending,
      saveError: saveMutation.error,

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
      isDirty,
      save,
      discard,
      saveMutation.isPending,
      saveMutation.error,
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
