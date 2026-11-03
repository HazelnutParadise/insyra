# arrow-ipc-reading Specification

## Purpose
Provide metadata-only schema inspection and bounded, cancellable DataTable iteration for Arrow IPC files and Feather V2 while preserving scalar types and explicit errors.
## Requirements
### Requirement: Metadata-only schema inspection
The arrowipc package SHALL inspect IPC file and Feather V2 schemas from paths or io.ReaderAt plus size without reading record or dictionary bodies. It SHALL return the full Arrow schema, IPC version and native batch count, including unsupported field types.

#### Scenario: Dictionary and nested schema over ranges
- **WHEN** a caller inspects a file with dictionary and list columns through a ReaderAt that rejects body reads
- **THEN** inspection succeeds with original types, nullability, order and metadata and fetches only header, trailer and footer bytes

### Requirement: Bounded typed batch reading
Stream and StreamFrom SHALL emit independently owned DataTables of at most the positive batchSize rows, preserving native batch boundaries and requested top-level column order. Empty Columns SHALL select all fields. Columns SHALL accept Excel-style strings, insyra.Name names and int positions (negative from the end). Invalid, duplicate and ambiguous selections SHALL error. The producer SHALL hold at most one native record plus dictionary state and one output table.

#### Scenario: Multiple batches and selection
- **WHEN** a generated file contains several records and Columns selects two fields in reverse order
- **THEN** every row is emitted once with those fields in that order and no output exceeds batchSize

### Requirement: Decoded byte budget
ReadOptions.MaxBatchBytes SHALL cap decoded native record buffers plus retained dictionary buffers before decompression. Zero SHALL use 256 MiB and negative values SHALL error before data reads. The budget SHALL cover native buffers for all fields, including unselected ones, without claiming to limit process RSS or DataTable overhead.

#### Scenario: Decoded length exceeds budget
- **WHEN** a compressed buffer declares a decoded length above the budget or dictionary buffers leave insufficient room for a record
- **THEN** reading fails explicitly before that allocation and no affected table is emitted

### Requirement: Disjoint IPC block extents
Stream readers SHALL reject overlapping or duplicate footer block extents, across both dictionary and record blocks, before reading native block bodies. Extent validation SHALL include metadata and body bytes, preserve footer record order and allow adjacent non-overlapping blocks.

#### Scenario: Overlapping block dispatch
- **WHEN** a footer block overlaps another block's metadata or body, or duplicates its extent
- **THEN** streaming fails explicitly before any body read or table emission

### Requirement: Bounded decoded metadata strings
Readers SHALL charge schema field names, timestamp timezones and metadata keys/values per decoded occurrence, including aliased wire strings. Footer/schema strings and each message's custom metadata SHALL have a 64 MiB decoded-string budget checked before Arrow decoding. Exceeding the budget SHALL return an explicit error without expanding the strings.

#### Scenario: Shared string offsets amplify decoded metadata
- **WHEN** many names or metadata entries reference one wire string and their decoded total exceeds the budget
- **THEN** inspection or streaming rejects the metadata before Arrow copies the strings

### Requirement: Bounded decoded schema entries
Readers SHALL reject a footer/schema with more than 65,536 total decoded field and metadata-entry occurrences before Arrow allocation. Repeated references SHALL count separately. Each native message custom-metadata map SHALL also be limited to 65,536 entries, independently of wire size and string bytes.

#### Scenario: Shared empty field offsets
- **WHEN** a bounded footer repeats one empty-name scalar field or metadata table more than the decoded entry limit
- **THEN** inspection and streaming reject it before Arrow materializes the expanded objects

### Requirement: Supported decimal parameters
Selected Decimal128 and Decimal256 columns SHALL use precision 1..38 and 1..76 respectively and scale 0..precision, matching Parquet's decimal parameter range. Other parameters SHALL return an explicit unsupported-column error before cell construction, including dictionary value types. Inspection SHALL preserve the original decimal schema.

#### Scenario: Extreme or negative decimal scale
- **WHEN** a selected decimal has an excessive or negative scale, or precision beyond its storage width
- **THEN** streaming returns an error naming the column and type without emitting a cell or formatting its value

