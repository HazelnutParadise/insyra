package atomic_test

import (
	"sync"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra/engine/atomic"
)

// engine/atomic is documented as a general-purpose primitive: any struct that
// holds an *Actor gets per-instance serialisation and deadlock-free
// multi-instance locking. It had no test, so nothing checked that a consumer
// outside the module can actually build that.

// account is the shape the package's own doc comment describes: a user struct
// with an actor of its own.
type account struct {
	actor   *atomic.Actor
	balance int
}

func newAccount(g *atomic.Group) *account {
	return &account{actor: atomic.NewActor(g)}
}

// mustFinish fails instead of hanging when the locking goes wrong.
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

// The increment is a plain read-modify-write: unserialised, it is a data race
// under -race and comes out short without it.
func TestAtomicDoSerialises(t *testing.T) {
	a := newAccount(atomic.NewGroup())

	const goroutines, each = 8, 200
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < each; j++ {
				atomic.AtomicDo(a.actor, a, func(a *account) { a.balance++ })
			}
		}()
	}
	wg.Wait()

	if want := goroutines * each; a.balance != want {
		t.Errorf("balance: got %d, want %d", a.balance, want)
	}
}

func TestAtomicDoWithInitRunsTheHookOnce(t *testing.T) {
	a := newAccount(atomic.NewGroup())
	inits := 0

	for i := 0; i < 3; i++ {
		atomic.AtomicDoWithInit(a.actor, a, func(a *account) { a.balance++ }, func() { inits++ })
	}

	if inits != 1 {
		t.Errorf("init hook ran %d times, want 1", inits)
	}
	if a.balance != 3 {
		t.Errorf("callback ran %d times, want 3", a.balance)
	}
}

// The documented use of AtomicDoN: move value between two structs with both
// locked, from either direction, without an AB-BA deadlock.
func TestAtomicDoNLocksTwoStructsTogether(t *testing.T) {
	a := newAccount(atomic.NewGroup())
	b := newAccount(atomic.NewGroup())
	a.balance = 1000

	mustFinish(t, "transfers from both directions", func() {
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < 250; i++ {
				atomic.AtomicDoN([]*atomic.Actor{a.actor, b.actor}, func() {
					a.balance--
					b.balance++
				})
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < 250; i++ {
				atomic.AtomicDoN([]*atomic.Actor{b.actor, a.actor}, func() {
					a.balance--
					b.balance++
				})
			}
		}()
		wg.Wait()
	})

	if a.balance != 500 || b.balance != 500 {
		t.Errorf("balances: got %d and %d, want 500 and 500", a.balance, b.balance)
	}
}

// nil entries and a duplicate of the same actor are documented as fine; a
// duplicate would self-deadlock without de-duplication.
func TestAtomicDoNSkipsNilAndDeduplicates(t *testing.T) {
	a := newAccount(atomic.NewGroup())
	ran := false

	mustFinish(t, "AtomicDoN with a duplicate and a nil", func() {
		atomic.AtomicDoN([]*atomic.Actor{a.actor, nil, a.actor}, func() { ran = true })
	})

	if !ran {
		t.Error("the callback did not run")
	}
	mustFinish(t, "a second AtomicDoN", func() {
		atomic.AtomicDoN([]*atomic.Actor{a.actor}, func() {})
	})
}

func TestAtomicDoNWithInitRunsEachHookOnce(t *testing.T) {
	a := newAccount(atomic.NewGroup())
	b := newAccount(atomic.NewGroup())
	initA, initB := 0, 0

	for i := 0; i < 3; i++ {
		atomic.AtomicDoNWithInit(
			[]*atomic.Actor{a.actor, b.actor},
			[]func(){func() { initA++ }, func() { initB++ }},
			func() {},
		)
	}

	if initA != 1 || initB != 1 {
		t.Errorf("init hooks ran %d and %d times, want 1 and 1", initA, initB)
	}
}

// NewActor(nil) falls back to the package-wide default group, so two such
// actors share a re-entrancy scope.
func TestNewActorNilGroupUsesTheDefault(t *testing.T) {
	a := newAccount(nil)
	b := atomic.NewActor(atomic.DefaultGroup())
	inner := 0

	mustFinish(t, "nesting into the default group", func() {
		atomic.AtomicDo(a.actor, a, func(a *account) {
			atomic.AtomicDo(b, &inner, func(n *int) { *n++ })
		})
	})

	if inner != 1 {
		t.Errorf("the inner callback ran %d times, want 1", inner)
	}
}
