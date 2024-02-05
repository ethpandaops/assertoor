## `check_execution_sync_status` Task

### Description
The `check_execution_sync_status` task checks the synchronization status of execution clients in the blockchain network. It ensures that these clients are syncing correctly with the network's current state.

### Configuration Parameters

- **`clientPattern`**:\
  A regular expression pattern used to specify which clients to check. This allows for targeted health checks of specific clients or groups of clients within the network. A blank pattern targets all clients.

- **`pollInterval`**:\
  The interval at which the task checks the clients' sync status. This defines the frequency of the synchronization checks.

- **`expectSyncing`**:\
  Set this to `true` if the clients are expected to be in a syncing state. If `false`, the task expects the clients to be fully synced.

- **`expectMinPercent`**:\
  The minimum expected percentage of synchronization. Clients should be synced at least to this level for the task to succeed.

- **`expectMaxPercent`**:\
  The maximum allowable percentage of synchronization. Clients should not be synced beyond this level for the task to pass.

- **`minBlockHeight`**:\
  The minimum block height that the clients should be synced to. This sets a specific block height requirement for the task.

- **`waitForChainProgression`**:\
  If `true`, the task checks for blockchain progression in addition to the synchronization status. If `false`, it only checks for synchronization without waiting for further chain progression.

### Defaults

These are the default settings for the `check_execution_sync_status` task:

```yaml
- name: check_execution_sync_status
  config:
    clientPattern: ""
    pollInterval: 5s
    expectSyncing: false
    expectMinPercent: 100
    expectMaxPercent: 100
    minBlockHeight: 10
    waitForChainProgression: false
```
