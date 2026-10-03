package commands

import (
	"testing"
)

// A flag the shell cannot register would panic in BuildCobraCommands, so
// Register refuses it up front.
func TestRegisterRefusesAFlagTheShellCannotTake(t *testing.T) {
	oldRegistry := Registry
	Registry = map[string]*CommandHandler{}
	t.Cleanup(func() { Registry = oldRegistry })
	run := func(ctx *ExecContext, args []string) error { return nil }
	for _, flags := range [][]CommandFlag{
		{{Name: ""}},
		{{Name: "force"}, {Name: "force", TakesValue: true}},
	} {
		if err := Register(&CommandHandler{Name: "bad", Run: run, Flags: flags}); err == nil {
			t.Errorf("Register accepted flags %+v", flags)
		}
	}
	if _, ok := Registry["bad"]; ok {
		t.Error("a refused command was registered")
	}
}
