# `run_network_disruption` Task

## Description

The `run_network_disruption` task drives a [disruptoor](https://github.com/ethpandaops/disruptoor) instance to apply or heal network disruptions — hard partitions, single-target isolations, and traffic shaping — on a Kurtosis-launched devnet.

### Task Behavior

- Waits for the disruptoor API to report healthy (configurable timeout), then performs the configured action.
- Partition, isolation, and shaping entries are passed through to disruptoor **verbatim** (the wire format of `PUT /v1/state`, see the [disruptoor JSON schema](https://github.com/ethpandaops/disruptoor/blob/master/schemas/v1-state.json)). Assertoor only checks that every entry carries a unique `name`; everything else is validated by disruptoor, whose error messages are surfaced in the task failure.
- Task outputs are read back via `GET /v1/state` after the action, which reflects the *applied* state.
- The task completes as soon as the disruption is applied (disruptoor applies synchronously); pair it with `check_*` tasks to assert the network effects, and put a `clear` invocation in `cleanupTasks` so an aborted test heals the network.

### Actions

- **`set`** (default): Replace the entire disruptoor state with the configured entries. Anything previously active that is not part of this request is healed.
- **`update`**: Read-merge-write. Entries named in `removeNames` are dropped from the current state, then each configured entry replaces its same-name predecessor or is appended. The write is guarded with `If-Match` and retried when a concurrent writer wins the race. Use this to compose disruptions across tasks (e.g. keep a baseline jitter while toggling a blackout).
- **`clear`**: Heal everything.

## Configuration Parameters

- **`disruptoorUrl`**:\
  Base URL of the disruptoor HTTP API (e.g. `http://disruptoor:7700`). Required.

- **`action`**:\
  Action to perform: `set`, `update`, or `clear`. Default: `set`.

- **`partitions`**:\
  Disruptoor partition entries. Each splits the enclave into 2+ disjoint groups; traffic crossing group boundaries is dropped. Fields: `name`, `groups` (list of selectors), optional `scope`, optional `symmetric`.

- **`isolations`**:\
  Disruptoor isolation entries. Each cuts the containers matched by its `target` selector off from **the rest of the enclave** — the counterparty group is computed by disruptoor at apply time, so it never needs to be enumerated and stays correct when the topology changes. A target matching multiple containers is isolated *as a group* (traffic among its members keeps flowing); declare one isolation per container to black out several containers individually. Fields: `name`, `target` (selector), optional `scope`.

- **`shaping`**:\
  Disruptoor shaping entries: per-target `delay`/`jitter`/`loss`/`bandwidth` degradation. Requires `scope: [include_control]` acknowledgement (disruptoor v0 shapes all egress traffic).

- **`removeNames`**:\
  Entry names to remove from the current state before merging. Only valid with `action: update`.

- **`awaitApiTimeout`**:\
  How long to wait for the disruptoor API to report healthy before acting. `0` acts immediately. Default: `30s`.

- **`pollInterval`**:\
  Interval between health probes while waiting for the API. Default: `2s`.

- **`requestTimeout`**:\
  Timeout for a single HTTP request. Default: `10s`.

### Selectors and scopes

Group and target selectors are label matches against the enclave's containers; keys without a dot get the `com.kurtosistech.custom.ethereum-package.` prefix. Common keys on ethereum-package devnets: `node-index` (1-based participant index) and `client-type` (`beacon`, `execution`, `validator`). Multiple values within a key OR together; multiple keys AND together.

`scope` selects the port classes a disruption bites on: `cl_p2p`, `el_p2p` (the default pair), and `include_control` as an explicit opt-in to also cut RPC/engine/metrics/VC↔CL traffic. Without `include_control`, tests keep their visibility into the disrupted node.

## Outputs

- **`appliedState`**:\
  The disruptoor state after the action (object; reflects applied reality).

- **`partitionCount`** / **`isolationCount`** / **`shapingCount`**:\
  Number of active entries of each kind after the action.

## Examples

### Fully black out one node's beacon client, then heal

Cuts participant 1's CL off from everything — other participants *and* its own execution/validator client:

```yaml
- name: run_network_disruption
  title: "Black out the target beacon node"
  config:
    disruptoorUrl: "http://disruptoor:7700"
    isolations:
      - name: blackout-target-cl
        target: { node-index: 1, client-type: beacon }
        scope: [cl_p2p, el_p2p, include_control]

# ... assert the network effects with check_* tasks ...

- name: run_network_disruption
  title: "Heal the blackout"
  config:
    disruptoorUrl: "http://disruptoor:7700"
    action: clear
```

### Isolate a whole participant

Without `client-type`, the target matches the participant's CL, EL, and VC together — they keep talking to *each other* but lose the rest of the network. Useful for "node offline" scenarios where the stack itself stays coherent:

```yaml
- name: run_network_disruption
  title: "Take participant 2 off the network"
  config:
    disruptoorUrl: "http://disruptoor:7700"
    isolations:
      - name: offline-node-2
        target: { node-index: 2 }
```

### Variable-driven targets

Task `config` is static YAML; to build entries from test variables, set the whole field via a `configVars` jq expression:

```yaml
- name: run_network_disruption
  title: "Black out the configured participant"
  configVars:
    disruptoorUrl: "disruptoorApiUrl"
    isolations: >-
      | [{
        name: "assertoor-blackout-target-cl",
        target: {"node-index": (.targetParticipantIndex | tonumber), "client-type": "beacon"},
        scope: ["cl_p2p", "el_p2p", "include_control"]
      }]
  config: {}
```

### Two-way network split

```yaml
- name: run_network_disruption
  title: "Split the network in half"
  config:
    disruptoorUrl: "http://disruptoor:7700"
    partitions:
      - name: fork-split
        groups:
          - { node-index: [1, 2] }
          - { node-index: [3, 4] }
        scope: [cl_p2p, el_p2p]
```

### Compose disruptions with update

Keep a baseline jitter active while toggling a blackout on and off:

```yaml
- name: run_network_disruption
  title: "Add blackout on top of existing disruptions"
  config:
    disruptoorUrl: "http://disruptoor:7700"
    action: update
    isolations:
      - name: blackout-node-3
        target: { node-index: 3 }

- name: run_network_disruption
  title: "Remove only the blackout"
  config:
    disruptoorUrl: "http://disruptoor:7700"
    action: update
    removeNames: [blackout-node-3]
```

### Cleanup task

```yaml
cleanupTasks:
  - name: run_network_disruption
    title: "Heal all network disruptions"
    config:
      disruptoorUrl: "http://disruptoor:7700"
      action: clear
```
