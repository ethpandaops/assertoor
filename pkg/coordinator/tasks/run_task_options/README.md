## `run_task_options` Task

### Description
The `run_task_options` task is designed to execute a single task with configurable behaviors and response actions. This flexibility allows for precise control over how the task's outcome is handled and how it interacts with the overall test environment.

### Configuration Parameters

- **`task`**:\
  The task to be executed. This is defined following the standard task definition format.

- **`propagateResult`**:\
  This setting controls how the result of the child task influences the result of the `run_task_options` task. If set to `true`, any change in the result of the child task (success or failure) is immediately reflected in the result of the parent `run_task_options` task. If `false`, the child task's result is only propagated to the parent task after the child task has completed its execution.

- **`exitOnResult`**:\
  If set to `true`, the task will cancel the child task as soon as it sets a result, whether it is "success" or "failure." This option is useful for scenarios where immediate response to the child task's result is necessary.

- **`invertResult`**:\
  When `true`, the result of the child task is inverted. This means the `run_task_options` task will fail if the child task succeeds and succeed if the child task fails. This can be used to validate negative test scenarios.

- **`expectFailure`**:\
  If set to `true`, this option expects the child task to fail. The `run_task_options` task will fail if the child task does not end with a "failure" result, ensuring that failure scenarios are handled as expected.

- **`ignoreFailure`**:\
  When `true`, any failure result from the child task is ignored, and the `run_task_options` task will return a success result instead. This is useful for cases where the child task's failure is an acceptable outcome.

- **`retryOnFailure`**:\
  If set to `true`, the task will retry the execution of the child task if it fails, up to the maximum number of retries specified by `maxRetryCount`.

- **`maxRetryCount`**:\
  The maximum number of times the child task will be retried if it fails and `retryOnFailure` is true. A value of 0 means no retries.

- **`newVariableScope`**:\
  Determines whether to create a new variable scope for the child task. If `false`, the current scope is passed through, allowing the child task to share the same variable context as the `run_task_options` task.

### Defaults

Default settings for the `run_task_options` task:

```yaml
- name: run_task_options
  config:
    task: null
    propagateResult: false
    exitOnResult: false
    invertResult: false
    expectFailure: false
    ignoreFailure: false
    retryOnFailure: false
    maxRetryCount: 0
    newVariableScope: false
```
