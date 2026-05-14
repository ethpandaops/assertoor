## `check_consensus_api` Task

### Description

Probes a single beacon-API endpoint or SSE topic against every connected
consensus client (with optional include/exclude filters), classifies each
response (pass / partial / fail / skipped), and emits both per-client
results and a "matrix row" aggregated by client-type. The output is
consumed by `generate_api_compatibility_matrix` to render a markdown
compatibility table.

It is the building block of an API-compatibility playbook: instantiate
one `check_consensus_api` task per endpoint you want to cover, then
finish with a single `generate_api_compatibility_matrix` task.

#### Result classification

| Outcome      | Meaning                                                                                          |
|--------------|--------------------------------------------------------------------------------------------------|
| `pass`       | HTTP status is in `successStatuses` and `responseSchema` validates, OR status is in `errorStatuses` and `errorSchema` validates. For SSE: at least `minEvents` matching events received and all validate against `eventSchema`. |
| `partial`    | Endpoint exists (status is in `expectStatuses`) but body fails schema validation, OR SSE subscription opened but no events arrived in the window, OR HTTP status fell through to neither set. |
| `fail`       | HTTP status not in `expectStatuses`, connection error, or SSE subscription rejected.             |
| `skipped`    | Required fork (`requireForkActive`) is not yet active on this client.                            |

### Configuration

```yaml
- name: check_consensus_api
  config:
    checkId: "pool_payload_attestations_get"
    checkTitle: "GET /eth/v1/beacon/pool/payload_attestations"
    referenceUrl: "https://github.com/ethereum/beacon-APIs/pull/552"

    method: "GET"
    path: "/eth/v1/beacon/pool/payload_attestations"
    queryParams: {}
    headers: {}

    expectStatuses:  [200, 400, 404, 415, 503]
    successStatuses: [200]
    errorStatuses:   [400, 404, 415, 503]

    responseSchema:
      type: object
      required: [version, data]
      properties:
        version: { type: string }
        data:    { type: array }

    errorSchema:
      type: object
      required: [code, message]

    requireForkActive: "gloas"
```

#### Path placeholders

If `path` contains `{slot}`, `{epoch}`, `{block_id}`, `{state_id}`,
`{beacon_block_root}`, `{builder_index}`, or `{validator_index}` and the
value isn't in `pathParams`, the task resolves it from chain state
(head slot, head root, etc.) using the first ready CL endpoint.

Offsets are supported: `{slot+5}`, `{epoch-1}`.

#### SSE mode

Omit `method`/`path` and set `sse`:

```yaml
- name: check_consensus_api
  config:
    checkId: "sse_execution_payload"
    checkTitle: "SSE execution_payload"
    sse:
      topic: execution_payload
      timeoutSeconds: 36
      minEvents: 1
    eventSchema:
      type: object
      required: [version, data]
```

### Parameters (full list)

| Field                  | Type                  | Description                                                                                                                                                                                |
|------------------------|-----------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `checkId`              | string (required)     | Stable identifier for the check. Used by the aggregator.                                                                                                                                   |
| `checkTitle`           | string                | Display title used in the matrix.                                                                                                                                                          |
| `referenceUrl`         | string                | URL to the spec PR / docs.                                                                                                                                                                 |
| `clientPattern`        | string (regex)        | Restrict to clients whose name matches.                                                                                                                                                    |
| `excludeClientPattern` | string (regex)        | Exclude clients whose name matches.                                                                                                                                                        |
| `method`               | string                | HTTP method (default `GET`). Ignored when `sse` is set.                                                                                                                                    |
| `path`                 | string                | Endpoint path with optional `{placeholders}`.                                                                                                                                              |
| `pathParams`           | map[string]string     | Explicit placeholder overrides.                                                                                                                                                            |
| `queryParams`          | map[string]string     | Query parameters.                                                                                                                                                                          |
| `headers`              | map[string]string     | Extra headers.                                                                                                                                                                             |
| `body`                 | any                   | JSON request body.                                                                                                                                                                         |
| `bodyRaw`              | string                | Raw bytes (overrides `body` when set).                                                                                                                                                     |
| `sse.topic`            | string                | SSE topic name (required for SSE).                                                                                                                                                         |
| `sse.timeoutSeconds`   | int                   | How long to wait for events. Default `36`.                                                                                                                                                 |
| `sse.minEvents`        | int                   | Required event count for pass. Default `1`.                                                                                                                                                |
| `sse.eventName`        | string                | Override SSE `event:` name filter. Defaults to `topic`.                                                                                                                                    |
| `expectStatuses`       | []int                 | HTTP statuses considered "endpoint exists". Default `[200, 400, 404, 415, 503]`.                                                                                                          |
| `successStatuses`      | []int                 | Statuses where `responseSchema` is applied. Default `[200]`.                                                                                                                              |
| `errorStatuses`        | []int                 | Statuses where `errorSchema` is applied. Default `[400, 404, 415, 503]`.                                                                                                                  |
| `responseSchema`       | map (JSON Schema)     | Inline JSON Schema for success responses.                                                                                                                                                  |
| `errorSchema`          | map (JSON Schema)     | Inline JSON Schema for documented error responses (typically the `ErrorMessage` shape).                                                                                                    |
| `eventSchema`          | map (JSON Schema)     | Inline JSON Schema for SSE event payloads.                                                                                                                                                 |
| `requireForkActive`    | string                | If set (e.g. `"gloas"`), each CL where this fork isn't active records `skipped`.                                                                                                          |
| `ignoreSchema`         | bool                  | Skip schema validation entirely — any status in `expectStatuses` passes.                                                                                                                  |
| `requestTimeout`       | duration              | Per-client request timeout. Default `30s`.                                                                                                                                                |
| `overallTimeout`       | duration              | Overall wallclock budget across all clients. Default `90s`.                                                                                                                               |
| `concurrency`          | int                   | Max parallel client probes. Default `6`.                                                                                                                                                  |
| `failOnAllError`       | bool                  | If true, task fails when no client passes.                                                                                                                                                |
| `failOnAnyError`       | bool                  | If true, task fails when any client fails or partial.                                                                                                                                     |

### Outputs

- `results` — array of per-client objects: `{client, clientType, status, httpStatus, durationMs, note, error, schemaErrors, eventCount}`
- `matrixRow` — `map[clientType]{result, note, httpStatus}` collapsed by client type (worst-case status when multiple clients of the same type are tested)
- `passCount`, `partialCount`, `failCount`, `skippedCount`, `totalCount` — integer counters
- `checkId`, `checkTitle`, `referenceUrl` — echoed for downstream aggregation
