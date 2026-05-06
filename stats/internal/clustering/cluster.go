package clustering

import (
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/HazelnutParadise/insyra/stats/internal/parutil"
	"gonum.org/v1/gonum/floats"
)

type KMeansOptions struct {
	NStart  int
	IterMax int
	Seed    *int64
}

type KMeansResult struct {
	Cluster     []int
	Centers     [][]float64
	TotSS       float64
	WithinSS    []float64
	TotWithinSS float64
	BetweenSS   float64
	Size        []int
	Iter        int
	IFault      int
}

type HierarchicalResult struct {
	Merge      [][2]int
	Height     []float64
	Order      []int
	Labels     []string
	DistMethod string
}

type DBSCANOptions struct {
	BorderPoints *bool
}

type DBSCANResult struct {
	Cluster []int
	IsSeed  []bool
}

type SilhouettePoint struct {
	Cluster  int
	Neighbor int
	SilWidth float64
}

type SilhouetteResult struct {
	Points  []SilhouettePoint
	Average float64
}

type clusterNode struct {
	id       int
	members  []int
	size     int
	centroid []float64
	rID      int
	minLeaf  int
	height   float64
}

func EuclideanDistanceMatrix(data [][]float64) [][]float64 {
	n := len(data)
	dist := make([][]float64, n)
	for i := range n {
		dist[i] = make([]float64, n)
	}
	// Each row's strict lower-triangular fill is independent; we mirror to
	// the upper triangle within the same goroutine so no row is written by
	// two workers. Calibrated cutoff (BenchmarkCalib_DistMatrix on a 24-thread
	// AMD Ryzen): parallel beats serial when n²·p ≳ 200000. Below that the
	// goroutine launch dwarfs the work — e.g. (n=128, p=4) is 21% slower in
	// parallel because each row only has ~256 multiply-adds.
	p := 0
	if n > 0 {
		p = len(data[0])
	}
	parutil.Run(n, n*n*p >= 200_000, func(i int) {
		ai := data[i]
		for j := 0; j < i; j++ {
			d := euclidean(ai, data[j])
			dist[i][j] = d
			dist[j][i] = d
		}
	})
	return dist
}

func KMeans(data [][]float64, centers int, opts KMeansOptions) (*KMeansResult, error) {
	n := len(data)
	if n == 0 {
		return nil, errors.New("data must not be empty")
	}
	if centers <= 0 {
		return nil, errors.New("centers must be greater than 0")
	}
	if centers > n {
		return nil, errors.New("centers must not exceed row count")
	}
	if opts.NStart <= 0 {
		opts.NStart = 1
	}
	if opts.IterMax <= 0 {
		opts.IterMax = 10
	}
	initPool := data
	if opts.NStart >= 2 {
		initPool = uniqueRows(data)
		if len(initPool) < centers {
			return nil, errors.New("more cluster centers than distinct data points")
		}
	}

	seed := time.Now().UnixNano()
	if opts.Seed != nil {
		seed = *opts.Seed
	}
	rng := newRRNG(uint32(seed))

	// Pre-generate every start's centerIdx serially so the rRNG is consumed
	// in the exact same order regardless of parallel execution. Determinism
	// (for a given seed) is preserved bit-exactly.
	centerIdxs := make([][]int, opts.NStart)
	for s := range opts.NStart {
		centerIdxs[s] = rng.sampleInt(len(initPool), centers)
	}

	results := make([]*KMeansResult, opts.NStart)
	errs := make([]error, opts.NStart)
	// Each start runs an independent kmeansSingleStart whose cost is at
	// least ~1ms even for tiny n (Hartigan-Wong's OPTRA does ≥1 sweep over
	// all rows). Goroutine launch overhead is negligible relative to that,
	// so the parallel branch is worth taking from NStart=2 upwards.
	parutil.Run(opts.NStart, opts.NStart >= 2, func(s int) {
		results[s], errs[s] = kmeansSingleStart(data, initPool, centerIdxs[s], opts.IterMax)
	})

	// Reduce serially with the same comparison as before so the "first start
	// wins on near-tie" tie-break behaviour is identical to the previous loop.
	best := (*KMeansResult)(nil)
	for s := range opts.NStart {
		if errs[s] != nil {
			return nil, errs[s]
		}
		current := results[s]
		if best == nil || (current.TotWithinSS < best.TotWithinSS && !almostEqual(current.TotWithinSS, best.TotWithinSS)) {
			best = current
		}
	}
	return best, nil
}

