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
// platform, which uv builds from source there with the tools the build
// constraints pin. PyPI has no Windows arm64 wheel of blis 1.3.3 or statsmodels 0.15.0.
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

// buildConstraint is one entry of the build constraints: the exact version a
// source build may install and the SHA-256 of each file of that version.
type buildConstraint struct {
	version string
	hashes  []string // "sha256:<hex>", sorted
}

// normalizedName is a package name as PEP 503 compares it.
func normalizedName(name string) string {
	lower := strings.ToLower(name)
	var b strings.Builder
	b.Grow(len(lower))
	prevDash := false
	for _, r := range lower {
		if r == '-' || r == '_' || r == '.' {
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
			continue
		}
		prevDash = false
		b.WriteRune(r)
	}
	return b.String()
}

// An exact name==version build constraint, and one SHA-256 as uv writes it.
var (
	buildConstraintPin = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*==[0-9][0-9A-Za-z.+-]*$`)
	buildConstraintSHA = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	// oneHash finds the quoted strings of a hashes = [...] list.
	oneHash = regexp.MustCompile(`"([^"]*)"`)
)

// checkBuildConstraintHashes reports each hash that is not a SHA-256, and an
// empty list.
func checkBuildConstraintHashes(hashes []string) []string {
	var problems []string
	if len(hashes) == 0 {
		problems = append(problems, "a build constraint has an empty hashes list; give at least one sha256 so uv takes the file from PyPI")
	}
	for _, h := range hashes {
		if !buildConstraintSHA.MatchString(h) {
			problems = append(problems, fmt.Sprintf("hash %q is not sha256: followed by exactly 64 lowercase hex characters", h))
		}
	}
	return problems
}

// countConstraintItems counts the top-level items of a build-constraint list
// body, so an item that is not a { requirement = ..., hashes = [...] } table is
// still counted.
func countConstraintItems(list string) int {
	depth, items, open := 0, 0, false
	for _, r := range list {
		switch r {
		case '{':
			if depth == 0 && !open {
				items, open = items+1, true
			}
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
		case '"', '\'':
			// A list item written as a bare string, "name==version" or
			// 'name==version', rather than as a table.
			if depth == 0 && !open {
				items, open = items+1, true
			}
		case ',':
			if depth == 0 {
				open = false
			}
		}
	}
	return items
}

// pyprojectBuildConstraint finds the requirement and the hashes of one
// { requirement = "...", hashes = [ ... ] } item, in a list body.
var pyprojectBuildConstraint = regexp.MustCompile(`(?s)requirement = "([^"]*)".*?hashes = \[(.*?)\]`)

