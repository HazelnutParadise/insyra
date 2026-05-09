package isr_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	. "github.com/HazelnutParadise/insyra/isr"
	"github.com/stretchr/testify/assert"
)

func TestDT_PivotUnpivot_RoundTrip(t *testing.T) {
	wide := DT.From(DLs{
		DL.From(1, 2).SetName("id"),
		DL.From(10, 30).SetName("A"),
		DL.From(20, 40).SetName("B"),
	})

	long := wide.Unpivot(Unpivot{
		IDVars:    []string{"id"},
		VarName:   "var",
		ValueName: "val",
	})
	assert.Nil(t, long.Err(), "unpivot should not record an error")
	assert.Equal(t, 4, long.NumRows(), "long form should have 4 rows")
	assert.Equal(t, []string{"id", "var", "val"}, long.ColNames())

	round := long.Pivot(Pivot{
		Index:    []string{"id"},
		Columns:  "var",
		Values:   "val",
		SortCols: true,
	})
	assert.Nil(t, round.Err(), "pivot back should not record an error")
	assert.Equal(t, 2, round.NumRows())
	assert.Equal(t, []string{"id", "A", "B"}, round.ColNames())

	colA := round.GetColByName("A").Data()
	colB := round.GetColByName("B").Data()
	a0, _ := insyra.ToFloat64Safe(colA[0])
	a1, _ := insyra.ToFloat64Safe(colA[1])
	b0, _ := insyra.ToFloat64Safe(colB[0])
	b1, _ := insyra.ToFloat64Safe(colB[1])
	assert.Equal(t, 10.0, a0)
	assert.Equal(t, 30.0, a1)
	assert.Equal(t, 20.0, b0)
	assert.Equal(t, 40.0, b1)
}

func TestDT_Pivot_AggSum(t *testing.T) {
	dt := DT.From(DLs{
		DL.From("APAC", "APAC", "APAC", "EMEA").SetName("region"),
		DL.From("A", "A", "B", "A").SetName("product"),
		DL.From(10, 5, 20, 30).SetName("sales"),
	})
	wide := dt.Pivot(Pivot{
		Index:   []string{"region"},
		Columns: "product",
		Values:  "sales",
		Agg:     "sum",
		FillNA:  0,
	})
	assert.Nil(t, wide.Err())
	colA := wide.GetColByName("A").Data()
	got, _ := insyra.ToFloat64Safe(colA[0])
	assert.Equal(t, 15.0, got, "APAC.A should aggregate (10 + 5)")
}

func TestDT_Pivot_RecordsErrOnBadConfig(t *testing.T) {
	dt := DT.From(DLs{
		DL.From("APAC", "APAC").SetName("region"),
		DL.From("A", "A").SetName("product"),
		DL.From(10, 20).SetName("sales"),
	})
	out := dt.Pivot(Pivot{
		Index:   []string{"region"},
		Columns: "product",
		Values:  "sales",
	})
	assert.NotNil(t, out.Err(), "duplicate (Index, Columns) without Agg should record an error")
}
