package parquet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/utils"
	"github.com/apache/arrow/go/v17/arrow"
	"github.com/apache/arrow/go/v17/arrow/memory"
	"github.com/apache/arrow/go/v17/parquet"
	"github.com/apache/arrow/go/v17/parquet/compress"
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

// unreadableFile turns a panic inside the Arrow reader into the error the
// readers return for a file they cannot make sense of. A file whose data pages
// and footer come from different writes makes Arrow dereference a nil pointer,
// and this library never lets a panic reach its caller.
func unreadableFile(label string, r any) error {
	return fmt.Errorf("parquet: %s is not a readable Parquet file: %v", label, r)
}

// damagedFile is the error for a file that read a different number of rows than
// its metadata holds. Arrow reads a page it cannot decode as the end of its row
// group and goes on with the next one, so the count is the only sign of it, and
// groups are the row groups that did not read in full.
func damagedFile(label string, read, want int64, groups []int) error {
	return fmt.Errorf("parquet: %s is damaged: its metadata holds %d rows in the row groups read, but %d were read; row groups %v did not read in full", label, want, read, groups)
}

// Inspect reads a Parquet file's metadata without reading any values.
func Inspect(path string) (info FileInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			info, err = FileInfo{}, unreadableFile(path, r)
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

// errNilContext is what WriteContext and WriteToContext return for a nil
// context, where the standard library would panic.
var errNilContext = errors.New("parquet: nil context")

// Compression is the codec the writers compress every column with.
type Compression int

const (
	// CompressionNone stores the data uncompressed. It is the zero value, and
	// what Write wrote before it took options.
	CompressionNone Compression = iota
	// CompressionSnappy compresses with Snappy.
	CompressionSnappy
	// CompressionGzip compresses with gzip.
	CompressionGzip
	// CompressionBrotli compresses with Brotli.
	CompressionBrotli
	// CompressionZstd compresses with Zstandard.
	CompressionZstd
)

// defaultRowGroupSize is the most rows in one row group when
// WriteOptions.RowGroupSize is zero, and what Write always used before it
// took options.
const defaultRowGroupSize = 1024 * 1024

// WriteOptions are the settings of Write and WriteTo. The zero value writes
// what Write wrote before it took options: uncompressed, with up to 1,048,576
// rows in one row group.
type WriteOptions struct {
	// Compression is the codec for every column. The zero value is
	// CompressionNone.
	Compression Compression
	// RowGroupSize is the most rows one row group holds; zero means
	// 1,048,576. The Arrow writer caps a row group at 67,108,864 rows
	// (parquet.DefaultMaxRowGroupLen), so a larger value writes groups of that
	// size. A negative value is an error.
	RowGroupSize int
}

// resolveWriteOptions reads the settings of Write and WriteTo. More than one
// pack is an error rather than a silent first-wins, so a call that passes two
// says what it did. An empty pack means the zero value, which is what Write
// used before it took settings.
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
		return compress.Codecs.Uncompressed, 0, fmt.Errorf("parquet: unknown Compression %d", opt.Compression)
	}

	rowGroupSize := int64(opt.RowGroupSize)
	if opt.RowGroupSize < 0 {
		return codec, 0, fmt.Errorf("parquet: RowGroupSize must not be negative, got %d", opt.RowGroupSize)
	}
	if rowGroupSize == 0 {
		rowGroupSize = defaultRowGroupSize
	}
	// The Arrow writer caps a row group at DefaultMaxRowGroupLen, as its
	// WriteTable does, so a caller asking for more gets the cap.
	if rowGroupSize > parquet.DefaultMaxRowGroupLen {
		rowGroupSize = parquet.DefaultMaxRowGroupLen
	}
	return codec, rowGroupSize, nil
}

// Write writes dt to path as a Parquet file. It writes to a temporary file of
// its own in the same directory and renames it into place, so a failure
// part-way never leaves a truncated file at path, and two writes to the same
// path never mix their bytes. opts are the settings WriteOptions describes; at
// most one may be given. Write is WriteContext with context.Background().
func Write(dt insyra.IDataTable, path string, opts ...WriteOptions) error {
	return WriteContext(context.Background(), dt, path, opts...)
}

// WriteContext is Write with a context. ctx is checked before the table is
// converted, between columns while converting, before each row group and
// before the finished file replaces path; once it is done, WriteContext
// returns ctx's error and leaves the file at path as it was. A nil ctx is an
// error.
func WriteContext(ctx context.Context, dt insyra.IDataTable, path string, opts ...WriteOptions) error {
	if ctx == nil {
		return errNilContext
	}
	// The settings and ctx are checked before the temporary file is created,
	// so refusing either leaves nothing on disk.
	if _, _, err := resolveWriteOptions(opts); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// The temporary file has a name of its own in path's directory and is
	// renamed into place only when the callback returns nil, so a failure or a
	// cancelled ctx leaves the file at path as it was, and two writes to one
	// path cannot mix their bytes.
	return utils.WriteFileAtomically(path, func(w io.Writer) error {
		if err := WriteToContext(ctx, dt, w, opts...); err != nil {
			return err
		}
		// A context done after the last byte still keeps the finished file
		// out of path.
		return ctx.Err()
	})
}

