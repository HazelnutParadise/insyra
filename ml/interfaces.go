// Package ml is Insyra's classical machine-learning layer.
package ml

import (
	"io"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// Transformer is a fitted preprocessing step.
type Transformer interface {
	Transform(dt *insyra.DataTable) (*insyra.DataTable, error)
}

// Model is a fitted predictor.
type Model interface {
	Features() []string
	Predict(dt *insyra.DataTable) (*insyra.DataList, error)
}

// InverseTransformer is a fitted preprocessing step that can restore its input.
type InverseTransformer interface {
	InverseTransform(dt *insyra.DataTable) (*insyra.DataTable, error)
}

// ProbaModel is a model that reports class probabilities.
type ProbaModel interface {
	Model
	Classes() *insyra.DataList
	PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error)
}

// Importances is a model that reports one feature importance per feature.
type Importances interface {
	Model
	FeatureImportances() []float64
}

// Exporter writes a fitted model in a format another runtime can read.
type Exporter interface {
	Model
	ExportONNX(w io.Writer) error
}

// Step is an unfitted preprocessing stage.
type Step struct {
	Name string
	Fit  func(x *insyra.DataTable, y *insyra.DataList) (Transformer, error)
}

// Estimator is an unfitted model. Fit is a function so callers can close over
// configuration and refit it without reflection or cloning.
type Estimator struct {
	Name string
	Fit  func(x *insyra.DataTable, y *insyra.DataList) (Model, error)
}

// These aliases keep the options in ml identical to the options in stats.
type LogisticOptions = stats.LogisticRegressionOptions
type PoissonOptions = stats.PoissonRegressionOptions
type GLMOptions = stats.GLMOptions
type KMeansOptions = stats.KMeansOptions
type KNNOptions = stats.KNNOptions

var (
	_ Transformer = (*insyra.StandardScaler)(nil)
	_ Transformer = (*insyra.MinMaxScaler)(nil)
	_ Transformer = (*insyra.RobustScaler)(nil)
	_ Transformer = (*insyra.MaxAbsScaler)(nil)
	_ Transformer = (*insyra.OneHotEncoder)(nil)
	_ Transformer = (*insyra.LabelEncoder)(nil)
	_ Transformer = (*insyra.OrdinalEncoder)(nil)
)
