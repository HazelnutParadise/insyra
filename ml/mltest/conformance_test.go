package mltest_test

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml/mltest"
)

// transposedProbaModel is a conforming model in every respect except one: its
// probability columns are named in Classes() order but carry each other's
// values. A model that builds column names from its own Classes() list passes
// a name-only ordering check by construction, so this is the case the check
// has to catch.
type transposedProbaModel struct{}

func (transposedProbaModel) Features() []string { return []string{"f1", "f2"} }

func (m transposedProbaModel) firstFeature(dt *insyra.DataTable) ([]float64, error) {
	for _, name := range m.Features() {
		if dt == nil || dt.GetColByName(name) == nil {
			return nil, fmt.Errorf("mltest: missing feature column %q", name)
		}
	}
	return dt.GetColByName("f1").ToF64Slice(), nil
}

func (m transposedProbaModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	f1, err := m.firstFeature(dt)
	if err != nil {
		return nil, err
	}
	labels := make([]any, len(f1))
	for i, value := range f1 {
		labels[i] = "a"
		if value >= 2 {
			labels[i] = "b"
		}
	}
	return insyra.NewDataList(labels...), nil
}

func (transposedProbaModel) Classes() *insyra.DataList {
	return insyra.NewDataList("a", "b").SetName("classes")
}

func (m transposedProbaModel) PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error) {
	f1, err := m.firstFeature(dt)
	if err != nil {
		return nil, err
	}
	columnA := make([]any, len(f1))
	columnB := make([]any, len(f1))
	for i, value := range f1 {
		probabilityA, probabilityB := 0.9, 0.1
		if value >= 2 {
			probabilityA, probabilityB = 0.1, 0.9
		}
		// Swapped on purpose: rows still sum to 1 and the names are in class
		// order, so every check other than the value-level one still passes.
		columnA[i] = probabilityB
		columnB[i] = probabilityA
	}
	return insyra.NewDataTable(
		insyra.NewDataList(columnA...).SetName("a"),
		insyra.NewDataList(columnB...).SetName("b"),
	), nil
}

func transposedTrainingTable() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(0.0, 1.0, 2.0, 3.0).SetName("f1"),
		insyra.NewDataList(1.0, 1.0, 1.0, 1.0).SetName("f2"),
	)
}

// RunConformance reports failure by calling t.Fatalf, so the only way to assert
// that it rejects a model is to run it in a child process and inspect the exit
// status. The child runs this same test function with the environment variable
// set.
const transposedChildEnv = "MLTEST_TRANSPOSED_CHILD"

func TestRunConformanceRejectsTransposedProbabilities(t *testing.T) {
	if os.Getenv(transposedChildEnv) == "1" {
		mltest.RunConformance(t, transposedProbaModel{}, transposedTrainingTable(), nil)
		return
	}

	command := exec.Command(os.Args[0], "-test.run=^TestRunConformanceRejectsTransposedProbabilities$", "-test.v")
	command.Env = append(os.Environ(), transposedChildEnv+"=1")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("RunConformance accepted a model whose probability columns are named in class order but hold each other's values; child output:\n%s", output)
	}
	if !strings.Contains(string(output), "Predict chose class") {
		t.Fatalf("RunConformance failed for the wrong reason; child output:\n%s", output)
	}
}

// renamedOnlyModel is a conforming model in every respect except one: when a
// fitted feature is absent it names the unmatched incoming column instead of
// the feature it could not find. The column the harness renames is a
// superstring of the feature it replaced, so a substring check against the raw
// error text cannot tell the two apart.
type renamedOnlyModel struct{}

func (renamedOnlyModel) Features() []string { return []string{"f1", "f2"} }

func (m renamedOnlyModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	known := make(map[string]struct{}, len(m.Features()))
	for _, name := range m.Features() {
		known[name] = struct{}{}
	}
	if dt == nil {
		return nil, fmt.Errorf("mltest: data table is nil")
	}
	for _, name := range m.Features() {
		if dt.GetColByName(name) != nil {
			continue
		}
		for _, incoming := range dt.ColNames() {
			if _, ok := known[incoming]; !ok {
				return nil, fmt.Errorf("mltest: column %q is not a known feature", incoming)
			}
		}
		return nil, fmt.Errorf("mltest: input does not match the fitted features")
	}
	values := dt.GetColByName("f1").ToF64Slice()
	predictions := make([]any, len(values))
	for i, value := range values {
		predictions[i] = value
	}
	return insyra.NewDataList(predictions...), nil
}

func renamedOnlyTrainingTable() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(0.0, 1.0, 2.0, 3.0).SetName("f1"),
		insyra.NewDataList(1.0, 1.0, 1.0, 1.0).SetName("f2"),
	)
}

const renamedOnlyChildEnv = "MLTEST_RENAMED_ONLY_CHILD"

func TestRunConformanceRejectsErrorNamingOnlyTheRenamedColumn(t *testing.T) {
	if os.Getenv(renamedOnlyChildEnv) == "1" {
		mltest.RunConformance(t, renamedOnlyModel{}, renamedOnlyTrainingTable(), nil)
		return
	}

	command := exec.Command(os.Args[0], "-test.run=^TestRunConformanceRejectsErrorNamingOnlyTheRenamedColumn$", "-test.v")
	command.Env = append(os.Environ(), renamedOnlyChildEnv+"=1")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("RunConformance accepted a model that names only the renamed incoming column, never the missing feature; child output:\n%s", output)
	}
	if !strings.Contains(string(output), "want an error naming") {
		t.Fatalf("RunConformance failed for the wrong reason; child output:\n%s", output)
	}
}
