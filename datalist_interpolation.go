package insyra

import (
	"errors"
	"math"

	"github.com/HazelnutParadise/insyra/internal/algorithms"
)

// LinearInterpolation performs linear interpolation for the given x value using the DataList.
func (dl *DataList) LinearInterpolation(x float64) float64 {
	var data []float64
	dl.AtomicDo(func(l *DataList) {
		if len(l.data) < 2 {
			return
		}
		data = make([]float64, len(l.data))
		for i, v := range l.data {
			data[i] = v.(float64)
		}
	})
	if len(data) < 2 {
		dl.warn("LinearInterpolation", "Not enough data points")
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
	var data []float64
	dl.AtomicDo(func(l *DataList) {
		if len(l.data) < 3 {
			return
		}
		data = make([]float64, len(l.data))
		for i, v := range l.data {
			data[i] = v.(float64)
		}
	})
	if len(data) < 3 {
		dl.warn("QuadraticInterpolation", "Not enough data points")
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
	var floatData []float64
	dl.AtomicDo(func(l *DataList) {
		floatData = l.ToF64Slice()
	})
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
	var floatData []float64
	dl.AtomicDo(func(l *DataList) {
		floatData = l.ToF64Slice()
	})
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
	var floatData []float64
	dl.AtomicDo(func(l *DataList) {
		floatData = l.ToF64Slice()
	})
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
	var floatData []float64
	dl.AtomicDo(func(l *DataList) {
		floatData = l.ToF64Slice()
	})
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
