package knn

import (
	"math"
	"math/rand/v2"
	"os"
	"strconv"
	"testing"
	"time"
)

func calibrationRows(rows, dims int, seed uint64, clustered bool) [][]float64 {
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	out := make([][]float64, rows)
	for i := range out {
		out[i] = make([]float64, dims)
		if clustered {
			center := float64(i%8) * 8
			for j := range out[i] {
				out[i][j] = center + rng.NormFloat64()*0.15
			}
		} else {
			for j := range out[i] {
				out[i][j] = rng.NormFloat64()
			}
		}
	}
	return out
}

func calibrationClusterRows(rows, dims int, seed uint64, noise float64) [][]float64 {
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	out := make([][]float64, rows)
	for i := range out {
		out[i] = make([]float64, dims)
		center := float64(i%8) * 8
		for j := range out[i] {
			out[i][j] = center + rng.NormFloat64()*noise
		}
	}
	return out
}

func probeFraction(s treeProbe, test [][]float64, trainRows, k int) float64 {
	total := 0
	for _, query := range test {
		total += s.examinedCandidates(query, k)
	}
	return float64(total) / float64(len(test)*trainRows)
}

func calibrationSample(rows [][]float64, m, offset int) [][]float64 {
	if m >= len(rows) {
		return rows
	}
	out := make([][]float64, m)
	for i := range out {
		out[i] = rows[(offset+i*len(rows)/m)%len(rows)]
	}
	return out
}

func meanAndStd(values []float64) (float64, float64) {
	mean := 0.0
	for _, value := range values {
		mean += value
	}
	mean /= float64(len(values))
	variance := 0.0
	for _, value := range values {
		delta := value - mean
		variance += delta * delta
	}
	return mean, math.Sqrt(variance / float64(len(values)))
}

func calibrationFilter(values []string, key string) []string {
	if key == "" {
		return values
	}
	for _, value := range values {
		if value == key {
			return []string{value}
		}
	}
	return nil
}

