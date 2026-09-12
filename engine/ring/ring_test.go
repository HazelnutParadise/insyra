package ring_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra/engine/ring"
)

// A thin re-export of internal/core.Ring, and the only way a consumer outside
// the module can reach it. What needs checking is that the wrapper forwards
// correctly and that the alias is usable from another package.

func TestNewRingForwards(t *testing.T) {
	r := ring.NewRing[string](2)

	if got := r.Len(); got != 0 {
		t.Fatalf("Len on a new ring: got %d, want 0", got)
	}
	r.Push("a")
	r.Push("b")
	r.Push("c") // past the initial capacity

	if got := r.Len(); got != 3 {
		t.Errorf("Len: got %d, want 3", got)
	}
	if v, ok := r.Get(1); !ok || v != "b" {
		t.Errorf("Get(1): got (%q, %v), want (\"b\", true)", v, ok)
	}
	if v, ok := r.PopFront(); !ok || v != "a" {
		t.Errorf("PopFront: got (%q, %v), want (\"a\", true)", v, ok)
	}
	if v, ok := r.PopBack(); !ok || v != "c" {
		t.Errorf("PopBack: got (%q, %v), want (\"c\", true)", v, ok)
	}
}

// The alias is a type, not a wrapper struct, so a *ring.Ring[T] can be declared
// and passed around by a consumer.
// takesIntRing exists so the alias is spelled out somewhere the compiler checks.
func takesIntRing(*ring.Ring[int]) {}

func TestRingTypeAlias(t *testing.T) {
	r := ring.NewRing[int](1)
	// The alias has to name the same type the constructor returns. This call
	// only has to compile.
	takesIntRing(r)
	r.Push(7)
	if got := r.ToSlice(); len(got) != 1 || got[0] != 7 {
		t.Errorf("ToSlice: got %v, want [7]", got)
	}
	r.Clear()
	if got := r.Len(); got != 0 {
		t.Errorf("Len after Clear: got %d, want 0", got)
	}
}
