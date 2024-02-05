## `check_consensus_finality` Task

### Description
The `check_consensus_finality` task checks the finality status of the consensus chain. Finality in a blockchain context refers to the point where a block's transactions are considered irreversible.

### Configuration Parameters

- **`minUnfinalizedEpochs`**:\
  The minimum number of epochs that are allowed to be not yet finalized.

- **`maxUnfinalizedEpochs`**:\
  The maximum number of epochs that can remain unfinalized before the task fails.

- **`minFinalizedEpochs`**:\
  The minimum number of epochs that must be finalized for the task to be successful.

- **`failOnCheckMiss`**:\
  If set to `true`, the task will stop with a failure result if the finality status does not meet the criteria specified in the other parameters. \
  If `false`, the task will not fail immediately and will continue checking.

### Defaults

These are the default settings for the `check_consensus_finality` task:

```yaml
- name: check_consensus_finality
  config:
    minUnfinalizedEpochs: 0
    maxUnfinalizedEpochs: 0
    minFinalizedEpochs: 0
    failOnCheckMiss: false
```
