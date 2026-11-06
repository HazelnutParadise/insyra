## MODIFIED Requirements

### Requirement: Metadata-only schema inspection
The arrowipc package SHALL inspect IPC file and Feather V2 schemas from paths or io.ReaderAt plus size without reading record or dictionary bodies. It SHALL return, in Insyra's own types, every top-level field's name, Arrow type text, nullability and metadata, the schema metadata, the IPC version and the native batch count, including unsupported field types.

#### Scenario: Dictionary and nested schema over ranges
- **WHEN** a caller inspects a file with dictionary and list columns through a ReaderAt that rejects body reads
- **THEN** inspection succeeds with original types, nullability, order and metadata and fetches only header, trailer and footer bytes

### Requirement: Bounded typed batch reading
Stream and StreamFrom SHALL emit independently owned DataTables of at most the positive batchSize rows, preserving native batch boundaries and requested top-level column order. Empty Columns SHALL select all fields. Columns SHALL accept Excel-style strings, insyra.Name names and int positions (negative from the end). Invalid and ambiguous selectors and a field selected twice SHALL error. The producer SHALL hold at most one native record plus dictionary state and one output table.

#### Scenario: Multiple batches and selection
- **WHEN** a generated file contains several records and Columns selects two fields in reverse order
- **THEN** every row is emitted once with those fields in that order and no output exceeds batchSize

#### Scenario: Fields that share a name
- **WHEN** a file has fields named x, x and x_1
- **THEN** the table's columns are named x, x_1 and x_1_1, as DataTable.AppendCols names them

### Requirement: Decoded byte budget
ReadOptions.MaxBatchBytes SHALL cap decoded native record buffers plus retained dictionary buffers before decompression, and each encoded message body. Zero SHALL use 256 MiB and negative values SHALL error before data reads. A larger value SHALL let a larger batch be read. The budget SHALL cover native buffers for all fields, including unselected ones, without claiming to limit process RSS or DataTable overhead.

#### Scenario: Decoded length exceeds budget
- **WHEN** a compressed buffer declares a decoded length above the budget or dictionary buffers leave insufficient room for a record
- **THEN** reading fails explicitly before that allocation and no affected table is emitted

#### Scenario: Batch above the default
- **WHEN** a file holds one 288 MB batch and MaxBatchBytes is 512 MiB
- **THEN** every row is read, and with the default the read fails naming MaxBatchBytes

### Requirement: Faithful scalar cells and explicit unsupported types
Readers SHALL preserve Parquet's supported scalar cell mappings and nil nulls, including int64 IDs beyond 2^53, binary byte slices, dates, durations as time.Duration, decimals and dictionary values. A duration time.Duration cannot hold SHALL be an error. Selected unsupported types, including lists, SHALL produce an error naming the field and type before any table is emitted. Inspection SHALL still describe those fields.

#### Scenario: Ordered dictionary and exact nullable IDs
- **WHEN** reading nullable int64, UTF8, float64, int32 and ordered int8-to-UTF8 dictionary columns
- **THEN** cell types and values match the Parquet rules, IDs stay exact and null differs from zero

#### Scenario: Nested list refusal
- **WHEN** a selected column is list of int64 or a nested list
- **THEN** reading returns an explicit unsupported-type error and selecting only supported fields succeeds

#### Scenario: Duration units
- **WHEN** a duration column holds 1500 in seconds, milliseconds, microseconds or nanoseconds
- **THEN** each cell is the time.Duration of 1500 of that unit

### Requirement: Cancellation errors and ownership
Readers SHALL reject nil or cancelled contexts, report malformed/truncated input as errors, report a source shorter than its given size as an error wrapping io.ErrUnexpectedEOF, and never close caller-owned ReaderAt sources. Stream and StreamFrom SHALL return lazy iter.Seq2[*insyra.DataTable, error] sequences, yield terminal errors once as (nil, error), and cancel and drain internal production before returning on early loop exit. Tables SHALL survive reader cleanup without caller Release calls.

#### Scenario: Cancellation and damaged files
- **WHEN** a context is cancelled before reading or while a producer is blocked, or a source is truncated or not Arrow
- **THEN** the operation terminates with the corresponding error and no panic or producer remains blocked

#### Scenario: Retained output
- **WHEN** a caller retains earlier tables and file information after reading completes
- **THEN** their values remain unchanged and independent of Arrow buffers and the caller-owned source remains open

## REMOVED Requirements

### Requirement: Bounded native footer block entries
**Reason**: The cap made a file with more than 65,536 batches inspectable but unreadable. The 64 MiB footer limit already bounds the block list, and repeated or overlapping blocks are refused before any body is read.
**Migration**: None; such files now stream.
