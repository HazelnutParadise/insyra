package stats_test

import (
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// likertScale returns rows × items answers on a 1..5 scale driven by one
// latent trait, so the items correlate the way a real scale's do. negative
// lists the items that run against the trait.
func likertScale(rows, items int, seed int64, negative ...int) [][]float64 {
	r := rand.New(rand.NewSource(seed))
	neg := map[int]bool{}
	for _, i := range negative {
		neg[i] = true
	}
	out := make([][]float64, rows)
	for i := range out {
		trait := r.NormFloat64()
		out[i] = make([]float64, items)
		for j := range out[i] {
			v := trait + 0.8*r.NormFloat64()
			if neg[j] {
				v = -v
			}
			out[i][j] = math.Max(1, math.Min(5, math.Round(3+v)))
		}
	}
	return out
}

func tableOfRows(rows [][]float64, names ...string) *insyra.DataTable {
	cols := make([]*insyra.DataList, len(names))
	for j, name := range names {
		vals := make([]any, len(rows))
		for i := range rows {
			vals[i] = rows[i][j]
		}
		cols[j] = insyra.NewDataList(vals...).SetName(name)
	}
	return insyra.NewDataTable(cols...)
}

func closeTo(a, b float64) bool { return math.Abs(a-b) <= 1e-9*math.Max(1, math.Abs(b)) }

// The statistics agree with R's psych::alpha (psych 2.6.5) on complete data.
func TestCronbachAlphaMatchesPsych(t *testing.T) {
	requireRTools(t)
	cases := []struct {
		name    string
		rows    [][]float64
		reverse []int // 0-based items to reverse
	}{
		{"six items", likertScale(120, 6, 1), nil},
		{"two items", likertScale(80, 2, 2), nil},
		{"a negatively correlated item", likertScale(150, 5, 3, 2), nil},
		{"a reversed item", likertScale(150, 5, 4, 1), []int{1}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			k := len(tt.rows[0])
			names := make([]string, k)
			for j := range names {
				names[j] = string(rune('a' + j))
			}
			var opts []stats.CronbachAlphaOptions
			rev := make([]int, len(tt.reverse))
			if len(tt.reverse) > 0 {
				sel := make([]any, len(tt.reverse))
				for i, j := range tt.reverse {
					sel[i] = j
					rev[i] = j + 1
				}
				opts = append(opts, stats.CronbachAlphaOptions{ReverseScored: sel})
			}
			got, err := stats.CronbachAlpha(tableOfRows(tt.rows, names...), nil, opts...)
			if err != nil {
				t.Fatal(err)
			}
			rb := runRBaseline(t, "cronbach_alpha", map[string]any{"rows": tt.rows, "reverse": rev})
			for _, c := range []struct {
				what      string
				got, want float64
			}{
				{"raw alpha", got.Alpha, baselineFloat(t, rb, "raw_alpha")},
				{"standardized alpha", got.StandardizedAlpha, baselineFloat(t, rb, "std_alpha")},
				{"mean inter-item r", got.MeanInterItemCorrelation, baselineFloat(t, rb, "average_r")},
			} {
				if !closeTo(c.got, c.want) {
					t.Errorf("%s: got %.12f, psych %.12f", c.what, c.got, c.want)
				}
			}
			rDrop := baselineFloatSlice(t, rb, "r_drop")
			alphaDrop := baselineFloatSlice(t, rb, "alpha_drop")
			sds := baselineFloatSlice(t, rb, "item_sd")
			means := baselineFloatSlice(t, rb, "item_mean")
			for i, it := range got.Items {
				if !closeTo(it.ItemRestCorrelation, rDrop[i]) || !closeTo(it.SD, sds[i]) || !closeTo(it.Mean, means[i]) {
					t.Errorf("item %d: r.drop %v / %v, sd %v / %v, mean %v / %v", i, it.ItemRestCorrelation, rDrop[i], it.SD, sds[i], it.Mean, means[i])
				}
				// psych reports a number for a two-item scale where the alpha
				// of the one item left is not defined.
				if k > 2 && !closeTo(it.AlphaIfDeleted, alphaDrop[i]) {
					t.Errorf("item %d: alpha if deleted %v, psych %v", i, it.AlphaIfDeleted, alphaDrop[i])
				}
				if k == 2 && !math.IsNaN(it.AlphaIfDeleted) {
					t.Errorf("item %d of two: alpha if deleted %v, want NaN", i, it.AlphaIfDeleted)
				}
			}
			if got.N != len(tt.rows) || got.NumItems != k || got.RowsDropped != 0 {
				t.Errorf("N %d, items %d, dropped %d", got.N, got.NumItems, got.RowsDropped)
			}
		})
	}
}

