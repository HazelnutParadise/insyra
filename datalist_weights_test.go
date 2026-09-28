package insyra

import (
	"math"
	"reflect"
	"testing"
)

// The weights of WeightedMean and WeightedMovingAverage are one weight per
// element or per window position, held as a plain []float64, the same type
// RollingOptions.Weights takes. No conversion at the call site any more.
func TestWeightedMethodsTakeFloat64Weights(t *testing.T) {
	want := reflect.TypeOf([]float64(nil))
	dlType := reflect.TypeOf((*DataList)(nil))

	mean, ok := dlType.MethodByName("WeightedMean")
	if !ok {
		t.Fatal("DataList has no method WeightedMean")
	}
	if got := mean.Type.In(1); got != want {
		t.Errorf("WeightedMean weights parameter is %v, want %v", got, want)
	}

	movingAvg, ok := dlType.MethodByName("WeightedMovingAverage")
	if !ok {
		t.Fatal("DataList has no method WeightedMovingAverage")
	}
	if got := movingAvg.Type.In(2); got != want {
		t.Errorf("WeightedMovingAverage weights parameter is %v, want %v", got, want)
	}
}

func TestWeightedMeanWithFloat64Weights(t *testing.T) {
	t.Run("one weight per element", func(t *testing.T) {
		dl := NewDataList(1, 2, 3, 4)
		// (1*1 + 2*2 + 3*3 + 4*4) / 10 = 30 / 10 = 3
		if mean := dl.WeightedMean([]float64{1, 2, 3, 4}); !float64Equal(mean, 3) {
			t.Errorf("WeightedMean = %v, want 3", mean)
		}
		if err := dl.Err(); err != nil {
			t.Errorf("WeightedMean recorded an error: %v", err)
		}
	})

	t.Run("a non-numeric cell is skipped with its weight", func(t *testing.T) {
		restoreConfig(t)
		Config.SetLogLevel(LogLevelFatal)
		dl := NewDataList(1, "x", 3)
		// "x" is skipped together with the weight 100, leaving (1*1 + 3*1) / 2
		if mean := dl.WeightedMean([]float64{1, 100, 1}); !float64Equal(mean, 2) {
			t.Errorf("WeightedMean = %v, want 2", mean)
		}
	})

	t.Run("weights of the wrong length fail", func(t *testing.T) {
		dl := NewDataList(1.0, 2.0, 3.0)
		if mean := dl.WeightedMean([]float64{1, 2}); !math.IsNaN(mean) {
			t.Errorf("WeightedMean = %v, want NaN", mean)
		}
		if dl.Err() == nil {
			t.Error("WeightedMean with the wrong number of weights recorded no error")
		}
	})
}

func TestWeightedMovingAverageWithFloat64Weights(t *testing.T) {
	t.Run("one weight per window position", func(t *testing.T) {
		dl := NewDataList(1.0, 2.0, 3.0)
		// (1*1 + 2*3)/4 = 1.75, (2*1 + 3*3)/4 = 2.75
		got := dl.WeightedMovingAverage(2, []float64{1, 3})
		if got == nil {
			t.Fatal("WeightedMovingAverage returned nil")
		}
		if !reflect.DeepEqual(got.Data(), []any{1.75, 2.75}) {
			t.Errorf("WeightedMovingAverage = %v, want [1.75 2.75]", got.Data())
		}
	})

	t.Run("weights of the wrong length fail", func(t *testing.T) {
		dl := NewDataList(1.0, 2.0, 3.0)
		got := dl.WeightedMovingAverage(2, []float64{1})
		if got == nil {
			t.Fatal("WeightedMovingAverage returned nil")
		}
		if got.Len() != 0 {
			t.Errorf("WeightedMovingAverage = %v, want an empty result", got.Data())
		}
		if got.Err() == nil {
			t.Error("the result recorded no error")
		}
		if dl.Err() == nil {
			t.Error("the receiver recorded no error")
		}
	})
}
