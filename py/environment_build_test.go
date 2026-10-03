package py

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra/internal/utils"
)

// uv writes the tools it installs into a build environment under one of these
// prefixes. Its "Installing in ..." line is a different one and names the
// packages themselves, so it must not be read as a tool.
var buildRequirementPrefixes = []string{
	"Installing build requirements: ",
	"Installing build requirement: ",
}

// readBuildRequirements returns the name==version of each tool uv's debug log
// says it installed into a build environment, in the order the log gives
// them.
func readBuildRequirements(log string) []string {
	var reqs []string
	for _, line := range strings.Split(strings.ReplaceAll(log, "\r\n", "\n"), "\n") {
		for _, prefix := range buildRequirementPrefixes {
			_, list, found := strings.Cut(line, prefix)
			if !found {
				continue
			}
			for _, req := range strings.Split(list, ", ") {
				if req = strings.TrimSpace(req); req != "" {
					reqs = append(reqs, req)
				}
			}
			break
		}
	}
	return reqs
}

// uv installs a build tool that no build constraint names without an error,
// so only a real build shows whether the constraints cover every tool. This
// builds every package some supported platform builds from source, here,
// with the pinned uv and an empty cache.
func TestSourceBuildsUseOnlyPinnedTools(t *testing.T) {
	if os.Getenv("INSYRA_PY_E2E") != "1" {
		t.Skip("set INSYRA_PY_E2E=1 to build the source packages with the pinned uv")
	}
	// Two packages compiled from source, with an empty cache, take a few
	// minutes; the ceiling is here so a stuck download fails rather than hangs.
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()

	python, err := pinnedPythonVersion()
	if err != nil {
		t.Fatal(err)
	}
	sources := sourceBuiltPackages()
	if len(sources) == 0 {
		t.Skip("no supported platform builds a package from source")
	}

	// The builds run inside the cache, and Windows build tools that are not
	// long-path aware fail past 260 characters, so the directory gets a short
	// name rather than one from t.TempDir, which is named after the test.
	root, err := os.MkdirTemp("", "ib")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	uv, err := ensureUV(ctx, filepath.Join(root, ".insyra_env", "build_check"))
	if err != nil {
		t.Fatal(err)
	}

	project := filepath.Join(root, "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"pyproject.toml": envPyproject, "uv.lock": envLock} {
		if err := os.WriteFile(filepath.Join(project, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// --no-binary-package is what forces the build; -v is what makes uv write
	// the lines readBuildRequirements reads.
	args := []string{"sync", "--frozen", "--inexact", "--managed-python", "--python", python}
	for _, pkg := range sources {
		args = append(args, "--no-binary-package", pkg)
	}
	args = append(args, "-v")

	cmd := exec.CommandContext(ctx, uv, args...)
	cmd.Dir = project
	cmd.Env = append(uvSyncEnv(filepath.Join(project, ".venv")),
		"UV_CACHE_DIR="+filepath.Join(root, "cache"),
		"UV_NO_CONFIG=1")
	utils.ApplyHideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		tail := out
		if len(tail) > 8000 {
			tail = tail[len(tail)-8000:]
		}
		t.Fatalf("uv sync failed: %v. The end of its output:\n%s", err, tail)
	}

	log := string(out)
	for _, pkg := range sources {
		if !strings.Contains(log, "Built `"+pkg+"==") {
			t.Errorf("uv's log does not show %s built from source", pkg)
		}
	}
	reqs := readBuildRequirements(log)
	if len(reqs) == 0 {
		t.Fatal("uv's log names no build requirement; check whether its log format changed")
	}

	constraints, problems, found := parseLockBuildConstraints(string(envLock))
	if !found || len(problems) > 0 {
		t.Fatalf("uv.lock's build constraints cannot be read: found %v, problems: %v", found, problems)
	}

	seen := map[string]bool{}
	for _, req := range reqs {
		if seen[req] {
			continue
		}
		seen[req] = true
		name, version, ok := strings.Cut(req, "==")
		if !ok {
			t.Errorf("cannot read the build requirement %q", req)
			continue
		}
		t.Logf("build requirement %s", req)
		c, ok := constraints[normalizedName(name)]
		if !ok {
			t.Errorf("a source build installed %s, which the build constraints do not pin; add it to build-constraint-dependencies", req)
			continue
		}
		if version != c.version {
			t.Errorf("a source build installed %s, but the build constraints pin %s", req, c.version)
		}
	}
}
