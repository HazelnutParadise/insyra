package ml

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/HazelnutParadise/insyra"
)

// DecisionTreeOptions controls both decision-tree classifiers and regressors.
// A zero bound means unlimited, except MinSamplesLeaf and MaxBins, which use
// their documented defaults.
type DecisionTreeOptions struct {
	MaxDepth            int
	MaxLeaves           int
	MinSamplesLeaf      int
	MaxBins             int
	CategoricalFeatures []string
}

// DecisionTreeClassifierOptions and DecisionTreeRegressorOptions are aliases
// kept for callers that prefer a model-specific option name.
type DecisionTreeClassifierOptions = DecisionTreeOptions
type DecisionTreeRegressorOptions = DecisionTreeOptions

// DecisionTreeNode is one fitted node. A leaf has IsLeaf set and reports its
// double-precision Value or Probabilities. Internal nodes expose the split
// used to reach their children.
type DecisionTreeNode struct {
	IsLeaf               bool
	Feature              int
	FeatureName          string
	Categorical          bool
	Threshold            float64
	Categories           []any
	MissingGoLeft        bool
	UnseenCategoryGoLeft bool
	Samples              int
	Gain                 float64
	Impurity             float64
	Value                float64
	Prediction           any
	Probabilities        []float64
	ClassCounts          []int64
	Left                 *DecisionTreeNode
	Right                *DecisionTreeNode

	splitBin       int
	leftCategories map[uint32]struct{}
}

// DecisionTreeClassifier is a fitted histogram decision tree for labels.
type DecisionTreeClassifier struct {
	modelBase
	Root               *DecisionTreeNode
	classes            *insyra.DataList
	featureSchemas     []treeFeature
	featureImportances []float64
}

// DecisionTreeRegressor is a fitted histogram decision tree for continuous
// targets.
type DecisionTreeRegressor struct {
	modelBase
	Root               *DecisionTreeNode
	featureSchemas     []treeFeature
	featureImportances []float64
	quantizerScale     float64
}

// FitDecisionTreeClassifier fits a deterministic classification tree.
func FitDecisionTreeClassifier(x *insyra.DataTable, y *insyra.DataList, opts ...DecisionTreeOptions) (*DecisionTreeClassifier, error) {
	options, err := oneDecisionTreeOption(opts)
	if err != nil {
		return nil, err
	}
	fit, err := fitDecisionTree(x, y, options, true)
	if err != nil {
		return nil, err
	}
	return &DecisionTreeClassifier{
		modelBase:          modelBase{features: fit.features},
		Root:               fit.root,
		classes:            insyra.NewDataList(fit.classes...).SetName("classes"),
		featureSchemas:     fit.schemas,
		featureImportances: fit.importances,
	}, nil
}

// FitDecisionTreeRegressor fits a deterministic regression tree.
func FitDecisionTreeRegressor(x *insyra.DataTable, y *insyra.DataList, opts ...DecisionTreeOptions) (*DecisionTreeRegressor, error) {
	options, err := oneDecisionTreeOption(opts)
	if err != nil {
		return nil, err
	}
	fit, err := fitDecisionTree(x, y, options, false)
	if err != nil {
		return nil, err
	}
	return &DecisionTreeRegressor{
		modelBase:          modelBase{features: fit.features},
		Root:               fit.root,
		featureSchemas:     fit.schemas,
		featureImportances: fit.importances,
		quantizerScale:     fit.quantizerScale,
	}, nil
}

func oneDecisionTreeOption(opts []DecisionTreeOptions) (DecisionTreeOptions, error) {
	if len(opts) > 1 {
		return DecisionTreeOptions{}, errors.New("ml: opts accepts at most one value")
	}
	options := DecisionTreeOptions{}
	if len(opts) == 1 {
		options = opts[0]
	}
	if options.MaxDepth < 0 || options.MaxLeaves < 0 || options.MinSamplesLeaf < 0 || options.MaxBins < 0 {
		return DecisionTreeOptions{}, errors.New("ml: decision-tree bounds must not be negative")
	}
	if options.MinSamplesLeaf == 0 {
		options.MinSamplesLeaf = 1
	}
	if options.MaxBins == 0 {
		options.MaxBins = 32
	}
	if options.MaxBins < 2 {
		return DecisionTreeOptions{}, errors.New("ml: MaxBins must be at least 2")
	}
	return options, nil
}

