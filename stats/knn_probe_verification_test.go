package stats

import (
	"os"
	"strconv"
	"testing"
	"time"
)

// knnProbeBenchmarkTolerance is the recorded issue-190 ladder tolerance:
// post-wiring runs observed a maximum of 1.414x, rounded up to 1.45.
const knnProbeBenchmarkTolerance = 1.45

func verificationInts(values []int, key string) []int {
	if key == "" {
		return values
	}
	want, err := strconv.Atoi(key)
	if err != nil {
		return nil
	}
	for _, value := range values {
		if value == want {
			return []int{value}
		}
	}
	return nil
}

func TestKNNProbeIssue190LadderBenchmark(t *testing.T) {
	if testing.Short() || os.Getenv("INSYRA_KNN_PROBE_VERIFY") != "1" {
		t.Skip("set INSYRA_KNN_PROBE_VERIFY=1")
	}
	regimes := []string{"unstructured", "clustered"}
	if requested := os.Getenv("INSYRA_KNN_PROBE_REGIME"); requested != "" {
		regimes = []string{requested}
	}
	for _, regime := range regimes {
		for _, n := range verificationInts([]int{5000, 20000, 50000}, os.Getenv("INSYRA_KNN_PROBE_N")) {
			for _, dims := range verificationInts([]int{16, 64}, os.Getenv("INSYRA_KNN_PROBE_DIMS")) {
				train := knnProbeCalibrationTable(n, dims, uint64(n*100+dims), regime == "clustered")
				test := knnProbeCalibrationTable(1000, dims, uint64(n*100+dims+1), regime == "clustered")
				times := make(map[KNNAlgorithm]time.Duration, 3)
				for _, algorithm := range []KNNAlgorithm{KNNBruteForce, KNNBallTree, KNNAuto} {
					start := time.Now()
					if _, err := KNearestNeighbors(train, test, 5, KNNOptions{Algorithm: algorithm, LeafSize: 16}); err != nil {
						t.Fatal(err)
					}
					times[algorithm] = time.Since(start)
				}
				best := times[KNNBruteForce]
				if times[KNNBallTree] < best {
					best = times[KNNBallTree]
				}
				ratio := float64(times[KNNAuto]) / float64(best)
				t.Logf("%s n=%d dims=%d brute=%s ball=%s auto=%s auto/best=%.3f", regime, n, dims, times[KNNBruteForce], times[KNNBallTree], times[KNNAuto], ratio)
				if ratio > knnProbeBenchmarkTolerance {
					t.Fatalf("auto exceeded recorded tolerance: ratio=%.3f tolerance=%.2f", ratio, knnProbeBenchmarkTolerance)
				}
			}
		}
	}
}
