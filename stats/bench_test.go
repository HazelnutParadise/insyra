package stats

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/HazelnutParadise/insyra"
	internalcluster "github.com/HazelnutParadise/insyra/stats/internal/clustering"
	internalknn "github.com/HazelnutParadise/insyra/stats/internal/knn"
)

// Benchmarks for the parallelised hot paths. All inputs are deterministic
// (fixed-seed PRNG) so successive runs are comparable, and sizes are picked
// to (a) exceed parutil's small-n cutoff so the parallel branch fires, and
// (b) stay within a few hundred ms per op so go test -bench remains usable.
//
// To compare against the previous serial implementation, stash this file's
// changes alongside the production code on the optimisation branch and run
// `go test -bench=. ./stats -benchmem -run=^$` on each branch.

func benchMatrix(rows, cols int, seed uint64) [][]float64 {
	rng := rand.New(rand.NewPCG(seed, seed^0x9E3779B97F4A7C15))
	out := make([][]float64, rows)
	for i := range out {
		out[i] = make([]float64, cols)
		for j := range out[i] {
			out[i][j] = rng.NormFloat64()
		}
	}
	return out
}

func benchTable(rows, cols int, seed uint64) *insyra.DataTable {
	m := benchMatrix(rows, cols, seed)
	dt := insyra.NewDataTable()
	for j := range cols {
		col := insyra.NewDataList().SetName(fmt.Sprintf("V%d", j+1))
		for i := range rows {
			col.Append(m[i][j])
		}
		dt.AppendCols(col)
	}
	return dt
}

func BenchmarkHierarchical_Small(b *testing.B) {
	data := benchMatrix(150, 4, 1)
	labels := make([]string, len(data))
	for i := range labels {
		labels[i] = fmt.Sprintf("p%d", i)
	}
	b.ResetTimer()
	for b.Loop() {
		_, err := internalcluster.Hierarchical(data, labels, "average")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHierarchical_Medium(b *testing.B) {
	data := benchMatrix(400, 6, 2)
	labels := make([]string, len(data))
	for i := range labels {
		labels[i] = fmt.Sprintf("p%d", i)
	}
	b.ResetTimer()
	for b.Loop() {
		_, err := internalcluster.Hierarchical(data, labels, "ward.d2")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkKMeans_NStart(b *testing.B) {
	data := benchMatrix(500, 5, 3)
	seed := int64(42)
	opts := internalcluster.KMeansOptions{NStart: 10, IterMax: 20, Seed: &seed}
	b.ResetTimer()
	for b.Loop() {
		_, err := internalcluster.KMeans(data, 6, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkKMeans_Single(b *testing.B) {
	data := benchMatrix(1500, 8, 4)
	seed := int64(7)
	opts := internalcluster.KMeansOptions{NStart: 1, IterMax: 30, Seed: &seed}
	b.ResetTimer()
	for b.Loop() {
		_, err := internalcluster.KMeans(data, 10, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDBSCAN(b *testing.B) {
	data := benchMatrix(800, 4, 5)
	b.ResetTimer()
	for b.Loop() {
		_, err := internalcluster.DBSCAN(data, 0.6, 5, internalcluster.DBSCANOptions{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSilhouette(b *testing.B) {
	data := benchMatrix(400, 4, 6)
	labels := make([]int, len(data))
	for i := range labels {
		labels[i] = (i % 5) + 1
	}
	b.ResetTimer()
	for b.Loop() {
		_, err := internalcluster.Silhouette(data, labels)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEuclideanDistanceMatrix(b *testing.B) {
	data := benchMatrix(500, 8, 7)
	b.ResetTimer()
	for b.Loop() {
		_ = internalcluster.EuclideanDistanceMatrix(data)
	}
}

func BenchmarkKNN_BruteClassify(b *testing.B) {
	train := benchMatrix(200, 6, 8)
	test := benchMatrix(200, 6, 9)
	labels := make([]string, len(train))
	for i := range labels {
		labels[i] = fmt.Sprintf("c%d", i%4)
	}
	opts := internalknn.Options{Algorithm: internalknn.BruteForceAlgorithm, Weighting: internalknn.UniformWeighting}
	b.ResetTimer()
	for b.Loop() {
		_, err := internalknn.Classify(train, test, labels, 5, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkKNN_KDTreeQueries(b *testing.B) {
	train := benchMatrix(2000, 4, 10)
	test := benchMatrix(500, 4, 11)
	opts := internalknn.Options{Algorithm: internalknn.KDTreeAlgorithm, LeafSize: 16}
	b.ResetTimer()
	for b.Loop() {
		_, err := internalknn.Neighbors(train, test, 8, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkKNN_BallTreeRegress(b *testing.B) {
	train := benchMatrix(1500, 12, 12)
	test := benchMatrix(500, 12, 13)
	targets := make([]float64, len(train))
	for i := range targets {
		targets[i] = float64(i%7) + 0.5
	}
	opts := internalknn.Options{Algorithm: internalknn.BallTreeAlgorithm, LeafSize: 32, Weighting: internalknn.DistanceWeighting}
	b.ResetTimer()
	for b.Loop() {
		_, err := internalknn.Regress(train, test, targets, 10, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCorrelationMatrix_Pearson(b *testing.B) {
	dt := benchTable(500, 12, 20)
	b.ResetTimer()
	for b.Loop() {
		_, _, err := CorrelationMatrix(dt, PearsonCorrelation)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCorrelationMatrix_Spearman(b *testing.B) {
	dt := benchTable(400, 10, 21)
	b.ResetTimer()
	for b.Loop() {
		_, _, err := CorrelationMatrix(dt, SpearmanCorrelation)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCorrelationMatrix_Kendall(b *testing.B) {
	dt := benchTable(200, 6, 22)
	b.ResetTimer()
	for b.Loop() {
		_, _, err := CorrelationMatrix(dt, KendallCorrelation)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkKendallTauB(b *testing.B) {
	rows := benchMatrix(400, 1, 23)
	cols := benchMatrix(400, 1, 24)
	x := make([]float64, len(rows))
	y := make([]float64, len(cols))
	for i := range x {
		x[i] = rows[i][0]
		y[i] = cols[i][0]
	}
	b.ResetTimer()
	for b.Loop() {
		_, _, _ = kendallTauBStats(x, y)
	}
}

func BenchmarkPCA(b *testing.B) {
	dt := benchTable(800, 12, 30)
	b.ResetTimer()
	for b.Loop() {
		_, err := PCA(dt)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTwoWayANOVA(b *testing.B) {
	const A, B, perCell = 4, 5, 60
	cells := make([]insyra.IDataList, A*B)
	rng := rand.New(rand.NewPCG(40, 41))
	for c := range cells {
		dl := insyra.NewDataList()
		for range perCell {
			dl.Append(rng.NormFloat64() + float64(c%A))
		}
		cells[c] = dl
	}
	b.ResetTimer()
	for b.Loop() {
		_, err := TwoWayANOVA(A, B, cells...)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkChiSquareIndependence(b *testing.B) {
	const n = 5000
	rng := rand.New(rand.NewPCG(50, 51))
	rowDL := insyra.NewDataList()
	colDL := insyra.NewDataList()
	for range n {
		rowDL.Append(fmt.Sprintf("row%d", rng.IntN(8)))
		colDL.Append(fmt.Sprintf("col%d", rng.IntN(6)))
	}
	b.ResetTimer()
	for b.Loop() {
		_, err := ChiSquareIndependenceTest(rowDL, colDL)
		if err != nil {
			b.Fatal(err)
		}
	}
}
