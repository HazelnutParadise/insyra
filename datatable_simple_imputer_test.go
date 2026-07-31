package insyra

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestSimpleImputerUsesTrainingReplacementOnNewTable(t *testing.T) {
	training := NewDataTable(
		NewDataList(1.0, nil, 3.0).SetName("value"),
	)
	validation := NewDataTable(
		NewDataList(100.0, math.NaN(), 200.0).SetName("value"),
	)

	imputer := NewSimpleImputer(ImputeMean)
	if err := imputer.Fit(training, "value"); err != nil {
		t.Fatal(err)
	}
	got, err := imputer.Transform(validation)
	if err != nil {
		t.Fatal(err)
	}
	if want := []any{100.0, 2.0, 200.0}; !reflect.DeepEqual(got.GetColByName("value").Data(), want) {
		t.Fatalf("transformed values = %#v, want %#v", got.GetColByName("value").Data(), want)
	}
}

func TestSimpleImputerFitTransformMatchesInPlaceMethods(t *testing.T) {
	tests := []struct {
		name     string
		strategy ImputationStrategy
		inPlace  func(*DataTable)
	}{
		{name: "mean", strategy: ImputeMean, inPlace: func(dt *DataTable) { dt.FillWithMean("value") }},
		{name: "median", strategy: ImputeMedian, inPlace: func(dt *DataTable) { dt.FillWithMedian("value") }},
		{name: "mode", strategy: ImputeMode, inPlace: func(dt *DataTable) { dt.FillWithMode("value") }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := NewDataTable(NewDataList(1.0, nil, 3.0, 3.0).SetName("value"))
			want := input.Clone()
			test.inPlace(want)

			got, err := NewSimpleImputer(test.strategy).FitTransform(input, "value")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.GetColByName("value").Data(), want.GetColByName("value").Data()) {
				t.Fatalf("values = %#v, want %#v", got.GetColByName("value").Data(), want.GetColByName("value").Data())
			}
		})
	}
}

func TestSimpleImputerParamsAndStrategies(t *testing.T) {
	training := NewDataTable(
		NewDataList(1.0, nil, 5.0).SetName("mean"),
		NewDataList("red", nil, "red").SetName("color"),
		NewDataList(10.0, nil, 20.0).SetName("constant"),
	)

	imputer := NewSimpleImputer(ImputeMean)
	if err := imputer.Fit(training, "mean"); err != nil {
		t.Fatal(err)
	}
	params := imputer.Params()
	if params["mean"].Replacement != 3.0 || params["mean"].PassThrough {
		t.Fatalf("mean params = %#v", params["mean"])
	}
	if imputer.Kind() != "imputer-mean" {
		t.Fatalf("kind = %q, want %q", imputer.Kind(), "imputer-mean")
	}

	constant := NewSimpleImputer(ImputeConstant, 7.0)
	constantInput := NewDataTable(NewDataList(10.0, nil, 20.0).SetName("constant"))
	if err := constant.Fit(constantInput, "constant"); err != nil {
		t.Fatal(err)
	}
	if got := constant.Params()["constant"].Replacement; got != 7.0 {
		t.Fatalf("constant replacement = %#v, want 7", got)
	}
	constantOutput, err := constant.Transform(constantInput)
	if err != nil {
		t.Fatal(err)
	}
	if want := []any{10.0, 7.0, 20.0}; !reflect.DeepEqual(constantOutput.GetColByName("constant").Data(), want) {
		t.Fatalf("constant values = %#v, want %#v", constantOutput.GetColByName("constant").Data(), want)
	}
}

func TestSimpleImputerErrorsAndPassThrough(t *testing.T) {
	allMissing := NewDataTable(NewDataList(nil, math.NaN()).SetName("empty"))
	if err := NewSimpleImputer(ImputeMedian).Fit(allMissing, "empty"); err == nil || !strings.Contains(err.Error(), `column "empty"`) {
		t.Fatalf("all-missing error = %v", err)
	}

	text := NewDataTable(NewDataList("red", nil, "blue").SetName("color"))
	imputer := NewSimpleImputer(ImputeMean)
	if err := imputer.Fit(text, "color"); err != nil {
		t.Fatal(err)
	}
	if !imputer.Params()["color"].PassThrough {
		t.Fatal("numeric strategy should mark a mixed column as pass-through")
	}
	got, err := imputer.Transform(NewDataTable(
		NewDataList("green", nil, "yellow").SetName("color"),
		NewDataList(1, 2, 3).SetName("untouched"),
	))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.GetColByName("color").Data(), []any{"green", nil, "yellow"}) {
		t.Fatalf("pass-through values = %#v", got.GetColByName("color").Data())
	}
	if !reflect.DeepEqual(got.GetColByName("untouched").Data(), []any{1, 2, 3}) {
		t.Fatalf("unfitted values = %#v", got.GetColByName("untouched").Data())
	}
}

func TestSimpleImputerSupportsScalerAndPipelineTransformer(t *testing.T) {
	var _ Scaler = (*SimpleImputer)(nil)
	var _ interface {
		Transform(*DataTable) (*DataTable, error)
	} = (*SimpleImputer)(nil)
}
