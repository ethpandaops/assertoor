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

- **`clientPattern`**:\
  A regex pattern for selecting a specific client endpoint for sending the exit transactions. If left empty, any available endpoint will be used.

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
    clientPattern: ""
```
