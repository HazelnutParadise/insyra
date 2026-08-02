package dlbridge

import (
	"errors"
	"math"
	"os"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
	"github.com/HazelnutParadise/insyra/accel/internal/wgpu"
	"github.com/HazelnutParadise/insyra/dl"
)

func TestDLMatMulDeviceIsBitEqualToCPUOnHardware(t *testing.T) {
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	session := accel.Default()
	if len(session.Devices()) == 0 {
		t.Skipf("no usable GPU: %s", session.Report().FallbackReason)
	}

	const (
		rows  = 256
		inner = 512
		cols  = 256
	)
	a := make([]float32, rows*inner)
	b := make([]float32, inner*cols)
	for i := range a {
		a[i] = float32((i*31+11*17)%101-50) * 0.01
	}
	for i := range b {
		b[i] = float32((i*31+37*17)%101-50) * 0.01
	}

	device, err := matmul(a, rows, inner, b, inner, cols)
	if err != nil {
		if errors.Is(err, wgpu.ErrUnavailable) {
			t.Skipf("no usable GPU: %v", err)
		}
		t.Fatalf("device MatMul: %v", err)
	}
	cpu := make([]float32, rows*cols)
	for row := 0; row < rows; row++ {
		for column := 0; column < cols; column++ {
			var sum float32
			for k := 0; k < inner; k++ {
				sum += a[row*inner+k] * b[k*cols+column]
			}
			cpu[row*cols+column] = sum
		}
	}
	if len(device) != len(cpu) {
		t.Fatalf("device result length = %d, CPU length = %d", len(device), len(cpu))
	}
	for i := range cpu {
		if device[i] != cpu[i] {
			t.Fatalf("bit parity failed at %d: device=%08x cpu=%08x", i, math.Float32bits(device[i]), math.Float32bits(cpu[i]))
		}
	}

	// The bridge is the registered dl hook too. Its CPU fallback remains owned
	// by dl, so this call verifies the public dispatch seam is live on hardware.
	got, err := dl.MatMul(mustTensor(t, []int{rows, inner}, a), mustTensor(t, []int{inner, cols}, b))
	if err != nil {
		t.Fatalf("dl MatMul through bridge: %v", err)
	}
	for i, value := range got.Data() {
		if value != cpu[i] {
			t.Fatalf("dl parity failed at %d: got=%08x cpu=%08x", i, math.Float32bits(value), math.Float32bits(cpu[i]))
		}
	}
}

func TestStrictGPUModeStillFailsWithoutDevice(t *testing.T) {
	t.Setenv("INSYRA_ACCEL_DISABLE_NATIVE_PROBES", "1")
	t.Setenv("INSYRA_ACCEL_DISABLE_WGPU", "1")
	session, err := accel.Open(accel.Config{Mode: accel.ModeStrictGPU})
	if err == nil {
		t.Fatal("strict GPU mode returned nil error without a device")
	}
	if session == nil {
		t.Fatal("strict GPU mode did not return its session report")
	}
	if got := session.Report().FallbackReason; got != accel.FallbackReasonStrictGPUUnavailable {
		t.Fatalf("strict GPU fallback reason = %q, want %q", got, accel.FallbackReasonStrictGPUUnavailable)
	}
}

func mustTensor(t *testing.T, shape []int, data []float32) *dl.Tensor {
	t.Helper()
	tensor, err := dl.NewTensor(shape, data)
	if err != nil {
		t.Fatalf("tensor: %v", err)
	}
	return tensor
}
