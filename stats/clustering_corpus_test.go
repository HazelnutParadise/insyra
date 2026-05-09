package stats_test

import "github.com/HazelnutParadise/insyra/stats"

type kmeansParityCase struct {
	name    string
	rows    [][]float64
	k       int
	nstart  int
	itermax int
	seed    int64
}

type hierarchicalParityCase struct {
	name   string
	rows   [][]float64
	method stats.AgglomerativeMethod
	k      int
	h      float64
}

type dbscanParityCase struct {
	name   string
	rows   [][]float64
	eps    float64
	minPts int
}

type silhouetteParityCase struct {
	name   string
	rows   [][]float64
	labels []int
}

func generatedKMeansParityCases() []kmeansParityCase {
	return []kmeansParityCase{
		{
			name: "generated_three_blobs_2d",
			rows: synthBlobRows(101, [][]float64{{0, 0}, {6, 1}, {12, 0}}, []int{4, 4, 4}, 0.18, 0),
			k:    3, nstart: 6, itermax: 40, seed: 101,
		},
		{
			name: "generated_four_blobs_3d",
			rows: synthBlobRows(203, [][]float64{{0, 0, 0}, {4, 5, 4}, {9, 0, 9}, {13, 5, 13}}, []int{3, 4, 3, 4}, 0.16, 0),
			k:    4, nstart: 7, itermax: 50, seed: 203,
		},
		{
			name: "generated_zero_variance_tail",
			rows: withConstantColumns(
				synthBlobRows(307, [][]float64{{0, 0}, {7, 7}, {14, 0}}, []int{5, 4, 3}, 0.15, 2),
				3.0, -2.0,
			),
			k: 3, nstart: 8, itermax: 60, seed: 307,
		},
		{
			name: "generated_unbalanced_bridge",
			rows: append(
				synthBlobRows(409, [][]float64{{0, 0}, {8, 8}, {16, 0}}, []int{6, 3, 2}, 0.11, 3),
				[]float64{4.2, 3.9},
				[]float64{4.6, 4.4},
			),
			k: 3, nstart: 8, itermax: 60, seed: 409,
		},
		{
			name: "generated_duplicate_heavy_high_dim",
			rows: withConstantColumns(
				synthBlobRows(503, [][]float64{{0, 0, 0}, {5, 6, 5}, {11, 0, 11}}, []int{5, 5, 5}, 0.12, 4),
				7.0,
			),
			k: 3, nstart: 9, itermax: 60, seed: 503,
		},
		{
			name: "generated_tight_vs_diffuse",
			rows: append(
				synthBlobRows(601, [][]float64{{0, 0}, {10, 0}}, []int{7, 7}, 0.08, 1),
				[]float64{9.4, 0.7},
				[]float64{9.7, -0.6},
			),
			k: 2, nstart: 7, itermax: 50, seed: 601,
		},
		{
			name: "generated_symmetric_square_ties",
			rows: [][]float64{
				{0, 0}, {0, 0}, {0, 2}, {0, 2},
				{2, 0}, {2, 0}, {2, 2}, {2, 2},
				{1, 1}, {1, 1}, {1, 1.1},
			},
			k: 3, nstart: 10, itermax: 80, seed: 607,
		},
		{
			name: "generated_init_order_pressure",
			rows: append(
				synthBlobRows(613, [][]float64{{0, 0}, {5, 5}, {10, 0}}, []int{4, 4, 4}, 0.05, 0),
				[]float64{4.9, 4.8},
				[]float64{5.1, 5.2},
				[]float64{2.6, 2.4},
				[]float64{7.4, 2.6},
			),
			k: 3, nstart: 12, itermax: 90, seed: 613,
		},
	}
}

