package runpython

// preamble is prepended to every user script. It:
//   - parses each env var (set from envVars config) as JSON when possible,
//     exposing the result as the global 'env' dict;
//   - exposes ASSERTOOR_SUMMARY / ASSERTOOR_RESULT_DIR / ASSERTOOR_TEST_RESULT
//     as constants;
//   - provides 'set_output[_json]', 'set_var[_json]', 'write_result_file',
//     'write_summary', 'write_test_result', and 'append_test_result' helpers
//     that emit the same '::set-*' markers as run_shell / run_javascript and
//     write to the shared task dirs.
//
// The user's script runs inside an async coroutine so 'await' works at
// the top level.
const preamble = `
import asyncio
import json
import os
import sys
import traceback

SUMMARY_FILE = os.environ.get("ASSERTOOR_SUMMARY", "")
RESULT_DIR = os.environ.get("ASSERTOOR_RESULT_DIR", "")
TEST_RESULT_FILE = os.environ.get("ASSERTOOR_TEST_RESULT", "")
_env_keys = [k for k in os.environ.get("__ASSERTOOR_ENV_KEYS", "").split(",") if k]

env = {}
for _k in _env_keys:
    _raw = os.environ.get(_k)
    if _raw is None:
        continue
    try:
        env[_k] = json.loads(_raw)
    except (ValueError, TypeError):
        env[_k] = _raw


def set_var(name, value):
    print("::set-var " + name + " " + str(value), flush=True)


def set_var_json(name, value):
    print("::set-json " + name + " " + json.dumps(value), flush=True)


def set_output(name, value):
    print("::set-output " + name + " " + str(value), flush=True)


def set_output_json(name, value):
    print("::set-output-json " + name + " " + json.dumps(value), flush=True)


def write_result_file(name, content):
    if not RESULT_DIR:
        raise RuntimeError("ASSERTOOR_RESULT_DIR is not set")
    dest = os.path.join(RESULT_DIR, name)
    os.makedirs(os.path.dirname(dest) or RESULT_DIR, exist_ok=True)
    mode = "wb" if isinstance(content, (bytes, bytearray)) else "w"
    with open(dest, mode) as fh:
        fh.write(content)


def write_summary(content):
    if not SUMMARY_FILE:
        raise RuntimeError("ASSERTOOR_SUMMARY is not set")
    with open(SUMMARY_FILE, "w") as fh:
        fh.write(content)


# write_test_result and append_test_result target the shared per-test-run
# markdown file. Anything written here shows up on the run page's Result
# panel.
def write_test_result(content):
    if not TEST_RESULT_FILE:
        raise RuntimeError("ASSERTOOR_TEST_RESULT is not set")
    with open(TEST_RESULT_FILE, "w") as fh:
        fh.write(content)


def append_test_result(content):
    if not TEST_RESULT_FILE:
        raise RuntimeError("ASSERTOOR_TEST_RESULT is not set")
    with open(TEST_RESULT_FILE, "a") as fh:
        fh.write(content)


async def __assertoor_main():
%s


try:
    asyncio.run(__assertoor_main())
except SystemExit:
    raise
except BaseException:
    traceback.print_exc()
    sys.exit(1)
`
