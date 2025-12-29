package parquet

import (
	"context"
	"fmt"
	"os"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/ccl"
	"github.com/apache/arrow/go/v17/arrow"
	"github.com/apache/arrow/go/v17/arrow/array"
	"github.com/apache/arrow/go/v17/arrow/memory"
	"github.com/apache/arrow/go/v17/parquet/pqarrow"
)

// parquetContext implements ccl.Context for direct parquet file operations
type parquetContext struct {
	// Current record batch
	record arrow.Record

	// Column metadata
	colNames   []string
	colNameMap map[string]int

	// Current row info
	rowIndex   int
	currentRow []any

	// For accessing entire columns (used in aggregate functions)
	allRecords []arrow.Record // Cache all records for full data access
	totalRows  int64
}

func newParquetContext(record arrow.Record, colNames []string) *parquetContext {
	colNameMap := make(map[string]int)
	for i, name := range colNames {
		colNameMap[name] = i
	}

	ctx := &parquetContext{
		record:     record,
		colNames:   colNames,
		colNameMap: colNameMap,
		rowIndex:   0,
		currentRow: make([]any, len(colNames)),
	}

	// Initialize current row
	if record != nil && record.NumRows() > 0 {
		ctx.updateCurrentRow()
	}

	return ctx
}

func (c *parquetContext) updateCurrentRow() {
	if c.record == nil || c.rowIndex >= int(c.record.NumRows()) {
		return
	}

	for i, col := range c.record.Columns() {
		if c.rowIndex < col.Len() {
			if col.IsNull(c.rowIndex) {
				c.currentRow[i] = nil
			} else {
				c.currentRow[i] = getVal(col, c.rowIndex)
			}
		} else {
			c.currentRow[i] = nil
		}
	}
}

func (c *parquetContext) GetCol(index int) any {
	if index >= len(c.currentRow) {
		return nil
	}
	return c.currentRow[index]
}

func (c *parquetContext) GetColByName(name string) (any, error) {
	idx, ok := c.colNameMap[name]
	if !ok {
		return nil, fmt.Errorf("column name '%s' not found", name)
	}
	if idx >= len(c.currentRow) {
		return nil, fmt.Errorf("column ['%s'] (index %d) out of range", name, idx)
	}
	return c.currentRow[idx], nil
}

func (c *parquetContext) GetRowIndex() int {
	return c.rowIndex
}

func (c *parquetContext) GetCurrentRow() any {
	return c.currentRow
}

func (c *parquetContext) GetCell(colIndex, rowIndex int) (any, error) {
	if c.record == nil {
		return nil, fmt.Errorf("no record available")
	}
	if colIndex < 0 || colIndex >= int(c.record.NumCols()) {
		return nil, fmt.Errorf("column index %d out of range", colIndex)
	}
	if rowIndex < 0 || rowIndex >= int(c.record.NumRows()) {
		return nil, fmt.Errorf("row index %d out of range", rowIndex)
	}

	col := c.record.Column(colIndex)
	if col.IsNull(rowIndex) {
		return nil, nil
	}
	return getVal(col, rowIndex), nil
}

func (c *parquetContext) GetCellByName(colName string, rowIndex int) (any, error) {
	idx, ok := c.colNameMap[colName]
	if !ok {
		return nil, fmt.Errorf("column name '%s' not found", colName)
	}
	return c.GetCell(idx, rowIndex)
}

func (c *parquetContext) GetRowAt(rowIndex int) (any, error) {
	if c.record == nil {
		return nil, fmt.Errorf("no record available")
	}
	if rowIndex < 0 || rowIndex >= int(c.record.NumRows()) {
		return nil, fmt.Errorf("row index %d out of range", rowIndex)
	}

	row := make([]any, c.record.NumCols())
	for i, col := range c.record.Columns() {
		if col.IsNull(rowIndex) {
			row[i] = nil
		} else {
			row[i] = getVal(col, rowIndex)
		}
	}
	return row, nil
}

func (c *parquetContext) GetRowIndexByName(rowName string) (int, error) {
	return -1, fmt.Errorf("row names are not supported in parquet context")
}

func (c *parquetContext) GetColIndexByName(colName string) (int, error) {
	idx, ok := c.colNameMap[colName]
	if !ok {
		return -1, fmt.Errorf("column name '%s' not found", colName)
	}
	return idx, nil
}

func (c *parquetContext) GetColCount() int {
	if c.record == nil {
		return 0
	}
	return int(c.record.NumCols())
}

func (c *parquetContext) GetRowCount() int {
	if c.record == nil {
		return 0
	}
	return int(c.record.NumRows())
}

