package insyra

import "math"

// LinearInterpolation performs linear interpolation for the given x value using the DataList.
func (dl *DataList) LinearInterpolation(x float64) float64 {
	var earlyResult *float64
	dl.AtomicDo(func(l *DataList) {
		if l.Len() < 2 {
			LogWarning("DataList", "LinearInterpolation", "Not enough data points")
			earlyResult = new(float64)
			*earlyResult = math.NaN()
			return
		}
		for i := 0; i < len(l.data)-1; i++ {
			x0 := float64(i)
			x1 := float64(i + 1)
			y0 := l.data[i].(float64)
			y1 := l.data[i+1].(float64)

			if x >= x0 && x <= x1 {
				earlyResult = new(float64)
				*earlyResult = y0 + (y1-y0)*(x-x0)/(x1-x0)
				return
			}
		}
	})
	if earlyResult != nil {
		return *earlyResult
	}
	LogWarning("DataList", "LinearInterpolation", "X value out of bounds")
	return math.NaN()
}

// QuadraticInterpolation performs quadratic interpolation for the given x value using the DataList.
func (dl *DataList) QuadraticInterpolation(x float64) float64 {
	var earlyResult *float64
	dl.AtomicDo(func(l *DataList) {
		if l.Len() < 3 {
			LogWarning("DataList", "QuadraticInterpolation", "Not enough data points")
			earlyResult = new(float64)
			*earlyResult = math.NaN()
			return
		}
		for i := 0; i < len(l.data)-2; i++ {
			x0 := float64(i)
			x1 := float64(i + 1)
			x2 := float64(i + 2)
			y0 := l.data[i].(float64)
			y1 := l.data[i+1].(float64)
			y2 := l.data[i+2].(float64)

			if x >= x0 && x <= x2 {
				l0 := (x - x1) * (x - x2) / ((x0 - x1) * (x0 - x2))
				l1 := (x - x0) * (x - x2) / ((x1 - x0) * (x1 - x2))
				l2 := (x - x0) * (x - x1) / ((x2 - x0) * (x2 - x1))
				earlyResult = new(float64)
				*earlyResult = y0*l0 + y1*l1 + y2*l2
				return
			}
		}
	})
	if earlyResult != nil {
		return *earlyResult
	}
	LogWarning("DataList", "QuadraticInterpolation", "X value out of bounds")
	return math.NaN()
}

// LagrangeInterpolation performs Lagrange interpolation for the given x value using the DataList.
func (dl *DataList) LagrangeInterpolation(x float64) float64 {
	var n int
	var floatData []float64
	dl.AtomicDo(func(l *DataList) {
		n = l.Len()
		floatData = l.ToF64Slice()
	})
	if n < 2 {
		LogWarning("DataList", "LagrangeInterpolation", "Not enough data points")
		return math.NaN()
	}
	result := 0.0
	for i := 0; i < n; i++ {
		term := floatData[i]
		for j := 0; j < n; j++ {
			if i != j {
				term *= (x - float64(j)) / (float64(i) - float64(j))
			}
		}
		result += term
	}
	return result
}

// NearestNeighborInterpolation performs nearest-neighbor interpolation for the given x value using the DataList.
func (dl *DataList) NearestNeighborInterpolation(x float64) float64 {
	closestIndex := 0
	minDiff := math.Abs(x - 0) // 初始化差異

	var dataLen int
	var floatData []float64
	dl.AtomicDo(func(l *DataList) {
		dataLen = l.Len()
		floatData = l.ToF64Slice()
	})

	for i := 1; i < dataLen; i++ {
		diff := math.Abs(x - float64(i))
		if diff < minDiff {
			closestIndex = i
			minDiff = diff
		}
	}

	if closestIndex < 0 || closestIndex >= dl.Len() {
		LogWarning("DataList", "NearestNeighborInterpolation", "X value out of bounds")
		return math.NaN()
	}
	return floatData[closestIndex]
}

// NewtonInterpolation performs Newton's interpolation for the given x value using the DataList.
func (dl *DataList) NewtonInterpolation(x float64) float64 {
	var dataLen int
	var floatData []float64
	dl.AtomicDo(func(l *DataList) {
		dataLen = l.Len()
		floatData = l.ToF64Slice()
	})
	n := dataLen
	if n < 2 {
		LogWarning("DataList", "NewtonInterpolation", "Not enough data points")
		return math.NaN()
	}
	// 計算差分
	dividedDiff := make([]float64, n)
	copy(dividedDiff, floatData)
	for i := 1; i < n; i++ {
		for j := n - 1; j >= i; j-- {
			dividedDiff[j] = (dividedDiff[j] - dividedDiff[j-1]) / (float64(j) - float64(j-i))
		}
	}

	// 使用差分進行插值
	result := dividedDiff[n-1]
	for i := n - 2; i >= 0; i-- {
		result = result*(x-float64(i)) + dividedDiff[i]
	}
	return result
}

// HermiteInterpolation performs Hermite interpolation for the given x value using the DataList.
func (dl *DataList) HermiteInterpolation(x float64, derivatives []float64) float64 {
	var n int
	var floatData []float64
	dl.AtomicDo(func(l *DataList) {
		n = l.Len()
		floatData = l.ToF64Slice()
	})
	if n != len(derivatives) {
		LogWarning("DataList", "HermiteInterpolation", "Data and derivatives length mismatch")
		return math.NaN()
	}
	if n < 2 {
		LogWarning("DataList", "HermiteInterpolation", "Not enough data points")
		return math.NaN()
	}

	result := 0.0
	for i := 0; i < n; i++ {
		h := 1.0
		for j := 0; j < n; j++ {
			if i != j {
				h *= (x - float64(j)) / (float64(i) - float64(j))
			}
		}
		result += floatData[i]*h + derivatives[i]*h*(x-float64(i))
	}
	return result
}
