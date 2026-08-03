package wgpu_test

import (
	"context"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra/accel/internal/wgpu"
	dl "github.com/HazelnutParadise/insyra/dl"
)

type matmulMeasurementShape struct {
	name   string
	batch  int
	aShape [2]int
	bShape [2]int
}

func TestDLMatMulDeviceMeasurement(t *testing.T) {
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	if _, err := wgpu.Probe(); err != nil {
		t.Skipf("cannot discover a usable GPU: %v", err)
	}

	shapes := []matmulMeasurementShape{
		{name: "[4096,256]x[256,256]", aShape: [2]int{4096, 256}, bShape: [2]int{256, 256}},
		{name: "[4096,256]x[256,1024]", aShape: [2]int{4096, 256}, bShape: [2]int{256, 1024}},
		{name: "[4096,1024]x[1024,256]", aShape: [2]int{4096, 1024}, bShape: [2]int{1024, 256}},
		{name: "128x([128,64]x[64,128])", batch: 128, aShape: [2]int{128, 64}, bShape: [2]int{64, 128}},
		{name: "128x([128,128]x[128,64])", batch: 128, aShape: [2]int{128, 128}, bShape: [2]int{128, 64}},
		{name: "[4096,4096]x[4096,4096]", aShape: [2]int{4096, 4096}, bShape: [2]int{4096, 4096}},
	}

	for _, shape := range shapes {
		shape := shape
		t.Run(shape.name, func(t *testing.T) {
			a, b, err := measurementMatmulInputs(shape)
			if err != nil {
				t.Fatalf("inputs: %v", err)
			}
			cpuWant, cpuMS, err := measurementMatmulCPU(shape, a, b)
			if err != nil {
				t.Fatalf("CPU MatMul: %v", err)
			}
			deviceGot, deviceMS, err := measurementMatmulDeviceBestOfFive(t, shape, a, b)
			if err != nil {
				t.Fatalf("device MatMul: %v", err)
			}
			if len(deviceGot) != len(cpuWant) {
				t.Fatalf("device returned %d values, CPU returned %d", len(deviceGot), len(cpuWant))
			}
			maxAbs, maxULP := measurementDeviation(cpuWant, deviceGot)
			fmt.Printf("shape=%s device_ms=%.3f cpu_ms=%.3f ratio=%.3f maxAbs=%.9g maxULP=%d\n",
				shape.name, deviceMS, cpuMS, deviceMS/cpuMS, maxAbs, maxULP)
		})
	}
}

// TestDLMatMulDeviceFloorLadder is the operator-runnable crossover ladder.
// It spans the measured loss near 1M MACs and win near 268M MACs; the floor in
// dl remains provisional until this ladder is run on the target hardware.
func TestDLMatMulDeviceFloorLadder(t *testing.T) {
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	if _, err := wgpu.Probe(); err != nil {
		t.Skipf("cannot discover a usable GPU: %v", err)
	}

	shapes := []matmulMeasurementShape{
		{name: "1M", aShape: [2]int{32, 1024}, bShape: [2]int{1024, 32}},
		{name: "2M", aShape: [2]int{64, 1024}, bShape: [2]int{1024, 32}},
		{name: "4M", aShape: [2]int{64, 1024}, bShape: [2]int{1024, 64}},
		{name: "8M", aShape: [2]int{128, 1024}, bShape: [2]int{1024, 64}},
		{name: "16M", aShape: [2]int{128, 1024}, bShape: [2]int{1024, 128}},
		{name: "32M", aShape: [2]int{256, 1024}, bShape: [2]int{1024, 128}},
		{name: "64M", aShape: [2]int{256, 1024}, bShape: [2]int{1024, 256}},
		{name: "128M", aShape: [2]int{512, 1024}, bShape: [2]int{1024, 256}},
		{name: "268M", aShape: [2]int{512, 1024}, bShape: [2]int{1024, 512}},
	}

	for _, shape := range shapes {
		shape := shape
		t.Run(shape.name, func(t *testing.T) {
			a, b, err := measurementMatmulInputs(shape)
			if err != nil {
				t.Fatalf("inputs: %v", err)
			}
			cpuWant, cpuMS, err := measurementMatmulCPU(shape, a, b)
			if err != nil {
				t.Fatalf("CPU MatMul: %v", err)
			}
			deviceGot, deviceMS, err := measurementMatmulDeviceBestOfFive(t, shape, a, b)
			if err != nil {
				t.Fatalf("device MatMul: %v", err)
			}
			maxAbs, maxULP := measurementDeviation(cpuWant, deviceGot)
			macs := shape.aShape[0] * shape.aShape[1] * shape.bShape[1]
			fmt.Printf("ladder_macs=%d shape=%s device_ms=%.3f cpu_ms=%.3f ratio=%.3f maxAbs=%.9g maxULP=%d\n",
				macs, shape.name, deviceMS, cpuMS, deviceMS/cpuMS, maxAbs, maxULP)
		})
	}
}

