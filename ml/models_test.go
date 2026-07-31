package ml_test

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
	"github.com/HazelnutParadise/insyra/ml/mltest"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestConformanceEveryWrappedModel(t *testing.T) {
	features := testFeatures()
	x := features.table
	y := dataList([]any{3.1, 5.4, 7.8, 10.2, 12.7, 15.1, 17.6, 20.0}, "y")
	xOne := features.one
	seed := int64(7)

	models := []struct {
		name string
		fit  func() (ml.Model, error)
		x    *insyra.DataTable
	}{
		{"linear", func() (ml.Model, error) { return ml.FitLinearRegression(x, y) }, x},
		{"polynomial", func() (ml.Model, error) { return ml.FitPolynomialRegression(xOne, y, 2) }, xOne},
		{"exponential", func() (ml.Model, error) { return ml.FitExponentialRegression(xOne, y) }, xOne},
		{"logarithmic", func() (ml.Model, error) { return ml.FitLogarithmicRegression(xOne, y) }, xOne},
		{"logistic", func() (ml.Model, error) {
			return ml.FitLogisticRegression(x, dataList([]any{0, 0, 1, 0, 1, 1, 0, 1}, "label"))
		}, x},
		{"poisson", func() (ml.Model, error) {
			return ml.FitPoissonRegression(x, dataList([]any{1, 2, 1, 3, 4, 6, 5, 8}, "count"))
		}, x},
		{"glm", func() (ml.Model, error) {
			return ml.FitGLM(x, y, ml.GLMOptions{Family: stats.Gaussian, Link: stats.Identity})
		}, x},
		{"kmeans", func() (ml.Model, error) {
			return ml.FitKMeans(x, 2, ml.KMeansOptions{NStart: 1, Seed: &seed})
		}, x},
		{"knn classifier", func() (ml.Model, error) {
			return ml.FitKNNClassifier(x, dataList([]any{"a", "a", "b", "a", "b", "b", "a", "b"}, "label"), 3)
		}, x},
		{"knn regressor", func() (ml.Model, error) {
			return ml.FitKNNRegressor(x, y, 3)
		}, x},
	}

	for _, tc := range models {
		t.Run(tc.name, func(t *testing.T) {
			model, err := tc.fit()
			if err != nil {
				t.Fatalf("fit: %v", err)
			}
			mltest.RunConformance(t, model, tc.x, nil)
		})
	}
}

func TestPCAUsesFittedProjectionAndColumnNames(t *testing.T) {
	features := testFeatures()
	fitted, err := ml.FitPCA(features.table, 2)
	if err != nil {
		t.Fatalf("fit PCA: %v", err)
	}
	transformer := fitted.(*ml.PCATransformer)
	got, err := transformer.Transform(features.table)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if !reflect.DeepEqual(got.To2DSlice(), transformer.Result.Scores.To2DSlice()) {
		t.Fatalf("training transform differs from fitted scores: got=%v want=%v", got.To2DSlice(), transformer.Result.Scores.To2DSlice())
	}
	if !reflect.DeepEqual(got.ColNames(), []string{"PC1", "PC2"}) {
		t.Fatalf("component names = %v", got.ColNames())
	}

	reordered := insyra.NewDataTable(
		features.table.GetColByName("x2").Clone().SetName("x2"),
		features.table.GetColByName("x1").Clone().SetName("x1"),
	)
	reorderedGot, err := transformer.Transform(reordered)
	if err != nil {
		t.Fatalf("transform reordered: %v", err)
	}
	if !reflect.DeepEqual(got.To2DSlice(), reorderedGot.To2DSlice()) {
		t.Fatalf("reordering columns changed PCA scores")
	}

	extra := features.table.Clone().AppendCols(dataList([]any{9, 9, 9, 9, 9, 9, 9, 9}, "extra"))
	if _, err := transformer.Transform(extra); err != nil {
		t.Fatalf("extra column should be ignored: %v", err)
	}
	missing := insyra.NewDataTable(features.table.GetColByName("x2").Clone().SetName("x2"))
	if _, err := transformer.Transform(missing); err == nil || !strings.Contains(err.Error(), "x1") {
		t.Fatalf("missing feature error = %v", err)
	}
}