func (c *parquetContext) SetRowIndex(index int) error {
	if c.record == nil {
		return fmt.Errorf("no record available")
	}
	if index < 0 || index >= int(c.record.NumRows()) {
		return fmt.Errorf("row index %d out of range", index)
	}
	c.rowIndex = index
	c.updateCurrentRow()
	return nil
}

func (c *parquetContext) GetColData(index int) ([]any, error) {
	if c.record == nil {
		return nil, fmt.Errorf("no record available")
	}
	if index < 0 || index >= int(c.record.NumCols()) {
		return nil, fmt.Errorf("column index %d out of range", index)
	}

	col := c.record.Column(index)
	result := make([]any, col.Len())
	for i := 0; i < col.Len(); i++ {
		if col.IsNull(i) {
			result[i] = nil
		} else {
			result[i] = getVal(col, i)
		}
	}
	return result, nil
}

func (c *parquetContext) GetColDataByName(name string) ([]any, error) {
	idx, ok := c.colNameMap[name]
	if !ok {
		return nil, fmt.Errorf("column name '%s' not found", name)
	}
	return c.GetColData(idx)
}

func (c *parquetContext) GetAllData() ([]any, error) {
	if c.record == nil {
		return nil, fmt.Errorf("no record available")
	}

	var allData []any
	totalSize := int(c.record.NumCols() * c.record.NumRows())
	allData = make([]any, 0, totalSize)

	for i := 0; i < int(c.record.NumCols()); i++ {
		colData, err := c.GetColData(i)
		if err != nil {
			return nil, err
		}
		allData = append(allData, colData...)
	}

	return allData, nil
}

// applyBatchCCL applies CCL transformations to a single arrow.Record batch
func applyBatchCCL(rec arrow.Record, pqCtx *parquetContext, colNames []string, compiledNodes []ccl.CCLNode) (arrow.Record, error) {
	numRows := int(rec.NumRows())

	// Build column name map
	colNameMap := make(map[string]int)
	for i, name := range colNames {
		colNameMap[name] = i
	}

	// Prepare result columns - start with copies of existing columns
	resultCols := make(map[string][]any)
	for i, colName := range colNames {
		colData, err := pqCtx.GetColData(i)
		if err != nil {
			return nil, err
		}
		resultCols[colName] = colData
	}

	// Process each CCL statement
	for _, node := range compiledNodes {
		// Check if it's a new column creation
		if newColName, expr, isNew := ccl.GetNewColInfo(node); isNew {
			// Create new column
			newColData := make([]any, numRows)
			for rowIdx := 0; rowIdx < numRows; rowIdx++ {
				pqCtx.SetRowIndex(rowIdx)
				val, err := ccl.Evaluate(expr, pqCtx)
				if err != nil {
					return nil, fmt.Errorf("error evaluating NEW column '%s' at row %d: %w", newColName, rowIdx, err)
				}
				newColData[rowIdx] = val
			}
			resultCols[newColName] = newColData
			colNames = append(colNames, newColName)
			colNameMap[newColName] = len(colNames) - 1

		} else if target, isAssignment := ccl.GetAssignmentTarget(node); isAssignment {
			// Assignment to existing column
			expr := ccl.GetExpressionNode(node)

			// Check if expression depends on row
			if ccl.IsRowDependent(expr) {
				// Evaluate per row
				updatedCol := make([]any, numRows)
				for rowIdx := 0; rowIdx < numRows; rowIdx++ {
					pqCtx.SetRowIndex(rowIdx)
					val, err := ccl.Evaluate(expr, pqCtx)
					if err != nil {
						return nil, fmt.Errorf("error evaluating assignment to '%s' at row %d: %w", target, rowIdx, err)
					}
					updatedCol[rowIdx] = val
				}
				resultCols[target] = updatedCol
			} else {
				// Constant expression - evaluate once
				pqCtx.SetRowIndex(0)
				val, err := ccl.Evaluate(expr, pqCtx)
				if err != nil {
					return nil, fmt.Errorf("error evaluating assignment to '%s': %w", target, err)
				}
				updatedCol := make([]any, numRows)
				for i := range updatedCol {
					updatedCol[i] = val
				}
				resultCols[target] = updatedCol
			}
		}
	}

	// Convert result columns to arrow.Record
	return buildArrowRecord(resultCols, colNames, rec.Schema())
}

