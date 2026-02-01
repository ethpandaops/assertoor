import { useClients } from '../hooks/useApi';
import { useClientEvents } from '../hooks/useClientEvents';
import { formatRelativeTime } from '../utils/time';
import type { ClientData } from '../types/api';

function Clients() {
  const { data, isLoading, error } = useClients();

  // Subscribe to client SSE events for real-time updates
  useClientEvents();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full size-8 border-b-2 border-primary-600"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="card p-6 text-center">
        <p className="text-error-600">Failed to load clients: {error.message}</p>
      </div>
    );
  }

  const clients = data?.clients || [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Clients</h1>
        <div className="text-sm text-[var(--color-text-secondary)]">
          {data?.client_count || 0} clients configured
        </div>
      </div>

      {clients.length === 0 ? (
        <div className="card p-12 text-center">
          <p className="text-[var(--color-text-secondary)]">No clients configured.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {clients.map((client) => (
            <ClientCard key={client.index} client={client} />
          ))}
        </div>
      )}
    </div>
  );
}

interface ClientCardProps {
  client: ClientData;
}

function ClientCard({ client }: ClientCardProps) {
  return (
    <div className="card overflow-hidden">
      <div className="card-header flex items-center justify-between">
        <div className="flex items-center gap-3">
          <span className="font-semibold">{client.name}</span>
          <span className="text-xs text-[var(--color-text-tertiary)]">#{client.index}</span>
        </div>
        <div className="flex items-center gap-2">
          <ClientStatusIndicator ready={client.cl_ready} label="CL" />
          <ClientStatusIndicator ready={client.el_ready} label="EL" />
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 divide-y md:divide-y-0 md:divide-x divide-[var(--color-border)]">
        {/* Consensus Layer */}
        <div className="p-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-medium text-sm">Consensus Layer</h3>
            <ClientStatusBadge status={client.cl_status} ready={client.cl_ready} />
          </div>

          <div className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-[var(--color-text-secondary)]">Version</span>
              <span className="font-mono text-xs truncate max-w-xs">{client.cl_version || '-'}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-[var(--color-text-secondary)]">Head Slot</span>
              <span className="font-mono">{client.cl_head_slot}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-[var(--color-text-secondary)]">Head Root</span>
              <span className="font-mono text-xs truncate max-w-xs" title={client.cl_head_root}>
                {formatHash(client.cl_head_root)}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-[var(--color-text-secondary)]">Last Update</span>
              <span>{formatRelativeTime(client.cl_refresh)}</span>
            </div>
            {client.cl_error && (
              <div className="mt-2 p-2 bg-error-50 dark:bg-error-900/20 rounded-xs">
                <span className="text-error-600 text-xs">{client.cl_error}</span>
              </div>
            )}
          </div>
        </div>

        {/* Execution Layer */}
        <div className="p-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-medium text-sm">Execution Layer</h3>
            <ClientStatusBadge status={client.el_status} ready={client.el_ready} />
          </div>

          <div className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-[var(--color-text-secondary)]">Version</span>
              <span className="font-mono text-xs truncate max-w-xs">{client.el_version || '-'}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-[var(--color-text-secondary)]">Head Block</span>
              <span className="font-mono">{client.el_head_number}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-[var(--color-text-secondary)]">Head Hash</span>
              <span className="font-mono text-xs truncate max-w-xs" title={client.el_head_hash}>
                {formatHash(client.el_head_hash)}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-[var(--color-text-secondary)]">Last Update</span>
              <span>{formatRelativeTime(client.el_refresh)}</span>
            </div>
            {client.el_error && (
              <div className="mt-2 p-2 bg-error-50 dark:bg-error-900/20 rounded-xs">
                <span className="text-error-600 text-xs">{client.el_error}</span>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function ClientStatusIndicator({ ready, label }: { ready: boolean; label: string }) {
  return (
    <div className="flex items-center gap-1">
      <span className={`size-2 rounded-full ${ready ? 'bg-green-500' : 'bg-red-500'}`} />
      <span className="text-xs text-[var(--color-text-tertiary)]">{label}</span>
    </div>
  );
}

function ClientStatusBadge({ status, ready }: { status: string; ready: boolean }) {
  const getStatusColor = () => {
    if (!ready) return 'text-red-600 bg-red-100 dark:text-red-400 dark:bg-red-900/30';
    switch (status) {
      case 'online':
        return 'text-green-600 bg-green-100 dark:text-green-400 dark:bg-green-900/30';
      case 'synchronizing':
        return 'text-yellow-600 bg-yellow-100 dark:text-yellow-400 dark:bg-yellow-900/30';
      case 'optimistic':
        return 'text-blue-600 bg-blue-100 dark:text-blue-400 dark:bg-blue-900/30';
      case 'offline':
        return 'text-red-600 bg-red-100 dark:text-red-400 dark:bg-red-900/30';
      default:
        return 'text-gray-600 bg-gray-100 dark:text-gray-400 dark:bg-gray-900/30';
    }
  };

  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-xs text-xs font-medium ${getStatusColor()}`}>
      {status === 'online' && ready && (
        <span className="size-1.5 bg-current rounded-full mr-1" />
      )}
      {status === 'synchronizing' && (
        <span className="size-1.5 bg-current rounded-full mr-1 animate-pulse" />
      )}
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  );
}

function formatHash(hash: string): string {
  if (!hash || hash.length < 16) return hash || '-';
  return `${hash.slice(0, 10)}...${hash.slice(-6)}`;
}

export default Clients;
