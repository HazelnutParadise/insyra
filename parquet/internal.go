package parquet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
	"github.com/TimLai666/go-decimal/decimal"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
)

// streamAsArrowRecord：串流讀（可選欄、可選包），一批一批吐出 arrow.RecordBatch
func streamAsArrowRecord(ctx context.Context, path string, opt ReadOptions, batchSize int) (<-chan arrow.RecordBatch, <-chan error) {
	f, err := os.Open(path)
	if err != nil {
		recChan := make(chan arrow.RecordBatch)
		errChan := make(chan error, 1)
		errChan <- err
		close(errChan)
		close(recChan)
		return recChan, errChan
	}
	return streamArrowRecordsFrom(ctx, f, path, opt, batchSize, func() {
		if err := f.Close(); err != nil {
			// Reader.Close() may already close the underlying file, the same
			// way Read and Inspect allow for. Without this guard every
			// successful Stream, FilterWithCCL and ApplyCCL logged a warning.
			if errors.Is(err, os.ErrClosed) {
				return
			}
			insyra.LogWarning("parquet", "close", "failed to close file %s: %v", path, err)
		}
	})
}

// streamArrowRecordsFrom streams record batches from any seekable source.
// label names the source in messages, and done, when given, runs as the
// reader finishes. The error channel is closed before the record channel,
// which the consumers rely on to never trade a late error for a partial
// result.
func streamArrowRecordsFrom(ctx context.Context, src parquet.ReaderAtSeeker, label string, opt ReadOptions, batchSize int, done func()) (<-chan arrow.RecordBatch, <-chan error) {
	recChan := make(chan arrow.RecordBatch)
	errChan := make(chan error, 1)

	go func() {
		defer close(recChan)
		defer close(errChan)
		if done != nil {
			defer done()
		}
		// A panic in this goroutine would take the whole process down, and it
		// cannot be recovered by the caller reading these channels. It runs
		// before the channels close, so the error reaches the consumer; the
		// select's default keeps it from blocking on a channel that already
		// holds an error.
		defer func() {
			if r := recover(); r != nil {
				select {
				case errChan <- unreadableFile(label, r):
				default:
				}
			}
		}()

		r, err := file.NewParquetReader(src)
		if err != nil {
			errChan <- err
			return
		}
		defer func() {
			if err := r.Close(); err != nil {
				insyra.LogWarning("parquet", "close", "failed to close reader for %s: %v", label, err)
			}
		}()

		fr, err := pqarrow.NewFileReader(r, pqarrow.ArrowReadProperties{Parallel: true, BatchSize: int64(batchSize)}, memory.DefaultAllocator)
		if err != nil {
			errChan <- err
			return
		}

		var colIndices []int
		if len(opt.Columns) > 0 {
			schema := r.MetaData().Schema
			for _, colName := range opt.Columns {
				idx := schema.ColumnIndexByName(colName)
				if idx == -1 {
					errChan <- fmt.Errorf("column %s not found", colName)
					return
				}
				colIndices = append(colIndices, idx)
			}
		}

		rowGroups := opt.RowGroups
		if len(rowGroups) == 0 {
			numRG := r.NumRowGroups()
			rowGroups = make([]int, numRG)
			for i := range numRG {
				rowGroups[i] = i
			}
		}

		// The footer holds each row group's row count before any page is read, so
		// it says whether the records that come out are all of what was asked
		// for. An index out of range is left to GetRecordReader, which reports
		// it better, so the comparison below is skipped when one is there or
		// when no row group was selected at all, which leaves no row group to
		// name.
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

		rr, err := fr.GetRecordReader(ctx, colIndices, rowGroups)
		if err != nil {
			errChan <- err
			return
		}
		defer rr.Release()

		var read int64
		for rr.Next() {
			rec := rr.RecordBatch()
			rec.Retain()
			read += rec.NumRows()
			select {
			case <-ctx.Done():
				rec.Release()
				errChan <- ctx.Err()
				return
			case recChan <- rec:
			}
		}
		// shortRowGroups reads the same columns again; none selected means all.
		cols := colIndices
		if len(cols) == 0 {
			schema := r.MetaData().Schema
			cols = make([]int, schema.NumColumns())
			for i := 0; i < schema.NumColumns(); i++ {
				cols[i] = i
			}
		}
		if rr.Err() != nil && !errors.Is(rr.Err(), io.EOF) {
			errChan <- readerFailed(ctx, label, rr.Err(), fr, cols, rowGroups, counts, countable)
			return
		}
		// A reader that answers a page it cannot decode with no error at all is
		// still caught here: the rows the footer promised are what says the file
		// was damaged.
		if countable && read != want {
			errChan <- damagedFile(label, read, want, shortRowGroups(ctx, fr, cols, rowGroups, counts))
		}
	}()

	return recChan, errChan
}

