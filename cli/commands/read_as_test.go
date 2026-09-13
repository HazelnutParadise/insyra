package commands

import (
	"strings"
	"testing"
)

// read supplies its own alias before handing the arguments to load, so a
// user-supplied `as` used to come back as `unknown option "as"`.
func TestReadRefusesAsWithGuidance(t *testing.T) {
	ctx := newTestExecContext(t)
	err := Dispatch(ctx, "read", []string{"sales.csv", "as", "x"})
	if err == nil {
		t.Fatal("read accepted an alias")
	}
	msg := err.Error()
	if strings.Contains(msg, "unknown option") {
		t.Errorf("the message still points at the wrong thing: %q", msg)
	}
	if !strings.Contains(msg, "load sales.csv as <var>") {
		t.Errorf("the message %q does not say to use load", msg)
	}
}
