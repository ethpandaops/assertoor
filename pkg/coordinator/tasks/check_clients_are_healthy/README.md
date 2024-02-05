## `check_clients_are_healthy` Task

### Description
The `check_clients_are_healthy` task is designed to ensure the health of specified clients. It verifies if the clients are reachable and synchronized on the same network.

### Configuration Parameters

- **`clientNamePatterns`**:\
  An array of endpoint names to be checked. If left empty, the task checks all clients. Use this to target specific clients for health checks.

- **`pollInterval`**:\
  The interval at which the health check is performed. Set this to define how frequently the task should check the clients' health.

- **`skipConsensusCheck`**:\
  A boolean value that, when set to `true`, skips the health check for consensus clients. Useful if you only want to focus on execution clients.

- **`skipExecutionCheck`**:\
  A boolean value that, when set to `true`, skips the health check for execution clients. Use this to exclusively check the health of consensus clients.

- **`expectUnhealthy`**:\
  A boolean value that inverts the expected result of the health check. When `true`, the task succeeds if the clients are not ready or unhealthy. This can be useful in test scenarios where client unavailability is expected or being tested.

- **`minClientCount`**:\
  The minimum number of clients that must match the `clientNamePatterns` and pass the health checks for the task to succeed. A value of 0 indicates that all matching clients need to pass the health check. Use this to set a threshold for the number of healthy clients required by your test scenario.

### Defaults

These are the default settings for the `check_clients_are_healthy` task:

```yaml
- name: check_clients_are_healthy
  config:
    clientNamePatterns: []
    pollInterval: 5s
    skipConsensusCheck: false
    skipExecutionCheck: false
    expectUnhealthy: false
    minClientCount: 0
 ```