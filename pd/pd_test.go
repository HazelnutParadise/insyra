package pd

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestFromAndToDataTable(t *testing.T) {
	// build a simple DataTable
	col1 := insyra.NewDataList(int64(1), int64(2), int64(3)).SetName("id")
	col2 := insyra.NewDataList(10.5, 20.0, 30.25).SetName("v")
	col3 := insyra.NewDataList("a", "b", "c").SetName("s")
	dt := insyra.NewDataTable(col1, col2, col3)

	// convert to gpandas DataFrame
	wrap, err := FromDataTable(dt)
	if err != nil {
		t.Fatalf("FromDataTable failed: %v", err)
	}
	if wrap == nil || wrap.DataFrame == nil {
		t.Fatalf("wrap is nil or DF nil")
	}

	// head
	head := wrap.Head(2)
	if err != nil {
		t.Fatalf("Head failed: %v", err)
	}
	hdf, err := FromGPandasDataFrame(head)
	if err != nil {
		t.Fatalf("ToDataTable failed: %v", err)
	}
	hdt, err := hdf.ToDataTable()
	if err != nil {
		t.Fatalf("ToDataTable failed: %v", err)
	}
	r, c := hdt.Size()
	if r != 2 || c != 3 {
		t.Fatalf("unexpected head size: %d x %d", r, c)
	}

	// round trip
	dt2, err := wrap.ToDataTable()
	if err != nil {
		t.Fatalf("ToDataTable failed: %v", err)
	}
	r2, c2 := dt2.Size()
	if r2 != 3 || c2 != 3 {
		t.Fatalf("unexpected roundtrip size: %d x %d", r2, c2)
	}
}

func TestSeriesFromDataList(t *testing.T) {
	// int64 Series
	intDL := insyra.NewDataList(int(1), int32(2), int64(3))
	intSeries, err := FromDataList(intDL)
	if err != nil {
		t.Fatalf("FromDataList int failed: %v", err)
	}
	if intSeries.Len() != 3 {
		t.Fatalf("unexpected int Series length: %d", intSeries.Len())
	}
	// float64 Series
	floatDL := insyra.NewDataList(float32(1.5), float64(2.5), float64(3.0))
	floatSeries, err := FromDataList(floatDL)
	if err != nil {
		t.Fatalf("FromDataList float failed: %v", err)
	}
	if floatSeries.Len() != 3 {
		t.Fatalf("unexpected float Series length: %d", floatSeries.Len())
	}
	// string Series
	stringDL := insyra.NewDataList("a", "b", "c")
	stringSeries, err := FromDataList(stringDL)
	if err != nil {
		t.Fatalf("FromDataList string failed: %v", err)
	}
	if stringSeries.Len() != 3 {
		t.Fatalf("unexpected string Series length: %d", stringSeries.Len())
	}
	// mixed Series
	mixedDL := insyra.NewDataList(1, 2.5, "c")
	anySeries, err := FromDataList(mixedDL)
	if err != nil {
		t.Fatalf("FromDataList mixed failed: %v", err)
	}
	if anySeries.Len() != 3 {
		t.Fatalf("unexpected any Series length: %d", anySeries.Len())
	}
}
