package stats_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestKNNClassifyReturnsPredictionsAndProbabilities(t *testing.T) {
	train := dataTableFromRows([][]float64{
		{0, 0},
		{0, 1},
		{1, 0},
		{10, 10},
		{10, 11},
		{11, 10},
	})
	test := dataTableFromRows([][]float64{
		{0.1, 0.2},
		{10.2, 10.1},
		{5, 5},
	})
	labels := insyra.NewDataList("red", "red", "red", "blue", "blue", "blue")

	got, err := stats.KNNClassify(train, labels, test, 3)
	if err != nil {
		t.Fatalf("KNNClassify returned error: %v", err)
	}

	if got.Predictions.Len() != 3 {
		t.Fatalf("expected 3 predictions, got %d", got.Predictions.Len())
	}
	if got.Predictions.Get(0) != "red" {
		t.Fatalf("expected first prediction red, got %v", got.Predictions.Get(0))
	}
	if got.Predictions.Get(1) != "blue" {
		t.Fatalf("expected second prediction blue, got %v", got.Predictions.Get(1))
	}
	if got.Probabilities == nil {
		t.Fatalf("expected probability table")
	}
	r, c := got.Probabilities.(*insyra.DataTable).Size()
	if r != 3 || c != 2 {
		t.Fatalf("expected probability shape 3x2, got %dx%d", r, c)
	}
	for i := 0; i < 3; i++ {
		row := got.Probabilities.(*insyra.DataTable).GetRow(i)
		sum := 0.0
		for j := 0; j < row.Len(); j++ {
			sum += row.Get(j).(float64)
		}
		if math.Abs(sum-1) > 1e-12 {
			t.Fatalf("row %d probability sum=%v, want 1", i, sum)
		}
	}
}

func TestKNNClassifyDistanceWeightingHandlesExactMatch(t *testing.T) {
	train := dataTableFromRows([][]float64{
		{0, 0},
		{0, 1},
		{10, 10},
	})
	test := dataTableFromRows([][]float64{
		{10, 10},
	})
	labels := insyra.NewDataList("left", "left", "right")

	got, err := stats.KNNClassify(train, labels, test, 3, stats.KNNOptions{Weighting: stats.KNNDistanceWeighting})
	if err != nil {
		t.Fatalf("KNNClassify returned error: %v", err)
	}
	if got.Predictions.Get(0) != "right" {
		t.Fatalf("expected exact-match class right, got %v", got.Predictions.Get(0))
	}
	row := got.Probabilities.(*insyra.DataTable).GetRow(0)
	if row.Get(1).(float64) != 1 {
		t.Fatalf("expected exact-match probability 1 for right class, got %v", row.Get(1))
	}
}

func TestKNNRegressSupportsUniformAndDistanceWeights(t *testing.T) {
	train := dataTableFromRows([][]float64{
		{0, 0},
		{0, 1},
		{10, 10},
		{10, 11},
	})
	test := dataTableFromRows([][]float64{
		{0.1, 0.2},
		{9.9, 10.1},
	})
	targets := insyra.NewDataList(1.0, 1.5, 9.0, 9.5)

	uniform, err := stats.KNNRegress(train, targets, test, 2)
	if err != nil {
		t.Fatalf("KNNRegress uniform returned error: %v", err)
	}
	if len(uniform.Predictions) != 2 {
		t.Fatalf("expected 2 regression predictions, got %d", len(uniform.Predictions))
	}
	if math.Abs(uniform.Predictions[0]-1.25) > 1e-12 {
		t.Fatalf("uniform prediction[0]=%v, want 1.25", uniform.Predictions[0])
	}
	if math.Abs(uniform.Predictions[1]-9.25) > 1e-12 {
		t.Fatalf("uniform prediction[1]=%v, want 9.25", uniform.Predictions[1])
	}

	distance, err := stats.KNNRegress(train, targets, test, 2, stats.KNNOptions{Weighting: stats.KNNDistanceWeighting})
	if err != nil {
		t.Fatalf("KNNRegress distance returned error: %v", err)
	}
	if !(distance.Predictions[0] > 1.0 && distance.Predictions[0] < 1.5) {
		t.Fatalf("distance prediction[0]=%v should lie between nearest targets", distance.Predictions[0])
	}
	if !(distance.Predictions[1] > 9.0 && distance.Predictions[1] < 9.5) {
		t.Fatalf("distance prediction[1]=%v should lie between nearest targets", distance.Predictions[1])
	}
}

