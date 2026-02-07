// gplot/step.go

package gplot

import (
	"image/color"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// StepChartConfig defines the configuration for a multi-series step chart.
type StepChartConfig struct {
	Title     string    // Title of the chart.
	XAxis     []float64 // X-axis data.
	XAxisName string    // Optional: X-axis name.
	YAxisName string    // Optional: Y-axis name.
	StepStyle string    // Optional: Step style - "pre", "mid", "post". Default is "post".
}

// CreateStepChart generates and returns a plot.Plot object based on StepChartConfig.
// The Data field can be of type map[string][]float64, []*insyra.DataList, or []insyra.IDataList.
func CreateStepChart(config StepChartConfig, data any) *plot.Plot {
	// Create a new plot.
	plt := plot.New()

	// Set chart title and axis labels.
	plt.Title.Text = config.Title
	plt.X.Label.Text = config.XAxisName
	plt.Y.Label.Text = config.YAxisName

	// Determine step style
	var stepKind plotter.StepKind
	switch config.StepStyle {
	case "pre":
		stepKind = plotter.PreStep
	case "mid":
		stepKind = plotter.MidStep
	case "post", "":
		stepKind = plotter.PostStep
	default:
		insyra.LogWarning("gplot", "CreateStepChart", "Unknown StepStyle: %s, using PostStep", config.StepStyle)
		stepKind = plotter.PostStep
	}

	// Handle different types of Data
	switch data := data.(type) {
	case map[string][]float64:
		if config.XAxis == nil {
			config.XAxis = autoGenerateXAxis(data)
		}
		// If Data is map[string][]float64
		i := 0
		for seriesName, values := range data {
			addStepSeries(plt, seriesName, values, config.XAxis, nil, i, stepKind)
			i++
		}
	case []*insyra.DataList:
		if config.XAxis == nil {
			config.XAxis = autoGenerateXAxisForDataList(data)
		}
		for i, dataList := range data {
			addStepSeries(plt, dataList.GetName(), dataList.ToF64Slice(), config.XAxis, nil, i, stepKind)
		}

	case []insyra.IDataList:
		if config.XAxis == nil {
			config.XAxis = autoGenerateXAxisForIDataList(data)
		}
		for i, dataList := range data {
			addStepSeries(plt, dataList.GetName(), dataList.ToF64Slice(), config.XAxis, nil, i, stepKind)
		}
	default:
		insyra.LogWarning("gplot", "CreateStepChart", "Unsupported Data type: %T\n", data)
		return nil
	}

	return plt
}

// addStepSeries is a helper function to add a step series to the plot.
func addStepSeries(plt *plot.Plot, seriesName string, values []float64, xAxis []float64, colors []color.Color, index int, stepKind plotter.StepKind) {
	// Check if X-axis and Data lengths match
	if len(xAxis) != len(values) {
		insyra.LogWarning("gplot", "addStepSeries", "Length of XAxis and Data for series %s do not match", seriesName)
		return
	}

	// Prepare the points for the step plot
	stepData := make(plotter.XYs, len(xAxis))
	for j := range xAxis {
		stepData[j].X = xAxis[j]
		stepData[j].Y = values[j]
	}

	// Create the line plot with step style
	line, err := plotter.NewLine(stepData)
	if err != nil {
		panic(err)
	}

	// Set the step style
	line.StepStyle = stepKind

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
		line.Dashes = []vg.Length{vg.Points(6), vg.Points(2), vg.Points(1), vg.Points(2)}
	}

	// Add the step plot to the chart
	plt.Add(line)
	plt.Legend.Add(seriesName, line)
}
