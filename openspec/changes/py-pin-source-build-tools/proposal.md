# Proposal: py-pin-source-build-tools

## Why

`py-pinned-environment` pinned uv, Python and every package with its file hashes, but not the tools uv installs when it builds a package from source; those come from the ranges the source archive declares (`cython>=3.0,<4.0`, `meson-python`, `setuptools`, …), with no hash check. `Docs/py.md` said that on Windows on arm64, where PyPI has no wheel of `blis` 1.3.3 or `statsmodels` 0.15.0, uv builds those two from source. The owner asked on 2026-10-03 to pin those tools too and to run the build once on a real Windows arm64 machine.

That run, on GitHub's `windows-11-arm` runner on 2026-10-03, showed the premise was wrong: uv installs the x86-64 CPython on Windows on arm64 by default (uv 0.7.18 added the arm64 builds "not downloaded by default, since x86-64 Python has broader ecosystem support"), so every package installs from its `win_amd64` wheel and nothing is built. No supported platform builds from source today. A native arm64 Python on Windows would build the two, and so would any platform a later pin leaves without a wheel, so the tools are still worth pinning, as a guard for when that happens. Python failing to start on that runner is fixed by `py-windows-arm64-runs`.

uv has a setting for this: `[tool.uv] build-constraint-dependencies` restricts the versions a source build may use, and since uv 0.12.16 each entry can carry hashes, which `uv lock` records in `uv.lock`. The pinned uv is 0.12.20.

## What Changes

- `py/environment/pyproject.toml` gains `build-constraint-dependencies` with the ten packages the two builds install, measured on 2026-10-03 by building both from source with the pinned uv: `cython`, `numpy`, `setuptools` for `blis`; `meson-python`, `meson`, `ninja`, `cython`, `numpy`, `scipy`, `setuptools-scm`, `setuptools`, `packaging`, `pyproject-metadata` for `statsmodels`. `ninja` is in the list because meson-python asks for it when no `ninja` is on `PATH`. Each entry is an exact version with the SHA-256 of every file PyPI has for that version. `numpy`, `scipy` and `packaging`, which the environment also installs, are pinned at the versions it runs. Every one of the ten has a wheel for Windows on arm64 or a pure-Python wheel, so no build tool is itself built from source.
- `uv.lock` is regenerated and records them under `[manifest] build-constraints`. The resolution of the environment's own packages does not change. `uv sync --frozen` takes the build constraints from the lock, so a build tool whose download does not match is refused and the setup fails.
- The tests fail when a build constraint is not an exact `name==version` with at least one SHA-256, when the lock does not record the same constraints with the same hashes, or when a build tool the environment also installs is pinned at another version.
- uv installs a build tool that no constraint names without an error (measured by leaving `ninja` out), so the tests above cannot tell whether the list is complete. `TestSourceBuildsUseOnlyPinnedTools`, gated on `INSYRA_PY_E2E`, builds every package in `knownSourceBuilds` from source with the pinned uv, an empty cache and uv's log on, and fails if a build installed a tool the lock's build constraints do not pin at that version.
- A new workflow, `Python Build Tools`, runs that test on GitHub's `macos-latest` runner when a file that decides it changes (`py/environment/`, `py/environment.go`, `py/environment_build_test.go`, `py/environment_pins_test.go`, the workflow), and fails unless the test is seen to pass, since it skips itself without `INSYRA_PY_E2E`.
- `Docs/py.md`: the Windows arm64 note says uv runs an x86-64 Python there and builds nothing, and that a native arm64 Python would build `blis` and `statsmodels` with the pinned tools. "Pinned versions" describes the constraints and how to bump them.
- An environment built before this change is synced once more on its next run, because the pins it was built from changed. Every package has a wheel for the Python each supported platform runs, so that sync installs nothing.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `py-environment`: the pins include the tools a source build uses, and a real build checks that they cover every tool whenever the pins change.

## Impact

- Pins: `py/environment/pyproject.toml`, `py/environment/uv.lock`.
- Tests: `py/environment_pins_test.go`, `py/environment_build_test.go` (new), `py/environment_buildlog_test.go` (new).
- CI: `.github/workflows/py-build-tools.yml` (new).
- Docs: `Docs/py.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `delivery-status.md`, `AGENTS.md` (a follow-up on musl Linux).
