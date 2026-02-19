import React, { useState } from 'react';
import { useAIStore } from '../../../stores/aiStore';

export const ApiKeyInput: React.FC = () => {
  const { userApiKey, setUserApiKey } = useAIStore();
  const [inputValue, setInputValue] = useState('');
  const [showKey, setShowKey] = useState(false);
  const [isEditing, setIsEditing] = useState(!userApiKey);

  const handleSave = () => {
    const trimmed = inputValue.trim();
    if (trimmed) {
      setUserApiKey(trimmed);
      setInputValue('');
      setIsEditing(false);
    }
  };

  const handleRemove = () => {
    setUserApiKey(null);
    setInputValue('');
    setIsEditing(true);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSave();
    }
  };

  if (userApiKey && !isEditing) {
    const masked = userApiKey.slice(0, 8) + '...' + userApiKey.slice(-4);

    return (
      <div className="flex items-center gap-2">
        <span className="text-sm text-[var(--color-text-tertiary)]">API Key:</span>
        <code className="text-xs text-[var(--color-text-secondary)] bg-[var(--color-bg-primary)] px-1.5 py-0.5 rounded">
          {showKey ? userApiKey : masked}
        </code>
        <button
          onClick={() => setShowKey(!showKey)}
          className="text-xs text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]"
          title={showKey ? 'Hide key' : 'Show key'}
        >
          {showKey ? (
            <svg className="size-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"
              />
            </svg>
          ) : (
            <svg className="size-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
              />
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
              />
            </svg>
          )}
        </button>
        <button
          onClick={() => setIsEditing(true)}
          className="text-xs text-blue-500 hover:text-blue-600"
        >
          Change
        </button>
        <button
          onClick={handleRemove}
          className="text-xs text-red-500 hover:text-red-600"
        >
          Remove
        </button>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2">
      <span className="text-sm text-[var(--color-text-tertiary)] shrink-0">API Key:</span>
      <input
        type="password"
        value={inputValue}
        onChange={(e) => setInputValue(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="sk-or-..."
        className="flex-1 min-w-0 text-sm bg-[var(--color-bg-primary)] border border-[var(--color-border)]
                   rounded-sm px-2 py-1 focus:outline-hidden focus:ring-2 focus:ring-blue-500
                   text-[var(--color-text-primary)] placeholder-[var(--color-text-tertiary)]"
      />
      <button
        onClick={handleSave}
        disabled={!inputValue.trim()}
        className="text-xs px-2 py-1 bg-blue-600 text-white rounded-sm hover:bg-blue-700
                   disabled:opacity-50 disabled:cursor-not-allowed shrink-0"
      >
        Save
      </button>
      {userApiKey && (
        <button
          onClick={() => setIsEditing(false)}
          className="text-xs text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]"
        >
          Cancel
        </button>
      )}
    </div>
  );
};