func kmeansSingleStart(data, initPool [][]float64, centerIdx []int, iterMax int) (*KMeansResult, error) {
	n := len(data)
	p := len(data[0])
	centers := len(centerIdx)
	if centers == 1 {
		return singleClusterResult(data), nil
	}

	currentCenters := make([][]float64, centers)
	for i, idx := range centerIdx {
		currentCenters[i] = append([]float64(nil), initPool[idx]...)
	}

	ic1 := make([]int, n)
	ic2 := make([]int, n)
	nc := make([]int, centers)
	an1 := make([]float64, centers)
	an2 := make([]float64, centers)
	ncp := make([]int, centers)
	d := make([]float64, n)
	itran := make([]int, centers)
	live := make([]int, centers)

	// Initial assignment: each row picks its closest (ic1) and second-closest
	// (ic2) center. Rows are independent, so we run this phase in parallel.
	// Bit-exact: each row's choice depends only on its own distances and the
	// fixed initial centers, with deterministic tie-breaking baked into the
	// cascade. The subsequent centroid accumulation stays sequential to
	// preserve the running-sum order Hartigan-Wong expects.
	//
	// Calibrated cutoff (BenchmarkCalib_KMeansInit): per-row cost is ≈O(k·p)
	// inner ops, total work ≈n·k·p. Parallel beats serial when n·k·p ≳ 50000.
	// Below that the goroutine overhead dominates (e.g. n=500,k=8,p=8 totals
	// 32K ops and is 6% slower in parallel; n=1500,k=8,p=8 totals 96K and is
	// 1.5× faster).
	parutil.Run(n, n*centers*p >= 50_000, func(i int) {
		row := data[i]
		ic1[i] = 0
		ic2[i] = 1
		dt1 := squaredEuclidean(row, currentCenters[0])
		dt2 := squaredEuclidean(row, currentCenters[1])
		if dt1 > dt2 {
			ic1[i], ic2[i] = 1, 0
			dt1, dt2 = dt2, dt1
		}
		for l := 2; l < centers; l++ {
			db := squaredEuclidean(row, currentCenters[l])
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
	})

	for l := 0; l < centers; l++ {
		for j := 0; j < p; j++ {
			currentCenters[l][j] = 0
		}
	}
	for i, row := range data {
		l := ic1[i]
		nc[l]++
		for j := 0; j < p; j++ {
			currentCenters[l][j] += row[j]
		}
	}
	for l := 0; l < centers; l++ {
		if nc[l] == 0 {
			return nil, errors.New("empty cluster: try a better set of initial centers")
		}
		aa := float64(nc[l])
		for j := 0; j < p; j++ {
			currentCenters[l][j] /= aa
		}
		an2[l] = aa / (aa + 1)
		an1[l] = 1e30
		if aa > 1 {
			an1[l] = aa / (aa - 1)
		}
		itran[l] = 1
		ncp[l] = -1
	}

	indx := 0
	ifault := 0
	iter := 0
	maxQtran := 50 * n
	for iter = 1; iter <= iterMax; iter++ {
		indx = hwOptra(data, currentCenters, ic1, ic2, nc, an1, an2, ncp, d, itran, live, indx)
		if indx == n {
			break
		}
		if hwQtran(data, currentCenters, ic1, ic2, nc, an1, an2, ncp, d, itran, &indx, maxQtran) {
			ifault = 4
			break
		}
		if centers == 2 {
			break
		}
		for l := 0; l < centers; l++ {
			ncp[l] = 0
		}
	}
	if iter > iterMax {
		ifault = 2
		iter = iterMax
	}

	return buildKMeansResult(data, currentCenters, ic1, nc, iter, ifault), nil
}

func hwOptra(data, centers [][]float64, ic1, ic2, nc []int, an1, an2 []float64, ncp []int, d []float64, itran, live []int, indx int) int {
	n := len(data)
	k := len(centers)
	for l := 0; l < k; l++ {
		if itran[l] == 1 {
			live[l] = n + 1
		}
	}
	for i, row := range data {
		indx++
		l1 := ic1[i]
		l2 := ic2[i]
		ll := l2
		if nc[l1] == 1 {
			continue
		}
		if ncp[l1] != 0 {
			d[i] = squaredEuclidean(row, centers[l1]) * an1[l1]
		}
		r2 := squaredEuclidean(row, centers[l2]) * an2[l2]
		for l := 0; l < k; l++ {
			if ((i+1) >= live[l1] && (i+1) >= live[l]) || l == l1 || l == ll {
				continue
			}
			rr := r2 / an2[l]
			dc := boundedSquaredEuclidean(row, centers[l], rr)
			if dc < rr {
				r2 = dc * an2[l]
				l2 = l
			}
		}
		if r2 >= d[i] {
			ic2[i] = l2
		} else {
			indx = 0
			live[l1] = n + i + 1
			live[l2] = n + i + 1
			ncp[l1] = i + 1
			ncp[l2] = i + 1
			updateCentersForTransfer(row, centers, nc, an1, an2, l1, l2)
			ic1[i] = l2
			ic2[i] = l1
		}
		if indx == n {
			return indx
		}
	}
	for l := 0; l < k; l++ {
		itran[l] = 0
		live[l] -= n
	}
	return indx
}

func hwQtran(data, centers [][]float64, ic1, ic2, nc []int, an1, an2 []float64, ncp []int, d []float64, itran []int, indx *int, maxQtran int) bool {
	n := len(data)
	icoun := 0
	istep := 0
	for {
		for i, row := range data {
			icoun++
			istep++
			if istep >= maxQtran {
				return true
			}
			l1 := ic1[i]
			l2 := ic2[i]
			if nc[l1] == 1 {
				if icoun == n {
					return false
				}
				continue
			}
			if istep <= ncp[l1] {
				d[i] = squaredEuclidean(row, centers[l1]) * an1[l1]
			}
			if istep < ncp[l1] || istep < ncp[l2] {
				r2 := d[i] / an2[l2]
				dd := boundedSquaredEuclidean(row, centers[l2], r2)
				if dd < r2 {
					icoun = 0
					*indx = 0
					itran[l1] = 1
					itran[l2] = 1
					ncp[l1] = istep + n
					ncp[l2] = istep + n
					updateCentersForTransfer(row, centers, nc, an1, an2, l1, l2)
					ic1[i] = l2
					ic2[i] = l1
				}
			}
			if icoun == n {
				return false
			}
		}
	}
}

func updateCentersForTransfer(row []float64, centers [][]float64, nc []int, an1, an2 []float64, from, to int) {
	al1 := float64(nc[from])
	alw := al1 - 1
	al2 := float64(nc[to])
	alt := al2 + 1
	for j := range row {
		centers[from][j] = (centers[from][j]*al1 - row[j]) / alw
		centers[to][j] = (centers[to][j]*al2 + row[j]) / alt
	}
	nc[from]--
	nc[to]++
	an2[from] = alw / al1
	an1[from] = 1e30
	if alw > 1 {
		an1[from] = alw / (alw - 1)
	}
	an1[to] = alt / al2
	an2[to] = alt / (alt + 1)
}

func buildKMeansResult(data, centers [][]float64, assignments, nc []int, iter, ifault int) *KMeansResult {
	p := len(data[0])
	overallMean := make([]float64, p)
	for _, row := range data {
		for j := 0; j < p; j++ {
			overallMean[j] += row[j]
		}
	}
	for j := 0; j < p; j++ {
		overallMean[j] /= float64(len(data))
	}

	wss := make([]float64, len(centers))
	finalCenters := make([][]float64, len(centers))
	for i := range finalCenters {
		finalCenters[i] = make([]float64, p)
	}
	for i, row := range data {
		l := assignments[i]
		for j := 0; j < p; j++ {
			finalCenters[l][j] += row[j]
		}
	}
	for l := range finalCenters {
		for j := 0; j < p; j++ {
			finalCenters[l][j] /= float64(nc[l])
		}
	}

	totss := 0.0
	for i, row := range data {
		l := assignments[i]
		wss[l] += squaredEuclidean(row, finalCenters[l])
		totss += squaredEuclidean(row, overallMean)
	}
	totWithin := 0.0
	for _, v := range wss {
		totWithin += v
	}
	return &KMeansResult{
		Cluster:     addOne(assignments),
		Centers:     finalCenters,
		TotSS:       totss,
		WithinSS:    wss,
		TotWithinSS: totWithin,
		BetweenSS:   totss - totWithin,
		Size:        append([]int(nil), nc...),
		Iter:        iter,
		IFault:      ifault,
	}
}

func singleClusterResult(data [][]float64) *KMeansResult {
	p := len(data[0])
	center := make([]float64, p)
	for _, row := range data {
		for j := 0; j < p; j++ {
			center[j] += row[j]
		}
	}
	for j := 0; j < p; j++ {
		center[j] /= float64(len(data))
	}
	totss := 0.0
	for _, row := range data {
		totss += squaredEuclidean(row, center)
	}
	cluster := make([]int, len(data))
	for i := range cluster {
		cluster[i] = 1
	}
	return &KMeansResult{
		Cluster:     cluster,
		Centers:     [][]float64{center},
		TotSS:       totss,
		WithinSS:    []float64{totss},
		TotWithinSS: totss,
		BetweenSS:   0,
		Size:        []int{len(data)},
		Iter:        1,
		IFault:      0,
	}
}

// distStore is a flat dense distance matrix indexed by cluster ID.
//
// During hierarchical clustering at most 2n-1 cluster IDs are ever created
// (n leaves + n-1 internal merges). Storing all pairs in a (2n-1)² float64
// flat slice gives O(1) lookups and updates without map-hash overhead.
//
// The previous implementation used map[[2]int]float64. Map operations on a
// composite ([2]int) key dominated runtime: pickClosestPair scans O(active²)
// pairs per merge → O(n²) merges total → ~n³ map lookups. Replacing the map
// with array indexing dropped that constant by ~10× in microbenchmarks while
// preserving exact tie-breaking semantics (this struct does not change order).
//
// Memory: (2n-1)² × 8 bytes. For n=2000 that's ≈64 MB which is well within
// the regime where stats users do hierarchical clustering. Agglomerative
// clustering is O(n²) memory in any case (the distance matrix itself).
type distStore struct {
	stride int
	buf    []float64
}

func newDistStore(maxID int) *distStore {
	return &distStore{stride: maxID, buf: make([]float64, maxID*maxID)}
}

func (d *distStore) get(a, b int) float64 { return d.buf[a*d.stride+b] }
func (d *distStore) set(a, b int, v float64) {
	d.buf[a*d.stride+b] = v
	d.buf[b*d.stride+a] = v
}

// isLanceWilliamsReducible reports whether the linkage method is reducible
// in the sense of Bruynooghe (1977) — i.e. after merging clusters a and b
// at distance d_ab, the new distance from the merged cluster to any other
// cluster x is ≥ d_ab. Equivalent: the dendrogram heights are guaranteed
// monotone non-decreasing under any agglomeration order.
//
// This is the prerequisite for NN-chain to produce a correct dendrogram:
// the algorithm merges mutual nearest neighbours, and reducibility ensures
// that each such merge has a height no smaller than any preceding one,
// which is exactly the dendrogram-construction invariant.
//
// Median and centroid linkage are NOT reducible — their heights can
// "invert" (a parent merge below its child). The standard greedy
// "always merge the smallest pair" algorithm still produces the correct
// dendrogram for them, so we keep that path for those two methods.
func isLanceWilliamsReducible(method string) bool {
	switch method {
	case "single", "complete", "average", "mcquitty", "ward.d", "ward.d2":
		return true
	default:
		return false
	}
}

func Hierarchical(data [][]float64, labels []string, method string) (*HierarchicalResult, error) {
	n := len(data)
	if n < 2 {
		return nil, errors.New("hierarchical clustering requires at least 2 rows")
	}
	if len(labels) != n {
		return nil, errors.New("labels length mismatch")
	}
	method = normalizeMethod(method)
	if !isSupportedMethod(method) {
		return nil, errors.New("unsupported agglomerative method")
	}
	if isLanceWilliamsReducible(method) {
		return hierarchicalNNChain(data, labels, method)
	}
	return hierarchicalGreedy(data, labels, method)
}

// hierarchicalGreedy is the textbook O(N³) "find smallest pair, merge,
// update distances, repeat" algorithm. Used only for median and centroid
// linkage where Lance-Williams reducibility doesn't hold and NN-chain
// can't be substituted.
func hierarchicalGreedy(data [][]float64, labels []string, method string) (*HierarchicalResult, error) {
	n := len(data)
	maxID := 2*n - 1
	clusters := make([]*clusterNode, maxID)
	active := make([]int, n)
	for i := range n {
		clusters[i] = &clusterNode{
			id:       i,
			members:  []int{i},
			size:     1,
			centroid: append([]float64(nil), data[i]...),
			rID:      -(i + 1),
			minLeaf:  i,
			height:   0,
		}
		active[i] = i
	}

	dists := newDistStore(maxID)
	useSquared := method == "ward.d2"
	p := 0
	if n > 0 {
		p = len(data[0])
	}
	parutil.Run(n, n*n*p >= 200_000, func(i int) {
		row := data[i]
		base := i * dists.stride
		for j := i + 1; j < n; j++ {
			var d float64
			if useSquared {
				d = squaredEuclidean(row, data[j])
			} else {
				d = euclidean(row, data[j])
			}
			dists.buf[base+j] = d
			dists.buf[j*dists.stride+i] = d
		}
	})

	merge := make([][2]int, 0, n-1)
	height := make([]float64, 0, n-1)
	nextID := n
	for step := 1; step < n; step++ {
		aID, bID, dist := pickClosestPair(active, clusters, dists)
		a := clusters[aID]
		b := clusters[bID]
		left, right := orientClusters(a, b)

		merge = append(merge, [2]int{left.rID, right.rID})
		if useSquared {
			height = append(height, math.Sqrt(dist))
		} else {
			height = append(height, dist)
		}

		newCluster := &clusterNode{
			id:       nextID,
			members:  append(append([]int{}, left.members...), right.members...),
			size:     left.size + right.size,
			centroid: mergedCentroid(left, right, method),
			rID:      step,
			minLeaf:  min(left.minLeaf, right.minLeaf),
			height:   dist,
		}
		clusters[nextID] = newCluster

		newActive := make([]int, 0, len(active)-1)
		for _, id := range active {
			if id == aID || id == bID {
				continue
			}
			newActive = append(newActive, id)
			dists.set(id, nextID, updatedDistance(method, clusters[id], a, b, dists.get(id, aID), dists.get(id, bID), dist))
		}
		newActive = append(newActive, nextID)
		active = newActive
		nextID++
	}

	root := clusters[active[0]]
	order := append([]int(nil), root.members...)
	for i := range order {
		order[i]++
	}
	return &HierarchicalResult{
		Merge:      merge,
		Height:     height,
		Order:      order,
		Labels:     append([]string(nil), labels...),
		DistMethod: "euclidean",
	}, nil
}

// hierarchicalNNChain is Murtagh's (1985) NN-chain agglomerative algorithm.
// O(N²) time and space (the dense distance matrix is the dominant memory
// cost). Reduces hierarchical clustering on N points from O(N³) — the
// greedy "scan every pair every step" cost — to O(N²) for any Lance-
// Williams reducible linkage.
//
// Why this is bit-equivalent to greedy on this codebase's tests: R's hclust
// also uses NN-chain for these methods, and our existing R-reference tests
// are written against that output. The greedy implementation passed those
// tests because the test fixtures happen to be non-degenerate (no near-ties
// in inter-cluster distance), which is the only regime where the two
// algorithms can disagree on merge ordering.
//
// The algorithm:
//
//  1. Maintain a chain (stack) of clusters. Push any active cluster.
//  2. Walk: at each step look at the chain's top cluster, find its nearest
//     active neighbour (with tie-break preferring the second-from-top of
//     the chain — guarantees we detect mutual NNs as soon as one exists).
//  3. If the NN is the second-from-top, we have a mutual-NN pair. Pop both
//     and merge them. Update distances from the new cluster to all other
//     active clusters via the Lance-Williams formula. Restart at step 1
//     (the chain may still have entries; those are still valid because
//     their NN distances cannot increase).
//  4. Otherwise push the NN onto the chain and continue.
//
// Reducibility (Bruynooghe 1977) guarantees that the merges produced this
// way have monotone non-decreasing heights, which is exactly the dendrogram
// invariant.
func hierarchicalNNChain(data [][]float64, labels []string, method string) (*HierarchicalResult, error) {
	n := len(data)
	maxID := 2*n - 1
	clusters := make([]*clusterNode, maxID)
	// `alive[id]` flags currently-active cluster IDs. Iteration order in
	// the NN-search loop is ascending ID — R's hclust uses the same order
	// and the existing single_ties_case crosslang fixture pins it: any
	// switch to a packed-active-slice representation alters which leaf is
	// found first as NN-of-merged-cluster on tied data, producing a
	// different (but equally valid) Merge[] that no longer matches R.
	alive := make([]bool, maxID)
	for i := range n {
		clusters[i] = &clusterNode{
			id:      i,
			members: []int{i},
			size:    1,
			rID:     -(i + 1),
			minLeaf: i,
			height:  0,
		}
		alive[i] = true
	}

	dists := newDistStore(maxID)
	useSquared := method == "ward.d2"
	p := 0
	if n > 0 {
		p = len(data[0])
	}
	parutil.Run(n, n*n*p >= 200_000, func(i int) {
		row := data[i]
		base := i * dists.stride
		for j := i + 1; j < n; j++ {
			var d float64
			if useSquared {
				d = squaredEuclidean(row, data[j])
			} else {
				d = euclidean(row, data[j])
			}
			dists.buf[base+j] = d
			dists.buf[j*dists.stride+i] = d
		}
	})

	// During NN-chain we record each merge as (leftRID_atMerge,
	// rightRID_atMerge, distAtMerge, orderDiscovered). NN-chain finds
	// mutual-NN pairs in some order; reducibility makes their distances
	// monotone for fresh leaves, but distances between two pre-existing
	// inter-leaf pairs can be smaller than an earlier merge's distance
	// (because reducibility only constrains distances FROM the merged
	// cluster, not between two unrelated old leaves). So the raw merge
	// list is not yet a valid dendrogram order — we sort it by distance
	// at the end and remap the internal rID references.
	type rawMerge struct {
		leftOldRID, rightOldRID int
		dist                    float64
		discovered              int
	}
	raws := make([]rawMerge, 0, n-1)

	chain := make([]int, 0, n)
	nextID := n
	mergesDone := 0

	// firstAlive returns the smallest-ID alive cluster — used to re-seed
	// the chain after the very first iteration if the chainSeed has since
	// been merged. After every merge we set chainSeed to the newly-created
	// cluster (R hclust convention), so this fallback only triggers if the
	// caller has somehow mid-aborted; in normal operation the alive cluster
	// chosen as chainSeed is always still alive.
	firstAlive := func() int {
		for c := 0; c < maxID; c++ {
			if alive[c] {
				return c
			}
		}
		return -1
	}

	// Seed for an empty chain. For the very first iteration this is the
	// smallest-ID original cluster; after a merge that empties the chain we
	// seed with the just-created merged cluster (R hclust convention) — that
	// keeps the chain productive across merges and matches R's tie-break
	// behaviour on data with multiple equidistant pairs (e.g. four corners of
	// a square plus an outlier: greedy and NN-chain produce the same merge
	// sequence only if NN-chain seeds from the merged cluster, otherwise the
	// second merge picks two independent leaves instead of grafting a leaf
	// onto the first merge).
	chainSeed := firstAlive()
	if chainSeed < 0 {
		return nil, errors.New("NN-chain ran out of active clusters early")
	}

	for mergesDone < n-1 {
		if len(chain) == 0 {
			if !alive[chainSeed] {
				chainSeed = firstAlive()
				if chainSeed < 0 {
					return nil, errors.New("NN-chain ran out of active clusters early")
				}
			}
			chain = append(chain, chainSeed)
		}
		top := chain[len(chain)-1]
		prev := -1
		if len(chain) >= 2 {
			prev = chain[len(chain)-2]
		}

		// Find the nearest neighbour of `top` among active clusters,
		// breaking ties in favour of `prev` if present. Initialising
		// (bestNN, bestDist) with (prev, d(top, prev)) — when prev exists
		// — gives prev priority on ties because we update only on strict
		// "<" below.
		//
		// Iteration order: ascending cluster ID. R's hclust uses the same
		// order; the single_ties_case crosslang fixture pins this — any
		// re-ordering (e.g. via a packed active slice with swap-remove)
		// changes which leaf is picked as NN-of-merged-cluster on tied
		// data and breaks R alignment.
		bestNN := -1
		bestDist := math.Inf(1)
		if prev >= 0 {
			bestNN = prev
			bestDist = dists.get(top, prev)
		}
		topRow := top * dists.stride
		for c := 0; c < maxID; c++ {
			if !alive[c] || c == top || c == prev {
				continue
			}
			d := dists.buf[topRow+c]
			if d < bestDist {
				bestDist = d
				bestNN = c
			}
		}

		if prev >= 0 && bestNN == prev {
			// Mutual NN: pop top and prev, merge them.
			chain = chain[:len(chain)-2]
			a := clusters[top]
			b := clusters[prev]
			left, right := orientClusters(a, b)

			step := mergesDone + 1
			raws = append(raws, rawMerge{
				leftOldRID:  left.rID,
				rightOldRID: right.rID,
				dist:        bestDist,
				discovered:  step,
			})

			// Members: single allocation of the exact size, two copies.
			// The previous `append(append([]int{}, left), right)` form
			// could re-allocate twice and is heavier on the GC for large
			// merges. With sole-allocation form, GC pressure on n=400
			// dropped from 31% gcDrain in the profile to a non-bottleneck.
			combined := make([]int, left.size+right.size)
			copy(combined, left.members)
			copy(combined[left.size:], right.members)
			newCluster := &clusterNode{
				id:      nextID,
				members: combined,
				size:    left.size + right.size,
				rID:     step,
				minLeaf: min(left.minLeaf, right.minLeaf),
				height:  bestDist,
			}
			clusters[nextID] = newCluster

			alive[top] = false
			alive[prev] = false
			alive[nextID] = true

			// Lance-Williams update: distance from the new cluster to
			// every other active cluster, derived from the two old
			// distances and d_ab.
			for c := 0; c < maxID; c++ {
				if !alive[c] || c == nextID {
					continue
				}
				dac := dists.get(top, c)
				dbc := dists.get(prev, c)
				newD := updatedDistance(method, clusters[c], a, b, dac, dbc, bestDist)
				dists.set(nextID, c, newD)
			}

			// Seed for the next empty-chain restart: the merged cluster.
			chainSeed = nextID
			nextID++
			mergesDone++
			continue
		}

		chain = append(chain, bestNN)
	}

	// Sort merges by distance (stable: ties keep the discovery order so
	// the final rID assignment is deterministic for tied data).
	sortedIdx := make([]int, len(raws))
	for i := range sortedIdx {
		sortedIdx[i] = i
	}
	sort.SliceStable(sortedIdx, func(i, j int) bool {
		return raws[sortedIdx[i]].dist < raws[sortedIdx[j]].dist
	})

	// Build remap: old rID (in discovery order, 1..n-1) → new rID (in
	// sorted-by-distance order, also 1..n-1). Leaf rIDs are negative and
	// unchanged.
	remap := make([]int, n)
	for newPos, oldI := range sortedIdx {
		remap[raws[oldI].discovered] = newPos + 1
	}
	mapRID := func(rid int) int {
		if rid < 0 {
			return rid
		}
		return remap[rid]
	}

	merge := make([][2]int, len(raws))
	height := make([]float64, len(raws))
	for newPos, oldI := range sortedIdx {
		r := raws[oldI]
		merge[newPos] = [2]int{mapRID(r.leftOldRID), mapRID(r.rightOldRID)}
		if useSquared {
			height[newPos] = math.Sqrt(r.dist)
		} else {
			height[newPos] = r.dist
		}
	}

	// Only one cluster remains alive after n-1 merges — the root.
	rootID := firstAlive()
	root := clusters[rootID]
	order := append([]int(nil), root.members...)
	for i := range order {
		order[i]++
	}
	return &HierarchicalResult{
		Merge:      merge,
		Height:     height,
		Order:      order,
		Labels:     append([]string(nil), labels...),
		DistMethod: "euclidean",
	}, nil
}

func CutTreeByK(tree *HierarchicalResult, k int) ([]int, error) {
	n := len(tree.Labels)
	if n == 0 {
		return nil, errors.New("tree must not be empty")
	}
	if k <= 0 || k > n {
		return nil, errors.New("k must be between 1 and number of observations")
	}
	mergesToApply := n - k
	return cutTree(tree, func(step int, _ float64) bool { return step < mergesToApply })
}

func CutTreeByHeight(tree *HierarchicalResult, h float64) ([]int, error) {
	if len(tree.Labels) == 0 {
		return nil, errors.New("tree must not be empty")
	}
	return cutTree(tree, func(_ int, height float64) bool { return height <= h })
}

func cutTree(tree *HierarchicalResult, include func(step int, height float64) bool) ([]int, error) {
	n := len(tree.Labels)
	parent := make([]int, n)
	for i := range n {
		parent[i] = i
	}

	nodes := make(map[int][]int, n+len(tree.Merge))
	for i := range n {
		nodes[-(i + 1)] = []int{i}
	}
	for step, row := range tree.Merge {
		left := append([]int(nil), nodes[row[0]]...)
		right := append([]int(nil), nodes[row[1]]...)
		combined := append(left, right...)
		nodes[step+1] = combined
		if include(step, tree.Height[step]) {
			unionMembers(parent, combined)
		}
	}

	labelMap := map[int]int{}
	nextLabel := 1
	out := make([]int, n)
	for i := 0; i < n; i++ {
		root := find(parent, i)
		if _, ok := labelMap[root]; !ok {
			labelMap[root] = nextLabel
			nextLabel++
		}
		out[i] = labelMap[root]
	}
	return out, nil
}

func DBSCAN(data [][]float64, eps float64, minPts int, opts DBSCANOptions) (*DBSCANResult, error) {
	n := len(data)
	if n == 0 {
		return nil, errors.New("data must not be empty")
	}
	if eps <= 0 {
		return nil, errors.New("eps must be greater than 0")
	}
	if minPts < 1 {
		return nil, errors.New("minPts must be at least 1")
	}
	borderPoints := true
	if opts.BorderPoints != nil {
		borderPoints = *opts.BorderPoints
	}

	// Neighbour-finding strategy is chosen from calibration data
	// (BenchmarkCalib_DBSCAN_Brute_vs_KD on 24 threads).
	dim := 0
	if n > 0 {
		dim = len(data[0])
	}
	useKD := dbscanShouldUseKD(n, dim)
	neighbors, isSeed := dbscanBuildNeighbors(data, eps, minPts, useKD)

	cluster := make([]int, n)
	visited := make([]bool, n)
	clusterID := 0
	for i := range n {
		if visited[i] || !isSeed[i] {
			continue
		}
		clusterID++
		queue := append([]int(nil), neighbors[i]...)
		cluster[i] = clusterID
		visited[i] = true
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			if !visited[cur] {
				visited[cur] = true
				if isSeed[cur] {
					queue = append(queue, neighbors[cur]...)
				}
			}
			if cluster[cur] == 0 && (isSeed[cur] || borderPoints) {
				cluster[cur] = clusterID
			}
		}
	}
	return &DBSCANResult{
		Cluster: cluster,
		IsSeed:  isSeed,
	}, nil
}

