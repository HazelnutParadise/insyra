package knn

import (
	"errors"
	"math"
	"sort"
)

type Weighting string
type Algorithm string

const (
	UniformWeighting  Weighting = "uniform"
	DistanceWeighting Weighting = "distance"

	AutoAlgorithm       Algorithm = "auto"
	BruteForceAlgorithm Algorithm = "brute"
	KDTreeAlgorithm     Algorithm = "kd_tree"
	BallTreeAlgorithm   Algorithm = "ball_tree"
)

type Options struct {
	Weighting Weighting
	Algorithm Algorithm
	LeafSize  int
}

type ClassificationResult struct {
	Predictions   []string
	Classes       []string
	Probabilities [][]float64
}

type RegressionResult struct {
	Predictions []float64
}

type NeighborResult struct {
	Indices   [][]int
	Distances [][]float64
}

type neighbor struct {
	index int
	dist2 float64
}

type searcher interface {
	QueryKNN(query []float64, k int) []neighbor
}

func Classify(train, test [][]float64, labels []string, k int, opts Options) (*ClassificationResult, error) {
	normalized, err := normalizeOptions(opts)
	if err != nil {
		return nil, err
	}
	if err := validateInputs(train, test, k); err != nil {
		return nil, err
	}
	if len(labels) != len(train) {
		return nil, errors.New("labels length must match training row count")
	}
	search, err := newSearcher(train, normalized)
	if err != nil {
		return nil, err
	}

	classes, classIndex := orderedClasses(labels)
	predictions := make([]string, len(test))
	probabilities := make([][]float64, len(test))
	for i, row := range test {
		neighbors := search.QueryKNN(row, k)
		probabilities[i] = classifyProbabilities(neighbors, labels, classIndex, len(classes), normalized.Weighting)
		best := 0
		bestMeanDistance := classMeanDistance(neighbors, labels, classIndex, 0)
		for c := 1; c < len(classes); c++ {
			if probabilities[i][c] > probabilities[i][best] && !almostEqual(probabilities[i][c], probabilities[i][best]) {
				best = c
				bestMeanDistance = classMeanDistance(neighbors, labels, classIndex, c)
				continue
			}
			if almostEqual(probabilities[i][c], probabilities[i][best]) {
				currentMeanDistance := classMeanDistance(neighbors, labels, classIndex, c)
				if currentMeanDistance < bestMeanDistance && !almostEqual(currentMeanDistance, bestMeanDistance) {
					best = c
					bestMeanDistance = currentMeanDistance
				}
			}
		}
		predictions[i] = classes[best]
	}
	return &ClassificationResult{
		Predictions:   predictions,
		Classes:       classes,
		Probabilities: probabilities,
	}, nil
}

func Regress(train, test [][]float64, targets []float64, k int, opts Options) (*RegressionResult, error) {
	normalized, err := normalizeOptions(opts)
	if err != nil {
		return nil, err
	}
	if err := validateInputs(train, test, k); err != nil {
		return nil, err
	}
	if len(targets) != len(train) {
		return nil, errors.New("targets length must match training row count")
	}
	search, err := newSearcher(train, normalized)
	if err != nil {
		return nil, err
	}

	predictions := make([]float64, len(test))
	for i, row := range test {
		neighbors := search.QueryKNN(row, k)
		predictions[i] = regressPrediction(neighbors, targets, normalized.Weighting)
	}
	return &RegressionResult{Predictions: predictions}, nil
}

func Neighbors(train, test [][]float64, k int, opts Options) (*NeighborResult, error) {
	normalized, err := normalizeOptions(opts)
	if err != nil {
		return nil, err
	}
	if err := validateInputs(train, test, k); err != nil {
		return nil, err
	}
	search, err := newSearcher(train, normalized)
	if err != nil {
		return nil, err
	}

	indices := make([][]int, len(test))
	distances := make([][]float64, len(test))
	for i, row := range test {
		neighbors := search.QueryKNN(row, k)
		indices[i] = make([]int, len(neighbors))
		distances[i] = make([]float64, len(neighbors))
		for j, nb := range neighbors {
			indices[i][j] = nb.index
			distances[i][j] = math.Sqrt(nb.dist2)
		}
	}
	return &NeighborResult{Indices: indices, Distances: distances}, nil
}

