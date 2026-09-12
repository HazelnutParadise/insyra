package core

import (
	"sort"
	"testing"
)

// biindex_test.go covers Assign, Set, DeleteByID and DeleteAndShift. The rest of
// the type had no test at all.

func TestBiIndex_GetHasLenIDs(t *testing.T) {
	s := NewBiIndex(0)
	idA, _ := s.Assign("a")
	idB, _ := s.Assign("b")

	if name, ok := s.Get(idA); !ok || name != "a" {
		t.Errorf("Get(%d): got (%q, %v), want (\"a\", true)", idA, name, ok)
	}
	if name, ok := s.Get(999); ok || name != "" {
		t.Errorf("Get on an unknown id: got (%q, %v), want (\"\", false)", name, ok)
	}
	if !s.Has("a") {
		t.Error("Has(\"a\") = false, want true")
	}
	if s.Has("nope") {
		t.Error("Has on an unknown name = true, want false")
	}
	if got := s.Len(); got != 2 {
		t.Errorf("Len: got %d, want 2", got)
	}

	ids := s.IDs()
	sort.Ints(ids)
	if len(ids) != 2 || ids[0] != idA || ids[1] != idB {
		t.Errorf("IDs: got %v, want the two assigned ids %d and %d", ids, idA, idB)
	}
}

func TestBiIndex_DeleteByName(t *testing.T) {
	s := NewBiIndex(0)
	_, _ = s.Assign("a")
	idB, _ := s.Assign("b")

	if !s.DeleteByName("b") {
		t.Fatal("DeleteByName(\"b\") = false, want true")
	}
	if s.Has("b") {
		t.Error("b still present after DeleteByName")
	}
	if _, ok := s.Get(idB); ok {
		t.Errorf("id %d still resolves after DeleteByName", idB)
	}
	if s.DeleteByName("b") {
		t.Error("deleting the same name twice returned true the second time")
	}
	if s.DeleteByName("never registered") {
		t.Error("DeleteByName on an unknown name returned true")
	}

	// DeleteByName goes through DeleteByID, so the id joins the free list.
	if id, added := s.Assign("c"); !added || id != idB {
		t.Errorf("Assign after DeleteByName: got id %d (added %v), want the freed id %d", id, added, idB)
	}
}

func TestBiIndex_RejectsEmptyNameAndNegativeID(t *testing.T) {
	s := NewBiIndex(0)

	if id, added := s.Assign(""); added || id != -1 {
		t.Errorf("Assign(\"\"): got (%d, %v), want (-1, false)", id, added)
	}
	if prev, ok := s.Set(-1, "a"); ok || prev != "" {
		t.Errorf("Set(-1, \"a\"): got (%q, %v), want (\"\", false)", prev, ok)
	}
	if prev, ok := s.Set(0, ""); ok || prev != "" {
		t.Errorf("Set(0, \"\"): got (%q, %v), want (\"\", false)", prev, ok)
	}
	if got := s.Len(); got != 0 {
		t.Errorf("Len after three rejected calls: got %d, want 0", got)
	}
}

// Set moves a name off whatever id it used to hold, and evicts whoever held the
// target id. Both halves have to happen or the two maps disagree.
func TestBiIndex_SetMovesAndEvicts(t *testing.T) {
	s := NewBiIndex(0)
	idA, _ := s.Assign("a") // 0
	_, _ = s.Assign("b")    // 1

	prev, ok := s.Set(idA, "b")
	if !ok {
		t.Fatal("Set failed")
	}
	if prev != "a" {
		t.Errorf("Set returned the previous occupant %q, want \"a\"", prev)
	}
	if s.Has("a") {
		t.Error("the evicted name \"a\" is still present")
	}
	if id, ok := s.Index("b"); !ok || id != idA {
		t.Errorf("b should now be at id %d, got (%d, %v)", idA, id, ok)
	}
	if name, ok := s.Get(1); ok {
		t.Errorf("b's old id 1 still resolves to %q", name)
	}
	if got := s.Len(); got != 1 {
		t.Errorf("Len after the move: got %d, want 1", got)
	}
}

