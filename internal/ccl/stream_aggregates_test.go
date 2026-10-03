package ccl

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

var streamingAggregateNames = []string{
	"SUM", "AVG", "COUNT", "MAX", "MIN", "VAR", "VARP", "STDEV", "STDEVP",
}

var streamingAggregateBatchSizes = []int{1, 2, 7, 1000}

const maxStreamingAggregateNestDepth = 3

// streamingAggregateValue builds one cell of the kind a column can hold: an
// ordinary number, a value no number can come from, or occasionally a nested
// []any, which forEachValue walks just as it walks an aggregate's arguments.
//
// The draw is weighted, and the weights are what make this test worth running.
// Both infinities in one column sum to NaN, and a column holding both leaves
// SUM, AVG and the variance family answering NaN, which the comparison treats
// as equal whatever its bits. Giving each sign a twelfth of the cells put both
// in nearly every column, and 29 of the 30 seeds compared NaN against NaN —
// the test then passed while the implementation was wrong. At one cell in four
// hundred each sign still turns up across the 1298 cells these 30 seeds build,
// but only 6 seeds go non-finite, which leaves the bit-for-bit comparison
// something to actually compare. Measured with the data this generates.
func streamingAggregateValue(r *rand.Rand, depth int) any {
	if depth < maxStreamingAggregateNestDepth && r.Intn(100) < 6 {
		nested := make([]any, 2+r.Intn(2))
		for i := range nested {
			nested[i] = streamingAggregateValue(r, depth+1)
		}
		return nested
	}
	switch draw := r.Intn(400); {
	case draw < 150: // ordinary, from tenths to a million, positive and negative
		return r.NormFloat64() * math.Pow(10, float64(r.Intn(7)))
	case draw < 300: // ordinary, and large
		return -r.Float64() * 1e18
	case draw < 340:
		return r.Intn(2001) - 1000
	case draw < 375:
		return int64(r.Int63n(1 << 40))
	case draw < 385:
		return nil
	case draw < 392:
		return math.NaN()
	case draw == 392:
		return math.Inf(1)
	case draw == 393:
		return math.Inf(-1)
	case draw < 398:
		return "x"
	case draw == 398:
		return "3.5"
	default:
		return true
	}
}

// streamingAggregateArgs builds one to three argument columns of zero to sixty
// values each, so that a call sees the shapes the aggregate functions do: an
// empty column beside a full one, a column of nothing but nil, and so on.
//
// One column in four is kept short, because otherwise every column is long
// enough to hold two numeric values and the variance family never has to refuse
// anything, which leaves its four refusal messages untested.
func streamingAggregateArgs(r *rand.Rand) [][]any {
	longest := 60
	if r.Intn(4) == 0 {
		longest = 2
	}
	args := make([][]any, 1+r.Intn(3))
	for i := range args {
		col := make([]any, r.Intn(longest+1))
		for j := range col {
			col[j] = streamingAggregateValue(r, 0)
		}
		args[i] = col
	}
	return args
}

// feedStreamingAggregate runs the streaming form over args the way a batched
// reader would: every column in order, cut into pieces of batch values.
func feedStreamingAggregate(name string, args [][]any, batch int) (any, error) {
	s, ok := NewStreamingAggregate(name)
	if !ok {
		return nil, fmt.Errorf("NewStreamingAggregate(%q) reported no streaming form", name)
	}
	for p := 0; p < s.Passes(); p++ {
		s.BeginPass(p)
		for _, col := range args {
			for start := 0; start < len(col); start += batch {
				end := start + batch
				if end > len(col) {
					end = len(col)
				}
				s.Add(col[start:end])
			}
		}
	}
	return s.Result()
}

// sameStreamingResult reports whether the streaming form answered exactly what
// the aggregate function answered: the same error message, the same int64, or
// the same value down to the last bit of its float64.
func sameStreamingResult(want any, wantErr error, got any, gotErr error) bool {
	if (wantErr == nil) != (gotErr == nil) {
		return false
	}
	if wantErr != nil {
		return wantErr.Error() == gotErr.Error()
	}
	if want == nil || got == nil {
		return want == nil && got == nil
	}
	if wi, ok := want.(int64); ok {
		gi, ok := got.(int64)
		return ok && wi == gi
	}
	wf, wok := want.(float64)
	gf, gok := got.(float64)
	if !wok || !gok {
		return false
	}
	if math.IsNaN(wf) && math.IsNaN(gf) {
		return true
	}
	return math.Float64bits(wf) == math.Float64bits(gf)
}

func TestStreamingAggregatesMatchTheAggregateFunctions(t *testing.T) {
	for _, name := range streamingAggregateNames {
		for seed := int64(1); seed <= 30; seed++ {
			args := streamingAggregateArgs(rand.New(rand.NewSource(seed)))
			want, wantErr := callAggregateFunction(name, args)

			for _, batch := range streamingAggregateBatchSizes {
				got, gotErr := feedStreamingAggregate(name, args, batch)
				if sameStreamingResult(want, wantErr, got, gotErr) {
					continue
				}
				t.Errorf("%s seed %d batch %d: streaming gave (%#v, %v), aggregate function gave (%#v, %v)",
					name, seed, batch, got, gotErr, want, wantErr)
			}
		}
	}
}

func TestStreamingAggregateHasNoFormForOtherNames(t *testing.T) {
	for _, name := range []string{"MEDIAN", "FOO", "LAG"} {
		if s, ok := NewStreamingAggregate(name); ok {
			t.Errorf("NewStreamingAggregate(%q) = %#v, want no streaming form", name, s)
		}
	}
	if _, ok := NewStreamingAggregate("sum"); !ok {
		t.Error(`NewStreamingAggregate("sum") reported no streaming form, want the SUM form`)
	}
}

// TestStreamingAggregateHasNoFormForAReRegisteredName touches the process-wide
// aggregate registry, so it must not run in parallel with anything.
func TestStreamingAggregateHasNoFormForAReRegisteredName(t *testing.T) {
	original, ok := lookupAggregateFunction("SUM")
	if !ok {
		t.Fatal("SUM is not registered")
	}
	t.Cleanup(func() {
		registerAggregateFunction("SUM", original)
	})

	RegisterAggregateFunction("SUM", func(args ...[]any) (any, error) {
		return 42.0, nil
	})

	if s, ok := NewStreamingAggregate("SUM"); ok {
		t.Errorf("NewStreamingAggregate(\"SUM\") = %#v after a caller re-registered it, want no streaming form", s)
	}
}
