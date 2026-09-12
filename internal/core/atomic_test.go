package core

import (
	"sync"
	"testing"
	"time"
)

// atomic_alt_test.go only benchmarks. Nothing asserted what the actor does.

// mustFinish runs f in its own goroutine and fails the test if it has not
// returned within a second. Several of the cases below would hang rather than
// fail if the locking went wrong — a double Lock on the same mutex, or an AB-BA
// pair — and a hung test tells nobody anything until the package times out.
func mustFinish(t *testing.T, what string, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("%s did not finish within a second (deadlock)", what)
	}
}

// withHook installs a TrustZoneFallbackHook for the duration of the test and
// returns a function reporting how many times it fired. The hook is a package
// variable, so it is restored before the next test runs.
func withHook(t *testing.T) func() int {
	t.Helper()
	var mu sync.Mutex
	calls := 0
	prev := TrustZoneFallbackHook
	TrustZoneFallbackHook = func() {
		mu.Lock()
		calls++
		mu.Unlock()
	}
	t.Cleanup(func() { TrustZoneFallbackHook = prev })
	return func() int {
		mu.Lock()
		defer mu.Unlock()
		return calls
	}
}

type counter struct {
	n int
}

// The callbacks of one actor never overlap. The increment is deliberately a
// plain read-modify-write: under -race an unserialised actor is reported here,
// and without -race the final count comes out short.
func TestAtomicDo_Serialises(t *testing.T) {
	actor := NewAtomicActor(NewAtomicGroup())
	c := &counter{}

	const goroutines, each = 8, 200
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < each; j++ {
				AtomicDo(actor, c, func(c *counter) { c.n++ })
			}
		}()
	}
	wg.Wait()

	if want := goroutines * each; c.n != want {
		t.Errorf("counter: got %d, want %d", c.n, want)
	}
}

// A method that calls another method of the same instance inside an outer
// callback — DataList.Stdev calling Var — must run inline, not deadlock.
func TestAtomicDo_SameActorReentry(t *testing.T) {
	hookCalls := withHook(t)
	actor := NewAtomicActor(NewAtomicGroup())
	c := &counter{}

	mustFinish(t, "same-actor re-entry", func() {
		AtomicDo(actor, c, func(c *counter) {
			c.n++
			AtomicDo(actor, c, func(c *counter) { c.n++ })
		})
	})

	if c.n != 2 {
		t.Errorf("both callbacks should have run: got n = %d, want 2", c.n)
	}
	if got := hookCalls(); got != 0 {
		t.Errorf("same-actor re-entry fired the trust-zone hook %d times, want 0", got)
	}
}

// Re-entering a *different* actor of the same group runs inline without taking
// that actor's lock. That is the documented trust zone, and the hook is how the
// root package logs it.
func TestAtomicDo_CrossActorReentryRunsInlineAndReports(t *testing.T) {
	hookCalls := withHook(t)
	group := NewAtomicGroup()
	a := NewAtomicActor(group)
	b := NewAtomicActor(group)
	ca, cb := &counter{}, &counter{}

	mustFinish(t, "cross-actor re-entry", func() {
		AtomicDo(a, ca, func(ca *counter) {
			ca.n++
			AtomicDo(b, cb, func(cb *counter) { cb.n++ })
		})
	})

	if ca.n != 1 || cb.n != 1 {
		t.Errorf("both callbacks should have run: got %d and %d, want 1 and 1", ca.n, cb.n)
	}
	if got := hookCalls(); got != 1 {
		t.Errorf("trust-zone hook fired %d times, want 1", got)
	}
}

// The trust zone is per group: an actor in another group is locked normally,
// so no inline path and no hook.
func TestAtomicDo_OtherGroupIsLockedNormally(t *testing.T) {
	hookCalls := withHook(t)
	a := NewAtomicActor(NewAtomicGroup())
	b := NewAtomicActor(NewAtomicGroup())
	ca, cb := &counter{}, &counter{}

	mustFinish(t, "nesting across groups", func() {
		AtomicDo(a, ca, func(ca *counter) {
			ca.n++
			AtomicDo(b, cb, func(cb *counter) { cb.n++ })
		})
	})

	if got := hookCalls(); got != 0 {
		t.Errorf("nesting across groups fired the trust-zone hook %d times, want 0", got)
	}
	if ca.n != 1 || cb.n != 1 {
		t.Errorf("both callbacks should have run: got %d and %d", ca.n, cb.n)
	}
}

