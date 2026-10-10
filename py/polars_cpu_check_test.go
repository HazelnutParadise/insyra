package py

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// pythonInterpreterOnPath returns the Python to run the preamble with, or "".
func pythonInterpreterOnPath() string {
	for _, name := range []string{"python3", "python"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

// envWithoutPolarsSkipCPUCheck is os.Environ() without POLARS_SKIP_CPU_CHECK,
// matched without regard to case because Windows environment names are.
func envWithoutPolarsSkipCPUCheck() []string {
	env := os.Environ()
	out := make([]string, 0, len(env))
	for _, kv := range env {
		name, _, ok := strings.Cut(kv, "=")
		if ok && strings.EqualFold(name, "POLARS_SKIP_CPU_CHECK") {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// TestPolarsCPUCheckIsSkippedOnlyForX86PythonOnArmWindows pins the preamble's
// guard to exactly one situation: Windows, on an ARM64 machine, running an
// x86-64 interpreter. There, polars reads platform.machine() as ARM64, cannot
// read the x86 feature flags under emulation, and `import polars` raises
// RuntimeError: unknown feature flag. Everywhere else the variable must stay
// as the caller left it, because on a native interpreter the check is correct
// and skipping it would hide a genuinely unsupported CPU.
func TestPolarsCPUCheckIsSkippedOnlyForX86PythonOnArmWindows(t *testing.T) {
	python := pythonInterpreterOnPath()
	if python == "" {
		t.Skip("no Python on PATH to run the preamble with")
	}

	cases := []struct {
		name     string
		osName   string
		machine  string
		platform string
		preset   string
		want     string
	}{
		{"emulated x86 python on arm windows", "nt", "ARM64", "win-amd64", "", "1"},
		{"machine reported in lower case", "nt", "arm64", "win-amd64", "", "1"},
		{"native arm python on arm windows", "nt", "ARM64", "win-arm64", "", "<unset>"},
		{"real x64 windows", "nt", "AMD64", "win-amd64", "", "<unset>"},
		{"apple silicon is not windows", "posix", "arm64", "macosx-11.0-arm64", "", "<unset>"},
		{"a value the caller set is kept", "nt", "ARM64", "win-amd64", "0", "0"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := []string{
				"import os, platform, sysconfig",
				fmt.Sprintf("os.name = %s", strconv.Quote(tc.osName)),
				fmt.Sprintf("platform.machine = lambda: %s", strconv.Quote(tc.machine)),
				fmt.Sprintf("sysconfig.get_platform = lambda: %s", strconv.Quote(tc.platform)),
			}
			if tc.preset != "" {
				lines = append(lines, fmt.Sprintf("os.environ[\"POLARS_SKIP_CPU_CHECK\"] = %s", strconv.Quote(tc.preset)))
			} else {
				lines = append(lines, "os.environ.pop(\"POLARS_SKIP_CPU_CHECK\", None)")
			}
			lines = append(lines, skipPolarsCPUCheckUnderEmulation)
			lines = append(lines, "print(os.environ.get(\"POLARS_SKIP_CPU_CHECK\", \"<unset>\"))")

			script := strings.Join(lines, "\n")

			cmd := exec.Command(python, "-c", script)
			cmd.Env = envWithoutPolarsSkipCPUCheck()
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("python -c failed: %v\nfull output:\n%s\nscript:\n%s", err, out, script)
			}

			got := strings.TrimSpace(string(out))
			if got != tc.want {
				t.Errorf("os.name=%q platform.machine()=%q sysconfig.get_platform()=%q (preset %q): POLARS_SKIP_CPU_CHECK is %q, want %q; the preamble must set it to 1 only for an x86-64 interpreter on an ARM64 Windows machine, leave it alone everywhere else, and keep a value the caller already set",
					tc.osName, tc.machine, tc.platform, tc.preset, got, tc.want)
			}
		})
	}
}

// TestPolarsCPUCheckGuardRunsBeforePolarsIsImported checks the ordering that
// makes the fix work at all. generateDefaultPyCode emits the dependency
// imports, `import polars as pl` among them, before anything else. A guard that
// ran after them would set the variable too late, and every run would still
// fail with RuntimeError: unknown feature flag.
func TestPolarsCPUCheckGuardRunsBeforePolarsIsImported(t *testing.T) {
	code := generateDefaultPyCode("test-id", "test-addr")

	guard := strings.Index(code, skipPolarsCPUCheckUnderEmulation)
	if guard < 0 {
		t.Errorf("the generated preamble does not contain the polars CPU-check guard; on Windows arm64 running an x86-64 interpreter every run fails with RuntimeError: unknown feature flag, so generateDefaultPyCode must put skipPolarsCPUCheckUnderEmulation in the preamble, before the imports")
	}

	polars := strings.Index(code, "import polars as pl")
	if polars < 0 {
		t.Fatalf("the generated preamble does not import polars; the dependency is registered as %q, so generateDefaultPyCode is expected to emit it", "import polars as pl")
	}

	if guard < 0 || guard > polars {
		t.Errorf("the polars CPU-check guard sits at offset %d and the polars import at %d; the guard must come first, otherwise polars reads platform.machine() before POLARS_SKIP_CPU_CHECK is set and every run on Windows arm64 with an emulated x86-64 interpreter raises RuntimeError: unknown feature flag", guard, polars)
	}
}
