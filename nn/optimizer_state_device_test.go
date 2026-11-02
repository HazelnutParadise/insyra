package nn

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

func stateDeviceTape(t *testing.T, device accel.Device, weights map[string]*Tensor) (*Tape, map[string]*Parameter) {
	t.Helper()
	tape := NewTape()
	t.Cleanup(tape.Close)
	if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
		t.Fatal(err)
	}
	named := make(map[string]*Parameter)
	for _, name := range []string{"device/weight", "device/bias"} {
		v := weights[name]
		if v == nil {
			v, _ = NewTensor([]int{2}, []float32{0.25, -0.75})
		}
		p, err := tape.Param(mustTo(t, v, device))
		if err != nil {
			t.Fatal(err)
		}
		named[name] = p
	}
	return tape, named
}

func stateDeviceSteps(t *testing.T, tape *Tape, named map[string]*Parameter, optimizer string, steps int) {
	t.Helper()
	for i := 0; i < steps; i++ {
		tape.Clear()
		y, err := tape.Add(named["device/weight"].Value(), named["device/bias"].Value())
		if err != nil {
			t.Fatal(err)
		}
		if err := tape.BackwardFrom(y, y); err != nil {
			t.Fatal(err)
		}
		if optimizer == "adamw" {
			err = tape.AdamW(0.03, 0.2)
		} else {
			err = tape.Adam(0.03)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}

func stateDeviceSnapshot(t *testing.T, tape *Tape, named map[string]*Parameter) ([]byte, []byte) {
	t.Helper()
	var state, weights bytes.Buffer
	if err := tape.SaveOptimizerState(&state, named); err != nil {
		t.Fatal(err)
	}
	host := make(map[string]*Tensor)
	for name, p := range named {
		v, err := p.Value().ToHost()
		if err != nil {
			t.Fatal(err)
		}
		host[name] = v
	}
	if err := SaveSafeTensors(&weights, host); err != nil {
		t.Fatal(err)
	}
	return state.Bytes(), weights.Bytes()
}

func TestOptimizerStateDeviceRestart(t *testing.T) {
	device := residentTestDevice(t)
	for _, optimizer := range []string{"adam", "adamw"} {
		t.Run(optimizer, func(t *testing.T) {
			full, fullNamed := stateDeviceTape(t, device, nil)
			stateDeviceSteps(t, full, fullNamed, optimizer, 7)
			wantState, wantWeights := stateDeviceSnapshot(t, full, fullNamed)
			split, splitNamed := stateDeviceTape(t, device, nil)
			stateDeviceSteps(t, split, splitNamed, optimizer, 3)
			state, weights := stateDeviceSnapshot(t, split, splitNamed)
			dir := t.TempDir()
			for path, data := range map[string][]byte{"state": state, "weights": weights} {
				if err := os.WriteFile(filepath.Join(dir, path), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(executable, "-test.run=^TestOptimizerStateDeviceChild$")
			cmd.Env = append(os.Environ(), "INSYRA_OPTIMIZER_DEVICE_CHILD="+dir, "INSYRA_OPTIMIZER_KIND="+optimizer)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("new process: %v\n%s", err, out)
			}
			for path, want := range map[string][]byte{"resumed-state": wantState, "resumed-weights": wantWeights} {
				got, err := os.ReadFile(filepath.Join(dir, path))
				if err != nil || !bytes.Equal(got, want) {
					t.Fatalf("new process %s differs bit for bit: %v", path, err)
				}
			}
			// The same rounded checkpoint is portable to a host tape.
			hostWeights, err := LoadSafeTensors(bytes.NewReader(weights))
			if err != nil {
				t.Fatal(err)
			}
			host, hostNamed := stateDeviceTape(t, accel.Device{}, hostWeights)
			if err := host.LoadOptimizerState(bytes.NewReader(state), hostNamed); err != nil {
				t.Fatal(err)
			}
			stateDeviceSteps(t, host, hostNamed, optimizer, 4)
			gotState, gotWeights := stateDeviceSnapshot(t, host, hostNamed)
			if !bytes.Equal(gotState, wantState) || !bytes.Equal(gotWeights, wantWeights) {
				t.Fatal("device-to-host resume differs bit for bit")
			}
		})
	}
}

func TestOptimizerStateDeviceChild(t *testing.T) {
	dir := os.Getenv("INSYRA_OPTIMIZER_DEVICE_CHILD")
	if dir == "" {
		t.Skip("subprocess helper")
	}
	device := residentTestDevice(t)
	weightBytes, err := os.ReadFile(filepath.Join(dir, "weights"))
	if err != nil {
		t.Fatal(err)
	}
	weights, err := LoadSafeTensors(bytes.NewReader(weightBytes))
	if err != nil {
		t.Fatal(err)
	}
	tape, named := stateDeviceTape(t, device, weights)
	stateBytes, err := os.ReadFile(filepath.Join(dir, "state"))
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.LoadOptimizerState(bytes.NewReader(stateBytes), named); err != nil {
		t.Fatal(err)
	}
	stateDeviceSteps(t, tape, named, os.Getenv("INSYRA_OPTIMIZER_KIND"), 4)
	state, values := stateDeviceSnapshot(t, tape, named)
	for path, data := range map[string][]byte{"resumed-state": state, "resumed-weights": values} {
		if err := os.WriteFile(filepath.Join(dir, path), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestOptimizerStateDeviceAllocationFailureIsAtomic(t *testing.T) {
	device := residentTestDevice(t)
	baseline := accel.ResidentBytes(device)
	source, sourceNamed := stateDeviceTape(t, device, nil)
	stateDeviceSteps(t, source, sourceNamed, "adamw", 3)
	state, _ := stateDeviceSnapshot(t, source, sourceNamed)
	beforeTarget := accel.ResidentBytes(device)
	limited := device
	// Two parameters + four moments = 48 bytes. Staging another four
	// moments gets through the first parameter and fails on the second.
	limited.BudgetBytes = beforeTarget + 48 + 24
	target, targetNamed := stateDeviceTape(t, limited, nil)
	stateDeviceSteps(t, target, targetNamed, "adam", 1)
	target.Clear()
	oldState, oldWeights := stateDeviceSnapshot(t, target, targetNamed)
	oldBytes := accel.ResidentBytes(device)
	if err := target.LoadOptimizerState(bytes.NewReader(state), targetNamed); err == nil {
		t.Fatal("staged allocation should exceed the resident budget")
	}
	newState, newWeights := stateDeviceSnapshot(t, target, targetNamed)
	if !bytes.Equal(oldState, newState) || !bytes.Equal(oldWeights, newWeights) || accel.ResidentBytes(device) != oldBytes {
		t.Fatal("failed import changed old state or leaked staged buffers")
	}
	target.Close()
	source.Close()
	if accel.ResidentBytes(device) != baseline {
		t.Fatal("checkpoint lifecycle leaked device memory")
	}
}
