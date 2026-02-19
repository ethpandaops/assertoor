## `run_tasks_concurrent` Task

### Description
The `run_tasks_concurrent` task allows for the parallel execution of multiple tasks. This task is crucial in scenarios where tasks need to be run simultaneously, such as in testing environments that require concurrent processes or operations.

#### Task Behavior
- All child tasks are started concurrently.
- By default, the task waits for all children to complete.
- The result is failure if any child task fails, success if all succeed.
- Use `stopOnThreshold` to cancel remaining tasks when success/failure threshold is reached.

### Configuration Parameters

- **`tasks`**:\
  An array of child tasks to be executed concurrently. Each task in this array should be defined according to the standard task structure.

- **`successThreshold`**:\
  The minimum number of child tasks that need to complete with a "success" result for the task to be considered successful. A value of `0` (default) means all child tasks must succeed.

- **`failureThreshold`**:\
  The number of child tasks that need to fail before the task is considered failed. Default: `1` (any single failure causes overall failure).

- **`stopOnThreshold`**:\
  If set to `true`, remaining child tasks are cancelled when either the success or failure threshold is reached. If `false` (default), the task waits for all children to complete before determining the result.

- **`invertResult`**:\
  If set to `true`, the final result is inverted: success becomes failure and failure becomes success. Default: `false`.

- **`ignoreResult`**:\
  If set to `true`, the task always returns success regardless of child task outcomes. Default: `false`.

- **`newVariableScope`**:\
  If set to `true`, a new variable scope will be created for each child task. If `false`, the child tasks will use the same variable scope as the parent task. Default: `true`.

### Defaults

Default settings for the `run_tasks_concurrent` task:

```yaml
- name: run_tasks_concurrent
  config:
    tasks: []
    successThreshold: 0
    failureThreshold: 1
    stopOnThreshold: false
    invertResult: false
    ignoreResult: false
    newVariableScope: true
```

### Outputs

This task does not produce any outputs.