// parsePyprojectBuildConstraints reads the build-constraint-dependencies list
// of a pyproject.toml. It returns the entries by normalised name, one message
// for each entry that is not an exact name==version pin with at least one
// SHA-256 or whose name is listed twice, and whether the list is there.
func parsePyprojectBuildConstraints(text string) (map[string]buildConstraint, []string, bool) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	const open = "build-constraint-dependencies = ["
	lines := strings.Split(text, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == open {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return nil, nil, false
	}
	body := lines[start:]
	end := -1
	for i, line := range body {
		if strings.TrimSpace(line) == "]" {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, []string{"the build-constraint-dependencies list is not closed by a line holding ]"}, true
	}
	// A commented-out entry is not one, for uv or for this check.
	var kept []string
	for _, line := range body[:end] {
		if !strings.HasPrefix(strings.TrimSpace(line), "#") {
			kept = append(kept, line)
		}
	}
	list := strings.Join(kept, "\n")

	var problems []string
	constraints := map[string]buildConstraint{}
	for _, m := range pyprojectBuildConstraint.FindAllStringSubmatch(list, -1) {
		requirement := m[1]
		var hashes []string
		for _, h := range oneHash.FindAllStringSubmatch(m[2], -1) {
			hashes = append(hashes, h[1])
		}
		if !buildConstraintPin.MatchString(requirement) {
			problems = append(problems, fmt.Sprintf("build constraint %q is not pinned as \"name==version\"; a source build must not be free to install another version of a build tool", requirement))
			continue
		}
		problems = append(problems, checkBuildConstraintHashes(hashes)...)
		name, version, _ := strings.Cut(requirement, "==")
		name = normalizedName(name)
		if _, dup := constraints[name]; dup {
			problems = append(problems, fmt.Sprintf("%q is listed twice; after PEP 503 normalisation one name pins one version", name))
			continue
		}
		slices.Sort(hashes)
		constraints[name] = buildConstraint{version: version, hashes: hashes}
	}
	if items, pins := countConstraintItems(list), strings.Count(list, "requirement = "); items != len(constraints) || pins != len(constraints) {
		problems = append(problems, fmt.Sprintf("the build-constraint-dependencies list has %d item(s) and %d \"requirement = \" but %d usable entries; write each item as { requirement = \"name==version\", hashes = [...] }", items, pins, len(constraints)))
	}
	return constraints, problems, true
}

// lockBuildConstraint matches one line of the build-constraints list of the
// [manifest] table of a uv.lock.
var lockBuildConstraint = regexp.MustCompile(`^\s*\{\s*name = "([^"]+)",\s*specifier = "==([^"]+)",\s*hashes = \[(.+)\]\s*\},?\s*$`)

// parseLockBuildConstraints reads the build-constraints list under
// [manifest] of a uv.lock, the same way.
func parseLockBuildConstraints(text string) (map[string]buildConstraint, []string, bool) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	manifest := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "[manifest]" {
			manifest = i
			break
		}
	}
	if manifest < 0 {
		return nil, nil, false
	}
	start := -1
	for i, line := range lines[manifest+1:] {
		if strings.TrimSpace(line) == "build-constraints = [" {
			start = manifest + 1 + i + 1
			break
		}
	}
	if start < 0 {
		return nil, nil, false
	}
	body := lines[start:]
	end := -1
	for i, line := range body {
		if strings.TrimSpace(line) == "]" {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, []string{"the build-constraints list under [manifest] is not closed by a line holding ]"}, true
	}

	var problems []string
	constraints := map[string]buildConstraint{}
	for _, line := range body[:end] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		m := lockBuildConstraint.FindStringSubmatch(line)
		if m == nil {
			problems = append(problems, fmt.Sprintf("uv.lock records a build constraint as %q; the line uv writes is { name = \"name\", specifier = \"==version\", hashes = [\"sha256:...\"] }", strings.TrimSpace(line)))
			continue
		}
		var hashes []string
		for _, h := range oneHash.FindAllStringSubmatch(m[3], -1) {
			hashes = append(hashes, h[1])
		}
		problems = append(problems, checkBuildConstraintHashes(hashes)...)
		name := normalizedName(m[1])
		if _, dup := constraints[name]; dup {
			problems = append(problems, fmt.Sprintf("uv.lock lists the build constraint %q twice", name))
			continue
		}
		slices.Sort(hashes)
		constraints[name] = buildConstraint{version: m[2], hashes: hashes}
	}
	return constraints, problems, true
}

// lockedVersions returns the versions uv.lock resolves each package to, by
// normalised name. A lock that forks by platform can resolve one package to
// more than one version.
func lockedVersions(t *testing.T) map[string][]string {
	t.Helper()
	lock := strings.ReplaceAll(string(envLock), "\r\n", "\n")
	name := regexp.MustCompile(`(?m)^name = "([^"]+)"$`)
	version := regexp.MustCompile(`(?m)^version = "([^"]+)"$`)
	versions := map[string][]string{}
	for _, block := range strings.Split(lock, "\n[[package]]\n")[1:] {
		m := name.FindStringSubmatch(block)
		if m == nil {
			t.Fatalf("a [[package]] block in uv.lock has no name:\n%s", block)
		}
		v := version.FindStringSubmatch(block)
		if v == nil {
			t.Fatalf("the [[package]] block of %s in uv.lock has no version", m[1])
		}
		key := normalizedName(m[1])
		versions[key] = append(versions[key], v[1])
	}
	return versions
}

// sourceBuiltPackages returns every package knownSourceBuilds names, sorted
// and without repeats.
func sourceBuiltPackages() []string {
	seen := map[string]bool{}
	var all []string
	for _, pkgs := range knownSourceBuilds {
		for _, p := range pkgs {
			if !seen[p] {
				seen[p] = true
				all = append(all, p)
			}
		}
	}
	slices.Sort(all)
	return all
}

