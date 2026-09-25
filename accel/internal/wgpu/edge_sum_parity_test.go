package wgpu

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"runtime"
	"testing"
)

type edgeSumParityGraph struct {
	name    string
	nodes   int
	batch   int
	sources []int
	targets []int
	weights []float32
	values  []float32
}

func TestEdgeSumPrototypeParity(t *testing.T) {
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	if _, err := Probe(); err != nil {
		t.Skipf("cannot discover a usable GPU: %v", err)
	}

	graphs := []edgeSumParityGraph{
		{
			name:    "issue",
			nodes:   3,
			batch:   1,
			sources: []int{0, 1, 2},
			targets: []int{1, 2, 0},
			weights: []float32{0.5, -1, 0.25},
			values:  []float32{0.1, 0.2, 0.3},
		},
		edgeSumParityRandomGraph("repeated", 2000, 60000, 200, 4, 7),
		edgeSumParityRandomGraph("wide", 100000, 1000000, 100000, 2, 13),
	}
	variants := []struct {
		name string
		wgsl string
	}{
		{name: "plain", wgsl: edgeSumPlainWGSL},
		{name: "separated", wgsl: edgeSumSeparatedWGSL},
	}
	references := []struct {
		name   string
		values []float32
	}{
		{name: "rounded", values: nil},
		{name: "unrounded", values: nil},
	}

	for _, graph := range graphs {
		csr := newEdgeSumCSR(graph.nodes, graph.sources, graph.targets)
		references[0].values = edgeSumCPURounded(graph.sources, graph.targets, graph.weights, graph.values, graph.batch, graph.nodes)
		references[1].values = edgeSumCPUUnrounded(graph.sources, graph.targets, graph.weights, graph.values, graph.batch, graph.nodes)

		for _, variant := range variants {
			got, err := runEdgeSumPrototype(context.Background(), variant.wgsl, csr, graph.weights, graph.values, graph.batch)
			if err != nil {
				t.Fatalf("edge-sum graph=%s variant=%s: %v", graph.name, variant.name, err)
			}
			total := graph.batch * graph.nodes
			if len(got) != total {
				t.Fatalf("edge-sum graph=%s variant=%s returned %d outputs, want %d", graph.name, variant.name, len(got), total)
			}

			for _, reference := range references {
				mismatched, maxULP := edgeSumParityCompare(got, reference.values)
				fmt.Printf(
					"edge-sum parity graph=%s variant=%s reference=%s mismatched=%d/%d maxULP=%d goarch=%s\n",
					graph.name,
					variant.name,
					reference.name,
					mismatched,
					total,
					maxULP,
					runtime.GOARCH,
				)
			}
		}
	}
}

func edgeSumParityRandomGraph(name string, nodes, edges, targetNodes, batch int, seed int64) edgeSumParityGraph {
	r := rand.New(rand.NewSource(seed))
	sources := make([]int, edges)
	targets := make([]int, edges)
	for edge := range edges {
		sources[edge] = r.Intn(nodes)
		targets[edge] = r.Intn(targetNodes)
	}
	weights := make([]float32, edges)
	for edge := range weights {
		weights[edge] = float32(r.NormFloat64())
	}
	values := make([]float32, batch*nodes)
	for value := range values {
		values[value] = float32(r.NormFloat64())
	}
	return edgeSumParityGraph{
		name:    name,
		nodes:   nodes,
		batch:   batch,
		sources: sources,
		targets: targets,
		weights: weights,
		values:  values,
	}
}

func edgeSumCPURounded(sources, targets []int, weights, values []float32, batch, nodes int) []float32 {
	result := make([]float32, batch*nodes)
	for b := 0; b < batch; b++ {
		for t := 0; t < nodes; t++ {
			acc := float32(0)
			for e := 0; e < len(sources); e++ {
				if targets[e] == t {
					acc += float32(weights[e] * values[b*nodes+sources[e]])
				}
			}
			result[b*nodes+t] = acc
		}
	}
	return result
}

func edgeSumCPUUnrounded(sources, targets []int, weights, values []float32, batch, nodes int) []float32 {
	result := make([]float32, batch*nodes)
	for b := 0; b < batch; b++ {
		for t := 0; t < nodes; t++ {
			acc := float32(0)
			for e := 0; e < len(sources); e++ {
				if targets[e] == t {
					acc += weights[e] * values[b*nodes+sources[e]]
				}
			}
			result[b*nodes+t] = acc
		}
	}
	return result
}

func edgeSumParityCompare(got, want []float32) (int, int64) {
	mismatched := 0
	maxULP := int64(0)
	for i := range want {
		if math.Float32bits(got[i]) != math.Float32bits(want[i]) {
			mismatched++
		}
		ulp := edgeSumParityULP(got[i], want[i])
		if ulp > maxULP {
			maxULP = ulp
		}
	}
	return mismatched, maxULP
}

func edgeSumParityULP(a, b float32) int64 {
	first := edgeSumParityOrderedBits(a)
	second := edgeSumParityOrderedBits(b)
	if first > second {
		return first - second
	}
	return second - first
}

func edgeSumParityOrderedBits(value float32) int64 {
	bits := math.Float32bits(value)
	if bits&0x80000000 != 0 {
		return -int64(bits & 0x7fffffff)
	}
	return int64(bits)
}
