// gplot/heatmap.go

package gplot

import (
	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette"
	"gonum.org/v1/plot/plotter"
)

// HeatmapChartConfig defines the configuration for a heatmap chart.
type HeatmapChartConfig struct {
	Title     string    // Title of the chart.
	XAxisName string    // Optional: X-axis name.
	YAxisName string    // Optional: Y-axis name.
	XAxis     []float64 // Optional: X-axis coordinates. If not provided, will use indices.
	YAxis     []float64 // Optional: Y-axis coordinates. If not provided, will use indices.
	Colors    int       // Optional: Number of colors in the palette. Default is 20.
	Alpha     float64   // Optional: Alpha (transparency) for colors. Default is 1.0.
}

// gridData implements the plotter.GridXYZ interface for heatmap data.
type gridData struct {
	data  [][]float64
	xAxis []float64
	yAxis []float64
}

// Dims returns the dimensions of the grid (columns, rows).
func (g *gridData) Dims() (c, r int) {
	if len(g.data) == 0 {
		return 0, 0
	}
	return len(g.data[0]), len(g.data)
}

// Z returns the value at grid position (c, r).
func (g *gridData) Z(c, r int) float64 {
	return g.data[r][c]
}

// X returns the X coordinate for column c.
func (g *gridData) X(c int) float64 {
	if g.xAxis != nil && c < len(g.xAxis) {
		return g.xAxis[c]
	}
	return float64(c)
}

// Y returns the Y coordinate for row r.
func (g *gridData) Y(r int) float64 {
	if g.yAxis != nil && r < len(g.yAxis) {
		return g.yAxis[r]
	}
	return float64(r)
}

// CreateHeatmapChart generates and returns a plot.Plot object based on HeatmapChartConfig.
// The Data field can be of type [][]float64, *insyra.DataTable, or insyra.IDataTable.
func CreateHeatmapChart(config HeatmapChartConfig, data any) *plot.Plot {
	// Create a new plot.
	plt := plot.New()

	// Set chart title and axis labels.
	plt.Title.Text = config.Title
	plt.X.Label.Text = config.XAxisName
	plt.Y.Label.Text = config.YAxisName

	var dataSlice [][]float64

	// Determine the type of Data and handle it accordingly
	switch d := data.(type) {
	case [][]float64:
		dataSlice = d
	case *insyra.DataTable:
		// Convert DataTable to [][]float64
		dataSlice = convertDataTableToGrid(d)
	case insyra.IDataTable:
		// Convert IDataTable to [][]float64
		dataSlice = convertIDataTableToGrid(d)
	default:
		insyra.LogWarning("gplot", "CreateHeatmapChart", "Unsupported Data type: %T\n", data)
		return nil
	}

	// Validate data
	if len(dataSlice) == 0 || len(dataSlice[0]) == 0 {
		insyra.LogWarning("gplot", "CreateHeatmapChart", "Empty data provided")
		return nil
	}

	// Create grid data
	grid := &gridData{
		data:  dataSlice,
		xAxis: config.XAxis,
		yAxis: config.YAxis,
	}

	// Set default values
	colors := config.Colors
	if colors == 0 {
		colors = 20
	}
	alpha := config.Alpha
	if alpha == 0.0 {
		alpha = 1.0
	}

	// Create color palette
	pal := palette.Heat(colors, alpha)

	// Create heatmap
	hm := plotter.NewHeatMap(grid, pal)

	// Add heatmap to plot
	plt.Add(hm)

	return plt
}

// convertDataTableToGrid converts a DataTable to [][]float64 for heatmap use.
func convertDataTableToGrid(dt *insyra.DataTable) [][]float64 {
	var data [][]float64
	isFailed := false
	dt.AtomicDo(func(dt *insyra.DataTable) {
		rows, cols := dt.Size()

		if rows == 0 || cols == 0 {
			isFailed = true
			return
		}

		data = make([][]float64, rows)
		for i := range rows {
			data[i] = make([]float64, cols)
			for j := 0; j < cols; j++ {
				val := dt.GetElementByNumberIndex(i, j)
				if val != nil {
					switch v := val.(type) {
					case float64:
						data[i][j] = v
					case float32:
						data[i][j] = float64(v)
					case int:
						data[i][j] = float64(v)
					case int32:
						data[i][j] = float64(v)
					case int64:
						data[i][j] = float64(v)
					default:
						// For non-numeric values, use 0
						data[i][j] = 0.0
					}
				} else {
					data[i][j] = 0.0
				}
			}
		}
	})
	if isFailed {
		return nil
	}
	return data
}

// convertIDataTableToGrid converts an IDataTable to [][]float64 for heatmap use.
func convertIDataTableToGrid(dt insyra.IDataTable) [][]float64 {
	var data [][]float64
	isFailed := false
	dt.AtomicDo(func(dt *insyra.DataTable) {
		rows, cols := dt.Size()

		if rows == 0 || cols == 0 {
			isFailed = true
			return
		}

		data = make([][]float64, rows)
		for i := 0; i < rows; i++ {
			data[i] = make([]float64, cols)
			for j := 0; j < cols; j++ {
				val := dt.GetElementByNumberIndex(i, j)
				if val != nil {
					switch v := val.(type) {
					case float64:
						data[i][j] = v
					case float32:
						data[i][j] = float64(v)
					case int:
						data[i][j] = float64(v)
					case int32:
						data[i][j] = float64(v)
					case int64:
						data[i][j] = float64(v)
					default:
						// For non-numeric values, use 0
						data[i][j] = 0.0
					}
				} else {
					data[i][j] = 0.0
				}
			}
		}
	})
	if isFailed {
		return nil
	}

	return data
}
