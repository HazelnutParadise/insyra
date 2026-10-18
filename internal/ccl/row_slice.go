package ccl

import (
	"fmt"
	"strings"
)

// resolveResultValue turns the evaluator's internal range representations into
// something a cell can hold. A column range left as the whole answer means the
// current row restricted to those columns, the same defaulting that makes a
// bare `A` mean `A.#` and a bare `@` mean `@.#`, so `A:B` yields that row's A
// and B values.
//
// A row range has no such reading: it says which rows, but not of what, so it
// is only meaningful attached to a column.
func resolveResultValue(val any, ctx Context) (any, error) {
	switch v := val.(type) {
	case ColumnRange:
		return rowSliceAt(v, ctx.GetRowIndex(), ctx)
	case RowRange:
		return nil, fmt.Errorf("a row range says which rows but not of what: attach it to a column, as in A.(%d:%d)", v.Start, v.End)
	}
	return val, nil
}

// rowSliceAt returns row rowIdx restricted to the columns in cr.
func rowSliceAt(cr ColumnRange, rowIdx int, ctx Context) ([]any, error) {
	out := make([]any, 0, cr.End-cr.Start+1)
	for c := cr.Start; c <= cr.End; c++ {
		cell, err := ctx.GetCell(c, rowIdx)
		if err != nil {
			return nil, err
		}
		out = append(out, cell)
	}
	return out, nil
}

// rowShapedColumn builds the per-row column for an argument that stands for a
// whole row: '@' (every column) or a column range (some of them). It returns
// isRowShaped=false for anything else, leaving the caller to read it as an
// ordinary column of values.
func rowShapedColumn(arg cclNode, ctx Context, depth, callDepth int) (rows []any, isRowShaped bool, err error) {
	var cr *ColumnRange
	switch node := arg.(type) {
	case *cclAtNode:
	case *cclBinaryOpNode:
		if node.op != ":" {
			return nil, false, nil
		}
		v, err := evaluateRange(node.left, node.right, ctx, depth, callDepth)
		if err != nil {
			return nil, false, err
		}
		got, ok := v.(ColumnRange)
		if !ok {
			return nil, false, nil
		}
		cr = &got
	default:
		return nil, false, nil
	}

	// Reading a row moves the cursor on some contexts, so put it back.
	n := ctx.GetRowCount()
	if n > 0 {
		current := ctx.GetRowIndex()
		defer func() {
			if setErr := ctx.SetRowIndex(current); setErr != nil && err == nil {
				err = setErr
			}
		}()
	}

	rows = make([]any, 0, n)
	for i := 0; i < n; i++ {
		if cr == nil {
			row, err := ctx.GetRowAt(i)
			if err != nil {
				return nil, true, err
			}
			rows = append(rows, row)
			continue
		}
		slice, err := rowSliceAt(*cr, i, ctx)
		if err != nil {
			return nil, true, err
		}
		rows = append(rows, slice)
	}
	return rows, true, nil
}

// describeRowShaped names a row-shaped argument for an error message.
func describeRowShaped(arg cclNode) string {
	if _, ok := arg.(*cclAtNode); ok {
		return "'@'"
	}
	return "a column range"
}

// rowCapableSequenceFunctions lists the sequence functions that only move
// values around and never do arithmetic on them, so they can take a whole row
// per element: `LAG(@, 1)` gives every row the row before it. Every other
// sequence function needs numbers, and a row is never one. Adding a sequence
// function means deciding which of the two it is.
var rowCapableSequenceFunctions = map[string]bool{
	"LAG":  true,
	"LEAD": true,
}

// SequenceFunctionTakesRows reports whether name accepts a row-shaped column
// ('@' or a column range) instead of a column of values.
func SequenceFunctionTakesRows(name string) bool {
	return rowCapableSequenceFunctions[strings.ToUpper(name)]
}
