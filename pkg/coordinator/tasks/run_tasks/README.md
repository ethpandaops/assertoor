## `run_tasks` Task

### Description
The `run_tasks` task executes a series of specified tasks sequentially. This is particularly useful for scenarios where tasks need to be performed in a specific order, with the outcome of one potentially affecting the subsequent ones.

#### Task Behavior
- The task starts the child tasks one after the other in the order they are listed.
- It continuously monitors the result of the currently running child task. As soon as the child task returns a "success" or "failure" result, the execution of that task is stopped.
- After cancelling the current task, the `run_tasks` task then initiates the next task in the sequence.

An important aspect of this task is that it cancels tasks once they return a result. This is particularly significant for check tasks, which, by their nature, would continue running indefinitely according to their logic. In this sequential setup, however, they are stopped once they achieve a result, allowing the sequence to proceed.

### Configuration Parameters

- **`tasks`**:\
  An array of tasks to be executed one after the other. Each task is defined according to the standard task structure.

- **`stopChildOnResult`**:\
  If set to `true`, each child task in the sequence is stopped as soon as it sets a result (either "success" or "failure"). This ensures that once a task has reached a outcome, it does not continue to run unnecessarily, allowing the next task in the sequence to commence.

- **`expectFailure`**:\
  If set to `true`, this option expects each task in the sequence to fail. The task sequence stops with a "failure" result if any task does not fail as expected.

- **`continueOnFailure`**:\
  When `true`, the sequence of tasks continues even if individual tasks fail, allowing the entire sequence to be executed regardless of individual task outcomes.

- **`newVariableScope`**:\
  Determines whether to create a new variable scope for the child tasks. If `false`, the current scope is passed through, allowing the child tasks to share the same variable context as the `run_tasks` task.

### Defaults

Default settings for the `run_tasks` task:

```yaml
- name: run_tasks
  config:
    tasks: []
    stopChildOnResult: true
    expectFailure: false
    continueOnFailure: false
    newVariableScope: false
```
