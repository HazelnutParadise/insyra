package insyra

import "testing"

func TestDataListSampleSeedReproducible(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4, 5)
	a := dl.Sample(3, false, SamplingOptions{UseSeed: true, Seed: 42})
	b := dl.Sample(3, false, SamplingOptions{UseSeed: true, Seed: 42})
	if !equalAnySlices(a.Data(), b.Data()) {
		t.Fatalf("same seed should produce same sample: %v vs %v", a.Data(), b.Data())
	}
	if a.Len() != 3 {
		t.Fatalf("sample length = %d, want 3", a.Len())
	}
	if hasDuplicateValues(a.Data()) {
		t.Fatalf("sample without replacement should not duplicate values: %v", a.Data())
	}
}

func TestDataListSampleWithReplacementCanRepeat(t *testing.T) {
	dl := NewDataList("a", "b", "c")
	got := dl.Sample(8, true, SamplingOptions{UseSeed: true, Seed: 7})
	if got.Len() != 8 {
		t.Fatalf("sample length = %d, want 8", got.Len())
	}
	if !hasDuplicateValues(got.Data()) {
		t.Fatalf("sample with replacement should be able to repeat values, got %v", got.Data())
	}
}

func TestDataListSampleFracFloorAtLeastOne(t *testing.T) {
	dl := NewDataList(1, 2, 3)
	got := dl.SampleFrac(0.1, false, SamplingOptions{UseSeed: true, Seed: 1})
	if got.Len() != 1 {
		t.Fatalf("SampleFrac length = %d, want 1", got.Len())
	}
}

func TestDataListShuffleSeedReproducible(t *testing.T) {
	dl := NewDataList(1, 2, 3, 4, 5)
	a := dl.Shuffle(SamplingOptions{UseSeed: true, Seed: 9})
	b := dl.Shuffle(SamplingOptions{UseSeed: true, Seed: 9})
	if !equalAnySlices(a.Data(), b.Data()) {
		t.Fatalf("same seed should produce same shuffle: %v vs %v", a.Data(), b.Data())
	}
	if a.Len() != dl.Len() {
		t.Fatalf("shuffle length = %d, want %d", a.Len(), dl.Len())
	}
}

func TestDataListSamplingErrors(t *testing.T) {
	dl := NewDataList(1, 2)
	if got := dl.Sample(3, false); got.Len() != 0 {
		t.Fatalf("out-of-range sample should return empty list")
	}
	if dl.Err() == nil {
		t.Fatalf("out-of-range sample should set Err")
	}
	dl.ClearErr()
	if got := dl.SampleFrac(0, false); got.Len() != 0 {
		t.Fatalf("invalid frac should return empty list")
	}
	if dl.Err() == nil {
		t.Fatalf("invalid frac should set Err")
	}
	dl.ClearErr()
	empty := NewDataList()
	if got := empty.Shuffle(); got.Len() != 0 {
		t.Fatalf("empty shuffle should return empty list")
	}
	if empty.Err() == nil {
		t.Fatalf("empty shuffle should set Err")
	}
}

func equalAnySlices(a, b []any) bool {
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

func hasDuplicateValues(values []any) bool {
	seen := map[any]bool{}
	for _, value := range values {
		if seen[value] {
			return true
		}
		seen[value] = true
	}
	return false
}