func generatedHierarchicalParityCases() []hierarchicalParityCase {
	return []hierarchicalParityCase{
		{
			name:   "generated_complete_ladder",
			rows:   synthLadderRows(701, 6, 1.6, 0.25),
			method: stats.AggloComplete, k: 3, h: 2.9,
		},
		{
			name:   "generated_single_bridge",
			rows:   [][]float64{{0, 0}, {0.2, 0.1}, {5, 5}, {5.2, 5.1}, {9.9, 10}, {10.1, 10.2}, {7.5, 7.4}},
			method: stats.AggloSingle, k: 2, h: 2.0,
		},
		{
			name:   "generated_average_rectangles",
			rows:   [][]float64{{0, 0}, {0, 1}, {3, 3}, {3, 4}, {8, 8}, {8, 9}, {12, 1}},
			method: stats.AggloAverage, k: 4, h: 2.5,
		},
		{
			name:   "generated_ward_d2_blobs",
			rows:   synthBlobRows(809, [][]float64{{0, 0}, {4, 4}, {9, 0}, {13, 4}}, []int{3, 3, 3, 3}, 0.09, 0),
			method: stats.AggloWardD2, k: 4, h: 3.0,
		},
		{
			name:   "generated_mcquitty_offset",
			rows:   [][]float64{{1, 1}, {1.1, 2.1}, {5, 5}, {5.1, 6.2}, {9, 1}, {9.2, 2.3}},
			method: stats.AggloMcQuitty, k: 3, h: 2.4,
		},
		{
			name:   "generated_centroid_triangle",
			rows:   [][]float64{{0, 0}, {0.2, 1.0}, {4.2, 4.0}, {5.0, 4.1}, {9.5, 0.2}, {10.1, 0.8}},
			method: stats.AggloCentroid, k: 3, h: 2.4,
		},
		{
			name:   "generated_median_triangle",
			rows:   [][]float64{{0, 0}, {1, 0.2}, {4, 4}, {5, 4.1}, {8.8, 8.7}, {9.1, 9.4}},
			method: stats.AggloMedian, k: 3, h: 2.2,
		},
		{
			name:   "generated_complete_equal_distance_grid",
			rows:   [][]float64{{0, 0}, {0, 2}, {2, 0}, {2, 2}, {6, 6}, {6, 8}, {8, 6}, {8, 8}},
			method: stats.AggloComplete, k: 2, h: 4.1,
		},
		{
			name:   "generated_average_chain_ties",
			rows:   [][]float64{{0, 0}, {0.5, 0.5}, {1, 1}, {4, 4}, {4.5, 4.5}, {8, 8}, {8.5, 8.5}},
			method: stats.AggloAverage, k: 3, h: 2.6,
		},
		{
			name:   "generated_ward_d_rectangles",
			rows:   [][]float64{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {6, 6}, {6, 7}, {7, 6}, {7, 7}},
			method: stats.AggloWardD, k: 2, h: 8.5,
		},
	}
}

func generatedDBSCANParityCases() []dbscanParityCase {
	return []dbscanParityCase{
		{
			name:   "generated_two_dense_islands",
			rows:   [][]float64{{0, 0}, {0.05, 0.1}, {0.1, 0}, {4, 4}, {4.1, 4}, {4, 4.1}, {9, 9}},
			eps:    0.18,
			minPts: 3,
		},
		{
			name:   "generated_duplicate_dense_core",
			rows:   [][]float64{{0, 0}, {0, 0}, {0, 0}, {0.1, 0}, {0, 0.1}, {2, 2}, {5, 5}},
			eps:    0.16,
			minPts: 4,
		},
		{
			name:   "generated_three_clusters_and_noise",
			rows:   [][]float64{{0, 0}, {0.1, 0}, {0, 0.1}, {5, 5}, {5.1, 5}, {5, 5.1}, {10, 0}, {10.1, 0}, {10, 0.1}, {20, 20}},
			eps:    0.16,
			minPts: 3,
		},
		{
			name:   "generated_border_chain",
			rows:   [][]float64{{0, 0}, {0.1, 0}, {0.2, 0}, {1.0, 0}, {1.1, 0}, {1.2, 0}, {3.5, 3.5}},
			eps:    0.16,
			minPts: 3,
		},
		{
			name:   "generated_two_cores_with_shared_border",
			rows:   [][]float64{{0, 0}, {0.06, 0}, {0, 0.06}, {0.12, 0.02}, {1.0, 1.0}, {1.06, 1.0}, {1.0, 1.06}, {0.56, 0.55}},
			eps:    0.13,
			minPts: 3,
		},
	}
}

