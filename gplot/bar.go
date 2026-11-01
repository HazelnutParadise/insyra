// gplot/bar.go

package gplot

import (
	"strconv"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// BarChartConfig defines the configuration for a bar chart.
type BarChartConfig struct {
	Title     string    // Title of the chart.
	XAxis     []string  // X-axis data (categories).
	XAxisName string    // Optional: X-axis name.
	YAxisName string    // Optional: Y-axis name.
	BarWidth  float64   // Optional: Width of each bar, in points. Default is 20.
	ErrorBars []float64 // Optional: Error bar values for each bar. Only with one list, and its length must match the list's. They stay black whatever Colors says.
	Colors    []string  // Optional: Series colours in place of the shared palette, as #RGB, #RRGGBB, #RRGGBBAA or a CSS name. A short list repeats.
}

// CreateBarChart draws one bar per value in each list. Several lists are
// drawn side by side in each category, one colour per list, with a legend
// naming each list. A plain slice is passed as insyra.NewDataList(values).
//
// The values are read through DataList.ToF64Slice: a number, a fixed-point
// decimal included, is drawn as its value, and any other cell, whether nil,
// text (a numeric string such as "2" included) or a bool, is drawn as 0.
//
// It returns a nil chart and an error when no list is left to draw (none was
// given, or every one was nil), when a list is empty, has a different length
// from the first, or holds a NaN or an infinity, and when config.ErrorBars is
// given with more than one list, its length differs from the data's, or it
// holds a NaN or an infinity. A list is named by its position among the
// arguments, counting from one. A nil list among real ones is skipped with a
// warning.
func CreateBarChart(config BarChartConfig, data ...insyra.IDataList) (*plot.Plot, error) {
	colors, err := readColors("CreateBarChart", config.Colors)
	if err != nil {
		return nil, err
	}
	data, pos, dropped := nonNilLists(data)
	if len(data) == 0 {
		return nil, chartError("CreateBarChart", "no data to draw")
	}
	if len(config.ErrorBars) > 0 && len(data) > 1 {
		return nil, chartError("CreateBarChart", "ErrorBars needs exactly one list, but %d lists were given", len(data))
	}
	values := make([][]float64, len(data))
	for i, dl := range data {
		values[i] = readValues(dl)
		if len(values[i]) == 0 {
			return nil, chartError("CreateBarChart", "list %d is empty", pos[i]+1)
		}
		if len(values[i]) != len(values[0]) {
			return nil, chartError("CreateBarChart", "list %d has %d values but list %d has %d", pos[i]+1, len(values[i]), pos[0]+1, len(values[0]))
		}
	}

	barWidth := config.BarWidth
	if barWidth == 0 {
		barWidth = 20 // Default bar width
	}

	groups := make([]*plotter.BarChart, len(values))
	for i, vs := range values {
		bars, err := plotter.NewBarChart(plotter.Values(vs), vg.Points(barWidth))
		if err != nil {
			return nil, chartError("CreateBarChart", "cannot draw list %d: %w", pos[i]+1, err)
		}
		bars.Color = colors.at(i)
		bars.LineStyle.Width = 0
		if len(values) > 1 {
			// Centre the group on its category: bar i sits i widths right of
			// the leftmost, and the group's middle sits on the tick.
			bars.Offset = vg.Points((float64(i) - float64(len(values)-1)/2) * barWidth)
		}
		groups[i] = bars
	}

	// Error bars that were asked for are drawn, or the call fails: a chart
	// without them would look complete while missing what was requested.
	var errBars *plotter.YErrorBars
	if len(config.ErrorBars) > 0 {
		if len(config.ErrorBars) != len(values[0]) {
			return nil, chartError("CreateBarChart", "ErrorBars has %d values but the data has %d", len(config.ErrorBars), len(values[0]))
		}
		var err error
		errBars, err = plotter.NewYErrorBars(&barErrorData{values: values[0], errorBars: config.ErrorBars})
		if err != nil {
			return nil, chartError("CreateBarChart", "cannot draw the error bars: %w", err)
		}
	}

	// Create a new plot.
	plt := plot.New()

	// Set chart title and axis labels.
	plt.Title.Text = config.Title
	plt.X.Label.Text = config.XAxisName
	plt.Y.Label.Text = config.YAxisName

	// Set axis labels (categories). gonum's NominalX indexes names[0] with no
	// length check, so an empty XAxis — which is what a zero-value config has —
	// used to panic here. Number the bars instead, the way plot.CreateBarChart
	// does, so a config without labels still gives a readable chart.
	labels := config.XAxis
	if len(labels) == 0 {
		labels = make([]string, len(values[0]))
		for i := range labels {
			labels[i] = strconv.Itoa(i + 1)
		}
	}
	plt.NominalX(labels...)

	for i, bars := range groups {
		plt.Add(bars)
		if len(groups) > 1 {
			name := data[i].GetName()
			if name == "" {
				name = "list " + strconv.Itoa(pos[i]+1)
			}
			plt.Legend.Add(name, bars)
		}
	}
	// Half a category of margin on each side keeps the outermost bars, and a
	// group's outer bars, inside the frame instead of cut by it.
	plt.X.Min = -0.5
	plt.X.Max = float64(len(values[0])) - 0.5
	if len(groups) > 1 {
		// The legend sits in the top corner; headroom keeps it off the bars.
		plt.Legend.Top = true
		plt.Y.Max += 0.2 * (plt.Y.Max - plt.Y.Min)
	}
	if errBars != nil {
		plt.Add(errBars)
	}

	warnNilLists("CreateBarChart", dropped)
	return plt, nil
}

// barErrorData implements the XYer and YErrorer interfaces for bar chart error bars
type barErrorData struct {
	values    []float64
	errorBars []float64
}

func (d *barErrorData) Len() int {
	return len(d.values)
}

func (d *barErrorData) XY(i int) (float64, float64) {
	return float64(i), d.values[i]
}

func (d *barErrorData) YError(i int) (float64, float64) {
	// Return symmetric error bars (low and high are the same)
	return d.errorBars[i], d.errorBars[i]
}
