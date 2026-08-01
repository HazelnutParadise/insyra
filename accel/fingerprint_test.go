package accel

import (
	"math"
	"testing"
)

func numericDataset(name string, values []float64) *Dataset {
	ds := &Dataset{
		Name:    name,
		Lineage: "test",
		Rows:    len(values),
		Buffers: []Buffer{{Name: "col", Type: DataTypeFloat64, Values: values, Len: len(values)}},
	}
	assignDatasetFingerprint(ds)
	return ds
}

func stringDataset(name string, values []string) *Dataset {
	ds := &Dataset{
		Name:    name,
		Lineage: "test",
		Rows:    len(values),
		Buffers: []Buffer{{Name: "col", Type: DataTypeString, Values: values, Len: len(values)}},
	}
	assignDatasetFingerprint(ds)
	return ds
}

func TestFingerprintIsStableForIdenticalData(t *testing.T) {
	a := numericDataset("numbers", []float64{1, 2, 3, 4})
	b := numericDataset("numbers", []float64{1, 2, 3, 4})
	if a.Fingerprint == "" {
		t.Fatal("expected a fingerprint")
	}
	if a.Fingerprint != b.Fingerprint {
		t.Fatalf("identical data must fingerprint identically: %q vs %q", a.Fingerprint, b.Fingerprint)
	}
}

func TestFingerprintChangesWithASingleValue(t *testing.T) {
	a := numericDataset("numbers", []float64{1, 2, 3, 4})
	b := numericDataset("numbers", []float64{1, 2, 3, 5})
	if a.Fingerprint == b.Fingerprint {
		t.Fatal("a changed value must change the fingerprint")
	}
}

func TestFingerprintDistinguishesSignedZero(t *testing.T) {
	// %v renders both as "0", so the old text-based fingerprint could not tell
	// these apart. Raw bits can, and stricter is safe: it cannot cause a false
	// cache hit.
	a := numericDataset("numbers", []float64{0})
	b := numericDataset("numbers", []float64{math.Copysign(0, -1)})
	if a.Fingerprint == b.Fingerprint {
		t.Fatal("0.0 and -0.0 must fingerprint differently")
	}
}

func TestFingerprintDistinguishesStringSplits(t *testing.T) {
	a := stringDataset("labels", []string{"ab", "c"})
	b := stringDataset("labels", []string{"a", "bc"})
	if a.Fingerprint == b.Fingerprint {
		t.Fatal(`["ab","c"] and ["a","bc"] share their bytes and must still fingerprint differently`)
	}
}

func TestFingerprintDistinguishesIntAndFloatColumns(t *testing.T) {
	ints := &Dataset{
		Name: "x", Lineage: "test", Rows: 2,
		Buffers: []Buffer{{Name: "col", Type: DataTypeInt64, Values: []int64{1, 2}, Len: 2}},
	}
	assignDatasetFingerprint(ints)
	floats := numericDataset("x", []float64{1, 2})
	if ints.Fingerprint == floats.Fingerprint {
		t.Fatal("columns of different types must fingerprint differently")
	}
}

func TestFingerprintDistinguishesNullPositions(t *testing.T) {
	a := &Dataset{
		Name: "x", Lineage: "test", Rows: 3,
		Buffers: []Buffer{{Name: "col", Type: DataTypeFloat64, Values: []float64{1, 0, 3}, Nulls: []bool{false, true, false}, Len: 3}},
	}
	assignDatasetFingerprint(a)
	b := &Dataset{
		Name: "x", Lineage: "test", Rows: 3,
		Buffers: []Buffer{{Name: "col", Type: DataTypeFloat64, Values: []float64{1, 0, 3}, Nulls: []bool{false, false, false}, Len: 3}},
	}
	assignDatasetFingerprint(b)
	if a.Fingerprint == b.Fingerprint {
		t.Fatal("differing null positions must change the fingerprint")
	}
}

func TestFingerprintHandlesBoolAndAnyColumns(t *testing.T) {
	bools := &Dataset{
		Name: "flags", Lineage: "test", Rows: 3,
		Buffers: []Buffer{{Name: "col", Type: DataTypeBool, Values: []bool{true, false, true}, Len: 3}},
	}
	assignDatasetFingerprint(bools)
	flipped := &Dataset{
		Name: "flags", Lineage: "test", Rows: 3,
		Buffers: []Buffer{{Name: "col", Type: DataTypeBool, Values: []bool{true, true, true}, Len: 3}},
	}
	assignDatasetFingerprint(flipped)
	if bools.Fingerprint == flipped.Fingerprint {
		t.Fatal("differing bool values must change the fingerprint")
	}

	mixed := &Dataset{
		Name: "mixed", Lineage: "test", Rows: 2,
		Buffers: []Buffer{{Name: "col", Type: DataTypeAny, Values: []any{1, "a"}, Len: 2}},
	}
	assignDatasetFingerprint(mixed)
	if mixed.Fingerprint == "" {
		t.Fatal("an untyped column must still produce a fingerprint")
	}
}

func BenchmarkDatasetFingerprint(b *testing.B) {
	const n = 4 << 20
	values := make([]float64, n)
	for i := range values {
		values[i] = float64(i%1000) * 0.5
	}
	dataset := &Dataset{
		Name: "numbers", Lineage: "test", Rows: n,
		Buffers: []Buffer{{Name: "col", Type: DataTypeFloat64, Values: values, Len: n}},
	}
	b.SetBytes(int64(n * 8))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if datasetFingerprint(dataset) == "" {
			b.Fatal("empty fingerprint")
		}
	}
}

// BenchmarkProjectValues isolates the typed-projection loop from the
// fingerprint, so the remaining cost of ProjectDataList is attributable.
func BenchmarkProjectValues(b *testing.B) {
	const n = 4 << 20
	values := make([]any, n)
	for i := range values {
		values[i] = float64(i%1000) * 0.5
	}
	b.SetBytes(int64(n * 8))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := projectValues("col", values); err != nil {
			b.Fatalf("project: %v", err)
		}
	}
}