func Silhouette(data [][]float64, labels []int) (*SilhouetteResult, error) {
	n := len(data)
	if n == 0 {
		return nil, errors.New("data must not be empty")
	}
	if len(labels) != n {
		return nil, errors.New("labels length mismatch")
	}
	clusterMembers := map[int][]int{}
	for i, label := range labels {
		if label <= 0 {
			return nil, errors.New("labels must be positive integers")
		}
		clusterMembers[label] = append(clusterMembers[label], i)
	}
	if len(clusterMembers) < 2 {
		return nil, errors.New("silhouette requires at least 2 clusters")
	}

	dist := EuclideanDistanceMatrix(data)
	points := make([]SilhouettePoint, n)

	// Iterating clusterMembers via `for k := range map` gives a randomised
	// order per Go run. The previous tie-break logic
	//   `avg < bestB || (almostEqual(avg, bestB) && otherLabel < neighborLabel)`
	// is NOT order-independent when two avg values differ by less than the
	// 1e-12 almostEqual tolerance: visiting the slightly-smaller one first
	// locks bestB to it, then a slightly-larger but smaller-labelled cluster
	// arriving second satisfies the almostEqual branch and overwrites — but
	// the OPPOSITE order keeps the slightly-smaller (larger-label) cluster.
	// On tied data (e.g. axis-aligned grids) this surfaces as a different
	// neighborLabel between runs.
	//
	// Fix: pre-sort the cluster keys ascending. With a deterministic order
	// we visit smaller labels first and the tie-break collapses to "take the
	// first cluster whose avg is no further from the running best than 1ε" —
	// a single condition rather than a three-way OR.
	clusterKeys := make([]int, 0, len(clusterMembers))
	for k := range clusterMembers {
		clusterKeys = append(clusterKeys, k)
	}
	sort.Ints(clusterKeys)

	// Per-worker partial sums avoid contention on a single accumulator.
	// Each worker only writes to its own slot in `partial` and only writes
	// points[i] for its assigned i's, so no shared state is mutated.
	// Per-i cost is O(n) (sum of distances to own cluster + scan of other
	// clusters). Total ≈ O(n²). Same break-even shape as Kendall pair
	// counting (~n=192 on 24 threads), so we use n ≥ 200 here.
	workers := 1
	if n >= 200 {
		workers = parutil.MaxWorkers(n)
	}
	partial := make([]float64, workers)
	chunk := (n + workers - 1) / workers
	var wg sync.WaitGroup
	for w := range workers {
		start := w * chunk
		if start >= n {
			break
		}
		end := min(start+chunk, n)
		wg.Add(1)
		go func(w, start, end int) {
			defer wg.Done()
			localSum := 0.0
			for i := start; i < end; i++ {
				label := labels[i]
				own := clusterMembers[label]
				a := 0.0
				if len(own) > 1 {
					for _, j := range own {
						if j == i {
							continue
						}
						a += dist[i][j]
					}
					a /= float64(len(own) - 1)
				}

				neighborLabel := 0
				bestB := math.Inf(1)
				// Visit clusters in ascending-label order (clusterKeys is
				// pre-sorted). This makes the result independent of the
				// hash-map iteration order. Update only on strictly-smaller
				// avg (outside the almostEqual tolerance band) — within the
				// band we keep the running best, which by sort order has
				// the smaller label.
				for _, otherLabel := range clusterKeys {
					if otherLabel == label {
						continue
					}
					members := clusterMembers[otherLabel]
					avg := 0.0
					for _, j := range members {
						avg += dist[i][j]
					}
					avg /= float64(len(members))
					if neighborLabel == 0 || (avg < bestB && !almostEqual(avg, bestB)) {
						bestB = avg
						neighborLabel = otherLabel
					}
				}

				s := 0.0
				if len(own) > 1 {
					denom := math.Max(a, bestB)
					if denom > 0 {
						s = (bestB - a) / denom
					}
				}
				points[i] = SilhouettePoint{
					Cluster:  label,
					Neighbor: neighborLabel,
					SilWidth: s,
				}
				localSum += s
			}
			partial[w] = localSum
		}(w, start, end)
	}
	wg.Wait()
	sum := 0.0
	for _, v := range partial {
		sum += v
	}
	return &SilhouetteResult{
		Points:  points,
		Average: sum / float64(n),
	}, nil
}