// WriteTo writes dt as Parquet to any destination — an HTTP response, an S3
// upload, a buffer — the same way Write writes a file. It does not close w:
// the caller owns it. opts work as they do for Write. WriteTo is WriteToContext
// with context.Background().
func WriteTo(dt insyra.IDataTable, w io.Writer, opts ...WriteOptions) error {
	return WriteToContext(context.Background(), dt, w, opts...)
}

// WriteToContext is WriteTo with a context, checked as WriteContext checks it.
// Once ctx is done it returns ctx's error without writing the Parquet footer,
// so what already reached w is not a readable file. A nil ctx is an error.
func WriteToContext(ctx context.Context, dt insyra.IDataTable, w io.Writer, opts ...WriteOptions) error {
	if ctx == nil {
		return errNilContext
	}
	codec, rowGroupSize, err := resolveWriteOptions(opts)
	if err != nil {
		return err
	}
	// Checked before the conversion, so a context that is already done writes
	// nothing at all rather than an empty file.
	if err := ctx.Err(); err != nil {
		return err
	}

	arrowTable, err := dataTableToArrowTable(ctx, dt)
	if err != nil {
		return err
	}
	defer arrowTable.Release()

	createdBy := fmt.Sprintf("go-insyra v%s", insyra.Version)

	// The Parquet writer closes its sink when it is closed, if the sink is an
	// io.Closer. Hiding Close keeps the caller's writer open.
	writer, err := pqarrow.NewFileWriter(arrowTable.Schema(), writerOnly{w}, parquet.NewWriterProperties(parquet.WithCreatedBy(createdBy), parquet.WithCompression(codec)), pqarrow.DefaultWriterProps())
	if err != nil {
		return err
	}

	// The row groups are written here rather than through WriteTable so ctx
	// can be checked between them. The loop is the one WriteTable runs, so an
	// unset WriteOptions writes the same bytes as before.
	writeRowGroup := func(offset, size int64) error {
		writer.NewRowGroup()
		for i := 0; i < int(arrowTable.NumCols()); i++ {
			if err := writer.WriteColumnChunked(arrowTable.Column(i).Data(), offset, size); err != nil {
				return err
			}
		}
		return nil
	}
	rows := arrowTable.NumRows()
	if rows == 0 {
		// Arrow's WriteTable writes one empty row group for an empty table.
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := writeRowGroup(0, 0); err != nil {
			_ = writer.Close()
			return err
		}
	}
	for offset := int64(0); offset < rows; offset += rowGroupSize {
		// Returning without closing the writer is deliberate: Close writes the
		// footer, and a file with a footer is a readable one, so a caller that
		// kept what reached w would read a short table as if it were whole.
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := writeRowGroup(offset, min(rowGroupSize, rows-offset)); err != nil {
			_ = writer.Close()
			return err
		}
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
func readTableFrom(ctx context.Context, src parquet.ReaderAtSeeker, label string, opt ReadOptions) (dt *insyra.DataTable, err error) {
	defer func() {
		if r := recover(); r != nil {
			dt, err = nil, unreadableFile(label, r)
		}
	}()

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

	// The footer holds each row group's row count before any page is read, so
	// it says whether the table that comes back is all of what was asked for.
	// An index out of range is left to ReadRowGroups, which reports it better,
	// so the comparison below is skipped when one is there or when no row group
	// was selected at all, which leaves no row group to name.
	counts := make([]int64, len(rowGroups))
	var want int64
	countable := len(rowGroups) > 0
	for i, rg := range rowGroups {
		if rg < 0 || rg >= r.NumRowGroups() {
			countable = false
			break
		}
		counts[i] = r.MetaData().RowGroup(rg).NumRows()
		want += counts[i]
	}

	var arrowTable arrow.Table
	arrowTable, err = fr.ReadRowGroups(ctx, colIndices, rowGroups)
	if err != nil {
		return nil, err
	}
	if countable && arrowTable.NumRows() != want {
		// Read before releasing: the table's own fields do not outlive it.
		read := arrowTable.NumRows()
		arrowTable.Release()
		colIdx := colIndices
		if len(colIdx) == 0 {
			schema := r.MetaData().Schema
			colIdx = make([]int, schema.NumColumns())
			for i := 0; i < schema.NumColumns(); i++ {
				colIdx[i] = i
			}
		}
		return nil, damagedFile(label, read, want, shortRowGroups(ctx, fr, colIdx, rowGroups, counts))
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

// shortRowGroups reads each of rowGroups on its own and returns those whose row
// count differs from counts, the metadata's, in order. It runs only once a read
// has come up short, to name the damaged row groups.
func shortRowGroups(ctx context.Context, fr *pqarrow.FileReader, colIndices []int, rowGroups []int, counts []int64) []int {
	var res []int
	for i, rg := range rowGroups {
		t, err := fr.ReadRowGroups(ctx, colIndices, []int{rg})
		if err != nil {
			res = append(res, rg)
			continue
		}
		if t.NumRows() != counts[i] {
			t.Release()
			res = append(res, rg)
			continue
		}
		t.Release()
	}
	return res
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
