// API response wrapper
export interface ApiResponse<T> {
  status: string;
  data: T;
}

// Auth types
export interface AuthState {
  // True when an auth provider is configured. False = open mode (no auth
  // required); the UI should hide login controls and treat the user as
  // authorized.
  authEnabled: boolean;
  isLoggedIn: boolean;
  user: string | null;
  // Expiry of the current session's token (ms). Display-only: the token
  // itself is never cached in the app — authStore.getAuthHeader() fetches
  // a fresh one from the authenticatoor client per request.
  expiresAt: number | null;  // Local timestamp
}

// Task result enum
export type TaskResult = 'none' | 'success' | 'failure';

// Task status enum
export type TaskStatus = 'pending' | 'running' | 'complete';

// Test status
export type TestStatus = 'pending' | 'running' | 'success' | 'failure' | 'aborted' | 'skipped';

// Test definition
export interface TestDefinition {
  id: string;
  name: string;
  source: string;
  config: Record<string, unknown>;
}

// Test registry data (from registry endpoint)
export interface TestRegistryItem {
  index: number;
  test_id: string;
  name: string;
  source: string;
  base_path: string;
  error: string;
  config: string;
  run_count: number;
  last_run: string | null;
}

// Test run summary (from /api/v1/test_runs)
export interface TestRun {
  run_id: number;
  test_id: string;
  name: string;
  status: TestStatus;
  start_time: number;  // Unix timestamp
  stop_time: number;   // Unix timestamp
}

// Test definition (from /api/v1/tests)
export interface Test {
  id: string;
  source: string;
  basePath: string;
  name: string;
  description?: string;
  version?: string;
  tags?: string[];
}

// Test details (from /api/v1/test/{testId})
export interface TestDetails {
  id: string;
  source: string;
  basePath: string;
  name: string;
  description?: string;
  version?: string;
  tags?: string[];
  timeout: number;
  config: Record<string, unknown>;
  configVars: Record<string, string>;
  schedule: TestSchedule | null;
  vars?: Record<string, unknown>; // Config with global vars merged in (only for authenticated users)
}

// Test YAML source (from /api/v1/test/{testId}/yaml)
export interface TestYamlResponse {
  yaml: string;
  source: string;
}

// Latest run-level result envelope returned by
// `GET /api/v1/test/{testId}/latest_result?meta=1`. `markdown` is the
// rendered $ASSERTOOR_TEST_RESULT blob from the newest run that
// produced one. When no run has produced a result yet the envelope is
// returned with `run_id === 0` and an empty markdown string.
export interface LatestResultResponse {
  run_id: number;
  status: TestStatus | '';
  start_time: number;
  stop_time: number;
  markdown: string;
}

// Response from POST /api/v1/tests/register_external
export interface RegisterExternalTestResponse {
  test_id: string;
  name: string;
  config: Record<string, unknown>;
}

// Playbook library types ////////////////////////////////////////////////

export interface LibraryFolder {
  path: string;
  name: string;
  description?: string;
}

export interface LibraryEntry {
  file: string;
  id: string;
  name: string;
  description?: string;
  version?: string;
  tags?: string[];
  timeout?: string;
}

// Response from GET /api/v1/playbook_library
export interface LibraryIndex {
  generated: string;
  base_url: string;
  index_url: string;
  folders: LibraryFolder[];
  playbooks: LibraryEntry[];
}

export type LibraryCheckState = 'absent' | 'same' | 'different';

// Response from GET /api/v1/playbook_library/check
export interface LibraryCheckResponse {
  state: LibraryCheckState;
  remote_id: string;
  remote_name: string;
  remote_url: string;
  remote_yaml: string;
  local_test_id?: string;
  local_name?: string;
  local_yaml?: string;
}

// Test schedule configuration
export interface TestSchedule {
  startup: boolean;
  cron: string[];
  skipQueue?: boolean;
}

// Single planned execution for a test's cron schedule.
export interface TestNextRunEntry {
  expression: string;
  next: number; // Unix seconds
}

// Response from GET /api/v1/test/{testId}/next_run
export interface TestNextRunResponse {
  test_id: string;
  entries: TestNextRunEntry[];
  earliest?: TestNextRunEntry;
}

// Queue entry returned by GET /api/v1/test_queue. The first entries
// represent currently running tests; the rest are pending in
// execution order.
export interface TestQueueEntry {
  run_id: number;
  test_id: string;
  name: string;
  status: TestStatus;
}

export interface TestQueueResponse {
  queue: TestQueueEntry[];
}

