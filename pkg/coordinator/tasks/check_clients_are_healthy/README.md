## `check_clients_are_healthy` Task

### Description
The `check_clients_are_healthy` task is designed to ensure the health of specified clients. It verifies if the clients are reachable and synchronized on the same network.

### Configuration Parameters

- **`clientPattern`**:\
  A regular expression pattern used to specify which clients to check. This allows for targeted health checks of specific clients or groups of clients within the network. A blank pattern targets all clients.

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

- **`maxUnhealthyCount`**:\
  Specifies the maximum number of unhealthy clients allowed before the health check fails. A value of 0 means that any unhealthy client will cause the health check to fail, enforcing strict health criteria.

- **`failOnCheckMiss`**: \
  Determines the task's behavior when a health check fails. If true, the task reports a failure upon the first unsuccessful health check. If false, the task continues to poll the clients until a successful check occurs, allowing for temporary issues to be resolved without immediate failure.

### Defaults

These are the default settings for the `check_clients_are_healthy` task:

```yaml
- name: check_clients_are_healthy
  config:
    clientPattern: ""
    pollInterval: 5s
    skipConsensusCheck: false
    skipExecutionCheck: false
    expectUnhealthy: false
    minClientCount: 0
    maxUnhealthyCount: -1
    failOnCheckMiss: false
```
