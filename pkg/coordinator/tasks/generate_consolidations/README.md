## `generate_consolidations` Task

### Description
The `generate_consolidations` task is designed to create and send consolidation transactions to the Ethereum network. Consolidations are specialized transactions used for consolidating multiple validator balances into a single validator, optimizing the management and operation of validators within the network.

The source validators can be specified in two ways:
- By providing `sourceMnemonic`, `sourceStartIndex` & `sourceIndexCount` to select the source validators by the pubkeys derived from the mnemonic & key range
- By providing `sourceStartValidatorIndex` & `sourceIndexCount` to select the source validators by their validator index

### Configuration Parameters

- **`limitPerSlot`**:
  Specifies the maximum number of consolidation transactions allowed per slot.

- **`limitTotal`**:
  Sets the total allowable number of consolidation transactions that the task can generate.

- **`limitPending`**:
  Defines the maximum number of pending consolidation transactions allowed at any given time.

- **`sourceMnemonic`**:
  The mnemonic used to derive source validator keys; these validators are getting consolidated into the target validator.

- **`sourceStartIndex`**:
  The starting index for key derivation from the source mnemonic, identifying the first source validator in the consolidation process.

- **`sourceStartValidatorIndex`**:
  The exact starting validator index from which to begin consolidation, providing precise control over the selection of source validators.

- **`sourceIndexCount`**:
  The number of validators to include in the consolidation process from the source mnemonic.

- **`targetValidatorIndex`**:
  The index of the target validator to which all consolidated funds will be transferred.

- **`consolidationEpoch`**:
  The specific blockchain epoch during which the consolidations are to be executed, aligning the transactions with defined blockchain timings.

- **`walletPrivkey`**:
  The private key of the wallet initiating the consolidation transactions, necessary for transaction authorization.
  This wallet must be set as withdrawal address for the source & target validator for successful consolidation.

- **`consolidationContract`**:
  The address of the contract on the blockchain that handles the consolidation operations.

- **`txAmount`**:
  The amount of ETH to be sent to the consolidation contract in each transaction (for consolidation fees).

- **`txFeeCap`**:
  The maximum fee cap for each transaction, controlling the cost associated with the consolidation.

- **`txTipCap`**:
  The tip cap for each transaction, influencing transaction priority.

- **`txGasLimit`**:
  The gas limit for each transaction, ensuring transactions are executed within the cost constraints.

- **`clientPattern`**:
  A regex pattern to select specific clients for sending the transactions, targeting appropriate network nodes.

- **`excludeClientPattern`**:
  A regex pattern to exclude certain clients from sending transactions, optimizing network interactions.

- **`awaitReceipt`**:
  When enabled, the task waits for a receipt for each transaction, confirming execution on the network.

- **`failOnReject`**:
  Determines if the task should fail upon transaction rejection, enhancing error handling.

- **`consolidationTransactionsResultVar`**:
  A variable where the hashes of the executed consolidation transactions are stored.

- **`consolidationReceiptsResultVar`**:
  A variable for storing receipts of the consolidation transactions, applicable when receipts are awaited.

### Defaults

Default settings for the `generate_consolidations` task:

```yaml
- name: generate_consolidations
  config:
    limitPerSlot: 0
    limitTotal: 0
    limitPending: 0
    sourceMnemonic: ""
    sourceStartIndex: 0
    sourceStartValidatorIndex: null
    sourceIndexCount: 0
    targetValidatorIndex: null
    consolidationEpoch: null
    walletPrivkey: ""
    consolidationContract: 0x00b42dbF2194e931E80326D950320f7d9Dbeac02
    txAmount: "500000000000000000"
    txFeeCap: "100000000000"
    txTipCap: "1000000000"
    txGasLimit: 100000
    clientPattern: ""
    excludeClientPattern: ""
    awaitReceipt: false
    failOnReject: false
    consolidationTransactionsResultVar: ""
    consolidationReceiptsResultVar: ""
```
