package plot

import (
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/plot/internal"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

// ScatterSeries is one set of points for CreateScatterChart: point i is
// (X[i], Y[i]), and Name labels the series in the legend. Two columns of a
// table are one series: ScatterSeries{Name: "s", X: dt.GetColByName("x"),
// Y: dt.GetColByName("y")}.
type ScatterSeries struct {
	Name string
	X    insyra.IDataList
	Y    insyra.IDataList
}

// ScatterChartConfig defines the configuration for a scatter chart.
type ScatterChartConfig struct {
	Width           string   // Width of the chart (default "900px").
	Height          string   // Height of the chart (default "500px").
	BackgroundColor string   // Background color of the chart (default "white").
	Theme           Theme    // Theme of the chart.
	Title           string   // Title of the chart.
	Subtitle        string   // Subtitle of the chart.
	TitlePos        Position // Optional: Use const PositionXXX.
	HideLegend      bool     // Optional: Whether to hide the legend.
	LegendPos       Position // Optional: Use const PositionXXX.

	XAxisName        string   // Optional: X-axis name.
	XAxisMin         *float64 // Optional: minimum value of X axis.
	XAxisMax         *float64 // Optional: maximum value of X axis.
	XAxisSplitNumber *int     // Optional: split number for X axis.
	XAxisFormatter   string   // Optional: label formatter for X axis, e.g. "{value}°C".

	YAxisName        string        // Optional: Y-axis name.
	YAxisMin         *float64      // Optional: minimum value of Y axis.
	YAxisMax         *float64      // Optional: maximum value of Y axis.
	YAxisSplitNumber *int          // Optional: split number for Y axis.
	YAxisFormatter   string        // Optional: label formatter for Y axis, e.g. "{value}°C".
	Colors           []string      // Optional: Colors for the scatter points.
	ShowLabels       bool          // Optional: Show labels on the scatter points.
	LabelPos         LabelPosition // Optional: Position of the labels, default is "right".
	SplitLine        bool          // Optional: Whether to show split lines on the X and Y axes.
	Symbol           []string      // Optional: Symbol of the scatter points. Default is "circle". If there are multiple series, you can specify different symbols for each series. If the length of the array is less than the number of series, the remaining series will repeat the order of the symbols.
	SymbolSize       int           // Optional: Size of the scatter points. Default is 10.
}

// CreateScatterChart draws one set of points per series, in argument order.
//
// It returns a nil chart and an error when no series is given, or when a
// series has a nil X or Y, X and Y of different lengths, no points, or a cell
// that is not a Go number (nil, text, a numeric string such as "2", or a
// bool), a NaN or an infinity. The error names the series by its position,
// counting from one, and its name.
func CreateScatterChart(config ScatterChartConfig, series ...ScatterSeries) (*charts.Scatter, error) {
	if len(series) == 0 {
		return nil, chartError("CreateScatterChart", "no data to draw")
	}
	xs := make([][]float64, len(series))
	ys := make([][]float64, len(series))
	for i, s := range series {
		var err error
		if xs[i], ys[i], err = readScatterSeries(i, s); err != nil {
			return nil, chartError("CreateScatterChart", "%v", err)
		}
	}
	scatter := charts.NewScatter()

	internal.SetBaseChartGlobalOptions(scatter, internal.BaseChartConfig{
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

	if config.Symbol == nil {
		config.Symbol = []string{"circle"}
	}
	if config.SymbolSize == 0 {
		config.SymbolSize = 10
	}

	// Build X and Y axis options with optional min/max/split/formatter and splitline
	xAxis := opts.XAxis{
		Name: config.XAxisName,
		SplitLine: &opts.SplitLine{
			Show: opts.Bool(config.SplitLine),
		},
	}
	if config.XAxisMin != nil {
		xAxis.Min = *config.XAxisMin
	}
	if config.XAxisMax != nil {
		xAxis.Max = *config.XAxisMax
	}
	if config.XAxisSplitNumber != nil {
		xAxis.SplitNumber = *config.XAxisSplitNumber
	}
	if config.XAxisFormatter != "" {
		xAxis.AxisLabel = &opts.AxisLabel{Formatter: types.FuncStr(config.XAxisFormatter)}
	}

	yAxis := opts.YAxis{
		Name: config.YAxisName,
		SplitLine: &opts.SplitLine{
			Show: opts.Bool(config.SplitLine),
		},
	}
	if config.YAxisMin != nil {
		yAxis.Min = *config.YAxisMin
	}
	if config.YAxisMax != nil {
		yAxis.Max = *config.YAxisMax
	}
	if config.YAxisSplitNumber != nil {
		yAxis.SplitNumber = *config.YAxisSplitNumber
	}
	if config.YAxisFormatter != "" {
		yAxis.AxisLabel = &opts.AxisLabel{Formatter: types.FuncStr(config.YAxisFormatter)}
	}

	// Apply axis options
	scatter.SetGlobalOptions(
		charts.WithXAxisOpts(xAxis),
		charts.WithYAxisOpts(yAxis),
	)

	// 設置系列顏色（如果未提供則使用默認配色）
	if len(config.Colors) == 0 {
		config.Colors = internal.GetColors(len(series))
	}
	scatter.SetGlobalOptions(
		charts.WithColorsOpts(opts.Colors(config.Colors)),
	)

	for i, sr := range series {
		scatter.AddSeries(sr.Name, convertToScatterData(xs[i], ys[i], config, i%len(config.Symbol)))
	}

	// 設置標籤
	internal.SetShowLabels(scatter, config.ShowLabels, string(config.LabelPos), string(LabelPositionRight))

	return scatter, nil
}

// readScatterSeries reads one series' X and Y, or says why it cannot be drawn.
// i is the series' place among the arguments, counting from zero.
func readScatterSeries(i int, s ScatterSeries) ([]float64, []float64, error) {
	if isNilList(s.X) || isNilList(s.Y) {
		return nil, nil, fmt.Errorf("series %d (%q) needs both X and Y", i+1, s.Name)
	}
	if s.X.Len() != s.Y.Len() {
		return nil, nil, fmt.Errorf("series %d (%q) has %d X values and %d Y values", i+1, s.Name, s.X.Len(), s.Y.Len())
	}
	if s.X.Len() == 0 {
		return nil, nil, fmt.Errorf("series %d (%q) has no points", i+1, s.Name)
	}
	x, err := readScatterAxis(i, s.Name, "X", s.X)
	if err != nil {
		return nil, nil, err
	}
	y, err := readScatterAxis(i, s.Name, "Y", s.Y)
	if err != nil {
		return nil, nil, err
	}
	return x, y, nil
}

// readScatterAxis reads every cell of dl as a number. A scatter point has no
// category axis to fall back on, so a cell that is not a Go number would be a
// point drawn in the wrong place; it is refused instead.
func readScatterAxis(i int, name, axis string, dl insyra.IDataList) ([]float64, error) {
	cells := dl.Data()
	out := make([]float64, len(cells))
	for row, cell := range cells {
		f, ok := insyra.ToFloat64Safe(cell)
		if !ok {
			return nil, fmt.Errorf("series %d (%q): %s at row index %d is %v (%T), not a number", i+1, name, axis, row, cell, cell)
		}
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, fmt.Errorf("series %d (%q): %s at row index %d is %v; a chart cannot draw NaN or an infinity", i+1, name, axis, row, f)
		}
		out[row] = f
	}
	return out, nil
}

// convertToScatterData pairs xs with ys as go-echarts points.
func convertToScatterData(xs, ys []float64, config ScatterChartConfig, symbolIndex int) []opts.ScatterData {
	scatterData := make([]opts.ScatterData, len(xs))
	for i := range xs {
		scatterData[i] = opts.ScatterData{
			Value:      [2]float64{xs[i], ys[i]},
			Symbol:     config.Symbol[symbolIndex],
			SymbolSize: config.SymbolSize,
		}
	}
	return scatterData
}