func TestCronbachAlphaDropsIncompleteRows(t *testing.T) {
	rows := likertScale(60, 4, 5)
	dt := tableOfRows(rows, "q1", "q2", "q3", "q4")
	full, err := stats.CronbachAlpha(dt, nil)
	if err != nil {
		t.Fatal(err)
	}
	// The same rows plus two with a missing answer give the same answer.
	withGaps := dt.Clone()
	withGaps.AppendRowsFromDataList(insyra.NewDataList(math.NaN(), 2.0, 3.0, 4.0), insyra.NewDataList(1.0, 2.0, math.Inf(1), 4.0))
	got, err := stats.CronbachAlpha(withGaps, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.N != 60 || got.RowsDropped != 2 || got.Alpha != full.Alpha {
		t.Errorf("N %d, dropped %d, alpha %v vs %v", got.N, got.RowsDropped, got.Alpha, full.Alpha)
	}
}

func TestCronbachAlphaChoosesItemsBySelector(t *testing.T) {
	rows := likertScale(90, 5, 6)
	dt := tableOfRows(rows, "q1", "q2", "q3", "q4", "q5")
	byName, err := stats.CronbachAlpha(dt, []any{insyra.Name("q1"), insyra.Name("q3"), insyra.Name("q4")})
	if err != nil {
		t.Fatal(err)
	}
	byLetter, err := stats.CronbachAlpha(dt, []any{"A", "C", "D"})
	if err != nil {
		t.Fatal(err)
	}
	byPosition, err := stats.CronbachAlpha(dt, []any{0, 2, -2})
	if err != nil {
		t.Fatal(err)
	}
	if byName.Alpha != byLetter.Alpha || byName.Alpha != byPosition.Alpha || byName.Items[1].Name != "q3" {
		t.Errorf("alphas %v, %v, %v; second item %q", byName.Alpha, byLetter.Alpha, byPosition.Alpha, byName.Items[1].Name)
	}
	// Reversing an item keeps its own mean and only flips its correlations.
	rev, err := stats.CronbachAlpha(dt, []any{insyra.Name("q1"), insyra.Name("q3"), insyra.Name("q4")},
		stats.CronbachAlphaOptions{ReverseScored: []any{"C"}})
	if err != nil {
		t.Fatal(err)
	}
	if !rev.Items[1].Reversed || rev.Items[1].Mean != byName.Items[1].Mean || rev.Items[0].Reversed {
		t.Errorf("reversed item: %+v, plain %+v", rev.Items[1], byName.Items[1])
	}
	if dt.Err() != nil {
		t.Errorf("the caller's table carries an error: %v", dt.Err())
	}
}

func TestCronbachAlphaRefusals(t *testing.T) {
	rows := likertScale(40, 3, 7)
	dt := tableOfRows(rows, "q1", "q2", "q3")
	constant := dt.Clone()
	constant.AppendCols(insyra.NewDataList(func() []any {
		v := make([]any, 40)
		for i := range v {
			v[i] = 3.0
		}
		return v
	}()...).SetName("same"))
	text := dt.Clone()
	text.UpdateElement(5, "B", "n/a")
	cases := []struct {
		name  string
		call  func() error
		words string
	}{
		{"nil table", func() error { _, err := stats.CronbachAlpha(nil, nil); return err }, "nil"},
		{"one item", func() error { _, err := stats.CronbachAlpha(dt, []any{0}); return err }, "two items"},
		{"an item twice", func() error { _, err := stats.CronbachAlpha(dt, []any{0, "A"}); return err }, "same column"},
		{"no such column", func() error { _, err := stats.CronbachAlpha(dt, []any{0, insyra.Name("nope")}); return err }, "nope"},
		{"reverse not an item", func() error {
			_, err := stats.CronbachAlpha(dt, []any{0, 1}, stats.CronbachAlphaOptions{ReverseScored: []any{2}})
			return err
		}, "not one of the items"},
		{"reverse twice", func() error {
			_, err := stats.CronbachAlpha(dt, nil, stats.CronbachAlphaOptions{ReverseScored: []any{0, "A"}})
			return err
		}, "twice"},
		{"two options", func() error {
			_, err := stats.CronbachAlpha(dt, nil, stats.CronbachAlphaOptions{}, stats.CronbachAlphaOptions{})
			return err
		}, "at most one"},
		{"a constant item", func() error { _, err := stats.CronbachAlpha(constant, nil); return err }, "same"},
		{"a text cell", func() error { _, err := stats.CronbachAlpha(text, nil); return err }, "row index 5 column index 1"},
	}
	for _, tt := range cases {
		err := tt.call()
		if err == nil || !strings.Contains(err.Error(), tt.words) {
			t.Errorf("%s: got %v, want an error mentioning %q", tt.name, err, tt.words)
		}
	}
}

// The review measured items x and 6 - x, whose total never varies, giving an
// alpha of -Inf with a nil error, and three ipsative items one of -1.8e16.
func TestCronbachAlphaRefusesATotalWithNoVariance(t *testing.T) {
	x := []any{1.0, 2.0, 3.0, 4.0, 5.0, 2.0, 4.0}
	y := make([]any, len(x))
	for i, v := range x {
		y[i] = 6 - v.(float64)
	}
	dt := insyra.NewDataTable(insyra.NewDataList(x...).SetName("x"), insyra.NewDataList(y...).SetName("y"))
	if _, err := stats.CronbachAlpha(dt, nil); err == nil || !strings.Contains(err.Error(), "total score") {
		t.Errorf("got %v, want an error saying the total score has no variance", err)
	}
	r := rand.New(rand.NewSource(8))
	a, b, c := make([]any, 50), make([]any, 50), make([]any, 50)
	for i := range a {
		p, q := float64(r.Intn(5)), float64(r.Intn(4))
		a[i], b[i], c[i] = p, q, 9-p-q
	}
	ipsative := insyra.NewDataTable(insyra.NewDataList(a...).SetName("a"), insyra.NewDataList(b...).SetName("b"), insyra.NewDataList(c...).SetName("c"))
	if _, err := stats.CronbachAlpha(ipsative, nil); err == nil || !strings.Contains(err.Error(), "total score") {
		t.Errorf("ipsative items: got %v", err)
	}
}

// Values whose squares overflow gave NaN statistics with a nil error.
func TestCronbachAlphaRefusesANonFiniteResult(t *testing.T) {
	rows := likertScale(30, 3, 9)
	for i := range rows {
		for j := range rows[i] {
			rows[i][j] *= 1e200
		}
	}
	if res, err := stats.CronbachAlpha(tableOfRows(rows, "a", "b", "c"), nil); err == nil {
		t.Errorf("got alpha %v and no error", res.Alpha)
	}
}

// A string that is a number is still a text cell, and the error quotes it so
// it does not read as a number.
func TestCronbachAlphaQuotesATextCell(t *testing.T) {
	dt := insyra.NewDataTable(insyra.NewDataList("1", "2", "3").SetName("a"), insyra.NewDataList(1.0, 2.0, 4.0).SetName("b"))
	if _, err := stats.CronbachAlpha(dt, nil); err == nil || !strings.Contains(err.Error(), `"1"`) {
		t.Errorf("got %v, want the cell quoted", err)
	}
}

// On data with missing answers the rows used are psych's complete cases.
func TestCronbachAlphaMatchesPsychOnCompleteCases(t *testing.T) {
	requireRTools(t)
	rows := likertScale(100, 4, 10)
	dt := tableOfRows(rows, "a", "b", "c", "d")
	dt.UpdateElement(3, "B", math.NaN())
	dt.UpdateElement(40, "D", math.NaN())
	got, err := stats.CronbachAlpha(dt, nil)
	if err != nil {
		t.Fatal(err)
	}
	complete := append(append([][]float64{}, rows[:3]...), rows[4:40]...)
	complete = append(complete, rows[41:]...)
	rb := runRBaseline(t, "cronbach_alpha", map[string]any{"rows": complete, "reverse": []int{}})
	if got.N != 98 || got.RowsDropped != 2 || !closeTo(got.Alpha, baselineFloat(t, rb, "raw_alpha")) {
		t.Errorf("N %d, dropped %d, alpha %v, psych %v", got.N, got.RowsDropped, got.Alpha, baselineFloat(t, rb, "raw_alpha"))
	}
}
