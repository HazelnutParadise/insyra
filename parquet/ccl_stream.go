package parquet

import (
	"context"
	"errors"
	"fmt"
	"iter"

	"github.com/HazelnutParadise/insyra"
)

// StreamFilterWithCCL gives the rows of the file at path that filterExpr keeps,
// in file order, batch by batch, without holding them. Range over it:
//
//	for batch, err := range parquet.StreamFilterWithCCL(ctx, path, "A > 100") {
//		if err != nil {
//			return err
//		}
//		// write batch somewhere
//	}
//
// The rows, their values and their order are those FilterWithCCL returns,
// which is the same filter collected into one table. A batch holds the kept
// rows of one read of the file, at most 1,000 here; batch sizes are not
// promised. When the filter keeps nothing, one table with the columns and no
// rows is given, so a sink still has its header.
//
// Memory: the batch being read, and for a sequence function the rows a later
// batch is needed for. An expression that needs whole columns still holds
// them and one value per row, as ExplainFilterWithCCL reports;
// RefuseWholeColumns refuses it. opts work as they do for FilterWithCCL; with
// MaxRows, the rows within the limit are given before the error matching
// ErrCCLRowLimit.
//
// A failure arrives once, as a nil table and the error, and ends the sequence.
// One after the first batch, such as an evaluation error at a later row or a
// cancelled ctx, comes after the earlier batches were given; what a sink
// already wrote is the caller's to keep or discard. Leaving the loop early
// stops the read and closes the file.
func StreamFilterWithCCL(ctx context.Context, path string, filterExpr string, opts ...FilterWithCCLOptions) iter.Seq2[*insyra.DataTable, error] {
	return func(yield func(*insyra.DataTable, error) bool) {
		opt, err := filterOptions(opts)
		if err != nil {
			yield(nil, err)
			return
		}
		given, stopped := false, false
		names, err := filterKept(ctx, path, filterExpr, opt, func(names []string, cols [][]any) error {
			given = true
			if !yield(tableOf(names, cols), nil) {
				stopped = true
				return errStreamStopped
			}
			return nil
		})
		if stopped {
			return
		}
		if err != nil {
			yield(nil, err)
			return
		}
		if !given {
			yield(tableOf(names, nil), nil)
		}
	}
}

// errStreamStopped is what the emit of StreamFilterWithCCL returns when the
// caller left the loop, which stops the read.
var errStreamStopped = errors.New("parquet: the stream was stopped")

// tableOf builds a table whose columns are names, holding cols; nil cols give
// a table with the columns and no rows.
func tableOf(names []string, cols [][]any) *insyra.DataTable {
	result := insyra.NewDataTable()
	for i, name := range names {
		dl := insyra.NewDataList()
		dl.SetName(name)
		if i < len(cols) {
			dl.Append(cols[i]...)
		}
		result.AppendCols(dl)
	}
	return result
}

// projectColumns returns the names and positions of the result's columns:
// want, in its order, or every column when want is empty.
func projectColumns(colNames []string, want []string) ([]string, []int, error) {
	if len(want) == 0 {
		proj := make([]int, len(colNames))
		for i := range proj {
			proj[i] = i
		}
		return colNames, proj, nil
	}
	index := make(map[string]int, len(colNames))
	for i, name := range colNames {
		if _, taken := index[name]; !taken {
			index[name] = i
		}
	}
	proj := make([]int, len(want))
	seen := make(map[string]bool, len(want))
	for i, name := range want {
		pos, ok := index[name]
		if !ok {
			return nil, nil, fmt.Errorf("FilterWithCCLOptions.Columns: column %s not found", name)
		}
		if seen[name] {
			return nil, nil, fmt.Errorf("FilterWithCCLOptions.Columns: column %s is given twice", name)
		}
		seen[name] = true
		proj[i] = pos
	}
	return append([]string(nil), want...), proj, nil
}
