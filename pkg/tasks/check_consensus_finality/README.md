## `check_consensus_finality` Task

### Description
The `check_consensus_finality` task checks the finality status of the consensus chain. Finality in a blockchain context refers to the point where a block's transactions are considered irreversible.

#### Task Behavior
- The task monitors the finality status of the consensus chain.
- By default, the task returns immediately when the finality criteria are met.
- Use `continueOnPass: true` to keep monitoring even after success (useful when running concurrently with other tasks).

### Configuration Parameters

- **`minUnfinalizedEpochs`**:\
  The minimum number of epochs that are allowed to be not yet finalized. Default: `0`.

- **`maxUnfinalizedEpochs`**:\
  The maximum number of epochs that can remain unfinalized before the task fails. Default: `0`.

- **`minFinalizedEpochs`**:\
  The minimum number of epochs that must be finalized for the task to be successful. Default: `0`.

- **`failOnCheckMiss`**:\
  If set to `true`, the task will stop with a failure result if the finality status does not meet the criteria specified in the other parameters. If `false`, the task will not fail immediately and will continue checking. Default: `false`.

- **`continueOnPass`**:\
  If set to `true`, the task continues monitoring even after the finality criteria are met. This is useful when running concurrently with other tasks where you want to keep checking that finality is maintained. If `false` (default), the task exits immediately on success.

### Defaults

These are the default settings for the `check_consensus_finality` task:

```yaml
- name: check_consensus_finality
  config:
    minUnfinalizedEpochs: 0
    maxUnfinalizedEpochs: 0
    minFinalizedEpochs: 0
    failOnCheckMiss: false
    continueOnPass: false
```

### Outputs

This task does not produce any outputs.
