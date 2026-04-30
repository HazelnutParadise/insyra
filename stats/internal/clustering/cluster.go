package clustering

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

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
		for j := 0; j < i; j++ {
			d := euclidean(data[i], data[j])
			dist[i][j] = d
			dist[j][i] = d
		}
	}
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

	best := (*KMeansResult)(nil)
	for start := 0; start < opts.NStart; start++ {
		centerIdx := rng.sampleInt(len(initPool), centers)
		current, err := kmeansSingleStart(data, initPool, centerIdx, opts.IterMax)
		if err != nil {
			return nil, err
		}
		// Keep the first start when objectives are equal within floating-point noise.
		// This matches R's deterministic "best nstart" behavior more closely than
		// replacing the incumbent on sub-ulp drift.
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

	for i, row := range data {
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
	}

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

	clusters := map[int]*clusterNode{}
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

	dists := map[[2]int]float64{}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if method == "ward.d2" {
				dists[pairKey(i, j)] = squaredEuclidean(data[i], data[j])
			} else {
				dists[pairKey(i, j)] = euclidean(data[i], data[j])
			}
		}
	}

	merge := make([][2]int, 0, n-1)
	height := make([]float64, 0, n-1)
	nextID := n
	for step := 1; step < n; step++ {
		aID, bID, dist := pickClosestPair(active, clusters, dists)
		a := clusters[aID]
		b := clusters[bID]
		left, right := orientClusters(a, b)

		merge = append(merge, [2]int{left.rID, right.rID})
		if method == "ward.d2" {
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
			dists[pairKey(id, nextID)] = updatedDistance(method, clusters[id], a, b, dists[pairKey(id, aID)], dists[pairKey(id, bID)], dist)
			delete(dists, pairKey(id, aID))
			delete(dists, pairKey(id, bID))
		}
		delete(dists, pairKey(aID, bID))
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

	neighbors := make([][]int, n)
	isSeed := make([]bool, n)
	for i := range n {
		for j := range n {
			if euclidean(data[i], data[j]) <= eps {
				neighbors[i] = append(neighbors[i], j)
			}
		}
		if len(neighbors[i]) >= minPts {
			isSeed[i] = true
		}
	}

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
	sum := 0.0
	for i, label := range labels {
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
		for otherLabel, members := range clusterMembers {
			if otherLabel == label {
				continue
			}
			avg := 0.0
			for _, j := range members {
				avg += dist[i][j]
			}
			avg /= float64(len(members))
			if avg < bestB || (almostEqual(avg, bestB) && otherLabel < neighborLabel) || neighborLabel == 0 {
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
		sum += s
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

func pickClosestPair(active []int, clusters map[int]*clusterNode, dists map[[2]int]float64) (int, int, float64) {
	bestI, bestJ := 0, 1
	bestDist := math.Inf(1)
	for i := 0; i < len(active); i++ {
		for j := i + 1; j < len(active); j++ {
			a, b := active[i], active[j]
			d := dists[pairKey(a, b)]
			ca, cb := orientClusters(clusters[a], clusters[b])
			if d < bestDist || (almostEqual(d, bestDist) && tieBreakPair(ca, cb, clusters[bestI], clusters[bestJ])) {
				bestDist = d
				bestI = a
				bestJ = b
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
