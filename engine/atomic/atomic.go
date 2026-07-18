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

// AtomicDoN runs f with the locks of ALL given actors held together, acquired in
// a canonical (pointer-address) order and released in reverse. This is the
// deadlock-free way to operate on several instances atomically — use it instead
// of nesting AtomicDo on different instances (which does NOT lock the inner one).
//
// It is a general-purpose primitive: any struct that holds an *Actor (via
// NewActor) gets per-instance serialization (AtomicDo) AND deadlock-free
// multi-instance locking (AtomicDoN). Actors may belong to different groups; nil
// or already-held actors are skipped and duplicate pointers de-duplicated.
//
//	// Lock two custom structs together, deadlock-free:
//	atomic.AtomicDoN([]*atomic.Actor{a.actor, b.actor}, func() {
//		b.data = append(b.data, a.data...)
//	})
func AtomicDoN(actors []*Actor, f func()) {
	core.AtomicDoN(actors, f)
}

// AtomicDoNWithInit is AtomicDoN plus a per-actor one-time init hook (initHooks[i]
// pairs with actors[i]); nil entries and short slices are fine.
func AtomicDoNWithInit(actors []*Actor, initHooks []func(), f func()) {
	core.AtomicDoNWithInit(actors, initHooks, f)
}
