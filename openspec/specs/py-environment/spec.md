# py-environment Specification

## Purpose
How the Python environment the `py` package runs code in is pinned and verified: which uv, Python and package versions it installs, where those pins live, how an installed environment is kept in step with them, and what `ReinstallPyEnv` deletes.
## Requirements
### Requirement: uv is a pinned, verified download

The Python environment SHALL be built with the uv version pinned in `py/environment/pyproject.toml` (`[tool.uv] required-version`), downloaded as that release's archive for the platform from `https://github.com/astral-sh/uv/releases/download/<version>/`. The archive's SHA-256 SHALL equal the checksum for it in `py/environment/uv-sha256.sum`, a byte-for-byte copy of the release's published `sha256.sum`; an archive that does not match SHALL be refused with an error and SHALL leave no executable behind. The executable SHALL be kept in `.insyra_env/uv-<version>_<os>_<arch>/` beside the environment directory and SHALL be the only uv the package runs; no install script SHALL run and a `uv` on `PATH` SHALL NOT be used. macOS, Linux and Windows on amd64 and arm64 SHALL be supported, Linux with the static musl build; any other platform SHALL be an error naming it.

#### Scenario: Matching archive
- **WHEN** the archive served for the pinned version has the checksum recorded for it
- **THEN** the uv executable inside it, `uv` in a `.tar.gz` or `uv.exe` in a `.zip`, is written to its destination and is executable

#### Scenario: Tampered archive
- **WHEN** the archive served has a different SHA-256
- **THEN** the setup returns an error naming both checksums, and nothing is written at the destination

#### Scenario: Unsupported platform
- **WHEN** the platform is not one of the six supported
- **THEN** the setup returns an error naming the platform

### Requirement: The pins live in one directory and agree

`py/environment/` SHALL hold every version the package installs: the uv version and the exact Python version (`requires-python = "==X.Y.Z"`) in `pyproject.toml`, every package the Python preamble imports as an exact `name==version` dependency, every tool a source build installs as an exact `name==version` entry of `[tool.uv] build-constraint-dependencies` with the SHA-256 of every file of that version, the full resolution with file hashes and the build constraints with their hashes in `uv.lock`, and the uv checksums in `uv-sha256.sum`. The lock SHALL record the same Python requirement, the same version for every pinned package and the same build constraints with the same hashes, SHALL take every package from `https://pypi.org/simple`, and SHALL hold a wheel of every package for every supported platform except the source builds the tests list; a build tool the environment also installs SHALL be pinned at the version the environment runs; `uv-sha256.sum` SHALL hold a checksum for the archive of every supported platform.

#### Scenario: Pins checked by the tests
- **WHEN** `go test ./py/` runs
- **THEN** it fails if a package the preamble imports is not pinned exactly, if the lock disagrees with a pin or the Python requirement, takes a package from another index, or lacks a wheel it is expected to have, if either version cannot be read from `pyproject.toml`, or if a supported platform has no checksum
- **AND** it fails if a build constraint is not an exact `name==version` with at least one SHA-256, if the lock does not record the same build constraints with the same hashes, or if a build tool the environment also installs is pinned at another version

### Requirement: The environment follows the pins

The setup SHALL write the embedded `pyproject.toml` and `uv.lock` into the environment directory and run `uv sync --frozen --inexact --managed-python --python <pinned version>` there, with `UV_PROJECT_ENVIRONMENT` set to the directory's `.venv`, `UV_PYTHON_DOWNLOADS` set to `automatic`, and any `UV_PYTHON_PREFERENCE` of the caller removed. After a successful sync it SHALL record the pin set's fingerprint in a marker file; a later setup whose marker matches and whose interpreter exists SHALL run no uv command. A setup whose marker is missing or different SHALL sync again, so an environment synced to other package pins of the same Python is brought to the pins while keeping packages installed with `PipInstall`. A setup that fails SHALL NOT mark the environment ready or write the marker, and the next call SHALL run it again.

#### Scenario: First setup
- **WHEN** the environment directory is empty
- **THEN** the setup writes both files, runs `uv sync` once with those flags, and writes the marker

#### Scenario: Already in sync
- **WHEN** the marker matches the embedded pin set and the interpreter exists
- **THEN** the setup runs no uv command

#### Scenario: Pins changed
- **WHEN** the marker holds another pin set's fingerprint
- **THEN** the setup runs `uv sync` again and rewrites the marker

#### Scenario: The user's uv settings
- **WHEN** the caller's environment sets `UV_PYTHON_PREFERENCE`, `UV_PYTHON_DOWNLOADS=never` or `UV_PROJECT_ENVIRONMENT`
- **THEN** none of them reaches `uv sync`, and the virtual environment is built in the environment directory

#### Scenario: Sync fails
- **WHEN** `uv sync` exits with an error
- **THEN** the setup returns an error carrying uv's error output, writes no marker, and the next call runs the setup again

### Requirement: The caller's context bounds the setup

