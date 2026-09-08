package quant

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// bootstrapInput is a small fat-tailed daily return series used across the
// bootstrap tests (50 values).
func bootstrapInput() []float64 {
	out := make([]float64, 50)
	for i := range out {
		// deterministic, non-symmetric, includes a -6% day and a +5% day
		out[i] = 0.001 + 0.01*math.Sin(float64(i)*0.7)
	}
	out[13] = -0.06
	out[37] = 0.05
	return out
}

func TestBlockBootstrapShapes(t *testing.T) {
	res, err := BlockBootstrap(toDL(bootstrapInput()...), BootstrapConfig{Horizon: 252, BlockSize: 20, Paths: 100, Seed: 7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Returns) != 100 || len(res.Equity) != 100 {
		t.Fatalf("got %d return paths and %d equity paths, want 100 each", len(res.Returns), len(res.Equity))
	}
	for p := range res.Returns {
		if len(res.Returns[p]) != 252 {
			t.Fatalf("Returns[%d] has %d steps, want 252", p, len(res.Returns[p]))
		}
		if len(res.Equity[p]) != 253 {
			t.Fatalf("Equity[%d] has %d steps, want 253", p, len(res.Equity[p]))
		}
		if res.Equity[p][0] != 1 {
			t.Fatalf("Equity[%d][0] = %v, want 1", p, res.Equity[p][0])
		}
	}
}

func TestBlockBootstrapFullBlockReproducesSeries(t *testing.T) {
	in := bootstrapInput()
	res, err := blockBootstrapF64(in, BootstrapConfig{Horizon: len(in), BlockSize: len(in), Paths: 3, Seed: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantEquity := make([]float64, len(in)+1)
	wantEquity[0] = 1
	for i, r := range in {
		wantEquity[i+1] = wantEquity[i] * (1 + r)
	}
	for p := range res.Returns {
		if !reflect.DeepEqual(res.Returns[p], in) {
			t.Errorf("Returns[%d] differs from the input series", p)
		}
		if !reflect.DeepEqual(res.Equity[p], wantEquity) {
			t.Errorf("Equity[%d] differs from the compounded input", p)
		}
	}
}

func TestBlockBootstrapConstantReturnCompounds(t *testing.T) {
	const r = 0.01
	in := make([]float64, 30)
	for i := range in {
		in[i] = r
	}
	for _, stationary := range []bool{false, true} {
		res, err := blockBootstrapF64(in, BootstrapConfig{Horizon: 40, BlockSize: 5, Paths: 4, Seed: 3, Stationary: stationary})
		if err != nil {
			t.Fatalf("stationary=%v: unexpected error: %v", stationary, err)
		}
		for p := range res.Equity {
			for step, got := range res.Equity[p] {
				want := math.Pow(1+r, float64(step))
				if math.Abs(got-want) > 1e-12 {
					t.Fatalf("stationary=%v: Equity[%d][%d] = %v, want %v", stationary, p, step, got, want)
				}
			}
		}
	}
}

func TestBlockBootstrapReproducible(t *testing.T) {
	in := bootstrapInput()
	for _, stationary := range []bool{false, true} {
		cfg := BootstrapConfig{Horizon: 100, BlockSize: 10, Paths: 20, Seed: 42, Stationary: stationary}
		a, err := blockBootstrapF64(in, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		b, _ := blockBootstrapF64(in, cfg)
		if !reflect.DeepEqual(a, b) {
			t.Errorf("stationary=%v: same seed produced different output", stationary)
		}
		cfg.Seed = 43
		c, _ := blockBootstrapF64(in, cfg)
		if reflect.DeepEqual(a.Returns, c.Returns) {
			t.Errorf("stationary=%v: different seeds produced identical output", stationary)
		}
	}
}

// TestBlockBootstrapGolden pins the first resampled values for a fixed seed
// so a change in the random stream (PCG seeding or the in-package
// reductions) is caught rather than silently shifting every user's
// forecast. The input is the index itself, so each value names the source
// position it was drawn from.
func TestBlockBootstrapGolden(t *testing.T) {
	in := make([]float64, 10)
	for i := range in {
		in[i] = float64(i)
	}
	moving, err := blockBootstrapF64(in, BootstrapConfig{Horizon: 8, BlockSize: 3, Paths: 1, Seed: 2024})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantMoving := []float64{0, 1, 2, 4, 5, 6, 4, 5}
	if !reflect.DeepEqual(moving.Returns[0], wantMoving) {
		t.Errorf("moving block golden = %v, want %v", moving.Returns[0], wantMoving)
	}
	stationary, err := blockBootstrapF64(in, BootstrapConfig{Horizon: 8, BlockSize: 3, Paths: 1, Seed: 2024, Stationary: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantStationary := []float64{2, 3, 4, 5, 4, 4, 5, 6}
	if !reflect.DeepEqual(stationary.Returns[0], wantStationary) {
		t.Errorf("stationary golden = %v, want %v", stationary.Returns[0], wantStationary)
	}
}

func TestGeometricBlockLengthMean(t *testing.T) {
	rng := newBootstrapRNG(9)
	const draws = 200000
	for _, mean := range []int{1, 2, 5, 20} {
		sum := 0
		for range draws {
			l := geometricBlockLength(rng, mean)
			if l < 1 {
				t.Fatalf("mean %d: drew length %d < 1", mean, l)
			}
			sum += l
		}
		got := float64(sum) / draws
		// standard error of the mean is sqrt(mean*(mean-1)/draws) < 0.05 for mean 20
		if math.Abs(got-float64(mean)) > 0.25 {
			t.Errorf("mean block length for BlockSize %d = %.3f, want ≈ %d", mean, got, mean)
		}
	}
}

func TestAppendCircularWraps(t *testing.T) {
	src := []float64{0, 1, 2, 3, 4}
	got := appendCircular(nil, src, 3, 4)
	want := []float64{3, 4, 0, 1}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("appendCircular = %v, want %v", got, want)
	}
	if got := appendCircular([]float64{9}, src, 4, 7); !reflect.DeepEqual(got, []float64{9, 4, 0, 1, 2, 3, 4, 0}) {
		t.Errorf("appendCircular with prefix = %v", got)
	}
}

func TestStationaryBootstrapWrapsAndStaysContiguous(t *testing.T) {
	// Identity input: every value names its source index, so a within-block
	// step is +1 mod n and a wrap shows up as n-1 followed by 0.
	n := 7
	in := make([]float64, n)
	for i := range in {
		in[i] = float64(i)
	}
	res, err := blockBootstrapF64(in, BootstrapConfig{Horizon: 5000, BlockSize: 4, Paths: 1, Seed: 11, Stationary: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := res.Returns[0]
	wraps, breaks := 0, 0
	for i := 1; i < len(s); i++ {
		prev, cur := int(s[i-1]), int(s[i])
		if cur != (prev+1)%n {
			breaks++
		} else if prev == n-1 {
			wraps++
		}
	}
	if wraps == 0 {
		t.Error("expected at least one wrap-around past the end of the series")
	}
	// Mean block length 4 over 5000 steps → about 1250 blocks; a new block
	// starts contiguously with probability 1/n, so breaks ≈ 1250·6/7.
	if breaks < 800 || breaks > 1500 {
		t.Errorf("got %d block breaks, expected roughly 1070", breaks)
	}
}

// TestBlockStartsAreUniform checks the in-package uniform reduction through
// both schemes: with BlockSize 1 every value is one independent start
// draw, so the identity input's histogram must be flat over all n indices.
func TestBlockStartsAreUniform(t *testing.T) {
	const n, horizon = 7, 70000
	in := make([]float64, n)
	for i := range in {
		in[i] = float64(i)
	}
	for _, stationary := range []bool{false, true} {
		res, err := blockBootstrapF64(in, BootstrapConfig{Horizon: horizon, BlockSize: 1, Paths: 1, Seed: 17, Stationary: stationary})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		counts := make([]int, n)
		for _, v := range res.Returns[0] {
			counts[int(v)]++
		}
		expected := float64(horizon) / n // 10000; std dev ≈ 93, so ±5% is > 5σ
		for i, c := range counts {
			if math.Abs(float64(c)-expected) > 0.05*expected {
				t.Errorf("stationary=%v: start index %d drawn %d times, want ≈ %.0f", stationary, i, c, expected)
			}
		}
	}
}

func TestBlockBootstrapRejectsBadConfig(t *testing.T) {
	in := bootstrapInput()
	cases := []struct {
		name string
		cfg  BootstrapConfig
		want string
	}{
		{"zero horizon", BootstrapConfig{Horizon: 0, BlockSize: 5, Paths: 10}, "Horizon"},
		{"zero paths", BootstrapConfig{Horizon: 10, BlockSize: 5, Paths: 0}, "Paths"},
		{"zero block", BootstrapConfig{Horizon: 10, BlockSize: 0, Paths: 10}, "BlockSize"},
		{"block longer than series", BootstrapConfig{Horizon: 10, BlockSize: len(in) + 1, Paths: 10}, "BlockSize"},
	}
	for _, c := range cases {
		res, err := blockBootstrapF64(in, c.cfg)
		if err == nil {
			t.Errorf("%s: expected error", c.name)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %q does not mention %s", c.name, err, c.want)
		}
		if res != nil {
			t.Errorf("%s: expected nil result", c.name)
		}
	}
	if _, err := blockBootstrapF64(nil, BootstrapConfig{Horizon: 10, BlockSize: 1, Paths: 1}); err == nil {
		t.Error("empty series: expected error")
	}
}

func TestBlockBootstrapRejectsUnreadableInput(t *testing.T) {
	cfg := BootstrapConfig{Horizon: 5, BlockSize: 2, Paths: 2, Seed: 1}
	if _, err := BlockBootstrap(nil, cfg); err == nil {
		t.Error("nil DataList: expected error")
	}
	bad := insyra.NewDataList(0.01, 0.02, "n/a", 0.03)
	res, err := BlockBootstrap(bad, cfg)
	if err == nil {
		t.Fatal("non-numeric element: expected error")
	}
	if !strings.Contains(err.Error(), "row 3") {
		t.Errorf("error %q does not name row 3", err)
	}
	if res != nil {
		t.Error("expected nil result on refusal")
	}
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		dl := insyra.NewDataList(0.01, v, 0.02)
		if _, err := BlockBootstrap(dl, cfg); err == nil || !strings.Contains(err.Error(), "row 2") {
			t.Errorf("value %v: expected an error naming row 2, got %v", v, err)
		}
	}
}

func TestPercentileBandsMatchDataListPercentile(t *testing.T) {
	res, err := blockBootstrapF64(bootstrapInput(), BootstrapConfig{Horizon: 30, BlockSize: 7, Paths: 101, Seed: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	percentiles := []float64{0, 5, 25, 50, 75, 95, 100, 33.3}
	bands, err := PercentileBands(res.Equity, percentiles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bands) != len(percentiles) {
		t.Fatalf("got %d bands, want %d", len(bands), len(percentiles))
	}
	width := len(res.Equity[0])
	for step := range width {
		col := make([]any, len(res.Equity))
		for p := range res.Equity {
			col[p] = res.Equity[p][step]
		}
		dl := insyra.NewDataList(col...)
		for i, q := range percentiles {
			if len(bands[i]) != width {
				t.Fatalf("band %d has %d steps, want %d", i, len(bands[i]), width)
			}
			if got, want := bands[i][step], dl.Percentile(q); got != want {
				t.Fatalf("band p%v step %d = %v, DataList.Percentile = %v", q, step, got, want)
			}
		}
	}
}

func TestPercentileBandsOrderAndMonotone(t *testing.T) {
	paths := [][]float64{{1, 4}, {2, 5}, {3, 6}, {4, 7}}
	bands, err := PercentileBands(paths, []float64{95, 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bands[0][0] <= bands[1][0] || bands[0][1] <= bands[1][1] {
		t.Errorf("caller order not preserved: p95 band %v, p5 band %v", bands[0], bands[1])
	}
	asc, _ := PercentileBands(paths, []float64{5, 25, 50, 75, 95})
	for t2 := range 2 {
		for i := 1; i < len(asc); i++ {
			if asc[i][t2] < asc[i-1][t2] {
				t.Errorf("bands not monotone at step %d: %v < %v", t2, asc[i][t2], asc[i-1][t2])
			}
		}
	}
}

func TestPercentileBandsRejectsBadInput(t *testing.T) {
	good := [][]float64{{1, 2}, {3, 4}}
	cases := []struct {
		name        string
		paths       [][]float64
		percentiles []float64
	}{
		{"empty paths", nil, []float64{50}},
		{"no steps", [][]float64{{}}, []float64{50}},
		{"ragged paths", [][]float64{{1, 2}, {3}}, []float64{50}},
		{"empty percentiles", good, nil},
		{"percentile above 100", good, []float64{50, 101}},
		{"negative percentile", good, []float64{-1}},
		{"NaN percentile", good, []float64{math.NaN()}},
	}
	for _, c := range cases {
		if bands, err := PercentileBands(c.paths, c.percentiles); err == nil || bands != nil {
			t.Errorf("%s: expected error and nil bands, got %v, %v", c.name, bands, err)
		}
	}
}
