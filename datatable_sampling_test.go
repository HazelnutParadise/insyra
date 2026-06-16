package insyra

import (
	"testing"
)

func TestDataTable_SimpleRandomSample(t *testing.T) {
	// Create a test DataTable with 5 rows and 3 columns
	dt := NewDataTable()
	dl1 := NewDataList(1, 2, 3, 4, 5)
	dl2 := NewDataList(10, 20, 30, 40, 50)
	dl3 := NewDataList(100, 200, 300, 400, 500)
	dt.AppendCols(dl1, dl2, dl3)

	// Test normal sampling: sampleSize < numRows
	sample := dt.SimpleRandomSample(3)
	rows, cols := sample.Size()
	if rows != 3 || cols != 3 {
		t.Errorf("SimpleRandomSample(3) returned wrong size: got (%d, %d), want (3, 3)", rows, cols)
	}

	// Test boundary: sampleSize == numRows
	sample2 := dt.SimpleRandomSample(5)
	rows2, cols2 := sample2.Size()
	if rows2 != 5 || cols2 != 3 {
		t.Errorf("SimpleRandomSample(5) returned wrong size: got (%d, %d), want (5, 3)", rows2, cols2)
	}

	// Test boundary: sampleSize > numRows
	sample3 := dt.SimpleRandomSample(10)
	rows3, cols3 := sample3.Size()
	if rows3 != 5 || cols3 != 3 {
		t.Errorf("SimpleRandomSample(10) returned wrong size: got (%d, %d), want (5, 3)", rows3, cols3)
	}

	// Test edge case: sampleSize = 0
	sample4 := dt.SimpleRandomSample(0)
	rows4, cols4 := sample4.Size()
	if rows4 != 0 || cols4 != 0 {
		t.Errorf("SimpleRandomSample(0) returned wrong size: got (%d, %d), want (0, 0)", rows4, cols4)
	}

	// Test edge case: sampleSize < 0 (should handle gracefully, return empty)
	sample5 := dt.SimpleRandomSample(-1)
	rows5, cols5 := sample5.Size()
	if rows5 != 0 || cols5 != 0 {
		t.Errorf("SimpleRandomSample(-1) returned wrong size: got (%d, %d), want (0, 0)", rows5, cols5)
	}
}

func TestDataTableSampleSeedReproducibleAndAligned(t *testing.T) {
	dt := samplingTestTable()
	a := dt.Sample(3, false, SamplingOptions{UseSeed: true, Seed: 42})
	b := dt.Sample(3, false, SamplingOptions{UseSeed: true, Seed: 42})
	if !dataTablesRowsEqual(a, b) {
		t.Fatalf("same seed should produce same sample")
	}
	if a.NumRows() != 3 || a.NumCols() != 3 {
		t.Fatalf("sample size = (%d,%d), want (3,3)", a.NumRows(), a.NumCols())
	}
	assertSamplingRowsAligned(t, a)
	assertUniqueFirstColumn(t, a)
	for _, name := range a.RowNames() {
		if name == "" {
			t.Fatalf("sample should preserve row names, got %v", a.RowNames())
		}
	}
}

func TestDataTableSampleWithReplacementCanRepeat(t *testing.T) {
	dt := samplingTestTable()
	got := dt.Sample(8, true, SamplingOptions{UseSeed: true, Seed: 8})
	if got.NumRows() != 8 {
		t.Fatalf("sample rows = %d, want 8", got.NumRows())
	}
	if !hasDuplicateFirstColumn(got) {
		t.Fatalf("sample with replacement should be able to repeat source rows")
	}
	assertSamplingRowsAligned(t, got)
}

func TestDataTableSampleFracAndShuffle(t *testing.T) {
	dt := samplingTestTable()
	frac := dt.SampleFrac(0.1, false, SamplingOptions{UseSeed: true, Seed: 3})
	if frac.NumRows() != 1 {
		t.Fatalf("SampleFrac rows = %d, want 1", frac.NumRows())
	}
	shuffledA := dt.Shuffle(SamplingOptions{UseSeed: true, Seed: 4})
	shuffledB := dt.Shuffle(SamplingOptions{UseSeed: true, Seed: 4})
	if shuffledA.NumRows() != dt.NumRows() {
		t.Fatalf("Shuffle rows = %d, want %d", shuffledA.NumRows(), dt.NumRows())
	}
	if !dataTablesRowsEqual(shuffledA, shuffledB) {
		t.Fatalf("same seed should produce same shuffle")
	}
	assertSamplingRowsAligned(t, shuffledA)
}

