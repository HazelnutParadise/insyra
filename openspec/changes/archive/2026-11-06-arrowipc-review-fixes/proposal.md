## Why

The review of the merged Arrow IPC reader (#675, #676) found that a batch over 256 MiB could not be read whatever `MaxBatchBytes` said, because the reader compared against a constant and Arrow kept its own default body limit; that a file with more than 65,536 batches could be inspected but not streamed; that Duration cells lost their unit; that `FileInfo.Schema` exposed arrow-go's type, so an arrow-go major version would break Insyra callers; that some hand-written checks repeated what Arrow already does; that a wrong `size` came back as a bare `io.EOF`; that a file with two fields of one name could not be read at all; and, on `dev`, that the channel-based `Stream` leaked a goroutine and an open file for every stream stopped early. The owner ruled on 2026-11-06 that `dev` takes 0.4's column selectors, so both lines have one API. None of it is released.

## What Changes

- `MaxBatchBytes` also bounds each encoded message body, passed to Arrow with `ipc.WithBodySizeLimit`; `ipc.WithMetadataSizeLimit` carries the 64 MiB metadata limit. The 65,536-block cap is removed.
- **BREAKING (unreleased)**: `FileInfo` has `Columns []ColumnInfo` (name, Arrow type text, nullability, metadata) and `Metadata` instead of `Schema *arrow.Schema`.
- Duration cells are `time.Duration`; a count that does not fit is an error.
- A short source is an error wrapping `io.ErrUnexpectedEOF`.
- Fields that share a name are renamed `name_1`, `name_2`, ... instead of refused; a field selected twice is still an error.
- A test pins arrow-go v18.8.0, which the read watcher was checked against.
- On `dev`: `Stream`/`StreamFrom` return `iter.Seq2[*insyra.DataTable, error]`, `ReadOptions.Columns` is `[]any` with `insyra.Name`, and the root package gains `insyra.Name` and `NameSelector`, with `isr.Name` returning that type, as on 0.4.

## Impact

- `arrowipc/`, `Docs/arrowipc.md`, both changelogs, `delivery-status.md` (0.4). On `dev` also `column_selector.go` in the root package.
