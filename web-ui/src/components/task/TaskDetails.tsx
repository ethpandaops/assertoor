import { useState, useMemo, useRef, useEffect, useCallback } from 'react';
import { useTaskDetails } from '../../hooks/useApi';
import { useAuthContext } from '../../context/AuthContext';
import type { TaskState, TaskLogEntry } from '../../types/api';
import { formatDurationMs, formatTime } from '../../utils/time';

interface TaskDetailsProps {
  runId: number;
  task: TaskState;
  sseLogs: TaskLogEntry[];
}

function TaskDetails({ runId, task, sseLogs }: TaskDetailsProps) {
  const [activeTab, setActiveTab] = useState<'overview' | 'logs' | 'config' | 'result'>('overview');
  const { isLoggedIn } = useAuthContext();
  const { data: details, error: detailsError } = useTaskDetails(runId, task.index, { enabled: isLoggedIn });
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  const tabs = [
    { id: 'overview' as const, label: 'Overview' },
    { id: 'logs' as const, label: 'Logs', requiresAuth: true },
    { id: 'config' as const, label: 'Config', requiresAuth: true },
    { id: 'result' as const, label: 'Result', requiresAuth: true },
  ];

  // Check if current tab requires auth
  const currentTabRequiresAuth = tabs.find(t => t.id === activeTab)?.requiresAuth;
  const showAuthMessage = currentTabRequiresAuth && !isLoggedIn;

  // Merge API-fetched logs with SSE logs using a watermark approach.
  // When API details load, all SSE logs accumulated up to that point are already
  // included in the API response. We record that count as the watermark and only
  // append SSE logs received after it.
  const detailsLoadedRef = useRef(false);
  const sseWatermarkRef = useRef(0);

  const mergedLogs = useMemo(() => {
    const apiLogs = details?.log || [];

    // When API logs first become available, set the watermark
    if (!detailsLoadedRef.current && apiLogs.length > 0) {
      detailsLoadedRef.current = true;
      sseWatermarkRef.current = sseLogs.length;
    }

    // SSE logs received after the API fetch
    const newSseLogs = sseLogs.slice(sseWatermarkRef.current);

    if (apiLogs.length === 0 && !detailsLoadedRef.current) return sseLogs;
    if (newSseLogs.length === 0) return apiLogs;
    return [...apiLogs, ...newSseLogs];
  }, [details?.log, sseLogs]);

  return (
    <div className="flex flex-col h-full">
      {/* Tabs */}
      <div className="flex border-b border-[var(--color-border)]">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`px-3 py-1.5 text-xs font-medium border-b-2 transition-colors ${
              activeTab === tab.id
                ? 'border-primary-600 text-primary-600'
                : 'border-transparent text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]'
            }`}
          >
            {tab.label}
            {tab.requiresAuth && !isLoggedIn && (
              <span className="ml-1 text-[var(--color-text-tertiary)]" title="Requires login">🔒</span>
            )}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div ref={scrollContainerRef} className="flex-1 overflow-y-auto p-3">
        {showAuthMessage ? (
          <div className="text-center py-6">
            <p className="text-[var(--color-text-secondary)] text-sm mb-2">
              Log in to view task {activeTab}
            </p>
            <p className="text-[var(--color-text-tertiary)] text-xs">
              Task details contain potentially sensitive configuration and logs
            </p>
          </div>
        ) : detailsError && activeTab !== 'overview' ? (
          <div className="text-center py-6">
            <p className="text-error-600 text-sm">
              Failed to load task details
            </p>
          </div>
        ) : (
          <>
            {activeTab === 'overview' && <OverviewTab task={task} />}
            {activeTab === 'logs' && <LogsTab logs={mergedLogs} scrollContainerRef={scrollContainerRef} />}
            {activeTab === 'config' && <ConfigTab yaml={details?.config_yaml} />}
            {activeTab === 'result' && <ResultTab yaml={details?.result_yaml} />}
          </>
        )}
      </div>
    </div>
  );
}