func updatedDistance(method string, other, a, b *clusterNode, dik, djk, dij float64) float64 {
	switch method {
	case "ward.d", "ward.d2":
		return ((float64(a.size+other.size) * dik) + (float64(b.size+other.size) * djk) - float64(other.size)*dij) / float64(a.size+b.size+other.size)
	case "single":
		return math.Min(dik, djk)
	case "complete":
		return math.Max(dik, djk)
	case "average":
		return (float64(a.size)*dik + float64(b.size)*djk) / float64(a.size+b.size)
	case "mcquitty":
		return 0.5*dik + 0.5*djk
	case "median":
		return ((dik + djk) - dij/2) / 2
	case "centroid":
		return (float64(a.size)*dik + float64(b.size)*djk - float64(a.size*b.size)*dij/float64(a.size+b.size)) / float64(a.size+b.size)
	default:
		return math.Max(dik, djk)
	}
}

func mergedCentroid(a, b *clusterNode, method string) []float64 {
	p := len(a.centroid)
	out := make([]float64, p)
	switch method {
	case "median":
		for i := range p {
			out[i] = (a.centroid[i] + b.centroid[i]) / 2
		}
	default:
		total := float64(a.size + b.size)
		for i := range p {
			out[i] = (float64(a.size)*a.centroid[i] + float64(b.size)*b.centroid[i]) / total
		}
	}
	return out
}