// Setting an id that is sitting on the free list has to take it back off, or a
// later Assign would hand out an id that is already in use.
func TestBiIndex_SetReclaimsAFreedID(t *testing.T) {
	s := NewBiIndex(0)
	_, _ = s.Assign("a")
	idB, _ := s.Assign("b")
	_, _ = s.DeleteByID(idB) // idB is now on the free list

	if _, ok := s.Set(idB, "c"); !ok {
		t.Fatal("Set on a freed id failed")
	}

	id, added := s.Assign("d")
	if !added {
		t.Fatal("d should be newly added")
	}
	if id == idB {
		t.Errorf("Assign handed out id %d, which Set had already given to \"c\"", id)
	}
	if name, _ := s.Get(idB); name != "c" {
		t.Errorf("id %d now holds %q, want \"c\"", idB, name)
	}
}

func TestBiIndex_Clear(t *testing.T) {
	s := NewBiIndex(0)
	_, _ = s.Assign("a")
	idB, _ := s.Assign("b")
	_, _ = s.DeleteByID(idB) // leave something on the free list too

	s.Clear()

	if got := s.Len(); got != 0 {
		t.Errorf("Len after Clear: got %d, want 0", got)
	}
	if s.Has("a") {
		t.Error("\"a\" survived Clear")
	}
	if got := s.IDs(); len(got) != 0 {
		t.Errorf("IDs after Clear: got %v, want empty", got)
	}
	// Ids start from 0 again, and the stale free-list entry is gone.
	if id, added := s.Assign("z"); !added || id != 0 {
		t.Errorf("Assign after Clear: got (%d, %v), want (0, true)", id, added)
	}
}

func TestBiIndex_CloneIsIndependent(t *testing.T) {
	s := NewBiIndex(0)
	idA, _ := s.Assign("a")
	idB, _ := s.Assign("b")
	_, _ = s.DeleteByID(idB) // put idB on the free list so the clone must copy it

	c := s.Clone()

	if name, ok := c.Get(idA); !ok || name != "a" {
		t.Errorf("clone lost id %d: got (%q, %v)", idA, name, ok)
	}
	if got, want := c.Len(), s.Len(); got != want {
		t.Errorf("clone Len: got %d, want %d", got, want)
	}
	// The clone inherited the free list, so it reuses idB rather than minting a new id.
	if id, added := c.Assign("fromClone"); !added || id != idB {
		t.Errorf("clone Assign: got (%d, %v), want the freed id %d", id, added, idB)
	}

	// Changes on either side must not reach the other.
	_, _ = s.Assign("onlyInOriginal")
	if c.Has("onlyInOriginal") {
		t.Error("a name added to the original appeared in the clone")
	}
	if s.Has("fromClone") {
		t.Error("a name added to the clone appeared in the original")
	}
}

func TestBiIndex_CloneOfNil(t *testing.T) {
	var s *BiIndex

	c := s.Clone()

	if c == nil {
		t.Fatal("Clone on a nil receiver returned nil")
	}
	if got := c.Len(); got != 0 {
		t.Errorf("Len of a nil clone: got %d, want 0", got)
	}
	if id, added := c.Assign("a"); !added || id != 0 {
		t.Errorf("the nil clone is not usable: Assign gave (%d, %v)", id, added)
	}
}

// Assigning a name that is already registered returns the existing id and
// reports that nothing was added.
func TestBiIndex_AssignExistingName(t *testing.T) {
	s := NewBiIndex(0)
	id1, added1 := s.Assign("a")
	id2, added2 := s.Assign("a")

	if !added1 {
		t.Error("the first Assign should report the name as newly added")
	}
	if added2 {
		t.Error("the second Assign reported the name as newly added")
	}
	if id1 != id2 {
		t.Errorf("the same name got two ids: %d and %d", id1, id2)
	}
	if got := s.Len(); got != 1 {
		t.Errorf("Len: got %d, want 1", got)
	}
}
