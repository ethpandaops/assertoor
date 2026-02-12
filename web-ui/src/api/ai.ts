import { authStore } from '../stores/authStore';

const API_BASE = '/api/v1';

// Types
export interface AIConfig {
  enabled: boolean;
  defaultModel: string;
  allowedModels: string[];
  serverKeyConfigured: boolean;
}

export interface AIUsageStats {
  totalPromptTokens: number;
  totalCompletionTokens: number;
  totalTokens: number;
  totalRequests: number;
}

export interface AIUsageResponse {
  lastDay: AIUsageStats;
  lastMonth: AIUsageStats;
}

export interface ChatMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
}

export interface AIChatRequest {
  model: string;
  messages: ChatMessage[];
  testName: string;
  currentYaml?: string;
}

export interface ValidationIssue {
  type: 'error' | 'warning';
  path: string;
  message: string;
}

export interface ValidationResult {
  valid: boolean;
  issues?: ValidationIssue[];
}

// Session status enum
export type SessionStatus = 'pending' | 'streaming' | 'validating' | 'fixing' | 'complete' | 'error';

// Session state from the backend
export interface AISession {
  id: string;
  status: SessionStatus;
  createdAt: string;
  updatedAt: string;
  response: string;
  generatedYaml?: string;
  validation?: ValidationResult;
  usage: {
    promptTokens: number;
    completionTokens: number;
    totalTokens: number;
  };
  error?: string;
  fixAttempts: number;
}

// Response for starting a chat session
export interface AIChatStartResponse {
  sessionId: string;
}

interface ApiResponse<T> {
  status: string;
  data: T;
}

function getAuthHeaders(): Record<string, string> {
  const authHeader = authStore.getAuthHeader();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  if (authHeader) {
    headers['Authorization'] = authHeader;
  }

  return headers;
}

