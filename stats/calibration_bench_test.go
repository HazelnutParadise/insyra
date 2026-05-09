package stats

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"sync"
	"testing"

	internalcluster "github.com/HazelnutParadise/insyra/stats/internal/clustering"
)

// This file calibrates the parallel-vs-serial cutover for each parallelised
// kernel by running both implementations across a size grid. It is the
// primary tool used to set the per-call-site threshold constants below in
// each caller — when re-tuning for new hardware, run:
//
//   go test -run=^$ -bench=Calib -benchtime=200ms ./stats > calib.txt
//
// then read off the smallest n where Par_<n> beats Ser_<n>.
//
// Reference (kept here, not in production code) implementations replicate
// what production used to do before the parallelisation patch — they are
// strictly serial so the comparison is honest.

// ---------- KendallTauB inner pair-count loop ----------

// kendallPairCountSerial mirrors the inner double loop of kendallTauBStats
// without any parallel reduction. Used only as a calibration baseline.
func kendallPairCountSerial(x, y []float64) (nC, nD float64) {
	n := len(x)
	for i := 0; i < n; i++ {
		xi, yi := x[i], y[i]
		for j := i + 1; j < n; j++ {
			dx := xi - x[j]
			dy := yi - y[j]
			if dx == 0 || dy == 0 {
				continue
			}
			if (dx > 0) == (dy > 0) {
				nC++
			} else {
				nD++
			}
		}
	}
	return
}

func kendallPairCountParallel(x, y []float64, workers int) (nC, nD float64) {
	n := len(x)
	if workers <= 1 || n < 2 {
		return kendallPairCountSerial(x, y)
	}
	if workers > n {
		workers = n
	}
	cArr := make([]float64, workers)
	dArr := make([]float64, workers)
	chunk := (n + workers - 1) / workers
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		start := w * chunk
		if start >= n {
			break
		}
		end := start + chunk
		if end > n {
			end = n
		}
		wg.Add(1)
		go func(w, start, end int) {
			defer wg.Done()
			var lc, ld float64
			for i := start; i < end; i++ {
				xi, yi := x[i], y[i]
				for j := i + 1; j < n; j++ {
					dx := xi - x[j]
					dy := yi - y[j]
					if dx == 0 || dy == 0 {
						continue
					}
					if (dx > 0) == (dy > 0) {
						lc++
					} else {
						ld++
					}
				}
			}
			cArr[w] = lc
			dArr[w] = ld
		}(w, start, end)
	}
	wg.Wait()
	for i := 0; i < workers; i++ {
		nC += cArr[i]
		nD += dArr[i]
	}
	return
}

func calibVec(n int, seed uint64) []float64 {
	rng := rand.New(rand.NewPCG(seed, seed^0xDEADBEEFCAFEBABE))
	out := make([]float64, n)
	for i := range out {
		out[i] = rng.NormFloat64()
	}
	return out
}

func BenchmarkCalib_KendallPairs(b *testing.B) {
	sizes := []int{32, 64, 96, 128, 192, 256, 384, 512, 768, 1024, 1536, 2048}
	for _, n := range sizes {
		x := calibVec(n, uint64(n))
		y := calibVec(n, uint64(n)+1)
		b.Run(fmt.Sprintf("Ser_n=%d", n), func(b *testing.B) {
			for b.Loop() {
				_, _ = kendallPairCountSerial(x, y)
			}
		})
		b.Run(fmt.Sprintf("Par_n=%d", n), func(b *testing.B) {
			for b.Loop() {
				_, _ = kendallPairCountParallel(x, y, 24)
			}
		})
	}
}

// BenchmarkCalib_KendallStrategies extends the brute-vs-parallel grid with
// Knight's O(n log n) algorithm to find the second crossover (where Knight
// beats parallel brute). Knight has higher constant factor (mergesort,
// allocation, lex sort) so it loses for small n where the n² loop's dense
// inner kernel dominates; the crossover lands well past the brute-parallel
// crossover and gates Knight's dispatch.
func BenchmarkCalib_KendallStrategies(b *testing.B) {
	sizes := []int{32, 64, 96, 128, 192, 256, 512, 1024, 1536, 2048, 3072, 4096, 6144, 8192}
	for _, n := range sizes {
		x := calibVec(n, uint64(n))
		y := calibVec(n, uint64(n)+1)
		b.Run(fmt.Sprintf("BruteSer_n=%d", n), func(b *testing.B) {
			for b.Loop() {
				_, _ = kendallPairCountBruteSerial(x, y)
			}
		})
		b.Run(fmt.Sprintf("BrutePar_n=%d", n), func(b *testing.B) {
			for b.Loop() {
				_, _ = kendallPairCountBruteParallel(x, y)
			}
		})
		b.Run(fmt.Sprintf("Knight_n=%d", n), func(b *testing.B) {
			for b.Loop() {
				_, _ = kendallPairCountKnight(x, y)
			}
		})
	}
}

