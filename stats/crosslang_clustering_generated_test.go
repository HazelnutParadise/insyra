package stats_test

import (
	"os"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestCrossLangClusteringGeneratedCorpus(t *testing.T) {
	if os.Getenv("INSYRA_RUN_SLOW_PARITY") != "1" {
		t.Skip("set INSYRA_RUN_SLOW_PARITY=1 to run generated clustering parity corpus")
	}
	requireCrossLangTools(t)

	for _, tc := range generatedKMeansParityCases() {
		t.Run("kmeans/"+tc.name, func(t *testing.T) {
			dt := dataTableFromRows(tc.rows)
			got, err := stats.KMeans(dt, tc.k, stats.KMeansOptions{NStart: tc.nstart, IterMax: tc.itermax, Seed: &tc.seed})
			if err != nil {
				t.Fatalf("KMeans error: %v", err)
			}
			payload := map[string]any{"rows": tc.rows, "k": tc.k, "nstart": tc.nstart, "itermax": tc.itermax, "seed": tc.seed}
			rb := runRBaseline(t, "kmeans", payload)
			pb := runPythonBaseline(t, "kmeans", payload)
			assertIntSliceEqualToBoth(t, "cluster", got.Cluster, baselineIntSlice(t, rb, "cluster"), baselineIntSlice(t, pb, "cluster"))
			assertSliceCloseToBoth(t, "size", intsToFloat64(got.Size), intsToFloat64(baselineIntSlice(t, rb, "size")), intsToFloat64(baselineIntSlice(t, pb, "size")), 1e-10)
			assertSliceCloseToBoth(t, "withinss", got.WithinSS, baselineFloatSlice(t, rb, "withinss"), baselineFloatSlice(t, pb, "withinss"), 1e-10)
			assertCloseToBoth(t, "totss", got.TotSS, baselineFloat(t, rb, "totss"), baselineFloat(t, pb, "totss"), 1e-10)
			assertCloseToBoth(t, "tot.withinss", got.TotWithinSS, baselineFloat(t, rb, "totwithinss"), baselineFloat(t, pb, "totwithinss"), 1e-10)
			assertCloseToBoth(t, "betweenss", got.BetweenSS, baselineFloat(t, rb, "betweenss"), baselineFloat(t, pb, "betweenss"), 1e-10)
			assertCloseToBoth(t, "iter", float64(got.Iter), baselineFloat(t, rb, "iter"), baselineFloat(t, pb, "iter"), 1e-10)
			assertCloseToBoth(t, "ifault", float64(got.IFault), baselineFloat(t, rb, "ifault"), baselineFloat(t, pb, "ifault"), 1e-10)
			assertMatrixCloseToBoth(t, "centers", tableToFloatMatrix(got.Centers.(*insyra.DataTable)), baselineFloatMatrix(t, rb, "centers"), baselineFloatMatrix(t, pb, "centers"), 1e-10)
		})
	}

	for _, tc := range generatedHierarchicalParityCases() {
		t.Run("hclust/"+tc.name, func(t *testing.T) {
			dt := dataTableFromRows(tc.rows)
			got, err := stats.HierarchicalAgglomerative(dt, tc.method)
			if err != nil {
				t.Fatalf("HierarchicalAgglomerative error: %v", err)
			}
			byK, err := stats.CutTreeByK(got, tc.k)
			if err != nil {
				t.Fatalf("CutTreeByK error: %v", err)
			}
			byH, err := stats.CutTreeByHeight(got, tc.h)
			if err != nil {
				t.Fatalf("CutTreeByHeight error: %v", err)
			}
			payload := map[string]any{"rows": tc.rows, "method": string(tc.method), "k": tc.k, "h": tc.h}
			rb := runRBaseline(t, "hclust", payload)
			pb := runPythonBaseline(t, "hclust", payload)
			assertIntMatrixEqualToBoth(t, "merge", got.Merge, baselineIntMatrix(t, rb, "merge"), baselineIntMatrix(t, pb, "merge"))
			assertSliceCloseToBoth(t, "height", got.Height, baselineFloatSlice(t, rb, "height"), baselineFloatSlice(t, pb, "height"), 1e-10)
			assertIntSliceEqualToBoth(t, "order", got.Order, baselineIntSlice(t, rb, "order"), baselineIntSlice(t, pb, "order"))
			assertStringSliceEqualToBoth(t, "labels", got.Labels, baselineStringSlice(t, rb, "labels"), baselineStringSlice(t, pb, "labels"))
			assertIntSliceEqualToBoth(t, "cut_k", byK, baselineIntSlice(t, rb, "cut_k"), baselineIntSlice(t, pb, "cut_k"))
			assertIntSliceEqualToBoth(t, "cut_h", byH, baselineIntSlice(t, rb, "cut_h"), baselineIntSlice(t, pb, "cut_h"))
		})
	}

	for _, tc := range generatedDBSCANParityCases() {
		t.Run("dbscan/"+tc.name, func(t *testing.T) {
			dt := dataTableFromRows(tc.rows)
			got, err := stats.DBSCAN(dt, tc.eps, tc.minPts)
			if err != nil {
				t.Fatalf("DBSCAN error: %v", err)
			}
			payload := map[string]any{"rows": tc.rows, "eps": tc.eps, "min_pts": tc.minPts}
			rb := runRBaseline(t, "dbscan", payload)
			pb := runPythonBaseline(t, "dbscan", payload)
			assertIntSliceEqualToBoth(t, "cluster", got.Cluster, baselineIntSlice(t, rb, "cluster"), baselineIntSlice(t, pb, "cluster"))
			assertBoolSliceEqualToBoth(t, "isseed", got.IsSeed, baselineBoolSlice(t, rb, "isseed"), baselineBoolSlice(t, pb, "isseed"))
		})
	}

	for _, tc := range generatedSilhouetteParityCases() {
		t.Run("silhouette/"+tc.name, func(t *testing.T) {
			dt := dataTableFromRows(tc.rows)
			got, err := stats.Silhouette(dt, insyra.NewDataList(intsToAny(tc.labels)...))
			if err != nil {
				t.Fatalf("Silhouette error: %v", err)
			}
			payload := map[string]any{"rows": tc.rows, "labels": tc.labels}
			rb := runRBaseline(t, "silhouette", payload)
			pb := runPythonBaseline(t, "silhouette", payload)
			assertCloseToBoth(t, "avg.width", got.AverageSilhouette, baselineFloat(t, rb, "avg_width"), baselineFloat(t, pb, "avg_width"), 1e-10)
			for i, pt := range got.Points {
				assertCloseToBoth(t, "sil.cluster", float64(pt.Cluster), baselineFloatMatrix(t, rb, "points")[i][0], baselineFloatMatrix(t, pb, "points")[i][0], 1e-10)
				assertCloseToBoth(t, "sil.neighbor", float64(pt.Neighbor), baselineFloatMatrix(t, rb, "points")[i][1], baselineFloatMatrix(t, pb, "points")[i][1], 1e-10)
				assertCloseToBoth(t, "sil.width", pt.SilWidth, baselineFloatMatrix(t, rb, "points")[i][2], baselineFloatMatrix(t, pb, "points")[i][2], 1e-10)
			}
		})
	}
}
