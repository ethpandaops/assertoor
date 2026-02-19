## `check_consensus_reorgs` Task

### Description
The `check_consensus_reorgs` task is designed to monitor for reorganizations (reorgs) in the consensus layer of the blockchain. Reorgs occur when the blockchain switches to a different chain due to more blocks being added to it, which can be a normal part of blockchain operation or indicate issues.

#### Task Behavior
- The task monitors for chain reorganizations over a specified number of epochs.
- By default, the task returns immediately when the reorg criteria are met for the minimum number of epochs.
- Use `continueOnPass: true` to keep monitoring even after success (useful for detecting late reorgs).

### Configuration Parameters

- **`minCheckEpochCount`**:\
  The minimum number of epochs to be checked for reorgs. An epoch is a specific period in blockchain time. Default: `1`.

- **`maxReorgDistance`**:\
  The maximum allowable distance for a reorg to occur. This is measured in terms of the number of blocks. Default: `0`.

- **`maxReorgsPerEpoch`**:\
  The maximum number of reorgs allowed within a single epoch. If this number is exceeded, it could indicate unusual activity on the blockchain. Default: `0`.

- **`maxTotalReorgs`**:\
  The total maximum number of reorgs allowed across all checked epochs. Exceeding this number could be a sign of instability in the blockchain. Default: `0`.

- **`continueOnPass`**:\
  If set to `true`, the task continues monitoring for reorgs even after the criteria are met. This is useful for detecting late reorgs during long-running tests. If `false` (default), the task exits immediately on success.

### Defaults

These are the default settings for the `check_consensus_reorgs` task:

```yaml
- name: check_consensus_reorgs
  config:
    minCheckEpochCount: 1
    maxReorgDistance: 0
    maxReorgsPerEpoch: 0
    maxTotalReorgs: 0
    continueOnPass: false
```

### Outputs

This task does not produce any outputs.
