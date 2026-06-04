## `run_python_uv` Task

### Description

Initializes a [uv](https://github.com/astral-sh/uv)-managed Python
virtual environment that is shared by subsequent `run_python` tasks in
the same test. The venv path is exposed via a runtime variable
(default `python_uv_path`), and a cleanup task is prepended to the
cleanup queue so the venv is removed when the test ends (LIFO order).

Call this task explicitly when you need a pinned Python version or
preinstalled packages. If you don't, plain `run_python` tasks will
auto-initialize an empty venv on first use.

### Configuration

```yaml
- name: run_python_uv
  config:
    pythonVersion: "3.12"
    requirements:
      - web3
      - requests>=2.32

- name: run_python
  config:
    script: |
      from web3 import Web3
      ...
```

### Parameters

| Field           | Type      | Description                                                                                       |
|-----------------|-----------|---------------------------------------------------------------------------------------------------|
| `uvPath`        | string    | Path to the `uv` binary. Default `uv`.                                                            |
| `pythonVersion` | string    | Optional Python version pin (e.g. `3.12`). uv will download the matching CPython if missing.      |
| `requirements`  | []string  | Pip-installable specs to install into the venv (e.g. `['web3', 'requests>=2.32']`).               |
| `venvVar`       | string    | Variable to populate with the venv path. Default `python_uv_path`.                                |
| `skipIfSet`     | bool      | If true, do nothing when `venvVar` is already populated. Default `true`.                          |

### Behavior

1. If `skipIfSet` is `true` and `venvVar` already resolves to a
   non-empty string, the task no-ops.
2. Otherwise, a temp directory is created and `uv venv [--python X]` is
   run inside it.
3. If `requirements` is non-empty, `uv pip install --python <venv>/bin/python <reqs...>` is run.
4. The venv path is written to the variable named by `venvVar`.
5. A cleanup task (`run_shell` with `rm -rf <venvDir>`) is **prepended**
   to the cleanup queue, so cleanup runs in reverse order of setup.