func TestSourceBuildToolsArePinned(t *testing.T) {
	built := sourceBuiltPackages()
	constraints, problems, found := parsePyprojectBuildConstraints(string(envPyproject))
	for _, p := range problems {
		t.Errorf("pyproject.toml: %s", p)
	}
	if !found || len(constraints) == 0 {
		if len(built) == 0 {
			return
		}
		t.Fatalf("pyproject.toml has no build-constraint-dependencies, but uv builds %s from source; pin the tools their builds install", strings.Join(built, " and "))
	}
	locked, lockProblems, lockFound := parseLockBuildConstraints(string(envLock))
	if !lockFound {
		t.Fatalf("uv.lock has no build-constraints under [manifest]; run uv lock in py/environment")
	}
	for _, p := range lockProblems {
		t.Errorf("uv.lock: %s", p)
	}
	for name, c := range constraints {
		other, ok := locked[name]
		if !ok {
			t.Errorf("pyproject.toml pins the build tool %s==%s, but uv.lock has no build constraint for it; run uv lock in py/environment", name, c.version)
			continue
		}
		if other.version != c.version {
			t.Errorf("pyproject.toml pins %s at %s, but uv.lock records %s; run uv lock in py/environment", name, c.version, other.version)
		}
		if !slices.Equal(other.hashes, c.hashes) {
			t.Errorf("pyproject.toml gives %s the hashes %v, but uv.lock gives %v; run uv lock in py/environment", name, c.hashes, other.hashes)
		}
	}
	for name := range locked {
		if _, ok := constraints[name]; !ok {
			t.Errorf("uv.lock pins the build tool %s, but pyproject.toml does not; run uv lock in py/environment after editing pyproject.toml", name)
		}
	}
	environment := lockedVersions(t)
	for name, c := range constraints {
		for _, version := range environment[name] {
			if version != c.version {
				t.Errorf("the build tool %s is pinned at %s, but the environment runs %s", name, c.version, version)
			}
		}
	}
}

