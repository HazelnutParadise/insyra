# Proposal: py-explicit-setup

## Why

The last open point of PY-1 ([#254](https://github.com/HazelnutParadise/insyra/issues/254)): the first `RunCode`, `PipInstall` or other `py` call downloads uv, Python and the packages as a side effect of doing something else, and a program cannot choose when that happens. A server would rather prepare the environment at start-up and fail there than stall its first request for minutes; a CI job would rather see a network failure as a setup step. The owner ruled on 2026-09-30 to keep the setup on first use as the default and add an explicit, optional `Setup(ctx)` (option A on #254), because requiring it would break every program that calls `RunCode` today.

## What Changes

- New `py.Setup(ctx context.Context) error` prepares the environment now: it runs exactly what the first call would, the pinned uv download and verification and the sync to the pinned versions, and returns its error. Calling it is optional; after it succeeds, later calls start without preparing anything. When the environment is already prepared, it returns nil without calling uv.
- `ctx` bounds the download and the sync, and the wait for another call's setup. A nil context is an error, and a context that is already done returns `ctx.Err()` before anything starts.
- Nothing else changes: `RunCode` and every other function still prepare the environment on first use when `Setup` was not called.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `py-environment`: adds the explicit `Setup(ctx)`.

## Impact

- Code: `py/py.go`.
- Tests: `py/setup_test.go`, with the test binary standing in for uv.
- Docs: `Docs/py.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md` (PY-1 fixed), `delivery-status.md`. `skills/insyra/SKILL.md` teaches no `py` API and does not change.
