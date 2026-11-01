package insyra

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestDataTableFillForwardSpecificColumns(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1.0, nil, 3.0).SetName("num"),
		NewDataList("x", nil, "z").SetName("text"),
		NewDataList(10.0, math.NaN(), 30.0).SetName("other"),
	)

	dt.FillForward(0, Name("num"))

	assertImputeData(t, dt.GetColByName("num"), []any{1.0, 1.0, 3.0})
	assertImputeData(t, dt.GetColByName("text"), []any{"x", nil, "z"})
	assertImputeData(t, dt.GetColByName("other"), []any{10.0, math.NaN(), 30.0})
}

func TestDataTableNumericStrategiesSkipStringColumns(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1.0, nil, 3.0).SetName("num"),
		NewDataList("x", nil, "z").SetName("text"),
	)

	dt.FillWithMedian()

	assertImputeData(t, dt.GetColByName("num"), []any{1.0, 2.0, 3.0})
	assertImputeData(t, dt.GetColByName("text"), []any{"x", nil, "z"})
}

func TestDataTableEmptyColumnsDefaultToAllApplicable(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1.0, nil, 3.0).SetName("a"),
		NewDataList(10.0, math.NaN(), 30.0).SetName("b"),
		NewDataList("x", nil, "z").SetName("text"),
	)

	dt.FillByInterpolation(false)

	assertImputeData(t, dt.GetColByName("a"), []any{1.0, 2.0, 3.0})
	assertImputeData(t, dt.GetColByName("b"), []any{10.0, 20.0, 30.0})
	assertImputeData(t, dt.GetColByName("text"), []any{"x", nil, "z"})
}

func TestDataTableFillWithMeanAndMode(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1.0, nil, 3.0).SetName("num"),
		NewDataList("red", nil, "red").SetName("color"),
	)

	dt.FillWithMean(Name("num"))
	dt.FillWithMode(Name("color"))

	assertImputeData(t, dt.GetColByName("num"), []any{1.0, 2.0, 3.0})
	assertImputeData(t, dt.GetColByName("color"), []any{"red", "red", "red"})
}

func TestDataTableFillBackwardWithLimit(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1.0, nil, math.NaN(), nil, 5.0).SetName("num"),
	)

	dt.FillBackward(2)

	assertImputeData(t, dt.GetColByName("num"), []any{1.0, nil, 5.0, 5.0, 5.0})
}

func fillTable() *DataTable {
	return NewDataTable(
		NewDataList(10.0, nil, 30.0).SetName("price"),
		NewDataList("Taipei", nil, "Taichung").SetName("city"),
		NewDataList(nil, nil, nil).SetName("blank"),
	)
}

// A table-level fill skips a column it cannot fill when the caller named no
// columns, because it was asked to do what it can. A column the caller named
// is different: skipping it silently let them think it was filled.
func TestTableFillReportsANamedColumnItCannotFill(t *testing.T) {
	fills := map[string]func(dt *DataTable, cols ...any) *DataTable{
		"FillWithMean":   func(dt *DataTable, cols ...any) *DataTable { return dt.FillWithMean(cols...) },
		"FillWithMedian": func(dt *DataTable, cols ...any) *DataTable { return dt.FillWithMedian(cols...) },
		"FillByInterpolation": func(dt *DataTable, cols ...any) *DataTable {
			return dt.FillByInterpolation(false, cols...)
		},
	}
	for name, fill := range fills {
		t.Run(name+"/text column named", func(t *testing.T) {
			dt := fillTable()
			fill(dt, Name("price"), Name("city"))
			err := dt.Err()
			if err == nil || !strings.Contains(err.Error(), "city") || !strings.Contains(err.Error(), "string") {
				t.Fatalf("got %v, want an error naming city and its string values", err)
			}
			if dt.GetColByName("price").Get(1) == nil {
				t.Error("the fillable named column was left unfilled")
			}
		})
		t.Run(name+"/empty column named", func(t *testing.T) {
			dt := fillTable()
			fill(dt, Name("blank"))
			if err := dt.Err(); err == nil || !strings.Contains(err.Error(), "blank") {
				t.Fatalf("got %v, want an error naming the column with no values", err)
			}
		})
		t.Run(name+"/no column named", func(t *testing.T) {
			dt := fillTable()
			fill(dt)
			if err := dt.Err(); err != nil {
				t.Fatalf("filling every column reported %v", err)
			}
			if dt.GetColByName("price").Get(1) == nil {
				t.Error("price was not filled")
			}
			if dt.GetColByName("city").Get(1) != nil {
				t.Error("city was changed")
			}
		})
	}
}

// The table version could not extrapolate at all, so the CLI's extrapolate
// option was dropped for tables.
func TestTableFillByInterpolationCanExtrapolate(t *testing.T) {
	build := func() *DataTable { return NewDataTable(NewDataList(nil, 2.0, 3.0, nil).SetName("x")) }
	if got := build().FillByInterpolation(true).GetColByName("x").Data(); !reflect.DeepEqual(got, []any{1.0, 2.0, 3.0, 4.0}) {
		t.Fatalf("with extrapolation: %v", got)
	}
	if got := build().FillByInterpolation(false).GetColByName("x").Data(); got[0] != nil || got[3] != nil {
		t.Fatalf("without extrapolation the ends were filled: %v", got)
	}
}
