package commands

import (
	"reflect"
	"strings"
	"testing"
	"time"

	insyra "github.com/HazelnutParadise/insyra"
)

func resampleTestTable() *insyra.DataTable {
	day := func(y int, m time.Month, d int) time.Time {
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}
	date := insyra.NewDataList(
		day(2024, time.January, 2), day(2024, time.January, 15), day(2024, time.January, 31),
		day(2024, time.February, 1), day(2024, time.February, 20),
	)
	date.SetName("Date")
	open := insyra.NewDataList(10.0, 12.0, 11.0, 20.0, 21.0)
	open.SetName("Open")
	high := insyra.NewDataList(13.0, 14.0, 12.0, 22.0, 25.0)
	high.SetName("High")
	low := insyra.NewDataList(9.0, 11.0, 10.5, 19.0, 20.0)
	low.SetName("Low")
	closeCol := insyra.NewDataList(12.0, 11.5, 11.8, 21.0, 24.0)
	closeCol.SetName("Close")
	volume := insyra.NewDataList(100, 200, 300, 400, 500)
	volume.SetName("Volume")

	dt := insyra.NewDataTable()
	dt.AppendCols(date, open, high, low, closeCol, volume)
	return dt
}

func TestRunResampleCommand_MonthlyOHLCV(t *testing.T) {
	ctx := newTimeSeriesContext(t, map[string]any{"dt": resampleTestTable()})
	err := runResampleCommand(ctx, []string{
		"dt", "Date", "monthly",
		"Open:first", "High:max", "Low:min", "Close:last:MonthClose", "Volume:sum",
		"as", "m",
	})
	if err != nil {
		t.Fatalf("runResampleCommand failed: %v", err)
	}
	m, ok := ctx.Vars["m"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("expected DataTable, got %T", ctx.Vars["m"])
	}
	if m.NumRows() != 2 {
		t.Fatalf("rows = %d want 2", m.NumRows())
	}
	want := []string{"Date", "Open", "High", "Low", "MonthClose", "Volume"}
	if got := m.ColNames(); !reflect.DeepEqual(got, want) {
		t.Fatalf("columns = %v want %v", got, want)
	}
	firstOpen, _ := insyra.ToFloat64Safe(m.GetColByName("Open").Data()[0])
	if firstOpen != 10 {
		t.Errorf("January Open = %v want 10", firstOpen)
	}
	janClose, _ := insyra.ToFloat64Safe(m.GetColByName("MonthClose").Data()[0])
	if janClose != 11.8 {
		t.Errorf("January MonthClose = %v want 11.8", janClose)
	}
	febVolume, _ := insyra.ToFloat64Safe(m.GetColByName("Volume").Data()[1])
	if febVolume != 900 {
		t.Errorf("February Volume = %v want 900", febVolume)
	}
}

func TestRunResampleCommand_DefaultsToResultVar(t *testing.T) {
	ctx := newTimeSeriesContext(t, map[string]any{"dt": resampleTestTable()})
	if err := runResampleCommand(ctx, []string{"dt", "Date", "yearly", "Volume:sum"}); err != nil {
		t.Fatalf("runResampleCommand failed: %v", err)
	}
	result, ok := ctx.Vars["$result"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("expected $result DataTable, got %T", ctx.Vars["$result"])
	}
	if result.NumRows() != 1 {
		t.Errorf("yearly rows = %d want 1", result.NumRows())
	}
}

func TestRunResampleCommand_UnknownOpListsOperators(t *testing.T) {
	ctx := newTimeSeriesContext(t, map[string]any{"dt": resampleTestTable()})
	err := runResampleCommand(ctx, []string{"dt", "Date", "monthly", "Close:average"})
	if err == nil {
		t.Fatalf("expected an error for an unknown op")
	}
	message := err.Error()
	if !strings.HasPrefix(message, "resample: ") {
		t.Errorf("error %q should carry the resample: prefix", message)
	}
	for _, op := range []string{"sum", "mean", "median", "min", "max", "count", "first", "last", "std", "var"} {
		if !strings.Contains(message, op) {
			t.Errorf("error %q should list the available operator %q", message, op)
		}
	}
	if _, exists := ctx.Vars["$result"]; exists {
		t.Errorf("no variable should be stored on error")
	}
}

func TestRunResampleCommand_UnknownFrequency(t *testing.T) {
	ctx := newTimeSeriesContext(t, map[string]any{"dt": resampleTestTable()})
	err := runResampleCommand(ctx, []string{"dt", "Date", "daily", "Close:last"})
	if err == nil || !strings.Contains(err.Error(), "unknown frequency") {
		t.Fatalf("expected unknown frequency error, got %v", err)
	}
}

func TestRunResampleCommand_BadSpecShape(t *testing.T) {
	ctx := newTimeSeriesContext(t, map[string]any{"dt": resampleTestTable()})
	for _, spec := range []string{"Close:last:MonthClose:extra", "Close"} {
		err := runResampleCommand(ctx, []string{"dt", "Date", "monthly", spec})
		if err == nil {
			t.Fatalf("expected an error for spec %q", spec)
		}
		if !strings.Contains(err.Error(), "<col>:<op>[:<name>]") {
			t.Errorf("error for %q = %q, should show the expected shape", spec, err)
		}
	}
}

func TestRunResampleCommand_NonTimeColumn(t *testing.T) {
	dt := insyra.NewDataTable()
	date := insyra.NewDataList("2024-01-02", "2024-01-15")
	date.SetName("Date")
	closeCol := insyra.NewDataList(1.0, 2.0)
	closeCol.SetName("Close")
	dt.AppendCols(date, closeCol)

	ctx := newTimeSeriesContext(t, map[string]any{"dt": dt})
	err := runResampleCommand(ctx, []string{"dt", "Date", "monthly", "Close:last"})
	if err == nil {
		t.Fatalf("expected an error for a non-time column")
	}
	message := err.Error()
	if !strings.HasPrefix(message, "resample: ") {
		t.Errorf("error %q should carry the resample: prefix", message)
	}
	if !strings.Contains(message, "row 1") || !strings.Contains(message, "time.Time") {
		t.Errorf("error %q should carry the library's row-numbered message", message)
	}
}

func TestRunResampleCommand_NotADataTable(t *testing.T) {
	ctx := newTimeSeriesContext(t, map[string]any{"x": insyra.NewDataList(1, 2, 3)})
	err := runResampleCommand(ctx, []string{"x", "Date", "monthly", "Close:last"})
	if err == nil || !strings.HasPrefix(err.Error(), "resample: ") {
		t.Fatalf("expected resample-prefixed error, got %v", err)
	}
}

func TestRunResampleCommand_Usage(t *testing.T) {
	ctx := newTimeSeriesContext(t, map[string]any{"dt": resampleTestTable()})
	err := runResampleCommand(ctx, []string{"dt", "Date", "monthly"})
	if err == nil || !strings.Contains(err.Error(), "usage: resample") {
		t.Fatalf("expected usage error, got %v", err)
	}
}
