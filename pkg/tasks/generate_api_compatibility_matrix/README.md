## `generate_api_compatibility_matrix` Task

### Description

Walks the test run's task list, picks up every task whose name matches
`sourceTaskName` (default `check_consensus_api`), and renders a markdown
compatibility matrix.

Two artifacts are produced:

- `summary` — the rendered markdown is stored as the task's summary so
  the assertoor UI can render it inline in the task pane.
- `result/matrix.md` and `result/matrix.json` — downloadable result
  files (the markdown render and a machine-readable JSON dump).

The aggregator looks for these outputs on every collected source task:

| Output         | Used as                                       |
|----------------|-----------------------------------------------|
| `checkId`      | Stable row id (and ordering key)              |
| `checkTitle`   | Row label in the matrix                       |
| `referenceUrl` | (Reserved for future linking)                 |
| `matrixRow`    | Map of `clientType → {result, note, httpStatus}` |

### Configuration

```yaml
- name: generate_api_compatibility_matrix
  config:
    title: "GLOAS Beacon-API Compatibility"
    description: |
      Compatibility matrix for the new GLOAS beacon API endpoints and
      events across the connected consensus clients.
    includeCheckIds:
      - publish_block
      - produce_block_v4
      - publish_bid
      - get_bid
      - publish_envelope
      - get_envelope_by_block
      - get_envelope_by_slot
      - ptc_duties
      - payload_attestation_data
      - post_payload_attestations
      - get_payload_attestations
      - sse_execution_payload_bid
      - sse_execution_payload_available
      - sse_payload_attestation_message
      - sse_execution_payload
      - sse_execution_payload_gossip
```

### Parameters

| Field                | Type     | Description                                                                                       |
|----------------------|----------|---------------------------------------------------------------------------------------------------|
| `title`              | string   | Matrix title (rendered as H1).                                                                    |
| `description`        | string   | Free-form markdown rendered between the title and the table.                                      |
| `includeCheckIds`    | []string | Restrict + order the rows by checkId. Default: all completed source tasks in scheduler order.     |
| `sourceTaskName`     | string   | Task name to pick rows from. Default `check_consensus_api`.                                       |
| `clientOrder`        | []string | Column ordering by client-type. Default `[lighthouse, teku, prysm, grandine, nimbus, lodestar, caplin]`. |
| `showAllClientTypes` | bool     | If true, render columns even for client-types that no test reached. Default false.                |
| `emojiPass`          | string   | Default `✅`.                                                                                     |
| `emojiPartial`       | string   | Default `🟡`.                                                                                     |
| `emojiFail`          | string   | Default `❌`.                                                                                     |
| `emojiSkipped`       | string   | Default `⚪`.                                                                                     |
| `emojiAbsent`        | string   | Default `—`.                                                                                     |
| `includeFootnotes`   | bool     | Render numbered footnotes for cells with notes. Default true.                                     |
| `includeLegend`      | bool     | Render the legend below the matrix. Default true.                                                |
| `failOnFailures`     | bool     | If true, fail the task whenever any cell is `fail`.                                              |

### Outputs

- `matrixMarkdown` — the rendered markdown string
- `passRate` — fraction of pass cells out of (pass + partial + fail)
- `cellCounts` — counts per status

### Result artifacts

The task writes the same markdown to two places:

- As the task's **summary** (`Type: summary`, `Name: matrix.md`) so the
  UI renders it inline at the top of the task details pane.
- As a regular **result file** (`Type: result`, `Name: matrix.md`) so
  it's downloadable.

Additionally `result/matrix.json` provides a JSON dump for machine
consumption.
