package insyra

// These tests pin the conclusion of #222: six pairs of DataList methods give the
// same numbers on purely numeric input over the positions both produce, but they
// are not two names for one function. They differ in result length, in what a
// nil or NaN cell does, in which windows they emit, and in whether a call that
// cannot be done fails or returns a usable answer. Difference, MovingAverage,
// MovingStdev and WeightedMovingAverage are deprecated and keep this behaviour
// until they are removed. The differences are tabulated in Docs/DataList.md under
// "Methods that look alike but differ".
//
// Every scenario below builds its own lists, because Err() is sticky and a list
// that has recorded a failure cannot be reused.

import (
	"math"
	"reflect"
	"testing"
)

// assertLegacyFailed is the "failure" half of the contract shared with
// Docs/DataList.md: the call handed back a usable but empty list, and both it
// and the list it was called on recorded why.
func assertLegacyFailed(t *testing.T, result, source *DataList) {
	t.Helper()
	if result == nil {
		t.Fatalf("failed transform returned nil, want a usable empty list")
	}
	if got := result.Len(); got != 0 {
		t.Errorf("failed transform returned a list of length %d, want 0", got)
	}
	if result.Err() == nil {
		t.Errorf("failed transform returned a list with a nil Err()")
	}
	if source.Err() == nil {
		t.Errorf("source list Err() is nil, want the recorded failure")
	}
}

// assertLegacySucceeded is the "not a failure" half: a usable answer, and no
// error on the result or on the list it came from.
func assertLegacySucceeded(t *testing.T, result, source *DataList) {
	t.Helper()
	if result == nil {
		t.Fatalf("transform returned nil, want a usable list")
	}
	if result.Err() != nil {
		t.Errorf("result Err() = %v, want nil", result.Err())
	}
	if source.Err() != nil {
		t.Errorf("source list Err() = %v, want nil", source.Err())
	}
}

