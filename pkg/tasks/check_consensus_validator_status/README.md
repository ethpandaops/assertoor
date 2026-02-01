## `check_consensus_validator_status` Task

### Description
The `check_consensus_validator_status` task is focused on verifying the status of validators on the consensus chain. It checks if the validators are in the expected state, as per the specified criteria.

#### Task Behavior
- The task monitors validator status at each epoch.
- By default, the task returns immediately when a matching validator is found.
- Use `continueOnPass: true` to keep monitoring even after success (useful for tracking status changes).

### Configuration Parameters

- **`validatorPubKey`**:\
  The public key of the validator to be checked. If specified, the task will focus on the validator with this public key. Default: `""`.

- **`validatorNamePattern`**:\
  A pattern for identifying validators by name. Useful for filtering validators to be checked based on their names. Default: `""`.

- **`validatorIndex`**:\
  The index of a specific validator. If set, the task focuses on the validator with this index. If `null`, no filter on validator index is applied. Default: `null`.

- **`validatorStatus`**:\
  A list of allowed validator statuses. The task will check if the validator's status matches any of the statuses in this list. Default: `[]`.

- **`minValidatorBalance`**:\
  The minimum balance of the validator to match. Default: `0`.

- **`maxValidatorBalance`**:\
  The maximum balance of the validator to match. Default: `null`.

- **`withdrawalCredsPrefix`**:\
  The withdrawal credentials prefix the validator should have. Default: `""`.

- **`failOnCheckMiss`**:\
  Determines the task's behavior if the validator's status does not match any of the statuses in `validatorStatus`. If `false`, the task will continue running and wait for the validator to match the expected status. If `true`, the task will fail immediately upon a status mismatch. Default: `false`.

- **`continueOnPass`**:\
  If set to `true`, the task continues monitoring validator status even after a matching validator is found. This is useful for tracking status changes over time. If `false` (default), the task exits immediately on success.

- **`validatorInfoResultVar`**:\
  The name of the variable where the resulting information about the validator will be stored. This includes status, index, balance and any other relevant data fetched during the check. Default: `""`.

- **`validatorPubKeyResultVar`**:\
  The name of the variable where the validator's public key will be stored. Default: `""`.

### Defaults

These are the default settings for the `check_consensus_validator_status` task:

```yaml
- name: check_consensus_validator_status
  config:
    validatorPubKey: ""
    validatorNamePattern: ""
    validatorIndex: null
    validatorStatus: []
    minValidatorBalance: 0
    maxValidatorBalance: null
    withdrawalCredsPrefix: ""
    failOnCheckMiss: false
    continueOnPass: false
    validatorInfoResultVar: ""
    validatorPubKeyResultVar: ""
```

### Outputs

- **`validator`**:\
  The validator information object containing status, index, balance, and other data.

- **`pubkey`**:\
  The validator's public key as a hex string.
