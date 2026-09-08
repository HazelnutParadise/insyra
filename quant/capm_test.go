package quant

import (
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func assertClose(t *testing.T, got, want, tolerance float64) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Errorf("got %.17g, want %.17g (tolerance %.1g)", got, want, tolerance)
	}
}

func TestCAPMAgreesWithLinearRegression(t *testing.T) {
	rng := rand.New(rand.NewSource(20260903))
	asset := make([]float64, 64)
	market := make([]float64, 64)
	for i := range asset {
		market[i] = (rng.Float64() - 0.5) * 0.04
		asset[i] = 0.0007 + 1.25*market[i] + (rng.Float64()-0.5)*0.01
	}

	assetDL, marketDL := toDL(asset...), toDL(market...)
	got, err := CAPM(assetDL, marketDL, 0)
	if err != nil {
		t.Fatalf("CAPM returned unexpected error: %v", err)
	}
	want, err := stats.LinearRegression(assetDL, marketDL)
	if err != nil {
		t.Fatalf("LinearRegression returned unexpected error: %v", err)
	}
	assertClose(t, got.Beta, want.Slope, 1e-12)
	assertClose(t, got.Alpha, want.Intercept, 1e-12)
	assertClose(t, got.BetaStdErr, want.StandardError, 1e-12)
	assertClose(t, got.AlphaStdErr, want.StandardErrorIntercept, 1e-12)
	assertClose(t, got.RSquared, want.RSquared, 1e-12)
	if got.N != len(asset) {
		t.Errorf("N = %d, want %d", got.N, len(asset))
	}
	beta, err := Beta(assetDL, marketDL)
	if err != nil {
		t.Fatalf("Beta returned unexpected error: %v", err)
	}
	assertClose(t, beta, want.Slope, 1e-12)
}

func TestCAPMHandComputedGolden(t *testing.T) {
	asset := []float64{0.010, -0.004, 0.012, 0.003, -0.008, 0.006}
	market := []float64{0.006, -0.002, 0.008, 0.001, -0.005, 0.004}
	got, err := CAPM(toDL(asset...), toDL(market...), 0)
	if err != nil {
		t.Fatalf("CAPM returned unexpected error: %v", err)
	}
	// Hand calculation: x̄ = 0.002, ȳ = 0.003166666666666667,
	// Sxx = Σ(x−x̄)² = 0.000122, Sxy = Σ(x−x̄)(y−ȳ) = 0.000193.
	// Therefore β = Sxy/Sxx = 1.581967213114754, and
	// α = ȳ−βx̄ = 0.00000273224043715847. The residual sum of squares is
	// SSR = 0.000003513661202185792, while SST = 0.0003088333333333333,
	// so R² = 1−SSR/SST = 0.9886227915741421.
	assertClose(t, got.Beta, 1.581967213114754, 1e-9)
	assertClose(t, got.Alpha, 0.00000273224043715847, 1e-9)
	assertClose(t, got.RSquared, 0.9886227915741421, 1e-9)
	if got.N != 6 {
		t.Errorf("N = %d, want 6", got.N)
	}
}

func TestBetaOfScaledMarket(t *testing.T) {
	market := []float64{0.01, -0.02, 0.015, 0.004, -0.008}
	asset := make([]float64, len(market))
	for i, value := range market {
		asset[i] = 1.5*value + 0.001
	}
	got, err := Beta(toDL(asset...), toDL(market...))
	if err != nil {
		t.Fatalf("Beta returned unexpected error: %v", err)
	}
	assertClose(t, got, 1.5, 1e-12)
}

func TestCAPMConstantAsset(t *testing.T) {
	market := toDL(0.01, -0.02, 0.015, 0.004, -0.008)
	asset := toDL(0.007, 0.007, 0.007, 0.007, 0.007)

	beta, err := Beta(asset, market)
	if err != nil {
		t.Fatalf("Beta returned unexpected error: %v", err)
	}
	assertClose(t, beta, 0, 1e-12)

	got, err := CAPM(asset, market, 0)
	if err != nil {
		t.Fatalf("CAPM returned unexpected error: %v", err)
	}
	assertClose(t, got.Beta, 0, 1e-12)
	assertClose(t, got.Alpha, 0.007, 1e-12)
	if !math.IsNaN(got.RSquared) {
		t.Errorf("RSquared = %v, want NaN", got.RSquared)
	}
	assertClose(t, got.BetaStdErr, 0, 1e-12)
	assertClose(t, got.AlphaStdErr, 0, 1e-12)
}

