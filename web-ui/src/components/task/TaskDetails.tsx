import { useState } from 'react';
import { useTaskDetails } from '../../hooks/useApi';
import type { TaskState, TaskLogEntry } from '../../types/api';
import { formatDurationMs, formatTime } from '../../utils/time';

interface TaskDetailsProps {
  runId: number;
  task: TaskState;
}

function TaskDetails({ runId, task }: TaskDetailsProps) {
  const [activeTab, setActiveTab] = useState<'overview' | 'logs' | 'config' | 'result'>('overview');
  const { data: details } = useTaskDetails(runId, task.index);

  const tabs = [
    { id: 'overview' as const, label: 'Overview' },
    { id: 'logs' as const, label: 'Logs' },
    { id: 'config' as const, label: 'Config' },
    { id: 'result' as const, label: 'Result' },
  ];

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
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-y-auto p-3">
        {activeTab === 'overview' && <OverviewTab task={task} />}
        {activeTab === 'logs' && <LogsTab logs={details?.log || []} />}
        {activeTab === 'config' && <ConfigTab yaml={details?.config_yaml} />}
        {activeTab === 'result' && <ResultTab yaml={details?.result_yaml} />}
      </div>
    </div>
  );
}

function OverviewTab({ task }: { task: TaskState }) {
  // Calculate duration
  const duration = task.runtime || 0;

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

function LogsTab({ logs }: { logs: TaskLogEntry[] }) {
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
