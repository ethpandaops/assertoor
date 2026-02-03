import { create } from 'zustand';
import type { TestDetails, TaskDescriptor } from '../types/api';
import { serializeToYaml, deserializeFromYaml } from '../utils/builder/yamlSerializer';
import { generateTaskId, findTaskById, removeTaskById, insertTaskAt, moveTaskTo } from '../utils/builder/taskUtils';

// Configuration for named child slots
export interface NamedChildSlot {
  name: string;           // Slot name (e.g., 'background', 'foreground')
  label: string;          // Display label (e.g., 'BG', 'FG')
  yamlKey: string;        // Key used in YAML config (e.g., 'backgroundTask')
  colorClass: string;     // Tailwind color class for styling
}

// Task types that have named child slots instead of a children array
export const NAMED_CHILD_TASK_TYPES: Record<string, NamedChildSlot[]> = {
  'run_task_background': [
    { name: 'background', label: 'BG', yamlKey: 'backgroundTask', colorClass: 'amber' },
    { name: 'foreground', label: 'FG', yamlKey: 'foregroundTask', colorClass: 'emerald' },
  ],
  // Future task types with named children can be added here
};

// Helper to get named child config for a task type
export function getNamedChildSlots(taskType: string): NamedChildSlot[] | undefined {
  return NAMED_CHILD_TASK_TYPES[taskType];
}

// Helper to check if a task type uses named children
export function hasNamedChildren(taskType: string): boolean {
  return taskType in NAMED_CHILD_TASK_TYPES;
}

// Helper to get slot index by name
export function getSlotIndex(taskType: string, slotName: string): number {
  const slots = NAMED_CHILD_TASK_TYPES[taskType];
  if (!slots) return -1;
  return slots.findIndex((s) => s.name === slotName);
}

// Helper to get slot name by index
export function getSlotName(taskType: string, index: number): string | undefined {
  const slots = NAMED_CHILD_TASK_TYPES[taskType];
  return slots?.[index]?.name;
}

// Builder task representation
export interface BuilderTask {
  id: string;                          // UUID for internal tracking
  taskType: string;                    // e.g., "check_clients_are_healthy"
  taskId?: string;                     // Optional user-defined ID for variable refs
  title?: string;                      // Optional display name
  timeout?: string;                    // e.g., "5m"
  ifCondition?: string;                // Optional skip condition
  config: Record<string, unknown>;     // Direct config values
  configVars: Record<string, string>;  // JQ expressions for dynamic values
  children?: BuilderTask[];            // For run_tasks, run_tasks_concurrent, run_task_options, run_task_matrix
  namedChildren?: Record<string, BuilderTask>;  // For tasks with named child slots (e.g., run_task_background)
}

// Test configuration structure
export interface TestConfig {
  id?: string;
  name: string;
  timeout?: string;
  testVars?: Record<string, unknown>;
  tasks: BuilderTask[];
  cleanupTasks?: BuilderTask[];
}

// Validation error
export interface ValidationError {
  taskId?: string;
  field?: string;
  message: string;
  severity: 'error' | 'warning';
}

// Selection state
export interface SelectionState {
  taskIds: Set<string>;
  primaryTaskId: string | null;
}

// Builder store state
export interface BuilderState {
  // Test configuration
  testConfig: TestConfig;

  // UI state
  activeView: 'graph' | 'list' | 'yaml';
  selection: SelectionState;
  yamlSource: string;
  validationErrors: ValidationError[];
  isDirty: boolean;

  // Source test ID (when editing existing test)
  sourceTestId: string | null;

  // Actions
  setTestConfig: (config: TestConfig) => void;
  setTestName: (name: string) => void;
  setTestTimeout: (timeout: string) => void;
  setTestVars: (vars: Record<string, unknown>) => void;
  setTestId: (id: string) => void;

