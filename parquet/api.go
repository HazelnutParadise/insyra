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

// ReadOptions: The options for reading Parquet files
type ReadOptions struct {
	Columns   []string // empty=all
	RowGroups []int    // empty=all
}

// ReadColumnOptions: Only for ReadColumn (to avoid putting individual requirements into ReadOptions)
type ReadColumnOptions struct {
	RowGroups []int // empty=all
	MaxValues int64 // 0=no limit; if exceeded, return error to avoid RAM explosion
}

type FileInfo struct {
	NumRows      int64
	NumRowGroups int
	Version      string
	CreatedBy    string
	Metadata     map[string]string
	Columns      []ColumnInfo
	RowGroups    []RowGroupInfo
}

type ColumnInfo struct {
	Name         string
	PhysicalType string
	LogicalType  string
	Repetition   string
}

type RowGroupInfo struct {
	NumRows             int64
	TotalByteSize       int64
	TotalCompressedSize int64
}

// Inspect：只讀 metadata（schema / rows / row groups），不讀實際資料
func Inspect(path string) (FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return FileInfo{}, err
	}
	defer f.Close()

	r, err := file.NewParquetReader(f)
	if err != nil {
		return FileInfo{}, err
	}
	defer r.Close()

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

// Read：一次把「options 指定的範圍」讀完（可選欄、可選包）
// 回傳 insyra.DataTable
func Read(ctx context.Context, path string, opt ReadOptions) (*insyra.DataTable, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r, err := file.NewParquetReader(f)
	if err != nil {
		return nil, err
	}
	defer r.Close()

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
		dataTable.AppendCols(insyra.NewDataList(data).SetName(col.Name()))
	}

	return dataTable, nil
}

// Stream：串流讀（可選欄、可選包），一批一批吐出 arrow.Record
func Stream(ctx context.Context, path string, opt ReadOptions, batchSize int) (<-chan arrow.Record, <-chan error) {
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

// ReadColumn：一次讀完整欄（可選包）
// 回傳值型別是實際 slice：[]int64 / []float64 / []string / []bool …（用 any 承接）
func ReadColumn(ctx context.Context, path string, column string, opt ReadColumnOptions) (*insyra.DataList, error) {
	table, err := Read(ctx, path, ReadOptions{Columns: []string{column}, RowGroups: opt.RowGroups})
	if err != nil {
		return nil, err
	}
	return table.GetCol("A"), nil
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
