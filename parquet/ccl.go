package parquet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/ccl"
	"github.com/HazelnutParadise/insyra/internal/utils"
	"github.com/apache/arrow/go/v17/arrow"
	"github.com/apache/arrow/go/v17/arrow/array"
	"github.com/apache/arrow/go/v17/arrow/memory"
	"github.com/apache/arrow/go/v17/parquet"
	"github.com/apache/arrow/go/v17/parquet/compress"
	"github.com/apache/arrow/go/v17/parquet/file"
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

	// Where this batch's first row sits in the file
	offset int

	// written holds the values a statement before this one wrote, by column
	// position: a column it replaced, or one a NEW statement created past the
	// record's own columns. A column with no entry is read from record.
	written map[int][]any
}

func newParquetContext(record arrow.Record, colNames []string, offset int) *parquetContext {
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
		offset:     offset,
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

	for i := range c.colNames {
		c.currentRow[i] = c.cell(i, c.rowIndex)
	}
}

// cell is the value at rowIndex of the column at colIndex, which is what a
// statement before this one wrote when there is one, and the record's own value
// otherwise. It covers a column past the record's columns, which only a NEW
// statement creates and which therefore always has an entry in written.
func (c *parquetContext) cell(colIndex, rowIndex int) any {
	if data, ok := c.written[colIndex]; ok {
		if rowIndex < len(data) {
			return data[rowIndex]
		}
		return nil
	}
	if colIndex >= int(c.record.NumCols()) {
		return nil
	}
	col := c.record.Column(colIndex)
	if col.IsNull(rowIndex) {
		return nil
	}
	return getVal(col, rowIndex)
}

// addColumn registers a column a NEW statement created, so the statements
// after it read it the way they read a column the file had.
func (c *parquetContext) addColumn(name string, data []any) {
	index := len(c.colNames)
	// Clip, so the append always gives a new array: colNames is the caller's
	// slice, and writing into its spare capacity would change what it holds.
	c.colNames = append(slices.Clip(c.colNames), name)
	c.colNameMap[name] = index
	c.currentRow = append(c.currentRow, nil)
	c.setColumn(index, data)
}

