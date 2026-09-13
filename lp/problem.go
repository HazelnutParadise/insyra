package lp

// problem is an LP file after reading, with every bound resolved the way GLPK
// resolves it, so an engine needs no knowledge of the file format.
type problem struct {
	maximize  bool
	objective []term
	rows      []row
	cols      []column
	// warnings are what GLPK prints while reading the same text, line numbers
	// included.
	warnings []string
}

type column struct {
	name         string
	lower, upper float64 // math.Inf where a side is unbounded
	integer      bool
}

type term struct {
	col  int
	coef float64
}

type relation int

const (
	lessEq relation = iota
	greaterEq
	equalTo
)

type row struct {
	terms []term
	rel   relation
	rhs   float64
}