func generatedSilhouetteParityCases() []silhouetteParityCase {
	return []silhouetteParityCase{
		{
			name:   "generated_three_imbalanced",
			rows:   [][]float64{{0, 0}, {0, 1}, {0, 2}, {5, 5}, {9, 9}, {9, 10}, {9, 11}},
			labels: []int{1, 1, 1, 2, 3, 3, 3},
		},
		{
			name:   "generated_four_clusters",
			rows:   [][]float64{{0, 0}, {0, 1}, {4, 4}, {4, 5}, {8, 0}, {8, 1}, {12, 4}, {12, 5}},
			labels: []int{1, 1, 2, 2, 3, 3, 4, 4},
		},
		{
			name:   "generated_duplicate_members",
			rows:   [][]float64{{0, 0}, {0, 0}, {3, 3}, {3, 4}, {7, 7}, {7, 7}, {7, 8}},
			labels: []int{1, 1, 2, 2, 3, 3, 3},
		},
		{
			name:   "generated_asymmetric_spacing",
			rows:   [][]float64{{0, 0}, {0.2, 0.9}, {2.5, 2.5}, {5, 5}, {5.2, 6}, {10, 10}, {10.2, 10.9}},
			labels: []int{1, 1, 2, 3, 3, 4, 4},
		},
		{
			name:   "generated_bridge_neighbor_switch",
			rows:   [][]float64{{0, 0}, {0.1, 0.2}, {3, 3}, {3.1, 3.2}, {6, 0}, {6.1, 0.2}, {4.6, 1.4}},
			labels: []int{1, 1, 2, 2, 3, 3, 2},
		},
	}
}

func synthBlobRows(seed int64, centers [][]float64, sizes []int, jitter float64, duplicateEvery int) [][]float64 {
	gen := newCorpusLCG(seed)
	rows := make([][]float64, 0)
	for i, center := range centers {
		for j := 0; j < sizes[i]; j++ {
			row := make([]float64, len(center))
			for d, base := range center {
				row[d] = round4(base + gen.nextSigned(jitter))
			}
			rows = append(rows, row)
			if duplicateEvery > 0 && j%duplicateEvery == 0 {
				rows = append(rows, append([]float64(nil), row...))
			}
		}
	}
	return rows
}

func synthLadderRows(seed int64, n int, spacing, wobble float64) [][]float64 {
	gen := newCorpusLCG(seed)
	rows := make([][]float64, 0, n)
	for i := 0; i < n; i++ {
		rows = append(rows, []float64{
			round4(float64(i) * spacing),
			round4(float64(i%2) + gen.nextSigned(wobble)),
		})
	}
	return rows
}

func withConstantColumns(rows [][]float64, values ...float64) [][]float64 {
	out := make([][]float64, len(rows))
	for i, row := range rows {
		next := append([]float64(nil), row...)
		next = append(next, values...)
		out[i] = next
	}
	return out
}

func round4(v float64) float64 {
	const scale = 10000
	if v >= 0 {
		return float64(int(v*scale+0.5)) / scale
	}
	return float64(int(v*scale-0.5)) / scale
}

type corpusLCG struct {
	state uint64
}

func newCorpusLCG(seed int64) *corpusLCG {
	return &corpusLCG{state: uint64(seed)}
}

func (g *corpusLCG) next() uint64 {
	g.state = (1664525*g.state + 1013904223) % (1 << 32)
	return g.state
}

func (g *corpusLCG) nextSigned(scale float64) float64 {
	u := float64(g.next()%10001) / 10000.0
	return (u*2 - 1) * scale
}
