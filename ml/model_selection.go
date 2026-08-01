package ml

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/HazelnutParadise/insyra"
)

const rowMarkerPrefix = "__ml_row_index__"

// MetricKind identifies the kind of model a metric can score.
type MetricKind string

const (
	// ClassificationMetric is used for metrics over class labels or class
	// probabilities.
	ClassificationMetric MetricKind = "classification"
	// RegressionMetric is used for metrics over continuous predictions.
	RegressionMetric MetricKind = "regression"
)

// Prediction contains the outputs available to a metric. Values are class or
// continuous predictions; probabilities and classes are populated when the
// fitted model implements ProbaModel.
type Prediction struct {
	Values        *insyra.DataList
	Probabilities *insyra.DataTable
	Classes       *insyra.DataList
}

// MetricResult is the result of scoring one held-out fold.
type MetricResult struct {
	Name            string
	Score           float64
	ConfusionMatrix *ConfusionMatrixResult
}

// Metric scores a prediction and names the quantity it measured.
type Metric interface {
	Name() string
	Kind() MetricKind
	Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error)
}

// CrossValidationResult contains every fold's score and their arithmetic
// mean. Confusion-matrix metrics populate FoldResults and leave Scores and
// Mean as NaN because a matrix is not a scalar.
type CrossValidationResult struct {
	Metric      string
	Scores      []float64
	Mean        float64
	FoldResults []MetricResult
}

// ConfusionMatrixResult stores counts indexed as [actual][predicted].
type ConfusionMatrixResult struct {
	Labels []any
	Counts [][]int
}

// KFold splits a table into k disjoint folds. By default rows are shuffled;
// SamplingOptions{PreserveOrder: true} keeps their original order. A seeded
// SamplingOptions makes the fold assignment reproducible.
func KFold(dt *insyra.DataTable, k int, options ...insyra.SamplingOptions) ([]*insyra.DataTable, error) {
	opts, err := oneSamplingOption(options)
	if err != nil {
		return nil, err
	}
	indices, err := kFoldIndices(dt, k, opts)
	if err != nil {
		return nil, err
	}
	folds := make([]*insyra.DataTable, len(indices))
	for i, fold := range indices {
		folds[i], err = tableFromIndices(dt, fold, false)
		if err != nil {
			return nil, err
		}
	}
	return folds, nil
}

// StratifiedKFold splits a table so each fold retains approximately the class
// proportions in labels. labels must contain one value for every table row.
// Every class needs at least k observations.
func StratifiedKFold(dt *insyra.DataTable, labels *insyra.DataList, k int, options ...insyra.SamplingOptions) ([]*insyra.DataTable, error) {
	opts, err := oneSamplingOption(options)
	if err != nil {
		return nil, err
	}
	indices, err := stratifiedKFoldIndices(dt, labels, k, opts)
	if err != nil {
		return nil, err
	}
	folds := make([]*insyra.DataTable, len(indices))
	for i, fold := range indices {
		folds[i], err = tableFromIndices(dt, fold, false)
		if err != nil {
			return nil, err
		}
	}
	return folds, nil
}

