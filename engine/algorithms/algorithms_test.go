package algorithms_test

import (
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra/engine/algorithms"
)

// Thin re-exports of internal/algorithms, and the only way a consumer outside
// the module can order mixed-type values the way a DataList does.

func TestCompareAnyOrdersAcrossTypes(t *testing.T) {
	if got := algorithms.CompareAny(1, 2); got >= 0 {
		t.Errorf("CompareAny(1, 2) = %d, want negative", got)
	}
	if got := algorithms.CompareAny(2, 1); got <= 0 {
		t.Errorf("CompareAny(2, 1) = %d, want positive", got)
	}
	if got := algorithms.CompareAny(1, 1); got != 0 {
		t.Errorf("CompareAny(1, 1) = %d, want 0", got)
	}
	// Numbers of different Go types compare by value.
	if got := algorithms.CompareAny(int64(2), 2.0); got != 0 {
		t.Errorf("CompareAny(int64(2), 2.0) = %d, want 0", got)
	}
	if got := algorithms.CompareAny("abc", "abd"); got >= 0 {
		t.Errorf("CompareAny(\"abc\", \"abd\") = %d, want negative", got)
	}
}

// The rank is what puts values of different kinds into a stable order; equal
// kinds share a rank and different kinds do not.
func TestGetTypeSortingRank(t *testing.T) {
	if algorithms.GetTypeSortingRank(1) != algorithms.GetTypeSortingRank(2) {
		t.Error("two ints have different type ranks")
	}
	if algorithms.GetTypeSortingRank(1) == algorithms.GetTypeSortingRank("a") {
		t.Error("an int and a string share a type rank")
	}
}

func TestParallelSortStableFunc(t *testing.T) {
	type row struct {
		key int
		seq int
	}
	// Every key repeats, so an unstable sort shows up as seq out of order.
	xs := make([]row, 0, 2000)
	for i := 0; i < 2000; i++ {
		xs = append(xs, row{key: i % 4, seq: i})
	}

	algorithms.ParallelSortStableFunc(xs, func(a, b row) int { return a.key - b.key })

	prevKey, prevSeq := -1, -1
	for i, r := range xs {
		if r.key < prevKey {
			t.Fatalf("index %d: key %d after %d — not sorted", i, r.key, prevKey)
		}
		if r.key == prevKey && r.seq < prevSeq {
			t.Fatalf("index %d: seq %d after %d within key %d — not stable", i, r.seq, prevSeq, r.key)
		}
		prevKey, prevSeq = r.key, r.seq
	}

	// An already-sorted slice comes back unchanged.
	sorted := []int{1, 2, 3, 4}
	algorithms.ParallelSortStableFunc(sorted, func(a, b int) int { return a - b })
	if want := []int{1, 2, 3, 4}; !reflect.DeepEqual(sorted, want) {
		t.Errorf("sorting an ordered slice: got %v, want %v", sorted, want)
	}

	// Empty and single-element slices do not panic.
	algorithms.ParallelSortStableFunc([]int{}, func(a, b int) int { return a - b })
	algorithms.ParallelSortStableFunc([]int{1}, func(a, b int) int { return a - b })
}