### Requirement: Cancellable wide table construction
Readers SHALL construct output tables without scanning all earlier column names for every selected field. They SHALL use already validated unique names, preserve order and cell ownership, and check Context while filling columns. Wide schemas within the inspection limits SHALL remain readable without an additional selected-column count cap.

#### Scenario: Wide null record with cancellation
- **WHEN** a generated record has 65,536 uniquely named Null columns and reading has a short deadline
- **THEN** reading completes with all names and nulls intact or terminates promptly with cancellation, without entering quadratic name collision checks

### Requirement: Bounded native footer block entries
Stream readers SHALL cap the aggregate dictionary and record block count at 65,536 before allocating, appending or sorting decoded block entries. Shared vectors SHALL count at every occurrence. Exceeding this limit SHALL fail before native body reads; inspection MAY report the larger batch count without materializing block entries.

#### Scenario: Dictionary and record vectors share block entries
- **WHEN** dictionary and record vectors each pass wire bounds but their combined entry count exceeds 65,536
- **THEN** streaming rejects the count before constructing the decoded block collection or reading a body

### Requirement: Bounded variadic view-buffer counts
Stream readers SHALL validate variadic view-buffer counts before native allocation and body reads, including unselected fields. Counts SHALL be nonnegative, their vector and aggregate SHALL not exceed 65,536, and their aggregate SHALL not exceed the message's declared buffer entries. Invalid counts SHALL produce an explicit error.

#### Scenario: Unselected view requests excessive buffers
- **WHEN** an unselected StringView field declares negative or excessive variadic buffers
- **THEN** reading selected scalar columns fails before body reads or Arrow allocation, while a valid unselected view remains readable

### Requirement: Indexed named column selection
Readers SHALL resolve requested names through the schema's prebuilt name index without rescanning every field per selector. Resolution SHALL preserve requested order, unknown/ambiguous-name errors and each maintained line's duplicate and selector semantics.

#### Scenario: Wide schema selected by name
- **WHEN** a caller selects all 65,536 uniquely named fields in reverse order with a short deadline
- **THEN** selection and construction complete in the requested order or terminate promptly with cancellation, without quadratic schema scans

### Requirement: Explicit dictionary delta refusal
Stream readers SHALL reject dictionary delta messages before reading that message's body or invoking native dictionary decoding, including when its column is excluded. Ordinary dictionary values, ordering metadata and nulls SHALL remain supported. Inspect SHALL preserve dictionary schemas without fetching delta metadata or bodies.

#### Scenario: Delta with unsafe logical lengths or offsets
- **WHEN** a file contains a dictionary delta, with ordinary data or forged logical lengths, offsets or a zero-length body
- **THEN** streaming emits no table and reports an explicit unsupported-delta error before fetching the delta body, while inspection succeeds

### Requirement: Expanded output budget
ReadOptions.MaxOutputBytes SHALL bound each table's cell slots (charged as 16 bytes each) and copied string/binary payloads, including expanded dictionary values, before allocation. Zero SHALL use 256 MiB and negative values SHALL error before data reads. Exceeding the limit SHALL return an error without emitting the affected table. This SHALL NOT claim to bound process RSS or total stream rows.

#### Scenario: Dictionary expansion and null cell slots
- **WHEN** repeated dictionary values or null-only cell slots would exceed the per-table output budget
- **THEN** reading fails before the excess allocation and emits no affected table

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
Readers SHALL reject nil or cancelled contexts, report malformed/truncated input as errors, and never close caller-owned ReaderAt sources. Stream and StreamFrom SHALL return lazy iter.Seq2[*insyra.DataTable, error] sequences, yield terminal errors once as (nil, error), and cancel and drain internal production before returning on early loop exit. Tables and schemas SHALL survive reader cleanup without caller Release calls. Caller-owned sources SHALL be safe to close after iteration returns.

#### Scenario: Cancellation and damaged files
- **WHEN** a context is cancelled before reading or while a producer is blocked, or a source is truncated or not Arrow
- **THEN** the operation terminates with the corresponding error and no panic or producer remains blocked

#### Scenario: Retained output
- **WHEN** a caller retains earlier tables and schema after reading completes
- **THEN** their values remain unchanged and independent of Arrow buffers and the caller-owned source remains open
