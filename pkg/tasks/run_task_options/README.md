## `run_task_options` Task

### Description
The `run_task_options` task is designed to execute a single task with configurable behaviors and response actions. This flexibility allows for precise control over how the task's outcome is handled and how it interacts with the overall test environment.

#### Task Behavior
- Executes the child task and waits for it to complete naturally.
- Supports retry logic for failed tasks.
- Can transform the result (invert, ignore, expect failure).

### Configuration Parameters

- **`task`**:\
  The task to be executed. This is defined following the standard task definition format.

- **`retryOnFailure`**:\
  If set to `true`, the task will retry the execution of the child task if it fails, up to the maximum number of retries specified by `maxRetryCount`. Default: `false`.

- **`maxRetryCount`**:\
  The maximum number of times the child task will be retried if it fails and `retryOnFailure` is true. Default: `0` (no retries).

- **`invertResult`**:\
  When `true`, the result of the child task is inverted. This means the `run_task_options` task will fail if the child task succeeds and succeed if the child task fails. This can be used to validate negative test scenarios. Default: `false`.

- **`expectFailure`**:\
  Alias for `invertResult`. If set to `true`, this option expects the child task to fail. The `run_task_options` task will fail if the child task does not end with a "failure" result. Default: `false`.

- **`ignoreResult`**:\
  When `true`, any failure result from the child task is ignored, and the `run_task_options` task will return a success result instead. This is useful for cases where the child task's failure is an acceptable outcome. Default: `false`.

- **`newVariableScope`**:\
  Determines whether to create a new variable scope for the child task. If `false`, the current scope is passed through, allowing the child task to share the same variable context as the `run_task_options` task. Default: `false`.

### Defaults

Default settings for the `run_task_options` task:

```yaml
- name: run_task_options
  config:
    task: null
    retryOnFailure: false
    maxRetryCount: 0
    invertResult: false
    expectFailure: false
    ignoreResult: false
    newVariableScope: false
```

### Outputs

This task does not produce any outputs.