func orientClusters(a, b *clusterNode) (*clusterNode, *clusterNode) {
	if !almostEqual(a.height, b.height) {
		if a.height < b.height {
			return a, b
		}
		return b, a
	}
	if a.minLeaf < b.minLeaf {
		return a, b
	}
	if b.minLeaf < a.minLeaf {
		return b, a
	}
	if a.rID <= b.rID {
		return a, b
	}
	return b, a
}

// pickClosestPair scans every (i, j) pair in active for the smallest distance,
// breaking ties via tieBreakPair. Worker-local minima are reduced sequentially
// using the same comparison, so the parallel version is bit-identical to the
// previous serial scan.
//
// Calibrated cutoff (BenchmarkCalib_PickPair on 24 threads): the inner kernel
// is just an array load + compare per pair, so per-i work is ~m·5ns. Goroutine
// overhead is much larger than that for m ≲ 480 — at m=384 parallel is 18%
// slower because the pair-count work (≈74K compares) is dwarfed by 24-way
// fan-out cost. Crossover lands at m ≈ 480-512 (m=512 is 1.09× faster, m=768
// is 1.45×). We gate at 480.
//
// Initial bestDist = +Inf means the first finite distance always wins outright,
// so the placeholder (bestI=0, bestJ=1) is never read for tie-breaking.
func pickClosestPair(active []int, clusters []*clusterNode, dists *distStore) (int, int, float64) {
	m := len(active)
	if m < 2 {
		return active[0], active[0], 0
	}
	workers := 1
	if m >= 480 {
		workers = parutil.MaxWorkers(m)
	}
	if workers <= 1 {
		bestI, bestJ := 0, 1
		bestDist := math.Inf(1)
		for i := range m {
			a := active[i]
			rowBase := a * dists.stride
			for j := i + 1; j < m; j++ {
				b := active[j]
				d := dists.buf[rowBase+b]
				if d < bestDist {
					bestDist, bestI, bestJ = d, a, b
				} else if almostEqual(d, bestDist) {
					ca, cb := orientClusters(clusters[a], clusters[b])
					if tieBreakPair(ca, cb, clusters[bestI], clusters[bestJ]) {
						bestDist, bestI, bestJ = d, a, b
					}
				}
			}
		}
		return bestI, bestJ, bestDist
	}

	type localBest struct {
		dist float64
		a, b int
		set  bool
	}
	locals := make([]localBest, workers)
	chunk := (m + workers - 1) / workers
	var wg sync.WaitGroup
	for w := range workers {
		start := w * chunk
		if start >= m {
			break
		}
		end := min(start+chunk, m)
		wg.Add(1)
		go func(w, start, end int) {
			defer wg.Done()
			lb := localBest{dist: math.Inf(1)}
			for i := start; i < end; i++ {
				a := active[i]
				rowBase := a * dists.stride
				for j := i + 1; j < m; j++ {
					b := active[j]
					d := dists.buf[rowBase+b]
					if !lb.set || d < lb.dist {
						lb.dist, lb.a, lb.b, lb.set = d, a, b, true
					} else if almostEqual(d, lb.dist) {
						ca, cb := orientClusters(clusters[a], clusters[b])
						if tieBreakPair(ca, cb, clusters[lb.a], clusters[lb.b]) {
							lb.dist, lb.a, lb.b = d, a, b
						}
					}
				}
			}
			locals[w] = lb
		}(w, start, end)
	}
	wg.Wait()

	bestI, bestJ := 0, 1
	bestDist := math.Inf(1)
	first := true
	for _, lb := range locals {
		if !lb.set {
			continue
		}
		if first || lb.dist < bestDist {
			bestDist, bestI, bestJ = lb.dist, lb.a, lb.b
			first = false
		} else if almostEqual(lb.dist, bestDist) {
			ca, cb := orientClusters(clusters[lb.a], clusters[lb.b])
			if tieBreakPair(ca, cb, clusters[bestI], clusters[bestJ]) {
				bestDist, bestI, bestJ = lb.dist, lb.a, lb.b
			}
		}
	}
	return bestI, bestJ, bestDist
}

