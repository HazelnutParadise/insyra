package py

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// SEC-10 of #289: the environment installed uv through its install script,
// Python as "3.12.*" and every package at its newest version. Every version
// now lives in py/environment, and these tests fail when the files there stop
// agreeing with each other.

var exactVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

func TestPinnedVersionsCanBeRead(t *testing.T) {
	uv, err := pinnedUVVersion()
	if err != nil {
		t.Fatal(err)
	}
	python, err := pinnedPythonVersion()
	if err != nil {
		t.Fatal(err)
	}
	if !exactVersion.MatchString(uv) || !exactVersion.MatchString(python) {
		t.Errorf("uv %q and Python %q are not exact versions", uv, python)
	}
}

// pinnedDependencies returns the dependencies pyproject.toml lists, name to
// version, and fails the test on any that is not an exact name==version pin.
func pinnedDependencies(t *testing.T) map[string]string {
	t.Helper()
	text := strings.ReplaceAll(string(envPyproject), "\r\n", "\n")
	const open = "dependencies = [\n"
	start := strings.Index(text, open)
	if start < 0 {
		t.Fatal("pyproject.toml has no dependencies list")
	}
	body := text[start+len(open):]
	end := strings.Index(body, "]")
	if end < 0 {
		t.Fatal("the dependencies list in pyproject.toml is not closed")
	}
	pin := regexp.MustCompile(`^"([a-z0-9][a-z0-9._-]*)==([0-9][0-9A-Za-z.+-]*)",$`)
	deps := map[string]string{}
	for _, line := range strings.Split(body[:end], "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := pin.FindStringSubmatch(line)
		if m == nil {
			t.Errorf("dependency %s is not pinned as \"name==version\",", line)
			continue
		}
		deps[m[1]] = m[2]
	}
	return deps
}

func TestEveryImportedPackageIsPinned(t *testing.T) {
	deps := pinnedDependencies(t)
	imported := map[string]bool{}
	for imp, pkg := range pyDependencies {
		if pkg == "" {
			continue
		}
		imported[pkg] = true
		if _, ok := deps[pkg]; !ok {
			t.Errorf("the preamble runs %q, but %s is not pinned in environment/pyproject.toml", imp, pkg)
		}
	}
	for pkg := range deps {
		if !imported[pkg] {
			t.Errorf("%s is pinned, but the preamble does not import it", pkg)
		}
	}
}

func TestLockAgreesWithThePins(t *testing.T) {
	lock := strings.ReplaceAll(string(envLock), "\r\n", "\n")
	python, err := pinnedPythonVersion()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lock, "\nrequires-python = \"=="+python+"\"\n") {
		t.Errorf("uv.lock does not require Python ==%s; run uv lock in py/environment", python)
	}
	for name, version := range pinnedDependencies(t) {
		entry := fmt.Sprintf("[[package]]\nname = %q\nversion = %q\n", name, version)
		if !strings.Contains(lock, entry) {
			t.Errorf("uv.lock does not resolve %s to %s; run uv lock in py/environment", name, version)
		}
		spec := fmt.Sprintf("{ name = %q, specifier = \"==%s\" }", name, version)
		if !strings.Contains(lock, spec) {
			t.Errorf("uv.lock was not made from the pin %s==%s; run uv lock in py/environment", name, version)
		}
	}
}

func TestEverySupportedPlatformHasAUVChecksum(t *testing.T) {
	sha256Hex := regexp.MustCompile(`^[0-9a-f]{64}$`)
	for _, p := range [][2]string{
		{"darwin", "amd64"}, {"darwin", "arm64"},
		{"linux", "amd64"}, {"linux", "arm64"},
		{"windows", "amd64"}, {"windows", "arm64"},
	} {
		name, sum, err := uvArchive(p[0], p[1])
		if err != nil {
			t.Errorf("%s/%s: %v", p[0], p[1], err)
			continue
		}
		if !sha256Hex.MatchString(sum) {
			t.Errorf("%s: checksum %q is not a SHA-256", name, sum)
		}
		kind := ".tar.gz"
		if p[0] == "windows" {
			kind = ".zip"
		}
		if !strings.HasSuffix(name, kind) {
			t.Errorf("%s/%s takes %s, want a %s archive", p[0], p[1], name, kind)
		}
	}
	if _, _, err := uvArchive("plan9", "386"); err == nil || !strings.Contains(err.Error(), "plan9/386") {
		t.Errorf("an unsupported platform gave %v, want an error naming it", err)
	}
}

