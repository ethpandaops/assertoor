## `run_task_matrix` Task

### Description
The `run_task_matrix` task is designed to execute a specified task multiple times, each with different input values drawn from an array. This task is ideal for scenarios where you need to test a task under various conditions or with different sets of data.

#### Task Behavior
- Creates one instance of the child task for each value in `matrixValues`.
- Tasks can run sequentially (default) or concurrently with `runConcurrent: true`.
- By default, waits for all task instances to complete.
- The result is failure if any instance fails, success if all succeed.

### Configuration Parameters

- **`task`**:\
  The definition of the task to be executed for each matrix value. This task is run repeatedly, once for each value in the `matrixValues` array, with the current value made accessible via the variable named in `matrixVar`.

- **`matrixVar`**:\
  The name of the variable to which the current matrix value is assigned for each child task. This allows the child task to access and use the specific value from the matrix.

- **`matrixValues`**:\
  An array of values that form the matrix. Each value in this array is used to run the child task with a different input.

- **`runConcurrent`**:\
  Determines whether the child tasks should run concurrently or sequentially. If `true`, all tasks run at the same time; if `false` (default), they run one after the other.

- **`successThreshold`**:\
  The number of child tasks that need to succeed for the overall task to be considered successful. A value of `0` (default) means all child tasks must succeed.

- **`failureThreshold`**:\
  The number of child tasks that need to fail before the overall task is considered failed. Default: `1` (any single failure causes overall failure).

- **`stopOnThreshold`**:\
  If set to `true`, remaining child tasks are cancelled when either the success or failure threshold is reached. If `false` (default), the task waits for all children to complete.

- **`invertResult`**:\
  If set to `true`, the final result is inverted: success becomes failure and failure becomes success. Default: `false`.

- **`ignoreResult`**:\
  If set to `true`, the task always returns success regardless of child task outcomes. Default: `false`.

### Defaults

Default settings for the `run_task_matrix` task:

```yaml
- name: run_task_matrix
  config:
    task: {}
    matrixVar: ""
    matrixValues: []
    runConcurrent: false
    successThreshold: 0
    failureThreshold: 1
    stopOnThreshold: false
    invertResult: false
    ignoreResult: false
```

### Outputs

This task does not produce any outputs.
