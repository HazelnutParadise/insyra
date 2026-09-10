package ml

import (
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// Classes() returns an *insyra.DataList and no error, so Err() on what comes
// back is the only way a caller learns anything. A nil one panics on every
// method it has — Err() included — which makes the caller's first safe move
// impossible. These methods must hand back something usable.
//
// The zero values below stand in for a model that was never fitted; the
// constructors refuse bad input, so this is the state a caller reaches by
// declaring a model and using it too early.
func TestClassesNeverReturnsNil(t *testing.T) {
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)

	for name, classes := range map[string]func() *insyra.DataList{
		"DecisionTreeClassifier":     func() *insyra.DataList { return (&DecisionTreeClassifier{}).Classes() },
		"GradientBoostingClassifier": func() *insyra.DataList { return (&GradientBoostingClassifier{}).Classes() },
		"RandomForestClassifier":     func() *insyra.DataList { return (&RandomForestClassifier{}).Classes() },
		"LogisticModel":              func() *insyra.DataList { return (&LogisticModel{}).Classes() },
		"KNNClassifier":              func() *insyra.DataList { return (&KNNClassifier{}).Classes() },
		"nil DecisionTreeClassifier": func() *insyra.DataList {
			var m *DecisionTreeClassifier
			return m.Classes()
		},
		"nil RandomForestClassifier": func() *insyra.DataList {
			var m *RandomForestClassifier
			return m.Classes()
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := classes()
			if got == nil {
				t.Fatal("Classes() returned nil; a caller cannot even ask Err() why")
			}
			// Everything a caller might reach for must be safe.
			if n := got.Len(); n != 0 {
				t.Errorf("Len() = %d, want 0", n)
			}
			_ = got.Data()
			err := got.Err()
			if err == nil {
				t.Fatal("Err() is nil; the empty result says nothing about why it is empty")
			}
			if !strings.Contains(strings.ToLower(err.Error()), "fit") {
				t.Errorf("the reason should say the model is not fitted: %v", err)
			}
		})
	}
}

// A pipeline wrapper delegates to the model it holds; the same rule applies.
func TestPipelineClassesNeverReturnsNil(t *testing.T) {
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)

	for name, classes := range map[string]func() *insyra.DataList{
		"fittedPipelineClassifier": func() *insyra.DataList {
			return (&fittedPipelineClassifier{fittedPipeline: &fittedPipeline{}}).Classes()
		},
		"fittedPipelineProba": func() *insyra.DataList {
			return (&fittedPipelineProba{fittedPipeline: &fittedPipeline{}}).Classes()
		},
		"fittedPipelineClassifierImportances": func() *insyra.DataList {
			return (&fittedPipelineClassifierImportances{fittedPipeline: &fittedPipeline{}}).Classes()
		},
		"fittedPipelineProbaImportances": func() *insyra.DataList {
			return (&fittedPipelineProbaImportances{fittedPipeline: &fittedPipeline{}}).Classes()
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := classes()
			if got == nil {
				t.Fatal("Classes() returned nil")
			}
			if got.Err() == nil {
				t.Error("Err() is nil; the empty result says nothing about why it is empty")
			}
			_ = got.Data()
		})
	}
}

// A fitted model still reports its classes; the guard must not swallow them.
func TestFittedClassifierStillReportsClasses(t *testing.T) {
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)

	x := insyra.NewDataTable(
		insyra.NewDataList(1.0, 2.0, 3.0, 4.0).SetName("a"),
		insyra.NewDataList(1.0, 1.0, 2.0, 2.0).SetName("b"),
	)
	y := insyra.NewDataList("low", "low", "high", "high")
	model, err := FitDecisionTreeClassifier(x, y)
	if err != nil {
		t.Fatal(err)
	}
	got := model.Classes()
	if got == nil || got.Len() != 2 {
		t.Fatalf("Classes() = %v, want two labels", got)
	}
	if got.Err() != nil {
		t.Errorf("a fitted model must not record an error: %v", got.Err())
	}
}
