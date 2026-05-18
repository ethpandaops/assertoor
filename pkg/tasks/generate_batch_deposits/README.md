## `generate_batch_deposits` Task

### Description
The `generate_batch_deposits` task spams the chain with valid, unique validator deposits by routing them through a small forwarder contract (`BatchDeposit`). Every deposit carries a freshly-derived BLS keypair and a fully-valid signature, which forces the consensus layer through the worst-case per-deposit signature verification path.

If `batchContract` is empty the task deploys a fresh forwarder bound to the configured deposit contract before generation starts and exposes its address via task outputs.

### Use Cases

- Stress-test consensus client deposit signature verification.
- Fill the deposit/pending queue at maximum throughput against a devnet.
- Reproduce edge cases where many deposits land in a single block.

### Configuration Parameters

- **`limitPerSlot`**: Maximum number of deposits to generate per slot. Counted in deposits, not in batches.
- **`limitTotal`**: Total deposits to generate.
- **`limitPendingBatches`**: Maximum number of in-flight batch transactions.
- **`mnemonic`**: Mnemonic phrase used to derive validator keys.
- **`startIndex`**: Index within the mnemonic at which to start key derivation.
- **`indexCount`**: Maximum number of validator keys to derive (an alternative cap to `limitTotal`).
- **`walletPrivkey`**: Private key of the wallet used to send the batch transactions and (if needed) deploy the contract.
- **`depositContract`**: Address of the beacon chain deposit contract.
- **`batchContract`**: Optional address of an already-deployed `BatchDeposit` forwarder. If empty, the task deploys one on start.
- **`batchSize`**: Number of deposits per batched transaction. Default `100`.
- **`batchTxGasLimit`**: Gas limit per batched transaction. Default `12_000_000`.
- **`depositAmount`**: ETH amount per deposit. Default `32` ETH.
- **`depositTxFeeCap`** / **`depositTxTipCap`**: Fee/tip caps (wei) for batch transactions.
- **`withdrawalCredentials`**: Required 32-byte withdrawal credentials shared by every deposit in every batch (e.g. `0x03 + 11 zero bytes + 20-byte address` for builder credentials).
- **`clientPattern`** / **`excludeClientPattern`**: Client selection regexes.
- **`awaitReceipt`**: Wait for every batch transaction receipt before completing.
- **`failOnReject`**: Fail the task if any batch transaction is rejected or reverted.
- **`awaitInclusion`**: Wait for every individual deposit to appear in a beacon block before completing.
- **`invalidSigPercent`**: Random percentage (0-100) of deposits to generate with corrupted signatures. Default `0`.

### Outputs

- **`batchContract`**: Address of the forwarder contract used (or deployed by) this task.
- **`validatorPubkeys`**: All derived validator pubkeys, in the order they were submitted.
- **`batchTransactions`**: Hashes of the submitted batch transactions.
- **`batchReceipts`**: Receipts for the batch transactions (when `awaitReceipt` is enabled).
- **`includedDeposits`**: Number of deposits confirmed on the beacon chain (when `awaitInclusion` is enabled).

### Defaults

```yaml
- name: generate_batch_deposits
  config:
    limitPerSlot: 0
    limitTotal: 0
    limitPendingBatches: 0
    mnemonic: ""
    startIndex: 0
    indexCount: 0
    walletPrivkey: ""
    depositContract: ""
    batchContract: ""
    batchSize: 100
    batchTxGasLimit: 12000000
    depositAmount: 32
    depositTxFeeCap: 100000000000
    depositTxTipCap: 1000000000
    withdrawalCredentials: ""
    clientPattern: ""
    excludeClientPattern: ""
    awaitReceipt: false
    failOnReject: false
    awaitInclusion: false
    invalidSigPercent: 0
```
