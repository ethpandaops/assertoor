## `check_consensus_forks` Task

### Description
The `check_consensus_forks` task is designed to check for forks in the consensus layer of the blockchain. Forks occur when there are divergences in the blockchain, leading to two or more competing chains.

### Configuration Parameters

- **`minCheckEpochCount`**:\
  The minimum number of epochs to check for forks. 

- **`maxForkDistance`**:\
  The maximum distance allowed before a divergence in the chain is counted as a fork. \
  The distance is measured by the number of blocks between the heads of the forked chains.

- **`maxForkCount`**:\
  The maximum number of forks that are acceptable. If the number of forks exceeds this limit, the task will complete with a failure result.

### Defaults

These are the default settings for the `check_consensus_forks` task:

```yaml
- name: check_consensus_forks
  config:
    minCheckEpochCount: 1
    maxForkDistance: 1
    maxForkCount: 0
```
