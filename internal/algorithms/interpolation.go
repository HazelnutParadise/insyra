package algorithms

import (
	"errors"
	"math"
)

var (
	ErrNotEnoughData  = errors.New("not enough data points")
	ErrOutOfBounds    = errors.New("x value out of bounds")
	ErrLengthMismatch = errors.New("data and derivatives length mismatch")
)

func LinearInterpolation(data []float64, x float64) (float64, error) {
	if len(data) < 2 {
		return 0, ErrNotEnoughData
	}
	for i := 0; i < len(data)-1; i++ {
		x0 := float64(i)
		x1 := float64(i + 1)
		if x >= x0 && x <= x1 {
			y0 := data[i]
			y1 := data[i+1]
			return y0 + (y1-y0)*(x-x0)/(x1-x0), nil
		}
	}
	return 0, ErrOutOfBounds
}

func QuadraticInterpolation(data []float64, x float64) (float64, error) {
	if len(data) < 3 {
		return 0, ErrNotEnoughData
	}
	for i := 0; i < len(data)-2; i++ {
		x0 := float64(i)
		x1 := float64(i + 1)
		x2 := float64(i + 2)
		if x >= x0 && x <= x2 {
			y0 := data[i]
			y1 := data[i+1]
			y2 := data[i+2]
			l0 := (x - x1) * (x - x2) / ((x0 - x1) * (x0 - x2))
			l1 := (x - x0) * (x - x2) / ((x1 - x0) * (x1 - x2))
			l2 := (x - x0) * (x - x1) / ((x2 - x0) * (x2 - x1))
			return y0*l0 + y1*l1 + y2*l2, nil
		}
	}
	return 0, ErrOutOfBounds
}

func LagrangeInterpolation(data []float64, x float64) (float64, error) {
	n := len(data)
	if n < 2 {
		return 0, ErrNotEnoughData
	}
	result := 0.0
	for i := 0; i < n; i++ {
		term := data[i]
		for j := 0; j < n; j++ {
			if i != j {
				term *= (x - float64(j)) / (float64(i) - float64(j))
			}
		}
		result += term
	}
	return result, nil
}

func NearestNeighborInterpolation(data []float64, x float64) (float64, error) {
	if len(data) == 0 {
		return 0, ErrOutOfBounds
	}
	closestIndex := 0
	minDiff := math.Abs(x - 0)
	for i := 1; i < len(data); i++ {
		diff := math.Abs(x - float64(i))
		if diff < minDiff {
			closestIndex = i
			minDiff = diff
		}
	}
	if closestIndex < 0 || closestIndex >= len(data) {
		return 0, ErrOutOfBounds
	}
	return data[closestIndex], nil
}

func NewtonInterpolation(data []float64, x float64) (float64, error) {
	n := len(data)
	if n < 2 {
		return 0, ErrNotEnoughData
	}
	dividedDiff := make([]float64, n)
	copy(dividedDiff, data)
	for i := 1; i < n; i++ {
		for j := n - 1; j >= i; j-- {
			dividedDiff[j] = (dividedDiff[j] - dividedDiff[j-1]) / (float64(j) - float64(j-i))
		}
	}
	result := dividedDiff[n-1]
	for i := n - 2; i >= 0; i-- {
		result = result*(x-float64(i)) + dividedDiff[i]
	}
	return result, nil
}

// HermiteInterpolation evaluates the Hermite interpolant of data and its
// derivatives at x. The nodes are the indices 0, 1, ... len(data)-1, so the
// result passes through every data[i] at x = i AND has slope derivatives[i]
// there.
//
// It uses the standard basis
//
//	H_i(x)  = (1 - 2(x - x_i) L'_i(x_i)) L_i(x)^2
//	Ĥ_i(x) = (x - x_i) L_i(x)^2
//
// where L_i is the Lagrange basis polynomial and L'_i(x_i) = Σ_{j≠i} 1/(x_i - x_j).
// The earlier implementation used L_i itself instead of L_i squared and left
// out the derivative correction factor, so it met none of the slope
// conditions and was not a Hermite interpolant at all.
func HermiteInterpolation(data []float64, derivatives []float64, x float64) (float64, error) {
	n := len(data)
	if n != len(derivatives) {
		return 0, ErrLengthMismatch
	}
	if n < 2 {
		return 0, ErrNotEnoughData
	}
	result := 0.0
	for i := 0; i < n; i++ {
		xi := float64(i)
		li := 1.0      // L_i(x)
		lipAtXi := 0.0 // L'_i(x_i)
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			xj := float64(j)
			li *= (x - xj) / (xi - xj)
			lipAtXi += 1.0 / (xi - xj)
		}
		li2 := li * li
		h := (1 - 2*(x-xi)*lipAtXi) * li2
		hHat := (x - xi) * li2
		result += data[i]*h + derivatives[i]*hHat
	}
	return result, nil
}
