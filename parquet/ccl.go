package parquet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
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
				if err := pqCtx.SetRowIndex(rowIdx); err != nil {
					return nil, fmt.Errorf("failed to set row index %d: %w", rowIdx, err)
				}
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
			// Assignment to existing column.
			// The parser encodes a named target ['x'] as "'x'" (quoted) and a
			// column-index target A/B/... as the bare letter. Resolve it to the
			// actual column name used as the resultCols key; otherwise ['x'] = ...
			// would write to key "'x'" and leave the real column untouched.
			resolvedTarget, ok := resolveAssignTarget(target, colNames)
			if !ok {
				return nil, assignTargetError(target, colNames)
			}
			expr := ccl.GetExpressionNode(node)

			// Check if expression depends on row
			if ccl.IsRowDependent(expr) {
				// Evaluate per row
				updatedCol := make([]any, numRows)
				for rowIdx := 0; rowIdx < numRows; rowIdx++ {
					if err := pqCtx.SetRowIndex(rowIdx); err != nil {
						return nil, fmt.Errorf("failed to set row index %d: %w", rowIdx, err)
					}
					val, err := ccl.Evaluate(expr, pqCtx)
					if err != nil {
						return nil, fmt.Errorf("error evaluating assignment to '%s' at row %d: %w", target, rowIdx, err)
					}
					updatedCol[rowIdx] = val
				}
				resultCols[resolvedTarget] = updatedCol
			} else {
				// Constant expression - evaluate once
				if err := pqCtx.SetRowIndex(0); err != nil {
					return nil, fmt.Errorf("failed to set row index to 0: %w", err)
				}
				val, err := ccl.Evaluate(expr, pqCtx)
				if err != nil {
					return nil, fmt.Errorf("error evaluating assignment to '%s': %w", target, err)
				}
				updatedCol := make([]any, numRows)
				for i := range updatedCol {
					updatedCol[i] = val
				}
				resultCols[resolvedTarget] = updatedCol
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

// cclBatchSize is how many rows FilterWithCCL and ApplyCCL read and evaluate
// at a time. It is not a setting: each batch is evaluated on its own, so an
// expression that reads beyond the current row, such as AVG(A), the row index
// # or A.0, sees only its batch, and a different size would change the rows
// FilterWithCCL keeps and the values ApplyCCL writes.
const cclBatchSize = 1000

// FilterWithCCL applies a CCL filter expression to a parquet file and returns filtered results.
// The filter expression should evaluate to boolean for each row.
//
// Example: FilterWithCCL(ctx, "input.parquet", "(A > 100) && (B == 'active')")
//
// Returns a new DataTable containing only rows that satisfy the filter condition.
//
//	Will not modify the original parquet file.
func FilterWithCCL(ctx context.Context, path string, filterExpr string) (*insyra.DataTable, error) {
	// Compile CCL expression once
	compiledExpr, err := ccl.CompileExpression(filterExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to compile CCL expression: %w", err)
	}

	var colNames []string
	// One slice per column, grown across every batch. Appending into
	// result.GetColByNumber(i) per batch threw away everything past the first
	// batch, because that method returns a copy of the column: a 2500-row file
	// filtered on a condition every row satisfies came back with 1000 rows.
	var kept [][]any
	firstBatch := true

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

			// Get column names from first batch
			if firstBatch {
				for i := 0; i < int(rec.NumCols()); i++ {
					colNames = append(colNames, rec.Schema().Field(i).Name)
				}
				kept = make([][]any, len(colNames))
				firstBatch = false
			}

			// Create context for this batch
			pqCtx := newParquetContext(rec, colNames)

			for rowIdx := 0; rowIdx < int(rec.NumRows()); rowIdx++ {
				if err := pqCtx.SetRowIndex(rowIdx); err != nil {
					rec.Release()
					return nil, fmt.Errorf("failed to set row index %d: %w", rowIdx, err)
				}

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
					// Add this row to the filtered results
					for colIdx := 0; colIdx < len(kept); colIdx++ {
						cellVal, _ := pqCtx.GetCell(colIdx, rowIdx)
						kept[colIdx] = append(kept[colIdx], cellVal)
					}
				}
			}

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
// The file is read and written batch by batch; see cclBatchSize for what that
// means for an expression. It is written back with the codec each column had
// and row groups as large as the original's largest one, and a column the
// script adds takes the first column's codec. One opts replaces both, as Write
// uses it. The new file goes to a temporary file of its own and replaces path
// only when it is complete, so a failure leaves the original as it was; an
// input with no rows leaves it untouched too.
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

	err = utils.WriteFileAtomically(path, func(w io.Writer) error {
		var writer *pqarrow.FileWriter
		var colNames []string
		firstBatch := true

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
			}
		}
	})
	if errors.Is(err, errNothingToWrite) {
		return nil
	}
	return err
}
