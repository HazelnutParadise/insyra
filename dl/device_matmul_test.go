package dl

import (
	"errors"
	"math"
	"os"
	"sync/atomic"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/accel"
)

func TestDeviceMatMulIsBitEqualToCPUOnHardware(t *testing.T) {
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
	a := patternedTestData(rows * inner)
	b := patternedTestData(inner * cols)
	left := mustTestTensor(t, []int{rows, inner}, a)
	right := mustTestTensor(t, []int{inner, cols}, b)
	cpu, err := matMul2DWithWorkers(left, right, 1)
	if err != nil {
		t.Fatalf("CPU MatMul: %v", err)
	}
	got, err := MatMul(left, right)
	if err != nil {
		t.Fatalf("device MatMul: %v", err)
	}
	if report := accel.Default().Report(); !report.Accelerated {
		if report.FallbackReason == accel.FallbackReasonNoAccelerator {
			t.Skipf("no usable GPU: %s", report.FallbackReason)
		}
		t.Fatalf("default MatMul did not use the device: %s", report.FallbackReason)
	}
	if len(got.Data()) != len(cpu.data) {
		t.Fatalf("device result length = %d, CPU length = %d", len(got.Data()), len(cpu.data))
	}
	for i, value := range got.Data() {
		if value != cpu.data[i] {
			t.Fatalf("bit parity failed at %d: device=%08x cpu=%08x", i, math.Float32bits(value), math.Float32bits(cpu.data[i]))
		}
	}
}

func TestDeviceMatMulHookNilAndErrorKeepCPUResult(t *testing.T) {
	t.Cleanup(func() { RegisterDeviceMatMul(nil) })
	a := mustTestTensor(t, []int{256, 512}, patternedTestData(256*512))
	b := mustTestTensor(t, []int{512, 256}, patternedTestData(512*256))
	cpu, err := matMul2DWithWorkers(a, b, 1)
	if err != nil {
		t.Fatalf("CPU MatMul: %v", err)
	}

	RegisterDeviceMatMul(nil)
	got, err := MatMul(a, b)
	if err != nil {
		t.Fatalf("nil-hook MatMul: %v", err)
	}
	assertExactTensorEqual(t, got, cpu)

	called := 0
	RegisterDeviceMatMul(func([]float32, int, int, []float32, int, int) ([]float32, error) {
		called++
		return nil, errors.New("device unavailable")
	})
	got, err = MatMul(a, b)
	if err != nil {
		t.Fatalf("error-hook MatMul: %v", err)
	}
	if called != 1 {
		t.Fatalf("error hook calls = %d, want 1", called)
	}
	assertExactTensorEqual(t, got, cpu)
}

func TestDeviceMatMulConfigSwitchesDeviceHook(t *testing.T) {
	previous := insyra.Config.GetAccelerationEnabled()
	t.Cleanup(func() {
		insyra.Config.SetAcceleration(previous)
		RegisterDeviceMatMul(nil)
	})

	a := mustTestTensor(t, []int{256, 512}, patternedTestData(256*512))
	b := mustTestTensor(t, []int{512, 256}, patternedTestData(512*256))
	cpu, err := matMul2DWithWorkers(a, b, 1)
	if err != nil {
		t.Fatalf("CPU MatMul: %v", err)
	}

	var calls atomic.Int32
	RegisterDeviceMatMul(func(_ []float32, aRows, _ int, _ []float32, _, bCols int) ([]float32, error) {
		calls.Add(1)
		return append([]float32(nil), cpu.data...), nil
	})

	insyra.Config.SetAcceleration(false)
	got, err := MatMul(a, b)
	if err != nil {
		t.Fatalf("Config-disabled MatMul: %v", err)
	}
	assertExactTensorEqual(t, got, cpu)
	if got := calls.Load(); got != 0 {
		t.Fatalf("disabled device hook calls = %d, want 0", got)
	}

	insyra.Config.SetAcceleration(true)
	got, err = MatMul(a, b)
	if err != nil {
		t.Fatalf("re-enabled MatMul: %v", err)
	}
	assertExactTensorEqual(t, got, cpu)
	if got := calls.Load(); got != 1 {
		t.Fatalf("re-enabled device hook calls = %d, want 1", got)
	}
}

