// gplot/heatmap.go

package gplot

import (
	"math"

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

// CreateHeatmapChart draws data as a grid of coloured cells: row i of the
// table is row i of the grid and column j is column j. A [][]float64 grid is
// passed as insyra.ReadSlice2D(grid).
//
// Each column is read through DataList.ToF64Slice: a number, a fixed-point
// decimal included, is drawn as its value, and any other cell, whether nil,
// text (a numeric string such as "2" included) or a bool, is drawn as 0. A
// column shorter than the others is padded with nil by the table, so its
// missing cells are drawn as 0 too.
//
// It returns a nil chart and an error when data is nil, has no rows or no
// columns, holds an infinity, or holds nothing but NaN. A NaN among numbers is
// drawn as an empty cell.
func CreateHeatmapChart(config HeatmapChartConfig, data insyra.IDataTable) (*plot.Plot, error) {
	if isNilTable(data) {
		return nil, chartError("CreateHeatmapChart", "no data to draw")
	}
	dataSlice := tableToGrid(data)
	if len(dataSlice) == 0 {
		return nil, chartError("CreateHeatmapChart", "the table is empty")
	}
	// gonum builds the colour scale from the grid's range when the chart is
	// saved: an infinity there panicked on amd64, and a grid of NaN alone
	// panicked on every platform.
	anyNumber := false
	for i, row := range dataSlice {
		for j, v := range row {
			if math.IsInf(v, 0) {
				return nil, chartError("CreateHeatmapChart", "the cell at row %d, column %d is %v, which no colour can show", i, j, v)
			}
			if !math.IsNaN(v) {
				anyNumber = true
			}
		}
	}
	if !anyNumber {
		return nil, chartError("CreateHeatmapChart", "every cell is NaN; there is nothing to colour")
	}

	// Create a new plot.
	plt := plot.New()

	// Set chart title and axis labels.
	plt.Title.Text = config.Title
	plt.X.Label.Text = config.XAxisName
	plt.Y.Label.Text = config.YAxisName

	// Create grid data
	grid := &gridData{
		data:  dataSlice,
		xAxis: config.XAxis,
		yAxis: config.YAxis,
	}

	// Set default values
	colors := config.Colors
	// palette.Heat makes a slice of this length, so a negative count panicked
	// with "makeslice: len out of range". Anything non-positive takes the
	// default, the way 0 already did.
	if colors <= 0 {
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

	return plt, nil
}

// tableToGrid reads dt into rows of float64, each column through ToF64Slice,
// so the heat map reads a cell the way every other gplot chart does. It
// returns nil for a table with no rows or no columns.
func tableToGrid(dt insyra.IDataTable) [][]float64 {
	var grid [][]float64
	dt.AtomicDo(func(t *insyra.DataTable) {
		rows, cols := t.Size()
		if rows == 0 || cols == 0 {
			return
		}
		grid = make([][]float64, rows)
		for i := range grid {
			grid[i] = make([]float64, cols)
		}
		for j := 0; j < cols; j++ {
			col := t.GetColByNumber(j).ToF64Slice()
			for i := 0; i < rows && i < len(col); i++ {
				grid[i][j] = col[i]
			}
		}
	})
	return grid
}
