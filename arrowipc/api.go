// Package arrowipc reads Arrow IPC files and Feather V2. It does not write files.
package arrowipc

import (
	"context"
	"io"
	"iter"

	"github.com/HazelnutParadise/insyra"
	"github.com/apache/arrow-go/v18/arrow"
)

// ReadOptions selects top-level columns. Empty Columns selects all fields.
// A string is an Excel-style index, insyra.Name selects a name, and an int
// selects a zero-based position (negative counts from the end).
type ReadOptions struct {
	Columns []any
	// MaxBatchBytes caps decoded native buffers plus retained dictionaries.
	// Zero uses 256 MiB. It does not include DataTable cell overhead.
	MaxBatchBytes int64
	// MaxOutputBytes caps each table's cell slots and copied string/binary bytes.
	// Zero uses 256 MiB. This is not a process memory limit.
	MaxOutputBytes int64
}

// FileInfo describes metadata without reading record or dictionary values.
type FileInfo struct {
	// Columns lists every top-level field in file order, including fields
	// Stream cannot read.
	Columns []ColumnInfo
	// Metadata is the schema's custom key-value metadata, nil when it has none.
	Metadata   map[string]string
	NumBatches int
	Version    string
}

// ColumnInfo describes one top-level field.
type ColumnInfo struct {
	Name string
	// Type is Arrow's text for the field type, such as "int64",
	// "timestamp[ms, tz=UTC]" or "dictionary<values=utf8, indices=int8, ordered=true>".
	Type     string
	Nullable bool
	// Metadata is the field's custom key-value metadata, nil when it has none.
	Metadata map[string]string
}

// Inspect reads only the header, trailer and footer of path.
func Inspect(path string) (FileInfo, error) { return InspectContext(context.Background(), path) }

// InspectContext is Inspect with cancellation.
func InspectContext(ctx context.Context, path string) (FileInfo, error) {
	schema, batches, version, err := inspectFile(ctx, path)
	if err != nil {
		return FileInfo{}, err
	}
	return newFileInfo(schema, batches, version), nil
}

// InspectFrom inspects a caller-owned random-access source of the given size.
func InspectFrom(ctx context.Context, r io.ReaderAt, size int64) (FileInfo, error) {
	meta, err := inspectSource(ctx, r, size)
	if err != nil {
		return FileInfo{}, err
	}
	return newFileInfo(meta.schema, meta.batches, meta.version), nil
}

func newFileInfo(schema *arrow.Schema, batches int, version string) FileInfo {
	info := FileInfo{NumBatches: batches, Version: version, Metadata: metadataMap(schema.Metadata())}
	info.Columns = make([]ColumnInfo, schema.NumFields())
	for i, field := range schema.Fields() {
		info.Columns[i] = ColumnInfo{Name: field.Name, Type: field.Type.String(), Nullable: field.Nullable, Metadata: metadataMap(field.Metadata)}
	}
	return info
}

func metadataMap(metadata arrow.Metadata) map[string]string {
	if metadata.Len() == 0 {
		return nil
	}
	return metadata.ToMap()
}

// Stream returns a lazy sequence of independently owned DataTable batches.
// Each table has at most batchSize rows. A failure is yielded once as (nil, err).
// Leaving the range loop stops and drains the reader before returning.
func Stream(ctx context.Context, path string, opt ReadOptions, batchSize int) iter.Seq2[*insyra.DataTable, error] {
	opt.Columns = append([]any(nil), opt.Columns...)
	return streamSequence(ctx, func(inner context.Context) (<-chan *insyra.DataTable, <-chan error) {
		return startStream(func(tables chan<- *insyra.DataTable) error {
			return streamPath(inner, path, opt, batchSize, tables)
		})
	})
}

// StreamFrom is Stream for a caller-owned random-access source.
// It never closes r. After iteration returns, r is safe for the caller to close.
func StreamFrom(ctx context.Context, r io.ReaderAt, size int64, opt ReadOptions, batchSize int) iter.Seq2[*insyra.DataTable, error] {
	opt.Columns = append([]any(nil), opt.Columns...)
	return streamSequence(ctx, func(inner context.Context) (<-chan *insyra.DataTable, <-chan error) {
		return startStream(func(tables chan<- *insyra.DataTable) error {
			return streamSource(inner, r, size, opt, batchSize, tables)
		})
	})
}
