import React, { useState } from 'react';
import { useAIStore } from '../../../stores/aiStore';

interface MessageInputProps {
  testName: string;
  currentYaml?: string;
}

export const MessageInput: React.FC<MessageInputProps> = ({ testName, currentYaml }) => {
  const [input, setInput] = useState('');
  const { sendMessage, isLoading } = useAIStore();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isLoading) return;

    const message = input;
    setInput('');
    await sendMessage(message, testName, currentYaml);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="border-t border-gray-200 dark:border-gray-700 p-3">
      <div className="flex gap-2">
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Describe what you want to build or change..."
          disabled={isLoading}
          rows={2}
          className="flex-1 resize-none bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600
                     rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500
                     disabled:bg-gray-100 dark:disabled:bg-gray-900 disabled:cursor-not-allowed"
        />
        <button
          type="submit"
          disabled={isLoading || !input.trim()}
          className="px-4 py-2 bg-blue-500 text-white rounded-lg text-sm font-medium
                     hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2
                     disabled:bg-gray-300 dark:disabled:bg-gray-600 disabled:cursor-not-allowed
                     transition-colors"
        >
          {isLoading ? 'Thinking...' : 'Send'}
        </button>
      </div>
      <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
        Press Enter to send, Shift+Enter for new line
      </p>
    </form>
  );
};