// CrossValidate fits estimator once per fold and evaluates the supplied
// metric on that fold's held-out rows. Classification metrics use a
// stratified split; regression metrics use an ordinary k-fold split.
func CrossValidate(x *insyra.DataTable, y *insyra.DataList, estimator Estimator, k int, metric Metric, options ...insyra.SamplingOptions) (*CrossValidationResult, error) {
	if err := requireTable(x); err != nil {
		return nil, err
	}
	if y == nil || isNilPointer(y) {
		return nil, fmt.Errorf("ml: target data list is nil")
	}
	if x.NumRows() != y.Len() {
		return nil, fmt.Errorf("ml: feature rows (%d) and target values (%d) do not match", x.NumRows(), y.Len())
	}
	if estimator.Fit == nil {
		return nil, fmt.Errorf("ml: estimator %q has no Fit function", estimator.Name)
	}
	if metric == nil || isNilPointer(metric) {
		return nil, fmt.Errorf("ml: metric is nil")
	}
	opts, err := oneSamplingOption(options)
	if err != nil {
		return nil, err
	}

	var foldIndices [][]int
	if metric.Kind() == ClassificationMetric {
		foldIndices, err = stratifiedKFoldIndices(x, y, k, opts)
	} else {
		foldIndices, err = kFoldIndices(x, k, opts)
	}
	if err != nil {
		return nil, err
	}

	result := &CrossValidationResult{
		Metric:      metric.Name(),
		Scores:      make([]float64, 0, len(foldIndices)),
		Mean:        math.NaN(),
		FoldResults: make([]MetricResult, 0, len(foldIndices)),
	}
	meanTotal := 0.0
	meanCount := 0
	for foldNumber, testIndices := range foldIndices {
		trainIndices := complementIndices(x.NumRows(), testIndices)
		trainX, err := tableFromIndices(x, trainIndices, false)
		if err != nil {
			return nil, fmt.Errorf("ml: fold %d: build training table: %w", foldNumber+1, err)
		}
		testX, err := tableFromIndices(x, testIndices, false)
		if err != nil {
			return nil, fmt.Errorf("ml: fold %d: build test table: %w", foldNumber+1, err)
		}
		trainY := listFromIndices(y, trainIndices)
		model, err := estimator.Fit(trainX, trainY)
		if err != nil {
			return nil, fmt.Errorf("ml: fold %d: fit failed: %w", foldNumber+1, err)
		}
		if err := validateMetricModel(metric, model); err != nil {
			return nil, fmt.Errorf("ml: fold %d: %w", foldNumber+1, err)
		}

		prediction, err := predictionForMetric(model, testX, metric)
		if err != nil {
			return nil, fmt.Errorf("ml: fold %d: predict failed: %w", foldNumber+1, err)
		}
		score, err := metric.Evaluate(listFromIndices(y, testIndices), prediction)
		if err != nil {
			return nil, fmt.Errorf("ml: fold %d: score failed: %w", foldNumber+1, err)
		}
		// Metric.Name is the authoritative name of the supplied metric. Do
		// not let a custom evaluator return a different display name for one
		// fold and make the aggregate result ambiguous.
		score.Name = metric.Name()
		result.FoldResults = append(result.FoldResults, score)
		result.Scores = append(result.Scores, score.Score)
		if !math.IsNaN(score.Score) {
			meanTotal += score.Score
			meanCount++
		}
	}
	if meanCount > 0 {
		result.Mean = meanTotal / float64(meanCount)
	}
	return result, nil
}

// Accuracy returns the fraction of exact label matches.
func Accuracy(yTrue, yPred *insyra.DataList) (float64, error) {
	if err := validatePredictionLists(yTrue, yPred); err != nil {
		return 0, err
	}
	if yTrue.Len() == 0 {
		return 0, fmt.Errorf("ml: accuracy is undefined for empty data")
	}
	correct := 0
	for i := 0; i < yTrue.Len(); i++ {
		if sameLabel(yTrue.Get(i), yPred.Get(i)) {
			correct++
		}
	}
	return float64(correct) / float64(yTrue.Len()), nil
}

// LogLoss returns multiclass or binary cross-entropy from a probability table.
func LogLoss(yTrue *insyra.DataList, probabilities *insyra.DataTable, classes ...*insyra.DataList) (float64, error) {
	prediction, err := probabilityPrediction(probabilities, classes)
	if err != nil {
		return 0, err
	}
	return logLossScore(yTrue, prediction)
}

// ROCAUC returns the binary receiver-operating characteristic area under the
// curve. The probability for the second class is treated as the positive score.
func ROCAUC(yTrue *insyra.DataList, probabilities *insyra.DataTable, classes ...*insyra.DataList) (float64, error) {
	prediction, err := probabilityPrediction(probabilities, classes)
	if err != nil {
		return 0, err
	}
	return rocAUCScore(yTrue, prediction)
}

