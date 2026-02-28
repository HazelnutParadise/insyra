package dsl

import (
	"io"

	"github.com/HazelnutParadise/insyra/cli/repl"
)

// Session is the programmatic DSL execution session.
//
// It supports Execute(line), ExecuteFile(path), and Context().
type Session = repl.DSLSession

// NewSession creates a DSL session bound to an Insyra environment.
//
// envName empty string means "default".
// output nil means discard output.
func NewSession(envName string, output io.Writer) (*Session, error) {
	return repl.NewDSLSession(envName, output)
}
