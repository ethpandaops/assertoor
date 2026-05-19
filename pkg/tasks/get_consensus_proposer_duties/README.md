# `get_consensus_proposer_duties` Task

### Description
The `get_consensus_proposer_duties` task fetches the proposer schedule for one epoch and surfaces:

- the full duties array (one entry per slot, with `slot`, `validator_index`, `pubkey`),
- the first duty strictly after the current head slot (handy as a "next proposer" coordinate for endpoints like `produceBlockV4`), and
- up to `maxDuties` deduplicated validator indices drawn from that schedule (handy as a body of real validator indices for endpoints like `POST /validator/duties/ptc/{epoch}`).

It is a small, generic primitive — playbooks compose it with `configVars` to craft realistic inputs for downstream tasks.

### Configuration Parameters

- **`clientPattern`** *(string)*: Regex pattern selecting the source CL endpoint by name. Empty = first online client from the pool.
- **`epoch`** *(uint64)*: Absolute epoch number to fetch duties for. When zero, falls back to "current epoch + `epochOffset`".
- **`epochOffset`** *(int)*: Offset added to the current epoch when `epoch` is zero. Default `0`. A value of `1` is useful when you need a duty strictly in the future.
- **`maxDuties`** *(int)*: Cap on the number of validator indices surfaced on `validatorIndices`. Default `16`.
- **`requestTimeout`** *(duration)*: Per-RPC timeout. Default `15s`.

### Outputs

- **`epoch`** *(int)*: The epoch whose duties were fetched.
- **`duties`** *(array)*: Array of `{slot, validator_index, pubkey}` objects in slot order, one per slot in the epoch.
- **`firstFutureSlot`** *(int)*: Slot of the first duty strictly after the current head slot, or `0` if no future duty exists in this epoch.
- **`firstFutureValidatorIndex`** *(int)*: Validator index of the proposer at `firstFutureSlot`.
- **`validatorIndices`** *(array)*: Up to `maxDuties` unique validator indices drawn from this epoch's schedule.

### Defaults

```yaml
- name: get_consensus_proposer_duties
  config:
    clientPattern: ""
    epoch: 0
    epochOffset: 0
    maxDuties: 16
    requestTimeout: 15s
```

### Examples

Fetch duties for the next epoch:

```yaml
- name: get_consensus_proposer_duties
  id: duties
  config:
    epochOffset: 1
```

Fetch duties for a specific historical epoch:

```yaml
- name: get_consensus_proposer_duties
  id: duties
  config:
    epoch: 1234
```
