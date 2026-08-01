package mltest_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
	"github.com/HazelnutParadise/insyra/ml/mltest"
)

// misalignedImportances wraps a real fitted model and reports one importance
// too many. Everything else — feature matching, the renamed-column refusal,
// reorder and extra-column invariance — is the real model's behaviour, so this
// probe can only fail on the count. An earlier version used a hand-written
// model and passed with the check disabled, because it was being caught by the
// renamed-column assertion instead.
type misalignedImportances struct{ ml.Model }

func (m misalignedImportances) FeatureImportances() []float64 {
	return make([]float64, len(m.Model.Features())+1)
}

var _ ml.Importances = misalignedImportances{}

func fitTreeForAlignment(t *testing.T) (ml.Model, *insyra.DataTable) {
	t.Helper()
	const n = 40
	a := make([]any, n)
	b := make([]any, n)
	y := make([]any, n)
	for i := 0; i < n; i++ {
		a[i] = float64(i % 7)
		b[i] = float64(i % 5)
		y[i] = 3*float64(i%7) + 2*float64(i%5)
	}
	x := insyra.NewDataTable(
		insyra.NewDataList(a...).SetName("a"),
		insyra.NewDataList(b...).SetName("b"),
	)
	model, err := ml.FitDecisionTreeRegressor(x, insyra.NewDataList(y...))
	if err != nil {
		t.Fatalf("fit: %v", err)
	}
	return model, x
}

func TestConformanceAcceptsAlignedImportances(t *testing.T) {
	model, x := fitTreeForAlignment(t)
	mltest.RunConformance(t, model, x, nil)
}

func TestConformanceCatchesMisalignedImportances(t *testing.T) {
	model, x := fitTreeForAlignment(t)

	// t.Fatalf calls runtime.Goexit, which unwinds the calling goroutine — so
	// the fake check has to run in one of its own or it takes the test with it.
	fake := &testing.T{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		mltest.RunConformance(fake, misalignedImportances{Model: model}, x, nil)
	}()
	<-done
	if !fake.Failed() {
		t.Fatal("a model reporting one importance too many passed conformance")
	}
}
