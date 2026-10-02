package parquet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"time"

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

// cclRun is a run of consecutive rows of the file, held as one slice of Go
// values per column; offset is the file position of its first row.
type cclRun struct {
	offset int
	names  []string
	cols   [][]any

	// arrays holds, for each column that still has the values the file gave it,
	// the Arrow array they came from, aligned with cols; nil for a column a
	// statement created or replaced, and for every column of a run that did not
	// come from the file. It may be shorter than names.
	arrays []arrow.Array
}

// rows returns how many rows r holds.
func (r cclRun) rows() int {
	if len(r.cols) == 0 {
		return 0
	}
	return len(r.cols[0])
}

// runFromRecord reads rec into a run starting at offset, a null as nil and
// every other cell through getVal, named names. The values it holds are Go
// values of their own, so a run outlives the record's buffers, and each column
// keeps the array it was read from, retained, so the writer can put a column
// nothing wrote back into the file unchanged.
func runFromRecord(rec arrow.Record, names []string, offset int) cclRun {
	cols := make([][]any, rec.NumCols())
	arrays := make([]arrow.Array, rec.NumCols())
	for i := range cols {
		col := rec.Column(i)
		arrays[i] = col
		col.Retain()
		data := make([]any, col.Len())
		for j := range data {
			if col.IsNull(j) {
				continue
			}
			data[j] = getVal(col, j)
		}
		cols[i] = data
	}
	return cclRun{offset: offset, names: names, cols: cols, arrays: arrays}
}

// parquetContext implements ccl.Context for direct parquet file operations. It
// holds its rows as columns of Go values, so a statement writes into the shape
// the next one reads instead of into an override beside it.
type parquetContext struct {
	// Column metadata, and the values of each column: one entry for a column
	// the file had, one for a column a NEW statement created and one for a
	// column an assignment replaced.
	colNames   []string
	colNameMap map[string]int
	cols       [][]any

	// arrays holds the array of the file each column still has, in step with
	// cols: nil for a column a statement created or replaced, and for a context
	// built from no record at all.
	arrays []arrow.Array

	// How many rows the columns hold
	rows int

	// Current row info
	rowIndex   int
	currentRow []any

	// Where this run's first row sits in the file
	offset int

	// hasData is false for a context built from no record at all, which is what
	// an empty batch is: it reports no record available rather than an empty
	// one.
	hasData bool

	// loaded says which columns hold their values: a column that is not loaded was
	// not read for the expression the context evaluates, and reading it is an
	// error rather than a column of missing values. nil means every column is.
	loaded []bool
}

// newRunContext returns a context over r. The outer slices are copied, so what
// a statement writes does not change the run the caller handed over.
func newRunContext(r cclRun) *parquetContext {
	names := slices.Clone(r.names)
	colNameMap := make(map[string]int, len(names))
	for i, name := range names {
		colNameMap[name] = i
	}
	cols := slices.Clone(r.cols)

	ctx := &parquetContext{
		colNames:   names,
		colNameMap: colNameMap,
		cols:       cols,
		arrays:     runArrays(r),
		rows:       r.rows(),
		offset:     r.offset,
		hasData:    true,
		currentRow: make([]any, len(cols)),
	}
	ctx.updateCurrentRow()
	return ctx
}

// newParquetContext returns a context over record, whose first row sits at
// offset in the file. A nil record gives the context of an empty batch.
func newParquetContext(record arrow.Record, colNames []string, offset int) *parquetContext {
	if record == nil {
		colNameMap := make(map[string]int, len(colNames))
		for i, name := range colNames {
			colNameMap[name] = i
		}
		return &parquetContext{
			colNames:   colNames,
			colNameMap: colNameMap,
			currentRow: make([]any, len(colNames)),
			offset:     offset,
		}
	}
	return newRunContext(runFromRecord(record, colNames, offset))
}

// newWholeColumnsContext returns a context over the whole of the file: its
// columns are the ones in names, cols[i] holds the values of column i, and a
// column whose cols[i] is nil was not read. The context starts at the file's
// first row and holds totalRows rows, so a row written in an expression is the
// row it is read by. Its column count is the file's, so a column letter keeps the
// meaning it has on the whole table.
func newWholeColumnsContext(names []string, cols [][]any, totalRows int) *parquetContext {
	names = slices.Clone(names)
	colNameMap := make(map[string]int, len(names))
	for i, name := range names {
		colNameMap[name] = i
	}
	held := make([][]any, len(names))
	copy(held, cols)
	loaded := make([]bool, len(names))
	for i := range loaded {
		loaded[i] = held[i] != nil
	}

	ctx := &parquetContext{
		colNames:   names,
		colNameMap: colNameMap,
		cols:       held,
		rows:       totalRows,
		hasData:    true,
		currentRow: make([]any, len(names)),
		loaded:     loaded,
	}
	ctx.updateCurrentRow()
	return ctx
}

// notRead returns the error reading column index is, when the context did not
// read that column, and nil when it did.
func (c *parquetContext) notRead(index int) error {
	if c.loaded == nil || index < 0 || index >= len(c.loaded) || c.loaded[index] {
		return nil
	}
	return fmt.Errorf("column %s was not read for this expression", c.colNames[index])
}

// run returns the rows the context holds, with the columns the statements run
// through it so far have written.
func (c *parquetContext) run() cclRun {
	return cclRun{offset: c.offset, names: c.colNames, cols: c.cols, arrays: c.arrays}
}

func (c *parquetContext) updateCurrentRow() {
	if c.rowIndex >= c.rows {
		return
	}

	// A column that was not read holds no values, so its cell is nil here. Reading
	// it by name (GetColByName), through GetCell, GetColData or as part of a row
	// (GetRowAt) is the error notRead gives; GetCol, which reads the current row
	// by position, is the one accessor that has no error to give and returns nil.
	for i := range c.currentRow {
		c.currentRow[i] = nil
		if i < len(c.cols) && c.rowIndex < len(c.cols[i]) {
			c.currentRow[i] = c.cols[i][c.rowIndex]
		}
	}
}

// addColumn registers a column a NEW statement created, so the statements
// after it read it the way they read a column the file had.
func (c *parquetContext) addColumn(name string, data []any) {
	index := len(c.colNames)
	// Clip, so the append always gives a new array: colNames may be a run's
	// own slice, and writing into its spare capacity would change what it holds.
	c.colNames = append(slices.Clip(c.colNames), name)
	c.colNameMap[name] = index
	c.cols = append(slices.Clip(c.cols), data)
	// One entry per name, so the columns after the new one are where they were
	// and the new one has no array of the file's: its values are what it is
	// written from.
	arrays := make([]arrow.Array, len(c.colNames))
	copy(arrays, c.arrays)
	c.arrays = arrays
	if c.loaded != nil {
		// A column a statement wrote holds all of its values.
		c.loaded = append(slices.Clip(c.loaded), true)
	}
	c.currentRow = append(c.currentRow, nil)
	c.updateCurrentRow()
}

// setColumn records the values an assignment wrote to the column at index, so
// the statements after it read those values instead of the file's.
func (c *parquetContext) setColumn(index int, data []any) {
	if index < 0 || index >= len(c.cols) {
		return
	}
	c.cols[index] = data
	if index < len(c.arrays) {
		// The column no longer holds the file's values, so the array of those is
		// not what it is written from.
		c.arrays[index] = nil
	}
	if c.loaded != nil {
		c.loaded[index] = true
	}
	c.updateCurrentRow()
}

// GetCol returns the current row's cell at position index. It has no error to
// return, so a column that was not read (see notRead) and a position past the last
// column both give nil; the accessors that return an error report the first.
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
	if err := c.notRead(idx); err != nil {
		return nil, err
	}
	return c.currentRow[idx], nil
}

func (c *parquetContext) GetRowIndex() int {
	return c.rowIndex
}

// GlobalRowIndex is the current row's position in the file, which is what #
// means; the run's other methods work on its own rows.
func (c *parquetContext) GlobalRowIndex() int { return c.offset + c.rowIndex }

// GetCurrentRow returns a copy of the current row. updateCurrentRow overwrites
// the row buffer in place as the context moves from row to row, so handing out the
// buffer itself would make every cell that @ produced hold the last row, which is
// what the loaded table's context avoids by copying too.
func (c *parquetContext) GetCurrentRow() any {
	return slices.Clone(c.currentRow)
}

func (c *parquetContext) GetCell(colIndex, rowIndex int) (any, error) {
	if !c.hasData {
		return nil, fmt.Errorf("no record available")
	}
	if colIndex < 0 || colIndex >= c.GetColCount() {
		return nil, fmt.Errorf("column index %d out of range", colIndex)
	}
	if err := c.notRead(colIndex); err != nil {
		return nil, err
	}
	if rowIndex < 0 || rowIndex >= c.rows {
		// The loaded table's words, so one script fails with one message
		// whichever backend runs it.
		return nil, fmt.Errorf("row index %d out of range for column %d", rowIndex, colIndex)
	}
	if rowIndex >= len(c.cols[colIndex]) {
		return nil, nil
	}

	return c.cols[colIndex][rowIndex], nil
}

func (c *parquetContext) GetCellByName(colName string, rowIndex int) (any, error) {
	idx, ok := c.colNameMap[colName]
	if !ok {
		return nil, fmt.Errorf("column name '%s' not found", colName)
	}
	return c.GetCell(idx, rowIndex)
}

