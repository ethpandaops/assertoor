## `check_consensus_forks` Task

### Description
The `check_consensus_forks` task is designed to check for forks in the consensus layer of the blockchain. Forks occur when there are divergences in the blockchain, leading to two or more competing chains.

#### Task Behavior
- The task monitors for chain forks over a specified number of epochs.
- By default, the task returns immediately when the fork criteria are met for the minimum number of epochs.
- Use `continueOnPass: true` to keep monitoring even after success (useful for detecting late forks).

### Configuration Parameters

- **`minCheckEpochCount`**:\
  The minimum number of epochs to check for forks. Default: `1`.

- **`maxForkDistance`**:\
  The maximum distance allowed before a divergence in the chain is counted as a fork. The distance is measured by the number of blocks between the heads of the forked chains. Default: `1`.

- **`maxForkCount`**:\
  The maximum number of forks that are acceptable. If the number of forks exceeds this limit, the task will complete with a failure result. Default: `0`.

- **`continueOnPass`**:\
  If set to `true`, the task continues monitoring for forks even after the criteria are met. This is useful for detecting forks that may occur later in the test. If `false` (default), the task exits immediately on success.

### Defaults

These are the default settings for the `check_consensus_forks` task:

```yaml
- name: check_consensus_forks
  config:
    minCheckEpochCount: 1
    maxForkDistance: 1
    maxForkCount: 0
    continueOnPass: false
```

### Outputs

- **`forks`**:\
  Array of fork info objects containing head slot, root, and clients on each fork.
