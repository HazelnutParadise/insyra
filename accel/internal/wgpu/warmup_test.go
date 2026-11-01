package wgpu

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
	"time"
)

// Every package-level kernel declared as var x = &kernel{...} outside the
// tests is in builtinKernels, so warmup builds it.
func TestBuiltinKernelsAreAllWarmed(t *testing.T) {
	listed := map[*kernel]bool{}
	for _, k := range builtinKernels {
		if listed[k] {
			t.Errorf("kernel %s is listed twice", k.name)
		}
		listed[k] = true
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	declared := 0
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.VAR {
				continue
			}
			for _, s := range g.Specs {
				vs := s.(*ast.ValueSpec)
				for _, v := range vs.Values {
					u, ok := v.(*ast.UnaryExpr)
					if !ok || u.Op != token.AND {
						continue
					}
					if c, ok := u.X.(*ast.CompositeLit); ok {
						if id, ok := c.Type.(*ast.Ident); ok && id.Name == "kernel" {
							declared++
						}
					}
				}
			}
		}
	}
	if declared != len(builtinKernels) {
		t.Errorf("%d kernels are declared and %d are in builtinKernels", declared, len(builtinKernels))
	}
}

// Making a resident buffer starts warmup on the device, and warmup builds
// every built-in kernel. warm runs here on the current device directly,
// because another test in this package can swap the device and reset the
// kernels after warmup for it has run.
func TestFirstBufferWarmsTheKernels(t *testing.T) {
	requireResidentDevice(t)
	b, err := Upload([]float32{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Release()
	h, err := acquire()
	if err != nil {
		t.Fatal(err)
	}
	submitMu.Lock()
	started := h.warmStarted
	submitMu.Unlock()
	if !started {
		t.Fatal("making a buffer did not start warmup")
	}
	done := make(chan struct{})
	go func() { warm(h); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Minute):
		t.Fatal("warmup did not finish")
	}
	submitMu.Lock()
	defer submitMu.Unlock()
	for _, k := range builtinKernels {
		if k.err != nil {
			t.Errorf("kernel %s: %v", k.name, k.err)
		} else if k.pipe == nil {
			t.Errorf("kernel %s was not built by warmup", k.name)
		}
	}
}
