# Proposal: py-pin-source-build-tools

## Why

`py-pinned-environment` pinned uv, Python and every package with its file hashes, except one part. On Windows on arm64, PyPI has no wheel of `blis` 1.3.3 or `statsmodels` 0.15.0, so uv builds those two from source there, and the tools the builds use were chosen at setup time from the ranges the two source archives declare (`cython>=3.0,<4.0`, `meson-python`, `setuptools`, …), with no hash check. They are the only thing the environment installs that is neither pinned nor verified, and `Docs/py.md` says so. The owner asked on 2026-10-03 to pin them too and to run the build once on a real Windows arm64 machine.

uv has a setting for this: `[tool.uv] build-constraint-dependencies` restricts the versions a source build may use, and since uv 0.12.16 each entry can carry hashes, which `uv lock` records in `uv.lock`. The pinned uv is 0.12.20.

## What Changes

- `py/environment/pyproject.toml` gains `build-constraint-dependencies` with the ten packages the two builds install, measured on 2026-10-03 by building both from source with the pinned uv: `cython`, `numpy`, `setuptools` for `blis`; `meson-python`, `meson`, `ninja`, `cython`, `numpy`, `scipy`, `setuptools-scm`, `setuptools`, `packaging`, `pyproject-metadata` for `statsmodels`. `ninja` is in the list because meson-python asks for it when no `ninja` is on `PATH`. Each entry is an exact version with the SHA-256 of every file PyPI has for that version. `numpy`, `scipy` and `packaging`, which the environment also installs, are pinned at the versions it runs. Every one of the ten has a wheel for Windows on arm64 or a pure-Python wheel, so no build tool is itself built from source.
- `uv.lock` is regenerated and records them under `[manifest] build-constraints`. The resolution of the environment's own packages does not change. `uv sync --frozen` takes the build constraints from the lock, so a build tool whose download does not match is refused and the setup fails.
- The tests fail when a build constraint is not an exact `name==version` with at least one SHA-256, when the lock does not record the same constraints with the same hashes, or when a build tool the environment also installs is pinned at another version.
- uv installs a build tool that no constraint names without an error (measured by leaving `ninja` out), so the tests above cannot tell whether the list is complete. `TestSourceBuildsUseOnlyPinnedTools`, gated on `INSYRA_PY_E2E`, builds every package in `knownSourceBuilds` from source with the pinned uv, an empty cache and uv's log on, and fails if a build installed a tool the lock's build constraints do not pin at that version.
- A new workflow runs that test and `TestPinnedEnvironmentEndToEnd`, which builds the environment from nothing through the `py` package, on GitHub's `windows-11-arm` runner, free for public repositories. It runs when a file that decides the build or the checks changes (`py/environment/`, `py/environment.go`, `py/init.go`, `py/const.go`, the three environment test files, the workflow), so every change of the pins is built on the one platform that compiles from source, and it fails unless both tests are seen to pass, since each skips itself without `INSYRA_PY_E2E`.
- `Docs/py.md`: the Windows arm64 note says the build tools are pinned and hash-checked, that `blis` needs an arm64 LLVM's `clang` there, which its `setup.py` looks for, besides the Visual Studio C compiler, and how to make uv build the two again: uv reuses a wheel it built from the same source archive, so a machine that built them before with other tools keeps that build until it is cleared from uv's cache. "Pinned versions" describes the constraints and how to bump them.
- An environment built before this change is synced once more on its next run, because the pins it was built from changed. Where every package has a wheel, which is every platform except Windows arm64, that sync installs nothing.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `py-environment`: the pins include the tools a source build uses, and the environment is built on Windows arm64 whenever the pins change.

## Impact

- Pins: `py/environment/pyproject.toml`, `py/environment/uv.lock`.
- Tests: `py/environment_pins_test.go`, `py/environment_build_test.go` (new), `py/environment_buildlog_test.go` (new).
- CI: `.github/workflows/py-environment-windows-arm64.yml` (new).
- Docs: `Docs/py.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `delivery-status.md`.
