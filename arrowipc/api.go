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
// Schema is independent of the reader lifetime and needs no Release call.
type FileInfo struct {
	Schema     *arrow.Schema
	NumBatches int
	Version    string
}

// Inspect reads only the header, trailer and footer of path.
func Inspect(path string) (FileInfo, error) { return InspectContext(context.Background(), path) }

// InspectContext is Inspect with cancellation.
func InspectContext(ctx context.Context, path string) (FileInfo, error) {
	return inspectFile(ctx, path)
}

// InspectFrom inspects a caller-owned random-access source of the given size.
func InspectFrom(ctx context.Context, r io.ReaderAt, size int64) (FileInfo, error) {
	return inspectSource(ctx, r, size)
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
