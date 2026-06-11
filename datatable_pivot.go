package insyra

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/HazelnutParadise/insyra/internal/algorithms"
)

// PivotConfig describes a long-to-wide reshape produced by (*DataTable).Pivot.
//
// Given a long-form table, Pivot keeps the rows identified by Index, spreads
// the unique values of Columns into new column headers, and fills the new
// cells with values drawn from the Values column. When the same (Index,
// Columns) combination appears in more than one input row, the duplicate
// values are reduced via AggFunc.
//
// Column reference resolution: every column-name field below (Index, Columns,
// Values) is matched against column.name first; if no column has that name,
// it falls back to the Excel-style alphabetic index ("A" → column 0,
// "B" → column 1, ..., "AA" → column 26). The first row of data is never
// consulted. Tokens that match neither a name nor a valid alphabetic index
// produce an error.
type PivotConfig struct {
	// Index lists the columns kept as identifiers; their unique combinations
	// form the row keys of the output. At least one entry is required. Each
	// entry is resolved by name first, then as an Excel-style index.
	Index []string

	// Columns names the column whose unique values become new column headers
	// in the output. Required. Resolved by name first, then as an Excel-style
	// index.
	Columns string

	// Values names the column whose cell values fill the new (Index, Columns)
	// cells. Required. Resolved by name first, then as an Excel-style index.
	Values string

	// AggFunc is the aggregator applied when an (Index, Columns) combination
	// occurs more than once. Recognised names: "sum", "mean" (alias "avg"),
	// "median", "min", "max", "count" (non-nil), "countall" (group size),
	// "stdev" (alias "std"), "stdevp" (alias "stdp"), "var", "varp", "first",
	// "last", "nunique", "custom" (requires Custom). When empty, Pivot
	// returns an error if any (Index, Columns) combination has duplicates.
	AggFunc string

	// Custom is required when AggFunc == "custom". It receives the values
	// belonging to the (Index, Columns) cell as a *DataList in original row
	// order, including nil entries.
	Custom func(group *DataList) any

	// FillNA fills cells where no input row matched the (Index, Columns)
	// combination. Default nil.
	FillNA any

	// SortCols controls whether generated columns are emitted in sorted
	// order of their key value (true) or first-seen order (false, default).
	SortCols bool
}

// UnpivotConfig describes a wide-to-long reshape produced by (*DataTable).Unpivot.
//
// Column reference resolution: every column-name field below (IDVars,
// ValueVars) is matched against column.name first; if no column has that
// name, it falls back to the Excel-style alphabetic index ("A" → column 0,
// "B" → column 1, ..., "AA" → column 26). The first row of data is never
// consulted. Tokens that match neither a name nor a valid alphabetic index
// produce an error.
type UnpivotConfig struct {
	// IDVars lists the columns kept as-is (identifier columns). Each entry
	// is resolved by name first, then as an Excel-style index.
	IDVars []string

	// ValueVars lists the columns to unpivot. Each entry is resolved by name
	// first, then as an Excel-style index. When empty, all columns not
	// listed in IDVars are unpivoted.
	ValueVars []string

	// VarName is the name of the new "variable" column in the output. When
	// empty it defaults to "variable".
	VarName string

	// ValueName is the name of the new "value" column in the output. When
	// empty it defaults to "value".
	ValueName string

	// DropNA, when true, omits rows whose value is nil or NaN.
	DropNA bool
}

