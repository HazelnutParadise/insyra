package accel

import (
	"math"
	"os"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// requireGPU keeps the hardware tests off machines that have no device.
func requireGPU(t *testing.T) {
	t.Helper()
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1 to run tests against a real GPU")
	}
}

func TestGPUSumMatchesCPUSum(t *testing.T) {
	requireGPU(t)

	const n = 1 << 20
	values := make([]any, n)
	want := 0.0
	for i := range values {
		value := float64(i%1000) * 0.5
		values[i] = value
		want += value
	}

	// No registration import and no second module: the backend registered itself
	// when this package was initialised.
	session, err := Open(Config{})
	if err != nil {
		t.Fatalf("open accel session: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if len(session.Devices()) == 0 {
		t.Skip("no GPU discovered on this host")
	}

	result, err := session.ExecuteDataList(
		insyra.NewDataList(values...).SetName("numbers"),
		WorkloadEstimate{Precision: PrecisionFloat32},
	)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if !result.Accelerated {
		t.Fatalf("expected real GPU execution, got fallback %q", result.FallbackReason)
	}

	got := result.Reductions["numbers"]
	// Only the within-workgroup tree runs in f32; the host folds partials in
	// float64, so the error is bounded by one workgroup's reduction depth.
	if relErr := math.Abs(got-want) / math.Abs(want); relErr > 1e-5 {
		t.Fatalf("GPU sum %v does not match CPU sum %v (relative error %g)", got, want, relErr)
	}
	if result.Counts["numbers"] != n {
		t.Fatalf("expected %d values folded, got %d", n, result.Counts["numbers"])
	}
	if result.BytesUploaded != uint64(n*4) {
		t.Fatalf("expected %d bytes uploaded, got %d", n*4, result.BytesUploaded)
	}
	if result.Transfer <= 0 || result.Readback <= 0 {
		t.Fatalf("expected measured cost, got transfer=%v readback=%v", result.Transfer, result.Readback)
	}
	t.Logf("device=%v transfer=%v dispatch=%v readback=%v", result.DeviceIDs, result.Transfer, result.Dispatch, result.Readback)
}

func TestGPUExecutionRefusedWithoutPrecisionOptIn(t *testing.T) {
	requireGPU(t)

	values := make([]any, 4096)
	for i := range values {
		values[i] = float64(i) + 0.5
	}

	session, err := Open(Config{})
	if err != nil {
		t.Fatalf("open accel session: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if len(session.Devices()) == 0 {
		t.Skip("no GPU discovered on this host")
	}

	result, err := session.ExecuteDataList(insyra.NewDataList(values...).SetName("numbers"), WorkloadEstimate{})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.Accelerated {
		t.Fatal("expected a float64 column to be refused without an explicit precision opt-in")
	}
	if result.FallbackReason != FallbackReasonPrecisionNotAccepted {
		t.Fatalf("expected precision-not-accepted, got %q", result.FallbackReason)
	}
}

func TestWGPUDeviceClassification(t *testing.T) {
	// No GPU needed: this is the mapping from the backend's vocabulary to the
	// runtime's, including the unified-memory case that AdapterInfo gets wrong.
	apple := wgpuDevice(wgpuInfoForTest("Apple M3", "apple", true, true))
	if apple.Backend != BackendMetal || apple.MemoryClass != MemoryClassShared || !apple.SharedMemory {
		t.Fatalf("unified Metal device misclassified: %+v", apple)
	}
	discrete := wgpuDevice(wgpuInfoForTest("NVIDIA RTX 4090", "nvidia", false, false))
	if discrete.Backend != BackendWebGPU || discrete.MemoryClass != MemoryClassDevice || discrete.SharedMemory {
		t.Fatalf("discrete device misclassified: %+v", discrete)
	}
	if apple.CapabilitySummary["float64"] {
		t.Fatal("no WebGPU backend supports float64")
	}
}
