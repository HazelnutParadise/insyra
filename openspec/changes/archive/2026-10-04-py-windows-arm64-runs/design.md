# Design: py-windows-arm64-runs

## Context

polars 1.44.2 decides whether it can read x86 feature flags from `platform.machine()` (`_SUPPORTS_CPUID` in `polars/_cpu_check.py`). On Windows, CPython 3.12 answers that from the processor itself through WMI, so an x86-64 interpreter running under emulation on an ARM64 processor gets `ARM64`, reads no flags, and `check_cpu_flags` raises `unknown feature flag` for the first flag of its x86-64 build. The check is skipped when `POLARS_SKIP_CPU_CHECK` is set.

## Decisions

### The guard runs in Python, on Python's own view of the machine

The condition is checked by the script, not by the Go side, because it is about the interpreter: the same `platform.machine()` call polars makes, together with `sysconfig.get_platform()`, which names the interpreter's own build. That also covers an x86-64 Go program running under emulation on ARM64 Windows, where `runtime.GOARCH` is `amd64` and the Go side could not tell.

### Only for an x86-64 Python on ARM64 Windows

Everywhere else polars' check means something: on an x86-64 processor without the features polars was built for, it warns before a crash. So the guard needs all three of `os.name == "nt"`, an `arm64` machine and a `win-amd64` interpreter, and a native arm64 Python, which installs polars' `win_arm64` build, keeps the check. `setdefault` leaves a value the caller set, such as `0`, alone.

### Before every import

The preamble imports its packages in the order of a Go map, so the guard is placed before the whole import block rather than next to `import polars as pl`. A test checks it comes first.

### Checked on Windows arm64 on every change to the package

`TestPinnedEnvironmentEndToEnd` already builds the environment from nothing and runs Python through the package, so a workflow runs it on `windows-11-arm`. It is triggered by any change under `py/`, not only by the pins, because what it checks is whether the package works on that platform, and the environment there installs wheels only, so a run takes minutes. The test skips itself without `INSYRA_PY_E2E`, so the workflow's last step fails unless it is seen to pass.

## Risks

- Skipping the check does not make polars' x86-64 code run: it needs the AVX2 and related instructions its build targets, which Windows' emulator provides on the runner's Windows 11. An older emulator without them would crash in polars instead of raising at import. A native arm64 Python is the fallback the owner agreed to if the emulator cannot run it.
- Code runs under emulation, slower than on a native Python.
