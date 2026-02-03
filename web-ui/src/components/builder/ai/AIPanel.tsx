import React, { useEffect } from 'react';
import { useAIStore } from '../../../stores/aiStore';
import { ModelSelector } from './ModelSelector';
import { TokenUsageDisplay } from './TokenUsageDisplay';
import { ChatHistory } from './ChatHistory';
import { MessageInput } from './MessageInput';
import { YamlPreview } from './YamlPreview';

interface AIPanelProps {
  testName: string;
  currentYaml?: string;
  onApplyYaml: (yaml: string) => void;
  onClose: () => void;
}

export const AIPanel: React.FC<AIPanelProps> = ({
  testName,
  currentYaml,
  onApplyYaml,
  onClose,
}) => {
  const {
    config,
    configLoading,
    configError,
    loadConfig,
    pendingYaml,
    pendingValidation,
    setPendingYaml,
    error,
    clearError,
    clearMessages,
  } = useAIStore();

  useEffect(() => {
    if (!config && !configLoading) {
      loadConfig();
    }
  }, [config, configLoading, loadConfig]);

  const handleApplyYaml = () => {
    if (pendingYaml) {
      onApplyYaml(pendingYaml);
      setPendingYaml(null);
    }
  };

  const handleDiscardYaml = () => {
    setPendingYaml(null);
  };

  return (
    <div className="flex flex-col h-full bg-[var(--color-bg-secondary)] overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--color-border)] bg-[var(--color-bg-tertiary)]">
        <div className="flex items-center gap-2">
          <svg
            className="w-5 h-5 text-blue-500"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"
            />
          </svg>
          <h3 className="font-medium text-[var(--color-text-primary)]">AI Assistant</h3>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={clearMessages}
            className="p-1 text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]"
            title="Clear chat"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
              />
            </svg>
          </button>
          <button
            onClick={onClose}
            className="p-1 text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>
      </div>

      {/* Loading state */}
      {configLoading && (
        <div className="flex-1 flex items-center justify-center">
          <div className="text-[var(--color-text-secondary)]">Loading AI configuration...</div>
        </div>
      )}

      {/* Error state */}
      {configError && (
        <div className="flex-1 flex items-center justify-center p-4">
          <div className="text-center">
            <p className="text-red-500 mb-2">{configError}</p>
            <button
              onClick={loadConfig}
              className="text-sm text-primary-500 hover:text-primary-600"
            >
              Retry
            </button>
          </div>
        </div>
      )}

      {/* Not enabled state */}
      {config && !config.enabled && (
        <div className="flex-1 flex items-center justify-center p-4">
          <div className="text-center text-[var(--color-text-secondary)]">
            <p className="mb-2">AI features are not enabled.</p>
            <p className="text-sm">Configure an OpenRouter API key to enable AI assistance.</p>
          </div>
        </div>
      )}

      {/* Main content */}
      {config && config.enabled && (
        <>
          {/* Config bar */}
          <div className="px-3 py-2 border-b border-[var(--color-border)] space-y-2">
            <ModelSelector />
            <TokenUsageDisplay />
          </div>

          {/* Error banner */}
          {error && (
            <div className="mx-3 mt-2 px-3 py-2 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
              <div className="flex items-start justify-between">
                <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
                <button
                  onClick={clearError}
                  className="ml-2 text-red-400 hover:text-red-600"
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M6 18L18 6M6 6l12 12"
                    />
                  </svg>
                </button>
              </div>
            </div>
          )}

          {/* Chat history */}
          <ChatHistory />

          {/* YAML preview (if pending) */}
          {pendingYaml && (
            <YamlPreview
              yamlContent={pendingYaml}
              validation={pendingValidation}
              onApply={handleApplyYaml}
              onDiscard={handleDiscardYaml}
            />
          )}

          {/* Message input */}
          <MessageInput testName={testName} currentYaml={currentYaml} />
        </>
      )}
    </div>
  );
};
