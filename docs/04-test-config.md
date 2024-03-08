# Test Configuration

Assertoor allows you to set up tests to check various aspects of the Ethereum network. You can organize these tests in two ways: directly within your main configuration file or through external files for more complex scenarios. Here’s how it works:

## Local Tests

Local tests are defined in your main Assertoor configuration file. You can list tasks that you want to run as part of your test, along with any cleanup tasks to run afterward, regardless of whether your test passes or fails.

```yaml
- id: "test1"
  name: "Test with local tasks"
  timeout: "48h"
  config: {}
  tasks: []
  cleanupTasks: []
  schedule:
    startup: true
    cron:
      - "* * * * *"
 ```

- **`id`**:\
  A unique identifier for your test.
- **`name`**:\
  The test's name.
- **`timeout`**:\
  How long to run the test before stopping it. This is optional.
- **`config`**:\
  A place to set static variables for your test. Optional.
- **`tasks`**:\
  The tasks to run for your test.
- **`cleanupTasks`**:\
  Tasks that clean up after your main tasks. These run no matter if the main tasks pass or fail. This is optional.
- **`schedule`**:\
  Determines when your test runs. You can set it to start when Assertoor starts or on a schedule using cron format. This is optional.

## External Tests

External test playbooks, loaded via the `file` attribute in the Assertoor configuration, follow a structured format. These playbooks allow for defining comprehensive tests with specific tasks, variable configurations, and scheduling options.

**Example External Test Playbook:**

```yaml
id: test1
name: "Test 1"
timeout: 1h
config:
  # walletPrivkey: ""
  # validatorPairNames: []
configVars: {}
tasks: []
cleanupTasks: []
schedule:
  startup: true
  cron:
    - "* * * * *"
```

**Key Properties Explained:**

- **`id`**: A unique identifier for the test, allowing for easy reference.
- **`name`**: The descriptive name of the test, providing clarity on its purpose.
- **`timeout`**: Specifies the duration after which the test should be considered failed if not completed.
- **`config`**: Static variable configuration, where you can define variables directly used by the test.
- **`configVars`**: Dynamic variable configuration that copies variables from the global scope, supporting complex expressions through jq syntax.
- **`tasks`**: The list of tasks to be executed as part of the test. Refer to the task configuration section for detailed task structures.
- **`cleanupTasks`**: Specifies tasks to be executed after the main tasks, regardless of their success or failure.
- **`schedule`**: Determines when the test should be run. If omitted, the test is scheduled to start upon Assertoor startup. It also supports cron expressions for more precise scheduling.

This format provides a flexible and powerful way to define tests outside the main configuration file, allowing for modular test management and reusability across different scenarios or environments.

