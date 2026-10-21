package quant

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"os"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/isr"
)

type romanoWolfCase struct {
	Name           string      `json:"name"`
	Perf           [][]float64 `json:"perf"`
	Benchmark      []float64   `json:"benchmark"`
	BlockSize      int         `json:"block_size"`
	Alpha          float64     `json:"alpha"`
	Positions      [][]int     `json:"positions"`
	Sigma          []float64   `json:"sigma"`
	Statistics     []float64   `json:"statistics"`
	CriticalValues []float64   `json:"critical_values"`
	RejectedAt     []int       `json:"rejected_at"`
}

func relClose(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol*math.Max(1, math.Max(math.Abs(a), math.Abs(b)))
}

// TestRomanoWolfMatchesReference feeds the bootstrap positions stored by the
// Python implementation (testdata/romano_wolf_baseline.py) to the Go core and
// compares the studentized statistics, the long-run standard deviations, the
// critical value of every step and the step at which each column is rejected.
func TestRomanoWolfMatchesReference(t *testing.T) {
	raw, err := os.ReadFile("testdata/romano_wolf_baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []romanoWolfCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	sawSecondStep := false
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			opts := DataSnoopingOptions{Replications: len(c.Positions), BlockSize: 1}
			if c.Benchmark != nil {
				opts.Benchmark = toDL(c.Benchmark...)
			}
			d, _, err := relativePerformance("test", toDT(c.Perf), opts)
			if err != nil {
				t.Fatal(err)
			}
			res, err := romanoWolfF64(d, c.BlockSize, c.Alpha, len(c.Positions), replay(c.Positions))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(res.RejectedAt, c.RejectedAt) {
				t.Errorf("RejectedAt = %v, want %v", res.RejectedAt, c.RejectedAt)
			}
			if len(res.CriticalValues) != len(c.CriticalValues) {
				t.Fatalf("%d critical values, want %d", len(res.CriticalValues), len(c.CriticalValues))
			}
			for s := range c.CriticalValues {
				if !relClose(res.CriticalValues[s], c.CriticalValues[s], 1e-9) {
					t.Errorf("critical value of step %d = %v, want %v", s+1, res.CriticalValues[s], c.CriticalValues[s])
				}
			}
			for j := range c.Statistics {
				if !relClose(res.Statistics[j], c.Statistics[j], 1e-9) {
					t.Errorf("statistic[%d] = %v, want %v", j, res.Statistics[j], c.Statistics[j])
				}
			}
			means := columnMeans(d)
			for j := range c.Sigma {
				col := make([]float64, len(d))
				for i := range d {
					col[i] = d[i][j]
				}
				if got := math.Sqrt(longRunVarianceFFT(col, means[j], c.BlockSize)); !relClose(got, c.Sigma[j], 1e-9) {
					t.Errorf("sigma[%d] = %v, want %v", j, got, c.Sigma[j])
				}
			}
			for _, step := range c.RejectedAt {
				if step >= 2 {
					sawSecondStep = true
				}
			}
		})
	}
	if !sawSecondStep {
		t.Error("no reference case rejects a column after step 1, so the stepdown itself is not exercised")
	}
}

// TestFFTVarianceMatchesTheDirectSum compares the FFT estimate with
// SPATest's direct sum for series of several shapes and block sizes.
func TestFFTVarianceMatchesTheDirectSum(t *testing.T) {
	rng := rand.New(rand.NewPCG(31, 41))
	series := func(n int, phi, scale float64) []float64 {
		x := make([]float64, n)
		prev := 0.0
		for i := range x {
			prev = phi*prev + scale*rng.NormFloat64()
			x[i] = prev + 0.3
		}
		return x
	}
	for _, c := range []struct {
		name  string
		n     int
		phi   float64
		scale float64
	}{
		{"random", 600, 0, 1}, {"strongly autocorrelated", 600, 0.97, 1}, {"short", 40, 0.5, 1},
		{"medium", 250, 0.7, 1}, {"long", 2500, 0.3, 1}, {"almost constant", 300, 0, 1e-7}} {
		for _, bs := range []int{1, 2, 5, 10, 50} {
			x := series(c.n, c.phi, c.scale)
			d := make([][]float64, len(x))
			mean := 0.0
			for i, v := range x {
				d[i] = []float64{v}
				mean += v / float64(len(x))
			}
			want := spaLongRunVariance(d, 0, mean, bs)
			got := longRunVarianceFFT(x, mean, bs)
			if !relClose(got, want, 1e-9) {
				t.Errorf("%s n=%d BlockSize=%d: FFT %v, direct %v", c.name, c.n, bs, got, want)
			}
		}
	}
}