func tieBreakPair(a1, b1, a2, b2 *clusterNode) bool {
	la1, lb1 := orientClusters(a1, b1)
	la2, lb2 := orientClusters(a2, b2)
	if la1.minLeaf != la2.minLeaf {
		return la1.minLeaf < la2.minLeaf
	}
	return lb1.minLeaf < lb2.minLeaf
}

func nearestCenter(row []float64, centers [][]float64) int {
	bestIdx := 0
	bestDist := squaredEuclidean(row, centers[0])
	for i := 1; i < len(centers); i++ {
		d := squaredEuclidean(row, centers[i])
		if d < bestDist || (almostEqual(d, bestDist) && i < bestIdx) {
			bestDist = d
			bestIdx = i
		}
	}
	return bestIdx
}

func farthestPoint(data [][]float64, assignments []int, centers [][]float64) int {
	bestIdx := 0
	bestDist := -1.0
	for i, row := range data {
		d := squaredEuclidean(row, centers[assignments[i]])
		if d > bestDist {
			bestDist = d
			bestIdx = i
		}
	}
	return bestIdx
}

func addOne(xs []int) []int {
	out := make([]int, len(xs))
	for i, x := range xs {
		out[i] = x + 1
	}
	return out
}

func cloneMatrix(in [][]float64) [][]float64 {
	out := make([][]float64, len(in))
	for i := range in {
		out[i] = append([]float64(nil), in[i]...)
	}
	return out
}

