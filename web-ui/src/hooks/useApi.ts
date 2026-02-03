import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as api from '../api/client';

// Query keys
export const queryKeys = {
  testRuns: (testId?: string) => ['testRuns', testId] as const,
  tests: ['tests'] as const,
  testRunDetails: (id: number) => ['testRunDetails', id] as const,
  taskDetails: (runId: number, taskIndex: number) => ['taskDetails', runId, taskIndex] as const,
  taskDescriptors: ['taskDescriptors'] as const,
  taskDescriptor: (name: string) => ['taskDescriptor', name] as const,
};

// Test runs list
export function useTestRuns(testId?: string) {
  return useQuery({
    queryKey: queryKeys.testRuns(testId),
    queryFn: () => api.getTestRuns(testId),
    refetchInterval: 5000,
  });
}

// Tests (registry) list
export function useTests() {
  return useQuery({
    queryKey: queryKeys.tests,
    queryFn: api.getTests,
    refetchInterval: 30000,
  });
}

// Clients list - reduced refresh interval since we use SSE for updates
export function useClients() {
  return useQuery({
    queryKey: ['clients'] as const,
    queryFn: api.getClients,
    refetchInterval: 5 * 60 * 1000, // 5 minutes - SSE handles live updates
  });
}

// Test run details
export function useTestRunDetails(
  runId: number,
  options?: { enabled?: boolean; refetchInterval?: number | false }
) {
  return useQuery({
    queryKey: queryKeys.testRunDetails(runId),
    queryFn: () => api.getTestRunDetails(runId),
    enabled: options?.enabled !== false && runId > 0,
    refetchInterval: options?.refetchInterval ?? false,
  });
}

// Task details
export function useTaskDetails(runId: number, taskIndex: number, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.taskDetails(runId, taskIndex),
    queryFn: () => api.getTaskDetails(runId, taskIndex),
    enabled: options?.enabled !== false && runId > 0 && taskIndex >= 0,
  });
}

// Task descriptors
export function useTaskDescriptors() {
  return useQuery({
    queryKey: queryKeys.taskDescriptors,
    queryFn: api.getTaskDescriptors,
    staleTime: 60000,
  });
}

export function useTaskDescriptor(name: string) {
  return useQuery({
    queryKey: queryKeys.taskDescriptor(name),
    queryFn: () => api.getTaskDescriptor(name),
    enabled: !!name,
    staleTime: 60000,
  });
}

// Mutations
export function useScheduleTestRun() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: api.scheduleTestRun,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['testRuns'] });
    },
  });
}

// Get test details
export function useTestDetails(testId: string, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['testDetails', testId] as const,
    queryFn: () => api.getTestDetails(testId),
    enabled: options?.enabled !== false && !!testId,
    staleTime: 60000,
  });
}

// Get test YAML source (for loading existing tests in builder)
export function useTestYaml(testId: string, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['testYaml', testId] as const,
    queryFn: () => api.getTestYaml(testId),
    enabled: options?.enabled !== false && !!testId,
    staleTime: 60000,
  });
}

export function useCancelTestRun() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: api.cancelTestRun,
    onSuccess: (_data, runId) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.testRunDetails(runId) });
      queryClient.invalidateQueries({ queryKey: ['testRuns'] });
    },
  });
}

export function useDeleteTestRuns() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: api.deleteTestRuns,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['testRuns'] });
    },
  });
}

export function useDeleteTest() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: api.deleteTest,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tests'] });
    },
  });
}

export function useRegisterTest() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: api.registerTest,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tests'] });
    },
  });
}

export function useRegisterExternalTest() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: api.registerExternalTest,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tests'] });
    },
  });
}
