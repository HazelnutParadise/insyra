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

`InspectFrom` lets a caller supply an HTTP Range-backed `io.ReaderAt` and the file's total byte size. It does not download or buffer the whole file and does not close the source. Requests depend on footer size, rather than total data size. Inspection limits the footer to 64 MiB and schema nesting to 64 field levels. Invalid vector lengths and references are rejected before schema decoding.

## Read batches

```go
type ReadOptions struct {
    Columns       []string // Empty means all top-level fields.
    MaxBatchBytes int64    // Decoded buffers + dictionaries; 0 = 256 MiB.
}

func Stream(ctx context.Context, path string, opt ReadOptions, batchSize int) iter.Seq2[*insyra.DataTable, error]
func StreamFrom(ctx context.Context, r io.ReaderAt, size int64, opt ReadOptions, batchSize int) iter.Seq2[*insyra.DataTable, error]
```

The following function processes a file and returns any terminal read error. It needs `context` and `github.com/HazelnutParadise/insyra/arrowipc` imports. The sequence follows this line's Parquet API; the `dev` / 0.3.6 companion has the two-channel form instead.

```go
func readConnections(ctx context.Context, path string) error {
    for table, err := range arrowipc.Stream(ctx, path, arrowipc.ReadOptions{
        Columns: []string{"body_pre", "body_post", "weight"},
    }, 4096) {
        if err != nil {
            return err
        }
        // Owns its cells; breaking here also finishes the internal reader.
        table.Show()
    }
    return nil
}
```

`batchSize` must be positive. Each table contains at most that many rows. Native record boundaries remain visible: two short native records are not combined. Columns use exact, case-sensitive top-level names and appear in request order. Unknown names, repeated selections or ambiguous field names return errors. Options are copied when Stream or StreamFrom is called. Reading starts only when the sequence is iterated. Each iteration opens a fresh reader.

### Memory and ownership

- The producer holds one native Arrow record batch, its dictionary state and one output table. It uses an internal unbuffered table channel, so a slow consumer does not build a queue.
- Arrow v18 reads and decodes every column of that native record. Column selection reduces DataTable conversion, not network bytes or native record memory. A small `batchSize` does **not** cap decoded bytes. `MaxBatchBytes` limits the combined decoded native record buffers and retained dictionary buffers, defaulting to 256 MiB. A negative value is an error. The limit is checked before decompression. Arrow's encoded message limit of 256 MiB and metadata limit of 64 MiB also remain in effect. DataTable cells, encoded buffers and runtime overhead use additional memory, so this is not a process RSS limit.
- Strings and binary cells are copied. Received DataTables stay valid across later batches, cancellation and reader closure. Consumers never call Arrow `Retain` or `Release`.
- Path entry points close their files. `StreamFrom` leaves the source open. Keep it usable during iteration; it is safe to close after the range loop returns.

### Cancellation and errors

The sequence yields each DataTable with a nil error. A failure yields one `(nil, error)` and ends the sequence. An error after earlier batches does not undo those batches.

Breaking or returning from the range loop cancels and drains internal production before the sequence returns. Cleanup also runs if the caller's loop panics, without swallowing that panic. A nil or already cancelled context yields one error without opening the source. Cancellation is checked around reads, conversion and delivery. An `io.ReaderAt` call already in progress cannot be interrupted by this interface, so a remote reader must bind its HTTP requests to the supplied context; early-loop cleanup waits for an in-flight request to return.

Malformed files, truncated input and unsupported selected types return errors. A record with rows but zero columns also returns an error because a DataTable cannot represent those rows. A file with an unsupported field can still be inspected, and `Columns` can exclude that field.

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
| Decimal128 / Decimal256 | Exact `github.com/TimLai666/go-decimal` `decimal.Decimal` |
| Dictionary with supported values | Decoded value in the corresponding type, or `nil` for a null index/value |
| List / LargeList / FixedSizeList, nested lists, Struct, Map, Union, extension types and other unlisted types | Explicit error identifying the selected field and Arrow type |

Dictionary ordering is retained in `FileInfo.Schema`; a DataTable contains values, without category ordering semantics. An ordered int8-to-UTF8 dictionary therefore produces strings, never its int8 codes. A selected unsupported type fails before the first table is emitted, including an unsupported type containing only nulls. Parquet currently records unsupported types on `DataTable.Err()` and substitutes nil cells. This package instead yields a terminal error to avoid confusing unsupported values with real nulls.

## Format references

- [Arrow Go v18.8.0 IPC API](https://pkg.go.dev/github.com/apache/arrow-go/v18@v18.8.0/arrow/ipc)
- [IPC file layout](https://arrow.apache.org/docs/format/Columnar.html#ipc-file-format)
- [Feather V2 is the Arrow IPC file format](https://arrow.apache.org/docs/python/feather.html)
- [Parquet cell mappings](parquet.md#read)
