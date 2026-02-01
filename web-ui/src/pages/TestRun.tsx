import { useState, useMemo, useCallback, useRef, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { useTestRunDetails, useCancelTestRun, queryKeys } from '../hooks/useApi';
import { useEventStream } from '../hooks/useEventStream';
import StatusBadge from '../components/common/StatusBadge';
import SplitPane from '../components/common/SplitPane';
import TaskList from '../components/task/TaskList';
import TaskDetails from '../components/task/TaskDetails';
import { TaskGraph } from '../components/graph';
import { formatDuration, formatRelativeTime } from '../utils/time';
import * as api from '../api/client';
import type { SSEEvent, TaskDetails as TaskDetailsType, TaskLogEntry, TestRunDetails } from '../types/api';

type ViewMode = 'list' | 'graph';

const VIEW_MODE_STORAGE_KEY = 'testrun-view-mode';

function TestRun() {
  const { runId } = useParams<{ runId: string }>();
  const runIdNum = parseInt(runId || '0', 10);
  const [selectedTaskIndex, setSelectedTaskIndex] = useState<number | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>(() => {
    const stored = localStorage.getItem(VIEW_MODE_STORAGE_KEY);
    return (stored === 'list' || stored === 'graph') ? stored : 'list';
  });
  const queryClient = useQueryClient();
  const pendingTaskRefreshRef = useRef<Set<number>>(new Set());
  const refreshTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Persist view mode preference
  useEffect(() => {
    localStorage.setItem(VIEW_MODE_STORAGE_KEY, viewMode);
  }, [viewMode]);

  const flushTaskRefresh = useCallback(async () => {
    if (refreshTimerRef.current) {
      clearTimeout(refreshTimerRef.current);
      refreshTimerRef.current = null;
    }

    const taskIndexes = Array.from(pendingTaskRefreshRef.current);
    pendingTaskRefreshRef.current.clear();

    if (taskIndexes.length === 0 || runIdNum <= 0) {
      return;
    }

    await Promise.all(
      taskIndexes.map(async (taskIndex) => {
        try {
          const taskDetails = await queryClient.fetchQuery({
            queryKey: queryKeys.taskDetails(runIdNum, taskIndex),
            queryFn: () => api.getTaskDetails(runIdNum, taskIndex),
          });

          queryClient.setQueryData(queryKeys.testRunDetails(runIdNum), (oldData?: TestRunDetails) => {
            if (!oldData?.tasks) return oldData;
            return {
              ...oldData,
              tasks: oldData.tasks.map((task) => (task.index === taskIndex ? { ...task, ...taskDetails } : task)),
            };
          });
        } catch (err) {
          console.warn('Failed to refresh task details', { runId: runIdNum, taskIndex, err });
        }
      })
    );
  }, [queryClient, runIdNum]);

  const scheduleTaskRefresh = useCallback(
    (taskIndex: number) => {
      pendingTaskRefreshRef.current.add(taskIndex);

      if (!refreshTimerRef.current) {
        refreshTimerRef.current = setTimeout(flushTaskRefresh, 5000);
      }
    },
    [flushTaskRefresh]
  );

  const appendTaskLog = useCallback(
    (taskIndex: number, logEntry: TaskLogEntry) => {
      queryClient.setQueryData(
        queryKeys.taskDetails(runIdNum, taskIndex),
        (oldData?: TaskDetailsType) => {
          if (!oldData) return oldData;
          const nextLog = oldData.log ? [...oldData.log, logEntry] : [logEntry];
          return {
            ...oldData,
            log: nextLog,
          };
        }
      );
    },
    [queryClient, runIdNum]
  );

  const handleEvent = useCallback(
    (event: SSEEvent) => {
      if (event.testRunId !== runIdNum) return;

      switch (event.type) {
        case 'test.started':
        case 'test.completed':
        case 'test.failed':
          queryClient.invalidateQueries({ queryKey: queryKeys.testRunDetails(runIdNum) });
          break;
        case 'task.created':
          // Add new task to the task list
          if (event.taskIndex !== undefined) {
            const data = event.data as {
              taskName?: string;
              taskTitle?: string;
              taskId?: string;
              parentIndex?: number;
            };
            queryClient.setQueryData(queryKeys.testRunDetails(runIdNum), (oldData?: TestRunDetails) => {
              if (!oldData?.tasks) return oldData;
              // Check if task already exists
              if (oldData.tasks.some((t) => t.index === event.taskIndex)) {
                return oldData;
              }
              // Add new task
              const newTask = {
                index: event.taskIndex!,
                parent_index: data.parentIndex ?? -1,
                name: data.taskName ?? '',
                title: data.taskTitle ?? data.taskName ?? '',
                started: false,
                completed: false,
                start_time: 0,
                stop_time: 0,
                timeout: 0,
                runtime: 0,
                status: 'pending' as const,
                result: 'none' as const,
                result_error: '',
                progress: 0,
                progress_message: '',
              };
              return {
                ...oldData,
                tasks: [...oldData.tasks, newTask],
              };
            });
          }
          break;
        case 'task.started':
        case 'task.completed':
        case 'task.failed':
        case 'task.progress':
          if (event.taskIndex !== undefined) {
            scheduleTaskRefresh(event.taskIndex);
          }
          break;
        case 'task.log':
          if (event.taskIndex !== undefined) {
            const data = event.data as {
              level?: string;
              message?: string;
              fields?: Record<string, unknown>;
              timestamp?: string;
            };
            const fields = Object.entries(data.fields ?? {}).reduce<Record<string, string>>((acc, [key, value]) => {
              acc[key] = value === undefined ? '' : String(value);
              return acc;
            }, {});
            const logEntry: TaskLogEntry = {
              time: data.timestamp ?? new Date().toISOString(),
              level: mapLogLevelToUI(data.level),
              msg: data.message ?? '',
              datalen: Object.keys(fields).length,
              data: fields,
            };
            appendTaskLog(event.taskIndex, logEntry);
          }
          break;
      }
    },
    [appendTaskLog, queryClient, runIdNum, scheduleTaskRefresh]
  );

  // Subscribe to SSE events for this run
  useEventStream({ runId: runIdNum, onEvent: handleEvent, enableDefaultInvalidation: false });

  useEffect(() => {
    return () => {
      if (refreshTimerRef.current) {
        clearTimeout(refreshTimerRef.current);
        refreshTimerRef.current = null;
      }
    };
  }, []);

  const { data: details, isLoading, error } = useTestRunDetails(runIdNum, { refetchInterval: false });
  const cancelMutation = useCancelTestRun();

  // Calculate task statistics
  const taskStats = useMemo(() => {
    if (!details?.tasks) return { total: 0, completed: 0, passed: 0, failed: 0 };

    let completed = 0;
    let passed = 0;
    let failed = 0;

    for (const task of details.tasks) {
      if (task.completed) {
        completed++;
        if (task.result === 'success') passed++;
        else if (task.result === 'failure') failed++;
      }
    }

    return { total: details.tasks.length, completed, passed, failed };
  }, [details?.tasks]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full size-8 border-b-2 border-primary-600"></div>
      </div>
    );
  }

  if (error || !details) {
    return (
      <div className="card p-6 text-center">
        <p className="text-error-600">
          {error ? `Failed to load test run: ${error.message}` : 'Test run not found'}
        </p>
        <Link to="/" className="text-primary-600 hover:underline mt-4 inline-block">
          Back to Dashboard
        </Link>
      </div>
    );
  }

  const handleCancel = async () => {
    if (confirm('Are you sure you want to cancel this test run?')) {
      try {
        await cancelMutation.mutateAsync(runIdNum);
      } catch (err) {
        console.error('Failed to cancel test run:', err);
      }
    }
  };

  const selectedTask = selectedTaskIndex !== null
    ? details.tasks.find((t) => t.index === selectedTaskIndex)
    : null;

  // Calculate current runtime
  const runtime = details.stop_time
    ? details.stop_time - details.start_time
    : details.start_time
      ? Math.floor(Date.now() / 1000) - details.start_time
      : 0;

  const canCancel = details.status === 'pending' || details.status === 'running';

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3">
            <Link to="/" className="text-[var(--color-text-secondary)] hover:text-primary-600">
              <ChevronLeftIcon className="size-5" />
            </Link>
            <h1 className="text-xl font-bold">Test Run #{details.run_id}</h1>
            <StatusBadge status={details.status} />
          </div>
          <p className="mt-1 text-[var(--color-text-secondary)]">
            {details.name} ({details.test_id})
          </p>
        </div>

        {canCancel && (
          <button
            onClick={handleCancel}
            disabled={cancelMutation.isPending}
            className="btn btn-danger btn-sm"
          >
            {cancelMutation.isPending ? 'Canceling...' : 'Cancel'}
          </button>
        )}
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
        <SummaryCard
          label="Tasks"
          value={`${taskStats.completed}/${taskStats.total}`}
        />
        <SummaryCard
          label="Passed"
          value={taskStats.passed.toString()}
          className="text-success-600"
        />
        <SummaryCard
          label="Failed"
          value={taskStats.failed.toString()}
          className="text-error-600"
        />
        <SummaryCard
          label="Duration"
          value={formatDuration(runtime)}
        />
      </div>

      {/* Timeline */}
      <div className="card p-3">
        <div className="flex items-center justify-between text-xs">
          <div>
            <span className="text-[var(--color-text-secondary)]">Started: </span>
            <span>{details.start_time ? formatRelativeTime(details.start_time) : 'Not started'}</span>
          </div>
          {details.stop_time > 0 && (
            <div>
              <span className="text-[var(--color-text-secondary)]">Finished: </span>
              <span>{formatRelativeTime(details.stop_time)}</span>
            </div>
          )}
        </div>
      </div>

      {/* Task list and details */}
      <SplitPane
        storageKey="testrun-tasks"
        defaultLeftWidth={40}
        minLeftWidth={25}
        maxLeftWidth={70}
        maxHeight="calc(95vh - 200px)"
        left={
          <div className="card overflow-hidden h-full flex flex-col">
            <div className="card-header flex items-center justify-between">
              <span>Tasks</span>
              <ViewModeToggle viewMode={viewMode} onViewModeChange={setViewMode} />
            </div>
            <div className="flex-1 overflow-hidden">
              {viewMode === 'list' ? (
                <TaskList
                  tasks={details.tasks}
                  selectedIndex={selectedTaskIndex}
                  onSelect={setSelectedTaskIndex}
                />
              ) : (
                <TaskGraph
                  tasks={details.tasks}
                  selectedIndex={selectedTaskIndex}
                  onSelect={setSelectedTaskIndex}
                />
              )}
            </div>
          </div>
        }
        right={
          <div className="card overflow-hidden h-full flex flex-col">
            <div className="card-header">Task Details</div>
            <div className="flex-1 overflow-hidden">
              {selectedTask ? (
                <TaskDetails runId={runIdNum} task={selectedTask} />
              ) : (
                <div className="p-4 text-center text-[var(--color-text-secondary)] text-sm">
                  Select a task to view details
                </div>
              )}
            </div>
          </div>
        }
      />
    </div>
  );
}

