package plot

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

func quietLogs(t *testing.T) {
	t.Helper()
	level := insyra.Config.GetLogLevel()
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)
	t.Cleanup(func() { insyra.Config.SetLogLevel(level) })
}

// eachDontPanic runs f with Config.SetDontPanic off and then on.
func eachDontPanic(t *testing.T, f func(t *testing.T)) {
	t.Helper()
	for _, dontPanic := range []bool{false, true} {
		t.Run(fmt.Sprintf("dontPanic=%v", dontPanic), func(t *testing.T) {
			previous := insyra.Config.GetDontPanicStatus()
			insyra.Config.SetDontPanic(dontPanic)
			t.Cleanup(func() { insyra.Config.SetDontPanic(previous) })
			f(t)
		})
	}
}

// A radar chart with neither Indicators nor MaxValues ended the program
// through LogFatal on v0.3.2 unless SetDontPanic(true) was set; with it, the
// call went on and returned a chart with no indicators. Both configurations
// now log a warning and return that chart.
func TestCreateRadarChartWithoutIndicatorsReturnsTheChart(t *testing.T) {
	quietLogs(t)

	eachDontPanic(t, func(t *testing.T) {
		chart := CreateRadarChart(RadarChartConfig{Title: "no indicators"}, []RadarSeries{{Name: "s", Values: []float32{1, 2, 3}}})
		if chart == nil {
			t.Fatal("CreateRadarChart without indicators returned nil; v0.3.2 returned the chart")
		}
		if err := SaveHTML(chart, filepath.Join(t.TempDir(), "radar.html")); err != nil {
			t.Fatalf("SaveHTML: %v", err)
		}
	})
}

// Calendar mode panicked on a non-time X value or a missing CalendarOpts on
// v0.3.2, whatever SetDontPanic said. It now returns nil in both.
func TestCreateHeatMapCalendarMisuseReturnsNil(t *testing.T) {
	quietLogs(t)

	eachDontPanic(t, func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("CreateHeatMap panicked: %v", r)
			}
		}()

		if chart := CreateHeatMap(HeatMapConfig{UseCalendar: true}, HeatMapPoint(1, 1, 3.0)); chart != nil {
			t.Fatal("calendar mode with an int X should return nil")
		}
		day := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		if chart := CreateHeatMap(HeatMapConfig{UseCalendar: true}, HeatMapPoint(day, 0, 3.0)); chart != nil {
			t.Fatal("calendar mode without CalendarOpts should return nil")
		}
	})
}
