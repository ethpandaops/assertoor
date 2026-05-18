import { useClients } from '../../hooks/useApi';
import { tileHeightStyle, type ClientStatusConfig, type DashboardTile } from './types';
import type { ClientData } from '../../types/api';

interface ClientStatusTileProps {
  tile: DashboardTile;
  config: ClientStatusConfig;
}

// ClientStatusTile renders one row per configured endpoint, with a
// pair of liveness dots for CL/EL plus the most recent head info.
// The point is to scan the fleet at a glance — no detail dives. The
// dedicated /clients page handles the deep dive.
export function ClientStatusTile({ tile, config }: ClientStatusTileProps) {
  const { data, isLoading, error } = useClients();
  const title = tile.title || 'Client status';
  const showExecution = config.showExecution !== false;

  const clients = data?.clients ?? [];

  return (
    <div
      className="card overflow-hidden h-full flex flex-col"
      style={tileHeightStyle(config.heightPx)}
    >
      <header className="card-header flex items-center justify-between">
        <span className="font-medium">{title}</span>
        <span className="text-xs text-[var(--color-text-tertiary)]">
          {clients.length} client{clients.length === 1 ? '' : 's'}
        </span>
      </header>

      <div className="flex-1 overflow-auto">
        {isLoading ? (
          <p className="p-3 text-xs text-[var(--color-text-tertiary)]">Loading…</p>
        ) : error ? (
          <p className="p-3 text-xs text-error-600">{error.message}</p>
        ) : clients.length === 0 ? (
          <p className="p-3 text-xs text-[var(--color-text-tertiary)] italic">
            No clients configured.
          </p>
        ) : (
          <table className="w-full text-xs">
            <thead className="text-[10px] uppercase tracking-wider text-[var(--color-text-tertiary)] border-b border-[var(--color-border)] sticky top-0 bg-[var(--color-bg-primary)]">
              <tr>
                <th className="text-left px-3 py-2">Client</th>
                <th className="text-left px-2 py-2 w-16">CL</th>
                <th className="text-left px-2 py-2">CL head</th>
                {showExecution && (
                  <>
                    <th className="text-left px-2 py-2 w-16">EL</th>
                    <th className="text-left px-2 py-2">EL head</th>
                  </>
                )}
              </tr>
            </thead>
            <tbody>
              {clients.map((c) => (
                <ClientRow key={c.index} client={c} showExecution={showExecution} />
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}

function ClientRow({ client, showExecution }: { client: ClientData; showExecution: boolean }) {
  return (
    <tr className="hover:bg-[var(--color-bg-tertiary)] border-b border-[var(--color-border)]">
      <td className="px-3 py-1.5">
        <div className="font-medium truncate" title={client.name}>
          {client.name}
        </div>
        {(client.cl_version || client.el_version) && (
          <div
            className="text-[10px] text-[var(--color-text-tertiary)] truncate font-mono"
            title={`${client.cl_version}\n${client.el_version}`}
          >
            {client.cl_version || client.el_version}
          </div>
        )}
      </td>
      <td className="px-2 py-1.5">
        <StatusDot status={client.cl_status} ready={client.cl_ready} />
      </td>
      <td className="px-2 py-1.5 font-mono">
        {client.cl_head_slot > 0 ? (
          <>
            <div>slot {client.cl_head_slot.toLocaleString()}</div>
            <div className="text-[10px] text-[var(--color-text-tertiary)] truncate">
              {shortRoot(client.cl_head_root)}
            </div>
          </>
        ) : (
          <span className="text-[var(--color-text-tertiary)]">–</span>
        )}
      </td>
      {showExecution && (
        <>
          <td className="px-2 py-1.5">
            <StatusDot status={client.el_status} ready={client.el_ready} />
          </td>
          <td className="px-2 py-1.5 font-mono">
            {client.el_head_number > 0 ? (
              <>
                <div>#{client.el_head_number.toLocaleString()}</div>
                <div className="text-[10px] text-[var(--color-text-tertiary)] truncate">
                  {shortRoot(client.el_head_hash)}
                </div>
              </>
            ) : (
              <span className="text-[var(--color-text-tertiary)]">–</span>
            )}
          </td>
        </>
      )}
    </tr>
  );
}

function StatusDot({ status, ready }: { status: string; ready: boolean }) {
  // Three buckets keep the status legend small: ready (green),
  // online-but-not-ready (amber), anything else (red/gray).
  const cls = ready
    ? 'bg-green-500'
    : status === 'online'
      ? 'bg-amber-500'
      : status === 'synchronizing' || status === 'optimistic'
        ? 'bg-blue-500 animate-pulse'
        : 'bg-red-500';

  return (
    <span className="inline-flex items-center gap-1.5" title={status}>
      <span className={`size-2 rounded-full ${cls}`} />
      <span className="text-[11px] text-[var(--color-text-secondary)]">{status}</span>
    </span>
  );
}

function shortRoot(root: string): string {
  if (!root || /^0x0+$/.test(root)) return '';
  if (root.length <= 14) return root;
  return `${root.slice(0, 8)}…${root.slice(-4)}`;
}

export default ClientStatusTile;
