import { useState, useCallback, useMemo, useRef, useEffect } from 'react';
import { useBuilderStore } from '../../../stores/builderStore';
import { useTaskDescriptors } from '../../../hooks/useApi';
import { findPrecedingTasks } from '../../../utils/builder/taskUtils';
import type { TaskDescriptor, TaskOutputField } from '../../../types/api';

interface VariableSelectorProps {
  taskId: string;
  varValue: string | undefined;
  onVarChange: (value: string) => void;
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

function VariableSelector({
  taskId,
  varValue,
  onVarChange,
}: VariableSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [filterText, setFilterText] = useState('');
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
    // Global test vars
    const globalVars = Object.keys(testVars || {});

    // Preceding task outputs
    const precedingTasks = findPrecedingTasks(tasks, taskId);
    const taskOutputs: TaskOutput[] = precedingTasks
      .filter((t) => t.taskId) // Only tasks with an ID can be referenced
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

  // Build suggestions list for typeahead
  const allSuggestions = useMemo((): Suggestion[] => {
    const suggestions: Suggestion[] = [];

    // Global vars
    for (const varName of variableContext.globalVars) {
      suggestions.push({
        label: varName,
        value: `.${varName}`,
        category: 'global',
      });
    }

    // Task outputs - use lowercase .tasks. and .outputs.
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

  // Filter suggestions based on input
  const filteredSuggestions = useMemo(() => {
    if (!filterText) return allSuggestions;
    const lower = filterText.toLowerCase();
    return allSuggestions.filter(
      (s) =>
        s.label.toLowerCase().includes(lower) ||
        s.value.toLowerCase().includes(lower) ||
        s.taskInfo?.taskId.toLowerCase().includes(lower) ||
        s.taskInfo?.title?.toLowerCase().includes(lower)
    );
  }, [allSuggestions, filterText]);

  const hasVar = !!varValue;

  // Toggle variable mode
  const handleToggle = useCallback(() => {
    if (hasVar) {
      // Clear variable
      onVarChange('');
      setFilterText('');
    } else {
      // Enable variable mode
      setIsOpen(true);
      setFilterText('');
      setTimeout(() => inputRef.current?.focus(), 0);
    }
  }, [hasVar, onVarChange]);

  // Select a suggestion
  const handleSelectSuggestion = useCallback((suggestion: Suggestion) => {
    onVarChange(suggestion.value);
    setIsOpen(false);
    setFilterText('');
  }, [onVarChange]);

  // Handle keyboard navigation
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (!isOpen) return;

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
        setIsOpen(false);
        break;
    }
  }, [isOpen, filteredSuggestions, selectedIndex, handleSelectSuggestion]);

  // Reset selected index when filter changes
  useEffect(() => {
    setSelectedIndex(0);
  }, [filterText]);

  // Scroll selected item into view
  useEffect(() => {
    if (dropdownRef.current && isOpen) {
      const selectedEl = dropdownRef.current.querySelector('[data-selected="true"]');
      selectedEl?.scrollIntoView({ block: 'nearest' });
    }
  }, [selectedIndex, isOpen]);

  return (
    <div className="relative">
      <button
        onClick={handleToggle}
        className={`
          px-1.5 py-0.5 text-xs rounded-sm transition-colors
          ${hasVar
            ? 'bg-primary-100 dark:bg-primary-900/30 text-primary-700 dark:text-primary-300'
            : 'text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-tertiary)]'
          }
        `}
        title={hasVar ? 'Using variable reference' : 'Use variable reference'}
      >
        <span className="font-mono">$</span>
        {hasVar ? ' on' : ''}
      </button>

      {/* Variable picker dropdown */}
      {isOpen && (
        <>
          {/* Backdrop */}
          <div
            className="fixed inset-0 z-40"
            onClick={() => setIsOpen(false)}
          />

          {/* Dropdown with search */}
          <div className="absolute right-0 top-full mt-1 z-50 w-80 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-sm shadow-lg">
            {/* Search input */}
            <div className="p-2 border-b border-[var(--color-border)]">
              <input
                ref={inputRef}
                type="text"
                value={filterText}
                onChange={(e) => setFilterText(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Search variables..."
                className="w-full px-2 py-1.5 text-sm bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-sm focus:outline-hidden focus:ring-2 focus:ring-primary-500"
              />
            </div>

            {/* Suggestions list */}
            <div ref={dropdownRef} className="max-h-64 overflow-y-auto">
              {filteredSuggestions.length > 0 ? (
                <div className="py-1">
                  {filteredSuggestions.map((suggestion, index) => (
                    <button
                      key={suggestion.value}
                      data-selected={index === selectedIndex}
                      onClick={() => handleSelectSuggestion(suggestion)}
                      className={`
                        w-full text-left px-3 py-2 text-sm transition-colors
                        ${index === selectedIndex
                          ? 'bg-primary-50 dark:bg-primary-900/30'
                          : 'hover:bg-[var(--color-bg-tertiary)]'
                        }
                      `}
                    >
                      {suggestion.category === 'global' ? (
                        <div className="flex items-center gap-2">
                          <span className="px-1.5 py-0.5 text-xs bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 rounded-sm">
                            var
                          </span>
                          <span className="font-mono text-primary-600">{suggestion.value}</span>
                        </div>
                      ) : (
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="px-1.5 py-0.5 text-xs bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded-sm">
                              output
                            </span>
                            <span className="font-medium">{suggestion.label}</span>
                            {suggestion.description && (
                              <span className="text-xs text-[var(--color-text-tertiary)]">
                                ({suggestion.description})
                              </span>
                            )}
                          </div>
                          {suggestion.taskInfo && (
                            <div className="ml-12 mt-0.5 text-xs text-[var(--color-text-tertiary)]">
                              from{' '}
                              <span className="font-medium">
                                {suggestion.taskInfo.title || suggestion.taskInfo.taskType}
                              </span>
                              {' '}
                              <span className="font-mono">#{suggestion.taskInfo.taskId}</span>
                            </div>
                          )}
                        </div>
                      )}
                    </button>
                  ))}
                </div>
              ) : (
                <div className="p-3 text-center">
                  <p className="text-sm text-[var(--color-text-tertiary)]">
                    {allSuggestions.length === 0 ? 'No variables available' : 'No matching variables'}
                  </p>
                  {allSuggestions.length === 0 && (
                    <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
                      Add test vars or give preceding tasks an ID
                    </p>
                  )}
                </div>
              )}
            </div>

            {/* Hint */}
            <div className="p-2 border-t border-[var(--color-border)] text-xs text-[var(--color-text-tertiary)]">
              <span className="font-mono">↑↓</span> to navigate, <span className="font-mono">Enter</span> to select
            </div>
          </div>
        </>
      )}
    </div>
  );
}

export default VariableSelector;
