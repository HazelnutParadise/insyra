package ml_test

import (
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
	"github.com/HazelnutParadise/insyra/ml/mltest"
)

func TestDecisionTreeClassificationAndProbabilities(t *testing.T) {
	x := insyra.NewDataTable(
		insyra.NewDataList(0, 1, 2, 3, 4, 5).SetName("x"),
	)
	y := insyra.NewDataList("low", "low", "low", "high", "high", "high")
	model, err := ml.FitDecisionTreeClassifier(x, y, ml.DecisionTreeOptions{MaxBins: 4})
	if err != nil {
		t.Fatal(err)
	}
	predictions, err := model.Predict(x)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(predictions.Data(), y.Data()) {
		t.Fatalf("predictions = %v, want %v", predictions.Data(), y.Data())
	}
	probabilities, err := model.PredictProba(x)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(model.Classes().Data(), []any{"high", "low"}) {
		t.Fatalf("classes = %v, want sorted classes", model.Classes().Data())
	}
	if !reflect.DeepEqual(probabilities.ColNames(), []string{"high", "low"}) {
		t.Fatalf("probability columns = %v", probabilities.ColNames())
	}
	for row := 0; row < probabilities.NumRows(); row++ {
		sum := 0.0
		for col := 0; col < probabilities.NumCols(); col++ {
			sum += probabilities.GetColByNumber(col).ToF64Slice()[row]
		}
		if math.Abs(sum-1) > 1e-12 {
			t.Fatalf("probability row %d sums to %.17g", row, sum)
		}
	}
	if got := model.FeatureImportances(); len(got) != 1 || got[0] != 1 {
		t.Fatalf("feature importances = %v, want [1]", got)
	}
}

func TestDecisionTreeRegressionReportsDoubleValues(t *testing.T) {
	x := insyra.NewDataTable(insyra.NewDataList(0, 1, 2, 3).SetName("x"))
	y := insyra.NewDataList(0.0, 0.0, 10.0, 10.0)
	model, err := ml.FitDecisionTreeRegressor(x, y, ml.DecisionTreeOptions{MaxBins: 4})
	if err != nil {
		t.Fatal(err)
	}
	predictions, err := model.Predict(x)
	if err != nil {
		t.Fatal(err)
	}
	for index, value := range predictions.ToF64Slice() {
		want := 0.0
		if index >= 2 {
			want = 10.0
		}
		if math.Abs(value-want) > 1e-6 {
			t.Fatalf("prediction[%d] = %.17g, want %.17g", index, value, want)
		}
	}
	for _, value := range model.LeafValues() {
		if _, ok := any(value).(float64); !ok {
			t.Fatalf("leaf value %v is not float64", value)
		}
	}
	if got := model.FeatureImportances(); len(got) != 1 || got[0] != 1 {
		t.Fatalf("feature importances = %v, want [1]", got)
	}
}

