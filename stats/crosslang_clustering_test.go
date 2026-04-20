package stats_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestCrossLangClusteringMethods(t *testing.T) {
	requireCrossLangTools(t)

	t.Run("kmeans", func(t *testing.T) {
		cases := []struct {
			name    string
			rows    [][]float64
			k       int
			nstart  int
			itermax int
			seed    int64
		}{
			{
				name: "two_blobs",
				rows: [][]float64{{0, 0}, {0, 1}, {1, 0}, {10, 10}, {10, 11}, {11, 10}},
				k:    2, nstart: 3, itermax: 20, seed: 7,
			},
			{
				name: "three_blobs_with_ties",
				rows: [][]float64{{0, 0}, {0, 0}, {5, 5}, {5, 6}, {9, 9}, {9, 10}},
				k:    3, nstart: 4, itermax: 25, seed: 11,
			},
			{
				name: "high_dim_zero_variance",
				rows: [][]float64{
					{0, 0, 1, 7}, {0, 1, 1, 7}, {1, 0, 1, 7},
					{10, 10, 1, 7}, {10, 11, 1, 7}, {11, 10, 1, 7},
					{20, 20, 1, 7}, {20, 21, 1, 7}, {21, 20, 1, 7},
				},
				k: 3, nstart: 5, itermax: 40, seed: 17,
			},
			{
				name: "unbalanced_duplicate_heavy",
				rows: [][]float64{
					{0, 0}, {0, 0}, {0, 0.1}, {0.2, 0}, {0.1, 0.1},
					{8, 8}, {8, 8}, {8.1, 8}, {8, 8.1},
					{20, 20},
				},
				k: 3, nstart: 6, itermax: 50, seed: 29,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
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
	})

	t.Run("hierarchical_and_cutree", func(t *testing.T) {
		cases := []struct {
			name   string
			rows   [][]float64
			method stats.AgglomerativeMethod
			k      int
			h      float64
		}{
			{name: "complete_case", rows: [][]float64{{0, 0}, {0, 1}, {10, 10}, {10, 11}}, method: stats.AggloComplete, k: 2, h: 5},
			{name: "average_case", rows: [][]float64{{0, 0}, {1, 0}, {4, 4}, {5, 4}, {9, 9}}, method: stats.AggloAverage, k: 3, h: 2.5},
			{name: "single_ties_case", rows: [][]float64{{0, 0}, {0, 2}, {2, 0}, {2, 2}, {10, 10}}, method: stats.AggloSingle, k: 2, h: 3},
			{name: "ward_d2_case", rows: [][]float64{{0, 0}, {0, 1}, {5, 5}, {5, 6}, {9, 9}, {9, 10}}, method: stats.AggloWardD2, k: 3, h: 6},
			{name: "mcquitty_case", rows: [][]float64{{1, 1}, {1, 2}, {4, 4}, {4, 5}, {8, 8}, {9, 8}}, method: stats.AggloMcQuitty, k: 3, h: 2.1},
			{name: "centroid_case", rows: [][]float64{{0, 0}, {0, 1}, {3, 3}, {4, 3}, {8, 8}}, method: stats.AggloCentroid, k: 3, h: 2.0},
			{name: "median_case", rows: [][]float64{{0, 0}, {1, 0}, {4, 4}, {4, 5}, {9, 9}}, method: stats.AggloMedian, k: 3, h: 2.0},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
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
	})

	t.Run("dbscan", func(t *testing.T) {
		cases := []struct {
			name   string
			rows   [][]float64
			eps    float64
			minPts int
		}{
			{name: "noise_case", rows: [][]float64{{0, 0}, {0.1, 0}, {0, 0.1}, {8, 8}}, eps: 0.25, minPts: 3},
			{name: "two_clusters", rows: [][]float64{{0, 0}, {0.2, 0.1}, {5, 5}, {5.2, 5.1}, {10, 10}}, eps: 0.35, minPts: 2},
			{name: "all_noise", rows: [][]float64{{0, 0}, {3, 3}, {6, 6}, {9, 9}}, eps: 0.5, minPts: 2},
			{name: "duplicate_core_and_border", rows: [][]float64{{0, 0}, {0, 0}, {0.1, 0}, {0, 0.1}, {1.2, 1.2}, {1.3, 1.2}, {3.5, 3.5}}, eps: 0.18, minPts: 3},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
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
	})

	t.Run("silhouette", func(t *testing.T) {
		cases := []struct {
			name   string
			rows   [][]float64
			labels []int
		}{
			{name: "two_clusters", rows: [][]float64{{0, 0}, {0, 1}, {10, 10}, {10, 11}}, labels: []int{1, 1, 2, 2}},
			{name: "three_clusters", rows: [][]float64{{0, 0}, {0, 1}, {5, 5}, {5, 6}, {10, 10}, {10, 11}}, labels: []int{1, 1, 2, 2, 3, 3}},
			{name: "imbalanced_clusters", rows: [][]float64{{0, 0}, {0, 1}, {0, 2}, {5, 5}, {10, 10}, {10, 11}, {10, 12}}, labels: []int{1, 1, 1, 2, 3, 3, 3}},
			{name: "duplicate_points", rows: [][]float64{{0, 0}, {0, 0}, {5, 5}, {5, 5}, {9, 9}, {9, 10}}, labels: []int{1, 1, 2, 2, 3, 3}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
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
	})
}

func intsToAny(xs []int) []any {
	out := make([]any, len(xs))
	for i, v := range xs {
		out[i] = v
	}
	return out
}

func intsToFloat64(xs []int) []float64 {
	out := make([]float64, len(xs))
	for i, v := range xs {
		out[i] = float64(v)
	}
	return out
}