function SummaryCard({
  label,
  value,
  className = '',
}: {
  label: string;
  value: string;
  className?: string;
}) {
  return (
    <div className="card p-3">
      <div className="text-xs text-[var(--color-text-secondary)]">{label}</div>
      <div className={`text-xl font-bold ${className}`}>{value}</div>
    </div>
  );
}

function ChevronLeftIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
    </svg>
  );
}

function mapLogLevelToUI(level?: string): string {
  const l = (level || '').toLowerCase();
  if (['trace', 'debug', 'info', 'warn', 'warning', 'error', 'fatal', 'panic'].includes(l)) {
    return l === 'warning' ? 'warn' : l;
  }
  return 'info';
}

interface ViewModeToggleProps {
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
}

function ViewModeToggle({ viewMode, onViewModeChange }: ViewModeToggleProps) {
  return (
    <div className="flex items-center gap-1 bg-[var(--color-bg-tertiary)] rounded-md p-0.5">
      <button
        onClick={() => onViewModeChange('list')}
        className={`flex items-center gap-1 px-2 py-1 rounded text-xs transition-colors ${
          viewMode === 'list'
            ? 'bg-[var(--color-bg-primary)] text-[var(--color-text-primary)] shadow-sm'
            : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]'
        }`}
        title="List view"
      >
        <ListIcon className="size-3.5" />
        <span className="hidden sm:inline">List</span>
      </button>
      <button
        onClick={() => onViewModeChange('graph')}
        className={`flex items-center gap-1 px-2 py-1 rounded text-xs transition-colors ${
          viewMode === 'graph'
            ? 'bg-[var(--color-bg-primary)] text-[var(--color-text-primary)] shadow-sm'
            : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]'
        }`}
        title="Graph view"
      >
        <GraphIcon className="size-3.5" />
        <span className="hidden sm:inline">Graph</span>
      </button>
    </div>
  );
}

function ListIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
    </svg>
  );
}

function GraphIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M4 5a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM14 5a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1V5zM4 15a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1v-4zM14 15a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z"
      />
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 10v4M17 10v4M10 7h4" />
    </svg>
  );
}

export default TestRun;
