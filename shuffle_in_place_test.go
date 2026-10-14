package insyra

import (
	"slices"
	"testing"
)

// Shuffle is the in-place method and Shuffled the copying one, after Swift's
// and Kotlin's shuffle/shuffled pair (#218, D-7).

func shuffleTestList() *DataList {
	values := make([]any, 20)
	for i := range values {
		values[i] = i + 1
	}
	return NewDataList(values...).SetName("ids")
}

func TestDataListShuffleChangesTheListInPlace(t *testing.T) {
	seed := SamplingOptions{UseSeed: true, Seed: 9}
	dl := shuffleTestList()
	original := dl.Data()

	got := dl.Shuffle(seed)

	if got != dl {
		t.Fatalf("Shuffle returned another list; it must return its receiver")
	}
	if dl.Err() != nil {
		t.Fatalf("Shuffle recorded an error: %v", dl.Err())
	}
	if dl.GetName() != "ids" {
		t.Fatalf("Shuffle renamed the list to %q", dl.GetName())
	}
	if slices.Equal(dl.Data(), original) {
		t.Fatalf("Shuffle left the list in its original order")
	}
	sorted := dl.Clone().Sort().Data()
	if !slices.Equal(sorted, original) {
		t.Fatalf("Shuffle changed the values: sorted %v, want %v", sorted, original)
	}
}

func TestDataListShuffledLeavesTheListAndReturnsACopy(t *testing.T) {
	seed := SamplingOptions{UseSeed: true, Seed: 9}
	dl := shuffleTestList()
	original := dl.Data()

	got := dl.Shuffled(seed)

	if got == dl {
		t.Fatalf("Shuffled returned its receiver; it must return a new list")
	}
	if !slices.Equal(dl.Data(), original) {
		t.Fatalf("Shuffled changed the receiver to %v", dl.Data())
	}
	if got.GetName() != "ids_Shuffled" {
		t.Fatalf("Shuffled name = %q, want ids_Shuffled", got.GetName())
	}
	if slices.Equal(got.Data(), original) {
		t.Fatalf("Shuffled returned the original order")
	}
}

func TestDataListShuffleAndShuffledGiveTheSameOrderForOneSeed(t *testing.T) {
	for seed := uint64(0); seed < 50; seed++ {
		opts := SamplingOptions{UseSeed: true, Seed: seed}
		copied := shuffleTestList().Shuffled(opts).Data()
		inPlace := shuffleTestList().Shuffle(opts).Data()
		if !slices.Equal(copied, inPlace) {
			t.Fatalf("seed %d: Shuffle gave %v, Shuffled gave %v", seed, inPlace, copied)
		}
	}
}

func TestDataListShuffleFailureLeavesTheList(t *testing.T) {
	seed := SamplingOptions{UseSeed: true, Seed: 9}
	dl := shuffleTestList()
	original := dl.Data()

	got := dl.Shuffle(seed, seed)

	if got != dl {
		t.Fatalf("a failed Shuffle must still return its receiver")
	}
	if !slices.Equal(dl.Data(), original) {
		t.Fatalf("a failed Shuffle changed the list to %v", dl.Data())
	}
	if dl.Err() == nil {
		t.Fatalf("Shuffle with two options recorded no error")
	}
}

func TestDataListShuffledFailureCarriesTheError(t *testing.T) {
	seed := SamplingOptions{UseSeed: true, Seed: 9}
	dl := shuffleTestList()
	original := dl.Data()

	got := dl.Shuffled(seed, seed)

	if got == dl || got.Len() != 0 {
		t.Fatalf("a failed Shuffled must return a new empty list, got %v", got.Data())
	}
	if got.Err() == nil || dl.Err() == nil {
		t.Fatalf("a failed Shuffled must record the error on the result and the receiver: result %v, receiver %v", got.Err(), dl.Err())
	}
	if !slices.Equal(dl.Data(), original) {
		t.Fatalf("a failed Shuffled changed the receiver to %v", dl.Data())
	}
}

