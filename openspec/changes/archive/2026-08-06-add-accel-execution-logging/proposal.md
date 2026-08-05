# Proposal: add-accel-execution-logging

## Why

Acceleration is completely silent on the console: `accel/` contains no `LogInfo` at all, so a user cannot tell whether the GPU ran, which device ran, or why a call fell back — that information exists only in the report API nobody reads casually. The library logs through the root package's structured logger everywhere else; acceleration should too.

## What Changes

- On a session's first accelerated execution, one `LogInfo` line names the device, backend, mode, and shard strategy — once per session, not per call.
- On a session's first CPU fallback after acceleration was possible in principle, one `LogInfo` line names the `FallbackReason` — a missing device is a performance event, and the user hears about it exactly once.
- Per-execution detail (operation, rows, chunk count, per-assignment placement, fallback reasons) goes to `LogDebug`, so `Config.SetLogLevel` turns the firehose on when wanted and off by default.
- A spam guard is part of the contract: N executions on one session produce one info line, verified by test.
- Changelog entries in both languages.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `accel-observability`: acceleration SHALL announce device use and first fallback once per session through the root logger, with per-execution detail at debug level.

## Impact

- **Code**: `accel/` session/executor logging call sites; uses the root package logger already imported by the module.
- **Behavior**: log output only; no result or API changes.
- **Docs**: `CHANGELOG.md` / `CHANGELOG_TW.md`; a line in `Docs/accel.md`.
- **Dependencies**: none.
