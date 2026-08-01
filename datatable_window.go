package insyra

// =============================================================================
// DataTable per-column window transforms
//
// All public methods accept a column reference (column name or Excel-style
// index such as "A"/"B") and return either a transformed *DataList (for the
// scalar transforms) or a builder that produces one (for Rolling / Expanding).
// The returned DataList is not attached to dt — call dt.AppendCols(...) or
// dt.UpdateCol(...) to wire it in.
// =============================================================================

// snapshotCol resolves col against dt and returns a stand-alone *DataList
// containing a copy of the column's data, plus its display label. ok is false
// when the column is missing; a warning is recorded on dt.
func (dt *DataTable) snapshotCol(funcName, col string) (snap *DataList, label string, ok bool) {
	dt.AtomicDo(func(t *DataTable) {
		num, lbl, found := resolveColForGroup(t, col)
		if !found {
			t.warn(funcName, "column %q not found", col)
			return
		}
		src := t.columns[num]
		buf := make([]any, len(src.data))
		copy(buf, src.data)
		snap = NewDataList(buf...)
		snap.name = lbl
		label = lbl
		ok = true
	})
	return snap, label, ok
}

// ShiftCol returns a new column equal to dt[col].Shift(periods, fill...).
// Returns an empty DataList when the column is missing.
func (dt *DataTable) ShiftCol(col string, periods int, fill ...any) *DataList {
	snap, _, ok := dt.snapshotCol("ShiftCol", col)
	if !ok {
		return NewDataList()
	}
	return snap.Shift(periods, fill...)
}

// DiffCol returns a new column equal to dt[col].Diff(periods).
func (dt *DataTable) DiffCol(col string, periods int) *DataList {
	snap, _, ok := dt.snapshotCol("DiffCol", col)
	if !ok {
		return NewDataList()
	}
	out := snap.Diff(periods)
	if out == nil {
		return NewDataList()
	}
	return out
}

// PctChangeCol returns a new column equal to dt[col].PctChange(periods).
func (dt *DataTable) PctChangeCol(col string, periods int) *DataList {
	snap, _, ok := dt.snapshotCol("PctChangeCol", col)
	if !ok {
		return NewDataList()
	}
	out := snap.PctChange(periods)
	if out == nil {
		return NewDataList()
	}
	return out
}

// CumSumCol returns the cumulative sum of dt[col].
func (dt *DataTable) CumSumCol(col string) *DataList {
	snap, _, ok := dt.snapshotCol("CumSumCol", col)
	if !ok {
		return NewDataList()
	}
	return snap.CumSum()
}

// CumProdCol returns the cumulative product of dt[col].
func (dt *DataTable) CumProdCol(col string) *DataList {
	snap, _, ok := dt.snapshotCol("CumProdCol", col)
	if !ok {
		return NewDataList()
	}
	return snap.CumProd()
}

// CumMaxCol returns the running maximum of dt[col].
func (dt *DataTable) CumMaxCol(col string) *DataList {
	snap, _, ok := dt.snapshotCol("CumMaxCol", col)
	if !ok {
		return NewDataList()
	}
	return snap.CumMax()
}

// CumMinCol returns the running minimum of dt[col].
func (dt *DataTable) CumMinCol(col string) *DataList {
	snap, _, ok := dt.snapshotCol("CumMinCol", col)
	if !ok {
		return NewDataList()
	}
	return snap.CumMin()
}

// RollingCol returns a RollingDataList view of dt[col]. Terminal reducers
// (Mean, Sum, Min, Max, Median, Std, Var, Apply, Corr) produce a *DataList
// the same length as the column.
func (dt *DataTable) RollingCol(col string, opts RollingOptions) *RollingDataList {
	snap, _, ok := dt.snapshotCol("RollingCol", col)
	if !ok {
		return &RollingDataList{opts: opts, err: "RollingCol: column not found"}
	}
	return snap.Rolling(opts)
}

// ExpandingCol returns an ExpandingDataList view of dt[col].
func (dt *DataTable) ExpandingCol(col string, minObs int) *ExpandingDataList {
	snap, _, ok := dt.snapshotCol("ExpandingCol", col)
	if !ok {
		return &ExpandingDataList{minObs: minObs, err: "ExpandingCol: column not found"}
	}
	return snap.Expanding(minObs)
}
