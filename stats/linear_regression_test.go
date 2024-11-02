package stats

import (
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestLinearRegression(t *testing.T) {
	dlX := insyra.NewDataList(1, 4, 9, 110, 8)
	dlY := insyra.NewDataList(3, 6, 8, 6, 2)
	result := LinearRegression(dlX, dlY)

	var expectedValues = LinearRegressionResult{
		Slope:            0.013102128241352597,
		Intercept:        4.654103814428291,
		RSquared:         0.06278103115648115,
		PValue:           0.6843455560821862,
		StandardError:    0.029227221522309617,
		TValue:           0.4482851108974395,
		AdjustedRSquared: -0.2496252917913584,
		Residuals:        []float64{-1.6672059426696437, 1.2934876726062985, 3.2279770313995355, -0.0953379209770766, -2.758920840359112},
	}

	if result == nil {
		t.Error("LinearRegression returned nil")
	} else if !reflect.DeepEqual(result, &expectedValues) {
		t.Errorf("LinearRegression result mismatch: expected %v, got %v", expectedValues, result)
	}
}