// A nil actor is not an error: the callback still runs, unserialised.
func TestAtomicDo_NilActor(t *testing.T) {
	c := &counter{}
	AtomicDo(nil, c, func(c *counter) { c.n++ })
	if c.n != 1 {
		t.Errorf("callback with a nil actor: got n = %d, want 1", c.n)
	}
}

// AtomicDoWithInit runs its hook exactly once across the actor's life.
func TestAtomicDoWithInit_RunsTheHookOnce(t *testing.T) {
	actor := NewAtomicActor(NewAtomicGroup())
	c := &counter{}
	inits := 0

	for i := 0; i < 3; i++ {
		AtomicDoWithInit(actor, c, func(c *counter) { c.n++ }, func() { inits++ })
	}

	if inits != 1 {
		t.Errorf("init hook ran %d times, want 1", inits)
	}
	if c.n != 3 {
		t.Errorf("callback ran %d times, want 3", c.n)
	}
}

// Close means "stop locking", not "stop running": work already on its way in
// still happens, because dropping it would lose a caller's write silently.
func TestAtomicActor_CloseKeepsRunningTheCallback(t *testing.T) {
	actor := NewAtomicActor(NewAtomicGroup())
	c := &counter{}

	AtomicDo(actor, c, func(c *counter) { c.n++ })
	actor.Close()
	AtomicDo(actor, c, func(c *counter) { c.n++ })

	if c.n != 2 {
		t.Errorf("the callback after Close did not run: got n = %d, want 2", c.n)
	}
	if !actor.IsClosed() {
		t.Error("IsClosed after Close = false, want true")
	}
}

func TestAtomicActor_NilReceiver(t *testing.T) {
	var actor *AtomicActor

	actor.Close() // must not panic
	if !actor.IsClosed() {
		t.Error("IsClosed on a nil actor = false, want true")
	}
	actor.SetGroupOnce(NewAtomicGroup()) // must not panic
}

// SetGroupOnce is once: a second call does not move the actor.
func TestAtomicActor_SetGroupOnce(t *testing.T) {
	hookCalls := withHook(t)
	first := NewAtomicGroup()
	second := NewAtomicGroup()

	a := &AtomicActor{}
	a.SetGroupOnce(first)
	a.SetGroupOnce(second)

	// b is in `first`. If a had moved to `second`, nesting b inside a would lock
	// instead of taking the inline trust-zone path, and the hook would not fire.
	b := NewAtomicActor(first)
	ca, cb := &counter{}, &counter{}
	mustFinish(t, "nesting after SetGroupOnce", func() {
		AtomicDo(a, ca, func(ca *counter) {
			AtomicDo(b, cb, func(cb *counter) { cb.n++ })
		})
	})

	if got := hookCalls(); got != 1 {
		t.Errorf("trust-zone hook fired %d times, want 1 — the second SetGroupOnce took effect", got)
	}
}

// An actor built with a nil group falls back to the package-wide default.
func TestNewAtomicActor_NilGroupUsesTheDefault(t *testing.T) {
	hookCalls := withHook(t)
	a := NewAtomicActor(nil)
	b := NewAtomicActor(DefaultAtomicGroup())
	ca, cb := &counter{}, &counter{}

	mustFinish(t, "nesting into the default group", func() {
		AtomicDo(a, ca, func(ca *counter) {
			AtomicDo(b, cb, func(cb *counter) { cb.n++ })
		})
	})

	if got := hookCalls(); got != 1 {
		t.Errorf("trust-zone hook fired %d times, want 1 — a nil group did not fall back to the default", got)
	}
}

