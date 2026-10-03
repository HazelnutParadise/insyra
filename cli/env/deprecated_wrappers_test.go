package env

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// deprecatedWrappers are the package-level functions that only call the same
// method on Default(). #261 (CL-3): under the one-name rule of #211 each stays
// for one release, Deprecated in favour of Default().<Name>.
var deprecatedWrappers = map[string][]string{
	"env.go": {"SetBasePath", "BasePath", "EnvsPath", "ResolveEnvPath", "EnsureDefaultEnvironment",
		"EnsureBaseStructure", "Exists", "Create", "Open", "Delete", "Clear", "Rename", "List", "Info",
		"Export", "Import", "GlobalConfigPath", "LoadGlobalConfig", "SaveGlobalConfig", "UpdateGlobalConfig",
		"SaveState", "SaveVariables", "LoadState", "RestoreVariables", "AppendHistory", "ReadHistory"},
}

func TestPackageLevelWrappersAreDeprecated(t *testing.T) {
	for file, names := range deprecatedWrappers {
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		docs := map[string]string{}
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil {
				docs[fn.Name.Name] = strings.Join(strings.Fields(fn.Doc.Text()), " ")
			}
		}
		for _, name := range names {
			doc, ok := docs[name]
			if !ok {
				t.Errorf("%s: %s is missing", file, name)
				continue
			}
			want := "Deprecated: use Default()." + name + " instead."
			if !strings.Contains(doc, want) {
				t.Errorf("%s: %s doc %q does not contain %q", file, name, doc, want)
			}
		}
	}
}

// The wrappers keep their meaning until they are removed: each acts on
// Default().
func TestDeprecatedWrappersActOnDefault(t *testing.T) {
	base := t.TempDir()
	SetBasePath(base)
	t.Cleanup(func() { SetBasePath("") })

	got, err := Default().BasePath()
	if err != nil || got != base {
		t.Fatalf("Default().BasePath() = %q, %v after SetBasePath(%q)", got, err, base)
	}
	if err := Create("wrapped"); err != nil {
		t.Fatal(err)
	}
	if !Default().Exists("wrapped") {
		t.Fatal("Create did not create the environment in Default()")
	}
	if err := SaveState("wrapped", map[string]any{"n": int64(7)}); err != nil {
		t.Fatal(err)
	}
	vars, err := Default().RestoreVariables("wrapped")
	if err != nil || vars["n"] != int64(7) {
		t.Fatalf("Default().RestoreVariables = %v, %v", vars, err)
	}
	path, err := ResolveEnvPath("wrapped")
	if err != nil || path != filepath.Join(base, "envs", "wrapped") {
		t.Fatalf("ResolveEnvPath = %q, %v", path, err)
	}
}
