package py

import (
	"strings"
	"testing"
)

// SEC-20 of #297: PipInstall handed the caller's string to `uv pip install` as
// one argv, so "--requirement=/path" made uv read that file and install
// whatever it listed. A name is now checked before anything runs, and the
// command puts "--" ahead of it so uv cannot read it as an option either.
func TestCheckDependencyName(t *testing.T) {
	refused := []string{
		"--requirement=/etc/passwd",
		"-r/etc/passwd",
		"--index-url=http://example.invalid",
		"-e.",
		"-",
		"",
	}
	for _, dep := range refused {
		if err := checkDependencyName(dep); err == nil {
			t.Errorf("checkDependencyName(%q) accepted it", dep)
		}
	}

	accepted := []string{
		"numpy",
		"numpy==1.26.4",
		"pandas>=2",
		"scikit-learn",
		"git+https://example.invalid/pkg.git",
		"./local-package",
	}
	for _, dep := range accepted {
		if err := checkDependencyName(dep); err != nil {
			t.Errorf("checkDependencyName(%q) refused it: %v", dep, err)
		}
	}
}

// The refusal happens before the environment is set up, so a bad name cannot
// trigger a Python download either.
func TestPipInstall_RefusesAnOptionBeforeDoingAnything(t *testing.T) {
	err := PipInstall("--requirement=/etc/passwd")
	if err == nil {
		t.Fatal("PipInstall accepted an option as a dependency name")
	}
	if !strings.Contains(err.Error(), "PipInstall") {
		t.Errorf("the error %q does not name the function", err)
	}
	if !strings.Contains(err.Error(), "option") {
		t.Errorf("the error %q does not explain the problem", err)
	}
}
