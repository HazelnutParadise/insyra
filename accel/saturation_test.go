package accel

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra/accel/internal/wgpu"
)

const (
	saturationTrainRows = 100_000
	saturationBestOf    = 5
	saturationM         = 5
)

var saturationRungs = []int{1_000, 2_000, 4_000, 8_000, 16_000, 32_000, 64_000, 128_000}

type saturationArm struct {
	name string
	dims int
}

type saturationSample struct {
	duration  time.Duration
	result    ExecutionResult
	bytes     uint64
	rechecked int
}

// BenchmarkDeviceSaturation measures the exact-nearest operation in the
// direction that KNN uses: the test rows are the dataset and the training rows
// are the query points. Run with -benchtime=1x. The SATURATION_* filters keep a
// single rung or a small group runnable, so a long sweep never has to be one
// process invocation.
func BenchmarkDeviceSaturation(b *testing.B) {
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		b.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	backendInfo, backendErr := wgpu.Probe()
	if backendErr != nil {
		b.Skipf("no usable GPU backend: %v", backendErr)
	}

	probe, err := Open(Config{})
	if err != nil && len(probe.Devices()) == 0 {
		b.Skipf("no usable device: %v", err)
	}
	devices := probe.Devices()
	_ = probe.Close()
	device, ok := saturationDevice(devices)
	if !ok {
		b.Skip("only a software, virtual, or environment-stub adapter was discovered")
	}
	if backendInfo.Name == "" {
		b.Skip("GPU backend returned no hardware adapter information")
	}

	armFilter := strings.TrimSpace(os.Getenv("INSYRA_ACCEL_SATURATION_ARM"))
	rungFilter, err := saturationRungFilter()
	if err != nil {
		b.Fatal(err)
	}

	matched := false
	for _, arm := range []saturationArm{
		{name: "train=100000/dims=32", dims: 32},
		{name: "train=100000/dims=128", dims: 128},
	} {
		if armFilter != "" && armFilter != arm.name && armFilter != fmt.Sprintf("d%d", arm.dims) {
			continue
		}
		matched = true
		b.Run(arm.name, func(b *testing.B) {
			aborted := false
			for _, testRows := range saturationRungs {
				if aborted {
					break
				}
				if rungFilter > 0 && rungFilter != testRows {
					continue
				}
				b.Run(fmt.Sprintf("test_rows=%d", testRows), func(b *testing.B) {
					aborted = saturationRung(b, arm, testRows, device)
				})
			}
		})
	}
	if !matched {
		b.Fatalf("INSYRA_ACCEL_SATURATION_ARM=%q did not select an arm", armFilter)
	}
}

func saturationRungFilter() (int, error) {
	value := strings.TrimSpace(os.Getenv("INSYRA_ACCEL_SATURATION_RUNG"))
	if value == "" {
		return 0, nil
	}
	rung, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("INSYRA_ACCEL_SATURATION_RUNG=%q is not an integer: %w", value, err)
	}
	for _, candidate := range saturationRungs {
		if candidate == rung {
			return rung, nil
		}
	}
	return 0, fmt.Errorf("INSYRA_ACCEL_SATURATION_RUNG=%d is not one of %v", rung, saturationRungs)
}

func saturationDevice(devices []Device) (Device, bool) {
	for _, device := range devices {
		if device.ProbeSource == ProbeSourceEnvStub || device.Type == DeviceTypeCPU || device.Type == DeviceTypeVirtual {
			continue
		}
		return device, true
	}
	return Device{}, false
}

