import { useEffect, useRef, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import type { SSEEvent } from '../types/api';
import { queryKeys } from './useApi';
import { authStore } from '../stores/authStore';

interface UseEventStreamOptions {
  runId?: number;
  onEvent?: (event: SSEEvent) => void;
  enableDefaultInvalidation?: boolean;
}

export function useEventStream(options: UseEventStreamOptions = {}) {
  const { runId, onEvent, enableDefaultInvalidation = false } = options;
  const queryClient = useQueryClient();
  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleEvent = useCallback(
    (event: SSEEvent) => {
      // Call custom handler if provided
      onEvent?.(event);

      if (!enableDefaultInvalidation) {
        return;
      }

      // Update React Query cache based on event type
      switch (event.type) {
        case 'test.started':
        case 'test.completed':
        case 'test.failed':
          queryClient.invalidateQueries({ queryKey: ['testRuns'] });
          queryClient.invalidateQueries({ queryKey: queryKeys.testRunDetails(event.testRunId) });
          break;

        case 'task.started':
        case 'task.completed':
        case 'task.failed':
          queryClient.invalidateQueries({ queryKey: queryKeys.testRunDetails(event.testRunId) });
          if (event.taskIndex !== undefined) {
            queryClient.invalidateQueries({
              queryKey: queryKeys.taskDetails(event.testRunId, event.taskIndex),
            });
          }
          break;

        case 'task.progress':
          // Update task progress in cache without full refetch
          if (event.taskIndex !== undefined) {
            queryClient.setQueryData(
              queryKeys.testRunDetails(event.testRunId),
              (oldData: { tasks?: { index: number; progress: number; progressMessage?: string }[] }) => {
                if (!oldData?.tasks) return oldData;
                return {
                  ...oldData,
                  tasks: oldData.tasks.map((task) =>
                    task.index === event.taskIndex
                      ? {
                          ...task,
                          progress: (event.data.progress as number) ?? task.progress,
                          progressMessage: (event.data.message as string) ?? task.progressMessage,
                        }
                      : task
                  ),
                };
              }
            );
          }
          break;

        case 'task.log':
          // Log events can be handled by specific components
          break;
      }
    },
    [queryClient, onEvent, enableDefaultInvalidation]
  );

  const connect = useCallback(() => {
    // Close existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    // Clear any pending reconnect
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    // Build URL based on whether we're watching a specific run
    const authHeader = authStore.getAuthHeader();
    const authToken = authHeader?.startsWith('Bearer ') ? authHeader.slice('Bearer '.length) : undefined;
    const baseUrl = runId ? `/api/v1/test_run/${runId}/events` : '/api/v1/events/stream';
    const url = authToken ? `${baseUrl}?token=${encodeURIComponent(authToken)}` : baseUrl;

    const eventSource = new EventSource(url);
    eventSourceRef.current = eventSource;

    const handleMessage = (event: MessageEvent) => {
      try {
        const data: SSEEvent = JSON.parse(event.data);
        handleEvent(data);
      } catch (error) {
        console.error('Failed to parse SSE event:', error);
      }
    };

    eventSource.onmessage = handleMessage;

    const eventTypes = [
      'connected',
      'test.started',
      'test.completed',
      'test.failed',
      'task.created',
      'task.started',
      'task.progress',
      'task.completed',
      'task.failed',
      'task.log',
    ];

    eventTypes.forEach((eventType) => {
      eventSource.addEventListener(eventType, handleMessage);
    });

    eventSource.onerror = () => {
      eventSource.close();
      // Reconnect after 5 seconds
      reconnectTimeoutRef.current = setTimeout(connect, 5000);
    };
  }, [runId, handleEvent]);

  useEffect(() => {
    connect();

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
    };
  }, [connect]);

  const disconnect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
  }, []);

  return { disconnect, reconnect: connect };
}
