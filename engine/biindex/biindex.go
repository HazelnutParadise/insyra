package biindex

import "github.com/HazelnutParadise/insyra/internal/core"

// BiIndex is a two-way index used by DataTable internals.
type BiIndex = core.BiIndex

// NewBiIndex creates a new BiIndex with the given capacity hint.
func NewBiIndex(cap int) *BiIndex {
	return core.NewBiIndex(cap)
}
