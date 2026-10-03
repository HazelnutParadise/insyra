package dsl

import (
	"io"
	"testing"

	"github.com/HazelnutParadise/insyra/cli/commands"
	"github.com/HazelnutParadise/insyra/cli/env"
)

// #260 (EN-2): moving the session out of cli/ left the exported API of
// engine/dsl as it was. NewSession still takes the Manager from cli/env, and
// Context still returns the ExecContext from cli/commands, because both are
// the same types under their public names.
func TestEngineDSLAPIIsUnchanged(t *testing.T) {
	api := struct {
		newSession  func(*env.Manager, string, io.Writer) (*Session, error)
		execute     func(*Session, string) error
		executeFile func(*Session, string) error
		context     func(*Session) *commands.ExecContext
	}{NewSession, (*Session).Execute, (*Session).ExecuteFile, (*Session).Context}
	if api.newSession == nil || api.execute == nil || api.executeFile == nil || api.context == nil {
		t.Fatal("engine/dsl lost part of its API")
	}
}
