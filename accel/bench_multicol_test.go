package accel

import (
	"fmt"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// BenchmarkMultiColumnTable measures a whole DataTable through the seam. Before
// batching this paid one device round trip per column.
func BenchmarkMultiColumnTable(b *testing.B) {
	if !gpuTestsEnabled(b) {
		return
	}
	const rows = 1 << 18
	for _, cols := range []int{1, 4, 8} {
		lists := make([]*insyra.DataList, cols)
		for c := range lists {
			values := make([]any, rows)
			for i := range values {
				values[i] = float64((i+c)%1000) * 0.5
			}
			lists[c] = insyra.NewDataList(values...).SetName(fmt.Sprintf("c%d", c))
		}
		dt := insyra.NewDataTable(lists...).SetName("wide")

		b.Run(fmt.Sprintf("cols=%d", cols), func(b *testing.B) {
			session, err := Open(Config{})
			if err != nil {
				b.Fatalf("open: %v", err)
			}
			b.Cleanup(func() { _ = session.Close() })
			var readback int64
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err := session.ExecuteDataTable(dt, WorkloadEstimate{Precision: PrecisionFloat32})
				if err != nil {
					b.Fatalf("execute: %v", err)
				}
				if !result.Accelerated {
					b.Fatalf("expected GPU execution, got %q", result.FallbackReason)
				}
				readback += result.Readback.Nanoseconds()
			}
			b.StopTimer()
			b.ReportMetric(float64(readback)/float64(b.N)/1e6, "readback_ms/op")
		})
	}
}