  // Task operations
  addTask: (task: BuilderTask, parentId?: string, index?: number) => void;
  updateTask: (taskId: string, updates: Partial<BuilderTask>) => void;
  removeTask: (taskId: string) => void;
  moveTask: (taskId: string, targetParentId: string | undefined, targetIndex: number) => void;
  duplicateTask: (taskId: string) => void;

  // Cleanup task operations
  addCleanupTask: (task: BuilderTask, parentId?: string, index?: number) => void;
  updateCleanupTask: (taskId: string, updates: Partial<BuilderTask>) => void;
  removeCleanupTask: (taskId: string) => void;
  moveCleanupTask: (taskId: string, targetParentId: string | undefined, targetIndex: number) => void;

  // Cross-area task operations
  moveTaskToCleanup: (taskId: string, targetParentId: string | undefined, targetIndex: number) => void;
  moveCleanupTaskToMain: (taskId: string, targetParentId: string | undefined, targetIndex: number) => void;

  // Selection
  setSelection: (taskIds: string[], primaryId?: string) => void;
  addToSelection: (taskId: string) => void;
  removeFromSelection: (taskId: string) => void;
  clearSelection: () => void;
  selectAll: () => void;

  // View
  setActiveView: (view: 'graph' | 'list' | 'yaml') => void;

  // YAML sync
  setYamlSource: (yaml: string) => void;
  syncToYaml: () => void;
  syncFromYaml: () => boolean; // Returns true if parse succeeded

  // Load/export
  loadTest: (testDetails: TestDetails, descriptors: Map<string, TaskDescriptor>) => void;
  loadFromYaml: (yaml: string) => boolean;
  reset: () => void;
  exportYaml: () => string;

  // Validation
  validate: (descriptors: Map<string, TaskDescriptor>) => ValidationError[];
  clearValidationErrors: () => void;
}

// Default empty test config
const createEmptyTestConfig = (): TestConfig => ({
  name: 'New Test',
  tasks: [],
});

// Helper to recursively update a task tree (generic traversal for namedChildren)
function traverseAndUpdate(
  tasks: BuilderTask[],
  callback: (task: BuilderTask) => BuilderTask | null // Return null to keep unchanged, or updated task
): BuilderTask[] {
  return tasks.map((task) => {
    const result = callback(task);
    if (result !== null) {
      return result;
    }

    let updated = task;
    let wasUpdated = false;

    if (task.children) {
      const newChildren = traverseAndUpdate(task.children, callback);
      if (newChildren !== task.children) {
        updated = { ...updated, children: newChildren };
        wasUpdated = true;
      }
    }

    if (task.namedChildren) {
      const newNamedChildren: Record<string, BuilderTask> = {};
      let namedChildrenUpdated = false;

      for (const [slotName, child] of Object.entries(task.namedChildren)) {
        const [newChild] = traverseAndUpdate([child], callback);
        if (newChild !== child) {
          namedChildrenUpdated = true;
        }
        newNamedChildren[slotName] = newChild;
      }

      if (namedChildrenUpdated) {
        updated = { ...updated, namedChildren: newNamedChildren };
        wasUpdated = true;
      }
    }

    return wasUpdated ? updated : task;
  });
}

// Helper to set a named child slot
function setNamedChildSlot(
  tasks: BuilderTask[],
  parentId: string,
  slotName: string,
  newTask: BuilderTask | undefined
): BuilderTask[] {
  return traverseAndUpdate(tasks, (task) => {
    if (task.id === parentId) {
      const namedChildren = { ...(task.namedChildren || {}) };
      if (newTask) {
        namedChildren[slotName] = newTask;
      } else {
        delete namedChildren[slotName];
      }
      return {
        ...task,
        namedChildren: Object.keys(namedChildren).length > 0 ? namedChildren : undefined,
      };
    }
    return null; // Continue traversal
  });
}

