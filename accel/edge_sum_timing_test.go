package accel_test

import (
	"math/rand"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra/accel"
	"github.com/HazelnutParadise/insyra/nn"
)

// TestRoundedEdgeSumTiming records the device kernels' time on inputs already
// on the device against nn's all-core CPU path, for milestone M36
// (engineering/decisions/0016). It
// only logs; set INSYRA_ACCEL_TIMING=1 as well to run it.
func TestRoundedEdgeSumTiming(t *testing.T) {
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" || os.Getenv("INSYRA_ACCEL_TIMING") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1 and INSYRA_ACCEL_TIMING=1")
	}
	device := accel.DefaultDevice()
	if device.ID == "" {
		t.Skip("no usable device")
	}
	best := func(run func()) time.Duration {
		run() // build kernels and warm caches
		var fastest time.Duration
		for i := 0; i < 5; i++ {
			start := time.Now()
			run()
			if d := time.Since(start); i == 0 || d < fastest {
				fastest = d
			}
		}
		return fastest
	}
	t.Logf("device %s, CPU %d cores (GOMAXPROCS %d)", device.ID, runtime.NumCPU(), runtime.GOMAXPROCS(0))
	t.Logf("%-8s %-6s %-5s | %-10s %-10s | %-10s %-10s %-10s", "nodes", "degree", "batch", "cpu fwd", "gpu fwd", "cpu bwd", "gpu dW", "gpu dV")
	for _, size := range []struct{ nodes, degree int }{
		{10000, 10}, {10000, 100}, {10000, 1000}, {100000, 10}, {100000, 100}, {1000000, 10},
	} {
		for _, batch := range []int{1, 16} {
			nodes, edges := size.nodes, size.nodes*size.degree
			rng := rand.New(rand.NewSource(int64(nodes + size.degree*7 + batch)))
			sources, targets := make([]int, edges), make([]int, edges)
			sources32, targets32 := make([]int32, edges), make([]int32, edges)
			for e := range sources {
				sources[e], targets[e] = rng.Intn(nodes), rng.Intn(nodes)
				sources32[e], targets32[e] = int32(sources[e]), int32(targets[e])
			}
			random := func(n int) []float32 {
				out := make([]float32, n)
				for i := range out {
					out[i] = float32(rng.NormFloat64())
				}
				return out
			}
			weights, values, upstream := random(edges), random(batch*nodes), random(batch*nodes)

			top, err := nn.NewEdgeTopology(nodes, sources, targets)
			if err != nil {
				t.Fatal(err)
			}
			shape := []int{batch, nodes}
			w, _ := nn.NewTensor([]int{edges}, weights)
			v, _ := nn.NewTensor(shape, values)
			u, _ := nn.NewTensor(shape, upstream)
			cpuForward := best(func() {
				if _, err := nn.EdgeSum(top, w, v); err != nil {
					t.Fatal(err)
				}
			})
			cpuBackward := best(func() {
				tape := nn.NewTape()
				defer tape.Close()
				out, err := tape.EdgeSum(top, w, v)
				if err != nil {
					t.Fatal(err)
				}
				if err := tape.BackwardFrom(out, u); err != nil {
					t.Fatal(err)
				}
			}) - cpuForward

			dt, err := accel.NewDeviceEdgeTopology(device, nodes, sources32, targets32)
			if err != nil {
				t.Fatal(err)
			}
			dw, err := accel.NewDeviceBuffer(device, weights)
			if err != nil {
				t.Fatal(err)
			}
			dv, err := accel.NewDeviceBuffer(device, values)
			if err != nil {
				t.Fatal(err)
			}
			du, err := accel.NewDeviceBuffer(device, upstream)
			if err != nil {
				t.Fatal(err)
			}
			if err := accel.SynchronizeResident(); err != nil {
				t.Fatal(err)
			}
			timed := func(op func() (*accel.DeviceBuffer, error)) time.Duration {
				return best(func() {
					out, err := op()
					if err != nil {
						t.Fatal(err)
					}
					if err := accel.SynchronizeResident(); err != nil {
						t.Fatal(err)
					}
					out.Release()
				})
			}
			gpuForward := timed(func() (*accel.DeviceBuffer, error) { return accel.RoundedEdgeSum(dt, dw, dv, batch) })
			gpuWeights := timed(func() (*accel.DeviceBuffer, error) { return accel.RoundedEdgeSumGradWeights(dt, dv, du, batch) })
			gpuValues := timed(func() (*accel.DeviceBuffer, error) { return accel.RoundedEdgeSumGradValues(dt, dw, du, batch) })
			t.Logf("%-8d %-6d %-5d | %-10s %-10s | %-10s %-10s %-10s", nodes, size.degree, batch,
				cpuForward.Round(time.Microsecond), gpuForward.Round(time.Microsecond),
				cpuBackward.Round(time.Microsecond), gpuWeights.Round(time.Microsecond), gpuValues.Round(time.Microsecond))
			dt.Release()
			dw.Release()
			dv.Release()
			du.Release()
		}
	}
}
