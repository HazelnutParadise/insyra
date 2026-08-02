package dl

import (
	"errors"
	"testing"
)

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
