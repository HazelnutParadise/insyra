package lp_test

import (
	"fmt"
	"log"

	"github.com/HazelnutParadise/insyra/lp"
	"github.com/HazelnutParadise/insyra/lpgen"
)

// The quick start in Docs/lp.md.
func ExampleSolve() {
	model := lpgen.NewLPModel().
		SetObjective("Maximize", "3 x + 2 y").
		AddConstraint("x + y <= 4").
		AddConstraint("x + 3 y <= 6")

	sol, err := lp.Solve(model, lp.Options{})
	if err != nil {
		log.Fatal(err)
	}
	if sol.Status != lp.StatusOptimal {
		log.Fatalf("no optimal solution: %v", sol.Status)
	}
	fmt.Println(sol.Objective)
	fmt.Println(sol.Values["x"])
	fmt.Println(sol.Values["y"])
	// Output:
	// 12
	// 4
	// 0
}
