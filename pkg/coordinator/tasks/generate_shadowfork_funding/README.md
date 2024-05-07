## `generate_child_wallet` Task

### Description
The `generate_child_wallet` task is designed to create a new, funded child wallet. This task is especially useful in scenarios requiring the setup of additional wallets for transaction testing, smart contract interactions, or other Ethereum network activities.

### Configuration Parameters

- **`privateKey`**:
  The private key of the parent wallet used for funding the new child wallet.

- **`walletSeed`**:
  A seed phrase used for generating the child wallet. This allows for deterministic wallet creation.

- **`randomSeed`**:
  If set to `true`, the task generates the child wallet using a random seed, resulting in a non-deterministic wallet.

- **`prefundAmount`**:
  The amount of cryptocurrency to be transferred to the child wallet during prefunding.

- **`prefundMinBalance`**:
  The minimum balance threshold in the parent wallet required to execute the prefunding. Prefunding occurs only if the parent wallet's balance is above this amount.

- **`prefundFeeCap`**:
  The maximum fee cap for the prefunding transaction to the child wallet.

- **`prefundTipCap`**:
  The tip cap for the prefunding transaction, determining the priority fee.

- **`walletAddressResultVar`**:
  The name of the variable to store the address of the newly created child wallet. This can be used for reference in subsequent tasks.

- **`walletPrivateKeyResultVar`**:
  The name of the variable to store the private key of the new child wallet. This ensures the child wallet can be accessed and used in later tasks.

### Defaults

Default settings for the `generate_child_wallet` task:

```yaml
- name: generate_child_wallet
  config:
    privateKey: ""
    walletSeed: ""
    randomSeed: false
    prefundAmount: "1000000000000000000"
    prefundMinBalance: "500000000000000000"
    prefundFeeCap: "500000000000"
    prefundTipCap: "1000000000"
    walletAddressResultVar: ""
    walletPrivateKeyResultVar: ""
```
