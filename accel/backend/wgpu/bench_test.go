package wgpu

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/accel"
)

// BenchmarkColumnSum measures what a column reduction actually costs end to end
// so the runtime's profitability rule can be built on measurements rather than
// on per-backend constants. Run with INSYRA_ACCEL_GPU_TESTS=1.
func BenchmarkColumnSum(b *testing.B) {
	for _, size := range []int{1 << 16, 1 << 20, 1 << 22} {
		values := make([]any, size)
		for i := range values {
			values[i] = float64(i%1000) * 0.5
		}
		dl := insyra.NewDataList(values...).SetName("numbers")

		b.Run(sizeName(size)+"/gpu", func(b *testing.B) {
			if !gpuTestsEnabled(b) {
				return
			}
			session, err := accel.Open(accel.Config{})
			if err != nil {
				b.Fatalf("open accel session: %v", err)
			}
			b.Cleanup(func() { _ = session.Close() })

			var transfer, dispatch, readback int64
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err := session.ExecuteDataList(dl, accel.WorkloadEstimate{Precision: accel.PrecisionFloat32})
				if err != nil {
					b.Fatalf("execute failed: %v", err)
				}
				if !result.Accelerated {
					b.Fatalf("expected GPU execution, got %q", result.FallbackReason)
				}
				transfer += result.Transfer.Nanoseconds()
				dispatch += result.Dispatch.Nanoseconds()
				readback += result.Readback.Nanoseconds()
			}
			b.StopTimer()
			b.ReportMetric(float64(transfer)/float64(b.N)/1e6, "transfer_ms/op")
			b.ReportMetric(float64(dispatch)/float64(b.N)/1e6, "dispatch_ms/op")
			b.ReportMetric(float64(readback)/float64(b.N)/1e6, "readback_ms/op")
		})

		b.Run(sizeName(size)+"/cpu", func(b *testing.B) {
			raw := make([]float64, size)
			for i := range raw {
				raw[i] = float64(i%1000) * 0.5
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sum := 0.0
				for _, value := range raw {
					sum += value
				}
				_ = sum
			}
		})
	}
}

func gpuTestsEnabled(b *testing.B) bool {
	b.Helper()
	if _, err := acquire(); err != nil {
		b.Skipf("no usable GPU on this host: %v", err)
		return false
	}
	return true
}

func sizeName(size int) string {
	switch {
	case size >= 1<<20:
		return itoa(size>>20) + "Mi"
	default:
		return itoa(size>>10) + "Ki"
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	digits := ""
	for v > 0 {
		digits = string(rune('0'+v%10)) + digits
		v /= 10
	}
	return digits
}

// BenchmarkProjectionOnly isolates the host-side projection and fingerprint
// cost from the device cost.
func BenchmarkProjectionOnly(b *testing.B) {
	const size = 1 << 22
	values := make([]any, size)
	for i := range values {
		values[i] = float64(i%1000) * 0.5
	}
	dl := insyra.NewDataList(values...).SetName("numbers")
	session, err := accel.Open(accel.Config{})
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	b.Cleanup(func() { _ = session.Close() })
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := session.ProjectDataList(dl); err != nil {
			b.Fatalf("project: %v", err)
		}
	}
}
