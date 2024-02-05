## `check_consensus_reorgs` Task

### Description
The `check_consensus_reorgs` task is designed to monitor for reorganizations (reorgs) in the consensus layer of the blockchain. Reorgs occur when the blockchain switches to a different chain due to more blocks being added to it, which can be a normal part of blockchain operation or indicate issues.

### Configuration Parameters

- **`minCheckEpochCount`**:\
  The minimum number of epochs to be checked for reorgs. An epoch is a specific period in blockchain time.

- **`maxReorgDistance`**:\
  The maximum allowable distance for a reorg to occur. This is measured in terms of the number of blocks.

- **`maxReorgsPerEpoch`**:\
  The maximum number of reorgs allowed within a single epoch. If this number is exceeded, it could indicate unusual activity on the blockchain.

- **`maxTotalReorgs`**:\
  The total maximum number of reorgs allowed across all checked epochs. Exceeding this number could be a sign of instability in the blockchain.

### Defaults

These are the default settings for the `check_consensus_reorgs` task:

```yaml
- name: check_consensus_reorgs
  config:
    minCheckEpochCount: 1
    maxReorgDistance: 0
    maxReorgsPerEpoch: 0
    maxTotalReorgs: 0
```
