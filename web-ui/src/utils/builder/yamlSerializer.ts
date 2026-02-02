import yaml from 'js-yaml';
import type { BuilderTask, TestConfig } from '../../stores/builderStore';
import { generateTaskId } from './taskUtils';

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

// Glue task with background/foreground children
const BACKGROUND_GLUE_TASK = 'run_task_background';

// Check if a task type is a glue task
export function isGlueTask(taskType: string): boolean {
  return GLUE_TASKS_WITH_CHILDREN.has(taskType) ||
    GLUE_TASKS_WITH_SINGLE_CHILD.has(taskType) ||
    taskType === BACKGROUND_GLUE_TASK;
}

// Raw YAML task structure
interface YamlTask {
  name: string;
  id?: string;
  title?: string;
  timeout?: string;
  if?: string;
  config?: Record<string, unknown>;
  configVars?: Record<string, string>;
}

// Raw YAML test structure
interface YamlTest {
  id?: string;
  name: string;
  timeout?: string;
  config?: Record<string, unknown>;
  configVars?: Record<string, string>;
  tasks?: YamlTask[];
  cleanupTasks?: YamlTask[];
}

// Convert BuilderTask to YAML task structure
function builderTaskToYaml(task: BuilderTask): YamlTask {
  const yamlTask: YamlTask = {
    name: task.taskType,
  };

  if (task.taskId) {
    yamlTask.id = task.taskId;
  }

  if (task.title) {
    yamlTask.title = task.title;
  }

  if (task.timeout) {
    yamlTask.timeout = task.timeout;
  }

  if (task.ifCondition) {
    yamlTask.if = task.ifCondition;
  }

  // Start with existing config
  const config: Record<string, unknown> = { ...task.config };

  // Handle children for glue tasks - they go into config
  if (task.children && task.children.length > 0) {
    if (GLUE_TASKS_WITH_CHILDREN.has(task.taskType)) {
      // Children go into config.tasks
      config.tasks = task.children.map(builderTaskToYaml);
    } else if (GLUE_TASKS_WITH_SINGLE_CHILD.has(task.taskType)) {
      // Single child goes into config.task
      config.task = builderTaskToYaml(task.children[0]);
    } else if (task.taskType === BACKGROUND_GLUE_TASK) {
      // Background task has backgroundTask and foregroundTask in config
      if (task.children.length >= 1) {
        config.backgroundTask = builderTaskToYaml(task.children[0]);
      }
      if (task.children.length >= 2) {
        config.foregroundTask = builderTaskToYaml(task.children[1]);
      }
    }
  }

  // Add config if not empty
  if (Object.keys(config).length > 0) {
    yamlTask.config = config;
  }

  // Add configVars if not empty
  if (task.configVars && Object.keys(task.configVars).length > 0) {
    yamlTask.configVars = task.configVars;
  }

  return yamlTask;
}

// Convert YAML task structure to BuilderTask
function yamlTaskToBuilder(yamlTask: YamlTask): BuilderTask {
  const builderTask: BuilderTask = {
    id: generateTaskId(),
    taskType: yamlTask.name,
    config: {},
    configVars: yamlTask.configVars || {},
  };

  if (yamlTask.id) {
    builderTask.taskId = yamlTask.id;
  }

  if (yamlTask.title) {
    builderTask.title = yamlTask.title;
  }

  if (yamlTask.timeout) {
    builderTask.timeout = yamlTask.timeout;
  }

  if (yamlTask.if) {
    builderTask.ifCondition = yamlTask.if;
  }

  // Handle config - extract children for glue tasks
  if (yamlTask.config) {
    const config = { ...yamlTask.config };

    // Extract children from config for glue tasks
    if (GLUE_TASKS_WITH_CHILDREN.has(yamlTask.name)) {
      const childTasks = config.tasks as YamlTask[] | undefined;
      if (childTasks && Array.isArray(childTasks) && childTasks.length > 0) {
        builderTask.children = childTasks.map(yamlTaskToBuilder);
      }
      delete config.tasks;
    } else if (GLUE_TASKS_WITH_SINGLE_CHILD.has(yamlTask.name)) {
      const childTask = config.task as YamlTask | undefined;
      if (childTask && typeof childTask === 'object') {
        builderTask.children = [yamlTaskToBuilder(childTask)];
      }
      delete config.task;
    } else if (yamlTask.name === BACKGROUND_GLUE_TASK) {
      const children: BuilderTask[] = [];
      const bgTask = config.backgroundTask as YamlTask | undefined;
      const fgTask = config.foregroundTask as YamlTask | undefined;

      if (bgTask && typeof bgTask === 'object') {
        children.push(yamlTaskToBuilder(bgTask));
      }
      if (fgTask && typeof fgTask === 'object') {
        children.push(yamlTaskToBuilder(fgTask));
      }

      if (children.length > 0) {
        builderTask.children = children;
      }

      delete config.backgroundTask;
      delete config.foregroundTask;
    }

    // Store remaining config
    builderTask.config = config;
  }

  return builderTask;
}

