package ml

import (
	"errors"
	"fmt"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// LinearModel wraps stats.LinearRegressionResult.
type LinearModel struct {
	Result *stats.LinearRegressionResult
	modelBase
}

// PolynomialModel wraps stats.PolynomialRegressionResult.
type PolynomialModel struct {
	Result *stats.PolynomialRegressionResult
	modelBase
}

// ExponentialModel wraps stats.ExponentialRegressionResult.
type ExponentialModel struct {
	Result *stats.ExponentialRegressionResult
	modelBase
}

// LogarithmicModel wraps stats.LogarithmicRegressionResult.
type LogarithmicModel struct {
	Result *stats.LogarithmicRegressionResult
	modelBase
}

// LogisticModel wraps stats.LogisticRegressionResult.
type LogisticModel struct {
	Result *stats.LogisticRegressionResult
	modelBase
}

// PoissonModel wraps stats.PoissonRegressionResult.
type PoissonModel struct {
	Result *stats.PoissonRegressionResult
	modelBase
}

// GLMModel wraps stats.GLMResult.
type GLMModel struct {
	Result *stats.GLMResult
	modelBase
}

// KMeansModel wraps stats.KMeansResult.
type KMeansModel struct {
	Result *stats.KMeansResult
	modelBase
}

// PCATransformer wraps stats.PCAResult and projects new observations with the
// fitted centering, scaling, and component loadings.
type PCATransformer struct {
	Result *stats.PCAResult
	modelBase
}

// KNNClassifier keeps the training data because stats.KNNClassify accepts both
// training and test data in one call. Result is the fit-time result on the
// training data and remains reachable after later predictions.
type KNNClassifier struct {
	Result *stats.KNNClassificationResult
	modelBase
	train     *insyra.DataTable
	labels    *insyra.DataList
	k         int
	options   KNNOptions
	hasOption bool
}

// KNNRegressor keeps the training data because stats.KNNRegress accepts both
// training and test data in one call. Result is the fit-time result on the
// training data and remains reachable after later predictions.
type KNNRegressor struct {
	Result *stats.KNNRegressionResult
	modelBase
	train     *insyra.DataTable
	targets   *insyra.DataList
	k         int
	options   KNNOptions
	hasOption bool
}

func FitLinearRegression(x *insyra.DataTable, y *insyra.DataList) (Model, error) {
	features, xs, err := fitFeatures(x)
	if err != nil {
		return nil, err
	}
	result, err := stats.LinearRegression(y, xs...)
	if err != nil {
		return nil, err
	}
	return &LinearModel{Result: result, modelBase: modelBase{features: features}}, nil
}

func FitPolynomialRegression(x *insyra.DataTable, y *insyra.DataList, degree int) (Model, error) {
	features, xs, err := fitFeatures(x)
	if err != nil {
		return nil, err
	}
	if len(xs) != 1 {
		return nil, fmt.Errorf("ml: polynomial regression requires exactly one feature, got %d", len(xs))
	}
	result, err := stats.PolynomialRegression(y, xs[0], degree)
	if err != nil {
		return nil, err
	}
	return &PolynomialModel{Result: result, modelBase: modelBase{features: features}}, nil
}

func FitExponentialRegression(x *insyra.DataTable, y *insyra.DataList) (Model, error) {
	features, xs, err := fitFeatures(x)
	if err != nil {
		return nil, err
	}
	if len(xs) != 1 {
		return nil, fmt.Errorf("ml: exponential regression requires exactly one feature, got %d", len(xs))
	}
	result, err := stats.ExponentialRegression(y, xs[0])
	if err != nil {
		return nil, err
	}
	return &ExponentialModel{Result: result, modelBase: modelBase{features: features}}, nil
}

func FitLogarithmicRegression(x *insyra.DataTable, y *insyra.DataList) (Model, error) {
	features, xs, err := fitFeatures(x)
	if err != nil {
		return nil, err
	}
	if len(xs) != 1 {
		return nil, fmt.Errorf("ml: logarithmic regression requires exactly one feature, got %d", len(xs))
	}
	result, err := stats.LogarithmicRegression(y, xs[0])
	if err != nil {
		return nil, err
	}
	return &LogarithmicModel{Result: result, modelBase: modelBase{features: features}}, nil
}

func FitLogisticRegression(x *insyra.DataTable, y *insyra.DataList, opts ...LogisticOptions) (ProbaModel, error) {
	features, xs, err := fitFeatures(x)
	if err != nil {
		return nil, err
	}
	option, hasOption, err := oneLogisticOption(opts)
	if err != nil {
		return nil, err
	}
	var result *stats.LogisticRegressionResult
	if hasOption {
		result, err = stats.LogisticRegressionWithOptions(option, y, xs...)
	} else {
		result, err = stats.LogisticRegression(y, xs...)
	}
	if err != nil {
		return nil, err
	}
	return &LogisticModel{Result: result, modelBase: modelBase{features: features}}, nil
}

func FitPoissonRegression(x *insyra.DataTable, y *insyra.DataList, opts ...PoissonOptions) (Model, error) {
	features, xs, err := fitFeatures(x)
	if err != nil {
		return nil, err
	}
	option, hasOption, err := onePoissonOption(opts)
	if err != nil {
		return nil, err
	}
	if hasOption && option.Offset != nil {
		return nil, errors.New("ml: poisson regression offsets are not supported by Model.Predict")
	}
	var result *stats.PoissonRegressionResult
	if hasOption {
		result, err = stats.PoissonRegressionWithOptions(option, y, xs...)
	} else {
		result, err = stats.PoissonRegression(y, xs...)
	}
	if err != nil {
		return nil, err
	}
	return &PoissonModel{Result: result, modelBase: modelBase{features: features}}, nil
}

func FitGLM(x *insyra.DataTable, y *insyra.DataList, opts GLMOptions) (Model, error) {
	features, xs, err := fitFeatures(x)
	if err != nil {
		return nil, err
	}
	if opts.Offset != nil {
		return nil, errors.New("ml: GLM offsets are not supported by Model.Predict")
	}
	result, err := stats.GLM(opts, y, xs...)
	if err != nil {
		return nil, err
	}
	return &GLMModel{Result: result, modelBase: modelBase{features: features}}, nil
}

func FitKMeans(x *insyra.DataTable, k int, opts ...KMeansOptions) (Model, error) {
	features, _, err := fitFeatures(x)
	if err != nil {
		return nil, err
	}
	option, hasOption, err := oneKMeansOption(opts)
	if err != nil {
		return nil, err
	}
	var result *stats.KMeansResult
	if hasOption {
		result, err = stats.KMeans(x, k, option)
	} else {
		result, err = stats.KMeans(x, k)
	}
	if err != nil {
		return nil, err
	}
	return &KMeansModel{Result: result, modelBase: modelBase{features: features}}, nil
}

func FitPCA(x *insyra.DataTable, components int) (Transformer, error) {
	features, _, err := fitFeatures(x)
	if err != nil {
		return nil, err
	}
	result, err := stats.PCA(x, components)
	if err != nil {
		return nil, err
	}
	return &PCATransformer{Result: result, modelBase: modelBase{features: features}}, nil
}

func FitKNNClassifier(x *insyra.DataTable, y *insyra.DataList, k int, opts ...KNNOptions) (ProbaModel, error) {
	features, _, err := fitFeatures(x)
	if err != nil {
		return nil, err
	}
	option, hasOption, err := oneKNNOption(opts)
	if err != nil {
		return nil, err
	}
	var result *stats.KNNClassificationResult
	if hasOption {
		result, err = stats.KNNClassify(x, y, x, k, option)
	} else {
		result, err = stats.KNNClassify(x, y, x, k)
	}
	if err != nil {
		return nil, err
	}
	return &KNNClassifier{
		Result:    result,
		modelBase: modelBase{features: features},
		train:     x.Clone(), labels: y.Clone(), k: k,
		options: option, hasOption: hasOption,
	}, nil
}

func FitKNNRegressor(x *insyra.DataTable, y *insyra.DataList, k int, opts ...KNNOptions) (Model, error) {
	features, _, err := fitFeatures(x)
	if err != nil {
		return nil, err
	}
	option, hasOption, err := oneKNNOption(opts)
	if err != nil {
		return nil, err
	}
	var result *stats.KNNRegressionResult
	if hasOption {
		result, err = stats.KNNRegress(x, y, x, k, option)
	} else {
		result, err = stats.KNNRegress(x, y, x, k)
	}
	if err != nil {
		return nil, err
	}
	return &KNNRegressor{
		Result:    result,
		modelBase: modelBase{features: features},
		train:     x.Clone(), targets: y.Clone(), k: k,
		options: option, hasOption: hasOption,
	}, nil
}

func (m *LinearModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	return predictRegression(dt, m.features, func(xs []insyra.IDataList) (*insyra.DataList, error) {
		return m.Result.Predict(stats.PredictResponse, xs...)
	})
}

func (m *PolynomialModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	return predictRegression(dt, m.features, func(xs []insyra.IDataList) (*insyra.DataList, error) {
		return m.Result.Predict(stats.PredictResponse, xs...)
	})
}

func (m *ExponentialModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	return predictRegression(dt, m.features, func(xs []insyra.IDataList) (*insyra.DataList, error) {
		return m.Result.Predict(stats.PredictResponse, xs...)
	})
}

func (m *LogarithmicModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	return predictRegression(dt, m.features, func(xs []insyra.IDataList) (*insyra.DataList, error) {
		return m.Result.Predict(stats.PredictResponse, xs...)
	})
}

func (m *LogisticModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	return predictRegression(dt, m.features, func(xs []insyra.IDataList) (*insyra.DataList, error) {
		return m.Result.Predict(stats.PredictClass, xs...)
	})
}

func (m *LogisticModel) Classes() *insyra.DataList {
	if m == nil || m.Result == nil {
		return nil
	}
	return insyra.NewDataList(m.Result.ClassLabels...).SetName("classes")
}

func (m *LogisticModel) PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error) {
	if m == nil || m.Result == nil {
		return nil, errors.New("ml: logistic model is nil")
	}
	probability, err := predictRegression(dt, m.features, func(xs []insyra.IDataList) (*insyra.DataList, error) {
		return m.Result.Predict(stats.PredictResponse, xs...)
	})
	if err != nil {
		return nil, err
	}
	positive := probability.ToF64Slice()
	negative := make([]float64, len(positive))
	for i, value := range positive {
		negative[i] = 1 - value
	}
	classes := m.Classes()
	return probabilityTable(classes, [][]float64{negative, positive}), nil
}

