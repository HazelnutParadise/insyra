# arrow-ipc-reading Specification

## Purpose
Provide metadata-only schema inspection and bounded, cancellable DataTable reading for Arrow IPC files and Feather V2 while preserving scalar types and explicit errors.
## Requirements
### Requirement: Metadata-only schema inspection
The arrowipc package SHALL inspect IPC file and Feather V2 schemas from paths or io.ReaderAt plus size without reading record or dictionary bodies. It SHALL return the full Arrow schema, IPC version and native batch count, including unsupported field types.

#### Scenario: Dictionary and nested schema over ranges
- **WHEN** a caller inspects a file with dictionary and list columns through a ReaderAt that rejects body reads
- **THEN** inspection succeeds with original types, nullability, order and metadata and fetches only header, trailer and footer bytes

### Requirement: Bounded typed batch reading
Stream and StreamFrom SHALL emit independently owned DataTables of at most the positive batchSize rows, preserving native batch boundaries and requested top-level column order. Empty Columns SHALL select all fields. Missing, duplicate and ambiguous selected names SHALL error. The producer SHALL hold at most one native record plus dictionary state and one output table.

#### Scenario: Multiple batches and selection
- **WHEN** a generated file contains several records and Columns selects two fields in reverse order
- **THEN** every row is emitted once with those fields in that order and no output exceeds batchSize

### Requirement: Decoded byte budget
ReadOptions.MaxBatchBytes SHALL cap decoded native record buffers plus retained dictionary buffers before decompression. Zero SHALL use 256 MiB and negative values SHALL error before data reads. The budget SHALL cover native buffers for all fields, including unselected ones, without claiming to limit process RSS or DataTable overhead.

#### Scenario: Decoded length exceeds budget
- **WHEN** a compressed buffer declares a decoded length above the budget or dictionary buffers leave insufficient room for a record
- **THEN** reading fails explicitly before that allocation and no affected table is emitted

### Requirement: Faithful scalar cells and explicit unsupported types
Readers SHALL preserve Parquet's supported scalar cell mappings and nil nulls, including int64 IDs beyond 2^53, binary byte slices, dates, decimals and dictionary values. Selected unsupported types, including lists, SHALL produce an error naming the field and type before any table is emitted. Inspection SHALL still describe those fields.

#### Scenario: Ordered dictionary and exact nullable IDs
- **WHEN** reading nullable int64, UTF8, float64, int32 and ordered int8-to-UTF8 dictionary columns
- **THEN** cell types and values match the Parquet rules, IDs stay exact and null differs from zero

#### Scenario: Nested list refusal
- **WHEN** a selected column is list of int64 or a nested list
- **THEN** reading returns an explicit unsupported-type error and selecting only supported fields succeeds

### Requirement: Zero-column row counts remain explicit
A record with positive rows and zero columns SHALL return an explicit error instead of emitting DataTables with a different row count. Schema-only files with no records SHALL terminate successfully without tables.

#### Scenario: Empty schema with rows
- **WHEN** a valid IPC record contains rows but no columns
- **THEN** the reader reports that DataTable cannot represent a nonempty zero-column record and emits no table for it

### Requirement: Cancellation errors and ownership
Readers SHALL reject nil or cancelled contexts, report malformed/truncated input as errors, and never close caller-owned ReaderAt sources. Streams SHALL close both channels and report terminal errors through a buffered error channel. Tables and inspected schemas SHALL remain valid after reader cleanup without caller Release calls. Callers SHALL drain tables or cancel when stopping early.

#### Scenario: Cancellation and damaged files
- **WHEN** a context is cancelled before reading or while a producer is blocked, or a source is truncated or not Arrow
- **THEN** the operation terminates with the corresponding error and no panic or producer remains blocked

#### Scenario: Retained output
- **WHEN** a caller retains earlier tables and schema after reading completes
- **THEN** their values remain unchanged and independent of Arrow buffers and the caller-owned source remains open
