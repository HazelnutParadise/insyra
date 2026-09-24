package parquet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"

	"github.com/HazelnutParadise/insyra"
	"github.com/apache/arrow/go/v17/arrow"
	"github.com/apache/arrow/go/v17/arrow/memory"
	"github.com/apache/arrow/go/v17/parquet"
	"github.com/apache/arrow/go/v17/parquet/file"
	"github.com/apache/arrow/go/v17/parquet/pqarrow"
)

// ReadOptions selects what a read covers. An empty field means everything.
type ReadOptions struct {
	Columns   []string // empty=all
	RowGroups []int    // empty=all
}

// ReadColumnOptions are the options ReadColumn takes. They are separate from
// ReadOptions so a limit that only makes sense for one column does not appear on
// every read.
type ReadColumnOptions struct {
	RowGroups []int // empty=all
	MaxValues int64 // 0=no limit; if exceeded, return error to avoid RAM explosion
}

// FileInfo is what Inspect reports about a file.
type FileInfo struct {
	NumRows      int64
	NumRowGroups int
	Version      string
	CreatedBy    string
	Metadata     map[string]string
	Columns      []ColumnInfo
	RowGroups    []RowGroupInfo
}

// ColumnInfo describes one column's schema.
type ColumnInfo struct {
	Name         string
	PhysicalType string
	LogicalType  string
	Repetition   string
}

// RowGroupInfo describes one row group's size.
type RowGroupInfo struct {
	NumRows             int64
	TotalByteSize       int64
	TotalCompressedSize int64
}

// Inspect reads a Parquet file's metadata without reading any values.
func Inspect(path string) (FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return FileInfo{}, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			// Sometimes the underlying writer/reader already closed the file.
			// Ignore "file already closed" as it's harmless; log others.
			if errors.Is(err, os.ErrClosed) {
				return
			}
			insyra.LogWarning("parquet", "close", "failed to close file %s: %v", path, err)
		}
	}()

	r, err := file.NewParquetReader(f)
	if err != nil {
		return FileInfo{}, err
	}
	defer func() {
		if err := r.Close(); err != nil {
			insyra.LogWarning("parquet", "close", "failed to close reader for %s: %v", path, err)
		}
	}()

	metadata := r.MetaData()
	schema := metadata.Schema

	kv := make(map[string]string)
	kvMeta := metadata.KeyValueMetadata()
	if kvMeta != nil {
		keys := kvMeta.Keys()
		values := kvMeta.Values()
		for i := 0; i < len(keys); i++ {
			kv[keys[i]] = values[i]
		}
	}

	createdBy := ""
	if wv := metadata.WriterVersion(); wv != nil {
		createdBy = wv.App
	}

	info := FileInfo{
		NumRows:      r.NumRows(),
		NumRowGroups: r.NumRowGroups(),
		Version:      metadata.Version().String(),
		CreatedBy:    createdBy,
		Metadata:     kv,
		Columns:      make([]ColumnInfo, schema.NumColumns()),
		RowGroups:    make([]RowGroupInfo, r.NumRowGroups()),
	}

	for i := 0; i < schema.NumColumns(); i++ {
		col := schema.Column(i)
		info.Columns[i] = ColumnInfo{
			Name:         col.Name(),
			PhysicalType: col.PhysicalType().String(),
			LogicalType:  col.LogicalType().String(),
			Repetition:   col.SchemaNode().RepetitionType().String(),
		}
	}

	for i := 0; i < r.NumRowGroups(); i++ {
		rg := metadata.RowGroup(i)
		info.RowGroups[i] = RowGroupInfo{
			NumRows:             rg.NumRows(),
			TotalByteSize:       rg.TotalByteSize(),
			TotalCompressedSize: rg.TotalCompressedSize(),
		}
	}

	return info, nil
}

// Write writes dt to path as a Parquet file. It writes to a sibling temporary
// file and renames it into place, so a failure part-way never leaves a truncated
// file at path.
func Write(dt insyra.IDataTable, path string) error {
	// Write to a sibling temp file and rename so a failure part-way never
	// leaves a truncated file at path (same shape as ApplyCCL).
	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	cleanup := func() { _ = os.Remove(tmpPath) }

	if err := WriteTo(dt, f); err != nil {
		_ = f.Close()
		cleanup()
		return err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return fmt.Errorf("parquet: failed to close %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("parquet: failed to replace %s: %w", path, err)
	}
	return nil
}

// WriteTo writes dt as Parquet to any destination — an HTTP response, an S3
// upload, a buffer — the same way Write writes a file. It does not close w:
// the caller owns it.
func WriteTo(dt insyra.IDataTable, w io.Writer) error {
	arrowTable, err := dataTableToArrowTable(dt)
	if err != nil {
		return err
	}
	defer arrowTable.Release()

	createdBy := fmt.Sprintf("go-insyra v%s", insyra.Version)

	// The Parquet writer closes its sink when it is closed, if the sink is an
	// io.Closer. Hiding Close keeps the caller's writer open.
	writer, err := pqarrow.NewFileWriter(arrowTable.Schema(), writerOnly{w}, parquet.NewWriterProperties(parquet.WithCreatedBy(createdBy)), pqarrow.DefaultWriterProps())
	if err != nil {
		return err
	}
	if err := writer.WriteTable(arrowTable, 1024*1024); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("parquet: failed to close writer: %w", err)
	}
	return nil
}

