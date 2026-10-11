package ml

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sync"

	"github.com/HazelnutParadise/insyra"
)

// RandomForestOptions configures a forest. The zero value means scikit-learn's
// defaults: 100 trees, √p features per split for classification and all p for
// regression, unbounded depth.
type RandomForestOptions struct {
	// Trees is the number of trees. Default 100.
	Trees int
	// MaxFeatures is how many features each split may consider. Default 0
	// means scikit-learn's: √p rounded down (at least 1) for classification,
	// all p for regression. Values above p are capped at p.
	MaxFeatures int
	// Seed makes the bootstrap draws and feature subsampling reproducible.
	// Nil draws one seed for the whole forest and reports it on the model.
	Seed *int64
	// Tree is applied to every tree in the forest.
	Tree DecisionTreeOptions
}

// RandomForestClassifier is a fitted forest of classification trees. It
// predicts by averaging the trees' class probabilities and taking the largest
// — scikit-learn's rule, which uses more of what each tree knows than
// majority voting does.
type RandomForestClassifier struct {
	modelBase
	trees       []*treeFit
	classes     *insyra.DataList
	importances []float64
	// Seed reproduces this forest exactly: refitting with it in
	// RandomForestOptions.Seed gives identical trees.
	Seed int64
}

// RandomForestRegressor is a fitted forest of regression trees, predicting the
// mean of the trees' predictions.
type RandomForestRegressor struct {
	modelBase
	trees       []*treeFit
	importances []float64
	// Seed reproduces this forest exactly.
	Seed int64
}

// FitRandomForestClassifier fits a random forest for classification.
func FitRandomForestClassifier(x *insyra.DataTable, y *insyra.DataList, opts ...RandomForestOptions) (*RandomForestClassifier, error) {
	forest, seed, err := fitRandomForest(x, y, opts, true)
	if err != nil {
		return nil, err
	}
	return &RandomForestClassifier{
		modelBase:   modelBase{features: forest.preparation.features},
		trees:       forest.trees,
		classes:     insyra.NewDataList(forest.preparation.classes...).SetName("classes"),
		importances: forest.importances,
		Seed:        seed,
	}, nil
}

// FitRandomForestRegressor fits a random forest for regression.
func FitRandomForestRegressor(x *insyra.DataTable, y *insyra.DataList, opts ...RandomForestOptions) (*RandomForestRegressor, error) {
	forest, seed, err := fitRandomForest(x, y, opts, false)
	if err != nil {
		return nil, err
	}
	return &RandomForestRegressor{
		modelBase:   modelBase{features: forest.preparation.features},
		trees:       forest.trees,
		importances: forest.importances,
		Seed:        seed,
	}, nil
}

type fittedForest struct {
	preparation *treePreparation
	trees       []*treeFit
	importances []float64
}

func fitRandomForest(x *insyra.DataTable, y *insyra.DataList, opts []RandomForestOptions, classification bool) (*fittedForest, int64, error) {
	if len(opts) > 1 {
		return nil, 0, errors.New("ml: opts accepts at most one value")
	}
	options := RandomForestOptions{}
	if len(opts) == 1 {
		options = opts[0]
	}
	trees := options.Trees
	if trees == 0 {
		trees = 100
	}
	if trees < 0 {
		return nil, 0, fmt.Errorf("ml: a forest needs a positive number of trees, got %d", trees)
	}
	if options.MaxFeatures < 0 {
		return nil, 0, fmt.Errorf("ml: max features must not be negative, got %d", options.MaxFeatures)
	}

	// The same defaulting and bounds validation the single-tree entry points
	// apply — skipping it hands a zero MaxBins to the histogram builder.
	treeOptions, err := oneDecisionTreeOption([]DecisionTreeOptions{options.Tree})
	if err != nil {
		return nil, 0, err
	}
	options.Tree = treeOptions

	preparation, err := prepareDecisionTree(x, y, options.Tree, classification)
	if err != nil {
		return nil, 0, err
	}

	p := len(preparation.features)
	maxFeatures := options.MaxFeatures
	if maxFeatures == 0 {
		if classification {
			// scikit-learn's "sqrt", the decorrelation default.
			maxFeatures = int(math.Sqrt(float64(p)))
			if maxFeatures < 1 {
				maxFeatures = 1
			}
		} else {
			maxFeatures = p
		}
	}
	if maxFeatures > p {
		maxFeatures = p
	}

	seed := int64(0)
	if options.Seed != nil {
		seed = *options.Seed
	} else {
		seed = rand.Int63()
	}

	// Trees are independent, so they fit in parallel — but each owns an RNG
	// seeded from the forest seed and its own index, so the result is
	// identical however the goroutines interleave. Determinism must not
	// depend on scheduling.
	forest := &fittedForest{preparation: preparation, trees: make([]*treeFit, trees)}
	var wg sync.WaitGroup
	for t := 0; t < trees; t++ {
		wg.Add(1)
		go func(t int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(seed + int64(t)))
			rows := make([]int, preparation.n)
			for i := range rows {
				rows[i] = rng.Intn(preparation.n)
			}
			forest.trees[t] = preparation.grow(rows, options.Tree, maxFeatures, rng)
		}(t)
	}
	wg.Wait()

	// Each tree's importances are normalized to sum 1; the forest's are their
	// mean, renormalized — scikit-learn's aggregation.
	forest.importances = make([]float64, p)
	for _, tree := range forest.trees {
		for j, value := range tree.importances {
			forest.importances[j] += value
		}
	}
	total := 0.0
	for _, value := range forest.importances {
		total += value
	}
	if total != 0 {
		for j := range forest.importances {
			forest.importances[j] /= total
		}
	}
	return forest, seed, nil
}

