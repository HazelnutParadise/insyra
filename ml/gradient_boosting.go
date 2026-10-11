package ml

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// GradientBoostingOptions configures boosting. The zero value means
// scikit-learn's defaults: 100 stages, learning rate 0.1, trees of depth 3.
type GradientBoostingOptions struct {
	// Stages is the number of boosting rounds. Default 100.
	Stages int
	// LearningRate shrinks each stage's contribution. Default 0.1. Smaller
	// rates need more stages and generalise better; 0 is refused because it
	// would fit the base value forever.
	LearningRate float64
	// Tree is applied to every stage's tree. A zero MaxDepth becomes 3 —
	// boosting wants weak learners, and an unbounded tree would interpolate
	// the residuals in one stage and leave nothing to boost.
	Tree DecisionTreeOptions
}

// GradientBoostingRegressor is a fitted boosted ensemble for regression:
// squared loss, each stage a tree fitted to the current residuals.
type GradientBoostingRegressor struct {
	modelBase
	base         float64
	learningRate float64
	trees        []*treeFit
	importances  []float64
	// Stages is how many rounds actually ran — fewer than requested when the
	// residuals reached zero early and there was nothing left to fit.
	Stages int
}

// GradientBoostingClassifier is a fitted boosted ensemble for binary
// classification: logistic loss, trees fitted to the probability residuals
// with Newton-step leaf values.
//
// Only binary targets are supported in this version. Multiclass boosting
// fits one tree per class per stage and is a different amount of machinery;
// a target with more classes is refused rather than approximated.
type GradientBoostingClassifier struct {
	modelBase
	classes      *insyra.DataList
	base         float64 // prior log-odds of the second class
	learningRate float64
	trees        []*treeFit
	importances  []float64
	Stages       int
}

func oneGradientBoostingOption(opts []GradientBoostingOptions) (GradientBoostingOptions, error) {
	if len(opts) > 1 {
		return GradientBoostingOptions{}, errors.New("ml: opts accepts at most one value")
	}
	options := GradientBoostingOptions{}
	if len(opts) == 1 {
		options = opts[0]
	}
	if options.Stages == 0 {
		options.Stages = 100
	}
	if options.Stages < 0 {
		return GradientBoostingOptions{}, fmt.Errorf("ml: boosting needs a positive number of stages, got %d", options.Stages)
	}
	if options.LearningRate == 0 {
		options.LearningRate = 0.1
	}
	if options.LearningRate < 0 || math.IsNaN(options.LearningRate) || math.IsInf(options.LearningRate, 0) {
		return GradientBoostingOptions{}, fmt.Errorf("ml: learning rate must be positive and finite, got %v", options.LearningRate)
	}
	if options.Tree.MaxDepth == 0 {
		options.Tree.MaxDepth = 3
	}
	// The same defaulting the single-tree entry points apply — the forest
	// needed this too, and for the same reason: a zero MaxBins reaching the
	// histogram builder is a panic, not a default.
	tree, err := oneDecisionTreeOption([]DecisionTreeOptions{options.Tree})
	if err != nil {
		return GradientBoostingOptions{}, err
	}
	options.Tree = tree
	return options, nil
}

// FitGradientBoostingRegressor fits a boosted regression ensemble under
// squared loss. The negative gradient of squared loss is the residual, and a
// regression tree's leaf mean is already the optimal squared-loss leaf value,
// so each stage is exactly: fit a tree to what is left unexplained.
func FitGradientBoostingRegressor(x *insyra.DataTable, y *insyra.DataList, opts ...GradientBoostingOptions) (*GradientBoostingRegressor, error) {
	options, err := oneGradientBoostingOption(opts)
	if err != nil {
		return nil, err
	}
	features, _, err := fitFeatures(x)
	if err != nil {
		return nil, err
	}
	if y == nil || isNilPointer(y) {
		return nil, errors.New("ml: target list is nil")
	}
	n := x.NumRows()
	if y.Len() != n || n == 0 {
		return nil, fmt.Errorf("ml: target length %d does not match training row count %d", y.Len(), n)
	}
	targets, err := numericTargets(y)
	if err != nil {
		return nil, err
	}

	base := 0.0
	for _, value := range targets {
		base += value
	}
	base /= float64(n)

	current := make([]float64, n)
	for i := range current {
		current[i] = base
	}
	model := &GradientBoostingRegressor{
		modelBase:    modelBase{features: features},
		base:         base,
		learningRate: options.LearningRate,
	}
	residuals := make([]any, n)
	for stage := 0; stage < options.Stages; stage++ {
		flat := true
		for i := range targets {
			r := targets[i] - current[i]
			residuals[i] = r
			if math.Abs(r) > 1e-13 {
				flat = false
			}
		}
		// Nothing left to explain: fitting further stages would add trees
		// that predict zero everywhere and claim importances for it.
		if flat {
			break
		}
		tree, err := fitDecisionTree(x, insyra.NewDataList(residuals...), options.Tree, false)
		if err != nil {
			return nil, fmt.Errorf("ml: boosting stage %d: %w", stage+1, err)
		}
		leaves, err := predictTreeLeaves(x, features, tree.schemas, tree.root)
		if err != nil {
			return nil, fmt.Errorf("ml: boosting stage %d: %w", stage+1, err)
		}
		for i, leaf := range leaves {
			current[i] += options.LearningRate * leaf.Value
		}
		model.trees = append(model.trees, tree)
	}
	model.Stages = len(model.trees)
	model.importances = aggregateImportances(model.trees, len(features))
	return model, nil
}