// Pivot reshapes long-form data into wide form. Each unique combination of
// the Index columns becomes a row in the result; each unique value of the
// Columns column becomes a new column header; cells are filled from the
// Values column. When an (Index, Columns) pair is not unique, AggFunc must
// be set; missing combinations are filled with FillNA.
//
// The returned *DataTable is a new instance and shares no backing storage
// with the receiver. The receiver itself is not modified.
func (dt *DataTable) Pivot(cfg PivotConfig) (*DataTable, error) {
	out := NewDataTable()

	if len(cfg.Index) == 0 {
		return failPivot(out, "Index requires at least one column")
	}
	if strings.TrimSpace(cfg.Columns) == "" {
		return failPivot(out, "Columns is required")
	}
	if strings.TrimSpace(cfg.Values) == "" {
		return failPivot(out, "Values is required")
	}

	var aggOp AggregateOp
	aggSet := false
	if cfg.AggFunc != "" {
		op, ok := parseAggOpName(cfg.AggFunc)
		if !ok {
			return failPivot(out, fmt.Sprintf("unknown AggFunc %q", cfg.AggFunc))
		}
		aggOp = op
		aggSet = true
		if aggOp == OpCustom && cfg.Custom == nil {
			return failPivot(out, fmt.Sprintf("AggFunc %q requires Custom func", cfg.AggFunc))
		}
	}

	var errMsg string
	dt.AtomicDo(func(t *DataTable) {
		indexNums := make([]int, 0, len(cfg.Index))
		indexLabels := make([]string, 0, len(cfg.Index))
		for _, name := range cfg.Index {
			num, label, ok := resolveColForGroup(t, name)
			if !ok {
				errMsg = fmt.Sprintf("index column %q not found", name)
				return
			}
			indexNums = append(indexNums, num)
			indexLabels = append(indexLabels, label)
		}
		colsNum, _, ok := resolveColForGroup(t, cfg.Columns)
		if !ok {
			errMsg = fmt.Sprintf("columns column %q not found", cfg.Columns)
			return
		}
		valsNum, _, ok := resolveColForGroup(t, cfg.Values)
		if !ok {
			errMsg = fmt.Sprintf("values column %q not found", cfg.Values)
			return
		}

		// Reject overlap between Index and Columns/Values to keep the output
		// well-defined.
		for _, n := range indexNums {
			if n == colsNum {
				errMsg = fmt.Sprintf("column %q appears in both Index and Columns", cfg.Columns)
				return
			}
			if n == valsNum {
				errMsg = fmt.Sprintf("column %q appears in both Index and Values", cfg.Values)
				return
			}
		}

		nRows := 0
		for _, c := range t.columns {
			if l := len(c.data); l > nRows {
				nRows = l
			}
		}

		type indexInfo struct {
			values []any
			order  int
		}
		type colInfo struct {
			value any
			label string
			order int
		}

		indexOrder := []string{}
		indexInfos := map[string]*indexInfo{}
		colOrder := []string{}
		colInfos := map[string]*colInfo{}

		// (indexEnc, colEnc) -> []rowIdx
		type cellKey struct {
			idx string
			col string
		}
		cellRows := map[cellKey][]int{}

		valSourceCol := t.columns[valsNum]
		colsSourceCol := t.columns[colsNum]

		for row := 0; row < nRows; row++ {
			indexVals := make([]any, len(indexNums))
			for i, ci := range indexNums {
				col := t.columns[ci]
				if row < len(col.data) {
					indexVals[i] = col.data[row]
				}
			}
			indexEnc := encodeGroupKey(indexVals)
			if _, exists := indexInfos[indexEnc]; !exists {
				indexInfos[indexEnc] = &indexInfo{
					values: indexVals,
					order:  len(indexOrder),
				}
				indexOrder = append(indexOrder, indexEnc)
			}

			var colVal any
			if row < len(colsSourceCol.data) {
				colVal = colsSourceCol.data[row]
			}
			colEnc := encodeGroupKey([]any{colVal})
			if _, exists := colInfos[colEnc]; !exists {
				colInfos[colEnc] = &colInfo{
					value: colVal,
					label: pivotColLabel(colVal),
					order: len(colOrder),
				}
				colOrder = append(colOrder, colEnc)
			}

			ck := cellKey{indexEnc, colEnc}
			cellRows[ck] = append(cellRows[ck], row)
		}

		if !aggSet {
			for _, rs := range cellRows {
				if len(rs) > 1 {
					errMsg = "duplicate (Index, Columns) combinations found and AggFunc is empty"
					return
				}
			}
		}

		finalColOrder := append([]string(nil), colOrder...)
		if cfg.SortCols {
			sort.SliceStable(finalColOrder, func(i, j int) bool {
				return algorithms.CompareAny(colInfos[finalColOrder[i]].value, colInfos[finalColOrder[j]].value) < 0
			})
		}

		// Detect collisions: two distinct column key values that produce the
		// same display label (e.g. nil vs the literal string "<nil>"). Bail
		// rather than silently merge.
		seenLabels := map[string]string{}
		for _, enc := range finalColOrder {
			label := colInfos[enc].label
			if prev, ok := seenLabels[label]; ok && prev != enc {
				errMsg = fmt.Sprintf("distinct Columns values produce duplicate output column name %q", label)
				return
			}
			seenLabels[label] = enc
		}

		outIndexCols := make([]*DataList, len(indexLabels))
		for i, label := range indexLabels {
			outIndexCols[i] = NewDataList()
			outIndexCols[i].SetName(label)
		}
		outValCols := make([]*DataList, len(finalColOrder))
		for j, enc := range finalColOrder {
			outValCols[j] = NewDataList()
			outValCols[j].SetName(colInfos[enc].label)
		}

		for _, indexEnc := range indexOrder {
			inf := indexInfos[indexEnc]
			for k, v := range inf.values {
				outIndexCols[k].Append(v)
			}
			for j, colEnc := range finalColOrder {
				rs, exists := cellRows[cellKey{indexEnc, colEnc}]
				if !exists || len(rs) == 0 {
					outValCols[j].Append(cfg.FillNA)
					continue
				}
				if !aggSet {
					var v any
					idx := rs[0]
					if idx < len(valSourceCol.data) {
						v = valSourceCol.data[idx]
					}
					if v == nil {
						outValCols[j].Append(cfg.FillNA)
					} else {
						outValCols[j].Append(v)
					}
					continue
				}
				sub := NewDataList()
				for _, idx := range rs {
					if idx < len(valSourceCol.data) {
						sub.Append(valSourceCol.data[idx])
					} else {
						sub.Append(nil)
					}
				}
				v := applyAggregateOp(aggOp, sub, cfg.Custom, len(rs))
				if v == nil {
					outValCols[j].Append(cfg.FillNA)
				} else {
					outValCols[j].Append(v)
				}
			}
		}

		out.AppendCols(outIndexCols...)
		out.AppendCols(outValCols...)
	})

	if errMsg != "" {
		return failPivot(out, errMsg)
	}
	return out, nil
}

