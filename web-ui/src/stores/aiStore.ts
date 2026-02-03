import { create } from 'zustand';
import {
  fetchAIConfig,
  fetchAIUsage,
  startAIChat,
  pollAISession,
  type AIConfig,
  type AIUsageStats,
  type ChatMessage,
  type ValidationResult,
  type SessionStatus,
} from '../api/ai';

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

  // Actions
  loadConfig: () => Promise<void>;
  loadUsage: () => Promise<void>;
  setSelectedModel: (model: string) => void;
  sendMessage: (content: string, testName: string, currentYaml?: string) => Promise<void>;
  setPendingYaml: (yaml: string | null) => void;
  clearMessages: () => void;
  clearError: () => void;
}

// Generate unique message ID
function generateMessageId(): string {
  return `msg_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`;
}

export const useAIStore = create<AIState>((set, get) => ({
  // Initial state
  config: null,
  configLoading: false,
  configError: null,
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
        config: { enabled: false, defaultModel: '', allowedModels: [] },
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

  // Send chat message
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

      // Add current user message
      apiMessages.push({
        role: 'user',
        content,
      });

      // Start chat session
      const sessionId = await startAIChat({
        model: selectedModel,
        messages: apiMessages,
        testName,
        currentYaml,
      });

      // Poll for updates
      const finalSession = await pollAISession(sessionId, (session) => {
        set({
          sessionStatus: session.status,
          streamingResponse: session.response,
        });
      });

      // Handle error
      if (finalSession.status === 'error') {
        set({
          isLoading: false,
          error: finalSession.error || 'Unknown error',
          sessionStatus: null,
          streamingResponse: '',
        });
        return;
      }

      // Add assistant response to chat
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

      // Refresh usage stats after successful request
      get().loadUsage();
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
