package ccl

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"testing"
)

// sequenceStreamCase is one name and the parameter columns the test calls it
// with, each parameter the one-element column the evaluator hands a constant.
type sequenceStreamCase struct {
	name   string
	params [][]any
}

// sequenceStreamCases lists every built-in sequence function with the parameters
// worth pushing at it. The shifts run both ways and through zero, because LAG and
// LEAD read before the row for a positive count and after it for a negative one;
// 40 is longer than every column these seeds build, so the stream runs past its
// end and has to hold every row rather than answer.
func sequenceStreamCases() []sequenceStreamCase {
	var cases []sequenceStreamCase
	for _, name := range []string{"LAG", "LEAD"} {
		for _, p := range []int{-3, -1, 0, 1, 2, 5, 40} {
			cases = append(cases, sequenceStreamCase{name: name, params: [][]any{{float64(p)}}})
		}
	}
	for _, name := range []string{"DIFF", "PCT_CHANGE"} {
		cases = append(cases, sequenceStreamCase{name: name})
		for _, p := range []int{1, 3, 7} {
			cases = append(cases, sequenceStreamCase{name: name, params: [][]any{{float64(p)}}})
		}
	}
	for _, name := range []string{"ROLLING_SUM", "ROLLING_MEAN", "ROLLING_MIN", "ROLLING_MAX", "ROLLING_STD"} {
		for _, w := range []int{1, 2, 3, 10, 40} {
			cases = append(cases, sequenceStreamCase{name: name, params: [][]any{{float64(w)}}})
		}
	}
	for _, name := range []string{"CUMSUM", "CUMPROD", "CUMMAX", "CUMMIN"} {
		cases = append(cases, sequenceStreamCase{name: name})
	}
	return cases
}

// sequenceStreamValue builds one cell of the kind a column can hold: an ordinary
// number of some width, or a value no number comes from.
//
// Both infinities are drawn at about one cell in two hundred, which is what makes
// the cumulative functions reach NaN and stay there: their running value has to
// survive a batch boundary, and a NaN read back as a missing value would restart
// from the initial instead.
func sequenceStreamValue(r *rand.Rand) any {
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

// sequenceStreamColumn builds the column one name is fed. LAG and LEAD move
// values around without reading them as numbers, so a row of a table is one
// value to them; every so often their column holds rows instead of cells, which
// is a shape no arithmetic function could take.
func sequenceStreamColumn(r *rand.Rand, name string, rows int) []any {
	col := make([]any, rows)
	for i := range col {
		col[i] = sequenceStreamValue(r)
	}
	if rows > 0 && rowCapableSequenceFunctions[name] && r.Intn(4) == 0 {
		for i := range col {
			col[i] = []any{sequenceStreamValue(r), sequenceStreamValue(r)}
		}
	}
	return col
}

// sequenceStreamBatches cuts col into the pieces one feed mode pushes. A fixed
// size walks a batch boundary across every row; zero asks for random pieces of
// one to ten values, some of them empty, because an empty piece answers no rows
// and must not be mistaken for the end of the column.
func sequenceStreamBatches(r *rand.Rand, col []any, batch int) [][]any {
	if batch > 0 {
		var batches [][]any
		for start := 0; start < len(col); start += batch {
			end := start + batch
			if end > len(col) {
				end = len(col)
			}
			batches = append(batches, col[start:end])
		}
		return batches
	}
	var batches [][]any
	for start := 0; start < len(col); {
		if r.Intn(11) == 0 {
			batches = append(batches, nil)
		}
		end := start + 1 + r.Intn(10)
		if end > len(col) {
			end = len(col)
		}
		batches = append(batches, col[start:end])
		start = end
	}
	return batches
}

func sequenceStreamBatchLabel(batch int) string {
	if batch <= 0 {
		return "random"
	}
	return fmt.Sprintf("%d", batch)
}

// feedSequenceStream runs the streaming form over col the way a batched reader
// would, and returns every output it produced, in order.
func feedSequenceStream(name string, col []any, params [][]any, batches [][]any) ([]any, error) {
	s, ok, err := NewSequenceStream(name, params)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("NewSequenceStream(%q, %v) reported no streaming form", name, params)
	}
	var out []any
	for _, batch := range batches {
		got, err := s.Push(batch)
		if err != nil {
			return nil, err
		}
		out = append(out, got...)
	}
	tail, err := s.Flush()
	if err != nil {
		return nil, err
	}
	return append(out, tail...), nil
}

