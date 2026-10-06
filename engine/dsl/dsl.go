// Package dsl runs the insyra command language from Go: the commands the
// insyra command, its REPL and .isr scripts take, against variables and an
// environment the program owns.
package dsl

import (
	"io"

	"github.com/HazelnutParadise/insyra/internal/dsl"
)

// Session runs command lines against one environment: its variables, kept in
// memory and saved after every command that succeeds, and its history.
// Execute runs one line, ExecuteFile a .isr script, and Context gives the
// variables and the rest of the execution state. A Session is not safe for use
// by more than one goroutine at a time.
type Session = dsl.Session

// NewSession creates a session on the environment envName of mgr. A missing
// environment is created and an existing one is reused, so a program can name
// its environment without checking first; call mgr.Exists beforehand to refuse
// a name that is not there yet. A name an environment may not have, such as
// one containing a slash, is an error.
//
// mgr is required: DefaultManager() keeps environments where the insyra command
// does, under <UserHomeDir>/.insyra, and NewManager(root, dir) somewhere else,
// such as a workspace. Each session keeps its own Manager, so sessions in one
// process can work on different roots without interfering with each other.
//
// envName "" means "default". A nil output discards what the commands print.
func NewSession(mgr *Manager, envName string, output io.Writer) (*Session, error) {
	return dsl.NewSession(mgr, envName, output)
}
