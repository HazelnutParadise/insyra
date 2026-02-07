package core

// Ring is a non-thread-safe circular buffer with dynamic growth.
// It is suitable for building higher-level queues or error rings.
type Ring[T any] struct {
	buf  []T
	head int
	size int
}

// NewRing creates a ring with the given initial capacity.
func NewRing[T any](capacity int) *Ring[T] {
	if capacity < 1 {
		capacity = 1
	}
	return &Ring[T]{buf: make([]T, capacity)}
}

// Len returns the number of elements currently in the ring.
func (r *Ring[T]) Len() int {
	return r.size
}

// Get returns the element at logical index i (0..Len-1).
func (r *Ring[T]) Get(i int) (T, bool) {
	var zero T
	if i < 0 || i >= r.size || len(r.buf) == 0 {
		return zero, false
	}
	return r.buf[(r.head+i)%len(r.buf)], true
}

// ToSlice returns a copy of the ring contents in logical order.
func (r *Ring[T]) ToSlice() []T {
	out := make([]T, r.size)
	for i := 0; i < r.size; i++ {
		out[i], _ = r.Get(i)
	}
	return out
}

// Clear removes all elements while keeping the capacity.
func (r *Ring[T]) Clear() {
	if len(r.buf) == 0 {
		r.buf = make([]T, 1)
	} else {
		r.buf = make([]T, len(r.buf))
	}
	r.head = 0
	r.size = 0
}

// Push adds an element to the back of the ring.
func (r *Ring[T]) Push(v T) {
	if len(r.buf) == 0 {
		r.buf = make([]T, 1)
	}
	if r.size == len(r.buf) {
		r.grow()
	}
	idx := (r.head + r.size) % len(r.buf)
	r.buf[idx] = v
	r.size++
}

// PopFront removes and returns the front element.
func (r *Ring[T]) PopFront() (T, bool) {
	var zero T
	if r.size == 0 {
		return zero, false
	}
	v, _ := r.Get(0)
	r.head = (r.head + 1) % len(r.buf)
	r.size--
	return v, true
}

// PopBack removes and returns the last element.
func (r *Ring[T]) PopBack() (T, bool) {
	var zero T
	if r.size == 0 {
		return zero, false
	}
	backIdx := (r.head + r.size - 1) % len(r.buf)
	v := r.buf[backIdx]
	r.size--
	return v, true
}

// DeleteAt removes the element at logical index idx.
func (r *Ring[T]) DeleteAt(idx int) (T, bool) {
	var zero T
	if idx < 0 || idx >= r.size {
		return zero, false
	}
	newBuf := make([]T, len(r.buf))
	n := 0
	var removed T
	for i := 0; i < r.size; i++ {
		if i == idx {
			removed, _ = r.Get(i)
			continue
		}
		newBuf[n], _ = r.Get(i)
		n++
	}
	r.buf = newBuf
	r.head = 0
	r.size = n
	return removed, true
}

func (r *Ring[T]) grow() {
	newCap := 1
	if len(r.buf) > 0 {
		newCap = len(r.buf) * 2
	}
	newBuf := make([]T, newCap)
	for i := 0; i < r.size; i++ {
		newBuf[i], _ = r.Get(i)
	}
	r.buf = newBuf
	r.head = 0
}