// Create the builder store
export const useBuilderStore = create<BuilderState>((set, get) => ({
  // Initial state
  testConfig: createEmptyTestConfig(),
  activeView: 'list',
  selection: { taskIds: new Set(), primaryTaskId: null },
  yamlSource: '',
  validationErrors: [],
  isDirty: false,
  sourceTestId: null,

  // Set entire test config
  setTestConfig: (config) => set({
    testConfig: config,
    isDirty: true,
    yamlSource: serializeToYaml(config),
  }),

  // Update test metadata
  setTestName: (name) => set((state) => ({
    testConfig: { ...state.testConfig, name },
    isDirty: true,
  })),

  setTestTimeout: (timeout) => set((state) => ({
    testConfig: { ...state.testConfig, timeout: timeout || undefined },
    isDirty: true,
  })),

  setTestVars: (vars) => set((state) => ({
    testConfig: { ...state.testConfig, testVars: Object.keys(vars).length > 0 ? vars : undefined },
    isDirty: true,
  })),

  setTestId: (id) => set((state) => ({
    testConfig: { ...state.testConfig, id: id || undefined },
    isDirty: true,
  })),

  // Add a task
  addTask: (task, parentId, index) => set((state) => {
    const newTask = { ...task, id: task.id || generateTaskId() };

    if (parentId) {
      const parent = findTaskById(state.testConfig.tasks, parentId);
      if (parent && hasNamedChildren(parent.taskType) && index !== undefined) {
        const slotName = getSlotName(parent.taskType, index);
        if (!slotName) return state;

        // If slot is already filled, don't add
        if (parent.namedChildren?.[slotName]) {
          return state;
        }

        return {
          testConfig: {
            ...state.testConfig,
            tasks: setNamedChildSlot(state.testConfig.tasks, parentId, slotName, newTask),
          },
          isDirty: true,
        };
      }

      // Standard insertion for other parent tasks
      const newTasks = insertTaskAt(state.testConfig.tasks, newTask, parentId, index);
      return {
        testConfig: { ...state.testConfig, tasks: newTasks },
        isDirty: true,
      };
    } else {
      // Add to root level
      const tasks = [...state.testConfig.tasks];
      if (index !== undefined && index >= 0 && index <= tasks.length) {
        tasks.splice(index, 0, newTask);
      } else {
        tasks.push(newTask);
      }
      return {
        testConfig: { ...state.testConfig, tasks },
        isDirty: true,
      };
    }
  }),

  // Update a task
  updateTask: (taskId, updates) => set((state) => {
    const newTasks = traverseAndUpdate(state.testConfig.tasks, (task) => {
      if (task.id === taskId) {
        return { ...task, ...updates };
      }
      return null;
    });

    return {
      testConfig: { ...state.testConfig, tasks: newTasks },
      isDirty: true,
    };
  }),

  // Remove a task
  removeTask: (taskId) => set((state) => {
    const newTasks = removeTaskById(state.testConfig.tasks, taskId);
    const newSelection = new Set(state.selection.taskIds);
    newSelection.delete(taskId);

    return {
      testConfig: { ...state.testConfig, tasks: newTasks },
      selection: {
        taskIds: newSelection,
        primaryTaskId: state.selection.primaryTaskId === taskId ? null : state.selection.primaryTaskId,
      },
      isDirty: true,
    };
  }),

  // Move a task
  moveTask: (taskId, targetParentId, targetIndex) => set((state) => {
    const task = findTaskById(state.testConfig.tasks, taskId);
    if (!task) return state;

    // Remove from current position
    let newTasks = removeTaskById(state.testConfig.tasks, taskId);

    // Check if target parent uses named children
    if (targetParentId) {
      const parent = findTaskById(newTasks, targetParentId);
      if (parent && hasNamedChildren(parent.taskType)) {
        const slotName = getSlotName(parent.taskType, targetIndex);
        if (!slotName) {
          // Invalid slot index, move to root
          newTasks = moveTaskTo(newTasks, task, undefined, state.testConfig.tasks.length);
          return { testConfig: { ...state.testConfig, tasks: newTasks }, isDirty: false };
        }

        // If slot is already filled, don't move
        if (parent.namedChildren?.[slotName]) {
          newTasks = moveTaskTo(newTasks, task, undefined, state.testConfig.tasks.length);
          return { testConfig: { ...state.testConfig, tasks: newTasks }, isDirty: false };
        }

        return {
          testConfig: {
            ...state.testConfig,
            tasks: setNamedChildSlot(newTasks, targetParentId, slotName, task),
          },
          isDirty: true,
        };
      }
    }

    // Standard insertion for other parent tasks
    newTasks = moveTaskTo(newTasks, task, targetParentId, targetIndex);

    return {
      testConfig: { ...state.testConfig, tasks: newTasks },
      isDirty: true,
    };
  }),

  // Cleanup task operations
  addCleanupTask: (task, parentId, index) => set((state) => {
    const newTask = { ...task, id: task.id || generateTaskId() };
    const cleanupTasks = state.testConfig.cleanupTasks || [];

    if (parentId) {
      const parent = findTaskById(cleanupTasks, parentId);
      if (parent && hasNamedChildren(parent.taskType) && index !== undefined) {
        const slotName = getSlotName(parent.taskType, index);
        if (!slotName) return state;

        if (parent.namedChildren?.[slotName]) {
          return state;
        }

        return {
          testConfig: {
            ...state.testConfig,
            cleanupTasks: setNamedChildSlot(cleanupTasks, parentId, slotName, newTask),
          },
          isDirty: true,
        };
      }

      const newTasks = insertTaskAt(cleanupTasks, newTask, parentId, index);
      return {
        testConfig: { ...state.testConfig, cleanupTasks: newTasks },
        isDirty: true,
      };
    } else {
      const tasks = [...cleanupTasks];
      if (index !== undefined && index >= 0 && index <= tasks.length) {
        tasks.splice(index, 0, newTask);
      } else {
        tasks.push(newTask);
      }
      return {
        testConfig: { ...state.testConfig, cleanupTasks: tasks },
        isDirty: true,
      };
    }
  }),

  updateCleanupTask: (taskId, updates) => set((state) => {
    const newTasks = traverseAndUpdate(state.testConfig.cleanupTasks || [], (task) => {
      if (task.id === taskId) {
        return { ...task, ...updates };
      }
      return null;
    });

    return {
      testConfig: { ...state.testConfig, cleanupTasks: newTasks },
      isDirty: true,
    };
  }),

  removeCleanupTask: (taskId) => set((state) => {
    const newTasks = removeTaskById(state.testConfig.cleanupTasks || [], taskId);
    const newSelection = new Set(state.selection.taskIds);
    newSelection.delete(taskId);

    return {
      testConfig: { ...state.testConfig, cleanupTasks: newTasks.length > 0 ? newTasks : undefined },
      selection: {
        taskIds: newSelection,
        primaryTaskId: state.selection.primaryTaskId === taskId ? null : state.selection.primaryTaskId,
      },
      isDirty: true,
    };
  }),

  moveCleanupTask: (taskId, targetParentId, targetIndex) => set((state) => {
    const cleanupTasks = state.testConfig.cleanupTasks || [];
    const task = findTaskById(cleanupTasks, taskId);
    if (!task) return state;

    let newTasks = removeTaskById(cleanupTasks, taskId);

    if (targetParentId) {
      const parent = findTaskById(newTasks, targetParentId);
      if (parent && hasNamedChildren(parent.taskType)) {
        const slotName = getSlotName(parent.taskType, targetIndex);
        if (!slotName) {
          newTasks = moveTaskTo(newTasks, task, undefined, cleanupTasks.length);
          return { testConfig: { ...state.testConfig, cleanupTasks: newTasks }, isDirty: false };
        }

        if (parent.namedChildren?.[slotName]) {
          newTasks = moveTaskTo(newTasks, task, undefined, cleanupTasks.length);
          return { testConfig: { ...state.testConfig, cleanupTasks: newTasks }, isDirty: false };
        }

        return {
          testConfig: {
            ...state.testConfig,
            cleanupTasks: setNamedChildSlot(newTasks, targetParentId, slotName, task),
          },
          isDirty: true,
        };
      }
    }

    newTasks = moveTaskTo(newTasks, task, targetParentId, targetIndex);

    return {
      testConfig: { ...state.testConfig, cleanupTasks: newTasks },
      isDirty: true,
    };
  }),

  // Move task from main to cleanup
  moveTaskToCleanup: (taskId, targetParentId, targetIndex) => set((state) => {
    const task = findTaskById(state.testConfig.tasks, taskId);
    if (!task) return state;

    const newMainTasks = removeTaskById(state.testConfig.tasks, taskId);
    const cleanupTasks = state.testConfig.cleanupTasks || [];

    if (targetParentId) {
      const parent = findTaskById(cleanupTasks, targetParentId);
      if (parent && hasNamedChildren(parent.taskType)) {
        const slotName = getSlotName(parent.taskType, targetIndex);
        if (!slotName || parent.namedChildren?.[slotName]) {
          return state;
        }

        return {
          testConfig: {
            ...state.testConfig,
            tasks: newMainTasks,
            cleanupTasks: setNamedChildSlot(cleanupTasks, targetParentId, slotName, task),
          },
          isDirty: true,
        };
      }
    }

    const newCleanupTasks = moveTaskTo(cleanupTasks, task, targetParentId, targetIndex);

    return {
      testConfig: {
        ...state.testConfig,
        tasks: newMainTasks,
        cleanupTasks: newCleanupTasks,
      },
      isDirty: true,
    };
  }),

  // Move task from cleanup to main
  moveCleanupTaskToMain: (taskId, targetParentId, targetIndex) => set((state) => {
    const cleanupTasks = state.testConfig.cleanupTasks || [];
    const task = findTaskById(cleanupTasks, taskId);
    if (!task) return state;

    // Check if target slot is available before removing
    if (targetParentId) {
      const parent = findTaskById(state.testConfig.tasks, targetParentId);
      if (parent && hasNamedChildren(parent.taskType)) {
        const slotName = getSlotName(parent.taskType, targetIndex);
        if (!slotName || parent.namedChildren?.[slotName]) {
          return state;
        }
      }
    }

    let newCleanupTasks: BuilderTask[] | undefined = removeTaskById(cleanupTasks, taskId);
    if (newCleanupTasks.length === 0) {
      newCleanupTasks = undefined;
    }

    if (targetParentId) {
      const parent = findTaskById(state.testConfig.tasks, targetParentId);
      if (parent && hasNamedChildren(parent.taskType)) {
        const slotName = getSlotName(parent.taskType, targetIndex)!;

        return {
          testConfig: {
            ...state.testConfig,
            tasks: setNamedChildSlot(state.testConfig.tasks, targetParentId, slotName, task),
            cleanupTasks: newCleanupTasks,
          },
          isDirty: true,
        };
      }
    }

    const newMainTasks = moveTaskTo(state.testConfig.tasks, task, targetParentId, targetIndex);

    return {
      testConfig: {
        ...state.testConfig,
        tasks: newMainTasks,
        cleanupTasks: newCleanupTasks,
      },
      isDirty: true,
    };
  }),

  // Duplicate a task
  duplicateTask: (taskId) => set((state) => {
    const task = findTaskById(state.testConfig.tasks, taskId);
    if (!task) return state;

    const duplicateWithNewIds = (t: BuilderTask): BuilderTask => {
      const newTask: BuilderTask = {
        ...t,
        id: generateTaskId(),
        taskId: t.taskId ? `${t.taskId}_copy` : undefined,
        children: t.children?.map(duplicateWithNewIds),
      };

      if (t.namedChildren) {
        newTask.namedChildren = {};
        for (const [slotName, child] of Object.entries(t.namedChildren)) {
          newTask.namedChildren[slotName] = duplicateWithNewIds(child);
        }
      }

      return newTask;
    };

    const newTask = duplicateWithNewIds(task);

    // Find parent and index
    const findParentAndIndex = (
      tasks: BuilderTask[],
      searchId: string,
      parent: string | null = null
    ): { parentId: string | null; index: number; slotName?: string } | null => {
      for (let i = 0; i < tasks.length; i++) {
        const t = tasks[i];
        if (t.id === searchId) {
          return { parentId: parent, index: i + 1 };
        }
        if (t.children) {
          const found = findParentAndIndex(t.children, searchId, t.id);
          if (found) return found;
        }
        if (t.namedChildren) {
          const slots = getNamedChildSlots(t.taskType);
          if (slots) {
            for (let slotIdx = 0; slotIdx < slots.length; slotIdx++) {
              const slot = slots[slotIdx];
              const child = t.namedChildren[slot.name];
              if (child) {
                if (child.id === searchId) {
                  // Can't duplicate into same named slot, return null
                  return null;
                }
                const found = findParentAndIndex([child], searchId, t.id);
                if (found) return found;
              }
            }
          }
        }
      }
      return null;
    };

    const location = findParentAndIndex(state.testConfig.tasks, taskId);
    if (!location) return state;

    if (location.parentId) {
      const newTasks = insertTaskAt(state.testConfig.tasks, newTask, location.parentId, location.index);
      return {
        testConfig: { ...state.testConfig, tasks: newTasks },
        isDirty: true,
      };
    } else {
      const tasks = [...state.testConfig.tasks];
      tasks.splice(location.index, 0, newTask);
      return {
        testConfig: { ...state.testConfig, tasks },
        isDirty: true,
      };
    }
  }),

  // Selection management
  setSelection: (taskIds, primaryId) => set({
    selection: {
      taskIds: new Set(taskIds),
      primaryTaskId: primaryId ?? taskIds[0] ?? null,
    },
  }),

  addToSelection: (taskId) => set((state) => {
    const newSet = new Set(state.selection.taskIds);
    newSet.add(taskId);
    return {
      selection: {
        taskIds: newSet,
        primaryTaskId: state.selection.primaryTaskId || taskId,
      },
    };
  }),

  removeFromSelection: (taskId) => set((state) => {
    const newSet = new Set(state.selection.taskIds);
    newSet.delete(taskId);
    return {
      selection: {
        taskIds: newSet,
        primaryTaskId: state.selection.primaryTaskId === taskId
          ? (newSet.size > 0 ? Array.from(newSet)[0] : null)
          : state.selection.primaryTaskId,
      },
    };
  }),

  clearSelection: () => set({
    selection: { taskIds: new Set(), primaryTaskId: null },
  }),

  selectAll: () => set((state) => {
    const getAllIds = (tasks: BuilderTask[]): string[] => {
      const ids: string[] = [];
      for (const task of tasks) {
        ids.push(task.id);
        if (task.children) {
          ids.push(...getAllIds(task.children));
        }
        if (task.namedChildren) {
          for (const child of Object.values(task.namedChildren)) {
            ids.push(...getAllIds([child]));
          }
        }
      }
      return ids;
    };

    const allIds = getAllIds(state.testConfig.tasks);
    return {
      selection: {
        taskIds: new Set(allIds),
        primaryTaskId: allIds[0] ?? null,
      },
    };
  }),

  // View management
  setActiveView: (view) => {
    const state = get();

    // Sync YAML when switching to YAML view
    if (view === 'yaml' && state.activeView !== 'yaml') {
      const yaml = serializeToYaml(state.testConfig);
      set({ activeView: view, yamlSource: yaml });
    }
    // Parse YAML when switching away from YAML view
    else if (state.activeView === 'yaml' && view !== 'yaml') {
      const result = deserializeFromYaml(state.yamlSource);
      if (result.success && result.config) {
        set({
          activeView: view,
          testConfig: result.config,
          validationErrors: [],
        });
      } else {
        // Keep in YAML view if parse fails
        set({
          validationErrors: [{
            message: result.error || 'Invalid YAML syntax',
            severity: 'error',
          }],
        });
        return; // Don't switch views
      }
    } else {
      set({ activeView: view });
    }
  },

  // YAML operations
  setYamlSource: (yaml) => set({ yamlSource: yaml, isDirty: true }),

  syncToYaml: () => set((state) => ({
    yamlSource: serializeToYaml(state.testConfig),
  })),

  syncFromYaml: () => {
    const state = get();
    const result = deserializeFromYaml(state.yamlSource);

    if (result.success && result.config) {
      set({
        testConfig: result.config,
        validationErrors: [],
        isDirty: true,
      });
      return true;
    } else {
      set({
        validationErrors: [{
          message: result.error || 'Invalid YAML syntax',
          severity: 'error',
        }],
      });
      return false;
    }
  },

  // Load from test details
  loadTest: (testDetails, _descriptors) => {
    const config = convertTestDetailsToBuilderConfig(testDetails);

    set({
      testConfig: config,
      yamlSource: serializeToYaml(config),
      sourceTestId: testDetails.id,
      isDirty: false,
      validationErrors: [],
      selection: { taskIds: new Set(), primaryTaskId: null },
    });
  },

  // Load from YAML string
  loadFromYaml: (yaml) => {
    const result = deserializeFromYaml(yaml);

    if (result.success && result.config) {
      set({
        testConfig: result.config,
        yamlSource: yaml,
        sourceTestId: null,
        isDirty: false,
        validationErrors: [],
        selection: { taskIds: new Set(), primaryTaskId: null },
      });
      return true;
    } else {
      set({
        validationErrors: [{
          message: result.error || 'Invalid YAML syntax',
          severity: 'error',
        }],
      });
      return false;
    }
  },

  // Reset to empty state
  reset: () => set({
    testConfig: createEmptyTestConfig(),
    yamlSource: '',
    sourceTestId: null,
    isDirty: false,
    validationErrors: [],
    selection: { taskIds: new Set(), primaryTaskId: null },
    activeView: 'list',
  }),

  // Export to YAML
  exportYaml: () => {
    const state = get();
    return serializeToYaml(state.testConfig);
  },

  // Validation
  validate: (descriptors) => {
    const state = get();
    const errors: ValidationError[] = [];

    if (!state.testConfig.name || state.testConfig.name.trim() === '') {
      errors.push({
        message: 'Test name is required',
        field: 'name',
        severity: 'error',
      });
    }

    if (!state.testConfig.id || state.testConfig.id.trim() === '') {
      errors.push({
        message: 'Test ID is required for saving',
        field: 'id',
        severity: 'error',
      });
    }

    if (state.testConfig.tasks.length === 0) {
      errors.push({
        message: 'Test must have at least one task',
        severity: 'warning',
      });
    }

    const validateTask = (task: BuilderTask, path: string) => {
      const descriptor = descriptors.get(task.taskType);

      if (!descriptor) {
        errors.push({
          taskId: task.id,
          message: `Unknown task type: ${task.taskType}`,
          severity: 'error',
        });
      }

      if (task.children) {
        task.children.forEach((child, i) => {
          validateTask(child, `${path}[${i}]`);
        });
      }

      if (task.namedChildren) {
        for (const [slotName, child] of Object.entries(task.namedChildren)) {
          validateTask(child, `${path}.${slotName}`);
        }
      }
    };

    state.testConfig.tasks.forEach((task, i) => {
      validateTask(task, `tasks[${i}]`);
    });

    set({ validationErrors: errors });
    return errors;
  },

  clearValidationErrors: () => set({ validationErrors: [] }),
}));

