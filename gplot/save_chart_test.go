package gplot

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
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

// PL-2: a chart writer that cannot report a failed write is unusable in a
// pipeline, and it must not end the process either.
func TestSaveChartReturnsError(t *testing.T) {
	quietFatal(t)

	plt := CreateBarChart(BarChartConfig{
		Title: "t",
		XAxis: []string{"a", "b"},
	}, []float64{1, 2})
	if plt == nil {
		t.Fatal("CreateBarChart returned nil for valid input")
	}

	ok := filepath.Join(t.TempDir(), "chart.png")
	if err := SaveChart(plt, ok); err != nil {
		t.Fatalf("SaveChart failed for a writable path: %v", err)
	}

	err := SaveChart(plt, filepath.Join(t.TempDir(), "no", "such", "dir", "chart.png"))
	if err == nil {
		t.Fatal("SaveChart returned nil for an unwritable path")
	}
	if !strings.Contains(err.Error(), "chart.png") {
		t.Fatalf("error should name the file: %v", err)
	}
}

// SEC-6: a zero-value config must not panic.
func TestCreateHistogramZeroConfigDoesNotPanic(t *testing.T) {
	quietFatal(t)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("CreateHistogram panicked: %v", r)
		}
	}()
	if plt := CreateHistogram(HistogramConfig{}, insyra.NewDataList(1.0, 2.0, 3.0)); plt == nil {
		t.Fatal("CreateHistogram returned nil for a zero-value config; it should pick a default bin count")
	}
}
