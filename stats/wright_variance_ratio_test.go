package stats

import (
	"math"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// wrightTies is 80 draws from N(0, 0.01) rounded to 0.001 (R, set.seed(11)),
// so it has ties and six zeros; wrightTrend is 60 increments that trend (R,
// set.seed(5)). The statistics below are R vrtest 1.2's Wright on the same
// numbers; testdata/wright_vrtest.R reproduces them.
var wrightTies = []any{
	-0.006, 0, -0.015, -0.014, 0.012, -0.009, 0.013, 0.006,
	0, -0.010, -0.008, -0.003, -0.015, -0.003, -0.011, 0,
	-0.002, 0.009, -0.006, -0.007, -0.007, 0, -0.004, 0.004,
	0.001, 0, -0.002, -0.008, -0.002, -0.010, -0.011, -0.009,
	0.007, -0.016, -0.009, 0.005, -0.002, 0.015, -0.006, -0.003,
	-0.016, 0, 0.009, -0.009, 0.009, -0.003, -0.022, 0.009,
	0.007, 0.002, 0.008, -0.002, -0.008, 0.005, 0.002, 0.005,
	-0.002, 0.004, 0.015, 0.001, -0.008, 0.023, -0.001, -0.020,
	0.005, 0.017, -0.008, -0.011, -0.006, 0.008, 0.005, -0.001,
	-0.008, 0.002, 0.011, 0.002, -0.007, 0.005, -0.011, 0.004,
}

var wrightTrend = []any{
	-1.4091468823605449e-03, 9.7091907425651358e-04, -1.0011732094163082e-03, -3.9366446399297800e-04, 6.8956667388842517e-04, 2.9599452253993748e-04, -1.0566432876661935e-03, -7.9170118063602330e-04,
	-1.5628153959083312e-03, -9.2834850491842126e-04, 3.3706623342986589e-04, -4.7547165357184773e-04, -1.2695429348295105e-03, -1.5844075956484805e-03, -2.4655859119894594e-03, -2.8890119356878934e-03,
	-2.9219080958466058e-03, -5.5320724807929756e-03, -5.2572134505042920e-03, -5.8925982378902498e-03, -4.4614356751433524e-03, -4.6399256104578202e-03, -1.8130479117547281e-03, -1.6060928071668780e-03,
	-4.3720020287946567e-04, -1.4236790917656543e-03, 1.5436811064356848e-03, 1.6121226045552430e-03, 1.1538702378344257e-03, 7.8041047926669029e-05, 1.1178996818084531e-03, 1.2333646963790261e-03,
	4.2585477870131618e-03, 5.6474997852776088e-03, 7.1713291213038894e-03, 7.8197341556674946e-03, 6.4451055590657308e-03, 4.3478301022749268e-03, 2.7644610581917016e-03, 2.6834809038731398e-03,
	3.2654976144412279e-03, 4.0284235572238649e-03, 3.7240862796326171e-03, 5.3860985830318031e-03, 4.7963651895152086e-03, 5.8344170802520513e-03, 4.3860701713015421e-03, 3.0574319606738775e-03,
	2.9391745765148941e-03, 3.8867944558126305e-03, 4.2335147266701640e-03, 5.7088830004790696e-03, 7.2197871753664385e-03, 5.3977055849151589e-03, 4.5403208407543017e-03, 4.6901931004263127e-03,
	4.2665344695252449e-03, 5.1632833479418258e-03, 4.8935471736116292e-03, 4.9995917845105202e-03,
}

func TestWrightVarianceRatioMatchesVrtest(t *testing.T) {
	cases := []struct {
		name   string
		series []any
		qs     []int
		want   [][3]float64 // R1, R2, S1 per q
		zeros  int
	}{
		{"ties and zeros", wrightTies, []int{2, 5, 10}, [][3]float64{
			{-0.36471681231991937, -0.46984584216782749, 0.89442719099991674},
			{-0.74251644914151593, -1.05037693937633181, 0.85732140997411221},
			{-0.39903280389974050, -0.68623603014584733, 1.97355321202691569},
		}, 6},
		{"trend", wrightTrend, []int{2, 4}, [][3]float64{
			{6.8745185363273746, 6.7881493456519326, 5.6803755744375461},
			{10.1236586534983886, 9.8409549103658236, 8.5568129358451941},
		}, 0},
	}
	for _, c := range cases {
		res, err := WrightVarianceRatio(insyra.NewDataList(c.series...), c.qs, WrightVarianceRatioOptions{Replications: 200})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		for k, r := range res {
			got := [3]float64{r.R1, r.R2, r.S1}
			for j := range got {
				if math.Abs(got[j]-c.want[k][j]) > 1e-12*(1+math.Abs(c.want[k][j])) {
					t.Errorf("%s q=%d stat %d = %.17g, vrtest %.17g", c.name, r.Q, j, got[j], c.want[k][j])
				}
			}
			if r.Q != c.qs[k] || r.Zeros != c.zeros {
				t.Errorf("%s: Q=%d Zeros=%d", c.name, r.Q, r.Zeros)
			}
		}
	}
}

func TestWrightVarianceRatioPValues(t *testing.T) {
	// A trending series is rejected at every q, with the smallest p-value a
	// simulation of B draws can give.
	res, err := WrightVarianceRatio(insyra.NewDataList(wrightTrend...), []int{2, 4}, WrightVarianceRatioOptions{Replications: 999})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range res {
		for _, p := range []float64{r.PR1, r.PR2, r.PS1} {
			if p != 1.0/1000 {
				t.Errorf("q=%d p = %v, want 1/1000", r.Q, p)
			}
		}
	}

	// The same seed repeats; another seed gives other draws.
	a, _ := WrightVarianceRatio(insyra.NewDataList(wrightTies...), []int{2, 5})
	b, _ := WrightVarianceRatio(insyra.NewDataList(wrightTies...), []int{2, 5})
	c, _ := WrightVarianceRatio(insyra.NewDataList(wrightTies...), []int{2, 5}, WrightVarianceRatioOptions{Seed: 9})
	for k := range a {
		if a[k] != b[k] {
			t.Errorf("the default seed did not repeat: %+v vs %+v", a[k], b[k])
		}
	}
	if a[0] == c[0] && a[1] == c[1] {
		t.Error("another seed gave the same p-values")
	}
	for _, r := range a {
		for _, p := range []float64{r.PR1, r.PR2, r.PS1} {
			if p <= 0 || p > 1 {
				t.Errorf("p-value %v outside (0, 1]", p)
			}
		}
	}
}

func TestWrightVarianceRatioHasItsNominalSize(t *testing.T) {
	// Under independent returns about 5% of the p-values fall below 0.05.
	rng := rand.New(rand.NewPCG(3, 4))
	const series = 400
	var rejects [3]int
	for s := 0; s < series; s++ {
		x := make([]any, 60)
		for i := range x {
			x[i] = rng.NormFloat64() * math.Exp(rng.NormFloat64()) // heavy tails
		}
		res, err := WrightVarianceRatio(insyra.NewDataList(x...), []int{5}, WrightVarianceRatioOptions{Replications: 199, Seed: uint64(s)})
		if err != nil {
			t.Fatal(err)
		}
		for j, p := range []float64{res[0].PR1, res[0].PR2, res[0].PS1} {
			if p < 0.05 {
				rejects[j]++
			}
		}
	}
	for j, r := range rejects {
		if rate := float64(r) / series; rate < 0.02 || rate > 0.09 {
			t.Errorf("statistic %d rejected %.3f of true nulls at 5%%", j, rate)
		}
	}
}

func TestWrightVarianceRatioRefusesBadInput(t *testing.T) {
	ok := insyra.NewDataList(wrightTies...)
	cases := map[string]func() error{
		"no q":        func() error { _, e := WrightVarianceRatio(ok, nil); return e },
		"q of 1":      func() error { _, e := WrightVarianceRatio(ok, []int{1}); return e },
		"q too large": func() error { _, e := WrightVarianceRatio(ok, []int{80}); return e },
		"negative reps": func() error {
			_, e := WrightVarianceRatio(ok, []int{2}, WrightVarianceRatioOptions{Replications: -1})
			return e
		},
		"two options": func() error {
			_, e := WrightVarianceRatio(ok, []int{2}, WrightVarianceRatioOptions{}, WrightVarianceRatioOptions{})
			return e
		},
		"constant":  func() error { _, e := WrightVarianceRatio(insyra.NewDataList(1, 1, 1, 1, 1), []int{2}); return e },
		"text cell": func() error { _, e := WrightVarianceRatio(insyra.NewDataList(0.1, "x", 0.2, 0.3), []int{2}); return e },
	}
	for name, f := range cases {
		if f() == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestVarianceRatioResultsShowAsATable(t *testing.T) {
	vr, err := VarianceRatio(insyra.NewDataList(varianceRatioSeries...), 2, 5)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(vr.String(), "\n")
	if len(lines) != 4 || lines[0] != "Variance ratio" || !strings.Contains(lines[1], "ZRobust") || !strings.HasPrefix(strings.TrimSpace(lines[3]), "5 ") {
		t.Errorf("table:\n%s", vr.String())
	}
	w, err := WrightVarianceRatio(insyra.NewDataList(wrightTies...), []int{2, 5, 10}, WrightVarianceRatioOptions{Replications: 50})
	if err != nil {
		t.Fatal(err)
	}
	if s := w.String(); !strings.HasPrefix(s, "Wright variance ratio\n") || strings.Count(s, "\n") != 4 || !strings.Contains(s, "Zeros") {
		t.Errorf("table:\n%s", s)
	}
	ac, err := Autocorrelation(insyra.NewDataList(varianceRatioSeries...), 3)
	if err != nil {
		t.Fatal(err)
	}
	if s := ac.String(); !strings.HasPrefix(s, "Autocorrelation\n") || strings.Count(s, "\n") != 4 || !strings.Contains(s, "PRobust") {
		t.Errorf("table:\n%s", s)
	}
	// A slice of results still works where a plain slice is expected.
	var plain []VarianceRatioResult = vr
	if len(plain) != 2 {
		t.Error("VarianceRatioResults is not a slice of results")
	}
}
