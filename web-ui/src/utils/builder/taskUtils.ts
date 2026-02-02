import type { BuilderTask } from '../../stores/builderStore';

// Generate a unique task ID
let idCounter = 0;
export function generateTaskId(): string {
  idCounter++;
  return `task_${Date.now()}_${idCounter}_${Math.random().toString(36).substring(2, 9)}`;
}

// Reset ID counter (useful for testing)
export function resetIdCounter(): void {
  idCounter = 0;
}

// Find a task by ID in a task tree
export function findTaskById(tasks: BuilderTask[], taskId: string): BuilderTask | null {
  for (const task of tasks) {
    if (task.id === taskId) {
      return task;
    }
    if (task.children) {
      const found = findTaskById(task.children, taskId);
      if (found) return found;
    }
  }
  return null;
}

// Find the parent of a task
export function findParentTask(
  tasks: BuilderTask[],
  taskId: string,
  parent: BuilderTask | null = null
): BuilderTask | null {
  for (const task of tasks) {
    if (task.id === taskId) {
      return parent;
    }
    if (task.children) {
      const found = findParentTask(task.children, taskId, task);
      if (found !== null || task.children.some((c) => c.id === taskId)) {
        return found !== null ? found : task;
      }
    }
  }
  return null;
}

// Find the path to a task (array of parent IDs)
export function findTaskPath(tasks: BuilderTask[], taskId: string, path: string[] = []): string[] | null {
  for (const task of tasks) {
    if (task.id === taskId) {
      return path;
    }
    if (task.children) {
      const found = findTaskPath(task.children, taskId, [...path, task.id]);
      if (found) return found;
    }
  }
  return null;
}

// Get all task IDs in the tree
export function getAllTaskIds(tasks: BuilderTask[]): string[] {
  const ids: string[] = [];
  for (const task of tasks) {
    ids.push(task.id);
    if (task.children) {
      ids.push(...getAllTaskIds(task.children));
    }
  }
  return ids;
}

// Get all tasks in the tree (flattened)
export function getAllTasks(tasks: BuilderTask[]): BuilderTask[] {
  const result: BuilderTask[] = [];
  for (const task of tasks) {
    result.push(task);
    if (task.children) {
      result.push(...getAllTasks(task.children));
    }
  }
  return result;
}

// Find tasks that precede a given task (for variable context)
export function findPrecedingTasks(
  tasks: BuilderTask[],
  taskId: string,
  includeParents = true
): BuilderTask[] {
  const preceding: BuilderTask[] = [];

  function collectPreceding(taskList: BuilderTask[], targetId: string): boolean {
    for (let i = 0; i < taskList.length; i++) {
      const task = taskList[i];

      if (task.id === targetId) {
        // Found the target, stop here
        return true;
      }

      // Check if target is in children
      if (task.children) {
        const foundInChildren = collectPreceding(task.children, targetId);
        if (foundInChildren) {
          // All siblings before this parent are preceding
          for (let j = 0; j < i; j++) {
            addAllTasks(taskList[j]);
          }
          // If including parents, add the parent too
          if (includeParents) {
            preceding.push(task);
          }
          return true;
        }
      }

      // If we haven't found target yet, this task precedes it
      // (will be added if we find target later)
    }

    return false;
  }

  function addAllTasks(task: BuilderTask): void {
    preceding.push(task);
    if (task.children) {
      for (const child of task.children) {
        addAllTasks(child);
      }
    }
  }

  // Collect all tasks before the target in execution order
  for (let i = 0; i < tasks.length; i++) {
    if (tasks[i].id === taskId) {
      // Found at root level, all previous root tasks precede
      for (let j = 0; j < i; j++) {
        addAllTasks(tasks[j]);
      }
      return preceding;
    }

    if (tasks[i].children) {
      const foundInChildren = collectPreceding(tasks[i].children!, taskId);
      if (foundInChildren) {
        // All tasks before this root task precede
        for (let j = 0; j < i; j++) {
          addAllTasks(tasks[j]);
        }
        return preceding;
      }
    }
  }

  return preceding;
}

// Remove a task by ID (returns new array)
export function removeTaskById(tasks: BuilderTask[], taskId: string): BuilderTask[] {
  return tasks
    .filter((task) => task.id !== taskId)
    .map((task) => {
      if (task.children) {
        return { ...task, children: removeTaskById(task.children, taskId) };
      }
      return task;
    });
}

