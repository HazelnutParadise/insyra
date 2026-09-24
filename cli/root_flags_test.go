package cli

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/cli/env"
)

// CLI-3: root flags placed before a DisableFlagParsing subcommand still
// apply, instead of being swallowed as data.
func TestRootFlagsBeforeRawArgCommand(t *testing.T) {
	base := t.TempDir()
	env.SetBasePath(base)
	t.Cleanup(func() { env.SetBasePath("") })

	if err := env.Default().Create("e2"); err != nil {
		t.Fatal(err)
	}
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"--env", "e2", "newdl", "1", "2", "3", "as", "ex"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	vars, err := env.Default().RestoreVariables("e2")
	if err != nil {
		t.Fatalf("environment e2 was not written: %v", err)
	}
	dl, ok := vars["ex"].(*insyra.DataList)
	if !ok {
		t.Fatalf("ex not stored in e2: %v", vars)
	}
	if dl.Len() != 3 {
		t.Fatalf("ex has %d items, want 3 (flags leaked into data: %v)", dl.Len(), dl.Data())
	}
	if defVars, err := env.Default().RestoreVariables("default"); err == nil {
		if _, leaked := defVars["ex"]; leaked {
			t.Fatal("ex was written to the default environment")
		}
	}
}
