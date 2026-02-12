import React from 'react';
import { useAIStore } from '../../../stores/aiStore';

const POPULAR_MODELS = [
  'anthropic/claude-sonnet-4',
  'openai/gpt-4o',
  'google/gemini-2.0-flash',
  'google/gemini-2.5-pro-preview',
  'meta-llama/llama-3.3-70b-instruct',
  'deepseek/deepseek-chat-v3-0324',
];

// Format model name for display
function formatModelName(model: string): string {
  const parts = model.split('/');
  if (parts.length !== 2) return model;

  const [provider, name] = parts;
  const formattedName = name
    .split('-')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');

  return `${formattedName} (${provider})`;
}

export const ModelSelector: React.FC = () => {
  const { config, selectedModel, setSelectedModel } = useAIStore();

  if (!config) return null;

  const hasAllowedModels = config.allowedModels.length > 0;

  if (hasAllowedModels) {
    return (
      <div className="flex items-center gap-2">
        <label htmlFor="ai-model" className="text-sm text-[var(--color-text-tertiary)]">
          Model:
        </label>
        <select
          id="ai-model"
          value={selectedModel}
          onChange={(e) => setSelectedModel(e.target.value)}
          className="flex-1 text-sm bg-[var(--color-bg-primary)] border border-[var(--color-border)]
                     rounded-sm px-2 py-1 focus:outline-hidden focus:ring-2 focus:ring-blue-500
                     text-[var(--color-text-primary)]"
        >
          {config.allowedModels.map((model) => (
            <option key={model} value={model}>
              {formatModelName(model)}
            </option>
          ))}
        </select>
      </div>
    );
  }

  // Free-form input with datalist suggestions for user-key mode
  return (
    <div className="flex items-center gap-2">
      <label htmlFor="ai-model" className="text-sm text-[var(--color-text-tertiary)]">
        Model:
      </label>
      <input
        id="ai-model"
        list="ai-model-suggestions"
        value={selectedModel}
        onChange={(e) => setSelectedModel(e.target.value)}
        placeholder="e.g. anthropic/claude-sonnet-4"
        className="flex-1 min-w-0 text-sm bg-[var(--color-bg-primary)] border border-[var(--color-border)]
                   rounded-sm px-2 py-1 focus:outline-hidden focus:ring-2 focus:ring-blue-500
                   text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)]"
      />
      <datalist id="ai-model-suggestions">
        {POPULAR_MODELS.map((model) => (
          <option key={model} value={model}>
            {formatModelName(model)}
          </option>
        ))}
      </datalist>
    </div>
  );
};
