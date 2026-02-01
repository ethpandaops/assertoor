## `check_consensus_sync_status` Task

### Description
The `check_consensus_sync_status` task checks the synchronization status of consensus clients, ensuring they are aligned with the current state of the blockchain network.

#### Task Behavior
- The task polls consensus clients at regular intervals to check their sync status.
- By default, the task returns immediately when the sync criteria are met.
- Use `continueOnPass: true` to keep monitoring even after success.

### Configuration Parameters

- **`clientPattern`**:\
  A regular expression pattern used to specify which clients to check. This allows for targeted health checks of specific clients or groups of clients within the network. A blank pattern targets all clients.

- **`pollInterval`**:\
  The frequency for checking the clients' sync status. Default: `5s`.

- **`expectSyncing`**:\
  Set to `true` if the clients are expected to be in a syncing state, or `false` if they should be fully synced. Default: `false`.

- **`expectOptimistic`**:\
  When `true`, expects clients to be in an optimistic sync state. Default: `false`.

- **`expectMinPercent`**:\
  The minimum sync progress percentage required for the task to succeed. Default: `100`.

- **`expectMaxPercent`**:\
  The maximum sync progress percentage allowable for the task to succeed. Default: `100`.

- **`minSlotHeight`**:\
  The minimum slot height that clients should be synced to. Default: `10`.

- **`waitForChainProgression`**:\
  If set to `true`, the task checks for blockchain progression in addition to synchronization status. If `false`, the task solely checks for synchronization status, without waiting for further chain progression. Default: `false`.

- **`continueOnPass`**:\
  If set to `true`, the task continues monitoring sync status even after the criteria are met. This is useful when running concurrently with other tasks. If `false` (default), the task exits immediately on success.

### Defaults

Default settings for the `check_consensus_sync_status` task:

```yaml
- name: check_consensus_sync_status
  config:
    clientPattern: ""
    pollInterval: 5s
    expectSyncing: false
    expectOptimistic: false
    expectMinPercent: 100
    expectMaxPercent: 100
    minSlotHeight: 10
    waitForChainProgression: false
    continueOnPass: false
```

### Outputs

This task does not produce any outputs.
