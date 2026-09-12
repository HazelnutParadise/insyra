package biindex_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra/engine/biindex"
)

// A thin re-export of internal/core.BiIndex. Checks the forwarder and that the
// alias is usable from a consumer's package.

// takesBiIndex exists so the alias is spelled out somewhere the compiler checks.
func takesBiIndex(*biindex.BiIndex) {}

func TestNewBiIndexForwards(t *testing.T) {
	idx := biindex.NewBiIndex(4)
	// The alias has to name the same type the constructor returns, which is the
	// whole point of re-exporting it. This call only has to compile.
	takesBiIndex(idx)

	id, added := idx.Assign("first")
	if !added {
		t.Fatal("the first Assign reported the name as already present")
	}
	if name, ok := idx.Get(id); !ok || name != "first" {
		t.Errorf("Get(%d): got (%q, %v), want (\"first\", true)", id, name, ok)
	}
	if got, ok := idx.Index("first"); !ok || got != id {
		t.Errorf("Index(\"first\"): got (%d, %v), want (%d, true)", got, ok, id)
	}
	if !idx.Has("first") {
		t.Error("Has(\"first\") is false")
	}
	if got := idx.Len(); got != 1 {
		t.Errorf("Len: got %d, want 1", got)
	}

	// The capacity hint is a hint, not a limit.
	for i := 0; i < 10; i++ {
		idx.Assign(string(rune('a' + i)))
	}
	if got := idx.Len(); got != 11 {
		t.Errorf("Len after ten more names: got %d, want 11", got)
	}
}

// A negative capacity hint is clamped rather than rejected.
func TestNewBiIndexNegativeCapacity(t *testing.T) {
	idx := biindex.NewBiIndex(-1)
	if id, added := idx.Assign("x"); !added || id != 0 {
		t.Errorf("Assign on a negative-capacity index: got (%d, %v), want (0, true)", id, added)
	}
}