func TestKNNRejectsInvalidInputs(t *testing.T) {
	train := dataTableFromRows([][]float64{
		{0, 0},
		{1, 1},
	})
	test := dataTableFromRows([][]float64{
		{0, 0},
	})

	if _, err := stats.KNNClassify(train, insyra.NewDataList("a"), test, 1); err == nil {
		t.Fatalf("expected error for label length mismatch")
	}
	if _, err := stats.KNNClassify(train, insyra.NewDataList("a", "b"), test, 0); err == nil {
		t.Fatalf("expected error for k <= 0")
	}
	if _, err := stats.KNNClassify(train, insyra.NewDataList("a", "b"), test, 3); err == nil {
		t.Fatalf("expected error for k > n")
	}
	if _, err := stats.KNNClassify(train, insyra.NewDataList("a", "b"), test, 1, stats.KNNOptions{Weighting: "bad"}); err == nil {
		t.Fatalf("expected error for unsupported weighting")
	}
	if _, err := stats.KNNRegress(train, insyra.NewDataList("x", "y"), test, 1); err == nil {
		t.Fatalf("expected error for non-numeric regression targets")
	}
}

func TestKNNClassifyReportsClassesInProbabilityColumns(t *testing.T) {
	train := dataTableFromRows([][]float64{
		{0, 0},
		{1, 1},
		{9, 9},
	})
	test := dataTableFromRows([][]float64{
		{0.1, 0.1},
	})
	labels := insyra.NewDataList("alpha", "alpha", "beta")

	got, err := stats.KNNClassify(train, labels, test, 2)
	if err != nil {
		t.Fatalf("KNNClassify returned error: %v", err)
	}
	if !reflect.DeepEqual(anyListToSlice(got.Classes), []any{"alpha", "beta"}) {
		t.Fatalf("unexpected classes: %v", anyListToSlice(got.Classes))
	}
}

func TestKNearestNeighborsReturnsIndicesDistancesAndBackendParity(t *testing.T) {
	train := dataTableFromRows([][]float64{
		{0, 0},
		{0, 1},
		{1, 0},
		{10, 10},
		{10, 11},
		{11, 10},
	})
	test := dataTableFromRows([][]float64{
		{0.1, 0.2},
		{10.1, 10.2},
	})

	brute, err := stats.KNearestNeighbors(train, test, 2, stats.KNNOptions{Algorithm: stats.KNNBruteForce})
	if err != nil {
		t.Fatalf("KNearestNeighbors brute returned error: %v", err)
	}
	kd, err := stats.KNearestNeighbors(train, test, 2, stats.KNNOptions{Algorithm: stats.KNNKDTree})
	if err != nil {
		t.Fatalf("KNearestNeighbors kd-tree returned error: %v", err)
	}
	ball, err := stats.KNearestNeighbors(train, test, 2, stats.KNNOptions{Algorithm: stats.KNNBallTree})
	if err != nil {
		t.Fatalf("KNearestNeighbors ball-tree returned error: %v", err)
	}

	wantFirst := []int{1, 2}
	wantSecond := []int{4, 5}
	if !reflect.DeepEqual(brute.Indices[0], wantFirst) {
		t.Fatalf("unexpected first brute neighbor indices: %v", brute.Indices[0])
	}
	if !reflect.DeepEqual(brute.Indices[1], wantSecond) {
		t.Fatalf("unexpected second brute neighbor indices: %v", brute.Indices[1])
	}
	if !reflect.DeepEqual(brute.Indices, kd.Indices) || !reflect.DeepEqual(brute.Indices, ball.Indices) {
		t.Fatalf("backend index mismatch: brute=%v kd=%v ball=%v", brute.Indices, kd.Indices, ball.Indices)
	}
	for i := range brute.Distances {
		for j := range brute.Distances[i] {
			if math.Abs(brute.Distances[i][j]-kd.Distances[i][j]) > 1e-12 || math.Abs(brute.Distances[i][j]-ball.Distances[i][j]) > 1e-12 {
				t.Fatalf("backend distance mismatch at row=%d col=%d: brute=%v kd=%v ball=%v", i, j, brute.Distances[i][j], kd.Distances[i][j], ball.Distances[i][j])
			}
		}
	}
}

func anyListToSlice(dl insyra.IDataList) []any {
	out := make([]any, dl.Len())
	for i := 0; i < dl.Len(); i++ {
		out[i] = dl.Get(i)
	}
	return out
}
