package wgpu

import (
	"math"
	"os"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/accel"
	"github.com/gogpu/gputypes"
)

// requireGPU keeps the hardware tests off machines that have no device. CI and
// contributors without a GPU still get the unit tests below.
func requireGPU(t *testing.T) {
	t.Helper()
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1 to run tests against a real GPU")
	}
	if _, err := acquire(); err != nil {
		t.Skipf("no usable GPU on this host: %v", err)
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

	session, err := accel.Open(accel.Config{})
	if err != nil {
		t.Fatalf("open accel session: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	if len(session.Devices()) == 0 {
		t.Fatal("expected the wgpu probe to report a device")
	}

	result, err := session.ExecuteDataList(
		insyra.NewDataList(values...).SetName("numbers"),
		accel.WorkloadEstimate{Precision: accel.PrecisionFloat32},
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
	if relativeError(got, want) > 1e-5 {
		t.Fatalf("GPU sum %v does not match CPU sum %v (relative error %g)", got, want, relativeError(got, want))
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

	session, err := accel.Open(accel.Config{})
	if err != nil {
		t.Fatalf("open accel session: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	result, err := session.ExecuteDataList(insyra.NewDataList(values...).SetName("numbers"), accel.WorkloadEstimate{})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.Accelerated {
		t.Fatal("expected a float64 column to be refused without an explicit precision opt-in")
	}
	if result.FallbackReason != accel.FallbackReasonPrecisionNotAccepted {
		t.Fatalf("expected precision-not-accepted, got %q", result.FallbackReason)
	}
}

func TestGPUSumHandlesColumnLargerThanOneDispatch(t *testing.T) {
	requireGPU(t)

	h, err := acquire()
	if err != nil {
		t.Fatalf("acquire device: %v", err)
	}
	// Force chunking without allocating a multi-hundred-megabyte column by
	// checking the boundary arithmetic directly against a small column.
	chunk := h.maxElementsPerChunk()
	if chunk%elemsPerGroup != 0 {
		t.Fatalf("chunk size %d must be a whole number of workgroups", chunk)
	}
	if chunk > maxElemsPerDispatch {
		t.Fatalf("chunk size %d exceeds the workgroup-count limit %d", chunk, maxElemsPerDispatch)
	}

	// A column that spans several workgroups still reduces correctly.
	const n = elemsPerGroup*3 + 7
	values := make([]float32, n)
	want := 0.0
	for i := range values {
		values[i] = float32(i % 97)
		want += float64(values[i])
	}
	pipeline, layout, err := h.computePipeline()
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}
	got, cost, err := h.sumChunk(t.Context(), pipeline, layout, values)
	if err != nil {
		t.Fatalf("sum chunk: %v", err)
	}
	if relativeError(got, want) > 1e-6 {
		t.Fatalf("partial sum %v does not match %v", got, want)
	}
	if cost.uploaded != uint64(n*4) {
		t.Fatalf("expected %d bytes uploaded, got %d", n*4, cost.uploaded)
	}
}

func TestSoftwareAdapterIsNotAnAccelerationDevice(t *testing.T) {
	cases := []struct {
		name string
		info gputypes.AdapterInfo
		want bool
	}{
		{"cpu device type", gputypes.AdapterInfo{Name: "Whatever", DeviceType: gputypes.DeviceTypeCPU}, true},
		{"software renderer by name", gputypes.AdapterInfo{Name: "Software Renderer", DeviceType: gputypes.DeviceTypeOther}, true},
		{"real gpu", gputypes.AdapterInfo{Name: "Apple M3", DeviceType: gputypes.DeviceTypeDiscreteGPU}, false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := isSoftwareAdapter(testCase.info); got != testCase.want {
				t.Fatalf("isSoftwareAdapter(%+v) = %v, want %v", testCase.info, got, testCase.want)
			}
		})
	}
}

func TestSharedMemoryIgnoresTheReportedDeviceType(t *testing.T) {
	// gogpu reports an Apple M3 as DiscreteGPU, which would put a unified-memory
	// device in the discrete VRAM class.
	m3 := gputypes.AdapterInfo{
		Name:       "Apple M3",
		Vendor:     "Apple",
		Backend:    gputypes.BackendMetal,
		DeviceType: gputypes.DeviceTypeDiscreteGPU,
	}
	if !sharedMemory(m3) {
		t.Fatal("Apple Silicon has unified memory and must be classified as shared")
	}

	nvidia := gputypes.AdapterInfo{
		Name:       "NVIDIA GeForce RTX 4090",
		Vendor:     "NVIDIA",
		Backend:    gputypes.BackendVulkan,
		DeviceType: gputypes.DeviceTypeDiscreteGPU,
	}
	if sharedMemory(nvidia) {
		t.Fatal("a discrete NVIDIA GPU must not be classified as shared memory")
	}
}

func relativeError(got, want float64) float64 {
	if want == 0 {
		return math.Abs(got)
	}
	return math.Abs(got-want) / math.Abs(want)
}
