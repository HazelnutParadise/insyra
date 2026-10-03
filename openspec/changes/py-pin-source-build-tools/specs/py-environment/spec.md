## MODIFIED Requirements

### Requirement: The pins live in one directory and agree

`py/environment/` SHALL hold every version the package installs: the uv version and the exact Python version (`requires-python = "==X.Y.Z"`) in `pyproject.toml`, every package the Python preamble imports as an exact `name==version` dependency, every tool a source build installs as an exact `name==version` entry of `[tool.uv] build-constraint-dependencies` with the SHA-256 of every file of that version, the full resolution with file hashes and the build constraints with their hashes in `uv.lock`, and the uv checksums in `uv-sha256.sum`. The lock SHALL record the same Python requirement, the same version for every pinned package and the same build constraints with the same hashes, SHALL take every package from `https://pypi.org/simple`, and SHALL hold a wheel of every package for every supported platform except the source builds the tests list; a build tool the environment also installs SHALL be pinned at the version the environment runs; `uv-sha256.sum` SHALL hold a checksum for the archive of every supported platform.

#### Scenario: Pins checked by the tests
- **WHEN** `go test ./py/` runs
- **THEN** it fails if a package the preamble imports is not pinned exactly, if the lock disagrees with a pin or the Python requirement, takes a package from another index, or lacks a wheel it is expected to have, if either version cannot be read from `pyproject.toml`, or if a supported platform has no checksum
- **AND** it fails if a build constraint is not an exact `name==version` with at least one SHA-256, if the lock does not record the same build constraints with the same hashes, or if a build tool the environment also installs is pinned at another version

## ADDED Requirements

### Requirement: A source build uses only pinned, verified tools

When uv builds a locked package from source, every tool the build installs SHALL be one the build constraints pin, at that version, and its download SHALL match one of the constraint's hashes, or the setup SHALL fail. Because uv installs a build tool that no constraint names without an error, a test gated on `INSYRA_PY_E2E=1` SHALL build every package some supported platform builds from source, with the pinned uv and an empty cache, and fail if a build installed a tool the constraints do not pin at that version. A workflow SHALL run that test and `TestPinnedEnvironmentEndToEnd` on GitHub's Windows arm64 runner whenever a file that decides the build or the checks changes (anything under `py/environment/`, `py/environment.go`, `py/init.go`, `py/const.go`, `py/environment_build_test.go`, `py/environment_pins_test.go`, `py/environment_setup_test.go` or the workflow), and SHALL fail unless both tests are seen to pass.

#### Scenario: A tampered build tool
- **WHEN** a build tool's download does not match the hashes the lock records for it
- **THEN** uv refuses it and the setup fails with uv's `Hash mismatch` error

#### Scenario: A build tool the constraints do not name
- **WHEN** a source build installs a tool that is not in the build constraints, or at another version
- **THEN** the gated test fails naming that tool and version

#### Scenario: The pins change
- **WHEN** a push or pull request changes a file under `py/environment/`, or `knownSourceBuilds`
- **THEN** the Windows arm64 workflow builds the environment from nothing, compiling `blis` and `statsmodels`, runs both gated tests, and fails if either of them skipped or did not run
