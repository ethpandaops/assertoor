## `generate_builder_deposits` Task

### Description
The `generate_builder_deposits` task generates and submits builder deposits to the Ethereum network as defined by EIP-8282 (Gloas). Builder deposits register builders into the consensus-layer builder registry and are submitted directly to the builder deposit system contract as **raw calldata** — there is no `deposit()` function selector and no ABI encoding.

The raw 184-byte calldata layout is:

```
0-48:    pubkey (48 bytes)
48-80:   withdrawal credentials (32 bytes, 0xB0-prefixed)
80-88:   amount (8 bytes, big-endian gwei)
88-184:  signature (96 bytes, BLS proof-of-possession)
```

The signature is a BLS proof-of-possession over the `DepositMessage{pubkey, withdrawal_credentials, amount}` signed under the dedicated `DOMAIN_BUILDER_DEPOSIT` (`0x0E000000`) domain, which separates builder deposits from regular validator deposits.

Builders are identified either by deriving keys from a `mnemonic` & key range, or by an explicit `publicKey` (only valid with `topUpDeposit`). For top-up deposits to an existing builder, the withdrawal credentials and signature are ignored by the consensus layer.

### Configuration Parameters

- **`limitPerSlot`**:
  Maximum number of builder deposits to generate per slot.

- **`limitTotal`**:
  Total limit on the number of builder deposits to generate.

- **`limitPending`**:
  Maximum number of pending builder deposits allowed before waiting.

- **`mnemonic`**:
  Mnemonic phrase used to derive builder BLS keys.

- **`startIndex`**:
  Index within the mnemonic from which to start deriving builder keys.

- **`indexCount`**:
  Number of builder keys to derive from the mnemonic.

- **`publicKey`**:
  Public key of an existing builder for top-up deposits (requires `topUpDeposit`).

- **`walletPrivkey`**:
  Private key of the wallet used to fund builder deposit transactions.

- **`builderDepositContract`**:
  Address of the builder deposit system contract (EIP-8282).

- **`depositAmount`**:
  Amount of ETH to deposit per builder. Must be at least `1` ETH (`BUILDER_MIN_DEPOSIT`).

- **`withdrawalCredentials`**:
  Custom withdrawal credentials (must be 0xB0-prefixed, 32 bytes). If empty, credentials are derived from `builderAddress`, or from the funding wallet address.

- **`builderAddress`**:
  Execution address used to build the 0xB0 withdrawal credentials when `withdrawalCredentials` is not set. This address is the only one allowed to exit the builder later.

- **`topUpDeposit`**:
  If true, adds to an existing builder balance instead of registering a new builder.

- **`txFeeCap`**, **`txTipCap`**, **`txGasLimit`**:
  Transaction fee parameters.

- **`txFeeBuffer`**:
  Extra value (in wei) sent on top of the deposit amount to cover the request fee. The contract requires `msg.value - fee >= amount * 1 gwei`.

- **`clientPattern`**, **`excludeClientPattern`**:
  Regular expressions for selecting or excluding specific client endpoints.

- **`awaitReceipt`**:
  Wait for transaction receipts before completing.

- **`failOnReject`**:
  Fail the task if any deposit transaction is rejected.

- **`awaitInclusion`**:
  Wait for builder deposits to be included as builder deposit requests in beacon blocks before completing.

- **`invalidSigPercent`**:
  Random percentage (0-100) of deposits to generate with corrupted signatures (for negative testing).

### Outputs

- **`builderPubkeys`**:
  Array of builder public keys for the deposits.

- **`depositTransactions`**:
  Array of builder deposit transaction hashes.

- **`depositReceipts`**:
  Array of transaction receipts (when `awaitReceipt` is enabled).

- **`includedDeposits`**:
  Number of builder deposits included on the beacon chain (when `awaitInclusion` is enabled).

### Defaults

```yaml
- name: generate_builder_deposits
  config:
    limitPerSlot: 0
    limitTotal: 0
    limitPending: 0
    mnemonic: ""
    startIndex: 0
    indexCount: 0
    publicKey: ""
    walletPrivkey: ""
    builderDepositContract: "0x0000884d2AA32eAa155F59A2f24eFa73D9008282"
    depositAmount: 1
    withdrawalCredentials: ""
    builderAddress: ""
    topUpDeposit: false
    txFeeCap: "100000000000"
    txTipCap: "1000000000"
    txGasLimit: 400000
    txFeeBuffer: "1000000000000000"
    clientPattern: ""
    excludeClientPattern: ""
    awaitReceipt: false
    failOnReject: false
    awaitInclusion: false
    invalidSigPercent: 0
```
