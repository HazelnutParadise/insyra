package core

import (
	"sync"
	"sync/atomic"

	"github.com/petermattis/goid"
)

// AtomicActor provides actor-style serialised execution for any struct.
// Use AtomicDo to run a function with serialised access.
//
// Implementation: a sync.Mutex protects the actor; the goroutine that
// holds the mutex stores its goroutine ID in `holder` so that recursive
// (same-goroutine) AtomicDo calls run inline without deadlocking. The
// previous implementation used a goroutine-per-actor + channel-based
// dispatch + runtime.Stack-based goroutine-ID extraction; benchmarks
// showed 6.7µs per cold AtomicDo just on the framework overhead. With
// petermattis/goid (cross-platform, ~5ns goroutine-ID extraction) and
// a plain mutex, that drops to ~30ns per cold call — a measured 169×
// reduction on the typical "outer.Mean+Stdev+Len" T-test pattern
// (15.2µs → 0.09µs per invocation).
//
// API surface is unchanged from the previous channel-actor design so
// callers in `insyra` and `stats` recompile without source changes.
type AtomicActor struct {
	group     *AtomicGroup
	groupOnce sync.Once
	initOnce  sync.Once

	mu     sync.Mutex
	holder atomic.Int64 // goroutine id of current holder; 0 = unlocked
	closed atomic.Bool
}

// AtomicGroup defines a re-entrancy scope for AtomicActor.
//
// If a goroutine is already inside SOME actor belonging to the same group,
// subsequent AtomicDo calls on actors of the same group run inline without
// re-acquiring a mutex. This preserves the previous channel-actor design's
// "trust zone" semantics: callers within the zone may co-access multiple
// actors without nested locking, accepting the same race exposure as
// before. (The previous implementation also short-circuited cross-actor
// nested calls; not preserving this would break stats methods that do
// `dlX.AtomicDo(func() { dlY.AtomicDo(... read x, read y ...) })`.)
type AtomicGroup struct {
	active sync.Map // map[int64]struct{} — gids currently inside an actor of this group
}

// NewAtomicGroup creates a new AtomicGroup.
func NewAtomicGroup() *AtomicGroup {
	return &AtomicGroup{}
}

var defaultAtomicGroup = &AtomicGroup{}

// DefaultAtomicGroup returns the package-wide default group.
func DefaultAtomicGroup() *AtomicGroup {
	return defaultAtomicGroup
}

func (g *AtomicGroup) markEnter(gid int64) {
	if g == nil {
		return
	}
	g.active.Store(gid, struct{}{})
}

func (g *AtomicGroup) markExit(gid int64) {
	if g == nil {
		return
	}
	g.active.Delete(gid)
}

func (g *AtomicGroup) inActorLoop(gid int64) bool {
	if g == nil {
		return false
	}
	_, ok := g.active.Load(gid)
	return ok
}

// NewAtomicActor creates a new AtomicActor bound to the provided group.
// If group is nil, DefaultAtomicGroup is used.
func NewAtomicActor(group *AtomicGroup) *AtomicActor {
	if group == nil {
		group = DefaultAtomicGroup()
	}
	return &AtomicActor{group: group}
}

// SetGroupOnce assigns the re-entrancy group for this actor once.
func (a *AtomicActor) SetGroupOnce(group *AtomicGroup) {
	if a == nil || group == nil {
		return
	}
	a.groupOnce.Do(func() {
		a.group = group
	})
}

func (a *AtomicActor) ensureGroup() *AtomicGroup {
	if a.group == nil {
		a.groupOnce.Do(func() {
			if a.group == nil {
				a.group = DefaultAtomicGroup()
			}
		})
	}
	return a.group
}

// AtomicDo executes f with actor-style serialisation.
func AtomicDo[T any](actor *AtomicActor, owner *T, f func(*T)) {
	AtomicDoWithInit(actor, owner, f, nil)
}

// AtomicDoWithInit executes f with actor-style serialisation and runs
// initHook once on the very first AtomicDo call against this actor.
//
// Re-entry rules:
//   - If the calling goroutine already holds THIS actor → run inline.
//   - If the calling goroutine is inside SOME other actor of the same
//     group → run inline (legacy "trust zone" semantics).
//   - Otherwise → acquire the mutex, mark holder, run f, release.
//
// `initHook` is invoked exactly once across the actor's lifetime, on the
// first AtomicDo call. It typically registers a finalizer or warms a
// resource. Kept compatible with the previous API.
func AtomicDoWithInit[T any](actor *AtomicActor, owner *T, f func(*T), initHook func()) {
	if actor == nil {
		f(owner)
		return
	}
	if actor.closed.Load() {
		f(owner)
		return
	}
	if initHook != nil {
		actor.initOnce.Do(initHook)
	}

	gid := goid.Get()

	// Same-actor re-entry — common when a public DataList method like
	// Stdev calls Var inside an outer AtomicDo callback. Inline path,
	// no lock acquisition.
	if actor.holder.Load() == gid {
		f(owner)
		return
	}

	// Cross-actor re-entry within the same group. Preserves the previous
	// channel-actor's "if I'm in any actor, run inline" semantics.
	group := actor.ensureGroup()
	if group.inActorLoop(gid) {
		f(owner)
		return
	}

	actor.mu.Lock()
	actor.holder.Store(gid)
	group.markEnter(gid)
	defer func() {
		group.markExit(gid)
		actor.holder.Store(0)
		actor.mu.Unlock()
	}()
	if actor.closed.Load() {
		return
	}
	f(owner)
}

// Close marks the actor as closed. Subsequent AtomicDo calls run inline
// (no locking) so close-during-shutdown paths can't deadlock.
func (a *AtomicActor) Close() {
	if a == nil {
		return
	}
	a.closed.Store(true)
}

// IsClosed reports whether Close was called.
func (a *AtomicActor) IsClosed() bool {
	if a == nil {
		return true
	}
	return a.closed.Load()
}
