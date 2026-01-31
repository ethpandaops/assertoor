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

- **`exitEpoch`**:\
  The exit epoch number set within the exit message. (defaults to head epoch)

- **`clientPattern`**:\
  A regex pattern for selecting specific client endpoints for sending the exit transactions. If left empty, any available endpoint will be used.

- **`excludeClientPattern`**:\
  A regex pattern to exclude certain client endpoints from being used for exit transactions. This parameter adds a layer of control by allowing the exclusion of specific clients, which can be useful for testing under various network scenarios.


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
    exitEpoch: -1
    clientPattern: ""
    excludeClientPattern: ""
```
