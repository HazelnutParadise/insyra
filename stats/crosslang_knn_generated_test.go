package stats_test

import (
	"os"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestCrossLangKNNGeneratedCorpus(t *testing.T) {
	if os.Getenv("INSYRA_RUN_SLOW_KNN_PARITY") != "1" {
		t.Skip("set INSYRA_RUN_SLOW_KNN_PARITY=1 to run generated KNN parity corpus")
	}
	requireCrossLangTools(t)

	for _, tc := range generatedKNNClassifyParityCases() {
		t.Run("classify/"+tc.name, func(t *testing.T) {
			got, err := stats.KNNClassify(
				dataTableFromRows(tc.trainRows),
				insyra.NewDataList(stringsToAny(tc.labels)...),
				dataTableFromRows(tc.testRows),
				tc.k,
				stats.KNNOptions{Weighting: tc.weighting, Algorithm: tc.algorithm},
			)
			if err != nil {
				t.Fatalf("KNNClassify error: %v", err)
			}
			payload := map[string]any{
				"train_rows": tc.trainRows,
				"test_rows":  tc.testRows,
				"labels":     tc.labels,
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

	for _, tc := range generatedKNNRegressionParityCases() {
		t.Run("regress/"+tc.name, func(t *testing.T) {
			got, err := stats.KNNRegress(
				dataTableFromRows(tc.trainRows),
				insyra.NewDataList(floatSliceToAny(tc.targets)...),
				dataTableFromRows(tc.testRows),
				tc.k,
				stats.KNNOptions{Weighting: tc.weighting, Algorithm: tc.algorithm},
			)
			if err != nil {
				t.Fatalf("KNNRegress error: %v", err)
			}
			payload := map[string]any{
				"train_rows": tc.trainRows,
				"test_rows":  tc.testRows,
				"targets":    tc.targets,
				"k":          tc.k,
				"weighting":  string(tc.weighting),
			}
			rb := runRBaseline(t, "knn_regress", payload)
			pb := runPythonBaseline(t, "knn_regress", payload)
			assertSliceCloseToBoth(t, "predictions", got.Predictions, baselineFloatSlice(t, rb, "predictions"), baselineFloatSlice(t, pb, "predictions"), 1e-10)
		})
	}

	for _, tc := range generatedKNNNeighborParityCases() {
		t.Run("neighbors/"+tc.name, func(t *testing.T) {
			got, err := stats.KNearestNeighbors(
				dataTableFromRows(tc.trainRows),
				dataTableFromRows(tc.testRows),
				tc.k,
				stats.KNNOptions{Algorithm: tc.algorithm},
			)
			if err != nil {
				t.Fatalf("KNearestNeighbors error: %v", err)
			}
			payload := map[string]any{
				"train_rows": tc.trainRows,
				"test_rows":  tc.testRows,
				"k":          tc.k,
			}
			rb := runRBaseline(t, "knn_neighbors", payload)
			pb := runPythonBaseline(t, "knn_neighbors", payload)
			assertIntRowsEqualToBoth(t, "indices", got.Indices, baselineIntRows(t, rb, "indices"), baselineIntRows(t, pb, "indices"))
			assertMatrixCloseToBoth(t, "distances", got.Distances, baselineFloatMatrix(t, rb, "distances"), baselineFloatMatrix(t, pb, "distances"), 1e-10)
		})
	}
}
