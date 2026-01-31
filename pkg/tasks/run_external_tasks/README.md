## `run_external_tasks` Task

### Description
The `run_external_tasks` task is designed to execute a set of tasks specified in an external test playbook. This functionality is key for integrating modular and reusable test configurations that are maintained outside the primary Assertoor configuration file.

### Configuration Parameters

- **`testFile`**:\
  The path to the external test playbook file. This file contains the configuration for the tasks to be executed, including task definitions and any specific settings required for those tasks.

- **`testConfig`**:\
  A dictionary of static configuration parameters that are passed to the external test playbook. These configurations are used to customize or override settings within the external test playbook.

- **`testConfigVars`**:\
  A dictionary of dynamic variable expressions that are evaluated and also passed to the external test playbook.

- **`expectFailure`**:\
  A boolean that specifies whether the task is expected to fail. If set to `true`, the task will be considered successful if the external tasks fail.

- **`ignoreFailure`**:\
  A boolean that indicates whether failures in the external tasks should be ignored. If `true`, the `run_external_tasks` task will report success regardless of any failures that occur during the execution of the external tasks. This can be useful when the outcomes of the tasks are not critical to the overall test objectives.

### Defaults

Default settings for the `run_external_tasks` task:

```yaml
- name: run_external_tasks
  config:
    testFile: ""
    testConfig: {}
    testConfigVars: {}
    expectFailure: false
    ignoreFailure: false
```
