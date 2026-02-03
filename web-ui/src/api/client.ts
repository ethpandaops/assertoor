import type {
  ApiResponse,
  TestRun,
  Test,
  TestDetails,
  TestYamlResponse,
  TestRunDetails,
  TaskDetails,
  TaskDescriptor,
  ClientsPage,
  ScheduleTestRunRequest,
} from '../types/api';
import { authStore } from '../stores/authStore';

const API_BASE = '/api/v1';

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!response.ok) {
    throw new Error(`API error: ${response.status} ${response.statusText}`);
  }

  const result: ApiResponse<T> = await response.json();

  if (result.status !== 'OK') {
    throw new Error(result.status);
  }

  return result.data;
}

// Fetch API with Authorization header for protected endpoints
async function fetchApiWithAuth<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const authHeader = authStore.getAuthHeader();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options?.headers as Record<string, string>),
  };

  if (authHeader) {
    headers['Authorization'] = authHeader;
  }

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  if (response.status === 401) {
    throw new Error('Unauthorized: Please log in to perform this action');
  }

  if (!response.ok) {
    throw new Error(`API error: ${response.status} ${response.statusText}`);
  }

  const result: ApiResponse<T> = await response.json();

  if (result.status !== 'OK') {
    throw new Error(result.status);
  }

  return result.data;
}

// Test runs list
export async function getTestRuns(testId?: string): Promise<TestRun[]> {
  const params = testId ? `?test_id=${encodeURIComponent(testId)}` : '';
  return fetchApi<TestRun[]>(`/test_runs${params}`);
}

// Tests (registry) list
export async function getTests(): Promise<Test[]> {
  return fetchApi<Test[]>('/tests');
}

// Test details (uses auth to get vars with global variables merged in)
export async function getTestDetails(testId: string): Promise<TestDetails> {
  return fetchApiWithAuth<TestDetails>(`/test/${encodeURIComponent(testId)}`);
}

// Test YAML source (for loading existing tests in builder)
export async function getTestYaml(testId: string): Promise<TestYamlResponse> {
  return fetchApiWithAuth<TestYamlResponse>(`/test/${encodeURIComponent(testId)}/yaml`);
}

// Clients list
export async function getClients(): Promise<ClientsPage> {
  return fetchApi<ClientsPage>('/clients');
}

// Test run details
export async function getTestRunDetails(runId: number): Promise<TestRunDetails> {
  return fetchApiWithAuth<TestRunDetails>(`/test_run/${runId}/details`);
}

// Task details
export async function getTaskDetails(runId: number, taskIndex: number): Promise<TaskDetails> {
  return fetchApiWithAuth<TaskDetails>(`/test_run/${runId}/task/${taskIndex}/details`);
}

// Task descriptors
export async function getTaskDescriptors(): Promise<TaskDescriptor[]> {
  return fetchApi<TaskDescriptor[]>('/task_descriptors');
}

export async function getTaskDescriptor(name: string): Promise<TaskDescriptor> {
  return fetchApi<TaskDescriptor>(`/task_descriptor/${encodeURIComponent(name)}`);
}

// Admin operations (require authentication)
export async function scheduleTestRun(request: ScheduleTestRunRequest): Promise<{ run_id: number }> {
  return fetchApiWithAuth<{ run_id: number }>('/test_runs/schedule', {
    method: 'POST',
    body: JSON.stringify(request),
  });
}

export async function cancelTestRun(runId: number): Promise<void> {
  await fetchApiWithAuth<void>(`/test_run/${runId}/cancel`, {
    method: 'POST',
    body: JSON.stringify({}),
  });
}

export async function deleteTestRuns(runIds: number[]): Promise<void> {
  await fetchApiWithAuth<void>('/test_runs/delete', {
    method: 'POST',
    body: JSON.stringify({ test_runs: runIds }),
  });
}

export async function deleteTest(testId: string): Promise<void> {
  await fetchApiWithAuth<void>('/tests/delete', {
    method: 'POST',
    body: JSON.stringify({ tests: [testId] }),
  });
}

export async function registerTest(yaml: string): Promise<void> {
  // Send raw YAML with application/yaml content type
  // The backend expects either YAML body or JSON with test fields directly
  const authHeader = authStore.getAuthHeader();
  const headers: Record<string, string> = {
    'Content-Type': 'application/yaml',
  };

  if (authHeader) {
    headers['Authorization'] = authHeader;
  }

  const response = await fetch(`${API_BASE}/tests/register`, {
    method: 'POST',
    headers,
    body: yaml,
  });

  if (response.status === 401) {
    throw new Error('Unauthorized: Please log in to perform this action');
  }

  if (!response.ok) {
    throw new Error(`API error: ${response.status} ${response.statusText}`);
  }

  const result = await response.json();

  if (result.status !== 'OK') {
    throw new Error(result.status);
  }
}

export async function registerExternalTest(url: string): Promise<void> {
  await fetchApiWithAuth<void>('/tests/register_external', {
    method: 'POST',
    body: JSON.stringify({ file: url }),
  });
}
