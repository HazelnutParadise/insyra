package insyra

// =============================================================================
// Group-aware window transforms.
//
// Each public method on *GroupedDataTable picks a source column and returns
// a builder. The terminal call .As(name) produces a *DataList of length
// numRows, aligned to the parent's original row order, with each group's
// transform applied independently. Rolling and Expanding go through their
// own intermediate types so the reducer step matches the ungrouped API.
// =============================================================================

// GroupedColumnTransform is the terminal-stage intermediate for group-aware
// column transforms. It carries the per-group transform function and the
// source column; .As(name) executes the transform group-by-group and scatters
// results back into a row-aligned column.
type GroupedColumnTransform struct {
	parent      *GroupedDataTable
	sourceCol   int
	sourceLabel string
	perGroupFn  func(group *DataList) *DataList
	err         string
}

// As executes the configured transform and returns the resulting column with
// the given name. The output has the same length as the parent DataTable
// (rows that didn't appear in any group, if any, remain nil).
func (t *GroupedColumnTransform) As(name string) *DataList {
	if t == nil {
		out := NewDataList()
		out.SetName(name)
		return out
	}
	if t.err != "" {
		if t.parent != nil && t.parent.parent != nil {
			t.parent.parent.warn("GroupedColumnTransform.As", "%s", t.err)
		}
		out := NewDataList()
		out.SetName(name)
		return out
	}
	g := t.parent
	if g == nil || t.perGroupFn == nil {
		out := NewDataList()
		out.SetName(name)
		return out
	}

	// Total row count = max length across snapshot columns.
	nRows := 0
	for _, c := range g.columnsSnapshot {
		if l := len(c.data); l > nRows {
			nRows = l
		}
	}

	aligned := make([]any, nRows)
	src := g.columnsSnapshot[t.sourceCol]
	for _, encoded := range g.groupOrder {
		rowIdxs := g.rowsByGroup[encoded]
		sub := NewDataList()
		for _, idx := range rowIdxs {
			if idx < len(src.data) {
				sub.Append(src.data[idx])
			} else {
				sub.Append(nil)
			}
		}
		result := t.perGroupFn(sub)
		if result == nil {
			for _, idx := range rowIdxs {
				if idx < nRows {
					aligned[idx] = nil
				}
			}
			continue
		}
		rd := result.Data()
		for i, idx := range rowIdxs {
			if idx >= nRows {
				continue
			}
			if i < len(rd) {
				aligned[idx] = rd[i]
			} else {
				aligned[idx] = nil
			}
		}
	}

	out := NewDataList(aligned...)
	out.SetName(name)
	return out
}

// resolveSource picks a source column from the parent's snapshot. ok=false
// records an error on the returned transform so .As emits an empty column.
func (g *GroupedDataTable) resolveSource(funcName, col string) (int, string, bool) {
	if g == nil {
		return 0, "", false
	}
	if g.initErr != "" {
		if g.parent != nil {
			g.parent.warn(funcName, "%s", g.initErr)
		}
		return 0, "", false
	}
	num, label, ok := g.lookupSnapshotCol(col)
	if !ok {
		if g.parent != nil {
			g.parent.warn(funcName, "column %q not found", col)
		}
		return 0, col, false
	}
	return num, label, true
}

// ShiftCol applies Shift per group.
func (g *GroupedDataTable) ShiftCol(col string, periods int, fill ...any) *GroupedColumnTransform {
	num, label, ok := g.resolveSource("ShiftCol", col)
	t := &GroupedColumnTransform{parent: g, sourceCol: num, sourceLabel: label}
	if !ok {
		t.err = "ShiftCol: column not found"
		return t
	}
	t.perGroupFn = func(group *DataList) *DataList { return group.Shift(periods, fill...) }
	return t
}

// DiffCol applies Diff per group.
func (g *GroupedDataTable) DiffCol(col string, periods int) *GroupedColumnTransform {
	num, label, ok := g.resolveSource("DiffCol", col)
	t := &GroupedColumnTransform{parent: g, sourceCol: num, sourceLabel: label}
	if !ok {
		t.err = "DiffCol: column not found"
		return t
	}
	t.perGroupFn = func(group *DataList) *DataList { return group.Diff(periods) }
	return t
}

