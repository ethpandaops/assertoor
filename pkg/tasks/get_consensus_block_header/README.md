# `get_consensus_block_header` Task

### Description
The `get_consensus_block_header` task fetches a single beacon-block header from a chosen consensus client and exposes the identifying fields (`slot`, `root`, `proposerIndex`, `parentRoot`, `stateRoot`) as task outputs.

It is intentionally small: one beacon-API call against one client, returning the result. Playbooks compose it with `configVars` to feed realistic chain coordinates into other tasks — e.g. the GLOAS API compatibility matrix uses it to find a canonical slot/root pair to query envelope endpoints against.

### Configuration Parameters

- **`clientPattern`** *(string)*: Regex pattern selecting the source CL endpoint by name. Empty = first online client from the pool.
- **`slot`** *(uint64)*: Fetch the canonical block at this slot. Mutually exclusive with `blockRoot`.
- **`blockRoot`** *(string)*: Fetch the block with this root (0x-prefixed hex). Mutually exclusive with `slot`.
- **`headOffset`** *(int)*: When neither `slot` nor `blockRoot` is set, fetch `head - headOffset`. Default `0` (= head). Useful for pulling a slightly-back-from-head reference point that's stable across all clients.
- **`maxLookback`** *(int)*: Maximum number of consecutive missed slots to skip while resolving `slot` / `headOffset`. Default `8`.
- **`requestTimeout`** *(duration)*: Per-RPC timeout. Default `15s`.

### Outputs

- **`slot`** *(int)*: Slot of the returned block.
- **`root`** *(string)*: Canonical block root (0x-prefixed hex).
- **`proposerIndex`** *(int)*: Validator index of the block's proposer.
- **`parentRoot`** *(string)*: Parent block root.
- **`stateRoot`** *(string)*: State root committed in this block.

### Defaults

```yaml
- name: get_consensus_block_header
  config:
    clientPattern: ""
    slot: 0
    blockRoot: ""
    headOffset: 0
    maxLookback: 8
    requestTimeout: 15s
```

### Examples

Fetch the current head:

```yaml
- name: get_consensus_block_header
  id: head
```

Fetch a slot 4 back from head (stable for derived data lookups):

```yaml
- name: get_consensus_block_header
  id: recent
  config:
    headOffset: 4
```

Fetch a specific historical block by root:

```yaml
- name: get_consensus_block_header
  id: target
  config:
    blockRoot: "0x12ab34cd..."
```
