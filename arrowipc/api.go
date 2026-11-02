// Package arrowipc reads Arrow IPC files and Feather V2. It does not write files.
package arrowipc

import (
	"context"
	"io"

	"github.com/HazelnutParadise/insyra"
	"github.com/apache/arrow-go/v18/arrow"
)

// ReadOptions selects top-level columns. Empty Columns selects all fields.
type ReadOptions struct {
	Columns []string
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

// Stream sends independently owned tables with at most batchSize rows.
// Drain tables before receiving the terminal error, or cancel ctx when stopping.
func Stream(ctx context.Context, path string, opt ReadOptions, batchSize int) (<-chan *insyra.DataTable, <-chan error) {
	opt.Columns = append([]string(nil), opt.Columns...)
	return startStream(func(tables chan<- *insyra.DataTable) error {
		return streamPath(ctx, path, opt, batchSize, tables)
	})
}

// StreamFrom is Stream for a caller-owned random-access source.
// Keep r open until both returned channels close. r is never closed here.
func StreamFrom(ctx context.Context, r io.ReaderAt, size int64, opt ReadOptions, batchSize int) (<-chan *insyra.DataTable, <-chan error) {
	opt.Columns = append([]string(nil), opt.Columns...)
	return startStream(func(tables chan<- *insyra.DataTable) error {
		return streamSource(ctx, r, size, opt, batchSize, tables)
	})
}
