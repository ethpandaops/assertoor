## `tysm_hook_deactivation` Task

### Description

Cancels a TYSM hook activation by issuing
`DELETE /tysm/v1/activations/{id}`. The hook reverts to its baseline
`(enabled, config)` state immediately on the server side.

This task is the cleanup half of the `tysm_hook_activation` /
`tysm_hook_deactivation` pair. The intended placement is in a test's
top-level `cleanupTasks:` block, so the deactivation runs even if the test
fails. Because the server-side TTL on the activation will eventually fire
on its own, this task tolerates a `404 Not Found` (treats it as success by
default — see `ignoreNotFound`).

### Configuration Parameters

- **`endpoint`**:\
  Base URL of the TYSM API, e.g. `http://beacon:8080`. Required.

- **`auth_token`**:\
  Bearer token sent in the `Authorization` header. Required when the TYSM
  API has auth enabled. Recommended to supply via `configVars`.

- **`activation_id`**:\
  ID of the activation to cancel. Required. Typically supplied via
  `configVars` from the upstream `tysm_hook_activation` task's outputs:
  `configVars: { activation_id: "tasks.<id>.outputs.activation_id" }`.

- **`ignoreNotFound`**:\
  If `true` (default), treat HTTP `404` as success on the assumption the
  server-side TTL already fired. Set to `false` if you want explicit
  failure when the activation is no longer present.

### Defaults

```yaml
- name: tysm_hook_deactivation
  config:
    endpoint: ""
    auth_token: ""
    activation_id: ""
    ignoreNotFound: true
```

### Outputs

This task does not produce any outputs.

### Example

```yaml
cleanupTasks:
  - name: tysm_hook_deactivation
    config:
      endpoint: "http://beacon:8080"
    configVars:
      auth_token: "tysmApiToken"
      activation_id: "tasks.kzg_chaos.outputs.activation_id"
```
