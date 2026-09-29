## Context

`py` runs Python as a child process and receives the value passed to `insyra.Return` over a local IPC server: a Unix socket in `os.TempDir()`, or a named pipe on Windows. Today `pyEnvInit` starts the server once per process in a goroutine (`serverStartOnce`, then `serverOnce` inside `startServer`) and callers read the address after `serverReady` closes. A failed listen closes `serverReady` with the address still set, so nothing distinguishes "listening" from "failed".

## Goals / Non-Goals

**Goals:**

- A failed listen reaches the caller as the error of the call that needed the server, with the listen error wrapped.
- A later call can succeed once the cause is gone.
- The socket file does not outlive the runs that use it.

**Non-Goals:**

- Changing how the environment is prepared on first use (#254's open question).
- Changing the wire format, the ten-minute connection deadline or the accept loop's handling of a broken listener.

## Decisions

**The listener is reference-counted per run.** `acquireIPCServer` opens a listener when no run holds one and returns its address; `releaseIPCServer` closes it when the last run lets go. On Unix, `net.UnixListener.Close` unlinks the socket file, which gives the cleanup the changelog promised. Keeping one listener for the life of the process was the alternative: it saves one `listen` per call, which costs microseconds next to starting a Python interpreter, but it can only be cleaned up at exit, where Go runs nothing, and a failed start would need its own retry logic.

**The server is taken before the environment is prepared.** A listen failure then costs nothing, where preparing the environment first can mean minutes of downloads before the failure shows.

**The error wraps the listen error.** `fmt.Errorf("py: failed to start the IPC server on %s: %w", addr, err)` keeps `errors.Is` working for callers that want to tell a path-length failure (`syscall.EINVAL`) from anything else.

**The test forces a real failure, not a seam.** A `TMPDIR` of 200 characters makes `net.Listen("unix", …)` fail with `EINVAL` on Linux and macOS, so the test needs no hook in production code. It skips on Windows, where pipe names do not depend on `TMPDIR`. The runner test holds the setup lock while it calls `RunCodeContext` and `RunCode`, so a runner that went to the environment first waits on the lock and the test fails at its timeout. It also marks the environment as prepared with an interpreter path that does not exist, so such a runner, once released, fails at once instead of downloading anything.

## Risks / Trade-offs

- [A process killed during a run leaves the socket file] → Accepted. The file name is random, so a leftover never blocks a later run, and the old code left one on every run.
- [An accept loop that stops on an error that is not a timeout, such as running out of file descriptors] → The loop closes its listener as it stops. Left open, the listener still queued connections, and the Python process behind one waited for an acknowledgement nobody sent, so its run never returned. Closed, the connect fails, the run returns an error, and the next call after the runs in flight finish opens a new listener.
