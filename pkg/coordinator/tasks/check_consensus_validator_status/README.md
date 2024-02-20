## `check_consensus_validator_status` Task

### Description
The `check_consensus_validator_status` task is focused on verifying the status of validators on the consensus chain. It checks if the validators are in the expected state, as per the specified criteria.

### Configuration Parameters

- **`validatorPubKey`**:\
  The public key of the validator to be checked. If specified, the task will focus on the validator with this public key.

- **`validatorNamePattern`**:\
  A pattern for identifying validators by name. Useful for filtering validators to be checked based on their names.

- **`validatorIndex`**:\
  The index of a specific validator. If set, the task focuses on the validator with this index. If `null`, no filter on validator index is applied.

- **`validatorStatus`**:\
  A list of allowed validator statuses. The task will check if the validator's status matches any of the statuses in this list.

- **`failOnCheckMiss`**:\
  Determines the task's behavior if the validator's status does not match any of the statuses in `validatorStatus`. If `false`, the task will continue running and wait for the validator to match the expected status. If `true`, the task will fail immediately upon a status mismatch.

- **`validatorInfoResultVar`**:\
  The name of the variable where the resulting information about the validator will be stored. This includes status, index, balance and any other relevant data fetched during the check.

### Defaults

These are the default settings for the `check_consensus_validator_status` task:

```yaml
- name: check_consensus_validator_status
  config:
    validatorPubKey: ""
    validatorNamePattern: ""
    validatorIndex: null
    validatorStatus: []
    failOnCheckMiss: false
    validatorInfoResultVar: ""
```