func chunkedToSlice(chunked *arrow.Chunked) any {
	if chunked.Len() == 0 {
		return nil
	}

	// 若欄位含 null，typed 快速路徑會直接複製值緩衝區（null 位置為 0/""/false），
	// 造成 null 靜默遺失。此時改走保留 nil 的 []any 路徑；無 null 時維持原本高效的
	// typed slice 輸出。
	for _, chunk := range chunked.Chunks() {
		if chunk.NullN() > 0 {
			res := make([]any, 0, chunked.Len())
			for _, c := range chunked.Chunks() {
				for i := 0; i < c.Len(); i++ {
					if c.IsNull(i) {
						res = append(res, nil)
					} else {
						res = append(res, getVal(c, i))
					}
				}
			}
			return res
		}
	}

	dataType := chunked.DataType()

	switch dataType.ID() {
	case arrow.INT64:
		res := make([]int64, 0, chunked.Len())
		for _, chunk := range chunked.Chunks() {
			arr := chunk.(*array.Int64)
			res = append(res, arr.Int64Values()...)
		}
		return res
	case arrow.INT32:
		res := make([]int32, 0, chunked.Len())
		for _, chunk := range chunked.Chunks() {
			arr := chunk.(*array.Int32)
			res = append(res, arr.Int32Values()...)
		}
		return res
	case arrow.FLOAT64:
		res := make([]float64, 0, chunked.Len())
		for _, chunk := range chunked.Chunks() {
			arr := chunk.(*array.Float64)
			res = append(res, arr.Float64Values()...)
		}
		return res
	case arrow.FLOAT32:
		res := make([]float32, 0, chunked.Len())
		for _, chunk := range chunked.Chunks() {
			arr := chunk.(*array.Float32)
			res = append(res, arr.Float32Values()...)
		}
		return res
	case arrow.STRING:
		res := make([]string, 0, chunked.Len())
		for _, chunk := range chunked.Chunks() {
			arr := chunk.(*array.String)
			for i := 0; i < arr.Len(); i++ {
				res = append(res, arr.Value(i))
			}
		}
		return res
	case arrow.BOOL:
		res := make([]bool, 0, chunked.Len())
		for _, chunk := range chunked.Chunks() {
			arr := chunk.(*array.Boolean)
			for i := 0; i < arr.Len(); i++ {
				res = append(res, arr.Value(i))
			}
		}
		return res
	default:
		// Fallback to []any
		res := make([]any, 0, chunked.Len())
		for _, chunk := range chunked.Chunks() {
			for i := 0; i < chunk.Len(); i++ {
				if chunk.IsNull(i) {
					res = append(res, nil)
				} else {
					res = append(res, getVal(chunk, i))
				}
			}
		}
		return res
	}
}

// unsupportedColumnMsg is the reason recorded on a table or list holding a
// column whose Arrow type has no faithful Go representation.
const unsupportedColumnMsg = "column %q: unsupported Arrow column type %s; its cells were read as nil"