// TestDifferenceAgainstDiff pins Difference() against Diff(1). They agree on
// the numbers over the overlap and disagree about everything else: length,
// a nil operand, a NaN operand, a non-numeric cell, and a list too short to
// difference at all.
func TestDifferenceAgainstDiff(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	t.Run("numeric: same values, one shorter", func(t *testing.T) {
		legacySrc := NewDataList(1, 4, 9, 16)
		legacy := legacySrc.Difference()
		replacementSrc := NewDataList(1, 4, 9, 16)
		replacement := replacementSrc.Diff(1)

		if got, want := legacy.Data(), []any{3.0, 5.0, 7.0}; !reflect.DeepEqual(got, want) {
			t.Errorf("Difference().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		if got, want := replacement.Data(), []any{nil, 3.0, 5.0, 7.0}; !reflect.DeepEqual(got, want) {
			t.Errorf("Diff(1).Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		if got, want := replacement.Data()[1:], legacy.Data(); !reflect.DeepEqual(got, want) {
			t.Errorf("Diff(1).Data()[1:] = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("nil operand: NaN against nil", func(t *testing.T) {
		legacySrc := NewDataList(1.0, nil, 4.0)
		legacy := legacySrc.Difference()
		replacementSrc := NewDataList(1.0, nil, 4.0)
		replacement := replacementSrc.Diff(1)

		sliceEqualApprox(t, legacy.Data(), []any{math.NaN(), math.NaN()}, 0)
		if got, want := replacement.Data(), []any{nil, nil, nil}; !reflect.DeepEqual(got, want) {
			t.Errorf("Diff(1).Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("NaN operand: same NaN over the overlap", func(t *testing.T) {
		legacySrc := NewDataList(1.0, math.NaN(), 4.0)
		legacy := legacySrc.Difference()
		replacementSrc := NewDataList(1.0, math.NaN(), 4.0)
		replacement := replacementSrc.Diff(1)

		sliceEqualApprox(t, legacy.Data(), []any{math.NaN(), math.NaN()}, 0)
		sliceEqualApprox(t, replacement.Data(), []any{nil, math.NaN(), math.NaN()}, 0)
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("non-numeric cell: failure against nil", func(t *testing.T) {
		legacySrc := NewDataList("a", 1, 2)
		legacy := legacySrc.Difference()
		assertLegacyFailed(t, legacy, legacySrc)

		replacementSrc := NewDataList("a", 1, 2)
		replacement := replacementSrc.Diff(1)
		if got, want := replacement.Data(), []any{nil, nil, 1.0}; !reflect.DeepEqual(got, want) {
			t.Errorf("Diff(1).Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("one value: empty against one nil", func(t *testing.T) {
		legacySrc := NewDataList(5.0)
		legacy := legacySrc.Difference()
		if legacy == nil {
			t.Fatalf("Difference() returned nil, want a usable empty list")
		}
		if got := legacy.Len(); got != 0 {
			t.Errorf("Difference().Len() = %d, want 0", got)
		}
		if legacy.Err() != nil {
			t.Errorf("Difference() Err() = %v, want nil: a list too short only warns", legacy.Err())
		}
		if legacySrc.Err() != nil {
			t.Errorf("source list Err() = %v, want nil: a list too short only warns", legacySrc.Err())
		}

		replacementSrc := NewDataList(5.0)
		replacement := replacementSrc.Diff(1)
		if got, want := replacement.Data(), []any{nil}; !reflect.DeepEqual(got, want) {
			t.Errorf("Diff(1).Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, replacement, replacementSrc)
	})
}

// TestFillNaNWithMeanAgainstFillWithMean pins FillNaNWithMean() against
// FillWithMean(). Both fill a gap with the mean of the observed values, but the
// older one fills NaN only and rewrites every number it leaves behind as
// float64, while the newer one fills nil as well and touches nothing else.
func TestFillNaNWithMeanAgainstFillWithMean(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	t.Run("NaN only against NaN and nil", func(t *testing.T) {
		legacy := NewDataList(1, nil, math.NaN(), 3)
		legacy.FillNaNWithMean()
		if got, want := legacy.Data(), []any{1.0, nil, 2.0, 3.0}; !reflect.DeepEqual(got, want) {
			t.Errorf("FillNaNWithMean().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		if legacy.Err() != nil {
			t.Errorf("FillNaNWithMean() Err() = %v, want nil", legacy.Err())
		}

		replacement := NewDataList(1, nil, math.NaN(), 3)
		replacement.FillWithMean()
		if got, want := replacement.Data(), []any{1, 2.0, 2.0, 3}; !reflect.DeepEqual(got, want) {
			t.Errorf("FillWithMean().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		if replacement.Err() != nil {
			t.Errorf("FillWithMean() Err() = %v, want nil", replacement.Err())
		}
	})

	t.Run("no numeric value: both fail and leave the list", func(t *testing.T) {
		legacy := NewDataList(math.NaN(), math.NaN())
		legacy.FillNaNWithMean()
		if legacy.Err() == nil {
			t.Errorf("FillNaNWithMean() Err() is nil, want the recorded failure")
		}
		sliceEqualApprox(t, legacy.Data(), []any{math.NaN(), math.NaN()}, 0)

		replacement := NewDataList(math.NaN(), math.NaN())
		replacement.FillWithMean()
		if replacement.Err() == nil {
			t.Errorf("FillWithMean() Err() is nil, want the recorded failure")
		}
		sliceEqualApprox(t, replacement.Data(), []any{math.NaN(), math.NaN()}, 0)
	})
}

// TestMovingAverageAgainstRollingMean pins MovingAverage(w) against
// Rolling(RollingOptions{Window: w}).Mean(). They agree on the windows both
// emit, and disagree about length, about a nil or non-numeric cell (the older
// one fails the whole call, the newer one leaves the affected windows nil), and
// about a window the list is too short to fill.
func TestMovingAverageAgainstRollingMean(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	t.Run("numeric: same values, window-1 shorter", func(t *testing.T) {
		legacySrc := NewDataList(1.0, 2.0, 3.0, 4.0, 5.0)
		legacy := legacySrc.MovingAverage(3)
		replacementSrc := NewDataList(1.0, 2.0, 3.0, 4.0, 5.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 3}).Mean()

		if got, want := legacy.Data(), []any{2.0, 3.0, 4.0}; !reflect.DeepEqual(got, want) {
			t.Errorf("MovingAverage(3).Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		if got, want := replacement.Data(), []any{nil, nil, 2.0, 3.0, 4.0}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 3).Mean().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		if got, want := replacement.Data()[2:], legacy.Data(); !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 3).Mean().Data()[2:] = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("window of one", func(t *testing.T) {
		legacySrc := NewDataList(1.0, 2.0, 3.0)
		legacy := legacySrc.MovingAverage(1)
		replacementSrc := NewDataList(1.0, 2.0, 3.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 1}).Mean()

		if got, want := legacy.Data(), []any{1.0, 2.0, 3.0}; !reflect.DeepEqual(got, want) {
			t.Errorf("MovingAverage(1).Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		if got, want := replacement.Data(), []any{1.0, 2.0, 3.0}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 1).Mean().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("window equal to the length", func(t *testing.T) {
		legacySrc := NewDataList(1.0, 2.0, 3.0)
		legacy := legacySrc.MovingAverage(3)
		replacementSrc := NewDataList(1.0, 2.0, 3.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 3}).Mean()

		if got, want := legacy.Data(), []any{2.0}; !reflect.DeepEqual(got, want) {
			t.Errorf("MovingAverage(3).Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		if got, want := replacement.Data(), []any{nil, nil, 2.0}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 3).Mean().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("nil cell: failure against nil windows", func(t *testing.T) {
		legacySrc := NewDataList(1.0, nil, 3.0, 4.0, 5.0)
		legacy := legacySrc.MovingAverage(2)
		assertLegacyFailed(t, legacy, legacySrc)

		replacementSrc := NewDataList(1.0, nil, 3.0, 4.0, 5.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 2}).Mean()
		if got, want := replacement.Data(), []any{nil, nil, nil, 3.5, 4.5}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 2).Mean().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("NaN cell: propagates against nil", func(t *testing.T) {
		legacySrc := NewDataList(1.0, math.NaN(), 3.0, 4.0)
		legacy := legacySrc.MovingAverage(2)
		replacementSrc := NewDataList(1.0, math.NaN(), 3.0, 4.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 2}).Mean()

		sliceEqualApprox(t, legacy.Data(), []any{math.NaN(), math.NaN(), 3.5}, 0)
		if got, want := replacement.Data(), []any{nil, nil, nil, 3.5}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 2).Mean().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("non-numeric cell: failure against nil", func(t *testing.T) {
		legacySrc := NewDataList("a", 2.0, 3.0)
		legacy := legacySrc.MovingAverage(2)
		assertLegacyFailed(t, legacy, legacySrc)

		replacementSrc := NewDataList("a", 2.0, 3.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 2}).Mean()
		if got, want := replacement.Data(), []any{nil, nil, 2.5}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 2).Mean().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("window above the length: failure against all nil", func(t *testing.T) {
		legacySrc := NewDataList(1.0, 2.0)
		legacy := legacySrc.MovingAverage(3)
		assertLegacyFailed(t, legacy, legacySrc)

		replacementSrc := NewDataList(1.0, 2.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 3}).Mean()
		if got, want := replacement.Data(), []any{nil, nil}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 3).Mean().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("window of zero: both refuse", func(t *testing.T) {
		legacySrc := NewDataList(1.0, 2.0)
		legacy := legacySrc.MovingAverage(0)
		if legacy == nil {
			t.Errorf("MovingAverage(0) returned nil, want a usable empty list")
		} else if got := legacy.Len(); got != 0 {
			t.Errorf("MovingAverage(0).Len() = %d, want 0", got)
		}
		if legacySrc.Err() == nil {
			t.Errorf("source list Err() is nil, want the recorded failure")
		}

		replacementSrc := NewDataList(1.0, 2.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 0}).Mean()
		if replacement == nil {
			t.Errorf("Rolling(Window: 0).Mean() returned nil, want a usable empty list")
		} else if got := replacement.Len(); got != 0 {
			t.Errorf("Rolling(Window: 0).Mean().Len() = %d, want 0", got)
		}
		if replacementSrc.Err() == nil {
			t.Errorf("source list Err() is nil, want the recorded failure")
		}
	})
}
