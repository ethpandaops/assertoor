## `check_consensus_block_proposals` Task

### Description
The `check_consensus_block_proposals` task checks consensus block proposals to make sure they meet certain requirements. It looks at various details of the blocks to confirm they follow the rules or patterns you set.

### Configuration Parameters

- **`blockCount`**:\
  The number of blocks that need to match your criteria for the task to be successful.

- **`graffitiPattern`**:\
  A pattern to match the graffiti on the blocks.

- **`validatorNamePattern`**:\
  A pattern to identify blocks by the names of their validators.

- **`minAttestationCount`**:\
  The minimum number of attestations (votes or approvals) in a block.

- **`minDepositCount`**:\
  The minimum number of deposit actions required in a block.

- **`minExitCount`**:\
  The minimum number of exit operations in a block.

- **`minSlashingCount`**:\
  The minimum total number of slashing events (penalties for bad actions) in a block.

- **`minAttesterSlashingCount`**:\
  The minimum number of attester slashings in a block.

- **`minProposerSlashingCount`**:\
  The minimum number of proposer slashings in a block.

- **`minBlsChangeCount`**:\
  The minimum number of BLS changes in a block.

- **`minWithdrawalCount`**:\
  The minimum number of withdrawal actions in a block.

- **`minTransactionCount`**:\
  The minimum total number of transactions (any type) needed in a block.

- **`minBlobCount`**:\
  The minimum number of blob sidecars (extra data packets) in a block.

- **`expectDeposits`**:\
  A list of validator public keys expected to have deposit operations included in the block.

- **`expectExits`**:\
  A list of validator public keys expected to have exit operations included in the block.

- **`expectSlashings`**:\
  A list of expected slashing operations in the block, each specified as an object with a `publicKey` and a `slashingType` ("attester" or "proposer"). If `slashingType` is omitted, any type of slashing is accepted.

- **`expectBlsChanges`**:\
  A list of expected BLS change operations in the block, each as an object with a `publicKey` and the target `address` (optional).

- **`expectWithdrawals`**:\
  A list of expected withdrawal operations in the block, each as an object with a `publicKey`, `address`, and a `minAmount` specifying the minimum amount expected for the withdrawal.


### Defaults

These are the default settings for the `check_consensus_block_proposals` task:

```yaml
- name: check_consensus_block_proposals
  config:
    blockCount: 1
    graffitiPattern: ""
    validatorNamePattern: ""
    minAttestationCount: 0
    minDepositCount: 0
    minExitCount: 0
    minSlashingCount: 0
    minAttesterSlashingCount: 0
    minProposerSlashingCount: 0
    minBlsChangeCount: 0
    minWithdrawalCount: 0
    minTransactionCount: 0
    minBlobCount: 0
    expectDeposits: []
    expectExits: []
    expectSlashings: []
    expectBlsChanges: []
    expectWithdrawals: []
```