// supportedArrowType reports whether a column of this type can be read into a
// Go value that means what the file says.
//
// It must stay in step with getVal: a type listed here and missing from getVal
// reads as nil while claiming to be supported, and a type getVal handles but
// this does not makes the reader report a column it read perfectly well.
// TestSupportedTypesAreExactlyWhatGetValHandles pins the two together.
func supportedArrowType(dt arrow.DataType) bool {
	switch dt.ID() {
	case arrow.INT8, arrow.INT16, arrow.INT32, arrow.INT64,
		arrow.UINT8, arrow.UINT16, arrow.UINT32, arrow.UINT64,
		arrow.FLOAT32, arrow.FLOAT64,
		arrow.BOOL,
		arrow.STRING, arrow.LARGE_STRING,
		arrow.BINARY, arrow.LARGE_BINARY, arrow.FIXED_SIZE_BINARY,
		arrow.TIMESTAMP, arrow.DATE32, arrow.DATE64,
		arrow.DECIMAL128, arrow.DECIMAL256,
		arrow.NULL:
		return true
	case arrow.DICTIONARY:
		// A dictionary is read through its values, so it is as readable as
		// they are.
		return supportedArrowType(dt.(*arrow.DictionaryType).ValueType)
	}
	return false
}

// getVal reads one cell.
//
// The default arm returns nil rather than a value, because a value that does
// not mean what the file says is worse than no value at all. It used to return
// arr.String() — the string form of the whole array, ignoring i — so every row
// of a column this switch does not cover read back as one identical string,
// with nothing reported anywhere (#371).
//
// Callers must check supportedArrowType at column level so the reason reaches
// the caller; a nil here on its own is indistinguishable from a null cell.
func getVal(arr arrow.Array, i int) any {
	switch a := arr.(type) {
	case *array.Int64:
		return a.Value(i)
	case *array.Int32:
		return a.Value(i)
	case *array.Int16:
		return a.Value(i)
	case *array.Int8:
		return a.Value(i)
	case *array.Uint64:
		// Stays uint64: a value above 2^63 does not fit an int64, and since
		// match-integers-by-value an integer is matched by value whatever its
		// Go type.
		return a.Value(i)
	case *array.Uint32:
		return a.Value(i)
	case *array.Uint16:
		return a.Value(i)
	case *array.Uint8:
		return a.Value(i)
	case *array.Float64:
		return a.Value(i)
	case *array.Float32:
		return a.Value(i)
	case *array.String:
		return a.Value(i)
	case *array.LargeString:
		return a.Value(i)
	case *array.Boolean:
		return a.Value(i)
	case *array.Binary:
		// []byte, so a binary column is not indistinguishable from a text one.
		// The value is copied because Arrow owns the buffer behind it.
		return append([]byte(nil), a.Value(i)...)
	case *array.LargeBinary:
		return append([]byte(nil), a.Value(i)...)
	case *array.FixedSizeBinary:
		return append([]byte(nil), a.Value(i)...)
	case *array.Timestamp:
		return a.Value(i).ToTime(a.DataType().(*arrow.TimestampType).Unit)
	case *array.Date32:
		return a.Value(i).ToTime().UTC()
	case *array.Date64:
		return a.Value(i).ToTime().UTC()
	case *array.Decimal128:
		// NewFromScaledInt rounds and normalises nothing, and the coefficient
		// is a big.Int, so the full 38-digit range survives. A float64 would
		// lose the exactness that is the whole reason a column is decimal.
		n := a.Value(i).BigInt()
		return decimal.NewFromScaledInt(n, a.DataType().(*arrow.Decimal128Type).Scale)
	case *array.Decimal256:
		n := a.Value(i).BigInt()
		return decimal.NewFromScaledInt(n, a.DataType().(*arrow.Decimal256Type).Scale)
	case *array.Null:
		// Every cell of a null-typed column is null, and nil is exactly that.
		return nil
	case *array.Dictionary:
		// A file that stores its Arrow schema hands a dictionary column over
		// as indices into a values array; the cell is the value its index
		// points to. A null index or a null value is nil.
		if a.IsNull(i) {
			return nil
		}
		values, idx := a.Dictionary(), a.GetValueIndex(i)
		if values.IsNull(idx) {
			return nil
		}
		return getVal(values, idx)
	default:
		return nil
	}
}

