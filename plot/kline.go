// plot/kline.go

package plot

import (
	"sort"

	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/utils"
	"github.com/HazelnutParadise/insyra/plot/internal"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

type KlinePoint struct {
	Date  time.Time `json:"date"` // Timestamp of the data point
	Open  float64   `json:"open"`
	High  float64   `json:"high"`
	Low   float64   `json:"low"`
	Close float64   `json:"close"`
}

// KlineChartConfig defines the configuration for a K-line chart.
type KlineChartConfig struct {
	Width           string   // Width of the chart (default "900px").
	Height          string   // Height of the chart (default "500px").
	BackgroundColor string   // Background color of the chart (default "white").
	Theme           Theme    // Theme of the chart.
	Title           string   // Title of the chart.
	Subtitle        string   // Subtitle of the chart.
	TitlePos        Position // Optional: Use const PositionXXX.

	// DateFormat controls how dates are displayed on the X axis. Use Go time format strings or
	// common patterns like "YYYY-MM-DD HH:mm:ss" which will be converted automatically.
	DateFormat string
	DataZoom   bool // Turn on/off data zoom
}

// CreateKlineChart generates and returns a *charts.Kline object.
func CreateKlineChart(config KlineChartConfig, klinePoints ...KlinePoint) *charts.Kline {
	if len(klinePoints) == 0 {
		insyra.LogWarning("plot", "CreateKlineChart", "No data available for kline chart. Returning nil.")
		return nil
	}
	klineChart := charts.NewKLine()

	// Set title and subtitle
	internal.SetBaseChartGlobalOptions(klineChart, internal.BaseChartConfig{
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

	// Set title and subtitle
	klineChart.SetGlobalOptions(
		charts.WithXAxisOpts(opts.XAxis{
			SplitNumber: 20,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Scale: opts.Bool(true),
		}),
	)

	// Enable data zoom if needed
	if config.DataZoom {
		klineChart.SetGlobalOptions(
			charts.WithDataZoomOpts(opts.DataZoom{
				Start:      50,
				End:        100,
				XAxisIndex: []int{0},
			}),
		)
	}

	var xAxis []string
	var series []opts.KlineData

	// Sort by date ascending
	sort.Slice(klinePoints, func(i, j int) bool {
		return klinePoints[i].Date.Before(klinePoints[j].Date)
	})

	// Determine date format (fall back to default if empty). Accepts Go format or common
	// patterns (e.g. "YYYY-MM-DD") which are converted using internal utils.ConvertDateFormat.
	df := config.DateFormat
	if df == "" {
		df = "2006-01-02 15:04:05"
	} else {
		// Convert common patterns like YYYY-MM-DD to Go time formats
		df = utils.ConvertDateFormat(df)
	}

	// Build x axis and series
	xAxis = make([]string, 0, len(klinePoints))
	series = make([]opts.KlineData, 0, len(klinePoints))
	for _, pt := range klinePoints {
		xAxis = append(xAxis, pt.Date.Format(df))
		series = append(series, opts.KlineData{
			Value: [4]float32{
				float32(pt.Open),  // Open
				float32(pt.Close), // Close
				float32(pt.Low),   // Lowest
				float32(pt.High),  // Highest
			},
		})
	}

	// Set X axis and add series data
	klineChart.SetXAxis(xAxis).AddSeries("Kline", series)

	return klineChart
}