// Serialize TestConfig to YAML string
export function serializeToYaml(config: TestConfig): string {
  const yamlConfig: YamlTest = {
    name: config.name,
  };

  if (config.id) {
    yamlConfig.id = config.id;
  }

  if (config.timeout) {
    yamlConfig.timeout = config.timeout;
  }

  if (config.testVars && Object.keys(config.testVars).length > 0) {
    yamlConfig.config = config.testVars;
  }

  if (config.tasks && config.tasks.length > 0) {
    yamlConfig.tasks = config.tasks.map(builderTaskToYaml);
  }

  if (config.cleanupTasks && config.cleanupTasks.length > 0) {
    yamlConfig.cleanupTasks = config.cleanupTasks.map(builderTaskToYaml);
  }

  return yaml.dump(yamlConfig, {
    indent: 2,
    lineWidth: -1,
    noRefs: true,
    sortKeys: false,
  });
}

// Result of deserialization
export interface DeserializeResult {
  success: boolean;
  config?: TestConfig;
  error?: string;
}

// Deserialize YAML string to TestConfig
export function deserializeFromYaml(yamlString: string): DeserializeResult {
  if (!yamlString || yamlString.trim() === '') {
    return {
      success: true,
      config: {
        name: 'New Test',
        tasks: [],
      },
    };
  }

  try {
    const parsed = yaml.load(yamlString) as YamlTest;

    if (!parsed || typeof parsed !== 'object') {
      return {
        success: false,
        error: 'YAML must be an object',
      };
    }

    const config: TestConfig = {
      name: parsed.name || 'Untitled Test',
      tasks: [],
    };

    if (parsed.id) {
      config.id = parsed.id;
    }

    if (parsed.timeout) {
      config.timeout = parsed.timeout;
    }

    if (parsed.config && typeof parsed.config === 'object') {
      config.testVars = parsed.config as Record<string, unknown>;
    }

    // Also check for configVars at root level
    if (parsed.configVars && typeof parsed.configVars === 'object') {
      // Merge into testVars
      config.testVars = {
        ...(config.testVars || {}),
        ...parsed.configVars,
      };
    }

    if (parsed.tasks && Array.isArray(parsed.tasks)) {
      config.tasks = parsed.tasks.map(yamlTaskToBuilder);
    }

    if (parsed.cleanupTasks && Array.isArray(parsed.cleanupTasks)) {
      config.cleanupTasks = parsed.cleanupTasks.map(yamlTaskToBuilder);
    }

    return {
      success: true,
      config,
    };
  } catch (err) {
    return {
      success: false,
      error: err instanceof Error ? err.message : 'Failed to parse YAML',
    };
  }
}

// Validate YAML syntax without full deserialization
export function validateYamlSyntax(yamlString: string): { valid: boolean; error?: string } {
  if (!yamlString || yamlString.trim() === '') {
    return { valid: true };
  }

  try {
    yaml.load(yamlString);
    return { valid: true };
  } catch (err) {
    return {
      valid: false,
      error: err instanceof Error ? err.message : 'Invalid YAML syntax',
    };
  }
}

// Format YAML with consistent styling
export function formatYaml(yamlString: string): string {
  try {
    const parsed = yaml.load(yamlString);
    return yaml.dump(parsed, {
      indent: 2,
      lineWidth: -1,
      noRefs: true,
      sortKeys: false,
    });
  } catch {
    return yamlString;
  }
}