// newColumn builds a column from what chunkedToSlice returned.
//
// A []any is appended rather than handed to NewDataList, because the
// constructor flattens every slice and a binary column's cells are []byte.
// For anything else the two are the same: flattening a []any of scalars
// already produces one cell per element.
func newColumn(data any, name string) *insyra.DataList {
	if vals, ok := data.([]any); ok {
		return insyra.NewDataList().Append(vals...).SetName(name)
	}
	return insyra.NewDataList(data).SetName(name)
}

func recordToDataTable(rec arrow.RecordBatch) *insyra.DataTable {
	dataTable := insyra.NewDataTable()
	if rec == nil {
		return dataTable
	}

	for i, col := range rec.Columns() {
		// col is an arrow.Array, we can wrap it in a Chunked to reuse chunkedToSlice
		chunked := arrow.NewChunked(col.DataType(), []arrow.Array{col})
		data := chunkedToSlice(chunked)
		chunked.Release()

		colName := rec.Schema().Field(i).Name
		if !supportedArrowType(col.DataType()) {
			dataTable.SetErr("parquet", "Stream", unsupportedColumnMsg, colName, col.DataType())
		}
		dataTable.AppendCols(newColumn(data, colName))
	}
	return dataTable
}

// dataTableToArrowTable converts dt into an Arrow table. ctx is checked once
// per column, so a caller that gives up part-way through a wide table releases
// the columns already built instead of leaving them behind, and returns ctx's
// error.
func dataTableToArrowTable(ctx context.Context, dt insyra.IDataTable) (arrow.Table, error) {
	mem := memory.DefaultAllocator
	numRows, numCols := dt.Size()

	fields := make([]arrow.Field, numCols)
	columns := make([]arrow.Column, numCols)

	for i := range numCols {
		if err := ctx.Err(); err != nil {
			// The columns built so far are never handed to a table, so they are
			// released here; the ones not reached yet were never built.
			for j := range i {
				columns[j].Release()
			}
			return nil, err
		}
		col := dt.GetColByNumber(i)
		colName := dt.GetColNameByNumber(i)
		data := col.Data()

		// Infer type from data
		arrowType := inferArrowType(data)
		fields[i] = arrow.Field{Name: colName, Type: arrowType, Nullable: true}

		builder := array.NewBuilder(mem, arrowType)

		for _, v := range data {
			if v == nil {
				builder.AppendNull()
				continue
			}
			appendValue(builder, v)
		}

		arr := builder.NewArray()

		chunked := arrow.NewChunked(arrowType, []arrow.Array{arr})
		columns[i] = *arrow.NewColumn(fields[i], chunked)

		// Release temporary objects
		arr.Release()
		chunked.Release()
		builder.Release()
	}

	schema := arrow.NewSchema(fields, nil)
	table := array.NewTable(schema, columns, int64(numRows))

	for i := range columns {
		columns[i].Release()
	}

	return table, nil
}

// columnKinds is which kinds of value a column holds, kept apart from the type
// it decides so that a caller with several batches of the same column can feed
// all of them in and read one answer, which is what ApplyCCL needs when a
// column's type is settled from a file rather than from one batch.
type columnKinds struct {
	hasInt, hasFloat, hasString, hasBool, hasTime, hasBytes, hasOther bool

	// lossyInt64 says a value has been added that an int64 cannot hold without
	// loss: a float that is not a whole number, one outside int64's range, or a
	// number of a type that is not one. The flags above cannot say that on their
	// own, because hasFloat is just as true of 1.5 as of 2, which is what lets
	// ApplyCCL keep an int64 column written into with whole numbers and widen it
	// for anything else.
	lossyInt64 bool

	// lossyFloat64 says an integer has been added that a float64 cannot hold
	// exactly: one of magnitude above 2^53, whatever Go type it arrived as. A
	// float64 holds every integer up to there and skips some past it, so one
	// written into a float64 column comes back as a neighbour of itself, a number
	// nobody wrote. A float needs no flag: it is a float64 already.
	lossyFloat64 bool

	// lossyNarrow says, for each type narrower than int64 and float64 that a column
	// the file had can keep, that a value has been added which that type cannot
	// hold: a whole number outside an integer type's range, a number with a
	// fraction for an integer type, a float that does not survive the trip through
	// float32, a time that is not midnight UTC for a date, and any value of another
	// kind. Like lossyInt64 it does not change the type Write gives the column; it
	// is what lets ApplyCCL keep the file's own type.
	lossyNarrow narrowTypes
}

