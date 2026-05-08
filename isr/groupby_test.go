package isr_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	. "github.com/HazelnutParadise/insyra/isr"
	"github.com/stretchr/testify/assert"
)

func TestDT_GroupBy_Aggregate(t *testing.T) {
	dt := DT.From(
		DLs{
			DL.From("east", "east", "west", "west", "south").SetName("region"),
			DL.From(100, 200, 50, 75, 300).SetName("revenue"),
			DL.From(1, 2, 3, 4, 5).SetName("qty"),
		},
	)

	report := dt.GroupBy("region").Aggregate(
		insyra.AggregateConfig{SourceCol: "revenue", Op: insyra.OpSum, As: "total_rev"},
		insyra.AggregateConfig{SourceCol: "qty", Op: insyra.OpMean, As: "avg_qty"},
	)

	assert.Equal(t, 3, report.NumRows(), "expected 3 groups")
	assert.Equal(t, []string{"region", "total_rev", "avg_qty"}, report.ColNames(), "column order")

	regions := report.GetColByName("region").Data()
	assert.Equal(t, []any{"east", "west", "south"}, regions, "first-seen group order")

	rev := report.GetColByName("total_rev").Data()
	assert.InDelta(t, 300.0, rev[0], 1e-9)
	assert.InDelta(t, 125.0, rev[1], 1e-9)
	assert.InDelta(t, 300.0, rev[2], 1e-9)

	avgQty := report.GetColByName("avg_qty").Data()
	assert.InDelta(t, 1.5, avgQty[0], 1e-9)
	assert.InDelta(t, 3.5, avgQty[1], 1e-9)
	assert.InDelta(t, 5.0, avgQty[2], 1e-9)
}
