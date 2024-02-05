## `generate_deposits` Task

### Description
The `generate_deposits` task focuses on creating deposit transactions and sending them to the network. This task is crucial for testing how the network handles new deposits.

### Configuration Parameters

- **`limitPerSlot`**:\
  The maximum number of deposit transactions to be generated for each slot.

- **`limitTotal`**:\
  The total limit on the number of deposit transactions that this task will generate.

- **`mnemonic`**:\
  A mnemonic phrase used to generate validator keys. These keys are essential for creating valid deposit transactions.

- **`startIndex`**:\
  The starting index within the mnemonic for generating validator keys. This defines the beginning point for the key generation process.

- **`indexCount`**:\
  The total number of validator keys to generate from the mnemonic. This number determines how many unique deposit transactions will be created.

- **`walletPrivkey`**:\
  The private key of the wallet from which the deposit will be made. This key is crucial for initiating the deposit transaction.

- **`depositContract`**:\
  The address of the deposit contract on the blockchain. This is the destination where the deposit transactions will be sent.

- **`depositTxFeeCap`**:\
  The maximum fee cap for each deposit transaction. This limits the transaction fees for deposit operations.

- **`depositTxTipCap`**:\
  The maximum tip cap for each deposit transaction. This controls the tip or priority fee for each transaction.

- **`clientPattern`**:\
  A regex pattern to select specific client endpoints for sending deposit transactions. If left blank, any available endpoint will be used.

- **`excludeClientPattern`**:\
  A regex pattern to exclude certain clients from being used for deposit transactions. This parameter adds an extra layer of control over client selection.


### Defaults

Default settings for the `generate_deposits` task:

```yaml
- name: generate_deposits
  config:
    limitPerSlot: 0
    limitTotal: 0
    mnemonic: ""
    startIndex: 0
    indexCount: 0
    walletPrivkey: ""
    depositContract: ""
    depositTxFeeCap: 100000000000
    depositTxTipCap: 1000000000
    clientPattern: ""
    excludeClientPattern: ""
```
