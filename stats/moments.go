package stats

import (
	"errors"
	"math"
	"runtime"
	"sync"

	"github.com/HazelnutParadise/insyra"
)

// intPow returns x^n for non-negative integer n by repeated multiplication.
// Replaces math.Pow(x, float64(n)) in the general-order moment paths:
// math.Pow walks an exp(n·log(x)) implementation regardless of n, costing
// 1–2 ULPs of precision per call; for the integer exponents we hit here
// (5..50 range, used in higher-order moments) the multiplicative form is
// both faster and bit-exact except for a single rounding per multiply.
func intPow(x float64, n int) float64 {
	if n < 0 {
		return math.Pow(x, float64(n)) // negative exponent: fall back
	}
	r := 1.0
	for n > 0 {
		if n&1 == 1 {
			r *= x
		}
		x *= x
		n >>= 1
	}
	return r
}

// CalculateMoment calculates the n-th moment of the DataList.
// If central is true, it computes the central moment; otherwise, raw moment.
func CalculateMoment(dl insyra.IDataList, n int, central bool) (float64, error) {
	if n <= 0 {
		return math.NaN(), errors.New("invalid moment order")
	}

	var length int
	var mean float64
	var floatData []float64
	var result float64
	hasResult := false
	dl.AtomicDo(func(l *insyra.DataList) {
		length = l.Len()
		if length == 0 {
			result = math.NaN()
			hasResult = true
			return
		}

		if central {
			if n == 1 {
				result = 0.0
				hasResult = true
				return
			}
			if n == 2 {
				result = l.VarP()
				hasResult = true
				return
			}
		} else if n == 1 {
			result = l.Mean()
			hasResult = true
			return
		}

		floatData = l.ToF64Slice()
		if central {
			mean = l.Mean()
		}
	})

	if hasResult {
		if math.IsNaN(result) {
			return result, errors.New("empty data")
		}
		return result, nil
	}

	if length > 10000 && n > 2 {
		return calculateMomentParallel(floatData, n, central, mean, length), nil
	}

	return calculateMomentSequential(floatData, n, central, mean, length), nil
}

func calculateMomentSequential(data []float64, n int, central bool, mean float64, length int) float64 {
	if central {
		switch n {
		case 2:
			return calculateCentralMoment2(data, mean, length)
		case 3:
			return calculateCentralMoment3(data, mean, length)
		case 4:
			return calculateCentralMoment4(data, mean, length)
		default:
			return calculateGeneralCentralMoment(data, mean, n, length)
		}
	}

	switch n {
	case 2:
		return calculateRawMoment2(data, length)
	case 3:
		return calculateRawMoment3(data, length)
	case 4:
		return calculateRawMoment4(data, length)
	default:
		return calculateGeneralRawMoment(data, n, length)
	}
}

func calculateMomentParallel(data []float64, n int, central bool, mean float64, length int) float64 {
	numCPU := runtime.NumCPU()
	chunkSize := (length + numCPU - 1) / numCPU

	var wg sync.WaitGroup
	results := make([]float64, numCPU)

	for i := range numCPU {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			start := i * chunkSize
			end := start + chunkSize
			if end > length {
				end = length
			}

			var sum float64
			if central {
				for j := start; j < end; j++ {
					val := data[j]
					diff := val - mean
					switch n {
					case 2:
						sum += diff * diff
					case 3:
						sum += diff * diff * diff
					case 4:
						diff2 := diff * diff
						sum += diff2 * diff2
					default:
						sum += intPow(diff, n)
					}
				}
			} else {
				for j := start; j < end; j++ {
					val := data[j]
					switch n {
					case 2:
						sum += val * val
					case 3:
						sum += val * val * val
					case 4:
						val2 := val * val
						sum += val2 * val2
					default:
						sum += intPow(val, n)
					}
				}
			}

			results[i] = sum
		}(i)
	}

	wg.Wait()

	var totalSum float64
	for _, r := range results {
		totalSum += r
	}

	return totalSum / float64(length)
}

func calculateCentralMoment2(data []float64, mean float64, length int) float64 {
	var sum2 float64
	for _, val := range data {
		diff := val - mean
		sum2 += diff * diff
	}
	return sum2 / float64(length)
}

func calculateCentralMoment3(data []float64, mean float64, length int) float64 {
	var sum float64
	for _, val := range data {
		diff := val - mean
		diff2 := diff * diff
		sum += diff2 * diff
	}
	return sum / float64(length)
}

func calculateCentralMoment4(data []float64, mean float64, length int) float64 {
	var sum float64
	for _, val := range data {
		diff := val - mean
		diff2 := diff * diff
		sum += diff2 * diff2
	}
	return sum / float64(length)
}

func calculateRawMoment2(data []float64, length int) float64 {
	var sum float64
	for _, val := range data {
		sum += val * val
	}
	return sum / float64(length)
}

func calculateRawMoment3(data []float64, length int) float64 {
	var sum float64
	for _, val := range data {
		sum += val * val * val
	}
	return sum / float64(length)
}

func calculateRawMoment4(data []float64, length int) float64 {
	var sum float64
	for _, val := range data {
		val2 := val * val
		sum += val2 * val2
	}
	return sum / float64(length)
}

func calculateGeneralCentralMoment(data []float64, mean float64, n int, length int) float64 {
	var sum float64
	for _, val := range data {
		sum += intPow(val-mean, n)
	}
	return sum / float64(length)
}

func calculateGeneralRawMoment(data []float64, n int, length int) float64 {
	var sum float64
	for _, val := range data {
		sum += intPow(val, n)
	}
	return sum / float64(length)
}
