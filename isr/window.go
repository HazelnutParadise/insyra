package isr

import "github.com/HazelnutParadise/insyra"

// =============================================================================
// Window transforms — isr wrappers
//
// These thin wrappers expose insyra's window transforms with isr-style
// ergonomics. Each scalar transform (Shift / Diff / PctChange / CumSum /
// CumProd / CumMax / CumMin) returns a *insyra.DataList ready to AppendCols
// onto the table. Rolling / Expanding return the underlying builder so that
// callers can pick a reducer.
// =============================================================================

// Rolling mirrors insyra.RollingOptions with the same field semantics. It's
// kept here so callers can use isr.Rolling{...} naturally.
type Rolling struct {
	Window  int
	MinObs  int
	Center  bool
	Weights []float64
}

// Shift returns dt[col].Shift(periods, fill...).
func (t *dt) Shift(col string, periods int, fill ...any) *insyra.DataList {
	return t.ShiftCol(col, periods, fill...)
}

// Diff returns dt[col].Diff(periods).
func (t *dt) Diff(col string, periods int) *insyra.DataList {
	return t.DiffCol(col, periods)
}

// PctChange returns dt[col].PctChange(periods).
func (t *dt) PctChange(col string, periods int) *insyra.DataList {
	return t.PctChangeCol(col, periods)
}

// CumSum returns dt[col].CumSum().
func (t *dt) CumSum(col string) *insyra.DataList {
	return t.CumSumCol(col)
}

// CumProd returns dt[col].CumProd().
func (t *dt) CumProd(col string) *insyra.DataList {
	return t.CumProdCol(col)
}

// CumMax returns dt[col].CumMax().
func (t *dt) CumMax(col string) *insyra.DataList {
	return t.CumMaxCol(col)
}

// CumMin returns dt[col].CumMin().
func (t *dt) CumMin(col string) *insyra.DataList {
	return t.CumMinCol(col)
}

// RollingOn builds a rolling-window view over dt[col]. The terminal call
// (Mean / Sum / Min / Max / Median / Std / Var / Apply / Corr) materialises
// a column the same length as the source.
func (t *dt) RollingOn(col string, r Rolling) *insyra.RollingDataList {
	return t.RollingCol(col, insyra.RollingOptions{
		Window:  r.Window,
		MinObs:  r.MinObs,
		Center:  r.Center,
		Weights: r.Weights,
	})
}

// ExpandingOn builds an expanding-window view over dt[col].
func (t *dt) ExpandingOn(col string, minObs int) *insyra.ExpandingDataList {
	return t.ExpandingCol(col, minObs)
}