func (c *parquetContext) GetRowAt(rowIndex int) (any, error) {
	if !c.hasData {
		return nil, fmt.Errorf("no record available")
	}
	if rowIndex < 0 || rowIndex >= c.rows {
		return nil, fmt.Errorf("row index %d out of range", rowIndex)
	}
	// A row is every column, so one that was not read makes the row unreadable.
	for i := range c.loaded {
		if err := c.notRead(i); err != nil {
			return nil, err
		}
	}

	row := make([]any, c.GetColCount())
	for i := range row {
		if i < len(c.cols) && rowIndex < len(c.cols[i]) {
			row[i] = c.cols[i][rowIndex]
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
	if !c.hasData {
		return 0
	}
	return len(c.cols)
}

func (c *parquetContext) GetRowCount() int {
	if !c.hasData {
		return 0
	}
	return c.rows
}

func (c *parquetContext) SetRowIndex(index int) error {
	if !c.hasData {
		return fmt.Errorf("no record available")
	}
	if index < 0 || index >= c.rows {
		return fmt.Errorf("row index %d out of range", index)
	}
	c.rowIndex = index
	c.updateCurrentRow()
	return nil
}

func (c *parquetContext) GetColData(index int) ([]any, error) {
	if !c.hasData {
		return nil, fmt.Errorf("no record available")
	}
	if index < 0 || index >= c.GetColCount() {
		return nil, fmt.Errorf("column index %d out of range", index)
	}
	if err := c.notRead(index); err != nil {
		return nil, err
	}

	// A copy, so the caller cannot change what the next statement reads.
	return slices.Clone(c.cols[index]), nil
}

func (c *parquetContext) GetColDataByName(name string) ([]any, error) {
	idx, ok := c.colNameMap[name]
	if !ok {
		return nil, fmt.Errorf("column name '%s' not found", name)
	}
	return c.GetColData(idx)
}

func (c *parquetContext) GetAllData() ([]any, error) {
	if !c.hasData {
		return nil, fmt.Errorf("no record available")
	}

	var allData []any
	totalSize := len(c.cols) * c.rows
	allData = make([]any, 0, totalSize)

	// As many columns as the context reports, not as many as the file had: a
	// column a statement before this one created is beside them, and the other
	// accessors here read it as well.
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
// ['name'] form when the file does have a column of that name. A named target
// that is not a column is said the way the loaded table says it, because the
// target is the reason the statement fails and the table's message is the one a
// caller already knows.
func assignTargetError(target string, colNames []string) error {
	if len(target) >= 2 && strings.HasPrefix(target, "'") && strings.HasSuffix(target, "'") {
		return fmt.Errorf("assignment target column '%s' does not exist", target[1:len(target)-1])
	}
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

// cancelCheckRows is how many rows applyStatement and wholeColumnValues evaluate
// between two looks at the context: often enough that a statement that costs a
// pass over a column per row stops soon after the context ends, and rarely enough
// that the look costs nothing against evaluating a row.
const cancelCheckRows = 1024

// applyStatement runs one statement over every row pqCtx holds and writes its
// result into pqCtx: a NEW adds a column, an assignment replaces one, and any
// other statement writes nothing, the way ExecuteCCL leaves it alone. The
// columns and the names the statements run through the context are its own, so
// each one sees what the ones before it wrote. A statement evaluated row by row
// looks at ctx every cancelCheckRows rows and returns its error once it has ended.
func applyStatement(ctx context.Context, pqCtx *parquetContext, node ccl.CCLNode, totalRows int) error {
	numRows := pqCtx.GetRowCount()

	// Check if it's a new column creation
	if newColName, expr, isNew := ccl.GetNewColInfo(node); isNew {
		// Create new column. An expression that does not vary from row to
		// row is computed once, and a value as long as the file is spread
		// over the rows the way a loaded table spreads it.
		newColData := make([]any, numRows)
		if !ccl.IsRowDependent(expr) {
			if numRows > 0 {
				// The first row stands for every row: the loaded table computes
				// such an expression once too, and without a row to name in the
				// error, because no row is the cause of it.
				if err := pqCtx.SetRowIndex(0); err != nil {
					return fmt.Errorf("failed to set row index to 0: %w", err)
				}
				val, err := ccl.Evaluate(expr, pqCtx)
				if err != nil {
					return fmt.Errorf("error evaluating NEW column '%s': %w", newColName, err)
				}
				offset := pqCtx.offset
				for i := range newColData {
					newColData[i] = rowOf(val, true, totalRows, offset+i)
				}
			}
		} else {
			for rowIdx := 0; rowIdx < numRows; rowIdx++ {
				if rowIdx%cancelCheckRows == 0 {
					if err := ctx.Err(); err != nil {
						return err
					}
				}
				if err := pqCtx.SetRowIndex(rowIdx); err != nil {
					return fmt.Errorf("failed to set row index %d: %w", rowIdx, err)
				}
				val, err := ccl.Evaluate(expr, pqCtx)
				if err != nil {
					return fmt.Errorf("error evaluating NEW column '%s' at row %d: %w", newColName, pqCtx.GlobalRowIndex(), err)
				}
				newColData[rowIdx] = val
			}
		}
		// The next statement reads this column through the context, which now
		// knows it as one of the file's own.
		pqCtx.addColumn(newColName, newColData)
		return nil
	}

	target, isAssignment := ccl.GetAssignmentTarget(node)
	if !isAssignment {
		// A statement that is neither a NEW nor an assignment writes nothing,
		// the way ExecuteCCL leaves it alone, so there is nothing to do here
		// and nothing for the caller to build a column from.
		return nil
	}

	// Assignment to existing column.
	// The parser encodes a named target ['x'] as "'x'" (quoted) and a
	// column-index target A/B/... as the bare letter. Resolve it to the
	// actual column name of the context; otherwise ['x'] = ... would leave the
	// real column untouched.
	resolvedTarget, ok := resolveAssignTarget(target, pqCtx.colNames)
	if !ok {
		return assignTargetError(target, pqCtx.colNames)
	}
	expr := ccl.GetExpressionNode(node)

	// Check if expression depends on row
	if ccl.IsRowDependent(expr) {
		// Evaluate per row
		updatedCol := make([]any, numRows)
		for rowIdx := 0; rowIdx < numRows; rowIdx++ {
			if rowIdx%cancelCheckRows == 0 {
				if err := ctx.Err(); err != nil {
					return err
				}
			}
			if err := pqCtx.SetRowIndex(rowIdx); err != nil {
				return fmt.Errorf("failed to set row index %d: %w", rowIdx, err)
			}
			val, err := ccl.Evaluate(expr, pqCtx)
			if err != nil {
				return fmt.Errorf("error evaluating assignment to '%s' at row %d: %w", target, pqCtx.GlobalRowIndex(), err)
			}
			updatedCol[rowIdx] = val
		}
		// The next statement reads this column through the context, which now
		// holds what was written instead of what the file had.
		pqCtx.setColumn(pqCtx.colNameMap[resolvedTarget], updatedCol)
		return nil
	}

	// Constant expression - evaluate once. A value as long as the file is
	// spread over the rows the way a loaded table spreads it.
	if err := pqCtx.SetRowIndex(0); err != nil {
		return fmt.Errorf("failed to set row index to 0: %w", err)
	}
	val, err := ccl.Evaluate(expr, pqCtx)
	if err != nil {
		return fmt.Errorf("error evaluating assignment to '%s': %w", target, err)
	}
	offset := pqCtx.offset
	updatedCol := make([]any, numRows)
	for i := range updatedCol {
		updatedCol[i] = rowOf(val, true, totalRows, offset+i)
	}
	pqCtx.setColumn(pqCtx.colNameMap[resolvedTarget], updatedCol)
	return nil
}

// cclStage is one statement of an ApplyCCL script. push takes the next run of
// rows and returns the rows it has finished, in order; flush returns what it
// still holds once the file has ended.
type cclStage interface {
	push(r cclRun) ([]cclRun, error)
	flush() ([]cclRun, error)
}

// statementStage runs a statement that finishes every row at once.
type statementStage struct {
	node      ccl.CCLNode
	totalRows int
}

// push runs the statement over r and hands on the rows it wrote. A run with no
// rows has nothing for a statement to read, so none is evaluated, but the columns
// it hands on are the ones a run with rows would have: a column a NEW statement
// creates is still a column of the run afterwards, an empty one, or the stages
// after this one would see a different set of columns than they do for every
// other run and have nothing to write into.
func (s *statementStage) push(r cclRun) ([]cclRun, error) {
	if r.rows() == 0 {
		cols := fullColumns(r)
		arrays := runArrays(r)
		if newName, _, isNew := ccl.GetNewColInfo(s.node); isNew {
			// Clip, so the append always gives a new array: names may be a run's own
			// slice, and writing into its spare capacity would change what it holds.
			r = cclRun{
				offset: r.offset,
				names:  append(slices.Clip(r.names), newName),
				cols:   append(cols, []any{}),
				arrays: append(arrays, nil),
			}
			return []cclRun{r}, nil
		}
		// An assignment replaces a column that is there already, and any other
		// statement writes nothing.
		return []cclRun{{offset: r.offset, names: r.names, cols: cols, arrays: arrays}}, nil
	}
	runCtx := newRunContext(r)
	// A stage is not handed the caller's context: it works on the one run it was
	// pushed, which is at most a batch of the file's rows plus whatever an earlier
	// sequence stage held back, and eachRun and appliedBatches look at the caller's
	// context between the batches they read. The whole-column pass, which hands
	// applyStatement the whole file in one call, passes its own.
	if err := applyStatement(context.Background(), runCtx, s.node, s.totalRows); err != nil {
		return nil, err
	}
	return []cclRun{runCtx.run()}, nil
}

// flush returns nothing: every row this stage was given is finished.
func (s *statementStage) flush() ([]cclRun, error) {
	return nil, nil
}

// wholeColumnStage stands for a statement that was computed over the whole of the
// columns it reads before the file was read for writing (wholeColumnValues): the
// column it writes is already there, and the stage only puts each run's share of
// it where the statement writes it. A NEW adds the column and an assignment
// replaces the one it names, so the statements after this one read the answers as
// columns of their own, the way they do after any other stage.
type wholeColumnStage struct {
	values []any

	// newName is the column a NEW statement creates, and target the position an
	// assignment replaces, which is -1 for a NEW.
	newName string
	target  int
}

// newWholeColumnStage returns the stage for the statement node, whose column is
// values, over the columns names. An assignment's target is resolved here, so a
// target that names no column is refused before the file is read.
func newWholeColumnStage(node ccl.CCLNode, values []any, names []string) (*wholeColumnStage, error) {
	if newName, _, isNew := ccl.GetNewColInfo(node); isNew {
		return &wholeColumnStage{values: values, newName: newName, target: -1}, nil
	}
	rawTarget, isAssignment := ccl.GetAssignmentTarget(node)
	if !isAssignment {
		return nil, fmt.Errorf("a statement that writes no column has no values to hand on")
	}
	resolved, ok := resolveAssignTarget(rawTarget, names)
	if !ok {
		return nil, assignTargetError(rawTarget, names)
	}
	// The last column of that name, which is the one applyStatement writes: a
	// name held by two columns is read from the later one.
	target := -1
	for i, name := range names {
		if name == resolved {
			target = i
		}
	}
	return &wholeColumnStage{values: values, target: target}, nil
}

// push writes the run's own rows of the column into the run. A run with no rows
// gets the column too, an empty one, so the stages after this one see the columns
// they see for every other run.
func (s *wholeColumnStage) push(r cclRun) ([]cclRun, error) {
	end := r.offset + r.rows()
	if end > len(s.values) {
		return nil, fmt.Errorf("the file holds rows %d to %d, past the %d rows the statement was computed over; it changed while it was being read",
			r.offset, end-1, len(s.values))
	}
	// Capped, so no stage that appends to the column can write into the values
	// the next run is cut from.
	part := s.values[r.offset:end:end]

	cols := fullColumns(r)
	arrays := runArrays(r)
	if s.target >= 0 {
		cols[s.target] = part
		arrays[s.target] = nil
		return []cclRun{{offset: r.offset, names: r.names, cols: cols, arrays: arrays}}, nil
	}
	// Clip, so the append always gives a new array: names may be a run's own
	// slice, and writing into its spare capacity would change what it holds.
	return []cclRun{{
		offset: r.offset,
		names:  append(slices.Clip(r.names), s.newName),
		cols:   append(cols, part),
		arrays: append(arrays, nil),
	}}, nil
}

// flush returns nothing: every row this stage was given is finished.
func (s *wholeColumnStage) flush() ([]cclRun, error) {
	return nil, nil
}

// cclPipeline runs its stages in order: the rows a stage finishes go on to the
// next one, and the rows leaving the last one are the result.
type cclPipeline []cclStage

// push runs one run through every stage, in order.
func (p cclPipeline) push(r cclRun) ([]cclRun, error) {
	runs := []cclRun{r}
	for _, stage := range p {
		next := make([]cclRun, 0, len(runs))
		for _, in := range runs {
			out, err := stage.push(in)
			if err != nil {
				return nil, err
			}
			next = append(next, out...)
		}
		runs = next
	}
	return runs, nil
}

// flush finishes what the stages still hold: each stage is given the runs the
// stages before it held back and is flushed itself, in order, so every row the
// file held leaves the pipeline exactly once.
func (p cclPipeline) flush() ([]cclRun, error) {
	var runs []cclRun
	for _, stage := range p {
		next := make([]cclRun, 0, len(runs)+1)
		for _, in := range runs {
			out, err := stage.push(in)
			if err != nil {
				return nil, err
			}
			next = append(next, out...)
		}
		held, err := stage.flush()
		if err != nil {
			return nil, err
		}
		runs = append(next, held...)
	}
	return runs, nil
}

// sequenceStage runs a statement whose whole right-hand side is a built-in
// sequence function. The function reads rows the batch after the current one has
// not arrived with, so the rows whose values are not settled yet are held until
// the values they wait for come, and handed on in file order. Only the history
// the function itself reads crosses a batch boundary: the rows being held are
// the ones the function asked to look ahead over.
type sequenceStage struct {
	seq *ccl.TopSequence

	// newName is the column a NEW statement creates, and target the position an
	// assignment replaces. The answers go there.
	newName string
	target  int

	// pending holds the rows the sequence has not answered yet, in file order,
	// as the batches they arrived in. They stay the batches they were read as:
	// the rows are handed on as they are, each batch keeping its own columns and
	// its own row count, so a run is never copied to join two batches together.
	pending []cclRun
}

// push answers the rows the sequence has every value for and holds the rest
// until the next push or the flush. Only the rows just pushed are handed to the
// sequence, which keeps the rows it is still waiting on itself; the rows the
// stage holds are here to know which rows the answers it gives back belong to.
func (s *sequenceStage) push(r cclRun) ([]cclRun, error) {
	if r.rows() == 0 {
		// A run with no rows has nothing for a sequence to read, but the
		// column it writes into is still a column of the run afterwards, or the
		// stages after this one would see a different set of columns than the
		// stages before it.
		return []cclRun{s.write(r, nil)}, nil
	}

	values, err := s.seq.Push(newRunContext(r))
	if err != nil {
		return nil, err
	}

	// The rows just pushed join the rows still waiting at the back of the queue,
	// so the answers the stream gives back can be taken off the front in file
	// order.
	s.pending = append(s.pending, r)
	if len(values) > 0 {
		// The stream answers in row order, so the rows it has answered are the
		// first ones it was given, oldest batch first, and the rest wait for the
		// values after them.
		answered, _ := s.take(len(values))
		return s.answer(answered, values), nil
	}
	return nil, nil
}

// flush answers the rows still held: the file has ended, so the values a
// look-ahead was waiting for will not arrive, and the function answers them the
// way it answers them on the whole column.
func (s *sequenceStage) flush() ([]cclRun, error) {
	held, rows := s.take(-1)
	if rows == 0 {
		return nil, nil
	}
	values, err := s.seq.Flush()
	if err != nil {
		return nil, err
	}
	if len(values) != rows {
		return nil, fmt.Errorf("a sequence function answered %d of the %d rows it was still holding",
			len(values), rows)
	}
	return s.answer(held, values), nil
}

// take removes rows rows from the front of what the stage holds, or everything
// when rows is negative, and returns them in file order. The runs it returns are
// the runs it held: a count that does not reach the end of a batch takes that
// batch's share of the columns with it, which is what a run of part of a batch
// is.
func (s *sequenceStage) take(rows int) (taken []cclRun, count int) {
	held := s.pending
	s.pending = nil
	for i, r := range held {
		if rows >= 0 && count+r.rows() > rows {
			// This batch holds rows past the count asked for, so it is split: what
			// was asked for is taken and what was not goes back to the front, with
			// the batches behind it still in file order.
			at := rows - count
			if at > 0 {
				taken = append(taken, headRun(r, at))
			}
			if kept := tailRun(r, at); kept.rows() > 0 {
				s.pending = append(s.pending, kept)
			}
			s.pending = append(s.pending, held[i+1:]...)
			return taken, rows
		}
		taken = append(taken, r)
		count += r.rows()
		if rows >= 0 && count == rows {
			s.pending = append(s.pending, held[i+1:]...)
			return taken, count
		}
	}
	return taken, count
}

// answer puts the values a sequence answered into the rows they belong to, which
// may span several batches, and hands those rows on in file order.
func (s *sequenceStage) answer(rows []cclRun, values []any) []cclRun {
	out := make([]cclRun, 0, len(rows))
	at := 0
	for _, r := range rows {
		out = append(out, s.write(r, values[at:at+r.rows()]))
		at += r.rows()
	}
	return out
}

// write puts the values a sequence answered into a run: a NEW statement's column
// is added and an assignment's target is replaced, so the statement after this
// one reads the answers as columns of its own.
func (s *sequenceStage) write(r cclRun, values []any) cclRun {
	cols := fullColumns(r)
	arrays := runArrays(r)
	if s.target >= 0 {
		cols[s.target] = values
		arrays[s.target] = nil
		return cclRun{offset: r.offset, names: r.names, cols: cols, arrays: arrays}
	}
	// Clip, so the append always gives a new array: names may be a run's own
	// slice, and writing into its spare capacity would change what it holds.
	return cclRun{
		offset: r.offset,
		names:  append(slices.Clip(r.names), s.newName),
		cols:   append(cols, values),
		arrays: append(arrays, nil),
	}
}

// fullColumns returns a copy of r's columns with one column for every name r
// has. A run of no rows may know its names and hold no columns at all (tailRun
// makes one), and every stage that writes into a column by its position needs the
// column to be there.
func fullColumns(r cclRun) [][]any {
	cols := slices.Clone(r.cols)
	for len(cols) < len(r.names) {
		cols = append(cols, []any{})
	}
	return cols
}

// runArrays returns a copy of r's arrays with one entry, possibly nil, for every
// name r has, the way fullColumns does for its values.
func runArrays(r cclRun) []arrow.Array {
	arrays := slices.Clone(r.arrays)
	for len(arrays) < len(r.names) {
		arrays = append(arrays, nil)
	}
	return arrays
}

// headRun returns the first rows of r, which still starts where r starts.
func headRun(r cclRun, rows int) cclRun {
	if rows >= r.rows() {
		return r
	}
	cols := make([][]any, len(r.cols))
	for i, col := range r.cols {
		cols[i] = col[:rows]
	}
	arrays := make([]arrow.Array, len(r.arrays))
	for i, arr := range r.arrays {
		if arr != nil {
			arrays[i] = array.NewSlice(arr, 0, int64(rows))
		}
	}
	return cclRun{offset: r.offset, names: r.names, cols: cols, arrays: arrays}
}

// tailRun returns the rows of r from rows onwards, which start that many rows
// further into the file. A run of no rows keeps its names, because the columns
// are the same however many rows there are, and no arrays: it has no rows of the
// file's own to keep.
func tailRun(r cclRun, rows int) cclRun {
	if rows <= 0 {
		return r
	}
	if rows >= r.rows() {
		return cclRun{names: r.names}
	}
	cols := make([][]any, len(r.cols))
	for i, col := range r.cols {
		cols[i] = col[rows:]
	}
	arrays := make([]arrow.Array, len(r.arrays))
	for i, arr := range r.arrays {
		if arr != nil {
			arrays[i] = array.NewSlice(arr, int64(rows), int64(r.rows()))
		}
	}
	return cclRun{offset: r.offset + rows, names: r.names, cols: cols, arrays: arrays}
}

// newPipeline builds the stages for the statements nodes, which have been
// resolved for a table of totalRows rows whose columns are colNames. Every
// statement is a stage of its own, so the rows one writes reach the next one
// whatever order they finish in, and a statement is built against the columns
// the statements before it leave behind. A statement whose position is in
// precomputed was computed over whole columns already (wholeColumnValues), and is
// a stage that hands on that column instead of computing it.
func newPipeline(nodes []ccl.CCLNode, totalRows int, colNames []string, precomputed map[int][]any) (cclPipeline, error) {
	names := slices.Clone(colNames)
	p := make(cclPipeline, 0, len(nodes))
	for i, node := range nodes {
		var stage cclStage
		var err error
		if values, whole := precomputed[i]; whole {
			stage, err = newWholeColumnStage(node, values, names)
		} else {
			stage, err = newStatementStage(node, totalRows, names)
		}
		if err != nil {
			return nil, err
		}
		p = append(p, stage)
		if newName, _, isNew := ccl.GetNewColInfo(node); isNew {
			names = append(names, newName)
		}
	}
	return p, nil
}

// newStatementStage returns the stage that runs one statement over the runs a
// file arrives in. A statement whose whole right-hand side is a built-in
// sequence function is a sequenceStage, which settles the rows the sequence has
// every value for and holds the rest back; every other statement is a
// statementStage, which finishes every row of the run it is given.
func newStatementStage(node ccl.CCLNode, totalRows int, names []string) (cclStage, error) {
	newName, _, isNew := ccl.GetNewColInfo(node)
	target := -1
	if !isNew {
		rawTarget, isAssignment := ccl.GetAssignmentTarget(node)
		if !isAssignment {
			// A statement that is neither a NEW nor an assignment writes
			// nothing, the way ExecuteCCL leaves it alone, so there is no column
			// for a sequence to answer into. It is run the ordinary way, over
			// the rows of the run, and what it answers is left alone too.
			return &statementStage{node: node, totalRows: totalRows}, nil
		}
		// The target is resolved here rather than per run, so a target that
		// names no column is refused before the file is read.
		resolved, ok := resolveAssignTarget(rawTarget, names)
		if !ok {
			return nil, assignTargetError(rawTarget, names)
		}
		target = slices.Index(names, resolved)
	}

	seq, isSequence, err := ccl.NewTopSequence(ccl.GetExpressionNode(node), totalRows, names)
	if err != nil {
		return nil, err
	}
	if !isSequence {
		return &statementStage{node: node, totalRows: totalRows}, nil
	}
	return &sequenceStage{seq: seq, newName: newName, target: target}, nil
}

// exceedsInt64Loss reports whether v is a value an int64 cannot hold without
// loss: a float that is not a whole number, one outside int64's range, or a
// number of a type that is not one. A nil is a missing value, which every type
// holds as a null, so it is never lossy.
func exceedsInt64Loss(v any) bool {
	switch n := v.(type) {
	case int, int8, int16, int32, int64, uint8, uint16, uint32:
		return false
	case uint:
		// As wide as an int on the platforms this runs on, so a uint above
		// int64's maximum is a value an int64 cannot hold.
		return uint64(n) > math.MaxInt64
	case uint64:
		return n > math.MaxInt64
	case float64:
		return !isInt64WholeFloat(n)
	case float32:
		// A float32 widens to float64 exactly, so the float64 answer is the one.
		return !isInt64WholeFloat(float64(n))
	default:
		return true
	}
}

// int64MinAsFloat and int64LimitAsFloat are int64's own bounds as float64s, which
// is the only way they are exactly representable: math.MaxInt64 rounds up to the
// limit, so a float64 test written with it would admit a value int64 cannot hold.
const (
	int64MinAsFloat   = -9223372036854775808.0
	int64LimitAsFloat = 9223372036854775808.0
)

// isInt64WholeFloat reports whether f is a finite whole number inside int64's
// range, which is what int64(v) would then hold without loss. The upper bound is
// exclusive: int64 stops one below 2^63.
func isInt64WholeFloat(f float64) bool {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return false
	}
	return f == math.Trunc(f) && f >= int64MinAsFloat && f < int64LimitAsFloat
}

// float64ExactInteger is the largest magnitude up to which a float64 holds every
// integer: its significand is 53 bits wide, so past 2^53 it holds every second
// integer, then every fourth, and an integer between them rounds to a neighbour.
const float64ExactInteger = 1 << 53

// exceedsFloat64Loss reports whether v is an integer a float64 cannot hold
// exactly: one of magnitude above 2^53. Every other value is not this function's
// business: a float is a float64 already (a float32 widens to one exactly), a nil
// is a null, and a value of another kind is held or refused by the kind flags.
func exceedsFloat64Loss(v any) bool {
	switch n := v.(type) {
	case int:
		return int64(n) > float64ExactInteger || int64(n) < -float64ExactInteger
	case int64:
		return n > float64ExactInteger || n < -float64ExactInteger
	case uint:
		return uint64(n) > float64ExactInteger
	case uint64:
		return n > float64ExactInteger
	default:
		// int8 to int32 and uint8 to uint32 are within 2^32.
		return false
	}
}

// narrowTypes is a set of the types narrower than int64 and float64 that a column
// the file had can keep when a statement assigns to it: the integer widths below
// 64 bits, uint64, float32, and the two date types. columnKinds keeps, for each,
// whether a value has been added that the type cannot hold.
type narrowTypes uint16

const (
	narrowInt8 narrowTypes = 1 << iota
	narrowInt16
	narrowInt32
	narrowUint8
	narrowUint16
	narrowUint32
	narrowUint64
	narrowFloat32
	narrowDate32
	narrowDate64

	narrowWholeNumbers = narrowInt8 | narrowInt16 | narrowInt32 | narrowUint8 | narrowUint16 | narrowUint32 | narrowUint64
	narrowDates        = narrowDate32 | narrowDate64
	narrowEvery        = narrowWholeNumbers | narrowFloat32 | narrowDates
)

// narrowTypeOf returns the narrow type the Arrow type id names, and zero for a
// type that is not one.
func narrowTypeOf(id arrow.Type) narrowTypes {
	switch id {
	case arrow.INT8:
		return narrowInt8
	case arrow.INT16:
		return narrowInt16
	case arrow.INT32:
		return narrowInt32
	case arrow.UINT8:
		return narrowUint8
	case arrow.UINT16:
		return narrowUint16
	case arrow.UINT32:
		return narrowUint32
	case arrow.UINT64:
		return narrowUint64
	case arrow.FLOAT32:
		return narrowFloat32
	case arrow.DATE32:
		return narrowDate32
	case arrow.DATE64:
		return narrowDate64
	default:
		return 0
	}
}

// wholeNumber is a whole number as a sign and a magnitude, which holds every
// value of every Go integer type, int64's smallest and uint64's largest included:
// no one of int64 and uint64 holds both.
type wholeNumber struct {
	negative  bool
	magnitude uint64
}

// wholeBounds is the largest magnitude an integer type holds on each side of
// zero.
type wholeBounds struct {
	negative, positive uint64
}

// The bounds of the integer types narrower than int64. An unsigned type holds no
// negative magnitude, and a zero is never negative, so its negative bound of zero
// refuses exactly the values below zero.
var (
	boundsInt8   = wholeBounds{negative: 1 << 7, positive: 1<<7 - 1}
	boundsInt16  = wholeBounds{negative: 1 << 15, positive: 1<<15 - 1}
	boundsInt32  = wholeBounds{negative: 1 << 31, positive: 1<<31 - 1}
	boundsUint8  = wholeBounds{positive: math.MaxUint8}
	boundsUint16 = wholeBounds{positive: math.MaxUint16}
	boundsUint32 = wholeBounds{positive: math.MaxUint32}
	boundsUint64 = wholeBounds{positive: math.MaxUint64}
)

// wholeTypes lists the integer types of narrowTypes with their bounds.
var wholeTypes = [...]struct {
	bit    narrowTypes
	bounds wholeBounds
}{
	{narrowInt8, boundsInt8},
	{narrowInt16, boundsInt16},
	{narrowInt32, boundsInt32},
	{narrowUint8, boundsUint8},
	{narrowUint16, boundsUint16},
	{narrowUint32, boundsUint32},
	{narrowUint64, boundsUint64},
}

// fits reports whether w is within bounds.
func (w wholeNumber) fits(bounds wholeBounds) bool {
	if w.negative {
		return w.magnitude <= bounds.negative
	}
	return w.magnitude <= bounds.positive
}

// signed returns w as an int64, which is w itself for a number a signed type
// narrower than int64 holds.
func (w wholeNumber) signed() int64 {
	if w.negative {
		return -int64(w.magnitude)
	}
	return int64(w.magnitude)
}

// lost returns the integer types of narrowTypes that w is outside of, leaving out
// the ones in known: the caller has a value outside of those already, and asks
// only about the rest.
func (w wholeNumber) lost(known narrowTypes) narrowTypes {
	var lost narrowTypes
	for i := range wholeTypes {
		t := &wholeTypes[i]
		if known&t.bit == 0 && !w.fits(t.bounds) {
			lost |= t.bit
		}
	}
	return lost
}

// wholeNumberOf returns v as a whole number when it is one: a Go integer of any
// type, or a float that is finite and has no fraction and is smaller than 2^64 in
// magnitude. Anything else, a text, a boolean, a time, a float with a fraction, a
// NaN or an infinity, is not.
func wholeNumberOf(v any) (wholeNumber, bool) {
	switch n := v.(type) {
	case int:
		return wholeOfInt(int64(n)), true
	case int8:
		return wholeOfInt(int64(n)), true
	case int16:
		return wholeOfInt(int64(n)), true
	case int32:
		return wholeOfInt(int64(n)), true
	case int64:
		return wholeOfInt(n), true
	case uint:
		return wholeNumber{magnitude: uint64(n)}, true
	case uint8:
		return wholeNumber{magnitude: uint64(n)}, true
	case uint16:
		return wholeNumber{magnitude: uint64(n)}, true
	case uint32:
		return wholeNumber{magnitude: uint64(n)}, true
	case uint64:
		return wholeNumber{magnitude: n}, true
	case float64:
		return wholeOfFloat(n)
	case float32:
		// A float32 widens to float64 exactly.
		return wholeOfFloat(float64(n))
	default:
		return wholeNumber{}, false
	}
}

func wholeOfInt(n int64) wholeNumber {
	if n < 0 {
		// Written so that int64's smallest value, whose negation overflows, has
		// its magnitude.
		return wholeNumber{negative: true, magnitude: uint64(-(n + 1)) + 1}
	}
	return wholeNumber{magnitude: uint64(n)}
}

// float64Limit64 is 2^64, the first magnitude a uint64 cannot hold, exactly as a
// float64.
const float64Limit64 = 1 << 64

func wholeOfFloat(f float64) (wholeNumber, bool) {
	if math.IsNaN(f) || math.IsInf(f, 0) || f != math.Trunc(f) || math.Abs(f) >= float64Limit64 {
		return wholeNumber{}, false
	}
	// A zero, either sign, is no negative number.
	return wholeNumber{negative: f < 0, magnitude: uint64(math.Abs(f))}, true
}

// float32ExactInteger is the largest magnitude up to which a float32 holds every
// integer: its significand is 24 bits wide.
const float32ExactInteger = 1 << 24

// exactInFloat32 reports whether a float32 holds w exactly, which it does up to
// 2^24 in magnitude.
func (w wholeNumber) exactInFloat32() bool {
	return w.magnitude <= float32ExactInteger
}

// float32Of returns v as the float32 that holds it without loss, and false for a
// value no float32 holds: a float64 that is not a float32 once rounded to one, an
// integer past 2^24 in magnitude, or a value that is not a number. A NaN and the
// infinities are held, and a float32 is its own.
func float32Of(v any) (float32, bool) {
	switch n := v.(type) {
	case float32:
		return n, true
	case float64:
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return float32(n), true
		}
		// Past the largest float32 the conversion would be an infinity, which is
		// not the number, and Go leaves an out of range conversion unspecified.
		if math.Abs(n) > math.MaxFloat32 {
			return 0, false
		}
		f := float32(n)
		return f, float64(f) == n
	}
	w, ok := wholeNumberOf(v)
	if !ok || !w.exactInFloat32() {
		return 0, false
	}
	if w.negative {
		return -float32(w.magnitude), true
	}
	return float32(w.magnitude), true
}

// secondsPerDay is the length of a day in a Unix timestamp, which counts no leap
// seconds.
const secondsPerDay = 24 * 60 * 60

// utcMidnight returns v as a time when it is exactly midnight UTC, in whatever
// zone it was given, and false for any other value.
func utcMidnight(v any) (time.Time, bool) {
	t, ok := v.(time.Time)
	if !ok {
		return time.Time{}, false
	}
	return t, t.Nanosecond() == 0 && t.Unix()%secondsPerDay == 0
}

// date32Of returns v as the date32 that holds it without loss: a time at midnight
// UTC whose day number an int32 holds.
func date32Of(v any) (arrow.Date32, bool) {
	t, ok := utcMidnight(v)
	if !ok {
		return 0, false
	}
	if days := t.Unix() / secondsPerDay; days < math.MinInt32 || days > math.MaxInt32 {
		return 0, false
	}
	return arrow.Date32FromTime(t), true
}

// date64Of returns v as the date64 that holds it without loss: a time at midnight
// UTC, which a date64 holds as the milliseconds from the epoch to that midnight,
// so long as an int64 holds them.
func date64Of(v any) (arrow.Date64, bool) {
	t, ok := utcMidnight(v)
	if !ok {
		return 0, false
	}
	if sec := t.Unix(); sec < math.MinInt64/1000 || sec > math.MaxInt64/1000 {
		return 0, false
	}
	return arrow.Date64FromTime(t), true
}

// narrowLossOf returns the narrow types v is a value outside of, leaving out some
// of those in known: the caller has a value outside of them already, and is not
// asking about them again. A nil is a missing value, which every type holds as a
// null, so it is the caller's to skip. It answers with the same rules the builders
// convert by, so a value it says a type holds is a value that type's builder
// writes as itself.
//
// It runs for every value of every column Write infers a type from, so the two
// integer types a table holds nearly always are answered first and by one switch.
func narrowLossOf(v any, known narrowTypes) narrowTypes {
	switch x := v.(type) {
	case int64:
		return integerLoss(wholeOfInt(x), known)
	case int:
		return integerLoss(wholeOfInt(int64(x)), known)
	case float64:
		lost := narrowDates | floatLoss(x, known)
		if known&narrowFloat32 == 0 {
			if _, ok := float32Of(x); !ok {
				lost |= narrowFloat32
			}
		}
		return lost
	case float32:
		// A float32 is a float32 already.
		return narrowDates | floatLoss(float64(x), known)
	case int8, int16, int32, uint, uint8, uint16, uint32, uint64:
		w, _ := wholeNumberOf(v)
		return integerLoss(w, known)
	case time.Time:
		lost := narrowWholeNumbers | narrowFloat32
		if known&narrowDate32 == 0 {
			if _, ok := date32Of(x); !ok {
				lost |= narrowDate32
			}
		}
		if known&narrowDate64 == 0 {
			if _, ok := date64Of(x); !ok {
				lost |= narrowDate64
			}
		}
		return lost
	default:
		return narrowEvery
	}
}

// integerLoss returns the narrow types a Go integer w is a value outside of.
func integerLoss(w wholeNumber, known narrowTypes) narrowTypes {
	lost := narrowDates | w.lost(known)
	if !w.exactInFloat32() {
		lost |= narrowFloat32
	}
	return lost
}

// floatLoss returns the integer types a float f is a value outside of: all of them
// unless it is a whole number.
func floatLoss(f float64, known narrowTypes) narrowTypes {
	if w, ok := wholeOfFloat(f); ok {
		return w.lost(known)
	}
	return narrowWholeNumbers
}

// holdsNoValue reports whether kinds has seen no value at all: every value that
// is not a missing one sets one of the kind flags, so a column with none set has
// held nothing but missing values, or nothing.
func holdsNoValue(kinds *columnKinds) bool {
	return !kinds.hasInt && !kinds.hasFloat && !kinds.hasString && !kinds.hasBool &&
		!kinds.hasTime && !kinds.hasBytes && !kinds.hasOther
}

// holdsEveryValue reports whether every value recorded in kinds so far can be
// held by dtype without loss, which is what lets a column the file had keep its
// own type. A column holding nothing but missing values holds every type: there
// is no value of another kind, and a missing value is a null whatever the type.
//
// Only the types ApplyCCL builds an array of answer. Any other type is refused,
// so a column the file declared at a type this builder cannot rebuild is written
// the way Write would write it rather than kept.
func holdsEveryValue(kinds *columnKinds, dtype arrow.DataType) bool {
	if kinds == nil || dtype == nil {
		return false
	}
	if holdsNoValue(kinds) {
		return true
	}

	// Each arm is the kind the type holds and nothing else: a column of mixed
	// kinds is written as the type Write gives it, because there is no type this
	// builder holds every one of them in.
	switch dtype.ID() {
	case arrow.INT64:
		// Any whole number an int64 holds, whatever Go type it arrived as.
		return !kinds.lossyInt64
	case arrow.FLOAT64:
		// Any float, and the integers a float64 holds exactly: one past 2^53 would
		// come back as a neighbour of itself.
		return !kinds.hasString && !kinds.hasBool && !kinds.hasTime && !kinds.hasBytes && !kinds.hasOther &&
			!kinds.lossyFloat64
	case arrow.STRING:
		return !kinds.hasInt && !kinds.hasFloat && !kinds.hasBool && !kinds.hasTime && !kinds.hasBytes && !kinds.hasOther
	case arrow.BOOL:
		return kinds.hasBool && !kinds.hasInt && !kinds.hasFloat && !kinds.hasString &&
			!kinds.hasTime && !kinds.hasBytes && !kinds.hasOther
	case arrow.TIMESTAMP:
		return kinds.hasTime && !kinds.hasInt && !kinds.hasFloat && !kinds.hasString &&
			!kinds.hasBool && !kinds.hasBytes && !kinds.hasOther
	case arrow.BINARY:
		return kinds.hasBytes && !kinds.hasInt && !kinds.hasFloat && !kinds.hasString &&
			!kinds.hasBool && !kinds.hasTime && !kinds.hasOther
	case arrow.INT8, arrow.INT16, arrow.INT32, arrow.UINT8, arrow.UINT16, arrow.UINT32, arrow.UINT64, arrow.FLOAT32:
		// Numbers and nothing else, every one of them a value this type holds: a
		// whole number inside its range from any Go integer or a whole float, and
		// for a float32 a float that survives the trip through it.
		return !kinds.hasString && !kinds.hasBool && !kinds.hasTime && !kinds.hasBytes && !kinds.hasOther &&
			kinds.lossyNarrow&narrowTypeOf(dtype.ID()) == 0
	case arrow.DATE32, arrow.DATE64:
		// Times and nothing else, every one of them at midnight UTC.
		return kinds.hasTime && !kinds.hasInt && !kinds.hasFloat && !kinds.hasString &&
			!kinds.hasBool && !kinds.hasBytes && !kinds.hasOther &&
			kinds.lossyNarrow&narrowTypeOf(dtype.ID()) == 0
	default:
		return false
	}
}

// writtenType is the type a column a statement writes is written as. A column the
// file had keeps its own type, original, when every value written into it can be
// held by that type without loss; otherwise, and for a column the script creates
// (original nil), it takes the type Write would give its values.
func writtenType(original arrow.DataType, kinds *columnKinds) arrow.DataType {
	if original != nil && holdsEveryValue(kinds, original) {
		return original
	}
	return kinds.arrowType()
}

// originalField returns the field the file gave the column named name, and false
// for a column the file did not have, which is every column a script creates and
// every one a stage added without the script naming it.
func originalField(schema *arrow.Schema, name string) (arrow.Field, bool) {
	if schema == nil {
		return arrow.Field{}, false
	}
	if idx := schema.FieldIndices(name); len(idx) > 0 {
		return schema.Field(idx[0]), true
	}
	return arrow.Field{}, false
}

// writtenField is the field a column the script writes is written with. A column
// keeping the file's own type keeps the file's own field with it — its unit, its
// time zone and its metadata — and only nullability is turned on, because a value
// expression is nil wherever it has no answer, which for a look-ahead is the rows
// before its window is full. A column written as another type is a field of its
// own, since the file's field would describe a type the column no longer has.
func writtenField(name string, dtype arrow.DataType, original arrow.Field, hasOriginal bool) arrow.Field {
	if hasOriginal && arrow.TypeEqual(dtype, original.Type) {
		original.Nullable = true
		return original
	}
	return arrow.Field{Name: name, Type: dtype, Nullable: true}
}

// writtenColumns returns the names of the columns the script writes, in the
// order it first writes each one: a column a statement creates and a column a
// statement assigns to both count, because both have their values decided by the
// script. names is the columns the file came with.
//
// A column the script writes takes the type its values answer, except that a
// column the file had keeps the file's own type while every value written into it
// is one that type holds; either way its field can hold a missing value: a value
// expression is nil wherever it has no answer, which for a look-ahead is the rows
// before its window is full. A column the script does not write is none of the
// script's business and keeps the field the file gave it, nullability included.
func writtenColumns(nodes []ccl.CCLNode, names []string) []string {
	running := slices.Clone(names)
	var written []string
	seen := make(map[string]bool, len(running))
	note := func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		written = append(written, name)
	}
	for _, node := range nodes {
		newName, _, isNew := ccl.GetNewColInfo(node)
		if isNew {
			note(newName)
			running = append(running, newName)
			continue
		}
		rawTarget, isAssignment := ccl.GetAssignmentTarget(node)
		if !isAssignment {
			// A statement that is neither a NEW nor an assignment writes
			// nothing, the way ExecuteCCL leaves it alone.
			continue
		}
		// Resolved the way the stage resolves its target, against the columns
		// the statements before it leave behind. A target naming no column was
		// refused when the stage was built, so it is not this function's error
		// to report again.
		if resolved, ok := resolveAssignTarget(rawTarget, running); ok {
			note(resolved)
		}
	}
	return written
}

// cclOutputSchema builds the schema the written file has: a column the script
// writes is a nullable field of the type given in writtenTypes, or of the type
// writtenType settles from the values the column has held (kinds) when
// writtenTypes has none, and a column it does not write keeps the field it came
// with.
//
// The written types are the type of the values of the whole file, so a column
// whose first rows are missing or of one kind and whose later rows are of another
// is written as the type all of them answer, which is what Write gives the same
// column. A column the file had and whose every value its own type holds is the
// one exception: it keeps the file's type and the file's field, and only its
// nullability is turned on.
func cclOutputSchema(colNames []string, written []string, writtenTypes map[string]arrow.DataType, kinds map[string]*columnKinds, originalSchema *arrow.Schema) *arrow.Schema {
	isWritten := make(map[string]bool, len(written))
	for _, name := range written {
		isWritten[name] = true
	}
	// A column that held nothing at all is a column of missing values, which is
	// what a kinds nobody added to is.
	kindsOf := func(name string) *columnKinds {
		if k := kinds[name]; k != nil {
			return k
		}
		return &columnKinds{}
	}

	fields := make([]arrow.Field, 0, len(colNames))
	for _, colName := range colNames {
		if isWritten[colName] {
			original, hasOriginal := originalField(originalSchema, colName)
			dtype := writtenTypes[colName]
			if dtype == nil {
				dtype = writtenType(original.Type, kindsOf(colName))
			}
			fields = append(fields, writtenField(colName, dtype, original, hasOriginal))
			continue
		}
		if field, ok := originalField(originalSchema, colName); ok {
			fields = append(fields, field)
			continue
		}
		// A column neither the file nor writtenColumns knows, which a stage added
		// without the script naming it: its own values are all there is to write it
		// from.
		fields = append(fields, arrow.Field{
			Name:     colName,
			Type:     kindsOf(colName).arrowType(),
			Nullable: true,
		})
	}
	return arrow.NewSchema(fields, nil)
}

// runValues returns the values of every column of r, the column names naming
// them. The slices are the run's own, which nothing it is handed to writes to.
func runValues(r cclRun) map[string][]any {
	values := make(map[string][]any, len(r.names))
	for i, name := range r.names {
		if i < len(r.cols) {
			values[name] = r.cols[i]
		}
	}
	return values
}

// buildArrowRecord constructs an arrow.Record from the values of the runs that
// have left the pipeline, in the schema the file is written with. No rows is
// errNothingToWrite, so an input with none leaves the original alone.
//
// schema holds one field per column in colNames, in that order, with each
// column's type already settled: a column the script writes takes the type its
// values answer, and a column it does not write keeps the field the file gave
// it. Every array is checked against its field before the record is built, so a
// type that does not match is an error here rather than a panic inside Arrow.
//
// arrays holds the file's own array for every column that still has it, in step
// with colNames, possibly shorter and possibly nil in places. A column whose
// array is the one its field describes is written from that array rather than
// from the Go values, so it comes back as the file had it whatever its type: a
// list, a struct or a decimal at its own scale never passed through Go at all.
func buildArrowRecord(values map[string][]any, colNames []string, arrays []arrow.Array, schema *arrow.Schema) (rec arrow.Record, err error) {
	mem := memory.DefaultAllocator

	if len(colNames) == 0 {
		return nil, errNothingToWrite
	}
	rows := len(values[colNames[0]])
	for _, colName := range colNames {
		if len(values[colName]) != rows {
			return nil, fmt.Errorf("column %q holds %d values, column %q holds %d",
				colNames[0], rows, colName, len(values[colName]))
		}
	}
	if rows == 0 {
		return nil, errNothingToWrite
	}

	built := make([]arrow.Array, 0, len(colNames))
	defer func() {
		// An error after a builder finished leaves the arrays already built
		// holding their buffers, so they are released here rather than by a
		// caller that only sees the error.
		if err != nil {
			for _, arr := range built {
				arr.Release()
			}
		}
	}()

	for i, colName := range colNames {
		field := schema.Field(i)
		// The file's own array, where there is one and it is the array of this
		// column's rows: a run holds an array exactly as long as it holds values
		// of that column, so a length that does not match is a run that does not
		// describe the same rows as its values, and the values are what the
		// column is written from then.
		if i < len(arrays) && arrays[i] != nil && arrays[i].Len() == rows &&
			arrow.TypeEqual(arrays[i].DataType(), field.Type) {
			arrays[i].Retain()
			built = append(built, arrays[i])
			continue
		}
		if field.Type.ID() == arrow.EXTENSION {
			// A column the file gave an extension type for: the extension's
			// storage type is what a value can be written as, and the field
			// keeps the extension, so the file comes back holding the type it
			// had. Write builds every array with the plain type and never
			// reaches this arm.
			storage := field.Type
			if ext, ok := field.Type.(arrow.ExtensionType); ok {
				storage = ext.StorageType()
			}
			arr, buildErr := buildArrowArray(mem, colName, values[colName], storage)
			if buildErr != nil {
				return nil, buildErr
			}
			if !arrow.TypeEqual(arr.DataType(), storage) {
				err = fmt.Errorf("column %q built a %s array, its field is %s",
					colName, arr.DataType(), field.Type)
				arr.Release()
				return nil, err
			}
			built = append(built, arr)
			continue
		}

		arr, buildErr := buildArrowArray(mem, colName, values[colName], field.Type)
		if buildErr != nil {
			return nil, buildErr
		}
		if !arrow.TypeEqual(arr.DataType(), field.Type) {
			err = fmt.Errorf("column %q built a %s array, its field is %s",
				colName, arr.DataType(), field.Type)
			arr.Release()
			return nil, err
		}
		built = append(built, arr)
	}

	rec = array.NewRecord(schema, built, int64(rows))
	for _, arr := range built {
		arr.Release()
	}
	return rec, nil
}

// buildArrowArray constructs an arrow.Array of dtype from the values of one
// column. A missing value is written as a null whatever the column's type, and a
// value that does not convert to that type is an error: the file is left alone
// rather than holding a number that says something else.
//
// A type the builder has no builder for, such as a list column a file outside
// Write was written with, is an error naming the column and its type. Write's
// own inferArrowType never answers such a type, so this is a column of a file
// ApplyCCL did not write, not one of its own.
func buildArrowArray(mem memory.Allocator, name string, data []any, dtype arrow.DataType) (arr arrow.Array, err error) {
	defer func() {
		// conv reports a value it cannot read by panicking, which is the one
		// thing a library call must not do to the program calling it. The
		// column and its type are named here, because that is what says which
		// value of which column the file refuses to hold.
		if r := recover(); r != nil {
			arr = nil
			err = fmt.Errorf("cannot write column %q as %s: %v", name, dtype, r)
		}
	}()

	switch dtype.ID() {
	case arrow.INT64:
		builder := array.NewInt64Builder(mem)
		defer builder.Release()
		for _, v := range data {
			switch n := v.(type) {
			case nil:
				builder.AppendNull()
			case float64:
				// A whole number inside int64's range: the type was settled with
				// holdsEveryValue, which refused a fraction and a value out of
				// range, so int64(n) is the value itself and not a truncation of
				// one the way conv.ParseInt would be.
				builder.Append(int64(n))
			default:
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

	case arrow.BINARY:
		// Same answer as Write's appendValue: a byte slice is written as
		// itself, anything else as its text.
		builder := array.NewBinaryBuilder(mem, dtype.(arrow.BinaryDataType))
		defer builder.Release()
		for _, v := range data {
			if v == nil {
				builder.AppendNull()
				continue
			}
			b, ok := v.([]byte)
			if !ok {
				b = []byte(conv.ToString(v))
			}
			builder.Append(b)
		}
		return builder.NewArray(), nil

	case arrow.TIMESTAMP:
		// A timestamp is written at the unit the column's own type says, which is
		// the only way the file comes back holding the instant it held before. A
		// value that is not a time is a missing value here, the same answer
		// Write's appendValue gives it.
		tt := dtype.(*arrow.TimestampType)
		builder := array.NewTimestampBuilder(mem, tt)
		defer builder.Release()
		for _, v := range data {
			t, ok := v.(time.Time)
			switch {
			case v == nil, !ok:
				builder.AppendNull()
			default:
				ts, convErr := arrow.TimestampFromTime(t, tt.Unit)
				if convErr != nil {
					return nil, fmt.Errorf("cannot write %v into column %q as %s: %w",
						v, name, dtype, convErr)
				}
				builder.Append(ts)
			}
		}
		return builder.NewArray(), nil

	case arrow.INT8:
		return buildWholeArray(name, data, dtype, array.NewInt8Builder(mem), boundsInt8,
			func(w wholeNumber) int8 { return int8(w.signed()) })
	case arrow.INT16:
		return buildWholeArray(name, data, dtype, array.NewInt16Builder(mem), boundsInt16,
			func(w wholeNumber) int16 { return int16(w.signed()) })
	case arrow.INT32:
		return buildWholeArray(name, data, dtype, array.NewInt32Builder(mem), boundsInt32,
			func(w wholeNumber) int32 { return int32(w.signed()) })
	case arrow.UINT8:
		return buildWholeArray(name, data, dtype, array.NewUint8Builder(mem), boundsUint8,
			func(w wholeNumber) uint8 { return uint8(w.magnitude) })
	case arrow.UINT16:
		return buildWholeArray(name, data, dtype, array.NewUint16Builder(mem), boundsUint16,
			func(w wholeNumber) uint16 { return uint16(w.magnitude) })
	case arrow.UINT32:
		return buildWholeArray(name, data, dtype, array.NewUint32Builder(mem), boundsUint32,
			func(w wholeNumber) uint32 { return uint32(w.magnitude) })
	case arrow.UINT64:
		return buildWholeArray(name, data, dtype, array.NewUint64Builder(mem), boundsUint64,
			func(w wholeNumber) uint64 { return w.magnitude })

	case arrow.FLOAT32:
		builder := array.NewFloat32Builder(mem)
		defer builder.Release()
		for _, v := range data {
			if v == nil {
				builder.AppendNull()
				continue
			}
			f, ok := float32Of(v)
			if !ok {
				return nil, cannotHoldError(v, name, dtype, "a float32 holds no such number")
			}
			builder.Append(f)
		}
		return builder.NewArray(), nil

	case arrow.DATE32:
		builder := array.NewDate32Builder(mem)
		defer builder.Release()
		for _, v := range data {
			if v == nil {
				builder.AppendNull()
				continue
			}
			d, ok := date32Of(v)
			if !ok {
				return nil, cannotHoldError(v, name, dtype, "a date32 holds only a time at midnight UTC")
			}
			builder.Append(d)
		}
		return builder.NewArray(), nil

	case arrow.DATE64:
		builder := array.NewDate64Builder(mem)
		defer builder.Release()
		for _, v := range data {
			if v == nil {
				builder.AppendNull()
				continue
			}
			d, ok := date64Of(v)
			if !ok {
				return nil, cannotHoldError(v, name, dtype, "a date64 holds only a time at midnight UTC")
			}
			builder.Append(d)
		}
		return builder.NewArray(), nil

	default:
		return nil, fmt.Errorf("cannot write column %q as %s: ApplyCCL builds no array of that type",
			name, dtype)
	}
}

// cannotHoldError is the error for a value a column's type does not hold as it
// is, which is written as nothing rather than as another value: a conversion
// would wrap it around, cut its fraction or round it.
func cannotHoldError(v any, name string, dtype arrow.DataType, reason string) error {
	return fmt.Errorf("cannot write %v into column %q as %s: %s", v, name, dtype, reason)
}

// buildWholeArray builds the array of an integer type narrower than int64 from the
// values of one column: each is a whole number inside bounds, from any Go integer
// or a whole float, and a missing value is a null. A value that is not one is an
// error, since the type was settled with holdsEveryValue, which refused it.
func buildWholeArray[T any, B interface {
	array.Builder
	Append(T)
}](name string, data []any, dtype arrow.DataType, builder B, bounds wholeBounds, convert func(wholeNumber) T) (arrow.Array, error) {
	defer builder.Release()
	for _, v := range data {
		if v == nil {
			builder.AppendNull()
			continue
		}
		w, ok := wholeNumberOf(v)
		if !ok || !w.fits(bounds) {
			return nil, cannotHoldError(v, name, dtype, fmt.Sprintf("it is not a whole number %s holds", dtype))
		}
		builder.Append(convert(w))
	}
	return builder.NewArray(), nil
}

// cclBatchSize is how many rows FilterWithCCL and ApplyCCL read at a time.
// Every part of an expression that reads beyond the current row is computed
// over the whole file before the rows are evaluated (ccl.ResolveWholeTable),
// so the size changes how much is held in memory, not the answer.
const cclBatchSize = 1000

// cclFileInfo returns the row count, the column names and the schema of the
// file at path, as the batches FilterWithCCL and ApplyCCL read will name them.
// The schema is what a column the script does not write keeps: its own field,
// nullability and all, so an unchanged column comes back as the file had it.
func cclFileInfo(path string) (totalRows int, colNames []string, schema *arrow.Schema, err error) {
	defer func() {
		if r := recover(); r != nil {
			totalRows, colNames, schema, err = 0, nil, nil, unreadableFile(path, r)
		}
	}()

	f, err := os.Open(path)
	if err != nil {
		return 0, nil, nil, err
	}
	defer func() { _ = f.Close() }()

	r, err := file.NewParquetReader(f)
	if err != nil {
		return 0, nil, nil, err
	}
	defer func() { _ = r.Close() }()

	fr, err := pqarrow.NewFileReader(r, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
	if err != nil {
		return 0, nil, nil, err
	}

	schema, err = fr.Schema()
	if err != nil {
		return 0, nil, nil, err
	}
	colNames = make([]string, len(schema.Fields()))
	for i, field := range schema.Fields() {
		colNames[i] = field.Name
	}
	return int(r.NumRows()), colNames, schema, nil
}

// forEachRecord reads the file at path batch by batch and calls fn with every
// record, in order. The reader stops when fn returns, and fn must not keep rec
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

// appliedBatches reads the file at path for ccl.ResolveWholeTable as the
// statements before the one being resolved leave it: each run has had prior
// applied to it, so a statement reading a column an earlier one created or
// replaced sees that column. The context handed to the resolver is the very one
// the row-by-row pass evaluates those statements through, so a value a statement
// writes is the one every later statement reads; a record rebuilt from the file
// would round a written value through the column's own parquet type instead, and
// a column the script creates would take its type from the first batch only.
// With no prior statement every run is the file's own rows, which is what
// resolving the first statement needs. A prior statement in precomputed was
// computed over whole columns, and its column is read from there.
func appliedBatches(ctx context.Context, path string, colNames []string, prior []ccl.CCLNode, totalRows int, precomputed map[int][]any) ccl.Batches {
	return func(yield func(ccl.GlobalRowContext) error) error {
		// A stage holds what it has not finished, so the pipeline is built for
		// this pass rather than shared between them. The file's own columns
		// come first, and a statement creating one adds the next, which is the
		// order the statements before this one leave them in.
		pipeline, err := newPipeline(prior, totalRows, colNames, precomputed)
		if err != nil {
			return err
		}

		// A run holds Go values of its own, so the resolver may keep the
		// context it is given for longer than the record it was read from.
		hand := func(runs []cclRun) error {
			for _, run := range runs {
				if err := yield(newRunContext(run)); err != nil {
					return err
				}
			}
			return nil
		}

		offset := 0
		if err := forEachRecord(ctx, path, func(rec arrow.Record) error {
			run := runFromRecord(rec, colNames, offset)
			offset += int(rec.NumRows())
			runs, err := pipeline.push(run)
			if err != nil {
				return err
			}
			return hand(runs)
		}); err != nil {
			return err
		}

		// The file has ended, so whatever the stages still hold is finished
		// now; otherwise those rows would never reach the resolver.
		runs, err := pipeline.flush()
		if err != nil {
			return err
		}
		return hand(runs)
	}
}

// wholeColumnsLoaded counts the columns the whole-column passes have read, one for
// each column of each pass. The tests hold a pass to the columns its expression
// reads, which is what keeps an expression that needs whole columns from holding
// the file.
var wholeColumnsLoaded atomic.Int64

// columnsAfter returns the columns the statements nodes leave, in order: the
// file's own, and one for each NEW.
func columnsAfter(colNames []string, nodes []ccl.CCLNode) []string {
	names := slices.Clone(colNames)
	for _, node := range nodes {
		if newName, _, isNew := ccl.GetNewColInfo(node); isNew {
			names = append(names, newName)
		}
	}
	return names
}

// loadColumns reads the whole of the columns want, by position, from the file
// as the statements prior leave it, and nothing else: the columns not in want
// are nil. names are the columns those statements leave, in order.
func loadColumns(ctx context.Context, path string, colNames []string, prior []ccl.CCLNode, totalRows int, precomputed map[int][]any, want []int) (names []string, cols [][]any, err error) {
	wholeColumnsLoaded.Add(int64(len(want)))

	names = columnsAfter(colNames, prior)
	cols = make([][]any, len(names))
	for _, c := range want {
		if c < 0 || c >= len(names) {
			return nil, nil, fmt.Errorf("column %d was asked for, and the file has %d", c, len(names))
		}
		// Never nil, so a column with no rows is still a column that was read.
		cols[c] = make([]any, 0, totalRows)
	}

	pipeline, err := newPipeline(prior, totalRows, colNames, precomputed)
	if err != nil {
		return nil, nil, err
	}
	take := func(runs []cclRun) {
		for _, run := range runs {
			for _, c := range want {
				if c < len(run.cols) {
					cols[c] = append(cols[c], run.cols[c]...)
				}
			}
		}
	}
	if err := eachRun(ctx, path, colNames, func(r cclRun) error {
		runs, err := pipeline.push(r)
		if err != nil {
			return err
		}
		take(runs)
		return nil
	}); err != nil {
		return nil, nil, err
	}
	// The file has ended, so whatever the stages still hold is finished now.
	runs, err := pipeline.flush()
	if err != nil {
		return nil, nil, err
	}
	take(runs)
	return names, cols, nil
}

// wholeColumnValues evaluates the expression or statement node over the whole of
// the columns it reads, the way the DataTable CCL methods do, and returns the
// value each row of the file takes: for an expression, the expression's value;
// for a statement, the column it writes.
//
// The columns are the ones node reads and no others, read as the statements prior
// leave the file, and the context reports every column of the file, so a column
// letter means what it means on the whole table. A column node does not read is an
// error to read, not a column of missing values. The expression is bound against
// the column names, a column past the last one is refused, and an aggregate that
// does not change from row to row is computed once, as the table's methods do;
// the answer, and the error where there is one, are the table's.
func wholeColumnValues(ctx context.Context, path string, colNames []string, prior []ccl.CCLNode, totalRows int, precomputed map[int][]any, node ccl.CCLNode) ([]any, error) {
	names := columnsAfter(colNames, prior)
	colNameMap := make(map[string]int, len(names))
	for i, name := range names {
		colNameMap[name] = i
	}

	// An assignment's target is resolved before anything else, as the table does,
	// so a target that is not a column is the error, and not a column the
	// right-hand side reads, and the file is not read for a statement that cannot
	// be written.
	if target, isAssignment := ccl.GetAssignmentTarget(node); isAssignment {
		if _, ok := resolveAssignTarget(target, names); !ok {
			return nil, assignTargetError(target, names)
		}
	}

	bound, err := ccl.Bind(node, colNameMap)
	if err != nil {
		return nil, err
	}
	if word, index, found := ccl.FirstColPastEnd(bound, len(names)); found {
		letters, _ := insyra.CalcColIndex(index)
		return nil, ccl.PastLastColumnError(word, letters, len(names), colNameMap)
	}

	isStatement := ccl.IsNewColNode(bound) || ccl.IsAssignmentNode(bound)
	if isStatement && totalRows == 0 {
		// A file with no rows has nothing for a statement to evaluate, which is
		// what every other path does with it, and ApplyCCL leaves such a file as it
		// was.
		return []any{}, nil
	}

	want, err := ccl.ReferencedColumns(bound, len(names))
	if err != nil {
		return nil, err
	}
	_, cols, err := loadColumns(ctx, path, colNames, prior, totalRows, precomputed, want)
	if err != nil {
		return nil, err
	}
	wctx := newWholeColumnsContext(names, cols, totalRows)

	// An aggregate that does not read the current row has the same answer on every
	// row, so it is computed once here, which is what keeps A > MEDIAN(A) from
	// sorting the column once per row.
	if totalRows > 0 {
		bound = ccl.FoldRowInvariantAggregates(bound, wctx)
	}

	if isStatement {
		if err := applyStatement(ctx, wctx, bound, totalRows); err != nil {
			return nil, err
		}
		if _, _, isNew := ccl.GetNewColInfo(bound); isNew {
			return wctx.cols[len(wctx.cols)-1], nil
		}
		target, _ := ccl.GetAssignmentTarget(bound)
		resolved, ok := resolveAssignTarget(target, names)
		if !ok {
			return nil, assignTargetError(target, names)
		}
		return wctx.cols[wctx.colNameMap[resolved]], nil
	}

	values := make([]any, totalRows)
	if ccl.IsRowDependent(bound) {
		for row := range values {
			if row%cancelCheckRows == 0 {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
			}
			if err := wctx.SetRowIndex(row); err != nil {
				return nil, fmt.Errorf("failed to set row index %d: %w", row, err)
			}
			val, err := ccl.Evaluate(bound, wctx)
			if err != nil {
				return nil, fmt.Errorf("error evaluating CCL at row %d: %w", row, err)
			}
			values[row] = val
		}
		return values, nil
	}

	// An expression that does not depend on the row is evaluated once, on the first
	// row, and a value as long as the file is spread over the rows the way a loaded
	// table spreads it. A table with no rows is evaluated once as well.
	if totalRows > 0 {
		if err := wctx.SetRowIndex(0); err != nil {
			return nil, fmt.Errorf("failed to set row index 0: %w", err)
		}
	}
	val, err := ccl.Evaluate(bound, wctx)
	if err != nil {
		return nil, fmt.Errorf("error evaluating CCL: %w", err)
	}
	for row := range values {
		values[row] = rowOf(val, true, totalRows, row)
	}
	return values, nil
}

// filterColumnName is the column a streamed filter writes its decisions into, so
// the rows a sequence has not settled yet are held back with the rows they are
// waiting for and handed on in the same shape every other stage hands rows on.
const filterColumnName = "\x00filter"

// passesFilter reports whether a row a filter decided on is a row to keep: a
// boolean is its own answer, a number keeps the row when it is not zero, and
// anything else keeps the row when it is there at all. Both filter paths ask
// the same question, so both ask it here.
func passesFilter(val any) bool {
	switch v := val.(type) {
	case bool:
		return v
	case float64:
		return v != 0
	case int:
		return v != 0
	case int64:
		return v != 0
	default:
		return val != nil
	}
}

// eachRun reads the file at path cclBatchSize rows at a time and calls fn with
// every run of consecutive rows, in order, with the file position of its first
// row. The reading is forEachRecord's, so there is one stream of records in the
// file rather than two copies of the same loop to keep in step, and the reader
// stops when this call returns, so an early return cannot leave it blocked on a
// batch nobody reads, holding the file open.
func eachRun(ctx context.Context, path string, colNames []string, fn func(r cclRun) error) error {
	offset := 0
	return forEachRecord(ctx, path, func(rec arrow.Record) error {
		// A run holds Go values of its own, so it outlives the record it was read
		// from: forEachRecord releases the record once this call returns, and fn
		// may well have kept the run.
		run := runFromRecord(rec, colNames, offset)
		offset += int(rec.NumRows())
		return fn(run)
	})
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
// whole file, # is the row's position in the file, a fixed row such as A.0 is
// that row of the file, and a built-in sequence function that is the whole
// expression — LAG, LEAD, DIFF, PCT_CHANGE, a cumulative one or a ROLLING_* one
// — is streamed, holding the rows whose values need rows of a later batch.
// Computing the whole-file parts reads the file again before the rows are
// evaluated. An expression with a part that cannot be computed batch by batch —
// MEDIAN, an aggregate registered with RegisterAggregateFunction, a sequence
// function inside a larger expression or one a caller registered, a row
// reference computed from the current row such as A.(# - 1), a column range
// inside an aggregate's argument — is computed by reading the whole of the
// columns it reads, and only those, into memory and evaluating it the way the
// DataTable CCL methods do, so its answer, or its error, is the loaded table's.
// An expression that needs every column (@) holds the whole file that way.
func FilterWithCCL(ctx context.Context, path string, filterExpr string) (*insyra.DataTable, error) {
	// Compile CCL expression once
	compiledExpr, err := ccl.CompileExpression(filterExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to compile CCL expression: %w", err)
	}

	// The whole-file parts are settled before any row is kept, so a filter is
	// answered or fails before the file is read into a result. The file's shape
	// is read first: an expression that needs nothing beyond the current row
	// costs no extra pass, and a row or column the expression names is checked
	// against the file rather than against a batch.
	totalRows, colNames, _, err := cclFileInfo(path)
	if err != nil {
		return nil, err
	}

	// An expression the batches cannot answer is computed over whole columns. So
	// is a top-level sequence function the stream cannot take (a name a caller
	// registered, a column argument that does not change from row to row): the
	// table's own procedure gives the table's answer or error for those.
	var resolved ccl.CCLNode
	var seq *ccl.TopSequence
	isSequence := false
	wholeColumns := ccl.StreamRefusal(compiledExpr) != nil
	if !wholeColumns {
		resolved, err = ccl.ResolveWholeTable(compiledExpr, totalRows, colNames, appliedBatches(ctx, path, colNames, nil, 0, nil))
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate CCL expression: %w", err)
		}
		var seqErr error
		seq, isSequence, seqErr = ccl.NewTopSequence(resolved, totalRows, colNames)
		wholeColumns = seqErr != nil
	}

	// One slice per column, grown across every batch. Appending into
	// result.GetColByNumber(i) per batch threw away everything past the first
	// batch, because that method returns a copy of the column: a 2500-row file
	// filtered on a condition every row satisfies came back with 1000 rows.
	kept := make([][]any, len(colNames))
	keep := func(r cclRun, decisions []any) {
		for rowIdx, val := range decisions {
			if !passesFilter(val) {
				continue
			}
			// Add this row to the filtered results
			for colIdx := range kept {
				kept[colIdx] = append(kept[colIdx], r.cols[colIdx][rowIdx])
			}
		}
	}

	switch {
	case wholeColumns:
		// The decision of every row is computed over the whole of the columns the
		// expression reads, and the file is read again for the rows to keep, so
		// the memory held is those columns and the rows kept.
		decisions, err := wholeColumnValues(ctx, path, colNames, nil, totalRows, nil, compiledExpr)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate CCL expression: %w", err)
		}
		if err := eachRun(ctx, path, colNames, func(r cclRun) error {
			end := r.offset + r.rows()
			if end > len(decisions) {
				return fmt.Errorf("the file holds rows %d to %d, past the %d rows the filter was computed over; it changed while it was being read",
					r.offset, end-1, len(decisions))
			}
			keep(r, decisions[r.offset:end])
			return nil
		}); err != nil {
			return nil, err
		}

	case isSequence:
		// The filter is one sequence function, which reads rows the batch after
		// the current one has not arrived with, so its decisions are streamed
		// through a stage that holds the rows it has not settled yet. The
		// decisions are written into a column of their own, which is the last
		// column of every run that leaves the stage.
		stage := &sequenceStage{seq: seq, newName: filterColumnName, target: -1}
		take := func(runs []cclRun) {
			for _, run := range runs {
				keep(run, run.cols[len(colNames)])
			}
		}
		if err := eachRun(ctx, path, colNames, func(r cclRun) error {
			out, err := stage.push(r)
			if err != nil {
				// The rows are named as the range of the file they are, both ends
				// included: which of them the sequence was asked about is the
				// stage's own business, and it has not said.
				return fmt.Errorf("error evaluating CCL in rows %d to %d: %w",
					r.offset, r.offset+r.rows()-1, err)
			}
			take(out)
			return nil
		}); err != nil {
			return nil, err
		}
		// The file has ended, so the rows the sequence held back are decided now.
		tail, err := stage.flush()
		if err != nil {
			return nil, fmt.Errorf("error evaluating CCL: %w", err)
		}
		take(tail)

	default:
		// Every other filter is answered over each batch as it arrives. A value
		// that does not vary from row to row and is a slice as long as the file
		// is spread over the rows, the way a loaded table spreads it: the
		// filter keeps the rows their own element keeps.
		rowInvariant := !ccl.IsRowDependent(resolved)
		if err := eachRun(ctx, path, colNames, func(r cclRun) error {
			pqCtx := newRunContext(r)
			decisions := make([]any, r.rows())
			for rowIdx := range decisions {
				if err := pqCtx.SetRowIndex(rowIdx); err != nil {
					return fmt.Errorf("failed to set row index %d: %w", rowIdx, err)
				}
				val, err := ccl.Evaluate(resolved, pqCtx)
				if err != nil {
					return fmt.Errorf("error evaluating CCL at row %d: %w", r.offset+rowIdx, err)
				}
				decisions[rowIdx] = rowOf(val, rowInvariant, totalRows, r.offset+rowIdx)
			}
			keep(r, decisions)
			return nil
		}); err != nil {
			return nil, err
		}
	}

	// build assembles the result once, on whichever path ends the stream, so an
	// empty file and a filter that matched nothing produce the same shape.
	if len(kept[0]) == 0 {
		result := insyra.NewDataTable()
		for _, name := range colNames {
			dl := insyra.NewDataList()
			dl.SetName(name)
			result.AppendCols(dl)
		}
		return result, nil
	}
	result := insyra.NewDataTable()
	for i, name := range colNames {
		dl := insyra.NewDataList()
		dl.SetName(name)
		dl.Append(kept[i]...)
		result.AppendCols(dl)
	}
	return result, nil
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
// read a column an earlier one created. A statement with a part that cannot be
// computed batch by batch (the ones FilterWithCCL describes: MEDIAN, a
// registered aggregate, a row computed from the current one, a sequence function
// inside a larger expression, a column range inside an aggregate's argument) is
// computed first, by reading the whole of the columns it reads and only those
// and evaluating it the way the DataTable CCL methods do; its column is held in
// memory until the file is written, and its answer, or its error, is the loaded
// table's. All of that happens before anything is written. It is written back
// with the codec each column had and row groups as large as the original's
// largest one, and a column the script adds takes the first column's codec. One
// opts replaces both, as Write uses it. The new file goes to a temporary file of
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
	totalRows, colNames, fileSchema, err := cclFileInfo(path)
	if err != nil {
		return err
	}

	// precomputed holds the column of each statement that was computed over
	// whole columns, by the statement's position: the pipeline hands on that
	// column instead of computing the statement, and the statements after it read
	// it as the file's own.
	precomputed := make(map[int][]any)
	resolved := make([]ccl.CCLNode, 0, len(compiledNodes))
	names := slices.Clone(colNames)
	for _, node := range compiledNodes {
		if _, _, isNew := ccl.GetNewColInfo(node); !isNew {
			if _, isAssignment := ccl.GetAssignmentTarget(node); !isAssignment {
				// A statement that is neither a NEW nor an assignment writes
				// nothing, the way ExecuteCCL leaves it alone, so there is no
				// result to resolve or to compute: either would read the file
				// for a part the loaded table never evaluates.
				resolved = append(resolved, node)
				continue
			}
		}

		// A statement the batches cannot answer is computed over whole columns. So
		// is one whose right-hand side is a sequence function the stream cannot
		// take (a name a caller registered, a column argument that does not change
		// from row to row): the table's own procedure gives the table's answer or
		// error for those.
		position := len(resolved)
		r := node
		wholeColumns := ccl.StreamRefusal(node) != nil
		if !wholeColumns {
			var err error
			r, err = ccl.ResolveWholeTable(node, totalRows, names, appliedBatches(ctx, path, colNames, slices.Clone(resolved), totalRows, precomputed))
			if err != nil {
				return fmt.Errorf("failed to apply CCL: %w", err)
			}
			_, _, seqErr := ccl.NewTopSequence(ccl.GetExpressionNode(r), totalRows, names)
			wholeColumns = seqErr != nil
		}
		if wholeColumns {
			// The original node stands for the statement from here on: nothing of
			// it was resolved, and the column is the one the table would write.
			r = node
			values, err := wholeColumnValues(ctx, path, colNames, resolved, totalRows, precomputed, node)
			if err != nil {
				return fmt.Errorf("failed to apply CCL: %w", err)
			}
			precomputed[position] = values
		}
		resolved = append(resolved, r)
		if newName, _, isNew := ccl.GetNewColInfo(node); isNew {
			names = append(names, newName)
		}
	}

	// The file is written from the values the script gives the columns it
	// writes. Those values are settled from the whole file rather than from the
	// first batch, because a column can hold one kind of value in the first rows
	// and another further on: a number in the rows a look-ahead has an answer
	// for and missing values before them, a whole number where a later row
	// divides into a fraction, or a word beside a number. The first pass settles
	// each type from the values the column has held up to the point every column
	// has shown one, and says so when later values ask for another type; the
	// file it was writing is thrown away untouched, the types are settled from
	// every value, and the file is written once more.
	err = writeApplied(ctx, path, resolved, totalRows, colNames, fileSchema, layout, nil, precomputed)
	if errors.Is(err, errNothingToWrite) {
		return nil
	}
	if !errors.Is(err, errRetype) {
		return err
	}

	writtenTypes, err := writtenColumnTypes(ctx, path, resolved, totalRows, colNames, fileSchema, precomputed)
	if err != nil {
		return err
	}
	if err := writeApplied(ctx, path, resolved, totalRows, colNames, fileSchema, layout, writtenTypes, precomputed); err != nil {
		if errors.Is(err, errNothingToWrite) {
			return nil
		}
		return err
	}
	return nil
}

// errRetype says a column the script writes holds a value of another type than
// the schema it was written with, which is what the type of a column settled
// from the first rows looks like once later rows hold another kind of value. It
// is not a failure of the script: the file has to be read again to settle the
// type from every value, and written once more.
var errRetype = errors.New("parquet: a written column holds a value of another type")

// writeApplied runs the resolved statements over the file at path and writes the
// result back to it, with writtenTypes as the type of each column the script
// writes. Written types of nil let the values settle them, which is what the first
// pass does: see appliedWriter. A column that later turns out to hold a value of
// another type is errRetype rather than a file holding values of a type they are
// not.
//
// A statement in precomputed was computed over whole columns, and its column is
// written from there.
//
// The new file goes to a temporary file of its own and replaces path only when
// it is complete, so a failure at any point leaves the original as it was, and
// an input with no rows leaves it untouched too.
func writeApplied(ctx context.Context, path string, resolved []ccl.CCLNode, totalRows int, colNames []string, fileSchema *arrow.Schema, layout cclOutputLayout, writtenTypes map[string]arrow.DataType, precomputed map[int][]any) error {
	return utils.WriteFileAtomically(path, func(w io.Writer) error {
		// Each statement is a stage of the pipeline, so the rows a stage finishes
		// go on to the next statement and only the rows leaving the last one are
		// written. A stage is built for this pass, because a stage holds what it
		// has not finished.
		pipeline, err := newPipeline(resolved, totalRows, colNames, precomputed)
		if err != nil {
			return fmt.Errorf("failed to apply CCL: %w", err)
		}
		out := newAppliedWriter(w, writtenColumns(resolved, colNames), writtenTypes, fileSchema, layout)

		// Stream through the input file. It is safe to replace it afterwards
		// because the output goes to a temporary file of its own.
		if err := eachRun(ctx, path, colNames, func(r cclRun) error {
			// Run the rows through every statement, in order, and write what the
			// last one finished.
			runs, err := pipeline.push(r)
			if err != nil {
				return fmt.Errorf("failed to apply CCL: %w", err)
			}
			for _, run := range runs {
				if err := out.add(run); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}

		// The file has ended, so the rows a stage held back leave the pipeline
		// now; skipping them would drop them from the output.
		runs, err := pipeline.flush()
		if err != nil {
			return err
		}
		for _, run := range runs {
			if err := out.add(run); err != nil {
				return err
			}
		}
		return out.finish()
	})
}

// appliedWriter writes the runs that leave ApplyCCL's pipeline into one parquet
// file, and settles the type of each column the script writes from the values it
// has held.
//
// A column's type cannot be read off the first rows alone: a look-ahead leaves a
// column missing until its window is full, and a value of another kind can come
// at any row. So the runs wait, as Go values, only until every column the script
// writes has shown a value, a row group's worth of rows has arrived, or the file
// has ended; the types are then settled from everything held, the writer is
// created and the waiting runs are written. From then on a run is turned into a
// record and written as it arrives, and the writer fills its row groups by
// itself. What every later run holds is added to what the columns have held
// from the start, and a column whose type that changes is errRetype: nothing
// already written can change, so the file is thrown away and written again with
// the types the whole file settles. A run that holds only missing values adds
// nothing, so it never asks for that.
type appliedWriter struct {
	w            io.Writer
	written      []string
	writtenTypes map[string]arrow.DataType
	fileSchema   *arrow.Schema
	layout       cclOutputLayout

	// kinds is what each column whose type is settled from its values has held,
	// from the file's first row to the last one added: every column the script
	// writes, and one no statement named and the file did not have. Not read when
	// writtenTypes settled the types already.
	kinds map[string]*columnKinds

	// staged is the runs waiting for the types to be settled, and stagedRows how
	// many rows they hold.
	staged     []cclRun
	stagedRows int

	// schema is the file's schema, settled once the first run is written, and
	// writer the file writer made from it.
	schema *arrow.Schema
	writer *pqarrow.FileWriter
}

// newAppliedWriter returns a writer of the file at w. written is the columns the
// script writes, and writtenTypes the types the whole file settled for them, or
// nil when the first rows are to settle them.
func newAppliedWriter(w io.Writer, written []string, writtenTypes map[string]arrow.DataType, fileSchema *arrow.Schema, layout cclOutputLayout) *appliedWriter {
	kinds := make(map[string]*columnKinds, len(written))
	for _, name := range written {
		kinds[name] = &columnKinds{}
	}
	return &appliedWriter{
		w:            w,
		written:      written,
		writtenTypes: writtenTypes,
		fileSchema:   fileSchema,
		layout:       layout,
		kinds:        kinds,
	}
}

// observe adds the values of one run to what each column whose type is settled
// from its values has held. A column the script does not write and the file has
// is not one of them: it keeps the field the file gave it.
func (a *appliedWriter) observe(values map[string][]any) {
	for name, col := range values {
		k, tracked := a.kinds[name]
		if !tracked {
			if _, inFile := originalField(a.fileSchema, name); inFile {
				continue
			}
			k = &columnKinds{}
			a.kinds[name] = k
		}
		k.add(col)
	}
}

// ready reports whether the runs waiting can be written: the types are settled
// already, or every column the script writes has held a value, or the waiting
// rows fill a row group, which is as much as the writer holds anyway.
func (a *appliedWriter) ready() bool {
	if a.writtenTypes != nil || int64(a.stagedRows) >= a.layout.rowGroupSize {
		return true
	}
	for _, name := range a.written {
		if holdsNoValue(a.kinds[name]) {
			return false
		}
	}
	return true
}

// add takes the next run leaving the pipeline. A run with no rows has nothing to
// write.
func (a *appliedWriter) add(r cclRun) error {
	if r.rows() == 0 {
		return nil
	}
	values := runValues(r)

	if a.schema == nil {
		a.observe(values)
		a.staged = append(a.staged, r)
		a.stagedRows += r.rows()
		if !a.ready() {
			return nil
		}
		return a.settle()
	}

	// A type that was settled from every value of the file has nothing left to
	// compare; the values of this run are part of what settled it.
	if a.writtenTypes == nil {
		a.observe(values)
		if err := agreesWithOutputSchema(r.names, a.written, a.kinds, a.schema, a.fileSchema); err != nil {
			return err
		}
	}
	return a.write(r, values)
}

// settle settles the file's schema from what the waiting runs hold and writes
// them, one run at a time.
func (a *appliedWriter) settle() error {
	staged := a.staged
	a.staged, a.stagedRows = nil, 0
	a.schema = cclOutputSchema(staged[0].names, a.written, a.writtenTypes, a.kinds, a.fileSchema)
	for _, r := range staged {
		if err := a.write(r, runValues(r)); err != nil {
			return err
		}
	}
	return nil
}

// write turns one run into a record of the settled schema and writes it. The
// writer is created from the first record, so every later one is written as the
// same columns in the same order.
func (a *appliedWriter) write(r cclRun, values map[string][]any) error {
	rec, err := buildArrowRecord(values, r.names, runArrays(r), a.schema)
	if err != nil {
		return fmt.Errorf("failed to apply CCL: %w", err)
	}
	defer rec.Release()

	if a.writer == nil {
		props := []parquet.WriterProperty{
			parquet.WithCreatedBy(fmt.Sprintf("go-insyra v%s", insyra.Version)),
			parquet.WithCompression(a.layout.defaultCodec),
			parquet.WithMaxRowGroupLength(a.layout.rowGroupSize),
		}
		for name, codec := range a.layout.codecs {
			props = append(props, parquet.WithCompressionFor(name, codec))
		}
		// The Arrow schema is stored in the file, which is the only place a column's
		// time zone lives: parquet itself knows only whether an instant is in UTC.
		// Without it every zoned column the file had would come back as UTC.
		a.writer, err = pqarrow.NewFileWriter(
			rec.Schema(),
			writerOnly{a.w},
			parquet.NewWriterProperties(props...),
			pqarrow.NewArrowWriterProperties(pqarrow.WithStoreSchema()),
		)
		if err != nil {
			return fmt.Errorf("failed to create parquet writer: %w", err)
		}
	}

	// Buffered, so a row group keeps filling across records until it holds
	// layout.rowGroupSize rows.
	if err := a.writer.WriteBuffered(rec); err != nil {
		return fmt.Errorf("failed to write batch: %w", err)
	}
	return nil
}

// finish writes what still waits, now that the file has ended, and closes the
// file. An input with no rows at all is errNothingToWrite.
func (a *appliedWriter) finish() error {
	if a.schema == nil && len(a.staged) > 0 {
		if err := a.settle(); err != nil {
			return err
		}
	}
	if a.writer == nil {
		// No rows at all (the input is empty): keep the original rather than
		// replace it with an empty file.
		return errNothingToWrite
	}
	// writer wraps w in writerOnly, so closing it writes the footer and leaves
	// the file for WriteFileAtomically to close and rename.
	if err := a.writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}
	return nil
}

// agreesWithOutputSchema reports whether every column the script writes still
// asks for the type it was written with, given everything it has held since the
// file's first row, and errRetype for the first one that does not.
//
// kinds is cumulative, so a run that holds only missing values changes nothing
// and a run that holds another kind of value changes the type it asks for. A
// column whose type is the same after every run has had that type for every
// value in the file. fileSchema is what the type of a column is settled against,
// so this settles it the way cclOutputSchema did.
func agreesWithOutputSchema(colNames []string, written []string, kinds map[string]*columnKinds, schema, fileSchema *arrow.Schema) error {
	for _, name := range written {
		idx := slices.Index(colNames, name)
		if idx < 0 {
			continue
		}
		original, _ := originalField(fileSchema, name)
		got := writtenType(original.Type, kinds[name])
		fieldType := schema.Field(idx).Type
		if !arrow.TypeEqual(got, fieldType) {
			return fmt.Errorf("%w: column %q is %s, written as %s", errRetype, name, got, fieldType)
		}
	}
	return nil
}

// writtenColumnTypesCalls counts the times the types of the written columns had to
// be settled from the whole file, which is a second read of it. A file whose types
// the first pass settles correctly never needs one, and the tests hold ApplyCCL to
// that.
var writtenColumnTypesCalls atomic.Int64

// writtenColumnTypes settles the type of every column the script writes from the
// values it gives it over the whole file, which is what Write would infer from
// the table the script leaves — except that a column the file had and whose every
// value its own type holds keeps that type, which is writtenType's rule again.
// A statement in precomputed was computed over whole columns, and its column is
// taken from there. Nothing is written, so a file whose values cannot be computed
// leaves it as it was.
func writtenColumnTypes(ctx context.Context, path string, resolved []ccl.CCLNode, totalRows int, colNames []string, fileSchema *arrow.Schema, precomputed map[int][]any) (map[string]arrow.DataType, error) {
	writtenColumnTypesCalls.Add(1)
	written := writtenColumns(resolved, colNames)
	if len(written) == 0 {
		return nil, nil
	}
	kinds := make(map[string]*columnKinds, len(written))
	for _, name := range written {
		kinds[name] = &columnKinds{}
	}

	pipeline, err := newPipeline(resolved, totalRows, colNames, precomputed)
	if err != nil {
		return nil, fmt.Errorf("failed to apply CCL: %w", err)
	}

	// add records what one record's rows are for every column the script writes,
	// and reports whether every one of those columns is in it.
	add := func(names []string, values map[string][]any) {
		for _, name := range written {
			idx := slices.Index(names, name)
			if idx < 0 {
				continue
			}
			kinds[name].add(values[name])
		}
	}
	// There are no stages to answer to here, so a value is a missing value; what
	// is settled is which kinds of value the script gives the column, and a nil
	// is one no batch holds.
	hand := func(runs []cclRun) {
		for _, run := range runs {
			add(run.names, runValues(run))
		}
	}

	if err := eachRun(ctx, path, colNames, func(r cclRun) error {
		runs, err := pipeline.push(r)
		if err != nil {
			return fmt.Errorf("failed to apply CCL: %w", err)
		}
		hand(runs)
		return nil
	}); err != nil {
		return nil, err
	}
	runs, err := pipeline.flush()
	if err != nil {
		return nil, err
	}
	hand(runs)

	types := make(map[string]arrow.DataType, len(written))
	for name, k := range kinds {
		original, _ := originalField(fileSchema, name)
		types[name] = writtenType(original.Type, k)
	}
	return types, nil
}
