package core

import (
	"reflect"
	"testing"
)

func TestRing_Empty(t *testing.T) {
	r := NewRing[int](4)

	if got := r.Len(); got != 0 {
		t.Errorf("Len on a new ring: got %d, want 0", got)
	}
	if v, ok := r.Get(0); ok || v != 0 {
		t.Errorf("Get(0) on an empty ring: got (%v, %v), want (0, false)", v, ok)
	}
	if v, ok := r.PopFront(); ok || v != 0 {
		t.Errorf("PopFront on an empty ring: got (%v, %v), want (0, false)", v, ok)
	}
	if v, ok := r.PopBack(); ok || v != 0 {
		t.Errorf("PopBack on an empty ring: got (%v, %v), want (0, false)", v, ok)
	}
	if v, ok := r.DeleteAt(0); ok || v != 0 {
		t.Errorf("DeleteAt(0) on an empty ring: got (%v, %v), want (0, false)", v, ok)
	}
	if got := r.ToSlice(); len(got) != 0 {
		t.Errorf("ToSlice on an empty ring: got %v, want empty", got)
	}
}

// A capacity below 1 would make the modulo arithmetic divide by zero, so the
// constructor raises it to 1.
func TestRing_CapacityBelowOne(t *testing.T) {
	for _, capacity := range []int{0, -1, -100} {
		r := NewRing[string](capacity)
		r.Push("a")
		r.Push("b")
		if got, want := r.ToSlice(), []string{"a", "b"}; !reflect.DeepEqual(got, want) {
			t.Errorf("NewRing(%d) then two pushes: got %v, want %v", capacity, got, want)
		}
	}
}

func TestRing_GetOutOfRange(t *testing.T) {
	r := NewRing[int](4)
	r.Push(10)
	r.Push(20)

	if v, ok := r.Get(-1); ok || v != 0 {
		t.Errorf("Get(-1): got (%v, %v), want (0, false)", v, ok)
	}
	if v, ok := r.Get(2); ok || v != 0 {
		t.Errorf("Get(2) with two elements: got (%v, %v), want (0, false)", v, ok)
	}
}

// Pushing past the initial capacity grows the buffer; the elements keep their
// order and their logical indices.
func TestRing_GrowsPastCapacity(t *testing.T) {
	r := NewRing[int](2)
	want := make([]int, 0, 10)
	for i := 0; i < 10; i++ {
		r.Push(i)
		want = append(want, i)
	}

	if got := r.Len(); got != 10 {
		t.Errorf("Len after 10 pushes into a capacity-2 ring: got %d, want 10", got)
	}
	if got := r.ToSlice(); !reflect.DeepEqual(got, want) {
		t.Errorf("ToSlice after growth: got %v, want %v", got, want)
	}
	for i, w := range want {
		if v, ok := r.Get(i); !ok || v != w {
			t.Errorf("Get(%d): got (%v, %v), want (%v, true)", i, v, ok, w)
		}
	}
}

// After a PopFront the head moves, so the next Push lands before the head in the
// backing array. This is the wrap-around case: everything below indexes through
// the modulo, and a bug there shows up as the wrong order, not a panic.
func TestRing_WrapsAround(t *testing.T) {
	r := NewRing[int](3)
	r.Push(1)
	r.Push(2)
	r.Push(3)

	if v, ok := r.PopFront(); !ok || v != 1 {
		t.Fatalf("PopFront: got (%v, %v), want (1, true)", v, ok)
	}
	r.Push(4) // wraps into the slot 1 left behind

	if got, want := r.ToSlice(), []int{2, 3, 4}; !reflect.DeepEqual(got, want) {
		t.Errorf("after wrap-around: got %v, want %v", got, want)
	}

	// Grow while wrapped: the copy has to walk logical order, not buffer order.
	r.Push(5)
	if got, want := r.ToSlice(), []int{2, 3, 4, 5}; !reflect.DeepEqual(got, want) {
		t.Errorf("after growing a wrapped ring: got %v, want %v", got, want)
	}
}