func saturationRung(b *testing.B, arm saturationArm, testRows int, device Device) bool {
	b.Helper()
	if reason, ok := saturationMemoryAbort(testRows, arm.dims, device); ok {
		fmt.Printf("saturation arm=%s test_rows=%d status=aborted abort_reason=%q\n", arm.name, testRows, reason)
		return true
	}

	rnd := rand.New(rand.NewSource(int64(20260805 + arm.dims + testRows)))
	dataset := exactDataset(testRows, arm.dims, rnd)
	queries := exactQueries(saturationTrainRows, arm.dims, rnd)

	session, openErr := Open(Config{})
	if openErr != nil && len(session.Devices()) == 0 {
		fmt.Printf("saturation arm=%s test_rows=%d status=aborted abort_reason=%q\n", arm.name, testRows, "device unavailable: "+openErr.Error())
		return true
	}
	defer func() { _ = session.Close() }()

	samples := make([]saturationSample, 0, saturationBestOf)
	for attempt := 0; attempt < saturationBestOf; attempt++ {
		start := time.Now()
		result, err := session.ExecuteNearestExact(dataset, queries, saturationM, WorkloadEstimate{Rows: testRows})
		elapsed := time.Since(start)
		if err != nil {
			fmt.Printf("saturation arm=%s test_rows=%d status=aborted abort_reason=%q attempt=%d\n", arm.name, testRows, "device execution: "+err.Error(), attempt+1)
			return true
		}
		if !result.Accelerated {
			fmt.Printf("saturation arm=%s test_rows=%d status=aborted abort_reason=%q attempt=%d\n", arm.name, testRows, "device fallback: "+string(result.FallbackReason), attempt+1)
			return true
		}
		samples = append(samples, saturationSample{
			duration:  elapsed,
			result:    result.ExecutionResult,
			bytes:     result.BytesUploaded,
			rechecked: result.Rechecked,
		})
	}

	best := samples[0]
	for _, sample := range samples[1:] {
		if sample.duration < best.duration {
			best = sample
		}
	}
	wallMS := make([]float64, len(samples))
	for i, sample := range samples {
		wallMS[i] = float64(sample.duration) / float64(time.Millisecond)
	}
	fmt.Printf(
		"saturation arm=%s test_rows=%d status=ok device=%q best_ms=%.3f samples_ms=%v transfer_ms=%.3f dispatch_ms=%.3f readback_ms=%.3f bytes_uploaded=%d rechecked_rows=%d\n",
		arm.name,
		testRows,
		device.Name,
		float64(best.duration)/float64(time.Millisecond),
		wallMS,
		float64(best.result.Transfer)/float64(time.Millisecond),
		float64(best.result.Dispatch)/float64(time.Millisecond),
		float64(best.result.Readback)/float64(time.Millisecond),
		best.bytes,
		best.rechecked,
	)
	return false
}

func saturationMemoryAbort(rows, dims int, device Device) (string, bool) {
	const shortlist = saturationM + 2
	elements := uint64(rows) * uint64(dims)
	queryElements := uint64(saturationTrainRows) * uint64(dims)
	listBytes := uint64(rows) * shortlist * 4
	dataBytes := elements * 4
	queryBytes := queryElements * 4
	deviceBytes := dataBytes + queryBytes + listBytes*2 + uint64(rows)*4 + listBytes*2 + uint64(rows)*4
	if device.BudgetBytes > 0 && deviceBytes > device.BudgetBytes {
		return fmt.Sprintf("estimated device buffers %d bytes exceed normalized device budget %d bytes", deviceBytes, device.BudgetBytes), true
	}

	// The operation briefly owns the float64 inputs, two float32 copies of each
	// input (the runtime narrowing plus backend flattening), readback buffers,
	// and exact-result storage. This is a preflight guard, not a measured time.
	hostBytes := (elements+queryElements)*16 + uint64(rows)*(shortlist*24+saturationM*12)
	if total := saturationHostMemoryBytes(); total > 0 {
		runtime.GC()
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)
		available := uint64(0)
		if stats.Sys < total {
			available = total - stats.Sys
		}
		if hostBytes > available {
			return fmt.Sprintf("estimated host allocations %d bytes exceed available host budget %d bytes", hostBytes, available), true
		}
	}
	return "", false
}

func saturationHostMemoryBytes() uint64 {
	if total := hostMemoryBytes(); total > 0 {
		return total
	}
	if runtime.GOOS != "darwin" {
		return 0
	}
	output, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0
	}
	total, err := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return 0
	}
	return total
}
