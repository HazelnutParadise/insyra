package dsl

import (
	"io"
	"strings"

	"github.com/HazelnutParadise/insyra/cli/env"
	"github.com/HazelnutParadise/insyra/cli/repl"
)

// Session is the programmatic DSL execution session.
//
// It supports Execute(line), ExecuteFile(path), and Context(). It is not safe
// for concurrent use.
type Session = repl.DSLSession

// NewSession creates a DSL session bound to mgr's environment storage. A
// missing environment is created before the session opens it.
//
// mgr is required: pass DefaultManager() to use the standard
// <UserHomeDir>/.insyra root, or NewManager(path, dir) for a custom root
// (e.g. per-workspace embedding). Each session keeps its own Manager, so
// concurrent sessions in the same process can target different roots
// without interfering with each other.
//
// envName "" defaults to "default". output nil silently discards.
func NewSession(mgr *env.Manager, envName string, output io.Writer) (*Session, error) {
	if mgr != nil {
		if strings.TrimSpace(envName) == "" {
			envName = "default"
		}
		if !mgr.Exists(envName) {
			if err := mgr.Create(envName); err != nil && !mgr.Exists(envName) {
				return nil, err
			}
		}
	}
	return repl.NewDSLSession(mgr, envName, output)
}
