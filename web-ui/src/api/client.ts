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
  GlobalVariablesResponse,
  ScheduleTestRunRequest,
  VersionResponse,
  LibraryIndex,
  LibraryCheckResponse,
  RegisterExternalTestResponse,
  LatestResultResponse,
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

// Latest run-level result markdown for a test (envelope form). Walks
// the newest runs server-side and returns the first one that produced
// a $ASSERTOOR_TEST_RESULT blob. `run_id === 0` and empty `markdown`
// indicate "no result available across recent runs" (the dashboard
// tile renders an empty state in that case).
export async function getLatestTestResult(testId: string): Promise<LatestResultResponse> {
  return fetchApi<LatestResultResponse>(
    `/test/${encodeURIComponent(testId)}/latest_result?meta=1`,
  ).then((data) => {
    // Server returns `data: null` on 200 when no result exists yet —
    // normalise to an empty envelope so callers don't have to.
    return (
      (data as LatestResultResponse) ?? {
        run_id: 0,
        status: '',
        start_time: 0,
        stop_time: 0,
        markdown: '',
      }
    );
  });
}

// Run-level Result markdown. Returns null when the run has not produced
// a $ASSERTOOR_TEST_RESULT blob (server replies 204 No Content).
export async function getTestRunResult(runId: number): Promise<string | null> {
  const response = await fetch(`${API_BASE}/test_run/${runId}/result`);
  if (response.status === 204) return null;

  if (!response.ok) {
    throw new Error(`API error: ${response.status} ${response.statusText}`);
  }

  return response.text();
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

// Global variable names (auth required)
export async function getGlobalVariables(): Promise<GlobalVariablesResponse> {
  return fetchApiWithAuth<GlobalVariablesResponse>('/global_variables');
}

// Build version info
export async function getVersion(): Promise<VersionResponse> {
  return fetchApi<VersionResponse>('/version');
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

export async function registerExternalTest(
  url: string,
  options?: { name?: string },
): Promise<RegisterExternalTestResponse> {
  return fetchApiWithAuth<RegisterExternalTestResponse>('/tests/register_external', {
    method: 'POST',
    body: JSON.stringify({ file: url, name: options?.name }),
  });
}

// Dashboard config
//
// The server stores an opaque JSON blob (the schema is owned by the
// client). GET is open; PUT requires authentication. A 204 from GET
// means "no config yet" and the UI falls back to its built-in
// default dashboard.

export async function getDashboardConfig(): Promise<unknown | null> {
  const response = await fetch(`${API_BASE}/dashboard_config`);
  if (response.status === 204) return null;
  if (!response.ok) {
    throw new Error(`API error: ${response.status} ${response.statusText}`);
  }
  return response.json();
}

export async function putDashboardConfig(cfg: unknown): Promise<void> {
  const authHeader = authStore.getAuthHeader();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  if (authHeader) headers['Authorization'] = authHeader;

  const response = await fetch(`${API_BASE}/dashboard_config`, {
    method: 'PUT',
    headers,
    body: JSON.stringify(cfg),
  });

  if (response.status === 401) {
    throw new Error('Unauthorized: log in to save dashboard changes');
  }

  if (!response.ok) {
    throw new Error(`API error: ${response.status} ${response.statusText}`);
  }
}

// Network status (used by the dashboard's `network_status` tile).
export interface NetworkStatusResponse {
  chain_id: number;
  network_name: string;
  genesis_time: number;
  slot_duration_ms: number;
  slots_per_epoch: number;
  current_slot: number;
  current_epoch: number;
  head_slot: number;
  head_root: string;
  finalized_epoch: number;
  finalized_root: string;
  justified_epoch: number;
  justified_root: string;
  client_count: number;
  cl_ready_count: number;
  el_ready_count: number;
  el_head_number: number;
  el_head_hash: string;
  tests_running: number;
  tests_queued: number;
}

export async function getNetworkStatus(): Promise<NetworkStatusResponse> {
  return fetchApi<NetworkStatusResponse>('/network_status');
}

// Playbook library

export async function getPlaybookLibrary(): Promise<LibraryIndex> {
  return fetchApi<LibraryIndex>('/playbook_library');
}

export async function checkPlaybookLibrary(file: string): Promise<LibraryCheckResponse> {
  return fetchApiWithAuth<LibraryCheckResponse>(
    `/playbook_library/check?file=${encodeURIComponent(file)}`,
  );
}
