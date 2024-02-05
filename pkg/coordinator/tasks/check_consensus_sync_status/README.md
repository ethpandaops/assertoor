## `check_consensus_sync_status` Task

### Description
The `check_consensus_sync_status` task checks the synchronization status of consensus clients, ensuring they are aligned with the current state of the blockchain network.

### Configuration Parameters

- **`clientNamePatterns`**:\
  Regex patterns for selecting specific consensus clients by name. The default `".*"` targets all clients.

- **`pollInterval`**:\
  The frequency for checking the clients' sync status.

- **`expectSyncing`**:\
  Set to `true` if the clients are expected to be in a syncing state, or `false` if they should be fully synced.

- **`expectOptimistic`**:\
  When `true`, expects clients to be in an optimistic sync state.

- **`expectMinPercent`**:\
  The minimum sync progress percentage required for the task to succeed.

- **`expectMaxPercent`**:\
  The maximum sync progress percentage allowable for the task to succeed.

- **`minSlotHeight`**:\
  The minimum slot height that clients should be synced to.

- **`waitForChainProgression`**:\
  If set to `true`, the task checks for blockchain progression in addition to synchronization status. If `false`, the task solely checks for synchronization status, without waiting for further chain progression.

### Defaults

Default settings for the `check_consensus_sync_status` task:

```yaml
- name: check_consensus_sync_status
  config:
    clientNamePatterns: [".*"]
    pollInterval: 5s
    expectSyncing: false
    expectOptimistic: false
    expectMinPercent: 100
    expectMaxPercent: 100
    minSlotHeight: 10
    waitForChainProgression: false
```
