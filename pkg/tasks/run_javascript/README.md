## `run_javascript` Task

### Description

Executes a JavaScript snippet via Node.js. Suited to data-shaping
problems where shell + jq becomes unreadable (rendering a markdown
matrix from a set of sibling task outputs, generating a JSON config
from chain state, etc.). Mirrors `run_shell`'s I/O protocol:

- `envVars` are JSON-encoded into process env vars and additionally
  parsed back into a JS object named `env` for direct use.
- Stdout markers `::set-var`, `::set-json`, `::set-output`,
  `::set-output-json` set runtime variables / task outputs (same as
  `run_shell`).
- `$ASSERTOOR_RESULT_DIR` and `$ASSERTOOR_SUMMARY` are exposed; files
  written under them are stored as task result artifacts.
- The user script is wrapped in `(async () => { ... })()` so top-level
  `await` is supported.

The script runs against the system Node.js (default `node` on `PATH`,
overridable via `nodePath`).

### Configuration

```yaml
- name: run_javascript
  config:
    envVars:
      ROWS: "| [ tasks | to_entries[] | .value.outputs | select(.matrixRow != null) ]"
    script: |
      const rows = env.ROWS;
      const md = rows.map(r => `- ${r.rowId}: ${JSON.stringify(r.matrixRow)}`).join('\n');
      writeResultFile('matrix.md', md);
      setOutputJSON('rowCount', rows.length);
```

### Parameters

| Field      | Type                | Description                                                                                                  |
|------------|---------------------|--------------------------------------------------------------------------------------------------------------|
| `script`   | string (required)   | The JavaScript source to execute.                                                                            |
| `envVars`  | map[string]string   | Runtime variable queries (same syntax as `run_shell.envVars`). Each is JSON-encoded into a process env var.  |
| `nodePath` | string              | Node.js binary path. Default `node` (resolved from `PATH`).                                                  |
| `nodeArgs` | []string            | Extra arguments passed to node before the script path.                                                       |

### Provided globals

| Name                  | Description                                                                                       |
|-----------------------|---------------------------------------------------------------------------------------------------|
| `env`                 | Object whose keys are your `envVars`; each value is the JSON-decoded form (falls back to string). |
| `SUMMARY_FILE`        | String path to the summary file (also `process.env.ASSERTOOR_SUMMARY`).                           |
| `RESULT_DIR`          | String path to the result-file directory.                                                         |
| `setVar(name, v)`     | Set a runtime variable (string).                                                                  |
| `setVarJSON(name, v)` | Set a runtime variable (JSON).                                                                    |
| `setOutput(name, v)`  | Set a task output (string).                                                                       |
| `setOutputJSON(n, v)` | Set a task output (JSON).                                                                         |
| `writeSummary(s)`     | Convenience: write to the summary file.                                                           |
| `writeResultFile(name, content)` | Convenience: write a file under the result directory.                                  |
