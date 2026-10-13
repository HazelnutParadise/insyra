# Proposal: color-only-on-terminals

## Why

The `AGENTS.md` follow-up of 2026-10-13 ("red error lines carry colour codes into files and pipes") is fixed by this change, as the owner asked on 2026-10-13.

`colorText` checked only `NO_COLOR` and `TERM=dumb` (`internal/utils.IsColorSupported`). It never checked whether the output was a terminal. So with the default `SetUseColoredOutput(true)`, colour codes landed wherever the output went. Measured on `0.4` at 21733e18:

- **Log lines in a buffer:** with the standard log pointed at a buffer, `LogFatal` wrote `"\x1b[31m<{[insyra - FATAL!]}> pkg.Fn: Worse\n\x1b[0m\n"`. The reset sat after the newline, so it spilled onto the next line.
- **Show and Summary in a buffer:** `DataList.ShowTo`, `DataList.SummaryTo`, `DataTable.ShowTo`, `ShowTypesTo` and `SummaryTo` all wrote colour codes into a `bytes.Buffer`.
- **Through slog:** routed with `slog.SetDefault`, the codes ended up inside the JSON message.

## What Changes

- **When colour is used:** `colorText` takes the destination writer and colours only when four things hold:
  - coloured output is on;
  - `NO_COLOR` is unset;
  - `TERM` is not `dumb`;
  - the writer is an `*os.File` that is a terminal, checked with `golang.org/x/term`, already a dependency.
- **What gets plain text:** a file, a pipe or a buffer.
- **Writers checked:** log lines check `log.Writer()`. `Show…To` and `Summary…To` check the writer they were given. `Show` checks stdout.
- **Reset placement:** a coloured log line puts its reset before the newline.
- **Unchanged:** output to a terminal keeps its colours, and the CLI writes to stdout and stderr, so its colours stay too.

## Capabilities

### New Capabilities

- `colored-output`: when insyra's text output is coloured.

### Modified Capabilities

None.

## Impact

- `utils.go`, `logger.go`, `show.go`, `show_head_tail.go`, `datalist_summary.go`, `datatable_summary.go`, a test call in `batch3_core_test.go`, and a new test file.
- `Docs/Configuration.md`, `Docs/DataList.md`, `Docs/DataTable.md`, both changelogs (`### Core`), `AGENTS.md` (the follow-up is removed), `delivery-status.md`.
- **Who notices:** a program that saved `Show`, `Summary` or log output to a file or buffer gets plain text where it got colour codes. A test comparing such output to a coloured string would change. No return value changes.
