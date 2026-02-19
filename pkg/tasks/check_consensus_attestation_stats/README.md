## `check_consensus_attestation_stats` Task

### Description
The `check_consensus_attestation_stats` task is designed to monitor attestation voting statistics on the consensus chain, ensuring that voting patterns align with specified criteria.

#### Task Behavior
- The task monitors attestation statistics over a specified number of epochs.
- By default, the task returns immediately when the attestation criteria are met.
- Use `continueOnPass: true` to keep monitoring even after success.

### Configuration Parameters

- **`minTargetPercent`**:\
  The minimum percentage of correct target votes per checked epoch required for the task to succeed. The range is 0-100%. Default: `0`.

- **`maxTargetPercent`**:\
  The maximum allowable percentage of correct target votes per checked epoch for the task to succeed. The range is 0-100%. Default: `100`.

- **`minHeadPercent`**:\
  The minimum percentage of correct head votes per checked epoch needed for the task to succeed. The range is 0-100%. Default: `0`.

- **`maxHeadPercent`**:\
  The maximum allowable percentage of correct head votes per checked epoch for the task to succeed. The range is 0-100%. Default: `100`.

- **`minTotalPercent`**:\
  The minimum overall voting participation per checked epoch in percent needed for the task to succeed. The range is 0-100%. Default: `0`.

- **`maxTotalPercent`**:\
  The maximum allowable overall voting participation per checked epoch for the task to succeed. The range is 0-100%. Default: `100`.

- **`failOnCheckMiss`**:\
  Determines whether the task should stop with a failure result if a checked epoch does not meet the specified voting ranges. If `false`, the task continues checking subsequent epochs until it succeeds or times out. Default: `false`.

- **`minCheckedEpochs`**:\
  The minimum number of consecutive epochs that must pass the check for the task to succeed. Default: `1`.

- **`continueOnPass`**:\
  If set to `true`, the task continues monitoring attestation statistics even after the criteria are met. This is useful for long-running monitoring scenarios. If `false` (default), the task exits immediately on success.

### Defaults

These are the default settings for the `check_consensus_attestation_stats` task:

```yaml
- name: check_consensus_attestation_stats
  config:
    minTargetPercent: 0
    maxTargetPercent: 100
    minHeadPercent: 0
    maxHeadPercent: 100
    minTotalPercent: 0
    maxTotalPercent: 100
    failOnCheckMiss: false
    minCheckedEpochs: 1
    continueOnPass: false
```

### Outputs

This task does not produce any outputs.
