package gplot

import (
	"math"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
)

// FunctionPlotConfig defines the configuration for a function plot.
type FunctionPlotConfig struct {
	Title     string  // Title of the chart.
	XAxisName string  // Label for the X-axis.
	YAxisName string  // Label for the Y-axis.
	XMin      float64 // Minimum value of X (optional).
	XMax      float64 // Maximum value of X (optional).
	YMin      float64 // Minimum value of Y (optional).
	YMax      float64 // Maximum value of Y (optional).
}

// CreateFunctionPlot generates and returns a plot.Plot object based on FunctionPlotConfig.
func CreateFunctionPlot(config FunctionPlotConfig, function func(x float64) float64) *plot.Plot {
	// 如果沒有設置 X 軸範圍，則默認使用 [-10, 10]
	if config.XMin == 0 && config.XMax == 0 {
		config.XMin = -10
		config.XMax = 10
	}

	if config.XAxisName == "" {
		config.XAxisName = "X"
	}
	if config.YAxisName == "" {
		config.YAxisName = "Y"
	}

	// Create a new plot.
	plt := plot.New()

	// Set chart title and axis labels.
	plt.Title.Text = config.Title
	plt.X.Label.Text = config.XAxisName
	plt.Y.Label.Text = config.YAxisName

	// Create the function plot using plotter.NewFunction
	functionPlot := plotter.NewFunction(function)

	// Set X-axis range for the function
	functionPlot.XMin = config.XMin
	functionPlot.XMax = config.XMax

	// Automatically calculate the number of samples based on X-axis range
	functionPlot.Samples = calculateSamples(config.XMin, config.XMax)

	// 動態設置 Y 軸範圍
	if config.YMin == 0 && config.YMax == 0 {
		yMin, yMax := calculateYRange(function, config.XMin, config.XMax, functionPlot.Samples)
		config.YMin = yMin
		config.YMax = yMax
	}
	plt.Y.Min = config.YMin
	plt.Y.Max = config.YMax

	// Add function plot to the plot.
	plt.Add(functionPlot)

	// 手動設置 X 軸範圍
	plt.X.Min = config.XMin
	plt.X.Max = config.XMax

	return plt
}

// calculateSamples dynamically calculates an appropriate number of samples based on the X-axis range.
func calculateSamples(xMin, xMax float64) int {
	// 根據 X 軸範圍自動計算樣本數量
	rangeX := math.Abs(xMax - xMin)
	// 每單位範圍 100 個樣本點，可根據需要調整
	return int(rangeX * 100)
}

// calculateYRange calculates the minimum and maximum Y values for the function over the specified X range.
func calculateYRange(f func(x float64) float64, xMin, xMax float64, samples int) (float64, float64) {
	yMin := math.Inf(1)
	yMax := math.Inf(-1)

	step := (xMax - xMin) / float64(samples-1)

	for i := 0; i < samples; i++ {
		x := xMin + float64(i)*step
		y := f(x)

		// 更新 Y 軸範圍
		if y < yMin {
			yMin = y
		}
		if y > yMax {
			yMax = y
		}
	}

	// 增加一些緩衝區以確保圖形不會剛好切齊 Y 軸範圍
	padding := (yMax - yMin) * 0.1
	if padding == 0 {
		padding = 1 // 如果範圍太小，給定一個固定的最小範圍
	}
	yMin -= padding
	yMax += padding

	return yMin, yMax
}
