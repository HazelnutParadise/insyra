## ADDED Requirements

### Requirement: Python runs on Windows on arm64

The script `py` generates SHALL begin, before any import, with a guard that sets `POLARS_SKIP_CPU_CHECK` to `1` through `os.environ.setdefault` exactly when `os.name` is `nt`, `platform.machine()` in lower case is `arm64` and `sysconfig.get_platform()` is `win-amd64`, so that the x86-64 CPython uv installs on Windows on arm64 by default, which Windows runs through emulation, can import polars. A workflow SHALL run `TestPinnedEnvironmentEndToEnd` on GitHub's Windows arm64 runner whenever anything under `py/` or the workflow changes, and SHALL fail unless the test is seen to pass.

#### Scenario: An x86-64 Python on ARM64 Windows
- **WHEN** the script runs in an interpreter whose `sysconfig.get_platform()` is `win-amd64` on a machine `platform.machine()` reports as `ARM64`
- **THEN** `POLARS_SKIP_CPU_CHECK` is `1` before polars is imported

#### Scenario: Any other interpreter or platform
- **WHEN** the interpreter is a native arm64 Python on Windows, an x86-64 Python on an x86-64 machine, or any Python on another operating system
- **THEN** the guard leaves `POLARS_SKIP_CPU_CHECK` unset and polars runs its check

#### Scenario: The caller set the variable
- **WHEN** `POLARS_SKIP_CPU_CHECK` is already set, to any value
- **THEN** the guard keeps that value

#### Scenario: A change to the package
- **WHEN** a push or pull request changes a file under `py/`
- **THEN** the `Python on Windows arm64` workflow builds the environment on `windows-11-arm`, runs Python through the package, and fails if the test skipped or did not run