func TestDataListShuffleOfAnEmptyListIsNotAnError(t *testing.T) {
	empty := NewDataList()
	if got := empty.Shuffle(); got != empty || got.Len() != 0 {
		t.Fatalf("Shuffle of an empty list must return the empty receiver")
	}
	if got := empty.Shuffled(); got.Len() != 0 {
		t.Fatalf("Shuffled of an empty list must be empty")
	}
	if empty.Err() != nil {
		t.Fatalf("shuffling an empty list recorded an error: %v", empty.Err())
	}
}

func TestDataTableShuffleChangesTheTableInPlace(t *testing.T) {
	seed := SamplingOptions{UseSeed: true, Seed: 4}
	dt := samplingTestTable().SetName("people")
	want := samplingTestTable().SetName("people").Shuffled(seed)

	got := dt.Shuffle(seed)

	if got != dt {
		t.Fatalf("Shuffle returned another table; it must return its receiver")
	}
	if dt.Err() != nil {
		t.Fatalf("Shuffle recorded an error: %v", dt.Err())
	}
	if dt.GetName() != "people" {
		t.Fatalf("Shuffle renamed the table to %q", dt.GetName())
	}
	if !slices.Equal(dt.ColNames(), []string{"id", "value", "label"}) {
		t.Fatalf("Shuffle changed the column names to %v", dt.ColNames())
	}
	if !dataTablesRowsEqual(dt, want) {
		t.Fatalf("Shuffle and Shuffled disagree for one seed:\n in place %v %v\n copy     %v %v",
			dt.GetColByNumber(0).Data(), dt.RowNames(), want.GetColByNumber(0).Data(), want.RowNames())
	}
	if slices.Equal(dt.GetColByNumber(0).Data(), []any{1, 2, 3, 4, 5}) {
		t.Fatalf("Shuffle left the rows in their original order")
	}
	assertSamplingRowsAligned(t, dt)
	for i, name := range dt.RowNames() {
		id := dt.GetElementByNumberIndex(i, 0).(int)
		if name != "r"+string(rune('0'+id)) {
			t.Fatalf("row %d: name %q does not follow its row (id %d)", i, name, id)
		}
	}
}

func TestDataTableShuffledLeavesTheTable(t *testing.T) {
	seed := SamplingOptions{UseSeed: true, Seed: 4}
	dt := samplingTestTable()
	original := samplingTestTable()

	got := dt.Shuffled(seed)

	if got == dt {
		t.Fatalf("Shuffled returned its receiver; it must return a new table")
	}
	if !dataTablesRowsEqual(dt, original) {
		t.Fatalf("Shuffled changed the receiver")
	}
	assertSamplingRowsAligned(t, got)
}

func TestDataTableShuffleFailureLeavesTheTable(t *testing.T) {
	seed := SamplingOptions{UseSeed: true, Seed: 4}
	dt := samplingTestTable()

	got := dt.Shuffle(seed, seed)

	if got != dt {
		t.Fatalf("a failed Shuffle must still return its receiver")
	}
	if !dataTablesRowsEqual(dt, samplingTestTable()) {
		t.Fatalf("a failed Shuffle changed the table")
	}
	if dt.Err() == nil {
		t.Fatalf("Shuffle with two options recorded no error")
	}
}

func TestDataTableShufflePadsAShortColumnLikeShuffled(t *testing.T) {
	seed := SamplingOptions{UseSeed: true, Seed: 7}
	build := func() *DataTable {
		dt := NewDataTable()
		dt.AppendCols(NewDataList(1, 2, 3, 4).SetName("a"), NewDataList("x", "y").SetName("b"))
		return dt
	}
	want := build().Shuffled(seed)
	got := build().Shuffle(seed)
	if !dataTablesRowsEqual(got, want) {
		t.Fatalf("Shuffle and Shuffled disagree on a ragged table:\n in place %v %v\n copy     %v %v",
			got.GetColByNumber(0).Data(), got.GetColByNumber(1).Data(), want.GetColByNumber(0).Data(), want.GetColByNumber(1).Data())
	}
}
