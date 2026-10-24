package stats_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestEveryTestResultIsAHypothesisTestResult(t *testing.T) {
	var _ = []stats.HypothesisTestResult{
		&stats.TTestResult{}, &stats.ZTestResult{}, &stats.FTestResult{},
		&stats.ChiSquareTestResult{}, &stats.CorrelationResult{},
		&stats.WilcoxonTestResult{}, &stats.MannWhitneyUResult{},
		&stats.KruskalWallisResult{}, &stats.FriedmanTestResult{},
	}
}

func TestBaseIsTheEmbeddedResultNotACopy(t *testing.T) {
	tt, err := stats.SingleSampleTTest(insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0), 0)
	if err != nil {
		t.Fatal(err)
	}
	mw, err := stats.MannWhitneyU(insyra.NewDataList(1.0, 2.0, 3.0), insyra.NewDataList(4.0, 5.0, 6.0), stats.TwoSided)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []stats.HypothesisTestResult{tt, mw} {
		if r.Base().PValue <= 0 || r.Base().PValue > 1 {
			t.Fatalf("p-value %v out of range", r.Base().PValue)
		}
	}
	if tt.Base() != &tt.TestResult {
		t.Fatal("Base returned a copy")
	}
	tt.Base().PValue = 0.5
	if tt.PValue != 0.5 {
		t.Fatal("a change through Base is not visible on the result")
	}
}