// ---------- EuclideanDistanceMatrix ----------

func euclideanDistanceMatrixSerial(data [][]float64) [][]float64 {
	n := len(data)
	dist := make([][]float64, n)
	for i := range n {
		dist[i] = make([]float64, n)
		for j := 0; j < i; j++ {
			d := euclideanDist(data[i], data[j])
			dist[i][j] = d
			dist[j][i] = d
		}
	}
	return dist
}

func euclideanDist(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return math.Sqrt(sum)
}

func euclideanDistanceMatrixParallel(data [][]float64, workers int) [][]float64 {
	n := len(data)
	dist := make([][]float64, n)
	for i := range n {
		dist[i] = make([]float64, n)
	}
	if workers <= 1 {
		return euclideanDistanceMatrixSerial(data)
	}
	if workers > n {
		workers = n
	}
	chunk := (n + workers - 1) / workers
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		start := w * chunk
		if start >= n {
			break
		}
		end := start + chunk
		if end > n {
			end = n
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				ai := data[i]
				for j := 0; j < i; j++ {
					d := euclideanDist(ai, data[j])
					dist[i][j] = d
					dist[j][i] = d
				}
			}
		}(start, end)
	}
	wg.Wait()
	return dist
}

func BenchmarkCalib_DistMatrix(b *testing.B) {
	type cfg struct{ n, p int }
	cfgs := []cfg{
		{32, 4}, {64, 4}, {128, 4}, {256, 4}, {512, 4},
		{32, 16}, {64, 16}, {128, 16}, {256, 16}, {512, 16},
		{32, 64}, {64, 64}, {128, 64}, {256, 64},
	}
	for _, c := range cfgs {
		data := benchMatrix(c.n, c.p, uint64(c.n*c.p))
		b.Run(fmt.Sprintf("Ser_n=%d_p=%d", c.n, c.p), func(b *testing.B) {
			for b.Loop() {
				_ = euclideanDistanceMatrixSerial(data)
			}
		})
		b.Run(fmt.Sprintf("Par_n=%d_p=%d", c.n, c.p), func(b *testing.B) {
			for b.Loop() {
				_ = euclideanDistanceMatrixParallel(data, 24)
			}
		})
	}
}

// ---------- pickClosestPair (active-size sensitivity) ----------
//
// For hierarchical clustering the active list shrinks from N down to 2.
// Most steps are at small m, so this loop's total cost is dominated by the
// cumulative ∑m² behaviour, but each individual call's parallel viability
// only depends on m. We benchmark the cost per call across active sizes.

type pairPick struct{ a, b int }

func pickClosestPairSerial(active []int, dists []float64, stride int) pairPick {
	m := len(active)
	bestI, bestJ := active[0], active[1]
	bestDist := math.Inf(1)
	for i := 0; i < m; i++ {
		a := active[i]
		base := a * stride
		for j := i + 1; j < m; j++ {
			b := active[j]
			d := dists[base+b]
			if d < bestDist {
				bestDist, bestI, bestJ = d, a, b
			}
		}
	}
	return pairPick{bestI, bestJ}
}

