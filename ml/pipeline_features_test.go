package ml_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
)

// expandingTransformer replaces one column with several, the way a one-hot
// encoder does. It is what makes the pipeline's input names and its estimator's
// column count disagree.
type expandingTransformer struct{ into int }

func (e expandingTransformer) Transform(dt *insyra.DataTable) (*insyra.DataTable, error) {
	source := dt.GetColByNumber(0)
	name := source.GetName()
	columns := make([]*insyra.DataList, 0, e.into+dt.NumCols()-1)
	for part := 0; part < e.into; part++ {
		values := make([]any, dt.NumRows())
		for row := range values {
			value, _ := insyra.ToFloat64Safe(source.Get(row))
			values[row] = value + float64(part)
		}
		columns = append(columns, insyra.NewDataList(values...).SetName(fmt.Sprintf("%s_%d", name, part)))
	}
	for col := 1; col < dt.NumCols(); col++ {
		columns = append(columns, dt.GetColByNumber(col))
	}
	return insyra.NewDataTable(columns...), nil
}

func TestPipelineReportsTheColumnsItsEstimatorSaw(t *testing.T) {
	const n = 40
	first := make([]any, n)
	second := make([]any, n)
	y := make([]any, n)
	for i := 0; i < n; i++ {
		first[i] = float64(i%7) + 1
		second[i] = float64(i%5) + 2
		y[i] = 3*(float64(i%7)+1) + 2*(float64(i%5)+2)
	}
	x := insyra.NewDataTable(
		insyra.NewDataList(first...).SetName("num"),
		insyra.NewDataList(second...).SetName("cat"),
	)
	target := insyra.NewDataList(y...).SetName("y")

	pipeline := ml.NewPipeline(
		[]ml.Step{{Name: "expand", Fit: func(*insyra.DataTable, *insyra.DataList) (ml.Transformer, error) {
			return expandingTransformer{into: 3}, nil
		}}},
		ml.Estimator{Name: "tree", Fit: func(x *insyra.DataTable, y *insyra.DataList) (ml.Model, error) {
			return ml.FitDecisionTreeRegressor(x, y)
		}},
	)
	fitted, err := pipeline.Fit(x, target)
	if err != nil {
		t.Fatalf("fit: %v", err)
	}

	expanded, ok := fitted.(ml.TransformedFeatures)
	if !ok {
		t.Fatalf("%T does not report the columns its estimator saw", fitted)
	}
	transformed := expanded.TransformedFeatureNames()
	if want := []string{"num_0", "num_1", "num_2", "cat"}; !reflect.DeepEqual(transformed, want) {
		t.Fatalf("estimator saw %v, want %v", transformed, want)
	}
	if input := fitted.Features(); len(input) == len(transformed) {
		t.Fatalf("input names %v and transformed names %v have the same length; the test is not exercising an expansion", input, transformed)
	}

	// The whole point: importances line up with the transformed names.
	importances, ok := fitted.(ml.Importances)
	if !ok {
		t.Fatal("the pipeline lost the estimator's importances")
	}
	if got := len(importances.FeatureImportances()); got != len(transformed) {
		t.Fatalf("%d importances against %d transformed names", got, len(transformed))
	}
}

func TestPipelineWithNoStepsReportsItsOwnFeatures(t *testing.T) {
	x, y := directionRegressionData()
	pipeline := ml.NewPipeline(nil, ml.Estimator{
		Name: "linear",
		Fit: func(x *insyra.DataTable, y *insyra.DataList) (ml.Model, error) {
			return ml.FitLinearRegression(x, y)
		},
	})
	fitted, err := pipeline.Fit(x, y)
	if err != nil {
		t.Fatalf("fit: %v", err)
	}
	expanded, ok := fitted.(ml.TransformedFeatures)
	if !ok {
		t.Fatalf("%T does not report the columns its estimator saw", fitted)
	}
	if got, want := expanded.TransformedFeatureNames(), fitted.Features(); !reflect.DeepEqual(got, want) {
		t.Fatalf("a pipeline with no steps reports %v, want its own features %v", got, want)
	}
}

// identityTransformer passes its input through. The step's Fit is where the
// property under test lives — which rows it was handed — so the transform
// itself does not need to do anything.
type identityTransformer struct{}

func (identityTransformer) Transform(dt *insyra.DataTable) (*insyra.DataTable, error) {
	return dt, nil
}

// Leakage is a property of which rows a step is fitted on, so the test asserts
// that directly rather than through a summary statistic. An earlier version
// compared each fold's training mean against the whole dataset's; on evenly
// spaced values a 30-row subset hits that mean by coincidence, which made the
// test fail on correct behaviour.
func TestPipelineInCrossValidationFitsOnTrainingRowsOnly(t *testing.T) {
	const n = 40
	const k = 4
	xs := make([]any, n)
	ys := make([]any, n)
	for i := 0; i < n; i++ {
		xs[i] = float64(i)
		ys[i] = 2*float64(i) + 1
	}
	x := insyra.NewDataTable(insyra.NewDataList(xs...).SetName("x"))
	y := insyra.NewDataList(ys...).SetName("y")

	var seenPerFold []map[float64]struct{}
	pipeline := ml.NewPipeline(
		[]ml.Step{{Name: "observe", Fit: func(train *insyra.DataTable, _ *insyra.DataList) (ml.Transformer, error) {
			seen := make(map[float64]struct{}, train.NumRows())
			column := train.GetColByNumber(0)
			for row := 0; row < train.NumRows(); row++ {
				value, _ := insyra.ToFloat64Safe(column.Get(row))
				seen[value] = struct{}{}
			}
			seenPerFold = append(seenPerFold, seen)
			return identityTransformer{}, nil
		}}},
		ml.Estimator{Name: "linear", Fit: func(x *insyra.DataTable, y *insyra.DataList) (ml.Model, error) {
			return ml.FitLinearRegression(x, y)
		}},
	)

	if _, err := ml.CrossValidate(x, y, pipeline, k, ml.RMSEMetric{},
		insyra.SamplingOptions{UseSeed: true, Seed: 3}); err != nil {
		t.Fatalf("cross-validate: %v", err)
	}
	if len(seenPerFold) != k {
		t.Fatalf("the step was fitted %d times over %d folds", len(seenPerFold), k)
	}

	// Each fold must have withheld exactly its own share, and the withheld
	// sets must partition the dataset — which is only true if every step saw
	// its own training rows and nothing else.
	withheldTotal := 0
	covered := make(map[float64]int, n)
	for fold, seen := range seenPerFold {
		if len(seen) != n-n/k {
			t.Fatalf("fold %d was fitted on %d distinct rows, want %d", fold+1, len(seen), n-n/k)
		}
		for i := 0; i < n; i++ {
			value := float64(i)
			if _, ok := seen[value]; !ok {
				withheldTotal++
				covered[value]++
			}
		}
	}
	if withheldTotal != n {
		t.Fatalf("%d rows were withheld across %d folds, want %d", withheldTotal, k, n)
	}
	for i := 0; i < n; i++ {
		if covered[float64(i)] != 1 {
			t.Fatalf("row %d was withheld from %d folds, want exactly 1", i, covered[float64(i)])
		}
	}
}
