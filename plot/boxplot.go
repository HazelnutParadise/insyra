// plot/boxplot.go

package plot

import (
	"fmt"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/plot/internal"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// BoxPlotSeries defines a single series in a box plot.
type BoxPlotSeries struct {
	Name  string
	Data  []insyra.IDataList
	Color string // Optional: per-series color
	Fill  bool   // Optional: whether to fill boxes (default false)
}

// BoxPlotConfig defines the configuration for a box plot chart.
type BoxPlotConfig struct {
	Width           string // Width of the chart (default "900px").
	Height          string // Height of the chart (default "500px").
	BackgroundColor string // Background color of the chart (default "white").
	Theme           Theme  // Theme of the chart.
	Title           string
	Subtitle        string
	TitlePos        Position // Optional: Use const PositionXXX.
	HideLegend      bool     // Whether to hide the legend.
	LegendPos       Position // Optional: Use const PositionXXX.

	// Preferred series representation (ordered)
	Series    []BoxPlotSeries // Ordered series with optional per-series metadata
	XAxis     []string        // X-axis data.
	XAxisName string          // Optional: X-axis name.
	YAxisName string          // Optional: Y-axis name.
}

// CreateBoxPlot generates and returns a *charts.BoxPlot object
func CreateBoxPlot(config BoxPlotConfig) *charts.BoxPlot {
	boxPlot := charts.NewBoxPlot()

	internal.SetBaseChartGlobalOptions(boxPlot, internal.BaseChartConfig{
		Width:           config.Width,
		Height:          config.Height,
		BackgroundColor: config.BackgroundColor,
		Theme:           string(config.Theme),
		Title:           config.Title,
		Subtitle:        config.Subtitle,
		TitlePos:        string(config.TitlePos),
		HideLegend:      config.HideLegend,
		LegendPos:       string(config.LegendPos),
	})

	// Use provided ordered Series (no backwards compatibility for map-based Data)
	seriesList := config.Series
	if len(seriesList) == 0 {
		insyra.LogWarning("plot.boxplot", "CreateBoxPlot", "no series provided in BoxPlotConfig.Series; returning empty chart")
		return boxPlot
	}

	// Determine number of categories
	numCats := 0
	if len(config.XAxis) > 0 {
		numCats = len(config.XAxis)
	} else if len(seriesList) > 0 {
		numCats = len(seriesList[0].Data)
	}

	// If XAxis not provided, create default labels
	if len(config.XAxis) == 0 {
		config.XAxis = []string{}
		for i := 0; i < numCats; i++ {
			config.XAxis = append(config.XAxis, fmt.Sprintf("Category %d", i+1))
		}
	}

	// Set X axis and add each series (support per-series color/fill)
	boxPlot.SetXAxis(config.XAxis)
	for _, s := range seriesList {
		if numCats == 0 {
			continue
		}
		// ensure we only use up to numCats items
		items := s.Data
		if len(items) > numCats {
			items = items[:numCats]
		}
		if len(items) < numCats {
			// fallback: truncate numCats to len(items) to avoid empty items
			insyra.LogWarning("plot.boxplot", "CreateBoxPlot", "series %s has %d categories but expected %d; truncating to %d", s.Name, len(items), numCats, len(items))
			numCats = len(items)
			if len(config.XAxis) > numCats {
				config.XAxis = config.XAxis[:numCats]
			}
		}
		boxPlotItems := generateBoxPlotItemsFromIDataList(items)

		// decide color and fill (no global defaults: rely on per-series settings)
		color := s.Color
		fill := s.Fill

		if color != "" {
			if fill {
				boxPlot.AddSeries(s.Name, boxPlotItems, charts.WithItemStyleOpts(opts.ItemStyle{
					Color:       color,
					BorderColor: color,
				}))
			} else {
				boxPlot.AddSeries(s.Name, boxPlotItems, charts.WithItemStyleOpts(opts.ItemStyle{
					Color:       "transparent",
					BorderColor: color,
				}))
			}
		} else {
			if fill {
				boxPlot.AddSeries(s.Name, boxPlotItems)
			} else {
				boxPlot.AddSeries(s.Name, boxPlotItems, charts.WithItemStyleOpts(opts.ItemStyle{
					Color: "transparent",
				}))
			}
		}
	}

	// Set X, Y axis names
	boxPlot.SetGlobalOptions(
		charts.WithXAxisOpts(opts.XAxis{
			Name: config.XAxisName,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: config.YAxisName,
		}),
	)

	return boxPlot
}

// createBoxPlotData generates the five-number summary (Min, Q1, Q2, Q3, Max)
func createBoxPlotData(data insyra.IDataList) []float64 {
	return []float64{
		data.Min(),
		data.Quartile(1),
		data.Quartile(2),
		data.Quartile(3),
		data.Max(),
	}
}

// generateBoxPlotItemsFromIDataList generates BoxPlotData from []insyra.IDataList
func generateBoxPlotItemsFromIDataList(dataLists []insyra.IDataList) []opts.BoxPlotData {
	items := make([]opts.BoxPlotData, len(dataLists))
	for i, dataList := range dataLists {
		items[i] = opts.BoxPlotData{Value: createBoxPlotData(dataList)}
	}
	return items
}
