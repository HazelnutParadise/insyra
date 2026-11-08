package nn

import (
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra/accel"
)

// TestRecurrentStepTiming times a training step of a sparse recurrent network
// unrolled over a few steps, s = tanh(EdgeSum(W, s) + input), then the reverse
// pass and SGD: on the default tape (host, every core), on a
// PrecisionFloat32Rounded tape on the host, and on the same tape with every
// tensor and the topology on the device. It is M38's measurement of whether a
// resident recurrent step beats the all-core CPU. It only logs; set
// INSYRA_ACCEL_TIMING=1 to run it, and INSYRA_ACCEL_GPU_TESTS=1 for the device
// column.
// deviceFailure carries an error out of a device run so the table can go on.
type deviceFailure struct{ err error }

func TestRecurrentStepTiming(t *testing.T) {
	if os.Getenv("INSYRA_ACCEL_TIMING") != "1" {
		t.Skip("set INSYRA_ACCEL_TIMING=1: this is a measurement, not a correctness check")
	}
	var device accel.Device
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") == "1" {
		device = accel.DefaultDevice()
	}
	const unroll, repeats = 4, 3
	t.Logf("unroll %d, best of %d after one warm-up; device %q", unroll, repeats, device.ID)
	t.Logf("%-8s %-6s %-5s | %-12s %-12s %-12s", "nodes", "degree", "batch", "host default", "host rounded", "device")
	for _, size := range [][3]int{{10000, 10, 1}, {10000, 10, 16}, {100000, 10, 1}, {100000, 10, 16}, {100000, 100, 16}, {1000000, 10, 16}} {
		nodes, degree, batch := size[0], size[1], size[2]
		rng := rand.New(rand.NewSource(int64(nodes + degree + batch)))
		edges := nodes * degree
		sources, targets := make([]int, edges), make([]int, edges)
		for i := range sources {
			sources[i] = rng.Intn(nodes)
			targets[i] = i / degree
		}
		graph, err := NewEdgeTopology(nodes, sources, targets)
		if err != nil {
			t.Fatal(err)
		}
		tensor := func(scale float64, shape ...int) *Tensor {
			n := 1
			for _, d := range shape {
				n *= d
			}
			data := make([]float32, n)
			for i := range data {
				data[i] = float32(rng.NormFloat64() * scale)
			}
			x, err := NewTensor(shape, data)
			if err != nil {
				t.Fatal(err)
			}
			return x
		}
		weights := tensor(1/float64(degree), edges)
		state := tensor(0.5, batch, nodes)
		input := tensor(0.5, batch, nodes)
		upstream := tensor(1, batch, nodes)

		run := func(on accel.Device, precision accel.Precision) (best time.Duration, failure error) {
			fail := func(err error) {
				if on.ID == "" {
					t.Fatal(err)
				}
				panic(deviceFailure{err})
			}
			if on.ID != "" {
				defer func() {
					if r := recover(); r != nil {
						f, ok := r.(deviceFailure)
						if !ok {
							panic(r)
						}
						best, failure = 0, f.err
					}
				}()
			}
			g, err := graph.To(on)
			if err != nil {
				fail(err)
			}
			defer g.Release()
			move := func(x *Tensor) *Tensor {
				m, err := x.To(on)
				if err != nil {
					fail(err)
				}
				return m
			}
			w, s0, in, up := move(weights), move(state), move(input), move(upstream)
			defer s0.Release()
			defer in.Release()
			defer up.Release()
			tape := NewTape()
			defer tape.Close()
			if precision != "" {
				if err := tape.SetPrecision(precision); err != nil {
					fail(err)
				}
			}
			if _, err := tape.Param(w); err != nil {
				fail(err)
			}
			step := func() {
				tape.Clear()
				s := s0
				for i := 0; i < unroll; i++ {
					e, err := tape.EdgeSum(g, w, s)
					if err != nil {
						fail(err)
					}
					if e, err = tape.Add(e, in); err != nil {
						fail(err)
					}
					if s, err = tape.Tanh(e); err != nil {
						fail(err)
					}
				}
				if err := tape.BackwardFrom(s, up); err != nil {
					fail(err)
				}
				if err := tape.SGD(1e-3); err != nil {
					fail(err)
				}
				if on.ID != "" {
					if err := accel.SynchronizeResident(); err != nil {
						fail(err)
					}
				}
			}
			step()
			best = time.Duration(1<<63 - 1)
			for r := 0; r < repeats; r++ {
				start := time.Now()
				step()
				if d := time.Since(start); d < best {
					best = d
				}
			}
			return best, nil
		}
		def, _ := run(accel.Device{}, "") // the default tape
		rounded, _ := run(accel.Device{}, accel.PrecisionFloat32Rounded)
		dev := "-"
		if device.ID != "" {
			// A card too small for the row reports it instead of ending the table.
			if d, err := run(device, accel.PrecisionFloat32Rounded); err != nil {
				dev = "failed"
				t.Logf("device run of %d×%d×%d: %v", nodes, degree, batch, err)
			} else {
				dev = d.String()
			}
		}
		t.Logf("%-8d %-6d %-5d | %-12s %-12s %-12s", nodes, degree, batch, def, rounded, dev)
	}
}
