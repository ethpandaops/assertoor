import { useState, useCallback, useMemo, useRef, useEffect } from 'react';
import { createPortal } from 'react-dom';
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

// Group suggestions by task
interface SuggestionGroup {
  type: 'global' | 'task';
  taskInfo?: { taskId: string; title?: string; taskType: string };
  suggestions: Suggestion[];
}

interface DropdownPosition {
  top: number;
  left: number;
  width: number;
  direction: 'below' | 'above';
}

function ExpressionInput({
  taskId,
  value,
  onChange,
  placeholder = 'JQ expression',
}: ExpressionInputProps) {
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [dropdownPosition, setDropdownPosition] = useState<DropdownPosition | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

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

  // Filter grouped suggestions
  const filteredGroups = useMemo((): SuggestionGroup[] => {
    if (!value) return groupedSuggestions;
    const lower = value.toLowerCase();

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
  }, [groupedSuggestions, value]);

  // Calculate dropdown position when showing suggestions
  useEffect(() => {
    if (showSuggestions && containerRef.current) {
      const containerRect = containerRef.current.getBoundingClientRect();
      const viewportHeight = window.innerHeight;
      const spaceBelow = viewportHeight - containerRect.bottom;
      const spaceAbove = containerRect.top;
      const dropdownHeight = 240; // Approximate max height of dropdown

      // Position above if not enough space below and more space above
      const direction = spaceBelow < dropdownHeight && spaceAbove > spaceBelow ? 'above' : 'below';

      setDropdownPosition({
        top: direction === 'below' ? containerRect.bottom + 4 : containerRect.top - dropdownHeight - 4,
        left: containerRect.left,
        width: containerRect.width,
        direction,
      });
    }
  }, [showSuggestions]);

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

  // Dropdown content rendered via portal
  const dropdownContent = showSuggestions && filteredGroups.length > 0 && dropdownPosition && (
    <>
      {/* Backdrop to close on click outside */}
      <div
        className="fixed inset-0 z-[9998]"
        onClick={() => setShowSuggestions(false)}
      />

      {/* Suggestions dropdown */}
      <div
        ref={dropdownRef}
        style={{
          position: 'fixed',
          top: dropdownPosition.top,
          left: dropdownPosition.left,
          width: dropdownPosition.width,
        }}
        className="z-[9999] max-h-48 overflow-y-auto bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-sm shadow-lg"
      >
        {filteredGroups.map((group, groupIndex) => {
          // Calculate the starting index for this group's suggestions
          let startIndex = 0;
          for (let i = 0; i < groupIndex; i++) {
            startIndex += filteredGroups[i].suggestions.length;
          }

          return (
            <div key={group.type === 'global' ? 'global' : group.taskInfo?.taskId} className="border-b border-[var(--color-border)] last:border-b-0">
              {/* Group header */}
              <div className="px-2 py-1 bg-[var(--color-bg-tertiary)] text-xs">
                {group.type === 'global' ? (
                  <span className="font-medium text-green-600 dark:text-green-400">Test Variables</span>
                ) : (
                  <div className="flex items-center gap-1.5">
                    <span className="font-medium text-blue-600 dark:text-blue-400 truncate">
                      {group.taskInfo?.title || group.taskInfo?.taskType}
                    </span>
                    <span className="font-mono text-[var(--color-text-tertiary)]">
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
                    onMouseDown={(e) => {
                      e.preventDefault();
                      handleSelectSuggestion(suggestion);
                    }}
                    className={`
                      w-full text-left px-2 py-1 text-sm transition-colors
                      ${globalIndex === selectedIndex
                        ? 'bg-primary-50 dark:bg-primary-900/30'
                        : 'hover:bg-[var(--color-bg-tertiary)]'
                      }
                    `}
                  >
                    {suggestion.category === 'global' ? (
                      <span className="font-mono text-xs">{suggestion.value}</span>
                    ) : (
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-xs">{suggestion.label}</span>
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
    </>
  );

  return (
    <div ref={containerRef} className="relative">
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

      {/* Render dropdown via portal to escape overflow constraints */}
      {createPortal(dropdownContent, document.body)}
    </div>
  );
}

export default ExpressionInput;