function OverviewTab({ task }: { task: TaskState }) {
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    if (task.status !== 'running') return;
    const interval = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(interval);
  }, [task.status]);

  // Calculate duration: live for running tasks, static for completed
  const duration =
    task.status === 'running' && task.start_time > 0
      ? now - task.start_time
      : task.runtime || 0;

  // Determine status display
  const getStatusText = () => {
    if (task.status === 'running') return 'Running';
    if (task.status === 'complete') {
      if (task.result === 'success') return 'Success';
      if (task.result === 'failure') return 'Failure';
      return 'Complete';
    }
    return 'Pending';
  };

  return (
    <div className="space-y-3">
      <div>
        <h4 className="text-sm font-medium text-[var(--color-text-secondary)]">Task Name</h4>
        <p className="mt-1 font-mono text-xs">{task.name}</p>
      </div>

      {task.title && task.title !== task.name && (
        <div>
          <h4 className="text-sm font-medium text-[var(--color-text-secondary)]">Title</h4>
          <p className="mt-1 text-sm">{task.title}</p>
        </div>
      )}

      <div>
        <h4 className="text-sm font-medium text-[var(--color-text-secondary)]">Status</h4>
        <p className="mt-1">{getStatusText()}</p>
      </div>

      {task.started && (
        <div>
          <h4 className="text-sm font-medium text-[var(--color-text-secondary)]">Duration</h4>
          <p className="mt-1 font-mono text-sm">{formatDurationMs(duration)}</p>
        </div>
      )}

      {task.timeout > 0 && (
        <div>
          <h4 className="text-sm font-medium text-[var(--color-text-secondary)]">Timeout</h4>
          <p className="mt-1 font-mono text-sm">{formatDurationMs(task.timeout)}</p>
        </div>
      )}

      {task.result_error && (
        <div>
          <h4 className="text-sm font-medium text-error-600 dark:text-error-400">Error</h4>
          <p className="mt-1 text-error-700 dark:text-error-300 font-mono text-xs bg-error-50 dark:bg-error-900/30 p-2 rounded-sm border border-error-200 dark:border-error-800">
            {task.result_error}
          </p>
        </div>
      )}

      {task.result_files && task.result_files.length > 0 && (
        <div>
          <h4 className="text-sm font-medium text-[var(--color-text-secondary)]">Result Files</h4>
          <ul className="mt-1 space-y-1">
            {task.result_files.map((file) => (
              <li key={file.index}>
                <a
                  href={file.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-primary-600 hover:underline text-xs"
                >
                  {file.name} ({formatFileSize(file.size)})
                </a>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

interface LogsTabProps {
  logs: TaskLogEntry[];
  scrollContainerRef: React.RefObject<HTMLDivElement | null>;
}

function LogsTab({ logs, scrollContainerRef }: LogsTabProps) {
  const isAtBottomRef = useRef(true);
  const prevLogCountRef = useRef(0);

  // Check if container is scrolled to bottom
  const checkIsAtBottom = useCallback(() => {
    if (scrollContainerRef.current) {
      const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.current;
      // Consider "at bottom" if within 30px of the bottom
      return scrollHeight - scrollTop - clientHeight < 30;
    }
    return true;
  }, [scrollContainerRef]);

  // Track scroll position via scroll event
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const handleScroll = () => {
      isAtBottomRef.current = checkIsAtBottom();
    };

    // Initial check
    isAtBottomRef.current = checkIsAtBottom();

    container.addEventListener('scroll', handleScroll, { passive: true });
    return () => container.removeEventListener('scroll', handleScroll);
  }, [scrollContainerRef, checkIsAtBottom]);

  // Auto-scroll to bottom when new logs are added (only if was at bottom)
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    // Only scroll if we have new logs and were at bottom
    if (logs.length > prevLogCountRef.current && isAtBottomRef.current) {
      // Use requestAnimationFrame to ensure DOM has updated
      requestAnimationFrame(() => {
        container.scrollTop = container.scrollHeight;
      });
    }
    prevLogCountRef.current = logs.length;
  }, [logs.length, scrollContainerRef]);

  // Initial scroll to bottom when logs first load
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (container && logs.length > 0 && prevLogCountRef.current === 0) {
      requestAnimationFrame(() => {
        container.scrollTop = container.scrollHeight;
        isAtBottomRef.current = true;
      });
    }
  }, [logs.length, scrollContainerRef]);

  if (logs.length === 0) {
    return (
      <p className="text-center text-[var(--color-text-secondary)] text-xs">No logs available</p>
    );
  }

  const getLevelStyle = (level: string): { class: string; label: string } => {
    switch (level) {
      case 'trace':
        return { class: 'text-gray-400', label: 'TRC' };
      case 'debug':
        return { class: 'text-gray-500', label: 'DBG' };
      case 'info':
        return { class: 'text-blue-500', label: 'INF' };
      case 'warn':
        return { class: 'text-yellow-500', label: 'WRN' };
      case 'error':
        return { class: 'text-red-500', label: 'ERR' };
      case 'fatal':
      case 'panic':
        return { class: 'text-red-600 font-bold', label: level.toUpperCase().slice(0, 3) };
      default:
        return { class: 'text-gray-500', label: level.toUpperCase().slice(0, 3) };
    }
  };

  return (
    <div className="font-mono text-[11px] space-y-0.5">
      {logs.map((log, i) => {
        const style = getLevelStyle(log.level);
        return (
          <div key={i} className="flex flex-wrap leading-tight">
            <span className="text-[var(--color-text-tertiary)] mr-1.5 shrink-0">
              {formatTime(log.time)}
            </span>
            <span className={`mr-1.5 shrink-0 w-8 ${style.class}`}>
              [{style.label}]
            </span>
            <span className="break-all flex-1">{log.msg}</span>
            {log.datalen > 0 && (
              <div className="w-full ml-20 text-[10px] text-[var(--color-text-tertiary)]">
                {Object.entries(log.data).map(([key, value]) => (
                  <span key={key} className="mr-3">
                    <span className="text-[var(--color-text-secondary)]">{key}:</span> {value}
                  </span>
                ))}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

function ConfigTab({ yaml }: { yaml?: string }) {
  if (!yaml) {
    return (
      <p className="text-center text-[var(--color-text-secondary)]">No configuration available</p>
    );
  }

  return (
    <pre className="bg-[var(--color-bg-tertiary)] p-2 rounded-sm text-xs overflow-x-auto font-mono whitespace-pre">
      {yaml}
    </pre>
  );
}

function ResultTab({ yaml }: { yaml?: string }) {
  if (!yaml) {
    return (
      <p className="text-center text-[var(--color-text-secondary)]">No result data available</p>
    );
  }

  return (
    <pre className="bg-[var(--color-bg-tertiary)] p-2 rounded-sm text-xs overflow-x-auto font-mono whitespace-pre">
      {yaml}
    </pre>
  );
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export default TaskDetails;
