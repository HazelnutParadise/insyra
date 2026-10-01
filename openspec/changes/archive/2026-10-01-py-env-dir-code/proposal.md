# Proposal: py-env-dir-code

## Why

The `py` environment directory, `.insyra_env/py25c_<os>_<arch>`, carries a code the project changes when the Python version changes: `py25b` became `py25c` on 2025-10-19, when the Python spec went from `3.12.9` to `3.12.*`. `py-pinned-environment` changed it again, from `3.12.*` to an exact `3.12.14`, but kept `py25c` and relied on its sync marker to convert an old environment in place. The owner ruled on 2026-09-30 that a new Python version gets a new directory, named `py<two-digit year><letter>`, `a` for the year's first: this one is `py26a`. Nothing enforced the convention, which is how it was missed.

## What Changes

- **BREAKING (behaviour)**: the environment lives in `.insyra_env/py26a_<os>_<arch>`. The first run after upgrading builds it from scratch there; an environment in the old `.insyra_env/py25c_<os>_<arch>` is no longer used, so packages added to it with `PipInstall` must be installed again, and the directory can be deleted. insyra does not delete it.
- The code lives in `py/const.go` as `envDirCode`, next to `envDirPython`, the Python version it was given for. A test fails when the pinned Python in `py/environment/pyproject.toml` differs from `envDirPython`, or when the code is not `py<two digits><letter>`, so a Python bump cannot keep the old directory.
- A change of package versions alone keeps the directory, and the sync marker brings the environment to the new pins in place, as before.
- `Docs/py.md` says this in "Pinned versions", and its bump steps add the directory code. The Unreleased changelog entry of `py-pinned-environment`, which said an old environment is converted in place, is corrected.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `py-environment`: the environment directory is named for its Python version; an environment of another Python is not converted in place.

## Impact

- Code: `py/const.go`, the `ReinstallPyEnv` doc comment in `py/py.go`.
- Tests: `py/environment_pins_test.go`.
- Docs: `Docs/py.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `delivery-status.md`.
