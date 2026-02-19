import { create } from 'zustand';
import {
  fetchAIConfig,
  fetchAIUsage,
  fetchSystemPrompt,
  startAIChat,
  pollAISession,
  validateYaml,
  extractYamlFromResponse,
  buildFixPrompt,
  clientSideStreamChat,
  clientSideChat,
  type AIConfig,
  type AIUsageStats,
  type ChatMessage,
  type ValidationResult,
  type SessionStatus,
} from '../api/ai';

const LOCAL_STORAGE_KEY = 'assertoor-ai-apikey';
const MAX_FIX_ATTEMPTS = 3;
const CLIENT_SIDE_MAX_TOKENS = 16384;

// Chat message with metadata
export interface StoredChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  generatedYaml?: string;
  timestamp: Date;
}

// AI store state
export interface AIState {
  // Config
  config: AIConfig | null;
  configLoading: boolean;
  configError: string | null;

  // User API key (client-side mode)
  userApiKey: string | null;
  systemPrompt: string | null;

  // Chat
  messages: StoredChatMessage[];
  selectedModel: string;
  isLoading: boolean;
  error: string | null;

  // Session state
  sessionStatus: SessionStatus | null;
  streamingResponse: string;

  // Current generated YAML (pending application)
  pendingYaml: string | null;
  pendingValidation: ValidationResult | null;

  // Usage stats
  usageLastDay: AIUsageStats | null;
  usageLastMonth: AIUsageStats | null;

  // Computed helpers
  isAvailable: () => boolean;
  isClientSide: () => boolean;

  // Actions
  loadConfig: () => Promise<void>;
  loadUsage: () => Promise<void>;
  setSelectedModel: (model: string) => void;
  setUserApiKey: (key: string | null) => void;
  loadSystemPrompt: () => Promise<string>;
  sendMessage: (content: string, testName: string, currentYaml?: string) => Promise<void>;
  setPendingYaml: (yaml: string | null) => void;
  clearMessages: () => void;
  clearError: () => void;
}

// Generate unique message ID
function generateMessageId(): string {
  return `msg_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`;
}

function hasWarnings(validation: ValidationResult): boolean {
  return (validation.issues ?? []).some((i) => i.type === 'warning');
}

