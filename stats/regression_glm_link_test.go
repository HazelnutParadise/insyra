package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestSpecializedGLMResultsPublishLinks(t *testing.T) {
	logistic, err := stats.LogisticRegressionWithOptions(
		stats.LogisticRegressionOptions{MaxIter: 100, Tolerance: 1e-10},
		insyra.NewDataList(0, 0, 1, 0, 1, 1, 0, 1, 1, 0),
		insyra.NewDataList(-2.1, -1.5, -0.4, 0.1, 0.6, 1.2, 1.7, 2.1, 2.8, 3.3),
	)
	if err != nil {
		t.Fatalf("logistic fit: %v", err)
	}
	if logistic.Link != stats.Logit {
		t.Fatalf("logistic link = %q, want %q", logistic.Link, stats.Logit)
	}
	assertPublishedLinkMatchesPrediction(t, logistic.Link, logistic.Coefficients, logistic.Predict, insyra.NewDataList(-1.0, 1.0, 3.0))

	poisson, err := stats.PoissonRegressionWithOptions(
		stats.PoissonRegressionOptions{MaxIter: 100, Tolerance: 1e-10},
		insyra.NewDataList(1, 2, 3, 5, 8, 13),
		insyra.NewDataList(0, 1, 2, 3, 4, 5),
	)
	if err != nil {
		t.Fatalf("poisson fit: %v", err)
	}
	if poisson.Link != stats.Log {
		t.Fatalf("poisson link = %q, want %q", poisson.Link, stats.Log)
	}
	assertPublishedLinkMatchesPrediction(t, poisson.Link, poisson.Coefficients, poisson.Predict, insyra.NewDataList(0.5, 1.0, 2.5))
}

func assertPublishedLinkMatchesPrediction(
	t *testing.T,
	link stats.GLMLink,
	coefficients []float64,
	predict func(stats.PredictType, ...insyra.IDataList) (*insyra.DataList, error),
	newX insyra.IDataList,
) {
	t.Helper()
	linear, err := predict(stats.PredictLinear, newX)
	if err != nil {
		t.Fatalf("linear prediction: %v", err)
	}
	response, err := predict(stats.PredictResponse, newX)
	if err != nil {
		t.Fatalf("response prediction: %v", err)
	}
	eta := linear.ToF64Slice()
	mu := response.ToF64Slice()
	if len(eta) != len(mu) {
		t.Fatalf("linear and response prediction lengths differ: %d vs %d", len(eta), len(mu))
	}
	if len(coefficients) == 0 {
		t.Fatal("fit has no coefficients")
	}
	for i := range eta {
		got := applyPublishedLink(link, eta[i])
		if math.Abs(got-mu[i]) > 1e-12 {
			t.Fatalf("published %q link at row %d = %.15g, Predict = %.15g", link, i, got, mu[i])
		}
	}
}

func applyPublishedLink(link stats.GLMLink, eta float64) float64 {
	switch link {
	case stats.Logit:
		return 1 / (1 + math.Exp(-eta))
	case stats.Log:
		return math.Exp(eta)
	default:
		panic("unexpected published link: " + string(link))
	}
}
