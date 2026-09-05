package insyra

import (
	"errors"
	"math"

	"github.com/HazelnutParadise/insyra/internal/algorithms"
)

// LinearInterpolation performs linear interpolation for the given x value using the DataList.
func (dl *DataList) LinearInterpolation(x float64) float64 {
	data, ok := dl.interpolationInput("LinearInterpolation", 2)
	if !ok {
		return math.NaN()
	}

	result, err := algorithms.LinearInterpolation(data, x)
	if err != nil {
		if errors.Is(err, algorithms.ErrOutOfBounds) {
			dl.warn("LinearInterpolation", "X value out of bounds")
		} else {
			dl.warn("LinearInterpolation", "Interpolation failed: %v", err)
		}
		return math.NaN()
	}
	return result
}

// QuadraticInterpolation performs quadratic interpolation for the given x value using the DataList.
func (dl *DataList) QuadraticInterpolation(x float64) float64 {
	data, ok := dl.interpolationInput("QuadraticInterpolation", 3)
	if !ok {
		return math.NaN()
	}

	result, err := algorithms.QuadraticInterpolation(data, x)
	if err != nil {
		if errors.Is(err, algorithms.ErrOutOfBounds) {
			dl.warn("QuadraticInterpolation", "X value out of bounds")
		} else {
			dl.warn("QuadraticInterpolation", "Interpolation failed: %v", err)
		}
		return math.NaN()
	}
	return result
}

// LagrangeInterpolation performs Lagrange interpolation for the given x value using the DataList.
func (dl *DataList) LagrangeInterpolation(x float64) float64 {
	floatData, ok := dl.interpolationInput("LagrangeInterpolation", 0)
	if !ok {
		return math.NaN()
	}
	result, err := algorithms.LagrangeInterpolation(floatData, x)
	if err != nil {
		if errors.Is(err, algorithms.ErrNotEnoughData) {
			dl.warn("LagrangeInterpolation", "Not enough data points")
		} else {
			dl.warn("LagrangeInterpolation", "Interpolation failed: %v", err)
		}
		return math.NaN()
	}
	return result
}

// NearestNeighborInterpolation performs nearest-neighbor interpolation for the given x value using the DataList.
func (dl *DataList) NearestNeighborInterpolation(x float64) float64 {
	floatData, ok := dl.interpolationInput("NearestNeighborInterpolation", 0)
	if !ok {
		return math.NaN()
	}
	result, err := algorithms.NearestNeighborInterpolation(floatData, x)
	if err != nil {
		if errors.Is(err, algorithms.ErrOutOfBounds) {
			dl.warn("NearestNeighborInterpolation", "X value out of bounds")
		} else {
			dl.warn("NearestNeighborInterpolation", "Interpolation failed: %v", err)
		}
		return math.NaN()
	}
	return result
}

// NewtonInterpolation performs Newton's interpolation for the given x value using the DataList.
func (dl *DataList) NewtonInterpolation(x float64) float64 {
	floatData, ok := dl.interpolationInput("NewtonInterpolation", 0)
	if !ok {
		return math.NaN()
	}
	result, err := algorithms.NewtonInterpolation(floatData, x)
	if err != nil {
		if errors.Is(err, algorithms.ErrNotEnoughData) {
			dl.warn("NewtonInterpolation", "Not enough data points")
		} else {
			dl.warn("NewtonInterpolation", "Interpolation failed: %v", err)
		}
		return math.NaN()
	}
	return result
}

// HermiteInterpolation performs Hermite interpolation for the given x value using the DataList.
func (dl *DataList) HermiteInterpolation(x float64, derivatives []float64) float64 {
	floatData, ok := dl.interpolationInput("HermiteInterpolation", 0)
	if !ok {
		return math.NaN()
	}
	result, err := algorithms.HermiteInterpolation(floatData, derivatives, x)
	if err != nil {
		if errors.Is(err, algorithms.ErrLengthMismatch) {
			dl.warn("HermiteInterpolation", "Data and derivatives length mismatch")
		} else if errors.Is(err, algorithms.ErrNotEnoughData) {
			dl.warn("HermiteInterpolation", "Not enough data points")
		} else {
			dl.warn("HermiteInterpolation", "Interpolation failed: %v", err)
		}
		return math.NaN()
	}
	return result
}

// interpolationInput reads the list as a fully numeric grid. A nil, NaN, or
// non-numeric cell is refused: an interpolation grid cannot have a hole, and
// substituting 0 would silently bend the curve. minPoints > 0 additionally
// requires at least that many points.
func (dl *DataList) interpolationInput(funcName string, minPoints int) ([]float64, bool) {
	var data []float64
	var badRow int
	var ok bool
	var n int
	dl.AtomicDo(func(l *DataList) {
		n = len(l.data)
		data, badRow, ok = numericCells(l.data, false)
	})
	if !ok {
		dl.warn(funcName, "non-numeric or missing value at row %d", badRow)
		return nil, false
	}
	if minPoints > 0 && n < minPoints {
		dl.warn(funcName, "Not enough data points")
		return nil, false
	}
	return data, true
}
