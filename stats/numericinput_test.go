package stats

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestNumericValuesRefusesEveryUnreadableForm(t *testing.T) {
	for _, tc := range []struct {
		name string
		bad  any
		want string
	}{
		{"missing", nil, "non-numeric"},
		{"text", "abc", "non-numeric"},
		{"empty string", "", "non-numeric"},
		{"undefined", math.NaN(), "non-finite"},
		{"infinite", math.Inf(1), "non-finite"},
		{"negative infinite", math.Inf(-1), "non-finite"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := numericValues([]any{1.0, 2.0, tc.bad, 4.0}, "predictor 0")
			if err == nil {
				t.Fatalf("value %v was accepted", tc.bad)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not say %q", err, tc.want)
			}
			// The row is what makes the error actionable — a caller has to be
			// able to find the cell.
			if !strings.Contains(err.Error(), "row 3") {
				t.Fatalf("error %q does not name the row", err)
			}
			if !strings.Contains(err.Error(), "predictor 0") {
				t.Fatalf("error %q does not name the series", err)
			}
		})
	}
}

// A list whose cells are not observations — one subject's conditions, say —
// counts something other than rows, and "row 3" would point at the wrong axis
// in a table laid out one subject per row. appendNumericValues says what its
// one-based index counts, and appends rather than allocates a fresh slice so a
// caller looping over many groups can fill one; numericValues keeps saying
// "row", byte for byte.
func TestAppendNumericValuesNamesThePositionAndAppends(t *testing.T) {
	for _, tc := range []struct {
		name  string
		raw   []any
		label string
		pos   string
		want  string
	}{
		{
			"a non-numeric cell",
			[]any{1.0, nil},
			"subject 2",
			"condition",
			"subject 2 contains a non-numeric value at condition 2: <nil>",
		},
		{
			"a non-finite cell",
			[]any{math.NaN()},
			"subject 1",
			"condition",
			"subject 1 contains a non-finite value at condition 1: NaN",
		},
		{
			"row position is the default",
			[]any{1.0, "x"},
			"group 3",
			"row",
			"group 3 contains a non-numeric value at row 2: x",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.pos == "row" {
				_, err = numericValues(tc.raw, tc.label)
			} else {
				label := tc.label
				_, err = appendNumericValues(nil, tc.raw, func() string { return label }, tc.pos)
			}
			if err == nil {
				t.Fatalf("%v was accepted", tc.raw)
			}
			if err.Error() != tc.want {
				t.Fatalf("error %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

// A refused cell yields a nil slice, so a caller cannot mistake the values
// appended before it for a complete group.
func TestAppendNumericValuesReturnsNilOnRefusal(t *testing.T) {
	values, err := appendNumericValues([]float64{7}, []any{1.0, nil}, func() string { return "subject 2" }, "condition")
	if err == nil {
		t.Fatalf("nil was accepted")
	}
	if want := "subject 2 contains a non-numeric value at condition 2: <nil>"; err.Error() != want {
		t.Fatalf("error %q, want %q", err.Error(), want)
	}
	if values != nil {
		t.Fatalf("slice %v came back beside the refusal", values)
	}
}

// The label is built only when a cell is refused. A caller looping over many
// groups formats a name per group; paying for it on every group that is clean
// was the whole cost the append form removed.
func TestAppendNumericValuesBuildsTheLabelOnlyOnRefusal(t *testing.T) {
	calls := 0
	label := func() string {
		calls++
		return "group 1"
	}

	values, err := appendNumericValues([]float64{9}, []any{1.0, int64(2)}, label, "row")
	if err != nil {
		t.Fatalf("clean input refused: %v", err)
	}
	want := []float64{9, 1, 2}
	if len(values) != len(want) {
		t.Fatalf("values %v, want %v", values, want)
	}
	for i := range want {
		if values[i] != want[i] {
			t.Fatalf("value %d = %v, want %v", i, values[i], want[i])
		}
	}
	if calls != 0 {
		t.Fatalf("the label was built %d times for clean input", calls)
	}

	if _, err := appendNumericValues(nil, []any{1.0, "x"}, label, "row"); err == nil {
		t.Fatalf("a text cell was accepted")
	}
	if calls != 1 {
		t.Fatalf("the label was built %d times, want exactly 1", calls)
	}
}

// Numeric Go types convert; a string does not, even one that looks like a
// number. That is the rule `insyra.ToFloat64Safe` already enforced for
// clustering, PCA and KNN — this change brings regression and correlation onto
// the same rule rather than inventing a second one.
func TestNumericValuesAcceptsEveryNumericGoType(t *testing.T) {
	values, err := numericValues([]any{1, int64(2), float32(3), uint8(4)}, "x")
	if err != nil {
		t.Fatalf("unexpected refusal: %v", err)
	}
	want := []float64{1, 2, 3, 4}
	for i := range want {
		if values[i] != want[i] {
			t.Fatalf("value %d = %v, want %v", i, values[i], want[i])
		}
	}
}

// The regression and correlation families used to route every value through a
// conversion with no failure channel, so anything unparseable silently became
// zero. These pin that each entry point now refuses instead.
func TestRegressionFamiliesRefuseUnreadableInput(t *testing.T) {
	clean := func() []any {
		out := make([]any, 40)
		for i := range out {
			out[i] = float64(i%7) + 1
		}
		return out
	}
	target := func() []any {
		out := make([]any, 40)
		for i := range out {
			out[i] = float64(i)*1.5 + 3
		}
		return out
	}

	for _, tc := range []struct {
		name string
		fit  func(x, y *insyra.DataList) error
	}{
		{"LinearRegression", func(x, y *insyra.DataList) error {
			_, err := LinearRegression(y, x)
			return err
		}},
		{"PolynomialRegression", func(x, y *insyra.DataList) error {
			_, err := PolynomialRegression(y, x, 2)
			return err
		}},
		{"ExponentialRegression", func(x, y *insyra.DataList) error {
			_, err := ExponentialRegression(y, x)
			return err
		}},
		{"LogarithmicRegression", func(x, y *insyra.DataList) error {
			_, err := LogarithmicRegression(y, x)
			return err
		}},
		{"PoissonRegression", func(x, y *insyra.DataList) error {
			_, err := PoissonRegression(y, x)
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, where := range []string{"predictor", "target"} {
				for _, bad := range []any{nil, "abc"} {
					xs, ys := clean(), target()
					if where == "predictor" {
						xs[5] = bad
					} else {
						ys[5] = bad
					}
					err := tc.fit(insyra.NewDataList(xs...), insyra.NewDataList(ys...))
					if err == nil {
						t.Fatalf("%s holding %v in the %s was accepted", tc.name, bad, where)
					}
				}
			}
			// The clean case must still work, or the guard is just breaking
			// the family.
			if err := tc.fit(insyra.NewDataList(clean()...), insyra.NewDataList(target()...)); err != nil {
				t.Fatalf("clean input refused: %v", err)
			}
		})
	}
}

// Measured before the fix: one nil among six observations moved Pearson's r
// from 0.9992 to 0.9879, with no error and nothing downstream able to tell.
func TestCorrelationRefusesUnreadableInput(t *testing.T) {
	y := insyra.NewDataList(2.0, 4.1, 5.9, 8.2, 9.8, 12.1)
	clean := []any{1.0, 2.0, 3.0, 4.0, 5.0, 6.0}

	if _, err := Correlation(insyra.NewDataList(clean...), y, PearsonCorrelation); err != nil {
		t.Fatalf("clean input refused: %v", err)
	}
	for _, bad := range []any{nil, "abc"} {
		dirty := append([]any(nil), clean...)
		dirty[2] = bad
		if _, err := Correlation(insyra.NewDataList(dirty...), y, PearsonCorrelation); err == nil {
			t.Fatalf("correlation over a series holding %v was scored rather than refused", bad)
		}
		if _, err := Covariance(insyra.NewDataList(dirty...), y); err == nil {
			t.Fatalf("covariance over a series holding %v was scored rather than refused", bad)
		}
		table := insyra.NewDataTable(
			insyra.NewDataList(dirty...).SetName("x"),
			y.Clone().SetName("y"),
		)
		if _, _, err := CorrelationMatrix(table, PearsonCorrelation); err == nil {
			t.Fatalf("correlation matrix over a column holding %v was scored rather than refused", bad)
		}
	}
}

// The families with a deliberate policy must keep it. Sweeping them into the
// refusal rule would replace one wrong answer with a different wrong answer.
func TestFactorAnalysisStillDeletesIncompleteObservations(t *testing.T) {
	// Six observed variables over two latent factors plus unique noise, so the
	// correlation matrix is well conditioned enough for the extraction to run.
	// A fixed linear congruential sequence keeps it reproducible without a
	// seeded generator.
	const rows = 200
	state := uint64(20260801)
	next := func() float64 {
		state = state*6364136223846793005 + 1442695040888963407
		return float64(state>>11)/float64(uint64(1)<<53)*2 - 1
	}
	loadings := [6][2]float64{{0.9, 0.1}, {0.8, 0.2}, {0.85, 0.0}, {0.1, 0.9}, {0.2, 0.8}, {0.0, 0.85}}
	columns := make([]*insyra.DataList, len(loadings))
	values := make([][]any, len(loadings))
	for j := range values {
		values[j] = make([]any, rows)
	}
	for i := 0; i < rows; i++ {
		f1, f2 := next(), next()
		for j, load := range loadings {
			values[j][i] = load[0]*f1 + load[1]*f2 + 0.35*next()
		}
	}
	for j := range columns {
		columns[j] = insyra.NewDataList(values[j]...).SetName(fmt.Sprintf("v%d", j+1))
	}
	options := FactorAnalysisOptions{Count: FactorCountSpec{Method: FactorCountFixed, FixedK: 2}}

	complete := insyra.NewDataTable(columns...)
	if _, err := FactorAnalysis(complete, options); err != nil {
		t.Fatalf("complete input refused: %v", err)
	}

	withHole := insyra.NewDataTable(columns...)
	withHole.UpdateElement(3, "A", math.NaN())
	if _, err := FactorAnalysis(withHole, options); err != nil {
		t.Fatalf("factor analysis refused an incomplete observation instead of deleting it: %v", err)
	}
}
