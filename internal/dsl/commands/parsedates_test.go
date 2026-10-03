package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	insyra "github.com/HazelnutParadise/insyra"
)

func parsedatesTable() *insyra.DataTable {
	dt := insyra.NewDataTable()
	dt.AppendCols(
		namedList("Date", "2024-01-02", "2024-01-15", "2024-02-01"),
		namedList("Close", 10.0, 11.0, 20.0),
	)
	return dt
}

func TestRunParseDatesCommand_DataList(t *testing.T) {
	ctx := newTimeSeriesContext(t, map[string]any{"d": namedList("d", "2024-01-02", "nope")})
	if err := runParseDatesCommand(ctx, []string{"d", "as", "out"}); err != nil {
		t.Fatalf("runParseDatesCommand failed: %v", err)
	}
	out, ok := ctx.Vars["out"].(*insyra.DataList)
	if !ok {
		t.Fatalf("expected DataList, got %T", ctx.Vars["out"])
	}
	parsed, ok := out.Get(0).(time.Time)
	if !ok {
		t.Fatalf("element 0 = %T want time.Time", out.Get(0))
	}
	if !parsed.Equal(time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("element 0 = %v want 2024-01-02", parsed)
	}
	if out.Get(1) != nil {
		t.Errorf("element 1 = %v want nil", out.Get(1))
	}
	// The source variable must not be touched.
	source := ctx.Vars["d"].(*insyra.DataList)
	if _, isString := source.Get(0).(string); !isString {
		t.Errorf("source element 0 = %T want the original string", source.Get(0))
	}
}

func TestRunParseDatesCommand_DataListRejectsCols(t *testing.T) {
	ctx := newTimeSeriesContext(t, map[string]any{"d": namedList("d", "2024-01-02")})
	err := runParseDatesCommand(ctx, []string{"d", "cols", "Date"})
	if err == nil || !strings.HasPrefix(err.Error(), "parsedates: ") {
		t.Fatalf("expected a parsedates-prefixed error, got %v", err)
	}
}

func TestRunParseDatesCommand_DataTableCols(t *testing.T) {
	ctx := newTimeSeriesContext(t, map[string]any{"dt": parsedatesTable()})
	if err := runParseDatesCommand(ctx, []string{"dt", "cols", "Date", "as", "out"}); err != nil {
		t.Fatalf("runParseDatesCommand failed: %v", err)
	}
	out, ok := ctx.Vars["out"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("expected DataTable, got %T", ctx.Vars["out"])
	}
	if _, isTime := out.GetColByName("Date").Get(0).(time.Time); !isTime {
		t.Fatalf("Date[0] = %T want time.Time", out.GetColByName("Date").Get(0))
	}
	if got := out.GetColByName("Close").Get(0); got != 10.0 {
		t.Errorf("Close[0] = %v want 10", got)
	}
	// The source table is a separate copy and keeps its strings.
	source := ctx.Vars["dt"].(*insyra.DataTable)
	if _, isString := source.GetColByName("Date").Get(0).(string); !isString {
		t.Errorf("source Date[0] = %T want the original string", source.GetColByName("Date").Get(0))
	}
}