// writerOnly exposes only Write, so a writer handed to WriteTo is never
// closed on the caller's behalf.
type writerOnly struct{ w io.Writer }

func (o writerOnly) Write(p []byte) (int, error) { return o.w.Write(p) }

// Read reads a Parquet file into a DataTable in one go. For a file too large to
// hold in memory, use Stream.
func Read(ctx context.Context, path string, opt ReadOptions) (*insyra.DataTable, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			// Reader.Close() may already close the underlying file.
			if errors.Is(err, os.ErrClosed) {
				return
			}
			insyra.LogWarning("parquet", "close", "failed to close file %s: %v", path, err)
		}
	}()
	return readTableFrom(ctx, f, path, opt)
}

// ReadFrom reads a Parquet file from any source with random access — an open
// file, a *bytes.Reader, an S3 range reader — the same way Read reads a path.
// Parquet keeps its index at the end of the file, so reading needs to seek:
// hence an io.ReaderAt and the total size rather than a plain io.Reader.
func ReadFrom(ctx context.Context, r io.ReaderAt, size int64, opt ReadOptions) (*insyra.DataTable, error) {
	return readTableFrom(ctx, io.NewSectionReader(r, 0, size), "the Parquet input", opt)
}

// readTableFrom is the one reader behind Read and ReadFrom. label names the
// source in messages.
func readTableFrom(ctx context.Context, src parquet.ReaderAtSeeker, label string, opt ReadOptions) (*insyra.DataTable, error) {
	r, err := file.NewParquetReader(src)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := r.Close(); err != nil {
			insyra.LogWarning("parquet", "close", "failed to close reader for %s: %v", label, err)
		}
	}()

	fr, err := pqarrow.NewFileReader(r, pqarrow.ArrowReadProperties{Parallel: true}, memory.DefaultAllocator)
	if err != nil {
		return nil, err
	}

	var colIndices []int
	if len(opt.Columns) > 0 {
		schema := r.MetaData().Schema
		for _, colName := range opt.Columns {
			idx := schema.ColumnIndexByName(colName)
			if idx == -1 {
				return nil, fmt.Errorf("column %s not found", colName)
			}
			colIndices = append(colIndices, idx)
		}
	} else {
		// 如果未指定 Columns，預設讀取所有欄位
		schema := r.MetaData().Schema
		colIndices = make([]int, schema.NumColumns())
		for i := 0; i < schema.NumColumns(); i++ {
			colIndices[i] = i
		}
	}

	rowGroups := opt.RowGroups
	if len(rowGroups) == 0 {
		// 如果未指定 RowGroups，預設讀取所有 RowGroups
		numRG := r.NumRowGroups()
		rowGroups = make([]int, numRG)
		for i := range rowGroups {
			rowGroups[i] = i
		}
	}

	var arrowTable arrow.Table
	arrowTable, err = fr.ReadRowGroups(ctx, colIndices, rowGroups)
	if err != nil {
		return nil, err
	}
	defer arrowTable.Release()

	// 手動將 arrow.Table 轉換為 insyra.DataTable
	dataTable := insyra.NewDataTable()
	for i := 0; i < int(arrowTable.NumCols()); i++ {
		col := arrowTable.Column(i)
		data := chunkedToSlice(col.Data())
		// getVal already yields nil for a type it cannot represent; the reason
		// has to be recorded here, where the column's name and type are known.
		if !supportedArrowType(col.DataType()) {
			dataTable.SetErr("parquet", "Read", unsupportedColumnMsg, col.Name(), col.DataType())
		}
		dataTable.AppendCols(newColumn(data, col.Name()))
	}

	return dataTable, nil
}

