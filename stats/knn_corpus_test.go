package stats_test

import "github.com/HazelnutParadise/insyra/stats"

type knnClassifyParityCase struct {
	name      string
	trainRows [][]float64
	testRows  [][]float64
	labels    []string
	k         int
	weighting stats.KNNWeighting
	algorithm stats.KNNAlgorithm
}

type knnRegressionParityCase struct {
	name      string
	trainRows [][]float64
	testRows  [][]float64
	targets   []float64
	k         int
	weighting stats.KNNWeighting
	algorithm stats.KNNAlgorithm
}

type knnNeighborParityCase struct {
	name      string
	trainRows [][]float64
	testRows  [][]float64
	k         int
	algorithm stats.KNNAlgorithm
}

func generatedKNNClassifyParityCases() []knnClassifyParityCase {
	return []knnClassifyParityCase{
		{
			name: "high_dim_imbalanced_clusters",
			trainRows: knnLiftRows([][]float64{
				{0.0, 0.0}, {0.2, -0.1}, {-0.2, 0.1}, {0.1, 0.2}, {-0.1, -0.2}, {0.15, -0.15}, {0.0, 0.0}, {0.05, -0.05},
				{5.0, 5.0}, {5.2, 4.9}, {4.8, 5.1},
				{10.0, 0.0}, {10.2, 0.1}, {9.8, -0.2},
			}, 8),
			testRows: knnLiftRows([][]float64{
				{0.08, 0.04}, {5.1, 5.05}, {10.1, 0.02}, {2.6, 2.4},
			}, 8),
			labels: []string{
				"red", "red", "red", "red", "red", "red", "red", "red",
				"blue", "blue", "blue",
				"green", "green", "green",
			},
			k:         5,
			weighting: stats.KNNDistanceWeighting,
			algorithm: stats.KNNBallTree,
		},
		{
			name: "duplicate_exact_match_pressure",
			trainRows: [][]float64{
				{1, 1}, {1, 1}, {1, 1}, {1, 1},
				{3, 3}, {3, 3}, {3.2, 3.1}, {4.5, 4.4},
			},
			testRows: [][]float64{
				{1, 1}, {3.05, 3.0}, {1.2, 1.1},
			},
			labels:    []string{"red", "red", "red", "blue", "blue", "blue", "blue", "blue"},
			k:         4,
			weighting: stats.KNNDistanceWeighting,
			algorithm: stats.KNNKDTree,
		},
		{
			name: "tie_heavy_square",
			trainRows: [][]float64{
				{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {-2, 0}, {2, 0},
			},
			testRows: [][]float64{
				{0, 0}, {0.1, 0},
			},
			labels:    []string{"alpha", "beta", "alpha", "beta", "alpha", "beta"},
			k:         4,
			weighting: stats.KNNUniformWeighting,
			algorithm: stats.KNNBruteForce,
		},
		{
			name: "tie_mean_distance_break",
			trainRows: [][]float64{
				{1, 0}, {4, 0}, {0, 2}, {0, -2}, {8, 8},
			},
			testRows: [][]float64{
				{0, 0}, {0.2, 0.1},
			},
			labels:    []string{"left", "left", "right", "right", "left"},
			k:         4,
			weighting: stats.KNNUniformWeighting,
			algorithm: stats.KNNBruteForce,
		},
		{
			name: "duplicate_bridge_with_auto_backend",
			trainRows: [][]float64{
				{0, 0}, {0, 0}, {0.1, 0.1}, {0.2, -0.1}, {6, 6}, {6.1, 6.1}, {6.2, 5.9}, {3.0, 3.0}, {3.1, 3.0},
			},
			testRows: [][]float64{
				{0.05, 0.02}, {6.05, 6.0}, {3.05, 3.02},
			},
			labels:    []string{"major", "major", "major", "major", "minor", "minor", "minor", "bridge", "bridge"},
			k:         3,
			weighting: stats.KNNDistanceWeighting,
			algorithm: stats.KNNAuto,
		},
	}
}

func generatedKNNRegressionParityCases() []knnRegressionParityCase {
	return []knnRegressionParityCase{
		{
			name: "high_dim_cluster_targets",
			trainRows: knnLiftRows([][]float64{
				{0.0, 0.0}, {0.2, -0.1}, {-0.2, 0.1}, {0.1, 0.2}, {-0.1, -0.2}, {0.0, 0.0},
				{5.0, 5.0}, {5.2, 4.9}, {4.8, 5.1},
				{10.0, 0.0}, {10.2, 0.1}, {9.8, -0.2},
			}, 9),
			testRows: knnLiftRows([][]float64{
				{0.1, 0.0}, {5.05, 5.0}, {9.95, -0.05}, {2.6, 2.5},
			}, 9),
			targets:   []float64{1, 1.2, 0.8, 1.1, 0.9, 1.0, 10, 10.2, 9.8, 20, 19.7, 20.3},
			k:         4,
			weighting: stats.KNNDistanceWeighting,
			algorithm: stats.KNNBallTree,
		},
		{
			name: "duplicate_exact_match_targets",
			trainRows: [][]float64{
				{1, 1}, {1, 1}, {1, 1}, {2, 2}, {2.1, 2.0}, {4, 4},
			},
			testRows: [][]float64{
				{1, 1}, {2.05, 2.0}, {1.1, 1.0},
			},
			targets:   []float64{4, 6, 8, 20, 22, 40},
			k:         3,
			weighting: stats.KNNDistanceWeighting,
			algorithm: stats.KNNKDTree,
		},
		{
			name: "tie_heavy_cross_average",
			trainRows: [][]float64{
				{-1, 0}, {1, 0}, {0, -1}, {0, 1},
			},
			testRows: [][]float64{
				{0, 0}, {0.2, 0.1},
			},
			targets:   []float64{0, 10, 20, 30},
			k:         4,
			weighting: stats.KNNUniformWeighting,
			algorithm: stats.KNNBruteForce,
		},
		{
			name: "imbalanced_bridge_targets",
			trainRows: [][]float64{
				{0, 0}, {0.1, 0}, {-0.1, 0.1}, {0, -0.1}, {0.2, 0.1},
				{3, 3}, {3.1, 3}, {3, 3.1},
				{6, 0}, {6.2, 0.1},
			},
			testRows: [][]float64{
				{2.9, 2.95}, {0.05, 0.02}, {4.8, 0.8},
			},
			targets:   []float64{1, 1.2, 0.9, 1.1, 1.0, 12, 12.5, 11.8, 30, 31},
			k:         5,
			weighting: stats.KNNDistanceWeighting,
			algorithm: stats.KNNAuto,
		},
		{
			name: "high_dim_duplicate_negative_targets",
			trainRows: knnLiftRows([][]float64{
				{-4, -4}, {-4.1, -4.0}, {-3.9, -4.2}, {-4, -4},
				{4, 4}, {4.2, 4.1}, {3.8, 3.9},
			}, 10),
			testRows: knnLiftRows([][]float64{
				{-4, -4}, {4.1, 4.0}, {0, 0},
			}, 10),
			targets:   []float64{-8, -7.5, -8.2, -7.8, 9, 9.5, 8.7},
			k:         3,
			weighting: stats.KNNDistanceWeighting,
			algorithm: stats.KNNBallTree,
		},
	}
}

func generatedKNNNeighborParityCases() []knnNeighborParityCase {
	return []knnNeighborParityCase{
		{
			name: "high_dim_duplicate_neighbors",
			trainRows: knnLiftRows([][]float64{
				{0, 0}, {0, 0}, {0.1, 0.1}, {0.2, -0.1},
				{4, 4}, {4.1, 4.1}, {4.2, 3.9},
				{9, 0}, {9.1, 0.1},
			}, 9),
			testRows: knnLiftRows([][]float64{
				{0, 0}, {4.05, 4.0}, {8.9, 0.0},
			}, 9),
			k:         4,
			algorithm: stats.KNNBallTree,
		},
		{
			name: "tie_heavy_cross_indices",
			trainRows: [][]float64{
				{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {-2, 0}, {2, 0},
			},
			testRows: [][]float64{
				{0, 0}, {0.1, 0},
			},
			k:         4,
			algorithm: stats.KNNBruteForce,
		},
		{
			name: "imbalanced_bridge_neighbors",
			trainRows: [][]float64{
				{0, 0}, {0.1, 0}, {-0.1, 0.1}, {0, -0.1}, {0.2, 0.1},
				{3, 3}, {3.1, 3}, {3, 3.1},
				{6, 0}, {6.2, 0.1},
			},
			testRows: [][]float64{
				{2.95, 3.0}, {0.05, 0.01}, {5.8, 0.1},
			},
			k:         5,
			algorithm: stats.KNNKDTree,
		},
		{
			name: "zero_variance_tail_neighbors",
			trainRows: withConstantColumns([][]float64{
				{0, 0}, {0, 0.2}, {0.2, 0}, {5, 5}, {5.1, 5.1}, {10, 0}, {10.1, 0.1},
			}, 3, 3, 3, 3),
			testRows: withConstantColumns([][]float64{
				{0.05, 0.02}, {5.05, 5.0}, {10.05, 0.02},
			}, 3, 3, 3, 3),
			k:         3,
			algorithm: stats.KNNAuto,
		},
		{
			name: "duplicate_border_ordering",
			trainRows: [][]float64{
				{1, 1}, {1, 1}, {1, 1}, {1.1, 1}, {2, 2}, {2, 2}, {4, 4},
			},
			testRows: [][]float64{
				{1, 1}, {1.05, 1.0},
			},
			k:         5,
			algorithm: stats.KNNKDTree,
		},
	}
}

func knnLiftRows(rows [][]float64, extraDims int) [][]float64 {
	out := make([][]float64, len(rows))
	for i, row := range rows {
		next := append([]float64(nil), row...)
		for e := 0; e < extraDims; e++ {
			a := row[e%len(row)]
			b := row[(e+1)%len(row)]
			mix := float64((e%5)+1) * 0.15
			bias := float64((e%3)-1) * 0.07
			next = append(next, round4(a*(1-mix)+b*mix+bias))
		}
		out[i] = next
	}
	return out
}
