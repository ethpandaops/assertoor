import { useMemo, useCallback, useState } from 'react';
import { useBuilderStore } from '../../../stores/builderStore';
import { useTaskDescriptors, useGlobalVariables } from '../../../hooks/useApi';
import { useAuthContext } from '../../../context/AuthContext';
import { findTaskById, canHaveChildren } from '../../../utils/builder/taskUtils';
import TaskConfigForm from './TaskConfigForm';
import type { TaskDescriptor } from '../../../types/api';

interface ConfigPanelProps {
  taskId: string;
}

function ConfigPanel({ taskId }: ConfigPanelProps) {
  // Check if this is the test header config
  if (taskId === '__test_header__') {
    return <TestHeaderConfig />;
  }

  return <TaskConfig taskId={taskId} />;
}

// Test header configuration component
function TestHeaderConfig() {
  const testConfig = useBuilderStore((state) => state.testConfig);
  const setTestName = useBuilderStore((state) => state.setTestName);
  const setTestTimeout = useBuilderStore((state) => state.setTestTimeout);
  const setTestVars = useBuilderStore((state) => state.setTestVars);
  const setTestId = useBuilderStore((state) => state.setTestId);
  const { isLoggedIn } = useAuthContext();
  const { data: globalVarsData } = useGlobalVariables({ enabled: isLoggedIn });

  const [newVarKey, setNewVarKey] = useState('');
  const [newVarValue, setNewVarValue] = useState('');

  // Handle adding a new variable
  const handleAddVar = useCallback(() => {
    if (!newVarKey.trim()) return;

    // Try to parse the value as JSON, otherwise treat as string
    let parsedValue: unknown = newVarValue;
    try {
      parsedValue = JSON.parse(newVarValue);
    } catch {
      // Keep as string
    }

    const newVars = {
      ...(testConfig.testVars || {}),
      [newVarKey.trim()]: parsedValue,
    };
    setTestVars(newVars);
    setNewVarKey('');
    setNewVarValue('');
  }, [newVarKey, newVarValue, testConfig.testVars, setTestVars]);

  // Handle adding a global variable suggestion (pass-through with empty default)
  const handleAddGlobalVar = useCallback((name: string) => {
    const newVars = {
      ...(testConfig.testVars || {}),
      [name]: '',
    };
    setTestVars(newVars);
  }, [testConfig.testVars, setTestVars]);

  // Global variable names that haven't been added yet
  const availableGlobalVars = useMemo(() => {
    if (!globalVarsData?.names) return [];
    const existing = new Set(Object.keys(testConfig.testVars || {}));
    return globalVarsData.names.filter((name) => !existing.has(name)).sort();
  }, [globalVarsData, testConfig.testVars]);

  // Handle removing a variable
  const handleRemoveVar = useCallback((key: string) => {
    const newVars = { ...(testConfig.testVars || {}) };
    delete newVars[key];
    setTestVars(newVars);
  }, [testConfig.testVars, setTestVars]);

  // Handle updating a variable value
  const handleUpdateVar = useCallback((key: string, value: string) => {
    let parsedValue: unknown = value;
    try {
      parsedValue = JSON.parse(value);
    } catch {
      // Keep as string
    }

    const newVars = {
      ...(testConfig.testVars || {}),
      [key]: parsedValue,
    };
    setTestVars(newVars);
  }, [testConfig.testVars, setTestVars]);

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="p-3 border-b border-[var(--color-border)]">
        <div className="flex items-center gap-2">
          <SettingsIcon className="size-5 text-emerald-600" />
          <h3 className="text-sm font-semibold">Test Configuration</h3>
        </div>
        <p className="text-xs text-[var(--color-text-secondary)] mt-1">
          Configure test metadata and default variables
        </p>
      </div>

      {/* Config form */}
      <div className="flex-1 overflow-y-auto p-3 space-y-4">
        {/* Basic properties */}
        <section>
          <h4 className="text-xs font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider mb-2">
            Basic
          </h4>
          <div className="space-y-3">
            {/* Test ID */}
            <div>
              <label className="block text-xs text-[var(--color-text-secondary)] mb-1">
                Test ID <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                value={testConfig.id}
                onChange={(e) => setTestId(e.target.value)}
                placeholder="e.g., my-test-id"
                className="w-full px-2 py-1.5 text-sm bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              />
              <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
                Unique identifier for the test
              </p>
            </div>

            {/* Test Name */}
            <div>
              <label className="block text-xs text-[var(--color-text-secondary)] mb-1">
                Name <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                value={testConfig.name}
                onChange={(e) => setTestName(e.target.value)}
                placeholder="Test name"
                className="w-full px-2 py-1.5 text-sm bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              />
            </div>

            {/* Timeout */}
            <div>
              <label className="block text-xs text-[var(--color-text-secondary)] mb-1">
                Timeout <span className="text-[var(--color-text-tertiary)]">(optional)</span>
              </label>
              <input
                type="text"
                value={testConfig.timeout || ''}
                onChange={(e) => setTestTimeout(e.target.value)}
                placeholder="e.g., 1h, 30m"
                className="w-full px-2 py-1.5 text-sm bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              />
              <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
                Maximum time for test execution
              </p>
            </div>
          </div>
        </section>

        {/* Default Variables */}
        <section>
          <h4 className="text-xs font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider mb-2">
            Default Variables
          </h4>
          <p className="text-xs text-[var(--color-text-tertiary)] mb-3">
            Variables available to all tasks in this test
          </p>

          {/* Existing variables */}
          {testConfig.testVars && Object.keys(testConfig.testVars).length > 0 && (
            <div className="space-y-2 mb-3">
              {Object.entries(testConfig.testVars).map(([key, value]) => (
                <div key={key} className="p-2 bg-[var(--color-bg-tertiary)] rounded-sm">
                  <div className="flex items-center justify-between mb-1">
                    <div className="text-xs font-mono text-primary-600 truncate">{key}</div>
                    <button
                      onClick={() => handleRemoveVar(key)}
                      className="p-0.5 hover:bg-red-100 dark:hover:bg-red-900/30 rounded-xs shrink-0"
                      title="Remove variable"
                    >
                      <DeleteIcon className="size-3.5 text-red-500" />
                    </button>
                  </div>
                  <input
                    type="text"
                    value={typeof value === 'string' ? value : JSON.stringify(value)}
                    onChange={(e) => handleUpdateVar(key, e.target.value)}
                    className="w-full px-2 py-1 text-xs font-mono bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-xs focus:outline-hidden focus:ring-1 focus:ring-primary-500"
                  />
                </div>
              ))}
            </div>
          )}

          {/* Global variable suggestions */}
          {availableGlobalVars.length > 0 && (
            <div className="mb-3">
              <div className="text-xs text-[var(--color-text-secondary)] mb-1.5">Global Variables</div>
              <div className="flex flex-wrap gap-1.5">
                {availableGlobalVars.map((name) => (
                  <button
                    key={name}
                    onClick={() => handleAddGlobalVar(name)}
                    className="px-2 py-0.5 text-xs font-mono bg-emerald-50 dark:bg-emerald-900/20 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800 rounded-xs hover:bg-emerald-100 dark:hover:bg-emerald-900/40 transition-colors"
                    title={`Add ${name} as a pass-through variable`}
                  >
                    + {name}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Add new variable */}
          <div className="p-2 border border-dashed border-[var(--color-border)] rounded-sm">
            <div className="space-y-1.5">
              <input
                type="text"
                value={newVarKey}
                onChange={(e) => setNewVarKey(e.target.value)}
                placeholder="Variable name"
                className="w-full px-2 py-1 text-xs bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-xs focus:outline-hidden focus:ring-1 focus:ring-primary-500"
              />
              <div className="flex gap-1.5">
                <input
                  type="text"
                  value={newVarValue}
                  onChange={(e) => setNewVarValue(e.target.value)}
                  placeholder="Value (optional)"
                  className="flex-1 px-2 py-1 text-xs font-mono bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-xs focus:outline-hidden focus:ring-1 focus:ring-primary-500"
                  onKeyDown={(e) => e.key === 'Enter' && handleAddVar()}
                />
                <button
                  onClick={handleAddVar}
                  disabled={!newVarKey.trim()}
                  className="px-2 py-1 text-xs bg-primary-600 text-white rounded-xs hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Add
                </button>
              </div>
            </div>
            <p className="text-xs text-[var(--color-text-tertiary)] mt-1.5">
              Values can be JSON (objects, arrays, numbers) or plain strings
            </p>
          </div>
        </section>
      </div>
    </div>
  );
}

// Task configuration component
function TaskConfig({ taskId }: { taskId: string }) {
  const tasks = useBuilderStore((state) => state.testConfig.tasks);
  const cleanupTasks = useBuilderStore((state) => state.testConfig.cleanupTasks || []);
  const updateTask = useBuilderStore((state) => state.updateTask);
  const updateCleanupTask = useBuilderStore((state) => state.updateCleanupTask);
  const removeTask = useBuilderStore((state) => state.removeTask);
  const removeCleanupTask = useBuilderStore((state) => state.removeCleanupTask);
  const duplicateTask = useBuilderStore((state) => state.duplicateTask);
  const { data: descriptors } = useTaskDescriptors();

  // Find the selected task (could be in main or cleanup tasks)
  const task = useMemo(() => {
    const mainTask = findTaskById(tasks, taskId);
    if (mainTask) return { task: mainTask, isCleanup: false };
    const cleanupTask = findTaskById(cleanupTasks, taskId);
    if (cleanupTask) return { task: cleanupTask, isCleanup: true };
    return null;
  }, [tasks, cleanupTasks, taskId]);

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

  // Get descriptor for current task
  const descriptor = useMemo(() => {
    if (!task) return null;
    return descriptorMap.get(task.task.taskType) || null;
  }, [task, descriptorMap]);

  // Filter config schema for glue tasks
  const { filteredConfigSchema, hasConfigProperties } = useMemo(() => {
    if (!descriptor?.configSchema) {
      return { filteredConfigSchema: null, hasConfigProperties: false };
    }

    if (!task || !canHaveChildren(task.task.taskType)) {
      const schema = descriptor.configSchema as Record<string, unknown>;
      const props = schema.properties as Record<string, unknown> | undefined;
      return {
        filteredConfigSchema: schema,
        hasConfigProperties: props && Object.keys(props).length > 0,
      };
    }

    const schema = { ...descriptor.configSchema } as Record<string, unknown>;
    if (schema.properties && typeof schema.properties === 'object') {
      const properties = { ...(schema.properties as Record<string, unknown>) };

      delete properties.tasks;
      delete properties.task;

      if (task.task.taskType === 'run_task_background') {
        delete properties.foregroundTask;
        delete properties.backgroundTask;
      }

      schema.properties = properties;

      if (Array.isArray(schema.required)) {
        const hiddenFields = ['tasks', 'task', 'foregroundTask', 'backgroundTask'];
        schema.required = (schema.required as string[]).filter((r) => !hiddenFields.includes(r));
      }

      return {
        filteredConfigSchema: schema,
        hasConfigProperties: Object.keys(properties).length > 0,
      };
    }

    return { filteredConfigSchema: schema, hasConfigProperties: false };
  }, [descriptor, task]);

  // Update handler based on whether it's a cleanup task
  const doUpdateTask = useCallback((updates: Parameters<typeof updateTask>[1]) => {
    if (!task) return;
    if (task.isCleanup) {
      updateCleanupTask(taskId, updates);
    } else {
      updateTask(taskId, updates);
    }
  }, [task, taskId, updateTask, updateCleanupTask]);

  const handleTaskIdChange = useCallback((value: string) => {
    doUpdateTask({ taskId: value || undefined });
  }, [doUpdateTask]);

  const handleTitleChange = useCallback((value: string) => {
    doUpdateTask({ title: value || undefined });
  }, [doUpdateTask]);

  const handleTimeoutChange = useCallback((value: string) => {
    doUpdateTask({ timeout: value || undefined });
  }, [doUpdateTask]);

  const handleIfConditionChange = useCallback((value: string) => {
    doUpdateTask({ ifCondition: value || undefined });
  }, [doUpdateTask]);

  const handleConfigChange = useCallback((key: string, value: unknown) => {
    if (!task) return;
    const newConfig = { ...task.task.config };
    if (value === undefined || value === '' || value === null) {
      delete newConfig[key];
    } else {
      newConfig[key] = value;
    }
    doUpdateTask({ config: newConfig });
  }, [doUpdateTask, task]);

  const handleConfigVarChange = useCallback((key: string, value: string) => {
    if (!task) return;
    const newConfigVars = { ...task.task.configVars };
    if (!value) {
      delete newConfigVars[key];
    } else {
      newConfigVars[key] = value;
    }
    doUpdateTask({ configVars: newConfigVars });
  }, [doUpdateTask, task]);

  const handleDelete = useCallback(() => {
    if (!task) return;
    if (confirm('Delete this task?')) {
      if (task.isCleanup) {
        removeCleanupTask(taskId);
      } else {
        removeTask(taskId);
      }
    }
  }, [task, taskId, removeTask, removeCleanupTask]);

  const handleDuplicate = useCallback(() => {
    // Note: duplicateTask only works for main tasks currently
    if (!task?.isCleanup) {
      duplicateTask(taskId);
    }
  }, [task, duplicateTask, taskId]);

  if (!task) {
    return (
      <div className="p-4 text-center text-[var(--color-text-secondary)]">
        <p>No task selected</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="p-3 border-b border-[var(--color-border)]">
        <div className="flex items-center justify-between mb-1">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-semibold">Task Configuration</h3>
            {task.isCleanup && (
              <span className="px-1.5 py-0.5 text-xs font-medium bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300 rounded">
                Cleanup
              </span>
            )}
          </div>
          <div className="flex items-center gap-1">
            {!task.isCleanup && (
              <button
                onClick={handleDuplicate}
                className="p-1.5 hover:bg-[var(--color-bg-tertiary)] rounded"
                title="Duplicate task"
              >
                <DuplicateIcon className="size-4 text-[var(--color-text-tertiary)]" />
              </button>
            )}
            <button
              onClick={handleDelete}
              className="p-1.5 hover:bg-red-100 dark:hover:bg-red-900/30 rounded"
              title="Delete task"
            >
              <DeleteIcon className="size-4 text-red-500" />
            </button>
          </div>
        </div>
        <p className="text-xs text-[var(--color-text-secondary)] font-mono">{task.task.taskType}</p>
      </div>

      {/* Config form */}
      <div className="flex-1 overflow-y-auto p-3 space-y-4">
        {/* Basic properties */}
        <section>
          <h4 className="text-xs font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider mb-2">
            Basic
          </h4>
          <div className="space-y-3">
            <div>
              <label className="block text-xs text-[var(--color-text-secondary)] mb-1">
                Task ID <span className="text-[var(--color-text-tertiary)]">(optional)</span>
              </label>
              <input
                type="text"
                value={task.task.taskId || ''}
                onChange={(e) => handleTaskIdChange(e.target.value)}
                placeholder="e.g., my-task-id"
                className="w-full px-2 py-1.5 text-sm bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              />
              <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
                Used for referencing task outputs
              </p>
            </div>

            <div>
              <label className="block text-xs text-[var(--color-text-secondary)] mb-1">
                Title <span className="text-[var(--color-text-tertiary)]">(optional)</span>
              </label>
              <input
                type="text"
                value={task.task.title || ''}
                onChange={(e) => handleTitleChange(e.target.value)}
                placeholder={task.task.taskType}
                className="w-full px-2 py-1.5 text-sm bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-xs text-[var(--color-text-secondary)] mb-1">
                Timeout <span className="text-[var(--color-text-tertiary)]">(optional)</span>
              </label>
              <input
                type="text"
                value={task.task.timeout || ''}
                onChange={(e) => handleTimeoutChange(e.target.value)}
                placeholder="e.g., 5m, 1h"
                className="w-full px-2 py-1.5 text-sm bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-xs text-[var(--color-text-secondary)] mb-1">
                Condition (if) <span className="text-[var(--color-text-tertiary)]">(optional)</span>
              </label>
              <input
                type="text"
                value={task.task.ifCondition || ''}
                onChange={(e) => handleIfConditionChange(e.target.value)}
                placeholder="e.g., .skipTask != true"
                className="w-full px-2 py-1.5 text-sm font-mono bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              />
              <p className="text-xs text-[var(--color-text-tertiary)] mt-1">
                JQ expression to conditionally skip task
              </p>
            </div>
          </div>
        </section>

        {/* Task-specific config */}
        {hasConfigProperties && filteredConfigSchema && (
          <section>
            <h4 className="text-xs font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider mb-2">
              Configuration
            </h4>
            <TaskConfigForm
              schema={filteredConfigSchema}
              config={task.task.config}
              configVars={task.task.configVars}
              onConfigChange={handleConfigChange}
              onConfigVarChange={handleConfigVarChange}
              taskId={taskId}
            />
          </section>
        )}

        {/* Task description */}
        {descriptor?.description && (
          <section>
            <h4 className="text-xs font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider mb-2">
              Description
            </h4>
            <p className="text-sm text-[var(--color-text-secondary)]">
              {descriptor.description}
            </p>
          </section>
        )}

        {/* Task outputs */}
        {descriptor?.outputs && descriptor.outputs.length > 0 && (
          <section>
            <h4 className="text-xs font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider mb-2">
              Outputs
            </h4>
            <div className="space-y-2">
              {descriptor.outputs.map((output) => (
                <div key={output.name} className="p-2 bg-[var(--color-bg-tertiary)] rounded">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-mono text-primary-600">{output.name}</span>
                    <span className="text-xs text-[var(--color-text-tertiary)]">({output.type})</span>
                  </div>
                  {output.description && (
                    <p className="text-xs text-[var(--color-text-secondary)] mt-1">
                      {output.description}
                    </p>
                  )}
                </div>
              ))}
            </div>
          </section>
        )}
      </div>
    </div>
  );
}

function SettingsIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
      />
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
      />
    </svg>
  );
}

function DuplicateIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
      />
    </svg>
  );
}

function DeleteIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
      />
    </svg>
  );
}

export default ConfigPanel;
