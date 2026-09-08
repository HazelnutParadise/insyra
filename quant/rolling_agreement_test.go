package quant

import (
	"math"
	"math/rand"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// The root package cannot import stats or quant from its own tests (import
// cycle), so the full-window agreement the datalist-rolling-cov-beta spec asks
// for is asserted here, where both sides are importable.
func TestRollingCovAndBetaAgreeWithStatsAndQuant(t *testing.T) {
	rng := rand.New(rand.NewSource(20260904))
	n := 40
	asset := make([]float64, n)
	market := make([]float64, n)
	for i := range asset {
		market[i] = (rng.Float64() - 0.5) * 0.04
		asset[i] = 0.0003 + 0.9*market[i] + (rng.Float64()-0.5)*0.01
	}
	assetDL, marketDL := toDL(asset...), toDL(market...)

	cov, err := stats.Covariance(assetDL, marketDL)
	if err != nil {
		t.Fatalf("Covariance: %v", err)
	}
	beta, err := Beta(assetDL, marketDL)
	if err != nil {
		t.Fatalf("Beta: %v", err)
	}

	rolling := insyra.NewDataList(toAny(asset)...).Rolling(insyra.RollingOptions{Window: n})
	other := insyra.NewDataList(toAny(market)...)
	gotCov := rolling.Cov(other).Data()
	gotBeta := rolling.Beta(other).Data()
	if len(gotCov) != n || len(gotBeta) != n {
		t.Fatalf("rolling output length = %d/%d, want %d", len(gotCov), len(gotBeta), n)
	}
	lastCov, ok := gotCov[n-1].(float64)
	if !ok || math.Abs(lastCov-cov) > 1e-12 {
		t.Errorf("Rolling.Cov full window = %v, stats.Covariance = %v", gotCov[n-1], cov)
	}
	lastBeta, ok := gotBeta[n-1].(float64)
	if !ok || math.Abs(lastBeta-beta) > 1e-12 {
		t.Errorf("Rolling.Beta full window = %v, quant.Beta = %v", gotBeta[n-1], beta)
	}
}

func toAny(vs []float64) []any {
	out := make([]any, len(vs))
	for i, v := range vs {
		out[i] = v
	}
	return out
}
