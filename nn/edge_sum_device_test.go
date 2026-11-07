package nn

import (
	"math"
	"math/rand"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

// The zero Device is the host: To returns the topology itself and Release
// does nothing.
func TestEdgeTopologyToHostIsItself(t *testing.T) {
	top, err := NewEdgeTopology(3, []int{0, 1}, []int{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	moved, err := top.To(accel.Device{})
	if err != nil {
		t.Fatal(err)
	}
	if moved != top || moved.Device().ID != "" {
		t.Fatalf("To(host) = %p on %q, want the topology itself on the host", moved, moved.Device().ID)
	}
	top.Release()
	top.Release()
	var none *EdgeTopology
	none.Release()
	if _, err := none.To(accel.Device{}); err == nil {
		t.Fatal("To on a nil topology should fail")
	}
}

type edgeSumCase struct {
	name             string
	nodes            int
	sources, targets []int
	weights          []float32
	values           []float32
	batch            int // 0 means one-dimensional values, -1 a [0, N] batch
	upstream         []float32
}

func edgeSumCases() []edgeSumCase {
	r := rand.New(rand.NewSource(379))
	randomFloats := func(n int) []float32 {
		out := make([]float32, n)
		for i := range out {
			out[i] = float32(r.NormFloat64()) * float32(math.Ldexp(1, r.Intn(20)-10))
		}
		return out
	}
	randomEdges := func(nodes, edges int) ([]int, []int) {
		s, d := make([]int, edges), make([]int, edges)
		for e := range s {
			s[e], d[e] = r.Intn(nodes), r.Intn(nodes)
		}
		// Repeated targets, a self loop and a node with many incoming edges.
		d[0], d[1], s[2], d[2] = 3, 3, 4, 4
		for e := 3; e < edges && e < 150; e++ {
			d[e] = 5
		}
		return s, d
	}
	var cases []edgeSumCase
	s, d := randomEdges(37, 400)
	cases = append(cases, edgeSumCase{"random one row", 37, s, d, randomFloats(400), randomFloats(37), 0, randomFloats(37)})
	cases = append(cases, edgeSumCase{"random batch", 37, s, d, randomFloats(400), randomFloats(4 * 37), 4, randomFloats(4 * 37)})

	// Cancellation: products that cancel to exactly zero, and to a tiny rest.
	big := float32(math.Ldexp(1, 60))
	negZero := float32(math.Copysign(0, -1))
	cases = append(cases, edgeSumCase{"cancellation", 3,
		[]int{0, 1, 2, 0}, []int{2, 2, 2, 1},
		[]float32{big, -big, 1, negZero}, []float32{3, 3, float32(math.Ldexp(1, -100))}, 0,
		[]float32{1, -1, float32(math.Ldexp(1, -126))}})

	// Subnormal, overflow, infinity, NaN and signed zeros.
	tiny := math.Float32frombits(1)
	cases = append(cases, edgeSumCase{"special values", 6,
		[]int{0, 1, 2, 3, 4, 5, 0, 1}, []int{0, 0, 1, 1, 2, 3, 4, 5},
		[]float32{tiny, tiny, math.MaxFloat32, math.MaxFloat32, float32(math.Inf(1)), float32(math.NaN()), negZero, 0},
		[]float32{1, 1, 2, -2, 0, 1}, 0,
		[]float32{negZero, tiny, math.MaxFloat32, 2, float32(math.Inf(-1)), 0}})

	// A batch longer than the register's 64 deferred additions, so the weight
	// gradient normalises mid-sum, with many edges between one pair of nodes.
	ws, wd := make([]int, 200), make([]int, 200)
	for e := range ws {
		ws[e], wd[e] = 1, 2
	}
	cases = append(cases, edgeSumCase{"batch past 64", 4, ws, wd, randomFloats(200), randomFloats(130 * 4), 130, randomFloats(130 * 4)})

	// A batch of zero rows.
	cases = append(cases, edgeSumCase{"empty batch", 37, s, d, randomFloats(400), []float32{}, -1, []float32{}})

	// No edges, and isolated nodes in a batch.
	cases = append(cases, edgeSumCase{"no edges", 5, nil, nil, []float32{}, randomFloats(2 * 5), 2, randomFloats(2 * 5)})
	return cases
}

type edgeSumBits struct{ output, dWeights, dValues []float32 }

// runEdgeSum runs Tape.EdgeSum and its backward pass on device; the zero
// Device runs on the host.
func runEdgeSum(t *testing.T, c edgeSumCase, device accel.Device) edgeSumBits {
	t.Helper()
	top, err := NewEdgeTopology(c.nodes, c.sources, c.targets)
	if err != nil {
		t.Fatal(err)
	}
	top, err = top.To(device)
	if err != nil {
		t.Fatal(err)
	}
	defer top.Release()
	shape := []int{c.nodes}
	switch {
	case c.batch > 0:
		shape = []int{c.batch, c.nodes}
	case c.batch < 0:
		shape = []int{0, c.nodes}
	}
	tensor := func(shape []int, data []float32) *Tensor {
		x, err := NewTensor(shape, append([]float32(nil), data...))
		if err != nil {
			t.Fatal(err)
		}
		moved, err := x.To(device)
		if err != nil {
			t.Fatal(err)
		}
		return moved
	}
	weights := tensor([]int{len(c.sources)}, c.weights)
	values := tensor(shape, c.values)
	upstream := tensor(shape, c.upstream)
	defer upstream.Release()
	tape := NewTape()
	defer tape.Close()
	if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
		t.Fatal(err)
	}
	for _, p := range []*Tensor{weights, values} {
		if _, err := tape.Param(p); err != nil {
			t.Fatal(err)
		}
	}
	out, err := tape.EdgeSum(top, weights, values)
	if err != nil {
		t.Fatal(err)
	}
	if out.Device().ID != device.ID {
		t.Fatalf("output on %q, want %q", out.Device().ID, device.ID)
	}
	if err := tape.BackwardFrom(out, upstream); err != nil {
		t.Fatal(err)
	}
	host := func(x *Tensor) []float32 {
		h, err := x.ToHost()
		if err != nil {
			t.Fatal(err)
		}
		return h.data
	}
	gradOf := func(p *Tensor) []float32 {
		g, err := tape.Grad(p)
		if err != nil {
			t.Fatal(err)
		}
		defer g.Release()
		return host(g)
	}
	return edgeSumBits{output: host(out), dWeights: gradOf(weights), dValues: gradOf(values)}
}

// On a device, EdgeSum and both gradients are the host's bits.
func TestEdgeSumOnDeviceMatchesHostBits(t *testing.T) {
	device := residentTestDevice(t)
	for _, c := range edgeSumCases() {
		t.Run(c.name, func(t *testing.T) {
			want := runEdgeSum(t, c, accel.Device{})
			got := runEdgeSum(t, c, device)
			sameBitsSlice(t, "output", got.output, want.output)
			sameBitsSlice(t, "weight gradient", got.dWeights, want.dWeights)
			sameBitsSlice(t, "value gradient", got.dValues, want.dValues)
		})
	}
}

// Mixed residency and a released topology are refused with what to do.
func TestEdgeSumOnDeviceRefusals(t *testing.T) {
	device := residentTestDevice(t)
	hostTop, err := NewEdgeTopology(3, []int{0, 1}, []int{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	weights, _ := NewTensor([]int{2}, []float32{1, 2})
	values, _ := NewTensor([]int{3}, []float32{1, 2, 3})
	dw, err := weights.To(device)
	if err != nil {
		t.Fatal(err)
	}
	defer dw.Release()
	dv, err := values.To(device)
	if err != nil {
		t.Fatal(err)
	}
	defer dv.Release()
	tape := NewTape()
	defer tape.Close()
	if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
		t.Fatal(err)
	}
	if _, err := tape.EdgeSum(hostTop, dw, dv); err == nil || !strings.Contains(err.Error(), "move the topology with To") {
		t.Errorf("host topology with device tensors: %v", err)
	}
	if _, err := tape.EdgeSum(hostTop, weights, dv); err == nil {
		t.Error("host weights with device values should fail")
	}
	devTop, err := hostTop.To(device)
	if err != nil {
		t.Fatal(err)
	}
	devTop.Release()
	if _, err := tape.EdgeSum(devTop, dw, dv); err == nil || !strings.Contains(err.Error(), "released") {
		t.Errorf("released topology: %v", err)
	}
	exact := NewTape()
	defer exact.Close()
	if _, err := exact.EdgeSum(devTop, dw, dv); err == nil || !strings.Contains(err.Error(), "PrecisionFloat32Rounded") {
		t.Errorf("tape not in PrecisionFloat32Rounded: %v", err)
	}
	// Host tensors with a moved topology still run on the host.
	if _, err := tape.EdgeSum(devTop, weights, values); err != nil {
		t.Errorf("host tensors with a moved topology: %v", err)
	}
}