// Queue option for the new (v2) schedule POST body. The legacy
// `skip_queue` boolean still works server-side; the typed `Queue`
// variant is preferred by the UI.
export type ScheduleQueueMode = 'immediate' | 'end' | 'after';

export interface ScheduleQueueOption {
  mode: ScheduleQueueMode;
  after_run_id?: number;
}

// Schedule test run request
//
// The body shape is intentionally a union of legacy + v2 fields so
// older callers keep working. New code should set `queue` and leave
// `skip_queue` undefined; when both are provided, `queue` wins.
export interface ScheduleTestRunRequest {
  test_id: string;
  config?: Record<string, unknown>;
  allow_duplicate?: boolean;

  // Deprecated: use `queue: { mode: 'immediate' }` instead.
  skip_queue?: boolean;

  queue?: ScheduleQueueOption;
}

// Test run details with tasks
export interface TestRunDetails {
  run_id: number;
  test_id: string;
  name: string;
  status: TestStatus;
  start_time: number;
  stop_time: number;
  timeout: number;
  tasks: TaskState[];
}

// Task state
export interface TaskState {
  index: number;
  parent_index: number;
  name: string;
  title: string;
  started: boolean;
  completed: boolean;
  start_time: number;
  stop_time: number;
  timeout: number;
  runtime: number;
  status: TaskStatus;
  result: TaskResult;
  result_error: string;
  result_files?: TaskResultFile[];
  progress: number;
  progress_message: string;
  run_concurrent?: boolean;
}

// Task result file
export interface TaskResultFile {
  type: string;
  index: number;
  name: string;
  size: number;
  url: string;
}

// Task details (with logs)
export interface TaskDetails extends TaskState {
  log: TaskLogEntry[];
  config_yaml: string;
  result_yaml: string;
}

// Task log entry
export interface TaskLogEntry {
  time: string;
  level: string;  // "trace" | "debug" | "info" | "warn" | "error" | "fatal" | "panic"
  msg: string;
  datalen: number;
  data: Record<string, string>;
}

// Task descriptor (for task palette)
export interface TaskDescriptor {
  name: string;
  aliases?: string[];
  description: string;
  category: string;
  configSchema: Record<string, unknown>;
  outputs: TaskOutputField[];
  examples?: string[];
}

// Task output field
export interface TaskOutputField {
  name: string;
  type: string;
  description: string;
}

// Global variables (from /api/v1/global_variables)
export interface GlobalVariablesResponse {
  names: string[];
}

// Build version info (from /api/v1/version)
export interface VersionResponse {
  version: string;
  release: string;
}

// Client data
export interface ClientData {
  index: number;
  name: string;
  cl_version: string;
  cl_type: number;
  cl_head_slot: number;
  cl_head_root: string;
  cl_status: string;
  cl_refresh: string;
  cl_error: string;
  cl_ready: boolean;
  el_version: string;
  el_type: number;
  el_head_number: number;
  el_head_hash: string;
  el_status: string;
  el_refresh: string;
  el_error: string;
  el_ready: boolean;
}

// Clients page data
export interface ClientsPage {
  clients: ClientData[];
  client_count: number;
}

// Sidebar data
export interface SidebarData {
  client_count: number;
  cl_ready_count: number;
  cl_head_slot: number;
  cl_head_root: string;
  el_ready_count: number;
  el_head_number: number;
  el_head_hash: string;
  tests: SidebarTest[];
  all_tests_active: boolean;
  registry_active: boolean;
  can_register_tests: boolean;
  version: string;
}

// Sidebar test entry
export interface SidebarTest {
  id: string;
  name: string;
  active: boolean;
}

// SSE event types
export type SSEEventType =
  | 'test.started'
  | 'test.completed'
  | 'test.failed'
  | 'task.created'
  | 'task.started'
  | 'task.progress'
  | 'task.completed'
  | 'task.failed'
  | 'task.log'
  | 'client.head_update'
  | 'client.status_update';

// SSE event data
export interface SSEEvent {
  type: SSEEventType;
  testRunId: number;
  taskIndex?: number;
  data: Record<string, unknown>;
}

// Client SSE event data
export interface ClientHeadUpdateEvent {
  clientIndex: number;
  clientName: string;
  clHeadSlot: number;
  clHeadRoot: string;
  elHeadNumber: number;
  elHeadHash: string;
}

export interface ClientStatusUpdateEvent {
  clientIndex: number;
  clientName: string;
  clStatus: string;
  clReady: boolean;
  elStatus: string;
  elReady: boolean;
}
