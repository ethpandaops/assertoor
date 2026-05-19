import { useQuery } from '@tanstack/react-query';
import { getNetworkStatus } from '../../api/client';
import type { DashboardTile, NetworkStatusConfig } from './types';

interface NetworkStatusTileProps {
  tile: DashboardTile;
  // NetworkStatus has no configurable fields yet. Keeping `config`
  // on the renderer signature for symmetry with the other tiles.
  config: NetworkStatusConfig;
}

// NetworkStatusTile renders an at-a-glance summary of the network's
// state — current slot/epoch, finalized + justified checkpoints, head
// info, client readiness, and the test-runner queue depth. Updates
// every 5s so the slot/epoch counter feels alive.
export function NetworkStatusTile({ tile }: NetworkStatusTileProps) {
  const { data, isLoading, error } = useQuery({
    queryKey: ['networkStatus'],
    queryFn: getNetworkStatus,
    refetchInterval: 5_000,
    staleTime: 2_000,
  });

  const title = tile.title || 'Network status';

  return (
    <div className="card overflow-hidden h-full flex flex-col">
      <header className="card-header flex items-center justify-between">
        <span className="font-medium">{title}</span>
        {data?.network_name && (
          <span className="text-xs text-[var(--color-text-tertiary)] font-mono">
            {data.network_name}
            {data.chain_id ? ` · chain ${data.chain_id}` : ''}
          </span>
        )}
      </header>

      <div className="p-4 flex-1 overflow-auto text-sm">
        {isLoading ? (
          <p className="text-xs text-[var(--color-text-tertiary)]">Loading…</p>
        ) : error ? (
          <p className="text-xs text-error-600">{error.message}</p>
        ) : !data ? (
          <p className="text-xs text-[var(--color-text-tertiary)] italic">
            No data.
          </p>
        ) : (
          <div className="grid grid-cols-2 gap-x-4 gap-y-2">
            <Field
              label="Current slot"
              value={data.current_slot.toLocaleString()}
              sub={`epoch ${data.current_epoch.toLocaleString()}`}
            />
            <Field
              label="Head slot"
              value={data.head_slot.toLocaleString()}
              sub={shortRoot(data.head_root)}
            />
            <Field
              label="Justified"
              value={`epoch ${data.justified_epoch.toLocaleString()}`}
              sub={shortRoot(data.justified_root)}
            />
            <Field
              label="Finalized"
              value={`epoch ${data.finalized_epoch.toLocaleString()}`}
              sub={shortRoot(data.finalized_root)}
            />
            <Field
              label="Clients"
              value={`${data.cl_ready_count + data.el_ready_count} / ${data.client_count * 2}`}
              sub={`CL ${data.cl_ready_count}/${data.client_count} · EL ${data.el_ready_count}/${data.client_count}`}
            />
            <Field
              label="Tests"
              value={`${data.tests_running} running`}
              sub={`${data.tests_queued} queued`}
            />
            {data.el_head_number > 0 && (
              <Field
                label="EL head"
                value={`#${data.el_head_number.toLocaleString()}`}
                sub={shortRoot(data.el_head_hash)}
                className="col-span-2"
              />
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function Field({
  label,
  value,
  sub,
  className,
}: {
  label: string;
  value: string;
  sub?: string;
  className?: string;
}) {
  return (
    <div className={className}>
      <div className="text-[10px] uppercase tracking-wider text-[var(--color-text-tertiary)] font-semibold">
        {label}
      </div>
      <div className="font-mono text-sm">{value}</div>
      {sub && (
        <div className="text-[11px] text-[var(--color-text-tertiary)] font-mono truncate">
          {sub}
        </div>
      )}
    </div>
  );
}

function shortRoot(root: string): string {
  if (!root || root === '0x0000000000000000000000000000000000000000000000000000000000000000') {
    return '';
  }
  if (root.length <= 14) return root;
  return `${root.slice(0, 8)}…${root.slice(-4)}`;
}

export default NetworkStatusTile;
