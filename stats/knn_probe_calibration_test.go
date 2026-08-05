package stats

import (
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

type knnProbeCalibrationCell struct {
	regime string
	n      int
	dims   int
}

func knnProbeCalibrationTable(rows, dims int, seed uint64, clustered bool) *insyra.DataTable {
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	data := make([][]float64, rows)
	for i := range data {
		data[i] = make([]float64, dims)
		if clustered {
			center := float64(i%8) * 8
			for j := range data[i] {
				data[i][j] = center + rng.NormFloat64()*0.15
			}
		} else {
			for j := range data[i] {
				data[i][j] = rng.NormFloat64()
			}
		}
	}
	dt := insyra.NewDataTable()
	for j := 0; j < dims; j++ {
		col := insyra.NewDataList()
		for i := range data {
			col.Append(data[i][j])
		}
		dt.AppendCols(col)
	}
	return dt
}

func knnProbeCalibrationMeasure(c knnProbeCalibrationCell, algorithm KNNAlgorithm, leafSize int) time.Duration {
	train := knnProbeCalibrationTable(c.n, c.dims, uint64(c.n*100+c.dims), c.regime == "clustered")
	test := knnProbeCalibrationTable(1000, c.dims, uint64(c.n*100+c.dims+1), c.regime == "clustered")
	start := time.Now()
	if _, err := KNearestNeighbors(train, test, 5, KNNOptions{Algorithm: algorithm, LeafSize: leafSize}); err != nil {
		panic(err)
	}
	return time.Since(start)
}

func TestKNNProbeCalibrationBaseline(t *testing.T) {
	if testing.Short() || getenv("INSYRA_KNN_PROBE_CALIBRATE") != "1" {
		t.Skip("set INSYRA_KNN_PROBE_CALIBRATE=1")
	}
	for _, regime := range []string{"unstructured", "clustered"} {
		for _, n := range []int{5000, 20000, 50000} {
			for _, dims := range []int{16, 64} {
				c := knnProbeCalibrationCell{regime: regime, n: n, dims: dims}
				brute := knnProbeCalibrationMeasure(c, KNNBruteForce, 16)
				ball := knnProbeCalibrationMeasure(c, KNNBallTree, 16)
				t.Logf("%s n=%d dims=%d brute=%s ball=%s ball/brute=%.3f", regime, n, dims, brute, ball, float64(ball)/float64(brute))
			}
		}
	}
}

func TestKNNProbeCalibrationLeafSizes(t *testing.T) {
	if testing.Short() || getenv("INSYRA_KNN_PROBE_CALIBRATE") != "1" {
		t.Skip("set INSYRA_KNN_PROBE_CALIBRATE=1")
	}
	for _, regime := range []string{"unstructured", "clustered"} {
		for _, n := range []int{5000, 20000, 50000} {
			for _, dims := range []int{16, 64} {
				c := knnProbeCalibrationCell{regime: regime, n: n, dims: dims}
				for _, leafSize := range []int{8, 16, 32, 64} {
					ball := knnProbeCalibrationMeasure(c, KNNBallTree, leafSize)
					t.Logf("%s n=%d dims=%d leaf=%d ball=%s", regime, n, dims, leafSize, ball)
				}
			}
		}
	}
}

func getenv(key string) string { return os.Getenv(key) }
