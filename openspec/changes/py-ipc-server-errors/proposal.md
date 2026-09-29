# Proposal: py-ipc-server-errors

## Why

PY-1 of the API review ([#254](https://github.com/HazelnutParadise/insyra/issues/254)), the part that needs no decision from the owner. Python sends its result back over a local IPC server: a Unix socket in `os.TempDir()`, or a named pipe on Windows. When that server cannot listen, the error never reaches the caller:

- The server is started once per process, in a goroutine, the first time the environment is prepared. A failed listen is logged (the review found `LogFatal`; `make-errors-non-terminating` turned it into `LogError`) and the address is kept anyway. The generated Python code connects to an address nobody listens on, `insyra.Return` fails, Python exits with status 1, and `RunCode` returns `exit status 1`. The reason is only in insyra's log.
- Because the start runs once, every later call in the process fails the same way, even after the cause is gone.
- One cause is easy to reach: a socket path longer than the platform allows. Measured on 2026-09-30 on macOS with a 140-character `TMPDIR`, the listen fails with `bind: invalid argument`.

The `## Unreleased` changelog also says the server "removes its socket file from the temp directory when the process ends". It does not. The cleanup is registered with `runtime.AddCleanup` on a package-level variable, which never becomes unreachable, and cleanups do not run at exit. Measured on 2026-09-30: after a test process that started the server exited, its socket file was still in `TMPDIR`.

## What Changes

- The IPC listener opens when a `Run…` call needs it and closes when the last call running at the same time finishes. Concurrent calls share one listener.
- A listener that cannot open makes the call return an error that names the address and wraps the listen error, so `errors.Is(err, syscall.EINVAL)` works for the path-length case. The call returns before the environment is prepared and before Python starts. The next call tries to open a listener again.
- Closing the listener removes the socket file on Unix, so the file exists only while a Python run is in flight. A process that ends in the middle of a run can still leave one behind.
- The changelog entry's socket claim is corrected to the new behaviour.

Whether `RunCode` should keep preparing the environment on first use, or require an explicit setup call, changes a default every user sees and is left on #254 for the owner.

## Capabilities

### New Capabilities

- `py-ipc-server`: when the IPC server that carries results back from Python is open, and what a call returns when it cannot open.

### Modified Capabilities

(none)

## Impact

- Code: `py/pyresult.go` (server lifecycle), `py/py.go` (the two runners take the server before preparing the environment), `py/init.go` (no longer starts the server).
- Tests: a new `py/ipc_server_test.go`. Its failure test uses a `TMPDIR` too long for a socket path, so it runs without Python and skips on Windows, where the pipe name does not depend on `TMPDIR`.
- Docs: `Docs/py.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md` (PY-1 stays open, with a note), `delivery-status.md`. `skills/insyra/SKILL.md` teaches nothing about the IPC server and does not change.
