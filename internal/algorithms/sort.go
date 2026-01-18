package algorithms

import (
	"fmt"
	"math"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/HazelnutParadise/insyra/internal/utils"
)

// GetTypeSortingRank returns the type rank for sorting mixed types.
// Lower rank means higher priority (comes first in ascending order).
func GetTypeSortingRank(v any) int {
	if v == nil {
		return 0
	}
	switch v.(type) {
	case bool:
		return 1
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return 2
	case string:
		return 3
	case time.Time:
		return 4
	default:
		return 5
	}
}

// CompareAny compares two values of any type and returns:
// -1 if a < b
//
//	0 if a == b
//	1 if a > b
//
// It uses type ranking and type-specific comparison logic.
func CompareAny(a, b any) int {
	typeRankA := GetTypeSortingRank(a)
	typeRankB := GetTypeSortingRank(b)
	if typeRankA != typeRankB {
		return typeRankA - typeRankB
	}
	// Same type rank, compare values
	var cmp int
	switch va := a.(type) {
	case string:
		if vb, ok := b.(string); ok {
			cmp = strings.Compare(va, vb)
		} else {
			cmp = strings.Compare(va, fmt.Sprint(b))
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		fa := utils.ToFloat64(a)
		if fb, ok := utils.ToFloat64Safe(b); ok {
			if fa < fb {
				cmp = -1
			} else if fa > fb {
				cmp = 1
			} else {
				cmp = 0
			}
		} else {
			cmp = strings.Compare(fmt.Sprint(a), fmt.Sprint(b))
		}
	case time.Time:
		if vb, ok := b.(time.Time); ok {
			if va.Before(vb) {
				cmp = -1
			} else if va.After(vb) {
				cmp = 1
			} else {
				cmp = 0
			}
		} else {
			cmp = strings.Compare(fmt.Sprint(a), fmt.Sprint(b))
		}
	default:
		cmp = strings.Compare(fmt.Sprint(a), fmt.Sprint(b))
	}
	return cmp
}

// ParallelSortStableFunc sorts the slice x in ascending order as determined by the cmp function.
// It is a parallelized version of slices.SortStableFunc, using goroutines to improve performance on large datasets.
// The function maintains stability: equal elements preserve their original order.
// This optimized version uses adaptive goroutines scaling and improved chunking strategy.
func ParallelSortStableFunc[S ~[]E, E any](x S, cmp func(E, E) int) {
	n := len(x)
	if n <= 1 {
		return
	}

	// Use sequential sort for small arrays
	if n < 4910 {
		slices.SortStableFunc(x, cmp)
		return
	}

	// Determine optimal number of goroutines based on data size
	numGoroutines := min(getOptimalGoroutines(n), runtime.NumCPU())

	// Sort chunks in parallel using the same logic as the default version
	sortChunksOptimized(x, cmp, numGoroutines)

	// Merge chunks using the original stable merge
	ParallelMergeStable(x, cmp, numGoroutines)
}

// ParallelMergeStable merges the sorted chunks in the slice x.
// It assumes x is divided into numChunks sorted sub-slices.
func ParallelMergeStable[S ~[]E, E any](x S, cmp func(E, E) int, numChunks int) {
	n := len(x)
	if numChunks <= 1 {
		return
	}

	chunkSize := n / numChunks
	temp := make(S, n)
	copy(temp, x)

	// Merge pairs of chunks
	for size := 1; size < numChunks; size *= 2 {
		for left := 0; left < numChunks-size; left += 2 * size {
			mid := left + size
			right := min(left+2*size, numChunks)

			leftStart := left * chunkSize
			midStart := mid * chunkSize
			rightEnd := right * chunkSize
			if right == numChunks {
				rightEnd = n
			}

			mergeStable(temp[leftStart:midStart], temp[midStart:rightEnd], x[leftStart:rightEnd], cmp)
		}
		copy(temp, x)
	}
}

// sortChunksOptimized sorts data chunks in parallel with consistent chunking.
func sortChunksOptimized[S ~[]E, E any](x S, cmp func(E, E) int, numChunks int) {
	n := len(x)
	chunkSize := n / numChunks
	if chunkSize == 0 {
		chunkSize = 1
	}

	var wg sync.WaitGroup
	for i := range numChunks {
		start := i * chunkSize
		end := start + chunkSize
		if i == numChunks-1 {
			end = n
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			slices.SortStableFunc(x[start:end], cmp)
		}(start, end)
	}
	wg.Wait()
}

// mergeStable merges two sorted slices a and b into dst, maintaining stability.
func mergeStable[S ~[]E, E any](a, b, dst S, cmp func(E, E) int) {
	i, j, k := 0, 0, 0
	for i < len(a) && j < len(b) {
		if cmp(a[i], b[j]) <= 0 {
			dst[k] = a[i]
			i++
		} else {
			dst[k] = b[j]
			j++
		}
		k++
	}
	for i < len(a) {
		dst[k] = a[i]
		i++
		k++
	}
	for j < len(b) {
		dst[k] = b[j]
		j++
		k++
	}
}

// getOptimalGoroutines returns the optimal number of goroutines for a given data size.
func getOptimalGoroutines(n int) int {
	// Adaptive growth strategy: slow growth for small datasets, faster growth for large datasets
	if n < 10000 {
		// For small datasets: use logarithmic-like growth
		if n < 5500 {
			return 2
		} else if n < 6500 {
			return 3
		} else if n < 7500 {
			return 4
		} else if n < 8500 {
			return 5
		} else {
			return 6
		}
	} else if n < 50000 {
		// For medium datasets: moderate linear growth
		return 6 + (n-10000)/5000 // Increases by 1 every 5000 elements
	} else if n < 200000 {
		// For large datasets: accelerated growth
		return 12 + (n-50000)/15000 // Increases by 1 every 15000 elements
	} else {
		// For very large datasets: use square root scaling
		goroutines := int(math.Sqrt(float64(n)) / 50) // sqrt(n)/50 gives good scaling
		if goroutines > runtime.NumCPU() {
			goroutines = runtime.NumCPU()
		}
		if goroutines < 16 {
			goroutines = 16 // Minimum for very large datasets
		}
		return goroutines
	}
}
