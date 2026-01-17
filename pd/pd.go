package pd

import (
	"fmt"

	"github.com/HazelnutParadise/insyra"
	gpd "github.com/apoplexi24/gpandas"
	gpdf "github.com/apoplexi24/gpandas/dataframe"
)

// DataFrame is a thin wrapper around gpandas.DataFrame providing helpers to convert
// to/from Insyra's DataTable and expose a pandas-like API mapped to gpandas.
type DataFrame struct {
	*gpdf.DataFrame
}

// FromDataTable converts an insyra.DataTable into a gpandas DataFrame and wraps it.
// Types are inferred per-column (int64/float64/bool/string) and fallback to Any when mixed.
func FromDataTable(dt *insyra.DataTable) (*DataFrame, error) {
	if dt == nil {
		return nil, fmt.Errorf("nil DataTable")
	}

	// Read a consistent snapshot of the DataTable under the DataTable lock.
	var numRows, numCols int
	var columnNames []string
	var columnVals [][]any
	var rowNames []string
	dt.AtomicDo(func(dt *insyra.DataTable) {
		numRows, numCols = dt.Size()
		if numCols == 0 {
			// create an empty gpandas DataFrame
			gp := gpd.GoPandas{}
			df, err := gp.DataFrame([]string{}, []gpd.Column{}, map[string]any{})
			if err != nil {
				// bubble up by assigning to outer scope err via closure; but here just return early from caller
				// note: since we are inside AtomicDo, we can't return from outer function directly. Use panic as short-circuit is messy.
				panic(err)
			}
			// store and early-return by panic carry
			panic(struct {
				df *gpdf.DataFrame
			}{df})
		}

		columnNames = make([]string, numCols)
		columnVals = make([][]any, numCols)
		for i := 0; i < numCols; i++ {
			dl := dt.GetColByNumber(i)
			name := dl.GetName()
			if name == "" {
				name = fmt.Sprintf("col%d", i)
			}
			columnNames[i] = name

			// copy values to avoid holding references to internal slices
			vals := dl.Data()
			if len(vals) == 0 {
				vals = make([]any, dl.Len())
			}
			arr := make([]any, len(vals))
			copy(arr, vals)
			columnVals[i] = arr
		}

		rowNames = dt.RowNames()
	})

	// Handle early-return path from panic used above for empty DataTable
	// When AtomicDo panics with a wrapper value, unwrap it here.
	// This is a pragmatic approach to return a created empty DataFrame error-free.
	// Note: keep this minimal and specific.
	if r := recover(); r != nil {
		switch v := r.(type) {
		case error:
			return nil, v
		case struct{ df *gpdf.DataFrame }:
			return &DataFrame{v.df}, nil
		default:
			panic(r)
		}
	}

	columns := columnNames
	data := make([]gpd.Column, numCols)
	types := make(map[string]any, numCols)

	for i := 0; i < numCols; i++ {
		name := columns[i]
		vals := columnVals[i]

		typ := inferColType(vals)
		converted := convertValuesToType(vals, typ)

		data[i] = gpd.Column(converted)
		switch typ {
		case "int":
			types[name] = gpd.IntCol{}
		case "float":
			types[name] = gpd.FloatCol{}
		case "bool":
			types[name] = gpd.BoolCol{}
		case "string":
			types[name] = gpd.StringCol{}
		default:
			// leave nil -> AnySeries fallback
			types[name] = nil
		}
	}

	gp := gpd.GoPandas{}
	df, err := gp.DataFrame(columns, data, types)
	if err != nil {
		return nil, err
	}

	// propagate row names if any
	if len(rowNames) == numRows {
		_ = df.SetIndex(rowNames)
	}

	return &DataFrame{df}, nil
}

// ToDataTable converts the wrapped gpandas DataFrame into an insyra.DataTable.
func (t *DataFrame) ToDataTable() (*insyra.DataTable, error) {
	if t == nil || t.DataFrame == nil {
		return nil, fmt.Errorf("nil Table or DataFrame")
	}

	cols := make([]*insyra.DataList, 0, len(t.DataFrame.ColumnOrder))
	for _, name := range t.DataFrame.ColumnOrder {
		series, ok := t.DataFrame.Columns[name]
		if !ok {
			return nil, fmt.Errorf("missing column %s", name)
		}
		vals := series.ValuesCopy()
		dl := insyra.NewDataList(vals...)
		dl.SetName(name)
		cols = append(cols, dl)
	}

	dt := insyra.NewDataTable(cols...)
	// set row names if DataFrame has an index
	if len(t.DataFrame.Index) > 0 {
		dt.SetRowNames(t.DataFrame.Index)
	}
	return dt, nil
}

// inferColType inspects a column's values and returns one of: "int", "float", "bool", "string", or "any".
func inferColType(vals []any) string {
	var sawInt, sawFloat, sawBool, sawString bool
	for _, v := range vals {
		if v == nil {
			continue
		}
		switch vv := v.(type) {
		case int, int8, int16, int32, int64:
			sawInt = true
		case float32, float64:
			sawFloat = true
		case bool:
			sawBool = true
		case string:
			sawString = true
		default:
			// unknown / complex types -> any
			_ = vv
			return "any"
		}
		// quick conflicts
		if sawString && (sawInt || sawFloat || sawBool) {
			return "any"
		}
		if sawBool && (sawInt || sawFloat) {
			return "any"
		}
	}
	// prefer float if any float present
	if sawFloat {
		return "float"
	}
	if sawInt {
		return "int"
	}
	if sawBool {
		return "bool"
	}
	if sawString {
		return "string"
	}
	return "any"
}

// convertValuesToType converts values into normalized Go types expected by gpandas:
// int -> int64, float -> float64, others left as-is (nil stays nil)
func convertValuesToType(vals []any, typ string) []any {
	out := make([]any, len(vals))
	switch typ {
	case "int":
		for i, v := range vals {
			if v == nil {
				out[i] = nil
				continue
			}
			switch n := v.(type) {
			case int:
				out[i] = int64(n)
			case int8:
				out[i] = int64(n)
			case int16:
				out[i] = int64(n)
			case int32:
				out[i] = int64(n)
			case int64:
				out[i] = n
			case float32:
				out[i] = int64(n)
			case float64:
				out[i] = int64(n)
			default:
				out[i] = nil
			}
		}
	case "float":
		for i, v := range vals {
			if v == nil {
				out[i] = nil
				continue
			}
			switch n := v.(type) {
			case int:
				out[i] = float64(n)
			case int8:
				out[i] = float64(n)
			case int16:
				out[i] = float64(n)
			case int32:
				out[i] = float64(n)
			case int64:
				out[i] = float64(n)
			case float32:
				out[i] = float64(n)
			case float64:
				out[i] = n
			default:
				out[i] = nil
			}
		}
	case "bool", "string", "any":
		copy(out, vals)
	default:
		copy(out, vals)
	}
	return out
}

// FromGPandasDataFrame helper to wrap an existing gpandas.DataFrame.
func FromGPandasDataFrame(df *gpdf.DataFrame) (*DataFrame, error) {
	if df == nil {
		return nil, fmt.Errorf("nil gpandas DataFrame")
	}
	wrap := &DataFrame{df}
	return wrap, nil
}