type treeFeature struct {
	name         string
	categorical  bool
	edges        []float32
	categoryCode map[string]uint32
	categoryVals []any
}

type treeFit struct {
	features       []string
	schemas        []treeFeature
	classes        []any
	root           *DecisionTreeNode
	importances    []float64
	quantizerScale float64
}

func fitDecisionTree(x *insyra.DataTable, y *insyra.DataList, options DecisionTreeOptions, classification bool) (*treeFit, error) {
	features, columns, err := fitFeatures(x)
	if err != nil {
		return nil, err
	}
	if y == nil {
		return nil, errors.New("ml: target list is nil")
	}
	if len(features) == 0 {
		return nil, errors.New("ml: decision tree requires at least one feature")
	}
	n := x.NumRows()
	if n == 0 {
		return nil, errors.New("ml: decision tree requires at least one observation")
	}
	if y.Len() != n {
		return nil, fmt.Errorf("ml: target length %d does not match training row count %d", y.Len(), n)
	}

	categorical := make(map[string]bool, len(options.CategoricalFeatures))
	knownFeatures := make(map[string]struct{}, len(features))
	for _, feature := range features {
		knownFeatures[feature] = struct{}{}
	}
	for _, name := range options.CategoricalFeatures {
		if _, ok := knownFeatures[name]; !ok {
			return nil, fmt.Errorf("ml: categorical feature %q is not fitted", name)
		}
		categorical[name] = true
	}

	schemas := make([]treeFeature, len(features))
	encoded := make([][]uint32, len(features))
	for j, column := range columns {
		schema, values, err := encodeTrainingFeature(features[j], column.Data(), n, categorical[features[j]], options.MaxBins)
		if err != nil {
			return nil, err
		}
		schemas[j] = schema
		encoded[j] = values
	}

	var classes []any
	var targetClasses []int
	if classification {
		classes, targetClasses, err = treeClasses(y.Data(), n)
		if err != nil {
			return nil, err
		}
	} else {
		classes = nil
		targetClasses = nil
	}

	var targetQ, targetQ2 []int64
	quantizerScale := 0.0
	if classification {
		targetQ = nil
		targetQ2 = nil
	} else {
		targetQ, targetQ2, quantizerScale, err = quantizeTargets(y.Data(), n)
		if err != nil {
			return nil, err
		}
	}

	rows := make([]int, n)
	for i := range rows {
		rows[i] = i
	}
	growth := &treeGrowth{
		features:       schemas,
		encoded:        encoded,
		classification: classification,
		classCount:     len(classes),
		targetClasses:  targetClasses,
		targetQ:        targetQ,
		targetQ2:       targetQ2,
		quantizerScale: quantizerScale,
		options:        options,
		leafCount:      1,
		importances:    make([]float64, len(features)),
		classes:        classes,
	}
	root := growth.grow(rows, 0)
	importanceTotal := 0.0
	for _, value := range growth.importances {
		importanceTotal += value
	}
	if importanceTotal != 0 {
		for i := range growth.importances {
			growth.importances[i] /= importanceTotal
		}
	}
	return &treeFit{
		features:       features,
		schemas:        schemas,
		classes:        classes,
		root:           root,
		importances:    append([]float64(nil), growth.importances...),
		quantizerScale: quantizerScale,
	}, nil
}

