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
	"github.com/apache/arrow/go/v17/arrow"
	"github.com/apache/arrow/go/v17/arrow/array"
	"github.com/apache/arrow/go/v17/arrow/memory"
	"github.com/apache/arrow/go/v17/parquet/file"
	"github.com/apache/arrow/go/v17/parquet/pqarrow"
)

// streamAsArrowRecord：串流讀（可選欄、可選包），一批一批吐出 arrow.Record
func streamAsArrowRecord(ctx context.Context, path string, opt ReadOptions, batchSize int) (<-chan arrow.Record, <-chan error) {
	recChan := make(chan arrow.Record)
	errChan := make(chan error, 1)

	go func() {
		defer close(recChan)
		defer close(errChan)

		f, err := os.Open(path)
		if err != nil {
			errChan <- err
			return
		}
		defer func() {
			if err := f.Close(); err != nil {
				insyra.LogWarning("parquet", "close", "failed to close file %s: %v", path, err)
			}
		}()

		r, err := file.NewParquetReader(f)
		if err != nil {
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

		rr, err := fr.GetRecordReader(ctx, colIndices, rowGroups)
		if err != nil {
			errChan <- err
			return
		}
		defer rr.Release()

		for rr.Next() {
			rec := rr.Record()
			rec.Retain()
			select {
			case <-ctx.Done():
				rec.Release()
				errChan <- ctx.Err()
				return
			case recChan <- rec:
			}
		}
		if rr.Err() != nil && !errors.Is(rr.Err(), io.EOF) {
			errChan <- rr.Err()
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

func getVal(arr arrow.Array, i int) any {
	switch a := arr.(type) {
	case *array.Int64:
		return a.Value(i)
	case *array.Int32:
		return a.Value(i)
	case *array.Float64:
		return a.Value(i)
	case *array.Float32:
		return a.Value(i)
	case *array.String:
		return a.Value(i)
	case *array.Boolean:
		return a.Value(i)
	case *array.Timestamp:
		return a.Value(i).ToTime(a.DataType().(*arrow.TimestampType).Unit)
	default:
		return arr.String()
	}
}

func recordToDataTable(rec arrow.Record) *insyra.DataTable {
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
		dataTable.AppendCols(insyra.NewDataList(data).SetName(colName))
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

// inferArrowType scans ALL values in a column (not just the first non-nil one)
// so a mixed column does not panic or silently truncate on Write:
//   - a column mixing ints and floats is promoted to Float64 (no truncation);
//   - any string / unknown value, or an incompatible mix (e.g. bool+number,
//     time+number), falls back to String, which conv.ToString can represent for
//     every value.
func inferArrowType(data []any) arrow.DataType {
	var hasInt, hasFloat, hasString, hasBool, hasTime, hasOther bool
	for _, v := range data {
		if v == nil {
			continue
		}
		switch v.(type) {
		case int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
			hasInt = true
		case float64, float32:
			hasFloat = true
		case string:
			hasString = true
		case bool:
			hasBool = true
		case time.Time:
			hasTime = true
		default:
			hasOther = true
		}
	}
	numeric := hasInt || hasFloat
	switch {
	case hasString || hasOther:
		return arrow.BinaryTypes.String
	case hasBool && !numeric && !hasTime:
		return arrow.FixedWidthTypes.Boolean
	case hasTime && !numeric && !hasBool:
		return arrow.FixedWidthTypes.Timestamp_ns
	case hasFloat && !hasBool && !hasTime:
		return arrow.PrimitiveTypes.Float64
	case hasInt && !hasBool && !hasTime:
		return arrow.PrimitiveTypes.Int64
	default:
		// mixed incompatible kinds (bool+numeric, time+numeric, ...) or all nil
		return arrow.BinaryTypes.String
	}
}

func appendValue(b array.Builder, v any) {
	switch builder := b.(type) {
	case *array.Int64Builder:
		builder.Append(int64(conv.ParseInt(v)))
	case *array.Float64Builder:
		builder.Append(conv.ParseF64(v))
	case *array.StringBuilder:
		builder.Append(conv.ToString(v))
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
