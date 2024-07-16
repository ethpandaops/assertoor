## `generate_transaction` Task

### Description
The `generate_transaction` task creates and sends a single transaction to the network and optionally checks the transaction receipt. This task is useful for testing specific transaction behaviors, including contract deployments, and verifying receipt properties like triggered events.

### Configuration Parameters

- **`privateKey`**:\
  The private key used for generating the transaction.

- **`legacyTxType`**:\
  If `true`, generates a legacy (type 0) transaction. If `false`, a dynamic fee (type 2) transaction is created.

- **`blobTxType`**:\
  If `true`, generates a blob (type 3) transaction. Otherwise, a dynamic fee (type 2) transaction is used.

- **`setCodeTxType`**:\
  If `true`, generates a set code (type 4) transaction. Otherwise, a dynamic fee (type 2) transaction is used.

- **`blobFeeCap`**:\
  The fee cap for blob transactions. Used only if `blobTxType` is `true`.

- **`feeCap`**:\
  The maximum fee cap for the transaction.

- **`tipCap`**:\
  The tip cap for the transaction.

- **`gasLimit`**:\
  The gas limit for the transaction.

- **`targetAddress`**:\
  The target address for the transaction.

- **`randomTarget`**:\
  If `true`, the transaction is sent to a random address.

- **`contractDeployment`**:\
  If `true`, the transaction is for deploying a contract.

- **`callData`**:\
  Call data included in the transaction.

- **`blobData`**:\
  Data for the blob component of the transaction. Used only if `blobTxType` is `true`.

- **`authorizations`**:\
  EOA code authorizations. Used only if `setCodeTxType` is `true`.
  ```yaml
  - { "chainId": 0, "nonce": null, "codeAddress": "0x000...", "signerPrivkey": "000..." }
  ```

- **`randomAmount`**:\
  If `true`, the transaction amount is randomized.

- **`amount`**:\
  The amount of cryptocurrency to be sent in the transaction.

- **`clientPattern`**:\
  A regex pattern to select specific client endpoints for sending the transaction.

- **`excludeClientPattern`**:\
  A regex pattern to exclude certain clients from being used for sending the transaction.

- **`awaitReceipt`**:\
  If `false`, the task succeeds immediately after sending the transaction without waiting for the receipt. If `true`, it waits for the receipt.

- **`failOnReject`**:\
  If `true`, the task fails if the transaction is rejected.

- **`failOnSuccess`**:\
  If `true`, the task fails if the transaction is successful and not rejected.

- **`expectEvents`**:\
  A list of events that the transaction is expected to trigger, specified in a structured object format. Each event object can have the following properties: `topic0`, `topic1`, `topic2`, `topic3`, and `data`. All these properties are optional and expressed as hexadecimal strings (e.g., "0x000..."). The task checks all triggered events against these objects and looks for a match that satisfies all specified properties in any single event. An example event object might look like this:
  
  ```yaml
  - { "topic0": "0x000...", "topic1": "0x000...", "topic2": "0x000...", "topic3": "0x000...", "data": "0x000..." }
  ```

- **`transactionHashResultVar`**:\
  The variable name to store the transaction hash, available for use by subsequent tasks.

- **`transactionReceiptResultVar`**:\
  The variable name to store the full transaction receipt, available for use by subsequent tasks.

- **`contractAddressResultVar`**:\
  The variable name to store the deployed contract address if the transaction was a contract deployment, available for use by subsequent tasks.

### Defaults

Default settings for the `generate_transaction` task:

```yaml
- name: generate_transaction
  config:
    privateKey: ""
    legacyTxType: false
    blobTxType: false
    setCodeTxType: false
    blobFeeCap: null
    feeCap: "100000000000"
    tipCap: "1000000000"
    gasLimit: 50000
    targetAddress: ""
    randomTarget: false
    contractDeployment: false
    callData: ""
    blobData: ""
    authorizations: []
    randomAmount: false
    amount: "0"
    clientPattern: ""
    excludeClientPattern: ""
    awaitReceipt: true
    failOnReject: false
    failOnSuccess: false
    expectEvents: []
    transactionHashResultVar: ""
    transactionReceiptResultVar: ""
    contractAddressResultVar: ""
```
