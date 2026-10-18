package parquet

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
	"unicode/utf8"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
	"github.com/TimLai666/go-decimal/decimal"
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/flight"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
)

// pqarrow v18 cannot decode a stored Duration schema even though its Parquet
// physical type is INT64. Present those columns to pqarrow as INT64; the public
// table representation has always been the raw int64 count.
func normalizeStoredDurationSchema(r *file.Reader) error {
	pairs := r.MetaData().FileMetaData.KeyValueMetadata
	for _, pair := range pairs {
		if pair.Key != "ARROW:schema" || pair.Value == nil {
			continue
		}
		serialized, err := base64.StdEncoding.DecodeString(*pair.Value)
		if err != nil {
			return fmt.Errorf("parquet: decode stored Arrow schema: %w", err)
		}
		stored, err := flight.DeserializeSchema(serialized, memory.DefaultAllocator)
		if err != nil {
			return fmt.Errorf("parquet: decode stored Arrow schema: %w", err)
		}
		fields := stored.Fields()
		changed := false
		for i := range fields {
			if fields[i].Type.ID() == arrow.DURATION {
				column := r.MetaData().Schema.ColumnIndexByName(fields[i].Name)
				if column >= 0 && r.MetaData().Schema.Column(column).PhysicalType() == parquet.Types.Int64 {
					fields[i].Type = arrow.PrimitiveTypes.Int64
					changed = true
				}
			}
		}
		if changed {
			metadata := stored.Metadata()
			normalized := arrow.NewSchema(fields, &metadata)
			*pair.Value = base64.StdEncoding.EncodeToString(flight.SerializeSchema(normalized, memory.DefaultAllocator))
		}
		return nil
	}
	return nil
}

func readerProperties() *parquet.ReaderProperties {
	props := parquet.NewReaderProperties(memory.DefaultAllocator)
	props.MaxCompressedPageSize = 1 << 40
	props.MaxUncompressedPageSize = 1 << 40
	return props
}

