package knn

import (
	"math"
	"reflect"
	"testing"
)

type neighborCorpusCase struct {
	name     string
	train    [][]float64
	test     [][]float64
	k        int
	leafSize int
}

func TestGeneratedNeighborCorporaRemainExactAcrossBackends(t *testing.T) {
	for _, tc := range generatedNeighborCorpusCases() {
		t.Run(tc.name, func(t *testing.T) {
			var base *NeighborResult
			for _, algo := range []Algorithm{BruteForceAlgorithm, KDTreeAlgorithm, BallTreeAlgorithm} {
				got, err := Neighbors(tc.train, tc.test, tc.k, Options{Algorithm: algo, LeafSize: tc.leafSize})
				if err != nil {
					t.Fatalf("Neighbors(%s) error: %v", algo, err)
				}
				if base == nil {
					base = got
					continue
				}
				if !reflect.DeepEqual(got.Indices, base.Indices) {
					t.Fatalf("indices mismatch for %s: got=%v want=%v", algo, got.Indices, base.Indices)
				}
				for i := range got.Distances {
					for j := range got.Distances[i] {
						if math.Abs(got.Distances[i][j]-base.Distances[i][j]) > 1e-12 {
							t.Fatalf("distance mismatch for %s at row=%d col=%d: got=%v want=%v", algo, i, j, got.Distances[i][j], base.Distances[i][j])
						}
					}
				}
			}
		})
	}
}

func generatedNeighborCorpusCases() []neighborCorpusCase {
	return []neighborCorpusCase{
		{
			name: "high_dim_duplicate_neighbors",
			train: liftRows([][]float64{
				{0, 0}, {0, 0}, {0.1, 0.1}, {0.2, -0.1},
				{4, 4}, {4.1, 4.1}, {4.2, 3.9},
				{9, 0}, {9.1, 0.1},
			}, 9),
			test: liftRows([][]float64{
				{0, 0}, {4.05, 4.0}, {8.9, 0.0},
			}, 9),
			k:        4,
			leafSize: 2,
		},
		{
			name: "tie_heavy_cross",
			train: [][]float64{
				{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {-2, 0}, {2, 0},
			},
			test: [][]float64{
				{0, 0}, {0.1, 0},
			},
			k:        4,
			leafSize: 1,
		},
		{
			name: "imbalanced_bridge",
			train: [][]float64{
				{0, 0}, {0.1, 0}, {-0.1, 0.1}, {0, -0.1}, {0.2, 0.1},
				{3, 3}, {3.1, 3}, {3, 3.1},
				{6, 0}, {6.2, 0.1},
			},
			test: [][]float64{
				{2.95, 3.0}, {0.05, 0.01}, {5.8, 0.1},
			},
			k:        5,
			leafSize: 2,
		},
		{
			name: "duplicate_border_ordering",
			train: [][]float64{
				{1, 1}, {1, 1}, {1, 1}, {1.1, 1}, {2, 2}, {2, 2}, {4, 4},
			},
			test: [][]float64{
				{1, 1}, {1.05, 1.0},
			},
			k:        5,
			leafSize: 1,
		},
	}
}

func liftRows(rows [][]float64, extraDims int) [][]float64 {
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

func round4(v float64) float64 {
	const scale = 10000.0
	if v >= 0 {
		return math.Trunc(v*scale+0.5) / scale
	}
	return math.Trunc(v*scale-0.5) / scale
}
