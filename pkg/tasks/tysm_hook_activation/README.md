## `tysm_hook_activation` Task

### Description

Creates a TTL-bound activation against a TYSM beacon node's hook-control API
(`POST /tysm/v1/activations`).

An activation overlays `(enabled, configPatch)` on the hook's baseline state
for a bounded duration. When the TTL expires (or the activation is deleted
via `tysm_hook_deactivation`), the hook reverts to its baseline. Designed
for chaos-style flows where assertoor flips a hook to a non-default state,
exercises the network, and then cleans up.

This task is a paired primitive: pair it with `tysm_hook_deactivation`
placed in the test's top-level `cleanupTasks:` block so the activation is
torn down even if the test fails.

#### Task Behavior

- POSTs the activation request and returns immediately on `201 Created`.
- Records `activation_id` and `expires_at` as task outputs for later use
  (typically by a deactivation task in cleanup).
- Fails on any non-`201` response, surfacing the server's error message.
- Does not wait for or assert anything about hook side-effects — that is
  the responsibility of subsequent tasks in the playbook.

### Configuration Parameters

- **`endpoint`**:\
  Base URL of the TYSM API, e.g. `http://beacon:8080`. Required.

- **`auth_token`**:\
  Bearer token sent in the `Authorization` header. Required when the TYSM
  API has auth enabled. Recommended to supply via `configVars` so the
  secret is not hard-coded into the playbook.

- **`hook`**:\
  Name of the hook to activate. Must be a hook implementing
  `RuntimeReconfigurable` on the server side (currently `blob-mutator`,
  `data-column-mutator`); other names are rejected with `400`.

- **`enabled`**:\
  Optional boolean override of the hook's enabled flag while the activation
  is in force. Either this or `configPatch` (or both) must be supplied.

- **`configPatch`**:\
  Optional shallow patch over the hook's baseline configuration. Top-level
  keys present here wholly replace the corresponding baseline keys; absent
  keys keep their baseline value.

- **`duration`**:\
  Activation TTL as a Go duration string (`10m`, `1h`, ...). Required. The
  server enforces a hard cap (`api.max_activation_duration`); requests
  exceeding it are rejected.

- **`replace`**:\
  If `true`, replace any existing activation against the same hook instead
  of returning `409 Conflict`. Default `false`.

### Defaults

```yaml
- name: tysm_hook_activation
  config:
    endpoint: ""
    auth_token: ""
    hook: ""
    enabled: null
    configPatch: {}
    duration: "0s"
    replace: false
```

### Outputs

| Name             | Type     | Description                                                                  |
|------------------|----------|------------------------------------------------------------------------------|
| `activation_id`  | `string` | Server-assigned activation ID. Pass to `tysm_hook_deactivation`.             |
| `expires_at`     | `string` | RFC3339 timestamp at which the server-side TTL expires.                      |
| `hook`           | `string` | Hook the activation targets (echoes the input).                              |

### Example

```yaml
tests:
  - id: kzg_chaos_run
    name: "blob-mutator KZG chaos"
    cleanupTasks:
      - name: tysm_hook_deactivation
        config:
          endpoint: "http://beacon:8080"
        configVars:
          auth_token: "tysmApiToken"
          activation_id: "tasks.kzg_chaos.outputs.activation_id"
    tasks:
      - name: tysm_hook_activation
        id: kzg_chaos
        config:
          endpoint: "http://beacon:8080"
          hook: "blob-mutator"
          enabled: true
          configPatch:
            mutationProbability: 1.0
            enabledStrategies: ["kzg-corruption"]
          duration: "10m"
          replace: false
        configVars:
          auth_token: "tysmApiToken"

      # ... assertions about network behaviour while activation is in force ...
```