func normalizeOptions(opts Options) (Options, error) {
	if opts.Weighting == "" {
		opts.Weighting = UniformWeighting
	}
	switch opts.Weighting {
	case UniformWeighting, DistanceWeighting:
	default:
		return Options{}, errors.New("unsupported KNN weighting")
	}

	if opts.Algorithm == "" {
		opts.Algorithm = AutoAlgorithm
	}
	switch opts.Algorithm {
	case AutoAlgorithm, BruteForceAlgorithm, KDTreeAlgorithm, BallTreeAlgorithm:
	default:
		return Options{}, errors.New("unsupported KNN algorithm")
	}
	if opts.LeafSize <= 0 {
		opts.LeafSize = 16
	}
	return opts, nil
}

func validateInputs(train, test [][]float64, k int) error {
	if len(train) == 0 || len(train[0]) == 0 {
		return errors.New("training data must have at least 1 row and 1 column")
	}
	if len(test) == 0 || len(test[0]) == 0 {
		return errors.New("test data must have at least 1 row and 1 column")
	}
	if k <= 0 {
		return errors.New("k must be greater than 0")
	}
	if k > len(train) {
		return errors.New("k must not exceed training row count")
	}
	p := len(train[0])
	for _, row := range train {
		if len(row) != p {
			return errors.New("training rows must all have the same dimension")
		}
		if hasInvalidFloat(row) {
			return errors.New("training data contains invalid numeric values")
		}
	}
	for _, row := range test {
		if len(row) != p {
			return errors.New("training and test data must have the same column count")
		}
		if hasInvalidFloat(row) {
			return errors.New("test data contains invalid numeric values")
		}
	}
	return nil
}

func newSearcher(train [][]float64, opts Options) (searcher, error) {
	switch resolveAlgorithm(len(train), len(train[0]), opts.Algorithm) {
	case BruteForceAlgorithm:
		return &bruteSearcher{train: train}, nil
	case KDTreeAlgorithm:
		return newKDTreeSearcher(train, opts.LeafSize), nil
	case BallTreeAlgorithm:
		return newBallTreeSearcher(train, opts.LeafSize), nil
	default:
		return nil, errors.New("unsupported KNN algorithm")
	}
}

func resolveAlgorithm(n, dims int, algo Algorithm) Algorithm {
	if algo != AutoAlgorithm {
		return algo
	}
	if n < 64 {
		return BruteForceAlgorithm
	}
	if dims <= 8 {
		return KDTreeAlgorithm
	}
	return BallTreeAlgorithm
}

type bruteSearcher struct {
	train [][]float64
}

func (s *bruteSearcher) QueryKNN(query []float64, k int) []neighbor {
	set := newNeighborSet(k)
	for i, row := range s.train {
		set.tryAdd(neighbor{index: i, dist2: squaredEuclidean(row, query)})
	}
	return set.results()
}

type kdTreeSearcher struct {
	train    [][]float64
	leafSize int
	root     *kdNode
}

type kdNode struct {
	axis    int
	pivot   int
	indices []int
	left    *kdNode
	right   *kdNode
}

func newKDTreeSearcher(train [][]float64, leafSize int) *kdTreeSearcher {
	indices := make([]int, len(train))
	for i := range indices {
		indices[i] = i
	}
	return &kdTreeSearcher{
		train:    train,
		leafSize: leafSize,
		root:     buildKDNode(train, indices, leafSize),
	}
}