func encodeTrainingFeature(name string, raw []any, n int, categorical bool, maxBins int) (treeFeature, []uint32, error) {
	if len(raw) != n {
		return treeFeature{}, nil, fmt.Errorf("ml: feature %q has %d rows, want %d", name, len(raw), n)
	}
	feature := treeFeature{name: name, categorical: categorical}
	encoded := make([]uint32, n)
	if categorical {
		keys := make(map[string]any)
		for i, value := range raw {
			if treeMissing(value) {
				continue
			}
			key := treeCategoryKey(value)
			keys[key] = value
			encoded[i] = 1
		}
		ordered := make([]string, 0, len(keys))
		for key := range keys {
			ordered = append(ordered, key)
		}
		sort.Strings(ordered)
		feature.categoryCode = make(map[string]uint32, len(ordered))
		feature.categoryVals = make([]any, len(ordered)+1)
		for i, key := range ordered {
			code := uint32(i + 1)
			feature.categoryCode[key] = code
			feature.categoryVals[code] = keys[key]
		}
		for i, value := range raw {
			if treeMissing(value) {
				encoded[i] = 0
			} else {
				encoded[i] = feature.categoryCode[treeCategoryKey(value)]
			}
		}
		return feature, encoded, nil
	}

	observed := make([]float32, 0, n)
	for i, value := range raw {
		if treeMissing(value) {
			continue
		}
		converted, ok := insyra.ToFloat64Safe(value)
		if !ok {
			return treeFeature{}, nil, fmt.Errorf("ml: feature %q contains non-numeric value at row %d", name, i)
		}
		observed = append(observed, float32(converted))
	}
	sort.Slice(observed, func(i, j int) bool { return observed[i] < observed[j] })
	feature.edges = quantileEdges(observed, maxBins)
	for i, value := range raw {
		if treeMissing(value) {
			encoded[i] = 0
			continue
		}
		converted, _ := insyra.ToFloat64Safe(value)
		encoded[i] = numericBin(feature.edges, float32(converted))
	}
	return feature, encoded, nil
}

func quantileEdges(sortedValues []float32, maxBins int) []float32 {
	if len(sortedValues) < 2 {
		return nil
	}
	edges := make([]float32, 0, maxBins-1)
	for bin := 1; bin < maxBins; bin++ {
		position := float64(bin) * float64(len(sortedValues)-1) / float64(maxBins)
		lower := int(math.Floor(position))
		upper := int(math.Ceil(position))
		value := sortedValues[lower]
		if upper != lower {
			value = float32(float64(sortedValues[lower]) + (float64(sortedValues[upper])-float64(sortedValues[lower]))*(position-float64(lower)))
		}
		if len(edges) == 0 || value > edges[len(edges)-1] {
			edges = append(edges, value)
		}
	}
	return edges
}

func numericBin(edges []float32, value float32) uint32 {
	return uint32(sort.Search(len(edges), func(i int) bool { return value <= edges[i] }) + 1)
}

func treeMissing(value any) bool {
	if value == nil {
		return true
	}
	switch value := value.(type) {
	case float32:
		return math.IsNaN(float64(value))
	case float64:
		return math.IsNaN(value)
	default:
		return false
	}
}

func treeCategoryKey(value any) string {
	return fmt.Sprintf("%T:%#v", value, value)
}

