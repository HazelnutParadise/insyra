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

// exactSumTimingMaxPrinted is how many disagreeing rows the throughput
// measurement prints before it keeps only the count.
const exactSumTimingMaxPrinted = 10

// exactSumTimingRepeats is how many times each side is timed, the shortest run
// being reported.
const exactSumTimingRepeats = 5

// exactSumTimingSize is one measured shape: a regular block of products, split
// into rows of degree products each.
type exactSumTimingSize struct {
	rows   int
	degree int
}

// TestExactSumDeviceThroughput measures the device's exact-register row sum
// against nn.EdgeSum on all cores over the same products, each side doing one
// accumulation per row. Both sides are timed around the whole call — upload,
// dispatch and readback for the device, output allocation included for the CPU
// — so the numbers are what a caller would actually wait for. It is a
// measurement, not a correctness check, which is why it also compares the two
// answers bit for bit: a device that is fast because it did less work must not
// pass quietly.
func TestExactSumDeviceThroughput(t *testing.T) {
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	if os.Getenv("INSYRA_ACCEL_TIMING") != "1" {
		t.Skip("set INSYRA_ACCEL_TIMING=1: this is a throughput measurement, not a correctness check")
	}
	if _, err := wgpu.Probe(); err != nil {
		t.Skipf("cannot discover a usable GPU: %v", err)
	}

	sizes := []exactSumTimingSize{
		{rows: 100_000, degree: 100},
		{rows: 1_000_000, degree: 10},
		{rows: 10_000, degree: 1_000},
	}
	if os.Getenv("INSYRA_ACCEL_TIMING_SMALL") == "1" {
		sizes = []exactSumTimingSize{
			{rows: 1_000, degree: 10},
			{rows: 100, degree: 100},
		}
	}

	ctx := context.Background()
	for _, size := range sizes {
		rows := size.rows
		degree := size.degree
		products := rows * degree

		r := rand.New(rand.NewSource(int64(rows*31 + degree)))
		valueBits := make([]uint32, rows)
		for j := 0; j < rows; j++ {
			valueBits[j] = math.Float32bits(float32(math.Ldexp(r.NormFloat64(), r.Intn(41)-20)))
		}
		xs := make([]uint32, products)
		ys := make([]uint32, products)
		for e := 0; e < products; e++ {
			xs[e] = math.Float32bits(float32(math.Ldexp(r.NormFloat64(), r.Intn(41)-20)))
			ys[e] = valueBits[e%rows]
		}
		offsets := make([]uint32, rows+1)
		for i := 0; i <= rows; i++ {
			offsets[i] = uint32(i * degree)
		}

		deviceBits, deviceBest, err := exactSumTimingDevice(ctx, offsets, xs, ys, rows)
		if err != nil {
			t.Fatalf("rows=%d degree=%d: device exact sum: %v", rows, degree, err)
		}
		cpuOut, cpuBest, err := exactSumTimingCPU(rows, degree, products, xs, valueBits)
		if err != nil {
			t.Fatalf("rows=%d degree=%d: CPU nn.EdgeSum: %v", rows, degree, err)
		}

		mismatches := 0
		for i := 0; i < rows; i++ {
			cpu := math.Float32bits(cpuOut[i])
			if cpu == deviceBits[i] {
				continue
			}
			mismatches++
			if mismatches <= exactSumTimingMaxPrinted {
				t.Errorf("row %d of %d: cpu=%#08x device=%#08x", i, rows, cpu, deviceBits[i])
			}
		}

		deviceMS := float64(deviceBest) / float64(time.Millisecond)
		cpuMS := float64(cpuBest) / float64(time.Millisecond)
		t.Logf("rows=%d degree=%d products=%d device=%.2fms cpu=%.2fms (all %d cores) cpu/device=%.2fx device=%.0fM products/s cpu=%.0fM products/s mismatches=%d",
			rows,
			degree,
			products,
			deviceMS,
			cpuMS,
			runtime.GOMAXPROCS(0),
			cpuMS/deviceMS,
			exactSumTimingPerSecond(products, deviceBest),
			exactSumTimingPerSecond(products, cpuBest),
			mismatches,
		)
	}
}

// exactSumTimingDevice runs the device exact sum once to warm the pipeline and
// keep its single-register output, then times exactSumTimingRepeats more runs
// of the whole call and reports the shortest.
func exactSumTimingDevice(ctx context.Context, offsets, xs, ys []uint32, rows int) ([]uint32, time.Duration, error) {
	deviceBits, _, err := wgpu.RunExactSumFlatForTest(ctx, offsets, xs, ys, true)
	if err != nil {
		return nil, 0, err
	}
	if len(deviceBits) != rows {
		return nil, 0, fmt.Errorf("device exact sum returned %d outputs for %d rows", len(deviceBits), rows)
	}

	var best time.Duration
	for attempt := 0; attempt < exactSumTimingRepeats; attempt++ {
		start := time.Now()
		_, _, err := wgpu.RunExactSumFlatForTest(ctx, offsets, xs, ys, true)
		elapsed := time.Since(start)
		if err != nil {
			return nil, 0, err
		}
		if attempt == 0 || elapsed < best {
			best = elapsed
		}
	}
	return deviceBits, best, nil
}

// exactSumTimingCPU feeds the same products to nn.EdgeSum: product e is the
// incoming edge of node e/degree, its source is node e%rows, and its weight is
// the x pattern, so the CPU sums exactly the products the device sums and node
// r's output is the exact sum of row r. The graph has one node per row, not one
// per product, so nn.EdgeSum allocates and writes exactly the rows the device
// returns rather than degree times as many.
func exactSumTimingCPU(rows, degree, products int, xs, valueBits []uint32) ([]float32, time.Duration, error) {
	sources := make([]int, products)
	targets := make([]int, products)
	weights := make([]float32, products)
	values := make([]float32, rows)
	for e := 0; e < products; e++ {
		sources[e] = e % rows
		targets[e] = e / degree
		weights[e] = math.Float32frombits(xs[e])
	}
	for j := 0; j < rows; j++ {
		values[j] = math.Float32frombits(valueBits[j])
	}

	topology, err := nn.NewEdgeTopology(rows, sources, targets)
	if err != nil {
		return nil, 0, fmt.Errorf("nn.NewEdgeTopology: %w", err)
	}
	weightsTensor, err := nn.NewTensor([]int{products}, weights)
	if err != nil {
		return nil, 0, fmt.Errorf("nn.NewTensor weights: %w", err)
	}
	valuesTensor, err := nn.NewTensor([]int{rows}, values)
	if err != nil {
		return nil, 0, fmt.Errorf("nn.NewTensor values: %w", err)
	}

	result, err := nn.EdgeSum(topology, weightsTensor, valuesTensor)
	if err != nil {
		return nil, 0, err
	}
	out := result.Data()
	if len(out) < rows {
		return nil, 0, fmt.Errorf("nn.EdgeSum returned %d outputs for %d rows", len(out), rows)
	}

	var best time.Duration
	for attempt := 0; attempt < exactSumTimingRepeats; attempt++ {
		start := time.Now()
		_, err := nn.EdgeSum(topology, weightsTensor, valuesTensor)
		elapsed := time.Since(start)
		if err != nil {
			return nil, 0, err
		}
		if attempt == 0 || elapsed < best {
			best = elapsed
		}
	}
	return out, best, nil
}

// exactSumTimingPerSecond reports millions of products per second, from the
// shortest measured run.
func exactSumTimingPerSecond(products int, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return float64(products) / elapsed.Seconds() / 1e6
}
