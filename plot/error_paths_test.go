package plot

import (
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

// A radar chart with neither Indicators nor MaxValues used to end the program
// through LogFatal; it now logs a warning and returns nil.
func TestCreateRadarChartWithoutIndicatorsReturnsNil(t *testing.T) {
	quietLogs(t)

	chart := CreateRadarChart(RadarChartConfig{}, []RadarSeries{{Name: "s", Values: []float32{1, 2, 3}}})
	if chart != nil {
		t.Fatal("CreateRadarChart without indicators should return nil")
	}
}

// Calendar mode used to panic on a non-time X value or a missing CalendarOpts.
func TestCreateHeatMapCalendarMisuseReturnsNil(t *testing.T) {
	quietLogs(t)
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
}
