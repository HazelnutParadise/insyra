package py

import (
	"strings"
	"testing"
)

func TestBuiltInPreservesExplicitDataFrameNameWithNameColumn(t *testing.T) {
	source := builtInFunc("ipc", "run")
	want := `result.__dict__.get("name") if "name" in result.columns else getattr(result, "name", None)`
	if !strings.Contains(source, want) {
		t.Fatalf("generated pandas normalization does not preserve an explicit DataFrame name beside a 'name' column")
	}
}
