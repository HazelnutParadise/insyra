# [ arrowipc ] Package

`arrowipc` reads Arrow IPC **files**, including Feather V2, through the repository's Arrow Go v18 dependency. It offers schema inspection and batch reading. Feather V1, the IPC streaming wire format and writing are outside this package.

## Inspect a schema

```go
info, err := arrowipc.Inspect("connections.feather")
if err != nil {
    return err
}
fmt.Println(info.NumBatches, info.Version)
for _, field := range info.Schema.Fields() {
    fmt.Println(field.Name, field.Type, field.Nullable)
}
```

```go
type FileInfo struct {
    Schema     *arrow.Schema
    NumBatches int
    Version    string
}

func Inspect(path string) (FileInfo, error)
func InspectContext(ctx context.Context, path string) (FileInfo, error)
func InspectFrom(ctx context.Context, r io.ReaderAt, size int64) (FileInfo, error)
```

Inspection fetches only the header, trailer and footer. It never reads record or dictionary values. The full Arrow schema preserves field order, nullability, dictionary index/value types and ordering, nested fields and metadata, even for types the DataTable reader refuses. The footer records the native batch count but no total row count, so `FileInfo` does not report rows. The returned schema remains usable after inspection and needs no `Release`.

`InspectFrom` lets a caller supply an HTTP Range-backed `io.ReaderAt` and the file's total byte size. It does not download or buffer the whole file and does not close the source. Requests depend on footer size, rather than total data size. Inspection limits the footer to 64 MiB and schema nesting to 64 field levels. Invalid vector lengths and references are rejected before schema decoding. Decoded field names, timezones and metadata keys/values have a separate 64 MiB aggregate budget per footer/schema; shared wire strings count on every occurrence. Each native message has the same decoded custom-metadata string cap. A footer/schema also allows at most 65,536 total field and metadata-entry occurrences; repeated references count separately. Each native message custom-metadata map has the same independent entry cap.

## Read batches

```go
type ReadOptions struct {
    Columns       []string // Empty means all top-level fields.
    MaxBatchBytes int64    // Decoded buffers + dictionaries; 0 = 256 MiB.
    MaxOutputBytes int64   // Cell slots + expanded payloads per table; 0 = 256 MiB.
}

func Stream(ctx context.Context, path string, opt ReadOptions, batchSize int) (<-chan *insyra.DataTable, <-chan error)
func StreamFrom(ctx context.Context, r io.ReaderAt, size int64, opt ReadOptions, batchSize int) (<-chan *insyra.DataTable, <-chan error)
```

The following function processes a file and returns any terminal read error. It needs `context` and `github.com/HazelnutParadise/insyra/arrowipc` imports.

```go
func readConnections(ctx context.Context, path string) error {
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()

    batches, errs := arrowipc.Stream(ctx, path, arrowipc.ReadOptions{
        Columns: []string{"body_pre", "body_post", "weight"},
    }, 4096)
    for table := range batches {
        // Process this independently owned table before accepting the next.
        table.Show()
    }
    return <-errs
}
```

`batchSize` must be positive. Each table contains at most that many rows. Native record boundaries remain visible: two short native records are not combined. Columns use exact, case-sensitive top-level names and appear in request order. Unknown names, repeated selections or ambiguous field names return errors. Options are copied when the stream starts.

### Memory and ownership

