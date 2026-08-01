package accel

import (
	"fmt"
	"math/rand"
	"testing"
)

// KNN-shaped shapes: every test row ranks every training row. The question the
// measurement answers: at realistic KNN sizes, does the device-assisted exact
// ranking beat the all-core CPU path, and from which shape onward?
//
// NOTE: this benchmark runs with the roles transposed relative to the actual
// KNN wiring (dataset=train, queries=test). The arithmetic is identical but
// device efficiency need not be; BenchmarkKNNTrueDirection below measures the
// direction knnbridge actually runs.
func BenchmarkKNNShapes(b *testing.B) {
	if !gpuTestsEnabled(b) {
		b.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	session, err := Open(Config{})
	if err != nil {
		b.Fatalf("open failed: %v", err)
	}
	defer func() { _ = session.Close() }()

	rnd := rand.New(rand.NewSource(31))
	for _, shape := range []struct{ train, test, dims int }{
		{10_000, 1_000, 8},
		{10_000, 1_000, 32},
		{100_000, 1_000, 8},
		{100_000, 1_000, 32},
		{100_000, 10_000, 32},
		{200_000, 1_000, 128},
	} {
		ds := exactDataset(shape.train, shape.dims, rnd)
		queries := exactQueries(shape.test, shape.dims, rnd)
		name := fmt.Sprintf("train=%d/test=%d/d=%d", shape.train, shape.test, shape.dims)
		b.Run(name+"/cpu", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, _, _, err := NearestExactCPU(ds, queries, 5); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(name+"/runtime", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				result, err := session.ExecuteNearestExact(ds, queries, 5, WorkloadEstimate{})
				if err != nil {
					b.Fatal(err)
				}
				if result.Accelerated {
					b.ReportMetric(1, "ondevice")
				} else {
					b.ReportMetric(0, "ondevice")
				}
			}
		})
	}
}

// BenchmarkKNNTrueDirection measures the direction the bridge actually runs:
// the TEST set is the dataset and the TRAINING set is the queries, k = 5 —
// exactly the call stats' auto algorithm makes through accel/knnbridge.
func BenchmarkKNNTrueDirection(b *testing.B) {
	if !gpuTestsEnabled(b) {
		b.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	session, err := Open(Config{})
	if err != nil {
		b.Fatalf("open failed: %v", err)
	}
	defer func() { _ = session.Close() }()

	rnd := rand.New(rand.NewSource(37))
	for _, shape := range []struct{ train, test, dims int }{
		{10_000, 1_000, 8},
		{10_000, 4_000, 8},
		{100_000, 1_000, 32},
		{100_000, 2_000, 32},
		{100_000, 4_000, 32},
		{100_000, 10_000, 32},
	} {
		testSet := exactDataset(shape.test, shape.dims, rnd)
		trainQueries := exactQueries(shape.train, shape.dims, rnd)
		name := fmt.Sprintf("train=%d/test=%d/d=%d", shape.train, shape.test, shape.dims)
		b.Run(name+"/cpu", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, _, _, err := NearestExactCPU(testSet, trainQueries, 5); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(name+"/runtime", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				result, err := session.ExecuteNearestExact(testSet, trainQueries, 5, WorkloadEstimate{})
				if err != nil {
					b.Fatal(err)
				}
				if result.Accelerated {
					b.ReportMetric(1, "ondevice")
				} else {
					b.ReportMetric(0, "ondevice")
				}
			}
		})
	}
}