// Stream reads a Parquet file batch by batch. Range over it:
//
//	for dt, err := range parquet.Stream(ctx, path, parquet.ReadOptions{}, 1000) {
//		if err != nil {
//			return err
//		}
//		// use dt
//	}
//
// Each batch arrives as a DataTable with a nil error. A failure arrives once,
// as a nil table and the error, and ends the sequence. Leaving the loop early
// stops the reader, so nothing is left running whether or not ctx is
// cancelled. Cancelling ctx ends the sequence with the context's error.
func Stream(ctx context.Context, path string, opt ReadOptions, batchSize int) iter.Seq2[*insyra.DataTable, error] {
	return streamSeq(ctx, func(inner context.Context) (<-chan arrow.Record, <-chan error) {
		return streamAsArrowRecord(inner, path, opt, batchSize)
	})
}

// StreamFrom reads a Parquet file batch by batch from any source with random
// access, the same way Stream reads a path; see ReadFrom for why it takes an
// io.ReaderAt and the size.
func StreamFrom(ctx context.Context, r io.ReaderAt, size int64, opt ReadOptions, batchSize int) iter.Seq2[*insyra.DataTable, error] {
	return streamSeq(ctx, func(inner context.Context) (<-chan arrow.Record, <-chan error) {
		return streamArrowRecordsFrom(inner, io.NewSectionReader(r, 0, size), "the Parquet input", opt, batchSize, nil)
	})
}

// streamSeq turns a record stream into the iterator Stream and StreamFrom
// return. The reader answers to a context of its own, so returning from here
// — the loop ended, or the caller broke out of it — stops the reader too.
func streamSeq(ctx context.Context, start func(context.Context) (<-chan arrow.Record, <-chan error)) iter.Seq2[*insyra.DataTable, error] {
	return func(yield func(*insyra.DataTable, error) bool) {
		inner, cancel := context.WithCancel(ctx)
		defer cancel()
		recChan, recErrChan := start(inner)
		dtChan, errChan := streamTables(inner, recChan, recErrChan)
		for dt := range dtChan {
			if !yield(dt, nil) {
				return
			}
		}
		if err := <-errChan; err != nil {
			yield(nil, err)
		}
	}
}

// streamTables is the reader behind Stream: it sends each batch on dtChan and
// the one error, if any, on errChan, and stops when ctx is cancelled.
func streamTables(ctx context.Context, recChan <-chan arrow.Record, internalErrChan <-chan error) (<-chan *insyra.DataTable, <-chan error) {
	dtChan := make(chan *insyra.DataTable)
	errChan := make(chan error, 1)

	go func() {
		defer close(dtChan)
		defer close(errChan)

		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			case err := <-internalErrChan:
				if err != nil {
					errChan <- err
				}
				return // Stream finished or errored
			case rec, ok := <-recChan:
				if !ok {
					// The reader closes its error channel before its record
					// channel, so this receive cannot block. Read it rather than
					// letting the select choose between two ready cases, which
					// would drop an error reported after the last batch and end
					// the stream as if the file had been read in full.
					if err := <-internalErrChan; err != nil {
						errChan <- err
					}
					return
				}
				dt := recordToDataTable(rec)
				rec.Release()

				select {
				case <-ctx.Done():
					errChan <- ctx.Err()
					return
				case dtChan <- dt:
				}
			}
		}
	}()

	return dtChan, errChan
}

// ReadColumn reads a single column from a Parquet file into a DataList.
// When opt.MaxValues > 0, the row count of the selected row groups (all when
// none are selected) is taken from the file metadata first, and an error is
// returned before any value is read if it exceeds the limit.
func ReadColumn(ctx context.Context, path string, column string, opt ReadColumnOptions) (*insyra.DataList, error) {
	if opt.MaxValues > 0 {
		n, err := selectedRowCount(path, opt.RowGroups)
		if err != nil {
			return nil, err
		}
		if n > opt.MaxValues {
			return nil, fmt.Errorf("column %s holds %d values in the selected row groups, exceeding MaxValues %d", column, n, opt.MaxValues)
		}
	}
	table, err := Read(ctx, path, ReadOptions{Columns: []string{column}, RowGroups: opt.RowGroups})
	if err != nil {
		return nil, err
	}
	list := table.GetCol("A")
	// Err() lives on the table, and the caller only gets the list.
	if e := table.Err(); e != nil {
		list.SetErr("parquet", "ReadColumn", "%s", e.Message)
	}
	return list, nil
}

// selectedRowCount sums the row counts of the given row groups from the file
// metadata, or of every row group when none are given.
func selectedRowCount(path string, rowGroups []int) (int64, error) {
	info, err := Inspect(path)
	if err != nil {
		return 0, err
	}
	if len(rowGroups) == 0 {
		return info.NumRows, nil
	}
	var n int64
	for _, rg := range rowGroups {
		if rg < 0 || rg >= len(info.RowGroups) {
			return 0, fmt.Errorf("row group %d out of range (file has %d)", rg, len(info.RowGroups))
		}
		n += info.RowGroups[rg].NumRows
	}
	return n, nil
}