// ConfusionMatrix returns counts indexed as [actual][predicted]. Labels are
// ordered by first appearance in yTrue followed by new predicted labels.
func ConfusionMatrix(yTrue, yPred *insyra.DataList) (*ConfusionMatrixResult, error) {
	if err := validatePredictionLists(yTrue, yPred); err != nil {
		return nil, err
	}
	labels := make([]any, 0)
	addLabel := func(value any) {
		for _, label := range labels {
			if sameLabel(label, value) {
				return
			}
		}
		labels = append(labels, value)
	}
	for i := 0; i < yTrue.Len(); i++ {
		addLabel(yTrue.Get(i))
	}
	for i := 0; i < yPred.Len(); i++ {
		addLabel(yPred.Get(i))
	}
	counts := make([][]int, len(labels))
	for i := range counts {
		counts[i] = make([]int, len(labels))
	}
	for i := 0; i < yTrue.Len(); i++ {
		actual := labelIndex(labels, yTrue.Get(i))
		predicted := labelIndex(labels, yPred.Get(i))
		counts[actual][predicted]++
	}
	return &ConfusionMatrixResult{Labels: labels, Counts: counts}, nil
}

// RMSE returns the root mean squared error.
func RMSE(yTrue, yPred *insyra.DataList) (float64, error) {
	actual, predicted, err := numericPair(yTrue, yPred)
	if err != nil {
		return 0, err
	}
	if len(actual) == 0 {
		return 0, fmt.Errorf("ml: RMSE is undefined for empty data")
	}
	total := 0.0
	for i := range actual {
		delta := predicted[i] - actual[i]
		total += delta * delta
	}
	return math.Sqrt(total / float64(len(actual))), nil
}

// MAE returns the mean absolute error.
func MAE(yTrue, yPred *insyra.DataList) (float64, error) {
	actual, predicted, err := numericPair(yTrue, yPred)
	if err != nil {
		return 0, err
	}
	if len(actual) == 0 {
		return 0, fmt.Errorf("ml: MAE is undefined for empty data")
	}
	total := 0.0
	for i := range actual {
		total += math.Abs(predicted[i] - actual[i])
	}
	return total / float64(len(actual)), nil
}

// R2 returns the coefficient of determination.
func R2(yTrue, yPred *insyra.DataList) (float64, error) {
	actual, predicted, err := numericPair(yTrue, yPred)
	if err != nil {
		return 0, err
	}
	if len(actual) == 0 {
		return 0, fmt.Errorf("ml: R² is undefined for empty data")
	}
	mean := 0.0
	for _, value := range actual {
		mean += value
	}
	mean /= float64(len(actual))
	residual, total := 0.0, 0.0
	for i, value := range actual {
		delta := predicted[i] - value
		residual += delta * delta
		centered := value - mean
		total += centered * centered
	}
	if total == 0 {
		return 0, fmt.Errorf("ml: R² is undefined when all observed values are equal")
	}
	return 1 - residual/total, nil
}

// AccuracyMetric scores class predictions.
type AccuracyMetric struct{}

func (AccuracyMetric) Name() string     { return "accuracy" }
func (AccuracyMetric) Kind() MetricKind { return ClassificationMetric }
func (AccuracyMetric) Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error) {
	score, err := Accuracy(yTrue, prediction.Values)
	return MetricResult{Name: "accuracy", Score: score}, err
}
func (AccuracyMetric) needsClassLabels() bool { return true }

// LogLossMetric scores class probabilities with logarithmic loss.
type LogLossMetric struct{}

func (LogLossMetric) Name() string     { return "log_loss" }
func (LogLossMetric) Kind() MetricKind { return ClassificationMetric }
func (LogLossMetric) Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error) {
	score, err := logLossScore(yTrue, prediction)
	return MetricResult{Name: "log_loss", Score: score}, err
}
func (LogLossMetric) needsProbabilities() bool { return true }

// ROCAUCMetric scores binary class probabilities using the second class as
// the positive class.
type ROCAUCMetric struct{}