func measurementMatmulInputs(shape matmulMeasurementShape) ([]float32, []float32, error) {
	batch := shape.batch
	if batch == 0 {
		batch = 1
	}
	aSize := batch * shape.aShape[0] * shape.aShape[1]
	bSize := batch * shape.bShape[0] * shape.bShape[1]
	a := make([]float32, aSize)
	b := make([]float32, bSize)
	for i := range a {
		a[i] = measurementValue(i, 11)
	}
	for i := range b {
		b[i] = measurementValue(i, 37)
	}
	return a, b, nil
}

func measurementValue(index, salt int) float32 {
	return float32((index*31+salt*17)%101-50) * 0.01
}

func measurementMatmulCPU(shape matmulMeasurementShape, a, b []float32) ([]float32, float64, error) {
	aTensor, err := dl.NewTensor(measurementTensorShape(shape, true), a)
	if err != nil {
		return nil, 0, err
	}
	bTensor, err := dl.NewTensor(measurementTensorShape(shape, false), b)
	if err != nil {
		return nil, 0, err
	}
	var result *dl.Tensor
	var bestDuration time.Duration
	var bestResult []float32
	for attempt := 0; attempt < 5; attempt++ {
		start := time.Now()
		result, err = dl.MatMul(aTensor, bTensor)
		elapsed := time.Since(start)
		if err != nil {
			return nil, 0, err
		}
		if attempt == 0 || elapsed < bestDuration {
			bestDuration = elapsed
			bestResult = result.Data()
		}
	}
	return bestResult, float64(bestDuration) / float64(time.Millisecond), nil
}

func measurementTensorShape(shape matmulMeasurementShape, left bool) []int {
	if shape.batch == 1 || shape.batch == 0 {
		if left {
			return []int{shape.aShape[0], shape.aShape[1]}
		}
		return []int{shape.bShape[0], shape.bShape[1]}
	}
	if left {
		return []int{shape.batch, shape.aShape[0], shape.aShape[1]}
	}
	return []int{shape.batch, shape.bShape[0], shape.bShape[1]}
}

func measurementMatmulDevice(ctx context.Context, shape matmulMeasurementShape, a, b []float32) ([]float32, time.Duration, error) {
	start := time.Now()
	if shape.batch <= 1 {
		result, _, err := wgpu.MatMul(ctx, a, b, shape.aShape[0], shape.aShape[1], shape.bShape[1])
		return result, time.Since(start), err
	}

	// Batched cases intentionally loop the 2-D kernel per batch. This harness
	// measures the round-trip cost of the existing kernel, not batched dispatch.
	aSize := shape.aShape[0] * shape.aShape[1]
	bSize := shape.bShape[0] * shape.bShape[1]
	cSize := shape.aShape[0] * shape.bShape[1]
	result := make([]float32, 0, shape.batch*cSize)
	for batch := 0; batch < shape.batch; batch++ {
		batchResult, _, err := wgpu.MatMul(ctx,
			a[batch*aSize:(batch+1)*aSize], b[batch*bSize:(batch+1)*bSize],
			shape.aShape[0], shape.aShape[1], shape.bShape[1])
		if err != nil {
			return nil, time.Since(start), err
		}
		result = append(result, batchResult...)
	}
	return result, time.Since(start), nil
}

func measurementMatmulDeviceBestOfFive(t *testing.T, shape matmulMeasurementShape, a, b []float32) ([]float32, float64, error) {
	t.Helper()
	var best []float32
	var bestDuration time.Duration
	for attempt := 0; attempt < 5; attempt++ {
		result, elapsed, err := measurementMatmulDevice(context.Background(), shape, a, b)
		if err != nil {
			return nil, 0, err
		}
		if attempt == 0 || elapsed < bestDuration {
			best = result
			bestDuration = elapsed
		}
	}
	return best, float64(bestDuration) / float64(time.Millisecond), nil
}

func measurementDeviation(want, got []float32) (float32, uint64) {
	var maxAbs float32
	var maxULP uint64
	for i := range want {
		abs := float32(math.Abs(float64(got[i] - want[i])))
		if abs > maxAbs {
			maxAbs = abs
		}
		if ulp := measurementULP(want[i], got[i]); ulp > maxULP {
			maxULP = ulp
		}
	}
	return maxAbs, maxULP
}

func measurementULP(a, b float32) uint64 {
	if a == b {
		return 0
	}
	ordered := func(value float32) uint32 {
		bits := math.Float32bits(value)
		if bits&0x80000000 != 0 {
			return ^bits
		}
		return bits | 0x80000000
	}
	first, second := ordered(a), ordered(b)
	if first > second {
		return uint64(first - second)
	}
	return uint64(second - first)
}
