# AGENTS.md (py/)

Rules for working in `py/`. The top-level [AGENTS.md](/AGENTS.md) covers the wider repo.

## Known pitfalls

- To check the Python that `py/builtin.go` generates without building the pinned environment, write `builtInFunc(addr, id)` to a file from a throwaway test, run it with `uv run --no-project --python 3.12 --with <package>==<version> … python -I`, and call `insyra._normalize_result` and `json.dumps(default=insyra._encode_default)`; the gated end-to-end test, where the line has one, stays the regression test (0.4: `INSYRA_PY_E2E=1` `TestPinnedEnvironmentEndToEnd`, with a short `TMPDIR` for the IPC socket).
- Test the runner and the environment setup without Python or network by making the test binary the fake: `TestMain` in `py/main_test.go` dispatches on `INSYRA_PY_TEST_FAKE_PYTHON` (and on 0.4 `INSYRA_PY_TEST_FAKE_UV`), and the binary stands in as the Python path or the uv the setup runs, speaking the IPC protocol.
