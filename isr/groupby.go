package isr

import "github.com/HazelnutParadise/insyra"

// GroupBy returns a *insyra.GroupedDataTable for the underlying DataTable,
// matching the chain pattern of the rest of the isr package. Use it together
// with insyra.AggregateConfig to do split-apply-combine:
//
//	report := dt.GroupBy("region").Aggregate(
//	    insyra.AggregateConfig{SourceCol: "revenue", Op: insyra.OpSum},
//	)
func (t *dt) GroupBy(keyCols ...string) *insyra.GroupedDataTable {
	return t.DataTable.GroupBy(keyCols...)
}
