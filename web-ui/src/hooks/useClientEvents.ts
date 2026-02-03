import { useEffect, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import type { ClientsPage, ClientHeadUpdateEvent, ClientStatusUpdateEvent } from '../types/api';

interface ClientEventData {
  type: string;
  data: ClientHeadUpdateEvent | ClientStatusUpdateEvent;
}

export function useClientEvents() {
  const queryClient = useQueryClient();

  const handleEvent = useCallback(
    (event: ClientEventData) => {
      queryClient.setQueryData(['clients'], (oldData: ClientsPage | undefined) => {
        if (!oldData?.clients) return oldData;

        const updatedClients = oldData.clients.map((client) => {
          if (event.type === 'client.head_update') {
            const data = event.data as ClientHeadUpdateEvent;
            if (client.index === data.clientIndex) {
              return {
                ...client,
                cl_head_slot: data.clHeadSlot,
                cl_head_root: data.clHeadRoot,
                el_head_number: data.elHeadNumber,
                el_head_hash: data.elHeadHash,
                cl_refresh: new Date().toISOString(),
                el_refresh: new Date().toISOString(),
              };
            }
          } else if (event.type === 'client.status_update') {
            const data = event.data as ClientStatusUpdateEvent;
            if (client.index === data.clientIndex) {
              return {
                ...client,
                cl_status: data.clStatus,
                cl_ready: data.clReady,
                el_status: data.elStatus,
                el_ready: data.elReady,
              };
            }
          }
          return client;
        });

        return {
          ...oldData,
          clients: updatedClients,
        };
      });
    },
    [queryClient]
  );

  useEffect(() => {
    const eventSource = new EventSource('/api/v1/events/clients');

    eventSource.addEventListener('client.head_update', (e) => {
      try {
        const eventData = JSON.parse(e.data);
        handleEvent({ type: 'client.head_update', data: eventData.data || eventData });
      } catch (err) {
        console.warn('Failed to parse client.head_update event:', err);
      }
    });

    eventSource.addEventListener('client.status_update', (e) => {
      try {
        const eventData = JSON.parse(e.data);
        handleEvent({ type: 'client.status_update', data: eventData.data || eventData });
      } catch (err) {
        console.warn('Failed to parse client.status_update event:', err);
      }
    });

    eventSource.onerror = () => {
      // EventSource will automatically try to reconnect
      console.debug('Client SSE connection error, will reconnect');
    };

    return () => {
      eventSource.close();
    };
  }, [handleEvent]);
}
