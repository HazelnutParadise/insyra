package parquet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/utils"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/compress"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
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
func Inspect(path string) (info FileInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			info = FileInfo{}
			err = unreadableFile(path, r)
		}
	}()
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

	r, err := file.NewParquetReader(f, file.WithReadProps(readerProperties()))
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

	info = FileInfo{
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

// Write writes dt to path as a Parquet file using the default WriteOptions.
// The target is opened in place and closed after writing.
func Write(dt insyra.IDataTable, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	writeErr := writeFileContents(context.Background(), dt, f)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return fmt.Errorf("parquet: failed to close writer: %w", closeErr)
	}
	return nil
}

// Compression selects the codec used for every column.
type Compression int

const (
	CompressionNone Compression = iota
	CompressionSnappy
	CompressionGzip
	CompressionBrotli
	CompressionZstd
)

const defaultRowGroupSize = 1024 * 1024

// WriteOptions configures Parquet compression and row group size.
type WriteOptions struct {
	// Compression selects a codec. CompressionNone is the zero value.
	Compression Compression
	// RowGroupSize is the maximum rows per row group. Zero uses the default.
	RowGroupSize int
}

func resolveWriteOptions(opts []WriteOptions) (compress.Compression, int64, error) {
	if len(opts) > 1 {
		return compress.Codecs.Uncompressed, 0, fmt.Errorf("parquet: at most one WriteOptions may be given, got %d", len(opts))
	}
	var opt WriteOptions
	if len(opts) == 1 {
		opt = opts[0]
	}
	var codec compress.Compression
	switch opt.Compression {
	case CompressionNone:
		codec = compress.Codecs.Uncompressed
	case CompressionSnappy:
		codec = compress.Codecs.Snappy
	case CompressionGzip:
		codec = compress.Codecs.Gzip
	case CompressionBrotli:
		codec = compress.Codecs.Brotli
	case CompressionZstd:
		codec = compress.Codecs.Zstd
	default:
		return codec, 0, fmt.Errorf("parquet: unknown Compression %d", opt.Compression)
	}
	if opt.RowGroupSize < 0 {
		return codec, 0, fmt.Errorf("parquet: RowGroupSize must not be negative, got %d", opt.RowGroupSize)
	}
	size := int64(opt.RowGroupSize)
	if size == 0 {
		size = defaultRowGroupSize
	}
	if size > parquet.DefaultMaxRowGroupLen {
		size = parquet.DefaultMaxRowGroupLen
	}
	return codec, size, nil
}

var errNilContext = errors.New("parquet: nil context")

func WriteContext(ctx context.Context, dt insyra.IDataTable, path string, opts ...WriteOptions) error {
	if ctx == nil {
		return errNilContext
	}
	if _, _, err := resolveWriteOptions(opts); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return utils.WriteFileAtomically(path, func(w io.Writer) error {
		if err := writeFileContents(ctx, dt, w, opts...); err != nil {
			return err
		}
		return ctx.Err()
	})
}

// WriteTo writes a Parquet file to w without closing the caller's writer.
func WriteTo(dt insyra.IDataTable, w io.Writer, opts ...WriteOptions) error {
	return WriteToContext(context.Background(), dt, w, opts...)
}
func WriteToContext(ctx context.Context, dt insyra.IDataTable, w io.Writer, opts ...WriteOptions) error {
	return writeFileContents(ctx, dt, w, opts...)
}

