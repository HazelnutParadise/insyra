package core

import "testing"

func TestBiIndexStability(t *testing.T) {
	s := NewBiIndex(0)
	_, _ = s.Assign("a")
	idB, _ := s.Assign("b")
	idC, _ := s.Assign("c")

	// delete middle
	_, ok := s.DeleteByID(idB)
	if !ok {
		t.Fatalf("delete failed")
	}

	// c's id should remain the same
	if id, ok := s.Index("c"); !ok || id != idC {
		t.Fatalf("c id changed: got %d want %d", id, idC)
	}
}

func TestBiIndexReuse(t *testing.T) {
	s := NewBiIndex(0)
	id1, _ := s.Assign("x")
	id2, _ := s.Assign("y")
	_, _ = s.DeleteByID(id2)
	// next assign should reuse id2
	id3, added := s.Assign("z")
	if !added {
		t.Fatalf("z should be newly added")
	}
	if id3 != id2 {
		t.Fatalf("expected reuse id %d but got %d", id2, id3)
	}
	// ensure x's id unchanged
	if id, _ := s.Index("x"); id != id1 {
		t.Fatalf("x id changed")
	}
}

func TestBiIndexSetAndEdgeCases(t *testing.T) {
	s := NewBiIndex(0)
	// set at specific id
	prev, ok := s.Set(5, "u")
	if !ok || prev != "" {
		t.Fatalf("Set failed: prev='%s' ok=%v", prev, ok)
	}
	if id, _ := s.Index("u"); id != 5 {
		t.Fatalf("u expected at 5 got %d", id)
	}

	// set existing name to different id
	s.Assign("v")
	_, oldOK := s.Index("v")
	if !oldOK {
		t.Fatalf("v should exist")
	}
	prev2, ok2 := s.Set(2, "v")
	if !ok2 {
		t.Fatalf("Set move failed")
	}
	if prev2 != "" && prev2 != "v" { // prev2 either empty or previous occupant
		t.Fatalf("unexpected prev value")
	}

	// delete non-existing
	if _, ok := s.DeleteByID(100); ok {
		t.Fatalf("delete non-existing should be false")
	}
}

func TestDeleteAndShift(t *testing.T) {
	s := NewBiIndex(0)
	_, _ = s.Assign("a")
	idB, _ := s.Assign("b")
	idC, _ := s.Assign("c")
	idD, _ := s.Assign("d")
	idE, _ := s.Assign("e")

	deleted, mapping, ok := s.DeleteAndShift(idB)
	if !ok || deleted != "b" {
		t.Fatalf("DeleteAndShift did not delete b")
	}
	// validate mapping
	if mapping == nil || mapping[idC] != idC-1 || mapping[idD] != idD-1 || mapping[idE] != idE-1 {
		t.Fatalf("unexpected mapping: %v", mapping)
	}
	// validate ids
	if id, ok := s.Index("c"); !ok || id != idC-1 {
		t.Fatalf("c id wrong: %d", id)
	}
	if id, ok := s.Index("e"); !ok || id != idE-1 {
		t.Fatalf("e id wrong: %d", id)
	}

	// ensure nextID decreased
	if s.nextID != idE {
		t.Fatalf("nextID not adjusted: got %d want %d", s.nextID, idE)
	}
}