func TestCAPMRiskFreeRate(t *testing.T) {
	asset := toDL(0.010, -0.004, 0.012, 0.003, -0.008, 0.006)
	market := toDL(0.006, -0.002, 0.008, 0.001, -0.005, 0.004)
	withoutRF, err := CAPM(asset, market, 0)
	if err != nil {
		t.Fatalf("CAPM without risk-free rate returned unexpected error: %v", err)
	}
	const riskFreeRate = 0.0002
	withRF, err := CAPM(asset, market, riskFreeRate)
	if err != nil {
		t.Fatalf("CAPM with risk-free rate returned unexpected error: %v", err)
	}
	beta, err := Beta(asset, market)
	if err != nil {
		t.Fatalf("Beta returned unexpected error: %v", err)
	}
	assertClose(t, withoutRF.Beta, beta, 1e-12)
	assertClose(t, withRF.Beta, beta, 1e-12)
	assertClose(t, withRF.Alpha, withoutRF.Alpha-riskFreeRate*(1-beta), 1e-12)
}

func TestCAPMRejectsInvalidInput(t *testing.T) {
	validAsset := toDL(0.01, -0.004, 0.012)
	validMarket := toDL(0.006, -0.002, 0.008)
	cases := []struct {
		name   string
		asset  insyra.IDataList
		market insyra.IDataList
		want   string
	}{
		{"length mismatch", toDL(0.01, 0.02, 0.03, 0.04), validMarket, "length"},
		{"too few observations", toDL(0.01, 0.02), toDL(0.006, 0.008), "at least 3"},
		{"flat benchmark", validAsset, toDL(0.006, 0.006, 0.006), "benchmark variance"},
		{"nil asset", nil, validMarket, "nil"},
		{"nil market", validAsset, nil, "nil"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Beta(tc.asset, tc.market); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Beta error = %v, want mention %q", err, tc.want)
			}
			if _, err := CAPM(tc.asset, tc.market, 0); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("CAPM error = %v, want mention %q", err, tc.want)
			}
		})
	}

	badCases := []struct {
		name   string
		asset  insyra.IDataList
		market insyra.IDataList
		series string
		row    string
	}{
		{"nil asset cell", insyra.NewDataList(nil, 0.02, 0.03), validMarket, "asset", "row 1"},
		{"asset text cell", insyra.NewDataList(0.01, 0.02, "n/a"), validMarket, "asset", "row 3"},
		{"market text cell", validAsset, insyra.NewDataList(0.006, "n/a", 0.008), "market", "row 2"},
		{"asset NaN", insyra.NewDataList(0.01, math.NaN(), 0.03), validMarket, "asset", "row 2"},
		{"market NaN", validAsset, insyra.NewDataList(0.006, math.NaN(), 0.008), "market", "row 2"},
		{"asset Inf", insyra.NewDataList(0.01, math.Inf(1), 0.03), validMarket, "asset", "row 2"},
		{"market Inf", validAsset, insyra.NewDataList(0.006, math.Inf(-1), 0.008), "market", "row 2"},
	}
	for _, tc := range badCases {
		t.Run(tc.name, func(t *testing.T) {
			check := func(name string, err error) {
				t.Helper()
				if err == nil {
					t.Fatalf("%s returned nil error", name)
				}
				if !strings.Contains(err.Error(), tc.series) || !strings.Contains(err.Error(), tc.row) {
					t.Errorf("%s error = %q, want %s and %s", name, err, tc.series, tc.row)
				}
			}
			_, betaErr := Beta(tc.asset, tc.market)
			check("Beta", betaErr)
			_, capmErr := CAPM(tc.asset, tc.market, 0)
			check("CAPM", capmErr)
		})
	}
}
