package insyra_test

import (
	"strconv"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// Every search, count, replace and drop runs one comparison per cell, so the
// cost of that comparison is the cost of the call. These cover the three
// common cell types over a million cells.

var valueLookupSink int

func millionCells(mk func(i int) any) *insyra.DataList {
	vals := make([]any, 1_000_000)
	for i := range vals {
		vals[i] = mk(i)
	}
	return insyra.NewDataList(vals...)
}

func BenchmarkCountInt64(b *testing.B) {
	dl := millionCells(func(i int) any { return int64(i % 100) })
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valueLookupSink = dl.Count(7)
	}
}

func BenchmarkCountFloat(b *testing.B) {
	dl := millionCells(func(i int) any { return float64(i % 100) })
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valueLookupSink = dl.Count(7.0)
	}
}

func BenchmarkCountString(b *testing.B) {
	dl := millionCells(func(i int) any { return strconv.Itoa(i % 100) })
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valueLookupSink = dl.Count("7")
	}
}
