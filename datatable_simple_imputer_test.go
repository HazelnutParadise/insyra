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

	imputer := NewSimpleImputer(SimpleImputerOptions{Strategy: ImputeMean})
	if err := imputer.Fit(training, Name("value")); err != nil {
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
		{name: "mean", strategy: ImputeMean, inPlace: func(dt *DataTable) { dt.FillWithMean(Name("value")) }},
		{name: "median", strategy: ImputeMedian, inPlace: func(dt *DataTable) { dt.FillWithMedian(Name("value")) }},
		{name: "mode", strategy: ImputeMode, inPlace: func(dt *DataTable) { dt.FillWithMode(Name("value")) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := NewDataTable(NewDataList(1.0, nil, 3.0, 3.0).SetName("value"))
			want := input.Clone()
			test.inPlace(want)

			got, err := NewSimpleImputer(SimpleImputerOptions{Strategy: test.strategy}).FitTransform(input, Name("value"))
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

	imputer := NewSimpleImputer(SimpleImputerOptions{Strategy: ImputeMean})
	if err := imputer.Fit(training, Name("mean")); err != nil {
		t.Fatal(err)
	}
	params := imputer.Params()
	if params["mean"].Replacement != 3.0 {
		t.Fatalf("mean params = %#v", params["mean"])
	}
	if imputer.Kind() != "imputer-mean" {
		t.Fatalf("kind = %q, want %q", imputer.Kind(), "imputer-mean")
	}

	constant := NewSimpleImputer(SimpleImputerOptions{Strategy: ImputeConstant, FillValue: 7.0})
	constantInput := NewDataTable(NewDataList(10.0, nil, 20.0).SetName("constant"))
	if err := constant.Fit(constantInput, Name("constant")); err != nil {
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

func TestSimpleImputerErrorsOnColumnsItCannotFill(t *testing.T) {
	allMissing := NewDataTable(NewDataList(nil, math.NaN()).SetName("empty"))
	if err := NewSimpleImputer(SimpleImputerOptions{Strategy: ImputeMedian}).Fit(allMissing, Name("empty")); err == nil || !strings.Contains(err.Error(), `column "empty"`) {
		t.Fatalf("all-missing error = %v", err)
	}

	// A column the caller selected and the strategy cannot fill is an error,
	// as it is for the table fills: the owner ruled on 2026-09-26 (#213) that
	// the old pass-through, reported only through Params, let a fit look
	// complete when a selected column was never going to be filled.
	train := NewDataTable(
		NewDataList(10.0, nil, 30.0).SetName("income"),
		NewDataList("red", nil, "blue").SetName("color"),
	)
	for _, strategy := range []ImputationStrategy{ImputeMean, ImputeMedian} {
		imputer := NewSimpleImputer(SimpleImputerOptions{Strategy: strategy})
		err := imputer.Fit(train, Name("income"), Name("color"))
		if err == nil || !strings.Contains(err.Error(), `"color"`) || !strings.Contains(err.Error(), "string") {
			t.Fatalf("%s over a text column: got %v, want an error naming color and its string values", strategy, err)
		}
		if _, err := imputer.Transform(train); err == nil {
			t.Fatalf("%s: a failed fit left the imputer usable", strategy)
		}
	}

	// Mode fills any kind of value, so a text column is fine there.
	mode := NewSimpleImputer(SimpleImputerOptions{Strategy: ImputeMode})
	got, err := mode.FitTransform(train, Name("color"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.GetColByName("color").Data(), []any{"red", "red", "blue"}) {
		t.Fatalf("mode over text = %#v", got.GetColByName("color").Data())
	}
	if !reflect.DeepEqual(got.GetColByName("income").Data(), []any{10.0, nil, 30.0}) {
		t.Fatalf("an unselected column changed: %#v", got.GetColByName("income").Data())
	}
}

func TestSimpleImputerSupportsScalerAndPipelineTransformer(t *testing.T) {
	var _ interface {
		Transform(*DataTable) (*DataTable, error)
	} = (*SimpleImputer)(nil)
}

// TestSimpleImputerDoesNotClaimReversibility pins the reason SimpleImputer is
// not a Scaler. An always-erroring InverseTransform would satisfy every
// interface that asks for the method, so a caller probing for the capability by
// type assertion would be told it is present and then refused at the call.
func TestSimpleImputerDoesNotClaimReversibility(t *testing.T) {
	var imputer any = NewSimpleImputer(SimpleImputerOptions{Strategy: ImputeMean})

	if _, ok := imputer.(interface {
		InverseTransform(dt *DataTable) (*DataTable, error)
	}); ok {
		t.Fatal("SimpleImputer must not carry InverseTransform: a type assertion would report a capability it cannot honour")
	}
	if _, ok := imputer.(Scaler); ok {
		t.Fatal("SimpleImputer must not satisfy Scaler, which requires InverseTransform")
	}
	if _, ok := imputer.(interface {
		Transform(dt *DataTable) (*DataTable, error)
	}); !ok {
		t.Fatal("SimpleImputer must satisfy the transformer shape a pipeline needs")
	}
}
