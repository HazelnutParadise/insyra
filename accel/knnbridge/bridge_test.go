package knnbridge

import (
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func bridgeFixture(train, test, dims int, seed int64) (trainRows, testRows [][]float64, labels []string, targets []float64) {
	rnd := rand.New(rand.NewSource(seed))
	trainRows = make([][]float64, train)
	labels = make([]string, train)
	targets = make([]float64, train)
	for i := range trainRows {
		row := make([]float64, dims)
		for c := range row {
			row[c] = rnd.NormFloat64()
		}
		trainRows[i] = row
		if row[0] > 0 {
			labels[i] = "pos"
		} else {
			labels[i] = "neg"
		}
		targets[i] = row[0]*2 + row[1%dims]
	}
	testRows = make([][]float64, test)
	for i := range testRows {
		row := make([]float64, dims)
		for c := range row {
			row[c] = rnd.NormFloat64()
		}
		testRows[i] = row
	}
	return
}

// The device path must be indistinguishable from brute force in everything
// but speed: the exact operation decides in float64 on the host, so parity is
// asserted exactly — index for index, distance for distance — not within a
// tolerance.
func TestBridgeMatchesBruteForceExactly(t *testing.T) {
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1 to run the device parity test")
	}
	// Shapes above both of the bridge's floors — per-row work and test rows —
	// with k inside the shortlist budget.
	train, test, dims := 4096, 4096, 8
	trainRows, testRows, _, _ := bridgeFixture(train, test, dims, 7)

	// The bridge registered itself in init; the fixture just has to be a
	// shape it accepts. Verify it actually engages by asking it directly.
	indices, dist2, ok := search(trainRows, testRows, 5)
	if !ok {
		t.Fatal("the bridge declined a shape chosen to clear its gates; the parity test exercised nothing")
	}
	if len(indices) != test || len(dist2) != test {
		t.Fatalf("bridge answered %d rows for %d test rows", len(indices), test)
	}

	// Brute force through the internal socket-free path: explicitly named,
	// so the socket is never consulted.
	viaDevice, err := knnNeighbors(trainRows, testRows, 5, "auto")
	if err != nil {
		t.Fatal(err)
	}
	viaBrute, err := knnNeighbors(trainRows, testRows, 5, "brute")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(viaDevice.Indices, viaBrute.Indices) {
		t.Fatal("device indices differ from brute force")
	}
	if !reflect.DeepEqual(viaDevice.Distances, viaBrute.Distances) {
		t.Fatal("device distances differ from brute force")
	}
}

func knnNeighbors(train, test [][]float64, k int, algorithm string) (*stats.KNNNeighborsResult, error) {
	return stats.KNearestNeighbors(tableFromRows(train), tableFromRows(test), k,
		stats.KNNOptions{Algorithm: stats.KNNAlgorithm(algorithm)})
}

func tableFromRows(rows [][]float64) *insyra.DataTable {
	dims := len(rows[0])
	columns := make([]*insyra.DataList, dims)
	for c := 0; c < dims; c++ {
		values := make([]any, len(rows))
		for i, row := range rows {
			values[i] = row[c]
		}
		columns[c] = insyra.NewDataList(values...).SetName(fmt.Sprintf("f%d", c))
	}
	return insyra.NewDataTable(columns...)
}

// Small shapes stay below the work floor, so the bridge declines them and the
// CPU path answers — with the bridge imported and no device consulted, the
// result must equal plain brute force.
func TestBridgeDeclinesSmallShapesAndNothingChanges(t *testing.T) {
	trainRows, testRows, _, _ := bridgeFixture(64, 8, 4, 11)
	if _, _, ok := search(trainRows, testRows, 3); ok {
		t.Fatal("the bridge accepted a shape below its own work floor")
	}
	// A large training set does not rescue a small test set: the kernel's
	// parallelism comes from test rows, and below the measured floor the
	// device's flat dispatch cost exceeds the CPU's whole job.
	bigTrain, smallTest, _, _ := bridgeFixture(4096, 256, 8, 12)
	if _, _, ok := search(bigTrain, smallTest, 3); ok {
		t.Fatal("the bridge accepted a test set below the measured test-row floor")
	}
	viaAuto, err := knnNeighbors(trainRows, testRows, 3, "auto")
	if err != nil {
		t.Fatal(err)
	}
	viaBrute, err := knnNeighbors(trainRows, testRows, 3, "brute")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(viaAuto.Indices, viaBrute.Indices) {
		t.Fatal("a declined shape changed the answer")
	}
}

// k beyond the shortlist budget is declined up front.
func TestBridgeDeclinesLargeK(t *testing.T) {
	trainRows, testRows, _, _ := bridgeFixture(4096, 32, 8, 13)
	if _, _, ok := search(trainRows, testRows, 8); ok {
		t.Fatal("k=8 exceeds the shortlist budget and was accepted")
	}
}
