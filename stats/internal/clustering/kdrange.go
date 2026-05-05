package clustering

import "sort"

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
