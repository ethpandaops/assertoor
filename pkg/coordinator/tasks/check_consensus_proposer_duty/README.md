## `check_consensus_proposer_duty` Task

### Description
The `check_consensus_proposer_duty` task is designed to check for a specific proposer duty on the consensus chain. It verifies if a matching validator is scheduled to propose a block within a specified future time frame (slot distance).

### Configuration Parameters

- **`validatorNamePattern`**:\
  A pattern to identify validators by name. This parameter is used to select validators for the duty check based on their names.

- **`validatorIndex`**:\
  The index of a specific validator to be checked. If this is set, the task focuses on the validator with this index. If it is `null`, the task does not filter by a specific validator index.

- **`minSlotDistance`**:\
  The minimum slot distance from the current slot at which to start checking for the validator's proposer duty. A value of 0 indicates the current slot.

- **`maxSlotDistance`**:\
  The maximum number of slots (individual time periods in the blockchain) within which the validator is expected to propose a block. The task succeeds if a matching validator is scheduled for block proposal within this slot distance.

- **`failOnCheckMiss`**:\
  This parameter specifies the task's behavior if a matching proposer duty is not found within the `maxSlotDistance`. If set to `false`, the task continues running until it either finds a matching proposer duty or reaches its timeout. If `true`, the task will fail immediately upon not finding a matching duty.

### Defaults

These are the default settings for the `check_consensus_proposer_duty` task:

```yaml
- name: check_consensus_proposer_duty
  config:
    validatorNamePattern: ""
    validatorIndex: null
    maxSlotDistance: 0
    failOnCheckMiss: false
```