// FitGradientBoostingClassifier fits a boosted binary classifier under
// logistic loss. Each stage fits a regression tree to the probability
// residuals y − p, then replaces every leaf's value with the Newton step
// Σ(y−p) / Σ p(1−p) over the training rows in that leaf — the standard
// second-order update that makes the additive log-odds model converge.
func FitGradientBoostingClassifier(x *insyra.DataTable, y *insyra.DataList, opts ...GradientBoostingOptions) (*GradientBoostingClassifier, error) {
	options, err := oneGradientBoostingOption(opts)
	if err != nil {
		return nil, err
	}
	features, _, err := fitFeatures(x)
	if err != nil {
		return nil, err
	}
	if y == nil || isNilPointer(y) {
		return nil, errors.New("ml: target list is nil")
	}
	n := x.NumRows()
	if y.Len() != n || n == 0 {
		return nil, fmt.Errorf("ml: target length %d does not match training row count %d", y.Len(), n)
	}
	classes, targetClasses, err := treeClasses(y.Data(), n)
	if err != nil {
		return nil, err
	}
	if len(classes) != 2 {
		return nil, fmt.Errorf("ml: gradient boosting supports binary classification only in this version; got %d classes (%v)", len(classes), classes)
	}

	// The second class in the tree ordering plays 1, matching how the other
	// binary models in this package orient their probability columns.
	labels := make([]float64, n)
	positives := 0
	for i, class := range targetClasses {
		labels[i] = float64(class)
		positives += class
	}
	if positives == 0 || positives == n {
		return nil, errors.New("ml: boosting needs both classes present in the target")
	}
	prior := float64(positives) / float64(n)
	base := math.Log(prior / (1 - prior))

	current := make([]float64, n)
	for i := range current {
		current[i] = base
	}
	model := &GradientBoostingClassifier{
		modelBase:    modelBase{features: features},
		classes:      insyra.NewDataList(classes...).SetName("classes"),
		base:         base,
		learningRate: options.LearningRate,
	}
	residuals := make([]any, n)
	for stage := 0; stage < options.Stages; stage++ {
		flat := true
		probabilities := make([]float64, n)
		for i := range labels {
			probabilities[i] = sigmoid(current[i])
			r := labels[i] - probabilities[i]
			residuals[i] = r
			if math.Abs(r) > 1e-13 {
				flat = false
			}
		}
		if flat {
			break
		}
		tree, err := fitDecisionTree(x, insyra.NewDataList(residuals...), options.Tree, false)
		if err != nil {
			return nil, fmt.Errorf("ml: boosting stage %d: %w", stage+1, err)
		}
		leaves, err := predictTreeLeaves(x, features, tree.schemas, tree.root)
		if err != nil {
			return nil, fmt.Errorf("ml: boosting stage %d: %w", stage+1, err)
		}
		// Newton per leaf: numerator Σ(y−p), denominator Σ p(1−p). The tree's
		// own leaf means minimise squared error on the residuals; these values
		// minimise the logistic loss instead, which is the loss being boosted.
		numerator := make(map[*DecisionTreeNode]float64)
		denominator := make(map[*DecisionTreeNode]float64)
		for i, leaf := range leaves {
			numerator[leaf] += labels[i] - probabilities[i]
			denominator[leaf] += probabilities[i] * (1 - probabilities[i])
		}
		for leaf, top := range numerator {
			if denominator[leaf] < 1e-12 {
				leaf.Value = 0
				continue
			}
			leaf.Value = top / denominator[leaf]
		}
		for i, leaf := range leaves {
			current[i] += options.LearningRate * leaf.Value
		}
		model.trees = append(model.trees, tree)
	}
	model.Stages = len(model.trees)
	model.importances = aggregateImportances(model.trees, len(features))
	return model, nil
}

