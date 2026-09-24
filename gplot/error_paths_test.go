package gplot

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func quietLogs(t *testing.T) {
	t.Helper()
	level := insyra.Config.GetLogLevel()
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)
	t.Cleanup(func() { insyra.Config.SetLogLevel(level) })
}

func mustNotPanic(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked: %v", name, r)
		}
	}()
	f()
}

// SEC-6: a zero-value config (or a negative bin count) must not panic.
func TestCreateHistogramZeroConfigDoesNotPanic(t *testing.T) {
	quietLogs(t)

	for _, bins := range []int{0, -3} {
		mustNotPanic(t, "CreateHistogram", func() {
			if plt := CreateHistogram(HistogramConfig{Bins: bins}, insyra.NewDataList(1.0, 2.0, 3.0)); plt == nil {
				t.Fatalf("CreateHistogram returned nil for Bins=%d; it should pick the default bin count", bins)
			}
		})
	}
}

// A line or step series gonum refuses (NaN) is logged and left out instead of
// panicking.
func TestSeriesThatCannotBeBuiltDoesNotPanic(t *testing.T) {
	quietLogs(t)

	mustNotPanic(t, "CreateLineChart", func() {
		plt := CreateLineChart(LineChartConfig{XAxis: []float64{0, 1, 2}}, map[string][]float64{"s": {1, math.NaN(), 3}})
		if plt == nil {
			t.Fatal("CreateLineChart returned nil; only the bad series should be left out")
		}
	})
	mustNotPanic(t, "CreateStepChart", func() {
		plt := CreateStepChart(StepChartConfig{XAxis: []float64{0, 1, 2}}, map[string][]float64{"s": {1, math.NaN(), 3}})
		if plt == nil {
			t.Fatal("CreateStepChart returned nil; only the bad series should be left out")
		}
	})
}
