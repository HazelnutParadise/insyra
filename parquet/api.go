package parquet

import (
	"context"
	"fmt"
	"os"

	"github.com/HazelnutParadise/insyra"
	"github.com/apache/arrow/go/v17/arrow"
	"github.com/apache/arrow/go/v17/arrow/memory"
	"github.com/apache/arrow/go/v17/parquet"
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

// Inspect: inspect parquet file metadata
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

// Write: write insyra.DataTable to parquet file
func Write(dt insyra.IDataTable, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	arrowTable, err := dataTableToArrowTable(dt)
	if err != nil {
		return err
	}
	defer arrowTable.Release()

	createdBy := fmt.Sprintf("go-insyra v%s", insyra.Version)

	writer, err := pqarrow.NewFileWriter(arrowTable.Schema(), f, parquet.NewWriterProperties(parquet.WithCreatedBy(createdBy)), pqarrow.DefaultWriterProps())
	if err != nil {
		return err
	}
	defer writer.Close()

	return writer.WriteTable(arrowTable, 1024*1024) // chunk size
}

// Read: read parquet file into insyra.DataTable at once
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

// Stream: streaming read parquet file, returning insyra.DataTable batches
func Stream(ctx context.Context, path string, opt ReadOptions, batchSize int) (<-chan *insyra.DataTable, <-chan error) {
	dtChan := make(chan *insyra.DataTable)
	errChan := make(chan error, 1)

	go func() {
		defer close(dtChan)
		defer close(errChan)

		recChan, internalErrChan := streamAsArrowRecord(ctx, path, opt, batchSize)

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
					return // Stream closed
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

// ReadColumn: read a single column's data from Parquet file.
// Returns insyra.DataList.
func ReadColumn(ctx context.Context, path string, column string, opt ReadColumnOptions) (*insyra.DataList, error) {
	table, err := Read(ctx, path, ReadOptions{Columns: []string{column}, RowGroups: opt.RowGroups})
	if err != nil {
		return nil, err
	}
	return table.GetCol("A"), nil
}