async function fetchAIApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const headers = getAuthHeaders();

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      ...headers,
      ...(options?.headers as Record<string, string>),
    },
  });

  if (response.status === 401) {
    throw new Error('Unauthorized: Please log in to use AI features');
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

// Get AI configuration
export async function fetchAIConfig(): Promise<AIConfig> {
  return fetchAIApi<AIConfig>('/ai/config');
}

// Get AI usage statistics
export async function fetchAIUsage(): Promise<AIUsageResponse> {
  return fetchAIApi<AIUsageResponse>('/ai/usage');
}

// Start a chat session (returns session ID immediately)
export async function startAIChat(request: AIChatRequest): Promise<string> {
  const response = await fetchAIApi<AIChatStartResponse>('/ai/chat', {
    method: 'POST',
    body: JSON.stringify(request),
  });

  return response.sessionId;
}

// Poll for session status
export async function getAISession(sessionId: string): Promise<AISession> {
  return fetchAIApi<AISession>(`/ai/chat/${sessionId}`);
}

// Stream session updates using SSE
export function streamAISession(
  sessionId: string,
  onUpdate: (session: AISession) => void,
  onError: (error: Error) => void,
  onComplete: () => void
): () => void {
  const authHeader = authStore.getAuthHeader();
  let url = `${API_BASE}/ai/chat/${sessionId}/stream`;

  // Add auth token as query param for SSE (headers don't work well with EventSource)
  if (authHeader) {
    const token = authHeader.replace('Bearer ', '');
    url += `?token=${encodeURIComponent(token)}`;
  }

  const eventSource = new EventSource(url);

  eventSource.addEventListener('update', (event) => {
    try {
      const session = JSON.parse(event.data) as AISession;
      onUpdate(session);

      // Close connection when session is complete or errored
      if (session.status === 'complete' || session.status === 'error') {
        eventSource.close();
        onComplete();
      }
    } catch (err) {
      onError(err instanceof Error ? err : new Error('Failed to parse session update'));
    }
  });

  eventSource.onerror = () => {
    eventSource.close();
    onError(new Error('Connection to session stream failed'));
  };

  // Return cleanup function
  return () => {
    eventSource.close();
  };
}

// Helper to poll for session completion (fallback if SSE doesn't work)
export async function pollAISession(
  sessionId: string,
  onUpdate: (session: AISession) => void,
  intervalMs: number = 500,
  maxAttempts: number = 600 // 5 minutes with 500ms interval
): Promise<AISession> {
  let attempts = 0;

  while (attempts < maxAttempts) {
    const session = await getAISession(sessionId);
    onUpdate(session);

    if (session.status === 'complete' || session.status === 'error') {
      return session;
    }

    await new Promise((resolve) => setTimeout(resolve, intervalMs));
    attempts++;
  }

  throw new Error('Session polling timed out');
}

// Get AI system prompt (for client-side mode)
export async function fetchSystemPrompt(): Promise<string> {
  const data = await fetchAIApi<{ prompt: string }>('/ai/system_prompt');
  return data.prompt;
}

// Validate YAML via backend (for client-side mode)
export async function validateYaml(yaml: string): Promise<ValidationResult> {
  return fetchAIApi<ValidationResult>('/ai/validate', {
    method: 'POST',
    body: JSON.stringify({ yaml }),
  });
}

// Extract YAML code blocks from AI response
export function extractYamlFromResponse(response: string): string {
  const re = /```ya?ml\s*\n([\s\S]*?)```/;
  const match = re.exec(response);
  return match ? match[1] : '';
}

// Build a fix prompt for validation issues (mirrors Go logic)
export function buildFixPrompt(
  brokenYaml: string,
  validation: ValidationResult,
  warningsOnly: boolean
): string {
  let prompt: string;

  if (warningsOnly) {
    prompt =
      'The YAML you generated has validation warnings. Please review and fix the following issues if appropriate:\n\n';
    for (const issue of validation.issues ?? []) {
      if (issue.type === 'warning') {
        prompt += `- ${issue.path}: ${issue.message}\n`;
      }
    }
  } else {
    prompt = 'The YAML you generated has validation errors. Please fix the following issues:\n\n';
    for (const issue of validation.issues ?? []) {
      if (issue.type === 'error') {
        prompt += `- ${issue.path}: ${issue.message}\n`;
      }
    }
  }

  prompt += '\nHere is the YAML:\n```yaml\n' + brokenYaml + '\n```\n\n';

  if (warningsOnly) {
    prompt +=
      'Please provide an improved version of the YAML that addresses the warnings where appropriate. ';
  } else {
    prompt += 'Please provide a corrected version of the YAML that fixes all the errors. ';
  }

  prompt += 'Only output the corrected YAML in a code block, no explanations needed.';
  return prompt;
}

// Stream chat directly to OpenRouter from the browser (client-side mode)
export async function clientSideStreamChat(
  apiKey: string,
  model: string,
  messages: ChatMessage[],
  maxTokens: number,
  onChunk: (chunk: string) => void
): Promise<string> {
  const response = await fetch('https://openrouter.ai/api/v1/chat/completions', {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
      'HTTP-Referer': window.location.origin,
    },
    body: JSON.stringify({
      model,
      messages,
      max_tokens: maxTokens,
      stream: true,
    }),
  });

  if (!response.ok) {
    const errorBody = await response.text();
    throw new Error(`OpenRouter API error: ${response.status} - ${errorBody}`);
  }

  const reader = response.body?.getReader();
  if (!reader) {
    throw new Error('No response body');
  }

  const decoder = new TextDecoder();
  let fullResponse = '';
  let buffer = '';

  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop() ?? '';

    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed.startsWith('data: ')) continue;

      const data = trimmed.slice(6);
      if (data === '[DONE]') continue;

      try {
        const parsed = JSON.parse(data);
        const content = parsed.choices?.[0]?.delta?.content;
        if (content) {
          fullResponse += content;
          onChunk(content);
        }
      } catch {
        // skip malformed SSE chunks
      }
    }
  }

  return fullResponse;
}

// Non-streaming chat call to OpenRouter (for fix attempts)
export async function clientSideChat(
  apiKey: string,
  model: string,
  messages: ChatMessage[],
  maxTokens: number
): Promise<string> {
  const response = await fetch('https://openrouter.ai/api/v1/chat/completions', {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
      'HTTP-Referer': window.location.origin,
    },
    body: JSON.stringify({
      model,
      messages,
      max_tokens: maxTokens,
    }),
  });

  if (!response.ok) {
    const errorBody = await response.text();
    throw new Error(`OpenRouter API error: ${response.status} - ${errorBody}`);
  }

  const result = await response.json();
  return result.choices?.[0]?.message?.content ?? '';
}
