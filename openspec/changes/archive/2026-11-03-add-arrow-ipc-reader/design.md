## Context

Arrow v18.8.0 is already pinned. Parquet has path and ReaderAt stream entry points, typed cells and nil nulls. Its unsupported nested types must not be silently converted in the new package.

## Goals / Non-Goals

Goals: inspect schema with bounded range reads, stream owned DataTables, preserve scalar cell types, and deliver the same addition on both maintained lines.

Non-goals: writing, Feather V1, IPC streaming format, whole-file DataTable reads, new nested cell representations or Parquet changes.

## Decisions

- Use `arrowipc.Inspect(path)`, `InspectContext(ctx,path)`, `InspectFrom(ctx,reader,size)` returning `FileInfo{Schema *arrow.Schema, NumBatches int, Version string}`. Schema retains dictionaries, ordering, nullability, nested fields and metadata. IPC footer has no row counts, so do not invent them.
- `Stream` and `StreamFrom` take `(ctx, source, ReadOptions{Columns: ..., MaxBatchBytes: ...}, batchSize)` like Parquet. Columns are exact top-level names in request order. Empty means all. Missing, repeated or ambiguous names error. Positive batchSize caps output rows; native record boundaries are retained.
- Inspect reads header, trailer and footer only. Arrow FileReader eagerly reads dictionary bodies, so copy the footer into a metadata-only IPC envelope, zero its dictionary-block vector length using the public FlatBuffers table API, then let Arrow decode its schema. Dictionary type metadata remains intact. Never alter source bytes. Reference: https://github.com/apache/arrow/blob/main/format/File.fbs and Arrow v18.8.0 arrow/ipc/file_reader.go.
- Streaming uses Arrow FileReader.RecordBatchAt, one native record at a time, always released before the next. Arrow still reads all columns of that record; column selection restricts conversion, not transport. Dictionary data lives until reader close. `MaxBatchBytes` caps the sum of decoded native buffers and retained dictionary buffers at 256 MiB by default, before decompression; a negative budget errors. This excludes encoded buffers and DataTable overhead. A ReaderAt guard validates the existing metadata/body reads without extra network requests. Each output copies strings, bytes and other cells, so consumers never call Release.
- Match parquet/internal.go scalar mappings, including int64, []byte, time.Time, exact go-decimal and duration counts as int64 in the schema unit. Decode dictionaries recursively to their values, with nil for null index/value. Reject selected list/struct/map/union/extension and other unsupported types with column name and type before emitting data. Inspection remains available for them.
- Independently cap expanded output with `MaxOutputBytes` (zero = 256 MiB). Before allocating a table, charge 16 bytes per selected cell and every copied string/binary payload, including repeated dictionary references. Reject the affected table on overflow; preserve prior tables. This is a per-table allocation charge, not a process RSS or total-stream row limit. Keep the native `MaxBatchBytes` semantics unchanged.
- Single producer, unbuffered table channel and buffered terminal error channel. Consume tables to closure then read the error channel, or cancel and drain. Check context before and after every ReaderAt call, conversion and send. Caller-owned sources stay open; path sources close. ReaderAt cannot interrupt an in-flight call; remote implementations must bind their transport to the same context.

## Risks / Trade-offs

- FlatBuffers footer layout: use only the documented Footer dictionaries field, bounds-check it and all schema vectors, enforce 64 field levels and forward offsets, recover malformed-input panics to errors, and pin metadata-only tests with dictionary and nested schemas.
- Native IPC records and dictionaries can be large: row splitting bounds DataTable output, not native decoded bytes. Document this limitation and retain Arrow's message limits. No whole-file buffering or producer queue.
- Truncation and invalid inputs: errors, no panic or fabricated success. A late stream failure preserves earlier delivered batches and appears on the terminal channel.

## Validation

Generated Arrow writer fixtures exercise public entry points, nullable >2^53 IDs, UTF8, f64, i32, ordered i8 dictionaries, lists (inspection and explicit refusal), multiple batches, selection, ownership, cancellation and damaged input. A recording ReaderAt refuses data-range access during inspection. Cross-format fixtures assert Parquet scalar cell equality. Manual Range probes of the three public MaleCNS files are excluded from CI; [verification.md](verification.md) names the inputs and observed results.
