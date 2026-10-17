package accel

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// The backend-author API and the test hooks are not user API (#262). Their
// exported names stay one release as Deprecated wrappers, then go.
func TestBackendAuthorNamesAreDeprecated(t *testing.T) {
	want := map[string]bool{
		"RegisterBackendExecutor": false, "BackendExecutor": false,
		"ExecuteRequest": false, "ExecuteResponse": false, "ExecuteColumn": false,
		"RegisterDiscoverer": false, "Discoverer": false,
		"RegisterSDKProbe": false, "SDKProbe": false,
		"Session.RegisterDevice": false, "Session.RecordReport": false,
		"Session.PlanShardableWorkload": false, "NearestExactCPU": false,
		"ResetBackendExecutorsForTest": false, "ResetDefaultForTest": false,
		"ResetDiscoverersForTest": false, "ResetSDKProbesForTest": false,
	}
	fset := token.NewFileSet()
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var files []*ast.File
	for _, p := range paths {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, p, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	deprecated := func(doc *ast.CommentGroup) bool {
		return doc != nil && strings.Contains(doc.Text(), "Deprecated:")
	}
	for _, f := range files {
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				name := d.Name.Name
				if d.Recv != nil {
					name = "Session." + name
				}
				if _, ok := want[name]; ok {
					want[name] = deprecated(d.Doc)
				}
			case *ast.GenDecl:
				for _, s := range d.Specs {
					ts, ok := s.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if _, ok := want[ts.Name.Name]; ok {
						want[ts.Name.Name] = deprecated(ts.Doc) || deprecated(d.Doc)
					}
				}
			}
		}
	}
	for name, ok := range want {
		if !ok {
			t.Errorf("%s is exported without a Deprecated: paragraph", name)
		}
	}
}

func TestDeprecatedRegisterDeviceStillRegisters(t *testing.T) {
	s := NewSession()
	if err := s.RegisterDevice(Device{ID: "test:0", Backend: BackendCUDA}); err != nil {
		t.Fatal(err)
	}
	devices := s.Devices()
	if len(devices) == 0 || devices[len(devices)-1].ID != "test:0" {
		t.Fatalf("devices = %+v, want test:0 registered", devices)
	}
}
