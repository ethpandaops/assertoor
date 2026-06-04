## `run_python` Task

### Description

Executes a Python snippet. Mirrors `run_shell` / `run_javascript`'s I/O protocol:

- `envVars` are JSON-encoded into process env vars and additionally parsed
  back into a `env` dict for direct use.
- Stdout markers `::set-var`, `::set-json`, `::set-output`,
  `::set-output-json` set runtime variables / task outputs.
- `$ASSERTOOR_RESULT_DIR`, `$ASSERTOOR_SUMMARY`, and `$ASSERTOOR_TEST_RESULT`
  are exposed; files written under them are persisted as task / test
  result artifacts.
- The user script runs inside an `async def` wrapper, so top-level
  `await` is supported.

When `useUv` is true (default), the script runs inside the
uv-managed venv referenced by `venvVar`. If that variable is not yet
set, `run_python` auto-initializes a default venv by spawning a child
`run_python_uv` task, which registers a cleanup task at the front of the
cleanup queue (LIFO teardown). Use `run_python_uv` directly when you
need to pin a Python version or pre-install packages.

### Configuration

```yaml
- name: run_python
  config:
    requirements: ["web3"]
    envVars:
      RPC: "endpoints[0].rpcUrl"
    script: |
      from web3 import Web3
      w3 = Web3(Web3.HTTPProvider(env["RPC"]))
      set_output_json("chainId", w3.eth.chain_id)
```

### Parameters

| Field          | Type                | Description                                                                                       |
|----------------|---------------------|---------------------------------------------------------------------------------------------------|
| `script`       | string (required)   | The Python source to execute.                                                                     |
| `envVars`      | map[string]string   | Runtime variable queries. Each is JSON-encoded into a process env var.                            |
| `pythonPath`   | string              | Fallback interpreter when no uv venv is available. Default `python3`.                             |
| `pythonArgs`   | []string            | Extra arguments passed to python before the script path.                                          |
| `useUv`        | bool                | If true, run inside the uv venv referenced by `venvVar`. Default `true`.                          |
| `venvVar`      | string              | Variable name holding the venv path. Default `python_uv_path`.                                    |
| `uvPath`       | string              | Path to the `uv` binary used for auto-init / requirements install. Default `uv`.                  |
| `requirements` | []string            | Pip-installable specs to ensure are present in the venv before running.                           |

### Provided globals

| Name                            | Description                                                                                |
|---------------------------------|--------------------------------------------------------------------------------------------|
| `env`                           | Dict whose keys are your `envVars`; values are JSON-decoded (falls back to string).        |
| `SUMMARY_FILE`                  | String path to the summary file (also `os.environ['ASSERTOOR_SUMMARY']`).                  |
| `RESULT_DIR`                    | String path to the result-file directory.                                                  |
| `TEST_RESULT_FILE`              | String path to the shared per-test-run markdown file.                                      |
| `set_var(name, v)`              | Set a runtime variable (string).                                                           |
| `set_var_json(name, v)`         | Set a runtime variable (JSON).                                                             |
| `set_output(name, v)`           | Set a task output (string).                                                                |
| `set_output_json(name, v)`      | Set a task output (JSON).                                                                  |
| `write_summary(s)`              | Convenience: write to the summary file.                                                    |
| `write_result_file(name, c)`    | Convenience: write a file under the result directory.                                      |
| `write_test_result(s)`          | Overwrite the shared per-test-run markdown.                                                |
| `append_test_result(s)`         | Append to the shared per-test-run markdown.                                                |
