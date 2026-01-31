## `run_task_matrix` Task

### Description
The `run_task_matrix` task is designed to execute a specified task multiple times, each with different input values drawn from an array. This task is ideal for scenarios where you need to test a task under various conditions or with different sets of data.

### Configuration Parameters

- **`runConcurrent`**:\
  Determines whether the child tasks (instances of the task being run for each matrix value) should run concurrently or sequentially. If `true`, all tasks run at the same time; if `false`, they run one after the other.

- **`succeedTaskCount`**:\
  The number of child tasks that need to succeed (result status "success") for the `run_task_matrix` task to stop and return a success result. A value of 0 means all child tasks need to succeed for the overall task to be considered successful.

- **`failTaskCount`**:\
  The number of child tasks that may to fail (result status "failure") before the `run_task_matrix` task to stops and returns a failure result. A value of 0 means that the appearance of any failure in child tasks will cause the overall task to fail.

- **`failOnUndecided`**:\
  If set to true, the `run_task_matrix` task will fail if neither the `succeedTaskCount` nor the `failTaskCount` is reached.

- **`matrixValues`**:\
  An array of values that form the matrix. Each value in this array is used to run the child task with a different input.

- **`matrixVar`**:\
  The name of the variable to which the current matrix value is assigned for each child task. This allows the child task to access and use the specific value from the matrix.

- **`task`**:\
  The definition of the task to be executed for each matrix value. This task is run repeatedly, once for each value in the `matrixValues` array, with the current value made accessible via the variable named in `matrixVar`.

### Defaults

Default settings for the `run_task_matrix` task:

```yaml
- name: run_task_matrix
  config:
    runConcurrent: false
    succeedTaskCount: 0
    failTaskCount: 0
    failOnUndecided: true
    matrixValues: []
    matrixVar: ""
    task: {}
```

