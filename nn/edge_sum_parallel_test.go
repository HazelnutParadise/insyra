package nn

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"
)

func TestEdgeSumWorkerCountDoesNotChangeBits(t *testing.T) {
	const (
		nodes       = 20_000
		targetNodes = 2_000
		edges       = 200_000
		batch       = 3
	)
	rng := rand.New(rand.NewSource(11))
	topology := mustRandomEdgeTopology(t, nodes, targetNodes, edges, rng)
	weights := mustRandomFloat32Tensor(t, []int{edges}, edges, rng)
	values := mustRandomFloat32Tensor(t, []int{batch, nodes}, batch*nodes, rng)
	upstream := mustRandomFloat32Tensor(t, []int{batch, nodes}, batch*nodes, rng)
	workers := max(4, runtime.NumCPU())

	serialForward, err := edgeSumForward(topology, weights, values, batch, 1)
	if err != nil {
		t.Fatalf("edgeSumForward with one worker: %v", err)
	}
	parallelForward, err := edgeSumForward(topology, weights, values, batch, workers)
	if err != nil {
		t.Fatalf("edgeSumForward with %d workers: %v", workers, err)
	}
	serialGradients, err := edgeSumVJPWith(topology, weights, values, upstream, batch, 1)
	if err != nil {
		t.Fatalf("edgeSumVJPWith with one worker: %v", err)
	}
	parallelGradients, err := edgeSumVJPWith(topology, weights, values, upstream, batch, workers)
	if err != nil {
		t.Fatalf("edgeSumVJPWith with %d workers: %v", workers, err)
	}
	publicForward, err := EdgeSum(topology, weights, values)
	if err != nil {
		t.Fatalf("EdgeSum: %v", err)
	}

	assertExactTensorEqual(t, parallelForward, serialForward)
	assertExactTensorEqual(t, parallelGradients[0], serialGradients[0])
	assertExactTensorEqual(t, parallelGradients[1], serialGradients[1])
	assertExactTensorEqual(t, publicForward, serialForward)
}

func TestEdgeSumWorkerThreshold(t *testing.T) {
	if got := edgeSumWorkers(1, 3, 3); got != 1 {
		t.Fatalf("edgeSumWorkers(1, 3, 3) = %d, want 1", got)
	}
	if got, want := edgeSumWorkers(3, 20_000, 200_000), runtime.NumCPU(); got != want {
		t.Fatalf("edgeSumWorkers(3, 20000, 200000) = %d, want %d", got, want)
	}
}

func BenchmarkEdgeSum(b *testing.B) {
	shapes := []struct {
		nodes  int
		degree int
	}{
		{nodes: 10_000, degree: 10},
		{nodes: 10_000, degree: 100},
		{nodes: 100_000, degree: 10},
		{nodes: 100_000, degree: 100},
		{nodes: 1_000_000, degree: 10},
	}
	coreCounts := []struct {
		name    string
		workers int
	}{
		{name: "1", workers: 1},
		{name: "all", workers: runtime.NumCPU()},
	}

	for _, shape := range shapes {
		for _, batch := range []int{1, 16} {
			for _, direction := range []string{"forward", "backward"} {
				for _, cores := range coreCounts {
					b.Run(fmt.Sprintf("N=%d/deg=%d/B=%d/%s/cores=%s", shape.nodes, shape.degree, batch, direction, cores.name), func(b *testing.B) {
						edges := shape.nodes * shape.degree
						graphRNG := rand.New(rand.NewSource(23))
						topology := mustRandomEdgeTopology(b, shape.nodes, shape.nodes, edges, graphRNG)
						dataRNG := rand.New(rand.NewSource(29))
						weights := mustRandomFloat32Tensor(b, []int{edges}, edges, dataRNG)
						values := mustRandomFloat32Tensor(b, []int{batch, shape.nodes}, batch*shape.nodes, dataRNG)
						upstream := mustRandomFloat32Tensor(b, []int{batch, shape.nodes}, batch*shape.nodes, dataRNG)
						b.ResetTimer()
						if direction == "forward" {
							for i := 0; i < b.N; i++ {
								if _, err := edgeSumForward(topology, weights, values, batch, cores.workers); err != nil {
									b.Fatal(err)
								}
							}
							return
						}
						for i := 0; i < b.N; i++ {
							if _, err := edgeSumVJPWith(topology, weights, values, upstream, batch, cores.workers); err != nil {
								b.Fatal(err)
							}
						}
					})
				}
			}
		}
	}
}

func mustRandomEdgeTopology(tb testing.TB, nodes, targetNodes, edges int, rng *rand.Rand) *EdgeTopology {
	tb.Helper()
	sources := make([]int, edges)
	targets := make([]int, edges)
	for edge := range edges {
		sources[edge] = rng.Intn(nodes)
		targets[edge] = rng.Intn(targetNodes)
	}
	topology, err := NewEdgeTopology(nodes, sources, targets)
	if err != nil {
		tb.Fatalf("NewEdgeTopology: %v", err)
	}
	return topology
}

func mustRandomFloat32Tensor(tb testing.TB, shape []int, size int, rng *rand.Rand) *Tensor {
	tb.Helper()
	data := make([]float32, size)
	for index := range data {
		data[index] = float32(rng.NormFloat64())
	}
	tensor, err := NewFloat32Tensor(shape, data)
	if err != nil {
		tb.Fatalf("NewFloat32Tensor: %v", err)
	}
	return tensor
}
