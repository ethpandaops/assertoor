## `run_command` Task

### Description
The `run_command` task is designed to execute a specified shell command. This task is useful for integrating external scripts or commands into the testing workflow.

### Configuration Parameters

- **`allowed_to_fail`**:
  Determines the task's behavior in response to the command's exit status. If set to `false`, the task will not fail even if the command exits with a failure code. This is useful for commands where a non-zero exit status does not necessarily indicate a critical problem.

- **`command`**:
  The command to be executed, along with its arguments. This should be provided as an array, where the first element is the command and subsequent elements are the arguments. For example, to list files in the current directory with detailed information, you would use `["ls", "-la", "."]`.

### Defaults

Default settings for the `run_command` task:

```yaml
- name: run_command
  config:
    allowed_to_fail: false
    command: []
```
