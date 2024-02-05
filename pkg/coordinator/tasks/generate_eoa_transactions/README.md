## `generate_eoa_transactions` Task

### Description
The `generate_eoa_transactions` task creates and sends standard transactions from End-User Owned Accounts (EOAs) to the network, essential for testing regular transaction processing.
The task is intended for mass transaction generation.

### Configuration Parameters

- **`limitPerBlock`**:\
  The maximum number of transactions to generate per block.

- **`limitTotal`**:\
  The total limit on the number of transactions to be generated.

- **`limitPending`**:\
  The limit based on the number of pending transactions.

- **`privateKey`**:\
  The private key of the main wallet.

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

- **`legacyTxType`**:\
  Determines whether to use the legacy type for transactions.

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

- **`contractDeployment`**:\
  Determines whether the transactions are for contract deployment.

- **`callData`**:\
  Call data included in the transactions.

- **`randomAmount`**:\
  If true, the transaction amount is randomized.

- **`amount`**:\
  The amount of ETH (in wei) to be sent in each transaction.

- **`clientPattern`**:\
  A regex pattern for selecting specific client endpoints for sending the transactions. This allows targeting particular clients or groups for transaction dispatch.

- **`excludeClientPattern`**:\
  A regex pattern to exclude certain client endpoints from being used to send the transactions. This feature provides an additional layer of control by allowing the exclusion of specific clients, which can be useful for testing under various network scenarios.


### Defaults

Default settings for the `generate_eoa_transactions` task:

```yaml
- name: generate_eoa_transactions
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
    legacyTxType: false
    feeCap: "100000000000"
    tipCap: "1000000000"
    gasLimit: 50000
    targetAddress: ""
    randomTarget: false
    contractDeployment: false
    callData: ""
    randomAmount: false
    amount: "0"
    clientPattern: ""
    excludeClientPattern: ""
```