// sameSequenceCell reports whether one streaming output is what the whole-column
// call gave in that position: both missing, both float64 down to the last bit
// (a NaN against a NaN counts as the same answer, whichever payload it carries),
// or the same value of any other type, which is what LAG and LEAD hand back.
func sameSequenceCell(want, got any) bool {
	if want == nil || got == nil {
		return want == nil && got == nil
	}
	wf, wok := want.(float64)
	gf, gok := got.(float64)
	if wok || gok {
		if !wok || !gok {
			return false
		}
		if math.IsNaN(wf) && math.IsNaN(gf) {
			return true
		}
		return math.Float64bits(wf) == math.Float64bits(gf)
	}
	return reflect.DeepEqual(want, got)
}

func TestSequenceStreamsMatchTheFunctions(t *testing.T) {
	for _, tc := range sequenceStreamCases() {
		for seed := int64(1); seed <= 20; seed++ {
			r := rand.New(rand.NewSource(seed))
			col := sequenceStreamColumn(r, tc.name, r.Intn(121))
			args := append([][]any{col}, tc.params...)
			want, err := callSequenceFunction(tc.name, args)
			if err != nil {
				t.Fatalf("%s %v seed %d: the function on the whole column failed: %v", tc.name, tc.params, seed, err)
			}

			for _, batch := range []int{1, 2, 7, 1000, 0} {
				label := sequenceStreamBatchLabel(batch)
				// A separate draw, so every feed mode is handed the same column.
				batches := sequenceStreamBatches(rand.New(rand.NewSource(seed+100)), col, batch)
				got, err := feedSequenceStream(tc.name, col, tc.params, batches)
				if err != nil {
					t.Errorf("%s %v seed %d batch %s: streaming failed: %v", tc.name, tc.params, seed, label, err)
					continue
				}
				if len(got) != len(want) {
					t.Errorf("%s %v seed %d batch %s: streaming gave %d outputs, the function gave %d (%d rows)",
						tc.name, tc.params, seed, label, len(got), len(want), len(col))
					continue
				}
				for i := range want {
					if sameSequenceCell(want[i], got[i]) {
						continue
					}
					t.Errorf("%s %v seed %d batch %s row %d: streaming gave %#v, the function gave %#v",
						tc.name, tc.params, seed, label, i, got[i], want[i])
					break
				}
			}
		}
	}
}

// TestSequenceStreamCarriesARunningNaN covers what carrying the last output
// instead of the running state cannot do. A sum that reached NaN through +Inf and
// -Inf answers NaN for every row after it, and a value read back as a missing one
// would start again from the initial; a product through +Inf and 0 does the same.
// One value per Push, so every row of these columns crosses a batch boundary.
func TestSequenceStreamCarriesARunningNaN(t *testing.T) {
	cases := []struct {
		name string
		col  []any
	}{
		{"CUMSUM", []any{math.Inf(1), math.Inf(-1), 1.0, 2.0}},
		{"CUMPROD", []any{math.Inf(1), 0.0, 5.0}},
	}
	for _, tc := range cases {
		want, err := callSequenceFunction(tc.name, [][]any{tc.col})
		if err != nil {
			t.Fatalf("%s on the whole column failed: %v", tc.name, err)
		}
		for i, v := range want {
			if i == 0 {
				// The first value seeds the running state, so it is that value.
				continue
			}
			if f, ok := v.(float64); !ok || !math.IsNaN(f) {
				t.Fatalf("%s on the whole column gave %#v at row %d, want the running value to be NaN there", tc.name, v, i)
			}
		}

		var batches [][]any
		for _, v := range tc.col {
			batches = append(batches, []any{v})
		}
		got, err := feedSequenceStream(tc.name, tc.col, nil, batches)
		if err != nil {
			t.Fatalf("%s streamed one value at a time: %v", tc.name, err)
		}
		if len(got) != len(want) {
			t.Fatalf("%s streamed one value at a time gave %d outputs, want %d", tc.name, len(got), len(want))
		}
		for i := range want {
			if sameSequenceCell(want[i], got[i]) {
				continue
			}
			t.Errorf("%s row %d: streaming gave %#v, the function gave %#v", tc.name, i, got[i], want[i])
		}
	}
}

