package ml_test

import (
	"bytes"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
	"github.com/HazelnutParadise/insyra/preprocess"
)

// The preprocess types replace the core ones, so a pipeline built from them
// must export the same graph, byte for byte, as the pipeline in onnxPipeline.
func TestONNXExportOfPreprocessTypesMatchesTheCoreTypes(t *testing.T) {
	input := onnxPipelineInput()
	targets := onnxRegressionTargets()
	core, err := onnxPipeline().Fit(input, targets)
	if err != nil {
		t.Fatal(err)
	}
	moved, err := ml.NewPipeline([]ml.Step{
		{Name: "scale", Fit: func(x *insyra.DataTable, _ *insyra.DataList) (ml.Transformer, error) {
			scaler := preprocess.NewStandardScaler()
			if err := scaler.Fit(x, insyra.Name("x1")); err != nil {
				return nil, err
			}
			return ml.NewColumnTransformer(scaler, "x1"), nil
		}},
		{Name: "encode", Fit: func(x *insyra.DataTable, _ *insyra.DataList) (ml.Transformer, error) {
			_, encoder, err := preprocess.OneHotEncode(x, preprocess.OneHotOptions{Columns: []any{insyra.Name("color")}, DropFirst: true, SortCategories: true})
			if err != nil {
				return nil, err
			}
			return ml.NewColumnTransformer(encoder, "color"), nil
		}},
	}, ml.Estimator{Name: "linear", Fit: ml.FitLinearRegression}).Fit(input, targets)
	if err != nil {
		t.Fatal(err)
	}
	var want, got bytes.Buffer
	if err := ml.ExportONNX(&want, core); err != nil {
		t.Fatal(err)
	}
	if err := ml.ExportONNX(&got, moved); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Bytes(), want.Bytes()) {
		t.Fatalf("preprocess pipeline exported %d bytes that differ from the core pipeline's %d", got.Len(), want.Len())
	}
}

func TestONNXExportOfPreprocessOrdinalEncoderMatchesTheCoreOne(t *testing.T) {
	input := onnxPipelineInput()
	targets := onnxRegressionTargets()
	build := func(fit func(x *insyra.DataTable) (ml.Transformer, error)) ml.Model {
		t.Helper()
		model, err := ml.NewPipeline([]ml.Step{
			{Name: "encode", Fit: func(x *insyra.DataTable, _ *insyra.DataList) (ml.Transformer, error) { return fit(x) }},
		}, ml.Estimator{Name: "linear", Fit: ml.FitLinearRegression}).Fit(input, targets)
		if err != nil {
			t.Fatal(err)
		}
		return model
	}
	core := build(func(x *insyra.DataTable) (ml.Transformer, error) {
		_, encoder, err := x.OrdinalEncode(insyra.OrdinalEncodeOptions{Column: "color", Order: []any{"red", "green", "blue"}}) //nolint:staticcheck // the core encoder is the reference the moved one must match
		return encoder, err
	})
	moved := build(func(x *insyra.DataTable) (ml.Transformer, error) {
		_, encoder, err := preprocess.OrdinalEncode(x, preprocess.OrdinalOptions{Column: insyra.Name("color"), Order: []any{"red", "green", "blue"}})
		return encoder, err
	})
	var want, got bytes.Buffer
	if err := ml.ExportONNX(&want, core); err != nil {
		t.Fatal(err)
	}
	if err := ml.ExportONNX(&got, moved); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Bytes(), want.Bytes()) {
		t.Fatalf("preprocess ordinal encoder exported %d bytes that differ from the core encoder's %d", got.Len(), want.Len())
	}
}
