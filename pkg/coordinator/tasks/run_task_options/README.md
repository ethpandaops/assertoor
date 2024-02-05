## `run_task_options` Task

### Description
The `run_task_options` task is designed to execute a single task with configurable behaviors and response actions. This flexibility allows for precise control over how the task's outcome is handled and how it interacts with the overall test environment.

### Configuration Parameters

- **`task`**:\
  The task to be executed. This is defined following the standard task definition format.

- **`exitOnResult`**:\
  If set to `true`, the task will cancel the child task as soon as it sets a result, whether it is "success" or "failure." This option is useful for scenarios where immediate response to the child task's result is necessary.

- **`invertResult`**:\
  When `true`, the result of the child task is inverted. This means the `run_task_options` task will fail if the child task succeeds and succeed if the child task fails. This can be used to validate negative test scenarios.

- **`expectFailure`**:\
  If set to `true`, this option expects the child task to fail. The `run_task_options` task will fail if the child task does not end with a "failure" result, ensuring that failure scenarios are handled as expected.

- **`ignoreFailure`**:\
  When `true`, any failure result from the child task is ignored, and the `run_task_options` task will return a success result instead. This is useful for cases where the child task's failure is an acceptable outcome.

- **`newVariableScope`**:\
  Determines whether to create a new variable scope for the child task. If `false`, the current scope is passed through, allowing the child task to share the same variable context as the `run_task_options` task.

### Defaults

Default settings for the `run_task_options` task:

```yaml
- name: run_task_options
  config:
    task: null
    exitOnResult: false
    invertResult: false
    expectFailure: false
    ignoreFailure: false
    newVariableScope: false
```
