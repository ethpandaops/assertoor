import React from 'react';
import { useAIStore } from '../../../stores/aiStore';

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

  return (
    <div className="flex items-center gap-2">
      <label htmlFor="ai-model" className="text-sm text-gray-600 dark:text-gray-400">
        Model:
      </label>
      <select
        id="ai-model"
        value={selectedModel}
        onChange={(e) => setSelectedModel(e.target.value)}
        className="flex-1 text-sm bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600
                   rounded px-2 py-1 focus:outline-none focus:ring-2 focus:ring-blue-500"
      >
        {config.allowedModels.map((model) => (
          <option key={model} value={model}>
            {formatModelName(model)}
          </option>
        ))}
      </select>
    </div>
  );
};
