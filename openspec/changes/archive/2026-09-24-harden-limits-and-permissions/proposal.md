# Proposal: harden-limits-and-permissions

## Why

Findings where the library does something unbounded, over-permissive, or louder than it should be. None needs a decision beyond picking a number, and each is a small, local change: #291 (SEC-12), #292 (SEC-13), #293 (SEC-15), #295 (SEC-17) and #339 (IN-19). On 0.4 the change also covered the first half of #294 (SEC-16) and #297 (SEC-20); see the backport section.

- **Bound parameters reached the terminal.** gorm's default logger prints a failing or slow query with its parameters interpolated, so a `WHERE token = ?` showed the token.
- **The online PNG fallback had no timeout and no size limit.** A `http.Client{}` with no `Timeout` waits forever on a server that accepts the connection and stops talking, and `io.ReadAll` on a remote body is an out-of-memory waiting for a large reply.
- **Six directories were created 0777.** The Python environment and the GLPK extraction directories were world-writable.
- **`ipc.WriteMessage` wrote a length its own reader refuses.** A payload over 256 MiB was written and then rejected at the other end; past 4 GiB the uint32 prefix truncates, so the reader takes the wrong number of bytes and every message after it is misframed.

## What Changes

- The CLI opens every database connection with `gorm.Config{Logger: gormlogger.Discard}`. The CLI reports its own errors, so nothing is lost.
- `SavePNG`'s online fallback (the default fallback on this line) uses a 60-second client timeout and reads through an `io.LimitReader` capped at 64 MiB, refusing a larger reply rather than writing it.
- The six `os.ModePerm` directory creations become `0o755`.
- `ipc.WriteMessage` refuses a payload over `maxMessageSize` before writing anything, so a failed call leaves the stream untouched.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `io-error-hygiene`: bounded remote reads, a framing layer that refuses what it cannot read back, and permissions that are not world-writable.
- `cli-secret-hygiene`: bound parameters do not reach the log.

## Impact

- `cli/commands/db_conn.go`, `plot/save_chart.go`, `py/py.go`, `py/init.go`, `py/internal/ipc/framing.go`, `lp/init.go`.
- User-visible, so both changelogs get entries: the CLI no longer prints gorm's query log, the online PNG fallback gives up after a minute, `ipc.WriteMessage` refuses an oversized payload, and the `py`/`lp` directories are 0o755.

## Backport to dev (0.3.x)

Dev received: gorm's `logger.Discard` on CLI database connections; `0o755` at the six `os.ModePerm` sites in `py` and `lp`; `ipc.WriteMessage` refusing a payload over 256 MiB before writing; and the 60-second timeout and 64 MiB reply cap on `SavePNG`'s online fallback, which on this line is still the default fallback path.

Left on 0.4:
- the 512 MB Excel unzip limit and exported `insyra.ExcelReadOptions`: breaking, it changes excelize's default and refuses files that read today;
- `PipInstall`/`PipUninstall` refusing a name starting with `-` and passing `--`: breaking, a caller passing `-r req.txt` stops working;
- the IPC accept loop stopping on a non-timeout error: changes when the server stops serving;
- the ten-minute per-connection deadline: a Python computation longer than ten minutes would fail;
- the `runtime.AddCleanup` socket-file removal: its object is a package-level `sync.Once` that is never collected, so the cleanup never runs;
- `api-review.md` and `delivery-status.md` edits.
