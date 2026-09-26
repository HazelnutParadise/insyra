# Proposal: harden-limits-and-permissions

## Why

Six findings where the library does something unbounded, over-permissive, or louder than it should be. None needs a decision beyond picking a number, and each is a small, local change: #291 (SEC-12), #292 (SEC-13), #293 (SEC-15), the first half of #294 (SEC-16), #295 (SEC-17), #297 (SEC-20) and #339 (IN-19).

- **Bound parameters reached the terminal.** gorm's default logger prints a failing or slow query with its parameters interpolated, so a `WHERE token = ?` showed the token.
- **The opt-in online PNG fallback had no timeout and no size limit.** A `http.Client{}` with no `Timeout` waits forever on a server that accepts the connection and stops talking, and `io.ReadAll` on a remote body is an out-of-memory waiting for a large reply.
- **Six directories were created 0777.** The Python environment and the GLPK extraction directories were world-writable.
- **Excel reads had no unzip limit.** excelize defaults to 16 GB, so a few kilobytes on disk can decompress into more memory than the host has.
- **The IPC accept loop spun.** A listener that fails permanently made `Accept` fail every time, and `continue` turned that into a warning per iteration for the life of the process. The socket file was also left in `os.TempDir()`, and a peer that connected and said nothing held the goroutine open for good.
- **`PipInstall` took an option as a package name.** `uv pip install --requirement=/path` reads that file and installs whatever it lists, and the caller's string went through as one argv.
- **`ipc.WriteMessage` wrote a length its own reader refuses.** A payload over 256 MiB was written and then rejected at the other end; past 4 GiB the uint32 prefix truncates, so the reader takes the wrong number of bytes and every message after it is misframed.

## What Changes

- The CLI opens every database connection with `gorm.Config{Logger: gormlogger.Discard}`. The CLI reports its own errors, so nothing is lost.
- `SavePNG`'s online fallback uses a 60-second client timeout and reads through an `io.LimitReader` capped at 64 MiB, refusing a larger reply rather than writing it.
- The six `os.ModePerm` directory creations become `0o755`.
- Excel reads pass `UnzipSizeLimit: 512 MB` through a new exported `insyra.ExcelReadOptions`, so `csvxl` applies the same limit.
- The IPC accept loop returns on a closed listener, continues only on a timeout, and reports anything else once before stopping. The socket file is removed when the process ends, and each connection carries a ten-minute deadline.
- `PipInstall` and `PipUninstall` refuse a name starting with `-` and pass `--` before it, so `uv` cannot read it as an option either way.
- `ipc.WriteMessage` refuses a payload over `maxMessageSize` before writing anything, so a failed call leaves the stream untouched.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `io-error-hygiene`: bounded reads and unzip limits, and permissions that are not world-writable.
- `cli-secret-hygiene`: bound parameters do not reach the log.

## Impact

- `cli/commands/db_conn.go`, `plot/save_chart.go`, `py/py.go`, `py/init.go`, `py/pyresult.go`, `py/internal/ipc/framing.go`, `lp/init.go`, `read.go`, `csvxl/convert.go`, `csvxl/convertDir.go`.
- User-visible in three places, so both changelogs get entries: the CLI no longer prints gorm's query log, the online PNG fallback gives up after a minute, and a dependency name starting with `-` is refused.
- The second half of SEC-16 — a streaming entry point for CSV — is the reader/writer API decision (K-11, #210) and is left alone.
