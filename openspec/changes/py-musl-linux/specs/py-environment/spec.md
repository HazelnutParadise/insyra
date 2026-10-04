## MODIFIED Requirements

### Requirement: A source build uses only pinned, verified tools

When uv builds a locked package from source, every tool the build installs SHALL be one the build constraints pin, at that version, and its download SHALL match one of the constraint's hashes, or the setup SHALL fail. The build constraints SHALL be one list that the build requirements of every package the tests list as a source build accept together. Because uv installs a build tool that no constraint names without an error, a test gated on `INSYRA_PY_E2E=1` SHALL build every package the tests list as a source build for the platform it runs on, telling musl Linux from glibc Linux, with the pinned uv, an empty cache and `--no-binary-package`, and fail if a build installed a tool the constraints do not pin at that version; on a platform that builds nothing it SHALL skip. A workflow SHALL run that test in an Alpine container on amd64 and on arm64, with a C and C++ compiler installed and neither `ninja` nor `patchelf`, whenever a file that decides it changes (anything under `py/environment/`, `py/environment.go`, `py/init.go`, `py/const.go`, `py/environment_build_test.go`, `py/environment_pins_test.go`, `py/environment_setup_test.go` or the workflow), and SHALL fail unless the test is seen to pass.

#### Scenario: A tampered build tool
- **WHEN** a build tool's download does not match the hashes the lock records for it
- **THEN** uv refuses it and the setup fails with uv's `Hash mismatch` error

#### Scenario: A build tool the constraints do not name
- **WHEN** a source build installs a tool that is not in the build constraints, or at another version
- **THEN** the gated test fails naming that tool and version

#### Scenario: A source build the constraints do not fit
- **WHEN** a package the tests list as a source build requires a range of a build tool that excludes the pinned version
- **THEN** uv cannot resolve that build's requirements and the gated test fails

#### Scenario: The pins change
- **WHEN** a push or pull request changes a file under `py/environment/`, or `knownSourceBuilds`
- **THEN** the `Python on musl Linux` workflow builds the packages listed for musl Linux from source in an Alpine container on each architecture, runs the gated test, and fails if it skipped or did not run

## ADDED Requirements

### Requirement: musl Linux is a supported host

On a musl Linux host, such as Alpine, the setup SHALL let uv install its musl CPython build of the pinned version, and SHALL build from source, with the pinned build tools, every locked package that has no musllinux wheel for the host's architecture. The tests SHALL tell glibc from musl wheels and list those packages for `linux-musl/amd64` and `linux-musl/arm64`, so a pin that leaves a musl platform without a wheel fails until it is listed. `Docs/py.md` SHALL name the packages a musl host builds and the compiler it needs. A workflow SHALL build the environment from nothing and run `TestPinnedEnvironmentEndToEnd` in an Alpine container on amd64 and on arm64, with only a C and C++ compiler installed, whenever a file that decides the build changes, and SHALL fail unless the test is seen to pass on both.

#### Scenario: A package without a musllinux wheel
- **WHEN** the lock has no musllinux wheel of a package for a musl platform and the tests do not list it there
- **THEN** `go test ./py/` fails naming the package and the platform

#### Scenario: Setup on Alpine
- **WHEN** the environment is set up on Alpine on amd64 or arm64 with a C and C++ compiler installed
- **THEN** uv installs the musl CPython, builds the listed packages from source with the pinned tools, and Python runs every package the preamble imports
