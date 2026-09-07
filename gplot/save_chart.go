// gplot/save_chart.go

package gplot

import (
	"fmt"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
)

// SaveChart writes the plot to a file and reports whether it worked.
// Supported file formats: .jpg|.jpeg|.pdf|.png|.svg|.tex|.tif|.tiff, chosen
// from the filename's extension.
//
// A write failure (missing directory, no permission, disk full) is returned;
// it no longer ends the program.
func SaveChart(plt *plot.Plot, filename string) error {
	if plt == nil {
		err := fmt.Errorf("gplot: SaveChart: no plot to save (chart creation failed?)")
		insyra.LogError("gplot", "SaveChart", "%v", err)
		return err
	}
	if err := plt.Save(8*vg.Inch, 4*vg.Inch, filename); err != nil {
		wrapped := fmt.Errorf("gplot: failed to save chart to %s: %w", filename, err)
		insyra.LogError("gplot", "SaveChart", "%v", wrapped)
		return wrapped
	}
	insyra.LogInfo("gplot", "SaveChart", "saved chart to %s", filename)
	return nil
}
