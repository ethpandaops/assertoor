## `check_consensus_blob_count` Task

### Description
The `check_consensus_blob_count` task verifies that proposed consensus blocks carry as many blobs as the chain allows. Instead of comparing against a number written into the playbook, it reads the blob limits of the active fork from the network itself, so the same task keeps working across forks that change the blob parameters.

The limits come from the execution client's `eth_config` response (EIP-7910), because the consensus config only publishes `MAX_BLOBS_PER_BLOCK` while the target lives on the execution side. They are the limits of the fork the client is on when the block is checked, so a lookback reaching across a fork boundary is checked against the newer limits. The task fails if the selected execution client does not serve `eth_config`, does not include a blob schedule in it, or cannot be reached.

A block carrying more blobs than the max is a protocol violation rather than a missed target, so it is reported separately and fails the task by default.

### Configuration Parameters

- **`checkLookback`**:\
  The number of already cached blocks to check before waiting for new proposals. Set to `0` to only look at blocks proposed while the task runs.

- **`blockCount`**:\
  The number of blocks that need to reach the blob limit for the task to be successful. Has to be at least `1`.

- **`blobLimit`**:\
  The limit a block has to reach, either `target` or `max`. With `target` a block passes once it carries at least the target number of blobs, with `max` it has to carry the full max.

- **`validatorNamePattern`**:\
  A regex pattern to only check blocks proposed by matching validators. An empty pattern checks every block.

- **`clientPattern`**:\
  A regex pattern selecting the execution client that `eth_config` is read from. Any ready client is used when empty.

- **`excludeClientPattern`**:\
  A regex pattern excluding execution clients from the `eth_config` lookup.

- **`failOnOverMax`**:\
  Whether a block carrying more blobs than the max fails the task. When `false`, such a block is logged as a warning and the task keeps waiting.

### Outputs

- **`matchingBlockRoots`**:\
  The roots of the blocks that reached the blob limit.

- **`blobTarget`**:\
  The target number of blobs per block reported by the execution client.

- **`blobMax`**:\
  The max number of blobs per block reported by the execution client.

- **`maxBlobCount`**:\
  The highest blob count seen in any checked block.

### Defaults

These are the default settings for the `check_consensus_blob_count` task:

```yaml
- name: check_consensus_blob_count
  config:
    checkLookback: 1
    blockCount: 1
    blobLimit: target
    validatorNamePattern: ""
    clientPattern: ""
    excludeClientPattern: ""
    failOnOverMax: true
```
