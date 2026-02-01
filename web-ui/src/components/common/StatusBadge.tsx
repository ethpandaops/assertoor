import type { TestStatus, TaskStatus, TaskResult } from '../../types/api';

type Status = TestStatus | TaskStatus | TaskResult;

interface StatusBadgeProps {
  status: Status;
  size?: 'sm' | 'md';
}

const statusConfig: Record<string, { label: string; className: string }> = {
  // Test statuses
  pending: { label: 'Pending', className: 'status-pending' },
  running: { label: 'Running', className: 'status-running' },
  success: { label: 'Success', className: 'status-success' },
  failure: { label: 'Failure', className: 'status-failure' },
  aborted: { label: 'Aborted', className: 'status-aborted' },
  skipped: { label: 'Skipped', className: 'status-skipped' },
  // Task statuses
  complete: { label: 'Complete', className: 'status-complete' },
  // Task results
  none: { label: 'Pending', className: 'status-pending' },
};

function StatusBadge({ status, size = 'md' }: StatusBadgeProps) {
  const config = statusConfig[status] || statusConfig.pending;

  const sizeClasses = size === 'sm' ? 'px-1.5 py-0.5 text-xs' : 'px-2 py-1 text-sm';

  return (
    <span className={`inline-flex items-center rounded-sm font-medium ${sizeClasses} ${config.className}`}>
      {status === 'running' && (
        <span className="size-1.5 bg-current rounded-full mr-1.5 animate-pulse" />
      )}
      {config.label}
    </span>
  );
}

export default StatusBadge;
