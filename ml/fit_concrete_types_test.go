package ml_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
	"github.com/HazelnutParadise/insyra/stats"
)

// #263 (ML-1): every Fit function returns the model's own type, so the
// wrapped stats result is read without a type assertion, and each type still
// satisfies the interface the function used to return.
func TestFitFunctionsReturnTheirOwnTypes(t *testing.T) {
	features := testFeatures()
	x, xOne := features.table, features.one
	y := dataList([]any{3.1, 5.4, 7.8, 10.2, 12.7, 15.1, 17.6, 20.0}, "y")
	labels := dataList([]any{0, 0, 1, 0, 1, 1, 0, 1}, "label")
	names := dataList([]any{"a", "a", "b", "a", "b", "b", "a", "b"}, "label")
	counts := dataList([]any{1, 2, 1, 3, 4, 6, 5, 8}, "count")
	weights := dataList([]any{1, 2, 1, 2, 1, 2, 1, 2}, "w")
	seed := int64(7)

	var fitted struct {
		linear      *ml.LinearModel
		polynomial  *ml.PolynomialModel
		weighted    *ml.WeightedLinearModel
		ridge       *ml.RidgeModel
		lasso       *ml.LassoModel
		exponential *ml.ExponentialModel
		logarithmic *ml.LogarithmicModel
		logistic    *ml.LogisticModel
		poisson     *ml.PoissonModel
		glm         *ml.GLMModel
		kmeans      *ml.KMeansModel
		pca         *ml.PCATransformer
		knnClass    *ml.KNNClassifier
		knnReg      *ml.KNNRegressor
	}
	var err error
	must := func(name string) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	fitted.linear, err = ml.FitLinearRegression(x, y)
	must("FitLinearRegression")
	fitted.polynomial, err = ml.FitPolynomialRegression(xOne, y, 2)
	must("FitPolynomialRegression")
	fitted.weighted, err = ml.FitWeightedLinearRegression(x, y, weights)
	must("FitWeightedLinearRegression")
	fitted.ridge, err = ml.FitRidgeRegression(x, y, 0.5)
	must("FitRidgeRegression")
	fitted.lasso, err = ml.FitLassoRegression(x, y, 0.1)
	must("FitLassoRegression")
	fitted.exponential, err = ml.FitExponentialRegression(xOne, y)
	must("FitExponentialRegression")
	fitted.logarithmic, err = ml.FitLogarithmicRegression(xOne, y)
	must("FitLogarithmicRegression")
	fitted.logistic, err = ml.FitLogisticRegression(x, labels)
	must("FitLogisticRegression")
	fitted.poisson, err = ml.FitPoissonRegression(x, counts)
	must("FitPoissonRegression")
	fitted.glm, err = ml.FitGLM(x, y, ml.GLMOptions{Family: stats.Gaussian, Link: stats.Identity})
	must("FitGLM")
	fitted.kmeans, err = ml.FitKMeans(x, 2, ml.KMeansOptions{NStart: 1, Seed: &seed})
	must("FitKMeans")
	fitted.pca, err = ml.FitPCA(x, 2)
	must("FitPCA")
	fitted.knnClass, err = ml.FitKNNClassifier(x, names, 3)
	must("FitKNNClassifier")
	fitted.knnReg, err = ml.FitKNNRegressor(x, y, 3)
	must("FitKNNRegressor")

	// The stats result of every wrapped model, read straight off the type.
	results := map[string]any{
		"linear":      fitted.linear.Result,
		"polynomial":  fitted.polynomial.Result,
		"weighted":    fitted.weighted.Result,
		"ridge":       fitted.ridge.Result,
		"lasso":       fitted.lasso.Result,
		"exponential": fitted.exponential.Result,
		"logarithmic": fitted.logarithmic.Result,
		"logistic":    fitted.logistic.Result,
		"poisson":     fitted.poisson.Result,
		"glm":         fitted.glm.Result,
		"kmeans":      fitted.kmeans.Result,
		"pca":         fitted.pca.Result,
		"knn class":   fitted.knnClass.Result,
		"knn reg":     fitted.knnReg.Result,
	}
	for name, result := range results {
		if value := reflect.ValueOf(result); !value.IsValid() || value.IsNil() {
			t.Errorf("%s: Result is nil after a successful fit", name)
		}
	}

	// Each type still goes wherever the interface it used to be returned as goes.
	models := []struct {
		model ml.Model
		input *insyra.DataTable
	}{
		{fitted.linear, x}, {fitted.polynomial, xOne}, {fitted.weighted, x},
		{fitted.ridge, x}, {fitted.lasso, x}, {fitted.exponential, xOne},
		{fitted.logarithmic, xOne}, {fitted.poisson, x}, {fitted.glm, x},
		{fitted.knnReg, x},
	}
	probaModels := []ml.ProbaModel{fitted.logistic, fitted.knnClass}
	clusterers := []ml.Clusterer{fitted.kmeans}
	transformers := []ml.Transformer{fitted.pca}
	if covered := len(models) + len(probaModels) + len(clusterers) + len(transformers); covered != len(results) {
		t.Fatalf("interface checks cover %d types, want %d", covered, len(results))
	}
	for _, tc := range models {
		if _, err := tc.model.Predict(tc.input); err != nil {
			t.Errorf("%T.Predict: %v", tc.model, err)
		}
	}
	for _, model := range probaModels {
		if _, err := model.PredictProba(x); err != nil {
			t.Errorf("%T.PredictProba: %v", model, err)
		}
	}
	if got := clusterers[0].Clusters(); got != 2 {
		t.Errorf("KMeansModel.Clusters() = %d, want 2", got)
	}
	if _, err := transformers[0].Transform(x); err != nil {
		t.Errorf("PCATransformer.Transform: %v", err)
	}
}

