package commands

import (
	"testing"
)

func TestNewDLAddRowAddColDisableFlagParsing(t *testing.T) {
	for _, name := range []string{"newdl", "addrow", "addcol"} {
		handler, ok := Registry[name]
		if !ok {
			t.Fatalf("%s is not registered", name)
		}
		if !handler.DisableFlagParsing {
			t.Errorf("%s should set DisableFlagParsing so negative literals survive", name)
		}
	}
}