func writeFileContents(ctx context.Context, dt insyra.IDataTable, w io.Writer, opts ...WriteOptions) error {
	if ctx == nil {
		return errNilContext
	}
	codec, groupSize, err := resolveWriteOptions(opts)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	arrowTable, err := dataTableToArrowTable(dt)
	if err != nil {
		return err
	}
	defer arrowTable.Release()
	createdBy := fmt.Sprintf("go-insyra v%s", insyra.Version)
	writer, err := pqarrow.NewFileWriter(arrowTable.Schema(), writerOnly{w}, parquet.NewWriterProperties(parquet.WithCreatedBy(createdBy), parquet.WithCompression(codec)), pqarrow.DefaultWriterProps())
	if err != nil {
		return err
	}
	rows := arrowTable.NumRows()
	writeGroup := func(offset, size int64) error {
		if err := writer.NewRowGroupChecked(); err != nil {
			return err
		}
		for i := 0; i < int(arrowTable.NumCols()); i++ {
			if err := writer.WriteColumnChunked(arrowTable.Column(i).Data(), offset, size); err != nil {
				return err
			}
		}
		return nil
	}
	if rows == 0 {
		if err := writeGroup(0, 0); err != nil {
			_ = writer.Close()
			return err
		}
	}
	for offset := int64(0); offset < rows; offset += groupSize {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := writeGroup(offset, min(groupSize, rows-offset)); err != nil {
			_ = writer.Close()
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("parquet: failed to close writer: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

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

// ReadFrom reads a Parquet file from any random access source.
func ReadFrom(ctx context.Context, r io.ReaderAt, size int64, opt ReadOptions) (*insyra.DataTable, error) {
	return readTableFrom(ctx, io.NewSectionReader(panicSafeReaderAt{r}, 0, size), "the Parquet input", opt)
}

type panicSafeReaderAt struct{ io.ReaderAt }

func (r panicSafeReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	defer func() {
		if v := recover(); v != nil {
			n = 0
			err = unreadableFile("the Parquet input", v)
		}
	}()
	return r.ReaderAt.ReadAt(p, off)
}

func readTableFrom(ctx context.Context, src parquet.ReaderAtSeeker, label string, opt ReadOptions) (dt *insyra.DataTable, err error) {
	defer func() {
		if r := recover(); r != nil {
			dt = nil
			err = unreadableFile(label, r)
		}
	}()
	r, err := file.NewParquetReader(src, file.WithReadProps(readerProperties()))
	if err != nil {
		return nil, err
	}
	if err := normalizeStoredDurationSchema(r); err != nil {
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
	var expectedRows int64
	counts := make([]int64, len(rowGroups))
	countable := len(rowGroups) > 0
	for i, rg := range rowGroups {
		if rg < 0 || rg >= r.NumRowGroups() {
			countable = false
			break
		}
		counts[i] = r.MetaData().RowGroup(rg).NumRows()
		expectedRows += counts[i]
	}

	var arrowTable arrow.Table
	arrowTable, err = fr.ReadRowGroups(ctx, colIndices, rowGroups)
	if err != nil {
		return nil, readerFailed(ctx, label, err, src, colIndices, rowGroups, counts, countable)
	}
	if countable && int64(arrowTable.NumRows()) != expectedRows {
		got := arrowTable.NumRows()
		arrowTable.Release()
		return nil, damagedFile(label, int64(got), expectedRows, shortRowGroupsFrom(ctx, src, colIndices, rowGroups, counts))
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

// Stream reads a Parquet file batch by batch, sending each batch as a DataTable.
// The caller must either read dtChan to completion or cancel ctx. The channel
// is unbuffered, so a consumer that stops part-way leaves the producing
// goroutine parked on its next send for the life of the process. Ranging over
// dtChan and then checking errChan does the right thing; breaking out of the
// range does not unless ctx is cancelled:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//	dtChan, errChan := parquet.Stream(ctx, path, parquet.ReadOptions{}, 1000)
//	for dt := range dtChan { … }
//	if err := <-errChan; err != nil { … }
func Stream(ctx context.Context, path string, opt ReadOptions, batchSize int) (<-chan *insyra.DataTable, <-chan error) {
	return streamTablesFrom(ctx, func() (<-chan arrow.RecordBatch, <-chan error) { return streamAsArrowRecord(ctx, path, opt, batchSize) })
}

// StreamFrom streams batches from a random-access source using the same channel contract as Stream.
func StreamFrom(ctx context.Context, r io.ReaderAt, size int64, opt ReadOptions, batchSize int) (<-chan *insyra.DataTable, <-chan error) {
	return streamTablesFrom(ctx, func() (<-chan arrow.RecordBatch, <-chan error) {
		return streamArrowRecordsFrom(ctx, io.NewSectionReader(panicSafeReaderAt{r}, 0, size), "the Parquet input", opt, batchSize, nil)
	})
}

func streamTablesFrom(ctx context.Context, start func() (<-chan arrow.RecordBatch, <-chan error)) (<-chan *insyra.DataTable, <-chan error) {
	dtChan := make(chan *insyra.DataTable)
	errChan := make(chan error, 1)

	go func() {
		defer close(dtChan)
		defer close(errChan)

		recChan, internalErrChan := start()

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
