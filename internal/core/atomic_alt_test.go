package core

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
)

// slowGetGID is the runtime.Stack-based goroutine-ID extractor that the
// production code used before the petermattis/goid switch. Kept here as
// a benchmark reference point so we can show how much of the speedup is
// "no channel-actor" vs "no stack walk".
func slowGetGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	i := bytes.IndexByte(b, ' ')
	id, _ := strconv.ParseUint(string(b[:i]), 10, 64)
	return id
}

// This file benchmarks alternative AtomicDo implementations against the
// current channel-actor + runtime.Stack getGID design. The goal is to
// validate (or refute) the claim that swapping the actor for a mutex +
// fast goroutine-ID lookup gives a >>10× speedup on the dominant cold
// and re-entry paths.
//
// All variants implement the same conceptual API:
//   - serialised execution: only one goroutine runs the callback at a time
//   - re-entry support: same-goroutine recursive calls run inline, no deadlock

// --------- VARIANT 1: Current implementation (baseline) ---------
// Uses the production AtomicActor from atomic.go. Channel-actor +
// runtime.Stack-based getGID + sync.Map for re-entry context.

func benchCurrent(actor *AtomicActor, work func()) {
	AtomicDo(actor, &struct{}{}, func(_ *struct{}) {
		work()
	})
}

// --------- VARIANT 2: Mutex only, no re-entry support ---------
// Simplest possible: a sync.Mutex. Recursive same-goroutine calls deadlock.
// Useful as the absolute lower-bound cost of "just lock something".

type mutexOnlyActor struct {
	mu sync.Mutex
}

func (a *mutexOnlyActor) Do(work func()) {
	a.mu.Lock()
	defer a.mu.Unlock()
	work()
}

// --------- VARIANT 3: Mutex + atomic holder + runtime.Stack goid ---------
// Real re-entry support, but uses the slow stack-walk getGID. This isolates
// the cost of "channel-actor → mutex" without changing how we identify the
// goroutine. Should sit between variant 1 and variant 4.

type mutexSlowGIDActor struct {
	mu     sync.Mutex
	holder atomic.Uint64
}

func (a *mutexSlowGIDActor) Do(work func()) {
	gid := slowGetGID() // slow stack-walk
	if a.holder.Load() == gid {
		work() // re-entry
		return
	}
	a.mu.Lock()
	a.holder.Store(gid)
	work()
	a.holder.Store(0)
	a.mu.Unlock()
}

// VARIANT 4 was a fast-goid attempt via `//go:linkname runtime.getg`. That
// fails because runtime.getg is a compiler intrinsic, not a linkable
// function — the standard external trick is per-arch assembly that reads
// the g pointer from TLS (e.g. petermattis/goid). We don't ship that
// here; the comparison below already shows what changing the channel-
// actor for a mutex buys us, holding the goid mechanism constant.
//
// If the mutex variant wins big, the fast-goid speedup multiplies on
// top — that work would be a follow-up.

// --------- Benchmarks ---------

// noopWork is a near-empty body — the Do() infrastructure cost is what
// dominates. Real callbacks (e.g. dl.Mean) do meaningful work, but the
// per-call OVERHEAD is what we're measuring.
func noopWork() {}

// shortWork simulates a typical small computation (e.g. summing a slice).
func shortWork() {
	sum := 0.0
	for i := 0; i < 16; i++ {
		sum += float64(i)
	}
	_ = sum
}

// ---- Cold path: each call enters from outside the actor ----

func BenchmarkCurrent_Cold(b *testing.B) {
	actor := NewAtomicActor(NewAtomicGroup())
	for b.Loop() {
		benchCurrent(actor, noopWork)
	}
}

func BenchmarkMutexOnly_Cold(b *testing.B) {
	a := &mutexOnlyActor{}
	for b.Loop() {
		a.Do(noopWork)
	}
}

func BenchmarkMutexSlowGID_Cold(b *testing.B) {
	a := &mutexSlowGIDActor{}
	for b.Loop() {
		a.Do(noopWork)
	}
}

// ---- Cold path with realistic small work ----

func BenchmarkCurrent_ShortWork(b *testing.B) {
	actor := NewAtomicActor(NewAtomicGroup())
	for b.Loop() {
		benchCurrent(actor, shortWork)
	}
}

func BenchmarkMutexSlowGID_ShortWork(b *testing.B) {
	a := &mutexSlowGIDActor{}
	for b.Loop() {
		a.Do(shortWork)
	}
}

// ---- Re-entry path: nested calls on the same actor ----
// Models stats methods that call Mean() / Stdev() inside an outer AtomicDo.