func (ROCAUCMetric) Name() string     { return "roc_auc" }
func (ROCAUCMetric) Kind() MetricKind { return ClassificationMetric }
func (ROCAUCMetric) Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error) {
	score, err := rocAUCScore(yTrue, prediction)
	return MetricResult{Name: "roc_auc", Score: score}, err
}
func (ROCAUCMetric) needsProbabilities() bool { return true }

// ConfusionMatrixMetric returns a confusion matrix for each fold. Its scalar
// Score is NaN because a confusion matrix has no single aggregate value.
type ConfusionMatrixMetric struct{}

func (ConfusionMatrixMetric) Name() string     { return "confusion_matrix" }
func (ConfusionMatrixMetric) Kind() MetricKind { return ClassificationMetric }
func (ConfusionMatrixMetric) Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error) {
	matrix, err := ConfusionMatrix(yTrue, prediction.Values)
	return MetricResult{Name: "confusion_matrix", Score: math.NaN(), ConfusionMatrix: matrix}, err
}
func (ConfusionMatrixMetric) needsClassLabels() bool { return true }

// RMSEMetric scores continuous predictions with root mean squared error.
type RMSEMetric struct{}

func (RMSEMetric) Name() string     { return "rmse" }
func (RMSEMetric) Kind() MetricKind { return RegressionMetric }
func (RMSEMetric) Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error) {
	score, err := RMSE(yTrue, prediction.Values)
	return MetricResult{Name: "rmse", Score: score}, err
}

// MAEMetric scores continuous predictions with mean absolute error.
type MAEMetric struct{}

func (MAEMetric) Name() string     { return "mae" }
func (MAEMetric) Kind() MetricKind { return RegressionMetric }
func (MAEMetric) Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error) {
	score, err := MAE(yTrue, prediction.Values)
	return MetricResult{Name: "mae", Score: score}, err
}

// R2Metric scores continuous predictions with R².
type R2Metric struct{}

func (R2Metric) Name() string     { return "r2" }
func (R2Metric) Kind() MetricKind { return RegressionMetric }
func (R2Metric) Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error) {
	score, err := R2(yTrue, prediction.Values)
	return MetricResult{Name: "r2", Score: score}, err
}

type classLabelMetric interface {
	needsClassLabels() bool
}

type probabilityMetric interface {
	needsProbabilities() bool
}

func oneSamplingOption(options []insyra.SamplingOptions) (insyra.SamplingOptions, error) {
	if len(options) > 1 {
		return insyra.SamplingOptions{}, fmt.Errorf("ml: sampling options accepts at most one value")
	}
	if len(options) == 0 {
		return insyra.SamplingOptions{}, nil
	}
	return options[0], nil
}

func kFoldIndices(dt *insyra.DataTable, k int, options insyra.SamplingOptions) ([][]int, error) {
	if err := requireTable(dt); err != nil {
		return nil, err
	}
	indices := make([]int, dt.NumRows())
	for i := range indices {
		indices[i] = i
	}
	marked, err := tableFromIndices(dt, indices, true)
	if err != nil {
		return nil, err
	}
	return splitMarkedTable(marked, k, options)
}

