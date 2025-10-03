package utils

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
				ParallelSortStableFuncDefault(copyData, func(a, b int) int {
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
	ParallelSortStableFuncDefault(data, func(a, b int) int {
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
	ParallelSortStableFuncDefault(data2, func(a, b item) int {
		return a.value - b.value
	})
	if !slices.Equal(data2, expected2) {
		t.Errorf("ParallelSortStableFunc stability failed: got %v, want %v", data2, expected2)
	}
}

func TestSortComparison(t *testing.T) {
	sizes := []int{10000, 100000, 1000000}

	for _, size := range sizes {
		data := make([]int, size)
		for i := range data {
			data[i] = rand.Intn(size)
		}

		// Test slices.SortFunc (not stable)
		copyData1 := make([]int, size)
		copy(copyData1, data)
		start1 := time.Now()
		slices.SortFunc(copyData1, func(a, b int) int {
			return a - b
		})
		time1 := time.Since(start1)

		// Test slices.SortStableFunc
		copyData2 := make([]int, size)
		copy(copyData2, data)
		start2 := time.Now()
		slices.SortStableFunc(copyData2, func(a, b int) int {
			return a - b
		})
		time2 := time.Since(start2)

		// Test our parallel stable sort
		copyData3 := make([]int, size)
		copy(copyData3, data)
		start3 := time.Now()
		ParallelSortStableFuncDefault(copyData3, func(a, b int) int {
			return a - b
		})
		time3 := time.Since(start3)

		// Verify SortFunc and SortStableFunc results are sorted
		for i := 1; i < len(copyData1); i++ {
			if copyData1[i] < copyData1[i-1] {
				t.Errorf("SortFunc result not sorted at index %d", i)
			}
		}
		for i := 1; i < len(copyData2); i++ {
			if copyData2[i] < copyData2[i-1] {
				t.Errorf("SortStableFunc result not sorted at index %d", i)
			}
		}
		for i := 1; i < len(copyData3); i++ {
			if copyData3[i] < copyData3[i-1] {
				t.Errorf("ParallelSortStableFunc result not sorted at index %d", i)
			}
		}

		t.Logf("Size %d:", size)
		t.Logf("  SortFunc (not stable): %v", time1)
		t.Logf("  SortStableFunc: %v", time2)
		t.Logf("  ParallelSortStableFunc: %v", time3)
		t.Logf("  Parallel vs SortFunc: %.2fx", float64(time1)/float64(time3))
		t.Logf("  Parallel vs SortStableFunc: %.2fx", float64(time2)/float64(time3))
	}
}