// setColumn records the values an assignment wrote to the column at index, so
// the statements after it read those values instead of the file's.
func (c *parquetContext) setColumn(index int, data []any) {
	if c.written == nil {
		c.written = make(map[int][]any)
	}
	c.written[index] = data
	if c.record != nil && c.rowIndex < int(c.record.NumRows()) {
		c.updateCurrentRow()
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

// GlobalRowIndex is the current row's position in the file, which is what #
// means; the batch's other methods work on its own rows.
func (c *parquetContext) GlobalRowIndex() int { return c.offset + c.rowIndex }

func (c *parquetContext) GetCurrentRow() any {
	return c.currentRow
}

func (c *parquetContext) GetCell(colIndex, rowIndex int) (any, error) {
	if c.record == nil {
		return nil, fmt.Errorf("no record available")
	}
	if colIndex < 0 || colIndex >= c.GetColCount() {
		return nil, fmt.Errorf("column index %d out of range", colIndex)
	}
	if rowIndex < 0 || rowIndex >= int(c.record.NumRows()) {
		return nil, fmt.Errorf("row index %d out of range", rowIndex)
	}

	return c.cell(colIndex, rowIndex), nil
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

	row := make([]any, c.GetColCount())
	for i := range row {
		row[i] = c.cell(i, rowIndex)
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
	// A column a statement before this one created is past the record's own.
	return max(len(c.colNames), int(c.record.NumCols()))
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
	if index < 0 || index >= c.GetColCount() {
		return nil, fmt.Errorf("column index %d out of range", index)
	}

	// A copy, so the caller cannot change what the next statement reads.
	if data, ok := c.written[index]; ok {
		return slices.Clone(data), nil
	}

	col := c.record.Column(index)
	result := make([]any, col.Len())
	for i := range result {
		result[i] = c.cell(index, i)
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

	// As many columns as the context reports, not as many as the record holds:
	// a column a statement before this one created is past the record's own,
	// and the other accessors here read it as well.
	for i := 0; i < c.GetColCount(); i++ {
		colData, err := c.GetColData(i)
		if err != nil {
			return nil, err
		}
		allData = append(allData, colData...)
	}

	return allData, nil
}

// rowOf returns the value a row takes from val, an expression's value. A value
// that does not depend on the row and is a slice as long as the table is a
// column, as it is on a loaded table, so the row takes its own element;
// anything else is the row's value as it is.
func rowOf(val any, rowInvariant bool, totalRows, row int) any {
	if !rowInvariant || val == nil {
		return val
	}
	rv := reflect.ValueOf(val)
	if rv.Kind() != reflect.Slice || rv.Len() != totalRows {
		return val
	}
	return rv.Index(row).Interface()
}

// resolveAssignTarget maps a CCL assignment target to an existing column name.
// A named target is encoded by the parser as "'name'" (surrounded by single
// quotes); a bare target is a column letter (A, B, ..., AA) and nothing else,
// the same rule the DataTable CCL path follows, so one script addresses the
// same column whichever backend runs it.
func resolveAssignTarget(target string, colNames []string) (string, bool) {
	if len(target) >= 2 && strings.HasPrefix(target, "'") && strings.HasSuffix(target, "'") {
		name := target[1 : len(target)-1]
		for _, c := range colNames {
			if c == name {
				return name, true
			}
		}
		return "", false
	}
	if idx, ok := insyra.ParseColIndex(target); ok && idx >= 0 && idx < len(colNames) {
		return colNames[idx], true
	}
	return "", false
}

// assignTargetError explains a target that resolved to no column, offering the
// ['name'] form when the file does have a column of that name.
func assignTargetError(target string, colNames []string) error {
	names := make(map[string]int, len(colNames))
	for index, name := range colNames {
		names[name] = index
	}
	idx, ok := insyra.ParseColIndex(target)
	if !ok {
		return ccl.NotAnIndexError(target, names)
	}
	letters, _ := insyra.CalcColIndex(idx)
	return ccl.PastLastColumnError(target, letters, len(colNames), names)
}

// applyStatements runs nodes, in order, over the batch pqCtx holds. Each
// statement reads what the ones before it wrote, through pqCtx. It returns the
// batch's columns as the statements leave them and their names in order.
func applyStatements(pqCtx *parquetContext, colNames []string, nodes []ccl.CCLNode, totalRows int) (map[string][]any, []string, error) {
	numRows := pqCtx.GetRowCount()

	// Build column name map
	colNameMap := make(map[string]int)
	for i, name := range colNames {
		colNameMap[name] = i
	}

	// Prepare result columns - start with copies of existing columns
	resultCols := make(map[string][]any, len(colNames)+len(nodes))
	for i, colName := range colNames {
		colData, err := pqCtx.GetColData(i)
		if err != nil {
			return nil, nil, err
		}
		resultCols[colName] = colData
	}

	// Process each CCL statement
	for _, node := range nodes {
		// Check if it's a new column creation
		if newColName, expr, isNew := ccl.GetNewColInfo(node); isNew {
			// Create new column. An expression that does not vary from row to
			// row is computed once, and a value as long as the file is spread
			// over the rows the way a loaded table spreads it.
			rowInvariant := !ccl.IsRowDependent(expr)
			newColData := make([]any, numRows)
			for rowIdx := 0; rowIdx < numRows; rowIdx++ {
				if err := pqCtx.SetRowIndex(rowIdx); err != nil {
					return nil, nil, fmt.Errorf("failed to set row index %d: %w", rowIdx, err)
				}
				val, err := ccl.Evaluate(expr, pqCtx)
				if err != nil {
					return nil, nil, fmt.Errorf("error evaluating NEW column '%s' at row %d: %w", newColName, pqCtx.GlobalRowIndex(), err)
				}
				newColData[rowIdx] = rowOf(val, rowInvariant, totalRows, pqCtx.GlobalRowIndex())
			}
			resultCols[newColName] = newColData
			colNames = append(colNames, newColName)
			colNameMap[newColName] = len(colNames) - 1
			// The next statement reads this column through the context, which
			// so far only knows the columns the file had.
			pqCtx.addColumn(newColName, newColData)
			continue
		}

		target, isAssignment := ccl.GetAssignmentTarget(node)
		if !isAssignment {
			// A statement that is neither a NEW nor an assignment writes nothing,
			// the way ExecuteCCL leaves it alone, so there is nothing to do here
			// and nothing for the caller to build a column from.
			continue
		}

		// Assignment to existing column.
		// The parser encodes a named target ['x'] as "'x'" (quoted) and a
		// column-index target A/B/... as the bare letter. Resolve it to the
		// actual column name used as the resultCols key; otherwise ['x'] = ...
		// would write to key "'x'" and leave the real column untouched.
		resolvedTarget, ok := resolveAssignTarget(target, colNames)
		if !ok {
			return nil, nil, assignTargetError(target, colNames)
		}
		expr := ccl.GetExpressionNode(node)

		// Check if expression depends on row
		if ccl.IsRowDependent(expr) {
			// Evaluate per row
			updatedCol := make([]any, numRows)
			for rowIdx := 0; rowIdx < numRows; rowIdx++ {
				if err := pqCtx.SetRowIndex(rowIdx); err != nil {
					return nil, nil, fmt.Errorf("failed to set row index %d: %w", rowIdx, err)
				}
				val, err := ccl.Evaluate(expr, pqCtx)
				if err != nil {
					return nil, nil, fmt.Errorf("error evaluating assignment to '%s' at row %d: %w", target, pqCtx.GlobalRowIndex(), err)
				}
				updatedCol[rowIdx] = val
			}
			resultCols[resolvedTarget] = updatedCol
			// The next statement reads this column through the context,
			// which so far only knows what the file and the NEW statements
			// hold.
			pqCtx.setColumn(colNameMap[resolvedTarget], updatedCol)
			continue
		}

		// Constant expression - evaluate once. A value as long as the file is
		// spread over the rows the way a loaded table spreads it.
		if err := pqCtx.SetRowIndex(0); err != nil {
			return nil, nil, fmt.Errorf("failed to set row index to 0: %w", err)
		}
		val, err := ccl.Evaluate(expr, pqCtx)
		if err != nil {
			return nil, nil, fmt.Errorf("error evaluating assignment to '%s': %w", target, err)
		}
		offset := pqCtx.offset
		updatedCol := make([]any, numRows)
		for i := range updatedCol {
			updatedCol[i] = rowOf(val, true, totalRows, offset+i)
		}
		resultCols[resolvedTarget] = updatedCol
		pqCtx.setColumn(colNameMap[resolvedTarget], updatedCol)
	}

	return resultCols, colNames, nil
}

// applyBatchCCL applies the CCL statements to one arrow.Record batch and builds
// the record the batch leaves behind, for ApplyCCL to write. The statements
// themselves run in applyStatements, which a caller that does not write the
// batch out uses on its own.
func applyBatchCCL(rec arrow.Record, pqCtx *parquetContext, colNames []string, compiledNodes []ccl.CCLNode, totalRows int) (arrow.Record, error) {
	resultCols, appliedNames, err := applyStatements(pqCtx, colNames, compiledNodes, totalRows)
	if err != nil {
		return nil, err
	}

	// Convert result columns to arrow.Record
	return buildArrowRecord(resultCols, appliedNames, rec.Schema())
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

// cclBatchSize is how many rows FilterWithCCL and ApplyCCL read at a time.
// Every part of an expression that reads beyond the current row is computed
// over the whole file before the rows are evaluated (ccl.ResolveWholeTable),
// so the size changes how much is held in memory, not the answer.
const cclBatchSize = 1000

// cclFileInfo returns the row count and the column names of the file at path,
// as the batches FilterWithCCL and ApplyCCL read will name them.
func cclFileInfo(path string) (totalRows int, colNames []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			totalRows, colNames, err = 0, nil, unreadableFile(path, r)
		}
	}()

	f, err := os.Open(path)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = f.Close() }()

	r, err := file.NewParquetReader(f)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = r.Close() }()

	fr, err := pqarrow.NewFileReader(r, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
	if err != nil {
		return 0, nil, err
	}

	schema, err := fr.Schema()
	if err != nil {
		return 0, nil, err
	}
	colNames = make([]string, len(schema.Fields()))
	for i, field := range schema.Fields() {
		colNames[i] = field.Name
	}
	return int(r.NumRows()), colNames, nil
}

// forEachRecord reads the file at path batch by batch and calls fn with every
// record, in order. The reader stops when it returns, and fn must not keep rec
// after it returns.
func forEachRecord(ctx context.Context, path string, fn func(rec arrow.Record) error) error {
	// The reader stops when this call returns, so an early return cannot leave
	// it blocked on a batch nobody reads, holding the file open.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	recChan, errChan := streamAsArrowRecord(ctx, path, ReadOptions{}, cclBatchSize)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errChan:
			if err != nil {
				return err
			}
			return nil
		case rec, ok := <-recChan:
			if !ok {
				// The producer closes errChan before recChan, so by now errChan
				// is closed too and this receive cannot block. Read it rather
				// than letting the select above choose between two ready cases,
				// which would drop an error reported after the last batch.
				if err := <-errChan; err != nil {
					return err
				}
				return nil
			}
			err := fn(rec)
			rec.Release()
			if err != nil {
				return err
			}
		}
	}
}