func (m *PoissonModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	return predictRegression(dt, m.features, func(xs []insyra.IDataList) (*insyra.DataList, error) {
		return m.Result.Predict(stats.PredictResponse, xs...)
	})
}

func (m *GLMModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	return predictRegression(dt, m.features, func(xs []insyra.IDataList) (*insyra.DataList, error) {
		return m.Result.Predict(stats.PredictResponse, xs...)
	})
}

// Clusters reports how many groups the fit converged on, and marks this model
// as a Clusterer so that a regression metric will not score its cluster ids.
func (m *KMeansModel) Clusters() int {
	if m == nil || m.Result == nil || m.Result.Centers == nil {
		return 0
	}
	rows, _ := m.Result.Centers.Size()
	return rows
}

func (m *KMeansModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	if m == nil || m.Result == nil {
		return nil, errors.New("ml: kmeans model is nil")
	}
	ordered, err := orderedTable(dt, m.features)
	if err != nil {
		return nil, err
	}
	assignments, _, err := m.Result.Assign(ordered)
	if err != nil {
		return nil, err
	}
	values := make([]any, len(assignments))
	for i, assignment := range assignments {
		values[i] = assignment
	}
	return insyra.NewDataList(values...), nil
}

func (m *KNNClassifier) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	result, err := m.predict(dt)
	if err != nil {
		return nil, err
	}
	return asDataList(result.Predictions), nil
}