func TestRomanoWolfCriticalValueTies(t *testing.T) {
	maxima := []float64{2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
	if c := stepCriticalValue(maxima, 0.1); c != 2 {
		t.Errorf("critical value of tied maxima = %v, want 2", c)
	}
	// Alpha*B is an integer: the 1st largest of 10 at 0.1.
	m := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if c := stepCriticalValue(m, 0.1); c != 10 {
		t.Errorf("critical value = %v, want the largest, 10", c)
	}
	if c := stepCriticalValue(m, 0.11); c != 9 {
		t.Errorf("critical value = %v, want the 2nd largest, 9", c)
	}
}

func TestRomanoWolfCriticalRankIsExact(t *testing.T) {
	for _, c := range []struct {
		alpha float64
		b     int
		want  int
	}{
		{0.1, 120, 12},             // 0.1*120 is 12.000000000000002 in float64
		{0.1, 10, 1},               // a grid point: one exceedance is p = alpha, not below it
		{0.10000000001, 10, 2},     // just above the grid point: one exceedance has p < alpha
		{0.05, 1000, 50},           // exact in decimal, not in binary
		{0.3, 10, 3},               // 0.3*10 is 3.0000000000000004 in float64
		{0.5, 3, 2},                // 1.5 rounds up
		{1e-12, 10, 1},             // never below the largest
		{0.9999999999999999, 4, 4}, // never past B
	} {
		if got := criticalRank(c.alpha, c.b); got != c.want {
			t.Errorf("criticalRank(%v, %d) = %d, want %d", c.alpha, c.b, got, c.want)
		}
	}
	m := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if c := stepCriticalValue(m, 0.10000000001); c != 9 {
		t.Errorf("critical value at alpha just above 1/10 = %v, want the 2nd largest, 9", c)
	}
}

func rwOpts(alpha float64, reps, block int, seed uint64) RomanoWolfOptions {
	return RomanoWolfOptions{DataSnoopingOptions: DataSnoopingOptions{Replications: reps, BlockSize: block, Seed: seed}, Alpha: alpha}
}

func TestRomanoWolfPlantedStrategies(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	shift := make([]float64, 12)
	shift[3] = 0.006
	shift[8] = 0.0035
	perf := simulatedPerf(rng, 500, shift)
	res, err := RomanoWolf(toDT(perf), rwOpts(0.1, 1000, 10, 5))
	if err != nil {
		t.Fatal(err)
	}
	for j, step := range res.RejectedAt {
		strong := j == 3 || j == 8
		if strong && step == 0 {
			t.Errorf("planted column %d not rejected (statistic %v, critical values %v)", j, res.Statistics[j], res.CriticalValues)
		}
		if !strong && step != 0 {
			t.Errorf("null column %d rejected at step %d", j, step)
		}
	}
	if res.Alpha != 0.1 || res.Replications != 1000 || len(res.Names) != 12 || len(res.Statistics) != 12 {
		t.Errorf("result fields wrong: %+v", res)
	}
	if res.RejectedAt[3] > res.RejectedAt[8] {
		t.Errorf("the stronger column was rejected later (%d) than the weaker (%d)", res.RejectedAt[3], res.RejectedAt[8])
	}
}

func TestRomanoWolfSingleColumnAndNames(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	perf := simulatedPerf(rng, 300, []float64{0.004})
	col := make([]any, len(perf))
	for i := range perf {
		col[i] = perf[i][0]
	}
	tb := insyra.NewDataTable(insyra.NewDataList(col...).SetName("only"))
	res, err := RomanoWolf(tb, rwOpts(0.05, 500, 5, 1))
	if err != nil {
		t.Fatal(err)
	}
	if res.RejectedAt[0] != 1 || len(res.CriticalValues) != 1 || res.Names[0] != "only" {
		t.Errorf("single column result = %+v", res)
	}
}

// TestRomanoWolfFamilywiseErrorUnderTheNull counts the null tables in which
// any column is rejected at Alpha = 0.1. The stepdown controls the chance of
// that at 0.1 in large samples; the bound adds three standard errors of the
// simulation (0.021 at 200 tables) and 0.03 for the finite sample (T = 250).
func TestRomanoWolfFamilywiseErrorUnderTheNull(t *testing.T) {
	rng := rand.New(rand.NewPCG(10, 20))
	const tables, alpha = 200, 0.1
	falseClaims := 0
	for i := range tables {
		perf := simulatedPerf(rng, 250, make([]float64, 6))
		res, err := RomanoWolf(toDT(perf), rwOpts(alpha, 199, 3, uint64(i)))
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range res.RejectedAt {
			if s != 0 {
				falseClaims++
				break
			}
		}
	}
	share := float64(falseClaims) / tables
	t.Logf("%d of %d null tables had a rejection (%.3f) at alpha %v", falseClaims, tables, share, alpha)
	if share > alpha+0.021+0.03 {
		t.Errorf("familywise error %.3f at alpha %v is too large", share, alpha)
	}
}

func TestRomanoWolfStricterAlphaNeverRejectsMore(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 9))
	shift := []float64{0.006, 0.004, 0.003, 0.002, 0.001, 0, 0, 0}
	perf := toDT(simulatedPerf(rng, 300, shift))
	var prev []int
	for _, alpha := range []float64{0.5, 0.3, 0.2, 0.1, 0.05, 0.01, 0.001} {
		res, err := RomanoWolf(perf, rwOpts(alpha, 400, 5, 3))
		if err != nil {
			t.Fatal(err)
		}
		for j := range prev {
			if prev[j] == 0 && res.RejectedAt[j] != 0 {
				t.Errorf("alpha %v rejects column %d that a looser level did not", alpha, j)
			}
		}
		prev = res.RejectedAt
	}
}

