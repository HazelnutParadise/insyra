package knn

import (
	"math"
	"reflect"
	"testing"
)

func TestNeighborBackendsAreExactAndConsistent(t *testing.T) {
	train := [][]float64{
		{0, 0}, {0, 1}, {1, 0},
		{10, 10}, {10, 11}, {11, 10},
	}
	test := [][]float64{
		{0.1, 0.2},
		{10.1, 10.2},
	}
	cases := []struct {
		name string
		opts Options
	}{
		{name: "brute", opts: Options{Algorithm: BruteForceAlgorithm}},
		{name: "kd_tree", opts: Options{Algorithm: KDTreeAlgorithm}},
		{name: "ball_tree", opts: Options{Algorithm: BallTreeAlgorithm}},
	}

	var base *NeighborResult
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Neighbors(train, test, 2, tc.opts)
			if err != nil {
				t.Fatalf("Neighbors error: %v", err)
			}
			if base == nil {
				base = got
				return
			}
			if !reflect.DeepEqual(got.Indices, base.Indices) {
				t.Fatalf("indices mismatch: got=%v want=%v", got.Indices, base.Indices)
			}
			for i := range got.Distances {
				for j := range got.Distances[i] {
					if math.Abs(got.Distances[i][j]-base.Distances[i][j]) > 1e-12 {
						t.Fatalf("distance mismatch at row=%d col=%d: got=%v want=%v", i, j, got.Distances[i][j], base.Distances[i][j])
					}
				}
			}
		})
	}
}

func TestDistanceWeightedClassificationUsesExactMatchesOnly(t *testing.T) {
	train := [][]float64{{0, 0}, {0, 1}, {10, 10}}
	test := [][]float64{{10, 10}}
	labels := []string{"left", "left", "right"}

	got, err := Classify(train, test, labels, 3, Options{Weighting: DistanceWeighting, Algorithm: KDTreeAlgorithm})
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}
	if got.Predictions[0] != "right" {
		t.Fatalf("prediction=%v want right", got.Predictions[0])
	}
	if got.Probabilities[0][1] != 1 {
		t.Fatalf("right probability=%v want 1", got.Probabilities[0][1])
	}
}
