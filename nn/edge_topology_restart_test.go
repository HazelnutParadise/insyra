package nn

import (
	"bytes"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

// recurrentCheckpoint is what a resident recurrent run saves at a completed
// step: its topology, its per-edge weights, the state it carries to the next
// step, and the Adam moments.
type recurrentCheckpoint struct {
	topology, weights, state, optimizer []byte
}

func (c recurrentCheckpoint) files() map[string][]byte {
	return map[string][]byte{"topology": c.topology, "weights": c.weights, "state": c.state, "optimizer": c.optimizer}
}

// recurrentRun is a sparse recurrent network on device: each training step
// unrolls s = tanh(EdgeSum(W, s) + input) twice from the carried state, takes
// the reverse pass and an Adam step, and carries the last s to the next step.
type recurrentRun struct {
	t      *testing.T
	tape   *Tape
	graph  *EdgeTopology
	weight *Parameter
	input  *Tensor
	carry  *Tensor
	named  map[string]*Parameter
}

func recurrentFixture(t *testing.T) (*EdgeTopology, *Tensor, *Tensor, *Tensor) {
	t.Helper()
	const nodes, degree, batch = 64, 3, 2
	rng := rand.New(rand.NewSource(39))
	sources, targets := make([]int, nodes*degree), make([]int, nodes*degree)
	for e := range sources {
		sources[e], targets[e] = rng.Intn(nodes), e/degree
	}
	graph, err := NewEdgeTopology(nodes, sources, targets)
	if err != nil {
		t.Fatal(err)
	}
	return graph, randomTensor(t, rng, nodes*degree), randomTensor(t, rng, batch, nodes), randomTensor(t, rng, batch, nodes)
}

func newRecurrentRun(t *testing.T, device accel.Device, graph *EdgeTopology, weights, state, input *Tensor) *recurrentRun {
	t.Helper()
	moved, err := graph.To(device)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(moved.Release)
	tape := NewTape()
	t.Cleanup(tape.Close)
	if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
		t.Fatal(err)
	}
	weight, err := tape.Param(mustTo(t, weights, device))
	if err != nil {
		t.Fatal(err)
	}
	in := mustTo(t, input, device)
	t.Cleanup(in.Release)
	run := &recurrentRun{t: t, tape: tape, graph: moved, weight: weight, input: in, carry: mustTo(t, state, device), named: map[string]*Parameter{"recurrent/weight": weight}}
	t.Cleanup(func() { run.carry.Release() })
	return run
}

func (r *recurrentRun) steps(n int) {
	t := r.t
	for i := 0; i < n; i++ {
		r.tape.Clear()
		s := r.carry
		for k := 0; k < 2; k++ {
			e, err := r.tape.EdgeSum(r.graph, r.weight.Value(), s)
			if err != nil {
				t.Fatal(err)
			}
			if e, err = r.tape.Add(e, r.input); err != nil {
				t.Fatal(err)
			}
			if s, err = r.tape.Tanh(e); err != nil {
				t.Fatal(err)
			}
		}
		if err := r.tape.BackwardFrom(s, s); err != nil {
			t.Fatal(err)
		}
		if err := r.tape.Adam(0.01); err != nil {
			t.Fatal(err)
		}
		// The tape releases s at the next Clear, so the carried state is a
		// copy the run owns.
		host, err := s.ToHost()
		if err != nil {
			t.Fatal(err)
		}
		next := mustTo(t, host, r.carry.Device())
		r.carry.Release()
		r.carry = next
	}
}

func (r *recurrentRun) save() recurrentCheckpoint {
	t := r.t
	var topology, weights, state, optimizer bytes.Buffer
	if err := SaveEdgeTopology(&topology, r.graph); err != nil {
		t.Fatal(err)
	}
	w, err := r.weight.Value().ToHost()
	if err != nil {
		t.Fatal(err)
	}
	s, err := r.carry.ToHost()
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveSafeTensors(&weights, map[string]*Tensor{"recurrent/weight": w}); err != nil {
		t.Fatal(err)
	}
	if err := SaveSafeTensors(&state, map[string]*Tensor{"recurrent/state": s}); err != nil {
		t.Fatal(err)
	}
	if err := r.tape.SaveOptimizerState(&optimizer, r.named); err != nil {
		t.Fatal(err)
	}
	return recurrentCheckpoint{topology.Bytes(), weights.Bytes(), state.Bytes(), optimizer.Bytes()}
}