`RunCodeContext`, `RunCodefContext`, `RunFileContext` and `RunFilefContext` SHALL pass their context to the setup. When the context ends during the uv download, during `uv sync`, or while the call waits for another call's setup, the setup SHALL stop and the call SHALL return an error for which `errors.Is(err, ctx.Err())` holds; a later call SHALL run the setup again.

#### Scenario: Deadline during the sync
- **WHEN** the context's deadline passes while `uv sync` runs
- **THEN** the call returns promptly with `context.DeadlineExceeded`, and the environment is not marked ready

### Requirement: ReinstallPyEnv deletes the environment directory

`ReinstallPyEnv` SHALL hold the setup lock, mark the environment not ready, delete the environment directory, `.insyra_env/py26a_<os>_<arch>` under the working directory, with everything in it, packages installed with `PipInstall` included, and build it again from the pins. It SHALL keep the uv executable, which lives outside that directory. Its doc comment and `Docs/py.md` SHALL say what it deletes.

#### Scenario: The delete fails part-way
- **WHEN** part of the environment directory cannot be deleted
- **THEN** `ReinstallPyEnv` returns an error and the environment is not marked ready

#### Scenario: A file added to the environment
- **WHEN** a file was added to the environment directory and `ReinstallPyEnv` runs
- **THEN** the file is gone, the environment is synced and marked, and the uv executable is still in place

### Requirement: Setup prepares the environment ahead of first use

`py.Setup(ctx context.Context) error` SHALL do what the first call of any `py` function does to prepare the environment: download and verify the pinned uv if it is missing, and bring the environment directory to the pinned versions. It SHALL return the setup's error. Once it succeeds, later calls SHALL NOT prepare the environment again, and a `Setup` on a prepared environment SHALL return nil without running uv. `ctx` SHALL bound the download, the sync and the wait for another call's setup; a nil `ctx` SHALL be an error, and a `ctx` that is already done SHALL return `ctx.Err()` before anything starts. Calling `Setup` SHALL stay optional: without it, the first call prepares the environment as before.

#### Scenario: Setup before the first call
- **WHEN** `Setup(ctx)` is called on an environment directory that was never prepared
- **THEN** uv sync runs once and the environment is marked ready, and a later `pyEnvInit` runs no uv command

#### Scenario: Setup twice
- **WHEN** `Setup(ctx)` is called on an environment that is already prepared
- **THEN** it returns nil and runs no uv command

#### Scenario: A failing setup
- **WHEN** the sync fails
- **THEN** `Setup` returns the error, the environment is not marked ready, and the next `Setup` runs the setup again

#### Scenario: Nil or finished context
- **WHEN** `Setup` is called with a nil context, or with a cancelled one on an environment that was never prepared
- **THEN** it returns `errNilContext` or `context.Canceled`, and the environment directory is not touched

### Requirement: The environment directory is named for its Python

The environment directory SHALL be `.insyra_env/<code>_<os>_<arch>` under the working directory, where the code is `py<two-digit year><letter>` and SHALL change whenever the pinned Python version changes: the letter is `a` for the year's first new code, `b` for the second. `py/const.go` SHALL hold the code (`envDirCode`) and the Python version it was given for (`envDirPython`), and a test SHALL fail when the pinned Python differs from `envDirPython`. A change of package versions alone SHALL keep the code. The code for CPython 3.12.14 SHALL be `py26a`. insyra SHALL NOT delete the directory of an earlier code.

#### Scenario: The current directory
- **WHEN** the environment is prepared with CPython 3.12.14 pinned
- **THEN** it is built in `.insyra_env/py26a_<os>_<arch>`

#### Scenario: A Python bump without a new code
- **WHEN** `requires-python` in `py/environment/pyproject.toml` is changed and `envDirPython` is not
- **THEN** `go test ./py/` fails, naming both versions and asking for a new directory code

### Requirement: A source build uses only pinned, verified tools

When uv builds a locked package from source, every tool the build installs SHALL be one the build constraints pin, at that version, and its download SHALL match one of the constraint's hashes, or the setup SHALL fail. Because uv installs a build tool that no constraint names without an error, a test gated on `INSYRA_PY_E2E=1` SHALL build every package the tests list as a source build, with the pinned uv, an empty cache and `--no-binary-package`, and fail if a build installed a tool the constraints do not pin at that version. A workflow SHALL run that test on macOS whenever a file that decides it changes (anything under `py/environment/`, `py/environment.go`, `py/environment_build_test.go`, `py/environment_pins_test.go` or the workflow), and SHALL fail unless the test is seen to pass.

#### Scenario: A tampered build tool
- **WHEN** a build tool's download does not match the hashes the lock records for it
- **THEN** uv refuses it and the setup fails with uv's `Hash mismatch` error

#### Scenario: A build tool the constraints do not name
- **WHEN** a source build installs a tool that is not in the build constraints, or at another version
- **THEN** the gated test fails naming that tool and version

#### Scenario: The pins change
- **WHEN** a push or pull request changes a file under `py/environment/`, or `knownSourceBuilds`
- **THEN** the `Python Build Tools` workflow builds `blis` and `statsmodels` from source on macOS, runs the gated test, and fails if it skipped or did not run

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

