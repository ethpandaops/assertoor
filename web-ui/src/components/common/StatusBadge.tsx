import type { TestStatus, TaskStatus, TaskResult } from '../../types/api';

type Status = TestStatus | TaskStatus | TaskResult;

interface StatusBadgeProps {
  status: Status;
  size?: 'sm' | 'md';
  progress?: number; // 0-100, only shown when status is 'running'
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

function StatusBadge({ status, size = 'md', progress }: StatusBadgeProps) {
  const config = statusConfig[status] || statusConfig.pending;

  const sizeClasses = size === 'sm' ? 'px-1.5 py-0.5 text-xs' : 'px-2 py-1 text-sm';
  const isRunning = status === 'running';
  const hasProgress = isRunning && progress !== undefined && progress > 0;

  return (
    <span className={`inline-flex items-center rounded-sm font-medium ${sizeClasses} ${config.className}`}>
      {isRunning && (
        hasProgress ? (
          <ProgressRing progress={progress} size={size === 'sm' ? 14 : 18} />
        ) : (
          <span className="size-1.5 bg-current rounded-full mr-1.5 animate-pulse" />
        )
      )}
      {config.label}
    </span>
  );
}

interface ProgressRingProps {
  progress: number;
  size: number;
}

function ProgressRing({ progress, size }: ProgressRingProps) {
  const strokeWidth = 2;
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (progress / 100) * circumference;
  const center = size / 2;

  return (
    <div className="relative mr-1.5 animate-[pulse-subtle_2s_ease-in-out_infinite]" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="transform -rotate-90">
        {/* Background circle */}
        <circle
          cx={center}
          cy={center}
          r={radius}
          fill="none"
          stroke="currentColor"
          strokeWidth={strokeWidth}
          opacity={0.2}
        />
        {/* Progress circle */}
        <circle
          cx={center}
          cy={center}
          r={radius}
          fill="none"
          stroke="currentColor"
          strokeWidth={strokeWidth}
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          strokeLinecap="round"
          className="transition-all duration-300"
        />
      </svg>
      {/* Percentage text */}
      <span
        className="absolute inset-0 flex items-center justify-center font-mono font-bold"
        style={{ fontSize: size * 0.35 }}
      >
        {Math.round(progress)}
      </span>
    </div>
  );
}

export default StatusBadge;
