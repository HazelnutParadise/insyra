package dsl

import (
	"io"
	"testing"

	"github.com/HazelnutParadise/insyra/cli/commands"
	"github.com/HazelnutParadise/insyra/cli/env"
)

// #260 (EN-2): moving the session out of cli/ left the exported API of
// engine/dsl as it was. A program written against it still compiles: it passes
// NewSession the Manager from cli/env, now Deprecated in favour of the one here,
// and names what Context returns as the ExecContext from cli/commands; both are
// the same types as before.
func TestEngineDSLAPIIsUnchanged(t *testing.T) {
	api := struct {
		newSession  func(*env.Manager, string, io.Writer) (*Session, error) //nolint:staticcheck // the deprecated spelling must still compile
		execute     func(*Session, string) error
		executeFile func(*Session, string) error
		context     func(*Session) *commands.ExecContext //nolint:staticcheck // the deprecated spelling must still compile
	}{NewSession, (*Session).Execute, (*Session).ExecuteFile, (*Session).Context}
	if api.newSession == nil || api.execute == nil || api.executeFile == nil || api.context == nil {
		t.Fatal("engine/dsl lost part of its API")
	}
}
