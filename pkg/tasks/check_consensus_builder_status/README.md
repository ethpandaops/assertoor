## `check_consensus_builder_status` Task

### Description
The `check_consensus_builder_status` task verifies the status of builders on the consensus chain. Builders are a GLOAS-specific concept stored in a separate section of the beacon state (not the validator set). The task uses a shared builder cache that loads the full beacon state to extract builder information.

#### Task Behavior
- The task monitors builder status at each epoch.
- By default, the task returns immediately when a matching builder is found that meets all criteria.
- Use `continueOnPass: true` to keep monitoring even after success (useful for tracking status changes).
- The builder cache is shared across tasks and also updates the validator set cache when loading the full beacon state.

### Configuration Parameters

- **`builderPubKey`**:\
  The public key of the builder to check. If specified, the task will focus on the builder with this public key. Default: `""`.

- **`builderIndex`**:\
  The index of a specific builder in the builder list. If set, the task focuses on the builder at this index. If `null`, no filter on builder index is applied. Default: `null`.

- **`minBuilderBalance`**:\
  The minimum balance (in gwei) the builder must have. Default: `0`.

- **`maxBuilderBalance`**:\
  The maximum balance (in gwei) the builder may have. Default: `null`.

- **`expectActive`**:\
  If `true`, expect the builder to be active (withdrawable_epoch == FAR_FUTURE_EPOCH). Default: `false`.

- **`expectExiting`**:\
  If `true`, expect the builder to be exiting or exited (withdrawable_epoch != FAR_FUTURE_EPOCH). Default: `false`.

- **`failOnCheckMiss`**:\
  If `false`, the task will continue running and wait for the builder to match the expected status. If `true`, the task will fail immediately upon a status mismatch. Default: `false`.

- **`continueOnPass`**:\
  If set to `true`, the task continues monitoring builder status even after a matching builder is found. Default: `false`.

### Defaults

```yaml
- name: check_consensus_builder_status
  config:
    builderPubKey: ""
    builderIndex: null
    minBuilderBalance: 0
    maxBuilderBalance: null
    expectActive: false
    expectExiting: false
    failOnCheckMiss: false
    continueOnPass: false
```

### Outputs

- **`builder`**:\
  The builder information object containing pubkey, balance, deposit_epoch, withdrawable_epoch, and other data.

- **`builderIndex`**:\
  The builder's index in the builder list.

- **`pubkey`**:\
  The builder's public key as a hex string.
