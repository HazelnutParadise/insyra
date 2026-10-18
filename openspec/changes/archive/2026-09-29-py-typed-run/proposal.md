# Proposal: py-typed-run

## Why

Reading a Python result needed a variable declared first: `var r T; err := py.RunCode(&r, code)`. Installing a package could not be cancelled, and a nil context passed to a context form crashed instead of returning an error. `RunCodeWithTimeout` is the same call as `RunCodeContext` with a timeout context, and `0.4` removes it. Ported from `0.4`; `Setup(ctx)` needs `0.4`'s pinned environment and stays there.

## What Changes

- `py.Run[T](code) (T, error)` decodes the Python result into a `T` and returns the zero value with the error on failure.
- `PipInstallContext` and `PipUninstallContext` stop when the context ends and return `ctx.Err()`; `PipInstall` and `PipUninstall` call them with `context.Background()`.
- Every context form refuses a nil context with an error and an already-ended context with `ctx.Err()` before starting anything.
- `RunCodeWithTimeout` is Deprecated: use `RunCodeContext` with `context.WithTimeout`. Removed in 0.4.

## Capabilities

### New Capabilities

- `py-run-api`: the shape of the py package's run and pip entry points.

### Modified Capabilities

None.

## Impact

- `py/py.go`, `py/run_test.go`, `py/main_test.go`, `Docs/py.md`, both changelogs, `AGENTS.md` follow-up.
