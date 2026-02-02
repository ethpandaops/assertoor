import { useState, useCallback, useMemo, useRef, useEffect } from 'react';
import { useBuilderStore } from '../../../stores/builderStore';
import { useTaskDescriptors } from '../../../hooks/useApi';
import { findPrecedingTasks } from '../../../utils/builder/taskUtils';
import type { TaskDescriptor, TaskOutputField } from '../../../types/api';

interface ExpressionInputProps {
  taskId: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}

interface TaskOutput {
  taskId: string;
  taskType: string;
  title?: string;
  outputs: TaskOutputField[];
}

interface Suggestion {
  label: string;
  value: string;
  description?: string;
  category: 'global' | 'task';
  taskInfo?: { taskId: string; title?: string; taskType: string };
}

function ExpressionInput({
  taskId,
  value,
  onChange,
  placeholder = 'JQ expression',
}: ExpressionInputProps) {
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const tasks = useBuilderStore((state) => state.testConfig.tasks);
  const testVars = useBuilderStore((state) => state.testConfig.testVars);
  const { data: descriptors } = useTaskDescriptors();

  // Build descriptor map
  const descriptorMap = useMemo(() => {
    const map = new Map<string, TaskDescriptor>();
    if (descriptors) {
      for (const d of descriptors) {
        map.set(d.name, d);
      }
    }
    return map;
  }, [descriptors]);

  // Get available variables
  const variableContext = useMemo(() => {
    const globalVars = Object.keys(testVars || {});
    const precedingTasks = findPrecedingTasks(tasks, taskId);
    const taskOutputs: TaskOutput[] = precedingTasks
      .filter((t) => t.taskId)
      .map((t) => {
        const descriptor = descriptorMap.get(t.taskType);
        return {
          taskId: t.taskId!,
          taskType: t.taskType,
          title: t.title,
          outputs: descriptor?.outputs || [],
        };
      });
    return { globalVars, taskOutputs };
  }, [tasks, taskId, testVars, descriptorMap]);

  // Build suggestions list
  const allSuggestions = useMemo((): Suggestion[] => {
    const suggestions: Suggestion[] = [];

    for (const varName of variableContext.globalVars) {
      suggestions.push({
        label: varName,
        value: `.${varName}`,
        category: 'global',
      });
    }

    for (const task of variableContext.taskOutputs) {
      for (const output of task.outputs) {
        suggestions.push({
          label: output.name,
          value: `.tasks.${task.taskId}.outputs.${output.name}`,
          description: output.description || output.type,
          category: 'task',
          taskInfo: { taskId: task.taskId, title: task.title, taskType: task.taskType },
        });
      }
    }

    return suggestions;
  }, [variableContext]);

  // Filter based on current input
  const filteredSuggestions = useMemo(() => {
    if (!value) return allSuggestions;
    const lower = value.toLowerCase();
    return allSuggestions.filter(
      (s) =>
        s.label.toLowerCase().includes(lower) ||
        s.value.toLowerCase().includes(lower) ||
        s.taskInfo?.taskId.toLowerCase().includes(lower)
    );
  }, [allSuggestions, value]);

  // Handle suggestion selection
  const handleSelectSuggestion = useCallback((suggestion: Suggestion) => {
    onChange(suggestion.value);
    setShowSuggestions(false);
    inputRef.current?.focus();
  }, [onChange]);

  // Handle keyboard navigation
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (!showSuggestions || filteredSuggestions.length === 0) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setSelectedIndex((i) => Math.min(i + 1, filteredSuggestions.length - 1));
        break;
      case 'ArrowUp':
        e.preventDefault();
        setSelectedIndex((i) => Math.max(i - 1, 0));
        break;
      case 'Enter':
        e.preventDefault();
        if (filteredSuggestions[selectedIndex]) {
          handleSelectSuggestion(filteredSuggestions[selectedIndex]);
        }
        break;
      case 'Escape':
        e.preventDefault();
        setShowSuggestions(false);
        break;
      case 'Tab':
        if (filteredSuggestions[selectedIndex]) {
          e.preventDefault();
          handleSelectSuggestion(filteredSuggestions[selectedIndex]);
        }
        break;
    }
  }, [showSuggestions, filteredSuggestions, selectedIndex, handleSelectSuggestion]);

  // Reset selected index when filter changes
  useEffect(() => {
    setSelectedIndex(0);
  }, [value]);

  // Scroll selected item into view
  useEffect(() => {
    if (dropdownRef.current && showSuggestions) {
      const selectedEl = dropdownRef.current.querySelector('[data-selected="true"]');
      selectedEl?.scrollIntoView({ block: 'nearest' });
    }
  }, [selectedIndex, showSuggestions]);

  return (
    <div className="relative">
      <div className="flex items-center gap-2">
        <span className="text-xs text-primary-600 shrink-0">$</span>
        <input
          ref={inputRef}
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onFocus={() => setShowSuggestions(true)}
          onBlur={() => setTimeout(() => setShowSuggestions(false), 200)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          className="w-full px-2 py-1.5 text-sm font-mono bg-primary-50 dark:bg-primary-900/20 border border-primary-200 dark:border-primary-800 rounded-sm focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent"
        />
      </div>

      {/* Suggestions dropdown */}
      {showSuggestions && filteredSuggestions.length > 0 && (
        <div
          ref={dropdownRef}
          className="absolute left-0 right-0 top-full mt-1 z-50 max-h-48 overflow-y-auto bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-sm shadow-lg"
        >
          {filteredSuggestions.map((suggestion, index) => (
            <button
              key={suggestion.value}
              data-selected={index === selectedIndex}
              onMouseDown={(e) => {
                e.preventDefault();
                handleSelectSuggestion(suggestion);
              }}
              className={`
                w-full text-left px-2 py-1.5 text-sm transition-colors
                ${index === selectedIndex
                  ? 'bg-primary-50 dark:bg-primary-900/30'
                  : 'hover:bg-[var(--color-bg-tertiary)]'
                }
              `}
            >
              <div className="flex items-center gap-2">
                <span className={`px-1 py-0.5 text-xs rounded-sm ${
                  suggestion.category === 'global'
                    ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300'
                    : 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                }`}>
                  {suggestion.category === 'global' ? 'var' : 'out'}
                </span>
                <span className="font-mono text-xs truncate">{suggestion.value}</span>
              </div>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

export default ExpressionInput;