func TestWrappersReturnStatsResultsAndPredictionsExactly(t *testing.T) {
	features := testFeatures()
	y := dataList([]any{3.1, 5.4, 7.8, 10.2, 12.7, 15.1, 17.6, 20.0}, "y")

	linear, err := ml.FitLinearRegression(features.table, y)
	if err != nil {
		t.Fatal(err)
	}
	directLinear, err := stats.LinearRegression(y, features.table.GetColByNumber(0), features.table.GetColByNumber(1))
	if err != nil {
		t.Fatal(err)
	}
	linearModel := linear.(*ml.LinearModel)
	if !bitwiseEqual(linearModel.Result, directLinear) {
		t.Fatal("linear result changed by wrapper")
	}
	directPrediction, directErr := directLinear.Predict(stats.PredictResponse, features.table.GetColByNumber(0), features.table.GetColByNumber(1))
	assertPredictionMatches(t, linearModel, features.table, directPrediction, directErr)

	one := features.one
	polynomial, err := ml.FitPolynomialRegression(one, y, 2)
	if err != nil {
		t.Fatal(err)
	}
	directPolynomial, err := stats.PolynomialRegression(y, one.GetColByNumber(0), 2)
	if err != nil {
		t.Fatal(err)
	}
	polynomialModel := polynomial.(*ml.PolynomialModel)
	if !bitwiseEqual(polynomialModel.Result, directPolynomial) {
		t.Fatal("polynomial result changed by wrapper")
	}
	directPrediction, directErr = directPolynomial.Predict(stats.PredictResponse, one.GetColByNumber(0))
	assertPredictionMatches(t, polynomialModel, one, directPrediction, directErr)

	exponential, err := ml.FitExponentialRegression(one, y)
	if err != nil {
		t.Fatal(err)
	}
	directExponential, err := stats.ExponentialRegression(y, one.GetColByNumber(0))
	if err != nil {
		t.Fatal(err)
	}
	exponentialModel := exponential.(*ml.ExponentialModel)
	if !bitwiseEqual(exponentialModel.Result, directExponential) {
		t.Fatal("exponential result changed by wrapper")
	}
	directPrediction, directErr = directExponential.Predict(stats.PredictResponse, one.GetColByNumber(0))
	assertPredictionMatches(t, exponentialModel, one, directPrediction, directErr)

	logarithmic, err := ml.FitLogarithmicRegression(one, y)
	if err != nil {
		t.Fatal(err)
	}
	directLogarithmic, err := stats.LogarithmicRegression(y, one.GetColByNumber(0))
	if err != nil {
		t.Fatal(err)
	}
	logarithmicModel := logarithmic.(*ml.LogarithmicModel)
	if !bitwiseEqual(logarithmicModel.Result, directLogarithmic) {
		t.Fatal("logarithmic result changed by wrapper")
	}
	directPrediction, directErr = directLogarithmic.Predict(stats.PredictResponse, one.GetColByNumber(0))
	assertPredictionMatches(t, logarithmicModel, one, directPrediction, directErr)

	classification := dataList([]any{0, 0, 1, 0, 1, 1, 0, 1}, "label")
	logistic, err := ml.FitLogisticRegression(features.table, classification)
	if err != nil {
		t.Fatal(err)
	}
	directLogistic, err := stats.LogisticRegression(classification, features.table.GetColByNumber(0), features.table.GetColByNumber(1))
	if err != nil {
		t.Fatal(err)
	}
	logisticModel := logistic.(*ml.LogisticModel)
	if !bitwiseEqual(logisticModel.Result, directLogistic) {
		t.Fatal("logistic result changed by wrapper")
	}
	directPrediction, directErr = directLogistic.Predict(stats.PredictResponse, features.table.GetColByNumber(0), features.table.GetColByNumber(1))
	assertPredictionMatches(t, logisticModel, features.table, directPrediction, directErr)

	count := dataList([]any{1, 2, 1, 3, 4, 6, 5, 8}, "count")
	poisson, err := ml.FitPoissonRegression(features.table, count)
	if err != nil {
		t.Fatal(err)
	}
	directPoisson, err := stats.PoissonRegression(count, features.table.GetColByNumber(0), features.table.GetColByNumber(1))
	if err != nil {
		t.Fatal(err)
	}
	poissonModel := poisson.(*ml.PoissonModel)
	if !bitwiseEqual(poissonModel.Result, directPoisson) {
		t.Fatal("poisson result changed by wrapper")
	}
	directPrediction, directErr = directPoisson.Predict(stats.PredictResponse, features.table.GetColByNumber(0), features.table.GetColByNumber(1))
	assertPredictionMatches(t, poissonModel, features.table, directPrediction, directErr)

	glm, err := ml.FitGLM(features.table, y, ml.GLMOptions{Family: stats.Gaussian, Link: stats.Identity})
	if err != nil {
		t.Fatal(err)
	}
	directGLM, err := stats.GLM(stats.GLMOptions{Family: stats.Gaussian, Link: stats.Identity}, y, features.table.GetColByNumber(0), features.table.GetColByNumber(1))
	if err != nil {
		t.Fatal(err)
	}
	glmModel := glm.(*ml.GLMModel)
	if !bitwiseEqual(glmModel.Result, directGLM) {
		t.Fatal("GLM result changed by wrapper")
	}
	directPrediction, directErr = directGLM.Predict(stats.PredictResponse, features.table.GetColByNumber(0), features.table.GetColByNumber(1))
	assertPredictionMatches(t, glmModel, features.table, directPrediction, directErr)

	seed := int64(7)
	kmeans, err := ml.FitKMeans(features.table, 2, ml.KMeansOptions{NStart: 1, Seed: &seed})
	if err != nil {
		t.Fatal(err)
	}
	directKMeans, err := stats.KMeans(features.table, 2, stats.KMeansOptions{NStart: 1, Seed: &seed})
	if err != nil {
		t.Fatal(err)
	}
	kmeansModel := kmeans.(*ml.KMeansModel)
	if !bitwiseEqual(kmeansModel.Result, directKMeans) {
		t.Fatal("KMeans result changed by wrapper")
	}
	kmeansPrediction, err := kmeansModel.Predict(features.table)
	if err != nil {
		t.Fatal(err)
	}
	assignments, _, err := directKMeans.Assign(features.table)
	if err != nil {
		t.Fatal(err)
	}
	wantAssignments := make([]any, len(assignments))
	for i, assignment := range assignments {
		wantAssignments[i] = assignment
	}
	if !reflect.DeepEqual(kmeansPrediction.Data(), wantAssignments) {
		t.Fatalf("KMeans prediction changed by wrapper: got=%v want=%v", kmeansPrediction.Data(), wantAssignments)
	}

	knnClassifier, err := ml.FitKNNClassifier(features.table, classification, 3)
	if err != nil {
		t.Fatal(err)
	}
	directKNNClassifier, err := stats.KNNClassify(features.table, classification, features.table, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !bitwiseEqual(knnClassifier.(*ml.KNNClassifier).Result, directKNNClassifier) {
		t.Fatal("KNN classifier result changed by wrapper")
	}
	knnClassifierPrediction, err := knnClassifier.Predict(features.table)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(knnClassifierPrediction.Data(), directKNNClassifier.Predictions.Data()) {
		t.Fatalf("KNN classifier prediction changed by wrapper")
	}
	knnClassifierProba, err := knnClassifier.(ml.ProbaModel).PredictProba(features.table)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(knnClassifierProba.To2DSlice(), directKNNClassifier.Probabilities.To2DSlice()) {
		t.Fatalf("KNN classifier probabilities changed by wrapper")
	}

	knnRegressor, err := ml.FitKNNRegressor(features.table, y, 3)
	if err != nil {
		t.Fatal(err)
	}
	directKNNRegressor, err := stats.KNNRegress(features.table, y, features.table, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !bitwiseEqual(knnRegressor.(*ml.KNNRegressor).Result, directKNNRegressor) {
		t.Fatal("KNN regressor result changed by wrapper")
	}
	knnRegressorPrediction, err := knnRegressor.Predict(features.table)
	if err != nil {
		t.Fatal(err)
	}
	knnRegressorWant := make([]any, len(directKNNRegressor.Predictions))
	for i, value := range directKNNRegressor.Predictions {
		knnRegressorWant[i] = value
	}
	if !reflect.DeepEqual(knnRegressorPrediction.Data(), knnRegressorWant) {
		t.Fatalf("KNN regressor prediction changed by wrapper")
	}
}

func TestUnderlyingFitErrorIsReturnedUnchanged(t *testing.T) {
	x := insyra.NewDataTable(dataList([]any{1, 2, 3}, "x"))
	y := dataList([]any{"not numeric", "not numeric", "not numeric"}, "y")
	_, wantErr := stats.LinearRegression(y, x.GetColByNumber(0))
	if wantErr == nil {
		t.Fatal("direct stats fit unexpectedly succeeded")
	}
	_, gotErr := ml.FitLinearRegression(x, y)
	if gotErr == nil || gotErr.Error() != wantErr.Error() {
		t.Fatalf("wrapper error = %v, direct error = %v", gotErr, wantErr)
	}
}

func TestRootScalersAndEncodersArePipelineTransformers(t *testing.T) {
	train := testFeatures().table
	scaler := insyra.NewStandardScaler()
	if _, err := scaler.FitTransform(train, "x1"); err != nil {
		t.Fatal(err)
	}
	var transformer ml.Transformer = scaler
	if _, err := transformer.Transform(train); err != nil {
		t.Fatal(err)
	}

	categories := insyra.NewDataTable(dataList([]any{"red", "blue", "red"}, "color"))
	_, encoder, err := categories.OneHotEncode(insyra.OneHotOptions{Columns: []string{"color"}})
	if err != nil {
		t.Fatal(err)
	}
	transformer = encoder
	if _, err := transformer.Transform(categories); err != nil {
		t.Fatal(err)
	}

	imputationInput := insyra.NewDataTable(dataList([]any{1.0, nil, 3.0}, "value"))
	imputer := insyra.NewSimpleImputer(insyra.ImputeMean)
	if err := imputer.Fit(imputationInput, "value"); err != nil {
		t.Fatal(err)
	}
	transformer = imputer
	if _, err := transformer.Transform(imputationInput); err != nil {
		t.Fatal(err)
	}
}

func assertPredictionMatches(t *testing.T, model ml.Model, input *insyra.DataTable, direct *insyra.DataList, directErr error) {
	t.Helper()
	if directErr != nil {
		t.Fatal(directErr)
	}
	got, err := model.Predict(input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Data(), direct.Data()) {
		t.Fatalf("prediction changed by wrapper: got=%v want=%v", got.Data(), direct.Data())
	}
}

type featureTables struct {
	table *insyra.DataTable
	one   *insyra.DataTable
}

func testFeatures() featureTables {
	x1 := dataList([]any{1, 2, 3, 4, 5, 6, 7, 8}, "x1")
	x2 := dataList([]any{2, 1, 3, 2, 4, 3, 5, 4}, "x2")
	return featureTables{
		table: insyra.NewDataTable(x1, x2),
		one:   insyra.NewDataTable(x1.Clone()),
	}
}

func dataList(values []any, name string) *insyra.DataList {
	return insyra.NewDataList(values...).SetName(name)
}

func bitwiseEqual(left, right any) bool {
	return bitwiseValueEqual(reflect.ValueOf(left), reflect.ValueOf(right))
}

func bitwiseValueEqual(left, right reflect.Value) bool {
	if !left.IsValid() || !right.IsValid() {
		return left.IsValid() == right.IsValid()
	}
	if left.Type() != right.Type() {
		return false
	}
	switch left.Kind() {
	case reflect.Float32:
		return math.Float32bits(float32(left.Float())) == math.Float32bits(float32(right.Float()))
	case reflect.Float64:
		return math.Float64bits(left.Float()) == math.Float64bits(right.Float())
	case reflect.Interface:
		if left.IsNil() || right.IsNil() {
			return left.IsNil() == right.IsNil()
		}
		return bitwiseValueEqual(left.Elem(), right.Elem())
	case reflect.Pointer:
		if left.IsNil() || right.IsNil() {
			return left.IsNil() == right.IsNil()
		}
		return bitwiseValueEqual(left.Elem(), right.Elem())
	case reflect.Struct:
		for i := 0; i < left.NumField(); i++ {
			if !bitwiseValueEqual(left.Field(i), right.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Slice, reflect.Array:
		if left.Len() != right.Len() {
			return false
		}
		for i := 0; i < left.Len(); i++ {
			if !bitwiseValueEqual(left.Index(i), right.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Map:
		if left.IsNil() || right.IsNil() {
			return left.IsNil() == right.IsNil()
		}
		if left.Len() != right.Len() {
			return false
		}
		for _, key := range left.MapKeys() {
			if !bitwiseValueEqual(left.MapIndex(key), right.MapIndex(key)) {
				return false
			}
		}
		return true
	case reflect.Bool:
		return left.Bool() == right.Bool()
	case reflect.String:
		return left.String() == right.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return left.Int() == right.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return left.Uint() == right.Uint()
	default:
		return reflect.DeepEqual(left.Interface(), right.Interface())
	}
}
