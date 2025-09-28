// gplot/line.go

package gplot

import (
	"image/color"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// LineChartConfig defines the configuration for a multi-series line chart.
type LineChartConfig struct {
	Title     string    // Title of the chart.
	XAxis     []float64 // X-axis data.
	Data      any       // Supports map[string][]float64 or []*insyra.DataList.
	XAxisName string    // Optional: X-axis name.
	YAxisName string    // Optional: Y-axis name.
}

// CreateLineChart generates and returns a plot.Plot object based on LineChartConfig.
func CreateLineChart(config LineChartConfig) *plot.Plot {
	// Create a new plot.
	plt := plot.New()

	// Set chart title and axis labels.
	plt.Title.Text = config.Title
	plt.X.Label.Text = config.XAxisName
	plt.Y.Label.Text = config.YAxisName

	// Handle different types of Data
	switch data := config.Data.(type) {
	case map[string][]float64:
		if config.XAxis == nil {
			config.XAxis = autoGenerateXAxis(data)
		}
		// If Data is map[string][]float64
		i := 0
		for seriesName, values := range data {
			addLineSeries(plt, seriesName, values, config.XAxis, nil, i)
			i++
		}
	case []*insyra.DataList:
		if config.XAxis == nil {
			config.XAxis = autoGenerateXAxisForDataList(data)
		}
		for i, dataList := range data {
			addLineSeries(plt, dataList.GetName(), dataList.ToF64Slice(), config.XAxis, nil, i)
		}

	case []insyra.IDataList:
		if config.XAxis == nil {
			config.XAxis = autoGenerateXAxisForIDataList(data)
		}
		for i, dataList := range data {
			addLineSeries(plt, dataList.GetName(), dataList.ToF64Slice(), config.XAxis, nil, i)
		}
	default:
		insyra.LogWarning("gplot", "CreateLineChart", "Unsupported Data type: %T\n", config.Data)
		return nil
	}

	return plt
}

// Helper function to auto-generate X-axis for map[string][]float64
func autoGenerateXAxis(data map[string][]float64) []float64 {
	var maxLen int
	for _, values := range data {
		if len(values) > maxLen {
			maxLen = len(values)
		}
	}
	xAxis := make([]float64, maxLen)
	for i := range xAxis {
		xAxis[i] = float64(i)
	}
	return xAxis
}

// Helper function to auto-generate X-axis for []*insyra.DataList
func autoGenerateXAxisForDataList(data []*insyra.DataList) []float64 {
	if len(data) == 0 {
		return nil
	}
	maxLen := len(data[0].ToF64Slice())
	xAxis := make([]float64, maxLen)
	for i := range xAxis {
		xAxis[i] = float64(i)
	}
	return xAxis
}

// Helper function to auto-generate X-axis for []insyra.IDataList
func autoGenerateXAxisForIDataList(data []insyra.IDataList) []float64 {
	if len(data) == 0 {
		return nil
	}
	maxLen := len(data[0].ToF64Slice())
	xAxis := make([]float64, maxLen)
	for i := range xAxis {
		xAxis[i] = float64(i)
	}
	return xAxis
}

// addLineSeries is a helper function to add a line series to the plot.
func addLineSeries(plt *plot.Plot, seriesName string, values []float64, xAxis []float64, colors []color.Color, index int) {
	// Check if X-axis and Data lengths match
	if len(xAxis) != len(values) {
		insyra.LogWarning("gplot", "addLineSeries", "Length of XAxis and Data for series %s do not match", seriesName)
		return
	}

	// Prepare the points for the line plot
	lineData := make(plotter.XYs, len(xAxis))
	for j := range xAxis {
		lineData[j].X = xAxis[j]
		lineData[j].Y = values[j]
	}

	// Create the line plot
	line, err := plotter.NewLine(lineData)
	if err != nil {
		panic(err)
	}

	// Apply color if provided
	if len(colors) > index {
		line.Color = colors[index]
	}

	// Set different line styles for each series
	switch index % 5 {
	case 0:
		// 實線
		line.LineStyle = plotter.DefaultLineStyle
	case 1:
		// 長虛線
		line.LineStyle = plotter.DefaultLineStyle
		line.Dashes = []vg.Length{vg.Points(8), vg.Points(4)}
	case 2:
		// 點線
		line.LineStyle = plotter.DefaultLineStyle
		line.Dashes = []vg.Length{vg.Points(2), vg.Points(2)}
	case 3:
		// 短虛線
		line.LineStyle = plotter.DefaultLineStyle
		line.Dashes = []vg.Length{vg.Points(4), vg.Points(2)}
	case 4:
		// 交替虛線和實線
		line.LineStyle = plotter.DefaultLineStyle
		line.LineStyle.Dashes = []vg.Length{vg.Points(6), vg.Points(2), vg.Points(1), vg.Points(2)}
	}

	// Add the line plot to the chart
	plt.Add(line)
	plt.Legend.Add(seriesName, line)
}