func TestSequenceStreamHasNoFormForOtherNames(t *testing.T) {
	for _, name := range []string{"MEDIAN", "SUM", "FOO"} {
		s, ok, err := NewSequenceStream(name, nil)
		if err != nil {
			t.Errorf("NewSequenceStream(%q, nil) failed: %v", name, err)
			continue
		}
		if ok {
			t.Errorf("NewSequenceStream(%q, nil) = %#v, want no streaming form", name, s)
		}
	}
	if _, ok, err := NewSequenceStream("lag", [][]any{{float64(1)}}); err != nil || !ok {
		t.Errorf(`NewSequenceStream("lag", [[1]]) = %v, %v, want the LAG form and no error`, ok, err)
	}
}

// TestSequenceStreamHasNoFormForAReRegisteredName touches the process-wide
// sequence registry, so it must not run in parallel with anything.
func TestSequenceStreamHasNoFormForAReRegisteredName(t *testing.T) {
	original, ok := lookupSequenceFunction("CUMSUM")
	if !ok {
		t.Fatal("CUMSUM is not registered")
	}
	t.Cleanup(func() {
		registerSequenceFunction("CUMSUM", original)
	})

	RegisterSequenceFunction("CUMSUM", func(args ...[]any) ([]any, error) {
		return []any{42.0}, nil
	})

	s, ok, err := NewSequenceStream("CUMSUM", nil)
	if err != nil {
		t.Errorf(`NewSequenceStream("CUMSUM", nil) failed: %v`, err)
	}
	if ok {
		t.Errorf(`NewSequenceStream("CUMSUM", nil) = %#v after a caller re-registered it, want no streaming form`, s)
	}
}

func TestSequenceStreamReportsBadParams(t *testing.T) {
	cases := []struct {
		name   string
		params [][]any
	}{
		{"DIFF", [][]any{{float64(0)}}},
		{"ROLLING_MEAN", [][]any{{2.5}}},
		{"LAG", nil},
		{"CUMSUM", [][]any{{float64(1)}}},
	}
	for _, tc := range cases {
		_, wantErr := callSequenceFunction(tc.name, append([][]any{{}}, tc.params...))
		if wantErr == nil {
			t.Fatalf("%s %v: the function itself accepted these params, so there is no message to match", tc.name, tc.params)
		}
		s, ok, err := NewSequenceStream(tc.name, tc.params)
		if !ok {
			t.Errorf("%s %v: no streaming form, want one reporting the refused params", tc.name, tc.params)
			continue
		}
		if err == nil {
			t.Errorf("%s %v: NewSequenceStream gave %#v, want the error %q", tc.name, tc.params, s, wantErr)
			continue
		}
		if s != nil {
			t.Errorf("%s %v: NewSequenceStream gave %#v with an error, want no stream", tc.name, tc.params, s)
		}
		if err.Error() != wantErr.Error() {
			t.Errorf("%s %v: got the error %q, want %q", tc.name, tc.params, err, wantErr)
		}
	}
}

