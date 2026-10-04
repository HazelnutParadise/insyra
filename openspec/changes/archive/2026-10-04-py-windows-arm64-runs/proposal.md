# Proposal: py-windows-arm64-runs

## Why

Every Python run failed on Windows on arm64. The build on GitHub's `windows-11-arm` runner that `py-pin-source-build-tools` set up (run 37141744239, 2026-10-03) prepared the environment and then stopped at `import polars`, which the preamble of every script runs, with `RuntimeError: unknown feature flag: 'sse3'`. uv installs the x86-64 CPython on Windows on arm64 by default, and Windows runs it through its x64 emulation. polars' CPU check reads the machine with `platform.machine()`, gets `ARM64`, so it reads no x86 feature flags, and raises on the first flag its `win_amd64` build expects.

The owner chose on 2026-10-04 to keep uv's default, which needs no compiler, over a native arm64 Python, which would have to compile `blis` and `statsmodels` and so need Visual Studio and an arm64 LLVM on every such machine.

## What Changes

- The script `py` generates begins, before any import, with a guard that sets `POLARS_SKIP_CPU_CHECK=1` through `os.environ.setdefault` when `os.name` is `nt`, `platform.machine()` is `ARM64` in any case and `sysconfig.get_platform()` is `win-amd64`: an x86-64 Python on ARM64 Windows. A value the caller set is kept, and every other platform keeps polars' check.
- A new workflow, `Python on Windows arm64`, runs `TestPinnedEnvironmentEndToEnd`, which builds the environment from nothing and runs Python through the `py` package, on GitHub's `windows-11-arm` runner whenever anything under `py/` or the workflow changes, and fails unless the test is seen to pass.
- `Docs/py.md` says Python runs under emulation on Windows on arm64 and why polars' check is skipped there.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `py-environment`: Python runs on Windows on arm64, checked on every change to the package.

## Impact

- Code: `py/py.go`.
- Tests: `py/polars_cpu_check_test.go` (new).
- CI: `.github/workflows/py-windows-arm64.yml` (new).
- Docs: `Docs/py.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `delivery-status.md`.