// A closure declared to return (ml.Model, error) turns a failed fit's nil
// pointer into a Model that is not nil. A caller that looks at the model
// instead of the error must get an error back, not a panic.
func TestAFailedFitReachedThroughAnInterfaceDoesNotPanic(t *testing.T) {
	fit := func(x *insyra.DataTable, y *insyra.DataList) (ml.Model, error) {
		return ml.FitLinearRegression(x, y)
	}
	model, err := fit(nil, dataList([]any{1.0, 2.0}, "y"))
	if err == nil {
		t.Fatal("fitting a nil table must fail")
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("a method on the failed fit panicked: %v", r)
		}
	}()
	if got := model.Features(); got != nil {
		t.Errorf("Features() = %v, want nil", got)
	}
	if _, err := model.Predict(testFeatures().table); err == nil {
		t.Error("Predict on a failed fit returned no error")
	}
}

// Every method of every type a wrapping Fit function returns, called on a
// nil pointer, answers with an error or an empty value instead of panicking.
func TestWrappedModelsAreSafeWhenNil(t *testing.T) {
	table := testFeatures().table
	var (
		linear      *ml.LinearModel
		polynomial  *ml.PolynomialModel
		weighted    *ml.WeightedLinearModel
		ridge       *ml.RidgeModel
		lasso       *ml.LassoModel
		exponential *ml.ExponentialModel
		logarithmic *ml.LogarithmicModel
		logistic    *ml.LogisticModel
		poisson     *ml.PoissonModel
		glm         *ml.GLMModel
		kmeans      *ml.KMeansModel
		pca         *ml.PCATransformer
		knnClass    *ml.KNNClassifier
		knnReg      *ml.KNNRegressor
	)
	predictFails := func(model ml.Model) func() bool {
		return func() bool {
			_, err := model.Predict(table)
			return err != nil
		}
	}
	exportFails := func(model ml.Exporter) func() bool {
		return func() bool {
			var buffer bytes.Buffer
			return model.ExportONNX(&buffer) != nil && buffer.Len() == 0
		}
	}
	noClasses := func(model ml.Classifier) func() bool {
		return func() bool {
			classes := model.Classes()
			return classes != nil && classes.Len() == 0 && classes.Err() != nil
		}
	}
	probaFails := func(model ml.ProbaModel) func() bool {
		return func() bool {
			_, err := model.PredictProba(table)
			return err != nil
		}
	}
	cases := []struct {
		name string
		ok   func() bool
	}{
		{"LinearModel.Features", func() bool { return linear.Features() == nil }},
		{"LinearModel.Predict", predictFails(linear)},
		{"LinearModel.ExportONNX", exportFails(linear)},
		{"PolynomialModel.Features", func() bool { return polynomial.Features() == nil }},
		{"PolynomialModel.Predict", predictFails(polynomial)},
		{"WeightedLinearModel.Features", func() bool { return weighted.Features() == nil }},
		{"WeightedLinearModel.Predict", predictFails(weighted)},
		{"WeightedLinearModel.ExportONNX", exportFails(weighted)},
		{"RidgeModel.Features", func() bool { return ridge.Features() == nil }},
		{"RidgeModel.Predict", predictFails(ridge)},
		{"RidgeModel.ExportONNX", exportFails(ridge)},
		{"LassoModel.Features", func() bool { return lasso.Features() == nil }},
		{"LassoModel.Predict", predictFails(lasso)},
		{"LassoModel.ExportONNX", exportFails(lasso)},
		{"ExponentialModel.Features", func() bool { return exponential.Features() == nil }},
		{"ExponentialModel.Predict", predictFails(exponential)},
		{"LogarithmicModel.Features", func() bool { return logarithmic.Features() == nil }},
		{"LogarithmicModel.Predict", predictFails(logarithmic)},
		{"LogisticModel.Features", func() bool { return logistic.Features() == nil }},
		{"LogisticModel.Predict", predictFails(logistic)},
		{"LogisticModel.Classes", noClasses(logistic)},
		{"LogisticModel.PredictProba", probaFails(logistic)},
		{"LogisticModel.ExportONNX", exportFails(logistic)},
		{"PoissonModel.Features", func() bool { return poisson.Features() == nil }},
		{"PoissonModel.Predict", predictFails(poisson)},
		{"GLMModel.Features", func() bool { return glm.Features() == nil }},
		{"GLMModel.Predict", predictFails(glm)},
		{"KMeansModel.Features", func() bool { return kmeans.Features() == nil }},
		{"KMeansModel.Predict", predictFails(kmeans)},
		{"KMeansModel.Clusters", func() bool { return kmeans.Clusters() == 0 }},
		{"PCATransformer.Features", func() bool { return pca.Features() == nil }},
		{"PCATransformer.Transform", func() bool {
			_, err := pca.Transform(table)
			return err != nil
		}},
		{"KNNClassifier.Features", func() bool { return knnClass.Features() == nil }},
		{"KNNClassifier.Predict", predictFails(knnClass)},
		{"KNNClassifier.Classes", noClasses(knnClass)},
		{"KNNClassifier.PredictProba", probaFails(knnClass)},
		{"KNNRegressor.Features", func() bool { return knnReg.Features() == nil }},
		{"KNNRegressor.Predict", predictFails(knnReg)},
		{"ExportONNX of a nil pointer", func() bool {
			var buffer bytes.Buffer
			return ml.ExportONNX(&buffer, linear) != nil && buffer.Len() == 0
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panicked on a nil model: %v", r)
				}
			}()
			if !tc.ok() {
				t.Fatal("a nil model must answer with an error or an empty value")
			}
		})
	}
}