func pickClosestPairParallel(active []int, dists []float64, stride, workers int) pairPick {
	m := len(active)
	if workers <= 1 || m < 4 {
		return pickClosestPairSerial(active, dists, stride)
	}
	if workers > m {
		workers = m
	}
	type lb struct {
		dist float64
		a, b int
		set  bool
	}
	locals := make([]lb, workers)
	chunk := (m + workers - 1) / workers
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		start := w * chunk
		if start >= m {
			break
		}
		end := start + chunk
		if end > m {
			end = m
		}
		wg.Add(1)
		go func(w, start, end int) {
			defer wg.Done()
			loc := lb{dist: math.Inf(1)}
			for i := start; i < end; i++ {
				a := active[i]
				base := a * stride
				for j := i + 1; j < m; j++ {
					b := active[j]
					d := dists[base+b]
					if !loc.set || d < loc.dist {
						loc.dist, loc.a, loc.b, loc.set = d, a, b, true
					}
				}
			}
			locals[w] = loc
		}(w, start, end)
	}
	wg.Wait()
	bestDist := math.Inf(1)
	bestI, bestJ := active[0], active[1]
	first := true
	for _, l := range locals {
		if !l.set {
			continue
		}
		if first || l.dist < bestDist {
			bestDist, bestI, bestJ = l.dist, l.a, l.b
			first = false
		}
	}
	return pairPick{bestI, bestJ}
}

func makePairBench(m int) ([]int, []float64, int) {
	stride := m
	rng := rand.New(rand.NewPCG(uint64(m), uint64(m)*7))
	dists := make([]float64, stride*stride)
	for i := 0; i < stride; i++ {
		for j := i + 1; j < stride; j++ {
			d := rng.Float64()
			dists[i*stride+j] = d
			dists[j*stride+i] = d
		}
	}
	active := make([]int, m)
	for i := range active {
		active[i] = i
	}
	return active, dists, stride
}

func BenchmarkCalib_PickPair(b *testing.B) {
	sizes := []int{8, 16, 32, 64, 128, 192, 256, 384, 512, 768, 1024}
	for _, m := range sizes {
		active, dists, stride := makePairBench(m)
		b.Run(fmt.Sprintf("Ser_m=%d", m), func(b *testing.B) {
			for b.Loop() {
				_ = pickClosestPairSerial(active, dists, stride)
			}
		})
		b.Run(fmt.Sprintf("Par_m=%d", m), func(b *testing.B) {
			for b.Loop() {
				_ = pickClosestPairParallel(active, dists, stride, 24)
			}
		})
	}
}

// ---------- KMeans initial center assignment ----------

func kmeansInitAssignSerial(data, centers [][]float64) ([]int, []int) {
	n := len(data)
	k := len(centers)
	ic1 := make([]int, n)
	ic2 := make([]int, n)
	for i, row := range data {
		ic1[i] = 0
		ic2[i] = 1
		dt1 := sqEucDist(row, centers[0])
		dt2 := sqEucDist(row, centers[1])
		if dt1 > dt2 {
			ic1[i], ic2[i] = 1, 0
			dt1, dt2 = dt2, dt1
		}
		for l := 2; l < k; l++ {
			db := sqEucDist(row, centers[l])
			if db >= dt2 {
				continue
			}
			if db >= dt1 {
				dt2 = db
				ic2[i] = l
				continue
			}
			dt2 = dt1
			ic2[i] = ic1[i]
			dt1 = db
			ic1[i] = l
		}
	}
	return ic1, ic2
}

func kmeansInitAssignParallel(data, centers [][]float64, workers int) ([]int, []int) {
	n := len(data)
	k := len(centers)
	ic1 := make([]int, n)
	ic2 := make([]int, n)
	if workers <= 1 {
		return kmeansInitAssignSerial(data, centers)
	}
	if workers > n {
		workers = n
	}
	chunk := (n + workers - 1) / workers
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		start := w * chunk
		if start >= n {
			break
		}
		end := start + chunk
		if end > n {
			end = n
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				row := data[i]
				ic1[i] = 0
				ic2[i] = 1
				dt1 := sqEucDist(row, centers[0])
				dt2 := sqEucDist(row, centers[1])
				if dt1 > dt2 {
					ic1[i], ic2[i] = 1, 0
					dt1, dt2 = dt2, dt1
				}
				for l := 2; l < k; l++ {
					db := sqEucDist(row, centers[l])
					if db >= dt2 {
						continue
					}
					if db >= dt1 {
						dt2 = db
						ic2[i] = l
						continue
					}
					dt2 = dt1
					ic2[i] = ic1[i]
					dt1 = db
					ic1[i] = l
				}
			}
		}(start, end)
	}
	wg.Wait()
	return ic1, ic2
}

func sqEucDist(a, b []float64) float64 {
	s := 0.0
	for i := range a {
		d := a[i] - b[i]
		s += d * d
	}
	return s
}