func TestRunParseDatesCommand_MultipleColsAndRepeatedLayout(t *testing.T) {
	dt := insyra.NewDataTable()
	dt.AppendCols(
		namedList("Start", "02/01/2024", "15/01/2024"),
		namedList("End", "2024-01-31", "2024-02-15"),
	)
	ctx := newTimeSeriesContext(t, map[string]any{"dt": dt})
	err := runParseDatesCommand(ctx, []string{
		"dt", "cols", "Start,End", "layout", "02/01/2006", "layout", "2006-01-02", "as", "out",
	})
	if err != nil {
		t.Fatalf("runParseDatesCommand failed: %v", err)
	}
	out := ctx.Vars["out"].(*insyra.DataTable)
	start, ok := out.GetColByName("Start").Get(0).(time.Time)
	if !ok {
		t.Fatalf("Start[0] = %T want time.Time", out.GetColByName("Start").Get(0))
	}
	if !start.Equal(time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Start[0] = %v want 2024-01-02", start)
	}
	end, ok := out.GetColByName("End").Get(1).(time.Time)
	if !ok {
		t.Fatalf("End[1] = %T want time.Time", out.GetColByName("End").Get(1))
	}
	if !end.Equal(time.Date(2024, time.February, 15, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("End[1] = %v want 2024-02-15", end)
	}
}

func TestRunParseDatesCommand_DefaultsToResultVar(t *testing.T) {
	ctx := newTimeSeriesContext(t, map[string]any{"d": namedList("d", "2024-01-02")})
	if err := runParseDatesCommand(ctx, []string{"d"}); err != nil {
		t.Fatalf("runParseDatesCommand failed: %v", err)
	}
	if _, ok := ctx.Vars["$result"].(*insyra.DataList); !ok {
		t.Fatalf("expected $result DataList, got %T", ctx.Vars["$result"])
	}
}

func TestRunParseDatesCommand_TableWithoutCols(t *testing.T) {
	ctx := newTimeSeriesContext(t, map[string]any{"dt": parsedatesTable()})
	err := runParseDatesCommand(ctx, []string{"dt"})
	if err == nil {
		t.Fatal("expected an error when cols is missing on a DataTable")
	}
	message := err.Error()
	if !strings.HasPrefix(message, "parsedates: ") {
		t.Errorf("error %q should carry the parsedates: prefix", message)
	}
	if !strings.Contains(message, "cols") {
		t.Errorf("error %q should say that cols is required", message)
	}
	if _, exists := ctx.Vars["$result"]; exists {
		t.Error("no variable should be stored on error")
	}
}

func TestRunParseDatesCommand_UnknownOption(t *testing.T) {
	ctx := newTimeSeriesContext(t, map[string]any{"d": namedList("d", "2024-01-02")})
	err := runParseDatesCommand(ctx, []string{"d", "format", "2006-01-02"})
	if err == nil || !strings.Contains(err.Error(), "unknown option") {
		t.Fatalf("expected an unknown option error, got %v", err)
	}
}

func TestRunParseDatesCommand_OptionWithoutValue(t *testing.T) {
	ctx := newTimeSeriesContext(t, map[string]any{"d": namedList("d", "2024-01-02")})
	for _, args := range [][]string{{"d", "layout"}, {"d", "cols"}} {
		if err := runParseDatesCommand(ctx, args); err == nil {
			t.Errorf("expected an error for %v", args)
		}
	}
}

func TestRunParseDatesCommand_NotAListOrTable(t *testing.T) {
	ctx := newTimeSeriesContext(t, map[string]any{"n": 1.5})
	err := runParseDatesCommand(ctx, []string{"n"})
	if err == nil || !strings.HasPrefix(err.Error(), "parsedates: ") {
		t.Fatalf("expected a parsedates-prefixed error, got %v", err)
	}
}

func TestRunParseDatesCommand_MissingVariable(t *testing.T) {
	ctx := newTimeSeriesContext(t, nil)
	err := runParseDatesCommand(ctx, []string{"nope"})
	if err == nil || !strings.HasPrefix(err.Error(), "parsedates: ") {
		t.Fatalf("expected a parsedates-prefixed error, got %v", err)
	}
}

func TestRunParseDatesCommand_Usage(t *testing.T) {
	ctx := newTimeSeriesContext(t, nil)
	err := runParseDatesCommand(ctx, nil)
	if err == nil || !strings.Contains(err.Error(), "usage: parsedates") {
		t.Fatalf("expected usage error, got %v", err)
	}
}

// End-to-end: the CSV path that AGENTS.md recorded as unreachable — load a CSV
// whose Date column is a string, convert it, then resample into monthly bars.
func TestParseDates_CSVToMonthlyBarsScript(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "bars.csv")
	csv := "Date,Close\n2024-01-02,10\n2024-01-15,11\n2024-02-01,20\n"
	if err := os.WriteFile(csvPath, []byte(csv), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}
	scriptPath := filepath.Join(dir, "bars.isr")
	script := strings.Join([]string{
		"load " + csvPath + " as dt",
		"parsedates dt cols Date as dt",
		"resample dt Date monthly Close:last as m",
	}, "\n") + "\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	ctx := newTimeSeriesContext(t, nil)
	if err := runScriptCommand(ctx, []string{scriptPath}); err != nil {
		t.Fatalf("runScriptCommand failed: %v", err)
	}
	output := ctx.Output.(*bytes.Buffer).String()
	if strings.Contains(output, "line ") {
		t.Fatalf("script reported an error:\n%s", output)
	}
	m, ok := ctx.Vars["m"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("expected DataTable in m, got %T", ctx.Vars["m"])
	}
	if rows := m.NumRows(); rows != 2 {
		t.Fatalf("monthly rows = %d want 2\n%s", rows, output)
	}
}
