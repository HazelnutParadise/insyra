package insyra

// =============================================================================
// DataTable per-column window transforms
//
// All public methods accept a column reference (column name or Excel-style
// index such as "A"/"B") and return either a transformed *DataList (for the
// scalar transforms) or a builder that produces one (for Rolling / Expanding).
// The returned DataList is not attached to dt — call dt.AppendCols(...) or
// dt.UpdateCol(...) to wire it in.
//
// A failing call returns an empty DataList carrying the error and records the
// error on dt as well.
// =============================================================================

// snapshotCol resolves col against dt and returns a stand-alone *DataList
// containing a copy of the column's data, plus its display label. ok is false
// when the column is missing; a warning is recorded on dt.
func (dt *DataTable) snapshotCol(funcName string, col any) (snap *DataList, label string, ok bool) {
	dt.AtomicDo(func(t *DataTable) {
		num, warning, problem := t.lookupColSelector(col)
		if problem != "" {
			t.fail(funcName, "%s", problem)
			return
		}
		if warning != "" {
			t.warn(funcName, "%s", warning)
		}
		lbl := selectorLabel(col)
		if name := t.columns[num].name; name != "" {
			lbl = name
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
// Returns an empty DataList carrying the table's error when the column is
// missing.
func (dt *DataTable) ShiftCol(col any, periods int, fill ...any) *DataList {
	snap, _, ok := dt.snapshotCol("ShiftCol", col)
	if !ok {
		return emptyWindowResult("", dt.Err())
	}
	return snap.Shift(periods, fill...)
}

// DiffCol returns a new column equal to dt[col].Diff(periods). Returns an empty
// DataList carrying the table's error when the column is missing.
func (dt *DataTable) DiffCol(col any, periods int) *DataList {
	snap, _, ok := dt.snapshotCol("DiffCol", col)
	if !ok {
		return emptyWindowResult("", dt.Err())
	}
	out := snap.Diff(periods)
	if err := out.Err(); err != nil {
		// snap is a throwaway copy, so its error would be lost; report it on
		// the table the caller actually holds.
		dt.SetErr("DataTable", "DiffCol", "%s", err.Message)
	}
	return out
}

// PctChangeCol returns a new column equal to dt[col].PctChange(periods).
// Returns an empty DataList carrying the table's error when the column is
// missing.
func (dt *DataTable) PctChangeCol(col any, periods int) *DataList {
	snap, _, ok := dt.snapshotCol("PctChangeCol", col)
	if !ok {
		return emptyWindowResult("", dt.Err())
	}
	out := snap.PctChange(periods)
	if err := out.Err(); err != nil {
		dt.SetErr("DataTable", "PctChangeCol", "%s", err.Message)
	}
	return out
}

// CumSumCol returns the cumulative sum of dt[col]. Returns an empty DataList
// carrying the table's error when the column is missing.
func (dt *DataTable) CumSumCol(col any) *DataList {
	snap, _, ok := dt.snapshotCol("CumSumCol", col)
	if !ok {
		return emptyWindowResult("", dt.Err())
	}
	return snap.CumSum()
}

// CumProdCol returns the cumulative product of dt[col]. Returns an empty
// DataList carrying the table's error when the column is missing.
func (dt *DataTable) CumProdCol(col any) *DataList {
	snap, _, ok := dt.snapshotCol("CumProdCol", col)
	if !ok {
		return emptyWindowResult("", dt.Err())
	}
	return snap.CumProd()
}

// CumMaxCol returns the running maximum of dt[col]. Returns an empty DataList
// carrying the table's error when the column is missing.
func (dt *DataTable) CumMaxCol(col any) *DataList {
	snap, _, ok := dt.snapshotCol("CumMaxCol", col)
	if !ok {
		return emptyWindowResult("", dt.Err())
	}
	return snap.CumMax()
}

// CumMinCol returns the running minimum of dt[col]. Returns an empty DataList
// carrying the table's error when the column is missing.
func (dt *DataTable) CumMinCol(col any) *DataList {
	snap, _, ok := dt.snapshotCol("CumMinCol", col)
	if !ok {
		return emptyWindowResult("", dt.Err())
	}
	return snap.CumMin()
}

// RollingCol returns a RollingDataList view of dt[col]. Terminal reducers
// (Mean, Sum, Min, Max, Median, Std, Var, Apply, Corr, Cov, Beta) produce a
// *DataList the same length as the column. An invalid option is recorded on dt
// as well as carried by every reducer's empty result.
func (dt *DataTable) RollingCol(col any, opts RollingOptions) *RollingDataList {
	snap, _, ok := dt.snapshotCol("RollingCol", col)
	if !ok {
		return &RollingDataList{opts: opts, err: dt.Err()}
	}
	r := snap.Rolling(opts)
	if r.err != nil {
		// snap is a throwaway copy that has already logged the failure;
		// record it on the table the caller holds without logging it twice.
		dt.setError(LogLevelError, "DataTable", "RollingCol", r.err.Message)
	}
	return r
}

// ExpandingCol returns an ExpandingDataList view of dt[col]. Expanding has no
// option that can be invalid, so nothing here can fail beyond a missing column.
func (dt *DataTable) ExpandingCol(col any, minObs int) *ExpandingDataList {
	snap, _, ok := dt.snapshotCol("ExpandingCol", col)
	if !ok {
		return &ExpandingDataList{minObs: minObs, err: dt.Err()}
	}
	return snap.Expanding(minObs)
}

// EWMCol returns an EWMDataList view of dt[col]. The column may be named or
// addressed by its Excel-style index. An invalid option is recorded on dt as
// well as carried by every reducer's empty result.
func (dt *DataTable) EWMCol(col any, opts EWMOptions) *EWMDataList {
	snap, _, ok := dt.snapshotCol("EWMCol", col)
	if !ok {
		return &EWMDataList{opts: opts, err: dt.Err()}
	}
	e := snap.EWM(opts)
	if e.err != nil {
		// snap is a throwaway copy that has already logged the failure;
		// record it on the table the caller holds without logging it twice.
		dt.setError(LogLevelError, "DataTable", "EWMCol", e.err.Message)
	}
	return e
}
