package plot

import "testing"

// The point type was unexported, so a caller building points in a loop could
// not name the slice to append to. It is HeatmapPoint now, built with
// NewHeatmapPoint and NewHeatmapMissingPoint.
func TestHeatMapPointsCanBeCollected(t *testing.T) {
	var points []HeatmapPoint[int, int]
	for x := 0; x < 2; x++ {
		for y := 0; y < 2; y++ {
			points = append(points, NewHeatmapPoint(x, y, float64(x+y)))
		}
	}
	points = append(points, NewHeatmapMissingPoint(2, 2))
	if len(points) != 5 || !points[0].Valid || points[4].Valid {
		t.Fatalf("points = %+v", points)
	}
	if chart, err := CreateHeatmapChart(HeatmapChartConfig{Title: "loop"}, points...); err != nil || chart == nil {
		t.Fatalf("CreateHeatmapChart(collected points) = %v, %v", chart, err)
	}
}

// HeatmapAxis is the exported constraint, so generic code over heat map axes
// can be written outside the package.
func collectRow[X HeatmapAxis](xs []X, y string) []HeatmapPoint[X, string] {
	row := make([]HeatmapPoint[X, string], 0, len(xs))
	for _, x := range xs {
		row = append(row, NewHeatmapPoint(x, y, 1))
	}
	return row
}

func TestHeatMapAxisIsUsable(t *testing.T) {
	if got := collectRow([]string{"mon", "tue"}, "am"); len(got) != 2 {
		t.Fatalf("row = %+v", got)
	}
}
