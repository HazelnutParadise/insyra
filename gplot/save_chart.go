// gplot/save_chart.go

package gplot

import (
	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
)

// SaveChart saves the plot to a file.
// Supported file formats: .jpg|.jpeg|.pdf|.png|.svg|.tex|.tif|.tiff
// Automatically determine the file extension based on the filename.
func SaveChart(plt *plot.Plot, filename string) {
	// Save the plot to a PNG file.
	if err := plt.Save(8*vg.Inch, 4*vg.Inch, filename); err != nil {
		insyra.LogFatal("gplot.SaveChart: failed to save chart: %w", err)
	}
}
