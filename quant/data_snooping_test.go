package quant

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"os"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

type dataSnoopingCase struct {
	Name         string      `json:"name"`
	Perf         [][]float64 `json:"perf"`
	Benchmark    []float64   `json:"benchmark"`
	BlockSize    int         `json:"block_size"`
	Positions    [][]int     `json:"positions"`
	DBar         []float64   `json:"dbar"`
	Omega        []float64   `json:"omega"`
	SPAStatistic float64     `json:"spa_statistic"`
	SPABest      int         `json:"spa_best"`
	RCStatistic  float64     `json:"rc_statistic"`
	RCBest       int         `json:"rc_best"`
	PRC          float64     `json:"p_rc"`
	PLower       float64     `json:"p_lower"`
	PConsistent  float64     `json:"p_consistent"`
	PUpper       float64     `json:"p_upper"`
}

func closeRel(a, b float64) bool {
	return math.Abs(a-b) <= 1e-12*math.Max(1, math.Max(math.Abs(a), math.Abs(b)))
}

// replay hands out the stored bootstrap positions in order.
func replay(positions [][]int) func() []int {
	i := 0
	return func() []int {
		p := positions[i]
		i++
		return p
	}
}

// TestDataSnoopingMatchesReference compares the deterministic parts and,
// given the same bootstrap positions, the p-values with an independent
// Python implementation of the paper's formulas
// (testdata/data_snooping_baseline.py).
func TestDataSnoopingMatchesReference(t *testing.T) {
	raw, err := os.ReadFile("testdata/data_snooping_baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []dataSnoopingCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			opts := DataSnoopingOptions{Replications: len(c.Positions), BlockSize: c.BlockSize}
			if c.Benchmark != nil {
				opts.Benchmark = toDL(c.Benchmark...)
			}
			d, _, err := relativePerformance("test", toDT(c.Perf), opts)
			if err != nil {
				t.Fatal(err)
			}
			means := columnMeans(d)
			for j := range means {
				if !closeRel(means[j], c.DBar[j]) {
					t.Errorf("dbar[%d] = %v, want %v", j, means[j], c.DBar[j])
				}
				omega := math.Sqrt(spaLongRunVariance(d, j, means[j], c.BlockSize))
				if !closeRel(omega, c.Omega[j]) {
					t.Errorf("omega[%d] = %v, want %v", j, omega, c.Omega[j])
				}
			}

			rc := realityCheckF64(d, len(c.Positions), replay(c.Positions))
			if !closeRel(rc.Statistic, c.RCStatistic) || rc.Best != c.RCBest || rc.PValue != c.PRC {
				t.Errorf("RealityCheck = %+v, want statistic %v, best %d, p %v", rc, c.RCStatistic, c.RCBest, c.PRC)
			}
			spa, err := spaF64(d, c.BlockSize, len(c.Positions), replay(c.Positions))
			if err != nil {
				t.Fatal(err)
			}
			if !closeRel(spa.Statistic, c.SPAStatistic) || spa.Best != c.SPABest {
				t.Errorf("SPA statistic %v best %d, want %v best %d", spa.Statistic, spa.Best, c.SPAStatistic, c.SPABest)
			}
			if spa.PValueLower != c.PLower || spa.PValueConsistent != c.PConsistent || spa.PValueUpper != c.PUpper {
				t.Errorf("SPA p-values %v/%v/%v, want %v/%v/%v", spa.PValueLower, spa.PValueConsistent, spa.PValueUpper, c.PLower, c.PConsistent, c.PUpper)
			}

			// The public functions report the same statistics.
			pubRC, err := RealityCheck(toDT(c.Perf), opts)
			if err != nil {
				t.Fatal(err)
			}
			pubSPA, err := SPATest(toDT(c.Perf), opts)
			if err != nil {
				t.Fatal(err)
			}
			if pubRC.Statistic != rc.Statistic || pubSPA.Statistic != spa.Statistic {
				t.Errorf("public statistics %v/%v differ from the core's %v/%v", pubRC.Statistic, pubSPA.Statistic, rc.Statistic, spa.Statistic)
			}
		})
	}
}