func treeClasses(raw []any, n int) ([]any, []int, error) {
	if len(raw) != n {
		return nil, nil, fmt.Errorf("ml: target length %d does not match training row count %d", len(raw), n)
	}
	valuesByKey := make(map[string]any)
	keys := make([]string, 0)
	for i, value := range raw {
		if treeMissing(value) {
			return nil, nil, fmt.Errorf("ml: classification target is missing at row %d", i)
		}
		key := treeCategoryKey(value)
		if _, ok := valuesByKey[key]; !ok {
			valuesByKey[key] = value
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	classes := make([]any, len(keys))
	classByKey := make(map[string]int, len(keys))
	for i, key := range keys {
		classes[i] = valuesByKey[key]
		classByKey[key] = i
	}
	targets := make([]int, n)
	for i, value := range raw {
		targets[i] = classByKey[treeCategoryKey(value)]
	}
	return classes, targets, nil
}

func quantizeTargets(raw []any, n int) ([]int64, []int64, float64, error) {
	values := make([]float64, n)
	maxAbs := 0.0
	for i, value := range raw {
		converted, ok := insyra.ToFloat64Safe(value)
		if !ok || math.IsNaN(converted) || math.IsInf(converted, 0) {
			return nil, nil, 0, fmt.Errorf("ml: regression target must be finite numeric at row %d", i)
		}
		values[i] = converted
		if abs := math.Abs(converted); abs > maxAbs {
			maxAbs = abs
		}
	}
	const maxInt64 = int64(^uint64(0) >> 1)
	maxQuantized := math.Sqrt(float64(maxInt64) / (4 * float64(max(1, n))))
	scale := 1e6
	if maxAbs > 0 && maxQuantized/maxAbs < scale {
		scale = maxQuantized / maxAbs
	}
	if scale <= 0 || math.IsNaN(scale) {
		scale = 1
	}
	quantized := make([]int64, n)
	squares := make([]int64, n)
	for i, value := range values {
		qFloat := math.Round(value * scale)
		if qFloat > maxQuantized {
			qFloat = maxQuantized
		}
		if qFloat < -maxQuantized {
			qFloat = -maxQuantized
		}
		q := int64(qFloat)
		quantized[i] = q
		squares[i] = q * q
	}
	return quantized, squares, scale, nil
}

type treeStats struct {
	count       int
	classCounts []int64
	sum         int64
	sumSquares  int64
}

func newTreeStats(classCount int, classification bool) treeStats {
	stats := treeStats{}
	if classification {
		stats.classCounts = make([]int64, classCount)
	}
	return stats
}

func (s *treeStats) add(other treeStats) {
	s.count += other.count
	s.sum += other.sum
	s.sumSquares += other.sumSquares
	for i := range s.classCounts {
		s.classCounts[i] += other.classCounts[i]
	}
}

func (s *treeStats) subtract(other treeStats) {
	s.count -= other.count
	s.sum -= other.sum
	s.sumSquares -= other.sumSquares
	for i := range s.classCounts {
		s.classCounts[i] -= other.classCounts[i]
	}
}

func (s treeStats) clone() treeStats {
	s.classCounts = append([]int64(nil), s.classCounts...)
	return s
}

type treeGrowth struct {
	features       []treeFeature
	encoded        [][]uint32
	classification bool
	classCount     int
	targetClasses  []int
	targetQ        []int64
	targetQ2       []int64
	quantizerScale float64
	options        DecisionTreeOptions
	leafCount      int
	classes        []any
	importances    []float64
}

func (g *treeGrowth) statsFor(rows []int) treeStats {
	stats := newTreeStats(g.classCount, g.classification)
	for _, row := range rows {
		stats.count++
		if g.classification {
			stats.classCounts[g.targetClasses[row]]++
		} else {
			stats.sum += g.targetQ[row]
			stats.sumSquares += g.targetQ2[row]
		}
	}
	return stats
}

func (g *treeGrowth) grow(rows []int, depth int) *DecisionTreeNode {
	stats := g.statsFor(rows)
	node := g.leafNode(stats)
	if g.options.MaxDepth > 0 && depth >= g.options.MaxDepth {
		return node
	}
	if g.options.MaxLeaves > 0 && g.leafCount >= g.options.MaxLeaves {
		return node
	}
	if g.options.MinSamplesLeaf > len(rows)/2 {
		return node
	}
	candidate := g.bestSplit(rows, stats)
	if candidate == nil || candidate.gain <= 0 {
		return node
	}
	leftRows, rightRows := g.partition(rows, candidate)
	if len(leftRows) < g.options.MinSamplesLeaf || len(rightRows) < g.options.MinSamplesLeaf {
		return node
	}
	g.leafCount++
	g.importances[candidate.feature] += candidate.gain * float64(stats.count)
	node.IsLeaf = false
	node.Feature = candidate.feature
	node.FeatureName = g.features[candidate.feature].name
	node.Categorical = g.features[candidate.feature].categorical
	node.Threshold = candidate.threshold
	node.Categories = append([]any(nil), candidate.categories...)
	node.MissingGoLeft = candidate.missingLeft
	node.UnseenCategoryGoLeft = candidate.missingLeft
	node.Gain = candidate.gain
	node.Left = g.grow(leftRows, depth+1)
	node.Right = g.grow(rightRows, depth+1)
	node.splitBin = candidate.splitBin
	node.leftCategories = candidate.leftCategories
	return node
}

func (g *treeGrowth) leafNode(stats treeStats) *DecisionTreeNode {
	node := &DecisionTreeNode{IsLeaf: true, Samples: stats.count, Impurity: g.impurity(stats)}
	if g.classification {
		node.ClassCounts = append([]int64(nil), stats.classCounts...)
		node.Probabilities = make([]float64, len(stats.classCounts))
		best := 0
		for i, count := range stats.classCounts {
			if stats.count > 0 {
				node.Probabilities[i] = float64(count) / float64(stats.count)
			}
			if count > stats.classCounts[best] {
				best = i
			}
		}
		if len(g.classes) > 0 {
			node.Prediction = g.classes[best]
			node.Value = node.Probabilities[best]
		}
	} else if stats.count > 0 {
		node.Value = float64(stats.sum) / g.quantizerScale / float64(stats.count)
		node.Prediction = node.Value
	}
	return node
}

func (g *treeGrowth) impurity(stats treeStats) float64 {
	if stats.count == 0 {
		return 0
	}
	if g.classification {
		result := 1.0
		for _, count := range stats.classCounts {
			probability := float64(count) / float64(stats.count)
			result -= probability * probability
		}
		return result
	}
	meanSquare := float64(stats.sumSquares)
	meanSum := float64(stats.sum)
	meanSquare -= meanSum * meanSum / float64(stats.count)
	if meanSquare < 0 {
		return 0
	}
	return meanSquare / (g.quantizerScale * g.quantizerScale)
}

type treeSplit struct {
	feature        int
	splitBin       int
	splitRank      int
	threshold      float64
	categories     []any
	leftCategories map[uint32]struct{}
	missingLeft    bool
	gain           float64
}

func (g *treeGrowth) bestSplit(rows []int, parent treeStats) *treeSplit {
	var best *treeSplit
	for featureIndex, feature := range g.features {
		if feature.categorical {
			best = g.bestCategoricalSplit(rows, parent, featureIndex, best)
		} else {
			best = g.bestNumericSplit(rows, parent, featureIndex, best)
		}
	}
	return best
}

func (g *treeGrowth) bestNumericSplit(rows []int, parent treeStats, featureIndex int, best *treeSplit) *treeSplit {
	feature := g.features[featureIndex]
	observed := newTreeStats(g.classCount, g.classification)
	missing := newTreeStats(g.classCount, g.classification)
	histogram := make([]treeStats, len(feature.edges)+2)
	for i := range histogram {
		histogram[i] = newTreeStats(g.classCount, g.classification)
	}
	for _, row := range rows {
		stats := g.rowStats(row)
		code := g.encoded[featureIndex][row]
		if code == 0 {
			missing.add(stats)
		} else {
			observed.add(stats)
			histogram[code].add(stats)
		}
	}
	leftObserved := newTreeStats(g.classCount, g.classification)
	rightObserved := observed.clone()
	for splitBin := 1; splitBin <= len(feature.edges); splitBin++ {
		leftObserved.add(histogram[splitBin])
		rightObserved.subtract(histogram[splitBin])
		candidate := g.considerDirections(parent, leftObserved, rightObserved, missing, &treeSplit{
			feature:   featureIndex,
			splitBin:  splitBin,
			splitRank: splitBin,
			threshold: float64(feature.edges[splitBin-1]),
		})
		best = g.chooseBetter(best, candidate)
	}
	return best
}

func (g *treeGrowth) bestCategoricalSplit(rows []int, parent treeStats, featureIndex int, best *treeSplit) *treeSplit {
	feature := g.features[featureIndex]
	histogram := make([]treeStats, len(feature.categoryVals))
	missing := newTreeStats(g.classCount, g.classification)
	observed := newTreeStats(g.classCount, g.classification)
	for i := range histogram {
		histogram[i] = newTreeStats(g.classCount, g.classification)
	}
	for _, row := range rows {
		stats := g.rowStats(row)
		code := g.encoded[featureIndex][row]
		if code == 0 {
			missing.add(stats)
		} else {
			observed.add(stats)
			histogram[code].add(stats)
		}
	}
	ordered := make([]uint32, 0, len(histogram)-1)
	for code := uint32(1); int(code) < len(histogram); code++ {
		if histogram[code].count > 0 {
			ordered = append(ordered, code)
		}
	}
	sort.Slice(ordered, func(i, j int) bool {
		left, right := histogram[ordered[i]], histogram[ordered[j]]
		if g.classification {
			for class := range left.classCounts {
				leftRate := float64(left.classCounts[class]) / float64(max(1, left.count))
				rightRate := float64(right.classCounts[class]) / float64(max(1, right.count))
				if leftRate != rightRate {
					return leftRate > rightRate
				}
			}
		} else {
			leftMean := float64(left.sum) / float64(max(1, left.count))
			rightMean := float64(right.sum) / float64(max(1, right.count))
			if leftMean != rightMean {
				return leftMean < rightMean
			}
		}
		return ordered[i] < ordered[j]
	})
	leftObserved := newTreeStats(g.classCount, g.classification)
	rightObserved := observed.clone()
	for rank := 1; rank < len(ordered); rank++ {
		code := ordered[rank-1]
		leftObserved.add(histogram[code])
		rightObserved.subtract(histogram[code])
		leftCodes := make(map[uint32]struct{}, rank)
		categories := make([]any, rank)
		for i := 0; i < rank; i++ {
			leftCodes[ordered[i]] = struct{}{}
			categories[i] = feature.categoryVals[ordered[i]]
		}
		candidate := g.considerDirections(parent, leftObserved, rightObserved, missing, &treeSplit{
			feature:        featureIndex,
			splitRank:      rank,
			categories:     categories,
			leftCategories: leftCodes,
		})
		best = g.chooseBetter(best, candidate)
	}
	return best
}

func (g *treeGrowth) rowStats(row int) treeStats {
	stats := newTreeStats(g.classCount, g.classification)
	stats.count = 1
	if g.classification {
		stats.classCounts[g.targetClasses[row]] = 1
	} else {
		stats.sum = g.targetQ[row]
		stats.sumSquares = g.targetQ2[row]
	}
	return stats
}

func (g *treeGrowth) considerDirections(parent, leftObserved, rightObserved, missing treeStats, base *treeSplit) *treeSplit {
	var best *treeSplit
	for _, missingLeft := range []bool{true, false} {
		left := leftObserved.clone()
		right := rightObserved.clone()
		if missingLeft {
			left.add(missing)
		} else {
			right.add(missing)
		}
		if left.count < g.options.MinSamplesLeaf || right.count < g.options.MinSamplesLeaf {
			continue
		}
		gain := g.impurity(parent) - float64(left.count)/float64(parent.count)*g.impurity(left) - float64(right.count)/float64(parent.count)*g.impurity(right)
		candidate := *base
		candidate.missingLeft = missingLeft
		candidate.gain = gain
		if g.chooseBetter(best, &candidate) == &candidate {
			best = &candidate
		}
	}
	return best
}

func (g *treeGrowth) chooseBetter(current, candidate *treeSplit) *treeSplit {
	if candidate == nil || candidate.gain <= 0 {
		return current
	}
	if current == nil || betterTreeSplit(candidate, current) {
		return candidate
	}
	return current
}

func betterTreeSplit(left, right *treeSplit) bool {
	if left.gain != right.gain {
		return left.gain > right.gain
	}
	if left.feature != right.feature {
		return left.feature < right.feature
	}
	if left.splitRank != right.splitRank {
		return left.splitRank < right.splitRank
	}
	if left.splitBin != right.splitBin {
		return left.splitBin < right.splitBin
	}
	if left.missingLeft != right.missingLeft {
		return left.missingLeft
	}
	return treeCategoryCodesKey(left.leftCategories) < treeCategoryCodesKey(right.leftCategories)
}

func treeCategoryCodesKey(values map[uint32]struct{}) string {
	if len(values) == 0 {
		return ""
	}
	codes := make([]int, 0, len(values))
	for value := range values {
		codes = append(codes, int(value))
	}
	sort.Ints(codes)
	return fmt.Sprint(codes)
}

func (g *treeGrowth) partition(rows []int, split *treeSplit) ([]int, []int) {
	left := make([]int, 0, len(rows)/2)
	right := make([]int, 0, len(rows)/2)
	feature := g.features[split.feature]
	for _, row := range rows {
		code := g.encoded[split.feature][row]
		goLeft := false
		if code == 0 {
			goLeft = split.missingLeft
		} else if feature.categorical {
			_, goLeft = split.leftCategories[code]
		} else {
			goLeft = int(code) <= split.splitBin
		}
		if goLeft {
			left = append(left, row)
		} else {
			right = append(right, row)
		}
	}
	return left, right
}

func (m *DecisionTreeClassifier) Classes() *insyra.DataList {
	if m == nil || m.classes == nil {
		return nil
	}
	return m.classes.Clone()
}

func (m *DecisionTreeClassifier) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	if m == nil || m.Root == nil {
		return nil, errors.New("ml: decision-tree classifier is nil")
	}
	leaves, err := m.predictLeaves(dt)
	if err != nil {
		return nil, err
	}
	values := make([]any, len(leaves))
	for i, leaf := range leaves {
		values[i] = leaf.Prediction
	}
	return insyra.NewDataList(values...), nil
}

func (m *DecisionTreeClassifier) PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error) {
	if m == nil || m.Root == nil {
		return nil, errors.New("ml: decision-tree classifier is nil")
	}
	leaves, err := m.predictLeaves(dt)
	if err != nil {
		return nil, err
	}
	byClass := make([][]float64, m.classes.Len())
	for class := range byClass {
		byClass[class] = make([]float64, len(leaves))
	}
	for row, leaf := range leaves {
		for class, probability := range leaf.Probabilities {
			byClass[class][row] = probability
		}
	}
	return probabilityTable(m.classes, byClass), nil
}

