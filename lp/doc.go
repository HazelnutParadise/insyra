// Package lp solves linear and mixed-integer programs built with lpgen or read
// from CPLEX LP files.
//
// The default engine is go-milp, a pure-Go solver that needs nothing
// installed. GLPK can be selected instead with Options.Engine; lp then runs an
// installed glpsol and never installs one itself.
package lp