// TestSequenceStreamKeepsOnlyItsWindow feeds a long column a batch at a time and
// checks, after every Push, that the stream holds the values its shift or window
// reaches and no more, and that what it answered is what the function gives on
// the whole column. The bound is twice the shift or window plus a batch, because
// the stream drops the values it no longer needs only once they are over half of
// what it holds.
func TestSequenceStreamKeepsOnlyItsWindow(t *testing.T) {
	const rows, batch = 10000, 100
	cases := []struct {
		name  string
		param float64
	}{
		{"LAG", 50},
		{"ROLLING_SUM", 30},
		{"LEAD", 20},
	}
	for _, tc := range cases {
		r := rand.New(rand.NewSource(7))
		col := make([]any, rows)
		for i := range col {
			col[i] = r.NormFloat64() * 100
		}
		params := [][]any{{tc.param}}
		want, err := callSequenceFunction(tc.name, append([][]any{col}, params...))
		if err != nil {
			t.Fatalf("%s %v: the function on the whole column failed: %v", tc.name, params, err)
		}

		s, ok, err := NewSequenceStream(tc.name, params)
		if err != nil || !ok {
			t.Fatalf("NewSequenceStream(%q, %v) = %v, %v, want a stream and no error", tc.name, params, ok, err)
		}
		w, ok := s.(*windowStream)
		if !ok {
			t.Fatalf("%s %v: the stream is a %T, want a *windowStream", tc.name, params, s)
		}

		limit := 2*(int(tc.param)+batch) + 1
		var got []any
		for start := 0; start < rows; start += batch {
			out, err := s.Push(col[start : start+batch])
			if err != nil {
				t.Fatalf("%s %v: Push at row %d failed: %v", tc.name, params, start, err)
			}
			got = append(got, out...)
			if len(w.vals) > limit {
				t.Fatalf("%s %v: after %d rows the stream holds %d values, want at most %d",
					tc.name, params, start+batch, len(w.vals), limit)
			}
			if w.answered > len(w.vals) {
				t.Fatalf("%s %v: after %d rows the stream has answered %d of the %d values it holds",
					tc.name, params, start+batch, w.answered, len(w.vals))
			}
		}
		tail, err := s.Flush()
		if err != nil {
			t.Fatalf("%s %v: Flush failed: %v", tc.name, params, err)
		}
		got = append(got, tail...)

		if len(got) != len(want) {
			t.Fatalf("%s %v: streaming gave %d outputs, the function gave %d", tc.name, params, len(got), len(want))
		}
		for i := range want {
			if !sameSequenceCell(want[i], got[i]) {
				t.Fatalf("%s %v row %d: streaming gave %#v, the function gave %#v", tc.name, params, i, got[i], want[i])
			}
		}
	}
}

// wideRollingColumn is the column the wide-window benchmarks run over: 60,000
// ordinary values, so a window of 20,000 is a third of it.
func wideRollingColumn() []any {
	r := rand.New(rand.NewSource(1))
	col := make([]any, 60000)
	for i := range col {
		col[i] = r.NormFloat64()
	}
	return col
}

// BenchmarkSequenceStreamWideRollingWindow streams ROLLING_MAX over a window of
// 20,000 a thousand rows at a time. Its cost should sit next to the function's
// on the whole column, which BenchmarkSequenceFunctionWideRollingWindow measures.
func BenchmarkSequenceStreamWideRollingWindow(b *testing.B) {
	const batch = 1000
	col := wideRollingColumn()
	params := [][]any{{float64(20000)}}
	b.ReportAllocs()
	for b.Loop() {
		s, ok, err := NewSequenceStream("ROLLING_MAX", params)
		if err != nil || !ok {
			b.Fatalf("NewSequenceStream = %v, %v, want a stream and no error", ok, err)
		}
		for start := 0; start < len(col); start += batch {
			if _, err := s.Push(col[start : start+batch]); err != nil {
				b.Fatal(err)
			}
		}
		if _, err := s.Flush(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSequenceFunctionWideRollingWindow is the whole-column call the stream
// above is measured against.
func BenchmarkSequenceFunctionWideRollingWindow(b *testing.B) {
	col := wideRollingColumn()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := callSequenceFunction("ROLLING_MAX", [][]any{col, {float64(20000)}}); err != nil {
			b.Fatal(err)
		}
	}
}
