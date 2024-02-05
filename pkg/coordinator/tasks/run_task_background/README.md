## `run_task_background` Task

### Description
The `run_task_background` task facilitates the concurrent execution of a foreground task and a background task, with configurable dependencies and outcomes. This task is essential for simulating complex scenarios involving parallel processes.

### Task Behavior
- The task initiates both the foreground and background tasks simultaneously.
- It continuously monitors the status and result of both tasks.
- Upon completion of the foreground task, the `run_task_background` task also completes, and the background task is cancelled if it's still running.
- The result of the `run_task_background` task mirrors the result of the foreground task. For instance, if the foreground task updates its result to "success", the `run_task_background` task will also set its result to "success".
- The behavior following the completion of the background task is configurable based on the `onBackgroundComplete` setting.


### Configuration Parameters

- **`foregroundTask`**:\
  The task that runs in the foreground. This is the primary task and is defined as per the standard task definition format.

- **`backgroundTask`**:\
  The task that runs in the background concurrently with the foreground task. It is also defined following the standard task definition format.

- **`exitOnForegroundSuccess`**:\
  If set to `true`, the `run_task_background` task will exit with a success result when the foreground task's result is set to "success". Note that this does not necessarily mean the foreground task has completed. If still running, both the background and foreground tasks will be cancelled.

- **`exitOnForegroundFailure`**:\
  If `true`, the task exits with a failure result when the foreground task's result is set to "failure". This does not imply the foreground task's completion. Both the background and foreground tasks will be cancelled if they are still running.

- **`onBackgroundComplete`**:\
  Specifies the action to take when the background task completes. Options are:
  - `ignore`: No action is taken.
  - `fail`: Exits the task with a failure result.
  - `succeed`: Exits the task with a success result.
  - `failOrIgnore`: Exits with a failure result if the background task fails, otherwise no action is taken.

- **`newVariableScope`**:\
  Determines if a new variable scope should be created for the foreground task. If `false`, the current scope is passed through. The background task always operates in a new variable scope, which inherits from the parent but does not propagate changes upwards.

### Defaults

Default settings for the `run_task_background` task:

```yaml
- name: run_task_background
  config:
    foregroundTask: {}
    backgroundTask: {}
    exitOnForegroundSuccess: false
    exitOnForegroundFailure: false
    onBackgroundComplete: "ignore"
    newVariableScope: false
```
