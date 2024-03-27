## `check_execution_conversion_state` Task

### Description
The `check_execution_conversion_state` task is designed to monitor the status of execution clients regarding their conversion to the Verkle tree structure, a significant upgrade in the Ethereum network. This task assesses whether the execution clients have started, are in the process of, or have completed the conversion to Verkle trees, ensuring that the network's upgrade transitions are proceeding as expected.

### Configuration Parameters

- **`clientPattern`**:
  A regex pattern for selecting specific execution client endpoints to check. This allows for targeted monitoring of clients based on identifiers or characteristics defined in their endpoint URLs.

- **`pollInterval`**:
  The time interval, in seconds, at which the task will poll the clients to check their Verkle conversion status. A shorter interval results in more frequent checks, allowing for timely detection of state changes.

- **`expectStarted`**:
  If set to `true`, this option indicates the expectation that the Verkle conversion process has started on the targeted execution clients. The task checks for evidence that the conversion process is underway.

- **`expectFinished`**:
  When `true`, the task expects that the Verkle conversion process has been completed on the targeted execution clients. It verifies that the clients are fully upgraded to the new tree structure.

- **`failOnUnexpected`**:
  If set to `true`, the task will fail if the actual conversion status of the clients does not match the expected states (`expectStarted`, `expectFinished`). This is useful for scenarios where strict compliance with the conversion timeline is critical.

### Defaults

Default settings for the `check_execution_conversion_state` task:

```yaml
- name: check_execution_conversion_state
  config:
    clientPattern: ""
    pollInterval: 10s
    expectStarted: false
    expectFinished: false
    failOnUnexpected: false
```