func buildKDNode(train [][]float64, indices []int, leafSize int) *kdNode {
	if len(indices) == 0 {
		return nil
	}
	if len(indices) <= leafSize {
		out := append([]int(nil), indices...)
		sort.Ints(out)
		return &kdNode{indices: out}
	}

	axis := widestDimension(train, indices)
	sort.Slice(indices, func(i, j int) bool {
		ai := train[indices[i]][axis]
		aj := train[indices[j]][axis]
		if almostEqual(ai, aj) {
			return indices[i] < indices[j]
		}
		return ai < aj
	})
	mid := len(indices) / 2
	pivot := indices[mid]
	left := append([]int(nil), indices[:mid]...)
	right := append([]int(nil), indices[mid+1:]...)
	return &kdNode{
		axis:  axis,
		pivot: pivot,
		left:  buildKDNode(train, left, leafSize),
		right: buildKDNode(train, right, leafSize),
	}
}

func (s *kdTreeSearcher) QueryKNN(query []float64, k int) []neighbor {
	set := newNeighborSet(k)
	searchKDTree(s.train, s.root, query, set)
	return set.results()
}

func searchKDTree(train [][]float64, node *kdNode, query []float64, set *neighborSet) {
	if node == nil {
		return
	}
	if len(node.indices) > 0 {
		for _, idx := range node.indices {
			set.tryAdd(neighbor{index: idx, dist2: squaredEuclidean(train[idx], query)})
		}
		return
	}

	pivotPoint := train[node.pivot]
	set.tryAdd(neighbor{index: node.pivot, dist2: squaredEuclidean(pivotPoint, query)})
	diff := query[node.axis] - pivotPoint[node.axis]
	near, far := node.left, node.right
	if diff > 0 {
		near, far = node.right, node.left
	}
	searchKDTree(train, near, query, set)
	if !set.full() || diff*diff <= set.worstDist2()+1e-12 {
		searchKDTree(train, far, query, set)
	}
}

type ballTreeSearcher struct {
	train    [][]float64
	leafSize int
	root     *ballNode
}

type ballNode struct {
	center  []float64
	radius  float64
	indices []int
	left    *ballNode
	right   *ballNode
}

func newBallTreeSearcher(train [][]float64, leafSize int) *ballTreeSearcher {
	indices := make([]int, len(train))
	for i := range indices {
		indices[i] = i
	}
	return &ballTreeSearcher{
		train:    train,
		leafSize: leafSize,
		root:     buildBallNode(train, indices, leafSize),
	}
}

func buildBallNode(train [][]float64, indices []int, leafSize int) *ballNode {
	if len(indices) == 0 {
		return nil
	}
	center := centroid(train, indices)
	radius := 0.0
	for _, idx := range indices {
		d := math.Sqrt(squaredEuclidean(train[idx], center))
		if d > radius {
			radius = d
		}
	}
	if len(indices) <= leafSize {
		out := append([]int(nil), indices...)
		sort.Ints(out)
		return &ballNode{center: center, radius: radius, indices: out}
	}

	leftPivot, rightPivot := chooseBallPivots(train, indices)
	left := make([]int, 0, len(indices)/2)
	right := make([]int, 0, len(indices)/2)
	for _, idx := range indices {
		dl := squaredEuclidean(train[idx], train[leftPivot])
		dr := squaredEuclidean(train[idx], train[rightPivot])
		if dl < dr || (almostEqual(dl, dr) && idx <= rightPivot) {
			left = append(left, idx)
		} else {
			right = append(right, idx)
		}
	}
	if len(left) == 0 || len(right) == 0 {
		sorted := append([]int(nil), indices...)
		sort.Ints(sorted)
		half := len(sorted) / 2
		left = sorted[:half]
		right = sorted[half:]
	}
	return &ballNode{
		center: center,
		radius: radius,
		left:   buildBallNode(train, left, leafSize),
		right:  buildBallNode(train, right, leafSize),
	}
}

