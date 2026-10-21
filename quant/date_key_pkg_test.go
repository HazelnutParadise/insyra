package quant

import (
	"testing"

	xa "github.com/HazelnutParadise/insyra/quant/internal/datekeytest/a/x"
	xb "github.com/HazelnutParadise/insyra/quant/internal/datekeytest/b/x"
)

// Both types print as "x.Day"; their import paths differ, so their keys must too.
func TestDateKeySameNamedTypesInDifferentPackages(t *testing.T) {
	ka, err := dateKey(xa.Day(1))
	if err != nil {
		t.Fatal(err)
	}
	kb, err := dateKey(xb.Day(1))
	if err != nil {
		t.Fatal(err)
	}
	if ka == kb {
		t.Errorf("types from different packages share the key %q", ka)
	}
}
