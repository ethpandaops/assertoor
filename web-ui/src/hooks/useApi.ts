import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as api from '../api/client';

// Query keys
export const queryKeys = {
  testRuns: (testId?: string) => ['testRuns', testId] as const,
  tests: ['tests'] as const,
  testRunDetails: (id: number) => ['testRunDetails', id] as const,
  testRunResult: (id: number) => ['testRunResult', id] as const,
  testLatestResult: (testId: string) => ['testLatestResult', testId] as const,
  testNextRun: (testId: string) => ['testNextRun', testId] as const,
  testQueue: ['testQueue'] as const,
  taskDetails: (runId: number, taskIndex: number) => ['taskDetails', runId, taskIndex] as const,
  taskDescriptors: ['taskDescriptors'] as const,
  taskDescriptor: (name: string) => ['taskDescriptor', name] as const,
  playbookLibrary: ['playbookLibrary'] as const,
  playbookLibraryCheck: (file: string) => ['playbookLibraryCheck', file] as const,
};

// Test runs list
export function useTestRuns(
  testId?: string,
  options?: { enabled?: boolean; refetchInterval?: number | false; staleTime?: number },
) {
  return useQuery({
    queryKey: queryKeys.testRuns(testId),
    queryFn: () => api.getTestRuns(testId),
    enabled: options?.enabled !== false,
    refetchInterval: options?.refetchInterval ?? 5000,
    staleTime: options?.staleTime,
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

// Run-level Result markdown. Returns null when the run has not produced
// a $ASSERTOOR_TEST_RESULT blob (HTTP 204).
export function useTestRunResult(
  runId: number,
  options?: { enabled?: boolean; refetchInterval?: number | false }
) {
  return useQuery({
    queryKey: queryKeys.testRunResult(runId),
    queryFn: () => api.getTestRunResult(runId),
    enabled: options?.enabled !== false && runId > 0,
    refetchInterval: options?.refetchInterval ?? false,
  });
}

// Latest run-level result markdown for a test. The envelope contains
// metadata + markdown body, or an empty envelope when no run has
// produced a result yet.
export function useLatestTestResult(
  testId: string,
  options?: { enabled?: boolean; refetchInterval?: number | false; staleTime?: number },
) {
  return useQuery({
    queryKey: queryKeys.testLatestResult(testId),
    queryFn: () => api.getLatestTestResult(testId),
    enabled: options?.enabled !== false && !!testId,
    refetchInterval: options?.refetchInterval ?? 30_000,
    staleTime: options?.staleTime ?? 10_000,
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

// Global variable names (for builder suggestions)
export function useGlobalVariables(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ['globalVariables'] as const,
    queryFn: api.getGlobalVariables,
    enabled: options?.enabled !== false,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

// Test schedule (cron + startup).
export function useTestNextRun(testId: string, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.testNextRun(testId),
    queryFn: () => api.getTestNextRun(testId),
    enabled: options?.enabled !== false && !!testId,
    // refresh every 30s so the displayed "next run" tick moves
    // forward at a reasonable cadence.
    refetchInterval: 30_000,
    staleTime: 15_000,
  });
}

export function useUpdateTestSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (args: { testId: string; schedule: import('../types/api').TestSchedule | null }) =>
      api.updateTestSchedule(args.testId, args.schedule),
    onSuccess: (_, args) => {
      queryClient.invalidateQueries({ queryKey: ['testDetails', args.testId] });
      queryClient.invalidateQueries({ queryKey: queryKeys.testNextRun(args.testId) });
    },
  });
}

// Live runner queue (running + pending) — used by the schedule UI.
export function useTestQueue(options?: { enabled?: boolean; refetchInterval?: number | false }) {
  return useQuery({
    queryKey: queryKeys.testQueue,
    queryFn: api.getTestQueue,
    enabled: options?.enabled !== false,
    refetchInterval: options?.refetchInterval ?? 5_000,
    staleTime: 2_000,
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
    mutationFn: (args: { url: string; name?: string }) =>
      api.registerExternalTest(args.url, { name: args.name }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tests'] });
    },
  });
}

// Playbook library hooks

export function usePlaybookLibrary(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.playbookLibrary,
    queryFn: api.getPlaybookLibrary,
    staleTime: 5 * 60 * 1000, // 5 min; server caches with its own TTL
    enabled: options?.enabled !== false,
  });
}

export function useCheckPlaybookLibrary() {
  // Mutation-shaped so callers can imperatively trigger a check on
  // button-click without entering the query cache.
  return useMutation({
    mutationFn: (file: string) => api.checkPlaybookLibrary(file),
  });
}