func uniqueRows(data [][]float64) [][]float64 {
	seen := make(map[string]struct{}, len(data))
	out := make([][]float64, 0, len(data))
	for _, row := range data {
		key := rowKey(row)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, append([]float64(nil), row...))
	}
	return out
}

func rowKey(row []float64) string {
	var b strings.Builder
	for i, v := range row {
		if i > 0 {
			b.WriteByte('|')
		}
		b.WriteString(strconv.FormatFloat(v, 'g', 17, 64))
	}
	return b.String()
}

func normalizeMethod(method string) string {
	method = strings.ToLower(method)
	switch method {
	case "ward.d":
		return "ward.d"
	case "ward.d2":
		return "ward.d2"
	default:
		return method
	}
}

func isSupportedMethod(method string) bool {
	switch method {
	case "complete", "single", "average", "ward.d", "ward.d2", "mcquitty", "median", "centroid":
		return true
	default:
		return false
	}
}

func pairKey(a, b int) [2]int {
	if a < b {
		return [2]int{a, b}
	}
	return [2]int{b, a}
}

func unionMembers(parent []int, members []int) {
	if len(members) == 0 {
		return
	}
	base := members[0]
	for _, m := range members[1:] {
		union(parent, base, m)
	}
}

func find(parent []int, x int) int {
	if parent[x] != x {
		parent[x] = find(parent, parent[x])
	}
	return parent[x]
}

