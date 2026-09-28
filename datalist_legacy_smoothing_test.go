package insyra

// These tests pin the conclusion of #222 for the smoothing and averaging half
// of the table: six pairs of DataList methods give the same numbers on purely
// numeric input over the positions both produce, but they are not two names for
// one function. They differ in result length, in what a nil or NaN cell does, in
// which windows they emit, and in whether a call that cannot be done fails or
// returns a usable answer. MovingStdev and WeightedMovingAverage are deprecated
// and keep this behaviour until they are removed; ExponentialSmoothing is not.
// The differences are tabulated in Docs/DataList.md under "Methods that look
// alike but differ".
//
// Every scenario below builds its own lists, because Err() is sticky and a list
// that has recorded a failure cannot be reused.

import (
	"math"
	"reflect"
	"testing"
)

// TestMovingStdevAgainstRollingStd pins MovingStdev(w) against
// Rolling(RollingOptions{Window: w}).Std(). They agree on the windows both
// emit, and disagree about length, about which cells count as observed (the
// older one reads a nil as a number it cannot use, the newer one drops the
// window), about a window of one, and about a window the list is too short to
// fill.
func TestMovingStdevAgainstRollingStd(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	t.Run("numeric: same values within 1e-12, window-1 shorter", func(t *testing.T) {
		legacySrc := NewDataList(1.0, 2.0, 4.0, 8.0, 16.0)
		legacy := legacySrc.MovingStdev(3)
		replacementSrc := NewDataList(1.0, 2.0, 4.0, 8.0, 16.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 3}).Std()

		if got, want := legacy.Len(), 3; got != want {
			t.Errorf("MovingStdev(3).Len() = %d, want %d", got, want)
		}
		if got, want := replacement.Len(), 5; got != want {
			t.Errorf("Rolling(Window: 3).Std().Len() = %d, want %d", got, want)
		}
		if got, want := replacement.Data()[:2], []any{nil, nil}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 3).Std().Data()[:2] = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		for i := range 3 {
			ms, okMS := ToFloat64Safe(legacy.Data()[i])
			rs, okRS := ToFloat64Safe(replacement.Data()[i+2])
			if !okMS || !okRS {
				t.Errorf("position %d: MovingStdev = %#v (%T), Rolling Std = %#v (%T), want two numbers",
					i, legacy.Data()[i], legacy.Data()[i], replacement.Data()[i+2], replacement.Data()[i+2])
				continue
			}
			if d := math.Abs(ms - rs); d > 1e-12 {
				t.Errorf("position %d: MovingStdev(3) = %v, Rolling(Window: 3).Std() = %v, difference %v > 1e-12",
					i, ms, rs, d)
			}
		}
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("window of one: NaN against nil", func(t *testing.T) {
		legacySrc := NewDataList(1.0, 2.0, 3.0)
		legacy := legacySrc.MovingStdev(1)
		sliceEqualApprox(t, legacy.Data(), []any{math.NaN(), math.NaN(), math.NaN()}, 0)

		replacementSrc := NewDataList(1.0, 2.0, 3.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 1}).Std()
		if got, want := replacement.Data(), []any{nil, nil, nil}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 1).Std().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("nil cell: skipped against nil windows", func(t *testing.T) {
		legacySrc := NewDataList(1.0, nil, 3.0, 5.0)
		legacy := legacySrc.MovingStdev(2)
		sliceEqualApprox(t, legacy.Data(), []any{math.NaN(), math.NaN(), math.Sqrt(2)}, 1e-12)

		replacementSrc := NewDataList(1.0, nil, 3.0, 5.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 2}).Std()
		if got, want := replacement.Data(), []any{nil, nil, nil, math.Sqrt(2)}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 2).Std().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("NaN cell: propagates against nil", func(t *testing.T) {
		legacySrc := NewDataList(1.0, math.NaN(), 3.0, 5.0)
		legacy := legacySrc.MovingStdev(2)
		sliceEqualApprox(t, legacy.Data(), []any{math.NaN(), math.NaN(), math.Sqrt(2)}, 1e-12)

		replacementSrc := NewDataList(1.0, math.NaN(), 3.0, 5.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 2}).Std()
		if got, want := replacement.Data(), []any{nil, nil, nil, math.Sqrt(2)}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 2).Std().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("non-numeric cell: skipped against nil", func(t *testing.T) {
		legacySrc := NewDataList("a", 1.0, 3.0)
		legacy := legacySrc.MovingStdev(2)
		sliceEqualApprox(t, legacy.Data(), []any{math.NaN(), math.Sqrt(2)}, 1e-12)

		replacementSrc := NewDataList("a", 1.0, 3.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 2}).Std()
		if got, want := replacement.Data(), []any{nil, nil, math.Sqrt(2)}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 2).Std().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("window above the length: failure against all nil", func(t *testing.T) {
		legacySrc := NewDataList(1.0, 2.0)
		legacy := legacySrc.MovingStdev(3)
		assertLegacyFailed(t, legacy, legacySrc)

		replacementSrc := NewDataList(1.0, 2.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 3}).Std()
		if got, want := replacement.Data(), []any{nil, nil}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 3).Std().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, replacement, replacementSrc)
	})
}

