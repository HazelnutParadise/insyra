package py

import (
	"testing"

	"github.com/HazelnutParadise/Go-Utils/conv"
	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/isr"
)

func TestReturnPandasSeriesToInterface(t *testing.T) {
	var dl insyra.IDataList
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

func TestReturnPolarsSeriesToInterface(t *testing.T) {
	var dl insyra.IDataList
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

func TestReturnNoneSetsInterfaceNilForSeriesPandas(t *testing.T) {
	var dl insyra.IDataList = insyra.NewDataList(1, 2)
	err := RunCode(&dl, `
insyra.Return(None)
`)
	if err != nil {
		t.Fatalf("RunCode: %v", err)
	}
	if dl != nil {
		t.Fatalf("expected dl to be nil, got %v", dl)
	}
}

func TestReturnNoneSetsInterfaceNilForDataTablePandas(t *testing.T) {
	var dt insyra.IDataTable = insyra.NewDataTable()
	err := RunCode(&dt, `
insyra.Return(None)
`)
	if err != nil {
		t.Fatalf("RunCode: %v", err)
	}
	if dt != nil {
		t.Fatalf("expected dt to be nil, got %v", dt)
	}
}

func TestRunCodeIntoIsrDL_PassPointer(t *testing.T) {
	dl := isr.UseDL(insyra.NewDataList())
	err := RunCode(dl, `
import pandas as pd
s = pd.Series([0, 1, 2], name="s1")
insyra.Return(s)
`)
	if err != nil {
		t.Fatalf("RunCode: %v", err)
	}
	if dl == nil || dl.DataList == nil {
		t.Fatalf("expected dl.DataList to be non-nil, got %v", dl)
	}
	if dl.Len() != 3 {
		t.Fatalf("len: %d", dl.Len())
	}
	if got := conv.ParseF64(dl.Get(1)); got != 1 {
		t.Fatalf("index 1: %v", got)
	}
}

func TestRunCodeIntoIsrDL_PassPointerToPointer(t *testing.T) {
	dl := isr.UseDL(insyra.NewDataList())
	dl = nil
	err := RunCode(&dl, `
import pandas as pd
s = pd.Series([5, 6, 7], name="s2")
insyra.Return(s)
`)
	if err != nil {
		t.Fatalf("RunCode: %v", err)
	}
	if dl == nil || dl.DataList == nil {
		t.Fatalf("expected dl.DataList to be non-nil, got %v", dl)
	}
	if dl.Len() != 3 {
		t.Fatalf("len: %d", dl.Len())
	}
	if got := conv.ParseF64(dl.Get(2)); got != 7 {
		t.Fatalf("index 2: %v", got)
	}
}

func TestRunCodeIntoIsrDT_PassPointer(t *testing.T) {
	dt := isr.UseDT(insyra.NewDataTable())
	err := RunCode(dt, `
import pandas as pd
df = pd.DataFrame({"A": [1, 2], "B": [3, 4]})
insyra.Return(df)
`)
	if err != nil {
		t.Fatalf("RunCode: %v", err)
	}
	if dt == nil || dt.DataTable == nil {
		t.Fatalf("expected dt.DataTable to be non-nil, got %v", dt)
	}
	if got := conv.ParseF64(dt.GetElement(1, "A")); got != 2 {
		t.Fatalf("A2: %v", got)
	}
}

func TestRunCodeIntoIsrDT_PassPointerToPointer(t *testing.T) {
	dt := isr.UseDT(insyra.NewDataTable())
	dt = nil
	err := RunCode(&dt, `
import pandas as pd
df = pd.DataFrame({"A": [5, 6], "B": [7, 8]})
insyra.Return(df)
`)
	if err != nil {
		t.Fatalf("RunCode: %v", err)
	}
	if dt == nil || dt.DataTable == nil {
		t.Fatalf("expected dt.DataTable to be non-nil, got %v", dt)
	}
	if got := conv.ParseF64(dt.GetElement(1, "B")); got != 8 {
		t.Fatalf("B2: %v", got)
	}
}
