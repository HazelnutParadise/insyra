package clustering

import (
	"sort"

	"github.com/HazelnutParadise/insyra/stats/internal/parutil"
)

// dbscanShouldUseKD returns true if DBSCAN's neighbour finder should switch
// from parallel brute O(n²·p) to a KD-tree range query.
//
// Calibrated cutoff (BenchmarkCalib_DBSCAN_Brute_vs_KD on 24 threads):
//   - At (n=500, p=4)  brute-par 197µs vs KD-par 226µs  → brute wins (KD overhead).
//   - At (n=500, p=16) brute-par 529µs vs KD-par 510µs  → KD wins.
//   - At (n=2000, p=8) brute-par 3.32ms vs KD-par 2.19ms → KD wins (1.5×).
//
// The gate `n ≥ 500 ∧ n·p ≥ 8000` captures the empirical wedge.
//
// Exposed (lowercase) so the dispatch_coverage tests can force either path
// and assert bit-equal neighbour sets — see TestDBSCANBruteVsKDEquivalence.
func dbscanShouldUseKD(n, dim int) bool {
	return n >= 500 && n*dim >= 8000
}

// dbscanBuildNeighbors constructs the per-point neighbour lists and seed
// flags using either the brute-force or KD-tree path.
//
// Both paths produce identical sorted neighbour lists by construction:
//   - Brute iterates j in 0..n order and appends qualifying j; output is
//     naturally sorted ascending.
//   - KD's tree traversal can deliver indices in any order, so we sort.Ints
//     the result to match.
//
// Behaviour for the two paths is therefore bit-equal at the neighbour-list
// level, which is the only piece of state DBSCAN's downstream cluster-
// expansion BFS reads. The expansion is deterministic given identical
// neighbours, so cluster IDs are also bit-equal.
//
// Exposed (lowercase) for the equivalence test in dispatch_coverage_test.go.
func dbscanBuildNeighbors(data [][]float64, eps float64, minPts int, useKD bool) (neighbors [][]int, isSeed []bool) {
	n := len(data)
	dim := 0
	if n > 0 {
		dim = len(data[0])
	}
	neighbors = make([][]int, n)
	isSeed = make([]bool, n)
	if useKD {
		idx := make([]int, n)
		for i := range idx {
			idx[i] = i
		}
		root := buildKdRange(data, idx, 16)
		eps2 := eps * eps
		parutil.Run(n, true, func(i int) {
			var nbrs []int
			kdRangeSearch(root, data, data[i], eps2, &nbrs)
			sort.Ints(nbrs)
			neighbors[i] = nbrs
			if len(nbrs) >= minPts {
				isSeed[i] = true
			}
		})
		return
	}
	parutil.Run(n, n*n*dim >= 30_000, func(i int) {
		ai := data[i]
		nbrs := make([]int, 0, 8)
		for j := range n {
			if euclideanDist1(ai, data[j]) <= eps {
				nbrs = append(nbrs, j)
			}
		}
		neighbors[i] = nbrs
		if len(nbrs) >= minPts {
			isSeed[i] = true
		}
	})
	return
}

// euclideanDist1 forwards to the cluster.go euclidean primitive without
// pulling that file into kdrange.go's import surface — same numbers, same
// gonum/floats backend.
var euclideanDist1 = euclidean

// kdRangeNode is a binary KD-tree node specialised for ε-ball range queries.
// It is private to this package and used only by the DBSCAN neighbour finder
// when the brute-force O(n²) path would be slower (see dbscanNeighbors).
//
// Why not reuse stats/internal/knn's kd-tree? That one is structured for
// top-k nearest-neighbour search with a max-heap of size k; the inner loop
// tracks "worst distance still in the set" to prune. ε-ball range queries
// have a fixed radius, no k, no heap — the pruning rule is simpler and the
// per-node bookkeeping is different. Forcing the two query types through
// one tree class would either bloat the kNN path or slow this one.
type kdRangeNode struct {
	axis    int
	pivot   int
	left    *kdRangeNode
	right   *kdRangeNode
	indices []int // populated only on leaves
}

// buildKdRange constructs a balanced KD-tree over `idx` (a permutation of
// some subset of [0, len(data))). leafSize controls how many points sit in
// a leaf bucket — the neighbour-search inner loop visits every point in a
// leaf, so larger leaves trade tighter pruning for less recursion overhead.
// Empirically 16 was a sweet spot for the existing knn ball-tree.
func buildKdRange(data [][]float64, idx []int, leafSize int) *kdRangeNode {
	if len(idx) == 0 {
		return nil
	}
	if len(idx) <= leafSize {
		out := append([]int(nil), idx...)
		sort.Ints(out)
		return &kdRangeNode{indices: out}
	}
	axis := widestAxis(data, idx)
	sort.Slice(idx, func(a, b int) bool {
		return data[idx[a]][axis] < data[idx[b]][axis]
	})
	mid := len(idx) / 2
	pivot := idx[mid]
	left := append([]int(nil), idx[:mid]...)
	right := append([]int(nil), idx[mid+1:]...)
	return &kdRangeNode{
		axis:  axis,
		pivot: pivot,
		left:  buildKdRange(data, left, leafSize),
		right: buildKdRange(data, right, leafSize),
	}
}

func widestAxis(data [][]float64, idx []int) int {
	best := 0
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
		if spread := mx - mn; spread > bestSpread {
			bestSpread = spread
			best = a
		}
	}
	return best
}

// kdRangeSearch appends every index whose point is within `eps` (Euclidean,
// not squared) of `q` to *out, using `eps2 = eps*eps` for the squared-distance
// comparison. Indices are appended in tree-traversal order — the caller is
// responsible for sorting if the downstream consumer relies on numeric order.
func kdRangeSearch(node *kdRangeNode, data [][]float64, q []float64, eps2 float64, out *[]int) {
	if node == nil {
		return
	}
	if len(node.indices) > 0 {
		for _, i := range node.indices {
			if squaredEuclidean(data[i], q) <= eps2 {
				*out = append(*out, i)
			}
		}
		return
	}
	pivot := data[node.pivot]
	if squaredEuclidean(pivot, q) <= eps2 {
		*out = append(*out, node.pivot)
	}
	diff := q[node.axis] - pivot[node.axis]
	near, far := node.left, node.right
	if diff > 0 {
		near, far = node.right, node.left
	}
	kdRangeSearch(near, data, q, eps2, out)
	// Standard KD pruning: only descend into the far subtree if the splitting
	// hyperplane could intersect the ε-ball.
	if diff*diff <= eps2 {
		kdRangeSearch(far, data, q, eps2, out)
	}
}
