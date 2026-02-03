import React, { useRef, useEffect } from 'react';
import { useAIStore, type StoredChatMessage } from '../../../stores/aiStore';

// Simple markdown-like rendering for code blocks
function renderContent(content: string): React.ReactNode {
  const parts: React.ReactNode[] = [];
  let lastIndex = 0;
  const codeBlockRegex = /```(\w+)?\n([\s\S]*?)```/g;
  let match;

  while ((match = codeBlockRegex.exec(content)) !== null) {
    // Add text before code block
    if (match.index > lastIndex) {
      parts.push(
        <span key={`text-${lastIndex}`} className="whitespace-pre-wrap">
          {content.slice(lastIndex, match.index)}
        </span>
      );
    }

    // Add code block
    const language = match[1] || 'text';
    const code = match[2];
    parts.push(
      <pre
        key={`code-${match.index}`}
        className="bg-gray-100 dark:bg-gray-900 rounded p-2 my-2 overflow-x-auto text-sm"
      >
        <code className={`language-${language}`}>{code}</code>
      </pre>
    );

    lastIndex = match.index + match[0].length;
  }

  // Add remaining text
  if (lastIndex < content.length) {
    parts.push(
      <span key={`text-${lastIndex}`} className="whitespace-pre-wrap">
        {content.slice(lastIndex)}
      </span>
    );
  }

  return parts.length > 0 ? parts : <span className="whitespace-pre-wrap">{content}</span>;
}

interface ChatMessageProps {
  message: StoredChatMessage;
}

const ChatMessageComponent: React.FC<ChatMessageProps> = ({ message }) => {
  const isUser = message.role === 'user';

  return (
    <div className={`flex ${isUser ? 'justify-end' : 'justify-start'} mb-3`}>
      <div
        className={`max-w-[85%] rounded-lg px-3 py-2 ${
          isUser
            ? 'bg-blue-500 text-white'
            : 'bg-gray-100 dark:bg-gray-700 text-gray-900 dark:text-gray-100'
        }`}
      >
        <div className="text-sm">{renderContent(message.content)}</div>
        <div
          className={`text-xs mt-1 ${
            isUser ? 'text-blue-100' : 'text-gray-500 dark:text-gray-400'
          }`}
        >
          {message.timestamp.toLocaleTimeString()}
        </div>
      </div>
    </div>
  );
};

// Status indicator component
const StatusIndicator: React.FC<{ status: string }> = ({ status }) => {
  const getStatusText = () => {
    switch (status) {
      case 'pending':
        return 'Starting...';
      case 'streaming':
        return 'Generating response...';
      case 'validating':
        return 'Validating YAML...';
      case 'fixing':
        return 'Fixing validation errors...';
      default:
        return 'Processing...';
    }
  };

  return (
    <div className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400 mb-2">
      <div className="flex items-center gap-1">
        <span
          className="w-2 h-2 bg-blue-400 rounded-full animate-pulse"
        />
        <span>{getStatusText()}</span>
      </div>
    </div>
  );
};

export const ChatHistory: React.FC = () => {
  const { messages, isLoading, sessionStatus, streamingResponse } = useAIStore();
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, streamingResponse]);

  return (
    <div className="flex-1 overflow-y-auto p-3">
      {messages.length === 0 && !isLoading && (
        <div className="text-center text-gray-500 dark:text-gray-400 py-8">
          <p className="mb-4">Ask the AI to help you build a test.</p>
          <p className="text-sm mb-2">Examples:</p>
          <ul className="text-sm space-y-1">
            <li>"Create a test that checks consensus finality"</li>
            <li>"Add a task to generate 10 deposits"</li>
            <li>"Modify the timeout to 30 minutes"</li>
          </ul>
        </div>
      )}

      {messages.map((message) => (
        <ChatMessageComponent key={message.id} message={message} />
      ))}

      {/* Show streaming response */}
      {isLoading && streamingResponse && (
        <div className="flex justify-start mb-3">
          <div className="max-w-[85%] rounded-lg px-3 py-2 bg-gray-100 dark:bg-gray-700 text-gray-900 dark:text-gray-100">
            <div className="text-sm">{renderContent(streamingResponse)}</div>
          </div>
        </div>
      )}

      {/* Show status indicator */}
      {isLoading && sessionStatus && (
        <div className="flex justify-start mb-3">
          <div className="bg-gray-100 dark:bg-gray-700 rounded-lg px-3 py-2">
            <StatusIndicator status={sessionStatus} />
            {!streamingResponse && (
              <div className="flex items-center gap-1">
                <span className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
                <span className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
                <span className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
              </div>
            )}
          </div>
        </div>
      )}

      <div ref={bottomRef} />
    </div>
  );
};