// PctChangeCol applies PctChange per group.
func (g *GroupedDataTable) PctChangeCol(col string, periods int) *GroupedColumnTransform {
	num, label, ok := g.resolveSource("PctChangeCol", col)
	t := &GroupedColumnTransform{parent: g, sourceCol: num, sourceLabel: label}
	if !ok {
		t.err = "PctChangeCol: column not found"
		return t
	}
	t.perGroupFn = func(group *DataList) *DataList { return group.PctChange(periods) }
	return t
}

// CumSumCol applies CumSum per group.
func (g *GroupedDataTable) CumSumCol(col string) *GroupedColumnTransform {
	num, label, ok := g.resolveSource("CumSumCol", col)
	t := &GroupedColumnTransform{parent: g, sourceCol: num, sourceLabel: label}
	if !ok {
		t.err = "CumSumCol: column not found"
		return t
	}
	t.perGroupFn = func(group *DataList) *DataList { return group.CumSum() }
	return t
}

// CumProdCol applies CumProd per group.
func (g *GroupedDataTable) CumProdCol(col string) *GroupedColumnTransform {
	num, label, ok := g.resolveSource("CumProdCol", col)
	t := &GroupedColumnTransform{parent: g, sourceCol: num, sourceLabel: label}
	if !ok {
		t.err = "CumProdCol: column not found"
		return t
	}
	t.perGroupFn = func(group *DataList) *DataList { return group.CumProd() }
	return t
}

// CumMaxCol applies CumMax per group.
func (g *GroupedDataTable) CumMaxCol(col string) *GroupedColumnTransform {
	num, label, ok := g.resolveSource("CumMaxCol", col)
	t := &GroupedColumnTransform{parent: g, sourceCol: num, sourceLabel: label}
	if !ok {
		t.err = "CumMaxCol: column not found"
		return t
	}
	t.perGroupFn = func(group *DataList) *DataList { return group.CumMax() }
	return t
}

// CumMinCol applies CumMin per group.
func (g *GroupedDataTable) CumMinCol(col string) *GroupedColumnTransform {
	num, label, ok := g.resolveSource("CumMinCol", col)
	t := &GroupedColumnTransform{parent: g, sourceCol: num, sourceLabel: label}
	if !ok {
		t.err = "CumMinCol: column not found"
		return t
	}
	t.perGroupFn = func(group *DataList) *DataList { return group.CumMin() }
	return t
}

// GroupedRollingCol is the intermediate produced by GroupedDataTable.RollingCol.
// Its terminal reducers (Sum / Mean / Min / Max / Median / Std / Var / Apply)
// each return a GroupedColumnTransform whose .As(name) materializes the
// per-group rolling column.
type GroupedRollingCol struct {
	parent      *GroupedDataTable
	sourceCol   int
	sourceLabel string
	opts        RollingOptions
	err         string
}

// RollingCol builds a rolling-window view over the given column, scoped to
// each group separately.
func (g *GroupedDataTable) RollingCol(col string, opts RollingOptions) *GroupedRollingCol {
	num, label, ok := g.resolveSource("RollingCol", col)
	gr := &GroupedRollingCol{parent: g, sourceCol: num, sourceLabel: label, opts: opts}
	if !ok {
		gr.err = "RollingCol: column not found"
	}
	return gr
}

func (gr *GroupedRollingCol) toTransform(perGroup func(group *DataList) *DataList) *GroupedColumnTransform {
	t := &GroupedColumnTransform{parent: gr.parent, sourceCol: gr.sourceCol, sourceLabel: gr.sourceLabel}
	if gr.err != "" {
		t.err = gr.err
		return t
	}
	t.perGroupFn = perGroup
	return t
}

// Sum executes a per-group rolling sum.
func (gr *GroupedRollingCol) Sum() *GroupedColumnTransform {
	opts := gr.opts
	return gr.toTransform(func(group *DataList) *DataList { return group.Rolling(opts).Sum() })
}

// Mean executes a per-group rolling mean.
func (gr *GroupedRollingCol) Mean() *GroupedColumnTransform {
	opts := gr.opts
	return gr.toTransform(func(group *DataList) *DataList { return group.Rolling(opts).Mean() })
}

// Min executes a per-group rolling min.
func (gr *GroupedRollingCol) Min() *GroupedColumnTransform {
	opts := gr.opts
	return gr.toTransform(func(group *DataList) *DataList { return group.Rolling(opts).Min() })
}

