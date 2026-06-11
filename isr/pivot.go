package isr

import "github.com/HazelnutParadise/insyra"

// Pivot is the isr-style argument struct for (*DataTable).Pivot. Field names
// mirror insyra.PivotConfig but use a shorter Agg field for ergonomics in
// chained calls. See insyra.PivotConfig for full semantics.
type Pivot struct {
	Index    []string
	Columns  string
	Values   string
	Agg      string
	Custom   func(group *insyra.DataList) any
	FillNA   any
	SortCols bool
}

// Unpivot is the isr-style argument struct for (*DataTable).Unpivot. Field
// names mirror insyra.UnpivotConfig. See insyra.UnpivotConfig for semantics.
type Unpivot struct {
	IDVars    []string
	ValueVars []string
	VarName   string
	ValueName string
	DropNA    bool
}

// Pivot reshapes long-form data into wide form. Errors from the underlying
// (*DataTable).Pivot are surfaced via the returned dt's Err(); callers who
// want to branch on failure should check Err() before using the result.
func (t *dt) Pivot(p Pivot) *dt {
	out, _ := t.DataTable.Pivot(insyra.PivotConfig{
		Index:    p.Index,
		Columns:  p.Columns,
		Values:   p.Values,
		AggFunc:  p.Agg,
		Custom:   p.Custom,
		FillNA:   p.FillNA,
		SortCols: p.SortCols,
	})
	return UseDT(out)
}

// Unpivot reshapes wide-form data into long form. See Pivot for the chain /
// error contract.
func (t *dt) Unpivot(u Unpivot) *dt {
	out, _ := t.DataTable.Unpivot(insyra.UnpivotConfig{
		IDVars:    u.IDVars,
		ValueVars: u.ValueVars,
		VarName:   u.VarName,
		ValueName: u.ValueName,
		DropNA:    u.DropNA,
	})
	return UseDT(out)
}