func (m *KNNClassifier) Classes() *insyra.DataList {
	if m == nil || m.Result == nil {
		return nil
	}
	return asDataList(m.Result.Classes)
}

func (m *KNNClassifier) PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error) {
	result, err := m.predict(dt)
	if err != nil {
		return nil, err
	}
	return asDataTable(result.Probabilities), nil
}

func (m *KNNClassifier) predict(dt *insyra.DataTable) (*stats.KNNClassificationResult, error) {
	if m == nil || m.train == nil || m.labels == nil {
		return nil, errors.New("ml: KNN classifier is nil")
	}
	ordered, err := orderedTable(dt, m.features)
	if err != nil {
		return nil, err
	}
	if m.hasOption {
		return stats.KNNClassify(m.train, m.labels, ordered, m.k, m.options)
	}
	return stats.KNNClassify(m.train, m.labels, ordered, m.k)
}

func (m *KNNRegressor) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	if m == nil || m.train == nil || m.targets == nil {
		return nil, errors.New("ml: KNN regressor is nil")
	}
	ordered, err := orderedTable(dt, m.features)
	if err != nil {
		return nil, err
	}
	var result *stats.KNNRegressionResult
	if m.hasOption {
		result, err = stats.KNNRegress(m.train, m.targets, ordered, m.k, m.options)
	} else {
		result, err = stats.KNNRegress(m.train, m.targets, ordered, m.k)
	}
	if err != nil {
		return nil, err
	}
	values := make([]any, len(result.Predictions))
	for i, value := range result.Predictions {
		values[i] = value
	}
	return insyra.NewDataList(values...), nil
}