// TestDataSnoopingDrawsMatchBlockBootstrap pins the joint resampling: each
// replication's positions are what BlockBootstrap with the same seed and
// block settings would draw for one path of length T.
func TestDataSnoopingDrawsMatchBlockBootstrap(t *testing.T) {
	const n, reps = 37, 25
	identity := make([]float64, n)
	for i := range identity {
		identity[i] = float64(i)
	}
	for _, scheme := range []ResamplingScheme{ResampleStationary, ResampleMovingBlock} {
		opts := DataSnoopingOptions{Replications: reps, BlockSize: 6, Seed: 99, Scheme: scheme}
		boot, err := blockBootstrapF64(identity, BootstrapConfig{Horizon: n, BlockSize: 6, Paths: reps, Seed: 99, Stationary: scheme == ResampleStationary})
		if err != nil {
			t.Fatal(err)
		}
		draw := jointDraws(n, opts)
		for p := range reps {
			got := draw()
			for i, pos := range got {
				if float64(pos) != boot.Returns[p][i] {
					t.Fatalf("%v: replication %d position %d = %d, BlockBootstrap drew %v", scheme, p, i, pos, boot.Returns[p][i])
				}
			}
		}
	}
}

// simulatedPerf builds a T×N table of correlated noise around zero, plus
// shift[j] added to column j.
func simulatedPerf(rng *rand.Rand, n int, shift []float64) [][]float64 {
	perf := make([][]float64, n)
	for t := range perf {
		common := rng.NormFloat64() * 0.01
		perf[t] = make([]float64, len(shift))
		for j := range shift {
			perf[t][j] = 0.7*common + 0.01*rng.NormFloat64() + shift[j]
		}
	}
	return perf
}

func TestDataSnoopingStrongStrategy(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	shift := make([]float64, 10)
	shift[4] = 0.004
	perf := simulatedPerf(rng, 500, shift)
	opts := DataSnoopingOptions{Replications: 1000, BlockSize: 10, Seed: 5}
	rc, err := RealityCheck(toDT(perf), opts)
	if err != nil {
		t.Fatal(err)
	}
	if rc.PValue > 0.01 || rc.Best != 4 {
		t.Errorf("RealityCheck = %+v, want p near 0 and best 4", rc)
	}
	spa, err := SPATest(toDT(perf), opts)
	if err != nil {
		t.Fatal(err)
	}
	if spa.PValueConsistent > 0.01 || spa.Best != 4 {
		t.Errorf("SPATest = %+v, want consistent p near 0 and best 4", spa)
	}
}

// TestSPAPoorStrategiesDoNotHideAGoodOne plants a modest winner among many
// columns far below the benchmark: the consistent p-value stays small, the
// ordering of the three p-values holds.
func TestSPAPoorStrategiesDoNotHideAGoodOne(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	shift := make([]float64, 41)
	for j := range shift {
		shift[j] = -0.01
	}
	shift[0] = 0.0015
	perf := simulatedPerf(rng, 500, shift)
	spa, err := SPATest(toDT(perf), DataSnoopingOptions{Replications: 1000, BlockSize: 10, Seed: 8})
	if err != nil {
		t.Fatal(err)
	}
	if spa.Best != 0 || spa.PValueConsistent > 0.02 {
		t.Errorf("SPATest = %+v, want best 0 and consistent p near 0", spa)
	}
	if !(spa.PValueLower <= spa.PValueConsistent && spa.PValueConsistent <= spa.PValueUpper) {
		t.Errorf("p-values out of order: %+v", spa)
	}
}

// TestDataSnoopingSizeUnderTheNull checks the tests do not reject too often
// when no column beats the benchmark: over 60 seeded null tables with every
// mean exactly at the benchmark, the share of p-values below 0.05 stays near
// 0.05, and the p-values keep their order in every table.
func TestDataSnoopingSizeUnderTheNull(t *testing.T) {
	rng := rand.New(rand.NewPCG(10, 20))
	const tables = 60
	var rejectRC, rejectSPA int
	for i := range tables {
		perf := simulatedPerf(rng, 250, make([]float64, 8))
		opts := DataSnoopingOptions{Replications: 300, BlockSize: 5, Seed: uint64(i)}
		rc, err := RealityCheck(toDT(perf), opts)
		if err != nil {
			t.Fatal(err)
		}
		spa, err := SPATest(toDT(perf), opts)
		if err != nil {
			t.Fatal(err)
		}
		if rc.PValue < 0.05 {
			rejectRC++
		}
		if spa.PValueConsistent < 0.05 {
			rejectSPA++
		}
		if !(spa.PValueLower <= spa.PValueConsistent && spa.PValueConsistent <= spa.PValueUpper) {
			t.Errorf("table %d: p-values out of order: %+v", i, spa)
		}
	}
	t.Logf("rejected %d (RC) and %d (SPA) of %d null tables at 5%%", rejectRC, rejectSPA, tables)
	// 3 rejections are expected; 10 or more has probability below 0.2%.
	if rejectRC >= 10 || rejectSPA >= 10 {
		t.Errorf("rejected %d (RC) and %d (SPA) of %d null tables at 5%%", rejectRC, rejectSPA, tables)
	}
}

