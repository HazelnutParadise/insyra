package lp

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra/lpgen"
)

// A LINGO model that declares a variable free used to come back from the
// parser without that declaration, so the solver treated the variable as
// non-negative and answered a different problem from the one LINGO showed.
// Reading @FREE through to the solver is what makes the parsed model the model
// that was written: with the declaration the objective reaches -3, without it
// the same text gives 0, because X is then held at 0.
func TestALingoFreeVariableReachesTheSolver(t *testing.T) {
	model, err := lpgen.ParseLingo("MODEL:\nMIN= X;\n@FREE(X);\nX + Y >= -3;\nY <= 0;\nEND")
	if err != nil {
		t.Fatalf("lpgen.ParseLingo: %v", err)
	}

	sol, err := Solve(model, Options{})
	if err != nil {
		t.Fatalf("Solve: %v", err)
	}
	if sol.Status != StatusOptimal {
		t.Fatalf("status: got %v, want optimal", sol.Status)
	}
	if math.Abs(sol.Objective-(-3)) > 1e-9 {
		t.Errorf("objective: got %v, want -3", sol.Objective)
	}
}
