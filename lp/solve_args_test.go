package lp

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// More than one timeout is a call that is already wrong. SolveFromFile refuses
// it before initGLPK, so it does not go looking for — or install — a solver,
// and it still logs a warning and returns nil, nil as it always did.
//
// The probe consumes glpkInitOnce after the call: had SolveFromFile run
// initGLPK, the Once would already be spent and the probe would not run. No
// test in this package may reach initGLPK, or the probe would read it wrong.
func TestSolveFromFile_TooManyTimeoutsIsRefusedBeforeInit(t *testing.T) {
	level := insyra.Config.GetLogLevel()
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)
	t.Cleanup(func() { insyra.Config.SetLogLevel(level) })
	insyra.ClearErrors()

	result, info := SolveFromFile("no_such_model.lp", 1, 2)
	if result != nil || info != nil {
		t.Errorf("got (%v, %v), want (nil, nil)", result, info)
	}

	warned := false
	for _, e := range insyra.GetAllErrors() {
		if e.PackageName == "lp" && e.FuncName == "SolveFromFile" {
			warned = true
		}
	}
	if !warned {
		t.Error("the refused call logged no warning")
	}

	probed := false
	glpkInitOnce.Do(func() { probed = true })
	if !probed {
		t.Error("SolveFromFile ran initGLPK before refusing its arguments")
	}
}