func (s *ballTreeSearcher) QueryKNN(query []float64, k int) []neighbor {
	set := newNeighborSet(k)
	searchBallTree(s.train, s.root, query, set)
	return set.results()
}

func searchBallTree(train [][]float64, node *ballNode, query []float64, set *neighborSet) {
	if node == nil {
		return
	}
	if len(node.indices) > 0 {
		for _, idx := range node.indices {
			set.tryAdd(neighbor{index: idx, dist2: squaredEuclidean(train[idx], query)})
		}
		return
	}

	leftBound := ballLowerBound(query, node.left)
	rightBound := ballLowerBound(query, node.right)
	first, second := node.left, node.right
	firstBound, secondBound := leftBound, rightBound
	if rightBound < leftBound && !almostEqual(rightBound, leftBound) {
		first, second = node.right, node.left
		firstBound, secondBound = rightBound, leftBound
	}
	if !set.full() || firstBound <= set.worstDist2()+1e-12 {
		searchBallTree(train, first, query, set)
	}
	if !set.full() || secondBound <= set.worstDist2()+1e-12 {
		searchBallTree(train, second, query, set)
	}
}

func ballLowerBound(query []float64, node *ballNode) float64 {
	if node == nil {
		return math.Inf(1)
	}
	dist := math.Sqrt(squaredEuclidean(query, node.center)) - node.radius
	if dist < 0 {
		return 0
	}
	return dist * dist
}

type neighborSet struct {
	k   int
	buf []neighbor
}

func newNeighborSet(k int) *neighborSet {
	return &neighborSet{k: k, buf: make([]neighbor, 0, k)}
}

func (s *neighborSet) full() bool {
	return len(s.buf) >= s.k
}

func (s *neighborSet) worstDist2() float64 {
	if len(s.buf) == 0 {
		return math.Inf(1)
	}
	worst := s.buf[0]
	for _, nb := range s.buf[1:] {
		if worseNeighbor(nb, worst) {
			worst = nb
		}
	}
	return worst.dist2
}

func (s *neighborSet) tryAdd(nb neighbor) {
	if len(s.buf) < s.k {
		s.buf = append(s.buf, nb)
		return
	}
	worstIdx := 0
	for i := 1; i < len(s.buf); i++ {
		if worseNeighbor(s.buf[i], s.buf[worstIdx]) {
			worstIdx = i
		}
	}
	if betterNeighbor(nb, s.buf[worstIdx]) {
		s.buf[worstIdx] = nb
	}
}

func (s *neighborSet) results() []neighbor {
	out := append([]neighbor(nil), s.buf...)
	sort.Slice(out, func(i, j int) bool {
		if almostEqual(out[i].dist2, out[j].dist2) {
			return out[i].index < out[j].index
		}
		return out[i].dist2 < out[j].dist2
	})
	return out
}

func betterNeighbor(a, b neighbor) bool {
	if almostEqual(a.dist2, b.dist2) {
		return a.index < b.index
	}
	return a.dist2 < b.dist2
}

func worseNeighbor(a, b neighbor) bool {
	if almostEqual(a.dist2, b.dist2) {
		return a.index > b.index
	}
	return a.dist2 > b.dist2
}

func orderedClasses(labels []string) ([]string, map[string]int) {
	classes := make([]string, 0)
	classIndex := make(map[string]int, len(labels))
	for _, label := range labels {
		if _, ok := classIndex[label]; ok {
			continue
		}
		classIndex[label] = len(classes)
		classes = append(classes, label)
	}
	return classes, classIndex
}