func (p *PCATransformer) Features() []string {
	if p == nil {
		return nil
	}
	return p.modelBase.Features()
}

func (p *PCATransformer) Transform(dt *insyra.DataTable) (*insyra.DataTable, error) {
	if p == nil || p.Result == nil {
		return nil, errors.New("ml: PCA transformer is nil")
	}
	ordered, err := orderedTable(dt, p.features)
	if err != nil {
		return nil, err
	}
	values, err := numericColumns(ordered)
	if err != nil {
		return nil, err
	}
	if len(p.Result.Center) != len(values) || len(p.Result.Scale) != len(values) {
		return nil, errors.New("ml: PCA fitted parameters do not match fitted features")
	}
	componentNames := p.Result.Components.ColNames()
	componentValues := make([][]float64, len(componentNames))
	for component := range componentNames {
		loadings := p.Result.Components.GetColByNumber(component).ToF64Slice()
		if len(loadings) != len(values) {
			return nil, fmt.Errorf("ml: PCA component %d has %d loadings, want %d", component+1, len(loadings), len(values))
		}
		componentValues[component] = make([]float64, len(values[0]))
		for row := range componentValues[component] {
			for feature := range values {
				componentValues[component][row] += (values[feature][row] - p.Result.Center[feature]) / p.Result.Scale[feature] * loadings[feature]
			}
		}
	}
	return probabilityTable(nil, componentValues).SetColNames(componentNames), nil
}

func predictRegression(dt *insyra.DataTable, features []string, predict func([]insyra.IDataList) (*insyra.DataList, error)) (*insyra.DataList, error) {
	if err := requireTable(dt); err != nil {
		return nil, err
	}
	xs, err := orderedFeatures(dt, features)
	if err != nil {
		return nil, err
	}
	return predict(xs)
}