// Insert a task at a specific position
export function insertTaskAt(
  tasks: BuilderTask[],
  newTask: BuilderTask,
  parentId: string,
  index?: number
): BuilderTask[] {
  return tasks.map((task) => {
    if (task.id === parentId) {
      const children = [...(task.children || [])];
      if (index !== undefined && index >= 0 && index <= children.length) {
        children.splice(index, 0, newTask);
      } else {
        children.push(newTask);
      }
      return { ...task, children };
    }
    if (task.children) {
      return { ...task, children: insertTaskAt(task.children, newTask, parentId, index) };
    }
    return task;
  });
}

// Move a task to a new position
export function moveTaskTo(
  tasks: BuilderTask[],
  task: BuilderTask,
  targetParentId: string | undefined,
  targetIndex: number
): BuilderTask[] {
  if (!targetParentId) {
    // Moving to root level
    const result = [...tasks];
    if (targetIndex >= 0 && targetIndex <= result.length) {
      result.splice(targetIndex, 0, task);
    } else {
      result.push(task);
    }
    return result;
  }

  // Moving into a parent
  return insertTaskAt(tasks, task, targetParentId, targetIndex);
}

// Check if a task is a descendant of another
export function isDescendantOf(tasks: BuilderTask[], taskId: string, potentialAncestorId: string): boolean {
  const path = findTaskPath(tasks, taskId);
  return path !== null && path.includes(potentialAncestorId);
}

// Get task index within its parent (or root)
export function getTaskIndex(tasks: BuilderTask[], taskId: string): { parentId: string | null; index: number } | null {
  // Check root level
  for (let i = 0; i < tasks.length; i++) {
    if (tasks[i].id === taskId) {
      return { parentId: null, index: i };
    }
  }

  // Check children
  for (const task of tasks) {
    if (task.children) {
      for (let i = 0; i < task.children.length; i++) {
        if (task.children[i].id === taskId) {
          return { parentId: task.id, index: i };
        }
      }
      const found = getTaskIndex(task.children, taskId);
      if (found) return found;
    }
  }

  return null;
}

// Count total tasks in tree
export function countTasks(tasks: BuilderTask[]): number {
  let count = 0;
  for (const task of tasks) {
    count++;
    if (task.children) {
      count += countTasks(task.children);
    }
  }
  return count;
}

// Get maximum nesting depth
export function getMaxDepth(tasks: BuilderTask[], currentDepth = 0): number {
  let maxDepth = currentDepth;
  for (const task of tasks) {
    if (task.children && task.children.length > 0) {
      const childDepth = getMaxDepth(task.children, currentDepth + 1);
      maxDepth = Math.max(maxDepth, childDepth);
    }
  }
  return maxDepth;
}

// Check if a task type can have children (is a glue task)
export function canHaveChildren(taskType: string): boolean {
  const GLUE_TASKS = new Set([
    'run_tasks',
    'run_tasks_concurrent',
    'run_task_matrix',
    'run_task_options',
    'run_task_background',
  ]);
  return GLUE_TASKS.has(taskType);
}

// Check if moving task would create a circular reference
export function wouldCreateCircular(
  tasks: BuilderTask[],
  taskId: string,
  targetParentId: string | null
): boolean {
  if (targetParentId === null) return false;
  if (taskId === targetParentId) return true;
  return isDescendantOf(tasks, targetParentId, taskId);
}

// Deep clone a task tree
export function cloneTaskTree(tasks: BuilderTask[]): BuilderTask[] {
  return tasks.map((task) => ({
    ...task,
    config: { ...task.config },
    configVars: { ...task.configVars },
    children: task.children ? cloneTaskTree(task.children) : undefined,
  }));
}

// Create a new task from a task type
export function createTask(taskType: string, title?: string): BuilderTask {
  const task: BuilderTask = {
    id: generateTaskId(),
    taskType,
    config: {},
    configVars: {},
  };

  if (title) {
    task.title = title;
  }

  // Initialize children array for glue tasks
  if (canHaveChildren(taskType)) {
    task.children = [];
  }

  return task;
}
