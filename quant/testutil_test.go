package quant

import "github.com/HazelnutParadise/insyra"

// toDL builds an IDataList from float64 values for tests.
func toDL(vs ...float64) insyra.IDataList {
	anys := make([]any, len(vs))
	for i, v := range vs {
		anys[i] = v
	}
	return insyra.NewDataList(anys...)
}

// toDT builds a T×N DataTable from a row-major matrix (rows = periods,
// columns = strategies) — the layout PBO expects.
func toDT(perf [][]float64) insyra.IDataTable {
	if len(perf) == 0 {
		return insyra.NewDataTable()
	}
	t := len(perf)
	n := len(perf[0])
	cols := make([]*insyra.DataList, n)
	for j := range n {
		vals := make([]any, t)
		for i := range t {
			vals[i] = perf[i][j]
		}
		cols[j] = insyra.NewDataList(vals...)
	}
	return insyra.NewDataTable(cols...)
}