func TestRing_PopFrontAndBack(t *testing.T) {
	r := NewRing[int](8)
	for i := 1; i <= 5; i++ {
		r.Push(i)
	}

	if v, ok := r.PopFront(); !ok || v != 1 {
		t.Errorf("PopFront: got (%v, %v), want (1, true)", v, ok)
	}
	if v, ok := r.PopBack(); !ok || v != 5 {
		t.Errorf("PopBack: got (%v, %v), want (5, true)", v, ok)
	}
	if got, want := r.ToSlice(), []int{2, 3, 4}; !reflect.DeepEqual(got, want) {
		t.Errorf("after one PopFront and one PopBack: got %v, want %v", got, want)
	}

	// Drain completely, then confirm the ring is reusable.
	for r.Len() > 0 {
		if _, ok := r.PopFront(); !ok {
			t.Fatal("PopFront returned false while Len > 0")
		}
	}
	r.Push(42)
	if got, want := r.ToSlice(), []int{42}; !reflect.DeepEqual(got, want) {
		t.Errorf("after draining and pushing again: got %v, want %v", got, want)
	}
}

func TestRing_DeleteAt(t *testing.T) {
	tests := []struct {
		name string
		idx  int
		want []int
		val  int
		ok   bool
	}{
		{name: "front", idx: 0, want: []int{2, 3, 4}, val: 1, ok: true},
		{name: "middle", idx: 1, want: []int{1, 3, 4}, val: 2, ok: true},
		{name: "back", idx: 3, want: []int{1, 2, 3}, val: 4, ok: true},
		{name: "past the end", idx: 4, want: []int{1, 2, 3, 4}, val: 0, ok: false},
		{name: "negative", idx: -1, want: []int{1, 2, 3, 4}, val: 0, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRing[int](4)
			for i := 1; i <= 4; i++ {
				r.Push(i)
			}
			v, ok := r.DeleteAt(tt.idx)
			if v != tt.val || ok != tt.ok {
				t.Errorf("DeleteAt(%d): got (%v, %v), want (%v, %v)", tt.idx, v, ok, tt.val, tt.ok)
			}
			if got := r.ToSlice(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("after DeleteAt(%d): got %v, want %v", tt.idx, got, tt.want)
			}
		})
	}
}

// DeleteAt has to work on a ring whose head is not at 0.
func TestRing_DeleteAtWhileWrapped(t *testing.T) {
	r := NewRing[int](3)
	r.Push(1)
	r.Push(2)
	r.Push(3)
	_, _ = r.PopFront()
	r.Push(4) // logical order is now 2, 3, 4 with head at index 1

	if v, ok := r.DeleteAt(1); !ok || v != 3 {
		t.Errorf("DeleteAt(1) on a wrapped ring: got (%v, %v), want (3, true)", v, ok)
	}
	if got, want := r.ToSlice(), []int{2, 4}; !reflect.DeepEqual(got, want) {
		t.Errorf("after DeleteAt on a wrapped ring: got %v, want %v", got, want)
	}
}

func TestRing_Clear(t *testing.T) {
	r := NewRing[int](4)
	for i := 1; i <= 6; i++ { // forces a grow, so capacity is 8 by now
		r.Push(i)
	}

	r.Clear()

	if got := r.Len(); got != 0 {
		t.Errorf("Len after Clear: got %d, want 0", got)
	}
	if v, ok := r.Get(0); ok || v != 0 {
		t.Errorf("Get(0) after Clear: got (%v, %v), want (0, false)", v, ok)
	}
	r.Push(99)
	if got, want := r.ToSlice(), []int{99}; !reflect.DeepEqual(got, want) {
		t.Errorf("after Clear and one Push: got %v, want %v", got, want)
	}
}

// ToSlice hands out a copy: writing to it must not reach back into the ring.
func TestRing_ToSliceIsACopy(t *testing.T) {
	r := NewRing[int](4)
	r.Push(1)
	r.Push(2)

	s := r.ToSlice()
	s[0] = 999

	if v, _ := r.Get(0); v != 1 {
		t.Errorf("writing to the ToSlice result changed the ring: Get(0) = %v, want 1", v)
	}
}