// forestLeaves runs every tree over the same ordered table once.
func forestLeaves(dt *insyra.DataTable, features []string, trees []*treeFit) ([][]*DecisionTreeNode, error) {
	leaves := make([][]*DecisionTreeNode, len(trees))
	for t, tree := range trees {
		treeLeaves, err := predictTreeLeaves(dt, features, tree.schemas, tree.root)
		if err != nil {
			return nil, err
		}
		leaves[t] = treeLeaves
	}
	return leaves, nil
}

func (m *RandomForestClassifier) averagedProbabilities(dt *insyra.DataTable) ([][]float64, error) {
	if m == nil || len(m.trees) == 0 {
		return nil, errors.New("ml: random-forest classifier is nil")
	}
	leaves, err := forestLeaves(dt, m.features, m.trees)
	if err != nil {
		return nil, err
	}
	rows := len(leaves[0])
	classes := m.classes.Len()
	averaged := make([][]float64, rows)
	for row := range averaged {
		averaged[row] = make([]float64, classes)
		for _, treeLeaves := range leaves {
			for class, probability := range treeLeaves[row].Probabilities {
				averaged[row][class] += probability
			}
		}
		for class := range averaged[row] {
			averaged[row][class] /= float64(len(m.trees))
		}
	}
	return averaged, nil
}

func (m *RandomForestClassifier) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	averaged, err := m.averagedProbabilities(dt)
	if err != nil {
		return nil, err
	}
	values := make([]any, len(averaged))
	for row, probabilities := range averaged {
		best := 0
		for class, probability := range probabilities {
			if probability > probabilities[best] {
				best = class
			}
		}
		values[row] = m.classes.Get(best)
	}
	return insyra.NewDataList(values...), nil
}

func (m *RandomForestClassifier) PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error) {
	averaged, err := m.averagedProbabilities(dt)
	if err != nil {
		return nil, err
	}
	byClass := make([][]float64, m.classes.Len())
	for class := range byClass {
		byClass[class] = make([]float64, len(averaged))
		for row := range averaged {
			byClass[class][row] = averaged[row][class]
		}
	}
	return probabilityTable(m.classes, byClass), nil
}

func (m *RandomForestClassifier) Classes() *insyra.DataList {
	if m == nil || m.classes == nil {
		return nil
	}
	return m.classes.Clone()
}

func (m *RandomForestClassifier) FeatureImportances() []float64 {
	if m == nil {
		return nil
	}
	return append([]float64(nil), m.importances...)
}

func (m *RandomForestRegressor) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	if m == nil || len(m.trees) == 0 {
		return nil, errors.New("ml: random-forest regressor is nil")
	}
	leaves, err := forestLeaves(dt, m.features, m.trees)
	if err != nil {
		return nil, err
	}
	values := make([]any, len(leaves[0]))
	for row := range values {
		total := 0.0
		for _, treeLeaves := range leaves {
			total += treeLeaves[row].Value
		}
		values[row] = total / float64(len(m.trees))
	}
	return insyra.NewDataList(values...), nil
}

func (m *RandomForestRegressor) FeatureImportances() []float64 {
	if m == nil {
		return nil
	}
	return append([]float64(nil), m.importances...)
}

var (
	_ ProbaModel  = (*RandomForestClassifier)(nil)
	_ Importances = (*RandomForestClassifier)(nil)
	_ Model       = (*RandomForestRegressor)(nil)
	_ Importances = (*RandomForestRegressor)(nil)
)
