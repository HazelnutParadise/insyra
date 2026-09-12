package core

import "testing"

// IN-16 of #338: Set(id, name) for a name that already belongs to another id
// deleted the old mapping without freeing the id, so that id became a hole no
// Assign could ever hand out again.
func TestBiIndex_SetReleasesTheNameOldID(t *testing.T) {
	s := NewBiIndex(0)
	idA, _ := s.Assign("a") // 0
	idB, _ := s.Assign("b") // 1

	// Move "b" onto a's id. b's old id is now unused.
	if _, ok := s.Set(idA, "b"); !ok {
		t.Fatal("Set failed")
	}
	if _, ok := s.Get(idB); ok {
		t.Fatalf("id %d still resolves", idB)
	}

	// It has to come back, or every such move leaks an id.
	id, added := s.Assign("c")
	if !added {
		t.Fatal("c should be newly added")
	}
	if id != idB {
		t.Errorf("Assign gave id %d, want the released id %d", id, idB)
	}
	if got := s.Len(); got != 2 {
		t.Errorf("Len: got %d, want 2", got)
	}
}

// Releasing it must not hand the same id out twice.
func TestBiIndex_ReleasedIDIsHandedOutOnce(t *testing.T) {
	s := NewBiIndex(0)
	_, _ = s.Assign("a") // 0
	idB, _ := s.Assign("b")
	_, _ = s.Assign("c")

	_, _ = s.Set(0, "b") // frees idB

	seen := map[int]bool{}
	for _, name := range []string{"d", "e", "f"} {
		id, _ := s.Assign(name)
		if seen[id] {
			t.Fatalf("id %d was handed out twice", id)
		}
		seen[id] = true
	}
	if !seen[idB] {
		t.Errorf("the released id %d was never reused", idB)
	}

	// Every active name still resolves to its own id.
	for _, name := range []string{"b", "c", "d", "e", "f"} {
		id, ok := s.Index(name)
		if !ok {
			t.Errorf("%q lost its id", name)
			continue
		}
		if got, _ := s.Get(id); got != name {
			t.Errorf("id %d maps to %q, but %q maps to %d", id, got, name, id)
		}
	}
}
