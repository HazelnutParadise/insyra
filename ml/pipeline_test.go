package ml_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
)

func TestPipelineMatchesManualOrder(t *testing.T) {
	input := namedTable(
		pipelineDataList([]any{1.0, 2.0, 4.0}, "x"),
	)
	y := pipelineDataList([]any{0.0, 0.0, 0.0}, "y")
	steps := []ml.Step{
		{Name: "first", Fit: func(*insyra.DataTable, *insyra.DataList) (ml.Transformer, error) {
			return offsetTransformer{amount: 1}, nil
		}},
		{Name: "second", Fit: func(*insyra.DataTable, *insyra.DataList) (ml.Transformer, error) {
			return offsetTransformer{amount: 2}, nil
		}},
	}
	estimator := ml.Estimator{
		Name: "capture",
		Fit: func(x *insyra.DataTable, _ *insyra.DataList) (ml.Model, error) {
			return &firstValueModel{features: x.ColNames(), value: toFloat(t, x.GetColByNumber(0).Get(0))}, nil
		},
	}

	pipeline := ml.NewPipeline(steps, estimator)
	fitted, err := pipeline.Fit(input, y)
	if err != nil {
		t.Fatalf("fit pipeline: %v", err)
	}

	manual := input
	for _, step := range steps {
		transformer, err := step.Fit(manual, y)
		if err != nil {
			t.Fatal(err)
		}
		manual, err = transformer.Transform(manual)
		if err != nil {
			t.Fatal(err)
		}
	}
	manualModel, err := estimator.Fit(manual, y)
	if err != nil {
		t.Fatal(err)
	}
	want, err := manualModel.Predict(input)
	if err != nil {
		t.Fatal(err)
	}
	got, err := fitted.Predict(input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Data(), want.Data()) {
		t.Fatalf("pipeline prediction = %v, manual prediction = %v", got.Data(), want.Data())
	}
	if !reflect.DeepEqual(fitted.Features(), []string{"x"}) {
		t.Fatalf("pipeline features = %v", fitted.Features())
	}
}