// AtomicDoN holds every actor at once. Two goroutines locking the same pair in
// opposite argument order must not deadlock: the canonical order is by pointer
// address, not by the order the caller passed them.
func TestAtomicDoN_LocksTogetherWithoutABBA(t *testing.T) {
	a := NewAtomicActor(NewAtomicGroup())
	b := NewAtomicActor(NewAtomicGroup())
	shared := &counter{}

	mustFinish(t, "AtomicDoN from both directions", func() {
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				AtomicDoN([]*AtomicActor{a, b}, func() { shared.n++ })
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				AtomicDoN([]*AtomicActor{b, a}, func() { shared.n++ })
			}
		}()
		wg.Wait()
	})

	if shared.n != 1000 {
		t.Errorf("counter: got %d, want 1000", shared.n)
	}
}

// The same actor passed twice would self-deadlock on a second mu.Lock without
// de-duplication.
func TestAtomicDoN_DeduplicatesAndSkipsNil(t *testing.T) {
	a := NewAtomicActor(NewAtomicGroup())
	ran := false

	mustFinish(t, "AtomicDoN with a duplicate and a nil", func() {
		AtomicDoN([]*AtomicActor{a, nil, a, nil}, func() { ran = true })
	})

	if !ran {
		t.Error("the callback did not run")
	}
	// The locks must all be released: another AtomicDoN on the same actor works.
	mustFinish(t, "a second AtomicDoN", func() {
		AtomicDoN([]*AtomicActor{a}, func() {})
	})
}

func TestAtomicDoN_EmptyAndAllNil(t *testing.T) {
	ran := 0
	mustFinish(t, "AtomicDoN with no actors", func() {
		AtomicDoN(nil, func() { ran++ })
		AtomicDoN([]*AtomicActor{}, func() { ran++ })
		AtomicDoN([]*AtomicActor{nil, nil}, func() { ran++ })
	})
	if ran != 3 {
		t.Errorf("the callback ran %d times, want 3", ran)
	}
}

// Called from inside an AtomicDo on one of its own actors, AtomicDoN takes the
// inline trust-zone path instead of locking the rest — locking there would
// deadlock against a goroutine doing the mirror image.
func TestAtomicDoN_ReentryRunsInline(t *testing.T) {
	hookCalls := withHook(t)
	a := NewAtomicActor(NewAtomicGroup())
	b := NewAtomicActor(NewAtomicGroup())
	ca := &counter{}
	inner := false

	mustFinish(t, "AtomicDoN inside AtomicDo", func() {
		AtomicDo(a, ca, func(ca *counter) {
			AtomicDoN([]*AtomicActor{a, b}, func() { inner = true })
		})
	})

	if !inner {
		t.Error("the inner callback did not run")
	}
	if got := hookCalls(); got != 1 {
		t.Errorf("trust-zone hook fired %d times, want 1", got)
	}
}

func TestAtomicDoNWithInit_RunsEachHookOnce(t *testing.T) {
	a := NewAtomicActor(NewAtomicGroup())
	b := NewAtomicActor(NewAtomicGroup())
	initA, initB := 0, 0

	for i := 0; i < 3; i++ {
		AtomicDoNWithInit(
			[]*AtomicActor{a, b},
			[]func(){func() { initA++ }, func() { initB++ }},
			func() {},
		)
	}

	if initA != 1 || initB != 1 {
		t.Errorf("init hooks ran %d and %d times, want 1 and 1", initA, initB)
	}

	// A short initHooks slice and nil entries are allowed.
	c := NewAtomicActor(NewAtomicGroup())
	mustFinish(t, "AtomicDoNWithInit with a short hook slice", func() {
		AtomicDoNWithInit([]*AtomicActor{a, b, c}, []func(){nil}, func() {})
	})
}

// A closed actor is left out of the lock set, and the callback still runs.
func TestAtomicDoN_SkipsClosedActors(t *testing.T) {
	a := NewAtomicActor(NewAtomicGroup())
	b := NewAtomicActor(NewAtomicGroup())
	b.Close()
	ran := false

	mustFinish(t, "AtomicDoN with a closed actor", func() {
		AtomicDoN([]*AtomicActor{a, b}, func() { ran = true })
	})

	if !ran {
		t.Error("the callback did not run when one actor was closed")
	}
}
