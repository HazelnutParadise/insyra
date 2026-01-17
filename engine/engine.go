package engine

import (
	"github.com/HazelnutParadise/insyra/internal/algorithms"
	"github.com/HazelnutParadise/insyra/internal/core"
)

// BiIndex is a two-way index used by DataTable internals.
type BiIndex = core.BiIndex

// NewBiIndex creates a new BiIndex with the given capacity hint.
func NewBiIndex(cap int) *BiIndex {
	return core.NewBiIndex(cap)
}

// GetTypeSortingRank returns the type rank for sorting mixed types.
func GetTypeSortingRank(v any) int {
	return algorithms.GetTypeSortingRank(v)
}

// CompareAny compares two values of any type for mixed-type sorting.
func CompareAny(a, b any) int {
	return algorithms.CompareAny(a, b)
}

// ParallelSortStableFunc performs a stable, parallel sort using cmp.
func ParallelSortStableFunc[S ~[]E, E any](x S, cmp func(E, E) int) {
	algorithms.ParallelSortStableFunc(x, cmp)
}