// add records one batch of a column's values. A value it does not know is
// counted as hasOther, which is what makes the column's answer String, the one
// type every value can be written as.
func (k *columnKinds) add(data []any) {
	for _, v := range data {
		if v == nil {
			continue
		}
		if exceedsInt64Loss(v) {
			k.lossyInt64 = true
		}
		if exceedsFloat64Loss(v) {
			k.lossyFloat64 = true
		}
		if k.lossyNarrow != narrowEvery {
			k.lossyNarrow |= narrowLossOf(v, k.lossyNarrow)
		}
		switch v.(type) {
		case int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
			k.hasInt = true
		case float64, float32:
			k.hasFloat = true
		case string:
			k.hasString = true
		case bool:
			k.hasBool = true
		case time.Time:
			k.hasTime = true
		case []byte:
			k.hasBytes = true
		default:
			k.hasOther = true
		}
	}
}

// arrowType is the Arrow type the kinds answer, under the same rules
// inferArrowType applies to one batch.
func (k *columnKinds) arrowType() arrow.DataType {
	numeric := k.hasInt || k.hasFloat
	switch {
	// A column of nothing but byte slices round-trips as binary. Mixed with
	// anything else it falls through to String, which conv.ToString can
	// represent for every value.
	case k.hasBytes && !k.hasString && !k.hasOther && !numeric && !k.hasBool && !k.hasTime:
		return arrow.BinaryTypes.Binary
	case k.hasString || k.hasOther || k.hasBytes:
		return arrow.BinaryTypes.String
	case k.hasBool && !numeric && !k.hasTime:
		return arrow.FixedWidthTypes.Boolean
	case k.hasTime && !numeric && !k.hasBool:
		return arrow.FixedWidthTypes.Timestamp_ns
	case k.hasFloat && !k.hasBool && !k.hasTime:
		return arrow.PrimitiveTypes.Float64
	case k.hasInt && !k.hasBool && !k.hasTime:
		return arrow.PrimitiveTypes.Int64
	default:
		// mixed incompatible kinds (bool+numeric, time+numeric, ...) or all nil
		return arrow.BinaryTypes.String
	}
}

// inferArrowType scans ALL values in a column (not just the first non-nil one)
// so a mixed column does not panic or silently truncate on Write:
//   - a column mixing ints and floats is promoted to Float64 (no truncation);
//   - any string / unknown value, or an incompatible mix (e.g. bool+number,
//     time+number), falls back to String, which conv.ToString can represent for
//     every value.
func inferArrowType(data []any) arrow.DataType {
	var kinds columnKinds
	kinds.add(data)
	return kinds.arrowType()
}

func appendValue(b array.Builder, v any) {
	switch builder := b.(type) {
	case *array.Int64Builder:
		builder.Append(int64(conv.ParseInt(v)))
	case *array.Float64Builder:
		builder.Append(conv.ParseF64(v))
	case *array.StringBuilder:
		builder.Append(conv.ToString(v))
	case *array.BinaryBuilder:
		if b, ok := v.([]byte); ok {
			builder.Append(b)
		} else {
			builder.Append([]byte(conv.ToString(v)))
		}
	case *array.BooleanBuilder:
		builder.Append(conv.ParseBool(v))
	case *array.TimestampBuilder:
		if t, ok := v.(time.Time); ok {
			builder.Append(arrow.Timestamp(t.UnixNano()))
		} else {
			builder.AppendNull()
		}
	default:
		builder.AppendNull()
	}
}