// A removed element must not stay reachable from the backing array, or a ring of
// pointers keeps whatever they point to alive after it is popped.
func TestRingPopsLeaveNoReferenceBehind(t *testing.T) {
	type payload struct{ n int }
	newRing := func() *Ring[*payload] {
		r := NewRing[*payload](4)
		for i := 1; i <= 3; i++ {
			r.Push(&payload{i})
		}
		return r
	}
	nonNil := func(r *Ring[*payload]) int {
		count := 0
		for _, p := range r.buf {
			if p != nil {
				count++
			}
		}
		return count
	}

	r := newRing()
	if v, ok := r.PopFront(); !ok || v.n != 1 {
		t.Fatalf("PopFront = %v, %v", v, ok)
	}
	if got := nonNil(r); got != 2 {
		t.Errorf("after PopFront %d slots hold a reference, want 2", got)
	}

	r = newRing()
	if v, ok := r.PopBack(); !ok || v.n != 3 {
		t.Fatalf("PopBack = %v, %v", v, ok)
	}
	if got := nonNil(r); got != 2 {
		t.Errorf("after PopBack %d slots hold a reference, want 2", got)
	}
	if got := r.ToSlice(); len(got) != 2 || got[0].n != 1 || got[1].n != 2 {
		t.Errorf("the rest of the ring changed: %v", got)
	}

	// Wrapped around, grown, and a single element, alternating the two ends.
	r = NewRing[*payload](2)
	for i := 1; i <= 6; i++ {
		r.Push(&payload{i})
		if i%2 == 0 {
			r.PopFront()
		}
	}
	for r.Len() > 0 {
		r.PopBack()
		if r.Len() > 0 {
			r.PopFront()
		}
	}
	if got := nonNil(r); got != 0 {
		t.Errorf("an emptied ring still holds %d references", got)
	}
	if _, ok := r.PopFront(); ok {
		t.Error("PopFront on an empty ring returned a value")
	}
	if _, ok := r.PopBack(); ok {
		t.Error("PopBack on an empty ring returned a value")
	}
}

func TestRingClearEmptiesTheSlotsAndKeepsTheArray(t *testing.T) {
	type payload struct{ n int }
	r := NewRing[*payload](8)
	for i := 0; i < 8; i++ {
		r.Push(&payload{i})
	}
	r.PopFront()
	r.PopFront()
	r.Push(&payload{9}) // the elements now wrap around the end of the array
	before := &r.buf[0]
	capBefore := len(r.buf)

	r.Clear()
	if r.Len() != 0 || len(r.buf) != capBefore {
		t.Errorf("after Clear: len %d, capacity %d, want 0 and %d", r.Len(), len(r.buf), capBefore)
	}
	if &r.buf[0] != before {
		t.Error("Clear allocated a new backing array")
	}
	for i, p := range r.buf {
		if p != nil {
			t.Errorf("slot %d still holds a reference after Clear", i)
		}
	}
	r.Push(&payload{1})
	r.Push(&payload{2})
	if v, _ := r.PopFront(); v.n != 1 {
		t.Errorf("the ring does not work after Clear: front is %v", v)
	}
}

func TestRingClearOfAnAllocatedRingDoesNotAllocate(t *testing.T) {
	r := NewRing[*int](16)
	x := 1
	allocs := testing.AllocsPerRun(100, func() {
		for i := 0; i < 16; i++ {
			r.Push(&x)
		}
		r.Clear()
	})
	if allocs != 0 {
		t.Errorf("Push and Clear allocated %v times, want 0", allocs)
	}
}

// A zero Ring is usable, and clearing it gives it a buffer.
func TestRingClearOfAZeroRing(t *testing.T) {
	var r Ring[int]
	r.Clear()
	r.Push(7)
	if v, ok := r.PopFront(); !ok || v != 7 {
		t.Errorf("PopFront after Clear of a zero ring = %v, %v", v, ok)
	}
}

func BenchmarkRingPushPop(b *testing.B) {
	r := NewRing[*int](64)
	x := 1
	b.ReportAllocs()
	for b.Loop() {
		for i := 0; i < 64; i++ {
			r.Push(&x)
		}
		for i := 0; i < 64; i++ {
			r.PopFront()
		}
	}
}

func BenchmarkRingPushClear(b *testing.B) {
	r := NewRing[*int](64)
	x := 1
	b.ReportAllocs()
	for b.Loop() {
		for i := 0; i < 64; i++ {
			r.Push(&x)
		}
		r.Clear()
	}
}