func TestRomanoWolfReproducibleAndSameAsCoreForWrappers(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 6))
	perfRows := simulatedPerf(rng, 150, []float64{0, 0.005, 0})
	bench := make([]float64, 150)
	for i := range bench {
		bench[i] = 0.0002 * float64(i%3)
	}
	opts := rwOpts(0.1, 200, 4, 11)
	opts.Benchmark = toDL(bench...)
	a, err := RomanoWolf(toDT(perfRows), opts)
	if err != nil {
		t.Fatal(err)
	}
	b, err := RomanoWolf(toDT(perfRows), opts)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("same input, different results: %+v vs %+v", a, b)
	}
	core := toDT(perfRows).(*insyra.DataTable)
	wrapped := rwOpts(0.1, 200, 4, 11)
	wrapped.Benchmark = isr.UseDL(toDL(bench...).(*insyra.DataList))
	c, err := RomanoWolf(isr.UseDT(core), wrapped)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, c) {
		t.Errorf("isr wrappers gave %+v, want %+v", c, a)
	}
}

func TestRomanoWolfRejectsBadInput(t *testing.T) {
	good := toDT(simulatedPerf(rand.New(rand.NewPCG(1, 1)), 40, make([]float64, 3)))
	ok := rwOpts(0.1, 10, 2, 0)
	with := func(f func(*RomanoWolfOptions)) RomanoWolfOptions {
		o := ok
		f(&o)
		return o
	}
	cases := []struct {
		name string
		perf insyra.IDataTable
		opts RomanoWolfOptions
		want string
	}{
		{"alpha zero", good, with(func(o *RomanoWolfOptions) { o.Alpha = 0 }), "Alpha"},
		{"alpha one", good, with(func(o *RomanoWolfOptions) { o.Alpha = 1 }), "Alpha"},
		{"alpha NaN", good, with(func(o *RomanoWolfOptions) { o.Alpha = math.NaN() }), "Alpha"},
		{"alpha negative", good, with(func(o *RomanoWolfOptions) { o.Alpha = -0.1 }), "Alpha"},
		{"no replications", good, with(func(o *RomanoWolfOptions) { o.Replications = 0 }), "Replications"},
		{"block too long", good, with(func(o *RomanoWolfOptions) { o.BlockSize = 21 }), "at most half"},
		{"block zero", good, with(func(o *RomanoWolfOptions) { o.BlockSize = 0 }), "BlockSize"},
		{"nil table", nil, ok, "perf is nil"},
		{"unknown scheme", good, with(func(o *RomanoWolfOptions) { o.Scheme = ResamplingScheme(5) }), "ResamplingScheme(5)"},
		{"short benchmark", good, with(func(o *RomanoWolfOptions) { o.Benchmark = toDL(1, 2) }), "Benchmark has 2 values"},
		{"nil list benchmark", good, with(func(o *RomanoWolfOptions) { o.Benchmark = (*insyra.DataList)(nil) }), "Benchmark is nil"},
		{"overflow", toDT([][]float64{{1e308}, {1e308}, {1e308}, {-1e308}}), ok, "not finite"},
		{"constant column", toDT([][]float64{{0.1}, {0.1}, {0.1}, {0.1}}), ok, "no variance"},
	}
	for _, c := range cases {
		res, err := RomanoWolf(c.perf, c.opts)
		wantErrContains(t, err, c.want)
		if res != nil {
			t.Errorf("%s: got a result with an error", c.name)
		}
	}

	same := insyra.NewDataTable(insyra.NewDataList(0.01, 0.02, 0.03, 0.04), insyra.NewDataList(0.03, 0.01, 0.0, 0.02))
	o := ok
	o.Benchmark = toDL(0.03, 0.01, 0.0, 0.02)
	_, err := RomanoWolf(same, o)
	wantErrContains(t, err, "column 1 has no variance relative to the benchmark")
}

func TestRomanoWolfRefusesOverflowingResampledMean(t *testing.T) {
	// Original mean is finite (the large values cancel) but a draw that
	// repeats same-sign rows overflows.
	rows := [][]float64{{1.7e308}, {-1.7e308}, {1.7e308}, {-1.7e308}, {1.7e308}, {-1.7e308}}
	opts := rwOpts(0.1, 200, 1, 1)
	_, err := RomanoWolf(toDT(rows), opts)
	wantErrContains(t, err, "not finite")
}

func BenchmarkRomanoWolfLarge(b *testing.B) {
	rng := rand.New(rand.NewPCG(1, 2))
	shift := make([]float64, 100)
	shift[0] = 0.002
	perf := toDT(simulatedPerf(rng, 2500, shift))
	opts := rwOpts(0.1, 5000, 10, 1)
	for b.Loop() {
		if _, err := RomanoWolf(perf, opts); err != nil {
			b.Fatal(err)
		}
	}
}