// Unpivot reshapes wide-form data into long form. Each input row is expanded
// into one output row per ValueVar, with VarName recording the source column
// name and ValueName recording the cell value. IDVars are copied unchanged.
//
// The returned *DataTable is a new instance and shares no backing storage
// with the receiver. The receiver itself is not modified.
func (dt *DataTable) Unpivot(cfg UnpivotConfig) (*DataTable, error) {
	out := NewDataTable()

	varName := cfg.VarName
	if varName == "" {
		varName = "variable"
	}
	valueName := cfg.ValueName
	if valueName == "" {
		valueName = "value"
	}
	if varName == valueName {
		return failUnpivot(out, fmt.Sprintf("VarName and ValueName must differ (got %q)", varName))
	}

	var errMsg string
	dt.AtomicDo(func(t *DataTable) {
		idvarNums := make([]int, 0, len(cfg.IDVars))
		idvarLabels := make([]string, 0, len(cfg.IDVars))
		idvarSet := map[int]struct{}{}
		for _, name := range cfg.IDVars {
			num, label, ok := resolveColForGroup(t, name)
			if !ok {
				errMsg = fmt.Sprintf("id var column %q not found", name)
				return
			}
			if _, dup := idvarSet[num]; dup {
				errMsg = fmt.Sprintf("id var column %q listed more than once", name)
				return
			}
			idvarSet[num] = struct{}{}
			idvarNums = append(idvarNums, num)
			idvarLabels = append(idvarLabels, label)
		}

		var valueNums []int
		var valueLabels []string
		if len(cfg.ValueVars) == 0 {
			for i, c := range t.columns {
				if _, isID := idvarSet[i]; isID {
					continue
				}
				label := c.name
				if label == "" {
					label = pivotColLabelForIndex(i)
				}
				valueNums = append(valueNums, i)
				valueLabels = append(valueLabels, label)
			}
		} else {
			seen := map[int]struct{}{}
			for _, name := range cfg.ValueVars {
				num, label, ok := resolveColForGroup(t, name)
				if !ok {
					errMsg = fmt.Sprintf("value var column %q not found", name)
					return
				}
				if _, isID := idvarSet[num]; isID {
					errMsg = fmt.Sprintf("column %q appears in both IDVars and ValueVars", name)
					return
				}
				if _, dup := seen[num]; dup {
					errMsg = fmt.Sprintf("value var column %q listed more than once", name)
					return
				}
				seen[num] = struct{}{}
				valueNums = append(valueNums, num)
				valueLabels = append(valueLabels, label)
			}
		}

		if len(valueNums) == 0 {
			// Build output schema even when there are no value columns: emit
			// IDVars + var/value columns, no rows.
			idCols := make([]*DataList, len(idvarLabels))
			for i, label := range idvarLabels {
				idCols[i] = NewDataList()
				idCols[i].SetName(label)
			}
			vCol := NewDataList()
			vCol.SetName(varName)
			valCol := NewDataList()
			valCol.SetName(valueName)
			out.AppendCols(idCols...)
			out.AppendCols(vCol, valCol)
			return
		}

		nRows := 0
		for _, c := range t.columns {
			if l := len(c.data); l > nRows {
				nRows = l
			}
		}

		idCols := make([]*DataList, len(idvarLabels))
		for i, label := range idvarLabels {
			idCols[i] = NewDataList()
			idCols[i].SetName(label)
		}
		vCol := NewDataList()
		vCol.SetName(varName)
		valCol := NewDataList()
		valCol.SetName(valueName)

		for row := 0; row < nRows; row++ {
			for j, num := range valueNums {
				var v any
				src := t.columns[num]
				if row < len(src.data) {
					v = src.data[row]
				}
				if cfg.DropNA && isNilOrNaN(v) {
					continue
				}
				for k, ci := range idvarNums {
					col := t.columns[ci]
					if row < len(col.data) {
						idCols[k].Append(col.data[row])
					} else {
						idCols[k].Append(nil)
					}
				}
				vCol.Append(valueLabels[j])
				valCol.Append(v)
			}
		}

		out.AppendCols(idCols...)
		out.AppendCols(vCol, valCol)
	})

	if errMsg != "" {
		return failUnpivot(out, errMsg)
	}
	return out, nil
}

