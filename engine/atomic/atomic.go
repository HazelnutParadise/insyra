package atomic

import "github.com/HazelnutParadise/insyra/internal/core"

// Group defines a reentrancy scope for AtomicDo.
type Group = core.AtomicGroup

// Actor provides actor-style serialized execution for any struct.
type Actor = core.AtomicActor

// NewGroup creates a new Group.
func NewGroup() *Group {
	return core.NewAtomicGroup()
}

// DefaultGroup returns the package-wide default group.
func DefaultGroup() *Group {
	return core.DefaultAtomicGroup()
}

// NewActor creates a new Actor bound to the provided group.
// If group is nil, DefaultGroup is used.
func NewActor(group *Group) *Actor {
	return core.NewAtomicActor(group)
}

// AtomicDo executes f with actor-style serialization.
func AtomicDo[T any](actor *Actor, owner *T, f func(*T)) {
	core.AtomicDo(actor, owner, f)
}

// AtomicDoWithInit executes f with actor-style serialization and runs initHook once.
func AtomicDoWithInit[T any](actor *Actor, owner *T, f func(*T), initHook func()) {
	core.AtomicDoWithInit(actor, owner, f, initHook)
}
