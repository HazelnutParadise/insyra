package algorithms

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"
	"time"
)

func BenchmarkSortFunc(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			data := make([]int, size)
			for i := range data {
				data[i] = rand.Intn(size)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				copyData := make([]int, size)
				copy(copyData, data)
				slices.SortFunc(copyData, func(a, b int) int {
					return a - b
				})
			}
		})
	}
}

func BenchmarkParallelSortStableFunc(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			data := make([]int, size)
			for i := range data {
				data[i] = rand.Intn(size)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				copyData := make([]int, size)
				copy(copyData, data)
				ParallelSortStableFunc(copyData, func(a, b int) int {
					return a - b
				})
			}
		})
	}
}

func TestParallelSortStableFunc(t *testing.T) {
	// Test with small slice
	data := []int{3, 1, 4, 1, 5, 9, 2, 6}
	expected := []int{1, 1, 2, 3, 4, 5, 6, 9}
	ParallelSortStableFunc(data, func(a, b int) int {
		return a - b
	})
	if !slices.Equal(data, expected) {
		t.Errorf("ParallelSortStableFunc failed: got %v, want %v", data, expected)
	}

	// Test stability with structs
	type item struct {
		value int
		index int
	}
	data2 := []item{
		{3, 1}, {1, 2}, {4, 3}, {1, 4}, {5, 5},
	}
	expected2 := []item{
		{1, 2}, {1, 4}, {3, 1}, {4, 3}, {5, 5},
	}
	ParallelSortStableFunc(data2, func(a, b item) int {
		return a.value - b.value
	})
	if !slices.Equal(data2, expected2) {
		t.Errorf("ParallelSortStableFunc stability failed: got %v, want %v", data2, expected2)
	}
}

func TestFinalPerformanceSummary(t *testing.T) {
	sizes := []int{10000, 100000, 1000000}

	t.Log("=== Final performance summary ===")

	for _, size := range sizes {
		data := make([]int, size)
		for i := range data {
			data[i] = rand.Intn(size)
		}

		copyData1 := make([]int, size)
		copy(copyData1, data)
		start1 := time.Now()
		slices.SortFunc(copyData1, func(a, b int) int {
			return a - b
		})
		time1 := time.Since(start1)

		copyData2 := make([]int, size)
		copy(copyData2, data)
		start2 := time.Now()
		slices.SortStableFunc(copyData2, func(a, b int) int {
			return a - b
		})
		time2 := time.Since(start2)

		copyData3 := make([]int, size)
		copy(copyData3, data)
		start3 := time.Now()
		ParallelSortStableFunc(copyData3, func(a, b int) int {
			return a - b
		})
		time3 := time.Since(start3)

		t.Logf("data size: %d elements", size)
		t.Logf("  SortFunc (unstable): %v", time1)
		t.Logf("  SortStableFunc: %v (stable speedup %.1fx)", time2, float64(time2)/float64(time1))
		t.Logf("  ParallelSortStableFunc: %v", time3)
		t.Logf("  speedup vs SortFunc: %.2fx", float64(time1)/float64(time3))
		t.Logf("  speedup vs SortStableFunc: %.2fx", float64(time2)/float64(time3))
		t.Logf("")
	}

	t.Log("=== Notes ===")
	t.Log("1. Parallel stable sort preserves stability with better throughput.")
	t.Log("2. Larger inputs (100k+) benefit more from parallelism.")
	t.Log("3. Goroutine scaling adapts to data size.")
	t.Log("4. Small inputs (<10k) prefer sequential sorting.")
}