func TestDataSnoopingReproducibleAndNamed(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 6))
	perf := simulatedPerf(rng, 120, []float64{0, 0.002, 0})
	cols := make([]*insyra.DataList, 3)
	for j, name := range []string{"slow", "fast", "mid"} {
		vals := make([]any, len(perf))
		for i := range perf {
			vals[i] = perf[i][j]
		}
		cols[j] = insyra.NewDataList(vals...).SetName(name)
	}
	table := insyra.NewDataTable(cols...)
	for _, scheme := range []ResamplingScheme{ResampleStationary, ResampleMovingBlock} {
		opts := DataSnoopingOptions{Replications: 200, BlockSize: 4, Seed: 7, Scheme: scheme}
		a, err := SPATest(table, opts)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := SPATest(table, opts)
		if !reflect.DeepEqual(a, b) {
			t.Errorf("%v: same options gave %+v and %+v", scheme, a, b)
		}
		if a.BestName != "fast" || a.Replications != 200 {
			t.Errorf("%v: best %q, replications %d", scheme, a.BestName, a.Replications)
		}
		r, err := RealityCheck(table, opts)
		if err != nil {
			t.Fatal(err)
		}
		if r.BestName != "fast" {
			t.Errorf("%v: RealityCheck best %q", scheme, r.BestName)
		}
	}
}

func TestDataSnoopingNilBenchmarkIsZero(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 8))
	perf := simulatedPerf(rng, 80, []float64{0.001, 0, -0.001})
	opts := DataSnoopingOptions{Replications: 100, BlockSize: 3, Seed: 1}
	withNil, err := SPATest(toDT(perf), opts)
	if err != nil {
		t.Fatal(err)
	}
	opts.Benchmark = toDL(make([]float64, 80)...)
	withZeros, err := SPATest(toDT(perf), opts)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(withNil, withZeros) {
		t.Errorf("nil benchmark %+v differs from zeros %+v", withNil, withZeros)
	}
}

func TestDataSnoopingRejectsBadInput(t *testing.T) {
	good := [][]float64{{0.01, 0.02}, {-0.01, 0.00}, {0.02, -0.01}, {0.00, 0.01}}
	ok := DataSnoopingOptions{Replications: 10, BlockSize: 2}
	cases := []struct {
		name string
		perf insyra.IDataTable
		opts DataSnoopingOptions
		want string
	}{
		{"nil table", nil, ok, "perf is nil"},
		{"no columns", insyra.NewDataTable(), ok, "no columns"},
		{"two rows", toDT(good[:2]), ok, "at least 3"},
		{"no replications", toDT(good), DataSnoopingOptions{BlockSize: 2}, "Replications"},
		{"block zero", toDT(good), DataSnoopingOptions{Replications: 10}, "BlockSize"},
		{"block too long", toDT(good), DataSnoopingOptions{Replications: 10, BlockSize: 5}, "BlockSize"},
		{"unknown scheme", toDT(good), DataSnoopingOptions{Replications: 10, BlockSize: 2, Scheme: ResamplingScheme(7)}, "ResamplingScheme(7)"},
		{"short benchmark", toDT(good), DataSnoopingOptions{Replications: 10, BlockSize: 2, Benchmark: toDL(0, 0, 0)}, "Benchmark has 3 values"},
		{"nil list benchmark", toDT(good), DataSnoopingOptions{Replications: 10, BlockSize: 2, Benchmark: (*insyra.DataList)(nil)}, "Benchmark is nil"},
		{"text cell", insyra.NewDataTable(insyra.NewDataList(0.01, "x", 0.02, 0.0)), ok, "perf column 0 contains a non-numeric value at row index 1"},
		{"NaN cell", insyra.NewDataTable(insyra.NewDataList(0.01, 0.0, math.NaN(), 0.0)), ok, "perf column 0 contains a non-finite value at row index 2"},
	}
	for _, c := range cases {
		_, err := RealityCheck(c.perf, c.opts)
		wantErrContains(t, err, c.want)
		_, err = SPATest(c.perf, c.opts)
		wantErrContains(t, err, c.want)
	}

	same := insyra.NewDataTable(insyra.NewDataList(0.01, 0.02, 0.03, 0.04), insyra.NewDataList(0.03, 0.01, 0.0, 0.02))
	_, err := SPATest(same, DataSnoopingOptions{Replications: 10, BlockSize: 2, Benchmark: toDL(0.03, 0.01, 0.0, 0.02)})
	wantErrContains(t, err, "column 1 has no variance relative to the benchmark")
	if _, err := RealityCheck(same, DataSnoopingOptions{Replications: 10, BlockSize: 2, Benchmark: toDL(0.03, 0.01, 0.0, 0.02)}); err != nil {
		t.Errorf("RealityCheck refused a column equal to the benchmark: %v", err)
	}
}
