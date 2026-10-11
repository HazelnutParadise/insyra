package py

import (
	"bytes"
	"os/exec"
	"slices"
	"testing"

	"github.com/HazelnutParadise/insyra"
	json "github.com/goccy/go-json"
)

// pythonWithPandas returns a Python on PATH that can import pandas, and skips
// the test when there is none.
func pythonWithPandas(t *testing.T) string {
	t.Helper()
	for _, name := range []string{"python3", "python"} {
		python, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		if exec.Command(python, "-c", "import pandas").Run() == nil {
			return python
		}
	}
	t.Skip("no Python on PATH that can import pandas")
	return ""
}

// insyra.Return read a pandas DataFrame's name with getattr, and pandas
// returns a column as an attribute, so a DataFrame with a column called name
// came back as a table named after that column's printed contents. A name set
// with df.name still comes through. The insyra class the runners generate is
// run here with a real pandas, and what _normalize_result makes of each
// DataFrame is decoded as a result would be.
func TestADataFrameColumnCalledNameIsNotItsName(t *testing.T) {
	python := pythonWithPandas(t)
	script := "import json\nimport pandas as pd\n" + builtInFunc("unused", "0") + `
name_column = pd.DataFrame({"name": ["x", "y"], "v": [1, 2]})
name_set = pd.DataFrame({"v": [1, 2]})
name_set.name = "scores"
print(json.dumps([insyra._normalize_result(name_column), insyra._normalize_result(name_set)]))
`
	cmd := exec.Command(python, "-c", script)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("python failed: %v\n%s", err, stderr.String())
	}
	var payloads []any
	if err := json.Unmarshal(out, &payloads); err != nil || len(payloads) != 2 {
		t.Fatalf("python printed %s (%v)", out, err)
	}

	var nameColumn *insyra.DataTable
	if err := bindPyResult(&nameColumn, payloads[0]); err != nil {
		t.Fatal(err)
	}
	if names := nameColumn.ColNames(); nameColumn.GetName() != "" || !slices.Equal(names, []string{"name", "v"}) {
		t.Errorf("a DataFrame with a column called name came back named %q with columns %q", nameColumn.GetName(), names)
	}

	var nameSet *insyra.DataTable
	if err := bindPyResult(&nameSet, payloads[1]); err != nil {
		t.Fatal(err)
	}
	if nameSet.GetName() != "scores" {
		t.Errorf("a DataFrame named scores came back named %q", nameSet.GetName())
	}
}

// A DataFrame with a column called name whose name was also set explicitly:
// pandas returns the column for the name attribute, so the explicitly set
// name lives only in the instance's __dict__ and is the one that comes back.
func TestAnExplicitDataFrameNameSurvivesANameColumn(t *testing.T) {
	python := pythonWithPandas(t)
	script := "import json\nimport pandas as pd\n" + builtInFunc("unused", "0") + `
explicit = pd.DataFrame({"name": ["x", "y"], "v": [1, 2]})
object.__setattr__(explicit, "name", "scores")
print(json.dumps(insyra._normalize_result(explicit)))
`
	cmd := exec.Command(python, "-c", script)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("python failed: %v\n%s", err, stderr.String())
	}
	var payload any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("python printed %s (%v)", out, err)
	}

	var explicit *insyra.DataTable
	if err := bindPyResult(&explicit, payload); err != nil {
		t.Fatal(err)
	}
	if explicit.GetName() != "scores" {
		t.Errorf("a DataFrame named scores explicitly came back named %q", explicit.GetName())
	}
}