func TestPipelineRefitsEveryStepAndEstimator(t *testing.T) {
	step := ml.Step{
		Name: "center",
		Fit: func(x *insyra.DataTable, _ *insyra.DataList) (ml.Transformer, error) {
			values := x.GetColByNumber(0).ToF64Slice()
			mean := 0.0
			for _, value := range values {
				mean += value
			}
			mean /= float64(len(values))
			return offsetTransformer{amount: -mean}, nil
		},
	}
	estimator := ml.Estimator{
		Name: "capture",
		Fit: func(x *insyra.DataTable, _ *insyra.DataList) (ml.Model, error) {
			return &firstValueModel{features: x.ColNames(), value: toFloat(t, x.GetColByNumber(0).Get(0))}, nil
		},
	}
	pipeline := ml.NewPipeline([]ml.Step{step}, estimator)

	first, err := pipeline.Fit(namedTable(pipelineDataList([]any{1.0, 3.0}, "x")), nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := pipeline.Fit(namedTable(pipelineDataList([]any{10.0, 30.0}, "x")), nil)
	if err != nil {
		t.Fatal(err)
	}
	firstPrediction, err := first.Predict(namedTable(pipelineDataList([]any{1.0}, "x")))
	if err != nil {
		t.Fatal(err)
	}
	secondPrediction, err := second.Predict(namedTable(pipelineDataList([]any{10.0}, "x")))
	if err != nil {
		t.Fatal(err)
	}
	if got := firstPrediction.Get(0); got != -1.0 {
		t.Fatalf("first fit value = %v, want -1", got)
	}
	if got := secondPrediction.Get(0); got != -10.0 {
		t.Fatalf("second fit value = %v, want -10", got)
	}
}

func TestPipelineNamesTheFailingStep(t *testing.T) {
	pipeline := ml.NewPipeline([]ml.Step{{
		Name: "broken encoder",
		Fit: func(*insyra.DataTable, *insyra.DataList) (ml.Transformer, error) {
			return nil, fmt.Errorf("bad categories")
		},
	}}, ml.Estimator{
		Name: "model",
		Fit: func(*insyra.DataTable, *insyra.DataList) (ml.Model, error) {
			return nil, fmt.Errorf("should not run")
		},
	})
	_, err := pipeline.Fit(namedTable(dataList([]any{1.0}, "x")), nil)
	if err == nil || !strings.Contains(err.Error(), `pipeline step "broken encoder"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestColumnTransformerScopesAndPreservesOrder(t *testing.T) {
	input := namedTable(
		pipelineDataList([]any{1.0, 2.0}, "left"),
		pipelineDataList([]any{10.0, 20.0}, "middle"),
		pipelineDataList([]any{"a", "b"}, "right"),
	)
	scoped := ml.NewColumnTransformer(offsetTransformer{amount: 5}, "middle")
	got, err := scoped.Transform(input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.ColNames(), []string{"left", "middle", "right"}) {
		t.Fatalf("column order = %v", got.ColNames())
	}
	if !reflect.DeepEqual(got.To2DSlice(), [][]any{{1.0, 15.0, "a"}, {2.0, 25.0, "b"}}) {
		t.Fatalf("scoped transform = %v", got.To2DSlice())
	}
	if _, err := ml.NewColumnTransformer(offsetTransformer{amount: 1}, "missing").Transform(input); err == nil || !strings.Contains(err.Error(), `"missing"`) {
		t.Fatalf("missing column error = %v", err)
	}
}

func TestPipelineMixesRootScalersAndEncoders(t *testing.T) {
	input := namedTable(
		pipelineDataList([]any{1.0, 2.0, 3.0}, "number"),
		pipelineDataList([]any{"red", "blue", "red"}, "color"),
	)
	pipeline := ml.NewPipeline([]ml.Step{
		{Name: "numeric scale", Fit: func(x *insyra.DataTable, _ *insyra.DataList) (ml.Transformer, error) {
			scaler := insyra.NewStandardScaler()
			if err := scaler.Fit(x, "number"); err != nil {
				return nil, err
			}
			return scaler, nil
		}},
		{Name: "categorical encode", Fit: func(x *insyra.DataTable, _ *insyra.DataList) (ml.Transformer, error) {
			_, encoder, err := x.OneHotEncode(insyra.OneHotOptions{Columns: []string{"color"}, SortCategories: true})
			return encoder, err
		}},
	}, ml.Estimator{
		Name: "capture",
		Fit: func(x *insyra.DataTable, _ *insyra.DataList) (ml.Model, error) {
			return &firstValueModel{features: x.ColNames(), value: toFloat(t, x.GetColByNumber(0).Get(0))}, nil
		},
	})

	fitted, err := pipeline.Fit(input, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fitted.Features(), []string{"number", "color"}) {
		t.Fatalf("raw pipeline features = %v", fitted.Features())
	}
	if _, ok := fitted.(ml.Model); !ok {
		t.Fatal("fitted pipeline does not satisfy ml.Model")
	}
	if _, err := fitted.Predict(input); err != nil {
		t.Fatalf("predict with root scaler and encoder: %v", err)
	}
}

type offsetTransformer struct {
	amount float64
}

func (t offsetTransformer) Transform(dt *insyra.DataTable) (*insyra.DataTable, error) {
	if dt == nil {
		return nil, fmt.Errorf("table is nil")
	}
	names := dt.ColNames()
	columns := make([]*insyra.DataList, len(names))
	for i, name := range names {
		values := dt.GetColByNumber(i).Data()
		out := make([]any, len(values))
		for row, value := range values {
			number, ok := insyra.ToFloat64Safe(value)
			if !ok {
				return nil, fmt.Errorf("column %q is not numeric", name)
			}
			out[row] = number + t.amount
		}
		columns[i] = insyra.NewDataList(out...).SetName(name)
	}
	return insyra.NewDataTable(columns...), nil
}

type firstValueModel struct {
	features []string
	value    float64
}

func (m *firstValueModel) Features() []string { return append([]string(nil), m.features...) }

func (m *firstValueModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	if dt == nil {
		return nil, fmt.Errorf("table is nil")
	}
	values := make([]any, dt.NumRows())
	for i := range values {
		values[i] = m.value
	}
	return insyra.NewDataList(values...), nil
}

func namedTable(columns ...*insyra.DataList) *insyra.DataTable {
	return insyra.NewDataTable(columns...)
}

func pipelineDataList(values []any, name string) *insyra.DataList {
	return insyra.NewDataList(values...).SetName(name)
}

func toFloat(t *testing.T, value any) float64 {
	t.Helper()
	converted, ok := insyra.ToFloat64Safe(value)
	if !ok {
		t.Fatalf("value %v is not numeric", value)
	}
	return converted
}
