package stats_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestCrossLangKNNMethods(t *testing.T) {
	requireCrossLangTools(t)

	t.Run("classify", func(t *testing.T) {
		trainRows := [][]float64{
			{0, 0}, {0, 1}, {1, 0},
			{10, 10}, {10, 11}, {11, 10},
		}
		testRows := [][]float64{
			{0.1, 0.2}, {10.1, 10.2}, {5, 5},
		}
		labels := []string{"red", "red", "red", "blue", "blue", "blue"}
		cases := []struct {
			name      string
			k         int
			weighting stats.KNNWeighting
			algo      stats.KNNAlgorithm
		}{
			{name: "uniform_brute", k: 3, weighting: stats.KNNUniformWeighting, algo: stats.KNNBruteForce},
			{name: "distance_kd_tree", k: 3, weighting: stats.KNNDistanceWeighting, algo: stats.KNNKDTree},
			{name: "distance_ball_tree", k: 3, weighting: stats.KNNDistanceWeighting, algo: stats.KNNBallTree},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := stats.KNNClassify(
					dataTableFromRows(trainRows),
					insyra.NewDataList(stringsToAny(labels)...),
					dataTableFromRows(testRows),
					tc.k,
					stats.KNNOptions{Weighting: tc.weighting, Algorithm: tc.algo},
				)
				if err != nil {
					t.Fatalf("KNNClassify error: %v", err)
				}
				payload := map[string]any{
					"train_rows": trainRows,
					"test_rows":  testRows,
					"labels":     labels,
					"k":          tc.k,
					"weighting":  string(tc.weighting),
				}
				rb := runRBaseline(t, "knn_classify", payload)
				pb := runPythonBaseline(t, "knn_classify", payload)
				assertStringSliceEqualToBoth(t, "predictions", dataListToStringSlice(t, got.Predictions), baselineStringSlice(t, rb, "predictions"), baselineStringSlice(t, pb, "predictions"))
				assertStringSliceEqualToBoth(t, "classes", dataListToStringSlice(t, got.Classes), baselineStringSlice(t, rb, "classes"), baselineStringSlice(t, pb, "classes"))
				assertMatrixCloseToBoth(t, "probabilities", tableToFloatMatrix(got.Probabilities.(*insyra.DataTable)), baselineFloatMatrix(t, rb, "probabilities"), baselineFloatMatrix(t, pb, "probabilities"), 1e-10)
			})
		}
	})

	t.Run("regress", func(t *testing.T) {
		trainRows := [][]float64{
			{0, 0}, {0, 1}, {10, 10}, {10, 11},
		}
		testRows := [][]float64{
			{0.1, 0.2}, {9.9, 10.1},
		}
		targets := []float64{1.0, 1.5, 9.0, 9.5}
		cases := []struct {
			name      string
			weighting stats.KNNWeighting
			algo      stats.KNNAlgorithm
		}{
			{name: "uniform_brute", weighting: stats.KNNUniformWeighting, algo: stats.KNNBruteForce},
			{name: "distance_ball_tree", weighting: stats.KNNDistanceWeighting, algo: stats.KNNBallTree},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := stats.KNNRegress(
					dataTableFromRows(trainRows),
					insyra.NewDataList(floatSliceToAny(targets)...),
					dataTableFromRows(testRows),
					2,
					stats.KNNOptions{Weighting: tc.weighting, Algorithm: tc.algo},
				)
				if err != nil {
					t.Fatalf("KNNRegress error: %v", err)
				}
				payload := map[string]any{
					"train_rows": trainRows,
					"test_rows":  testRows,
					"targets":    targets,
					"k":          2,
					"weighting":  string(tc.weighting),
				}
				rb := runRBaseline(t, "knn_regress", payload)
				pb := runPythonBaseline(t, "knn_regress", payload)
				assertSliceCloseToBoth(t, "predictions", got.Predictions, baselineFloatSlice(t, rb, "predictions"), baselineFloatSlice(t, pb, "predictions"), 1e-10)
			})
		}
	})

	t.Run("neighbors", func(t *testing.T) {
		trainRows := [][]float64{
			{0, 0}, {0, 1}, {1, 0},
			{10, 10}, {10, 11}, {11, 10},
		}
		testRows := [][]float64{
			{0.1, 0.2}, {10.1, 10.2},
		}
		got, err := stats.KNearestNeighbors(
			dataTableFromRows(trainRows),
			dataTableFromRows(testRows),
			2,
			stats.KNNOptions{Algorithm: stats.KNNKDTree},
		)
		if err != nil {
			t.Fatalf("KNearestNeighbors error: %v", err)
		}
		payload := map[string]any{
			"train_rows": trainRows,
			"test_rows":  testRows,
			"k":          2,
		}
		rb := runRBaseline(t, "knn_neighbors", payload)
		pb := runPythonBaseline(t, "knn_neighbors", payload)
		assertIntMatrixEqualToBoth(t, "indices", intPairs(got.Indices), baselineIntMatrix(t, rb, "indices"), baselineIntMatrix(t, pb, "indices"))
		assertMatrixCloseToBoth(t, "distances", got.Distances, baselineFloatMatrix(t, rb, "distances"), baselineFloatMatrix(t, pb, "distances"), 1e-10)
	})
}

func stringsToAny(xs []string) []any {
	out := make([]any, len(xs))
	for i, v := range xs {
		out[i] = v
	}
	return out
}

func floatSliceToAny(xs []float64) []any {
	out := make([]any, len(xs))
	for i, v := range xs {
		out[i] = v
	}
	return out
}

func dataListToStringSlice(t *testing.T, dl insyra.IDataList) []string {
	t.Helper()
	out := make([]string, dl.Len())
	for i := 0; i < dl.Len(); i++ {
		s, ok := dl.Get(i).(string)
		if !ok {
			t.Fatalf("data list index %d is %T, want string", i, dl.Get(i))
		}
		out[i] = s
	}
	return out
}

func intPairs(rows [][]int) [][2]int {
	out := make([][2]int, len(rows))
	for i := range rows {
		out[i] = [2]int{rows[i][0], rows[i][1]}
	}
	return out
}
