import { useState, useCallback, useMemo, useRef, useEffect } from 'react';
import { createPortal } from 'react-dom';
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

// Group suggestions by task
interface SuggestionGroup {
  type: 'global' | 'task';
  taskInfo?: { taskId: string; title?: string; taskType: string };
  suggestions: Suggestion[];
}

interface DropdownPosition {
  top: number;
  left: number;
  direction: 'below' | 'above';
}

function VariableSelector({
  taskId,
  varValue,
  onVarChange,
}: VariableSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [filterText, setFilterText] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [dropdownPosition, setDropdownPosition] = useState<DropdownPosition | null>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
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

  // Group suggestions by task for better rendering
  const groupedSuggestions = useMemo((): SuggestionGroup[] => {
    const groups: SuggestionGroup[] = [];

    // Global vars group
    const globalSuggestions = allSuggestions.filter((s) => s.category === 'global');
    if (globalSuggestions.length > 0) {
      groups.push({ type: 'global', suggestions: globalSuggestions });
    }

    // Group task outputs by task
    const taskMap = new Map<string, Suggestion[]>();
    for (const s of allSuggestions) {
      if (s.category === 'task' && s.taskInfo) {
        const key = s.taskInfo.taskId;
        if (!taskMap.has(key)) {
          taskMap.set(key, []);
        }
        taskMap.get(key)!.push(s);
      }
    }

    for (const [, suggestions] of taskMap) {
      if (suggestions.length > 0) {
        groups.push({
          type: 'task',
          taskInfo: suggestions[0].taskInfo,
          suggestions,
        });
      }
    }

    return groups;
  }, [allSuggestions]);

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

  // Filter grouped suggestions
  const filteredGroups = useMemo((): SuggestionGroup[] => {
    if (!filterText) return groupedSuggestions;
    const lower = filterText.toLowerCase();

    return groupedSuggestions
      .map((group) => ({
        ...group,
        suggestions: group.suggestions.filter(
          (s) =>
            s.label.toLowerCase().includes(lower) ||
            s.value.toLowerCase().includes(lower) ||
            s.taskInfo?.taskId.toLowerCase().includes(lower) ||
            s.taskInfo?.title?.toLowerCase().includes(lower)
        ),
      }))
      .filter((group) => group.suggestions.length > 0);
  }, [groupedSuggestions, filterText]);

  const hasVar = !!varValue;

  // Calculate dropdown position when opening
  useEffect(() => {
    if (isOpen && buttonRef.current) {
      const buttonRect = buttonRef.current.getBoundingClientRect();
      const viewportHeight = window.innerHeight;
      const spaceBelow = viewportHeight - buttonRect.bottom;
      const spaceAbove = buttonRect.top;
      const dropdownHeight = 320; // Approximate max height of dropdown
      const dropdownWidth = 320;

      // Position above if not enough space below and more space above
      const direction = spaceBelow < dropdownHeight && spaceAbove > spaceBelow ? 'above' : 'below';

      // Calculate left position - align right edge with button right edge
      let left = buttonRect.right - dropdownWidth;
      // Ensure it doesn't go off-screen to the left
      if (left < 8) left = 8;

      setDropdownPosition({
        top: direction === 'below' ? buttonRect.bottom + 4 : buttonRect.top - dropdownHeight - 4,
        left,
        direction,
      });
    }
  }, [isOpen]);

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

  // Use custom expression
  const handleUseCustomExpression = useCallback(() => {
    // Use filter text as custom expression, prefixed with . if not already
    let expr = filterText.trim();
    if (expr && !expr.startsWith('.') && !expr.startsWith('|')) {
      expr = `.${expr}`;
    }
    if (expr) {
      onVarChange(expr);
    }
    setIsOpen(false);
    setFilterText('');
  }, [filterText, onVarChange]);

  // Handle keyboard navigation
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (!isOpen) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setSelectedIndex((i) => Math.min(i + 1, filteredSuggestions.length));
        break;
      case 'ArrowUp':
        e.preventDefault();
        setSelectedIndex((i) => Math.max(i - 1, 0));
        break;
      case 'Enter':
        e.preventDefault();
        if (selectedIndex < filteredSuggestions.length) {
          handleSelectSuggestion(filteredSuggestions[selectedIndex]);
        } else {
          // Use custom expression
          handleUseCustomExpression();
        }
        break;
      case 'Escape':
        e.preventDefault();
        setIsOpen(false);
        break;
    }
  }, [isOpen, filteredSuggestions, selectedIndex, handleSelectSuggestion, handleUseCustomExpression]);

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

  // Dropdown content rendered via portal
  const dropdownContent = isOpen && dropdownPosition && (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-[9998]"
        onClick={() => setIsOpen(false)}
      />

      {/* Dropdown with search */}
      <div
        style={{
          position: 'fixed',
          top: dropdownPosition.top,
          left: dropdownPosition.left,
          width: 320,
        }}
        className="z-[9999] bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-sm shadow-lg"
      >
        {/* Search input */}
        <div className="p-2 border-b border-[var(--color-border)]">
          <input
            ref={inputRef}
            type="text"
            value={filterText}
            onChange={(e) => setFilterText(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Search variables or type expression..."
            className="w-full px-2 py-1.5 text-sm bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-sm focus:outline-hidden focus:ring-2 focus:ring-primary-500"
          />
        </div>

        {/* Suggestions list */}
        <div ref={dropdownRef} className="max-h-64 overflow-y-auto">
          {filteredGroups.length > 0 ? (
            <div className="py-1">
              {filteredGroups.map((group, groupIndex) => {
                // Calculate the starting index for this group's suggestions
                let startIndex = 0;
                for (let i = 0; i < groupIndex; i++) {
                  startIndex += filteredGroups[i].suggestions.length;
                }

                return (
                  <div key={group.type === 'global' ? 'global' : group.taskInfo?.taskId} className="mb-2 last:mb-0">
                    {/* Group header */}
                    <div className="px-3 py-1.5 bg-[var(--color-bg-tertiary)] border-y border-[var(--color-border)]">
                      {group.type === 'global' ? (
                        <div className="flex items-center gap-2">
                          <span className="px-1.5 py-0.5 text-xs font-medium bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 rounded-sm">
                            vars
                          </span>
                          <span className="text-xs font-medium text-[var(--color-text-secondary)]">
                            Test Variables
                          </span>
                        </div>
                      ) : (
                        <div className="flex items-center gap-2">
                          <span className="px-1.5 py-0.5 text-xs font-medium bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded-sm">
                            task
                          </span>
                          <span className="text-xs font-medium text-[var(--color-text-secondary)] truncate">
                            {group.taskInfo?.title || group.taskInfo?.taskType}
                          </span>
                          <span className="text-xs font-mono text-[var(--color-text-tertiary)]">
                            #{group.taskInfo?.taskId}
                          </span>
                        </div>
                      )}
                    </div>

                    {/* Group suggestions */}
                    {group.suggestions.map((suggestion, index) => {
                      const globalIndex = startIndex + index;
                      return (
                        <button
                          key={suggestion.value}
                          data-selected={globalIndex === selectedIndex}
                          onClick={() => handleSelectSuggestion(suggestion)}
                          className={`
                            w-full text-left px-3 py-1.5 text-sm transition-colors
                            ${globalIndex === selectedIndex
                              ? 'bg-primary-50 dark:bg-primary-900/30'
                              : 'hover:bg-[var(--color-bg-tertiary)]'
                            }
                          `}
                        >
                          {suggestion.category === 'global' ? (
                            <span className="font-mono text-primary-600 dark:text-primary-400">{suggestion.value}</span>
                          ) : (
                            <div className="flex items-center gap-2">
                              <span className="font-medium">{suggestion.label}</span>
                              {suggestion.description && (
                                <span className="text-xs text-[var(--color-text-tertiary)] truncate">
                                  ({suggestion.description})
                                </span>
                              )}
                            </div>
                          )}
                        </button>
                      );
                    })}
                  </div>
                );
              })}
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

          {/* Custom expression option */}
          {filterText.trim() && (
            <button
              data-selected={selectedIndex === filteredSuggestions.length}
              onClick={handleUseCustomExpression}
              className={`
                w-full text-left px-3 py-2 text-sm border-t border-[var(--color-border)] transition-colors
                ${selectedIndex === filteredSuggestions.length
                  ? 'bg-primary-50 dark:bg-primary-900/30'
                  : 'hover:bg-[var(--color-bg-tertiary)]'
                }
              `}
            >
              <div className="flex items-center gap-2">
                <span className="px-1.5 py-0.5 text-xs font-medium bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300 rounded-sm">
                  custom
                </span>
                <span className="font-mono text-xs truncate">
                  {filterText.trim().startsWith('.') || filterText.trim().startsWith('|')
                    ? filterText.trim()
                    : `.${filterText.trim()}`
                  }
                </span>
              </div>
            </button>
          )}
        </div>

        {/* Hint */}
        <div className="p-2 border-t border-[var(--color-border)] text-xs text-[var(--color-text-tertiary)]">
          <span className="font-mono">↑↓</span> navigate, <span className="font-mono">Enter</span> select
          {filterText.trim() && (
            <span className="ml-2">• Type expression for custom JQ</span>
          )}
        </div>
      </div>
    </>
  );

  return (
    <div className="relative">
      <button
        ref={buttonRef}
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

      {/* Render dropdown via portal to escape overflow constraints */}
      {createPortal(dropdownContent, document.body)}
    </div>
  );
}

export default VariableSelector;
