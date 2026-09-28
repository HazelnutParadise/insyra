package wgpu_test

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/accel/internal/wgpu"
	"github.com/HazelnutParadise/insyra/nn"
)

// exactSumEdgeMaxPrinted is how many disagreeing rows each accumulation prints
// before it keeps only the count.
const exactSumEdgeMaxPrinted = 20

// exactSumEdgeVariants names the two device accumulations under test, in the
// order they are reported.
var exactSumEdgeVariants = []string{"single", "split"}

// exactSumEdgePairPrefix renders up to the first four operand pairs of a row as
// float32 bit patterns, so a disagreeing row can be reproduced without printing
// the rest of it.
func exactSumEdgePairPrefix(row [][2]uint32) string {
	limit := len(row)
	if limit > 4 {
		limit = 4
	}
	parts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		parts = append(parts, fmt.Sprintf("(%#08x,%#08x)", row[i][0], row[i][1]))
	}
	return strings.Join(parts, " ")
}

// exactSumEdgeCPUReference sums every row of the parity set with nn.EdgeSum and
// returns its output indexed by row. Row i is fed to the graph as the incoming
// edges of node i, with each operand pair as one edge whose weight is the x
// pattern and whose source holds the y pattern, so node i's output is the same
// quantity the device harness computes for row i.
func exactSumEdgeCPUReference(t *testing.T, rows [][][2]uint32) []float32 {
	t.Helper()

	edges := 0
	for _, row := range rows {
		edges += len(row)
	}
	nodes := edges
	if len(rows) > nodes {
		nodes = len(rows)
	}
	if nodes < 1 {
		nodes = 1
	}

	sources := make([]int, edges)
	targets := make([]int, edges)
	weights := make([]float32, edges)
	values := make([]float32, nodes)
	e := 0
	for i, row := range rows {
		for _, pair := range row {
			sources[e] = e
			targets[e] = i
			weights[e] = math.Float32frombits(pair[0])
			values[e] = math.Float32frombits(pair[1])
			e++
		}
	}

	topology, err := nn.NewEdgeTopology(nodes, sources, targets)
	if err != nil {
		t.Fatalf("nn.NewEdgeTopology: %v", err)
	}
	weightsTensor, err := nn.NewTensor([]int{edges}, weights)
	if err != nil {
		t.Fatalf("nn.NewTensor weights: %v", err)
	}
	valuesTensor, err := nn.NewTensor([]int{nodes}, values)
	if err != nil {
		t.Fatalf("nn.NewTensor values: %v", err)
	}
	result, err := nn.EdgeSum(topology, weightsTensor, valuesTensor)
	if err != nil {
		t.Fatalf("nn.EdgeSum: %v", err)
	}
	return result.Data()
}

// TestExactSumDeviceMatchesEdgeSum holds the device exact-sum harness to
// nn.EdgeSum over the adversarial row set: each row's operand pairs become the
// incoming edges of one node, and every device output has to equal the CPU
// output bit for bit, for both the single-register and the split accumulation.
func TestExactSumDeviceMatchesEdgeSum(t *testing.T) {
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	if _, err := wgpu.Probe(); err != nil {
		t.Skipf("cannot discover a usable GPU: %v", err)
	}

	names, rows := wgpu.ExactSumParityRowsForTest(rand.New(rand.NewSource(35)))
	if len(names) != len(rows) {
		t.Fatalf("row set has %d names for %d rows", len(names), len(rows))
	}

	single, split, err := wgpu.RunExactSumHarnessForTest(context.Background(), rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(single) != len(rows) || len(split) != len(rows) {
		t.Fatalf("harness returned %d single and %d split outputs for %d rows", len(single), len(split), len(rows))
	}

	out := exactSumEdgeCPUReference(t, rows)
	if len(out) < len(rows) {
		t.Fatalf("nn.EdgeSum returned %d outputs for %d rows", len(out), len(rows))
	}

	totalPairs := 0
	for _, row := range rows {
		totalPairs += len(row)
	}

	wrong := make(map[string]int, len(exactSumEdgeVariants))
	printed := make(map[string]int, len(exactSumEdgeVariants))
	for i := range rows {
		cpu := math.Float32bits(out[i])
		for _, variant := range exactSumEdgeVariants {
			got := single[i]
			if variant == "split" {
				got = split[i]
			}
			if got == cpu {
				continue
			}
			wrong[variant]++
			if printed[variant] == exactSumEdgeMaxPrinted {
				continue
			}
			printed[variant]++
			t.Errorf("%s row %d of %d: %d pairs, cpu=%#08x single=%#08x split=%#08x, first pairs %s",
				variant, i, len(rows), len(rows[i]), cpu, single[i], split[i], exactSumEdgePairPrefix(rows[i]))
		}
	}

	for _, variant := range exactSumEdgeVariants {
		t.Logf("%s: %d rows, %d pairs, %d rows disagree with nn.EdgeSum", variant, len(rows), totalPairs, wrong[variant])
	}
	for _, variant := range exactSumEdgeVariants {
		if wrong[variant] > 0 {
			t.Errorf("%s: %d of %d rows disagree with nn.EdgeSum", variant, wrong[variant], len(rows))
		}
	}
}
