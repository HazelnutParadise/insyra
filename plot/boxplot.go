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

	XAxis     []string // X-axis data.
	XAxisName string   // Optional: X-axis name.
	YAxisName string   // Optional: Y-axis name.
	// Y axis customization (numeric-only: min/max/split/formatter)
	YAxisMin         *float64 // Optional: minimum value of Y axis.
	YAxisMax         *float64 // Optional: maximum value of Y axis.
	YAxisSplitNumber *int     // Optional: split number for Y axis.
	YAxisFormatter   string   // Optional: label formatter for Y axis, e.g. "{value}°C".
}

// CreateBoxPlot generates and returns a *charts.BoxPlot object
func CreateBoxPlot(config BoxPlotConfig, series ...BoxPlotSeries) *charts.BoxPlot {
	if len(series) == 0 {
		insyra.LogWarning("plot", "CreateBoxPlot", "no series provided in BoxPlotConfig.Series; returning nil")
		return nil
	}
	boxPlot := charts.NewBoxPlot()

	internal.SetBaseChartGlobalOptions(boxPlot, internal.BaseChartConfig{
		Width:           config.Width,
		Height:          config.Height,
		BackgroundColor: config.BackgroundColor,
		Theme:           string(config.Theme),
		Title:           sanitizeChartText(config.Title),
		Subtitle:        sanitizeChartText(config.Subtitle),
		TitlePos:        string(config.TitlePos),
		HideLegend:      config.HideLegend,
		LegendPos:       string(config.LegendPos),
	})

	// Determine number of categories ONCE, before adding any series, as the
	// smallest series length so no series ends up with empty items. Keep the X
	// axis consistent with that count. (Previously numCats was based on the first
	// series and then mutated inside the add-loop, making the result depend on
	// series order, while the in-loop XAxis reslice was dead code because
	// SetXAxis had already been called.)
	numCats := len(series[0].Data)
	for _, s := range series {
		if len(s.Data) < numCats {
			numCats = len(s.Data)
		}
	}
	if len(config.XAxis) > 0 && len(config.XAxis) < numCats {
		numCats = len(config.XAxis)
	}

	if len(config.XAxis) == 0 {
		config.XAxis = []string{}
		for i := 0; i < numCats; i++ {
			config.XAxis = append(config.XAxis, fmt.Sprintf("Category %d", i+1))
		}
	} else if len(config.XAxis) > numCats {
		config.XAxis = config.XAxis[:numCats]
	}

	// Set X axis and add each series (support per-series color/fill)
	boxPlot.SetXAxis(config.XAxis)

	// Apply default colors to series that don't have colors specified
	colorIndex := 0
	for i := range series {
		if series[i].Color == "" {
			series[i].Color = internal.GetColor(colorIndex)
			colorIndex++
		}
	}

	for _, s := range series {
		if numCats == 0 {
			continue
		}
		// numCats is already the min across all series, so this only trims longer
		// series to the fixed category count (no order-dependent mutation).
		items := s.Data
		if len(items) > numCats {
			items = items[:numCats]
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

	// Apply Y axis settings via internal helper (flatten series data for detection)
	allData := make([]insyra.IDataList, 0)
	for _, s := range series {
		allData = append(allData, s.Data...)
	}
	// Apply shared Y axis logic (numeric-only for boxplot)
	internal.ApplyYAxis(boxPlot, config.YAxisName, nil, config.YAxisMin, config.YAxisMax, config.YAxisSplitNumber, config.YAxisFormatter, allData...)

	boxPlot.SetGlobalOptions(
		charts.WithXAxisOpts(opts.XAxis{
			Name: config.XAxisName,
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