func TestDataTableTrainTestSplit(t *testing.T) {
	dt := samplingTestTable()
	train, test := dt.TrainTestSplit(0.5, SamplingOptions{PreserveOrder: true})
	if train.NumRows() != 2 || test.NumRows() != 3 {
		t.Fatalf("split rows = (%d,%d), want (2,3)", train.NumRows(), test.NumRows())
	}
	if !equalStringSlices(train.RowNames(), []string{"r1", "r2"}) {
		t.Fatalf("train row names = %v, want [r1 r2]", train.RowNames())
	}
	if !equalStringSlices(test.RowNames(), []string{"r3", "r4", "r5"}) {
		t.Fatalf("test row names = %v, want [r3 r4 r5]", test.RowNames())
	}
	assertSamplingRowsAligned(t, train)
	assertSamplingRowsAligned(t, test)
}

func TestDataTableTrainTestSplitSeedReproducible(t *testing.T) {
	dt := samplingTestTable()
	trainA, testA := dt.TrainTestSplit(0.8, SamplingOptions{UseSeed: true, Seed: 99})
	trainB, testB := dt.TrainTestSplit(0.8, SamplingOptions{UseSeed: true, Seed: 99})
	if !dataTablesRowsEqual(trainA, trainB) || !dataTablesRowsEqual(testA, testB) {
		t.Fatalf("same seed should produce same split")
	}
	if trainA.NumRows() != 4 || testA.NumRows() != 1 {
		t.Fatalf("split rows = (%d,%d), want (4,1)", trainA.NumRows(), testA.NumRows())
	}
}

func TestDataTableSamplingErrors(t *testing.T) {
	dt := samplingTestTable()
	if got := dt.Sample(6, false); got.NumRows() != 0 {
		t.Fatalf("out-of-range sample should return empty table")
	}
	if dt.Err() == nil {
		t.Fatalf("out-of-range sample should set Err")
	}
	dt.ClearErr()
	if got := dt.SampleFrac(1.5, false); got.NumRows() != 0 {
		t.Fatalf("invalid frac should return empty table")
	}
	if dt.Err() == nil {
		t.Fatalf("invalid frac should set Err")
	}
	dt.ClearErr()
	train, test := dt.TrainTestSplit(0)
	if train.NumRows() != 0 || test.NumRows() != 0 {
		t.Fatalf("invalid split should return empty tables")
	}
	if dt.Err() == nil {
		t.Fatalf("invalid split should set Err")
	}
}

func samplingTestTable() *DataTable {
	dt := NewDataTable()
	dt.AppendCols(
		NewDataList(1, 2, 3, 4, 5).SetName("id"),
		NewDataList(10, 20, 30, 40, 50).SetName("value"),
		NewDataList("a", "b", "c", "d", "e").SetName("label"),
	)
	dt.SetRowNames([]string{"r1", "r2", "r3", "r4", "r5"})
	return dt
}

func assertSamplingRowsAligned(t *testing.T, dt *DataTable) {
	t.Helper()
	for i := range dt.NumRows() {
		id, ok := dt.GetElementByNumberIndex(i, 0).(int)
		if !ok {
			t.Fatalf("row %d id type = %T", i, dt.GetElementByNumberIndex(i, 0))
		}
		value := dt.GetElementByNumberIndex(i, 1)
		if value != id*10 {
			t.Fatalf("row %d misaligned: id=%v value=%v", i, id, value)
		}
	}
}

func assertUniqueFirstColumn(t *testing.T, dt *DataTable) {
	t.Helper()
	seen := map[any]bool{}
	for i := range dt.NumRows() {
		value := dt.GetElementByNumberIndex(i, 0)
		if seen[value] {
			t.Fatalf("first column should not repeat without replacement, got %v", value)
		}
		seen[value] = true
	}
}

func hasDuplicateFirstColumn(dt *DataTable) bool {
	seen := map[any]bool{}
	for i := range dt.NumRows() {
		value := dt.GetElementByNumberIndex(i, 0)
		if seen[value] {
			return true
		}
		seen[value] = true
	}
	return false
}

func dataTablesRowsEqual(a, b *DataTable) bool {
	if a.NumRows() != b.NumRows() || a.NumCols() != b.NumCols() {
		return false
	}
	for r := range a.NumRows() {
		for c := range a.NumCols() {
			if a.GetElementByNumberIndex(r, c) != b.GetElementByNumberIndex(r, c) {
				return false
			}
		}
	}
	return equalStringSlices(a.RowNames(), b.RowNames())
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
