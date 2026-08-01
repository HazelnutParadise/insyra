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

// Classifier is a fitted model that predicts one of a known set of classes.
// Models that also expose probabilities implement ProbaModel.
type Classifier interface {
	Model
	Classes() *insyra.DataList
}

// InverseTransformer is a fitted preprocessing step that can restore its input.
type InverseTransformer interface {
	InverseTransform(dt *insyra.DataTable) (*insyra.DataTable, error)
}

// ProbaModel is a model that reports class probabilities.
type ProbaModel interface {
	Classifier
	PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error)
}

// Importances is a model that reports one feature importance per feature.
//
// "Per feature" means per column the model was fitted on. For a plain model
// that is Features(); for a model whose preprocessing changes the column count
// it is TransformedFeatureNames(), which is why that capability exists.
type Importances interface {
	Model
	FeatureImportances() []float64
}

// TransformedFeatures is a model that was fitted on columns other than the
// ones it is called with — a pipeline whose preprocessing adds, removes or
// replaces columns.
//
// Without it, a pipeline that one-hot encodes a categorical column reports two
// feature names and four importances, and a caller reading them together
// attributes each number to the wrong column with no signal that anything is
// wrong. scikit-learn answers the same question with get_feature_names_out.
type TransformedFeatures interface {
	Model
	// TransformedFeatureNames returns the columns the model was fitted on
	// after every preprocessing step, in the order it saw them.
	TransformedFeatureNames() []string
}

// Clusterer is a model whose predictions are group assignments rather than
// measurements or classes from a known set.
//
// The distinction matters because Predict returns a DataList either way, and
// nothing in its type says whether a number in it is a predicted quantity or a
// group label. Without the declaration a regression metric will happily compute
// an RMSE over cluster ids — arithmetically correct and meaningless. A model
// implementing this is refused by regression metrics, the way a Classifier
// already is.
//
// A clustering model does not implement Classifier: its groups are discovered,
// not drawn from a set the caller supplied, so there is no Classes() to report.
type Clusterer interface {
	Model
	// Clusters reports how many groups the fit converged on.
	Clusters() int
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