func stratifiedKFoldIndices(dt *insyra.DataTable, labels *insyra.DataList, k int, options insyra.SamplingOptions) ([][]int, error) {
	if err := requireTable(dt); err != nil {
		return nil, err
	}
	if labels == nil || isNilPointer(labels) {
		return nil, fmt.Errorf("ml: labels data list is nil")
	}
	if dt.NumRows() != labels.Len() {
		return nil, fmt.Errorf("ml: table rows (%d) and labels (%d) do not match", dt.NumRows(), labels.Len())
	}
	if k < 2 {
		return nil, fmt.Errorf("ml: k must be at least 2, got %d", k)
	}
	if dt.NumRows() < k {
		return nil, fmt.Errorf("ml: k (%d) cannot exceed row count (%d)", k, dt.NumRows())
	}

	type labelGroup struct {
		label   any
		indices []int
	}
	groups := make([]labelGroup, 0)
	for i, value := range labels.Data() {
		groupIndex := -1
		for j := range groups {
			if sameLabel(groups[j].label, value) {
				groupIndex = j
				break
			}
		}
		if groupIndex == -1 {
			groups = append(groups, labelGroup{label: value, indices: []int{i}})
		} else {
			groups[groupIndex].indices = append(groups[groupIndex].indices, i)
		}
	}

	folds := make([][]int, k)
	for _, group := range groups {
		if len(group.indices) < k {
			return nil, fmt.Errorf("ml: class %v has %d observations, fewer than %d folds", group.label, len(group.indices), k)
		}
		marked, err := tableFromIndices(dt, group.indices, true)
		if err != nil {
			return nil, err
		}
		groupFolds, err := splitMarkedTable(marked, k, options)
		if err != nil {
			return nil, fmt.Errorf("ml: class %v: %w", group.label, err)
		}
		for i := range folds {
			folds[i] = append(folds[i], groupFolds[i]...)
		}
	}
	for i := range folds {
		sort.Ints(folds[i])
	}
	return folds, nil
}

func splitMarkedTable(marked *insyra.DataTable, k int, options insyra.SamplingOptions) ([][]int, error) {
	if err := requireTable(marked); err != nil {
		return nil, err
	}
	rows := marked.NumRows()
	if k < 2 {
		return nil, fmt.Errorf("ml: k must be at least 2, got %d", k)
	}
	if rows < k {
		return nil, fmt.Errorf("ml: k (%d) cannot exceed row count (%d)", k, rows)
	}
	remaining := marked
	if !options.PreserveOrder {
		remaining = marked.Shuffle(options)
	}
	folds := make([][]int, 0, k)
	for foldNumber := 0; foldNumber < k-1; foldNumber++ {
		remainingRows := remaining.NumRows()
		foldsLeft := k - foldNumber
		holdoutRows := remainingRows / foldsLeft
		if holdoutRows < 1 {
			return nil, fmt.Errorf("ml: cannot allocate fold %d from %d remaining rows", foldNumber+1, remainingRows)
		}
		trainRows := remainingRows - holdoutRows
		// Add a half-row margin so TrainTestSplit's floor-based count remains
		// stable even when the quotient is just below an integer in float64.
		trainFrac := (float64(trainRows) + 0.5) / float64(remainingRows)
		train, test := remaining.TrainTestSplit(trainFrac, insyra.SamplingOptions{PreserveOrder: true})
		if train.NumRows() != trainRows || test.NumRows() != holdoutRows {
			return nil, fmt.Errorf("ml: TrainTestSplit returned unexpected fold sizes (%d,%d), want (%d,%d)", train.NumRows(), test.NumRows(), trainRows, holdoutRows)
		}
		fold, err := parseMarkedRows(test)
		if err != nil {
			return nil, fmt.Errorf("ml: fold %d: %w", foldNumber+1, err)
		}
		folds = append(folds, fold)
		remaining = train
	}
	last, err := parseMarkedRows(remaining)
	if err != nil {
		return nil, fmt.Errorf("ml: fold %d: %w", k, err)
	}
	return append(folds, last), nil
}

func parseMarkedRows(dt *insyra.DataTable) ([]int, error) {
	names := dt.RowNames()
	indices := make([]int, len(names))
	for i, name := range names {
		if !strings.HasPrefix(name, rowMarkerPrefix) {
			return nil, fmt.Errorf("row %d has no internal marker", i)
		}
		index, err := strconv.Atoi(strings.TrimPrefix(name, rowMarkerPrefix))
		if err != nil || index < 0 {
			return nil, fmt.Errorf("invalid row marker %q", name)
		}
		indices[i] = index
	}
	return indices, nil
}