func TestDecisionTreeRegressionUsesMSEAndRetainsSmallTargets(t *testing.T) {
	x := insyra.NewDataTable(insyra.NewDataList(0, 1).SetName("x"))
	y := insyra.NewDataList(1e-9, 3e-9)
	model, err := ml.FitDecisionTreeRegressor(x, y, ml.DecisionTreeOptions{MinSamplesLeaf: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := model.Root.Impurity, 1e-18; math.Abs(got-want) > 1e-27 {
		t.Fatalf("root impurity = %.17g, want MSE %.17g", got, want)
	}
	if got, want := model.Root.Value, 2e-9; math.Abs(got-want) > 1e-15 {
		t.Fatalf("root value = %.17g, want %.17g", got, want)
	}
}

func TestDecisionTreesSatisfyMLConformance(t *testing.T) {
	x := insyra.NewDataTable(
		insyra.NewDataList(0, 1, 2, 3, 4, 5).SetName("x1"),
		insyra.NewDataList(5, 4, 3, 2, 1, 0).SetName("x2"),
	)
	classification, err := ml.FitDecisionTreeClassifier(x, insyra.NewDataList("a", "a", "a", "b", "b", "b"))
	if err != nil {
		t.Fatal(err)
	}
	mltest.RunConformance(t, classification, x, nil)
	regression, err := ml.FitDecisionTreeRegressor(x, insyra.NewDataList(0.0, 0.0, 0.0, 1.0, 1.0, 1.0))
	if err != nil {
		t.Fatal(err)
	}
	mltest.RunConformance(t, regression, x, nil)
}

func TestDecisionTreeIsDeterministicIncludingTiesAndRowPermutation(t *testing.T) {
	x := insyra.NewDataTable(
		insyra.NewDataList(0, 1, 2, 3, 4, 5).SetName("first"),
		insyra.NewDataList(0, 1, 2, 3, 4, 5).SetName("second"),
	)
	y := insyra.NewDataList(0.0, 0.0, 0.0, 1.0, 1.0, 1.0)
	opts := ml.DecisionTreeOptions{MaxBins: 4, MaxDepth: 2}
	first, err := ml.FitDecisionTreeRegressor(x, y, opts)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ml.FitDecisionTreeRegressor(x, y, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Root, second.Root) {
		t.Fatal("repeated fits produced different trees")
	}
	if first.Root.Feature != 0 {
		t.Fatalf("tied root split used feature %d, want first feature", first.Root.Feature)
	}
	permuted := permuteTable(x, []int{5, 2, 0, 4, 1, 3})
	permutedY := permuteList(y, []int{5, 2, 0, 4, 1, 3})
	third, err := ml.FitDecisionTreeRegressor(permuted, permutedY, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Root, third.Root) {
		t.Fatal("row permutation produced a different tree")
	}
}

func TestDecisionTreeLearnsMissingDirectionAndDefaultsMissingScoringValues(t *testing.T) {
	x := insyra.NewDataTable(insyra.NewDataList(0, 0, 1, 1, math.NaN()).SetName("x"))
	y := insyra.NewDataList(0, 0, 1, 1, 0)
	model, err := ml.FitDecisionTreeClassifier(x, y, ml.DecisionTreeOptions{MaxBins: 3})
	if err != nil {
		t.Fatal(err)
	}
	if !model.Root.MissingGoLeft {
		t.Fatal("missing direction = right, want the better left branch")
	}
	test := insyra.NewDataTable(insyra.NewDataList(math.NaN()).SetName("x"))
	predictions, err := model.Predict(test)
	if err != nil {
		t.Fatal(err)
	}
	if predictions.Get(0) != 0 {
		t.Fatalf("missing scoring prediction = %v, want 0", predictions.Get(0))
	}
}

func TestDecisionTreeCategoricalSubsetAndUnseenDefault(t *testing.T) {
	x := insyra.NewDataTable(insyra.NewDataList("a", "b", "c", "a", "b", "c").SetName("kind"))
	y := insyra.NewDataList(0, 1, 0, 0, 1, 0)
	model, err := ml.FitDecisionTreeClassifier(x, y, ml.DecisionTreeOptions{CategoricalFeatures: []string{"kind"}})
	if err != nil {
		t.Fatal(err)
	}
	if !model.Root.Categorical || len(model.Root.Categories) != 2 {
		t.Fatalf("root categorical split = %#v, want a two-category subset", model.Root)
	}
	training, err := model.Predict(x)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(training.Data(), y.Data()) {
		t.Fatalf("categorical predictions = %v, want %v", training.Data(), y.Data())
	}
	unseen, err := model.Predict(insyra.NewDataTable(insyra.NewDataList("unseen").SetName("kind")))
	if err != nil {
		t.Fatal(err)
	}
	if unseen.Get(0) != 0 {
		t.Fatalf("unseen category prediction = %v, want left default 0", unseen.Get(0))
	}
}

func TestDecisionTreeGrowthBounds(t *testing.T) {
	x := insyra.NewDataTable(insyra.NewDataList(0, 1, 2, 3, 4, 5, 6, 7).SetName("x"))
	y := insyra.NewDataList(0, 0, 1, 1, 2, 2, 3, 3)
	model, err := ml.FitDecisionTreeRegressor(x, y, ml.DecisionTreeOptions{
		MaxBins:        8,
		MaxDepth:       2,
		MaxLeaves:      3,
		MinSamplesLeaf: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	leaves, depth := treeBounds(model.Root)
	if leaves > 3 || depth > 2 {
		t.Fatalf("tree bounds: leaves=%d depth=%d", leaves, depth)
	}
	checkMinSamples(t, model.Root, 2)
}

func TestDecisionTreeAccuracyAgainstScikitLearn(t *testing.T) {
	if os.Getenv("INSYRA_RUN_ML_SKLEARN") != "1" {
		t.Skip("set INSYRA_RUN_ML_SKLEARN=1 to compare against scikit-learn")
	}
	xTrain := [][]float64{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {2, 0}, {2, 1}, {3, 0}, {3, 1}}
	yTrain := []any{0, 0, 1, 1, 1, 1, 0, 0}
	xTest := [][]float64{{0.2, 0.2}, {0.8, 0.8}, {1.2, 0.2}, {2.8, 0.8}}
	yTest := []any{0, 1, 1, 0}
	train := tableFromRows(xTrain, []string{"x1", "x2"})
	test := tableFromRows(xTest, []string{"x1", "x2"})
	model, err := ml.FitDecisionTreeClassifier(train, insyra.NewDataList(yTrain...), ml.DecisionTreeOptions{MaxDepth: 2, MaxBins: 8})
	if err != nil {
		t.Fatal(err)
	}
	predicted, err := model.Predict(test)
	if err != nil {
		t.Fatal(err)
	}
	goAccuracy := accuracy(predicted.Data(), yTest)
	python := "import json; from sklearn.tree import DecisionTreeClassifier; X=[[0,0],[0,1],[1,0],[1,1],[2,0],[2,1],[3,0],[3,1]]; y=[0,0,1,1,1,1,0,0]; xt=[[0.2,0.2],[0.8,0.8],[1.2,0.2],[2.8,0.8]]; yt=[0,1,1,0]; m=DecisionTreeClassifier(max_depth=2, random_state=0).fit(X,y); print(json.dumps(sum(a==b for a,b in zip(m.predict(xt),yt))/len(yt)))"
	pythonCommand := "python"
	if _, err := exec.LookPath(pythonCommand); err != nil {
		pythonCommand = "python3"
	}
	output, err := exec.Command(pythonCommand, "-c", python).Output()
	if err != nil {
		t.Skipf("python with scikit-learn unavailable: %v", err)
	}
	var sklearnAccuracy float64
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(output))), &sklearnAccuracy); err != nil {
		t.Fatal(err)
	}
	difference := goAccuracy - sklearnAccuracy
	t.Logf("accuracy: insyra=%.6f scikit-learn=%.6f difference=%+.6f", goAccuracy, sklearnAccuracy, difference)
}

func tableFromRows(rows [][]float64, names []string) *insyra.DataTable {
	columns := make([]*insyra.DataList, len(names))
	for column := range names {
		values := make([]any, len(rows))
		for row := range rows {
			values[row] = rows[row][column]
		}
		columns[column] = insyra.NewDataList(values...).SetName(names[column])
	}
	return insyra.NewDataTable(columns...)
}

func permuteTable(table *insyra.DataTable, order []int) *insyra.DataTable {
	columns := make([]*insyra.DataList, table.NumCols())
	for column := range columns {
		values := table.GetColByNumber(column).Data()
		permuted := make([]any, len(order))
		for i, row := range order {
			permuted[i] = values[row]
		}
		columns[column] = insyra.NewDataList(permuted...).SetName(table.ColNames()[column])
	}
	return insyra.NewDataTable(columns...)
}

func permuteList(list *insyra.DataList, order []int) *insyra.DataList {
	values := list.Data()
	permuted := make([]any, len(order))
	for i, row := range order {
		permuted[i] = values[row]
	}
	return insyra.NewDataList(permuted...)
}

func treeBounds(node *ml.DecisionTreeNode) (leaves, depth int) {
	if node == nil || node.IsLeaf {
		return 1, 0
	}
	leftLeaves, leftDepth := treeBounds(node.Left)
	rightLeaves, rightDepth := treeBounds(node.Right)
	return leftLeaves + rightLeaves, 1 + maxInt(leftDepth, rightDepth)
}

func checkMinSamples(t *testing.T, node *ml.DecisionTreeNode, minimum int) {
	t.Helper()
	if node == nil {
		return
	}
	if node.IsLeaf {
		if node.Samples < minimum {
			t.Fatalf("leaf has %d samples, want at least %d", node.Samples, minimum)
		}
		return
	}
	checkMinSamples(t, node.Left, minimum)
	checkMinSamples(t, node.Right, minimum)
}

func accuracy(predicted, actual []any) float64 {
	correct := 0
	for i := range predicted {
		if predicted[i] == actual[i] {
			correct++
		}
	}
	return float64(correct) / float64(len(actual))
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

var _ ml.Model = (*ml.DecisionTreeClassifier)(nil)
var _ ml.ProbaModel = (*ml.DecisionTreeClassifier)(nil)
var _ ml.Importances = (*ml.DecisionTreeClassifier)(nil)
var _ ml.Model = (*ml.DecisionTreeRegressor)(nil)
var _ ml.Importances = (*ml.DecisionTreeRegressor)(nil)
