## `sleep` Task

### Description
The `sleep` task is designed to introduce a pause or delay in the execution flow for a specified duration. This task is useful in scenarios where a time-based delay is necessary between operations, such as waiting for certain conditions to be met or simulating real-time interactions.

### Configuration Parameters

- **`duration`**:\
  The length of time for which the task should pause execution. The duration is specified in a time format (e.g., '5s' for five seconds, '1m' for one minute). A duration of '0s' means no delay.

### Defaults

Default settings for the `sleep` task:

```yaml
- name: sleep
  config:
    duration: 0s
```
