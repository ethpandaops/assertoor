## `run_tasks` Task

### Description
The `run_tasks` task executes a series of specified tasks sequentially. This is particularly useful for scenarios where tasks need to be performed in a specific order, with the outcome of one potentially affecting the subsequent ones.

#### Task Behavior
- The task starts the child tasks one after the other in the order they are listed.
- Each child task runs until it completes naturally (returns success or failure).
- After a child task completes, the `run_tasks` task initiates the next task in the sequence.
- By default, the sequence stops if any child task fails. Use `continueOnFailure` to continue despite failures.

### Configuration Parameters

- **`tasks`**:\
  An array of tasks to be executed one after the other. Each task is defined according to the standard task structure.

- **`continueOnFailure`**:\
  When `true`, the sequence of tasks continues even if individual tasks fail, allowing the entire sequence to be executed regardless of individual task outcomes. Default: `false`.

- **`invertResult`**:\
  If set to `true`, the final result is inverted: success becomes failure and failure becomes success. Useful when you expect all tasks to fail. Default: `false`.

- **`ignoreResult`**:\
  If set to `true`, the task always returns success regardless of child task outcomes. Default: `false`.

- **`newVariableScope`**:\
  Determines whether to create a new variable scope for the child tasks. If `false`, the current scope is passed through, allowing the child tasks to share the same variable context as the `run_tasks` task. Default: `false`.

### Defaults

Default settings for the `run_tasks` task:

```yaml
- name: run_tasks
  config:
    tasks: []
    continueOnFailure: false
    invertResult: false
    ignoreResult: false
    newVariableScope: false
```

### Outputs

This task does not produce any outputs.