func TestDeviceMatMulHookNeverSeesSubFloorOrBatched(t *testing.T) {
	t.Cleanup(func() { RegisterDeviceMatMul(nil) })
	calls := 0
	RegisterDeviceMatMul(func(a []float32, aRows, _ int, _ []float32, _, bCols int) ([]float32, error) {
		calls++
		return make([]float32, aRows*bCols), nil
	})

	smallA := mustTestTensor(t, []int{32, 32}, patternedTestData(32*32))
	smallB := mustTestTensor(t, []int{32, 32}, patternedTestData(32*32))
	if _, err := MatMul(smallA, smallB); err != nil {
		t.Fatalf("sub-floor MatMul: %v", err)
	}
	batchedA := mustTestTensor(t, []int{2, 32, 32}, patternedTestData(2*32*32))
	batchedB := mustTestTensor(t, []int{2, 32, 32}, patternedTestData(2*32*32))
	if _, err := MatMul(batchedA, batchedB); err != nil {
		t.Fatalf("batched MatMul: %v", err)
	}
	if calls != 0 {
		t.Fatalf("hook calls for ineligible shapes = %d, want 0", calls)
	}
}

func TestRegisterDeviceMatMulReplacesAndClears(t *testing.T) {
	t.Cleanup(func() { RegisterDeviceMatMul(nil) })
	a := mustTestTensor(t, []int{256, 512}, patternedTestData(256*512))
	b := mustTestTensor(t, []int{512, 256}, patternedTestData(512*256))
	makeHook := func(value float32) DeviceMatMul {
		return func(_ []float32, aRows, _ int, _ []float32, _, bCols int) ([]float32, error) {
			data := make([]float32, aRows*bCols)
			for i := range data {
				data[i] = value
			}
			return data, nil
		}
	}

	RegisterDeviceMatMul(makeHook(7))
	got, err := MatMul(a, b)
	if err != nil {
		t.Fatalf("first hook MatMul: %v", err)
	}
	if got.data[0] != 7 {
		t.Fatalf("first hook result = %v, want 7", got.data[0])
	}
	RegisterDeviceMatMul(makeHook(8))
	got, err = MatMul(a, b)
	if err != nil {
		t.Fatalf("replacement hook MatMul: %v", err)
	}
	if got.data[0] != 8 {
		t.Fatalf("replacement hook result = %v, want 8", got.data[0])
	}
	RegisterDeviceMatMul(nil)
	cpu, err := matMul2DWithWorkers(a, b, 1)
	if err != nil {
		t.Fatalf("CPU MatMul after clear: %v", err)
	}
	got, err = MatMul(a, b)
	if err != nil {
		t.Fatalf("cleared hook MatMul: %v", err)
	}
	assertExactTensorEqual(t, got, cpu)
}

func TestDeviceMatMulDisabledWGPUKeepsCPUResult(t *testing.T) {
	previous := insyra.Config.GetAccelerationEnabled()
	t.Cleanup(func() { insyra.Config.SetAcceleration(previous) })
	insyra.Config.SetAcceleration(true)
	t.Setenv("INSYRA_ACCEL_DISABLE_WGPU", "1")
	accel.ResetDefaultForTest()
	t.Cleanup(accel.ResetDefaultForTest)
	t.Cleanup(func() { RegisterDeviceMatMul(nil) })
	RegisterDeviceMatMul(accel.DeviceMatMul)

	a := mustTestTensor(t, []int{256, 512}, patternedTestData(256*512))
	b := mustTestTensor(t, []int{512, 256}, patternedTestData(512*256))
	want, err := matMul2DWithWorkers(a, b, 1)
	if err != nil {
		t.Fatalf("CPU MatMul: %v", err)
	}
	got, err := MatMul(a, b)
	if err != nil {
		t.Fatalf("disabled-device MatMul: %v", err)
	}
	assertExactTensorEqual(t, got, want)
	if report := accel.Default().Report(); report.Accelerated {
		t.Fatal("disabled WGPU path reported acceleration")
	}
}
