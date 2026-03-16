## `generate_exits` Task

### Description
The `generate_exits` task is designed to create and send voluntary exit transactions to the network. This task is essential for testing how the network handles the process of validators voluntarily exiting from their responsibilities.

### Configuration Parameters

- **`limitPerSlot`**:\
  The maximum number of exit transactions to generate per slot.

- **`limitTotal`**:\
  The total limit on the number of exit transactions that the task will generate.

- **`mnemonic`**:\
  A mnemonic phrase used for generating the validators' keys involved in the exit transactions.

- **`startIndex`**:\
  The starting index within the mnemonic from which to begin generating validator keys. This sets the initial point for key generation.

- **`indexCount`**:\
  The number of validator keys to generate from the mnemonic, determining how many unique exit transactions will be created.

- **`builderExit`**:\
  If set to `true`, generates builder exits instead of validator exits. Builder exits use the `BUILDER_INDEX_FLAG` (2^40) OR'd with the builder index in the `ValidatorIndex` field of the voluntary exit message. The task looks up the pubkey in the shared builder set cache instead of the validator set. Default: `false`.

- **`sendToAllClients`**:\
  If set to `true`, submits the voluntary exit to all ready consensus clients in parallel instead of just one. Useful when not all clients support a particular exit type (e.g. builder exits). The task succeeds if at least one client accepts the exit. Default: `false`.

- **`exitEpoch`**:\
  The exit epoch number set within the exit message. (defaults to head epoch)

- **`clientPattern`**:\
  A regex pattern for selecting specific client endpoints for sending the exit transactions. If left empty, any available endpoint will be used.

- **`excludeClientPattern`**:\
  A regex pattern to exclude certain client endpoints from being used for exit transactions. This parameter adds a layer of control by allowing the exclusion of specific clients, which can be useful for testing under various network scenarios.

- **`awaitInclusion`**:\
  If set to `true`, the task waits for all submitted voluntary exits to be included in beacon blocks before completing. The task monitors new blocks and checks for the presence of the submitted exit operations.

### Outputs

- **`exitedValidators`**:\
  Array of validator indices that were submitted for exit.

- **`includedExits`**:\
  Number of exits confirmed on-chain (only populated when `awaitInclusion` is enabled).

### Defaults

Default settings for the `generate_exits` task:

```yaml
- name: generate_exits
  config:
    limitPerSlot: 0
    limitTotal: 0
    mnemonic: ""
    startIndex: 0
    indexCount: 0
    builderExit: false
    sendToAllClients: false
    exitEpoch: -1
    clientPattern: ""
    excludeClientPattern: ""
    awaitInclusion: false
```
