package insyra

import "testing"

// D-3: a transform that computes a new list never returns nil, so a failed
// step cannot turn the next one into a nil dereference. (Lookups such as
// GetColByName still return nil when the column is absent — that is the
// documented "not found" answer, and they record an error so the caller has a
// signal; changing them to a non-nil value is tracked separately as D-6.)
func TestChainableTransformsNeverReturnNil(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	cases := []struct {
		name string
		call func(dl *DataList) *DataList
	}{
		{"Normalize", func(dl *DataList) *DataList { return dl.Clone().Append("x").Normalize() }},
		{"MovingAverage", func(dl *DataList) *DataList { return dl.Clone().MovingAverage(0) }},
		{"WeightedMovingAverage", func(dl *DataList) *DataList { return dl.Clone().WeightedMovingAverage(2, []float64{1}) }},
		{"ExponentialSmoothing", func(dl *DataList) *DataList { return dl.Clone().ExponentialSmoothing(5) }},
		{"DoubleExponentialSmoothing", func(dl *DataList) *DataList { return dl.Clone().DoubleExponentialSmoothing(5, 0.5) }},
		{"MovingStdev", func(dl *DataList) *DataList { return dl.Clone().MovingStdev(99) }},
		{"Difference", func(dl *DataList) *DataList { return dl.Clone().Append("x").Difference() }},
		{"Rank", func(dl *DataList) *DataList { return dl.Clone().Append("x").Rank() }},
		{"Diff", func(dl *DataList) *DataList { return dl.Clone().Diff(0) }},
		{"PctChange", func(dl *DataList) *DataList { return dl.Clone().PctChange(-1) }},
	}
	base := NewDataList(1.0, 2.0, 3.0)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.call(base)
			if got == nil {
				t.Fatalf("%s returned nil on failure", c.name)
			}
			// The result must be usable: chaining off it must not panic.
			if n := got.Sort().Len(); n != 0 {
				t.Fatalf("%s returned %d items on failure, want an empty list", c.name, n)
			}
			if got.Err() == nil {
				t.Fatalf("%s returned a result with no error recorded", c.name)
			}
		})
	}
}

// The receiver also learns about the failure, so a chain can be checked at
// either end.
func TestFailedTransformRecordsOnReceiver(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	dl := NewDataList(1.0, 2.0, 3.0)
	dl.MovingAverage(0)
	if dl.Err() == nil {
		t.Fatal("receiver did not record the failure")
	}
}

// The whole point of the rule: this chain used to panic.
func TestFailedTransformChainDoesNotPanic(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("chaining off a failed transform panicked: %v", r)
		}
	}()
	n := NewDataList(1.0, 2.0, 3.0).MovingAverage(0).Sort().Reverse().Len()
	if n != 0 {
		t.Fatalf("chain produced %d items, want 0", n)
	}
}
