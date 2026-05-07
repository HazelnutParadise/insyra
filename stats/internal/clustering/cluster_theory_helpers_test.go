package clustering

import "testing"

type internalKMeansCase struct {
	name    string
	rows    [][]float64
	k       int
	nstart  int
	itermax int
	seed    int64
}

type lanceWilliamsCase struct {
	name   string
	method string
	a      *clusterNode
	b      *clusterNode
	other  *clusterNode
	dik    float64
	djk    float64
	dij    float64
	want   float64
}

type reducibleHierarchyCase struct {
	name   string
	rows   [][]float64
	method string
}

func generatedInternalKMeansCases() []internalKMeansCase {
	return []internalKMeansCase{
		{
			name: "three_blobs_low_dim",
			rows: [][]float64{{0, 0}, {0.2, 0.1}, {5, 5}, {5.1, 5.2}, {10, 0}, {10.2, 0.1}},
			k:    3, nstart: 5, itermax: 40, seed: 101,
		},
		{
			name: "zero_variance_extra_column",
			rows: [][]float64{{0, 0, 7}, {0, 1, 7}, {6, 6, 7}, {6, 7, 7}, {12, 0, 7}, {12, 1, 7}},
			k:    3, nstart: 5, itermax: 40, seed: 203,
		},
		{
			name: "duplicate_heavy",
			rows: [][]float64{{0, 0}, {0, 0}, {0.1, 0}, {4, 4}, {4, 4}, {4.1, 4}, {9, 0}},
			k:    3, nstart: 6, itermax: 50, seed: 307,
		},
	}
}

func assertKMeansSSEIdentities(t *testing.T, rows [][]float64, got *KMeansResult) {
	t.Helper()
	if len(got.Cluster) != len(rows) {
		t.Fatalf("cluster assignment length mismatch: got=%d want=%d", len(got.Cluster), len(rows))
	}
	if len(got.Centers) != len(got.Size) || len(got.Centers) != len(got.WithinSS) {
		t.Fatalf("result dimension mismatch: centers=%d size=%d within=%d", len(got.Centers), len(got.Size), len(got.WithinSS))
	}
	totalN := 0
	for _, n := range got.Size {
		totalN += n
	}
	if totalN != len(rows) {
		t.Fatalf("cluster sizes do not sum to row count: got=%d want=%d", totalN, len(rows))
	}
	p := len(rows[0])
	sum := make([]float64, p)
	for _, row := range rows {
		for j := 0; j < p; j++ {
			sum[j] += row[j]
		}
	}
	overallMean := make([]float64, p)
	for j := 0; j < p; j++ {
		overallMean[j] = sum[j] / float64(len(rows))
	}

	clusterCounts := make([]int, len(got.Centers))
	clusterSums := make([][]float64, len(got.Centers))
	for i := range clusterSums {
		clusterSums[i] = make([]float64, p)
	}
	for i, row := range rows {
		cluster := got.Cluster[i] - 1
		if cluster < 0 || cluster >= len(got.Centers) {
			t.Fatalf("invalid cluster label %d at row %d", got.Cluster[i], i)
		}
		clusterCounts[cluster]++
		for j := 0; j < p; j++ {
			clusterSums[cluster][j] += row[j]
		}
	}

	for c := range got.Centers {
		if clusterCounts[c] != got.Size[c] {
			t.Fatalf("cluster %d size mismatch: got=%d want=%d", c, got.Size[c], clusterCounts[c])
		}
		for j := 0; j < p; j++ {
			want := clusterSums[c][j] / float64(clusterCounts[c])
			if !almostEqual(got.Centers[c][j], want) {
				t.Fatalf("cluster %d center[%d]=%v want=%v", c, j, got.Centers[c][j], want)
			}
		}
	}

	within := make([]float64, len(got.Centers))
	totss := 0.0
	for i, row := range rows {
		cluster := got.Cluster[i] - 1
		within[cluster] += squaredEuclidean(row, got.Centers[cluster])
		totss += squaredEuclidean(row, overallMean)
	}
	totWithin := 0.0
	for c := range within {
		if !almostEqual(within[c], got.WithinSS[c]) {
			t.Fatalf("cluster %d withinss=%v want=%v", c, got.WithinSS[c], within[c])
		}
		totWithin += within[c]
	}
	if !almostEqual(got.TotWithinSS, totWithin) {
		t.Fatalf("tot.withinss=%v want=%v", got.TotWithinSS, totWithin)
	}
	if !almostEqual(got.TotSS, totss) {
		t.Fatalf("totss=%v want=%v", got.TotSS, totss)
	}
	if !almostEqual(got.BetweenSS+got.TotWithinSS, got.TotSS) {
		t.Fatalf("between + within != total: between=%v within=%v total=%v", got.BetweenSS, got.TotWithinSS, got.TotSS)
	}
}

func lanceWilliamsFormulaCases() []lanceWilliamsCase {
	a := &clusterNode{size: 2}
	b := &clusterNode{size: 3}
	other := &clusterNode{size: 4}
	return []lanceWilliamsCase{
		{name: "single", method: "single", a: a, b: b, other: other, dik: 4, djk: 10, dij: 5, want: 4},
		{name: "complete", method: "complete", a: a, b: b, other: other, dik: 4, djk: 10, dij: 5, want: 10},
		{name: "average", method: "average", a: a, b: b, other: other, dik: 4, djk: 10, dij: 5, want: 7.6},
		{name: "mcquitty", method: "mcquitty", a: a, b: b, other: other, dik: 4, djk: 10, dij: 5, want: 7},
		{name: "median", method: "median", a: a, b: b, other: other, dik: 4, djk: 10, dij: 5, want: 5.75},
		{name: "centroid", method: "centroid", a: a, b: b, other: other, dik: 4, djk: 10, dij: 5, want: 6.4},
		{name: "ward_d", method: "ward.d", a: a, b: b, other: other, dik: 9, djk: 25, dij: 16, want: 165.0 / 9.0},
		{name: "ward_d2", method: "ward.d2", a: a, b: b, other: other, dik: 9, djk: 25, dij: 16, want: 165.0 / 9.0},
	}
}

func reducibleHierarchyCases() []reducibleHierarchyCase {
	return []reducibleHierarchyCase{
		{
			name:   "single_chain",
			rows:   [][]float64{{0, 0}, {0.5, 0.5}, {1, 1}, {4, 4}, {8, 8}},
			method: "single",
		},
		{
			name:   "complete_blocks",
			rows:   [][]float64{{0, 0}, {0, 1}, {1, 0}, {5, 5}, {5, 6}, {6, 5}},
			method: "complete",
		},
		{
			name:   "average_rectangles",
			rows:   [][]float64{{0, 0}, {0, 2}, {2, 0}, {2, 2}, {7, 7}, {7, 9}, {9, 7}, {9, 9}},
			method: "average",
		},
		{
			name:   "ward_d_blobs",
			rows:   [][]float64{{0, 0}, {0, 1}, {1, 0}, {8, 8}, {8, 9}, {9, 8}},
			method: "ward.d",
		},
		{
			name:   "ward_d2_blobs",
			rows:   [][]float64{{0, 0}, {0, 1}, {1, 0}, {8, 8}, {8, 9}, {9, 8}},
			method: "ward.d2",
		},
	}
}