func TestBuildConstraintParsingRefusesLoosePins(t *testing.T) {
	hashA := "sha256:" + strings.Repeat("a", 64)
	hashB := "sha256:" + strings.Repeat("b", 64)
	cases := []struct {
		name        string
		text        string
		wantFound   bool
		wantProblem bool
		wantEntry   *buildConstraint
	}{
		{
			name:      "an exact pin with two hashes in the wrong order",
			wantFound: true,
			wantEntry: &buildConstraint{version: "3.3.0", hashes: []string{hashA, hashB}},
			text: `[tool.uv]
build-constraint-dependencies = [
    { requirement = "cython==3.3.0", hashes = [
        "` + hashB + `",
        "` + hashA + `",
    ] },
]
`,
		},
		{
			name:        "a range instead of an exact pin",
			wantFound:   true,
			wantProblem: true,
			text: `[tool.uv]
build-constraint-dependencies = [
    { requirement = "numpy>=2.0", hashes = ["` + hashA + `"] },
]
`,
		},
		{
			name:        "a wildcard version",
			wantFound:   true,
			wantProblem: true,
			text: `[tool.uv]
build-constraint-dependencies = [
    { requirement = "numpy==2.*", hashes = ["` + hashA + `"] },
]
`,
		},
		{
			name:        "a requirement with no version",
			wantFound:   true,
			wantProblem: true,
			text: `[tool.uv]
build-constraint-dependencies = [
    { requirement = "numpy", hashes = ["` + hashA + `"] },
]
`,
		},
		{
			name:        "no hashes at all",
			wantFound:   true,
			wantProblem: true,
			text: `[tool.uv]
build-constraint-dependencies = [
    { requirement = "cython==3.3.0", hashes = [] },
]
`,
		},
		{
			name:        "a hash one character short",
			wantFound:   true,
			wantProblem: true,
			text: `[tool.uv]
build-constraint-dependencies = [
    { requirement = "cython==3.3.0", hashes = ["sha256:` + strings.Repeat("a", 63) + `"] },
]
`,
		},
		{
			name:        "a hash that is not SHA-256",
			wantFound:   true,
			wantProblem: true,
			text: `[tool.uv]
build-constraint-dependencies = [
    { requirement = "cython==3.3.0", hashes = ["md5:` + strings.Repeat("a", 64) + `"] },
]
`,
		},
		{
			name:        "a hash in uppercase hex",
			wantFound:   true,
			wantProblem: true,
			text: `[tool.uv]
build-constraint-dependencies = [
    { requirement = "cython==3.3.0", hashes = ["sha256:` + strings.Repeat("A", 64) + `"] },
]
`,
		},
		{
			name:        "one name written two ways",
			wantFound:   true,
			wantProblem: true,
			text: `[tool.uv]
build-constraint-dependencies = [
    { requirement = "setuptools-scm==9.2.2", hashes = ["` + hashA + `"] },
    { requirement = "setuptools_scm==9.2.2", hashes = ["` + hashA + `"] },
]
`,
		},
		{
			name:        "an item written as a bare string",
			wantFound:   true,
			wantProblem: true,
			text: `[tool.uv]
build-constraint-dependencies = [
    "numpy==2.5.3",
]
`,
		},
		{
			name:        "an item written as a single-quoted string",
			wantFound:   true,
			wantProblem: true,
			text: `[tool.uv]
build-constraint-dependencies = [
    'ninja>=1',
]
`,
		},
		{
			name:        "an entry that is commented out",
			wantFound:   true,
			wantProblem: false,
			wantEntry:   &buildConstraint{version: "3.3.0", hashes: []string{hashA}},
			text: `[tool.uv]
build-constraint-dependencies = [
    { requirement = "cython==3.3.0", hashes = ["` + hashA + `"] },
    # { requirement = "ninja==1.13.2", hashes = ["` + hashA + `"] },
]
`,
		},
		{
			name:      "no list at all",
			wantFound: false,
			text: `[project]
name = "insyra-py-env"
dependencies = [
    "numpy==2.5.3",
]
`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, problems, found := parsePyprojectBuildConstraints(c.text)
			if found != c.wantFound {
				t.Fatalf("found is %v, want %v", found, c.wantFound)
			}
			if c.wantProblem && len(problems) == 0 {
				t.Errorf("no problem was reported for:\n%s", c.text)
			}
			if !c.wantProblem && len(problems) > 0 {
				t.Errorf("problems were reported for a well-formed list: %v", problems)
			}
			if c.wantEntry == nil {
				return
			}
			e, ok := got["cython"]
			if !ok || len(got) != 1 {
				t.Fatalf("the map has %v, want only an entry for cython", got)
			}
			if e.version != c.wantEntry.version || !slices.Equal(e.hashes, c.wantEntry.hashes) {
				t.Errorf("cython is %+v, want version %s and hashes %v", e, c.wantEntry.version, c.wantEntry.hashes)
			}
		})
	}
}

func TestLockBuildConstraintParsingRefusesAMalformedLine(t *testing.T) {
	hashA := "sha256:" + strings.Repeat("a", 64)
	t.Run("a well-formed line", func(t *testing.T) {
		text := "[manifest]\n" +
			"build-constraints = [\n" +
			"    { name = \"cython\", specifier = \"==3.3.0\", hashes = [\"" + hashA + "\"] },\n" +
			"]\n"
		got, problems, found := parseLockBuildConstraints(text)
		if !found {
			t.Fatal("found is false, want true")
		}
		if len(problems) > 0 {
			t.Errorf("problems were reported for a well-formed line: %v", problems)
		}
		e, ok := got["cython"]
		if !ok {
			t.Fatalf("the map has %v, want an entry for cython", got)
		}
		if e.version != "3.3.0" || !slices.Equal(e.hashes, []string{hashA}) {
			t.Errorf("cython is %+v, want version 3.3.0 and hashes [%s]", e, hashA)
		}
	})
	t.Run("a line uv would not write", func(t *testing.T) {
		text := "[manifest]\n" +
			"build-constraints = [\n" +
			"    { name = \"cython\", specifier = \"==3.3.0\" },\n" +
			"]\n"
		_, problems, found := parseLockBuildConstraints(text)
		if !found {
			t.Fatal("found is false, want true")
		}
		if len(problems) == 0 {
			t.Error("no problem was reported for a line without a hashes list")
		}
	})
	t.Run("no manifest table", func(t *testing.T) {
		text := "version = 1\n\n[[package]]\nname = \"numpy\"\nversion = \"2.5.3\"\n"
		if _, _, found := parseLockBuildConstraints(text); found {
			t.Error("found is true, want false when uv.lock has no [manifest]")
		}
	})
}