func tableFromIndices(dt *insyra.DataTable, indices []int, marked bool) (*insyra.DataTable, error) {
	if err := requireTable(dt); err != nil {
		return nil, err
	}
	for _, index := range indices {
		if index < 0 || index >= dt.NumRows() {
			return nil, fmt.Errorf("ml: row index %d is out of range", index)
		}
	}
	columns := make([]*insyra.DataList, dt.NumCols())
	for i, name := range dt.ColNames() {
		columns[i] = insyra.NewDataList().SetName(name)
	}
	out := insyra.NewDataTable(columns...)
	for _, index := range indices {
		row := dt.GetRow(index)
		if row == nil {
			return nil, fmt.Errorf("ml: row index %d could not be read", index)
		}
		if marked {
			row.SetName(rowMarkerPrefix + strconv.Itoa(index))
		}
		out.AppendRowsFromDataList(row)
	}
	return out, nil
}

func listFromIndices(list *insyra.DataList, indices []int) *insyra.DataList {
	values := make([]any, len(indices))
	for i, index := range indices {
		values[i] = list.Get(index)
	}
	return insyra.NewDataList(values...).SetName(list.GetName())
}

func complementIndices(size int, excluded []int) []int {
	excludedSet := make(map[int]struct{}, len(excluded))
	for _, index := range excluded {
		excludedSet[index] = struct{}{}
	}
	result := make([]int, 0, size-len(excluded))
	for i := 0; i < size; i++ {
		if _, ok := excludedSet[i]; !ok {
			result = append(result, i)
		}
	}
	return result
}

func validateMetricModel(metric Metric, model Model) error {
	if model == nil || isNilPointer(model) {
		return fmt.Errorf("estimator returned a nil model")
	}
	_, isClassifier := model.(Classifier)
	switch metric.Kind() {
	case ClassificationMetric:
		if !isClassifier {
			return fmt.Errorf("metric %q requires a classification model; %T is not a Classifier", metric.Name(), model)
		}
	case RegressionMetric:
		if isClassifier {
			return fmt.Errorf("metric %q requires a regression model; %T is a Classifier", metric.Name(), model)
		}
	}
	if _, needsProbability := metric.(probabilityMetric); needsProbability {
		if _, ok := model.(ProbaModel); !ok {
			return fmt.Errorf("metric %q requires a model implementing ProbaModel; %T does not", metric.Name(), model)
		}
	}
	return nil
}

func predictionForMetric(model Model, test *insyra.DataTable, metric Metric) (Prediction, error) {
	proba, hasProba := model.(ProbaModel)
	classes := (*insyra.DataList)(nil)
	if hasProba {
		classes = proba.Classes()
	}
	if _, needsProbability := metric.(probabilityMetric); needsProbability {
		probabilities, err := proba.PredictProba(test)
		if err != nil {
			return Prediction{}, err
		}
		return Prediction{Probabilities: probabilities, Classes: classes}, nil
	}
	if _, needsLabels := metric.(classLabelMetric); needsLabels && hasProba {
		probabilities, err := proba.PredictProba(test)
		if err != nil {
			return Prediction{}, err
		}
		values, err := classLabelsFromProbabilities(probabilities, classes)
		if err != nil {
			return Prediction{}, err
		}
		return Prediction{Values: values, Probabilities: probabilities, Classes: classes}, nil
	}
	values, err := model.Predict(test)
	return Prediction{Values: values, Classes: classes}, err
}

func classLabelsFromProbabilities(probabilities *insyra.DataTable, classes *insyra.DataList) (*insyra.DataList, error) {
	if probabilities == nil || isNilPointer(probabilities) {
		return nil, fmt.Errorf("ml: probability table is nil")
	}
	if probabilities.NumCols() == 0 {
		return nil, fmt.Errorf("ml: probability table has no classes")
	}
	classValues := make([]any, probabilities.NumCols())
	if classes != nil && !isNilPointer(classes) {
		if classes.Len() != probabilities.NumCols() {
			return nil, fmt.Errorf("ml: probability classes (%d) and columns (%d) do not match", classes.Len(), probabilities.NumCols())
		}
		copy(classValues, classes.Data())
	} else {
		for i, name := range probabilities.ColNames() {
			classValues[i] = name
		}
	}
	values := make([]any, probabilities.NumRows())
	for row := 0; row < probabilities.NumRows(); row++ {
		best := 0
		bestValue := math.Inf(-1)
		for col := 0; col < probabilities.NumCols(); col++ {
			value, ok := insyra.ToFloat64Safe(probabilities.GetElementByNumberIndex(row, col))
			if !ok || math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("ml: probability at row %d, class %d is not finite", row, col)
			}
			if value > bestValue {
				bestValue = value
				best = col
			}
		}
		values[row] = classValues[best]
	}
	return insyra.NewDataList(values...), nil
}

