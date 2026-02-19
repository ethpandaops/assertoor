## `check_consensus_slot_range` Task

### Description
The `check_consensus_slot_range` task verifies that the current wall clock time on the consensus chain falls within a specified range of slots and epochs. This is important for ensuring that the chain operates within expected time boundaries.

### Configuration Parameters

- **`minSlotNumber`**:\
  The minimum slot number that the consensus wall clock should be at or above. This sets the lower bound for the check.

- **`maxSlotNumber`**:\
  The maximum slot number that the consensus wall clock should not exceed. This sets the upper bound for the slot range.

- **`minEpochNumber`**:\
  The minimum epoch number that the consensus wall clock should be in or above. Similar to the minSlotNumber, this sets a lower limit, but in terms of epochs.

- **`maxEpochNumber`**:\
  The maximum epoch number that the consensus wall clock should not go beyond. This parameter sets the upper limit for the epoch range.

- **`failIfLower`**:\
  A flag that determines the task's behavior if the current wall clock time is below the specified minimum slot or epoch number. If `true`, the task will fail in such cases; if `false`, it will continue without failing.

- **`continueOnPass`**:\
  When set to `false` (default), the task exits immediately upon the slot/epoch being within range. When set to `true`, the task continues running after success, allowing it to be used for continuous monitoring within concurrent task execution.

### Defaults

These are the default settings for the `check_consensus_slot_range` task:

```yaml
- name: check_consensus_slot_range
  config:
    minSlotNumber: 0
    maxSlotNumber: 18446744073709551615
    minEpochNumber: 0
    maxEpochNumber: 18446744073709551615
    failIfLower: false
    continueOnPass: false
```