// cclBatches reads the file at path for ccl.ResolveWholeTable: each call is one
// pass, and every batch knows where its rows start in the file.
func cclBatches(ctx context.Context, path string, colNames []string) ccl.Batches {
	return func(yield func(ccl.GlobalRowContext) error) error {
		offset := 0
		return forEachRecord(ctx, path, func(rec arrow.Record) error {
			// The whole rows the resolver keeps out of a batch are copied out
			// of it while yield runs, so rec can be released right after.
			if err := yield(newParquetContext(rec, colNames, offset)); err != nil {
				return err
			}
			offset += int(rec.NumRows())
			return nil
		})
	}
}

// appliedBatches reads the file at path for ccl.ResolveWholeTable as the
// statements before the one being resolved leave it: each batch has had prior
// applied to it, so a statement reading a column an earlier one created or
// replaced sees that column. The context handed to the resolver is the very one
// the row-by-row pass evaluates those statements through, so a value a statement
// writes is the one every later statement reads; a record rebuilt from the file
// would round a written value through the column's own parquet type instead, and
// a column the script creates would take its type from the first batch only.
func appliedBatches(ctx context.Context, path string, colNames []string, prior []ccl.CCLNode, totalRows int) ccl.Batches {
	return func(yield func(ccl.GlobalRowContext) error) error {
		offset := 0
		return forEachRecord(ctx, path, func(rec arrow.Record) error {
			// applyStatements appends the columns a NEW statement creates to
			// the names it is given, so it gets a copy of its own.
			pqCtx := newParquetContext(rec, slices.Clone(colNames), offset)
			if len(prior) > 0 {
				if _, _, err := applyStatements(pqCtx, slices.Clone(colNames), prior, totalRows); err != nil {
					return err
				}
			}

			// The whole rows the resolver keeps out of a batch are copied out
			// of it while yield runs, so rec can be released right after.
			if err := yield(pqCtx); err != nil {
				return err
			}
			offset += int(rec.NumRows())
			return nil
		})
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
//
// The file is read cclBatchSize rows at a time, and the answer is the one the
// same expression gives on the loaded table: an aggregate is computed over the
// whole file, # is the row's position in the file, and a fixed row such as A.0
// is that row of the file. Computing those parts reads the file again before
// the rows are evaluated. MEDIAN, an aggregate registered with
// RegisterAggregateFunction, a sequence function such as LAG or CUMSUM, and a
// row reference computed from the current row such as A.(# - 1) are refused
// with an error before any row is evaluated.
func FilterWithCCL(ctx context.Context, path string, filterExpr string) (*insyra.DataTable, error) {
	// Compile CCL expression once
	compiledExpr, err := ccl.CompileExpression(filterExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to compile CCL expression: %w", err)
	}

	// The whole-file parts are settled before any row is kept, so a filter is
	// either answered or refused before the file is read into a result. The
	// file's shape is read first: an expression that needs nothing beyond the
	// current row costs no extra pass, and a row or column the expression names
	// is checked against the file rather than against a batch.
	totalRows, colNames, err := cclFileInfo(path)
	if err != nil {
		return nil, err
	}
	resolved, err := ccl.ResolveWholeTable(compiledExpr, totalRows, colNames, cclBatches(ctx, path, colNames))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate CCL expression: %w", err)
	}

	// One slice per column, grown across every batch. Appending into
	// result.GetColByNumber(i) per batch threw away everything past the first
	// batch, because that method returns a copy of the column: a 2500-row file
	// filtered on a condition every row satisfies came back with 1000 rows.
	kept := make([][]any, len(colNames))

	// The reader stops when this call returns, so an early return cannot leave
	// it blocked on a batch nobody reads, holding the file open.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	recChan, errChan := streamAsArrowRecord(ctx, path, ReadOptions{}, cclBatchSize)

	// build assembles the result once, on whichever path ends the stream, so an
	// empty file and a filter that matched nothing produce the same shape.
	build := func() *insyra.DataTable {
		result := insyra.NewDataTable()
		for i, name := range colNames {
			dl := insyra.NewDataList()
			dl.SetName(name)
			if len(kept[i]) > 0 {
				dl.Append(kept[i]...)
			}
			result.AppendCols(dl)
		}
		return result
	}

	// Where the batch being evaluated starts in the file, which is what # is
	// counted from.
	offset := 0

	// A value that does not vary from row to row and is a slice as long as the
	// file is spread over the rows, the way a loaded table spreads it: the
	// filter keeps the rows their own element keeps.
	rowInvariant := !ccl.IsRowDependent(resolved)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errChan:
			if err != nil {
				return nil, err
			}
			// Stream finished, return result
			return build(), nil
		case rec, ok := <-recChan:
			if !ok {
				// The producer closes errChan before recChan, so by now errChan
				// is closed too and this receive cannot block. Read it rather
				// than letting the select above choose between two ready cases,
				// which would drop an error reported after the last batch.
				if err := <-errChan; err != nil {
					return nil, err
				}
				return build(), nil
			}

			// Create context for this batch
			pqCtx := newParquetContext(rec, colNames, offset)

			for rowIdx := 0; rowIdx < int(rec.NumRows()); rowIdx++ {
				if err := pqCtx.SetRowIndex(rowIdx); err != nil {
					rec.Release()
					return nil, fmt.Errorf("failed to set row index %d: %w", rowIdx, err)
				}

				// Evaluate filter expression
				val, err := ccl.Evaluate(resolved, pqCtx)
				if err != nil {
					rec.Release()
					return nil, fmt.Errorf("error evaluating CCL at row %d: %w", offset+rowIdx, err)
				}
				val = rowOf(val, rowInvariant, totalRows, offset+rowIdx)

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
					// Add this row to the filtered results
					for colIdx := 0; colIdx < len(kept); colIdx++ {
						cellVal, _ := pqCtx.GetCell(colIdx, rowIdx)
						kept[colIdx] = append(kept[colIdx], cellVal)
					}
				}
			}

			offset += int(rec.NumRows())
			rec.Release()
		}
	}
}