// failPivot records msg as an error on out (so chained callers can see it via
// out.Err()) and returns it alongside the matching error value.
func failPivot(out *DataTable, msg string) (*DataTable, error) {
	out.setError(LogLevelWarning, "DataTable", "Pivot", msg)
	return out, fmt.Errorf("Pivot: %s", msg)
}

// failUnpivot is the Unpivot counterpart to failPivot.
func failUnpivot(out *DataTable, msg string) (*DataTable, error) {
	out.setError(LogLevelWarning, "DataTable", "Unpivot", msg)
	return out, fmt.Errorf("Unpivot: %s", msg)
}

// applyAggregateOp computes a single aggregation value over the values
// collected for a group. groupSize is the number of input rows that fed
// the group (used by OpCountAll). Returns nil for unrecognised ops.
func applyAggregateOp(op AggregateOp, sub *DataList, custom func(*DataList) any, groupSize int) any {
	switch op {
	case OpCountAll:
		return groupSize
	case OpSum:
		return sub.Sum()
	case OpMean:
		return sub.Mean()
	case OpMedian:
		return sub.Median()
	case OpMin:
		return sub.Min()
	case OpMax:
		return sub.Max()
	case OpStdev:
		return sub.Stdev()
	case OpStdevP:
		return sub.StdevP()
	case OpVar:
		return sub.Var()
	case OpVarP:
		return sub.VarP()
	case OpCount:
		count := 0
		for _, v := range sub.data {
			if v != nil {
				count++
			}
		}
		return count
	case OpFirst:
		for _, v := range sub.data {
			if v != nil {
				return v
			}
		}
		return nil
	case OpLast:
		for i := len(sub.data) - 1; i >= 0; i-- {
			if sub.data[i] != nil {
				return sub.data[i]
			}
		}
		return nil
	case OpNUnique:
		return countUniqueNonNil(sub.data)
	case OpCustom:
		if custom == nil {
			return nil
		}
		return custom(sub)
	}
	return nil
}

// parseAggOpName resolves a canonical aggregate name (and common aliases)
// to an AggregateOp. Returns (OpSum, false) for unrecognised names.
func parseAggOpName(s string) (AggregateOp, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "sum":
		return OpSum, true
	case "mean", "avg", "average":
		return OpMean, true
	case "median":
		return OpMedian, true
	case "min":
		return OpMin, true
	case "max":
		return OpMax, true
	case "count":
		return OpCount, true
	case "countall":
		return OpCountAll, true
	case "std", "stdev", "stddev":
		return OpStdev, true
	case "stdp", "stdevp", "stddevp":
		return OpStdevP, true
	case "var", "variance":
		return OpVar, true
	case "varp":
		return OpVarP, true
	case "first":
		return OpFirst, true
	case "last":
		return OpLast, true
	case "nunique":
		return OpNUnique, true
	case "custom":
		return OpCustom, true
	}
	return OpSum, false
}

// pivotColLabel renders a Columns key value as the new output column name.
func pivotColLabel(v any) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", v)
}

// pivotColLabelForIndex returns a stable fallback label for an unnamed
// column at the given numeric index. Used by Unpivot when ValueVars is
// empty and the source column has no name.
func pivotColLabelForIndex(i int) string {
	// Mirror the alphabetic indexing used elsewhere; if conversion fails
	// (shouldn't, in practice), fall back to a numeric tag.
	if name, ok := alphaColIndex(i); ok {
		return name
	}
	return fmt.Sprintf("col_%d", i)
}

// alphaColIndex computes the Excel-style label for a numeric column index.
func alphaColIndex(i int) (string, bool) {
	if i < 0 {
		return "", false
	}
	name := ""
	for i >= 0 {
		name = string(rune('A'+(i%26))) + name
		i = i/26 - 1
	}
	return name, true
}

// isNilOrNaN reports whether v is nil or a NaN float.
func isNilOrNaN(v any) bool {
	if v == nil {
		return true
	}
	switch f := v.(type) {
	case float32:
		return math.IsNaN(float64(f))
	case float64:
		return math.IsNaN(f)
	}
	return false
}
