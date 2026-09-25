package plot

import "testing"

// The point type was unexported, so a caller building points in a loop could
// not name the slice to append to. It is HeatMapPoint now, built with
// NewHeatMapPoint and NewHeatMapMissingPoint.
func TestHeatMapPointsCanBeCollected(t *testing.T) {
	var points []HeatMapPoint[int, int]
	for x := 0; x < 2; x++ {
		for y := 0; y < 2; y++ {
			points = append(points, NewHeatMapPoint(x, y, float64(x+y)))
		}
	}
	points = append(points, NewHeatMapMissingPoint(2, 2))
	if len(points) != 5 || !points[0].Valid || points[4].Valid {
		t.Fatalf("points = %+v", points)
	}
	if chart := CreateHeatMap(HeatMapConfig{Title: "loop"}, points...); chart == nil {
		t.Fatal("CreateHeatMap returned nil for collected points")
	}
}

// HeatMapAxis is the exported constraint, so generic code over heat map axes
// can be written outside the package.
func collectRow[X HeatMapAxis](xs []X, y string) []HeatMapPoint[X, string] {
	row := make([]HeatMapPoint[X, string], 0, len(xs))
	for _, x := range xs {
		row = append(row, NewHeatMapPoint(x, y, 1))
	}
	return row
}

func TestHeatMapAxisIsUsable(t *testing.T) {
	if got := collectRow([]string{"mon", "tue"}, "am"); len(got) != 2 {
		t.Fatalf("row = %+v", got)
	}
}