func BenchmarkCalib_KMeansInit(b *testing.B) {
	type cfg struct{ n, k, p int }
	cfgs := []cfg{
		{100, 4, 4}, {200, 4, 4}, {500, 4, 4}, {1000, 4, 4},
		{100, 8, 8}, {500, 8, 8}, {1500, 8, 8},
		{100, 20, 16}, {500, 20, 16}, {1500, 20, 16},
	}
	for _, c := range cfgs {
		data := benchMatrix(c.n, c.p, uint64(c.n))
		centers := benchMatrix(c.k, c.p, uint64(c.k+1000))
		b.Run(fmt.Sprintf("Ser_n=%d_k=%d_p=%d", c.n, c.k, c.p), func(b *testing.B) {
			for b.Loop() {
				_, _ = kmeansInitAssignSerial(data, centers)
			}
		})
		b.Run(fmt.Sprintf("Par_n=%d_k=%d_p=%d", c.n, c.k, c.p), func(b *testing.B) {
			for b.Loop() {
				_, _ = kmeansInitAssignParallel(data, centers, 24)
			}
		})
	}
}

// ---------- DBSCAN brute vs kd-tree neighbour search (algorithmic) ----------

// dbscanBruteNeighbours: O(n²) all-pairs. Mirrors the production DBSCAN
// neighbour build but in serial so the dispatch experiment is fair.
func dbscanBruteNeighbours(data [][]float64, eps float64) [][]int {
	n := len(data)
	nbrs := make([][]int, n)
	eps2 := eps * eps
	for i := 0; i < n; i++ {
		ai := data[i]
		var local []int
		for j := 0; j < n; j++ {
			if sqEucDist(ai, data[j]) <= eps2 {
				local = append(local, j)
			}
		}
		nbrs[i] = local
	}
	return nbrs
}

// dbscanBruteNeighboursParallel parallelises the production approach.
func dbscanBruteNeighboursParallel(data [][]float64, eps float64, workers int) [][]int {
	n := len(data)
	nbrs := make([][]int, n)
	eps2 := eps * eps
	if workers <= 1 {
		return dbscanBruteNeighbours(data, eps)
	}
	if workers > n {
		workers = n
	}
	chunk := (n + workers - 1) / workers
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		start := w * chunk
		if start >= n {
			break
		}
		end := start + chunk
		if end > n {
			end = n
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				ai := data[i]
				var local []int
				for j := 0; j < n; j++ {
					if sqEucDist(ai, data[j]) <= eps2 {
						local = append(local, j)
					}
				}
				nbrs[i] = local
			}
		}(start, end)
	}
	wg.Wait()
	return nbrs
}

// kdTree neighbour search: constructs a kd-tree once then queries each
// point's eps-ball. For low-dim data this is O(n log n + n * |neighbours|).
type kdNodeBench struct {
	axis    int
	pivot   int
	left    *kdNodeBench
	right   *kdNodeBench
	indices []int
}

func buildKdNodeBench(data [][]float64, idx []int, leafSize int) *kdNodeBench {
	if len(idx) == 0 {
		return nil
	}
	if len(idx) <= leafSize {
		out := append([]int(nil), idx...)
		return &kdNodeBench{indices: out}
	}
	axis := 0
	bestSpread := -1.0
	for a := range data[0] {
		mn := data[idx[0]][a]
		mx := mn
		for _, k := range idx[1:] {
			v := data[k][a]
			if v < mn {
				mn = v
			}
			if v > mx {
				mx = v
			}
		}
		if mx-mn > bestSpread {
			bestSpread = mx - mn
			axis = a
		}
	}
	sort.Slice(idx, func(a, b int) bool {
		return data[idx[a]][axis] < data[idx[b]][axis]
	})
	mid := len(idx) / 2
	pivot := idx[mid]
	left := append([]int(nil), idx[:mid]...)
	right := append([]int(nil), idx[mid+1:]...)
	return &kdNodeBench{
		axis:  axis,
		pivot: pivot,
		left:  buildKdNodeBench(data, left, leafSize),
		right: buildKdNodeBench(data, right, leafSize),
	}
}

