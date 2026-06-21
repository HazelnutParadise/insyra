package insyra

import (
	"math"
	"testing"
)

func TestDataTableFillForwardSpecificColumns(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1.0, nil, 3.0).SetName("num"),
		NewDataList("x", nil, "z").SetName("text"),
		NewDataList(10.0, math.NaN(), 30.0).SetName("other"),
	)

	dt.FillForward(0, "num")

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

	dt.FillByInterpolation()

	assertImputeData(t, dt.GetColByName("a"), []any{1.0, 2.0, 3.0})
	assertImputeData(t, dt.GetColByName("b"), []any{10.0, 20.0, 30.0})
	assertImputeData(t, dt.GetColByName("text"), []any{"x", nil, "z"})
}

func TestDataTableFillWithMeanAndMode(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1.0, nil, 3.0).SetName("num"),
		NewDataList("red", nil, "red").SetName("color"),
	)

	dt.FillWithMean("num")
	dt.FillWithMode("color")

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
