package benchmark

import (
	"testing"
	"github.com/HazelnutParadise/insyra"
)

func createLargeDataList(size int) *insyra.DataList {
	data := make([]any, size)
	for i := 0; i < size; i++ {
		data[i] = float64(i)
	}
	return insyra.NewDataList(data...)
}

func BenchmarkDataList_Sum(b *testing.B) {
	dl := createLargeDataList(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dl.Sum()
	}
}

func BenchmarkDataList_Mean(b *testing.B) {
	dl := createLargeDataList(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dl.Mean()
	}
}

func BenchmarkDataList_Max(b *testing.B) {
	dl := createLargeDataList(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dl.Max()
	}
}

func BenchmarkDataList_Count(b *testing.B) {
	dl := createLargeDataList(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dl.Count(float64(5000))
	}
}