// cclOutputLayout is how ApplyCCL writes the file back: the codec of each
// column the source has, by its path in the schema, the codec for a column the
// script adds, and the most rows one row group holds.
type cclOutputLayout struct {
	codecs       map[string]compress.Compression
	defaultCodec compress.Compression
	rowGroupSize int64
}

// errNothingToWrite tells ApplyCCL that the input had no rows, so the original
// is kept rather than replaced by an empty file.
var errNothingToWrite = errors.New("parquet: nothing to write")

// sourceLayout reads the layout of the file at path: the codec of each of its
// columns from the first row group, the first column's codec for a column the
// script adds, and the largest row group's row count. A file with no row groups
// has nothing to keep, so it gets the defaults Write uses.
func sourceLayout(path string) (layout cclOutputLayout, err error) {
	defer func() {
		if r := recover(); r != nil {
			layout, err = cclOutputLayout{}, unreadableFile(path, r)
		}
	}()

	f, err := os.Open(path)
	if err != nil {
		return cclOutputLayout{}, err
	}
	defer func() { _ = f.Close() }()

	r, err := file.NewParquetReader(f)
	if err != nil {
		return cclOutputLayout{}, err
	}
	defer func() { _ = r.Close() }()

	if r.NumRowGroups() == 0 {
		return cclOutputLayout{
			codecs:       map[string]compress.Compression{},
			defaultCodec: compress.Codecs.Uncompressed,
			rowGroupSize: defaultRowGroupSize,
		}, nil
	}

	md := r.MetaData()
	layout = cclOutputLayout{
		codecs:       make(map[string]compress.Compression),
		rowGroupSize: 1,
	}
	for i := 0; i < r.NumRowGroups(); i++ {
		layout.rowGroupSize = max(layout.rowGroupSize, md.RowGroup(i).NumRows())
	}
	first := md.RowGroup(0)
	for i := 0; i < first.NumColumns(); i++ {
		chunk, err := first.ColumnChunk(i)
		if err != nil {
			return cclOutputLayout{}, err
		}
		layout.codecs[chunk.PathInSchema().String()] = chunk.Compression()
		if i == 0 {
			layout.defaultCodec = chunk.Compression()
		}
	}
	return layout, nil
}

