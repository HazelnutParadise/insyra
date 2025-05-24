package stats

import (
	"math"
	"runtime"
	"sync"

	"github.com/HazelnutParadise/insyra"
)

// CalculateMoment calculates the n-th moment of the DataList.
// If central is true, it computes the central moment; otherwise, raw moment.
// Returns NaN if the DataList is empty or the moment cannot be calculated.
func CalculateMoment(dl insyra.IDataList, n int, central bool) float64 {
	// 處理無效輸入
	if n <= 0 {
		insyra.LogWarning("stats.CalculateMoment: invalid moment order")
		return math.NaN()
	}

	length := dl.Len()
	if length == 0 {
		insyra.LogWarning("stats.CalculateMoment: empty DataList")
		return math.NaN()
	}

	// 快速處理特殊情況
	if central {
		if n == 1 {
			return 0.0
		}
		if n == 2 {
			return dl.VarP()
		}
	} else {
		if n == 1 {
			return dl.Mean()
		}
	}

	// 獲取數據並預檢查
	data := dl.Data()

	// 預先驗證所有數據類型
	floatData := make([]float64, length)
	for i, v := range data {
		val, ok := v.(float64)
		if !ok {
			insyra.LogWarning("stats.CalculateMoment: data contains non-float64 value")
			return math.NaN()
		}
		floatData[i] = val
	}

	// 預先計算平均值（如果需要）
	var mean float64
	if central {
		mean = dl.Mean()
	}

	// 對於大型數據集和高階矩，考慮並行計算
	if length > 10000 && n > 2 {
		return calculateMomentParallel(floatData, n, central, mean, length)
	}

	// 對於小型數據集或低階矩，使用順序計算
	return calculateMomentSequential(floatData, n, central, mean, length)
}

// calculateMomentSequential 使用順序計算方式計算矩
func calculateMomentSequential(data []float64, n int, central bool, mean float64, length int) float64 {
	// 根據不同情況選擇計算方法
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
	} else {
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
}

// calculateMomentParallel 使用並行計算方式計算矩
func calculateMomentParallel(data []float64, n int, central bool, mean float64, length int) float64 {
	// 確定子任務數量（基於CPU核心數）
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

			// 分塊計算
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
						sum += math.Pow(diff, float64(n))
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
						sum += math.Pow(val, float64(n))
					}
				}
			}

			results[i] = sum
		}(i)
	}

	wg.Wait()

	// 合併結果
	var totalSum float64
	for _, r := range results {
		totalSum += r
	}

	return totalSum / float64(length)
}

// calculateCentralMoment2 計算第二中心矩
func calculateCentralMoment2(data []float64, mean float64, length int) float64 {
	var sum2 float64
	for _, val := range data {
		diff := val - mean
		sum2 += diff * diff
	}
	return sum2 / float64(length)
}

// calculateCentralMoment3 計算第三中心矩
func calculateCentralMoment3(data []float64, mean float64, length int) float64 {
	var sum float64
	for _, val := range data {
		diff := val - mean
		diff2 := diff * diff
		sum += diff2 * diff
	}
	return sum / float64(length)
}

// calculateCentralMoment4 計算第四中心矩
func calculateCentralMoment4(data []float64, mean float64, length int) float64 {
	var sum float64
	for _, val := range data {
		diff := val - mean
		diff2 := diff * diff
		sum += diff2 * diff2
	}
	return sum / float64(length)
}

// calculateRawMoment2 計算第二原始矩
func calculateRawMoment2(data []float64, length int) float64 {
	var sum float64
	for _, val := range data {
		sum += val * val
	}
	return sum / float64(length)
}

// calculateRawMoment3 計算第三原始矩
func calculateRawMoment3(data []float64, length int) float64 {
	var sum float64
	for _, val := range data {
		sum += val * val * val
	}
	return sum / float64(length)
}

// calculateRawMoment4 計算第四原始矩
func calculateRawMoment4(data []float64, length int) float64 {
	var sum float64
	for _, val := range data {
		val2 := val * val
		sum += val2 * val2
	}
	return sum / float64(length)
}

// calculateGeneralCentralMoment 計算一般中心矩
func calculateGeneralCentralMoment(data []float64, mean float64, n int, length int) float64 {
	var sum float64
	for _, val := range data {
		diff := val - mean
		sum += math.Pow(diff, float64(n))
	}
	return sum / float64(length)
}

// calculateGeneralRawMoment 計算一般原始矩
func calculateGeneralRawMoment(data []float64, n int, length int) float64 {
	var sum float64
	for _, val := range data {
		sum += math.Pow(val, float64(n))
	}
	return sum / float64(length)
}
