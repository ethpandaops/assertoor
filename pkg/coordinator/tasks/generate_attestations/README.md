## `generate_attestations` Task

### Description
The `generate_attestations` task is designed to generate valid attestations for a specified range of validator keys and submit them to the network. This task fetches attester duties for the configured validators, retrieves attestation data from the beacon node, signs the attestations with the validator private keys, and submits them via the beacon API.

The task supports advanced configuration options for testing various attestation scenarios, including attesting for previous epochs, using delayed head blocks, and randomizing late head offsets per attestation.

### Configuration Parameters

- **`mnemonic`**:\
  A mnemonic phrase used for generating the validators' private keys. The keys are derived using the standard BIP39/BIP44 path (`m/12381/3600/{index}/0/0`).

- **`startIndex`**:\
  The starting index within the mnemonic from which to begin generating validator keys. This sets the initial point for key derivation.

- **`indexCount`**:\
  The number of validator keys to generate from the mnemonic, determining how many validators will be used for attestation generation.

- **`limitTotal`**:\
  The total limit on the number of attestations that the task will generate. The task will stop after reaching this limit.

- **`limitEpochs`**:\
  The total number of epochs to generate attestations for. The task will stop after processing this many epochs.

- **`clientPattern`**:\
  A regex pattern for selecting specific client endpoints for fetching attestation data and submitting attestations. If left empty, any available endpoint will be used.

- **`excludeClientPattern`**:\
  A regex pattern to exclude certain client endpoints from being used. This parameter adds a layer of control by allowing the exclusion of specific clients.

### Advanced Settings

- **`lastEpochAttestations`**:\
  When set to `true`, the task will generate attestations for the previous epoch's duties instead of the current epoch. This is useful for testing late attestation scenarios. Attestations are sent one slot at a time (each wallclock slot sends attestations for the corresponding slot in the previous epoch).

- **`sendAllLastEpoch`**:\
  When set to `true`, instead of sending attestations slot-by-slot, all attestations for the previous epoch are sent at once at each epoch boundary. This is useful for bulk testing of late attestations. Requires `lastEpochAttestations` to be implicitly treated as true.

- **`lateHead`**:\
  Offsets the beacon block root in the attestation by the specified number of blocks. For example, setting `lateHead: 5` will use the block root from 5 blocks before the current head. This simulates validators with a delayed view of the chain. Positive values go back (older blocks), negative values go forward.

- **`randomLateHead`**:\
  Specifies a range for randomizing the late head offset per attestation in `"min:max"` or `"min-max"` format. For example, `randomLateHead: "1-5"` will apply a random late head offset between 1 and 5 blocks (inclusive). By default, each attestation gets its own random offset. Use `lateHeadClusterSize` to group attestations with the same offset.

- **`lateHeadClusterSize`**:\
  Controls how many attestations share the same random late head offset. Default is `1` (each attestation gets its own random offset). Setting this to a higher value groups attestations together with the same late head value. For example, `lateHeadClusterSize: 10` means every 10 attestations will share the same random offset.


### Defaults

Default settings for the `generate_attestations` task:

```yaml
- name: generate_attestations
  config:
    mnemonic: ""
    startIndex: 0
    indexCount: 0
    limitTotal: 0
    limitEpochs: 0
    clientPattern: ""
    excludeClientPattern: ""
    lastEpochAttestations: false
    sendAllLastEpoch: false
    lateHead: 0
    randomLateHead: ""
    lateHeadClusterSize: 1
```

### Example Usage

Basic usage to generate attestations for 100 validators over 5 epochs:

```yaml
- name: generate_attestations
  config:
    mnemonic: "your mnemonic phrase here"
    startIndex: 0
    indexCount: 100
    limitEpochs: 5
```

Advanced usage with late head for testing delayed attestations:

```yaml
- name: generate_attestations
  config:
    mnemonic: "your mnemonic phrase here"
    startIndex: 0
    indexCount: 50
    limitTotal: 1000
    clientPattern: "lighthouse.*"
    lastEpochAttestations: true
    lateHead: 3
```

Bulk send all previous epoch attestations at once with random late head:

```yaml
- name: generate_attestations
  config:
    mnemonic: "your mnemonic phrase here"
    startIndex: 0
    indexCount: 100
    limitEpochs: 3
    sendAllLastEpoch: true
    randomLateHead: "1:10"
```
