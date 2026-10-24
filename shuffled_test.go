package insyra

import "testing"

// Shuffled returns the shuffled copy and leaves its receiver alone. On this
// line Shuffle does the same, and for one seed both give the same order; 0.4
// makes Shuffle reorder the receiver, so callers who want a copy use Shuffled.
func TestDataListShuffledLeavesTheListAlone(t *testing.T) {
	opts := SamplingOptions{UseSeed: true, Seed: 42}
	dl := NewDataList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10).SetName("x")

	got := dl.Shuffled(opts)
	if !dl.IsEqualTo(NewDataList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)) {
		t.Fatalf("Shuffled changed the list: %v", dl.Data())
	}
	if got.GetName() != "x_Shuffled" || got.Len() != 10 {
		t.Fatalf("got name %q, length %d", got.GetName(), got.Len())
	}
	if !got.IsEqualTo(dl.Shuffle(opts)) {
		t.Fatal("Shuffled and Shuffle gave different orders for one seed")
	}
	if got.IsEqualTo(dl) {
		t.Fatal("seed 42 left ten elements in order")
	}
}

func TestDataTableShuffledLeavesTheTableAlone(t *testing.T) {
	opts := SamplingOptions{UseSeed: true, Seed: 42}
	dt := NewDataTable(NewDataList(1, 2, 3, 4, 5, 6).SetName("a"), NewDataList("a", "b", "c", "d", "e", "f").SetName("b"))
	dt.SetRowNames([]string{"r1", "r2", "r3", "r4", "r5", "r6"})
	before := dt.Clone()

	got := dt.Shuffled(opts)
	if !dt.IsTheSameAs(before) {
		t.Fatal("Shuffled changed the table")
	}
	if got.NumRows() != 6 || got.NumCols() != 2 {
		t.Fatalf("got %d rows and %d columns", got.NumRows(), got.NumCols())
	}
	if !got.IsEqualTo(dt.Shuffle(opts)) {
		t.Fatal("Shuffled and Shuffle gave different tables for one seed")
	}
	// Rows move whole: each number still sits beside its letter.
	for r := 0; r < got.NumRows(); r++ {
		n := got.GetElementByNumberIndex(r, 0).(int)
		if want := string(rune('a' + n - 1)); got.GetElementByNumberIndex(r, 1) != want {
			t.Fatalf("row %d holds %v beside %v", r, n, got.GetElementByNumberIndex(r, 1))
		}
	}
}
