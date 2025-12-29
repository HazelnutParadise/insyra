package parquet

import (
	"context"
	"fmt"
	"os"

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
		defer f.Close()

		r, err := file.NewParquetReader(f)
		if err != nil {
			errChan <- err
			return
		}
		defer r.Close()

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
		if rr.Err() != nil {
			errChan <- rr.Err()
		}
	}()

	return recChan, errChan
}

func chunkedToSlice(chunked *arrow.Chunked) any {
	if chunked.Len() == 0 {
		return nil
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
