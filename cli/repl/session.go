package repl

import (
	"io"

	"github.com/HazelnutParadise/insyra/cli/env"
	"github.com/HazelnutParadise/insyra/internal/dsl"
)

// DSLSession runs the insyra command language from Go.
//
// Deprecated: use Session from engine/dsl instead, the same type. Removed in
// the release after the one that deprecated it.
type DSLSession = dsl.Session

// NewDSLSession creates a DSL session bound to mgr's environment storage.
//
// Deprecated: use NewSession from engine/dsl instead, which does the same.
// Removed in the release after the one that deprecated it.
func NewDSLSession(mgr *env.Manager, envName string, output io.Writer) (*DSLSession, error) {
	return dsl.NewSession(mgr, envName, output)
}