func classifyProbabilities(neighbors []neighbor, labels []string, classIndex map[string]int, nClasses int, weighting Weighting) []float64 {
	weights := make([]float64, nClasses)
	// Distance weighting must collapse to zero-distance neighbors only,
	// because 1/dist diverges as dist → 0. Uniform weighting has no such
	// pathology and must use ALL k neighbors equally — previously this
	// branch ignored the weighting argument and incorrectly dropped the
	// non-matching neighbors for uniform mode too, which made KNNClassify
	// disagree with KNearestNeighbors whenever a test point coincided
	// with a training point.
	hasZeroDistance := false
	if weighting == DistanceWeighting {
		for _, nb := range neighbors {
			if almostEqual(nb.dist2, 0) {
				hasZeroDistance = true
				break
			}
		}
	}
	for _, nb := range neighbors {
		if hasZeroDistance && !almostEqual(nb.dist2, 0) {
			continue
		}
		idx := classIndex[labels[nb.index]]
		weights[idx] += neighborWeight(math.Sqrt(nb.dist2), weighting)
	}
	total := 0.0
	for _, weight := range weights {
		total += weight
	}
	if total == 0 {
		return weights
	}
	for i := range weights {
		weights[i] /= total
	}
	return weights
}

func classMeanDistance(neighbors []neighbor, labels []string, classIndex map[string]int, class int) float64 {
	sum := 0.0
	count := 0.0
	for _, nb := range neighbors {
		if classIndex[labels[nb.index]] != class {
			continue
		}
		sum += math.Sqrt(nb.dist2)
		count++
	}
	if count == 0 {
		return math.Inf(1)
	}
	return sum / count
}

func regressPrediction(neighbors []neighbor, targets []float64, weighting Weighting) float64 {
	// See classifyProbabilities for the rationale: zero-distance collapse is
	// only required for distance weighting (where weights diverge as 1/dist).
	hasZeroDistance := false
	if weighting == DistanceWeighting {
		for _, nb := range neighbors {
			if almostEqual(nb.dist2, 0) {
				hasZeroDistance = true
				break
			}
		}
	}
	sumWeight := 0.0
	sumTarget := 0.0
	for _, nb := range neighbors {
		if hasZeroDistance && !almostEqual(nb.dist2, 0) {
			continue
		}
		weight := neighborWeight(math.Sqrt(nb.dist2), weighting)
		sumWeight += weight
		sumTarget += weight * targets[nb.index]
	}
	if sumWeight == 0 {
		return math.NaN()
	}
	return sumTarget / sumWeight
}

func neighborWeight(distance float64, weighting Weighting) float64 {
	switch weighting {
	case DistanceWeighting:
		if almostEqual(distance, 0) {
			return 1
		}
		return 1 / distance
	default:
		return 1
	}
}

func widestDimension(train [][]float64, indices []int) int {
	bestAxis := 0
	bestSpread := -1.0
	for axis := range train[0] {
		minV := train[indices[0]][axis]
		maxV := minV
		for _, idx := range indices[1:] {
			v := train[idx][axis]
			if v < minV {
				minV = v
			}
			if v > maxV {
				maxV = v
			}
		}
		spread := maxV - minV
		if spread > bestSpread {
			bestSpread = spread
			bestAxis = axis
		}
	}
	return bestAxis
}

func chooseBallPivots(train [][]float64, indices []int) (int, int) {
	first := indices[0]
	farthest := first
	best := -1.0
	for _, idx := range indices[1:] {
		d := squaredEuclidean(train[first], train[idx])
		if d > best {
			best = d
			farthest = idx
		}
	}
	second := farthest
	best = -1.0
	for _, idx := range indices {
		d := squaredEuclidean(train[farthest], train[idx])
		if d > best {
			best = d
			second = idx
		}
	}
	if farthest == second && len(indices) > 1 {
		second = indices[1]
	}
	return farthest, second
}

func centroid(train [][]float64, indices []int) []float64 {
	center := make([]float64, len(train[0]))
	for _, idx := range indices {
		for j, v := range train[idx] {
			center[j] += v
		}
	}
	for j := range center {
		center[j] /= float64(len(indices))
	}
	return center
}

func squaredEuclidean(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return sum
}

func hasInvalidFloat(row []float64) bool {
	for _, v := range row {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return true
		}
	}
	return false
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= 1e-12
}