func (m *DecisionTreeClassifier) FeatureImportances() []float64 {
	if m == nil {
		return nil
	}
	return append([]float64(nil), m.featureImportances...)
}

// LeafValues reports the winning-class probability for each leaf in left to
// right tree order. Use the node Probabilities for the full class vector.
func (m *DecisionTreeClassifier) LeafValues() []float64 {
	if m == nil {
		return nil
	}
	return treeLeafValues(m.Root)
}

func (m *DecisionTreeClassifier) predictLeaves(dt *insyra.DataTable) ([]*DecisionTreeNode, error) {
	return predictTreeLeaves(dt, m.features, m.featureSchemas, m.Root)
}

func (m *DecisionTreeRegressor) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	if m == nil || m.Root == nil {
		return nil, errors.New("ml: decision-tree regressor is nil")
	}
	leaves, err := predictTreeLeaves(dt, m.features, m.featureSchemas, m.Root)
	if err != nil {
		return nil, err
	}
	values := make([]any, len(leaves))
	for i, leaf := range leaves {
		values[i] = leaf.Value
	}
	return insyra.NewDataList(values...), nil
}

func (m *DecisionTreeRegressor) FeatureImportances() []float64 {
	if m == nil {
		return nil
	}
	return append([]float64(nil), m.featureImportances...)
}

