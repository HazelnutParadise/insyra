package ml_test

import "github.com/HazelnutParadise/insyra/ml"

// Features is a value method promoted from the embedded modelBase, so a model
// value, not only a pointer, satisfies an interface naming it. v0.3.4 code
// relying on that keeps compiling.
var (
	_ interface{ Features() []string } = ml.LinearModel{}
	_ interface{ Features() []string } = ml.PolynomialModel{}
	_ interface{ Features() []string } = ml.WeightedLinearModel{}
	_ interface{ Features() []string } = ml.RidgeModel{}
	_ interface{ Features() []string } = ml.LassoModel{}
	_ interface{ Features() []string } = ml.ExponentialModel{}
	_ interface{ Features() []string } = ml.LogarithmicModel{}
	_ interface{ Features() []string } = ml.LogisticModel{}
	_ interface{ Features() []string } = ml.PoissonModel{}
	_ interface{ Features() []string } = ml.GLMModel{}
	_ interface{ Features() []string } = ml.KMeansModel{}
	_ interface{ Features() []string } = ml.KNNClassifier{}
	_ interface{ Features() []string } = ml.KNNRegressor{}
	_ interface{ Features() []string } = ml.DecisionTreeClassifier{}
	_ interface{ Features() []string } = ml.DecisionTreeRegressor{}
	_ interface{ Features() []string } = ml.RandomForestClassifier{}
	_ interface{ Features() []string } = ml.RandomForestRegressor{}
	_ interface{ Features() []string } = ml.GradientBoostingClassifier{}
	_ interface{ Features() []string } = ml.GradientBoostingRegressor{}
)