func sigmoid(f float64) float64 {
	// Clamped so an extreme log-odds cannot overflow exp; at ±36 the result
	// is already indistinguishable from 0 or 1 in float64.
	if f > 36 {
		return 1
	}
	if f < -36 {
		return 0
	}
	return 1 / (1 + math.Exp(-f))
}

func aggregateImportances(trees []*treeFit, features int) []float64 {
	importances := make([]float64, features)
	for _, tree := range trees {
		for j, value := range tree.importances {
			importances[j] += value
		}
	}
	total := 0.0
	for _, value := range importances {
		total += value
	}
	if total != 0 {
		for j := range importances {
			importances[j] /= total
		}
	}
	return importances
}

func numericTargets(y *insyra.DataList) ([]float64, error) {
	raw := y.Data()
	out := make([]float64, len(raw))
	for i, value := range raw {
		converted, ok := insyra.ToFloat64Safe(value)
		if !ok || math.IsNaN(converted) || math.IsInf(converted, 0) {
			return nil, fmt.Errorf("ml: regression target must be finite numeric at row %d", i)
		}
		out[i] = converted
	}
	return out, nil
}

func (m *GradientBoostingRegressor) score(dt *insyra.DataTable) ([]float64, error) {
	if m == nil || len(m.trees) == 0 {
		return nil, errors.New("ml: gradient-boosting regressor is nil")
	}
	leaves, err := forestLeaves(dt, m.features, m.trees)
	if err != nil {
		return nil, err
	}
	scores := make([]float64, len(leaves[0]))
	for row := range scores {
		scores[row] = m.base
		for _, treeLeaves := range leaves {
			scores[row] += m.learningRate * treeLeaves[row].Value
		}
	}
	return scores, nil
}

func (m *GradientBoostingRegressor) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	scores, err := m.score(dt)
	if err != nil {
		return nil, err
	}
	values := make([]any, len(scores))
	for i, score := range scores {
		values[i] = score
	}
	return insyra.NewDataList(values...), nil
}

func (m *GradientBoostingRegressor) FeatureImportances() []float64 {
	if m == nil {
		return nil
	}
	return append([]float64(nil), m.importances...)
}

func (m *GradientBoostingClassifier) logOdds(dt *insyra.DataTable) ([]float64, error) {
	if m == nil || len(m.trees) == 0 {
		return nil, errors.New("ml: gradient-boosting classifier is nil")
	}
	leaves, err := forestLeaves(dt, m.features, m.trees)
	if err != nil {
		return nil, err
	}
	scores := make([]float64, len(leaves[0]))
	for row := range scores {
		scores[row] = m.base
		for _, treeLeaves := range leaves {
			scores[row] += m.learningRate * treeLeaves[row].Value
		}
	}
	return scores, nil
}

func (m *GradientBoostingClassifier) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	scores, err := m.logOdds(dt)
	if err != nil {
		return nil, err
	}
	values := make([]any, len(scores))
	for i, score := range scores {
		if sigmoid(score) >= 0.5 {
			values[i] = m.classes.Get(1)
		} else {
			values[i] = m.classes.Get(0)
		}
	}
	return insyra.NewDataList(values...), nil
}

func (m *GradientBoostingClassifier) PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error) {
	scores, err := m.logOdds(dt)
	if err != nil {
		return nil, err
	}
	positive := make([]float64, len(scores))
	negative := make([]float64, len(scores))
	for i, score := range scores {
		positive[i] = sigmoid(score)
		negative[i] = 1 - positive[i]
	}
	return probabilityTable(m.classes, [][]float64{negative, positive}), nil
}

func (m *GradientBoostingClassifier) Classes() *insyra.DataList {
	if m == nil || m.classes == nil {
		return nil
	}
	return m.classes.Clone()
}

func (m *GradientBoostingClassifier) FeatureImportances() []float64 {
	if m == nil {
		return nil
	}
	return append([]float64(nil), m.importances...)
}

var (
	_ Model       = (*GradientBoostingRegressor)(nil)
	_ Importances = (*GradientBoostingRegressor)(nil)
	_ ProbaModel  = (*GradientBoostingClassifier)(nil)
	_ Importances = (*GradientBoostingClassifier)(nil)
)
