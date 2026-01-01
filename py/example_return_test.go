package py

import (
	"reflect"
	"testing"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
)

func TestReturnPandasDataFrame(t *testing.T) {
	var dt *insyra.DataTable
	err := RunCode(&dt, `
import pandas as pd
df = pd.DataFrame({"a": [1, 2], "b": [3, 4]}, index=["r1", "r2"])
insyra.Return(df)
`)
	if err != nil {
		t.Fatalf("RunCode: %v", err)
	}
	if dt == nil {
		t.Fatal("datatable is nil")
	}
	if got := dt.NumRows(); got != 2 {
		t.Fatalf("rows: %d", got)
	}
	if got := dt.NumCols(); got != 2 {
		t.Fatalf("cols: %d", got)
	}
	if !reflect.DeepEqual(dt.ColNames(), []string{"a", "b"}) {
		t.Fatalf("col names: %v", dt.ColNames())
	}
	if !reflect.DeepEqual(dt.RowNames(), []string{"r1", "r2"}) {
		t.Fatalf("row names: %v", dt.RowNames())
	}
	if got := conv.ParseF64(dt.GetElement(0, "A")); got != 1 {
		t.Fatalf("A1: %v", got)
	}
}

func TestReturnPandasSeries(t *testing.T) {
	var dl *insyra.DataList
	err := RunCode(&dl, `
import pandas as pd
s = pd.Series([10, 20], name="s1")
insyra.Return(s)
`)
	if err != nil {
		t.Fatalf("RunCode: %v", err)
	}
	if dl == nil {
		t.Fatal("datalist is nil")
	}
	if dl.GetName() != "s1" {
		t.Fatalf("name: %s", dl.GetName())
	}
	if dl.Len() != 2 {
		t.Fatalf("len: %d", dl.Len())
	}
	if got := conv.ParseF64(dl.Get(1)); got != 20 {
		t.Fatalf("index 1: %v", got)
	}
}

func TestReturnPolarsDataFrame(t *testing.T) {
	var dt *insyra.DataTable
	err := RunCode(&dt, `
import polars as pl
df = pl.DataFrame({"a": [1, 2], "b": [3, 4]})
insyra.Return(df)
`)
	if err != nil {
		t.Fatalf("RunCode: %v", err)
	}
	if dt == nil {
		t.Fatal("datatable is nil")
	}
	if got := dt.NumRows(); got != 2 {
		t.Fatalf("rows: %d", got)
	}
	if got := dt.NumCols(); got != 2 {
		t.Fatalf("cols: %d", got)
	}
	if !reflect.DeepEqual(dt.ColNames(), []string{"a", "b"}) {
		t.Fatalf("col names: %v", dt.ColNames())
	}
	if got := conv.ParseF64(dt.GetElement(1, "B")); got != 4 {
		t.Fatalf("B2: %v", got)
	}
}

func TestReturnPolarsSeries(t *testing.T) {
	var dl *insyra.DataList
	err := RunCode(&dl, `
import polars as pl
s = pl.Series("s1", [10, 20])
insyra.Return(s)
`)
	if err != nil {
		t.Fatalf("RunCode: %v", err)
	}
	if dl == nil {
		t.Fatal("datalist is nil")
	}
	if dl.GetName() != "s1" {
		t.Fatalf("name: %s", dl.GetName())
	}
	if dl.Len() != 2 {
		t.Fatalf("len: %d", dl.Len())
	}
	if got := conv.ParseF64(dl.Get(0)); got != 10 {
		t.Fatalf("index 0: %v", got)
	}
}