func BenchmarkCurrent_Reentry(b *testing.B) {
	actor := NewAtomicActor(NewAtomicGroup())
	for b.Loop() {
		AtomicDo(actor, &struct{}{}, func(_ *struct{}) {
			// 3 inner re-entries — modelling Mean+Stdev+Len
			AtomicDo(actor, &struct{}{}, func(_ *struct{}) {})
			AtomicDo(actor, &struct{}{}, func(_ *struct{}) {})
			AtomicDo(actor, &struct{}{}, func(_ *struct{}) {})
		})
	}
}

func BenchmarkMutexSlowGID_Reentry(b *testing.B) {
	a := &mutexSlowGIDActor{}
	for b.Loop() {
		a.Do(func() {
			a.Do(func() {})
			a.Do(func() {})
			a.Do(func() {})
		})
	}
}

// ---- Cross-actor pattern: many actors, one call each ----
// Models TwoWayANOVA / OneWayANOVA / clustering input loaders that call
// AtomicDo on dozens of separate DataLists.

func BenchmarkCurrent_ManyActors(b *testing.B) {
	const n = 20
	actors := make([]*AtomicActor, n)
	for i := range actors {
		actors[i] = NewAtomicActor(NewAtomicGroup())
	}
	for b.Loop() {
		for _, a := range actors {
			AtomicDo(a, &struct{}{}, func(_ *struct{}) {})
		}
	}
}

func BenchmarkMutexSlowGID_ManyActors(b *testing.B) {
	const n = 20
	actors := make([]*mutexSlowGIDActor, n)
	for i := range actors {
		actors[i] = &mutexSlowGIDActor{}
	}
	for b.Loop() {
		for _, a := range actors {
			a.Do(func() {})
		}
	}
}

// ---- Sanity: cost of getGID in isolation ----

func BenchmarkGoID_Slow(b *testing.B) {
	for b.Loop() {
		_ = slowGetGID()
	}
}

// ---- End-to-end pattern: realistic stats access shape ----
//
// Models a typical T-test invocation pattern:
//   outer.AtomicDo {
//     inner.Len()     // re-entry to inner (owned by outer? or different?)
//     inner.Mean()    // re-entry
//     inner.Stdev()   // re-entry → calls Var() → re-entry
//   }
//
// where Len/Mean/Stdev each make their own AtomicDo call. For DataList in
// production these are all the SAME actor (the DataList), so they'd be
// re-entries.
//
// To avoid biasing toward either implementation we keep the inner work
// non-trivial (sum a 100-element slice).

// BenchmarkE2E_Current uses the production channel-actor.
func BenchmarkE2E_Current(b *testing.B) {
	actor := NewAtomicActor(NewAtomicGroup())
	data := make([]float64, 100)
	for i := range data {
		data[i] = float64(i)
	}
	owner := &data
	for b.Loop() {
		AtomicDo(actor, owner, func(d *[]float64) {
			// emulate Len()
			AtomicDo(actor, d, func(d *[]float64) {
				_ = len(*d)
			})
			// emulate Mean()
			AtomicDo(actor, d, func(d *[]float64) {
				sum := 0.0
				for _, v := range *d {
					sum += v
				}
				_ = sum / float64(len(*d))
			})
			// emulate Stdev() → Var()
			AtomicDo(actor, d, func(d *[]float64) {
				sum := 0.0
				for _, v := range *d {
					sum += v
				}
				mean := sum / float64(len(*d))
				ss := 0.0
				for _, v := range *d {
					ss += (v - mean) * (v - mean)
				}
				_ = ss
			})
		})
	}
}

// BenchmarkE2E_Mutex previously compared the mutex-prototype against the
// channel-actor production code. The production AtomicActor IS now the
// mutex-prototype, so this bench duplicates BenchmarkE2E_Current. Kept
// as a no-op stub to preserve the benchmark name in any external scripts
// that grep for it; running it just measures the same thing.
func BenchmarkE2E_Mutex(b *testing.B) {
	actor := NewAtomicActor(NewAtomicGroup())
	data := make([]float64, 100)
	for i := range data {
		data[i] = float64(i)
	}
	owner := &data
	for b.Loop() {
		AtomicDo(actor, owner, func(d *[]float64) {
			AtomicDo(actor, d, func(d *[]float64) {
				_ = len(*d)
			})
			AtomicDo(actor, d, func(d *[]float64) {
				sum := 0.0
				for _, v := range *d {
					sum += v
				}
				_ = sum / float64(len(*d))
			})
			AtomicDo(actor, d, func(d *[]float64) {
				sum := 0.0
				for _, v := range *d {
					sum += v
				}
				mean := sum / float64(len(*d))
				ss := 0.0
				for _, v := range *d {
					ss += (v - mean) * (v - mean)
				}
				_ = ss
			})
		})
	}
}
