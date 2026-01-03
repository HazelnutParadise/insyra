// plot/heatmap.go

package plot

import (
	"fmt"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/plot/internal"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// heapMapAxisValue constrains allowed axis types for strict generics.
// We allow int, string, and time.Time.
type heapMapAxisValue interface {
	int | string | time.Time
}

// HeatMapConfig defines the configuration for a heatmap chart.
// It supports two modes:
//   - grid heatmap: X/Y are either indices (int) or labels (string)
//   - calendar heatmap: X should be time.Time and Y is ignored
type HeatMapConfig struct {
	Width           string // Width of the chart (default "900px").
	Height          string // Height of the chart (default "500px").
	BackgroundColor string // Background color of the chart (default "white").
	Theme           Theme  // Theme of the chart.
	Title           string
	Subtitle        string
	TitlePos        Position // Optional: Use const PositionXXX.

	XAxis []string // X-axis data (optional for label-based points).
	YAxis []string // Y-axis data (optional for label-based points).

	// Colors for the visual map (optional)
	Colors []string

	// Min/Max for visual map: if nil, computed from data values
	Min *float64
	Max *float64

	// Calendar mode: if true, `CalendarOpts` will be used and points' X are formatted as dates
	UseCalendar  bool
	CalendarOpts *opts.Calendar
}

// heatMapPoint represents a single heatmap datum with constrained axis types.
// X/Y types are enforced by generics; Value is a non-pointer and Valid indicates presence.
// e.g. heatMapPoint[int,string] or heatMapPoint[time.Time, int]
type heatMapPoint[X heapMapAxisValue, Y heapMapAxisValue] struct {
	X     X
	Y     Y
	Value float64
	Valid bool
}

// HeatMapPoint creates a valid point with a numeric value.
func HeatMapPoint[X heapMapAxisValue, Y heapMapAxisValue](x X, y Y, value float64) heatMapPoint[X, Y] {
	return heatMapPoint[X, Y]{X: x, Y: y, Value: value, Valid: true}
}

// HeatMapMissingPoint creates a point that is considered missing (renders as "-").
func HeatMapMissingPoint[X heapMapAxisValue, Y heapMapAxisValue](x X, y Y) heatMapPoint[X, Y] {
	return heatMapPoint[X, Y]{X: x, Y: y, Valid: false}
}

// CreateHeatMap generates and returns a *charts.HeatMap object based on HeatmapConfig.
// It accepts optional variadic heatMapPoint arguments which will be appended to `config.Data`.
func CreateHeatMap[X heapMapAxisValue, Y heapMapAxisValue](config HeatMapConfig, points ...heatMapPoint[X, Y]) *charts.HeatMap {
	if len(points) == 0 {
		insyra.LogWarning("plot.heatmap", "CreateHeatMap", "no data points provided; returning nil")
		return nil
	}
	hm := charts.NewHeatMap()

	internal.SetBaseChartGlobalOptions(hm, internal.BaseChartConfig{
		Width:           config.Width,
		Height:          config.Height,
		BackgroundColor: config.BackgroundColor,
		Theme:           string(config.Theme),
		Title:           config.Title,
		Subtitle:        config.Subtitle,
		TitlePos:        string(config.TitlePos),
		HideLegend:      true,
		LegendPos:       "",
	})

	// Use beautiful default heatmap gradient colors if not specified
	if config.Colors == nil {
		config.Colors = []string{"#E74C3C", "#F39C12", "#50C878", "#4A90E2"}
	}

	// Compute Min/Max if not provided
	minVal, maxVal := computeMinMax[X, Y](points)
	if config.Min != nil {
		minVal = *config.Min
	}
	if config.Max != nil {
		maxVal = *config.Max
	}

	// Calendar mode: add calendar options and use coordinate system
	if config.UseCalendar {
		// Enforce that X is time.Time in calendar mode
		for _, p := range points {
			switch any(p.X).(type) {
			case time.Time:
				// ok
			default:
				panic("CreateHeatMap: calendar mode requires X axis values to be time.Time")
			}
		}
		if config.CalendarOpts == nil {
			panic("CreateHeatMap: calendar mode requires CalendarOpts to be set")
		}
		if config.CalendarOpts.ItemStyle == nil {
			config.CalendarOpts.ItemStyle = &opts.ItemStyle{BorderWidth: 0.5}
		}

		hm.SetGlobalOptions(
			charts.WithTitleOpts(opts.Title{
				Title:    config.Title,
				Subtitle: config.Subtitle,
			}),
			charts.WithVisualMapOpts(opts.VisualMap{
				Min:     float32(minVal),
				Max:     float32(maxVal),
				InRange: &opts.VisualMapInRange{Color: config.Colors},
			}),
		)

		hm.AddCalendar(config.CalendarOpts).AddSeries("heatmap calendar", convertToCalendarHeatMapData[X, Y](points), charts.WithCoordinateSystem("calendar"))
		return hm
	}

	// Regular grid heatmap
	hm.SetGlobalOptions(
		charts.WithXAxisOpts(opts.XAxis{
			Type:      "category",
			Data:      config.XAxis,
			SplitArea: &opts.SplitArea{Show: opts.Bool(true)},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Type:      "category",
			Data:      config.YAxis,
			SplitArea: &opts.SplitArea{Show: opts.Bool(true)},
		}),
		charts.WithVisualMapOpts(opts.VisualMap{
			Calculable: opts.Bool(true),
			Min:        float32(minVal),
			Max:        float32(maxVal),
			InRange: &opts.VisualMapInRange{
				Color: config.Colors,
			},
			Bottom: "0%",
			Right:  "0%",
		}),
	)

	// Add heatmap data
	hm.SetXAxis(config.XAxis).AddSeries("heatmap", convertToHeatMapData[X, Y](points))
	return hm
}

// convertToHeatMapData converts the heatmap data into the format needed by go-echarts.
// Handles axis types int, string and time.Time (time.Time formatted as YYYY-MM-DD).
func convertToHeatMapData[X heapMapAxisValue, Y heapMapAxisValue](data []heatMapPoint[X, Y]) []opts.HeatMapData {
	items := make([]opts.HeatMapData, 0, len(data))
	for _, p := range data {
		var x any
		switch t := any(p.X).(type) {
		case time.Time:
			x = t.Format("2006-01-02")
		case string:
			x = t
		case int:
			x = t
		default:
			x = fmt.Sprint(t)
		}

		var y any
		switch t := any(p.Y).(type) {
		case time.Time:
			y = t.Format("2006-01-02")
		case string:
			y = t
		case int:
			y = t
		default:
			y = fmt.Sprint(t)
		}

		var v any
		if !p.Valid {
			v = "-"
		} else {
			v = p.Value
		}
		items = append(items, opts.HeatMapData{Value: [3]any{x, y, v}})
	}
	return items
}

// convertToCalendarHeatMapData converts points to calendar style heatmap data where each item is [date, value]
// X is formatted as YYYY-MM-DD if it's a time.Time; otherwise it's stringified.
func convertToCalendarHeatMapData[X heapMapAxisValue, Y heapMapAxisValue](data []heatMapPoint[X, Y]) []opts.HeatMapData {
	items := make([]opts.HeatMapData, 0, len(data))
	for _, p := range data {
		var date any
		switch t := any(p.X).(type) {
		case time.Time:
			date = t.Format("2006-01-02")
		case string:
			date = t
		case int:
			date = fmt.Sprint(t)
		default:
			date = fmt.Sprint(t)
		}

		var v any
		if !p.Valid {
			v = "-"
		} else {
			v = p.Value
		}
		items = append(items, opts.HeatMapData{Value: [2]any{date, v}})
	}
	return items
}

// computeMinMax returns the min and max among numeric values in points. If no numeric values exist, returns 0 and 1.
func computeMinMax[X heapMapAxisValue, Y heapMapAxisValue](data []heatMapPoint[X, Y]) (float64, float64) {
	var minVal float64 = 0
	var maxVal float64 = 1
	found := false
	for _, p := range data {
		if !p.Valid {
			continue
		}
		v := p.Value
		if !found {
			minVal = v
			maxVal = v
			found = true
			continue
		}
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	return minVal, maxVal
}
