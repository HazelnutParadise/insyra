package wgpu_test

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra/accel/internal/wgpu"
	"github.com/HazelnutParadise/insyra/nn"
)

func TestEdgeSumDeviceTiming(t *testing.T) {
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	if _, err := wgpu.Probe(); err != nil {
		t.Skipf("cannot discover a usable GPU: %v", err)
	}

	ctx := context.Background()
	sizes := []edgeSumTimingSize{
		{nodes: 10000, degree: 10},
		{nodes: 10000, degree: 100},
		{nodes: 100000, degree: 10},
		{nodes: 100000, degree: 100},
		{nodes: 1000000, degree: 10},
	}
	for _, size := range sizes {
		for _, batch := range []int{1, 16} {
			nodes := size.nodes
			degree := size.degree
			edges := nodes * degree
			rng := rand.New(rand.NewSource(int64(nodes + degree*7 + batch*131)))

			sources := make([]int, edges)
			for i := range sources {
				sources[i] = rng.Intn(nodes)
			}
			targets := make([]int, edges)
			for i := range targets {
				targets[i] = rng.Intn(nodes)
			}
			weights := make([]float32, edges)
			for i := range weights {
				weights[i] = float32(rng.NormFloat64())
			}
			values := make([]float32, batch*nodes)
			for i := range values {
				values[i] = float32(rng.NormFloat64())
			}

			topology, err := nn.NewEdgeTopology(nodes, sources, targets)
			if err != nil {
				t.Fatalf("nn.NewEdgeTopology: %v", err)
			}
			weightsTensor, err := nn.NewTensor([]int{edges}, weights)
			if err != nil {
				t.Fatalf("nn.NewTensor weights: %v", err)
			}
			valuesTensor, err := nn.NewTensor([]int{batch, nodes}, values)
			if err != nil {
				t.Fatalf("nn.NewTensor values: %v", err)
			}
			want, cpuMS, err := edgeSumTimingCPU(topology, weightsTensor, valuesTensor)
			if err != nil {
				t.Fatalf("CPU nn.EdgeSum: %v", err)
			}

			csr := wgpu.NewEdgeSumCSRForTest(nodes, sources, targets)
			fullResult, fullMS, err := edgeSumTimingFullUpload(ctx, csr, weights, values, batch)
			if err != nil {
				t.Fatalf("full-upload edge sum: %v", err)
			}
			mismatchedFull := edgeSumTimingMismatches(want, fullResult)

			resident, err := wgpu.NewEdgeSumResidentForTest(csr)
			if err != nil {
				t.Fatalf("create resident edge sum: %v", err)
			}
			func() {
				defer resident.Release()

				residentResult, residentMS, err := edgeSumTimingResident(ctx, resident, weights, values, batch)
				if err != nil {
					t.Fatalf("resident edge sum: %v", err)
				}
				mismatchedResident := edgeSumTimingMismatches(want, residentResult)
				fmt.Printf(
					"edge-sum timing N=%d deg=%d B=%d cpu_ms=%.3f full_ms=%.3f resident_ms=%.3f speedup_full=%.2f speedup_resident=%.2f mismatched_full=%d mismatched_resident=%d cores=%d\n",
					nodes,
					degree,
					batch,
					cpuMS,
					fullMS,
					residentMS,
					cpuMS/fullMS,
					cpuMS/residentMS,
					mismatchedFull,
					mismatchedResident,
					runtime.NumCPU(),
				)
			}()

			runtime.GC()
		}
	}
}

type edgeSumTimingSize struct {
	nodes  int
	degree int
}

func edgeSumTimingCPU(topology *nn.EdgeTopology, weights, values *nn.Tensor) ([]float32, float64, error) {
	result, err := nn.EdgeSum(topology, weights, values)
	if err != nil {
		return nil, 0, err
	}
	want := result.Data()

	var best time.Duration
	for attempt := 0; attempt < 5; attempt++ {
		start := time.Now()
		_, err = nn.EdgeSum(topology, weights, values)
		elapsed := time.Since(start)
		if err != nil {
			return nil, 0, err
		}
		if attempt == 0 || elapsed < best {
			best = elapsed
		}
	}
	return want, float64(best) / float64(time.Millisecond), nil
}

func edgeSumTimingFullUpload(ctx context.Context, csr wgpu.EdgeSumCSRForTest, weights, values []float32, batch int) ([]float32, float64, error) {
	result, err := wgpu.RunEdgeSumFullUploadForTest(ctx, wgpu.EdgeSumSeparatedWGSLForTest, csr, weights, values, batch)
	if err != nil {
		return nil, 0, err
	}

	var best time.Duration
	for attempt := 0; attempt < 5; attempt++ {
		start := time.Now()
		_, err = wgpu.RunEdgeSumFullUploadForTest(ctx, wgpu.EdgeSumSeparatedWGSLForTest, csr, weights, values, batch)
		elapsed := time.Since(start)
		if err != nil {
			return nil, 0, err
		}
		if attempt == 0 || elapsed < best {
			best = elapsed
		}
	}
	return result, float64(best) / float64(time.Millisecond), nil
}

func edgeSumTimingResident(ctx context.Context, resident *wgpu.EdgeSumResidentForTest, weights, values []float32, batch int) ([]float32, float64, error) {
	result, err := resident.Run(ctx, wgpu.EdgeSumSeparatedWGSLForTest, weights, values, batch)
	if err != nil {
		return nil, 0, err
	}

	var best time.Duration
	for attempt := 0; attempt < 5; attempt++ {
		start := time.Now()
		_, err = resident.Run(ctx, wgpu.EdgeSumSeparatedWGSLForTest, weights, values, batch)
		elapsed := time.Since(start)
		if err != nil {
			return nil, 0, err
		}
		if attempt == 0 || elapsed < best {
			best = elapsed
		}
	}
	return result, float64(best) / float64(time.Millisecond), nil
}

func edgeSumTimingMismatches(want, got []float32) int {
	shared := len(want)
	if len(got) < shared {
		shared = len(got)
	}
	mismatches := len(want) + len(got) - 2*shared
	for i := 0; i < shared; i++ {
		if math.Float32bits(want[i]) != math.Float32bits(got[i]) {
			mismatches++
		}
	}
	return mismatches
}
