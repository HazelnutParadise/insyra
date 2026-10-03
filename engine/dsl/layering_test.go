package dsl

import (
	"os/exec"
	"strings"
	"testing"
)

// #260 (EN-2): engine/dsl is the library's way into the command language, so
// nothing it builds on may come from the CLI: not cli/, and not the shell and
// line-editing libraries the CLI is made of.
func TestEngineDSLDoesNotDependOnTheCLI(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	forbidden := []string{
		"github.com/HazelnutParadise/insyra/cli",
		"github.com/spf13/cobra",
		"github.com/ergochat/readline",
	}
	deps := strings.Fields(string(out))
	if len(deps) == 0 {
		t.Fatal("go list -deps printed nothing")
	}
	for _, dep := range deps {
		for _, prefix := range forbidden {
			if dep == prefix || strings.HasPrefix(dep, prefix+"/") {
				t.Errorf("engine/dsl depends on %s", dep)
			}
		}
	}
}
