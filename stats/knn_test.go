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

// ============================================================
// R reference suite — Batch 9
// ============================================================

const (
	tolKNN = 1e-12
)

// knnRef is the same physical file as clusterRef but accessed via its own
// var; the refTable cache makes either path equivalent. Declared separately
// here to keep KNN tests independent from clustering tests in case the
// reference files diverge later.

func TestKNNClassify_R(t *testing.T) {
	t.Run("basic_k3_two_class", func(t *testing.T) {
		train := dataTableFromRows([][]float64{
			{0, 0}, {0, 1}, {1, 0},
			{10, 10}, {10, 11}, {11, 10},
		})
		test := dataTableFromRows([][]float64{
			{0.1, 0.2}, {10.2, 10.1}, {5, 5},
		})
		labels := insyra.NewDataList("red", "red", "red", "blue", "blue", "blue")

		got, err := stats.KNNClassify(train, labels, test, 3)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		// Class column order is first-appearance from training labels:
		// labels = [red, red, red, blue, blue, blue] → classes = [red, blue].
		assertKNNClassify(t, got, "knn_basic_k3", 3, []string{"red", "blue"})
	})

	t.Run("3class_k5_n_train_20", func(t *testing.T) {
		train := dataTableFromRows(loadKnnTrain3c(t))
		// R script picks rows 1, 5, 10, 15, 20 from train as test.
		trainRows := loadKnnTrain3c(t)
		testRows := [][]float64{
			trainRows[0], trainRows[4], trainRows[9], trainRows[14], trainRows[19],
		}
		test := dataTableFromRows(testRows)
		labels := insyra.NewDataList(
			"a", "a", "a", "a", "a", "a", "a",
			"b", "b", "b", "b", "b", "b", "b",
			"c", "c", "c", "c", "c", "c",
		)
		got, err := stats.KNNClassify(train, labels, test, 5)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		assertKNNClassify(t, got, "knn_3class_k5", 5, []string{"a", "b", "c"})
	})
}

func loadKnnTrain3c(t *testing.T) [][]float64 {
	t.Helper()
	rows := make([][]float64, 20)
	for i := range rows {
		rows[i] = clusterDump.get(t, "knn3c_train_row"+itoa(i))
	}
	return rows
}

func assertKNNClassify(t *testing.T, got *stats.KNNClassificationResult, prefix string, nTest int, classes []string) {
	t.Helper()
	if got.Predictions.Len() != nTest {
		t.Fatalf("predictions length: got %d, want %d", got.Predictions.Len(), nTest)
	}
	probTbl, ok := got.Probabilities.(*insyra.DataTable)
	if !ok {
		t.Fatalf("Probabilities is not *DataTable: %T", got.Probabilities)
	}
	rows, cols := probTbl.Size()
	if rows != nTest || cols != len(classes) {
		t.Fatalf("Probabilities shape: got (%d,%d), want (%d,%d)",
			rows, cols, nTest, len(classes))
	}
	// Class column order — first-appearance, matches R reference.
	for ci := range len(classes) {
		expClass := clusterRef.getString(t, prefix+".class["+itoa(ci)+"]_str")
		gotClass, ok := got.Classes.Get(ci).(string)
		if !ok {
			t.Fatalf("class[%d] not string: %T", ci, got.Classes.Get(ci))
		}
		if gotClass != expClass {
			t.Errorf("class[%d]: got %q, want %q", ci, gotClass, expClass)
		}
	}
	for q := range nTest {
		row := probTbl.GetRow(q)
		probs := make([]float64, len(classes))
		for ci := range len(classes) {
			expProb := clusterRef.get(t, prefix+".prob["+itoa(q)+"]["+itoa(ci)+"]")
			gotProb, ok := row.Get(ci).(float64)
			if !ok {
				t.Fatalf("prob[%d][%d] not float: %T", q, ci, row.Get(ci))
			}
			probs[ci] = gotProb
			if math.Abs(gotProb-expProb) > tolKNN {
				t.Errorf("prob[%d][%d]: got %.17g, want %.17g", q, ci, gotProb, expProb)
			}
		}
		// Prediction must be the class with strict-max probability. Skip when
		// the top probabilities are tied (insyra's tie-breaker uses mean
		// distance per class, which we don't reproduce in the R reference).
		maxIdx, secondIdx := 0, -1
		for ci := 1; ci < len(probs); ci++ {
			if probs[ci] > probs[maxIdx] {
				secondIdx = maxIdx
				maxIdx = ci
			} else if secondIdx < 0 || probs[ci] > probs[secondIdx] {
				secondIdx = ci
			}
		}
		if secondIdx >= 0 && math.Abs(probs[maxIdx]-probs[secondIdx]) <= tolKNN {
			continue // tied → skip prediction equality check
		}
		gotPred, ok := got.Predictions.Get(q).(string)
		if !ok {
			t.Fatalf("pred[%d] not string: %T", q, got.Predictions.Get(q))
		}
		if gotPred != classes[maxIdx] {
			t.Errorf("pred[%d]: got %q, want %q (probabilities=%v)",
				q, gotPred, classes[maxIdx], probs)
		}
	}
}

func TestKNNRegress_R(t *testing.T) {
	train := dataTableFromRows([][]float64{
		{0, 0}, {0, 1}, {10, 10}, {10, 11},
	})
	test := dataTableFromRows([][]float64{
		{0.1, 0.2}, {9.9, 10.1},
	})
	targets := insyra.NewDataList(1.0, 1.5, 9.0, 9.5)
	got, err := stats.KNNRegress(train, targets, test, 2)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	for q := range 2 {
		exp := clusterRef.get(t, "knn_reg_k2_uniform.pred["+itoa(q)+"]")
		if math.Abs(got.Predictions[q]-exp) > tolKNN {
			t.Errorf("pred[%d]: got %.17g, want %.17g", q, got.Predictions[q], exp)
		}
	}
}

func TestKNearestNeighbors_R(t *testing.T) {
	train := dataTableFromRows([][]float64{
		{0, 0}, {0, 1}, {1, 0},
		{10, 10}, {10, 11}, {11, 10},
	})
	test := dataTableFromRows([][]float64{
		{0.1, 0.2}, {10.1, 10.2},
	})

	for _, alg := range []struct {
		name string
		opt  stats.KNNAlgorithm
	}{
		{"brute", stats.KNNBruteForce},
		{"kd_tree", stats.KNNKDTree},
		{"ball_tree", stats.KNNBallTree},
	} {
		t.Run(alg.name, func(t *testing.T) {
			got, err := stats.KNearestNeighbors(train, test, 2, stats.KNNOptions{Algorithm: alg.opt})
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			for q := range 2 {
				for j := range 2 {
					expIdx := int(clusterRef.get(t, "knn_nbr_k2.idx["+itoa(q)+"]["+itoa(j)+"]"))
					if got.Indices[q][j] != expIdx {
						t.Errorf("idx[%d][%d]: got %d, want %d", q, j, got.Indices[q][j], expIdx)
					}
					expDist := clusterRef.get(t, "knn_nbr_k2.dist["+itoa(q)+"]["+itoa(j)+"]")
					if math.Abs(got.Distances[q][j]-expDist) > tolKNN {
						t.Errorf("dist[%d][%d]: got %.17g, want %.17g",
							q, j, got.Distances[q][j], expDist)
					}
				}
			}
		})
	}
}