- The producer holds one native Arrow record batch, its dictionary state and one output table. It uses an unbuffered table channel, so a slow consumer does not build a queue.
- Arrow v18 reads and decodes every column of that native record. Column selection reduces DataTable conversion, not network bytes or native record memory. A small `batchSize` does **not** cap decoded bytes. `MaxBatchBytes` limits the combined decoded native record buffers and retained dictionary buffers, defaulting to 256 MiB. A negative value is an error. The limit is checked before decompression. Arrow's encoded message limit of 256 MiB and metadata limit of 64 MiB also remain in effect. DataTable cells, encoded buffers and runtime overhead use additional memory, so this is not a process RSS limit.
- Strings and binary cells are copied. Received DataTables stay valid across later batches, cancellation and reader closure. Consumers never call Arrow `Retain` or `Release`.
- `MaxOutputBytes` separately limits each output table before its cell slices or string/binary copies are allocated. It charges 16 bytes per selected cell plus each string/binary payload, counting a dictionary value on every occurrence. Zero means 256 MiB; negative values fail before data reads. Exceeding the limit returns an error without emitting that table. Runtime objects and temporary copies use additional memory. There is no total-stream row limit; callers use cancellation to stop reading.
- Path entry points close their files. `StreamFrom` leaves the source open, and the caller must keep it usable until both channels close.

### Cancellation and errors

Drain the table channel before reading the terminal error channel. A successful stream closes the error channel without a value, so `<-errs` is nil. A failed stream sends one error to its buffered error channel and closes both channels. An error after earlier batches does not undo those batches.

If processing stops early, cancel the supplied context. To wait for cleanup, drain the table channel and then receive from the error channel. Cancellation is checked around reads, conversion and delivery. A nil context is an error. An `io.ReaderAt` call already in progress cannot be interrupted by this interface, so a remote reader must bind its HTTP requests to the same context.

Malformed files, truncated input and unsupported selected types return errors. Overlapping or duplicate footer blocks are rejected before reading dictionary or record bodies. A record with rows but zero columns also returns an error because a DataTable cannot represent those rows. A file with an unsupported field can still be inspected, and `Columns` can exclude that field.

## Cell types

The mappings match `parquet` for supported Arrow types. There is no conversion of IDs through `float64`.

| Arrow type | DataTable cell |
| --- | --- |
| Null or null slot in any supported column | `nil`, distinct from numeric zero |
| Int8 / Int16 / Int32 / Int64 | Corresponding signed Go integer width |
| Uint8 / Uint16 / Uint32 / Uint64 | Corresponding unsigned Go integer width |
| Float32 / Float64 | `float32` / `float64` |
| Bool | `bool` |
| String / LargeString | Copied `string` |
| Binary / LargeBinary / FixedSizeBinary | Copied `[]byte`, including empty byte strings |
| Timestamp | `time.Time` using the Arrow time unit |
| Date32 / Date64 | `time.Time` in UTC, as in Parquet |
| Duration | `int64` count in the schema's duration unit |
| Decimal128 / Decimal256 | Exact `github.com/TimLai666/go-decimal` `decimal.Decimal`; precision 1..38 / 1..76, scale 0..precision |
| Dictionary with supported values | Decoded value in the corresponding type, or `nil` for a null index/value |
| List / LargeList / FixedSizeList, nested lists, Struct, Map, Union, extension types and other unlisted types | Explicit error identifying the selected field and Arrow type |

Other decimal parameters, including negative scales, remain inspectable but are explicit selected-column errors. This conversion range follows [Parquet DECIMAL](https://github.com/apache/parquet-format/blob/master/LogicalTypes.md#decimal) and keeps returned cells safe to format.

Dictionary ordering is retained in `FileInfo.Schema`; a DataTable contains values, without category ordering semantics. An ordered int8-to-UTF8 dictionary therefore produces strings, never its int8 codes. A selected unsupported type fails before the first table is emitted, including an unsupported type containing only nulls. Parquet currently records unsupported types on `DataTable.Err()` and substitutes nil cells. This package instead returns a terminal error to avoid confusing unsupported values with real nulls.

## Format references

- [Arrow Go v18.8.0 IPC API](https://pkg.go.dev/github.com/apache/arrow-go/v18@v18.8.0/arrow/ipc)
- [IPC file layout](https://arrow.apache.org/docs/format/Columnar.html#ipc-file-format)
- [Feather V2 is the Arrow IPC file format](https://arrow.apache.org/docs/python/feather.html)
- [Parquet cell mappings](parquet.md#read)
