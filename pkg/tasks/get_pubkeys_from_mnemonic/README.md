## `get_pubkeys_from_mnemonic` Task

### Description
The `get_pubkeys_from_mnemonic` task generates public keys from a given mnemonic phrase. This task is essential for setting up and verifying validator identities in scenarios involving multiple validators derived from a single mnemonic, commonly used in Ethereum staking operations.

### Configuration Parameters

- **`mnemonic`**:
  The mnemonic phrase used to generate the public keys. This should be a BIP-39 compliant seed phrase that is used to derive Ethereum validator keys.

- **`startIndex`**:
  The starting index from which to begin deriving public keys. This allows users to specify a segment of the key sequence for generation, rather than starting from the beginning.

- **`count`**:
  The number of public keys to generate from the specified `startIndex`.

### Outputs

- **`pubkeys`**:
  A list of validator public keys derived from the mnemonic. Each key corresponds to a validator index starting from the `startIndex` and continuing for `count` keys. This output is crucial for tasks that involve setting up validators, validating identities, or performing any operation that requires public key data.

### Defaults

Default settings for the `get_pubkeys_from_mnemonic` task:

```yaml
- name: get_pubkeys_from_mnemonic
  config:
    mnemonic: ""
    startIndex: 0
    count: 1
```
