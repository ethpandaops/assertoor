## `generate_blob_transactions` Task

### Description
The `generate_blob_transactions` task creates and sends a large number of blob transactions to the network. It's configured to operate under various limits, and at least one limit parameter is necessary for the task to function.

### Configuration Parameters

- **`limitPerBlock`**:\
  The maximum number of blob transactions to generate per block.

- **`limitTotal`**:\
  The total limit on the number of blob transactions to be generated.

- **`limitPending`**:\
  The limit based on the number of pending blob transactions.

- **`privateKey`**:\
  The private key used for transaction generation.

- **`childWallets`**:\
  The number of child wallets to be created and funded. (If 0, send blob transactions directly from privateKey wallet)

- **`walletSeed`**:\
  The seed phrase used for generating child wallets. (Will be used in combination with privateKey to generate unique child wallets that do not collide with other tasks)

- **`refillPendingLimit`**:\
  The maximum number of pending refill transactions allowed. This limit is used to control the refill process for child wallets, ensuring that the number of refill transactions does not exceed this threshold.

- **`refillFeeCap`**:\
  The maximum fee cap for refilling transactions.

- **`refillTipCap`**:\
  The maximum tip cap for refill transactions.

- **`refillAmount`**:\
  The amount to refill in each child wallet.

- **`refillMinBalance`**:\
  The minimum balance required before triggering a refill.

- **`blobSidecars`**:\
  The number of blob sidecars to include in each transaction.

- **`blobFeeCap`**:\
  The fee cap specifically for blob transactions.

- **`feeCap`**:\
  The maximum fee cap for transactions.

- **`tipCap`**:\
  The tip cap for transactions.

- **`gasLimit`**:\
  The gas limit for each transaction.

- **`targetAddress`**:\
  The target address for transactions.

- **`randomTarget`**:\
  If true, transactions are sent to random addresses.

- **`callData`**:\
  Call data to be included in the transactions.

- **`blobData`**:\
  Data for the blob component of the transactions.

- **`randomAmount`**:\
  If true, the transaction amount is randomized, using `amount` as limit.

- **`amount`**:\
  The amount of ETH (in Wei) to be sent in each blob transaction.

- **`clientPattern`**:\
  A regex pattern for selecting specific client endpoints to send transactions. If unspecified, transactions are sent through any available endpoint.

- **`excludeClientPattern`**:\
  A regex pattern to exclude certain client endpoints from being used for sending transactions. This allows for more precise control over which clients are utilized.


### Defaults

Default settings for the `generate_blob_transactions` task:

```yaml
- name: generate_blob_transactions
  config:
    limitPerBlock: 0
    limitTotal: 0
    limitPending: 0
    privateKey: ""
    childWallets: 0
    walletSeed: ""
    refillPendingLimit: 200
    refillFeeCap: "500000000000"
    refillTipCap: "1000000000"
    refillAmount: "1000000000000000000"
    refillMinBalance: "500000000000000000"
    blobSidecars: 1
    blobFeeCap: "10000000000"
    feeCap: "100000000000"
    tipCap: "2000000000"
    gasLimit: 100000
    targetAddress: ""
    randomTarget: false
    callData: ""
    blobData: ""
    randomAmount: false
    amount: "0"
    clientPattern: ""
    excludeClientPattern: ""
```