func (m *DecisionTreeRegressor) LeafValues() []float64 {
	if m == nil {
		return nil
	}
	return treeLeafValues(m.Root)
}

func predictTreeLeaves(dt *insyra.DataTable, features []string, schemas []treeFeature, root *DecisionTreeNode) ([]*DecisionTreeNode, error) {
	if err := requireTable(dt); err != nil {
		return nil, err
	}
	columns, err := orderedFeatures(dt, features)
	if err != nil {
		return nil, err
	}
	data := make([][]any, len(columns))
	for i, column := range columns {
		data[i] = column.Data()
	}
	leaves := make([]*DecisionTreeNode, dt.NumRows())
	for row := range leaves {
		node := root
		for !node.IsLeaf {
			feature := schemas[node.Feature]
			code, missing, err := encodeScoringValue(feature, data[node.Feature][row])
			if err != nil {
				return nil, fmt.Errorf("ml: score feature %q row %d: %w", feature.name, row, err)
			}
			goLeft := false
			if missing {
				goLeft = node.MissingGoLeft
			} else if feature.categorical && code == 0 {
				goLeft = node.UnseenCategoryGoLeft
			} else if feature.categorical {
				_, goLeft = node.leftCategories[code]
			} else {
				goLeft = int(code) <= node.splitBin
			}
			if goLeft {
				node = node.Left
			} else {
				node = node.Right
			}
		}
		leaves[row] = node
	}
	return leaves, nil
}

func encodeScoringValue(feature treeFeature, value any) (uint32, bool, error) {
	if treeMissing(value) {
		return 0, true, nil
	}
	if feature.categorical {
		return feature.categoryCode[treeCategoryKey(value)], false, nil
	}
	converted, ok := insyra.ToFloat64Safe(value)
	if !ok {
		return 0, false, errors.New("value is not numeric")
	}
	return numericBin(feature.edges, float32(converted)), false, nil
}

func treeLeafValues(root *DecisionTreeNode) []float64 {
	if root == nil {
		return nil
	}
	values := make([]float64, 0)
	var walk func(*DecisionTreeNode)
	walk = func(node *DecisionTreeNode) {
		if node.IsLeaf {
			values = append(values, node.Value)
			return
		}
		walk(node.Left)
		walk(node.Right)
	}
	walk(root)
	return values
}
