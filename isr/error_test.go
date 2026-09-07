package isr_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/isr"
)

func quietFatal(t *testing.T) {
	t.Helper()
	level := insyra.Config.GetLogLevel()
	panicOnError := insyra.Config.GetPanicOnError()
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)
	insyra.Config.SetPanicOnError(false)
	t.Cleanup(func() {
		insyra.Config.SetLogLevel(level)
		insyra.Config.SetPanicOnError(panicOnError)
	})
}

// I-1 / K-1: isr is the recommended entry point and is built for block
// syntax, so a failure must leave a chainable object carrying the error
// instead of ending the process.
func TestDTFromFailureIsChainable(t *testing.T) {
	quietFatal(t)

	dt := isr.DT.From(isr.CSV{FilePath: "no_such_file_for_insyra_test.csv"})
	if dt == nil {
		t.Fatal("DT.From returned nil for a missing file")
	}
	if dt.Err() == nil {
		t.Fatal("DT.From did not record the read failure")
	}
	// The chain continues without panicking.
	rows, cols := dt.Push(isr.Row{"a": 1}).Size()
	if rows < 0 || cols < 0 {
		t.Fatalf("unexpected size after a failed From: %dx%d", rows, cols)
	}
}

func TestDTFromUnexpectedTypeIsChainable(t *testing.T) {
	quietFatal(t)

	dt := isr.DT.From(struct{ X int }{1})
	if dt == nil {
		t.Fatal("DT.From returned nil for an unsupported type")
	}
	if dt.Err() == nil {
		t.Fatal("DT.From did not record the unsupported type")
	}
}

func TestColRowPushFailuresAreChainable(t *testing.T) {
	quietFatal(t)

	dt := isr.DT.From(isr.Row{"a": 1, "b": 2})
	if dt.Err() != nil {
		t.Fatalf("setup failed: %v", dt.Err())
	}

	col := dt.Col(3.5) // not an int, string, or name
	if col == nil {
		t.Fatal("Col returned nil for an unsupported selector")
	}
	if col.Err() == nil {
		t.Fatal("Col did not record the unsupported selector")
	}

	row := dt.Row(3.5)
	if row == nil {
		t.Fatal("Row returned nil for an unsupported selector")
	}
	if row.Err() == nil {
		t.Fatal("Row did not record the unsupported selector")
	}

	pushed := isr.DT.From(nil).Push(3.5)
	if pushed == nil {
		t.Fatal("Push returned nil for an unsupported type")
	}
	if pushed.Err() == nil {
		t.Fatal("Push did not record the unsupported type")
	}
}

func TestUseDLUseDTNeverReturnNil(t *testing.T) {
	quietFatal(t)

	if got := isr.UseDL(insyra.NewDataList(1, 2)); got == nil {
		t.Fatal("UseDL returned nil for a valid DataList")
	}
	if got := isr.UseDT(insyra.NewDataTable()); got == nil {
		t.Fatal("UseDT returned nil for a valid DataTable")
	}
	var nilList *insyra.DataList
	got := isr.UseDL(nilList)
	if got == nil {
		t.Fatal("UseDL returned nil for a nil DataList")
	}
	if got.Err() == nil {
		t.Fatal("UseDL did not record the nil input")
	}
	var nilTable *insyra.DataTable
	gotDT := isr.UseDT(nilTable)
	if gotDT == nil {
		t.Fatal("UseDT returned nil for a nil DataTable")
	}
	if gotDT.Err() == nil {
		t.Fatal("UseDT did not record the nil input")
	}
}