// Max executes a per-group rolling max.
func (gr *GroupedRollingCol) Max() *GroupedColumnTransform {
	opts := gr.opts
	return gr.toTransform(func(group *DataList) *DataList { return group.Rolling(opts).Max() })
}

// Median executes a per-group rolling median.
func (gr *GroupedRollingCol) Median() *GroupedColumnTransform {
	opts := gr.opts
	return gr.toTransform(func(group *DataList) *DataList { return group.Rolling(opts).Median() })
}

// Std executes a per-group rolling sample std.
func (gr *GroupedRollingCol) Std() *GroupedColumnTransform {
	opts := gr.opts
	return gr.toTransform(func(group *DataList) *DataList { return group.Rolling(opts).Std() })
}

// Var executes a per-group rolling sample variance.
func (gr *GroupedRollingCol) Var() *GroupedColumnTransform {
	opts := gr.opts
	return gr.toTransform(func(group *DataList) *DataList { return group.Rolling(opts).Var() })
}

// Apply executes a per-group rolling custom reducer.
func (gr *GroupedRollingCol) Apply(fn func(window []any) any) *GroupedColumnTransform {
	opts := gr.opts
	return gr.toTransform(func(group *DataList) *DataList { return group.Rolling(opts).Apply(fn) })
}

// GroupedExpandingCol is the intermediate produced by
// GroupedDataTable.ExpandingCol. Each terminal reducer returns a
// GroupedColumnTransform whose .As(name) materializes the column.
type GroupedExpandingCol struct {
	parent      *GroupedDataTable
	sourceCol   int
	sourceLabel string
	minObs      int
	err         string
}

// ExpandingCol builds an expanding-window view over the given column, scoped
// to each group separately.
func (g *GroupedDataTable) ExpandingCol(col string, minObs int) *GroupedExpandingCol {
	num, label, ok := g.resolveSource("ExpandingCol", col)
	ge := &GroupedExpandingCol{parent: g, sourceCol: num, sourceLabel: label, minObs: minObs}
	if !ok {
		ge.err = "ExpandingCol: column not found"
	}
	return ge
}

func (ge *GroupedExpandingCol) toTransform(perGroup func(group *DataList) *DataList) *GroupedColumnTransform {
	t := &GroupedColumnTransform{parent: ge.parent, sourceCol: ge.sourceCol, sourceLabel: ge.sourceLabel}
	if ge.err != "" {
		t.err = ge.err
		return t
	}
	t.perGroupFn = perGroup
	return t
}

// Sum executes a per-group expanding sum.
func (ge *GroupedExpandingCol) Sum() *GroupedColumnTransform {
	m := ge.minObs
	return ge.toTransform(func(group *DataList) *DataList { return group.Expanding(m).Sum() })
}

// Mean executes a per-group expanding mean.
func (ge *GroupedExpandingCol) Mean() *GroupedColumnTransform {
	m := ge.minObs
	return ge.toTransform(func(group *DataList) *DataList { return group.Expanding(m).Mean() })
}

// Min executes a per-group expanding min.
func (ge *GroupedExpandingCol) Min() *GroupedColumnTransform {
	m := ge.minObs
	return ge.toTransform(func(group *DataList) *DataList { return group.Expanding(m).Min() })
}

// Max executes a per-group expanding max.
func (ge *GroupedExpandingCol) Max() *GroupedColumnTransform {
	m := ge.minObs
	return ge.toTransform(func(group *DataList) *DataList { return group.Expanding(m).Max() })
}

// Median executes a per-group expanding median.
func (ge *GroupedExpandingCol) Median() *GroupedColumnTransform {
	m := ge.minObs
	return ge.toTransform(func(group *DataList) *DataList { return group.Expanding(m).Median() })
}

// Std executes a per-group expanding sample std.
func (ge *GroupedExpandingCol) Std() *GroupedColumnTransform {
	m := ge.minObs
	return ge.toTransform(func(group *DataList) *DataList { return group.Expanding(m).Std() })
}

// Var executes a per-group expanding sample variance.
func (ge *GroupedExpandingCol) Var() *GroupedColumnTransform {
	m := ge.minObs
	return ge.toTransform(func(group *DataList) *DataList { return group.Expanding(m).Var() })
}