// streamAsArrowRecord：串流讀（可選欄、可選包），一批一批吐出 arrow.RecordBatch
func streamAsArrowRecord(ctx context.Context, path string, opt ReadOptions, batchSize int) (<-chan arrow.RecordBatch, <-chan error) {
	recChan := make(chan arrow.RecordBatch)
	errChan := make(chan error, 1)

	go func() {
		defer close(recChan)
		defer close(errChan)
		defer func() {
			if r := recover(); r != nil {
				select {
				case errChan <- unreadableFile(path, r):
				default:
				}
			}
		}()

		f, err := os.Open(path)
		if err != nil {
			errChan <- err
			return
		}
		defer func() {
			if err := f.Close(); err != nil {
				// Reader.Close() may already close the underlying file, the same
				// way Read and Inspect allow for. Without this guard every
				// successful Stream, FilterWithCCL and ApplyCCL logged a warning.
				if errors.Is(err, os.ErrClosed) {
					return
				}
				insyra.LogWarning("parquet", "close", "failed to close file %s: %v", path, err)
			}
		}()

		r, err := file.NewParquetReader(f, file.WithReadProps(readerProperties()))
		if err != nil {
			errChan <- err
			return
		}
		if err := normalizeStoredDurationSchema(r); err != nil {
			errChan <- err
			return
		}
		defer func() {
			if err := r.Close(); err != nil {
				insyra.LogWarning("parquet", "close", "failed to close reader for %s: %v", path, err)
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
		if len(colIndices) == 0 {
			colIndices = make([]int, r.MetaData().Schema.NumColumns())
			for i := range colIndices {
				colIndices[i] = i
			}
		}
		groups := opt.RowGroups
		if len(groups) == 0 {
			groups = make([]int, r.NumRowGroups())
			for i := range groups {
				groups[i] = i
			}
		}
		if len(colIndices) == 0 {
			colIndices = make([]int, r.MetaData().Schema.NumColumns())
			for i := range colIndices {
				colIndices[i] = i
			}
		}
		counts := make([]int64, len(groups))
		var want int64
		countable := len(groups) > 0
		for i, rg := range groups {
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
		if rr.Err() != nil && !errors.Is(rr.Err(), io.EOF) {
			errChan <- readerFailed(ctx, path, rr.Err(), f, colIndices, groups, counts, countable)
			return
		}
		if countable && read != want {
			errChan <- damagedFile(path, read, want, shortRowGroupsFrom(ctx, f, colIndices, groups, counts))
		}
	}()

	return recChan, errChan
}

// streamArrowRecordsFrom streams records from a random-access Parquet source.
func streamArrowRecordsFrom(ctx context.Context, src parquet.ReaderAtSeeker, label string, opt ReadOptions, batchSize int, done func()) (<-chan arrow.RecordBatch, <-chan error) {
	recChan := make(chan arrow.RecordBatch)
	errChan := make(chan error, 1)
	go func() {
		defer close(recChan)
		defer close(errChan)
		defer func() {
			if r := recover(); r != nil {
				select {
				case errChan <- unreadableFile(label, r):
				default:
				}
			}
		}()
		if done != nil {
			defer done()
		}
		r, err := file.NewParquetReader(src, file.WithReadProps(readerProperties()))
		if err != nil {
			errChan <- err
			return
		}
		if err := normalizeStoredDurationSchema(r); err != nil {
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
			for _, name := range opt.Columns {
				idx := r.MetaData().Schema.ColumnIndexByName(name)
				if idx < 0 {
					errChan <- fmt.Errorf("column %s not found", name)
					return
				}
				colIndices = append(colIndices, idx)
			}
		}
		groups := opt.RowGroups
		if len(groups) == 0 {
			groups = make([]int, r.NumRowGroups())
			for i := range groups {
				groups[i] = i
			}
		}
		if len(colIndices) == 0 {
			colIndices = make([]int, r.MetaData().Schema.NumColumns())
			for i := range colIndices {
				colIndices[i] = i
			}
		}
		counts := make([]int64, len(groups))
		var want int64
		countable := len(groups) > 0
		for i, rg := range groups {
			if rg < 0 || rg >= r.NumRowGroups() {
				countable = false
				break
			}
			counts[i] = r.MetaData().RowGroup(rg).NumRows()
			want += counts[i]
		}
		rr, err := fr.GetRecordReader(ctx, colIndices, groups)
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
		if rr.Err() != nil && !errors.Is(rr.Err(), io.EOF) {
			errChan <- readerFailed(ctx, label, rr.Err(), src, colIndices, groups, counts, countable)
			return
		}
		if countable && read != want {
			errChan <- damagedFile(label, read, want, shortRowGroupsFrom(ctx, src, colIndices, groups, counts))
		}
	}()
	return recChan, errChan
}

func unreadableFile(label string, r any) error {
	return fmt.Errorf("parquet: %s is not a readable Parquet file: %v", label, r)
}

func damagedFile(label string, read, want int64, groups []int) error {
	return fmt.Errorf("parquet: %s is damaged: its metadata holds %d rows in the row groups read, but %d were read; row groups %v did not read in full", label, want, read, groups)
}

func readerFailed(ctx context.Context, label string, err error, src parquet.ReaderAtSeeker, columns, groups []int, counts []int64, countable bool) error {
	if ctx.Err() != nil || !countable {
		return err
	}
	bad := shortRowGroupsFrom(ctx, src, columns, groups, counts)
	if len(bad) == 0 {
		return err
	}
	var want int64
	for _, n := range counts {
		want += n
	}
	return fmt.Errorf("parquet: %s is damaged: its metadata holds %d rows in the row groups read; row groups %v did not read in full: %w", label, want, bad, err)
}

func shortRowGroupsFrom(ctx context.Context, src parquet.ReaderAtSeeker, columns, groups []int, counts []int64) []int {
	var bad []int
	for i, rg := range groups {
		pr, err := file.NewParquetReader(readerAtSeekerNoClose{ReaderAtSeeker: src}, file.WithReadProps(readerProperties()))
		if err != nil {
			bad = append(bad, rg)
			continue
		}
		if err := normalizeStoredDurationSchema(pr); err != nil {
			_ = pr.Close()
			bad = append(bad, rg)
			continue
		}
		fr, err := pqarrow.NewFileReader(pr, pqarrow.ArrowReadProperties{Parallel: true}, memory.DefaultAllocator)
		if err != nil {
			_ = pr.Close()
			bad = append(bad, rg)
			continue
		}
		t, err := fr.ReadRowGroups(ctx, columns, []int{rg})
		if err != nil {
			_ = pr.Close()
			bad = append(bad, rg)
			continue
		}
		rows := t.NumRows()
		t.Release()
		_ = pr.Close()
		if rows != counts[i] {
			bad = append(bad, rg)
			continue
		}
	}
	return bad
}

type readerAtSeekerNoClose struct{ parquet.ReaderAtSeeker }

func (readerAtSeekerNoClose) Close() error { return nil }

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
		arrow.TIMESTAMP, arrow.DATE32, arrow.DATE64, arrow.DURATION,
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
	case *array.Duration:
		return int64(a.Value(i))
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

func dataTableToArrowTable(dt insyra.IDataTable) (arrow.Table, error) {
	mem := memory.DefaultAllocator
	numRows, numCols := dt.Size()

	fields := make([]arrow.Field, numCols)
	columns := make([]arrow.Column, numCols)

	for i := range numCols {
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
	case k.hasBytes && !k.hasString && !k.hasOther && !k.hasInt && !k.hasFloat && !k.hasBool && !k.hasTime:
		// Only byte slices: the column is Binary, so the bytes come back as bytes.
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
		if raw, ok := v.([]byte); ok && utf8.Valid(raw) {
			// conv.ToString would write a byte slice as "[65 45 48 49]".
			builder.Append(string(raw))
		} else {
			builder.Append(conv.ToString(v))
		}
	case *array.BinaryBuilder:
		if raw, ok := v.([]byte); ok {
			builder.Append(raw)
		} else {
			builder.AppendNull()
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
