## `run_tasks_concurrent` Task

### Description
The `run_tasks_concurrent` task allows for the parallel execution of multiple tasks. This task is crucial in scenarios where tasks need to be run simultaneously, such as in testing environments that require concurrent processes or operations.

### Configuration Parameters

- **`succeedTaskCount`**:\
  The minimum number of child tasks that need to complete with a "success" result for the `run_tasks_concurrent` task to stop and return a success result. A value of 0 indicates that all child tasks need to succeed for the overall task to be considered successful.

- **`failTaskCount`**:\
  The minimum number of child tasks that need to complete with a "failure" result for the `run_tasks_concurrent` task to stop and return a failure result. A value of 1 means the overall task will fail as soon as one child task fails.

- **`failOnUndecided`**:\
  If set to true, the `run_tasks_concurrent` task will fail if neither the `succeedTaskCount` nor the `failTaskCount` is reached.

- **`newVariableScope`**:\
  If set to true, a new variable scope will be created for the child tasks, if not, the child tasks will use the same variable scope as the parent task.

- **`tasks`**:\
  An array of child tasks to be executed concurrently. Each task in this array should be defined according to the standard task structure.

### Defaults

Default settings for the `run_tasks_concurrent` task:

```yaml
- name: run_tasks_concurrent
  config:
    succeedTaskCount: 0
    failTaskCount: 1
    failOnUndecided: false
    newVariableScope: true
    tasks: []
```
