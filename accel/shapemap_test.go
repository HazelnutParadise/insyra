package accel

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"
)

// TestShapeMap measures where narrowing on a device actually wins, across the
// row and query counts the library produces. It asserts nothing: it is the
// calibration source for minQueriesForDevice and for deciding which callers are
// worth wiring up. Run with INSYRA_ACCEL_SHAPE_MAP=1.
func TestShapeMap(t *testing.T) {
	if os.Getenv("INSYRA_ACCEL_SHAPE_MAP") != "1" || os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_SHAPE_MAP=1 and INSYRA_ACCEL_GPU_TESTS=1")
	}
	// Look below the profitability floor: measuring it is the point.
	previous := minWorkPerRowForDevice
	minWorkPerRowForDevice = 0
	defer func() { minWorkPerRowForDevice = previous }()

	session, err := Open(Config{})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	defer func() { _ = session.Close() }()

	rowCounts := []int{2_000, 10_000, 50_000, 200_000}
	queryCounts := []int{8, 16, 32, 64, 128, 256, 512, 1024}
	dimCounts := []int{4, 16, 64}
	const m = 2

	best := func(reps int, f func() time.Duration) time.Duration {
		lowest := time.Duration(1<<62 - 1)
		for i := 0; i < reps; i++ {
			if d := f(); d < lowest {
				lowest = d
			}
		}
		return lowest
	}

	fmt.Printf("\n%6s %6s %6s | %10s %10s %7s | %9s %9s %9s %9s\n",
		"rows", "dims", "q", "cpu", "gpu", "speedup", "transfer", "dispatch", "readback", "recheck")
	for _, dims := range dimCounts {
		rnd := rand.New(rand.NewSource(int64(dims)))
		for _, rows := range rowCounts {
			ds := exactDataset(rows, dims, rnd)
			for _, queries := range queryCounts {
				points := exactQueries(queries, dims, rnd)

				// Cost model: keep the heavy cells to one repetition.
				reps := 3
				if int64(rows)*int64(queries)*int64(dims) > 200_000_000 {
					reps = 1
				}

				// Warm the device and the pipeline before timing.
				if _, err := session.ExecuteNearestExact(ds, points, m, WorkloadEstimate{}); err != nil {
					t.Fatalf("warmup failed: %v", err)
				}

				cpu := best(reps, func() time.Duration {
					start := time.Now()
					if _, _, _, err := NearestExactCPU(ds, points, m); err != nil {
						t.Fatal(err)
					}
					return time.Since(start)
				})

				// The breakdown must come from the repetition whose total is
				// reported, or the columns describe different runs and cannot be
				// added up.
				var sample ExactNearestResult
				gpu := time.Duration(1<<62 - 1)
				for i := 0; i < reps; i++ {
					start := time.Now()
					result, err := session.ExecuteNearestExact(ds, points, m, WorkloadEstimate{})
					if err != nil {
						t.Fatal(err)
					}
					if elapsed := time.Since(start); elapsed < gpu {
						gpu = elapsed
						sample = result
					}
				}

				marker := ""
				if !sample.Accelerated {
					marker = fmt.Sprintf(" [host: %s]", sample.FallbackReason)
				}
				fmt.Printf("%6d %6d %6d | %10s %10s %6.2fx | %9s %9s %9s %9d%s\n",
					rows, dims, queries,
					cpu.Round(time.Microsecond), gpu.Round(time.Microsecond),
					float64(cpu)/float64(gpu),
					sample.Transfer.Round(time.Microsecond),
					sample.Dispatch.Round(time.Microsecond),
					sample.Readback.Round(time.Microsecond),
					sample.Rechecked, marker)
			}
		}
	}
}