func TestLockDownloadsFromPyPIOnly(t *testing.T) {
	lock := strings.ReplaceAll(string(envLock), "\r\n", "\n")
	source := regexp.MustCompile(`source = \{ registry = "([^"]+)" \}`)
	registries := source.FindAllStringSubmatch(lock, -1)
	if len(registries) == 0 {
		t.Fatal("uv.lock names no registry")
	}
	for _, m := range registries {
		if m[1] != "https://pypi.org/simple" {
			t.Errorf("uv.lock takes a package from %s; lock with UV_NO_CONFIG=1 and no index settings, so every file comes from PyPI", m[1])
		}
	}
}

// lockedWheels returns each locked package's name and the file names of its
// wheels. Packages the lock records without wheels are left out.
func lockedWheels(t *testing.T) map[string][]string {
	t.Helper()
	lock := strings.ReplaceAll(string(envLock), "\r\n", "\n")
	name := regexp.MustCompile(`(?m)^name = "([^"]+)"$`)
	wheel := regexp.MustCompile(`url = "[^"]*/([^"/]+\.whl)"`)
	wheels := map[string][]string{}
	for _, block := range strings.Split(lock, "\n[[package]]\n")[1:] {
		m := name.FindStringSubmatch(block)
		if m == nil {
			t.Fatalf("a [[package]] block in uv.lock has no name:\n%s", block)
		}
		for _, w := range wheel.FindAllStringSubmatch(block, -1) {
			wheels[m[1]] = append(wheels[m[1]], w[1])
		}
	}
	return wheels
}

var (
	// wheelPlatforms matches the wheel platform tags each supported platform
	// installs.
	wheelPlatforms = map[string]*regexp.Regexp{
		"darwin/amd64":  regexp.MustCompile(`macosx_[0-9_]+_(x86_64|intel|universal2)\.whl$`),
		"darwin/arm64":  regexp.MustCompile(`macosx_[0-9_]+_(arm64|universal2)\.whl$`),
		"linux/amd64":   regexp.MustCompile(`linux[0-9_]*_x86_64\.whl$`),
		"linux/arm64":   regexp.MustCompile(`linux[0-9_]*_aarch64\.whl$`),
		"windows/amd64": regexp.MustCompile(`win_amd64\.whl$`),
		"windows/arm64": regexp.MustCompile(`win_arm64\.whl$`),
	}
	// python312Wheel matches a wheel CPython 3.12 can install.
	python312Wheel = regexp.MustCompile(`-(cp312-cp312|cp3[0-9]+-abi3|py3-none|py2\.py3-none)-`)
	// anyPlatformWheel matches a pure-Python wheel, which installs anywhere.
	anyPlatformWheel = regexp.MustCompile(`-(py3|py2\.py3)-none-any\.whl$`)
)

// knownSourceBuilds lists the locked packages with no wheel for a supported
// platform, which uv builds from source there. That needs a C compiler, and
// the build's own dependencies are not in the lock. PyPI has no Windows arm64
// wheel of blis at any 1.x release, nor of statsmodels 0.15.0.
var knownSourceBuilds = map[string][]string{
	"windows/arm64": {"blis", "statsmodels"},
}

func TestLockHasAWheelForEverySupportedPlatform(t *testing.T) {
	for pkg, wheels := range lockedWheels(t) {
		if slices.ContainsFunc(wheels, anyPlatformWheel.MatchString) {
			continue
		}
		for platform, tag := range wheelPlatforms {
			found := slices.ContainsFunc(wheels, func(w string) bool {
				return tag.MatchString(w) && python312Wheel.MatchString(w)
			})
			exception := slices.Contains(knownSourceBuilds[platform], pkg)
			if !found && !exception {
				t.Errorf("uv.lock has no %s wheel of %s for Python 3.12, so uv would build it from source", platform, pkg)
			}
			if found && exception {
				t.Errorf("%s now has a %s wheel; take it off knownSourceBuilds and the note in Docs/py.md", pkg, platform)
			}
		}
	}
}

// The environment directory is named for the Python it runs, as
// py<two-digit year><letter>: a new Python version gets a new directory, so
// an environment never switches Python in place. py25b became py25c when the
// Python spec changed on 2025-10-19, and py25c became py26a with 3.12.14.
func TestTheEnvironmentDirectoryIsNamedForThePinnedPython(t *testing.T) {
	python, err := pinnedPythonVersion()
	if err != nil {
		t.Fatal(err)
	}
	if python != envDirPython {
		t.Errorf("the pinned Python is %s, but the directory code %s was given for %s; give the environment directory a new code (py<two-digit year><letter>) in py/const.go and set envDirPython to %s", python, envDirCode, envDirPython, python)
	}
	if !regexp.MustCompile(`^py[0-9]{2}[a-z]$`).MatchString(envDirCode) {
		t.Errorf("the directory code %q is not py<two-digit year><letter>", envDirCode)
	}
	if want := envDirCode + "_" + runtime.GOOS + "_" + runtime.GOARCH; filepath.Base(installDir) != want {
		t.Errorf("the environment directory is %s, want %s", installDir, want)
	}
}
