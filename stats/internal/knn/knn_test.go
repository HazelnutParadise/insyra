package knn

import (
	"fmt"
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

func TestAutoProbeHonorsExplicitAlgorithms(t *testing.T) {
	train := calibrationRows(128, 16, 101, false)
	test := calibrationRows(3, 16, 102, false)
	for _, tc := range []struct {
		name string
		algo Algorithm
		want any
	}{
		{name: "brute", algo: BruteForceAlgorithm, want: &bruteSearcher{}},
		{name: "kd_tree", algo: KDTreeAlgorithm, want: &kdTreeSearcher{}},
		{name: "ball_tree", algo: BallTreeAlgorithm, want: &ballTreeSearcher{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := newSearcher(train, test, 5, Options{Algorithm: tc.algo, LeafSize: 16})
			if err != nil {
				t.Fatal(err)
			}
			switch tc.want.(type) {
			case *bruteSearcher:
				if _, ok := got.(*bruteSearcher); !ok {
					t.Fatalf("explicit brute returned %T", got)
				}
			case *kdTreeSearcher:
				if _, ok := got.(*kdTreeSearcher); !ok {
					t.Fatalf("explicit kd-tree returned %T", got)
				}
			case *ballTreeSearcher:
				if _, ok := got.(*ballTreeSearcher); !ok {
					t.Fatalf("explicit ball-tree returned %T", got)
				}
			}
		})
	}
}

func TestAutoProbeSelectionIsDeterministicAndUsesBothSides(t *testing.T) {
	for _, tc := range []struct {
		name      string
		clustered bool
		wantTree  bool
	}{
		{name: "unstructured_discards_tree", clustered: false, wantTree: false},
		{name: "clustered_keeps_tree", clustered: true, wantTree: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			train := calibrationRows(256, 16, 201, tc.clustered)
			test := calibrationRows(17, 16, 202, tc.clustered)
			first, err := newSearcher(train, test, 5, Options{Algorithm: AutoAlgorithm, LeafSize: 16})
			if err != nil {
				t.Fatal(err)
			}
			second, err := newSearcher(train, test, 5, Options{Algorithm: AutoAlgorithm, LeafSize: 16})
			if err != nil {
				t.Fatal(err)
			}
			if fmt.Sprintf("%T", first) != fmt.Sprintf("%T", second) {
				t.Fatalf("selection changed: first=%T second=%T", first, second)
			}
			_, isTree := first.(*ballTreeSearcher)
			if isTree != tc.wantTree {
				t.Fatalf("selection type=%T want tree=%v", first, tc.wantTree)
			}
		})
	}
}

func TestAutoProbeSmallBatchesAndResultParity(t *testing.T) {
	for _, clustered := range []bool{false, true} {
		train := calibrationRows(256, 16, 301, clustered)
		test := calibrationRows(3, 16, 302, clustered)
		brute, err := Neighbors(train, test, 5, Options{Algorithm: BruteForceAlgorithm})
		if err != nil {
			t.Fatal(err)
		}
		auto, err := Neighbors(train, test, 5, Options{Algorithm: AutoAlgorithm})
		if err != nil {
			t.Fatal(err)
		}
		ball, err := Neighbors(train, test, 5, Options{Algorithm: BallTreeAlgorithm})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(auto.Indices, brute.Indices) || !reflect.DeepEqual(auto.Distances, brute.Distances) {
			t.Fatalf("auto differs from brute for clustered=%v", clustered)
		}
		if !reflect.DeepEqual(ball.Indices, brute.Indices) || !reflect.DeepEqual(ball.Distances, brute.Distances) {
			t.Fatalf("ball differs from brute for clustered=%v", clustered)
		}
	}
}
