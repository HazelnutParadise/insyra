package insyra

import (
	"runtime"

	"github.com/HazelnutParadise/insyra/internal/core"
)

func init() {
	// Wire the trust-zone fallback warning. internal/core cannot import insyra
	// (circular), so it exposes a hook; we set it here, before any concurrency.
	core.TrustZoneFallbackHook = func() {
		LogDebug("core", "AtomicDo", "nested cross-instance AtomicDo took the unlocked trust-zone path; use insyra.AtomicDoAll to lock instances together")
	}
}

// ----------------------- DataList Atomic ----------------------

var dataListAtomicGroup = core.NewAtomicGroup()

// AtomicDo runs f with this DataList's actor lock held, serializing concurrent
// access to THIS instance.
//
// Nesting AtomicDo on the SAME instance is safe (e.g. Stdev calling Var). But do
// NOT nest AtomicDo on a DIFFERENT instance inside f to read two instances
// together: that inner call runs WITHOUT locking the other instance (a data race
// if another goroutine mutates it). To operate on multiple instances atomically,
// use AtomicDoAll instead.
func (s *DataList) AtomicDo(f func(*DataList)) {
	// LogDebug("DataList", "AtomicDo", "threadSafe: %v", Config.threadSafe.Load())
	if !Config.threadSafe.Load() {
		// Thread safety is off: run inline without the actor lock.
		f(s)
		return
	}
	s.atomicActor.SetGroupOnce(dataListAtomicGroup)
	core.AtomicDoWithInit(&s.atomicActor, s, f, func() {
		// Register a finalizer so an abandoned instance releases its actor.
		runtime.SetFinalizer(s, (*DataList).cleanup)
	})
}

// Close releases the DataList's actor. Later AtomicDo calls run inline.
func (s *DataList) Close() {
	if s.atomicActor.IsClosed() {
		return
	}
	s.atomicActor.Close()
}

// cleanup is the finalizer hook: it closes the actor when the value is collected.
func (s *DataList) cleanup() {
	s.Close()
}

// ----------------------- DataTable Atomic ----------------------

var dataTableAtomicGroup = core.NewAtomicGroup()

// AtomicDo runs f with this DataTable's actor lock held, serializing concurrent
// access to THIS instance. Same-instance nesting is safe; to operate on multiple
// instances atomically use AtomicDoAll (nesting AtomicDo on a different instance
// does NOT lock it and can race).
func (dt *DataTable) AtomicDo(f func(*DataTable)) {
	// LogDebug("DataTable", "AtomicDo", "threadSafe: %v", Config.threadSafe.Load())
	if !Config.threadSafe.Load() {
		// Thread safety is off: run inline without the actor lock.
		f(dt)
		return
	}
	dt.atomicActor.SetGroupOnce(dataTableAtomicGroup)
	core.AtomicDoWithInit(&dt.atomicActor, dt, f, func() {
		// Register a finalizer so an abandoned instance releases its actor.
		runtime.SetFinalizer(dt, (*DataTable).cleanup)
	})
}

// Close releases the DataTable's actor. Later AtomicDo calls run inline.
func (dt *DataTable) Close() {
	if dt.atomicActor.IsClosed() {
		return
	}
	dt.atomicActor.Close()
}

// cleanup is the finalizer hook: it closes the actor when the value is collected.
func (dt *DataTable) cleanup() {
	dt.Close()
}

// ----------------------- Multi-instance Atomic ----------------------

// AtomicDoAll runs f with the actor locks of ALL given DataList/DataTable
// instances held together. The locks are acquired in a canonical order and
// released in reverse, so it is deadlock-free even across many instances and
// across both DataList and DataTable.
//
// Use this instead of nesting AtomicDo on different instances — a nested
// AtomicDo on another instance does NOT lock it and can race a concurrent
// mutation. Accepts *DataList / *DataTable values (or IDataList / IDataTable
// whose concrete type is one of those). Instances already locked by the current
// goroutine, and duplicates, are handled automatically. Callers close over their
// own typed variables inside f.
//
//	a, b := ... // two *DataList
//	insyra.AtomicDoAll(func() {
//		// both a and b are locked here
//	}, a, b)
func AtomicDoAll(f func(), instances ...any) {
	if !Config.threadSafe.Load() {
		f()
		return
	}
	actors := make([]*core.AtomicActor, 0, len(instances))
	hooks := make([]func(), 0, len(instances))
	for _, inst := range instances {
		switch v := inst.(type) {
		case *DataList:
			v.atomicActor.SetGroupOnce(dataListAtomicGroup)
			actors = append(actors, &v.atomicActor)
			hooks = append(hooks, func() { runtime.SetFinalizer(v, (*DataList).cleanup) })
		case *DataTable:
			v.atomicActor.SetGroupOnce(dataTableAtomicGroup)
			actors = append(actors, &v.atomicActor)
			hooks = append(hooks, func() { runtime.SetFinalizer(v, (*DataTable).cleanup) })
		default:
			LogWarning("insyra", "AtomicDoAll", "unsupported instance type %T; skipped (not locked)", inst)
		}
	}
	core.AtomicDoNWithInit(actors, hooks, f)
}