func calibrationInts(values []int, key string) []int {
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

func calibrationFloats(values []float64, key string) []float64 {
	if key == "" {
		return values
	}
	want, err := strconv.ParseFloat(key, 64)
	if err != nil {
		return nil
	}
	return []float64{want}
}

func TestKNNProbeCalibrationFractions(t *testing.T) {
	if testing.Short() || os.Getenv("INSYRA_KNN_PROBE_CALIBRATE") != "1" {
		t.Skip("set INSYRA_KNN_PROBE_CALIBRATE=1")
	}
	for _, regime := range []string{"unstructured", "clustered"} {
		for _, n := range []int{5000, 20000, 50000} {
			for _, dims := range []int{16, 64} {
				train := calibrationRows(n, dims, uint64(n*100+dims), regime == "clustered")
				test := calibrationRows(1000, dims, uint64(n*100+dims+1), regime == "clustered")
				ball := newBallTreeSearcher(train, 16)
				t.Logf("%s n=%d dims=%d ball examined fraction=%.6f", regime, n, dims, probeFraction(ball, test, n, 5))
			}
		}
	}
}

func TestKNNProbeCalibrationSampleSizes(t *testing.T) {
	if testing.Short() || os.Getenv("INSYRA_KNN_PROBE_CALIBRATE") != "1" {
		t.Skip("set INSYRA_KNN_PROBE_CALIBRATE=1")
	}
	for _, regime := range []string{"unstructured", "clustered"} {
		for _, n := range []int{5000, 20000, 50000} {
			for _, dims := range []int{16, 64} {
				train := calibrationRows(n, dims, uint64(n*100+dims), regime == "clustered")
				test := calibrationRows(1000, dims, uint64(n*100+dims+1), regime == "clustered")
				ball := newBallTreeSearcher(train, 16)
				for _, m := range []int{8, 16, 32, 64, 128} {
					fractions := make([]float64, 0, 8)
					start := time.Now()
					for offset := 0; offset < 8; offset++ {
						fractions = append(fractions, probeFraction(ball, calibrationSample(test, m, offset), n, 5))
					}
					mean, std := meanAndStd(fractions)
					min, max := fractions[0], fractions[0]
					for _, fraction := range fractions[1:] {
						if fraction < min {
							min = fraction
						}
						if fraction > max {
							max = fraction
						}
					}
					t.Logf("%s n=%d dims=%d m=%d mean=%.6f std=%.6f min=%.6f max=%.6f probe8=%s", regime, n, dims, m, mean, std, min, max, time.Since(start))
				}
			}
		}
	}
}

func TestKNNProbeCalibrationFloor(t *testing.T) {
	if testing.Short() || os.Getenv("INSYRA_KNN_PROBE_CALIBRATE") != "1" {
		t.Skip("set INSYRA_KNN_PROBE_CALIBRATE=1")
	}
	for _, regime := range []string{"unstructured", "clustered"} {
		for _, n := range []int{64, 128, 256, 512, 1024, 2048, 5000, 10000} {
			for _, dims := range []int{16, 64} {
				train := calibrationRows(n, dims, uint64(n*100+dims), regime == "clustered")
				test := calibrationRows(1000, dims, uint64(n*100+dims+1), regime == "clustered")
				buildStart := time.Now()
				ball := newBallTreeSearcher(train, 16)
				build := time.Since(buildStart)
				probeStart := time.Now()
				fraction := probeFraction(ball, calibrationSample(test, 16, 0), n, 5)
				probe := time.Since(probeStart)
				ballStart := time.Now()
				for _, query := range test {
					_ = ball.QueryKNN(query, 5)
				}
				ballQuery := time.Since(ballStart)
				brute := &bruteSearcher{train: train}
				bruteStart := time.Now()
				for _, query := range test {
					_ = brute.QueryKNN(query, 5)
				}
				bruteQuery := time.Since(bruteStart)
				t.Logf("%s n=%d dims=%d fraction=%.6f build=%s probe16=%s ball1000=%s brute1000=%s", regime, n, dims, fraction, build, probe, ballQuery, bruteQuery)
			}
		}
	}
}

func TestKNNProbeCalibrationCrossover(t *testing.T) {
	if testing.Short() || os.Getenv("INSYRA_KNN_PROBE_CALIBRATE") != "1" {
		t.Skip("set INSYRA_KNN_PROBE_CALIBRATE=1")
	}
	for _, n := range calibrationInts([]int{5000, 20000, 50000}, os.Getenv("INSYRA_KNN_CALIBRATION_N")) {
		for _, dims := range calibrationInts([]int{16, 64}, os.Getenv("INSYRA_KNN_CALIBRATION_DIMS")) {
			for _, noise := range calibrationFloats([]float64{0.15, 0.5, 1, 2, 4, 8}, os.Getenv("INSYRA_KNN_CALIBRATION_NOISE")) {
				train := calibrationClusterRows(n, dims, uint64(n*100+dims), noise)
				test := calibrationClusterRows(1000, dims, uint64(n*100+dims+1), noise)
				ball := newBallTreeSearcher(train, 16)
				fraction := probeFraction(ball, test, n, 5)
				ballStart := time.Now()
				for _, query := range test {
					_ = ball.QueryKNN(query, 5)
				}
				ballTime := time.Since(ballStart)
				brute := &bruteSearcher{train: train}
				bruteStart := time.Now()
				for _, query := range test {
					_ = brute.QueryKNN(query, 5)
				}
				bruteTime := time.Since(bruteStart)
				t.Logf("n=%d dims=%d noise=%.2f fraction=%.6f ball1000=%s brute1000=%s ball/brute=%.3f", n, dims, noise, fraction, ballTime, bruteTime, float64(ballTime)/float64(bruteTime))
			}
		}
	}
}

func TestKNNProbeCalibrationKDTree(t *testing.T) {
	if testing.Short() || os.Getenv("INSYRA_KNN_PROBE_CALIBRATE") != "1" {
		t.Skip("set INSYRA_KNN_PROBE_CALIBRATE=1")
	}
	regimes := calibrationFilter([]string{"unstructured", "clustered"}, os.Getenv("INSYRA_KNN_CALIBRATION_REGIME"))
	for _, regime := range regimes {
		for _, n := range []int{5000, 20000, 50000} {
			for _, dims := range []int{4, 8} {
				train := calibrationRows(n, dims, uint64(n*100+dims), regime == "clustered")
				test := calibrationRows(1000, dims, uint64(n*100+dims+1), regime == "clustered")
				buildStart := time.Now()
				kd := newKDTreeSearcher(train, 16)
				build := time.Since(buildStart)
				fraction := probeFraction(kd, test, n, 5)
				kdStart := time.Now()
				for _, query := range test {
					_ = kd.QueryKNN(query, 5)
				}
				kdTime := time.Since(kdStart)
				brute := &bruteSearcher{train: train}
				bruteStart := time.Now()
				for _, query := range test {
					_ = brute.QueryKNN(query, 5)
				}
				bruteTime := time.Since(bruteStart)
				t.Logf("%s n=%d dims=%d kd fraction=%.6f build=%s kd1000=%s brute1000=%s kd/brute=%.3f", regime, n, dims, fraction, build, kdTime, bruteTime, float64(kdTime)/float64(bruteTime))
			}
		}
	}
}