// resumeRecurrentRun restores a checkpoint in the documented order: the
// topology, a check that the caller's rebuilt graph is the same, the weights
// and state, then the optimizer state.
func resumeRecurrentRun(t *testing.T, device accel.Device, c recurrentCheckpoint, rebuilt *EdgeTopology, input *Tensor) *recurrentRun {
	t.Helper()
	graph, err := LoadEdgeTopology(bytes.NewReader(c.topology))
	if err != nil {
		t.Fatal(err)
	}
	if !graph.Equal(rebuilt) {
		t.Fatal("the rebuilt graph is not the checkpoint's")
	}
	weights, err := LoadSafeTensors(bytes.NewReader(c.weights))
	if err != nil {
		t.Fatal(err)
	}
	state, err := LoadSafeTensors(bytes.NewReader(c.state))
	if err != nil {
		t.Fatal(err)
	}
	run := newRecurrentRun(t, device, graph, weights["recurrent/weight"], state["recurrent/state"], input)
	if err := run.tape.LoadOptimizerState(bytes.NewReader(c.optimizer), run.named); err != nil {
		t.Fatal(err)
	}
	return run
}

func TestResidentTopologySavesItsHostBytes(t *testing.T) {
	device := residentTestDevice(t)
	g := edgeTopologyFixture(t)
	moved, err := g.To(device)
	if err != nil {
		t.Fatal(err)
	}
	defer moved.Release()
	if !bytes.Equal(savedEdgeTopology(t, moved), savedEdgeTopology(t, g)) || !moved.Equal(g) {
		t.Fatal("a topology on the device saves or compares differently from its host original")
	}
}

// TestResidentRecurrentRunResumesInANewProcess trains seven steps in one
// process, and three steps, a checkpoint and four more steps in a new process,
// and requires the same bits; then resumes the same checkpoint on the host.
func TestResidentRecurrentRunResumesInANewProcess(t *testing.T) {
	device := residentTestDevice(t)
	graph, weights, state, input := recurrentFixture(t)

	full := newRecurrentRun(t, device, graph, weights, state, input)
	full.steps(7)
	want := full.save()

	split := newRecurrentRun(t, device, graph, weights, state, input)
	split.steps(3)
	checkpoint := split.save()
	dir := t.TempDir()
	for name, data := range checkpoint.files() {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, "-test.run=^TestResidentRecurrentRunChild$")
	cmd.Env = append(os.Environ(), "INSYRA_RECURRENT_CHILD="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("new process: %v\n%s", err, out)
	}
	for name, data := range want.files() {
		got, err := os.ReadFile(filepath.Join(dir, "resumed-"+name))
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("new process %s differs bit for bit: %v", name, err)
		}
	}

	// The same checkpoint resumes on the host with the same bits.
	host := resumeRecurrentRun(t, accel.Device{}, checkpoint, graph, input)
	host.steps(4)
	for name, data := range host.save().files() {
		if !bytes.Equal(data, want.files()[name]) {
			t.Fatalf("device-to-host resume: %s differs bit for bit", name)
		}
	}
}

func TestResidentRecurrentRunChild(t *testing.T) {
	dir := os.Getenv("INSYRA_RECURRENT_CHILD")
	if dir == "" {
		t.Skip("subprocess helper")
	}
	device := residentTestDevice(t)
	read := func(name string) []byte {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	checkpoint := recurrentCheckpoint{read("topology"), read("weights"), read("state"), read("optimizer")}
	graph, _, _, input := recurrentFixture(t) // the caller rebuilds its graph and input from its own source
	run := resumeRecurrentRun(t, device, checkpoint, graph, input)
	run.steps(4)
	for name, data := range run.save().files() {
		if err := os.WriteFile(filepath.Join(dir, "resumed-"+name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