func kdRangeQuery(node *kdNodeBench, data [][]float64, q []float64, eps2 float64, out *[]int) {
	if node == nil {
		return
	}
	if len(node.indices) > 0 {
		for _, i := range node.indices {
			if sqEucDist(data[i], q) <= eps2 {
				*out = append(*out, i)
			}
		}
		return
	}
	pivot := data[node.pivot]
	if sqEucDist(pivot, q) <= eps2 {
		*out = append(*out, node.pivot)
	}
	diff := q[node.axis] - pivot[node.axis]
	near, far := node.left, node.right
	if diff > 0 {
		near, far = node.right, node.left
	}
	kdRangeQuery(near, data, q, eps2, out)
	if diff*diff <= eps2 {
		kdRangeQuery(far, data, q, eps2, out)
	}
}

func dbscanKdNeighbours(data [][]float64, eps float64, leafSize int) [][]int {
	n := len(data)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	root := buildKdNodeBench(data, idx, leafSize)
	eps2 := eps * eps
	out := make([][]int, n)
	for i := range data {
		var nbrs []int
		kdRangeQuery(root, data, data[i], eps2, &nbrs)
		sort.Ints(nbrs)
		out[i] = nbrs
	}
	return out
}

func dbscanKdNeighboursParallel(data [][]float64, eps float64, leafSize, workers int) [][]int {
	n := len(data)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	root := buildKdNodeBench(data, idx, leafSize)
	eps2 := eps * eps
	out := make([][]int, n)
	if workers <= 1 {
		for i := range data {
			var nbrs []int
			kdRangeQuery(root, data, data[i], eps2, &nbrs)
			sort.Ints(nbrs)
			out[i] = nbrs
		}
		return out
	}
	if workers > n {
		workers = n
	}
	chunk := (n + workers - 1) / workers
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		start := w * chunk
		if start >= n {
			break
		}
		end := start + chunk
		if end > n {
			end = n
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				var nbrs []int
				kdRangeQuery(root, data, data[i], eps2, &nbrs)
				sort.Ints(nbrs)
				out[i] = nbrs
			}
		}(start, end)
	}
	wg.Wait()
	return out
}

func BenchmarkCalib_DBSCAN_Brute_vs_KD(b *testing.B) {
	type cfg struct {
		n, p int
		eps  float64
	}
	cfgs := []cfg{
		{200, 2, 0.4}, {500, 2, 0.4}, {1000, 2, 0.4}, {2000, 2, 0.4},
		{200, 4, 0.6}, {500, 4, 0.6}, {1000, 4, 0.6}, {2000, 4, 0.6},
		{200, 8, 1.0}, {500, 8, 1.0}, {1000, 8, 1.0}, {2000, 8, 1.0},
		{200, 16, 1.5}, {500, 16, 1.5}, {1000, 16, 1.5}, {2000, 16, 1.5},
	}
	for _, c := range cfgs {
		data := benchMatrix(c.n, c.p, uint64(c.n)+uint64(c.p)*1000)
		b.Run(fmt.Sprintf("BruteSer_n=%d_p=%d", c.n, c.p), func(b *testing.B) {
			for b.Loop() {
				_ = dbscanBruteNeighbours(data, c.eps)
			}
		})
		b.Run(fmt.Sprintf("BrutePar_n=%d_p=%d", c.n, c.p), func(b *testing.B) {
			for b.Loop() {
				_ = dbscanBruteNeighboursParallel(data, c.eps, 24)
			}
		})
		b.Run(fmt.Sprintf("KDSer_n=%d_p=%d", c.n, c.p), func(b *testing.B) {
			for b.Loop() {
				_ = dbscanKdNeighbours(data, c.eps, 16)
			}
		})
		b.Run(fmt.Sprintf("KDPar_n=%d_p=%d", c.n, c.p), func(b *testing.B) {
			for b.Loop() {
				_ = dbscanKdNeighboursParallel(data, c.eps, 16, 24)
			}
		})
	}
}

// ---------- Hierarchical full-pipeline (algorithmic comparison stub) ----------
//
// Smoke benchmark for current Hierarchical implementation across a few
// sizes — useful for confirming that any threshold change in pickClosestPair
// propagates into a measurable end-to-end win.
func BenchmarkCalib_HierarchicalEnd2End(b *testing.B) {
	for _, n := range []int{100, 200, 400, 800} {
		data := benchMatrix(n, 6, uint64(n))
		labels := make([]string, n)
		for i := range labels {
			labels[i] = fmt.Sprintf("p%d", i)
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for b.Loop() {
				_, err := internalcluster.Hierarchical(data, labels, "average")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