func union(parent []int, a, b int) {
	ra := find(parent, a)
	rb := find(parent, b)
	if ra == rb {
		return
	}
	if ra < rb {
		parent[rb] = ra
	} else {
		parent[ra] = rb
	}
}

// euclidean delegates to gonum/floats.Distance which uses the same
// loop-and-sqrt formulation but is the package-standard primitive.
// The boundedSquaredEuclidean and squaredEuclidean helpers below stay
// hand-rolled — gonum has no direct equivalent for "early-exit if the
// running sum exceeds a bound" or for plain squared-Euclidean without
// a sqrt at the end.
func euclidean(a, b []float64) float64 {
	return floats.Distance(a, b, 2)
}

func boundedSquaredEuclidean(a, b []float64, bound float64) float64 {
	sum := 0.0
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
		if sum >= bound {
			return sum
		}
	}
	return sum
}

// squaredEuclidean is the hottest primitive in the package (KMeans
// Hartigan-Wong, KNN, DBSCAN, Silhouette all call it heavily). Tried a
// `_ = b[len(a)-1]` BCE hint; it measured slower in BenchmarkKNN_
// BruteClassify (~390µs → ~460µs) because the entry-time bounds-check +
// empty-len guard cost more than the per-iter check savings the
// compiler would extract. Keep the simple form.
func squaredEuclidean(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return sum
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= 1e-12
}

type lcg struct {
	state uint64
}

func newLCG(seed uint64) *lcg {
	return &lcg{state: seed % 2147483647}
}

func (r *lcg) next() uint64 {
	r.state = (1103515245*r.state + 12345) % 2147483647
	return r.state
}

func (r *lcg) perm(n int) []int {
	out := make([]int, n)
	for i := range n {
		out[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j := int(r.next() % uint64(i+1))
		out[i], out[j] = out[j], out[i]
	}
	return out
}

type rRNG struct {
	mt  [624]uint32
	mti int
}

func newRRNG(seed uint32) *rRNG {
	r := &rRNG{}
	for i := 0; i < 50; i++ {
		seed = 69069*seed + 1
	}
	for i := 0; i < 625; i++ {
		seed = 69069*seed + 1
		if i > 0 {
			r.mt[i-1] = seed
		}
	}
	r.mti = 624
	return r
}

func (r *rRNG) unif() float64 {
	const i232m1 = 2.328306437080797e-10
	x := r.mtGenRand()
	if x <= 0 {
		return 0.5 * i232m1
	}
	if 1.0-x <= 0 {
		return 1.0 - 0.5*i232m1
	}
	return x
}

func (r *rRNG) mtGenRand() float64 {
	const (
		n              = 624
		m              = 397
		matrixA        = uint32(0x9908b0df)
		upperMask      = uint32(0x80000000)
		lowerMask      = uint32(0x7fffffff)
		temperingMaskB = uint32(0x9d2c5680)
		temperingMaskC = uint32(0xefc60000)
	)
	if r.mti >= n {
		mag01 := [2]uint32{0x0, matrixA}
		for kk := 0; kk < n-m; kk++ {
			y := (r.mt[kk] & upperMask) | (r.mt[kk+1] & lowerMask)
			r.mt[kk] = r.mt[kk+m] ^ (y >> 1) ^ mag01[y&0x1]
		}
		for kk := n - m; kk < n-1; kk++ {
			y := (r.mt[kk] & upperMask) | (r.mt[kk+1] & lowerMask)
			r.mt[kk] = r.mt[kk+(m-n)] ^ (y >> 1) ^ mag01[y&0x1]
		}
		y := (r.mt[n-1] & upperMask) | (r.mt[0] & lowerMask)
		r.mt[n-1] = r.mt[m-1] ^ (y >> 1) ^ mag01[y&0x1]
		r.mti = 0
	}
	y := r.mt[r.mti]
	r.mti++
	y ^= y >> 11
	y ^= (y << 7) & temperingMaskB
	y ^= (y << 15) & temperingMaskC
	y ^= y >> 18
	return float64(y) * 2.3283064365386963e-10
}

func (r *rRNG) rbits(bits int) float64 {
	var v uint64
	for n := 0; n <= bits; n += 16 {
		v1 := uint64(math.Floor(r.unif() * 65536))
		v = 65536*v + v1
	}
	if bits <= 0 {
		return 0
	}
	return float64(v & ((uint64(1) << bits) - 1))
}

func (r *rRNG) unifIndex(dn int) int {
	if dn <= 0 {
		return 0
	}
	bits := int(math.Ceil(math.Log2(float64(dn))))
	for {
		dv := r.rbits(bits)
		if float64(dn) > dv {
			return int(dv)
		}
	}
}

func (r *rRNG) sampleInt(n, k int) []int {
	x := make([]int, n)
	for i := 0; i < n; i++ {
		x[i] = i
	}
	out := make([]int, k)
	for i := 0; i < k; i++ {
		j := r.unifIndex(n)
		out[i] = x[j]
		n--
		x[j] = x[n]
	}
	return out
}
