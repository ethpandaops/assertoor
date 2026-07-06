## `generate_builder_exits` Task

### Description
The `generate_builder_exits` task generates and submits builder exit requests to the Ethereum network as defined by EIP-8282 (Gloas). As of Gloas, builder exits are **no longer voluntary (consensus-layer) exits** — they are submitted as execution-layer requests to the builder exit system contract.

The request is submitted as **raw calldata** (no function selector): the 48-byte builder public key. The contract prepends `msg.sender` as the source address when emitting the request, so the transaction **must be sent from the builder's execution address** (the address embedded in its 0xB0 withdrawal credentials). Only that address is authorized to exit the builder.

The builders to exit can be specified either by an explicit `sourcePubkey`, or by deriving keys from a `sourceMnemonic` & key range.

### Configuration Parameters

- **`limitPerSlot`**:
  Maximum number of builder exits to generate per slot.

- **`limitTotal`**:
  Total limit on the number of builder exits to generate.

- **`limitPending`**:
  Maximum number of pending builder exits allowed before waiting.

- **`sourcePubkey`**:
  The static builder public key to exit.

- **`sourceMnemonic`**:
  Mnemonic phrase used to derive builder keys for exits.

- **`sourceStartIndex`**:
  Index within the mnemonic from which to start deriving builder keys.

- **`sourceIndexCount`**:
  Number of builders to generate exit requests for.

- **`walletPrivkey`**:
  Private key of the wallet used to send builder exit transactions. Must match the builder's execution address (its 0xB0 withdrawal credentials).

- **`builderExitContract`**:
  Address of the builder exit system contract (EIP-8282).

- **`txAmount`**:
  Amount (in wei) to send with the transaction to cover the request fee.

- **`txFeeCap`**, **`txTipCap`**, **`txGasLimit`**:
  Transaction fee parameters.

- **`clientPattern`**, **`excludeClientPattern`**:
  Regular expressions for selecting or excluding specific client endpoints.

- **`awaitReceipt`**:
  Wait for transaction receipts before completing.

- **`failOnReject`**:
  Fail the task if any transaction is rejected.

### Outputs

- **`transactionHashes`**:
  Array of builder exit transaction hashes.

- **`transactionReceipts`**:
  Array of transaction receipts (when `awaitReceipt` is enabled).

### Defaults

```yaml
- name: generate_builder_exits
  config:
    limitPerSlot: 0
    limitTotal: 0
    limitPending: 0
    sourcePubkey: ""
    sourceMnemonic: ""
    sourceStartIndex: 0
    sourceIndexCount: 0
    walletPrivkey: ""
    builderExitContract: "0x000014574A74c805590AFF9499fc7A690f008282"
    txAmount: "1000000000000000"
    txFeeCap: "100000000000"
    txTipCap: "1000000000"
    txGasLimit: 200000
    clientPattern: ""
    excludeClientPattern: ""
    awaitReceipt: false
    failOnReject: false
```
