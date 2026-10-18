package stats

import (
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/isr"
)

// Expected values come from R 4.x p.adjust on the same input
// (Rscript -e 'p.adjust(p, "holm")' and so on, printed with %.15g).
func TestAdjustPValuesMatchesR(t *testing.T) {
	p := []any{0.01, 0.04, 0.03, 0.005, 0.5, 0.04, 0.2, 0.9}
	withNaN := []any{0.02, math.NaN(), 0.001, 0.3, math.NaN(), 0.04}
	nan := math.NaN()
	cases := []struct {
		name   string
		input  []any
		method PAdjustMethod
		want   []float64
	}{
		{"holm", p, PAdjustHolm, []float64{0.07, 0.2, 0.18, 0.04, 1, 0.2, 0.6, 1}},
		{"bonferroni", p, PAdjustBonferroni, []float64{0.08, 0.32, 0.24, 0.04, 1, 0.32, 1, 1}},
		{"hochberg", p, PAdjustHochberg, []float64{0.07, 0.16, 0.16, 0.04, 0.9, 0.16, 0.6, 0.9}},
		{"bh", p, PAdjustBH, []float64{0.04, 0.064, 0.064, 0.04, 0.571428571428571, 0.064, 0.266666666666667, 0.9}},
		{"by", p, PAdjustBY, []float64{0.108714285714286, 0.173942857142857, 0.173942857142857, 0.108714285714286, 1, 0.173942857142857, 0.724761904761905, 1}},
		{"holm NaN", withNaN, PAdjustHolm, []float64{0.06, nan, 0.004, 0.3, nan, 0.08}},
		{"bonferroni NaN", withNaN, PAdjustBonferroni, []float64{0.08, nan, 0.004, 1, nan, 0.16}},
		{"hochberg NaN", withNaN, PAdjustHochberg, []float64{0.06, nan, 0.004, 0.3, nan, 0.08}},
		{"bh NaN", withNaN, PAdjustBH, []float64{0.04, nan, 0.004, 0.3, nan, 0.0533333333333333}},
		{"by NaN", withNaN, PAdjustBY, []float64{0.0833333333333333, nan, 0.00833333333333333, 0.625, nan, 0.111111111111111}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := AdjustPValues(insyra.NewDataList(c.input...), c.method)
			if err != nil {
				t.Fatal(err)
			}
			data := got.Data()
			if len(data) != len(c.want) {
				t.Fatalf("got %d values, want %d", len(data), len(c.want))
			}
			for i, w := range c.want {
				g := data[i].(float64)
				if math.IsNaN(w) != math.IsNaN(g) || (!math.IsNaN(w) && math.Abs(g-w) > 1e-12) {
					t.Errorf("row index %d: got %v, want %v", i, g, w)
				}
			}
		})
	}
}

func TestAdjustPValuesRefusesBadInput(t *testing.T) {
	cases := map[string]insyra.IDataList{
		"nil":        isr.DL,
		"empty":      insyra.NewDataList(),
		"above one":  insyra.NewDataList(0.1, 1.5),
		"negative":   insyra.NewDataList(0.1, -0.2),
		"infinite":   insyra.NewDataList(0.1, math.Inf(1)),
		"not number": insyra.NewDataList(0.1, "x"),
	}
	for name, list := range cases {
		if _, err := AdjustPValues(list, PAdjustHolm); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if _, err := AdjustPValues(insyra.NewDataList(0.1), PAdjustMethod(99)); err == nil || !strings.Contains(err.Error(), "PAdjustMethod(99)") {
		t.Errorf("unknown method: got %v", err)
	}
}

func TestAdjustPValuesAllNaNStaysNaN(t *testing.T) {
	got, err := AdjustPValues(insyra.NewDataList(math.NaN(), math.NaN()), PAdjustBH)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range got.Data() {
		if !math.IsNaN(v.(float64)) {
			t.Fatalf("got %v", v)
		}
	}
}
