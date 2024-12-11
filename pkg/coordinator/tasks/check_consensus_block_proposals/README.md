## `check_consensus_block_proposals` Task

### Description
The `check_consensus_block_proposals` task assesses consensus block proposals against specified criteria to ensure they comply with expected blockchain operations and standards. This task is crucial for validating the integrity and content of blocks proposed on the consensus layer.

### Configuration Parameters

- **`blockCount`**:\
  The number of blocks that need to match your criteria for the task to be successful.

- **`graffitiPattern`**:\
  A regex pattern to match against the graffiti field of the block, allowing for specific textual content verification.

- **`validatorNamePattern`**:\
  A regex pattern to select validators by name involved in block proposals.

- **`extraDataPattern`**:\
  A regex pattern to validate the extra data field within the block header.

- **`minAttestationCount`**:\
  The minimum number of attestations required in the block to satisfy the check.

- **`minDepositCount`**:\
  The minimum number of deposit events that must be included in the block.

- **`minExitCount`**:\
  The minimum number of validator exits required in the block.

- **`minSlashingCount`**:\
  The minimum number of slashing events the block must contain.

- **`minAttesterSlashingCount`**:\
  The minimum number of attester slashing operations required in the block.

- **`minProposerSlashingCount`**:\
  The minimum number of proposer slashing operations the block must include.

- **`minBlsChangeCount`**:\
  The minimum number of BLS key changes needed in the block.

- **`minWithdrawalCount`**:\
  The minimum number of withdrawals that must be processed in the block.

- **`minTransactionCount`**:\
  The minimum number of transactions (of any type) required in the block.

- **`minBlobCount`**:\
  The minimum number of blob sidecars that must be included in the block.

- **`minDepositRequestCount`**:\
  The minimum number of deposit request operations needed in the block.

- **`minWithdrawalRequestCount`**:\
  The minimum number of withdrawal requests required in the block.

- **`minConsolidationRequestCount`**:\
  The minimum number of consolidation requests that the block must include.

- **`expectDeposits`**:\
  A list of validator public keys, specifying which validators should have deposit transactions included in the block.

- **`expectExits`**:\
  A list of validator public keys indicating which validators should have exit transactions included in the block.

- **`expectSlashings`**:\
  A list detailing expected slashing operations in the block, with each entry specifying a public key and a slashing type (`attester` or `proposer`). If the slashing type is omitted, any type of slashing is accepted.
  `{publicKey: "0x0000...", slashingType: "attester"|"proposer"}`

- **`expectBlsChanges`**:\
  Specifies expected BLS key changes in the block, with each object detailing a public key and a new address for the key change.
  `{publicKey: "0x0000...", address: "0x00..."}`

- **`expectWithdrawals`**:\
  Specifies expected withdrawal operations, including public keys, destination addresses, and minimum withdrawal amounts.
  `{publicKey: "0x0000...", address: "0x00...", minAmount: 0}`

- **`expectDepositRequests`**:\
  Specifies expected deposit request operations, each object detailing the public key, withdrawal credentials, and deposit amount.
  `{publicKey:"0x0000...", withdrawalCredentials: "0x0000...", amount: 0}`

- **`expectWithdrawalRequests`**:\
  Specifies expected withdrawal request operations, each detailing the source address, validator public key, and amount to be withdrawn.
  `{sourceAddress:"0x0000...", validatorPubkey: "0x0000...", amount: 0}`

- **`expectConsolidationRequests`**:\
  Specifies expected consolidation request operations, each specifying source addresses, source public keys, and target public keys for consolidation.
  `{sourceAddress:"0x0000...", sourcePubkey: "0x0000...", targetPubkey: "0x0000..."}`

### Defaults

Default settings for the `check_consensus_block_proposals` task:

```yaml
- name: check_consensus_block_proposals
  config:
    blockCount: 1
    graffitiPattern: ""
    validatorNamePattern: ""
    extraDataPattern: ""
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
    minDepositRequestCount: 0
    minWithdrawalRequestCount: 0
    minConsolidationRequestCount: 0
    expectDeposits: []
    expectExits: []
    expectSlashings: []
    expectBlsChanges: []
    expectWithdrawals: []
    expectDepositRequests: []
    expectWithdrawalRequests: []
    expectConsolidationRequests: []
```
