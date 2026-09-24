package clustering

import "testing"

// IN-5: with NStart=1 the initial centres were drawn from the raw rows, so a
// dataset with repeated rows kept drawing the same point twice and failed
// with "empty cluster". R draws from the distinct rows instead.
func TestKMeansNStart1WithDuplicateRows(t *testing.T) {
	data := make([][]float64, 0, 21)
	for i := 0; i < 20; i++ {
		data = append(data, []float64{0, 0})
	}
	data = append(data, []float64{10, 10})

	failures := 0
	for seed := int64(1); seed <= 50; seed++ {
		s := seed
		_, err := KMeans(data, 2, KMeansOptions{NStart: 1, Seed: &s})
		if err != nil {
			failures++
			t.Logf("seed %d: %v", seed, err)
		}
	}
	if failures > 0 {
		t.Fatalf("%d of 50 seeds failed on duplicated rows", failures)
	}
}

// The guard must not fire when the draw was already distinct: results for a
// clean dataset stay bit-identical for a given seed.
func TestKMeansDeterminismUnchangedWithoutDuplicates(t *testing.T) {
	data := [][]float64{{0, 0}, {0.2, 0.1}, {5, 5}, {5.2, 4.9}, {10, 10}, {9.8, 10.1}}
	seed := int64(42)
	first, err := KMeans(data, 3, KMeansOptions{NStart: 1, Seed: &seed})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		again, err := KMeans(data, 3, KMeansOptions{NStart: 1, Seed: &seed})
		if err != nil {
			t.Fatal(err)
		}
		for j := range first.Cluster {
			if first.Cluster[j] != again.Cluster[j] {
				t.Fatalf("run %d differs: %v vs %v", i, first.Cluster, again.Cluster)
			}
		}
	}
}