// Helper to format duration in seconds to string
function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  return `${Math.floor(seconds / 3600)}h`;
}

// Raw task structure from API
interface RawTask {
  name: string;
  id?: string;
  title?: string;
  timeout?: string;
  if?: string;
  config?: Record<string, unknown>;
  configVars?: Record<string, string>;
}

// Glue task names that have children in config.tasks
const GLUE_TASKS_WITH_CHILDREN = new Set([
  'run_tasks',
  'run_tasks_concurrent',
]);

// Glue tasks with single child in config.task
const GLUE_TASKS_WITH_SINGLE_CHILD = new Set([
  'run_task_options',
  'run_task_matrix',
]);

// Convert raw task from API to BuilderTask
function convertRawTaskToBuilder(rawTask: RawTask): BuilderTask {
  const config: Record<string, unknown> = { ...(rawTask.config || {}) };
  let children: BuilderTask[] | undefined;
  let namedChildren: Record<string, BuilderTask> | undefined;

  // Extract children from config for glue tasks
  if (GLUE_TASKS_WITH_CHILDREN.has(rawTask.name)) {
    const childTasks = config.tasks as RawTask[] | undefined;
    if (childTasks && Array.isArray(childTasks) && childTasks.length > 0) {
      children = childTasks.map(convertRawTaskToBuilder);
    }
    delete config.tasks;
  } else if (GLUE_TASKS_WITH_SINGLE_CHILD.has(rawTask.name)) {
    const childTask = config.task as RawTask | undefined;
    if (childTask && typeof childTask === 'object') {
      children = [convertRawTaskToBuilder(childTask)];
    }
    delete config.task;
  } else if (hasNamedChildren(rawTask.name)) {
    // Handle tasks with named children (e.g., run_task_background)
    const slots = getNamedChildSlots(rawTask.name);
    if (slots) {
      namedChildren = {};
      for (const slot of slots) {
        const childTask = config[slot.yamlKey] as RawTask | undefined;
        if (childTask && typeof childTask === 'object') {
          namedChildren[slot.name] = convertRawTaskToBuilder(childTask);
        }
        delete config[slot.yamlKey];
      }
      if (Object.keys(namedChildren).length === 0) {
        namedChildren = undefined;
      }
    }
  }

  const task: BuilderTask = {
    id: generateTaskId(),
    taskType: rawTask.name,
    config,
    configVars: rawTask.configVars || {},
  };

  if (rawTask.id) {
    task.taskId = rawTask.id;
  }

  if (rawTask.title) {
    task.title = rawTask.title;
  }

  if (rawTask.timeout) {
    task.timeout = rawTask.timeout;
  }

  if (rawTask.if) {
    task.ifCondition = rawTask.if;
  }

  if (children) {
    task.children = children;
  }

  if (namedChildren) {
    task.namedChildren = namedChildren;
  }

  return task;
}

// Convert test details to BuilderConfig
function convertTestDetailsToBuilderConfig(testDetails: TestDetails): TestConfig {
  const config: TestConfig = {
    id: testDetails.id,
    name: testDetails.name,
    tasks: [],
  };

  if (testDetails.timeout > 0) {
    config.timeout = formatDuration(testDetails.timeout);
  }

  const rawConfig = testDetails.config as {
    tasks?: RawTask[];
    cleanupTasks?: RawTask[];
    config?: Record<string, unknown>;
  };

  if (rawConfig) {
    if (rawConfig.config && typeof rawConfig.config === 'object') {
      config.testVars = rawConfig.config;
    }

    if (rawConfig.tasks && Array.isArray(rawConfig.tasks)) {
      config.tasks = rawConfig.tasks.map(convertRawTaskToBuilder);
    }

    if (rawConfig.cleanupTasks && Array.isArray(rawConfig.cleanupTasks)) {
      config.cleanupTasks = rawConfig.cleanupTasks.map(convertRawTaskToBuilder);
    }
  }

  return config;
}