// buildArrowRecord constructs an arrow.Record from column data
func buildArrowRecord(cols map[string][]any, colNames []string, originalSchema *arrow.Schema) (arrow.Record, error) {
	mem := memory.DefaultAllocator

	// Build schema for result
	fields := make([]arrow.Field, 0, len(colNames))
	for _, colName := range colNames {
		// Try to find field in original schema
		fieldIdx := originalSchema.FieldIndices(colName)
		if len(fieldIdx) > 0 {
			fields = append(fields, originalSchema.Field(fieldIdx[0]))
		} else {
			// New column - infer type from data
			colData := cols[colName]
			fields = append(fields, arrow.Field{
				Name: colName,
				Type: inferArrowType(colData),
			})
		}
	}

	schema := arrow.NewSchema(fields, nil)

	// Build arrays
	arrays := make([]arrow.Array, len(colNames))
	for i, colName := range colNames {
		colData := cols[colName]
		arr, err := buildArrowArray(mem, colData, fields[i].Type)
		if err != nil {
			return nil, fmt.Errorf("failed to build array for column '%s': %w", colName, err)
		}
		arrays[i] = arr
	}

	return array.NewRecord(schema, arrays, int64(len(cols[colNames[0]]))), nil
}

// buildArrowArray constructs an arrow.Array from Go slice
func buildArrowArray(mem memory.Allocator, data []any, dtype arrow.DataType) (arrow.Array, error) {
	switch dtype.ID() {
	case arrow.INT64:
		builder := array.NewInt64Builder(mem)
		defer builder.Release()
		for _, v := range data {
			if v == nil {
				builder.AppendNull()
			} else {
				builder.Append(int64(conv.ParseInt(v)))
			}
		}
		return builder.NewArray(), nil

	case arrow.FLOAT64:
		builder := array.NewFloat64Builder(mem)
		defer builder.Release()
		for _, v := range data {
			if v == nil {
				builder.AppendNull()
			} else {
				builder.Append(conv.ParseF64(v))
			}
		}
		return builder.NewArray(), nil

	case arrow.BOOL:
		builder := array.NewBooleanBuilder(mem)
		defer builder.Release()
		for _, v := range data {
			if v == nil {
				builder.AppendNull()
			} else {
				builder.Append(conv.ParseBool(v))
			}
		}
		return builder.NewArray(), nil

	case arrow.STRING:
		builder := array.NewStringBuilder(mem)
		defer builder.Release()
		for _, v := range data {
			if v == nil {
				builder.AppendNull()
			} else {
				builder.Append(conv.ToString(v))
			}
		}
		return builder.NewArray(), nil

	default:
		// Fallback to string
		builder := array.NewStringBuilder(mem)
		defer builder.Release()
		for _, v := range data {
			if v == nil {
				builder.AppendNull()
			} else {
				builder.Append(conv.ToString(v))
			}
		}
		return builder.NewArray(), nil
	}
}

// FilterWithCCL applies a CCL filter expression to a parquet file and returns filtered results.
// The filter expression should evaluate to boolean for each row.
//
// Example: FilterWithCCL(ctx, "input.parquet", "(A > 100) && (B == 'active')")
//
// Returns a new DataTable containing only rows that satisfy the filter condition.
//
//	Will not modify the original parquet file.
func FilterWithCCL(ctx context.Context, path string, filterExpr string) (*insyra.DataTable, error) {
	batchSize := 1000

	// Compile CCL expression once
	compiledExpr, err := ccl.CompileExpression(filterExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to compile CCL expression: %w", err)
	}

	result := insyra.NewDataTable()
	var colNames []string
	firstBatch := true

	recChan, errChan := streamAsArrowRecord(ctx, path, ReadOptions{}, batchSize)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errChan:
			if err != nil {
				return nil, err
			}
			// Stream finished, return result
			return result, nil
		case rec, ok := <-recChan:
			if !ok {
				return result, nil
			}

			// Get column names from first batch
			if firstBatch {
				for i := 0; i < int(rec.NumCols()); i++ {
					colNames = append(colNames, rec.Schema().Field(i).Name)
				}
				firstBatch = false
			}

			// Create context for this batch
			pqCtx := newParquetContext(rec, colNames)

			// Filter rows in this batch
			filteredRows := make([][]any, int(rec.NumCols()))
			for i := range filteredRows {
				filteredRows[i] = make([]any, 0)
			}

			for rowIdx := 0; rowIdx < int(rec.NumRows()); rowIdx++ {
				pqCtx.SetRowIndex(rowIdx)

				// Evaluate filter expression
				val, err := ccl.Evaluate(compiledExpr, pqCtx)
				if err != nil {
					rec.Release()
					return nil, fmt.Errorf("error evaluating CCL at row %d: %w", rowIdx, err)
				}

				// Check if row passes filter
				passes := false
				switch v := val.(type) {
				case bool:
					passes = v
				case float64:
					passes = v != 0
				case int:
					passes = v != 0
				case int64:
					passes = v != 0
				default:
					passes = val != nil
				}

				if passes {
					// Add this row to filtered results
					for colIdx := 0; colIdx < int(rec.NumCols()); colIdx++ {
						cellVal, _ := pqCtx.GetCell(colIdx, rowIdx)
						filteredRows[colIdx] = append(filteredRows[colIdx], cellVal)
					}
				}
			}

			rec.Release()

			// Append filtered rows to result
			_, numCols := result.Size()
			if numCols == 0 {
				// First batch: create columns with names
				for i := 0; i < len(filteredRows); i++ {
					dl := insyra.NewDataList()
					dl.SetName(colNames[i])
					if len(filteredRows[i]) > 0 {
						dl.Append(filteredRows[i]...)
					}
					result.AppendCols(dl)
				}
			} else {
				// Subsequent batches: append to existing columns
				for i := 0; i < len(filteredRows); i++ {
					if len(filteredRows[i]) > 0 {
						col := result.GetColByNumber(i)
						if col != nil {
							col.Append(filteredRows[i]...)
						}
					}
				}
			}
		}
	}
}

