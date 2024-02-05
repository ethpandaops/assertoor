## `run_shell` Task

### Description
The `run_shell` task executes a series of commands within a shell environment. This task is especially useful for complex operations that require executing a shell script or a sequence of commands within a specific shell.


### Configuration Parameters

- **`shell`**:\
  Specifies the type of shell to use for running the commands. Common shells include `sh`, `bash`, `zsh`, etc. The default is `sh`.

- **`envVars`**:\
  A dictionary specifying the environment variables to be used within the shell. Each key in this dictionary represents the name of an environment variable, and its corresponding value indicates the name of a variable from which the actual value should be read. \
  For instance, if `envVars` is set as `{"PATH_VAR": "MY_PATH", "USER_VAR": "MY_USER"}`, the task will use the values of `MY_PATH` and `MY_USER` variables from the current task variable context as the values for `PATH_VAR` and `USER_VAR` within the shell.

- **`command`**:\
  The command or series of commands to be executed in the shell. This can be a single command or a full script. The commands should be provided as a single string, and if multiple commands are needed, they can be separated by the appropriate shell command separator (e.g., newlines or semicolons in `sh` or `bash`).

### Output Handling

The `run_shell` task scans the standard output (stdout) of the shell script for specific triggers that enable actions from within the script. \
For instance, a script can set a variable in the current task context by outputting a formatted string. \
An example of setting a variable `USER_VAR` to "new value" would be:
```bash
echo "::set-var USER_VAR new value"
```
This feature allows the shell script to interact dynamically with the task context, modifying variables based on script execution.

Please note that this feature is still under development and the format of these triggers may change in future versions of the tool. It's important to stay updated with the latest documentation and release notes for any modifications to this functionality.
  

### Defaults

Default settings for the `run_shell` task:

```yaml
- name: run_shell
  config:
    shell: sh
    envVars: {}
    command: ""
```
