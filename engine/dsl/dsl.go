// Package dsl runs the insyra command language from Go: the commands the
// insyra command, its REPL and .isr scripts take, against variables and an
// environment the program owns.
package dsl

import (
	"io"

	"github.com/HazelnutParadise/insyra/internal/dsl"
	"github.com/HazelnutParadise/insyra/internal/dsl/env"
)

// Session is the programmatic DSL execution session.
//
// It supports Execute(line), ExecuteFile(path), and Context().
type Session = dsl.Session

// NewSession creates a DSL session bound to mgr's environment storage.
//
// mgr is required: pass env.Default() from cli/env to use the standard
// <UserHomeDir>/.insyra root, or env.NewManager(path) for a custom root
// (e.g. per-workspace embedding). Each session keeps its own Manager, so
// concurrent sessions in the same process can target different roots
// without interfering with each other.
//
// envName "" defaults to "default". output nil silently discards.
func NewSession(mgr *env.Manager, envName string, output io.Writer) (*Session, error) {
	return dsl.NewSession(mgr, envName, output)
}