// TestExponentialSmoothingAgainstEWMMean pins ExponentialSmoothing(alpha)
// against EWM(EWMOptions{Alpha: alpha}).Mean(). On a fully numeric list they
// are the same recursion and the same length, but they disagree about every
// boundary: which alphas are accepted, and what a gap does.
func TestExponentialSmoothingAgainstEWMMean(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	t.Run("numeric: same values within 1e-12", func(t *testing.T) {
		legacySrc := NewDataList(1.0, 2.0, 3.0, 4.0)
		legacy := legacySrc.ExponentialSmoothing(0.3)
		replacementSrc := NewDataList(1.0, 2.0, 3.0, 4.0)
		replacement := replacementSrc.EWM(EWMOptions{Alpha: 0.3}).Mean()

		if got, want := legacy.Len(), 4; got != want {
			t.Errorf("ExponentialSmoothing(0.3).Len() = %d, want %d", got, want)
		}
		if got, want := replacement.Len(), 4; got != want {
			t.Errorf("EWM(Alpha: 0.3).Mean().Len() = %d, want %d", got, want)
		}
		for i := range 4 {
			es, okES := ToFloat64Safe(legacy.Data()[i])
			ew, okEW := ToFloat64Safe(replacement.Data()[i])
			if !okES || !okEW {
				t.Errorf("position %d: ExponentialSmoothing = %#v (%T), EWM Mean = %#v (%T), want two numbers",
					i, legacy.Data()[i], legacy.Data()[i], replacement.Data()[i], replacement.Data()[i])
				continue
			}
			if d := math.Abs(es - ew); d > 1e-12 {
				t.Errorf("position %d: ExponentialSmoothing(0.3) = %v, EWM(Alpha: 0.3).Mean() = %v, difference %v > 1e-12",
					i, es, ew, d)
			}
		}
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("alpha one half is exact", func(t *testing.T) {
		want := []any{1.0, 1.5, 2.25, 3.125}

		legacySrc := NewDataList(1.0, 2.0, 3.0, 4.0)
		legacy := legacySrc.ExponentialSmoothing(0.5)
		if got := legacy.Data(); !reflect.DeepEqual(got, want) {
			t.Errorf("ExponentialSmoothing(0.5).Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}

		replacementSrc := NewDataList(1.0, 2.0, 3.0, 4.0)
		replacement := replacementSrc.EWM(EWMOptions{Alpha: 0.5}).Mean()
		if got := replacement.Data(); !reflect.DeepEqual(got, want) {
			t.Errorf("EWM(Alpha: 0.5).Mean().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("alpha zero: accepted against refused", func(t *testing.T) {
		legacySrc := NewDataList(1.0, 2.0, 3.0)
		legacy := legacySrc.ExponentialSmoothing(0)
		if got, want := legacy.Data(), []any{1.0, 1.0, 1.0}; !reflect.DeepEqual(got, want) {
			t.Errorf("ExponentialSmoothing(0).Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, legacy, legacySrc)

		replacementSrc := NewDataList(1.0, 2.0, 3.0)
		replacement := replacementSrc.EWM(EWMOptions{Alpha: 0}).Mean()
		assertLegacyFailed(t, replacement, replacementSrc)
	})

	t.Run("alpha one: both return the data", func(t *testing.T) {
		want := []any{1.0, 2.0, 3.0}

		legacySrc := NewDataList(1.0, 2.0, 3.0)
		legacy := legacySrc.ExponentialSmoothing(1)
		if got := legacy.Data(); !reflect.DeepEqual(got, want) {
			t.Errorf("ExponentialSmoothing(1).Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}

		replacementSrc := NewDataList(1.0, 2.0, 3.0)
		replacement := replacementSrc.EWM(EWMOptions{Alpha: 1}).Mean()
		if got := replacement.Data(); !reflect.DeepEqual(got, want) {
			t.Errorf("EWM(Alpha: 1).Mean().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("nil cell: failure against carrying the mean forward", func(t *testing.T) {
		legacySrc := NewDataList(1.0, nil, 3.0)
		legacy := legacySrc.ExponentialSmoothing(0.5)
		assertLegacyFailed(t, legacy, legacySrc)

		replacementSrc := NewDataList(1.0, nil, 3.0)
		replacement := replacementSrc.EWM(EWMOptions{Alpha: 0.5}).Mean()
		sliceEqualApprox(t, replacement.Data(), []any{1.0, 1.0, 7.0 / 3}, 1e-12)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("NaN cell: failure against carrying the mean forward", func(t *testing.T) {
		legacySrc := NewDataList(1.0, math.NaN(), 3.0)
		legacy := legacySrc.ExponentialSmoothing(0.5)
		assertLegacyFailed(t, legacy, legacySrc)

		replacementSrc := NewDataList(1.0, math.NaN(), 3.0)
		replacement := replacementSrc.EWM(EWMOptions{Alpha: 0.5}).Mean()
		sliceEqualApprox(t, replacement.Data(), []any{1.0, 1.0, 7.0 / 3}, 1e-12)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("non-numeric cell: failure against nil then values", func(t *testing.T) {
		legacySrc := NewDataList("a", 1.0)
		legacy := legacySrc.ExponentialSmoothing(0.5)
		assertLegacyFailed(t, legacy, legacySrc)

		replacementSrc := NewDataList("a", 1.0)
		replacement := replacementSrc.EWM(EWMOptions{Alpha: 0.5}).Mean()
		if got, want := replacement.Data(), []any{nil, 1.0}; !reflect.DeepEqual(got, want) {
			t.Errorf("EWM(Alpha: 0.5).Mean().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("empty list", func(t *testing.T) {
		legacySrc := NewDataList()
		legacy := legacySrc.ExponentialSmoothing(0.5)
		if legacy == nil {
			t.Fatalf("ExponentialSmoothing(0.5) on an empty list returned nil, want a usable empty list")
		}
		if got := legacy.Len(); got != 0 {
			t.Errorf("ExponentialSmoothing(0.5).Len() = %d, want 0", got)
		}
		if legacy.Err() != nil {
			t.Errorf("ExponentialSmoothing(0.5) Err() = %v, want nil: an empty list only warns", legacy.Err())
		}
		if legacySrc.Err() != nil {
			t.Errorf("source list Err() = %v, want nil: an empty list only warns", legacySrc.Err())
		}

		replacementSrc := NewDataList()
		replacement := replacementSrc.EWM(EWMOptions{Alpha: 0.5}).Mean()
		if replacement == nil {
			t.Fatalf("EWM(Alpha: 0.5).Mean() on an empty list returned nil, want a usable empty list")
		}
		if got := replacement.Len(); got != 0 {
			t.Errorf("EWM(Alpha: 0.5).Mean().Len() = %d, want 0", got)
		}
		if replacement.Err() != nil {
			t.Errorf("EWM(Alpha: 0.5).Mean() Err() = %v, want nil", replacement.Err())
		}
		if replacementSrc.Err() != nil {
			t.Errorf("source list Err() = %v, want nil", replacementSrc.Err())
		}
	})
}

// TestWeightedMovingAverageAgainstRollingWeightedMean pins
// WeightedMovingAverage(w, weights) against
// Rolling(RollingOptions{Window: w, Weights: weights}).Mean(). They agree on
// the windows both emit, and disagree about length, about a zero weight sum
// (the older one divides by it, the newer one returns nil), about which cells
// count as observed, and about a window the list is too short to fill.
func TestWeightedMovingAverageAgainstRollingWeightedMean(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)

	t.Run("numeric: same values, window-1 shorter", func(t *testing.T) {
		weights := []float64{1, 3}
		legacySrc := NewDataList(1.0, 2.0, 3.0, 4.0)
		legacy := legacySrc.WeightedMovingAverage(2, weights)
		replacementSrc := NewDataList(1.0, 2.0, 3.0, 4.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 2, Weights: weights}).Mean()

		if got, want := legacy.Data(), []any{1.75, 2.75, 3.75}; !reflect.DeepEqual(got, want) {
			t.Errorf("WeightedMovingAverage(2, {1, 3}).Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		if got, want := replacement.Data(), []any{nil, 1.75, 2.75, 3.75}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 2, Weights: {1, 3}).Mean().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		if got, want := replacement.Data()[1:], legacy.Data(); !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 2, Weights: {1, 3}).Mean().Data()[1:] = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("weights summing to zero: -Inf against nil", func(t *testing.T) {
		weights := []float64{1, -1}
		legacySrc := NewDataList(1.0, 2.0, 3.0)
		legacy := legacySrc.WeightedMovingAverage(2, weights)
		if got, want := legacy.Len(), 2; got != want {
			t.Errorf("WeightedMovingAverage(2, {1, -1}).Len() = %d, want %d", got, want)
		}
		for i, cell := range legacy.Data() {
			v, ok := ToFloat64Safe(cell)
			if !ok {
				t.Errorf("WeightedMovingAverage(2, {1, -1}) position %d = %#v (%T), want a number", i, cell, cell)
				continue
			}
			if !math.IsInf(v, -1) {
				t.Errorf("WeightedMovingAverage(2, {1, -1}) position %d = %v, want -Inf", i, v)
			}
		}

		replacementSrc := NewDataList(1.0, 2.0, 3.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 2, Weights: weights}).Mean()
		if got, want := replacement.Data(), []any{nil, nil, nil}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 2, Weights: {1, -1}).Mean().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("nil cell: failure against nil", func(t *testing.T) {
		weights := []float64{1, 1}
		legacySrc := NewDataList(1.0, nil, 3.0, 4.0)
		legacy := legacySrc.WeightedMovingAverage(2, weights)
		assertLegacyFailed(t, legacy, legacySrc)

		replacementSrc := NewDataList(1.0, nil, 3.0, 4.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 2, Weights: weights}).Mean()
		if got, want := replacement.Data(), []any{nil, nil, nil, 3.5}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 2, Weights: {1, 1}).Mean().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("NaN cell: propagates against nil", func(t *testing.T) {
		weights := []float64{1, 1}
		legacySrc := NewDataList(1.0, math.NaN(), 3.0, 4.0)
		legacy := legacySrc.WeightedMovingAverage(2, weights)
		sliceEqualApprox(t, legacy.Data(), []any{math.NaN(), math.NaN(), 3.5}, 0)

		replacementSrc := NewDataList(1.0, math.NaN(), 3.0, 4.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 2, Weights: weights}).Mean()
		if got, want := replacement.Data(), []any{nil, nil, nil, 3.5}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 2, Weights: {1, 1}).Mean().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, legacy, legacySrc)
		assertLegacySucceeded(t, replacement, replacementSrc)
	})

	t.Run("window above the length: failure against all nil", func(t *testing.T) {
		weights := []float64{1, 1, 1}
		legacySrc := NewDataList(1.0, 2.0)
		legacy := legacySrc.WeightedMovingAverage(3, weights)
		assertLegacyFailed(t, legacy, legacySrc)

		replacementSrc := NewDataList(1.0, 2.0)
		replacement := replacementSrc.Rolling(RollingOptions{Window: 3, Weights: weights}).Mean()
		if got, want := replacement.Data(), []any{nil, nil}; !reflect.DeepEqual(got, want) {
			t.Errorf("Rolling(Window: 3, Weights: {1, 1, 1}).Mean().Data() = %#v (%T), want %#v (%T)", got, got, want, want)
		}
		assertLegacySucceeded(t, replacement, replacementSrc)
	})
}
