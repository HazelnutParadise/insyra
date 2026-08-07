package ml_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
)

func ridgeCandidate(name string, alpha float64) ml.Estimator {
	return ml.Estimator{
		Name: name,
		Fit: func(x *insyra.DataTable, y *insyra.DataList) (ml.Model, error) {
			return ml.FitRidgeRegression(x, y, alpha)
		},
	}
}

// On nearly noiseless linear data, a crushing penalty must lose to a light one
// whichever way the metric points — this is the direction actually deciding.
func TestGridSearchPicksTheKnowableWinnerInBothDirections(t *testing.T) {
	x, y := regularizedTable()
	candidates := []ml.Estimator{
		ridgeCandidate("crushed", 1e6),
		ridgeCandidate("light", 0.01),
	}

	byLoss, err := ml.GridSearch(x, y, candidates, 4, ml.RMSEMetric{})
	if err != nil {
		t.Fatalf("grid search by RMSE: %v", err)
	}
	if byLoss.BestName != "light" {
		t.Fatalf("RMSE picked %q; a crushing penalty on linear data cannot win", byLoss.BestName)
	}

	byGain, err := ml.GridSearch(x, y, candidates, 4, ml.R2Metric{})
	if err != nil {
		t.Fatalf("grid search by R²: %v", err)
	}
	if byGain.BestName != "light" {
		t.Fatalf("R² picked %q", byGain.BestName)
	}
}

// The same-folds guarantee: identical candidates must earn identical per-fold
// scores, which only happens if their folds were identical.
func TestGridSearchScoresEveryCandidateOnIdenticalFolds(t *testing.T) {
	x, y := regularizedTable()
	result, err := ml.GridSearch(x, y, []ml.Estimator{
		ridgeCandidate("twin-a", 0.5),
		ridgeCandidate("twin-b", 0.5),
	}, 4, ml.RMSEMetric{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Results[0].Scores, result.Results[1].Scores) {
		t.Fatalf("identical candidates scored differently:\n%v\n%v\nfolds were not shared", result.Results[0].Scores, result.Results[1].Scores)
	}
	// Ties keep the earliest candidate.
	if result.BestName != "twin-a" {
		t.Fatalf("tie went to %q, want the earlier candidate", result.BestName)
	}
}

func TestGridSearchSeedReproducesTheRun(t *testing.T) {
	x, y := regularizedTable()
	candidates := []ml.Estimator{
		ridgeCandidate("crushed", 1e6),
		ridgeCandidate("light", 0.01),
	}
	first, err := ml.GridSearch(x, y, candidates, 4, ml.RMSEMetric{})
	if err != nil {
		t.Fatal(err)
	}
	again, err := ml.GridSearch(x, y, candidates, 4, ml.RMSEMetric{},
		insyra.SamplingOptions{UseSeed: true, Seed: first.Seed})
	if err != nil {
		t.Fatal(err)
	}
	if again.Seed != first.Seed {
		t.Fatalf("echoed seed %d, want %d", again.Seed, first.Seed)
	}
	for i := range first.Results {
		if !reflect.DeepEqual(first.Results[i].Scores, again.Results[i].Scores) {
			t.Fatalf("candidate %d scored differently under the reported seed", i)
		}
	}
	if again.BestName != first.BestName {
		t.Fatalf("winner changed under the reported seed: %q vs %q", again.BestName, first.BestName)
	}
}

func TestGridSearchRefitsTheWinnerOnTheFullData(t *testing.T) {
	x, y := regularizedTable()
	result, err := ml.GridSearch(x, y, []ml.Estimator{ridgeCandidate("only", 0.5)}, 4, ml.RMSEMetric{})
	if err != nil {
		t.Fatal(err)
	}
	direct, err := ml.FitRidgeRegression(x, y, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	fromSearch, err := result.BestModel.Predict(x)
	if err != nil {
		t.Fatal(err)
	}
	fromDirect, err := direct.Predict(x)
	if err != nil {
		t.Fatal(err)
	}
	// A fold-fitted model would disagree with the full-data fit; the refit
	// must be indistinguishable from fitting directly.
	if !reflect.DeepEqual(fromSearch.Data(), fromDirect.Data()) {
		t.Fatal("the returned model does not match a full-data fit; a fold model leaked out")
	}
}

func TestGridSearchRefusals(t *testing.T) {
	x, y := regularizedTable()
	valid := ridgeCandidate("ok", 0.5)

	if _, err := ml.GridSearch(x, y, nil, 4, ml.RMSEMetric{}); err == nil {
		t.Fatal("an empty grid was searched")
	}
	unnamed := valid
	unnamed.Name = ""
	if _, err := ml.GridSearch(x, y, []ml.Estimator{unnamed}, 4, ml.RMSEMetric{}); err == nil {
		t.Fatal("an unnamed candidate was accepted")
	}
	if _, err := ml.GridSearch(x, y, []ml.Estimator{valid, ridgeCandidate("ok", 1.0)}, 4, ml.RMSEMetric{}); err == nil {
		t.Fatal("duplicate names were accepted")
	}
	noFit := ml.Estimator{Name: "hollow"}
	if _, err := ml.GridSearch(x, y, []ml.Estimator{noFit}, 4, ml.RMSEMetric{}); err == nil {
		t.Fatal("a candidate without Fit was accepted")
	}
	if _, err := ml.GridSearch(x, y, []ml.Estimator{valid}, 4, ml.ConfusionMatrixMetric{}); err == nil {
		t.Fatal("a directionless metric searched a grid")
	}

	failing := ml.Estimator{
		Name: "broken",
		Fit: func(*insyra.DataTable, *insyra.DataList) (ml.Model, error) {
			return nil, fmt.Errorf("deliberate failure")
		},
	}
	_, err := ml.GridSearch(x, y, []ml.Estimator{failing}, 4, ml.RMSEMetric{})
	if err == nil {
		t.Fatal("a failing candidate did not fail the search")
	}
	if !strings.Contains(err.Error(), "broken") {
		t.Fatalf("error %q does not name the failing candidate", err)
	}
}