// ApplyCCL applies CCL statements to a Parquet file and writes the result back
// to the same path. cclScript can hold several statements separated by
// semicolons or new lines, for example NEW('C') = ['A'] + ['B'].
//
// The file is read and written batch by batch, and every statement gives what
// it gives on the loaded table, the way FilterWithCCL does; each statement is
// computed against the file as the statements before it leave it, so it can
// read a column an earlier one created. A statement FilterWithCCL would refuse
// is refused here before anything is written. It is written back with the
// codec each column had and row groups as large as the original's largest
// one, and a column the script adds takes the first column's codec. One opts
// replaces both, as Write uses it. The new file goes to a temporary file of
// its own and replaces path only when it is complete, so a failure leaves the
// original as it was; an input with no rows leaves it untouched too.
func ApplyCCL(ctx context.Context, path string, cclScript string, opts ...WriteOptions) error {
	// The layout is settled before anything is read or written, so settings
	// Write would refuse, or a file whose layout cannot be read, leave the file
	// as it was.
	var layout cclOutputLayout
	if len(opts) > 0 {
		codec, size, err := resolveWriteOptions(opts)
		if err != nil {
			return err
		}
		layout = cclOutputLayout{codecs: nil, defaultCodec: codec, rowGroupSize: size}
	} else {
		var err error
		layout, err = sourceLayout(path)
		if err != nil {
			return err
		}
	}

	// Compile CCL statements once
	compiledNodes, err := ccl.CompileMultiline(cclScript)
	if err != nil {
		return fmt.Errorf("failed to compile CCL script: %w", err)
	}

	// Every statement is resolved against the file as the statements before it
	// leave it, one statement at a time, because a statement can read a column
	// an earlier one created. This happens before anything is written, so a
	// statement that cannot be computed leaves the original alone.
	totalRows, colNames, err := cclFileInfo(path)
	if err != nil {
		return err
	}

	resolved := make([]ccl.CCLNode, 0, len(compiledNodes))
	names := slices.Clone(colNames)
	for _, node := range compiledNodes {
		if _, _, isNew := ccl.GetNewColInfo(node); !isNew {
			if _, isAssignment := ccl.GetAssignmentTarget(node); !isAssignment {
				// A statement that is neither a NEW nor an assignment writes
				// nothing, the way ExecuteCCL leaves it alone, so there is no
				// result to resolve: resolving it would refuse a part the
				// loaded table never evaluates.
				resolved = append(resolved, node)
				continue
			}
		}
		r, err := ccl.ResolveWholeTable(node, totalRows, names, appliedBatches(ctx, path, colNames, slices.Clone(resolved), totalRows))
		if err != nil {
			return fmt.Errorf("failed to apply CCL: %w", err)
		}
		resolved = append(resolved, r)
		if newName, _, isNew := ccl.GetNewColInfo(node); isNew {
			names = append(names, newName)
		}
	}

	err = utils.WriteFileAtomically(path, func(w io.Writer) error {
		var writer *pqarrow.FileWriter
		var batchColNames []string
		firstBatch := true
		offset := 0

		// Stream through the input file. It is safe to replace it afterwards
		// because the output goes to a temporary file of its own.
		// The reader stops when this call returns, so an early return cannot
		// leave it blocked on a batch nobody reads, holding the file open.
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		recChan, errChan := streamAsArrowRecord(ctx, path, ReadOptions{}, cclBatchSize)

		// Every way out below that is not a clean end returns without closing
		// writer: the temporary file is thrown away, so it needs no footer.
		finish := func() error {
			if writer == nil {
				// No record batch at all (the input is empty): keep the original
				// rather than replace it with an empty file.
				return errNothingToWrite
			}
			// writer wraps w in writerOnly, so closing it writes the footer and
			// leaves the file for WriteFileAtomically to close and rename.
			if err := writer.Close(); err != nil {
				return fmt.Errorf("failed to close writer: %w", err)
			}
			return nil
		}

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-errChan:
				if err != nil {
					return err
				}
				// errChan closed without an error = the stream ended well.
				return finish()

			case rec, ok := <-recChan:
				if !ok {
					// recChan closed = every record has been consumed (it is
					// unbuffered). The producer closes errChan before recChan, and
					// when both are ready the select above picks one at random, so
					// a read error reported after the last batch could be lost and
					// finish() would then replace the original with a truncated
					// file. Read errChan here first (it is closed by now, so this
					// cannot block): an error means no finishing.
					if err := <-errChan; err != nil {
						return err
					}
					return finish()
				}

				// Get column names from first batch
				if firstBatch {
					for i := 0; i < int(rec.NumCols()); i++ {
						batchColNames = append(batchColNames, rec.Schema().Field(i).Name)
					}
					firstBatch = false
				}

				// Create context for this batch, carrying where the batch
				// starts in the file so # is the file's row and an error names
				// the row the user can look up.
				pqCtx := newParquetContext(rec, batchColNames, offset)

				// Apply CCL transformations to this batch. applyBatchCCL
				// appends the columns a NEW statement creates to the names it
				// is given, so it gets a copy of its own.
				transformedRec, err := applyBatchCCL(rec, pqCtx, slices.Clone(batchColNames), resolved, totalRows)
				if err != nil {
					rec.Release()
					return fmt.Errorf("failed to apply CCL: %w", err)
				}

				// Initialize writer with schema from first transformed batch
				if writer == nil {
					props := []parquet.WriterProperty{
						parquet.WithCreatedBy(fmt.Sprintf("go-insyra v%s", insyra.Version)),
						parquet.WithCompression(layout.defaultCodec),
						parquet.WithMaxRowGroupLength(layout.rowGroupSize),
					}
					for name, codec := range layout.codecs {
						props = append(props, parquet.WithCompressionFor(name, codec))
					}
					writer, err = pqarrow.NewFileWriter(
						transformedRec.Schema(),
						writerOnly{w},
						parquet.NewWriterProperties(props...),
						pqarrow.DefaultWriterProps(),
					)
					if err != nil {
						rec.Release()
						transformedRec.Release()
						return fmt.Errorf("failed to create parquet writer: %w", err)
					}
				}

				// Write transformed batch. Buffered, so a row group keeps filling
				// across batches until it holds layout.rowGroupSize rows.
				err = writer.WriteBuffered(transformedRec)
				rec.Release()
				transformedRec.Release()

				if err != nil {
					return fmt.Errorf("failed to write batch: %w", err)
				}

				offset += int(rec.NumRows())
			}
		}
	})
	if errors.Is(err, errNothingToWrite) {
		return nil
	}
	return err
}