// ApplyCCL applies CCL expressions directly to parquet file in streaming mode.
// This function operates directly on parquet files without loading into DataTable.
// Processing is done batch by batch to minimize memory usage.
// cclScript can contain multiple statements separated by semicolons.
//
// Example: ApplyCCL(ctx, "input.parquet", "NEW('C') = ['A'] + ['B']; ['D'] = ['D'] * 2", CCLFilterOptions{})
//
//	The input file will be overwritten.
func ApplyCCL(ctx context.Context, path string, cclScript string) error {
	batchSize := 1000

	// Compile CCL statements once
	compiledNodes, err := ccl.CompileMultiline(cclScript)
	if err != nil {
		return fmt.Errorf("failed to compile CCL script: %w", err)
	}

	// Create temporary output file
	tmpPath := path + ".tmp"
	outFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary output file: %w", err)
	}
	defer func() {
		outFile.Close()
		// Clean up temporary file if it still exists (error case)
		os.Remove(tmpPath)
	}()

	var writer *pqarrow.FileWriter
	var resultSchema *arrow.Schema
	var colNames []string
	firstBatch := true

	// Stream through input file (now safe since we write to tmpPath)
	recChan, errChan := streamAsArrowRecord(ctx, path, ReadOptions{}, batchSize)

	for {
		select {
		case <-ctx.Done():
			if writer != nil {
				writer.Close()
			}
			return ctx.Err()
		case err := <-errChan:
			if writer != nil {
				writer.Close()
			}
			if err != nil {
				return err
			}
			// Stream finished successfully
			if writer != nil {
				if err := writer.Close(); err != nil {
					return fmt.Errorf("failed to close writer: %w", err)
				}
			}

			// Close output file before moving
			outFile.Close()

			// Move temporary file to original path
			if err := os.Rename(tmpPath, path); err != nil {
				return fmt.Errorf("failed to replace original file: %w", err)
			}

			return nil

		case rec, ok := <-recChan:
			if !ok {
				// Channel closed
				if writer != nil {
					if err := writer.Close(); err != nil {
						return fmt.Errorf("failed to close writer: %w", err)
					}
				}

				// Close output file before moving
				outFile.Close()

				// Move temporary file to original path
				if err := os.Rename(tmpPath, path); err != nil {
					return fmt.Errorf("failed to replace original file: %w", err)
				}

				return nil
			}

			// Get column names from first batch
			if firstBatch {
				for i := 0; i < int(rec.NumCols()); i++ {
					colNames = append(colNames, rec.Schema().Field(i).Name)
				}
				firstBatch = false
			}

			// Create context for this batch
			pqCtx := newParquetContext(rec, colNames)

			// Apply CCL transformations to this batch
			transformedRec, err := applyBatchCCL(rec, pqCtx, colNames, compiledNodes)
			if err != nil {
				rec.Release()
				if writer != nil {
					writer.Close()
				}
				return fmt.Errorf("failed to apply CCL: %w", err)
			}

			// Initialize writer with schema from first transformed batch
			if writer == nil {
				resultSchema = transformedRec.Schema()
				writer, err = pqarrow.NewFileWriter(
					resultSchema,
					outFile,
					nil,
					pqarrow.DefaultWriterProps(),
				)
				if err != nil {
					rec.Release()
					transformedRec.Release()
					return fmt.Errorf("failed to create parquet writer: %w", err)
				}
			}

			// Write transformed batch
			err = writer.Write(transformedRec)
			rec.Release()
			transformedRec.Release()

			if err != nil {
				writer.Close()
				return fmt.Errorf("failed to write batch: %w", err)
			}
		}
	}
}
