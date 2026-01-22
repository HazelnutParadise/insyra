package core

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
)

// AtomicGroup defines a reentrancy scope for AtomicActor.
// If a goroutine is inside any AtomicActor belonging to the same group,
// subsequent AtomicDo calls within that goroutine will run inline.
type AtomicGroup struct {
	context sync.Map // map[uint64]bool
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

func (g *AtomicGroup) inActorLoop() bool {
	if g == nil {
		return false
	}
	gid := getGID()
	_, ok := g.context.Load(gid)
	return ok
}

// AtomicActor provides actor-style serialized execution for any struct.
// Use AtomicDo to run a function with serialized access.
type AtomicActor struct {
	group     *AtomicGroup
	groupOnce sync.Once
	initOnce  sync.Once
	cmdCh     chan func()
	closed    atomic.Bool
}

// NewAtomicActor creates a new AtomicActor bound to the provided group.
// If group is nil, DefaultAtomicGroup is used.
func NewAtomicActor(group *AtomicGroup) *AtomicActor {
	if group == nil {
		group = DefaultAtomicGroup()
	}
	return &AtomicActor{group: group}
}

// SetGroupOnce assigns the reentrancy group for this actor once.
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

// AtomicDo executes f with actor-style serialization.
func AtomicDo[T any](actor *AtomicActor, owner *T, f func(*T)) {
	AtomicDoWithInit(actor, owner, f, nil)
}

// AtomicDoWithInit executes f with actor-style serialization and runs initHook once.
func AtomicDoWithInit[T any](actor *AtomicActor, owner *T, f func(*T), initHook func()) {
	if actor == nil {
		f(owner)
		return
	}

	if actor.closed.Load() {
		f(owner)
		return
	}

	actor.initOnce.Do(func() {
		actor.cmdCh = make(chan func())
		go actor.actorLoop()
		if initHook != nil {
			initHook()
		}
	})

	group := actor.ensureGroup()
	if group.inActorLoop() {
		f(owner)
		return
	}

	if actor.closed.Load() {
		f(owner)
		return
	}

	done := make(chan struct{})
	cmdCh := actor.cmdCh
	if cmdCh == nil {
		f(owner)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			f(owner)
		}
	}()

	cmdCh <- func() {
		gid := getGID()
		group.context.Store(gid, true)
		defer group.context.Delete(gid)

		if !actor.closed.Load() {
			f(owner)
		}
		close(done)
	}
	<-done
}

// Close closes the actor loop and prevents further scheduling.
func (a *AtomicActor) Close() {
	if a == nil {
		return
	}
	if a.closed.Load() {
		return
	}
	a.closed.Store(true)

	if a.cmdCh != nil {
		close(a.cmdCh)
		a.cmdCh = nil
	}
}

// IsClosed reports whether the actor has been closed.
func (a *AtomicActor) IsClosed() bool {
	if a == nil {
		return true
	}
	return a.closed.Load()
}

func (a *AtomicActor) actorLoop() {
	for fn := range a.cmdCh {
		fn()
	}
}

func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	i := bytes.IndexByte(b, ' ')
	id, _ := strconv.ParseUint(string(b[:i]), 10, 64)
	return id
}
