## `run_tasks` Task

### Description
The `run_tasks` task is designed for executing a series of tasks sequentially, ensuring each task is completed before starting the next. This setup is essential for tests requiring a specific order of task execution.

#### Task Behavior
- The task starts the child tasks one after the other in the order they are listed.
- It continuously monitors the result of the currently running child task. As soon as the child task returns a "success" or "failure" result, the execution of that task is stopped.
- After cancelling the current task, the `run_tasks` task then initiates the next task in the sequence.

An important aspect of this task is that it cancels tasks once they return a result. This is particularly significant for check tasks, which, by their nature, would continue running indefinitely according to their logic. In this sequential setup, however, they are stopped once they achieve a result, allowing the sequence to proceed.

### Configuration Parameters

- **`tasks`**:\
  An array of tasks to be executed one after the other. Each task is defined according to the standard task structure.

- **`expectFailure`**:\
  If set to `true`, this option expects each task in the sequence to fail. The task sequence stops with a "failure" result if any task does not fail as expected.

- **`continueOnFailure`**:\
  When `true`, the sequence of tasks continues even if individual tasks fail, allowing the entire sequence to be executed regardless of individual task outcomes.

### Defaults

Default settings for the `run_tasks` task:

```yaml
- name: run_tasks
  config:
    tasks: []
    expectFailure: false
    continueOnFailure: false
```
