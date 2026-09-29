# Proposal: py-typed-run

## Why

PY-2 of the API review ([#255](https://github.com/HazelnutParadise/insyra/issues/255)):

- `RunCode(out any, code string)` binds Python's result into whatever `out` points at, so the type is only checked at run time and the caller declares a variable, passes its address and checks the error separately. Go has had generics since 1.18; a call can say the type it wants and get the value back.
- `RunCodeWithTimeout(timeout, out, code)` and `RunCodeContext(ctx, out, code)` are two ways of doing the same thing: the first is the second with a context from `context.WithTimeout`. Under the owner's ruling on #211, a function has one name, and the old one stays one release as a Deprecated wrapper.
- `PipInstall` and `PipUninstall` take no context. Installing a package can take minutes, and so can the environment setup the first call triggers, with no way to stop either from a server's request context.
- Testing cancellation turned up an older defect. When Python fails, is killed because its context ended, or crashes before returning a value, the goroutine running it reports the error and then marks the process done, so the runner sees both at once and `select` picks either. Picking "done" returns no result and no error, so about half of all killed or crashed runs reported success. Measured: `RunCodeWithTimeout(300ms, nil, code)` on a Python that never finishes returned `nil`.
- A nil context makes every `…Context` form panic. Before `py-pinned-environment`, the panic came from `exec.CommandContext`, in a goroutine the runner had started, where it could not be recovered and ended the program; since then it comes from waiting on the setup lock. Measured on 2026-09-30 against the code before this change, with the test binary standing in for Python: `RunCodeContext(nil, nil, code)` panicked with a nil pointer dereference and ended the test binary.

## What Changes

- New `Run[T any](ctx context.Context, code string, args ...any) (T, error)` runs the code, with `$v1`, `$v2`, … filled from `args` as `RunCodef` fills them, and returns the value passed to `insyra.Return` decoded into a `T` by the same rules `RunCode` binds with: `*insyra.DataTable` and `*insyra.DataList` for DataFrames and Series, a struct, map, slice or scalar through JSON. On failure it returns `T`'s zero value and the error.
- New `PipInstallContext(ctx, dep)` and `PipUninstallContext(ctx, dep)`. The context bounds the first-use setup and the `uv pip` command; cancelling it stops the command and returns `ctx.Err()`. `PipInstall` and `PipUninstall` call them with `context.Background()`.
- `RunCodeWithTimeout` is Deprecated in favour of `RunCodeContext` with `context.WithTimeout`, keeps its meaning, and will be removed in the release after the one that deprecates it. An `AGENTS.md` follow-up records the removal.
- Every `…Context` form and `Run` refuse a nil context with an error instead of panicking, and return `ctx.Err()` without starting anything when the context is already done.
- A run whose Python process fails returns that failure, or `ctx.Err()` when the context ended, instead of a nil error about half the time. A result Python delivered before the process failed still comes back.
- `Run[insyra.DataTable]` and `Run[insyra.DataList]` are refused before anything starts, pointing to the pointer types, where the decoder would only fail after Python ran.
- The plain forms (`RunCode`, `RunCodef`, `RunFile`, `RunFilef`) call their `…Context` forms with `context.Background()`. The two copies of the runner become one; behaviour does not change.

`RunCode` and its family stay: they are what existing code calls, and the owner asked for `Run` to be added beside them rather than to replace them. `PipList`, `PipFreeze` and `ReinstallPyEnv` still take no context; they are not part of #255, and whether the setup gets its own `Setup(ctx)` is #254's open question.

## Capabilities

### New Capabilities

- `py-run-api`: the typed `Run`, the context forms and what they do with a nil or finished context, and the deprecated timeout helper.

### Modified Capabilities

(none)

## Impact

- Code: `py/py.go`, and `waitForResult` in `py/pyresult.go`.
- Tests: the test binary also stands in for Python. It reads the execution ID and IPC address from the script it is given and sends back a canned result, so the runner is tested through the real IPC server without a Python environment.
- Docs: `Docs/py.md`, `skills/insyra/SKILL.md` (the `…Context` convention now covers installing Python packages), both changelogs, `api-review.md` (PY-2), `delivery-status.md`, and an `AGENTS.md` follow-up for removing `RunCodeWithTimeout`.
