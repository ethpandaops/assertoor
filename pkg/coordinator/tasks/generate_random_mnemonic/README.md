## `generate_random_mnemonic` Task

### Description
The `generate_random_mnemonic` task is designed to create a random mnemonic phrase. This task is particularly useful in scenarios where generating new wallets or accounts is necessary for testing purposes. A mnemonic phrase, often used in cryptocurrency wallets, is a set of words that can be translated into a binary seed, which in turn can be used to generate wallet addresses.

### Configuration Parameters

- **`mnemonicResultVar`**:
  The name of the variable where the generated mnemonic will be stored. This allows the mnemonic to be used in subsequent tasks, enabling the dynamic creation of wallets or accounts based on the mnemonic.

### Defaults

Default settings for the `generate_random_mnemonic` task:

```yaml
- name: generate_random_mnemonic
  config:
    mnemonicResultVar: ""
```