func probabilityPrediction(probabilities *insyra.DataTable, classes []*insyra.DataList) (Prediction, error) {
	if len(classes) > 1 {
		return Prediction{}, fmt.Errorf("ml: classes accepts at most one value")
	}
	if probabilities == nil || isNilPointer(probabilities) {
		return Prediction{}, fmt.Errorf("ml: probability table is nil")
	}
	var classList *insyra.DataList
	if len(classes) == 1 {
		classList = classes[0]
	}
	if probabilities.NumCols() == 0 {
		return Prediction{}, fmt.Errorf("ml: probability table has no classes")
	}
	return Prediction{Probabilities: probabilities, Classes: classList}, nil
}

func logLossScore(yTrue *insyra.DataList, prediction Prediction) (float64, error) {
	if yTrue == nil || isNilPointer(yTrue) {
		return 0, fmt.Errorf("ml: true labels are nil")
	}
	probabilities := prediction.Probabilities
	if probabilities == nil || isNilPointer(probabilities) {
		return 0, fmt.Errorf("ml: log loss requires class probabilities")
	}
	if probabilities.NumRows() != yTrue.Len() {
		return 0, fmt.Errorf("ml: true labels (%d) and probability rows (%d) do not match", yTrue.Len(), probabilities.NumRows())
	}
	if yTrue.Len() == 0 {
		return 0, fmt.Errorf("ml: log loss is undefined for empty data")
	}
	classValues, err := probabilityClassValues(probabilities, prediction.Classes)
	if err != nil {
		return 0, err
	}
	total := 0.0
	for row := 0; row < yTrue.Len(); row++ {
		class := labelIndex(classValues, yTrue.Get(row))
		if class < 0 {
			return 0, fmt.Errorf("ml: true label %v is absent from probability classes", yTrue.Get(row))
		}
		probability, ok := insyra.ToFloat64Safe(probabilities.GetElementByNumberIndex(row, class))
		if !ok || math.IsNaN(probability) || probability < 0 || probability > 1 {
			return 0, fmt.Errorf("ml: probability at row %d, class %d is outside [0,1]", row, class)
		}
		if probability == 0 {
			return math.Inf(1), nil
		}
		total -= math.Log(probability)
	}
	return total / float64(yTrue.Len()), nil
}

