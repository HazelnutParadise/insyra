package knn

import (
	"reflect"
	"testing"
)

func deviceFixture() (train, test [][]float64, labels []string, targets []float64) {
	train = [][]float64{{0, 0}, {1, 0}, {0, 1}, {5, 5}, {6, 5}, {5, 6}}
	test = [][]float64{{0.2, 0.1}, {5.2, 5.1}}
	labels = []string{"a", "a", "a", "b", "b", "b"}
	targets = []float64{1, 1, 1, 9, 9, 9}
	return
}

// The socket is consulted for auto only: a caller who named a machine gets
// that machine, whatever is registered.
func TestDeviceSocketConsultedForAutoOnly(t *testing.T) {
	train, test, labels, _ := deviceFixture()
	calls := 0
	RegisterDeviceSearcher(func(_, testRows [][]float64, k int) ([][]int, [][]float64, bool) {
		calls++
		// A deliberately wrong but well-formed answer, so its use is visible.
		indices := make([][]int, len(testRows))
		dist2 := make([][]float64, len(testRows))
		for i := range testRows {
			indices[i] = make([]int, k)
			dist2[i] = make([]float64, k)
			for j := 0; j < k; j++ {
				indices[i][j] = 3 + j // the far cluster, always
				dist2[i][j] = float64(j)
			}
		}
		return indices, dist2, true
	})
	defer RegisterDeviceSearcher(nil)

	viaAuto, err := Classify(train, test, labels, 3, Options{Algorithm: AutoAlgorithm})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("auto consulted the socket %d times, want 1", calls)
	}
	// The canned answer names the far cluster for both rows, so if the socket
	// was really used, both predictions are "b".
	if viaAuto.Predictions[0] != "b" || viaAuto.Predictions[1] != "b" {
		t.Fatalf("the socket's answer was not used: predictions %v", viaAuto.Predictions)
	}

	viaBrute, err := Classify(train, test, labels, 3, Options{Algorithm: BruteForceAlgorithm})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("an explicit algorithm consulted the socket (calls=%d)", calls)
	}
	if viaBrute.Predictions[0] != "a" || viaBrute.Predictions[1] != "b" {
		t.Fatalf("brute force gave %v", viaBrute.Predictions)
	}
}

// A declining or malformed searcher must degrade to the CPU path — the
// searcher is third-party code from stats' point of view, and its bugs must
// not become stats' panics.
func TestDeviceSocketFallsBackOnDeclineAndMalformedAnswers(t *testing.T) {
	train, test, labels, _ := deviceFixture()
	want, err := Classify(train, test, labels, 3, Options{Algorithm: BruteForceAlgorithm})
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]DeviceSearcher{
		"decline": func(_, _ [][]float64, _ int) ([][]int, [][]float64, bool) {
			return nil, nil, false
		},
		"wrong row count": func(_, _ [][]float64, k int) ([][]int, [][]float64, bool) {
			return [][]int{{0, 1, 2}}, [][]float64{{0, 1, 2}}, true
		},
		"wrong k": func(_, testRows [][]float64, _ int) ([][]int, [][]float64, bool) {
			indices := make([][]int, len(testRows))
			dist2 := make([][]float64, len(testRows))
			for i := range testRows {
				indices[i] = []int{0}
				dist2[i] = []float64{0}
			}
			return indices, dist2, true
		},
		"index out of range": func(_, testRows [][]float64, k int) ([][]int, [][]float64, bool) {
			indices := make([][]int, len(testRows))
			dist2 := make([][]float64, len(testRows))
			for i := range testRows {
				indices[i] = []int{0, 1, 99}
				dist2[i] = []float64{0, 1, 2}
			}
			return indices, dist2, true
		},
	}
	for name, searcher := range cases {
		t.Run(name, func(t *testing.T) {
			RegisterDeviceSearcher(searcher)
			defer RegisterDeviceSearcher(nil)
			got, err := Classify(train, test, labels, 3, Options{Algorithm: AutoAlgorithm})
			if err != nil {
				t.Fatalf("fallback errored: %v", err)
			}
			if !reflect.DeepEqual(got.Predictions, want.Predictions) {
				t.Fatalf("fallback predictions %v, want brute force's %v", got.Predictions, want.Predictions)
			}
		})
	}
}

// A correct socket answer must flow through regression and neighbors too, and
// the neighbor distances must be the square roots of the socket's squared
// distances — the same contract the CPU searchers honour.
func TestDeviceSocketAnswersFlowThroughAllThreeEntryPoints(t *testing.T) {
	train, test, _, targets := deviceFixture()
	brute := Options{Algorithm: BruteForceAlgorithm}
	auto := Options{Algorithm: AutoAlgorithm}

	// An honest searcher: computes the true answer the slow way.
	RegisterDeviceSearcher(func(trainRows, testRows [][]float64, k int) ([][]int, [][]float64, bool) {
		result, err := Neighbors(trainRows, testRows, k, brute)
		if err != nil {
			return nil, nil, false
		}
		dist2 := make([][]float64, len(result.Distances))
		for i, row := range result.Distances {
			dist2[i] = make([]float64, len(row))
			for j, d := range row {
				dist2[i][j] = d * d
			}
		}
		return result.Indices, dist2, true
	})
	defer RegisterDeviceSearcher(nil)

	wantRegress, err := Regress(train, test, targets, 3, brute)
	if err != nil {
		t.Fatal(err)
	}
	gotRegress, err := Regress(train, test, targets, 3, auto)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotRegress.Predictions, wantRegress.Predictions) {
		t.Fatalf("regression through the socket %v, want %v", gotRegress.Predictions, wantRegress.Predictions)
	}

	wantNeighbors, err := Neighbors(train, test, 3, brute)
	if err != nil {
		t.Fatal(err)
	}
	gotNeighbors, err := Neighbors(train, test, 3, auto)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotNeighbors.Indices, wantNeighbors.Indices) ||
		!reflect.DeepEqual(gotNeighbors.Distances, wantNeighbors.Distances) {
		t.Fatal("neighbors through the socket differ from brute force")
	}
}
