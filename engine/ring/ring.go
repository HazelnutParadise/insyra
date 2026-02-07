package ring

import "github.com/HazelnutParadise/insyra/internal/core"

// Ring is a non-thread-safe circular buffer with dynamic growth.
// Suitable for building higher-level queues or error rings.
type Ring[T any] = core.Ring[T]

// NewRing creates a ring with the given initial capacity.
func NewRing[T any](capacity int) *Ring[T] {
	return core.NewRing[T](capacity)
}
