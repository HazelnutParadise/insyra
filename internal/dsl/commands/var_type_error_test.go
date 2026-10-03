package commands

import (
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// CLI-19 of #327: clone, replace, clean, fillna and count try a DataTable then
// a DataList, and told the caller "variable not found" when the variable was
// right there holding something else — sending them looking for a typo that was
// not there.
func TestCommandsTellAMissingVariableFromAWrongOne(t *testing.T) {
	commands := []struct {
		name string
		args []string
	}{
		{name: "clone", args: []string{"x", "as", "y"}},
		{name: "clean", args: []string{"x", "nan"}},
		{name: "replace", args: []string{"x", "1", "2"}},
		{name: "fillna", args: []string{"x", "mean"}},
		{name: "count", args: []string{"x", "1"}},
	}

	for _, c := range commands {
		t.Run(c.name+" missing", func(t *testing.T) {
			ctx := newTestExecContext(t)
			err := Dispatch(ctx, c.name, c.args)
			if err == nil {
				t.Fatal("a missing variable gave no error")
			}
			if !strings.Contains(err.Error(), "not found") {
				t.Errorf("error %q does not say the variable is missing", err)
			}
		})

		t.Run(c.name+" wrong type", func(t *testing.T) {
			ctx := newTestExecContext(t)
			ctx.Vars["x"] = 42 // present, but neither a table nor a list
			err := Dispatch(ctx, c.name, c.args)
			if err == nil {
				t.Fatal("a variable of the wrong type gave no error")
			}
			if strings.Contains(err.Error(), "not found") {
				t.Errorf("error %q claims the variable is missing; it is right there", err)
			}
			if !strings.Contains(err.Error(), c.name) {
				t.Errorf("error %q does not name the command", err)
			}
		})
	}
}

// A DataTable and a DataList both still work, so the new branch only fires for
// a genuinely unusable value.
func TestCloneStillAcceptsBothKinds(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["t"] = insyra.NewDataTable(insyra.NewDataList(1, 2))
	ctx.Vars["l"] = insyra.NewDataList(1, 2)

	for _, name := range []string{"t", "l"} {
		if err := Dispatch(ctx, "clone", []string{name, "as", name + "2"}); err != nil {
			t.Errorf("clone %s: %v", name, err)
		}
	}
}