export const useAIStore = create<AIState>((set, get) => ({
  // Initial state
  config: null,
  configLoading: false,
  configError: null,
  userApiKey: localStorage.getItem(LOCAL_STORAGE_KEY) || null,
  systemPrompt: null,
  messages: [],
  selectedModel: '',
  isLoading: false,
  error: null,
  sessionStatus: null,
  streamingResponse: '',
  pendingYaml: null,
  pendingValidation: null,
  usageLastDay: null,
  usageLastMonth: null,

  // Computed: AI is available if server has key OR user has key
  isAvailable: () => {
    const { config, userApiKey } = get();
    return !!config?.enabled && (!!config.serverKeyConfigured || !!userApiKey);
  },

  // Computed: using client-side mode when user has key and server doesn't,
  // or user explicitly provided a key
  isClientSide: () => {
    const { config, userApiKey } = get();
    if (!userApiKey) return false;
    return !config?.serverKeyConfigured || !!userApiKey;
  },

  // Load AI configuration
  loadConfig: async () => {
    set({ configLoading: true, configError: null });

    try {
      const config = await fetchAIConfig();
      set({
        config,
        configLoading: false,
        selectedModel: config.defaultModel || config.allowedModels[0] || '',
      });
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load AI config';
      set({
        configLoading: false,
        configError: errorMessage,
        config: {
          enabled: false,
          defaultModel: '',
          allowedModels: [],
          serverKeyConfigured: false,
        },
      });
    }
  },

  // Load usage statistics
  loadUsage: async () => {
    try {
      const usage = await fetchAIUsage();
      set({
        usageLastDay: usage.lastDay,
        usageLastMonth: usage.lastMonth,
      });
    } catch (err) {
      console.error('Failed to load AI usage:', err);
    }
  },

  // Set selected model
  setSelectedModel: (model: string) => {
    set({ selectedModel: model });
  },

  // Set user API key (persists to localStorage)
  setUserApiKey: (key: string | null) => {
    if (key) {
      localStorage.setItem(LOCAL_STORAGE_KEY, key);
    } else {
      localStorage.removeItem(LOCAL_STORAGE_KEY);
    }

    set({ userApiKey: key });
  },

  // Fetch and cache the system prompt
  loadSystemPrompt: async () => {
    const cached = get().systemPrompt;
    if (cached) return cached;

    const prompt = await fetchSystemPrompt();
    set({ systemPrompt: prompt });

    return prompt;
  },

  // Send chat message (dual-mode: server-side or client-side)
  sendMessage: async (content: string, testName: string, currentYaml?: string) => {
    const { messages, selectedModel, config } = get();

    if (!config?.enabled) {
      set({ error: 'AI is not enabled' });
      return;
    }

    // Add user message to chat
    const userMessage: StoredChatMessage = {
      id: generateMessageId(),
      role: 'user',
      content,
      timestamp: new Date(),
    };

    set({
      messages: [...messages, userMessage],
      isLoading: true,
      error: null,
      sessionStatus: 'pending',
      streamingResponse: '',
    });

    try {
      // Build message history for API
      const apiMessages: ChatMessage[] = messages.map((msg) => ({
        role: msg.role,
        content: msg.content,
      }));

      apiMessages.push({ role: 'user', content });

      const isClientSide = get().isClientSide();

      if (isClientSide) {
        await sendClientSide(get, set, apiMessages, selectedModel, testName, currentYaml);
      } else {
        await sendServerSide(get, set, apiMessages, selectedModel, testName, currentYaml);
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to send message';
      set({
        isLoading: false,
        error: errorMessage,
        sessionStatus: null,
        streamingResponse: '',
      });
    }
  },

  // Set pending YAML
  setPendingYaml: (yaml: string | null) => {
    set({ pendingYaml: yaml, pendingValidation: yaml ? get().pendingValidation : null });
  },

  // Clear all messages
  clearMessages: () => {
    set({
      messages: [],
      pendingYaml: null,
      pendingValidation: null,
      error: null,
      sessionStatus: null,
      streamingResponse: '',
    });
  },

  // Clear error
  clearError: () => {
    set({ error: null });
  },
}));

type GetState = () => AIState;
type SetState = (
  partial: AIState | Partial<AIState> | ((state: AIState) => AIState | Partial<AIState>),
) => void;

// Server-side flow (existing behavior)
async function sendServerSide(
  get: GetState,
  set: SetState,
  apiMessages: ChatMessage[],
  model: string,
  testName: string,
  currentYaml?: string,
) {
  const sessionId = await startAIChat({
    model,
    messages: apiMessages,
    testName,
    currentYaml,
  });

  const finalSession = await pollAISession(sessionId, (session) => {
    set({
      sessionStatus: session.status,
      streamingResponse: session.response,
    });
  });

  if (finalSession.status === 'error') {
    set({
      isLoading: false,
      error: finalSession.error || 'Unknown error',
      sessionStatus: null,
      streamingResponse: '',
    });

    return;
  }

  const assistantMessage: StoredChatMessage = {
    id: generateMessageId(),
    role: 'assistant',
    content: finalSession.response,
    generatedYaml: finalSession.generatedYaml,
    timestamp: new Date(),
  };

  set((state) => ({
    messages: [...state.messages, assistantMessage],
    isLoading: false,
    sessionStatus: null,
    streamingResponse: '',
    pendingYaml: finalSession.generatedYaml || null,
    pendingValidation: finalSession.validation || null,
  }));

  get().loadUsage();
}

// Client-side flow (browser calls OpenRouter directly)
async function sendClientSide(
  get: GetState,
  set: SetState,
  apiMessages: ChatMessage[],
  model: string,
  _testName: string,
  currentYaml?: string,
) {
  const userApiKey = get().userApiKey;
  if (!userApiKey) {
    throw new Error('No API key configured');
  }

  // Fetch system prompt (cached after first call)
  const systemPrompt = await get().loadSystemPrompt();

  // Build full messages array with system prompt + context
  const fullMessages: ChatMessage[] = [{ role: 'system', content: systemPrompt }];

  if (currentYaml) {
    fullMessages.push({
      role: 'system',
      content:
        'The user is currently working on the following test configuration:\n\n```yaml\n' +
        currentYaml +
        '\n```',
    });
  }

  fullMessages.push(...apiMessages);

  // Stream response from OpenRouter
  set({ sessionStatus: 'streaming' });

  const fullResponse = await clientSideStreamChat(
    userApiKey,
    model,
    fullMessages,
    CLIENT_SIDE_MAX_TOKENS,
    (chunk) => {
      set((state) => ({ streamingResponse: state.streamingResponse + chunk }));
    },
  );

  // Extract and validate YAML
  set({ sessionStatus: 'validating' });

  let generatedYaml = extractYamlFromResponse(fullResponse);
  let validation: ValidationResult | null = null;

  if (generatedYaml) {
    validation = await validateYaml(generatedYaml);

    // Attempt fixes for errors (up to MAX_FIX_ATTEMPTS)
    let fixAttempts = 0;

    while (!validation.valid && fixAttempts < MAX_FIX_ATTEMPTS) {
      set({ sessionStatus: 'fixing' });
      fixAttempts++;

      const fixPrompt = buildFixPrompt(generatedYaml, validation, false);
      const fixMessages: ChatMessage[] = [
        ...fullMessages,
        { role: 'assistant', content: fullResponse },
        { role: 'user', content: fixPrompt },
      ];

      const fixResponse = await clientSideChat(
        userApiKey,
        model,
        fixMessages,
        CLIENT_SIDE_MAX_TOKENS,
      );

      const fixedYaml = extractYamlFromResponse(fixResponse);
      if (!fixedYaml) break;

      generatedYaml = fixedYaml;
      validation = await validateYaml(fixedYaml);
    }

    // One attempt to fix warnings if errors are resolved
    if (validation.valid && hasWarnings(validation)) {
      set({ sessionStatus: 'fixing' });

      const fixPrompt = buildFixPrompt(generatedYaml, validation, true);
      const fixMessages: ChatMessage[] = [
        ...fullMessages,
        { role: 'assistant', content: fullResponse },
        { role: 'user', content: fixPrompt },
      ];

      const fixResponse = await clientSideChat(
        userApiKey,
        model,
        fixMessages,
        CLIENT_SIDE_MAX_TOKENS,
      );

      const fixedYaml = extractYamlFromResponse(fixResponse);

      if (fixedYaml) {
        const fixValidation = await validateYaml(fixedYaml);
        generatedYaml = fixedYaml;
        validation = fixValidation;
      }
    }
  }

  const assistantMessage: StoredChatMessage = {
    id: generateMessageId(),
    role: 'assistant',
    content: fullResponse,
    generatedYaml: generatedYaml || undefined,
    timestamp: new Date(),
  };

  set((state) => ({
    messages: [...state.messages, assistantMessage],
    isLoading: false,
    sessionStatus: null,
    streamingResponse: '',
    pendingYaml: generatedYaml || null,
    pendingValidation: validation,
  }));
}