func rocAUCScore(yTrue *insyra.DataList, prediction Prediction) (float64, error) {
	if yTrue == nil || isNilPointer(yTrue) {
		return 0, fmt.Errorf("ml: true labels are nil")
	}
	probabilities := prediction.Probabilities
	if probabilities == nil || isNilPointer(probabilities) {
		return 0, fmt.Errorf("ml: ROC AUC requires class probabilities")
	}
	if probabilities.NumCols() != 2 {
		return 0, fmt.Errorf("ml: ROC AUC requires exactly two probability classes, got %d", probabilities.NumCols())
	}
	if probabilities.NumRows() != yTrue.Len() || yTrue.Len() == 0 {
		return 0, fmt.Errorf("ml: ROC AUC requires non-empty labels aligned with probabilities")
	}
	classValues, err := probabilityClassValues(probabilities, prediction.Classes)
	if err != nil {
		return 0, err
	}
	positive := classValues[1]
	type scoredLabel struct {
		score float64
		pos   bool
	}
	values := make([]scoredLabel, yTrue.Len())
	positiveCount := 0
	for row := 0; row < yTrue.Len(); row++ {
		score, ok := insyra.ToFloat64Safe(probabilities.GetElementByNumberIndex(row, 1))
		if !ok || math.IsNaN(score) || math.IsInf(score, 0) {
			return 0, fmt.Errorf("ml: ROC AUC score at row %d is not finite", row)
		}
		// A label belonging to neither class is not a negative — it is a
		// malformed input, and treating it as negative reports confident
		// discrimination over data the metric never understood. logLossScore
		// already refuses the same input; the two must not disagree.
		if labelIndex(classValues, yTrue.Get(row)) < 0 {
			return 0, fmt.Errorf("ml: true label %v is absent from probability classes", yTrue.Get(row))
		}
		isPositive := sameLabel(yTrue.Get(row), positive)
		if isPositive {
			positiveCount++
		}
		values[row] = scoredLabel{score: score, pos: isPositive}
	}
	negativeCount := len(values) - positiveCount
	if positiveCount == 0 || negativeCount == 0 {
		return 0, fmt.Errorf("ml: ROC AUC requires both classes in the true labels")
	}
	sort.SliceStable(values, func(i, j int) bool { return values[i].score < values[j].score })
	rankSum := 0.0
	for i := 0; i < len(values); {
		j := i + 1
		for j < len(values) && values[j].score == values[i].score {
			j++
		}
		averageRank := (float64(i+1) + float64(j)) / 2
		for index := i; index < j; index++ {
			if values[index].pos {
				rankSum += averageRank
			}
		}
		i = j
	}
	return (rankSum - float64(positiveCount*(positiveCount+1))/2) / float64(positiveCount*negativeCount), nil
}

func probabilityClassValues(probabilities *insyra.DataTable, classes *insyra.DataList) ([]any, error) {
	values := make([]any, probabilities.NumCols())
	if classes != nil && !isNilPointer(classes) {
		if classes.Len() != probabilities.NumCols() {
			return nil, fmt.Errorf("ml: probability classes (%d) and columns (%d) do not match", classes.Len(), probabilities.NumCols())
		}
		copy(values, classes.Data())
		return values, nil
	}
	for i, name := range probabilities.ColNames() {
		values[i] = name
	}
	return values, nil
}

func validatePredictionLists(actual, predicted *insyra.DataList) error {
	if actual == nil || isNilPointer(actual) {
		return fmt.Errorf("ml: true values are nil")
	}
	if predicted == nil || isNilPointer(predicted) {
		return fmt.Errorf("ml: predicted values are nil")
	}
	if actual.Len() != predicted.Len() {
		return fmt.Errorf("ml: true values (%d) and predictions (%d) do not match", actual.Len(), predicted.Len())
	}
	return nil
}

func numericPair(actual, predicted *insyra.DataList) ([]float64, []float64, error) {
	if err := validatePredictionLists(actual, predicted); err != nil {
		return nil, nil, err
	}
	actualValues := make([]float64, actual.Len())
	predictedValues := make([]float64, predicted.Len())
	for i := 0; i < actual.Len(); i++ {
		var ok bool
		actualValues[i], ok = insyra.ToFloat64Safe(actual.Get(i))
		if !ok || math.IsNaN(actualValues[i]) {
			return nil, nil, fmt.Errorf("ml: true value at row %d is not numeric", i)
		}
		predictedValues[i], ok = insyra.ToFloat64Safe(predicted.Get(i))
		if !ok || math.IsNaN(predictedValues[i]) {
			return nil, nil, fmt.Errorf("ml: prediction at row %d is not numeric", i)
		}
	}
	return actualValues, predictedValues, nil
}

func sameLabel(left, right any) bool {
	if reflect.DeepEqual(left, right) {
		return true
	}
	leftNumber, leftOK := insyra.ToFloat64Safe(left)
	rightNumber, rightOK := insyra.ToFloat64Safe(right)
	return leftOK && rightOK && leftNumber == rightNumber
}

func labelIndex(labels []any, value any) int {
	for i, label := range labels {
		if sameLabel(label, value) {
			return i
		}
	}
	return -1
}
